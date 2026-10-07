package settings

import (
	"os"
	"path/filepath"
	"reflect"
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
fontPx = 20
  identity.terminal=xterm
theme.cursor = #FF8000
IDENTITY.DARK = no
identity.answerback = " hello "
`))
	if err != nil || len(warnings) != 0 {
		t.Fatal(warnings, err)
	}
	want := Default()
	want.FontPx = 20
	want.Identity.Terminal = "xterm"
	want.Theme.Cursor = "#FF8000"
	want.Identity.Dark = false
	want.Identity.Answerback = " hello "
	if !reflect.DeepEqual(s, want) {
		t.Errorf("a partial file:\n got %+v\nwant %+v", s, want)
	}
	if s.Replies().Identity != replies.XTerm {
		t.Error("identity.terminal xterm did not reach the replies")
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
	s, warnings, err := Parse([]byte("fontPx = 20\ntabs.max = 8\nligatures = on\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.FontPx != 20 {
		t.Errorf("fontPx %v, want 20", s.FontPx)
	}
	if len(warnings) != 2 || !strings.Contains(warnings[0], "line 2") || !strings.Contains(warnings[0], "tabs.max") || !strings.Contains(warnings[1], "ligatures") {
		t.Errorf("warnings: %q", warnings)
	}
}

func TestABadValueNamesTheSettingAndNothingIsUsed(t *testing.T) {
	cases := []struct{ in, names string }{
		{"frontend = opengl", "frontend"},
		{"fontPx = 0", "fontPx"},
		{"fontPx = 1000", "fontPx"},
		{"fontPx = big", "fontPx"},
		{`shell = ""`, "shell"},
		{"term = xterm 256", "term"},
		{`term = ""`, "term"},
		{"scrollbackBytes = 99999999999", "scrollbackBytes"},
		{"scrollbackBytes = -1", "scrollbackBytes"},
		{"rawKeys = maybe", "rawKeys"},
		{"theme.foreground = red", "theme.foreground"},
		{"theme.background = #12345", "theme.background"},
		{"theme.cursor = #gggggg", "theme.cursor"},
		{"theme.palette = #000000\ntheme.palette = blue", "theme.palette[1]"},
		{"identity.terminal = vt52", "identity.terminal"},
		{`identity.name = "x\u001b[2J"`, "identity.name"},
		{`identity.answerback = "a\nb"`, "identity.answerback"},
		{"identity.versionNumber = -1", "identity.versionNumber"},
		{"fontPx = 20\nfontPx = 30", "fontPx is set twice"},
		{"fontPx = 20\njust some words", "line 4"}, // two lines come before each case
		{`identity.name = "unclosed`, "identity.name"},
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

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.conf")

	// No file: the defaults, and no error.
	s, warnings, err := Load(path)
	if err != nil || warnings != nil || !reflect.DeepEqual(s, Default()) {
		t.Fatalf("Load without a file: %+v, %v", s, err)
	}

	// The defaults written out and read back are the defaults.
	if got, w, err := Parse(Default().Format()); err != nil || len(w) != 0 || !reflect.DeepEqual(got, Default()) {
		t.Errorf("the default file read back: %+v, %v, %v", got, w, err)
	}

	s.Frontend = FrontendPixels
	s.RawKeys = true
	s.FontPx = 28.5
	s.Shell = "/bin/bash"
	s.Args = []string{"-l", "", " spaced ", `"quoted"`}
	s.Term = "xterm"
	s.ScrollbackBytes = 1 << 20
	s.Theme = Theme{Foreground: "#ffffff", Background: "#000080", Cursor: "#ff0000", Palette: []string{"#010203", "#040506"}}
	s.Identity.Terminal = "vt100"
	s.Identity.Answerback = "term51 # not a comment"
	s.Identity.ColorQueries = false
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
	bad.FontPx = -1
	if err := Save(path, bad); err == nil {
		t.Error("Save accepted fontPx -1")
	}
	if again, _, _ := Load(path); !reflect.DeepEqual(again, s) {
		t.Error("a refused Save changed the file")
	}

	// A damaged file: the defaults, with the error.
	if err := os.WriteFile(path, []byte("fontPx 20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if s, _, err := Load(path); err == nil || !reflect.DeepEqual(s, Default()) {
		t.Errorf("Load of a damaged file: %+v, %v", s, err)
	}
}

// The settings start a real session: the shell named is run with the TERM
// named, and the terminal identity reaches a program that asks.
func TestTheSettingsReachARunningSession(t *testing.T) {
	s := Default()
	s.Shell = "/bin/bash"
	s.Args = []string{"--norc", "--noprofile", "-c",
		// Ask "what are you" (primary device attributes) and print the answer.
		`echo "term=$TERM"; stty -echo raw; printf '\033[c'; IFS= read -r -s -d c -t 5 a; stty sane; printf 'answer=%q\n' "$a"; sleep 5`}
	s.Term = "vt100"
	s.Identity.Terminal = "vt100"
	sess, err := session.Start(s.Session(60, 10, 9, 18, []string{"PATH=/usr/bin:/bin", "TERM=dumb"}))
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
		if strings.Contains(text, "term=vt100") && strings.Contains(text, `answer=$'\E[?1;2'`) {
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
