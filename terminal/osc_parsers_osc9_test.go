// Tests translated one for one from src/terminal/osc/parsers/osc9.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers expect,
// expectEqual, expectEqualSlices, expectEqualStrings and zigUnwrap are defined
// in Parser_test.go. The helpers of this file are prefixed oscB_osc9_.
//
// Translation notes:
//   - `var p: Parser = .init(null)` is OSCParserInit(nil) and
//     `.init(testing.allocator)` is OSCParserInit(oscB_osc9_allocator): the
//     port keeps only whether an allocator was given (osc.go). The leak check
//     (`defer p.deinit()` under testing.allocator) is kept as p.Deinit().
//   - `p.end('\x1b').?.*` is oscB_osc9_end: it fails the test when End
//     returns nil (Zig's `.?`) and copies the command (Zig's `.*`).
//     `parse(&p, null).?.*` is oscB_osc9_parse likewise.
//   - A `[:0]const u8` payload is a []uint8; the sentinel `ptr[len]` read of
//     the term51 tests is oscB_osc9_sentinel, which re-slices one byte past
//     the end and so needs the capacity to hold the NUL.
//   - `try testing.expectEqual(expected_command, cmd)` on a Command union is
//     oscB_osc9_expectEqualCommand: Command holds slices and is not
//     comparable, so it compares the active tag and payload as Zig does.
//   - `inline for` over a tuple of cases is a loop over a slice of structs.

package terminal

import (
	"strings"
	"testing"
)

// oscB_osc9_allocator stands for `testing.allocator`: any non-nil value
// (OSCParser.alloc, osc.go).
var oscB_osc9_allocator any = struct{}{}

// oscB_osc9_feed is `for (input) |ch| p.next(ch)`.
func oscB_osc9_feed(p *OSCParser, input string) {
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
}

// oscB_osc9_deref is Zig's `.?.*` on a ?*Command.
func oscB_osc9_deref(t *testing.T, c *OSCCommand) OSCCommand {
	t.Helper()
	if c == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return *c
}

// oscB_osc9_end is `p.end(ch).?.*`.
func oscB_osc9_end(t *testing.T, p *OSCParser, ch uint8) OSCCommand {
	t.Helper()
	return oscB_osc9_deref(t, p.End(Some(ch)))
}

// oscB_osc9_parse is `parse(&p, null).?.*`.
func oscB_osc9_parse(t *testing.T, p *OSCParser) OSCCommand {
	t.Helper()
	return oscB_osc9_deref(t, OSCParsersOsc9Parse(p, None[uint8]()))
}

// oscB_osc9_sentinel is `s.ptr[s.len]`, the byte after the end of a
// [:0]const u8, which must be the NUL.
func oscB_osc9_sentinel(t *testing.T, s []uint8) uint8 {
	t.Helper()
	if cap(s) <= len(s) {
		t.Fatalf("no sentinel byte after a slice of length %d (capacity %d)", len(s), cap(s))
	}
	return s[:len(s)+1][len(s)]
}

// oscB_osc9_expectEqualCommand is `try testing.expectEqual(expected, actual)`
// for a Command that is a conemu_change_tab_title.
func oscB_osc9_expectEqualCommand(t *testing.T, want OSCCommand, got OSCCommand) {
	t.Helper()
	expectEqual(t, want.Tag, got.Tag)
	expect(t, want.Tag == OSCCommandKeyConemuChangeTabTitle, "expected command is a conemu_change_tab_title")
	expectEqual(t, want.ConemuChangeTabTitle.Tag, got.ConemuChangeTabTitle.Tag)
	if want.ConemuChangeTabTitle.Tag == OSCCommandConemuChangeTabTitlePayloadValue {
		expectEqualSlices(t, want.ConemuChangeTabTitle.Value, got.ConemuChangeTabTitle.Value)
	}
}

// Zig: test "OSC 9: show desktop notification"
func TestOSCParsersOsc9_OSC_9_show_desktop_notification(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;Hello world"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "", string(cmd.ShowDesktopNotification.title))
	expectEqualStrings(t, "Hello world", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9: show single character desktop notification"
func TestOSCParsersOsc9_OSC_9_show_single_character_desktop_notification(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;H"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "", string(cmd.ShowDesktopNotification.title))
	expectEqualStrings(t, "H", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;1: ConEmu sleep"
func TestOSCParsersOsc9_OSC_9_1_ConEmu_sleep(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;1;420"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuSleep, "cmd == .conemu_sleep")
	expectEqual(t, uint16(420), cmd.ConemuSleep.duration_ms)
}

// Zig: test "OSC 9;1: ConEmu sleep with no value default to 100ms"
func TestOSCParsersOsc9_OSC_9_1_ConEmu_sleep_with_no_value_default_to_100ms(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;1;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuSleep, "cmd == .conemu_sleep")
	expectEqual(t, uint16(100), cmd.ConemuSleep.duration_ms)
}

// Zig: test "OSC 9;1: conemu sleep cannot exceed 10000ms"
func TestOSCParsersOsc9_OSC_9_1_conemu_sleep_cannot_exceed_10000ms(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;1;12345"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuSleep, "cmd == .conemu_sleep")
	expectEqual(t, uint16(10000), cmd.ConemuSleep.duration_ms)
}

// Zig: test "OSC 9;1: conemu sleep invalid input"
func TestOSCParsersOsc9_OSC_9_1_conemu_sleep_invalid_input(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;1;foo"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuSleep, "cmd == .conemu_sleep")
	expectEqual(t, uint16(100), cmd.ConemuSleep.duration_ms)
}

// Zig: test "OSC 9;1: conemu sleep -> desktop notification 1"
func TestOSCParsersOsc9_OSC_9_1_conemu_sleep_desktop_notification_1(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;1"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "1", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;1: conemu sleep -> desktop notification 2"
func TestOSCParsersOsc9_OSC_9_1_conemu_sleep_desktop_notification_2(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;1a"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "1a", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;2: ConEmu message box"
func TestOSCParsersOsc9_OSC_9_2_ConEmu_message_box(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;2;hello world"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuShowMessageBox, "cmd == .conemu_show_message_box")
	expectEqualStrings(t, "hello world", string(cmd.ConemuShowMessageBox))
}

// Zig: test "OSC 9;2: ConEmu message box invalid input"
func TestOSCParsersOsc9_OSC_9_2_ConEmu_message_box_invalid_input(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;2"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "2", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;2: ConEmu message box empty message"
func TestOSCParsersOsc9_OSC_9_2_ConEmu_message_box_empty_message(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;2;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuShowMessageBox, "cmd == .conemu_show_message_box")
	expectEqualStrings(t, "", string(cmd.ConemuShowMessageBox))
}

// Zig: test "OSC 9;2: ConEmu message box spaces only message"
func TestOSCParsersOsc9_OSC_9_2_ConEmu_message_box_spaces_only_message(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;2;   "
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuShowMessageBox, "cmd == .conemu_show_message_box")
	expectEqualStrings(t, "   ", string(cmd.ConemuShowMessageBox))
}

// Zig: test "OSC 9;2: message box -> desktop notification 1"
func TestOSCParsersOsc9_OSC_9_2_message_box_desktop_notification_1(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;2"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "2", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;2: message box -> desktop notification 2"
func TestOSCParsersOsc9_OSC_9_2_message_box_desktop_notification_2(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;2a"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "2a", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;3: ConEmu change tab title"
func TestOSCParsersOsc9_OSC_9_3_ConEmu_change_tab_title(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;3;foo bar"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuChangeTabTitle, "cmd == .conemu_change_tab_title")
	expectEqualStrings(t, "foo bar", string(cmd.ConemuChangeTabTitle.Value))
}

// Zig: test "OSC 9;3: ConEmu change tab title reset"
func TestOSCParsersOsc9_OSC_9_3_ConEmu_change_tab_title_reset(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;3;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expectedCommand := OSCCommand{Tag: OSCCommandKeyConemuChangeTabTitle, ConemuChangeTabTitle: OSCCommandConemuChangeTabTitlePayload{Tag: OSCCommandConemuChangeTabTitlePayloadReset}}
	oscB_osc9_expectEqualCommand(t, expectedCommand, cmd)
}

// Zig: test "OSC 9;3: ConEmu change tab title spaces only"
func TestOSCParsersOsc9_OSC_9_3_ConEmu_change_tab_title_spaces_only(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;3;   "
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuChangeTabTitle, "cmd == .conemu_change_tab_title")
	expectEqualStrings(t, "   ", string(cmd.ConemuChangeTabTitle.Value))
}

// Zig: test "OSC 9;3: change tab title -> desktop notification 1"
func TestOSCParsersOsc9_OSC_9_3_change_tab_title_desktop_notification_1(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;3"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "3", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;3: message box -> desktop notification 2"
func TestOSCParsersOsc9_OSC_9_3_message_box_desktop_notification_2(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;3a"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "3a", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;4: ConEmu progress set"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_set(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;1;100"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateSet, "cmd.conemu_progress_report.state == .set")
	expect(t, cmd.ConemuProgressReport.progress == Some[uint8](100), "cmd.conemu_progress_report.progress == 100")
}

// Zig: test "OSC 9;4: ConEmu progress set overflow"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_set_overflow(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;1;900"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateSet, "cmd.conemu_progress_report.state == .set")
	expectEqual(t, Some[uint8](100), cmd.ConemuProgressReport.progress)
}

// Zig: test "OSC 9;4: ConEmu progress set single digit"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_set_single_digit(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;1;9"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateSet, "cmd.conemu_progress_report.state == .set")
	expect(t, cmd.ConemuProgressReport.progress == Some[uint8](9), "cmd.conemu_progress_report.progress == 9")
}

// Zig: test "OSC 9;4: ConEmu progress set double digit"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_set_double_digit(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;1;94"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateSet, "cmd.conemu_progress_report.state == .set")
	expectEqual(t, Some[uint8](94), cmd.ConemuProgressReport.progress)
}

// Zig: test "OSC 9;4: ConEmu progress set extra semicolon ignored"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_set_extra_semicolon_ignored(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;1;100"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateSet, "cmd.conemu_progress_report.state == .set")
	expectEqual(t, Some[uint8](100), cmd.ConemuProgressReport.progress)
}

// Zig: test "OSC 9;4: ConEmu progress remove with no progress"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_remove_with_no_progress(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;0;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateRemove, "cmd.conemu_progress_report.state == .remove")
	expect(t, !cmd.ConemuProgressReport.progress.Valid, "cmd.conemu_progress_report.progress == null")
}

// Zig: test "OSC 9;4: ConEmu progress remove with double semicolon"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_remove_with_double_semicolon(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;0;;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateRemove, "cmd.conemu_progress_report.state == .remove")
	expect(t, !cmd.ConemuProgressReport.progress.Valid, "cmd.conemu_progress_report.progress == null")
}

// Zig: test "OSC 9;4: ConEmu progress remove ignores progress"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_remove_ignores_progress(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;0;100"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateRemove, "cmd.conemu_progress_report.state == .remove")
	expect(t, !cmd.ConemuProgressReport.progress.Valid, "cmd.conemu_progress_report.progress == null")
}

// Zig: test "OSC 9;4: ConEmu progress remove extra semicolon"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_remove_extra_semicolon(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;0;100;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateRemove, "cmd.conemu_progress_report.state == .remove")
}

// Zig: test "OSC 9;4: ConEmu progress error"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_error(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;2"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateError, "cmd.conemu_progress_report.state == .@\"error\"")
	expect(t, !cmd.ConemuProgressReport.progress.Valid, "cmd.conemu_progress_report.progress == null")
}

// Zig: test "OSC 9;4: ConEmu progress error with progress"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_error_with_progress(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;2;100"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateError, "cmd.conemu_progress_report.state == .@\"error\"")
	expect(t, cmd.ConemuProgressReport.progress == Some[uint8](100), "cmd.conemu_progress_report.progress == 100")
}

// Zig: test "OSC 9;4: progress pause"
func TestOSCParsersOsc9_OSC_9_4_progress_pause(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;4"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStatePause, "cmd.conemu_progress_report.state == .pause")
	expect(t, !cmd.ConemuProgressReport.progress.Valid, "cmd.conemu_progress_report.progress == null")
}

// Zig: test "OSC 9;4: ConEmu progress pause with progress"
func TestOSCParsersOsc9_OSC_9_4_ConEmu_progress_pause_with_progress(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;4;100"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
	expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStatePause, "cmd.conemu_progress_report.state == .pause")
	expect(t, cmd.ConemuProgressReport.progress == Some[uint8](100), "cmd.conemu_progress_report.progress == 100")
}

// Zig: test "OSC 9;4: progress -> desktop notification 1"
func TestOSCParsersOsc9_OSC_9_4_progress_desktop_notification_1(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "4", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;4: progress -> desktop notification 2"
func TestOSCParsersOsc9_OSC_9_4_progress_desktop_notification_2(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "4;", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;4: progress -> desktop notification 3"
func TestOSCParsersOsc9_OSC_9_4_progress_desktop_notification_3(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;5"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "4;5", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;4: progress -> desktop notification 4"
func TestOSCParsersOsc9_OSC_9_4_progress_desktop_notification_4(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;4;5a"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "4;5a", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC 9;5: ConEmu wait input"
func TestOSCParsersOsc9_OSC_9_5_ConEmu_wait_input(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;5"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuWaitInput, "cmd == .conemu_wait_input")
}

// Zig: test "OSC 9;5: ConEmu wait ignores trailing characters"
func TestOSCParsersOsc9_OSC_9_5_ConEmu_wait_ignores_trailing_characters(t *testing.T) {
	p := OSCParserInit(nil)

	input := "9;5;foo"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuWaitInput, "cmd == .conemu_wait_input")
}

// Zig: test "OSC 9;6: ConEmu guimacro 1"
func TestOSCParsersOsc9_OSC_9_6_ConEmu_guimacro_1(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;6;a"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuGuimacro, "cmd == .conemu_guimacro")
	expectEqualStrings(t, "a", string(cmd.ConemuGuimacro))
}

// Zig: test "OSC: 9;6: ConEmu guimacro 2"
func TestOSCParsersOsc9_OSC_9_6_ConEmu_guimacro_2(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;6;ab"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuGuimacro, "cmd == .conemu_guimacro")
	expectEqualStrings(t, "ab", string(cmd.ConemuGuimacro))
}

// Zig: test "OSC: 9;6: ConEmu guimacro 3 incomplete -> desktop notification"
func TestOSCParsersOsc9_OSC_9_6_ConEmu_guimacro_3_incomplete_desktop_notification(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;6"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "6", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC: 9;7: ConEmu run process 1"
func TestOSCParsersOsc9_OSC_9_7_ConEmu_run_process_1(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;7;ab"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuRunProcess, "cmd == .conemu_run_process")
	expectEqualStrings(t, "ab", string(cmd.ConemuRunProcess))
}

// Zig: test "OSC: 9;7: ConEmu run process 2"
func TestOSCParsersOsc9_OSC_9_7_ConEmu_run_process_2(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;7;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuRunProcess, "cmd == .conemu_run_process")
	expectEqualStrings(t, "", string(cmd.ConemuRunProcess))
}

// Zig: test "OSC: 9;7: ConEmu run process incomplete -> desktop notification"
func TestOSCParsersOsc9_OSC_9_7_ConEmu_run_process_incomplete_desktop_notification(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;7"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "7", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC: 9;8: ConEmu output environment variable 1"
func TestOSCParsersOsc9_OSC_9_8_ConEmu_output_environment_variable_1(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;8;ab"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuOutputEnvironmentVariable, "cmd == .conemu_output_environment_variable")
	expectEqualStrings(t, "ab", string(cmd.ConemuOutputEnvironmentVariable))
}

// Zig: test "OSC: 9;8: ConEmu output environment variable 2"
func TestOSCParsersOsc9_OSC_9_8_ConEmu_output_environment_variable_2(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;8;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuOutputEnvironmentVariable, "cmd == .conemu_output_environment_variable")
	expectEqualStrings(t, "", string(cmd.ConemuOutputEnvironmentVariable))
}

// Zig: test "OSC: 9;8: ConEmu output environment variable incomplete -> desktop notification"
func TestOSCParsersOsc9_OSC_9_8_ConEmu_output_environment_variable_incomplete_desktop_notification(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;8"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "8", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC: 9;9: ConEmu set current working directory"
func TestOSCParsersOsc9_OSC_9_9_ConEmu_set_current_working_directory(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;9;ab"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
	expectEqualStrings(t, "ab", string(cmd.ReportPwd.value))
}

// Zig: test "OSC: 9;9: ConEmu set current working directory incomplete -> desktop notification"
func TestOSCParsersOsc9_OSC_9_9_ConEmu_set_current_working_directory_incomplete_desktop_notification(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;9"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "9", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC: 9;10: ConEmu xterm keyboard and output emulation 1"
func TestOSCParsersOsc9_OSC_9_10_ConEmu_xterm_keyboard_and_output_emulation_1(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;10"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuXtermEmulation, "cmd == .conemu_xterm_emulation")
	expect(t, cmd.ConemuXtermEmulation.keyboard.Valid, "cmd.conemu_xterm_emulation.keyboard != null")
	expect(t, cmd.ConemuXtermEmulation.keyboard.Value == true, "cmd.conemu_xterm_emulation.keyboard.? == true")
	expect(t, cmd.ConemuXtermEmulation.output.Valid, "cmd.conemu_xterm_emulation.output != null")
	expect(t, cmd.ConemuXtermEmulation.output.Value == true, "cmd.conemu_xterm_emulation.output.? == true")
}

// Zig: test "OSC: 9;10: ConEmu xterm keyboard and output emulation 2"
func TestOSCParsersOsc9_OSC_9_10_ConEmu_xterm_keyboard_and_output_emulation_2(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;10;0"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuXtermEmulation, "cmd == .conemu_xterm_emulation")
	expect(t, cmd.ConemuXtermEmulation.keyboard.Valid, "cmd.conemu_xterm_emulation.keyboard != null")
	expect(t, cmd.ConemuXtermEmulation.keyboard.Value == false, "cmd.conemu_xterm_emulation.keyboard.? == false")
	expect(t, cmd.ConemuXtermEmulation.output.Valid, "cmd.conemu_xterm_emulation.output != null")
	expect(t, cmd.ConemuXtermEmulation.output.Value == false, "cmd.conemu_xterm_emulation.output.? == false")
}

// Zig: test "OSC: 9;10: ConEmu xterm keyboard and output emulation 3"
func TestOSCParsersOsc9_OSC_9_10_ConEmu_xterm_keyboard_and_output_emulation_3(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;10;1"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuXtermEmulation, "cmd == .conemu_xterm_emulation")
	expect(t, cmd.ConemuXtermEmulation.keyboard.Valid, "cmd.conemu_xterm_emulation.keyboard != null")
	expect(t, cmd.ConemuXtermEmulation.keyboard.Value == true, "cmd.conemu_xterm_emulation.keyboard.? == true")
	expect(t, cmd.ConemuXtermEmulation.output.Valid, "cmd.conemu_xterm_emulation.output != null")
	expect(t, cmd.ConemuXtermEmulation.output.Value == true, "cmd.conemu_xterm_emulation.output.? == true")
}

// Zig: test "OSC: 9;10: ConEmu xterm keyboard and output emulation 4"
func TestOSCParsersOsc9_OSC_9_10_ConEmu_xterm_keyboard_and_output_emulation_4(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;10;2"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuXtermEmulation, "cmd == .conemu_xterm_emulation")
	expect(t, !cmd.ConemuXtermEmulation.keyboard.Valid, "cmd.conemu_xterm_emulation.keyboard == null")
	expect(t, cmd.ConemuXtermEmulation.output.Valid, "cmd.conemu_xterm_emulation.output != null")
	expect(t, cmd.ConemuXtermEmulation.output.Value == false, "cmd.conemu_xterm_emulation.output.? == false")
}

// Zig: test "OSC: 9;10: ConEmu xterm keyboard and output emulation 5"
func TestOSCParsersOsc9_OSC_9_10_ConEmu_xterm_keyboard_and_output_emulation_5(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;10;3"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuXtermEmulation, "cmd == .conemu_xterm_emulation")
	expect(t, !cmd.ConemuXtermEmulation.keyboard.Valid, "cmd.conemu_xterm_emulation.keyboard == null")
	expect(t, cmd.ConemuXtermEmulation.output.Valid, "cmd.conemu_xterm_emulation.output != null")
	expect(t, cmd.ConemuXtermEmulation.output.Value == true, "cmd.conemu_xterm_emulation.output.? == true")
}

// Zig: test "OSC: 9;10: ConEmu xterm keyboard and output emulation 6"
func TestOSCParsersOsc9_OSC_9_10_ConEmu_xterm_keyboard_and_output_emulation_6(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;10;4"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "10;4", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC: 9;10: ConEmu xterm keyboard and output emulation 7"
func TestOSCParsersOsc9_OSC_9_10_ConEmu_xterm_keyboard_and_output_emulation_7(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;10;"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "10;", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC: 9;10: ConEmu xterm keyboard and output emulation 8"
func TestOSCParsersOsc9_OSC_9_10_ConEmu_xterm_keyboard_and_output_emulation_8(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;10;abc"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "10;abc", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC: 9;11: ConEmu comment"
func TestOSCParsersOsc9_OSC_9_11_ConEmu_comment(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;11;ab"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyConemuComment, "cmd == .conemu_comment")
	expectEqualStrings(t, "ab", string(cmd.ConemuComment))
}

// Zig: test "OSC: 9;11: ConEmu comment incomplete -> desktop notification"
func TestOSCParsersOsc9_OSC_9_11_ConEmu_comment_incomplete_desktop_notification(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;11"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "11", string(cmd.ShowDesktopNotification.body))
}

// Zig: test "OSC: 9;12: ConEmu mark prompt start 1"
func TestOSCParsersOsc9_OSC_9_12_ConEmu_mark_prompt_start_1(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;12"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
}

// Zig: test "OSC: 9;12: ConEmu mark prompt start 2"
func TestOSCParsersOsc9_OSC_9_12_ConEmu_mark_prompt_start_2(t *testing.T) {
	p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
	defer p.Deinit()

	input := "9;12;abc"
	oscB_osc9_feed(&p, input)

	cmd := oscB_osc9_end(t, &p, '\x1b')
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

func oscB_osc9_term51Feed(p *OSCParser, input string) {
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
}

// oscB_osc9_term51ExpectNotification expects the input to be parsed as an
// iTerm2 desktop notification with the given body.
func oscB_osc9_term51ExpectNotification(t *testing.T, input string, body string) {
	t.Helper()
	p := OSCParserInit(nil)
	oscB_osc9_term51Feed(&p, input)
	cmd := oscB_osc9_parse(t, &p)
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, "", string(cmd.ShowDesktopNotification.title))
	expectEqualSlices(t, []uint8(body), cmd.ShowDesktopNotification.body)
	expectEqual(t, uint8(0), oscB_osc9_sentinel(t, cmd.ShowDesktopNotification.body))
}

// Zig: test "term51: parse: missing separator after 9 is invalid"
func TestOSCParsersOsc9_term51_parse_missing_separator_after_9_is_invalid(t *testing.T) {
	p := OSCParserInit(nil)
	oscB_osc9_term51Feed(&p, "9")
	expect(t, p.writer == nil, "p.writer == null")
	expect(t, OSCParsersOsc9Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
}

// Zig: test "term51: parse: notification body"
func TestOSCParsersOsc9_term51_parse_notification_body(t *testing.T) {
	// An empty body is still a notification.
	oscB_osc9_term51ExpectNotification(t, "9;", "")
	oscB_osc9_term51ExpectNotification(t, "9;Hello", "Hello")
	// Semicolons and arbitrary bytes are kept.
	oscB_osc9_term51ExpectNotification(t, "9;a;b;;c", "a;b;;c")
	oscB_osc9_term51ExpectNotification(t, "9;a\x00\x01\xff", "a\x00\x01\xff")
	// A first byte that is not a ConEmu digit.
	oscB_osc9_term51ExpectNotification(t, "9;0", "0")
	oscB_osc9_term51ExpectNotification(t, "9;0;1", "0;1")
	oscB_osc9_term51ExpectNotification(t, "9; 1;100", " 1;100")
}

// Zig: test "term51: parse: terminator is ignored"
func TestOSCParsersOsc9_term51_parse_terminator_is_ignored(t *testing.T) {
	for _, term := range []Optional[uint8]{None[uint8](), Some[uint8](0x07), Some[uint8](0x1b)} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, "9;Hello")
		cmd := oscB_osc9_deref(t, OSCParsersOsc9Parse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
		expectEqualStrings(t, "Hello", string(cmd.ShowDesktopNotification.body))
	}
}

// Zig: test "term51: parse: sleep duration"
func TestOSCParsersOsc9_term51_parse_sleep_duration(t *testing.T) {
	for _, c := range []struct {
		input string
		want  uint16
	}{
		{"9;1;0", 0},
		{"9;1;1", 1},
		{"9;1;9999", 9999},
		// values up to the u16 maximum are clamped to 10000
		{"9;1;10000", 10000},
		{"9;1;10001", 10000},
		{"9;1;65535", 10000},
		// values that do not fit in a u16 are a parse error and give the
		// 100ms default instead of the 10000ms maximum
		{"9;1;65536", 100},
		{"9;1;99999999999999999999", 100},
		// missing or malformed values give the default
		{"9;1;", 100},
		{"9;1;abc", 100},
		{"9;1;-1", 100},
		{"9;1;+5", 100},
		{"9;1; 5", 100},
		{"9;1;5 ", 100},
		{"9;1;5;6", 100},
		{"9;1;5.5", 100},
		// std.fmt.parseUnsigned accepts leading zeros and '_' between
		// digits
		{"9;1;0005", 5},
		{"9;1;1_0", 10},
	} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, c.input)
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuSleep, "cmd == .conemu_sleep")
		expectEqual(t, c.want, cmd.ConemuSleep.duration_ms)
	}
}

// Zig: test "term51: parse: a 1 that is not a ConEmu command is a notification"
func TestOSCParsersOsc9_term51_parse_a_1_that_is_not_a_ConEmu_command_is_a_notification(t *testing.T) {
	oscB_osc9_term51ExpectNotification(t, "9;1", "1")
	oscB_osc9_term51ExpectNotification(t, "9;1x", "1x")
	oscB_osc9_term51ExpectNotification(t, "9;13", "13")
	oscB_osc9_term51ExpectNotification(t, "9;19;x", "19;x")
	oscB_osc9_term51ExpectNotification(t, "9;1 ;5", "1 ;5")
}

// Zig: test "term51: parse: xterm emulation"
func TestOSCParsersOsc9_term51_parse_xterm_emulation(t *testing.T) {
	for _, c := range []struct {
		input    string
		keyboard Optional[bool]
		output   Optional[bool]
	}{
		{"9;10", Some(true), Some(true)},
		{"9;10;0", Some(false), Some(false)},
		{"9;10;1", Some(true), Some(true)},
		{"9;10;2", None[bool](), Some(false)},
		{"9;10;3", None[bool](), Some(true)},
		// anything after the digit is ignored
		{"9;10;0abc", Some(false), Some(false)},
		{"9;10;3;x", None[bool](), Some(true)},
		{"9;10;25", None[bool](), Some(false)},
	} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, c.input)
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuXtermEmulation, "cmd == .conemu_xterm_emulation")
		expectEqual(t, c.keyboard, cmd.ConemuXtermEmulation.keyboard)
		expectEqual(t, c.output, cmd.ConemuXtermEmulation.output)
	}

	oscB_osc9_term51ExpectNotification(t, "9;10;", "10;")
	oscB_osc9_term51ExpectNotification(t, "9;10;4", "10;4")
	oscB_osc9_term51ExpectNotification(t, "9;10;a", "10;a")
	oscB_osc9_term51ExpectNotification(t, "9;10x", "10x")
	oscB_osc9_term51ExpectNotification(t, "9;10x0", "10x0")
	oscB_osc9_term51ExpectNotification(t, "9;100", "100")
	oscB_osc9_term51ExpectNotification(t, "9;10; 1", "10; 1")
}

// Zig: test "term51: parse: comment"
func TestOSCParsersOsc9_term51_parse_comment(t *testing.T) {
	for _, c := range []struct {
		input string
		want  string
	}{
		{"9;11;", ""},
		{"9;11;hello", "hello"},
		{"9;11;a;b\x01\xff", "a;b\x01\xff"},
	} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, c.input)
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuComment, "cmd == .conemu_comment")
		expectEqualSlices(t, []uint8(c.want), cmd.ConemuComment)
		expectEqual(t, uint8(0), oscB_osc9_sentinel(t, cmd.ConemuComment))
	}

	oscB_osc9_term51ExpectNotification(t, "9;11", "11")
	oscB_osc9_term51ExpectNotification(t, "9;11x", "11x")
	oscB_osc9_term51ExpectNotification(t, "9;111;x", "111;x")
}

// Zig: test "term51: parse: 12 followed by anything is a prompt mark"
func TestOSCParsersOsc9_term51_parse_12_followed_by_anything_is_a_prompt_mark(t *testing.T) {
	// Nothing after the "12" is checked, so notifications that happen to
	// start with "12" are swallowed.
	for _, input := range []string{"9;12", "9;12;", "9;12;x", "9;12x", "9;123", "9;12 files copied"} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, input)
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
		expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
		expectEqualStrings(t, "", string(cmd.SemanticPrompt.options_unvalidated))
	}
}

// Zig: test "term51: parse: message box, guimacro, run process, environment variable"
func TestOSCParsersOsc9_term51_parse_message_box_guimacro_run_process_environment_variable(t *testing.T) {
	for _, c := range []struct {
		input string
		want  string
	}{
		{"9;2;", ""},
		{"9;2;hello", "hello"},
		{"9;2;a;b\x01\xff", "a;b\x01\xff"},
	} {
		{
			p := OSCParserInit(nil)
			oscB_osc9_term51Feed(&p, c.input)
			cmd := oscB_osc9_parse(t, &p)
			expect(t, cmd.Tag == OSCCommandKeyConemuShowMessageBox, "cmd == .conemu_show_message_box")
			expectEqualSlices(t, []uint8(c.want), cmd.ConemuShowMessageBox)
			expectEqual(t, uint8(0), oscB_osc9_sentinel(t, cmd.ConemuShowMessageBox))
		}
		{
			p := OSCParserInit(nil)
			oscB_osc9_term51Feed(&p, "9;6"+c.input[3:])
			cmd := oscB_osc9_parse(t, &p)
			expect(t, cmd.Tag == OSCCommandKeyConemuGuimacro, "cmd == .conemu_guimacro")
			expectEqualSlices(t, []uint8(c.want), cmd.ConemuGuimacro)
		}
		{
			p := OSCParserInit(nil)
			oscB_osc9_term51Feed(&p, "9;7"+c.input[3:])
			cmd := oscB_osc9_parse(t, &p)
			expect(t, cmd.Tag == OSCCommandKeyConemuRunProcess, "cmd == .conemu_run_process")
			expectEqualSlices(t, []uint8(c.want), cmd.ConemuRunProcess)
		}
		{
			p := OSCParserInit(nil)
			oscB_osc9_term51Feed(&p, "9;8"+c.input[3:])
			cmd := oscB_osc9_parse(t, &p)
			expect(t, cmd.Tag == OSCCommandKeyConemuOutputEnvironmentVariable, "cmd == .conemu_output_environment_variable")
			expectEqualSlices(t, []uint8(c.want), cmd.ConemuOutputEnvironmentVariable)
		}
		{
			// 9;9 reports the working directory; an empty value is accepted.
			p := OSCParserInit(nil)
			oscB_osc9_term51Feed(&p, "9;9"+c.input[3:])
			cmd := oscB_osc9_parse(t, &p)
			expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
			expectEqualSlices(t, []uint8(c.want), cmd.ReportPwd.value)
		}
	}

	// Without the separator all of them are notifications.
	oscB_osc9_term51ExpectNotification(t, "9;2", "2")
	oscB_osc9_term51ExpectNotification(t, "9;2x", "2x")
	oscB_osc9_term51ExpectNotification(t, "9;6", "6")
	oscB_osc9_term51ExpectNotification(t, "9;6x", "6x")
	oscB_osc9_term51ExpectNotification(t, "9;7", "7")
	oscB_osc9_term51ExpectNotification(t, "9;7x", "7x")
	oscB_osc9_term51ExpectNotification(t, "9;8", "8")
	oscB_osc9_term51ExpectNotification(t, "9;8x", "8x")
	oscB_osc9_term51ExpectNotification(t, "9;9", "9")
	oscB_osc9_term51ExpectNotification(t, "9;9x", "9x")
}

// Zig: test "term51: parse: change tab title"
func TestOSCParsersOsc9_term51_parse_change_tab_title(t *testing.T) {
	{
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, "9;3;")
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuChangeTabTitle, "cmd == .conemu_change_tab_title")
		expect(t, cmd.ConemuChangeTabTitle.Tag == OSCCommandConemuChangeTabTitlePayloadReset, "cmd.conemu_change_tab_title == .reset")
	}
	for _, c := range []struct {
		input string
		want  string
	}{
		{"9;3;t", "t"},
		{"9;3; ", " "},
		{"9;3;;", ";"},
		{"9;3;a;b\x01\xff", "a;b\x01\xff"},
	} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, c.input)
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuChangeTabTitle, "cmd == .conemu_change_tab_title")
		expect(t, cmd.ConemuChangeTabTitle.Tag == OSCCommandConemuChangeTabTitlePayloadValue, "cmd.conemu_change_tab_title == .value")
		expectEqualSlices(t, []uint8(c.want), cmd.ConemuChangeTabTitle.Value)
	}

	oscB_osc9_term51ExpectNotification(t, "9;3", "3")
	oscB_osc9_term51ExpectNotification(t, "9;3x", "3x")
}

// Zig: test "term51: parse: progress state"
func TestOSCParsersOsc9_term51_parse_progress_state(t *testing.T) {
	for _, c := range []struct {
		input    string
		state    OSCCommandProgressReportState
		progress Optional[uint8]
	}{
		{"9;4;0", OSCCommandProgressReportStateRemove, None[uint8]()},
		// "set" starts at 0, the others have no progress
		{"9;4;1", OSCCommandProgressReportStateSet, Some[uint8](0)},
		{"9;4;2", OSCCommandProgressReportStateError, None[uint8]()},
		{"9;4;3", OSCCommandProgressReportStateIndeterminate, None[uint8]()},
		{"9;4;4", OSCCommandProgressReportStatePause, None[uint8]()},
		// remove and indeterminate never take a progress value
		{"9;4;0;50", OSCCommandProgressReportStateRemove, None[uint8]()},
		{"9;4;3;50", OSCCommandProgressReportStateIndeterminate, None[uint8]()},
		{"9;4;1;50", OSCCommandProgressReportStateSet, Some[uint8](50)},
		{"9;4;2;50", OSCCommandProgressReportStateError, Some[uint8](50)},
		{"9;4;4;50", OSCCommandProgressReportStatePause, Some[uint8](50)},
		// anything other than ';' after the state is ignored
		{"9;4;1x", OSCCommandProgressReportStateSet, Some[uint8](0)},
		{"9;4;12", OSCCommandProgressReportStateSet, Some[uint8](0)},
		{"9;4;2x50", OSCCommandProgressReportStateError, None[uint8]()},
		{"9;4;0x", OSCCommandProgressReportStateRemove, None[uint8]()},
	} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, c.input)
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
		expect(t, cmd.ConemuProgressReport.state == c.state, "cmd.conemu_progress_report.state == case[1]")
		expectEqual(t, c.progress, cmd.ConemuProgressReport.progress)
	}

	oscB_osc9_term51ExpectNotification(t, "9;4", "4")
	oscB_osc9_term51ExpectNotification(t, "9;4;", "4;")
	oscB_osc9_term51ExpectNotification(t, "9;4x", "4x")
	oscB_osc9_term51ExpectNotification(t, "9;4;5", "4;5")
	oscB_osc9_term51ExpectNotification(t, "9;4;9;50", "4;9;50")
	oscB_osc9_term51ExpectNotification(t, "9;4; 1", "4; 1")
	oscB_osc9_term51ExpectNotification(t, "9;40", "40")
}

// Zig: test "term51: parse: progress value"
func TestOSCParsersOsc9_term51_parse_progress_value(t *testing.T) {
	for _, c := range []struct {
		input    string
		progress Optional[uint8]
	}{
		{"9;4;1;0", Some[uint8](0)},
		{"9;4;1;1", Some[uint8](1)},
		{"9;4;1;99", Some[uint8](99)},
		{"9;4;1;100", Some[uint8](100)},
		// clamped to 100
		{"9;4;1;101", Some[uint8](100)},
		{"9;4;1;255", Some[uint8](100)},
		{"9;4;1;256", Some[uint8](100)},
		{"9;4;1;18446744073709551615", Some[uint8](100)},
		// std.fmt.parseUnsigned accepts leading zeros and '_' between
		// digits
		{"9;4;1;007", Some[uint8](7)},
		{"9;4;1;5_0", Some[uint8](50)},
		// A value that fails to parse gives null, which also replaces the
		// default 0 of the "set" state.
		{"9;4;1;", None[uint8]()},
		{"9;4;1;abc", None[uint8]()},
		{"9;4;1;-1", None[uint8]()},
		{"9;4;1;+1", None[uint8]()},
		{"9;4;1; 1", None[uint8]()},
		{"9;4;1;1 ", None[uint8]()},
		{"9;4;1;50;", None[uint8]()},
		{"9;4;1;50;60", None[uint8]()},
		{"9;4;1;5.5", None[uint8]()},
		// one past the largest usize is a parse error, not clamped
		{"9;4;1;18446744073709551616", None[uint8]()},
	} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, c.input)
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
		expect(t, cmd.ConemuProgressReport.state == OSCCommandProgressReportStateSet, "cmd.conemu_progress_report.state == .set")
		expectEqual(t, c.progress, cmd.ConemuProgressReport.progress)
	}
}

// Zig: test "term51: parse: 5 followed by anything is wait input"
func TestOSCParsersOsc9_term51_parse_5_followed_by_anything_is_wait_input(t *testing.T) {
	// Nothing after the "5" is checked, so notifications that happen to
	// start with "5" are swallowed.
	for _, input := range []string{"9;5", "9;5;", "9;5;x", "9;5x", "9;50", "9;5 minutes left"} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, input)
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuWaitInput, "cmd == .conemu_wait_input")
	}
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersOsc9_term51_parse_buffer_limit(t *testing.T) {
	// Commands that return a string need one byte for the NUL terminator.
	oscB_osc9_term51ExpectNotification(t, "9;"+strings.Repeat("a", OSCParserMAX_BUF-1), strings.Repeat("a", OSCParserMAX_BUF-1))
	{
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, "9;"+strings.Repeat("a", OSCParserMAX_BUF))
		expect(t, p.state == OSCParserState9, "p.state == .@\"9\"")
		expect(t, OSCParsersOsc9Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, "9;2;"+strings.Repeat("a", OSCParserMAX_BUF-3))
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuShowMessageBox, "cmd == .conemu_show_message_box")
		expectEqual(t, OSCParserMAX_BUF-3, len(cmd.ConemuShowMessageBox))
	}
	for _, prefix := range []string{"2;", "3;", "6;", "7;", "8;", "9;", "11;"} {
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, "9;"+prefix+strings.Repeat("a", OSCParserMAX_BUF-len(prefix)))
		expect(t, p.state == OSCParserState9, "p.state == .@\"9\"")
		expect(t, OSCParsersOsc9Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		// An allocator does not change the limit.
		p := OSCParserInit(oscB_osc9_allocator) // testing.allocator
		defer p.Deinit()
		oscB_osc9_term51Feed(&p, "9;"+strings.Repeat("a", OSCParserMAX_BUF))
		expect(t, OSCParsersOsc9Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	}

	// Commands that do not return a string can use the whole buffer.
	{
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, "9;5"+strings.Repeat("a", OSCParserMAX_BUF-1))
		expect(t, p.state == OSCParserState9, "p.state == .@\"9\"")
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuWaitInput, "cmd == .conemu_wait_input")
	}
	{
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, "9;4;1;"+strings.Repeat("0", OSCParserMAX_BUF-6)+"42")
		expect(t, p.state == OSCParserState9, "p.state == .@\"9\"")
		cmd := oscB_osc9_parse(t, &p)
		expect(t, cmd.Tag == OSCCommandKeyConemuProgressReport, "cmd == .conemu_progress_report")
		expectEqual(t, Some[uint8](42), cmd.ConemuProgressReport.progress)
	}
	{
		// One byte more overflows while feeding.
		p := OSCParserInit(nil)
		oscB_osc9_term51Feed(&p, "9;5"+strings.Repeat("a", OSCParserMAX_BUF))
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}
}
