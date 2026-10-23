package keys

// The keyboard's own Alt and Shift as the phone's text fields have them: a
// tap of either by itself (pressed and released with no other key between)
// applies it to the next key, a second tap locks it, and a third releases
// it. The phone does this only in its text fields, so the page has to do
// it for the terminal; held down, the keys work as they always did.
//
// A tapped Alt types the symbol printed above the key. The raw event of a
// key pressed after the tap names only the key ('q'), so the symbols come
// from a table: the BlackBerry QWERTY keyboard's, corrected by what the
// keyboard itself delivers whenever Alt is held with a key (on another
// layout, Alt+Q may be something else than '#').

// Layer is the state of a tapped Alt or Shift.
type Layer int

const (
	LayerOff    Layer = iota
	LayerOnce         // for the next key
	LayerLocked       // until tapped again
)

// next is the state after one more tap.
func (l Layer) next() Layer { return (l + 1) % 3 }

// qwertyAlt is the symbol above each letter of the BlackBerry QWERTY
// keyboard (the Q10's, the Classic's, the Bold's).
var qwertyAlt = map[rune]rune{
	'q': '#', 'w': '1', 'e': '2', 'r': '3', 't': '(', 'y': ')', 'u': '_', 'i': '-', 'o': '+', 'p': '@',
	'a': '*', 's': '4', 'd': '5', 'f': '6', 'g': '/', 'h': ':', 'j': ';', 'k': '\'', 'l': '"',
	'z': '7', 'x': '8', 'c': '9', 'v': '?', 'b': '!', 'n': ',', 'm': '.',
}

// tapState follows one modifier key between its press and its release.
type tapState struct {
	down  bool // the key is held
	other bool // another key was pressed while it was held
}

// layers is the part of a Translator that does the tapped Alt and Shift.
type layers struct {
	alt, shift       Layer
	altTap, shiftTap tapState
	learnedAlt       map[rune]rune // cap → symbol, from Alt held with a key
	learnedShift     map[rune]rune // cap → character, from Shift held with a key
	pressedAs        map[rune]rune // cap → what its press typed, for its release
}

// Layers reports the state of the keyboard's tapped Alt and Shift, for the
// page to show.
func (t *Translator) Layers() (alt, shift Layer) { return t.alt, t.shift }

// tapKind is which of the two keys a modifier key is, if either.
func tapKind(c rune) (alt, shift bool) {
	switch c {
	case CodeLeftAlt, CodeRightAlt:
		return true, false
	case CodeLeftShift, CodeRightShift:
		return false, true
	}
	return false, false
}

// layer follows the taps of Alt and Shift and changes the character of a
// key pressed after one: it returns the character the key types. It is
// given every event before it is translated.
func (l *layers) layer(r Raw, sym, cap rune) rune {
	if alt, shift := tapKind(sym); alt || shift {
		tap, state := &l.altTap, &l.alt
		if shift {
			tap, state = &l.shiftTap, &l.shift
		}
		switch {
		case r.Down && !tap.down:
			*tap = tapState{down: true}
		case !r.Down && tap.down:
			if !tap.other {
				*state = state.next()
			}
			*tap = tapState{}
		}
		return sym
	}
	if isModifierKey(sym) {
		return sym
	}
	if r.Down {
		l.altTap.other = l.altTap.other || l.altTap.down
		l.shiftTap.other = l.shiftTap.other || l.shiftTap.down
	}

	// Only a character key has another character on a layer.
	if isCode(sym) || sym < 0x20 || sym == 0x7F {
		if r.Down {
			l.used()
		}
		return sym
	}
	key := cap
	if key == 0 || isCode(key) {
		key = sym
	}
	key = lower(key)

	if !r.Down {
		if c, ok := l.pressedAs[key]; ok {
			delete(l.pressedAs, key)
			return c
		}
		return sym
	}

	// Alt or Shift held: what the keyboard typed is the layer's character
	// for that key, to remember.
	if r.Mods&(ModAlt|ModShift) != 0 {
		if sym != key && !isCode(sym) {
			switch {
			case r.Mods&ModAlt != 0 && !onlyCaseDiffers(sym, key):
				l.learnedAlt = remember(l.learnedAlt, key, sym)
			case r.Mods == ModShift:
				l.learnedShift = remember(l.learnedShift, key, sym)
			}
		}
		l.used()
		return sym
	}

	out := sym
	switch {
	case l.alt != LayerOff:
		if c, ok := l.learnedAlt[key]; ok {
			out = c
		} else if c, ok := qwertyAlt[key]; ok {
			out = c
		}
	case l.shift != LayerOff:
		if c, ok := l.learnedShift[key]; ok {
			out = c
		} else if key >= 'a' && key <= 'z' {
			out = key - 'a' + 'A'
		}
	}
	l.used()
	if out != sym {
		l.pressedAs = remember(l.pressedAs, key, out)
	}
	return out
}

// used ends a layer that was for one key only.
func (l *layers) used() {
	if l.alt == LayerOnce {
		l.alt = LayerOff
	}
	if l.shift == LayerOnce {
		l.shift = LayerOff
	}
}

func remember(m map[rune]rune, k, v rune) map[rune]rune {
	if m == nil {
		m = map[rune]rune{}
	}
	m[k] = v
	return m
}

func lower(c rune) rune {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}
