// Tests translated one for one from src/terminal/hyperlink.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact Zig
// title in a "// Zig: test" comment. The file has no upstream tests; all of
// them are the project's `term51:` tests.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's
//     (expected, actual) argument order. `try testing.expectError(error.X,
//     call)` is hlExpectError(t, ErrX, <the error of the call>).
//   - Zig's test helpers are the hl helpers: hlSpec (term51FSpec), hlPage
//     (term51FPage), hlUsed (term51FUsed), hlStr (term51FStr), hlEntry
//     (term51FEntry), hlIdStr (term51FIdStr), hlMaxImplicit
//     (term51FMaxImplicit) and hlDistinct (term51FDistinct). hlSpecOf,
//     hlImplicit and hlExplicit build the Zig literals `.{ .uri = ..., .id =
//     ... }`, `.{ .implicit = v }` and `.{ .explicit = s }`.
//   - Go memory: an `Offset(u8).Slice` is a Go []uint8 that is a sub-slice of
//     the page's string allocator, so `x.slice(page.memory)` is x itself and
//     the `base` argument of hash, eql and free-by-memory is gone:
//     `a.hash(a.memory)` is a.Hash() and `a.eql(a.memory, &b, b.memory)` is
//     a.Eql(&b). `allocator` arguments are gone; `defer x.deinit(alloc)`
//     of a hyperlink or a page is `defer x.Deinit()` (kept stubs).
//   - `x.offset.offset` comparisons: within one page, "different offsets"
//     is "different start addresses" (&a[0] != &b[0]). Between two pages the
//     numbers say nothing in Go (the memory is always different); those lines
//     are Zig-only.
//   - Zig's `defer` inside a loop body runs at the end of each iteration, and
//     in a block at the end of the block; Go's runs when the function
//     returns. Where the order matters the body is a func literal called
//     at once.
//   - Zig-only: `const alloc = testing.allocator;` and the leak detection
//     that comes with it are dropped in every test without a per-test note.

package terminal

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

// hlExpectError is Zig's `try testing.expectError(want, got)`.
func hlExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// hlExpectPanic runs f and fails the test unless f panics. Zig's tests cannot
// test an assert or a safety-checked panic: the test would abort.
func hlExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// hlSpec is term51FSpec.
type hlSpec struct {
	uri []uint8
	id  HyperlinkHyperlinkId
}

// hlSpecOf is the Zig literal `.{ .uri = uri, .id = id }`.
func hlSpecOf(uri string, id HyperlinkHyperlinkId) hlSpec {
	return hlSpec{uri: []uint8(uri), id: id}
}

// hlImplicit is the Zig literal `.{ .implicit = v }` of a Hyperlink.Id.
func hlImplicit(v SizeOffsetInt) HyperlinkHyperlinkId {
	return HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: v}
}

// hlExplicit is the Zig literal `.{ .explicit = s }` of a Hyperlink.Id.
func hlExplicit(s string) HyperlinkHyperlinkId {
	return HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8(s)}
}

// hlPage is term51FPage: a page with the given string capacity and room for
// `links` set items. The string allocator rounds its capacity up to 64 chunks
// of 32 bytes.
func hlPage(t *testing.T, string_bytes SizeStringBytesInt, links uint) Page {
	t.Helper()
	cap := PageCapacityInit(4, 4)
	cap.styles = 4
	cap.hyperlink_bytes = SizeHyperlinkCountInt(links * hyperlinkSetItemSizeOf)
	cap.string_bytes = string_bytes
	page, err := PageInit(cap)
	zigTry(t, err)
	return page
}

// hlUsed is term51FUsed.
func hlUsed(page *Page) uint64 {
	return page.string_alloc.UsedBytes()
}

// hlStr is term51FStr: copy `s` into the page string allocator. Empty strings
// are not allocated (the allocator asserts n > 0) and get a zero slice.
func hlStr(t *testing.T, page *Page, s []uint8) []uint8 {
	t.Helper()
	if len(s) == 0 {
		return nil
	}
	buf, err := page.string_alloc.Alloc(uint(len(s)))
	zigTry(t, err)
	copy(buf, s)
	return buf
}

// hlEntry is term51FEntry: build a page entry (not in any set) whose strings
// live in `page`.
func hlEntry(t *testing.T, page *Page, spec hlSpec) HyperlinkPageEntry {
	t.Helper()
	uri := hlStr(t, page, spec.uri)
	var id HyperlinkPageEntryId
	switch spec.id.Tag {
	case HyperlinkHyperlinkIdImplicit:
		id = HyperlinkPageEntryId{Tag: HyperlinkPageEntryIdImplicit, Implicit: spec.id.Implicit}
	case HyperlinkHyperlinkIdExplicit:
		id = HyperlinkPageEntryId{Tag: HyperlinkPageEntryIdExplicit, Explicit: hlStr(t, page, spec.id.Explicit)}
	}
	return HyperlinkPageEntry{id: id, uri: uri}
}

// hlIdStr is term51FIdStr.
func hlIdStr(entry HyperlinkPageEntry) []uint8 {
	switch entry.id.Tag {
	case HyperlinkPageEntryIdImplicit:
		return []uint8("")
	case HyperlinkPageEntryIdExplicit:
		return entry.id.Explicit
	}
	panic("unreachable")
}

// hlMaxImplicit is term51FMaxImplicit.
const hlMaxImplicit SizeOffsetInt = math.MaxUint32

// hlDistinct is term51FDistinct: specs that are all pairwise distinct
// (including the empty edge cases).
var hlDistinct = []hlSpec{
	hlSpecOf("u", hlImplicit(0)),
	hlSpecOf("u", hlImplicit(1)),
	hlSpecOf("u", hlImplicit(hlMaxImplicit)),
	hlSpecOf("u", hlExplicit("0")),
	hlSpecOf("u", hlExplicit("1")),
	hlSpecOf("c", hlExplicit("ab")),
	hlSpecOf("bc", hlExplicit("a")),
	hlSpecOf("abc", hlExplicit("")),
	hlSpecOf("v", hlImplicit(0)),
	hlSpecOf("uu", hlImplicit(0)),
	hlSpecOf("", hlImplicit(0)),
	hlSpecOf("", hlImplicit(1)),
	hlSpecOf("", hlExplicit("")),
	hlSpecOf("", hlExplicit("x")),
	hlSpecOf("u", hlExplicit("")),
}

// Zig: test "term51: Hyperlink.deinit: frees explicit id and uri"
func TestHyperlink_term51_Hyperlink_deinit_frees_explicit_id_and_uri(t *testing.T) {
	// Zig-only: alloc.dupe(u8, ...) of both strings and the leak detection of
	// testing.allocator (the Go strings are plain slices).
	link := Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8("my-id")},
		URI: []uint8("https://example.com"),
	}
	link.Deinit()
}

// Zig: test "term51: Hyperlink.deinit: implicit id frees only the uri"
func TestHyperlink_term51_Hyperlink_deinit_implicit_id_frees_only_the_uri(t *testing.T) {
	// The implicit id owns no memory; freeing it would be detected.
	// Zig-only: the detection (testing.allocator).
	link := Hyperlink{
		ID:  hlImplicit(7),
		URI: []uint8("https://example.com"),
	}
	link.Deinit()
}

// Zig: test "term51: Hyperlink.deinit: empty strings"
func TestHyperlink_term51_Hyperlink_deinit_empty_strings(t *testing.T) {
	link := Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8{}},
		URI: []uint8{},
	}
	link.Deinit()
}

// Zig: test "term51: Hyperlink.dupe: explicit id"
func TestHyperlink_term51_Hyperlink_dupe_explicit_id(t *testing.T) {
	orig := Hyperlink{
		ID:  hlExplicit("my-id"),
		URI: []uint8("https://example.com"),
	}
	cp, err := orig.Dupe()
	zigTry(t, err)
	defer cp.Deinit()

	expectEqualStrings(t, "https://example.com", string(cp.URI))
	expectEqualStrings(t, "my-id", string(cp.ID.Explicit))
	expect(t, &cp.URI[0] != &orig.URI[0], "copy.uri.ptr != orig.uri.ptr")
	expect(t, &cp.ID.Explicit[0] != &orig.ID.Explicit[0], "copy.id.explicit.ptr != orig.id.explicit.ptr")
}

// Zig: test "term51: Hyperlink.dupe: implicit id copied by value"
func TestHyperlink_term51_Hyperlink_dupe_implicit_id_copied_by_value(t *testing.T) {
	for _, n := range []SizeOffsetInt{0, 1, hlMaxImplicit} {
		func() {
			orig := Hyperlink{ID: hlImplicit(n), URI: []uint8("https://a.b")}
			cp, err := orig.Dupe()
			zigTry(t, err)
			defer cp.Deinit()
			expectEqual(t, n, cp.ID.Implicit)
			expectEqualStrings(t, "https://a.b", string(cp.URI))
			expect(t, &cp.URI[0] != &orig.URI[0], "copy.uri.ptr != orig.uri.ptr")
		}()
	}
}

// Zig: test "term51: Hyperlink.dupe: empty uri and empty id"
func TestHyperlink_term51_Hyperlink_dupe_empty_uri_and_empty_id(t *testing.T) {
	orig := Hyperlink{ID: hlExplicit(""), URI: []uint8("")}
	cp, err := orig.Dupe()
	zigTry(t, err)
	defer cp.Deinit()
	expectEqual(t, 0, len(cp.URI))
	expectEqual(t, 0, len(cp.ID.Explicit))
}

// Zig: test "term51: Hyperlink.dupe: very long uri and id"
func TestHyperlink_term51_Hyperlink_dupe_very_long_uri_and_id(t *testing.T) {
	uri := make([]uint8, 100_000)
	// Zig-only: defer alloc.free(uri)
	for i := range uri {
		uri[i] = 'x'
	}
	uri[99_999] = 'y'
	id := make([]uint8, 70_000)
	// Zig-only: defer alloc.free(id)
	for i := range id {
		id[i] = 'i'
	}

	orig := Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: id},
		URI: uri,
	}
	cp, err := orig.Dupe()
	zigTry(t, err)
	defer cp.Deinit()
	expectEqualSlices(t, uri, cp.URI)
	expectEqualSlices(t, id, cp.ID.Explicit)
}

// Not translated (std.testing.FailingAllocator has no Go counterpart): see scripts/not-translated.txt.
//   Zig: test "term51: Hyperlink.dupe: allocation failures leak nothing"

// Zig: test "term51: PageEntry.dupe: same page is a shallow copy"
func TestHyperlink_term51_PageEntry_dupe_same_page_is_a_shallow_copy(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()

	entry := hlEntry(t, &page, hlSpecOf("https://a.b", hlExplicit("id1")))
	used := hlUsed(&page)
	expectEqual(t, uint64(64), used)

	cp, err := entry.Dupe(&page, &page)
	zigTry(t, err)
	expect(t, &entry.uri[0] == &cp.uri[0], "entry.uri.offset.offset == copy.uri.offset.offset")
	expectEqual(t, len(entry.uri), len(cp.uri))
	expect(t, &entry.id.Explicit[0] == &cp.id.Explicit[0], "entry.id.explicit.offset.offset == copy.id.explicit.offset.offset")
	expectEqual(t, len(entry.id.Explicit), len(cp.id.Explicit))
	expectEqual(t, used, hlUsed(&page))

	// Even when strings are empty (nothing would be allocatable).
	empty := HyperlinkPageEntry{id: HyperlinkPageEntryId{Tag: HyperlinkPageEntryIdExplicit}, uri: nil}
	emptyCopy, err := empty.Dupe(&page, &page)
	zigTry(t, err)
	expect(t, empty.Eql(&emptyCopy), "empty.eql(page.memory, &empty_copy, page.memory)")
}

// Zig: test "term51: PageEntry.dupe: explicit id to another page"
func TestHyperlink_term51_PageEntry_dupe_explicit_id_to_another_page(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()

	// Make dst offsets differ from src offsets.
	_, err := dst.string_alloc.Alloc(40)
	zigTry(t, err)
	filler := hlUsed(&dst)
	expectEqual(t, uint64(64), filler)

	entry := hlEntry(t, &src, hlSpecOf("https://example.com/path", hlExplicit("the-id")))
	src_used := hlUsed(&src)

	cp, err := entry.Dupe(&src, &dst)
	zigTry(t, err)
	expectEqualStrings(t, "https://example.com/path", string(cp.uri))
	expectEqualStrings(t, "the-id", string(cp.id.Explicit))
	expectEqual(t, 24, len(cp.uri))
	expectEqual(t, 6, len(cp.id.Explicit))
	// Zig-only: try testing.expect(copy.uri.offset.offset != entry.uri.offset.offset) (offsets of two pages mean nothing in Go)
	expect(t, &cp.uri[0] != &entry.uri[0], "copy.uri.slice(dst.memory).ptr != entry.uri.slice(src.memory).ptr")

	// One chunk of 32 bytes for each string.
	expectEqual(t, filler+64, hlUsed(&dst))
	// The source is untouched.
	expectEqual(t, src_used, hlUsed(&src))
	expectEqualStrings(t, "https://example.com/path", string(entry.uri))

	// The copy owns its memory: freeing the source does not affect it.
	entry.Free(&src)
	expectEqual(t, uint64(0), hlUsed(&src))
	expectEqualStrings(t, "https://example.com/path", string(cp.uri))
	expectEqualStrings(t, "the-id", string(cp.id.Explicit))
}

// Zig: test "term51: PageEntry.dupe: implicit id to another page"
func TestHyperlink_term51_PageEntry_dupe_implicit_id_to_another_page(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()

	for _, n := range []SizeOffsetInt{0, 1, hlMaxImplicit} {
		func() {
			entry := hlEntry(t, &src, hlSpecOf("https://a.b", hlImplicit(n)))
			defer entry.Free(&src)
			before := hlUsed(&dst)
			cp, err := entry.Dupe(&src, &dst)
			zigTry(t, err)
			defer cp.Free(&dst)
			expectEqual(t, n, cp.id.Implicit)
			expectEqualStrings(t, "https://a.b", string(cp.uri))
			// Only the uri is allocated.
			expectEqual(t, before+32, hlUsed(&dst))
		}()
	}
	expectEqual(t, uint64(0), hlUsed(&dst))
}

// Zig: test "term51: PageEntry.dupe: chunk rounding of string allocation"
func TestHyperlink_term51_PageEntry_dupe_chunk_rounding_of_string_allocation(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()

	// Lengths around the 32 byte chunk boundary.
	lens := []int{1, 31, 32, 33, 64, 65, 2048}
	chunks := []int{1, 1, 1, 2, 2, 3, 64}
	for k := range lens {
		length, n := lens[k], chunks[k]
		func() {
			uri := make([]uint8, length)
			// Zig-only: defer testing.allocator.free(uri)
			for i := range uri {
				uri[i] = uint8('a' + i%26)
			}

			entry := hlEntry(t, &src, hlSpec{uri: uri, id: hlImplicit(1)})
			defer entry.Free(&src)
			cp, err := entry.Dupe(&src, &dst)
			zigTry(t, err)
			defer cp.Free(&dst)
			expectEqualSlices(t, uri, cp.uri)
			expectEqual(t, uint64(n*32), hlUsed(&dst))
		}()
	}
	expectEqual(t, uint64(0), hlUsed(&dst))
}

// Zig: test "term51: PageEntry.dupe: each dupe allocates anew"
func TestHyperlink_term51_PageEntry_dupe_each_dupe_allocates_anew(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()

	entry := hlEntry(t, &src, hlSpecOf("u", hlExplicit("i")))
	a, err := entry.Dupe(&src, &dst)
	zigTry(t, err)
	b, err := entry.Dupe(&src, &dst)
	zigTry(t, err)
	expectEqual(t, uint64(128), hlUsed(&dst))
	expect(t, &a.uri[0] != &b.uri[0], "a.uri.offset.offset != b.uri.offset.offset")
	expect(t, &a.id.Explicit[0] != &b.id.Explicit[0], "a.id.explicit.offset.offset != b.id.explicit.offset.offset")
	expect(t, a.Eql(&b), "a.eql(dst.memory, &b, dst.memory)")
}

// Zig: test "term51: PageEntry.dupe: out of memory for the uri"
func TestHyperlink_term51_PageEntry_dupe_out_of_memory_for_the_uri(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	// Zero string bytes: no bitmaps, so every allocation fails.
	dst := hlPage(t, 0, 4)
	defer dst.Deinit()

	entry := hlEntry(t, &src, hlSpecOf("u", hlExplicit("i")))
	_, err := entry.Dupe(&src, &dst)
	hlExpectError(t, ErrOutOfMemory, err)
	expectEqual(t, uint64(0), hlUsed(&dst))
	expectEqual(t, uint64(0), dst.string_alloc.CapacityBytes())
}

// Zig: test "term51: PageEntry.dupe: out of memory for the explicit id rolls back the uri"
func TestHyperlink_term51_PageEntry_dupe_out_of_memory_for_the_explicit_id_rolls_back_the_uri(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()

	// Leave exactly one free chunk in dst: capacity is 64 chunks.
	hog, err := dst.string_alloc.Alloc(63 * 32)
	zigTry(t, err)
	expectEqual(t, uint64(63*32), hlUsed(&dst))

	// uri takes the last chunk, the id then has no room.
	entry := hlEntry(t, &src, hlSpecOf("u", hlExplicit("i")))
	_, err = entry.Dupe(&src, &dst)
	hlExpectError(t, ErrOutOfMemory, err)
	expectEqual(t, uint64(63*32), hlUsed(&dst))

	// The implicit variant of the same link fits.
	imp := hlEntry(t, &src, hlSpecOf("u", hlImplicit(1)))
	cp, err := imp.Dupe(&src, &dst)
	zigTry(t, err)
	expectEqual(t, uint64(64*32), hlUsed(&dst))
	cp.Free(&dst)
	dst.string_alloc.Free(hog)
	expectEqual(t, uint64(0), hlUsed(&dst))
}

// Zig: test "term51: PageEntry.dupe: very long uri boundaries"
func TestHyperlink_term51_PageEntry_dupe_very_long_uri_boundaries(t *testing.T) {
	// Requests of more than 64 chunks (2048 bytes) take a separate path
	// of the bitmap allocator that is only safe with room to spare; here
	// the 2049 byte uri goes to a destination with two bitmaps.
	src := hlPage(t, 4096, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()
	dst2 := hlPage(t, 4096, 4)
	defer dst2.Deinit()

	uri := make([]uint8, 2049)
	// Zig-only: defer testing.allocator.free(uri)
	for i := range uri {
		uri[i] = uint8('a' + i%26)
	}

	// Exactly the whole capacity fits.
	func() {
		entry := hlEntry(t, &src, hlSpec{uri: uri[0:2048], id: hlImplicit(1)})
		defer entry.Free(&src)
		cp, err := entry.Dupe(&src, &dst)
		zigTry(t, err)
		expectEqual(t, uint64(2048), hlUsed(&dst))
		expectEqual(t, 2048, len(cp.uri))
		expectEqualSlices(t, uri[0:2048], cp.uri)

		// Now it is full, so even one byte more does not fit.
		small := hlEntry(t, &src, hlSpecOf("u", hlImplicit(1)))
		defer small.Free(&src)
		_, err = small.Dupe(&src, &dst)
		hlExpectError(t, ErrOutOfMemory, err)
		expectEqual(t, uint64(2048), hlUsed(&dst))
		cp.Free(&dst)
		expectEqual(t, uint64(0), hlUsed(&dst))
	}()

	// 2049 bytes take 65 chunks.
	func() {
		entry := hlEntry(t, &src, hlSpec{uri: uri, id: hlExplicit("id")})
		defer entry.Free(&src)
		cp, err := entry.Dupe(&src, &dst2)
		zigTry(t, err)
		defer cp.Free(&dst2)
		expectEqualSlices(t, uri, cp.uri)
		expectEqual(t, uint64(66*32), hlUsed(&dst2))
	}()
	expectEqual(t, uint64(0), hlUsed(&dst2))
}

// Zig: test "term51: PageEntry.hash: independent of the page memory"
func TestHyperlink_term51_PageEntry_hash_independent_of_the_page_memory(t *testing.T) {
	a := hlPage(t, 2048, 4)
	defer a.Deinit()
	b := hlPage(t, 2048, 4)
	defer b.Deinit()
	_, err := b.string_alloc.Alloc(100)
	zigTry(t, err)

	for _, spec := range hlDistinct {
		ea := hlEntry(t, &a, spec)
		eb := hlEntry(t, &b, spec)
		expectEqual(t, ea.Hash(), ea.Hash())
		expectEqual(t, ea.Hash(), eb.Hash())
	}
	// Offsets really differ between the pages for non-empty specs.
	// Zig-only: the last three lines (ea, eb of hlDistinct[0] and
	// `try testing.expect(ea.uri.offset.offset != eb.uri.offset.offset)`):
	// offsets of two pages mean nothing in Go.
}

// Zig: test "term51: PageEntry.hash: every field and the id kind matter"
func TestHyperlink_term51_PageEntry_hash_every_field_and_the_id_kind_matter(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()

	entries := make([]HyperlinkPageEntry, len(hlDistinct))
	for i, spec := range hlDistinct {
		entries[i] = hlEntry(t, &page, spec)
	}

	for i, x := range entries {
		for _, y := range entries[i+1:] {
			expect(t, x.Hash() != y.Hash(), "x.hash(page.memory) != y.hash(page.memory)")
		}
	}
}

// Zig: test "term51: PageEntry.hash: slice boundary between id and uri"
func TestHyperlink_term51_PageEntry_hash_slice_boundary_between_id_and_uri(t *testing.T) {
	// The hash covers lengths, so moving a byte from the id to the uri
	// changes it.
	page := hlPage(t, 2048, 4)
	defer page.Deinit()
	a := hlEntry(t, &page, hlSpecOf("c", hlExplicit("ab")))
	b := hlEntry(t, &page, hlSpecOf("bc", hlExplicit("a")))
	expect(t, a.Hash() != b.Hash(), "a.hash(page.memory) != b.hash(page.memory)")
}

// Zig: test "term51: PageEntry.eql: same specs across pages"
func TestHyperlink_term51_PageEntry_eql_same_specs_across_pages(t *testing.T) {
	a := hlPage(t, 2048, 4)
	defer a.Deinit()
	b := hlPage(t, 2048, 4)
	defer b.Deinit()
	_, err := b.string_alloc.Alloc(100)
	zigTry(t, err)

	for _, spec := range hlDistinct {
		ea := hlEntry(t, &a, spec)
		eb := hlEntry(t, &b, spec)
		expect(t, ea.Eql(&eb), "ea.eql(a.memory, &eb, b.memory)")
		expect(t, eb.Eql(&ea), "eb.eql(b.memory, &ea, a.memory)")
		expect(t, ea.Eql(&ea), "ea.eql(a.memory, &ea, a.memory)")
	}
}

// Zig: test "term51: PageEntry.eql: every pair of distinct specs is unequal"
func TestHyperlink_term51_PageEntry_eql_every_pair_of_distinct_specs_is_unequal(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()
	other := hlPage(t, 2048, 4)
	defer other.Deinit()

	entries := make([]HyperlinkPageEntry, len(hlDistinct))
	copies := make([]HyperlinkPageEntry, len(hlDistinct))
	for i, spec := range hlDistinct {
		entries[i] = hlEntry(t, &page, spec)
		copies[i] = hlEntry(t, &other, spec)
	}

	for i, x := range entries {
		for j, y := range copies {
			expectEqual(t, i == j, x.Eql(&y))
			expectEqual(t, i == j, y.Eql(&x))
		}
	}
}

// Zig: test "term51: PageEntry.eql: id kind mismatch with equal text"
func TestHyperlink_term51_PageEntry_eql_id_kind_mismatch_with_equal_text(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()
	imp := hlEntry(t, &page, hlSpecOf("u", hlImplicit(0)))
	exp := hlEntry(t, &page, hlSpecOf("u", hlExplicit("0")))
	expect(t, !imp.Eql(&exp), "!imp.eql(page.memory, &exp, page.memory)")
	expect(t, !exp.Eql(&imp), "!exp.eql(page.memory, &imp, page.memory)")
}

// Zig: test "term51: PageEntry.eql: same length differing in one byte"
func TestHyperlink_term51_PageEntry_eql_same_length_differing_in_one_byte(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()
	base := hlEntry(t, &page, hlSpecOf("https://example.com/a", hlExplicit("idA")))
	uri := hlEntry(t, &page, hlSpecOf("https://example.com/b", hlExplicit("idA")))
	id := hlEntry(t, &page, hlSpecOf("https://example.com/a", hlExplicit("idB")))
	same := hlEntry(t, &page, hlSpecOf("https://example.com/a", hlExplicit("idA")))
	expect(t, !base.Eql(&uri), "!base.eql(page.memory, &uri, page.memory)")
	expect(t, !base.Eql(&id), "!base.eql(page.memory, &id, page.memory)")
	expect(t, base.Eql(&same), "base.eql(page.memory, &same, page.memory)")
	// Same strings, different offsets inside one page.
	expect(t, &base.uri[0] != &same.uri[0], "base.uri.offset.offset != same.uri.offset.offset")
}

// Zig: test "term51: PageEntry.free: releases strings back to the page"
func TestHyperlink_term51_PageEntry_free_releases_strings_back_to_the_page(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()

	exp := hlEntry(t, &page, hlSpecOf("https://a.b", hlExplicit("id")))
	imp := hlEntry(t, &page, hlSpecOf("https://a.b", hlImplicit(2)))
	expectEqual(t, uint64(96), hlUsed(&page))

	// Implicit ids own only the uri.
	imp.Free(&page)
	expectEqual(t, uint64(64), hlUsed(&page))

	// Explicit ids own both.
	exp.Free(&page)
	expectEqual(t, uint64(0), hlUsed(&page))

	// Freed chunks can be allocated again.
	again := hlEntry(t, &page, hlSpecOf("https://a.b", hlExplicit("id")))
	expectEqual(t, uint64(64), hlUsed(&page))
	again.Free(&page)
}

// Zig: test "term51: hash: set context uses the source page when present"
func TestHyperlink_term51_hash_set_context_uses_the_source_page_when_present(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()
	_, err := dst.string_alloc.Alloc(100)
	zigTry(t, err)

	spec := hlSpecOf("https://a.b", hlExplicit("id"))
	in_src := hlEntry(t, &src, spec)
	in_dst := hlEntry(t, &dst, spec)

	plain := HyperlinkSetContext{page: &dst}
	with_src := HyperlinkSetContext{page: &dst, src_page: &src}

	expectEqual(t, in_dst.Hash(), plain.Hash(in_dst))
	expectEqual(t, in_src.Hash(), with_src.Hash(in_src))
	expectEqual(t, plain.Hash(in_dst), with_src.Hash(in_src))
}

// Zig: test "term51: eql: set context compares a against src_page and b against page"
func TestHyperlink_term51_eql_set_context_compares_a_against_src_page_and_b_against_page(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()
	_, err := dst.string_alloc.Alloc(100)
	zigTry(t, err)

	spec := hlSpecOf("https://a.b", hlExplicit("id"))
	diff := hlSpecOf("https://a.c", hlExplicit("id"))
	in_src := hlEntry(t, &src, spec)
	in_dst := hlEntry(t, &dst, spec)
	other_dst := hlEntry(t, &dst, diff)

	plain := HyperlinkSetContext{page: &dst}
	with_src := HyperlinkSetContext{page: &dst, src_page: &src}

	expect(t, plain.Eql(in_dst, in_dst), "plain.eql(in_dst, in_dst)")
	expect(t, !plain.Eql(in_dst, other_dst), "!plain.eql(in_dst, other_dst)")
	expect(t, with_src.Eql(in_src, in_dst), "with_src.eql(in_src, in_dst)")
	expect(t, !with_src.Eql(in_src, other_dst), "!with_src.eql(in_src, other_dst)")
}

// Zig: test "term51: deleted: set context frees the entry strings from its page"
func TestHyperlink_term51_deleted_set_context_frees_the_entry_strings_from_its_page(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()

	ctx := HyperlinkSetContext{page: &page}
	exp := hlEntry(t, &page, hlSpecOf("https://a.b", hlExplicit("id")))
	expectEqual(t, uint64(64), hlUsed(&page))
	ctx.Deleted(exp)
	expectEqual(t, uint64(0), hlUsed(&page))

	imp := hlEntry(t, &page, hlSpecOf("https://a.b", hlImplicit(9)))
	expectEqual(t, uint64(32), hlUsed(&page))
	ctx.Deleted(imp)
	expectEqual(t, uint64(0), hlUsed(&page))
}

// Zig: test "term51: deleted: set layout and id space"
func TestHyperlink_term51_deleted_set_layout_and_id_space(t *testing.T) {
	// Zig compares types (`Id == u16`); the Go counterpart compares the
	// reflected types. `Offset(Cell)` is SizeOffsetInt in Go (the Offset type
	// is omitted); `Set.Id` is the type parameter of HyperlinkSet, seen
	// through the element type of its `table`.
	expect(t, reflect.TypeOf(HyperlinkID(0)) == reflect.TypeOf(uint16(0)), "Id == u16")
	expect(t, reflect.TypeOf(HyperlinkID(0)) == reflect.TypeOf(SizeHyperlinkCountInt(0)), "Id == size.HyperlinkCountInt")
	expect(t, reflect.TypeOf(HyperlinkSet{}.table).Elem() == reflect.TypeOf(HyperlinkID(0)), "Set.Id == Id")
	expect(t, reflect.TypeOf(HyperlinkMap{}) == reflect.TypeOf(HashMapAutoOffsetHashMap[SizeOffsetInt, uint16]{}), "Map == AutoOffsetHashMap(Offset(Cell), u16)")
	expect(t, reflect.TypeOf(SizeOffsetInt(0)) == reflect.TypeOf(uint32(0)), "size.OffsetInt == u32")

	cases := []struct{ req, cap uint }{
		{req: 0, cap: 0},
		{req: 1, cap: 0},
		{req: 2, cap: 1},
		{req: 4, cap: 3},
		{req: 8, cap: 6},
		{req: 1365, cap: 1664},
		{req: 65535, cap: 53248},
	}
	for _, c := range cases {
		l := RefCountedSetLayoutInit[HyperlinkID](c.req, hyperlinkSetItemSizes)
		expectEqual(t, c.cap, l.cap)
	}
}

// Zig: test "term51: deleted: insertHyperlink dedupes equal links and frees the duplicate strings"
func TestHyperlink_term51_deleted_insertHyperlink_dedupes_equal_links_and_frees_the_duplicate_strings(t *testing.T) {
	page := hlPage(t, 2048, 8)
	defer page.Deinit()

	link := Hyperlink{ID: hlExplicit("id"), URI: []uint8("https://a.b")}
	id1, err := page.InsertHyperlink(link)
	zigTry(t, err)
	expectEqual(t, HyperlinkID(1), id1)
	expectEqual(t, uint64(64), hlUsed(&page))

	// The same link: same id, ref bumped, and the strings that were
	// allocated for the duplicate are released by the context.
	id2, err := page.InsertHyperlink(link)
	zigTry(t, err)
	expectEqual(t, id1, id2)
	expectEqual(t, uint16(2), page.hyperlink_set.RefCount(id1))
	expectEqual(t, uint64(64), hlUsed(&page))
	expectEqual(t, uint(1), page.hyperlink_set.Count())
}

// Zig: test "term51: deleted: links differing in one field get different ids"
func TestHyperlink_term51_deleted_links_differing_in_one_field_get_different_ids(t *testing.T) {
	page := hlPage(t, 2048, 16)
	defer page.Deinit()

	ids := make([]HyperlinkID, len(hlDistinct)-3)
	n := 0
	// The first entries only (those with non-empty uris), since the page
	// string allocator cannot allocate empty strings.
	for _, spec := range hlDistinct {
		if len(spec.uri) == 0 {
			continue
		}
		if spec.id.Tag == HyperlinkHyperlinkIdExplicit && len(spec.id.Explicit) == 0 {
			continue
		}
		link := Hyperlink{ID: spec.id, URI: spec.uri}
		id, err := page.InsertHyperlink(link)
		zigTry(t, err)
		ids[n] = id
		expectEqual(t, HyperlinkID(n+1), ids[n])
		n += 1
	}
	expectEqual(t, 9, n)
	expectEqual(t, uint(9), page.hyperlink_set.Count())

	// Inserting every one again finds the same ids.
	n = 0
	for _, spec := range hlDistinct {
		if len(spec.uri) == 0 {
			continue
		}
		if spec.id.Tag == HyperlinkHyperlinkIdExplicit && len(spec.id.Explicit) == 0 {
			continue
		}
		link := Hyperlink{ID: spec.id, URI: spec.uri}
		id, err := page.InsertHyperlink(link)
		zigTry(t, err)
		expectEqual(t, ids[n], id)
		expectEqual(t, uint16(2), page.hyperlink_set.RefCount(ids[n]))
		n += 1
	}
}

// Zig: test "term51: deleted: implicit id counter values"
func TestHyperlink_term51_deleted_implicit_id_counter_values(t *testing.T) {
	page := hlPage(t, 2048, 8)
	defer page.Deinit()

	vals := []SizeOffsetInt{0, 1, 2, hlMaxImplicit - 1, hlMaxImplicit}
	for i, v := range vals {
		id, err := page.InsertHyperlink(Hyperlink{ID: hlImplicit(v), URI: []uint8("https://a.b")})
		zigTry(t, err)
		expectEqual(t, HyperlinkID(i+1), id)
		entry := page.hyperlink_set.Get(id)
		expectEqual(t, v, entry.id.Implicit)
	}
	// Same counter and same uri dedupe.
	id, err := page.InsertHyperlink(Hyperlink{ID: hlImplicit(hlMaxImplicit), URI: []uint8("https://a.b")})
	zigTry(t, err)
	expectEqual(t, HyperlinkID(5), id)
}

// Zig: test "term51: deleted: set full reports SetOutOfMemory and frees strings"
func TestHyperlink_term51_deleted_set_full_reports_SetOutOfMemory_and_frees_strings(t *testing.T) {
	// 4 requested items: table 4, cap 3, so two usable ids.
	page := hlPage(t, 2048, 4)
	defer page.Deinit()

	_, err := page.InsertHyperlink(Hyperlink{ID: hlImplicit(1), URI: []uint8("https://a.b")})
	zigTry(t, err)
	_, err = page.InsertHyperlink(Hyperlink{ID: hlExplicit("x"), URI: []uint8("https://c.d")})
	zigTry(t, err)
	used := hlUsed(&page)
	expectEqual(t, uint64(96), used)

	_, err = page.InsertHyperlink(Hyperlink{ID: hlExplicit("y"), URI: []uint8("https://e.f")})
	hlExpectError(t, ErrSetOutOfMemory, err)
	// The uri and id strings of the failed insert were released.
	expectEqual(t, used, hlUsed(&page))
	expectEqual(t, uint(2), page.hyperlink_set.Count())

	// Re-inserting an existing link still works at capacity.
	id, err := page.InsertHyperlink(Hyperlink{ID: hlImplicit(1), URI: []uint8("https://a.b")})
	zigTry(t, err)
	expectEqual(t, HyperlinkID(1), id)
}

// Zig: test "term51: deleted: set with dead items reports SetNeedsRehash"
func TestHyperlink_term51_deleted_set_with_dead_items_reports_SetNeedsRehash(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()

	a, err := page.InsertHyperlink(Hyperlink{ID: hlImplicit(1), URI: []uint8("https://a.b")})
	zigTry(t, err)
	_, err = page.InsertHyperlink(Hyperlink{ID: hlImplicit(2), URI: []uint8("https://c.d")})
	zigTry(t, err)
	page.hyperlink_set.Release(a)

	used := hlUsed(&page)
	_, err = page.InsertHyperlink(Hyperlink{ID: hlImplicit(3), URI: []uint8("https://e.f")})
	hlExpectError(t, ErrSetNeedsRehash, err)
	expectEqual(t, used, hlUsed(&page))
}

// Zig: test "term51: deleted: dead entry is reaped when its slot is reused"
func TestHyperlink_term51_deleted_dead_entry_is_reaped_when_its_slot_is_reused(t *testing.T) {
	page := hlPage(t, 2048, 8)
	defer page.Deinit()

	link := Hyperlink{ID: hlExplicit("id"), URI: []uint8("https://a.b")}
	id, err := page.InsertHyperlink(link)
	zigTry(t, err)
	other, err := page.InsertHyperlink(Hyperlink{ID: hlImplicit(1), URI: []uint8("https://c.d")})
	zigTry(t, err)
	expectEqual(t, uint64(96), hlUsed(&page))

	// A dead entry keeps its strings until something overwrites it.
	page.hyperlink_set.Release(id)
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	expectEqual(t, uint64(96), hlUsed(&page))

	// Re-adding the same link resurrects the same id; the old strings are
	// deleted (via the context) and the new ones remain.
	again, err := page.InsertHyperlink(link)
	zigTry(t, err)
	expectEqual(t, id, again)
	expectEqual(t, uint16(1), page.hyperlink_set.RefCount(id))
	expectEqual(t, uint64(96), hlUsed(&page))
	expectEqual(t, uint16(1), page.hyperlink_set.RefCount(other))
}

// Zig: test "term51: deleted: string capacity limits of insertHyperlink"
func TestHyperlink_term51_deleted_string_capacity_limits_of_insertHyperlink(t *testing.T) {
	big := make([]uint8, 2048)
	// Zig-only: defer testing.allocator.free(big)
	for i := range big {
		big[i] = 'u'
	}

	// A uri of the whole capacity (64 chunks) fits; nothing else does.
	func() {
		page := hlPage(t, 2048, 8)
		defer page.Deinit()
		id, err := page.InsertHyperlink(Hyperlink{ID: hlImplicit(0), URI: big})
		zigTry(t, err)
		expectEqual(t, uint64(2048), hlUsed(&page))
		expectEqual(t, 2048, len(page.hyperlink_set.Get(id).uri))
		_, err = page.InsertHyperlink(Hyperlink{ID: hlImplicit(1), URI: []uint8("u")})
		hlExpectError(t, ErrStringsOutOfMemory, err)
		expectEqual(t, uint64(2048), hlUsed(&page))
		expectEqual(t, uint(1), page.hyperlink_set.Count())
	}()

	// The explicit id does not fit after the uri: the uri is rolled back.
	func() {
		page := hlPage(t, 2048, 8)
		defer page.Deinit()
		hog, err := page.string_alloc.Alloc(63 * 32)
		zigTry(t, err)
		_, err = page.InsertHyperlink(Hyperlink{ID: hlExplicit("x"), URI: []uint8("u")})
		hlExpectError(t, ErrStringsOutOfMemory, err)
		expectEqual(t, uint64(63*32), hlUsed(&page))
		expectEqual(t, uint(0), page.hyperlink_set.Count())

		// Without an explicit id the link fits in the last chunk.
		_, err = page.InsertHyperlink(Hyperlink{ID: hlImplicit(0), URI: []uint8("u")})
		zigTry(t, err)
		expectEqual(t, uint64(2048), hlUsed(&page))
		page.string_alloc.Free(hog)
	}()

	// A uri and an explicit id each taking half of the capacity fit.
	func() {
		page := hlPage(t, 2048, 8)
		defer page.Deinit()
		_, err := page.InsertHyperlink(Hyperlink{
			ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: big[0:1024]},
			URI: big[0:1024],
		})
		zigTry(t, err)
		expectEqual(t, uint64(2048), hlUsed(&page))
	}()
}

// Zig: test "term51: deleted: lookup finds a link stored in another page via src_page"
func TestHyperlink_term51_deleted_lookup_finds_a_link_stored_in_another_page_via_src_page(t *testing.T) {
	src := hlPage(t, 2048, 8)
	defer src.Deinit()
	dst := hlPage(t, 2048, 8)
	defer dst.Deinit()

	link := Hyperlink{ID: hlExplicit("id"), URI: []uint8("https://a.b")}
	dst_id, err := dst.InsertHyperlink(link)
	zigTry(t, err)
	src_id, err := src.InsertHyperlink(link)
	zigTry(t, err)
	src_entry := *src.hyperlink_set.Get(src_id)

	found := dst.hyperlink_set.LookupContext(
		src_entry,
		HyperlinkSetContext{page: &dst, src_page: &src},
	)
	expectEqual(t, Some(dst_id), found)

	// A different link is not found.
	other_id, err := src.InsertHyperlink(Hyperlink{ID: hlExplicit("id"), URI: []uint8("https://other")})
	zigTry(t, err)
	other := *src.hyperlink_set.Get(other_id)
	expectEqual(t, None[HyperlinkID](), dst.hyperlink_set.LookupContext(
		other,
		HyperlinkSetContext{page: &dst, src_page: &src},
	))

	// Clone it over with dupe + addWithIdContext, as Page.cloneFrom does.
	cp, err := other.Dupe(&src, &dst)
	zigTry(t, err)
	added, err := dst.hyperlink_set.AddWithIdContext(cp, other_id, HyperlinkSetContext{page: &dst})
	zigTry(t, err)
	// The destination had no such id yet, so a plain add assigned it and
	// the result is the id itself, not null.
	expectEqual(t, Some(other_id), added)
	expectEqualStrings(t,
		"https://other",
		string(dst.hyperlink_set.Get(other_id).uri),
	)
}

// Not translated (only produces data/sets/hyperlink_hash.txt, gated by TERM51_SET_DUMP_DIR): see scripts/not-translated.txt.
//   Zig: test "term51: PageEntry.hash: dump exact data when TERM51_SET_DUMP_DIR is set"

// Go-only: panic test: PageEntry.dupe: empty uri to another page: string_alloc.alloc asserts n > 0
func TestHyperlink_panic_PageEntry_dupe_empty_uri_to_another_page(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()

	entry := hlEntry(t, &src, hlSpecOf("", hlImplicit(1)))
	hlExpectPanic(t, func() { _, _ = entry.Dupe(&src, &dst) })
}

// Go-only: panic test: PageEntry.dupe: empty explicit id to another page: string_alloc.alloc asserts n > 0
func TestHyperlink_panic_PageEntry_dupe_empty_explicit_id_to_another_page(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()
	dst := hlPage(t, 2048, 4)
	defer dst.Deinit()

	entry := hlEntry(t, &src, hlSpecOf("u", hlExplicit("")))
	hlExpectPanic(t, func() { _, _ = entry.Dupe(&src, &dst) })
}

// Go-only: panic test: PageEntry.free: empty uri: BitmapAllocator.free of a slice that is not from the allocator
func TestHyperlink_panic_PageEntry_free_empty_uri(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()

	// The zero slice `.{}` does not point into the string allocator; Zig's
	// free computes the chunk index from the address and underflows.
	entry := hlEntry(t, &page, hlSpecOf("", hlImplicit(1)))
	hlExpectPanic(t, func() { entry.Free(&page) })
}

// Go-only: panic test: PageEntry.free: empty explicit id: BitmapAllocator.free of a slice that is not from the allocator
func TestHyperlink_panic_PageEntry_free_empty_explicit_id(t *testing.T) {
	page := hlPage(t, 2048, 4)
	defer page.Deinit()

	entry := hlEntry(t, &page, hlSpecOf("u", hlExplicit("")))
	hlExpectPanic(t, func() { entry.Free(&page) })
}

// Go-only: panic test: Set.Context.hash: neither page nor src_page is set (page.? on null)
func TestHyperlink_panic_SetContext_hash_without_page(t *testing.T) {
	hlExpectPanic(t, func() { _ = HyperlinkSetContext{}.Hash(HyperlinkPageEntry{}) })
}

// Go-only: panic test: Set.Context.eql: page is not set (page.? on null), even with src_page
func TestHyperlink_panic_SetContext_eql_without_page(t *testing.T) {
	src := hlPage(t, 2048, 4)
	defer src.Deinit()

	ctx := HyperlinkSetContext{src_page: &src}
	hlExpectPanic(t, func() { _ = ctx.Eql(HyperlinkPageEntry{}, HyperlinkPageEntry{}) })
}

// Go-only: panic test: Set.Context.deleted: page is not set (page.? on null)
func TestHyperlink_panic_SetContext_deleted_without_page(t *testing.T) {
	hlExpectPanic(t, func() { HyperlinkSetContext{}.Deleted(HyperlinkPageEntry{}) })
}
