// Package replies is term51's own answers to the questions programs ask the
// terminal: "what terminal are you", "where is the cursor", "is this mode
// set", "what colour is the background", "how big is the window", and the
// Kitty keyboard flags. Programs print a query as an escape sequence and wait
// for the terminal to type the answer back; some pause or draw wrongly when
// nobody answers.
//
// The terminal library recognises the queries and hands them to the app; this
// package decides the answer. It has its own small types (Query, State) so it
// does not depend on the library: the session translates the library's
// actions into a Query and implements State on top of the terminal. Every
// kind of answer can be switched off in Config, for programs that misbehave;
// unknown or switched-off queries get no answer.
package replies

import (
	"fmt"
	"strconv"
	"strings"
)

// Identity is what the terminal answers to the primary device attributes
// query ("what terminal are you").
type Identity int

const (
	// VT220 answers as a VT220 with ANSI colour: CSI ? 62 ; 22 c.
	VT220 Identity = iota
	// VT100 answers as a VT100 with advanced video: CSI ? 1 ; 2 c.
	VT100
	// XTerm answers like xterm in VT420 mode with its usual feature list:
	// CSI ? 64 ; 1 ; 2 ; 6 ; 9 ; 15 ; 16 ; 17 ; 18 ; 21 ; 22 ; 28 c.
	XTerm
)

// Config chooses the identity and which kinds of query are answered. The
// zero value answers nothing; use DefaultConfig.
type Config struct {
	// Identity answered to the primary device attributes query.
	Identity Identity
	// Name and Version are reported to XTVERSION (CSI > q) and, for the
	// version number, to the secondary device attributes query.
	Name    string
	Version string
	// VersionNumber is the firmware version in the secondary device
	// attributes answer (CSI > Pp ; Pv ; 0 c).
	VersionNumber int

	DeviceAttributes bool // primary, secondary and tertiary DA, XTVERSION
	StatusReports    bool // DSR 5 (status) and 6 (cursor position)
	ColorScheme      bool // DSR ?996 (dark or light)
	ModeQueries      bool // DECRQM
	ColorQueries     bool // OSC 4, 10, 11, 12 queries
	SizeReports      bool // CSI 14 t, 16 t, 18 t
	KittyKeyboard    bool // CSI ? u

	// Dark is the colour scheme reported to DSR ?996.
	Dark bool
	// Answerback is sent for ENQ (Ctrl-E); empty sends nothing.
	Answerback string
}

// DefaultConfig is what term51 uses unless the user changes it.
func DefaultConfig() Config {
	return Config{
		Identity:         VT220,
		Name:             "term51",
		Version:          "0.1",
		VersionNumber:    1,
		DeviceAttributes: true,
		StatusReports:    true,
		ColorScheme:      true,
		ModeQueries:      true,
		ColorQueries:     true,
		SizeReports:      true,
		KittyKeyboard:    true,
		Dark:             true,
	}
}

// Kind is the kind of query.
type Kind int

const (
	PrimaryDA      Kind = iota // CSI c
	SecondaryDA                // CSI > c
	TertiaryDA                 // CSI = c
	XTVersion                  // CSI > q
	StatusReport               // CSI 5 n
	CursorReport               // CSI 6 n
	ColorScheme                // CSI ? 996 n
	ModeQuery                  // CSI Pm $ p (ANSI) or CSI ? Pm $ p (DEC)
	ColorQuery                 // OSC 4 ; i ; ? / OSC 10, 11, 12 ; ?
	TextAreaChars              // CSI 18 t
	TextAreaPixels             // CSI 14 t
	CellPixels                 // CSI 16 t
	KittyFlags                 // CSI ? u
	Enquiry                    // ENQ
)

// ColorTarget is which colour an OSC colour query asks for.
type ColorTarget int

const (
	Palette    ColorTarget = iota // OSC 4 ; index
	Foreground                    // OSC 10
	Background                    // OSC 11
	Cursor                        // OSC 12
)

// Query is one question from a program.
type Query struct {
	Kind Kind

	// ModeQuery: the mode number and whether it is a DEC private mode.
	Mode int
	DEC  bool

	// ColorQuery: which colour, the palette index for Palette, and whether
	// the query ended with BEL (the answer ends the same way; ST otherwise).
	Target ColorTarget
	Index  int
	BEL    bool
}

// ModeState is the answer to a mode query (DECRPM).
type ModeState int

const (
	ModeUnknown        ModeState = 0
	ModeSet            ModeState = 1
	ModeReset          ModeState = 2
	ModePermanentSet   ModeState = 3
	ModePermanentReset ModeState = 4
)

// RGB is a colour.
type RGB struct{ R, G, B uint8 }

// State is what the answers need to know about the terminal.
type State interface {
	// CursorReportPos is the cursor position to report, 1-based, already
	// relative to the scrolling region when origin mode is on.
	CursorReportPos() (row, col int)
	// Size is the text area in cells.
	Size() (cols, rows int)
	// CellSize is one cell in pixels.
	CellSize() (width, height int)
	// Mode reports the state of a mode.
	Mode(mode int, dec bool) ModeState
	// KittyKeyboardFlags is the active screen's current Kitty keyboard flags.
	KittyKeyboardFlags() int
	// Color returns a colour; ok is false when it is unset (the default
	// colours can be unset).
	Color(target ColorTarget, index int) (c RGB, ok bool)
}

const (
	csi = "\x1b["
	osc = "\x1b]"
	st  = "\x1b\\"
	dcs = "\x1bP"
	bel = "\a"
)

// Answer returns the bytes to type back to the program for q, or nil when the
// query is unknown, switched off in cfg, or has no answer.
func Answer(cfg Config, s State, q Query) []byte {
	switch q.Kind {
	case PrimaryDA:
		if !cfg.DeviceAttributes {
			return nil
		}
		switch cfg.Identity {
		case VT100:
			return []byte(csi + "?1;2c")
		case XTerm:
			return []byte(csi + "?64;1;2;6;9;15;16;17;18;21;22;28c")
		default:
			return []byte(csi + "?62;22c")
		}
	case SecondaryDA:
		if !cfg.DeviceAttributes {
			return nil
		}
		// Pp: 1 is a VT220, 41 a VT420 (xterm's choice); Pv: firmware
		// version; Pc: 0 (no ROM cartridge).
		pp := 1
		switch cfg.Identity {
		case VT100:
			pp = 0
		case XTerm:
			pp = 41
		}
		return []byte(fmt.Sprintf("%s>%d;%d;0c", csi, pp, cfg.VersionNumber))
	case TertiaryDA:
		if !cfg.DeviceAttributes {
			return nil
		}
		// DECRPTUI: a unit ID of zeros, as xterm reports.
		return []byte(dcs + "!|00000000" + st)
	case XTVersion:
		if !cfg.DeviceAttributes {
			return nil
		}
		name := strings.TrimSpace(cfg.Name + " " + cfg.Version)
		return []byte(dcs + ">|" + name + st)
	case StatusReport:
		if !cfg.StatusReports {
			return nil
		}
		return []byte(csi + "0n")
	case CursorReport:
		if !cfg.StatusReports {
			return nil
		}
		row, col := s.CursorReportPos()
		return []byte(fmt.Sprintf("%s%d;%dR", csi, row, col))
	case ColorScheme:
		if !cfg.ColorScheme {
			return nil
		}
		if cfg.Dark {
			return []byte(csi + "?997;1n")
		}
		return []byte(csi + "?997;2n")
	case ModeQuery:
		if !cfg.ModeQueries {
			return nil
		}
		st := s.Mode(q.Mode, q.DEC)
		if q.DEC {
			return []byte(fmt.Sprintf("%s?%d;%d$y", csi, q.Mode, int(st)))
		}
		return []byte(fmt.Sprintf("%s%d;%d$y", csi, q.Mode, int(st)))
	case ColorQuery:
		if !cfg.ColorQueries {
			return nil
		}
		c, ok := s.Color(q.Target, q.Index)
		if !ok {
			return nil
		}
		end := st
		if q.BEL {
			end = bel
		}
		var prefix string
		switch q.Target {
		case Palette:
			prefix = "4;" + strconv.Itoa(q.Index)
		case Foreground:
			prefix = "10"
		case Background:
			prefix = "11"
		case Cursor:
			prefix = "12"
		}
		// xterm's reply format: 16 bits per channel, each 8-bit value
		// repeated.
		return []byte(fmt.Sprintf("%s%s;rgb:%02x%02x/%02x%02x/%02x%02x%s",
			osc, prefix, c.R, c.R, c.G, c.G, c.B, c.B, end))
	case TextAreaChars:
		if !cfg.SizeReports {
			return nil
		}
		cols, rows := s.Size()
		return []byte(fmt.Sprintf("%s8;%d;%dt", csi, rows, cols))
	case TextAreaPixels:
		if !cfg.SizeReports {
			return nil
		}
		cols, rows := s.Size()
		w, h := s.CellSize()
		return []byte(fmt.Sprintf("%s4;%d;%dt", csi, rows*h, cols*w))
	case CellPixels:
		if !cfg.SizeReports {
			return nil
		}
		w, h := s.CellSize()
		return []byte(fmt.Sprintf("%s6;%d;%dt", csi, h, w))
	case KittyFlags:
		if !cfg.KittyKeyboard {
			return nil
		}
		return []byte(fmt.Sprintf("%s?%du", csi, s.KittyKeyboardFlags()))
	case Enquiry:
		if cfg.Answerback == "" {
			return nil
		}
		return []byte(cfg.Answerback)
	}
	return nil
}
