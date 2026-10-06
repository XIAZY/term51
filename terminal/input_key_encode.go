// Ported from src/input/key_encode.zig (Ghostty v1.3.1).
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
// ErrWriteFailed. `writeAll` and `writeByte` are osc.go's ports.
// `writer.print(fmt, args)` is one `writer.writeAll` of the same bytes,
// with Zig's print line in a comment above (style.go and formatter.go
// precedent): `{d}` and `{}` of an integer are decimal (`itoa`, zig.go),
// `{c}` of a u8 is that raw byte (a value 0x80-0xFF is one byte, not
// UTF-8). Zig 0.15.2's print writes each literal piece with writeAll and
// each value with alignBuffer -> writeAll (no width is ever given here), and
// a fixed writer's drain copies what fits before failing, so every sequence
// of writes leaves the same prefix in the buffer as one write of the
// concatenation, and fails at the same point: the partial output with a
// buffer that is too small is identical.
//
// # std.unicode.Utf8View
//
// `std.unicode.Utf8View.init(s) catch ...` is `if !utf8.Valid(s) { ... }`
// (Go's validation rejects exactly what Zig's does: overlongs, surrogate
// halves, values above U+10FFFF, truncated sequences); `view.iterator()`
// and `it.nextCodepoint()` are the file-private `inputKeyEncodeUtf8Iterator`
// and its `nextCodepoint`, which decode with utf8.DecodeRune (identical to
// Zig on validated input) and return null at the end.
//
// # Control flow
//
// Labeled blocks left from inside nested statements (`entry:`,
// `preprocessing:`, `plain_text:`, `alternates:`, `associated:`, `pc_style:`,
// `modify_other:`, `alt:`, `should_modify:`, `csiu:`, `char:`, `shift:`,
// `text:`) are Go blocks left with `goto <label>`, the label placed right
// after the block (page.go/formatter.go precedent). Go labels have their
// own namespace, so a label may share its name with a variable (`entry`,
// `char`, `should_modify`), as in Zig. A `break :utf8` that leaves a Zig
// block whose only statement is a `switch` is a plain Go `break` out of the
// switch. `inline .enter, .backspace => |tag|` is one Go case; `tag` is
// `event.key` and `comptime tag == .backspace` an ordinary comparison.
// Exhaustive Zig switches on enums get `default: panic("unreachable")`.
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
// The library is platform-neutral: both branches are ported. The two checks
// are `inputKeyEncodeIsMacos()` (`builtin.os.tag == .macos`) and
// `inputKeyEncodeIsDarwin()` (`isDarwin()`), which return platformIsMacos()
// and platformIsDarwin() (platform.go). Off Apple platforms
// `macos_option_as_alt` is never read, alt always prevents associated text,
// alt is kept in modifyOtherKeys, and super+text encodes the text.
//
// # Declarations that are not Zig declarations
//
//   - `inputKeyEncodeCsiUModsFromInt`, `inputKeyEncodeKittyModsFromInt`
//     (above);
//   - `inputKeyEncodeKittySequenceEventCount` and the `String` method of
//     `KittySequence.Event`;
//   - `inputKeyEncodeIsMacos`, `inputKeyEncodeIsDarwin` (above);
//   - `inputKeyEncodeUtf8Iterator` (std.unicode.Utf8Iterator, above);
//   - `inputKeyEncodeCtrlSeqCtrlOnly`, ctrlSeq's
//     `comptime (key.Mods{ .ctrl = true }).int()`, computed once at
//     start-up;
//   - `inputKeyEncodeKittySequenceEventNames`, the names table of String.
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

import "unicode/utf8"

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
	return InputKeyEncodeOptions{
		alt_esc_prefix:             t.modes.Get(ModesModeAltEscPrefix),
		cursor_key_application:     t.modes.Get(ModesModeCursorKeys),
		keypad_key_application:     t.modes.Get(ModesModeKeypadKeys),
		ignore_keypad_with_numlock: t.modes.Get(ModesModeIgnoreKeypadWithNumlock),
		modify_other_keys_state_2:  t.flags.modify_other_keys_2,
		kitty_flags:                t.screens.active.kitty_keyboard.Current(),

		// These can't be known from the terminal state.
		macos_option_as_alt: InputConfigOptionAsAltFalse,
	}
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
	//std.log.warn("KEYENCODER event={} opts={}", .{ event, opts });
	if opts.kitty_flags.Int() != 0 {
		return inputKeyEncodeKitty(
			writer,
			event,
			opts,
		)
	}
	return inputKeyEncodeLegacy(
		writer,
		event,
		opts,
	)
}

// inputKeyEncodeKitty is key_encode.kitty: perform Kitty keyboard protocol
// encoding of the key event.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
// Darwin: see the file header (report_associated).
func inputKeyEncodeKitty(writer *oscWriter, event InputKeyKeyEvent, opts InputKeyEncodeOptions) error {
	// This should never happen but we'll check anyway.
	if opts.kitty_flags.Int() == 0 {
		return inputKeyEncodeLegacy(
			writer,
			event,
			opts,
		)
	}

	// We only processed "press" events unless report events is active
	if event.action == InputKeyActionRelease {
		if !opts.kitty_flags.report_events {
			return nil
		}

		// Enter, backspace, and tab do not report release events unless "report
		// all" is set
		if !opts.kitty_flags.report_all {
			switch event.key {
			case InputKeyKeyEnter, InputKeyKeyBackspace, InputKeyKeyTab:
				return nil
			default:
			}
		}
	}

	all_mods := event.mods
	effective_mods := event.EffectiveMods()
	binding_mods := effective_mods.Binding()

	// Find the entry for this key in the kitty table.
	var entry_ Optional[InputKittyEntry]
	{ // entry:
		// Functional or predefined keys
		for _, entry := range InputKittyEntries {
			if entry.key == event.key {
				entry_ = Some(entry)
				goto entry // break :entry entry
			}
		}

		// Otherwise, we use our unicode codepoint from UTF8. We
		// always use the unshifted value.
		if event.unshifted_codepoint > 0 {
			entry_ = Some(InputKittyEntry{
				key:      event.key,
				code:     event.unshifted_codepoint,
				final:    'u',
				modifier: false,
			})
			goto entry // break :entry .{ ... }
		}

		entry_ = Optional[InputKittyEntry]{} // break :entry null
	}
entry:

	{ // preprocessing:
		// When composing, the only keys sent are plain modifiers.
		if event.composing {
			if entry_.Valid {
				entry := entry_.Value
				if entry.modifier {
					goto preprocessing // break :preprocessing
				}
			}

			return nil
		}

		// IME confirmation still sends an enter key so if we have enter
		// and UTF8 text we just send it directly since we assume that is
		// what's happening. See legacy()'s similar logic for more details
		// on how to verify this.
		if len(event.utf8) > 0 { // utf8:
			switch event.key {
			default:
			case InputKeyKeyEnter, InputKeyKeyBackspace: // inline ... => |tag|
				tag := event.key
				// See legacy for why we handle this this way.
				if inputKeyEncodeIsControlUtf8(event.utf8) {
					break // break :utf8
				}
				if tag == InputKeyKeyBackspace {
					return nil
				}
				return writer.writeAll(event.utf8)
			}
		}

		// If we're reporting all then we always send CSI sequences.
		if !opts.kitty_flags.report_all {
			// Quote:
			// The only exceptions are the Enter, Tab and Backspace keys which
			// still generate the same bytes as in legacy mode this is to allow the
			// user to type and execute commands in the shell such as reset after a
			// program that sets this mode crashes without clearing it.
			//
			// Quote ("report all" mode):
			// Note that all keys are reported as escape codes, including Enter,
			// Tab, Backspace etc.
			if binding_mods.Empty() {
				switch event.key {
				case InputKeyKeyEnter:
					return writer.writeByte('\r')
				case InputKeyKeyTab:
					return writer.writeByte('\t')
				case InputKeyKeyBackspace:
					return writer.writeByte(0x7F)
				default:
				}
			}

			// Send plain-text non-modified text directly to the terminal.
			// We don't send release events because those are specially encoded.
			if len(event.utf8) > 0 &&
				binding_mods.Empty() &&
				event.action != InputKeyActionRelease { // plain_text:
				// We only do this for printable characters. We should
				// inspect the real unicode codepoint properties here but
				// the real world issue is usually control characters.
				if !utf8.Valid(event.utf8) {
					// Invalid UTF-8 so let's fallback to encoding the
					// key press as if it didn't produce UTF-8 text. I'm
					// not sure what should happen here according to the spec,
					// since it doesn't specify this behavior. Presumably
					// this is a caller bug.
					goto plain_text // break :plain_text
				}
				it := inputKeyEncodeUtf8Iterator{bytes: event.utf8}
				for cp := it.nextCodepoint(); cp.Valid; cp = it.nextCodepoint() {
					if inputKeyEncodeIsControl(cp.Value) {
						goto plain_text // break :plain_text
					}
				}

				return writer.writeAll(event.utf8)
			}
		plain_text:
		}
	}
preprocessing:

	if !entry_.Valid {
		// No entry found. If we have UTF-8 text this is a pure text event
		// (e.g. composed/IME text), so send it as-is so programs can
		// still receive it.
		if len(event.utf8) > 0 {
			return writer.writeAll(event.utf8)
		}
		return nil
	}
	entry := entry_.Value

	// If this is just a modifier we require "report all" to send the sequence.
	if entry.modifier && !opts.kitty_flags.report_all {
		return nil
	}

	var seq inputKeyEncodeKittySequence
	{ // seq:
		seq = inputKeyEncodeKittySequence{
			key:   entry.code,
			final: entry.final,
			mods: inputKeyEncodeKittyModsFromInput(
				event.action,
				event.key,
				all_mods,
			),
		}

		if opts.kitty_flags.report_events {
			switch event.action {
			case InputKeyActionPress:
				seq.event = inputKeyEncodeKittySequenceEventPress
			case InputKeyActionRelease:
				seq.event = inputKeyEncodeKittySequenceEventRelease
			case InputKeyActionRepeat:
				seq.event = inputKeyEncodeKittySequenceEventRepeat
			default:
				panic("unreachable")
			}
		}

		if opts.kitty_flags.report_alternates { // alternates:
			// Break early if this is a control key
			if inputKeyEncodeIsControl(seq.key) {
				goto alternates // break :alternates
			}

			if !utf8.Valid(event.utf8) {
				// Assume invalid UTF-8 means no UTF-8.
				goto alternates // break :alternates
			}
			it := inputKeyEncodeUtf8Iterator{bytes: event.utf8}

			// If we have a codepoint in our UTF-8 sequence, then we can
			// report the shifted version.
			if cp1_ := it.nextCodepoint(); cp1_.Valid {
				cp1 := cp1_.Value
				// Set the first alternate (shifted version)
				if cp1 != seq.key && seq.mods.shift {
					seq.alternates[0] = Some(cp1)
				}

				// We want to know if there are additional codepoints because
				// our logic below depends on the utf8 being a single codepoint.
				has_cp2 := it.nextCodepoint().Valid

				// Set the base layout key. We only report this if this codepoint
				// differs from our pressed key.
				if base_ := event.key.Codepoint(); base_.Valid {
					base := base_.Value
					if base != seq.key &&
						(cp1 != base && !has_cp2) {
						seq.alternates[1] = Some(base)
					}
				}
			} else {
				// No UTF-8 so we can't report a shifted key but we can still
				// report a base layout key.
				if base_ := event.key.Codepoint(); base_.Valid {
					base := base_.Value
					if base != seq.key {
						seq.alternates[1] = Some(base)
					}
				}
			}
		}
	alternates:

		if opts.kitty_flags.report_associated &&
			seq.event != inputKeyEncodeKittySequenceEventRelease { // associated:
			// Determine if the Alt modifier should be treated as an actual
			// modifier (in which case it prevents associated text) or as
			// the macOS Option key, which does not prevent associated text.
			var alt_prevents_text bool
			if inputKeyEncodeIsMacos() { // comptime builtin.os.tag == .macos
				switch opts.macos_option_as_alt {
				case InputConfigOptionAsAltLeft:
					alt_prevents_text = all_mods.sides.alt == InputKeyModsModSideLeft
				case InputConfigOptionAsAltRight:
					alt_prevents_text = all_mods.sides.alt == InputKeyModsModSideRight
				case InputConfigOptionAsAltTrue:
					alt_prevents_text = true
				case InputConfigOptionAsAltFalse:
					alt_prevents_text = false
				default:
					panic("unreachable")
				}
			} else {
				alt_prevents_text = true
			}

			if seq.mods.PreventsText(alt_prevents_text) {
				goto associated // break :associated
			}

			seq.text = event.utf8
		}
	associated:

		// break :seq seq
	}

	return seq.Encode(writer)
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
//
// Zig's local `const utf8 = event.utf8` is `utf8_`: the name `utf8` is the
// unicode/utf8 package, which the modify_other block needs.
func inputKeyEncodeLegacy(writer *oscWriter, event InputKeyKeyEvent, opts InputKeyEncodeOptions) error {
	all_mods := event.mods
	effective_mods := event.EffectiveMods()
	binding_mods := effective_mods.Binding()

	// Legacy encoding only does press/repeat
	if event.action != InputKeyActionPress && event.action != InputKeyActionRepeat {
		return nil
	}

	// If we're in a dead key state then we never emit a sequence.
	if event.composing {
		return nil
	}

	// If we match a PC style function key then that is our result.
	if sequence_ := inputKeyEncodePcStyleFunctionKey(
		event.key,
		all_mods,
		opts.cursor_key_application,
		opts.keypad_key_application,
		opts.ignore_keypad_with_numlock,
		opts.modify_other_keys_state_2,
	); sequence_.Valid { // pc_style:
		sequence := sequence_.Value
		// If we have UTF-8 text, then we never emit PC style function
		// keys. Many function keys (escape, enter, backspace) have
		// a specific meaning when dead keys are active and so we don't
		// want to send that to the terminal. Examples:
		//
		//   - Japanese: escape clears the dead key state
		//   - Korean: escape commits the dead key state
		//   - Korean: backspace should delete a single preedit char
		//
		if len(event.utf8) > 0 { // utf8:
			switch event.key {
			default:
			case InputKeyKeyBackspace, InputKeyKeyEnter, InputKeyKeyEscape: // inline ... => |tag|
				tag := event.key
				// We want to ignore control characters. This is because
				// some apprts (macOS) will send control characters as
				// UTF-8 encodings and we handle that manually.
				if inputKeyEncodeIsControlUtf8(event.utf8) {
					break // break :utf8
				}

				// Backspace encodes nothing because we modified IME.
				// Enter/escape don't encode the PC-style encoding
				// because we want to encode committed text.
				if tag == InputKeyKeyBackspace {
					return nil
				}
				goto pc_style // break :pc_style
			}
		}

		return writer.writeAll(sequence)
	}
pc_style:

	// If we match a control sequence, we output that directly. For
	// ctrlSeq we have to use all mods because we want it to only
	// match ctrl+<char>.
	if char_ := inputKeyEncodeCtrlSeq(
		event.key,
		event.utf8,
		event.unshifted_codepoint,
		all_mods,
	); char_.Valid {
		char := char_.Value
		// C0 sequences support alt-as-esc prefixing.
		if binding_mods.alt {
			if err := writer.writeByte(0x1B); err != nil {
				return err
			}
			if err := writer.writeByte(char); err != nil {
				return err
			}
			return nil
		}

		if err := writer.writeByte(char); err != nil {
			return err
		}
		return nil
	}

	// If we have no UTF8 text then the only possibility is the
	// alt-prefix handling of unshifted codepoints... so we process that.
	utf8_ := event.utf8
	if len(utf8_) == 0 {
		byte_, err := inputKeyEncodeLegacyAltPrefix(
			event,
			binding_mods,
			all_mods,
			opts,
		)
		if err != nil {
			return err
		}
		if byte_.Valid {
			// try writer.print("\x1B{c}", .{byte});
			if err := writer.writeAll([]uint8{0x1B, byte_.Value}); err != nil {
				return err
			}
		}
		return nil
	}

	// In modify other keys state 2, we send the CSI 27 sequence
	// for any char with a modifier. Ctrl sequences like Ctrl+a
	// are already handled above.
	if opts.modify_other_keys_state_2 { // modify_other:
		if !utf8.Valid(utf8_) {
			// Assume invalid UTF-8 means we no UTF-8.
			goto modify_other // break :modify_other
		}
		it := inputKeyEncodeUtf8Iterator{bytes: utf8_}
		codepoint_ := it.nextCodepoint()
		if !codepoint_.Valid {
			goto modify_other // orelse break :modify_other
		}
		codepoint := codepoint_.Value

		// We only do this if we have a single codepoint. There shouldn't
		// ever be a multi-codepoint sequence that triggers this.
		if it.nextCodepoint().Valid {
			goto modify_other // break :modify_other
		}

		// The mods we encode for this are just the binding mods (shift, ctrl,
		// super, alt unless it is actually option).
		var mods InputKeyMods
		{ // mods:
			mods_binding := event.mods.Binding()
			if inputKeyEncodeIsDarwin() { // comptime builtin.target.os.tag.isDarwin(); alt:
				switch opts.macos_option_as_alt {
				case InputConfigOptionAsAltFalse:
				case InputConfigOptionAsAltTrue:
					goto alt // break :alt
				case InputConfigOptionAsAltLeft:
					if event.mods.sides.alt == InputKeyModsModSideLeft {
						goto alt // break :alt
					}
				case InputConfigOptionAsAltRight:
					if event.mods.sides.alt == InputKeyModsModSideRight {
						goto alt // break :alt
					}
				default:
					panic("unreachable")
				}
				mods_binding.alt = false
			}
		alt:
			mods = mods_binding // break :mods mods_binding
		}

		// This copies xterm's `ModifyOtherKeys` function that returns
		// whether modify other keys should be encoded for the given
		// input.
		var should_modify bool
		{ // should_modify:
			// xterm IsControlInput
			if codepoint >= 0x40 && codepoint <= 0x7F {
				should_modify = true
				goto should_modify // break :should_modify true
			}

			// If we have anything other than shift pressed, encode.
			mods_no_shift := mods
			mods_no_shift.shift = false
			if !mods_no_shift.Empty() {
				should_modify = true
				goto should_modify // break :should_modify true
			}

			// We only have shift pressed. We only allow space.
			if codepoint == ' ' {
				should_modify = true
				goto should_modify // break :should_modify true
			}

			// This logic isn't complete but I don't fully understand
			// the rest so I'm going to wait until we can have a
			// reasonable test scenario.
			should_modify = false // break :should_modify false
		}
	should_modify:

		if should_modify {
			for i, modset := range InputFunctionKeysModifiers {
				code := uint64(i) + 2 // for (function_keys.modifiers, 2..) |modset, code|
				if !mods.Equal(modset) {
					continue
				}
				// return try writer.print(
				//     "\x1B[27;{};{}~",
				//     .{ code, codepoint },
				// );
				return writer.writeAll([]uint8("\x1b[27;" + itoa(code) + ";" + itoa(uint64(codepoint)) + "~"))
			}
		}
	}
modify_other:

	// Let's see if we should apply fixterms to this codepoint.
	// At this stage of key processing, we only need to apply fixterms
	// to unicode codepoints if we have ctrl set.
	if event.mods.ctrl { // csiu:
		// Important: we want to use the original mods here, not the
		// effective mods. The fixterms spec states the shifted chars
		// should be sent uppercase but Kitty changes that behavior
		// so we'll send all the mods.
		var csi_u_mods inputKeyEncodeCsiUMods
		var char uint32
		{ // mods:
			mods := inputKeyEncodeCsiUModsFromInput(event.mods)

			// Get our codepoint. If we have more than one codepoint this
			// can't be valid CSIu.
			if !utf8.Valid(event.utf8) {
				goto csiu // catch break :csiu
			}
			it := inputKeyEncodeUtf8Iterator{bytes: event.utf8}
			char_ := it.nextCodepoint()
			if !char_.Valid {
				goto csiu // orelse break :csiu
			}
			char = char_.Value
			if it.nextCodepoint().Valid {
				goto csiu // break :csiu
			}

			// If our character is A to Z and we have shift set, then
			// we lowercase it. This is a Kitty-specific behavior that
			// we choose to follow and diverge from the fixterms spec.
			// This makes it easier for programs to detect shifted letters
			// for keybindings and is not just theoretical but used by
			// real programs.
			if char >= 'A' && char <= 'Z' && mods.shift {
				// We want to rely on apprt to send us the correct
				// unshifted codepoint...
				//
				// Zig: @intCast(std.ascii.toLower(@intCast(char))). char
				// is 'A'..'Z', so the u8 cast fits and toLower sets bit 5
				// (std.ascii.toLower: `c | 0b0010_0000` for an upper-case
				// letter).
				char = uint32(uint8(char) | 0b0010_0000)
			}

			// If our unshifted codepoint is identical to the shifted
			// then we consider shift. Otherwise, we do not because the
			// shift key was used to obtain the character. This is specified
			// by fixterms.
			if event.unshifted_codepoint != char {
				mods.shift = false
			}

			csi_u_mods = mods // break :mods .{ mods, char }
		}
		// return try writer.print(
		//     "\x1B[{};{}u",
		//     .{ char, csi_u_mods.seqInt() },
		// );
		return writer.writeAll([]uint8("\x1b[" + itoa(uint64(char)) + ";" + itoa(uint64(csi_u_mods.SeqInt())) + "u"))
	}
csiu:

	// If we have alt-pressed and alt-esc-prefix is enabled, then
	// we need to prefix the utf8 sequence with an esc.
	byte_, err := inputKeyEncodeLegacyAltPrefix(
		event,
		binding_mods,
		all_mods,
		opts,
	)
	if err != nil {
		return err
	}
	if byte_.Valid {
		// return try writer.print("\x1B{c}", .{byte});
		return writer.writeAll([]uint8{0x1B, byte_.Value})
	}

	// If we are on macOS, command+keys do not encode text. It isn't
	// typical for command+keys on macOS to ever encode text. They
	// don't in native text inputs (i.e. TextEdit) and they also don't
	// in other native terminals (Terminal.app officially but also
	// iTerm2).
	//
	// For Linux, we continue to encode text because it is typical.
	// For example on Gnome Console Super+b will encode a "b" character
	// with legacy encoding.
	if inputKeyEncodeIsMacos() && all_mods.super { // comptime builtin.os.tag == .macos
		return nil
	}

	return writer.writeAll(utf8_)
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
	// This only takes effect with alt pressed
	if !binding_mods.alt || !opts.alt_esc_prefix {
		return Optional[uint8]{}, nil
	}

	// On macOS, we only handle option like alt in certain
	// circumstances. Otherwise, macOS does a unicode translation
	// and we allow that to happen.
	if inputKeyEncodeIsMacos() { // comptime builtin.os.tag == .macos
		switch opts.macos_option_as_alt {
		case InputConfigOptionAsAltFalse:
			return Optional[uint8]{}, nil
		case InputConfigOptionAsAltLeft:
			if mods.sides.alt == InputKeyModsModSideRight {
				return Optional[uint8]{}, nil
			}
		case InputConfigOptionAsAltRight:
			if mods.sides.alt == InputKeyModsModSideLeft {
				return Optional[uint8]{}, nil
			}
		case InputConfigOptionAsAltTrue:
		default:
			panic("unreachable")
		}
	}

	// Otherwise, we require utf8 to already have the byte represented.
	utf8 := event.utf8
	if len(utf8) == 1 {
		// std.math.cast(u8, utf8[0]) always succeeds: utf8[0] is a u8.
		byte_ := utf8[0]
		return Some(byte_), nil
	}

	// If UTF8 isn't set, we will allow unshifted codepoints through.
	if event.unshifted_codepoint > 0 {
		// std.math.cast(u8, event.unshifted_codepoint)
		if event.unshifted_codepoint <= 0xFF {
			byte_ := uint8(event.unshifted_codepoint)
			return Some(byte_), nil
		}
	}

	// Else, we can't figure out the byte to alt-prefix so we
	// exit this handling.
	return Optional[uint8]{}, nil
}

// inputKeyEncodeCopyToBuf is key_encode.copyToBuf: a helper to memcpy a src
// value to a buffer and return the result.
//
// Zig: fn copyToBuf(buf: []u8, src: []const u8) ![]const u8. The result is
// buf[:len(src)]; the error is ErrOutOfMemory (dcs.go) when src does not
// fit. Not called anywhere in key_encode.zig; the project tests call it.
func inputKeyEncodeCopyToBuf(buf []uint8, src []uint8) ([]uint8, error) {
	if len(src) > len(buf) {
		return nil, ErrOutOfMemory
	}
	result := buf[0:len(src)]
	// Zig: @memcpy(result, src). A safe build stops when the two ranges
	// overlap ("@memcpy arguments alias"); Go's copy would move the bytes
	// anyway, so the check is explicit (Terminal.SetPwd precedent).
	if inputKeyEncodeSlicesOverlap(result, src) {
		panic("@memcpy arguments alias")
	}
	copy(result, src)
	return result, nil
}

// inputKeyEncodeSlicesOverlap reports whether a and b share an element:
// the aliasing check of copyToBuf's @memcpy. Two non-empty slices overlap
// exactly when one of them contains the other's first element; element
// addresses are compared for equality only (no package unsafe or reflect).
// Not a Zig declaration.
func inputKeyEncodeSlicesOverlap(a []uint8, b []uint8) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for i := range a {
		if &a[i] == &b[0] {
			return true
		}
	}
	for i := range b {
		if &b[i] == &a[0] {
			return true
		}
	}
	return false
}

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
	// We only want binding-sensitive mods because lock keys
	// and directional modifiers (left/right) don't matter for
	// pc-style function keys.
	mods_int := mods.Binding().Int()

	// Keypad application keymode isn't super straightforward.
	// On xterm, in VT220 mode, numlock alone is enough to trigger
	// application mode. But in more modern modes, numlock is
	// ignored by default via mode 1035 (default true). If mode
	// 1035 is on, we always are in numerical keypad mode. If
	// mode 1035 is off, we are in application mode if the
	// proper numlock state is pressed. The numlock state is implicitly
	// determined based on the keycode sent (i.e. 1 with numlock
	// on will be kp_end).
	var keypad_key_application bool
	{ // keypad:
		// If we're ignoring keypad then this is always false.
		// In other words, we're always in numerical keypad mode.
		if ignore_keypad_with_numlock {
			keypad_key_application = false // break :keypad false
		} else {
			// If we're not ignoring then we enable the desired state.
			keypad_key_application = keypad_key_application_req // break :keypad ...
		}
	}

	for _, entry := range InputFunctionKeysKeys.Get(keyval) {
		switch entry.cursor {
		case InputFunctionKeysCursorModeAny:
		case InputFunctionKeysCursorModeNormal:
			if cursor_key_application {
				continue
			}
		case InputFunctionKeysCursorModeApplication:
			if !cursor_key_application {
				continue
			}
		default:
			panic("unreachable")
		}

		switch entry.keypad {
		case InputFunctionKeysKeypadModeAny:
		case InputFunctionKeysKeypadModeNormal:
			if keypad_key_application {
				continue
			}
		case InputFunctionKeysKeypadModeApplication:
			if !keypad_key_application {
				continue
			}
		default:
			panic("unreachable")
		}

		switch entry.modify_other_keys {
		case InputFunctionKeysModifyKeysAny:
		case InputFunctionKeysModifyKeysSet:
			if modify_other_keys {
				continue
			}
		case InputFunctionKeysModifyKeysSetOther:
			if !modify_other_keys {
				continue
			}
		default:
			panic("unreachable")
		}

		entry_mods_int := entry.mods.Int()
		if entry_mods_int == 0 {
			if mods_int != 0 && !entry.mods_empty_is_any {
				continue
			}
			// mods are either empty, or empty means any so we allow it.
		} else if entry_mods_int != mods_int {
			// any set mods require an exact match
			continue
		}

		return Some(entry.sequence)
	}

	return Optional[[]uint8]{}
}

// inputKeyEncodeCtrlSeqCtrlOnly is ctrlSeq's
// `const ctrl_only = comptime (key.Mods{ .ctrl = true }).int();`, computed
// once at start-up.
var inputKeyEncodeCtrlSeqCtrlOnly InputKeyModsModsBacking = 0b10 // TEMP-ENC-B: becomes InputKeyMods{ctrl: true}.Int() once Mods.Int exists

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
	ctrl_only := inputKeyEncodeCtrlSeqCtrlOnly

	// If ctrl is not pressed then we never do anything.
	if !mods.ctrl {
		return Optional[uint8]{}
	}

	var char uint8
	var unset_mods InputKeyMods
	{ // unset_mods:
		// We need to only get binding modifiers so we strip lock
		// keys, sides, etc.
		unset_mods = mods.Binding()

		// Remove alt from our modifiers because it does not impact whether
		// we are generating a ctrl sequence and we handle the ESC-prefix
		// logic separately.
		unset_mods.alt = false

		{ // char:
			// If we have exactly one UTF8 byte, we assume that is the
			// character we want to convert to a C0 byte.
			if len(utf8) == 1 {
				char = utf8[0]
				goto char // break :char utf8[0]
			}

			// If we have a logical key that maps to a single byte
			// printable character, we use that. History to explain this:
			// this was added to support cyrillic keyboard layouts such
			// as Russian and Mongolian. These layouts have a `c` key that
			// maps to U+0441 (cyrillic small letter "c") but every
			// terminal I've tested encodes this as ctrl+c.
			if cp_ := logical_key.Codepoint(); cp_.Valid {
				cp := cp_.Value
				if cp <= 0xFF { // std.math.cast(u8, cp)
					byte_ := uint8(cp)
					// For this specific case, we only map to the key if
					// we have exactly ctrl pressed. This is because shift
					// would modify the key and we don't know how to do that
					// properly here (don't have the layout). And we want
					// to encode shift as CSIu.
					if unset_mods.Int() != ctrl_only {
						return Optional[uint8]{}
					}
					char = byte_
					goto char // break :char byte
				}
			}

			// Otherwise we don't have a character to convert that
			// we can reliably map to a C0 byte.
			return Optional[uint8]{}
		}
	char:

		// Remove shift if we have something outside of the US letter
		// range. This is so that characters such as `ctrl+shift+-`
		// generate the correct ctrl-seq (used by emacs).
		if unset_mods.shift && (char < 'A' || char > 'Z') { // shift:
			// Special case for fixterms awkward case as specified.
			if char == '@' {
				goto shift // break :shift
			}
			unset_mods.shift = false
		}
	shift:

		// If the character is uppercase, we convert it to lowercase. We
		// rely on the unshifted codepoint to do this. This handles
		// the scenario where we have caps lock pressed. Note that
		// shifted characters are handled above, if we are just pressing
		// shift then the ctrl-only check will fail later and we won't
		// ctrl-seq encode.
		if char >= 'A' && char <= 'Z' && unshifted_codepoint > 0 {
			if unshifted_codepoint <= 0xFF { // std.math.cast(u8, unshifted_codepoint)
				byte_ := uint8(unshifted_codepoint)
				char = byte_
			}
		}

		// An additional note on caps lock and shift interaction.
		// If we have caps lock set and an ASCII letter is pressed,
		// we lowercase it (above). If we have only control pressed,
		// we process it as a ctrl seq. For example ctrl+M with caps
		// lock but no shift will encode as 0x0D.
		//
		// But, if you press ctrl+shift+m, this will not encode as a
		// ctrl-seq and falls through to CSIu encoding. This lets programs
		// detect the difference between ctrl+M and ctrl+shift+M. This
		// diverges from the fixterms "spec" and most terminals. This
		// only matches Kitty in behavior. But I believe this is a
		// justified divergence because it's a useful distinction.

		// break :unset_mods .{ char, unset_mods };
	}

	// After unsetting, we only continue if we have ONLY control set.
	if unset_mods.Int() != ctrl_only {
		return Optional[uint8]{}
	}

	// From Kitty's key encoding logic. I tried to discern the exact
	// behavior across different terminals but it's not clear, so I'm
	// just going to repeat what Kitty does.
	switch char {
	case ' ':
		return Some[uint8](0)
	case '/':
		return Some[uint8](31)
	case '0':
		return Some[uint8](48)
	case '1':
		return Some[uint8](49)
	case '2':
		return Some[uint8](0)
	case '3':
		return Some[uint8](27)
	case '4':
		return Some[uint8](28)
	case '5':
		return Some[uint8](29)
	case '6':
		return Some[uint8](30)
	case '7':
		return Some[uint8](31)
	case '8':
		return Some[uint8](127)
	case '9':
		return Some[uint8](57)
	case '?':
		return Some[uint8](127)
	case '@':
		return Some[uint8](0)
	case '\\':
		return Some[uint8](28)
	case ']':
		return Some[uint8](29)
	case '^':
		return Some[uint8](30)
	case '_':
		return Some[uint8](31)
	case 'a':
		return Some[uint8](1)
	case 'b':
		return Some[uint8](2)
	case 'c':
		return Some[uint8](3)
	case 'd':
		return Some[uint8](4)
	case 'e':
		return Some[uint8](5)
	case 'f':
		return Some[uint8](6)
	case 'g':
		return Some[uint8](7)
	case 'h':
		return Some[uint8](8)
	case 'j':
		return Some[uint8](10)
	case 'k':
		return Some[uint8](11)
	case 'l':
		return Some[uint8](12)
	case 'n':
		return Some[uint8](14)
	case 'o':
		return Some[uint8](15)
	case 'p':
		return Some[uint8](16)
	case 'q':
		return Some[uint8](17)
	case 'r':
		return Some[uint8](18)
	case 's':
		return Some[uint8](19)
	case 't':
		return Some[uint8](20)
	case 'u':
		return Some[uint8](21)
	case 'v':
		return Some[uint8](22)
	case 'w':
		return Some[uint8](23)
	case 'x':
		return Some[uint8](24)
	case 'y':
		return Some[uint8](25)
	case 'z':
		return Some[uint8](26)
	case '~':
		return Some[uint8](30)

	// These are purposely NOT handled here because of the fixterms
	// specification: https://www.leonerd.org.uk/hacks/fixterms/
	// These are processed as CSI u.
	// 'i' => 0x09,
	// 'm' => 0x0D,
	// '[' => 0x1B,

	default:
		return Optional[uint8]{}
	}
}

// inputKeyEncodeIsControl is key_encode.isControl: returns true if this is
// an ASCII control character, matches libc implementation.
//
// Zig: cp is u21.
func inputKeyEncodeIsControl(cp uint32) bool {
	return cp < 0x20 || cp == 0x7F
}

// inputKeyEncodeIsControlUtf8 is key_encode.isControlUtf8: returns true if
// this string is comprised of a single control character. This returns
// false for multi-byte strings.
func inputKeyEncodeIsControlUtf8(str []uint8) bool {
	return len(str) == 1 && inputKeyEncodeIsControl(uint32(str[0]))
}

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
	return inputKeyEncodeCsiUMods{
		shift: mods.shift,
		alt:   mods.alt,
		ctrl:  mods.ctrl,
	}
}

// Int is key_encode.CsiUMods.int: returns the raw int value of this packed
// struct.
//
// Zig: returns u3, `@bitCast(self)`: shift is bit 0, alt bit 1, ctrl bit 2.
func (self inputKeyEncodeCsiUMods) Int() uint8 {
	var v uint8 = 0
	if self.shift {
		v |= 1 << 0
	}
	if self.alt {
		v |= 1 << 1
	}
	if self.ctrl {
		v |= 1 << 2
	}
	return v
}

// SeqInt is key_encode.CsiUMods.seqInt: returns the integer value sent as
// part of the CSI u sequence. This adds 1 to the bitmask value as described
// in the spec.
//
// Zig: returns u4. The u3 is at most 7, so `raw + 1` fits.
func (self inputKeyEncodeCsiUMods) SeqInt() uint8 {
	raw := self.Int()
	return raw + 1
}

// inputKeyEncodeCsiUModsFromInt stands for Zig's `@bitCast` from u3 to
// CsiUMods, the inverse of Int. Not a Zig declaration; only the tests cast.
// A value above 7 cannot be a u3, so it is asserted (the port stops instead
// of dropping bits).
func inputKeyEncodeCsiUModsFromInt(v uint8) inputKeyEncodeCsiUMods {
	assert(v <= 7)
	return inputKeyEncodeCsiUMods{
		shift: v&(1<<0) != 0,
		alt:   v&(1<<1) != 0,
		ctrl:  v&(1<<2) != 0,
	}
}

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
	_ = action
	_ = k
	return inputKeyEncodeKittyMods{
		shift:     mods.shift,
		alt:       mods.alt,
		ctrl:      mods.ctrl,
		super:     mods.super,
		caps_lock: mods.caps_lock,
		num_lock:  mods.num_lock,
	}
}

// PreventsText is key_encode.KittyMods.preventsText: returns true if the
// modifiers prevent printable text.
//
// The alt_prevents_text parameter determines whether or not the Alt
// modifier prevents printable text. On Linux, this is always true. On
// macOS, this is only true if macos-option-as-alt is set.
func (self inputKeyEncodeKittyMods) PreventsText(alt_prevents_text bool) bool {
	return (self.alt && alt_prevents_text) ||
		self.ctrl ||
		self.super ||
		self.hyper ||
		self.meta
}

// Int is key_encode.KittyMods.int: returns the raw int value of this packed
// struct.
//
// Zig: `@bitCast(self)`: shift is bit 0 through num_lock in bit 7.
func (self inputKeyEncodeKittyMods) Int() uint8 {
	var v uint8 = 0
	if self.shift {
		v |= 1 << 0
	}
	if self.alt {
		v |= 1 << 1
	}
	if self.ctrl {
		v |= 1 << 2
	}
	if self.super {
		v |= 1 << 3
	}
	if self.hyper {
		v |= 1 << 4
	}
	if self.meta {
		v |= 1 << 5
	}
	if self.caps_lock {
		v |= 1 << 6
	}
	if self.num_lock {
		v |= 1 << 7
	}
	return v
}

// SeqInt is key_encode.KittyMods.seqInt: returns the integer value sent as
// part of the Kitty sequence. This adds 1 to the bitmask value as described
// in the spec.
//
// Zig: returns u9. The u8 is at most 255, so `raw + 1` fits.
func (self inputKeyEncodeKittyMods) SeqInt() uint16 {
	raw := uint16(self.Int())
	return raw + 1
}

// inputKeyEncodeKittyModsFromInt stands for Zig's `@bitCast` from u8 to
// KittyMods, the inverse of Int. Not a Zig declaration; only the tests cast.
func inputKeyEncodeKittyModsFromInt(v uint8) inputKeyEncodeKittyMods {
	return inputKeyEncodeKittyMods{
		shift:     v&(1<<0) != 0,
		alt:       v&(1<<1) != 0,
		ctrl:      v&(1<<2) != 0,
		super:     v&(1<<3) != 0,
		hyper:     v&(1<<4) != 0,
		meta:      v&(1<<5) != 0,
		caps_lock: v&(1<<6) != 0,
		num_lock:  v&(1<<7) != 0,
	}
}

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

// inputKeyEncodeKittySequenceEventNames are the tag names of
// KittySequence.Event, indexed by value.
var inputKeyEncodeKittySequenceEventNames = [inputKeyEncodeKittySequenceEventCount]string{
	"none",
	"press",
	"repeat",
	"release",
}

// String returns the Zig tag name (@tagName). A value outside the enum
// panics, as @tagName of an invalid value of an exhaustive enum does.
func (self inputKeyEncodeKittySequenceEvent) String() string {
	return inputKeyEncodeKittySequenceEventNames[self]
}

// Encode is key_encode.KittySequence.encode.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
func (self inputKeyEncodeKittySequence) Encode(writer *oscWriter) error {
	if self.final == 'u' || self.final == '~' {
		return self.encodeFull(writer)
	}
	return self.encodeSpecial(writer)
}

// encodeFull is key_encode.KittySequence.encodeFull.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
func (self inputKeyEncodeKittySequence) encodeFull(writer *oscWriter) error {
	// Key section
	// try writer.print("\x1B[{d}", .{self.key});
	if err := writer.writeAll([]uint8("\x1b[" + itoa(uint64(self.key)))); err != nil {
		return err
	}
	// Write our alternates
	if shifted := self.alternates[0]; shifted.Valid {
		// try writer.print(":{d}", .{shifted});
		if err := writer.writeAll([]uint8(":" + itoa(uint64(shifted.Value)))); err != nil {
			return err
		}
	}
	if base := self.alternates[1]; base.Valid {
		if !self.alternates[0].Valid {
			// try writer.print("::{d}", .{base});
			if err := writer.writeAll([]uint8("::" + itoa(uint64(base.Value)))); err != nil {
				return err
			}
		} else {
			// try writer.print(":{d}", .{base});
			if err := writer.writeAll([]uint8(":" + itoa(uint64(base.Value)))); err != nil {
				return err
			}
		}
	}

	// Mods and events section
	mods := self.mods.SeqInt()
	emit_prior := false
	if self.event != inputKeyEncodeKittySequenceEventNone && self.event != inputKeyEncodeKittySequenceEventPress {
		// try writer.print(
		//     ";{d}:{d}",
		//     .{ mods, @intFromEnum(self.event) },
		// );
		if err := writer.writeAll([]uint8(";" + itoa(uint64(mods)) + ":" + itoa(uint64(self.event)))); err != nil {
			return err
		}
		emit_prior = true
	} else if mods > 1 {
		// try writer.print(";{d}", .{mods});
		if err := writer.writeAll([]uint8(";" + itoa(uint64(mods)))); err != nil {
			return err
		}
		emit_prior = true
	}

	// Text section
	if len(self.text) > 0 { // text:
		if !utf8.Valid(self.text) {
			// Assume invalid UTF-8 means we have no text.
			goto text // break :text
		}
		it := inputKeyEncodeUtf8Iterator{bytes: self.text}
		var count uint = 0
		for cp_ := it.nextCodepoint(); cp_.Valid; cp_ = it.nextCodepoint() {
			cp := cp_.Value
			// If the codepoint is non-printable ASCII character, skip.
			if inputKeyEncodeIsControl(cp) {
				continue
			}

			// We need to add our ";". We need to add two if we didn't emit
			// the modifier section. We only do this initially.
			if count == 0 {
				if !emit_prior {
					if err := writer.writeByte(';'); err != nil {
						return err
					}
				}
				if err := writer.writeByte(';'); err != nil {
					return err
				}
			} else {
				if err := writer.writeByte(':'); err != nil {
					return err
				}
			}

			// try writer.print("{d}", .{cp});
			if err := writer.writeAll([]uint8(itoa(uint64(cp)))); err != nil {
				return err
			}
			count += 1
		}
	}
text:

	// try writer.print("{c}", .{self.final});
	return writer.writeAll([]uint8{self.final})
}

// encodeSpecial is key_encode.KittySequence.encodeSpecial.
//
// Zig: std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go).
func (self inputKeyEncodeKittySequence) encodeSpecial(writer *oscWriter) error {
	mods := self.mods.SeqInt()
	if self.event != inputKeyEncodeKittySequenceEventNone {
		// return try writer.print("\x1B[1;{d}:{d}{c}", .{
		//     mods,
		//     @intFromEnum(self.event),
		//     self.final,
		// });
		return writer.writeAll(append([]uint8("\x1b[1;"+itoa(uint64(mods))+":"+itoa(uint64(self.event))), self.final))
	}

	if mods > 1 {
		// return try writer.print("\x1B[1;{d}{c}", .{
		//     mods,
		//     self.final,
		// });
		return writer.writeAll(append([]uint8("\x1b[1;"+itoa(uint64(mods))), self.final))
	}

	// return try writer.print("\x1B[{c}", .{self.final});
	return writer.writeAll([]uint8{0x1B, '[', self.final})
}

// inputKeyEncodeIsMacos stands for the comptime check `builtin.os.tag ==
// .macos` in kitty, legacy and legacyAltPrefix. It returns platformIsMacos()
// (platform.go): the macOS branch on macOS, the other branch elsewhere.
func inputKeyEncodeIsMacos() bool { return platformIsMacos() }

// inputKeyEncodeIsDarwin stands for the comptime check
// `builtin.target.os.tag.isDarwin()` in legacy. It returns
// platformIsDarwin() (platform.go).
func inputKeyEncodeIsDarwin() bool { return platformIsDarwin() }

// inputKeyEncodeUtf8Iterator stands for std.unicode.Utf8Iterator over a
// Utf8View, i.e. over bytes that utf8.Valid accepted (Utf8View.init's
// validation). Not a Zig declaration of this file.
type inputKeyEncodeUtf8Iterator struct {
	// Zig: bytes: []const u8.
	bytes []uint8
	// Zig: i: usize.
	i int
}

// nextCodepoint is std.unicode.Utf8Iterator.nextCodepoint: the next
// codepoint, or null at the end. Zig decodes the slice found by
// nextCodepointSlice with utf8Decode (`catch unreachable` on a validated
// view); utf8.DecodeRune gives the same codepoint on valid input.
func (it *inputKeyEncodeUtf8Iterator) nextCodepoint() Optional[uint32] {
	if it.i >= len(it.bytes) {
		return Optional[uint32]{}
	}
	r, size := utf8.DecodeRune(it.bytes[it.i:])
	// Zig: utf8Decode(...) catch unreachable. The bytes were validated, so
	// DecodeRune cannot report an error (RuneError with size 1).
	if r == utf8.RuneError && size == 1 {
		panic("unreachable")
	}
	it.i += size
	return Some(uint32(r))
}
