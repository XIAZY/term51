// WIREFRAME for src/input/key_encode.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: key_encode.zig lives in src/input/, so the file is
// `input_key_encode.go` and every declaration takes the prefix
// `InputKeyEncode` (`inputKeyEncode` for private ones):
// `key_encode.Options` is `InputKeyEncodeOptions`, `Options.default` is
// `InputKeyEncodeOptionsDefault`, `Options.fromTerminal` is
// `InputKeyEncodeOptionsFromTerminal`, `key_encode.encode` is
// `InputKeyEncodeEncode`. The private functions are `inputKeyEncodeKitty`,
// `inputKeyEncodeLegacy`, `inputKeyEncodeLegacyAltPrefix`,
// `inputKeyEncodeCopyToBuf`, `inputKeyEncodePcStyleFunctionKey`,
// `inputKeyEncodeCtrlSeq`, `inputKeyEncodeIsControl`,
// `inputKeyEncodeIsControlUtf8`. The private types are
// `inputKeyEncodeCsiUMods`, `inputKeyEncodeKittyMods`,
// `inputKeyEncodeKittySequence` and `inputKeyEncodeKittySequenceEvent`
// (`KittySequence.Event`, values `inputKeyEncodeKittySequenceEventNone`,
// `...Press`, `...Repeat`, `...Release`). Their `pub fn` methods are
// exported (`Int`, `SeqInt`, `PreventsText`, `Encode`), private ones keep
// the Zig name (`encodeFull`, `encodeSpecial`); the `fromInput` functions
// (no `self`) are `inputKeyEncodeCsiUModsFromInput` and
// `inputKeyEncodeKittyModsFromInput`.
//
// Declarations appear in the same order as in key_encode.zig.
//
// # Encoding a key event
//
// Zig:
//
//	var buf: [128]u8 = undefined;
//	var writer: std.Io.Writer = .fixed(&buf);
//	try encode(&writer, .{ .key = .key_a, .utf8 = "a" }, .{});
//	try testing.expectEqualStrings("a", writer.buffered());
//
// Go:
//
//	var buf [128]uint8
//	writer := oscWriterFixed(buf[:])
//	err := InputKeyEncodeEncode(&writer, InputKeyKeyEvent{
//		action: InputKeyActionPress, key: InputKeyKeyKeyA,
//		utf8: []uint8("a")}, InputKeyEncodeOptions{})
//	// err == nil; writer.buffered() is "a"
//
// `kitty` and `legacy` have the same shape. A fixed writer that is too
// small fails with ErrWriteFailed (osc.go), as Zig's error.WriteFailed.
//
// # std.Io.Writer
//
// Every `*std.Io.Writer` parameter is `*oscWriter` (osc.go), the package's
// stand-in for std.Io.Writer; `.fixed(buf)` is `oscWriterFixed(buf)`,
// `writer.buffered()` is `writer.buffered()`, `std.Io.Writer.Error` is
// ErrWriteFailed. The bodies need `writeAll`, `writeByte` and `print`
// with `{d}`, `{c}` and `{}` (decimal); the implementation adds them to
// osc.go with Zig's names.
//
// # Widths
//
// `u21` is `uint32` (codepoints: `KittySequence.key`, `alternates`,
// `ctrlSeq`'s `unshifted_codepoint`, `isControl`'s `cp`). Packed-struct
// integers: `CsiUMods.int` u3 and `seqInt` u4 are `uint8`;
// `KittyMods.int` u8 is `uint8` and `seqInt` u9 is `uint16`;
// `KittySequence.Event` enum(u2) is `uint8`.
//
// # Packed structs
//
// `CsiUMods` (packed struct(u3)) and `KittyMods` (packed struct(u8)) are
// ordinary Go structs with the same fields in the same order, comparable
// with `==`. Zig's own `int` methods are `Int`. The tests also build them
// with `@bitCast` from an integer; `inputKeyEncodeCsiUModsFromInt` and
// `inputKeyEncodeKittyModsFromInt` stand for that cast (bit 0 is the first
// field).
//
// # Darwin
//
// Four places are compiled differently on Apple targets:
//   - `kitty`, report_associated: `alt_prevents_text` depends on
//     `macos_option_as_alt` when `builtin.os.tag == .macos`, else it is
//     true;
//   - `legacy`, modifyOtherKeys state 2: alt is dropped from the mods
//     according to `macos_option_as_alt` when
//     `builtin.target.os.tag.isDarwin()`;
//   - `legacy`: with `.macos` and super pressed, text is not encoded;
//   - `legacyAltPrefix`: with `.macos`, `macos_option_as_alt` decides
//     whether alt is an ESC prefix.
// The port runs on Linux and QNX and so follows the non-Darwin branch
// everywhere: `macos_option_as_alt` is never read, alt always prevents
// associated text, alt is kept in modifyOtherKeys, super+text encodes the
// text. The two checks are `inputKeyEncodeIsMacos()` (`builtin.os.tag ==
// .macos`) and `inputKeyEncodeIsDarwin()` (`isDarwin()`: macOS, iOS,
// tvOS, watchOS, visionOS), stubs standing for the comptime tests; Darwin
// behaviour would be selected by making them return true (runtime.GOOS
// "darwin", and "darwin" or "ios"). A Zig test that returns SkipZigTest
// off Darwin becomes a Go test that calls `t.Skip` when the matching stub
// returns false (until the stubs are implemented they panic, like every
// other wireframe function).
//
// # Declarations that are not Zig declarations
//
//   - `inputKeyEncodeCsiUModsFromInt`, `inputKeyEncodeKittyModsFromInt`
//     (above);
//   - `inputKeyEncodeKittySequenceEventCount` and the `String` method of
//     `KittySequence.Event`;
//   - `inputKeyEncodeIsMacos`, `inputKeyEncodeIsDarwin` (above).
//
// # Omitted
//
// Aliases and imports with nothing to declare: `std`, `builtin`,
// `testing`, `KittyFlags` (use KittyKeyFlags, kitty_key.go), `OptionAsAlt`
// (use InputConfigOptionAsAlt), `Terminal`, `function_keys`, `key`,
// `KittyEntry` (use InputKittyEntry), `kitty_entries` (use
// InputKittyEntries). The commented-out log line in `encode`.
// No C ABI declaration exists in this file.
//
// # Zig defaults a Go literal does not give
//
// None in this file: every field default is the Go zero value.
//   - `Options`: every field defaults to false, `kitty_flags` to `.disabled`
//     (KittyKeyFlagsDisabled, the zero value) and `macos_option_as_alt` to
//     `.false` (InputConfigOptionAsAltFalse, 0). So `Options{}` (Zig `.{}`)
//     equals `Options.default` (InputKeyEncodeOptionsDefault).
//   - `CsiUMods`, `KittyMods`: every field defaults to false.
//   - `KittySequence`: `mods` .{}, `event` .none (0), `alternates`
//     `.{ null, null }`, `text` "" are all the zero value; `key` and `final`
//     have no default.
// NOTE for every test: the KeyEvent passed to the encoders has
// `action` defaulting to `.press` in Zig, which is NOT the Go zero value;
// see input_key.go.

package terminal

// InputKeyEncodeOptions is key_encode.Options: options that affect key
// encoding behavior. This is a mix of behavior from terminal state as well
// as application configuration.
//
// Every Zig default is the Go zero value; see the file header.
type InputKeyEncodeOptions struct {
	// Terminal DEC mode 1
	//
	// Zig default: false.
	cursor_key_application bool

	// Terminal DEC mode 66
	//
	// Zig default: false.
	keypad_key_application bool

	// Terminal DEC mode 1035
	//
	// Zig default: false.
	ignore_keypad_with_numlock bool

	// Terminal DEC mode 1036
	//
	// Zig default: false.
	alt_esc_prefix bool

	// xterm "modifyOtherKeys mode 2". Details here:
	// https://invisible-island.net/xterm/modified-keys.html
	//
	// Zig default: false.
	modify_other_keys_state_2 bool

	// Kitty keyboard protocol flags.
	//
	// Zig default: .disabled (the zero value).
	kitty_flags KittyKeyFlags

	// Determines whether the "option" key on macOS is treated
	// as "alt" or not. See the Ghostty `macos_option-as-alt` config
	// docs for a more detailed description of why this is needed.
	//
	// Zig default: .false (the zero value).
	macos_option_as_alt InputConfigOptionAsAlt
}

// InputKeyEncodeOptionsDefault is key_encode.Options.default: every field
// false, kitty_flags .disabled, macos_option_as_alt .false. That is the
// zero value.
var InputKeyEncodeOptionsDefault = InputKeyEncodeOptions{}

// InputKeyEncodeOptionsFromTerminal is key_encode.Options.fromTerminal:
// initialize our options from the terminal state.
//
// Note that `macos_option_as_alt` cannot be determined from
// terminal state so it must be set manually after this call.
func InputKeyEncodeOptionsFromTerminal(t *Terminal) InputKeyEncodeOptions {
	panic(notImplemented)
}

// InputKeyEncodeEncode is key_encode.encode: encode the key event to the
// writer in the proper format given the options. For example, this will
// properly encode a key press such as "ctrl+A" to Kitty format if Kitty
// encoding is enabled.
//
// Not all key events will result in output. It is up to the caller
// to use a writer that can track whether any output was written if
// they care about that.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
func InputKeyEncodeEncode(writer *oscWriter, event InputKeyKeyEvent, opts InputKeyEncodeOptions) error {
	panic(notImplemented)
}

// inputKeyEncodeKitty is key_encode.kitty: perform Kitty keyboard protocol
// encoding of the key event.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
// Darwin: see the file header (report_associated).
func inputKeyEncodeKitty(writer *oscWriter, event InputKeyKeyEvent, opts InputKeyEncodeOptions) error {
	panic(notImplemented)
}

// inputKeyEncodeLegacy is key_encode.legacy: perform legacy encoding of the
// key event. "Legacy" in this case is referring to the behavior of
// traditional terminals, plus xterm's `modifyOtherKeys`, plus Paul Evans's
// "fixterms" spec. These together combine the legacy protocol because
// they're all meant to be extensions that do not change any existing
// behavior and therefore safe to combine.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
// Darwin: see the file header (modifyOtherKeys alt, super+text).
func inputKeyEncodeLegacy(writer *oscWriter, event InputKeyKeyEvent, opts InputKeyEncodeOptions) error {
	panic(notImplemented)
}

// inputKeyEncodeLegacyAltPrefix is key_encode.legacyAltPrefix.
//
// Zig: fn legacyAltPrefix(event, binding_mods, mods, opts) !?u8. The
// inferred error set is empty (the body has no `try`), so the error is
// always nil; it is kept because callers and tests `try` it.
// Darwin: see the file header.
func inputKeyEncodeLegacyAltPrefix(
	event InputKeyKeyEvent,
	binding_mods InputKeyMods,
	mods InputKeyMods,
	opts InputKeyEncodeOptions,
) (Optional[uint8], error) {
	panic(notImplemented)
}

// inputKeyEncodeCopyToBuf is key_encode.copyToBuf: a helper to memcpy a src
// value to a buffer and return the result.
//
// Zig: fn copyToBuf(buf: []u8, src: []const u8) ![]const u8. The result is
// buf[:len(src)]; the error is ErrOutOfMemory (dcs.go) when src does not
// fit. Not called anywhere in key_encode.zig; the project tests call it.
func inputKeyEncodeCopyToBuf(buf []uint8, src []uint8) ([]uint8, error) { panic(notImplemented) }

// inputKeyEncodePcStyleFunctionKey is key_encode.pcStyleFunctionKey:
// determines whether the key should be encoded in the xterm "PC-style
// Function Key" syntax (roughly). This is a hardcoded table of keys and
// modifiers that result in a specific sequence.
//
// Zig: ?[]const u8; the slice is an entry's `sequence` in
// function_keys.keys (static memory).
func inputKeyEncodePcStyleFunctionKey(
	keyval InputKeyKey,
	mods InputKeyMods,
	cursor_key_application bool,
	keypad_key_application_req bool,
	ignore_keypad_with_numlock bool,
	modify_other_keys bool, // True if state 2
) Optional[[]uint8] {
	panic(notImplemented)
}

// inputKeyEncodeCtrlSeq is key_encode.ctrlSeq: returns the C0 byte for the
// key event if it should be used. This converts a key event into the
// expected terminal behavior such as Ctrl+C turning into 0x03, amongst many
// other translations.
//
// This will return null if the key event should not be converted
// into a C0 byte. There are many cases for this and you should read
// the source code to understand them.
func inputKeyEncodeCtrlSeq(
	logical_key InputKeyKey,
	utf8 []uint8,
	unshifted_codepoint uint32,
	mods InputKeyMods,
) Optional[uint8] {
	panic(notImplemented)
}

// inputKeyEncodeIsControl is key_encode.isControl: returns true if this is
// an ASCII control character, matches libc implementation.
//
// Zig: cp is u21.
func inputKeyEncodeIsControl(cp uint32) bool { panic(notImplemented) }

// inputKeyEncodeIsControlUtf8 is key_encode.isControlUtf8: returns true if
// this string is comprised of a single control character. This returns
// false for multi-byte strings.
func inputKeyEncodeIsControlUtf8(str []uint8) bool { panic(notImplemented) }

// inputKeyEncodeCsiUMods is key_encode.CsiUMods: this is the bitmask for
// fixterm CSI u modifiers.
//
// Zig: packed struct(u3), every field default false.
type inputKeyEncodeCsiUMods struct {
	shift bool
	alt   bool
	ctrl  bool
}

// inputKeyEncodeCsiUModsFromInput is key_encode.CsiUMods.fromInput: convert
// an input mods value into the CSI u mods value.
func inputKeyEncodeCsiUModsFromInput(mods InputKeyMods) inputKeyEncodeCsiUMods {
	panic(notImplemented)
}

// Int is key_encode.CsiUMods.int: returns the raw int value of this packed
// struct.
//
// Zig: returns u3.
func (self inputKeyEncodeCsiUMods) Int() uint8 { panic(notImplemented) }

// SeqInt is key_encode.CsiUMods.seqInt: returns the integer value sent as
// part of the CSI u sequence. This adds 1 to the bitmask value as described
// in the spec.
//
// Zig: returns u4.
func (self inputKeyEncodeCsiUMods) SeqInt() uint8 { panic(notImplemented) }

// inputKeyEncodeCsiUModsFromInt stands for Zig's `@bitCast` from u3 to
// CsiUMods, the inverse of Int. Not a Zig declaration; only the tests cast.
func inputKeyEncodeCsiUModsFromInt(v uint8) inputKeyEncodeCsiUMods { panic(notImplemented) }

// inputKeyEncodeKittyMods is key_encode.KittyMods: this is the bitfields for
// Kitty modifiers.
//
// Zig: packed struct(u8), every field default false.
type inputKeyEncodeKittyMods struct {
	shift     bool
	alt       bool
	ctrl      bool
	super     bool
	hyper     bool
	meta      bool
	caps_lock bool
	num_lock  bool
}

// inputKeyEncodeKittyModsFromInput is key_encode.KittyMods.fromInput:
// convert an input mods value into the CSI u mods value. `action` and `k`
// are unused (Zig discards them with `_ =`).
func inputKeyEncodeKittyModsFromInput(action InputKeyAction, k InputKeyKey, mods InputKeyMods) inputKeyEncodeKittyMods {
	panic(notImplemented)
}

// PreventsText is key_encode.KittyMods.preventsText: returns true if the
// modifiers prevent printable text.
//
// The alt_prevents_text parameter determines whether or not the Alt
// modifier prevents printable text. On Linux, this is always true. On
// macOS, this is only true if macos-option-as-alt is set.
func (self inputKeyEncodeKittyMods) PreventsText(alt_prevents_text bool) bool {
	panic(notImplemented)
}

// Int is key_encode.KittyMods.int: returns the raw int value of this packed
// struct.
func (self inputKeyEncodeKittyMods) Int() uint8 { panic(notImplemented) }

// SeqInt is key_encode.KittyMods.seqInt: returns the integer value sent as
// part of the Kitty sequence. This adds 1 to the bitmask value as described
// in the spec.
//
// Zig: returns u9.
func (self inputKeyEncodeKittyMods) SeqInt() uint16 { panic(notImplemented) }

// inputKeyEncodeKittyModsFromInt stands for Zig's `@bitCast` from u8 to
// KittyMods, the inverse of Int. Not a Zig declaration; only the tests cast.
func inputKeyEncodeKittyModsFromInt(v uint8) inputKeyEncodeKittyMods { panic(notImplemented) }

// inputKeyEncodeKittySequence is key_encode.KittySequence: represents a
// kitty key sequence and has helpers for encoding it. The sequence from the
// Kitty specification:
//
// CSI unicode-key-code:alternate-key-codes ; modifiers:event-type ; text-as-codepoints u
type inputKeyEncodeKittySequence struct {
	// Zig: u21, no default.
	key uint32
	// Zig: no default.
	final uint8
	// Zig default: .{} (the zero value).
	mods inputKeyEncodeKittyMods
	// Zig default: .none (the zero value).
	event inputKeyEncodeKittySequenceEvent
	// Zig: [2]?u21, default .{ null, null } (the zero value).
	alternates [2]Optional[uint32]
	// Zig: []const u8, default "" (nil and empty behave the same).
	text []uint8
}

// inputKeyEncodeKittySequenceEvent is key_encode.KittySequence.Event:
// values for the event code (see "event-type" in above comment). Note that
// Kitty omits the ":1" for the press event but other terminals include it.
// We'll include it.
//
// Zig: enum(u2).
type inputKeyEncodeKittySequenceEvent uint8

const (
	inputKeyEncodeKittySequenceEventNone    inputKeyEncodeKittySequenceEvent = 0
	inputKeyEncodeKittySequenceEventPress   inputKeyEncodeKittySequenceEvent = 1
	inputKeyEncodeKittySequenceEventRepeat  inputKeyEncodeKittySequenceEvent = 2
	inputKeyEncodeKittySequenceEventRelease inputKeyEncodeKittySequenceEvent = 3

	inputKeyEncodeKittySequenceEventCount = 4
)

// String returns the Zig tag name (@tagName).
func (self inputKeyEncodeKittySequenceEvent) String() string { panic(notImplemented) }

// Encode is key_encode.KittySequence.encode.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
func (self inputKeyEncodeKittySequence) Encode(writer *oscWriter) error { panic(notImplemented) }

// encodeFull is key_encode.KittySequence.encodeFull.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
func (self inputKeyEncodeKittySequence) encodeFull(writer *oscWriter) error {
	panic(notImplemented)
}

// encodeSpecial is key_encode.KittySequence.encodeSpecial.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
func (self inputKeyEncodeKittySequence) encodeSpecial(writer *oscWriter) error {
	panic(notImplemented)
}

// inputKeyEncodeIsMacos stands for the comptime check `builtin.os.tag ==
// .macos` in kitty, legacy and legacyAltPrefix. The port follows the
// non-macOS branch (Linux, QNX); see the file header.
func inputKeyEncodeIsMacos() bool { panic(notImplemented) }

// inputKeyEncodeIsDarwin stands for the comptime check
// `builtin.target.os.tag.isDarwin()` in legacy. The port follows the
// non-Darwin branch (Linux, QNX); see the file header.
func inputKeyEncodeIsDarwin() bool { panic(notImplemented) }
