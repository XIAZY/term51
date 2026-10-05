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
// `page.memory.len`: `Page.layout(capacity).total_size` of the page's own
// capacity, for pooled and non-pooled pages alike (measured on upstream: a
// fresh pooled 3353-column page has memory.len 430,080 while std_size is
// 454,656). Whether a page is pooled is decided by comparing that length
// with `std_size`; the pool buffer itself is std_size long, which is what
// page_size counts on creation (see the page_size drift finding).
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

import (
	"errors"
	"math"
)

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
	new_node.prev = node
	if next_node := node.next; next_node != nil {
		// Intermediate node.
		new_node.next = next_node
		next_node.prev = new_node
	} else {
		// Last element of the list.
		new_node.next = nil
		list.last = new_node
	}
	node.next = new_node
}

// InsertBefore is DoublyLinkedList.insertBefore: insert a new node before
// an existing one.
func (list *PageListList) InsertBefore(node *PageListNode, new_node *PageListNode) {
	new_node.next = node
	if prev_node := node.prev; prev_node != nil {
		// Intermediate node.
		new_node.prev = prev_node
		prev_node.next = new_node
	} else {
		// First element of the list.
		new_node.prev = nil
		list.first = new_node
	}
	node.prev = new_node
}

// Append is DoublyLinkedList.append: insert a new node at the end of the
// list.
func (list *PageListList) Append(new_node *PageListNode) {
	if last := list.last; last != nil {
		// Insert after last.
		list.InsertAfter(last, new_node)
	} else {
		// Empty list.
		list.Prepend(new_node)
	}
}

// Prepend is DoublyLinkedList.prepend: insert a new node at the beginning
// of the list.
func (list *PageListList) Prepend(new_node *PageListNode) {
	if first := list.first; first != nil {
		// Insert before first.
		list.InsertBefore(first, new_node)
	} else {
		// Empty list.
		list.first = new_node
		list.last = new_node
		new_node.prev = nil
		new_node.next = nil
	}
}

// Remove is DoublyLinkedList.remove: remove a node from the list.
func (list *PageListList) Remove(node *PageListNode) {
	if prev_node := node.prev; prev_node != nil {
		// Intermediate node.
		prev_node.next = node.next
	} else {
		// First element of the list.
		list.first = node.next
	}

	if next_node := node.next; next_node != nil {
		// Intermediate node.
		next_node.prev = node.prev
	} else {
		// Last element of the list.
		list.last = node.prev
	}
}

// PopFirst is DoublyLinkedList.popFirst: remove and return the first node
// in the list, or nil if the list is empty.
func (list *PageListList) PopFirst() *PageListNode {
	first := list.first
	if first == nil {
		return nil
	}
	list.Remove(first)
	return first
}

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
func pageListStdSize() uint64 {
	return PageLayoutOf(PageStdCapacity).total_size
}

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
func pageListPagePoolItemSize() uint64 {
	// Zig (std.heap.MemoryPoolAligned): item_size = @max(@sizeOf(Node),
	// @sizeOf(Item)), Item = [std_size]u8; the free-list node is one
	// pointer, smaller than std_size.
	return pageListStdSize()
}

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
func (self *pageListPinSet) PutNoClobber(key *PageListPin) error {
	// Zig: `assert(!result.found_existing)`.
	for _, k := range self.keys {
		assert(k != key)
	}
	self.keys = append(self.keys, key)
	return nil
}

// SwapRemove is std's swapRemove: if key is present, remove it by moving
// the last key into its place and return true; otherwise return false.
func (self *pageListPinSet) SwapRemove(key *PageListPin) bool {
	for i, k := range self.keys {
		if k != key {
			continue
		}
		last := len(self.keys) - 1
		self.keys[i] = self.keys[last]
		self.keys[last] = nil
		self.keys = self.keys[:last]
		return true
	}
	return false
}

// Keys is std's keys: the keys in order. The slice is the set's own and is
// only valid until the set is modified.
func (self *pageListPinSet) Keys() []*PageListPin { return self.keys }

// Count is std's count: the number of keys.
func (self *pageListPinSet) Count() uint { return uint(len(self.keys)) }

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
func (self PageListMemoryPoolResetModeTag) String() string {
	return pageListMemoryPoolResetModeTagNames[self]
}

// pageListMemoryPoolResetModeTagNames are the tag names of
// PageListMemoryPoolResetModeTag, indexed by value (Zig's @tagName).
var pageListMemoryPoolResetModeTagNames = [...]string{"free_all", "retain_capacity", "retain_with_limit"}

// PageListMemoryPoolInit is PageList.MemoryPool.init.
//
// Zig: pub fn init(gen_alloc: Allocator, page_alloc: Allocator,
// preheat: usize) Allocator.Error!MemoryPool. Both allocator parameters are
// dropped. `preheat` is a number of nodes and pages (the pin pool is always
// preheated with 8). The error is ErrOutOfMemory (dcs.go).
func PageListMemoryPoolInit(preheat uint) (PageListMemoryPool, error) {
	// Zig: NodePool.initPreheated(gen_alloc, preheat),
	// PagePool.initPreheated(page_alloc, preheat) and
	// PinPool.initPreheated(gen_alloc, 8), each with an errdefer deinit.
	// The pools hold no memory in Go (see the file header).
	_ = preheat
	return PageListMemoryPool{
		nodes: pageListNodePool{},
		pages: pageListPagePool{},
		pins:  pageListPinPool{},
	}, nil
}

// Deinit is PageList.MemoryPool.deinit.
func (self *PageListMemoryPool) Deinit() {
	// Zig: self.pages.deinit(); self.nodes.deinit(); self.pins.deinit().
	// The pools hold no memory in Go (see the file header).
}

// Reset is PageList.MemoryPool.reset.
func (self *PageListMemoryPool) Reset(mode PageListMemoryPoolResetMode) {
	// Zig: `_ = self.pages.reset(mode)`, and the same for nodes and pins.
	// The pools hold no memory in Go (see the file header).
	_ = mode
}

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

	// NOT a field of upstream's PageList: the term51 "candidate B"
	// verification patch (term51 patches-candidates/
	// B-0001-tolerate-transient-pin-check-in-resize.patch), which
	// docs/mapping.md ("Requirements the wireframe tests impose on the
	// implementation") requires. True while Resize runs, so that the
	// transient "viewport pin has too few rows" state inside the padding
	// grow()s is tolerated exactly as release builds (which compile the
	// check out) tolerate it. The check is enforced again when Resize
	// returns. See docs/upstream-findings.md and docs/deliberate-fixes.md
	// entry 3 in the term51 repository.
	term51_resizing bool
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
func (self PageListViewportTag) String() string {
	return pageListViewportTagNames[self]
}

// pageListViewportTagNames are the tag names of PageListViewportTag,
// indexed by value (Zig's @tagName).
var pageListViewportTagNames = [...]string{"active", "top", "pin"}

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
	// Invariant required to ensure our divCeil below cannot overflow.
	//
	// Zig: a comptime block, `divCeil(usize, maxInt(CellCountInt), 1)
	// catch unreachable`; a compile-time check with nothing to do at run
	// time.

	// Get our capacity to fit our rows. If the cols are too big, it may
	// force less rows than we want meaning we need more than one page to
	// represent a viewport.
	cap := pageListInitialCapacity(cols)

	// Calculate the number of standard sized pages we need to represent
	// an active area.
	var pages_exact uint64
	if cap.rows >= rows {
		pages_exact = 1
	} else {
		// Not possible to fail (Zig: `catch unreachable`):
		// - initialCapacity guarantees at least 1 row
		// - numerator/denominator can't overflow because of comptime check above
		pages_exact = pageStdMathDivCeil(uint64(rows), uint64(cap.rows))
	}

	// We always need at least one page extra so that we
	// can fit partial pages to spread our active area across two pages.
	// Even for caps that can't fit all rows in a single page, we add one
	// because the most extra space we need at any given time is only
	// the partial amount of one page.
	pages := pages_exact + 1
	assert(pages >= 2)

	// log.debug("minMaxSize cols={} rows={} cap={} pages={}", .{
	//     cols,
	//     rows,
	//     cap,
	//     pages,
	// });

	return pageListPagePoolItemSize() * pages
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
func pageListInitialCapacity(cols SizeCellCountInt) PageCapacity {
	// This is an important invariant that ensures that this function
	// can never return an error. We verify here that our standard capacity
	// when increased to maximum possible columns can always support at
	// least one row in memory.
	//
	// IF THIS EVER FAILS: We probably need to modify our logic below
	// to reduce other elements of the capacity (styles, graphemes, etc.).
	// But, instead, I recommend taking a step back and re-evaluating
	// life choices.
	//
	// Zig: a comptime block asserting `Page.layout(cap).total_size <=
	// size.max_page_size` for std_capacity with cols = maxInt; a
	// compile-time check with nothing to do at run time.

	if cap, err := PageStdCapacity.Adjust(
		PageCapacityAdjustment{cols: Some(cols)},
	); err == nil {
		// If we can adjust our standard capacity, we fit within the
		// standard size and we're good!
		return cap
	}
	// Ensure our error set doesn't change.
	// Zig: comptime assert(@TypeOf(err) == error{OutOfMemory}).

	// This code path means that our standard capacity can't even
	// accommodate our column count! The only solution is to increase
	// our capacity and go non-standard.
	cap := PageStdCapacity
	cap.cols = cols
	return cap
}

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
	// Zig: const tw = init_tw; the tw.check calls are omitted (see the
	// file header).

	// The screen starts with a single page that is the entire viewport,
	// and we'll split it thereafter if it gets too large and add more as
	// necessary.
	pool, err := PageListMemoryPoolInit(pageListPagePreheat)
	if err != nil {
		return PageList{}, err
	}
	// errdefer pool.deinit(): the pool holds no memory in Go.

	var page_serial uint64 = 0
	page_list, page_size, err := pageListInitPages(
		&pool,
		&page_serial,
		cols,
		rows,
	)
	if err != nil {
		return PageList{}, err
	}

	// Get our minimum max size, see doc comments for more details.
	min_max_size := pageListMinMaxSize(cols, rows)

	// We always track our viewport pin to ensure this is never an allocation
	viewport_pin := new(PageListPin) // pool.pins.create()
	assert(page_list.first != nil)   // page_list.first.?
	*viewport_pin = PageListPin{node: page_list.first}
	var tracked_pins pageListPinSet
	// errdefer tracked_pins.deinit(pool.alloc): Go memory.

	if err := tracked_pins.PutNoClobber(viewport_pin); err != nil {
		return PageList{}, err
	}

	// errdefer comptime unreachable;
	explicit_max_size := uint64(math.MaxUint64) // std.math.maxInt(usize)
	if max_size.Valid {
		explicit_max_size = max_size.Value
	}
	result := PageList{
		cols:                    cols,
		rows:                    rows,
		pool:                    pool,
		pages:                   page_list,
		page_serial:             page_serial,
		page_serial_min:         0,
		page_size:               page_size,
		explicit_max_size:       explicit_max_size,
		min_max_size:            min_max_size,
		total_rows:              uint(rows),
		tracked_pins:            tracked_pins,
		viewport:                PageListViewport{Tag: PageListViewportActive},
		viewport_pin:            viewport_pin,
		viewport_pin_row_offset: None[uint](),
	}
	result.AssertIntegrity()
	return result, nil
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
	// Zig: const tw = initPages_tw; the tw.check calls are omitted (see
	// the file header).
	_ = pool

	var page_list PageListList
	var page_size uint64 = 0

	// Add pages as needed to create our initial viewport.
	cap := pageListInitialCapacity(cols)
	layout := PageLayoutOf(cap)
	std_size := pageListStdSize()
	pooled := layout.total_size <= std_size
	// Zig: const page_alloc = pool.pages.arena.child_allocator; Go memory.

	// Guaranteed by comptime checks in initialCapacity but
	// redundant here for safety.
	assert(layout.total_size <= SizeMaxPageSize)

	// If we have an error, we need to clean up our non-standard pages
	// since they're not in the pool.
	//
	// Zig: an errdefer that frees the buffers of non-standard pages; the
	// garbage collector frees in Go.

	rem := rows
	for rem > 0 {
		node := new(PageListNode) // pool.nodes.create()
		// errdefer pool.nodes.destroy(node): Go memory.

		// Zig: the buffer is a pool item of std_size bytes when pooled,
		// otherwise an allocation of layout.total_size bytes. Its length
		// is what page_size counts. In Go the buffer is the page's rows
		// and cells.
		var page_buf_len uint64
		if pooled {
			page_buf_len = std_size
		} else {
			page_buf_len = layout.total_size
		}
		page_rows := make([]PageRow, layout.capacity.rows)
		page_cells := make([]PageCell, uint(layout.capacity.cols)*uint(layout.capacity.rows))
		// errdefer destroy/free of page_buf: Go memory.

		// In runtime safety modes we have to memset because the Zig allocator
		// interface will always memset to 0xAA for undefined. In non-safe modes
		// we use a page allocator and the OS guarantees zeroed memory.
		//
		// Zig: @memset(page_buf, 0). Go memory is zeroed.

		// Initialize the first set of pages to contain our viewport so that
		// the top of the first page is always the active area.
		*node = PageListNode{
			data:   pageInitBuf(page_rows, page_cells, layout),
			serial: *serial,
		}
		node.data.size.rows = min(rem, node.data.capacity.rows)
		rem -= node.data.size.rows

		// Add the page to the list
		page_list.Append(node)
		page_size += page_buf_len
		// errdefer comptime unreachable;

		// Increment our serial
		*serial += 1
	}

	assert(page_list.first != nil)

	return page_list, page_size, nil
}

// AssertIntegrity is PageList.assertIntegrity.
//
// Assert that the PageList is in a valid state. This is a no-op in
// release builds.
//
// Zig: self is *const PageList.
func (self *PageList) AssertIntegrity() {
	if !pageSlowRuntimeSafety {
		return
	}

	if err := self.verifyIntegrity(); err != nil {
		// Zig: log.err
		warn("page_list", "PageList integrity check failed: error.%v", err)
		panic("PageList integrity check failed")
	}
}

// PauseIntegrityChecks is PageList.pauseIntegrityChecks.
//
// Pause or resume integrity checks. This is useful when you're doing
// a multi-step operation that temporarily leaves the PageList in an
// inconsistent state.
func (self *PageList) PauseIntegrityChecks(pause bool) {
	if !pageSlowRuntimeSafety {
		return
	}
	if pause {
		// Zig: += traps on overflow.
		assert(self.pause_integrity_checks < ^uint(0))
		self.pause_integrity_checks += 1
	} else {
		// Zig: -= traps on underflow.
		assert(self.pause_integrity_checks > 0)
		self.pause_integrity_checks -= 1
	}
}

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
func (self *PageList) verifyIntegrity() error {
	if !pageSlowRuntimeSafety {
		return nil
	}
	if self.pause_integrity_checks > 0 {
		return nil
	}

	// Our viewport pin should never be garbage
	assert(!self.viewport_pin.garbage)

	// Grab our total rows
	var actual_total uint = 0
	{
		node_ := self.pages.first
		for node_ != nil {
			node := node_
			actual_total += uint(node.data.size.rows)
			node_ = node.next

			// While doing this traversal, verify no node has a serial
			// number lower than our min.
			if node.serial < self.page_serial_min {
				warn(
					"page_list",
					"PageList integrity violation: page serial too low serial=%d min=%d",
					node.serial, self.page_serial_min,
				)
				return ErrPageSerialInvalid
			}
		}
	}

	// Verify that our cached total_rows matches the actual row count
	if actual_total != self.total_rows {
		warn(
			"page_list",
			"PageList integrity violation: total_rows mismatch cached=%d actual=%d",
			self.total_rows, actual_total,
		)
		return ErrTotalRowsMismatch
	}

	// Verify that all our tracked pins point to valid pages.
	for _, p := range self.tracked_pins.Keys() {
		if !self.PinIsValid(*p) {
			return ErrTrackedPinInvalid
		}
	}

	if self.viewport.Tag == PageListViewportPin {
		// Verify that our viewport pin row offset is correct.
		//
		// Zig: a labeled block `offset:` whose fall-through returns the
		// error.
		var actual_offset uint
		found := false
		{
			var offset uint = 0
			for n := self.pages.last; n != nil; n = n.prev {
				offset += uint(n.data.size.rows)
				if n == self.viewport_pin.node {
					// Zig: usize `-=` and `-` trap on underflow.
					assert(offset >= uint(self.viewport_pin.y))
					offset -= uint(self.viewport_pin.y)
					assert(self.total_rows >= offset)
					actual_offset = self.total_rows - offset
					found = true
					break
				}
			}
		}
		if !found {
			warn(
				"page_list",
				"PageList integrity violation: viewport pin not in list",
			)
			return ErrViewportPinOffsetMismatch
		}

		if cached := self.viewport_pin_row_offset; cached.Valid {
			cached_offset := cached.Value
			if cached_offset != actual_offset {
				warn(
					"page_list",
					"PageList integrity violation: viewport pin offset mismatch cached=%d actual=%d",
					cached_offset, actual_offset,
				)
				return ErrViewportPinOffsetMismatch
			}
		}

		// Ensure our viewport has enough rows.
		//
		// term51 candidate B (docs/mapping.md): not reported while Resize
		// runs; Resize asserts integrity again when it returns.
		rows := self.total_rows - actual_offset
		if rows < uint(self.rows) && !self.term51_resizing {
			warn(
				"page_list",
				"PageList integrity violation: viewport pin rows too small rows=%d needed=%d",
				rows, self.rows,
			)
			return ErrViewportPinInsufficientRows
		}
	}

	return nil
}

// Deinit is PageList.deinit.
//
// Deinit the pagelist. If you own the memory pool (used clonePool) then
// this will reset the pool and retain capacity.
//
// Kept although it mostly frees: it asserts integrity first, and every
// test calls it.
func (self *PageList) Deinit() {
	// Verify integrity before cleanup
	self.AssertIntegrity()

	// Always deallocate our hashmap.
	//
	// Zig: self.tracked_pins.deinit(self.pool.alloc); Go memory.

	// Go through our linked list and deallocate all pages that are
	// not standard size.
	//
	// Zig: frees `node.data.memory` of every page whose
	// `memory.len > std_size`; Go memory.

	// Deallocate all the pages. We don't need to deallocate the list or
	// nodes because they all reside in the pool.
	self.pool.Deinit()
}

// Reset is PageList.reset.
//
// Reset the PageList back to an empty state. This is similar to
// deinit and reinit but it importantly preserves the pointer
// stability of tracked pins (they're moved to the top-left since
// all contents are cleared).
//
// This can't fail because we always retain at least enough allocated
// memory to fit the active area.
func (self *PageList) Reset() {
	defer self.AssertIntegrity()

	// We need enough pages/nodes to keep our active area. This should
	// never fail since we by definition have allocated a page already
	// that fits our size but I'm not confident to make that assertion.
	cap := pageListInitialCapacity(self.cols)
	assert(cap.rows > 0)

	// The number of pages we need is the number of rows in the active
	// area divided by the row capacity of a page.
	page_count := pageStdMathDivCeil(
		uint64(self.rows),
		uint64(cap.rows),
	) // catch unreachable

	// Before resetting our pools we need to free any pages that
	// are non-standard size since those were allocated outside
	// the pool.
	//
	// Zig: frees `node.data.memory` of every page whose
	// `memory.len > std_size`; Go memory.

	// Reset our pools to free as much memory as possible while retaining
	// the capacity for at least the minimum number of pages we need.
	// The return value is whether memory was reclaimed or not, but in
	// either case the pool is left in a valid state.
	//
	// Zig: `_ = self.pool.pages.reset(.{ .retain_with_limit = page_count *
	// PagePool.item_size })` and the same for `self.pool.nodes` with
	// `NodePool.item_size`: std methods on pools that hold no memory in Go
	// (see the file header).
	_ = page_count

	// Our page pool relies on mmap to zero our page memory. Since we're
	// retaining a certain amount of memory, it won't use mmap and won't
	// be zeroed. This block zeroes out all the memory in the pool arena.
	//
	// Zig: a walk over the page arena's buffers with @memset; initPages
	// below gives every page fresh, zeroed Go memory.

	// Initialize our pages. This should not be able to fail since
	// we retained the capacity for the minimum number of pages we need.
	pages, page_size, err := pageListInitPages(
		&self.pool,
		&self.page_serial,
		self.cols,
		self.rows,
	)
	if err != nil {
		panic("initPages failed")
	}
	self.pages, self.page_size = pages, page_size

	// Our total rows always goes back to the default
	self.total_rows = uint(self.rows)

	// Update all our tracked pins to point to our first page top-left
	// and mark them as garbage, because it got mangled in a way where
	// semantically it really doesn't make sense.
	{
		for _, p := range self.tracked_pins.Keys() {
			assert(self.pages.first != nil) // self.pages.first.?
			p.node = self.pages.first
			p.x = 0
			p.y = 0
			p.garbage = true
		}

		// Our viewport pin is never garbage
		self.viewport_pin.garbage = false
	}

	// Move our viewport back to the active area since everything is gone.
	self.viewport = PageListViewport{Tag: PageListViewportActive}
}

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
func (self *PageList) Clone(opts PageListClone) (PageList, error) {
	it := self.PageIterator(
		PageListDirectionRightDown,
		opts.top,
		opts.bot,
	)

	// First, count our pages so our preheat is exactly what we need.
	it_copy := it
	var page_count uint
	{
		var count uint = 0
		for it_copy.Next().Valid {
			count += 1
		}
		page_count = count
	}

	// Setup our pool
	pool, err := PageListMemoryPoolInit(page_count)
	if err != nil {
		return PageList{}, err
	}
	// errdefer pool.deinit(): the pool holds no memory in Go.

	// Create our viewport. In a clone, the viewport always goes
	// to the top.
	viewport_pin := new(PageListPin) // pool.pins.create()
	var tracked_pins pageListPinSet
	// errdefer tracked_pins.deinit(pool.alloc): Go memory.
	if err := tracked_pins.PutNoClobber(viewport_pin); err != nil {
		return PageList{}, err
	}

	// Our list of pages
	var page_list PageListList
	// Zig: an errdefer that frees the buffers of non-standard pages in
	// page_list; Go memory.

	// Copy our pages
	var page_serial uint64 = 0
	var total_rows uint = 0
	var page_size uint64 = 0
	for chunk_ := it.Next(); chunk_.Valid; chunk_ = it.Next() {
		chunk := chunk_.Value

		// Clone the page. We have to use createPageExt here because
		// we don't know if the source page has a standard size.
		node, err := pageListCreatePageExt(
			&pool,
			chunk.node.data.capacity,
			&page_serial,
			&page_size,
		)
		if err != nil {
			return PageList{}, err
		}
		assert(chunk.end >= chunk.start) // Zig: u16 `-` traps
		assert(node.data.capacity.rows >= chunk.end-chunk.start)

		// Zig: `defer node.data.assertIntegrity()` at the end of each
		// iteration, also when an error leaves it: the rest of the body is
		// a closure.
		err = func() error {
			defer node.data.AssertIntegrity()
			node.data.size.rows = chunk.end - chunk.start
			node.data.size.cols = chunk.node.data.size.cols
			if err := node.data.CloneFrom(
				&chunk.node.data,
				uint(chunk.start),
				uint(chunk.end),
			); err != nil {
				return err
			}

			node.data.dirty = chunk.node.data.dirty

			page_list.Append(node)

			total_rows += uint(node.data.size.rows)

			// Remap our tracked pins by changing the page and
			// offsetting the Y position based on the chunk start.
			if remap := opts.tracked_pins; remap != nil {
				pin_keys := self.tracked_pins.Keys()
				for _, p := range pin_keys {
					// We're only interested in pins that were within the chunk.
					if p.node != chunk.node ||
						p.y < chunk.start ||
						p.y >= chunk.end {
						continue
					}
					new_p := new(PageListPin) // pool.pins.create()
					*new_p = *p
					new_p.node = node
					new_p.y -= chunk.start
					// Zig: remap.putNoClobber asserts that the key is new.
					_, found := (*remap)[p]
					assert(!found)
					(*remap)[p] = new_p
					if err := tracked_pins.PutNoClobber(new_p); err != nil {
						return err
					}
				}
			}
			return nil
		}()
		if err != nil {
			return PageList{}, err
		}
	}

	// Initialize our viewport pin to point to the first cloned page
	// so it points to valid memory.
	assert(page_list.first != nil) // page_list.first.?
	*viewport_pin = PageListPin{node: page_list.first}

	result := PageList{
		pool:                    pool,
		pages:                   page_list,
		page_serial:             page_serial,
		page_serial_min:         0,
		page_size:               page_size,
		explicit_max_size:       self.explicit_max_size,
		min_max_size:            self.min_max_size,
		cols:                    self.cols,
		rows:                    self.rows,
		total_rows:              total_rows,
		tracked_pins:            tracked_pins,
		viewport:                PageListViewport{Tag: PageListViewportActive},
		viewport_pin:            viewport_pin,
		viewport_pin_row_offset: None[uint](),
	}

	// We always need to have enough rows for our viewport because this is
	// a pagelist invariant that other code relies on.
	if total_rows < uint(self.rows) {
		length := uint(self.rows) - total_rows
		for range length {
			if _, err := result.Grow(); err != nil {
				return PageList{}, err
			}

			// Clear the row. This is not very fast but in reality right
			// now we rarely clone less than the active area and if we do
			// the area is by definition very small.
			last := result.pages.last
			assert(last.data.size.rows > 0) // Zig: u16 `- 1` traps
			row := &last.data.rows[last.data.size.rows-1]
			last.data.ClearCells(row, 0, uint(result.cols))
		}

		// Update our total rows to be our row size.
		result.total_rows = uint(result.rows)
	}

	result.AssertIntegrity()
	return result, nil
}

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
func (self *PageList) Resize(opts PageListResize) (err error) {
	defer self.AssertIntegrity()
	// term51 candidate B (docs/mapping.md): runs before the
	// AssertIntegrity defer above.
	self.term51_resizing = true
	defer func() { self.term51_resizing = false }()

	// Zig: `if (comptime std.debug.runtime_safety)`; the reference is a
	// safe build.
	{
		// Resize does not work with 0 values, this should be protected
		// upstream
		if opts.cols.Valid {
			assert(opts.cols.Value > 0)
		}
		if opts.rows.Valid {
			assert(opts.rows.Value > 0)
		}
	}

	// Resizing (especially with reflow) can cause our row offset to
	// become invalid. Rather than do something fancy like we do other
	// places and try to update it in place, we just invalidate it because
	// its too easy to get the logic wrong in here.
	self.viewport_pin_row_offset = None[uint]()

	if !opts.reflow {
		return self.resizeWithoutReflow(opts)
	}

	// Recalculate our minimum max size. This allows grow to work properly
	// when increasing beyond our initial minimum max size or explicit max
	// size to fit the active area.
	old_min_max_size := self.min_max_size
	{
		cols := self.cols
		if opts.cols.Valid {
			cols = opts.cols.Value
		}
		rows := self.rows
		if opts.rows.Valid {
			rows = opts.rows.Value
		}
		self.min_max_size = pageListMinMaxSize(cols, rows)
	}
	defer func() {
		// errdefer
		if err != nil {
			self.min_max_size = old_min_max_size
		}
	}()

	// On reflow, the main thing that causes reflow is column changes. If
	// only rows change, reflow is impossible. So we change our behavior based
	// on the change of columns.
	cols := self.cols
	if opts.cols.Valid {
		cols = opts.cols.Value
	}
	switch {
	case cols == self.cols:
		if err := self.resizeWithoutReflow(opts); err != nil {
			return err
		}

	case cols > self.cols:
		// We grow rows after cols so that we can do our unwrapping/reflow
		// before we do a no-reflow grow.
		if err := self.resizeCols(cols, opts.cursor); err != nil {
			return err
		}
		if err := self.resizeWithoutReflow(opts); err != nil {
			return err
		}

	default: // .lt
		// We first change our row count so that we have the proper amount
		// we can use when shrinking our cols.
		copy_ := opts
		copy_.cols = Some(self.cols)
		if err := self.resizeWithoutReflow(copy_); err != nil {
			return err
		}
		if err := self.resizeCols(cols, opts.cursor); err != nil {
			return err
		}
	}

	// Various resize operations can change our total row count such
	// that our viewport pin is now in the active area and has insufficient
	// space. We need to check for this case and fix it up.
	switch self.viewport.Tag {
	case PageListViewportPin:
		if self.pinIsActive(*self.viewport_pin) {
			self.viewport = PageListViewport{Tag: PageListViewportActive}
		}
	case PageListViewportActive, PageListViewportTop:
	}

	return nil
}

// resizeCols is PageList.resizeCols: resize the pagelist with reflow by
// adding or removing columns.
//
// Zig: fn resizeCols(self: *PageList, cols: size.CellCountInt,
// cursor: ?Resize.Cursor) Allocator.Error!void.
func (self *PageList) resizeCols(
	cols SizeCellCountInt,
	cursor Optional[PageListResizeCursor],
) (err error) {
	assert(cols != self.cols)

	// Update our cols. We have to do this early because grow() that we
	// may call below relies on this to calculate the proper page size.
	self.cols = cols

	// If we have a cursor position (x,y), then we try under any col resizing
	// to keep the same number remaining active rows beneath it. This is a
	// very special case if you can imagine clearing the screen (i.e.
	// scrollClear), having an empty active area, and then resizing to less
	// cols then we don't want the active area to "jump" to the bottom and
	// pull down scrollback.
	//
	// Zig: an optional anonymous struct.
	type preservedCursor struct {
		tracked_pin    *PageListPin
		remaining_rows uint
		wrapped_rows   uint
	}
	var preserved_cursor Optional[preservedCursor]
	if cursor.Valid {
		c := cursor.Value
		// Zig: `orelse break :cursor null`.
		if p := self.Pin(Point{Tag: PointTagActive, Active: PointCoordinate{
			x: c.x,
			y: uint32(c.y),
		}}); p.Valid {
			active_pin := self.Pin(Point{Tag: PointTagActive})

			// We count how many wraps the cursor had before it to begin with
			// so that we can offset any additional wraps to avoid pushing the
			// original row contents in to the scrollback.
			var wrapped uint
			{
				var wrapped_ uint = 0

				row_it := p.Value.RowIterator(PageListDirectionLeftUp, active_pin)
				for next := row_it.Next(); next.Valid; next = row_it.Next() {
					row := next.Value.RowAndCell().row
					if row.wrap_continuation {
						wrapped_ += 1
					}
				}

				wrapped = wrapped_
			}

			tracked_pin, err := self.TrackPin(p.Value)
			if err != nil {
				return err
			}
			// Zig: `self.rows - c.y - 1` on u16 traps on underflow.
			assert(c.y < self.rows)
			preserved_cursor = Some(preservedCursor{
				tracked_pin:    tracked_pin,
				remaining_rows: uint(self.rows - c.y - 1),
				wrapped_rows:   wrapped,
			})
		}
	}
	defer func() {
		if preserved_cursor.Valid {
			self.UntrackPin(preserved_cursor.Value.tracked_pin)
		}
	}()

	// Create the first node that contains our reflow.
	var first_rewritten_node *PageListNode
	{
		page := &self.pages.first.data
		cap, aerr := page.capacity.Adjust(
			PageCapacityAdjustment{cols: Some(cols)},
		)
		if aerr != nil {
			// Zig: comptime assert(@TypeOf(err) == error{OutOfMemory}).

			// We verify all maxed out page layouts work.
			cap = page.capacity
			cap.cols = cols

			// We're growing columns so we can only get less rows so use
			// the lesser of our capacity and size so we minimize wasted
			// rows.
			cap.rows = min(page.size.rows, cap.rows)
		}

		node, err := self.createPage(cap)
		if err != nil {
			return err
		}
		node.data.size.rows = 1
		first_rewritten_node = node
	}

	// We need to grab our rowIterator now before we rewrite our
	// linked list below.
	it := self.RowIterator(
		PageListDirectionRightDown,
		Point{Tag: PointTagScreen},
		None[Point](),
	)
	defer func() {
		// errdefer
		if err == nil {
			return
		}
		// If an error occurs, we're in a pretty disastrous broken state,
		// but we should still try to clean up our leaked memory. Free
		// any of the remaining orphaned pages from before. If we reflowed
		// successfully this will be null.
		var node_ *PageListNode
		if it.chunk.Valid {
			node_ = it.chunk.Value.node
		}
		for node_ != nil {
			node := node_
			node_ = node.next
			self.destroyNode(node)
		}
	}()

	// Set our new page as the only page. This orphans the existing pages
	// in the list, but that's fine since we're gonna delete them anyway.
	self.pages.first = first_rewritten_node
	self.pages.last = first_rewritten_node

	// Reflow all our rows.
	{
		reflow_cursor := pageListReflowCursorInit(first_rewritten_node)
		for row_ := it.Next(); row_.Valid; row_ = it.Next() {
			row := row_.Value
			if err := reflow_cursor.reflowRow(self, row); err != nil {
				return err
			}

			// Once we're done reflowing a page, destroy it immediately.
			// This frees memory and makes it more likely in memory
			// constrained environments that the next reflow will work.
			if uint(row.y) == uint(row.node.data.size.rows)-1 {
				self.destroyNode(row.node)
			}
		}

		// At the end of the reflow, setup our total row cache
		// log.warn("total old={} new={}", .{ self.total_rows, reflow_cursor.total_rows });
		self.total_rows = reflow_cursor.total_rows
	}

	// If our total rows is less than our active rows, we need to grow.
	// This can happen if you're growing columns such that enough active
	// rows unwrap that we no longer have enough.
	node_it := self.pages.first
	var total uint = 0
	for ; node_it != nil; node_it = node_it.next {
		node := node_it
		total += uint(node.data.size.rows)
		if total >= uint(self.rows) {
			break
		}
	}
	if node_it == nil {
		// Zig: the `else` of the while, reached without a break.
		for range uint(self.rows) - total {
			if _, err := self.Grow(); err != nil {
				return err
			}
		}
	}

	// See preserved_cursor setup for why.
	if preserved_cursor.Valid {
		c := preserved_cursor.Value
		// Zig: `orelse break :cursor`.
		if active_pt := self.PointFromPin(
			PointTagActive,
			*c.tracked_pin,
		); active_pt.Valid {
			active_pin := self.Pin(Point{Tag: PointTagActive})

			// We need to determine how many rows we wrapped from the original
			// and subtract that from the remaining rows we expect because if
			// we wrap down we don't want to push our original row contents into
			// the scrollback.
			var wrapped uint
			{
				var wrapped_ uint = 0

				row_it := c.tracked_pin.RowIterator(PageListDirectionLeftUp, active_pin)
				for next := row_it.Next(); next.Valid; next = row_it.Next() {
					row := next.Value.RowAndCell().row
					if row.wrap_continuation {
						wrapped_ += 1
					}
				}

				wrapped = wrapped_
			}

			// Zig: `self.rows - active_pt.active.y - 1` in u32 traps on
			// underflow.
			assert(active_pt.Value.Active.y < uint32(self.rows))
			current := uint(uint32(self.rows) - active_pt.Value.Active.y - 1)

			// Zig: `-|=` (saturating).
			req_rows := c.remaining_rows
			req_rows = pageListSatSub(req_rows, pageListSatSub(wrapped, c.wrapped_rows))
			req_rows = pageListSatSub(req_rows, current)

			for req_rows > 0 {
				if _, err := self.Grow(); err != nil {
					return err
				}
				req_rows -= 1
			}
		}
	}

	return nil
}

// pageListSatSub is NOT a Zig declaration: Zig's saturating `a -| b` on
// usize.
func pageListSatSub(a uint, b uint) uint {
	if a > b {
		return a - b
	}
	return 0
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
func pageListReflowCursorInit(node *PageListNode) pageListReflowCursor {
	page := &node.data
	rows := page.rows
	return pageListReflowCursor{
		x:            0,
		y:            0,
		pending_wrap: false,
		node:         node,
		page:         page,
		page_row:     &rows[0],
		page_cell:    &page.rowCells(&rows[0])[0],
		new_rows:     0,

		// Initially whatever size our input node is.
		total_rows: uint(node.data.size.rows),
	}
}

// reflowRow is PageList.ReflowCursor.reflowRow: reflow the provided row in
// to this cursor.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *pageListReflowCursor) reflowRow(list *PageList, row PageListPin) error {
	src_page := &row.node.data
	src_row := row.RowAndCell().row
	src_y := row.y
	cells := src_page.rowCells(src_row)[0:src_page.size.cols]

	// Calculate the columns in this row. First up we trim non-semantic
	// rightmost blanks.
	cols_len := src_page.size.cols
	if !src_row.wrap {
		for cols_len > 0 {
			if !cells[cols_len-1].IsEmpty() {
				break
			}
			cols_len -= 1
		}

		// If the row has a semantic prompt then the blank row is meaningful
		// so we just consider pretend the first cell of the row isn't empty.
		if cols_len == 0 && src_row.semantic_prompt != PageRowSemanticPromptNone {
			cols_len = 1
		}
	}

	// Handle tracked pin adjustments.
	{
		pin_keys := list.tracked_pins.Keys()
		for _, p := range pin_keys {
			if &p.node.data != src_page ||
				p.y != src_y {
				continue
			}

			// If this pin is in the blanks on the right and past the end
			// of the dst col width then we move it to the end of the dst
			// col width instead.
			if p.x >= cols_len {
				// Zig: u16 `-` traps on underflow.
				assert(self.x < self.page.size.cols)
				p.x = min(
					p.x,
					self.page.size.cols-1-self.x,
				)
			}

			// We increase our col len to at least include this pin.
			// This ensures that blank rows with pins are processed,
			// so that the pins can be properly remapped.
			assert(p.x < math.MaxUint16) // Zig: u16 `+` traps on overflow.
			cols_len = max(cols_len, p.x+1)
		}
	}

	// Defer processing of blank rows so that blank rows
	// at the end of the page list are never written.
	if cols_len == 0 {
		// If this blank row was a wrap continuation somehow
		// then we won't need to write it since it should be
		// a part of the previously written row.
		if !src_row.wrap_continuation {
			self.new_rows += 1
		}
		return nil
	}

	// Inherit increased styles or grapheme bytes from the src page
	// we're reflowing from for new pages.
	cap, aerr := src_page.capacity.Adjust(
		PageCapacityAdjustment{cols: Some(self.page.size.cols)},
	)
	if aerr != nil {
		// Zig: comptime assert(@TypeOf(err) == error{OutOfMemory}).

		cap = src_page.capacity
		cap.cols = self.page.size.cols
		// We're already a non-standard page. We don't want to
		// inherit a massive set of rows, so cap it at our std size.
		cap.rows = min(src_page.size.rows, PageStdCapacity.rows)
	}

	// Our row isn't blank, write any new rows we deferred.
	for self.new_rows > 0 {
		if err := self.cursorScrollOrNewPage(list, cap); err != nil {
			return err
		}
		self.new_rows -= 1
	}

	self.copyRowMetadata(src_row)

	var x uint = 0
	for x < uint(cols_len) {
		if self.pending_wrap {
			self.page_row.wrap = true
			if err := self.cursorScrollOrNewPage(list, cap); err != nil {
				return err
			}
			self.copyRowMetadata(src_row)
			self.page_row.wrap_continuation = true
		}

		// Move any tracked pins from the source.
		{
			pin_keys := list.tracked_pins.Keys()
			for _, p := range pin_keys {
				if &p.node.data != src_page ||
					p.y != src_y ||
					uint(p.x) != x {
					continue
				}

				p.node = self.node
				p.x = self.x
				p.y = self.y
			}
		}

		if result, err := self.writeCell(
			list,
			&cells[x],
			src_page,
		); err == nil {
			switch result {
			// Wrote the cell, move to the next.
			case pageListReflowCursorWriteCellResultSuccess:
				x += 1

			// Wrote the cell but request to skip the next so skip it.
			// This is used for things like spacers.
			case pageListReflowCursorWriteCellResultSkipNext:
				// Remap any tracked pins at the skipped position (x+1)
				// since we won't process that cell in the loop.
				pin_keys := list.tracked_pins.Keys()
				for _, p := range pin_keys {
					if &p.node.data != src_page ||
						p.y != src_y ||
						uint(p.x) != x+1 {
						continue
					}

					p.node = self.node
					p.x = self.x
					p.y = self.y
				}

				x += 2

			// Didn't write the cell, repeat writing this same cell.
			case pageListReflowCursorWriteCellResultRepeat:
			}
		} else {
			switch err {
			// System out of memory, we can't fix this.
			case ErrOutOfMemory:
				return ErrOutOfMemory

			// We reached the capacity of a single page and can't
			// add any more of some type of managed memory. When this
			// happens we split out the current row we're working on
			// into a new page and continue from there.
			case ErrOutOfSpace:
				if self.y == 0 {
					// If we're already on the first-row, we can't split
					// any further, so we just ignore bad cells and take
					// corrupted (but valid) cell contents.
					warn("page_list", "reflowRow OutOfSpace on first row, discarding cell managed memory")
					x += 1
					self.cursorForward()
				} else {
					// Move our last row to a new page.
					if err := self.moveLastRowToNewPage(list, cap); err != nil {
						return err
					}

					// Do NOT increment x so that we retry writing
					// the same existing cell.
				}

			default:
				panic("unreachable")
			}
		}
	}

	// If the source row isn't wrapped then we should scroll afterwards.
	if !src_row.wrap {
		self.new_rows += 1
	}

	return nil
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
func (self pageListReflowCursorWriteCellResult) String() string {
	return pageListReflowCursorWriteCellResultNames[self]
}

// pageListReflowCursorWriteCellResultNames are the tag names of
// pageListReflowCursorWriteCellResult, indexed by value (Zig's @tagName).
var pageListReflowCursorWriteCellResultNames = [...]string{"success", "repeat", "skip_next"}

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
	// Initialize self.page_cell with basic, unmanaged memory contents.
	{
		// This must not fail because we want to make sure we atomically
		// setup our page cell to be valid.
		//
		// errdefer comptime unreachable;

		// Copy cell contents.
		switch cell.content_tag {
		case PageCellContentTagCodepoint,
			PageCellContentTagCodepointGrapheme:
			switch cell.wide {
			case PageCellWideNarrow:
				*self.page_cell = *cell

			case PageCellWideWide:
				if self.page.size.cols > 1 {
					if self.x == self.page.size.cols-1 {
						// If there's a wide character in the last column of
						// the reflowed page then we need to insert a spacer
						// head and wrap before handling it.
						*self.page_cell = PageCell{
							content_tag: PageCellContentTagCodepoint,
							content:     PageCellContent{codepoint: 0},
							wide:        PageCellWideSpacerHead,
						}

						// Move to the next row (this sets pending wrap
						// which will cause us to wrap on the next
						// iteration).
						self.cursorForward()

						// Decrement the source position so that when we
						// loop we'll process this source cell again,
						// since we can't copy it into a spacer head.
						return pageListReflowCursorWriteCellResultRepeat, nil
					} else {
						*self.page_cell = *cell
					}
				} else {
					// Edge case, when resizing to 1 column, wide
					// characters are just destroyed and replaced
					// with empty narrow cells.
					self.page_cell.content.codepoint = 0
					self.page_cell.wide = PageCellWideNarrow
					self.cursorForward()

					// Skip spacer tail so it doesn't cause a wrap.
					return pageListReflowCursorWriteCellResultSkipNext, nil
				}

			case PageCellWideSpacerTail:
				if self.page.size.cols > 1 {
					*self.page_cell = *cell
				} else {
					// Edge case, when resizing to 1 column, wide
					// characters are just destroyed and replaced
					// with empty narrow cells, so we should just
					// discard any spacer tails.
					return pageListReflowCursorWriteCellResultSuccess, nil
				}

			case PageCellWideSpacerHead:
				// Spacer heads should be ignored. If we need a
				// spacer head in our reflowed page, it is added
				// when processing the wide cell it belongs to.
				return pageListReflowCursorWriteCellResultSuccess, nil

			default:
				panic("unreachable")
			}

		case PageCellContentTagBgColorPalette,
			PageCellContentTagBgColorRgb:
			// These are guaranteed to have no style or grapheme
			// data associated with them so we can fast path them.
			*self.page_cell = *cell
			self.cursorForward()
			return pageListReflowCursorWriteCellResultSuccess, nil

		default:
			panic("unreachable")
		}

		// These will create issues by trying to clone managed memory that
		// isn't set if the current dst row needs to be moved to a new page.
		// They'll be fixed once we do properly copy the relevant memory.
		self.page_cell.content_tag = PageCellContentTagCodepoint
		self.page_cell.hyperlink = false
		self.page_cell.style_id = StyleDefaultId

		// Zig: the `if (comptime build_options.kitty_graphics)` block that
		// copies the Kitty virtual placeholder status is compiled out (see
		// the file header).
	}

	// std.log.warn("\nsrc_y={} src_x={} dst_y={} dst_x={} dst_cols={} cp={X} wide={} page_cell_wide={}", .{
	//     src_y,
	//     x,
	//     self.y,
	//     self.x,
	//     self.page.size.cols,
	//     cell.content.codepoint,
	//     cell.wide,
	//     self.page_cell.wide,
	// });

	// From this point on we're moving on to failable, managed memory.
	// If we reach an error, we do the minimal cleanup necessary to
	// not leave dangling memory but otherwise we gracefully degrade
	// into some functional but not strictly correct cell.

	// Copy grapheme data.
	if cell.content_tag == PageCellContentTagCodepointGrapheme {
		// Copy the graphemes
		cps_ := src_page.LookupGrapheme(cell)
		assert(cps_.Valid) // .?
		cps := cps_.Value

		// If our page can't support an additional cell
		// with graphemes then we increase capacity.
		if self.page.GraphemeCount() >= self.page.GraphemeCapacity() {
			if err := self.increaseCapacity(
				list,
				Some(PageListIncreaseCapacityGraphemeBytes),
			); err != nil {
				return 0, err
			}
		}

		// Attempt to allocate the space that would be required
		// for these graphemes, and if it's not available, then
		// increase capacity. Keep trying until we succeed.
		for {
			if slice, err := self.page.grapheme_alloc.Alloc(
				uint(len(cps)),
			); err == nil {
				self.page.grapheme_alloc.Free(
					slice,
				)
				break
			} else {
				// Grow our capacity until we can fit the extra bytes.
				if err := self.increaseCapacity(list, Some(PageListIncreaseCapacityGraphemeBytes)); err != nil {
					return 0, err
				}
			}
		}

		if err := self.page.SetGraphemes(
			self.page_row,
			self.page_cell,
			cps,
		); err != nil {
			// This shouldn't fail since we made sure we have space
			// above. There is no reasonable behavior we can take here
			// so we have a warn level log. This is ALMOST non-recoverable,
			// though we choose to recover by corrupting the cell
			// to a non-grapheme codepoint.
			//
			// Zig: log.err
			warn("page_list", "setGraphemes failed after capacity increase err=error.%v", err)
			// Zig: `if (comptime std.debug.runtime_safety)`; the
			// reference is a safe build. Force a crash with safe builds.
			// (The unsafe-build fallback, which throws away the grapheme
			// data, is not reached.)
			panic("unreachable")
		}
	}

	// Copy hyperlink data.
	//
	// Zig: the block is labeled `hyperlink:`; every `break :hyperlink`
	// follows a safe-build `unreachable`, so none is reached.
	if cell.hyperlink {
		src_id_ := src_page.LookupHyperlink(cell)
		assert(src_id_.Valid) // .?
		src_id := src_id_.Value
		src_link := src_page.hyperlink_set.Get(src_id)

		// If our page can't support an additional cell
		// with a hyperlink then we increase capacity.
		if self.page.HyperlinkCount() >= self.page.HyperlinkCapacity() {
			if err := self.increaseCapacity(list, Some(PageListIncreaseCapacityHyperlinkBytes)); err != nil {
				return 0, err
			}
		}

		// Ensure that the string alloc has sufficient capacity
		// to dupe the link (and the ID if it's not implicit).
		additional_required_string_capacity := uint(len(src_link.uri))
		switch src_link.id.Tag {
		case HyperlinkPageEntryIdExplicit:
			additional_required_string_capacity += uint(len(src_link.id.Explicit))
		case HyperlinkPageEntryIdImplicit:
			additional_required_string_capacity += 0
		}
		// Keep trying until we have enough capacity.
		for {
			if slice, err := self.page.string_alloc.Alloc(
				additional_required_string_capacity,
			); err == nil {
				// We have enough capacity, free the test alloc.
				self.page.string_alloc.Free(
					slice,
				)
				break
			} else {
				// Grow our capacity until we can fit the extra bytes.
				if err := self.increaseCapacity(
					list,
					Some(PageListIncreaseCapacityStringBytes),
				); err != nil {
					return 0, err
				}
			}
		}

		dst_link, err := src_link.Dupe(
			src_page,
			self.page,
		)
		if err != nil {
			// This shouldn't fail since we did a capacity
			// check above.
			//
			// Zig: log.err
			warn("page_list", "link dupe failed with capacity check err=error.%v", err)
			// Zig: `if (comptime std.debug.runtime_safety)`; force a crash
			// with safe builds (the `break :hyperlink` is not reached).
			panic("unreachable")
		}

		dst_id_, err := self.page.hyperlink_set.AddWithIdContext(
			dst_link,
			src_id,
			HyperlinkSetContext{page: self.page},
		)
		if err != nil {
			// Zig: `catch |err| id: { ... }`.

			// Always free our original link in case the increaseCap
			// call fails so we aren't leaking memory.
			dst_link.Free(self.page)

			// If the add failed then either the set needs to grow
			// or it needs to be rehashed. Either one of those can
			// be accomplished by increasing capacity, either with
			// no actual change or with an increased hyperlink cap.
			var adjustment Optional[PageListIncreaseCapacity]
			switch err {
			case ErrOutOfMemory:
				adjustment = Some(PageListIncreaseCapacityHyperlinkBytes)
			case ErrNeedsRehash:
				adjustment = None[PageListIncreaseCapacity]()
			default:
				panic("unreachable")
			}
			if err := self.increaseCapacity(list, adjustment); err != nil {
				return 0, err
			}

			// We need to recreate the link into the new page.
			dst_link2, err2 := src_link.Dupe(
				src_page,
				self.page,
			)
			if err2 != nil {
				// This shouldn't fail since we did a capacity
				// check above.
				//
				// Zig: log.err
				warn("page_list", "link dupe failed with capacity check err=error.%v", err2)
				// Force a crash with safe builds.
				panic("unreachable")
			}

			// We assume this one will succeed. We dupe the link
			// again, and don't have to worry about the other one
			// because increasing the capacity naturally clears up
			// any managed memory not associated with a cell yet.
			id2, err2 := self.page.hyperlink_set.AddWithIdContext(
				dst_link2,
				src_id,
				HyperlinkSetContext{page: self.page},
			)
			if err2 != nil {
				// This shouldn't happen since we increased capacity
				// above so we handle it like the other similar
				// cases and log it, crash in safe builds, and
				// remove the hyperlink in unsafe builds.
				//
				// Zig: log.err
				warn(
					"page_list",
					"addWithIdContext failed after capacity increase err=error.%v",
					err2,
				)
				// Force a crash with safe builds.
				panic("unreachable")
			}
			dst_id_ = id2
		}
		// Zig: `orelse src_id`.
		dst_id := src_id
		if dst_id_.Valid {
			dst_id = dst_id_.Value
		}

		// We expect this to succeed due to the hyperlinkCapacity
		// check we did before. If it doesn't succeed let's
		// log it, crash (in safe builds), and clear our state.
		if err := self.page.SetHyperlink(
			self.page_row,
			self.page_cell,
			dst_id,
		); err != nil {
			// Zig: log.err
			warn(
				"page_list",
				"setHyperlink failed after capacity increase err=error.%v",
				err,
			)
			// Force a crash with safe builds. (The unsafe-build fallback,
			// which throws away the hyperlink data, is not reached.)
			panic("unreachable")
		}
	}

	// Copy style data.
	//
	// Zig: the block is labeled `style:`; its `break :style` follows a
	// safe-build `unreachable` and is not reached.
	if cell.HasStyling() {
		style := *src_page.styles.Get(
			cell.style_id,
		)

		id_, err := self.page.styles.AddWithId(
			style,
			cell.style_id,
		)
		if err != nil {
			// Zig: `catch |err| id: { ... }`.

			// If the add failed then either the set needs to grow
			// or it needs to be rehashed. Either one of those can
			// be accomplished by increasing capacity, either with
			// no actual change or with an increased style cap.
			var adjustment Optional[PageListIncreaseCapacity]
			switch err {
			case ErrOutOfMemory:
				adjustment = Some(PageListIncreaseCapacityStyles)
			case ErrNeedsRehash:
				adjustment = None[PageListIncreaseCapacity]()
			default:
				panic("unreachable")
			}
			if err := self.increaseCapacity(list, adjustment); err != nil {
				return 0, err
			}

			// We assume this one will succeed.
			id2, err2 := self.page.styles.AddWithId(
				style,
				cell.style_id,
			)
			if err2 != nil {
				// Should not fail since we just modified capacity
				// above. Log it, crash in safe builds, clear style
				// in unsafe builds.
				//
				// Zig: log.err
				warn(
					"page_list",
					"addWithId failed after capacity increase err=error.%v",
					err2,
				)
				// Force a crash with safe builds.
				panic("unreachable")
			}
			id_ = id2
		}
		// Zig: `orelse cell.style_id`.
		id := cell.style_id
		if id_.Valid {
			id = id_.Value
		}

		self.page_row.styled = true
		self.page_cell.style_id = id
	}

	self.cursorForward()
	return pageListReflowCursorWriteCellResultSuccess, nil
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
	assert(self.page.size.rows > 0) // Zig: u16 `- 1` traps
	assert(self.y == self.page.size.rows-1)
	assert(!self.pending_wrap)

	old_node := self.node
	old_page := self.page
	old_row := self.page_row
	old_x := self.x

	// Our total row count never changes, because we're removing one
	// row from the last page and moving it into a new page.
	old_total_rows := self.total_rows
	defer func() { self.total_rows = old_total_rows }()

	if err := self.cursorNewPage(list, cap); err != nil {
		return err
	}
	assert(self.node != old_node)
	assert(self.y == 0)

	// We have no cleanup for our old state from here on out. No failures!
	//
	// errdefer comptime unreachable;

	// Restore the x position of the cursor.
	self.cursorAbsolute(old_x, 0)

	// Copy our old data. This should NOT fail because we have the
	// capacity of the old page which already fits the data we requested.
	if err := self.page.CloneRowFrom(
		old_page,
		self.page_row,
		old_row,
	); err != nil {
		// Zig: log.err
		warn(
			"page_list",
			"error cloning single row for moveLastRowToNewPage err=error.%v",
			err,
		)
		panic("unexpected copy row failure")
	}

	// Move any tracked pins from that last row into this new node.
	{
		pin_keys := list.tracked_pins.Keys()
		for _, p := range pin_keys {
			if &p.node.data != old_page ||
				p.y != old_page.size.rows-1 {
				continue
			}

			p.node = self.node
			p.y = self.y
			// p.x remains the same since we're copying the row as-is
		}
	}

	// Clear the row from the old page and truncate it.
	old_page.ClearCells(old_row, 0, uint(old_page.size.cols))
	old_page.size.rows -= 1

	// If that was the last row in that page
	// then we should remove it from the list.
	if old_page.size.rows == 0 {
		list.pages.Remove(old_node)
		list.destroyNode(old_node)
	}

	return nil
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
	old_x := self.x
	old_y := self.y
	old_total_rows := self.total_rows

	node, err := func() (*PageListNode, error) {
		// Pause integrity checks because the total row count won't
		// be correct during a reflow.
		list.PauseIntegrityChecks(true)
		defer list.PauseIntegrityChecks(false)
		return list.IncreaseCapacity(
			self.node,
			adjustment,
		)
	}()
	if err != nil {
		return err
	}
	// We must not fail after this, we've modified our self.node
	// and we need to fix it up.
	//
	// errdefer comptime unreachable;

	*self = pageListReflowCursorInit(node)
	self.cursorAbsolute(old_x, old_y)
	self.total_rows = old_total_rows
	return nil
}

// bottom is PageList.ReflowCursor.bottom.
//
// True if this cursor is at the bottom of the page by capacity,
// i.e. we can't scroll anymore.
//
// Zig: self is *const ReflowCursor.
func (self *pageListReflowCursor) bottom() bool {
	assert(self.page.capacity.rows > 0) // Zig: u16 `- 1` traps
	return self.y == self.page.capacity.rows-1
}

// cursorForward is PageList.ReflowCursor.cursorForward.
func (self *pageListReflowCursor) cursorForward() {
	assert(self.page.size.cols > 0) // Zig: u16 `- 1` traps
	if self.x == self.page.size.cols-1 {
		self.pending_wrap = true
	} else {
		// Zig: `cell + 1` on a many-item pointer.
		cell := self.page.cellIndex(self.page_cell)
		self.page_cell = &self.page.cells[cell+1]
		self.x += 1
	}
}

// cursorScroll is PageList.ReflowCursor.cursorScroll.
//
// Create a new row and move the cursor down.
//
// Asserts that the cursor is on the bottom row of the
// page and that there is capacity to add a new one.
func (self *pageListReflowCursor) cursorScroll() {
	// Scrolling requires that we're on the bottom of our page.
	// We also assert that we have capacity because reflow always
	// works within the capacity of the page.
	assert(self.page.size.rows > 0) // Zig: u16 `- 1` traps
	assert(self.y == self.page.size.rows-1)
	assert(self.page.size.rows < self.page.capacity.rows)

	// Increase our page size
	self.page.size.rows += 1

	// With the increased page size, safely move down a row.
	//
	// Zig: `rows + 1` on a many-item pointer.
	rows := self.page.rowIndex(self.page_row)
	row := &self.page.rows[rows+1]
	self.page_row = row
	self.page_cell = &self.page.rowCells(row)[0]
	self.pending_wrap = false
	self.x = 0
	self.y += 1
}

// cursorNewPage is PageList.ReflowCursor.cursorNewPage.
//
// Create a new page in the provided list with the provided
// capacity and one row and move the cursor in to it at 0,0
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *pageListReflowCursor) cursorNewPage(list *PageList, cap PageCapacity) error {
	// Remember our new row count so we can restore it
	// after reinitializing our cursor on the new page.
	new_rows := self.new_rows

	node, err := list.createPage(cap)
	if err != nil {
		return err
	}
	// errdefer comptime unreachable;
	node.data.size.rows = 1
	list.pages.InsertAfter(self.node, node)

	*self = pageListReflowCursorInit(node)
	self.new_rows = new_rows
	return nil
}

// cursorScrollOrNewPage is PageList.ReflowCursor.cursorScrollOrNewPage.
//
// Performs `cursorScroll` or `cursorNewPage` as necessary
// depending on if the cursor is currently at the bottom.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *pageListReflowCursor) cursorScrollOrNewPage(list *PageList, cap PageCapacity) error {
	// The functions below may overwrite self so we need to cache
	// our total rows. We add one because no matter what when this
	// returns we'll have one more row added.
	new_total_rows := self.total_rows + 1
	defer func() { self.total_rows = new_total_rows }()

	if self.bottom() {
		if err := self.cursorNewPage(list, cap); err != nil {
			return err
		}
	} else {
		self.cursorScroll()
	}
	return nil
}

// cursorAbsolute is PageList.ReflowCursor.cursorAbsolute.
func (self *pageListReflowCursor) cursorAbsolute(x SizeCellCountInt, y SizeCellCountInt) {
	assert(x < self.page.size.cols)
	assert(y < self.page.size.rows)

	// Zig: `rows - n` / `rows + n` on a many-item pointer.
	var row *PageRow
	switch {
	case y == self.y:
		row = self.page_row
	case y < self.y:
		row = &self.page.rows[self.page.rowIndex(self.page_row)-uint(self.y-y)]
	default:
		row = &self.page.rows[self.page.rowIndex(self.page_row)+uint(y-self.y)]
	}
	self.page_row = row
	self.page_cell = &self.page.rowCells(row)[x]
	self.pending_wrap = false
	self.x = x
	self.y = y
}

// countTrailingEmptyCells is PageList.ReflowCursor.countTrailingEmptyCells.
//
// Zig: self is *const ReflowCursor; returns usize, a count of cells.
func (self *pageListReflowCursor) countTrailingEmptyCells() uint {
	// If the row is wrapped, all empty cells are meaningful.
	if self.page_row.wrap {
		return 0
	}

	// Zig: `cells` is the many-item pointer at page_cell.
	cells := self.page.cells[self.page.cellIndex(self.page_cell):]
	assert(self.x <= self.page.size.cols) // Zig: u16 `-` traps
	length := uint(self.page.size.cols - self.x)
	for i := range length {
		rev_i := length - i - 1
		if !cells[rev_i].IsEmpty() {
			return i
		}
	}

	// If the row has a semantic prompt then the blank row is meaningful
	// so we always return all but one so that the row is drawn.
	if self.page_row.semantic_prompt != PageRowSemanticPromptNone {
		assert(length > 0) // Zig: usize `- 1` traps
		return length - 1
	}

	return length
}

// copyRowMetadata is PageList.ReflowCursor.copyRowMetadata.
//
// Zig: other is *const Row.
func (self *pageListReflowCursor) copyRowMetadata(other *PageRow) {
	self.page_row.semantic_prompt = other.semantic_prompt
}

// resizeWithoutReflow is PageList.resizeWithoutReflow.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *PageList) resizeWithoutReflow(opts PageListResize) (err error) {
	// We only set the new min_max_size if we're not reflowing. If we are
	// reflowing, then resize handles this for us.
	old_min_max_size := self.min_max_size
	if !opts.reflow {
		cols := self.cols
		if opts.cols.Valid {
			cols = opts.cols.Value
		}
		rows := self.rows
		if opts.rows.Valid {
			rows = opts.rows.Value
		}
		self.min_max_size = pageListMinMaxSize(cols, rows)
	} else {
		self.min_max_size = old_min_max_size
	}
	defer func() {
		// errdefer
		if err != nil {
			self.min_max_size = old_min_max_size
		}
	}()

	// Important! We have to do cols first because cols may cause us to
	// destroy pages if we're increasing cols which will free up page_size
	// so that when we call grow() in the row mods, we won't prune.
	if opts.cols.Valid {
		cols := opts.cols.Value
		// Zig: the block's `defer assert(...)` runs when the block ends,
		// also on an error: the block is a closure.
		if err := func() error {
			// Any column change without reflow should not result in row counts
			// changing.
			old_total_rows := self.total_rows
			defer func() { assert(self.total_rows == old_total_rows) }()

			switch {
			case cols == self.cols:

			// Making our columns smaller. We always have space for this
			// in existing pages so we need to go through the pages,
			// resize the columns, and clear any cells that are beyond
			// the new size.
			case cols < self.cols:
				it := self.PageIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
				for chunk := it.Next(); chunk.Valid; chunk = it.Next() {
					page := &chunk.Value.node.data
					func() {
						defer page.AssertIntegrity()
						rows := page.rows
						for i := range uint(page.size.rows) {
							row := &rows[i]
							page.ClearCells(row, uint(cols), uint(self.cols))
						}

						page.size.cols = cols
					}()
				}

				// Update all our tracked pins. If they have an X
				// beyond the edge, clamp it.
				pin_keys := self.tracked_pins.Keys()
				for _, p := range pin_keys {
					if p.x >= cols {
						p.x = cols - 1
					}
				}

				self.cols = cols

			// Make our columns larger. This is a bit more complicated because
			// pages may not have the capacity for this. If they don't have
			// the capacity we need to allocate a new page and copy the data.
			default: // .gt
				// See the comment in the while loop when setting self.cols
				old_cols := self.cols

				it := self.PageIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
				for chunk := it.Next(); chunk.Valid; chunk = it.Next() {
					// We need to restore our old cols after we resize because
					// we have an assertion on this and we want to be able to
					// call this method multiple times.
					self.cols = old_cols
					if err := self.resizeWithoutReflowGrowCols(cols, chunk.Value); err != nil {
						return err
					}
				}

				self.cols = cols
			}
			return nil
		}(); err != nil {
			return err
		}
	}

	if opts.rows.Valid {
		rows := opts.rows.Value
		switch {
		case rows == self.rows:

		// Making rows smaller, we simply change our rows value. Changing
		// the row size doesn't affect anything else since max size and
		// so on are all byte-based.
		case rows < self.rows:
			// If our rows are shrinking, we prefer to trim trailing
			// blank lines from the active area instead of creating
			// history if we can.
			//
			// This matches macOS Terminal.app behavior. I chose to match that
			// behavior because it seemed fine in an ocean of differing behavior
			// between terminal apps. I'm completely open to changing it as long
			// as resize behavior isn't regressed in a user-hostile way.
			trimmed := self.trimTrailingBlankRows(self.rows - rows)

			// Account for our trimmed rows in the total row cache
			assert(self.total_rows >= uint(trimmed)) // Zig: usize `-=` traps
			self.total_rows -= uint(trimmed)

			// If we didn't trim enough, just modify our row count and this
			// will create additional history.
			self.rows = rows

		// Making rows larger we adjust our row count, and then grow
		// to the row count.
		default: // .gt (Zig: the labeled block `gt:`; `break :gt` is
			// Go's `break` out of the switch)
			// If our rows increased and our cursor is NOT at the bottom,
			// we want to try to preserve the y value of the old cursor.
			// In other words, we don't want to "pull down" scrollback.
			// This is purely a UX feature.
			//
			// Zig: `break :cursor` when the cursor is at the bottom.
			if opts.cursor.Valid && opts.cursor.Value.y < self.rows-1 {
				// Cursor is not at the bottom, so we just grow our
				// rows and we're done. Cursor does NOT change for this
				// since we're not pulling down scrollback.
				delta := rows - self.rows
				self.rows = rows
				for range delta {
					if _, err := self.Grow(); err != nil {
						return err
					}
				}
				break // break :gt
			}

			// This must be set BEFORE any calls to grow() so that
			// grow() doesn't prune pages that we need for the active
			// area.
			self.rows = rows

			// Cursor is at the bottom or we don't care about cursors.
			// In this case, if we have enough rows in our pages, we
			// just update our rows and we're done. This effectively
			// "pulls down" scrollback.
			//
			// If we don't have enough scrollback, we add the difference,
			// to the active area.
			var count uint = 0
			page := self.pages.first
			for ; page != nil; page = page.next {
				p := page
				count += uint(p.data.size.rows)
				if count >= uint(rows) {
					break
				}
			}
			if page == nil {
				// Zig: the `else` of the while, reached without a break.
				assert(count < uint(rows))
				for range uint(rows) - count {
					if _, err := self.Grow(); err != nil {
						return err
					}
				}
			}

			// Make sure that the viewport pin isn't below the active
			// area, since that will lead to all sorts of problems.
			switch self.viewport.Tag {
			case PageListViewportPin:
				if self.pinIsActive(*self.viewport_pin) {
					self.viewport = PageListViewport{Tag: PageListViewportActive}
				}
			case PageListViewportActive, PageListViewportTop:
			}
		}

		if pageSlowRuntimeSafety {
			// We never have less rows than our active screen has.
			assert(self.totalRows() >= uint(self.rows))
		}
	}

	return nil
}

// resizeWithoutReflowGrowCols is PageList.resizeWithoutReflowGrowCols.
//
// Zig: fn resizeWithoutReflowGrowCols(self: *PageList,
// cols: size.CellCountInt, chunk: PageIterator.Chunk) Allocator.Error!void.
func (self *PageList) resizeWithoutReflowGrowCols(
	cols SizeCellCountInt,
	chunk PageListPageIteratorChunk,
) (err error) {
	assert(cols > self.cols)
	page := &chunk.node.data

	// Update our col count
	old_cols := self.cols
	self.cols = cols
	defer func() {
		// errdefer
		if err != nil {
			self.cols = old_cols
		}
	}()

	// Unlikely fast path: we have capacity in the page. This
	// is only true if we resized to less cols earlier.
	//
	// Zig: the labeled block `fast:`; `break :fast` leaves it.
	if page.capacity.cols >= cols {
		fast := true
		// If any row has a spacer head at the old last column, it will
		// be invalid at the new (wider) size. Fall through to the slow
		// path which handles spacer heads correctly via cloneRowFrom.
		rows := page.rows[0:page.size.rows]
		for i := range rows {
			row := &rows[i]
			cells := page.GetCells(row)
			if cells[old_cols-1].wide == PageCellWideSpacerHead {
				fast = false // break :fast
				break
			}
		}

		if fast {
			page.size.cols = cols
			return nil
		}
	}

	// Likely slow path: we don't have capacity, so we need
	// to allocate a page, and copy the old data into it.

	// Try to fit our new column size into our existing page capacity.
	// If that doesn't work then use a non-standard page with the
	// given columns.
	cap, aerr := page.capacity.Adjust(
		PageCapacityAdjustment{cols: Some(cols)},
	)
	if aerr != nil {
		// Zig: comptime assert(@TypeOf(err) == error{OutOfMemory}).

		// We verify all maxed out page layouts don't overflow,
		cap = page.capacity
		cap.cols = cols

		// We're growing columns so we can only get less rows so use
		// the lesser of our capacity and size so we minimize wasted
		// rows.
		cap.rows = min(page.size.rows, cap.rows)
	}

	// On error, we need to undo all the pages we've added.
	prev := chunk.node.prev
	defer func() {
		// errdefer
		if err == nil {
			return
		}
		current := chunk.node.prev
		for current != nil {
			p := current
			if current == prev {
				break
			}
			current = p.prev
			self.pages.Remove(p)
			self.destroyNode(p)
		}
	}()

	// Keeps track of all our copied rows. Assertions at the end is that
	// we copied exactly our page size.
	var copied SizeCellCountInt = 0

	// This function has an unfortunate side effect in that it causes memory
	// fragmentation on rows if the columns are increasing in a way that
	// shrinks capacity rows. If we have pages that don't divide evenly then
	// we end up creating a final page that is not using its full capacity.
	// If this chunk isn't the last chunk in the page list, then we've created
	// a page where we'll never reclaim that capacity. This makes our max size
	// calculation incorrect since we'll throw away data even though we have
	// excess capacity. To avoid this, we try to fill our previous page
	// first if it has capacity.
	//
	// This can fail for many reasons (can't fit styles/graphemes, etc.) so
	// if it fails then we give up and drop back into creating new pages.
	if prev_node := prev; prev_node != nil {
		// Zig: the labeled block `prev:`; `break :prev` is a return from
		// the closure.
		func() {
			prev_page := &prev_node.data

			// We only want scenarios where we have excess capacity.
			if prev_page.size.rows >= prev_page.capacity.rows {
				return
			}

			// We can copy as much as we can to fill the capacity or our
			// current page size.
			length := min(
				prev_page.capacity.rows-prev_page.size.rows,
				page.size.rows,
			)

			src_rows := page.rows[0:length]
			// Zig: a many-item pointer from prev_page.size.rows; the loop
			// length is src_rows'.
			dst_rows := prev_page.rows[prev_page.size.rows:]
			for i := range src_rows {
				dst_row := &dst_rows[i]
				src_row := &src_rows[i]
				prev_page.size.rows += 1
				copied += 1
				if err := prev_page.CloneRowFrom(
					page,
					dst_row,
					src_row,
				); err != nil {
					// If an error happens, we undo our row copy and break out
					// into creating a new page.
					prev_page.size.rows -= 1
					copied -= 1
					return // break :prev
				}
			}

			assert(copied == length)
			assert(prev_page.size.rows <= prev_page.capacity.rows)

			// Remap any tracked pins that pointed to rows we just copied to prev.
			pin_keys := self.tracked_pins.Keys()
			for _, p := range pin_keys {
				if p.node != chunk.node || p.y >= length {
					continue
				}
				p.node = prev_node
				p.y += prev_page.size.rows - length
			}
		}()
	}

	// If we have an error, we clear the rows we just added to our prev page.
	prev_copied := copied
	defer func() {
		// errdefer
		if err == nil {
			return
		}
		if prev_copied > 0 {
			prev_page := &prev.data
			prev_size := prev_page.size.rows - prev_copied
			prev_rows := prev_page.rows[prev_size:prev_page.size.rows]
			for i := range prev_rows {
				prev_page.ClearCells(
					&prev_rows[i],
					0,
					uint(prev_page.size.cols),
				)
			}
			prev_page.size.rows = prev_size
		}
	}()

	// We delete any of the nodes we added.
	defer func() {
		// errdefer
		if err == nil {
			return
		}
		it := chunk.node.prev
		for it != nil {
			node := it
			if node == prev {
				break
			}
			it = node.prev
			self.pages.Remove(node)
			self.destroyNode(node)
		}
	}()

	// We need to loop because our col growth may force us
	// to split pages.
	for copied < page.size.rows {
		new_node, err := self.createPage(cap)
		if err != nil {
			return err
		}
		// Zig: `defer new_node.data.assertIntegrity()` at the end of each
		// iteration: the rest of the body is a closure.
		func() {
			defer new_node.data.AssertIntegrity()

			// The length we can copy into the new page is at most the number
			// of rows in our cap. But if we can finish our source page we use that.
			length := min(cap.rows, page.size.rows-copied)

			// Perform the copy
			y_start := copied
			src_rows := page.rows[y_start : copied+length]
			dst_rows := new_node.data.rows[0:length]
			for i := range dst_rows {
				dst_row := &dst_rows[i]
				src_row := &src_rows[i]
				new_node.data.size.rows += 1
				if err := new_node.data.CloneRowFrom(
					page,
					dst_row,
					src_row,
				); err == nil {
					copied += 1
				} else {
					// I don't THINK this should be possible, because while our
					// row count may diminish due to the adjustment, our
					// prior capacity should have been sufficient to hold all the
					// managed memory.
					warn(
						"page_list",
						"unexpected cloneRowFrom failure during resizeWithoutReflowGrowCols: error.%v",
						err,
					)

					// We can actually safely handle this though by exiting
					// this loop early and cutting our copy short.
					new_node.data.size.rows -= 1
					break
				}
			}
			y_end := copied

			// Insert our new page
			self.pages.InsertBefore(chunk.node, new_node)

			// Update our tracked pins that pointed to this previous page.
			pin_keys := self.tracked_pins.Keys()
			for _, p := range pin_keys {
				if p.node != chunk.node ||
					p.y < y_start ||
					p.y >= y_end {
					continue
				}
				p.node = new_node
				p.y -= y_start
			}
		}()
	}
	assert(copied == page.size.rows)

	// Our prior errdeferes are invalid after this point so ensure
	// we don't have any more errors.
	//
	// errdefer comptime unreachable;

	// Remove the old page.
	// Deallocate the old page.
	self.pages.Remove(chunk.node)
	self.destroyNode(chunk.node)

	return nil
}

// trailingBlankLines is PageList.trailingBlankLines.
//
// Returns the number of trailing blank lines, not to exceed max. Max
// is used to limit our traversal in the case of large scrollback.
//
// Zig: self is *const PageList.
func (self *PageList) trailingBlankLines(max SizeCellCountInt) SizeCellCountInt {
	var count SizeCellCountInt = 0

	// Go through our pages backwards since we're counting trailing blanks.
	for page := self.pages.last; page != nil; page = page.prev {
		length := uint(page.data.size.rows)
		rows := page.data.rows[0:length]
		for i := range length {
			rev_i := length - i - 1
			cells := page.data.rowCells(&rows[rev_i])[0:page.data.size.cols]

			// If the row has any text then we're done.
			if PageCellHasTextAny(cells) {
				return count
			}

			// Inc count, if we're beyond max then we're done.
			assert(count < math.MaxUint16) // Zig: u16 `+=` traps
			count += 1
			if count >= max {
				return count
			}
		}
	}

	return count
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
	var trimmed SizeCellCountInt = 0
	bl_pin_ := self.GetBottomRight(PointTagScreen)
	assert(bl_pin_.Valid) // .?
	bl_pin := bl_pin_.Value
	it := bl_pin.RowIterator(PageListDirectionLeftUp, None[PageListPin]())
	for row_pin_ := it.Next(); row_pin_.Valid; row_pin_ = it.Next() {
		row_pin := row_pin_.Value
		cells := row_pin.Cells(PageListPinCellSubsetAll)

		// If the row has any text then we're done.
		if PageCellHasTextAny(cells) {
			return trimmed
		}

		// If our tracked pins are in this row then we cannot trim it
		// because it implies some sort of importance. If we trimmed this
		// we'd invalidate this pin, as well.
		pin_keys := self.tracked_pins.Keys()
		for _, p := range pin_keys {
			if p.node != row_pin.node ||
				p.y != row_pin.y {
				continue
			}
			return trimmed
		}

		// No text, we can trim this row. Because it has
		// no text we can also be sure it has no styling
		// so we don't need to worry about memory.
		row_pin.node.data.size.rows -= 1
		if row_pin.node.data.size.rows == 0 {
			self.erasePage(row_pin.node)
		} else {
			row_pin.node.data.AssertIntegrity()
		}

		trimmed += 1
		if trimmed >= max {
			return trimmed
		}
	}

	return trimmed
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
func (self PageListScrollTag) String() string {
	return pageListScrollTagNames[self]
}

// pageListScrollTagNames are the tag names of PageListScrollTag, indexed by
// value (Zig's @tagName).
var pageListScrollTagNames = [...]string{"active", "top", "row", "delta_row", "delta_prompt", "pin"}

// Scroll is PageList.scroll.
//
// Scroll the viewport. This will never create new scrollback, allocate
// pages, etc. This can only be used to move the viewport within the
// previously allocated pages.
func (self *PageList) Scroll(behavior PageListScroll) {
	defer self.AssertIntegrity()

	// Special case no-scrollback mode to never allow scrolling.
	if self.explicit_max_size == 0 {
		self.viewport = PageListViewport{Tag: PageListViewportActive}
		return
	}

	switch behavior.Tag {
	case PageListScrollActive:
		self.viewport = PageListViewport{Tag: PageListViewportActive}
	case PageListScrollTop:
		self.viewport = PageListViewport{Tag: PageListViewportTop}
	case PageListScrollPin:
		p := behavior.Pin
		if self.pinIsActive(p) {
			self.viewport = PageListViewport{Tag: PageListViewportActive}
			return
		} else if self.pinIsTop(p) {
			self.viewport = PageListViewport{Tag: PageListViewportTop}
			return
		}

		*self.viewport_pin = p
		self.viewport = PageListViewport{Tag: PageListViewportPin}
		self.viewport_pin_row_offset = None[uint]() // invalidate cache
	case PageListScrollRow:
		// Zig: the labeled block `row:`; `break :row` is a return from
		// the closure (nothing follows the switch).
		n := behavior.Row
		func() {
			// If we're at the top, pin the top.
			if n == 0 {
				self.viewport = PageListViewport{Tag: PageListViewportTop}
				return
			}

			// If we're below the top of the active area, pin the active area.
			assert(self.total_rows >= uint(self.rows)) // Zig: usize `-` traps
			if n >= self.total_rows-uint(self.rows) {
				self.viewport = PageListViewport{Tag: PageListViewportActive}
				return
			}

			// See if there are any other faster paths we can take.
			switch self.viewport.Tag {
			case PageListViewportTop, PageListViewportActive:
			case PageListViewportPin:
				if v := self.viewport_pin_row_offset; v.Valid {
					// If we have a pin and we already calculated a row offset,
					// then we can efficiently calculate the delta and move
					// that much from that pin.
					var delta int
					{
						// Zig: @intCast to isize traps when it does not fit.
						assert(n <= math.MaxInt)
						n_isize := int(n)
						assert(v.Value <= math.MaxInt)
						v_isize := int(v.Value)
						delta = n_isize - v_isize
					}
					self.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: delta})
					return
				}
			}

			// We have an accurate row offset so store it to prevent
			// calculating this again.
			self.viewport_pin_row_offset = Some(n)
			self.viewport = PageListViewport{Tag: PageListViewportPin}

			// Slow path, we've just got to traverse the linked list and
			// get to our row. As a slight speedup, let's pick the traversal
			// that's likely faster based on our absolute row and total rows.
			midpoint := self.total_rows / 2
			if n < midpoint {
				// Iterate forward from the first node.
				node_it := self.pages.first
				// Zig: std.math.cast(size.CellCountInt, n) orelse { ... }.
				if n > math.MaxUint16 {
					self.viewport = PageListViewport{Tag: PageListViewportActive}
					return
				}
				rem := SizeCellCountInt(n)
				for ; node_it != nil; node_it = node_it.next {
					node := node_it
					if rem < node.data.size.rows {
						*self.viewport_pin = PageListPin{
							node: node,
							y:    rem,
						}
						return
					}

					rem -= node.data.size.rows
				}
			} else {
				// Iterate backwards from the last node.
				node_it := self.pages.last
				// Zig: std.math.cast(size.CellCountInt, self.total_rows - n)
				// orelse { ... }.
				if self.total_rows-n > math.MaxUint16 {
					self.viewport = PageListViewport{Tag: PageListViewportActive}
					return
				}
				rem := SizeCellCountInt(self.total_rows - n)
				for ; node_it != nil; node_it = node_it.prev {
					node := node_it
					if rem <= node.data.size.rows {
						*self.viewport_pin = PageListPin{
							node: node,
							y:    node.data.size.rows - rem,
						}
						return
					}

					rem -= node.data.size.rows
				}
			}

			// If we reached here, then we couldn't find the offset.
			// This feels impossible? Just clamp to active, screw it lol.
			self.viewport = PageListViewport{Tag: PageListViewportActive}
		}()
	case PageListScrollDeltaPrompt:
		self.scrollPrompt(behavior.DeltaPrompt)
	case PageListScrollDeltaRow:
		// Zig: the labeled block `delta_row:`; `break :delta_row` is a
		// return from the closure (nothing follows the switch).
		n := behavior.DeltaRow
		func() {
			switch self.viewport.Tag {
			// If we're at the top and we're scrolling backwards,
			// we don't have to do anything, because there's nowhere to go.
			case PageListViewportTop:
				if n <= 0 {
					return
				}

			// If we're at active and we're scrolling forwards, we don't
			// have to do anything because it'll result in staying in
			// the active.
			case PageListViewportActive:
				if n >= 0 {
					return
				}

			// If we're already a pin type, then we can fast-path our
			// delta by simply moving the pin. This has the added benefit
			// that we can update our row offset cache efficiently, too.
			case PageListViewportPin:
				switch {
				case n == 0:
					return

				case n < 0:
					assert(n != math.MinInt) // Zig: `-n` traps on overflow
					ov := self.viewport_pin.UpOverflow(uint(-n))
					switch ov.Tag {
					case PageListPinOverflowOffset:
						new_pin := ov.Offset
						*self.viewport_pin = new_pin
						if self.viewport_pin_row_offset.Valid {
							v := &self.viewport_pin_row_offset.Value
							assert(*v >= uint(-n)) // Zig: usize `-=` traps
							*v -= uint(-n)
						}
						return

					// If we overflow up we're at the top.
					case PageListPinOverflowOverflow:
						self.viewport = PageListViewport{Tag: PageListViewportTop}
						return
					}

				default: // .gt
					ov := self.viewport_pin.DownOverflow(uint(n))
					switch ov.Tag {
					// If we offset its a valid pin but we still have to
					// check if we're in the active area.
					case PageListPinOverflowOffset:
						new_pin := ov.Offset
						if self.pinIsActive(new_pin) {
							self.viewport = PageListViewport{Tag: PageListViewportActive}
						} else {
							*self.viewport_pin = new_pin
							if self.viewport_pin_row_offset.Valid {
								v := &self.viewport_pin_row_offset.Value
								*v += uint(n)
							}
						}
						return

					// If we overflow down we're at active.
					case PageListPinOverflowOverflow:
						self.viewport = PageListViewport{Tag: PageListViewportActive}
						return
					}
				}
			}

			// Slow path: we have to calculate the new pin by moving
			// from our viewport.
			top := self.GetTopLeft(PointTagViewport)
			var p PageListPin
			if n < 0 {
				assert(n != math.MinInt) // Zig: `-n` traps on overflow
				ov := top.UpOverflow(uint(-n))
				switch ov.Tag {
				case PageListPinOverflowOffset:
					p = ov.Offset
				case PageListPinOverflowOverflow:
					p = ov.Overflow.end
				}
			} else {
				ov := top.DownOverflow(uint(n))
				switch ov.Tag {
				case PageListPinOverflowOffset:
					p = ov.Offset
				case PageListPinOverflowOverflow:
					p = ov.Overflow.end
				}
			}

			// If we are still within the active area, then we pin the
			// viewport to active. This isn't EXACTLY the same behavior as
			// other scrolling because normally when you scroll the viewport
			// is pinned to _that row_ even if new scrollback is created.
			// But in a terminal when you get to the bottom and back into the
			// active area, you usually expect that the viewport will now
			// follow the active area.
			if self.pinIsActive(p) {
				self.viewport = PageListViewport{Tag: PageListViewportActive}
				return
			}

			// If we're at the top, then just set the top. This is a lot
			// more efficient everywhere. We must check this after the
			// active check above because we prefer active if they overlap.
			if self.pinIsTop(p) {
				self.viewport = PageListViewport{Tag: PageListViewportTop}
				return
			}

			// Pin is not active so we need to track it.
			*self.viewport_pin = p
			self.viewport = PageListViewport{Tag: PageListViewportPin}
			self.viewport_pin_row_offset = None[uint]() // invalidate cache
		}()
	default:
		panic("unreachable")
	}
}

// scrollPrompt is PageList.scrollPrompt.
//
// Jump the viewport forwards (positive) or backwards (negative) a set number of
// prompts (delta).
//
// Zig: delta is isize.
func (self *PageList) scrollPrompt(delta int) {
	// If we aren't jumping any prompts then we don't need to do anything.
	if delta == 0 {
		return
	}
	var delta_start uint
	if delta > 0 {
		delta_start = uint(delta)
	} else {
		assert(delta != math.MinInt) // Zig: `-delta` traps on overflow
		delta_start = uint(-delta)
	}
	delta_rem := delta_start

	// We start at the row before or after our viewport depending on the
	// delta so that we don't land back on our current viewport.
	var start_pin PageListPin
	{
		tl := self.GetTopLeft(PointTagViewport)

		// If we're moving up we can just move the viewport up because
		// promptIterator handles jumpting to the start of prompts.
		if delta <= 0 {
			up := tl.Up(1)
			if !up.Valid {
				return
			}
			start_pin = up.Value
		} else {
			// If we're moving down and we're presently at some kind of
			// prompt, we need to skip all the continuation lines because
			// promptIterator can't know if we're cutoff or continuing.
			adjusted_ := tl.Down(1)
			if !adjusted_.Valid {
				return
			}
			adjusted := adjusted_.Value
			if tl.RowAndCell().row.semantic_prompt != PageRowSemanticPromptNone {
				// Zig: the labeled block `skip:`.
				for adjusted.RowAndCell().row.semantic_prompt == PageRowSemanticPromptPromptContinuation {
					next := adjusted.Down(1)
					if !next.Valid {
						break // break :skip
					}
					adjusted = next.Value
				}
			}

			start_pin = adjusted
		}
	}

	// Go through prompts delta times
	direction := PageListDirectionLeftUp
	if delta > 0 {
		direction = PageListDirectionRightDown
	}
	it := start_pin.PromptIterator(
		direction,
		None[PageListPin](),
	)
	var prompt_pin Optional[PageListPin]
	for next := it.Next(); next.Valid; next = it.Next() {
		prompt_pin = next
		delta_rem -= 1
		if delta_rem == 0 {
			break
		}
	}

	// If we found a prompt, we move to it. If the prompt is in the active
	// area we keep our viewport as active because we can't scroll DOWN
	// into the active area. Otherwise, we scroll up to the pin.
	if prompt_pin.Valid {
		p := prompt_pin.Value
		if self.pinIsActive(p) {
			self.viewport = PageListViewport{Tag: PageListViewportActive}
		} else {
			*self.viewport_pin = p
			self.viewport = PageListViewport{Tag: PageListViewportPin}
			self.viewport_pin_row_offset = None[uint]() // invalidate cache
		}
	}
}

// ScrollClear is PageList.scrollClear.
//
// Clear the screen by scrolling written contents up into the scrollback.
// This will not update the viewport.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *PageList) ScrollClear() error {
	defer self.AssertIntegrity()

	// Go through the active area backwards to find the first non-empty
	// row. We use this to determine how many rows to scroll up.
	//
	// Zig: the labeled block `non_empty:`, a closure here.
	non_empty := func() uint {
		page := self.pages.last
		assert(page != nil) // .?
		var n uint = 0
		for {
			rows := page.data.rows
			for i := range uint(page.data.size.rows) {
				rev_i := uint(page.data.size.rows) - i - 1
				row := &rows[rev_i]
				cells := page.data.rowCells(row)[0:self.cols]
				for _, cell := range cells {
					if !cell.IsEmpty() {
						return uint(self.rows) - n
					}
				}

				n += 1
				if n > uint(self.rows) {
					return 0
				}
			}

			if page.prev == nil {
				return 0
			}
			page = page.prev
		}
	}()

	// Scroll
	for range non_empty {
		if _, err := self.Grow(); err != nil {
			return err
		}
	}
	return nil
}

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
	defer self.AssertIntegrity()
	page := &node.data

	// We should never have empty rows in our pagelist anyways...
	assert(page.size.rows > 0)

	// We never compact standard size or smaller pages because changing
	// the capacity to something smaller won't save memory.
	if pageListMemoryLen(page) <= pageListStdSize() {
		return nil, nil
	}

	// Compute the minimum capacity required for this page's content
	req_cap := page.ExactRowCapacity(0, uint(page.size.rows))
	new_size := PageLayoutOf(req_cap).total_size
	old_size := pageListMemoryLen(page)
	if new_size >= old_size {
		return nil, nil
	}

	// Create the new smaller page
	new_node, err := self.createPage(req_cap)
	if err != nil {
		return nil, err
	}
	// errdefer self.destroyNode(new_node): no error is returned after
	// this point.
	new_page := &new_node.data
	new_page.size = page.size
	new_page.dirty = page.dirty
	if err := new_page.CloneFrom(
		page,
		0,
		uint(page.size.rows),
	); err != nil {
		// cloneFrom should not fail when compacting since req_cap is
		// computed to exactly fit the source content and our expectation
		// of exactRowCapacity ensures it can fit all the requested
		// data.
		//
		// Zig: log.err
		warn("page_list", "compact clone failed err=error.%v", err)

		// In this case, let's gracefully degrade by pretending we
		// didn't need to compact.
		self.destroyNode(new_node)
		return nil, nil
	}

	// Fix up all tracked pins to point to the new page
	pin_keys := self.tracked_pins.Keys()
	for _, p := range pin_keys {
		if p.node != node {
			continue
		}
		p.node = new_node
	}

	// Insert the new page and destroy the old one
	self.pages.InsertBefore(node, new_node)
	self.pages.Remove(node)
	self.destroyNode(node)

	new_page.AssertIntegrity()
	return new_node, nil
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
func (self *PageList) Split(p PageListPin) (err error) {
	if pageSlowRuntimeSafety {
		assert(self.PinIsValid(p))
	}

	// Ran into a bug that I can only explain via aliasing. If a tracked
	// pin is passed in, its possible Zig will alias the memory and then
	// when we modify it later it updates our p here. Copying the node
	// fixes this.
	//
	// (Go passes p by value, so it never aliases a tracked pin.)
	original_node := p.node
	page := &original_node.data

	// A page that is already 1 row can't be split. In the future we can
	// theoretically maybe split by soft-wrapping multiple pages but that
	// seems crazy and the rest of our PageList can't handle heterogeneously
	// sized pages today.
	if page.size.rows <= 1 {
		return ErrOutOfSpace
	}

	// Splitting at row 0 is a no-op since there's nothing before the split point.
	if p.y == 0 {
		return nil
	}

	// At this point we're doing actual modification so make sure
	// on the return that we're good.
	defer self.AssertIntegrity()

	// Create a new node with the same capacity of managed memory.
	target, err := self.createPage(page.capacity)
	if err != nil {
		return err
	}
	defer func() {
		// errdefer
		if err != nil {
			self.destroyNode(target)
		}
	}()

	// Determine how many rows we're copying
	y_start := p.y
	y_end := page.size.rows
	target.data.size.rows = y_end - y_start
	assert(target.data.size.rows <= target.data.capacity.rows)

	// Copy our old data. This should NOT fail because we have the
	// capacity of the old page which already fits the data we requested.
	if err := target.data.CloneFrom(page, uint(y_start), uint(y_end)); err != nil {
		// Zig: log.err
		warn(
			"page_list",
			"error cloning rows for split err=error.%v",
			err,
		)

		// Rather than crash, we return an OutOfSpace to show that
		// we couldn't split and let our callers gracefully handle it.
		// Realistically though... this should not happen.
		return ErrOutOfSpace
	}

	// From this point forward there is no going back. We have no
	// error handling. It is possible but we haven't written it.
	//
	// errdefer comptime unreachable;

	// Move any tracked pins from the copied rows
	for _, tracked := range self.tracked_pins.Keys() {
		if &tracked.node.data != page ||
			tracked.y < p.y {
			continue
		}

		tracked.node = target
		tracked.y -= p.y
		// p.x remains the same since we're copying the row as-is
	}

	// Clear our rows
	rows := page.rows[y_start:y_end]
	for i := range rows {
		page.ClearCells(
			&rows[i],
			0,
			uint(page.size.cols),
		)
	}
	page.size.rows -= y_end - y_start

	self.pages.InsertAfter(original_node, target)

	return nil
}

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
func (self PageListScrollbar) Eql(other PageListScrollbar) bool {
	return self.total == other.total &&
		self.offset == other.offset &&
		self.len == other.len
}

// Scrollbar is PageList.scrollbar.
//
// Return the scrollbar state for this PageList.
//
// This may be expensive to calculate depending on where the viewport
// is (arbitrary pins are expensive). The caller should take care to only
// call this as needed and not too frequently.
func (self *PageList) Scrollbar() PageListScrollbar {
	// If we have no scrollback, special case no scrollbar.
	// We need to do this because the way PageList works is that
	// it always has SOME extra space (due to the way we allocate by page).
	// So even with no scrollback we have some growth. It is architecturally
	// much simpler to just hide that for no-scrollback cases.
	if self.explicit_max_size == 0 {
		return PageListScrollbar{
			total:  uint(self.rows),
			offset: 0,
			len:    uint(self.rows),
		}
	}

	return PageListScrollbar{
		total:  self.total_rows,
		offset: self.viewportRowOffset(),
		len:    uint(self.rows), // Length is always rows
	}
}

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
func (self *PageList) viewportRowOffset() uint {
	switch self.viewport.Tag {
	case PageListViewportTop:
		return 0
	case PageListViewportActive:
		assert(self.total_rows >= uint(self.rows)) // Zig: usize `-` traps
		return self.total_rows - uint(self.rows)
	case PageListViewportPin:
		// We assert integrity on this code path because it verifies
		// that the cached value is correct.
		defer self.AssertIntegrity()

		// Return cached value if available
		if cached := self.viewport_pin_row_offset; cached.Valid {
			return cached.Value
		}

		// Traverse from the end and count rows until we reach the
		// viewport pin. We count backwards because most of the time
		// a user is scrolling near the active area.
		var top_offset uint
		{
			found := false
			var offset uint = 0
			for n := self.pages.last; n != nil; n = n.prev {
				offset += uint(n.data.size.rows)
				if n == self.viewport_pin.node {
					assert(n.data.size.rows > self.viewport_pin.y)
					offset -= uint(self.viewport_pin.y)
					assert(self.total_rows >= offset) // Zig: usize `-` traps
					top_offset = self.total_rows - offset
					found = true
					break
				}
			}

			// Invalid pins are not possible.
			if !found {
				panic("unreachable")
			}
		}

		// The offset is from the bottom and our cached value and this
		// function returns from the top, so we need to invert it.
		self.viewport_pin_row_offset = Some(top_offset)
		return top_offset
	default:
		panic("unreachable")
	}
}

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
func (self *PageList) fixupViewport(removed uint) {
	switch self.viewport.Tag {
	case PageListViewportActive:

	// For pin, we check if our pin is now in the active area and if so
	// we move our viewport back to the active area.
	case PageListViewportPin:
		if self.pinIsActive(*self.viewport_pin) {
			self.viewport = PageListViewport{Tag: PageListViewportActive}
		} else if self.viewport_pin_row_offset.Valid {
			v := &self.viewport_pin_row_offset.Value
			// If we have a cached row offset, we need to update it
			// to account for the erased rows.
			if *v < removed {
				self.viewport = PageListViewport{Tag: PageListViewportTop}
			} else {
				*v -= removed
			}
		}

	// For top, we move back to active if our erasing moved our
	// top page into the active area.
	case PageListViewportTop:
		assert(self.pages.first != nil) // .?
		if self.pinIsActive(PageListPin{node: self.pages.first}) {
			self.viewport = PageListViewport{Tag: PageListViewportActive}
		}

	default:
		panic("unreachable")
	}
}

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
func (self *PageList) MaxSize() uint64 {
	return max(self.explicit_max_size, self.min_max_size)
}

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
func (self *PageList) Grow() (*PageListNode, error) {
	defer self.AssertIntegrity()

	last := self.pages.last
	assert(last != nil) // .?
	if last.data.capacity.rows > last.data.size.rows {
		// Fast path: we have capacity in the last page.
		last.data.size.rows += 1
		last.data.AssertIntegrity()

		// Increase our total rows by one
		self.total_rows += 1

		return nil, nil
	}

	// Slower path: we have no space, we need to allocate a new page.

	// Get the layout first so our failable work is done early.
	// We'll need this for both paths.
	cap := pageListInitialCapacity(self.cols)

	// If allocation would exceed our max size, we prune the first page.
	// We don't need to reallocate because we can simply reuse that first
	// page.
	//
	// We only take this path if we have more than one page since pruning
	// reuses the popped page. It is possible to have a single page and
	// exceed the max size if that page was adjusted to be larger after
	// initial allocation.
	if self.pages.first != nil &&
		self.pages.first != self.pages.last &&
		self.page_size+pageListPagePoolItemSize() > self.MaxSize() {
		// Zig: the labeled block `prune:`; `break :prune` returns false
		// from the closure, `return first` returns true.
		if first, pruned := func() (*PageListNode, bool) {
			first := self.pages.PopFirst()
			assert(first != nil) // .?
			assert(first != last)

			// Decrease our total row count from the pruned page
			assert(self.total_rows >= uint(first.data.size.rows)) // Zig: usize `-=` traps
			self.total_rows -= uint(first.data.size.rows)

			// If our total row count is now less than our required
			// rows then we can't prune. The "+ 1" is because we'll add one
			// more row below.
			if self.total_rows+1 < uint(self.rows) {
				self.pages.Prepend(first)
				assert(self.pages.first == first)
				self.total_rows += uint(first.data.size.rows)
				return nil, false // break :prune
			}

			// If we have a pin viewport cache then we need to update it.
			if self.viewport.Tag == PageListViewportPin {
				// Zig: the labeled block `viewport:`.
				if self.viewport_pin_row_offset.Valid {
					v := &self.viewport_pin_row_offset.Value
					// If our offset is less than the number of rows in the
					// pruned page, then we are now at the top.
					if *v < uint(first.data.size.rows) {
						self.viewport = PageListViewport{Tag: PageListViewportTop}
					} else {
						// Otherwise, our viewport pin is below what we pruned
						// so we just decrement our offset.
						*v -= uint(first.data.size.rows)
					}
				}
			}

			// Update any tracked pins that point to this page to point to the
			// new first page to the top-left, and mark them as garbage.
			pin_keys := self.tracked_pins.Keys()
			for _, p := range pin_keys {
				if p.node != first {
					continue
				}
				assert(self.pages.first != nil) // .?
				p.node = self.pages.first
				p.y = 0
				p.x = 0
				p.garbage = true
			}
			self.viewport_pin.garbage = false

			// Non-standard pages can't be reused, just destroy them.
			std_size := pageListStdSize()
			if pageListMemoryLen(&first.data) > std_size {
				self.destroyNode(first)
				return nil, false // break :prune
			}

			// Reset our memory
			//
			// Zig: `const buf = first.data.memory; @memset(buf, 0);
			// assert(buf.len <= std_size)`. The buffer is the page's rows
			// and cells; initBuf below reuses them as Zig reuses the
			// buffer (a new capacity with more rows or cells than the old
			// one gets new slices).
			clear(first.data.rows)
			clear(first.data.cells)
			assert(pageListMemoryLen(&first.data) <= std_size)

			// Initialize our new page and reinsert it as the last
			layout := PageLayoutOf(cap)
			buf_rows := first.data.rows
			if uint(len(buf_rows)) >= uint(layout.capacity.rows) {
				buf_rows = buf_rows[:layout.capacity.rows]
			} else {
				buf_rows = make([]PageRow, layout.capacity.rows)
			}
			buf_cells := first.data.cells
			cells_len := uint(layout.capacity.cols) * uint(layout.capacity.rows)
			if uint(len(buf_cells)) >= cells_len {
				buf_cells = buf_cells[:cells_len]
			} else {
				buf_cells = make([]PageCell, cells_len)
			}
			first.data = pageInitBuf(buf_rows, buf_cells, layout)
			first.data.size.rows = 1
			self.pages.InsertAfter(last, first)
			self.total_rows += 1

			// We also need to reset the serial number. Since this is the only
			// place we ever reuse a serial number, we also can safely set
			// page_serial_min to be one more than the old serial because we
			// only ever prune the oldest pages.
			self.page_serial_min = first.serial + 1
			first.serial = self.page_serial
			self.page_serial += 1

			// In this case we do NOT need to update page_size because
			// we're reusing an existing page so nothing has changed.

			first.data.AssertIntegrity()
			return first, true
		}(); pruned {
			return first, nil
		}
	}

	// We need to allocate a new memory buffer.
	next_node, err := self.createPage(cap)
	if err != nil {
		return nil, err
	}
	// we don't errdefer this because we've added it to the linked
	// list and its fine to have dangling unused pages.
	self.pages.Append(next_node)
	next_node.data.size.rows = 1

	// We should never be more than our max size here because we've
	// verified the case above.
	next_node.data.AssertIntegrity()

	// Record the increased row count
	self.total_rows += 1

	return next_node, nil
}

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
func (self PageListIncreaseCapacity) String() string {
	return pageListIncreaseCapacityNames[self]
}

// pageListIncreaseCapacityNames are the tag names of
// PageListIncreaseCapacity, indexed by value (Zig's @tagName).
var pageListIncreaseCapacityNames = [...]string{"styles", "grapheme_bytes", "hyperlink_bytes", "string_bytes"}

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
	defer self.AssertIntegrity()
	page := &node.data

	// Apply our adjustment
	cap := page.capacity
	if adjustment.Valid {
		// Zig: `inline else` over the tags, each naming a field of
		// Capacity (@field, @FieldType); written out per field. The
		// doubling is pageListIncreaseCapacityDouble.
		var ok bool
		switch adjustment.Value {
		case PageListIncreaseCapacityStyles:
			cap.styles, ok = pageListIncreaseCapacityDouble(cap.styles)
		case PageListIncreaseCapacityGraphemeBytes:
			cap.grapheme_bytes, ok = pageListIncreaseCapacityDouble(cap.grapheme_bytes)
		case PageListIncreaseCapacityHyperlinkBytes:
			cap.hyperlink_bytes, ok = pageListIncreaseCapacityDouble(cap.hyperlink_bytes)
		case PageListIncreaseCapacityStringBytes:
			cap.string_bytes, ok = pageListIncreaseCapacityDouble(cap.string_bytes)
		default:
			panic("unreachable")
		}
		if !ok {
			return nil, ErrOutOfSpace
		}

		// If our capacity exceeds the maximum page size, treat it
		// as an OutOfSpace because things like page splitting will
		// help.
		layout := PageLayoutOf(cap)
		if layout.total_size > SizeMaxPageSize {
			return nil, ErrOutOfSpace
		}
	}

	// Zig: log.info("adjusting page capacity={}", .{cap}); not printed by
	// the port.

	// Create our new page and clone the old page into it.
	new_node, err := self.createPage(cap)
	if err != nil {
		return nil, err
	}
	// errdefer self.destroyNode(new_node): no error is returned after
	// this point.
	new_page := &new_node.data
	assert(new_page.capacity.rows >= page.capacity.rows)
	assert(new_page.capacity.cols >= page.capacity.cols)
	new_page.size.rows = page.size.rows
	new_page.size.cols = page.size.cols
	if err := new_page.CloneFrom(
		page,
		0,
		uint(page.size.rows),
	); err != nil {
		// cloneFrom only errors if there isn't capacity for the data
		// from the source page but we're only increasing capacity so
		// this should never be possible. If it happens, we should crash
		// because we're in no man's land and can't safely recover.
		//
		// Zig: log.err
		warn("page_list", "increaseCapacity clone failed err=error.%v", err)
		panic("unexpected clone failure")
	}

	// Preserve page-level dirty flag (cloneFrom only copies row data)
	new_page.dirty = page.dirty

	// Must not fail after this because the operations we do after this
	// can't be recovered.
	//
	// errdefer comptime unreachable;

	// Fix up all our tracked pins to point to the new page.
	pin_keys := self.tracked_pins.Keys()
	for _, p := range pin_keys {
		if p.node != node {
			continue
		}
		p.node = new_node
	}

	// Insert this page and destroy the old page
	self.pages.InsertBefore(node, new_node)
	self.pages.Remove(node)
	self.destroyNode(node)

	new_page.AssertIntegrity()
	return new_node, nil
}

// pageListIncreaseCapacityDouble is NOT a Zig declaration: the body of
// increaseCapacity's `inline else` for one Capacity field of integer type
// T. It returns the doubled value, or false where Zig returns
// error.OutOfSpace.
//
// We use checked math to prevent overflow. If there is an
// overflow it means we're out of space in this dimension,
// since pages can take up to their maxInt capacity in any
// category.
func pageListIncreaseCapacityDouble[T uint16 | uint32](old T) (T, bool) {
	max_int := ^T(0) // std.math.maxInt(Int)
	// Zig: std.math.mul(Int, old, 2) catch |err| overflow: { ... }
	if old > max_int/2 {
		// Our final doubling would overflow since maxInt is
		// 2^N - 1 for an unsignged int of N bits. So, if we overflow
		// and we haven't used all the bits, use all the bits.
		if old < max_int {
			return max_int, true
		}
		return 0, false
	}
	return old * 2, true
}

// createPage is PageList.createPage.
//
// Create a new page node. This does not add it to the list and this
// does not do any memory size accounting with max_size/page_size.
//
// Zig: Allocator.Error!*List.Node. The error is ErrOutOfMemory (dcs.go).
// (It does add the page's bytes to `page_size`, through createPageExt.)
func (self *PageList) createPage(cap PageCapacity) (*PageListNode, error) {
	// log.debug("create page cap={}", .{cap});
	return pageListCreatePageExt(
		&self.pool,
		cap,
		&self.page_serial,
		&self.page_size,
	)
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
	_ = pool
	page := new(PageListNode) // pool.nodes.create()
	// errdefer pool.nodes.destroy(page): Go memory.

	layout := PageLayoutOf(cap)
	std_size := pageListStdSize()
	pooled := layout.total_size <= std_size
	// Zig: const page_alloc = pool.pages.arena.child_allocator; Go memory.

	// It would be better to encode this into the Zig error handling
	// system but that is a big undertaking and we only have a few
	// centralized call sites so it is handled on its own currently.
	assert(layout.total_size <= SizeMaxPageSize)

	// Our page buffer comes from our standard memory pool if it
	// is within our standard size since this is what the pool
	// dispenses. Otherwise, we use the heap allocator to allocate.
	//
	// Zig: the buffer is a pool item of std_size bytes when pooled,
	// otherwise an allocation of layout.total_size bytes. Its length is
	// what total_size counts. In Go the buffer is the page's rows and
	// cells.
	var page_buf_len uint64
	if pooled {
		page_buf_len = std_size
	} else {
		page_buf_len = layout.total_size
	}
	page_rows := make([]PageRow, layout.capacity.rows)
	page_cells := make([]PageCell, uint(layout.capacity.cols)*uint(layout.capacity.rows))
	// errdefer destroy/free of page_buf: Go memory.

	// Required only with runtime safety because allocators initialize
	// to undefined, 0xAA.
	//
	// Zig: @memset(page_buf, 0). Go memory is zeroed.

	*page = PageListNode{
		data:   pageInitBuf(page_rows, page_cells, layout),
		serial: *serial,
	}
	page.data.size.rows = 0
	*serial += 1

	if v := total_size; v != nil {
		// Accumulate page size now. We don't assert or check max size
		// because we may exceed it here temporarily as we are allocating
		// pages before destroy.
		*v += page_buf_len
	}

	return page, nil
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
func (self *PageList) destroyNode(node *PageListNode) {
	pageListDestroyNodeExt(&self.pool, node, &self.page_size)
}

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
	_ = pool
	page := &node.data

	// Update our accounting for page size
	if v := total_size; v != nil {
		memory_len := pageListMemoryLen(page)
		assert(*v >= memory_len) // Zig: usize `-=` traps
		*v -= memory_len
	}

	// Zig: a standard page's memory is zeroed ("Reset the memory to zero
	// so it can be reused") and returned to the page pool; a non-standard
	// page's memory is freed; the node is returned to the node pool. The
	// garbage collector frees in Go.
}

// pageListMemoryLen is NOT a Zig declaration. It stands for
// `page.memory.len`: `PageLayoutOf(page.capacity).total_size`, for pooled
// and non-pooled pages alike (not the pool buffer's std_size; see the file
// header). PageList.zig reads it in initPages, deinit,
// reset, clone, compact, grow, createPageExt and destroyNodeExt, and tests
// read `node.data.memory.len`. See the file header.
func pageListMemoryLen(page *Page) uint64 {
	return PageLayoutOf(page.capacity).total_size
}

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

// HighlightSemanticContent is PageList.highlightSemanticContent.
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
//
// Zig: pub fn highlightSemanticContent(self: *const PageList, at: Pin,
// content: pagepkg.Cell.SemanticContent) ?highlight.Untracked.
// HighlightUntracked is in highlight.go.
func (self *PageList) HighlightSemanticContent(
	at PageListPin,
	content PageCellSemanticContent,
) Optional[HighlightUntracked] {
	panic(notImplemented)
}

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
