// Package settings is the app's settings: one file, with defaults, checked
// when it is read, giving every other package its configuration.
//
// The file is plain text that a person can edit on the phone:
//
//	# a comment
//	fontPt = 7
//	shell = /bin/sh
//	arg = -l
//	theme.background = #101418
//	term = xterm
//
// One setting per line, `name = value`. A line starting with # is a comment.
// `arg` and `theme.palette` may be given several times, one value each. A
// value may be put in double quotes to keep spaces at its ends or to be
// empty.
//
// A setting left out keeps its default, so an empty file is the defaults. A
// name that is not a setting is a warning, not an error, so that an older
// app still reads a newer file. A value that makes no sense is an error that
// names the setting and the line, and nothing of a bad file is used: the app
// then runs on the defaults and can show the error.
package settings

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"term51/internal/replies"
	"term51/internal/session"
)

// Settings is everything the user can set.
type Settings struct {
	// FontPt is the size of the terminal's text, in points.
	FontPt float64

	// Shell is the program to run and Args its arguments. Term is the TERM
	// it is given.
	Shell string
	Args  []string
	Term  string

	// ScrollbackBytes is the scrollback limit in bytes, as the terminal
	// library counts it.
	ScrollbackBytes uint64

	// Cursor is the cursor's shape while no program asks for one: "block",
	// "bar" or "underline".
	Cursor string

	// Bell makes a sound when a program rings the bell.
	Bell bool

	Theme Theme
}

var cursors = map[string]session.CursorShape{
	"block":     session.CursorBlock,
	"bar":       session.CursorBar,
	"underline": session.CursorUnderline,
}

// Terms are the terminal types the shell can be given as TERM, the default
// first. The phone's own terminal database (/usr/lib/terminfo) has xterm,
// vt100, ansi and qansi (QNX's own), so the default is one its programs
// know. It has no xterm-256color: that one is for programs on other
// machines, over ssh, where the other machine's database counts.
var Terms = []string{"xterm", "xterm-256color", "vt100", "ansi", "qansi"}

// Themes are the colour sets the settings sheet offers, by name. A theme
// is not a setting of its own: choosing one fills the four colour settings,
// and the theme in force is the one whose colours they hold.
var Themes = map[string]Theme{
	// Light grey on near black, the library's palette.
	"dark": {Foreground: "#d0d0d0", Background: "#101418", Palette: []string{}},
	// The same on pure black.
	"black": {Foreground: "#d0d0d0", Background: "#000000", Palette: []string{}},
	// Dark text on off-white, with colours dark enough to read on it.
	"light": {Foreground: "#202428", Background: "#fafaf5", Palette: []string{
		"#202428", "#c4262e", "#2e8b3d", "#9a6a00", "#1f5fbf", "#8b3fa8", "#127f8c", "#b8bcc0",
		"#5c6166", "#e0424a", "#3aa34d", "#b58100", "#2f76e0", "#a653c4", "#1b98a8", "#ffffff"}},
	// Solarized dark.
	"solarized": {Foreground: "#839496", Background: "#002b36", Palette: []string{
		"#073642", "#dc322f", "#859900", "#b58900", "#268bd2", "#d33682", "#2aa198", "#eee8d5",
		"#002b36", "#cb4b16", "#586e75", "#657b83", "#839496", "#6c71c4", "#93a1a1", "#fdf6e3"}},
}

// ThemeName is the name of the theme whose colours the settings hold, or
// "custom" when they are someone's own.
func (s Settings) ThemeName() string {
	for name, t := range Themes {
		if strings.EqualFold(s.Theme.Foreground, t.Foreground) && strings.EqualFold(s.Theme.Background, t.Background) &&
			s.Theme.Cursor == t.Cursor && len(s.Theme.Palette) == len(t.Palette) {
			same := true
			for i := range t.Palette {
				same = same && strings.EqualFold(s.Theme.Palette[i], t.Palette[i])
			}
			if same {
				return name
			}
		}
	}
	return "custom"
}

// Theme is the terminal's colours, each as "#rrggbb".
type Theme struct {
	Foreground string
	Background string
	// Cursor is the cursor's colour; empty draws it in the text colour.
	Cursor string
	// Palette replaces the first colours of the 256-colour palette (the 16
	// ANSI colours, usually); empty keeps the terminal library's.
	Palette []string
}

func hex(c session.RGB) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// Default is the settings of a fresh install. It suits the Passport: a
// 1440 pixel wide screen at about 450 pixels per inch.
func Default() Settings {
	t := session.DefaultTheme
	return Settings{
		FontPt:          7,
		Shell:           "/bin/sh",
		Args:            []string{},
		Term:            Terms[0],
		ScrollbackBytes: 8 << 20,
		Cursor:          "block",
		Bell:            true,
		Theme: Theme{
			Foreground: hex(t.Foreground),
			Background: hex(t.Background),
			Palette:    []string{},
		},
	}
}

// The limits a value is checked against.
const (
	MinFontPt          = 3
	MaxFontPt          = 40
	MaxScrollbackBytes = 256 << 20 // the phone has about 2 GB for everything
)

// parseColor reads "#rrggbb".
func parseColor(s string) (session.RGB, error) {
	if len(s) != 7 || s[0] != '#' {
		return session.RGB{}, fmt.Errorf("%q is not a colour of the form #rrggbb", s)
	}
	var v [3]uint8
	for i := 0; i < 3; i++ {
		var b uint8
		for _, c := range []byte(s[1+2*i : 3+2*i]) {
			switch {
			case c >= '0' && c <= '9':
				b = b<<4 | (c - '0')
			case c >= 'a' && c <= 'f':
				b = b<<4 | (c - 'a' + 10)
			case c >= 'A' && c <= 'F':
				b = b<<4 | (c - 'A' + 10)
			default:
				return session.RGB{}, fmt.Errorf("%q is not a colour of the form #rrggbb", s)
			}
		}
		v[i] = b
	}
	return session.RGB{R: v[0], G: v[1], B: v[2]}, nil
}

// Validate reports the first setting whose value makes no sense.
func (s Settings) Validate() error {
	bad := func(name, format string, args ...any) error {
		return fmt.Errorf("settings: %s: %s", name, fmt.Sprintf(format, args...))
	}
	if !(s.FontPt >= MinFontPt && s.FontPt <= MaxFontPt) {
		return bad("fontPt", "%v is not between %d and %d", s.FontPt, MinFontPt, MaxFontPt)
	}
	if s.Shell == "" {
		return bad("shell", "empty")
	}
	if !slices.Contains(Terms, s.Term) {
		return bad("term", "%q is not one of %s", s.Term, strings.Join(Terms, ", "))
	}
	if s.ScrollbackBytes > MaxScrollbackBytes {
		return bad("scrollbackBytes", "%d is more than %d", s.ScrollbackBytes, MaxScrollbackBytes)
	}
	if _, ok := cursors[s.Cursor]; !ok {
		return bad("cursor", "%q is not one of block, bar, underline", s.Cursor)
	}

	if _, err := parseColor(s.Theme.Foreground); err != nil {
		return bad("theme.foreground", "%v", err)
	}
	if _, err := parseColor(s.Theme.Background); err != nil {
		return bad("theme.background", "%v", err)
	}
	if s.Theme.Cursor != "" {
		if _, err := parseColor(s.Theme.Cursor); err != nil {
			return bad("theme.cursor", "%v", err)
		}
	}
	if len(s.Theme.Palette) > 256 {
		return bad("theme.palette", "%d colours; the palette has 256", len(s.Theme.Palette))
	}
	for i, c := range s.Theme.Palette {
		if _, err := parseColor(c); err != nil {
			return bad(fmt.Sprintf("theme.palette[%d]", i), "%v", err)
		}
	}
	return nil
}

// A field is one line of the file: how its value is read and written.
type field struct {
	name    string
	comment string
	set     func(s *Settings, v string) error
	get     func(s *Settings) []string // one value per line
}

func parseBool(v string) (bool, error) {
	switch strings.ToLower(v) {
	case "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0":
		return false, nil
	}
	return false, fmt.Errorf("%q is neither true nor false", v)
}

func boolField(name, comment string, p func(*Settings) *bool) field {
	return field{name, comment,
		func(s *Settings, v string) error {
			b, err := parseBool(v)
			if err == nil {
				*p(s) = b
			}
			return err
		},
		func(s *Settings) []string { return []string{strconv.FormatBool(*p(s))} }}
}

func textField(name, comment string, p func(*Settings) *string) field {
	return field{name, comment,
		func(s *Settings, v string) error { *p(s) = v; return nil },
		func(s *Settings) []string { return []string{*p(s)} }}
}

// fields is every setting, in the order the file is written.
var fields = []field{
	{"fontPt", "The size of the terminal's text, in points.",
		func(s *Settings, v string) error {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("%q is not a number", v)
			}
			s.FontPt = f
			return nil
		},
		func(s *Settings) []string { return []string{strconv.FormatFloat(s.FontPt, 'g', -1, 64)} }},
	textField("shell", "The program to run.", func(s *Settings) *string { return &s.Shell }),
	{"arg", "An argument for the shell; one line per argument.",
		func(s *Settings, v string) error { s.Args = append(s.Args, v); return nil },
		func(s *Settings) []string { return s.Args }},
	textField("term", "The terminal type (TERM) the shell is given: "+strings.Join(Terms, ", ")+".", func(s *Settings) *string { return &s.Term }),
	{"scrollbackBytes", "The scrollback limit in bytes.",
		func(s *Settings, v string) error {
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return fmt.Errorf("%q is not a whole number of bytes", v)
			}
			s.ScrollbackBytes = n
			return nil
		},
		func(s *Settings) []string { return []string{strconv.FormatUint(s.ScrollbackBytes, 10)} }},
	textField("cursor", "The cursor's shape while no program asks for one: block, bar or underline.", func(s *Settings) *string { return &s.Cursor }),
	boolField("bell", "Make a sound when a program rings the bell.", func(s *Settings) *bool { return &s.Bell }),

	textField("theme.foreground", "The text colour, as #rrggbb.", func(s *Settings) *string { return &s.Theme.Foreground }),
	textField("theme.background", "The background colour.", func(s *Settings) *string { return &s.Theme.Background }),
	textField("theme.cursor", "The cursor's colour; empty draws it in the text colour.", func(s *Settings) *string { return &s.Theme.Cursor }),
	{"theme.palette", "The first colours of the 256-colour palette (the 16 ANSI colours, usually); one line per colour, in order.",
		func(s *Settings, v string) error { s.Theme.Palette = append(s.Theme.Palette, v); return nil },
		func(s *Settings) []string { return s.Theme.Palette }},
}

// repeated is the settings that may be given several times.
var repeated = map[string]bool{"arg": true, "theme.palette": true}

func fieldByName(name string) *field {
	for i := range fields {
		if strings.EqualFold(fields[i].name, name) {
			return &fields[i]
		}
	}
	return nil
}

// Values is every setting that has one value, by its name in the file, as
// the file would hold it. (The two lists, arg and theme.palette, are not in
// it.) "theme" is the name of the theme the colours make (ThemeName).
func (s Settings) Values() map[string]string {
	out := make(map[string]string, len(fields)+1)
	out["theme"] = s.ThemeName()
	for i := range fields {
		if f := &fields[i]; !repeated[f.name] {
			out[f.name] = f.get(&s)[0]
		}
	}
	return out
}

// Set changes one setting, named and written as in the file. A name that is
// not a single-valued setting, or a value that makes no sense, is an error
// and changes nothing.
func (s *Settings) Set(name, value string) error {
	if name == "theme" {
		// Not a setting of the file: one of Themes, which fills the
		// colour settings.
		t, ok := Themes[value]
		if !ok {
			return fmt.Errorf("settings: theme: %q is not a theme", value)
		}
		t.Palette = append([]string{}, t.Palette...)
		s.Theme = t
		return nil
	}
	f := fieldByName(name)
	if f == nil || repeated[f.name] {
		return fmt.Errorf("settings: %q is not a setting", name)
	}
	next := *s
	if err := f.set(&next, value); err != nil {
		return fmt.Errorf("settings: %s: %v", f.name, err)
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*s = next
	return nil
}

// quote writes a value so that Parse reads it back: in double quotes when it
// is empty, has spaces at its ends, or starts with a quote.
func quote(v string) string {
	if v == "" || v != strings.TrimSpace(v) || strings.HasPrefix(v, `"`) || strings.ContainsAny(v, "\n\r") {
		return strconv.Quote(v)
	}
	return v
}

// Parse reads the settings file's text. Settings that are left out keep
// their defaults. It returns a warning for each line whose name is not a
// setting; a line that cannot be read or a value that makes no sense is an
// error, and then the defaults are returned.
func Parse(data []byte) (Settings, []string, error) {
	s := Default()
	var warnings []string
	seen := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		name, value = strings.TrimSpace(name), strings.TrimSpace(value)
		if !ok || name == "" {
			return Default(), nil, fmt.Errorf("settings: line %d: %q is not of the form name = value", n, line)
		}
		if strings.HasPrefix(value, `"`) {
			u, err := strconv.Unquote(value)
			if err != nil {
				return Default(), nil, fmt.Errorf("settings: line %d: %s: the quoted value %s is not closed or has a bad escape", n, name, value)
			}
			value = u
		}
		f := fieldByName(name)
		if f == nil {
			warnings = append(warnings, fmt.Sprintf("settings: line %d: %q is not a setting of this version; ignored", n, name))
			continue
		}
		if repeated[f.name] {
			// The first line of a list replaces the default list.
			if !seen[f.name] {
				switch f.name {
				case "arg":
					s.Args = []string{}
				case "theme.palette":
					s.Theme.Palette = []string{}
				}
			}
		} else if seen[f.name] {
			return Default(), nil, fmt.Errorf("settings: line %d: %s is set twice", n, f.name)
		}
		seen[f.name] = true
		if err := f.set(&s, value); err != nil {
			return Default(), nil, fmt.Errorf("settings: line %d: %s: %v", n, f.name, err)
		}
	}
	if err := sc.Err(); err != nil {
		return Default(), nil, fmt.Errorf("settings: %w", err)
	}
	if err := s.Validate(); err != nil {
		return Default(), nil, err
	}
	return s, warnings, nil
}

// Format is the settings as the text of a settings file, every setting
// written out with a comment saying what it is.
func (s Settings) Format() []byte {
	var b bytes.Buffer
	b.WriteString("# term51 settings. One setting per line: name = value.\n")
	b.WriteString("# A line that is removed goes back to its default.\n")
	for i := range fields {
		f := &fields[i]
		b.WriteString("\n# " + f.comment + "\n")
		values := f.get(&s)
		if len(values) == 0 {
			b.WriteString("# " + f.name + " = \n")
		}
		for _, v := range values {
			b.WriteString(f.name + " = " + quote(v) + "\n")
		}
	}
	return b.Bytes()
}

// Load reads the settings file. The first time, when there is no file, it
// writes one holding the defaults, every setting with a comment saying what
// it is, so that the user has something to edit. For a file it cannot read
// or that has a mistake in it, it returns the defaults and the error, so the
// app can start and say what is wrong.
func Load(path string) (Settings, []string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		s := Default()
		if err := Save(path, s); err != nil {
			return s, nil, fmt.Errorf("settings: %w", err)
		}
		return s, nil, nil
	}
	if err != nil {
		return Default(), nil, fmt.Errorf("settings: %w", err)
	}
	return Parse(data)
}

// Save writes the settings file, replacing it in one step so that a crash
// part-way leaves the old file. Comments a person added to the file are not
// kept.
func Save(path string, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	data := s.Format()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".settings-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), path)
	}
	if werr != nil {
		os.Remove(tmp.Name())
		return werr
	}
	return nil
}

// Replies is the configuration of the answers to programs' queries. What
// the terminal says about itself is not a setting: it is the replies
// package's own, except the colour scheme, which is the one the colours make.
func (s Settings) Replies() replies.Config {
	r := replies.DefaultConfig()
	r.Dark = s.SessionTheme().Dark()
	return r
}

// SessionTheme is the theme a session starts with.
func (s Settings) SessionTheme() session.Theme {
	// Validate has accepted the colours.
	t := session.Theme{}
	t.Foreground, _ = parseColor(s.Theme.Foreground)
	t.Background, _ = parseColor(s.Theme.Background)
	if s.Theme.Cursor != "" {
		c, _ := parseColor(s.Theme.Cursor)
		t.Cursor = &c
	}
	for _, p := range s.Theme.Palette {
		c, _ := parseColor(p)
		t.Palette = append(t.Palette, c)
	}
	return t
}

// SessionCursor is the cursor's shape as a session takes it.
func (s Settings) SessionCursor() session.CursorShape { return cursors[s.Cursor] }

// Session is the session to start for a terminal of cols by rows cells of
// cellW by cellH pixels. env is the environment the shell gets besides
// TERM (nil: the app's own).
func (s Settings) Session(cols, rows, cellW, cellH int, env []string) session.Config {
	if env == nil {
		env = os.Environ()
	}
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if !strings.HasPrefix(kv, "TERM=") {
			out = append(out, kv)
		}
	}
	out = append(out, "TERM="+s.Term)
	return session.Config{
		Shell: s.Shell, Args: append([]string(nil), s.Args...), Env: out,
		Cols: cols, Rows: rows, CellW: cellW, CellH: cellH,
		MaxScrollback: s.ScrollbackBytes,
		Theme:         s.SessionTheme(),
		Cursor:        s.SessionCursor(),
		Replies:       s.Replies(),
	}
}
