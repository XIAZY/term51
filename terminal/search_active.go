// WIREFRAME for src/terminal/search/active.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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
func SearchActiveInit(needle []uint8) (SearchActive, error) { panic(notImplemented) }

// Deinit is ActiveSearch.deinit.
//
// Kept as a stub: every test calls it.
func (self *SearchActive) Deinit() { panic(notImplemented) }

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
func (self *SearchActive) Update(list *PageList) (*PageListNode, error) { panic(notImplemented) }

// Next is ActiveSearch.next.
//
// Find the next match for the needle in the active area. This returns
// null when there are no more matches.
//
// Zig: ?FlattenedHighlight. The result shares the window's memory; see
// SearchSlidingWindow.Next.
func (self *SearchActive) Next() Optional[HighlightFlattened] { panic(notImplemented) }
