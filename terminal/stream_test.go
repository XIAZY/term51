// Tests translated one for one from src/terminal/stream.zig (Ghostty v1.3.1
// plus the term51 additional tests (A)): upstream's tests and the whole
// section (A). Each Go test carries the exact Zig title in a "// Zig: test"
// comment. The helpers expect, expectEqual, expectEqualSlices,
// expectEqualStrings, zigUnwrap and zigTry are defined in Parser_test.go and
// dcs_test.go. The helpers of this file are prefixed strA_.
//
// Translation notes:
//   - A Zig test handler (`const H = struct { ... pub fn vt(...) }`) is a
//     package-level type with a pointer-receiver Vt and Deinit, because Go
//     has no methods on local types; Stream[*H] is the stream, built with
//     StreamInit(&H{...}). Zig's `vt` has a comptime `action` and a payload
//     of type Action.Value(action); the Go Vt gets the key and the whole
//     StreamAction, and the payload is the field named by the tag. A Go
//     handler needs a Deinit: strA_noDeinit supplies an empty one. Where
//     several Zig tests declare the same handler, one Go type serves them.
//     Zig's `Stream(H).init(.{...})` takes the handler by value; the Go
//     stream holds the pointer, which the test keeps (`s.handler`).
//   - `var s: Stream(H) = .init(.{})` is `s := StreamInit(&H{})`;
//     `.initAlloc(testing.allocator, &rec)` is
//     StreamInitAlloc(&rec), a non-nil value (docs/mapping.md:
//     the field `alloc` keeps only whether an allocator was provided).
//   - Zig's `.{}` of the recording handler (log_prints defaults to true) is
//     strA_recorderInit(true), `.{ .log_prints = false }` is
//     strA_recorderInit(false); a bare Go zero value would not do.
//   - std.testing.allocator leak checks have no Go counterpart: the
//     `defer term51ARecRelease(&rec);` lines are kept as comments marked
//     Zig-only.
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and (expected, actual) argument order. `x.?` is zigUnwrap. `try f()`
//     on an error union is zigTry.
//   - A Zig `catch |err| { std.debug.print("..."); return err; }` that only
//     adds context to a failure is t.Logf before the call (go test shows it
//     when the test fails).
//   - Zig's `defer s.deinit()` inside a loop body runs at the end of each
//     iteration; Go's defer runs when the test function returns.
//   - `Parser.State` is ParserState, `s.parser.params_sep` has the methods
//     count and set (zig.go), `.utf8decoder = .{}` is UTF8Decoder{} and
//     `.parser = .init()` is ParserInit().
//   - Zig's `[:0]const u8` payloads are []uint8 in Go.
//   - Zig's `@tagName` of the enums that the recorder prints is the String()
//     method of the Go type. The wireframe has no String() for ColorSpecial
//     and ColorDynamic and OSCTerminator.String is Zig's `string` (the
//     terminator bytes), so strA_specialName, strA_dynamicName and
//     strA_terminatorName stand in for them (see the report).
//   - Not translated, see scripts/not-translated.txt: `test Action` and
//     the ten `cval` tests (C bindings).

package terminal

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// strA_allocator stands in for std.testing.allocator: any non-nil value.
var strA_allocator any = struct{}{}

// strA_noDeinit gives a test handler the empty Deinit that Go's
// StreamHandler requires.
type strA_noDeinit struct{}

// Deinit does nothing.
func (strA_noDeinit) Deinit() {}

// ---------------------------------------------------------------------------
// Upstream's tests
// ---------------------------------------------------------------------------

type strA_printH struct {
	strA_noDeinit
	c Optional[uint32]
}

func (self *strA_printH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyPrint:
		self.c = Some(value.Print.Cp)
	}
	return nil
}

// Zig: test "stream: print"
func TestStream_stream_print(t *testing.T) {
	// c: ?u21 = 0
	s := StreamInit(&strA_printH{c: Some(uint32(0))})
	zigTry(t, s.Next('x'))
	expectEqual(t, uint32('x'), zigUnwrap(t, s.handler.c))
}

// Zig: test "simd: print invalid utf-8"
func TestStream_simd_print_invalid_utf_8(t *testing.T) {
	// c: ?u21 = 0
	s := StreamInit(&strA_printH{c: Some(uint32(0))})
	zigTry(t, s.NextSlice([]uint8{0xFF}))
	expectEqual(t, uint32(0xFFFD), zigUnwrap(t, s.handler.c))
}

// Zig: test "simd: complete incomplete utf-8"
func TestStream_simd_complete_incomplete_utf_8(t *testing.T) {
	// c: ?u21 = null
	s := StreamInit(&strA_printH{})
	zigTry(t, s.NextSlice([]uint8{0xE0})) // 3 byte
	expect(t, !s.handler.c.Valid, "s.handler.c == null")
	zigTry(t, s.NextSlice([]uint8{0xA0})) // still incomplete
	expect(t, !s.handler.c.Valid, "s.handler.c == null")
	zigTry(t, s.NextSlice([]uint8{0x80}))
	expectEqual(t, uint32(0x800), zigUnwrap(t, s.handler.c))
}

type strA_cursorRightH struct {
	strA_noDeinit
	amount uint16
}

func (self *strA_cursorRightH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyCursorRight:
		self.amount = value.CursorRight.Value
	}
	return nil
}

// Zig: test "stream: cursor right (CUF)"
func TestStream_stream_cursor_right_CUF(t *testing.T) {
	s := StreamInit(&strA_cursorRightH{})
	zigTry(t, s.NextSlice([]uint8("\x1B[C")))
	expectEqual(t, uint16(1), s.handler.amount)

	zigTry(t, s.NextSlice([]uint8("\x1B[5C")))
	expectEqual(t, uint16(5), s.handler.amount)

	s.handler.amount = 0
	zigTry(t, s.NextSlice([]uint8("\x1B[5;4C")))
	expectEqual(t, uint16(0), s.handler.amount)

	s.handler.amount = 0
	zigTry(t, s.NextSlice([]uint8("\x1b[?3C")))
	expectEqual(t, uint16(0), s.handler.amount)
}

type strA_decSetModeH struct {
	strA_noDeinit
	mode ModesMode
}

func (self *strA_decSetModeH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeySetMode:
		self.mode = value.SetMode.Mode
	case StreamActionKeyResetMode:
		self.mode = ModesMode(1)
	}
	return nil
}

// Zig: test "stream: dec set mode (SM) and reset mode (RM)"
func TestStream_stream_dec_set_mode_SM_and_reset_mode_RM(t *testing.T) {
	// mode: modes.Mode = @as(modes.Mode, @enumFromInt(1))
	s := StreamInit(&strA_decSetModeH{mode: ModesMode(1)})
	zigTry(t, s.NextSlice([]uint8("\x1B[?6h")))
	expectEqual(t, ModesModeOrigin, s.handler.mode)

	zigTry(t, s.NextSlice([]uint8("\x1B[?6l")))
	expectEqual(t, ModesMode(1), s.handler.mode)

	s.handler.mode = ModesMode(1)
	zigTry(t, s.NextSlice([]uint8("\x1B[6 h")))
	expectEqual(t, ModesMode(1), s.handler.mode)
}

type strA_ansiSetModeH struct {
	strA_noDeinit
	mode Optional[ModesMode]
}

func (self *strA_ansiSetModeH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeySetMode:
		self.mode = Some(value.SetMode.Mode)
	case StreamActionKeyResetMode:
		self.mode = None[ModesMode]()
	}
	return nil
}

// Zig: test "stream: ansi set mode (SM) and reset mode (RM)"
func TestStream_stream_ansi_set_mode_SM_and_reset_mode_RM(t *testing.T) {
	s := StreamInit(&strA_ansiSetModeH{})
	zigTry(t, s.NextSlice([]uint8("\x1B[4h")))
	expectEqual(t, ModesModeInsert, zigUnwrap(t, s.handler.mode))

	zigTry(t, s.NextSlice([]uint8("\x1B[4l")))
	expect(t, !s.handler.mode.Valid, "s.handler.mode == null")

	s.handler.mode = None[ModesMode]()
	zigTry(t, s.NextSlice([]uint8("\x1B[>5h")))
	expect(t, !s.handler.mode.Valid, "s.handler.mode == null")
}

type strA_ansiSetModeUnknownH struct {
	strA_noDeinit
	mode Optional[ModesMode]
}

func (self *strA_ansiSetModeUnknownH) setMode(mode ModesMode, v bool) error {
	self.mode = None[ModesMode]()
	if v {
		self.mode = Some(mode)
	}
	return nil
}

func (self *strA_ansiSetModeUnknownH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = self
	_ = value
	return nil
}

// Zig: test "stream: ansi set mode (SM) and reset mode (RM) with unknown value"
func TestStream_stream_ansi_set_mode_SM_and_reset_mode_RM_with_unknown_value(t *testing.T) {
	s := StreamInit(&strA_ansiSetModeUnknownH{})
	zigTry(t, s.NextSlice([]uint8("\x1B[6h")))
	expect(t, !s.handler.mode.Valid, "s.handler.mode == null")

	zigTry(t, s.NextSlice([]uint8("\x1B[6l")))
	expect(t, !s.handler.mode.Valid, "s.handler.mode == null")
}

type strA_restoreModeH struct {
	strA_noDeinit
	called bool
}

func (self *strA_restoreModeH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyTopAndBottomMargin:
		self.called = true
	}
	return nil
}

// Zig: test "stream: restore mode"
func TestStream_stream_restore_mode(t *testing.T) {
	s := StreamInit(&strA_restoreModeH{})
	strA_feed(t, &s, "\x1B[?42r")
	expect(t, !s.handler.called, "!s.handler.called")
}

type strA_popKittyH struct {
	strA_noDeinit
	n uint16
}

func (self *strA_popKittyH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyKittyKeyboardPop:
		self.n = value.KittyKeyboardPop
	}
	return nil
}

// Zig: test "stream: pop kitty keyboard with no params defaults to 1"
func TestStream_stream_pop_kitty_keyboard_with_no_params_defaults_to_1(t *testing.T) {
	s := StreamInit(&strA_popKittyH{})
	strA_feed(t, &s, "\x1B[<u")
	expectEqual(t, uint16(1), s.handler.n)
}

type strA_decscaH struct {
	strA_noDeinit
	v Optional[ANSIProtectedMode]
}

func (self *strA_decscaH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyProtectedModeOff:
		self.v = Some(ANSIProtectedModeOff)
	case StreamActionKeyProtectedModeIso:
		self.v = Some(ANSIProtectedModeIso)
	case StreamActionKeyProtectedModeDec:
		self.v = Some(ANSIProtectedModeDec)
	}
	return nil
}

// Zig: test "stream: DECSCA"
func TestStream_stream_DECSCA(t *testing.T) {
	s := StreamInit(&strA_decscaH{})
	{
		strA_feed(t, &s, "\x1B[\"q")
		expectEqual(t, ANSIProtectedModeOff, zigUnwrap(t, s.handler.v))
	}
	{
		strA_feed(t, &s, "\x1B[0\"q")
		expectEqual(t, ANSIProtectedModeOff, zigUnwrap(t, s.handler.v))
	}
	{
		strA_feed(t, &s, "\x1B[2\"q")
		expectEqual(t, ANSIProtectedModeOff, zigUnwrap(t, s.handler.v))
	}
	{
		strA_feed(t, &s, "\x1B[1\"q")
		expectEqual(t, ANSIProtectedModeDec, zigUnwrap(t, s.handler.v))
	}
}

type strA_decedH struct {
	strA_noDeinit
	mode      Optional[CSIEraseDisplay]
	protected Optional[bool]
}

func (self *strA_decedH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyEraseDisplayBelow:
		self.mode = Some(CSIEraseDisplayBelow)
		self.protected = Some(value.EraseDisplayBelow)
	case StreamActionKeyEraseDisplayAbove:
		self.mode = Some(CSIEraseDisplayAbove)
		self.protected = Some(value.EraseDisplayAbove)
	case StreamActionKeyEraseDisplayComplete:
		self.mode = Some(CSIEraseDisplayComplete)
		self.protected = Some(value.EraseDisplayComplete)
	case StreamActionKeyEraseDisplayScrollback:
		self.mode = Some(CSIEraseDisplayScrollback)
		self.protected = Some(value.EraseDisplayScrollback)
	case StreamActionKeyEraseDisplayScrollComplete:
		self.mode = Some(CSIEraseDisplayScrollComplete)
		self.protected = Some(value.EraseDisplayScrollComplete)
	}
	return nil
}

// Zig: test "stream: DECED, DECSED"
func TestStream_stream_DECED_DECSED(t *testing.T) {
	s := StreamInit(&strA_decedH{})
	{
		strA_feed(t, &s, "\x1B[?J")
		expectEqual(t, CSIEraseDisplayBelow, zigUnwrap(t, s.handler.mode))
		expect(t, zigUnwrap(t, s.handler.protected), "s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[?0J")
		expectEqual(t, CSIEraseDisplayBelow, zigUnwrap(t, s.handler.mode))
		expect(t, zigUnwrap(t, s.handler.protected), "s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[?1J")
		expectEqual(t, CSIEraseDisplayAbove, zigUnwrap(t, s.handler.mode))
		expect(t, zigUnwrap(t, s.handler.protected), "s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[?2J")
		expectEqual(t, CSIEraseDisplayComplete, zigUnwrap(t, s.handler.mode))
		expect(t, zigUnwrap(t, s.handler.protected), "s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[?3J")
		expectEqual(t, CSIEraseDisplayScrollback, zigUnwrap(t, s.handler.mode))
		expect(t, zigUnwrap(t, s.handler.protected), "s.handler.protected.?")
	}

	{
		strA_feed(t, &s, "\x1B[J")
		expectEqual(t, CSIEraseDisplayBelow, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[0J")
		expectEqual(t, CSIEraseDisplayBelow, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[1J")
		expectEqual(t, CSIEraseDisplayAbove, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[2J")
		expectEqual(t, CSIEraseDisplayComplete, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[3J")
		expectEqual(t, CSIEraseDisplayScrollback, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
	{
		// Invalid and ignored by the handler
		strA_feed(t, &s, "\x1B[>0J")
		expectEqual(t, CSIEraseDisplayScrollback, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
}

type strA_decelH struct {
	strA_noDeinit
	mode      Optional[CSIEraseLine]
	protected Optional[bool]
}

func (self *strA_decelH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyEraseLineRight:
		self.mode = Some(CSIEraseLineRight)
		self.protected = Some(value.EraseLineRight)
	case StreamActionKeyEraseLineLeft:
		self.mode = Some(CSIEraseLineLeft)
		self.protected = Some(value.EraseLineLeft)
	case StreamActionKeyEraseLineComplete:
		self.mode = Some(CSIEraseLineComplete)
		self.protected = Some(value.EraseLineComplete)
	case StreamActionKeyEraseLineRightUnlessPendingWrap:
		self.mode = Some(CSIEraseLineRightUnlessPendingWrap)
		self.protected = Some(value.EraseLineRightUnlessPendingWrap)
	}
	return nil
}

// Zig: test "stream: DECEL, DECSEL"
func TestStream_stream_DECEL_DECSEL(t *testing.T) {
	s := StreamInit(&strA_decelH{})
	{
		strA_feed(t, &s, "\x1B[?K")
		expectEqual(t, CSIEraseLineRight, zigUnwrap(t, s.handler.mode))
		expect(t, zigUnwrap(t, s.handler.protected), "s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[?0K")
		expectEqual(t, CSIEraseLineRight, zigUnwrap(t, s.handler.mode))
		expect(t, zigUnwrap(t, s.handler.protected), "s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[?1K")
		expectEqual(t, CSIEraseLineLeft, zigUnwrap(t, s.handler.mode))
		expect(t, zigUnwrap(t, s.handler.protected), "s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[?2K")
		expectEqual(t, CSIEraseLineComplete, zigUnwrap(t, s.handler.mode))
		expect(t, zigUnwrap(t, s.handler.protected), "s.handler.protected.?")
	}

	{
		strA_feed(t, &s, "\x1B[K")
		expectEqual(t, CSIEraseLineRight, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[0K")
		expectEqual(t, CSIEraseLineRight, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[1K")
		expectEqual(t, CSIEraseLineLeft, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
	{
		strA_feed(t, &s, "\x1B[2K")
		expectEqual(t, CSIEraseLineComplete, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
	{
		// Invalid and ignored by the handler
		strA_feed(t, &s, "\x1B[<1K")
		expectEqual(t, CSIEraseLineComplete, zigUnwrap(t, s.handler.mode))
		expect(t, !zigUnwrap(t, s.handler.protected), "!s.handler.protected.?")
	}
}

type strA_decscusrH struct {
	strA_noDeinit
	style Optional[ANSICursorStyle]
}

func (self *strA_decscusrH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyCursorStyle:
		self.style = Some(value.CursorStyle)
	}
	return nil
}

// Zig: test "stream: DECSCUSR"
func TestStream_stream_DECSCUSR(t *testing.T) {
	s := StreamInit(&strA_decscusrH{})
	zigTry(t, s.NextSlice([]uint8("\x1B[ q")))
	expect(t, zigUnwrap(t, s.handler.style) == ANSICursorStyleDefault, "s.handler.style.? == .default")

	zigTry(t, s.NextSlice([]uint8("\x1B[1 q")))
	expect(t, zigUnwrap(t, s.handler.style) == ANSICursorStyleBlinkingBlock, "s.handler.style.? == .blinking_block")

	// Invalid and ignored by the handler
	zigTry(t, s.NextSlice([]uint8("\x1B[?0 q")))
	expect(t, zigUnwrap(t, s.handler.style) == ANSICursorStyleBlinkingBlock, "s.handler.style.? == .blinking_block")
}

// Zig: test "stream: DECSCUSR without space"
func TestStream_stream_DECSCUSR_without_space(t *testing.T) {
	s := StreamInit(&strA_decscusrH{})
	zigTry(t, s.NextSlice([]uint8("\x1B[q")))
	expect(t, !s.handler.style.Valid, "s.handler.style == null")

	zigTry(t, s.NextSlice([]uint8("\x1B[1q")))
	expect(t, !s.handler.style.Valid, "s.handler.style == null")
}

type strA_xtshiftescapeH struct {
	strA_noDeinit
	escape Optional[bool]
}

func (self *strA_xtshiftescapeH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyMouseShiftCapture:
		self.escape = Some(value.MouseShiftCapture)
	}
	return nil
}

// Zig: test "stream: XTSHIFTESCAPE"
func TestStream_stream_XTSHIFTESCAPE(t *testing.T) {
	s := StreamInit(&strA_xtshiftescapeH{})
	zigTry(t, s.NextSlice([]uint8("\x1B[>2s")))
	expect(t, !s.handler.escape.Valid, "s.handler.escape == null")

	zigTry(t, s.NextSlice([]uint8("\x1B[>s")))
	expect(t, zigUnwrap(t, s.handler.escape) == false, "s.handler.escape.? == false")

	zigTry(t, s.NextSlice([]uint8("\x1B[>0s")))
	expect(t, zigUnwrap(t, s.handler.escape) == false, "s.handler.escape.? == false")

	zigTry(t, s.NextSlice([]uint8("\x1B[>1s")))
	expect(t, zigUnwrap(t, s.handler.escape) == true, "s.handler.escape.? == true")

	// Invalid and ignored by the handler
	zigTry(t, s.NextSlice([]uint8("\x1B[1 s")))
	expect(t, zigUnwrap(t, s.handler.escape) == true, "s.handler.escape.? == true")
}

type strA_windowTitleSeenH struct {
	strA_noDeinit
	seen bool
}

func (self *strA_windowTitleSeenH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = value
	switch action {
	case StreamActionKeyWindowTitle:
		self.seen = true
	}
	return nil
}

// Zig: test "stream: change window title with invalid utf-8"
func TestStream_stream_change_window_title_with_invalid_utf_8(t *testing.T) {
	{
		s := StreamInit(&strA_windowTitleSeenH{})
		zigTry(t, s.NextSlice([]uint8("\x1b]2;abc\x1b\\")))
		expect(t, s.handler.seen, "s.handler.seen")
	}

	{
		s := StreamInit(&strA_windowTitleSeenH{})
		zigTry(t, s.NextSlice([]uint8("\x1b]2;abc\xc0\x1b\\")))
		expect(t, !s.handler.seen, "!s.handler.seen")
	}
}

type strA_insertBlanksCalledH struct {
	strA_noDeinit
	called bool
}

func (self *strA_insertBlanksCalledH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = value
	switch action {
	case StreamActionKeyInsertBlanks:
		self.called = true
	}
	return nil
}

// Zig: test "stream: insert characters"
func TestStream_stream_insert_characters(t *testing.T) {
	s := StreamInit(&strA_insertBlanksCalledH{})
	strA_feed(t, &s, "\x1B[42@")
	expect(t, s.handler.called, "s.handler.called")

	s.handler.called = false
	strA_feed(t, &s, "\x1B[?42@")
	expect(t, !s.handler.called, "!s.handler.called")
}

type strA_insertBlanksValueH struct {
	strA_noDeinit
	value Optional[uint]
}

func (self *strA_insertBlanksValueH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyInsertBlanks:
		self.value = Some(value.InsertBlanks)
	}
	return nil
}

// Zig: test "stream: insert characters explicit zero clamps to 1"
func TestStream_stream_insert_characters_explicit_zero_clamps_to_1(t *testing.T) {
	s := StreamInit(&strA_insertBlanksValueH{})
	strA_feed(t, &s, "\x1B[0@")
	expectEqual(t, uint(1), zigUnwrap(t, s.handler.value))
}

type strA_scoscH struct {
	strA_noDeinit
	called bool
}

func (self *strA_scoscH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = value
	switch action {
	case StreamActionKeyLeftAndRightMargin:
		panic("bad")
	case StreamActionKeyLeftAndRightMarginAmbiguous:
		self.called = true
	}
	return nil
}

// Zig: test "stream: SCOSC"
func TestStream_stream_SCOSC(t *testing.T) {
	s := StreamInit(&strA_scoscH{})
	strA_feed(t, &s, "\x1B[s")
	expect(t, s.handler.called, "s.handler.called")
}

type strA_scorcH struct {
	strA_noDeinit
	called bool
}

func (self *strA_scorcH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = value
	switch action {
	case StreamActionKeyRestoreCursor:
		self.called = true
	}
	return nil
}

// Zig: test "stream: SCORC"
func TestStream_stream_SCORC(t *testing.T) {
	s := StreamInit(&strA_scorcH{})
	strA_feed(t, &s, "\x1B[u")
	expect(t, s.handler.called, "s.handler.called")
}

type strA_tooManyParamsH struct {
	strA_noDeinit
}

func (self *strA_tooManyParamsH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = self
	_ = value
	switch action {
	case StreamActionKeyCursorRight:
		panic("unreachable")
	}
	return nil
}

// Zig: test "stream: too many csi params"
func TestStream_stream_too_many_csi_params(t *testing.T) {
	s := StreamInit(&strA_tooManyParamsH{})
	zigTry(t, s.NextSlice([]uint8("\x1B[1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1C")))
}

type strA_ignoreAllH struct {
	strA_noDeinit
}

func (self *strA_ignoreAllH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = self
	_ = action
	_ = value
	return nil
}

// Zig: test "stream: csi param too long"
func TestStream_stream_csi_param_too_long(t *testing.T) {
	s := StreamInit(&strA_ignoreAllH{})
	// 259 times '1'.
	zigTry(t, s.NextSlice([]uint8("\x1B["+strings.Repeat("1", 259)+"C")))
}

type strA_sizeReportH struct {
	strA_noDeinit
	style Optional[CSISizeReportStyle]
}

func (self *strA_sizeReportH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeySizeReport:
		self.style = Some(value.SizeReport)
	}
	return nil
}

// Zig: test "stream: send report with CSI t"
func TestStream_stream_send_report_with_CSI_t(t *testing.T) {
	s := StreamInit(&strA_sizeReportH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[14t")))
	expectEqual(t, Some(CSISizeReportStyleCsi14T), s.handler.style)

	zigTry(t, s.NextSlice([]uint8("\x1b[16t")))
	expectEqual(t, Some(CSISizeReportStyleCsi16T), s.handler.style)

	zigTry(t, s.NextSlice([]uint8("\x1b[18t")))
	expectEqual(t, Some(CSISizeReportStyleCsi18T), s.handler.style)

	zigTry(t, s.NextSlice([]uint8("\x1b[21t")))
	expectEqual(t, Some(CSISizeReportStyleCsi21T), s.handler.style)
}

type strA_invalidCsiTH struct {
	strA_noDeinit
	style Optional[CSISizeReportStyle]
}

func (self *strA_invalidCsiTH) sendSizeReport(style CSISizeReportStyle) {
	self.style = Some(style)
}

func (self *strA_invalidCsiTH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = self
	_ = action
	_ = value
	return nil
}

// Zig: test "stream: invalid CSI t"
func TestStream_stream_invalid_CSI_t(t *testing.T) {
	s := StreamInit(&strA_invalidCsiTH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[19t")))
	expectEqual(t, None[CSISizeReportStyle](), s.handler.style)
}

type strA_titlePushH struct {
	strA_noDeinit
	index Optional[uint16]
}

func (self *strA_titlePushH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyTitlePush:
		self.index = Some(value.TitlePush)
	}
	return nil
}

// Zig: test "stream: CSI t push title"
func TestStream_stream_CSI_t_push_title(t *testing.T) {
	s := StreamInit(&strA_titlePushH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[22;0t")))
	expectEqual(t, uint16(0), zigUnwrap(t, s.handler.index))
}

// Zig: test "stream: CSI t push title with explicit window"
func TestStream_stream_CSI_t_push_title_with_explicit_window(t *testing.T) {
	s := StreamInit(&strA_titlePushH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[22;2t")))
	expectEqual(t, uint16(0), zigUnwrap(t, s.handler.index))
}

// Zig: test "stream: CSI t push title with explicit icon"
func TestStream_stream_CSI_t_push_title_with_explicit_icon(t *testing.T) {
	s := StreamInit(&strA_titlePushH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[22;1t")))
	expectEqual(t, None[uint16](), s.handler.index)
}

// Zig: test "stream: CSI t push title with index"
func TestStream_stream_CSI_t_push_title_with_index(t *testing.T) {
	s := StreamInit(&strA_titlePushH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[22;0;5t")))
	expectEqual(t, uint16(5), zigUnwrap(t, s.handler.index))
}

type strA_titlePopH struct {
	strA_noDeinit
	index Optional[uint16]
}

func (self *strA_titlePopH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyTitlePop:
		self.index = Some(value.TitlePop)
	}
	return nil
}

// Zig: test "stream: CSI t pop title"
func TestStream_stream_CSI_t_pop_title(t *testing.T) {
	s := StreamInit(&strA_titlePopH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[23;0t")))
	expectEqual(t, uint16(0), zigUnwrap(t, s.handler.index))
}

// Zig: test "stream: CSI t pop title with explicit window"
func TestStream_stream_CSI_t_pop_title_with_explicit_window(t *testing.T) {
	s := StreamInit(&strA_titlePopH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[23;2t")))
	expectEqual(t, uint16(0), zigUnwrap(t, s.handler.index))
}

// Zig: test "stream: CSI t pop title with explicit icon"
func TestStream_stream_CSI_t_pop_title_with_explicit_icon(t *testing.T) {
	s := StreamInit(&strA_titlePopH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[23;1t")))
	expectEqual(t, None[uint16](), s.handler.index)
}

// Zig: test "stream: CSI t pop title with index"
func TestStream_stream_CSI_t_pop_title_with_index(t *testing.T) {
	s := StreamInit(&strA_titlePopH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[23;0;5t")))
	expectEqual(t, uint16(5), zigUnwrap(t, s.handler.index))
}

type strA_lastActionH struct {
	strA_noDeinit
	action Optional[StreamActionKey]
}

func (self *strA_lastActionH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = value
	self.action = Some(action)
	return nil
}

// Zig: test "stream CSI W clear tab stops"
func TestStream_stream_CSI_W_clear_tab_stops(t *testing.T) {
	s := StreamInit(&strA_lastActionH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[2W")))
	expectEqual(t, StreamActionKeyTabClearCurrent, zigUnwrap(t, s.handler.action))

	zigTry(t, s.NextSlice([]uint8("\x1b[5W")))
	expectEqual(t, StreamActionKeyTabClearAll, zigUnwrap(t, s.handler.action))
}

// Zig: test "stream CSI W tab set"
func TestStream_stream_CSI_W_tab_set(t *testing.T) {
	s := StreamInit(&strA_lastActionH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[W")))
	expectEqual(t, StreamActionKeyTabSet, zigUnwrap(t, s.handler.action))

	s.handler.action = None[StreamActionKey]()
	zigTry(t, s.NextSlice([]uint8("\x1b[0W")))
	expectEqual(t, StreamActionKeyTabSet, zigUnwrap(t, s.handler.action))

	s.handler.action = None[StreamActionKey]()
	zigTry(t, s.NextSlice([]uint8("\x1b[>W")))
	expect(t, !s.handler.action.Valid, "s.handler.action == null")

	s.handler.action = None[StreamActionKey]()
	zigTry(t, s.NextSlice([]uint8("\x1b[99W")))
	expect(t, !s.handler.action.Valid, "s.handler.action == null")
}

// Zig: test "stream CSI ? W reset tab stops"
func TestStream_stream_CSI_W_reset_tab_stops(t *testing.T) {
	s := StreamInit(&strA_lastActionH{})

	zigTry(t, s.NextSlice([]uint8("\x1b[?2W")))
	expect(t, !s.handler.action.Valid, "s.handler.action == null")

	zigTry(t, s.NextSlice([]uint8("\x1b[?5W")))
	expectEqual(t, StreamActionKeyTabReset, zigUnwrap(t, s.handler.action))

	// Invalid and ignored by the handler
	s.handler.action = None[StreamActionKey]()
	zigTry(t, s.NextSlice([]uint8("\x1b[?1;2;3W")))
	expect(t, !s.handler.action.Valid, "s.handler.action == null")
}

type strA_sgrUnderlineColorH struct {
	strA_noDeinit
	attrs  Optional[SGRAttribute]
	called bool
}

func (self *strA_sgrUnderlineColorH) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeySetAttribute:
		self.attrs = Some(value.SetAttribute)
		self.called = true
	}
	return nil
}

// Zig: test "stream: SGR with 17+ parameters for underline color"
func TestStream_stream_SGR_with_17_parameters_for_underline_color(t *testing.T) {
	s := StreamInit(&strA_sgrUnderlineColorH{})

	// Kakoune-style SGR with underline color as 17th parameter
	// This tests the fix where param 17 was being dropped
	zigTry(t, s.NextSlice([]uint8("\x1b[4:3;38;2;51;51;51;48;2;170;170;170;58;2;255;97;136;0m")))
	expect(t, s.handler.called, "s.handler.called")
}

type strA_tabClearOverflowH struct {
	strA_noDeinit
	called bool
}

func (self *strA_tabClearOverflowH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = value
	switch action {
	case StreamActionKeyTabClearCurrent, StreamActionKeyTabClearAll:
		self.called = true
	}
	return nil
}

// Zig: test "stream: tab clear with overflowing param"
func TestStream_stream_tab_clear_with_overflowing_param(t *testing.T) {
	// Regression test for a fuzz crash: CSI with a parameter value that
	// saturates to 65535 (u16 max) causes @enumFromInt to panic when
	// converting to TabClear (enum(u8)).
	s := StreamInit(&strA_tabClearOverflowH{})
	// This is the exact input from the fuzz crash (minus the mode byte):
	// CSI with a huge numeric param that saturates to 65535, followed by 'g'.
	zigTry(t, s.NextSlice([]uint8("\x1b[388888888888888888888888888888888888g\x1b[0m")))
}

// ---------------------------------------------------------------------------
// term51: additional tests (A). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// strA_recorder is term51ARecorder: it records every action a stream emits
// as one text line per action. Prints are always collected in cps, and only
// written to log when log_prints is set, so that large inputs can be
// compared cheaply.
type strA_recorder struct {
	// Zig: log: std.Io.Writer.Allocating.
	log []uint8
	// Zig: cps: std.ArrayList(u21).
	cps        []uint32
	log_prints bool
	deinits    uint
	// Prints seen since the last logged action, when log_prints is off.
	pending_prints uint
}

// strA_recorderInit is `.{}` of term51ARecorder (log_prints defaults to
// true), or `.{ .log_prints = false }`.
func strA_recorderInit(log_prints bool) strA_recorder {
	return strA_recorder{log_prints: log_prints}
}

// Deinit is term51ARecorder.deinit: it only counts the call.
func (self *strA_recorder) Deinit() {
	self.deinits += 1
}

// strA_recPut is term51ARecPut: append formatted text to the recorder's log.
func strA_recPut(self *strA_recorder, format string, args ...any) {
	self.log = append(self.log, fmt.Sprintf(format, args...)...)
}

// strA_recFlush is term51ARecFlush: with log_prints off, runs of prints are
// summarized as "prints N".
func strA_recFlush(self *strA_recorder) {
	if self.pending_prints == 0 {
		return
	}
	n := self.pending_prints
	self.pending_prints = 0
	strA_recPut(self, "prints %d\n", n)
}

// strA_recText is term51ARecText: the text log so far.
func strA_recText(self *strA_recorder) string {
	strA_recFlush(self)
	return string(self.log)
}

// strA_recReset is term51ARecReset: forget everything recorded so far.
func strA_recReset(self *strA_recorder) {
	self.pending_prints = 0
	self.log = nil
	self.cps = self.cps[:0]
}

// strA_byteStr is Zig's "{c}" of a u8.
func strA_byteStr(b uint8) string {
	return string([]uint8{b})
}

// strA_specialName is @tagName of a color.Special (the wireframe has no
// String()).
func strA_specialName(s ColorSpecial) string {
	switch s {
	case ColorSpecialBold:
		return "bold"
	case ColorSpecialUnderline:
		return "underline"
	case ColorSpecialBlink:
		return "blink"
	case ColorSpecialReverse:
		return "reverse"
	case ColorSpecialItalic:
		return "italic"
	}
	panic("unreachable")
}

// strA_dynamicName is @tagName of a color.Dynamic (the wireframe has no
// String()).
func strA_dynamicName(d ColorDynamic) string {
	switch d {
	case ColorDynamicForeground:
		return "foreground"
	case ColorDynamicBackground:
		return "background"
	case ColorDynamicCursor:
		return "cursor"
	case ColorDynamicPointerForeground:
		return "pointer_foreground"
	case ColorDynamicPointerBackground:
		return "pointer_background"
	case ColorDynamicTektronixForeground:
		return "tektronix_foreground"
	case ColorDynamicTektronixBackground:
		return "tektronix_background"
	case ColorDynamicHighlightBackground:
		return "highlight_background"
	case ColorDynamicTektronixCursor:
		return "tektronix_cursor"
	case ColorDynamicHighlightForeground:
		return "highlight_foreground"
	}
	panic("unreachable")
}

// strA_terminatorName is @tagName of an osc.Terminator (OSCTerminator.String
// is Zig's `string`, the terminator bytes).
func strA_terminatorName(t OSCTerminator) string {
	switch t {
	case OSCTerminatorSt:
		return "st"
	case OSCTerminatorBel:
		return "bel"
	}
	panic("unreachable")
}

// strA_recPutTarget is term51ARecPutTarget.
func strA_recPutTarget(self *strA_recorder, t OSCParsersColorTarget) {
	switch t.Tag {
	case OSCParsersColorTargetPalette:
		strA_recPut(self, "p%d", t.Palette)
	case OSCParsersColorTargetSpecial:
		strA_recPut(self, "s.%s", strA_specialName(t.Special))
	case OSCParsersColorTargetDynamic:
		strA_recPut(self, "d.%s", strA_dynamicName(t.Dynamic))
	}
}

// strA_cursorMovement is the CursorMovement payload of the eight cursor
// movement actions.
func strA_cursorMovement(action StreamActionKey, value StreamAction) StreamActionCursorMovement {
	switch action {
	case StreamActionKeyCursorUp:
		return value.CursorUp
	case StreamActionKeyCursorDown:
		return value.CursorDown
	case StreamActionKeyCursorLeft:
		return value.CursorLeft
	case StreamActionKeyCursorRight:
		return value.CursorRight
	case StreamActionKeyCursorCol:
		return value.CursorCol
	case StreamActionKeyCursorRow:
		return value.CursorRow
	case StreamActionKeyCursorColRelative:
		return value.CursorColRelative
	case StreamActionKeyCursorRowRelative:
		return value.CursorRowRelative
	}
	panic("unreachable")
}

// strA_modeOf is the Mode payload of the five mode actions.
func strA_modeOf(action StreamActionKey, value StreamAction) StreamActionMode {
	switch action {
	case StreamActionKeySetMode:
		return value.SetMode
	case StreamActionKeyResetMode:
		return value.ResetMode
	case StreamActionKeySaveMode:
		return value.SaveMode
	case StreamActionKeyRestoreMode:
		return value.RestoreMode
	case StreamActionKeyRequestMode:
		return value.RequestMode
	}
	panic("unreachable")
}

// strA_anyPayload is the `{any}` rendering of the payload of an action the
// recorder has no case for, and false for an action without a payload
// (Zig: @TypeOf(value) == void). No test of this file reaches a payload
// other than the integers and booleans, which print as Zig prints them; the
// other payloads are only printed with %v.
func strA_anyPayload(action StreamActionKey, value StreamAction) (string, bool) {
	switch action {
	case StreamActionKeyPrintRepeat:
		return fmt.Sprintf("%d", value.PrintRepeat), true
	case StreamActionKeyHorizontalTab:
		return fmt.Sprintf("%d", value.HorizontalTab), true
	case StreamActionKeyHorizontalTabBack:
		return fmt.Sprintf("%d", value.HorizontalTabBack), true
	case StreamActionKeyCursorStyle:
		return "." + value.CursorStyle.String(), true
	case StreamActionKeyEraseDisplayBelow:
		return fmt.Sprintf("%t", value.EraseDisplayBelow), true
	case StreamActionKeyEraseDisplayAbove:
		return fmt.Sprintf("%t", value.EraseDisplayAbove), true
	case StreamActionKeyEraseDisplayComplete:
		return fmt.Sprintf("%t", value.EraseDisplayComplete), true
	case StreamActionKeyEraseDisplayScrollback:
		return fmt.Sprintf("%t", value.EraseDisplayScrollback), true
	case StreamActionKeyEraseDisplayScrollComplete:
		return fmt.Sprintf("%t", value.EraseDisplayScrollComplete), true
	case StreamActionKeyEraseLineRight:
		return fmt.Sprintf("%t", value.EraseLineRight), true
	case StreamActionKeyEraseLineLeft:
		return fmt.Sprintf("%t", value.EraseLineLeft), true
	case StreamActionKeyEraseLineComplete:
		return fmt.Sprintf("%t", value.EraseLineComplete), true
	case StreamActionKeyEraseLineRightUnlessPendingWrap:
		return fmt.Sprintf("%t", value.EraseLineRightUnlessPendingWrap), true
	case StreamActionKeyDeleteChars:
		return fmt.Sprintf("%d", value.DeleteChars), true
	case StreamActionKeyEraseChars:
		return fmt.Sprintf("%d", value.EraseChars), true
	case StreamActionKeyInsertLines:
		return fmt.Sprintf("%d", value.InsertLines), true
	case StreamActionKeyInsertBlanks:
		return fmt.Sprintf("%d", value.InsertBlanks), true
	case StreamActionKeyDeleteLines:
		return fmt.Sprintf("%d", value.DeleteLines), true
	case StreamActionKeyScrollUp:
		return fmt.Sprintf("%d", value.ScrollUp), true
	case StreamActionKeyScrollDown:
		return fmt.Sprintf("%d", value.ScrollDown), true
	case StreamActionKeyRequestModeUnknown:
		return fmt.Sprintf("%v", value.RequestModeUnknown), true
	case StreamActionKeyTopAndBottomMargin:
		return fmt.Sprintf("%v", value.TopAndBottomMargin), true
	case StreamActionKeyLeftAndRightMargin:
		return fmt.Sprintf("%v", value.LeftAndRightMargin), true
	case StreamActionKeyModifyKeyFormat:
		return "." + value.ModifyKeyFormat.String(), true
	case StreamActionKeyMouseShiftCapture:
		return fmt.Sprintf("%t", value.MouseShiftCapture), true
	case StreamActionKeySizeReport:
		return fmt.Sprintf("%d", value.SizeReport), true
	case StreamActionKeyTitlePush:
		return fmt.Sprintf("%d", value.TitlePush), true
	case StreamActionKeyTitlePop:
		return fmt.Sprintf("%d", value.TitlePop), true
	case StreamActionKeyDeviceStatus:
		return fmt.Sprintf("%v", value.DeviceStatus), true
	case StreamActionKeyKittyKeyboardPush:
		return fmt.Sprintf("%v", value.KittyKeyboardPush), true
	case StreamActionKeyKittyKeyboardPop:
		return fmt.Sprintf("%d", value.KittyKeyboardPop), true
	case StreamActionKeyKittyKeyboardSet:
		return fmt.Sprintf("%v", value.KittyKeyboardSet), true
	case StreamActionKeyKittyKeyboardSetOr:
		return fmt.Sprintf("%v", value.KittyKeyboardSetOr), true
	case StreamActionKeyKittyKeyboardSetNot:
		return fmt.Sprintf("%v", value.KittyKeyboardSetNot), true
	case StreamActionKeyActiveStatusDisplay:
		return "." + value.ActiveStatusDisplay.String(), true
	}
	return "", false
}

// Vt is term51ARecorder.vt.
func (self *strA_recorder) Vt(action StreamActionKey, value *StreamAction) error {
	name := action.String()
	if action != StreamActionKeyPrint {
		strA_recFlush(self)
	}
	switch action {
	case StreamActionKeyPrint:
		self.cps = append(self.cps, value.Print.Cp)
		if self.log_prints {
			strA_recPut(self, "print %x\n", value.Print.Cp)
		} else {
			self.pending_prints += 1
		}
	case StreamActionKeyInvokeCharset:
		strA_recPut(
			self,
			"invoke_charset %s %s locking=%t\n",
			value.InvokeCharset.Bank.String(), value.InvokeCharset.Charset.String(), value.InvokeCharset.Locking,
		)
	case StreamActionKeyConfigureCharset:
		strA_recPut(
			self,
			"configure_charset %s %s\n",
			value.ConfigureCharset.Slot.String(), value.ConfigureCharset.Charset.String(),
		)
	case StreamActionKeySetMode, StreamActionKeyResetMode, StreamActionKeySaveMode, StreamActionKeyRestoreMode, StreamActionKeyRequestMode:
		strA_recPut(
			self,
			"%s %s\n",
			name, strA_modeOf(action, *value).Mode.String(),
		)
	case StreamActionKeyDeviceAttributes:
		strA_recPut(self, "device_attributes %s\n", value.DeviceAttributes.String())
	case StreamActionKeyWindowTitle:
		strA_recPut(self, "window_title %s\n", value.WindowTitle.Title)
	case StreamActionKeyReportPwd:
		strA_recPut(self, "report_pwd %s\n", value.ReportPwd.URL)
	case StreamActionKeyShowDesktopNotification:
		strA_recPut(
			self,
			"show_desktop_notification title=%s body=%s\n",
			value.ShowDesktopNotification.Title, value.ShowDesktopNotification.Body,
		)
	case StreamActionKeyStartHyperlink:
		// "{?s}" of the id.
		id := "null"
		if value.StartHyperlink.ID.Valid {
			id = string(value.StartHyperlink.ID.Value)
		}
		strA_recPut(
			self,
			"start_hyperlink uri=%s id=%s\n",
			value.StartHyperlink.URI, id,
		)
	case StreamActionKeyClipboardContents:
		strA_recPut(
			self,
			"clipboard_contents kind=%d data=%s\n",
			value.ClipboardContents.Kind, value.ClipboardContents.Data,
		)
	case StreamActionKeyMouseShape:
		strA_recPut(self, "mouse_shape %s\n", value.MouseShape.String())
	case StreamActionKeyProgressReport:
		// "{?d}" of the progress.
		progress := "null"
		if value.ProgressReport.Progress.Valid {
			progress = fmt.Sprintf("%d", value.ProgressReport.Progress.Value)
		}
		strA_recPut(
			self,
			"progress_report %s %s\n",
			value.ProgressReport.State.String(), progress,
		)
	case StreamActionKeySemanticPrompt:
		strA_recPut(
			self,
			"semantic_prompt %s opts=%s\n",
			value.SemanticPrompt.Action.String(), value.SemanticPrompt.OptionsUnvalidated,
		)
	case StreamActionKeyKittyColorReport:
		strA_recPut(self, "kitty_color_report %s n=%d", strA_terminatorName(value.KittyColorReport.Terminator), len(value.KittyColorReport.List))
		for _, req := range value.KittyColorReport.List {
			switch req.Tag {
			case KittyColorOSCRequestQuery:
				strA_recPut(self, " query(%s)", req.Query.Format())
			case KittyColorOSCRequestSet:
				v := req.Set
				strA_recPut(self, " set(%s=#%02x%02x%02x)", v.Key.Format(), v.Color.R, v.Color.G, v.Color.B)
			case KittyColorOSCRequestReset:
				strA_recPut(self, " reset(%s)", req.Reset.Format())
			}
		}
		strA_recPut(self, "\n")
	case StreamActionKeyColorOperation:
		strA_recPut(
			self,
			"color_operation %s %s n=%d",
			value.ColorOperation.Op.String(), strA_terminatorName(value.ColorOperation.Terminator), value.ColorOperation.Requests.count(),
		)
		reqs := value.ColorOperation.Requests
		it := reqs.constIterator(0)
		for req := it.next(); req != nil; req = it.next() {
			switch req.Tag {
			case OSCParsersColorRequestSet:
				v := req.Set
				strA_recPut(self, " set(")
				strA_recPutTarget(self, v.Target)
				strA_recPut(self, "=#%02x%02x%02x)", v.Color.R, v.Color.G, v.Color.B)
			case OSCParsersColorRequestQuery:
				strA_recPut(self, " query(")
				strA_recPutTarget(self, req.Query)
				strA_recPut(self, ")")
			case OSCParsersColorRequestReset:
				strA_recPut(self, " reset(")
				strA_recPutTarget(self, req.Reset)
				strA_recPut(self, ")")
			case OSCParsersColorRequestResetPalette:
				strA_recPut(self, " reset_palette")
			case OSCParsersColorRequestResetSpecial:
				strA_recPut(self, " reset_special")
			}
		}
		strA_recPut(self, "\n")
		// The list is owned by whoever receives the action.
		reqs.deinit()
	case StreamActionKeyCursorUp,
		StreamActionKeyCursorDown,
		StreamActionKeyCursorLeft,
		StreamActionKeyCursorRight,
		StreamActionKeyCursorCol,
		StreamActionKeyCursorRow,
		StreamActionKeyCursorColRelative,
		StreamActionKeyCursorRowRelative:
		strA_recPut(self, "%s %d\n", name, strA_cursorMovement(action, *value).Value)
	case StreamActionKeyCursorPos:
		strA_recPut(self, "cursor_pos %d %d\n", value.CursorPos.Row, value.CursorPos.Col)
	case StreamActionKeySetAttribute:
		strA_recPut(self, "set_attribute %s\n", value.SetAttribute.Tag.String())
	case StreamActionKeyDcsHook:
		strA_recPut(self, "dcs_hook inter=%s params=", value.DcsHook.Intermediates)
		for i, p := range value.DcsHook.Params {
			if i > 0 {
				strA_recPut(self, ",")
			}
			strA_recPut(self, "%d", p)
		}
		strA_recPut(self, " final=%s\n", strA_byteStr(value.DcsHook.Final))
	case StreamActionKeyDcsPut:
		strA_recPut(self, "%s %x\n", name, value.DcsPut)
	case StreamActionKeyApcPut:
		strA_recPut(self, "%s %x\n", name, value.ApcPut)
	default:
		if payload, ok := strA_anyPayload(action, *value); ok {
			strA_recPut(self, "%s %s\n", name, payload)
		} else {
			strA_recPut(self, "%s\n", name)
		}
	}
	return nil
}

type strA_stream = Stream[*strA_recorder]

// strA_errTestFailure is error.Term51ATestFailure.
var strA_errTestFailure = errors.New("Term51ATestFailure")

type strA_rawMode uint8

const (
	strA_rawPass strA_rawMode = iota
	strA_rawSkip
	strA_rawFail
)

// strA_rawRecorder is term51ARawRecorder: a recorder that also exposes
// vtRaw, to observe and control the raw parser actions before the stream
// processes them.
type strA_rawRecorder struct {
	base strA_recorder
	// Zig: raw: std.Io.Writer.Allocating.
	raw  []uint8
	mode strA_rawMode
	// When set, only this kind of raw action is skipped / failed; every
	// other action is passed through.
	only Optional[ParserActionTag]
}

// strA_rawRecorderInit is `.{}` of term51ARawRecorder.
func strA_rawRecorderInit() strA_rawRecorder {
	return strA_rawRecorder{base: strA_recorderInit(true)}
}

// Deinit is term51ARawRecorder.deinit.
func (self *strA_rawRecorder) Deinit() {
	self.base.deinits += 1
}

// Vt is term51ARawRecorder.vt.
func (self *strA_rawRecorder) Vt(action StreamActionKey, value *StreamAction) error {
	return self.base.Vt(action, value)
}

// VtRaw is term51ARawRecorder.vtRaw.
func (self *strA_rawRecorder) VtRaw(action *ParserAction) (bool, error) {
	w := func(format string, args ...any) {
		self.raw = append(self.raw, fmt.Sprintf(format, args...)...)
	}
	switch action.Tag {
	case ParserActionPrint:
		w("print %x\n", action.Print)
	case ParserActionExecute:
		w("execute %x\n", action.Execute)
	case ParserActionCsiDispatch:
		w("csi_dispatch %s\n", strA_byteStr(action.CsiDispatch.Final))
	case ParserActionEscDispatch:
		w("esc_dispatch %s%s\n", action.EscDispatch.Intermediates, strA_byteStr(action.EscDispatch.Final))
	case ParserActionOscDispatch:
		w("osc_dispatch %s\n", action.OscDispatch.Tag.String())
	case ParserActionDcsHook:
		w("dcs_hook %s\n", strA_byteStr(action.DcsHook.Final))
	case ParserActionDcsPut:
		w("dcs_put %x\n", action.DcsPut)
	case ParserActionDcsUnhook:
		w("dcs_unhook\n")
	case ParserActionApcStart:
		w("apc_start\n")
	case ParserActionApcPut:
		w("apc_put %x\n", action.ApcPut)
	case ParserActionApcEnd:
		w("apc_end\n")
	}
	if self.only.Valid {
		if self.only.Value != action.Tag {
			return false, nil
		}
	}
	switch self.mode {
	case strA_rawPass:
		return false, nil
	case strA_rawSkip:
		return true, nil
	case strA_rawFail:
		return false, strA_errTestFailure
	}
	panic("unreachable")
}

type strA_rawStream = Stream[*strA_rawRecorder]

// strA_feed is term51AFeed: feed input one byte at a time through
// Stream.next.
func strA_feed[H StreamHandler](t *testing.T, s *Stream[H], input string) {
	t.Helper()
	for _, c := range []uint8(input) {
		zigTry(t, s.Next(c))
	}
}

// strA_expectSlice is term51AExpectSlice: feed input through Stream.nextSlice
// into a fresh recorder and compare the recorded text with expected.
func strA_expectSlice(t *testing.T, input string, expected string) {
	t.Helper()
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(input)))
	expectEqualStrings(t, expected, strA_recText(&rec))
}

// strA_expectNext is term51AExpectNext: feed input through Stream.next into a
// fresh recorder and compare the recorded text with expected.
func strA_expectNext(t *testing.T, input string, expected string) {
	t.Helper()
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	strA_feed(t, &s, input)
	expectEqualStrings(t, expected, strA_recText(&rec))
}

// strA_expectBoth is term51AExpectBoth: both of the above: the
// byte-at-a-time and the slice entry points must emit exactly the same
// actions.
func strA_expectBoth(t *testing.T, input string, expected string) {
	t.Helper()
	strA_expectNext(t, input, expected)
	strA_expectSlice(t, input, expected)
}

// strA_feedChunked is term51AFeedChunked: mirror of the chunking loop in
// Stream.nextSlice (which, in a build without SIMD, takes a per-byte path
// instead), so that nextSliceCapped and its 4096 entry cp buffer are
// exercised in every build configuration.
func strA_feedChunked(t *testing.T, s *strA_stream, input []uint8) {
	t.Helper()
	var cp_buf [4096]uint32
	i := 0
	for {
		len_ := min(len(cp_buf), len(input)-i)
		zigTry(t, s.nextSliceCapped(input[i:i+len_], cp_buf[:]))
		i += len_
		if i >= len(input) {
			break
		}
	}
}

// strA_capped calls nextSliceCapped with the string as input.
func strA_capped(t *testing.T, s *strA_stream, input string, buf []uint32) {
	t.Helper()
	zigTry(t, s.nextSliceCapped([]uint8(input), buf))
}

// strA_expectSpec is term51AExpect.
type strA_expectSpec struct {
	// Every codepoint printed, in order.
	cps []uint32
	// Every non-print action, as text.
	log string
	// Whether the UTF-8 decoder is left in the middle of a sequence.
	decoder_pending bool
	// Zig: default .ground, which is the zero value.
	parser_state ParserState
}

// strA_checkAllWays is term51ACheckAllWays: feed input three ways (per byte,
// nextSlice, and the 4096-chunked nextSliceCapped loop) and check each
// against expect.
func strA_checkAllWays(t *testing.T, input []uint8, expect_ strA_expectSpec) {
	t.Helper()
	for way := 0; way < 3; way++ {
		rec := strA_recorderInit(false)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		switch way {
		case 0:
			strA_feed(t, &s, string(input))
		case 1:
			zigTry(t, s.NextSlice(input))
		case 2:
			strA_feedChunked(t, &s, input)
		default:
			panic("unreachable")
		}
		expectEqualSlices(t, expect_.cps, rec.cps)
		expectEqualStrings(t, expect_.log, strA_recText(&rec))
		expectEqual(t, expect_.decoder_pending, s.utf8decoder.state != 0)
		expectEqual(t, expect_.parser_state, s.parser.state)
	}
}

// strA_fill is term51AFill.
func strA_fill(byte_ uint8, n int) []uint8 {
	buf := make([]uint8, n)
	for i := range buf {
		buf[i] = byte_
	}
	return buf
}

// strA_repeatCp is term51ARepeatCp.
func strA_repeatCp(cp uint32, n int) []uint32 {
	buf := make([]uint32, n)
	for i := range buf {
		buf[i] = cp
	}
	return buf
}

// strA_esc is term51AEsc: call escDispatch with the given intermediates and
// final.
func strA_esc(t *testing.T, s *strA_stream, intermediates []uint8, final uint8) {
	t.Helper()
	var buf [8]uint8
	copy(buf[0:len(intermediates)], intermediates)
	zigTry(t, s.escDispatch(ParserActionESC{Intermediates: buf[0:len(intermediates)], Final: final}))
}

// strA_escNone is term51AEscNone: what `ESC <final>` (no intermediates)
// emits.
func strA_escNone(final uint8) string {
	switch final {
	case '7':
		return "save_cursor\n"
	case '8':
		return "restore_cursor\n"
	case 'D':
		return "index\n"
	case 'E':
		return "next_line\n"
	case 'H':
		return "tab_set\n"
	case 'M':
		return "reverse_index\n"
	case 'N':
		return "invoke_charset GL G2 locking=true\n"
	case 'O':
		return "invoke_charset GL G3 locking=true\n"
	case 'V':
		return "protected_mode_iso\n"
	case 'W':
		return "protected_mode_off\n"
	case 'Z':
		return "device_attributes primary\n"
	case 'c':
		return "full_reset\n"
	case 'n':
		return "invoke_charset GL G2 locking=false\n"
	case 'o':
		return "invoke_charset GL G3 locking=false\n"
	case '~':
		return "invoke_charset GR G1 locking=false\n"
	case '}':
		return "invoke_charset GR G2 locking=false\n"
	case '|':
		return "invoke_charset GR G3 locking=false\n"
	case '=':
		return "set_mode keypad_keys\n"
	case '>':
		return "reset_mode keypad_keys\n"
	}
	return ""
}

// strA_untilGround is `try s.consumeUntilGround(input)`.
func strA_untilGround[H StreamHandler](t *testing.T, s *Stream[H], input string) uint {
	t.Helper()
	n, err := s.consumeUntilGround([]uint8(input))
	zigTry(t, err)
	return n
}

// strA_allEscapes is `try s.consumeAllEscapes(input)`.
func strA_allEscapes[H StreamHandler](t *testing.T, s *Stream[H], input string) uint {
	t.Helper()
	n, err := s.consumeAllEscapes([]uint8(input))
	zigTry(t, err)
	return n
}

// ---------------------------------------------------------------------------
// Action payload C helpers
// ---------------------------------------------------------------------------

// Not translated (Zig: test Action): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.Print.cval: widens the codepoint to u32"): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.DeviceStatus.cval: is the packed request tag"): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.Mode.cval: is the packed mode tag"): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.KittyKeyboardFlags.cval: is the flag bit set"): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.WindowTitle.cval: borrows the title"): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.ReportPwd.cval: borrows the url"): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.ShowDesktopNotification.cval: borrows title and body"): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.StartHyperlink.cval: a missing id becomes an empty string"): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.ClipboardContents.cval: kind and borrowed data"): see scripts/not-translated.txt.

// Not translated (Zig: test "term51: Action.ColorOperation.cval: is void"): see scripts/not-translated.txt.

// ---------------------------------------------------------------------------
// Stream.init / initAlloc / deinit
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.init: starts in the ground state with no allocator"
func TestStream_term51_Stream_init_starts_in_the_ground_state_with_no_allocator(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	expect(t, s.handler == &rec, "s.handler == &rec")
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqual(t, uint8(0), s.utf8decoder.state)
	expectEqual(t, uint32(0), s.utf8decoder.accumulator)
	expectEqual(t, uint8(0), s.parser.intermediates_idx)
	expectEqual(t, uint8(0), s.parser.params_idx)
	expectEqual(t, uint16(0), s.parser.param_acc)
	expectEqual(t, uint8(0), s.parser.param_acc_idx)
	expectEqual(t, 0, s.parser.params_sep.count())
	expect(t, !s.parser.osc_parser.alloc, "s.parser.osc_parser.alloc == null")
	expectEqual(t, OSCParserStateStart, s.parser.osc_parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))
}

type strA_initByValueH struct {
	n uint32
}

func (self *strA_initByValueH) Deinit() {}

func (self *strA_initByValueH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = value
	if action == StreamActionKeyBell {
		self.n += 1
	}
	return nil
}

// Zig: test "term51: Stream.init: takes the handler by value"
func TestStream_term51_Stream_init_takes_the_handler_by_value(t *testing.T) {
	s := StreamInit(&strA_initByValueH{n: 41})
	expectEqual(t, uint32(41), s.handler.n)
	zigTry(t, s.Next(0x07))
	expectEqual(t, uint32(42), s.handler.n)
}

// Zig: test "term51: Stream.init: a stream without an allocator drops OSC sequences that need one"
func TestStream_term51_Stream_init_a_stream_without_an_allocator_drops_OSC_sequences_that_need_one(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// Colour operations (with and without a payload) need an allocator.
	zigTry(t, s.NextSlice([]uint8("\x1b]4;1;rgb:ff/00/00\x07")))
	zigTry(t, s.NextSlice([]uint8("\x1b]11;?\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b]104\x07")))
	zigTry(t, s.NextSlice([]uint8("\x1b]104;1\x07")))
	zigTry(t, s.NextSlice([]uint8("\x1b]110\x07")))
	// So does the kitty colour protocol.
	zigTry(t, s.NextSlice([]uint8("\x1b]21;foreground=?\x07")))
	expectEqualStrings(t, "", strA_recText(&rec))

	// Sequences that fit in the fixed buffer still work.
	zigTry(t, s.NextSlice([]uint8("\x1b]0;title\x07")))
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;http://a\x07")))
	zigTry(t, s.NextSlice([]uint8("\x1b]52;c;aGk=\x07")))
	expectEqualStrings(
		t,
		"window_title title\n"+
			"start_hyperlink uri=http://a id=null\n"+
			"clipboard_contents kind=99 data=aGk=\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.initAlloc: stores the allocator in the OSC parser and is otherwise like init"
func TestStream_term51_Stream_initAlloc_stores_the_allocator_in_the_OSC_parser_and_is_otherwise_like_init(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInitAlloc(&rec)
	defer s.Deinit()

	expect(t, s.handler == &rec, "s.handler == &rec")
	expect(t, s.parser.osc_parser.alloc, "s.parser.osc_parser.alloc != null")
	// Zig compares the vtable pointers of the two allocators.
	// Zig: expectEqual(testing.allocator.vtable, ...alloc.?.vtable). Go keeps
	// only whether there is one.
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqual(t, uint8(0), s.utf8decoder.state)
	expectEqual(t, uint8(0), s.parser.params_idx)
	expectEqual(t, uint8(0), s.parser.intermediates_idx)
	expectEqual(t, OSCParserStateStart, s.parser.osc_parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.initAlloc: OSC sequences that need an allocator are processed"
func TestStream_term51_Stream_initAlloc_OSC_sequences_that_need_an_allocator_are_processed(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInitAlloc(&rec)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("\x1b]4;1;rgb:ff/00/00\x07")))
	expectEqualStrings(
		t,
		"color_operation osc_4 bel n=1 set(p1=#ff0000)\n",
		strA_recText(&rec),
	)
	strA_recReset(&rec)

	zigTry(t, s.NextSlice([]uint8("\x1b]11;?\x1b\\")))
	expectEqualStrings(
		t,
		"color_operation osc_11 st n=1 query(d.background)\n",
		strA_recText(&rec),
	)
	strA_recReset(&rec)

	// OSC 104 with no payload: reset the whole palette.
	zigTry(t, s.NextSlice([]uint8("\x1b]104\x07")))
	expectEqualStrings(
		t,
		"color_operation osc_104 bel n=1 reset_palette\n",
		strA_recText(&rec),
	)
	strA_recReset(&rec)

	zigTry(t, s.NextSlice([]uint8("\x1b]110\x1b\\")))
	expectEqualStrings(
		t,
		"color_operation osc_110 st n=1 reset(d.foreground)\n",
		strA_recText(&rec),
	)
	strA_recReset(&rec)

	// The kitty colour list is owned by the parser and freed by deinit.
	zigTry(t, s.NextSlice([]uint8("\x1b]21;foreground=?;background=rgb:f0/f8/ff\x07")))
	expectEqualStrings(
		t,
		"kitty_color_report bel n=2 query(foreground) set(background=#f0f8ff)\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.initAlloc: large OSC 52 payloads are accepted only with an allocator"
func TestStream_term51_Stream_initAlloc_large_OSC_52_payloads_are_accepted_only_with_an_allocator(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	big := string(strA_fill('A', 3000))
	// Zig-only: defer alloc.free(big);
	input := "\x1b]52;c;" + big + "\x07"
	// Zig-only: defer alloc.free(input);
	expected := "clipboard_contents kind=99 data=" + big + "\n"
	// Zig-only: defer alloc.free(expected);

	// With an allocator the whole payload is delivered.
	{
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInitAlloc(&rec)
		defer s.Deinit()
		zigTry(t, s.NextSlice([]uint8(input)))
		expectEqualStrings(t, expected, strA_recText(&rec))
	}

	// Without one it exceeds the 2048-byte fixed buffer and is dropped.
	{
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		zigTry(t, s.NextSlice([]uint8(input)))
		expectEqualStrings(t, "", strA_recText(&rec))
		// ... and the stream is still usable afterwards.
		zigTry(t, s.NextSlice([]uint8("ok")))
		expectEqualStrings(t, "print 6f\nprint 6b\n", strA_recText(&rec))
	}

	// The fixed buffer holds exactly MAX_BUF bytes of "<kind>;<data>":
	// 2048 bytes in total (the terminating NUL does not fit otherwise).
	{
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		fits := string(strA_fill('B', OSCParserMAX_BUF-2-1))
		// Zig-only: defer alloc.free(fits);
		ok_input := "\x1b]52;c;" + fits + "\x07"
		// Zig-only: defer alloc.free(ok_input);
		zigTry(t, s.NextSlice([]uint8(ok_input)))
		ok_expected := "clipboard_contents kind=99 data=" + fits + "\n"
		// Zig-only: defer alloc.free(ok_expected);
		expectEqualStrings(t, ok_expected, strA_recText(&rec))
	}
}

// Zig: test "term51: Stream.initAlloc: OSC titles longer than the fixed buffer are dropped even with an allocator"
func TestStream_term51_Stream_initAlloc_OSC_titles_longer_than_the_fixed_buffer_are_dropped_even_with_an_allocator(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	long := string(strA_fill('x', OSCParserMAX_BUF))
	// Zig-only: defer alloc.free(long);
	input := "\x1b]0;" + long + "\x07"
	// Zig-only: defer alloc.free(input);

	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInitAlloc(&rec)
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(input)))
	expectEqualStrings(t, "", strA_recText(&rec))

	// One byte shorter than the buffer (leaving room for the NUL) is fine.
	fits := string(strA_fill('y', OSCParserMAX_BUF-1))
	// Zig-only: defer alloc.free(fits);
	input2 := "\x1b]0;" + fits + "\x07"
	// Zig-only: defer alloc.free(input2);
	zigTry(t, s.NextSlice([]uint8(input2)))
	expected := "window_title " + fits + "\n"
	// Zig-only: defer alloc.free(expected);
	expectEqualStrings(t, expected, strA_recText(&rec))
}

// Zig: test "term51: Stream.deinit: deinitializes the handler exactly once"
func TestStream_term51_Stream_deinit_deinitializes_the_handler_exactly_once(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	expectEqual(t, uint(0), rec.deinits)
	s.Deinit()
	expectEqual(t, uint(1), rec.deinits)
}

// Zig: test "term51: Stream.deinit: frees OSC state that was still being collected"
func TestStream_term51_Stream_deinit_frees_OSC_state_that_was_still_being_collected(t *testing.T) {
	// testing.allocator reports a leak if the heap-backed OSC 52 buffer of
	// the parser is not released by deinit.
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInitAlloc(&rec)
	zigTry(t, s.NextSlice([]uint8("\x1b]52;c;QUJD")))
	expect(t, s.parser.osc_parser.allocating.Valid, "s.parser.osc_parser.allocating != null")
	s.Deinit()
	expectEqual(t, uint(1), rec.deinits)
	expectEqualStrings(t, "", strA_recText(&rec))
}

type strA_deinitFlagH struct {
	flag *bool
}

func (self *strA_deinitFlagH) Deinit() {
	*self.flag = true
}

func (self *strA_deinitFlagH) Vt(action StreamActionKey, value *StreamAction) error {
	_ = self
	_ = action
	_ = value
	return nil
}

// Zig: test "term51: Stream.deinit: calls deinit on a handler passed by value"
func TestStream_term51_Stream_deinit_calls_deinit_on_a_handler_passed_by_value(t *testing.T) {
	flag := false
	s := StreamInit(&strA_deinitFlagH{flag: &flag})
	expect(t, !flag, "!flag")
	s.Deinit()
	expect(t, flag, "flag")
}

// ---------------------------------------------------------------------------
// Stream.print / Stream.execute
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.print: emits a print action with the codepoint"
func TestStream_term51_Stream_print_emits_a_print_action_with_the_codepoint(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.Print('A'))
	zigTry(t, s.Print(0x20AC))
	zigTry(t, s.Print(0x1F600))
	zigTry(t, s.Print(0x10FFFF))
	zigTry(t, s.Print(0x1FFFFF))
	// No control-character interpretation happens here.
	zigTry(t, s.Print(0))
	zigTry(t, s.Print(0x07))
	zigTry(t, s.Print(0x1B))
	zigTry(t, s.Print(0x9B))
	zigTry(t, s.Print(0x7F))
	expectEqualStrings(
		t,
		"print 41\nprint 20ac\nprint 1f600\nprint 10ffff\nprint 1fffff\n"+
			"print 0\nprint 7\nprint 1b\nprint 9b\nprint 7f\n",
		strA_recText(&rec),
	)
	expectEqual(t, 10, len(rec.cps))
	// Printing does not touch the parser.
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.execute: every C0 control"
func TestStream_term51_Stream_execute_every_C0_control(t *testing.T) {
	for i := 0; i < 0x80; i++ {
		c := uint8(i)
		var expected string
		switch c {
		case 0x05:
			expected = "enquiry\n"
		case 0x07:
			expected = "bell\n"
		case 0x08:
			expected = "backspace\n"
		case 0x09:
			expected = "horizontal_tab 1\n"
		case 0x0A, 0x0B, 0x0C:
			expected = "linefeed\n"
		case 0x0D:
			expected = "carriage_return\n"
		case 0x0E:
			expected = "invoke_charset GL G1 locking=false\n"
		case 0x0F:
			expected = "invoke_charset GL G0 locking=false\n"
		// NUL, SOH and STX are ignored on purpose; everything else
		// (including ESC, CAN, SUB, DEL and all printable ASCII) is
		// an unknown control and is ignored.
		default:
			expected = ""
		}
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		zigTry(t, s.Execute(c))
		t.Logf("execute 0x%x", c)
		expectEqualStrings(t, expected, strA_recText(&rec))
		// execute never changes the parser state.
		expectEqual(t, ParserStateGround, s.parser.state)
	}
}

// Zig: test "term51: Stream.execute: named controls emit exactly one action each"
func TestStream_term51_Stream_execute_named_controls_emit_exactly_one_action_each(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// NUL, SOH, STX: ignored.
	zigTry(t, s.Execute(0x00))
	zigTry(t, s.Execute(0x01))
	zigTry(t, s.Execute(0x02))
	expectEqualStrings(t, "", strA_recText(&rec))
	// Unnamed C0 controls: ignored.
	zigTry(t, s.Execute(0x03))
	zigTry(t, s.Execute(0x04))
	zigTry(t, s.Execute(0x06))
	zigTry(t, s.Execute(0x10))
	zigTry(t, s.Execute(0x18))
	zigTry(t, s.Execute(0x1A))
	zigTry(t, s.Execute(0x1B))
	zigTry(t, s.Execute(0x1F))
	expectEqualStrings(t, "", strA_recText(&rec))

	zigTry(t, s.Execute(0x05))
	zigTry(t, s.Execute(0x07))
	zigTry(t, s.Execute(0x08))
	zigTry(t, s.Execute(0x09))
	zigTry(t, s.Execute(0x0A))
	zigTry(t, s.Execute(0x0B))
	zigTry(t, s.Execute(0x0C))
	zigTry(t, s.Execute(0x0D))
	zigTry(t, s.Execute(0x0E))
	zigTry(t, s.Execute(0x0F))
	expectEqualStrings(
		t,
		"enquiry\nbell\nbackspace\nhorizontal_tab 1\n"+
			"linefeed\nlinefeed\nlinefeed\ncarriage_return\n"+
			"invoke_charset GL G1 locking=false\n"+
			"invoke_charset GL G0 locking=false\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.execute: 0x20..0x7F are not controls and are ignored"
func TestStream_term51_Stream_execute_0x20_0x7F_are_not_controls_and_are_ignored(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	for c := 0x20; c < 0x80; c++ {
		zigTry(t, s.Execute(uint8(c)))
	}
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.execute: C1 controls are ESC plus the byte minus 0x40"
func TestStream_term51_Stream_execute_C1_controls_are_ESC_plus_the_byte_minus_0x40(t *testing.T) {
	// The bytes that emit something, with the equivalent ESC final.
	emitting := []struct {
		first  uint8
		second uint8
	}{
		{0x84, 'D'}, // IND
		{0x85, 'E'}, // NEL
		{0x88, 'H'}, // HTS
		{0x8D, 'M'}, // RI
		{0x8E, 'N'}, // SS2
		{0x8F, 'O'}, // SS3
		{0x96, 'V'}, // SPA
		{0x97, 'W'}, // EPA
		{0x9A, 'Z'}, // DECID
		{0xA3, 'c'}, // not a real C1, but 0xA3 - 0x40 is 'c' (RIS)
		{0xAE, 'n'},
		{0xAF, 'o'},
		{0xBC, '|'},
		{0xBD, '}'},
		{0xBE, '~'},
	}

	for i := 0x80; i < 0x100; i++ {
		c := uint8(i)
		expected := ""
		for _, e := range emitting {
			if e.first == c {
				expected = strA_escNone(e.second)
			}
		}

		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		zigTry(t, s.Execute(c))
		t.Logf("execute 0x%x", c)
		expectEqualStrings(t, expected, strA_recText(&rec))
	}
}

// Zig: test "term51: Stream.execute: C1 forms of the charset finals configure nothing"
func TestStream_term51_Stream_execute_C1_forms_of_the_charset_finals_configure_nothing(t *testing.T) {
	// 0x81 and 0x82 map to ESC A and ESC B, which without an intermediate
	// are not valid charset designations.
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	zigTry(t, s.Execute(0x81))
	zigTry(t, s.Execute(0x82))
	// CSI, OSC, DCS, SOS, PM and APC introducers and ST are not handled
	// as ESC dispatches either.
	zigTry(t, s.Execute(0x9B))
	zigTry(t, s.Execute(0x9D))
	zigTry(t, s.Execute(0x90))
	zigTry(t, s.Execute(0x98))
	zigTry(t, s.Execute(0x9E))
	zigTry(t, s.Execute(0x9F))
	zigTry(t, s.Execute(0x9C))
	expectEqualStrings(t, "", strA_recText(&rec))
}

// ---------------------------------------------------------------------------
// Stream.handleCodepoint
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.handleCodepoint: codepoints 0..0xF are executed as C0 controls"
func TestStream_term51_Stream_handleCodepoint_codepoints_0_0xF_are_executed_as_C0_controls(t *testing.T) {
	expected := [16]string{
		"", // NUL
		"", // SOH
		"", // STX
		"", // ETX
		"", // EOT
		"enquiry\n",
		"", // ACK
		"bell\n",
		"backspace\n",
		"horizontal_tab 1\n",
		"linefeed\n",
		"linefeed\n",
		"linefeed\n",
		"carriage_return\n",
		"invoke_charset GL G1 locking=false\n",
		"invoke_charset GL G0 locking=false\n",
	}
	for c, e := range expected {
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		zigTry(t, s.handleCodepoint(uint32(c)))
		expectEqualStrings(t, e, strA_recText(&rec))
		expectEqual(t, 0, len(rec.cps))
		expectEqual(t, ParserStateGround, s.parser.state)
	}
}

// Zig: test "term51: Stream.handleCodepoint: codepoints above 0xF are printed, except ESC"
func TestStream_term51_Stream_handleCodepoint_codepoints_above_0xF_are_printed_except_ESC(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// 0x10..0x1A and 0x1C..0x1F are not executed here, even though the
	// parser table treats them as C0 controls: they are printed.
	for c := 0x10; c < 0x1B; c++ {
		zigTry(t, s.handleCodepoint(uint32(c)))
	}
	for c := 0x1C; c < 0x20; c++ {
		zigTry(t, s.handleCodepoint(uint32(c)))
	}
	expectEqual(t, 15, len(rec.cps))
	for i, got := range rec.cps[0:11] {
		want := 0x10 + i
		expectEqual(t, uint32(want), got)
	}
	for i, got := range rec.cps[11:15] {
		want := 0x1C + i
		expectEqual(t, uint32(want), got)
	}
	expectEqual(t, ParserStateGround, s.parser.state)

	strA_recReset(&rec)
	// Space, printable ASCII, DEL, C1 range as codepoints, and the extremes.
	printed := []uint32{0x20, 'a', '~', 0x7F, 0x80, 0x84, 0x9B, 0x9C, 0x9F, 0xA0, 0xFF, 0x100, 0xFFFD, 0x10FFFF, 0x1FFFFF}
	for _, c := range printed {
		zigTry(t, s.handleCodepoint(c))
	}
	expectEqualSlices(t, printed, rec.cps)
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.handleCodepoint: ESC enters the escape state and clears the parser"
func TestStream_term51_Stream_handleCodepoint_ESC_enters_the_escape_state_and_clears_the_parser(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// Leave stale values in the parser, as an aborted sequence would.
	s.parser.intermediates_idx = 2
	s.parser.params_idx = 5
	s.parser.param_acc = 77
	s.parser.param_acc_idx = 3
	s.parser.params_sep.set(2)

	zigTry(t, s.handleCodepoint(0x1B))
	expectEqual(t, ParserStateEscape, s.parser.state)
	expectEqual(t, uint8(0), s.parser.intermediates_idx)
	expectEqual(t, uint8(0), s.parser.params_idx)
	expectEqual(t, uint16(0), s.parser.param_acc)
	expectEqual(t, uint8(0), s.parser.param_acc_idx)
	expectEqual(t, 0, s.parser.params_sep.count())
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.handleCodepoint: 8-bit controls arriving as codepoints are printed"
func TestStream_term51_Stream_handleCodepoint_8_bit_controls_arriving_as_codepoints_are_printed(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// U+0080..U+009F (UTF-8 C2 80..C2 9F) are not interpreted as C1.
	for c := 0x80; c < 0xA0; c++ {
		zigTry(t, s.handleCodepoint(uint32(c)))
	}
	expectEqual(t, 32, len(rec.cps))
	expectEqual(t, ParserStateGround, s.parser.state)
	// Via the byte stream too.
	strA_recReset(&rec)
	zigTry(t, s.NextSlice([]uint8("\xC2\x9B\xC2\x84")))
	expectEqualSlices(t, []uint32{0x9B, 0x84}, rec.cps)
	expectEqual(t, ParserStateGround, s.parser.state)
}

// ---------------------------------------------------------------------------
// Stream.nextUtf8
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.nextUtf8: ASCII is printed and C0 controls are executed"
func TestStream_term51_Stream_nextUtf8_ASCII_is_printed_and_C0_controls_are_executed(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.nextUtf8('a'))
	zigTry(t, s.nextUtf8(' '))
	zigTry(t, s.nextUtf8('\n'))
	zigTry(t, s.nextUtf8(0x7F))
	zigTry(t, s.nextUtf8(0x10))
	zigTry(t, s.nextUtf8(0x07))
	expectEqualStrings(
		t,
		"print 61\nprint 20\nlinefeed\nprint 7f\nprint 10\nbell\n",
		strA_recText(&rec),
	)
	expectEqual(t, uint8(0), s.utf8decoder.state)
}

// Zig: test "term51: Stream.nextUtf8: multi-byte sequences are decoded across calls"
func TestStream_term51_Stream_nextUtf8_multi_byte_sequences_are_decoded_across_calls(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// 2 bytes: U+00E9
	zigTry(t, s.nextUtf8(0xC3))
	expect(t, s.utf8decoder.state != 0, "s.utf8decoder.state != 0")
	expectEqualStrings(t, "", strA_recText(&rec))
	zigTry(t, s.nextUtf8(0xA9))
	expectEqual(t, uint8(0), s.utf8decoder.state)
	// 3 bytes: U+20AC
	zigTry(t, s.nextUtf8(0xE2))
	zigTry(t, s.nextUtf8(0x82))
	expectEqual(t, 1, len(rec.cps))
	zigTry(t, s.nextUtf8(0xAC))
	// 4 bytes: U+1F600
	zigTry(t, s.nextUtf8(0xF0))
	zigTry(t, s.nextUtf8(0x9F))
	zigTry(t, s.nextUtf8(0x98))
	expectEqual(t, 2, len(rec.cps))
	zigTry(t, s.nextUtf8(0x80))
	expectEqualSlices(t, []uint32{0xE9, 0x20AC, 0x1F600}, rec.cps)
	expectEqual(t, uint8(0), s.utf8decoder.state)
}

// Zig: test "term51: Stream.nextUtf8: ESC enters the escape state and clears the parser"
func TestStream_term51_Stream_nextUtf8_ESC_enters_the_escape_state_and_clears_the_parser(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	s.parser.params_idx = 4
	s.parser.param_acc = 9
	zigTry(t, s.nextUtf8(0x1B))
	expectEqual(t, ParserStateEscape, s.parser.state)
	expectEqual(t, uint8(0), s.parser.params_idx)
	expectEqual(t, uint16(0), s.parser.param_acc)
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextUtf8: an invalid lead byte prints one replacement character"
func TestStream_term51_Stream_nextUtf8_an_invalid_lead_byte_prints_one_replacement_character(t *testing.T) {
	// Continuation bytes as leads, overlong leads, and bytes beyond F4.
	bad := []uint8{0x80, 0x9B, 0xBF, 0xC0, 0xC1, 0xF5, 0xF8, 0xFE, 0xFF}
	for _, b := range bad {
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		zigTry(t, s.nextUtf8(b))
		expectEqualSlices(t, []uint32{0xFFFD}, rec.cps)
		expectEqual(t, uint8(0), s.utf8decoder.state)
		expectEqual(t, ParserStateGround, s.parser.state)
	}
}

// Zig: test "term51: Stream.nextUtf8: a rejected byte is replayed after the replacement character"
func TestStream_term51_Stream_nextUtf8_a_rejected_byte_is_replayed_after_the_replacement_character(t *testing.T) {
	type Case struct {
		in   string
		want string
	}
	cases := []Case{
		// Truncated sequence followed by ASCII.
		{in: "\xE2a", want: "print fffd\nprint 61\n"},
		{in: "\xE2\x82a", want: "print fffd\nprint 61\n"},
		{in: "\xF0\x9F\x98a", want: "print fffd\nprint 61\n"},
		{in: "\xC3a", want: "print fffd\nprint 61\n"},
		// ... followed by a C0 control: the control still runs.
		{in: "\xE2\x07", want: "print fffd\nbell\n"},
		{in: "\xC3\n", want: "print fffd\nlinefeed\n"},
		// ... followed by another lead byte: a new sequence starts.
		{in: "\xE2\xC3\xA9", want: "print fffd\nprint e9\n"},
		{in: "\xC3\xE2\x82\xAC", want: "print fffd\nprint 20ac\n"},
		// Overlong, surrogate and out-of-range encodings: the prefix and each
		// byte that cannot continue it are replaced individually.
		{in: "\xE0\x80", want: "print fffd\nprint fffd\n"},
		{in: "\xC0\x80", want: "print fffd\nprint fffd\n"},
		{in: "\xED\xA0\x80", want: "print fffd\nprint fffd\nprint fffd\n"},
		{in: "\xF4\x90\x80\x80", want: "print fffd\nprint fffd\nprint fffd\nprint fffd\n"},
		{in: "\xF0\x80\x80\x80", want: "print fffd\nprint fffd\nprint fffd\nprint fffd\n"},
		// Stray continuation bytes.
		{in: "\x80\x80", want: "print fffd\nprint fffd\n"},
		{in: "a\x80b", want: "print 61\nprint fffd\nprint 62\n"},
	}
	for _, c := range cases {
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		for _, b := range []uint8(c.in) {
			zigTry(t, s.nextUtf8(b))
		}
		t.Logf("input %v", []uint8(c.in))
		expectEqualStrings(t, c.want, strA_recText(&rec))
		expectEqual(t, uint8(0), s.utf8decoder.state)
	}
}

// Zig: test "term51: Stream.nextUtf8: a truncated sequence followed by ESC starts an escape sequence"
func TestStream_term51_Stream_nextUtf8_a_truncated_sequence_followed_by_ESC_starts_an_escape_sequence(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	s.parser.params_idx = 3
	zigTry(t, s.nextUtf8(0xE2))
	zigTry(t, s.nextUtf8(0x82))
	expect(t, s.utf8decoder.state != 0, "s.utf8decoder.state != 0")
	// The ESC is not consumed by the decoder on its first try: a replacement
	// character is printed and the ESC is then handled as a control.
	zigTry(t, s.nextUtf8(0x1B))
	expectEqualStrings(t, "print fffd\n", strA_recText(&rec))
	expectEqual(t, uint8(0), s.utf8decoder.state)
	expectEqual(t, ParserStateEscape, s.parser.state)
	expectEqual(t, uint8(0), s.parser.params_idx)
}

// Zig: test "term51: Stream.nextUtf8: a raw 0x9B in the ground state is a stray byte, not CSI"
func TestStream_term51_Stream_nextUtf8_a_raw_0x9B_in_the_ground_state_is_a_stray_byte_not_CSI(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.nextUtf8(0x9B))
	expectEqualStrings(t, "print fffd\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)
}

// ---------------------------------------------------------------------------
// Stream.nextNonUtf8
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.nextNonUtf8: ESC [ is a fast path straight to csi_entry"
func TestStream_term51_Stream_nextNonUtf8_ESC_is_a_fast_path_straight_to_csi_entry(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	s.parser.state = ParserStateEscape
	zigTry(t, s.nextNonUtf8('['))
	expectEqual(t, ParserStateCsiEntry, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))

	// In any other state '[' goes through the state machine: here it is a
	// CSI final byte (0x5B) with no handler.
	s.parser.state = ParserStateCsiEntry
	zigTry(t, s.nextNonUtf8('['))
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))

	// After ESC and an intermediate it is an ESC final.
	s.parser.state = ParserStateEscapeIntermediate
	zigTry(t, s.nextNonUtf8('['))
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextNonUtf8: C0 controls inside csi_param are executed and the sequence continues"
func TestStream_term51_Stream_nextNonUtf8_C0_controls_inside_csi_param_are_executed_and_the_sequence_continues(t *testing.T) {
	for i := 0; i < 0x10; i++ {
		c := uint8(i)
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		strA_feed(t, &s, "\x1b[1")
		expectEqual(t, ParserStateCsiParam, s.parser.state)
		zigTry(t, s.nextNonUtf8(c))
		var expected string
		switch c {
		case 0x05:
			expected = "enquiry\n"
		case 0x07:
			expected = "bell\n"
		case 0x08:
			expected = "backspace\n"
		case 0x09:
			expected = "horizontal_tab 1\n"
		case 0x0A, 0x0B, 0x0C:
			expected = "linefeed\n"
		case 0x0D:
			expected = "carriage_return\n"
		case 0x0E:
			expected = "invoke_charset GL G1 locking=false\n"
		case 0x0F:
			expected = "invoke_charset GL G0 locking=false\n"
		default:
			expected = ""
		}
		expectEqualStrings(t, expected, strA_recText(&rec))
		expectEqual(t, ParserStateCsiParam, s.parser.state)
		// The parameter is intact.
		expectEqual(t, uint16(1), s.parser.param_acc)
	}

	// And the sequence still completes afterwards.
	strA_expectNext(t, "\x1b[1\n2;3\rH", "linefeed\ncarriage_return\ncursor_pos 12 3\n")
}

// Zig: test "term51: Stream.nextNonUtf8: other C0 controls inside csi_param are ignored"
func TestStream_term51_Stream_nextNonUtf8_other_C0_controls_inside_csi_param_are_ignored(t *testing.T) {
	ignored := []uint8{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x19, 0x1C, 0x1D, 0x1E, 0x1F, 0x7F}
	for _, c := range ignored {
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		strA_feed(t, &s, "\x1b[12")
		zigTry(t, s.nextNonUtf8(c))
		expectEqualStrings(t, "", strA_recText(&rec))
		expectEqual(t, ParserStateCsiParam, s.parser.state)
		expectEqual(t, uint16(12), s.parser.param_acc)
		expectEqual(t, uint8(0), s.parser.params_idx)
	}
	strA_expectNext(t, "\x1b[1\x10\x7F\x1f2A", "cursor_up 12\n")
}

// Zig: test "term51: Stream.nextNonUtf8: CAN and SUB abort a CSI sequence"
func TestStream_term51_Stream_nextNonUtf8_CAN_and_SUB_abort_a_CSI_sequence(t *testing.T) {
	for _, c := range []uint8{0x18, 0x1A} {
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		strA_feed(t, &s, "\x1b[12;3")
		zigTry(t, s.nextNonUtf8(c))
		expectEqual(t, ParserStateGround, s.parser.state)
		expectEqualStrings(t, "", strA_recText(&rec))
		// The final byte is then plain text.
		zigTry(t, s.Next('H'))
		expectEqualStrings(t, "print 48\n", strA_recText(&rec))
	}
}

// Zig: test "term51: Stream.nextNonUtf8: digits accumulate and saturate at 65535"
func TestStream_term51_Stream_nextNonUtf8_digits_accumulate_and_saturate_at_65535(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "\x1b[")
	expectEqual(t, uint8(0), s.parser.param_acc_idx)
	// The first digit goes through the state machine.
	zigTry(t, s.nextNonUtf8('6'))
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	expectEqual(t, uint16(6), s.parser.param_acc)
	expectEqual(t, uint8(1), s.parser.param_acc_idx)
	// The rest take the fast path.
	zigTry(t, s.nextNonUtf8('5'))
	expectEqual(t, uint16(65), s.parser.param_acc)
	zigTry(t, s.nextNonUtf8('5'))
	zigTry(t, s.nextNonUtf8('3'))
	expectEqual(t, uint16(6553), s.parser.param_acc)
	zigTry(t, s.nextNonUtf8('5'))
	expectEqual(t, uint16(65535), s.parser.param_acc)
	// One more digit saturates rather than wraps.
	zigTry(t, s.nextNonUtf8('9'))
	expectEqual(t, uint16(65535), s.parser.param_acc)
	zigTry(t, s.nextNonUtf8('0'))
	expectEqual(t, uint16(65535), s.parser.param_acc)
	// The "has a parameter" marker is non-zero.
	expect(t, s.parser.param_acc_idx != 0, "s.parser.param_acc_idx != 0")
	zigTry(t, s.nextNonUtf8('A'))
	expectEqualStrings(t, "cursor_up 65535\n", strA_recText(&rec))

	// 65536 saturates as well, it does not wrap to 0.
	strA_recReset(&rec)
	strA_feed(t, &s, "\x1b[65536A")
	expectEqualStrings(t, "cursor_up 65535\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextNonUtf8: separators store parameters in the fast path"
func TestStream_term51_Stream_nextNonUtf8_separators_store_parameters_in_the_fast_path(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "\x1b[1;2")
	expectEqual(t, uint8(1), s.parser.params_idx)
	expectEqual(t, uint16(1), s.parser.params[0])
	expectEqual(t, uint16(2), s.parser.param_acc)
	expectEqual(t, 0, s.parser.params_sep.count())

	// A colon records its position in the separator set.
	zigTry(t, s.nextNonUtf8(':'))
	expectEqual(t, uint8(2), s.parser.params_idx)
	expectEqual(t, uint16(2), s.parser.params[1])
	expectEqual(t, uint16(0), s.parser.param_acc)
	expectEqual(t, uint8(0), s.parser.param_acc_idx)
	expect(t, !s.parser.params_sep.isSet(0), "!s.parser.params_sep.isSet(0)")
	expect(t, s.parser.params_sep.isSet(1), "s.parser.params_sep.isSet(1)")
	expectEqual(t, 1, s.parser.params_sep.count())

	// An empty parameter is stored as 0.
	zigTry(t, s.nextNonUtf8(';'))
	expectEqual(t, uint8(3), s.parser.params_idx)
	expectEqual(t, uint16(0), s.parser.params[2])
	expect(t, !s.parser.params_sep.isSet(2), "!s.parser.params_sep.isSet(2)")

	// The next digit sets the marker again.
	zigTry(t, s.nextNonUtf8('7'))
	expectEqual(t, uint16(7), s.parser.param_acc)
	expect(t, s.parser.param_acc_idx != 0, "s.parser.param_acc_idx != 0")
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextNonUtf8: colons are only accepted by SGR"
func TestStream_term51_Stream_nextNonUtf8_colons_are_only_accepted_by_SGR(t *testing.T) {
	// Colon-separated parameters on any other final drop the sequence.
	strA_expectNext(t, "\x1b[1:2H", "")
	strA_expectNext(t, "\x1b[1;2:3H", "")
	strA_expectNext(t, "\x1b[1:2A", "")
	// ... and the parser is usable afterwards.
	strA_expectNext(t, "\x1b[1:2H\x1b[3;4H", "cursor_pos 3 4\n")
	// SGR keeps them: 4:3 is a curly underline.
	strA_expectNext(t, "\x1b[4:3m", "set_attribute underline\n")
}

// Zig: test "term51: Stream.nextNonUtf8: parameters beyond MAX_PARAMS are ignored"
func TestStream_term51_Stream_nextNonUtf8_parameters_beyond_MAX_PARAMS_are_ignored(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "\x1b[")
	// The 24 separators fill every slot.
	for i := 0; i < MAX_PARAMS; i++ {
		zigTry(t, s.nextNonUtf8(';'))
	}
	expectEqual(t, uint8(MAX_PARAMS), s.parser.params_idx)
	// More separators and digits are ignored by the fast path.
	zigTry(t, s.nextNonUtf8(';'))
	zigTry(t, s.nextNonUtf8(':'))
	zigTry(t, s.nextNonUtf8('5'))
	expectEqual(t, uint8(MAX_PARAMS), s.parser.params_idx)
	expectEqual(t, uint16(0), s.parser.param_acc)
	expectEqual(t, uint8(0), s.parser.param_acc_idx)
	expectEqual(t, 0, s.parser.params_sep.count())
	// A sequence that overflowed the parameter list is dropped.
	zigTry(t, s.nextNonUtf8('m'))
	expectEqualStrings(t, "", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.nextNonUtf8: exactly MAX_PARAMS parameters are accepted"
func TestStream_term51_Stream_nextNonUtf8_exactly_MAX_PARAMS_parameters_are_accepted(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	// 24 parameters of "1" (bold), separated by 23 semicolons.
	ok := "\x1b[" + strings.Repeat("1;", 23) + "1m"
	// Zig-only: defer alloc.free(ok);
	// Zig-only: var expected: std.ArrayList(u8) = .empty;
	// Zig-only: defer expected.deinit(alloc);
	expected := strings.Repeat("set_attribute bold\n", 24)
	strA_expectNext(t, ok, expected)
	strA_expectSlice(t, ok, expected)

	// One more parameter (25) is dropped entirely.
	too_many := "\x1b[" + strings.Repeat("1;", 24) + "1m"
	// Zig-only: defer alloc.free(too_many);
	strA_expectNext(t, too_many, "")
	strA_expectSlice(t, too_many, "")

	// With a trailing separator (24 separators, no final digit) also dropped.
	trailing := "\x1b[" + strings.Repeat("1;", 24) + "m"
	// Zig-only: defer alloc.free(trailing);
	strA_expectNext(t, trailing, "")
}

// Zig: test "term51: Stream.nextNonUtf8: bytes the fast path does not handle go through the parser"
func TestStream_term51_Stream_nextNonUtf8_bytes_the_fast_path_does_not_handle_go_through_the_parser(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// Intermediate: csi_param -> csi_intermediate.
	strA_feed(t, &s, "\x1b[1")
	zigTry(t, s.nextNonUtf8(' '))
	expectEqual(t, ParserStateCsiIntermediate, s.parser.state)
	expectEqual(t, uint8(1), s.parser.intermediates_idx)

	// '?' after a digit: csi_param -> csi_ignore, and the sequence is
	// swallowed up to its final byte.
	s.parser.state = ParserStateGround
	strA_feed(t, &s, "\x1b[1")
	zigTry(t, s.nextNonUtf8('?'))
	expectEqual(t, ParserStateCsiIgnore, s.parser.state)
	zigTry(t, s.nextNonUtf8('A'))
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))

	// ESC restarts: csi_param -> escape.
	strA_feed(t, &s, "\x1b[1")
	zigTry(t, s.nextNonUtf8(0x1B))
	expectEqual(t, ParserStateEscape, s.parser.state)
	expectEqual(t, uint16(0), s.parser.param_acc)

	// A byte >= 0x80 that is not a C1 control is ignored in place.
	s.parser.state = ParserStateGround
	strA_feed(t, &s, "\x1b[1")
	zigTry(t, s.nextNonUtf8(0xC3))
	zigTry(t, s.nextNonUtf8(0xA9))
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	zigTry(t, s.nextNonUtf8('A'))
	expectEqualStrings(t, "cursor_up 1\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextNonUtf8: C1 controls abort the current sequence"
func TestStream_term51_Stream_nextNonUtf8_C1_controls_abort_the_current_sequence(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// 0x84 (IND) in the middle of CSI params: executed, back to ground.
	strA_feed(t, &s, "\x1b[1")
	zigTry(t, s.nextNonUtf8(0x84))
	expectEqualStrings(t, "index\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)

	// 0x9B (CSI) and the other introducers switch states.
	s.parser.state = ParserStateEscape
	zigTry(t, s.nextNonUtf8(0x9B))
	expectEqual(t, ParserStateCsiEntry, s.parser.state)
	s.parser.state = ParserStateCsiParam
	zigTry(t, s.nextNonUtf8(0x9D))
	expectEqual(t, ParserStateOscString, s.parser.state)
	s.parser.state = ParserStateEscape
	zigTry(t, s.nextNonUtf8(0x90))
	expectEqual(t, ParserStateDcsEntry, s.parser.state)
	s.parser.state = ParserStateEscape
	zigTry(t, s.nextNonUtf8(0x98))
	expectEqual(t, ParserStateSosPmApcString, s.parser.state)
	s.parser.state = ParserStateEscape
	zigTry(t, s.nextNonUtf8(0x9E))
	expectEqual(t, ParserStateSosPmApcString, s.parser.state)
	s.parser.state = ParserStateEscape
	zigTry(t, s.nextNonUtf8(0x9F))
	expectEqual(t, ParserStateSosPmApcString, s.parser.state)
	// ST returns to ground without an action.
	s.parser.state = ParserStateEscape
	strA_recReset(&rec)
	zigTry(t, s.nextNonUtf8(0x9C))
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextNonUtf8: routes ESC, CSI and C0 actions"
func TestStream_term51_Stream_nextNonUtf8_routes_ESC_CSI_and_C0_actions(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// execute in the escape state, which stays in that state.
	s.parser.state = ParserStateEscape
	zigTry(t, s.nextNonUtf8('\n'))
	expectEqualStrings(t, "linefeed\n", strA_recText(&rec))
	expectEqual(t, ParserStateEscape, s.parser.state)
	// esc_dispatch.
	zigTry(t, s.nextNonUtf8('M'))
	expectEqualStrings(t, "linefeed\nreverse_index\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)

	// csi_dispatch.
	strA_recReset(&rec)
	strA_feed(t, &s, "\x1b[5A")
	expectEqualStrings(t, "cursor_up 5\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextNonUtf8: routes OSC dispatch"
func TestStream_term51_Stream_nextNonUtf8_routes_OSC_dispatch(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// Terminated by BEL.
	strA_feed(t, &s, "\x1b]0;one\x07")
	// Terminated by ST (ESC \): the dispatch happens when ESC leaves the
	// string, and the trailing backslash is a no-op ESC dispatch.
	strA_feed(t, &s, "\x1b]2;two\x1b\\")
	expectEqualStrings(t, "window_title one\nwindow_title two\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)

	// A raw 0x9C does not end an OSC string, it is part of the payload.
	strA_recReset(&rec)
	strA_feed(t, &s, "\x1b]2;a\x9c")
	expectEqual(t, ParserStateOscString, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))
	zigTry(t, s.Next(0x07))
	// The title contains the raw 0x9C byte, which is not valid UTF-8, so the
	// request is ignored.
	expectEqualStrings(t, "", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.nextNonUtf8: routes DCS and APC actions"
func TestStream_term51_Stream_nextNonUtf8_routes_DCS_and_APC_actions(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "\x1bP1;2$qm\x1b\\")
	expectEqualStrings(
		t,
		"dcs_hook inter=$ params=1,2 final=q\ndcs_put 6d\ndcs_unhook\n",
		strA_recText(&rec),
	)
	expectEqual(t, ParserStateGround, s.parser.state)

	strA_recReset(&rec)
	strA_feed(t, &s, "\x1b_Gab\x1b\\")
	expectEqualStrings(
		t,
		"apc_start\napc_put 47\napc_put 61\napc_put 62\napc_end\n",
		strA_recText(&rec),
	)
	expectEqual(t, ParserStateGround, s.parser.state)

	// SOS (ESC X) and PM (ESC ^) use the same APC actions.
	strA_recReset(&rec)
	strA_feed(t, &s, "\x1bXq\x1b\\\x1b^r\x1b\\")
	expectEqualStrings(
		t,
		"apc_start\napc_put 71\napc_end\napc_start\napc_put 72\napc_end\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.nextNonUtf8: vtRaw sees parser actions and can skip them"
func TestStream_term51_Stream_nextNonUtf8_vtRaw_sees_parser_actions_and_can_skip_them(t *testing.T) {
	// With a pass-through vtRaw, the stream still processes every action.
	{
		rec := strA_rawRecorderInit()
		// Zig-only: defer term51ARawRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		strA_feed(t, &s, "\x1b[2A\x1b7\x1b\n\x1b]0;t\x07")
		// The ESC and '[' bytes (UTF-8 path and the fast path), the digit
		// from csi_entry and bytes consumed by the csi_param fast path are
		// not parser actions at all.
		expectEqualStrings(
			t,
			"csi_dispatch A\nesc_dispatch 7\nexecute a\nosc_dispatch change_window_title\n",
			string(rec.raw),
		)
		expectEqualStrings(
			t,
			"cursor_up 2\nsave_cursor\nlinefeed\nwindow_title t\n",
			strA_recText(&rec.base),
		)
	}

	// When vtRaw returns true the action is not processed.
	{
		rec := strA_rawRecorderInit()
		rec.mode = strA_rawSkip
		// Zig-only: defer term51ARawRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		strA_feed(t, &s, "\x1b[2A\x1b7\x1b\n\x1b]0;t\x07\x1bc")
		expectEqualStrings(
			t,
			"csi_dispatch A\nesc_dispatch 7\nexecute a\nosc_dispatch change_window_title\nesc_dispatch c\n",
			string(rec.raw),
		)
		expectEqualStrings(t, "", strA_recText(&rec.base))
		expectEqual(t, ParserStateGround, s.parser.state)
	}

	// When vtRaw fails the action is skipped as well and the error is
	// swallowed.
	{
		rec := strA_rawRecorderInit()
		rec.mode = strA_rawFail
		// Zig-only: defer term51ARawRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		strA_feed(t, &s, "\x1b[2A\x1b7\x1b]0;t\x07")
		expectEqualStrings(
			t,
			"csi_dispatch A\nesc_dispatch 7\nosc_dispatch change_window_title\n",
			string(rec.raw),
		)
		expectEqualStrings(t, "", strA_recText(&rec.base))
	}
}

// Zig: test "term51: Stream.nextNonUtf8: vtRaw can skip individual kinds of action"
func TestStream_term51_Stream_nextNonUtf8_vtRaw_can_skip_individual_kinds_of_action(t *testing.T) {
	// Skip only DCS/APC actions: the other kinds pass.
	rec := strA_rawRecorderInit()
	rec.mode = strA_rawSkip
	rec.only = Some(ParserActionDcsPut)
	// Zig-only: defer term51ARawRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "\x1bPq\x41\x1b\\")
	expectEqualStrings(
		t,
		"dcs_hook q\ndcs_put 41\ndcs_unhook\nesc_dispatch \\\n",
		string(rec.raw),
	)
	// dcs_put was skipped, hook and unhook were not.
	expectEqualStrings(
		t,
		"dcs_hook inter= params= final=q\ndcs_unhook\n",
		strA_recText(&rec.base),
	)

	strA_recReset(&rec.base)
	// Zig-only: rec.raw.deinit();
	rec.raw = nil // Zig: rec.raw = .init(testing.allocator);
	rec.only = Some(ParserActionApcStart)
	strA_feed(t, &s, "\x1b_z\x1b\\")
	expectEqualStrings(
		t,
		"apc_start\napc_put 7a\napc_end\nesc_dispatch \\\n",
		string(rec.raw),
	)
	expectEqualStrings(t, "apc_put 7a\napc_end\n", strA_recText(&rec.base))
}

// Zig: test "term51: Stream.nextNonUtf8: vtRaw is not consulted for text or for ground-state controls"
func TestStream_term51_Stream_nextNonUtf8_vtRaw_is_not_consulted_for_text_or_for_ground_state_controls(t *testing.T) {
	rec := strA_rawRecorderInit()
	rec.mode = strA_rawSkip
	// Zig-only: defer term51ARawRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "ab\n\x07")
	zigTry(t, s.NextSlice([]uint8("cd\r")))
	expectEqualStrings(t, "", string(rec.raw))
	expectEqualStrings(
		t,
		"print 61\nprint 62\nlinefeed\nbell\nprint 63\nprint 64\ncarriage_return\n",
		strA_recText(&rec.base),
	)
}

// ---------------------------------------------------------------------------
// Stream.consumeUntilGround / Stream.consumeAllEscapes
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.consumeUntilGround: does nothing in the ground state"
func TestStream_term51_Stream_consumeUntilGround_does_nothing_in_the_ground_state(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	expectEqual(t, uint(0), strA_untilGround(t, &s, "hello\x1b[1A"))
	expectEqual(t, uint(0), strA_untilGround(t, &s, ""))
	expectEqualStrings(t, "", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.consumeUntilGround: stops right after the byte that reaches ground"
func TestStream_term51_Stream_consumeUntilGround_stops_right_after_the_byte_that_reaches_ground(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "\x1b[1;2")
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	expectEqual(t, uint(1), strA_untilGround(t, &s, "Hxyz"))
	expectEqualStrings(t, "cursor_pos 1 2\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)

	// A bare ESC final byte.
	strA_recReset(&rec)
	s.parser.state = ParserStateEscape
	expectEqual(t, uint(1), strA_untilGround(t, &s, "Mabc"))
	expectEqualStrings(t, "reverse_index\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.consumeUntilGround: consumes everything when the sequence is not finished"
func TestStream_term51_Stream_consumeUntilGround_consumes_everything_when_the_sequence_is_not_finished(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "\x1b[")
	expectEqual(t, uint(3), strA_untilGround(t, &s, "1;2"))
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))

	// With empty input it consumes nothing and leaves the state alone.
	expectEqual(t, uint(0), strA_untilGround(t, &s, ""))
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	// And the sequence can be finished later.
	expectEqual(t, uint(1), strA_untilGround(t, &s, "H"))
	expectEqualStrings(t, "cursor_pos 1 2\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.consumeUntilGround: an ESC inside a sequence restarts it"
func TestStream_term51_Stream_consumeUntilGround_an_ESC_inside_a_sequence_restarts_it(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "\x1b[9")
	// ESC [ 3 A: consumed as the restarted sequence: 4 bytes.
	expectEqual(t, uint(4), strA_untilGround(t, &s, "\x1b[3Azz"))
	expectEqualStrings(t, "cursor_up 3\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.consumeUntilGround: a C1 introducer inside an OSC string is payload"
func TestStream_term51_Stream_consumeUntilGround_a_C1_introducer_inside_an_OSC_string_is_payload(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_feed(t, &s, "\x1b]0;a")
	expectEqual(t, ParserStateOscString, s.parser.state)
	expectEqual(t, uint(3), strA_untilGround(t, &s, "bc\x07rest"))
	expectEqualStrings(t, "window_title abc\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.consumeAllEscapes: consumes one complete sequence"
func TestStream_term51_Stream_consumeAllEscapes_consumes_one_complete_sequence(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	expectEqual(t, uint(4), strA_allEscapes(t, &s, "\x1b[2A"))
	expectEqualStrings(t, "cursor_up 2\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)

	strA_recReset(&rec)
	expectEqual(t, uint(2), strA_allEscapes(t, &s, "\x1bM"))
	expectEqualStrings(t, "reverse_index\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.consumeAllEscapes: consumes back-to-back sequences and stops at text"
func TestStream_term51_Stream_consumeAllEscapes_consumes_back_to_back_sequences_and_stops_at_text(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	expectEqual(t, uint(10), strA_allEscapes(t, &s, "\x1b[2A\x1b[3B\x1bMtext"))
	expectEqualStrings(t, "cursor_up 2\ncursor_down 3\nreverse_index\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)

	// Only the escape sequence is consumed; text after it is left alone.
	strA_recReset(&rec)
	expectEqual(t, uint(4), strA_allEscapes(t, &s, "\x1b[2Ahello\x1b[3B"))
	expectEqualStrings(t, "cursor_up 2\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.consumeAllEscapes: input that is not an escape sequence consumes nothing"
func TestStream_term51_Stream_consumeAllEscapes_input_that_is_not_an_escape_sequence_consumes_nothing(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	expectEqual(t, uint(0), strA_allEscapes(t, &s, "abc"))
	expectEqual(t, uint(0), strA_allEscapes(t, &s, "\x00"))
	expectEqual(t, uint(0), strA_allEscapes(t, &s, "\xC3\xA9"))
	expectEqualStrings(t, "", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.consumeAllEscapes: an unfinished sequence consumes the whole input"
func TestStream_term51_Stream_consumeAllEscapes_an_unfinished_sequence_consumes_the_whole_input(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	expectEqual(t, uint(3), strA_allEscapes(t, &s, "\x1b[1"))
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))

	// A lone ESC.
	s.parser.state = ParserStateGround
	expectEqual(t, uint(1), strA_allEscapes(t, &s, "\x1b"))
	expectEqual(t, ParserStateEscape, s.parser.state)

	// A complete sequence followed by the start of another.
	s.parser.state = ParserStateGround
	expectEqual(t, uint(5), strA_allEscapes(t, &s, "\x1b[2A\x1b"))
	expectEqualStrings(t, "cursor_up 2\n", strA_recText(&rec))
	expectEqual(t, ParserStateEscape, s.parser.state)

	// A complete sequence followed by the middle of another.
	strA_recReset(&rec)
	s.parser.state = ParserStateGround
	expectEqual(t, uint(7), strA_allEscapes(t, &s, "\x1b[2A\x1b[3"))
	expectEqualStrings(t, "cursor_up 2\n", strA_recText(&rec))
	expectEqual(t, ParserStateCsiParam, s.parser.state)
}

// Zig: test "term51: Stream.consumeAllEscapes: every sequence starts from a cleared escape state"
func TestStream_term51_Stream_consumeAllEscapes_every_sequence_starts_from_a_cleared_escape_state(t *testing.T) {
	r := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&r);
	s := StreamInit(&r)
	defer s.Deinit()

	// Half a sequence with parameters is pending...
	strA_feed(t, &s, "\x1b[1;2")
	expectEqual(t, uint8(1), s.parser.params_idx)
	// ... but consumeAllEscapes starts a new one: ESC forces the escape state
	// and clears the old parameters.
	expectEqual(t, uint(4), strA_allEscapes(t, &s, "\x1b[3A"))
	expectEqualStrings(t, "cursor_up 3\n", strA_recText(&r))
}

// Zig: test "term51: Stream.consumeAllEscapes: ESC ESC is one restarted sequence"
func TestStream_term51_Stream_consumeAllEscapes_ESC_ESC_is_one_restarted_sequence(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	expectEqual(t, uint(3), strA_allEscapes(t, &s, "\x1b\x1bM"))
	expectEqualStrings(t, "reverse_index\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.consumeAllEscapes: OSC, DCS and APC strings are consumed whole"
func TestStream_term51_Stream_consumeAllEscapes_OSC_DCS_and_APC_strings_are_consumed_whole(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	expectEqual(t, uint(7), strA_allEscapes(t, &s, "\x1b]0;ab\x07z"))
	expectEqualStrings(t, "window_title ab\n", strA_recText(&rec))

	// An ST-terminated string: the ST's ESC dispatches the string and its
	// trailing backslash is a separate (no-op) escape sequence.
	strA_recReset(&rec)
	expectEqual(t, uint(8), strA_allEscapes(t, &s, "\x1b]0;ab\x1b\\z"))
	expectEqualStrings(t, "window_title ab\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)

	strA_recReset(&rec)
	expectEqual(t, uint(7), strA_allEscapes(t, &s, "\x1b_Gab\x1b\\"))
	expectEqualStrings(t, "apc_start\napc_put 47\napc_put 61\napc_put 62\napc_end\n", strA_recText(&rec))
}

// ---------------------------------------------------------------------------
// Stream.next
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.next: ground bytes go through the UTF-8 decoder"
func TestStream_term51_Stream_next_ground_bytes_go_through_the_UTF_8_decoder(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.Next('a'))
	zigTry(t, s.Next(0xE2))
	expect(t, s.utf8decoder.state != 0, "s.utf8decoder.state != 0")
	expectEqual(t, ParserStateGround, s.parser.state)
	zigTry(t, s.Next(0x82))
	zigTry(t, s.Next(0xAC))
	zigTry(t, s.Next('\n'))
	expectEqualStrings(t, "print 61\nprint 20ac\nlinefeed\n", strA_recText(&rec))
	expectEqual(t, uint8(0), s.utf8decoder.state)
}

// Zig: test "term51: Stream.next: ESC from the ground state switches to the parser"
func TestStream_term51_Stream_next_ESC_from_the_ground_state_switches_to_the_parser(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.Next(0x1B))
	expectEqual(t, ParserStateEscape, s.parser.state)
	// From now on bytes are parser input, not UTF-8: 0xC3 is ignored in the
	// escape state and the decoder never sees it.
	zigTry(t, s.Next(0xC3))
	expectEqual(t, uint8(0), s.utf8decoder.state)
	expectEqual(t, ParserStateEscape, s.parser.state)
	zigTry(t, s.Next('c'))
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqualStrings(t, "full_reset\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.next: bytes in a pending sequence are not text"
func TestStream_term51_Stream_next_bytes_in_a_pending_sequence_are_not_text(t *testing.T) {
	// 'A' after ESC [ is a CSI final, not a character.
	strA_expectNext(t, "\x1b[A", "cursor_up 1\n")
	// After the sequence ends text resumes.
	strA_expectNext(t, "\x1b[AA", "cursor_up 1\nprint 41\n")
	// A C1 byte in a pending sequence is a control, in the ground state a
	// stray UTF-8 byte.
	strA_expectNext(t, "\x1b\x84", "index\n")
	strA_expectNext(t, "\x84", "print fffd\n")
}

// Zig: test "term51: Stream.next: a truncated UTF-8 sequence does not swallow the next escape"
func TestStream_term51_Stream_next_a_truncated_UTF_8_sequence_does_not_swallow_the_next_escape(t *testing.T) {
	strA_expectNext(t, "\xE2\x82\x1b[2A", "print fffd\ncursor_up 2\n")
	strA_expectNext(t, "\xF0\x9F\x1b7", "print fffd\nsave_cursor\n")
}

// ---------------------------------------------------------------------------
// Stream.nextSlice
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.nextSlice: empty input is a no-op"
func TestStream_term51_Stream_nextSlice_empty_input_is_a_no_op(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("")))
	expectEqualStrings(t, "", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)

	// Also with a sequence pending.
	zigTry(t, s.NextSlice([]uint8("\x1b[1")))
	zigTry(t, s.NextSlice([]uint8("")))
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	zigTry(t, s.NextSlice([]uint8("A")))
	expectEqualStrings(t, "cursor_up 1\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextSlice: text, controls and escapes in one slice"
func TestStream_term51_Stream_nextSlice_text_controls_and_escapes_in_one_slice(t *testing.T) {
	strA_expectBoth(
		t,
		"ab\r\ncd\x1b[2Aef\x1b7\x07",
		"print 61\nprint 62\ncarriage_return\nlinefeed\nprint 63\nprint 64\ncursor_up 2\nprint 65\nprint 66\nsave_cursor\nbell\n",
	)
	strA_expectBoth(t, "héllo wörld €😀", "print 68\nprint e9\nprint 6c\nprint 6c\nprint 6f\nprint 20\nprint 77\nprint f6\nprint 72\nprint 6c\nprint 64\nprint 20\nprint 20ac\nprint 1f600\n")
}

// Zig: test "term51: Stream.nextSlice: state carries over between slices"
func TestStream_term51_Stream_nextSlice_state_carries_over_between_slices(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// A CSI split across three slices.
	zigTry(t, s.NextSlice([]uint8("x\x1b")))
	expectEqual(t, ParserStateEscape, s.parser.state)
	zigTry(t, s.NextSlice([]uint8("[1;")))
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	zigTry(t, s.NextSlice([]uint8("2Hy")))
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqualStrings(t, "print 78\ncursor_pos 1 2\nprint 79\n", strA_recText(&rec))

	// A UTF-8 sequence split across slices, one byte at a time.
	strA_recReset(&rec)
	zigTry(t, s.NextSlice([]uint8("\xF0")))
	zigTry(t, s.NextSlice([]uint8("\x9F")))
	zigTry(t, s.NextSlice([]uint8("\x98")))
	expectEqualStrings(t, "", strA_recText(&rec))
	zigTry(t, s.NextSlice([]uint8("\x80z")))
	expectEqualStrings(t, "print 1f600\nprint 7a\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextSlice: an OSC string split across slices"
func TestStream_term51_Stream_nextSlice_an_OSC_string_split_across_slices(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("\x1b]0;hel")))
	expectEqual(t, ParserStateOscString, s.parser.state)
	zigTry(t, s.NextSlice([]uint8("lo\x07!")))
	expectEqualStrings(t, "window_title hello\nprint 21\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextSlice: slices and single bytes give the same actions"
func TestStream_term51_Stream_nextSlice_slices_and_single_bytes_give_the_same_actions(t *testing.T) {
	inputs := []string{
		"plain text",
		"a\tb\x08c\x0bd\x0ce\x0ef\x0fg",
		"\x1b[1;31mred\x1b[0m",
		"x\x1b(0lqk\x1b(B",
		"\xE2\x82\xAC\xF0\x9F\x98\x80\xC3\xA9",
		"bad \xFF\xC3( \xE2\x82( ok",
		"\xC2\x9B\x9B\x84",
		"\x1b]0;t\x07\x1b]8;;u\x1b\\link\x1b]8;;\x1b\\",
		"\x1bP1$qm\x1b\\\x1b_Gx\x1b\\",
		"esc at end \x1b",
		"csi at end \x1b[1;",
		"\x1b[1\n\x1b[2A\x18done\x1a",
		"\x1b\x1b\x1bM\x1b7\x1b8",
	}
	for _, input := range inputs {
		t.Logf("input %v", []uint8(input))

		rec1 := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec1);
		s1 := StreamInit(&rec1)
		defer s1.Deinit()
		strA_feed(t, &s1, input)

		rec2 := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec2);
		s2 := StreamInit(&rec2)
		defer s2.Deinit()
		zigTry(t, s2.NextSlice([]uint8(input)))

		rec3 := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec3);
		s3 := StreamInit(&rec3)
		defer s3.Deinit()
		strA_feedChunked(t, &s3, []uint8(input))

		expectEqualStrings(t, strA_recText(&rec1), strA_recText(&rec2))
		expectEqualStrings(t, strA_recText(&rec1), strA_recText(&rec3))
		expectEqual(t, s1.parser.state, s2.parser.state)
		expectEqual(t, s1.parser.state, s3.parser.state)
		expectEqual(t, s1.utf8decoder.state, s2.utf8decoder.state)
		expectEqual(t, s1.utf8decoder.state, s3.utf8decoder.state)
	}
}

// ---------------------------------------------------------------------------
// Stream.nextSliceCapped and the 4096-codepoint chunk boundary of nextSlice
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.nextSliceCapped: text and C0 controls in the ground state"
func TestStream_term51_Stream_nextSliceCapped_text_and_C0_controls_in_the_ground_state(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [64]uint32

	// Codepoints <= 0xF are executed, everything else is printed.
	strA_capped(t, &s, "a\x00\x01\x02\x05\x07\x08\t\n\x0b\x0c\r\x0e\x0fb", buf[:])
	expectEqualStrings(
		t,
		"print 61\nenquiry\nbell\nbackspace\nhorizontal_tab 1\n"+
			"linefeed\nlinefeed\nlinefeed\ncarriage_return\n"+
			"invoke_charset GL G1 locking=false\n"+
			"invoke_charset GL G0 locking=false\nprint 62\n",
		strA_recText(&rec),
	)
	expectEqual(t, ParserStateGround, s.parser.state)

	// 0x10..0x1A, 0x1C..0x1F and DEL are printed as codepoints.
	strA_recReset(&rec)
	strA_capped(t, &s, "\x10\x17\x18\x19\x1a\x1c\x1f\x7f", buf[:])
	expectEqualSlices(t, []uint32{0x10, 0x17, 0x18, 0x19, 0x1A, 0x1C, 0x1F, 0x7F}, rec.cps)

	// Multi-byte text.
	strA_recReset(&rec)
	strA_capped(t, &s, "\xC3\xA9\xE2\x82\xAC\xF0\x9F\x98\x80", buf[:])
	expectEqualSlices(t, []uint32{0xE9, 0x20AC, 0x1F600}, rec.cps)
}

// Zig: test "term51: Stream.nextSliceCapped: the codepoint buffer may be exactly as long as the input"
func TestStream_term51_Stream_nextSliceCapped_the_codepoint_buffer_may_be_exactly_as_long_as_the_input(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [8]uint32

	strA_capped(t, &s, "abcdefgh", buf[:])
	expectEqualStrings(
		t,
		"print 61\nprint 62\nprint 63\nprint 64\nprint 65\nprint 66\nprint 67\nprint 68\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.nextSliceCapped: empty input does nothing"
func TestStream_term51_Stream_nextSliceCapped_empty_input_does_nothing(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [8]uint32

	strA_capped(t, &s, "", buf[:])
	expectEqualStrings(t, "", strA_recText(&rec))

	// With a pending UTF-8 sequence.
	zigTry(t, s.Next(0xE2))
	strA_capped(t, &s, "", buf[:])
	expect(t, s.utf8decoder.state != 0, "s.utf8decoder.state != 0")

	// With a pending escape sequence.
	s.utf8decoder = UTF8Decoder{}
	zigTry(t, s.Next(0x1B))
	strA_capped(t, &s, "", buf[:])
	expectEqual(t, ParserStateEscape, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextSliceCapped: escape sequences between text"
func TestStream_term51_Stream_nextSliceCapped_escape_sequences_between_text(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [64]uint32

	strA_capped(t, &s, "a\x1b[1Ab\x1b[2B\x1b7c\x1b[3;4Hd", buf[:])
	expectEqualStrings(
		t,
		"print 61\ncursor_up 1\nprint 62\ncursor_down 2\nsave_cursor\nprint 63\ncursor_pos 3 4\nprint 64\n",
		strA_recText(&rec),
	)
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.nextSliceCapped: input that starts with an escape sequence"
func TestStream_term51_Stream_nextSliceCapped_input_that_starts_with_an_escape_sequence(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [64]uint32

	strA_capped(t, &s, "\x1b[1A\x1b[2Bxy", buf[:])
	expectEqualStrings(t, "cursor_up 1\ncursor_down 2\nprint 78\nprint 79\n", strA_recText(&rec))

	// Back-to-back escapes ending the input.
	strA_recReset(&rec)
	strA_capped(t, &s, "\x1bM\x1bD", buf[:])
	expectEqualStrings(t, "reverse_index\nindex\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)
}

// Zig: test "term51: Stream.nextSliceCapped: continues a CSI sequence from the previous call"
func TestStream_term51_Stream_nextSliceCapped_continues_a_CSI_sequence_from_the_previous_call(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [64]uint32

	strA_capped(t, &s, "x\x1b[1", buf[:])
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	expectEqualStrings(t, "print 78\n", strA_recText(&rec))

	// The rest of the sequence and then text.
	strA_capped(t, &s, ";2Hyz", buf[:])
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqualStrings(
		t,
		"print 78\ncursor_pos 1 2\nprint 79\nprint 7a\n",
		strA_recText(&rec),
	)

	// The rest of the sequence is still not complete.
	strA_recReset(&rec)
	strA_capped(t, &s, "\x1b[3", buf[:])
	strA_capped(t, &s, ";", buf[:])
	expectEqual(t, ParserStateCsiParam, s.parser.state)
	expectEqualStrings(t, "", strA_recText(&rec))
	strA_capped(t, &s, "4H", buf[:])
	expectEqualStrings(t, "cursor_pos 3 4\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextSliceCapped: a lone ESC at the end of the input"
func TestStream_term51_Stream_nextSliceCapped_a_lone_ESC_at_the_end_of_the_input(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [64]uint32

	strA_capped(t, &s, "x\x1b", buf[:])
	expectEqual(t, ParserStateEscape, s.parser.state)
	expectEqualStrings(t, "print 78\n", strA_recText(&rec))
	strA_capped(t, &s, "[2A", buf[:])
	expectEqual(t, ParserStateGround, s.parser.state)
	expectEqualStrings(t, "print 78\ncursor_up 2\n", strA_recText(&rec))

	// The escape state, then ESC again: the sequence restarts.
	strA_recReset(&rec)
	strA_capped(t, &s, "\x1b", buf[:])
	strA_capped(t, &s, "\x1bM", buf[:])
	expectEqualStrings(t, "reverse_index\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextSliceCapped: a UTF-8 sequence cut at the end of the input is finished by the next call"
func TestStream_term51_Stream_nextSliceCapped_a_UTF_8_sequence_cut_at_the_end_of_the_input_is_finished_by_the_next_call(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [64]uint32

	// 2-byte lead only.
	strA_capped(t, &s, "ab\xC3", buf[:])
	expectEqualSlices(t, []uint32{'a', 'b'}, rec.cps)
	expect(t, s.utf8decoder.state != 0, "s.utf8decoder.state != 0")
	strA_capped(t, &s, "\xA9", buf[:])
	expectEqualSlices(t, []uint32{'a', 'b', 0xE9}, rec.cps)
	expectEqual(t, uint8(0), s.utf8decoder.state)

	// 3-byte, cut after 1 and after 2 bytes.
	strA_recReset(&rec)
	strA_capped(t, &s, "\xE2", buf[:])
	expect(t, s.utf8decoder.state != 0, "s.utf8decoder.state != 0")
	strA_capped(t, &s, "\x82", buf[:])
	expect(t, s.utf8decoder.state != 0, "s.utf8decoder.state != 0")
	expectEqual(t, 0, len(rec.cps))
	strA_capped(t, &s, "\xACz", buf[:])
	expectEqualSlices(t, []uint32{0x20AC, 'z'}, rec.cps)

	// 4-byte, cut after 1, 2 and 3 bytes.
	strA_recReset(&rec)
	strA_capped(t, &s, "x\xF0", buf[:])
	strA_capped(t, &s, "\x9F", buf[:])
	strA_capped(t, &s, "\x98", buf[:])
	expectEqualSlices(t, []uint32{'x'}, rec.cps)
	strA_capped(t, &s, "\x80", buf[:])
	expectEqualSlices(t, []uint32{'x', 0x1F600}, rec.cps)

	// A call that supplies several pieces of a pending sequence.
	strA_recReset(&rec)
	strA_capped(t, &s, "\xF0\x9F", buf[:])
	strA_capped(t, &s, "\x98\x80\xC3\xA9!", buf[:])
	expectEqualSlices(t, []uint32{0x1F600, 0xE9, '!'}, rec.cps)
}

// Zig: test "term51: Stream.nextSliceCapped: a pending UTF-8 sequence broken by the next call"
func TestStream_term51_Stream_nextSliceCapped_a_pending_UTF_8_sequence_broken_by_the_next_call(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [64]uint32

	// Followed by ASCII: replacement, then the text goes on normally.
	zigTry(t, s.Next(0xE2))
	strA_capped(t, &s, "abc", buf[:])
	expectEqualStrings(t, "print fffd\nprint 61\nprint 62\nprint 63\n", strA_recText(&rec))
	expectEqual(t, uint8(0), s.utf8decoder.state)

	// Followed by a control.
	strA_recReset(&rec)
	zigTry(t, s.Next(0xE2))
	zigTry(t, s.Next(0x82))
	strA_capped(t, &s, "\nz", buf[:])
	expectEqualStrings(t, "print fffd\nlinefeed\nprint 7a\n", strA_recText(&rec))

	// Followed by an escape sequence: the ESC is not lost.
	strA_recReset(&rec)
	zigTry(t, s.Next(0xE2))
	strA_capped(t, &s, "\x1b[3Ax", buf[:])
	expectEqualStrings(t, "print fffd\ncursor_up 3\nprint 78\n", strA_recText(&rec))
	expectEqual(t, ParserStateGround, s.parser.state)

	// ... including when the ESC is the last byte.
	strA_recReset(&rec)
	zigTry(t, s.Next(0xF0))
	strA_capped(t, &s, "\x1b", buf[:])
	expectEqualStrings(t, "print fffd\n", strA_recText(&rec))
	expectEqual(t, ParserStateEscape, s.parser.state)
	expectEqual(t, uint8(0), s.utf8decoder.state)
}

// Zig: test "term51: Stream.nextSliceCapped: an invalid lead byte is replaced and the rest of the input still runs"
func TestStream_term51_Stream_nextSliceCapped_an_invalid_lead_byte_is_replaced_and_the_rest_of_the_input_still_runs(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [64]uint32

	strA_capped(t, &s, "a\xFFb\n", buf[:])
	expectEqualStrings(t, "print 61\nprint fffd\nprint 62\nlinefeed\n", strA_recText(&rec))

	// An invalid lead byte on its own at the end.
	strA_recReset(&rec)
	strA_capped(t, &s, "a\xFF", buf[:])
	expectEqualStrings(t, "print 61\nprint fffd\n", strA_recText(&rec))
	expectEqual(t, uint8(0), s.utf8decoder.state)
}

// Zig: test "term51: Stream.nextSliceCapped: leaves a pending escape sequence in its state"
func TestStream_term51_Stream_nextSliceCapped_leaves_a_pending_escape_sequence_in_its_state(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()
	var buf [64]uint32

	strA_capped(t, &s, "ab\x1b]0;ti", buf[:])
	expectEqual(t, ParserStateOscString, s.parser.state)
	expectEqualStrings(t, "print 61\nprint 62\n", strA_recText(&rec))
	strA_capped(t, &s, "tle\x07cd", buf[:])
	expectEqualStrings(t, "print 61\nprint 62\nwindow_title title\nprint 63\nprint 64\n", strA_recText(&rec))
}

// Zig: test "term51: Stream.nextSlice: chunk boundary with 4095, 4096, 4097, 8192 and 8193 single-byte codepoints"
func TestStream_term51_Stream_nextSlice_chunk_boundary_with_4095_4096_4097_8192_and_8193_single_byte_codepoints(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	for _, n := range []int{1, 2, 4094, 4095, 4096, 4097, 4098, 8191, 8192, 8193, 8194, 12288, 12289} {
		input := strA_fill('a', n)
		// Zig-only: defer alloc.free(input);
		cps := strA_repeatCp('a', n)
		// Zig-only: defer alloc.free(cps);
		exp_log := fmt.Sprintf("prints %d\n", n)
		// Zig-only: defer alloc.free(exp_log);
		t.Logf("n=%d", n)
		strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log})
	}
}

// Zig: test "term51: Stream.nextSlice: chunk boundary with multi-byte codepoints"
func TestStream_term51_Stream_nextSlice_chunk_boundary_with_multi_byte_codepoints(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	chars := []struct {
		bytes string
		cp    uint32
	}{
		{"\xC3\xA9", 0xE9},
		{"\xE2\x82\xAC", 0x20AC},
		{"\xF0\x9F\x98\x80", 0x1F600},
	}
	for _, ch := range chars {
		// n codepoints: the byte count crosses 4096 / 8192 at all
		// alignments of the sequence length.
		for _, n := range []int{1365, 1366, 2047, 2048, 2049, 4095, 4096, 4097, 8192, 8193} {
			// Zig-only: var input: std.ArrayList(u8) = .empty;
			// Zig-only: defer input.deinit(alloc);
			input := []uint8(strings.Repeat(ch.bytes, n))
			cps := strA_repeatCp(ch.cp, n)
			// Zig-only: defer alloc.free(cps);
			exp_log := fmt.Sprintf("prints %d\n", n)
			// Zig-only: defer alloc.free(exp_log);
			t.Logf("len=%d n=%d", len(ch.bytes), n)
			strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log})
		}
	}
}

// Zig: test "term51: Stream.nextSlice: chunk boundary with control characters"
func TestStream_term51_Stream_nextSlice_chunk_boundary_with_control_characters(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	for _, n := range []int{2047, 2048, 2049, 4095, 4096, 4097} {
		// Zig-only: var input: std.ArrayList(u8) = .empty;
		// Zig-only: defer input.deinit(alloc);
		// Zig-only: var exp_log: std.ArrayList(u8) = .empty;
		// Zig-only: defer exp_log.deinit(alloc);
		var input []uint8
		var exp_log strings.Builder
		for i := 0; i < n; i++ {
			input = append(input, "a\n"...)
			exp_log.WriteString("prints 1\nlinefeed\n")
		}
		cps := strA_repeatCp('a', n)
		// Zig-only: defer alloc.free(cps);
		t.Logf("n=%d", n)
		strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log.String()})
	}
}

// Zig: test "term51: Stream.nextSlice: a CSI sequence split at every position around the chunk boundary"
func TestStream_term51_Stream_nextSlice_a_CSI_sequence_split_at_every_position_around_the_chunk_boundary(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	// The sequence is 8 bytes long; k filler bytes before it put the 4096
	// boundary before the ESC, after each of its bytes, and after its end.
	for k := 4086; k <= 4100; k++ {
		filler := strA_fill('x', k)
		// Zig-only: defer alloc.free(filler);
		input := append(append([]uint8{}, filler...), "\x1b[12;34Hz"...)
		// Zig-only: defer alloc.free(input);
		cps := strA_repeatCp('x', k+1)
		// Zig-only: defer alloc.free(cps);
		cps[k] = 'z'
		exp_log := fmt.Sprintf("prints %d\ncursor_pos 12 34\nprints 1\n", k)
		// Zig-only: defer alloc.free(exp_log);
		t.Logf("k=%d", k)
		strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log})
	}
}

// Zig: test "term51: Stream.nextSlice: a two-chunk escape sequence at the second boundary"
func TestStream_term51_Stream_nextSlice_a_two_chunk_escape_sequence_at_the_second_boundary(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	for k := 8184; k <= 8196; k++ {
		filler := strA_fill('x', k)
		// Zig-only: defer alloc.free(filler);
		input := append(append([]uint8{}, filler...), "\x1b[5A\x1b[6B"...)
		// Zig-only: defer alloc.free(input);
		cps := strA_repeatCp('x', k)
		// Zig-only: defer alloc.free(cps);
		exp_log := fmt.Sprintf("prints %d\ncursor_up 5\ncursor_down 6\n", k)
		// Zig-only: defer alloc.free(exp_log);
		t.Logf("k=%d", k)
		strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log})
	}
}

// Zig: test "term51: Stream.nextSlice: an unfinished sequence at the end of a full chunk"
func TestStream_term51_Stream_nextSlice_an_unfinished_sequence_at_the_end_of_a_full_chunk(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	filler := strA_fill('x', 4096-5)
	// Zig-only: defer alloc.free(filler);

	// The chunk ends in the middle of a CSI.
	{
		input := append(append([]uint8{}, filler...), "\x1b[12;"...)
		// Zig-only: defer alloc.free(input);
		expectEqual(t, 4096, len(input))
		cps := strA_repeatCp('x', len(filler))
		// Zig-only: defer alloc.free(cps);
		exp_log := fmt.Sprintf("prints %d\n", len(filler))
		// Zig-only: defer alloc.free(exp_log);
		strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log, parser_state: ParserStateCsiParam})
	}
	// The chunk ends with a lone ESC.
	{
		input := append(append([]uint8{}, filler...), "abcd\x1b"...)
		// Zig-only: defer alloc.free(input);
		expectEqual(t, 4096, len(input))
		cps := strA_repeatCp('x', len(filler)+4)
		// Zig-only: defer alloc.free(cps);
		copy(cps[len(filler):], []uint32{'a', 'b', 'c', 'd'})
		exp_log := fmt.Sprintf("prints %d\n", len(filler)+4)
		// Zig-only: defer alloc.free(exp_log);
		strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log, parser_state: ParserStateEscape})
	}
	// The chunk ends inside an OSC string.
	{
		input := append(append([]uint8{}, filler...), "\x1b]0;a"...)
		// Zig-only: defer alloc.free(input);
		expectEqual(t, 4096, len(input))
		cps := strA_repeatCp('x', len(filler))
		// Zig-only: defer alloc.free(cps);
		exp_log := fmt.Sprintf("prints %d\n", len(filler))
		// Zig-only: defer alloc.free(exp_log);
		strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log, parser_state: ParserStateOscString})
	}
}

// Zig: test "term51: Stream.nextSlice: a UTF-8 sequence split at every position around the chunk boundary"
func TestStream_term51_Stream_nextSlice_a_UTF_8_sequence_split_at_every_position_around_the_chunk_boundary(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	chars := []struct {
		bytes string
		cp    uint32
	}{
		{"\xC3\xA9", 0xE9},
		{"\xE2\x82\xAC", 0x20AC},
		{"\xF0\x9F\x98\x80", 0x1F600},
	}
	for _, ch := range chars {
		for k := 4090; k <= 4097; k++ {
			filler := strA_fill('x', k)
			// Zig-only: defer alloc.free(filler);
			input := append(append([]uint8{}, filler...), (ch.bytes + "yz")...)
			// Zig-only: defer alloc.free(input);
			cps := strA_repeatCp('x', k+3)
			// Zig-only: defer alloc.free(cps);
			cps[k] = ch.cp
			cps[k+1] = 'y'
			cps[k+2] = 'z'
			exp_log := fmt.Sprintf("prints %d\n", k+3)
			// Zig-only: defer alloc.free(exp_log);
			t.Logf("len=%d k=%d", len(ch.bytes), k)
			strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log})
		}
	}
}

// Zig: test "term51: Stream.nextSlice: input that ends in the middle of a UTF-8 sequence after a full chunk"
func TestStream_term51_Stream_nextSlice_input_that_ends_in_the_middle_of_a_UTF_8_sequence_after_a_full_chunk(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	partials := []string{"\xC3", "\xE2", "\xE2\x82", "\xF0", "\xF0\x9F", "\xF0\x9F\x98"}
	for _, partial := range partials {
		for _, k := range []int{4093, 4094, 4095, 4096, 4097, 8190, 8192} {
			filler := strA_fill('x', k)
			// Zig-only: defer alloc.free(filler);
			input := append(append([]uint8{}, filler...), partial...)
			// Zig-only: defer alloc.free(input);
			cps := strA_repeatCp('x', k)
			// Zig-only: defer alloc.free(cps);
			exp_log := fmt.Sprintf("prints %d\n", k)
			// Zig-only: defer alloc.free(exp_log);
			t.Logf("partial=%v k=%d", []uint8(partial), k)
			strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log, decoder_pending: true})
		}
	}
}

// Zig: test "term51: Stream.nextSlice: an OSC string split at the chunk boundary"
func TestStream_term51_Stream_nextSlice_an_OSC_string_split_at_the_chunk_boundary(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	for k := 4086; k <= 4100; k++ {
		filler := strA_fill('x', k)
		// Zig-only: defer alloc.free(filler);
		input := append(append([]uint8{}, filler...), "\x1b]0;title\x07!"...)
		// Zig-only: defer alloc.free(input);
		cps := strA_repeatCp('x', k+1)
		// Zig-only: defer alloc.free(cps);
		cps[k] = '!'
		exp_log := fmt.Sprintf("prints %d\nwindow_title title\nprints 1\n", k)
		// Zig-only: defer alloc.free(exp_log);
		t.Logf("k=%d", k)
		strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log})
	}
}

// Zig: test "term51: Stream.nextSlice: many escape sequences across several chunks"
func TestStream_term51_Stream_nextSlice_many_escape_sequences_across_several_chunks(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	// 5-byte units so that the chunk boundaries fall at every alignment.
	for _, n := range []int{818, 819, 820, 1638, 1639, 1640, 2000} {
		// Zig-only: var input: std.ArrayList(u8) = .empty;
		// Zig-only: defer input.deinit(alloc);
		// Zig-only: var exp_log: std.ArrayList(u8) = .empty;
		// Zig-only: defer exp_log.deinit(alloc);
		var input []uint8
		var exp_log strings.Builder
		for i := 0; i < n; i++ {
			input = append(input, "x\x1b[1A"...)
			exp_log.WriteString("prints 1\ncursor_up 1\n")
		}
		cps := strA_repeatCp('x', n)
		// Zig-only: defer alloc.free(cps);
		t.Logf("n=%d", n)
		strA_checkAllWays(t, input, strA_expectSpec{cps: cps, log: exp_log.String()})
	}
}

// Zig: test "term51: Stream.nextSlice: a large OSC 52 payload spans several chunks with an allocator"
func TestStream_term51_Stream_nextSlice_a_large_OSC_52_payload_spans_several_chunks_with_an_allocator(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	payload := string(strA_fill('Q', 9000))
	// Zig-only: defer alloc.free(payload);
	input := []uint8("ab\x1b]52;p;" + payload + "\x07cd")
	// Zig-only: defer alloc.free(input);
	expected := "print 61\nprint 62\nclipboard_contents kind=112 data=" + payload + "\nprint 63\nprint 64\n"
	// Zig-only: defer alloc.free(expected);

	// nextSlice
	{
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInitAlloc(&rec)
		defer s.Deinit()
		zigTry(t, s.NextSlice(input))
		expectEqualStrings(t, expected, strA_recText(&rec))
		expectEqual(t, ParserStateGround, s.parser.state)
	}
	// 4096-byte chunks
	{
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInitAlloc(&rec)
		defer s.Deinit()
		var cp_buf [4096]uint32
		i := 0
		for {
			len_ := min(len(cp_buf), len(input)-i)
			zigTry(t, s.nextSliceCapped(input[i:i+len_], cp_buf[:]))
			i += len_
			if i >= len(input) {
				break
			}
		}
		expectEqualStrings(t, expected, strA_recText(&rec))
	}
	// And it is cut off at the end of the input if it is not terminated, with
	// the allocation released by deinit.
	{
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInitAlloc(&rec)
		defer s.Deinit()
		zigTry(t, s.NextSlice(input[0:len(input)-3]))
		expectEqualStrings(t, "print 61\nprint 62\n", strA_recText(&rec))
		expectEqual(t, ParserStateOscString, s.parser.state)
	}
}

// Zig: test "term51: Stream.nextSlice: a slice larger than one chunk is the same as many small slices"
func TestStream_term51_Stream_nextSlice_a_slice_larger_than_one_chunk_is_the_same_as_many_small_slices(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	// Zig-only: var input: std.ArrayList(u8) = .empty;
	// Zig-only: defer input.deinit(alloc);
	var input []uint8
	i := 0
	for len(input) < 20_000 {
		input = append(input, "word \xE2\x82\xAC\xF0\x9F\x98\x80 \x1b[1;3"...)
		input = append(input, uint8('0'+(i%10)))
		input = append(input, "H\r\n\x1b7"...)
		i += 1
	}

	rec1 := strA_recorderInit(false)
	// Zig-only: defer term51ARecRelease(&rec1);
	s1 := StreamInit(&rec1)
	defer s1.Deinit()
	zigTry(t, s1.NextSlice(input))

	rec2 := strA_recorderInit(false)
	// Zig-only: defer term51ARecRelease(&rec2);
	s2 := StreamInit(&rec2)
	defer s2.Deinit()
	// Slices of 7 bytes, so every sequence is split.
	off := 0
	for off < len(input) {
		zigTry(t, s2.NextSlice(input[off:min(off+7, len(input))]))
		off += 7
	}
	expectEqualStrings(t, strA_recText(&rec1), strA_recText(&rec2))
	expectEqualSlices(t, rec1.cps, rec2.cps)
	expect(t, len(rec1.cps) > 4096, "rec1.cps.items.len > 4096")
	expectEqual(t, s1.parser.state, s2.parser.state)
}

// Zig: test "term51: Stream.nextSlice: matches the scalar path for every two-byte split of a mixed input"
func TestStream_term51_Stream_nextSlice_matches_the_scalar_path_for_every_two_byte_split_of_a_mixed_input(t *testing.T) {
	input := "a\xC3\xA9\x1b[1;2H\xE2\x82\xAC\x1b]0;t\x07\xF0\x9F\x98\x80\r\n\x1b(0q\x1b(B"
	rec0 := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec0);
	s0 := StreamInit(&rec0)
	defer s0.Deinit()
	strA_feed(t, &s0, input)

	for cut := 0; cut < len(input)+1; cut++ {
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		zigTry(t, s.NextSlice([]uint8(input[0:cut])))
		zigTry(t, s.NextSlice([]uint8(input[cut:])))
		expectEqualStrings(t, strA_recText(&rec0), strA_recText(&rec))

		strA_recReset(&rec)
		s.utf8decoder = UTF8Decoder{}
		s.parser = ParserInit()
		var cp_buf [4096]uint32
		strA_capped(t, &s, input[0:cut], cp_buf[:])
		strA_capped(t, &s, input[cut:], cp_buf[:])
		expectEqualStrings(t, strA_recText(&rec0), strA_recText(&rec))
	}
}

// ---------------------------------------------------------------------------
// Stream.configureCharset
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.configureCharset: each intermediate selects its slot"
func TestStream_term51_Stream_configureCharset_each_intermediate_selects_its_slot(t *testing.T) {
	slots := []struct {
		first  uint8
		second string
	}{
		{'(', "G0"},
		{')', "G1"},
		{'*', "G2"},
		{'+', "G3"},
	}
	sets := []struct {
		first  CharsetsCharset
		second string
	}{
		{CharsetsCharsetUtf8, "utf8"},
		{CharsetsCharsetAscii, "ascii"},
		{CharsetsCharsetBritish, "british"},
		{CharsetsCharsetDecSpecial, "dec_special"},
	}
	for _, slot := range slots {
		for _, set := range sets {
			rec := strA_recorderInit(true)
			// Zig-only: defer term51ARecRelease(&rec);
			s := StreamInit(&rec)
			defer s.Deinit()
			zigTry(t, s.configureCharset([]uint8{slot.first}, set.first))
			expected := fmt.Sprintf("configure_charset %s %s\n", slot.second, set.second)
			// Zig-only: defer testing.allocator.free(expected);
			expectEqualStrings(t, expected, strA_recText(&rec))
		}
	}
}

// Zig: test "term51: Stream.configureCharset: only '(' ')' '*' '+' are slots"
func TestStream_term51_Stream_configureCharset_only_are_slots(t *testing.T) {
	for i := 0; i < 256; i++ {
		c := uint8(i)
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		zigTry(t, s.configureCharset([]uint8{c}, CharsetsCharsetAscii))
		var expected string
		switch c {
		case '(':
			expected = "configure_charset G0 ascii\n"
		case ')':
			expected = "configure_charset G1 ascii\n"
		case '*':
			expected = "configure_charset G2 ascii\n"
		case '+':
			expected = "configure_charset G3 ascii\n"
		// Including '-', '.' and '/', which the VT220 uses for G1-G3 in
		// 96-character sets and which are not supported.
		default:
			expected = ""
		}
		t.Logf("intermediate 0x%x", c)
		expectEqualStrings(t, expected, strA_recText(&rec))
	}
}

// Zig: test "term51: Stream.configureCharset: the intermediate count must be exactly one"
func TestStream_term51_Stream_configureCharset_the_intermediate_count_must_be_exactly_one(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.configureCharset([]uint8(""), CharsetsCharsetAscii))
	zigTry(t, s.configureCharset([]uint8("()"), CharsetsCharsetAscii))
	zigTry(t, s.configureCharset([]uint8("(("), CharsetsCharsetAscii))
	zigTry(t, s.configureCharset([]uint8("+*"), CharsetsCharsetBritish))
	zigTry(t, s.configureCharset([]uint8("((("), CharsetsCharsetDecSpecial))
	zigTry(t, s.configureCharset([]uint8("(((("), CharsetsCharsetDecSpecial))
	expectEqualStrings(t, "", strA_recText(&rec))

	zigTry(t, s.configureCharset([]uint8("("), CharsetsCharsetAscii))
	expectEqualStrings(t, "configure_charset G0 ascii\n", strA_recText(&rec))
}

// ---------------------------------------------------------------------------
// Stream.escDispatch
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.escDispatch: every final byte without intermediates"
func TestStream_term51_Stream_escDispatch_every_final_byte_without_intermediates(t *testing.T) {
	for i := 0; i < 256; i++ {
		f := uint8(i)
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		strA_esc(t, &s, []uint8(""), f)
		t.Logf("final 0x%x", f)
		expectEqualStrings(t, strA_escNone(f), strA_recText(&rec))
	}
}

// Zig: test "term51: Stream.escDispatch: ESC 7 saves the cursor and ESC 8 restores it"
func TestStream_term51_Stream_escDispatch_ESC_7_saves_the_cursor_and_ESC_8_restores_it(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_esc(t, &s, []uint8(""), '7')
	expectEqualStrings(t, "save_cursor\n", strA_recText(&rec))
	strA_esc(t, &s, []uint8(""), '8')
	expectEqualStrings(t, "save_cursor\nrestore_cursor\n", strA_recText(&rec))

	// ESC 7 takes no intermediates at all.
	strA_recReset(&rec)
	for i := 0x20; i < 0x30; i++ {
		strA_esc(t, &s, []uint8{uint8(i)}, '7')
	}
	strA_esc(t, &s, []uint8("  "), '7')
	strA_esc(t, &s, []uint8("#"), '7')
	strA_esc(t, &s, []uint8("(("), '7')
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.escDispatch: ESC # 8 is DECALN, other intermediates on ESC 8 are not handled"
func TestStream_term51_Stream_escDispatch_ESC_8_is_DECALN_other_intermediates_on_ESC_8_are_not_handled(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_esc(t, &s, []uint8("#"), '8')
	expectEqualStrings(t, "decaln\n", strA_recText(&rec))

	strA_recReset(&rec)
	for i := 0x20; i < 0x30; i++ {
		if i == '#' {
			continue
		}
		strA_esc(t, &s, []uint8{uint8(i)}, '8')
	}
	// Two or more intermediates, even starting with '#'.
	strA_esc(t, &s, []uint8("##"), '8')
	strA_esc(t, &s, []uint8("# "), '8')
	strA_esc(t, &s, []uint8(" #"), '8')
	strA_esc(t, &s, []uint8("###"), '8')
	strA_esc(t, &s, []uint8("####"), '8')
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.escDispatch: charset designations in every slot with every supported final"
func TestStream_term51_Stream_escDispatch_charset_designations_in_every_slot_with_every_supported_final(t *testing.T) {
	slots := []struct {
		first  uint8
		second string
	}{
		{'(', "G0"},
		{')', "G1"},
		{'*', "G2"},
		{'+', "G3"},
	}
	finals := []struct {
		first  uint8
		second string
	}{
		{'B', "ascii"},
		{'A', "british"},
		{'0', "dec_special"},
	}
	for _, slot := range slots {
		for _, fin := range finals {
			rec := strA_recorderInit(true)
			// Zig-only: defer term51ARecRelease(&rec);
			s := StreamInit(&rec)
			defer s.Deinit()
			strA_esc(t, &s, []uint8{slot.first}, fin.first)
			expected := fmt.Sprintf("configure_charset %s %s\n", slot.second, fin.second)
			// Zig-only: defer testing.allocator.free(expected);
			expectEqualStrings(t, expected, strA_recText(&rec))
		}
	}
}

// Zig: test "term51: Stream.escDispatch: charset finals other than A, B and 0 are not handled"
func TestStream_term51_Stream_escDispatch_charset_finals_other_than_A_B_and_0_are_not_handled(t *testing.T) {
	// The DEC supplemental, national replacement and other sets that a real
	// VT220 supports are not implemented.
	finals := "1234567<=>@CEFGHIJKLMNOPQRSTUVWXYZ`afghijklmnopqrstuvwxyz"
	for _, f := range []uint8(finals) {
		for _, slot := range []uint8("()*+") {
			rec := strA_recorderInit(true)
			// Zig-only: defer term51ARecRelease(&rec);
			s := StreamInit(&rec)
			defer s.Deinit()
			strA_esc(t, &s, []uint8{slot}, f)
			t.Logf("slot %c final %c", slot, f)
			expectEqualStrings(t, "", strA_recText(&rec))
		}
	}
}

// Zig: test "term51: Stream.escDispatch: charset designations need exactly one valid intermediate"
func TestStream_term51_Stream_escDispatch_charset_designations_need_exactly_one_valid_intermediate(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// No intermediate.
	strA_esc(t, &s, []uint8(""), 'B')
	strA_esc(t, &s, []uint8(""), 'A')
	strA_esc(t, &s, []uint8(""), '0')
	// Unsupported slot intermediates.
	for _, slot := range []uint8("-./ !\"#$%&',") {
		strA_esc(t, &s, []uint8{slot}, 'B')
		strA_esc(t, &s, []uint8{slot}, 'A')
		strA_esc(t, &s, []uint8{slot}, '0')
	}
	// Several intermediates.
	for _, inter := range []string{"()", "((", "(  ", "( (", "(((", "((((", "%(", "+ "} {
		strA_esc(t, &s, []uint8(inter), 'B')
		strA_esc(t, &s, []uint8(inter), 'A')
		strA_esc(t, &s, []uint8(inter), '0')
	}
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.escDispatch: every other final with a single intermediate is not handled"
func TestStream_term51_Stream_escDispatch_every_other_final_with_a_single_intermediate_is_not_handled(t *testing.T) {
	for fi := 0; fi < 256; fi++ {
		f := uint8(fi)
		for ii := 0x20; ii < 0x30; ii++ {
			inter := uint8(ii)
			rec := strA_recorderInit(true)
			// Zig-only: defer term51ARecRelease(&rec);
			s := StreamInit(&rec)
			defer s.Deinit()
			strA_esc(t, &s, []uint8{inter}, f)

			// Zig: slot: ?[]const u8, set: ?[]const u8; "" is null here.
			var slot string
			switch inter {
			case '(':
				slot = "G0"
			case ')':
				slot = "G1"
			case '*':
				slot = "G2"
			case '+':
				slot = "G3"
			default:
				slot = ""
			}
			expected := ""
			if slot != "" {
				var set string
				switch f {
				case 'B':
					set = "ascii"
				case 'A':
					set = "british"
				case '0':
					set = "dec_special"
				default:
					set = ""
				}
				if set != "" {
					expected = fmt.Sprintf("configure_charset %s %s\n", slot, set)
				}
			}
			if f == '8' && inter == '#' {
				expected = "decaln\n"
			}

			t.Logf("intermediate 0x%x final 0x%x", inter, f)
			expectEqualStrings(t, expected, strA_recText(&rec))
		}
	}
}

// Zig: test "term51: Stream.escDispatch: two or more intermediates are never handled"
func TestStream_term51_Stream_escDispatch_two_or_more_intermediates_are_never_handled(t *testing.T) {
	inters := []string{"  ", "((", "()", "#8", "# ", "(0", "%G", "   ", "    ", "(((#"}
	for fi := 0; fi < 256; fi++ {
		f := uint8(fi)
		for _, inter := range inters {
			rec := strA_recorderInit(true)
			// Zig-only: defer term51ARecRelease(&rec);
			s := StreamInit(&rec)
			defer s.Deinit()
			strA_esc(t, &s, []uint8(inter), f)
			t.Logf("intermediates %s final 0x%x", inter, f)
			expectEqualStrings(t, "", strA_recText(&rec))
		}
	}
}

// Zig: test "term51: Stream.escDispatch: IND, NEL, HTS and RI"
func TestStream_term51_Stream_escDispatch_IND_NEL_HTS_and_RI(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_esc(t, &s, []uint8(""), 'D')
	strA_esc(t, &s, []uint8(""), 'E')
	strA_esc(t, &s, []uint8(""), 'H')
	strA_esc(t, &s, []uint8(""), 'M')
	expectEqualStrings(t, "index\nnext_line\ntab_set\nreverse_index\n", strA_recText(&rec))

	strA_recReset(&rec)
	for _, f := range []uint8("DEHM") {
		strA_esc(t, &s, []uint8(" "), f)
		strA_esc(t, &s, []uint8("#"), f)
		strA_esc(t, &s, []uint8("("), f)
		strA_esc(t, &s, []uint8("  "), f)
	}
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.escDispatch: single shifts and locking shifts"
func TestStream_term51_Stream_escDispatch_single_shifts_and_locking_shifts(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// SS2 / SS3 are reported as locking shifts of GL.
	strA_esc(t, &s, []uint8(""), 'N')
	strA_esc(t, &s, []uint8(""), 'O')
	// LS2 / LS3.
	strA_esc(t, &s, []uint8(""), 'n')
	strA_esc(t, &s, []uint8(""), 'o')
	// LS1R / LS2R / LS3R.
	strA_esc(t, &s, []uint8(""), '~')
	strA_esc(t, &s, []uint8(""), '}')
	strA_esc(t, &s, []uint8(""), '|')
	expectEqualStrings(
		t,
		"invoke_charset GL G2 locking=true\n"+
			"invoke_charset GL G3 locking=true\n"+
			"invoke_charset GL G2 locking=false\n"+
			"invoke_charset GL G3 locking=false\n"+
			"invoke_charset GR G1 locking=false\n"+
			"invoke_charset GR G2 locking=false\n"+
			"invoke_charset GR G3 locking=false\n",
		strA_recText(&rec),
	)

	strA_recReset(&rec)
	for _, f := range []uint8("NOno~}|") {
		strA_esc(t, &s, []uint8(" "), f)
		strA_esc(t, &s, []uint8("("), f)
		strA_esc(t, &s, []uint8("  "), f)
	}
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.escDispatch: SPA, EPA, DECID and RIS"
func TestStream_term51_Stream_escDispatch_SPA_EPA_DECID_and_RIS(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_esc(t, &s, []uint8(""), 'V')
	strA_esc(t, &s, []uint8(""), 'W')
	strA_esc(t, &s, []uint8(""), 'Z')
	strA_esc(t, &s, []uint8(""), 'c')
	expectEqualStrings(
		t,
		"protected_mode_iso\nprotected_mode_off\ndevice_attributes primary\nfull_reset\n",
		strA_recText(&rec),
	)

	strA_recReset(&rec)
	for _, f := range []uint8("VWZc") {
		strA_esc(t, &s, []uint8(" "), f)
		strA_esc(t, &s, []uint8("#"), f)
		strA_esc(t, &s, []uint8("  "), f)
	}
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.escDispatch: application keypad mode"
func TestStream_term51_Stream_escDispatch_application_keypad_mode(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_esc(t, &s, []uint8(""), '=')
	strA_esc(t, &s, []uint8(""), '>')
	expectEqualStrings(t, "set_mode keypad_keys\nreset_mode keypad_keys\n", strA_recText(&rec))

	strA_recReset(&rec)
	for _, f := range []uint8("=>") {
		strA_esc(t, &s, []uint8(" "), f)
		strA_esc(t, &s, []uint8("#"), f)
		strA_esc(t, &s, []uint8("  "), f)
	}
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.escDispatch: ST is accepted and ignored"
func TestStream_term51_Stream_escDispatch_ST_is_accepted_and_ignored(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	strA_esc(t, &s, []uint8(""), '\\')
	strA_esc(t, &s, []uint8(" "), '\\')
	strA_esc(t, &s, []uint8("  "), '\\')
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.escDispatch: unknown finals are ignored"
func TestStream_term51_Stream_escDispatch_unknown_finals_are_ignored(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	for _, f := range []uint8("P[]X^_@FGIJKLQRSTUY1234569:;<aefghijklmpqrstuvwxyz{") {
		strA_esc(t, &s, []uint8(""), f)
	}
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.escDispatch: sequences through the byte stream"
func TestStream_term51_Stream_escDispatch_sequences_through_the_byte_stream(t *testing.T) {
	strA_expectBoth(
		t,
		"\x1b7\x1b8\x1bD\x1bE\x1bH\x1bM\x1bN\x1bO\x1bV\x1bW\x1bZ\x1bc\x1bn\x1bo\x1b~\x1b}\x1b|\x1b=\x1b>\x1b\\",
		"save_cursor\nrestore_cursor\nindex\nnext_line\ntab_set\nreverse_index\n"+
			"invoke_charset GL G2 locking=true\ninvoke_charset GL G3 locking=true\n"+
			"protected_mode_iso\nprotected_mode_off\ndevice_attributes primary\nfull_reset\n"+
			"invoke_charset GL G2 locking=false\ninvoke_charset GL G3 locking=false\n"+
			"invoke_charset GR G1 locking=false\ninvoke_charset GR G2 locking=false\n"+
			"invoke_charset GR G3 locking=false\n"+
			"set_mode keypad_keys\nreset_mode keypad_keys\n",
	)
	strA_expectBoth(
		t,
		"\x1b(0\x1b)A\x1b*B\x1b+0\x1b(B",
		"configure_charset G0 dec_special\nconfigure_charset G1 british\n"+
			"configure_charset G2 ascii\nconfigure_charset G3 dec_special\n"+
			"configure_charset G0 ascii\n",
	)
	strA_expectBoth(t, "\x1b#8", "decaln\n")
	// Not handled: ESC # 3, ESC - B, ESC ( 1, ESC SP 7, ESC % G.
	strA_expectBoth(t, "\x1b#3\x1b-B\x1b(1\x1b 7\x1b%G\x1b#7", "")
	// Four intermediates are collected, a fifth is dropped by the parser but
	// the sequence still ends at its final byte.
	strA_expectBoth(t, "\x1b    7\x1b     7", "")
	// Text and controls around an escape.
	strA_expectBoth(t, "a\x1bMb", "print 61\nreverse_index\nprint 62\n")
}

// Zig: test "term51: Stream.escDispatch: C1 controls through the byte stream"
func TestStream_term51_Stream_escDispatch_C1_controls_through_the_byte_stream(t *testing.T) {
	// In a pending escape sequence the C1 bytes are controls.
	strA_expectNext(t, "\x1b\x84\x1b\x85\x1b\x88\x1b\x8D", "index\nnext_line\ntab_set\nreverse_index\n")
	strA_expectNext(t, "\x1b\x8E\x1b\x8F", "invoke_charset GL G2 locking=true\ninvoke_charset GL G3 locking=true\n")
	strA_expectNext(t, "\x1b\x96\x1b\x97\x1b\x9A", "protected_mode_iso\nprotected_mode_off\ndevice_attributes primary\n")
	// A C1 byte aborts the sequence it interrupts.
	strA_expectNext(t, "\x1b(\x84", "index\n")
	strA_expectNext(t, "\x1b[1;2\x8D", "reverse_index\n")
	// ... and the following bytes are text again.
	strA_expectNext(t, "\x1b\x84A", "index\nprint 41\n")
}

// ---------------------------------------------------------------------------
// Stream.oscDispatch
// ---------------------------------------------------------------------------

// Zig: test "term51: Stream.oscDispatch: window title is forwarded when valid UTF-8"
func TestStream_term51_Stream_oscDispatch_window_title_is_forwarded_when_valid_UTF_8(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyChangeWindowTitle, ChangeWindowTitle: []uint8("hello")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyChangeWindowTitle, ChangeWindowTitle: []uint8("")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyChangeWindowTitle, ChangeWindowTitle: []uint8("h\xC3\xA9llo \xE2\x82\xAC \xF0\x9F\x98\x80")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyChangeWindowTitle, ChangeWindowTitle: []uint8("with\ttab")}))
	// The maximum code point and the extremes of each length class.
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyChangeWindowTitle, ChangeWindowTitle: []uint8("\xF4\x8F\xBF\xBF\xC2\x80\xDF\xBF\xE0\xA0\x80\xEF\xBF\xBF\xF0\x90\x80\x80")}))
	expectEqualStrings(
		t,
		"window_title hello\nwindow_title \nwindow_title h\xC3\xA9llo \xE2\x82\xAC \xF0\x9F\x98\x80\nwindow_title with\ttab\n"+
			"window_title \xF4\x8F\xBF\xBF\xC2\x80\xDF\xBF\xE0\xA0\x80\xEF\xBF\xBF\xF0\x90\x80\x80\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.oscDispatch: window title that is not valid UTF-8 is dropped"
func TestStream_term51_Stream_oscDispatch_window_title_that_is_not_valid_UTF_8_is_dropped(t *testing.T) {
	bad := []string{
		"\xFF",
		"abc\xFF",
		"\x80",
		"\xC3",
		"\xC3(",
		"\xE2\x82",
		"\xE2\x28\xA1",
		"\xC0\x80",         // overlong NUL
		"\xE0\x80\x80",     // overlong
		"\xED\xA0\x80",     // UTF-16 surrogate
		"\xF4\x90\x80\x80", // above U+10FFFF
		"\xF5\x80\x80\x80",
		"\xF0\x80\x80\x80", // overlong
		"ok\xFEok",
	}
	for _, title := range bad {
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyChangeWindowTitle, ChangeWindowTitle: []uint8(title)}))
		t.Logf("title %v", []uint8(title))
		expectEqualStrings(t, "", strA_recText(&rec))
	}
}

// Zig: test "term51: Stream.oscDispatch: window icon is ignored"
func TestStream_term51_Stream_oscDispatch_window_icon_is_ignored(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyChangeWindowIcon, ChangeWindowIcon: []uint8("icon")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyChangeWindowIcon, ChangeWindowIcon: []uint8("")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyChangeWindowIcon, ChangeWindowIcon: []uint8("\xFF")}))
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.oscDispatch: semantic prompt is forwarded unchanged"
func TestStream_term51_Stream_oscDispatch_semantic_prompt_is_forwarded_unchanged(t *testing.T) {
	// Zig: const Action_ = osc.Command.SemanticPrompt.Action;
	actions := []struct {
		first  OSCParsersSemanticPromptCommandAction
		second string
	}{
		{OSCParsersSemanticPromptCommandActionFreshLine, "fresh_line"},
		{OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "fresh_line_new_prompt"},
		{OSCParsersSemanticPromptCommandActionNewCommand, "new_command"},
		{OSCParsersSemanticPromptCommandActionPromptStart, "prompt_start"},
		{OSCParsersSemanticPromptCommandActionEndPromptStartInput, "end_prompt_start_input"},
		{OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol, "end_prompt_start_input_terminate_eol"},
		{OSCParsersSemanticPromptCommandActionEndInputStartOutput, "end_input_start_output"},
		{OSCParsersSemanticPromptCommandActionEndCommand, "end_command"},
	}
	for _, a := range actions {
		rec := strA_recorderInit(true)
		// Zig-only: defer term51ARecRelease(&rec);
		s := StreamInit(&rec)
		defer s.Deinit()
		zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommandInit(a.first)}))
		zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCParsersSemanticPromptCommand{Action: a.first, OptionsUnvalidated: []uint8("aid=1;cl=m")}}))
		expected := fmt.Sprintf("semantic_prompt %s opts=\nsemantic_prompt %s opts=aid=1;cl=m\n", a.second, a.second)
		// Zig-only: defer testing.allocator.free(expected);
		expectEqualStrings(t, expected, strA_recText(&rec))
	}
}

// Zig: test "term51: Stream.oscDispatch: clipboard contents"
func TestStream_term51_Stream_oscDispatch_clipboard_contents(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyClipboardContents, ClipboardContents: OSCCommandClipboardContentsPayload{Kind: 'c', Data: []uint8("Zm9v")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyClipboardContents, ClipboardContents: OSCCommandClipboardContentsPayload{Kind: 's', Data: []uint8("?")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyClipboardContents, ClipboardContents: OSCCommandClipboardContentsPayload{Kind: 'p', Data: []uint8("")}}))
	// The kind is not validated.
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyClipboardContents, ClipboardContents: OSCCommandClipboardContentsPayload{Kind: 0, Data: []uint8("x")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyClipboardContents, ClipboardContents: OSCCommandClipboardContentsPayload{Kind: 0xFF, Data: []uint8("\xFF")}}))
	expectEqualStrings(
		t,
		"clipboard_contents kind=99 data=Zm9v\n"+
			"clipboard_contents kind=115 data=?\n"+
			"clipboard_contents kind=112 data=\n"+
			"clipboard_contents kind=0 data=x\n"+
			"clipboard_contents kind=255 data=\xFF\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.oscDispatch: report pwd is forwarded without validation"
func TestStream_term51_Stream_oscDispatch_report_pwd_is_forwarded_without_validation(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyReportPwd, ReportPwd: OSCCommandReportPwdPayload{Value: []uint8("file:///tmp/x")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyReportPwd, ReportPwd: OSCCommandReportPwdPayload{Value: []uint8("")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyReportPwd, ReportPwd: OSCCommandReportPwdPayload{Value: []uint8("not a url \xFF")}}))
	expectEqualStrings(
		t,
		"report_pwd file:///tmp/x\nreport_pwd \nreport_pwd not a url \xFF\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.oscDispatch: mouse shape by name"
func TestStream_term51_Stream_oscDispatch_mouse_shape_by_name(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	names := []struct {
		first  string
		second string
	}{
		{"default", "default"},
		{"pointer", "pointer"},
		{"context-menu", "context_menu"},
		{"vertical-text", "vertical_text"},
		{"zoom-out", "zoom_out"},
		{"nwse-resize", "nwse_resize"},
		// xterm / foot aliases
		{"left_ptr", "default"},
		{"hand", "pointer"},
		{"xterm", "text"},
		{"watch", "wait"},
		{"fleur", "all_scroll"},
		{"dnd-link", "alias"},
	}
	for _, n := range names {
		strA_recReset(&rec)
		zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyMouseShape, MouseShape: OSCCommandMouseShapePayload{Value: []uint8(n.first)}}))
		expected := fmt.Sprintf("mouse_shape %s\n", n.second)
		// Zig-only: defer testing.allocator.free(expected);
		expectEqualStrings(t, expected, strA_recText(&rec))
	}
}

// Zig: test "term51: Stream.oscDispatch: unknown mouse shapes are dropped"
func TestStream_term51_Stream_oscDispatch_unknown_mouse_shapes_are_dropped(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	unknown := []string{"", "bogus", "POINTER", "Pointer", "pointer ", " pointer", "context_menu", "col_resize", "zoom", "left-ptr", "\xFF"}
	for _, name := range unknown {
		zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyMouseShape, MouseShape: OSCCommandMouseShapePayload{Value: []uint8(name)}}))
	}
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.oscDispatch: colour operations pass op, terminator and requests through"
func TestStream_term51_Stream_oscDispatch_colour_operations_pass_op_terminator_and_requests_through(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	// Empty list, each terminator.
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyColorOperation, ColorOperation: OSCCommandColorOperationPayload{Op: OSCParsersColorOperationOsc104, Terminator: OSCTerminatorSt}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyColorOperation, ColorOperation: OSCCommandColorOperationPayload{Op: OSCParsersColorOperationOsc119, Terminator: OSCTerminatorBel}}))
	// Default terminator is ST.
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyColorOperation, ColorOperation: OSCCommandColorOperationPayload{Op: OSCParsersColorOperationOsc4}}))
	expectEqualStrings(
		t,
		"color_operation osc_104 st n=0\ncolor_operation osc_119 bel n=0\ncolor_operation osc_4 st n=0\n",
		strA_recText(&rec),
	)

	// A list longer than the in-place storage of two items.
	strA_recReset(&rec)
	var list OSCParsersColorList
	// Zig-only: errdefer list.deinit(alloc);
	// Zig: list.append(alloc, x) is an append to the items of the list.
	list.items = append(list.items, OSCParsersColorRequest{Tag: OSCParsersColorRequestQuery, Query: OSCParsersColorTarget{Tag: OSCParsersColorTargetDynamic, Dynamic: ColorDynamicForeground}})
	list.items = append(list.items, OSCParsersColorRequest{Tag: OSCParsersColorRequestSet, Set: OSCParsersColorColoredTarget{Target: OSCParsersColorTarget{Tag: OSCParsersColorTargetPalette, Palette: 7}, Color: ColorRGB{R: 1, G: 2, B: 3}}})
	list.items = append(list.items, OSCParsersColorRequest{Tag: OSCParsersColorRequestReset, Reset: OSCParsersColorTarget{Tag: OSCParsersColorTargetSpecial, Special: ColorSpecialBold}})
	list.items = append(list.items, OSCParsersColorRequest{Tag: OSCParsersColorRequestResetPalette})
	list.items = append(list.items, OSCParsersColorRequest{Tag: OSCParsersColorRequestResetSpecial})
	// The recorder takes ownership of the list and frees it.
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyColorOperation, ColorOperation: OSCCommandColorOperationPayload{Op: OSCParsersColorOperationOsc4, Requests: list, Terminator: OSCTerminatorBel}}))
	expectEqualStrings(
		t,
		"color_operation osc_4 bel n=5 query(d.foreground) set(p7=#010203) reset(s.bold) reset_palette reset_special\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.oscDispatch: kitty colour protocol is forwarded"
func TestStream_term51_Stream_oscDispatch_kitty_colour_protocol_is_forwarded(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyKittyColorProtocol, KittyColorProtocol: KittyColorOSC{List: nil, Terminator: OSCTerminatorSt}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyKittyColorProtocol, KittyColorProtocol: KittyColorOSC{List: nil, Terminator: OSCTerminatorBel}}))
	expectEqualStrings(
		t,
		"kitty_color_report st n=0\nkitty_color_report bel n=0\n",
		strA_recText(&rec),
	)

	strA_recReset(&rec)
	c := OSCCommand{Tag: OSCCommandKeyKittyColorProtocol, KittyColorProtocol: KittyColorOSC{List: nil, Terminator: OSCTerminatorBel}}
	defer c.KittyColorProtocol.Deinit()
	// Zig: list.append(alloc, x) is an append to the list.
	c.KittyColorProtocol.List = append(c.KittyColorProtocol.List, KittyColorOSCRequest{Tag: KittyColorOSCRequestQuery, Query: KittyColorKind{Tag: KittyColorKindSpecial, Special: KittyColorSpecialCursor}})
	c.KittyColorProtocol.List = append(c.KittyColorProtocol.List, KittyColorOSCRequest{Tag: KittyColorOSCRequestSet, Set: KittyColorOSCRequestSetPayload{Key: KittyColorKind{Tag: KittyColorKindPalette, Palette: 3}, Color: ColorRGB{R: 255, G: 0, B: 16}}})
	c.KittyColorProtocol.List = append(c.KittyColorProtocol.List, KittyColorOSCRequest{Tag: KittyColorOSCRequestReset, Reset: KittyColorKind{Tag: KittyColorKindPalette, Palette: 255}})
	zigTry(t, s.oscDispatch(c))
	expectEqualStrings(
		t,
		"kitty_color_report bel n=3 query(cursor) set(3=#ff0010) reset(255)\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.oscDispatch: desktop notification"
func TestStream_term51_Stream_oscDispatch_desktop_notification(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyShowDesktopNotification, ShowDesktopNotification: OSCCommandShowDesktopNotificationPayload{Title: []uint8("Title"), Body: []uint8("Body")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyShowDesktopNotification, ShowDesktopNotification: OSCCommandShowDesktopNotificationPayload{Title: []uint8(""), Body: []uint8("only body")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyShowDesktopNotification, ShowDesktopNotification: OSCCommandShowDesktopNotificationPayload{Title: []uint8(""), Body: []uint8("")}}))
	expectEqualStrings(
		t,
		"show_desktop_notification title=Title body=Body\n"+
			"show_desktop_notification title= body=only body\n"+
			"show_desktop_notification title= body=\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.oscDispatch: hyperlinks"
func TestStream_term51_Stream_oscDispatch_hyperlinks(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyHyperlinkStart, HyperlinkStart: OSCCommandHyperlinkStartPayload{URI: []uint8("http://example.com")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyHyperlinkStart, HyperlinkStart: OSCCommandHyperlinkStartPayload{URI: []uint8("http://example.com"), ID: Some([]uint8("abc"))}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyHyperlinkStart, HyperlinkStart: OSCCommandHyperlinkStartPayload{URI: []uint8(""), ID: None[[]uint8]()}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyHyperlinkEnd}))
	expectEqualStrings(
		t,
		"start_hyperlink uri=http://example.com id=null\n"+
			"start_hyperlink uri=http://example.com id=abc\n"+
			"start_hyperlink uri= id=null\n"+
			"end_hyperlink\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.oscDispatch: progress report"
func TestStream_term51_Stream_oscDispatch_progress_report(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuProgressReport, ConemuProgressReport: OSCCommandProgressReport{State: OSCCommandProgressReportStateSet, Progress: Some(uint8(50))}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuProgressReport, ConemuProgressReport: OSCCommandProgressReport{State: OSCCommandProgressReportStateRemove}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuProgressReport, ConemuProgressReport: OSCCommandProgressReport{State: OSCCommandProgressReportStateError, Progress: Some(uint8(100))}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuProgressReport, ConemuProgressReport: OSCCommandProgressReport{State: OSCCommandProgressReportStateIndeterminate}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuProgressReport, ConemuProgressReport: OSCCommandProgressReport{State: OSCCommandProgressReportStatePause, Progress: Some(uint8(0))}}))
	// The value is forwarded as is, even beyond 100.
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuProgressReport, ConemuProgressReport: OSCCommandProgressReport{State: OSCCommandProgressReportStateSet, Progress: Some(uint8(255))}}))
	expectEqualStrings(
		t,
		"progress_report set 50\nprogress_report remove null\nprogress_report error 100\n"+
			"progress_report indeterminate null\nprogress_report pause 0\nprogress_report set 255\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.oscDispatch: ConEmu and kitty commands that are not implemented are ignored"
func TestStream_term51_Stream_oscDispatch_ConEmu_and_kitty_commands_that_are_not_implemented_are_ignored(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuSleep, ConemuSleep: OSCCommandConemuSleepPayload{DurationMs: 100}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuShowMessageBox, ConemuShowMessageBox: []uint8("message")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuChangeTabTitle, ConemuChangeTabTitle: OSCCommandConemuChangeTabTitlePayload{Tag: OSCCommandConemuChangeTabTitlePayloadReset}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuChangeTabTitle, ConemuChangeTabTitle: OSCCommandConemuChangeTabTitlePayload{Tag: OSCCommandConemuChangeTabTitlePayloadValue, Value: []uint8("tab")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuWaitInput}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuGuimacro, ConemuGuimacro: []uint8("macro")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuRunProcess, ConemuRunProcess: []uint8("cmd")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuOutputEnvironmentVariable, ConemuOutputEnvironmentVariable: []uint8("VAR")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuXtermEmulation, ConemuXtermEmulation: OSCCommandConemuXtermEmulationPayload{Keyboard: Some(true), Output: None[bool]()}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuXtermEmulation, ConemuXtermEmulation: OSCCommandConemuXtermEmulationPayload{Keyboard: None[bool](), Output: Some(false)}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyConemuComment, ConemuComment: []uint8("comment")}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyKittyTextSizing, KittyTextSizing: OSCParsersKittyTextSizingOSC{Text: []uint8("text")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyKittyClipboardProtocol, KittyClipboardProtocol: OSCCommandKittyClipboardProtocol{Metadata: []uint8("type=read"), Payload: None[[]uint8](), Terminator: OSCTerminatorSt}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyKittyClipboardProtocol, KittyClipboardProtocol: OSCCommandKittyClipboardProtocol{Metadata: []uint8(""), Payload: Some([]uint8("abc")), Terminator: OSCTerminatorBel}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyContextSignal, ContextSignal: OSCParsersContextSignalCommand{Action: OSCParsersContextSignalCommandActionStart, ID: []uint8("ctx"), Metadata: []uint8("")}}))
	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyContextSignal, ContextSignal: OSCParsersContextSignalCommand{Action: OSCParsersContextSignalCommandActionEnd, ID: []uint8("ctx"), Metadata: []uint8("exit=success")}}))
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.oscDispatch: the invalid command is ignored"
func TestStream_term51_Stream_oscDispatch_the_invalid_command_is_ignored(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInit(&rec)
	defer s.Deinit()

	zigTry(t, s.oscDispatch(OSCCommand{Tag: OSCCommandKeyInvalid}))
	expectEqualStrings(t, "", strA_recText(&rec))
}

// Zig: test "term51: Stream.oscDispatch: parsed commands that are ignored never reach the handler"
func TestStream_term51_Stream_oscDispatch_parsed_commands_that_are_ignored_never_reach_the_handler(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	cases := []struct {
		first  string
		second OSCCommandKey
	}{
		{"1;icon", OSCCommandKeyChangeWindowIcon},
		{"9;1;420", OSCCommandKeyConemuSleep},
		{"9;2;hello world", OSCCommandKeyConemuShowMessageBox},
		{"9;3;foo bar", OSCCommandKeyConemuChangeTabTitle},
		{"9;3;", OSCCommandKeyConemuChangeTabTitle},
		{"9;5", OSCCommandKeyConemuWaitInput},
		{"9;6;ab", OSCCommandKeyConemuGuimacro},
		{"9;7;ab", OSCCommandKeyConemuRunProcess},
		{"9;8;ab", OSCCommandKeyConemuOutputEnvironmentVariable},
		{"9;10", OSCCommandKeyConemuXtermEmulation},
		{"9;10;1;0", OSCCommandKeyConemuXtermEmulation},
		{"9;11;a comment", OSCCommandKeyConemuComment},
		{"66;s=2;kurwa", OSCCommandKeyKittyTextSizing},
		{"5522;type=read;dGV4dC9wbGFpbg==", OSCCommandKeyKittyClipboardProtocol},
		{"3008;start=abc123", OSCCommandKeyContextSignal},
		{"3008;end=abc123;exit=success", OSCCommandKeyContextSignal},
	}
	for _, c := range cases {
		// Check the parsed command kind...
		{
			p := OSCParserInit(true)
			defer p.Deinit()
			for _, ch := range []uint8(c.first) {
				p.Next(ch)
			}
			ptr := p.End(Some(uint8(0x1b)))
			if ptr == nil {
				t.Fatalf("unwrapped a null optional")
			}
			cmd := *ptr
			expectEqual(t, c.second, cmd.Tag)

			// ... dispatching it directly ...
			rec := strA_recorderInit(true)
			// Zig-only: defer term51ARecRelease(&rec);
			s := StreamInit(&rec)
			defer s.Deinit()
			zigTry(t, s.oscDispatch(cmd))
			t.Logf("osc %s", c.first)
			expectEqualStrings(t, "", strA_recText(&rec))
		}
		// ... and through the byte stream.
		{
			input := "\x1b]" + c.first + "\x07"
			// Zig-only: defer alloc.free(input);
			strA_expectBoth(t, input, "")
			input2 := "\x1b]" + c.first + "\x1b\\"
			// Zig-only: defer alloc.free(input2);
			strA_expectBoth(t, input2, "")
		}
	}
}

// Zig: test "term51: Stream.oscDispatch: commands through the byte stream"
func TestStream_term51_Stream_oscDispatch_commands_through_the_byte_stream(t *testing.T) {
	// BEL and ST terminators give the same result for non-colour commands.
	cases := []struct {
		first  string
		second string
	}{
		{"0;a title", "window_title a title\n"},
		{"2;another", "window_title another\n"},
		{"0;", "window_title \n"},
		{"7;file:///tmp/dir", "report_pwd file:///tmp/dir\n"},
		{"8;;http://example.com", "start_hyperlink uri=http://example.com id=null\n"},
		{"8;id=x;http://example.com", "start_hyperlink uri=http://example.com id=x\n"},
		{"8;;", "end_hyperlink\n"},
		{"9;hello", "show_desktop_notification title= body=hello\n"},
		{"777;notify;T;B", "show_desktop_notification title=T body=B\n"},
		{"9;4;1;50", "progress_report set 50\n"},
		{"22;pointer", "mouse_shape pointer\n"},
		{"22;nonsense", ""},
		{"52;c;Zm9v", "clipboard_contents kind=99 data=Zm9v\n"},
		{"52;;?", "clipboard_contents kind=99 data=?\n"},
		{"133;A", "semantic_prompt fresh_line_new_prompt opts=\n"},
		{"133;D;0", "semantic_prompt end_command opts=0\n"},
		{"1;icon", ""},
		{"9999;x", ""},
	}
	for _, c := range cases {
		bel := "\x1b]" + c.first + "\x07"
		// Zig-only: defer testing.allocator.free(bel);
		st := "\x1b]" + c.first + "\x1b\\"
		// Zig-only: defer testing.allocator.free(st);
		t.Logf("osc %s (BEL)", c.first)
		strA_expectBoth(t, bel, c.second)
		t.Logf("osc %s (ST)", c.first)
		strA_expectBoth(t, st, c.second)
	}
}

// Zig: test "term51: Stream.oscDispatch: colour commands through the byte stream keep the terminator used"
func TestStream_term51_Stream_oscDispatch_colour_commands_through_the_byte_stream_keep_the_terminator_used(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	s := StreamInitAlloc(&rec)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("\x1b]10;?\x07")))
	zigTry(t, s.NextSlice([]uint8("\x1b]11;?\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b]12;rgb:01/02/03\x07")))
	zigTry(t, s.NextSlice([]uint8("\x1b]4;0;?;255;rgb:ff/ff/ff\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b]5;0;?\x07")))
	// OSC 105 (reset special colour) is not recognised by the OSC parser
	// at all, so nothing is emitted for it.
	zigTry(t, s.NextSlice([]uint8("\x1b]105\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b]112\x07")))
	expectEqualStrings(
		t,
		"color_operation osc_10 bel n=1 query(d.foreground)\n"+
			"color_operation osc_11 st n=1 query(d.background)\n"+
			"color_operation osc_12 bel n=1 set(d.cursor=#010203)\n"+
			"color_operation osc_4 st n=2 query(p0) set(p255=#ffffff)\n"+
			"color_operation osc_5 bel n=1 query(s.bold)\n"+
			"color_operation osc_112 bel n=1 reset(d.cursor)\n",
		strA_recText(&rec),
	)
}

// Zig: test "term51: Stream.oscDispatch: an invalid window title through the byte stream is dropped"
func TestStream_term51_Stream_oscDispatch_an_invalid_window_title_through_the_byte_stream_is_dropped(t *testing.T) {
	strA_expectBoth(t, "\x1b]0;a\xFFb\x07", "")
	strA_expectBoth(t, "\x1b]2;\xC0\x80\x07", "")
	strA_expectBoth(t, "\x1b]0;a\xC3\xA9b\x07", "window_title a\xC3\xA9b\n")
}

// ---------------------------------------------------------------------------
// Self-checks of the recording handlers used above
// ---------------------------------------------------------------------------

// Zig: test "term51: term51ARecorder.deinit: only counts the call"
func TestStream_term51_term51ARecorder_deinit_only_counts_the_call(t *testing.T) {
	rec := strA_recorderInit(true)
	// Zig-only: defer term51ARecRelease(&rec);
	expectEqual(t, uint(0), rec.deinits)
	rec.Deinit()
	rec.Deinit()
	expectEqual(t, uint(2), rec.deinits)
	// The log is still usable.
	strA_recPut(&rec, "x")
	expectEqualStrings(t, "x", strA_recText(&rec))
}

// Zig: test "term51: term51ARawRecorder.deinit: counts the call on the inner recorder"
func TestStream_term51_term51ARawRecorder_deinit_counts_the_call_on_the_inner_recorder(t *testing.T) {
	rec := strA_rawRecorderInit()
	// Zig-only: defer term51ARawRelease(&rec);
	rec.Deinit()
	expectEqual(t, uint(1), rec.base.deinits)
}

// Zig: test "term51: term51ARawRecorder.vtRaw: logs every kind of raw action and applies mode and only"
func TestStream_term51_term51ARawRecorder_vtRaw_logs_every_kind_of_raw_action_and_applies_mode_and_only(t *testing.T) {
	rec := strA_rawRecorderInit()
	// Zig-only: defer term51ARawRelease(&rec);
	inter := []uint8{'(', ' '}

	actions := []ParserAction{
		{Tag: ParserActionPrint, Print: 'a'},
		{Tag: ParserActionExecute, Execute: 0x0a},
		{Tag: ParserActionCsiDispatch, CsiDispatch: ParserActionCSI{Intermediates: []uint8{}, Params: []uint16{}, ParamsSep: StaticBitSet24InitEmpty(), Final: 'm'}},
		{Tag: ParserActionEscDispatch, EscDispatch: ParserActionESC{Intermediates: inter, Final: 'B'}},
		{Tag: ParserActionOscDispatch, OscDispatch: OSCCommand{Tag: OSCCommandKeyInvalid}},
		{Tag: ParserActionDcsHook, DcsHook: ParserActionDCS{Final: 'q'}},
		{Tag: ParserActionDcsPut, DcsPut: 0x41},
		{Tag: ParserActionDcsUnhook},
		{Tag: ParserActionApcStart},
		{Tag: ParserActionApcPut, ApcPut: 0x42},
		{Tag: ParserActionApcEnd},
	}
	for _, a := range actions {
		skip, err := rec.VtRaw(&a)
		zigTry(t, err)
		expect(t, !skip, "!(try rec.vtRaw(a))")
	}
	expectEqualStrings(
		t,
		"print 61\nexecute a\ncsi_dispatch m\nesc_dispatch ( B\nosc_dispatch invalid\n"+
			"dcs_hook q\ndcs_put 41\ndcs_unhook\napc_start\napc_put 42\napc_end\n",
		string(rec.raw),
	)

	rec.mode = strA_rawSkip
	skip, err := rec.VtRaw(&ParserAction{Tag: ParserActionExecute, Execute: 1})
	zigTry(t, err)
	expect(t, skip, "try rec.vtRaw(.{ .execute = 1 })")
	rec.mode = strA_rawFail
	_, err = rec.VtRaw(&ParserAction{Tag: ParserActionExecute, Execute: 1})
	expect(t, errors.Is(err, strA_errTestFailure), "error.Term51ATestFailure")
	// With `only` set, other kinds of action pass.
	rec.only = Some(ParserActionDcsPut)
	skip, err = rec.VtRaw(&ParserAction{Tag: ParserActionExecute, Execute: 1})
	zigTry(t, err)
	expect(t, !skip, "!(try rec.vtRaw(.{ .execute = 1 }))")
	_, err = rec.VtRaw(&ParserAction{Tag: ParserActionDcsPut, DcsPut: 1})
	expect(t, errors.Is(err, strA_errTestFailure), "error.Term51ATestFailure")
}
