// WIREFRAME for src/terminal/osc/parsers/semantic_prompt.zig (Ghostty
// v1.3.1). NOT YET PORTED.
//
// https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md
//
// Types and signatures only; see the header of color.go for the rules.
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
// src/os/string_encoding.zig. That file is not one of this wireframe's
// sources; nothing of it is declared here.

package terminal

import "errors"

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

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptCommandAction) String() string { panic(notImplemented) }

// OSCParsersSemanticPromptCommandInit is
// osc.parsers.semantic_prompt.Command.init: the command for action with no
// options.
func OSCParsersSemanticPromptCommandInit(action OSCParsersSemanticPromptCommandAction) OSCParsersSemanticPromptCommand {
	panic(notImplemented)
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
	panic(notImplemented)
}

// WriteCommandLine is osc.parsers.semantic_prompt.Command.writeCommandLine:
// write the decoded command line (if any) to the writer. If an error occurs
// garbage may have been written to the writer.
//
// Zig: the error set is std.Io.Writer.Error || error{DecodeError}, i.e.
// ErrWriteFailed (osc.go) or ErrDecodeError. The writer is a
// *std.Io.Writer; see oscWriter in osc.go.
func (self OSCParsersSemanticPromptCommand) WriteCommandLine(writer *oscWriter) error {
	panic(notImplemented)
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

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptOption) String() string { panic(notImplemented) }

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
func (self OSCParsersSemanticPromptOption) key() []uint8 { panic(notImplemented) }

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
	panic(notImplemented)
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

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptClick) String() string { panic(notImplemented) }

// OSCParsersSemanticPromptClickInit is
// osc.parsers.semantic_prompt.Click.init: the Click for the value of a `cl`
// option, or null.
func OSCParsersSemanticPromptClickInit(value []uint8) Optional[OSCParsersSemanticPromptClick] {
	panic(notImplemented)
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

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptPromptKind) String() string { panic(notImplemented) }

// OSCParsersSemanticPromptPromptKindInit is
// osc.parsers.semantic_prompt.PromptKind.init: the PromptKind for the byte
// of a `k` option, or null.
func OSCParsersSemanticPromptPromptKindInit(c uint8) Optional[OSCParsersSemanticPromptPromptKind] {
	panic(notImplemented)
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

// String returns the Zig tag name (@tagName).
func (self OSCParsersSemanticPromptRedraw) String() string { panic(notImplemented) }

// OSCParsersSemanticPromptParse is osc.parsers.semantic_prompt.parse: parse
// OSC 133, semantic prompts.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*OSCCommand`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersSemanticPromptParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}
