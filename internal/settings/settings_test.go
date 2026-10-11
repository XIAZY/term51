package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"term51/internal/replies"
	"term51/internal/session"
)

func TestTheDefaultsAreValidAndMatchThePackages(t *testing.T) {
	d := Default()
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	if got, want := d.Replies(), replies.DefaultConfig(); !reflect.DeepEqual(got, want) {
		t.Errorf("Default().Replies():\n got %+v\nwant %+v", got, want)
	}
	th := d.SessionTheme()
	if th.Foreground != session.DefaultTheme.Foreground || th.Background != session.DefaultTheme.Background || th.Cursor != nil || th.Palette != nil {
		t.Errorf("Default().SessionTheme() = %+v", th)
	}
}

func TestAnEmptyOrPartialFileKeepsTheDefaults(t *testing.T) {
	for _, in := range []string{"", "  \n", "# only a comment\n"} {
		s, warnings, err := Parse([]byte(in))
		if err != nil || len(warnings) != 0 || !reflect.DeepEqual(s, Default()) {
			t.Errorf("Parse(%q) = %+v, %v, %v", in, s, warnings, err)
		}
	}
	s, warnings, err := Parse([]byte(`
# what a person would type on the phone
fontPt = 8
  term=xterm
theme.cursor = #FF8000
BELL = no
shell = " /bin/odd name "
`))
	if err != nil || len(warnings) != 0 {
		t.Fatal(warnings, err)
	}
	want := Default()
	want.FontPt = 8
	want.Term = "xterm"
	want.Theme.Cursor = "#FF8000"
	want.Bell = false
	want.Shell = " /bin/odd name "
	if !reflect.DeepEqual(s, want) {
		t.Errorf("a partial file:\n got %+v\nwant %+v", s, want)
	}
	if c := s.SessionTheme().Cursor; c == nil || *c != (session.RGB{R: 0xFF, G: 0x80, B: 0}) {
		t.Errorf("theme.cursor: %v", c)
	}
}

func TestListsAreOneLinePerValue(t *testing.T) {
	s, _, err := Parse([]byte("shell = /bin/bash\narg = -l\narg = \"\"\narg = two words\ntheme.palette = #000000\ntheme.palette = #aa0000\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"-l", "", "two words"}; !reflect.DeepEqual(s.Args, want) {
		t.Errorf("args %q, want %q", s.Args, want)
	}
	if want := []string{"#000000", "#aa0000"}; !reflect.DeepEqual(s.Theme.Palette, want) {
		t.Errorf("palette %q, want %q", s.Theme.Palette, want)
	}
	if th := s.SessionTheme(); len(th.Palette) != 2 || th.Palette[1] != (session.RGB{R: 0xAA}) {
		t.Errorf("session palette %v", th.Palette)
	}
}

// A newer app's file read by this one: the names it does not know are
// warnings, and everything else is used.
func TestAnUnknownNameIsAWarning(t *testing.T) {
	s, warnings, err := Parse([]byte("fontPt = 8\ntabs.max = 8\nligatures = on\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.FontPt != 8 {
		t.Errorf("fontPt %v, want 8", s.FontPt)
	}
	if len(warnings) != 2 || !strings.Contains(warnings[0], "line 2") || !strings.Contains(warnings[0], "tabs.max") || !strings.Contains(warnings[1], "ligatures") {
		t.Errorf("warnings: %q", warnings)
	}
}

func TestABadValueNamesTheSettingAndNothingIsUsed(t *testing.T) {
	cases := []struct{ in, names string }{
		{"fontPt = 0", "fontPt"},
		{"fontPt = 1000", "fontPt"},
		{"fontPt = big", "fontPt"},
		{`shell = ""`, "shell"},
		{`term = ""`, "term"},
		{"scrollbackBytes = 99999999999", "scrollbackBytes"},
		{"scrollbackBytes = -1", "scrollbackBytes"},
		{"theme.foreground = red", "theme.foreground"},
		{"theme.background = #12345", "theme.background"},
		{"theme.cursor = #gggggg", "theme.cursor"},
		{"theme.palette = #000000\ntheme.palette = blue", "theme.palette[1]"},
		{"term = vt52", "term"},
		{"bell = perhaps", "bell"},
		{"fontPt = 8\nfontPt = 30", "fontPt is set twice"},
		{"fontPt = 8\njust some words", "line 4"}, // two lines come before each case
		{`shell = "unclosed`, "shell"},
	}
	for _, c := range cases {
		s, _, err := Parse([]byte("# a comment first\nshell = /bin/bash\n" + c.in + "\n"))
		if err == nil {
			t.Errorf("Parse(%q): no error", c.in)
			continue
		}
		if !strings.HasPrefix(err.Error(), "settings: ") || !strings.Contains(err.Error(), c.names) {
			t.Errorf("Parse(%q): error %q does not name %q", c.in, err, c.names)
		}
		if !reflect.DeepEqual(s, Default()) {
			t.Errorf("Parse(%q): a bad file changed the settings: %+v", c.in, s)
		}
	}
}

func TestSetAndValues(t *testing.T) {
	s := Default()
	v := s.Values()
	if v["fontPt"] != "7" || v["shell"] != "system" || v["bell"] != "true" {
		t.Errorf("values: fontPt %q, shell %q, bell %q", v["fontPt"], v["shell"], v["bell"])
	}
	if _, ok := v["arg"]; ok {
		t.Error("the list arg is among the single values")
	}
	if err := s.Set("fontPt", "5.5"); err != nil || s.FontPt != 5.5 {
		t.Errorf("Set fontPt: %v, %v", s.FontPt, err)
	}
	// A theme is chosen by name and fills the colour settings; the name in
	// force is read back from the colours.
	if v["theme"] != "dark" {
		t.Errorf("the default colours are theme %q, want dark", v["theme"])
	}
	for name, theme := range Themes {
		if err := s.Set("theme", name); err != nil || s.ThemeName() != name || s.Theme.Background != theme.Background {
			t.Errorf("Set theme %s: %v, name %q, background %s", name, err, s.ThemeName(), s.Theme.Background)
		}
		if err := s.Validate(); err != nil {
			t.Errorf("theme %s: %v", name, err)
		}
	}
	// A program asking for the colour scheme is told what the colours make.
	for name, dark := range map[string]bool{"dark": true, "black": true, "light": false, "solarized": true} {
		s.Set("theme", name)
		if got := s.Replies().Dark; got != dark {
			t.Errorf("theme %s is reported as dark=%v", name, got)
		}
	}
	s.Theme.Background = "#123456"
	if s.ThemeName() != "custom" {
		t.Errorf("someone's own colours are theme %q", s.ThemeName())
	}
	if err := s.Set("cursor", "bar"); err != nil || s.SessionCursor() != session.CursorBar {
		t.Errorf("Set cursor: %v, %v", s.SessionCursor(), err)
	}

	// A value that makes no sense, a name that is not a setting and a list
	// are refused, and nothing changes.
	for _, bad := range [][2]string{{"fontPt", "400"}, {"fontPt", "big"}, {"shell", ""}, {"colour", "red"}, {"arg", "-l"}, {"cursor", "dot"}, {"theme", "pink"}} {
		before := s
		before.Args, before.Theme.Palette = nil, nil
		err := s.Set(bad[0], bad[1])
		after := s
		after.Args, after.Theme.Palette = nil, nil
		if err == nil || !reflect.DeepEqual(before, after) {
			t.Errorf("Set(%q, %q): error %v, changed %v", bad[0], bad[1], err, !reflect.DeepEqual(before, after))
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.conf")

	// No file: the defaults, and the file is written for the user to edit,
	// every setting with its comment.
	s, warnings, err := Load(path)
	if err != nil || warnings != nil || !reflect.DeepEqual(s, Default()) {
		t.Fatalf("Load without a file: %+v, %v", s, err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the first Load wrote no file: %v", err)
	}
	if string(first) != string(Default().Format()) {
		t.Errorf("the file written by the first Load is not the default file:\n%s", first)
	}

	// The defaults written out and read back are the defaults.
	if got, w, err := Parse(Default().Format()); err != nil || len(w) != 0 || !reflect.DeepEqual(got, Default()) {
		t.Errorf("the default file read back: %+v, %v, %v", got, w, err)
	}

	s.FontPt = 8.5
	s.Shell = "/bin/bash"
	s.Args = []string{"-l", "", " spaced ", `"quoted"`}
	s.Term = "xterm"
	s.ScrollbackBytes = 1 << 20
	s.Theme = Theme{Foreground: "#ffffff", Background: "#000080", Cursor: "#ff0000", Palette: []string{"#010203", "#040506"}}
	s.Cursor = "underline"
	s.Bell = false
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	got, warnings, err := Load(path)
	if err != nil || len(warnings) != 0 {
		t.Fatal(warnings, err)
	}
	if !reflect.DeepEqual(got, s) {
		t.Errorf("after Save and Load:\n got %+v\nwant %+v", got, s)
	}
	// Every setting is in the file with a comment above it.
	text, _ := os.ReadFile(path)
	for i := range fields {
		if !strings.Contains(string(text), "\n# "+fields[i].comment+"\n") {
			t.Errorf("the file has no comment for %s", fields[i].name)
		}
	}
	// Nothing but the settings file is left behind.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("%d files in the directory after Save", len(entries))
	}

	// Settings that make no sense are not written, and the file stays.
	bad := s
	bad.FontPt = -1
	if err := Save(path, bad); err == nil {
		t.Error("Save accepted fontPt -1")
	}
	if again, _, _ := Load(path); !reflect.DeepEqual(again, s) {
		t.Error("a refused Save changed the file")
	}

	// A damaged file: the defaults, with the error.
	if err := os.WriteFile(path, []byte("fontPt 8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if s, _, err := Load(path); err == nil || !reflect.DeepEqual(s, Default()) {
		t.Errorf("Load of a damaged file: %+v, %v", s, err)
	}
}

// The settings start a real session: the shell named is run with the TERM
// named, and a program that asks what the terminal is gets the app's fixed
// answer (a VT220), whatever the terminal type.
func TestTheSettingsReachARunningSession(t *testing.T) {
	s := Default()
	s.Shell = "/bin/bash"
	s.Args = []string{"--norc", "--noprofile", "-c",
		// Ask "what are you" (primary device attributes) and print the answer.
		`echo "term=$TERM"; stty -echo raw; printf '\033[c'; IFS= read -r -s -d c -t 5 a; stty sane; printf 'answer=%q\n' "$a"; sleep 5`}
	s.Term = "vt100"
	sess, err := session.Start(s.Session(60, 10, 9, 18, []string{"PATH=/usr/bin:/bin", "TERM=dumb"}, Bundled{}))
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	deadline := time.Now().Add(10 * time.Second)
	for {
		text, err := sess.Text()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(text, "term=vt100") && strings.Contains(text, `answer=$'\E[?62;22'`) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the screen is %q", text)
		}
		select {
		case <-sess.Changed():
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// The shell setting names the phone's shell, the app's BusyBox shell, or a
// program by its path; and the tools folder goes at the end of the PATH.
func TestTheShellSetting(t *testing.T) {
	env := []string{"HOME=/h", "PATH=/usr/bin:/bin", "ENV=/etc/kshrc", "TERM=dumb"}
	has := func(c session.Config, kv string) bool { return slices.Contains(c.Env, kv) }

	s := Default()
	if s.Shell != ShellSystem {
		t.Fatalf("the default shell is %q", s.Shell)
	}
	c := s.Session(80, 24, 9, 18, env, Bundled{Tools: "/data/bin"})
	if c.Shell != "/bin/sh" {
		t.Errorf("the system shell is %q", c.Shell)
	}
	if !has(c, "PATH=/usr/bin:/bin:/data/bin") {
		t.Errorf("the tools folder is not at the end of the PATH: %v", c.Env)
	}
	if !has(c, "ENV=/etc/kshrc") || !has(c, "HOME=/h") || !has(c, "TERM=xterm") || has(c, "TERM=dumb") {
		t.Errorf("the system shell's environment: %v", c.Env)
	}

	s.Shell = ShellBusyBox
	c = s.Session(80, 24, 9, 18, env, Bundled{Tools: "/data/bin"})
	if c.Shell != "/data/bin/ash" {
		t.Errorf("the BusyBox shell is %q", c.Shell)
	}
	if has(c, "ENV=/etc/kshrc") {
		t.Errorf("the BusyBox shell is given the phone shell's start file: %v", c.Env)
	}
	if !has(c, "PATH=/usr/bin:/bin:/data/bin") {
		t.Errorf("the BusyBox shell's PATH: %v", c.Env)
	}

	// PowerShell: the program from the package, a home and a temporary
	// folder of its own in place of the app's, and xterm whatever the
	// terminal type setting says.
	s.Shell, s.Term = ShellPowerShell, "vt100"
	places := Bundled{Tools: "/data/bin", PowerShell: "/app/native/powershell/pwsh/pwsh", PowerShellHome: "/data/pwsh-home", PowerShellTmp: "/data/pwsh-tmp"}
	c = s.Session(80, 24, 9, 18, append(env, "TMPDIR=/tmp"), places)
	if c.Shell != "/app/native/powershell/pwsh/pwsh" {
		t.Errorf("PowerShell is %q", c.Shell)
	}
	if !has(c, "HOME=/data/pwsh-home") || has(c, "HOME=/h") || !has(c, "TMPDIR=/data/pwsh-tmp") || has(c, "TMPDIR=/tmp") {
		t.Errorf("PowerShell's home and temporary folders: %v", c.Env)
	}
	if !has(c, "TERM=xterm") || has(c, "TERM=vt100") || !has(c, "PATH=/usr/bin:/bin:/data/bin") || !has(c, "ENV=/etc/kshrc") {
		t.Errorf("PowerShell's environment: %v", c.Env)
	}
	s.Term = "xterm"

	s.Shell = "/opt/bin/zsh"
	c = s.Session(80, 24, 9, 18, env, Bundled{Tools: "/data/bin"})
	if c.Shell != "/opt/bin/zsh" || !has(c, "ENV=/etc/kshrc") {
		t.Errorf("a shell by its path: %q, %v", c.Shell, c.Env)
	}

	for shell, name := range map[string]string{ShellSystem: "System shell", ShellBusyBox: "BusyBox shell", ShellPowerShell: "PowerShell", "/opt/bin/zsh": "zsh"} {
		x := Default()
		x.Shell = shell
		if x.ShellName() != name {
			t.Errorf("the shell %q is called %q, want %q", shell, x.ShellName(), name)
		}
	}

	// No tools folder: the PATH is left as it is. No PATH at all: the
	// tools folder is all of it.
	c = Default().Session(80, 24, 9, 18, env, Bundled{})
	if !has(c, "PATH=/usr/bin:/bin") {
		t.Errorf("the PATH without a tools folder: %v", c.Env)
	}
	c = Default().Session(80, 24, 9, 18, []string{"HOME=/h"}, Bundled{Tools: "/data/bin"})
	if !has(c, "PATH=/data/bin") {
		t.Errorf("the PATH where there was none: %v", c.Env)
	}

	// A settings file from before the list, with the phone's shell by its
	// path, means the system shell; any other path is kept as it is.
	old, _, err := Parse([]byte("shell = /bin/sh\n"))
	if err != nil || old.Shell != ShellSystem {
		t.Errorf("shell = /bin/sh is read as %q (%v)", old.Shell, err)
	}
	other, _, err := Parse([]byte("shell = /base/bin/ksh\n"))
	if err != nil || other.Shell != "/base/bin/ksh" {
		t.Errorf("shell = /base/bin/ksh is read as %q (%v)", other.Shell, err)
	}
	for _, v := range []string{ShellSystem, ShellBusyBox, ShellPowerShell} {
		got, _, err := Parse(Settings.Format(func() Settings { x := Default(); x.Shell = v; return x }()))
		if err != nil || got.Shell != v {
			t.Errorf("shell = %s written and read back is %q (%v)", v, got.Shell, err)
		}
	}
}
