// WIREFRAME for src/terminal/StringMap.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// A string along with the mapping of each individual byte in the string
// to the point in the screen.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: StringMap.zig is a file-level struct, so the type is
// `StringMap`. It has no `init`: Screen.selectionString fills one in
// through a pointer (`Screen.SelectionString.map`).
//
// Declarations appear in the same order as in StringMap.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory. `string` is
// `[:0]const u8` and is a `[]uint8`; `map` is `[]Pin` and is a
// `[]PageListPin`, one pin per byte of `string`. `map` is a Go keyword, so
// the field is `map_`. `deinit` only frees; the stub is kept because the
// tests call it.
//
// # Omitted
//
// Compiled out (`build_options.oniguruma` is false in the library build,
// src/build/GhosttyZig.zig; with it false `searchIterator` is the type
// `void` and nothing below it is referenced), and dependent on the
// oniguruma regular-expression binding, which is outside src/terminal and
// has no Go counterpart (standard library only, governing rule 6):
//   - `oni_search_retry_limit` (line 18);
//   - `searchIterator` (line 30) and `searchIteratorOni` (line 35), which
//     take an `oni.Regex`;
//   - `SearchIterator` (line 43) with `next` (line 50): fields `map`,
//     `regex: oni.Regex`, `offset`;
//   - `Match` (line 94) with `deinit` (line 99) and `selection` (line 104):
//     fields `map`, `offset`, `region: oni.Region`.
// The three upstream tests of this file all start with
// `if (comptime !build_options.oniguruma) return error.SkipZigTest;`.
// Aliases and imports with nothing to declare: `point`, `Selection`,
// `Screen`, `Pin`, `Allocator`, `build_options`, `oni`.

package terminal

// StringMap is the file-level struct of StringMap.zig.
type StringMap struct {
	// Zig: [:0]const u8.
	//
	// Convention for Zig's sentinel-terminated `[:0]const u8`: the Go slice
	// has len = the string length (the 0 is not counted, as in Zig), and its
	// backing array holds a 0 byte at index len, so cap > len. Tests may
	// check the sentinel with `s[:len(s)+1][len(s)] == 0`.
	string []uint8
	// Zig: `map: []Pin`. `map` is a Go keyword.
	map_ []PageListPin
}

// Deinit is StringMap.deinit.
//
// Zig: pub fn deinit(self: StringMap, alloc: Allocator) void. The allocator
// parameter is dropped. Kept only because the tests call it.
func (self StringMap) Deinit() { panic(notImplemented) }
