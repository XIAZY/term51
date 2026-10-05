// Ported from src/terminal/stream.zig (Ghostty v1.3.1).
//
// File naming: stream.zig is a namespace file, so its declarations take the
// prefix `Stream` (`stream` for private ones): `stream.Action` is
// `StreamAction`, `stream.Action.Key` is `StreamActionKey`. The type that
// `stream.Stream(Handler)` returns is `Stream[H]`.
//
// Declarations appear in the same order as in stream.zig.
//
// The generic. Zig: `pub fn Stream(comptime Handler: type) type`, a function
// that returns a struct type for a handler type, which may be a struct or a
// pointer to one. Go: `type Stream[H StreamHandler] struct`, with the Zig
// struct's functions as methods on `*Stream[H]` and its `init`/`initAlloc`
// as the generic functions `StreamInit`/`StreamInitAlloc`. What Zig requires
// of a handler by duck typing is written as the interface StreamHandler:
//
//   - `vt`, called as `self.handler.vt(.<key>, <payload>)` throughout, with
//     the documented signature
//     `fn(comptime action: Action.Key, value: Action.Value(action)) !void`.
//     Go has no comptime parameter and no type that depends on a value, so
//     it is `Vt(action StreamActionKey, value StreamAction) error`: `value`
//     is the whole tagged union with `value.Tag == action`, and the payload
//     Zig passes is the field of `value` named by the tag. For a variant
//     with no payload Zig passes `{}` and no field is meaningful.
//     `try self.handler.vt(.k, v)` is therefore
//     `if err := self.handler.Vt(StreamActionKeyK, StreamAction{Tag:
//     StreamActionKeyK, K: v}); err != nil { return err }`.
//   - `deinit`, called by Stream.deinit ("The Handler type must also have a
//     `deinit` function"). It is `Deinit()`.
//   - `vtRaw` is optional in Zig (`@hasDecl(T, "vtRaw")` in nextNonUtf8). It
//     is the separate interface StreamHandlerVtRaw, which a handler may
//     also implement; the comptime `@hasDecl` is a run-time type assertion
//     on the handler.
//
// Not ported, see docs/mapping.md:
//   - the C ABI declarations: `lib_target`, the C variant of `Action.Key`
//     (lib.Enum with the .c target), `Action.c_union`, `Action.Value`,
//     `Action.C`, `Action.CValue`, `Action.cval`, and the `C` and `cval`
//     declarations of `Action.Print`, `DeviceStatus`, `Mode`,
//     `KittyKeyboardFlags`, `WindowTitle`, `ReportPwd`,
//     `ShowDesktopNotification`, `StartHyperlink`, `ClipboardContents` and
//     `ColorOperation`; `lib.Struct(lib_target, ...)` and `extern struct`
//     are ordinary structs here;
//   - the SIMD branches: build_options.simd is false for the reference, so
//     `nextSlice` is the scalar loop and the `simd` import is unused.
//     `nextSliceCapped` is only called from the SIMD branch of `nextSlice`
//     but is ported, because the project tests call it directly. Its call
//     to `simd.vt.utf8DecodeUntilControlSeq` is the stand-in
//     streamUtf8DecodeUntilControlSeq (declared after it), which decodes
//     exactly as the scalar per-byte path does (docs/mapping.md: the port
//     returns what nextSlice returns for the same bytes);
//   - `const log = std.log.scoped(.stream)`: the port uses
//     warn("stream", ...) for `log.warn` and `log.err`; `log.info` and
//     `log.debug` are not printed (the calls are kept as comments);
//   - `@branchHint` and `@setEvalBranchQuota` have no Go counterpart;
//   - inside `Stream(Handler)`: `Self` (the receiver type), `Action` (a
//     re-export of stream.Action; a Go generic type cannot hold a nested
//     type, use StreamAction) and `T` (the handler type with a pointer
//     removed, used only for @hasDecl).
//
// Labeled blocks. Every labeled block of csiDispatch and escDispatch that
// is left with `break :label` (`mode:`, `decrqm:`, `capture:`, `push:`,
// `set:`, `decsasd:`, `blk:`) is a whole prong of the outer
// `switch (input.final)` / `switch (action.final)`, and nothing follows
// that switch in the function. Leaving the block is leaving the switch, so
// it is `break prong`, a labeled break of the Go switch (a plain `break`
// would only leave an inner switch). Value-producing inner blocks
// (`ansi:`, `question:`) are written as if/else that assign the value; the
// one that is left from inside a nested switch (`ansi:` of DECRQM) uses
// `goto`, as docs/mapping.md prescribes.

package terminal

import (
	"bytes"
	"unicode/utf8"
)

// streamDebug is stream.debug: flip this to true when you want verbose debug
// output for debugging terminal stream issues. In addition to louder output
// this will also disable the SIMD optimizations in order to make it easier
// to see every byte.
const streamDebug = false

// StreamAction is stream.Action: the possible actions that can be emitted by
// the Stream function for handling.
//
// Zig: union(Key). Only the field named by Tag is meaningful. Variants with
// no payload have no field.
type StreamAction struct {
	Tag StreamActionKey

	Print StreamActionPrint
	// Zig: usize.
	PrintRepeat       uint
	HorizontalTab     uint16
	HorizontalTabBack uint16
	InvokeCharset     StreamActionInvokeCharset
	CursorUp          StreamActionCursorMovement
	CursorDown        StreamActionCursorMovement
	CursorLeft        StreamActionCursorMovement
	CursorRight       StreamActionCursorMovement
	CursorCol         StreamActionCursorMovement
	CursorRow         StreamActionCursorMovement
	CursorColRelative StreamActionCursorMovement
	CursorRowRelative StreamActionCursorMovement
	CursorPos         StreamActionCursorPos
	CursorStyle       ANSICursorStyle

	EraseDisplayBelow               bool
	EraseDisplayAbove               bool
	EraseDisplayComplete            bool
	EraseDisplayScrollback          bool
	EraseDisplayScrollComplete      bool
	EraseLineRight                  bool
	EraseLineLeft                   bool
	EraseLineComplete               bool
	EraseLineRightUnlessPendingWrap bool

	// Zig: usize, all seven.
	DeleteChars  uint
	EraseChars   uint
	InsertLines  uint
	InsertBlanks uint
	DeleteLines  uint
	ScrollUp     uint
	ScrollDown   uint

	SetMode             StreamActionMode
	ResetMode           StreamActionMode
	SaveMode            StreamActionMode
	RestoreMode         StreamActionMode
	RequestMode         StreamActionMode
	RequestModeUnknown  StreamActionRawMode
	TopAndBottomMargin  StreamActionMargin
	LeftAndRightMargin  StreamActionMargin
	ModifyKeyFormat     ANSIModifyKeyFormat
	MouseShiftCapture   bool
	SizeReport          CSISizeReportStyle
	TitlePush           uint16
	TitlePop            uint16
	DeviceAttributes    ANSIDeviceAttributeReq
	DeviceStatus        StreamActionDeviceStatus
	KittyKeyboardPush   StreamActionKittyKeyboardFlags
	KittyKeyboardPop    uint16
	KittyKeyboardSet    StreamActionKittyKeyboardFlags
	KittyKeyboardSetOr  StreamActionKittyKeyboardFlags
	KittyKeyboardSetNot StreamActionKittyKeyboardFlags
	DcsHook             ParserActionDCS
	DcsPut              uint8
	ApcPut              uint8
	ActiveStatusDisplay ANSIStatusDisplay

	WindowTitle             StreamActionWindowTitle
	ReportPwd               StreamActionReportPwd
	ShowDesktopNotification StreamActionShowDesktopNotification
	ProgressReport          OSCCommandProgressReport
	StartHyperlink          StreamActionStartHyperlink
	ClipboardContents       StreamActionClipboardContents
	MouseShape              MouseShape
	ConfigureCharset        StreamActionConfigureCharset
	SetAttribute            SGRAttribute
	KittyColorReport        KittyColorOSC
	ColorOperation          StreamActionColorOperation
	SemanticPrompt          StreamActionSemanticPrompt
}

// StreamActionKey is stream.Action.Key: the tag type of Action.
//
// Zig: lib.Enum(lib_target, ...), which for the Zig target is an ordinary
// enum whose values are numbered from zero in the order listed.
//
// The constants are named StreamActionKey<Value> (the enum rule, as for
// OSCCommandKey in osc.go) rather than StreamAction<Value>, because
// StreamActionPrint, StreamActionInvokeCharset, StreamActionCursorPos and
// others are the names of the payload types below.
type StreamActionKey uint8

const (
	StreamActionKeyPrint StreamActionKey = iota
	StreamActionKeyPrintRepeat
	StreamActionKeyBell
	StreamActionKeyBackspace
	StreamActionKeyHorizontalTab
	StreamActionKeyHorizontalTabBack
	StreamActionKeyLinefeed
	StreamActionKeyCarriageReturn
	StreamActionKeyEnquiry
	StreamActionKeyInvokeCharset
	StreamActionKeyCursorUp
	StreamActionKeyCursorDown
	StreamActionKeyCursorLeft
	StreamActionKeyCursorRight
	StreamActionKeyCursorCol
	StreamActionKeyCursorRow
	StreamActionKeyCursorColRelative
	StreamActionKeyCursorRowRelative
	StreamActionKeyCursorPos
	StreamActionKeyCursorStyle
	StreamActionKeyEraseDisplayBelow
	StreamActionKeyEraseDisplayAbove
	StreamActionKeyEraseDisplayComplete
	StreamActionKeyEraseDisplayScrollback
	StreamActionKeyEraseDisplayScrollComplete
	StreamActionKeyEraseLineRight
	StreamActionKeyEraseLineLeft
	StreamActionKeyEraseLineComplete
	StreamActionKeyEraseLineRightUnlessPendingWrap
	StreamActionKeyDeleteChars
	StreamActionKeyEraseChars
	StreamActionKeyInsertLines
	StreamActionKeyInsertBlanks
	StreamActionKeyDeleteLines
	StreamActionKeyScrollUp
	StreamActionKeyScrollDown
	StreamActionKeyTabClearCurrent
	StreamActionKeyTabClearAll
	StreamActionKeyTabSet
	StreamActionKeyTabReset
	StreamActionKeyIndex
	StreamActionKeyNextLine
	StreamActionKeyReverseIndex
	StreamActionKeyFullReset
	StreamActionKeySetMode
	StreamActionKeyResetMode
	StreamActionKeySaveMode
	StreamActionKeyRestoreMode
	StreamActionKeyRequestMode
	StreamActionKeyRequestModeUnknown
	StreamActionKeyTopAndBottomMargin
	StreamActionKeyLeftAndRightMargin
	StreamActionKeyLeftAndRightMarginAmbiguous
	StreamActionKeySaveCursor
	StreamActionKeyRestoreCursor
	StreamActionKeyModifyKeyFormat
	StreamActionKeyMouseShiftCapture
	StreamActionKeyProtectedModeOff
	StreamActionKeyProtectedModeIso
	StreamActionKeyProtectedModeDec
	StreamActionKeySizeReport
	StreamActionKeyTitlePush
	StreamActionKeyTitlePop
	StreamActionKeyXtversion
	StreamActionKeyDeviceAttributes
	StreamActionKeyDeviceStatus
	StreamActionKeyKittyKeyboardQuery
	StreamActionKeyKittyKeyboardPush
	StreamActionKeyKittyKeyboardPop
	StreamActionKeyKittyKeyboardSet
	StreamActionKeyKittyKeyboardSetOr
	StreamActionKeyKittyKeyboardSetNot
	StreamActionKeyDcsHook
	StreamActionKeyDcsPut
	StreamActionKeyDcsUnhook
	StreamActionKeyApcStart
	StreamActionKeyApcEnd
	StreamActionKeyApcPut
	StreamActionKeyEndHyperlink
	StreamActionKeyActiveStatusDisplay
	StreamActionKeyDecaln
	StreamActionKeyWindowTitle
	StreamActionKeyReportPwd
	StreamActionKeyShowDesktopNotification
	StreamActionKeyProgressReport
	StreamActionKeyStartHyperlink
	StreamActionKeyClipboardContents
	StreamActionKeyMouseShape
	StreamActionKeyConfigureCharset
	StreamActionKeySetAttribute
	StreamActionKeyKittyColorReport
	StreamActionKeyColorOperation
	StreamActionKeySemanticPrompt

	streamActionKeyCount = iota
)

// String returns the Zig tag name (@tagName).
func (self StreamActionKey) String() string {
	return streamActionKeyNames[self]
}

var streamActionKeyNames = [streamActionKeyCount]string{
	"print",
	"print_repeat",
	"bell",
	"backspace",
	"horizontal_tab",
	"horizontal_tab_back",
	"linefeed",
	"carriage_return",
	"enquiry",
	"invoke_charset",
	"cursor_up",
	"cursor_down",
	"cursor_left",
	"cursor_right",
	"cursor_col",
	"cursor_row",
	"cursor_col_relative",
	"cursor_row_relative",
	"cursor_pos",
	"cursor_style",
	"erase_display_below",
	"erase_display_above",
	"erase_display_complete",
	"erase_display_scrollback",
	"erase_display_scroll_complete",
	"erase_line_right",
	"erase_line_left",
	"erase_line_complete",
	"erase_line_right_unless_pending_wrap",
	"delete_chars",
	"erase_chars",
	"insert_lines",
	"insert_blanks",
	"delete_lines",
	"scroll_up",
	"scroll_down",
	"tab_clear_current",
	"tab_clear_all",
	"tab_set",
	"tab_reset",
	"index",
	"next_line",
	"reverse_index",
	"full_reset",
	"set_mode",
	"reset_mode",
	"save_mode",
	"restore_mode",
	"request_mode",
	"request_mode_unknown",
	"top_and_bottom_margin",
	"left_and_right_margin",
	"left_and_right_margin_ambiguous",
	"save_cursor",
	"restore_cursor",
	"modify_key_format",
	"mouse_shift_capture",
	"protected_mode_off",
	"protected_mode_iso",
	"protected_mode_dec",
	"size_report",
	"title_push",
	"title_pop",
	"xtversion",
	"device_attributes",
	"device_status",
	"kitty_keyboard_query",
	"kitty_keyboard_push",
	"kitty_keyboard_pop",
	"kitty_keyboard_set",
	"kitty_keyboard_set_or",
	"kitty_keyboard_set_not",
	"dcs_hook",
	"dcs_put",
	"dcs_unhook",
	"apc_start",
	"apc_end",
	"apc_put",
	"end_hyperlink",
	"active_status_display",
	"decaln",
	"window_title",
	"report_pwd",
	"show_desktop_notification",
	"progress_report",
	"start_hyperlink",
	"clipboard_contents",
	"mouse_shape",
	"configure_charset",
	"set_attribute",
	"kitty_color_report",
	"color_operation",
	"semantic_prompt",
}

// StreamActionTag is stream.Action.Tag: `c_union.Tag`, the tag type of the
// union, which is Key.
type StreamActionTag = StreamActionKey

// Field types

// StreamActionPrint is stream.Action.Print.
type StreamActionPrint struct {
	// Zig: u21.
	cp uint32
}

// StreamActionInvokeCharset is stream.Action.InvokeCharset.
type StreamActionInvokeCharset struct {
	bank    CharsetsActiveSlot
	charset CharsetsSlots
	locking bool
}

// StreamActionCursorMovement is stream.Action.CursorMovement.
type StreamActionCursorMovement struct {
	// The value of the cursor movement. Depending on the tag of this union
	// this may be an absolute value or it may be a relative value. For
	// example, `cursor_up` is relative, but `cursor_row` is absolute.
	value uint16
}

// StreamActionCursorPos is stream.Action.CursorPos.
type StreamActionCursorPos struct {
	row uint16
	col uint16
}

// StreamActionDeviceStatus is stream.Action.DeviceStatus.
type StreamActionDeviceStatus struct {
	request DeviceStatusRequest
}

// StreamActionMode is stream.Action.Mode.
type StreamActionMode struct {
	mode ModesMode
}

// StreamActionRawMode is stream.Action.RawMode.
type StreamActionRawMode struct {
	mode uint16
	ansi bool
}

// StreamActionMargin is stream.Action.Margin.
type StreamActionMargin struct {
	top_left     uint16
	bottom_right uint16
}

// StreamActionKittyKeyboardFlags is stream.Action.KittyKeyboardFlags.
type StreamActionKittyKeyboardFlags struct {
	// Zig: kitty.KeyFlags, which is kitty/key.zig's Flags.
	flags KittyKeyFlags
}

// StreamActionWindowTitle is stream.Action.WindowTitle.
type StreamActionWindowTitle struct {
	title []uint8
}

// StreamActionReportPwd is stream.Action.ReportPwd.
type StreamActionReportPwd struct {
	url []uint8
}

// StreamActionShowDesktopNotification is
// stream.Action.ShowDesktopNotification.
type StreamActionShowDesktopNotification struct {
	title []uint8
	body  []uint8
}

// StreamActionStartHyperlink is stream.Action.StartHyperlink.
type StreamActionStartHyperlink struct {
	uri []uint8
	// Zig: ?[]const u8.
	id Optional[[]uint8]
}

// StreamActionClipboardContents is stream.Action.ClipboardContents.
type StreamActionClipboardContents struct {
	kind uint8
	data []uint8
}

// StreamActionConfigureCharset is stream.Action.ConfigureCharset.
type StreamActionConfigureCharset struct {
	slot    CharsetsSlots
	charset CharsetsCharset
}

// StreamActionColorOperation is stream.Action.ColorOperation.
type StreamActionColorOperation struct {
	// Zig: osc.color.Operation.
	op OSCParsersColorOperation
	// Zig: osc.color.List.
	requests   OSCParsersColorList
	terminator OSCTerminator
}

// StreamActionSemanticPrompt is stream.Action.SemanticPrompt:
// osc.Command.SemanticPrompt.
type StreamActionSemanticPrompt = OSCCommandSemanticPrompt

// StreamHandler is what stream.Stream requires of its `Handler` type. Zig
// has no declaration for it: the requirement is the doc comment of Stream
// and the calls `self.handler.vt(...)` and `self.handler.deinit()`. See the
// file header.
type StreamHandler interface {
	// Vt is the handler's `vt`:
	//
	//   fn(comptime action: Action.Key, value: Action.Value(action)) !void
	//
	// The handler can choose to react to whatever actions it cares about.
	// `value.Tag` equals `action`; the payload is the field of `value`
	// named by it.
	Vt(action StreamActionKey, value StreamAction) error

	// Deinit is the handler's `deinit`. The stream takes ownership of the
	// handler and calls this when the stream is deinitialized.
	Deinit()
}

// StreamHandlerVtRaw is the optional handler function `vtRaw`, which Zig
// detects with `@hasDecl(T, "vtRaw")`. A handler can expose this to get the
// raw action before it is further parsed. If this returns `true` then the
// stream skips processing the action itself; an error is logged and treated
// as `true`.
type StreamHandlerVtRaw interface {
	VtRaw(action ParserAction) (bool, error)
}

// Stream is the type returned by stream.Stream(Handler): a type that can
// process a stream of tty control characters. This will call the `Vt`
// function of the handler for every action. See the file header for how the
// Zig type function is expressed.
type Stream[H StreamHandler] struct {
	handler     H
	parser      Parser
	utf8decoder UTF8Decoder
}

// StreamInit is Stream(Handler).init: initialize an allocation-free stream.
// This will preallocate various sizes as necessary and anything over that
// will be dropped. If you want to support more dynamic behavior use
// initAlloc instead.
//
// As a concrete example of something that requires heap allocation,
// consider OSC 52 (clipboard operations) which can be arbitrarily
// large.
//
// If you want to limit allocation size, use an allocator with
// a size limit with initAlloc.
//
// This takes ownership of the handler and will call deinit
// when the stream is deinitialized.
func StreamInit[H StreamHandler](h H) Stream[H] {
	return Stream[H]{
		handler:     h,
		parser:      ParserInit(),
		utf8decoder: UTF8Decoder{},
	}
}

// StreamInitAlloc is Stream(Handler).initAlloc: initialize the stream that
// supports heap allocation as necessary.
//
// Zig: `alloc` is an Allocator that is stored in parser.osc_parser.alloc.
// As for OSCParserInit, nothing is allocated through it; nil is Zig's null
// and any other value is "an allocator was provided".
func StreamInitAlloc[H StreamHandler](alloc any, h H) Stream[H] {
	var self = StreamInit(h)
	self.parser.osc_parser.alloc = alloc
	return self
}

// Deinit is Stream(Handler).deinit.
func (self *Stream[H]) Deinit() {
	self.parser.Deinit()
	self.handler.Deinit()
}

// NextSlice is Stream(Handler).nextSlice: process a string of characters.
//
// Zig: the rest of the function (the SIMD path, splitting the input into
// chunks of 4096 codepoints for nextSliceCapped) is behind
// `comptime debug or !build_options.simd`, which is true for the reference
// build, so only the scalar loop is ported.
func (self *Stream[H]) NextSlice(input []uint8) error {
	// Disable SIMD optimizations if build requests it or if our
	// manual debug mode is on.
	for _, c := range input {
		if err := self.Next(c); err != nil {
			return err
		}
	}
	return nil
}

// nextSliceCapped is Stream(Handler).nextSliceCapped.
//
// Zig asserts input.len <= cp_buf.len. Upstream decodes with SIMD here; the
// port calls streamUtf8DecodeUntilControlSeq, see the file header.
func (self *Stream[H]) nextSliceCapped(input []uint8, cp_buf []uint32) error {
	assert(len(input) <= len(cp_buf))

	var offset uint = 0

	// If the scalar UTF-8 decoder was in the middle of processing
	// a code sequence, we continue until it's not.
	for self.utf8decoder.state != 0 {
		if offset >= uint(len(input)) {
			return nil
		}
		if err := self.nextUtf8(input[offset]); err != nil {
			return err
		}
		offset += 1
	}
	if offset >= uint(len(input)) {
		return nil
	}

	// If we're not in the ground state then we process until
	// we are. This can happen if the last chunk of input put us
	// in the middle of a control sequence.
	n, err := self.consumeUntilGround(input[offset:])
	if err != nil {
		return err
	}
	offset += n
	if offset >= uint(len(input)) {
		return nil
	}
	n, err = self.consumeAllEscapes(input[offset:])
	if err != nil {
		return err
	}
	offset += n

	// If we're in the ground state then we can use SIMD to process
	// input until we see an ESC (0x1B), since all other characters
	// up to that point are just UTF-8.
	for self.parser.state == ParserStateGround && offset < uint(len(input)) {
		res := streamUtf8DecodeUntilControlSeq(input[offset:], cp_buf)
		for _, cp := range cp_buf[0:res.decoded] {
			if cp <= 0xF {
				if err := self.Execute(uint8(cp)); err != nil {
					return err
				}
			} else {
				if err := self.Print(cp); err != nil {
					return err
				}
			}
		}
		// Consume the bytes we just processed.
		offset += res.consumed

		if offset >= uint(len(input)) {
			return nil
		}

		// If our offset is NOT an escape then we must have a
		// partial UTF-8 sequence. In that case, we pass it off
		// to the scalar parser.
		if input[offset] != 0x1B {
			rem := input[offset:]
			for _, c := range rem {
				if err := self.nextUtf8(c); err != nil {
					return err
				}
			}
			return nil
		}

		// Process control sequences until we run out.
		n, err := self.consumeAllEscapes(input[offset:])
		if err != nil {
			return err
		}
		offset += n
	}
	return nil
}

// streamDecodeResult is simd/vt.zig's DecodeResult, the result of
// utf8DecodeUntilControlSeq.
//
// Zig: both fields are usize.
type streamDecodeResult struct {
	consumed uint
	decoded  uint
}

// streamUtf8DecodeUntilControlSeq stands in for
// simd.vt.utf8DecodeUntilControlSeq (simd/vt.zig), which nextSliceCapped
// calls: decode the UTF-8 in input up to the first ESC (0x1B) into output
// and report how many bytes were consumed and how many codepoints decoded.
// It is not a Ghostty function.
//
// Upstream's implementation is SIMD (vt.cpp), with a scalar fallback
// (utf8DecodeUntilControlSeqScalar) that differs from the per-byte path of
// the stream: it stops at the first invalid lead byte, so on input such as
// `FF ESC [ 1 A` nextSliceCapped feeds the ESC and the bytes after it to
// nextUtf8 and fails its assert (term51 docs/upstream-findings.md section
// 12), and on `E2 82 (` it gives two U+FFFD where the per-byte path gives
// one. The port must return what nextSlice returns for the same bytes
// (docs/mapping.md, `parser/nextslicecapped_inputs`), so this stand-in
// decodes with the stream's own UTF8Decoder, byte by byte, exactly as
// nextUtf8 would from the ground state with an idle decoder (nextSliceCapped
// only calls it in that state):
//
//   - every codepoint the decoder emits for the bytes before the first ESC
//     is written to output, including U+FFFD for ill-formed sequences, and
//     a byte the decoder does not consume is fed again, as in nextUtf8;
//   - consumed covers every byte up to the end of the last complete (or
//     rejected) sequence. A sequence still incomplete at the end of the
//     input is not consumed: nextSliceCapped passes those bytes to
//     nextUtf8, which leaves them pending in the stream's decoder;
//   - a sequence still incomplete when the ESC is reached is ill-formed:
//     the decoder rejects the ESC (U+FFFD, ESC not consumed), so U+FFFD is
//     written and the bytes up to the ESC are consumed. nextSliceCapped
//     then handles the ESC itself, as handleCodepoint would after nextUtf8
//     feeds it again.
//
// The codepoints are below 0x1B or above it, never 0x1B, so nextSliceCapped's
// `cp <= 0xF` execute / print split is handleCodepoint's.
//
// Zig asserts output.len >= input.len. At most one codepoint is written
// per byte before the ESC.
func streamUtf8DecodeUntilControlSeq(input []uint8, output []uint32) streamDecodeResult {
	assert(len(output) >= len(input))

	// Find our escape
	idx := bytes.IndexByte(input, 0x1B)
	if idx < 0 {
		idx = len(input)
	}
	decode := input[0:idx]

	var decoder UTF8Decoder
	var decode_offset uint = 0
	var decode_count uint = 0
	var consumed uint = 0
	for decode_offset < uint(len(decode)) {
		cp, ok := decoder.Next(decode[decode_offset])
		if cp.Valid {
			output[decode_count] = cp.Value
			decode_count += 1
		}
		if ok {
			decode_offset += 1
		}
		if decoder.state == ACCEPT_STATE {
			consumed = decode_offset
		}
	}

	// A sequence broken by the ESC: the decoder rejects the ESC.
	if decoder.state != ACCEPT_STATE && idx < len(input) {
		cp, ok := decoder.Next(0x1B)
		assert(cp.Valid && cp.Value == 0xFFFD && !ok)
		output[decode_count] = cp.Value
		decode_count += 1
		consumed = uint(idx)
	}

	return streamDecodeResult{consumed: consumed, decoded: decode_count}
}

// consumeAllEscapes is Stream(Handler).consumeAllEscapes: parses
// back-to-back escape sequences until none are left. Returns the number of
// bytes consumed from the provided input.
//
// Expects input to start with 0x1B, use consumeUntilGround first if the
// stream may be in the middle of an escape sequence.
//
// Zig: returns !usize.
func (self *Stream[H]) consumeAllEscapes(input []uint8) (uint, error) {
	var offset uint = 0
	for input[offset] == 0x1B {
		self.parser.state = ParserStateEscape
		self.parser.Clear()
		offset += 1
		n, err := self.consumeUntilGround(input[offset:])
		if err != nil {
			return 0, err
		}
		offset += n
		if offset >= uint(len(input)) {
			return uint(len(input)), nil
		}
	}
	return offset, nil
}

// consumeUntilGround is Stream(Handler).consumeUntilGround: parses escape
// sequences until the parser reaches the ground state. Returns the number of
// bytes consumed from the provided input.
//
// Zig: returns !usize.
func (self *Stream[H]) consumeUntilGround(input []uint8) (uint, error) {
	var offset uint = 0
	for self.parser.state != ParserStateGround {
		if offset >= uint(len(input)) {
			return uint(len(input)), nil
		}
		if err := self.nextNonUtf8(input[offset]); err != nil {
			return 0, err
		}
		offset += 1
	}
	return offset, nil
}

// Next is Stream(Handler).next: like nextSlice but takes one byte and is
// necessarily a scalar operation that can't use SIMD. Prefer nextSlice if
// you can and try to get multiple bytes at once.
func (self *Stream[H]) Next(c uint8) error {
	// The scalar path can be responsible for decoding UTF-8.
	if self.parser.state == ParserStateGround {
		return self.nextUtf8(c)
	}

	return self.nextNonUtf8(c)
}

// nextUtf8 is Stream(Handler).nextUtf8: process the next byte and print as
// necessary.
//
// This assumes we're in the UTF-8 decoding state. If we may not be in the
// UTF-8 decoding state call nextSlice or next.
func (self *Stream[H]) nextUtf8(c uint8) error {
	assert(self.parser.state == ParserStateGround)

	res0, res1 := self.utf8decoder.Next(c)
	consumed := res1
	if res0.Valid {
		codepoint := res0.Value
		if err := self.handleCodepoint(codepoint); err != nil {
			return err
		}
	}
	if !consumed {
		// We optimize for the scenario where the text being
		// printed in the terminal ISN'T full of ill-formed
		// UTF-8 sequences.

		retry0, retry1 := self.utf8decoder.Next(c)
		// It should be impossible for the decoder
		// to not consume the byte twice in a row.
		assert(retry1 == true)
		if retry0.Valid {
			codepoint := retry0.Value
			if err := self.handleCodepoint(codepoint); err != nil {
				return err
			}
		}
	}
	return nil
}

// handleCodepoint is Stream(Handler).handleCodepoint: to be called whenever
// the utf-8 decoder produces a codepoint.
//
// This function is abstracted this way to handle the case where the decoder
// emits a 0x1B after rejecting an ill-formed sequence.
//
// Zig: c is u21.
func (self *Stream[H]) handleCodepoint(c uint32) error {
	// Zig raises the eval branch quota here (@setEvalBranchQuota) because
	// a lot of tests end up running almost completely at comptime due to
	// a chain of inline functions. Nothing to do in Go.

	// C0 control
	if c <= 0xF {
		return self.Execute(uint8(c))
	}
	// ESC
	if c == 0x1B {
		self.parser.state = ParserStateEscape
		self.parser.Clear()
		return nil
	}
	return self.Print(c)
}

// nextNonUtf8 is Stream(Handler).nextNonUtf8: process the next character and
// call any callbacks if necessary.
//
// This assumes that we're not in the UTF-8 decoding state. If we may be in
// the UTF-8 decoding state call nextSlice or next.
func (self *Stream[H]) nextNonUtf8(c uint8) error {
	assert(self.parser.state != ParserStateGround)

	// Fast path for CSI entry.
	if self.parser.state == ParserStateEscape && c == '[' {
		self.parser.state = ParserStateCsiEntry
		return nil
	}
	// Fast path for CSI params.
	if self.parser.state == ParserStateCsiParam {
		// csi_param: (a labeled block; `break :csi_param` is the goto
		// below)
		//
		// csi_param is the most common parser state
		// other than ground by a fairly wide margin.
		//
		// ref: https://github.com/qwerasd205/asciinema-stats
		switch {
		// A C0 escape (yes, this is valid):
		case c <= 0x0F:
			if err := self.Execute(c); err != nil {
				return err
			}
		// We ignore C0 escapes > 0xF since execute
		// doesn't have processing for them anyway:
		case (c >= 0x10 && c <= 0x17) || c == 0x19 || (c >= 0x1C && c <= 0x1F):
		// We don't currently have any handling for
		// 0x18 or 0x1A, but they should still move
		// the parser state to ground.
		case c == 0x18 || c == 0x1A:
			self.parser.state = ParserStateGround
		// A parameter digit:
		case c >= '0' && c <= '9':
			if self.parser.params_idx < MAX_PARAMS {
				self.parser.param_acc = saturatingMulU16(self.parser.param_acc, 10)
				self.parser.param_acc = saturatingAddU16(self.parser.param_acc, uint16(c-'0'))
				// The parser's CSI param action uses param_acc_idx
				// to decide if there's a final param that needs to
				// be consumed or not, but it doesn't matter really
				// what it is as long as it's not 0.
				self.parser.param_acc_idx |= 1
			}
		// A parameter separator:
		case c == ':' || c == ';':
			if self.parser.params_idx < MAX_PARAMS {
				self.parser.params[self.parser.params_idx] = self.parser.param_acc
				if c == ':' {
					self.parser.params_sep.set(int(self.parser.params_idx))
				}
				self.parser.params_idx += 1

				self.parser.param_acc = 0
				self.parser.param_acc_idx = 0
			}
		// Explicitly ignored:
		case c == 0x7F:
		// Defer to the state machine to
		// handle any other characters:
		default:
			goto csi_param // break :csi_param
		}
		return nil
	}
csi_param:

	// We explicitly inline this call here for performance reasons.
	//
	// We do this rather than mark Parser.next as inline because doing
	// that causes weird behavior in some tests- I'm not sure if they
	// miscompile or it's just very counter-intuitive comptime stuff,
	// but regardless, this is the easy solution.
	actions := self.parser.Next(c)

	for _, action_opt := range actions {
		if !action_opt.Valid {
			continue
		}
		action := action_opt.Value
		// Zig: if (comptime debug) log.info("action: {f}", .{action});

		// A handler can expose this to get the raw action before
		// it is further parsed. If this returns `true` then we skip
		// processing ourselves.
		if raw, ok := any(self.handler).(StreamHandlerVtRaw); ok {
			skip, err := raw.VtRaw(action)
			if err != nil {
				warn("stream", "error handling action manually err=error.%v action=%s",
					err,
					action.Format(),
				)
				// Always skip erroneous actions because we can't
				// be sure...
				skip = true
			}

			if skip {
				continue
			}
		}

		switch action.Tag {
		case ParserActionPrint:
			if err := self.Print(action.Print); err != nil {
				return err
			}
		case ParserActionExecute:
			if err := self.Execute(action.Execute); err != nil {
				return err
			}
		case ParserActionCsiDispatch:
			if err := self.csiDispatch(action.CsiDispatch); err != nil {
				return err
			}
		case ParserActionEscDispatch:
			if err := self.escDispatch(action.EscDispatch); err != nil {
				return err
			}
		case ParserActionOscDispatch:
			if err := self.oscDispatch(action.OscDispatch); err != nil {
				return err
			}
		case ParserActionDcsHook:
			if err := self.handler.Vt(StreamActionKeyDcsHook, StreamAction{Tag: StreamActionKeyDcsHook, DcsHook: action.DcsHook}); err != nil {
				return err
			}
		case ParserActionDcsPut:
			if err := self.handler.Vt(StreamActionKeyDcsPut, StreamAction{Tag: StreamActionKeyDcsPut, DcsPut: action.DcsPut}); err != nil {
				return err
			}
		case ParserActionDcsUnhook:
			if err := self.handler.Vt(StreamActionKeyDcsUnhook, StreamAction{Tag: StreamActionKeyDcsUnhook}); err != nil {
				return err
			}
		case ParserActionApcStart:
			if err := self.handler.Vt(StreamActionKeyApcStart, StreamAction{Tag: StreamActionKeyApcStart}); err != nil {
				return err
			}
		case ParserActionApcPut:
			if err := self.handler.Vt(StreamActionKeyApcPut, StreamAction{Tag: StreamActionKeyApcPut, ApcPut: action.ApcPut}); err != nil {
				return err
			}
		case ParserActionApcEnd:
			if err := self.handler.Vt(StreamActionKeyApcEnd, StreamAction{Tag: StreamActionKeyApcEnd}); err != nil {
				return err
			}
		default:
			panic("unreachable")
		}
	}
	return nil
}

// Print is Stream(Handler).print.
//
// Zig: c is u21.
func (self *Stream[H]) Print(c uint32) error {
	return self.handler.Vt(StreamActionKeyPrint, StreamAction{Tag: StreamActionKeyPrint, Print: StreamActionPrint{cp: c}})
}

// Execute is Stream(Handler).execute.
func (self *Stream[H]) Execute(c uint8) error {
	// If the character is > 0x7F, it's a C1 (8-bit) control,
	// which is strictly equivalent to `ESC` plus `c - 0x40`.
	if c > 0x7F {
		// Zig: log.info("executing C1 0x{x} as ESC {c}", .{ c, c - 0x40 });
		return self.escDispatch(ParserActionESC{
			Intermediates: []uint8{},
			Final:         c - 0x40,
		})
	}

	c0 := ANSIC0(c)
	// Zig: if (comptime debug) log.info("execute: {f}", .{c0});
	switch c0 {
	// We ignore SOH/STX: https://github.com/microsoft/terminal/issues/10786
	case ANSIC0NUL, ANSIC0SOH, ANSIC0STX:

	case ANSIC0ENQ:
		if err := self.handler.Vt(StreamActionKeyEnquiry, StreamAction{Tag: StreamActionKeyEnquiry}); err != nil {
			return err
		}
	case ANSIC0BEL:
		if err := self.handler.Vt(StreamActionKeyBell, StreamAction{Tag: StreamActionKeyBell}); err != nil {
			return err
		}
	case ANSIC0BS:
		if err := self.handler.Vt(StreamActionKeyBackspace, StreamAction{Tag: StreamActionKeyBackspace}); err != nil {
			return err
		}
	case ANSIC0HT:
		if err := self.handler.Vt(StreamActionKeyHorizontalTab, StreamAction{Tag: StreamActionKeyHorizontalTab, HorizontalTab: 1}); err != nil {
			return err
		}
	case ANSIC0LF, ANSIC0VT, ANSIC0FF:
		if err := self.handler.Vt(StreamActionKeyLinefeed, StreamAction{Tag: StreamActionKeyLinefeed}); err != nil {
			return err
		}
	case ANSIC0CR:
		if err := self.handler.Vt(StreamActionKeyCarriageReturn, StreamAction{Tag: StreamActionKeyCarriageReturn}); err != nil {
			return err
		}
	case ANSIC0SO:
		if err := self.handler.Vt(StreamActionKeyInvokeCharset, StreamAction{Tag: StreamActionKeyInvokeCharset, InvokeCharset: StreamActionInvokeCharset{bank: CharsetsActiveSlotGL, charset: CharsetsSlotsG1, locking: false}}); err != nil {
			return err
		}
	case ANSIC0SI:
		if err := self.handler.Vt(StreamActionKeyInvokeCharset, StreamAction{Tag: StreamActionKeyInvokeCharset, InvokeCharset: StreamActionInvokeCharset{bank: CharsetsActiveSlotGL, charset: CharsetsSlotsG0, locking: false}}); err != nil {
			return err
		}

	default:
		warn("stream", "invalid C0 character, ignoring: 0x%x", c)
	}
	return nil
}

// csiDispatch is Stream(Handler).csiDispatch.
func (self *Stream[H]) csiDispatch(input ParserActionCSI) error {
	// The branch hints here are based on real world data
	// which indicates that the most common CSI finals are:
	//
	// 1. m
	// 2. H
	// 3. K
	// 4. A
	// 5. C
	// 6. X
	// 7. l
	// 8. h
	// 9. r
	//
	// Together, these 9 finals make up about 96% of all
	// CSI sequences encountered in real world scenarios.
	//
	// Additionally, within the prongs, unlikely branch
	// hints have been added to branches that deal with
	// invalid sequences/commands, this is in order to
	// optimize for the happy path where we're getting
	// valid data from the program we're running.
	//
	// ref: https://github.com/qwerasd205/asciinema-stats

prong:
	switch input.Final {
	// CUU - Cursor Up
	case 'A', 'k':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid cursor up command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorUp, StreamAction{Tag: StreamActionKeyCursorUp, CursorUp: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI A with intermediates: %s",
				input.Intermediates,
			)
		}

	// CUD - Cursor Down
	case 'B':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid cursor down command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorDown, StreamAction{Tag: StreamActionKeyCursorDown, CursorDown: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI B with intermediates: %s",
				input.Intermediates,
			)
		}

	// CUF - Cursor Right
	case 'C':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid cursor right command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorRight, StreamAction{Tag: StreamActionKeyCursorRight, CursorRight: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI C with intermediates: %s",
				input.Intermediates,
			)
		}

	// CUB - Cursor Left
	case 'D', 'j':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid cursor left command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorLeft, StreamAction{Tag: StreamActionKeyCursorLeft, CursorLeft: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI D with intermediates: %s",
				input.Intermediates,
			)
		}

	// CNL - Cursor Next Line
	case 'E':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid cursor up command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorDown, StreamAction{Tag: StreamActionKeyCursorDown, CursorDown: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}
			if err := self.handler.Vt(StreamActionKeyCarriageReturn, StreamAction{Tag: StreamActionKeyCarriageReturn}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI E with intermediates: %s",
				input.Intermediates,
			)
		}

	// CPL - Cursor Previous Line
	case 'F':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid cursor down command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorUp, StreamAction{Tag: StreamActionKeyCursorUp, CursorUp: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}
			if err := self.handler.Vt(StreamActionKeyCarriageReturn, StreamAction{Tag: StreamActionKeyCarriageReturn}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI F with intermediates: %s",
				input.Intermediates,
			)
		}

	// HPA - Cursor Horizontal Position Absolute
	// TODO: test
	case 'G', '`':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid HPA command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorCol, StreamAction{Tag: StreamActionKeyCursorCol, CursorCol: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI G with intermediates: %s",
				input.Intermediates,
			)
		}

	// CUP - Set Cursor Position.
	// TODO: test
	case 'H', 'f':
		switch len(input.Intermediates) {
		case 0:
			var pos StreamActionCursorPos
			switch len(input.Params) {
			case 0:
				pos = StreamActionCursorPos{row: 1, col: 1}
			case 1:
				pos = StreamActionCursorPos{row: input.Params[0], col: 1}
			case 2:
				pos = StreamActionCursorPos{row: input.Params[0], col: input.Params[1]}
			default:
				warn("stream", "invalid CUP command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorPos, StreamAction{Tag: StreamActionKeyCursorPos, CursorPos: pos}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI H with intermediates: %s",
				input.Intermediates,
			)
		}

	// CHT - Cursor Horizontal Tabulation
	case 'I':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid horizontal tab command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyHorizontalTab, StreamAction{Tag: StreamActionKeyHorizontalTab, HorizontalTab: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI I with intermediates: %s",
				input.Intermediates,
			)
		}

	// Erase Display
	case 'J':
		var protected_ Optional[bool]
		switch len(input.Intermediates) {
		case 0:
			protected_ = Some(false)
		case 1:
			if input.Intermediates[0] == '?' {
				protected_ = Some(true)
			} else {
				protected_ = None[bool]()
			}
		default:
			protected_ = None[bool]()
		}

		if !protected_.Valid {
			warn("stream", "invalid erase display command: %s", input.Format())
			return nil
		}
		protected := protected_.Value

		var mode_ Optional[CSIEraseDisplay]
		switch len(input.Params) {
		case 0:
			mode_ = Some(CSIEraseDisplayBelow)
		case 1:
			// std.meta.intToEnum(csi.EraseDisplay, input.params[0]) catch null:
			// EraseDisplay is exhaustive, so only its five values convert.
			switch input.Params[0] {
			case uint16(CSIEraseDisplayBelow),
				uint16(CSIEraseDisplayAbove),
				uint16(CSIEraseDisplayComplete),
				uint16(CSIEraseDisplayScrollback),
				uint16(CSIEraseDisplayScrollComplete):
				mode_ = Some(CSIEraseDisplay(input.Params[0]))
			default:
				mode_ = None[CSIEraseDisplay]()
			}
		default:
			mode_ = None[CSIEraseDisplay]()
		}

		if !mode_.Valid {
			warn("stream", "invalid erase display command: %s", input.Format())
			return nil
		}
		mode := mode_.Value

		switch mode {
		case CSIEraseDisplayBelow:
			if err := self.handler.Vt(StreamActionKeyEraseDisplayBelow, StreamAction{Tag: StreamActionKeyEraseDisplayBelow, EraseDisplayBelow: protected}); err != nil {
				return err
			}
		case CSIEraseDisplayAbove:
			if err := self.handler.Vt(StreamActionKeyEraseDisplayAbove, StreamAction{Tag: StreamActionKeyEraseDisplayAbove, EraseDisplayAbove: protected}); err != nil {
				return err
			}
		case CSIEraseDisplayComplete:
			if err := self.handler.Vt(StreamActionKeyEraseDisplayComplete, StreamAction{Tag: StreamActionKeyEraseDisplayComplete, EraseDisplayComplete: protected}); err != nil {
				return err
			}
		case CSIEraseDisplayScrollback:
			if err := self.handler.Vt(StreamActionKeyEraseDisplayScrollback, StreamAction{Tag: StreamActionKeyEraseDisplayScrollback, EraseDisplayScrollback: protected}); err != nil {
				return err
			}
		case CSIEraseDisplayScrollComplete:
			if err := self.handler.Vt(StreamActionKeyEraseDisplayScrollComplete, StreamAction{Tag: StreamActionKeyEraseDisplayScrollComplete, EraseDisplayScrollComplete: protected}); err != nil {
				return err
			}
		default:
			panic("unreachable")
		}

	// Erase Line
	case 'K':
		var protected_ Optional[bool]
		switch len(input.Intermediates) {
		case 0:
			protected_ = Some(false)
		case 1:
			if input.Intermediates[0] == '?' {
				protected_ = Some(true)
			} else {
				protected_ = None[bool]()
			}
		default:
			protected_ = None[bool]()
		}

		if !protected_.Valid {
			warn("stream", "invalid erase line command: %s", input.Format())
			return nil
		}
		protected := protected_.Value

		var mode_ Optional[CSIEraseLine]
		switch len(input.Params) {
		case 0:
			mode_ = Some(CSIEraseLineRight)
		case 1:
			if input.Params[0] < 3 {
				mode_ = Some(CSIEraseLine(input.Params[0]))
			} else {
				mode_ = None[CSIEraseLine]()
			}
		default:
			mode_ = None[CSIEraseLine]()
		}

		if !mode_.Valid {
			warn("stream", "invalid erase line command: %s", input.Format())
			return nil
		}
		mode := mode_.Value

		switch mode {
		case CSIEraseLineRight:
			if err := self.handler.Vt(StreamActionKeyEraseLineRight, StreamAction{Tag: StreamActionKeyEraseLineRight, EraseLineRight: protected}); err != nil {
				return err
			}
		case CSIEraseLineLeft:
			if err := self.handler.Vt(StreamActionKeyEraseLineLeft, StreamAction{Tag: StreamActionKeyEraseLineLeft, EraseLineLeft: protected}); err != nil {
				return err
			}
		case CSIEraseLineComplete:
			if err := self.handler.Vt(StreamActionKeyEraseLineComplete, StreamAction{Tag: StreamActionKeyEraseLineComplete, EraseLineComplete: protected}); err != nil {
				return err
			}
		case CSIEraseLineRightUnlessPendingWrap:
			if err := self.handler.Vt(StreamActionKeyEraseLineRightUnlessPendingWrap, StreamAction{Tag: StreamActionKeyEraseLineRightUnlessPendingWrap, EraseLineRightUnlessPendingWrap: protected}); err != nil {
				return err
			}
		default: // _
			// Zig's {} of a value of a non-exhaustive enum that has no
			// name prints @enumFromInt(N); every named value has its own
			// prong above.
			warn("stream", "invalid erase line mode: @enumFromInt(%d)", mode)
		}

	// IL - Insert Lines
	// TODO: test
	case 'L':
		switch len(input.Intermediates) {
		case 0:
			var value uint
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = uint(input.Params[0])
			default:
				warn("stream", "invalid IL command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyInsertLines, StreamAction{Tag: StreamActionKeyInsertLines, InsertLines: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI L with intermediates: %s",
				input.Intermediates,
			)
		}

	// DL - Delete Lines
	// TODO: test
	case 'M':
		switch len(input.Intermediates) {
		case 0:
			var value uint
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = uint(input.Params[0])
			default:
				warn("stream", "invalid DL command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyDeleteLines, StreamAction{Tag: StreamActionKeyDeleteLines, DeleteLines: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI M with intermediates: %s",
				input.Intermediates,
			)
		}

	// Delete Character (DCH)
	case 'P':
		switch len(input.Intermediates) {
		case 0:
			var value uint
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = uint(input.Params[0])
			default:
				warn("stream", "invalid delete characters command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyDeleteChars, StreamAction{Tag: StreamActionKeyDeleteChars, DeleteChars: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI P with intermediates: %s",
				input.Intermediates,
			)
		}

	// Scroll Up (SD)

	case 'S':
		switch len(input.Intermediates) {
		case 0:
			var value uint
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = uint(input.Params[0])
			default:
				warn("stream", "invalid scroll up command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyScrollUp, StreamAction{Tag: StreamActionKeyScrollUp, ScrollUp: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI S with intermediates: %s",
				input.Intermediates,
			)
		}

	// Scroll Down (SD)
	case 'T':
		switch len(input.Intermediates) {
		case 0:
			var value uint
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = uint(input.Params[0])
			default:
				warn("stream", "invalid scroll down command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyScrollDown, StreamAction{Tag: StreamActionKeyScrollDown, ScrollDown: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI T with intermediates: %s",
				input.Intermediates,
			)
		}

	// Cursor Tabulation Control
	case 'W':
		switch len(input.Intermediates) {
		case 0:
			if len(input.Params) == 0 ||
				(len(input.Params) == 1 && input.Params[0] == 0) {
				if err := self.handler.Vt(StreamActionKeyTabSet, StreamAction{Tag: StreamActionKeyTabSet}); err != nil {
					return err
				}
				return nil
			}

			switch len(input.Params) {
			case 0:
				panic("unreachable")

			case 1:
				switch input.Params[0] {
				case 0:
					panic("unreachable")

				case 2:
					if err := self.handler.Vt(StreamActionKeyTabClearCurrent, StreamAction{Tag: StreamActionKeyTabClearCurrent}); err != nil {
						return err
					}

				case 5:
					if err := self.handler.Vt(StreamActionKeyTabClearAll, StreamAction{Tag: StreamActionKeyTabClearAll}); err != nil {
						return err
					}

				default:
				}

			default:
			}

			warn("stream", "invalid cursor tabulation control: %s", input.Format())
			return nil

		case 1:
			if input.Intermediates[0] == '?' &&
				len(input.Params) == 1 &&
				input.Params[0] == 5 {
				if err := self.handler.Vt(StreamActionKeyTabReset, StreamAction{Tag: StreamActionKeyTabReset}); err != nil {
					return err
				}
			} else {
				warn("stream", "invalid cursor tabulation control: %s", input.Format())
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI W with intermediates: %s",
				input.Intermediates,
			)
		}

	// Erase Characters (ECH)
	case 'X':
		switch len(input.Intermediates) {
		case 0:
			var value uint
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = uint(input.Params[0])
			default:
				warn("stream", "invalid erase characters command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyEraseChars, StreamAction{Tag: StreamActionKeyEraseChars, EraseChars: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI X with intermediates: %s",
				input.Intermediates,
			)
		}

	// CHT - Cursor Horizontal Tabulation Back
	case 'Z':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid horizontal tab back command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyHorizontalTabBack, StreamAction{Tag: StreamActionKeyHorizontalTabBack, HorizontalTabBack: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI Z with intermediates: %s",
				input.Intermediates,
			)
		}

	// HPR - Cursor Horizontal Position Relative
	case 'a':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid HPR command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorColRelative, StreamAction{Tag: StreamActionKeyCursorColRelative, CursorColRelative: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI a with intermediates: %s",
				input.Intermediates,
			)
		}

	// Repeat Previous Char (REP)
	case 'b':
		switch len(input.Intermediates) {
		case 0:
			var value uint
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = uint(input.Params[0])
			default:
				warn("stream", "invalid print repeat command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyPrintRepeat, StreamAction{Tag: StreamActionKeyPrintRepeat, PrintRepeat: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI b with intermediates: %s",
				input.Intermediates,
			)
		}

	// c - Device Attributes (DA1)
	case 'c':
		var req Optional[ANSIDeviceAttributeReq]
		switch len(input.Intermediates) {
		case 0:
			req = Some(ANSIDeviceAttributeReqPrimary)
		case 1:
			switch input.Intermediates[0] {
			case '>':
				req = Some(ANSIDeviceAttributeReqSecondary)
			case '=':
				req = Some(ANSIDeviceAttributeReqTertiary)
			default:
				req = None[ANSIDeviceAttributeReq]()
			}
		default:
			req = None[ANSIDeviceAttributeReq]()
		}

		if req.Valid {
			r := req.Value
			if err := self.handler.Vt(StreamActionKeyDeviceAttributes, StreamAction{Tag: StreamActionKeyDeviceAttributes, DeviceAttributes: r}); err != nil {
				return err
			}
		} else {
			warn("stream", "invalid device attributes command: %s", input.Format())
			return nil
		}

	// VPA - Cursor Vertical Position Absolute
	case 'd':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid VPA command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorRow, StreamAction{Tag: StreamActionKeyCursorRow, CursorRow: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI d with intermediates: %s",
				input.Intermediates,
			)
		}

	// VPR - Cursor Vertical Position Relative
	case 'e':
		switch len(input.Intermediates) {
		case 0:
			var value uint16
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = input.Params[0]
			default:
				warn("stream", "invalid VPR command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyCursorRowRelative, StreamAction{Tag: StreamActionKeyCursorRowRelative, CursorRowRelative: StreamActionCursorMovement{value: value}}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI e with intermediates: %s",
				input.Intermediates,
			)
		}

	// TBC - Tab Clear
	// TODO: test
	case 'g':
		switch len(input.Intermediates) {
		case 0:
			var mode CSITabClear
			switch len(input.Params) {
			case 1:
				// std.meta.intToEnum(csi.TabClear, input.params[0]):
				// TabClear is a non-exhaustive enum(u8), so every value
				// that fits in a u8 converts.
				if input.Params[0] > 0xFF {
					warn("stream", "invalid tab clear mode: %d", input.Params[0])
					return nil
				}
				mode = CSITabClear(input.Params[0])
			default:
				warn("stream", "invalid tab clear command: %s", input.Format())
				return nil
			}
			switch mode {
			case CSITabClearCurrent:
				if err := self.handler.Vt(StreamActionKeyTabClearCurrent, StreamAction{Tag: StreamActionKeyTabClearCurrent}); err != nil {
					return err
				}
			case CSITabClearAll:
				if err := self.handler.Vt(StreamActionKeyTabClearAll, StreamAction{Tag: StreamActionKeyTabClearAll}); err != nil {
					return err
				}
			default: // _
				// Zig's {} of an unnamed value of a non-exhaustive enum.
				warn("stream", "unknown tab clear mode: @enumFromInt(%d)", mode)
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI g with intermediates: %s",
				input.Intermediates,
			)
		}

	// SM - Set Mode
	case 'h':
		// mode: (break :mode is break prong)
		// ansi:
		var ansi_mode bool
		if len(input.Intermediates) == 0 {
			ansi_mode = true
		} else if len(input.Intermediates) == 1 &&
			input.Intermediates[0] == '?' {
			ansi_mode = false
		} else {
			warn("stream", "invalid set mode command: %s", input.Format())
			break prong // break :mode
		}

		for _, mode_int := range input.Params {
			if mode := ModesModeFromInt(mode_int, ansi_mode); mode.Valid {
				if err := self.handler.Vt(StreamActionKeySetMode, StreamAction{Tag: StreamActionKeySetMode, SetMode: StreamActionMode{mode: mode.Value}}); err != nil {
					return err
				}
			} else {
				warn("stream", "unimplemented mode: %d", mode_int)
			}
		}

	// RM - Reset Mode
	case 'l':
		// mode: (break :mode is break prong)
		// ansi:
		var ansi_mode bool
		if len(input.Intermediates) == 0 {
			ansi_mode = true
		} else if len(input.Intermediates) == 1 &&
			input.Intermediates[0] == '?' {
			ansi_mode = false
		} else {
			warn("stream", "invalid set mode command: %s", input.Format())
			break prong // break :mode
		}

		for _, mode_int := range input.Params {
			if mode := ModesModeFromInt(mode_int, ansi_mode); mode.Valid {
				if err := self.handler.Vt(StreamActionKeyResetMode, StreamAction{Tag: StreamActionKeyResetMode, ResetMode: StreamActionMode{mode: mode.Value}}); err != nil {
					return err
				}
			} else {
				warn("stream", "unimplemented mode: %d", mode_int)
			}
		}

	// SGR - Select Graphic Rendition
	case 'm':
		switch len(input.Intermediates) {
		case 0:
			// This is the most common case.
			// log.info("parse SGR params={any}", .{input.params});
			var p = SGRParser{
				params:     input.Params,
				params_sep: input.ParamsSep,
			}
			for {
				attr_ := p.Next()
				if !attr_.Valid {
					break
				}
				attr := attr_.Value
				// log.info("SGR attribute: {}", .{attr});
				if err := self.handler.Vt(StreamActionKeySetAttribute, StreamAction{Tag: StreamActionKeySetAttribute, SetAttribute: attr}); err != nil {
					return err
				}
			}

		case 1:
			switch input.Intermediates[0] {
			case '>':
				// blk: (break :blk is break prong)
				if len(input.Params) == 0 {
					// Reset
					if err := self.handler.Vt(StreamActionKeyModifyKeyFormat, StreamAction{Tag: StreamActionKeyModifyKeyFormat, ModifyKeyFormat: ANSIModifyKeyFormatLegacy}); err != nil {
						return err
					}
					break prong // break :blk
				}

				var format ANSIModifyKeyFormat
				switch input.Params[0] {
				case 0:
					format = ANSIModifyKeyFormatLegacy
				case 1:
					format = ANSIModifyKeyFormatCursorKeys
				case 2:
					format = ANSIModifyKeyFormatFunctionKeys
				case 4:
					format = ANSIModifyKeyFormatOtherKeysNone
				default:
					warn("stream", "invalid setModifyKeyFormat: %s", input.Format())
					break prong // break :blk
				}

				if len(input.Params) > 2 {
					warn("stream", "invalid setModifyKeyFormat: %s", input.Format())
					break prong // break :blk
				}

				if len(input.Params) == 2 {
					switch format {
					// We don't support any of the subparams yet for these.
					case ANSIModifyKeyFormatLegacy:
					case ANSIModifyKeyFormatCursorKeys:
					case ANSIModifyKeyFormatFunctionKeys:

					// We only support the numeric form.
					case ANSIModifyKeyFormatOtherKeysNone:
						switch input.Params[1] {
						case 2:
							format = ANSIModifyKeyFormatOtherKeysNumeric
						default:
						}
					case ANSIModifyKeyFormatOtherKeysNumericExcept:
					case ANSIModifyKeyFormatOtherKeysNumeric:
					default:
						panic("unreachable")
					}
				}

				if err := self.handler.Vt(StreamActionKeyModifyKeyFormat, StreamAction{Tag: StreamActionKeyModifyKeyFormat, ModifyKeyFormat: format}); err != nil {
					return err
				}

			default:
				warn("stream",
					"unknown CSI m with intermediate: %d",
					input.Intermediates[0],
				)
			}

		default:
			// Nothing, but I wanted a place to put this comment:
			// there are others forms of CSI m that have intermediates.
			// `vim --clean` uses `CSI ? 4 m` and I don't know what
			// that means.
			warn("stream",
				"ignoring unimplemented CSI m with intermediates: %s",
				input.Intermediates,
			)
		}

	// TODO: test
	case 'n':
		// Handle deviceStatusReport first
		if len(input.Intermediates) == 0 ||
			input.Intermediates[0] == '?' {
			if len(input.Params) != 1 {
				warn("stream", "invalid device status report command: %s", input.Format())
				return nil
			}

			// question:
			var question bool
			if len(input.Intermediates) == 0 {
				question = false
			} else if len(input.Intermediates) == 1 &&
				input.Intermediates[0] == '?' {
				question = true
			} else {
				warn("stream", "invalid set mode command: %s", input.Format())
				return nil
			}

			req_ := DeviceStatusReqFromInt(input.Params[0], question)
			if !req_.Valid {
				warn("stream", "invalid device status report command: %s", input.Format())
				return nil
			}
			req := req_.Value

			if err := self.handler.Vt(StreamActionKeyDeviceStatus, StreamAction{Tag: StreamActionKeyDeviceStatus, DeviceStatus: StreamActionDeviceStatus{request: req}}); err != nil {
				return err
			}
			return nil
		}

		// Handle other forms of CSI n
		switch len(input.Intermediates) {
		case 0:
			panic("unreachable") // handled above

		case 1:
			switch input.Intermediates[0] {
			case '>':
				// This isn't strictly correct. CSI > n has parameters that
				// control what exactly is being disabled. However, we
				// only support reverting back to modify other keys in
				// numeric except format.
				if err := self.handler.Vt(StreamActionKeyModifyKeyFormat, StreamAction{Tag: StreamActionKeyModifyKeyFormat, ModifyKeyFormat: ANSIModifyKeyFormatOtherKeysNumericExcept}); err != nil {
					return err
				}

			default:
				warn("stream",
					"unknown CSI n with intermediate: %d",
					input.Intermediates[0],
				)
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI n with intermediates: %s",
				input.Intermediates,
			)
		}

	// DECRQM - Request Mode
	case 'p':
		switch len(input.Intermediates) {
		case 2:
			// decrqm: (break :decrqm is break prong)
			var ansi_mode bool
			{
				// ansi: (break :ansi is goto ansi)
				switch len(input.Intermediates) {
				case 1:
					if input.Intermediates[0] == '$' {
						ansi_mode = true
						goto ansi
					}
				case 2:
					if input.Intermediates[0] == '?' &&
						input.Intermediates[1] == '$' {
						ansi_mode = false
						goto ansi
					}
				default:
				}

				warn("stream",
					"ignoring unimplemented CSI p with intermediates: %s",
					input.Intermediates,
				)
				break prong // break :decrqm
			}
		ansi:

			if len(input.Params) != 1 {
				warn("stream", "invalid DECRQM command: %s", input.Format())
				break prong // break :decrqm
			}

			mode_raw := input.Params[0]
			mode := ModesModeFromInt(mode_raw, ansi_mode)
			if mode.Valid {
				m := mode.Value
				if err := self.handler.Vt(StreamActionKeyRequestMode, StreamAction{Tag: StreamActionKeyRequestMode, RequestMode: StreamActionMode{mode: m}}); err != nil {
					return err
				}
			} else {
				if err := self.handler.Vt(StreamActionKeyRequestModeUnknown, StreamAction{Tag: StreamActionKeyRequestModeUnknown, RequestModeUnknown: StreamActionRawMode{
					mode: mode_raw,
					ansi: ansi_mode,
				}}); err != nil {
					return err
				}
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI p with intermediates: %s",
				input.Intermediates,
			)
		}

	case 'q':
		switch len(input.Intermediates) {
		case 1:
			switch input.Intermediates[0] {
			// DECSCUSR - Select Cursor Style
			// TODO: test
			case ' ':
				var style ANSICursorStyle
				switch len(input.Params) {
				case 0:
					style = ANSICursorStyleDefault
				case 1:
					switch input.Params[0] {
					case 0:
						style = ANSICursorStyleDefault
					case 1:
						style = ANSICursorStyleBlinkingBlock
					case 2:
						style = ANSICursorStyleSteadyBlock
					case 3:
						style = ANSICursorStyleBlinkingUnderline
					case 4:
						style = ANSICursorStyleSteadyUnderline
					case 5:
						style = ANSICursorStyleBlinkingBar
					case 6:
						style = ANSICursorStyleSteadyBar
					default:
						warn("stream", "invalid cursor style value: %d", input.Params[0])
						return nil
					}
				default:
					warn("stream", "invalid set cursor style command: %s", input.Format())
					return nil
				}
				if err := self.handler.Vt(StreamActionKeyCursorStyle, StreamAction{Tag: StreamActionKeyCursorStyle, CursorStyle: style}); err != nil {
					return err
				}

			// DECSCA
			case '"':
				var mode_ Optional[ANSIProtectedMode]
				switch len(input.Params) {
				default:
					mode_ = None[ANSIProtectedMode]()
				case 0:
					mode_ = Some(ANSIProtectedModeOff)
				case 1:
					switch input.Params[0] {
					case 0, 2:
						mode_ = Some(ANSIProtectedModeOff)
					case 1:
						mode_ = Some(ANSIProtectedModeDec)
					default:
						mode_ = None[ANSIProtectedMode]()
					}
				}

				if !mode_.Valid {
					warn("stream", "invalid set protected mode command: %s", input.Format())
					return nil
				}
				mode := mode_.Value

				switch mode {
				case ANSIProtectedModeOff:
					if err := self.handler.Vt(StreamActionKeyProtectedModeOff, StreamAction{Tag: StreamActionKeyProtectedModeOff}); err != nil {
						return err
					}
				case ANSIProtectedModeIso:
					if err := self.handler.Vt(StreamActionKeyProtectedModeIso, StreamAction{Tag: StreamActionKeyProtectedModeIso}); err != nil {
						return err
					}
				case ANSIProtectedModeDec:
					if err := self.handler.Vt(StreamActionKeyProtectedModeDec, StreamAction{Tag: StreamActionKeyProtectedModeDec}); err != nil {
						return err
					}
				default:
					panic("unreachable")
				}

			// XTVERSION
			case '>':
				if err := self.handler.Vt(StreamActionKeyXtversion, StreamAction{Tag: StreamActionKeyXtversion}); err != nil {
					return err
				}
			default:
				warn("stream",
					"ignoring unimplemented CSI q with intermediates: %s",
					input.Intermediates,
				)
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI p with intermediates: %s",
				input.Intermediates,
			)
		}

	case 'r':
		switch len(input.Intermediates) {
		// DECSTBM - Set Top and Bottom Margins
		case 0:
			switch len(input.Params) {
			case 0:
				if err := self.handler.Vt(StreamActionKeyTopAndBottomMargin, StreamAction{Tag: StreamActionKeyTopAndBottomMargin, TopAndBottomMargin: StreamActionMargin{top_left: 0, bottom_right: 0}}); err != nil {
					return err
				}
			case 1:
				if err := self.handler.Vt(StreamActionKeyTopAndBottomMargin, StreamAction{Tag: StreamActionKeyTopAndBottomMargin, TopAndBottomMargin: StreamActionMargin{top_left: input.Params[0], bottom_right: 0}}); err != nil {
					return err
				}
			case 2:
				if err := self.handler.Vt(StreamActionKeyTopAndBottomMargin, StreamAction{Tag: StreamActionKeyTopAndBottomMargin, TopAndBottomMargin: StreamActionMargin{top_left: input.Params[0], bottom_right: input.Params[1]}}); err != nil {
					return err
				}
			default:
				warn("stream", "invalid DECSTBM command: %s", input.Format())
			}

		case 1:
			switch input.Intermediates[0] {
			// Restore Mode
			case '?':
				for _, mode_int := range input.Params {
					if mode := ModesModeFromInt(mode_int, false); mode.Valid {
						if err := self.handler.Vt(StreamActionKeyRestoreMode, StreamAction{Tag: StreamActionKeyRestoreMode, RestoreMode: StreamActionMode{mode: mode.Value}}); err != nil {
							return err
						}
					} else {
						warn("stream",
							"unimplemented restore mode: %d",
							mode_int,
						)
					}
				}

			default:
				warn("stream",
					"unknown CSI s with intermediate: %s",
					input.Format(),
				)
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI s with intermediates: %s",
				input.Format(),
			)
		}

	case 's':
		switch len(input.Intermediates) {
		// DECSLRM
		case 0:
			switch len(input.Params) {
			// CSI S is ambiguous with zero params so we defer
			// to our handler to do the proper logic. If mode 69
			// is set, then we should invoke DECSLRM, otherwise
			// we should invoke SC.
			case 0:
				if err := self.handler.Vt(StreamActionKeyLeftAndRightMarginAmbiguous, StreamAction{Tag: StreamActionKeyLeftAndRightMarginAmbiguous}); err != nil {
					return err
				}
			case 1:
				if err := self.handler.Vt(StreamActionKeyLeftAndRightMargin, StreamAction{Tag: StreamActionKeyLeftAndRightMargin, LeftAndRightMargin: StreamActionMargin{top_left: input.Params[0], bottom_right: 0}}); err != nil {
					return err
				}
			case 2:
				if err := self.handler.Vt(StreamActionKeyLeftAndRightMargin, StreamAction{Tag: StreamActionKeyLeftAndRightMargin, LeftAndRightMargin: StreamActionMargin{top_left: input.Params[0], bottom_right: input.Params[1]}}); err != nil {
					return err
				}
			default:
				warn("stream", "invalid DECSLRM command: %s", input.Format())
			}

		case 1:
			switch input.Intermediates[0] {
			case '?':
				for _, mode_int := range input.Params {
					if mode := ModesModeFromInt(mode_int, false); mode.Valid {
						if err := self.handler.Vt(StreamActionKeySaveMode, StreamAction{Tag: StreamActionKeySaveMode, SaveMode: StreamActionMode{mode: mode.Value}}); err != nil {
							return err
						}
					} else {
						warn("stream",
							"unimplemented save mode: %d",
							mode_int,
						)
					}
				}

			// XTSHIFTESCAPE
			case '>':
				// capture: (break :capture is break prong)
				var capture bool
				switch len(input.Params) {
				case 0:
					capture = false
				case 1:
					switch input.Params[0] {
					case 0:
						capture = false
					case 1:
						capture = true
					default:
						warn("stream", "invalid XTSHIFTESCAPE command: %s", input.Format())
						break prong // break :capture
					}
				default:
					warn("stream", "invalid XTSHIFTESCAPE command: %s", input.Format())
					break prong // break :capture
				}

				if err := self.handler.Vt(StreamActionKeyMouseShiftCapture, StreamAction{Tag: StreamActionKeyMouseShiftCapture, MouseShiftCapture: capture}); err != nil {
					return err
				}

			default:
				warn("stream",
					"unknown CSI s with intermediate: %s",
					input.Format(),
				)
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI s with intermediates: %s",
				input.Format(),
			)
		}

	// XTWINOPS
	case 't':
		switch len(input.Intermediates) {
		case 0:
			if len(input.Params) > 0 {
				switch input.Params[0] {
				case 14:
					if len(input.Params) == 1 {
						// report the text area size in pixels
						if err := self.handler.Vt(StreamActionKeySizeReport, StreamAction{Tag: StreamActionKeySizeReport, SizeReport: CSISizeReportStyleCsi14T}); err != nil {
							return err
						}
					} else {
						warn("stream",
							"ignoring CSI 14 t with extra parameters: %s",
							input.Format(),
						)
					}
				case 16:
					if len(input.Params) == 1 {
						// report cell size in pixels
						if err := self.handler.Vt(StreamActionKeySizeReport, StreamAction{Tag: StreamActionKeySizeReport, SizeReport: CSISizeReportStyleCsi16T}); err != nil {
							return err
						}
					} else {
						warn("stream",
							"ignoring CSI 16 t with extra parameters: %s",
							input.Format(),
						)
					}
				case 18:
					if len(input.Params) == 1 {
						// report screen size in characters
						if err := self.handler.Vt(StreamActionKeySizeReport, StreamAction{Tag: StreamActionKeySizeReport, SizeReport: CSISizeReportStyleCsi18T}); err != nil {
							return err
						}
					} else {
						warn("stream",
							"ignoring CSI 18 t with extra parameters: %s",
							input.Format(),
						)
					}
				case 21:
					if len(input.Params) == 1 {
						// report window title
						if err := self.handler.Vt(StreamActionKeySizeReport, StreamAction{Tag: StreamActionKeySizeReport, SizeReport: CSISizeReportStyleCsi21T}); err != nil {
							return err
						}
					} else {
						warn("stream",
							"ignoring CSI 21 t with extra parameters: %s",
							input.Format(),
						)
					}
				case 22, 23:
					// Zig: inline 22, 23 => |number|
					number := input.Params[0]
					if (len(input.Params) == 2 ||
						len(input.Params) == 3) &&
						// we only support window title
						(input.Params[1] == 0 ||
							input.Params[1] == 2) {
						// push/pop title
						var index uint16
						if len(input.Params) == 3 {
							index = input.Params[2]
						} else {
							index = 0
						}
						switch number {
						case 22:
							if err := self.handler.Vt(StreamActionKeyTitlePush, StreamAction{Tag: StreamActionKeyTitlePush, TitlePush: index}); err != nil {
								return err
							}
						case 23:
							if err := self.handler.Vt(StreamActionKeyTitlePop, StreamAction{Tag: StreamActionKeyTitlePop, TitlePop: index}); err != nil {
								return err
							}
						default:
							// Zig: @compileError("unreachable")
							panic("unreachable")
						}
					} else {
						warn("stream",
							"ignoring CSI 22/23 t with extra parameters: %s",
							input.Format(),
						)
					}
				default:
					warn("stream",
						"ignoring CSI t with unimplemented parameter: %s",
						input.Format(),
					)
				}
			} else {
				// Zig: log.err
				warn("stream",
					"ignoring CSI t with no parameters: %s",
					input.Format(),
				)
			}
		default:
			warn("stream",
				"ignoring unimplemented CSI t with intermediates: %s",
				input.Format(),
			)
		}

	case 'u':
		switch len(input.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyRestoreCursor, StreamAction{Tag: StreamActionKeyRestoreCursor}); err != nil {
				return err
			}

		// Kitty keyboard protocol
		case 1:
			switch input.Intermediates[0] {
			case '?':
				if err := self.handler.Vt(StreamActionKeyKittyKeyboardQuery, StreamAction{Tag: StreamActionKeyKittyKeyboardQuery}); err != nil {
					return err
				}

			case '>':
				// push: (break :push is break prong)
				// Zig: u5.
				var flags uint8
				if len(input.Params) == 1 {
					// std.math.cast(u5, input.params[0]) orelse
					if input.Params[0] > 0x1F {
						warn("stream", "invalid pushKittyKeyboard command: %s", input.Format())
						break prong // break :push
					}
					flags = uint8(input.Params[0])
				} else {
					flags = 0
				}

				if err := self.handler.Vt(StreamActionKeyKittyKeyboardPush, StreamAction{Tag: StreamActionKeyKittyKeyboardPush, KittyKeyboardPush: StreamActionKittyKeyboardFlags{flags: KittyKeyFlagsFromInt(flags)}}); err != nil {
					return err
				}

			case '<':
				var number uint16
				if len(input.Params) == 1 {
					number = input.Params[0]
				} else {
					number = 1
				}

				if err := self.handler.Vt(StreamActionKeyKittyKeyboardPop, StreamAction{Tag: StreamActionKeyKittyKeyboardPop, KittyKeyboardPop: number}); err != nil {
					return err
				}

			case '=':
				// set: (break :set is break prong)
				// Zig: u5.
				var flags uint8
				if len(input.Params) >= 1 {
					// std.math.cast(u5, input.params[0]) orelse
					if input.Params[0] > 0x1F {
						warn("stream", "invalid setKittyKeyboard command: %s", input.Format())
						break prong // break :set
					}
					flags = uint8(input.Params[0])
				} else {
					flags = 0
				}

				var number uint16
				if len(input.Params) >= 2 {
					number = input.Params[1]
				} else {
					number = 1
				}

				var action_tag StreamActionTag
				switch number {
				case 1:
					action_tag = StreamActionKeyKittyKeyboardSet
				case 2:
					action_tag = StreamActionKeyKittyKeyboardSetOr
				case 3:
					action_tag = StreamActionKeyKittyKeyboardSetNot
				default:
					warn("stream", "invalid setKittyKeyboard command: %s", input.Format())
					break prong // break :set
				}

				kitty_flags := StreamActionKittyKeyboardFlags{flags: KittyKeyFlagsFromInt(flags)}
				switch action_tag {
				case StreamActionKeyKittyKeyboardSet:
					if err := self.handler.Vt(StreamActionKeyKittyKeyboardSet, StreamAction{Tag: StreamActionKeyKittyKeyboardSet, KittyKeyboardSet: kitty_flags}); err != nil {
						return err
					}
				case StreamActionKeyKittyKeyboardSetOr:
					if err := self.handler.Vt(StreamActionKeyKittyKeyboardSetOr, StreamAction{Tag: StreamActionKeyKittyKeyboardSetOr, KittyKeyboardSetOr: kitty_flags}); err != nil {
						return err
					}
				case StreamActionKeyKittyKeyboardSetNot:
					if err := self.handler.Vt(StreamActionKeyKittyKeyboardSetNot, StreamAction{Tag: StreamActionKeyKittyKeyboardSetNot, KittyKeyboardSetNot: kitty_flags}); err != nil {
						return err
					}
				default:
					panic("unreachable")
				}

			default:
				warn("stream",
					"unknown CSI s with intermediate: %s",
					input.Format(),
				)
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI u: %s",
				input.Format(),
			)
		}

	// ICH - Insert Blanks
	case '@':
		switch len(input.Intermediates) {
		case 0:
			var value uint
			switch len(input.Params) {
			case 0:
				value = 1
			case 1:
				value = uint(max(1, input.Params[0]))
			default:
				warn("stream", "invalid ICH command: %s", input.Format())
				return nil
			}
			if err := self.handler.Vt(StreamActionKeyInsertBlanks, StreamAction{Tag: StreamActionKeyInsertBlanks, InsertBlanks: value}); err != nil {
				return err
			}

		default:
			warn("stream",
				"ignoring unimplemented CSI @: %s",
				input.Format(),
			)
		}

	// DECSASD - Select Active Status Display
	case '}':
		// decsasd: (break :decsasd is break prong)
		// Verify we're getting a DECSASD command
		if len(input.Intermediates) != 1 || input.Intermediates[0] != '$' {
			warn("stream", "unimplemented CSI callback: %s", input.Format())
			break prong // break :decsasd
		}
		if len(input.Params) != 1 {
			warn("stream", "unimplemented CSI callback: %s", input.Format())
			break prong // break :decsasd
		}

		var display ANSIStatusDisplay
		switch input.Params[0] {
		case 0:
			display = ANSIStatusDisplayMain
		case 1:
			display = ANSIStatusDisplayStatusLine
		default:
			warn("stream", "unimplemented CSI callback: %s", input.Format())
			break prong // break :decsasd
		}

		if err := self.handler.Vt(StreamActionKeyActiveStatusDisplay, StreamAction{Tag: StreamActionKeyActiveStatusDisplay, ActiveStatusDisplay: display}); err != nil {
			return err
		}

	default:
		warn("stream", "unimplemented CSI action: %s", input.Format())
	}
	return nil
}

// oscDispatch is Stream(Handler).oscDispatch.
func (self *Stream[H]) oscDispatch(cmd OSCCommand) error {
	// The branch hints here are based on real world data
	// which indicates that the most common OSC commands are:
	//
	// 1. hyperlink_end
	// 2. change_window_title
	// 3. change_window_icon
	// 4. hyperlink_start
	// 5. report_pwd
	// 6. color_operation
	// 7. semantic_prompt
	//
	// Together, these 7 commands make up about 96% of all
	// OSC commands encountered in real world scenarios.
	//
	// Additionally, within the prongs, unlikely branch
	// hints have been added to branches that deal with
	// invalid sequences/commands, this is in order to
	// optimize for the happy path where we're getting
	// valid data from the program we're running.
	//
	// ref: https://github.com/qwerasd205/asciinema-stats

	switch cmd.Tag {
	case OSCCommandKeySemanticPrompt:
		sp := cmd.SemanticPrompt
		if err := self.handler.Vt(StreamActionKeySemanticPrompt, StreamAction{Tag: StreamActionKeySemanticPrompt, SemanticPrompt: sp}); err != nil {
			return err
		}

	case OSCCommandKeyChangeWindowTitle:
		title := cmd.ChangeWindowTitle
		// std.unicode.utf8ValidateSlice is utf8.Valid (mapping rule 5;
		// the same equivalence as isSafeUtf8 in osc_encoding.go).
		if !utf8.Valid(title) {
			warn("stream", "change title request: invalid utf-8, ignoring request")
			return nil
		}

		if err := self.handler.Vt(StreamActionKeyWindowTitle, StreamAction{Tag: StreamActionKeyWindowTitle, WindowTitle: StreamActionWindowTitle{title: title}}); err != nil {
			return err
		}

	case OSCCommandKeyChangeWindowIcon:
		// Zig: log.info("OSC 1 (change icon) received and ignored icon={s}", .{icon});

	case OSCCommandKeyClipboardContents:
		clip := cmd.ClipboardContents
		if err := self.handler.Vt(StreamActionKeyClipboardContents, StreamAction{Tag: StreamActionKeyClipboardContents, ClipboardContents: StreamActionClipboardContents{
			kind: clip.kind,
			data: clip.data,
		}}); err != nil {
			return err
		}

	case OSCCommandKeyReportPwd:
		v := cmd.ReportPwd
		if err := self.handler.Vt(StreamActionKeyReportPwd, StreamAction{Tag: StreamActionKeyReportPwd, ReportPwd: StreamActionReportPwd{url: v.value}}); err != nil {
			return err
		}

	case OSCCommandKeyMouseShape:
		v := cmd.MouseShape
		shape_ := MouseShapeFromString(v.value)
		if !shape_.Valid {
			warn("stream", "unknown cursor shape: %s", v.value)
			return nil
		}
		shape := shape_.Value

		if err := self.handler.Vt(StreamActionKeyMouseShape, StreamAction{Tag: StreamActionKeyMouseShape, MouseShape: shape}); err != nil {
			return err
		}

	case OSCCommandKeyColorOperation:
		v := cmd.ColorOperation
		if err := self.handler.Vt(StreamActionKeyColorOperation, StreamAction{Tag: StreamActionKeyColorOperation, ColorOperation: StreamActionColorOperation{
			op:         v.op,
			requests:   v.requests,
			terminator: v.terminator,
		}}); err != nil {
			return err
		}

	case OSCCommandKeyKittyColorProtocol:
		v := cmd.KittyColorProtocol
		if err := self.handler.Vt(StreamActionKeyKittyColorReport, StreamAction{Tag: StreamActionKeyKittyColorReport, KittyColorReport: v}); err != nil {
			return err
		}

	case OSCCommandKeyShowDesktopNotification:
		v := cmd.ShowDesktopNotification
		if err := self.handler.Vt(StreamActionKeyShowDesktopNotification, StreamAction{Tag: StreamActionKeyShowDesktopNotification, ShowDesktopNotification: StreamActionShowDesktopNotification{
			title: v.title,
			body:  v.body,
		}}); err != nil {
			return err
		}

	case OSCCommandKeyHyperlinkStart:
		v := cmd.HyperlinkStart
		if err := self.handler.Vt(StreamActionKeyStartHyperlink, StreamAction{Tag: StreamActionKeyStartHyperlink, StartHyperlink: StreamActionStartHyperlink{
			uri: v.uri,
			id:  v.id,
		}}); err != nil {
			return err
		}

	case OSCCommandKeyHyperlinkEnd:
		if err := self.handler.Vt(StreamActionKeyEndHyperlink, StreamAction{Tag: StreamActionKeyEndHyperlink}); err != nil {
			return err
		}

	case OSCCommandKeyConemuProgressReport:
		v := cmd.ConemuProgressReport
		if err := self.handler.Vt(StreamActionKeyProgressReport, StreamAction{Tag: StreamActionKeyProgressReport, ProgressReport: v}); err != nil {
			return err
		}

	case OSCCommandKeyConemuSleep,
		OSCCommandKeyConemuShowMessageBox,
		OSCCommandKeyConemuChangeTabTitle,
		OSCCommandKeyConemuWaitInput,
		OSCCommandKeyConemuGuimacro,
		OSCCommandKeyConemuComment,
		OSCCommandKeyConemuXtermEmulation,
		OSCCommandKeyConemuOutputEnvironmentVariable,
		OSCCommandKeyConemuRunProcess,
		OSCCommandKeyKittyTextSizing,
		OSCCommandKeyKittyClipboardProtocol,
		OSCCommandKeyContextSignal:
		// Zig: log.debug("unimplemented OSC callback: {}", .{cmd});

	case OSCCommandKeyInvalid:
		// This is an invalid internal state, not an invalid OSC
		// string being parsed. We shouldn't see this.
		warn("stream", "invalid OSC, should never happen")

	default:
		panic("unreachable")
	}
	return nil
}

// configureCharset is Stream(Handler).configureCharset.
func (self *Stream[H]) configureCharset(intermediates []uint8, set CharsetsCharset) error {
	if len(intermediates) != 1 {
		warn("stream", "invalid charset intermediate: %s", zigFmtAnySlice(intermediates))
		return nil
	}

	var slot CharsetsSlots
	switch intermediates[0] {
	// TODO: support slots '-', '.', '/'

	case '(':
		slot = CharsetsSlotsG0
	case ')':
		slot = CharsetsSlotsG1
	case '*':
		slot = CharsetsSlotsG2
	case '+':
		slot = CharsetsSlotsG3
	default:
		warn("stream", "invalid charset intermediate: %s", zigFmtAnySlice(intermediates))
		return nil
	}

	return self.handler.Vt(StreamActionKeyConfigureCharset, StreamAction{Tag: StreamActionKeyConfigureCharset, ConfigureCharset: StreamActionConfigureCharset{
		slot:    slot,
		charset: set,
	}})
}

// escDispatch is Stream(Handler).escDispatch.
func (self *Stream[H]) escDispatch(action ParserActionESC) error {
	// The branch hints here are based on real world data
	// which indicates that the most common ESC finals are:
	//
	// 1. B
	// 2. \
	// 3. 0
	// 4. M
	// 5. 8
	// 6. 7
	// 7. >
	// 8. =
	//
	// Together, these 8 finals make up nearly 99% of all
	// ESC sequences encountered in real world scenarios.
	//
	// Additionally, within the prongs, unlikely branch
	// hints have been added to branches that deal with
	// invalid sequences/commands, this is in order to
	// optimize for the happy path where we're getting
	// valid data from the program we're running.
	//
	// ref: https://github.com/qwerasd205/asciinema-stats

prong:
	switch action.Final {
	// Charsets
	case 'B':
		if err := self.configureCharset(action.Intermediates, CharsetsCharsetAscii); err != nil {
			return err
		}
	case 'A':
		if err := self.configureCharset(action.Intermediates, CharsetsCharsetBritish); err != nil {
			return err
		}
	case '0':
		if err := self.configureCharset(action.Intermediates, CharsetsCharsetDecSpecial); err != nil {
			return err
		}

	// DECSC - Save Cursor
	case '7':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeySaveCursor, StreamAction{Tag: StreamActionKeySaveCursor}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid command: %s", action.Format())
			return nil
		}

	case '8':
		// blk: (break :blk is break prong)
		switch len(action.Intermediates) {
		// DECRC - Restore Cursor
		case 0:
			if err := self.handler.Vt(StreamActionKeyRestoreCursor, StreamAction{Tag: StreamActionKeyRestoreCursor}); err != nil {
				return err
			}
			break prong // break :blk {}

		case 1:
			switch action.Intermediates[0] {
			// DECALN - Fill Screen with E
			case '#':
				if err := self.handler.Vt(StreamActionKeyDecaln, StreamAction{Tag: StreamActionKeyDecaln}); err != nil {
					return err
				}
				break prong // break :blk {}

			default:
			}

		default: // fall through
		}

		warn("stream", "unimplemented ESC action: %s", action.Format())

	// IND - Index
	case 'D':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyIndex, StreamAction{Tag: StreamActionKeyIndex}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid index command: %s", action.Format())
			return nil
		}

	// NEL - Next Line
	case 'E':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyNextLine, StreamAction{Tag: StreamActionKeyNextLine}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid next line command: %s", action.Format())
			return nil
		}

	// HTS - Horizontal Tab Set
	case 'H':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyTabSet, StreamAction{Tag: StreamActionKeyTabSet}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid tab set command: %s", action.Format())
			return nil
		}

	// RI - Reverse Index
	case 'M':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyReverseIndex, StreamAction{Tag: StreamActionKeyReverseIndex}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid reverse index command: %s", action.Format())
			return nil
		}

	// SS2 - Single Shift 2
	case 'N':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyInvokeCharset, StreamAction{Tag: StreamActionKeyInvokeCharset, InvokeCharset: StreamActionInvokeCharset{
				bank:    CharsetsActiveSlotGL,
				charset: CharsetsSlotsG2,
				locking: true,
			}}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid single shift 2 command: %s", action.Format())
			return nil
		}

	// SS3 - Single Shift 3
	case 'O':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyInvokeCharset, StreamAction{Tag: StreamActionKeyInvokeCharset, InvokeCharset: StreamActionInvokeCharset{
				bank:    CharsetsActiveSlotGL,
				charset: CharsetsSlotsG3,
				locking: true,
			}}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid single shift 3 command: %s", action.Format())
			return nil
		}

	// SPA - Start of Guarded Area
	case 'V':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyProtectedModeIso, StreamAction{Tag: StreamActionKeyProtectedModeIso}); err != nil {
				return err
			}
		default:
			warn("stream", "unimplemented ESC callback: %s", action.Format())
		}

	// EPA - End of Guarded Area
	case 'W':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyProtectedModeOff, StreamAction{Tag: StreamActionKeyProtectedModeOff}); err != nil {
				return err
			}
		default:
			warn("stream", "unimplemented ESC callback: %s", action.Format())
		}

	// DECID
	case 'Z':
		if len(action.Intermediates) == 0 {
			if err := self.handler.Vt(StreamActionKeyDeviceAttributes, StreamAction{Tag: StreamActionKeyDeviceAttributes, DeviceAttributes: ANSIDeviceAttributeReqPrimary}); err != nil {
				return err
			}
		} else {
			warn("stream", "unimplemented ESC callback: %s", action.Format())
		}

	// RIS - Full Reset
	case 'c':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyFullReset, StreamAction{Tag: StreamActionKeyFullReset}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid full reset command: %s", action.Format())
			return nil
		}

	// LS2 - Locking Shift 2
	case 'n':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyInvokeCharset, StreamAction{Tag: StreamActionKeyInvokeCharset, InvokeCharset: StreamActionInvokeCharset{
				bank:    CharsetsActiveSlotGL,
				charset: CharsetsSlotsG2,
				locking: false,
			}}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid single shift 2 command: %s", action.Format())
			return nil
		}

	// LS3 - Locking Shift 3
	case 'o':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyInvokeCharset, StreamAction{Tag: StreamActionKeyInvokeCharset, InvokeCharset: StreamActionInvokeCharset{
				bank:    CharsetsActiveSlotGL,
				charset: CharsetsSlotsG3,
				locking: false,
			}}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid single shift 3 command: %s", action.Format())
			return nil
		}

	// LS1R - Locking Shift 1 Right
	case '~':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyInvokeCharset, StreamAction{Tag: StreamActionKeyInvokeCharset, InvokeCharset: StreamActionInvokeCharset{
				bank:    CharsetsActiveSlotGR,
				charset: CharsetsSlotsG1,
				locking: false,
			}}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid locking shift 1 right command: %s", action.Format())
			return nil
		}

	// LS2R - Locking Shift 2 Right
	case '}':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyInvokeCharset, StreamAction{Tag: StreamActionKeyInvokeCharset, InvokeCharset: StreamActionInvokeCharset{
				bank:    CharsetsActiveSlotGR,
				charset: CharsetsSlotsG2,
				locking: false,
			}}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid locking shift 2 right command: %s", action.Format())
			return nil
		}

	// LS3R - Locking Shift 3 Right
	case '|':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyInvokeCharset, StreamAction{Tag: StreamActionKeyInvokeCharset, InvokeCharset: StreamActionInvokeCharset{
				bank:    CharsetsActiveSlotGR,
				charset: CharsetsSlotsG3,
				locking: false,
			}}); err != nil {
				return err
			}
		default:
			warn("stream", "invalid locking shift 3 right command: %s", action.Format())
			return nil
		}

	// Set application keypad mode
	case '=':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeySetMode, StreamAction{Tag: StreamActionKeySetMode, SetMode: StreamActionMode{mode: ModesModeKeypadKeys}}); err != nil {
				return err
			}
		default:
			warn("stream", "unimplemented setMode: %s", action.Format())
		}

	// Reset application keypad mode
	case '>':
		switch len(action.Intermediates) {
		case 0:
			if err := self.handler.Vt(StreamActionKeyResetMode, StreamAction{Tag: StreamActionKeyResetMode, ResetMode: StreamActionMode{mode: ModesModeKeypadKeys}}); err != nil {
				return err
			}
		default:
			warn("stream", "unimplemented setMode: %s", action.Format())
		}

	// Sets ST (string terminator). We don't have to do anything
	// because our parser always accepts ST.
	case '\\':

	default:
		warn("stream", "unimplemented ESC action: %s", action.Format())
	}
	return nil
}
