// Package pwsh downloads PowerShell into the app's data folder when the
// user asks for it, and removes it again. It is too large to ship in the
// package (44 MB to download, 129 MB unpacked).
//
// The app holds one address, that of the manifest of the newest release of
// PowerShell for QNX (ManifestURL). The manifest names the archive for the
// phone with its size and sha256; an archive that is not of that size and
// checksum is not unpacked.
package pwsh

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ManifestURL is where the newest release's manifest is: a small JSON file
// that the release carries. GitHub answers with a redirect to it.
const ManifestURL = "https://github.com/XIAZY/powershell-qnx/releases/latest/download/powershell-qnx.json"

// Target is the phone's name in the manifest.
const Target = "bb10-arm"

// Release is one target's entry of a manifest, with the release's version:
// the archive to download, and how to know it is the right one.
type Release struct {
	Version string `json:"version"`
	Tag     string `json:"tag"`
	File    string `json:"file"`
	URL     string `json:"url"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

// manifest is the file at ManifestURL.
type manifest struct {
	Version string             `json:"version"`
	Tag     string             `json:"tag"`
	Targets map[string]Release `json:"targets"`
}

// The phases of a Store.
const (
	Absent      = "absent"      // not downloaded
	Downloading = "downloading" // the archive is coming in: Done of Total bytes
	Unpacking   = "unpacking"   // the archive is whole and right, and is being unpacked
	Ready       = "ready"       // PowerShell is there: Version says which
	Failed      = "failed"      // there is none, and the last download did not work: Reason says why
)

// State is where a Store is.
type State struct {
	Phase string
	Done  int64 // bytes so far, while downloading
	Total int64 // the archive's size, while downloading

	// Version is the version of the PowerShell that is there, when ready.
	// Available is the newest release's version when that is another
	// archive than the one that is there (Check found it). Reason is why
	// the last download did not work; with a PowerShell still there, the
	// phase stays ready.
	Version   string
	Available string
	Reason    string
}

// Store is PowerShell in a data folder: dir/powershell when it is there.
// While a download runs it also uses dir/pwsh.part (the archive) and
// dir/pwsh.new (the tree being unpacked); neither outlives the download.
type Store struct {
	dir      string
	manifest string // the manifest's address
	target   string
	client   *http.Client
	changed  func(State)

	mu    sync.Mutex
	state State
	busy  bool
}

// stamp is the file in the tree that holds the manifest entry it was
// installed from. It is written last, so a tree with it is a whole one.
const stamp = ".manifest"

// Open looks at what is in dir. manifestURL and target say what to download
// (ManifestURL and Target in the app). changed is called with every new
// state, from the goroutine that runs Check, Download or Remove.
func Open(dir, manifestURL, target string, client *http.Client, changed func(State)) *Store {
	s := &Store{dir: dir, manifest: manifestURL, target: target, client: client, changed: changed, state: State{Phase: Absent}}
	if rel, ok := s.installed(); ok {
		s.state = State{Phase: Ready, Version: rel.Version}
	}
	return s
}

// installed is the manifest entry the tree in the folder was installed
// from, if there is a whole tree.
func (s *Store) installed() (Release, bool) {
	data, err := os.ReadFile(filepath.Join(s.tree(), stamp))
	if err != nil {
		return Release{}, false
	}
	var rel Release
	if json.Unmarshal(data, &rel) != nil {
		return Release{}, false
	}
	return rel, true
}

// newest reads the manifest and gives the entry for the target.
func (s *Store) newest(ctx context.Context) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.manifest, nil)
	if err != nil {
		return Release{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("the list of releases could not be read: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("the list of releases could not be read: the server answered %s", resp.Status)
	}
	var m manifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return Release{}, fmt.Errorf("the list of releases is not what this app can read: %w", err)
	}
	rel, ok := m.Targets[s.target]
	if !ok || rel.URL == "" || rel.Size <= 0 || len(rel.SHA256) != sha256.Size*2 {
		return Release{}, fmt.Errorf("the list of releases has no download for this phone (%s)", s.target)
	}
	rel.Version, rel.Tag = m.Version, m.Tag
	return rel, nil
}

// Check asks whether the newest release is another archive than the one
// that is there, and says so in the state (Available). With no PowerShell
// there it does nothing.
func (s *Store) Check(ctx context.Context) error {
	if err := s.begin(); err != nil {
		return err
	}
	defer s.end()
	have, ok := s.installed()
	if !ok {
		return nil
	}
	rel, err := s.newest(ctx)
	if err != nil {
		return err
	}
	st := State{Phase: Ready, Version: have.Version}
	if rel.SHA256 != have.SHA256 {
		st.Available = rel.Version
	}
	s.set(st)
	return nil
}

func (s *Store) tree() string  { return filepath.Join(s.dir, "powershell") }
func (s *Store) part() string  { return filepath.Join(s.dir, "pwsh.part") }
func (s *Store) fresh() string { return filepath.Join(s.dir, "pwsh.new") }

// Program is the path of pwsh, whether or not it is there.
func (s *Store) Program() string { return filepath.Join(s.tree(), "pwsh", "pwsh") }

// State is where the store is now.
func (s *Store) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Store) set(st State) {
	s.mu.Lock()
	s.state = st
	s.mu.Unlock()
	s.changed(st)
}

// ErrBusy is returned when a download or a removal is already running.
var ErrBusy = errors.New("pwsh: busy")

func (s *Store) begin() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return ErrBusy
	}
	s.busy = true
	return nil
}

func (s *Store) end() {
	s.mu.Lock()
	s.busy = false
	s.mu.Unlock()
}

// Download reads the manifest, fetches the archive it names, checks its
// size and sha256, unpacks it and puts the tree in place of any older one;
// it returns when that is done or has failed, and the state says which.
// Whatever goes wrong, nothing of the attempt is left in the folder, and a
// PowerShell that was there stays. Cancelling ctx stops it.
func (s *Store) Download(ctx context.Context) error {
	if err := s.begin(); err != nil {
		return err
	}
	defer s.end()
	rel, err := s.download(ctx)
	os.Remove(s.part())
	os.RemoveAll(s.fresh())
	if err != nil {
		if have, ok := s.installed(); ok {
			s.set(State{Phase: Ready, Version: have.Version, Reason: err.Error()})
		} else {
			s.set(State{Phase: Failed, Reason: err.Error()})
		}
		return err
	}
	s.set(State{Phase: Ready, Version: rel.Version})
	return nil
}

func (s *Store) download(ctx context.Context) (Release, error) {
	// Nothing of an attempt that the app did not live to clean up.
	os.Remove(s.part())
	if err := os.RemoveAll(s.fresh()); err != nil {
		return Release{}, err
	}
	s.set(State{Phase: Downloading})
	rel, err := s.newest(ctx)
	if err != nil {
		return Release{}, err
	}
	s.set(State{Phase: Downloading, Total: rel.Size})
	if err := s.fetch(ctx, rel); err != nil {
		return Release{}, err
	}
	return rel, nil
}

// fetch downloads, checks and unpacks the archive of rel, and puts the tree
// in place.
func (s *Store) fetch(ctx context.Context, rel Release) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.URL, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the server answered %s", resp.Status)
	}
	archive, err := os.OpenFile(s.part(), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer archive.Close()

	// The archive goes to the file and through the checksum at once; the
	// page hears of the progress a few times a second.
	sum := sha256.New()
	var done int64
	told := time.Now()
	buf := make([]byte, 64<<10)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := archive.Write(buf[:n]); werr != nil {
				return werr
			}
			sum.Write(buf[:n])
			done += int64(n)
			if time.Since(told) >= 200*time.Millisecond {
				told = time.Now()
				s.set(State{Phase: Downloading, Done: done, Total: rel.Size})
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("the download broke off after %d bytes: %w", done, err)
		}
	}
	if done != rel.Size {
		return fmt.Errorf("the archive has %d bytes, and the release says %d", done, rel.Size)
	}
	if got := hex.EncodeToString(sum.Sum(nil)); got != rel.SHA256 {
		return fmt.Errorf("the archive is not the one the release names (sha256 %s, expected %s)", got, rel.SHA256)
	}

	s.set(State{Phase: Unpacking})
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := unpack(ctx, archive, s.fresh()); err != nil {
		return fmt.Errorf("unpacking: %w", err)
	}
	entry, _ := json.Marshal(rel) // strings and a number always encode
	if err := os.WriteFile(filepath.Join(s.fresh(), stamp), append(entry, '\n'), 0o644); err != nil {
		return err
	}
	// An older tree gives way only now, to a whole new one.
	if err := os.RemoveAll(s.tree()); err != nil {
		return err
	}
	return os.Rename(s.fresh(), s.tree())
}

// unpack writes the files and folders of a gzipped tar archive under dir,
// with the permissions the archive gives them. The archive is one made for
// this app, so anything else in it (a link, a device, a path that
// leaves dir) is an error, not something to work around.
func unpack(ctx context.Context, archive io.Reader, dir string) error {
	zr, err := gzip.NewReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if !filepath.IsLocal(filepath.Clean(h.Name)) && filepath.Clean(h.Name) != "." {
			return fmt.Errorf("%q leaves the folder", h.Name)
		}
		path := filepath.Join(dir, h.Name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fs.FileMode(h.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%q is neither a file nor a folder", h.Name)
		}
	}
}

// Remove deletes PowerShell from the folder.
func (s *Store) Remove() error {
	if err := s.begin(); err != nil {
		return err
	}
	defer s.end()
	if err := os.RemoveAll(s.tree()); err != nil {
		s.set(State{Phase: Failed, Reason: err.Error()})
		return err
	}
	s.set(State{Phase: Absent})
	return nil
}
