// Tests translated one for one from src/terminal/Terminal.zig (Ghostty
// v1.3.1 plus the term51 additional tests), the section under the banner
// "term51: additional tests (AA)" at the end of the file. Each Go test
// carries the exact Zig title in a "// Zig: test" comment. Helpers are
// prefixed tma.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err).
//   - The Zig terminal `t` is `term` here, because `t` is the *testing.T.
//   - `init(alloc, .{ .cols = c, .rows = r })` is tmaInit(t, c, r), which
//     writes out the Zig defaults (terminalOptionsDefault).
//   - `h.vt(.<key>, .{ ... })` is h.Vt(key, StreamAction{Tag: key, ...}).
//   - The Go stream holds its handler by pointer, so `.init(t.vtHandler())`
//     is StreamInit(&h) with h a variable.
//   - Allocator identity checks (`.ptr`, `.vtable`, FailingAllocator, the
//     `alloc` field of a Screen) are dropped with `// Zig-only:` comments;
//     Go has no allocator (see the header of Terminal.go).
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"reflect"
	"testing"
	"unsafe"
)

// tmaInit is `try init(alloc, .{ .cols = cols, .rows = rows })`.
func tmaInit(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	t.Helper()
	term, err := TerminalInit(terminalOptionsDefault(cols, rows))
	zigTry(t, err)
	return &term
}

// tmaPrint is `try h.vt(.print, .{ .cp = cp })`.
func tmaPrint(t *testing.T, h *StreamReadonlyHandler, cp uint32) {
	t.Helper()
	zigTry(t, h.Vt(StreamActionKeyPrint, &StreamAction{Tag: StreamActionKeyPrint, Print: StreamActionPrint{Cp: cp}}))
}

// tmaSetMode is `try h.vt(.set_mode, .{ .mode = mode })`.
func tmaSetMode(t *testing.T, h *StreamReadonlyHandler, mode ModesMode) {
	t.Helper()
	zigTry(t, h.Vt(StreamActionKeySetMode, &StreamAction{Tag: StreamActionKeySetMode, SetMode: StreamActionMode{Mode: mode}}))
}

// tmaResetMode is `try h.vt(.reset_mode, .{ .mode = mode })`.
func tmaResetMode(t *testing.T, h *StreamReadonlyHandler, mode ModesMode) {
	t.Helper()
	zigTry(t, h.Vt(StreamActionKeyResetMode, &StreamAction{Tag: StreamActionKeyResetMode, ResetMode: StreamActionMode{Mode: mode}}))
}

// tmaTerminalExpectPlain is term51AATerminalExpectPlain: expect the plain
// string of the viewport.
func tmaTerminalExpectPlain(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	str, err := term.PlainString()
	zigTry(t, err)
	// Zig-only: defer std.testing.allocator.free(str);
	expectEqualStrings(t, expected, string(str))
}

// Zig: test "term51: MouseEvents.motion: read from the terminal flags after each mouse mode"
func TestTma_term51_MouseEvents_motion_read_from_the_terminal_flags_after_each_mouse_mode(t *testing.T) {
	testStdCapacity(t)
	term := tmaInit(t, 10, 3)
	defer term.Deinit()
	h := term.VtHandler()
	defer h.Deinit()

	expectEqual(t, TerminalMouseEventsNone, term.Flags.MouseEvent)
	expect(t, !term.Flags.MouseEvent.Motion(), "!t.flags.mouse_event.motion()")

	type Case struct {
		mode   ModesMode
		event  TerminalMouseEvents
		motion bool
	}
	cases := [...]Case{
		{mode: ModesModeMouseEventX10, event: TerminalMouseEventsX10, motion: false},
		{mode: ModesModeMouseEventNormal, event: TerminalMouseEventsNormal, motion: false},
		{mode: ModesModeMouseEventButton, event: TerminalMouseEventsButton, motion: true},
		{mode: ModesModeMouseEventAny, event: TerminalMouseEventsAny, motion: true},
	}
	for _, c := range cases {
		tmaSetMode(t, &h, c.mode)
		expectEqual(t, c.event, term.Flags.MouseEvent)
		expectEqual(t, c.motion, term.Flags.MouseEvent.Motion())
		tmaResetMode(t, &h, c.mode)
		expectEqual(t, TerminalMouseEventsNone, term.Flags.MouseEvent)
		expect(t, !term.Flags.MouseEvent.Motion(), "!t.flags.mouse_event.motion()")
	}

	// The flag can also be written directly; the method follows it.
	for _, e := range [...]TerminalMouseEvents{
		TerminalMouseEventsNone,
		TerminalMouseEventsX10,
		TerminalMouseEventsNormal,
		TerminalMouseEventsButton,
		TerminalMouseEventsAny,
	} {
		term.Flags.MouseEvent = e
		expectEqual(t, e == TerminalMouseEventsButton || e == TerminalMouseEventsAny, term.Flags.MouseEvent.Motion())
	}
}

// Zig: test "term51: gpa: follows the active screen"
func TestTma_term51_gpa_follows_the_active_screen(t *testing.T) {
	testStdCapacity(t)
	term := tmaInit(t, 4, 4)
	defer term.Deinit()
	// Zig-only: var fa = std.testing.FailingAllocator.init(std.testing.allocator, .{});
	// Zig-only: const marker = fa.allocator();
	// Zig-only: var fa2 = std.testing.FailingAllocator.init(std.testing.allocator, .{});
	// Zig-only: const other = fa2.allocator();
	// Zig-only: const orig = std.testing.allocator;

	primary := term.Screens.Get(ScreenSetKeyPrimary)
	expect(t, primary != nil, "t.screens.get(.primary) != null")
	term.gpa() // Zig: t.gpa().ptr == orig.ptr
	// Zig-only: try std.testing.expect(t.gpa().ptr == orig.ptr);
	// Zig-only: primary.alloc = marker;
	// Zig-only: try std.testing.expect(t.gpa().ptr == marker.ptr);
	// Zig-only: try std.testing.expect(t.gpa().vtable == marker.vtable);
	// Zig-only: primary.alloc = orig;

	// The alternate screen is created with the allocator of the screen it
	// replaces, and is then the one asked.
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	alt := term.Screens.Get(ScreenSetKeyAlternate)
	expect(t, alt != nil, "t.screens.get(.alternate) != null")
	expect(t, alt != primary, "alt != primary")
	term.gpa() // Zig: t.gpa().ptr == orig.ptr
	// Zig-only: try std.testing.expect(t.gpa().ptr == orig.ptr);
	// Zig-only: alt.alloc = marker;
	// Zig-only: try std.testing.expect(t.gpa().ptr == marker.ptr);
	// The inactive primary screen is not consulted.
	// Zig-only: primary.alloc = other;
	// Zig-only: try std.testing.expect(t.gpa().ptr == marker.ptr);
	// Zig-only: primary.alloc = orig;
	// Zig-only: alt.alloc = orig;

	// Back on the primary screen.
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	term.gpa() // Zig: t.gpa().ptr == orig.ptr
	// Zig-only: try std.testing.expect(t.gpa().ptr == orig.ptr);
	// Zig-only: alt.alloc = marker;
	// Zig-only: try std.testing.expect(t.gpa().ptr == orig.ptr);
	// Zig-only: alt.alloc = orig;
}

// Zig: test "term51: gpa: is the allocator of the streams made by vtStream"
func TestTma_term51_gpa_is_the_allocator_of_the_streams_made_by_vtStream(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: var fa = std.testing.FailingAllocator.init(std.testing.allocator, .{});
	// Zig-only: const alloc = fa.allocator();
	term := tmaInit(t, 4, 4)
	defer term.Deinit()

	st := term.VtStream()
	defer st.Deinit()
	// osc_parser.alloc is `any` in Go: nil is Zig's null, any other value is
	// "an allocator was provided".
	expect(t, st.parser.osc_parser.alloc, "st.parser.osc_parser.alloc != null")
	// Zig-only: try std.testing.expect(st.parser.osc_parser.alloc.?.ptr == t.gpa().ptr);
	// Zig-only: try std.testing.expect(st.parser.osc_parser.alloc.?.ptr == alloc.ptr);
}

// Zig: test "term51: vtHandler: handlers of one terminal are equal and share its state"
func TestTma_term51_vtHandler_handlers_of_one_terminal_are_equal_and_share_its_state(t *testing.T) {
	testStdCapacity(t)
	term := tmaInit(t, 10, 3)
	defer term.Deinit()

	// Creating a handler does not touch the terminal.
	cursor_before := term.Screens.Active.Cursor.X
	h1 := term.VtHandler()
	defer h1.Deinit()
	h2 := term.VtHandler()
	defer h2.Deinit()
	expectEqual(t, cursor_before, term.Screens.Active.Cursor.X)
	tmaTerminalExpectPlain(t, term, "")

	expect(t, h1.terminal == term, "h1.terminal == &t")
	expect(t, h2.terminal == term, "h2.terminal == &t")
	expect(t, h1 == h2, "std.meta.eql(h1, h2)")

	tmaPrint(t, &h1, 'a')
	tmaPrint(t, &h2, 'b')
	tmaPrint(t, &h1, 'c')
	tmaTerminalExpectPlain(t, term, "abc")
	expectEqual(t, SizeCellCountInt(3), term.Screens.Active.Cursor.X)
}

// Zig: test "term51: vtHandler: a copy of the handler works after the original was deinitialized"
func TestTma_term51_vtHandler_a_copy_of_the_handler_works_after_the_original_was_deinitialized(t *testing.T) {
	testStdCapacity(t)
	term := tmaInit(t, 10, 3)
	defer term.Deinit()

	h := term.VtHandler()
	tmaPrint(t, &h, 'x')
	cp := h
	h.Deinit()
	tmaPrint(t, &cp, 'y')
	cp.Deinit()

	// The terminal is still usable and a new handler continues.
	h2 := term.VtHandler()
	defer h2.Deinit()
	tmaPrint(t, &h2, 'z')
	tmaTerminalExpectPlain(t, term, "xyz")
}

// Zig: test "term51: vtHandler: follows screen switches and resizes of its terminal"
func TestTma_term51_vtHandler_follows_screen_switches_and_resizes_of_its_terminal(t *testing.T) {
	testStdCapacity(t)
	term := tmaInit(t, 10, 3)
	defer term.Deinit()
	h := term.VtHandler()
	defer h.Deinit()

	tmaPrint(t, &h, 'p')
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tmaPrint(t, &h, 'q')
	tmaTerminalExpectPlain(t, term, "q")
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	tmaTerminalExpectPlain(t, term, "p")

	zigTry(t, term.Resize(20, 5))
	tmaPrint(t, &h, 'r')
	tmaTerminalExpectPlain(t, term, "pr")
	expect(t, h.terminal == term, "h.terminal == &t")
}

// Zig: test "term51: vtHandler: a stream built over the handler drives the terminal"
func TestTma_term51_vtHandler_a_stream_built_over_the_handler_drives_the_terminal(t *testing.T) {
	testStdCapacity(t)
	term := tmaInit(t, 10, 3)
	defer term.Deinit()

	h := term.VtHandler()
	var st StreamReadonlyStream = StreamInit(&h)
	defer st.Deinit()
	expect(t, st.handler.terminal == term, "st.handler.terminal == &t")
	zigTry(t, st.NextSlice([]uint8("ab\r\ncd")))
	tmaTerminalExpectPlain(t, term, "ab\ncd")
	expectEqual(t, SizeCellCountInt(2), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(1), term.Screens.Active.Cursor.Y)
}

// Zig: test "term51: vtHandler: layout of the returned handler"
func TestTma_term51_vtHandler_layout_of_the_returned_handler(t *testing.T) {
	testStdCapacity(t)
	term := tmaInit(t, 4, 2)
	defer term.Deinit()
	h := term.VtHandler()
	info := reflect.TypeOf(h)
	expectEqual(t, reflect.Struct, info.Kind())
	expectEqual(t, 1, info.NumField())
	expectEqualStrings(t, "terminal", info.Field(0).Name)
	expectEqual(t, unsafe.Sizeof((*Terminal)(nil)), unsafe.Sizeof(h))
}
