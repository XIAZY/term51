// Port of src/terminal/search.zig (Ghostty v1.3.1).
//
// Search functionality for the terminal.
//
// search.zig has no function. It is a namespace file whose declarations
// take the prefix `Search`, and it re-exports the main type of four files
// in src/terminal/search/.
//
// # Naming rule for the search files
//
// A file `search/<name>.zig` is `search_<name>.go` with the prefix
// `Search<Name>` (docs/mapping.md, "Wireframe naming decisions"). The MAIN
// type of each file, the one search.zig re-exports under the file's name,
// is named by that bare prefix, and everything nested in it takes the same
// prefix:
//
//	Zig file                    Zig type         search.zig name   Go type
//	search/active.zig           ActiveSearch     search.Active     SearchActive
//	search/pagelist.zig         PageListSearch   search.PageList   SearchPagelist
//	search/screen.zig           ScreenSearch     search.Screen     SearchScreen
//	search/viewport.zig         ViewportSearch   search.Viewport   SearchViewport
//	search/sliding_window.zig   SlidingWindow    (not exported)    SearchSlidingWindow
//
// This is the mapping rule "a declaration with the same name as its file is
// not doubled" (`Style`, `Point`) applied to the name the type is published
// under: search.zig names `ActiveSearch` `Active`, the name of its file, so
// `search.Active` and the Go type are both `SearchActive`; `SlidingWindow`
// has the name of its file directly. The rejected alternative,
// `<prefix><Zig type>` (`SearchActiveActiveSearch`,
// `SearchScreenScreenSearchSelectedMatch`), says every word twice.
//
// So `search.Active`, `search.Screen` and `search.Viewport` need no Go
// declaration: their Go names ARE the types. `search.PageList` differs from
// the file prefix only in the case of one letter (the file is
// `pagelist.zig`), so it is the alias `SearchPageList` below.
//
// # Omitted
//
//   - `options` (`@import("terminal_options")`): build options, not a
//     declaration with behaviour; `options.artifact` is `.lib` for this
//     library.
//   - `Thread`: OUT OF SCOPE. In the library build (`options.artifact ==
//     .lib`) search.zig declares `pub const Thread = void` ("The search
//     thread is not available in libghostty due to the xev dep for now"),
//     and search/Thread.zig is not part of the library build. Nothing is
//     declared for it and there is no search_Thread.go.
//   - The `test` block (refAllDecls and the import of sliding_window.zig)
//     has nothing to translate.
// No C ABI declaration exists in this file.
//
// # Zig defaults a Go literal does not give
//
// None: the file has no struct.

package terminal

// SearchPageList is search.PageList: pagelist.PageListSearch, whose Go type
// is SearchPagelist (search_pagelist.go). See the file header.
type SearchPageList = SearchPagelist
