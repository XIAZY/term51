package keys

import "testing"

func up(sym, cap rune, mods int) Raw { return Raw{Sym: sym, Cap: cap, Mods: mods} }

// tap is a press and release of a modifier key with nothing between.
func tap(t *testing.T, tr *Translator, code rune, mod int) {
	t.Helper()
	tr.Translate(down(code, code, mod))
	tr.Translate(up(code, code, 0))
}

func TestTappedAlt(t *testing.T) {
	tm := term(t, "")
	var tr Translator
	tap(t, &tr, CodeLeftAlt, ModAlt)
	if a, _ := tr.Layers(); a != LayerOnce {
		t.Fatalf("after one tap: Alt is %v, want for the next key", a)
	}
	if got := send(t, &tr, tm, down('q', 'q', 0)); got != "#" {
		t.Errorf("tapped Alt then Q: sent %q, want %q", got, "#")
	}
	if got := send(t, &tr, tm, down('q', 'q', 0)); got != "q" {
		t.Errorf("the key after: sent %q, want %q", got, "q")
	}

	// Two taps lock it, a third releases it.
	tap(t, &tr, CodeLeftAlt, ModAlt)
	tap(t, &tr, CodeLeftAlt, ModAlt)
	if a, _ := tr.Layers(); a != LayerLocked {
		t.Fatalf("after two taps: Alt is %v, want locked", a)
	}
	for _, k := range []struct{ key, want rune }{{'w', '1'}, {'e', '2'}, {'m', '.'}} {
		if got := send(t, &tr, tm, down(k.key, k.key, 0)); got != string(k.want) {
			t.Errorf("locked Alt then %c: sent %q, want %q", k.key, got, string(k.want))
		}
	}
	tap(t, &tr, CodeLeftAlt, ModAlt)
	if got := send(t, &tr, tm, down('w', 'w', 0)); got != "w" {
		t.Errorf("after the third tap: sent %q, want %q", got, "w")
	}
}

func TestHeldAltIsNoTap(t *testing.T) {
	tm := term(t, "")
	var tr Translator
	// Held with a key (the phone repeats the press of a held key): the
	// keyboard's symbol, and nothing armed after.
	tr.Translate(down(CodeLeftAlt, CodeLeftAlt, ModAlt))
	tr.Translate(down(CodeLeftAlt, CodeLeftAlt, ModAlt))
	if got := send(t, &tr, tm, down('#', 'q', ModAlt)); got != "#" {
		t.Errorf("Alt held with Q: sent %q, want %q", got, "#")
	}
	tr.Translate(up(CodeLeftAlt, CodeLeftAlt, 0))
	if a, _ := tr.Layers(); a != LayerOff {
		t.Errorf("after Alt held with a key: Alt is %v, want off", a)
	}
	if got := send(t, &tr, tm, down('q', 'q', 0)); got != "q" {
		t.Errorf("then Q: sent %q, want %q", got, "q")
	}
}

func TestTappedAltLearnsTheKeyboard(t *testing.T) {
	tm := term(t, "")
	var tr Translator
	// Another layout: Alt+Q is '!' there, as the keyboard showed once.
	send(t, &tr, tm, down('!', 'q', ModAlt))
	tap(t, &tr, CodeLeftAlt, ModAlt)
	if got := send(t, &tr, tm, down('q', 'q', 0)); got != "!" {
		t.Errorf("tapped Alt then Q after Alt+Q gave '!': sent %q, want %q", got, "!")
	}
}

func TestTappedShift(t *testing.T) {
	tm := term(t, "")
	var tr Translator
	tap(t, &tr, CodeRightShift, ModShift)
	if got := send(t, &tr, tm, down('a', 'a', 0)); got != "A" {
		t.Errorf("tapped Shift then A: sent %q, want %q", got, "A")
	}
	if got := send(t, &tr, tm, down('a', 'a', 0)); got != "a" {
		t.Errorf("the key after: sent %q, want %q", got, "a")
	}
	tap(t, &tr, CodeRightShift, ModShift)
	tap(t, &tr, CodeRightShift, ModShift)
	if _, s := tr.Layers(); s != LayerLocked {
		t.Fatalf("after two taps: Shift is %v, want locked", s)
	}
	if got := send(t, &tr, tm, down('b', 'b', 0)) + send(t, &tr, tm, down('c', 'c', 0)); got != "BC" {
		t.Errorf("locked Shift then B, C: sent %q, want %q", got, "BC")
	}
	// A key that is no character passes and leaves the lock.
	if got := send(t, &tr, tm, down(CodeBackspace, CodeBackspace, 0)); got != "\x7f" {
		t.Errorf("locked Shift then Backspace: sent %q, want %q", got, "\x7f")
	}
	if _, s := tr.Layers(); s != LayerLocked {
		t.Errorf("after Backspace: Shift is %v, want locked", s)
	}
}

func TestTappedAltWithTheQ10sCodes(t *testing.T) {
	tm := term(t, "")
	var tr Translator
	// The Q10's Alt, cut to its low byte and not reported held (FromCascades).
	tr.Translate(FromCascades(0xE9, 0xE9, "é", true, false, false, false))
	tr.Translate(FromCascades(0xE9, 0xE9, "é", false, false, false, false))
	if got := send(t, &tr, tm, FromCascades('q', 'q', "q", true, false, false, false)); got != "#" {
		t.Errorf("the Q10's tapped Alt then Q: sent %q, want %q", got, "#")
	}
}
