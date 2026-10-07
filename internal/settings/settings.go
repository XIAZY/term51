// Package settings is the app's settings: one file, with defaults, checked
// when it is read, giving every other package its configuration.
//
// The file is plain text that a person can edit on the phone:
//
//	# a comment
//	fontPx = 28
//	shell = /bin/sh
//	arg = -l
//	theme.background = #101418
//	identity.terminal = xterm
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
	"strconv"
	"strings"

	"term51/internal/replies"
	"term51/internal/session"
)

// The front ends (internal/app).
const (
	FrontendText   = "text"
	FrontendPixels = "pixels"
)

// Settings is everything the user can set.
type Settings struct {
	// Frontend is how the terminal is shown: FrontendText (Cascades text
	// labels) or FrontendPixels (a window drawn by the app).
	Frontend string
	// RawKeys makes the pixel front end read the window's raw key events
	// instead of Cascades key events.
	RawKeys bool
	// FontPx is the font size in pixels.
	FontPx float64

	// Shell is the program to run and Args its arguments. Term is the TERM
	// it is given.
	Shell string
	Args  []string
	Term  string

	// ScrollbackBytes is the scrollback limit in bytes, as the terminal
	// library counts it.
	ScrollbackBytes uint64

	Theme    Theme
	Identity Identity
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

// Identity is what the terminal tells programs about itself, and which of
// their questions it answers at all.
type Identity struct {
	// Terminal is the model answered to "what are you": "vt220", "vt100"
	// or "xterm".
	Terminal string
	// Name and Version are reported to the "what is your name and version"
	// query (XTVERSION).
	Name    string
	Version string
	// VersionNumber is the firmware version in the secondary device
	// attributes answer.
	VersionNumber int
	// Answerback is sent for ENQ (Ctrl-E); empty sends nothing.
	Answerback string
	// Dark is the colour scheme reported when a program asks: dark or
	// light.
	Dark bool

	// Which kinds of question are answered.
	DeviceAttributes bool
	StatusReports    bool
	ColorScheme      bool
	ModeQueries      bool
	ColorQueries     bool
	SizeReports      bool
	KittyKeyboard    bool
}

var terminals = map[string]replies.Identity{
	"vt220": replies.VT220,
	"vt100": replies.VT100,
	"xterm": replies.XTerm,
}

func terminalName(id replies.Identity) string {
	for name, v := range terminals {
		if v == id {
			return name
		}
	}
	return "vt220"
}

func hex(c session.RGB) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// Default is the settings of a fresh install. It suits the Passport: a
// 1440 pixel wide screen at about 450 pixels per inch.
func Default() Settings {
	r := replies.DefaultConfig()
	t := session.DefaultTheme
	return Settings{
		Frontend:        FrontendText,
		FontPx:          34,
		Shell:           "/bin/sh",
		Args:            []string{},
		Term:            session.DefaultTerm,
		ScrollbackBytes: 8 << 20,
		Theme: Theme{
			Foreground: hex(t.Foreground),
			Background: hex(t.Background),
			Palette:    []string{},
		},
		Identity: Identity{
			Terminal:         terminalName(r.Identity),
			Name:             r.Name,
			Version:          r.Version,
			VersionNumber:    r.VersionNumber,
			Answerback:       r.Answerback,
			Dark:             r.Dark,
			DeviceAttributes: r.DeviceAttributes,
			StatusReports:    r.StatusReports,
			ColorScheme:      r.ColorScheme,
			ModeQueries:      r.ModeQueries,
			ColorQueries:     r.ColorQueries,
			SizeReports:      r.SizeReports,
			KittyKeyboard:    r.KittyKeyboard,
		},
	}
}

// The limits a value is checked against.
const (
	MinFontPx          = 6
	MaxFontPx          = 200
	MaxScrollbackBytes = 256 << 20 // the phone has about 2 GB for everything
	maxTextLen         = 256       // names, versions, the answerback
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

// printable reports whether s can be sent to a program as an answer: no
// control characters, which could be read as commands.
func printable(s string) bool {
	for _, c := range s {
		if c < 0x20 || c == 0x7F || (c >= 0x80 && c < 0xA0) {
			return false
		}
	}
	return true
}

// Validate reports the first setting whose value makes no sense.
func (s Settings) Validate() error {
	bad := func(name, format string, args ...any) error {
		return fmt.Errorf("settings: %s: %s", name, fmt.Sprintf(format, args...))
	}
	if s.Frontend != FrontendText && s.Frontend != FrontendPixels {
		return bad("frontend", "%q is neither %q nor %q", s.Frontend, FrontendText, FrontendPixels)
	}
	if !(s.FontPx >= MinFontPx && s.FontPx <= MaxFontPx) {
		return bad("fontPx", "%v is not between %d and %d", s.FontPx, MinFontPx, MaxFontPx)
	}
	if s.Shell == "" {
		return bad("shell", "empty")
	}
	if strings.ContainsRune(s.Shell, 0) {
		return bad("shell", "contains a NUL character")
	}
	for i, a := range s.Args {
		if strings.ContainsRune(a, 0) {
			return bad(fmt.Sprintf("args[%d]", i), "contains a NUL character")
		}
	}
	if s.Term == "" || !printable(s.Term) || strings.ContainsAny(s.Term, " =") {
		return bad("term", "%q is not a terminal name", s.Term)
	}
	if s.ScrollbackBytes > MaxScrollbackBytes {
		return bad("scrollbackBytes", "%d is more than %d", s.ScrollbackBytes, MaxScrollbackBytes)
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

	id := s.Identity
	if _, ok := terminals[id.Terminal]; !ok {
		return bad("identity.terminal", "%q is not one of vt220, vt100, xterm", id.Terminal)
	}
	for _, f := range []struct{ name, value string }{
		{"identity.name", id.Name}, {"identity.version", id.Version}, {"identity.answerback", id.Answerback},
	} {
		if len(f.value) > maxTextLen {
			return bad(f.name, "longer than %d bytes", maxTextLen)
		}
		if !printable(f.value) {
			return bad(f.name, "contains a control character")
		}
	}
	if id.VersionNumber < 0 || id.VersionNumber > 99999 {
		return bad("identity.versionNumber", "%d is not between 0 and 99999", id.VersionNumber)
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
	textField("frontend", "How the terminal is shown: text (Cascades text labels) or pixels (a window drawn by the app).",
		func(s *Settings) *string { return &s.Frontend }),
	boolField("rawKeys", "With the pixels front end, read the window's raw key events instead of Cascades key events.",
		func(s *Settings) *bool { return &s.RawKeys }),
	{"fontPx", "The font size in pixels.",
		func(s *Settings, v string) error {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("%q is not a number", v)
			}
			s.FontPx = f
			return nil
		},
		func(s *Settings) []string { return []string{strconv.FormatFloat(s.FontPx, 'g', -1, 64)} }},
	textField("shell", "The program to run.", func(s *Settings) *string { return &s.Shell }),
	{"arg", "An argument for the shell; one line per argument.",
		func(s *Settings, v string) error { s.Args = append(s.Args, v); return nil },
		func(s *Settings) []string { return s.Args }},
	textField("term", "The TERM the shell is given.", func(s *Settings) *string { return &s.Term }),
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

	textField("theme.foreground", "The text colour, as #rrggbb.", func(s *Settings) *string { return &s.Theme.Foreground }),
	textField("theme.background", "The background colour.", func(s *Settings) *string { return &s.Theme.Background }),
	textField("theme.cursor", "The cursor's colour; empty draws it in the text colour.", func(s *Settings) *string { return &s.Theme.Cursor }),
	{"theme.palette", "The first colours of the 256-colour palette (the 16 ANSI colours, usually); one line per colour, in order.",
		func(s *Settings, v string) error { s.Theme.Palette = append(s.Theme.Palette, v); return nil },
		func(s *Settings) []string { return s.Theme.Palette }},

	textField("identity.terminal", "The model answered to \"what are you\": vt220, vt100 or xterm.",
		func(s *Settings) *string { return &s.Identity.Terminal }),
	textField("identity.name", "The name reported to the name-and-version query.", func(s *Settings) *string { return &s.Identity.Name }),
	textField("identity.version", "The version reported with it.", func(s *Settings) *string { return &s.Identity.Version }),
	{"identity.versionNumber", "The firmware version in the secondary device attributes answer.",
		func(s *Settings, v string) error {
			n, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("%q is not a whole number", v)
			}
			s.Identity.VersionNumber = n
			return nil
		},
		func(s *Settings) []string { return []string{strconv.Itoa(s.Identity.VersionNumber)} }},
	textField("identity.answerback", "Sent when a program sends ENQ (Ctrl-E); empty sends nothing.",
		func(s *Settings) *string { return &s.Identity.Answerback }),
	boolField("identity.dark", "The colour scheme reported when a program asks: true for dark, false for light.",
		func(s *Settings) *bool { return &s.Identity.Dark }),
	boolField("identity.deviceAttributes", "Answer \"what are you\" and the name-and-version query.",
		func(s *Settings) *bool { return &s.Identity.DeviceAttributes }),
	boolField("identity.statusReports", "Answer status and cursor position queries.",
		func(s *Settings) *bool { return &s.Identity.StatusReports }),
	boolField("identity.colorScheme", "Answer the dark-or-light query.",
		func(s *Settings) *bool { return &s.Identity.ColorScheme }),
	boolField("identity.modeQueries", "Answer queries about terminal modes.",
		func(s *Settings) *bool { return &s.Identity.ModeQueries }),
	boolField("identity.colorQueries", "Answer queries about colours.",
		func(s *Settings) *bool { return &s.Identity.ColorQueries }),
	boolField("identity.sizeReports", "Answer queries about the size of the terminal.",
		func(s *Settings) *bool { return &s.Identity.SizeReports }),
	boolField("identity.kittyKeyboard", "Answer the keyboard protocol query.",
		func(s *Settings) *bool { return &s.Identity.KittyKeyboard }),
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

// Load reads the settings file. A file that does not exist is the defaults,
// with no error. For any other failure it returns the defaults and the
// error, so the app can start and say what is wrong.
func Load(path string) (Settings, []string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil, nil
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

// Replies is the configuration of the answers to programs' queries.
func (s Settings) Replies() replies.Config {
	id := s.Identity
	return replies.Config{
		Identity:         terminals[id.Terminal],
		Name:             id.Name,
		Version:          id.Version,
		VersionNumber:    id.VersionNumber,
		DeviceAttributes: id.DeviceAttributes,
		StatusReports:    id.StatusReports,
		ColorScheme:      id.ColorScheme,
		ModeQueries:      id.ModeQueries,
		ColorQueries:     id.ColorQueries,
		SizeReports:      id.SizeReports,
		KittyKeyboard:    id.KittyKeyboard,
		Dark:             id.Dark,
		Answerback:       id.Answerback,
	}
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
		Replies:       s.Replies(),
	}
}
