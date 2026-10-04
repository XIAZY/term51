// WIREFRAME for src/terminal/search/viewport.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: `search/viewport.zig` is `search_viewport.go`, prefix
// `SearchViewport` (lowerCamel `searchViewport` for private declarations).
// The file's one declaration, `ViewportSearch`, is the type search.zig
// re-exports as `search.Viewport`, so it is named by the bare file prefix:
// `SearchViewport` (the rule for the main type of every search file; see
// search.go). Its `init` is `SearchViewportInit`; its `pub fn` methods are
// exported (`Deinit`, `Reset`, `Needle`, `Update`, `Next`). The private
// nested `ViewportSearch.Fingerprint` is `searchViewportFingerprint`; its
// `init` is `searchViewportFingerprintInit` and its `pub fn` methods are
// exported (`Deinit`, `Eql`).
//
// Declarations appear in the same order as in viewport.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Every allocator parameter and
// argument is dropped (`init`, `Fingerprint.init`, `Fingerprint.deinit`; the
// window's allocator is no longer read). `Deinit` stubs are kept.
// `Fingerprint.nodes` is `[]const *PageList.List.Node`: a
// `[]*PageListNode`; `std.ArrayList(...).toOwnedSlice` is the Go slice
// itself. Pointer equality of nodes (`a != b`) is Go pointer equality.
// Zig's `if (self.fingerprint) |*fp|` and `if (self.active_dirty) |*v|`
// modify the payload in place: `&self.fingerprint.Value`,
// `self.active_dirty.Value = false`.
//
// # Omitted
//
// Imports and aliases with nothing to declare: `std`, `assert`, `testing`,
// `Allocator`, `point`, `size`, `FlattenedHighlight` (use
// `HighlightFlattened`), `PageList`, `SlidingWindow` (use
// `SearchSlidingWindow`), `Terminal`. No function is omitted. No C ABI
// declaration exists in this file.
//
// # Zig defaults a Go literal does not give
//
// None: no field has a Zig default. `init` sets `fingerprint` and
// `active_dirty` to null, the Go zero value, but also makes a FORWARD
// sliding window, which a literal cannot; use SearchViewportInit. Dirty
// tracking is turned on by a test with `search.active_dirty = Some(false)`
// (Zig `search.active_dirty = false`).

package terminal

// SearchViewport is viewport.ViewportSearch (re-exported as
// search.Viewport).
//
// Searches for a substring within the viewport of a PageList.
//
// This contains logic to efficiently detect when the viewport changes
// and only re-search when necessary.
//
// The specialization for "viewport" is because the viewport is the
// only part of the search where the user can actively see the results,
// usually. In that case, it is more efficient to re-search only the
// viewport rather than store all the results for the entire screen.
//
// Note that this searches all the pages that viewport covers, so
// this can include extra matches outside the viewport if the data
// lives in the same page.
type SearchViewport struct {
	window SearchSlidingWindow

	// Zig: ?Fingerprint.
	fingerprint Optional[searchViewportFingerprint]

	// If this is null, then active dirty tracking is disabled and if the
	// viewport overlaps the active area we always re-search. If this is
	// non-null, then we only re-search if the active area is dirty. Dirty
	// marking is up to the caller.
	//
	// Zig: ?bool.
	active_dirty Optional[bool]
}

// SearchViewportInit is ViewportSearch.init.
//
// Zig: pub fn init(alloc: Allocator, needle_unowned: []const u8)
// Allocator.Error!ViewportSearch. The allocator parameter is dropped. The
// error is ErrOutOfMemory (dcs.go).
func SearchViewportInit(needle_unowned []uint8) (SearchViewport, error) { panic(notImplemented) }

// Deinit is ViewportSearch.deinit.
//
// Kept as a stub: every test calls it.
func (self *SearchViewport) Deinit() { panic(notImplemented) }

// Reset is ViewportSearch.reset.
//
// Reset our fingerprint and results so that the next update will
// always re-search.
func (self *SearchViewport) Reset() { panic(notImplemented) }

// Needle is ViewportSearch.needle.
//
// The needle that this search is using.
//
// Zig: []const u8, the window's own needle (not a copy).
func (self *SearchViewport) Needle() []uint8 { panic(notImplemented) }

// Update is ViewportSearch.update.
//
// Update the sliding window to reflect the current viewport. This
// will do nothing if the viewport hasn't changed since the last
// search.
//
// The PageList must be safe to read throughout the lifetime of this
// function.
//
// Returns true if the viewport changed and a re-search is needed.
// Returns false if the viewport is unchanged.
//
// Zig: Allocator.Error!bool. The error is ErrOutOfMemory (dcs.go).
func (self *SearchViewport) Update(list *PageList) (bool, error) { panic(notImplemented) }

// Next is ViewportSearch.next.
//
// Find the next match for the needle in the active area. This returns
// null when there are no more matches.
//
// Zig: ?FlattenedHighlight. The result shares the window's memory; see
// SearchSlidingWindow.Next.
func (self *SearchViewport) Next() Optional[HighlightFlattened] { panic(notImplemented) }

// searchViewportFingerprint is ViewportSearch.Fingerprint.
//
// Viewport fingerprint so we can detect when the viewport moves.
type searchViewportFingerprint struct {
	// The nodes that make up the viewport. We need to flatten this
	// to a single list because we can't safely traverse the cached values
	// because the page nodes may be invalid. All that is safe is comparing
	// the actual pointer values.
	//
	// Zig: []const *PageList.List.Node.
	nodes []*PageListNode
}

// searchViewportFingerprintInit is ViewportSearch.Fingerprint.init.
//
// Zig: pub fn init(alloc: Allocator, pages: *PageList)
// Allocator.Error!Fingerprint. The allocator parameter is dropped. The
// error is ErrOutOfMemory (dcs.go).
func searchViewportFingerprintInit(pages *PageList) (searchViewportFingerprint, error) {
	panic(notImplemented)
}

// Deinit is ViewportSearch.Fingerprint.deinit.
//
// Zig: pub fn deinit(self: *Fingerprint, alloc: Allocator) void. The
// allocator parameter is dropped. It only frees; the stub is kept for the
// callers.
func (self *searchViewportFingerprint) Deinit() { panic(notImplemented) }

// Eql is ViewportSearch.Fingerprint.eql.
func (self searchViewportFingerprint) Eql(other searchViewportFingerprint) bool {
	panic(notImplemented)
}
