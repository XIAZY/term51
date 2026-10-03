// Tests translated one for one from src/terminal/apc.zig (Ghostty v1.3.1 plus
// the term51 additional tests). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. The helpers expect and expectEqual are defined in
// Parser_test.go.
//
// Translation notes:
//   - Allocator arguments have no Go counterpart (docs/mapping.md), so
//     `h.feed(alloc, c)` is h.Feed(c) and the `const alloc =
//     testing.allocator;` lines are dropped.
//   - `var h: Handler = .{}` is the zero value of APCHandler.
//   - `h.state == .ignore` is `h.state.tag == APCStateIgnore`.
//   - Kitty graphics is compiled out of the library build
//     (build_options.kitty_graphics is false), so the tests that need it are
//     skipped, as `error.SkipZigTest` skips them in Zig.
//   - Zig's `defer` inside a loop body runs at the end of each iteration;
//     Go's defer runs when the test function returns.

package terminal

import (
	"testing"
)

// Zig: test "unknown APC command"
func TestAPC_unknown_APC_command(t *testing.T) {
	var h APCHandler
	h.Start()
	for _, c := range []uint8("Xabcdef1234") {
		h.Feed(c)
	}
	expect(t, !h.End().Valid, "h.end() == null")
}

// Zig: test "garbage Kitty command"
func TestAPC_garbage_Kitty_command(t *testing.T) {
	// Zig: if (comptime !build_options.kitty_graphics) return error.SkipZigTest;
	t.Skip("Kitty graphics is not compiled into the library build")
}

// Zig: test "Kitty command with overflow u32"
func TestAPC_Kitty_command_with_overflow_u32(t *testing.T) {
	// Zig: if (comptime !build_options.kitty_graphics) return error.SkipZigTest;
	t.Skip("Kitty graphics is not compiled into the library build")
}

// Zig: test "Kitty command with overflow i32"
func TestAPC_Kitty_command_with_overflow_i32(t *testing.T) {
	// Zig: if (comptime !build_options.kitty_graphics) return error.SkipZigTest;
	t.Skip("Kitty graphics is not compiled into the library build")
}

// Zig: test "valid Kitty command"
func TestAPC_valid_Kitty_command(t *testing.T) {
	// Zig: if (comptime !build_options.kitty_graphics) return error.SkipZigTest;
	t.Skip("Kitty graphics is not compiled into the library build")
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
//
// In the library build the Kitty graphics protocol is compiled out, so the
// only reachable behaviour is "identify, then ignore". Command.deinit cannot
// be called in that build (its only variant is unreachable), so it has no
// direct test here.
// ---------------------------------------------------------------------------

// Zig: test "term51: Handler.start: enters identify from any state"
func TestAPC_term51_Handler_start_enters_identify_from_any_state(t *testing.T) {
	var h APCHandler
	defer h.Deinit()
	expect(t, h.state.tag == APCStateInactive, "h.state == .inactive")
	h.Start()
	expect(t, h.state.tag == APCStateIdentify, "h.state == .identify")
	// starting again while a string is open restarts identification
	h.Feed('X')
	expect(t, h.state.tag == APCStateIgnore, "h.state == .ignore")
	h.Start()
	expect(t, h.state.tag == APCStateIdentify, "h.state == .identify")
}

// Zig: test "term51: Handler.feed: the first byte decides, the rest are ignored"
func TestAPC_term51_Handler_feed_the_first_byte_decides_the_rest_are_ignored(t *testing.T) {
	var byte_ int
	for byte_ = 0; byte_ < 256; byte_++ {
		// Zig-only: if (comptime build_options.kitty_graphics) { if (byte == 'G') continue; }
		// (Kitty graphics is compiled out, so 'G' is not skipped.)
		var h APCHandler
		defer h.Deinit()
		h.Start()
		h.Feed(uint8(byte_))
		expect(t, h.state.tag == APCStateIgnore, "h.state == .ignore")
		h.Feed('G')
		h.Feed(0)
		expect(t, h.state.tag == APCStateIgnore, "h.state == .ignore")
		expect(t, !h.End().Valid, "h.end() == null")
	}
}

// Zig: test "term51: Handler.feed: G is ignored when Kitty graphics is compiled out"
func TestAPC_term51_Handler_feed_G_is_ignored_when_Kitty_graphics_is_compiled_out(t *testing.T) {
	// Zig: if (comptime build_options.kitty_graphics) return error.SkipZigTest;
	// (Kitty graphics is compiled out, so the test runs.)
	var h APCHandler
	defer h.Deinit()
	h.Start()
	h.Feed('G')
	expect(t, h.state.tag == APCStateIgnore, "h.state == .ignore")
	for _, c := range []uint8("a=q,i=1;AAAA") {
		h.Feed(c)
	}
	expect(t, !h.End().Valid, "h.end() == null")
}

// Zig: test "term51: Handler.end: an empty string yields nothing and returns to inactive"
func TestAPC_term51_Handler_end_an_empty_string_yields_nothing_and_returns_to_inactive(t *testing.T) {
	var h APCHandler
	defer h.Deinit()
	h.Start()
	expect(t, !h.End().Valid, "h.end() == null")
	expect(t, h.state.tag == APCStateInactive, "h.state == .inactive")
}

// Zig: test "term51: Handler.end: the handler is reusable"
func TestAPC_term51_Handler_end_the_handler_is_reusable(t *testing.T) {
	var h APCHandler
	defer h.Deinit()
	for range 3 {
		h.Start()
		for _, c := range []uint8("Xabc") {
			h.Feed(c)
		}
		expect(t, !h.End().Valid, "h.end() == null")
		expect(t, h.state.tag == APCStateInactive, "h.state == .inactive")
	}
}

// Zig: test "term51: Handler.deinit: safe in every reachable state"
func TestAPC_term51_Handler_deinit_safe_in_every_reachable_state(t *testing.T) {
	{
		var h APCHandler
		h.Deinit()
	}
	{
		var h APCHandler
		h.Start()
		h.Deinit()
	}
	{
		var h APCHandler
		h.Start()
		h.Feed('X')
		h.Deinit()
	}
}

// Zig: test "term51: State.deinit: the states that hold nothing"
func TestAPC_term51_State_deinit_the_states_that_hold_nothing(t *testing.T) {
	a := APCState{tag: APCStateInactive}
	a.Deinit()
	b := APCState{tag: APCStateIgnore}
	b.Deinit()
	c := APCState{tag: APCStateIdentify}
	c.Deinit()
}
