// WIREFRAME for src/terminal/stream.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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
//   - `deinit`, called by Stream.deinit ("The Handler type must also have a
//     `deinit` function"). It is `Deinit()`.
//   - `vtRaw` is optional in Zig (`@hasDecl(T, "vtRaw")` in nextNonUtf8). It
//     is the separate interface StreamHandlerVtRaw, which a handler may
//     also implement.
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
//     but is declared, because the project tests call it directly and
//     docs/mapping.md states what it must do;
//   - `const log = std.log.scoped(.stream)`: the port uses
//     warn("stream", ...);
//   - inside `Stream(Handler)`: `Self` (the receiver type), `Action` (a
//     re-export of stream.Action; a Go generic type cannot hold a nested
//     type, use StreamAction) and `T` (the handler type with a pointer
//     removed, used only for @hasDecl).

package terminal

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
func (self StreamActionKey) String() string { panic(notImplemented) }

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
// This takes ownership of the handler and will call deinit when the stream
// is deinitialized.
func StreamInit[H StreamHandler](h H) Stream[H] { panic(notImplemented) }

// StreamInitAlloc is Stream(Handler).initAlloc: initialize the stream that
// supports heap allocation as necessary.
//
// Zig: `alloc` is an Allocator that is stored in parser.osc_parser.alloc.
// As for OSCParserInit, nothing is allocated through it; nil is Zig's null
// and any other value is "an allocator was provided".
func StreamInitAlloc[H StreamHandler](alloc any, h H) Stream[H] { panic(notImplemented) }

// Deinit is Stream(Handler).deinit.
func (self *Stream[H]) Deinit() { panic(notImplemented) }

// NextSlice is Stream(Handler).nextSlice: process a string of characters.
func (self *Stream[H]) NextSlice(input []uint8) error { panic(notImplemented) }

// nextSliceCapped is Stream(Handler).nextSliceCapped.
//
// Zig asserts input.len <= cp_buf.len. Upstream decodes with SIMD here; see
// docs/mapping.md for what the port must return.
func (self *Stream[H]) nextSliceCapped(input []uint8, cp_buf []uint32) error {
	panic(notImplemented)
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
	panic(notImplemented)
}

// consumeUntilGround is Stream(Handler).consumeUntilGround: parses escape
// sequences until the parser reaches the ground state. Returns the number of
// bytes consumed from the provided input.
//
// Zig: returns !usize.
func (self *Stream[H]) consumeUntilGround(input []uint8) (uint, error) {
	panic(notImplemented)
}

// Next is Stream(Handler).next: like nextSlice but takes one byte and is
// necessarily a scalar operation that can't use SIMD. Prefer nextSlice if
// you can and try to get multiple bytes at once.
func (self *Stream[H]) Next(c uint8) error { panic(notImplemented) }

// nextUtf8 is Stream(Handler).nextUtf8: process the next byte and print as
// necessary.
//
// This assumes we're in the UTF-8 decoding state. If we may not be in the
// UTF-8 decoding state call nextSlice or next.
func (self *Stream[H]) nextUtf8(c uint8) error { panic(notImplemented) }

// handleCodepoint is Stream(Handler).handleCodepoint: to be called whenever
// the utf-8 decoder produces a codepoint.
//
// This function is abstracted this way to handle the case where the decoder
// emits a 0x1B after rejecting an ill-formed sequence.
//
// Zig: c is u21.
func (self *Stream[H]) handleCodepoint(c uint32) error { panic(notImplemented) }

// nextNonUtf8 is Stream(Handler).nextNonUtf8: process the next character and
// call any callbacks if necessary.
//
// This assumes that we're not in the UTF-8 decoding state. If we may be in
// the UTF-8 decoding state call nextSlice or next.
func (self *Stream[H]) nextNonUtf8(c uint8) error { panic(notImplemented) }

// Print is Stream(Handler).print.
//
// Zig: c is u21.
func (self *Stream[H]) Print(c uint32) error { panic(notImplemented) }

// Execute is Stream(Handler).execute.
func (self *Stream[H]) Execute(c uint8) error { panic(notImplemented) }

// csiDispatch is Stream(Handler).csiDispatch.
func (self *Stream[H]) csiDispatch(input ParserActionCSI) error { panic(notImplemented) }

// oscDispatch is Stream(Handler).oscDispatch.
func (self *Stream[H]) oscDispatch(cmd OSCCommand) error { panic(notImplemented) }

// configureCharset is Stream(Handler).configureCharset.
func (self *Stream[H]) configureCharset(intermediates []uint8, set CharsetsCharset) error {
	panic(notImplemented)
}

// escDispatch is Stream(Handler).escDispatch.
func (self *Stream[H]) escDispatch(action ParserActionESC) error { panic(notImplemented) }
