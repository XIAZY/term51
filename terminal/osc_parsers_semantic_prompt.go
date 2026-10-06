// Port of src/terminal/osc/parsers/semantic_prompt.zig (Ghostty v1.3.1).
//
// https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md
//
// File naming: `osc/parsers/semantic_prompt.zig` becomes
// `osc_parsers_semantic_prompt.go` in package terminal, and its declarations
// take the prefix `OSCParsersSemanticPrompt`. The file has no C ABI
// declarations.
//
// Not ported:
//   - `Option.Type`, a comptime function that returns a type. Go has no such
//     thing; its counterpart is the struct
//     OSCParsersSemanticPromptOptionValue, which holds one result field per
//     Option value.
//   - `const log = std.log.scoped(.osc_semantic_prompt)`: the port uses
//     warn("osc_semantic_prompt", ...).
//
// `Command.writeCommandLine` calls printfQDecode and urlPercentDecode of
// src/os/string_encoding.zig; they are ported in os_string_encoding.go.
//
// Zig's labeled blocks (`valid: { ... }`, `value: { ... }`) whose `break
// :label` leaves the block are a Go block followed by the label and `goto`.
// A `break :label` that leaves a switch prong whose block is the whole prong
// is Go's `break` out of the switch.
//
// std.fmt.parseInt has no Go standard-library equivalent that accepts the
// same inputs (Zig accepts '_' between digits); it is ported as
// oscParsersSemanticPromptParseInt below.

package terminal

import (
	"bytes"
	"errors"
	"math"
)

// ErrDecodeError is Zig's error.DecodeError
// (semantic_prompt.Command.writeCommandLine, raised by the decoders of
// os/string_encoding.zig).
var ErrDecodeError = errors.New("DecodeError")

// OSCParsersSemanticPromptCommand is osc.parsers.semantic_prompt.Command: a
// single semantic prompt command.
//
// Technically according to the spec, not all commands have options but it is
// easier to be "liberal in what we accept" here since all except one do and
// the spec does also say to ignore unknown options. So, I think this is a
// fair interpretation.
type OSCParsersSemanticPromptCommand struct {
	action              OSCParsersSemanticPromptCommandAction
	options_unvalidated []uint8
}

// OSCParsersSemanticPromptCommandAction is
// osc.parsers.semantic_prompt.Command.Action.
type OSCParsersSemanticPromptCommandAction uint8

const (
	OSCParsersSemanticPromptCommandActionFreshLine                       OSCParsersSemanticPromptCommandAction = iota // 'L'
	OSCParsersSemanticPromptCommandActionFreshLineNewPrompt                                                           // 'A'
	OSCParsersSemanticPromptCommandActionNewCommand                                                                   // 'N'
	OSCParsersSemanticPromptCommandActionPromptStart                                                                  // 'P'
	OSCParsersSemanticPromptCommandActionEndPromptStartInput                                                          // 'B'
	OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol                                              // 'I'
	OSCParsersSemanticPromptCommandActionEndInputStartOutput                                                          // 'C'
	OSCParsersSemanticPromptCommandActionEndCommand                                                                   // 'D'

	oscParsersSemanticPromptCommandActionCount = iota
)

// oscParsersSemanticPromptCommandActionNames are the Zig tag names of
// Command.Action.
var oscParsersSemanticPromptCommandActionNames = [oscParsersSemanticPromptCommandActionCount]string{
	"fresh_line",
	"fresh_line_new_prompt",
	"new_command",
	"prompt_start",
	"end_prompt_start_input",
	"end_prompt_start_input_terminate_eol",
	"end_input_start_output",
	"end_command",
}

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptCommandAction) String() string {
	return oscParsersSemanticPromptCommandActionNames[self]
}

// OSCParsersSemanticPromptCommandInit is
// osc.parsers.semantic_prompt.Command.init: the command for action with no
// options.
func OSCParsersSemanticPromptCommandInit(action OSCParsersSemanticPromptCommandAction) OSCParsersSemanticPromptCommand {
	return OSCParsersSemanticPromptCommand{
		action:              action,
		options_unvalidated: []uint8(""),
	}
}

// ReadOption is osc.parsers.semantic_prompt.Command.readOption: read an
// option for this command. Returns null if unset or invalid.
//
// Zig: `option` is a comptime parameter and the result type is
// `?option.Type()`. Go: the result is in the field of the returned struct
// that is named after `option`; see OSCParsersSemanticPromptOptionValue.
func (self OSCParsersSemanticPromptCommand) ReadOption(
	option OSCParsersSemanticPromptOption,
) OSCParsersSemanticPromptOptionValue {
	return option.Read(self.options_unvalidated)
}

// WriteCommandLine is osc.parsers.semantic_prompt.Command.writeCommandLine:
// write the decoded command line (if any) to the writer. If an error occurs
// garbage may have been written to the writer.
//
// Zig: the error set is std.Io.Writer.Error || error{DecodeError}, i.e.
// ErrWriteFailed (osc.go) or ErrDecodeError. The writer is a
// *std.Io.Writer; see oscWriter in osc.go.
func (self OSCParsersSemanticPromptCommand) writeCommandLine(writer *oscWriter) error {
	if command_line := self.ReadOption(OSCParsersSemanticPromptOptionCmdline).Cmdline; command_line.Valid {
		if err := OsStringEncodingPrintfQDecode(writer, command_line.Value); err != nil {
			return err
		}
		return nil
	}
	if command_line := self.ReadOption(OSCParsersSemanticPromptOptionCmdlineUrl).CmdlineUrl; command_line.Valid {
		if err := OsStringEncodingUrlPercentDecode(writer, command_line.Value); err != nil {
			return err
		}
		return nil
	}
	return nil
}

// OSCParsersSemanticPromptOption is osc.parsers.semantic_prompt.Option.
type OSCParsersSemanticPromptOption uint8

const (
	OSCParsersSemanticPromptOptionAid OSCParsersSemanticPromptOption = iota
	OSCParsersSemanticPromptOptionCl
	OSCParsersSemanticPromptOptionPromptKind
	OSCParsersSemanticPromptOptionErr
	OSCParsersSemanticPromptOptionCmdline
	OSCParsersSemanticPromptOptionCmdlineUrl

	// https://sw.kovidgoyal.net/kitty/shell-integration/#notes-for-shell-developers
	// Kitty supports a "redraw" option for prompt_start. This is extended
	// by Ghostty with the "last" option. See Redraw the type for more
	// details.
	OSCParsersSemanticPromptOptionRedraw

	// Use a special key instead of arrow keys to move the cursor on
	// mouse click. Useful if arrow keys have side-effets like triggering
	// auto-complete. The shell integration script should bind the special
	// key as needed.
	// See: https://sw.kovidgoyal.net/kitty/shell-integration/#notes-for-shell-developers
	OSCParsersSemanticPromptOptionSpecialKey

	// If true, the shell is capable of handling mouse click events.
	// Ghostty will then send a click event to the shell when the user
	// clicks somewhere in the prompt. The shell can then move the cursor
	// to that position or perform some other appropriate action. If false,
	// Ghostty may generate a number of fake key events to move the cursor
	// which is not very robust.
	// See: https://sw.kovidgoyal.net/kitty/shell-integration/#notes-for-shell-developers
	OSCParsersSemanticPromptOptionClickEvents

	// Not technically an option that can be set with k=v and only
	// present currently with command 'D' but its easier to just
	// parse it into our options.
	OSCParsersSemanticPromptOptionExitCode

	oscParsersSemanticPromptOptionCount = iota
)

// oscParsersSemanticPromptOptionNames are the Zig tag names of Option.
var oscParsersSemanticPromptOptionNames = [oscParsersSemanticPromptOptionCount]string{
	"aid",
	"cl",
	"prompt_kind",
	"err",
	"cmdline",
	"cmdline_url",
	"redraw",
	"special_key",
	"click_events",
	"exit_code",
}

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptOption) String() string {
	return oscParsersSemanticPromptOptionNames[self]
}

// OSCParsersSemanticPromptOptionValue is the Go counterpart of
// osc.parsers.semantic_prompt.Option.Type, the comptime function that gives
// the type `read` and `readOption` return (as an optional) for each option:
//
//	.aid, .err, .cmdline, .cmdline_url  []const u8
//	.cl                                 Click
//	.prompt_kind                        PromptKind
//	.redraw                             Redraw
//	.special_key, .click_events         bool
//	.exit_code                          i32
//
// Go cannot make a result type depend on an argument value, so `Read` and
// `ReadOption` return this struct. It works like a tagged union whose tag is
// the Option that was asked for: only the field named after that Option is
// meaningful, and it is the null Optional where Zig returns null.
type OSCParsersSemanticPromptOptionValue struct {
	Aid         Optional[[]uint8]
	Cl          Optional[OSCParsersSemanticPromptClick]
	PromptKind  Optional[OSCParsersSemanticPromptPromptKind]
	Err         Optional[[]uint8]
	Cmdline     Optional[[]uint8]
	CmdlineUrl  Optional[[]uint8]
	Redraw      Optional[OSCParsersSemanticPromptRedraw]
	SpecialKey  Optional[bool]
	ClickEvents Optional[bool]
	ExitCode    Optional[int32]
}

// key is osc.parsers.semantic_prompt.Option.key: the key of the option in
// the raw options string. Zig: unreachable for .exit_code, which has no key.
func (self OSCParsersSemanticPromptOption) key() []uint8 {
	switch self {
	case OSCParsersSemanticPromptOptionAid:
		return []uint8("aid")
	case OSCParsersSemanticPromptOptionCl:
		return []uint8("cl")
	case OSCParsersSemanticPromptOptionPromptKind:
		return []uint8("k")
	case OSCParsersSemanticPromptOptionErr:
		return []uint8("err")
	case OSCParsersSemanticPromptOptionRedraw:
		return []uint8("redraw")
	case OSCParsersSemanticPromptOptionSpecialKey:
		return []uint8("special_key")
	case OSCParsersSemanticPromptOptionClickEvents:
		return []uint8("click_events")
	case OSCParsersSemanticPromptOptionCmdline:
		return []uint8("cmdline")
	case OSCParsersSemanticPromptOptionCmdlineUrl:
		return []uint8("cmdline_url")

	// special case, handled before ever calling key
	case OSCParsersSemanticPromptOptionExitCode:
		panic("unreachable")
	}
	panic("unreachable")
}

// Read is osc.parsers.semantic_prompt.Option.read: read the option value
// from the raw options string.
//
// The raw options string is the raw unparsed data after the OSC 133 command.
// e.g. for `133;A;aid=14;cl=line`, the raw options string would be
// `aid=14;cl=line`.
//
// Any errors in the raw string will return null since the OSC133
// specification says to ignore unknown or malformed options.
//
// Zig: `self` is a comptime parameter and the result type is
// `?self.Type()`. Go: the result is in the field of the returned struct that
// is named after `self`; see OSCParsersSemanticPromptOptionValue.
func (self OSCParsersSemanticPromptOption) Read(raw []uint8) OSCParsersSemanticPromptOptionValue {
	var result OSCParsersSemanticPromptOptionValue
	remaining := raw
	for len(remaining) > 0 {
		// Length of the next value is up to the `;` or the
		// end of the string.
		len_ := bytes.IndexByte(remaining, ';')
		if len_ == -1 {
			len_ = len(remaining)
		}

		// Grab our full value and move our cursor past the `;`
		full := remaining[0:len_]

		// If we're looking for exit_code we special case it.
		// as the first value.
		if self == OSCParsersSemanticPromptOptionExitCode {
			if v, ok := oscParsersSemanticPromptParseInt(full); ok {
				result.ExitCode = Some(v)
			}
			return result
		}

		// Parse our key=value and verify our key matches our
		// expectation.
		var value []uint8
		{
			if eql_idx := bytes.IndexByte(full, '='); eql_idx != -1 {
				if bytes.Equal(
					full[0:eql_idx],
					self.key(),
				) {
					value = full[eql_idx+1:]
					goto value
				}
			}

			// No match!
			if len_ < len(remaining) {
				remaining = remaining[len_+1:]
				continue
			}

			break
		}
	value:

		switch self {
		case OSCParsersSemanticPromptOptionAid:
			result.Aid = Some(value)
		case OSCParsersSemanticPromptOptionCl:
			result.Cl = OSCParsersSemanticPromptClickInit(value)
		case OSCParsersSemanticPromptOptionPromptKind:
			if len(value) == 1 {
				result.PromptKind = OSCParsersSemanticPromptPromptKindInit(value[0])
			}
		case OSCParsersSemanticPromptOptionErr:
			result.Err = Some(value)
		case OSCParsersSemanticPromptOptionRedraw:
			if bytes.Equal(value, []uint8("0")) {
				result.Redraw = Some(OSCParsersSemanticPromptRedrawFalse)
			} else if bytes.Equal(value, []uint8("1")) {
				result.Redraw = Some(OSCParsersSemanticPromptRedrawTrue)
			} else if bytes.Equal(value, []uint8("last")) {
				result.Redraw = Some(OSCParsersSemanticPromptRedrawLast)
			}
		case OSCParsersSemanticPromptOptionSpecialKey, OSCParsersSemanticPromptOptionClickEvents:
			var b Optional[bool]
			if len(value) == 1 {
				switch value[0] {
				case '0':
					b = Some(false)
				case '1':
					b = Some(true)
				}
			}
			if self == OSCParsersSemanticPromptOptionSpecialKey {
				result.SpecialKey = b
			} else {
				result.ClickEvents = b
			}
		case OSCParsersSemanticPromptOptionCmdline:
			result.Cmdline = Some(value)
		case OSCParsersSemanticPromptOptionCmdlineUrl:
			result.CmdlineUrl = Some(value)
		// Handled above
		case OSCParsersSemanticPromptOptionExitCode:
			panic("unreachable")
		default:
			panic("unreachable")
		}
		return result
	}

	// Not found
	return result
}

// OSCParsersSemanticPromptClick is osc.parsers.semantic_prompt.Click: the
// `cl` option specifies what kind of cursor key sequences are handled by the
// application for click-to-move-cursor functionality.
type OSCParsersSemanticPromptClick uint8

const (
	// Value: "line". Allows motion within a single input line using
	// standard left/right arrow escape sequences. Only a single left/right
	// sequence should be emitted for double-width characters.
	OSCParsersSemanticPromptClickLine OSCParsersSemanticPromptClick = iota

	// Value: "m". Allows movement between different lines in the same
	// group, but only using left/right arrow escape sequences.
	OSCParsersSemanticPromptClickMultiple

	// Value: "v". Like `multiple` but cursor up/down should be used. The
	// terminal should be conservative when moving between lines: move the
	// cursor left to the start of line, emit the needed up/down sequences,
	// then move the cursor right to the clicked destination.
	OSCParsersSemanticPromptClickConservativeVertical

	// Value: "w". Like `conservative_vertical` but specifies that there
	// are no spurious spaces at the end of the line, and the application
	// editor handles "smart vertical movement" (moving 2 lines up from
	// position 20, where the intermediate line is 15 chars wide and the
	// destination is 18 chars wide, ends at position 18).
	OSCParsersSemanticPromptClickSmartVertical
)

// oscParsersSemanticPromptClickNames are the Zig tag names of Click.
var oscParsersSemanticPromptClickNames = [...]string{
	"line",
	"multiple",
	"conservative_vertical",
	"smart_vertical",
}

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptClick) String() string {
	return oscParsersSemanticPromptClickNames[self]
}

// OSCParsersSemanticPromptClickInit is
// osc.parsers.semantic_prompt.Click.init: the Click for the value of a `cl`
// option, or null.
func OSCParsersSemanticPromptClickInit(value []uint8) Optional[OSCParsersSemanticPromptClick] {
	if len(value) == 1 {
		switch value[0] {
		case 'm':
			return Some(OSCParsersSemanticPromptClickMultiple)
		case 'v':
			return Some(OSCParsersSemanticPromptClickConservativeVertical)
		case 'w':
			return Some(OSCParsersSemanticPromptClickSmartVertical)
		default:
			return None[OSCParsersSemanticPromptClick]()
		}
	} else if bytes.Equal(
		value,
		[]uint8("line"),
	) {
		return Some(OSCParsersSemanticPromptClickLine)
	} else {
		return None[OSCParsersSemanticPromptClick]()
	}
}

// OSCParsersSemanticPromptPromptKind is
// osc.parsers.semantic_prompt.PromptKind.
type OSCParsersSemanticPromptPromptKind uint8

const (
	OSCParsersSemanticPromptPromptKindInitial OSCParsersSemanticPromptPromptKind = iota
	OSCParsersSemanticPromptPromptKindRight
	OSCParsersSemanticPromptPromptKindContinuation
	OSCParsersSemanticPromptPromptKindSecondary
)

// oscParsersSemanticPromptPromptKindNames are the Zig tag names of
// PromptKind.
var oscParsersSemanticPromptPromptKindNames = [...]string{
	"initial",
	"right",
	"continuation",
	"secondary",
}

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptPromptKind) String() string {
	return oscParsersSemanticPromptPromptKindNames[self]
}

// OSCParsersSemanticPromptPromptKindInit is
// osc.parsers.semantic_prompt.PromptKind.init: the PromptKind for the byte
// of a `k` option, or null.
func OSCParsersSemanticPromptPromptKindInit(c uint8) Optional[OSCParsersSemanticPromptPromptKind] {
	switch c {
	case 'i':
		return Some(OSCParsersSemanticPromptPromptKindInitial)
	case 'r':
		return Some(OSCParsersSemanticPromptPromptKindRight)
	case 'c':
		return Some(OSCParsersSemanticPromptPromptKindContinuation)
	case 's':
		return Some(OSCParsersSemanticPromptPromptKindSecondary)
	default:
		return None[OSCParsersSemanticPromptPromptKind]()
	}
}

// OSCParsersSemanticPromptRedraw is osc.parsers.semantic_prompt.Redraw: the
// values for the `redraw` extension to OSC133. This was started by Kitty[1]
// and extended by Ghostty (the "last" option).
//
// Zig: enum(u2).
//
// [1]: https://sw.kovidgoyal.net/kitty/shell-integration/#notes-for-shell-developers
type OSCParsersSemanticPromptRedraw uint8

const (
	// The shell supports redrawing the full prompt and all continuations.
	// This is the default value, it does not need to be explicitly set
	// unless it is to reset a prior other value.
	OSCParsersSemanticPromptRedrawTrue OSCParsersSemanticPromptRedraw = 0

	// The shell does NOT support redrawing. In this case, Ghostty will NOT
	// clear any prompt lines on resize.
	OSCParsersSemanticPromptRedrawFalse OSCParsersSemanticPromptRedraw = 1

	// The shell supports redrawing only the LAST line of the prompt.
	// Ghostty will only clear the last line of the prompt on resize.
	//
	// This is specifically introduced because Bash only redraws the last
	// line. It is literally the only shell that does this and it does this
	// because its bad and they should feel bad. Don't be like Bash.
	OSCParsersSemanticPromptRedrawLast OSCParsersSemanticPromptRedraw = 2
)

// oscParsersSemanticPromptRedrawNames are the Zig tag names of Redraw.
var oscParsersSemanticPromptRedrawNames = [...]string{
	"true",
	"false",
	"last",
}

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptRedraw) String() string {
	return oscParsersSemanticPromptRedrawNames[self]
}

// OSCParsersSemanticPromptParse is osc.parsers.semantic_prompt.parse: parse
// OSC 133, semantic prompts.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*OSCCommand`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersSemanticPromptParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	writer := parser.writer
	if writer == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	data := writer.buffered()
	if len(data) == 0 {
		parser.state = OSCParserStateInvalid
		return nil
	}

	// All valid cases terminate within this block. Any fallthroughs
	// are invalid. This makes some of our parse logic a little less
	// repetitive.
	{
		switch data[0] {
		case 'A': // fresh_line:
			parser.command = OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt)}
			if len(data) == 1 {
				break
			}
			if data[1] != ';' {
				goto valid
			}
			parser.command.SemanticPrompt.options_unvalidated = data[2:]

		case 'B': // end_prompt:
			parser.command = OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndPromptStartInput)}
			if len(data) == 1 {
				break
			}
			if data[1] != ';' {
				goto valid
			}
			parser.command.SemanticPrompt.options_unvalidated = data[2:]

		case 'I': // end_prompt_line:
			parser.command = OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol)}
			if len(data) == 1 {
				break
			}
			if data[1] != ';' {
				goto valid
			}
			parser.command.SemanticPrompt.options_unvalidated = data[2:]

		case 'C': // end_input:
			parser.command = OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndInputStartOutput)}
			if len(data) == 1 {
				break
			}
			if data[1] != ';' {
				goto valid
			}
			parser.command.SemanticPrompt.options_unvalidated = data[2:]

		case 'D': // end_command:
			parser.command = OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndCommand)}
			if len(data) == 1 {
				break
			}
			if data[1] != ';' {
				goto valid
			}
			parser.command.SemanticPrompt.options_unvalidated = data[2:]

		case 'L':
			if len(data) > 1 {
				goto valid
			}
			parser.command = OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionFreshLine)}

		case 'N': // new_command:
			parser.command = OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionNewCommand)}
			if len(data) == 1 {
				break
			}
			if data[1] != ';' {
				goto valid
			}
			parser.command.SemanticPrompt.options_unvalidated = data[2:]

		case 'P': // prompt_start:
			parser.command = OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart)}
			if len(data) == 1 {
				break
			}
			if data[1] != ';' {
				goto valid
			}
			parser.command.SemanticPrompt.options_unvalidated = data[2:]

		default:
			goto valid
		}

		return &parser.command
	}
valid:

	// Any fallthroughs are invalid
	parser.state = OSCParserStateInvalid
	return nil
}

// oscParsersSemanticPromptParseInt is std.fmt.parseInt(i32, buf, 10), as
// Option.read calls it.
//
// Not a function of semantic_prompt.zig: it is the port of the
// standard-library function (std/fmt.zig parseIntWithGenericCharacter and
// parseIntWithSign, base 10, Result = Accumulate = i32). The only caller
// discards the error (`catch null`), so only success is reported: ok is
// false where Zig returns error.InvalidCharacter or error.Overflow. Like
// Zig, it accepts one leading '+' or '-', leading zeros and '_' between
// digits, and rejects an empty string, a bare sign and a leading or trailing
// '_' (after the sign).
func oscParsersSemanticPromptParseInt(buf []uint8) (int32, bool) {
	if len(buf) == 0 {
		return 0, false
	}
	neg := false
	if buf[0] == '+' {
		buf = buf[1:]
	} else if buf[0] == '-' {
		buf = buf[1:]
		neg = true
	}

	// parseIntWithSign
	if len(buf) == 0 {
		return 0, false
	}
	if buf[0] == '_' || buf[len(buf)-1] == '_' {
		return 0, false
	}
	var accumulate int32
	for _, c := range buf {
		if c == '_' {
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
		digit := int32(c - '0')
		if accumulate != 0 {
			// math.mul(i32, accumulate, 10)
			if accumulate > math.MaxInt32/10 || accumulate < math.MinInt32/10 {
				return 0, false
			}
			accumulate *= 10
		} else if neg {
			// The first digit of a negative number.
			accumulate = -digit
			continue
		}
		if !neg {
			// math.add(i32, accumulate, digit)
			if accumulate > math.MaxInt32-digit {
				return 0, false
			}
			accumulate += digit
		} else {
			// math.sub(i32, accumulate, digit)
			if accumulate < math.MinInt32+digit {
				return 0, false
			}
			accumulate -= digit
		}
	}
	return accumulate, true
}
