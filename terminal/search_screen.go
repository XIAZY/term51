// Port of src/terminal/search/screen.zig (Ghostty v1.3.1).
//
// File naming: `search/screen.zig` is `search_screen.go`, prefix
// `SearchScreen` (lowerCamel `searchScreen` for private declarations). The
// file's main declaration, `ScreenSearch`, is the type search.zig re-exports
// as `search.Screen`, so it is named by the bare file prefix: `SearchScreen`
// (the rule for the main type of every search file; see search.go). What is
// nested in it takes that prefix: `ScreenSearch.SelectedMatch` is
// `SearchScreenSelectedMatch`, `ScreenSearch.Select` is `SearchScreenSelect`
// (values `SearchScreenSelectNext`, `SearchScreenSelectPrev`). The private
// nested `ScreenSearch.HistorySearch` is `searchScreenHistorySearch` and
// `ScreenSearch.State` is `searchScreenState` (values
// `searchScreenStateActive`, `...History`, `...HistoryFeed`,
// `...Complete`). `init` is `SearchScreenInit`. `pub fn` methods are
// exported, also on the private types (`Deinit`, `Needle`, `MatchesLen`,
// `Matches`, `SearchAll`, `Tick`, `Feed`, `ReloadActive`, `SelectedMatch`,
// `Select`, `IsComplete`, `NeedsFeed`); private methods keep their Zig name
// (`pruneHistory`, `tickActive`, `tickHistory`, `selectNext`,
// `selectPrev`).
//
// The method `SelectedMatch` (ScreenSearch.selectedMatch) and the type
// `SearchScreenSelectedMatch` (ScreenSearch.SelectedMatch) are different
// declarations, as in Zig.
//
// Declarations appear in the same order as in screen.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory.
//
//   - Every allocator parameter is dropped (`init`, `matches`). `Deinit`
//     stubs are kept (every test calls `search.Deinit()`).
//   - `history_results` and `active_results` are
//     `std.ArrayList(FlattenedHighlight)`: `[]HighlightFlattened`. Zig's
//     `.items.len` is `len(...)`, `.empty` is nil, `clearRetainingCapacity`
//     is `[:0]`, `shrinkAndFree(alloc, i)` is `[:i]`,
//     `replaceRangeAssumeCapacity(0, i, &.{})` removes the first i
//     elements. Each stored highlight is a `Clone` (highlight.go) of the
//     window's result, as in Zig.
//   - `matches` returns an owned slice the caller frees in Zig: a fresh Go
//     slice, `[]HighlightFlattened`.
//   - `screen: *Screen` is a Go pointer; `HistorySearch.start_pin` is a
//     tracked `*PageListPin` owned by `screen.pages`.
//   - `history: ?HistorySearch` and `selected: ?SelectedMatch` are
//     `Optional[...]`. Zig's `if (self.history) |*h|` and
//     `if (self.selected) |*m|` modify the payload in place: in Go
//     `h := &self.history.Value` after checking `Valid`.
//   - `feed` replaces the whole search on a resize (`self.* = new`): the Go
//     code assigns `*self = new` after `self.Deinit()`.
//
// # Errors
//
// `Allocator.Error` is ErrOutOfMemory (dcs.go). `TickError` is
// `Allocator.Error || error{ FeedRequired, SearchComplete }`; an error set
// is not a type in Go, so it is not declared: `Tick` returns ErrOutOfMemory,
// ErrFeedRequired or ErrSearchComplete (the last two declared below).
// `select` errors that Zig only logs with `log.info` in `reloadActive` are
// ignored there in Go too.
//
// # Widths
//
// `SelectedMatch.idx`, the result of `matchesLen` and the local counts in
// `reloadActive` are `usize`, so `uint`. `rows` and `cols` are
// `SizeCellCountInt`.
//
// # Build options
//
// `std.debug.runtime_safety` (the history-index assert in `reloadActive`)
// is on in upstream's tests; the Go implementation keeps the check
// unconditionally.
//
// # Declarations that are not Zig declarations
//
//   - ErrFeedRequired and ErrSearchComplete: members of `TickError`;
//   - the `String` methods of searchScreenState and SearchScreenSelect.
//
// # Omitted
//
// Test-only fault injection, as in Screen.go and PageList.go:
//   - `reloadActive_tw` (line 21, `tripwire.module(...)`) and the
//     `tw.check(.history_append_new)` and `tw.check(.history_append_existing)`
//     calls in `reloadActive`. There is no allocator in the Go port to fail.
//     The two upstream tests that arm it ("reloadActive partial history
//     cleanup on appendSlice error", "reloadActive partial history cleanup
//     on loop append error") check for leaked memory after an injected
//     OutOfMemory; they have nothing to run against in Go.
// Raw memory (Go-memory rule), as `pageAllocator` in PageList.go:
//   - `allocator()` (line 170): returns `self.active.window.alloc` and does
//     nothing else. Its callers drop the allocator argument.
// Imports and aliases with nothing to declare: `std`, `assert`, `testing`,
// `tripwire`, `Allocator`, `point`, `highlight`, `size`,
// `FlattenedHighlight` (use `HighlightFlattened`), `TrackedHighlight` (use
// `HighlightTracked`), `PageList`, `Pin` (use `PageListPin`), `Screen`,
// `Terminal`, `ActiveSearch` (use `SearchActive`), `PageListSearch` (use
// `SearchPagelist`), `SlidingWindow` (use `SearchSlidingWindow`), and the
// `log` scope `.search_screen` (used only for `log.info`, which the port does
// not print). No C ABI declaration exists in this file.
//
// # Zig defaults a Go literal does not give
//
// `selected` defaults to null, the Go zero value; no other field has a Zig
// default. A ScreenSearch is made only by its `init`, which sets `rows` and
// `cols` from the screen, makes the active search (a forward window) and
// runs `reloadActive`; `state` starts at `.active`, which is the Go zero
// value. So a translated test makes one with SearchScreenInit, never with a
// literal. SelectedMatch and HistorySearch have no field defaults.

package terminal

import (
	"errors"
	"math"
)

// SearchScreen is screen.ScreenSearch (re-exported as search.Screen).
//
// Searches for a needle within a Screen, handling active area updates,
// pages being pruned from the screen (e.g. scrollback limits), and more.
//
// Unlike our lower-level searchers (like ActiveSearch and PageListSearch),
// this will cache and store all search results so the caller can re-access
// them as needed. This structure does this because it is intended to help
// the caller handle the case where the Screen is changing while the user
// is searching.
//
// An inactive screen can continue to be searched in the background, and when
// screen state changes, the renderer/caller can access the existing search
// results without needing to re-search everything. This prevents a particularly
// nasty UX where going to alt screen (e.g. neovim) and then back would
// restart the full scrollback search.
type SearchScreen struct {
	// The screen being searched.
	screen *Screen

	// The active area search state
	active SearchActive

	// The history (scrollback) search state. May be null if there is
	// no history yet.
	//
	// Zig: ?HistorySearch.
	history Optional[searchScreenHistorySearch]

	// Current state of the search, a state machine.
	state searchScreenState

	// The currently selected match, if any. As the screen contents
	// change or get pruned, the screen search will do its best to keep
	// this accurate.
	//
	// Zig: ?SelectedMatch, default null.
	selected Optional[SearchScreenSelectedMatch]

	// The results found so far. These are stored separately because history
	// is mostly immutable once found, while active area results may
	// change. This lets us easily reset the active area results for a
	// re-search scenario.
	//
	// Zig: std.ArrayList(FlattenedHighlight).
	history_results []HighlightFlattened
	active_results  []HighlightFlattened

	// The dimensions of the screen. When this changes we need to
	// restart the whole search, currently.
	rows SizeCellCountInt
	cols SizeCellCountInt
}

// SearchScreenSelectedMatch is ScreenSearch.SelectedMatch.
type SearchScreenSelectedMatch struct {
	// Index from the end of the match list (0 = most recent match)
	//
	// Zig: usize.
	idx uint

	// Tracked highlight so we can detect movement.
	highlight HighlightTracked
}

// Deinit is ScreenSearch.SelectedMatch.deinit.
//
// Kept: it untracks the highlight's pins in the screen's page list.
func (self *SearchScreenSelectedMatch) Deinit(screen *Screen) {
	self.highlight.Deinit(screen)
}

// searchScreenHistorySearch is ScreenSearch.HistorySearch.
//
// History search state.
type searchScreenHistorySearch struct {
	// The actual searcher state.
	searcher SearchPagelist

	// The pin for the first node that this searcher is searching from.
	// We use this when the active area changes to find the diff between
	// the top of the new active area and the previous start point
	// to determine if we need to search more history.
	start_pin *PageListPin
}

// Deinit is ScreenSearch.HistorySearch.deinit.
//
// Kept: it deinits the searcher and untracks `start_pin`.
func (self *searchScreenHistorySearch) Deinit(screen *Screen) {
	self.searcher.Deinit()
	screen.pages.UntrackPin(self.start_pin)
}

// searchScreenState is ScreenSearch.State: search state machine.
//
// Zig: enum, values numbered from zero in the order listed.
type searchScreenState uint8

const (
	// Currently searching the active area
	searchScreenStateActive searchScreenState = iota

	// Currently searching the history area
	searchScreenStateHistory

	// History search is waiting for more data to be fed before
	// it can progress.
	searchScreenStateHistoryFeed

	// Search is complete given the current terminal state.
	searchScreenStateComplete
)

// searchScreenStateNames are the tag names of searchScreenState, in order.
var searchScreenStateNames = [...]string{
	"active",
	"history",
	"history_feed",
	"complete",
}

// String returns the Zig tag name (@tagName).
func (self searchScreenState) String() string {
	return searchScreenStateNames[self]
}

// IsComplete is ScreenSearch.State.isComplete.
func (self searchScreenState) IsComplete() bool {
	switch self {
	case searchScreenStateComplete:
		return true
	default:
		return false
	}
}

// NeedsFeed is ScreenSearch.State.needsFeed.
//
// True for `.history_feed`, and also for `.complete`: not obvious but
// complete search states will prune stale history results on feed.
func (self searchScreenState) NeedsFeed() bool {
	switch self {
	case searchScreenStateHistoryFeed:
		return true

	// Not obvious but complete search states will prune
	// stale history results on feed.
	case searchScreenStateComplete:
		return true

	default:
		return false
	}
}

// SearchScreenInit is ScreenSearch.init.
//
// Initialize a screen search for the given screen and needle.
//
// Zig: pub fn init(alloc: Allocator, screen: *Screen, needle_unowned:
// []const u8) Allocator.Error!ScreenSearch. The allocator parameter is
// dropped. The error is ErrOutOfMemory (dcs.go).
func SearchScreenInit(screen *Screen, needle_unowned []uint8) (_ SearchScreen, err error) {
	active, err := SearchActiveInit(needle_unowned)
	if err != nil {
		return SearchScreen{}, err
	}
	result := SearchScreen{
		screen:          screen,
		rows:            screen.pages.rows,
		cols:            screen.pages.cols,
		active:          active,
		history:         None[searchScreenHistorySearch](),
		state:           searchScreenStateActive,
		active_results:  nil, // .empty
		history_results: nil, // .empty
	}
	defer func() {
		// errdefer
		if err != nil {
			result.Deinit()
		}
	}()

	// Update our initial active area state
	if err = result.ReloadActive(); err != nil {
		return SearchScreen{}, err
	}

	return result, nil
}

// Deinit is ScreenSearch.deinit.
//
// Kept: besides freeing it untracks the pins of the history search and of
// the selected match.
func (self *SearchScreen) Deinit() {
	self.active.Deinit()
	if self.history.Valid {
		h := &self.history.Value
		h.Deinit(self.screen)
	}
	if self.selected.Valid {
		m := &self.selected.Value
		m.Deinit(self.screen)
	}
	for i := range self.active_results {
		self.active_results[i].Deinit()
	}
	self.active_results = nil // Zig: self.active_results.deinit(alloc)
	for i := range self.history_results {
		self.history_results[i].Deinit()
	}
	self.history_results = nil // Zig: self.history_results.deinit(alloc)
}

// ScreenSearch.allocator (line 170) is omitted; see the file header.

// Needle is ScreenSearch.needle.
//
// The needle that this search is using.
//
// Zig: []const u8, the active window's own needle (not a copy).
func (self *SearchScreen) Needle() []uint8 {
	assert(self.active.window.direction == searchSlidingWindowDirectionForward)
	return self.active.window.needle
}

// MatchesLen is ScreenSearch.matchesLen.
//
// Returns the total number of matches found so far.
func (self *SearchScreen) MatchesLen() uint {
	return uint(len(self.active_results) + len(self.history_results))
}

// Matches is ScreenSearch.matches.
//
// Returns all matches as an owned slice (caller must free).
// The matches are ordered from most recent to oldest (e.g. bottom
// of the screen to top of the screen).
//
// Zig: pub fn matches(self: *ScreenSearch, alloc: Allocator)
// Allocator.Error![]FlattenedHighlight. The allocator parameter is dropped;
// the result is a fresh Go slice (the highlights themselves are shallow
// copies of the stored ones, as Zig's @memcpy). The error is ErrOutOfMemory
// (dcs.go).
func (self *SearchScreen) Matches() ([]HighlightFlattened, error) {
	active_results := self.active_results
	history_results := self.history_results
	results := make([]HighlightFlattened, len(active_results)+len(history_results))
	// Zig: errdefer alloc.free(results); Go memory.

	// Active does a forward search, so we add the active results then
	// reverse them. There are usually not many active results so this
	// is fast enough compared to adding them in reverse order.
	assert(self.active.window.direction == searchSlidingWindowDirectionForward)
	copy(
		results[0:len(active_results)],
		active_results,
	)
	stdMemReverse(results[0:len(active_results)])

	// History does a backward search, so we can just append them
	// after.
	copy(
		results[len(active_results):],
		history_results,
	)

	return results, nil
}

// SearchAll is ScreenSearch.searchAll.
//
// Search the full screen state. This will block until the search
// is complete. For performance, it is recommended to use `tick` and
// `feed` to incrementally make progress on the search instead.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) SearchAll() error {
	for {
		if err := self.Tick(); err != nil {
			switch err {
			case ErrOutOfMemory:
				return ErrOutOfMemory
			case ErrFeedRequired:
				if err := self.Feed(); err != nil {
					return err
				}
			case ErrSearchComplete:
				return nil
			default:
				panic("unreachable")
			}
		}
	}
}

// ErrFeedRequired is Zig's error.FeedRequired (ScreenSearch.TickError).
var ErrFeedRequired = errors.New("FeedRequired")

// ErrSearchComplete is Zig's error.SearchComplete (ScreenSearch.TickError).
var ErrSearchComplete = errors.New("SearchComplete")

// Tick is ScreenSearch.tick.
//
// Make incremental progress on the search without accessing any
// screen state (so no lock is required).
//
// This will return error.FeedRequired if the search cannot make progress
// without being fed more data. In this case, the caller should call
// the `feed` function to provide more data to the searcher.
//
// This will return error.SearchComplete if the search is fully complete.
// This is to signal to the caller that it can move to a more efficient
// sleep/wait state until there is more work to do (e.g. new data to feed).
//
// Zig: TickError!void. The error is ErrOutOfMemory (dcs.go),
// ErrFeedRequired or ErrSearchComplete.
func (self *SearchScreen) Tick() error {
	switch self.state {
	case searchScreenStateActive:
		return self.tickActive()
	case searchScreenStateHistory:
		return self.tickHistory()
	case searchScreenStateHistoryFeed:
		return ErrFeedRequired
	case searchScreenStateComplete:
		return ErrSearchComplete
	default:
		panic("unreachable")
	}
}

// Feed is ScreenSearch.feed.
//
// Feed more data to the searcher so it can continue searching. This
// accesses the screen state, so the caller must hold the necessary locks.
//
// Feed on a complete screen search will perform some cleanup of
// potentially stale history results (pruned) and reclaim some memory.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) Feed() error {
	// If the screen resizes, we have to reset our entire search. That
	// isn't ideal but we don't have a better way right now to handle
	// reflowing the search results beyond putting a tracked pin for
	// every single result.
	if self.screen.pages.rows != self.rows ||
		self.screen.pages.cols != self.cols {
		// Reinit
		new, err := SearchScreenInit(
			self.screen,
			self.Needle(),
		)
		if err != nil {
			return err
		}

		// Deinit/reinit
		self.Deinit()
		*self = new

		// New result should have matching dimensions
		assert(self.screen.pages.rows == self.rows)
		assert(self.screen.pages.cols == self.cols)
	}

	if !self.history.Valid {
		// No history to feed, search is complete.
		self.state = searchScreenStateComplete
		return nil
	}
	history := &self.history.Value.searcher

	// Go-only (project lead's ruling, see the file header): feed on a
	// complete search whose history searcher still has data to feed would
	// reach the `.complete => unreachable` arm below. It returns
	// SearchComplete, what tick returns in this state, and changes
	// nothing. searchScreenHistoryWouldFeed runs PageListSearch.feed on a
	// copy, so the history searcher is left as it is.
	if self.state == searchScreenStateComplete {
		would, err := searchScreenHistoryWouldFeed(history)
		if err != nil {
			return err
		}
		if would {
			return ErrSearchComplete
		}
	}

	// Future: we may want to feed multiple pages at once here to
	// lower the frequency of lock acquisitions.
	fed, err := history.Feed()
	if err != nil {
		return err
	}
	if !fed {
		// No more data to feed, search is complete.
		self.state = searchScreenStateComplete

		// We use this opportunity to also clean up older history
		// results that may be gone due to scrollback pruning, though.
		self.pruneHistory()

		return nil
	}

	// Depending on our state handle where feed goes
	switch self.state {
	// If we're searching active or history, then feeding doesn't
	// change the state.
	case searchScreenStateActive, searchScreenStateHistory:

	// Feed goes back to searching history.
	case searchScreenStateHistoryFeed:
		self.state = searchScreenStateHistory

	// If we're complete then the feed call above should always
	// return false and we can't reach this.
	//
	// (In the port it cannot be reached: the ruling above returns
	// first.)
	case searchScreenStateComplete:
		panic("unreachable")

	default:
		panic("unreachable")
	}
	return nil
}

// pruneHistory is ScreenSearch.pruneHistory.
//
// Drops every history result from the first one whose lowest chunk serial
// is below `screen.pages.page_serial_min`.
func (self *SearchScreen) pruneHistory() {
	// Go through our history results in order (newest to oldest) to find
	// any result that contains an invalid serial. Prune up to that
	// point.
	n := len(self.history_results)
	for i := 0; i < n; i++ {
		hl := &self.history_results[i]
		// Zig: hl.chunks.items(.serial)[0]
		lowest := hl.chunks[0].serial
		if lowest < self.screen.pages.page_serial_min {
			// Everything from here forward we assume is invalid because
			// our history results only get older.
			for j := i; j < len(self.history_results); j++ {
				self.history_results[j].Deinit()
			}
			// Zig: shrinkAndFree(alloc, i)
			self.history_results = self.history_results[:i]
			return
		}
	}
}

// tickActive is ScreenSearch.tickActive.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) tickActive() error {
	// For the active area, we consume the entire search in one go
	// because the active area is generally small.
	for hl := self.active.Next(); hl.Valid; hl = self.active.Next() {
		// If this fails, then we miss a result since `active.next()`
		// moves forward and prunes data. In the future, we may want
		// to have some more robust error handling but the only
		// scenario this would fail is OOM and we're probably in
		// deeper trouble at that point anyways.
		hl_cloned, err := hl.Value.Clone()
		if err != nil {
			return err
		}
		// Zig: errdefer hl_cloned.deinit(alloc); Go memory.
		self.active_results = append(self.active_results, hl_cloned)
	}

	// We've consumed the entire active area, move to history.
	self.state = searchScreenStateHistory
	return nil
}

// tickHistory is ScreenSearch.tickHistory.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) tickHistory() error {
	if !self.history.Valid {
		// No history to search, we're done.
		self.state = searchScreenStateComplete
		return nil
	}
	history := &self.history.Value

	// Try to consume all the loaded matches in one go, because
	// the search is generally fast for loaded data.
	for hl := history.searcher.Next(); hl.Valid; hl = history.searcher.Next() {
		// Ignore selections that are found within the starting
		// node since those are covered by the active area search.
		if hl.Value.chunks[0].node == history.start_pin.node {
			continue
		}

		// Same note as tickActive for error handling.
		hl_cloned, err := hl.Value.Clone()
		if err != nil {
			return err
		}
		// Zig: errdefer hl_cloned.deinit(alloc); Go memory.
		self.history_results = append(self.history_results, hl_cloned)

		// Since history only appends to our results in reverse order,
		// we don't need to update any selected match state. The index
		// and prior results are unaffected.
	}

	// We need to be fed more data.
	self.state = searchScreenStateHistoryFeed
	return nil
}

// ReloadActive is ScreenSearch.reloadActive.
//
// Reload the active area because it has changed.
//
// Since it is very fast, this will also do the full active area
// search again, too. This avoids any complexity around the search
// state machine.
//
// The caller must hold the necessary locks to access the screen state.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go). The
// tripwire checks are omitted; see the file header.
func (self *SearchScreen) ReloadActive() (err error) {
	// Zig: const tw = reloadActive_tw; omitted (see the file header).

	// If our selection pin became garbage it means we scrolled off
	// the end. Clear our selection and on exit of this function,
	// try to select the last match.
	select_prev := func() bool {
		// select_prev:
		if !self.selected.Valid {
			return false
		}
		m := &self.selected.Value
		if !m.highlight.start.garbage &&
			!m.highlight.end.garbage {
			return false
		}

		m.Deinit(self.screen)
		self.selected = None[SearchScreenSelectedMatch]()
		return true
	}()
	defer func() {
		if select_prev {
			if _, err := self.Select(SearchScreenSelectPrev); err != nil {
				// Zig: log.info("reload failed to reset search selection
				// err={}", .{err}); not printed.
			}
		}
	}()

	list := &self.screen.pages
	history_node, err := self.active.Update(list)
	if err != nil {
		return err
	}
	if history_node != nil {
		err = func() error {
			// history:

			// We need to account for any active area growth that would
			// cause new pages to move into our history. If there are new
			// pages then we need to re-search the pages and add it to
			// our history results.

			// If our screen has no scrollback then we have no history.
			if self.screen.no_scrollback {
				assert(!self.history.Valid)
				return nil // break :history
			}

			var history_ *searchScreenHistorySearch
			if self.history.Valid {
				// state:
				h := &self.history.Value
				// If our start pin became garbage, it means we pruned all
				// the way up through it, so we have no history anymore.
				// Reset our history state.
				if h.start_pin.garbage {
					h.Deinit(self.screen)
					self.history = None[searchScreenHistorySearch]()
					for i := range self.history_results {
						self.history_results[i].Deinit()
					}
					self.history_results = self.history_results[:0]
					history_ = nil
				} else {
					history_ = h
				}
			}

			if history_ == nil {
				// No history search yet, but we now have history. So let's
				// initialize.

				search, err := SearchPagelistInit(
					self.Needle(),
					list,
					history_node,
				)
				if err != nil {
					return err
				}

				pin, err := list.TrackPin(PageListPin{node: history_node})
				if err != nil {
					// Zig: errdefer search.deinit()
					search.Deinit()
					return err
				}
				// Zig: errdefer list.untrackPin(pin); nothing after it can
				// fail, so it never runs.

				self.history = Some(searchScreenHistorySearch{
					searcher:  search,
					start_pin: pin,
				})

				// We don't need to update any history since we had no history
				// before, so we can break out of the whole conditional.
				return nil // break :history
			}
			history := history_

			if history.start_pin.node == history_node {
				// No change in the starting node, we're done.
				return nil // break :history
			}

			// Do a forward search from our prior node to this one. We
			// collect all the results into a new list. We ASSUME that
			// reloadActive is being called frequently enough that there isn't
			// a massive amount of history to search here.
			window, err := SearchSlidingWindowInit(
				searchSlidingWindowDirectionForward,
				self.Needle(),
			)
			if err != nil {
				return err
			}
			defer window.Deinit()
			for {
				if _, err := window.Append(history.start_pin.node); err != nil {
					return err
				}
				if history.start_pin.node == history_node {
					break
				}
				next := history.start_pin.node.next
				if next == nil {
					break
				}
				history.start_pin.node = next
			}
			assert(history.start_pin.node == history_node)

			results := make([]HighlightFlattened, 0, len(self.history_results))
			// Zig: errdefer { for (results.items) |*hl| hl.deinit(alloc);
			// results.deinit(alloc); } Go memory.
			for hl := window.Next(); hl.Valid; hl = window.Next() {
				if hl.Value.chunks[0].node == history_node {
					continue
				}

				hl_cloned, err := hl.Value.Clone()
				if err != nil {
					return err
				}
				// Zig: errdefer hl_cloned.deinit(alloc); Go memory.
				// Zig: try tw.check(.history_append_new); omitted.
				results = append(results, hl_cloned)
			}

			// If we have no matches then there is nothing to change
			// in our history (fast path)
			if len(results) == 0 {
				return nil // break :history
			}

			// The number added to our history. Needed for updating
			// our selection if we have one.
			added_len := uint(len(results))

			// Matches! Reverse our list then append all the remaining
			// history items that didn't start on our original node.
			stdMemReverse(results)
			// Zig: try tw.check(.history_append_existing); omitted.
			results = append(results, self.history_results...)
			// Zig: self.history_results.deinit(alloc); Go memory.
			self.history_results = results

			// If our prior selection was in the history area, update
			// the offset.
			if self.selected.Valid {
				// selected:
				m := &self.selected.Value
				active_len := uint(len(self.active_results))
				if m.idx >= active_len {
					assert(m.idx <= math.MaxUint-added_len) // Zig: usize addition traps
					m.idx += added_len

					// Moving the idx should not change our targeted result
					// since the history is immutable.
					//
					// Zig: if (comptime std.debug.runtime_safety); see the
					// file header.
					hl := self.history_results[m.idx-active_len]
					assert(m.highlight.start.Eql(hl.StartPin()))
				}
			}
			return nil
		}()
		if err != nil {
			return err
		}
	} else {
		// No history node means we have no history
		if self.history.Valid {
			h := &self.history.Value
			h.Deinit(self.screen)
			self.history = None[searchScreenHistorySearch]()
			for i := range self.history_results {
				self.history_results[i].Deinit()
			}
			self.history_results = self.history_results[:0]
		}

		// If we have a selection in the history area, we need to
		// move it to the end of the active area.
		if self.selected.Valid {
			// selected:
			m := &self.selected.Value
			active_len := uint(len(self.active_results))
			if m.idx >= active_len {
				m.Deinit(self.screen)
				self.selected = None[SearchScreenSelectedMatch]()
				if _, err := self.Select(SearchScreenSelectPrev); err != nil {
					// Zig: log.info("reload failed to reset search selection
					// err={}", .{err}); not printed.
				}
			}
		}
	}

	// Figure out if we need to fixup our selection later because
	// it was in the active area.
	old_active_len := uint(len(self.active_results))
	old_selection_idx := None[uint]()
	if self.selected.Valid {
		old_selection_idx = Some(self.selected.Value.idx)
	}
	defer func() {
		// errdefer
		if err != nil && old_selection_idx.Valid &&
			old_selection_idx.Value < old_active_len {
			// This is the error scenario. If something fails below,
			// our active area is probably gone, so we just go back
			// to the first result because our selection can't be trusted.
			if self.selected.Valid {
				m := &self.selected.Value
				m.Deinit(self.screen)
				self.selected = None[SearchScreenSelectedMatch]()
				if _, err := self.Select(SearchScreenSelectNext); err != nil {
					// Zig: log.info("reload failed to reset search selection
					// err={}", .{err}); not printed.
				}
			}
		}
	}()

	// Reset our active search results and search again.
	for i := range self.active_results {
		self.active_results[i].Deinit()
	}
	self.active_results = self.active_results[:0]
	switch self.state {
	// If we're in the active state we run a normal tick so
	// we can move into a better state.
	case searchScreenStateActive:
		if err = self.tickActive(); err != nil {
			return err
		}

	// Otherwise, just tick it and move back to whatever state
	// we were in.
	default:
		err = func() error {
			old_state := self.state
			defer func() { self.state = old_state }()
			return self.tickActive()
		}()
		if err != nil {
			return err
		}
	}

	// If we have no scrollback, we need to prune any active results
	// that aren't in the actual active area. We only do this for the
	// no scrollback scenario because with scrollback we actually
	// rely on our active search searching by page to find history
	// items as well. This is all related to the fact that PageList
	// scrollback limits are discrete by page size except we special
	// case zero.
	if self.screen.no_scrollback &&
		len(self.active_results) > 0 {
		func() {
			// active_prune:
			items := self.active_results
			tl := self.screen.pages.GetTopLeft(PointTagActive)
			for i := range items {
				hl := &items[i]
				if !tl.Before(hl.EndPin()) {
					// Deinit because its going to be pruned no matter
					// what at some point for not being in the active area.
					hl.Deinit()
					continue
				}

				// In the active area! Since our results are sorted
				// that means everything after this is also in the active
				// area, so we prune up to this i.
				if i > 0 {
					// Zig: replaceRangeAssumeCapacity(0, i, &.{})
					n := copy(self.active_results, self.active_results[i:])
					self.active_results = self.active_results[:n]
				}

				return // break :active_prune
			}

			// None are in the active area...
			self.active_results = self.active_results[:0]
		}()
	}

	// Now we have to fixup our selection if we had one.
	func() {
		// fixup:
		if !old_selection_idx.Valid {
			return // break :fixup
		}
		old_idx := old_selection_idx.Value
		if !self.selected.Valid {
			return // break :fixup
		}
		m := &self.selected.Value

		// If our old selection wasn't in the active area, then we
		// need to fix up our offsets.
		if old_idx >= old_active_len {
			assert(m.idx >= old_active_len) // Zig: usize subtraction traps
			m.idx -= old_active_len
			assert(m.idx <= math.MaxUint-uint(len(self.active_results))) // Zig: usize addition traps
			m.idx += uint(len(self.active_results))
			return // break :fixup
		}

		// We search for the matching highlight in the new active results.
		for i, hl := range self.active_results {
			untracked := hl.Untracked()
			if m.highlight.start.Eql(untracked.start) &&
				m.highlight.end.Eql(untracked.end) {
				// Found it! Update our index.
				m.idx = uint(len(self.active_results)) - 1 - uint(i)
				return // break :fixup
			}
		}

		// No match, just go back to the first match.
		m.Deinit(self.screen)
		self.selected = None[SearchScreenSelectedMatch]()
		if _, err := self.Select(SearchScreenSelectNext); err != nil {
			// Zig: log.info("reload failed to reset search selection
			// err={}", .{err}); not printed.
		}
	}()
	return nil
}

// SelectedMatch is ScreenSearch.selectedMatch.
//
// Return the selected match.
//
// This does not require read/write access to the underlying screen.
//
// Zig: ?FlattenedHighlight, a copy of the stored result.
func (self *SearchScreen) SelectedMatch() Optional[HighlightFlattened] {
	if !self.selected.Valid {
		return None[HighlightFlattened]()
	}
	sel := self.selected.Value
	active_len := uint(len(self.active_results))
	if sel.idx < active_len {
		return Some(self.active_results[active_len-1-sel.idx])
	}

	history_len := uint(len(self.history_results))
	if sel.idx < active_len+history_len {
		return Some(self.history_results[sel.idx-active_len])
	}

	return None[HighlightFlattened]()
}

// SearchScreenSelect is ScreenSearch.Select.
//
// Zig: enum, values numbered from zero in the order listed.
type SearchScreenSelect uint8

const (
	// Next selection, in reverse order (newest to oldest),
	// non-wrapping.
	SearchScreenSelectNext SearchScreenSelect = iota

	// Prev selection, in forward order (oldest to newest),
	// non-wrapping.
	SearchScreenSelectPrev
)

// searchScreenSelectNames are the tag names of SearchScreenSelect, in order.
var searchScreenSelectNames = [...]string{
	"next",
	"prev",
}

// String returns the Zig tag name (@tagName).
func (self SearchScreenSelect) String() string {
	return searchScreenSelectNames[self]
}

// Select is ScreenSearch.select.
//
// Select the next or previous search result. This requires read/write
// access to the underlying screen, since we utilize tracked pins to
// ensure our selection sticks with contents changing.
//
// Zig: Allocator.Error!bool. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) Select(to SearchScreenSelect) (bool, error) {
	// All selection requires valid pins so we prune history and
	// reload our active area immediately. This ensures all search
	// results point to valid nodes.
	if err := self.ReloadActive(); err != nil {
		return false, err
	}
	self.pruneHistory()

	switch to {
	case SearchScreenSelectNext:
		return self.selectNext()
	case SearchScreenSelectPrev:
		return self.selectPrev()
	default:
		panic("unreachable")
	}
}

// selectNext is ScreenSearch.selectNext.
//
// Zig: Allocator.Error!bool. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) selectNext() (bool, error) {
	// Get our previous match so we can change it. If we have no
	// prior match, we have the easy task of getting the first.
	if !self.selected.Valid {
		// Get our highlight
		var hl HighlightFlattened
		if len(self.active_results) > 0 {
			// Active is in forward order
			len_ := len(self.active_results)
			hl = self.active_results[len_-1]
		} else if len(self.history_results) > 0 {
			// History is in reverse order
			hl = self.history_results[0]
		} else {
			// No matches at all. Can't select anything.
			return false, nil
		}

		// Pin it so we can track any movement
		untracked := hl.Untracked()
		tracked, err := untracked.Track(self.screen)
		if err != nil {
			return false, err
		}
		// Zig: errdefer tracked.deinit(self.screen); nothing after it can
		// fail, so it never runs.

		// Our selection is index zero since we just started and
		// we store our selection.
		self.selected = Some(SearchScreenSelectedMatch{
			idx:       0,
			highlight: tracked,
		})
		return true, nil
	}
	prev := &self.selected.Value

	assert(prev.idx < math.MaxUint) // Zig: usize addition traps
	next_idx := prev.idx + 1
	active_len := uint(len(self.active_results))
	history_len := uint(len(self.history_results))
	if next_idx >= active_len+history_len {
		// No more matches. We don't wrap or reset the match currently.
		return false, nil
	}
	var hl HighlightFlattened
	if next_idx < active_len {
		hl = self.active_results[active_len-1-next_idx]
	} else {
		hl = self.history_results[next_idx-active_len]
	}

	// Pin it so we can track any movement
	untracked := hl.Untracked()
	tracked, err := untracked.Track(self.screen)
	if err != nil {
		return false, err
	}
	// Zig: errdefer tracked.deinit(self.screen); nothing after it can
	// fail, so it never runs.

	// Free our previous match and setup our new selection
	prev.Deinit(self.screen)
	self.selected = Some(SearchScreenSelectedMatch{
		idx:       next_idx,
		highlight: tracked,
	})

	return true, nil
}

// selectPrev is ScreenSearch.selectPrev.
//
// Zig: Allocator.Error!bool. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) selectPrev() (bool, error) {
	// Get our previous match so we can change it. If we have no
	// prior match, we have the easy task of getting the last.
	if !self.selected.Valid {
		// Get our highlight (oldest match)
		var hl HighlightFlattened
		if len(self.history_results) > 0 {
			// History is in reverse order, so last item is oldest
			len_ := len(self.history_results)
			hl = self.history_results[len_-1]
		} else if len(self.active_results) > 0 {
			// Active is in forward order, so first item is oldest
			hl = self.active_results[0]
		} else {
			// No matches at all. Can't select anything.
			return false, nil
		}

		// Pin it so we can track any movement
		untracked := hl.Untracked()
		tracked, err := untracked.Track(self.screen)
		if err != nil {
			return false, err
		}
		// Zig: errdefer tracked.deinit(self.screen); nothing after it can
		// fail, so it never runs.

		// Our selection is the last index since we just started
		// and we store our selection.
		active_len := uint(len(self.active_results))
		history_len := uint(len(self.history_results))
		self.selected = Some(SearchScreenSelectedMatch{
			idx:       active_len + history_len - 1,
			highlight: tracked,
		})
		return true, nil
	}
	prev := &self.selected.Value

	// Can't go below zero
	if prev.idx == 0 {
		// No more matches. We don't wrap or reset the match currently.
		return false, nil
	}

	next_idx := prev.idx - 1
	active_len := uint(len(self.active_results))
	var hl HighlightFlattened
	if next_idx < active_len {
		hl = self.active_results[active_len-1-next_idx]
	} else {
		hl = self.history_results[next_idx-active_len]
	}

	// Pin it so we can track any movement
	untracked := hl.Untracked()
	tracked, err := untracked.Track(self.screen)
	if err != nil {
		return false, err
	}
	// Zig: errdefer tracked.deinit(self.screen); nothing after it can
	// fail, so it never runs.

	// Free our previous match and setup our new selection
	prev.Deinit(self.screen)
	self.selected = Some(SearchScreenSelectedMatch{
		idx:       next_idx,
		highlight: tracked,
	})

	return true, nil
}

// searchScreenHistoryWouldFeed reports whether `history.feed()` would
// return true, without changing the history searcher. Not a Zig
// declaration: it serves the project lead's ruling in Feed (feed on a
// complete search). It runs PageListSearch.feed itself on a copy: a fresh
// reverse window with the same needle length and an untracked copy of the
// searcher's pin. feed's result depends only on the pin (garbage, node)
// and on the byte counts that `append` returns, which depend only on the
// node and not on the window's contents.
func searchScreenHistoryWouldFeed(history *SearchPagelist) (bool, error) {
	window, err := SearchSlidingWindowInit(
		searchSlidingWindowDirectionReverse,
		history.window.needle,
	)
	if err != nil {
		return false, err
	}
	pin := *history.pin
	probe := SearchPagelist{
		list:   history.list,
		window: window,
		pin:    &pin,
	}
	// Not probe.Deinit: the pin copy is not tracked.
	defer probe.window.Deinit()
	return probe.Feed()
}
