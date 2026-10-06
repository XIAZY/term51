// Port of src/terminal/search/pagelist.zig (Ghostty v1.3.1).
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
) (_ SearchPagelist, err error) {
	// We put a tracked pin into the node that we're starting from.
	// By using a tracked pin, we can keep our pagelist references safe
	// because if the pagelist prunes pages, the tracked pin will
	// be moved somewhere safe.
	assert(start.data.size.rows > 0) // Zig: u16 subtraction traps
	assert(start.data.size.cols > 0) // Zig: u16 subtraction traps
	pin, err := list.TrackPin(PageListPin{
		node: start,
		y:    start.data.size.rows - 1,
		x:    start.data.size.cols - 1,
	})
	if err != nil {
		return SearchPagelist{}, err
	}
	defer func() {
		// errdefer
		if err != nil {
			list.UntrackPin(pin)
		}
	}()

	// Create our sliding window we'll use for searching.
	window, err := SearchSlidingWindowInit(searchSlidingWindowDirectionReverse, needle)
	if err != nil {
		return SearchPagelist{}, err
	}
	// Zig: errdefer window.deinit(); Go memory.

	// We always feed our initial page data into the window, because
	// we have the lock anyways and this lets our `pin` point to our
	// current node and feed to work properly.
	if _, err = window.Append(start); err != nil {
		return SearchPagelist{}, err
	}

	return SearchPagelist{
		list:   list,
		window: window,
		pin:    pin,
	}, nil
}

// Deinit is PageListSearch.deinit.
//
// Modifies the PageList (to untrack a pin) so the caller must ensure
// that it is safe to do so.
func (self *SearchPagelist) Deinit() {
	self.window.Deinit()
	self.list.UntrackPin(self.pin)
}

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
func (self *SearchPagelist) Next() Optional[HighlightFlattened] {
	return self.window.Next()
}

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
func (self *SearchPagelist) Feed() (bool, error) {
	// If our pin becomes garbage it means wherever we were next
	// was reused and we can't make sense of our progress anymore.
	// It is effectively equivalent to reaching the end of the PageList.
	if self.pin.garbage {
		return false, nil
	}

	// Add at least enough data to find a single match.
	rem := uint(len(self.window.needle))

	// Start at our previous node and then continue adding until we
	// get our desired amount of data.
	for node := self.pin.node.prev; node != nil; node = node.prev {
		added, err := self.window.Append(node)
		if err != nil {
			return false, err
		}
		// Zig: rem -|= added
		if added >= rem {
			rem = 0
		} else {
			rem -= added
		}

		// Move our tracked pin to the new node.
		self.pin.node = node

		if rem == 0 {
			break
		}
	}

	// True if we fed any data.
	return rem < uint(len(self.window.needle)), nil
}
