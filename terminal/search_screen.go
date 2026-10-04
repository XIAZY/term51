// WIREFRAME for src/terminal/search/screen.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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

import "errors"

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
func (self *SearchScreenSelectedMatch) Deinit(screen *Screen) { panic(notImplemented) }

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
func (self *searchScreenHistorySearch) Deinit(screen *Screen) { panic(notImplemented) }

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

// String returns the Zig tag name (@tagName).
func (self searchScreenState) String() string { panic(notImplemented) }

// IsComplete is ScreenSearch.State.isComplete.
func (self searchScreenState) IsComplete() bool { panic(notImplemented) }

// NeedsFeed is ScreenSearch.State.needsFeed.
//
// True for `.history_feed`, and also for `.complete`: not obvious but
// complete search states will prune stale history results on feed.
func (self searchScreenState) NeedsFeed() bool { panic(notImplemented) }

// SearchScreenInit is ScreenSearch.init.
//
// Initialize a screen search for the given screen and needle.
//
// Zig: pub fn init(alloc: Allocator, screen: *Screen, needle_unowned:
// []const u8) Allocator.Error!ScreenSearch. The allocator parameter is
// dropped. The error is ErrOutOfMemory (dcs.go).
func SearchScreenInit(screen *Screen, needle_unowned []uint8) (SearchScreen, error) {
	panic(notImplemented)
}

// Deinit is ScreenSearch.deinit.
//
// Kept: besides freeing it untracks the pins of the history search and of
// the selected match.
func (self *SearchScreen) Deinit() { panic(notImplemented) }

// ScreenSearch.allocator (line 170) is omitted; see the file header.

// Needle is ScreenSearch.needle.
//
// The needle that this search is using.
//
// Zig: []const u8, the active window's own needle (not a copy).
func (self *SearchScreen) Needle() []uint8 { panic(notImplemented) }

// MatchesLen is ScreenSearch.matchesLen.
//
// Returns the total number of matches found so far.
func (self *SearchScreen) MatchesLen() uint { panic(notImplemented) }

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
func (self *SearchScreen) Matches() ([]HighlightFlattened, error) { panic(notImplemented) }

// SearchAll is ScreenSearch.searchAll.
//
// Search the full screen state. This will block until the search
// is complete. For performance, it is recommended to use `tick` and
// `feed` to incrementally make progress on the search instead.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) SearchAll() error { panic(notImplemented) }

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
func (self *SearchScreen) Tick() error { panic(notImplemented) }

// Feed is ScreenSearch.feed.
//
// Feed more data to the searcher so it can continue searching. This
// accesses the screen state, so the caller must hold the necessary locks.
//
// Feed on a complete screen search will perform some cleanup of
// potentially stale history results (pruned) and reclaim some memory.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) Feed() error { panic(notImplemented) }

// pruneHistory is ScreenSearch.pruneHistory.
//
// Drops every history result from the first one whose lowest chunk serial
// is below `screen.pages.page_serial_min`.
func (self *SearchScreen) pruneHistory() { panic(notImplemented) }

// tickActive is ScreenSearch.tickActive.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) tickActive() error { panic(notImplemented) }

// tickHistory is ScreenSearch.tickHistory.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) tickHistory() error { panic(notImplemented) }

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
func (self *SearchScreen) ReloadActive() error { panic(notImplemented) }

// SelectedMatch is ScreenSearch.selectedMatch.
//
// Return the selected match.
//
// This does not require read/write access to the underlying screen.
//
// Zig: ?FlattenedHighlight, a copy of the stored result.
func (self *SearchScreen) SelectedMatch() Optional[HighlightFlattened] { panic(notImplemented) }

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

// String returns the Zig tag name (@tagName).
func (self SearchScreenSelect) String() string { panic(notImplemented) }

// Select is ScreenSearch.select.
//
// Select the next or previous search result. This requires read/write
// access to the underlying screen, since we utilize tracked pins to
// ensure our selection sticks with contents changing.
//
// Zig: Allocator.Error!bool. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) Select(to SearchScreenSelect) (bool, error) { panic(notImplemented) }

// selectNext is ScreenSearch.selectNext.
//
// Zig: Allocator.Error!bool. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) selectNext() (bool, error) { panic(notImplemented) }

// selectPrev is ScreenSearch.selectPrev.
//
// Zig: Allocator.Error!bool. The error is ErrOutOfMemory (dcs.go).
func (self *SearchScreen) selectPrev() (bool, error) { panic(notImplemented) }
