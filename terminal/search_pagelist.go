// WIREFRAME for src/terminal/search/pagelist.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: `search/pagelist.zig` is `search_pagelist.go`, prefix
// `SearchPagelist`. The file's one declaration, `PageListSearch`, is the
// type search.zig re-exports as `search.PageList`, so it is named by the
// bare file prefix: `SearchPagelist` (the rule for the main type of every
// search file; see search.go). Note the lower-case `l`: the file is
// `pagelist.zig`. search.go declares the alias `SearchPageList` for
// search.zig's `PageList`. Its `init` is `SearchPagelistInit`; its `pub fn`
// methods are exported (`Deinit`, `Next`, `Feed`).
//
// Declarations appear in the same order as in pagelist.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) The allocator parameter of `init` is
// dropped; `Deinit` is kept (every test calls it; it also untracks `pin`).
// `list: *PageList` is a Go pointer; `pin: *Pin` is a tracked
// `*PageListPin` owned by `list` (PageList.TrackPin). Node pointers are
// `*PageListNode`. `?FlattenedHighlight` is `Optional[HighlightFlattened]`.
//
// # Widths
//
// The local `rem` in `feed` is a count (`usize`), `uint`, decreased with
// Zig's saturating `-|=`.
//
// # Omitted
//
// Imports and aliases with nothing to declare: `std`, `Allocator`,
// `testing`, `terminal`, `point`, `FlattenedHighlight` (use
// `HighlightFlattened`), `Page`, `PageList`, `Pin` (use `PageListPin`),
// `Selection`, `Screen`, `Terminal`, `SlidingWindow` (use
// `SearchSlidingWindow`). No function is omitted. No C ABI declaration
// exists in this file.
//
// # Zig defaults a Go literal does not give
//
// None: `PageListSearch` has no field defaults. It is made by its `init`,
// which tracks a pin, makes a REVERSE sliding window and appends `start` to
// it; use SearchPagelistInit.

package terminal

// SearchPagelist is pagelist.PageListSearch (re-exported as
// search.PageList).
//
// Searches for a term in a PageList structure.
//
// This searches in reverse order starting from the given node.
//
// This assumes that nodes do not change contents. For nodes that change
// contents, look at ActiveSearch, which is designed to re-search the active
// area since it assumed to change. When integrating ActiveSearch with
// PageListSearch, the caller should start the PageListSearch from the
// returned node from ActiveSearch.update().
//
// Concurrent access to a PageList or nodes in a PageList are not allowed,
// so the caller should ensure that necessary locks are held. Each function
// documents whether it accesses the PageList or not. For example, you can
// safely call `next()` without holding a lock, but you must hold a lock
// while calling `feed()`.
type SearchPagelist struct {
	// The list we're searching.
	list *PageList

	// The sliding window of page contents and nodes to search.
	window SearchSlidingWindow

	// The tracked pin for our current position in the pagelist. This
	// will always point to the CURRENT node we're searching from so that
	// we can track if we move.
	pin *PageListPin
}

// SearchPagelistInit is PageListSearch.init.
//
// Initialize the page list search. The needle is copied so it can
// be freed immediately.
//
// Accesses the PageList/Node so the caller must ensure it is safe
// to do so if there is any concurrent access.
//
// Zig: pub fn init(alloc: Allocator, needle: []const u8, list: *PageList,
// start: *PageList.List.Node) Allocator.Error!PageListSearch. The allocator
// parameter is dropped. The error is ErrOutOfMemory (dcs.go).
func SearchPagelistInit(
	needle []uint8,
	list *PageList,
	start *PageListNode,
) (SearchPagelist, error) {
	panic(notImplemented)
}

// Deinit is PageListSearch.deinit.
//
// Modifies the PageList (to untrack a pin) so the caller must ensure
// that it is safe to do so.
func (self *SearchPagelist) Deinit() { panic(notImplemented) }

// Next is PageListSearch.next.
//
// Return the next match in the loaded page nodes. If this returns
// null then the PageList search needs to be fed the next node(s).
// Call, `feed` to do this.
//
// Beware that the selection returned may point to a node that
// is freed if the caller does not hold necessary locks on the
// PageList while searching. The pins should be validated prior to
// final use.
//
// This does NOT access the PageList, so it can be called without
// a lock held.
//
// Zig: ?FlattenedHighlight. The result shares the window's memory; see
// SearchSlidingWindow.Next.
func (self *SearchPagelist) Next() Optional[HighlightFlattened] { panic(notImplemented) }

// Feed is PageListSearch.feed.
//
// Feed more data to the sliding window from the pagelist. This will
// feed enough data to cover at least one match (needle length) if it
// exists; this doesn't perform the search, it only feeds data.
//
// This accesses nodes in the PageList, so the caller must ensure
// it is safe to do so (i.e. hold necessary locks).
//
// This returns false if there is no more data to feed. This essentially
// means we've searched the entire pagelist.
//
// Zig: Allocator.Error!bool. The error is ErrOutOfMemory (dcs.go).
func (self *SearchPagelist) Feed() (bool, error) { panic(notImplemented) }
