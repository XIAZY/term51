// Port of src/terminal/search/active.zig (Ghostty v1.3.1).
//
// File naming: `search/active.zig` is `search_active.go`, prefix
// `SearchActive`. The file's one declaration, `ActiveSearch`, is the type
// search.zig re-exports as `search.Active`, so it is named by the bare file
// prefix: `SearchActive` (the rule for the main type of every search file;
// see search.go). Its `init` is `SearchActiveInit`; its `pub fn` methods are
// exported (`Deinit`, `Update`, `Next`).
//
// Declarations appear in the same order as in active.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) The allocator parameter of `init` is
// dropped; `Deinit` is kept (every test calls it). `*const PageList` is
// `*PageList`; `?*PageList.List.Node` is `*PageListNode`, nil for null.
// `?FlattenedHighlight` is `Optional[HighlightFlattened]`.
//
// # Widths
//
// The local `rem` in `update` is a count (`usize`), `uint`.
//
// # Empty needle (project lead's ruling)
//
// The overlap loop of `update` computes `self.window.needle.len - 1`
// (active.zig:94), which traps in a safe build when the needle is empty and
// a wrapped page precedes the active area; the release build wraps to
// maxInt(usize) and appends every wrapped previous page, which is what the
// corpus records (`empty_active_update`, case `pages3_wrap`,
// `unchecked_only empty_needle`). The port follows the release build: the
// uint subtraction wraps (to maxInt(uint), which no page's byte count
// reaches on either width). `next` with an empty needle finds nothing (see
// search_sliding_window.go).
//
// # Omitted
//
// Imports and aliases with nothing to declare: `std`, `testing`,
// `Allocator`, `point`, `size`, `FlattenedHighlight` (use
// `HighlightFlattened`), `PageList`, `SlidingWindow` (use
// `SearchSlidingWindow`), `Terminal`. No function is omitted. No C ABI
// declaration exists in this file.
//
// # Zig defaults a Go literal does not give
//
// None: `ActiveSearch` has no field defaults. It is made by its `init`,
// which makes a FORWARD sliding window (SearchSlidingWindowInit), which a
// literal cannot; use SearchActiveInit.

package terminal

// SearchActive is active.ActiveSearch (re-exported as search.Active).
//
// Searches for a substring within the active area of a PageList.
//
// The distinction for "active area" is important because it is the
// only part of a PageList that is mutable. Therefore, its the only part
// of the terminal that needs to be repeatedly searched as the contents
// change.
//
// This struct specializes in searching only within that active area,
// and handling the active area moving as new lines are added to the bottom.
type SearchActive struct {
	window SearchSlidingWindow
}

// SearchActiveInit is ActiveSearch.init.
//
// Zig: pub fn init(alloc: Allocator, needle: []const u8)
// Allocator.Error!ActiveSearch. The allocator parameter is dropped. The
// error is ErrOutOfMemory (dcs.go).
func SearchActiveInit(needle []uint8) (SearchActive, error) {
	// We just do a forward search since the active area is usually
	// pretty small so search results are instant anyways. This avoids
	// a small amount of work to reverse things.
	window, err := SearchSlidingWindowInit(searchSlidingWindowDirectionForward, needle)
	if err != nil {
		return SearchActive{}, err
	}
	// Zig: errdefer window.deinit(); Go memory.
	return SearchActive{window: window}, nil
}

// Deinit is ActiveSearch.deinit.
//
// Kept as a stub: every test calls it.
func (self *SearchActive) Deinit() {
	self.window.Deinit()
}

// Update is ActiveSearch.update.
//
// Update the active area to reflect the current state of the PageList.
//
// This doesn't do the search, it only copies the necessary data
// to perform the search later. This lets the caller hold the lock
// on the PageList for a minimal amount of time.
//
// This returns the first page (in reverse order) covered by this
// search. This allows the history search to overlap and search history.
// There CAN BE duplicates, and this page CAN BE mutable, so the history
// search results should prune anything that's in the active area.
//
// If the return value is null it means the active area covers the entire
// PageList, currently.
//
// Zig: pub fn update(self: *ActiveSearch, list: *const PageList)
// Allocator.Error!?*PageList.List.Node. nil is null. The error is
// ErrOutOfMemory (dcs.go).
func (self *SearchActive) Update(list *PageList) (*PageListNode, error) {
	// Clear our previous sliding window
	self.window.ClearAndRetainCapacity()

	// First up, add enough pages to cover the active area.
	rem := uint(list.rows)
	node_ := list.pages.last
	var last_node *PageListNode = nil
	for ; node_ != nil; node_ = node_.prev {
		node := node_
		if _, err := self.window.Append(node); err != nil {
			return nil, err
		}
		last_node = node

		// If we reached our target amount, then this is the last
		// page that contains the active area. We go to the previous
		// page once more since its the first page of our required
		// overlap.
		if rem <= uint(node.data.size.rows) {
			node_ = node.prev
			break
		}

		rem -= uint(node.data.size.rows)
	}

	// Next, add enough overlap to cover needle.len - 1 bytes (if it
	// exists) so we can cover the overlap.
	for ; node_ != nil; node_ = node_.prev {
		node := node_
		// If the last row of this node isn't wrapped we can't overlap.
		assert(node.data.size.rows > 0) // Zig: u16 subtraction traps
		row := node.data.GetRow(uint(node.data.size.rows - 1))
		if !row.wrap {
			break
		}

		// We could be more accurate here and count bytes since the
		// last wrap but its complicated and unlikely multiple pages
		// wrap so this should be fine.
		added, err := self.window.Append(node)
		if err != nil {
			return nil, err
		}
		// Zig: `needle.len - 1` traps on an empty needle in a safe build;
		// the release build wraps to maxInt(usize), which the corpus
		// records (project lead's ruling, see the file header). Go's uint
		// wraps the same way.
		if added >= uint(len(self.window.needle))-1 {
			break
		}
	}

	// Return the last node we added to our window.
	return last_node, nil
}

// Next is ActiveSearch.next.
//
// Find the next match for the needle in the active area. This returns
// null when there are no more matches.
//
// Zig: ?FlattenedHighlight. The result shares the window's memory; see
// SearchSlidingWindow.Next.
func (self *SearchActive) Next() Optional[HighlightFlattened] {
	return self.window.Next()
}
