// Tests translated one for one from src/terminal/osc/parsers/report_pwd.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers expect,
// expectEqual, expectEqualSlices, expectEqualStrings and zigUnwrap are
// defined in Parser_test.go.
//
// Translation notes:
//   - `.init(null)` is OSCParserInit(nil); `p.next(ch)` is p.Next(ch);
//     `p.end(null)` is p.End(None[uint8]()).
//   - `parse(&p, null).?.*` is oscB_pwd_unwrap(t, OSCParsersReportPwdParse(...)),
//     which fails on nil and copies the command out.
//   - `inline for` over the optional terminators is a loop over a slice.
//   - `value.ptr[n]` (the sentinel byte after the slice) is oscB_pwd_at,
//     which reads one byte past the length of the slice (up to its
//     capacity).
//   - `p.writer == null` is `p.writer == nil`.

package terminal

import (
	"strings"
	"testing"
)

// oscB_pwd_unwrap is Zig's `ptr.?.*` on the result of parse or end.
func oscB_pwd_unwrap(t *testing.T, c *OSCCommand) OSCCommand {
	t.Helper()
	if c == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return *c
}

// oscB_pwd_term51Feed is Zig's term51Feed.
func oscB_pwd_term51Feed(p *OSCParser, input []uint8) {
	for _, ch := range input {
		p.Next(ch)
	}
}

// oscB_pwd_at is Zig's `s.ptr[i]`: the byte at index i of the memory behind
// s, which may lie past its length (the sentinel).
func oscB_pwd_at(s []uint8, i int) uint8 {
	return s[:i+1][i]
}

// Zig: test "OSC 7: report pwd"
func TestOSCParsersReportPwd_OSC_7_report_pwd(t *testing.T) {
	p := OSCParserInit(nil)

	input := "7;file:///tmp/example"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_pwd_unwrap(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
	expectEqualStrings(t, "file:///tmp/example", string(cmd.ReportPwd.value))
}

// Zig: test "OSC 7: report pwd empty"
func TestOSCParsersReportPwd_OSC_7_report_pwd_empty(t *testing.T) {
	p := OSCParserInit(nil)

	input := "7;"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_pwd_unwrap(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
	expectEqualStrings(t, "", string(cmd.ReportPwd.value))
}

// Zig: test "term51: parse: basic url"
func TestOSCParsersReportPwd_term51_parse_basic_url(t *testing.T) {
	for _, term := range []Optional[uint8]{None[uint8](), Some[uint8](0x07), Some[uint8](0x1b)} {
		p := OSCParserInit(nil)
		oscB_pwd_term51Feed(&p, []uint8("7;file://host/home/user"))
		cmd := oscB_pwd_unwrap(t, OSCParsersReportPwdParse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
		expectEqualStrings(t, "file://host/home/user", string(cmd.ReportPwd.value))
		expectEqual(t, uint8(0), oscB_pwd_at(cmd.ReportPwd.value, len(cmd.ReportPwd.value)))
	}
}

// Zig: test "term51: parse: missing separator is invalid"
func TestOSCParsersReportPwd_term51_parse_missing_separator_is_invalid(t *testing.T) {
	p := OSCParserInit(nil)
	oscB_pwd_term51Feed(&p, []uint8("7"))
	expect(t, p.writer == nil, "p.writer == null")
	expect(t, OSCParsersReportPwdParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
}

// Zig: test "term51: parse: empty value"
func TestOSCParsersReportPwd_term51_parse_empty_value(t *testing.T) {
	p := OSCParserInit(nil)
	oscB_pwd_term51Feed(&p, []uint8("7;"))
	cmd := oscB_pwd_unwrap(t, OSCParsersReportPwdParse(&p, Some[uint8](0x07)))
	expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
	expectEqual(t, 0, len(cmd.ReportPwd.value))
}

// Zig: test "term51: parse: the value is not validated"
func TestOSCParsersReportPwd_term51_parse_the_value_is_not_validated(t *testing.T) {
	// Not a URL, contains semicolons, control bytes and invalid UTF-8.
	p := OSCParserInit(nil)
	oscB_pwd_term51Feed(&p, []uint8("7;not a url;x\x01\xff"))
	cmd := oscB_pwd_unwrap(t, OSCParsersReportPwdParse(&p, None[uint8]()))
	expectEqualSlices(t, []uint8("not a url;x\x01\xff"), cmd.ReportPwd.value)
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersReportPwd_term51_parse_buffer_limit(t *testing.T) {
	{
		p := OSCParserInit(nil)
		oscB_pwd_term51Feed(&p, []uint8("7;"+strings.Repeat("a", OSCParserMAX_BUF-1)))
		cmd := oscB_pwd_unwrap(t, OSCParsersReportPwdParse(&p, None[uint8]()))
		expectEqual(t, OSCParserMAX_BUF-1, len(cmd.ReportPwd.value))
	}
	{
		p := OSCParserInit(nil)
		oscB_pwd_term51Feed(&p, []uint8("7;"+strings.Repeat("a", OSCParserMAX_BUF)))
		expect(t, p.state == OSCParserState7, "p.state == .@\"7\"")
		expect(t, OSCParsersReportPwdParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}
