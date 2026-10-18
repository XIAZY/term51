package pwsh

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// SyncLink keeps link, a symbolic link to program, in step with st: made
// when PowerShell is ready, removed when there is none (absent, or a failed
// download with nothing there). While a download runs it is left as it is,
// since the program it points to is replaced only at the end.
func SyncLink(link, program string, st State) error {
	switch st.Phase {
	case Ready:
		if have, err := os.Readlink(link); err == nil && have == program {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			return err
		}
		if err := os.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return os.Symlink(program, link)
	case Absent, Failed:
		if err := os.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}
