// WIREFRAME for src/terminal/size.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types and constants only. size.zig has no function
// that survives the rules below, so there is nothing to implement here
// later; the header keeps the wireframe wording so the file is reviewed with
// the others. See the header of color.go for the rules.
//
// File naming: size.zig is a namespace file, so its declarations take the
// prefix `Size`: `size.CellCountInt` is `SizeCellCountInt`.
//
// Declarations appear in the same order as in size.zig.
//
// Omitted under the Go-memory rule (docs/mapping.md, governing rule 3: the
// port uses Go slices, pointers and the garbage collector; functions and
// types that exist only to compute byte layouts, map memory or hand it back
// have no Go counterpart):
//   - `Offset(T)` (a packed struct over OffsetInt holding a byte offset from
//     a page's base address), its field `offset`, `Offset(T).Slice` with its
//     fields `offset` and `len`, `Offset(T).Slice.slice` and `Offset(T).ptr`.
//     Go code holds a slice or a pointer where Zig holds an Offset.
//   - `OffsetBuf`, its fields `base` and `offset`, and `OffsetBuf.init`,
//     `OffsetBuf.initOffset`, `OffsetBuf.start`, `OffsetBuf.member`,
//     `OffsetBuf.add`, `OffsetBuf.rebase`.
//   - `getOffset`.
//   - `intFromBase`.
//
// Kept although they describe bytes:
//   - `OffsetInt`: hyperlink.zig and Screen.zig use it as the integer type
//     of the implicit hyperlink id counter (`implicit: size.OffsetInt`,
//     `hyperlink_implicit_id: size.OffsetInt`), which is not layout.
//   - `max_page_size`: OffsetInt is defined from it, and PageList.zig
//     compares page byte sizes against it (lines 298, 445, 3287, 3370); how
//     that limit is expressed as a count is decided with the page wireframe.
//
// The file has no C ABI declarations.

package terminal

// SizeMaxPageSize is size.max_page_size: the maximum size of a page in
// bytes. We use a u16 here because any smaller bit size by Zig is upgraded
// anyways to a u16 on mainstream CPU architectures, and because 65KB is a
// reasonable page size. To support better configurability, we derive
// everything from this.
//
// Zig: std.math.maxInt(u32), a comptime_int. In Go it is an untyped
// constant, so it does not overflow `uint` on a 32-bit build.
const SizeMaxPageSize = 1<<32 - 1

// SizeOffsetInt is size.OffsetInt: the int type that can contain the maximum
// memory offset in bytes, derived from the maximum terminal page size.
//
// Zig: std.math.IntFittingRange(0, max_page_size - 1), which is u32.
type SizeOffsetInt = uint32

// Int types for maximum values of things. A lot of these sizes are
// based on "X is enough for any reasonable use case" principles.
// The goal is that a user can have the maxInt amount of all of these
// present at one time and be able to address them in a single Page.zig.

// SizeCellCountInt is size.CellCountInt.
//
// Total number of cells that are possible in each dimension (row/col).
// Based on 2^16 being enough for any reasonable terminal size and allowing
// IDs to remain 16-bit.
type SizeCellCountInt = uint16

// SizeStyleCountInt is size.StyleCountInt, and SizeHyperlinkCountInt is
// size.HyperlinkCountInt.
//
// Total number of styles and hyperlinks that are possible in a page.
// We match CellCountInt here because each cell in a single row can have at
// most one style, making it simple to split a page by splitting rows.
//
// Note due to the way RefCountedSet works, we are short one value, but
// this is a theoretical limit we accept. A page with a single row max
// columns wide would be one short of having every cell have a unique style.
type SizeStyleCountInt = SizeCellCountInt
type SizeHyperlinkCountInt = SizeCellCountInt

// SizeGraphemeBytesInt is size.GraphemeBytesInt, and SizeStringBytesInt is
// size.StringBytesInt.
//
// Total number of bytes that can be taken up by grapheme data and string
// data. Both of these technically unlimited with malicious input, but
// we choose a reasonable limit of 2^32 (4GB) per.
type SizeGraphemeBytesInt = uint32
type SizeStringBytesInt = uint32
