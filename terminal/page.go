// WIREFRAME for src/terminal/page.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: page.zig is a namespace file, so its declarations take the
// prefix `Page`: `page.Cell` is `PageCell`, `page.Row` is `PageRow`,
// `page.Capacity` is `PageCapacity`, `page.Size` is `PageSize`,
// `page.std_capacity` is `PageStdCapacity`. The struct `page.Page` itself is
// `Page` (not `PagePage`), and what is nested in it follows from that:
// `Page.Layout` is `PageLayout`, `Page.init` is `PageInit`. Private
// file-level constants and types take the lowerCamel prefix:
// `pageGraphemeChunkLen`, `pageGraphemeAlloc`.
//
// `Page.layout` (function) and `Page.Layout` (type) would have the same Go
// name; the type is `PageLayout` and the function is `PageLayoutOf`.
//
// Declarations appear in the same order as in page.zig.
//
// # Go-memory rule, option A
//
// (docs/mapping.md, governing rule 3, as refined for this batch.) The port
// uses ordinary Go memory: slices, pointers, the garbage collector; no
// `unsafe`, no raw byte block. But it keeps upstream's bookkeeping exactly,
// because the scrollback limit is counted in bytes of page memory and a page
// grows when one of its internal stores fills, so fill points and byte
// sizes are visible behaviour. Concretely:
//
//   - `Page.memory` is dropped. `Page.rows: Offset(Row)` is `rows []PageRow`
//     (`capacity.rows` long) and `Page.cells: Offset(Cell)` is
//     `cells []PageCell`, one contiguous slice of
//     `capacity.rows * capacity.cols`; row r's cells are
//     `cells[r*capacity.cols : (r+1)*capacity.cols]`.
//   - `string_alloc`, `grapheme_alloc`, `grapheme_map`, `styles`,
//     `hyperlink_map` and `hyperlink_set` own Go slices; see
//     bitmap_allocator.go, hash_map.go, ref_counted_set.go.
//   - Byte-size ARITHMETIC is kept as plain numbers: `Page.Layout` keeps
//     every field, `Page.layout`, `Capacity.adjust`, `Capacity.maxCols` and
//     `Capacity.availableBitsForGrid` are kept, and the `@sizeOf`-based
//     constants keep Zig's values. @sizeOf(Row) and @sizeOf(Cell) are 8,
//     @sizeOf(u21) is 4. `std.heap.page_size_min`, to which `total_size` is
//     rounded up, is 4096 in the recorded layouts.
//   - A cell is identified in the two maps by the byte offset upstream
//     would have used: `layout.cells_start + index*8`, where index is the
//     cell's position in `cells`. That number is also what `Row.cells`
//     holds. It must stay that number, not the index, so that hash
//     placement in the maps (and so the point where a map reports that it
//     is full) matches upstream.
//   - Every `base` / `self.memory` argument is dropped; allocator
//     parameters are dropped.
//
// Omitted under the Go-memory rule, because they exist only to place a page
// in caller-supplied raw memory:
//   - `Page.initBuf(buf: OffsetBuf, l: Layout) Page`;
//   - `Page.cloneBuf(self: *const Page, buf: []align(page_size_min) u8) Page`.
//
// `Page.init`, `Page.reinit` and `Page.clone` call them in Zig; the Go
// implementations do that work themselves. `Page.deinit` only unmaps the
// memory; it is kept as a stub because every test calls it.
//
// # Pointer to index
//
// Zig turns a `*Cell` into its byte offset by pointer subtraction
// (`getOffset(Cell, self.memory, cell)`) in swapCells, lookupHyperlink,
// clearHyperlink, setHyperlink, moveHyperlink, setGraphemes,
// appendGrapheme, lookupGrapheme, moveGrapheme and clearGrapheme, and
// compares a `*Row` with the rows' addresses in getCells. The signatures
// keep the pointers, as in Zig. Go has no pointer subtraction, so three
// stubs that are not Zig declarations stand for it: `Page.cellIndex`,
// `Page.rowIndex` and `Page.cellOffset`. They are declared after
// `Page.isDirty`.
//
// # Widths
//
// Every byte size and byte offset is a `uint64`, whatever the platform word
// size, because the reference is the recorded 64-bit data and the target is
// 32-bit (with a maxed capacity the byte arithmetic exceeds 32 bits). Here:
// every `*_start`, `*_size` and `total_size` field of PageLayout and of the
// layouts nested in it, the result of availableBitsForGrid, and the
// `...Sizes` values. Counts, indexes and lengths (rows, columns, x and y,
// numbers of cells, graphemes and hyperlinks) stay `uint` or their fixed
// Zig width. PageCapacity's fields keep their Zig widths (u16 for cols,
// rows, styles and hyperlink_bytes; u32 for grapheme_bytes and
// string_bytes). The byte constants (pageGraphemeChunk and the like) are
// untyped Go constants and take the type of the expression they are used
// in.
//
// # Declarations that are not Zig declarations
//
//   - `pageGraphemeMapEntrySizes`, `styleSetItemSizes`,
//     `hyperlinkSetItemSizes`: sizes and alignments Zig reads from types at
//     compile time;
//   - `pageSlowRuntimeSafety`: the build option of that name;
//   - `PageCapacityInit`: a `Capacity` literal with its declared defaults;
//   - `Page.cellIndex`, `Page.rowIndex`, `Page.cellOffset`: pointer
//     subtraction;
//   - `PageCell.bitCast`: `@bitCast` of a cell.
//
// Build options:
//   - `build_options.kitty_graphics` is false in the library build. The
//     four `if (comptime build_options.kitty_graphics)` blocks (in
//     clonePartialRowFrom, moveCells twice, clearCells) are compiled out and
//     are not ported. The `Row.kitty_virtual_placeholder` bit stays, as
//     upstream says.
//   - `build_options.slow_runtime_safety` is the constant
//     `pageSlowRuntimeSafety`. `pause_integrity_checks` is `usize` when it
//     is set and `void` otherwise; the field is kept as a `uint`.
//
// Not ported:
//   - the `comptime` block at the start of `Page` (page-size alignment
//     assertion): a compile-time layout check;
//   - the unused imports and the `log` scope.
//
// Packed structs: `Row` and `Cell` are `packed struct(u64)`. Each is an
// ordinary struct with the same field names; the bit positions are given on
// the type. Cell's conversion to its integer, which `Cell.isZero` performs
// with `@bitCast`, is the stub `PageCell.bitCast`.
//
// The file has no C ABI declarations.

package terminal

import "errors"

// pageSlowRuntimeSafety is NOT a declaration of page.zig: it is Zig's build
// option `build_options.slow_runtime_safety`, which page.zig tests in
// pauseIntegrityChecks, assertIntegrity, verifyIntegrity, clonePartialRowFrom,
// getCells, appendGrapheme, moveGrapheme and clearGrapheme. Integrity checks
// are on in tests, so it is true.
const pageSlowRuntimeSafety = true

// pageGraphemeChunkLen is page.grapheme_chunk_len.
//
// The allocator to use for multi-codepoint grapheme data. We use
// a chunk size of 4 codepoints. It'd be best to set this empirically
// but it is currently set based on vibes. My thinking around 4 codepoints
// is that most skin-tone emoji are <= 4 codepoints, letter combiners
// are usually <= 4 codepoints, and 4 codepoints is a nice power of two
// for alignment.
const pageGraphemeChunkLen = 4

// pageGraphemeChunk is page.grapheme_chunk:
// grapheme_chunk_len * @sizeOf(u21) = 4 * 4.
const pageGraphemeChunk = 16

// pageGraphemeAlloc is page.GraphemeAlloc: BitmapAllocator(grapheme_chunk).
//
// The chunk size (pageGraphemeChunk) is given to BitmapAllocatorInit; see
// bitmap_allocator.go. The element type is u21.
type pageGraphemeAlloc = BitmapAllocator[uint32]

// pageGraphemeCountDefault is page.grapheme_count_default:
// GraphemeAlloc.bitmap_bit_size.
const pageGraphemeCountDefault = 64

// PageGraphemeBytesDefault is page.grapheme_bytes_default:
// grapheme_count_default * grapheme_chunk = 64 * 16.
const PageGraphemeBytesDefault = 1024

// pageGraphemeMap is page.GraphemeMap.
//
// Zig: AutoOffsetHashMap(Offset(Cell), Offset(u21).Slice). The key is the
// cell's byte offset (see the file header); the value is the slice of extra
// codepoints, a sub-slice of the grapheme allocator's backing.
type pageGraphemeMap = HashMapAutoOffsetHashMap[SizeOffsetInt, []uint32]

// pageGraphemeMapEntrySizes is NOT a Zig declaration: @sizeOf and @alignOf
// of the key and value types of GraphemeMap, which
// HashMapOffsetHashMapLayoutOf needs (see hash_map.go). Key Offset(Cell): a
// packed struct(u32), 4 and 4. Value Offset(u21).Slice:
// { offset: u32, len: usize }, 16 and 8 on a 64-bit target. Confirmed by the
// recorded layouts (term51 data/layout/page_layout.txt: capacity=32
// keys_start=32 vals_start=160 total_size=688).
var pageGraphemeMapEntrySizes = hashMapEntrySizes{key_size: 4, key_align: 4, val_size: 16, val_align: 8}

// styleSetItemSizes is NOT a Zig declaration: @sizeOf and @alignOf of
// style.Set.Item, which RefCountedSetLayoutInit needs (see
// ref_counted_set.go). It is declared here, with the other sizes the page
// layout uses.
//
// Derived from the recorded layouts (term51 data/layout/page_layout.txt),
// in which `styles_layout` appears at every table capacity from 1 to 65536:
//   - size 36: (total_size - items_start) / cap is 36 in all of them, e.g.
//     cap=1 items_start=4 total_size=40, cap=104 items_start=256
//     total_size=4000, cap=53248 items_start=131072 total_size=2048000;
//   - alignment 4: with table_cap=1 the table ends at byte 2 and
//     items_start is 4, so the alignment is at least 4, and a size of 36 is
//     not a multiple of 8.
var styleSetItemSizes = refCountedSetItemSizes{item_size: 36, item_align: 4}

// hyperlinkSetItemSizes is NOT a Zig declaration: @sizeOf and @alignOf of
// hyperlink.Set.Item, which RefCountedSetLayoutInit needs.
//
//   - size 48: see hyperlinkSetItemSizeOf (hyperlink.go); in the recorded
//     layouts (total_size - items_start) / cap is 48 at every table capacity
//     from 2 to 2048, e.g. cap=1 items_start=8 total_size=56, cap=3
//     items_start=8 total_size=152;
//   - alignment 8: Item holds a usize (the `len` of an Offset(u8).Slice);
//     in the recorded layouts table_cap=1 has the table end at byte 2 and
//     items_start=8.
var hyperlinkSetItemSizes = refCountedSetItemSizes{item_size: hyperlinkSetItemSizeOf, item_align: 8}

// pageStringChunkLen is page.string_chunk_len.
//
// The allocator used for shared utf8-encoded strings within a page.
// Note the chunk size below is the minimum size of a single allocation
// and requires a single bit of metadata in our bitmap allocator. Therefore
// it should be tuned carefully (too small and we waste metadata, too large
// and we have fragmentation). We can probably use a better allocation
// strategy in the future.
//
// At the time of writing this, the strings table is only used for OSC8
// IDs and URIs. IDs are usually short and URIs are usually longer. I chose
// 32 bytes as a compromise between these two since it represents single
// domain links quite well and is not too wasteful for short IDs. We can
// continue to tune this as we see how it's used.
const pageStringChunkLen = 32

// pageStringChunk is page.string_chunk:
// string_chunk_len * @sizeOf(u8) = 32 * 1.
const pageStringChunk = 32

// pageStringAlloc is page.StringAlloc: BitmapAllocator(string_chunk).
//
// The chunk size (pageStringChunk) is given to BitmapAllocatorInit; see
// bitmap_allocator.go.
type pageStringAlloc = BitmapAllocator[uint8]

// pageStringCountDefault is page.string_count_default:
// StringAlloc.bitmap_bit_size.
const pageStringCountDefault = 64

// PageStringBytesDefault is page.string_bytes_default:
// string_count_default * string_chunk = 64 * 32.
const PageStringBytesDefault = 2048

// pageHyperlinkCountDefault is page.hyperlink_count_default.
//
// Default number of hyperlinks we support.
//
// The cell multiplier is the number of cells per hyperlink entry that
// we support. A hyperlink can be longer than this multiplier; the multiplier
// just sets the total capacity to simplify adjustable size metrics.
const pageHyperlinkCountDefault = 4

// pageHyperlinkBytesDefault is page.hyperlink_bytes_default:
// hyperlink_count_default * @sizeOf(hyperlink.Set.Item) = 4 * 48. See
// hyperlinkSetItemSizeOf (hyperlink.go) for the 48.
const pageHyperlinkBytesDefault = 192

// pageHyperlinkCellMultiplier is page.hyperlink_cell_multiplier.
const pageHyperlinkCellMultiplier = 16

// Page is page.Page.
//
// A page represents a specific section of terminal screen. The primary
// idea of a page is that it is a fully self-contained unit that can be
// serialized, copied, etc. as a convenient way to represent a section
// of the screen.
//
// This property is useful for renderers which want to copy just the pages
// for the visible portion of the screen, or for infinite scrollback where
// we may want to serialize and store pages that are sufficiently far
// away from the current viewport.
//
// Pages are always backed by a single contiguous block of memory that is
// aligned on a page boundary. This makes it easy and fast to copy pages
// around. Within the contiguous block of memory, the contents of a page are
// thoughtfully laid out to optimize primarily for terminal IO (VT streams)
// and to minimize memory usage.
//
// Zig's first field, `memory` (the backing memory for the page), is
// dropped; see the file header. The Go zero value is not an initialised
// page; use PageInit.
type Page struct {
	// The array of rows in the page. The rows are always in row order
	// (i.e. index 0 is the top row, index 1 is the row below that, etc.)
	//
	// Zig: Offset(Row). `capacity.rows` long.
	rows []PageRow

	// The array of cells in the page. The cells are NOT in row order,
	// but they are in column order. To determine the mapping of cells
	// to row, you must use the `rows` field. From the pointer to the
	// first column, all cells in that row are laid out in column order.
	//
	// Zig: Offset(Cell). `capacity.rows * capacity.cols` long; see the
	// file header.
	cells []PageCell

	// Set to true when an operation is performed that dirties all rows in
	// the page. See `Row.dirty` for more information on dirty tracking.
	//
	// NOTE: A value of false does NOT indicate that
	//       the page has no dirty rows in it, only
	//       that no full-page-dirtying operations
	//       have occurred since it was last cleared.
	dirty bool

	// The string allocator for this page used for shared utf-8 encoded
	// strings. Liveness of strings and memory management is deferred to
	// the individual use case.
	string_alloc pageStringAlloc

	// The multi-codepoint grapheme data for this page. This is where
	// any cell that has more than one codepoint will be stored. This is
	// relatively rare (typically only emoji) so this defaults to a very small
	// size and we force page realloc when it grows.
	grapheme_alloc pageGraphemeAlloc

	// The mapping of cell to grapheme data. The exact mapping is the
	// cell offset to the grapheme data offset. Therefore, whenever a
	// cell is moved (i.e. `erase`) then the grapheme data must be updated.
	// Grapheme data is relatively rare so this is considered a slow
	// path.
	//
	// The key must stay the byte offset upstream would have used
	// (`layout.cells_start + index*8`), so that hash placement matches.
	grapheme_map pageGraphemeMap

	// The available set of styles in use on this page.
	styles StyleSet

	// The structures used for tracking hyperlinks within the page.
	// The map maps cell offsets to hyperlink IDs and the IDs are in
	// the ref counted set. The strings within the hyperlink structures
	// are allocated in the string allocator.
	//
	// The map's key is the same byte offset as for grapheme_map.
	hyperlink_map HyperlinkMap
	hyperlink_set HyperlinkSet

	// The current dimensions of the page. The capacity may be larger
	// than this. This allows us to allocate a larger page than necessary
	// and also to resize a page smaller without reallocating.
	size PageSize

	// The capacity of this page. This is the full size of the backing
	// memory and is fixed at page creation time.
	capacity PageCapacity

	// If this is true then verifyIntegrity will do nothing. This is
	// only present with runtime safety enabled.
	//
	// Zig: usize (default 0) when build_options.slow_runtime_safety, void
	// otherwise. See the file header.
	pause_integrity_checks uint
}

// PageInit is page.Page.init.
//
// Initialize a new page, allocating the required backing memory.
// The size of the initialized page defaults to the full capacity.
//
// The backing memory is always allocated using mmap directly.
// You cannot use custom allocators with this structure because
// it is critical to performance that we use mmap.
//
// Zig: pub inline fn init(cap: Capacity) !Page. The error is mmap's.
func PageInit(cap PageCapacity) (Page, error) { panic(notImplemented) }

// page.Page.initBuf (page.zig line 191) is omitted; see the file header.

// Deinit is page.Page.deinit.
//
// Deinitialize the page, freeing any backing memory. Do NOT call
// this if you allocated the backing memory yourself (i.e. you used
// initBuf).
func (self *Page) Deinit() { panic(notImplemented) }

// Reinit is page.Page.reinit: reinitialize the page with the same capacity.
func (self *Page) Reinit() { panic(notImplemented) }

// Page.IntegrityError. An error set is not a type in Go; its members are
// the sentinels below.
var (
	ErrZeroRowCount              = errors.New("ZeroRowCount")
	ErrZeroColCount              = errors.New("ZeroColCount")
	ErrUnmarkedGraphemeRow       = errors.New("UnmarkedGraphemeRow")
	ErrMissingGraphemeData       = errors.New("MissingGraphemeData")
	ErrInvalidGraphemeCount      = errors.New("InvalidGraphemeCount")
	ErrUnmarkedGraphemeCell      = errors.New("UnmarkedGraphemeCell")
	ErrMissingStyle              = errors.New("MissingStyle")
	ErrUnmarkedStyleRow          = errors.New("UnmarkedStyleRow")
	ErrMismatchedStyleRef        = errors.New("MismatchedStyleRef")
	ErrInvalidStyleCount         = errors.New("InvalidStyleCount")
	ErrMissingHyperlinkData      = errors.New("MissingHyperlinkData")
	ErrMismatchedHyperlinkRef    = errors.New("MismatchedHyperlinkRef")
	ErrUnmarkedHyperlinkCell     = errors.New("UnmarkedHyperlinkCell")
	ErrUnmarkedHyperlinkRow      = errors.New("UnmarkedHyperlinkRow")
	ErrInvalidSpacerTailLocation = errors.New("InvalidSpacerTailLocation")
	ErrInvalidSpacerHeadLocation = errors.New("InvalidSpacerHeadLocation")
	ErrUnwrappedSpacerHead       = errors.New("UnwrappedSpacerHead")
)

// PauseIntegrityChecks is page.Page.pauseIntegrityChecks.
//
// Temporarily pause integrity checks. This is useful when you are
// doing a lot of operations that would trigger integrity check
// violations but you know the page will end up in a consistent state.
func (self *Page) PauseIntegrityChecks(v bool) { panic(notImplemented) }

// AssertIntegrity is page.Page.assertIntegrity.
//
// A helper that can be used to assert the integrity of the page
// when runtime safety is enabled. This is a no-op when runtime
// safety is disabled. This uses the libc allocator.
//
// Zig: self is *const Page.
func (self *Page) AssertIntegrity() { panic(notImplemented) }

// VerifyIntegrity is page.Page.verifyIntegrity.
//
// Verifies the integrity of the page data. This is not fast,
// but it is useful for assertions, deserialization, etc. The
// allocator is only used for temporary allocations -- all memory
// is freed before this function returns.
//
// Integrity errors are also logged as warnings.
//
// Zig: pub fn verifyIntegrity(self: *const Page, alloc_gpa: Allocator)
// !void. The allocator parameter is dropped. The error is one of the
// Page.IntegrityError sentinels.
func (self *Page) VerifyIntegrity() error { panic(notImplemented) }

// Clone is page.Page.clone.
//
// Clone the contents of this page. This will allocate new memory
// using the page allocator. If you want to manage memory manually,
// use cloneBuf.
//
// Zig: pub inline fn clone(self: *const Page) !Page. The error is mmap's.
func (self *Page) Clone() (Page, error) { panic(notImplemented) }

// page.Page.cloneBuf (page.zig line 596) is omitted; see the file header.

// Page.StyleSetError.
var (
	ErrStyleSetOutOfMemory = errors.New("StyleSetOutOfMemory")
	ErrStyleSetNeedsRehash = errors.New("StyleSetNeedsRehash")
)

// Page.HyperlinkError.
var (
	ErrStringAllocOutOfMemory  = errors.New("StringAllocOutOfMemory")
	ErrHyperlinkSetOutOfMemory = errors.New("HyperlinkSetOutOfMemory")
	ErrHyperlinkSetNeedsRehash = errors.New("HyperlinkSetNeedsRehash")
	ErrHyperlinkMapOutOfMemory = errors.New("HyperlinkMapOutOfMemory")
)

// Page.GraphemeError.
var (
	ErrGraphemeMapOutOfMemory   = errors.New("GraphemeMapOutOfMemory")
	ErrGraphemeAllocOutOfMemory = errors.New("GraphemeAllocOutOfMemory")
)

// Page.CloneFromError is StyleSetError || HyperlinkError || GraphemeError:
// the eight sentinels above.

// ExactRowCapacity is page.Page.exactRowCapacity.
//
// Compute the exact capacity required to store a range of rows from
// this page.
//
// The returned capacity will have the same number of columns as this
// page and the number of rows equal to the range given. The returned
// capacity is by definition strictly less than or equal to this
// page's capacity, so the layout is guaranteed to succeed.
//
// Preconditions:
//   - Range must be at least 1 row
//   - Start and end must be valid for this page
//
// Zig: self is *const Page.
func (self *Page) ExactRowCapacity(y_start uint, y_end uint) PageCapacity { panic(notImplemented) }

// CloneFrom is page.Page.cloneFrom.
//
// Clone the contents of another page into this page. The capacities
// can be different, but the size of the other page must fit into
// this page.
//
// The y_start and y_end parameters allow you to clone only a portion
// of the other page. This is useful for splitting a page into two
// or more pages.
//
// The column count of this page will always be the same as this page.
// If the other page has more columns, the extra columns will be
// truncated. If the other page has fewer columns, the extra columns
// will be zeroed.
//
// Zig: CloneFromError!void; other is *const Page.
func (self *Page) CloneFrom(other *Page, y_start uint, y_end uint) error { panic(notImplemented) }

// CloneRowFrom is page.Page.cloneRowFrom: clone a single row from another
// page into this page.
//
// Zig: CloneFromError!void; other is *const Page, src_row is *const Row.
func (self *Page) CloneRowFrom(other *Page, dst_row *PageRow, src_row *PageRow) error {
	panic(notImplemented)
}

// ClonePartialRowFrom is page.Page.clonePartialRowFrom: clone a single row
// from another page into this page, supporting partial copy. cloneRowFrom
// calls this.
//
// Zig: CloneFromError!void; other is *const Page, src_row is *const Row.
func (self *Page) ClonePartialRowFrom(
	other *Page,
	dst_row *PageRow,
	src_row *PageRow,
	x_start uint,
	x_end_req uint,
) error {
	panic(notImplemented)
}

// GetRow is page.Page.getRow: get a single row. y must be valid.
//
// Zig: self is *const Page.
func (self *Page) GetRow(y uint) *PageRow { panic(notImplemented) }

// GetCells is page.Page.getCells: get the cells for a row.
//
// Zig: self is *const Page. The result is `size.cols` long.
func (self *Page) GetCells(row *PageRow) []PageCell { panic(notImplemented) }

// GetRowAndCell is page.Page.getRowAndCell: get the row and cell for the
// given X/Y within this page.
//
// Zig: self is *const Page; the result is an anonymous struct.
func (self *Page) GetRowAndCell(x uint, y uint) struct {
	row  *PageRow
	cell *PageCell
} {
	panic(notImplemented)
}

// MoveCells is page.Page.moveCells.
//
// Move a cell from one location to another. This will replace the
// previous contents with a blank cell. Because this is a move, this
// doesn't allocate and can't fail.
func (self *Page) MoveCells(
	src_row *PageRow,
	src_left uint,
	dst_row *PageRow,
	dst_left uint,
	len uint,
) {
	panic(notImplemented)
}

// SwapCells is page.Page.swapCells: swap two cells within the same row as
// quickly as possible.
func (self *Page) SwapCells(src *PageCell, dst *PageCell) { panic(notImplemented) }

// ClearCells is page.Page.clearCells.
//
// Clear the cells in the given row. This will reclaim memory used
// by graphemes and styles. Note that if the style cleared is still
// active, Page cannot know this and it will still be ref counted down.
// The best solution for this is to artificially increment the ref count
// prior to calling this function.
func (self *Page) ClearCells(row *PageRow, left uint, end uint) { panic(notImplemented) }

// LookupHyperlink is page.Page.lookupHyperlink: returns the hyperlink ID
// for the given cell.
//
// Zig: self is *const Page, cell is *const Cell; returns ?hyperlink.Id.
func (self *Page) LookupHyperlink(cell *PageCell) Optional[HyperlinkId] { panic(notImplemented) }

// ClearHyperlink is page.Page.clearHyperlink.
//
// Clear the hyperlink from the given cell.
//
// In order to update the hyperlink flag on the row, call
// `updateRowHyperlinkFlag` after you finish clearing any
// hyperlinks in the row.
func (self *Page) ClearHyperlink(cell *PageCell) { panic(notImplemented) }

// UpdateRowHyperlinkFlag is page.Page.updateRowHyperlinkFlag.
//
// Checks if the row contains any hyperlinks and sets
// the hyperlink flag to false if none are found.
//
// Call after removing hyperlinks in a row.
func (self *Page) UpdateRowHyperlinkFlag(row *PageRow) { panic(notImplemented) }

// Page.InsertHyperlinkError.
var (
	// string_alloc errors
	ErrStringsOutOfMemory = errors.New("StringsOutOfMemory")

	// hyperlink_set errors
	ErrSetOutOfMemory = errors.New("SetOutOfMemory")
	ErrSetNeedsRehash = errors.New("SetNeedsRehash")
)

// InsertHyperlink is page.Page.insertHyperlink.
//
// Convert a hyperlink into a page entry, returning the ID.
//
// This does not de-dupe any strings, so if the URI, explicit ID,
// etc. is already in the strings table this will duplicate it.
//
// To release the memory associated with the given hyperlink,
// release the ID from the `hyperlink_set`. If the refcount reaches
// zero and the slot is needed then the context will reap the
// memory.
//
// Zig: InsertHyperlinkError!hyperlink.Id.
func (self *Page) InsertHyperlink(link Hyperlink) (HyperlinkId, error) { panic(notImplemented) }

// SetHyperlink is page.Page.setHyperlink.
//
// Set the hyperlink for the given cell. If the cell already has a
// hyperlink, then this will handle memory management and refcount
// update for the prior hyperlink.
//
// DOES NOT increment the reference count for the new hyperlink!
//
// Caller is responsible for updating the refcount in the hyperlink
// set as necessary by calling `use` if the id was not acquired with
// `add`.
//
// Zig: error{HyperlinkMapOutOfMemory}!void.
func (self *Page) SetHyperlink(row *PageRow, cell *PageCell, id HyperlinkId) error {
	panic(notImplemented)
}

// moveHyperlink is page.Page.moveHyperlink.
//
// Move the hyperlink from one cell to another. This can't fail
// because we avoid any allocations since we're just moving data.
// Destination must NOT have a hyperlink.
func (self *Page) moveHyperlink(src *PageCell, dst *PageCell) { panic(notImplemented) }

// HyperlinkCount is page.Page.hyperlinkCount.
//
// Returns the number of hyperlinks in the page. This isn't the byte
// size but the total number of unique cells that have hyperlink data.
//
// Zig: self is *const Page.
func (self *Page) HyperlinkCount() uint { panic(notImplemented) }

// HyperlinkCapacity is page.Page.hyperlinkCapacity.
//
// Returns the hyperlink capacity for the page. This isn't the byte
// size but the number of unique cells that can have hyperlink data.
//
// Zig: self is *const Page.
func (self *Page) HyperlinkCapacity() uint { panic(notImplemented) }

// SetGraphemes is page.Page.setGraphemes.
//
// Set the graphemes for the given cell. This asserts that the cell
// has no graphemes set, and only contains a single codepoint.
//
// Zig: GraphemeError!void; cps is []const u21.
func (self *Page) SetGraphemes(row *PageRow, cell *PageCell, cps []uint32) error {
	panic(notImplemented)
}

// AppendGrapheme is page.Page.appendGrapheme: append a codepoint to the
// given cell as a grapheme.
//
// Zig: Allocator.Error!void; cp is u21. The error is ErrOutOfMemory
// (dcs.go).
func (self *Page) AppendGrapheme(row *PageRow, cell *PageCell, cp uint32) error {
	panic(notImplemented)
}

// LookupGrapheme is page.Page.lookupGrapheme.
//
// Returns the codepoints for the given cell. These are the codepoints
// in addition to the first codepoint. The first codepoint is NOT
// included since it is on the cell itself.
//
// Zig: self is *const Page, cell is *const Cell; returns ?[]u21, a slice of
// the page's grapheme memory.
func (self *Page) LookupGrapheme(cell *PageCell) Optional[[]uint32] { panic(notImplemented) }

// MoveGrapheme is page.Page.moveGrapheme.
//
// Move the graphemes from one cell to another. This can't fail
// because we avoid any allocations since we're just moving data.
//
// WARNING: This will NOT change the content_tag on the cells because
// there are scenarios where we want to move graphemes without changing
// the content tag. Callers beware but assertIntegrity should catch this.
func (self *Page) MoveGrapheme(src *PageCell, dst *PageCell) { panic(notImplemented) }

// ClearGrapheme is page.Page.clearGrapheme.
//
// Clear the graphemes for a given cell.
//
// In order to update the grapheme flag on the row, call
// `updateRowGraphemeFlag` after you finish clearing any
// graphemes in the row.
func (self *Page) ClearGrapheme(cell *PageCell) { panic(notImplemented) }

// UpdateRowGraphemeFlag is page.Page.updateRowGraphemeFlag.
//
// Checks if the row contains any graphemes and sets
// the grapheme flag to false if none are found.
//
// Call after removing graphemes in a row.
func (self *Page) UpdateRowGraphemeFlag(row *PageRow) { panic(notImplemented) }

// GraphemeCount is page.Page.graphemeCount.
//
// Returns the number of graphemes in the page. This isn't the byte
// size but the total number of unique cells that have grapheme data.
//
// Zig: self is *const Page.
func (self *Page) GraphemeCount() uint { panic(notImplemented) }

// GraphemeCapacity is page.Page.graphemeCapacity.
//
// Returns the grapheme capacity for the page. This isn't the byte
// size but the number of unique cells that can have grapheme data.
//
// Zig: self is *const Page.
func (self *Page) GraphemeCapacity() uint { panic(notImplemented) }

// UpdateRowStyledFlag is page.Page.updateRowStyledFlag.
//
// Checks if the row contains any styles and sets
// the styled flag to false if none are found.
//
// Call after removing styles in a row.
func (self *Page) UpdateRowStyledFlag(row *PageRow) { panic(notImplemented) }

// IsDirty is page.Page.isDirty: returns true if this page is dirty at all.
//
// Zig: self is *const Page.
func (self *Page) IsDirty() bool { panic(notImplemented) }

// cellIndex is NOT a Zig declaration. It stands for Zig's pointer
// subtraction against page memory: the index in `self.cells` of the element
// `cell` points at.
//
// The implementation may use package unsafe for this and for nothing else,
// and must panic if the pointer is not inside the page's slice.
func (self *Page) cellIndex(cell *PageCell) uint { panic(notImplemented) }

// rowIndex is NOT a Zig declaration. It stands for Zig's pointer
// subtraction against page memory: the index in `self.rows` of the element
// `row` points at.
//
// The implementation may use package unsafe for this and for nothing else,
// and must panic if the pointer is not inside the page's slice.
func (self *Page) rowIndex(row *PageRow) uint { panic(notImplemented) }

// cellOffset is NOT a Zig declaration. It stands for
// `getOffset(Cell, self.memory, cell)`: the byte offset of the cell in the
// page as upstream would have it, `layout.cells_start + cellIndex(cell)*8`.
// This is the key of grapheme_map and hyperlink_map.
func (self *Page) cellOffset(cell *PageCell) SizeOffsetInt { panic(notImplemented) }

// PageLayout is page.Page.Layout. Every field is kept, as a number of bytes
// (the `*_start` and `*_size` fields and `total_size`, uint64) or as the
// layout of a sub-structure.
type PageLayout struct {
	total_size            uint64
	rows_start            uint64
	rows_size             uint64
	cells_start           uint64
	cells_size            uint64
	styles_start          uint64
	styles_layout         RefCountedSetLayout[StyleId]
	grapheme_alloc_start  uint64
	grapheme_alloc_layout BitmapAllocatorLayout
	grapheme_map_start    uint64
	grapheme_map_layout   HashMapOffsetHashMapLayout
	string_alloc_start    uint64
	string_alloc_layout   BitmapAllocatorLayout
	hyperlink_map_start   uint64
	hyperlink_map_layout  HashMapOffsetHashMapLayout
	hyperlink_set_start   uint64
	hyperlink_set_layout  RefCountedSetLayout[HyperlinkId]
	capacity              PageCapacity
}

// PageLayoutOf is page.Page.layout: the memory layout for a page given a
// desired minimum cols and rows size.
//
// Zig: pub inline fn layout(cap: Capacity) Layout. Named `PageLayoutOf`
// because `PageLayout` is the type. Must reproduce term51
// data/layout/page_layout.txt.
//
// The sizes Zig reads from types here are passed to the sub-layouts as:
// styleSetItemSizes (styles_layout), hyperlinkSetItemSizes
// (hyperlink_set_layout), pageGraphemeMapEntrySizes (grapheme_map_layout),
// hyperlinkMapEntrySizes (hyperlink_map_layout), and the chunk sizes
// pageGraphemeChunk and pageStringChunk (the two allocator layouts).
func PageLayoutOf(cap PageCapacity) PageLayout { panic(notImplemented) }

// PageStdCapacity is page.std_capacity.
//
// The standard capacity for a page that doesn't have special
// requirements. This is enough to support a very large number of cells.
// The standard capacity is chosen as the fast-path for allocation since
// pages of standard capacity use a pooled allocator instead of single-use
// mmaps.
//
// Zig: `.grapheme_bytes = if (builtin.is_test) 512 else 8192`; the other
// three fields take Capacity's defaults. The value here is the production
// one, 8192: the recorded corpus comes from a non-test build. Zig uses 512
// under builtin.is_test, so a translated unit test that relies on the test
// value must set `grapheme_bytes` to 512 explicitly in the test.
var PageStdCapacity = PageCapacity{
	cols:            215,
	rows:            215,
	styles:          128,
	hyperlink_bytes: pageHyperlinkBytesDefault,
	grapheme_bytes:  8192,
	string_bytes:    PageStringBytesDefault,
}

// PageSize is page.Size: the size of this page.
type PageSize struct {
	cols SizeCellCountInt
	rows SizeCellCountInt
}

// PageCapacity is page.Capacity: capacity of this page.
//
// This capacity can be maxed out (every field max) and still fit
// within a 64-bit memory space. If you need more than this, you will
// need to split data across separate pages.
//
// For 32-bit systems, it is possible to overflow the addressable
// space and this is something we still need to address in the future
// likely by limiting the maximum capacity on 32-bit systems further.
//
// In Zig four fields have non-zero defaults; a Go composite literal leaves
// omitted fields at zero. Use PageCapacityInit for a capacity with the
// defaults.
type PageCapacity struct {
	// Number of columns and rows we can know about.
	cols SizeCellCountInt
	rows SizeCellCountInt

	// Number of unique styles that can be used on this page.
	//
	// Zig default: 16.
	styles SizeStyleCountInt

	// Number of bytes to allocate for hyperlink data. Note that the
	// amount of data used for hyperlinks in total is more than this because
	// hyperlinks use string data as well as a small amount of lookup metadata.
	// This number is a rough approximation.
	//
	// Zig default: hyperlink_bytes_default (pageHyperlinkBytesDefault).
	hyperlink_bytes SizeHyperlinkCountInt

	// Number of bytes to allocate for grapheme data.
	//
	// Zig default: grapheme_bytes_default (PageGraphemeBytesDefault).
	grapheme_bytes SizeGraphemeBytesInt

	// Number of bytes to allocate for strings.
	//
	// Zig default: string_bytes_default (PageStringBytesDefault).
	string_bytes SizeStringBytesInt
}

// PageCapacityInit is NOT a Zig declaration. It stands for the Zig literal
// `Capacity{ .cols = cols, .rows = rows }`: the given columns and rows and
// every other field at its declared default (styles 16, hyperlink_bytes
// hyperlink_bytes_default = 192, grapheme_bytes grapheme_bytes_default =
// 1024, string_bytes string_bytes_default = 2048). A caller that sets
// further fields in the Zig literal assigns them to the result.
func PageCapacityInit(cols SizeCellCountInt, rows SizeCellCountInt) PageCapacity {
	panic(notImplemented)
}

// PageCapacityAdjustment is page.Capacity.Adjustment.
type PageCapacityAdjustment struct {
	// Zig: ?size.CellCountInt, default null.
	cols Optional[SizeCellCountInt]
}

// MaxCols is page.Capacity.maxCols.
//
// Returns the maximum number of columns that can be used with this
// capacity while still fitting at least one row. Returns null if even
// a single column cannot fit (which would indicate an unusable capacity).
//
// Note that this is the maximum number of columns that never increases
// the amount of memory the original capacity will take. If you modify
// the original capacity to add rows, then you can fit more columns.
func (self PageCapacity) MaxCols() Optional[SizeCellCountInt] { panic(notImplemented) }

// Adjust is page.Capacity.adjust.
//
// Adjust the capacity parameters while retaining the same total size.
//
// Adjustments always happen by limiting the rows in the page. Everything
// else can grow. If it is impossible to achieve the desired adjustment,
// OutOfMemory is returned.
//
// Zig: Allocator.Error!Capacity. The error is ErrOutOfMemory (dcs.go).
func (self PageCapacity) Adjust(req PageCapacityAdjustment) (PageCapacity, error) {
	panic(notImplemented)
}

// availableBitsForGrid is page.Capacity.availableBitsForGrid.
//
// Computes the number of bits available for rows and cells in the page.
//
// This is done by laying out the "meta" members (styles, graphemes,
// hyperlinks, strings) from the end of the page and finding where they
// start, which gives us the space available for rows and cells.
//
// Zig: returns usize, a number of bits computed from byte offsets; uint64.
func (self PageCapacity) availableBitsForGrid() uint64 { panic(notImplemented) }

// PageRow is page.Row.
//
// Zig: packed struct(u64); the fields occupy the bits from the least
// significant in the order listed: cells is bits 0..31, wrap 32,
// wrap_continuation 33, grapheme 34, styled 35, hyperlink 36,
// semantic_prompt 37..38, kitty_virtual_placeholder 39, dirty 40, _padding
// 41..63. Every field but `cells` defaults to false / .none / 0.
type PageRow struct {
	// The cells in the row offset from the page.
	//
	// Zig: Offset(Cell). Kept as the byte offset upstream would hold; the
	// row's first cell is `Page.cells[(cells - layout.cells_start) / 8]`.
	cells SizeOffsetInt

	// True if this row is soft-wrapped. The first cell of the next
	// row is a continuation of this row.
	wrap bool

	// True if the previous row to this one is soft-wrapped and
	// this row is a continuation of that row.
	wrap_continuation bool

	// True if any of the cells in this row have multi-codepoint
	// grapheme clusters. If this is true, some fast paths are not
	// possible because erasing for example may need to clear existing
	// grapheme data.
	grapheme bool

	// True if any of the cells in this row have a ref-counted style.
	// This can have false positives but never a false negative. Meaning:
	// this will be set to true the first time a style is used, but it
	// will not be set to false if the style is no longer used, because
	// checking for that condition is too expensive.
	//
	// Why have this weird false positive flag at all? This makes VT operations
	// that erase cells (such as insert lines, delete lines, erase chars,
	// etc.) MUCH MUCH faster in the case that the row was never styled.
	// At the time of writing this, the speed difference is around 4x.
	styled bool

	// True if any of the cells in this row are part of a hyperlink.
	// This is similar to styled: it can have false positives but never
	// false negatives. This is used to optimize hyperlink operations.
	hyperlink bool

	// The semantic prompt state for this row.
	//
	// This is ONLY meant to note if there are ANY cells in this
	// row that are part of a prompt. This is an optimization for more
	// efficiently implementing jump-to-prompt operations.
	//
	// This may contain false positives but never false negatives. If
	// this is set, you should still check individual cells to see if they
	// have prompt semantics.
	semantic_prompt PageRowSemanticPrompt

	// True if this row contains a virtual placeholder for the Kitty
	// graphics protocol. (U+10EEEE)
	// Note: We keep this as memory-using even if the kitty graphics
	// feature is disabled because we want to keep our padding and
	// everything throughout the same.
	kitty_virtual_placeholder bool

	// True if this row is dirty and requires a redraw. This is set to true
	// by any operation that modifies the row's contents or position, and
	// consumers of the page are expected to clear it when they redraw.
	//
	// Dirty status is only ever meant to convey that one or more cells in
	// the row have changed visually. A cell which changes in a way that
	// doesn't affect the visual representation may not be marked as dirty.
	//
	// Dirty tracking may have false positives but should never have false
	// negatives. A false negative would result in a visual artifact on the
	// screen.
	dirty bool

	// Zig: u23, default 0.
	_padding uint32
}

// PageRowSemanticPrompt is page.Row.SemanticPrompt: the semantic prompt
// state of the row. See `semantic_prompt`.
//
// Zig: enum(u2).
type PageRowSemanticPrompt uint8

const (
	// No prompt cells in this row.
	PageRowSemanticPromptNone PageRowSemanticPrompt = 0
	// Prompt cells exist in this row and this is a primary prompt
	// line. A primary prompt line is one that is not a continuation
	// and is the beginning of a prompt.
	PageRowSemanticPromptPrompt PageRowSemanticPrompt = 1
	// Prompt cells exist in this row that had k=c set (continuation)
	// line. This is used as a way to detect when a line should
	// be considered part of some prior prompt. If no prior prompt
	// is found, the last (most historical) prompt continuation line is
	// considered the prompt.
	PageRowSemanticPromptPromptContinuation PageRowSemanticPrompt = 2
)

// String returns the Zig tag name (@tagName).
func (self PageRowSemanticPrompt) String() string { panic(notImplemented) }

// ManagedMemory is page.Row.managedMemory.
//
// Returns true if this row has any managed memory outside of the
// row structure (graphemes, styles, etc.)
func (self PageRow) ManagedMemory() bool { panic(notImplemented) }

// PageCell is page.Cell.
//
// A cell represents a single terminal grid cell.
//
// The zero value of this struct must be a valid cell representing empty,
// since we zero initialize the backing memory for a page.
//
// Zig: packed struct(u64); the fields occupy the bits from the least
// significant in the order listed: content_tag is bits 0..1, content 2..25,
// style_id 26..41, wide 42..43, protected 44, hyperlink 45, semantic_content
// 46..47, _padding 48..63. The Go zero value is Zig's default `.{}`.
type PageCell struct {
	// The content tag dictates the active tag in content and possibly
	// some other behaviors.
	//
	// Zig default: .codepoint.
	content_tag PageCellContentTag

	// The content of the cell. This is a union based on content_tag.
	//
	// Zig default: .{ .codepoint = 0 }.
	content PageCellContent

	// The style ID to use for this cell within the style map. Zero
	// is always the default style so no lookup is required.
	style_id StyleId

	// The wide property of this cell, for wide characters. Characters in
	// a terminal grid can only be 1 or 2 cells wide. A wide character
	// is always next to a spacer. This is used to determine both the width
	// and spacer properties of a cell.
	//
	// Zig default: .narrow.
	wide PageCellWide

	// Whether this was written with the protection flag set.
	protected bool

	// Whether this cell is a hyperlink. If this is true then you must
	// look up the hyperlink ID in the page hyperlink_map and the ID in
	// the hyperlink_set to get the actual hyperlink data.
	hyperlink bool

	// The semantic type of the content of this cell. This is used
	// by the semantic prompt (OSC 133) set of sequences to understand
	// boundary points for content.
	//
	// Zig default: .output.
	semantic_content PageCellSemanticContent

	// Zig: u16, default 0.
	_padding uint16
}

// PageCellContent is the anonymous type of page.Cell.content.
//
// Zig: a packed union of three views of the same 24 bits, with no tag of
// its own: `codepoint: u21`, `color_palette: u8`, `color_rgb: RGB`. Go has
// no untagged union; this is a struct with one field per view, and
// `Cell.content_tag` decides which one is meaningful: `codepoint` for
// .codepoint and .codepoint_grapheme, `color_palette` for
// .bg_color_palette, `color_rgb` for .bg_color_rgb. In the 24 bits,
// codepoint is bits 0..20, color_palette is bits 0..7, and color_rgb has r
// in bits 0..7, g in 8..15, b in 16..23.
type PageCellContent struct {
	// The codepoint that this cell contains. If `grapheme` is false,
	// then this is the only codepoint in the cell. If `grapheme` is
	// true, then this is the first codepoint in the grapheme cluster.
	//
	// Zig: u21.
	codepoint uint32

	// The content is an empty cell with a background color.
	color_palette uint8
	color_rgb     PageCellRGB
}

// PageCellContentTag is page.Cell.ContentTag.
//
// Zig: enum(u2).
type PageCellContentTag uint8

const (
	// A single codepoint, could be zero to be empty cell.
	PageCellContentTagCodepoint PageCellContentTag = 0

	// A codepoint that is part of a multi-codepoint grapheme cluster.
	// The codepoint tag is active in content, but also expect more
	// codepoints in the grapheme data.
	PageCellContentTagCodepointGrapheme PageCellContentTag = 1

	// The cell has no text but only a background color. This is an
	// optimization so that cells with only backgrounds don't take up
	// style map space and also don't require a style map lookup.
	PageCellContentTagBgColorPalette PageCellContentTag = 2
	PageCellContentTagBgColorRgb     PageCellContentTag = 3
)

// String returns the Zig tag name (@tagName).
func (self PageCellContentTag) String() string { panic(notImplemented) }

// PageCellRGB is page.Cell.RGB.
//
// Zig: packed struct (24 bits): r is bits 0..7, g 8..15, b 16..23.
type PageCellRGB struct {
	r uint8
	g uint8
	b uint8
}

// PageCellWide is page.Cell.Wide.
//
// Zig: enum(u2).
type PageCellWide uint8

const (
	// Not a wide character, cell width 1.
	PageCellWideNarrow PageCellWide = 0

	// Wide character, cell width 2.
	PageCellWideWide PageCellWide = 1

	// Spacer after wide character. Do not render.
	PageCellWideSpacerTail PageCellWide = 2

	// Spacer at the end of a soft-wrapped line to indicate that a wide
	// character is continued on the next line.
	PageCellWideSpacerHead PageCellWide = 3
)

// String returns the Zig tag name (@tagName).
func (self PageCellWide) String() string { panic(notImplemented) }

// PageCellSemanticContent is page.Cell.SemanticContent.
//
// Zig: enum(u2).
type PageCellSemanticContent uint8

const (
	// Regular output content, such as command output.
	PageCellSemanticContentOutput PageCellSemanticContent = 0

	// Content that is part of user input, such as the command
	// to execute at a prompt.
	PageCellSemanticContentInput PageCellSemanticContent = 1

	// Content that is part of prompt emitted by the interactive
	// application, such as "user@host >"
	PageCellSemanticContentPrompt PageCellSemanticContent = 2
)

// String returns the Zig tag name (@tagName).
func (self PageCellSemanticContent) String() string { panic(notImplemented) }

// PageCellInit is page.Cell.init: helper to make a cell that just has a
// codepoint.
//
// Zig: pub fn init(cp: u21) Cell.
func PageCellInit(cp uint32) PageCell { panic(notImplemented) }

// bitCast is not a Zig declaration. It stands for `@as(u64, @bitCast(self))`
// in Cell.isZero (page.zig line 2066): the 64 bits of the packed struct,
// laid out as described on PageCell.
func (self PageCell) bitCast() uint64 { panic(notImplemented) }

// IsZero is page.Cell.isZero.
func (self PageCell) IsZero() bool { panic(notImplemented) }

// HasText is page.Cell.hasText.
//
// Returns true if this cell represents a cell with text to render.
//
// Cases this returns false:
//   - Cell text is blank
//   - Cell is styled but only with a background color and no text
//   - Cell has a unicode placeholder for Kitty graphics protocol
func (self PageCell) HasText() bool { panic(notImplemented) }

// Codepoint is page.Cell.codepoint.
//
// Zig: returns u21.
func (self PageCell) Codepoint() uint32 { panic(notImplemented) }

// GridWidth is page.Cell.gridWidth: the width in grid cells that this cell
// takes up.
//
// Zig: returns u2.
func (self PageCell) GridWidth() uint8 { panic(notImplemented) }

// HasStyling is page.Cell.hasStyling.
func (self PageCell) HasStyling() bool { panic(notImplemented) }

// IsEmpty is page.Cell.isEmpty: returns true if the cell has no text or
// styling.
func (self PageCell) IsEmpty() bool { panic(notImplemented) }

// HasGrapheme is page.Cell.hasGrapheme.
func (self PageCell) HasGrapheme() bool { panic(notImplemented) }

// PageCellHasTextAny is page.Cell.hasTextAny: returns true if the set of
// cells has text in it.
//
// Zig: pub inline fn hasTextAny(cells: []const Cell) bool, a function of
// the type.
func PageCellHasTextAny(cells []PageCell) bool { panic(notImplemented) }
