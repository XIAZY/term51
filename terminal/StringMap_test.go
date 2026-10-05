// Tests translated one for one from src/terminal/StringMap.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. Helpers are prefixed ss (shared
// with ScreenSet_test.go).
//
// Translation notes:
//   - The three upstream tests and five term51 tests that concern the
//     oniguruma search iterator are not translated (the library build has
//     oniguruma off and Go has no binding): scripts/not-translated.txt.
//   - `[:0]const u8` strings: a Go slice whose backing array has the
//     terminating zero after len (ssDupeZ), so that `string[len]` is
//     `string[:len+1][len]`.
//   - `.ptr != .ptr` comparisons of two strings are comparisons of the
//     addresses of their first elements.
//   - `defer alloc.free(...)` and the testing allocator are dropped with
//     `// Zig-only:` comments.

package terminal

import "testing"

// ssDupeZ is `try alloc.dupeZ(u8, s)`: a copy with a zero byte after the
// last element, within the capacity of the slice.
func ssDupeZ(s string) []uint8 {
	b := append([]uint8(s), 0)
	return b[:len(s)]
}

// Zig: test "term51: deinit: frees the string and the map"
func TestStringMap_term51_deinit_frees_the_string_and_the_map(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := ssScreenInit(t, 10, 5, 0)
	defer s.Deinit()

	pin := ssPin(t, s, ssScreen(3, 1))
	str := ssDupeZ("hello")
	m := make([]PageListPin, 5)
	for i := range m {
		m[i] = pin
	}

	sm := StringMap{string: str, map_: m}
	expectEqual(t, 5, len(sm.string))
	expectEqual(t, uint8(0), sm.string[:len(sm.string)+1][len(sm.string)])
	expectEqual(t, 5, len(sm.map_))
	// The testing allocator reports a leak if either is not freed.
	sm.Deinit()
}

// Zig: test "term51: deinit: empty string and empty map"
func TestStringMap_term51_deinit_empty_string_and_empty_map(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	sm := StringMap{
		string: ssDupeZ(""),
		map_:   make([]PageListPin, 0),
	}
	expectEqual(t, 0, len(sm.string))
	expectEqual(t, 0, len(sm.map_))
	sm.Deinit()
}

// Zig: test "term51: deinit: map produced by Screen.selectionString"
func TestStringMap_term51_deinit_map_produced_by_Screen_selectionString(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := ssScreenInit(t, 10, 5, 0)
	defer s.Deinit()
	// 'a' (1 byte), a wide hiragana A (3 bytes, cells 1-2), 'b' (cell 3).
	zigTry(t, s.TestWriteString([]uint8("aあb")))

	sel := SelectionInit(
		ssPin(t, s, ssScreen(0, 0)),
		ssPin(t, s, ssScreen(3, 0)),
		false,
	)
	var m StringMap
	text, err := s.SelectionString(ScreenSelectionString{sel: sel, trim: false, map_: &m})
	zigTry(t, err)
	// Zig-only: defer alloc.free(text);
	defer m.Deinit()

	expectEqualStrings(t, "aあb", string(text))
	// The map owns a separate copy of the string, with a sentinel.
	expectEqualStrings(t, "aあb", string(m.string))
	expect(t, &m.string[0] != &text[0], "map.string.ptr != text.ptr")
	expectEqual(t, uint8(0), m.string[:len(m.string)+1][len(m.string)])
	// One pin per byte.
	expectEqual(t, len(m.string), len(m.map_))
	expectEqual(t, 5, len(m.map_))
	expected_x := [...]SizeCellCountInt{0, 1, 1, 1, 3}
	for i, p := range m.map_ {
		x := expected_x[i]
		expectEqual(t,
			ssScreen(x, 0),
			zigUnwrap(t, s.pages.PointFromPin(PointTagScreen, p)),
		)
	}
}
