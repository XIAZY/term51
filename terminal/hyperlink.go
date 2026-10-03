// WIREFRAME for src/terminal/hyperlink.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: hyperlink.zig is a namespace file, so its declarations take
// the prefix `Hyperlink`: `hyperlink.Id` is `HyperlinkId`, `hyperlink.Map`
// is `HyperlinkMap`, `hyperlink.PageEntry` is `HyperlinkPageEntry`,
// `hyperlink.Set` is `HyperlinkSet`. The struct `hyperlink.Hyperlink` itself
// is `Hyperlink` (not `HyperlinkHyperlink`). Its nested union
// `Hyperlink.Id` would then also be `HyperlinkId`, which is taken by
// `hyperlink.Id`; it keeps the doubled name `HyperlinkHyperlinkId`.
//
// Declarations appear in the same order as in hyperlink.zig.
//
// # Go-memory rule, option A
//
// (docs/mapping.md, governing rule 3, as refined for this batch.)
//
//   - `Offset(u8).Slice` (an offset into the page's string memory and a
//     length) is a Go `[]uint8` that is a sub-slice of the page's string
//     allocator backing (`Page.string_alloc`). Zig's default `.{}` (offset
//     0, length 0) is the nil slice.
//   - every `base` parameter is dropped (`PageEntry.hash`, `PageEntry.eql`);
//     `*Page` parameters are kept.
//   - the key of `Map`, Zig `Offset(Cell)`, is the byte offset as a number;
//     see page.go.
//
// Two declarations here are not Zig declarations; both hold numbers that Zig
// reads from types at compile time and that the byte arithmetic of page.zig
// needs: hyperlinkMapEntrySizes and hyperlinkSetItemSizeOf. (The size and
// alignment of a set item as RefCountedSetLayoutInit takes them,
// hyperlinkSetItemSizes, are in page.go.)
//
// Widths: every byte size and byte offset is a `uint64`, whatever the
// platform word size, because the reference is the recorded 64-bit data and
// the target is 32-bit. Counts, indexes and lengths stay `uint` or their
// fixed Zig width.
// Here: hyperlinkMapEntrySizes holds uint64; hyperlinkSetItemSizeOf is an
// untyped constant.
//
// The file has no C ABI declarations and no tests upstream.

package terminal

// HyperlinkId is hyperlink.Id: the unique identifier for a hyperlink. This
// is at most the number of cells that can fit in a single terminal page,
// since each cell can only contain at most one hyperlink.
type HyperlinkId = SizeHyperlinkCountInt

// HyperlinkMap is hyperlink.Map.
//
// The mapping of cell to hyperlink. We use an offset hash map to save space
// since its very unlikely a cell is a hyperlink, so its a waste to store
// the hyperlink ID in the cell itself.
//
// Zig: AutoOffsetHashMap(Offset(Cell), Id). The key is the cell's byte
// offset in the page as upstream would have it; see Page.grapheme_map in
// page.go.
type HyperlinkMap = HashMapAutoOffsetHashMap[SizeOffsetInt, HyperlinkId]

// hyperlinkMapEntrySizes is NOT a Zig declaration: @sizeOf and @alignOf of
// the key and value types of hyperlink.Map, which
// HashMapOffsetHashMapLayoutOf needs (see hash_map.go). Key Offset(Cell): a
// packed struct(u32), 4 and 4. Value Id: u16, 2 and 2.
var hyperlinkMapEntrySizes = hashMapEntrySizes{key_size: 4, key_align: 4, val_size: 2, val_align: 2}

// Hyperlink is hyperlink.Hyperlink.
//
// A fully decoded hyperlink that may or may not have its
// memory within a page. The memory location of this is dependent
// on the context so users should check with the source of the
// hyperlink.
type Hyperlink struct {
	id  HyperlinkHyperlinkId
	uri []uint8
}

// HyperlinkHyperlinkId is hyperlink.Hyperlink.Id. See PageEntry.Id.
//
// Zig: union(enum). Only the field named by Tag is meaningful. The Go zero
// value is `.explicit` with an empty slice.
type HyperlinkHyperlinkId struct {
	Tag HyperlinkHyperlinkIdTag

	Explicit []uint8
	Implicit SizeOffsetInt
}

// HyperlinkHyperlinkIdTag is the tag of hyperlink.Hyperlink.Id, values
// numbered from zero in the order listed.
type HyperlinkHyperlinkIdTag uint8

const (
	HyperlinkHyperlinkIdExplicit HyperlinkHyperlinkIdTag = iota
	HyperlinkHyperlinkIdImplicit
)

// String returns the Zig tag name (@tagName).
func (self HyperlinkHyperlinkIdTag) String() string { panic(notImplemented) }

// Deinit is hyperlink.Hyperlink.deinit.
//
// Deinit and deallocate all the pointers using the given
// allocator.
//
// WARNING: This should only be called if the hyperlink was
// heap-allocated. This DOES NOT need to be unconditionally
// called.
//
// Zig: pub fn deinit(self: *const Hyperlink, alloc: Allocator) void. The
// allocator parameter is dropped. It only frees; it is kept as a stub
// because the tests call it.
func (self *Hyperlink) Deinit() { panic(notImplemented) }

// Dupe is hyperlink.Hyperlink.dupe.
//
// Duplicate a hyperlink by allocating all values with the
// given allocator. The returned hyperlink should have deinit
// called.
//
// Zig: pub fn dupe(self: *const Hyperlink, alloc: Allocator)
// Allocator.Error!Hyperlink. The allocator parameter is dropped.
func (self *Hyperlink) Dupe() (Hyperlink, error) { panic(notImplemented) }

// HyperlinkPageEntry is hyperlink.PageEntry.
//
// A hyperlink that has been committed to page memory. This
// is a "page entry" because while it represents a hyperlink,
// some decoding (pointer chasing) is still necessary to get the
// fully realized ID, URI, etc.
type HyperlinkPageEntry struct {
	id HyperlinkPageEntryId

	// Zig: Offset(u8).Slice. See the file header.
	uri []uint8
}

// HyperlinkPageEntryId is hyperlink.PageEntry.Id.
//
// Zig: union(enum). Only the field named by Tag is meaningful. The Go zero
// value is `.explicit` with an empty slice.
type HyperlinkPageEntryId struct {
	Tag HyperlinkPageEntryIdTag

	// An explicitly provided ID via the OSC8 sequence.
	//
	// Zig: Offset(u8).Slice. See the file header.
	Explicit []uint8

	// No ID was provided so we auto-generate the ID based on an
	// incrementing counter attached to the screen.
	Implicit SizeOffsetInt
}

// HyperlinkPageEntryIdTag is the tag of hyperlink.PageEntry.Id, values
// numbered from zero in the order listed.
type HyperlinkPageEntryIdTag uint8

const (
	HyperlinkPageEntryIdExplicit HyperlinkPageEntryIdTag = iota
	HyperlinkPageEntryIdImplicit
)

// String returns the Zig tag name (@tagName).
func (self HyperlinkPageEntryIdTag) String() string { panic(notImplemented) }

// Dupe is hyperlink.PageEntry.dupe: duplicate this hyperlink from one page
// to another.
//
// Zig: pub fn dupe(self: *const PageEntry, self_page: *const Page,
// dst_page: *Page) error{OutOfMemory}!PageEntry.
func (self *HyperlinkPageEntry) Dupe(self_page *Page, dst_page *Page) (HyperlinkPageEntry, error) {
	panic(notImplemented)
}

// Hash is hyperlink.PageEntry.hash.
//
// Zig: pub fn hash(self: *const PageEntry, base: anytype) u64. Must
// reproduce data/sets/hyperlink_hash.txt (docs/mapping.md).
func (self *HyperlinkPageEntry) Hash() uint64 { panic(notImplemented) }

// Eql is hyperlink.PageEntry.eql.
//
// Zig: pub fn eql(self: *const PageEntry, self_base: anytype,
// other: *const PageEntry, other_base: anytype) bool.
func (self *HyperlinkPageEntry) Eql(other *HyperlinkPageEntry) bool { panic(notImplemented) }

// Free is hyperlink.PageEntry.free: free the memory for this entry from its
// page.
//
// Zig: pub fn free(self: *const PageEntry, page: *Page) void.
func (self *HyperlinkPageEntry) Free(page *Page) { panic(notImplemented) }

// HyperlinkSet is hyperlink.Set.
//
// The set of hyperlinks. This is ref-counted so that a set of cells
// can share the same hyperlink without duplicating the data.
//
// Zig: RefCountedSet(PageEntry, Id, size.CellCountInt, struct { ... }).
type HyperlinkSet = RefCountedSet[HyperlinkPageEntry, HyperlinkId, SizeCellCountInt, HyperlinkSetContext]

// hyperlinkSetItemSizeOf is NOT a Zig declaration: it is
// @sizeOf(hyperlink.Set.Item), which page.zig and its tests use to convert
// between a hyperlink count and `Capacity.hyperlink_bytes`.
//
// Value on a 64-bit target: Offset(u8).Slice is { offset: u32, len: usize },
// 16 bytes; PageEntry.Id is a tagged union of that and a u32, 24 bytes;
// PageEntry is 24 + 16 = 40; Item.Metadata is three u16, 6; Item is 46
// rounded up to its alignment of 8 = 48. Confirmed by the recorded layouts
// (term51 data/layout/page_layout.txt: hyperlink_bytes=192 for 4 items, and
// hyperlink_set_layout cap=3 items_start=8 total_size=152).
const hyperlinkSetItemSizeOf = 48

// HyperlinkSetContext is the anonymous context struct of hyperlink.Set,
// reachable in Zig as `Set.Context`.
type HyperlinkSetContext struct {
	// The page which holds the strings for items in this set.
	//
	// Zig: ?*Page, default null.
	page *Page

	// The page which holds the strings for items
	// looked up with, e.g., `add` or `lookup`,
	// if different from the destination page.
	//
	// Zig: ?*const Page, default null.
	src_page *Page
}

// Hash is the context's `hash`.
//
// Zig: pub fn hash(self: *const @This(), link: PageEntry) u64. See
// RefCountedSetContext for the receiver.
func (self HyperlinkSetContext) Hash(link HyperlinkPageEntry) uint64 { panic(notImplemented) }

// Eql is the context's `eql`.
//
// Zig: pub fn eql(self: *const @This(), a: PageEntry, b: PageEntry) bool.
func (self HyperlinkSetContext) Eql(a HyperlinkPageEntry, b HyperlinkPageEntry) bool {
	panic(notImplemented)
}

// Deleted is the context's `deleted` (RefCountedSetContextDeleted).
//
// Zig: pub fn deleted(self: *const @This(), link: PageEntry) void.
func (self HyperlinkSetContext) Deleted(link HyperlinkPageEntry) { panic(notImplemented) }
