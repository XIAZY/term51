// Port of src/terminal/search/viewport.zig (Ghostty v1.3.1).
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
// # Empty needle (project lead's ruling)
//
// The leading and trailing overlap loops of `update` compute
// `self.window.needle.len - 1` (viewport.zig:165 and the trailing loop),
// which traps in a safe build when the needle is empty and wrapped pages
// border the viewport; the release build wraps to maxInt(usize), which is
// what the corpus records (`empty_viewport_update`, case `pages3_wrap`,
// `unchecked_only empty_needle`). The port follows the release build: the
// uint subtraction wraps. `next` with an empty needle finds nothing (see
// search_sliding_window.go).
//
// # Control flow
//
// `if (old.eql(fingerprint)) match: { ... }` is a closure whose `false`
// result is `break :match` (re-search) and whose `true` result is the end of
// the block (no change); the `active:` block is a closure returning the bool.
// The `errdefer` that resets `self.fingerprint` changes state, so it is a
// deferred closure on the named error result (the PageList idiom); the
// `trailing:` block is a closure that sets that error.
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

import "math"

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
func SearchViewportInit(needle_unowned []uint8) (SearchViewport, error) {
	// We just do a forward search since the viewport is usually
	// pretty small so search results are instant anyways. This avoids
	// a small amount of work to reverse things.
	window, err := SearchSlidingWindowInit(searchSlidingWindowDirectionForward, needle_unowned)
	if err != nil {
		return SearchViewport{}, err
	}
	// Zig: errdefer window.deinit(); Go memory.
	return SearchViewport{
		window:       window,
		fingerprint:  None[searchViewportFingerprint](),
		active_dirty: None[bool](),
	}, nil
}

// Deinit is ViewportSearch.deinit.
//
// Kept as a stub: every test calls it.
func (self *SearchViewport) Deinit() {
	if self.fingerprint.Valid {
		fp := &self.fingerprint.Value
		fp.Deinit()
	}
	self.window.Deinit()
}

// Reset is ViewportSearch.reset.
//
// Reset our fingerprint and results so that the next update will
// always re-search.
func (self *SearchViewport) Reset() {
	if self.fingerprint.Valid {
		fp := &self.fingerprint.Value
		fp.Deinit()
	}
	self.fingerprint = None[searchViewportFingerprint]()
	self.window.ClearAndRetainCapacity()
}

// Needle is ViewportSearch.needle.
//
// The needle that this search is using.
//
// Zig: []const u8, the window's own needle (not a copy).
func (self *SearchViewport) Needle() []uint8 {
	assert(self.window.direction == searchSlidingWindowDirectionForward)
	return self.window.needle
}

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
func (self *SearchViewport) Update(list *PageList) (_ bool, err error) {
	// See if our viewport changed
	fingerprint, err := searchViewportFingerprintInit(list)
	if err != nil {
		return false, err
	}
	if self.fingerprint.Valid {
		old := &self.fingerprint.Value
		if old.Eql(fingerprint) {
			// match: (true if the block ends without `break :match`)
			unchanged := func() bool {
				// Determine if we need to check if we overlap the active
				// area. If we have dirty tracking on we also set it to
				// false here.
				check_active := func() bool {
					// active:
					if !self.active_dirty.Valid {
						return true
					}
					dirty := self.active_dirty.Value
					if !dirty {
						return false
					}
					self.active_dirty = Some(false)
					return true
				}()

				if check_active {
					// If our fingerprint contains the active area, then we always
					// re-search since the active area is mutable.
					active_tl := list.GetTopLeft(PointTagActive)
					active_br_ := list.GetBottomRight(PointTagActive)
					assert(active_br_.Valid) // .?
					active_br := active_br_.Value

					// If our viewport contains the start or end of the active area,
					// we are in the active area. We purposely do this first
					// because our viewport is always larger than the active area.
					for _, node := range old.nodes {
						if node == active_tl.node {
							return false // break :match
						}
						if node == active_br.node {
							return false // break :match
						}
					}
				}

				return true
			}()
			if unchanged {
				// No change
				fingerprint.Deinit()
				return false, nil
			}
		}

		old.Deinit()
		self.fingerprint = None[searchViewportFingerprint]()
	}
	assert(!self.fingerprint.Valid)
	self.fingerprint = Some(fingerprint)
	defer func() {
		// errdefer
		if err != nil {
			fingerprint.Deinit()
			self.fingerprint = None[searchViewportFingerprint]()
		}
	}()

	// If our active area was set as dirty, we always unset it here
	// because we're re-searching now.
	if self.active_dirty.Valid {
		self.active_dirty.Value = false
	}

	// Clear our previous sliding window
	self.window.ClearAndRetainCapacity()

	// Add enough overlap to cover needle.len - 1 bytes (if it
	// exists) so we can cover the overlap. We only do this for the
	// soft-wrapped prior pages.
	node_ := fingerprint.nodes[0].prev
	// Zig: usize, 64 bits on the reference on every platform.
	var added uint64 = 0
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
		var n uint
		n, err = self.window.Append(node)
		if err != nil {
			return false, err
		}
		// Zig: usize addition traps on overflow.
		assert(added <= math.MaxUint64-uint64(n))
		added += uint64(n)
		// Zig: `needle.len - 1` wraps for an empty needle in the release
		// build (project lead's ruling, see the file header).
		if added >= uint64(len(self.window.needle))-1 {
			break
		}
	}

	// We can use our fingerprint nodes to initialize our sliding
	// window, since we already traversed the viewport once.
	for _, node := range fingerprint.nodes {
		if _, err = self.window.Append(node); err != nil {
			return false, err
		}
	}

	// Add any trailing overlap as well.
	func() {
		// trailing:
		end := fingerprint.nodes[len(fingerprint.nodes)-1]
		assert(end.data.size.rows > 0) // Zig: u16 subtraction traps
		if !end.data.GetRow(uint(end.data.size.rows - 1)).wrap {
			return // break :trailing
		}

		node_ = end.next
		added = 0
		for ; node_ != nil; node_ = node_.next {
			node := node_
			var n uint
			n, err = self.window.Append(node)
			if err != nil {
				return
			}
			// Zig: usize addition traps on overflow.
			assert(added <= math.MaxUint64-uint64(n))
			added += uint64(n)
			if added >= uint64(len(self.window.needle))-1 {
				break
			}

			// If this row doesn't wrap, then we can quit
			assert(node.data.size.rows > 0) // Zig: u16 subtraction traps
			row := node.data.GetRow(uint(node.data.size.rows - 1))
			if !row.wrap {
				break
			}
		}
	}()
	if err != nil {
		return false, err
	}

	return true, nil
}

// Next is ViewportSearch.next.
//
// Find the next match for the needle in the active area. This returns
// null when there are no more matches.
//
// Zig: ?FlattenedHighlight. The result shares the window's memory; see
// SearchSlidingWindow.Next.
func (self *SearchViewport) Next() Optional[HighlightFlattened] {
	return self.window.Next()
}

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
	var list []*PageListNode // std.ArrayList(*PageList.List.Node) = .empty
	// Zig: defer list.deinit(alloc); Go memory.

	// Get our viewport area. Bottom right of a viewport can never
	// fail.
	tl := pages.GetTopLeft(PointTagViewport)
	br := pages.GetBottomRight(PointTagViewport)
	assert(br.Valid) // .?

	it := tl.PageIterator(PageListDirectionRightDown, br)
	for chunk := it.Next(); chunk.Valid; chunk = it.Next() {
		list = append(list, chunk.Value.node)
	}
	// Zig: list.toOwnedSlice(alloc); the Go slice itself.
	return searchViewportFingerprint{nodes: list}, nil
}

// Deinit is ViewportSearch.Fingerprint.deinit.
//
// Zig: pub fn deinit(self: *Fingerprint, alloc: Allocator) void. The
// allocator parameter is dropped. It only frees; the stub is kept for the
// callers.
func (self *searchViewportFingerprint) Deinit() {
	// Zig: alloc.free(self.nodes), which leaves `nodes` dangling. The
	// garbage collector frees; the slice is dropped (nil).
	self.nodes = nil
}

// Eql is ViewportSearch.Fingerprint.eql.
func (self searchViewportFingerprint) Eql(other searchViewportFingerprint) bool {
	if len(self.nodes) != len(other.nodes) {
		return false
	}
	for i, a := range self.nodes {
		b := other.nodes[i]
		if a != b {
			return false
		}
	}
	return true
}
