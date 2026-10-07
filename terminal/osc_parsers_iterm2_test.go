// Tests translated one for one from src/terminal/osc/parsers/iterm2.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers (expect,
// expectEqual, ...) are in Parser_test.go; oscA_term51Feed, oscA_unwrapPtr,
// oscA_ptrAt, oscA_terms and oscA_testingAllocator are in
// osc_parsers_change_window_icon_test.go, where the translation notes are.
//
// `cmd.report_pwd.value` is cmd.ReportPwd.value and
// `cmd.clipboard_contents.kind/.data` are cmd.ClipboardContents.kind/.data.
// The test "OSC: 1337: test Copy with non-empty value that is invalid base64"
// returns error.SkipZigTest upstream; its Go counterpart calls t.Skip.

package terminal

import (
	"strings"
	"testing"
)

// Zig: test "OSC: 1337: test valid unimplemented key with no value"
func TestOSCParsersIterm2_OSC_1337_test_valid_unimplemented_key_with_no_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;SetBadgeFormat"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test valid unimplemented key with empty value"
func TestOSCParsersIterm2_OSC_1337_test_valid_unimplemented_key_with_empty_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;SetBadgeFormat="
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test valid unimplemented key with non-empty value"
func TestOSCParsersIterm2_OSC_1337_test_valid_unimplemented_key_with_non_empty_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;SetBadgeFormat=abc123"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test valid key with lower case and with no value"
func TestOSCParsersIterm2_OSC_1337_test_valid_key_with_lower_case_and_with_no_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;setbadgeformat"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test valid key with lower case and with empty value"
func TestOSCParsersIterm2_OSC_1337_test_valid_key_with_lower_case_and_with_empty_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;setbadgeformat="
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test valid key with lower case and with non-empty value"
func TestOSCParsersIterm2_OSC_1337_test_valid_key_with_lower_case_and_with_non_empty_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;setbadgeformat=abc123"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test invalid key with no value"
func TestOSCParsersIterm2_OSC_1337_test_invalid_key_with_no_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;BobrKurwa"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test invalid key with empty value"
func TestOSCParsersIterm2_OSC_1337_test_invalid_key_with_empty_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;BobrKurwa="
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test invalid key with non-empty value"
func TestOSCParsersIterm2_OSC_1337_test_invalid_key_with_non_empty_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;BobrKurwa=abc123"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test Copy with no value"
func TestOSCParsersIterm2_OSC_1337_test_Copy_with_no_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;Copy"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test Copy with empty value"
func TestOSCParsersIterm2_OSC_1337_test_Copy_with_empty_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;Copy="
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test Copy with only prefix colon"
func TestOSCParsersIterm2_OSC_1337_test_Copy_with_only_prefix_colon(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;Copy=:"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test Copy with question mark"
func TestOSCParsersIterm2_OSC_1337_test_Copy_with_question_mark(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;Copy=:?"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test Copy with non-empty value that is invalid base64"
func TestOSCParsersIterm2_OSC_1337_test_Copy_with_non_empty_value_that_is_invalid_base64(t *testing.T) {
	// For performance reasons, we don't check for valid base64 data
	// right now.
	t.Skip("error.SkipZigTest")

	// Zig: the rest of the test is commented out upstream.
}

// Zig: test "OSC: 1337: test Copy with non-empty value that is valid base64 but not prefixed with a colon"
func TestOSCParsersIterm2_OSC_1337_test_Copy_with_non_empty_value_that_is_valid_base64_but_not_prefixed_with_a_colon(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;Copy=YWJjMTIz"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test Copy with non-empty value that is valid base64"
func TestOSCParsersIterm2_OSC_1337_test_Copy_with_non_empty_value_that_is_valid_base64(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;Copy=:YWJjMTIz"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
	expectEqual(t, uint8('c'), cmd.ClipboardContents.Kind)
	expectEqualStrings(t, "YWJjMTIz", string(cmd.ClipboardContents.Data))
}

// Zig: test "OSC: 1337: test CurrentDir with no value"
func TestOSCParsersIterm2_OSC_1337_test_CurrentDir_with_no_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;CurrentDir"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test CurrentDir with empty value"
func TestOSCParsersIterm2_OSC_1337_test_CurrentDir_with_empty_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;CurrentDir="
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC: 1337: test CurrentDir with non-empty value"
func TestOSCParsersIterm2_OSC_1337_test_CurrentDir_with_non_empty_value(t *testing.T) {
	p := OSCParserInit(oscA_testingAllocator)
	defer p.Deinit()

	input := "1337;CurrentDir=abc123"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
	expectEqualStrings(t, "abc123", string(cmd.ReportPwd.Value))
}

// Zig: test "term51: parse: missing separator after 1337 is invalid"
func TestOSCParsersIterm2_term51_parse_missing_separator_after_1337_is_invalid(t *testing.T) {
	p := OSCParserInit(nil)
	oscA_term51Feed(&p, "1337")
	expect(t, p.state == OSCParserState1337, "p.state == .@\"1337\"")
	expect(t, OSCParsersIterm2Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
}

// Zig: test "term51: parse: empty and unknown keys are rejected without invalidating the state"
func TestOSCParsersIterm2_term51_parse_empty_and_unknown_keys_are_rejected_without_invalidating_the_state(t *testing.T) {
	for _, input := range []string{
		"1337;",
		"1337;=",
		"1337;=value",
		"1337;Bogus",
		"1337;Bogus=1",
		// keys are matched exactly, not trimmed and not by prefix
		"1337; Copy=:YQ==",
		"1337;Copy =:YQ==",
		"1337;Cop=:YQ==",
		"1337;Copyy=:YQ==",
		"1337;CurrentDirectory=/tmp",
	} {
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, input)
		expect(t, OSCParsersIterm2Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
		// Unlike the other parsers, the state is left alone.
		expect(t, p.state == OSCParserState1337, "p.state == .@\"1337\"")
	}
}

// Zig: test "term51: parse: every unimplemented key is rejected"
func TestOSCParsersIterm2_term51_parse_every_unimplemented_key_is_rejected(t *testing.T) {
	// Zig: inline for over the fields of Key; the Go loop goes over the
	// values of oscParsersIterm2Key.
	for k := oscParsersIterm2Key(0); k < oscParsersIterm2KeyCount; k++ {
		if k != oscParsersIterm2KeyCopy && k != oscParsersIterm2KeyCurrentDir {
			for _, suffix := range []string{"", "=", "=value"} {
				p := OSCParserInit(nil)
				oscA_term51Feed(&p, "1337;"+k.String()+suffix)
				expect(t, OSCParsersIterm2Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
				expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
				expect(t, p.state == OSCParserState1337, "p.state == .@\"1337\"")
			}
		}
	}
	// 36 keys are known in total.
	expectEqual(t, 36, int(oscParsersIterm2KeyCount))
}

// Zig: test "term51: parse: Copy"
func TestOSCParsersIterm2_term51_parse_Copy(t *testing.T) {
	for _, term := range oscA_terms {
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, "1337;Copy=:YWJj")
		cmd := *oscA_unwrapPtr(t, OSCParsersIterm2Parse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
		expectEqual(t, uint8('c'), cmd.ClipboardContents.Kind)
		expectEqualStrings(t, "YWJj", string(cmd.ClipboardContents.Data))
		expectEqual(t, uint8(0), oscA_ptrAt(cmd.ClipboardContents.Data, 4))
	}
}

// Zig: test "term51: parse: keys are ASCII case insensitive"
func TestOSCParsersIterm2_term51_parse_keys_are_ASCII_case_insensitive(t *testing.T) {
	for _, key := range []string{"Copy", "copy", "COPY", "cOPy"} {
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, "1337;"+key+"=:YQ==")
		cmd := *oscA_unwrapPtr(t, OSCParsersIterm2Parse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
		expectEqualStrings(t, "YQ==", string(cmd.ClipboardContents.Data))
	}
	for _, key := range []string{"CurrentDir", "currentdir", "CURRENTDIR"} {
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, "1337;"+key+"=/tmp")
		cmd := *oscA_unwrapPtr(t, OSCParsersIterm2Parse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
		expectEqualStrings(t, "/tmp", string(cmd.ReportPwd.Value))
	}
}

// Zig: test "term51: parse: Copy rejects missing, empty, unprefixed and query values"
func TestOSCParsersIterm2_term51_parse_Copy_rejects_missing_empty_unprefixed_and_query_values(t *testing.T) {
	for _, input := range []string{
		"1337;Copy",
		"1337;Copy=",
		"1337;Copy=:",
		"1337;Copy=:?",
		"1337;Copy=YQ==",
		"1337;Copy=?",
	} {
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, input)
		expect(t, OSCParsersIterm2Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
		expect(t, p.state == OSCParserState1337, "p.state == .@\"1337\"")
	}
}

// Zig: test "term51: parse: Copy value is not validated beyond the colon prefix"
func TestOSCParsersIterm2_term51_parse_Copy_value_is_not_validated_beyond_the_colon_prefix(t *testing.T) {
	for _, c := range []struct{ input, want string }{
		// only a lone '?' is rejected
		{"1337;Copy=:??", "??"},
		{"1337;Copy=:?x", "?x"},
		// the value is split at the first '=' only
		{"1337;Copy=:a=b=c", "a=b=c"},
		// only one ':' is removed
		{"1337;Copy=::", ":"},
		{"1337;Copy=:not base64!;\x01\xff", "not base64!;\x01\xff"},
	} {
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, c.input)
		cmd := *oscA_unwrapPtr(t, OSCParsersIterm2Parse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
		expectEqual(t, uint8('c'), cmd.ClipboardContents.Kind)
		expectEqualSlices(t, []uint8(c.want), cmd.ClipboardContents.Data)
	}
}

// Zig: test "term51: parse: CurrentDir"
func TestOSCParsersIterm2_term51_parse_CurrentDir(t *testing.T) {
	for _, c := range []struct{ input, want string }{
		{"1337;CurrentDir=/home/user", "/home/user"},
		// the value is not validated and is split at the first '=' only
		{"1337;CurrentDir=a=b;c\x01\xff", "a=b;c\x01\xff"},
		{"1337;CurrentDir= ", " "},
	} {
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, c.input)
		cmd := *oscA_unwrapPtr(t, OSCParsersIterm2Parse(&p, Some[uint8](0x07)))
		expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
		expectEqualSlices(t, []uint8(c.want), cmd.ReportPwd.Value)
		expectEqual(t, uint8(0), oscA_ptrAt(cmd.ReportPwd.Value, len(cmd.ReportPwd.Value)))
	}
	for _, input := range []string{"1337;CurrentDir", "1337;CurrentDir="} {
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, input)
		expect(t, OSCParsersIterm2Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
		expect(t, p.state == OSCParserState1337, "p.state == .@\"1337\"")
	}
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersIterm2_term51_parse_buffer_limit(t *testing.T) {
	// "CurrentDir=" plus the value plus the NUL terminator must fit in
	// MAX_BUF.
	const prefix = "CurrentDir="
	{
		value := strings.Repeat("d", OSCParserMAX_BUF-1-len(prefix))
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, "1337;"+prefix+value)
		cmd := *oscA_unwrapPtr(t, OSCParsersIterm2Parse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyReportPwd, "cmd == .report_pwd")
		expectEqualStrings(t, value, string(cmd.ReportPwd.Value))
	}
	{
		value := strings.Repeat("d", OSCParserMAX_BUF-len(prefix))
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, "1337;"+prefix+value)
		expect(t, p.state == OSCParserState1337, "p.state == .@\"1337\"")
		expect(t, OSCParsersIterm2Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		// Failing to write the NUL terminator does invalidate the state.
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		// An allocator does not change the limit.
		value := strings.Repeat("d", OSCParserMAX_BUF-len(prefix))
		p := OSCParserInit(oscA_testingAllocator)
		defer p.Deinit()
		oscA_term51Feed(&p, "1337;"+prefix+value)
		expect(t, OSCParsersIterm2Parse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	}
}
