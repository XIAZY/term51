// Port of src/input/key.zig (Ghostty v1.3.1).
//
// File naming: key.zig lives in src/input/, so the file is `input_key.go`
// and every declaration takes the prefix `InputKey` (`inputKey` for private
// ones): `key.KeyEvent` is `InputKeyKeyEvent`, `key.Action` is
// `InputKeyAction`, `key.Key` is `InputKeyKey`, `key.ctrlOrSuper` is
// `InputKeyCtrlOrSuper`, `key.Mods` (a re-export of key_mods.Mods) is the
// alias `InputKeyMods` = `InputKeyModsMods`. Enum values are
// `InputKeyKey<Value>` (`.key_a` is `InputKeyKeyKeyA`, `.digit_0` is
// `InputKeyKeyDigit0`, `.@"fn"` is `InputKeyKeyFn`) and
// `InputKeyAction<Value>`. Functions of Key without `self` are free
// functions: `Key.fromASCII` is `InputKeyKeyFromASCII`, `Key.fromW3C` is
// `InputKeyKeyFromW3C`; `pub fn` methods are exported (`W3c`, `Printable`,
// `Modifier`, `ShouldBeRemappable`, `Keypad`, `Codepoint`, `CtrlOrSuper`,
// `LeftOrRightShift`, `LeftOrRightAlt`, `EffectiveMods`, `BindingHash`).
// The private table `Key.codepoint_map` is `inputKeyKeyCodepointMap`.
//
// Declarations appear in the same order as in key.zig.
//
// # Widths
//
// `u21` is `uint32` (`unshifted_codepoint`, `codepoint`). `Key` and
// `Action` are `enum(c_int)` so that the embedding API can use them as is;
// the Go types are int32 (as `OSCCommandProgressReportState` in osc.go).
//
// # Comptime
//
// `fromASCII`, `w3c`, `printable`, `keypad` and `codepoint` compute their
// answer for every key at compile time (`inline else` + `comptime`). In Go
// each comptime block is a function run once at start-up that fills a
// table indexed by the key (or by the byte, for fromASCII), with the same
// loop over `codepoint_map` or over the tag name:
// inputKeyKeyFromASCIITable, inputKeyKeyW3cTable, inputKeyKeyPrintableTable,
// inputKeyKeyKeypadTable, inputKeyKeyCodepointTable. Each method is one
// table load. `w3c` returns static memory in Zig; Go returns the table's
// []uint8, which the caller must not modify.
//
// Tables transcribed from key.zig: the tag names (inputKeyKeyNames, what
// `@tagName` returns) and `codepoint_map` were generated mechanically from
// key.zig by a throwaway script, in upstream's order.
//
// `std.meta.stringToEnum(Key, s)` (fromW3C): Key has more than 100 fields,
// so Zig 0.15.2 compares s with every tag name in order (`inline for` +
// `mem.eql`); inputKeyStringToEnumKey does the same loop.
// `std.ascii.lowerString`, `toLower` and `toUpper` are
// inputKeyAsciiLowerString, inputKeyAsciiToLower and inputKeyAsciiToUpper
// (Zig 0.15.2's code). `std.io.fixedBufferStream` is key_mods.go's
// inputKeyModsFixedBufferStream (same std type, already ported there).
//
// # bindingHash
//
// `KeyEvent.bindingHash` is `std.hash.Wyhash.init(0)` fed with
// `std.hash.autoHash` of `key` (a c_int enum: its 4 bytes),
// `unshifted_codepoint` (a u21: autoHash feeds 3 bytes, little-endian,
// measured against Zig 0.15.2: autoHash(u21 0x41) equals
// Wyhash.hash(0, {0x41, 0, 0})) and `mods.binding()` (a packed
// struct(u16): its 2 bytes). The port reproduces Zig's values exactly with
// the ported Wyhash (std_hash_wyhash.go) and three instantiations of
// std.hash.autoHash added to std_hash_auto_hash.go for these key types
// (stdHashAutoHashAutoHashEnumCInt, ...U21, ...PackedStructU16).
//
// # Darwin
//
// `Key.ctrlOrSuper` and the free function `ctrlOrSuper` have a comptime
// branch on `builtin.target.os.tag.isDarwin()`. The library is
// platform-neutral: both branches are ported and the choice is made by
// `inputKeyIsDarwin()`, which returns platformIsDarwin() (platform.go). Off
// Darwin, `Key.ctrlOrSuper` is true for control_left/control_right and
// `ctrlOrSuper(mods)` sets `ctrl`; on Darwin both use super.
//
// # Declarations that are not Zig declarations
//
//   - `inputKeyActionCount`, `inputKeyKeyCount`, `InputKeyKeyValues`
//     (Zig's `std.enums.values(Key)` / `@typeInfo(Key).@"enum".fields`, which
//     the tests iterate) and the `String` methods;
//   - `inputKeyKeyCodepointMapEntry`: the anonymous tuple struct
//     `struct { u21, Key }` of `codepoint_map`; tuple fields "0" and "1"
//     take the prefix `x` (as `x132_column` in modes.go): `x0`, `x1`;
//   - `inputKeyIsDarwin` (above).
//
// # Omitted
//
//   - `Key.imguiKey`: it returns `cimgui.c.ImGuiKey_*` constants from the
//     `dcimgui` module (Dear ImGui, the app's inspector), which is not part
//     of libghostty-vt and has no Go counterpart. The project test
//     "term51: Key.imguiKey: not testable in the library build" says the
//     same.
//   - the `cimgui` import, and the imports with nothing to declare: `std`,
//     `builtin`, `Allocator`, `OptionAsAlt` (unused outside tests).
// keycodes.zig is not needed: no non-test declaration of key.zig uses its
// types, so there is no input_keycodes.go.
// No C ABI declaration exists in this file ("IMPORTANT: Any changes here
// update include/ghostty.h" concerns the C header only).
//
// # Zig defaults a Go literal does not give
//
// A Go composite literal leaves an omitted field at the Go zero value. The
// Zig default that is NOT the Go zero value, which a translated literal
// must write out:
//   - `KeyEvent.action` defaults to `.press` (`InputKeyActionPress`, 1);
//     the zero value is `.release` (0).
// So Zig's `.{ .key = .key_a, .utf8 = "a" }` is
// `InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA,
// utf8: []uint8("a")}`, and `KeyEvent{}` is
// `InputKeyKeyEvent{action: InputKeyActionPress}`.
// All other KeyEvent defaults (`key` .unidentified = 0, `mods` and
// `consumed_mods` `.{}`, `composing` false, `utf8` "", `unshifted_codepoint`
// 0) are the Go zero value.

package terminal

import "strings"

// InputKeyMods is key.Mods, a re-export of key_mods.Mods.
type InputKeyMods = InputKeyModsMods

// InputKeyKeyEvent is key.KeyEvent: a generic key input event. This is the
// information that is necessary regardless of apprt in order to generate
// the proper terminal control sequences for a given key press.
//
// Some apprts may not be able to provide all of this information, such
// as GLFW. In this case, the apprt should provide as much information
// as it can and it should be expected that the terminal behavior
// will not be totally correct.
//
// See the file header: `action` defaults to `.press` in Zig, which is NOT
// the Go zero value.
type InputKeyKeyEvent struct {
	// The action: press, release, etc.
	//
	// Zig default: .press (NOT the Go zero value).
	action InputKeyAction

	// The keycode of the physical key that was pressed. This is agnostic
	// to the layout. Layout-dependent matching can only be done via the
	// UTF-8 or unshifted codepoint.
	//
	// Zig default: .unidentified (the zero value).
	key InputKeyKey

	// Mods are the modifiers that are pressed.
	//
	// Zig default: .{} (the zero value).
	mods InputKeyMods

	// The mods that were consumed in order to generate the text
	// in utf8. This has the mods set that were consumed, so to
	// get the set of mods that are effective you must negate
	// mods with this.
	//
	// This field is meaningless if utf8 is empty.
	//
	// Zig default: .{} (the zero value).
	consumed_mods InputKeyMods

	// Composing is true when this key event is part of a dead key
	// composition sequence and we're in the middle of it.
	//
	// Zig default: false.
	composing bool

	// The utf8 sequence that was generated by this key event.
	// This will be an empty string if there is no text generated.
	// If composing is true and this is non-empty, this is preedit
	// text.
	//
	// Zig: []const u8, default "" (nil and empty behave the same).
	utf8 []uint8

	// The codepoint for this key when it is unshifted. For example,
	// shift+a is "A" in UTF-8 but unshifted would provide 'a'.
	//
	// Zig: u21, default 0.
	unshifted_codepoint uint32
}

// EffectiveMods is key.KeyEvent.effectiveMods: returns the effective
// modifiers for this event. The effective modifiers are the mods that
// should be considered for keybindings.
func (self InputKeyKeyEvent) EffectiveMods() InputKeyMods {
	if len(self.utf8) == 0 {
		return self.mods
	}
	return self.mods.Unset(self.consumed_mods)
}

// BindingHash is key.KeyEvent.bindingHash: returns a unique hash for this
// key event to be used for tracking uniquess specifically with bindings.
// This omits fields that are irrelevant for bindings. See the file header.
func (self InputKeyKeyEvent) BindingHash() uint64 {
	hasher := stdHashWyhashInit(0)

	// These are all the fields that are explicitly part of Trigger.
	stdHashAutoHashAutoHashEnumCInt(&hasher, int32(self.key))
	stdHashAutoHashAutoHashU21(&hasher, self.unshifted_codepoint)
	stdHashAutoHashAutoHashPackedStructU16(&hasher, self.mods.Binding().Int())

	// Notes on unmapped things and why:
	//
	// - action: we don't have action-specific bindings right now
	//   AND we want to know if a key resulted in a binding regardless
	//   of action because a press should also ignore a release and so on.
	//
	// We can add to this if there is other confusion.

	return hasher.final()
}

// InputKeyAction is key.Action: the action associated with an input event.
// This is backed by a c_int so that we can use the enum as-is for our
// embedding API.
//
// IMPORTANT: Any changes here update include/ghostty.h
//
// Zig: enum(c_int), values numbered from zero in the order listed.
type InputKeyAction int32

const (
	InputKeyActionRelease InputKeyAction = iota
	InputKeyActionPress
	InputKeyActionRepeat

	inputKeyActionCount = iota
)

// String returns the Zig tag name (@tagName).
// A value outside the enum panics (index out of range).
func (self InputKeyAction) String() string { return inputKeyActionNames[self] }

// inputKeyActionNames holds the Zig tag names, indexed by value.
var inputKeyActionNames = [inputKeyActionCount]string{"release", "press", "repeat"}

// InputKeyKey is key.Key: the set of key codes that Ghostty is aware of.
// These represent physical keys on the keyboard. The logical key (or key
// string) is the string that is generated by the key event and that is up
// to the apprt to provide.
//
// Note that these are layout-independent. For example, the "a"
// key on a US keyboard is the same as the "ф" key on a Russian
// keyboard, but both will report the "a" enum value in the key
// event. These values are based on the W3C standard. See:
// https://www.w3.org/TR/uievents-code
//
// Layout-dependent strings are provided in the KeyEvent struct as
// UTF-8 and are produced by the associated apprt. Ghostty core has
// no mechanism to map input events to strings without the apprt.
//
// IMPORTANT: Any changes here update include/ghostty.h ghostty_input_key_e
//
// Zig: enum(c_int), values numbered from zero in the order listed
// (176 values).
type InputKeyKey int32

const (
	InputKeyKeyUnidentified InputKeyKey = iota

	// "Writing System Keys" § 3.1.1
	InputKeyKeyBackquote
	InputKeyKeyBackslash
	InputKeyKeyBracketLeft
	InputKeyKeyBracketRight
	InputKeyKeyComma
	InputKeyKeyDigit0
	InputKeyKeyDigit1
	InputKeyKeyDigit2
	InputKeyKeyDigit3
	InputKeyKeyDigit4
	InputKeyKeyDigit5
	InputKeyKeyDigit6
	InputKeyKeyDigit7
	InputKeyKeyDigit8
	InputKeyKeyDigit9
	InputKeyKeyEqual
	InputKeyKeyIntlBackslash
	InputKeyKeyIntlRo
	InputKeyKeyIntlYen
	InputKeyKeyKeyA
	InputKeyKeyKeyB
	InputKeyKeyKeyC
	InputKeyKeyKeyD
	InputKeyKeyKeyE
	InputKeyKeyKeyF
	InputKeyKeyKeyG
	InputKeyKeyKeyH
	InputKeyKeyKeyI
	InputKeyKeyKeyJ
	InputKeyKeyKeyK
	InputKeyKeyKeyL
	InputKeyKeyKeyM
	InputKeyKeyKeyN
	InputKeyKeyKeyO
	InputKeyKeyKeyP
	InputKeyKeyKeyQ
	InputKeyKeyKeyR
	InputKeyKeyKeyS
	InputKeyKeyKeyT
	InputKeyKeyKeyU
	InputKeyKeyKeyV
	InputKeyKeyKeyW
	InputKeyKeyKeyX
	InputKeyKeyKeyY
	InputKeyKeyKeyZ
	InputKeyKeyMinus
	InputKeyKeyPeriod
	InputKeyKeyQuote
	InputKeyKeySemicolon
	InputKeyKeySlash

	// "Functional Keys" § 3.1.2
	InputKeyKeyAltLeft
	InputKeyKeyAltRight
	InputKeyKeyBackspace
	InputKeyKeyCapsLock
	InputKeyKeyContextMenu
	InputKeyKeyControlLeft
	InputKeyKeyControlRight
	InputKeyKeyEnter
	InputKeyKeyMetaLeft
	InputKeyKeyMetaRight
	InputKeyKeyShiftLeft
	InputKeyKeyShiftRight
	InputKeyKeySpace
	InputKeyKeyTab
	InputKeyKeyConvert
	InputKeyKeyKanaMode
	InputKeyKeyNonConvert

	// "Control Pad Section" § 3.2
	InputKeyKeyDelete
	InputKeyKeyEnd
	InputKeyKeyHelp
	InputKeyKeyHome
	InputKeyKeyInsert
	InputKeyKeyPageDown
	InputKeyKeyPageUp

	// "Arrow Pad Section" § 3.3
	InputKeyKeyArrowDown
	InputKeyKeyArrowLeft
	InputKeyKeyArrowRight
	InputKeyKeyArrowUp

	// "Numpad Section" § 3.4
	InputKeyKeyNumLock
	InputKeyKeyNumpad0
	InputKeyKeyNumpad1
	InputKeyKeyNumpad2
	InputKeyKeyNumpad3
	InputKeyKeyNumpad4
	InputKeyKeyNumpad5
	InputKeyKeyNumpad6
	InputKeyKeyNumpad7
	InputKeyKeyNumpad8
	InputKeyKeyNumpad9
	InputKeyKeyNumpadAdd
	InputKeyKeyNumpadBackspace
	InputKeyKeyNumpadClear
	InputKeyKeyNumpadClearEntry
	InputKeyKeyNumpadComma
	InputKeyKeyNumpadDecimal
	InputKeyKeyNumpadDivide
	InputKeyKeyNumpadEnter
	InputKeyKeyNumpadEqual
	InputKeyKeyNumpadMemoryAdd
	InputKeyKeyNumpadMemoryClear
	InputKeyKeyNumpadMemoryRecall
	InputKeyKeyNumpadMemoryStore
	InputKeyKeyNumpadMemorySubtract
	InputKeyKeyNumpadMultiply
	InputKeyKeyNumpadParenLeft
	InputKeyKeyNumpadParenRight
	InputKeyKeyNumpadSubtract

	// > For numpads that provide keys not listed here, a code value string
	// > should be created by starting with "Numpad" and appending an
	// > appropriate description of the key.
	//
	// These numpad entries are distinguished by various encoding protocols
	// (legacy and Kitty) so we support them here in case the apprt can
	// produce them.
	InputKeyKeyNumpadSeparator
	InputKeyKeyNumpadUp
	InputKeyKeyNumpadDown
	InputKeyKeyNumpadRight
	InputKeyKeyNumpadLeft
	InputKeyKeyNumpadBegin
	InputKeyKeyNumpadHome
	InputKeyKeyNumpadEnd
	InputKeyKeyNumpadInsert
	InputKeyKeyNumpadDelete
	InputKeyKeyNumpadPageUp
	InputKeyKeyNumpadPageDown

	// "Function Section" § 3.5
	InputKeyKeyEscape
	InputKeyKeyF1
	InputKeyKeyF2
	InputKeyKeyF3
	InputKeyKeyF4
	InputKeyKeyF5
	InputKeyKeyF6
	InputKeyKeyF7
	InputKeyKeyF8
	InputKeyKeyF9
	InputKeyKeyF10
	InputKeyKeyF11
	InputKeyKeyF12
	InputKeyKeyF13
	InputKeyKeyF14
	InputKeyKeyF15
	InputKeyKeyF16
	InputKeyKeyF17
	InputKeyKeyF18
	InputKeyKeyF19
	InputKeyKeyF20
	InputKeyKeyF21
	InputKeyKeyF22
	InputKeyKeyF23
	InputKeyKeyF24
	InputKeyKeyF25
	InputKeyKeyFn
	InputKeyKeyFnLock
	InputKeyKeyPrintScreen
	InputKeyKeyScrollLock
	InputKeyKeyPause

	// "Media Keys" § 3.6
	InputKeyKeyBrowserBack
	InputKeyKeyBrowserFavorites
	InputKeyKeyBrowserForward
	InputKeyKeyBrowserHome
	InputKeyKeyBrowserRefresh
	InputKeyKeyBrowserSearch
	InputKeyKeyBrowserStop
	InputKeyKeyEject
	InputKeyKeyLaunchApp1
	InputKeyKeyLaunchApp2
	InputKeyKeyLaunchMail
	InputKeyKeyMediaPlayPause
	InputKeyKeyMediaSelect
	InputKeyKeyMediaStop
	InputKeyKeyMediaTrackNext
	InputKeyKeyMediaTrackPrevious
	InputKeyKeyPower
	InputKeyKeySleep
	InputKeyKeyAudioVolumeDown
	InputKeyKeyAudioVolumeMute
	InputKeyKeyAudioVolumeUp
	InputKeyKeyWakeUp

	// "Legacy, Non-standard, and Special Keys" § 3.7
	InputKeyKeyCopy
	InputKeyKeyCut
	InputKeyKeyPaste

	inputKeyKeyCount = iota
)

// String returns the Zig tag name (@tagName), e.g. "key_a", "fn"; also the
// name std.meta.stringToEnum matches in fromW3C. A value outside the enum
// panics (index out of range), as @tagName of an invalid value does.
func (self InputKeyKey) String() string { return inputKeyKeyNames[self] }

// inputKeyKeyNames holds the Zig tag names, indexed by value (generated
// from key.zig).
var inputKeyKeyNames = [inputKeyKeyCount]string{
	"unidentified",
	"backquote",
	"backslash",
	"bracket_left",
	"bracket_right",
	"comma",
	"digit_0",
	"digit_1",
	"digit_2",
	"digit_3",
	"digit_4",
	"digit_5",
	"digit_6",
	"digit_7",
	"digit_8",
	"digit_9",
	"equal",
	"intl_backslash",
	"intl_ro",
	"intl_yen",
	"key_a",
	"key_b",
	"key_c",
	"key_d",
	"key_e",
	"key_f",
	"key_g",
	"key_h",
	"key_i",
	"key_j",
	"key_k",
	"key_l",
	"key_m",
	"key_n",
	"key_o",
	"key_p",
	"key_q",
	"key_r",
	"key_s",
	"key_t",
	"key_u",
	"key_v",
	"key_w",
	"key_x",
	"key_y",
	"key_z",
	"minus",
	"period",
	"quote",
	"semicolon",
	"slash",
	"alt_left",
	"alt_right",
	"backspace",
	"caps_lock",
	"context_menu",
	"control_left",
	"control_right",
	"enter",
	"meta_left",
	"meta_right",
	"shift_left",
	"shift_right",
	"space",
	"tab",
	"convert",
	"kana_mode",
	"non_convert",
	"delete",
	"end",
	"help",
	"home",
	"insert",
	"page_down",
	"page_up",
	"arrow_down",
	"arrow_left",
	"arrow_right",
	"arrow_up",
	"num_lock",
	"numpad_0",
	"numpad_1",
	"numpad_2",
	"numpad_3",
	"numpad_4",
	"numpad_5",
	"numpad_6",
	"numpad_7",
	"numpad_8",
	"numpad_9",
	"numpad_add",
	"numpad_backspace",
	"numpad_clear",
	"numpad_clear_entry",
	"numpad_comma",
	"numpad_decimal",
	"numpad_divide",
	"numpad_enter",
	"numpad_equal",
	"numpad_memory_add",
	"numpad_memory_clear",
	"numpad_memory_recall",
	"numpad_memory_store",
	"numpad_memory_subtract",
	"numpad_multiply",
	"numpad_paren_left",
	"numpad_paren_right",
	"numpad_subtract",
	"numpad_separator",
	"numpad_up",
	"numpad_down",
	"numpad_right",
	"numpad_left",
	"numpad_begin",
	"numpad_home",
	"numpad_end",
	"numpad_insert",
	"numpad_delete",
	"numpad_page_up",
	"numpad_page_down",
	"escape",
	"f1",
	"f2",
	"f3",
	"f4",
	"f5",
	"f6",
	"f7",
	"f8",
	"f9",
	"f10",
	"f11",
	"f12",
	"f13",
	"f14",
	"f15",
	"f16",
	"f17",
	"f18",
	"f19",
	"f20",
	"f21",
	"f22",
	"f23",
	"f24",
	"f25",
	"fn",
	"fn_lock",
	"print_screen",
	"scroll_lock",
	"pause",
	"browser_back",
	"browser_favorites",
	"browser_forward",
	"browser_home",
	"browser_refresh",
	"browser_search",
	"browser_stop",
	"eject",
	"launch_app_1",
	"launch_app_2",
	"launch_mail",
	"media_play_pause",
	"media_select",
	"media_stop",
	"media_track_next",
	"media_track_previous",
	"power",
	"sleep",
	"audio_volume_down",
	"audio_volume_mute",
	"audio_volume_up",
	"wake_up",
	"copy",
	"cut",
	"paste",
}

// InputKeyKeyValues stands for Zig's `std.enums.values(Key)`: every key, in
// declaration order. Not a Zig declaration.
// Returns a new slice on each call.
func InputKeyKeyValues() []InputKeyKey {
	values := make([]InputKeyKey, inputKeyKeyCount)
	for i := range values {
		values[i] = InputKeyKey(i)
	}
	return values
}

// InputKeyKeyFromASCII is key.Key.fromASCII: converts an ASCII character to
// a key, if possible. This returns null if the character is unknown.
//
// Note that this can't distinguish between physical keys, i.e. '0'
// may be from the number row or the keypad, but it always maps
// to '.zero'.
//
// This is what we want, we want people to create keybindings that
// are independent of the physical key.
//
// Zig: `switch (ch) { inline else => |comptime_ch| comptime ... }`: one
// load from inputKeyKeyFromASCIITable.
func InputKeyKeyFromASCII(ch uint8) Optional[InputKeyKey] {
	return inputKeyKeyFromASCIITable[ch]
}

// inputKeyKeyFromASCIITable is fromASCII's comptime result for every u8,
// filled once at start-up by inputKeyKeyFromASCIIInit.
var inputKeyKeyFromASCIITable = inputKeyKeyFromASCIIInit()

// inputKeyKeyFromASCIIInit runs fromASCII's comptime `result:` block for
// every value of ch.
func inputKeyKeyFromASCIIInit() [256]Optional[InputKeyKey] {
	var table [256]Optional[InputKeyKey]
	for comptime_ch := range table {
		table[comptime_ch] = func() Optional[InputKeyKey] {
			for _, entry := range inputKeyKeyCodepointMap {
				// No ASCII characters should ever map to a keypad key
				if entry.x1.Keypad() {
					continue
				}

				if entry.x0 == uint32(comptime_ch) {
					return Some(entry.x1)
				}
			}

			return None[InputKeyKey]()
		}()
	}
	return table
}

// InputKeyKeyFromW3C is key.Key.fromW3C: converts a W3C key code to a
// Ghostty key enum value.
//
// All required W3C key codes are supported, but there are a number of
// non-standard key codes that are not supported. In the case the value is
// invalid or unsupported, this function will return null.
func InputKeyKeyFromW3C(code []uint8) Optional[InputKeyKey] {
	var result [128]uint8

	// If the code is bigger than our buffer it can't possibly match.
	if len(code) > len(result) {
		return None[InputKeyKey]()
	}

	// First just check the whole thing lowercased, this is the simple case
	if key := inputKeyStringToEnumKey(
		inputKeyAsciiLowerString(result[:], code),
	); key.Valid {
		return key
	}

	// We need to convert FooBar to foo_bar
	fbs := inputKeyModsFixedBufferStreamInit(result[:])
	w := &fbs
	for i, ch := range code {
		switch {
		case ch >= 'a' && ch <= 'z':
			if w.writeByte(ch) != nil {
				return None[InputKeyKey]()
			}

		// Caps and numbers trigger underscores
		case ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9':
			if i > 0 {
				if w.writeByte('_') != nil {
					return None[InputKeyKey]()
				}
			}
			if w.writeByte(inputKeyAsciiToLower(ch)) != nil {
				return None[InputKeyKey]()
			}

		// We don't know of any key codes that aren't alphanumeric.
		default:
			return None[InputKeyKey]()
		}
	}

	return inputKeyStringToEnumKey(fbs.getWritten())
}

// inputKeyStringToEnumKey is std.meta.stringToEnum(Key, s). Key has more
// than 100 fields, so Zig 0.15.2 compares s with each tag name in
// declaration order (`inline for` + `mem.eql`) and returns the first match.
// Not a Ghostty declaration.
func inputKeyStringToEnumKey(s []uint8) Optional[InputKeyKey] {
	for i := range inputKeyKeyNames {
		if string(s) == inputKeyKeyNames[i] {
			return Some(InputKeyKey(i))
		}
	}
	return None[InputKeyKey]()
}

// inputKeyAsciiLowerString is std.ascii.lowerString (Zig 0.15.2): writes a
// lower case copy of ascii_string to output and returns that part of
// output. output must be at least as long (asserted, as std does).
func inputKeyAsciiLowerString(output []uint8, ascii_string []uint8) []uint8 {
	assert(len(output) >= len(ascii_string))
	for i, c := range ascii_string {
		output[i] = inputKeyAsciiToLower(c)
	}
	return output[0:len(ascii_string)]
}

// inputKeyAsciiToLower is std.ascii.toLower (Zig 0.15.2): lowercases the
// character and returns it as-is if already lowercase or not a letter.
func inputKeyAsciiToLower(c uint8) uint8 {
	var mask uint8
	if c >= 'A' && c <= 'Z' { // isUpper
		mask = 1 << 5
	}
	return c | mask
}

// inputKeyAsciiToUpper is std.ascii.toUpper (Zig 0.15.2): uppercases the
// character and returns it as-is if already uppercase or not a letter.
func inputKeyAsciiToUpper(c uint8) uint8 {
	var mask uint8
	if c >= 'a' && c <= 'z' { // isLower
		mask = 1 << 5
	}
	return c ^ mask
}

// W3c is key.Key.w3c: converts a Ghostty key enum value to a W3C key code.
//
// Zig: `switch (self) { inline else => |tag| comptime w3c: { ... } }`: one
// load from inputKeyKeyW3cTable. The caller must not modify the result.
func (self InputKeyKey) W3c() []uint8 {
	return inputKeyKeyW3cTable[self]
}

// inputKeyKeyW3cTable is w3c's comptime result for every key, filled once
// at start-up by inputKeyKeyW3cInit.
var inputKeyKeyW3cTable = inputKeyKeyW3cInit()

// inputKeyKeyW3cInit runs w3c's comptime `w3c:` block for every key.
func inputKeyKeyW3cInit() [inputKeyKeyCount][]uint8 {
	var table [inputKeyKeyCount][]uint8
	for t := range table {
		tag := InputKeyKey(t)
		name := tag.String()

		var buf [128]uint8
		fbs := inputKeyModsFixedBufferStreamInit(buf[:])
		w := &fbs
		i := 0
		for i < len(name) {
			if i == 0 {
				if w.writeByte(inputKeyAsciiToUpper(name[i])) != nil {
					panic("unreachable")
				}
			} else if name[i] == '_' {
				i += 1
				if w.writeByte(inputKeyAsciiToUpper(name[i])) != nil {
					panic("unreachable")
				}
			} else {
				if w.writeByte(name[i]) != nil {
					panic("unreachable")
				}
			}

			i += 1
		}

		written := buf
		result := written[0:len(fbs.getWritten())]
		table[t] = result
	}
	return table
}

// Printable is key.Key.printable: true if this key represents a printable
// character.
func (self InputKeyKey) Printable() bool {
	return inputKeyKeyPrintableTable[self]
}

// inputKeyKeyPrintableTable is printable's comptime result for every key,
// filled once at start-up by inputKeyKeyPrintableInit.
var inputKeyKeyPrintableTable = inputKeyKeyPrintableInit()

// inputKeyKeyPrintableInit runs printable's comptime `result:` block for
// every key.
func inputKeyKeyPrintableInit() [inputKeyKeyCount]bool {
	var table [inputKeyKeyCount]bool
	for t := range table {
		tag := InputKeyKey(t)
		table[t] = func() bool {
			for _, entry := range inputKeyKeyCodepointMap {
				if entry.x1 == tag {
					return true
				}
			}

			return false
		}()
	}
	return table
}

// Modifier is key.Key.modifier: true if this key is a modifier.
func (self InputKeyKey) Modifier() bool {
	switch self {
	case InputKeyKeyShiftLeft,
		InputKeyKeyControlLeft,
		InputKeyKeyAltLeft,
		InputKeyKeyMetaLeft,
		InputKeyKeyShiftRight,
		InputKeyKeyControlRight,
		InputKeyKeyAltRight,
		InputKeyKeyMetaRight:
		return true

	default:
		return false
	}
}

// ShouldBeRemappable is key.Key.shouldBeRemappable: whether this key should
// be remappable by the operating system.
//
// On certain OSes (namely Linux and the BSDs) certain keys like the
// functional keys are expected to be remappable by the user, such as
// in the very common use case of swapping the Caps Lock key with the
// Escape key with the XKB option `caps:swapescape`.
//
// However, the way XKB implements this is by essentially acting as a
// software key remapper that destroys all information about the original
// physical key, leading to very annoying bugs like #7309 where the
// physical key `XKB_KEY_c` gets remapped into `XKB_KEY_Cyrillic_tse`,
// which causes all of our physical key handling to completely break down.
// _Very naughty._
//
// As a compromise, given that writing system keys (§3.1.1) comprise the
// majority of keys that "change meaning [...] based on the current locale
// and keyboard layout", we allow all other keys to be remapped by default
// since they should be fairly harmless. We might consider making this
// configurable, but for now this should at least placate most people.
func (self InputKeyKey) ShouldBeRemappable() bool {
	switch self {
	// "Writing System Keys" § 3.1.1
	case InputKeyKeyBackquote,
		InputKeyKeyBackslash,
		InputKeyKeyBracketLeft,
		InputKeyKeyBracketRight,
		InputKeyKeyComma,
		InputKeyKeyDigit0,
		InputKeyKeyDigit1,
		InputKeyKeyDigit2,
		InputKeyKeyDigit3,
		InputKeyKeyDigit4,
		InputKeyKeyDigit5,
		InputKeyKeyDigit6,
		InputKeyKeyDigit7,
		InputKeyKeyDigit8,
		InputKeyKeyDigit9,
		InputKeyKeyEqual,
		InputKeyKeyIntlBackslash,
		InputKeyKeyIntlRo,
		InputKeyKeyIntlYen,
		InputKeyKeyKeyA,
		InputKeyKeyKeyB,
		InputKeyKeyKeyC,
		InputKeyKeyKeyD,
		InputKeyKeyKeyE,
		InputKeyKeyKeyF,
		InputKeyKeyKeyG,
		InputKeyKeyKeyH,
		InputKeyKeyKeyI,
		InputKeyKeyKeyJ,
		InputKeyKeyKeyK,
		InputKeyKeyKeyL,
		InputKeyKeyKeyM,
		InputKeyKeyKeyN,
		InputKeyKeyKeyO,
		InputKeyKeyKeyP,
		InputKeyKeyKeyQ,
		InputKeyKeyKeyR,
		InputKeyKeyKeyS,
		InputKeyKeyKeyT,
		InputKeyKeyKeyU,
		InputKeyKeyKeyV,
		InputKeyKeyKeyW,
		InputKeyKeyKeyX,
		InputKeyKeyKeyY,
		InputKeyKeyKeyZ,
		InputKeyKeyMinus,
		InputKeyKeyPeriod,
		InputKeyKeyQuote,
		InputKeyKeySemicolon,
		InputKeyKeySlash:
		return false

	default:
		return true
	}
}

// Keypad is key.Key.keypad: returns true if this is a keypad key (its tag
// name starts with "numpad_").
func (self InputKeyKey) Keypad() bool {
	return inputKeyKeyKeypadTable[self]
}

// inputKeyKeyKeypadTable is keypad's comptime result for every key
// (`comptime std.mem.startsWith(u8, @tagName(tag), "numpad_")`), filled
// once at start-up by inputKeyKeyKeypadInit.
var inputKeyKeyKeypadTable = inputKeyKeyKeypadInit()

// inputKeyKeyKeypadInit computes keypad's comptime result for every key.
func inputKeyKeyKeypadInit() [inputKeyKeyCount]bool {
	var table [inputKeyKeyCount]bool
	for t := range table {
		name := InputKeyKey(t).String()
		table[t] = strings.HasPrefix(name, "numpad_")
	}
	return table
}

// Codepoint is key.Key.codepoint: returns the codepoint representing this
// key, or null if the key is not printable.
//
// Zig: ?u21.
func (self InputKeyKey) Codepoint() Optional[uint32] {
	return inputKeyKeyCodepointTable[self]
}

// inputKeyKeyCodepointTable is codepoint's comptime result for every key,
// filled once at start-up by inputKeyKeyCodepointInit.
var inputKeyKeyCodepointTable = inputKeyKeyCodepointInit()

// inputKeyKeyCodepointInit runs codepoint's comptime `result:` block for
// every key.
func inputKeyKeyCodepointInit() [inputKeyKeyCount]Optional[uint32] {
	var table [inputKeyKeyCount]Optional[uint32]
	for t := range table {
		tag := InputKeyKey(t)
		table[t] = func() Optional[uint32] {
			for _, entry := range inputKeyKeyCodepointMap {
				if entry.x1 == tag {
					return Some(entry.x0)
				}
			}

			return None[uint32]()
		}()
	}
	return table
}

// CtrlOrSuper is key.Key.ctrlOrSuper: true if this key is one of the left
// or right versions of super (MacOS) or ctrl.
//
// Darwin: see the file header. Off Darwin: control_left or control_right.
func (self InputKeyKey) CtrlOrSuper() bool {
	if inputKeyIsDarwin() {
		return self == InputKeyKeyMetaLeft || self == InputKeyKeyMetaRight
	}
	return self == InputKeyKeyControlLeft || self == InputKeyKeyControlRight
}

// LeftOrRightShift is key.Key.leftOrRightShift: true if this key is either
// left or right shift.
func (self InputKeyKey) LeftOrRightShift() bool {
	return self == InputKeyKeyShiftLeft || self == InputKeyKeyShiftRight
}

// LeftOrRightAlt is key.Key.leftOrRightAlt: true if this key is either left
// or right alt.
func (self InputKeyKey) LeftOrRightAlt() bool {
	return self == InputKeyKeyAltLeft || self == InputKeyKeyAltRight
}

// inputKeyKeyCodepointMapEntry is the element type of Key.codepoint_map:
// `struct { u21, Key }`. Not a Zig declaration (an anonymous tuple).
type inputKeyKeyCodepointMapEntry struct {
	// Zig: u21.
	x0 uint32
	x1 InputKeyKey
}

// inputKeyKeyCodepointMap is key.Key.codepoint_map.
//
// Upstream's 65 entries in upstream's order (generated from key.zig).
var inputKeyKeyCodepointMap = []inputKeyKeyCodepointMapEntry{
	{'a', InputKeyKeyKeyA},
	{'b', InputKeyKeyKeyB},
	{'c', InputKeyKeyKeyC},
	{'d', InputKeyKeyKeyD},
	{'e', InputKeyKeyKeyE},
	{'f', InputKeyKeyKeyF},
	{'g', InputKeyKeyKeyG},
	{'h', InputKeyKeyKeyH},
	{'i', InputKeyKeyKeyI},
	{'j', InputKeyKeyKeyJ},
	{'k', InputKeyKeyKeyK},
	{'l', InputKeyKeyKeyL},
	{'m', InputKeyKeyKeyM},
	{'n', InputKeyKeyKeyN},
	{'o', InputKeyKeyKeyO},
	{'p', InputKeyKeyKeyP},
	{'q', InputKeyKeyKeyQ},
	{'r', InputKeyKeyKeyR},
	{'s', InputKeyKeyKeyS},
	{'t', InputKeyKeyKeyT},
	{'u', InputKeyKeyKeyU},
	{'v', InputKeyKeyKeyV},
	{'w', InputKeyKeyKeyW},
	{'x', InputKeyKeyKeyX},
	{'y', InputKeyKeyKeyY},
	{'z', InputKeyKeyKeyZ},
	{'0', InputKeyKeyDigit0},
	{'1', InputKeyKeyDigit1},
	{'2', InputKeyKeyDigit2},
	{'3', InputKeyKeyDigit3},
	{'4', InputKeyKeyDigit4},
	{'5', InputKeyKeyDigit5},
	{'6', InputKeyKeyDigit6},
	{'7', InputKeyKeyDigit7},
	{'8', InputKeyKeyDigit8},
	{'9', InputKeyKeyDigit9},
	{';', InputKeyKeySemicolon},
	{' ', InputKeyKeySpace},
	{'\'', InputKeyKeyQuote},
	{',', InputKeyKeyComma},
	{'`', InputKeyKeyBackquote},
	{'.', InputKeyKeyPeriod},
	{'/', InputKeyKeySlash},
	{'-', InputKeyKeyMinus},
	{'=', InputKeyKeyEqual},
	{'[', InputKeyKeyBracketLeft},
	{']', InputKeyKeyBracketRight},
	{'\\', InputKeyKeyBackslash},

	// Control characters
	{'\t', InputKeyKeyTab},

	// Keypad entries. We just assume keypad with the numpad_ prefix
	// so that has some special meaning. These must also always be last,
	// so that our `fromASCII` function doesn't accidentally map them
	// over normal numerics and other keys.
	{'0', InputKeyKeyNumpad0},
	{'1', InputKeyKeyNumpad1},
	{'2', InputKeyKeyNumpad2},
	{'3', InputKeyKeyNumpad3},
	{'4', InputKeyKeyNumpad4},
	{'5', InputKeyKeyNumpad5},
	{'6', InputKeyKeyNumpad6},
	{'7', InputKeyKeyNumpad7},
	{'8', InputKeyKeyNumpad8},
	{'9', InputKeyKeyNumpad9},
	{'.', InputKeyKeyNumpadDecimal},
	{'/', InputKeyKeyNumpadDivide},
	{'*', InputKeyKeyNumpadMultiply},
	{'-', InputKeyKeyNumpadSubtract},
	{'+', InputKeyKeyNumpadAdd},
	{'=', InputKeyKeyNumpadEqual},
}

// InputKeyCtrlOrSuper is key.ctrlOrSuper: this sets either "ctrl" or
// "super" to true (but not both) on mods depending on if the build target
// is Mac or not. On Mac, we default to super (i.e. super+c for copy) and on
// non-Mac we default to ctrl (i.e. ctrl+c for copy).
//
// Darwin: see the file header. Off Darwin it sets ctrl.
func InputKeyCtrlOrSuper(mods InputKeyMods) InputKeyMods {
	copy := mods
	if inputKeyIsDarwin() {
		copy.super = true
	} else {
		copy.ctrl = true
	}

	return copy
}

// inputKeyIsDarwin stands for the comptime check
// `builtin.target.os.tag.isDarwin()` in Key.ctrlOrSuper and ctrlOrSuper.
// It returns platformIsDarwin() (platform.go).
func inputKeyIsDarwin() bool { return platformIsDarwin() }
