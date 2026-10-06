// Tests translated one for one from section (B) of src/terminal/stream.zig
// (Ghostty v1.3.1 plus the term51 additional tests): the first half of the
// csiDispatch tests. Each Go test carries the exact Zig title in a
// "// Zig: test" comment. The helpers expect, expectEqual and zigTry are
// defined in other test files of the package (Parser_test.go, dcs_test.go);
// the helpers of this file are prefixed strB_.
//
// Translation notes:
//   - term51BRecorder is strB_recorder. The Zig handler is passed by value
//     and read back as `s.handler`; Go's handler must implement
//     StreamHandler, so it is a pointer: `StreamInit(&strB_recorder{})`, and
//     `s.handler.len` / `s.handler.events` work as in Zig.
//   - term51BEvent is a tagged union in Zig, so strB_event has a kind and
//     one field per payload; only the field named by kind is set. Zig's
//     expectEqualDeep is reflect.DeepEqual here.
//   - sgr.Attribute payloads are normalised by strB_sgrActive: a Go
//     SGRAttribute has one field per variant and only the field named by Tag
//     is meaningful, as in Zig, where the other payloads do not exist.
//   - std.fmt.bufPrint of "{s}{d}" is string concatenation with
//     strconv.Itoa; `@intCast` to u16/usize is the Go conversion;
//     `inline for` and `comptime` parameters are ordinary loops and
//     arguments.
//   - `testing.expectEqual(@as(usize, n), s.handler.len)` is
//     expectEqual(t, uint(n), s.handler.len).

package terminal

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

// strB_kind is the tag of term51BEvent. Every action the first half of
// csiDispatch can emit has its own variant; anything else is recorded as
// strB_other so an unexpected emission fails an exact comparison.
type strB_kind uint8

const (
	strB_cursor_up strB_kind = iota
	strB_cursor_down
	strB_cursor_left
	strB_cursor_right
	strB_cursor_col
	strB_cursor_row
	strB_cursor_col_relative
	strB_cursor_row_relative
	strB_cursor_pos
	strB_horizontal_tab
	strB_horizontal_tab_back
	strB_erase_display_below
	strB_erase_display_above
	strB_erase_display_complete
	strB_erase_display_scrollback
	strB_erase_display_scroll_complete
	strB_erase_line_right
	strB_erase_line_left
	strB_erase_line_complete
	strB_erase_line_right_unless_pending_wrap
	strB_insert_lines
	strB_delete_lines
	strB_delete_chars
	strB_erase_chars
	strB_scroll_up
	strB_scroll_down
	strB_print_repeat
	strB_tab_clear_current
	strB_tab_clear_all
	strB_tab_set
	strB_tab_reset
	strB_carriage_return
	strB_set_mode
	strB_reset_mode
	strB_device_attributes
	strB_modify_key_format
	strB_sgr_attr
	strB_sgr_unknown
	strB_other
)

// strB_unk is term51BUnk: a copy of sgr.Attribute.Unknown (the slices point
// into the parser's buffers, so they have to be copied while the action is
// delivered).
type strB_unk struct {
	full        [24]uint16
	full_len    uint8
	partial     [24]uint16
	partial_len uint8
}

// strB_event is term51BEvent: one recorded action.
type strB_event struct {
	kind strB_kind

	// cursor_up, cursor_down, cursor_left, cursor_right, cursor_col,
	// cursor_row, cursor_col_relative, cursor_row_relative,
	// horizontal_tab, horizontal_tab_back.
	u16 uint16
	// the erase_* variants.
	b bool
	// insert_lines, delete_lines, delete_chars, erase_chars, scroll_up,
	// scroll_down, print_repeat.
	usize uint
	pos   StreamActionCursorPos
	mode  ModesMode
	da    ANSIDeviceAttributeReq
	mkf   ANSIModifyKeyFormat
	sgr   SGRAttribute
	unk   strB_unk
	other StreamActionKey
}

// strB_none builds an event of a variant without payload.
func strB_none(kind strB_kind) strB_event {
	return strB_event{kind: kind}
}

// strB_boolEv is `@unionInit(term51BEvent, tag, b)` for a bool variant.
func strB_boolEv(kind strB_kind, b bool) strB_event {
	return strB_event{kind: kind, b: b}
}

func strB_pos(row uint16, col uint16) strB_event {
	return strB_event{kind: strB_cursor_pos, pos: StreamActionCursorPos{row: row, col: col}}
}

func strB_da(req ANSIDeviceAttributeReq) strB_event {
	return strB_event{kind: strB_device_attributes, da: req}
}

func strB_mkf(f ANSIModifyKeyFormat) strB_event {
	return strB_event{kind: strB_modify_key_format, mkf: f}
}

// strB_modeEvent is term51BModeEvent.
func strB_modeEvent(final uint8, mode ModesMode) strB_event {
	if final == 'h' {
		return strB_event{kind: strB_set_mode, mode: mode}
	}
	return strB_event{kind: strB_reset_mode, mode: mode}
}

// strB_sgrActive keeps only the payload named by the tag of a.
func strB_sgrActive(a SGRAttribute) SGRAttribute {
	r := SGRAttribute{Tag: a.Tag}
	switch a.Tag {
	case SGRAttributeTagUnderline:
		r.Underline = a.Underline
	case SGRAttributeTagUnderlineColor:
		r.UnderlineColor = a.UnderlineColor
	case SGRAttributeTag256UnderlineColor:
		r.X256UnderlineColor = a.X256UnderlineColor
	case SGRAttributeTagDirectColorFg:
		r.DirectColorFg = a.DirectColorFg
	case SGRAttributeTagDirectColorBg:
		r.DirectColorBg = a.DirectColorBg
	case SGRAttributeTag8Bg:
		r.X8Bg = a.X8Bg
	case SGRAttributeTag8Fg:
		r.X8Fg = a.X8Fg
	case SGRAttributeTag8BrightBg:
		r.X8BrightBg = a.X8BrightBg
	case SGRAttributeTag8BrightFg:
		r.X8BrightFg = a.X8BrightFg
	case SGRAttributeTag256Bg:
		r.X256Bg = a.X256Bg
	case SGRAttributeTag256Fg:
		r.X256Fg = a.X256Fg
	}
	return r
}

// strB_sgr is `.{ .sgr_attr = a }`.
func strB_sgr(a SGRAttribute) strB_event {
	return strB_event{kind: strB_sgr_attr, sgr: strB_sgrActive(a)}
}

// strB_mkUnk copies the two slices of an unknown attribute. Like Zig's
// @memcpy and @intCast it stops when a slice does not fit.
func strB_mkUnk(full []uint16, partial []uint16) strB_unk {
	var u strB_unk
	copy(u.full[0:len(full)], full)
	u.full_len = uint8(len(full))
	copy(u.partial[0:len(partial)], partial)
	u.partial_len = uint8(len(partial))
	return u
}

// strB_unkEv is term51BUnkEv.
func strB_unkEv(full []uint16, partial []uint16) strB_event {
	return strB_event{kind: strB_sgr_unknown, unk: strB_mkUnk(full, partial)}
}

// strB_recorder is term51BRecorder.
type strB_recorder struct {
	events [64]strB_event
	// Zig: usize.
	len uint
}

func (self *strB_recorder) Deinit() {
}

func (self *strB_recorder) push(e strB_event) error {
	if self.len >= uint(len(self.events)) {
		return errors.New("Term51BOverflow")
	}
	self.events[self.len] = e
	self.len += 1
	return nil
}

func (self *strB_recorder) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyCursorUp:
		return self.push(strB_event{kind: strB_cursor_up, u16: value.CursorUp.value})
	case StreamActionKeyCursorDown:
		return self.push(strB_event{kind: strB_cursor_down, u16: value.CursorDown.value})
	case StreamActionKeyCursorLeft:
		return self.push(strB_event{kind: strB_cursor_left, u16: value.CursorLeft.value})
	case StreamActionKeyCursorRight:
		return self.push(strB_event{kind: strB_cursor_right, u16: value.CursorRight.value})
	case StreamActionKeyCursorCol:
		return self.push(strB_event{kind: strB_cursor_col, u16: value.CursorCol.value})
	case StreamActionKeyCursorRow:
		return self.push(strB_event{kind: strB_cursor_row, u16: value.CursorRow.value})
	case StreamActionKeyCursorColRelative:
		return self.push(strB_event{kind: strB_cursor_col_relative, u16: value.CursorColRelative.value})
	case StreamActionKeyCursorRowRelative:
		return self.push(strB_event{kind: strB_cursor_row_relative, u16: value.CursorRowRelative.value})
	case StreamActionKeyCursorPos:
		return self.push(strB_event{kind: strB_cursor_pos, pos: value.CursorPos})
	case StreamActionKeyHorizontalTab:
		return self.push(strB_event{kind: strB_horizontal_tab, u16: value.HorizontalTab})
	case StreamActionKeyHorizontalTabBack:
		return self.push(strB_event{kind: strB_horizontal_tab_back, u16: value.HorizontalTabBack})
	case StreamActionKeyEraseDisplayBelow:
		return self.push(strB_boolEv(strB_erase_display_below, value.EraseDisplayBelow))
	case StreamActionKeyEraseDisplayAbove:
		return self.push(strB_boolEv(strB_erase_display_above, value.EraseDisplayAbove))
	case StreamActionKeyEraseDisplayComplete:
		return self.push(strB_boolEv(strB_erase_display_complete, value.EraseDisplayComplete))
	case StreamActionKeyEraseDisplayScrollback:
		return self.push(strB_boolEv(strB_erase_display_scrollback, value.EraseDisplayScrollback))
	case StreamActionKeyEraseDisplayScrollComplete:
		return self.push(strB_boolEv(strB_erase_display_scroll_complete, value.EraseDisplayScrollComplete))
	case StreamActionKeyEraseLineRight:
		return self.push(strB_boolEv(strB_erase_line_right, value.EraseLineRight))
	case StreamActionKeyEraseLineLeft:
		return self.push(strB_boolEv(strB_erase_line_left, value.EraseLineLeft))
	case StreamActionKeyEraseLineComplete:
		return self.push(strB_boolEv(strB_erase_line_complete, value.EraseLineComplete))
	case StreamActionKeyEraseLineRightUnlessPendingWrap:
		return self.push(strB_boolEv(strB_erase_line_right_unless_pending_wrap, value.EraseLineRightUnlessPendingWrap))
	case StreamActionKeyInsertLines:
		return self.push(strB_event{kind: strB_insert_lines, usize: value.InsertLines})
	case StreamActionKeyDeleteLines:
		return self.push(strB_event{kind: strB_delete_lines, usize: value.DeleteLines})
	case StreamActionKeyDeleteChars:
		return self.push(strB_event{kind: strB_delete_chars, usize: value.DeleteChars})
	case StreamActionKeyEraseChars:
		return self.push(strB_event{kind: strB_erase_chars, usize: value.EraseChars})
	case StreamActionKeyScrollUp:
		return self.push(strB_event{kind: strB_scroll_up, usize: value.ScrollUp})
	case StreamActionKeyScrollDown:
		return self.push(strB_event{kind: strB_scroll_down, usize: value.ScrollDown})
	case StreamActionKeyPrintRepeat:
		return self.push(strB_event{kind: strB_print_repeat, usize: value.PrintRepeat})
	case StreamActionKeyTabClearCurrent:
		return self.push(strB_none(strB_tab_clear_current))
	case StreamActionKeyTabClearAll:
		return self.push(strB_none(strB_tab_clear_all))
	case StreamActionKeyTabSet:
		return self.push(strB_none(strB_tab_set))
	case StreamActionKeyTabReset:
		return self.push(strB_none(strB_tab_reset))
	case StreamActionKeyCarriageReturn:
		return self.push(strB_none(strB_carriage_return))
	case StreamActionKeySetMode:
		return self.push(strB_event{kind: strB_set_mode, mode: value.SetMode.mode})
	case StreamActionKeyResetMode:
		return self.push(strB_event{kind: strB_reset_mode, mode: value.ResetMode.mode})
	case StreamActionKeyDeviceAttributes:
		return self.push(strB_da(value.DeviceAttributes))
	case StreamActionKeyModifyKeyFormat:
		return self.push(strB_mkf(value.ModifyKeyFormat))
	case StreamActionKeySetAttribute:
		switch value.SetAttribute.Tag {
		case SGRAttributeTagUnknown:
			u := value.SetAttribute.Unknown
			return self.push(strB_event{kind: strB_sgr_unknown, unk: strB_mkUnk(u.full, u.partial)})
		default:
			return self.push(strB_sgr(value.SetAttribute))
		}
	default:
		return self.push(strB_event{kind: strB_other, other: action})
	}
}

// strB_stream is term51BStream.
type strB_stream = Stream[*strB_recorder]

// strB_repeat is Zig's `[_]u16{v} ** n`.
func strB_repeat(v uint16, n int) []uint16 {
	r := make([]uint16, n)
	for i := range r {
		r[i] = v
	}
	return r
}

// strB_rgb is term51BRgb.
func strB_rgb(r uint8, g uint8, b uint8) ColorRGB {
	return ColorRGB{r: r, g: g, b: b}
}

// strB_ifs is Zig's `if (cond) a else b` on two strings.
func strB_ifs(cond bool, a string, b string) string {
	if cond {
		return a
	}
	return b
}

// strB_csi is term51BCsi: calls csiDispatch directly with a hand-built CSI
// action. `colons` lists the param indexes whose following separator is a
// colon.
func strB_csi(
	t *testing.T,
	s *strB_stream,
	inter []uint8,
	params []uint16,
	colons []uint,
	final uint8,
) {
	t.Helper()
	var ibuf [8]uint8
	var pbuf [40]uint16
	copy(ibuf[0:len(inter)], inter)
	copy(pbuf[0:len(params)], params)
	sep := StaticBitSet24InitEmpty()
	for _, c := range colons {
		sep.set(int(c))
	}
	zigTry(t, s.csiDispatch(ParserActionCSI{
		Intermediates: ibuf[0:len(inter)],
		Params:        pbuf[0:len(params)],
		ParamsSep:     sep,
		Final:         final,
	}))
}

// strB_feed is term51BFeed: feeds `ESC [ <body> <final>` through nextSlice.
func strB_feed(t *testing.T, s *strB_stream, body string, final uint8) {
	t.Helper()
	bytes := "\x1b[" + body + string(rune(final))
	zigTry(t, s.NextSlice([]uint8(bytes)))
}

// strB_expectEqualDeep is Zig's `testing.expectEqualDeep(expected, actual)`.
func strB_expectEqualDeep(t *testing.T, want strB_event, got strB_event) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// strB_expect is term51BExpect: compares the recorded events with the
// expected ones and clears them.
func strB_expect(t *testing.T, s *strB_stream, expected []strB_event) {
	t.Helper()
	defer func() { s.handler.len = 0 }()
	got := s.handler.events[0:s.handler.len]
	if len(got) != len(expected) {
		t.Fatalf("got %+v (len %d), want %+v (len %d)", got, len(got), expected, len(expected))
	}
	for i := range expected {
		if !reflect.DeepEqual(expected[i], got[i]) {
			t.Fatalf("got %+v, want %+v (first difference at index %d)", got, expected, i)
		}
	}
}

// strB_expectNone is term51BExpectNone.
func strB_expectNone(t *testing.T, s *strB_stream) {
	t.Helper()
	strB_expect(t, s, nil)
}

// strB_mk is term51BMk: `@unionInit(term51BEvent, @tagName(tag), @intCast(v))`.
func strB_mk(tag strB_kind, v uint16) strB_event {
	switch tag {
	case strB_insert_lines, strB_delete_lines, strB_delete_chars, strB_erase_chars,
		strB_scroll_up, strB_scroll_down, strB_print_repeat:
		return strB_event{kind: tag, usize: uint(v)}
	case strB_cursor_up, strB_cursor_down, strB_cursor_left, strB_cursor_right,
		strB_cursor_col, strB_cursor_row, strB_cursor_col_relative, strB_cursor_row_relative,
		strB_horizontal_tab, strB_horizontal_tab_back:
		return strB_event{kind: tag, u16: v}
	}
	panic("unreachable")
}

// strB_expectCount is term51BExpectCount.
func strB_expectCount(
	t *testing.T,
	s *strB_stream,
	tag strB_kind,
	v uint16,
	with_cr bool,
) {
	t.Helper()
	var buf [2]strB_event
	buf[0] = strB_mk(tag, v)
	var n uint = 1
	if with_cr {
		buf[1] = strB_none(strB_carriage_return)
		n = 2
	}
	strB_expect(t, s, buf[0:n])
}

// strB_eraseMode is one (param, tag) pair of term51BCheckErase's `valid`.
type strB_eraseMode struct {
	param uint16
	kind  strB_kind
}

// strB_modeCase is term51BModeCase.
type strB_modeCase struct {
	value uint16
	ansi  bool
	mode  ModesMode
}

// strB_mode_table is term51B_mode_table: every mode csiDispatch can map
// (modes.modeFromInt), as read from the entry table in modes.zig. The same
// number can be both an ANSI and a DEC mode (4, 12) with different meanings.
var strB_mode_table = []strB_modeCase{
	{2, true, ModesModeDisableKeyboard},
	{4, true, ModesModeInsert},
	{12, true, ModesModeSendReceiveMode},
	{20, true, ModesModeLinefeed},

	{1, false, ModesModeCursorKeys},
	{3, false, ModesMode132Column},
	{4, false, ModesModeSlowScroll},
	{5, false, ModesModeReverseColors},
	{6, false, ModesModeOrigin},
	{7, false, ModesModeWraparound},
	{8, false, ModesModeAutorepeat},
	{9, false, ModesModeMouseEventX10},
	{12, false, ModesModeCursorBlinking},
	{25, false, ModesModeCursorVisible},
	{40, false, ModesModeEnableMode3},
	{45, false, ModesModeReverseWrap},
	{47, false, ModesModeAltScreenLegacy},
	{66, false, ModesModeKeypadKeys},
	{69, false, ModesModeEnableLeftAndRightMargin},
	{1000, false, ModesModeMouseEventNormal},
	{1002, false, ModesModeMouseEventButton},
	{1003, false, ModesModeMouseEventAny},
	{1004, false, ModesModeFocusEvent},
	{1005, false, ModesModeMouseFormatUtf8},
	{1006, false, ModesModeMouseFormatSgr},
	{1007, false, ModesModeMouseAlternateScroll},
	{1015, false, ModesModeMouseFormatUrxvt},
	{1016, false, ModesModeMouseFormatSgrPixels},
	{1035, false, ModesModeIgnoreKeypadWithNumlock},
	{1036, false, ModesModeAltEscPrefix},
	{1039, false, ModesModeAltSendsEscape},
	{1045, false, ModesModeReverseWrapExtended},
	{1047, false, ModesModeAltScreen},
	{1048, false, ModesModeSaveCursor},
	{1049, false, ModesModeAltScreenSaveCursorClearEnter},
	{2004, false, ModesModeBracketedPaste},
	{2026, false, ModesModeSynchronizedOutput},
	{2027, false, ModesModeGraphemeCluster},
	{2031, false, ModesModeReportColorScheme},
	{2048, false, ModesModeInBandSizeReports},
}

// strB_modeLookup is term51BModeLookup.
func strB_modeLookup(value uint16, is_ansi bool) Optional[ModesMode] {
	for _, c := range strB_mode_table {
		if c.value == value && c.ansi == is_ansi {
			return Some(c.mode)
		}
	}
	return None[ModesMode]()
}

// strB_color_names is term51B_color_names.
var strB_color_names = [16]ColorName{
	ColorNameBlack,
	ColorNameRed,
	ColorNameGreen,
	ColorNameYellow,
	ColorNameBlue,
	ColorNameMagenta,
	ColorNameCyan,
	ColorNameWhite,
	ColorNameBrightBlack,
	ColorNameBrightRed,
	ColorNameBrightGreen,
	ColorNameBrightYellow,
	ColorNameBrightBlue,
	ColorNameBrightMagenta,
	ColorNameBrightCyan,
	ColorNameBrightWhite,
}

// strB_sgrSingle is term51BSgrSingle: the event expected for the SGR
// sequence `CSI p m` (one param).
func strB_sgrSingle(p uint16) strB_event {
	switch {
	case p == 0:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})
	case p == 1:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold})
	case p == 2:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagFaint})
	case p == 3:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic})
	case p == 4:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle})
	case p == 5 || p == 6:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagBlink})
	case p == 7:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagInverse})
	case p == 8:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagInvisible})
	case p == 9:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagStrikethrough})
	case p == 21:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineDouble})
	case p == 22:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetBold})
	case p == 23:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetItalic})
	case p == 24:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineNone})
	case p == 25:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetBlink})
	case p == 27:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetInverse})
	case p == 28:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetInvisible})
	case p == 29:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetStrikethrough})
	case p >= 30 && p <= 37:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: strB_color_names[p-30]})
	case p == 39:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetFg})
	case p >= 40 && p <= 47:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Bg, X8Bg: strB_color_names[p-40]})
	case p == 49:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetBg})
	case p == 53:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagOverline})
	case p == 55:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetOverline})
	case p == 59:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetUnderlineColor})
	case p >= 90 && p <= 97:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTag8BrightFg, X8BrightFg: strB_color_names[8+p-90]})
	case p >= 100 && p <= 107:
		return strB_sgr(SGRAttribute{Tag: SGRAttributeTag8BrightBg, X8BrightBg: strB_color_names[8+p-100]})
	default:
		// 10-20, 26, 38, 48 and 58 (which need sub-params), 50-52, 54,
		// 56-57, 60-89, 98-99 and everything above 107 are unknown.
		return strB_unkEv([]uint16{p}, []uint16{p})
	}
}

// strB_checkCount is term51BCheckCount: exercises a "single optional count"
// final (A, B, C, ..., b, d, e): no params means 1, one param is passed
// through unchanged (0 stays 0), two or more params and any intermediate
// emit nothing.
func strB_checkCount(
	t *testing.T,
	tag strB_kind,
	final uint8,
	withCR bool,
) {
	t.Helper()
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// Direct: no params -> default 1.
	strB_csi(t, &s, nil, nil, nil, final)
	strB_expectCount(t, &s, tag, 1, withCR)

	// Direct: one param, passed through (zero is NOT rewritten to 1).
	for _, v := range []uint16{0, 1, 2, 5, 255, 256, 65535} {
		strB_csi(t, &s, nil, []uint16{v}, nil, final)
		strB_expectCount(t, &s, tag, v, withCR)
	}

	// Direct: the colon separator bits are not looked at by this final.
	strB_csi(t, &s, nil, []uint16{5}, []uint{0}, final)
	strB_expectCount(t, &s, tag, 5, withCR)

	// Direct: two or more params are rejected (and nothing at all is
	// emitted, including the carriage return of CNL/CPL).
	strB_csi(t, &s, nil, []uint16{1, 1}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{0, 0}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{65535, 65535}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{1, 2, 3}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, strB_repeat(1, 24), nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, strB_repeat(1, 30), nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{1, 2}, []uint{0}, final)
	strB_expectNone(t, &s)

	// Direct: any intermediate rejects the sequence, with and without params.
	for _, c := range []uint8{'?', '>', '<', '=', ' ', '!', '"', '#', '$', '\''} {
		strB_csi(t, &s, []uint8{c}, nil, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{5}, nil, final)
		strB_expectNone(t, &s)
	}
	for _, inter := range [][]uint8{[]uint8("??"), []uint8("?>"), []uint8(" ?"), []uint8("  "), []uint8("!!!"), []uint8("????"), []uint8("$ \"")} {
		strB_csi(t, &s, inter, nil, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, []uint16{5}, nil, final)
		strB_expectNone(t, &s)
	}

	// Bytes through nextSlice.
	type Case struct {
		body  string
		value Optional[uint16]
	}
	cases := []Case{
		{body: "", value: Some[uint16](1)},
		{body: "0", value: Some[uint16](0)},
		{body: "1", value: Some[uint16](1)},
		{body: "5", value: Some[uint16](5)},
		{body: "12", value: Some[uint16](12)},
		{body: "65535", value: Some[uint16](65535)},
		// Saturating accumulator.
		{body: "65536", value: Some[uint16](65535)},
		{body: "99999999999999999999", value: Some[uint16](65535)},
		// An empty first param is a zero param.
		{body: ";", value: Some[uint16](0)},
		// A trailing empty param is not counted.
		{body: "5;", value: Some[uint16](5)},
		// Two params (one of them empty) are rejected.
		{body: ";5", value: None[uint16]()},
		{body: ";;", value: None[uint16]()},
		{body: "1;2", value: None[uint16]()},
		{body: "1;2;3", value: None[uint16]()},
		// Intermediates and private markers.
		{body: "?", value: None[uint16]()},
		{body: "?5", value: None[uint16]()},
		{body: ">5", value: None[uint16]()},
		{body: "<5", value: None[uint16]()},
		{body: "=5", value: None[uint16]()},
		{body: "5 ", value: None[uint16]()},
		{body: "5!", value: None[uint16]()},
		{body: "5\"", value: None[uint16]()},
		{body: "5$", value: None[uint16]()},
		{body: " ", value: None[uint16]()},
		// A private marker after a param ignores the whole sequence.
		{body: "5?", value: None[uint16]()},
		// Colon separators are dropped by the parser for non-m finals.
		{body: "1:2", value: None[uint16]()},
		{body: "5:", value: None[uint16]()},
		{body: ":", value: None[uint16]()},
	}
	for _, c := range cases {
		strB_feed(t, &s, c.body, final)
		if c.value.Valid {
			v := c.value.Value
			strB_expectCount(t, &s, tag, v, withCR)
		} else {
			strB_expectNone(t, &s)
		}
	}

	// 24 params is the parser's limit: it is delivered (and rejected by the
	// final), 25 is dropped by the parser. Neither emits anything.
	strB_feed(t, &s, "1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1", final)
	strB_expectNone(t, &s)

	// A sequence directly after another one is parsed independently.
	two := "\x1b[7" + string(rune(final)) + "\x1b[" + string(rune(final))
	zigTry(t, s.NextSlice([]uint8(two)))
	var expected [4]strB_event
	var n uint = 0
	expected[n] = strB_mk(tag, 7)
	n += 1
	if withCR {
		expected[n] = strB_none(strB_carriage_return)
		n += 1
	}
	expected[n] = strB_mk(tag, 1)
	n += 1
	if withCR {
		expected[n] = strB_none(strB_carriage_return)
		n += 1
	}
	strB_expect(t, &s, expected[0:n])
}

// Zig: test "term51: Stream.csiDispatch: CSI A cursor up (CUU)"
func TestStream_term51_Stream_csiDispatch_CSI_A_cursor_up_CUU(t *testing.T) {
	strB_checkCount(t, strB_cursor_up, 'A', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI k cursor up (alias of A)"
func TestStream_term51_Stream_csiDispatch_CSI_k_cursor_up_alias_of_A(t *testing.T) {
	strB_checkCount(t, strB_cursor_up, 'k', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI B cursor down (CUD)"
func TestStream_term51_Stream_csiDispatch_CSI_B_cursor_down_CUD(t *testing.T) {
	strB_checkCount(t, strB_cursor_down, 'B', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI C cursor right (CUF)"
func TestStream_term51_Stream_csiDispatch_CSI_C_cursor_right_CUF(t *testing.T) {
	strB_checkCount(t, strB_cursor_right, 'C', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI D cursor left (CUB)"
func TestStream_term51_Stream_csiDispatch_CSI_D_cursor_left_CUB(t *testing.T) {
	strB_checkCount(t, strB_cursor_left, 'D', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI j cursor left (alias of D)"
func TestStream_term51_Stream_csiDispatch_CSI_j_cursor_left_alias_of_D(t *testing.T) {
	strB_checkCount(t, strB_cursor_left, 'j', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI E cursor next line (CNL) is cursor down then carriage return"
func TestStream_term51_Stream_csiDispatch_CSI_E_cursor_next_line_CNL_is_cursor_down_then_carriage_return(t *testing.T) {
	strB_checkCount(t, strB_cursor_down, 'E', true)
}

// Zig: test "term51: Stream.csiDispatch: CSI F cursor previous line (CPL) is cursor up then carriage return"
func TestStream_term51_Stream_csiDispatch_CSI_F_cursor_previous_line_CPL_is_cursor_up_then_carriage_return(t *testing.T) {
	strB_checkCount(t, strB_cursor_up, 'F', true)
}

// Zig: test "term51: Stream.csiDispatch: CSI G cursor column (CHA)"
func TestStream_term51_Stream_csiDispatch_CSI_G_cursor_column_CHA(t *testing.T) {
	strB_checkCount(t, strB_cursor_col, 'G', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI backtick cursor column (HPA)"
func TestStream_term51_Stream_csiDispatch_CSI_backtick_cursor_column_HPA(t *testing.T) {
	strB_checkCount(t, strB_cursor_col, '`', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI I cursor horizontal tab (CHT)"
func TestStream_term51_Stream_csiDispatch_CSI_I_cursor_horizontal_tab_CHT(t *testing.T) {
	strB_checkCount(t, strB_horizontal_tab, 'I', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI L insert lines (IL)"
func TestStream_term51_Stream_csiDispatch_CSI_L_insert_lines_IL(t *testing.T) {
	strB_checkCount(t, strB_insert_lines, 'L', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI M delete lines (DL)"
func TestStream_term51_Stream_csiDispatch_CSI_M_delete_lines_DL(t *testing.T) {
	strB_checkCount(t, strB_delete_lines, 'M', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI P delete chars (DCH)"
func TestStream_term51_Stream_csiDispatch_CSI_P_delete_chars_DCH(t *testing.T) {
	strB_checkCount(t, strB_delete_chars, 'P', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI S scroll up"
func TestStream_term51_Stream_csiDispatch_CSI_S_scroll_up(t *testing.T) {
	strB_checkCount(t, strB_scroll_up, 'S', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI T scroll down"
func TestStream_term51_Stream_csiDispatch_CSI_T_scroll_down(t *testing.T) {
	strB_checkCount(t, strB_scroll_down, 'T', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI X erase chars (ECH)"
func TestStream_term51_Stream_csiDispatch_CSI_X_erase_chars_ECH(t *testing.T) {
	strB_checkCount(t, strB_erase_chars, 'X', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI Z horizontal tab back (CBT)"
func TestStream_term51_Stream_csiDispatch_CSI_Z_horizontal_tab_back_CBT(t *testing.T) {
	strB_checkCount(t, strB_horizontal_tab_back, 'Z', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI a cursor column relative (HPR)"
func TestStream_term51_Stream_csiDispatch_CSI_a_cursor_column_relative_HPR(t *testing.T) {
	strB_checkCount(t, strB_cursor_col_relative, 'a', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI b repeat previous char (REP)"
func TestStream_term51_Stream_csiDispatch_CSI_b_repeat_previous_char_REP(t *testing.T) {
	strB_checkCount(t, strB_print_repeat, 'b', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI d cursor row (VPA)"
func TestStream_term51_Stream_csiDispatch_CSI_d_cursor_row_VPA(t *testing.T) {
	strB_checkCount(t, strB_cursor_row, 'd', false)
}

// Zig: test "term51: Stream.csiDispatch: CSI e cursor row relative (VPR)"
func TestStream_term51_Stream_csiDispatch_CSI_e_cursor_row_relative_VPR(t *testing.T) {
	strB_checkCount(t, strB_cursor_row_relative, 'e', false)
}

// strB_checkCursorPos is term51BCheckCursorPos.
func strB_checkCursorPos(t *testing.T, final uint8) {
	t.Helper()
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// Direct: 0, 1 and 2 params. Missing means 1; a zero is passed through.
	strB_csi(t, &s, nil, nil, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(1, 1)})
	strB_csi(t, &s, nil, []uint16{0}, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(0, 1)})
	strB_csi(t, &s, nil, []uint16{5}, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(5, 1)})
	strB_csi(t, &s, nil, []uint16{5, 7}, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(5, 7)})
	strB_csi(t, &s, nil, []uint16{0, 0}, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(0, 0)})
	strB_csi(t, &s, nil, []uint16{0, 9}, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(0, 9)})
	strB_csi(t, &s, nil, []uint16{9, 0}, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(9, 0)})
	strB_csi(t, &s, nil, []uint16{65535}, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(65535, 1)})
	strB_csi(t, &s, nil, []uint16{65535, 65535}, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(65535, 65535)})
	strB_csi(t, &s, nil, []uint16{256, 255}, nil, final)
	strB_expect(t, &s, []strB_event{strB_pos(256, 255)})

	// Direct: the colon bits are not looked at.
	strB_csi(t, &s, nil, []uint16{3, 4}, []uint{0}, final)
	strB_expect(t, &s, []strB_event{strB_pos(3, 4)})

	// Direct: three or more params are rejected.
	strB_csi(t, &s, nil, []uint16{1, 2, 3}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{0, 0, 0}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, strB_repeat(1, 24), nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, strB_repeat(1, 30), nil, final)
	strB_expectNone(t, &s)

	// Direct: any intermediate rejects the sequence.
	for _, c := range []uint8{'?', '>', '<', '=', ' ', '!', '"', '#', '$', '\''} {
		strB_csi(t, &s, []uint8{c}, nil, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{2, 3}, nil, final)
		strB_expectNone(t, &s)
	}
	for _, inter := range [][]uint8{[]uint8("??"), []uint8("?>"), []uint8(" ?"), []uint8("  "), []uint8("!!!"), []uint8("????")} {
		strB_csi(t, &s, inter, []uint16{2, 3}, nil, final)
		strB_expectNone(t, &s)
	}

	// Bytes.
	type Case struct {
		body string
		pos  Optional[[2]uint16]
	}
	cases := []Case{
		{body: "", pos: Some([2]uint16{1, 1})},
		{body: "0", pos: Some([2]uint16{0, 1})},
		{body: "5", pos: Some([2]uint16{5, 1})},
		{body: "5;7", pos: Some([2]uint16{5, 7})},
		{body: "0;0", pos: Some([2]uint16{0, 0})},
		{body: "65535;65535", pos: Some([2]uint16{65535, 65535})},
		{body: "99999;70000", pos: Some([2]uint16{65535, 65535})},
		// Empty params count as zero, except a trailing one.
		{body: ";", pos: Some([2]uint16{0, 1})},
		{body: "5;", pos: Some([2]uint16{5, 1})},
		{body: ";7", pos: Some([2]uint16{0, 7})},
		{body: ";;", pos: Some([2]uint16{0, 0})},
		{body: "1;2;3", pos: None[[2]uint16]()},
		{body: ";;;", pos: None[[2]uint16]()},
		{body: "?1;2", pos: None[[2]uint16]()},
		{body: ">", pos: None[[2]uint16]()},
		{body: "1;2 ", pos: None[[2]uint16]()},
		{body: "1;2$", pos: None[[2]uint16]()},
		{body: "1:2", pos: None[[2]uint16]()},
		{body: "1;2:3", pos: None[[2]uint16]()},
	}
	for _, c := range cases {
		strB_feed(t, &s, c.body, final)
		if c.pos.Valid {
			p := c.pos.Value
			strB_expect(t, &s, []strB_event{strB_pos(p[0], p[1])})
		} else {
			strB_expectNone(t, &s)
		}
	}

	// 24 params are delivered and rejected, 25 are dropped by the parser.
	strB_feed(t, &s, "1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1", final)
	strB_expectNone(t, &s)
}

// Zig: test "term51: Stream.csiDispatch: CSI H cursor position (CUP)"
func TestStream_term51_Stream_csiDispatch_CSI_H_cursor_position_CUP(t *testing.T) {
	strB_checkCursorPos(t, 'H')
}

// Zig: test "term51: Stream.csiDispatch: CSI f cursor position (HVP, alias of H)"
func TestStream_term51_Stream_csiDispatch_CSI_f_cursor_position_HVP_alias_of_H(t *testing.T) {
	strB_checkCursorPos(t, 'f')
}

// strB_checkErase is term51BCheckErase: erase display/line tables are
// (param, tag) pairs; this runs one erase final over both the unprotected
// (no intermediate) and protected ('?') forms.
func strB_checkErase(
	t *testing.T,
	final uint8,
	valid []strB_eraseMode,
	default_tag strB_kind,
	invalid []uint16,
) {
	t.Helper()
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	for _, protected := range []bool{false, true} {
		inter := []uint8(nil)
		sigil := ""
		if protected {
			inter = []uint8("?")
			sigil = "?"
		}

		// No params: the default mode.
		strB_csi(t, &s, inter, nil, nil, final)
		strB_expect(t, &s, []strB_event{strB_boolEv(default_tag, protected)})
		strB_feed(t, &s, sigil, final)
		strB_expect(t, &s, []strB_event{strB_boolEv(default_tag, protected)})

		// Each valid mode.
		for _, v := range valid {
			strB_csi(t, &s, inter, []uint16{v.param}, nil, final)
			strB_expect(t, &s, []strB_event{strB_boolEv(v.kind, protected)})

			body := sigil + strconv.Itoa(int(v.param))
			strB_feed(t, &s, body, final)
			strB_expect(t, &s, []strB_event{strB_boolEv(v.kind, protected)})
		}

		// Each invalid mode, including the u16 maximum and values that only
		// match when truncated to 8 bits.
		for _, v := range invalid {
			strB_csi(t, &s, inter, []uint16{v}, nil, final)
			strB_expectNone(t, &s)

			body := sigil + strconv.Itoa(int(v))
			strB_feed(t, &s, body, final)
			strB_expectNone(t, &s)
		}

		// Two or more params are rejected, even if each is valid.
		strB_csi(t, &s, inter, []uint16{0, 0}, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, []uint16{1, 2}, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, strB_repeat(0, 24), nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, strB_repeat(0, 30), nil, final)
		strB_expectNone(t, &s)
		strB_feed(t, &s, strB_ifs(protected, "?1;2", "1;2"), final)
		strB_expectNone(t, &s)
		strB_feed(t, &s, strB_ifs(protected, "?;", ";"), final)
		// `;` alone is one zero param: default mode.
		strB_expect(t, &s, []strB_event{strB_boolEv(valid[0].kind, protected)})
		strB_feed(t, &s, strB_ifs(protected, "?;;", ";;"), final)
		strB_expectNone(t, &s)
	}

	// Bare sequences through bytes.
	strB_feed(t, &s, "", final)
	strB_expect(t, &s, []strB_event{strB_boolEv(default_tag, false)})

	// Any other intermediate (or more than one) is rejected whatever the params.
	for _, c := range []uint8{'>', '<', '=', ' ', '!', '"', '#', '$', '\''} {
		strB_csi(t, &s, []uint8{c}, nil, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{0}, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{2}, nil, final)
		strB_expectNone(t, &s)
	}
	for _, inter := range [][]uint8{[]uint8("??"), []uint8("?>"), []uint8(">?"), []uint8(" ?"), []uint8("? "), []uint8("  "), []uint8("!!!"), []uint8("????")} {
		strB_csi(t, &s, inter, nil, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, []uint16{0}, nil, final)
		strB_expectNone(t, &s)
	}

	// Colon bits are not looked at by this final when called directly.
	strB_csi(t, &s, nil, []uint16{valid[0].param}, []uint{0}, final)
	strB_expect(t, &s, []strB_event{strB_boolEv(valid[0].kind, false)})

	// Colon separators never reach it through the parser.
	strB_feed(t, &s, "1:2", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "0:", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?0:", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "5 ", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "0$", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, ">0", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "0?", final)
	strB_expectNone(t, &s)
}

// Zig: test "term51: Stream.csiDispatch: CSI J erase display (ED, DECSED)"
func TestStream_term51_Stream_csiDispatch_CSI_J_erase_display_ED_DECSED(t *testing.T) {
	strB_checkErase(t, 'J',
		[]strB_eraseMode{
			{0, strB_erase_display_below},
			{1, strB_erase_display_above},
			{2, strB_erase_display_complete},
			{3, strB_erase_display_scrollback},
			{22, strB_erase_display_scroll_complete},
		},
		strB_erase_display_below,
		[]uint16{4, 5, 6, 10, 21, 23, 24, 100, 255, 256, 278, 65535},
	)
}

// Zig: test "term51: Stream.csiDispatch: CSI K erase line (EL, DECSEL)"
func TestStream_term51_Stream_csiDispatch_CSI_K_erase_line_EL_DECSEL(t *testing.T) {
	// Mode 4 (right_unless_pending_wrap) exists in csi.EraseLine but the
	// dispatcher only accepts params below 3, so it is rejected here.
	strB_checkErase(t, 'K',
		[]strB_eraseMode{
			{0, strB_erase_line_right},
			{1, strB_erase_line_left},
			{2, strB_erase_line_complete},
		},
		strB_erase_line_right,
		[]uint16{3, 4, 5, 6, 22, 255, 256, 258, 65535},
	)
}

// Zig: test "term51: Stream.csiDispatch: CSI W cursor tabulation control without intermediates"
func TestStream_term51_Stream_csiDispatch_CSI_W_cursor_tabulation_control_without_intermediates(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// No params and a single zero param set a tab stop.
	strB_csi(t, &s, nil, nil, nil, 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_set)})
	strB_csi(t, &s, nil, []uint16{0}, nil, 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_set)})

	// 2 clears the current stop, 5 clears all of them.
	strB_csi(t, &s, nil, []uint16{2}, nil, 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_current)})
	strB_csi(t, &s, nil, []uint16{5}, nil, 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_all)})

	// Every other single value is rejected.
	for _, v := range []uint16{1, 3, 4, 6, 7, 22, 255, 256, 258, 261, 65535} {
		strB_csi(t, &s, nil, []uint16{v}, nil, 'W')
		strB_expectNone(t, &s)
	}

	// Two or more params are rejected, even valid ones.
	strB_csi(t, &s, nil, []uint16{0, 0}, nil, 'W')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{2, 2}, nil, 'W')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{5, 5}, nil, 'W')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{0, 5}, nil, 'W')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{2, 0}, nil, 'W')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, strB_repeat(0, 24), nil, 'W')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, strB_repeat(2, 30), nil, 'W')
	strB_expectNone(t, &s)

	// The colon bits are not looked at when called directly.
	strB_csi(t, &s, nil, []uint16{2}, []uint{0}, 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_current)})

	// Bytes.
	strB_feed(t, &s, "", 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_set)})
	strB_feed(t, &s, "0", 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_set)})
	strB_feed(t, &s, ";", 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_set)})
	strB_feed(t, &s, "2", 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_current)})
	strB_feed(t, &s, "5", 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_all)})
	strB_feed(t, &s, "2;", 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_current)})
	strB_feed(t, &s, "1", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "65535", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, ";2", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "0;0", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "2:5", 'W')
	strB_expectNone(t, &s)
}

// Zig: test "term51: Stream.csiDispatch: CSI ? W resets tab stops and every other intermediate is rejected"
func TestStream_term51_Stream_csiDispatch_CSI_W_resets_tab_stops_and_every_other_intermediate_is_rejected(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// Only "?" with exactly one param equal to 5.
	strB_csi(t, &s, []uint8("?"), []uint16{5}, nil, 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_reset)})
	strB_csi(t, &s, []uint8("?"), []uint16{5}, []uint{0}, 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_reset)})

	strB_csi(t, &s, []uint8("?"), nil, nil, 'W')
	strB_expectNone(t, &s)
	for _, v := range []uint16{0, 1, 2, 3, 4, 6, 255, 261, 65535} {
		strB_csi(t, &s, []uint8("?"), []uint16{v}, nil, 'W')
		strB_expectNone(t, &s)
	}
	strB_csi(t, &s, []uint8("?"), []uint16{5, 5}, nil, 'W')
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8("?"), []uint16{5, 0}, nil, 'W')
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8("?"), []uint16{0, 5}, nil, 'W')
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8("?"), strB_repeat(5, 24), nil, 'W')
	strB_expectNone(t, &s)

	// Other single intermediates (even with the "right" param) are rejected.
	for _, c := range []uint8{'>', '<', '=', ' ', '!', '"', '#', '$', '\''} {
		strB_csi(t, &s, []uint8{c}, []uint16{5}, nil, 'W')
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, nil, nil, 'W')
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{0}, nil, 'W')
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{2}, nil, 'W')
		strB_expectNone(t, &s)
	}

	// Two or more intermediates are rejected.
	for _, inter := range [][]uint8{[]uint8("??"), []uint8("?>"), []uint8(">?"), []uint8(" ?"), []uint8("? "), []uint8("  "), []uint8("!!!"), []uint8("????")} {
		strB_csi(t, &s, inter, []uint16{5}, nil, 'W')
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, nil, nil, 'W')
		strB_expectNone(t, &s)
	}

	// Bytes.
	strB_feed(t, &s, "?5", 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_reset)})
	strB_feed(t, &s, "?", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?0", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?2", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?5;5", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?1;2;3", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?5;", 'W')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_reset)})
	strB_feed(t, &s, "?;5", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, ">5", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "5 ", 'W')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?5:", 'W')
	strB_expectNone(t, &s)
}

// Zig: test "term51: Stream.csiDispatch: CSI g tab clear (TBC)"
func TestStream_term51_Stream_csiDispatch_CSI_g_tab_clear_TBC(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// 0 clears the current stop, 3 clears all.
	strB_csi(t, &s, nil, []uint16{0}, nil, 'g')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_current)})
	strB_csi(t, &s, nil, []uint16{3}, nil, 'g')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_all)})

	// Unlike most finals there is no default: no params emits nothing.
	strB_csi(t, &s, nil, nil, nil, 'g')
	strB_expectNone(t, &s)

	// Every other value is rejected: in range of the u8 backing enum (the
	// non-exhaustive "_" prong), and out of range of it.
	for _, v := range []uint16{1, 2, 4, 5, 6, 22, 254, 255, 256, 259, 65535} {
		strB_csi(t, &s, nil, []uint16{v}, nil, 'g')
		strB_expectNone(t, &s)
	}

	// More than one param is rejected, even valid values.
	strB_csi(t, &s, nil, []uint16{0, 0}, nil, 'g')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{3, 3}, nil, 'g')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{0, 3}, nil, 'g')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, strB_repeat(3, 24), nil, 'g')
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, strB_repeat(3, 30), nil, 'g')
	strB_expectNone(t, &s)

	// The colon bits are not looked at when called directly.
	strB_csi(t, &s, nil, []uint16{3}, []uint{0}, 'g')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_all)})

	// Any intermediate rejects it.
	for _, c := range []uint8{'?', '>', '<', '=', ' ', '!', '"', '#', '$', '\''} {
		strB_csi(t, &s, []uint8{c}, []uint16{0}, nil, 'g')
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{3}, nil, 'g')
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, nil, nil, 'g')
		strB_expectNone(t, &s)
	}
	for _, inter := range [][]uint8{[]uint8("??"), []uint8("?>"), []uint8(" ?"), []uint8("  "), []uint8("!!!"), []uint8("????")} {
		strB_csi(t, &s, inter, []uint16{3}, nil, 'g')
		strB_expectNone(t, &s)
	}

	// Bytes.
	strB_feed(t, &s, "0", 'g')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_current)})
	strB_feed(t, &s, "3", 'g')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_all)})
	strB_feed(t, &s, "", 'g')
	strB_expectNone(t, &s)
	strB_feed(t, &s, ";", 'g')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_current)})
	strB_feed(t, &s, "3;", 'g')
	strB_expect(t, &s, []strB_event{strB_none(strB_tab_clear_all)})
	strB_feed(t, &s, "3;3", 'g')
	strB_expectNone(t, &s)
	strB_feed(t, &s, ";3", 'g')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "1", 'g')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "259", 'g')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "65535", 'g')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "99999999999999999999", 'g')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?3", 'g')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "3 ", 'g')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "3:", 'g')
	strB_expectNone(t, &s)
}

// Zig: test "term51: Stream.csiDispatch: CSI c device attributes (DA1, DA2, DA3)"
func TestStream_term51_Stream_csiDispatch_CSI_c_device_attributes_DA1_DA2_DA3(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// No intermediate is primary, ">" secondary and "=" tertiary. The
	// params are never looked at, whatever they are.
	param_sets := [][]uint16{
		{},
		{0},
		{1},
		{1, 2, 3},
		{65535},
		strB_repeat(9, 24),
		strB_repeat(9, 30),
	}
	for _, params := range param_sets {
		strB_csi(t, &s, nil, params, nil, 'c')
		strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqPrimary)})
		strB_csi(t, &s, []uint8(">"), params, nil, 'c')
		strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqSecondary)})
		strB_csi(t, &s, []uint8("="), params, nil, 'c')
		strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqTertiary)})
	}

	// Colon bits are not looked at.
	strB_csi(t, &s, []uint8(">"), []uint16{1, 2}, []uint{0}, 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqSecondary)})

	// Any other single intermediate is rejected.
	for _, c := range []uint8{'?', '<', ' ', '!', '"', '#', '$', '\''} {
		strB_csi(t, &s, []uint8{c}, nil, nil, 'c')
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{0}, nil, 'c')
		strB_expectNone(t, &s)
	}

	// More than one intermediate is rejected, including repeats of valid ones.
	for _, inter := range [][]uint8{[]uint8(">>"), []uint8("=="), []uint8(">="), []uint8("=>"), []uint8("?>"), []uint8(">?"), []uint8(" >"), []uint8("> "), []uint8("  "), []uint8(">>>"), []uint8(">>>>")} {
		strB_csi(t, &s, inter, nil, nil, 'c')
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, []uint16{0}, nil, 'c')
		strB_expectNone(t, &s)
	}

	// Bytes.
	strB_feed(t, &s, "", 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqPrimary)})
	strB_feed(t, &s, "0", 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqPrimary)})
	strB_feed(t, &s, "1;2;3", 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqPrimary)})
	strB_feed(t, &s, ">", 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqSecondary)})
	strB_feed(t, &s, ">0", 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqSecondary)})
	strB_feed(t, &s, ">1;2", 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqSecondary)})
	strB_feed(t, &s, "=", 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqTertiary)})
	strB_feed(t, &s, "=0", 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqTertiary)})
	strB_feed(t, &s, "?", 'c')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?0", 'c')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "<", 'c')
	strB_expectNone(t, &s)
	strB_feed(t, &s, " ", 'c')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "0 ", 'c')
	strB_expectNone(t, &s)
	strB_feed(t, &s, ">>", 'c')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "0>", 'c')
	strB_expectNone(t, &s)
	// Colon separators are dropped by the parser for non-m finals.
	strB_feed(t, &s, "0:1", 'c')
	strB_expectNone(t, &s)
	// 24 params still dispatch, 25 are dropped by the parser.
	strB_feed(t, &s, "1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1", 'c')
	strB_expect(t, &s, []strB_event{strB_da(ANSIDeviceAttributeReqPrimary)})
	strB_feed(t, &s, "1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1", 'c')
	strB_expectNone(t, &s)
}

// strB_checkModeFinal is term51BCheckModeFinal.
func strB_checkModeFinal(t *testing.T, final uint8) {
	t.Helper()
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// Sanity: the table has 4 ANSI and 36 DEC modes.
	var n_ansi uint
	for _, c := range strB_mode_table {
		if c.ansi {
			n_ansi += 1
		}
	}
	expectEqual(t, uint(4), n_ansi)
	expectEqual(t, uint(40), uint(len(strB_mode_table)))

	// Every mapped mode, both through a direct dispatch and through bytes.
	for _, c := range strB_mode_table {
		inter := []uint8("?")
		if c.ansi {
			inter = nil
		}
		strB_csi(t, &s, inter, []uint16{c.value}, nil, final)
		strB_expect(t, &s, []strB_event{strB_modeEvent(final, c.mode)})

		body := string(inter) + strconv.Itoa(int(c.value))
		strB_feed(t, &s, body, final)
		strB_expect(t, &s, []strB_event{strB_modeEvent(final, c.mode)})
	}

	// Sweep of every number from 0 to 2100 (covers each mapped value, the
	// values around them and all the gaps) for both the ANSI and the DEC
	// namespace: emitted exactly when the table has the (value, namespace).
	for v := uint16(0); v <= 2100; v++ {
		for _, is_ansi := range []bool{true, false} {
			inter := []uint8("?")
			if is_ansi {
				inter = nil
			}
			strB_csi(t, &s, inter, []uint16{v}, nil, final)
			if m := strB_modeLookup(v, is_ansi); m.Valid {
				strB_expect(t, &s, []strB_event{strB_modeEvent(final, m.Value)})
			} else {
				strB_expectNone(t, &s)
			}
		}
	}

	// Explicit unknown numbers in both namespaces, including the u16 maximum.
	for _, u := range []uint16{0, 10, 11, 13, 19, 21, 22, 2005, 2049, 4096, 32768, 65534, 65535} {
		strB_csi(t, &s, nil, []uint16{u}, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8("?"), []uint16{u}, nil, final)
		strB_expectNone(t, &s)
	}

	// The same number maps differently in the two namespaces, and numbers
	// that exist in only one namespace are unknown in the other.
	strB_csi(t, &s, nil, []uint16{4}, nil, final)
	strB_expect(t, &s, []strB_event{strB_modeEvent(final, ModesModeInsert)})
	strB_csi(t, &s, []uint8("?"), []uint16{4}, nil, final)
	strB_expect(t, &s, []strB_event{strB_modeEvent(final, ModesModeSlowScroll)})
	strB_csi(t, &s, nil, []uint16{12}, nil, final)
	strB_expect(t, &s, []strB_event{strB_modeEvent(final, ModesModeSendReceiveMode)})
	strB_csi(t, &s, []uint8("?"), []uint16{12}, nil, final)
	strB_expect(t, &s, []strB_event{strB_modeEvent(final, ModesModeCursorBlinking)})
	strB_csi(t, &s, []uint8("?"), []uint16{2}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8("?"), []uint16{20}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{25}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{1}, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, nil, []uint16{2004}, nil, final)
	strB_expectNone(t, &s)

	// No params: nothing, with and without "?".
	strB_csi(t, &s, nil, nil, nil, final)
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8("?"), nil, nil, final)
	strB_expectNone(t, &s)

	// Several modes in one sequence are emitted in order; unknown numbers
	// are skipped without stopping the sequence; repeats are emitted again.
	strB_csi(t, &s, []uint8("?"), []uint16{1, 25, 99, 2004, 3, 0, 25}, nil, final)
	strB_expect(t, &s, []strB_event{
		strB_modeEvent(final, ModesModeCursorKeys),
		strB_modeEvent(final, ModesModeCursorVisible),
		strB_modeEvent(final, ModesModeBracketedPaste),
		strB_modeEvent(final, ModesMode132Column),
		strB_modeEvent(final, ModesModeCursorVisible),
	})
	strB_csi(t, &s, nil, []uint16{4, 2, 99, 20, 12, 5, 1, 4}, nil, final)
	strB_expect(t, &s, []strB_event{
		strB_modeEvent(final, ModesModeInsert),
		strB_modeEvent(final, ModesModeDisableKeyboard),
		strB_modeEvent(final, ModesModeLinefeed),
		strB_modeEvent(final, ModesModeSendReceiveMode),
		strB_modeEvent(final, ModesModeInsert),
	})
	// 24 and 30 params: every param is processed.
	strB_csi(t, &s, []uint8("?"), strB_repeat(25, 24), nil, final)
	expectEqual(t, uint(24), s.handler.len)
	for _, e := range s.handler.events[0:24] {
		strB_expectEqualDeep(t, strB_modeEvent(final, ModesModeCursorVisible), e)
	}
	s.handler.len = 0
	strB_csi(t, &s, nil, strB_repeat(4, 30), nil, final)
	expectEqual(t, uint(30), s.handler.len)
	for _, e := range s.handler.events[0:30] {
		strB_expectEqualDeep(t, strB_modeEvent(final, ModesModeInsert), e)
	}
	s.handler.len = 0

	// The colon bits are not looked at when called directly.
	strB_csi(t, &s, []uint8("?"), []uint16{25, 1}, []uint{0}, final)
	strB_expect(t, &s, []strB_event{
		strB_modeEvent(final, ModesModeCursorVisible),
		strB_modeEvent(final, ModesModeCursorKeys),
	})

	// Any intermediate other than a single "?" ignores the whole sequence,
	// even when every param would be a valid mode.
	for _, c := range []uint8{'>', '<', '=', ' ', '!', '"', '#', '$', '\''} {
		strB_csi(t, &s, []uint8{c}, []uint16{4}, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{1, 25}, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, nil, nil, final)
		strB_expectNone(t, &s)
	}
	for _, inter := range [][]uint8{[]uint8("??"), []uint8("?>"), []uint8(">?"), []uint8(" ?"), []uint8("? "), []uint8("  "), []uint8("!!!"), []uint8("????")} {
		strB_csi(t, &s, inter, []uint16{4, 25}, nil, final)
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, nil, nil, final)
		strB_expectNone(t, &s)
	}

	// Bytes: multiple modes, empty params, parser limits, rejected forms.
	strB_feed(t, &s, "?1;25;99;2004;3", final)
	strB_expect(t, &s, []strB_event{
		strB_modeEvent(final, ModesModeCursorKeys),
		strB_modeEvent(final, ModesModeCursorVisible),
		strB_modeEvent(final, ModesModeBracketedPaste),
		strB_modeEvent(final, ModesMode132Column),
	})
	strB_feed(t, &s, "4;2;99;20;12;5", final)
	strB_expect(t, &s, []strB_event{
		strB_modeEvent(final, ModesModeInsert),
		strB_modeEvent(final, ModesModeDisableKeyboard),
		strB_modeEvent(final, ModesModeLinefeed),
		strB_modeEvent(final, ModesModeSendReceiveMode),
	})
	// Empty params are zeros (unknown), a trailing empty one is not counted.
	strB_feed(t, &s, ";4", final)
	strB_expect(t, &s, []strB_event{strB_modeEvent(final, ModesModeInsert)})
	strB_feed(t, &s, "?25;", final)
	strB_expect(t, &s, []strB_event{strB_modeEvent(final, ModesModeCursorVisible)})
	strB_feed(t, &s, "?;;", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?", final)
	strB_expectNone(t, &s)
	// Saturated values are unknown.
	strB_feed(t, &s, "?99999999999", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?65536", final)
	strB_expectNone(t, &s)
	// Values that would only match with 16 bit wrap-around are not modes.
	strB_feed(t, &s, "?65537", final)
	strB_expectNone(t, &s)
	// Rejected prefixes and suffixes.
	strB_feed(t, &s, ">5", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "<4", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "=4", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "6 ", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?6 ", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "4$", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "4!", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "4\"", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "4?", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "??4", final)
	strB_expectNone(t, &s)
	// Colon separators are dropped by the parser for non-m finals.
	strB_feed(t, &s, "?25:1", final)
	strB_expectNone(t, &s)
	strB_feed(t, &s, "4:", final)
	strB_expectNone(t, &s)
	// 24 params dispatch, 25 are dropped by the parser.
	strB_feed(t, &s, "?25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25", final)
	expectEqual(t, uint(24), s.handler.len)
	s.handler.len = 0
	strB_feed(t, &s, "?25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25", final)
	strB_expectNone(t, &s)
}

// Zig: test "term51: Stream.csiDispatch: CSI h set mode (SM, DECSET)"
func TestStream_term51_Stream_csiDispatch_CSI_h_set_mode_SM_DECSET(t *testing.T) {
	strB_checkModeFinal(t, 'h')
}

// Zig: test "term51: Stream.csiDispatch: CSI l reset mode (RM, DECRST)"
func TestStream_term51_Stream_csiDispatch_CSI_l_reset_mode_RM_DECRST(t *testing.T) {
	strB_checkModeFinal(t, 'l')
}

// Zig: test "term51: Stream.csiDispatch: CSI m with no params and zero params is a single unset"
func TestStream_term51_Stream_csiDispatch_CSI_m_with_no_params_and_zero_params_is_a_single_unset(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	strB_csi(t, &s, nil, nil, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})})
	strB_csi(t, &s, nil, []uint16{0}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})})
	strB_csi(t, &s, nil, []uint16{0, 0}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})})
	strB_csi(t, &s, nil, []uint16{0, 1, 0}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})})

	strB_feed(t, &s, "", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})})
	strB_feed(t, &s, "0", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})})
	// A lone separator is one zero param.
	strB_feed(t, &s, ";", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})})
	strB_feed(t, &s, ";;", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})})
	strB_feed(t, &s, "1;;3", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic})})
	strB_feed(t, &s, "1;", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold})})
	strB_feed(t, &s, ";1", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold})})
}

// Zig: test "term51: Stream.csiDispatch: CSI m single params 0 to 110 each emit one attribute"
func TestStream_term51_Stream_csiDispatch_CSI_m_single_params_0_to_110_each_emit_one_attribute(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	for p := uint16(0); p <= 110; p++ {
		strB_csi(t, &s, nil, []uint16{p}, nil, 'm')
		strB_expect(t, &s, []strB_event{strB_sgrSingle(p)})

		body := strconv.Itoa(int(p))
		strB_feed(t, &s, body, 'm')
		strB_expect(t, &s, []strB_event{strB_sgrSingle(p)})
	}

	// Spot checks of the table itself, so a bug in it cannot hide.
	strB_csi(t, &s, nil, []uint16{31}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: ColorNameRed})})
	strB_csi(t, &s, nil, []uint16{47}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Bg, X8Bg: ColorNameWhite})})
	strB_csi(t, &s, nil, []uint16{90}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag8BrightFg, X8BrightFg: ColorNameBrightBlack})})
	strB_csi(t, &s, nil, []uint16{107}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag8BrightBg, X8BrightBg: ColorNameBrightWhite})})
	strB_csi(t, &s, nil, []uint16{38}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{38}, []uint16{38})})
	strB_csi(t, &s, nil, []uint16{108}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{108}, []uint16{108})})

	// Large and saturated values are unknown.
	strB_csi(t, &s, nil, []uint16{65535}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{65535}, []uint16{65535})})
	strB_feed(t, &s, "65535", 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{65535}, []uint16{65535})})
	strB_feed(t, &s, "99999999999", 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{65535}, []uint16{65535})})
	strB_feed(t, &s, "65536", 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{65535}, []uint16{65535})})
	strB_feed(t, &s, "256", 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{256}, []uint16{256})})
	strB_feed(t, &s, "1001", 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{1001}, []uint16{1001})})
}

// Zig: test "term51: Stream.csiDispatch: CSI m chunks multiple params into one set_attribute each"
func TestStream_term51_Stream_csiDispatch_CSI_m_chunks_multiple_params_into_one_set_attribute_each(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// Plain attributes in order.
	strB_csi(t, &s, nil, []uint16{1, 3, 4, 31}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: ColorNameRed}),
	})
	strB_csi(t, &s, nil, []uint16{0, 1}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold})})
	strB_csi(t, &s, nil, []uint16{1, 0}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset})})
	// Semicolon separated 4;3 is underline then italic, NOT curly underline.
	strB_csi(t, &s, nil, []uint16{4, 3}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle}), strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic})})
	strB_csi(t, &s, nil, []uint16{4, 5, 3}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBlink}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic}),
	})
	// Resets.
	strB_csi(t, &s, nil, []uint16{22, 23, 24, 25, 27, 28, 29, 39, 49, 55, 59}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetItalic}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineNone}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetBlink}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetInverse}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetInvisible}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetStrikethrough}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetFg}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetBg}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetOverline}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagResetUnderlineColor}),
	})

	// 256 color forms consume three params.
	strB_csi(t, &s, nil, []uint16{38, 5, 200}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 200})})
	strB_csi(t, &s, nil, []uint16{48, 5, 100}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 100})})
	strB_csi(t, &s, nil, []uint16{58, 5, 7}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 7})})
	strB_csi(t, &s, nil, []uint16{38, 5, 200, 48, 5, 100, 58, 5, 7, 1}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 200}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 100}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 7}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
	})
	// 256 index values are truncated to 8 bits.
	strB_csi(t, &s, nil, []uint16{38, 5, 255}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 255})})
	strB_csi(t, &s, nil, []uint16{38, 5, 256}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 0})})
	strB_csi(t, &s, nil, []uint16{48, 5, 300}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 44})})
	strB_csi(t, &s, nil, []uint16{58, 5, 65535}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 255})})

	// Direct color forms consume five params.
	strB_csi(t, &s, nil, []uint16{38, 2, 1, 2, 3}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(1, 2, 3)})})
	strB_csi(t, &s, nil, []uint16{48, 2, 4, 5, 6}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: strB_rgb(4, 5, 6)})})
	strB_csi(t, &s, nil, []uint16{58, 2, 7, 8, 9}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: strB_rgb(7, 8, 9)})})
	strB_csi(t, &s, nil, []uint16{38, 2, 255, 128, 0, 1}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(255, 128, 0)}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
	})
	// Channel values are truncated to 8 bits.
	strB_csi(t, &s, nil, []uint16{38, 2, 256, 257, 65535}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(0, 1, 255)})})
	strB_csi(t, &s, nil, []uint16{48, 2, 0, 0, 0}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: strB_rgb(0, 0, 0)})})

	// An incomplete color is unknown (with the remaining params), and the
	// parser then continues with the very next param.
	strB_csi(t, &s, nil, []uint16{38, 5}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{38, 5}, []uint16{38, 5}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBlink}),
	})
	strB_csi(t, &s, nil, []uint16{38, 2, 1, 2}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{38, 2, 1, 2}, []uint16{38, 2, 1, 2}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagFaint}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagFaint}),
	})
	strB_csi(t, &s, nil, []uint16{48, 2, 1, 2}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{48, 2, 1, 2}, []uint16{48, 2, 1, 2}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagFaint}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagFaint}),
	})
	strB_csi(t, &s, nil, []uint16{58, 2, 1}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{58, 2, 1}, []uint16{58, 2, 1}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagFaint}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
	})
	strB_csi(t, &s, nil, []uint16{38, 1, 2}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{38, 1, 2}, []uint16{38, 1, 2}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagFaint}),
	})
	strB_csi(t, &s, nil, []uint16{48, 9}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{48, 9}, []uint16{48, 9}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagStrikethrough}),
	})
	strB_csi(t, &s, nil, []uint16{58, 5}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{58, 5}, []uint16{58, 5}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBlink}),
	})
	// The unknown payload's `full` is the whole param list and `partial`
	// the params from the failing one onwards.
	strB_csi(t, &s, nil, []uint16{1, 38, 5}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_unkEv([]uint16{1, 38, 5}, []uint16{38, 5}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBlink}),
	})
	strB_csi(t, &s, nil, []uint16{1, 10, 3}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_unkEv([]uint16{1, 10, 3}, []uint16{10, 3}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic}),
	})
	// Unknown values do not stop later params.
	strB_csi(t, &s, nil, []uint16{10, 1, 60, 3}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{10, 1, 60, 3}, []uint16{10, 1, 60, 3}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_unkEv([]uint16{10, 1, 60, 3}, []uint16{60, 3}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic}),
	})

	// A trailing param after a complete color is parsed normally.
	strB_csi(t, &s, nil, []uint16{1, 38, 2, 10, 20, 30, 4}, nil, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(10, 20, 30)}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle}),
	})

	// Bytes.
	strB_feed(t, &s, "1;3;4;31", 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: ColorNameRed}),
	})
	strB_feed(t, &s, "38;5;196", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 196})})
	strB_feed(t, &s, "38;2;255;128;0", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(255, 128, 0)})})
	strB_feed(t, &s, "48;2;0;0;0;58;5;1", 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: strB_rgb(0, 0, 0)}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 1}),
	})
	strB_feed(t, &s, "38;5", 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{38, 5}, []uint16{38, 5}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBlink}),
	})
	strB_feed(t, &s, "38;2;300;400;500", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(44, 144, 244)})})
	// Two sequences back to back are independent.
	zigTry(t, s.NextSlice([]uint8("\x1b[1m\x1b[31m\x1b[m")))
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: ColorNameRed}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset}),
	})
	// Split across calls.
	zigTry(t, s.NextSlice([]uint8("\x1b[1;")))
	strB_expectNone(t, &s)
	zigTry(t, s.NextSlice([]uint8("31m")))
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}), strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: ColorNameRed})})
}

// Zig: test "term51: Stream.csiDispatch: CSI m with colon separators"
func TestStream_term51_Stream_csiDispatch_CSI_m_with_colon_separators(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// Underline styles: 4:N, N in 0..5, anything else is single.
	styles := []struct {
		n     uint16
		style SGRAttributeUnderline
	}{
		{0, SGRAttributeUnderlineNone},
		{1, SGRAttributeUnderlineSingle},
		{2, SGRAttributeUnderlineDouble},
		{3, SGRAttributeUnderlineCurly},
		{4, SGRAttributeUnderlineDotted},
		{5, SGRAttributeUnderlineDashed},
		{6, SGRAttributeUnderlineSingle},
		{7, SGRAttributeUnderlineSingle},
		{255, SGRAttributeUnderlineSingle},
		{65535, SGRAttributeUnderlineSingle},
	}
	for _, st := range styles {
		strB_csi(t, &s, nil, []uint16{4, st.n}, []uint{0}, 'm')
		strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: st.style})})

		body := "4:" + strconv.Itoa(int(st.n))
		strB_feed(t, &s, body, 'm')
		strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: st.style})})
	}
	// The sub-param is consumed: the next param stands alone.
	strB_csi(t, &s, nil, []uint16{4, 3, 1}, []uint{0}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineCurly}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
	})
	strB_csi(t, &s, nil, []uint16{1, 4, 3}, []uint{1}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineCurly}),
	})
	strB_csi(t, &s, nil, []uint16{1, 4, 3, 31}, []uint{1}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineCurly}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: ColorNameRed}),
	})
	// 4:3:1 has two sub-params after the 4: unknown (the whole group).
	strB_csi(t, &s, nil, []uint16{4, 3, 1}, []uint{0, 1}, 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{4, 3, 1}, []uint16{4, 3, 1})})
	// A trailing colon with no sub-param: 4 alone is unknown.
	strB_csi(t, &s, nil, []uint16{4}, []uint{0}, 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{4}, []uint16{4})})
	strB_feed(t, &s, "4:", 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{4}, []uint16{4})})
	// "58:4:" leaves the colon bit of the final 4 set with nothing after it.
	strB_csi(t, &s, nil, []uint16{58, 4}, []uint{0, 1}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{58, 4}, []uint16{58, 4}),
		strB_unkEv([]uint16{58, 4}, []uint16{4}),
	})
	strB_feed(t, &s, "58:4:", 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{58, 4}, []uint16{58, 4}),
		strB_unkEv([]uint16{58, 4}, []uint16{4}),
	})

	// Colon direct colors: 38:2:R:G:B.
	strB_csi(t, &s, nil, []uint16{38, 2, 1, 2, 3}, []uint{0, 1, 2, 3}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(1, 2, 3)})})
	strB_csi(t, &s, nil, []uint16{48, 2, 4, 5, 6}, []uint{0, 1, 2, 3}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: strB_rgb(4, 5, 6)})})
	strB_csi(t, &s, nil, []uint16{58, 2, 7, 8, 9}, []uint{0, 1, 2, 3}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: strB_rgb(7, 8, 9)})})
	// With a color space id: 38:2:CS:R:G:B (the id is ignored).
	strB_csi(t, &s, nil, []uint16{38, 2, 9, 1, 2, 3}, []uint{0, 1, 2, 3, 4}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(1, 2, 3)})})
	strB_csi(t, &s, nil, []uint16{48, 2, 9, 4, 5, 6}, []uint{0, 1, 2, 3, 4}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: strB_rgb(4, 5, 6)})})
	strB_csi(t, &s, nil, []uint16{58, 2, 9, 7, 8, 9}, []uint{0, 1, 2, 3, 4}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: strB_rgb(7, 8, 9)})})
	// The color form is followed by independent params.
	strB_csi(t, &s, nil, []uint16{38, 2, 1, 2, 3, 1}, []uint{0, 1, 2, 3}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(1, 2, 3)}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
	})
	strB_csi(t, &s, nil, []uint16{38, 2, 9, 1, 2, 3, 1}, []uint{0, 1, 2, 3, 4}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(1, 2, 3)}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
	})
	// Mixed: a semicolon after the 38 means the plain semicolon form wins
	// even though later separators are colons.
	strB_csi(t, &s, nil, []uint16{38, 2, 1, 2, 3}, []uint{1, 2, 3}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(1, 2, 3)})})
	// Mixed the other way: a colon after 38 but then semicolons: the count
	// of following colons is 0, so the 38 is unknown (partial is everything
	// from it) and only the `2` sub-param is consumed with it; parsing
	// resumes at the `1`.
	strB_csi(t, &s, nil, []uint16{38, 2, 1, 2, 3}, []uint{0}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{38, 2, 1, 2, 3}, []uint16{38, 2, 1, 2, 3}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagFaint}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic}),
	})
	// Only two colons after the 38: count 2 is neither 3 nor 4. The unknown
	// group swallows the colon-joined params; the last one is separate.
	strB_csi(t, &s, nil, []uint16{38, 2, 1, 2, 3}, []uint{0, 1, 2}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{38, 2, 1, 2, 3}, []uint16{38, 2, 1, 2, 3}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic}),
	})
	// Too short for a direct color with colons: the 38 is unknown, then the
	// `2` carries a colon, which is not allowed for it, so it swallows the
	// colon-joined `2:1:2` as another unknown.
	strB_csi(t, &s, nil, []uint16{38, 2, 1, 2}, []uint{0, 1, 2}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{38, 2, 1, 2}, []uint16{38, 2, 1, 2}),
		strB_unkEv([]uint16{38, 2, 1, 2}, []uint16{2, 1, 2}),
	})

	// Colon 256 colors: 38:5:N.
	strB_csi(t, &s, nil, []uint16{38, 5, 200}, []uint{0, 1}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 200})})
	strB_csi(t, &s, nil, []uint16{48, 5, 100}, []uint{0, 1}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 100})})
	strB_csi(t, &s, nil, []uint16{58, 5, 7}, []uint{0, 1}, 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 7})})
	strB_csi(t, &s, nil, []uint16{38, 5}, []uint{0}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{38, 5}, []uint16{38, 5}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBlink}),
	})

	// Colon after a param that does not allow it: the colon-joined group is
	// one unknown, and parsing continues after it.
	strB_csi(t, &s, nil, []uint16{1, 2}, []uint{0}, 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{1, 2}, []uint16{1, 2})})
	strB_csi(t, &s, nil, []uint16{3, 4, 5}, []uint{0, 1}, 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{3, 4, 5}, []uint16{3, 4, 5})})
	strB_csi(t, &s, nil, []uint16{1, 2, 3}, []uint{0}, 'm')
	strB_expect(t, &s, []strB_event{
		strB_unkEv([]uint16{1, 2, 3}, []uint16{1, 2}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagItalic}),
	})
	strB_csi(t, &s, nil, []uint16{0, 1}, []uint{0}, 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{0, 1}, []uint16{0, 1})})
	strB_csi(t, &s, nil, []uint16{31, 1}, []uint{0}, 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{31, 1}, []uint16{31, 1})})

	// Bytes.
	strB_feed(t, &s, "4:3", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineCurly})})
	strB_feed(t, &s, "38:2:1:2:3", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(1, 2, 3)})})
	strB_feed(t, &s, "38:2::1:2:3", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(1, 2, 3)})})
	strB_feed(t, &s, "48:2:0:4:5:6", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: strB_rgb(4, 5, 6)})})
	strB_feed(t, &s, "58:2::7:8:9", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: strB_rgb(7, 8, 9)})})
	strB_feed(t, &s, "38:5:200", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 200})})
	strB_feed(t, &s, "1:2", 'm')
	strB_expect(t, &s, []strB_event{strB_unkEv([]uint16{1, 2}, []uint16{1, 2})})
	// Mixed colon and semicolon in one sequence (kakoune style, 18 params).
	strB_feed(t, &s, "4:3;38;2;51;51;51;48;2;170;170;170;58;2;255;97;136;0", 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineCurly}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: strB_rgb(51, 51, 51)}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: strB_rgb(170, 170, 170)}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: strB_rgb(255, 97, 136)}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset}),
	})
	strB_feed(t, &s, "4:3;31", 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineCurly}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: ColorNameRed}),
	})
	strB_feed(t, &s, "1;4:2;7", 'm')
	strB_expect(t, &s, []strB_event{
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineDouble}),
		strB_sgr(SGRAttribute{Tag: SGRAttributeTagInverse}),
	})

	// Colons in the last representable slot: 22 bold params then 4:3.
	many := [24]uint16(strB_repeat(1, 24))
	many[22] = 4
	many[23] = 3
	strB_csi(t, &s, nil, many[:], []uint{22}, 'm')
	expectEqual(t, uint(23), s.handler.len)
	for _, e := range s.handler.events[0:22] {
		strB_expectEqualDeep(t, strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}), e)
	}
	strB_expectEqualDeep(t, strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineCurly}), s.handler.events[22])
	s.handler.len = 0
}

// Zig: test "term51: Stream.csiDispatch: CSI m param count limits through bytes"
func TestStream_term51_Stream_csiDispatch_CSI_m_param_count_limits_through_bytes(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// 24 params is the limit and each one is its own attribute.
	strB_feed(t, &s, "1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1", 'm')
	expectEqual(t, uint(24), s.handler.len)
	for _, e := range s.handler.events[0:24] {
		strB_expectEqualDeep(t, strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold}), e)
	}
	s.handler.len = 0

	// 25 params are dropped entirely by the parser: no unset either.
	strB_feed(t, &s, "1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1", 'm')
	strB_expectNone(t, &s)
	// 24 params followed by an empty one is also dropped.
	strB_feed(t, &s, "1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;1;", 'm')
	strB_expectNone(t, &s)

	// Direct: 24 zero params are 24 unsets. (More than 24 cannot be given
	// to the SGR parser: its separator bit set only has 24 bits.)
	strB_csi(t, &s, nil, strB_repeat(0, 24), nil, 'm')
	expectEqual(t, uint(24), s.handler.len)
	for _, e := range s.handler.events[0:24] {
		strB_expectEqualDeep(t, strB_sgr(SGRAttribute{Tag: SGRAttributeTagUnset}), e)
	}
	s.handler.len = 0
}

// Zig: test "term51: Stream.csiDispatch: CSI > m sets modifyOtherKeys format (XTMODKEYS)"
func TestStream_term51_Stream_csiDispatch_CSI_m_sets_modifyOtherKeys_format_XTMODKEYS(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// No params: reset to legacy.
	strB_csi(t, &s, []uint8(">"), nil, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatLegacy)})

	// One param.
	strB_csi(t, &s, []uint8(">"), []uint16{0}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatLegacy)})
	strB_csi(t, &s, []uint8(">"), []uint16{1}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatCursorKeys)})
	strB_csi(t, &s, []uint8(">"), []uint16{2}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatFunctionKeys)})
	strB_csi(t, &s, []uint8(">"), []uint16{4}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNone)})
	for _, v := range []uint16{3, 5, 6, 7, 10, 255, 256, 260, 65535} {
		strB_csi(t, &s, []uint8(">"), []uint16{v}, nil, 'm')
		strB_expectNone(t, &s)
	}

	// Two params: only 4;2 changes the result.
	strB_csi(t, &s, []uint8(">"), []uint16{4, 2}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNumeric)})
	strB_csi(t, &s, []uint8(">"), []uint16{4, 0}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNone)})
	strB_csi(t, &s, []uint8(">"), []uint16{4, 1}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNone)})
	strB_csi(t, &s, []uint8(">"), []uint16{4, 3}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNone)})
	strB_csi(t, &s, []uint8(">"), []uint16{4, 65535}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNone)})
	strB_csi(t, &s, []uint8(">"), []uint16{0, 2}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatLegacy)})
	strB_csi(t, &s, []uint8(">"), []uint16{1, 2}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatCursorKeys)})
	strB_csi(t, &s, []uint8(">"), []uint16{2, 2}, nil, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatFunctionKeys)})
	// An invalid first param rejects regardless of the second.
	strB_csi(t, &s, []uint8(">"), []uint16{3, 2}, nil, 'm')
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8(">"), []uint16{5, 2}, nil, 'm')
	strB_expectNone(t, &s)

	// Three or more params are rejected, with a valid or an invalid first.
	strB_csi(t, &s, []uint8(">"), []uint16{4, 2, 0}, nil, 'm')
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8(">"), []uint16{0, 0, 0}, nil, 'm')
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8(">"), []uint16{9, 9, 9}, nil, 'm')
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8(">"), strB_repeat(4, 24), nil, 'm')
	strB_expectNone(t, &s)
	strB_csi(t, &s, []uint8(">"), strB_repeat(4, 30), nil, 'm')
	strB_expectNone(t, &s)

	// The colon bits are not looked at.
	strB_csi(t, &s, []uint8(">"), []uint16{4, 2}, []uint{0}, 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNumeric)})

	// Bytes.
	strB_feed(t, &s, ">", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatLegacy)})
	strB_feed(t, &s, ">0", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatLegacy)})
	strB_feed(t, &s, ">1", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatCursorKeys)})
	strB_feed(t, &s, ">2", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatFunctionKeys)})
	strB_feed(t, &s, ">4", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNone)})
	strB_feed(t, &s, ">4;2", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNumeric)})
	strB_feed(t, &s, ">4:2", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNumeric)})
	strB_feed(t, &s, ">4;", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatOtherKeysNone)})
	strB_feed(t, &s, ">;", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatLegacy)})
	strB_feed(t, &s, ">;4", 'm')
	strB_expect(t, &s, []strB_event{strB_mkf(ANSIModifyKeyFormatLegacy)})
	strB_feed(t, &s, ">3", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, ">65535", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, ">4;2;0", 'm')
	strB_expectNone(t, &s)
}

// Zig: test "term51: Stream.csiDispatch: CSI m with other intermediates is ignored"
func TestStream_term51_Stream_csiDispatch_CSI_m_with_other_intermediates_is_ignored(t *testing.T) {
	s := StreamInit(&strB_recorder{})
	defer s.Deinit()

	// A single intermediate other than ">" is dropped, whatever the params.
	for _, c := range []uint8{'?', '<', '=', ' ', '!', '"', '#', '$', '\''} {
		strB_csi(t, &s, []uint8{c}, nil, nil, 'm')
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{0}, nil, 'm')
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{4, 2}, nil, 'm')
		strB_expectNone(t, &s)
		strB_csi(t, &s, []uint8{c}, []uint16{1}, nil, 'm')
		strB_expectNone(t, &s)
	}
	// Two or more intermediates are dropped, including ones starting with ">".
	for _, inter := range [][]uint8{[]uint8(">>"), []uint8(">?"), []uint8("?>"), []uint8(">="), []uint8(" >"), []uint8("> "), []uint8("??"), []uint8("  "), []uint8(">>>"), []uint8(">>>>")} {
		strB_csi(t, &s, inter, nil, nil, 'm')
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, []uint16{4}, nil, 'm')
		strB_expectNone(t, &s)
		strB_csi(t, &s, inter, []uint16{4, 2}, nil, 'm')
		strB_expectNone(t, &s)
	}

	// Bytes ("CSI ? 4 m" is what vim --clean sends).
	strB_feed(t, &s, "?4", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "?", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "<1", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "=1", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "1 ", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "1!", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "1\"", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "1$", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, " ", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, ">1 ", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "> ", 'm')
	strB_expectNone(t, &s)
	strB_feed(t, &s, "1>", 'm')
	strB_expectNone(t, &s)
	// The parser state recovers: a normal SGR works afterwards.
	strB_feed(t, &s, "1", 'm')
	strB_expect(t, &s, []strB_event{strB_sgr(SGRAttribute{Tag: SGRAttributeTagBold})})
}

// ---------------------------------------------------------------------------
