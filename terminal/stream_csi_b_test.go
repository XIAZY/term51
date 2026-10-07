// Tests translated one for one from section (C) of src/terminal/stream.zig
// (Ghostty v1.3.1 plus the term51 additional tests (C)): the second half of
// the csiDispatch tests. Each Go test carries the exact Zig title in a
// "// Zig: test" comment. The helpers expect, expectEqual and zigTry are
// defined in Parser_test.go and dcs_test.go.
//
// Translation notes:
//   - `term51CRecorder` is strC_recorder; `Stream(term51CRecorder)` is
//     Stream[*strC_recorder] (strC_stream). Zig's `.init(.{})` is
//     StreamInit(&strC_recorder{}); the handler is a pointer so that the
//     recorded events are visible through `s.handler`.
//   - The recorder's `vt` is Vt(action, value): `value` is the whole
//     StreamAction and the payload is the field named by the tag.
//   - `&.{}` for a parameter or event list is nil; `[_]u16{x} ** n` is
//     strC_repeat(x, n).
//   - `inline for` over the fields of modes.Mode is a loop over
//     strC_allModes, which lists the constants in the order of the Zig enum.
//   - A Go `for` variable cannot shadow `t`, so the loop variable named `t`
//     (a window target) in the "CSI 22 t and 23 t title stack" test is `tgt`.
//   - Zig's errdefer debug prints become a deferred t.Logf on failure.

package terminal

import (
	"errors"
	"fmt"
	"testing"
)

// strC_event is term51CEvent: one recorded action, the tag plus its payload
// flattened into two integers.
type strC_event struct {
	tag StreamActionKey
	a   uint32
	b   uint32
}

// strC_ev is term51CEv.
func strC_ev(tag StreamActionKey, a uint32, b uint32) strC_event {
	return strC_event{tag: tag, a: a, b: b}
}

// strC_errNoSpaceLeft is Zig's error.NoSpaceLeft.
var strC_errNoSpaceLeft = errors.New("NoSpaceLeft")

// strC_recorder is term51CRecorder: a handler that records every action (tag
// + flattened payload) in order.
type strC_recorder struct {
	events [64]strC_event
	n      uint
}

func (self *strC_recorder) Vt(action StreamActionKey, value *StreamAction) error {
	ev := strC_event{tag: action}
	switch action {
	case StreamActionKeySizeReport:
		ev.a = uint32(value.SizeReport)
	case StreamActionKeyCursorStyle:
		ev.a = uint32(value.CursorStyle)
	case StreamActionKeyActiveStatusDisplay:
		ev.a = uint32(value.ActiveStatusDisplay)
	case StreamActionKeyModifyKeyFormat:
		ev.a = uint32(value.ModifyKeyFormat)
	case StreamActionKeyDeviceStatus:
		ev.a = uint32(value.DeviceStatus.Request)
	case StreamActionKeyRequestMode:
		ev.a = uint32(value.RequestMode.Mode)
	case StreamActionKeySaveMode:
		ev.a = uint32(value.SaveMode.Mode)
	case StreamActionKeyRestoreMode:
		ev.a = uint32(value.RestoreMode.Mode)
	case StreamActionKeyRequestModeUnknown:
		ev.a = uint32(value.RequestModeUnknown.Mode)
		if value.RequestModeUnknown.Ansi {
			ev.b = 1
		}
	case StreamActionKeyTopAndBottomMargin:
		ev.a = uint32(value.TopAndBottomMargin.TopLeft)
		ev.b = uint32(value.TopAndBottomMargin.BottomRight)
	case StreamActionKeyLeftAndRightMargin:
		ev.a = uint32(value.LeftAndRightMargin.TopLeft)
		ev.b = uint32(value.LeftAndRightMargin.BottomRight)
	case StreamActionKeyMouseShiftCapture:
		if value.MouseShiftCapture {
			ev.a = 1
		}
	case StreamActionKeyTitlePush:
		ev.a = uint32(value.TitlePush)
	case StreamActionKeyTitlePop:
		ev.a = uint32(value.TitlePop)
	case StreamActionKeyKittyKeyboardPop:
		ev.a = uint32(value.KittyKeyboardPop)
	case StreamActionKeyInsertBlanks:
		ev.a = uint32(value.InsertBlanks)
	case StreamActionKeyKittyKeyboardPush:
		ev.a = uint32(value.KittyKeyboardPush.Flags.Int())
	case StreamActionKeyKittyKeyboardSet:
		ev.a = uint32(value.KittyKeyboardSet.Flags.Int())
	case StreamActionKeyKittyKeyboardSetOr:
		ev.a = uint32(value.KittyKeyboardSetOr.Flags.Int())
	case StreamActionKeyKittyKeyboardSetNot:
		ev.a = uint32(value.KittyKeyboardSetNot.Flags.Int())
	}
	if self.n >= uint(len(self.events)) {
		return strC_errNoSpaceLeft
	}
	self.events[self.n] = ev
	self.n += 1
	return nil
}

func (self *strC_recorder) Deinit() {}

// strC_stream is term51CStream.
type strC_stream = Stream[*strC_recorder]

// strC_expect is term51CExpect.
func strC_expect(t *testing.T, s *strC_stream, expected []strC_event) {
	t.Helper()
	expectEqual(t, uint(len(expected)), s.handler.n)
	for i, e := range expected {
		expectEqual(t, e, s.handler.events[i])
	}
}

// strC_check hand-builds a CSI action and calls csiDispatch directly on a
// fresh stream, then asserts exactly the expected recorded actions.
// (term51CCheck)
func strC_check(t *testing.T, inter string, params []uint16, final uint8, expected []strC_event) {
	t.Helper()
	defer func() {
		if t.Failed() {
			t.Logf("term51C direct failure: inter=%s params=%v final=%c", inter, params, final)
		}
	}()
	var ibuf [8]uint8
	var pbuf [64]uint16
	copy(ibuf[0:len(inter)], inter)
	copy(pbuf[0:len(params)], params)
	s := StreamInit(&strC_recorder{})
	zigTry(t, s.csiDispatch(ParserActionCSI{
		Intermediates: ibuf[0:len(inter)],
		Params:        pbuf[0:len(params)],
		ParamsSep:     StaticBitSet24InitEmpty(),
		Final:         final,
	}))
	strC_expect(t, &s, expected)
}

// strC_none is a direct dispatch that must emit nothing. (term51CNone)
func strC_none(t *testing.T, inter string, params []uint16, final uint8) {
	t.Helper()
	strC_check(t, inter, params, final, nil)
}

// strC_bytes feeds raw bytes through nextSlice on a fresh stream and asserts
// the recorded actions. (term51CBytes)
func strC_bytes(t *testing.T, bytes string, expected []strC_event) {
	t.Helper()
	defer func() {
		if t.Failed() {
			t.Logf("term51C bytes failure: %v", []uint8(bytes))
		}
	}()
	s := StreamInit(&strC_recorder{})
	zigTry(t, s.NextSlice([]uint8(bytes)))
	strC_expect(t, &s, expected)
}

// strC_mode is term51CMode.
func strC_mode(m ModesMode) uint32 {
	return uint32(m)
}

// strC_dsr is term51CDsr.
func strC_dsr(r DeviceStatusRequest) uint32 {
	return uint32(r)
}

// strC_repeat is Zig's `[_]u16{v} ** n`.
func strC_repeat(v uint16, n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		out[i] = v
	}
	return out
}

// strC_allModes is the list of fields of modes.Mode, which Zig walks with
// `inline for (@typeInfo(modes.Mode).@"enum".fields)`, in the order of
// modes.entries.
var strC_allModes = []ModesMode{
	ModesModeDisableKeyboard,
	ModesModeInsert,
	ModesModeSendReceiveMode,
	ModesModeLinefeed,
	ModesModeCursorKeys,
	ModesMode132Column,
	ModesModeSlowScroll,
	ModesModeReverseColors,
	ModesModeOrigin,
	ModesModeWraparound,
	ModesModeAutorepeat,
	ModesModeMouseEventX10,
	ModesModeCursorBlinking,
	ModesModeCursorVisible,
	ModesModeEnableMode3,
	ModesModeReverseWrap,
	ModesModeAltScreenLegacy,
	ModesModeKeypadKeys,
	ModesModeEnableLeftAndRightMargin,
	ModesModeMouseEventNormal,
	ModesModeMouseEventButton,
	ModesModeMouseEventAny,
	ModesModeFocusEvent,
	ModesModeMouseFormatUtf8,
	ModesModeMouseFormatSgr,
	ModesModeMouseAlternateScroll,
	ModesModeMouseFormatUrxvt,
	ModesModeMouseFormatSgrPixels,
	ModesModeIgnoreKeypadWithNumlock,
	ModesModeAltEscPrefix,
	ModesModeAltSendsEscape,
	ModesModeReverseWrapExtended,
	ModesModeAltScreen,
	ModesModeSaveCursor,
	ModesModeAltScreenSaveCursorClearEnter,
	ModesModeBracketedPaste,
	ModesModeSynchronizedOutput,
	ModesModeGraphemeCluster,
	ModesModeReportColorScheme,
	ModesModeInBandSizeReports,
}

// strC_modeTag is Zig's `@bitCast(@as(u16, v))` to modes.ModeTag (value in
// bits 0 to 14, ansi in bit 15). The wireframe has only ModesModeTag.Int, the
// opposite direction.
func strC_modeTag(v uint16) ModesModeTag {
	return ModesModeTag{Value: v & 0x7fff, Ansi: v&0x8000 != 0}
}

// ---- CSI n -----------------------------------------------------------------

// Zig: test "term51: Stream.csiDispatch: CSI n device status report plain"
func TestStream_term51_Stream_csiDispatch_CSI_n_device_status_report_plain(t *testing.T) {
	// 5 = operating status, 6 = cursor position (no "?" form valid).
	strC_check(t, "", []uint16{5}, 'n', []strC_event{strC_ev(StreamActionKeyDeviceStatus, strC_dsr(DeviceStatusRequestOperatingStatus), 0)})
	strC_check(t, "", []uint16{6}, 'n', []strC_event{strC_ev(StreamActionKeyDeviceStatus, strC_dsr(DeviceStatusRequestCursorPosition), 0)})
	strC_bytes(t, "\x1b[5n", []strC_event{strC_ev(StreamActionKeyDeviceStatus, strC_dsr(DeviceStatusRequestOperatingStatus), 0)})
	strC_bytes(t, "\x1b[6n", []strC_event{strC_ev(StreamActionKeyDeviceStatus, strC_dsr(DeviceStatusRequestCursorPosition), 0)})
	// The question-only request is rejected without the "?".
	strC_none(t, "", []uint16{996}, 'n')
	// Unknown values and boundaries.
	for _, p := range []uint16{0, 1, 4, 7, 15, 996 - 1, 997, 65535} {
		strC_none(t, "", []uint16{p}, 'n')
	}
	strC_bytes(t, "\x1b[0n", nil)
	strC_bytes(t, "\x1b[65535n", nil)
	strC_bytes(t, "\x1b[65536n", nil)
	// 5 and 6 never gain the question bit: "?5n" / "?6n" are invalid.
	strC_none(t, "?", []uint16{5}, 'n')
	strC_none(t, "?", []uint16{6}, 'n')
	strC_bytes(t, "\x1b[?5n", nil)
	strC_bytes(t, "\x1b[?6n", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI n device status report parameter counts"
func TestStream_term51_Stream_csiDispatch_CSI_n_device_status_report_parameter_counts(t *testing.T) {
	// Exactly one parameter is required.
	strC_none(t, "", nil, 'n')
	strC_none(t, "", []uint16{5, 6}, 'n')
	strC_none(t, "", []uint16{6, 6}, 'n')
	strC_none(t, "", []uint16{5, 0}, 'n')
	strC_none(t, "", []uint16{0, 5}, 'n')
	strC_none(t, "", strC_repeat(5, 30), 'n')
	strC_none(t, "?", nil, 'n')
	strC_none(t, "?", []uint16{996, 996}, 'n')
	strC_none(t, "?", []uint16{996, 0}, 'n')
	strC_none(t, "?", strC_repeat(996, 30), 'n')
	strC_bytes(t, "\x1b[n", nil)
	strC_bytes(t, "\x1b[5;6n", nil)
	strC_bytes(t, "\x1b[?n", nil)
	strC_bytes(t, "\x1b[?996;996n", nil)
	// A trailing empty parameter is not counted by the parser.
	strC_bytes(t, "\x1b[5;n", []strC_event{strC_ev(StreamActionKeyDeviceStatus, strC_dsr(DeviceStatusRequestOperatingStatus), 0)})
	// A leading empty parameter is a zero, so this is two params.
	strC_bytes(t, "\x1b[;5n", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI ? n DEC device status report"
func TestStream_term51_Stream_csiDispatch_CSI_n_DEC_device_status_report(t *testing.T) {
	// Only 996 (color scheme) is valid with the "?" prefix.
	strC_check(t, "?", []uint16{996}, 'n', []strC_event{strC_ev(StreamActionKeyDeviceStatus, strC_dsr(DeviceStatusRequestColorScheme), 0)})
	strC_bytes(t, "\x1b[?996n", []strC_event{strC_ev(StreamActionKeyDeviceStatus, strC_dsr(DeviceStatusRequestColorScheme), 0)})
	// The question bit is part of the request identity.
	expect(t, strC_dsr(DeviceStatusRequestColorScheme) != 996, "term51CDsr(.color_scheme) != 996")
	expectEqual(t, uint32(996|0x8000), strC_dsr(DeviceStatusRequestColorScheme))
	for _, p := range []uint16{0, 5, 6, 995, 997, 1996, 65535} {
		strC_none(t, "?", []uint16{p}, 'n')
	}
	strC_bytes(t, "\x1b[?995n", nil)
	strC_bytes(t, "\x1b[?997n", nil)
	strC_bytes(t, "\x1b[?65535n", nil)
	strC_bytes(t, "\x1b[?99999n", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI n device status report two intermediates"
func TestStream_term51_Stream_csiDispatch_CSI_n_device_status_report_two_intermediates(t *testing.T) {
	// "?" first with a second intermediate takes the DSR path and is rejected.
	strC_none(t, "?$", []uint16{996}, 'n')
	strC_none(t, "??", []uint16{996}, 'n')
	strC_none(t, "?>", []uint16{996}, 'n')
	strC_none(t, "?$", []uint16{6}, 'n')
	strC_none(t, "?$", nil, 'n')
	strC_bytes(t, "\x1b[?$996n", nil)
	// Other first intermediate with two: the generic "ignored" arm.
	strC_none(t, ">?", []uint16{996}, 'n')
	strC_none(t, ">>", nil, 'n')
	strC_none(t, "$?", []uint16{6}, 'n')
	strC_none(t, " ?", []uint16{6}, 'n')
	strC_none(t, "?>$", []uint16{996}, 'n')
	strC_none(t, ">$ ", nil, 'n')
}

// Zig: test "term51: Stream.csiDispatch: CSI > n resets modify other keys"
func TestStream_term51_Stream_csiDispatch_CSI_n_resets_modify_other_keys(t *testing.T) {
	want := strC_ev(StreamActionKeyModifyKeyFormat, uint32(ANSIModifyKeyFormatOtherKeysNumericExcept), 0)
	// Parameters are ignored entirely.
	strC_check(t, ">", nil, 'n', []strC_event{want})
	strC_check(t, ">", []uint16{0}, 'n', []strC_event{want})
	strC_check(t, ">", []uint16{4}, 'n', []strC_event{want})
	strC_check(t, ">", []uint16{65535}, 'n', []strC_event{want})
	strC_check(t, ">", []uint16{1, 2}, 'n', []strC_event{want})
	strC_check(t, ">", strC_repeat(7, 30), 'n', []strC_event{want})
	strC_bytes(t, "\x1b[>n", []strC_event{want})
	strC_bytes(t, "\x1b[>4n", []strC_event{want})
	strC_bytes(t, "\x1b[>1;2n", []strC_event{want})
}

// Zig: test "term51: Stream.csiDispatch: CSI n other single intermediates ignored"
func TestStream_term51_Stream_csiDispatch_CSI_n_other_single_intermediates_ignored(t *testing.T) {
	for _, c := range []uint8("<=!\"#$%&'()*+,-./ ") {
		strC_none(t, string([]uint8{c}), nil, 'n')
		strC_none(t, string([]uint8{c}), []uint16{5}, 'n')
		strC_none(t, string([]uint8{c}), []uint16{996}, 'n')
	}
	strC_bytes(t, "\x1b[<n", nil)
	strC_bytes(t, "\x1b[=6n", nil)
	strC_bytes(t, "\x1b[ 5n", nil)
	strC_bytes(t, "\x1b[$5n", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI ? Ps $ p DECRQM known DEC modes"
func TestStream_term51_Stream_csiDispatch_CSI_Ps_p_DECRQM_known_DEC_modes(t *testing.T) {
	// Every DEC (non-ANSI) mode in the table produces request_mode.
	for _, f := range strC_allModes {
		tag := strC_modeTag(uint16(f))
		if !tag.Ansi {
			want := strC_ev(StreamActionKeyRequestMode, uint32(f), 0)
			strC_check(t, "?$", []uint16{tag.Value}, 'p', []strC_event{want})
		}
	}
	strC_check(t, "?$", []uint16{1}, 'p', []strC_event{strC_ev(StreamActionKeyRequestMode, strC_mode(ModesModeCursorKeys), 0)})
	strC_check(t, "?$", []uint16{2004}, 'p', []strC_event{strC_ev(StreamActionKeyRequestMode, strC_mode(ModesModeBracketedPaste), 0)})
	strC_check(t, "?$", []uint16{2048}, 'p', []strC_event{strC_ev(StreamActionKeyRequestMode, strC_mode(ModesModeInBandSizeReports), 0)})
	strC_bytes(t, "\x1b[?1$p", []strC_event{strC_ev(StreamActionKeyRequestMode, strC_mode(ModesModeCursorKeys), 0)})
	strC_bytes(t, "\x1b[?2026$p", []strC_event{strC_ev(StreamActionKeyRequestMode, strC_mode(ModesModeSynchronizedOutput), 0)})
	strC_bytes(t, "\x1b[?1049$p", []strC_event{strC_ev(StreamActionKeyRequestMode, strC_mode(ModesModeAltScreenSaveCursorClearEnter), 0)})
	// DEC values that collide with ANSI numbers resolve to the DEC mode.
	strC_check(t, "?$", []uint16{4}, 'p', []strC_event{strC_ev(StreamActionKeyRequestMode, strC_mode(ModesModeSlowScroll), 0)})
	strC_check(t, "?$", []uint16{12}, 'p', []strC_event{strC_ev(StreamActionKeyRequestMode, strC_mode(ModesModeCursorBlinking), 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI ? Ps $ p DECRQM unknown modes"
func TestStream_term51_Stream_csiDispatch_CSI_Ps_p_DECRQM_unknown_modes(t *testing.T) {
	// Unknown modes are still reported, with ansi=false.
	unknown := []uint16{0, 2, 10, 11, 20, 26, 46, 1001, 2005, 2049, 9999, 32767, 32768, 65535}
	for _, p := range unknown {
		strC_check(t, "?$", []uint16{p}, 'p', []strC_event{strC_ev(StreamActionKeyRequestModeUnknown, uint32(p), 0)})
	}
	// 2 and 20 exist only as ANSI modes, so they are unknown as DEC modes.
	strC_bytes(t, "\x1b[?2$p", []strC_event{strC_ev(StreamActionKeyRequestModeUnknown, 2, 0)})
	strC_bytes(t, "\x1b[?20$p", []strC_event{strC_ev(StreamActionKeyRequestModeUnknown, 20, 0)})
	strC_bytes(t, "\x1b[?0$p", []strC_event{strC_ev(StreamActionKeyRequestModeUnknown, 0, 0)})
	strC_bytes(t, "\x1b[?65535$p", []strC_event{strC_ev(StreamActionKeyRequestModeUnknown, 65535, 0)})
	// One past the u16 range saturates to 65535.
	strC_bytes(t, "\x1b[?65536$p", []strC_event{strC_ev(StreamActionKeyRequestModeUnknown, 65535, 0)})
	strC_bytes(t, "\x1b[?9999$p", []strC_event{strC_ev(StreamActionKeyRequestModeUnknown, 9999, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI ? Ps $ p DECRQM parameter counts"
func TestStream_term51_Stream_csiDispatch_CSI_Ps_p_DECRQM_parameter_counts(t *testing.T) {
	// Exactly one parameter is required.
	strC_none(t, "?$", nil, 'p')
	strC_none(t, "?$", []uint16{1, 2}, 'p')
	strC_none(t, "?$", []uint16{1, 1}, 'p')
	strC_none(t, "?$", []uint16{9999, 1}, 'p')
	strC_none(t, "?$", strC_repeat(1, 30), 'p')
	strC_bytes(t, "\x1b[?$p", nil)
	strC_bytes(t, "\x1b[?1;2$p", nil)
	strC_bytes(t, "\x1b[?1;2;3$p", nil)
	// A trailing empty parameter is dropped by the parser.
	strC_bytes(t, "\x1b[?1;$p", []strC_event{strC_ev(StreamActionKeyRequestMode, strC_mode(ModesModeCursorKeys), 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI p wrong two-intermediate combinations"
func TestStream_term51_Stream_csiDispatch_CSI_p_wrong_two_intermediate_combinations(t *testing.T) {
	// Only "?" followed by "$" is accepted among two intermediates.
	bad := []string{"$?", "$$", "??", ">$", "?>", " $", "$ ", "!$", "\"$", "?!", "<$", "=$"}
	for _, inter := range bad {
		strC_none(t, inter, []uint16{1}, 'p')
		strC_none(t, inter, []uint16{9999}, 'p')
		strC_none(t, inter, nil, 'p')
	}
	strC_bytes(t, "\x1b[$?1p", nil)
	strC_bytes(t, "\x1b[>$1p", nil)
	strC_bytes(t, "\x1b[?>1p", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI p other intermediate counts ignored"
func TestStream_term51_Stream_csiDispatch_CSI_p_other_intermediate_counts_ignored(t *testing.T) {
	// No intermediates, three intermediates, and four intermediates.
	strC_none(t, "", nil, 'p')
	strC_none(t, "", []uint16{1}, 'p')
	strC_none(t, "", []uint16{1, 2}, 'p')
	strC_none(t, "?$ ", []uint16{1}, 'p')
	strC_none(t, "?$$", []uint16{1}, 'p')
	strC_none(t, "???", []uint16{1}, 'p')
	strC_none(t, "?$ !", []uint16{1}, 'p')
	strC_bytes(t, "\x1b[p", nil)
	strC_bytes(t, "\x1b[1p", nil)
	strC_bytes(t, "\x1b[?$ 1p", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI Ps $ p ANSI DECRQM is not implemented"
func TestStream_term51_Stream_csiDispatch_CSI_Ps_p_ANSI_DECRQM_is_not_implemented(t *testing.T) {
	// The ANSI DECRQM form has one intermediate ("$"); the p handler only
	// accepts two intermediates, so the ansi=true arm is unreachable and
	// nothing is emitted (no request_mode, no request_mode_unknown).
	for _, p := range []uint16{0, 2, 4, 12, 20, 9999, 65535} {
		strC_none(t, "$", []uint16{p}, 'p')
	}
	strC_none(t, "$", nil, 'p')
	strC_none(t, "$", []uint16{4, 4}, 'p')
	strC_bytes(t, "\x1b[4$p", nil)
	strC_bytes(t, "\x1b[12$p", nil)
	strC_bytes(t, "\x1b[2$p", nil)
	strC_bytes(t, "\x1b[20$p", nil)
	strC_bytes(t, "\x1b[9999$p", nil)
	strC_bytes(t, "\x1b[$p", nil)
	// Other single intermediates are ignored too (DECSTR "!", DECSCL "\"").
	for _, c := range []uint8("!\"#%&'()*+,-./ <=>?") {
		strC_none(t, string([]uint8{c}), nil, 'p')
		strC_none(t, string([]uint8{c}), []uint16{61}, 'p')
		strC_none(t, string([]uint8{c}), []uint16{61, 1}, 'p')
	}
	strC_bytes(t, "\x1b[!p", nil)
	strC_bytes(t, "\x1b[61;1\"p", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI SP q DECSCUSR styles"
func TestStream_term51_Stream_csiDispatch_CSI_SP_q_DECSCUSR_styles(t *testing.T) {
	styles := []ANSICursorStyle{
		ANSICursorStyleDefault,
		ANSICursorStyleBlinkingBlock,
		ANSICursorStyleSteadyBlock,
		ANSICursorStyleBlinkingUnderline,
		ANSICursorStyleSteadyUnderline,
		ANSICursorStyleBlinkingBar,
		ANSICursorStyleSteadyBar,
	}
	// No parameter means default.
	strC_check(t, " ", nil, 'q', []strC_event{strC_ev(StreamActionKeyCursorStyle, uint32(ANSICursorStyleDefault), 0)})
	strC_bytes(t, "\x1b[ q", []strC_event{strC_ev(StreamActionKeyCursorStyle, uint32(ANSICursorStyleDefault), 0)})
	for i, st := range styles {
		want := strC_ev(StreamActionKeyCursorStyle, uint32(st), 0)
		strC_check(t, " ", []uint16{uint16(i)}, 'q', []strC_event{want})
		bytes := fmt.Sprintf("\x1b[%d q", i)
		strC_bytes(t, bytes, []strC_event{want})
	}
	// An empty first parameter is zero (default), via ";" it is two params.
	strC_bytes(t, "\x1b[0 q", []strC_event{strC_ev(StreamActionKeyCursorStyle, uint32(ANSICursorStyleDefault), 0)})
	strC_bytes(t, "\x1b[6 q", []strC_event{strC_ev(StreamActionKeyCursorStyle, uint32(ANSICursorStyleSteadyBar), 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI SP q DECSCUSR invalid"
func TestStream_term51_Stream_csiDispatch_CSI_SP_q_DECSCUSR_invalid(t *testing.T) {
	// One past the last style, and large values.
	for _, p := range []uint16{7, 8, 100, 255, 256, 65535} {
		strC_none(t, " ", []uint16{p}, 'q')
	}
	strC_bytes(t, "\x1b[7 q", nil)
	strC_bytes(t, "\x1b[65535 q", nil)
	strC_bytes(t, "\x1b[65536 q", nil)
	// More than one parameter.
	strC_none(t, " ", []uint16{1, 1}, 'q')
	strC_none(t, " ", []uint16{0, 0}, 'q')
	strC_none(t, " ", []uint16{7, 1}, 'q')
	strC_none(t, " ", strC_repeat(1, 30), 'q')
	strC_bytes(t, "\x1b[1;1 q", nil)
	strC_bytes(t, "\x1b[1;2;3 q", nil)
	// Without the space intermediate (len 0) nothing is emitted.
	strC_none(t, "", nil, 'q')
	strC_none(t, "", []uint16{3}, 'q')
	strC_bytes(t, "\x1b[3q", nil)
	strC_bytes(t, "\x1b[q", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI q DECSCA protected mode"
func TestStream_term51_Stream_csiDispatch_CSI_q_DECSCA_protected_mode(t *testing.T) {
	// No parameter, 0, and 2 all turn protection off; 1 is DEC protection.
	strC_check(t, "\"", nil, 'q', []strC_event{strC_ev(StreamActionKeyProtectedModeOff, 0, 0)})
	strC_check(t, "\"", []uint16{0}, 'q', []strC_event{strC_ev(StreamActionKeyProtectedModeOff, 0, 0)})
	strC_check(t, "\"", []uint16{2}, 'q', []strC_event{strC_ev(StreamActionKeyProtectedModeOff, 0, 0)})
	strC_check(t, "\"", []uint16{1}, 'q', []strC_event{strC_ev(StreamActionKeyProtectedModeDec, 0, 0)})
	strC_bytes(t, "\x1b[\"q", []strC_event{strC_ev(StreamActionKeyProtectedModeOff, 0, 0)})
	strC_bytes(t, "\x1b[0\"q", []strC_event{strC_ev(StreamActionKeyProtectedModeOff, 0, 0)})
	strC_bytes(t, "\x1b[2\"q", []strC_event{strC_ev(StreamActionKeyProtectedModeOff, 0, 0)})
	strC_bytes(t, "\x1b[1\"q", []strC_event{strC_ev(StreamActionKeyProtectedModeDec, 0, 0)})
	// protected_mode_iso is never produced by CSI q.
	for _, p := range []uint16{3, 4, 10, 255, 65535} {
		strC_none(t, "\"", []uint16{p}, 'q')
	}
	strC_bytes(t, "\x1b[3\"q", nil)
	strC_bytes(t, "\x1b[65536\"q", nil)
	// Two or more parameters are rejected.
	strC_none(t, "\"", []uint16{1, 1}, 'q')
	strC_none(t, "\"", []uint16{0, 0}, 'q')
	strC_none(t, "\"", []uint16{2, 1}, 'q')
	strC_none(t, "\"", strC_repeat(1, 30), 'q')
	strC_bytes(t, "\x1b[1;1\"q", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI > q XTVERSION"
func TestStream_term51_Stream_csiDispatch_CSI_q_XTVERSION(t *testing.T) {
	// Parameters are ignored.
	strC_check(t, ">", nil, 'q', []strC_event{strC_ev(StreamActionKeyXtversion, 0, 0)})
	strC_check(t, ">", []uint16{0}, 'q', []strC_event{strC_ev(StreamActionKeyXtversion, 0, 0)})
	strC_check(t, ">", []uint16{1}, 'q', []strC_event{strC_ev(StreamActionKeyXtversion, 0, 0)})
	strC_check(t, ">", []uint16{65535}, 'q', []strC_event{strC_ev(StreamActionKeyXtversion, 0, 0)})
	strC_check(t, ">", []uint16{1, 2}, 'q', []strC_event{strC_ev(StreamActionKeyXtversion, 0, 0)})
	strC_check(t, ">", strC_repeat(1, 30), 'q', []strC_event{strC_ev(StreamActionKeyXtversion, 0, 0)})
	strC_bytes(t, "\x1b[>q", []strC_event{strC_ev(StreamActionKeyXtversion, 0, 0)})
	strC_bytes(t, "\x1b[>0q", []strC_event{strC_ev(StreamActionKeyXtversion, 0, 0)})
	strC_bytes(t, "\x1b[>1;2q", []strC_event{strC_ev(StreamActionKeyXtversion, 0, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI q other intermediates ignored"
func TestStream_term51_Stream_csiDispatch_CSI_q_other_intermediates_ignored(t *testing.T) {
	// Other single intermediates.
	for _, c := range []uint8("?<=!#$%&'()*+,-./") {
		strC_none(t, string([]uint8{c}), nil, 'q')
		strC_none(t, string([]uint8{c}), []uint16{1}, 'q')
	}
	// Two or more intermediates.
	multi := []string{" >", "> ", "\" ", " \"", ">>", "  ", "?>", ">?", " $", " > ", ">>>"}
	for _, inter := range multi {
		strC_none(t, inter, nil, 'q')
		strC_none(t, inter, []uint16{1}, 'q')
		strC_none(t, inter, []uint16{1, 2}, 'q')
	}
	strC_bytes(t, "\x1b[?q", nil)
	strC_bytes(t, "\x1b[$q", nil)
	strC_bytes(t, "\x1b[ >q", nil)
	strC_bytes(t, "\x1b[> q", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI r DECSTBM parameter forms"
func TestStream_term51_Stream_csiDispatch_CSI_r_DECSTBM_parameter_forms(t *testing.T) {
	strC_check(t, "", nil, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 0, 0)})
	strC_check(t, "", []uint16{5}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 5, 0)})
	strC_check(t, "", []uint16{5, 10}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 5, 10)})
	strC_check(t, "", []uint16{0}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 0, 0)})
	strC_check(t, "", []uint16{0, 0}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 0, 0)})
	strC_check(t, "", []uint16{0, 7}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 0, 7)})
	strC_check(t, "", []uint16{7, 0}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 7, 0)})
	strC_check(t, "", []uint16{10, 5}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 10, 5)})
	strC_check(t, "", []uint16{1, 1}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 1, 1)})
	strC_bytes(t, "\x1b[r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 0, 0)})
	strC_bytes(t, "\x1b[5r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 5, 0)})
	strC_bytes(t, "\x1b[5;10r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 5, 10)})
	// Empty leading parameter is zero; a trailing empty one is dropped.
	strC_bytes(t, "\x1b[;r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 0, 0)})
	strC_bytes(t, "\x1b[;5r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 0, 5)})
	strC_bytes(t, "\x1b[5;r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 5, 0)})
	strC_bytes(t, "\x1b[;;r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 0, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI r DECSTBM boundaries"
func TestStream_term51_Stream_csiDispatch_CSI_r_DECSTBM_boundaries(t *testing.T) {
	strC_check(t, "", []uint16{65535}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 65535, 0)})
	strC_check(t, "", []uint16{65535, 65535}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 65535, 65535)})
	strC_check(t, "", []uint16{1, 65535}, 'r', []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 1, 65535)})
	strC_bytes(t, "\x1b[65535;65535r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 65535, 65535)})
	// One past u16 saturates.
	strC_bytes(t, "\x1b[65536r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 65535, 0)})
	strC_bytes(t, "\x1b[65536;70000r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 65535, 65535)})
	strC_bytes(t, "\x1b[99999999999;1r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 65535, 1)})
	// Three or more parameters are rejected.
	strC_none(t, "", []uint16{1, 2, 3}, 'r')
	strC_none(t, "", []uint16{0, 0, 0}, 'r')
	strC_none(t, "", strC_repeat(1, 30), 'r')
	strC_bytes(t, "\x1b[1;2;3r", nil)
	strC_bytes(t, "\x1b[;;1r", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI ? Pm r restores DEC modes"
func TestStream_term51_Stream_csiDispatch_CSI_Pm_r_restores_DEC_modes(t *testing.T) {
	strC_check(t, "?", []uint16{1}, 'r', []strC_event{strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeCursorKeys), 0)})
	strC_check(t, "?", []uint16{2004}, 'r', []strC_event{strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeBracketedPaste), 0)})
	// Multiple modes: one action per known mode, in order; unknown skipped.
	strC_check(t, "?", []uint16{1, 6, 25}, 'r', []strC_event{strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeCursorKeys), 0), strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeOrigin), 0), strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeCursorVisible), 0)})
	strC_check(t, "?", []uint16{1, 9999, 7}, 'r', []strC_event{strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeCursorKeys), 0), strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeWraparound), 0)})
	strC_check(t, "?", []uint16{7, 7}, 'r', []strC_event{strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeWraparound), 0), strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeWraparound), 0)})
	// No parameters, unknown modes, and ANSI-only numbers emit nothing.
	strC_none(t, "?", nil, 'r')
	strC_none(t, "?", []uint16{0}, 'r')
	strC_none(t, "?", []uint16{42}, 'r')
	strC_none(t, "?", []uint16{2}, 'r')
	strC_none(t, "?", []uint16{20}, 'r')
	strC_none(t, "?", []uint16{65535}, 'r')
	strC_none(t, "?", []uint16{0, 65535, 9999}, 'r')
	strC_bytes(t, "\x1b[?r", nil)
	strC_bytes(t, "\x1b[?42r", nil)
	strC_bytes(t, "\x1b[?65536r", nil)
	strC_bytes(t, "\x1b[?1r", []strC_event{strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeCursorKeys), 0)})
	strC_bytes(t, "\x1b[?1;2004;9999;25r", []strC_event{strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeCursorKeys), 0), strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeBracketedPaste), 0), strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeCursorVisible), 0)})
	// 4 and 12 are DEC modes as well as ANSI ones.
	strC_check(t, "?", []uint16{4, 12}, 'r', []strC_event{strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeSlowScroll), 0), strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeCursorBlinking), 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI ? Pm r with many parameters"
func TestStream_term51_Stream_csiDispatch_CSI_Pm_r_with_many_parameters(t *testing.T) {
	// More than the parser's 24 slot limit goes straight to csiDispatch.
	many := strC_repeat(7, 30)
	var expected [30]strC_event
	for i := range expected {
		expected[i] = strC_ev(StreamActionKeyRestoreMode, strC_mode(ModesModeWraparound), 0)
	}
	strC_check(t, "?", many, 'r', expected[:])
	// Through bytes, 24 parameters dispatch and 25 are dropped by the parser.
	b24 := "\x1b[?7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7r"
	strC_bytes(t, b24, expected[:24])
	b25 := "\x1b[?7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7;7r"
	strC_bytes(t, b25, nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI r other intermediates ignored"
func TestStream_term51_Stream_csiDispatch_CSI_r_other_intermediates_ignored(t *testing.T) {
	for _, c := range []uint8("<=>!\"#$%&'()*+,-./ ") {
		strC_none(t, string([]uint8{c}), nil, 'r')
		strC_none(t, string([]uint8{c}), []uint16{1}, 'r')
		strC_none(t, string([]uint8{c}), []uint16{1, 2}, 'r')
	}
	// Two intermediates, including "?" first.
	multi := []string{"?$", "??", "$?", "?>", "> ", "?!", "$$"}
	for _, inter := range multi {
		strC_none(t, inter, nil, 'r')
		strC_none(t, inter, []uint16{1}, 'r')
		strC_none(t, inter, []uint16{1, 2}, 'r')
	}
	strC_none(t, "?$ ", []uint16{1}, 'r')
	strC_bytes(t, "\x1b[$r", nil)
	strC_bytes(t, "\x1b[1;2$r", nil)
	strC_bytes(t, "\x1b[?$1r", nil)
	strC_bytes(t, "\x1b[>1r", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI s DECSLRM and SCOSC parameter forms"
func TestStream_term51_Stream_csiDispatch_CSI_s_DECSLRM_and_SCOSC_parameter_forms(t *testing.T) {
	// Zero params is ambiguous: the handler decides (SCOSC vs DECSLRM).
	strC_check(t, "", nil, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMarginAmbiguous, 0, 0)})
	strC_check(t, "", []uint16{3}, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 3, 0)})
	strC_check(t, "", []uint16{3, 7}, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 3, 7)})
	strC_check(t, "", []uint16{0}, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 0, 0)})
	strC_check(t, "", []uint16{0, 0}, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 0, 0)})
	strC_check(t, "", []uint16{0, 9}, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 0, 9)})
	strC_check(t, "", []uint16{9, 0}, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 9, 0)})
	strC_check(t, "", []uint16{9, 3}, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 9, 3)})
	strC_bytes(t, "\x1b[s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMarginAmbiguous, 0, 0)})
	strC_bytes(t, "\x1b[3s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 3, 0)})
	strC_bytes(t, "\x1b[3;7s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 3, 7)})
	strC_bytes(t, "\x1b[0s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 0, 0)})
	strC_bytes(t, "\x1b[;s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 0, 0)})
	strC_bytes(t, "\x1b[;7s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 0, 7)})
	strC_bytes(t, "\x1b[3;s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 3, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI s DECSLRM boundaries"
func TestStream_term51_Stream_csiDispatch_CSI_s_DECSLRM_boundaries(t *testing.T) {
	strC_check(t, "", []uint16{65535}, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 65535, 0)})
	strC_check(t, "", []uint16{65535, 65535}, 's', []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 65535, 65535)})
	strC_bytes(t, "\x1b[65535;65535s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 65535, 65535)})
	strC_bytes(t, "\x1b[65536s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 65535, 0)})
	strC_bytes(t, "\x1b[1;65536s", []strC_event{strC_ev(StreamActionKeyLeftAndRightMargin, 1, 65535)})
	// Three or more parameters are rejected.
	strC_none(t, "", []uint16{1, 2, 3}, 's')
	strC_none(t, "", []uint16{0, 0, 0}, 's')
	strC_none(t, "", strC_repeat(1, 30), 's')
	strC_bytes(t, "\x1b[1;2;3s", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI ? Pm s saves DEC modes"
func TestStream_term51_Stream_csiDispatch_CSI_Pm_s_saves_DEC_modes(t *testing.T) {
	strC_check(t, "?", []uint16{1}, 's', []strC_event{strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeCursorKeys), 0)})
	strC_check(t, "?", []uint16{47}, 's', []strC_event{strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeAltScreenLegacy), 0)})
	strC_check(t, "?", []uint16{1, 6, 25}, 's', []strC_event{strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeCursorKeys), 0), strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeOrigin), 0), strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeCursorVisible), 0)})
	strC_check(t, "?", []uint16{9999, 1, 0, 3}, 's', []strC_event{strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeCursorKeys), 0), strC_ev(StreamActionKeySaveMode, strC_mode(ModesMode132Column), 0)})
	strC_none(t, "?", nil, 's')
	strC_none(t, "?", []uint16{0}, 's')
	strC_none(t, "?", []uint16{42}, 's')
	strC_none(t, "?", []uint16{2}, 's')
	strC_none(t, "?", []uint16{20}, 's')
	strC_none(t, "?", []uint16{65535}, 's')
	strC_bytes(t, "\x1b[?s", nil)
	strC_bytes(t, "\x1b[?42s", nil)
	strC_bytes(t, "\x1b[?65536s", nil)
	strC_bytes(t, "\x1b[?1s", []strC_event{strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeCursorKeys), 0)})
	strC_bytes(t, "\x1b[?1;1000;1006s", []strC_event{strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeCursorKeys), 0), strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeMouseEventNormal), 0), strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeMouseFormatSgr), 0)})
	// Parameter limit: 30 direct, 24 via bytes, 25 dropped by the parser.
	many := strC_repeat(25, 30)
	var expected [30]strC_event
	for i := range expected {
		expected[i] = strC_ev(StreamActionKeySaveMode, strC_mode(ModesModeCursorVisible), 0)
	}
	strC_check(t, "?", many, 's', expected[:])
	strC_bytes(t, "\x1b[?25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25s", expected[:24])
	strC_bytes(t, "\x1b[?25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25;25s", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI > s XTSHIFTESCAPE"
func TestStream_term51_Stream_csiDispatch_CSI_s_XTSHIFTESCAPE(t *testing.T) {
	strC_check(t, ">", nil, 's', []strC_event{strC_ev(StreamActionKeyMouseShiftCapture, 0, 0)})
	strC_check(t, ">", []uint16{0}, 's', []strC_event{strC_ev(StreamActionKeyMouseShiftCapture, 0, 0)})
	strC_check(t, ">", []uint16{1}, 's', []strC_event{strC_ev(StreamActionKeyMouseShiftCapture, 1, 0)})
	strC_bytes(t, "\x1b[>s", []strC_event{strC_ev(StreamActionKeyMouseShiftCapture, 0, 0)})
	strC_bytes(t, "\x1b[>0s", []strC_event{strC_ev(StreamActionKeyMouseShiftCapture, 0, 0)})
	strC_bytes(t, "\x1b[>1s", []strC_event{strC_ev(StreamActionKeyMouseShiftCapture, 1, 0)})
	// Invalid values: one past the last valid, and the maximum.
	strC_none(t, ">", []uint16{2}, 's')
	strC_none(t, ">", []uint16{3}, 's')
	strC_none(t, ">", []uint16{65535}, 's')
	strC_bytes(t, "\x1b[>2s", nil)
	strC_bytes(t, "\x1b[>65536s", nil)
	// Two or more parameters are rejected, even valid ones.
	strC_none(t, ">", []uint16{1, 1}, 's')
	strC_none(t, ">", []uint16{0, 0}, 's')
	strC_none(t, ">", []uint16{0, 1}, 's')
	strC_none(t, ">", strC_repeat(1, 30), 's')
	strC_bytes(t, "\x1b[>1;1s", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI s other intermediates ignored"
func TestStream_term51_Stream_csiDispatch_CSI_s_other_intermediates_ignored(t *testing.T) {
	for _, c := range []uint8("<=!\"#$%&'()*+,-./ ") {
		strC_none(t, string([]uint8{c}), nil, 's')
		strC_none(t, string([]uint8{c}), []uint16{1}, 's')
		strC_none(t, string([]uint8{c}), []uint16{1, 2}, 's')
	}
	multi := []string{"?$", "??", ">>", ">?", "?>", "$?", " >"}
	for _, inter := range multi {
		strC_none(t, inter, nil, 's')
		strC_none(t, inter, []uint16{1}, 's')
	}
	strC_bytes(t, "\x1b[<s", nil)
	strC_bytes(t, "\x1b[$s", nil)
	strC_bytes(t, "\x1b[?>1s", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI Ps t every first parameter"
func TestStream_term51_Stream_csiDispatch_CSI_Ps_t_every_first_parameter(t *testing.T) {
	// Single parameter: only 14, 16, 18 and 21 are size reports. 22 and 23
	// need an explicit window target so a lone parameter is ignored.
	p := uint16(0)
	for ; ; p++ {
		var expected []strC_event
		switch p {
		case 14:
			expected = []strC_event{strC_ev(StreamActionKeySizeReport, uint32(CSISizeReportStyleCsi14T), 0)}
		case 16:
			expected = []strC_event{strC_ev(StreamActionKeySizeReport, uint32(CSISizeReportStyleCsi16T), 0)}
		case 18:
			expected = []strC_event{strC_ev(StreamActionKeySizeReport, uint32(CSISizeReportStyleCsi18T), 0)}
		case 21:
			expected = []strC_event{strC_ev(StreamActionKeySizeReport, uint32(CSISizeReportStyleCsi21T), 0)}
		default:
			expected = nil
		}
		strC_check(t, "", []uint16{p}, 't', expected)

		bytes := fmt.Sprintf("\x1b[%dt", p)
		strC_bytes(t, bytes, expected)
		if p == 30 {
			break
		}
	}
	strC_none(t, "", []uint16{65535}, 't')
	strC_bytes(t, "\x1b[65535t", nil)
	strC_bytes(t, "\x1b[65536t", nil)
	// Values near the interesting ones.
	for _, q := range []uint16{13, 15, 17, 19, 20, 24, 31, 100, 1000, 65534} {
		strC_none(t, "", []uint16{q}, 't')
	}
	// A trailing empty parameter is dropped, so this is still "14".
	strC_bytes(t, "\x1b[14;t", []strC_event{strC_ev(StreamActionKeySizeReport, uint32(CSISizeReportStyleCsi14T), 0)})
	strC_bytes(t, "\x1b[22;t", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI Ps t size reports reject extra parameters"
func TestStream_term51_Stream_csiDispatch_CSI_Ps_t_size_reports_reject_extra_parameters(t *testing.T) {
	firsts := []uint16{14, 16, 18, 21}
	for _, f := range firsts {
		strC_none(t, "", []uint16{f, 0}, 't')
		strC_none(t, "", []uint16{f, 1}, 't')
		strC_none(t, "", []uint16{f, 2}, 't')
		strC_none(t, "", []uint16{f, 65535}, 't')
		strC_none(t, "", []uint16{f, 0, 0}, 't')
		strC_none(t, "", []uint16{f, 0, 0, 0}, 't')
		strC_none(t, "", strC_repeat(f, 30), 't')
	}
	strC_bytes(t, "\x1b[14;0t", nil)
	strC_bytes(t, "\x1b[16;1t", nil)
	strC_bytes(t, "\x1b[18;1;1t", nil)
	strC_bytes(t, "\x1b[21;2t", nil)
	strC_bytes(t, "\x1b[14;;t", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI 22 t and 23 t title stack"
func TestStream_term51_Stream_csiDispatch_CSI_22_t_and_23_t_title_stack(t *testing.T) {
	pairs := []struct {
		n   uint16
		tag StreamActionKey
	}{
		{n: 22, tag: StreamActionKeyTitlePush},
		{n: 23, tag: StreamActionKeyTitlePop},
	}
	for _, pr := range pairs {
		n := pr.n
		// Target 0 (icon and window) and 2 (window) are supported, index
		// defaults to 0.
		strC_check(t, "", []uint16{n, 0}, 't', []strC_event{strC_ev(pr.tag, 0, 0)})
		strC_check(t, "", []uint16{n, 2}, 't', []strC_event{strC_ev(pr.tag, 0, 0)})
		// Explicit index, including the u16 maximum.
		strC_check(t, "", []uint16{n, 0, 0}, 't', []strC_event{strC_ev(pr.tag, 0, 0)})
		strC_check(t, "", []uint16{n, 2, 0}, 't', []strC_event{strC_ev(pr.tag, 0, 0)})
		strC_check(t, "", []uint16{n, 0, 1}, 't', []strC_event{strC_ev(pr.tag, 1, 0)})
		strC_check(t, "", []uint16{n, 2, 5}, 't', []strC_event{strC_ev(pr.tag, 5, 0)})
		strC_check(t, "", []uint16{n, 0, 255}, 't', []strC_event{strC_ev(pr.tag, 255, 0)})
		strC_check(t, "", []uint16{n, 2, 256}, 't', []strC_event{strC_ev(pr.tag, 256, 0)})
		strC_check(t, "", []uint16{n, 0, 65534}, 't', []strC_event{strC_ev(pr.tag, 65534, 0)})
		strC_check(t, "", []uint16{n, 2, 65535}, 't', []strC_event{strC_ev(pr.tag, 65535, 0)})
		// Unsupported targets (icon only = 1, others) are ignored.
		for _, tgt := range []uint16{1, 3, 4, 5, 10, 255, 65535} {
			strC_none(t, "", []uint16{n, tgt}, 't')
			strC_none(t, "", []uint16{n, tgt, 0}, 't')
			strC_none(t, "", []uint16{n, tgt, 1}, 't')
		}
		// Wrong parameter counts.
		strC_none(t, "", []uint16{n}, 't')
		strC_none(t, "", []uint16{n, 0, 0, 0}, 't')
		strC_none(t, "", []uint16{n, 2, 1, 1}, 't')
		strC_none(t, "", append([]uint16{n, 0}, strC_repeat(0, 28)...), 't')
	}
}

// Zig: test "term51: Stream.csiDispatch: CSI 22 t and 23 t title stack through bytes"
func TestStream_term51_Stream_csiDispatch_CSI_22_t_and_23_t_title_stack_through_bytes(t *testing.T) {
	strC_bytes(t, "\x1b[22;0t", []strC_event{strC_ev(StreamActionKeyTitlePush, 0, 0)})
	strC_bytes(t, "\x1b[22;2t", []strC_event{strC_ev(StreamActionKeyTitlePush, 0, 0)})
	strC_bytes(t, "\x1b[22;1t", nil)
	strC_bytes(t, "\x1b[22;0;7t", []strC_event{strC_ev(StreamActionKeyTitlePush, 7, 0)})
	strC_bytes(t, "\x1b[22;2;65535t", []strC_event{strC_ev(StreamActionKeyTitlePush, 65535, 0)})
	// One past u16 saturates to 65535 rather than wrapping.
	strC_bytes(t, "\x1b[22;2;65536t", []strC_event{strC_ev(StreamActionKeyTitlePush, 65535, 0)})
	strC_bytes(t, "\x1b[23;0t", []strC_event{strC_ev(StreamActionKeyTitlePop, 0, 0)})
	strC_bytes(t, "\x1b[23;2t", []strC_event{strC_ev(StreamActionKeyTitlePop, 0, 0)})
	strC_bytes(t, "\x1b[23;1t", nil)
	strC_bytes(t, "\x1b[23;2;7t", []strC_event{strC_ev(StreamActionKeyTitlePop, 7, 0)})
	strC_bytes(t, "\x1b[23;0;65535t", []strC_event{strC_ev(StreamActionKeyTitlePop, 65535, 0)})
	strC_bytes(t, "\x1b[23;0;65536t", []strC_event{strC_ev(StreamActionKeyTitlePop, 65535, 0)})
	strC_bytes(t, "\x1b[23;0;1;1t", nil)
	// Empty middle parameter is zero, so "22;;t" is "22;0" -> push.
	strC_bytes(t, "\x1b[22;;t", []strC_event{strC_ev(StreamActionKeyTitlePush, 0, 0)})
	strC_bytes(t, "\x1b[23;;t", []strC_event{strC_ev(StreamActionKeyTitlePop, 0, 0)})
	strC_bytes(t, "\x1b[22;;3t", []strC_event{strC_ev(StreamActionKeyTitlePush, 3, 0)})
	// Empty window target with an explicit index is still target 0.
	strC_bytes(t, "\x1b[22;0;t", []strC_event{strC_ev(StreamActionKeyTitlePush, 0, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI t with intermediates ignored"
func TestStream_term51_Stream_csiDispatch_CSI_t_with_intermediates_ignored(t *testing.T) {
	inters := []string{"?", ">", "<", "=", "$", " ", "!", "\"", "?$", "$?", ">>", "?$ "}
	for _, inter := range inters {
		strC_none(t, inter, []uint16{14}, 't')
		strC_none(t, inter, []uint16{16}, 't')
		strC_none(t, inter, []uint16{18}, 't')
		strC_none(t, inter, []uint16{21}, 't')
		strC_none(t, inter, []uint16{22, 0}, 't')
		strC_none(t, inter, []uint16{22, 2, 1}, 't')
		strC_none(t, inter, []uint16{23, 0}, 't')
		strC_none(t, inter, []uint16{23, 2, 1}, 't')
		strC_none(t, inter, nil, 't')
		strC_none(t, inter, []uint16{0}, 't')
	}
	strC_bytes(t, "\x1b[?14t", nil)
	strC_bytes(t, "\x1b[>22;0t", nil)
	strC_bytes(t, "\x1b[$18t", nil)
	strC_bytes(t, "\x1b[ 21t", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI u restores cursor"
func TestStream_term51_Stream_csiDispatch_CSI_u_restores_cursor(t *testing.T) {
	// Parameters are not examined at all.
	strC_check(t, "", nil, 'u', []strC_event{strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
	strC_check(t, "", []uint16{0}, 'u', []strC_event{strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
	strC_check(t, "", []uint16{1}, 'u', []strC_event{strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
	strC_check(t, "", []uint16{65535}, 'u', []strC_event{strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
	strC_check(t, "", []uint16{1, 2, 3}, 'u', []strC_event{strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
	strC_check(t, "", strC_repeat(1, 30), 'u', []strC_event{strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
	strC_bytes(t, "\x1b[u", []strC_event{strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
	strC_bytes(t, "\x1b[5u", []strC_event{strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
	strC_bytes(t, "\x1b[1;2u", []strC_event{strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI ? u kitty keyboard query"
func TestStream_term51_Stream_csiDispatch_CSI_u_kitty_keyboard_query(t *testing.T) {
	strC_check(t, "?", nil, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardQuery, 0, 0)})
	// Parameters are ignored.
	strC_check(t, "?", []uint16{0}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardQuery, 0, 0)})
	strC_check(t, "?", []uint16{65535}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardQuery, 0, 0)})
	strC_check(t, "?", []uint16{1, 2}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardQuery, 0, 0)})
	strC_check(t, "?", strC_repeat(1, 30), 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardQuery, 0, 0)})
	strC_bytes(t, "\x1b[?u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardQuery, 0, 0)})
	strC_bytes(t, "\x1b[?1u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardQuery, 0, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI > u kitty keyboard push flags"
func TestStream_term51_Stream_csiDispatch_CSI_u_kitty_keyboard_push_flags(t *testing.T) {
	// No parameter pushes zero flags.
	strC_check(t, ">", nil, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPush, 0, 0)})
	strC_bytes(t, "\x1b[>u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardPush, 0, 0)})
	// Every legal flag combination 0..31.
	v := uint16(0)
	for ; v <= 31; v++ {
		strC_check(t, ">", []uint16{v}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPush, uint32(v), 0)})
		bytes := fmt.Sprintf("\x1b[>%du", v)
		strC_bytes(t, bytes, []strC_event{strC_ev(StreamActionKeyKittyKeyboardPush, uint32(v), 0)})
	}
	// 32 and everything above does not fit in 5 bits: ignored.
	for _, p := range []uint16{32, 33, 63, 64, 255, 256, 65534, 65535} {
		strC_none(t, ">", []uint16{p}, 'u')
	}
	strC_bytes(t, "\x1b[>32u", nil)
	strC_bytes(t, "\x1b[>65535u", nil)
	strC_bytes(t, "\x1b[>65536u", nil)
	// With two or more parameters the flags silently become zero.
	strC_check(t, ">", []uint16{1, 1}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPush, 0, 0)})
	strC_check(t, ">", []uint16{31, 0}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPush, 0, 0)})
	strC_check(t, ">", []uint16{32, 0}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPush, 0, 0)})
	strC_check(t, ">", strC_repeat(5, 30), 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPush, 0, 0)})
	strC_bytes(t, "\x1b[>5;5u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardPush, 0, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI < u kitty keyboard pop count"
func TestStream_term51_Stream_csiDispatch_CSI_u_kitty_keyboard_pop_count(t *testing.T) {
	// No parameter pops one entry.
	strC_check(t, "<", nil, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 1, 0)})
	strC_bytes(t, "\x1b[<u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 1, 0)})
	// A single explicit parameter is passed through unchanged (including 0).
	for _, p := range []uint16{0, 1, 2, 7, 8, 9, 31, 32, 255, 256, 65534, 65535} {
		strC_check(t, "<", []uint16{p}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, uint32(p), 0)})
	}
	strC_bytes(t, "\x1b[<0u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 0, 0)})
	strC_bytes(t, "\x1b[<3u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 3, 0)})
	strC_bytes(t, "\x1b[<65535u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 65535, 0)})
	// One past u16 saturates.
	strC_bytes(t, "\x1b[<65536u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 65535, 0)})
	// Two or more parameters fall back to the default of 1.
	strC_check(t, "<", []uint16{3, 4}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 1, 0)})
	strC_check(t, "<", []uint16{0, 0}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 1, 0)})
	strC_check(t, "<", strC_repeat(9, 30), 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 1, 0)})
	strC_bytes(t, "\x1b[<3;4u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardPop, 1, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI = u kitty keyboard set modes"
func TestStream_term51_Stream_csiDispatch_CSI_u_kitty_keyboard_set_modes(t *testing.T) {
	// No parameters: flags 0, mode 1 (set).
	strC_check(t, "=", nil, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, 0, 0)})
	strC_bytes(t, "\x1b[=u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, 0, 0)})
	// One parameter: flags, mode defaults to set.
	strC_check(t, "=", []uint16{5}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, 5, 0)})
	strC_bytes(t, "\x1b[=5u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, 5, 0)})
	// Two parameters: mode 1 set, 2 or, 3 not.
	strC_check(t, "=", []uint16{5, 1}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, 5, 0)})
	strC_check(t, "=", []uint16{5, 2}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetOr, 5, 0)})
	strC_check(t, "=", []uint16{5, 3}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetNot, 5, 0)})
	strC_bytes(t, "\x1b[=5;1u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, 5, 0)})
	strC_bytes(t, "\x1b[=5;2u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetOr, 5, 0)})
	strC_bytes(t, "\x1b[=5;3u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetNot, 5, 0)})
	// Empty flags with explicit mode: flags are zero.
	strC_bytes(t, "\x1b[=;2u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetOr, 0, 0)})
	strC_bytes(t, "\x1b[=;3u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetNot, 0, 0)})
	strC_bytes(t, "\x1b[=;1u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, 0, 0)})
	// Trailing empty parameter is dropped, leaving the default mode.
	strC_bytes(t, "\x1b[=5;u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, 5, 0)})
	// Extra parameters beyond the second are ignored.
	strC_check(t, "=", []uint16{5, 2, 9}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetOr, 5, 0)})
	strC_check(t, "=", []uint16{5, 3, 9, 9}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetNot, 5, 0)})
	strC_check(t, "=", append([]uint16{7, 2}, strC_repeat(0, 28)...), 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetOr, 7, 0)})
	strC_bytes(t, "\x1b[=1;2;3u", []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetOr, 1, 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI = u kitty flags 0..31 with every mode"
func TestStream_term51_Stream_csiDispatch_CSI_u_kitty_flags_0_31_with_every_mode(t *testing.T) {
	v := uint16(0)
	for ; v <= 31; v++ {
		strC_check(t, "=", []uint16{v}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, uint32(v), 0)})
		strC_check(t, "=", []uint16{v, 1}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSet, uint32(v), 0)})
		strC_check(t, "=", []uint16{v, 2}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetOr, uint32(v), 0)})
		strC_check(t, "=", []uint16{v, 3}, 'u', []strC_event{strC_ev(StreamActionKeyKittyKeyboardSetNot, uint32(v), 0)})
	}
	// Flags of 32 and above are invalid, whatever the mode.
	for _, f := range []uint16{32, 33, 64, 255, 256, 65534, 65535} {
		strC_none(t, "=", []uint16{f}, 'u')
		strC_none(t, "=", []uint16{f, 1}, 'u')
		strC_none(t, "=", []uint16{f, 2}, 'u')
		strC_none(t, "=", []uint16{f, 3}, 'u')
		strC_none(t, "=", []uint16{f, 4}, 'u')
	}
	strC_bytes(t, "\x1b[=32u", nil)
	strC_bytes(t, "\x1b[=32;2u", nil)
	strC_bytes(t, "\x1b[=65536;1u", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI = u kitty invalid mode"
func TestStream_term51_Stream_csiDispatch_CSI_u_kitty_invalid_mode(t *testing.T) {
	// Mode 0 and 4+ are rejected; one past the valid range 1..3 is 4.
	for _, m := range []uint16{0, 4, 5, 255, 256, 65534, 65535} {
		strC_none(t, "=", []uint16{0, m}, 'u')
		strC_none(t, "=", []uint16{5, m}, 'u')
		strC_none(t, "=", []uint16{31, m}, 'u')
		strC_none(t, "=", []uint16{5, m, 1}, 'u')
	}
	strC_bytes(t, "\x1b[=5;0u", nil)
	strC_bytes(t, "\x1b[=5;4u", nil)
	strC_bytes(t, "\x1b[=5;65535u", nil)
	strC_bytes(t, "\x1b[=5;65536u", nil)
	// An empty second parameter is a zero, hence an invalid mode.
	strC_bytes(t, "\x1b[=5;;1u", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI u other intermediates ignored"
func TestStream_term51_Stream_csiDispatch_CSI_u_other_intermediates_ignored(t *testing.T) {
	// Unknown single intermediates.
	for _, c := range []uint8("!\"#$%&'()*+,-./ ") {
		strC_none(t, string([]uint8{c}), nil, 'u')
		strC_none(t, string([]uint8{c}), []uint16{1}, 'u')
		strC_none(t, string([]uint8{c}), []uint16{1, 2}, 'u')
	}
	// Two or more intermediates, including the kitty ones.
	multi := []string{"?>", ">?", "??", ">>", "<<", "==", "=>", "?$", " =", "?<="}
	for _, inter := range multi {
		strC_none(t, inter, nil, 'u')
		strC_none(t, inter, []uint16{1}, 'u')
		strC_none(t, inter, []uint16{1, 2}, 'u')
	}
	strC_bytes(t, "\x1b[$u", nil)
	strC_bytes(t, "\x1b[?>u", nil)
	strC_bytes(t, "\x1b[>>1u", nil)
	strC_bytes(t, "\x1b[== 1u", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI @ ICH"
func TestStream_term51_Stream_csiDispatch_CSI_ICH(t *testing.T) {
	// Missing and zero default to one.
	strC_check(t, "", nil, '@', []strC_event{strC_ev(StreamActionKeyInsertBlanks, 1, 0)})
	strC_check(t, "", []uint16{0}, '@', []strC_event{strC_ev(StreamActionKeyInsertBlanks, 1, 0)})
	strC_check(t, "", []uint16{1}, '@', []strC_event{strC_ev(StreamActionKeyInsertBlanks, 1, 0)})
	strC_check(t, "", []uint16{2}, '@', []strC_event{strC_ev(StreamActionKeyInsertBlanks, 2, 0)})
	strC_check(t, "", []uint16{5}, '@', []strC_event{strC_ev(StreamActionKeyInsertBlanks, 5, 0)})
	strC_check(t, "", []uint16{255}, '@', []strC_event{strC_ev(StreamActionKeyInsertBlanks, 255, 0)})
	strC_check(t, "", []uint16{65534}, '@', []strC_event{strC_ev(StreamActionKeyInsertBlanks, 65534, 0)})
	strC_check(t, "", []uint16{65535}, '@', []strC_event{strC_ev(StreamActionKeyInsertBlanks, 65535, 0)})
	strC_bytes(t, "\x1b[@", []strC_event{strC_ev(StreamActionKeyInsertBlanks, 1, 0)})
	strC_bytes(t, "\x1b[0@", []strC_event{strC_ev(StreamActionKeyInsertBlanks, 1, 0)})
	strC_bytes(t, "\x1b[1@", []strC_event{strC_ev(StreamActionKeyInsertBlanks, 1, 0)})
	strC_bytes(t, "\x1b[7@", []strC_event{strC_ev(StreamActionKeyInsertBlanks, 7, 0)})
	strC_bytes(t, "\x1b[65535@", []strC_event{strC_ev(StreamActionKeyInsertBlanks, 65535, 0)})
	// One past u16 saturates.
	strC_bytes(t, "\x1b[65536@", []strC_event{strC_ev(StreamActionKeyInsertBlanks, 65535, 0)})
	strC_bytes(t, "\x1b[;@", []strC_event{strC_ev(StreamActionKeyInsertBlanks, 1, 0)})
	// Two or more parameters are rejected.
	strC_none(t, "", []uint16{1, 1}, '@')
	strC_none(t, "", []uint16{0, 0}, '@')
	strC_none(t, "", []uint16{5, 2}, '@')
	strC_none(t, "", strC_repeat(1, 30), '@')
	strC_bytes(t, "\x1b[1;1@", nil)
	strC_bytes(t, "\x1b[;1@", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI @ with intermediates ignored"
func TestStream_term51_Stream_csiDispatch_CSI_with_intermediates_ignored(t *testing.T) {
	for _, c := range []uint8("?<=>!\"#$%&'()*+,-./ ") {
		strC_none(t, string([]uint8{c}), nil, '@')
		strC_none(t, string([]uint8{c}), []uint16{3}, '@')
		strC_none(t, string([]uint8{c}), []uint16{3, 4}, '@')
	}
	multi := []string{"?$", "  ", ">>", "$ "}
	for _, inter := range multi {
		strC_none(t, inter, nil, '@')
		strC_none(t, inter, []uint16{3}, '@')
	}
	strC_bytes(t, "\x1b[ @", nil)
	strC_bytes(t, "\x1b[3 @", nil)
	strC_bytes(t, "\x1b[?3@", nil)
}

// Zig: test "term51: Stream.csiDispatch: CSI $ } DECSASD"
func TestStream_term51_Stream_csiDispatch_CSI_DECSASD(t *testing.T) {
	main_display := ANSIStatusDisplayMain
	status_line := ANSIStatusDisplayStatusLine
	strC_check(t, "$", []uint16{0}, '}', []strC_event{strC_ev(StreamActionKeyActiveStatusDisplay, uint32(main_display), 0)})
	strC_check(t, "$", []uint16{1}, '}', []strC_event{strC_ev(StreamActionKeyActiveStatusDisplay, uint32(status_line), 0)})
	strC_bytes(t, "\x1b[0$}", []strC_event{strC_ev(StreamActionKeyActiveStatusDisplay, uint32(main_display), 0)})
	strC_bytes(t, "\x1b[1$}", []strC_event{strC_ev(StreamActionKeyActiveStatusDisplay, uint32(status_line), 0)})
	// One past the last valid value and the maximum.
	for _, p := range []uint16{2, 3, 255, 256, 65534, 65535} {
		strC_none(t, "$", []uint16{p}, '}')
	}
	strC_bytes(t, "\x1b[2$}", nil)
	strC_bytes(t, "\x1b[65535$}", nil)
	strC_bytes(t, "\x1b[65536$}", nil)
	// Exactly one parameter is required.
	strC_none(t, "$", nil, '}')
	strC_none(t, "$", []uint16{0, 0}, '}')
	strC_none(t, "$", []uint16{1, 1}, '}')
	strC_none(t, "$", []uint16{0, 1}, '}')
	strC_none(t, "$", strC_repeat(1, 30), '}')
	strC_bytes(t, "\x1b[$}", nil)
	strC_bytes(t, "\x1b[0;1$}", nil)
	// A trailing empty parameter is dropped.
	strC_bytes(t, "\x1b[1;$}", []strC_event{strC_ev(StreamActionKeyActiveStatusDisplay, uint32(status_line), 0)})
}

// Zig: test "term51: Stream.csiDispatch: CSI } wrong intermediates ignored"
func TestStream_term51_Stream_csiDispatch_CSI_wrong_intermediates_ignored(t *testing.T) {
	// No intermediate.
	strC_none(t, "", nil, '}')
	strC_none(t, "", []uint16{0}, '}')
	strC_none(t, "", []uint16{1}, '}')
	strC_bytes(t, "\x1b[0}", nil)
	strC_bytes(t, "\x1b[1}", nil)
	// Other single intermediates.
	for _, c := range []uint8("?<=>!\"#%&'()*+,-./ ") {
		strC_none(t, string([]uint8{c}), []uint16{0}, '}')
		strC_none(t, string([]uint8{c}), []uint16{1}, '}')
	}
	// Two or more intermediates, even if one is "$".
	multi := []string{"$$", "$ ", " $", "?$", "$?", "$>", ">$", "$$$"}
	for _, inter := range multi {
		strC_none(t, inter, []uint16{0}, '}')
		strC_none(t, inter, []uint16{1}, '}')
	}
	strC_bytes(t, "\x1b[1 }", nil)
	strC_bytes(t, "\x1b[?1$}", nil)
	strC_bytes(t, "\x1b[1$$}", nil)
}

// Zig: test "term51: Stream.csiDispatch: unhandled final bytes emit nothing"
func TestStream_term51_Stream_csiDispatch_unhandled_final_bytes_emit_nothing(t *testing.T) {
	// Finals that neither half of csiDispatch handles hit the final `else`.
	finals := "NOQRUVYiovwxyz{|~^_\\]"
	for _, f := range []uint8(finals) {
		strC_none(t, "", nil, f)
		strC_none(t, "", []uint16{1}, f)
		strC_none(t, "", []uint16{1, 2}, f)
		strC_none(t, "?", []uint16{1}, f)
		strC_none(t, "$", []uint16{1}, f)
		strC_none(t, " ", []uint16{1}, f)
		strC_none(t, "?$", []uint16{1}, f)
	}
	strC_bytes(t, "\x1b[v", nil)
	strC_bytes(t, "\x1b[1;2w", nil)
	strC_bytes(t, "\x1b[?5x", nil)
	strC_bytes(t, "\x1b[>y", nil)
	strC_bytes(t, "\x1b[$z", nil)
	strC_bytes(t, "\x1b[1~", nil)
	strC_bytes(t, "\x1b[1;2|", nil)
	strC_bytes(t, "\x1b[ {", nil)
}

// Zig: test "term51: Stream.csiDispatch: colon separators outside SGR are dropped by the parser"
func TestStream_term51_Stream_csiDispatch_colon_separators_outside_SGR_are_dropped_by_the_parser(t *testing.T) {
	// The parser only lets colon/mixed separators through for 'm', so these
	// never reach csiDispatch.
	strC_bytes(t, "\x1b[1:2r", nil)
	strC_bytes(t, "\x1b[22:0t", nil)
	strC_bytes(t, "\x1b[14:1t", nil)
	strC_bytes(t, "\x1b[?1:2s", nil)
	strC_bytes(t, "\x1b[1:2@", nil)
	strC_bytes(t, "\x1b[=1:2u", nil)
	strC_bytes(t, "\x1b[1:2n", nil)
	// The same with a semicolon is accepted.
	strC_bytes(t, "\x1b[1;2r", []strC_event{strC_ev(StreamActionKeyTopAndBottomMargin, 1, 2)})
}

// Zig: test "term51: Stream.csiDispatch: second half sequences in one stream keep order"
func TestStream_term51_Stream_csiDispatch_second_half_sequences_in_one_stream_keep_order(t *testing.T) {
	s := StreamInit(&strC_recorder{})
	zigTry(t, s.NextSlice([]uint8("\x1b[5n\x1b[?996n\x1b[3 q\x1b[2;9r\x1b[22;0;4t\x1b[>5u\x1b[<2u\x1b[=3;2u\x1b[4@\x1b[1$}\x1b[u")))
	strC_expect(t, &s, []strC_event{strC_ev(StreamActionKeyDeviceStatus, strC_dsr(DeviceStatusRequestOperatingStatus), 0), strC_ev(StreamActionKeyDeviceStatus, strC_dsr(DeviceStatusRequestColorScheme), 0), strC_ev(StreamActionKeyCursorStyle, uint32(ANSICursorStyleBlinkingUnderline), 0), strC_ev(StreamActionKeyTopAndBottomMargin, 2, 9), strC_ev(StreamActionKeyTitlePush, 4, 0), strC_ev(StreamActionKeyKittyKeyboardPush, 5, 0), strC_ev(StreamActionKeyKittyKeyboardPop, 2, 0), strC_ev(StreamActionKeyKittyKeyboardSetOr, 3, 0), strC_ev(StreamActionKeyInsertBlanks, 4, 0), strC_ev(StreamActionKeyActiveStatusDisplay, uint32(ANSIStatusDisplayStatusLine), 0), strC_ev(StreamActionKeyRestoreCursor, 0, 0)})
}
