package terminal

// Translated tests for src/input/key_encode.zig (Ghostty v1.3.1), part 3:
// the Zig tests from "term51: encode: each kitty flag alone selects kitty"
// to the end of the file (tests 137 to 201). Helpers are prefixed ke3.
//
// Zig defaults a Go literal does not give: a KeyEvent's action defaults to
// .press, which is not the Go zero value, so every KeyEvent literal below
// writes `action: InputKeyActionPress` (or the action the Zig literal names).
// All encoder options default to the Go zero value.

import (
	"fmt"
	"testing"
)

// ke3WriterFn is the Zig `comptime f: anytype` taken by term51ABExpect:
// encode, kitty or legacy.
type ke3WriterFn func(writer *oscWriter, event InputKeyKeyEvent, opts InputKeyEncodeOptions) error

// ke3Bools is Zig's `[_]bool{ false, true }`.
var ke3Bools = []bool{false, true}

// ke3ExpectError is Zig's `try testing.expectError(want, got)`.
func ke3ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if got != want {
		t.Fatalf("expected error %v, got %v", want, got)
	}
}

// ke3ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func ke3ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, got none")
		}
		if s, ok := r.(string); ok && s == notImplemented {
			t.Fatalf("expected a panic from the assertion, got: %v", r)
		}
	}()
	f()
}

// ke3Key is Zig's `.{ .key = k }`: a KeyEvent with the default action
// (.press) and nothing else set.
func ke3Key(k InputKeyKey) InputKeyKeyEvent {
	return InputKeyKeyEvent{Action: InputKeyActionPress, Key: k}
}

// ke3IndexOfKey is std.mem.indexOfScalar(key.Key, keys, k) != null.
func ke3IndexOfKey(keys []InputKeyKey, k InputKeyKey) bool {
	for _, c := range keys {
		if c == k {
			return true
		}
	}
	return false
}

// ke3Pc calls pcStyleFunctionKey and unwraps the result (Zig's `.?`).
func ke3Pc(t *testing.T, keyval InputKeyKey, mods InputKeyMods, cursor bool, kpReq bool, ignore bool, mok bool) []uint8 {
	t.Helper()
	return zigUnwrap(t, inputKeyEncodePcStyleFunctionKey(keyval, mods, cursor, kpReq, ignore, mok))
}

// ke3PcNull is `try testing.expect(pcStyleFunctionKey(...) == null)`.
func ke3PcNull(t *testing.T, keyval InputKeyKey, mods InputKeyMods, cursor bool, kpReq bool, ignore bool, mok bool) {
	t.Helper()
	expect(t, !inputKeyEncodePcStyleFunctionKey(keyval, mods, cursor, kpReq, ignore, mok).Valid, "pcStyleFunctionKey(...) == null")
}

// ke3Lap is `try legacyAltPrefix(...)`.
func ke3Lap(t *testing.T, event InputKeyKeyEvent, bindingMods InputKeyMods, mods InputKeyMods, opts InputKeyEncodeOptions) Optional[uint8] {
	t.Helper()
	r, err := inputKeyEncodeLegacyAltPrefix(event, bindingMods, mods, opts)
	zigTry(t, err)
	return r
}

// term51AB helpers.

// ke3ABExpect is term51ABExpect: runs `f` into a roomy buffer and checks the
// exact output, then re-runs it with every smaller capacity (0..len-1) and
// expects error.WriteFailed, so that every write point of the encoder is
// checked for a too-small buffer.
func ke3ABExpect(t *testing.T, f ke3WriterFn, expected string, event InputKeyKeyEvent, opts InputKeyEncodeOptions) {
	t.Helper()
	var big [512]uint8
	w := oscWriterFixed(big[:])
	zigTry(t, f(&w, event, opts))
	expectEqualStrings(t, expected, string(w.buffered()))

	for capacity := 0; capacity <= len(expected); capacity++ {
		var small [512]uint8
		w2 := oscWriterFixed(small[0:capacity:capacity])
		if capacity < len(expected) {
			ke3ExpectError(t, ErrWriteFailed, f(&w2, event, opts))
			// Go-only check: a fixed writer keeps what fit before the failure,
			// which is the start of the expected output.
			expectEqualStrings(t, expected[0:capacity], string(w2.buffered()))
		} else {
			zigTry(t, f(&w2, event, opts))
			expectEqualStrings(t, expected, string(w2.buffered()))
		}
	}
}

// ke3ABCase is term51ABCase.
type ke3ABCase struct {
	ev   InputKeyKeyEvent
	opts InputKeyEncodeOptions
	want string
}

// ke3ABRunCases is term51ABRunCases.
func ke3ABRunCases(t *testing.T, f ke3WriterFn, cases []ke3ABCase) {
	t.Helper()
	for i, c := range cases {
		ok := false
		func() {
			defer func() {
				if !ok {
					t.Logf("term51AB case index %d failed", i)
				}
			}()
			ke3ABExpect(t, f, c.want, c.ev, c.opts)
			ok = true
		}()
	}
}

var ke3ABDis = InputKeyEncodeOptions{KittyFlags: KittyKeyFlags{Disambiguate: true}}
var ke3ABEvents = InputKeyEncodeOptions{KittyFlags: KittyKeyFlags{ReportEvents: true}}
var ke3ABDisAll = InputKeyEncodeOptions{KittyFlags: KittyKeyFlags{Disambiguate: true, ReportAll: true}}
var ke3ABFull = InputKeyEncodeOptions{KittyFlags: KittyKeyFlagsTrue}

// term51AC helpers.

// ke3ACMods is term51ACMods: mods from a 4 bit mask: bit 0 shift, bit 1
// ctrl, bit 2 alt, bit 3 super.
func ke3ACMods(bits int) InputKeyMods {
	return InputKeyMods{
		Shift: (bits & 1) != 0,
		Ctrl:  (bits & 2) != 0,
		Alt:   (bits & 4) != 0,
		Super: (bits & 8) != 0,
	}
}

// ke3ACCodes is term51ACCodes: the xterm modifier parameter
// (1 + shift + 2*alt + 4*ctrl + 8*super) for each mask accepted by ke3ACMods.
var ke3ACCodes = [16]uint8{1, 2, 5, 6, 3, 4, 7, 8, 9, 10, 13, 14, 11, 12, 15, 16}

// ke3ACRun is term51ACRun: runs the key event through `legacy` and through
// the public `encode` (kitty flags disabled), asserts both produce the same
// bytes and returns them (backed by buf).
func ke3ACRun(t *testing.T, buf []uint8, event InputKeyKeyEvent, opts InputKeyEncodeOptions) []uint8 {
	t.Helper()
	w := oscWriterFixed(buf)
	zigTry(t, inputKeyEncodeLegacy(&w, event, opts))
	var buf2 [128]uint8
	w2 := oscWriterFixed(buf2[:])
	zigTry(t, inputKeyEncodeEncode(&w2, event, opts))
	expectEqualStrings(t, string(w.buffered()), string(w2.buffered()))
	return w.buffered()
}

// ke3ACExpect is term51ACExpect.
func ke3ACExpect(t *testing.T, expected string, event InputKeyKeyEvent, opts InputKeyEncodeOptions) {
	t.Helper()
	var buf [128]uint8
	got := ke3ACRun(t, buf[:], event, opts)
	expectEqualStrings(t, expected, string(got))
}

// ke3ACExpectFit is term51ACExpectFit: asserts the event encodes to exactly
// `expected` and that a fixed writer of exactly that size succeeds while
// every smaller one (including zero length) fails with WriteFailed, for both
// entry points.
func ke3ACExpectFit(t *testing.T, expected string, event InputKeyKeyEvent, opts InputKeyEncodeOptions) {
	t.Helper()
	var buf [64]uint8
	{
		w := oscWriterFixed(buf[0:len(expected):len(expected)])
		zigTry(t, inputKeyEncodeLegacy(&w, event, opts))
		expectEqualStrings(t, expected, string(w.buffered()))
	}
	{
		w := oscWriterFixed(buf[0:len(expected):len(expected)])
		zigTry(t, inputKeyEncodeEncode(&w, event, opts))
		expectEqualStrings(t, expected, string(w.buffered()))
	}
	for n := 0; n < len(expected); n++ {
		w := oscWriterFixed(buf[0:n:n])
		ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeLegacy(&w, event, opts))
		w2 := oscWriterFixed(buf[0:n:n])
		ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeEncode(&w2, event, opts))
	}
}

// ke3ACPcCase is term51ACPcCase.
type ke3ACPcCase struct {
	key         InputKeyKey
	normal      string
	application string
	prefix      string
	suffix      string
}

// ke3ACPcCases is term51ACPcCases: keys whose entries are: pcStyle (15
// modifier sequences) followed by the plain sequence(s).
var ke3ACPcCases = []ke3ACPcCase{
	{key: InputKeyKeyArrowUp, normal: "\x1b[A", application: "\x1bOA", prefix: "\x1b[1;", suffix: "A"},
	{key: InputKeyKeyArrowDown, normal: "\x1b[B", application: "\x1bOB", prefix: "\x1b[1;", suffix: "B"},
	{key: InputKeyKeyArrowRight, normal: "\x1b[C", application: "\x1bOC", prefix: "\x1b[1;", suffix: "C"},
	{key: InputKeyKeyArrowLeft, normal: "\x1b[D", application: "\x1bOD", prefix: "\x1b[1;", suffix: "D"},
	{key: InputKeyKeyHome, normal: "\x1b[H", application: "\x1bOH", prefix: "\x1b[1;", suffix: "H"},
	{key: InputKeyKeyEnd, normal: "\x1b[F", application: "\x1bOF", prefix: "\x1b[1;", suffix: "F"},
	{key: InputKeyKeyNumpadUp, normal: "\x1b[A", application: "\x1bOA", prefix: "\x1b[1;", suffix: "A"},
	{key: InputKeyKeyNumpadDown, normal: "\x1b[B", application: "\x1bOB", prefix: "\x1b[1;", suffix: "B"},
	{key: InputKeyKeyNumpadRight, normal: "\x1b[C", application: "\x1bOC", prefix: "\x1b[1;", suffix: "C"},
	{key: InputKeyKeyNumpadLeft, normal: "\x1b[D", application: "\x1bOD", prefix: "\x1b[1;", suffix: "D"},
	{key: InputKeyKeyNumpadBegin, normal: "\x1b[E", application: "\x1bOE", prefix: "\x1b[1;", suffix: "E"},
	{key: InputKeyKeyNumpadHome, normal: "\x1b[H", application: "\x1bOH", prefix: "\x1b[1;", suffix: "H"},
	{key: InputKeyKeyNumpadEnd, normal: "\x1b[F", application: "\x1bOF", prefix: "\x1b[1;", suffix: "F"},
	{key: InputKeyKeyInsert, normal: "\x1b[2~", application: "\x1b[2~", prefix: "\x1b[2;", suffix: "~"},
	{key: InputKeyKeyDelete, normal: "\x1b[3~", application: "\x1b[3~", prefix: "\x1b[3;", suffix: "~"},
	{key: InputKeyKeyPageUp, normal: "\x1b[5~", application: "\x1b[5~", prefix: "\x1b[5;", suffix: "~"},
	{key: InputKeyKeyPageDown, normal: "\x1b[6~", application: "\x1b[6~", prefix: "\x1b[6;", suffix: "~"},
	{key: InputKeyKeyNumpadInsert, normal: "\x1b[2~", application: "\x1b[2~", prefix: "\x1b[2;", suffix: "~"},
	{key: InputKeyKeyNumpadDelete, normal: "\x1b[3~", application: "\x1b[3~", prefix: "\x1b[3;", suffix: "~"},
	{key: InputKeyKeyNumpadPageUp, normal: "\x1b[5~", application: "\x1b[5~", prefix: "\x1b[5;", suffix: "~"},
	{key: InputKeyKeyNumpadPageDown, normal: "\x1b[6~", application: "\x1b[6~", prefix: "\x1b[6;", suffix: "~"},
	{key: InputKeyKeyF1, normal: "\x1bOP", application: "\x1bOP", prefix: "\x1b[1;", suffix: "P"},
	{key: InputKeyKeyF2, normal: "\x1bOQ", application: "\x1bOQ", prefix: "\x1b[1;", suffix: "Q"},
	{key: InputKeyKeyF3, normal: "\x1bOR", application: "\x1bOR", prefix: "\x1b[13;", suffix: "~"},
	{key: InputKeyKeyF4, normal: "\x1bOS", application: "\x1bOS", prefix: "\x1b[1;", suffix: "S"},
	{key: InputKeyKeyF5, normal: "\x1b[15~", application: "\x1b[15~", prefix: "\x1b[15;", suffix: "~"},
	{key: InputKeyKeyF6, normal: "\x1b[17~", application: "\x1b[17~", prefix: "\x1b[17;", suffix: "~"},
	{key: InputKeyKeyF7, normal: "\x1b[18~", application: "\x1b[18~", prefix: "\x1b[18;", suffix: "~"},
	{key: InputKeyKeyF8, normal: "\x1b[19~", application: "\x1b[19~", prefix: "\x1b[19;", suffix: "~"},
	{key: InputKeyKeyF9, normal: "\x1b[20~", application: "\x1b[20~", prefix: "\x1b[20;", suffix: "~"},
	{key: InputKeyKeyF10, normal: "\x1b[21~", application: "\x1b[21~", prefix: "\x1b[21;", suffix: "~"},
	{key: InputKeyKeyF11, normal: "\x1b[23~", application: "\x1b[23~", prefix: "\x1b[23;", suffix: "~"},
	{key: InputKeyKeyF12, normal: "\x1b[24~", application: "\x1b[24~", prefix: "\x1b[24;", suffix: "~"},
}

// ke3ACKpCase is term51ACKpCase.
type ke3ACKpCase struct {
	key    InputKeyKey
	suffix string
}

// ke3ACKpCases is term51ACKpCases.
var ke3ACKpCases = []ke3ACKpCase{
	{key: InputKeyKeyNumpad0, suffix: "p"},
	{key: InputKeyKeyNumpad1, suffix: "q"},
	{key: InputKeyKeyNumpad2, suffix: "r"},
	{key: InputKeyKeyNumpad3, suffix: "s"},
	{key: InputKeyKeyNumpad4, suffix: "t"},
	{key: InputKeyKeyNumpad5, suffix: "u"},
	{key: InputKeyKeyNumpad6, suffix: "v"},
	{key: InputKeyKeyNumpad7, suffix: "w"},
	{key: InputKeyKeyNumpad8, suffix: "x"},
	{key: InputKeyKeyNumpad9, suffix: "y"},
	{key: InputKeyKeyNumpadDecimal, suffix: "n"},
	{key: InputKeyKeyNumpadDivide, suffix: "o"},
	{key: InputKeyKeyNumpadMultiply, suffix: "j"},
	{key: InputKeyKeyNumpadSubtract, suffix: "m"},
	{key: InputKeyKeyNumpadAdd, suffix: "k"},
	{key: InputKeyKeyNumpadEnter, suffix: "M"},
}

// ke3ACKeysWithEntries is term51ACKeysWithEntries: every key that has at
// least one entry in function_keys.keys.
var ke3ACKeysWithEntries = []InputKeyKey{
	InputKeyKeyArrowUp, InputKeyKeyArrowDown, InputKeyKeyArrowRight, InputKeyKeyArrowLeft, InputKeyKeyHome, InputKeyKeyEnd,
	InputKeyKeyInsert, InputKeyKeyDelete, InputKeyKeyPageUp, InputKeyKeyPageDown, InputKeyKeyF1, InputKeyKeyF2,
	InputKeyKeyF3, InputKeyKeyF4, InputKeyKeyF5, InputKeyKeyF6, InputKeyKeyF7, InputKeyKeyF8,
	InputKeyKeyF9, InputKeyKeyF10, InputKeyKeyF11, InputKeyKeyF12, InputKeyKeyNumpad0, InputKeyKeyNumpad1,
	InputKeyKeyNumpad2, InputKeyKeyNumpad3, InputKeyKeyNumpad4, InputKeyKeyNumpad5, InputKeyKeyNumpad6, InputKeyKeyNumpad7,
	InputKeyKeyNumpad8, InputKeyKeyNumpad9, InputKeyKeyNumpadDecimal, InputKeyKeyNumpadDivide, InputKeyKeyNumpadMultiply, InputKeyKeyNumpadSubtract,
	InputKeyKeyNumpadAdd, InputKeyKeyNumpadEnter, InputKeyKeyNumpadUp, InputKeyKeyNumpadDown, InputKeyKeyNumpadRight, InputKeyKeyNumpadLeft,
	InputKeyKeyNumpadBegin, InputKeyKeyNumpadHome, InputKeyKeyNumpadEnd, InputKeyKeyNumpadInsert, InputKeyKeyNumpadDelete, InputKeyKeyNumpadPageUp,
	InputKeyKeyNumpadPageDown, InputKeyKeyBackspace, InputKeyKeyTab, InputKeyKeyEnter, InputKeyKeyEscape,
}

// Expected outputs, indexed by the ke3ACMods bit mask, for the keys with
// explicit per modifier entries. "Off" is modify_other_keys_state_2 = false.
var ke3ACBackspaceOff = [16]string{
	"\x7f",
	"\x7f",
	"\x08",
	"\x08",
	"\x1b\x7f",
	"\x1b\x7f",
	"\x1b\x08",
	"\x7f",
	"\x7f",
	"\x7f",
	"\x08",
	"\x08",
	"\x1b\x7f",
	"\x1b\x7f",
	"\x1b\x08",
	"\x1b\x08",
}
var ke3ACBackspaceOn = [16]string{
	"\x7f",
	"\x1b[27;2;127~",
	"\x08",
	"\x1b[27;6;127~",
	"\x1b[27;3;127~",
	"\x1b[27;4;127~",
	"\x1b[27;7;127~",
	"\x1b[27;8;127~",
	"\x1b[27;9;127~",
	"\x1b[27;10;127~",
	"\x1b[27;13;127~",
	"\x1b[27;14;127~",
	"\x1b[27;11;127~",
	"\x1b[27;12;127~",
	"\x1b[27;15;127~",
	"\x1b[27;16;127~",
}
var ke3ACTabOff = [16]string{
	"\t",
	"\x1b[Z",
	"\x1b[27;5;9~",
	"\x1b[27;6;9~",
	"\x1b\t",
	"\x1b[27;4;9~",
	"\x1b[27;7;9~",
	"\x1b[27;8;9~",
	"\x1b[27;9;9~",
	"\x1b[27;10;9~",
	"\x1b[27;13;9~",
	"\x1b[27;14;9~",
	"\x1b[27;11;9~",
	"\x1b[27;12;9~",
	"\x1b[27;15;9~",
	"\x1b[27;16;9~",
}
var ke3ACTabOn = [16]string{
	"\t",
	"\x1b[27;2;9~",
	"\x1b[27;5;9~",
	"\x1b[27;6;9~",
	"\x1b[27;3;9~",
	"\x1b[27;4;9~",
	"\x1b[27;7;9~",
	"\x1b[27;8;9~",
	"\x1b[27;9;9~",
	"\x1b[27;10;9~",
	"\x1b[27;13;9~",
	"\x1b[27;14;9~",
	"\x1b[27;11;9~",
	"\x1b[27;12;9~",
	"\x1b[27;15;9~",
	"\x1b[27;16;9~",
}
var ke3ACEnterOff = [16]string{
	"\r",
	"\x1b[27;2;13~",
	"\x1b[27;5;13~",
	"\x1b[27;6;13~",
	"\x1b\r",
	"\x1b[27;4;13~",
	"\x1b[27;7;13~",
	"\x1b[27;8;13~",
	"\x1b[27;9;13~",
	"\x1b[27;10;13~",
	"\x1b[27;13;13~",
	"\x1b[27;14;13~",
	"\x1b[27;11;13~",
	"\x1b[27;12;13~",
	"\x1b[27;15;13~",
	"\x1b[27;16;13~",
}
var ke3ACEnterOn = [16]string{
	"\r",
	"\x1b[27;2;13~",
	"\x1b[27;5;13~",
	"\x1b[27;6;13~",
	"\x1b[27;3;13~",
	"\x1b[27;4;13~",
	"\x1b[27;7;13~",
	"\x1b[27;8;13~",
	"\x1b[27;9;13~",
	"\x1b[27;10;13~",
	"\x1b[27;13;13~",
	"\x1b[27;14;13~",
	"\x1b[27;11;13~",
	"\x1b[27;12;13~",
	"\x1b[27;15;13~",
	"\x1b[27;16;13~",
}
var ke3ACEscape = [16]string{
	"\x1b",
	"\x1b[27;2;27~",
	"\x1b[27;5;27~",
	"\x1b[27;6;27~",
	"\x1b\x1b",
	"\x1b[27;4;27~",
	"\x1b[27;7;27~",
	"\x1b[27;8;27~",
	"\x1b[27;9;27~",
	"\x1b[27;10;27~",
	"\x1b[27;13;27~",
	"\x1b[27;14;27~",
	"\x1b[27;11;27~",
	"\x1b[27;12;27~",
	"\x1b[27;15;27~",
	"\x1b[27;16;27~",
}

// ke3ACCtrlPair is the anonymous tuple `struct { u8, u8 }`.
type ke3ACCtrlPair struct {
	x0 uint8
	x1 uint8
}

// ke3ACCtrlPairs is term51ACCtrlPairs: all single bytes that ctrlSeq maps
// (with exactly ctrl held) and the resulting C0 byte. Every other byte maps
// to null.
var ke3ACCtrlPairs = []ke3ACCtrlPair{
	{' ', 0},
	{'/', 31},
	{'0', 48},
	{'1', 49},
	{'2', 0},
	{'3', 27},
	{'4', 28},
	{'5', 29},
	{'6', 30},
	{'7', 31},
	{'8', 127},
	{'9', 57},
	{'?', 127},
	{'@', 0},
	{'\\', 28},
	{']', 29},
	{'^', 30},
	{'_', 31},
	{'a', 1},
	{'b', 2},
	{'c', 3},
	{'d', 4},
	{'e', 5},
	{'f', 6},
	{'g', 7},
	{'h', 8},
	{'j', 10},
	{'k', 11},
	{'l', 12},
	{'n', 14},
	{'o', 15},
	{'p', 16},
	{'q', 17},
	{'r', 18},
	{'s', 19},
	{'t', 20},
	{'u', 21},
	{'v', 22},
	{'w', 23},
	{'x', 24},
	{'y', 25},
	{'z', 26},
	{'~', 30},
}

// ke3ACCtrlExpected is term51ACCtrlExpected.
func ke3ACCtrlExpected(b uint8) Optional[uint8] {
	for _, p := range ke3ACCtrlPairs {
		if p.x0 == b {
			return Some(p.x1)
		}
	}
	return None[uint8]()
}

// ke3ACCtrlKeyPair is the anonymous tuple `struct { key.Key, u8 }`.
type ke3ACCtrlKeyPair struct {
	x0 InputKeyKey
	x1 uint8
}

// ke3ACCtrlKeyPairs is term51ACCtrlKeyPairs: keys (with no utf8 text) that
// ctrlSeq maps through key.Key.codepoint.
var ke3ACCtrlKeyPairs = []ke3ACCtrlKeyPair{
	{InputKeyKeyKeyA, 1},
	{InputKeyKeyKeyB, 2},
	{InputKeyKeyKeyC, 3},
	{InputKeyKeyKeyD, 4},
	{InputKeyKeyKeyE, 5},
	{InputKeyKeyKeyF, 6},
	{InputKeyKeyKeyG, 7},
	{InputKeyKeyKeyH, 8},
	{InputKeyKeyKeyJ, 10},
	{InputKeyKeyKeyK, 11},
	{InputKeyKeyKeyL, 12},
	{InputKeyKeyKeyN, 14},
	{InputKeyKeyKeyO, 15},
	{InputKeyKeyKeyP, 16},
	{InputKeyKeyKeyQ, 17},
	{InputKeyKeyKeyR, 18},
	{InputKeyKeyKeyS, 19},
	{InputKeyKeyKeyT, 20},
	{InputKeyKeyKeyU, 21},
	{InputKeyKeyKeyV, 22},
	{InputKeyKeyKeyW, 23},
	{InputKeyKeyKeyX, 24},
	{InputKeyKeyKeyY, 25},
	{InputKeyKeyKeyZ, 26},
	{InputKeyKeyDigit0, 48},
	{InputKeyKeyDigit1, 49},
	{InputKeyKeyDigit2, 0},
	{InputKeyKeyDigit3, 27},
	{InputKeyKeyDigit4, 28},
	{InputKeyKeyDigit5, 29},
	{InputKeyKeyDigit6, 30},
	{InputKeyKeyDigit7, 31},
	{InputKeyKeyDigit8, 127},
	{InputKeyKeyDigit9, 57},
	{InputKeyKeyNumpad0, 48},
	{InputKeyKeyNumpad1, 49},
	{InputKeyKeyNumpad2, 0},
	{InputKeyKeyNumpad3, 27},
	{InputKeyKeyNumpad4, 28},
	{InputKeyKeyNumpad5, 29},
	{InputKeyKeyNumpad6, 30},
	{InputKeyKeyNumpad7, 31},
	{InputKeyKeyNumpad8, 127},
	{InputKeyKeyNumpad9, 57},
	{InputKeyKeySpace, 0},
	{InputKeyKeySlash, 31},
	{InputKeyKeyNumpadDivide, 31},
	{InputKeyKeyBackslash, 28},
	{InputKeyKeyBracketRight, 29},
}

// ke3ACCtrlKeyExpected is term51ACCtrlKeyExpected.
func ke3ACCtrlKeyExpected(k InputKeyKey) Optional[uint8] {
	for _, p := range ke3ACCtrlKeyPairs {
		if p.x0 == k {
			return Some(p.x1)
		}
	}
	return None[uint8]()
}

// ke3ACCase is term51ACCase (opts defaults to .{}: every case below writes it).
type ke3ACCase struct {
	ev   InputKeyKeyEvent
	opts InputKeyEncodeOptions
	want string
}

// ke3ACRunCases is term51ACRunCases.
func ke3ACRunCases(t *testing.T, cases []ke3ACCase) {
	t.Helper()
	for i, c := range cases {
		ok := false
		func() {
			defer func() {
				if !ok {
					t.Logf("term51AC: case %d failed", i)
				}
			}()
			ke3ACExpect(t, c.want, c.ev, c.opts)
			ok = true
		}()
	}
}

var ke3ACAltOpts = InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: InputConfigOptionAsAltTrue}
var ke3ACAltNoPrefix = InputKeyEncodeOptions{AltEscPrefix: false, MacosOptionAsAlt: InputConfigOptionAsAltTrue}
var ke3ACMokOpts = InputKeyEncodeOptions{ModifyOtherKeysState2: true, MacosOptionAsAlt: InputConfigOptionAsAltTrue}
var ke3ACMokAltOpts = InputKeyEncodeOptions{
	ModifyOtherKeysState2: true,
	AltEscPrefix:          true,
	MacosOptionAsAlt:      InputConfigOptionAsAltTrue,
}

// Zig: const term51ACMac = builtin.os.tag == .macos;
// Zig: const term51ACDarwin = builtin.target.os.tag.isDarwin();
// These are comptime constants. In Go they are the platform a test selects
// with testPlatform: term51ACMac is platformGOOS == "darwin" (ke3ACMac) and
// term51ACDarwin is "darwin" or "ios" (ke3ACDarwin). The tests that use
// them run once per platform in ke3Platforms.

// ke3Platforms is the three platform modes of the encoders.
var ke3Platforms = []string{"linux", "darwin", "ios"}

// ke3ACMac is term51ACMac for the platform goos.
func ke3ACMac(goos string) bool { return goos == "darwin" }

// ke3ACDarwin is term51ACDarwin for the platform goos.
func ke3ACDarwin(goos string) bool { return goos == "darwin" || goos == "ios" }

// Zig: test "term51: encode: each kitty flag alone selects kitty"
func TestKe3_term51_encode_each_kitty_flag_alone_selects_kitty(t *testing.T) {
	testPlatform(t, "linux")
	flags := []KittyKeyFlags{
		{Disambiguate: true},
		{ReportEvents: true},
		{ReportAlternates: true},
		{ReportAll: true},
		{ReportAssociated: true},
		KittyKeyFlagsTrue,
	}
	for _, f := range flags {
		ke3ABExpect(t, inputKeyEncodeEncode, "\x1b[27u", ke3Key(InputKeyKeyEscape), InputKeyEncodeOptions{KittyFlags: f})
	}
	ke3ABExpect(t, inputKeyEncodeEncode, "\x1b", ke3Key(InputKeyKeyEscape), InputKeyEncodeOptions{KittyFlags: KittyKeyFlagsDisabled})
	ke3ABExpect(t, inputKeyEncodeEncode, "\x1b", ke3Key(InputKeyKeyEscape), InputKeyEncodeOptionsDefault)
}

// Zig: test "term51: encode: matches kitty and legacy exactly"
func TestKe3_term51_encode_matches_kitty_and_legacy_exactly(t *testing.T) {
	testPlatform(t, "linux")
	events := []InputKeyKeyEvent{
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a"), UnshiftedCodepoint: 'a'},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Shift: true}},
		InputKeyKeyEvent{Key: InputKeyKeyF5, Action: InputKeyActionRepeat},
		InputKeyKeyEvent{Key: InputKeyKeyKeyA, Action: InputKeyActionRelease, UTF8: []uint8("a")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Composing: true, UTF8: []uint8("a")},
	}
	for _, ev := range events {
		var b1 [64]uint8
		var b2 [64]uint8
		w1 := oscWriterFixed(b1[:])
		w2 := oscWriterFixed(b2[:])
		zigTry(t, inputKeyEncodeEncode(&w1, ev, ke3ABFull))
		zigTry(t, inputKeyEncodeKitty(&w2, ev, ke3ABFull))
		expectEqualStrings(t, string(w2.buffered()), string(w1.buffered()))

		w3 := oscWriterFixed(b1[:])
		w4 := oscWriterFixed(b2[:])
		zigTry(t, inputKeyEncodeEncode(&w3, ev, InputKeyEncodeOptions{}))
		zigTry(t, inputKeyEncodeLegacy(&w4, ev, InputKeyEncodeOptions{}))
		expectEqualStrings(t, string(w4.buffered()), string(w3.buffered()))
	}
}

// Zig: test "term51: encode: legacy options"
func TestKe3_term51_encode_legacy_options(t *testing.T) {
	testPlatform(t, "linux")
	ke3ABRunCases(t, inputKeyEncodeEncode, []ke3ABCase{
		// cursor_key_application
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp}, opts: InputKeyEncodeOptions{}, want: "\x1b[A"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp}, opts: InputKeyEncodeOptions{CursorKeyApplication: true}, want: "\x1bOA"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyHome}, opts: InputKeyEncodeOptions{CursorKeyApplication: true}, want: "\x1bOH"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp, Mods: InputKeyMods{Shift: true}}, opts: InputKeyEncodeOptions{CursorKeyApplication: true}, want: "\x1b[1;2A"},
		// keypad_key_application and ignore_keypad_with_numlock
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{}, want: "1"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOq"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true, IgnoreKeypadWithNumlock: true}, want: "1"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1"), Mods: InputKeyMods{NumLock: true}}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOq"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEnter}, opts: InputKeyEncodeOptions{}, want: "\r"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEnter}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOM"},
		// alt_esc_prefix
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, UTF8: []uint8("c"), Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'c'}, opts: InputKeyEncodeOptions{MacosOptionAsAlt: InputConfigOptionAsAltTrue}, want: "c"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, UTF8: []uint8("c"), Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'c'}, opts: InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: InputConfigOptionAsAltTrue}, want: "\x1bc"},
		// modify_other_keys_state_2
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit8, UTF8: []uint8("8"), Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: '8'}, opts: InputKeyEncodeOptions{ModifyOtherKeysState2: true, MacosOptionAsAlt: InputConfigOptionAsAltTrue}, want: "\x1b[27;3;56~"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit8, UTF8: []uint8("8"), Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: '8'}, opts: InputKeyEncodeOptions{MacosOptionAsAlt: InputConfigOptionAsAltTrue}, want: "8"},
		// ... and none of them matter once kitty is on.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp}, opts: InputKeyEncodeOptions{KittyFlags: KittyKeyFlags{Disambiguate: true}, CursorKeyApplication: true}, want: "\x1b[A"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{KittyFlags: KittyKeyFlags{Disambiguate: true, ReportAll: true}, KeypadKeyApplication: true}, want: "\x1b[57400u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, UTF8: []uint8("c"), Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'c'}, opts: InputKeyEncodeOptions{KittyFlags: KittyKeyFlags{Disambiguate: true}, AltEscPrefix: true, MacosOptionAsAlt: InputConfigOptionAsAltTrue}, want: "\x1b[99;3u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit8, UTF8: []uint8("8"), Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: '8'}, opts: InputKeyEncodeOptions{KittyFlags: KittyKeyFlags{Disambiguate: true}, ModifyOtherKeysState2: true}, want: "\x1b[56;3u"},
	})
}

// Zig: test "term51: encode: macos_option_as_alt with every value"
func TestKe3_term51_encode_macos_option_as_alt_with_every_value(t *testing.T) {
	// Go-only: platform modes
	for _, goos := range ke3Platforms {
		t.Run(goos, func(t *testing.T) {
			testPlatform(t, goos)
			ev := InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, UTF8: []uint8("c"), Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'c'}
			evr := InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, UTF8: []uint8("c"), Mods: InputKeyMods{Alt: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideRight}}, UnshiftedCodepoint: 'c'}
			for _, oa := range []InputConfigOptionAsAlt{InputConfigOptionAsAltFalse, InputConfigOptionAsAltTrue, InputConfigOptionAsAltLeft, InputConfigOptionAsAltRight} {
				opts := InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: oa}
				// Left alt (default side).
				leftWant := "\x1bc"
				if ke3ACMac(goos) && (oa == InputConfigOptionAsAltFalse || oa == InputConfigOptionAsAltRight) {
					leftWant = "c"
				}
				ke3ABExpect(t, inputKeyEncodeEncode, leftWant, ev, opts)
				// Right alt.
				rightWant := "\x1bc"
				if ke3ACMac(goos) && (oa == InputConfigOptionAsAltFalse || oa == InputConfigOptionAsAltLeft) {
					rightWant = "c"
				}
				ke3ABExpect(t, inputKeyEncodeEncode, rightWant, evr, opts)
			}
		})
	}
}

// Zig: test "term51: encode: actions and composing"
func TestKe3_term51_encode_actions_and_composing(t *testing.T) {
	testPlatform(t, "linux")
	ke3ABRunCases(t, inputKeyEncodeEncode, []ke3ABCase{
		// Legacy only encodes press and repeat and never while composing.
		{ev: InputKeyKeyEvent{Key: InputKeyKeyEscape, Action: InputKeyActionRelease}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Key: InputKeyKeyEscape, Action: InputKeyActionRepeat}, opts: InputKeyEncodeOptions{}, want: "\x1b"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, Composing: true}, opts: InputKeyEncodeOptions{}, want: ""},
		// Kitty: release needs report_events.
		{ev: InputKeyKeyEvent{Key: InputKeyKeyEscape, Action: InputKeyActionRelease}, opts: ke3ABDis, want: ""},
		{ev: InputKeyKeyEvent{Key: InputKeyKeyEscape, Action: InputKeyActionRelease}, opts: ke3ABEvents, want: "\x1b[27;1:3u"},
		{ev: InputKeyKeyEvent{Key: InputKeyKeyEscape, Action: InputKeyActionRepeat}, opts: ke3ABEvents, want: "\x1b[27;1:2u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, Composing: true}, opts: ke3ABDis, want: ""},
	})
}

// Zig: test "term51: encode: write errors are propagated"
func TestKe3_term51_encode_write_errors_are_propagated(t *testing.T) {
	testPlatform(t, "linux")
	var empty [0]uint8
	w := oscWriterFixed(empty[:])
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeEncode(&w, ke3Key(InputKeyKeyEscape), InputKeyEncodeOptions{}))
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeEncode(&w, ke3Key(InputKeyKeyEscape), ke3ABDis))
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeEncode(&w, ke3Key(InputKeyKeyArrowUp), InputKeyEncodeOptions{}))
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeEncode(&w, ke3Key(InputKeyKeyArrowUp), ke3ABFull))
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeEncode(&w, InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a")}, InputKeyEncodeOptions{}))
	// No output means no error.
	zigTry(t, inputKeyEncodeEncode(&w, InputKeyKeyEvent{Key: InputKeyKeyEscape, Action: InputKeyActionRelease}, InputKeyEncodeOptions{}))
	zigTry(t, inputKeyEncodeEncode(&w, InputKeyKeyEvent{Key: InputKeyKeyEscape, Action: InputKeyActionRelease}, ke3ABDis))
	zigTry(t, inputKeyEncodeEncode(&w, ke3Key(InputKeyKeyUnidentified), InputKeyEncodeOptions{}))
	zigTry(t, inputKeyEncodeEncode(&w, ke3Key(InputKeyKeyUnidentified), ke3ABDisAll))

	// Each write point (a cut-off at every length) for a few sequences.
	ke3ABExpect(t, inputKeyEncodeEncode, "\x1bOA", ke3Key(InputKeyKeyArrowUp), InputKeyEncodeOptions{CursorKeyApplication: true})
	ke3ABExpect(t, inputKeyEncodeEncode, "\x1b[27;3;56~", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit8, UTF8: []uint8("8"), Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: '8'}, InputKeyEncodeOptions{ModifyOtherKeysState2: true, MacosOptionAsAlt: InputConfigOptionAsAltTrue})
	ke3ABExpect(t, inputKeyEncodeEncode, "\x1b[97:65;2;65u", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("A"), Mods: InputKeyMods{Shift: true}, UnshiftedCodepoint: 'a'}, InputKeyEncodeOptions{KittyFlags: KittyKeyFlags{Disambiguate: true, ReportAlternates: true, ReportAll: true, ReportAssociated: true}})
}

// Zig: test "term51: pcStyleFunctionKey: cursor and function keys with every modifier combination"
func TestKe3_term51_pcStyleFunctionKey_cursor_and_function_keys_with_every_modifier_combination(t *testing.T) {
	testPlatform(t, "linux")
	for _, c := range ke3ACPcCases {
		for _, cursor := range ke3Bools {
			for _, kpApp := range ke3Bools {
				for _, ignore := range ke3Bools {
					for _, mok := range ke3Bools {
						// No modifiers: cursor mode picks the sequence.
						plain := inputKeyEncodePcStyleFunctionKey(c.key, InputKeyMods{}, cursor, kpApp, ignore, mok)
						wantPlain := c.normal
						if cursor {
							wantPlain = c.application
						}
						expectEqualStrings(t, wantPlain, string(zigUnwrap(t, plain)))

						// Every non-empty combination: cursor mode is irrelevant.
						for bits := 1; bits < 16; bits++ {
							want := fmt.Sprintf("%s%d%s", c.prefix, ke3ACCodes[bits], c.suffix)
							got := inputKeyEncodePcStyleFunctionKey(
								c.key,
								ke3ACMods(bits),
								cursor,
								kpApp,
								ignore,
								mok,
							)
							expectEqualStrings(t, want, string(zigUnwrap(t, got)))
						}
					}
				}
			}
		}
	}
}

// Zig: test "term51: pcStyleFunctionKey: literal sequences"
func TestKe3_term51_pcStyleFunctionKey_literal_sequences(t *testing.T) {
	testPlatform(t, "linux")
	expectEqualStrings(t, "\x1b[A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{}, false, false, false, false)))
	expectEqualStrings(t, "\x1bOA", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{}, true, false, false, false)))
	expectEqualStrings(t, "\x1b[1;2A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{Shift: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[1;2A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{Shift: true}, true, false, false, false)))
	expectEqualStrings(t, "\x1b[1;16D", string(ke3Pc(t, InputKeyKeyArrowLeft, ke3ACMods(15), false, false, false, false)))
	expectEqualStrings(t, "\x1b[H", string(ke3Pc(t, InputKeyKeyHome, InputKeyMods{}, false, false, false, false)))
	expectEqualStrings(t, "\x1bOF", string(ke3Pc(t, InputKeyKeyEnd, InputKeyMods{}, true, false, false, false)))
	expectEqualStrings(t, "\x1b[2~", string(ke3Pc(t, InputKeyKeyInsert, InputKeyMods{}, true, true, true, true)))
	expectEqualStrings(t, "\x1b[3;5~", string(ke3Pc(t, InputKeyKeyDelete, InputKeyMods{Ctrl: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1bOP", string(ke3Pc(t, InputKeyKeyF1, InputKeyMods{}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[1;5P", string(ke3Pc(t, InputKeyKeyF1, InputKeyMods{Ctrl: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1bOR", string(ke3Pc(t, InputKeyKeyF3, InputKeyMods{}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[13;2~", string(ke3Pc(t, InputKeyKeyF3, InputKeyMods{Shift: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[24~", string(ke3Pc(t, InputKeyKeyF12, InputKeyMods{}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[24;9~", string(ke3Pc(t, InputKeyKeyF12, InputKeyMods{Super: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[E", string(ke3Pc(t, InputKeyKeyNumpadBegin, InputKeyMods{}, false, false, false, false)))
	expectEqualStrings(t, "\x1bOE", string(ke3Pc(t, InputKeyKeyNumpadBegin, InputKeyMods{}, true, false, false, false)))
}

// Zig: test "term51: pcStyleFunctionKey: lock keys and sides are ignored"
func TestKe3_term51_pcStyleFunctionKey_lock_keys_and_sides_are_ignored(t *testing.T) {
	testPlatform(t, "linux")
	allRight := InputKeyModsModsSide{Shift: InputKeyModsModSideRight, Ctrl: InputKeyModsModSideRight, Alt: InputKeyModsModSideRight, Super: InputKeyModsModSideRight}

	// Only locks and/or sides: same as no modifiers.
	expectEqualStrings(t, "\x1b[A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{CapsLock: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{NumLock: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1bOA", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{CapsLock: true, NumLock: true}, true, false, false, false)))
	expectEqualStrings(t, "\x1b[A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{Sides: allRight}, false, false, false, false)))
	expectEqualStrings(t, "\x1bOP", string(ke3Pc(t, InputKeyKeyF1, InputKeyMods{CapsLock: true, Sides: allRight}, false, false, false, false)))
	expectEqualStrings(t, "\x7f", string(ke3Pc(t, InputKeyKeyBackspace, InputKeyMods{NumLock: true}, false, false, false, false)))
	expectEqualStrings(t, "\t", string(ke3Pc(t, InputKeyKeyTab, InputKeyMods{CapsLock: true}, false, false, false, false)))
	// Locks and sides next to real modifiers do not change the result.
	expectEqualStrings(t, "\x1b[1;2A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{Shift: true, CapsLock: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[1;2A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{Shift: true, Sides: InputKeyModsModsSide{Shift: InputKeyModsModSideRight}}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[1;5A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{Ctrl: true, NumLock: true, CapsLock: true, Sides: allRight}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[1;16A", string(ke3Pc(t, InputKeyKeyArrowUp, InputKeyMods{Shift: true, Ctrl: true, Alt: true, Super: true, CapsLock: true, NumLock: true, Sides: allRight}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[27;5;9~", string(ke3Pc(t, InputKeyKeyTab, InputKeyMods{Ctrl: true, CapsLock: true, Sides: InputKeyModsModsSide{Ctrl: InputKeyModsModSideRight}}, false, false, false, false)))
	expectEqualStrings(t, "\x1bOq", string(ke3Pc(t, InputKeyKeyNumpad1, InputKeyMods{NumLock: true}, false, true, false, false)))
}

// Zig: test "term51: pcStyleFunctionKey: keypad keys in application mode"
func TestKe3_term51_pcStyleFunctionKey_keypad_keys_in_application_mode(t *testing.T) {
	testPlatform(t, "linux")
	for _, c := range ke3ACKpCases {
		for _, cursor := range ke3Bools {
			for _, mok := range ke3Bools {
				// Requested application mode, numlock not ignored.
				plain := inputKeyEncodePcStyleFunctionKey(c.key, InputKeyMods{}, cursor, true, false, mok)
				wantPlain := "\x1bO" + c.suffix
				expectEqualStrings(t, wantPlain, string(zigUnwrap(t, plain)))

				// Lock keys and sides do not count as modifiers.
				locks := inputKeyEncodePcStyleFunctionKey(c.key, InputKeyMods{CapsLock: true, NumLock: true, Sides: InputKeyModsModsSide{Shift: InputKeyModsModSideRight}}, cursor, true, false, mok)
				expectEqualStrings(t, wantPlain, string(zigUnwrap(t, locks)))

				for bits := 1; bits < 16; bits++ {
					want := fmt.Sprintf("\x1bO%d%s", ke3ACCodes[bits], c.suffix)
					got := inputKeyEncodePcStyleFunctionKey(c.key, ke3ACMods(bits), cursor, true, false, mok)
					expectEqualStrings(t, want, string(zigUnwrap(t, got)))
				}
			}
		}
	}
}

// Zig: test "term51: pcStyleFunctionKey: keypad keys outside application mode"
func TestKe3_term51_pcStyleFunctionKey_keypad_keys_outside_application_mode(t *testing.T) {
	testPlatform(t, "linux")
	for _, c := range ke3ACKpCases {
		// Not requested: normal (numerical) keypad mode.
		// Requested but numlock ignored (mode 1035): also normal.
		reqs := [][2]bool{{false, false}, {false, true}, {true, true}}
		for _, r := range reqs {
			for _, cursor := range ke3Bools {
				for _, mok := range ke3Bools {
					for bits := 0; bits < 16; bits++ {
						got := inputKeyEncodePcStyleFunctionKey(c.key, ke3ACMods(bits), cursor, r[0], r[1], mok)
						if c.key == InputKeyKeyNumpadEnter {
							// The keypad enter falls back to a carriage
							// return regardless of the modifiers.
							expectEqualStrings(t, "\r", string(zigUnwrap(t, got)))
						} else {
							expect(t, !got.Valid, "got == null")
						}
					}
				}
			}
		}
	}
}

// Zig: test "term51: pcStyleFunctionKey: numpad enter"
func TestKe3_term51_pcStyleFunctionKey_numpad_enter(t *testing.T) {
	testPlatform(t, "linux")
	expectEqualStrings(t, "\r", string(ke3Pc(t, InputKeyKeyNumpadEnter, InputKeyMods{}, false, false, false, false)))
	expectEqualStrings(t, "\r", string(ke3Pc(t, InputKeyKeyNumpadEnter, InputKeyMods{Ctrl: true}, false, false, false, false)))
	expectEqualStrings(t, "\r", string(ke3Pc(t, InputKeyKeyNumpadEnter, InputKeyMods{Shift: true, Alt: true}, false, false, false, true)))
	expectEqualStrings(t, "\x1bOM", string(ke3Pc(t, InputKeyKeyNumpadEnter, InputKeyMods{}, false, true, false, false)))
	expectEqualStrings(t, "\x1bO2M", string(ke3Pc(t, InputKeyKeyNumpadEnter, InputKeyMods{Shift: true}, false, true, false, false)))
	expectEqualStrings(t, "\x1bO5M", string(ke3Pc(t, InputKeyKeyNumpadEnter, InputKeyMods{Ctrl: true}, false, true, false, false)))
	expectEqualStrings(t, "\x1bO16M", string(ke3Pc(t, InputKeyKeyNumpadEnter, ke3ACMods(15), false, true, false, false)))
	expectEqualStrings(t, "\r", string(ke3Pc(t, InputKeyKeyNumpadEnter, InputKeyMods{}, false, true, true, false)))
	expectEqualStrings(t, "\r", string(ke3Pc(t, InputKeyKeyNumpadEnter, InputKeyMods{Ctrl: true}, false, true, true, false)))
}

// Zig: test "term51: pcStyleFunctionKey: keypad digits and operators"
func TestKe3_term51_pcStyleFunctionKey_keypad_digits_and_operators(t *testing.T) {
	testPlatform(t, "linux")
	ke3PcNull(t, InputKeyKeyNumpad1, InputKeyMods{}, false, false, false, false)
	expectEqualStrings(t, "\x1bOq", string(ke3Pc(t, InputKeyKeyNumpad1, InputKeyMods{}, false, true, false, false)))
	ke3PcNull(t, InputKeyKeyNumpad1, InputKeyMods{}, false, true, true, false)
	expectEqualStrings(t, "\x1bO3q", string(ke3Pc(t, InputKeyKeyNumpad1, InputKeyMods{Alt: true}, false, true, false, false)))
	ke3PcNull(t, InputKeyKeyNumpad1, InputKeyMods{Alt: true}, false, false, false, false)
	expectEqualStrings(t, "\x1bOp", string(ke3Pc(t, InputKeyKeyNumpad0, InputKeyMods{}, false, true, false, false)))
	expectEqualStrings(t, "\x1bOy", string(ke3Pc(t, InputKeyKeyNumpad9, InputKeyMods{}, false, true, false, false)))
	expectEqualStrings(t, "\x1bOn", string(ke3Pc(t, InputKeyKeyNumpadDecimal, InputKeyMods{}, false, true, false, false)))
	expectEqualStrings(t, "\x1bOk", string(ke3Pc(t, InputKeyKeyNumpadAdd, InputKeyMods{}, false, true, false, false)))

	// Other numpad keys have no entries at all.
	for _, k := range []InputKeyKey{InputKeyKeyNumpadEqual, InputKeyKeyNumpadComma, InputKeyKeyNumpadBackspace, InputKeyKeyNumpadClear, InputKeyKeyNumpadClearEntry, InputKeyKeyNumpadSeparator, InputKeyKeyNumpadParenLeft, InputKeyKeyNumpadParenRight, InputKeyKeyNumpadMemoryAdd} {
		ke3PcNull(t, k, InputKeyMods{}, false, true, false, false)
		ke3PcNull(t, k, InputKeyMods{Shift: true}, true, true, false, true)
	}
}

// Zig: test "term51: pcStyleFunctionKey: backspace entries by modifier and modify other keys"
func TestKe3_term51_pcStyleFunctionKey_backspace_entries_by_modifier_and_modify_other_keys(t *testing.T) {
	testPlatform(t, "linux")
	for bits := 0; bits < 16; bits++ {
		mods := ke3ACMods(bits)
		for _, cursor := range ke3Bools {
			for _, kp := range ke3Bools {
				for _, ignore := range ke3Bools {
					off := inputKeyEncodePcStyleFunctionKey(InputKeyKeyBackspace, mods, cursor, kp, ignore, false)
					expectEqualStrings(t, ke3ACBackspaceOff[bits], string(zigUnwrap(t, off)))
					on := inputKeyEncodePcStyleFunctionKey(InputKeyKeyBackspace, mods, cursor, kp, ignore, true)
					expectEqualStrings(t, ke3ACBackspaceOn[bits], string(zigUnwrap(t, on)))
				}
			}
		}
	}
}

// Zig: test "term51: pcStyleFunctionKey: tab entries by modifier and modify other keys"
func TestKe3_term51_pcStyleFunctionKey_tab_entries_by_modifier_and_modify_other_keys(t *testing.T) {
	testPlatform(t, "linux")
	for bits := 0; bits < 16; bits++ {
		mods := ke3ACMods(bits)
		for _, cursor := range ke3Bools {
			for _, kp := range ke3Bools {
				off := inputKeyEncodePcStyleFunctionKey(InputKeyKeyTab, mods, cursor, kp, false, false)
				expectEqualStrings(t, ke3ACTabOff[bits], string(zigUnwrap(t, off)))
				on := inputKeyEncodePcStyleFunctionKey(InputKeyKeyTab, mods, cursor, kp, false, true)
				expectEqualStrings(t, ke3ACTabOn[bits], string(zigUnwrap(t, on)))
			}
		}
	}
}

// Zig: test "term51: pcStyleFunctionKey: enter entries by modifier and modify other keys"
func TestKe3_term51_pcStyleFunctionKey_enter_entries_by_modifier_and_modify_other_keys(t *testing.T) {
	testPlatform(t, "linux")
	for bits := 0; bits < 16; bits++ {
		mods := ke3ACMods(bits)
		for _, cursor := range ke3Bools {
			for _, kp := range ke3Bools {
				off := inputKeyEncodePcStyleFunctionKey(InputKeyKeyEnter, mods, cursor, kp, false, false)
				expectEqualStrings(t, ke3ACEnterOff[bits], string(zigUnwrap(t, off)))
				on := inputKeyEncodePcStyleFunctionKey(InputKeyKeyEnter, mods, cursor, kp, false, true)
				expectEqualStrings(t, ke3ACEnterOn[bits], string(zigUnwrap(t, on)))
			}
		}
	}
}

// Zig: test "term51: pcStyleFunctionKey: escape entries by modifier"
func TestKe3_term51_pcStyleFunctionKey_escape_entries_by_modifier(t *testing.T) {
	testPlatform(t, "linux")
	for bits := 0; bits < 16; bits++ {
		mods := ke3ACMods(bits)
		for _, mok := range ke3Bools {
			got := inputKeyEncodePcStyleFunctionKey(InputKeyKeyEscape, mods, false, false, false, mok)
			expectEqualStrings(t, ke3ACEscape[bits], string(zigUnwrap(t, got)))
			got2 := inputKeyEncodePcStyleFunctionKey(InputKeyKeyEscape, mods, true, true, true, mok)
			expectEqualStrings(t, ke3ACEscape[bits], string(zigUnwrap(t, got2)))
		}
	}
}

// Zig: test "term51: pcStyleFunctionKey: backspace tab enter escape literals"
func TestKe3_term51_pcStyleFunctionKey_backspace_tab_enter_escape_literals(t *testing.T) {
	testPlatform(t, "linux")
	// The mods_empty_is_any fallback: ctrl+alt+shift+backspace has no
	// matching modify-other-keys-normal entry and falls to the plain DEL.
	expectEqualStrings(t, "\x7f", string(ke3Pc(t, InputKeyKeyBackspace, ke3ACMods(7), false, false, false, false)))
	expectEqualStrings(t, "\x08", string(ke3Pc(t, InputKeyKeyBackspace, InputKeyMods{Ctrl: true}, false, false, false, true)))
	expectEqualStrings(t, "\x08", string(ke3Pc(t, InputKeyKeyBackspace, InputKeyMods{Ctrl: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[27;2;127~", string(ke3Pc(t, InputKeyKeyBackspace, InputKeyMods{Shift: true}, false, false, false, true)))
	expectEqualStrings(t, "\x7f", string(ke3Pc(t, InputKeyKeyBackspace, InputKeyMods{Shift: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[Z", string(ke3Pc(t, InputKeyKeyTab, InputKeyMods{Shift: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[27;2;9~", string(ke3Pc(t, InputKeyKeyTab, InputKeyMods{Shift: true}, false, false, false, true)))
	expectEqualStrings(t, "\x1b\t", string(ke3Pc(t, InputKeyKeyTab, InputKeyMods{Alt: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[27;2;13~", string(ke3Pc(t, InputKeyKeyEnter, InputKeyMods{Shift: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[27;2;13~", string(ke3Pc(t, InputKeyKeyEnter, InputKeyMods{Shift: true}, false, false, false, true)))
	expectEqualStrings(t, "\x1b\r", string(ke3Pc(t, InputKeyKeyEnter, InputKeyMods{Alt: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[27;3;13~", string(ke3Pc(t, InputKeyKeyEnter, InputKeyMods{Alt: true}, false, false, false, true)))
	expectEqualStrings(t, "\x1b\x1b", string(ke3Pc(t, InputKeyKeyEscape, InputKeyMods{Alt: true}, false, false, false, false)))
	expectEqualStrings(t, "\x1b[27;5;27~", string(ke3Pc(t, InputKeyKeyEscape, InputKeyMods{Ctrl: true}, false, false, false, false)))
}

// Zig: test "term51: pcStyleFunctionKey: keys without entries return null"
func TestKe3_term51_pcStyleFunctionKey_keys_without_entries_return_null(t *testing.T) {
	testPlatform(t, "linux")
	for _, k := range InputKeyKeyValues() {
		if ke3IndexOfKey(ke3ACKeysWithEntries, k) {
			continue
		}
		for bits := 0; bits < 16; bits++ {
			for _, cursor := range ke3Bools {
				for _, kp := range ke3Bools {
					for _, ignore := range ke3Bools {
						for _, mok := range ke3Bools {
							ke3PcNull(t, k, ke3ACMods(bits), cursor, kp, ignore, mok)
						}
					}
				}
			}
		}
	}

	// F13 to F25 in particular, and the modifier keys.
	for _, k := range []InputKeyKey{InputKeyKeyF13, InputKeyKeyF14, InputKeyKeyF15, InputKeyKeyF16, InputKeyKeyF17, InputKeyKeyF18, InputKeyKeyF19, InputKeyKeyF20, InputKeyKeyF21, InputKeyKeyF22, InputKeyKeyF23, InputKeyKeyF24, InputKeyKeyF25, InputKeyKeyShiftLeft, InputKeyKeyShiftRight, InputKeyKeyControlLeft, InputKeyKeyControlRight, InputKeyKeyAltLeft, InputKeyKeyAltRight, InputKeyKeyMetaLeft, InputKeyKeyMetaRight, InputKeyKeyCapsLock, InputKeyKeyNumLock, InputKeyKeyUnidentified, InputKeyKeyKeyA, InputKeyKeySpace, InputKeyKeyDigit1} {
		ke3PcNull(t, k, InputKeyMods{}, false, false, false, false)
		ke3PcNull(t, k, InputKeyMods{Ctrl: true}, true, true, false, true)
	}
}

// Zig: test "term51: pcStyleFunctionKey: every key with entries matches with no modifiers"
func TestKe3_term51_pcStyleFunctionKey_every_key_with_entries_matches_with_no_modifiers(t *testing.T) {
	testPlatform(t, "linux")
	for _, k := range ke3ACKeysWithEntries {
		// Keypad keys only match in application mode (or enter, always).
		got := inputKeyEncodePcStyleFunctionKey(k, InputKeyMods{}, false, true, false, false)
		expect(t, got.Valid, "got != null")
	}
	count := 0
	for _, k := range InputKeyKeyValues() {
		if inputKeyEncodePcStyleFunctionKey(k, InputKeyMods{}, false, true, false, false).Valid {
			count++
		}
	}
	expectEqual(t, len(ke3ACKeysWithEntries), count)
}

// Zig: test "term51: ctrlSeq: every single byte with exactly ctrl"
func TestKe3_term51_ctrlSeq_every_single_byte_with_exactly_ctrl(t *testing.T) {
	testPlatform(t, "linux")
	for i := 0; i < 256; i++ {
		s := [1]uint8{uint8(i)}
		want := ke3ACCtrlExpected(uint8(i))
		expectEqual(t, want, inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0, InputKeyMods{Ctrl: true}))
		// The logical key is irrelevant when the utf8 is a single byte,
		// and so is a matching unshifted codepoint.
		expectEqual(t, want, inputKeyEncodeCtrlSeq(InputKeyKeyKeyA, s[:], uint32(i), InputKeyMods{Ctrl: true}))
	}
}

// Zig: test "term51: ctrlSeq: literal outputs"
func TestKe3_term51_ctrlSeq_literal_outputs(t *testing.T) {
	testPlatform(t, "linux")
	expectEqual(t, Some[uint8](0), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8(" "), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](31), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("/"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](48), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("0"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](49), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("1"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](0), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("2"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](27), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("3"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](28), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("4"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](29), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("5"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](30), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("6"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](31), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("7"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](127), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("8"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](57), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("9"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](127), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("?"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](0), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("@"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](28), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("\\"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](29), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("]"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](30), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("^"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](31), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("_"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](30), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("~"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](1), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("a"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](26), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("z"), 0, InputKeyMods{Ctrl: true}))
	// Letters handled by fixterms as CSI u instead.
	expectEqual(t, None[uint8](), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("i"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, None[uint8](), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("m"), 0, InputKeyMods{Ctrl: true}))
	expectEqual(t, None[uint8](), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("["), 0, InputKeyMods{Ctrl: true}))
}

// Zig: test "term51: ctrlSeq: lock keys, sides and alt do not matter"
func TestKe3_term51_ctrlSeq_lock_keys_sides_and_alt_do_not_matter(t *testing.T) {
	testPlatform(t, "linux")
	variants := []InputKeyMods{
		InputKeyMods{Ctrl: true, CapsLock: true},
		InputKeyMods{Ctrl: true, NumLock: true},
		InputKeyMods{Ctrl: true, CapsLock: true, NumLock: true},
		InputKeyMods{Ctrl: true, Sides: InputKeyModsModsSide{Ctrl: InputKeyModsModSideRight}},
		InputKeyMods{Ctrl: true, Sides: InputKeyModsModsSideMask.Sides},
		InputKeyMods{Ctrl: true, Alt: true},
		InputKeyMods{Ctrl: true, Alt: true, CapsLock: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideRight}},
	}
	for _, mods := range variants {
		for i := 0; i < 256; i++ {
			s := [1]uint8{uint8(i)}
			expectEqual(t, ke3ACCtrlExpected(uint8(i)), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0, mods))
		}
	}
}

// Zig: test "term51: ctrlSeq: shift is dropped except for letters and at-sign"
func TestKe3_term51_ctrlSeq_shift_is_dropped_except_for_letters_and_at_sign(t *testing.T) {
	testPlatform(t, "linux")
	for i := 0; i < 256; i++ {
		s := [1]uint8{uint8(i)}
		keepShift := (i >= 'A' && i <= 'Z') || i == '@'
		want := ke3ACCtrlExpected(uint8(i))
		if keepShift {
			want = None[uint8]()
		}
		expectEqual(t, want, inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0, InputKeyMods{Ctrl: true, Shift: true}))
		expectEqual(t, want, inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0, InputKeyMods{Ctrl: true, Shift: true, Alt: true}))
	}

	// Lowercase letters have shift dropped too.
	expectEqual(t, Some[uint8](3), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("c"), 'c', InputKeyMods{Ctrl: true, Shift: true}))
	// Fixterms awkward case: ctrl+shift+@ is not 0.
	expectEqual(t, None[uint8](), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("@"), '2', InputKeyMods{Ctrl: true, Shift: true}))
	// Emacs style ctrl+shift+minus (underscore) and friends.
	expectEqual(t, Some[uint8](31), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("_"), '-', InputKeyMods{Ctrl: true, Shift: true}))
	expectEqual(t, Some[uint8](0), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("@"), '2', InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](127), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("?"), '/', InputKeyMods{Ctrl: true, Shift: true}))
	expectEqual(t, Some[uint8](30), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("^"), '6', InputKeyMods{Ctrl: true, Shift: true}))
}

// Zig: test "term51: ctrlSeq: without ctrl never matches"
func TestKe3_term51_ctrlSeq_without_ctrl_never_matches(t *testing.T) {
	testPlatform(t, "linux")
	variants := []InputKeyMods{
		InputKeyMods{},
		InputKeyMods{Shift: true},
		InputKeyMods{Alt: true},
		InputKeyMods{Super: true},
		InputKeyMods{Shift: true, Alt: true, Super: true},
		InputKeyMods{CapsLock: true},
		InputKeyMods{NumLock: true},
		InputKeyMods{Sides: InputKeyModsModsSideMask.Sides},
	}
	for _, mods := range variants {
		for i := 0; i < 256; i++ {
			s := [1]uint8{uint8(i)}
			expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0, mods).Valid, "ctrlSeq(.unidentified, &s, 0, mods) == null")
			expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyKeyC, s[:], 'c', mods).Valid, "ctrlSeq(.key_c, &s, 'c', mods) == null")
		}
		for _, k := range InputKeyKeyValues() {
			expect(t, !inputKeyEncodeCtrlSeq(k, []uint8(""), 0, mods).Valid, "ctrlSeq(k, \"\", 0, mods) == null")
		}
	}
}

// Zig: test "term51: ctrlSeq: super or hyper-like extra modifiers prevent a match"
func TestKe3_term51_ctrlSeq_super_or_hyper_like_extra_modifiers_prevent_a_match(t *testing.T) {
	testPlatform(t, "linux")
	variants := []InputKeyMods{
		InputKeyMods{Ctrl: true, Super: true},
		InputKeyMods{Ctrl: true, Super: true, Alt: true},
		InputKeyMods{Ctrl: true, Super: true, Shift: true},
		InputKeyMods{Ctrl: true, Super: true, CapsLock: true},
	}
	for _, mods := range variants {
		for i := 0; i < 256; i++ {
			s := [1]uint8{uint8(i)}
			expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0, mods).Valid, "ctrlSeq(.unidentified, &s, 0, mods) == null")
		}
		for _, k := range InputKeyKeyValues() {
			expect(t, !inputKeyEncodeCtrlSeq(k, []uint8(""), 0, mods).Valid, "ctrlSeq(k, \"\", 0, mods) == null")
		}
	}
}

// Zig: test "term51: ctrlSeq: uppercase letters use the unshifted codepoint"
func TestKe3_term51_ctrlSeq_uppercase_letters_use_the_unshifted_codepoint(t *testing.T) {
	testPlatform(t, "linux")
	for c := uint8('A'); c <= 'Z'; c++ {
		s := [1]uint8{c}
		lower := uint32(c) + 32
		want := ke3ACCtrlExpected(c + 32)
		// Caps lock: uppercase text with a lowercase unshifted codepoint.
		expectEqual(t, want, inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], lower, InputKeyMods{Ctrl: true, CapsLock: true}))
		expectEqual(t, want, inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], lower, InputKeyMods{Ctrl: true}))
		// Shift keeps it from matching.
		expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], lower, InputKeyMods{Ctrl: true, Shift: true}).Valid, "ctrlSeq(.unidentified, &s, lower, .{ .ctrl = true, .shift = true }) == null")
		// No unshifted codepoint, or an uppercase one: stays uppercase.
		expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, &s, 0, .{ .ctrl = true }) == null")
		expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], uint32(c), InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, &s, c, .{ .ctrl = true }) == null")
		// An unshifted codepoint that does not fit a byte is not used.
		expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0x100, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, &s, 0x100, .{ .ctrl = true }) == null")
		expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0x441, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, &s, 0x441, .{ .ctrl = true }) == null")
		expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, s[:], 0x1FFFFF, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, &s, 0x1FFFFF, .{ .ctrl = true }) == null")
	}
}

// Zig: test "term51: ctrlSeq: unshifted codepoint boundaries"
func TestKe3_term51_ctrlSeq_unshifted_codepoint_boundaries(t *testing.T) {
	testPlatform(t, "linux")
	// 0xFF is the last codepoint that fits a byte; it maps to nothing.
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("C"), 0xFF, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, \"C\", 0xFF, .{ .ctrl = true }) == null")
	// 0x100 is the first that does not.
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("C"), 0x100, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, \"C\", 0x100, .{ .ctrl = true }) == null")
	// Any byte-sized unshifted codepoint replaces an uppercase char.
	expectEqual(t, Some[uint8](49), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("C"), '1', InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](0), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("C"), ' ', InputKeyMods{Ctrl: true}))
	expectEqual(t, None[uint8](), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("C"), 0x01, InputKeyMods{Ctrl: true}))
	// Only A-Z are replaced: lowercase and other characters keep their own.
	expectEqual(t, Some[uint8](3), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("c"), 'x', InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](49), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("1"), 'x', InputKeyMods{Ctrl: true}))
	// A and Z are the range boundaries; '@' (0x40) and '[' (0x5B) are outside.
	expectEqual(t, Some[uint8](1), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("A"), 'a', InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](26), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("Z"), 'z', InputKeyMods{Ctrl: true}))
	expectEqual(t, Some[uint8](0), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("@"), 'a', InputKeyMods{Ctrl: true}))
	expectEqual(t, None[uint8](), inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("["), 'a', InputKeyMods{Ctrl: true}))
	// I and M are never converted.
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("I"), 'i', InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, \"I\", 'i', .{ .ctrl = true }) == null")
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("M"), 'm', InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, \"M\", 'm', .{ .ctrl = true }) == null")
}

// Zig: test "term51: ctrlSeq: logical key used when utf8 is not a single byte"
func TestKe3_term51_ctrlSeq_logical_key_used_when_utf8_is_not_a_single_byte(t *testing.T) {
	testPlatform(t, "linux")
	for _, k := range InputKeyKeyValues() {
		want := ke3ACCtrlKeyExpected(k)
		// Empty text.
		expectEqual(t, want, inputKeyEncodeCtrlSeq(k, []uint8(""), 0, InputKeyMods{Ctrl: true}))
		// Unshifted codepoint is not consulted on this path.
		expectEqual(t, want, inputKeyEncodeCtrlSeq(k, []uint8(""), 'x', InputKeyMods{Ctrl: true}))
		expectEqual(t, want, inputKeyEncodeCtrlSeq(k, []uint8(""), 0x441, InputKeyMods{Ctrl: true}))
		// Multi-byte and multi-codepoint text.
		expectEqual(t, want, inputKeyEncodeCtrlSeq(k, []uint8("\xd1\x81"), 0x441, InputKeyMods{Ctrl: true}))
		expectEqual(t, want, inputKeyEncodeCtrlSeq(k, []uint8("ab"), 'a', InputKeyMods{Ctrl: true}))
		expectEqual(t, want, inputKeyEncodeCtrlSeq(k, []uint8("\xf0\x9f\x98\x80"), 0, InputKeyMods{Ctrl: true}))
		// Lock keys, sides and alt are tolerated.
		expectEqual(t, want, inputKeyEncodeCtrlSeq(k, []uint8(""), 0, InputKeyMods{Ctrl: true, CapsLock: true, NumLock: true}))
		expectEqual(t, want, inputKeyEncodeCtrlSeq(k, []uint8(""), 0, InputKeyMods{Ctrl: true, Sides: InputKeyModsModsSide{Ctrl: InputKeyModsModSideRight}}))
		expectEqual(t, want, inputKeyEncodeCtrlSeq(k, []uint8(""), 0, InputKeyMods{Ctrl: true, Alt: true}))
		// Any other modifier prevents it, including shift.
		expect(t, !inputKeyEncodeCtrlSeq(k, []uint8(""), 0, InputKeyMods{Ctrl: true, Shift: true}).Valid, "ctrlSeq(k, \"\", 0, .{ .ctrl = true, .shift = true }) == null")
		expect(t, !inputKeyEncodeCtrlSeq(k, []uint8(""), 0, InputKeyMods{Ctrl: true, Super: true}).Valid, "ctrlSeq(k, \"\", 0, .{ .ctrl = true, .super = true }) == null")
		expect(t, !inputKeyEncodeCtrlSeq(k, []uint8(""), 0, InputKeyMods{Ctrl: true, Shift: true, Alt: true}).Valid, "ctrlSeq(k, \"\", 0, .{ .ctrl = true, .shift = true, .alt = true }) == null")
		expect(t, !inputKeyEncodeCtrlSeq(k, []uint8(""), 0, InputKeyMods{Ctrl: true, Alt: true, Super: true}).Valid, "ctrlSeq(k, \"\", 0, .{ .ctrl = true, .alt = true, .super = true }) == null")
	}
}

// Zig: test "term51: ctrlSeq: unidentified key with non single byte text"
func TestKe3_term51_ctrlSeq_unidentified_key_with_non_single_byte_text(t *testing.T) {
	testPlatform(t, "linux")
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8(""), 0, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, \"\", 0, .{ .ctrl = true }) == null")
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8(""), 'c', InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, \"\", 'c', .{ .ctrl = true }) == null")
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("cc"), 'c', InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, \"cc\", 'c', .{ .ctrl = true }) == null")
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("\xd1\x81"), 0x441, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, \"\\xd1\\x81\", 0x441, .{ .ctrl = true }) == null")
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("\xf0\x9f\x98\x80"), 0x1F600, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.unidentified, \"\\xf0\\x9f\\x98\\x80\", 0x1F600, .{ .ctrl = true }) == null")
}

// Zig: test "term51: ctrlSeq: single byte text takes priority over the logical key"
func TestKe3_term51_ctrlSeq_single_byte_text_takes_priority_over_the_logical_key(t *testing.T) {
	testPlatform(t, "linux")
	// The text byte decides, even when the key would map elsewhere or nowhere.
	expectEqual(t, Some[uint8](3), inputKeyEncodeCtrlSeq(InputKeyKeyKeyA, []uint8("c"), 'a', InputKeyMods{Ctrl: true}))
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyKeyC, []uint8("i"), 'i', InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.key_c, \"i\", 'i', .{ .ctrl = true }) == null")
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyKeyC, []uint8("\x01"), 'c', InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.key_c, \"\\x01\", 'c', .{ .ctrl = true }) == null")
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyKeyC, []uint8("\xff"), 'c', InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.key_c, \"\\xff\", 'c', .{ .ctrl = true }) == null")
	expectEqual(t, Some[uint8](0), inputKeyEncodeCtrlSeq(InputKeyKeyKeyC, []uint8(" "), 'c', InputKeyMods{Ctrl: true}))
}

// Zig: test "term51: ctrlSeq: keys without a codepoint"
func TestKe3_term51_ctrlSeq_keys_without_a_codepoint(t *testing.T) {
	testPlatform(t, "linux")
	keys := []InputKeyKey{InputKeyKeyArrowUp, InputKeyKeyEnter, InputKeyKeyEscape, InputKeyKeyBackspace, InputKeyKeyF1, InputKeyKeyF25, InputKeyKeyShiftLeft, InputKeyKeyControlRight, InputKeyKeyAltLeft, InputKeyKeyMetaLeft, InputKeyKeyCapsLock, InputKeyKeyDelete, InputKeyKeyPageDown, InputKeyKeyNumpadEnter, InputKeyKeyNumpadUp, InputKeyKeyContextMenu, InputKeyKeyAudioVolumeMute}
	for _, k := range keys {
		expect(t, !inputKeyEncodeCtrlSeq(k, []uint8(""), 0, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(k, \"\", 0, .{ .ctrl = true }) == null")
		expect(t, !inputKeyEncodeCtrlSeq(k, []uint8("ab"), 0, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(k, \"ab\", 0, .{ .ctrl = true }) == null")
	}
	// Tab has a codepoint (0x09) that maps to nothing.
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyTab, []uint8(""), 0, InputKeyMods{Ctrl: true}).Valid, "ctrlSeq(.tab, \"\", 0, .{ .ctrl = true }) == null")
}

// Zig: test "term51: isControl: every codepoint up to the first non-ASCII block"
func TestKe3_term51_isControl_every_codepoint_up_to_the_first_non_ASCII_block(t *testing.T) {
	testPlatform(t, "linux")
	for i := 0; i < 0x200; i++ {
		want := i < 0x20 || i == 0x7F
		expectEqual(t, want, inputKeyEncodeIsControl(uint32(i)))
	}
	expect(t, inputKeyEncodeIsControl(0), "isControl(0)")
	expect(t, inputKeyEncodeIsControl(0x1F), "isControl(0x1F)")
	expect(t, !inputKeyEncodeIsControl(0x20), "!isControl(0x20)")
	expect(t, !inputKeyEncodeIsControl(0x7E), "!isControl(0x7E)")
	expect(t, inputKeyEncodeIsControl(0x7F), "isControl(0x7F)")
	expect(t, !inputKeyEncodeIsControl(0x80), "!isControl(0x80)")
	// C1 controls are not controls here.
	expect(t, !inputKeyEncodeIsControl(0x9F), "!isControl(0x9F)")
	expect(t, !inputKeyEncodeIsControl(0xFF), "!isControl(0xFF)")
	expect(t, !inputKeyEncodeIsControl(0x10FFFF), "!isControl(0x10FFFF)")
	expect(t, !inputKeyEncodeIsControl(0x1FFFFF), "!isControl(0x1FFFFF)")
}

// Zig: test "term51: isControlUtf8: single bytes"
func TestKe3_term51_isControlUtf8_single_bytes(t *testing.T) {
	testPlatform(t, "linux")
	for i := 0; i < 256; i++ {
		s := [1]uint8{uint8(i)}
		want := i < 0x20 || i == 0x7F
		expectEqual(t, want, inputKeyEncodeIsControlUtf8(s[:]))
	}
}

// Zig: test "term51: isControlUtf8: empty and multi byte strings"
func TestKe3_term51_isControlUtf8_empty_and_multi_byte_strings(t *testing.T) {
	testPlatform(t, "linux")
	expect(t, !inputKeyEncodeIsControlUtf8([]uint8("")), "!isControlUtf8(\"\")")
	expect(t, !inputKeyEncodeIsControlUtf8([]uint8("\x01\x01")), "!isControlUtf8(\"\\x01\\x01\")")
	expect(t, !inputKeyEncodeIsControlUtf8([]uint8("\r\n")), "!isControlUtf8(\"\\r\\n\")")
	expect(t, !inputKeyEncodeIsControlUtf8([]uint8("\x7f\x7f")), "!isControlUtf8(\"\\x7f\\x7f\")")
	expect(t, !inputKeyEncodeIsControlUtf8([]uint8("a\x01")), "!isControlUtf8(\"a\\x01\")")
	expect(t, !inputKeyEncodeIsControlUtf8([]uint8("\x01a")), "!isControlUtf8(\"\\x01a\")")
	expect(t, !inputKeyEncodeIsControlUtf8([]uint8("\xc2\x85")), "!isControlUtf8(\"\\xc2\\x85\")")
	expect(t, !inputKeyEncodeIsControlUtf8([]uint8("\xc3\xa9")), "!isControlUtf8(\"\\xc3\\xa9\")")
	expect(t, !inputKeyEncodeIsControlUtf8([]uint8("\xf0\x9f\x98\x80")), "!isControlUtf8(\"\\xf0\\x9f\\x98\\x80\")")
	expect(t, inputKeyEncodeIsControlUtf8([]uint8("\x1b")), "isControlUtf8(\"\\x1b\")")
	expect(t, inputKeyEncodeIsControlUtf8([]uint8("\x00")), "isControlUtf8(\"\\x00\")")
}

// Zig: test "term51: copyToBuf: exact fit, shorter source, and one short"
func TestKe3_term51_copyToBuf_exact_fit_shorter_source_and_one_short(t *testing.T) {
	testPlatform(t, "linux")
	var buf [8]uint8

	// Exact fit.
	ke3Memset(buf[:], 0xAA)
	exact, err := inputKeyEncodeCopyToBuf(buf[0:4], []uint8("abcd"))
	zigTry(t, err)
	expectEqualStrings(t, "abcd", string(exact))
	expectEqual(t, 4, len(exact))
	expect(t, &buf[0] == &exact[0], "@intFromPtr(&buf) == @intFromPtr(exact.ptr)")
	expectEqual(t, uint8(0xAA), buf[4])

	// Source shorter than the buffer: only src.len bytes are written.
	ke3Memset(buf[:], 0xAA)
	shorter, err := inputKeyEncodeCopyToBuf(buf[:], []uint8("abc"))
	zigTry(t, err)
	expectEqualStrings(t, "abc", string(shorter))
	expect(t, &buf[0] == &shorter[0], "@intFromPtr(&buf) == @intFromPtr(shorter.ptr)")
	expectEqualSlices(t, []uint8{'a', 'b', 'c', 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}, buf[:])

	// Full 8 byte buffer, exact fit.
	full, err := inputKeyEncodeCopyToBuf(buf[:], []uint8("12345678"))
	zigTry(t, err)
	expectEqualStrings(t, "12345678", string(full))

	// One byte too long: error and the buffer is untouched.
	ke3Memset(buf[:], 0xAA)
	_, err = inputKeyEncodeCopyToBuf(buf[:], []uint8("123456789"))
	ke3ExpectError(t, ErrOutOfMemory, err)
	_, err = inputKeyEncodeCopyToBuf(buf[0:3], []uint8("abcd"))
	ke3ExpectError(t, ErrOutOfMemory, err)
	_, err = inputKeyEncodeCopyToBuf(buf[0:1], []uint8("ab"))
	ke3ExpectError(t, ErrOutOfMemory, err)
	for _, c := range buf {
		expectEqual(t, uint8(0xAA), c)
	}
}

// ke3Memset is Zig's @memset.
func ke3Memset(buf []uint8, v uint8) {
	for i := range buf {
		buf[i] = v
	}
}

// Zig: test "term51: copyToBuf: empty buffers and empty source"
func TestKe3_term51_copyToBuf_empty_buffers_and_empty_source(t *testing.T) {
	testPlatform(t, "linux")
	var buf [4]uint8
	ke3Memset(buf[:], 0xAA)

	a, err := inputKeyEncodeCopyToBuf(buf[:], []uint8(""))
	zigTry(t, err)
	expectEqual(t, 0, len(a))
	expectEqual(t, uint8(0xAA), buf[0])

	b, err := inputKeyEncodeCopyToBuf(buf[0:0], []uint8(""))
	zigTry(t, err)
	expectEqual(t, 0, len(b))

	// Zero-length destination with one byte of source.
	_, err = inputKeyEncodeCopyToBuf(buf[0:0], []uint8("x"))
	ke3ExpectError(t, ErrOutOfMemory, err)

	// A single byte.
	c, err := inputKeyEncodeCopyToBuf(buf[0:1], []uint8("x"))
	zigTry(t, err)
	expectEqualStrings(t, "x", string(c))
	expectEqual(t, uint8('x'), buf[0])
	expectEqual(t, uint8(0xAA), buf[1])

	// Bytes are copied verbatim, including NUL and non-UTF-8.
	d, err := inputKeyEncodeCopyToBuf(buf[:], []uint8("\x00\xff\x80\x7f"))
	zigTry(t, err)
	expectEqualSlices(t, []uint8{0, 0xff, 0x80, 0x7f}, d)
}

// Zig: test "term51: legacyAltPrefix: requires alt and alt_esc_prefix"
func TestKe3_term51_legacyAltPrefix_requires_alt_and_alt_esc_prefix(t *testing.T) {
	testPlatform(t, "linux")
	ev := InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("c"), UnshiftedCodepoint: 'c'}
	on := InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: InputConfigOptionAsAltTrue}
	off := InputKeyEncodeOptions{AltEscPrefix: false, MacosOptionAsAlt: InputConfigOptionAsAltTrue}

	expectEqual(t, Some[uint8]('c'), ke3Lap(t, ev, InputKeyMods{Alt: true}, ev.Mods, on))
	expectEqual(t, None[uint8](), ke3Lap(t, ev, InputKeyMods{Alt: true}, ev.Mods, off))
	// Only the binding mods argument is consulted for alt, not the event.
	expectEqual(t, None[uint8](), ke3Lap(t, ev, InputKeyMods{}, ev.Mods, on))
	expectEqual(t, None[uint8](), ke3Lap(t, ev, InputKeyMods{Ctrl: true, Shift: true, Super: true}, ev.Mods, on))
	expectEqual(t, Some[uint8]('c'), ke3Lap(t, ev, InputKeyMods{Alt: true, Ctrl: true, Shift: true, Super: true}, ev.Mods, on))
	// The event's own mods are not consulted.
	ev2 := ev
	ev2.Mods = InputKeyMods{}
	expectEqual(t, Some[uint8]('c'), ke3Lap(t, ev2, InputKeyMods{Alt: true}, ev2.Mods, on))
	// Default options have alt_esc_prefix off.
	expectEqual(t, None[uint8](), ke3Lap(t, ev, InputKeyMods{Alt: true}, ev.Mods, InputKeyEncodeOptions{}))
}

// Zig: test "term51: legacyAltPrefix: byte selection from utf8 and unshifted codepoint"
func TestKe3_term51_legacyAltPrefix_byte_selection_from_utf8_and_unshifted_codepoint(t *testing.T) {
	testPlatform(t, "linux")
	on := InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: InputConfigOptionAsAltTrue}
	alt := InputKeyMods{Alt: true}

	// A single utf8 byte is used as is, even NUL and 0x80..0xFF.
	expectEqual(t, Some[uint8]('c'), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("c")}, alt, alt, on))
	expectEqual(t, Some[uint8]('C'), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("C"), UnshiftedCodepoint: 'c'}, alt, alt, on))
	expectEqual(t, Some[uint8](0), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("\x00")}, alt, alt, on))
	expectEqual(t, Some[uint8](0x1b), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("\x1b")}, alt, alt, on))
	expectEqual(t, Some[uint8](0xFF), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("\xff")}, alt, alt, on))
	expectEqual(t, Some[uint8](0x80), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("\x80"), UnshiftedCodepoint: 'x'}, alt, alt, on))
	// Empty utf8: unshifted codepoints 1..255 are allowed through.
	expectEqual(t, None[uint8](), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UnshiftedCodepoint: 0}, alt, alt, on))
	expectEqual(t, Some[uint8](1), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UnshiftedCodepoint: 1}, alt, alt, on))
	expectEqual(t, Some[uint8]('e'), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UnshiftedCodepoint: 'e'}, alt, alt, on))
	expectEqual(t, Some[uint8](0xE9), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UnshiftedCodepoint: 0xE9}, alt, alt, on))
	expectEqual(t, Some[uint8](0xFF), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UnshiftedCodepoint: 0xFF}, alt, alt, on))
	expectEqual(t, None[uint8](), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UnshiftedCodepoint: 0x100}, alt, alt, on))
	expectEqual(t, None[uint8](), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UnshiftedCodepoint: 0x441}, alt, alt, on))
	expectEqual(t, None[uint8](), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UnshiftedCodepoint: 0x1FFFFF}, alt, alt, on))
	// Multi byte utf8 falls back to the unshifted codepoint.
	expectEqual(t, Some[uint8]('e'), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("\xc3\xa9"), UnshiftedCodepoint: 'e'}, alt, alt, on))
	expectEqual(t, Some[uint8]('a'), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("ab"), UnshiftedCodepoint: 'a'}, alt, alt, on))
	expectEqual(t, None[uint8](), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("\xc3\xa9")}, alt, alt, on))
	expectEqual(t, None[uint8](), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("\xc3\xa9"), UnshiftedCodepoint: 0x100}, alt, alt, on))
	expectEqual(t, None[uint8](), ke3Lap(t, InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("\xf0\x9f\x98\x80"), UnshiftedCodepoint: 0x1F600}, alt, alt, on))
}

// Zig: test "term51: legacyAltPrefix: macos_option_as_alt and alt side"
func TestKe3_term51_legacyAltPrefix_macos_option_as_alt_and_alt_side(t *testing.T) {
	// Go-only: platform modes
	for _, goos := range ke3Platforms {
		t.Run(goos, func(t *testing.T) {
			testPlatform(t, goos)
			altLeft := InputKeyMods{Alt: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideLeft}}
			altRight := InputKeyMods{Alt: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideRight}}
			ev := InputKeyKeyEvent{Action: InputKeyActionPress, UTF8: []uint8("c")}
			opts := []InputConfigOptionAsAlt{InputConfigOptionAsAltFalse, InputConfigOptionAsAltTrue, InputConfigOptionAsAltLeft, InputConfigOptionAsAltRight}

			// Expected results for [option][side] on macOS: false never, true
			// always, left only the left alt, right only the right alt.
			c := Some[uint8]('c')
			n := None[uint8]()
			mac := [4][2]Optional[uint8]{
				{n, n},
				{c, c},
				{c, n},
				{n, c},
			}
			for i, opt := range opts {
				o := InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: opt}
				left := ke3Lap(t, ev, altLeft, altLeft, o)
				right := ke3Lap(t, ev, altRight, altRight, o)
				if ke3ACMac(goos) {
					expectEqual(t, mac[i][0], left)
					expectEqual(t, mac[i][1], right)
				} else {
					// Everywhere else the option is not consulted at all.
					expectEqual(t, Some[uint8]('c'), left)
					expectEqual(t, Some[uint8]('c'), right)
				}
			}

			// The side is read from the mods argument, not the binding mods.
			o := InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: InputConfigOptionAsAltLeft}
			got := ke3Lap(t, ev, InputKeyMods{Alt: true}, altRight, o)
			if ke3ACMac(goos) {
				expectEqual(t, None[uint8](), got)
			} else {
				expectEqual(t, Some[uint8]('c'), got)
			}
		})
	}
}

// Zig: test "term51: legacy: only press and repeat produce output"
func TestKe3_term51_legacy_only_press_and_repeat_produce_output(t *testing.T) {
	testPlatform(t, "linux")
	events := []ke3ACCase{
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a")}, opts: InputKeyEncodeOptions{}, want: "a"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter}, opts: InputKeyEncodeOptions{}, want: "\r"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyF1}, opts: InputKeyEncodeOptions{}, want: "\x1bOP"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp, Mods: InputKeyMods{Shift: true}}, opts: InputKeyEncodeOptions{}, want: "\x1b[1;2A"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{}, want: "\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace}, opts: InputKeyEncodeOptions{}, want: "\x7f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("i")}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: "\x1be"},
	}
	for _, c := range events {
		ev := c.ev
		ev.Action = InputKeyActionPress
		ke3ACExpect(t, c.want, ev, c.opts)
		ev.Action = InputKeyActionRepeat
		ke3ACExpect(t, c.want, ev, c.opts)
		ev.Action = InputKeyActionRelease
		ke3ACExpect(t, "", ev, c.opts)
	}

	// Release stays silent for state 2 and every key class too.
	o := ke3ACMokAltOpts
	for _, k := range InputKeyKeyValues() {
		ke3ACExpect(t, "", InputKeyKeyEvent{Action: InputKeyActionRelease, Key: k, UTF8: []uint8("a"), UnshiftedCodepoint: 'a', Mods: InputKeyMods{Shift: true}}, o)
		ke3ACExpect(t, "", InputKeyKeyEvent{Action: InputKeyActionRelease, Key: k, Mods: InputKeyMods{Ctrl: true}}, o)
	}
	o.ModifyOtherKeysState2 = false
	ke3ACExpect(t, "", InputKeyKeyEvent{Action: InputKeyActionRelease, Key: InputKeyKeyTab}, o)
}

// Zig: test "term51: legacy: composing produces no output"
func TestKe3_term51_legacy_composing_produces_no_output(t *testing.T) {
	testPlatform(t, "linux")
	events := []InputKeyKeyEvent{
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("\xc3\xa9")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8("\r")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\x7f")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyF1, Mods: InputKeyMods{Ctrl: true}},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("c")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("c")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("i")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'e'},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1")},
	}
	for _, ev0 := range events {
		ev := ev0
		ev.Composing = true
		for _, action := range []InputKeyAction{InputKeyActionPress, InputKeyActionRepeat, InputKeyActionRelease} {
			ev.Action = action
			ke3ACExpect(t, "", ev, InputKeyEncodeOptions{})
			ke3ACExpect(t, "", ev, ke3ACMokAltOpts)
			ke3ACExpect(t, "", ev, InputKeyEncodeOptions{CursorKeyApplication: true, KeypadKeyApplication: true})
		}
	}
}

// Zig: test "term51: legacy: pc style keys take priority over text and ctrl sequences"
func TestKe3_term51_legacy_pc_style_keys_take_priority_over_text_and_ctrl_sequences(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyTab, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: "\x1b[27;5;9~"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\r")}, opts: InputKeyEncodeOptions{}, want: "\x1b[27;5;13~"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\x08")}, opts: InputKeyEncodeOptions{}, want: "\x08"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: "\x08"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: "\x1b[27;5;27~"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowLeft, Mods: InputKeyMods{Alt: true}}, opts: ke3ACAltOpts, want: "\x1b[1;3D"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp, UTF8: []uint8("x")}, opts: InputKeyEncodeOptions{}, want: "\x1b[A"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp, UTF8: []uint8("x")}, opts: InputKeyEncodeOptions{CursorKeyApplication: true}, want: "\x1bOA"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyF5, UTF8: []uint8("x")}, opts: InputKeyEncodeOptions{}, want: "\x1b[15~"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDelete, UTF8: []uint8("\x7f")}, opts: InputKeyEncodeOptions{}, want: "\x1b[3~"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyTab, UTF8: []uint8("x")}, opts: InputKeyEncodeOptions{}, want: "\t"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyTab, UTF8: []uint8("\t")}, opts: InputKeyEncodeOptions{}, want: "\t"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyTab, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("\t")}, opts: InputKeyEncodeOptions{}, want: "\x1b[Z"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyHome, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("x")}, opts: InputKeyEncodeOptions{}, want: "\x1b[1;2H"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEnter, UTF8: []uint8("\r")}, opts: InputKeyEncodeOptions{}, want: "\r"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEnter, UTF8: []uint8("\r")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOM"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeySpace, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8(" ")}, opts: InputKeyEncodeOptions{}, want: "\x00"},
	})
}

// Zig: test "term51: legacy: backspace with text"
func TestKe3_term51_legacy_backspace_with_text(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		// Control characters are ignored and the key sequence is sent.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\x7f")}, opts: InputKeyEncodeOptions{}, want: "\x7f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\x08")}, opts: InputKeyEncodeOptions{}, want: "\x7f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\x00")}, opts: InputKeyEncodeOptions{}, want: "\x7f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\x1f")}, opts: InputKeyEncodeOptions{}, want: "\x7f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("\x7f")}, opts: InputKeyEncodeOptions{}, want: "\x7f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\x08")}, opts: InputKeyEncodeOptions{}, want: "\x08"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\x7f")}, opts: ke3ACMokOpts, want: "\x1b[27;3;127~"},
		// Anything else is IME text and emits nothing at all.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8(" ")}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("a")}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("A"), UnshiftedCodepoint: 0x0D}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\xc3\xa9")}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\xed\x95\x9c")}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\x7f\x7f")}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\x01\x01")}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("a")}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("a")}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("x")}, opts: ke3ACAltOpts, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("a")}, opts: ke3ACMokOpts, want: ""},
		// 0x80 is not a control character.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("\x80")}, opts: InputKeyEncodeOptions{}, want: ""},
	})
}

// Zig: test "term51: legacy: enter with text"
func TestKe3_term51_legacy_enter_with_text(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8("\r")}, opts: InputKeyEncodeOptions{}, want: "\r"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8("\n")}, opts: InputKeyEncodeOptions{}, want: "\r"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8("\x00")}, opts: InputKeyEncodeOptions{}, want: "\r"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("\r")}, opts: InputKeyEncodeOptions{}, want: "\x1b[27;2;13~"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\r")}, opts: InputKeyEncodeOptions{}, want: "\x1b\r"},
		// Non-control text replaces the key sequence.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8("A")}, opts: InputKeyEncodeOptions{}, want: "A"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8(" ")}, opts: InputKeyEncodeOptions{}, want: " "},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8("\xe3\x81\x82")}, opts: InputKeyEncodeOptions{}, want: "\xe3\x81\x82"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8("\r\n")}, opts: InputKeyEncodeOptions{}, want: "\r\n"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8("\x80")}, opts: InputKeyEncodeOptions{}, want: "\x80"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("A")}, opts: InputKeyEncodeOptions{}, want: "A"},
		// The text then continues through ctrl sequences and alt prefixing.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("x")}, opts: InputKeyEncodeOptions{}, want: "\x18"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("x")}, opts: ke3ACAltOpts, want: "\x1bx"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("x")}, opts: ke3ACAltNoPrefix, want: "x"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("A")}, opts: ke3ACMokOpts, want: "\x1b[27;2;65~"},
	})
}

// Zig: test "term51: legacy: escape with text"
func TestKe3_term51_legacy_escape_with_text(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, UTF8: []uint8("\x1b")}, opts: InputKeyEncodeOptions{}, want: "\x1b"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, UTF8: []uint8("\x00")}, opts: InputKeyEncodeOptions{}, want: "\x1b"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, UTF8: []uint8("\x7f")}, opts: InputKeyEncodeOptions{}, want: "\x1b"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\x1b")}, opts: InputKeyEncodeOptions{}, want: "\x1b\x1b"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\x1b")}, opts: InputKeyEncodeOptions{}, want: "\x1b[27;5;27~"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, UTF8: []uint8("x")}, opts: InputKeyEncodeOptions{}, want: "x"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, UTF8: []uint8("\xc3\xa9")}, opts: InputKeyEncodeOptions{}, want: "\xc3\xa9"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("x")}, opts: InputKeyEncodeOptions{}, want: "\x18"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("x")}, opts: ke3ACAltOpts, want: "\x1bx"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("x")}, opts: ke3ACAltNoPrefix, want: "x"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("A")}, opts: ke3ACMokOpts, want: "\x1b[27;2;65~"},
	})
}

// Zig: test "term51: legacy: plain text for letters, digits, punctuation and unicode"
func TestKe3_term51_legacy_plain_text_for_letters_digits_punctuation_and_unicode(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a"), UnshiftedCodepoint: 'a'}, opts: InputKeyEncodeOptions{}, want: "a"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("A"), UnshiftedCodepoint: 'a'}, opts: InputKeyEncodeOptions{}, want: "A"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{CapsLock: true}, UTF8: []uint8("A"), UnshiftedCodepoint: 'a'}, opts: InputKeyEncodeOptions{}, want: "A"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{NumLock: true, CapsLock: true, Sides: InputKeyModsModsSide{Shift: InputKeyModsModSideRight}}, UTF8: []uint8("a")}, opts: InputKeyEncodeOptions{}, want: "a"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyZ, UTF8: []uint8("z")}, opts: InputKeyEncodeOptions{}, want: "z"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit0, UTF8: []uint8("0")}, opts: InputKeyEncodeOptions{}, want: "0"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("!"), UnshiftedCodepoint: '1'}, opts: InputKeyEncodeOptions{}, want: "!"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit9, UTF8: []uint8("9")}, opts: InputKeyEncodeOptions{}, want: "9"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeySpace, UTF8: []uint8(" ")}, opts: InputKeyEncodeOptions{}, want: " "},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeySemicolon, UTF8: []uint8(";")}, opts: InputKeyEncodeOptions{}, want: ";"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyQuote, UTF8: []uint8("'")}, opts: InputKeyEncodeOptions{}, want: "'"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyMinus, UTF8: []uint8("-")}, opts: InputKeyEncodeOptions{}, want: "-"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackslash, UTF8: []uint8("\\")}, opts: InputKeyEncodeOptions{}, want: "\\"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBracketLeft, UTF8: []uint8("[")}, opts: InputKeyEncodeOptions{}, want: "["},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackquote, UTF8: []uint8("`")}, opts: InputKeyEncodeOptions{}, want: "`"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, UTF8: []uint8("x")}, opts: InputKeyEncodeOptions{}, want: "x"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, UTF8: []uint8("\xc3\xa9"), UnshiftedCodepoint: 'e'}, opts: InputKeyEncodeOptions{}, want: "\xc3\xa9"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("\xd1\x84"), UnshiftedCodepoint: 0x444}, opts: InputKeyEncodeOptions{}, want: "\xd1\x84"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, UTF8: []uint8("\xf0\x9f\x98\x80")}, opts: InputKeyEncodeOptions{}, want: "\xf0\x9f\x98\x80"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, UTF8: []uint8("e\xcc\x81")}, opts: InputKeyEncodeOptions{}, want: "e\xcc\x81"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, UTF8: []uint8("hello")}, opts: InputKeyEncodeOptions{}, want: "hello"},
		// Control characters as text without ctrl are passed through.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("\x01")}, opts: InputKeyEncodeOptions{}, want: "\x01"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("\x00")}, opts: InputKeyEncodeOptions{}, want: "\x00"},
		// Invalid utf8 is still written verbatim.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, UTF8: []uint8("\xff")}, opts: InputKeyEncodeOptions{}, want: "\xff"},
		// Consumed mods do not change the text.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, ConsumedMods: InputKeyMods{Shift: true}, UTF8: []uint8("A"), UnshiftedCodepoint: 'a'}, opts: InputKeyEncodeOptions{}, want: "A"},
		// Plain text is not affected by the other options.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a")}, opts: InputKeyEncodeOptions{CursorKeyApplication: true, KeypadKeyApplication: true, IgnoreKeypadWithNumlock: true}, want: "a"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a")}, opts: ke3ACMokAltOpts, want: "a"},
	})
}

// Zig: test "term51: legacy: super with text"
func TestKe3_term51_legacy_super_with_text(t *testing.T) {
	// Go-only: platform modes
	for _, goos := range ke3Platforms {
		t.Run(goos, func(t *testing.T) {
			testPlatform(t, goos)
			superText := "b"
			superShift := "B"
			if ke3ACMac(goos) {
				superText = ""
				superShift = ""
			}
			ke3ACRunCases(t, []ke3ACCase{
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyB, Mods: InputKeyMods{Super: true}, UTF8: []uint8("b")}, opts: InputKeyEncodeOptions{}, want: superText},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyB, Mods: InputKeyMods{Super: true, Shift: true}, UTF8: []uint8("B")}, opts: InputKeyEncodeOptions{}, want: superShift},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyB, Mods: InputKeyMods{Super: true, Sides: InputKeyModsModsSide{Super: InputKeyModsModSideRight}}, UTF8: []uint8("b")}, opts: InputKeyEncodeOptions{}, want: superText},
				// Super with alt prefixing: the alt prefix is applied first.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyB, Mods: InputKeyMods{Super: true, Alt: true}, UTF8: []uint8("b")}, opts: ke3ACAltOpts, want: "\x1bb"},
				// Super with ctrl is encoded as CSI u regardless of the platform.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyB, Mods: InputKeyMods{Super: true, Ctrl: true}, UTF8: []uint8("b")}, opts: InputKeyEncodeOptions{}, want: "\x1b[98;5u"},
				// Super with consumed super still counts as super (all_mods).
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyB, Mods: InputKeyMods{Super: true}, ConsumedMods: InputKeyMods{Super: true}, UTF8: []uint8("b")}, opts: InputKeyEncodeOptions{}, want: superText},
				// Super with no text emits nothing.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyB, Mods: InputKeyMods{Super: true}}, opts: InputKeyEncodeOptions{}, want: ""},
				// Super with pc style keys is unaffected.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp, Mods: InputKeyMods{Super: true}}, opts: InputKeyEncodeOptions{}, want: "\x1b[1;9A"},
			})
		})
	}
}

// Zig: test "term51: legacy: ctrl sequences and alt prefix"
func TestKe3_term51_legacy_ctrl_sequences_and_alt_prefix(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{}, want: "\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, CapsLock: true, NumLock: true}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{}, want: "\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, CapsLock: true}, UTF8: []uint8("C"), UnshiftedCodepoint: 'c'}, opts: InputKeyEncodeOptions{}, want: "\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Sides: InputKeyModsModsSide{Ctrl: InputKeyModsModSideRight}}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{}, want: "\x03"},
		// Alt always adds the ESC prefix to a control sequence.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{}, want: "\x1b\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("c")}, opts: ke3ACAltOpts, want: "\x1b\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("c")}, opts: ke3ACAltNoPrefix, want: "\x1b\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{MacosOptionAsAlt: InputConfigOptionAsAltFalse}, want: "\x1b\x03"},
		// The prefix follows the effective mods: consumed alt is no prefix.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, ConsumedMods: InputKeyMods{Alt: true}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{}, want: "\x03"},
		// But consumed mods are ignored when there is no text.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, ConsumedMods: InputKeyMods{Alt: true}}, opts: InputKeyEncodeOptions{}, want: "\x1b\x03"},
		// Consuming ctrl does not stop the ctrl sequence (all mods are used).
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}, ConsumedMods: InputKeyMods{Ctrl: true}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{}, want: "\x03"},
		// Shifted punctuation.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyMinus, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("_"), UnshiftedCodepoint: '-'}, opts: InputKeyEncodeOptions{}, want: "\x1f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeySlash, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("?"), UnshiftedCodepoint: '/'}, opts: InputKeyEncodeOptions{}, want: "\x7f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit2, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("2")}, opts: InputKeyEncodeOptions{}, want: "\x00"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit8, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("8")}, opts: InputKeyEncodeOptions{}, want: "\x7f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeySpace, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8(" ")}, opts: InputKeyEncodeOptions{}, want: "\x00"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit2, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("@"), UnshiftedCodepoint: '2'}, opts: InputKeyEncodeOptions{}, want: "\x1b[64;5u"},
		// Letters that fixterms reserves for CSI u.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("i")}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyM, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("m")}, opts: InputKeyEncodeOptions{}, want: "\x1b[109;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBracketLeft, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("[")}, opts: InputKeyEncodeOptions{}, want: "\x1b[91;5u"},
		// Ctrl text that maps to nothing is CSI u too.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("i")}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;5u"},
		// Ctrl with super is CSI u, super is not reported.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Super: true}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{}, want: "\x1b[99;5u"},
		// Key based ctrl sequences (no text).
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: "\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}}, opts: InputKeyEncodeOptions{}, want: "\x1b\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Shift: true}}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Super: true}}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit2, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: "\x00"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyM, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBracketLeft, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBracketRight, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: "\x1d"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackslash, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: "\x1c"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeySlash, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: "\x1f"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyMinus, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Ctrl: true, Alt: true}, UnshiftedCodepoint: 'e'}, opts: InputKeyEncodeOptions{}, want: "\x1b\x05"},
		// Cyrillic layouts: the logical key is used with exactly ctrl.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\xd1\x81"), UnshiftedCodepoint: 0x441}, opts: InputKeyEncodeOptions{}, want: "\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("\xd1\x81"), UnshiftedCodepoint: 0x441}, opts: InputKeyEncodeOptions{}, want: "\x1b\x03"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("\xd0\xa1"), UnshiftedCodepoint: 0x441}, opts: InputKeyEncodeOptions{}, want: "\x1b[1057;5u"},
		// Ctrl text with more than one codepoint is written as is.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("ab")}, opts: InputKeyEncodeOptions{}, want: "ab"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\xff")}, opts: InputKeyEncodeOptions{}, want: "\xff"},
	})
}

// Zig: test "term51: legacy: alt prefix without text uses the unshifted codepoint"
func TestKe3_term51_legacy_alt_prefix_without_text_uses_the_unshifted_codepoint(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: "\x1be"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true, Shift: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: "\x1be"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true, Super: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: "\x1be"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true, CapsLock: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideRight}}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: "\x1be"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, ConsumedMods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: "\x1be"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 0xE9}, opts: ke3ACAltOpts, want: "\x1b\xe9"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 0xFF}, opts: ke3ACAltOpts, want: "\x1b\xff"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 0x01}, opts: ke3ACAltOpts, want: "\x1b\x01"},
		// No prefix when disabled, without alt, or without a usable codepoint.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltNoPrefix, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'e'}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}}, opts: ke3ACAltOpts, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 0x100}, opts: ke3ACAltOpts, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 0x65 + 0x100}, opts: ke3ACAltOpts, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 0x10FFFF}, opts: ke3ACAltOpts, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Shift: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Super: true, Ctrl: false}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: ""},
		// Ctrl without text and alt: key based ctrl sequence with ESC.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true, Ctrl: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: "\x1b\x05"},
		// Ctrl+shift without text: no ctrl sequence and no CSI u (needs text);
		// only the alt prefix remains.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Ctrl: true, Shift: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Ctrl: true, Shift: true, Alt: true}, UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: "\x1be"},
	})
}

// Zig: test "term51: legacy: alt prefix with text"
func TestKe3_term51_legacy_alt_prefix_with_text(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		// Single byte text.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("c")}, opts: ke3ACAltOpts, want: "\x1bc"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true, Shift: true}, UTF8: []uint8("C"), UnshiftedCodepoint: 'c'}, opts: ke3ACAltOpts, want: "\x1bC"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("c")}, opts: ke3ACAltNoPrefix, want: "c"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("c")}, opts: InputKeyEncodeOptions{}, want: "c"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyPeriod, Mods: InputKeyMods{Alt: true, Shift: true}, UTF8: []uint8(">"), UnshiftedCodepoint: '.'}, opts: ke3ACAltOpts, want: "\x1b>"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xff")}, opts: ke3ACAltOpts, want: "\x1b\xff"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\x00")}, opts: ke3ACAltOpts, want: "\x1b\x00"},
		// Multi byte text uses the unshifted codepoint instead.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xc3\xa9"), UnshiftedCodepoint: 'e'}, opts: ke3ACAltOpts, want: "\x1be"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xc3\xa9")}, opts: ke3ACAltOpts, want: "\xc3\xa9"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xc3\xa9"), UnshiftedCodepoint: 0x100}, opts: ke3ACAltOpts, want: "\xc3\xa9"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xc3\xa9"), UnshiftedCodepoint: 'e'}, opts: ke3ACAltNoPrefix, want: "\xc3\xa9"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("ab"), UnshiftedCodepoint: 'a'}, opts: ke3ACAltOpts, want: "\x1ba"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("ab")}, opts: ke3ACAltOpts, want: "ab"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xf0\x9f\x98\x80"), UnshiftedCodepoint: 0x1F600}, opts: ke3ACAltOpts, want: "\xf0\x9f\x98\x80"},
		// Consumed alt (it produced the text) is not a prefix.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true}, ConsumedMods: InputKeyMods{Alt: true}, UTF8: []uint8("\xc3\xa7"), UnshiftedCodepoint: 'c'}, opts: ke3ACAltOpts, want: "\xc3\xa7"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true}, ConsumedMods: InputKeyMods{Alt: true}, UTF8: []uint8("c")}, opts: ke3ACAltOpts, want: "c"},
		// Consuming shift only leaves alt in the effective mods.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true, Shift: true}, ConsumedMods: InputKeyMods{Shift: true}, UTF8: []uint8("C"), UnshiftedCodepoint: 'c'}, opts: ke3ACAltOpts, want: "\x1bC"},
		// Alt with other modifiers.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true, Super: true}, UTF8: []uint8("c")}, opts: ke3ACAltOpts, want: "\x1bc"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true, CapsLock: true, NumLock: true}, UTF8: []uint8("C")}, opts: ke3ACAltOpts, want: "\x1bC"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideRight}}, UTF8: []uint8("c")}, opts: ke3ACAltOpts, want: "\x1bc"},
	})
}

// Zig: test "term51: legacy: modify other keys state 2 and macos option"
func TestKe3_term51_legacy_modify_other_keys_state_2_and_macos_option(t *testing.T) {
	// Go-only: platform modes
	for _, goos := range ke3Platforms {
		t.Run(goos, func(t *testing.T) {
			testPlatform(t, goos)
			const M = "\x1b[27;3;49~"
			// Alt held, text "1", alt_esc_prefix on: the CSI 27 sequence on Linux. On macOS
			// the option setting decides between the CSI 27 sequence and the text. On iOS
			// (isDarwin() true, `.macos` false) key translation drops alt unless it is set
			// as option-as-alt, so modify other keys does not apply, but the alt prefix
			// (which only checks `.macos`) still does: ESC and the text.
			left := InputKeyMods{Alt: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideLeft}}
			right := InputKeyMods{Alt: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideRight}}
			mac := [4][2]string{
				{"1", "1"},
				{M, M},
				{M, "1"},
				{"1", M},
			}
			ios := [4][2]string{
				{"\x1b1", "\x1b1"},
				{M, M},
				{M, "\x1b1"},
				{"\x1b1", M},
			}
			opts := []InputConfigOptionAsAlt{InputConfigOptionAsAltFalse, InputConfigOptionAsAltTrue, InputConfigOptionAsAltLeft, InputConfigOptionAsAltRight}
			for i, opt := range opts {
				o := InputKeyEncodeOptions{ModifyOtherKeysState2: true, AltEscPrefix: true, MacosOptionAsAlt: opt}
				wantLeft, wantRight := M, M
				if ke3ACMac(goos) {
					wantLeft, wantRight = mac[i][0], mac[i][1]
				} else if ke3ACDarwin(goos) {
					wantLeft, wantRight = ios[i][0], ios[i][1]
				}
				ke3ACExpect(t, wantLeft, InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: left, UTF8: []uint8("1")}, o)
				ke3ACExpect(t, wantRight, InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: right, UTF8: []uint8("1")}, o)
			}
		})
	}
}

// Zig: test "term51: legacy: modify other keys state 2 conditions"
func TestKe3_term51_legacy_modify_other_keys_state_2_conditions(t *testing.T) {
	// Go-only: platform modes
	for _, goos := range ke3Platforms {
		t.Run(goos, func(t *testing.T) {
			testPlatform(t, goos)
			// Zig: `if (term51ACMac) "" else text`: macOS never encodes text with super.
			superUnlessMac := func(text string) string {
				if ke3ACMac(goos) {
					return ""
				}
				return text
			}
			ke3ACRunCases(t, []ke3ACCase{
				// Control input range 0x40..0x7F with a modifier.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("A"), UnshiftedCodepoint: 'a'}, opts: ke3ACMokOpts, want: "\x1b[27;2;65~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Super: true}, UTF8: []uint8("a")}, opts: ke3ACMokOpts, want: "\x1b[27;9;97~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("@")}, opts: ke3ACMokOpts, want: "\x1b[27;2;64~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("~")}, opts: ke3ACMokOpts, want: "\x1b[27;2;126~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("\x7f")}, opts: ke3ACMokOpts, want: "\x1b[27;2;127~"},
				// Just below the range with only shift: not encoded.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("?")}, opts: ke3ACMokOpts, want: "?"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("!")}, opts: ke3ACMokOpts, want: "!"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("\x01")}, opts: ke3ACMokOpts, want: "\x01"},
				// Shift with space is allowed.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeySpace, Mods: InputKeyMods{Shift: true}, UTF8: []uint8(" ")}, opts: ke3ACMokOpts, want: "\x1b[27;2;32~"},
				// Anything other than shift is encoded, in or out of the range.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("1")}, opts: ke3ACMokOpts, want: "\x1b[27;3;49~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: InputKeyMods{Super: true}, UTF8: []uint8("1")}, opts: ke3ACMokOpts, want: "\x1b[27;9;49~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("!"), UnshiftedCodepoint: '1'}, opts: ke3ACMokOpts, want: "\x1b[27;6;33~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: InputKeyMods{Alt: true, Shift: true}, UTF8: []uint8("!"), UnshiftedCodepoint: '1'}, opts: ke3ACMokOpts, want: "\x1b[27;4;33~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\x01")}, opts: ke3ACMokOpts, want: "\x1b[27;3;1~"},
				// No modifiers: nothing matches, text is written.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a")}, opts: ke3ACMokOpts, want: "a"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("~")}, opts: ke3ACMokOpts, want: "~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, UTF8: []uint8("1")}, opts: ke3ACMokOpts, want: "1"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{CapsLock: true, NumLock: true}, UTF8: []uint8("A")}, opts: ke3ACMokOpts, want: "A"},
				// Ctrl sequences win over modify other keys.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("1")}, opts: ke3ACMokOpts, want: "1"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("c")}, opts: ke3ACMokOpts, want: "\x03"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("c")}, opts: ke3ACMokOpts, want: "\x1b\x03"},
				// Ctrl on a letter that is not a ctrl sequence: CSI 27 instead of CSI u.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("i")}, opts: ke3ACMokOpts, want: "\x1b[27;5;105~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("i")}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;5u"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBracketLeft, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("[")}, opts: ke3ACMokOpts, want: "\x1b[27;5;91~"},
				// The mods are the event mods, not reduced by consumed mods.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyH, Mods: InputKeyMods{Ctrl: true, Shift: true}, ConsumedMods: InputKeyMods{Shift: true}, UTF8: []uint8("H")}, opts: ke3ACMokOpts, want: "\x1b[27;6;72~"},
				// Multi byte codepoints are encoded with their codepoint value.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xc3\xa9")}, opts: ke3ACMokOpts, want: "\x1b[27;3;233~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\xf0\x9f\x98\x80")}, opts: ke3ACMokOpts, want: "\x1b[27;5;128512~"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("\xc3\x89")}, opts: ke3ACMokOpts, want: "\xc3\x89"},
				// More than one codepoint: not encoded, text is written (alt prefix
				// only when a byte is available).
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("ab")}, opts: ke3ACMokAltOpts, want: "ab"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("ab"), UnshiftedCodepoint: 'a'}, opts: ke3ACMokAltOpts, want: "\x1ba"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("ab")}, opts: ke3ACMokOpts, want: "ab"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Super: true}, UTF8: []uint8("e\xcc\x81")}, opts: ke3ACMokOpts, want: superUnlessMac("e\xcc\x81")},
				// Invalid utf8: not encoded.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Super: true}, UTF8: []uint8("\xff")}, opts: ke3ACMokOpts, want: superUnlessMac("\xff")},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xff")}, opts: ke3ACMokAltOpts, want: "\x1b\xff"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\xff")}, opts: ke3ACMokOpts, want: "\xff"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xc3")}, opts: ke3ACMokAltOpts, want: "\x1b\xc3"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xf0\x9f\x98")}, opts: ke3ACMokAltOpts, want: "\xf0\x9f\x98"},
				// Without the mode the same events are not CSI 27.
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("A")}, opts: InputKeyEncodeOptions{}, want: "A"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("1")}, opts: ke3ACAltNoPrefix, want: "1"},
				{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit1, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("1")}, opts: ke3ACAltOpts, want: "\x1b1"},
			})
		})
	}
}

// ke3ACModLetterOn is term51ACModLetterOn.
var ke3ACModLetterOn = [16]string{
	"a",
	"\x1b[27;2;97~",
	"\x01",
	"\x01",
	"\x1b[27;3;97~",
	"\x1b[27;4;97~",
	"\x1b\x01",
	"\x1b\x01",
	"\x1b[27;9;97~",
	"\x1b[27;10;97~",
	"\x1b[27;13;97~",
	"\x1b[27;14;97~",
	"\x1b[27;11;97~",
	"\x1b[27;12;97~",
	"\x1b[27;15;97~",
	"\x1b[27;16;97~",
}

// Zig: test "term51: legacy: modify other keys state 2 for every modifier combination"
func TestKe3_term51_legacy_modify_other_keys_state_2_for_every_modifier_combination(t *testing.T) {
	testPlatform(t, "linux")
	for bits := 0; bits < 16; bits++ {
		ke3ACExpect(
			t,
			ke3ACModLetterOn[bits],
			InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: ke3ACMods(bits), UTF8: []uint8("a"), UnshiftedCodepoint: 'a'},
			ke3ACMokOpts,
		)
		ke3ACExpect(
			t,
			ke3ACModLetterOn[bits],
			InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: ke3ACMods(bits), UTF8: []uint8("a"), UnshiftedCodepoint: 'a'},
			ke3ACMokAltOpts,
		)
	}
}

// Zig: test "term51: legacy: letter with every modifier combination without modify other keys"
func TestKe3_term51_legacy_letter_with_every_modifier_combination_without_modify_other_keys(t *testing.T) {
	// Go-only: platform modes
	for _, goos := range ke3Platforms {
		t.Run(goos, func(t *testing.T) {
			testPlatform(t, goos)
			st := "a"
			if ke3ACMac(goos) {
				st = ""
			}
			off := [16]string{
				"a",
				"a",
				"\x01",
				"\x01",
				"a",
				"a",
				"\x1b\x01",
				"\x1b\x01",
				st,
				st,
				"\x1b[97;5u",
				"\x1b[97;6u",
				st,
				st,
				"\x1b[97;7u",
				"\x1b[97;8u",
			}
			prefixed := [16]string{
				"a",
				"a",
				"\x01",
				"\x01",
				"\x1ba",
				"\x1ba",
				"\x1b\x01",
				"\x1b\x01",
				st,
				st,
				"\x1b[97;5u",
				"\x1b[97;6u",
				"\x1ba",
				"\x1ba",
				"\x1b[97;7u",
				"\x1b[97;8u",
			}
			for bits := 0; bits < 16; bits++ {
				ev := InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: ke3ACMods(bits), UTF8: []uint8("a"), UnshiftedCodepoint: 'a'}
				ke3ACExpect(t, off[bits], ev, ke3ACAltNoPrefix)
				ke3ACExpect(t, prefixed[bits], ev, ke3ACAltOpts)
			}
		})
	}
}

// Zig: test "term51: legacy: ctrl CSI u fallback"
func TestKe3_term51_legacy_ctrl_CSI_u_fallback(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("i"), UnshiftedCodepoint: 'i'}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("i"), UnshiftedCodepoint: 'i'}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;7u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true, Alt: true}, ConsumedMods: InputKeyMods{Alt: true}, UTF8: []uint8("i"), UnshiftedCodepoint: 'i'}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;7u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true, CapsLock: true, NumLock: true, Sides: InputKeyModsModsSide{Ctrl: InputKeyModsModSideRight}}, UTF8: []uint8("i")}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyM, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("m")}, opts: InputKeyEncodeOptions{}, want: "\x1b[109;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBracketLeft, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("[")}, opts: InputKeyEncodeOptions{}, want: "\x1b[91;5u"},
		// Shifted uppercase letters: lowercased, shift kept when the
		// unshifted codepoint matches the lowercase letter.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("A"), UnshiftedCodepoint: 'a'}, opts: InputKeyEncodeOptions{}, want: "\x1b[97;6u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("I"), UnshiftedCodepoint: 'i'}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;6u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyZ, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("Z"), UnshiftedCodepoint: 'z'}, opts: InputKeyEncodeOptions{}, want: "\x1b[122;6u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Ctrl: true, Shift: true, Alt: true}, UTF8: []uint8("A"), UnshiftedCodepoint: 'a'}, opts: InputKeyEncodeOptions{}, want: "\x1b[97;8u"},
		// Without an unshifted codepoint shift is dropped.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("I")}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;5u"},
		// Shift used to produce the character is not reported.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeySemicolon, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8(":"), UnshiftedCodepoint: ';'}, opts: InputKeyEncodeOptions{}, want: "\x1b[58;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyDigit2, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("@"), UnshiftedCodepoint: '2'}, opts: InputKeyEncodeOptions{}, want: "\x1b[64;5u"},
		// Lowercase text with shift reported when it matches the unshifted codepoint.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("i"), UnshiftedCodepoint: 'i'}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;6u"},
		// Super is not part of CSI u.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true, Super: true}, UTF8: []uint8("i")}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true, Super: true, Shift: true, Alt: true}, UTF8: []uint8("i"), UnshiftedCodepoint: 'i'}, opts: InputKeyEncodeOptions{}, want: "\x1b[105;8u"},
		// Non-ASCII codepoints use their value.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\xc3\xa9"), UnshiftedCodepoint: 0xE9}, opts: InputKeyEncodeOptions{}, want: "\x1b[233;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\xf0\x9f\x98\x80")}, opts: InputKeyEncodeOptions{}, want: "\x1b[128512;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("\xc3\x89"), UnshiftedCodepoint: 0xE9}, opts: InputKeyEncodeOptions{}, want: "\x1b[201;5u"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true, Shift: true}, UTF8: []uint8("\xc3\x89"), UnshiftedCodepoint: 0xC9}, opts: InputKeyEncodeOptions{}, want: "\x1b[201;6u"},
		// Not a single valid codepoint: text is written without CSI u.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("ab")}, opts: InputKeyEncodeOptions{}, want: "ab"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("e\xcc\x81")}, opts: InputKeyEncodeOptions{}, want: "e\xcc\x81"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\xff")}, opts: InputKeyEncodeOptions{}, want: "\xff"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\xc3")}, opts: InputKeyEncodeOptions{}, want: "\xc3"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("ab"), UnshiftedCodepoint: 'a'}, opts: ke3ACAltOpts, want: "\x1ba"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("ab")}, opts: ke3ACAltOpts, want: "ab"},
	})
}

// Zig: test "term51: legacy: modifier keys alone produce nothing"
func TestKe3_term51_legacy_modifier_keys_alone_produce_nothing(t *testing.T) {
	testPlatform(t, "linux")
	modkeys := []InputKeyKey{InputKeyKeyShiftLeft, InputKeyKeyShiftRight, InputKeyKeyControlLeft, InputKeyKeyControlRight, InputKeyKeyAltLeft, InputKeyKeyAltRight, InputKeyKeyMetaLeft, InputKeyKeyMetaRight, InputKeyKeyCapsLock, InputKeyKeyNumLock, InputKeyKeyFn, InputKeyKeyFnLock}
	allOpts := []InputKeyEncodeOptions{
		{},
		ke3ACAltOpts,
		ke3ACMokOpts,
		ke3ACMokAltOpts,
		{CursorKeyApplication: true, KeypadKeyApplication: true},
	}
	for _, k := range modkeys {
		for bits := 0; bits < 16; bits++ {
			for _, o := range allOpts {
				ke3ACExpect(t, "", InputKeyKeyEvent{Action: InputKeyActionPress, Key: k, Mods: ke3ACMods(bits)}, o)
				ke3ACExpect(t, "", InputKeyKeyEvent{Action: InputKeyActionPress, Key: k, Mods: InputKeyMods{CapsLock: true, NumLock: true, Sides: InputKeyModsModsSideMask.Sides}}, o)
			}
		}
	}
}

// Zig: test "term51: legacy: keys with no encoding produce nothing without text"
func TestKe3_term51_legacy_keys_with_no_encoding_produce_nothing_without_text(t *testing.T) {
	testPlatform(t, "linux")
	keys := []InputKeyKey{InputKeyKeyUnidentified, InputKeyKeyKeyA, InputKeyKeyDigit1, InputKeyKeySpace, InputKeyKeyF13, InputKeyKeyF14, InputKeyKeyF20, InputKeyKeyF25, InputKeyKeyContextMenu, InputKeyKeyPrintScreen, InputKeyKeyScrollLock, InputKeyKeyPause, InputKeyKeyNumpadEqual, InputKeyKeyNumpadComma, InputKeyKeyNumpadClear, InputKeyKeyHelp, InputKeyKeyConvert, InputKeyKeyPower, InputKeyKeyIntlRo, InputKeyKeyMediaStop}
	opts := []InputKeyEncodeOptions{{}, ke3ACMokAltOpts, {CursorKeyApplication: true, KeypadKeyApplication: true}}
	for _, k := range keys {
		for _, bits := range []int{0, 1, 4, 8, 5, 12} {
			for _, o := range opts {
				ke3ACExpect(t, "", InputKeyKeyEvent{Action: InputKeyActionPress, Key: k, Mods: ke3ACMods(bits)}, o)
			}
		}
	}
}

// Zig: test "term51: legacy: function keys F1 to F25"
func TestKe3_term51_legacy_function_keys_F1_to_F25(t *testing.T) {
	testPlatform(t, "linux")
	type Fk struct {
		x0 InputKeyKey
		x1 string
		x2 string
		x3 string
	}
	fkeys := []Fk{
		{InputKeyKeyF1, "\x1bOP", "\x1b[1;2P", "\x1b[1;5P"},
		{InputKeyKeyF2, "\x1bOQ", "\x1b[1;2Q", "\x1b[1;5Q"},
		{InputKeyKeyF3, "\x1bOR", "\x1b[13;2~", "\x1b[13;5~"},
		{InputKeyKeyF4, "\x1bOS", "\x1b[1;2S", "\x1b[1;5S"},
		{InputKeyKeyF5, "\x1b[15~", "\x1b[15;2~", "\x1b[15;5~"},
		{InputKeyKeyF6, "\x1b[17~", "\x1b[17;2~", "\x1b[17;5~"},
		{InputKeyKeyF7, "\x1b[18~", "\x1b[18;2~", "\x1b[18;5~"},
		{InputKeyKeyF8, "\x1b[19~", "\x1b[19;2~", "\x1b[19;5~"},
		{InputKeyKeyF9, "\x1b[20~", "\x1b[20;2~", "\x1b[20;5~"},
		{InputKeyKeyF10, "\x1b[21~", "\x1b[21;2~", "\x1b[21;5~"},
		{InputKeyKeyF11, "\x1b[23~", "\x1b[23;2~", "\x1b[23;5~"},
		{InputKeyKeyF12, "\x1b[24~", "\x1b[24;2~", "\x1b[24;5~"},
	}
	for _, f := range fkeys {
		for _, o := range []InputKeyEncodeOptions{{}, {CursorKeyApplication: true}, {KeypadKeyApplication: true}, ke3ACMokAltOpts} {
			ke3ACExpect(t, f.x1, ke3Key(f.x0), o)
			ke3ACExpect(t, f.x2, InputKeyKeyEvent{Action: InputKeyActionPress, Key: f.x0, Mods: InputKeyMods{Shift: true}}, o)
			ke3ACExpect(t, f.x3, InputKeyKeyEvent{Action: InputKeyActionPress, Key: f.x0, Mods: InputKeyMods{Ctrl: true}}, o)
			// Repeat and text do not change it.
			ke3ACExpect(t, f.x1, InputKeyKeyEvent{Action: InputKeyActionRepeat, Key: f.x0, UTF8: []uint8("x")}, o)
		}
	}

	// F13 to F25 have no encoding at all in the legacy protocol.
	high := []InputKeyKey{InputKeyKeyF13, InputKeyKeyF14, InputKeyKeyF15, InputKeyKeyF16, InputKeyKeyF17, InputKeyKeyF18, InputKeyKeyF19, InputKeyKeyF20, InputKeyKeyF21, InputKeyKeyF22, InputKeyKeyF23, InputKeyKeyF24, InputKeyKeyF25}
	for _, k := range high {
		for bits := 0; bits < 16; bits++ {
			ke3ACExpect(t, "", InputKeyKeyEvent{Action: InputKeyActionPress, Key: k, Mods: ke3ACMods(bits)}, InputKeyEncodeOptions{})
		}
	}
}

// Zig: test "term51: legacy: navigation keys and cursor key application mode"
func TestKe3_term51_legacy_navigation_keys_and_cursor_key_application_mode(t *testing.T) {
	testPlatform(t, "linux")
	type Nk struct {
		x0 InputKeyKey
		x1 string
		x2 string
		x3 string
	}
	nkeys := []Nk{
		{InputKeyKeyArrowUp, "\x1b[A", "\x1bOA", "\x1b[1;2A"},
		{InputKeyKeyArrowDown, "\x1b[B", "\x1bOB", "\x1b[1;2B"},
		{InputKeyKeyArrowRight, "\x1b[C", "\x1bOC", "\x1b[1;2C"},
		{InputKeyKeyArrowLeft, "\x1b[D", "\x1bOD", "\x1b[1;2D"},
		{InputKeyKeyHome, "\x1b[H", "\x1bOH", "\x1b[1;2H"},
		{InputKeyKeyEnd, "\x1b[F", "\x1bOF", "\x1b[1;2F"},
		{InputKeyKeyNumpadUp, "\x1b[A", "\x1bOA", "\x1b[1;2A"},
		{InputKeyKeyNumpadDown, "\x1b[B", "\x1bOB", "\x1b[1;2B"},
		{InputKeyKeyNumpadRight, "\x1b[C", "\x1bOC", "\x1b[1;2C"},
		{InputKeyKeyNumpadLeft, "\x1b[D", "\x1bOD", "\x1b[1;2D"},
		{InputKeyKeyNumpadBegin, "\x1b[E", "\x1bOE", "\x1b[1;2E"},
		{InputKeyKeyNumpadHome, "\x1b[H", "\x1bOH", "\x1b[1;2H"},
		{InputKeyKeyNumpadEnd, "\x1b[F", "\x1bOF", "\x1b[1;2F"},
		{InputKeyKeyInsert, "\x1b[2~", "\x1b[2~", "\x1b[2;2~"},
		{InputKeyKeyDelete, "\x1b[3~", "\x1b[3~", "\x1b[3;2~"},
		{InputKeyKeyPageUp, "\x1b[5~", "\x1b[5~", "\x1b[5;2~"},
		{InputKeyKeyPageDown, "\x1b[6~", "\x1b[6~", "\x1b[6;2~"},
		{InputKeyKeyNumpadInsert, "\x1b[2~", "\x1b[2~", "\x1b[2;2~"},
		{InputKeyKeyNumpadDelete, "\x1b[3~", "\x1b[3~", "\x1b[3;2~"},
		{InputKeyKeyNumpadPageUp, "\x1b[5~", "\x1b[5~", "\x1b[5;2~"},
		{InputKeyKeyNumpadPageDown, "\x1b[6~", "\x1b[6~", "\x1b[6;2~"},
	}
	for _, n := range nkeys {
		ke3ACExpect(t, n.x1, ke3Key(n.x0), InputKeyEncodeOptions{})
		ke3ACExpect(t, n.x2, ke3Key(n.x0), InputKeyEncodeOptions{CursorKeyApplication: true})
		ke3ACExpect(t, n.x3, InputKeyKeyEvent{Action: InputKeyActionPress, Key: n.x0, Mods: InputKeyMods{Shift: true}}, InputKeyEncodeOptions{})
		ke3ACExpect(t, n.x3, InputKeyKeyEvent{Action: InputKeyActionPress, Key: n.x0, Mods: InputKeyMods{Shift: true}}, InputKeyEncodeOptions{CursorKeyApplication: true})
		ke3ACExpect(t, n.x1, InputKeyKeyEvent{Action: InputKeyActionPress, Key: n.x0, Mods: InputKeyMods{CapsLock: true, NumLock: true}}, InputKeyEncodeOptions{})
		ke3ACExpect(t, n.x2, InputKeyKeyEvent{Action: InputKeyActionRepeat, Key: n.x0}, InputKeyEncodeOptions{CursorKeyApplication: true, KeypadKeyApplication: true})
	}
}

// Zig: test "term51: legacy: keypad keys"
func TestKe3_term51_legacy_keypad_keys(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACRunCases(t, []ke3ACCase{
		// Numerical mode: text is written; no text means nothing.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{}, want: "1"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1}, opts: InputKeyEncodeOptions{}, want: ""},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, Mods: InputKeyMods{NumLock: true}, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{}, want: "1"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadAdd, UTF8: []uint8("+")}, opts: InputKeyEncodeOptions{}, want: "+"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEnter}, opts: InputKeyEncodeOptions{}, want: "\r"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEnter, Mods: InputKeyMods{Shift: true}}, opts: InputKeyEncodeOptions{}, want: "\r"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEqual, UTF8: []uint8("=")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "="},
		// Application mode.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad0, UTF8: []uint8("0")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOp"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOq"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad9, UTF8: []uint8("9")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOy"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadDecimal, UTF8: []uint8(".")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOn"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadDivide, UTF8: []uint8("/")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOo"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadMultiply, UTF8: []uint8("*")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOj"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadSubtract, UTF8: []uint8("-")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOm"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadAdd, UTF8: []uint8("+")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOk"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEnter}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOM"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, Mods: InputKeyMods{NumLock: true}, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bOq"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, Mods: InputKeyMods{Shift: true}}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bO2q"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bO5q"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEnter, Mods: InputKeyMods{Alt: true}}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1bO3M"},
		// Numlock ignored (mode 1035) forces numerical mode.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true, IgnoreKeypadWithNumlock: true}, want: "1"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadEnter}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true, IgnoreKeypadWithNumlock: true}, want: "\r"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad1, UTF8: []uint8("1")}, opts: InputKeyEncodeOptions{IgnoreKeypadWithNumlock: true}, want: "1"},
		// Ctrl on keypad digits: text based ctrl sequences are not used for
		// keys with pc style entries, only the sequence in application mode.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad2, Mods: InputKeyMods{Ctrl: true}}, opts: InputKeyEncodeOptions{}, want: "\x00"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpad2, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("2")}, opts: InputKeyEncodeOptions{}, want: "\x00"},
		// The other keypad navigation keys follow cursor mode.
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadUp}, opts: InputKeyEncodeOptions{CursorKeyApplication: true}, want: "\x1bOA"},
		{ev: InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyNumpadUp}, opts: InputKeyEncodeOptions{KeypadKeyApplication: true}, want: "\x1b[A"},
	})
}

// Zig: test "term51: legacy: output too small for each write point"
func TestKe3_term51_legacy_output_too_small_for_each_write_point(t *testing.T) {
	testPlatform(t, "linux")
	ke3ACExpectFit(t, "\x1b[1;2A", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyArrowUp, Mods: InputKeyMods{Shift: true}}, InputKeyEncodeOptions{})
	ke3ACExpectFit(t, "\x1bOP", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyF1}, InputKeyEncodeOptions{})
	ke3ACExpectFit(t, "\x7f", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace}, InputKeyEncodeOptions{})
	ke3ACExpectFit(t, "\x1b[27;3;13~", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, Mods: InputKeyMods{Alt: true}}, ke3ACMokOpts)
	// ctrl sequence: one byte.
	ke3ACExpectFit(t, "\x03", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("c")}, InputKeyEncodeOptions{})
	// ctrl sequence with alt: ESC then the byte.
	ke3ACExpectFit(t, "\x1b\x03", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("c")}, InputKeyEncodeOptions{})
	// alt prefix from unshifted codepoint (formatted) and from text.
	ke3ACExpectFit(t, "\x1be", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'e'}, ke3ACAltOpts)
	ke3ACExpectFit(t, "\x1bc", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("c")}, ke3ACAltOpts)
	ke3ACExpectFit(t, "\x1b\xe9", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 0xE9}, ke3ACAltOpts)
	// modify other keys.
	ke3ACExpectFit(t, "\x1b[27;2;65~", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Shift: true}, UTF8: []uint8("A")}, ke3ACMokOpts)
	ke3ACExpectFit(t, "\x1b[27;5;128512~", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("\xf0\x9f\x98\x80")}, ke3ACMokOpts)
	// CSI u.
	ke3ACExpectFit(t, "\x1b[105;5u", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyI, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("i")}, InputKeyEncodeOptions{})
	ke3ACExpectFit(t, "\x1b[128512;7u", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, Mods: InputKeyMods{Ctrl: true, Alt: true}, UTF8: []uint8("\xf0\x9f\x98\x80")}, InputKeyEncodeOptions{})
	// Plain text.
	ke3ACExpectFit(t, "a", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a")}, InputKeyEncodeOptions{})
	ke3ACExpectFit(t, "\xc3\xa9", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, UTF8: []uint8("\xc3\xa9")}, InputKeyEncodeOptions{})
	ke3ACExpectFit(t, "\xf0\x9f\x98\x80", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, UTF8: []uint8("\xf0\x9f\x98\x80")}, InputKeyEncodeOptions{})
	ke3ACExpectFit(t, "hello", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyUnidentified, UTF8: []uint8("hello")}, InputKeyEncodeOptions{})
	ke3ACExpectFit(t, "\xe3\x81\x82", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEnter, UTF8: []uint8("\xe3\x81\x82")}, InputKeyEncodeOptions{})
}

// Zig: test "term51: legacy: ctrl sequence with alt writes the ESC before failing"
func TestKe3_term51_legacy_ctrl_sequence_with_alt_writes_the_ESC_before_failing(t *testing.T) {
	testPlatform(t, "linux")
	var buf [1]uint8
	w := oscWriterFixed(buf[:])
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeLegacy(&w, InputKeyKeyEvent{
		Action: InputKeyActionPress,
		Key:    InputKeyKeyKeyC,
		Mods:   InputKeyMods{Ctrl: true, Alt: true},
		UTF8:   []uint8("c"),
	}, InputKeyEncodeOptions{}))
	expectEqualStrings(t, "\x1b", string(w.buffered()))

	w0 := oscWriterFixed(buf[0:0:0])
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeLegacy(&w0, InputKeyKeyEvent{
		Action: InputKeyActionPress,
		Key:    InputKeyKeyKeyC,
		Mods:   InputKeyMods{Ctrl: true, Alt: true},
		UTF8:   []uint8("c"),
	}, InputKeyEncodeOptions{}))
	expectEqualStrings(t, "", string(w0.buffered()))

	// A one byte ctrl sequence fits exactly in a one byte buffer.
	w1 := oscWriterFixed(buf[:])
	zigTry(t, inputKeyEncodeLegacy(&w1, InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}, UTF8: []uint8("c")}, InputKeyEncodeOptions{}))
	expectEqualStrings(t, "\x03", string(w1.buffered()))
}

// Zig: test "term51: legacy: empty buffer works when there is no output"
func TestKe3_term51_legacy_empty_buffer_works_when_there_is_no_output(t *testing.T) {
	testPlatform(t, "linux")
	var empty [0]uint8
	events := []InputKeyKeyEvent{
		InputKeyKeyEvent{Action: InputKeyActionRelease, Key: InputKeyKeyKeyA, UTF8: []uint8("a")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a"), Composing: true},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyShiftLeft, Mods: InputKeyMods{Shift: true}},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyF13},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyBackspace, UTF8: []uint8("a")},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyE, Mods: InputKeyMods{Alt: true}, UnshiftedCodepoint: 'e'},
		InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true, Shift: true}},
	}
	for _, ev := range events {
		w := oscWriterFixed(empty[:])
		zigTry(t, inputKeyEncodeLegacy(&w, ev, InputKeyEncodeOptions{}))
		expectEqual(t, 0, len(w.buffered()))
		w2 := oscWriterFixed(empty[:])
		zigTry(t, inputKeyEncodeEncode(&w2, ev, InputKeyEncodeOptions{}))
		expectEqual(t, 0, len(w2.buffered()))
	}

	// Anything that produces a byte fails on an empty buffer.
	w := oscWriterFixed(empty[:])
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeLegacy(&w, InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, UTF8: []uint8("a")}, InputKeyEncodeOptions{}))
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeLegacy(&w, ke3Key(InputKeyKeyEnter), InputKeyEncodeOptions{}))
	ke3ExpectError(t, ErrWriteFailed, inputKeyEncodeLegacy(&w, InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyC, Mods: InputKeyMods{Ctrl: true}}, InputKeyEncodeOptions{}))
}

// Zig: test "term51: legacy: macos super and option behaviour"
// Zig skips this test unless term51ACMac (`builtin.os.tag == .macos`):
// "darwin" only.
func TestKe3_term51_legacy_macos_super_and_option_behaviour(t *testing.T) {
	testPlatform(t, "darwin")
	// Option as alt = false on macOS: no alt prefix, text is kept.
	ke3ACExpect(t, "\xc3\xa5", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Alt: true}, UTF8: []uint8("\xc3\xa5"), UnshiftedCodepoint: 'a'}, InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: InputConfigOptionAsAltFalse})
	// Left only: the right alt translates, the left one prefixes.
	ke3ACExpect(t, "\x1ba", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Alt: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideLeft}}, UTF8: []uint8("\xc3\xa5"), UnshiftedCodepoint: 'a'}, InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: InputConfigOptionAsAltLeft})
	ke3ACExpect(t, "\xc3\xa5", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyA, Mods: InputKeyMods{Alt: true, Sides: InputKeyModsModsSide{Alt: InputKeyModsModSideRight}}, UTF8: []uint8("\xc3\xa5"), UnshiftedCodepoint: 'a'}, InputKeyEncodeOptions{AltEscPrefix: true, MacosOptionAsAlt: InputConfigOptionAsAltLeft})
	ke3ACExpect(t, "", InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyKeyB, Mods: InputKeyMods{Super: true}, UTF8: []uint8("b")}, InputKeyEncodeOptions{})
}

// Go-only: panic test: inputKeyEncodePcStyleFunctionKey: a key outside the
// key table indexes past function_keys.keys (std.EnumArray.get)
func TestKe3_panic_pcStyleFunctionKey_key_outside_the_table(t *testing.T) {
	testPlatform(t, "linux")
	ke3ExpectPanic(t, func() {
		inputKeyEncodePcStyleFunctionKey(InputKeyKey(inputKeyKeyCount), InputKeyMods{}, false, false, false, false)
	})
	ke3ExpectPanic(t, func() {
		inputKeyEncodePcStyleFunctionKey(InputKeyKey(-1), InputKeyMods{}, false, false, false, false)
	})
}

// Go-only: panic test: inputKeyEncodeCopyToBuf: the source overlaps the
// destination (@memcpy arguments alias)
func TestKe3_panic_copyToBuf_arguments_alias(t *testing.T) {
	testPlatform(t, "linux")
	var buf [8]uint8
	// The same bytes.
	ke3ExpectPanic(t, func() {
		inputKeyEncodeCopyToBuf(buf[0:4], buf[0:4])
	})
	// The source starts inside the destination.
	ke3ExpectPanic(t, func() {
		inputKeyEncodeCopyToBuf(buf[0:8], buf[2:6])
	})
	// The destination starts inside the source.
	ke3ExpectPanic(t, func() {
		inputKeyEncodeCopyToBuf(buf[2:8], buf[0:4])
	})
	// Adjacent ranges do not alias: copied as usual.
	copy(buf[0:4], "abcd")
	got, err := inputKeyEncodeCopyToBuf(buf[4:8], buf[0:4])
	zigTry(t, err)
	expectEqualStrings(t, "abcd", string(got))
}
