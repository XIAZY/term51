// Tests of input_key.go, translated from src/input/key.zig.
//
// Translation notes (project lead's decisions, see
// scripts/not-translated.txt):
//
//   - Every KeyEvent literal says `action: InputKeyActionPress` (Zig's
//     default, not the Go zero value): kmKeyEventDefault is `KeyEvent{}`.
//   - `inline for (@typeInfo(Key).@"enum".fields)` is a loop over
//     InputKeyKeyValues(); `@tagName(key)` is `key.String()`.
//   - `std.testing.expectEqual(?T, ...)` compares Optional values.
//   - Platform branches (`builtin.target.os.tag.isDarwin()`) run once per
//     platform mode: linux, darwin and ios (kmPlatforms, kmIsDarwin in
//     input_key_mods_test.go). Every check in key.zig is isDarwin(), so
//     darwin and ios expect the same values.
//   - Checks of `@typeInfo(...).tag_type` are `// Zig-only:` comments.

package terminal

import (
	"strings"
	"testing"
)

// kmKeyEventDefault is Zig's `KeyEvent{}`: the action defaults to .press.
func kmKeyEventDefault() InputKeyKeyEvent {
	return InputKeyKeyEvent{Action: InputKeyActionPress}
}

// kmKeyTableEntry is an element of term51ADKeyTable: `struct { Key, c_int,
// []const u8 }`.
type kmKeyTableEntry struct {
	x0 InputKeyKey
	x1 int32
	x2 string
}

// kmKeyTable is term51ADKeyTable: every Key, its c_int value and its W3C
// code. Transcribed from the enum declaration order (176 keys).
var kmKeyTable = [...]kmKeyTableEntry{
	{InputKeyKeyUnidentified, 0, "Unidentified"},
	{InputKeyKeyBackquote, 1, "Backquote"},
	{InputKeyKeyBackslash, 2, "Backslash"},
	{InputKeyKeyBracketLeft, 3, "BracketLeft"},
	{InputKeyKeyBracketRight, 4, "BracketRight"},
	{InputKeyKeyComma, 5, "Comma"},
	{InputKeyKeyDigit0, 6, "Digit0"},
	{InputKeyKeyDigit1, 7, "Digit1"},
	{InputKeyKeyDigit2, 8, "Digit2"},
	{InputKeyKeyDigit3, 9, "Digit3"},
	{InputKeyKeyDigit4, 10, "Digit4"},
	{InputKeyKeyDigit5, 11, "Digit5"},
	{InputKeyKeyDigit6, 12, "Digit6"},
	{InputKeyKeyDigit7, 13, "Digit7"},
	{InputKeyKeyDigit8, 14, "Digit8"},
	{InputKeyKeyDigit9, 15, "Digit9"},
	{InputKeyKeyEqual, 16, "Equal"},
	{InputKeyKeyIntlBackslash, 17, "IntlBackslash"},
	{InputKeyKeyIntlRo, 18, "IntlRo"},
	{InputKeyKeyIntlYen, 19, "IntlYen"},
	{InputKeyKeyKeyA, 20, "KeyA"},
	{InputKeyKeyKeyB, 21, "KeyB"},
	{InputKeyKeyKeyC, 22, "KeyC"},
	{InputKeyKeyKeyD, 23, "KeyD"},
	{InputKeyKeyKeyE, 24, "KeyE"},
	{InputKeyKeyKeyF, 25, "KeyF"},
	{InputKeyKeyKeyG, 26, "KeyG"},
	{InputKeyKeyKeyH, 27, "KeyH"},
	{InputKeyKeyKeyI, 28, "KeyI"},
	{InputKeyKeyKeyJ, 29, "KeyJ"},
	{InputKeyKeyKeyK, 30, "KeyK"},
	{InputKeyKeyKeyL, 31, "KeyL"},
	{InputKeyKeyKeyM, 32, "KeyM"},
	{InputKeyKeyKeyN, 33, "KeyN"},
	{InputKeyKeyKeyO, 34, "KeyO"},
	{InputKeyKeyKeyP, 35, "KeyP"},
	{InputKeyKeyKeyQ, 36, "KeyQ"},
	{InputKeyKeyKeyR, 37, "KeyR"},
	{InputKeyKeyKeyS, 38, "KeyS"},
	{InputKeyKeyKeyT, 39, "KeyT"},
	{InputKeyKeyKeyU, 40, "KeyU"},
	{InputKeyKeyKeyV, 41, "KeyV"},
	{InputKeyKeyKeyW, 42, "KeyW"},
	{InputKeyKeyKeyX, 43, "KeyX"},
	{InputKeyKeyKeyY, 44, "KeyY"},
	{InputKeyKeyKeyZ, 45, "KeyZ"},
	{InputKeyKeyMinus, 46, "Minus"},
	{InputKeyKeyPeriod, 47, "Period"},
	{InputKeyKeyQuote, 48, "Quote"},
	{InputKeyKeySemicolon, 49, "Semicolon"},
	{InputKeyKeySlash, 50, "Slash"},
	{InputKeyKeyAltLeft, 51, "AltLeft"},
	{InputKeyKeyAltRight, 52, "AltRight"},
	{InputKeyKeyBackspace, 53, "Backspace"},
	{InputKeyKeyCapsLock, 54, "CapsLock"},
	{InputKeyKeyContextMenu, 55, "ContextMenu"},
	{InputKeyKeyControlLeft, 56, "ControlLeft"},
	{InputKeyKeyControlRight, 57, "ControlRight"},
	{InputKeyKeyEnter, 58, "Enter"},
	{InputKeyKeyMetaLeft, 59, "MetaLeft"},
	{InputKeyKeyMetaRight, 60, "MetaRight"},
	{InputKeyKeyShiftLeft, 61, "ShiftLeft"},
	{InputKeyKeyShiftRight, 62, "ShiftRight"},
	{InputKeyKeySpace, 63, "Space"},
	{InputKeyKeyTab, 64, "Tab"},
	{InputKeyKeyConvert, 65, "Convert"},
	{InputKeyKeyKanaMode, 66, "KanaMode"},
	{InputKeyKeyNonConvert, 67, "NonConvert"},
	{InputKeyKeyDelete, 68, "Delete"},
	{InputKeyKeyEnd, 69, "End"},
	{InputKeyKeyHelp, 70, "Help"},
	{InputKeyKeyHome, 71, "Home"},
	{InputKeyKeyInsert, 72, "Insert"},
	{InputKeyKeyPageDown, 73, "PageDown"},
	{InputKeyKeyPageUp, 74, "PageUp"},
	{InputKeyKeyArrowDown, 75, "ArrowDown"},
	{InputKeyKeyArrowLeft, 76, "ArrowLeft"},
	{InputKeyKeyArrowRight, 77, "ArrowRight"},
	{InputKeyKeyArrowUp, 78, "ArrowUp"},
	{InputKeyKeyNumLock, 79, "NumLock"},
	{InputKeyKeyNumpad0, 80, "Numpad0"},
	{InputKeyKeyNumpad1, 81, "Numpad1"},
	{InputKeyKeyNumpad2, 82, "Numpad2"},
	{InputKeyKeyNumpad3, 83, "Numpad3"},
	{InputKeyKeyNumpad4, 84, "Numpad4"},
	{InputKeyKeyNumpad5, 85, "Numpad5"},
	{InputKeyKeyNumpad6, 86, "Numpad6"},
	{InputKeyKeyNumpad7, 87, "Numpad7"},
	{InputKeyKeyNumpad8, 88, "Numpad8"},
	{InputKeyKeyNumpad9, 89, "Numpad9"},
	{InputKeyKeyNumpadAdd, 90, "NumpadAdd"},
	{InputKeyKeyNumpadBackspace, 91, "NumpadBackspace"},
	{InputKeyKeyNumpadClear, 92, "NumpadClear"},
	{InputKeyKeyNumpadClearEntry, 93, "NumpadClearEntry"},
	{InputKeyKeyNumpadComma, 94, "NumpadComma"},
	{InputKeyKeyNumpadDecimal, 95, "NumpadDecimal"},
	{InputKeyKeyNumpadDivide, 96, "NumpadDivide"},
	{InputKeyKeyNumpadEnter, 97, "NumpadEnter"},
	{InputKeyKeyNumpadEqual, 98, "NumpadEqual"},
	{InputKeyKeyNumpadMemoryAdd, 99, "NumpadMemoryAdd"},
	{InputKeyKeyNumpadMemoryClear, 100, "NumpadMemoryClear"},
	{InputKeyKeyNumpadMemoryRecall, 101, "NumpadMemoryRecall"},
	{InputKeyKeyNumpadMemoryStore, 102, "NumpadMemoryStore"},
	{InputKeyKeyNumpadMemorySubtract, 103, "NumpadMemorySubtract"},
	{InputKeyKeyNumpadMultiply, 104, "NumpadMultiply"},
	{InputKeyKeyNumpadParenLeft, 105, "NumpadParenLeft"},
	{InputKeyKeyNumpadParenRight, 106, "NumpadParenRight"},
	{InputKeyKeyNumpadSubtract, 107, "NumpadSubtract"},
	{InputKeyKeyNumpadSeparator, 108, "NumpadSeparator"},
	{InputKeyKeyNumpadUp, 109, "NumpadUp"},
	{InputKeyKeyNumpadDown, 110, "NumpadDown"},
	{InputKeyKeyNumpadRight, 111, "NumpadRight"},
	{InputKeyKeyNumpadLeft, 112, "NumpadLeft"},
	{InputKeyKeyNumpadBegin, 113, "NumpadBegin"},
	{InputKeyKeyNumpadHome, 114, "NumpadHome"},
	{InputKeyKeyNumpadEnd, 115, "NumpadEnd"},
	{InputKeyKeyNumpadInsert, 116, "NumpadInsert"},
	{InputKeyKeyNumpadDelete, 117, "NumpadDelete"},
	{InputKeyKeyNumpadPageUp, 118, "NumpadPageUp"},
	{InputKeyKeyNumpadPageDown, 119, "NumpadPageDown"},
	{InputKeyKeyEscape, 120, "Escape"},
	{InputKeyKeyF1, 121, "F1"},
	{InputKeyKeyF2, 122, "F2"},
	{InputKeyKeyF3, 123, "F3"},
	{InputKeyKeyF4, 124, "F4"},
	{InputKeyKeyF5, 125, "F5"},
	{InputKeyKeyF6, 126, "F6"},
	{InputKeyKeyF7, 127, "F7"},
	{InputKeyKeyF8, 128, "F8"},
	{InputKeyKeyF9, 129, "F9"},
	{InputKeyKeyF10, 130, "F10"},
	{InputKeyKeyF11, 131, "F11"},
	{InputKeyKeyF12, 132, "F12"},
	{InputKeyKeyF13, 133, "F13"},
	{InputKeyKeyF14, 134, "F14"},
	{InputKeyKeyF15, 135, "F15"},
	{InputKeyKeyF16, 136, "F16"},
	{InputKeyKeyF17, 137, "F17"},
	{InputKeyKeyF18, 138, "F18"},
	{InputKeyKeyF19, 139, "F19"},
	{InputKeyKeyF20, 140, "F20"},
	{InputKeyKeyF21, 141, "F21"},
	{InputKeyKeyF22, 142, "F22"},
	{InputKeyKeyF23, 143, "F23"},
	{InputKeyKeyF24, 144, "F24"},
	{InputKeyKeyF25, 145, "F25"},
	{InputKeyKeyFn, 146, "Fn"},
	{InputKeyKeyFnLock, 147, "FnLock"},
	{InputKeyKeyPrintScreen, 148, "PrintScreen"},
	{InputKeyKeyScrollLock, 149, "ScrollLock"},
	{InputKeyKeyPause, 150, "Pause"},
	{InputKeyKeyBrowserBack, 151, "BrowserBack"},
	{InputKeyKeyBrowserFavorites, 152, "BrowserFavorites"},
	{InputKeyKeyBrowserForward, 153, "BrowserForward"},
	{InputKeyKeyBrowserHome, 154, "BrowserHome"},
	{InputKeyKeyBrowserRefresh, 155, "BrowserRefresh"},
	{InputKeyKeyBrowserSearch, 156, "BrowserSearch"},
	{InputKeyKeyBrowserStop, 157, "BrowserStop"},
	{InputKeyKeyEject, 158, "Eject"},
	{InputKeyKeyLaunchApp1, 159, "LaunchApp1"},
	{InputKeyKeyLaunchApp2, 160, "LaunchApp2"},
	{InputKeyKeyLaunchMail, 161, "LaunchMail"},
	{InputKeyKeyMediaPlayPause, 162, "MediaPlayPause"},
	{InputKeyKeyMediaSelect, 163, "MediaSelect"},
	{InputKeyKeyMediaStop, 164, "MediaStop"},
	{InputKeyKeyMediaTrackNext, 165, "MediaTrackNext"},
	{InputKeyKeyMediaTrackPrevious, 166, "MediaTrackPrevious"},
	{InputKeyKeyPower, 167, "Power"},
	{InputKeyKeySleep, 168, "Sleep"},
	{InputKeyKeyAudioVolumeDown, 169, "AudioVolumeDown"},
	{InputKeyKeyAudioVolumeMute, 170, "AudioVolumeMute"},
	{InputKeyKeyAudioVolumeUp, 171, "AudioVolumeUp"},
	{InputKeyKeyWakeUp, 172, "WakeUp"},
	{InputKeyKeyCopy, 173, "Copy"},
	{InputKeyKeyCut, 174, "Cut"},
	{InputKeyKeyPaste, 175, "Paste"},
}

// kmKeyCodepoint is term51ADCodepoint: the expected codepoint of a key, from
// codepoint_map.
func kmKeyCodepoint(k InputKeyKey) Optional[uint32] {
	switch k {
	case InputKeyKeyKeyA:
		return Some(uint32('a'))
	case InputKeyKeyKeyB:
		return Some(uint32('b'))
	case InputKeyKeyKeyC:
		return Some(uint32('c'))
	case InputKeyKeyKeyD:
		return Some(uint32('d'))
	case InputKeyKeyKeyE:
		return Some(uint32('e'))
	case InputKeyKeyKeyF:
		return Some(uint32('f'))
	case InputKeyKeyKeyG:
		return Some(uint32('g'))
	case InputKeyKeyKeyH:
		return Some(uint32('h'))
	case InputKeyKeyKeyI:
		return Some(uint32('i'))
	case InputKeyKeyKeyJ:
		return Some(uint32('j'))
	case InputKeyKeyKeyK:
		return Some(uint32('k'))
	case InputKeyKeyKeyL:
		return Some(uint32('l'))
	case InputKeyKeyKeyM:
		return Some(uint32('m'))
	case InputKeyKeyKeyN:
		return Some(uint32('n'))
	case InputKeyKeyKeyO:
		return Some(uint32('o'))
	case InputKeyKeyKeyP:
		return Some(uint32('p'))
	case InputKeyKeyKeyQ:
		return Some(uint32('q'))
	case InputKeyKeyKeyR:
		return Some(uint32('r'))
	case InputKeyKeyKeyS:
		return Some(uint32('s'))
	case InputKeyKeyKeyT:
		return Some(uint32('t'))
	case InputKeyKeyKeyU:
		return Some(uint32('u'))
	case InputKeyKeyKeyV:
		return Some(uint32('v'))
	case InputKeyKeyKeyW:
		return Some(uint32('w'))
	case InputKeyKeyKeyX:
		return Some(uint32('x'))
	case InputKeyKeyKeyY:
		return Some(uint32('y'))
	case InputKeyKeyKeyZ:
		return Some(uint32('z'))
	case InputKeyKeyDigit0, InputKeyKeyNumpad0:
		return Some(uint32('0'))
	case InputKeyKeyDigit1, InputKeyKeyNumpad1:
		return Some(uint32('1'))
	case InputKeyKeyDigit2, InputKeyKeyNumpad2:
		return Some(uint32('2'))
	case InputKeyKeyDigit3, InputKeyKeyNumpad3:
		return Some(uint32('3'))
	case InputKeyKeyDigit4, InputKeyKeyNumpad4:
		return Some(uint32('4'))
	case InputKeyKeyDigit5, InputKeyKeyNumpad5:
		return Some(uint32('5'))
	case InputKeyKeyDigit6, InputKeyKeyNumpad6:
		return Some(uint32('6'))
	case InputKeyKeyDigit7, InputKeyKeyNumpad7:
		return Some(uint32('7'))
	case InputKeyKeyDigit8, InputKeyKeyNumpad8:
		return Some(uint32('8'))
	case InputKeyKeyDigit9, InputKeyKeyNumpad9:
		return Some(uint32('9'))
	case InputKeyKeySemicolon:
		return Some(uint32(';'))
	case InputKeyKeySpace:
		return Some(uint32(' '))
	case InputKeyKeyQuote:
		return Some(uint32('\''))
	case InputKeyKeyComma:
		return Some(uint32(','))
	case InputKeyKeyBackquote:
		return Some(uint32('`'))
	case InputKeyKeyPeriod, InputKeyKeyNumpadDecimal:
		return Some(uint32('.'))
	case InputKeyKeySlash, InputKeyKeyNumpadDivide:
		return Some(uint32('/'))
	case InputKeyKeyMinus, InputKeyKeyNumpadSubtract:
		return Some(uint32('-'))
	case InputKeyKeyEqual, InputKeyKeyNumpadEqual:
		return Some(uint32('='))
	case InputKeyKeyBracketLeft:
		return Some(uint32('['))
	case InputKeyKeyBracketRight:
		return Some(uint32(']'))
	case InputKeyKeyBackslash:
		return Some(uint32('\\'))
	case InputKeyKeyTab:
		return Some(uint32('\t'))
	case InputKeyKeyNumpadMultiply:
		return Some(uint32('*'))
	case InputKeyKeyNumpadAdd:
		return Some(uint32('+'))
	default:
		return None[uint32]()
	}
}

// Zig: test "fromASCII should not return keypad keys"
func TestKmKey_fromASCII_should_not_return_keypad_keys(t *testing.T) {
	testPlatform(t, "linux")
	expect(t, zigUnwrap(t, InputKeyKeyFromASCII('0')) == InputKeyKeyDigit0, "Key.fromASCII('0').? == .digit_0")
	expect(t, !InputKeyKeyFromASCII('*').Valid, "Key.fromASCII('*') == null")
}

// Zig: test "keypad keys"
func TestKmKey_keypad_keys(t *testing.T) {
	testPlatform(t, "linux")
	expect(t, InputKeyKeyNumpad0.Keypad(), "Key.numpad_0.keypad()")
	expect(t, !InputKeyKeyDigit1.Keypad(), "!Key.digit_1.keypad()")
}

// Zig: test "w3c"
func TestKmKey_w3c(t *testing.T) {
	testPlatform(t, "linux")
	// All our keys should convert to and from the W3C format.
	// We don't support every key in the W3C spec, so we only
	// check the enum fields.
	for _, key := range InputKeyKeyValues() {
		w3c_name := key.W3c()
		expectEqual(t, key, zigUnwrap(t, InputKeyKeyFromW3C(w3c_name)))
	}
}

// Zig: test "ctrlOrSuper"
// Go-only: platform modes (the helper sets ctrl or super by platform).
func TestKmKey_ctrlOrSuper(t *testing.T) {
	for _, goos := range kmPlatforms {
		t.Run(goos, func(t *testing.T) {
			testPlatform(t, goos)
			m := InputKeyCtrlOrSuper(InputKeyMods{})

			expect(t, m.CtrlOrSuper(), "m.ctrlOrSuper()")
		})
	}
}

// Zig: test "term51: Action: every value and its integer"
func TestKmKey_term51_Action_every_value_and_its_integer(t *testing.T) {
	testPlatform(t, "linux")
	expectEqual(t, int32(0), int32(InputKeyActionRelease))
	expectEqual(t, int32(1), int32(InputKeyActionPress))
	expectEqual(t, int32(2), int32(InputKeyActionRepeat))
	expectEqual(t, 3, inputKeyActionCount)
	// Zig-only: expect(@typeInfo(Action).@"enum".tag_type == c_int)
	expectEqual(t, InputKeyActionRelease, InputKeyAction(0))
	expectEqual(t, InputKeyActionPress, InputKeyAction(1))
	expectEqual(t, InputKeyActionRepeat, InputKeyAction(2))
}

// Zig: test "term51: Key: every value, count and integer"
func TestKmKey_term51_Key_every_value_count_and_integer(t *testing.T) {
	testPlatform(t, "linux")
	fields := InputKeyKeyValues()
	expectEqual(t, 176, len(fields))
	expectEqual(t, 176, inputKeyKeyCount)
	expectEqual(t, 176, len(kmKeyTable))
	// Zig-only: expect(@typeInfo(Key).@"enum".tag_type == c_int)
	for i, row := range kmKeyTable {
		expectEqual(t, int32(i), row.x1)
		expectEqual(t, row.x1, int32(row.x0))
		expectEqual(t, row.x0, InputKeyKey(row.x1))
	}

	// Spot checks on section boundaries.
	expectEqual(t, int32(0), int32(InputKeyKeyUnidentified))
	expectEqual(t, int32(1), int32(InputKeyKeyBackquote))
	expectEqual(t, int32(20), int32(InputKeyKeyKeyA))
	expectEqual(t, int32(50), int32(InputKeyKeySlash))
	expectEqual(t, int32(51), int32(InputKeyKeyAltLeft))
	expectEqual(t, int32(80), int32(InputKeyKeyNumpad0))
	expectEqual(t, int32(119), int32(InputKeyKeyNumpadPageDown))
	expectEqual(t, int32(120), int32(InputKeyKeyEscape))
	expectEqual(t, int32(175), int32(InputKeyKeyPaste))
}

// Zig: test "term51: KeyEvent: field defaults"
func TestKmKey_term51_KeyEvent_field_defaults(t *testing.T) {
	testPlatform(t, "linux")
	ev := kmKeyEventDefault()
	expectEqual(t, InputKeyActionPress, ev.Action)
	expectEqual(t, InputKeyKeyUnidentified, ev.Key)
	expectEqual(t, InputKeyMods{}, ev.Mods)
	expectEqual(t, InputKeyMods{}, ev.ConsumedMods)
	expectEqual(t, false, ev.Composing)
	expectEqualStrings(t, "", string(ev.UTF8))
	expectEqual(t, uint32(0), ev.UnshiftedCodepoint)
}

// Zig: test "term51: KeyEvent.effectiveMods: empty utf8 returns mods untouched"
func TestKmKey_term51_KeyEvent_effectiveMods_empty_utf8_returns_mods_untouched(t *testing.T) {
	testPlatform(t, "linux")
	ev := kmKeyEventDefault()
	ev.Mods = InputKeyMods{Shift: true, Ctrl: true, CapsLock: true}
	ev.ConsumedMods = InputKeyMods{Shift: true, Ctrl: true}
	expectEqual(t, ev.Mods, ev.EffectiveMods())
	expectEqual(t,
		InputKeyMods{Shift: true, Ctrl: true, CapsLock: true},
		ev.EffectiveMods(),
	)
}

// Zig: test "term51: KeyEvent.effectiveMods: utf8 subtracts consumed mods"
func TestKmKey_term51_KeyEvent_effectiveMods_utf8_subtracts_consumed_mods(t *testing.T) {
	testPlatform(t, "linux")
	{
		ev := kmKeyEventDefault()
		ev.UTF8 = []uint8("A")
		ev.Mods = InputKeyMods{Shift: true}
		ev.ConsumedMods = InputKeyMods{Shift: true}
		expectEqual(t, InputKeyMods{}, ev.EffectiveMods())
	}
	{
		ev := kmKeyEventDefault()
		ev.UTF8 = []uint8("A")
		ev.Mods = InputKeyMods{Shift: true, Ctrl: true, Alt: true, Super: true}
		ev.ConsumedMods = InputKeyMods{Shift: true}
		expectEqual(t,
			InputKeyMods{Ctrl: true, Alt: true, Super: true},
			ev.EffectiveMods(),
		)
	}
	{
		// Nothing consumed leaves everything, including locks.
		ev := kmKeyEventDefault()
		ev.UTF8 = []uint8("a")
		ev.Mods = InputKeyMods{Ctrl: true, NumLock: true}
		expectEqual(t,
			InputKeyMods{Ctrl: true, NumLock: true},
			ev.EffectiveMods(),
		)
	}
	{
		// Consumed bits that are not set in mods are ignored.
		ev := kmKeyEventDefault()
		ev.UTF8 = []uint8("a")
		ev.Mods = InputKeyMods{Ctrl: true}
		ev.ConsumedMods = InputKeyMods{Alt: true}
		expectEqual(t, InputKeyMods{Ctrl: true}, ev.EffectiveMods())
	}
}

// Zig: test "term51: KeyEvent.effectiveMods: composing does not change the result"
func TestKmKey_term51_KeyEvent_effectiveMods_composing_does_not_change_the_result(t *testing.T) {
	testPlatform(t, "linux")
	ev := kmKeyEventDefault()
	ev.UTF8 = []uint8("x")
	ev.Composing = true
	ev.Mods = InputKeyMods{Alt: true, Super: true}
	ev.ConsumedMods = InputKeyMods{Alt: true}
	expectEqual(t, InputKeyMods{Super: true}, ev.EffectiveMods())
}

// Zig: test "term51: KeyEvent.effectiveMods: consumed side bits are cleared too"
func TestKmKey_term51_KeyEvent_effectiveMods_consumed_side_bits_are_cleared_too(t *testing.T) {
	testPlatform(t, "linux")
	// Mods.unset is a plain bit operation, so consumed side bits clear the
	// matching side bits of mods even though the key bit itself is separate.
	ev := kmKeyEventDefault()
	ev.UTF8 = []uint8("A")
	ev.Mods = InputKeyMods{Shift: true, Sides: kmSides{Shift: kmRight}}
	ev.ConsumedMods = InputKeyMods{Shift: true, Sides: kmSides{Shift: kmRight}}
	expectEqual(t, InputKeyMods{}, ev.EffectiveMods())

	// A consumed mods without the side bit leaves the right side bit behind.
	ev2 := kmKeyEventDefault()
	ev2.UTF8 = []uint8("A")
	ev2.Mods = InputKeyMods{Shift: true, Sides: kmSides{Shift: kmRight}}
	ev2.ConsumedMods = InputKeyMods{Shift: true}
	expectEqual(t,
		InputKeyMods{Sides: kmSides{Shift: kmRight}},
		ev2.EffectiveMods(),
	)
}

// Zig: test "term51: KeyEvent.bindingHash: equal inputs hash equal"
func TestKmKey_term51_KeyEvent_bindingHash_equal_inputs_hash_equal(t *testing.T) {
	testPlatform(t, "linux")
	a := kmKeyEventDefault()
	a.Key = InputKeyKeyKeyA
	a.Mods = InputKeyMods{Ctrl: true}
	a.UnshiftedCodepoint = 'a'
	b := kmKeyEventDefault()
	b.Key = InputKeyKeyKeyA
	b.Mods = InputKeyMods{Ctrl: true}
	b.UnshiftedCodepoint = 'a'
	expectEqual(t, a.BindingHash(), b.BindingHash())
	// Deterministic across calls.
	expectEqual(t, a.BindingHash(), a.BindingHash())
	d := kmKeyEventDefault()
	expectEqual(t, d.BindingHash(), kmKeyEventDefault().BindingHash())
}

// Zig: test "term51: KeyEvent.bindingHash: key codepoint and binding mods change the hash"
func TestKmKey_term51_KeyEvent_bindingHash_key_codepoint_and_binding_mods_change_the_hash(t *testing.T) {
	testPlatform(t, "linux")
	base := kmKeyEventDefault()
	base.Key = InputKeyKeyKeyA
	base.UnshiftedCodepoint = 'a'
	h := base.BindingHash()

	other := base
	other.Key = InputKeyKeyKeyB
	expect(t, other.BindingHash() != h, "other.bindingHash() != h")

	other = base
	other.UnshiftedCodepoint = 'b'
	expect(t, other.BindingHash() != h, "other.bindingHash() != h")

	other = base
	other.UnshiftedCodepoint = 0
	expect(t, other.BindingHash() != h, "other.bindingHash() != h")

	other = base
	other.Mods = InputKeyMods{Shift: true}
	hs := other.BindingHash()
	expect(t, hs != h, "hs != h")
	other.Mods = InputKeyMods{Ctrl: true}
	hc := other.BindingHash()
	expect(t, hc != h && hc != hs, "hc != h and hc != hs")
	other.Mods = InputKeyMods{Alt: true}
	ha := other.BindingHash()
	expect(t, ha != h && ha != hs && ha != hc, "ha != h and ha != hs and ha != hc")
	other.Mods = InputKeyMods{Super: true}
	hu := other.BindingHash()
	expect(t, hu != h && hu != hs && hu != hc && hu != ha, "hu != h and hu != hs and hu != hc and hu != ha")
}

// Zig: test "term51: KeyEvent.bindingHash: irrelevant fields are ignored"
func TestKmKey_term51_KeyEvent_bindingHash_irrelevant_fields_are_ignored(t *testing.T) {
	testPlatform(t, "linux")
	base := kmKeyEventDefault()
	base.Key = InputKeyKeyEnter
	base.Mods = InputKeyMods{Ctrl: true}
	base.UnshiftedCodepoint = 13
	h := base.BindingHash()

	// action
	other := base
	other.Action = InputKeyActionRelease
	expectEqual(t, h, other.BindingHash())
	other.Action = InputKeyActionRepeat
	expectEqual(t, h, other.BindingHash())

	// composing
	other = base
	other.Composing = true
	expectEqual(t, h, other.BindingHash())

	// utf8
	other = base
	other.UTF8 = []uint8("\r")
	expectEqual(t, h, other.BindingHash())

	// consumed mods
	other = base
	other.ConsumedMods = InputKeyMods{Ctrl: true, Shift: true}
	expectEqual(t, h, other.BindingHash())

	// locks
	other = base
	other.Mods.CapsLock = true
	other.Mods.NumLock = true
	expectEqual(t, h, other.BindingHash())

	// sides
	other = base
	other.Mods.Sides = InputKeyModsModsSideMask.Sides
	expectEqual(t, h, other.BindingHash())
}

// Zig: test "term51: Key.fromASCII: every byte"
func TestKmKey_term51_Key_fromASCII_every_byte(t *testing.T) {
	testPlatform(t, "linux")
	found := 0
	for ch := 0; ch < 256; ch++ {
		c := uint8(ch)
		var expected Optional[InputKeyKey]
		switch {
		case c >= 'a' && c <= 'z':
			expected = Some(InputKeyKey(int32(InputKeyKeyKeyA) + int32(c-'a')))
		case c >= '0' && c <= '9':
			expected = Some(InputKeyKey(int32(InputKeyKeyDigit0) + int32(c-'0')))
		case c == ';':
			expected = Some(InputKeyKeySemicolon)
		case c == ' ':
			expected = Some(InputKeyKeySpace)
		case c == '\'':
			expected = Some(InputKeyKeyQuote)
		case c == ',':
			expected = Some(InputKeyKeyComma)
		case c == '`':
			expected = Some(InputKeyKeyBackquote)
		case c == '.':
			expected = Some(InputKeyKeyPeriod)
		case c == '/':
			expected = Some(InputKeyKeySlash)
		case c == '-':
			expected = Some(InputKeyKeyMinus)
		case c == '=':
			expected = Some(InputKeyKeyEqual)
		case c == '[':
			expected = Some(InputKeyKeyBracketLeft)
		case c == ']':
			expected = Some(InputKeyKeyBracketRight)
		case c == '\\':
			expected = Some(InputKeyKeyBackslash)
		case c == '\t':
			expected = Some(InputKeyKeyTab)
		default:
			expected = None[InputKeyKey]()
		}
		expectEqual(t, expected, InputKeyKeyFromASCII(c))
		if expected.Valid {
			found += 1
		}
	}
	// 26 letters + 10 digits + 12 punctuation/space characters + tab
	expectEqual(t, 26+10+12+1, found)
}

// Zig: test "term51: Key.fromASCII: specific values and keypad exclusion"
func TestKmKey_term51_Key_fromASCII_specific_values_and_keypad_exclusion(t *testing.T) {
	testPlatform(t, "linux")
	expectEqual(t, InputKeyKeyKeyA, zigUnwrap(t, InputKeyKeyFromASCII('a')))
	expectEqual(t, InputKeyKeyKeyZ, zigUnwrap(t, InputKeyKeyFromASCII('z')))
	expectEqual(t, InputKeyKeyDigit0, zigUnwrap(t, InputKeyKeyFromASCII('0')))
	expectEqual(t, InputKeyKeyDigit9, zigUnwrap(t, InputKeyKeyFromASCII('9')))
	expectEqual(t, InputKeyKeyPeriod, zigUnwrap(t, InputKeyKeyFromASCII('.')))
	expectEqual(t, InputKeyKeySlash, zigUnwrap(t, InputKeyKeyFromASCII('/')))
	expectEqual(t, InputKeyKeyMinus, zigUnwrap(t, InputKeyKeyFromASCII('-')))
	expectEqual(t, InputKeyKeyEqual, zigUnwrap(t, InputKeyKeyFromASCII('=')))
	expectEqual(t, InputKeyKeyTab, zigUnwrap(t, InputKeyKeyFromASCII(9)))

	// Keypad-only characters never map.
	expect(t, !InputKeyKeyFromASCII('*').Valid, "Key.fromASCII('*') == null")
	expect(t, !InputKeyKeyFromASCII('+').Valid, "Key.fromASCII('+') == null")

	// Uppercase, control characters other than tab, DEL and non-ASCII.
	expect(t, !InputKeyKeyFromASCII('A').Valid, "Key.fromASCII('A') == null")
	expect(t, !InputKeyKeyFromASCII('Z').Valid, "Key.fromASCII('Z') == null")
	expect(t, !InputKeyKeyFromASCII(0).Valid, "Key.fromASCII(0) == null")
	expect(t, !InputKeyKeyFromASCII('\n').Valid, "Key.fromASCII('\\n') == null")
	expect(t, !InputKeyKeyFromASCII('\r').Valid, "Key.fromASCII('\\r') == null")
	expect(t, !InputKeyKeyFromASCII(27).Valid, "Key.fromASCII(27) == null")
	expect(t, !InputKeyKeyFromASCII(127).Valid, "Key.fromASCII(127) == null")
	expect(t, !InputKeyKeyFromASCII(128).Valid, "Key.fromASCII(128) == null")
	expect(t, !InputKeyKeyFromASCII(255).Valid, "Key.fromASCII(255) == null")
}

// Zig: test "term51: Key.fromASCII: result is never a keypad key"
func TestKmKey_term51_Key_fromASCII_result_is_never_a_keypad_key(t *testing.T) {
	testPlatform(t, "linux")
	for ch := 0; ch < 256; ch++ {
		if r := InputKeyKeyFromASCII(uint8(ch)); r.Valid {
			k := r.Value
			expect(t, !k.Keypad(), "!k.keypad()")
			// The result always round trips to the same character.
			expectEqual(t, uint32(ch), zigUnwrap(t, k.Codepoint()))
		}
	}
}

// Zig: test "term51: Key.fromW3C: exact strings for every key"
func TestKmKey_term51_Key_fromW3C_exact_strings_for_every_key(t *testing.T) {
	testPlatform(t, "linux")
	for _, row := range kmKeyTable {
		expectEqual(t, row.x0, zigUnwrap(t, InputKeyKeyFromW3C([]uint8(row.x2))))
	}
}

// Zig: test "term51: Key.fromW3C: special cases"
func TestKmKey_term51_Key_fromW3C_special_cases(t *testing.T) {
	testPlatform(t, "linux")
	// Whole string lowercased matches an enum name directly.
	expectEqual(t, InputKeyKeyEscape, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("Escape"))))
	expectEqual(t, InputKeyKeyEscape, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("escape"))))
	expectEqual(t, InputKeyKeyEscape, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("ESCAPE"))))
	expectEqual(t, InputKeyKeyUnidentified, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("Unidentified"))))
	expectEqual(t, InputKeyKeyF1, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("F1"))))
	expectEqual(t, InputKeyKeyF25, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("F25"))))
	expectEqual(t, InputKeyKeyFn, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("Fn"))))

	// The snake case spelling matches directly after lowercasing.
	expectEqual(t, InputKeyKeyKeyA, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("key_a"))))
	expectEqual(t, InputKeyKeyKeyA, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("KEY_A"))))
	expectEqual(t, InputKeyKeyArrowDown, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("arrow_down"))))

	// CamelCase conversion: caps and digits start a new word, the first
	// character does not get an underscore. A lowercase first letter works.
	expectEqual(t, InputKeyKeyKeyA, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("KeyA"))))
	expectEqual(t, InputKeyKeyKeyA, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("keyA"))))
	expectEqual(t, InputKeyKeyDigit0, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("Digit0"))))
	expectEqual(t, InputKeyKeyNumpad9, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("Numpad9"))))
	expectEqual(t, InputKeyKeyLaunchApp1, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("LaunchApp1"))))
	expectEqual(t, InputKeyKeyLaunchApp2, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("LaunchApp2"))))
	expectEqual(t,
		InputKeyKeyNumpadMemorySubtract,
		zigUnwrap(t, InputKeyKeyFromW3C([]uint8("NumpadMemorySubtract"))),
	)
	expectEqual(t, InputKeyKeyIntlRo, zigUnwrap(t, InputKeyKeyFromW3C([]uint8("IntlRo"))))
}

// Zig: test "term51: Key.fromW3C: invalid input returns null"
func TestKmKey_term51_Key_fromW3C_invalid_input_returns_null(t *testing.T) {
	testPlatform(t, "linux")
	expect(t, !InputKeyKeyFromW3C([]uint8("")).Valid, `Key.fromW3C("") == null`)
	expect(t, !InputKeyKeyFromW3C([]uint8("Nope")).Valid, `Key.fromW3C("Nope") == null`)
	expect(t, !InputKeyKeyFromW3C([]uint8("KeyAA")).Valid, `Key.fromW3C("KeyAA") == null`)
	expect(t, !InputKeyKeyFromW3C([]uint8("Key")).Valid, `Key.fromW3C("Key") == null`)
	expect(t, !InputKeyKeyFromW3C([]uint8("Lang1")).Valid, `Key.fromW3C("Lang1") == null`)
	expect(t, !InputKeyKeyFromW3C([]uint8("Digit10")).Valid, `Key.fromW3C("Digit10") == null`)

	// Any non-alphanumeric byte makes the conversion fail.
	expect(t, !InputKeyKeyFromW3C([]uint8("Key-A")).Valid, `Key.fromW3C("Key-A") == null`)
	expect(t, !InputKeyKeyFromW3C([]uint8("Key A")).Valid, `Key.fromW3C("Key A") == null`)
	expect(t, !InputKeyKeyFromW3C([]uint8("KeyA ")).Valid, `Key.fromW3C("KeyA ") == null`)
	expect(t, !InputKeyKeyFromW3C([]uint8("Key\xC3\xA4")).Valid, `Key.fromW3C("Key\xC3\xA4") == null`)
	expect(t, !InputKeyKeyFromW3C([]uint8("\x00")).Valid, `Key.fromW3C("\x00") == null`)

	// Digit in first position: no underscore is emitted, so "0" alone is
	// not an enum value either.
	expect(t, !InputKeyKeyFromW3C([]uint8("0")).Valid, `Key.fromW3C("0") == null`)
}

// Zig: test "term51: Key.fromW3C: length limit of 128 bytes"
func TestKmKey_term51_Key_fromW3C_length_limit_of_128_bytes(t *testing.T) {
	testPlatform(t, "linux")
	// 128 bytes is processed (and does not match); 129 is rejected up front.
	long := strings.Repeat("a", 129)
	expect(t, !InputKeyKeyFromW3C([]uint8(long[0:128])).Valid, "Key.fromW3C(long[0..128]) == null")
	expect(t, !InputKeyKeyFromW3C([]uint8(long[0:129])).Valid, "Key.fromW3C(long[0..129]) == null")

	// Uppercase letters expand to two bytes and overflow the output buffer.
	caps := strings.Repeat("A", 100)
	expect(t, !InputKeyKeyFromW3C([]uint8(caps)).Valid, "Key.fromW3C(caps) == null")
	expect(t, !InputKeyKeyFromW3C([]uint8(caps[0:64])).Valid, "Key.fromW3C(caps[0..64]) == null")
}

// Zig: test "term51: Key.w3c: exact string for every key"
func TestKmKey_term51_Key_w3c_exact_string_for_every_key(t *testing.T) {
	testPlatform(t, "linux")
	for _, row := range kmKeyTable {
		expectEqualStrings(t, row.x2, string(row.x0.W3c()))
	}
	expectEqualStrings(t, "Unidentified", string(InputKeyKeyUnidentified.W3c()))
	expectEqualStrings(t, "KeyA", string(InputKeyKeyKeyA.W3c()))
	expectEqualStrings(t, "Digit0", string(InputKeyKeyDigit0.W3c()))
	expectEqualStrings(t, "Fn", string(InputKeyKeyFn.W3c()))
	expectEqualStrings(t, "FnLock", string(InputKeyKeyFnLock.W3c()))
	expectEqualStrings(t, "LaunchApp1", string(InputKeyKeyLaunchApp1.W3c()))
	expectEqualStrings(t, "Paste", string(InputKeyKeyPaste.W3c()))
}

// Zig: test "term51: Key.printable: every key"
func TestKmKey_term51_Key_printable_every_key(t *testing.T) {
	testPlatform(t, "linux")
	for _, k := range InputKeyKeyValues() {
		expectEqual(t, kmKeyCodepoint(k).Valid, k.Printable())
	}

	// Keypad keys with a codepoint count as printable.
	expect(t, InputKeyKeyNumpad0.Printable(), "Key.numpad_0.printable()")
	expect(t, InputKeyKeyNumpadAdd.Printable(), "Key.numpad_add.printable()")
	expect(t, InputKeyKeyNumpadEqual.Printable(), "Key.numpad_equal.printable()")
	expect(t, !InputKeyKeyNumpadEnter.Printable(), "!Key.numpad_enter.printable()")
	expect(t, !InputKeyKeyNumpadSeparator.Printable(), "!Key.numpad_separator.printable()")
	expect(t, !InputKeyKeyNumpadComma.Printable(), "!Key.numpad_comma.printable()")
	expect(t, InputKeyKeyTab.Printable(), "Key.tab.printable()")
	expect(t, InputKeyKeySpace.Printable(), "Key.space.printable()")
	expect(t, !InputKeyKeyEnter.Printable(), "!Key.enter.printable()")
	expect(t, !InputKeyKeyBackspace.Printable(), "!Key.backspace.printable()")
	expect(t, !InputKeyKeyUnidentified.Printable(), "!Key.unidentified.printable()")
	expect(t, !InputKeyKeyIntlBackslash.Printable(), "!Key.intl_backslash.printable()")
}

// Zig: test "term51: Key.codepoint: every key"
func TestKmKey_term51_Key_codepoint_every_key(t *testing.T) {
	testPlatform(t, "linux")
	for _, k := range InputKeyKeyValues() {
		expectEqual(t, kmKeyCodepoint(k), k.Codepoint())
	}

	expectEqual(t, Some(uint32('a')), InputKeyKeyKeyA.Codepoint())
	expectEqual(t, Some(uint32('0')), InputKeyKeyDigit0.Codepoint())
	// The keypad digit shares the codepoint with the number row.
	expectEqual(t, Some(uint32('0')), InputKeyKeyNumpad0.Codepoint())
	expectEqual(t, Some(uint32('*')), InputKeyKeyNumpadMultiply.Codepoint())
	expectEqual(t, Some(uint32('+')), InputKeyKeyNumpadAdd.Codepoint())
	expectEqual(t, Some(uint32(' ')), InputKeyKeySpace.Codepoint())
	expectEqual(t, Some(uint32(9)), InputKeyKeyTab.Codepoint())
	expectEqual(t, None[uint32](), InputKeyKeyEnter.Codepoint())
	expectEqual(t, None[uint32](), InputKeyKeyUnidentified.Codepoint())
	expectEqual(t, None[uint32](), InputKeyKeyNumpadEnter.Codepoint())
}

// Zig: test "term51: Key.modifier: every key"
func TestKmKey_term51_Key_modifier_every_key(t *testing.T) {
	testPlatform(t, "linux")
	for _, k := range InputKeyKeyValues() {
		var expected bool
		switch k {
		case InputKeyKeyShiftLeft,
			InputKeyKeyShiftRight,
			InputKeyKeyControlLeft,
			InputKeyKeyControlRight,
			InputKeyKeyAltLeft,
			InputKeyKeyAltRight,
			InputKeyKeyMetaLeft,
			InputKeyKeyMetaRight:
			expected = true
		default:
			expected = false
		}
		expectEqual(t, expected, k.Modifier())
	}

	// Lock keys and fn are not modifiers here.
	expect(t, !InputKeyKeyCapsLock.Modifier(), "!Key.caps_lock.modifier()")
	expect(t, !InputKeyKeyNumLock.Modifier(), "!Key.num_lock.modifier()")
	expect(t, !InputKeyKeyScrollLock.Modifier(), "!Key.scroll_lock.modifier()")
	expect(t, !InputKeyKeyFn.Modifier(), `!Key.@"fn".modifier()`)
	expect(t, !InputKeyKeyFnLock.Modifier(), "!Key.fn_lock.modifier()")
}

// Zig: test "term51: Key.shouldBeRemappable: every key"
func TestKmKey_term51_Key_shouldBeRemappable_every_key(t *testing.T) {
	testPlatform(t, "linux")
	// Writing system keys are enum values 1 through 50 (backquote..slash)
	// and are the only ones that are not remappable.
	for _, row := range kmKeyTable {
		writing := row.x1 >= 1 && row.x1 <= 50
		expectEqual(t, !writing, row.x0.ShouldBeRemappable())
	}

	expect(t, !InputKeyKeyBackquote.ShouldBeRemappable(), "!Key.backquote.shouldBeRemappable()")
	expect(t, !InputKeyKeyKeyC.ShouldBeRemappable(), "!Key.key_c.shouldBeRemappable()")
	expect(t, !InputKeyKeyDigit0.ShouldBeRemappable(), "!Key.digit_0.shouldBeRemappable()")
	expect(t, !InputKeyKeyIntlYen.ShouldBeRemappable(), "!Key.intl_yen.shouldBeRemappable()")
	expect(t, !InputKeyKeySlash.ShouldBeRemappable(), "!Key.slash.shouldBeRemappable()")
	expect(t, InputKeyKeyUnidentified.ShouldBeRemappable(), "Key.unidentified.shouldBeRemappable()")
	expect(t, InputKeyKeyAltLeft.ShouldBeRemappable(), "Key.alt_left.shouldBeRemappable()")
	expect(t, InputKeyKeyCapsLock.ShouldBeRemappable(), "Key.caps_lock.shouldBeRemappable()")
	expect(t, InputKeyKeyEscape.ShouldBeRemappable(), "Key.escape.shouldBeRemappable()")
	expect(t, InputKeyKeySpace.ShouldBeRemappable(), "Key.space.shouldBeRemappable()")
	expect(t, InputKeyKeyEnter.ShouldBeRemappable(), "Key.enter.shouldBeRemappable()")
	expect(t, InputKeyKeyNumpad0.ShouldBeRemappable(), "Key.numpad_0.shouldBeRemappable()")
	expect(t, InputKeyKeyPaste.ShouldBeRemappable(), "Key.paste.shouldBeRemappable()")
}

// Zig: test "term51: Key.keypad: every key"
func TestKmKey_term51_Key_keypad_every_key(t *testing.T) {
	testPlatform(t, "linux")
	// The numpad_ keys are the contiguous values 80 (numpad_0) through 119
	// (numpad_page_down); num_lock (79) and escape (120) are not keypad.
	for _, row := range kmKeyTable {
		expected := row.x1 >= 80 && row.x1 <= 119
		expectEqual(t, expected, row.x0.Keypad())
		expectEqual(t,
			strings.HasPrefix(row.x0.String(), "numpad_"),
			row.x0.Keypad(),
		)
	}
	expect(t, !InputKeyKeyNumLock.Keypad(), "!Key.num_lock.keypad()")
	expect(t, InputKeyKeyNumpad0.Keypad(), "Key.numpad_0.keypad()")
	expect(t, InputKeyKeyNumpadPageDown.Keypad(), "Key.numpad_page_down.keypad()")
	expect(t, !InputKeyKeyEscape.Keypad(), "!Key.escape.keypad()")
	expect(t, !InputKeyKeyDigit0.Keypad(), "!Key.digit_0.keypad()")
}

// "term51: Key.imguiKey: not testable in the library build" is not
// translated (Key.imguiKey is not ported). See scripts/not-translated.txt.

// Zig: test "term51: Key.ctrlOrSuper: every key"
// Go-only: platform modes (Key.ctrlOrSuper uses isDarwin(): darwin and ios agree).
func TestKmKey_term51_Key_ctrlOrSuper_every_key(t *testing.T) {
	for _, goos := range kmPlatforms {
		t.Run(goos, func(t *testing.T) {
			testPlatform(t, goos)
			is_darwin := kmIsDarwin(goos)
			for _, k := range InputKeyKeyValues() {
				var expected bool
				if is_darwin {
					expected = k == InputKeyKeyMetaLeft || k == InputKeyKeyMetaRight
				} else {
					expected = k == InputKeyKeyControlLeft || k == InputKeyKeyControlRight
				}
				expectEqual(t, expected, k.CtrlOrSuper())
			}

			if is_darwin {
				expect(t, InputKeyKeyMetaLeft.CtrlOrSuper(), "Key.meta_left.ctrlOrSuper()")
				expect(t, InputKeyKeyMetaRight.CtrlOrSuper(), "Key.meta_right.ctrlOrSuper()")
				expect(t, !InputKeyKeyControlLeft.CtrlOrSuper(), "!Key.control_left.ctrlOrSuper()")
				expect(t, !InputKeyKeyControlRight.CtrlOrSuper(), "!Key.control_right.ctrlOrSuper()")
			} else {
				expect(t, InputKeyKeyControlLeft.CtrlOrSuper(), "Key.control_left.ctrlOrSuper()")
				expect(t, InputKeyKeyControlRight.CtrlOrSuper(), "Key.control_right.ctrlOrSuper()")
				expect(t, !InputKeyKeyMetaLeft.CtrlOrSuper(), "!Key.meta_left.ctrlOrSuper()")
				expect(t, !InputKeyKeyMetaRight.CtrlOrSuper(), "!Key.meta_right.ctrlOrSuper()")
			}
			expect(t, !InputKeyKeyShiftLeft.CtrlOrSuper(), "!Key.shift_left.ctrlOrSuper()")
			expect(t, !InputKeyKeyAltLeft.CtrlOrSuper(), "!Key.alt_left.ctrlOrSuper()")
			expect(t, !InputKeyKeyKeyC.CtrlOrSuper(), "!Key.key_c.ctrlOrSuper()")
		})
	}
}

// Zig: test "term51: Key.leftOrRightShift: every key"
func TestKmKey_term51_Key_leftOrRightShift_every_key(t *testing.T) {
	testPlatform(t, "linux")
	for _, k := range InputKeyKeyValues() {
		expectEqual(t,
			k == InputKeyKeyShiftLeft || k == InputKeyKeyShiftRight,
			k.LeftOrRightShift(),
		)
	}
	expect(t, InputKeyKeyShiftLeft.LeftOrRightShift(), "Key.shift_left.leftOrRightShift()")
	expect(t, InputKeyKeyShiftRight.LeftOrRightShift(), "Key.shift_right.leftOrRightShift()")
	expect(t, !InputKeyKeyAltLeft.LeftOrRightShift(), "!Key.alt_left.leftOrRightShift()")
	expect(t, !InputKeyKeyControlLeft.LeftOrRightShift(), "!Key.control_left.leftOrRightShift()")
	expect(t, !InputKeyKeyUnidentified.LeftOrRightShift(), "!Key.unidentified.leftOrRightShift()")
}

// Zig: test "term51: Key.leftOrRightAlt: every key"
func TestKmKey_term51_Key_leftOrRightAlt_every_key(t *testing.T) {
	testPlatform(t, "linux")
	for _, k := range InputKeyKeyValues() {
		expectEqual(t,
			k == InputKeyKeyAltLeft || k == InputKeyKeyAltRight,
			k.LeftOrRightAlt(),
		)
	}
	expect(t, InputKeyKeyAltLeft.LeftOrRightAlt(), "Key.alt_left.leftOrRightAlt()")
	expect(t, InputKeyKeyAltRight.LeftOrRightAlt(), "Key.alt_right.leftOrRightAlt()")
	expect(t, !InputKeyKeyShiftLeft.LeftOrRightAlt(), "!Key.shift_left.leftOrRightAlt()")
	expect(t, !InputKeyKeyMetaLeft.LeftOrRightAlt(), "!Key.meta_left.leftOrRightAlt()")
	expect(t, !InputKeyKeyUnidentified.LeftOrRightAlt(), "!Key.unidentified.leftOrRightAlt()")
}

// Zig: test "term51: ctrlOrSuper: sets exactly one of ctrl or super"
// Go-only: platform modes (the helper uses isDarwin(): darwin and ios agree).
func TestKmKey_term51_ctrlOrSuper_sets_exactly_one_of_ctrl_or_super(t *testing.T) {
	for _, goos := range kmPlatforms {
		t.Run(goos, func(t *testing.T) {
			testPlatform(t, goos)
			is_darwin := kmIsDarwin(goos)

			m := InputKeyCtrlOrSuper(InputKeyMods{})
			expectEqual(t, is_darwin, m.Super)
			expectEqual(t, !is_darwin, m.Ctrl)
			expect(t, !m.Shift, "!m.shift")
			expect(t, !m.Alt, "!m.alt")
			expect(t, !m.CapsLock, "!m.caps_lock")
			expect(t, !m.NumLock, "!m.num_lock")

			// Other mods are preserved, including locks and sides.
			m2 := InputKeyCtrlOrSuper(InputKeyMods{
				Shift:    true,
				Alt:      true,
				CapsLock: true,
				Sides:    kmSides{Alt: kmRight},
			})
			expect(t, m2.Shift, "m2.shift")
			expect(t, m2.Alt, "m2.alt")
			expect(t, m2.CapsLock, "m2.caps_lock")
			expectEqual(t, kmSides{Alt: kmRight}, m2.Sides)
			expectEqual(t, is_darwin, m2.Super)
			expectEqual(t, !is_darwin, m2.Ctrl)

			// Idempotent, and the other one stays untouched when already set.
			expectEqual(t, m, InputKeyCtrlOrSuper(m))
			m3 := InputKeyCtrlOrSuper(InputKeyMods{Ctrl: true, Super: true})
			expect(t, m3.Ctrl && m3.Super, "m3.ctrl and m3.super")
			expect(t, InputKeyCtrlOrSuper(InputKeyMods{}).CtrlOrSuper(), "ctrlOrSuper(.{}).ctrlOrSuper()")
		})
	}
}
