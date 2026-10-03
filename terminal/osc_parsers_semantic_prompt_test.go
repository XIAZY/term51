// Tests translated one for one from src/terminal/osc/parsers/semantic_prompt.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualSlices, expectEqualStrings, zigUnwrap and zigTry are defined in
// Parser_test.go and dcs_test.go; the helpers declared here start with
// oscB_sp_.
//
// Translation notes:
//   - `var p: Parser = .init(null)` is OSCParserInit(nil), and
//     `.init(testing.allocator)` is OSCParserInit(oscB_sp_allocator), a non-nil
//     value (the parser only records whether an allocator was given).
//   - `p.end(null).?.*` is oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]())): the
//     optional pointer is a *OSCCommand, nil is null.
//   - `cmd == .semantic_prompt` is `cmd.Tag == OSCCommandKeySemanticPrompt`,
//     `cmd.semantic_prompt` is cmd.SemanticPrompt, and a comptime option
//     argument is a value of OSCParsersSemanticPromptOption whose result is
//     read from the field of OSCParsersSemanticPromptOptionValue named after it
//     (`readOption(.aid)` is `ReadOption(...OptionAid).Aid`).
//   - `std.Io.Writer.Allocating` is the zero value of oscWriterAllocating and
//     `w.written()` is oscB_sp_written(&w), buffer[:end] of its writer.
//     `defer w.deinit()` has no counterpart and is kept as a Zig-only line.
//   - `try x` on an error union is zigTry on the error, and
//     `expectError(error.DecodeError, x)` compares the error with
//     ErrDecodeError.
//   - Zig's `defer` inside a loop body runs at the end of each iteration; Go's
//     defer runs when the test function returns.

package terminal

import (
	"strings"
	"testing"
)

// oscB_sp_allocator stands in for `testing.allocator`: any non-nil value.
var oscB_sp_allocator any = "testing.allocator"

// oscB_sp_zigUnwrapPtr is Zig's `p.?.*`: the value an optional pointer points
// at, failing if it is null.
func oscB_sp_zigUnwrapPtr[T any](t *testing.T, p *T) T {
	t.Helper()
	if p == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return *p
}

// oscB_sp_written is Zig's `w.written()` for a std.Io.Writer.Allocating: the
// bytes written so far.
func oscB_sp_written(w *oscWriterAllocating) string {
	return string(w.writer.buffer[:w.writer.end])
}

// Zig: test "OSC 133: end_input_start_output"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;C"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid.Valid, "cmd.semantic_prompt.readOption(.aid) == null")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl.Valid, "cmd.semantic_prompt.readOption(.cl) == null")
}

// Zig: test "OSC 133: end_input_start_output extra contents"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_extra_contents(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;Cextra"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 133: end_input_start_output with options"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_options(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;C;aid=foo"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")
	expectEqualStrings(t, "foo", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
}

// Zig: test "OSC 133: end_input_start_output with cmdline"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline=echo bobr kurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	zigTry(t, cmd.SemanticPrompt.WriteCommandLine(&w.writer))
	expectEqualStrings(t, "echo bobr kurwa", oscB_sp_written(&w))
}

// Zig: test "OSC 133: end_input_start_output with cmdline 3"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_3(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline=echo bobr\\nkurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")

	zigTry(t, cmd.SemanticPrompt.WriteCommandLine(&w.writer))
	expectEqualStrings(t, "echo bobr\nkurwa", oscB_sp_written(&w))
}

// Zig: test "OSC 133: end_input_start_output with cmdline 4"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_4(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline=$'echo bobr kurwa'"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	zigTry(t, cmd.SemanticPrompt.WriteCommandLine(&w.writer))
	expectEqualStrings(t, "echo bobr kurwa", oscB_sp_written(&w))
}

// Zig: test "OSC 133: end_input_start_output with cmdline 5"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_5(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline='echo bobr kurwa'"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	zigTry(t, cmd.SemanticPrompt.WriteCommandLine(&w.writer))
	expectEqualStrings(t, "echo bobr kurwa", oscB_sp_written(&w))
}

// Zig: test "OSC 133: end_input_start_output with cmdline 6"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_6(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline='echo bobr kurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	expectEqual(t, error(ErrDecodeError), cmd.SemanticPrompt.WriteCommandLine(&w.writer))
}

// Zig: test "OSC 133: end_input_start_output with cmdline 7"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_7(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline=$'echo bobr kurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	expectEqual(t, error(ErrDecodeError), cmd.SemanticPrompt.WriteCommandLine(&w.writer))
}

// Zig: test "OSC 133: end_input_start_output with cmdline 8"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_8(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline=$'"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	expectEqual(t, error(ErrDecodeError), cmd.SemanticPrompt.WriteCommandLine(&w.writer))
}

// Zig: test "OSC 133: end_input_start_output with cmdline 9"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_9(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline="
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	zigTry(t, cmd.SemanticPrompt.WriteCommandLine(&w.writer))
	expectEqualStrings(t, "", oscB_sp_written(&w))
}

// Zig: test "OSC 133: end_input_start_output with cmdline_url 1"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_url_1(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline_url=echo bobr kurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	zigTry(t, cmd.SemanticPrompt.WriteCommandLine(&w.writer))
	expectEqualStrings(t, "echo bobr kurwa", oscB_sp_written(&w))
}

// Zig: test "OSC 133: end_input_start_output with cmdline_url 2"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_url_2(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline_url=echo bobr%20kurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	zigTry(t, cmd.SemanticPrompt.WriteCommandLine(&w.writer))
	expectEqualStrings(t, "echo bobr kurwa", oscB_sp_written(&w))
}

// Zig: test "OSC 133: end_input_start_output with cmdline_url 3"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_url_3(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline_url=echo bobr%3bkurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	zigTry(t, cmd.SemanticPrompt.WriteCommandLine(&w.writer))
	expectEqualStrings(t, "echo bobr;kurwa", oscB_sp_written(&w))
}

// Zig: test "OSC 133: end_input_start_output with cmdline_url 4"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_url_4(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline_url=echo bobr%3kurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	expectEqual(t, error(ErrDecodeError), cmd.SemanticPrompt.WriteCommandLine(&w.writer))
}

// Zig: test "OSC 133: end_input_start_output with cmdline_url 5"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_url_5(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline_url=echo bobr%kurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	expectEqual(t, error(ErrDecodeError), cmd.SemanticPrompt.WriteCommandLine(&w.writer))
}

// Zig: test "OSC 133: end_input_start_output with cmdline_url 6"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_url_6(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline_url=echo bobr kurwa%20"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	zigTry(t, cmd.SemanticPrompt.WriteCommandLine(&w.writer))
	expectEqualStrings(t, "echo bobr kurwa ", oscB_sp_written(&w))
}

// Zig: test "OSC 133: end_input_start_output with cmdline_url 7"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_url_7(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline_url=echo bobr kurwa%2"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	expectEqual(t, error(ErrDecodeError), cmd.SemanticPrompt.WriteCommandLine(&w.writer))
}

// Zig: test "OSC 133: end_input_start_output with cmdline_url 8"
func TestOSCParsersSemanticPrompt_OSC_133_end_input_start_output_with_cmdline_url_8(t *testing.T) {
	var w oscWriterAllocating
	// Zig-only: defer w.deinit();

	p := OSCParserInit(nil)
	input := "133;C;cmdline_url=echo bobr kurwa%"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndInputStartOutput, "cmd.semantic_prompt.action == .end_input_start_output")

	expectEqual(t, error(ErrDecodeError), cmd.SemanticPrompt.WriteCommandLine(&w.writer))
}

// Zig: test "OSC 133: fresh_line"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;L"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLine, "cmd.semantic_prompt.action == .fresh_line")
}

// Zig: test "OSC 133: fresh_line extra contents"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_extra_contents(t *testing.T) {
	// Random
	{
		p := OSCParserInit(nil)
		input := "133;Lol"
		for _, ch := range []uint8(input) {
			p.Next(ch)
		}
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}

	// Options
	{
		p := OSCParserInit(nil)
		input := "133;L;aid=foo"
		for _, ch := range []uint8(input) {
			p.Next(ch)
		}
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}
}

// Zig: test "OSC 133: fresh_line_new_prompt"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid.Valid, "cmd.semantic_prompt.readOption(.aid) == null")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl.Valid, "cmd.semantic_prompt.readOption(.cl) == null")
}

// Zig: test "OSC 133: fresh_line_new_prompt with aid"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_aid(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;aid=14"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expectEqualStrings(t, "14", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
}

// Zig: test "OSC 133: fresh_line_new_prompt with '=' in aid"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_in_aid(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;aid=a=b"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expectEqualStrings(t, "a=b", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
}

// Zig: test "OSC 133: fresh_line_new_prompt with cl=line"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_cl_line(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;cl=line"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expect(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl == Some(OSCParsersSemanticPromptClickLine), "cmd.semantic_prompt.readOption(.cl) == .line")
}

// Zig: test "OSC 133: fresh_line_new_prompt with cl=m"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_cl_m(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;cl=m"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expect(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl == Some(OSCParsersSemanticPromptClickMultiple), "cmd.semantic_prompt.readOption(.cl) == .multiple")
}

// Zig: test "OSC 133: fresh_line_new_prompt with invalid cl"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_invalid_cl(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;cl=invalid"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl.Valid, "cmd.semantic_prompt.readOption(.cl) == null")
}

// Zig: test "OSC 133: fresh_line_new_prompt with trailing ;"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_trailing(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
}

// Zig: test "OSC 133: fresh_line_new_prompt with bare key"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_bare_key(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;barekey"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid.Valid, "cmd.semantic_prompt.readOption(.aid) == null")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl.Valid, "cmd.semantic_prompt.readOption(.cl) == null")
}

// Zig: test "OSC 133: fresh_line_new_prompt with multiple options"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_multiple_options(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;aid=foo;cl=line"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expectEqualStrings(t, "foo", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
	expect(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl == Some(OSCParsersSemanticPromptClickLine), "cmd.semantic_prompt.readOption(.cl) == .line")
}

// Zig: test "OSC 133: fresh_line_new_prompt default redraw"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_default_redraw(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionRedraw).Redraw.Valid, "cmd.semantic_prompt.readOption(.redraw) == null")
}

// Zig: test "OSC 133: fresh_line_new_prompt with redraw=0"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_redraw_0(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;redraw=0"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expect(t, zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionRedraw).Redraw) == OSCParsersSemanticPromptRedrawFalse, "cmd.semantic_prompt.readOption(.redraw).? == .false")
}

// Zig: test "OSC 133: fresh_line_new_prompt with redraw=1"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_redraw_1(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;redraw=1"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expect(t, zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionRedraw).Redraw) == OSCParsersSemanticPromptRedrawTrue, "cmd.semantic_prompt.readOption(.redraw).? == .true")
}

// Zig: test "OSC 133: fresh_line_new_prompt with invalid redraw"
func TestOSCParsersSemanticPrompt_OSC_133_fresh_line_new_prompt_with_invalid_redraw(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;A;redraw=x"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cmd.semantic_prompt.action == .fresh_line_new_prompt")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionRedraw).Redraw.Valid, "cmd.semantic_prompt.readOption(.redraw) == null")
}

// Zig: test "OSC 133: prompt_start"
func TestOSCParsersSemanticPrompt_OSC_133_prompt_start(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;P"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionPromptStart, "cmd.semantic_prompt.action == .prompt_start")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionPromptKind).PromptKind.Valid, "cmd.semantic_prompt.readOption(.prompt_kind) == null")
}

// Zig: test "OSC 133: prompt_start with k=i"
func TestOSCParsersSemanticPrompt_OSC_133_prompt_start_with_k_i(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;P;k=i"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionPromptStart, "cmd.semantic_prompt.action == .prompt_start")
	expect(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionPromptKind).PromptKind == Some(OSCParsersSemanticPromptPromptKindInitial), "cmd.semantic_prompt.readOption(.prompt_kind) == .initial")
}

// Zig: test "OSC 133: prompt_start with k=r"
func TestOSCParsersSemanticPrompt_OSC_133_prompt_start_with_k_r(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;P;k=r"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionPromptStart, "cmd.semantic_prompt.action == .prompt_start")
	expect(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionPromptKind).PromptKind == Some(OSCParsersSemanticPromptPromptKindRight), "cmd.semantic_prompt.readOption(.prompt_kind) == .right")
}

// Zig: test "OSC 133: prompt_start with k=c"
func TestOSCParsersSemanticPrompt_OSC_133_prompt_start_with_k_c(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;P;k=c"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionPromptStart, "cmd.semantic_prompt.action == .prompt_start")
	expect(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionPromptKind).PromptKind == Some(OSCParsersSemanticPromptPromptKindContinuation), "cmd.semantic_prompt.readOption(.prompt_kind) == .continuation")
}

// Zig: test "OSC 133: prompt_start with k=s"
func TestOSCParsersSemanticPrompt_OSC_133_prompt_start_with_k_s(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;P;k=s"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionPromptStart, "cmd.semantic_prompt.action == .prompt_start")
	expect(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionPromptKind).PromptKind == Some(OSCParsersSemanticPromptPromptKindSecondary), "cmd.semantic_prompt.readOption(.prompt_kind) == .secondary")
}

// Zig: test "OSC 133: prompt_start with invalid k"
func TestOSCParsersSemanticPrompt_OSC_133_prompt_start_with_invalid_k(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;P;k=x"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionPromptStart, "cmd.semantic_prompt.action == .prompt_start")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionPromptKind).PromptKind.Valid, "cmd.semantic_prompt.readOption(.prompt_kind) == null")
}

// Zig: test "OSC 133: prompt_start extra contents"
func TestOSCParsersSemanticPrompt_OSC_133_prompt_start_extra_contents(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;Pextra"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 133: new_command"
func TestOSCParsersSemanticPrompt_OSC_133_new_command(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;N"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionNewCommand, "cmd.semantic_prompt.action == .new_command")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid.Valid, "cmd.semantic_prompt.readOption(.aid) == null")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl.Valid, "cmd.semantic_prompt.readOption(.cl) == null")
}

// Zig: test "OSC 133: new_command with aid"
func TestOSCParsersSemanticPrompt_OSC_133_new_command_with_aid(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;N;aid=foo"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionNewCommand, "cmd.semantic_prompt.action == .new_command")
	expectEqualStrings(t, "foo", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
}

// Zig: test "OSC 133: new_command with cl=line"
func TestOSCParsersSemanticPrompt_OSC_133_new_command_with_cl_line(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;N;cl=line"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionNewCommand, "cmd.semantic_prompt.action == .new_command")
	expect(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl == Some(OSCParsersSemanticPromptClickLine), "cmd.semantic_prompt.readOption(.cl) == .line")
}

// Zig: test "OSC 133: new_command with multiple options"
func TestOSCParsersSemanticPrompt_OSC_133_new_command_with_multiple_options(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;N;aid=foo;cl=line"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionNewCommand, "cmd.semantic_prompt.action == .new_command")
	expectEqualStrings(t, "foo", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
	expect(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionCl).Cl == Some(OSCParsersSemanticPromptClickLine), "cmd.semantic_prompt.readOption(.cl) == .line")
}

// Zig: test "OSC 133: new_command extra contents"
func TestOSCParsersSemanticPrompt_OSC_133_new_command_extra_contents(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;Nextra"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 133: end_prompt_start_input"
func TestOSCParsersSemanticPrompt_OSC_133_end_prompt_start_input(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;B"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndPromptStartInput, "cmd.semantic_prompt.action == .end_prompt_start_input")
}

// Zig: test "OSC 133: end_prompt_start_input extra contents"
func TestOSCParsersSemanticPrompt_OSC_133_end_prompt_start_input_extra_contents(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;Bextra"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 133: end_prompt_start_input with options"
func TestOSCParsersSemanticPrompt_OSC_133_end_prompt_start_input_with_options(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;B;aid=foo"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndPromptStartInput, "cmd.semantic_prompt.action == .end_prompt_start_input")
	expectEqualStrings(t, "foo", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
}

// Zig: test "OSC 133: end_prompt_start_input_terminate_eol"
func TestOSCParsersSemanticPrompt_OSC_133_end_prompt_start_input_terminate_eol(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;I"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol, "cmd.semantic_prompt.action == .end_prompt_start_input_terminate_eol")
}

// Zig: test "OSC 133: end_prompt_start_input_terminate_eol extra contents"
func TestOSCParsersSemanticPrompt_OSC_133_end_prompt_start_input_terminate_eol_extra_contents(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;Iextra"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 133: end_prompt_start_input_terminate_eol with options"
func TestOSCParsersSemanticPrompt_OSC_133_end_prompt_start_input_terminate_eol_with_options(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;I;aid=foo"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol, "cmd.semantic_prompt.action == .end_prompt_start_input_terminate_eol")
	expectEqualStrings(t, "foo", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
}

// Zig: test "OSC 133: end_command"
func TestOSCParsersSemanticPrompt_OSC_133_end_command(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;D"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndCommand, "cmd.semantic_prompt.action == .end_command")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionExitCode).ExitCode.Valid, "cmd.semantic_prompt.readOption(.exit_code) == null")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid.Valid, "cmd.semantic_prompt.readOption(.aid) == null")
	expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionErr).Err.Valid, "cmd.semantic_prompt.readOption(.err) == null")
}

// Zig: test "OSC 133: end_command extra contents"
func TestOSCParsersSemanticPrompt_OSC_133_end_command_extra_contents(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;Dextra"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 133: end_command with exit code 0"
func TestOSCParsersSemanticPrompt_OSC_133_end_command_with_exit_code_0(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;D;0"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndCommand, "cmd.semantic_prompt.action == .end_command")
	expect(t, zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionExitCode).ExitCode) == 0, "cmd.semantic_prompt.readOption(.exit_code) == 0")
}

// Zig: test "OSC 133: end_command with exit code and aid"
func TestOSCParsersSemanticPrompt_OSC_133_end_command_with_exit_code_and_aid(t *testing.T) {
	p := OSCParserInit(nil)
	input := "133;D;12;aid=foo"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_sp_zigUnwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
	expect(t, cmd.SemanticPrompt.action == OSCParsersSemanticPromptCommandActionEndCommand, "cmd.semantic_prompt.action == .end_command")
	expectEqualStrings(t, "foo", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
	expect(t, zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionExitCode).ExitCode) == 12, "cmd.semantic_prompt.readOption(.exit_code) == 12")
}

// Zig: test "Option.read aid"
func TestOSCParsersSemanticPrompt_Option_read_aid(t *testing.T) {
	expectEqualStrings(t, "test123", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid=test123")).Aid)))
	expectEqualStrings(t, "myaid", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("cl=line;aid=myaid;k=i")).Aid)))
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("cl=line;k=i")).Aid.Valid, "Option.aid.read(\"cl=line;k=i\") == null")
	expectEqualStrings(t, "", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid=")).Aid)))
	expectEqualStrings(t, "last", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("k=i;aid=last")).Aid)))
	expectEqualStrings(t, "first", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid=first;k=i")).Aid)))
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("")).Aid.Valid, "Option.aid.read(\"\") == null")
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("aid")).Aid.Valid, "Option.aid.read(\"aid\") == null")
	expectEqualStrings(t, "value", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8(";;aid=value;;")).Aid)))
}

// Zig: test "Option.read cl"
func TestOSCParsersSemanticPrompt_Option_read_cl(t *testing.T) {
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=line")).Cl) == OSCParsersSemanticPromptClickLine, "Option.cl.read(\"cl=line\").? == .line")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=m")).Cl) == OSCParsersSemanticPromptClickMultiple, "Option.cl.read(\"cl=m\").? == .multiple")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=v")).Cl) == OSCParsersSemanticPromptClickConservativeVertical, "Option.cl.read(\"cl=v\").? == .conservative_vertical")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=w")).Cl) == OSCParsersSemanticPromptClickSmartVertical, "Option.cl.read(\"cl=w\").? == .smart_vertical")
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=invalid")).Cl.Valid, "Option.cl.read(\"cl=invalid\") == null")
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("aid=foo")).Cl.Valid, "Option.cl.read(\"aid=foo\") == null")
}

// Zig: test "Option.read prompt_kind"
func TestOSCParsersSemanticPrompt_Option_read_prompt_kind(t *testing.T) {
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=i")).PromptKind) == OSCParsersSemanticPromptPromptKindInitial, "Option.prompt_kind.read(\"k=i\").? == .initial")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=r")).PromptKind) == OSCParsersSemanticPromptPromptKindRight, "Option.prompt_kind.read(\"k=r\").? == .right")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=c")).PromptKind) == OSCParsersSemanticPromptPromptKindContinuation, "Option.prompt_kind.read(\"k=c\").? == .continuation")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=s")).PromptKind) == OSCParsersSemanticPromptPromptKindSecondary, "Option.prompt_kind.read(\"k=s\").? == .secondary")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=x")).PromptKind.Valid, "Option.prompt_kind.read(\"k=x\") == null")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=ii")).PromptKind.Valid, "Option.prompt_kind.read(\"k=ii\") == null")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=")).PromptKind.Valid, "Option.prompt_kind.read(\"k=\") == null")
}

// Zig: test "Option.read err"
func TestOSCParsersSemanticPrompt_Option_read_err(t *testing.T) {
	expectEqualStrings(t, "some_error", string(zigUnwrap(t, OSCParsersSemanticPromptOptionErr.Read([]uint8("err=some_error")).Err)))
	expect(t, !OSCParsersSemanticPromptOptionErr.Read([]uint8("aid=foo")).Err.Valid, "Option.err.read(\"aid=foo\") == null")
}

// Zig: test "Option.read redraw"
func TestOSCParsersSemanticPrompt_Option_read_redraw(t *testing.T) {
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=1")).Redraw) == OSCParsersSemanticPromptRedrawTrue, "Option.redraw.read(\"redraw=1\").? == .true")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=0")).Redraw) == OSCParsersSemanticPromptRedrawFalse, "Option.redraw.read(\"redraw=0\").? == .false")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=last")).Redraw) == OSCParsersSemanticPromptRedrawLast, "Option.redraw.read(\"redraw=last\").? == .last")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=2")).Redraw.Valid, "Option.redraw.read(\"redraw=2\") == null")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=10")).Redraw.Valid, "Option.redraw.read(\"redraw=10\") == null")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=")).Redraw.Valid, "Option.redraw.read(\"redraw=\") == null")
}

// Zig: test "Option.read special_key"
func TestOSCParsersSemanticPrompt_Option_read_special_key(t *testing.T) {
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionSpecialKey.Read([]uint8("special_key=1")).SpecialKey) == true, "Option.special_key.read(\"special_key=1\").? == true")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionSpecialKey.Read([]uint8("special_key=0")).SpecialKey) == false, "Option.special_key.read(\"special_key=0\").? == false")
	expect(t, !OSCParsersSemanticPromptOptionSpecialKey.Read([]uint8("special_key=x")).SpecialKey.Valid, "Option.special_key.read(\"special_key=x\") == null")
}

// Zig: test "Option.read click_events"
func TestOSCParsersSemanticPrompt_Option_read_click_events(t *testing.T) {
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionClickEvents.Read([]uint8("click_events=1")).ClickEvents) == true, "Option.click_events.read(\"click_events=1\").? == true")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionClickEvents.Read([]uint8("click_events=0")).ClickEvents) == false, "Option.click_events.read(\"click_events=0\").? == false")
	expect(t, !OSCParsersSemanticPromptOptionClickEvents.Read([]uint8("click_events=yes")).ClickEvents.Valid, "Option.click_events.read(\"click_events=yes\") == null")
}

// Zig: test "Option.read exit_code"
func TestOSCParsersSemanticPrompt_Option_read_exit_code(t *testing.T) {
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionExitCode.Read([]uint8("42")).ExitCode) == 42, "Option.exit_code.read(\"42\").? == 42")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionExitCode.Read([]uint8("0")).ExitCode) == 0, "Option.exit_code.read(\"0\").? == 0")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionExitCode.Read([]uint8("-1")).ExitCode) == -1, "Option.exit_code.read(\"-1\").? == -1")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("abc")).ExitCode.Valid, "Option.exit_code.read(\"abc\") == null")
	expect(t, zigUnwrap(t, OSCParsersSemanticPromptOptionExitCode.Read([]uint8("127;aid=foo")).ExitCode) == 127, "Option.exit_code.read(\"127;aid=foo\").? == 127")
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// term51 test helper: feed every byte of input to the parser.
func oscB_sp_term51Feed(p *OSCParser, input string) {
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
}

// Zig: test "term51: Option.key: key string of every option"
func TestOSCParsersSemanticPrompt_term51_Option_key_key_string_of_every_option(t *testing.T) {
	expectEqualStrings(t, "aid", string(OSCParsersSemanticPromptOptionAid.key()))
	expectEqualStrings(t, "cl", string(OSCParsersSemanticPromptOptionCl.key()))
	// prompt_kind is the only option whose key differs from its name.
	expectEqualStrings(t, "k", string(OSCParsersSemanticPromptOptionPromptKind.key()))
	expectEqualStrings(t, "err", string(OSCParsersSemanticPromptOptionErr.key()))
	expectEqualStrings(t, "redraw", string(OSCParsersSemanticPromptOptionRedraw.key()))
	expectEqualStrings(t, "special_key", string(OSCParsersSemanticPromptOptionSpecialKey.key()))
	expectEqualStrings(t, "click_events", string(OSCParsersSemanticPromptOptionClickEvents.key()))
	expectEqualStrings(t, "cmdline", string(OSCParsersSemanticPromptOptionCmdline.key()))
	expectEqualStrings(t, "cmdline_url", string(OSCParsersSemanticPromptOptionCmdlineUrl.key()))
	// Option.exit_code.key() is unreachable (it has no key) and cannot be
	// called.
}

// Zig: test "term51: Option.read: empty input"
func TestOSCParsersSemanticPrompt_term51_Option_read_empty_input(t *testing.T) {
	for _, raw := range []string{"", ";", ";;;"} {
		expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8(raw)).Aid.Valid, "Option.aid.read(raw) == null")
		expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8(raw)).Cl.Valid, "Option.cl.read(raw) == null")
		expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8(raw)).PromptKind.Valid, "Option.prompt_kind.read(raw) == null")
		expect(t, !OSCParsersSemanticPromptOptionErr.Read([]uint8(raw)).Err.Valid, "Option.err.read(raw) == null")
		expect(t, !OSCParsersSemanticPromptOptionCmdline.Read([]uint8(raw)).Cmdline.Valid, "Option.cmdline.read(raw) == null")
		expect(t, !OSCParsersSemanticPromptOptionCmdlineUrl.Read([]uint8(raw)).CmdlineUrl.Valid, "Option.cmdline_url.read(raw) == null")
		expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8(raw)).Redraw.Valid, "Option.redraw.read(raw) == null")
		expect(t, !OSCParsersSemanticPromptOptionSpecialKey.Read([]uint8(raw)).SpecialKey.Valid, "Option.special_key.read(raw) == null")
		expect(t, !OSCParsersSemanticPromptOptionClickEvents.Read([]uint8(raw)).ClickEvents.Valid, "Option.click_events.read(raw) == null")
		expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8(raw)).ExitCode.Valid, "Option.exit_code.read(raw) == null")
	}
}

// Zig: test "term51: Option.read: string options are returned verbatim"
func TestOSCParsersSemanticPrompt_term51_Option_read_string_options_are_returned_verbatim(t *testing.T) {
	expectEqualStrings(t, "14", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid=14")).Aid)))
	expectEqualStrings(t, "E", string(zigUnwrap(t, OSCParsersSemanticPromptOptionErr.Read([]uint8("err=E")).Err)))
	expectEqualStrings(t, "ls -l", string(zigUnwrap(t, OSCParsersSemanticPromptOptionCmdline.Read([]uint8("cmdline=ls -l")).Cmdline)))
	expectEqualStrings(t, "ls%20-l", string(zigUnwrap(t, OSCParsersSemanticPromptOptionCmdlineUrl.Read([]uint8("cmdline_url=ls%20-l")).CmdlineUrl)))
	// An empty value is an empty string, not null.
	expectEqualStrings(t, "", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid=")).Aid)))
	expectEqualStrings(t, "", string(zigUnwrap(t, OSCParsersSemanticPromptOptionErr.Read([]uint8("err=;aid=1")).Err)))
	expectEqualStrings(t, "", string(zigUnwrap(t, OSCParsersSemanticPromptOptionCmdline.Read([]uint8("cmdline=")).Cmdline)))
	expectEqualStrings(t, "", string(zigUnwrap(t, OSCParsersSemanticPromptOptionCmdlineUrl.Read([]uint8("cmdline_url=")).CmdlineUrl)))
	// Everything after the first '=' up to the next ';' is the value.
	expectEqualStrings(t, "a=b=c", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid=a=b=c")).Aid)))
	expectEqualStrings(t, "=", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid==")).Aid)))
	expectEqualStrings(t, " x ", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid= x ")).Aid)))
	expectEqualSlices(t, []uint8("\x00\x01\xff"), zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid=\x00\x01\xff")).Aid))
}

// Zig: test "term51: Option.read: option position"
func TestOSCParsersSemanticPrompt_term51_Option_read_option_position(t *testing.T) {
	const raw = "aid=1;cl=m;k=c;err=2;redraw=last;special_key=1;click_events=0;cmdline=a;cmdline_url=b"
	expectEqualStrings(t, "1", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8(raw)).Aid)))
	expectEqual(t, Some(OSCParsersSemanticPromptClickMultiple), OSCParsersSemanticPromptOptionCl.Read([]uint8(raw)).Cl)
	expectEqual(t, Some(OSCParsersSemanticPromptPromptKindContinuation), OSCParsersSemanticPromptOptionPromptKind.Read([]uint8(raw)).PromptKind)
	expectEqualStrings(t, "2", string(zigUnwrap(t, OSCParsersSemanticPromptOptionErr.Read([]uint8(raw)).Err)))
	expectEqual(t, Some(OSCParsersSemanticPromptRedrawLast), OSCParsersSemanticPromptOptionRedraw.Read([]uint8(raw)).Redraw)
	expectEqual(t, Some(true), OSCParsersSemanticPromptOptionSpecialKey.Read([]uint8(raw)).SpecialKey)
	expectEqual(t, Some(false), OSCParsersSemanticPromptOptionClickEvents.Read([]uint8(raw)).ClickEvents)
	expectEqualStrings(t, "a", string(zigUnwrap(t, OSCParsersSemanticPromptOptionCmdline.Read([]uint8(raw)).Cmdline)))
	expectEqualStrings(t, "b", string(zigUnwrap(t, OSCParsersSemanticPromptOptionCmdlineUrl.Read([]uint8(raw)).CmdlineUrl)))
	// exit_code only looks at the first field.
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8(raw)).ExitCode.Valid, "Option.exit_code.read(raw) == null")

	// Empty fields and a trailing ';' are skipped.
	expectEqualStrings(t, "x", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8(";;aid=x;;")).Aid)))
	expectEqualStrings(t, "x", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("=;=y;novalue;aid=x")).Aid)))
	// Missing among other options.
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("cl=m;k=c")).Aid.Valid, "Option.aid.read(\"cl=m;k=c\") == null")
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("cl=m;k=c;")).Aid.Valid, "Option.aid.read(\"cl=m;k=c;\") == null")
}

// Zig: test "term51: Option.read: keys must match exactly"
func TestOSCParsersSemanticPrompt_term51_Option_read_keys_must_match_exactly(t *testing.T) {
	// case sensitive
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("AID=x")).Aid.Valid, "Option.aid.read(\"AID=x\") == null")
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("Aid=x")).Aid.Valid, "Option.aid.read(\"Aid=x\") == null")
	// no trimming
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8(" aid=x")).Aid.Valid, "Option.aid.read(\" aid=x\") == null")
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("aid =x")).Aid.Valid, "Option.aid.read(\"aid =x\") == null")
	// no '='
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("aid")).Aid.Valid, "Option.aid.read(\"aid\") == null")
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("aid;x")).Aid.Valid, "Option.aid.read(\"aid;x\") == null")
	// no prefix or suffix matching
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("aidx=1")).Aid.Valid, "Option.aid.read(\"aidx=1\") == null")
	expect(t, !OSCParsersSemanticPromptOptionAid.Read([]uint8("xaid=1")).Aid.Valid, "Option.aid.read(\"xaid=1\") == null")
	expect(t, !OSCParsersSemanticPromptOptionCmdline.Read([]uint8("cmdline_url=x")).Cmdline.Valid, "Option.cmdline.read(\"cmdline_url=x\") == null")
	expect(t, !OSCParsersSemanticPromptOptionCmdlineUrl.Read([]uint8("cmdline=x")).CmdlineUrl.Valid, "Option.cmdline_url.read(\"cmdline=x\") == null")
	expectEqualStrings(t, "y", string(zigUnwrap(t, OSCParsersSemanticPromptOptionCmdline.Read([]uint8("cmdline_url=x;cmdline=y")).Cmdline)))
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("click_events=1")).Cl.Valid, "Option.cl.read(\"click_events=1\") == null")
	// The prompt kind is read from "k", not from "prompt_kind".
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("prompt_kind=i")).PromptKind.Valid, "Option.prompt_kind.read(\"prompt_kind=i\") == null")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("K=i")).PromptKind.Valid, "Option.prompt_kind.read(\"K=i\") == null")
}

// Zig: test "term51: Option.read: the first occurrence of a key decides"
func TestOSCParsersSemanticPrompt_term51_Option_read_the_first_occurrence_of_a_key_decides(t *testing.T) {
	expectEqualStrings(t, "a", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid=a;aid=b")).Aid)))
	expectEqualStrings(t, "", string(zigUnwrap(t, OSCParsersSemanticPromptOptionAid.Read([]uint8("aid=;aid=b")).Aid)))
	// Even if the first occurrence is invalid.
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=bogus;cl=m")).Cl.Valid, "Option.cl.read(\"cl=bogus;cl=m\") == null")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=x;k=i")).PromptKind.Valid, "Option.prompt_kind.read(\"k=x;k=i\") == null")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=2;redraw=1")).Redraw.Valid, "Option.redraw.read(\"redraw=2;redraw=1\") == null")
	expect(t, !OSCParsersSemanticPromptOptionSpecialKey.Read([]uint8("special_key=;special_key=1")).SpecialKey.Valid, "Option.special_key.read(\"special_key=;special_key=1\") == null")
}

// Zig: test "term51: Option.read: cl values"
func TestOSCParsersSemanticPrompt_term51_Option_read_cl_values(t *testing.T) {
	expectEqual(t, Some(OSCParsersSemanticPromptClickLine), OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=line")).Cl)
	expectEqual(t, Some(OSCParsersSemanticPromptClickMultiple), OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=m")).Cl)
	expectEqual(t, Some(OSCParsersSemanticPromptClickConservativeVertical), OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=v")).Cl)
	expectEqual(t, Some(OSCParsersSemanticPromptClickSmartVertical), OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=w")).Cl)
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=")).Cl.Valid, "Option.cl.read(\"cl=\") == null")
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=l")).Cl.Valid, "Option.cl.read(\"cl=l\") == null")
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=LINE")).Cl.Valid, "Option.cl.read(\"cl=LINE\") == null")
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=M")).Cl.Valid, "Option.cl.read(\"cl=M\") == null")
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=mm")).Cl.Valid, "Option.cl.read(\"cl=mm\") == null")
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=line ")).Cl.Valid, "Option.cl.read(\"cl=line \") == null")
	expect(t, !OSCParsersSemanticPromptOptionCl.Read([]uint8("cl=multiple")).Cl.Valid, "Option.cl.read(\"cl=multiple\") == null")
}

// Zig: test "term51: Option.read: prompt kind values"
func TestOSCParsersSemanticPrompt_term51_Option_read_prompt_kind_values(t *testing.T) {
	expectEqual(t, Some(OSCParsersSemanticPromptPromptKindInitial), OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=i")).PromptKind)
	expectEqual(t, Some(OSCParsersSemanticPromptPromptKindRight), OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=r")).PromptKind)
	expectEqual(t, Some(OSCParsersSemanticPromptPromptKindContinuation), OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=c")).PromptKind)
	expectEqual(t, Some(OSCParsersSemanticPromptPromptKindSecondary), OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=s")).PromptKind)
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=")).PromptKind.Valid, "Option.prompt_kind.read(\"k=\") == null")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=x")).PromptKind.Valid, "Option.prompt_kind.read(\"k=x\") == null")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=I")).PromptKind.Valid, "Option.prompt_kind.read(\"k=I\") == null")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=ii")).PromptKind.Valid, "Option.prompt_kind.read(\"k=ii\") == null")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k=initial")).PromptKind.Valid, "Option.prompt_kind.read(\"k=initial\") == null")
	expect(t, !OSCParsersSemanticPromptOptionPromptKind.Read([]uint8("k= i")).PromptKind.Valid, "Option.prompt_kind.read(\"k= i\") == null")
}

// Zig: test "term51: Option.read: redraw values"
func TestOSCParsersSemanticPrompt_term51_Option_read_redraw_values(t *testing.T) {
	expectEqual(t, Some(OSCParsersSemanticPromptRedrawFalse), OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=0")).Redraw)
	expectEqual(t, Some(OSCParsersSemanticPromptRedrawTrue), OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=1")).Redraw)
	expectEqual(t, Some(OSCParsersSemanticPromptRedrawLast), OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=last")).Redraw)
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=")).Redraw.Valid, "Option.redraw.read(\"redraw=\") == null")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=2")).Redraw.Valid, "Option.redraw.read(\"redraw=2\") == null")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=00")).Redraw.Valid, "Option.redraw.read(\"redraw=00\") == null")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=true")).Redraw.Valid, "Option.redraw.read(\"redraw=true\") == null")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=false")).Redraw.Valid, "Option.redraw.read(\"redraw=false\") == null")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=LAST")).Redraw.Valid, "Option.redraw.read(\"redraw=LAST\") == null")
	expect(t, !OSCParsersSemanticPromptOptionRedraw.Read([]uint8("redraw=last ")).Redraw.Valid, "Option.redraw.read(\"redraw=last \") == null")
}

// Zig: test "term51: Option.read: boolean values"
func TestOSCParsersSemanticPrompt_term51_Option_read_boolean_values(t *testing.T) {
	expectEqual(t, Some(false), OSCParsersSemanticPromptOptionSpecialKey.Read([]uint8("special_key=0")).SpecialKey)
	expectEqual(t, Some(true), OSCParsersSemanticPromptOptionSpecialKey.Read([]uint8("special_key=1")).SpecialKey)
	expectEqual(t, Some(false), OSCParsersSemanticPromptOptionClickEvents.Read([]uint8("click_events=0")).ClickEvents)
	expectEqual(t, Some(true), OSCParsersSemanticPromptOptionClickEvents.Read([]uint8("click_events=1")).ClickEvents)
	for _, value := range []string{"", "2", "01", "10", "true", "false", " 1", "1 ", "y"} {
		expect(t, !OSCParsersSemanticPromptOptionSpecialKey.Read([]uint8("special_key="+value)).SpecialKey.Valid, "Option.special_key.read(\"special_key=\" ++ value) == null")
		expect(t, !OSCParsersSemanticPromptOptionClickEvents.Read([]uint8("click_events="+value)).ClickEvents.Valid, "Option.click_events.read(\"click_events=\" ++ value) == null")
	}
}

// Zig: test "term51: Option.read: exit code is the first field"
func TestOSCParsersSemanticPrompt_term51_Option_read_exit_code_is_the_first_field(t *testing.T) {
	expectEqual(t, Some(int32(0)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("0")).ExitCode)
	expectEqual(t, Some(int32(1)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("1")).ExitCode)
	expectEqual(t, Some(int32(127)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("127;aid=x")).ExitCode)
	expectEqual(t, Some(int32(-1)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("-1")).ExitCode)
	// smallest and largest i32, and one past each
	expectEqual(t, Some(int32(2147483647)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("2147483647")).ExitCode)
	expectEqual(t, Some(int32(-2147483648)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("-2147483648")).ExitCode)
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("2147483648")).ExitCode.Valid, "Option.exit_code.read(\"2147483648\") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("-2147483649")).ExitCode.Valid, "Option.exit_code.read(\"-2147483649\") == null")
	// std.fmt.parseInt accepts a leading '+', leading zeros and '_'
	// between digits.
	expectEqual(t, Some(int32(7)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("+7")).ExitCode)
	expectEqual(t, Some(int32(7)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("007")).ExitCode)
	expectEqual(t, Some(int32(10)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("1_0")).ExitCode)
	expectEqual(t, Some(int32(0)), OSCParsersSemanticPromptOptionExitCode.Read([]uint8("-0")).ExitCode)
	// Not a number.
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("abc")).ExitCode.Valid, "Option.exit_code.read(\"abc\") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8(" 1")).ExitCode.Valid, "Option.exit_code.read(\" 1\") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("1 ")).ExitCode.Valid, "Option.exit_code.read(\"1 \") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("0x10")).ExitCode.Valid, "Option.exit_code.read(\"0x10\") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("1.5")).ExitCode.Valid, "Option.exit_code.read(\"1.5\") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("+")).ExitCode.Valid, "Option.exit_code.read(\"+\") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("-")).ExitCode.Valid, "Option.exit_code.read(\"-\") == null")
	// Only the first field is looked at: an exit code after another
	// option or after an empty field is not found, and there is no
	// "exit_code" key.
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("aid=x;5")).ExitCode.Valid, "Option.exit_code.read(\"aid=x;5\") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8(";5")).ExitCode.Valid, "Option.exit_code.read(\";5\") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("exit_code=5")).ExitCode.Valid, "Option.exit_code.read(\"exit_code=5\") == null")
	expect(t, !OSCParsersSemanticPromptOptionExitCode.Read([]uint8("err=5")).ExitCode.Valid, "Option.exit_code.read(\"err=5\") == null")
}

// Zig: test "term51: Click.init: values are exact and case sensitive"
func TestOSCParsersSemanticPrompt_term51_Click_init_values_are_exact_and_case_sensitive(t *testing.T) {
	expectEqual(t, Some(OSCParsersSemanticPromptClickLine), OSCParsersSemanticPromptClickInit([]uint8("line")))
	expectEqual(t, Some(OSCParsersSemanticPromptClickMultiple), OSCParsersSemanticPromptClickInit([]uint8("m")))
	expectEqual(t, Some(OSCParsersSemanticPromptClickConservativeVertical), OSCParsersSemanticPromptClickInit([]uint8("v")))
	expectEqual(t, Some(OSCParsersSemanticPromptClickSmartVertical), OSCParsersSemanticPromptClickInit([]uint8("w")))
	expect(t, !OSCParsersSemanticPromptClickInit([]uint8("")).Valid, "Click.init(\"\") == null")
	expect(t, !OSCParsersSemanticPromptClickInit([]uint8("l")).Valid, "Click.init(\"l\") == null")
	expect(t, !OSCParsersSemanticPromptClickInit([]uint8("x")).Valid, "Click.init(\"x\") == null")
	expect(t, !OSCParsersSemanticPromptClickInit([]uint8("V")).Valid, "Click.init(\"V\") == null")
	expect(t, !OSCParsersSemanticPromptClickInit([]uint8("Line")).Valid, "Click.init(\"Line\") == null")
	expect(t, !OSCParsersSemanticPromptClickInit([]uint8("lines")).Valid, "Click.init(\"lines\") == null")
	expect(t, !OSCParsersSemanticPromptClickInit([]uint8("lin")).Valid, "Click.init(\"lin\") == null")
	expect(t, !OSCParsersSemanticPromptClickInit([]uint8("mv")).Valid, "Click.init(\"mv\") == null")
}

// Zig: test "term51: PromptKind.init: every byte value"
func TestOSCParsersSemanticPrompt_term51_PromptKind_init_every_byte_value(t *testing.T) {
	for i := 0; i < 256; i++ {
		c := uint8(i)
		var expected Optional[OSCParsersSemanticPromptPromptKind]
		switch c {
		case 'i':
			expected = Some(OSCParsersSemanticPromptPromptKindInitial)
		case 'r':
			expected = Some(OSCParsersSemanticPromptPromptKindRight)
		case 'c':
			expected = Some(OSCParsersSemanticPromptPromptKindContinuation)
		case 's':
			expected = Some(OSCParsersSemanticPromptPromptKindSecondary)
		default:
			expected = None[OSCParsersSemanticPromptPromptKind]()
		}
		expectEqual(t, expected, OSCParsersSemanticPromptPromptKindInit(c))
	}
}

// Zig: test "term51: Command.init: no options"
func TestOSCParsersSemanticPrompt_term51_Command_init_no_options(t *testing.T) {
	for i := 0; i < oscParsersSemanticPromptCommandActionCount; i++ {
		action := OSCParsersSemanticPromptCommandAction(i)
		cmd := OSCParsersSemanticPromptCommandInit(action)
		expectEqual(t, action, cmd.action)
		expectEqual(t, 0, len(cmd.options_unvalidated))
	}
	expectEqual(t, 8, oscParsersSemanticPromptCommandActionCount)
}

// Zig: test "term51: Command.readOption: reads from the unvalidated options"
func TestOSCParsersSemanticPrompt_term51_Command_readOption_reads_from_the_unvalidated_options(t *testing.T) {
	cmd := OSCParsersSemanticPromptCommand{
		action:              OSCParsersSemanticPromptCommandActionEndCommand,
		options_unvalidated: []uint8("3;aid=x;err=E"),
	}
	expectEqual(t, Some(int32(3)), cmd.ReadOption(OSCParsersSemanticPromptOptionExitCode).ExitCode)
	expectEqualStrings(t, "x", string(zigUnwrap(t, cmd.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
	expectEqualStrings(t, "E", string(zigUnwrap(t, cmd.ReadOption(OSCParsersSemanticPromptOptionErr).Err)))
	expect(t, !cmd.ReadOption(OSCParsersSemanticPromptOptionCl).Cl.Valid, "cmd.readOption(.cl) == null")

	empty := OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt)
	expect(t, !empty.ReadOption(OSCParsersSemanticPromptOptionAid).Aid.Valid, "empty.readOption(.aid) == null")
	expect(t, !empty.ReadOption(OSCParsersSemanticPromptOptionExitCode).ExitCode.Valid, "empty.readOption(.exit_code) == null")
	expect(t, !empty.ReadOption(OSCParsersSemanticPromptOptionRedraw).Redraw.Valid, "empty.readOption(.redraw) == null")
}

// Zig: test "term51: Command.writeCommandLine: cmdline takes precedence over cmdline_url"
func TestOSCParsersSemanticPrompt_term51_Command_writeCommandLine_cmdline_takes_precedence_over_cmdline_url(t *testing.T) {
	{
		var w oscWriterAllocating
		// Zig-only: defer w.deinit();
		cmd := OSCParsersSemanticPromptCommand{
			action:              OSCParsersSemanticPromptCommandActionEndInputStartOutput,
			options_unvalidated: []uint8("cmdline_url=from%20url;cmdline=from cmdline"),
		}
		zigTry(t, cmd.WriteCommandLine(&w.writer))
		expectEqualStrings(t, "from cmdline", oscB_sp_written(&w))
	}
	{
		var w oscWriterAllocating
		// Zig-only: defer w.deinit();
		cmd := OSCParsersSemanticPromptCommand{
			action:              OSCParsersSemanticPromptCommandActionEndInputStartOutput,
			options_unvalidated: []uint8("aid=1;cmdline_url=from%20url"),
		}
		zigTry(t, cmd.WriteCommandLine(&w.writer))
		expectEqualStrings(t, "from url", oscB_sp_written(&w))
	}
	{
		// Neither option: nothing is written.
		var w oscWriterAllocating
		// Zig-only: defer w.deinit();
		cmd := OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndInputStartOutput)
		zigTry(t, cmd.WriteCommandLine(&w.writer))
		expectEqualStrings(t, "", oscB_sp_written(&w))
	}
}

// Zig: test "term51: parse: missing separator and empty data are invalid"
func TestOSCParsersSemanticPrompt_term51_parse_missing_separator_and_empty_data_are_invalid(t *testing.T) {
	{
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, "133")
		expect(t, p.writer == nil, "p.writer == null")
		expect(t, OSCParsersSemanticPromptParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, "133;")
		expect(t, p.state == OSCParserState133, "p.state == .@\"133\"")
		expect(t, OSCParsersSemanticPromptParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: every command letter"
func TestOSCParsersSemanticPrompt_term51_parse_every_command_letter(t *testing.T) {
	for _, c := range []struct {
		letter string
		action OSCParsersSemanticPromptCommandAction
	}{
		{"A", OSCParsersSemanticPromptCommandActionFreshLineNewPrompt},
		{"B", OSCParsersSemanticPromptCommandActionEndPromptStartInput},
		{"C", OSCParsersSemanticPromptCommandActionEndInputStartOutput},
		{"D", OSCParsersSemanticPromptCommandActionEndCommand},
		{"I", OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol},
		{"N", OSCParsersSemanticPromptCommandActionNewCommand},
		{"P", OSCParsersSemanticPromptCommandActionPromptStart},
	} {
		for _, term := range []Optional[uint8]{None[uint8](), Some(uint8(0x07)), Some(uint8(0x1b))} {
			// The letter alone.
			{
				p := OSCParserInit(nil)
				oscB_sp_term51Feed(&p, "133;"+c.letter)
				cmd := oscB_sp_zigUnwrapPtr(t, OSCParsersSemanticPromptParse(&p, term))
				expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
				expectEqual(t, c.action, cmd.SemanticPrompt.action)
				expectEqualStrings(t, "", string(cmd.SemanticPrompt.options_unvalidated))
			}
			// The letter and a separator with nothing after it.
			{
				p := OSCParserInit(nil)
				oscB_sp_term51Feed(&p, "133;"+c.letter+";")
				cmd := oscB_sp_zigUnwrapPtr(t, OSCParsersSemanticPromptParse(&p, term))
				expectEqual(t, c.action, cmd.SemanticPrompt.action)
				expectEqualStrings(t, "", string(cmd.SemanticPrompt.options_unvalidated))
			}
			// Options are everything after the separator, unparsed.
			{
				p := OSCParserInit(nil)
				oscB_sp_term51Feed(&p, "133;"+c.letter+";aid=1;;junk\x00\x01\xff;")
				cmd := oscB_sp_zigUnwrapPtr(t, OSCParsersSemanticPromptParse(&p, term))
				expectEqual(t, c.action, cmd.SemanticPrompt.action)
				expectEqualSlices(t, []uint8("aid=1;;junk\x00\x01\xff;"), cmd.SemanticPrompt.options_unvalidated)
				expectEqualStrings(t, "1", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
			}
		}
		// Anything other than ';' after the letter is invalid.
		for _, suffix := range []string{"x", " ", "=", c.letter, " ;aid=1"} {
			p := OSCParserInit(nil)
			oscB_sp_term51Feed(&p, "133;"+c.letter+suffix)
			expect(t, OSCParsersSemanticPromptParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
			expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		}
	}
}

// Zig: test "term51: parse: L takes no options at all"
func TestOSCParsersSemanticPrompt_term51_parse_L_takes_no_options_at_all(t *testing.T) {
	{
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, "133;L")
		cmd := oscB_sp_zigUnwrapPtr(t, OSCParsersSemanticPromptParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeySemanticPrompt, "cmd == .semantic_prompt")
		expectEqual(t, OSCParsersSemanticPromptCommandActionFreshLine, cmd.SemanticPrompt.action)
		expectEqualStrings(t, "", string(cmd.SemanticPrompt.options_unvalidated))
	}
	// Unlike the other letters, not even a bare separator is allowed.
	for _, input := range []string{"133;L;", "133;L;aid=1", "133;Lx", "133;LL"} {
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, input)
		expect(t, OSCParsersSemanticPromptParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: unknown and lower case letters are invalid"
func TestOSCParsersSemanticPrompt_term51_parse_unknown_and_lower_case_letters_are_invalid(t *testing.T) {
	for _, input := range []string{
		"133;a",
		"133;b",
		"133;c",
		"133;d",
		"133;i",
		"133;l",
		"133;n",
		"133;p",
		"133;E",
		"133;Z",
		"133;0",
		"133;;A",
		"133; A",
		"133;\x00",
		"133;aid=1",
	} {
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, input)
		expect(t, OSCParsersSemanticPromptParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: exit code position"
func TestOSCParsersSemanticPrompt_term51_parse_exit_code_position(t *testing.T) {
	{
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, "133;D;0;aid=x")
		cmd := oscB_sp_zigUnwrapPtr(t, OSCParsersSemanticPromptParse(&p, None[uint8]()))
		expectEqual(t, Some(int32(0)), cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionExitCode).ExitCode)
		expectEqualStrings(t, "x", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
	}
	{
		// Not the first field: not found.
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, "133;D;aid=x;0")
		cmd := oscB_sp_zigUnwrapPtr(t, OSCParsersSemanticPromptParse(&p, None[uint8]()))
		expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionExitCode).ExitCode.Valid, "cmd.semantic_prompt.readOption(.exit_code) == null")
		expectEqualStrings(t, "x", string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
	}
	{
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, "133;D")
		cmd := oscB_sp_zigUnwrapPtr(t, OSCParsersSemanticPromptParse(&p, None[uint8]()))
		expect(t, !cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionExitCode).ExitCode.Valid, "cmd.semantic_prompt.readOption(.exit_code) == null")
	}
	{
		// The exit code is not tied to the D command.
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, "133;A;5")
		cmd := oscB_sp_zigUnwrapPtr(t, OSCParsersSemanticPromptParse(&p, None[uint8]()))
		expectEqual(t, OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, cmd.SemanticPrompt.action)
		expectEqual(t, Some(int32(5)), cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionExitCode).ExitCode)
	}
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersSemanticPrompt_term51_parse_buffer_limit(t *testing.T) {
	// No NUL terminator is needed, so the data may fill the whole buffer.
	const prefix = "A;aid="
	{
		value := strings.Repeat("v", OSCParserMAX_BUF-len(prefix))
		p := OSCParserInit(nil)
		oscB_sp_term51Feed(&p, "133;"+prefix+value)
		expect(t, p.state == OSCParserState133, "p.state == .@\"133\"")
		cmd := oscB_sp_zigUnwrapPtr(t, OSCParsersSemanticPromptParse(&p, None[uint8]()))
		expectEqual(t, OSCParserMAX_BUF-2, len(cmd.SemanticPrompt.options_unvalidated))
		expectEqualStrings(t, value, string(zigUnwrap(t, cmd.SemanticPrompt.ReadOption(OSCParsersSemanticPromptOptionAid).Aid)))
	}
	{
		// One more byte overflows while feeding; an allocator does not help.
		value := strings.Repeat("v", OSCParserMAX_BUF-len(prefix)+1)
		p := OSCParserInit(oscB_sp_allocator)
		defer p.Deinit()
		oscB_sp_term51Feed(&p, "133;"+prefix+value)
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}
}
