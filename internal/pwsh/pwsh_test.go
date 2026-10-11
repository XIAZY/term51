package pwsh

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type entry struct {
	name string
	mode int64
	body string
	kind byte
	link string
}

// archive makes a small tar.gz and its sha256.
func archive(t *testing.T, entries []entry) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Mode: e.mode, Typeflag: e.kind, Linkname: e.link, Size: int64(len(e.body))}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	zw.Close()
	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(sum[:])
}

// The tree of a release in small: folders, a program, a file that is not one.
var tree = []entry{
	{name: "./", mode: 0o755, kind: tar.TypeDir},
	{name: "./pwsh/", mode: 0o755, kind: tar.TypeDir},
	{name: "./pwsh/pwsh", mode: 0o755, kind: tar.TypeReg, body: "#!/bin/sh\necho pwsh\n"},
	{name: "./pwsh/pwsh.dll", mode: 0o644, kind: tar.TypeReg, body: "not a program"},
	{name: "./pwsh.props", mode: 0o644, kind: tar.TypeReg, body: "APP=$ROOT/pwsh/pwsh.dll\n"},
}

// states collects what a store reports.
type states struct {
	mu  sync.Mutex
	all []State
}

func (s *states) add(st State) { s.mu.Lock(); s.all = append(s.all, st); s.mu.Unlock() }
func (s *states) phases() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, st := range s.all {
		if len(out) == 0 || out[len(out)-1] != st.Phase {
			out = append(out, st.Phase)
		}
	}
	return strings.Join(out, " ")
}

// leftovers lists what is in the data folder.
func leftovers(t *testing.T, dir string) string {
	t.Helper()
	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, n := range names {
		out = append(out, n.Name())
	}
	return strings.Join(out, " ")
}

// site is a release as a server gives it: the manifest, and the archive it
// names. The fields can be changed before a test asks for them.
type site struct {
	url      string // the manifest's address
	version  string
	archive  []byte
	size     int64  // what the manifest says; 0: the archive's
	sum      string // what the manifest says; "": the archive's
	targets  string // "": the test's target
	manifest string // not empty: served as it is in place of the manifest
	cut      int    // not 0: the archive breaks off after so many bytes
	hold     chan struct{}
	started  chan struct{}
}

func serve(t *testing.T, archive []byte) *site {
	t.Helper()
	st := &site{version: "7.6.6", archive: archive}
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	st.url = srv.URL + "/latest/powershell-qnx.json"
	mux.HandleFunc("/latest/powershell-qnx.json", func(w http.ResponseWriter, r *http.Request) {
		if st.manifest != "" {
			w.Write([]byte(st.manifest))
			return
		}
		size, sum, target := st.size, st.sum, st.targets
		if size == 0 {
			size = int64(len(st.archive))
		}
		if sum == "" {
			h := sha256.Sum256(st.archive)
			sum = hex.EncodeToString(h[:])
		}
		if target == "" {
			target = "phone"
		}
		fmt.Fprintf(w, `{"version":%q,"tag":"v%s","targets":{%q:{"file":"p.tar.gz","url":%q,"size":%d,"sha256":%q}}}`,
			st.version, st.version, target, srv.URL+"/files/p.tar.gz", size, sum)
	})
	mux.HandleFunc("/files/p.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		if st.cut != 0 {
			w.Header().Set("Content-Length", fmt.Sprint(len(st.archive)))
			w.Write(st.archive[:st.cut])
			if st.started != nil {
				w.(http.Flusher).Flush()
				close(st.started)
				<-st.hold
			}
			return // short of the length it announced: the client sees it break off
		}
		w.Write(st.archive)
	})
	return st
}

func open(dir string, st *site, changed func(State)) *Store {
	return Open(dir, st.url, "phone", http.DefaultClient, changed)
}

func TestDownloadUnpackRemove(t *testing.T) {
	data, _ := archive(t, tree)
	st := serve(t, data)
	dir := t.TempDir()
	var seen states
	s := open(dir, st, seen.add)
	if got := s.State(); got.Phase != Absent {
		t.Fatalf("a new folder is %+v", got)
	}
	if err := s.Download(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := seen.phases(); got != "downloading unpacking ready" {
		t.Errorf("the phases were %q", got)
	}
	if got := s.State(); got != (State{Phase: Ready, Version: "7.6.6"}) {
		t.Errorf("after the download: %+v", got)
	}
	// The page was told the archive's size while it came in.
	seen.mu.Lock()
	told := false
	for _, x := range seen.all {
		told = told || (x.Phase == Downloading && x.Total == int64(len(data)))
	}
	seen.mu.Unlock()
	if !told {
		t.Error("the archive's size was never told while downloading")
	}
	// The tree is there, with the archive's permissions: the program can
	// be run, the other file cannot.
	for name, mode := range map[string]os.FileMode{"pwsh/pwsh": 0o755, "pwsh/pwsh.dll": 0o644, "pwsh.props": 0o644} {
		info, err := os.Stat(filepath.Join(dir, "powershell", name))
		if err != nil || info.Mode().Perm() != mode {
			t.Errorf("%s: %v, %v, want mode %o", name, info, err, mode)
		}
	}
	if s.Program() != filepath.Join(dir, "powershell", "pwsh", "pwsh") {
		t.Errorf("the program is %q", s.Program())
	}
	if got := leftovers(t, dir); got != "powershell" {
		t.Errorf("the data folder holds %q, want only the tree", got)
	}

	// The next start finds it, and says which version it is.
	again := open(dir, st, func(State) {})
	if got := again.State(); got != (State{Phase: Ready, Version: "7.6.6"}) {
		t.Errorf("at the next start: %+v", got)
	}
	// The same release: nothing newer. Another archive under a new
	// version: it is offered, and downloading it replaces the tree.
	if err := again.Check(context.Background()); err != nil || again.State().Available != "" {
		t.Errorf("a check against the same release: %v, %+v", err, again.State())
	}
	newer, _ := archive(t, append(tree[:len(tree):len(tree)], entry{name: "./new-in-7.6.7", mode: 0o644, kind: tar.TypeReg, body: "x"}))
	st.archive, st.version = newer, "7.6.7"
	if err := again.Check(context.Background()); err != nil || again.State() != (State{Phase: Ready, Version: "7.6.6", Available: "7.6.7"}) {
		t.Errorf("a check against a newer release: %v, %+v", err, again.State())
	}
	if err := again.Download(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := again.State(); got != (State{Phase: Ready, Version: "7.6.7"}) {
		t.Errorf("after the newer download: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "powershell", "new-in-7.6.7")); err != nil || leftovers(t, dir) != "powershell" {
		t.Errorf("the newer tree is not in place: %v, folder %q", err, leftovers(t, dir))
	}

	if err := again.Remove(); err != nil {
		t.Fatal(err)
	}
	if got := again.State(); got.Phase != Absent || leftovers(t, dir) != "" {
		t.Errorf("after the removal: %+v, folder %q", got, leftovers(t, dir))
	}
	// With none there, a check asks nothing and changes nothing.
	if err := again.Check(context.Background()); err != nil || again.State().Phase != Absent {
		t.Errorf("a check with nothing there: %v, %+v", err, again.State())
	}
}

// Every failure leaves the state failed with its reason, and nothing of the
// attempt in the folder.
func TestDownloadFailures(t *testing.T) {
	data, _ := archive(t, tree)
	link, _ := archive(t, []entry{{name: "./pwsh/pwsh", kind: tar.TypeSymlink, link: "/bin/sh"}})
	outside, _ := archive(t, []entry{{name: "../evil", mode: 0o644, kind: tar.TypeReg, body: "x"}})

	for _, c := range []struct {
		name   string
		change func(*site)
		reason string
		phases string
	}{
		{"another archive than the manifest names", func(s *site) { s.sum = strings.Repeat("0", 64) },
			"not the one the release names", "downloading failed"},
		{"another size than the manifest names", func(s *site) { s.size = int64(len(data)) + 1 },
			"the release says", "downloading failed"},
		{"the connection breaks part-way", func(s *site) { s.cut = 50 },
			"broke off", "downloading failed"},
		{"a manifest without this phone", func(s *site) { s.targets = "qnx65-x86" },
			"no download for this phone", "downloading failed"},
		{"a manifest that is not JSON", func(s *site) { s.manifest = "<html>not here</html>" },
			"not what this app can read", "downloading failed"},
		{"no manifest at all", func(s *site) { s.url += ".gone" },
			"404", "downloading failed"},
		{"a link in the archive", func(s *site) { s.archive = link },
			"neither a file nor a folder", "downloading unpacking failed"},
		{"a path out of the folder", func(s *site) { s.archive = outside },
			"leaves the folder", "downloading unpacking failed"},
	} {
		st := serve(t, data)
		c.change(st)
		dir := t.TempDir()
		var seen states
		s := open(dir, st, seen.add)
		err := s.Download(context.Background())
		got := s.State()
		if err == nil || got.Phase != Failed || !strings.Contains(got.Reason, c.reason) {
			t.Errorf("%s: error %v, state %+v, want a failure saying %q", c.name, err, got, c.reason)
		}
		if phases := seen.phases(); phases != c.phases {
			t.Errorf("%s: the phases were %q, want %q", c.name, phases, c.phases)
		}
		if left := leftovers(t, dir); left != "" {
			t.Errorf("%s: the data folder holds %q afterwards", c.name, left)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "evil")); err == nil {
			t.Errorf("%s: a file was written outside the folder", c.name)
		}
	}
}

// A newer download that fails leaves the PowerShell that was there, and the
// state says both: ready, and why the download did not work.
func TestAFailedUpdateKeepsWhatWasThere(t *testing.T) {
	data, _ := archive(t, tree)
	st := serve(t, data)
	dir := t.TempDir()
	s := open(dir, st, func(State) {})
	if err := s.Download(context.Background()); err != nil {
		t.Fatal(err)
	}
	st.version, st.sum = "7.6.7", strings.Repeat("0", 64)
	if err := s.Download(context.Background()); err == nil {
		t.Fatal("the download of a wrong archive succeeded")
	}
	got := s.State()
	if got.Phase != Ready || got.Version != "7.6.6" || !strings.Contains(got.Reason, "not the one the release names") {
		t.Errorf("after the failed update: %+v", got)
	}
	if _, err := os.Stat(s.Program()); err != nil || leftovers(t, dir) != "powershell" {
		t.Errorf("the old tree is not whole: %v, folder %q", err, leftovers(t, dir))
	}
}

// A download that is stopped (the app is closing) fails like any other and
// leaves nothing; and nothing else can start while one runs.
func TestDownloadStoppedAndBusy(t *testing.T) {
	data, _ := archive(t, tree)
	st := serve(t, data)
	st.cut, st.started, st.hold = 10, make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(st.hold) })
	dir := t.TempDir()
	s := open(dir, st, func(State) {})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Download(ctx) }()
	<-st.started
	if err := s.Download(context.Background()); err != ErrBusy {
		t.Errorf("a second download while one runs: %v", err)
	}
	if err := s.Remove(); err != ErrBusy {
		t.Errorf("a removal while a download runs: %v", err)
	}
	if err := s.Check(context.Background()); err != ErrBusy {
		t.Errorf("a check while a download runs: %v", err)
	}
	cancel()
	if err := <-done; err == nil {
		t.Error("the stopped download reported success")
	}
	if got := s.State(); got.Phase != Failed || leftovers(t, dir) != "" {
		t.Errorf("after the stop: %+v, folder %q", got, leftovers(t, dir))
	}
}

func TestTheAddress(t *testing.T) {
	if ManifestURL != "https://github.com/XIAZY/powershell-qnx/releases/latest/download/powershell-qnx.json" || Target != "bb10-arm" {
		t.Errorf("the manifest is at %q, the target %q", ManifestURL, Target)
	}
}
