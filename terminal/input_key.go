// WIREFRAME for src/input/key.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented) and every table is left at its zero value. The
// implementation is added only after the project owner and the project lead
// both approve it. Do not fill in bodies before then. See the header of
// color.go for the rules.
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
// `fromASCII`, `w3c`, `printable` and `codepoint` compute their answer for
// every key at compile time (`inline else` + `comptime`); the Go
// implementation computes the same answer at run time (or in a table filled
// at start-up). `w3c` returns static memory in Zig; Go returns a []uint8
// the caller must not modify.
//
// # bindingHash
//
// `KeyEvent.bindingHash` is `std.hash.Wyhash.init(0)` fed with
// `std.hash.autoHash` of `key` (a c_int enum: its 4 bytes),
// `unshifted_codepoint` (a u21: autoHash hashes its bytes as a u21, i.e.
// `std.mem.asBytes` of the u21, 4 bytes with the top padding zero) and
// `mods.binding()` (a packed struct(u16): its 2 bytes). The implementation
// must reproduce Zig's values exactly if any test or corpus records them;
// the project tests only compare hashes with each other. As for
// `PageEntry.hash` (docs/mapping.md), Go's standard library has no Wyhash.
//
// # Darwin
//
// `Key.ctrlOrSuper` and the free function `ctrlOrSuper` have a comptime
// branch on `builtin.target.os.tag.isDarwin()`. The port runs on Linux and
// QNX and so follows the non-Darwin branch: `Key.ctrlOrSuper` is true for
// control_left/control_right, `ctrlOrSuper(mods)` sets `ctrl`. The check is
// `inputKeyIsDarwin()`, a stub standing for the comptime test; Darwin
// behaviour would be selected by making it return true (runtime.GOOS
// "darwin" or "ios").
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
func (self InputKeyKeyEvent) EffectiveMods() InputKeyMods { panic(notImplemented) }

// BindingHash is key.KeyEvent.bindingHash: returns a unique hash for this
// key event to be used for tracking uniquess specifically with bindings.
// This omits fields that are irrelevant for bindings. See the file header.
func (self InputKeyKeyEvent) BindingHash() uint64 { panic(notImplemented) }

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
func (self InputKeyAction) String() string { panic(notImplemented) }

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
// name std.meta.stringToEnum matches in fromW3C.
func (self InputKeyKey) String() string { panic(notImplemented) }

// InputKeyKeyValues stands for Zig's `std.enums.values(Key)`: every key, in
// declaration order. Not a Zig declaration.
func InputKeyKeyValues() []InputKeyKey { panic(notImplemented) }

// InputKeyKeyFromASCII is key.Key.fromASCII: converts an ASCII character to
// a key, if possible. This returns null if the character is unknown.
//
// Note that this can't distinguish between physical keys, i.e. '0'
// may be from the number row or the keypad, but it always maps
// to '.zero'.
//
// This is what we want, we want people to create keybindings that
// are independent of the physical key.
func InputKeyKeyFromASCII(ch uint8) Optional[InputKeyKey] { panic(notImplemented) }

// InputKeyKeyFromW3C is key.Key.fromW3C: converts a W3C key code to a
// Ghostty key enum value.
//
// All required W3C key codes are supported, but there are a number of
// non-standard key codes that are not supported. In the case the value is
// invalid or unsupported, this function will return null.
func InputKeyKeyFromW3C(code []uint8) Optional[InputKeyKey] { panic(notImplemented) }

// W3c is key.Key.w3c: converts a Ghostty key enum value to a W3C key code.
func (self InputKeyKey) W3c() []uint8 { panic(notImplemented) }

// Printable is key.Key.printable: true if this key represents a printable
// character.
func (self InputKeyKey) Printable() bool { panic(notImplemented) }

// Modifier is key.Key.modifier: true if this key is a modifier.
func (self InputKeyKey) Modifier() bool { panic(notImplemented) }

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
func (self InputKeyKey) ShouldBeRemappable() bool { panic(notImplemented) }

// Keypad is key.Key.keypad: returns true if this is a keypad key (its tag
// name starts with "numpad_").
func (self InputKeyKey) Keypad() bool { panic(notImplemented) }

// Codepoint is key.Key.codepoint: returns the codepoint representing this
// key, or null if the key is not printable.
//
// Zig: ?u21.
func (self InputKeyKey) Codepoint() Optional[uint32] { panic(notImplemented) }

// CtrlOrSuper is key.Key.ctrlOrSuper: true if this key is one of the left
// or right versions of super (MacOS) or ctrl.
//
// Darwin: see the file header. Off Darwin: control_left or control_right.
func (self InputKeyKey) CtrlOrSuper() bool { panic(notImplemented) }

// LeftOrRightShift is key.Key.leftOrRightShift: true if this key is either
// left or right shift.
func (self InputKeyKey) LeftOrRightShift() bool { panic(notImplemented) }

// LeftOrRightAlt is key.Key.leftOrRightAlt: true if this key is either left
// or right alt.
func (self InputKeyKey) LeftOrRightAlt() bool { panic(notImplemented) }

// inputKeyKeyCodepointMapEntry is the element type of Key.codepoint_map:
// `struct { u21, Key }`. Not a Zig declaration (an anonymous tuple).
type inputKeyKeyCodepointMapEntry struct {
	// Zig: u21.
	x0 uint32
	x1 InputKeyKey
}

// inputKeyKeyCodepointMap is key.Key.codepoint_map.
//
// Wireframe: left nil; the implementation fills it with upstream's 65
// entries in upstream's order ('a'..'z', '0'..'9', the ASCII punctuation
// keys, '\t' tab, then the keypad entries, which must stay last so that
// fromASCII never maps to them).
var inputKeyKeyCodepointMap []inputKeyKeyCodepointMapEntry

// InputKeyCtrlOrSuper is key.ctrlOrSuper: this sets either "ctrl" or
// "super" to true (but not both) on mods depending on if the build target
// is Mac or not. On Mac, we default to super (i.e. super+c for copy) and on
// non-Mac we default to ctrl (i.e. ctrl+c for copy).
//
// Darwin: see the file header. Off Darwin it sets ctrl.
func InputKeyCtrlOrSuper(mods InputKeyMods) InputKeyMods { panic(notImplemented) }

// inputKeyIsDarwin stands for the comptime check
// `builtin.target.os.tag.isDarwin()` in Key.ctrlOrSuper and ctrlOrSuper.
// The port follows the non-Darwin branch (Linux, QNX); see the file header.
func inputKeyIsDarwin() bool { panic(notImplemented) }
