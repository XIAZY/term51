// Package keys turns the phone's key events into the terminal library's key
// events.
//
// BlackBerry 10 delivers a key two ways, and the app may use either:
//
//   - a Cascades KeyEvent: key (the character produced), keycap (the key
//     itself, unshifted), the text, and whether Shift, Alt and Ctrl are held;
//   - a raw Screen event: flags, a key symbol, a key cap and a modifier
//     mask, with the values of <sys/keycodes.h>.
//
// Both are reduced to a Raw event (FromCascades, FromScreen) and translated
// by one function, so the two paths cannot drift apart. The on-screen key
// bar's Ctrl, Alt and Esc go through the same Translator: Ctrl and Alt as
// sticky modifiers that apply to the next key, Esc as a key.
//
// One thing is particular to the phone's keyboard: its Alt key selects the
// digits and symbols printed above the letters (Alt+Q is '#'). When Alt or
// Shift changed which character a key produced, it has done its work and is
// not passed on as a modifier; a terminal Alt (the ESC prefix programs
// expect) comes from the key bar, or from Alt held with a key that has no
// symbol of its own.
package keys

import (
	"unicode/utf8"

	"github.com/XIAZY/libghostty-vt-go/vt/input"
)

// Flags of a raw Screen key event (<sys/keycodes.h>).
const (
	FlagDown     = 0x00000001 // KEY_DOWN
	FlagRepeat   = 0x00000002 // KEY_REPEAT
	FlagSymValid = 0x00000040 // KEY_SYM_VALID
	FlagCapValid = 0x00000080 // KEY_CAP_VALID
)

// Modifier bits of a raw Screen key event (<sys/keycodes.h>).
const (
	ModShift    = 1 << 0  // KEYMOD_SHIFT
	ModCtrl     = 1 << 1  // KEYMOD_CTRL
	ModAlt      = 1 << 2  // KEYMOD_ALT
	ModAltGr    = 1 << 3  // KEYMOD_ALTGR
	ModCapsLock = 1 << 16 // KEYMOD_CAPS_LOCK
	ModNumLock  = 1 << 17 // KEYMOD_NUM_LOCK
)

// Key codes of <sys/keycodes.h>: keys that are not characters are numbered
// from 0xF000, in Unicode's private use area.
const (
	codeBase       = 0xF000 // KEYCODE_PC_KEYS
	CodeBackspace  = codeBase + 0x08
	CodeTab        = codeBase + 0x09
	CodeReturn     = codeBase + 0x0D
	CodePause      = codeBase + 0x13
	CodeScrollLock = codeBase + 0x14
	CodeEscape     = codeBase + 0x1B
	CodeHome       = codeBase + 0x50
	CodeLeft       = codeBase + 0x51
	CodeUp         = codeBase + 0x52
	CodeRight      = codeBase + 0x53
	CodeDown       = codeBase + 0x54
	CodePgUp       = codeBase + 0x55
	CodePgDown     = codeBase + 0x56
	CodeEnd        = codeBase + 0x57
	CodePrint      = codeBase + 0x61
	CodeInsert     = codeBase + 0x63
	CodeMenu       = codeBase + 0x67
	CodeNumLock    = codeBase + 0x7F
	CodeBackTab    = codeBase + 0x89
	CodeKPEnter    = codeBase + 0x8D
	CodeKPMultiply = codeBase + 0xAA
	CodeKPPlus     = codeBase + 0xAB
	CodeKPMinus    = codeBase + 0xAD
	CodeKPDelete   = codeBase + 0xAE
	CodeKPDivide   = codeBase + 0xAF
	CodeKPInsert   = codeBase + 0xB0
	CodeKPEnd      = codeBase + 0xB1
	CodeKPDown     = codeBase + 0xB2
	CodeKPPgDown   = codeBase + 0xB3
	CodeKPLeft     = codeBase + 0xB4
	CodeKPFive     = codeBase + 0xB5
	CodeKPRight    = codeBase + 0xB6
	CodeKPHome     = codeBase + 0xB7
	CodeKPUp       = codeBase + 0xB8
	CodeKPPgUp     = codeBase + 0xB9
	CodeF1         = codeBase + 0xBE
	CodeF12        = codeBase + 0xC9
	CodeLeftShift  = codeBase + 0xE1
	CodeRightShift = codeBase + 0xE2
	CodeLeftCtrl   = codeBase + 0xE3
	CodeRightCtrl  = codeBase + 0xE4
	CodeCapsLock   = codeBase + 0xE5
	CodeLeftAlt    = codeBase + 0xE9
	CodeRightAlt   = codeBase + 0xEA
	CodeDelete     = codeBase + 0xFF
)

// Raw is a key event from either source, in the terms of <sys/keycodes.h>.
type Raw struct {
	Down   bool // pressed (or repeating); false for a release
	Repeat bool

	// Sym is what the key produced with the modifiers held: a Unicode code
	// point, or a Code* value for a key that is not a character. Zero when
	// unknown.
	Sym rune
	// Cap is the key itself, as if no modifier were held: 'a' for the A
	// key whatever Sym is. Zero when unknown.
	Cap rune
	// Mods is the Mod* bits held.
	Mods int
}

// FromScreen makes a Raw from a raw Screen key event: the values of
// SCREEN_PROPERTY_KEY_FLAGS, _KEY_SYM, _KEY_CAP and _KEY_MODIFIERS.
func FromScreen(flags, sym, cap, mods int) Raw {
	r := Raw{Down: flags&FlagDown != 0, Repeat: flags&FlagRepeat != 0, Mods: mods}
	if flags&FlagSymValid != 0 {
		r.Sym = rune(sym)
	}
	if flags&FlagCapValid != 0 {
		r.Cap = rune(cap)
	}
	return r
}

// FromCascades makes a Raw from a Cascades KeyEvent: its key, keycap and
// unicode properties and its pressed, alt, shift and ctrl flags. Cascades
// reports no repeats; a held key arrives as repeated presses.
func FromCascades(key, keycap int, text string, pressed, alt, shift, ctrl bool) Raw {
	r := Raw{Down: pressed, Sym: rune(key), Cap: rune(keycap)}
	// The text is the better source of the character when there is one
	// (key is documented as "the unicode of the key", which for a
	// character is the same value).
	if c, size := utf8.DecodeRuneInString(text); size > 0 && size == len(text) && c != utf8.RuneError && !isCode(c) && c >= 0x20 && c != 0x7F {
		r.Sym = c
	}
	if shift {
		r.Mods |= ModShift
	}
	if alt {
		r.Mods |= ModAlt
	}
	if ctrl {
		r.Mods |= ModCtrl
	}
	return r
}

// isCode reports whether c is a key code rather than a character.
func isCode(c rune) bool { return c >= 0xE000 && c <= 0xF8FF }

// special maps the keys that are not characters.
var special = map[rune]input.Key{
	CodeBackspace:  input.KeyBackspace,
	CodeTab:        input.KeyTab,
	CodeBackTab:    input.KeyTab,
	CodeReturn:     input.KeyEnter,
	CodeEscape:     input.KeyEscape,
	CodePause:      input.KeyPause,
	CodeScrollLock: input.KeyScrollLock,
	CodePrint:      input.KeyPrintScreen,
	CodeInsert:     input.KeyInsert,
	CodeDelete:     input.KeyDelete,
	CodeHome:       input.KeyHome,
	CodeEnd:        input.KeyEnd,
	CodePgUp:       input.KeyPageUp,
	CodePgDown:     input.KeyPageDown,
	CodeLeft:       input.KeyArrowLeft,
	CodeRight:      input.KeyArrowRight,
	CodeUp:         input.KeyArrowUp,
	CodeDown:       input.KeyArrowDown,
	CodeMenu:       input.KeyContextMenu,
	CodeNumLock:    input.KeyNumLock,
	CodeCapsLock:   input.KeyCapsLock,
	CodeKPEnter:    input.KeyNumpadEnter,
	CodeKPMultiply: input.KeyNumpadMultiply,
	CodeKPPlus:     input.KeyNumpadAdd,
	CodeKPMinus:    input.KeyNumpadSubtract,
	CodeKPDivide:   input.KeyNumpadDivide,
	CodeKPDelete:   input.KeyNumpadDelete,
	CodeKPInsert:   input.KeyNumpadInsert,
	CodeKPEnd:      input.KeyNumpadEnd,
	CodeKPDown:     input.KeyNumpadDown,
	CodeKPPgDown:   input.KeyNumpadPageDown,
	CodeKPLeft:     input.KeyNumpadLeft,
	CodeKPFive:     input.KeyNumpadBegin,
	CodeKPRight:    input.KeyNumpadRight,
	CodeKPHome:     input.KeyNumpadHome,
	CodeKPUp:       input.KeyNumpadUp,
	CodeKPPgUp:     input.KeyNumpadPageUp,
	CodeLeftShift:  input.KeyShiftLeft,
	CodeRightShift: input.KeyShiftRight,
	CodeLeftCtrl:   input.KeyControlLeft,
	CodeRightCtrl:  input.KeyControlRight,
	CodeLeftAlt:    input.KeyAltLeft,
	CodeRightAlt:   input.KeyAltRight,
}

// control maps the control characters some sources deliver in place of a
// key code (Cascades reports Enter as 13, for example).
var control = map[rune]input.Key{
	0x08: input.KeyBackspace,
	0x09: input.KeyTab,
	0x0A: input.KeyEnter,
	0x0D: input.KeyEnter,
	0x1B: input.KeyEscape,
	0x7F: input.KeyBackspace,
}

// punctuation maps the characters of the US layout's unshifted punctuation
// keys to their keys.
var punctuation = map[rune]input.Key{
	' ':  input.KeySpace,
	'-':  input.KeyMinus,
	'=':  input.KeyEqual,
	'[':  input.KeyBracketLeft,
	']':  input.KeyBracketRight,
	'\\': input.KeyBackslash,
	';':  input.KeySemicolon,
	'\'': input.KeyQuote,
	'`':  input.KeyBackquote,
	',':  input.KeyComma,
	'.':  input.KeyPeriod,
	'/':  input.KeySlash,
}

var letters = [26]input.Key{
	input.KeyKeyA, input.KeyKeyB, input.KeyKeyC, input.KeyKeyD, input.KeyKeyE, input.KeyKeyF, input.KeyKeyG,
	input.KeyKeyH, input.KeyKeyI, input.KeyKeyJ, input.KeyKeyK, input.KeyKeyL, input.KeyKeyM, input.KeyKeyN,
	input.KeyKeyO, input.KeyKeyP, input.KeyKeyQ, input.KeyKeyR, input.KeyKeyS, input.KeyKeyT, input.KeyKeyU,
	input.KeyKeyV, input.KeyKeyW, input.KeyKeyX, input.KeyKeyY, input.KeyKeyZ,
}

var digits = [10]input.Key{
	input.KeyDigit0, input.KeyDigit1, input.KeyDigit2, input.KeyDigit3, input.KeyDigit4,
	input.KeyDigit5, input.KeyDigit6, input.KeyDigit7, input.KeyDigit8, input.KeyDigit9,
}

// keyOf is the key a code or an unshifted character belongs to.
func keyOf(c rune) (input.Key, bool) {
	switch {
	case c >= CodeF1 && c <= CodeF12:
		return input.KeyF1 + input.Key(c-CodeF1), true
	case c >= 'a' && c <= 'z':
		return letters[c-'a'], true
	case c >= 'A' && c <= 'Z':
		return letters[c-'A'], true
	case c >= '0' && c <= '9':
		return digits[c-'0'], true
	}
	if k, ok := special[c]; ok {
		return k, true
	}
	if k, ok := control[c]; ok {
		return k, true
	}
	if k, ok := punctuation[c]; ok {
		return k, true
	}
	return input.KeyUnidentified, false
}

// isModifierKey reports whether c is a modifier key by itself.
func isModifierKey(c rune) bool {
	switch c {
	case CodeLeftShift, CodeRightShift, CodeLeftCtrl, CodeRightCtrl, CodeLeftAlt, CodeRightAlt, CodeCapsLock, CodeNumLock:
		return true
	}
	return false
}

// Translator turns Raw events into the library's key events. It holds the
// sticky modifiers of the on-screen key bar. It is not safe for use from
// several goroutines.
type Translator struct {
	stickyCtrl bool
	stickyAlt  bool
}

// StickyCtrl arms or disarms the key bar's Ctrl for the next key.
func (t *Translator) StickyCtrl(on bool) { t.stickyCtrl = on }

// StickyAlt arms or disarms the key bar's Alt for the next key.
func (t *Translator) StickyAlt(on bool) { t.stickyAlt = on }

// Sticky reports which key bar modifiers are armed, for the bar to show.
func (t *Translator) Sticky() (ctrl, alt bool) { return t.stickyCtrl, t.stickyAlt }

// Escape is the key bar's Esc: a press of the Escape key. The armed sticky
// modifiers apply to it and are used up, as for any other key.
func (t *Translator) Escape() input.KeyEvent {
	ev, _ := t.Translate(Raw{Down: true, Sym: CodeEscape, Cap: CodeEscape})
	return ev
}

// Translate turns a raw event into a key event for the library's encoder.
// It reports false for an event that carries no key at all.
//
// The sticky modifiers apply to the first press of a key that is not itself
// a modifier, and are disarmed by it.
func (t *Translator) Translate(r Raw) (input.KeyEvent, bool) {
	sym, cap := r.Sym, r.Cap
	if sym == 0 && cap == 0 {
		return input.KeyEvent{}, false
	}
	if sym == 0 {
		sym = cap
	}

	ev := input.KeyEvent{Action: input.KeyActionRelease}
	if r.Down {
		ev.Action = input.KeyActionPress
		if r.Repeat {
			ev.Action = input.KeyActionRepeat
		}
	}
	ev.Mods = input.Mods{
		Shift:    r.Mods&ModShift != 0,
		Ctrl:     r.Mods&ModCtrl != 0,
		Alt:      r.Mods&(ModAlt|ModAltGr) != 0,
		CapsLock: r.Mods&ModCapsLock != 0,
		NumLock:  r.Mods&ModNumLock != 0,
	}

	// The key: from the code when the event is a key that is not a
	// character, otherwise from the key cap, otherwise from the character.
	character := !isCode(sym) && sym >= 0x20 && sym != 0x7F
	var known bool
	switch {
	case !character:
		ev.Key, known = keyOf(sym)
		if !known && cap != 0 {
			ev.Key, known = keyOf(cap)
		}
	case cap != 0:
		ev.Key, known = keyOf(cap)
		if !known {
			ev.Key, _ = keyOf(sym)
		}
	default:
		ev.Key, _ = keyOf(sym)
	}
	if sym == CodeBackTab {
		ev.Mods.Shift = true
	}

	if character {
		ev.UTF8 = utf8.AppendRune(nil, sym)
		if cap != 0 && !isCode(cap) && cap >= 0x20 {
			ev.UnshiftedCodepoint = uint32(cap)
		}
		// A modifier that changed which character the key produced has
		// done its work: Shift for a capital or a shifted symbol, and on
		// the phone's keyboard Alt for the digit or symbol above a letter.
		// Without a key cap to compare with, Shift is taken to have
		// produced an upper-case letter or a symbol.
		changed := cap != 0 && !isCode(cap) && sym != cap
		if cap == 0 && ev.Mods.Shift && !(sym >= 'a' && sym <= 'z') && !(sym >= '0' && sym <= '9') && sym != ' ' {
			ev.ConsumedMods.Shift = true
		}
		if changed {
			if ev.Mods.Shift {
				ev.ConsumedMods.Shift = true
			}
			if ev.Mods.Alt && !onlyCaseDiffers(sym, cap) {
				// The symbol layer: Alt is not a terminal modifier here.
				ev.Mods.Alt = false
			}
		}
	} else if !known && sym >= 1 && sym <= 26 {
		// A source that already applied Ctrl to a letter and gives no key
		// cap (Ctrl+C arriving as 0x03 alone): it is Ctrl with that
		// letter. The encoder applies Ctrl itself.
		letter := 'a' + sym - 1
		ev.Key, _ = keyOf(letter)
		ev.Mods.Ctrl = true
		ev.UnshiftedCodepoint = uint32(letter)
	}

	// The key bar's sticky modifiers. They are real terminal modifiers and
	// are never consumed by the character.
	if r.Down && !isModifierKey(sym) {
		if t.stickyCtrl {
			ev.Mods.Ctrl = true
		}
		if t.stickyAlt {
			ev.Mods.Alt = true
		}
		if !r.Repeat {
			t.stickyCtrl, t.stickyAlt = false, false
		}
	}
	return ev, true
}

// onlyCaseDiffers reports whether a and b are the same letter in different
// case: Shift's doing, not the symbol layer's.
func onlyCaseDiffers(a, b rune) bool {
	lower := func(c rune) rune {
		if c >= 'A' && c <= 'Z' {
			return c + 'a' - 'A'
		}
		return c
	}
	return a != b && lower(a) == lower(b)
}
