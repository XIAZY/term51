// WIREFRAME for src/terminal/PageList.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// Maintains a linked list of pages to make up a terminal screen
// and provides higher level operations on top of those pages to
// make it slightly easier to work with.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: PageList.zig is a file-level struct, so the type is
// `PageList` and its constructor is `PageListInit`. What is nested in it
// takes the prefix `PageList`: `PageList.Pin` is `PageListPin`,
// `PageList.List` is `PageListList`, `PageList.Node` is `PageListNode`,
// `PageList.Viewport` is `PageListViewport`, `PageList.Cell` is
// `PageListCell`, `PageList.Scroll` is `PageListScroll`,
// `PageList.PageIterator.Chunk` is `PageListPageIteratorChunk`. Private
// types, private free functions and private file-level constants take the
// lowerCamel prefix `pageList`: `ReflowCursor` is `pageListReflowCursor`,
// `PinSet` is `pageListPinSet`, `minMaxSize` is `pageListMinMaxSize`,
// `std_size` is `pageListStdSize`, `page_preheat` is `pageListPagePreheat`.
// Private methods keep their Zig name (`resizeCols`, `pinIsActive`); `pub fn`
// methods are exported (`Resize`, `PinIsValid`).
//
// `Node` is private in Zig ("you can access the type via List.Node") but is
// reachable by every user as `List.Node`, so it is exported: `PageListNode`.
//
// Declarations appear in the same order as in PageList.zig.
//
// # Go-memory rule, option A
//
// (docs/mapping.md, governing rule 3, as refined for page.go.) Ordinary Go
// memory: slices, pointers, the garbage collector; no raw byte block. But
// upstream's bookkeeping is kept exactly, because the scrollback limit is
// counted in bytes of page memory: `page_size`, `explicit_max_size`,
// `min_max_size` and `std_size` hold the byte counts upstream would hold,
// and a page is pruned, reused, compacted or treated as non-standard at the
// same byte thresholds.
//
// The memory pools. Zig has `MemoryPool { alloc, nodes: NodePool, pages:
// PagePool, pins: PinPool }`, three std.heap memory pools that hand out raw
// nodes, raw page buffers of `std_size` bytes and raw pins, preheated at
// init. Go allocates a node with `new(PageListNode)`, a pin with
// `new(PageListPin)` and a page with the page layer, and the garbage
// collector frees them. What is dropped:
//   - the pools' actual memory, their preheating and their retention on
//     reset: `pageListNodePool`, `pageListPagePool` and `pageListPinPool`
//     are empty structs, kept only so that `MemoryPool` keeps its three
//     fields. The std methods called on them in Zig (`initPreheated`,
//     `create`, `destroy`, `deinit`, `reset`, `arena.child_allocator`) are
//     std library, not PageList.zig, and have no Go counterpart;
//   - `MemoryPool.alloc` (the general allocator) and every allocator
//     parameter;
//   - zeroing of reused buffers (`@memset(page_buf, 0)`, the BufNode walk in
//     `reset`): the Go implementation gives a reused page fresh, zeroed
//     slices.
// No field of the pools is read for a decision in PageList.zig (the preheat
// count is only passed through; there is no "owned" flag in this version),
// so `PageListMemoryPool` carries no bookkeeping field. `MemoryPool.init`,
// `deinit` and `reset` are declared in PageList.zig and are kept as stubs.
// `PagePool.item_size` IS read for decisions (`minMaxSize`, `grow`) and by
// tests; it is the variable `pageListPagePoolItemSize`.
//
// `Page.memory.len`. Zig reads the byte length of a page's buffer to keep
// `page_size` and to tell a standard (pooled) page from a non-standard one
// (`memory.len > std_size`) in initPages, deinit, reset, clone, compact,
// grow, createPageExt and destroyNodeExt. Page (page.go) has no `memory`.
// The stub `pageListMemoryLen`, which is NOT a Zig declaration, stands for
// `page.memory.len`: for a page owned by a PageList that is `std_size` if
// `Page.layout(capacity).total_size <= std_size` (a pooled buffer is always
// `std_size` long, whatever its layout needs) and `total_size` otherwise.
//
// The page list. Zig: `List = IntrusiveDoublyLinkedList(Node)`
// (src/datastruct/intrusive_linked_list.zig, not std), `Node { prev, next,
// data: Page, serial }`. Go: `PageListList { first, last *PageListNode }`
// and `PageListNode` with the same four fields, so `s.pages.first`,
// `node.next`, `node.prev`, `node.data` and `node.serial` translate
// directly. The six list operations PageList.zig calls are stubs on
// `PageListList` with the names of that file, exported because they are
// `pub`: `InsertAfter`, `InsertBefore`, `Append`, `Prepend`, `Remove`,
// `PopFirst`. (`pop` is not used and is not declared.)
//
// Tracked pins. Zig: `PinSet = std.AutoArrayHashMapUnmanaged(*Pin, void)`,
// an insertion-ordered set of pin pointers. Go: `pageListPinSet`, a slice
// in insertion order, with stubs for the operations PageList.zig uses under
// their std names: `PutNoClobber`, `SwapRemove`, `Keys`, `Count`. Not
// declared: `deinit` (only frees), `iterator` (used once, in `reset`; the
// Go code ranges over `Keys()`), `remove` (used once, in an `errdefer` in
// `trackPin` that follows the last failable statement and so never runs).
// A tracked pin is a `*PageListPin` allocated by Go.
//
// Pointers into pages. `ReflowCursor.page_row` and `page_cell`, `Cell.row`
// and `Cell.cell` and the result of `Pin.rowAndCell` are Go pointers to
// elements of the page's `rows` and `cells` slices. Where Zig steps such a
// pointer (`cell + 1`, `rows - n` in cursorForward, cursorScroll,
// cursorAbsolute, countTrailingEmptyCells) the Go implementation indexes
// the slice; `Page.cellIndex` and `Page.rowIndex` (page.go) give the index
// of a pointer. `fastmem.rotateOnce` (eraseRow, eraseRowBounded) is a
// rotation of a Go sub-slice.
//
// # Widths
//
// As in page.go: every byte quantity is a `uint64` whatever the platform
// word size, counts and indexes are `uint` (Zig `usize`), `isize` is `int`,
// and fixed Zig widths stay fixed. Byte quantities, all `uint64`:
//   - fields `PageList.page_size`, `explicit_max_size`, `min_max_size`;
//   - `pageListStdSize` (`std_size`), `pageListPagePoolItemSize`
//     (`PagePool.item_size`);
//   - the `max_size` parameter of `PageListInit` (`?usize`), the result of
//     `pageListMinMaxSize` and of `PageList.MaxSize`;
//   - the second result of `pageListInitPages`;
//   - the `total_size` parameters of `pageListCreatePageExt` and
//     `pageListDestroyNodeExt` (`?*usize` -> `*uint64`);
//   - `PageListMemoryPoolResetMode.RetainWithLimit`;
//   - the result of `pageListMemoryLen`.
// `explicit_max_size` is `max_size orelse std.math.maxInt(usize)`; the
// reference is the recorded 64-bit data, so "no limit" is the maximum
// uint64, also on a 32-bit target.
// Counts (`uint`): `total_rows`, `viewport_pin_row_offset`, the preheat
// count, `Scroll.row`, every field of `Scrollbar`, `ReflowCursor.new_rows`
// and `total_rows`, the row/column amounts taken by the `Pin` movement
// methods, `PageIterator.Limit.count`, the results of `totalRows`,
// `totalPages`, `countTrackedPins`, `viewportRowOffset`,
// `countTrailingEmptyCells`, the parameters of `growRows`, `fixupViewport`
// and `eraseRowBounded`. `page_serial`, `page_serial_min` and `Node.serial`
// are `u64` in Zig and `uint64` here (fixed width, not bytes).
//
// # Declarations that are not Zig declarations
//
//   - `pageListMemoryLen`: `page.memory.len`, see above;
//   - `pageListPagePoolItemSize`: `PagePool.item_size`;
//   - the methods of `PageListList` and `pageListPinSet`: the datastruct
//     and std operations PageList.zig calls;
//   - `pageListReflowCursorWriteCellResult`: the anonymous enum
//     `ReflowCursor.writeCell` returns;
//   - `PageListPinOverflow` and its tag: the anonymous union(enum)
//     `Pin.downOverflow` and `Pin.upOverflow` return. They are two
//     separate but identical anonymous types in Zig and one type here;
//   - the `...Tag` types of the tagged unions and the `String` methods.
//
// # Omitted
//
// Raw memory (Go-memory rule):
//   - `pageAllocator()` (line 322): returns the allocator for page buffers
//     and does nothing else.
//   - No other function is omitted under this rule. `initPages`,
//     `createPage`, `createPageExt`, `destroyNode` and `destroyNodeExt`
//     obtain or return buffers in Zig but also keep `page_size`, the serial
//     and the list, so they are declared.
// Test-only fault injection, as in Tabstops.go:
//   - `init_tw` and `initPages_tw` (`tripwire.module(...)`) and the
//     `tw.check(...)` calls in `init` and `initPages`. There is no
//     allocator in the Go port to fail.
// C ABI:
//   - `Scrollbar.C` and `Scrollbar.cval`.
// Compiled out (`build_options.kitty_graphics` is false in the library
// build):
//   - the block in `ReflowCursor.writeCell` that copies the Kitty virtual
//     placeholder flag, and the `kitty` import.
// Cannot compile upstream:
//   - `diagram` (line 4049) and the `lt` function nested in it: the body
//     does not compile with this Zig version (0.15.2) and so is never
//     referenced by any build; the tests only quote its output in comments.
// Waiting for another wireframe (see the note at its place in the file):
//   - `highlightSemanticContent` (line 4252) returns `?highlight.Untracked`
//     and there is no highlight.go yet. The type is not invented here.
// Aliases and imports with nothing to declare:
//   - `std_capacity` (`pagepkg.std_capacity`): use `PageStdCapacity`;
//     `Capacity`, `Page`, `Row`, `OffsetBuf`; `color` and `fastmem`
//     imports; the `log` scope.
//
// # Build options
//
// `build_options.slow_runtime_safety` is the constant
// `pageSlowRuntimeSafety` (page.go). `pause_integrity_checks` is `usize`
// when it is set and `void` otherwise; the field is kept as a `uint`.
//
// # Zig defaults a Go literal does not give
//
// `Resize.reflow` defaults to TRUE in Zig. A Go composite literal leaves it
// false, so a translated `s.resize(.{ .cols = 5 })` must write
// `reflow: true`. All other defaults in this file are the Go zero value
// (`null`, 0, false, the first enum value) and are noted on the fields.

package terminal

import "errors"

// pageListPagePreheat is PageList.page_preheat.
//
// The number of PageList.Nodes we preheat the pool with. A node is
// a very small struct so we can afford to preheat many, but the exact
// number is uncertain. Any number too large is wasting memory, any number
// too small will cause the pool to have to allocate more memory later.
// This should be set to some reasonable minimum that we expect a terminal
// window to scroll into quickly.
const pageListPagePreheat = 4

// PageListList is PageList.List.
//
// The list of pages in the screen. These are expected to be in order
// where the first page is the topmost page (scrollback) and the last is
// the bottommost page (the current active page).
//
// Zig: DoublyLinkedList(Node), the intrusive list of
// src/datastruct/intrusive_linked_list.zig: `first: ?*Node = null`,
// `last: ?*Node = null`. The methods below are that file's, limited to the
// ones PageList.zig calls; they are not declarations of PageList.zig.
type PageListList struct {
	first *PageListNode
	last  *PageListNode
}

// InsertAfter is DoublyLinkedList.insertAfter: insert a new node after an
// existing one.
func (list *PageListList) InsertAfter(node *PageListNode, new_node *PageListNode) {
	panic(notImplemented)
}

// InsertBefore is DoublyLinkedList.insertBefore: insert a new node before
// an existing one.
func (list *PageListList) InsertBefore(node *PageListNode, new_node *PageListNode) {
	panic(notImplemented)
}

// Append is DoublyLinkedList.append: insert a new node at the end of the
// list.
func (list *PageListList) Append(new_node *PageListNode) { panic(notImplemented) }

// Prepend is DoublyLinkedList.prepend: insert a new node at the beginning
// of the list.
func (list *PageListList) Prepend(new_node *PageListNode) { panic(notImplemented) }

// Remove is DoublyLinkedList.remove: remove a node from the list.
func (list *PageListList) Remove(node *PageListNode) { panic(notImplemented) }

// PopFirst is DoublyLinkedList.popFirst: remove and return the first node
// in the list, or nil if the list is empty.
func (list *PageListList) PopFirst() *PageListNode { panic(notImplemented) }

// PageListNode is PageList.Node: a single node within the PageList linked
// list.
//
// This isn't pub because you can access the type via List.Node.
type PageListNode struct {
	// Zig: ?*Node, default null.
	prev *PageListNode
	// Zig: ?*Node, default null.
	next *PageListNode
	data Page
	// Zig: u64.
	serial uint64
}

// pageListNodePool is PageList.NodePool: the memory pool we get page nodes
// from.
//
// Zig: std.heap.MemoryPool(List.Node). The pool's memory is dropped (see
// the file header); the type is kept only as the type of
// `MemoryPool.nodes`.
type pageListNodePool struct{}

// PageList.std_capacity (line 57), an alias of pagepkg.std_capacity, is not
// declared: use PageStdCapacity (page.go).
//
// The standard page capacity that we use as a starting point for
// all pages. This is chosen as a sane default that fits most terminal
// usage to support using our pool.

// pageListStdSize is PageList.std_size: the byte size required for a
// standard page.
//
// Zig: const std_size = Page.layout(std_capacity).total_size, computed at
// compile time. The implementation returns
// PageLayoutOf(PageStdCapacity).total_size. Note that PageStdCapacity
// has the production `grapheme_bytes` (8192) while Zig's tests run with
// 512 (page.go), so this value is not the one Zig's unit tests see.
//
// It is a function, not a variable: it is computed from PageStdCapacity each
// time, so a test that sets PageStdCapacity (Zig's unit tests run with a
// smaller standard capacity than the recorded corpus) sees a consistent
// value without any start-up ordering.
func pageListStdSize() uint64 { panic(notImplemented) }

// pageListPagePool is PageList.PagePool.
//
// The memory pool we use for page memory buffers. We use a separate pool
// so we can allocate these with a page allocator. We have to use a page
// allocator because we need memory that is zero-initialized and page-aligned.
//
// Zig: std.heap.MemoryPoolAligned([std_size]u8,
// .fromByteUnits(std.heap.page_size_min)). The pool's memory is dropped
// (see the file header); the type is kept only as the type of
// `MemoryPool.pages`.
type pageListPagePool struct{}

// pageListPagePoolItemSize is NOT a declaration of PageList.zig: it is
// `PagePool.item_size`, which std defines as the larger of the item's size
// and the size of the pool's free-list node, here `std_size`. minMaxSize
// and grow compute with it and so do tests
// (`s.page_size + PagePool.item_size > s.maxSize()`). Bytes. Left at zero
// in the wireframe; the implementation sets it with pageListStdSize.
//
// A function for the same reason as pageListStdSize.
func pageListPagePoolItemSize() uint64 { panic(notImplemented) }

// pageListPinSet is PageList.PinSet.
//
// List of pins, known as "tracked" pins. These are pins that are kept
// up to date automatically through page-modifying operations.
//
// Zig: std.AutoArrayHashMapUnmanaged(*Pin, void): a set of pin pointers
// that keeps insertion order, except that swapRemove moves the last key
// into the removed key's place. The methods below are std's, limited to
// the ones PageList.zig calls (see the file header); they are not
// declarations of PageList.zig. The zero value is the empty set (`.{}`).
type pageListPinSet struct {
	// The keys in order; what `keys()` returns.
	keys []*PageListPin
}

// PutNoClobber is std's putNoClobber: add key, asserting that it is not
// already present.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *pageListPinSet) PutNoClobber(key *PageListPin) error { panic(notImplemented) }

// SwapRemove is std's swapRemove: if key is present, remove it by moving
// the last key into its place and return true; otherwise return false.
func (self *pageListPinSet) SwapRemove(key *PageListPin) bool { panic(notImplemented) }

// Keys is std's keys: the keys in order. The slice is the set's own and is
// only valid until the set is modified.
func (self *pageListPinSet) Keys() []*PageListPin { panic(notImplemented) }

// Count is std's count: the number of keys.
func (self *pageListPinSet) Count() uint { panic(notImplemented) }

// pageListPinPool is PageList.PinPool.
//
// Zig: std.heap.MemoryPool(Pin). The pool's memory is dropped (see the file
// header); the type is kept only as the type of `MemoryPool.pins`.
type pageListPinPool struct{}

// PageListMemoryPool is PageList.MemoryPool.
//
// The pool of memory used for a pagelist. This can be shared between
// multiple pagelists but it is not threadsafe.
//
// Zig's first field, `alloc: Allocator`, is dropped. The three pools hold
// no memory in Go; see the file header.
type PageListMemoryPool struct {
	nodes pageListNodePool
	pages pageListPagePool
	pins  pageListPinPool
}

// PageListMemoryPoolResetMode is PageList.MemoryPool.ResetMode.
//
// Zig: std.heap.ArenaAllocator.ResetMode, union(enum) { free_all,
// retain_capacity, retain_with_limit: usize }. Only the field named by Tag
// is meaningful.
type PageListMemoryPoolResetMode struct {
	Tag PageListMemoryPoolResetModeTag

	// Zig: usize, a number of bytes.
	RetainWithLimit uint64
}

// PageListMemoryPoolResetModeTag is the tag of
// PageList.MemoryPool.ResetMode, values numbered from zero in the order
// listed.
type PageListMemoryPoolResetModeTag uint8

const (
	// Releases all allocated memory in the arena.
	PageListMemoryPoolResetModeFreeAll PageListMemoryPoolResetModeTag = iota
	// This will pre-heat the arena for future allocations by allocating a
	// large enough buffer for all previously done allocations.
	PageListMemoryPoolResetModeRetainCapacity
	// This is the same as `retain_capacity`, but the memory will be shrunk
	// to this value if it exceeds the limit.
	PageListMemoryPoolResetModeRetainWithLimit
)

// String returns the Zig tag name (@tagName).
func (self PageListMemoryPoolResetModeTag) String() string { panic(notImplemented) }

// PageListMemoryPoolInit is PageList.MemoryPool.init.
//
// Zig: pub fn init(gen_alloc: Allocator, page_alloc: Allocator,
// preheat: usize) Allocator.Error!MemoryPool. Both allocator parameters are
// dropped. `preheat` is a number of nodes and pages (the pin pool is always
// preheated with 8). The error is ErrOutOfMemory (dcs.go).
func PageListMemoryPoolInit(preheat uint) (PageListMemoryPool, error) { panic(notImplemented) }

// Deinit is PageList.MemoryPool.deinit.
func (self *PageListMemoryPool) Deinit() { panic(notImplemented) }

// Reset is PageList.MemoryPool.reset.
func (self *PageListMemoryPool) Reset(mode PageListMemoryPoolResetMode) { panic(notImplemented) }

// PageList is the file-level struct of PageList.zig.
//
// The Go zero value is not an initialised list; use PageListInit.
type PageList struct {
	// The memory pool we get page nodes, pages from.
	pool PageListMemoryPool

	// The list of pages in the screen.
	pages PageListList

	// A monotonically increasing serial number that is incremented each
	// time a page is allocated or reused as new. The serial is assigned to
	// the Node.
	//
	// The serial number can be used to detect whether the page is identical
	// to the page that was originally referenced by a pointer. Since we reuse
	// and pool memory, pointer stability is not guaranteed, but the serial
	// will always be different for different allocations.
	//
	// Developer note: we never do overflow checking on this. If we created
	// a new page every second it'd take 584 billion years to overflow. We're
	// going to risk it.
	//
	// Zig: u64.
	page_serial uint64

	// The lowest still valid serial number that could exist. This allows
	// for quick comparisons to find invalid pages in references.
	//
	// Zig: u64.
	page_serial_min uint64

	// Byte size of the total amount of allocated pages. Note this does
	// not include the total allocated amount in the pool which may be more
	// than this due to preheating.
	//
	// Zig: usize. Bytes.
	page_size uint64

	// Maximum size of the page allocation in bytes. This only includes pages
	// that are used ONLY for scrollback. If the active area is still partially
	// in a page that also includes scrollback, then that page is not included.
	//
	// Zig: usize. Bytes. "No limit" is the maximum uint64; see the file
	// header.
	explicit_max_size uint64

	// This is the minimum max size that we will respect due to the rows/cols
	// of the PageList. We must always be able to fit at least the active area
	// and at least two pages for our algorithms.
	//
	// Zig: usize. Bytes.
	min_max_size uint64

	// The total number of rows represented by this PageList. This is used
	// specifically for scrollbar information so we can have the total size.
	//
	// Zig: usize. A count.
	total_rows uint

	// The list of tracked pins. These are kept up to date automatically.
	tracked_pins pageListPinSet

	// The top-left of certain parts of the screen that are frequently
	// accessed so we don't have to traverse the linked list to find them.
	//
	// For other tags, don't need this:
	//   - screen: pages.first
	//   - history: active row minus one
	viewport PageListViewport

	// The pin used for when the viewport scrolls. This is always pre-allocated
	// so that scrolling doesn't have a failable memory allocation. This should
	// never be access directly; use `viewport`.
	viewport_pin *PageListPin

	// The row offset from the top that the viewport pin is at. We
	// store the offset from the top because it doesn't change while more
	// data is printed to the terminal.
	//
	// This is null when it isn't calculated. It is calculated on demand
	// when the viewportRowOffset function is called, because it is only
	// required for certain operations such as rendering the scrollbar.
	//
	// In order to make this more efficient, in many places where the value
	// would be invalidated, we update it in-place instead. This is key to
	// keeping our performance decent in normal cases since recalculating
	// this from scratch, depending on the size of the scrollback and position
	// of the pin, can be very expensive.
	//
	// This is only valid if viewport is `pin`. Every other offset is
	// self-evident or quick to calculate.
	//
	// Zig: ?usize. A count of rows.
	viewport_pin_row_offset Optional[uint]

	// The current desired screen dimensions. I say "desired" because individual
	// pages may still be a different size and not yet reflowed since we lazily
	// reflow text.
	cols SizeCellCountInt
	rows SizeCellCountInt

	// If this is true then verifyIntegrity will do nothing. This is
	// only present with runtime safety enabled.
	//
	// Zig: usize (default 0) when build_options.slow_runtime_safety, void
	// otherwise. See the file header.
	pause_integrity_checks uint
}

// PageListViewport is PageList.Viewport: the viewport location.
//
// Zig: union(enum) in which no variant has a payload, so the struct has
// only the Tag. The Go zero value is `.active`.
type PageListViewport struct {
	Tag PageListViewportTag
}

// PageListViewportTag is the tag of PageList.Viewport, values numbered from
// zero in the order listed.
type PageListViewportTag uint8

const (
	// The viewport is pinned to the active area. By using a specific marker
	// for this instead of tracking the row offset, we eliminate a number of
	// memory writes making scrolling faster.
	PageListViewportActive PageListViewportTag = iota

	// The viewport is pinned to the top of the screen, or the farthest
	// back in the scrollback history.
	PageListViewportTop

	// The viewport is pinned to a tracked pin. The tracked pin is ALWAYS
	// s.viewport_pin hence this has no value. We force that value to prevent
	// allocations.
	PageListViewportPin
)

// String returns the Zig tag name (@tagName).
func (self PageListViewportTag) String() string { panic(notImplemented) }

// pageListMinMaxSize is PageList.minMaxSize.
//
// Returns the minimum valid "max size" for a given number of rows and cols
// such that we can fit the active area AND at least two pages. Note we
// need the two pages for algorithms to work properly (such as grow) but
// we don't need to fit double the active area.
//
// This min size may not be totally correct in the case that a large
// number of other dimensions makes our row size in a page very small.
// But this gives us a nice fast heuristic for determining min/max size.
// Therefore, if the page size is violated you should always also verify
// that we have enough space for the active area.
//
// Zig: returns usize, `PagePool.item_size * pages`: bytes, uint64.
func pageListMinMaxSize(cols SizeCellCountInt, rows SizeCellCountInt) uint64 {
	panic(notImplemented)
}

// pageListInitialCapacity is PageList.initialCapacity.
//
// Calculates the initial capacity for a new page for a given column
// count. This will attempt to fit within std_size at all times so we
// can use our memory pool, but if cols is too big, this will return a
// larger capacity.
//
// The returned capacity is always guaranteed to layout properly (not
// overflow). We are able to support capacities up to the maximum int
// value of cols, so this will never overflow.
func pageListInitialCapacity(cols SizeCellCountInt) PageCapacity { panic(notImplemented) }

// PageList.pageAllocator (line 322) is omitted; see the file header.

// PageList.init_tw (line 335) is omitted; see the file header.

// PageListInit is PageList.init.
//
// Initialize the page. The top of the first page in the list is always the
// top of the active area of the screen (important knowledge for quickly
// setting up cursors in Screen).
//
// max_size is the maximum number of bytes that will be allocated for
// pages. If this is smaller than the bytes required to show the viewport
// then max_size will be ignored and the viewport will be shown, but no
// scrollback will be created. max_size is always rounded down to the nearest
// terminal page size (not virtual memory page), otherwise we would always
// slightly exceed max_size in the limits.
//
// If max_size is null then there is no defined limit and the screen will
// grow forever. In reality, the limit is set to the byte limit that your
// computer can address in memory. If you somehow require more than that
// (due to disk paging) then please contribute that yourself and perhaps
// search deep within yourself to find out why you need that.
//
// Zig: pub fn init(alloc: Allocator, cols: size.CellCountInt,
// rows: size.CellCountInt, max_size: ?usize) Allocator.Error!PageList. The
// allocator parameter is dropped; max_size is bytes. The error is
// ErrOutOfMemory (dcs.go).
func PageListInit(
	cols SizeCellCountInt,
	rows SizeCellCountInt,
	max_size Optional[uint64],
) (PageList, error) {
	panic(notImplemented)
}

// PageList.initPages_tw (line 420) is omitted; see the file header.

// pageListInitPages is PageList.initPages.
//
// Zig: fn initPages(pool: *MemoryPool, serial: *u64,
// cols: size.CellCountInt, rows: size.CellCountInt)
// Allocator.Error!struct { List, usize }. The second element of the result
// is the byte size of the pages created (uint64). The error is
// ErrOutOfMemory (dcs.go).
func pageListInitPages(
	pool *PageListMemoryPool,
	serial *uint64,
	cols SizeCellCountInt,
	rows SizeCellCountInt,
) (PageListList, uint64, error) {
	panic(notImplemented)
}

// AssertIntegrity is PageList.assertIntegrity.
//
// Assert that the PageList is in a valid state. This is a no-op in
// release builds.
//
// Zig: self is *const PageList.
func (self *PageList) AssertIntegrity() { panic(notImplemented) }

// PauseIntegrityChecks is PageList.pauseIntegrityChecks.
//
// Pause or resume integrity checks. This is useful when you're doing
// a multi-step operation that temporarily leaves the PageList in an
// inconsistent state.
func (self *PageList) PauseIntegrityChecks(pause bool) { panic(notImplemented) }

// PageList.IntegrityError. An error set is not a type in Go; its members
// are the sentinels below.
var (
	ErrPageSerialInvalid           = errors.New("PageSerialInvalid")
	ErrTotalRowsMismatch           = errors.New("TotalRowsMismatch")
	ErrTrackedPinInvalid           = errors.New("TrackedPinInvalid")
	ErrViewportPinOffsetMismatch   = errors.New("ViewportPinOffsetMismatch")
	ErrViewportPinInsufficientRows = errors.New("ViewportPinInsufficientRows")
)

// verifyIntegrity is PageList.verifyIntegrity.
//
// Verify the integrity of the PageList. This is expensive and should
// only be called in debug/test builds.
//
// Zig: fn verifyIntegrity(self: *const PageList) IntegrityError!void. The
// error is one of the PageList.IntegrityError sentinels.
func (self *PageList) verifyIntegrity() error { panic(notImplemented) }

// Deinit is PageList.deinit.
//
// Deinit the pagelist. If you own the memory pool (used clonePool) then
// this will reset the pool and retain capacity.
//
// Kept although it mostly frees: it asserts integrity first, and every
// test calls it.
func (self *PageList) Deinit() { panic(notImplemented) }

// Reset is PageList.reset.
//
// Reset the PageList back to an empty state. This is similar to
// deinit and reinit but it importantly preserves the pointer
// stability of tracked pins (they're moved to the top-left since
// all contents are cleared).
//
// This can't fail because we always retain at least enough allocated
// memory to fit the active area.
func (self *PageList) Reset() { panic(notImplemented) }

// PageListClone is PageList.Clone: the options of clone.
type PageListClone struct {
	// The top and bottom (inclusive) points of the region to clone.
	// The x coordinate is ignored; the full row is always cloned.
	top Point
	// Zig: ?point.Point, default null.
	bot Optional[Point]

	// If this is non-null then cloning will attempt to remap the tracked
	// pins into the new cloned area and will keep track of the old to
	// new mapping in this map. If this is null, the cloned pagelist will
	// not retain any previously tracked pins except those required for
	// internal operations.
	//
	// Any pins not present in the map were not remapped.
	//
	// Zig: ?*TrackedPinsRemap, default null.
	tracked_pins *PageListCloneTrackedPinsRemap
}

// PageListCloneTrackedPinsRemap is PageList.Clone.TrackedPinsRemap.
//
// Zig: std.AutoHashMap(*Pin, *Pin). A Go map; the std methods (`init`,
// `deinit`, `get`, `putNoClobber`) are map operations. A caller makes one
// with `PageListCloneTrackedPinsRemap{}` and passes its address.
type PageListCloneTrackedPinsRemap map[*PageListPin]*PageListPin

// Clone is PageList.clone.
//
// Clone this pagelist from the top to bottom (inclusive).
//
// The viewport is always moved to the active area.
//
// The cloned pagelist must contain at least enough rows for the active
// area. If the region specified has less rows than the active area then
// rows will be added to the bottom of the region to make up the difference.
//
// Zig: pub fn clone(self: *const PageList, alloc: Allocator, opts: Clone)
// !PageList. The allocator parameter is dropped. The inferred error set is
// ErrOutOfMemory (dcs.go) and Page.CloneFromError (page.go).
func (self *PageList) Clone(opts PageListClone) (PageList, error) { panic(notImplemented) }

// PageListResize is PageList.Resize: resize options.
type PageListResize struct {
	// The new cols/cells of the screen.
	//
	// Zig: ?size.CellCountInt, default null.
	cols Optional[SizeCellCountInt]
	// Zig: ?size.CellCountInt, default null.
	rows Optional[SizeCellCountInt]

	// Whether to reflow the text. If this is false then the text will
	// be truncated if the new size is smaller than the old size.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default must set it.
	reflow bool

	// Set this to the current cursor position in the active area. Some
	// resize/reflow behavior depends on the cursor position.
	//
	// Zig: ?Cursor, default null.
	cursor Optional[PageListResizeCursor]
}

// PageListResizeCursor is PageList.Resize.Cursor.
type PageListResizeCursor struct {
	x SizeCellCountInt
	y SizeCellCountInt
}

// Resize is PageList.resize.
//
// Resize
// TODO: docs
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
//
// docs/mapping.md ("Requirements the wireframe tests impose on the
// implementation") places one deferral of the integrity check inside this
// function.
func (self *PageList) Resize(opts PageListResize) error { panic(notImplemented) }

// resizeCols is PageList.resizeCols: resize the pagelist with reflow by
// adding or removing columns.
//
// Zig: fn resizeCols(self: *PageList, cols: size.CellCountInt,
// cursor: ?Resize.Cursor) Allocator.Error!void.
func (self *PageList) resizeCols(
	cols SizeCellCountInt,
	cursor Optional[PageListResizeCursor],
) error {
	panic(notImplemented)
}

// pageListReflowCursor is PageList.ReflowCursor.
//
// We use a cursor to track where we are in the src/dst. This is very
// similar to Screen.Cursor, so see that for docs on individual fields.
// We don't use a Screen because we don't need all the same data and we
// do our best to optimize having direct access to the page memory.
//
// `page`, `page_row` and `page_cell` are Go pointers into `node.data` and
// its slices; see "Pointers into pages" in the file header.
type pageListReflowCursor struct {
	x            SizeCellCountInt
	y            SizeCellCountInt
	pending_wrap bool
	node         *PageListNode
	page         *Page
	page_row     *PageRow
	page_cell    *PageCell
	// Zig: usize. A count of rows.
	new_rows uint

	// This is the final row count of the reflowed pages.
	//
	// Zig: usize. A count of rows.
	total_rows uint
}

// pageListReflowCursorInit is PageList.ReflowCursor.init.
func pageListReflowCursorInit(node *PageListNode) pageListReflowCursor { panic(notImplemented) }

// reflowRow is PageList.ReflowCursor.reflowRow: reflow the provided row in
// to this cursor.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *pageListReflowCursor) reflowRow(list *PageList, row PageListPin) error {
	panic(notImplemented)
}

// pageListReflowCursorWriteCellResult is the anonymous enum
// PageList.ReflowCursor.writeCell returns, values numbered from zero in the
// order listed. The name is not Zig's.
type pageListReflowCursorWriteCellResult uint8

const (
	// Wrote the cell, move to the next.
	pageListReflowCursorWriteCellResultSuccess pageListReflowCursorWriteCellResult = iota
	// Didn't write the cell, repeat writing this same cell.
	pageListReflowCursorWriteCellResultRepeat
	// Wrote the cell but request to skip the next so skip it.
	// This is used for things like spacers.
	pageListReflowCursorWriteCellResultSkipNext
)

// String returns the Zig tag name (@tagName).
func (self pageListReflowCursorWriteCellResult) String() string { panic(notImplemented) }

// writeCell is PageList.ReflowCursor.writeCell.
//
// Write a cell. On error, this will not unwrite the cell but
// the cell may be incomplete (but valid). For example, if the source
// cell is styled and we failed to allocate space for styles, the
// written cell may not be styled but it is valid.
//
// The key failure to recognize for callers is when we can't increase
// capacity in our destination page. In this case, the caller may want
// to split the page at this row, rewrite the row into a new page
// and continue from there.
//
// But this function guarantees the terminal/page will be in a
// coherent state even on error.
//
// Zig: fn writeCell(self: *ReflowCursor, list: *PageList,
// cell: *const pagepkg.Cell, src_page: *const Page)
// IncreaseCapacityError!enum { success, repeat, skip_next }. The error is
// ErrOutOfMemory (dcs.go) or ErrOutOfSpace.
//
// The block under `build_options.kitty_graphics` (line 1479) is compiled
// out and is not ported.
func (self *pageListReflowCursor) writeCell(
	list *PageList,
	cell *PageCell,
	src_page *Page,
) (pageListReflowCursorWriteCellResult, error) {
	panic(notImplemented)
}

// moveLastRowToNewPage is PageList.ReflowCursor.moveLastRowToNewPage.
//
// Create a new page in the provided list with the provided
// capacity then clone the row currently being worked on to
// it and delete it from the old page. Places cursor in the
// same position it was in in the old row in the new one.
//
// Asserts that the cursor is on the final row of the page.
//
// Expects that the provided capacity is sufficient to copy
// the row.
//
// If this is the only row in the page, the page is removed
// from the list after cloning the row.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *pageListReflowCursor) moveLastRowToNewPage(list *PageList, cap PageCapacity) error {
	panic(notImplemented)
}

// increaseCapacity is PageList.ReflowCursor.increaseCapacity: increase the
// capacity of the current page.
//
// Zig: fn increaseCapacity(self: *ReflowCursor, list: *PageList,
// adjustment: ?IncreaseCapacity) IncreaseCapacityError!void. The error is
// ErrOutOfMemory (dcs.go) or ErrOutOfSpace.
func (self *pageListReflowCursor) increaseCapacity(
	list *PageList,
	adjustment Optional[PageListIncreaseCapacity],
) error {
	panic(notImplemented)
}

// bottom is PageList.ReflowCursor.bottom.
//
// True if this cursor is at the bottom of the page by capacity,
// i.e. we can't scroll anymore.
//
// Zig: self is *const ReflowCursor.
func (self *pageListReflowCursor) bottom() bool { panic(notImplemented) }

// cursorForward is PageList.ReflowCursor.cursorForward.
func (self *pageListReflowCursor) cursorForward() { panic(notImplemented) }

// cursorScroll is PageList.ReflowCursor.cursorScroll.
//
// Create a new row and move the cursor down.
//
// Asserts that the cursor is on the bottom row of the
// page and that there is capacity to add a new one.
func (self *pageListReflowCursor) cursorScroll() { panic(notImplemented) }

// cursorNewPage is PageList.ReflowCursor.cursorNewPage.
//
// Create a new page in the provided list with the provided
// capacity and one row and move the cursor in to it at 0,0
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *pageListReflowCursor) cursorNewPage(list *PageList, cap PageCapacity) error {
	panic(notImplemented)
}

// cursorScrollOrNewPage is PageList.ReflowCursor.cursorScrollOrNewPage.
//
// Performs `cursorScroll` or `cursorNewPage` as necessary
// depending on if the cursor is currently at the bottom.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *pageListReflowCursor) cursorScrollOrNewPage(list *PageList, cap PageCapacity) error {
	panic(notImplemented)
}

// cursorAbsolute is PageList.ReflowCursor.cursorAbsolute.
func (self *pageListReflowCursor) cursorAbsolute(x SizeCellCountInt, y SizeCellCountInt) {
	panic(notImplemented)
}

// countTrailingEmptyCells is PageList.ReflowCursor.countTrailingEmptyCells.
//
// Zig: self is *const ReflowCursor; returns usize, a count of cells.
func (self *pageListReflowCursor) countTrailingEmptyCells() uint { panic(notImplemented) }

// copyRowMetadata is PageList.ReflowCursor.copyRowMetadata.
//
// Zig: other is *const Row.
func (self *pageListReflowCursor) copyRowMetadata(other *PageRow) { panic(notImplemented) }

// resizeWithoutReflow is PageList.resizeWithoutReflow.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *PageList) resizeWithoutReflow(opts PageListResize) error { panic(notImplemented) }

// resizeWithoutReflowGrowCols is PageList.resizeWithoutReflowGrowCols.
//
// Zig: fn resizeWithoutReflowGrowCols(self: *PageList,
// cols: size.CellCountInt, chunk: PageIterator.Chunk) Allocator.Error!void.
func (self *PageList) resizeWithoutReflowGrowCols(
	cols SizeCellCountInt,
	chunk PageListPageIteratorChunk,
) error {
	panic(notImplemented)
}

// trailingBlankLines is PageList.trailingBlankLines.
//
// Returns the number of trailing blank lines, not to exceed max. Max
// is used to limit our traversal in the case of large scrollback.
//
// Zig: self is *const PageList.
func (self *PageList) trailingBlankLines(max SizeCellCountInt) SizeCellCountInt {
	panic(notImplemented)
}

// trimTrailingBlankRows is PageList.trimTrailingBlankRows.
//
// Trims up to max trailing blank rows from the pagelist and returns the
// number of rows trimmed. A blank row is any row with no text (but may
// have styling).
//
// IMPORTANT: This function does NOT update `total_rows`. It returns the
// number of rows trimmed, and the caller is responsible for decrementing
// `total_rows` by this amount.
func (self *PageList) trimTrailingBlankRows(max SizeCellCountInt) SizeCellCountInt {
	panic(notImplemented)
}

// PageListScroll is PageList.Scroll: scroll options.
//
// Zig: union(enum). Only the field named by Tag is meaningful; `active`
// and `top` have no payload and no field. The Go zero value is `.active`.
type PageListScroll struct {
	Tag PageListScrollTag

	// Scroll to the given absolute row from the top. A value of zero
	// is the top row. This row will be the first visible row in the viewport.
	// Scrolling into or below the active area will clamp to the active area.
	//
	// Zig: usize. A count of rows.
	Row uint

	// Scroll up (negative) or down (positive) by the given number of
	// rows. This is clamped to the "top" and "active" top left.
	//
	// Zig: isize.
	DeltaRow int

	// Jump forwards (positive) or backwards (negative) a set number of
	// prompts. If the absolute value is greater than the number of prompts
	// in either direction, jump to the furthest prompt in that direction.
	//
	// Zig: isize.
	DeltaPrompt int

	// Scroll directly to a specific pin in the page. This will be set
	// as the top left of the viewport (ignoring the pin x value).
	Pin PageListPin
}

// PageListScrollTag is the tag of PageList.Scroll, values numbered from
// zero in the order listed.
type PageListScrollTag uint8

const (
	// Scroll to the active area. This is also sometimes referred to as
	// the "bottom" of the screen. This makes it so that the end of the
	// screen is fully visible since the active area is the bottom
	// rows/cols of the screen.
	PageListScrollActive PageListScrollTag = iota

	// Scroll to the top of the screen, which is the farthest back in
	// the scrollback history.
	PageListScrollTop

	PageListScrollRow
	PageListScrollDeltaRow
	PageListScrollDeltaPrompt
	PageListScrollPin
)

// String returns the Zig tag name (@tagName).
func (self PageListScrollTag) String() string { panic(notImplemented) }

// Scroll is PageList.scroll.
//
// Scroll the viewport. This will never create new scrollback, allocate
// pages, etc. This can only be used to move the viewport within the
// previously allocated pages.
func (self *PageList) Scroll(behavior PageListScroll) { panic(notImplemented) }

// scrollPrompt is PageList.scrollPrompt.
//
// Jump the viewport forwards (positive) or backwards (negative) a set number of
// prompts (delta).
//
// Zig: delta is isize.
func (self *PageList) scrollPrompt(delta int) { panic(notImplemented) }

// ScrollClear is PageList.scrollClear.
//
// Clear the screen by scrolling written contents up into the scrollback.
// This will not update the viewport.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *PageList) ScrollClear() error { panic(notImplemented) }

// Compact is PageList.compact.
//
// Compact a page to use the minimum required memory for the contents
// it stores. Returns the new node pointer if compaction occurred, or null
// if the page was already compact or compaction would not provide meaningful
// savings.
//
// The current design of PageList at the time of writing this doesn't
// allow for smaller than `std_size` nodes so if the current node's backing
// page is standard size or smaller, no compaction will occur. In the
// future we should fix this up.
//
// If this returns OOM, the PageList is left unchanged and no dangling
// memory references exist. It is safe to ignore the error and continue using
// the uncompacted page.
//
// Zig: Allocator.Error!?*List.Node. A nil node with a nil error is `null`.
func (self *PageList) Compact(node *PageListNode) (*PageListNode, error) {
	panic(notImplemented)
}

// PageList.SplitError is OutOfMemory (allocator OOM; ErrOutOfMemory,
// dcs.go) and OutOfSpace (page can't be split further because it is
// already a single row).
//
// ErrOutOfSpace is also a member of PageList.IncreaseCapacityError. It is
// declared here, at its first use in the file.
var ErrOutOfSpace = errors.New("OutOfSpace")

// Split is PageList.split.
//
// Split the given node in the PageList at the given pin.
//
// The row at the pin and after will be moved into a new page with
// the same capacity as the original page. Alternatively, you can "split
// above" by splitting the row following the desired split row.
//
// Since the split happens below the pin, the pin remains valid.
//
// Zig: SplitError!void.
func (self *PageList) Split(p PageListPin) error { panic(notImplemented) }

// PageListScrollbar is PageList.Scrollbar.
//
// This represents the state necessary to render a scrollbar for this
// PageList. It has the total size, the offset, and the size of the viewport.
//
// All three fields are `usize` counts of rows.
type PageListScrollbar struct {
	// Total size of the scrollable area.
	total uint

	// The offset into the total area that the viewport is at. This is
	// guaranteed to be less than or equal to total. This includes the
	// visible row.
	offset uint

	// The length of the visible area. This is including the offset row.
	len uint
}

// PageListScrollbarZero is PageList.Scrollbar.zero: a zero-sized scrollable
// region.
var PageListScrollbarZero = PageListScrollbar{total: 0, offset: 0, len: 0}

// PageList.Scrollbar.C and PageList.Scrollbar.cval (lines 2943 and 2949)
// are C ABI and are not ported.

// Eql is PageList.Scrollbar.eql: comparison for scrollbars.
func (self PageListScrollbar) Eql(other PageListScrollbar) bool { panic(notImplemented) }

// Scrollbar is PageList.scrollbar.
//
// Return the scrollbar state for this PageList.
//
// This may be expensive to calculate depending on where the viewport
// is (arbitrary pins are expensive). The caller should take care to only
// call this as needed and not too frequently.
func (self *PageList) Scrollbar() PageListScrollbar { panic(notImplemented) }

// viewportRowOffset is PageList.viewportRowOffset.
//
// Returns the offset of the current viewport from the top of the
// screen.
//
// This is potentially expensive to calculate because if the viewport
// is a pin and the pin is near the beginning of the scrollback, we
// will traverse a lot of linked list nodes.
//
// The result is cached so repeated calls are cheap.
//
// Zig: returns usize, a count of rows.
func (self *PageList) viewportRowOffset() uint { panic(notImplemented) }

// fixupViewport is PageList.fixupViewport.
//
// This fixes up the viewport data when rows are removed from the
// PageList. This will update a viewport to `active` if row removal
// puts the viewport into the active area, to `top` if the viewport
// is now at row 0, and updates any row offset caches as necessary.
//
// This is unit tested transitively through other tests such as
// eraseRows.
//
// Zig: removed is usize, a count of rows.
func (self *PageList) fixupViewport(removed uint) { panic(notImplemented) }

// MaxSize is PageList.maxSize.
//
// Returns the actual max size. This may be greater than the explicit
// value if the explicit value is less than the min_max_size.
//
// This value is a HEURISTIC. You cannot assert on this value. We may
// exceed this value if required to fit the active area. This may be
// required in some cases if the active area has a large number of
// graphemes, styles, etc.
//
// Zig: self is *const PageList; returns usize. Bytes, uint64.
func (self *PageList) MaxSize() uint64 { panic(notImplemented) }

// Grow is PageList.grow.
//
// Grow the active area by exactly one row.
//
// This may allocate, but also may not if our current page has more
// capacity we can use. This will prune scrollback if necessary to
// adhere to max_size.
//
// This returns the newly allocated page node if there is one.
//
// Zig: Allocator.Error!?*List.Node. A nil node with a nil error is `null`.
// The error is ErrOutOfMemory (dcs.go).
func (self *PageList) Grow() (*PageListNode, error) { panic(notImplemented) }

// PageListIncreaseCapacity is PageList.IncreaseCapacity: possible
// dimensions to increase capacity for.
//
// Zig: enum, values numbered from zero in the order listed. Each tag name
// is the name of a field of page.Capacity.
type PageListIncreaseCapacity uint8

const (
	PageListIncreaseCapacityStyles PageListIncreaseCapacity = iota
	PageListIncreaseCapacityGraphemeBytes
	PageListIncreaseCapacityHyperlinkBytes
	PageListIncreaseCapacityStringBytes
)

// String returns the Zig tag name (@tagName).
func (self PageListIncreaseCapacity) String() string { panic(notImplemented) }

// PageList.IncreaseCapacityError is:
//   - OutOfMemory (ErrOutOfMemory, dcs.go): an actual system OOM trying to
//     allocate memory;
//   - OutOfSpace (ErrOutOfSpace, above): the existing page is already at
//     max capacity for the given adjustment. The caller must create a new
//     page, remove data from the old page, etc. (up to the caller).

// IncreaseCapacity is PageList.increaseCapacity.
//
// Increase the capacity of the given page node in the given direction.
// This will always allocate a new node and remove the old node, so the
// existing node pointer will be invalid after this call. The newly created
// node on success is returned.
//
// The increase amount is at the control of the PageList implementation,
// but is guaranteed to always increase by at least one unit in the
// given dimension. Practically, we'll always increase by much more
// (we currently double every time) but callers shouldn't depend on that.
// The only guarantee is some amount of growth.
//
// Adjustment can be null if you want to recreate, reclone the page
// with the same capacity. This is a special case used for rehashing since
// the logic is otherwise the same. In this case, OutOfMemory is the
// only possible error.
//
// Zig: pub fn increaseCapacity(self: *PageList, node: *List.Node,
// adjustment: ?IncreaseCapacity) IncreaseCapacityError!*List.Node.
func (self *PageList) IncreaseCapacity(
	node *PageListNode,
	adjustment Optional[PageListIncreaseCapacity],
) (*PageListNode, error) {
	panic(notImplemented)
}

// createPage is PageList.createPage.
//
// Create a new page node. This does not add it to the list and this
// does not do any memory size accounting with max_size/page_size.
//
// Zig: Allocator.Error!*List.Node. The error is ErrOutOfMemory (dcs.go).
// (It does add the page's bytes to `page_size`, through createPageExt.)
func (self *PageList) createPage(cap PageCapacity) (*PageListNode, error) {
	panic(notImplemented)
}

// pageListCreatePageExt is PageList.createPageExt.
//
// Zig: inline fn createPageExt(pool: *MemoryPool, cap: Capacity,
// serial: *u64, total_size: ?*usize) Allocator.Error!*List.Node.
// `total_size`, if not nil, is increased by the byte size of the new page's
// buffer (pageListMemoryLen); `serial` is assigned to the node and then
// incremented.
func pageListCreatePageExt(
	pool *PageListMemoryPool,
	cap PageCapacity,
	serial *uint64,
	total_size *uint64,
) (*PageListNode, error) {
	panic(notImplemented)
}

// destroyNode is PageList.destroyNode.
//
// Destroy the memory of the given node in the PageList linked list
// and return it to the pool. The node is assumed to already be removed
// from the linked list.
//
// IMPORTANT: This function does NOT update `total_rows`. The caller is
// responsible for accounting for the removed rows. This function only
// updates `page_size` (byte accounting), not row accounting.
func (self *PageList) destroyNode(node *PageListNode) { panic(notImplemented) }

// pageListDestroyNodeExt is PageList.destroyNodeExt.
//
// Zig: fn destroyNodeExt(pool: *MemoryPool, node: *List.Node,
// total_size: ?*usize) void. `total_size`, if not nil, is decreased by the
// byte size of the node's page buffer (pageListMemoryLen).
func pageListDestroyNodeExt(
	pool *PageListMemoryPool,
	node *PageListNode,
	total_size *uint64,
) {
	panic(notImplemented)
}

// pageListMemoryLen is NOT a Zig declaration. It stands for
// `page.memory.len`, the byte length of the buffer backing a page that
// belongs to a PageList: pageListStdSize if
// `PageLayoutOf(page.capacity).total_size <= pageListStdSize` (the page
// came from the page pool, whose buffers are all `std_size` long), and that
// `total_size` otherwise. PageList.zig reads it in initPages, deinit,
// reset, clone, compact, grow, createPageExt and destroyNodeExt, and tests
// read `node.data.memory.len`. See the file header.
func pageListMemoryLen(page *Page) uint64 { panic(notImplemented) }

// EraseRow is PageList.eraseRow.
//
// Fast-path function to erase exactly 1 row. Erasing means that the row
// is completely REMOVED, not just cleared. All rows following the removed
// row will be shifted up by 1 to fill the empty space.
//
// Unlike eraseRows, eraseRow does not change the size of any pages. The
// caller is responsible for adjusting the row count of the final page if
// that behavior is required.
//
// Zig: !void. The inferred error set is Page.CloneFromError (page.go).
func (self *PageList) EraseRow(pt Point) error { panic(notImplemented) }

// EraseRowBounded is PageList.eraseRowBounded.
//
// A variant of eraseRow that shifts only a bounded number of following
// rows up, filling the space they leave behind with blank rows.
//
// `limit` is exclusive of the erased row. A limit of 1 will erase the target
// row and shift the row below in to its position, leaving a blank row below.
//
// Zig: !void; limit is usize, a count of rows. The inferred error set is
// Page.CloneFromError (page.go).
func (self *PageList) EraseRowBounded(pt Point, limit uint) error { panic(notImplemented) }

// EraseRows is PageList.eraseRows.
//
// Erase the rows from the given top to bottom (inclusive). Erasing
// the rows doesn't clear them but actually physically REMOVES the rows.
// If the top or bottom point is in the middle of a page, the other
// contents in the page will be preserved but the page itself will be
// underutilized (size < capacity).
//
// Zig: bl_pt is ?point.Point.
func (self *PageList) EraseRows(tl_pt Point, bl_pt Optional[Point]) { panic(notImplemented) }

// erasePage is PageList.erasePage.
//
// Erase a single page, freeing all its resources. The page can be
// anywhere in the linked list but must NOT be the final page in the
// entire list (i.e. must not make the list empty).
//
// IMPORTANT: This function does NOT update `total_rows`. The caller is
// responsible for accounting for the removed rows before or after calling
// this function.
func (self *PageList) erasePage(node *PageListNode) { panic(notImplemented) }

// Pin is PageList.pin.
//
// Returns the pin for the given point. The pin is NOT tracked so it
// is only valid as long as the pagelist isn't modified.
//
// This will return null if the point is out of bounds. The caller
// should clamp the point to the bounds of the coordinate space if
// necessary.
//
// Zig: self is *const PageList; returns ?Pin.
func (self *PageList) Pin(pt Point) Optional[PageListPin] { panic(notImplemented) }

// TrackPin is PageList.trackPin.
//
// Convert the given pin to a tracked pin. A tracked pin will always be
// automatically updated as the pagelist is modified. If the point the
// pin points to is removed completely, the tracked pin will be updated
// to the top-left of the screen.
//
// Zig: Allocator.Error!*Pin. The error is ErrOutOfMemory (dcs.go).
func (self *PageList) TrackPin(p PageListPin) (*PageListPin, error) { panic(notImplemented) }

// UntrackPin is PageList.untrackPin: untrack a previously tracked pin.
// This will deallocate the pin.
func (self *PageList) UntrackPin(p *PageListPin) { panic(notImplemented) }

// CountTrackedPins is PageList.countTrackedPins.
//
// Zig: self is *const PageList; returns usize, a count.
func (self *PageList) CountTrackedPins() uint { panic(notImplemented) }

// TrackedPins is PageList.trackedPins.
//
// Returns the tracked pins for this pagelist. The slice is owned by the
// pagelist and is only valid until the pagelist is modified.
//
// Zig: self is *const PageList; returns []const *Pin.
func (self *PageList) TrackedPins() []*PageListPin { panic(notImplemented) }

// PinIsValid is PageList.pinIsValid.
//
// Checks if a pin is valid for this pagelist. This is a very slow and
// expensive operation since we traverse the entire linked list in the
// worst case. Only for runtime safety/debug.
//
// Zig: self is *const PageList.
func (self *PageList) PinIsValid(p PageListPin) bool { panic(notImplemented) }

// pinIsActive is PageList.pinIsActive.
//
// Returns the viewport for the given pin, preferring to pin to
// "active" if the pin is within the active area.
//
// Zig: self is *const PageList.
func (self *PageList) pinIsActive(p PageListPin) bool { panic(notImplemented) }

// pinIsTop is PageList.pinIsTop: returns true if the pin is at the top of
// the scrollback area.
//
// Zig: self is *const PageList.
func (self *PageList) pinIsTop(p PageListPin) bool { panic(notImplemented) }

// PointFromPin is PageList.pointFromPin.
//
// Convert a pin to a point in the given context. If the pin can't fit
// within the given tag (i.e. its in the history but you requested active),
// then this will return null.
//
// Note that this can be a very expensive operation depending on the tag and
// the location of the pin. This works by traversing the linked list of pages
// in the tagged region.
//
// Therefore, this is recommended only very rarely.
//
// Zig: self is *const PageList; returns ?point.Point.
func (self *PageList) PointFromPin(tag PointTag, p PageListPin) Optional[Point] {
	panic(notImplemented)
}

// GetCell is PageList.getCell.
//
// Get the cell at the given point, or null if the cell does not
// exist or is out of bounds.
//
// Warning: this is slow and should not be used in performance critical paths
//
// Zig: self is *const PageList; returns ?Cell.
func (self *PageList) GetCell(pt Point) Optional[PageListCell] { panic(notImplemented) }

// PageList.diagram (line 4049) is omitted; see the file header.

// PageList.highlightSemanticContent (line 4252) is NOT YET DECLARED: it
// returns `?highlight.Untracked` and highlight.zig has no Go file. When
// highlight.go exists, the stub to add here is
//
//	func (self *PageList) HighlightSemanticContent(
//		at PageListPin,
//		content PageCellSemanticContent,
//	) Optional[HighlightUntracked] { panic(notImplemented) }
//
// Returns the boundaries of the given semantic content type for
// the prompt at the given pin. The pin row MUST be the first row
// of a prompt, otherwise the results may be nonsense.
//
// To get prompt pins, use promptIterator. Warning that if there are
// no semantic prompts ever present, promptIterator will iterate the
// entire PageList. Downstream callers should keep track of a flag if
// they've ever seen semantic prompt operations to prevent this performance
// case.
//
// Note that some semantic content type such as "input" is usually
// nested within prompt boundaries, so the returned boundaries may include
// prompt text.

// PageListDirection is PageList.Direction: direction that iterators can
// move.
//
// Zig: enum { left_up, right_down }, values numbered from zero in the
// order listed.
type PageListDirection uint8

const (
	PageListDirectionLeftUp PageListDirection = iota
	PageListDirectionRightDown
)

// String returns the Zig tag name (@tagName).
func (self PageListDirection) String() string { panic(notImplemented) }

// PageListPromptIterator is PageList.PromptIterator.
type PageListPromptIterator struct {
	// The pin that we are currently at. Also the starting pin when
	// initializing.
	//
	// Zig: ?Pin.
	current Optional[PageListPin]

	// The pin to end at or null if we end when we can't traverse
	// anymore.
	//
	// Zig: ?Pin.
	limit Optional[PageListPin]

	// The direction to do the traversal.
	direction PageListDirection
}

// PageListPromptIteratorEmpty is PageList.PromptIterator.empty.
var PageListPromptIteratorEmpty = PageListPromptIterator{
	direction: PageListDirectionLeftUp,
}

// Next is PageList.PromptIterator.next.
//
// Return the next pin that represents the first row in a prompt.
// From here, you can find the prompt input, command output, etc.
//
// Zig: returns ?Pin.
func (self *PageListPromptIterator) Next() Optional[PageListPin] { panic(notImplemented) }

// NextRightDown is PageList.PromptIterator.nextRightDown.
//
// Zig: returns ?Pin.
func (self *PageListPromptIterator) NextRightDown() Optional[PageListPin] {
	panic(notImplemented)
}

// NextLeftUp is PageList.PromptIterator.nextLeftUp.
//
// Zig: returns ?Pin.
func (self *PageListPromptIterator) NextLeftUp() Optional[PageListPin] {
	panic(notImplemented)
}

// PromptIterator is PageList.promptIterator.
//
// Zig: pub fn promptIterator(self: *const PageList, direction: Direction,
// tl_pt: point.Point, bl_pt: ?point.Point) PromptIterator.
func (self *PageList) PromptIterator(
	direction PageListDirection,
	tl_pt Point,
	bl_pt Optional[Point],
) PageListPromptIterator {
	panic(notImplemented)
}

// PageListCellIterator is PageList.CellIterator.
type PageListCellIterator struct {
	row_it PageListRowIterator
	// Zig: ?Pin, default null.
	cell Optional[PageListPin]
}

// Next is PageList.CellIterator.next.
//
// Zig: returns ?Pin.
func (self *PageListCellIterator) Next() Optional[PageListPin] { panic(notImplemented) }

// CellIterator is PageList.cellIterator.
//
// Zig: pub fn cellIterator(self: *const PageList, direction: Direction,
// tl_pt: point.Point, bl_pt: ?point.Point) CellIterator.
func (self *PageList) CellIterator(
	direction PageListDirection,
	tl_pt Point,
	bl_pt Optional[Point],
) PageListCellIterator {
	panic(notImplemented)
}

// PageListRowIterator is PageList.RowIterator.
type PageListRowIterator struct {
	page_it PageListPageIterator
	// Zig: ?PageIterator.Chunk, default null.
	chunk Optional[PageListPageIteratorChunk]
	// Zig default: 0.
	offset SizeCellCountInt
}

// Next is PageList.RowIterator.next.
//
// Zig: returns ?Pin.
func (self *PageListRowIterator) Next() Optional[PageListPin] { panic(notImplemented) }

// RowIterator is PageList.rowIterator.
//
// Create an iterator that can be used to iterate all the rows in
// a region of the screen from the given top-left. The tag of the
// top-left point will also determine the end of the iteration,
// so convert from one reference point to another to change the
// iteration bounds.
//
// Zig: pub fn rowIterator(self: *const PageList, direction: Direction,
// tl_pt: point.Point, bl_pt: ?point.Point) RowIterator.
func (self *PageList) RowIterator(
	direction PageListDirection,
	tl_pt Point,
	bl_pt Optional[Point],
) PageListRowIterator {
	panic(notImplemented)
}

// PageListPageIterator is PageList.PageIterator.
type PageListPageIterator struct {
	// Zig: ?Pin, default null.
	row Optional[PageListPin]
	// Zig default: .none, the Go zero value.
	limit pageListPageIteratorLimit
	// Zig default: .right_down. The Go zero value is .left_up; a literal
	// that relies on the default must set it.
	direction PageListDirection
}

// pageListPageIteratorLimit is PageList.PageIterator.Limit.
//
// Zig: union(enum) { none, count: usize, row: Pin }. Only the field named
// by Tag is meaningful; `none` has no payload and no field.
type pageListPageIteratorLimit struct {
	Tag pageListPageIteratorLimitTag

	// Zig: usize. A count of rows.
	Count uint
	Row   PageListPin
}

// pageListPageIteratorLimitTag is the tag of PageList.PageIterator.Limit,
// values numbered from zero in the order listed.
type pageListPageIteratorLimitTag uint8

const (
	pageListPageIteratorLimitNone pageListPageIteratorLimitTag = iota
	pageListPageIteratorLimitCount
	pageListPageIteratorLimitRow
)

// String returns the Zig tag name (@tagName).
func (self pageListPageIteratorLimitTag) String() string { panic(notImplemented) }

// Next is PageList.PageIterator.next.
//
// Zig: returns ?Chunk.
func (self *PageListPageIterator) Next() Optional[PageListPageIteratorChunk] {
	panic(notImplemented)
}

// nextDown is PageList.PageIterator.nextDown.
//
// Zig: returns ?Chunk.
func (self *PageListPageIterator) nextDown() Optional[PageListPageIteratorChunk] {
	panic(notImplemented)
}

// nextUp is PageList.PageIterator.nextUp.
//
// Zig: returns ?Chunk.
func (self *PageListPageIterator) nextUp() Optional[PageListPageIteratorChunk] {
	panic(notImplemented)
}

// PageListPageIteratorChunk is PageList.PageIterator.Chunk.
type PageListPageIteratorChunk struct {
	node *PageListNode

	// Start y index (inclusive) of this chunk in the page.
	start SizeCellCountInt

	// End y index (exclusive) of this chunk in the page.
	end SizeCellCountInt
}

// Rows is PageList.PageIterator.Chunk.rows.
//
// Zig: returns []Row, the page's rows `start` to `end`: a sub-slice of
// `node.data.rows`.
func (self PageListPageIteratorChunk) Rows() []PageRow { panic(notImplemented) }

// FullPage is PageList.PageIterator.Chunk.fullPage: returns true if this
// chunk represents every row in the page.
func (self PageListPageIteratorChunk) FullPage() bool { panic(notImplemented) }

// Overlaps is PageList.PageIterator.Chunk.overlaps: returns true if this
// chunk overlaps with the given other chunk in any way.
func (self PageListPageIteratorChunk) Overlaps(other PageListPageIteratorChunk) bool {
	panic(notImplemented)
}

// PageIterator is PageList.pageIterator.
//
// Return an iterator that iterates through the rows in the tagged area
// of the point. The iterator returns row "chunks", which are the largest
// contiguous set of rows in a single backing page for a given portion of
// the point region.
//
// This is a more efficient way to iterate through the data in a region,
// since you can do simple pointer math and so on.
//
// If bl_pt is non-null, iteration will stop at the bottom left point
// (inclusive). If bl_pt is null, the entire region specified by the point
// tag will be iterated over. tl_pt and bl_pt must be the same tag, and
// bl_pt must be greater than or equal to tl_pt.
//
// If direction is left_up, iteration will go from bl_pt to tl_pt. If
// direction is right_down, iteration will go from tl_pt to bl_pt.
// Both inclusive.
//
// Zig: pub fn pageIterator(self: *const PageList, direction: Direction,
// tl_pt: point.Point, bl_pt: ?point.Point) PageIterator.
func (self *PageList) PageIterator(
	direction PageListDirection,
	tl_pt Point,
	bl_pt Optional[Point],
) PageListPageIterator {
	panic(notImplemented)
}

// GetTopLeft is PageList.getTopLeft: get the top-left of the screen for
// the given tag.
//
// Zig: self is *const PageList. Upstream also calls it with a point.Point,
// which Zig coerces to its tag; in Go pass `pt.Tag`.
func (self *PageList) GetTopLeft(tag PointTag) PageListPin { panic(notImplemented) }

// GetBottomRight is PageList.getBottomRight.
//
// Returns the bottom right of the screen for the given tag. This can
// return null because it is possible that a tag is not in the screen
// (e.g. history does not yet exist).
//
// Zig: self is *const PageList; returns ?Pin.
func (self *PageList) GetBottomRight(tag PointTag) Optional[PageListPin] {
	panic(notImplemented)
}

// totalRows is PageList.totalRows.
//
// The total rows in the screen. This is the actual row count currently
// and not a capacity or maximum.
//
// This is very slow, it traverses the full list of pages to count the
// rows, so it is not pub. This is only used for testing/debugging.
//
// Zig: self is *const PageList; returns usize, a count.
func (self *PageList) totalRows() uint { panic(notImplemented) }

// TotalPages is PageList.totalPages.
//
// The total number of pages in this list. This should only be used
// for tests since it is O(N) over the list of pages.
//
// Zig: self is *const PageList; returns usize, a count.
func (self *PageList) TotalPages() uint { panic(notImplemented) }

// growRows is PageList.growRows.
//
// Grow the number of rows available in the page list by n.
// This is only used for testing so it isn't optimized in any way.
//
// Zig: Allocator.Error!void; n is usize, a count of rows.
func (self *PageList) growRows(n uint) error { panic(notImplemented) }

// ClearDirty is PageList.clearDirty.
//
// Clear all dirty bits on all pages. This is not efficient since it
// traverses the entire list of pages. This is used for testing/debugging.
func (self *PageList) ClearDirty() { panic(notImplemented) }

// IsDirty is PageList.isDirty: returns true if the point is dirty, used
// for testing.
//
// Zig: self is *const PageList.
func (self *PageList) IsDirty(pt Point) bool { panic(notImplemented) }

// markDirty is PageList.markDirty: mark a point as dirty, used for testing.
func (self *PageList) markDirty(pt Point) { panic(notImplemented) }

// PageListPin is PageList.Pin.
//
// Represents an exact x/y coordinate within the screen. This is called
// a "pin" because it is a fixed point within the pagelist direct to
// a specific page pointer and memory offset. The benefit is that this
// point remains valid even through scrolling without any additional work.
//
// A downside is that  the pin is only valid until the pagelist is modified
// in a way that may invalidate page pointers or shuffle rows, such as resizing,
// erasing rows, etc.
//
// A pin can also be "tracked" which means that it will be updated as the
// PageList is modified.
//
// The PageList maintains a list of active pin references and keeps them
// all up to date as the pagelist is modified. This isn't cheap so callers
// should limit the number of active pins as much as possible.
type PageListPin struct {
	node *PageListNode
	// Zig default: 0.
	y SizeCellCountInt
	// Zig default: 0.
	x SizeCellCountInt

	// This is flipped to true for tracked pins that were tracking
	// a page that got pruned for any reason and where the tracked pin
	// couldn't be moved to a sensical location. Users of the tracked
	// pin could use this data and make their own determination of
	// semantics.
	//
	// Zig default: false.
	garbage bool
}

// RowAndCell is PageList.Pin.rowAndCell.
//
// Zig: the result is an anonymous struct.
func (self PageListPin) RowAndCell() struct {
	row  *PageRow
	cell *PageCell
} {
	panic(notImplemented)
}

// PageListPinCellSubset is PageList.Pin.CellSubset.
//
// Zig: enum { all, left, right }, values numbered from zero in the order
// listed.
type PageListPinCellSubset uint8

const (
	PageListPinCellSubsetAll PageListPinCellSubset = iota
	PageListPinCellSubsetLeft
	PageListPinCellSubsetRight
)

// String returns the Zig tag name (@tagName).
func (self PageListPinCellSubset) String() string { panic(notImplemented) }

// Cells is PageList.Pin.cells.
//
// Returns the cells for the row that this pin is on. The subset determines
// what subset of the cells are returned. The "left/right" subsets are
// inclusive of the x coordinate of the pin.
func (self PageListPin) Cells(subset PageListPinCellSubset) []PageCell { panic(notImplemented) }

// Grapheme is PageList.Pin.grapheme.
//
// Returns the grapheme codepoints for the given cell. These are only
// the EXTRA codepoints and not the first codepoint.
//
// Zig: cell is *const pagepkg.Cell; returns ?[]u21.
func (self PageListPin) Grapheme(cell *PageCell) Optional[[]uint32] { panic(notImplemented) }

// Style is PageList.Pin.style: returns the style for the given cell in
// this pin.
//
// Zig: cell is *const pagepkg.Cell.
func (self PageListPin) Style(cell *PageCell) Style { panic(notImplemented) }

// IsDirty is PageList.Pin.isDirty: check if this pin is dirty.
func (self PageListPin) IsDirty() bool { panic(notImplemented) }

// MarkDirty is PageList.Pin.markDirty: mark this pin location as dirty.
func (self PageListPin) MarkDirty() { panic(notImplemented) }

// PageIterator is PageList.Pin.pageIterator.
//
// Iterators. These are the same as PageList iterator funcs but operate
// on pins rather than points. This is MUCH more efficient than calling
// pointFromPin and building up the iterator from points.
//
// The limit pin is inclusive.
//
// Zig: limit is ?Pin.
func (self PageListPin) PageIterator(
	direction PageListDirection,
	limit Optional[PageListPin],
) PageListPageIterator {
	panic(notImplemented)
}

// RowIterator is PageList.Pin.rowIterator.
//
// Zig: limit is ?Pin.
func (self PageListPin) RowIterator(
	direction PageListDirection,
	limit Optional[PageListPin],
) PageListRowIterator {
	panic(notImplemented)
}

// CellIterator is PageList.Pin.cellIterator.
//
// Zig: limit is ?Pin.
func (self PageListPin) CellIterator(
	direction PageListDirection,
	limit Optional[PageListPin],
) PageListCellIterator {
	panic(notImplemented)
}

// PromptIterator is PageList.Pin.promptIterator.
//
// Zig: limit is ?Pin.
func (self PageListPin) PromptIterator(
	direction PageListDirection,
	limit Optional[PageListPin],
) PageListPromptIterator {
	panic(notImplemented)
}

// IsBetween is PageList.Pin.isBetween: returns true if this pin is between
// the top and bottom, inclusive.
//
// Note: this is primarily unit tested as part of the Kitty
// graphics deletion code.
func (self PageListPin) IsBetween(top PageListPin, bottom PageListPin) bool {
	panic(notImplemented)
}

// Before is PageList.Pin.before.
//
// Returns true if self is before other. This is very expensive since
// it requires traversing the linked list of pages. This should not
// be called in performance critical paths.
func (self PageListPin) Before(other PageListPin) bool { panic(notImplemented) }

// Eql is PageList.Pin.eql.
func (self PageListPin) Eql(other PageListPin) bool { panic(notImplemented) }

// Left is PageList.Pin.left: move the pin left n columns. n must fit
// within the size.
//
// Zig: n is usize, a count of columns.
func (self PageListPin) Left(n uint) PageListPin { panic(notImplemented) }

// Right is PageList.Pin.right: move the pin right n columns. n must fit
// within the size.
//
// Zig: n is usize, a count of columns.
func (self PageListPin) Right(n uint) PageListPin { panic(notImplemented) }

// LeftClamp is PageList.Pin.leftClamp: move the pin left n columns,
// stopping at the start of the row.
func (self PageListPin) LeftClamp(n SizeCellCountInt) PageListPin { panic(notImplemented) }

// RightClamp is PageList.Pin.rightClamp: move the pin right n columns,
// stopping at the end of the row.
func (self PageListPin) RightClamp(n SizeCellCountInt) PageListPin { panic(notImplemented) }

// LeftWrap is PageList.Pin.leftWrap.
//
// Move the pin left n cells, wrapping to the previous row as needed.
//
// If the offset goes beyond the top of the screen, returns null.
//
// TODO: Unit tests.
//
// Zig: n is usize, a count of cells; returns ?Pin.
func (self PageListPin) LeftWrap(n uint) Optional[PageListPin] { panic(notImplemented) }

// RightWrap is PageList.Pin.rightWrap.
//
// Move the pin right n cells, wrapping to the next row as needed.
//
// If the offset goes beyond the bottom of the screen, returns null.
//
// TODO: Unit tests.
//
// Zig: n is usize, a count of cells; returns ?Pin.
func (self PageListPin) RightWrap(n uint) Optional[PageListPin] { panic(notImplemented) }

// Down is PageList.Pin.down.
//
// Move the pin down a certain number of rows, or return null if
// the pin goes beyond the end of the screen.
//
// Zig: n is usize, a count of rows; returns ?Pin.
func (self PageListPin) Down(n uint) Optional[PageListPin] { panic(notImplemented) }

// Up is PageList.Pin.up.
//
// Move the pin up a certain number of rows, or return null if
// the pin goes beyond the start of the screen.
//
// Zig: n is usize, a count of rows; returns ?Pin.
func (self PageListPin) Up(n uint) Optional[PageListPin] { panic(notImplemented) }

// PageListPinOverflow is the anonymous union(enum) that
// PageList.Pin.downOverflow and PageList.Pin.upOverflow return:
//
//	union(enum) {
//	    offset: Pin,
//	    overflow: struct { end: Pin, remaining: usize },
//	}
//
// The name is not Zig's, and Zig has one such anonymous type per function;
// they are identical and share this type. Only the field named by Tag is
// meaningful.
type PageListPinOverflow struct {
	Tag PageListPinOverflowTag

	Offset   PageListPin
	Overflow struct {
		end PageListPin
		// Zig: usize. A count of rows.
		remaining uint
	}
}

// PageListPinOverflowTag is the tag of PageListPinOverflow, values numbered
// from zero in the order listed.
type PageListPinOverflowTag uint8

const (
	PageListPinOverflowOffset PageListPinOverflowTag = iota
	PageListPinOverflowOverflow
)

// String returns the Zig tag name (@tagName).
func (self PageListPinOverflowTag) String() string { panic(notImplemented) }

// DownOverflow is PageList.Pin.downOverflow.
//
// Move the offset down n rows. If the offset goes beyond the
// end of the screen, return the overflow amount.
//
// Zig: n is usize, a count of rows.
func (self PageListPin) DownOverflow(n uint) PageListPinOverflow { panic(notImplemented) }

// UpOverflow is PageList.Pin.upOverflow.
//
// Move the offset up n rows. If the offset goes beyond the
// start of the screen, return the overflow amount.
//
// Zig: n is usize, a count of rows.
func (self PageListPin) UpOverflow(n uint) PageListPinOverflow { panic(notImplemented) }

// PageListCell is PageList.Cell.
//
// `row` and `cell` are Go pointers into the page's slices.
type PageListCell struct {
	node    *PageListNode
	row     *PageRow
	cell    *PageCell
	row_idx SizeCellCountInt
	col_idx SizeCellCountInt
}

// IsDirty is PageList.Cell.isDirty.
//
// Returns true if this cell is marked as dirty.
//
// This is not very performant this is primarily used for assertions
// and testing.
func (self PageListCell) IsDirty() bool { panic(notImplemented) }

// Style is PageList.Cell.style.
//
// Get the cell style.
//
// Not meant for non-test usage since this is inefficient.
func (self PageListCell) Style() Style { panic(notImplemented) }

// ScreenPoint is PageList.Cell.screenPoint.
//
// Gets the screen point for the given cell.
//
// This is REALLY expensive/slow so it isn't pub. This was built
// for debugging and tests. If you have a need for this outside of
// this file then consider a different approach and ask yourself very
// carefully if you really need this.
func (self PageListCell) ScreenPoint() Point { panic(notImplemented) }
