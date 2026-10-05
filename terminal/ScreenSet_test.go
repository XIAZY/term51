// Tests translated one for one from src/terminal/ScreenSet.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. Helpers are prefixed ss.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err); `x.?` is zigUnwrap or a nil check.
//   - `Screen.Options` literals are written with the Zig default of
//     kitty_image_storage_limit (320 * 1000 * 1000): ssSetOpts.
//   - Zig's `alloc` arguments are dropped; `testing.allocator` leak checks
//     are dropped with `// Zig-only:` comments.
//   - A pointer comparison of screens is a Go pointer comparison.
//   - `.{ .screen = .{} }` is Point{Tag: PointTagScreen}.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import "testing"

// ssExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func ssExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, got none")
		}
		if s, ok := r.(string); ok && s == notImplemented {
			t.Fatalf("expected a panic from the assertion, got: %v", r)
		}
	}()
	f()
}

// ssSetOpts is term51RSetOpts: `.{ .cols = cols, .rows = rows,
// .max_scrollback = 0 }` with the Zig default kitty_image_storage_limit.
func ssSetOpts(cols SizeCellCountInt, rows SizeCellCountInt) ScreenOptions {
	return ScreenOptions{
		cols:                      cols,
		rows:                      rows,
		max_scrollback:            0,
		kitty_image_storage_limit: 320 * 1000 * 1000,
	}
}

// ssSetInit is `try ScreenSet.init(alloc, opts)`; the set is returned by
// pointer (the methods take a pointer receiver).
func ssSetInit(t *testing.T, opts ScreenOptions) *ScreenSet {
	t.Helper()
	set, err := ScreenSetInit(opts)
	zigTry(t, err)
	return &set
}

// ssSetGetInit is `try set.getInit(alloc, key, opts)`.
func ssSetGetInit(t *testing.T, set *ScreenSet, key ScreenSetKey, opts ScreenOptions) *Screen {
	t.Helper()
	s, err := set.GetInit(key, opts)
	zigTry(t, err)
	return s
}

// ssScreenInit is `try Screen.init(alloc, .{ .cols = cols, .rows = rows,
// .max_scrollback = max_scrollback })` with the Zig default of
// kitty_image_storage_limit; the screen is returned by pointer.
func ssScreenInit(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Screen {
	t.Helper()
	s, err := ScreenInit(ScreenOptions{
		cols:                      cols,
		rows:                      rows,
		max_scrollback:            max_scrollback,
		kitty_image_storage_limit: 320 * 1000 * 1000,
	})
	zigTry(t, err)
	return &s
}

// ssActive, ssScreen, ssHistory and ssViewport are the Zig literals
// `.{ .active = .{ .x = x, .y = y } }` and so on.
func ssActive(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: x, y: y}}
}

func ssScreen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: x, y: y}}
}

func ssHistory(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{x: x, y: y}}
}

func ssViewport(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{x: x, y: y}}
}

// ssPin is `s.pages.pin(pt).?`.
func ssPin(t *testing.T, s *Screen, pt Point) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.pages.Pin(pt))
}

// ssYPin is term51YPin: the pin at a screen coordinate (history rows
// included).
func ssYPin(t *testing.T, s *Screen, x SizeCellCountInt, y uint32) PageListPin {
	t.Helper()
	return ssPin(t, s, ssScreen(x, y))
}

// ssYCoord is term51YCoord.
func ssYCoord(t *testing.T, s *Screen, p PageListPin) PointCoordinate {
	t.Helper()
	return zigUnwrap(t, s.pages.PointFromPin(PointTagScreen, p)).Screen
}

// ssYExpectPin is term51YExpectPin.
func ssYExpectPin(t *testing.T, s *Screen, p PageListPin, x SizeCellCountInt, y uint32) {
	t.Helper()
	expectEqual(t, PointCoordinate{x: x, y: y}, ssYCoord(t, s, p))
}

// ssYPages is term51YPages: a screen with exactly `pages` pages (200
// columns, 5 active rows). The earlier pages are full, the last page has 5
// rows.
func ssYPages(t *testing.T, pages uint) *Screen {
	t.Helper()
	s := ssScreenInit(t, 200, 5, 100*1024*1024)
	for s.pages.TotalPages() < pages {
		_, err := s.pages.Grow()
		zigTry(t, err)
	}
	for i := 0; i < 4; i++ {
		_, err := s.pages.Grow()
		zigTry(t, err)
	}
	return s
}

// ssYFirstRows is term51YFirstRows: the row count of the (full) first page.
func ssYFirstRows(s *Screen) uint32 {
	return uint32(s.pages.pages.first.data.size.rows)
}

// ssDumpScreen is `try s.dumpStringAlloc(alloc, .{ .screen = .{} })`.
func ssDumpScreen(t *testing.T, s *Screen) string {
	t.Helper()
	out, err := s.DumpStringAlloc(Point{Tag: PointTagScreen})
	zigTry(t, err)
	return string(out)
}

// Zig: test ScreenSet
func TestScreenSet_ScreenSet(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	set := ssSetInit(t, ScreenOptionsDefault)
	defer set.Deinit()
	expectEqual(t, ScreenSetKeyPrimary, set.active_key)

	// Initialize a secondary screen
	_ = ssSetGetInit(t, set, ScreenSetKeyAlternate, ScreenOptionsDefault)
	set.SwitchTo(ScreenSetKeyAlternate)
	expectEqual(t, ScreenSetKeyAlternate, set.active_key)
}

// Zig: test "term51: init: only the primary screen exists and is active"
func TestScreenSet_term51_init_only_the_primary_screen_exists_and_is_active(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	expectEqual(t, ScreenSetKeyPrimary, set.active_key)
	expect(t, set.Get(ScreenSetKeyPrimary) != nil, "set.get(.primary) != null")
	expect(t, set.active == set.Get(ScreenSetKeyPrimary), "set.active == set.get(.primary).?")
	expect(t, set.Get(ScreenSetKeyAlternate) == nil, "set.get(.alternate) == null")
	expectEqual(t, uint(1), set.all.Count())
}

// Zig: test "term51: init: screen uses the given options"
func TestScreenSet_term51_init_screen_uses_the_given_options(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	set := ssSetInit(t, ScreenOptions{cols: 17, rows: 9, max_scrollback: 0, kitty_image_storage_limit: 320 * 1000 * 1000})
	defer set.Deinit()

	expectEqual(t, SizeCellCountInt(17), set.active.pages.cols)
	expectEqual(t, SizeCellCountInt(9), set.active.pages.rows)
	// Zig-only: try std.testing.expect(set.active.alloc.ptr == alloc.ptr);
	expect(t, !set.active.selection.Valid, "set.active.selection == null")
	expectEqual(t, SizeCellCountInt(0), set.active.cursor.x)
	expectEqual(t, SizeCellCountInt(0), set.active.cursor.y)

	// The default options work as well.
	def := ssSetInit(t, ScreenOptionsDefault)
	defer def.Deinit()
	expectEqual(t, SizeCellCountInt(80), def.active.pages.cols)
	expectEqual(t, SizeCellCountInt(24), def.active.pages.rows)
}

// Zig: test "term51: deinit: frees every initialized screen"
func TestScreenSet_term51_deinit_frees_every_initialized_screen(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;

	// Only the primary.
	{
		set := ssSetInit(t, ssSetOpts(10, 5))
		set.Deinit()
	}

	// Primary and alternate; the testing allocator reports any leak.
	{
		set := ssSetInit(t, ssSetOpts(10, 5))
		alt := ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(10, 5))
		zigTry(t, alt.TestWriteString([]uint8("content")))
		zigTry(t, set.Get(ScreenSetKeyPrimary).TestWriteString([]uint8("primary content")))
		set.Deinit()
	}

	// Alternate created and then removed before the set is dropped.
	{
		set := ssSetInit(t, ssSetOpts(10, 5))
		_ = ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(10, 5))
		set.Remove(ScreenSetKeyAlternate)
		set.Deinit()
	}
}

// Zig: test "term51: get: reports initialized screens only"
func TestScreenSet_term51_get_reports_initialized_screens_only(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	expect(t, set.Get(ScreenSetKeyPrimary) != nil, "set.get(.primary) != null")
	expect(t, set.Get(ScreenSetKeyAlternate) == nil, "set.get(.alternate) == null")

	alt := ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(10, 5))
	expect(t, set.Get(ScreenSetKeyAlternate) == alt, "set.get(.alternate) == alt")
	expect(t, set.Get(ScreenSetKeyPrimary) != alt, "set.get(.primary) != alt")

	// Works through a const pointer.
	const_set := set
	expect(t, const_set.Get(ScreenSetKeyAlternate) == alt, "const_set.get(.alternate) == alt")
	expect(t, const_set.Get(ScreenSetKeyPrimary) == set.active, "const_set.get(.primary) == set.active")

	set.Remove(ScreenSetKeyAlternate)
	expect(t, set.Get(ScreenSetKeyAlternate) == nil, "set.get(.alternate) == null")
	expect(t, set.Get(ScreenSetKeyPrimary) != nil, "set.get(.primary) != null")
}

// Zig: test "term51: getInit: creates the screen with the given options"
func TestScreenSet_term51_getInit_creates_the_screen_with_the_given_options(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	alt := ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(4, 3))
	expectEqual(t, SizeCellCountInt(4), alt.pages.cols)
	expectEqual(t, SizeCellCountInt(3), alt.pages.rows)
	expect(t, alt != set.Get(ScreenSetKeyPrimary), "alt != set.get(.primary).?")
	expectEqual(t, uint(2), set.all.Count())

	// The primary is untouched and the active screen does not change.
	expectEqual(t, SizeCellCountInt(10), set.Get(ScreenSetKeyPrimary).pages.cols)
	expectEqual(t, ScreenSetKeyPrimary, set.active_key)
	expect(t, set.active == set.Get(ScreenSetKeyPrimary), "set.active == set.get(.primary).?")
}

// Zig: test "term51: getInit: existing screens are returned and options ignored"
func TestScreenSet_term51_getInit_existing_screens_are_returned_and_options_ignored(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	// The primary already exists: its options are not replaced.
	primary := ssSetGetInit(t, set, ScreenSetKeyPrimary, ssSetOpts(3, 2))
	expect(t, primary == set.Get(ScreenSetKeyPrimary), "primary == set.get(.primary).?")
	expectEqual(t, SizeCellCountInt(10), primary.pages.cols)

	alt := ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(4, 3))
	zigTry(t, alt.TestWriteString([]uint8("keep")))
	again := ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(8, 8))
	expect(t, again == alt, "again == alt")
	expectEqual(t, SizeCellCountInt(4), again.pages.cols)
	expectEqual(t, SizeCellCountInt(3), again.pages.rows)
	expectEqual(t, uint(2), set.all.Count())

	// Content survives the repeated call.
	out := ssDumpScreen(t, again)
	// Zig-only: defer alloc.free(out);
	expectEqualStrings(t, "keep", out)
}

// Zig: test "term51: remove: removes and frees the alternate screen"
func TestScreenSet_term51_remove_removes_and_frees_the_alternate_screen(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	alt := ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(4, 3))
	zigTry(t, alt.TestWriteString([]uint8("gone")))
	expectEqual(t, uint(2), set.all.Count())

	set.Remove(ScreenSetKeyAlternate)
	expect(t, set.Get(ScreenSetKeyAlternate) == nil, "set.get(.alternate) == null")
	expectEqual(t, uint(1), set.all.Count())
	expect(t, set.Get(ScreenSetKeyPrimary) != nil, "set.get(.primary) != null")
	expectEqual(t, ScreenSetKeyPrimary, set.active_key)
	expect(t, set.active == set.Get(ScreenSetKeyPrimary), "set.active == set.get(.primary).?")

	// A fresh alternate is a new empty screen.
	alt2 := ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(6, 2))
	expectEqual(t, SizeCellCountInt(6), alt2.pages.cols)
	out := ssDumpScreen(t, alt2)
	// Zig-only: defer alloc.free(out);
	expectEqualStrings(t, "", out)
}

// Zig: test "term51: remove: an uninitialized screen is a no-op"
func TestScreenSet_term51_remove_an_uninitialized_screen_is_a_no_op(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	set.Remove(ScreenSetKeyAlternate)
	set.Remove(ScreenSetKeyAlternate)
	expect(t, set.Get(ScreenSetKeyAlternate) == nil, "set.get(.alternate) == null")
	expectEqual(t, uint(1), set.all.Count())
	expect(t, set.Get(ScreenSetKeyPrimary) != nil, "set.get(.primary) != null")
}

// Zig: test "term51: remove: does not update the active screen"
func TestScreenSet_term51_remove_does_not_update_the_active_screen(t *testing.T) {
	testStdCapacity(t)
	// Upstream behaviour, pinned: removing the active (alternate) screen
	// leaves active_key/active pointing at the freed screen. Callers must
	// switch back first. The dangling pointer is only compared, never used.
	// Zig-only: const alloc = std.testing.allocator;
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	alt := ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(4, 3))
	set.SwitchTo(ScreenSetKeyAlternate)
	set.Remove(ScreenSetKeyAlternate)

	expectEqual(t, ScreenSetKeyAlternate, set.active_key)
	expect(t, set.active == alt, "set.active == alt")
	expect(t, set.Get(ScreenSetKeyAlternate) == nil, "set.get(.alternate) == null")

	// Switching back repairs the set.
	set.SwitchTo(ScreenSetKeyPrimary)
	expect(t, set.active == set.Get(ScreenSetKeyPrimary), "set.active == set.get(.primary).?")
}

// Zig: test "term51: switchTo: switches the active screen and key"
func TestScreenSet_term51_switchTo_switches_the_active_screen_and_key(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	primary := set.Get(ScreenSetKeyPrimary)
	expect(t, primary != nil, "set.get(.primary).?")
	alt := ssSetGetInit(t, set, ScreenSetKeyAlternate, ssSetOpts(4, 3))

	set.SwitchTo(ScreenSetKeyAlternate)
	expectEqual(t, ScreenSetKeyAlternate, set.active_key)
	expect(t, set.active == alt, "set.active == alt")

	// Switching to the already active screen is idempotent.
	set.SwitchTo(ScreenSetKeyAlternate)
	expectEqual(t, ScreenSetKeyAlternate, set.active_key)
	expect(t, set.active == alt, "set.active == alt")

	set.SwitchTo(ScreenSetKeyPrimary)
	expectEqual(t, ScreenSetKeyPrimary, set.active_key)
	expect(t, set.active == primary, "set.active == primary")
	set.SwitchTo(ScreenSetKeyPrimary)
	expect(t, set.active == primary, "set.active == primary")

	// Each screen keeps its own content while switching.
	zigTry(t, set.active.TestWriteString([]uint8("one")))
	set.SwitchTo(ScreenSetKeyAlternate)
	zigTry(t, set.active.TestWriteString([]uint8("two")))
	set.SwitchTo(ScreenSetKeyPrimary)
	{
		out := ssDumpScreen(t, set.active)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "one", out)
	}
	set.SwitchTo(ScreenSetKeyAlternate)
	{
		out := ssDumpScreen(t, set.active)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "two", out)
	}
}

// Go-only: panic test: ScreenSet.remove: the primary screen cannot be removed (assert(key != .primary))
func TestScreenSet_panic_remove_primary(t *testing.T) {
	testStdCapacity(t)
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	ssExpectPanic(t, func() { set.Remove(ScreenSetKeyPrimary) })
}

// Go-only: panic test: ScreenSet.switchTo: the screen is not initialized (`self.all.get(key).?` on null)
func TestScreenSet_panic_switchTo_uninitialized(t *testing.T) {
	testStdCapacity(t)
	set := ssSetInit(t, ssSetOpts(10, 5))
	defer set.Deinit()

	ssExpectPanic(t, func() { set.SwitchTo(ScreenSetKeyAlternate) })
}
