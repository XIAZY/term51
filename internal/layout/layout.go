// Package layout says where the app keeps what it writes: in its home, the
// data folder of its sandbox, in the conventional XDG places, as programs on
// other Unix systems do; and its log in the sandbox's logs folder.
package layout

import (
	"os"
	"path/filepath"
)

// Places are the app's files and folders in its sandbox. All paths are
// absolute.
type Places struct {
	Home   string // the app's home: the data folder of its sandbox
	Config string // the settings: .config/term51/settings.conf
	Log    string // the app's log: logs/term51.log in the sandbox, beside data
	Share  string // the app's own data: .local/share/term51
	Tools  string // the links to BusyBox's and Dropbear's tools: Share/bin
}

// For gives the places of the app whose folder is root.
func For(root string) Places {
	home := filepath.Join(root, "data")
	share := filepath.Join(home, ".local/share/term51")
	return Places{
		Home:   home,
		Config: filepath.Join(home, ".config/term51/settings.conf"),
		Log:    filepath.Join(root, "logs/term51.log"),
		Share:  share,
		Tools:  filepath.Join(share, "bin"),
	}
}

// Make makes the folders in the home the places are in. The sandbox's logs
// folder is the system's to make.
func (p Places) Make() error {
	for _, dir := range []string{filepath.Dir(p.Config), p.Share} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
