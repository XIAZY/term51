// WIREFRAME for src/terminal/search/sliding_window.zig (Ghostty v1.3.1). NOT
// YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: `search/sliding_window.zig` is `search_sliding_window.go`,
// prefix `SearchSlidingWindow` (lowerCamel `searchSlidingWindow` for
// private declarations). The file's main type, `SlidingWindow`, has the name
// of the file and so is not doubled: `SearchSlidingWindow` (the same rule
// that names the main type of every search file by its bare file prefix;
// see search.go). What is nested in it takes that prefix. Its private
// declarations are `searchSlidingWindowDirection` (values
// `searchSlidingWindowDirectionForward`, `...Reverse`),
// `searchSlidingWindowDataBuf`, `searchSlidingWindowMetaBuf` and
// `searchSlidingWindowMeta`. `init` is `SearchSlidingWindowInit`. `pub fn`
// methods are exported (`Next`, `Append`, `ClearAndRetainCapacity`,
// `Meta.Deinit`); private methods keep their Zig name (`highlight`,
// `testChangeNeedle`, `assertIntegrity`).
//
// The file is not re-exported by search.zig ("Non-public APIs"), but
// `SlidingWindow` is `pub` in its file and so exported here.
//
// Declarations appear in the same order as in sliding_window.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory.
//
//   - The field `alloc: Allocator` is dropped, and with it every allocator
//     argument (`DataBuf.init`, `Meta.deinit`, ...). `Deinit` stubs are
//     kept; the tests call `w.Deinit()`.
//   - `DataBuf` is `CircBuf(u8, 0)`: `DatastructCircBuf[uint8]`
//     (datastruct_circ_buf.go) made with default 0. `MetaBuf` is
//     `CircBuf(Meta, undefined)`: `DatastructCircBuf[searchSlidingWindowMeta]`
//     made with the zero Meta. Both are Go type aliases of the instantiated
//     generic type.
//   - `chunk_buf` is `std.MultiArrayList(FlattenedHighlight.Chunk)`: a
//     `[]HighlightFlattenedChunk`, as `HighlightFlattened.chunks` is
//     (highlight.go). `items(.node)[i]` is `chunk_buf[i].node`, `.len` is
//     `len(chunk_buf)`, `clearRetainingCapacity` is `chunk_buf[:0]`,
//     `ensureTotalCapacity` grows the slice's capacity. `highlight` ends with
//     `result.chunks = self.chunk_buf`, so a returned HighlightFlattened
//     shares the window's chunk slice exactly as in Zig: it is valid only
//     until the next `Next` or `Append`; callers that keep it `Clone` it.
//   - `needle` and `overlap_buf` are owned `[]uint8`; `alloc.dupe` is a Go
//     copy.
//   - `Meta.cell_map` is `std.ArrayList(point.Coordinate)`: a
//     `[]PointCoordinate`. `append` hands `&meta.cell_map` to the page
//     formatter as `FormatterPageFormatterPointMap{map_: &meta.cell_map}`
//     (formatter.go).
//   - `append` encodes into a `std.Io.Writer.Allocating`: the stand-in is
//     `oscWriterAllocating` (osc.go), as in formatter.go;
//     `encoded.written()` is its written bytes.
//   - Node pointers (`*PageList.List.Node`) are `*PageListNode`.
//   - `std.ascii.indexOfIgnoreCase` is not in the Go standard library with
//     Zig's semantics (ASCII-only case folding, first index); the
//     implementation writes it, ASCII only, not `strings.EqualFold`
//     (Unicode).
//   - `highlight`'s local anonymous struct (`tl`, with `br` and `prune`) is
//     a local Go type in the body; it is not declared here.
//
// # Widths
//
// `data_offset`, the `start_offset` and `len` parameters of `highlight` and
// the result of `Append` are counts (`usize`) and so `uint`. `Meta.serial`
// is `u64`, `uint64`.
//
// # Build options
//
// `std.debug.runtime_safety` (the asserts in `highlight` and the whole of
// `assertIntegrity`) is on in upstream's tests; the Go implementation keeps
// the checks unconditionally. The `std.valgrind.runningOnValgrind()` check
// has no Go counterpart.
//
// # Declarations that are not Zig declarations
//
//   - the `String` method of searchSlidingWindowDirection.
//
// # Omitted
//
// Imports and aliases with nothing to declare: `std`, `assert`,
// `Allocator`, `CircBuf`, `terminal`, `point`, `size`, `PageList`, `Pin`,
// `Selection`, `Screen`, `Terminal`, `PageFormatter` (use
// `FormatterPageFormatter`), `FlattenedHighlight` (use
// `HighlightFlattened`). No function is omitted. No C ABI declaration exists
// in this file.
//
// # Zig defaults a Go literal does not give
//
// `data_offset` defaults to 0, the Go zero value; no other field has a Zig
// default. A SearchSlidingWindow is made only by its `init`, which sets
// fields that are NOT the Go zero value: `data` and `meta` are
// `CircBuf.init(alloc, 0)`, which is a FULL zero-capacity buffer (the zero
// DatastructCircBuf is not full; see datastruct_circ_buf.go), and
// `overlap_buf` is `needle.len * 2` bytes. For `.reverse` the stored needle
// is reversed. So a translated test makes a window with
// SearchSlidingWindowInit, never with a literal.

package terminal

// SearchSlidingWindow is sliding_window.SlidingWindow.
//
// Searches page nodes via a sliding window. The sliding window maintains
// the invariant that data isn't pruned until (1) we've searched it and
// (2) we've accounted for overlaps across pages to fit the needle.
//
// The sliding window is first initialized empty. Pages are then appended
// in the order to search them. The sliding window supports both a forward
// and reverse order specified via `init`. The pages should be appended
// in the correct order matching the search direction.
//
// All appends grow the window. The window is only pruned when a search
// is done (positive or negative match) via `next()`.
//
// To avoid unnecessary memory growth, the recommended usage is to
// call `next()` until it returns null and then `append` the next page
// and repeat the process. This will always maintain the minimum
// required memory to search for the needle.
//
// The caller is responsible for providing the pages and ensuring they're
// in the proper order. The SlidingWindow itself doesn't own the pages, but
// it will contain pointers to them in order to return selections. If any
// pages become invalid, the caller should clear the sliding window and
// start over.
//
// Zig's first field, `alloc: Allocator` ("The allocator to use for all the
// data within this window."), is dropped.
type SearchSlidingWindow struct {
	// The data buffer is a circular buffer of u8 that contains the
	// encoded page text that we can use to search for the needle.
	data searchSlidingWindowDataBuf

	// The meta buffer is a circular buffer that contains the metadata
	// about the pages we're searching. This usually isn't that large
	// so callers must iterate through it to find the offset to map
	// data to meta.
	meta searchSlidingWindowMetaBuf

	// Buffer that can fit any amount of chunks necessary for next
	// to never fail allocation.
	//
	// Zig: std.MultiArrayList(FlattenedHighlight.Chunk). See the file
	// header.
	chunk_buf []HighlightFlattenedChunk

	// Offset into data for our current state. This handles the
	// situation where our search moved through meta[0] but didn't
	// do enough to prune it.
	//
	// Zig default: 0.
	data_offset uint

	// The needle we're searching for. Does own the memory.
	//
	// Zig: []const u8.
	needle []uint8

	// The search direction. If the direction is forward then pages should
	// be appended in forward linked list order from the PageList. If the
	// direction is reverse then pages should be appended in reverse order.
	//
	// This is important because in most cases, a reverse search is going
	// to be more desirable to search from the end of the active area
	// backwards so more recent data is found first. Supporting both is
	// trivial though and will let us do more complex optimizations in the
	// future (e.g. starting from the viewport and doing a forward/reverse
	// concurrently from that point).
	direction searchSlidingWindowDirection

	// A buffer to store the overlap search data. This is used to search
	// overlaps between pages where the match starts on one page and
	// ends on another. The length is always `needle.len * 2`.
	overlap_buf []uint8
}

// searchSlidingWindowDirection is SlidingWindow.Direction.
//
// Zig: enum { forward, reverse }, values numbered from zero in the order
// listed.
type searchSlidingWindowDirection uint8

const (
	searchSlidingWindowDirectionForward searchSlidingWindowDirection = iota
	searchSlidingWindowDirectionReverse
)

// String returns the Zig tag name (@tagName).
func (self searchSlidingWindowDirection) String() string { panic(notImplemented) }

// searchSlidingWindowDataBuf is SlidingWindow.DataBuf: CircBuf(u8, 0). Made
// with DatastructCircBufInit[uint8](0, size).
type searchSlidingWindowDataBuf = DatastructCircBuf[uint8]

// searchSlidingWindowMetaBuf is SlidingWindow.MetaBuf: CircBuf(Meta,
// undefined). Made with DatastructCircBufInit(searchSlidingWindowMeta{},
// size); Zig's `undefined` default is the Go zero value.
type searchSlidingWindowMetaBuf = DatastructCircBuf[searchSlidingWindowMeta]

// searchSlidingWindowMeta is SlidingWindow.Meta.
type searchSlidingWindowMeta struct {
	node *PageListNode
	// Zig: u64.
	serial uint64
	// Zig: std.ArrayList(point.Coordinate).
	cell_map []PointCoordinate
}

// Deinit is SlidingWindow.Meta.deinit.
//
// Zig: pub fn deinit(self: *Meta, alloc: Allocator) void. The allocator
// parameter is dropped. It only frees `cell_map`; the stub is kept for the
// callers.
func (self *searchSlidingWindowMeta) Deinit() { panic(notImplemented) }

// SearchSlidingWindowInit is SlidingWindow.init.
//
// Zig: pub fn init(alloc: Allocator, direction: Direction, needle_unowned:
// []const u8) Allocator.Error!SlidingWindow. The allocator parameter is
// dropped. The needle is copied (and reversed for `.reverse`). The error is
// ErrOutOfMemory (dcs.go).
func SearchSlidingWindowInit(
	direction searchSlidingWindowDirection,
	needle_unowned []uint8,
) (SearchSlidingWindow, error) {
	panic(notImplemented)
}

// Deinit is SlidingWindow.deinit.
//
// Kept as a stub: every test calls it. It deinits every Meta and the
// buffers.
func (self *SearchSlidingWindow) Deinit() { panic(notImplemented) }

// ClearAndRetainCapacity is SlidingWindow.clearAndRetainCapacity.
//
// Clear all data but retain allocated capacity.
func (self *SearchSlidingWindow) ClearAndRetainCapacity() { panic(notImplemented) }

// Next is SlidingWindow.next.
//
// Search the window for the next occurrence of the needle. As
// the window moves, the window will prune itself while maintaining
// the invariant that the window is always big enough to contain
// the needle.
//
// This returns a flattened highlight on a match. The
// flattened highlight requires allocation and is therefore more expensive
// than a normal selection, but it is more efficient to render since it
// has all the information without having to dereference pointers into
// the terminal state.
//
// The flattened highlight chunks reference internal memory for this
// sliding window and are only valid until the next call to `next()`
// or `append()`. If the caller wants to retain the flattened highlight
// then they should clone it.
//
// Zig: ?FlattenedHighlight.
func (self *SearchSlidingWindow) Next() Optional[HighlightFlattened] { panic(notImplemented) }

// highlight is SlidingWindow.highlight.
//
// Return a flattened highlight for the given start and length.
//
// The flattened highlight can be used to render the highlight
// in the most efficient way because it doesn't require a terminal
// lock to access terminal data to compare whether some viewport
// matches the highlight (because it doesn't need to traverse
// the page nodes).
//
// The start index is assumed to be relative to the offset. i.e.
// index zero is actually at `self.data[self.data_offset]`. The
// selection will account for the offset.
func (self *SearchSlidingWindow) highlight(start_offset uint, len uint) HighlightFlattened {
	panic(notImplemented)
}

// Append is SlidingWindow.append.
//
// Add a new node to the sliding window. This will always grow
// the sliding window; data isn't pruned until it is consumed
// via a search (via next()).
//
// Returns the number of bytes of content added to the sliding window.
// The total bytes will be larger since this omits metadata, but it is
// an accurate measure of the text content size added.
//
// Zig: Allocator.Error!usize. The error is ErrOutOfMemory (dcs.go); Zig
// also maps any formatter or writer error to it.
func (self *SearchSlidingWindow) Append(node *PageListNode) (uint, error) { panic(notImplemented) }

// testChangeNeedle is SlidingWindow.testChangeNeedle.
//
// Only for tests!
//
// Zig: new is []const u8; asserts it has the needle's length. It does NOT
// reverse the new needle for a reverse window.
func (self *SearchSlidingWindow) testChangeNeedle(new []uint8) { panic(notImplemented) }

// assertIntegrity is SlidingWindow.assertIntegrity.
//
// Integrity check: the data length equals the sum of the metas' cell map
// lengths, and the data offset is within bounds. See "Build options" in
// the file header.
func (self *SearchSlidingWindow) assertIntegrity() { panic(notImplemented) }
