// WIREFRAME for src/terminal/osc.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// OSC (Operating System Command) related functions and types.
//
// OSC is another set of control sequences for terminal programs that start
// with "ESC ]". Unlike CSI or standard ESC sequences, they may contain
// strings and other irregular formatting so a dedicated parser is created to
// handle it.
//
// Types and signatures only; see the header of color.go for the rules.
// Every function body is panic(notImplemented), with one exception: the six
// functions that the already-ported Parser.go calls (OSCCommand.zigFmtAny,
// OSCParserInit, Deinit, Reset, Next, End) keep the do-nothing bodies of the
// placeholder this file replaces, because existing passing Parser tests
// depend on them. Each is marked "Placeholder body kept from the previous
// osc.go; not the implementation."
//
// File naming: osc.zig is a namespace file, so its declarations take the
// prefix `OSC`. Declarations appear in the same order as in osc.zig.
//
// Not ported, see docs/mapping.md:
//   - the C ABI declarations: `Command.ProgressReport.C` and `cval`,
//     `Terminator.C` and `cval`, the C variant of `Command.Key` (LibEnum with
//     the .c target) and the "ghostty.h" enum check;
//   - the comptime size assertion on Command (memory layout only);
//   - the namespace re-exports `pub const color = parsers.color` and
//     `pub const semantic_prompt = parsers.semantic_prompt` (Go has one
//     package; the names are OSCParsersColor... and
//     OSCParsersSemanticPrompt...);
//   - `const log = std.log.scoped(.osc)`: the port uses warn("osc", ...).

package terminal

import "errors"

// ErrWriteFailed is Zig's error.WriteFailed (std.Io.Writer.Error): a write
// did not fit in the writer's buffer. osc.Parser.next turns it into the
// invalid state.
var ErrWriteFailed = errors.New("WriteFailed")

// oscWriter stands in for std.Io.Writer as osc.zig and the files under osc/
// use it: a buffer and the number of bytes written to it. Zig's
// `writer.buffered()` is buffer[:end]. Wireframe: fields, plus the stubs
// the tests need (oscWriterFixed, buffered, and toOwnedSlice on
// oscWriterAllocating, declared below); the implementation adds the other
// methods it needs (writeByte, ...) with Zig's names, as
// dcsWriterAllocating does in dcs.go.
type oscWriter struct {
	// Zig: buffer: []u8. For a fixed writer this is the parser's own
	// `buffer` array.
	buffer []uint8

	// Zig: end: usize. The number of bytes of buffer in use.
	end int
}

// oscWriterAllocating stands in for std.Io.Writer.Allocating: a writer whose
// buffer grows. Wireframe: fields only.
type oscWriterAllocating struct {
	// Zig: writer: std.Io.Writer.
	writer oscWriter
}

// oscWriterAllocatingInit is std.Io.Writer.Allocating.init: an empty
// allocating writer. The allocator parameter is dropped.
func oscWriterAllocatingInit() oscWriterAllocating { panic(notImplemented) }

// written is std.Io.Writer.Allocating.written: the bytes written so far.
func (self *oscWriterAllocating) written() []uint8 { panic(notImplemented) }

// deinit is std.Io.Writer.Allocating.deinit.
func (self *oscWriterAllocating) deinit() { panic(notImplemented) }

// toOwnedSlice is std.Io.Writer.Allocating.toOwnedSlice: the bytes written
// so far, handed to the caller, leaving the writer empty. The allocator
// parameter is dropped. Zig: Error![]u8; the error is ErrOutOfMemory
// (dcs.go).
func (self *oscWriterAllocating) toOwnedSlice() ([]uint8, error) { panic(notImplemented) }

// oscWriterFixed is std.Io.Writer.fixed: a writer that writes into buf and
// fails with ErrWriteFailed once buf is full. Zig: `var w: std.Io.Writer =
// .fixed(&buf);` is `w := oscWriterFixed(buf[:])`.
func oscWriterFixed(buf []uint8) oscWriter { panic(notImplemented) }

// buffered is std.Io.Writer.buffered: the bytes written into the writer's
// buffer so far, buffer[:end]. For an allocating writer Zig's
// `builder.writer.buffered()` is `builder.writer.buffered()`.
func (self *oscWriter) buffered() []uint8 { panic(notImplemented) }

// OSCCommand is osc.Command.
//
// Zig: union(Key). Only the field named by Tag is meaningful. The variants
// invalid, hyperlink_end and conemu_wait_input carry no payload and have no
// field.
type OSCCommand struct {
	Tag OSCCommandKey

	// Set the window title of the terminal
	//
	// If title mode 0 is set text is expect to be hex encoded (i.e. utf-8
	// with each code unit further encoded with two hex digits).
	//
	// If title mode 2 is set or the terminal is setup for unconditional
	// utf-8 titles text is interpreted as utf-8. Else text is interpreted
	// as latin1.
	ChangeWindowTitle []uint8

	// Set the icon of the terminal window. The name of the icon is not
	// well defined, so this is currently ignored by Ghostty at the time
	// of writing this. We just parse it so that we don't get parse errors
	// in the log.
	ChangeWindowIcon []uint8

	// Semantic prompt command: https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md
	SemanticPrompt OSCCommandSemanticPrompt

	// Set or get clipboard contents. If data is null, then the current
	// clipboard contents are sent to the pty. If data is set, this
	// contents is set on the clipboard.
	ClipboardContents OSCCommandClipboardContentsPayload

	// OSC 7. Reports the current working directory of the shell. This is
	// a moderately flawed escape sequence but one that many major terminals
	// support so we also support it. To understand the flaws, read through
	// this terminal-wg issue: https://gitlab.freedesktop.org/terminal-wg/specifications/-/issues/20
	ReportPwd OSCCommandReportPwdPayload

	// OSC 22. Set the mouse shape. There doesn't seem to be a standard
	// naming scheme for cursors but it looks like terminals such as Foot
	// are moving towards using the W3C CSS cursor names. For OSC parsing,
	// we just parse whatever string is given.
	MouseShape OSCCommandMouseShapePayload

	// OSC color operations to set, reset, or report color settings. Some
	// OSCs allow multiple operations to be specified in a single OSC so we
	// need a list-like datastructure to manage them.
	//
	// Currently, these OSCs are handled by `color_operation`:
	//
	// 4, 5, 10-19, 104, 105, 110-119
	ColorOperation OSCCommandColorOperationPayload

	// Kitty color protocol, OSC 21
	// https://sw.kovidgoyal.net/kitty/color-stack/#id1
	KittyColorProtocol KittyColorOSC

	// Show a desktop notification (OSC 9 or OSC 777)
	ShowDesktopNotification OSCCommandShowDesktopNotificationPayload

	// Start a hyperlink (OSC 8)
	HyperlinkStart OSCCommandHyperlinkStartPayload

	// ConEmu sleep (OSC 9;1)
	ConemuSleep OSCCommandConemuSleepPayload

	// ConEmu show GUI message box (OSC 9;2)
	ConemuShowMessageBox []uint8

	// ConEmu change tab title (OSC 9;3)
	ConemuChangeTabTitle OSCCommandConemuChangeTabTitlePayload

	// ConEmu progress report (OSC 9;4)
	ConemuProgressReport OSCCommandProgressReport

	// ConEmu GUI macro (OSC 9;6)
	ConemuGuimacro []uint8

	// ConEmu run process (OSC 9;7)
	ConemuRunProcess []uint8

	// ConEmu output environment variable (OSC 9;8)
	ConemuOutputEnvironmentVariable []uint8

	// ConEmu XTerm keyboard and output emulation (OSC 9;10)
	// https://conemu.github.io/en/TerminalModes.html
	ConemuXtermEmulation OSCCommandConemuXtermEmulationPayload

	// ConEmu comment (OSC 9;11)
	ConemuComment []uint8

	// Kitty text sizing protocol (OSC 66)
	KittyTextSizing OSCParsersKittyTextSizingOSC

	KittyClipboardProtocol OSCCommandKittyClipboardProtocol

	// OSC 3008. Hierarchical context signalling (UAPI spec).
	// https://uapi-group.org/specifications/specs/osc_context/
	ContextSignal OSCParsersContextSignalCommand
}

// zigFmtAny is not a declaration of osc.zig: it is the port's rendering of
// Zig's std.fmt "{any}" for a Command, which Parser.go calls when it formats
// an osc_dispatch action.
//
// Placeholder body kept from the previous osc.go; not the implementation.
func (self OSCCommand) zigFmtAny() string {
	return "<osc command: not yet ported>"
}

// OSCCommandClipboardContentsPayload is the payload of
// Command.clipboard_contents: struct { kind: u8, data: [:0]const u8 }.
type OSCCommandClipboardContentsPayload struct {
	kind uint8
	data []uint8
}

// OSCCommandReportPwdPayload is the payload of Command.report_pwd:
// struct { value: [:0]const u8 }.
type OSCCommandReportPwdPayload struct {
	// The reported pwd value. This is not checked for validity. It should
	// be a file URL but it is up to the caller to utilize this value.
	value []uint8
}

// OSCCommandMouseShapePayload is the payload of Command.mouse_shape:
// struct { value: [:0]const u8 }.
type OSCCommandMouseShapePayload struct {
	value []uint8
}

// OSCCommandColorOperationPayload is the payload of Command.color_operation.
type OSCCommandColorOperationPayload struct {
	op OSCParsersColorOperation

	// Zig: color.List, default .{} (empty). The zero value is the empty
	// list.
	requests OSCParsersColorList

	// Zig: default .st, which is the zero value.
	terminator OSCTerminator
}

// OSCCommandShowDesktopNotificationPayload is the payload of
// Command.show_desktop_notification.
type OSCCommandShowDesktopNotificationPayload struct {
	title []uint8
	body  []uint8
}

// OSCCommandHyperlinkStartPayload is the payload of Command.hyperlink_start.
type OSCCommandHyperlinkStartPayload struct {
	// Zig: ?[:0]const u8, default null.
	id  Optional[[]uint8]
	uri []uint8
}

// OSCCommandConemuSleepPayload is the payload of Command.conemu_sleep.
type OSCCommandConemuSleepPayload struct {
	duration_ms uint16
}

// OSCCommandConemuChangeTabTitlePayloadTag is the tag of the payload of
// Command.conemu_change_tab_title.
type OSCCommandConemuChangeTabTitlePayloadTag uint8

const (
	OSCCommandConemuChangeTabTitlePayloadReset OSCCommandConemuChangeTabTitlePayloadTag = iota
	OSCCommandConemuChangeTabTitlePayloadValue
)

// OSCCommandConemuChangeTabTitlePayload is the payload of
// Command.conemu_change_tab_title:
// union(enum) { reset, value: [:0]const u8 }.
//
// Zig: a tagged union. Only the field named by Tag is meaningful; reset
// carries no payload.
type OSCCommandConemuChangeTabTitlePayload struct {
	Tag OSCCommandConemuChangeTabTitlePayloadTag

	Value []uint8
}

// OSCCommandConemuXtermEmulationPayload is the payload of
// Command.conemu_xterm_emulation.
type OSCCommandConemuXtermEmulationPayload struct {
	// null => do not change
	// false => turn off
	// true => turn on
	keyboard Optional[bool]
	// null => do not change
	// false => turn off
	// true => turn on
	output Optional[bool]
}

// OSCCommandSemanticPrompt is osc.Command.SemanticPrompt:
// parsers.semantic_prompt.Command.
type OSCCommandSemanticPrompt = OSCParsersSemanticPromptCommand

// OSCCommandKittyClipboardProtocol is osc.Command.KittyClipboardProtocol:
// parsers.kitty_clipboard_protocol.OSC.
type OSCCommandKittyClipboardProtocol = OSCParsersKittyClipboardProtocolOSC

// OSCCommandKey is osc.Command.Key: the tag type of Command.
//
// Zig: LibEnum with the .zig target in the library build, i.e. a plain enum
// with these values in this order. NOTE: Order matters, see LibEnum
// documentation.
//
// The constants are named OSCCommandKey<Value> (the enum rule) rather than
// OSCCommand<Value>, because OSCCommandSemanticPrompt and
// OSCCommandKittyClipboardProtocol are already the names of the two type
// aliases above.
type OSCCommandKey uint8

const (
	OSCCommandKeyInvalid OSCCommandKey = iota
	OSCCommandKeyChangeWindowTitle
	OSCCommandKeyChangeWindowIcon
	OSCCommandKeySemanticPrompt
	OSCCommandKeyClipboardContents
	OSCCommandKeyReportPwd
	OSCCommandKeyMouseShape
	OSCCommandKeyColorOperation
	OSCCommandKeyKittyColorProtocol
	OSCCommandKeyShowDesktopNotification
	OSCCommandKeyHyperlinkStart
	OSCCommandKeyHyperlinkEnd
	OSCCommandKeyConemuSleep
	OSCCommandKeyConemuShowMessageBox
	OSCCommandKeyConemuChangeTabTitle
	OSCCommandKeyConemuProgressReport
	OSCCommandKeyConemuWaitInput
	OSCCommandKeyConemuGuimacro
	OSCCommandKeyConemuRunProcess
	OSCCommandKeyConemuOutputEnvironmentVariable
	OSCCommandKeyConemuXtermEmulation
	OSCCommandKeyConemuComment
	OSCCommandKeyKittyTextSizing
	OSCCommandKeyKittyClipboardProtocol
	OSCCommandKeyContextSignal

	oscCommandKeyCount = iota
)

// OSCCommandTag is std.meta.Tag(Command), which is Key. It exists so that
// the tag type of this tagged union can also be named the way every other
// tagged union's is.
type OSCCommandTag = OSCCommandKey

// String returns the Zig tag name (@tagName).
func (self OSCCommandKey) String() string { panic(notImplemented) }

// OSCCommandProgressReport is osc.Command.ProgressReport.
//
// The C ABI declarations (`C`, `cval`) are not ported.
type OSCCommandProgressReport struct {
	state OSCCommandProgressReportState

	// Zig: ?u8, default null.
	progress Optional[uint8]
}

// OSCCommandProgressReportState is osc.Command.ProgressReport.State.
//
// Zig: enum(c_int).
type OSCCommandProgressReportState int32

const (
	OSCCommandProgressReportStateRemove        OSCCommandProgressReportState = 0
	OSCCommandProgressReportStateSet           OSCCommandProgressReportState = 1
	OSCCommandProgressReportStateError         OSCCommandProgressReportState = 2
	OSCCommandProgressReportStateIndeterminate OSCCommandProgressReportState = 3
	OSCCommandProgressReportStatePause         OSCCommandProgressReportState = 4
)

// String returns the Zig tag name (@tagName).
func (self OSCCommandProgressReportState) String() string { panic(notImplemented) }

// OSCTerminator is osc.Terminator: the terminator used to end an OSC
// command. For OSC commands that demand a response, we try to match the
// terminator used in the request since that is most likely to be accepted by
// the calling program.
//
// The C ABI declarations (`C`, `cval`) are not ported.
type OSCTerminator uint8

const (
	// The preferred string terminator is ESC followed by \
	OSCTerminatorSt OSCTerminator = 0

	// Some applications and terminals use BELL (0x07) as the string
	// terminator.
	OSCTerminatorBel OSCTerminator = 1
)

// OSCTerminatorInit is osc.Terminator.init: initialize the terminator based
// on the last byte seen. If the last byte is a BEL then we use BEL,
// otherwise we just assume ST.
func OSCTerminatorInit(ch Optional[uint8]) OSCTerminator { panic(notImplemented) }

// String is osc.Terminator.string: the terminator as a string. This is
// static memory so it doesn't need to be freed.
//
// Note: this is Zig's `string` method (the bytes that end a reply), not the
// tag name; it returns []uint8 like every other `[]const u8`.
func (self OSCTerminator) String() []uint8 { panic(notImplemented) }

// TagName returns the Zig tag name (@tagName): "st" or "bel". Every other
// enum uses String() for this; here String is taken by Zig's own `string`
// function above, which returns the terminator bytes.
func (self OSCTerminator) TagName() string { panic(notImplemented) }

// Format is osc.Terminator.format: writes the terminator bytes. Per
// docs/mapping.md a Zig `format` method becomes `Format() string` with
// output identical to Zig's.
func (self OSCTerminator) Format() string { panic(notImplemented) }

// OSCParser is osc.Parser.
type OSCParser struct {
	// Optional allocator used to accept data longer than MAX_BUF.
	// This only applies to some commands (e.g. OSC 52) that can
	// reasonably exceed MAX_BUF.
	//
	// Zig: ?Allocator. Allocator parameters are dropped in the port, but
	// whether this one is null changes what the parser accepts
	// (ensureAllocator, writeToAllocating, the colour parsers), so the
	// field is kept: nil is Zig's null, any other value is "an allocator
	// was provided". Nothing is ever allocated through it.
	alloc any

	// Current state of the parser.
	state OSCParserState

	// Buffer for temporary storage of OSC data
	buffer [OSCParserMAX_BUF]uint8
	// Fixed writer for accumulating OSC data
	//
	// Zig: ?std.Io.Writer.
	fixed Optional[oscWriter]
	// Allocating writer for accumulating OSC data
	//
	// Zig: ?std.Io.Writer.Allocating.
	allocating Optional[oscWriterAllocating]
	// Pointer to the active writer for accumulating OSC data
	//
	// Zig: ?*std.Io.Writer; nil is null. It points at fixed.Value or at
	// allocating.Value.writer.
	writer *oscWriter

	// The command that is the result of parsing.
	command OSCCommand
}

// OSCParserMAX_BUF is osc.Parser.MAX_BUF: maximum size of a "normal" OSC.
const OSCParserMAX_BUF = 2048

// OSCParserState is osc.Parser.State.
type OSCParserState uint8

const (
	OSCParserStateStart OSCParserState = iota
	OSCParserStateInvalid

	// OSC command prefixes. Not all of these are valid OSCs, but may be
	// needed to "bridge" to a valid OSC (e.g. to support OSC 777 we need to
	// have a state "77" even though there is no OSC 77).
	OSCParserState0
	OSCParserState1
	OSCParserState2
	OSCParserState3
	OSCParserState4
	OSCParserState5
	OSCParserState6
	OSCParserState7
	OSCParserState8
	OSCParserState9
	OSCParserState30
	OSCParserState300
	OSCParserState3008
	OSCParserState10
	OSCParserState11
	OSCParserState12
	OSCParserState13
	OSCParserState14
	OSCParserState15
	OSCParserState16
	OSCParserState17
	OSCParserState18
	OSCParserState19
	OSCParserState21
	OSCParserState22
	OSCParserState52
	OSCParserState55
	OSCParserState66
	OSCParserState77
	OSCParserState104
	OSCParserState110
	OSCParserState111
	OSCParserState112
	OSCParserState113
	OSCParserState114
	OSCParserState115
	OSCParserState116
	OSCParserState117
	OSCParserState118
	OSCParserState119
	OSCParserState133
	OSCParserState552
	OSCParserState777
	OSCParserState1337
	OSCParserState5522

	oscParserStateCount = iota
)

// String returns the Zig tag name (@tagName; the "{t}" format in the
// parser's log lines).
func (self OSCParserState) String() string { panic(notImplemented) }

// OSCParserInit is osc.Parser.init.
//
// Placeholder body kept from the previous osc.go; not the implementation.
func OSCParserInit(alloc any) OSCParser {
	return OSCParser{}
}

// Deinit is osc.Parser.deinit: this must be called to clean up any allocated
// memory.
//
// Placeholder body kept from the previous osc.go; not the implementation.
func (self *OSCParser) Deinit() {}

// Reset is osc.Parser.reset: reset the parser state.
//
// Placeholder body kept from the previous osc.go; not the implementation.
func (self *OSCParser) Reset() {}

// ensureAllocator is osc.Parser.ensureAllocator: make sure that we have an
// allocator. If we don't, set the state to invalid so that any additional
// OSC data is discarded.
func (self *OSCParser) ensureAllocator() bool { panic(notImplemented) }

// writeToFixed is osc.Parser.writeToFixed: set up a fixed Writer to collect
// the rest of the OSC data.
func (self *OSCParser) writeToFixed() { panic(notImplemented) }

// writeToAllocating is osc.Parser.writeToAllocating: set up an allocating
// Writer to collect the rest of the OSC data. If we don't have an allocator
// or setting up the allocator fails, fall back to writing to a fixed buffer
// and hope that it's big enough.
func (self *OSCParser) writeToAllocating() { panic(notImplemented) }

// Next is osc.Parser.next: consume the next character c and advance the
// parser state.
//
// Placeholder body kept from the previous osc.go; not the implementation.
func (self *OSCParser) Next(c uint8) {}

// End is osc.Parser.end: end the sequence and return the command, if any.
// If the return value is null, then no valid command was found. The optional
// terminator_ch is the final character in the OSC sequence. This is used to
// determine the response terminator.
//
// The returned pointer is only valid until the next call to the parser.
// Callers should copy out any data they wish to retain across calls.
//
// Placeholder body kept from the previous osc.go; not the implementation.
// It never yields a command.
func (self *OSCParser) End(terminator_ch Optional[uint8]) *OSCCommand {
	return nil
}
