// Ported from src/terminal/hyperlink.zig (Ghostty v1.3.1).
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

import "bytes"

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

// hyperlinkHyperlinkIdTagNames are the tag names of HyperlinkHyperlinkIdTag,
// indexed by value (Zig's @tagName).
var hyperlinkHyperlinkIdTagNames = [...]string{"explicit", "implicit"}

// String returns the Zig tag name (@tagName).
func (self HyperlinkHyperlinkIdTag) String() string {
	return hyperlinkHyperlinkIdTagNames[self]
}

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
func (self *Hyperlink) Deinit() {
	// Zig: `alloc.free(self.uri)`, and `alloc.free(v)` of an explicit id
	// (an implicit id owns no memory). The garbage collector frees in Go;
	// the switch is kept for its exhaustiveness.
	switch self.id.Tag {
	case HyperlinkHyperlinkIdImplicit:
	case HyperlinkHyperlinkIdExplicit:
	default:
		panic("unreachable")
	}
}

// Dupe is hyperlink.Hyperlink.dupe.
//
// Duplicate a hyperlink by allocating all values with the
// given allocator. The returned hyperlink should have deinit
// called.
//
// Zig: pub fn dupe(self: *const Hyperlink, alloc: Allocator)
// Allocator.Error!Hyperlink. The allocator parameter is dropped.
func (self *Hyperlink) Dupe() (Hyperlink, error) {
	// alloc.dupe(u8, self.uri); the allocation cannot fail in Go.
	uri := make([]uint8, len(self.uri))
	copy(uri, self.uri)
	// Zig: errdefer alloc.free(uri): nothing below fails.

	var id HyperlinkHyperlinkId
	switch self.id.Tag {
	case HyperlinkHyperlinkIdImplicit:
		id = self.id
	case HyperlinkHyperlinkIdExplicit:
		v := self.id.Explicit
		dup := make([]uint8, len(v)) // alloc.dupe(u8, v)
		copy(dup, v)
		id = HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: dup}
	default:
		panic("unreachable")
	}
	// Zig: errdefer of the explicit id: nothing below fails.

	return Hyperlink{id: id, uri: uri}, nil
}

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

// hyperlinkPageEntryIdTagNames are the tag names of HyperlinkPageEntryIdTag,
// indexed by value (Zig's @tagName).
var hyperlinkPageEntryIdTagNames = [...]string{"explicit", "implicit"}

// String returns the Zig tag name (@tagName).
func (self HyperlinkPageEntryIdTag) String() string {
	return hyperlinkPageEntryIdTagNames[self]
}

// Dupe is hyperlink.PageEntry.dupe: duplicate this hyperlink from one page
// to another.
//
// Zig: pub fn dupe(self: *const PageEntry, self_page: *const Page,
// dst_page: *Page) error{OutOfMemory}!PageEntry.
func (self *HyperlinkPageEntry) Dupe(self_page *Page, dst_page *Page) (HyperlinkPageEntry, error) {
	var copy_ = *self

	// If the pages are the same then we can return a shallow copy.
	if self_page == dst_page {
		return copy_, nil
	}

	// Copy the URI
	{
		uri := self.uri // self.uri.slice(self_page.memory)
		buf, err := dst_page.string_alloc.Alloc(uint(len(uri)))
		if err != nil {
			return HyperlinkPageEntry{}, err
		}
		copy(buf, uri) // @memcpy(buf, uri)
		// Zig: .{ .offset = size.getOffset(u8, dst_page.memory, &buf[0]),
		// .len = uri.len }: the allocation from its first byte, uri.len
		// long.
		copy_.uri = buf[0:len(uri)]
	}
	// Zig: errdefer dst_page.string_alloc.free(dst_page.memory,
	// copy.uri.slice(dst_page.memory)): written out before the error
	// return below.

	// Copy the ID
	switch copy_.id.Tag {
	case HyperlinkPageEntryIdImplicit: // Shallow is fine
	case HyperlinkPageEntryIdExplicit:
		slice := copy_.id.Explicit
		id := slice // slice.slice(self_page.memory)
		buf, err := dst_page.string_alloc.Alloc(uint(len(id)))
		if err != nil {
			// Zig: errdefer (the uri)
			dst_page.string_alloc.Free(copy_.uri)
			return HyperlinkPageEntry{}, err
		}
		copy(buf, id) // @memcpy(buf, id)
		copy_.id = HyperlinkPageEntryId{Tag: HyperlinkPageEntryIdExplicit, Explicit: buf[0:len(id)]}
	default:
		panic("unreachable")
	}
	// Zig: errdefer of the explicit id: nothing below fails.

	return copy_, nil
}

// Hash is hyperlink.PageEntry.hash.
//
// Zig: pub fn hash(self: *const PageEntry, base: anytype) u64. Must
// reproduce data/sets/hyperlink_hash.txt (docs/mapping.md).
func (self *HyperlinkPageEntry) Hash() uint64 {
	var hasher = stdHashWyhashInit(0)
	// autoHash(&hasher, std.meta.activeTag(self.id)): the tag is an
	// enum(u1), explicit 0, implicit 1.
	stdHashAutoHashAutoHashEnumU1(&hasher, uint8(self.id.Tag))
	switch self.id.Tag {
	case HyperlinkPageEntryIdImplicit:
		v := self.id.Implicit
		stdHashAutoHashAutoHashU32(&hasher, v)
	case HyperlinkPageEntryIdExplicit:
		slice := self.id.Explicit
		stdHashAutoHashHashSliceU8( // autoHashStrat
			&hasher,
			slice, // slice.slice(base)
			stdHashAutoHashHashStrategyDeep,
		)
	default:
		panic("unreachable")
	}
	stdHashAutoHashHashSliceU8( // autoHashStrat
		&hasher,
		self.uri, // self.uri.slice(base)
		stdHashAutoHashHashStrategyDeep,
	)
	return hasher.final()
}

// Eql is hyperlink.PageEntry.eql.
//
// Zig: pub fn eql(self: *const PageEntry, self_base: anytype,
// other: *const PageEntry, other_base: anytype) bool.
func (self *HyperlinkPageEntry) Eql(other *HyperlinkPageEntry) bool {
	if self.id.Tag != other.id.Tag {
		return false
	}
	switch self.id.Tag {
	case HyperlinkPageEntryIdImplicit:
		if self.id.Implicit != other.id.Implicit {
			return false
		}
	case HyperlinkPageEntryIdExplicit:
		// Zig: self_ptr[0..self.id.explicit.len] and
		// other_ptr[0..other.id.explicit.len], the two slices.
		self_ptr := self.id.Explicit
		other_ptr := other.id.Explicit
		if !bytes.Equal(
			self_ptr[0:len(self.id.Explicit)],
			other_ptr[0:len(other.id.Explicit)],
		) {
			return false
		}
	default:
		panic("unreachable")
	}

	return bytes.Equal(
		self.uri,  // self.uri.slice(self_base)
		other.uri, // other.uri.slice(other_base)
	)
}

// Free is hyperlink.PageEntry.free: free the memory for this entry from its
// page.
//
// Zig: pub fn free(self: *const PageEntry, page: *Page) void.
func (self *HyperlinkPageEntry) Free(page *Page) {
	alloc := &page.string_alloc
	switch self.id.Tag {
	case HyperlinkPageEntryIdImplicit:
	case HyperlinkPageEntryIdExplicit:
		v := self.id.Explicit
		alloc.Free(v) // alloc.free(page.memory, v.slice(page.memory))
	default:
		panic("unreachable")
	}
	alloc.Free(self.uri) // alloc.free(page.memory, self.uri.slice(page.memory))
}

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
func (self HyperlinkSetContext) Hash(link HyperlinkPageEntry) uint64 {
	// Zig: link.hash((self.src_page orelse self.page.?).memory). The base
	// is gone in Go; the unwrap of self.page when src_page is null stays.
	if self.src_page == nil {
		assert(self.page != nil) // self.page.?
	}
	return link.Hash()
}

// Eql is the context's `eql`.
//
// Zig: pub fn eql(self: *const @This(), a: PageEntry, b: PageEntry) bool.
func (self HyperlinkSetContext) Eql(a HyperlinkPageEntry, b HyperlinkPageEntry) bool {
	// Zig: a.eql((self.src_page orelse self.page.?).memory, &b,
	// self.page.?.memory). The bases are gone in Go; the unwraps stay
	// (the second one covers the first).
	assert(self.page != nil) // self.page.?
	return a.Eql(&b)
}

// Deleted is the context's `deleted` (RefCountedSetContextDeleted).
//
// Zig: pub fn deleted(self: *const @This(), link: PageEntry) void.
func (self HyperlinkSetContext) Deleted(link HyperlinkPageEntry) {
	assert(self.page != nil) // self.page.?
	link.Free(self.page)
}
