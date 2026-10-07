// Ported from src/terminal/page.zig (Ghostty v1.3.1).
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
// `Page.init`, `Page.reinit` and `Page.clone` call them in Zig. Clone does
// cloneBuf's work itself. initBuf's work is the private pageInitBuf, whose
// "buffer" is the rows and cells slices, so that reinit, as in Zig, reuses
// the page's own rows and cells (pointers into them stay valid).
// `Page.deinit` only unmaps the memory; it is kept as a stub because every
// test calls it.
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
//   - `pageSlowRuntimeSafety`: the build option of that name (a variable);
//   - `PageCapacityInit`: a `Capacity` literal with its declared defaults;
//   - `Page.cellIndex`, `Page.rowIndex`, `Page.cellOffset`: pointer
//     subtraction;
//   - `PageCell.bitCast`: `@bitCast` of a cell.
//
// Added by the implementation (not Zig declarations, each says what it
// stands for): `pageSizeOfRow` and the other sizes and base alignments
// layout reads from types; `pageInitBuf` (initBuf, see above);
// `pageCloneBitmapAllocator`, `pageCloneHashMap`, `pageCloneRefCountedSet`
// and `pageRebase` (cloneBuf's memcpy); `pageCellCountSet`
// (exactRowCapacity's local `CellCountSet`); `pageCellsStart` and
// `Page.rowCells` (`row.cells.ptr(self.memory)`); the names arrays of the
// enums; `pageBoolBit`; and std stand-ins `pageAlignForward`,
// `pageAlignBackward`, `pageStdMathDivCeil`, `pageStdMathCeilPowerOfTwo`.
//
// Build options:
//   - `build_options.kitty_graphics` is false in the library build. The
//     four `if (comptime build_options.kitty_graphics)` blocks (in
//     clonePartialRowFrom, moveCells twice, clearCells) are compiled out and
//     are not ported. The `Row.kitty_virtual_placeholder` bit stays, as
//     upstream says.
//   - `build_options.slow_runtime_safety` is the package variable
//     `pageSlowRuntimeSafety`, false by default (upstream's release builds)
//     and switched on by the test binary. `pause_integrity_checks` is
//     `usize` when it is set and `void` otherwise; the field is kept as a
//     `uint64` (the 64-bit reference's usize, also on 32-bit ARM).
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

import (
	"errors"
	"math"
	"math/bits"
	"slices"
	"unsafe"
)

// pageSlowRuntimeSafety is NOT a declaration of page.zig: it is Zig's build
// option `build_options.slow_runtime_safety`, which page.zig tests in
// pauseIntegrityChecks, assertIntegrity, verifyIntegrity, clonePartialRowFrom,
// getCells, appendGrapheme, moveGrapheme and clearGrapheme (PageList.zig and
// Screen.zig test it too).
//
// It is a package variable, false by default: upstream's build sets the
// option only for Debug (build/Config.zig terminalOptions, build_config.zig:
// `.Debug => true`, every release mode false), so the reference build (the
// ReleaseSafe recorder of the corpus) and shipped Ghostty run without these
// checks. Only the checks Zig gates on the option test it; plain `assert`s,
// which ReleaseSafe enforces, stay unconditional. The test binary switches
// it on in TestMain (main_test.go), as Zig's Debug test build does. Set it
// only before any Page or PageList exists: the pause counters are only
// counted while it is on.
var pageSlowRuntimeSafety = false

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

// The sizes, alignments and base alignments below are NOT Zig declarations:
// they are the numbers Page.layout and Capacity.availableBitsForGrid read
// from types at compile time, in bytes (bits for the two `pageBitSizeOf`).
// Values of the 64-bit reference build.
const (
	// @sizeOf(Row), @sizeOf(Cell), @alignOf(Cell): both are
	// packed struct(u64).
	pageSizeOfRow   = 8
	pageSizeOfCell  = 8
	pageAlignOfCell = 8

	// @bitSizeOf(Row), @bitSizeOf(Cell).
	pageBitSizeOfRow  = 64
	pageBitSizeOfCell = 64

	// std.heap.page_size_min, to which Layout.total_size is rounded up
	// (4096 in the recorded layouts).
	pagePageSizeMin = 4096

	// StyleSet.base_align: RefCountedSet's @max(@alignOf(Context) = 1,
	// @alignOf(Layout) = 8 (usize fields), @alignOf(Item) = 4,
	// @alignOf(Id) = 2).
	pageStyleSetBaseAlign = 8

	// GraphemeAlloc.base_align and StringAlloc.base_align: BitmapAllocator's
	// @alignOf(u64).
	pageGraphemeAllocBaseAlign = 8
	pageStringAllocBaseAlign   = 8

	// GraphemeMap.base_align: HashMapUnmanaged's @max(@alignOf(Header) = 4,
	// @alignOf(K) = 4, @alignOf(V) = 8); see pageGraphemeMapEntrySizes.
	pageGraphemeMapBaseAlign = 8

	// hyperlink.Set.base_align: RefCountedSet's @max(@alignOf(Context) = 8
	// (two pointers), @alignOf(Layout) = 8, @alignOf(Item) = 8,
	// @alignOf(Id) = 2).
	pageHyperlinkSetBaseAlign = 8

	// hyperlink.Map.base_align: HashMapUnmanaged's @max(@alignOf(Header) =
	// 4, @alignOf(K) = 4, @alignOf(V) = 2); see hyperlinkMapEntrySizes.
	pageHyperlinkMapBaseAlign = 4
)

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
	// otherwise. uint64: the reference's usize. See the file header.
	pause_integrity_checks uint64
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
func PageInit(cap PageCapacity) (Page, error) {
	l := PageLayoutOf(cap)

	// We use mmap directly to avoid Zig allocator overhead
	// (small but meaningful for this path) and because a private
	// anonymous mmap is guaranteed on Linux and macOS to be zeroed,
	// which is a critical property for us.
	assert(l.total_size%pagePageSizeMin == 0)
	// Zig: posix.mmap(...) and `errdefer posix.munmap(backing)`: Go
	// memory, zeroed, whose allocation cannot fail here. The buffer is
	// the page's rows and cells (see pageInitBuf).
	rows := make([]PageRow, l.capacity.rows)
	cells := make([]PageCell, uint(l.capacity.cols)*uint(l.capacity.rows))

	return pageInitBuf(rows, cells, l), nil
}

// pageInitBuf is page.Page.initBuf for Go memory: initialize a new page
// using the given backing memory. Zig's `buf: OffsetBuf` is the page's rows
// and cells, `rows` (`l.capacity.rows` long) and `cells`
// (`l.capacity.rows * l.capacity.cols` long), zeroed; the sub-structures
// make their own memory. It is NOT among the wireframe's declarations (the
// wireframe omits initBuf, see the file header): it is kept, private,
// because init and reinit both call it in Zig, and reinit reuses the
// page's own rows and cells, so that pointers into them stay valid as they
// do in Zig.
//
// Zig: pub inline fn initBuf(buf: OffsetBuf, l: Layout) Page. Upstream's
// doc comment: "Initialize a new page using the given backing memory. It is
// up to the caller to not call deinit on these pages."
func pageInitBuf(rows []PageRow, cells []PageCell, l PageLayout) Page {
	cap := l.capacity
	// Zig: buf.member(Row, l.rows_start) and buf.member(Cell,
	// l.cells_start).
	assert(len(rows) == int(cap.rows))

	// We need to go through and initialize all the rows so that
	// they point to a valid offset into the cells, since the rows
	// zero-initialized aren't valid.
	cells_len := uint(cap.cols) * uint(cap.rows)
	self := Page{
		rows:     rows,
		cells:    cells,
		capacity: cap,
	}
	cells_ptr := cells[0:cells_len]
	for y := range rows[0:cap.rows] {
		row := &rows[y]
		start := uint(y) * uint(cap.cols)
		*row = PageRow{
			// getOffset(Cell, buf, &cells_ptr[start])
			cells: self.cellOffset(&cells_ptr[start]),
		}
	}

	self.styles = RefCountedSetInit[Style, StyleID, SizeCellCountInt, StyleSetContext](
		l.styles_layout,
		StyleSetContext{},
	)
	self.string_alloc = BitmapAllocatorInit[uint8](
		pageStringChunk,
		l.string_alloc_layout,
	)
	self.grapheme_alloc = BitmapAllocatorInit[uint32](
		pageGraphemeChunk,
		l.grapheme_alloc_layout,
	)
	self.grapheme_map = HashMapOffsetHashMapInit[SizeOffsetInt, []uint32, hashMapAutoContext[SizeOffsetInt]](
		l.grapheme_map_layout,
	)
	self.hyperlink_map = HashMapOffsetHashMapInit[SizeOffsetInt, HyperlinkID, hashMapAutoContext[SizeOffsetInt]](
		l.hyperlink_map_layout,
	)
	self.hyperlink_set = RefCountedSetInit[HyperlinkPageEntry, HyperlinkID, SizeCellCountInt, HyperlinkSetContext](
		l.hyperlink_set_layout,
		HyperlinkSetContext{},
	)
	self.size = PageSize{cols: cap.cols, rows: cap.rows}
	self.dirty = false
	return self
}

// Deinit is page.Page.deinit.
//
// Deinitialize the page, freeing any backing memory. Do NOT call
// this if you allocated the backing memory yourself (i.e. you used
// initBuf).
func (self *Page) Deinit() {
	// Zig: posix.munmap(self.memory): the garbage collector frees.
	*self = Page{} // self.* = undefined
}

// Reinit is page.Page.reinit: reinitialize the page with the same capacity.
func (self *Page) Reinit() {
	// We zero the page memory as u64 instead of u8 because
	// we can and it's empirically quite a bit faster.
	//
	// Zig: @memset over self.memory. Go: the rows and cells, which
	// pageInitBuf reuses; the sub-structures get new memory there.
	clear(self.rows)
	clear(self.cells)
	*self = pageInitBuf(self.rows, self.cells, PageLayoutOf(self.capacity))
}

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
func (self *Page) PauseIntegrityChecks(v bool) {
	if pageSlowRuntimeSafety {
		if v {
			// Zig: += traps on overflow.
			assert(self.pause_integrity_checks < math.MaxUint64)
			self.pause_integrity_checks += 1
		} else {
			// Zig: -= traps on underflow.
			assert(self.pause_integrity_checks > 0)
			self.pause_integrity_checks -= 1
		}
	}
}

// AssertIntegrity is page.Page.assertIntegrity.
//
// A helper that can be used to assert the integrity of the page
// when runtime safety is enabled. This is a no-op when runtime
// safety is disabled. This uses the libc allocator.
//
// Zig: self is *const Page.
func (self *Page) AssertIntegrity() {
	if pageSlowRuntimeSafety {
		// Zig: a std.heap.DebugAllocator for verifyIntegrity's temporary
		// allocations; Go memory.
		if err := self.VerifyIntegrity(); err != nil {
			// Zig: log.err
			warn("page", "page integrity violation, crashing. err=error.%v", err)
			panic("page integrity violation")
		}
	}
}

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
func (self *Page) VerifyIntegrity() error {
	// Some things that seem like we should check but do not:
	//
	// - We do not check that the style ref count is exact, only that
	//   it is at least what we see. We do this because some fast paths
	//   trim rows without clearing data.
	// - We do not check that styles seen is exactly the same as the
	//   styles count in the page for the same reason as above.
	// - We only check that we saw less graphemes than the total memory
	//   used for the same reason as styles above.
	//

	// We don't run integrity checks on Valgrind because its soooooo slow,
	// Valgrind is our integrity checker, and we run these during unit
	// tests (non-Valgrind) anyways so we're verifying anyways.
	//
	// Zig: if (std.valgrind.runningOnValgrind() > 0) return; Go never
	// runs on Valgrind.

	if pageSlowRuntimeSafety {
		if self.pause_integrity_checks > 0 {
			return nil
		}
	}

	if self.size.rows == 0 {
		warn("page", "page integrity violation zero row count")
		return ErrZeroRowCount
	}
	if self.size.cols == 0 {
		warn("page", "page integrity violation zero col count")
		return ErrZeroColCount
	}

	// Zig: an ArenaAllocator over alloc_gpa for the two maps below; Go
	// memory.

	var graphemes_seen uint = 0
	// Zig: std.AutoHashMap(StyleId, usize) and std.AutoHashMap(hyperlink.Id,
	// usize). Go maps: only the order of the ref count checks below
	// differs, which decides nothing but which mismatch is logged.
	styles_seen := map[StyleID]uint{}
	hyperlinks_seen := map[HyperlinkID]uint{}

	grapheme_count := self.GraphemeCount()

	rows := self.rows[0:self.size.rows]
	for y := range rows {
		row := &rows[y]
		graphemes_start := graphemes_seen
		cells := self.rowCells(row)[0:self.size.cols]
		for x := range cells {
			cell := &cells[x]
			if cell.HasGrapheme() {
				// If a cell has grapheme data, it must be present in
				// the grapheme map.
				if !self.LookupGrapheme(cell).Valid {
					warn(
						"page",
						"page integrity violation y=%d x=%d grapheme data missing",
						y, x,
					)
					return ErrMissingGraphemeData
				}

				graphemes_seen += 1
			} else if grapheme_count > 0 {
				// It should not have grapheme data if it isn't marked.
				// The grapheme_count check above is just an optimization
				// to speed up integrity checks.
				if self.LookupGrapheme(cell).Valid {
					warn(
						"page",
						"page integrity violation y=%d x=%d cell not marked as grapheme",
						y, x,
					)
					return ErrUnmarkedGraphemeCell
				}
			}

			if cell.style_id != StyleDefaultID {
				// If a cell has a style, it must be present in the styles
				// set. Accessing it with `get` asserts that.
				_ = self.styles.Get(
					cell.style_id,
				)

				if !row.styled {
					warn(
						"page",
						"page integrity violation y=%d x=%d row not marked as styled",
						y, x,
					)
					return ErrUnmarkedStyleRow
				}

				// Zig: getOrPut (try: an allocation failure; none in Go),
				// 0 when not found, then += 1.
				styles_seen[cell.style_id] += 1
			}

			if cell.hyperlink {
				id_opt := self.LookupHyperlink(cell)
				if !id_opt.Valid {
					warn(
						"page",
						"page integrity violation y=%d x=%d hyperlink data missing",
						y, x,
					)
					return ErrMissingHyperlinkData
				}
				id := id_opt.Value

				if !row.hyperlink {
					warn(
						"page",
						"page integrity violation y=%d x=%d row not marked as hyperlink",
						y, x,
					)
					return ErrUnmarkedHyperlinkRow
				}

				// Zig: getOrPut, 0 when not found, then += 1.
				hyperlinks_seen[id] += 1

				// Hyperlink ID should be valid. This just straight crashes
				// if this fails due to assertions.
				_ = self.hyperlink_set.Get(id)
			} else {
				// It should not have hyperlink data if it isn't marked
				if self.LookupHyperlink(cell).Valid {
					warn(
						"page",
						"page integrity violation y=%d x=%d cell not marked as hyperlink",
						y, x,
					)
					return ErrUnmarkedHyperlinkCell
				}
			}

			switch cell.wide {
			case PageCellWideNarrow:
			case PageCellWideWide:

			case PageCellWideSpacerTail:
				// Spacer tails can't be at the start because they follow
				// a wide char.
				if x == 0 {
					warn(
						"page",
						"page integrity violation y=%d x=%d spacer tail at start",
						y, x,
					)
					return ErrInvalidSpacerTailLocation
				}

				// Spacer tails must follow a wide char
				prev := cells[x-1]
				if prev.wide != PageCellWideWide {
					warn(
						"page",
						"page integrity violation y=%d x=%d spacer tail not following wide",
						y, x,
					)
					return ErrInvalidSpacerTailLocation
				}

			case PageCellWideSpacerHead:
				// Spacer heads must be at the end
				if x != int(self.size.cols-1) {
					warn(
						"page",
						"page integrity violation y=%d x=%d spacer head not at end",
						y, x,
					)
					return ErrInvalidSpacerHeadLocation
				}

				// The row must be wrapped
				if !row.wrap {
					warn(
						"page",
						"page integrity violation y=%d spacer head not wrapped",
						y,
					)
					return ErrUnwrappedSpacerHead
				}

			default:
				panic("unreachable")
			}
		}

		// Check row grapheme data
		if graphemes_seen > graphemes_start {
			// If a cell in a row has grapheme data, the row must
			// be marked as having grapheme data.
			if !row.grapheme {
				warn(
					"page",
					"page integrity violation y=%d grapheme data but row not marked",
					y,
				)
				return ErrUnmarkedGraphemeRow
			}
		}
	}

	// Our graphemes seen should exactly match the grapheme count
	if graphemes_seen > self.GraphemeCount() {
		warn(
			"page",
			"page integrity violation grapheme count mismatch expected=%d actual=%d",
			graphemes_seen, self.GraphemeCount(),
		)
		return ErrInvalidGraphemeCount
	}

	// Verify all our styles have the correct ref count.
	{
		for key, value := range styles_seen {
			ref_count := self.styles.RefCount(key)
			if uint(ref_count) < value {
				warn(
					"page",
					"page integrity violation style ref count mismatch id=%d expected=%d actual=%d",
					key, value, ref_count,
				)
				return ErrMismatchedStyleRef
			}
		}
	}

	// Verify all our hyperlinks have the correct ref count.
	{
		for key, value := range hyperlinks_seen {
			ref_count := self.hyperlink_set.RefCount(key)
			if uint(ref_count) < value {
				warn(
					"page",
					"page integrity violation hyperlink ref count mismatch id=%d expected=%d actual=%d",
					key, value, ref_count,
				)
				return ErrMismatchedHyperlinkRef
			}
		}
	}

	// Verify there are no zombie styles, that is, styles in the
	// set with ref counts > 0, which are not present in the page.
	{
		styles_table := self.styles.table[0:self.styles.layout.table_cap]
		styles_items := self.styles.items[0:self.styles.layout.cap]

		var zombies uint = 0

		for _, id := range styles_table {
			if id == 0 {
				continue
			}
			item := styles_items[id]
			if item.meta.ref == 0 {
				continue
			}

			expected := styles_seen[id] // styles_seen.get(id) orelse 0
			if expected > 0 {
				continue
			}

			if uint(item.meta.ref) > expected {
				zombies += 1
			}
		}

		// NOTE: This is currently disabled because @qwerasd says that
		// certain fast paths can cause this but its okay.
		// Just 1 zombie style might be the cursor style, so ignore it.
		// if (zombies > 1) {
		//     log.warn(
		//         "page integrity violation zombie styles count={}",
		//         .{zombies},
		//     );
		//     return IntegrityError.ZombieStyles;
		// }
		_ = zombies
	}

	return nil
}

// Clone is page.Page.clone.
//
// Clone the contents of this page. This will allocate new memory
// using the page allocator. If you want to manage memory manually,
// use cloneBuf.
//
// Zig: pub inline fn clone(self: *const Page) !Page. The error is mmap's.
func (self *Page) Clone() (Page, error) {
	// Zig: posix.mmap of self.memory.len bytes (errdefer posix.munmap),
	// then `return self.cloneBuf(backing)`, whose work follows (see the
	// file header).

	// Zig: cloneBuf.
	//
	// The entire concept behind a page is that everything is stored
	// as offsets so we can do a simple linear copy of the backing
	// memory and copy all the offsets and everything will work.
	result := *self

	// This is a memcpy. We may want to investigate if there are
	// faster ways to do this (i.e. copy-on-write tricks) but I suspect
	// they'll be slower. I haven't experimented though.
	// std.log.warn("copy bytes={}", .{self.memory.len});
	//
	// Zig: fastmem.copy(u8, result.memory, self.memory). Go: a copy of
	// every slice the page owns; the slices that point into the page's two
	// allocators are pointed at the same positions in the copies
	// (pageRebase), which is what copying offsets does.
	result.rows = slices.Clone(self.rows)
	result.cells = slices.Clone(self.cells)
	result.string_alloc = pageCloneBitmapAllocator(self.string_alloc)
	result.grapheme_alloc = pageCloneBitmapAllocator(self.grapheme_alloc)
	result.grapheme_map = pageCloneHashMap(self.grapheme_map, func(v []uint32) []uint32 {
		return pageRebase(v, self.grapheme_alloc.chunks, result.grapheme_alloc.chunks, pageGraphemeChunk/4)
	})
	result.styles = pageCloneRefCountedSet(self.styles, func(v Style) Style { return v })
	result.hyperlink_map = pageCloneHashMap(self.hyperlink_map, func(v HyperlinkID) HyperlinkID { return v })
	result.hyperlink_set = pageCloneRefCountedSet(self.hyperlink_set, func(v HyperlinkPageEntry) HyperlinkPageEntry {
		v.uri = pageRebase(v.uri, self.string_alloc.chunks, result.string_alloc.chunks, pageStringChunk)
		if v.id.Tag == HyperlinkPageEntryIdExplicit {
			v.id.Explicit = pageRebase(v.id.Explicit, self.string_alloc.chunks, result.string_alloc.chunks, pageStringChunk)
		}
		return v
	})

	return result, nil
}

// pageCloneBitmapAllocator is NOT a Zig declaration: the part of
// cloneBuf's memcpy that copies a bitmap allocator (its bitmaps and its
// chunks).
func pageCloneBitmapAllocator[T any](a BitmapAllocator[T]) BitmapAllocator[T] {
	result := a
	result.bitmap = slices.Clone(a.bitmap)
	result.chunks = slices.Clone(a.chunks)
	return result
}

// pageCloneHashMap is NOT a Zig declaration: the part of cloneBuf's memcpy
// that copies an offset hash map (metadata, header, keys, values). The copy
// gets its own header. `rebase` is applied to every value.
func pageCloneHashMap[K any, V any, C HashMapContext[K]](
	m HashMapOffsetHashMap[K, V, C],
	rebase func(V) V,
) HashMapOffsetHashMap[K, V, C] {
	result := m
	result.metadata = slices.Clone(m.metadata)
	if m.hdr != nil {
		hdr := *m.hdr
		hdr.keys = slices.Clone(m.hdr.keys)
		hdr.values = slices.Clone(m.hdr.values)
		for i := range hdr.values {
			hdr.values[i] = rebase(hdr.values[i])
		}
		result.hdr = &hdr
	}
	return result
}

// pageCloneRefCountedSet is NOT a Zig declaration: the part of cloneBuf's
// memcpy that copies a ref-counted set (table and items). `rebase` is
// applied to every item's value.
func pageCloneRefCountedSet[T any, Id RefCountedSetInt, Ref RefCountedSetInt, C RefCountedSetContext[T]](
	s RefCountedSet[T, Id, Ref, C],
	rebase func(T) T,
) RefCountedSet[T, Id, Ref, C] {
	result := s
	result.table = slices.Clone(s.table)
	result.items = slices.Clone(s.items)
	for i := range result.items {
		result.items[i].value = rebase(result.items[i].value)
	}
	return result
}

// pageRebase is NOT a Zig declaration. `s` is an allocation of a bitmap
// allocator whose chunks are `from` (it starts at a chunk boundary,
// `chunk_len` elements per chunk); the result is the same position and
// length, and the same capacity, in `to`, the chunks of the copied
// allocator. This is what copying an `Offset(T).Slice` with the page memory
// does. An empty slice, or one that is not in `from` (not page memory, so
// Zig would hold no offset for it), is returned as it is.
func pageRebase[T any](s []T, from []T, to []T, chunk_len int) []T {
	if len(s) == 0 {
		return s
	}
	for i := 0; i < len(from); i += chunk_len {
		if &from[i] == &s[0] {
			return to[i : i+len(s) : i+cap(s)]
		}
	}
	return s
}

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

// pageCellCountSet is the local type `CellCountSet` of
// Page.exactRowCapacity: std.StaticBitSet(maxInt(CellCountInt) + 1), an
// ArrayBitSet of 65536 bits in usize masks. Only the methods
// exactRowCapacity uses, with Zig's names. The zero value is `.initEmpty()`.
// Every u16 index is in range, so no index is asserted.
type pageCellCountSet struct {
	masks [65536 / 64]uint64
}

func (self *pageCellCountSet) set(index SizeCellCountInt) {
	self.masks[index>>6] |= uint64(1) << (index & 63)
}

func (self *pageCellCountSet) isSet(index SizeCellCountInt) bool {
	return self.masks[index>>6]&(uint64(1)<<(index&63)) != 0
}

func (self *pageCellCountSet) count() uint {
	var c uint = 0
	for _, m := range self.masks {
		c += uint(bits.OnesCount64(m))
	}
	return c
}

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
func (self *Page) ExactRowCapacity(y_start uint, y_end uint) PageCapacity {
	cap_, err := self.exactRowCapacityOrError(y_start, y_end)
	// Zig: `.styles = @intCast(styles_cap)` traps when the style capacity
	// does not fit the style count type.
	assert(err == nil)
	return cap_
}

// exactRowCapacityOrError is NOT a Zig declaration. It is the body of
// ExactRowCapacity (page.Page.exactRowCapacity), except that the one cast
// terminal input can make fail, `.styles = @intCast(styles_cap)`, returns
// ErrOutOfSpace instead of trapping: a range of rows that references every
// id of a full style set of the maximum capacity (53,247 styles) needs a
// style capacity of 65,536, one more than the u16 holds. upstream aborts
// there in splitForCapacity (term51 docs/upstream-crashes.md); the port
// fails the split instead (docs/mapping.md, "Page-layer bookkeeping").
// The other casts stay assertions.
func (self *Page) exactRowCapacityOrError(y_start uint, y_end uint) (PageCapacity, error) {
	assert(y_start < y_end)
	assert(y_end <= uint(self.size.rows))

	// Track unique IDs using a bitset. Both style IDs and hyperlink IDs
	// are CellCountInt (u16), so we reuse this set for both to save
	// stack memory (~8KB instead of ~16KB).
	//
	// Zig: const CellCountSet = std.StaticBitSet(maxInt(CellCountInt) + 1),
	// which is pageCellCountSet. The two comptime asserts that StyleCountInt
	// and HyperlinkCountInt are CellCountInt hold in size.go (aliases).

	// Accumulators
	id_set := pageCellCountSet{} // .initEmpty()
	var grapheme_bytes uint64 = 0
	var string_bytes uint64 = 0

	// First pass: count styles and grapheme bytes
	rows := self.rows[y_start:y_end]
	for i := range rows {
		row := &rows[i]
		cells := self.rowCells(row)[0:self.size.cols]
		for j := range cells {
			cell := &cells[j]
			if cell.style_id != StyleDefaultID {
				id_set.set(cell.style_id)
			}

			if cell.HasGrapheme() {
				if cps := self.LookupGrapheme(cell); cps.Valid {
					grapheme_bytes += BitmapAllocatorBytesRequired[uint32](pageGraphemeChunk, uint(len(cps.Value)))
				}
			}
		}
	}
	styles_cap := RefCountedSetCapacityForCount(id_set.count())

	// Second pass: count hyperlinks and string bytes
	// We count both unique hyperlinks (for hyperlink_set) and total
	// hyperlink cells (for hyperlink_map capacity).
	id_set = pageCellCountSet{} // .initEmpty()
	var hyperlink_cells uint = 0
	for i := range rows {
		row := &rows[i]
		cells := self.rowCells(row)[0:self.size.cols]
		for j := range cells {
			cell := &cells[j]
			if cell.hyperlink {
				hyperlink_cells += 1
				if id := self.LookupHyperlink(cell); id.Valid {
					// Only count each unique hyperlink once for set sizing
					if !id_set.isSet(id.Value) {
						id_set.set(id.Value)

						// Get the hyperlink entry to compute string bytes
						entry := self.hyperlink_set.Get(id.Value)
						string_bytes += BitmapAllocatorBytesRequired[uint8](pageStringChunk, uint(len(entry.uri)))

						switch entry.id.Tag {
						case HyperlinkPageEntryIdImplicit:
						case HyperlinkPageEntryIdExplicit:
							slice := entry.id.Explicit
							string_bytes += BitmapAllocatorBytesRequired[uint8](pageStringChunk, uint(len(slice)))
						default:
							panic("unreachable")
						}
					}
				}
			}
		}
	}

	// The hyperlink_map capacity in layout() is computed as:
	//   hyperlink_count * hyperlink_cell_multiplier (rounded to power of 2)
	// We need enough hyperlink_bytes so that when layout() computes
	// the map capacity, it can accommodate all hyperlink cells. This
	// is unit tested.
	var hyperlink_cap uint
	{ // cap:
		hyperlink_count := id_set.count()
		hyperlink_set_cap := RefCountedSetCapacityForCount(hyperlink_count)
		// Zig: std.math.divCeil(...) catch 0; the divisor is not zero.
		hyperlink_map_min := uint(pageStdMathDivCeil(
			uint64(hyperlink_cells),
			pageHyperlinkCellMultiplier,
		))
		hyperlink_cap = max(hyperlink_set_cap, hyperlink_map_min) // break :cap
	}

	// All the intCasts below are safe because we should have a
	// capacity strictly less than or equal to this page's capacity.
	//
	// Zig: each @intCast traps when the value does not fit; asserted,
	// except the style capacity (see above).
	if styles_cap > math.MaxUint16 {
		return PageCapacity{}, ErrOutOfSpace
	}
	assert(grapheme_bytes <= math.MaxUint32)
	assert(uint64(hyperlink_cap)*hyperlinkSetItemSizeOf <= math.MaxUint16)
	assert(string_bytes <= math.MaxUint32)
	return PageCapacity{
		cols:            self.size.cols,
		rows:            SizeCellCountInt(y_end - y_start),
		styles:          SizeStyleCountInt(styles_cap),
		grapheme_bytes:  SizeGraphemeBytesInt(grapheme_bytes),
		hyperlink_bytes: SizeHyperlinkCountInt(uint64(hyperlink_cap) * hyperlinkSetItemSizeOf),
		string_bytes:    SizeStringBytesInt(string_bytes),
	}, nil
}

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
func (self *Page) CloneFrom(other *Page, y_start uint, y_end uint) error {
	assert(y_start <= y_end)
	assert(y_end <= uint(other.size.rows))
	assert(y_end-y_start <= uint(self.size.rows))

	other_rows := other.rows[y_start:y_end]
	rows := self.rows[0 : y_end-y_start]
	for i := range rows {
		dst_row := &rows[i]
		src_row := &other_rows[i]
		if err := self.CloneRowFrom(other, dst_row, src_row); err != nil {
			return err
		}
	}

	// We should remain consistent
	self.AssertIntegrity()
	return nil
}

// CloneRowFrom is page.Page.cloneRowFrom: clone a single row from another
// page into this page.
//
// Zig: CloneFromError!void; other is *const Page, src_row is *const Row.
func (self *Page) CloneRowFrom(other *Page, dst_row *PageRow, src_row *PageRow) error {
	return self.ClonePartialRowFrom(
		other,
		dst_row,
		src_row,
		0,
		uint(self.size.cols),
	)
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
	// This whole operation breaks integrity until the end.
	self.PauseIntegrityChecks(true)
	defer func() {
		self.PauseIntegrityChecks(false)
		self.AssertIntegrity()
	}()

	cell_len := min(self.size.cols, other.size.cols)
	x_end := min(x_end_req, uint(cell_len))
	assert(x_start <= x_end)
	other_cells := other.rowCells(src_row)[x_start:x_end]
	cells := self.rowCells(dst_row)[x_start:x_end]

	// If our destination has styles or graphemes then we need to
	// clear some state. This will free up the managed memory as well.
	if dst_row.ManagedMemory() {
		self.ClearCells(dst_row, x_start, x_end)
	}

	// Copy all the row metadata but keep our cells offset
	{ // copy:
		copy_ := *src_row

		// If we're not copying the full row then we want to preserve
		// some original state from our dst row.
		if (x_end - x_start) < uint(self.size.cols) {
			copy_.wrap = dst_row.wrap
			copy_.wrap_continuation = dst_row.wrap_continuation
			copy_.grapheme = dst_row.grapheme
			copy_.hyperlink = dst_row.hyperlink
			copy_.styled = dst_row.styled
			copy_.dirty = copy_.dirty || dst_row.dirty // |=
		}

		// Our cell offset remains the same
		copy_.cells = dst_row.cells

		*dst_row = copy_ // break :copy copy
	}

	// If we have no managed memory in the source, then we can just
	// copy it directly.
	if !src_row.ManagedMemory() {
		// This is an integrity check: if the row claims it doesn't
		// have managed memory then all cells must also not have
		// managed memory.
		if pageSlowRuntimeSafety {
			for _, cell := range other_cells {
				assert(!cell.HasGrapheme())
				assert(!cell.hyperlink)
				assert(cell.style_id == StyleDefaultID)
			}
		}

		copy(cells, other_cells) // fastmem.copy(Cell, cells, other_cells)
	} else {
		// We have managed memory, so we have to do a slower copy to
		// get all of that right.
		for i := range cells {
			dst_cell := &cells[i]
			src_cell := &other_cells[i]
			*dst_cell = *src_cell

			// Reset any managed memory markers on the cell so that we don't
			// hit an integrity check if we have to return an error because
			// the page can't fit the new memory.
			dst_cell.hyperlink = false
			dst_cell.style_id = StyleDefaultID
			if dst_cell.content_tag == PageCellContentTagCodepointGrapheme {
				dst_cell.content_tag = PageCellContentTagCodepoint
			}

			if src_cell.HasGrapheme() {
				// Copy the grapheme codepoints
				cps_opt := other.LookupGrapheme(src_cell)
				assert(cps_opt.Valid) // .?
				cps := cps_opt.Value

				// Safe to use setGraphemes because we cleared all
				// managed memory for our destination cell range.
				if err := self.SetGraphemes(dst_row, dst_cell, cps); err != nil {
					return err
				}
			}
			if src_cell.hyperlink {
				// hyperlink:
				id_opt := other.LookupHyperlink(src_cell)
				assert(id_opt.Valid) // .?
				id := id_opt.Value

				// Fast-path: same page we can add with the same id.
				if other == self {
					self.hyperlink_set.Use(id)
					if err := self.SetHyperlink(dst_row, dst_cell, id); err != nil {
						return err
					}
					goto hyperlink // break :hyperlink
				}

				// Slow-path: get the hyperlink from the other page,
				// add it, and migrate.

				// If our page can't support an additional cell with
				// a hyperlink then we have to return an error.
				if self.HyperlinkCount() >= self.HyperlinkCapacity() {
					// The hyperlink map capacity needs to be increased.
					return ErrHyperlinkMapOutOfMemory
				}

				{
					other_link := other.hyperlink_set.Get(id)
					var dst_id HyperlinkID
					{ // dst_id:
						// First check if the link already exists in our page,
						// and increment its refcount if so, since we're about
						// to use it.
						if i := self.hyperlink_set.LookupContext(
							*other_link,
							HyperlinkSetContext{page: self, src_page: other},
						); i.Valid {
							self.hyperlink_set.Use(i.Value)
							dst_id = i.Value // break :dst_id i
							goto dst_id
						}

						// If we don't have this link in our page yet then
						// we need to clone it over and add it to our set.

						// Clone the link.
						dst_link, err := other_link.Dupe(other, self)
						if err != nil {
							// Zig: comptime assert(@TypeOf(e) == error{OutOfMemory});
							// The string alloc capacity needs to be increased.
							return ErrStringAllocOutOfMemory
						}

						// Add it, preferring to use the same ID as the other
						// page, since this *probably* speeds up full-page
						// clones.
						//
						// TODO(qwerasd): verify the assumption that `addWithId`
						// is ever actually useful, I think it may not be.
						added, err := self.hyperlink_set.AddWithIdContext(
							dst_link,
							id,
							HyperlinkSetContext{page: self},
						)
						if err != nil {
							switch err {
							// The hyperlink set capacity needs to be increased.
							case ErrOutOfMemory:
								return ErrHyperlinkSetOutOfMemory

							// The hyperlink set needs to be rehashed.
							case ErrNeedsRehash:
								return ErrHyperlinkSetNeedsRehash

							default:
								panic("unreachable")
							}
						}
						// orelse id
						if added.Valid {
							dst_id = added.Value
						} else {
							dst_id = id
						}
					}
				dst_id:

					if err := self.SetHyperlink(dst_row, dst_cell, dst_id); err != nil {
						return err
					}
				}
			}
		hyperlink:
			if src_cell.style_id != StyleDefaultID {
				// style:
				dst_row.styled = true

				if other == self {
					// If it's the same page we don't have to worry about
					// copying the style, we can use the style ID directly.
					dst_cell.style_id = src_cell.style_id
					self.styles.Use(dst_cell.style_id)
					continue // break :style (nothing follows in the loop body)
				}

				// Slow path: Get the style from the other
				// page and add it to this page's style set.
				other_style := other.styles.Get(src_cell.style_id)
				added, err := self.styles.AddWithId(
					*other_style,
					src_cell.style_id,
				)
				if err != nil {
					switch err {
					// The style set capacity needs to be increased.
					case ErrOutOfMemory:
						return ErrStyleSetOutOfMemory

					// The style set needs to be rehashed.
					case ErrNeedsRehash:
						return ErrStyleSetNeedsRehash

					default:
						panic("unreachable")
					}
				}
				// orelse src_cell.style_id
				if added.Valid {
					dst_cell.style_id = added.Value
				} else {
					dst_cell.style_id = src_cell.style_id
				}
			}
			// Zig: the `if (comptime build_options.kitty_graphics)` block
			// is compiled out (see the file header).
		}
	}

	// If we are growing columns, then we need to ensure spacer heads
	// are cleared.
	if self.size.cols > other.size.cols {
		last := &cells[other.size.cols-1]
		if last.wide == PageCellWideSpacerHead {
			last.wide = PageCellWideNarrow
		}
	}
	return nil
}

// GetRow is page.Page.getRow: get a single row. y must be valid.
//
// Zig: self is *const Page.
func (self *Page) GetRow(y uint) *PageRow {
	assert(y < uint(self.size.rows))
	return &self.rows[y]
}

// GetCells is page.Page.getCells: get the cells for a row.
//
// Zig: self is *const Page. The result is `size.cols` long.
func (self *Page) GetCells(row *PageRow) []PageCell {
	if pageSlowRuntimeSafety {
		// Zig: asserts that the row's address is at or after the rows
		// array and before the cells array, i.e. that it is one of this
		// page's rows. rowIndex panics if it is not.
		_ = self.rowIndex(row)
	}

	cells := self.rowCells(row)
	return cells[0:self.size.cols]
}

// GetRowAndCell is page.Page.getRowAndCell: get the row and cell for the
// given X/Y within this page.
//
// Zig: self is *const Page; the result is an anonymous struct.
func (self *Page) GetRowAndCell(x uint, y uint) struct {
	row  *PageRow
	cell *PageCell
} {
	assert(y < uint(self.size.rows))
	assert(x < uint(self.size.cols))

	rows := self.rows
	row := &rows[y]
	cell := &self.rowCells(row)[x]

	return struct {
		row  *PageRow
		cell *PageCell
	}{row: row, cell: cell}
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
	defer self.AssertIntegrity()

	src_cells := self.rowCells(src_row)[src_left : src_left+len]
	dst_cells := self.rowCells(dst_row)[dst_left : dst_left+len]

	// Clear our destination now matter what
	self.ClearCells(dst_row, dst_left, dst_left+len)

	// If src has no managed memory, this is very fast.
	if !src_row.ManagedMemory() {
		copy(dst_cells, src_cells) // fastmem.copy(Cell, dst_cells, src_cells)
	} else {
		// Source has graphemes or hyperlinks...
		for i := range src_cells {
			src := &src_cells[i]
			dst := &dst_cells[i]
			*dst = *src
			if src.HasGrapheme() {
				// Required for moveGrapheme assertions
				dst.content_tag = PageCellContentTagCodepoint
				self.MoveGrapheme(src, dst)
				src.content_tag = PageCellContentTagCodepoint
				dst.content_tag = PageCellContentTagCodepointGrapheme
				dst_row.grapheme = true
			}
			if src.hyperlink {
				dst.hyperlink = false
				self.moveHyperlink(src, dst)
				dst.hyperlink = true
				dst_row.hyperlink = true
			}
			// Zig: the `if (comptime build_options.kitty_graphics)` block
			// is compiled out (see the file header).
		}
	}

	// The destination row has styles if any of the cells are styled
	if !dst_row.styled {
		styled := false // styled: for (dst_cells) |c| { ... } else false
		for _, c := range dst_cells {
			if c.style_id != StyleDefaultID {
				styled = true // break :styled true
				break
			}
		}
		dst_row.styled = styled
	}

	// Clear our source row now that the copy is complete. We can NOT
	// use clearCells here because clearCells will garbage collect our
	// styles and graphames but we moved them above.
	//
	// Zero the cells as u64s since empirically this seems
	// to be a bit faster than using @memset(src_cells, .{})
	clear(src_cells)
	// Zig: src_cells.len; the parameter `len` (which hides Go's builtin
	// here) is that length.
	if len == uint(self.size.cols) {
		src_row.grapheme = false
		src_row.hyperlink = false
		src_row.styled = false
		// Zig: the `if (comptime build_options.kitty_graphics)` block is
		// compiled out (see the file header).
	}
}

// SwapCells is page.Page.swapCells: swap two cells within the same row as
// quickly as possible.
func (self *Page) SwapCells(src *PageCell, dst *PageCell) {
	defer self.AssertIntegrity()

	// Graphemes are keyed by cell offset so we do have to move them.
	// We do this first so that all our grapheme state is correct.
	if src.HasGrapheme() || dst.HasGrapheme() {
		if src.HasGrapheme() && !dst.HasGrapheme() {
			self.MoveGrapheme(src, dst)
		} else if !src.HasGrapheme() && dst.HasGrapheme() {
			self.MoveGrapheme(dst, src)
		} else {
			// Both had graphemes, so we have to manually swap
			src_offset := self.cellOffset(src)
			dst_offset := self.cellOffset(dst)
			m := self.grapheme_map.Map()
			src_entry := m.GetEntry(src_offset)
			assert(src_entry.Valid) // .?
			dst_entry := m.GetEntry(dst_offset)
			assert(dst_entry.Valid) // .?
			src_value := *src_entry.Value.value_ptr
			dst_value := *dst_entry.Value.value_ptr
			*src_entry.Value.value_ptr = dst_value
			*dst_entry.Value.value_ptr = src_value
		}
	}

	// Hyperlinks are keyed by cell offset.
	if src.hyperlink || dst.hyperlink {
		if src.hyperlink && !dst.hyperlink {
			self.moveHyperlink(src, dst)
		} else if !src.hyperlink && dst.hyperlink {
			self.moveHyperlink(dst, src)
		} else {
			// Both had hyperlinks, so we have to manually swap
			src_offset := self.cellOffset(src)
			dst_offset := self.cellOffset(dst)
			m := self.hyperlink_map.Map()
			src_entry := m.GetEntry(src_offset)
			assert(src_entry.Valid) // .?
			dst_entry := m.GetEntry(dst_offset)
			assert(dst_entry.Valid) // .?
			src_value := *src_entry.Value.value_ptr
			dst_value := *dst_entry.Value.value_ptr
			*src_entry.Value.value_ptr = dst_value
			*dst_entry.Value.value_ptr = src_value
		}
	}

	// Copy the metadata. Note that we do NOT have to worry about
	// styles because styles are keyed by ID and we're preserving the
	// exact ref count and row state here.
	old_dst := *dst
	*dst = *src
	*src = old_dst
}

// ClearCells is page.Page.clearCells.
//
// Clear the cells in the given row. This will reclaim memory used
// by graphemes and styles. Note that if the style cleared is still
// active, Page cannot know this and it will still be ref counted down.
// The best solution for this is to artificially increment the ref count
// prior to calling this function.
func (self *Page) ClearCells(row *PageRow, left uint, end uint) {
	defer self.AssertIntegrity()

	cells := self.rowCells(row)[left:end]

	// If we have managed memory (styles, graphemes, or hyperlinks)
	// in this row then we go cell by cell and clear them if present.
	if row.grapheme {
		for i := range cells {
			cell := &cells[i]
			if cell.HasGrapheme() {
				self.ClearGrapheme(cell) // @call(.always_inline, clearGrapheme, ...)
			}
		}

		// If we have no left/right scroll region we can be sure
		// that we've cleared all the graphemes, so we clear the
		// flag, otherwise we use the update function to update.
		if len(cells) == int(self.size.cols) {
			row.grapheme = false
		} else {
			self.UpdateRowGraphemeFlag(row)
		}
	}

	if row.hyperlink {
		for i := range cells {
			cell := &cells[i]
			if cell.hyperlink {
				self.ClearHyperlink(cell) // @call(.always_inline, clearHyperlink, ...)
			}
		}

		// If we have no left/right scroll region we can be sure
		// that we've cleared all the hyperlinks, so we clear the
		// flag, otherwise we use the update function to update.
		if len(cells) == int(self.size.cols) {
			row.hyperlink = false
		} else {
			self.UpdateRowHyperlinkFlag(row)
		}
	}

	if row.styled {
		for i := range cells {
			cell := &cells[i]
			if cell.HasStyling() {
				self.styles.Release(cell.style_id)
			}
		}

		// If we have no left/right scroll region we can be sure
		// that we've cleared all the styles, so we clear the
		// flag, otherwise we use the update function to update.
		if len(cells) == int(self.size.cols) {
			row.styled = false
		} else {
			self.UpdateRowStyledFlag(row)
		}
	}

	// Zig: the `if (comptime build_options.kitty_graphics)` block is
	// compiled out (see the file header).

	// Zero the cells as u64s since empirically this seems
	// to be a bit faster than using @memset(cells, .{})
	clear(cells)
}

// LookupHyperlink is page.Page.lookupHyperlink: returns the hyperlink ID
// for the given cell.
//
// Zig: self is *const Page, cell is *const Cell; returns ?hyperlink.Id.
func (self *Page) LookupHyperlink(cell *PageCell) Optional[HyperlinkID] {
	cell_offset := self.cellOffset(cell)
	m := self.hyperlink_map.Map()
	return m.Get(cell_offset)
}

// ClearHyperlink is page.Page.clearHyperlink.
//
// Clear the hyperlink from the given cell.
//
// In order to update the hyperlink flag on the row, call
// `updateRowHyperlinkFlag` after you finish clearing any
// hyperlinks in the row.
func (self *Page) ClearHyperlink(cell *PageCell) {
	defer self.AssertIntegrity()

	// Get our ID
	cell_offset := self.cellOffset(cell)
	m := self.hyperlink_map.Map()
	entry := m.GetEntry(cell_offset)
	if !entry.Valid {
		return
	}

	// Release our usage of this, free memory, unset flag
	self.hyperlink_set.Release(*entry.Value.value_ptr)
	m.RemoveByPtr(entry.Value.key_ptr)
	cell.hyperlink = false
}

// UpdateRowHyperlinkFlag is page.Page.updateRowHyperlinkFlag.
//
// Checks if the row contains any hyperlinks and sets
// the hyperlink flag to false if none are found.
//
// Call after removing hyperlinks in a row.
func (self *Page) UpdateRowHyperlinkFlag(row *PageRow) {
	cells := self.rowCells(row)[0:self.size.cols]
	for _, c := range cells {
		if c.hyperlink {
			return
		}
	}
	row.hyperlink = false
}

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
func (self *Page) InsertHyperlink(link Hyperlink) (HyperlinkID, error) {
	// Insert our URI into the page strings table.
	// Zig: Offset(u8).Slice
	var page_uri []uint8
	{ // uri:
		buf, err := self.string_alloc.Alloc(
			uint(len(link.URI)),
		)
		if err != nil {
			switch err {
			case ErrOutOfMemory:
				return 0, ErrStringsOutOfMemory
			default:
				panic("unreachable")
			}
		}
		// Zig: errdefer self.string_alloc.free(self.memory, buf): nothing
		// in this block fails after it.
		copy(buf, link.URI) // @memcpy(buf, link.uri)

		// Zig: .{ .offset = size.getOffset(u8, self.memory, &buf[0]),
		// .len = link.uri.len }: the allocation from its first byte.
		page_uri = buf[0:len(link.URI)] // break :uri
	}
	// Zig: errdefer self.string_alloc.free(self.memory,
	// page_uri.slice(self.memory)): written out before the error returns
	// below.

	// Allocate an ID for our page memory if we have to.
	var page_id HyperlinkPageEntryId
	switch link.ID.Tag {
	case HyperlinkHyperlinkIdExplicit:
		// explicit:
		id := link.ID.Explicit
		buf, err := self.string_alloc.Alloc(
			uint(len(id)),
		)
		if err != nil {
			// Zig: errdefer (page_uri)
			self.string_alloc.Free(page_uri)
			switch err {
			case ErrOutOfMemory:
				return 0, ErrStringsOutOfMemory
			default:
				panic("unreachable")
			}
		}
		// Zig: errdefer self.string_alloc.free(self.memory, buf): nothing
		// in this block fails after it.
		copy(buf, id) // @memcpy(buf, id)

		page_id = HyperlinkPageEntryId{ // break :explicit
			Tag:      HyperlinkPageEntryIdExplicit,
			Explicit: buf[0:len(id)],
		}

	case HyperlinkHyperlinkIdImplicit:
		page_id = HyperlinkPageEntryId{Tag: HyperlinkPageEntryIdImplicit, Implicit: link.ID.Implicit}

	default:
		panic("unreachable")
	}
	// Zig: errdefer switch (page_id) { .implicit => {}, .explicit => |slice|
	// self.string_alloc.free(...) }: written out before the error returns
	// below.

	// Build our entry
	entry := HyperlinkPageEntry{
		id:  page_id,
		uri: page_uri,
	}

	// Put our hyperlink into the hyperlink set to get an ID
	id, err := self.hyperlink_set.AddContext(
		entry,
		HyperlinkSetContext{page: self},
	)
	if err != nil {
		// Zig: errdefer (page_id), then errdefer (page_uri).
		switch page_id.Tag {
		case HyperlinkPageEntryIdImplicit:
		case HyperlinkPageEntryIdExplicit:
			self.string_alloc.Free(page_id.Explicit)
		default:
			panic("unreachable")
		}
		self.string_alloc.Free(page_uri)

		switch err {
		case ErrOutOfMemory:
			return 0, ErrSetOutOfMemory
		case ErrNeedsRehash:
			return 0, ErrSetNeedsRehash
		default:
			panic("unreachable")
		}
	}
	// Zig: errdefer self.hyperlink_set.release(self.memory, id): nothing
	// fails after it.

	return id, nil
}

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
func (self *Page) SetHyperlink(row *PageRow, cell *PageCell, id HyperlinkID) error {
	defer self.AssertIntegrity()

	cell_offset := self.cellOffset(cell)
	m := self.hyperlink_map.Map()
	gop, err := m.GetOrPut(cell_offset)
	if err != nil {
		// Zig: comptime assert(@TypeOf(e) == error{OutOfMemory});
		// The hyperlink map capacity needs to be increased.
		return ErrHyperlinkMapOutOfMemory
	}

	if gop.found_existing {
		// Always release the old hyperlink, because even if it's actually
		// the same as the one we're setting, we'd end up double-counting
		// if we left the reference count be, because the caller does not
		// know whether it's the same and will have increased the count
		// outside of this function.
		self.hyperlink_set.Release(*gop.value_ptr)

		// If the hyperlink matches then we don't need to do anything.
		if *gop.value_ptr == id {
			// It is possible for cell hyperlink to be false but row
			// must never be false. The cell hyperlink can be false because
			// in Terminal.print we clear the hyperlink for the cursor cell
			// before writing the cell again, so if someone prints over
			// a cell with a matching hyperlink this state can happen.
			// This is tested in Terminal.zig.
			assert(row.hyperlink)
			cell.hyperlink = true
			return nil
		}
	}

	// Set the hyperlink on the cell and in the map.
	*gop.value_ptr = id
	cell.hyperlink = true
	row.hyperlink = true
	return nil
}

// moveHyperlink is page.Page.moveHyperlink.
//
// Move the hyperlink from one cell to another. This can't fail
// because we avoid any allocations since we're just moving data.
// Destination must NOT have a hyperlink.
func (self *Page) moveHyperlink(src *PageCell, dst *PageCell) {
	assert(src.hyperlink)
	assert(!dst.hyperlink)

	src_offset := self.cellOffset(src)
	dst_offset := self.cellOffset(dst)
	m := self.hyperlink_map.Map()
	entry := m.GetEntry(src_offset)
	assert(entry.Valid) // .?
	value := *entry.Value.value_ptr
	m.RemoveByPtr(entry.Value.key_ptr)
	m.PutAssumeCapacity(dst_offset, value)

	// NOTE: We must not set src/dst.hyperlink here because this
	// function is used in various cases where we swap cell contents
	// and its unsafe. The flip side: the caller must be careful
	// to set the proper cell state to represent the move.
}

// HyperlinkCount is page.Page.hyperlinkCount.
//
// Returns the number of hyperlinks in the page. This isn't the byte
// size but the total number of unique cells that have hyperlink data.
//
// Zig: self is *const Page.
func (self *Page) HyperlinkCount() uint {
	return uint(self.hyperlink_map.Map().Count())
}

// HyperlinkCapacity is page.Page.hyperlinkCapacity.
//
// Returns the hyperlink capacity for the page. This isn't the byte
// size but the number of unique cells that can have hyperlink data.
//
// Zig: self is *const Page.
func (self *Page) HyperlinkCapacity() uint {
	return uint(self.hyperlink_map.Map().Capacity())
}

// SetGraphemes is page.Page.setGraphemes.
//
// Set the graphemes for the given cell. This asserts that the cell
// has no graphemes set, and only contains a single codepoint.
//
// Zig: GraphemeError!void; cps is []const u21.
func (self *Page) SetGraphemes(row *PageRow, cell *PageCell, cps []uint32) error {
	defer self.AssertIntegrity()

	assert(cell.Codepoint() > 0)
	assert(cell.content_tag == PageCellContentTagCodepoint)

	cell_offset := self.cellOffset(cell)
	m := self.grapheme_map.Map()

	slice, err := self.grapheme_alloc.Alloc(uint(len(cps)))
	if err != nil {
		// Zig: comptime assert(@TypeOf(e) == error{OutOfMemory});
		// The grapheme alloc capacity needs to be increased.
		return ErrGraphemeAllocOutOfMemory
	}
	// Zig: errdefer self.grapheme_alloc.free(self.memory, slice): written
	// out before the error return below.
	copy(slice, cps) // @memcpy(slice, cps)

	// Zig: .{ .offset = getOffset(u21, self.memory, @ptrCast(slice.ptr)),
	// .len = slice.len }: the allocation itself.
	if err := m.PutNoClobber(cell_offset, slice[0:len(slice)]); err != nil {
		// Zig: errdefer (slice)
		self.grapheme_alloc.Free(slice)
		// Zig: comptime assert(@TypeOf(e) == error{OutOfMemory});
		// The grapheme map capacity needs to be increased.
		return ErrGraphemeMapOutOfMemory
	}
	// Zig: errdefer map.remove(cell_offset): nothing fails after it.

	cell.content_tag = PageCellContentTagCodepointGrapheme
	row.grapheme = true

	return nil
}

// AppendGrapheme is page.Page.appendGrapheme: append a codepoint to the
// given cell as a grapheme.
//
// Zig: Allocator.Error!void; cp is u21. The error is ErrOutOfMemory
// (dcs.go).
func (self *Page) AppendGrapheme(row *PageRow, cell *PageCell, cp uint32) error {
	defer self.AssertIntegrity()

	if pageSlowRuntimeSafety {
		assert(cell.Codepoint() != 0)
	}

	cell_offset := self.cellOffset(cell)
	m := self.grapheme_map.Map()

	// If this cell has no graphemes, we can go faster by knowing we
	// need to allocate a new grapheme slice and update the map.
	if cell.content_tag != PageCellContentTagCodepointGrapheme {
		cps, err := self.grapheme_alloc.Alloc(1)
		if err != nil {
			return err
		}
		// Zig: errdefer self.grapheme_alloc.free(self.memory, cps): written
		// out before the error return below.
		cps[0] = cp

		// Zig: .{ .offset = getOffset(u21, ..., cps.ptr), .len = 1 }.
		if err := m.PutNoClobber(cell_offset, cps[0:1]); err != nil {
			// Zig: errdefer (cps)
			self.grapheme_alloc.Free(cps)
			return err
		}
		// Zig: errdefer map.remove(cell_offset): nothing fails after it.

		cell.content_tag = PageCellContentTagCodepointGrapheme
		row.grapheme = true

		return nil
	}

	// The cell already has graphemes. We need to append to the existing
	// grapheme slice and update the map.
	assert(row.grapheme)

	slice := m.GetPtr(cell_offset)
	assert(slice != nil) // .?

	// If our slice len doesn't divide evenly by the grapheme chunk
	// length then we can utilize the additional chunk space.
	if len(*slice)%pageGraphemeChunkLen != 0 {
		// Zig: cps = slice.offset.ptr(self.memory); cps[slice.len] = cp;
		// slice.len += 1. Go: the slice grows into its capacity, which
		// reaches the end of its last chunk (BitmapAllocator.Alloc).
		cps := (*slice)[0 : len(*slice)+1]
		cps[len(*slice)] = cp
		*slice = cps
		return nil
	}

	// We are out of chunk space. There is no fast path here. We need
	// to allocate a larger chunk. This is a very slow path. We expect
	// most graphemes to fit within our chunk size.
	cps, err := self.grapheme_alloc.Alloc(uint(len(*slice)) + 1)
	if err != nil {
		return err
	}
	// Zig: errdefer self.grapheme_alloc.free(self.memory, cps): nothing
	// fails after it.
	old_cps := *slice                  // slice.slice(self.memory)
	copy(cps[0:len(old_cps)], old_cps) // fastmem.copy(u21, cps[0..old_cps.len], old_cps)
	cps[len(*slice)] = cp
	// Zig: slice.* = .{ .offset = getOffset(u21, ..., cps.ptr), .len =
	// slice.len + 1 }.
	*slice = cps[0 : len(*slice)+1]

	// Free our old chunk
	self.grapheme_alloc.Free(old_cps)
	return nil
}

// LookupGrapheme is page.Page.lookupGrapheme.
//
// Returns the codepoints for the given cell. These are the codepoints
// in addition to the first codepoint. The first codepoint is NOT
// included since it is on the cell itself.
//
// Zig: self is *const Page, cell is *const Cell; returns ?[]u21, a slice of
// the page's grapheme memory.
func (self *Page) LookupGrapheme(cell *PageCell) Optional[[]uint32] {
	cell_offset := self.cellOffset(cell)
	m := self.grapheme_map.Map()
	slice := m.Get(cell_offset)
	if !slice.Valid {
		return None[[]uint32]()
	}
	return Some(slice.Value) // slice.slice(self.memory)
}

// MoveGrapheme is page.Page.moveGrapheme.
//
// Move the graphemes from one cell to another. This can't fail
// because we avoid any allocations since we're just moving data.
//
// WARNING: This will NOT change the content_tag on the cells because
// there are scenarios where we want to move graphemes without changing
// the content tag. Callers beware but assertIntegrity should catch this.
func (self *Page) MoveGrapheme(src *PageCell, dst *PageCell) {
	if pageSlowRuntimeSafety {
		assert(src.HasGrapheme())
		assert(!dst.HasGrapheme())
	}

	src_offset := self.cellOffset(src)
	dst_offset := self.cellOffset(dst)
	m := self.grapheme_map.Map()
	entry := m.GetEntry(src_offset)
	assert(entry.Valid) // .?
	value := *entry.Value.value_ptr
	m.RemoveByPtr(entry.Value.key_ptr)
	m.PutAssumeCapacity(dst_offset, value)
}

// ClearGrapheme is page.Page.clearGrapheme.
//
// Clear the graphemes for a given cell.
//
// In order to update the grapheme flag on the row, call
// `updateRowGraphemeFlag` after you finish clearing any
// graphemes in the row.
func (self *Page) ClearGrapheme(cell *PageCell) {
	defer self.AssertIntegrity()
	if pageSlowRuntimeSafety {
		assert(cell.HasGrapheme())
	}

	// Get our entry in the map, which must exist
	cell_offset := self.cellOffset(cell)
	m := self.grapheme_map.Map()
	entry := m.GetEntry(cell_offset)
	assert(entry.Valid) // .?

	// Free our grapheme data
	cps := *entry.Value.value_ptr // entry.value_ptr.slice(self.memory)
	self.grapheme_alloc.Free(cps)

	// Remove the entry
	m.RemoveByPtr(entry.Value.key_ptr)

	// Mark that we no longer have graphemes by changing the content tag.
	cell.content_tag = PageCellContentTagCodepoint
}

// UpdateRowGraphemeFlag is page.Page.updateRowGraphemeFlag.
//
// Checks if the row contains any graphemes and sets
// the grapheme flag to false if none are found.
//
// Call after removing graphemes in a row.
func (self *Page) UpdateRowGraphemeFlag(row *PageRow) {
	cells := self.rowCells(row)[0:self.size.cols]
	for _, c := range cells {
		if c.HasGrapheme() {
			return
		}
	}
	row.grapheme = false
}

// GraphemeCount is page.Page.graphemeCount.
//
// Returns the number of graphemes in the page. This isn't the byte
// size but the total number of unique cells that have grapheme data.
//
// Zig: self is *const Page.
func (self *Page) GraphemeCount() uint {
	return uint(self.grapheme_map.Map().Count())
}

// GraphemeCapacity is page.Page.graphemeCapacity.
//
// Returns the grapheme capacity for the page. This isn't the byte
// size but the number of unique cells that can have grapheme data.
//
// Zig: self is *const Page.
func (self *Page) GraphemeCapacity() uint {
	return uint(self.grapheme_map.Map().Capacity())
}

// UpdateRowStyledFlag is page.Page.updateRowStyledFlag.
//
// Checks if the row contains any styles and sets
// the styled flag to false if none are found.
//
// Call after removing styles in a row.
func (self *Page) UpdateRowStyledFlag(row *PageRow) {
	cells := self.rowCells(row)[0:self.size.cols]
	for _, c := range cells {
		if c.HasStyling() {
			return
		}
	}
	row.styled = false
}

// IsDirty is page.Page.isDirty: returns true if this page is dirty at all.
//
// Zig: self is *const Page.
func (self *Page) IsDirty() bool {
	if self.dirty {
		return true
	}
	for _, row := range self.rows[0:self.size.rows] {
		if row.dirty {
			return true
		}
	}
	return false
}

// cellIndex is NOT a Zig declaration. It stands for Zig's pointer
// subtraction against page memory: the index in `self.cells` of the element
// `cell` points at.
//
// The implementation may use package unsafe for this and for nothing else,
// and must panic if the pointer is not inside the page's slice.
func (self *Page) cellIndex(cell *PageCell) uint {
	if len(self.cells) == 0 {
		panic("cell is not in the page")
	}
	base := uintptr(unsafe.Pointer(unsafe.SliceData(self.cells)))
	p := uintptr(unsafe.Pointer(cell))
	size := unsafe.Sizeof(PageCell{})
	if p < base || p >= base+uintptr(len(self.cells))*size || (p-base)%size != 0 {
		panic("cell is not in the page")
	}
	return uint((p - base) / size)
}

// rowIndex is NOT a Zig declaration. It stands for Zig's pointer
// subtraction against page memory: the index in `self.rows` of the element
// `row` points at.
//
// The implementation may use package unsafe for this and for nothing else,
// and must panic if the pointer is not inside the page's slice.
func (self *Page) rowIndex(row *PageRow) uint {
	if len(self.rows) == 0 {
		panic("row is not in the page")
	}
	base := uintptr(unsafe.Pointer(unsafe.SliceData(self.rows)))
	p := uintptr(unsafe.Pointer(row))
	size := unsafe.Sizeof(PageRow{})
	if p < base || p >= base+uintptr(len(self.rows))*size || (p-base)%size != 0 {
		panic("row is not in the page")
	}
	return uint((p - base) / size)
}

// cellOffset is NOT a Zig declaration. It stands for
// `getOffset(Cell, self.memory, cell)`: the byte offset of the cell in the
// page as upstream would have it, `layout.cells_start + cellIndex(cell)*8`.
// This is the key of grapheme_map and hyperlink_map.
func (self *Page) cellOffset(cell *PageCell) SizeOffsetInt {
	offset := pageCellsStart(self.capacity) + uint64(self.cellIndex(cell))*pageSizeOfCell
	// Zig: getOffset's @intCast to OffsetInt traps when it does not fit.
	assert(offset <= math.MaxUint32)
	return SizeOffsetInt(offset)
}

// pageCellsStart is NOT a Zig declaration: `Page.layout(cap).cells_start`,
// computed the way layout computes it, without the rest of the layout
// (cellOffset and rowCells need it on every call).
func pageCellsStart(cap PageCapacity) uint64 {
	rows_end := uint64(cap.rows) * pageSizeOfRow
	return pageAlignForward(rows_end, pageAlignOfCell)
}

// rowCells is NOT a Zig declaration. It stands for
// `row.cells.ptr(self.memory)`, the many-item pointer to the row's first
// cell: the page's cells from that cell to the end of `cells`. Callers
// slice it as Zig slices the pointer.
func (self *Page) rowCells(row *PageRow) []PageCell {
	start := (uint64(row.cells) - pageCellsStart(self.capacity)) / pageSizeOfCell
	return self.cells[start:]
}

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
	styles_layout         RefCountedSetLayout[StyleID]
	grapheme_alloc_start  uint64
	grapheme_alloc_layout BitmapAllocatorLayout
	grapheme_map_start    uint64
	grapheme_map_layout   HashMapOffsetHashMapLayout
	string_alloc_start    uint64
	string_alloc_layout   BitmapAllocatorLayout
	hyperlink_map_start   uint64
	hyperlink_map_layout  HashMapOffsetHashMapLayout
	hyperlink_set_start   uint64
	hyperlink_set_layout  RefCountedSetLayout[HyperlinkID]
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
func PageLayoutOf(cap PageCapacity) PageLayout {
	rows_count := uint64(cap.rows) // @intCast(cap.rows)
	const rows_start = 0
	rows_end := rows_start + (rows_count * pageSizeOfRow)

	cells_count := uint64(cap.cols) * uint64(cap.rows)
	cells_start := pageAlignForward(rows_end, pageAlignOfCell)
	cells_end := cells_start + (cells_count * pageSizeOfCell)

	styles_layout := RefCountedSetLayoutInit[StyleID](uint(cap.styles), styleSetItemSizes)
	styles_start := pageAlignForward(cells_end, pageStyleSetBaseAlign)
	styles_end := styles_start + styles_layout.total_size

	grapheme_alloc_layout := BitmapAllocatorLayoutOf(pageGraphemeChunk, uint64(cap.grapheme_bytes))
	grapheme_alloc_start := pageAlignForward(styles_end, pageGraphemeAllocBaseAlign)
	grapheme_alloc_end := grapheme_alloc_start + grapheme_alloc_layout.total_size

	var grapheme_count uint64
	{ // count:
		if cap.grapheme_bytes == 0 {
			grapheme_count = 0 // break :count 0
		} else {
			// Use divCeil to match GraphemeAlloc.layout() which uses alignForward,
			// ensuring grapheme_map has capacity when grapheme_alloc has chunks.
			//
			// Zig: divCeil(...) catch unreachable (the divisor is not
			// zero) and ceilPowerOfTwo(usize, base) catch unreachable.
			base := pageStdMathDivCeil(uint64(cap.grapheme_bytes), pageGraphemeChunk)
			grapheme_count = pageStdMathCeilPowerOfTwo(base, 64) // break :count
		}
	}
	// Zig: @intCast(grapheme_count) to Size (u32) traps when it does not
	// fit.
	assert(grapheme_count <= math.MaxUint32)
	grapheme_map_layout := HashMapOffsetHashMapLayoutOf(HashMapSize(grapheme_count), pageGraphemeMapEntrySizes)
	grapheme_map_start := pageAlignForward(grapheme_alloc_end, pageGraphemeMapBaseAlign)
	grapheme_map_end := grapheme_map_start + grapheme_map_layout.total_size

	string_layout := BitmapAllocatorLayoutOf(pageStringChunk, uint64(cap.string_bytes))
	string_start := pageAlignForward(grapheme_map_end, pageStringAllocBaseAlign)
	string_end := string_start + string_layout.total_size

	// Zig: @divFloor of the u16 by @sizeOf(hyperlink.Set.Item): a u16.
	hyperlink_count := cap.hyperlink_bytes / hyperlinkSetItemSizeOf
	hyperlink_set_layout := RefCountedSetLayoutInit[HyperlinkID](uint(hyperlink_count), hyperlinkSetItemSizes)
	hyperlink_set_start := pageAlignForward(string_end, pageHyperlinkSetBaseAlign)
	hyperlink_set_end := hyperlink_set_start + hyperlink_set_layout.total_size

	var hyperlink_map_count uint32
	{ // count:
		if hyperlink_count == 0 {
			hyperlink_map_count = 0 // break :count 0
		} else {
			// Zig: `hyperlink_count * hyperlink_cell_multiplier` is a u16
			// product (at most 65535 / 48 * 16 = 21840: it cannot
			// overflow), so std.math.cast(u32, ...) always succeeds and
			// its `orelse break :count std.math.maxInt(u32)` is never
			// taken.
			mult := uint32(hyperlink_count * pageHyperlinkCellMultiplier)
			hyperlink_map_count = uint32(pageStdMathCeilPowerOfTwo(uint64(mult), 32)) // ceilPowerOfTwoAssert(u32, mult)
		}
	}
	hyperlink_map_layout := HashMapOffsetHashMapLayoutOf(hyperlink_map_count, hyperlinkMapEntrySizes)
	hyperlink_map_start := pageAlignForward(hyperlink_set_end, pageHyperlinkMapBaseAlign)
	hyperlink_map_end := hyperlink_map_start + hyperlink_map_layout.total_size

	total_size := pageAlignForward(hyperlink_map_end, pagePageSizeMin)

	return PageLayout{
		total_size:            total_size,
		rows_start:            rows_start,
		rows_size:             rows_end - rows_start,
		cells_start:           cells_start,
		cells_size:            cells_end - cells_start,
		styles_start:          styles_start,
		styles_layout:         styles_layout,
		grapheme_alloc_start:  grapheme_alloc_start,
		grapheme_alloc_layout: grapheme_alloc_layout,
		grapheme_map_start:    grapheme_map_start,
		grapheme_map_layout:   grapheme_map_layout,
		string_alloc_start:    string_start,
		string_alloc_layout:   string_layout,
		hyperlink_map_start:   hyperlink_map_start,
		hyperlink_map_layout:  hyperlink_map_layout,
		hyperlink_set_start:   hyperlink_set_start,
		hyperlink_set_layout:  hyperlink_set_layout,
		capacity:              cap,
	}
}

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
	return PageCapacity{
		cols:            cols,
		rows:            rows,
		styles:          16,
		hyperlink_bytes: pageHyperlinkBytesDefault,
		grapheme_bytes:  PageGraphemeBytesDefault,
		string_bytes:    PageStringBytesDefault,
	}
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
func (self PageCapacity) MaxCols() Optional[SizeCellCountInt] {
	available_bits := self.availableBitsForGrid()

	// If we can't even fit the row metadata, return null
	if available_bits <= pageBitSizeOfRow {
		return None[SizeCellCountInt]()
	}

	// We do the math of how many columns we can fit in the remaining
	// bits ignoring the metadata of a row.
	remaining_bits := available_bits - pageBitSizeOfRow
	max_cols := remaining_bits / pageBitSizeOfCell

	// Clamp to CellCountInt max
	return Some(SizeCellCountInt(min(math.MaxUint16, max_cols)))
}

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
	adjusted := self
	if req.cols.Valid {
		cols := req.cols.Value
		available_bits := self.availableBitsForGrid()

		// The size per row is:
		//   - The row metadata itself
		//   - The cells per row (n=cols)
		bits_per_row := uint64(pageBitSizeOfRow) + pageBitSizeOfCell*uint64(cols)
		new_rows := available_bits / bits_per_row // @divFloor

		// If our rows go to zero then we can't fit any row metadata
		// for the desired number of columns.
		if new_rows == 0 {
			return PageCapacity{}, ErrOutOfMemory
		}

		adjusted.cols = cols
		// Zig: @intCast(new_rows) traps when it does not fit.
		assert(new_rows <= math.MaxUint16)
		adjusted.rows = SizeCellCountInt(new_rows)
	}

	return adjusted, nil
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
func (self PageCapacity) availableBitsForGrid() uint64 {
	// The math below only works if there is no alignment gap between
	// the end of the rows array and the start of the cells array.
	//
	// To guarantee this, we assert that Row's size is a multiple of
	// Cell's alignment, so that any length array of Rows will end on
	// a valid alignment for the start of the Cell array.
	assert(pageSizeOfRow%pageAlignOfCell == 0)

	l := PageLayoutOf(self)

	// Layout meta members from the end to find styles_start
	//
	// Zig: each `-` is a usize subtraction that would trap on underflow;
	// none can: every start found here is at or after the start
	// Page.layout gave the same member.
	hyperlink_map_start := pageAlignBackward(l.total_size-l.hyperlink_map_layout.total_size, pageHyperlinkMapBaseAlign)
	hyperlink_set_start := pageAlignBackward(hyperlink_map_start-l.hyperlink_set_layout.total_size, pageHyperlinkSetBaseAlign)
	string_alloc_start := pageAlignBackward(hyperlink_set_start-l.string_alloc_layout.total_size, pageStringAllocBaseAlign)
	grapheme_map_start := pageAlignBackward(string_alloc_start-l.grapheme_map_layout.total_size, pageGraphemeMapBaseAlign)
	grapheme_alloc_start := pageAlignBackward(grapheme_map_start-l.grapheme_alloc_layout.total_size, pageGraphemeAllocBaseAlign)
	styles_start := pageAlignBackward(grapheme_alloc_start-l.styles_layout.total_size, pageStyleSetBaseAlign)

	// Multiply by 8 to convert bytes to bits
	return styles_start * 8
}

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

// pageRowSemanticPromptNames are the tag names of PageRowSemanticPrompt,
// indexed by value (Zig's @tagName).
var pageRowSemanticPromptNames = [...]string{"none", "prompt", "prompt_continuation"}

// String returns the Zig tag name (@tagName).
func (self PageRowSemanticPrompt) String() string {
	return pageRowSemanticPromptNames[self]
}

// ManagedMemory is page.Row.managedMemory.
//
// Returns true if this row has any managed memory outside of the
// row structure (graphemes, styles, etc.)
func (self PageRow) ManagedMemory() bool {
	// Ordered on purpose for likelihood.
	return self.styled || self.hyperlink || self.grapheme
}

// bitCast is not a Zig declaration. It stands for `@as(u64, @bitCast(self))`:
// the 64 bits of the packed struct, laid out as described on PageRow, with
// `cells` as the upstream byte offset.
func (self PageRow) bitCast() uint64 {
	// Fields narrower than their Go type keep only the bits of their Zig
	// width, as the packed layout would (wider values cannot exist in
	// Zig).
	var v uint64
	v |= uint64(self.cells)
	v |= pageBoolBit(self.wrap) << 32
	v |= pageBoolBit(self.wrap_continuation) << 33
	v |= pageBoolBit(self.grapheme) << 34
	v |= pageBoolBit(self.styled) << 35
	v |= pageBoolBit(self.hyperlink) << 36
	v |= (uint64(self.semantic_prompt) & 0x3) << 37
	v |= pageBoolBit(self.kitty_virtual_placeholder) << 39
	v |= pageBoolBit(self.dirty) << 40
	v |= (uint64(self._padding) & 0x7FFFFF) << 41
	return v
}

// pageRowFromBits is not a Zig declaration. It stands for
// `@as(Row, @bitCast(bits))`: the inverse of PageRow.bitCast.
func pageRowFromBits(bits uint64) PageRow {
	return PageRow{
		cells:                     SizeOffsetInt(bits & 0xFFFFFFFF),
		wrap:                      bits>>32&1 != 0,
		wrap_continuation:         bits>>33&1 != 0,
		grapheme:                  bits>>34&1 != 0,
		styled:                    bits>>35&1 != 0,
		hyperlink:                 bits>>36&1 != 0,
		semantic_prompt:           PageRowSemanticPrompt(bits >> 37 & 0x3),
		kitty_virtual_placeholder: bits>>39&1 != 0,
		dirty:                     bits>>40&1 != 0,
		_padding:                  uint32(bits >> 41 & 0x7FFFFF),
	}
}

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
	style_id StyleID

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

// pageCellContentTagNames are the tag names of PageCellContentTag, indexed
// by value (Zig's @tagName).
var pageCellContentTagNames = [...]string{"codepoint", "codepoint_grapheme", "bg_color_palette", "bg_color_rgb"}

// String returns the Zig tag name (@tagName).
func (self PageCellContentTag) String() string {
	return pageCellContentTagNames[self]
}

// PageCellRGB is page.Cell.RGB.
//
// Zig: packed struct (24 bits): r is bits 0..7, g 8..15, b 16..23.
type PageCellRGB struct {
	R uint8
	G uint8
	B uint8
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

// pageCellWideNames are the tag names of PageCellWide, indexed by value
// (Zig's @tagName).
var pageCellWideNames = [...]string{"narrow", "wide", "spacer_tail", "spacer_head"}

// String returns the Zig tag name (@tagName).
func (self PageCellWide) String() string {
	return pageCellWideNames[self]
}

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

// pageCellSemanticContentNames are the tag names of PageCellSemanticContent,
// indexed by value (Zig's @tagName).
var pageCellSemanticContentNames = [...]string{"output", "input", "prompt"}

// String returns the Zig tag name (@tagName).
func (self PageCellSemanticContent) String() string {
	return pageCellSemanticContentNames[self]
}

// PageCellInit is page.Cell.init: helper to make a cell that just has a
// codepoint.
//
// Zig: pub fn init(cp: u21) Cell.
func PageCellInit(cp uint32) PageCell {
	// We have to use this bitCast here to ensure that our memory is
	// zeroed. Otherwise, the content below will leave some uninitialized
	// memory in the packed union. Valgrind verifies this.
	//
	// Zig: `var cell: Cell = @bitCast(@as(u64, 0))`: the Go zero value.
	var cell PageCell
	// Zig: cp is a u21.
	assert(cp <= 0x1FFFFF)
	cell.content_tag = PageCellContentTagCodepoint
	cell.content = PageCellContent{codepoint: cp}
	return cell
}

// bitCast is not a Zig declaration. It stands for `@as(u64, @bitCast(self))`
// in Cell.isZero (page.zig line 2066): the 64 bits of the packed struct,
// laid out as described on PageCell.
func (self PageCell) bitCast() uint64 {
	// The 24 content bits: the view content_tag selects gives its own
	// bits; the bits above that view come from color_rgb, the one view
	// that covers all 24 (see PageCellContent).
	var content uint64
	rgb := uint64(self.content.color_rgb.R) |
		uint64(self.content.color_rgb.G)<<8 |
		uint64(self.content.color_rgb.B)<<16
	switch self.content_tag & 0x3 {
	case PageCellContentTagCodepoint, PageCellContentTagCodepointGrapheme:
		content = uint64(self.content.codepoint)&0x1FFFFF | rgb&^0x1FFFFF
	case PageCellContentTagBgColorPalette:
		content = uint64(self.content.color_palette) | rgb&^0xFF
	case PageCellContentTagBgColorRgb:
		content = rgb
	}

	// Fields narrower than their Go type keep only the bits of their Zig
	// width, as the packed layout would (wider values cannot exist in
	// Zig).
	var v uint64
	v |= uint64(self.content_tag) & 0x3
	v |= content << 2
	v |= uint64(self.style_id) << 26
	v |= (uint64(self.wide) & 0x3) << 42
	v |= pageBoolBit(self.protected) << 44
	v |= pageBoolBit(self.hyperlink) << 45
	v |= (uint64(self.semantic_content) & 0x3) << 46
	v |= uint64(self._padding) << 48
	return v
}

// pageCellFromBits is not a Zig declaration. It stands for
// `@as(Cell, @bitCast(bits))`: the inverse of PageCell.bitCast. Bits that
// carry no meaning in the packed struct are dropped.
func pageCellFromBits(bits uint64) PageCell {
	content := uint32(bits >> 2 & 0xFFFFFF)
	return PageCell{
		content_tag: PageCellContentTag(bits & 0x3),
		// Every view reads the same 24 bits, as in the packed union.
		content: PageCellContent{
			codepoint:     content & 0x1FFFFF,
			color_palette: uint8(content),
			color_rgb: PageCellRGB{
				R: uint8(content),
				G: uint8(content >> 8),
				B: uint8(content >> 16),
			},
		},
		style_id:         StyleID(bits >> 26 & 0xFFFF),
		wide:             PageCellWide(bits >> 42 & 0x3),
		protected:        bits>>44&1 != 0,
		hyperlink:        bits>>45&1 != 0,
		semantic_content: PageCellSemanticContent(bits >> 46 & 0x3),
		_padding:         uint16(bits >> 48),
	}
}

// IsZero is page.Cell.isZero.
func (self PageCell) IsZero() bool {
	return self.bitCast() == 0
}

// HasText is page.Cell.hasText.
//
// Returns true if this cell represents a cell with text to render.
//
// Cases this returns false:
//   - Cell text is blank
//   - Cell is styled but only with a background color and no text
//   - Cell has a unicode placeholder for Kitty graphics protocol
func (self PageCell) HasText() bool {
	switch self.content_tag {
	case PageCellContentTagCodepoint,
		PageCellContentTagCodepointGrapheme:
		return self.content.codepoint != 0

	case PageCellContentTagBgColorPalette,
		PageCellContentTagBgColorRgb:
		return false

	default:
		panic("unreachable")
	}
}

// Codepoint is page.Cell.codepoint.
//
// Zig: returns u21.
func (self PageCell) Codepoint() uint32 {
	switch self.content_tag {
	case PageCellContentTagCodepoint,
		PageCellContentTagCodepointGrapheme:
		return self.content.codepoint

	case PageCellContentTagBgColorPalette,
		PageCellContentTagBgColorRgb:
		return 0

	default:
		panic("unreachable")
	}
}

// GridWidth is page.Cell.gridWidth: the width in grid cells that this cell
// takes up.
//
// Zig: returns u2.
func (self PageCell) GridWidth() uint8 {
	switch self.wide {
	case PageCellWideNarrow, PageCellWideSpacerHead, PageCellWideSpacerTail:
		return 1
	case PageCellWideWide:
		return 2
	default:
		panic("unreachable")
	}
}

// HasStyling is page.Cell.hasStyling.
func (self PageCell) HasStyling() bool {
	return self.style_id != StyleDefaultID
}

// IsEmpty is page.Cell.isEmpty: returns true if the cell has no text or
// styling.
func (self PageCell) IsEmpty() bool {
	switch self.content_tag {
	// Textual cells are empty if they have no text and are narrow.
	// The "narrow" requirement is because wide spacers are meaningful.
	case PageCellContentTagCodepoint,
		PageCellContentTagCodepointGrapheme:
		return !self.HasText() && self.wide == PageCellWideNarrow

	case PageCellContentTagBgColorPalette,
		PageCellContentTagBgColorRgb:
		return false

	default:
		panic("unreachable")
	}
}

// HasGrapheme is page.Cell.hasGrapheme.
func (self PageCell) HasGrapheme() bool {
	return self.content_tag == PageCellContentTagCodepointGrapheme
}

// PageCellHasTextAny is page.Cell.hasTextAny: returns true if the set of
// cells has text in it.
//
// Zig: pub inline fn hasTextAny(cells: []const Cell) bool, a function of
// the type.
func PageCellHasTextAny(cells []PageCell) bool {
	for _, cell := range cells {
		if cell.HasText() {
			return true
		}
	}

	return false
}

// pageBoolBit is NOT a Zig declaration: a bool field's bit in a packed
// struct (bitCast).
func pageBoolBit(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// pageAlignForward is std.mem.alignForward(usize, addr, alignment) (Zig
// 0.15.2) on uint64, the width of the recorded byte arithmetic.
func pageAlignForward(addr uint64, alignment uint64) uint64 {
	// std: assert(isValidAlignGeneric(T, alignment))
	assert(alignment > 0 && alignment&(alignment-1) == 0)
	// Zig: `addr + (alignment - 1)` traps on overflow.
	assert(addr <= math.MaxUint64-(alignment-1))
	return pageAlignBackward(addr+(alignment-1), alignment)
}

// pageAlignBackward is std.mem.alignBackward(usize, addr, alignment) (Zig
// 0.15.2) on uint64.
func pageAlignBackward(addr uint64, alignment uint64) uint64 {
	// std: assert(isValidAlignGeneric(T, alignment))
	assert(alignment > 0 && alignment&(alignment-1) == 0)
	// 000010000 // example alignment
	// 000001111 // subtract 1
	// 111110000 // binary not
	return addr & ^(alignment - 1)
}

// pageStdMathDivCeil is std.math.divCeil(usize, numerator, denominator)
// (Zig 0.15.2) for unsigned operands. Every caller divides by a non-zero
// constant, so the DivisionByZero error is a precondition here.
func pageStdMathDivCeil(numerator uint64, denominator uint64) uint64 {
	assert(denominator != 0) // Zig: return error.DivisionByZero
	if numerator > 0 && denominator > 0 {
		return (numerator-1)/denominator + 1
	}
	return numerator / denominator
}

// pageStdMathCeilPowerOfTwo is std.math.ceilPowerOfTwo(T, value) (Zig
// 0.15.2) for an unsigned T of `int_bits` bits (32 or 64), with the
// callers' `catch unreachable` (ceilPowerOfTwoAssert): a result that does not
// fit T panics. ceilPowerOfTwoPromote asserts value != 0.
func pageStdMathCeilPowerOfTwo(value uint64, int_bits int) uint64 {
	assert(value != 0)
	// Zig: @as(PromotedType, 1) << (bits - @clz(value - 1)) in bits + 1
	// bits, then error.Overflow when the bit above T is set.
	shift := bits.Len64(value - 1)
	assert(shift < int_bits) // catch unreachable
	return uint64(1) << shift
}
