// WIREFRAME for src/terminal/render.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// Developer note: this is in src/terminal and not src/renderer because
// the goal is that this remains generic to multiple renderers. This can
// aid specifically with libghostty-vt with converting terminal state to
// a renderable form.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: render.zig is a namespace file, so its declarations take the
// prefix `Render`, except `render.RenderState`, which is `RenderState` (a
// declaration whose name already carries the file's name is not doubled,
// as for `Style` and `Point`). What is nested in RenderState takes the
// prefix `RenderState`: `RenderState.empty` is `RenderStateEmpty`,
// `RenderState.Colors` is `RenderStateColors`, `RenderState.Cursor` is
// `RenderStateCursor`, `RenderState.Cursor.Viewport` is
// `RenderStateCursorViewport`, `RenderState.Row` is `RenderStateRow`,
// `RenderState.Highlight` is `RenderStateHighlight`, `RenderState.Cell` is
// `RenderStateCell`, `RenderState.Dirty` is `RenderStateDirty` (values
// `RenderStateDirtyFalse`, `RenderStateDirtyPartial`,
// `RenderStateDirtyFull`), `RenderState.StringMap` is
// `RenderStateStringMap`, `RenderState.CellSet` is `RenderStateCellSet`.
// The private type `RenderState.SelectionCache` is
// `renderStateSelectionCache`. `pub fn` methods are exported (`Deinit`,
// `Update`, `UpdateHighlightsFlattened`, `String`, `LinkCells`).
//
// Declarations appear in the same order as in render.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory.
//
//   - Every allocator parameter is dropped (`deinit`, `update`,
//     `updateHighlightsFlattened`, `linkCells`, and the `alloc` field of the
//     anonymous struct `string` takes). The `Deinit` stub is kept; every
//     test calls it.
//   - `RenderState.row_data` is `std.MultiArrayList(Row)`, a struct-of-arrays
//     list: a `[]RenderStateRow`. Zig's `row_data.items(.cells)[y]` is
//     `row_data[y].cells`, `row_data.slice()` is `row_data`, `row_data.len`
//     is `len(row_data)`, `row_data.capacity` is `cap(row_data)`, `.empty`
//     is the nil slice.
//   - `Row.cells` is `std.MultiArrayList(Cell)`: a `[]RenderStateCell`.
//     Zig's `cells.get(x)` is `cells[x]`, `cells.items(.raw)[x]` is
//     `cells[x].raw`, `cells.len` is `len(cells)`.
//   - `Row.highlights` is `std.ArrayList(Highlight)`: a
//     `[]RenderStateHighlight`. Zig's `highlights.items` is the slice.
//   - `Row.arena` (`std.heap.ArenaAllocator.State`, the arena that holds a
//     row's grapheme copies and highlights) is dropped: it is allocator
//     state, and Go's garbage collector frees. A row's graphemes and
//     highlights are ordinary Go slices.
//   - `Cell.grapheme` is `[]const u21`: a `[]uint32`.
//   - `StringMap` is `std.ArrayListUnmanaged(point.Coordinate)`: the alias
//     `RenderStateStringMap = []PointCoordinate`.
//   - `CellSet` is `std.AutoArrayHashMapUnmanaged(point.Coordinate, void)`,
//     an insertion-ordered set: `RenderStateCellSet`, a slice of keys in
//     insertion order, with stubs for the std operations render.zig and the
//     tests use under their std names (`Put`, `Deinit`, `Count`,
//     `Contains`). As for `pageListPinSet` in PageList.go, these stubs are
//     not declarations of render.zig. Its zero value is the empty set.
//   - Pins (`PageList.Pin`) are `PageListPin` values (PageList.go), not
//     tracked; `Row.raw` is a copy of the `PageRow` and `Cell.raw` a copy of
//     the `PageCell`, as in Zig.
//
// # Widths
//
// Cell coordinates are `SizeCellCountInt` (u16). `Highlight.tag` is u8 and
// so `uint8`. `u21` is `uint32`.
//
// # std.Io.Writer
//
// `string` takes a `*std.Io.Writer`. The stand-in is `*oscWriter` (osc.go),
// the package's existing stand-in for `std.Io.Writer`;
// `oscWriterAllocating` is `std.Io.Writer.Allocating`,
// `std.Io.Writer.fixed(buf)` is `oscWriterFixed(buf)` and
// `writer.buffered()` is `writer.buffered()`. The error of a failed write is
// ErrWriteFailed (osc.go); `Allocator.Error` is ErrOutOfMemory (dcs.go).
//
// # Declarations that are not Zig declarations
//
//   - `RenderStateStringMapParam`: the anonymous struct that is the payload
//     of the optional parameter `map` of `string` (named for the parameter;
//     `RenderStateStringMap` is already the name of `StringMap`);
//   - the stubs of the std operations on `RenderStateCellSet` (see above);
//   - the `String` method of `RenderStateDirty`.
//
// # Omitted
//
// Aliases and imports with nothing to declare: `std`, `assert`,
// `Allocator`, `ArenaAllocator`, `fastmem` (its `copy` is Go's `copy`),
// `color`, `cursor`, `highlight`, `point`, `size`, `page`, `PageList`,
// `Selection`, `Screen`, `ScreenSet`, `Style`, `Terminal`.
// The field `Row.arena` (see the Go-memory rule).
// No declaration is compiled out: render.zig has no
// `build_options.kitty_graphics`, tmux or oniguruma code.
// No C ABI declaration exists in this file.
//
// # What the implementation needs from files that have no Go file yet
//
// No signature in this file needs a type from an unported file, so no
// function is blocked. `string` uses
// `std.unicode.utf8CodepointSequenceLength` (Go: `unicode/utf8.RuneLen`,
// which also returns -1 above U+10FFFF; Zig accepts surrogate halves, which
// `{u}` then prints as U+FFFD) and std.Io facilities that the writer
// stand-in does not have yet (`print` with `{u}`, `writeAll`), to be added
// to osc.go with Zig's names when implemented.
//
// # Zig defaults a Go literal does not give
//
// A Go composite literal leaves an omitted field at the Go zero value. The
// only field defaults in render.zig are `RenderState.viewport_pin = null`
// and `RenderState.selection_cache = null`, which are the Go zero value, so
// no field default differs. But `RenderState.empty`, the documented
// starting state, is NOT the zero RenderState:
//   - `colors.palette` is `color.default` (ColorDefault); the zero value is
//     an all-black palette;
//   - `cursor.visual_style` is `.block` (CursorStyleBlock); the zero value
//     is CursorStyleBar;
//   - `cursor.visible` is TRUE.
// So Zig's `var state: RenderState = .empty;` is
// `state := RenderStateEmpty` in Go, never `RenderState{}`.

package terminal

// RenderState is render.RenderState.
//
// Contains the state required to render the screen, including optimizing
// for repeated render calls and only rendering dirty regions.
//
// Previously, our renderer would use `clone` to clone the screen within
// the viewport to perform rendering. This worked well enough that we kept
// it all the way up through the Ghostty 1.2.x series, but the clone time
// was repeatedly a bottleneck blocking IO.
//
// Rather than a generic clone that tries to clone all screen state per call
// (within a region), a stateful approach that optimizes for only what a
// renderer needs to do makes more sense.
//
// To use this, initialize the render state to empty, then call `update`
// on each frame to update the state to the latest terminal state.
//
//	var state: RenderState = .empty;
//	defer state.deinit(alloc);
//	state.update(alloc, &terminal);
//
// In Go: `state := RenderStateEmpty; defer state.Deinit();
// err := state.Update(&terminal)`.
//
// Note: the render state retains as much memory as possible between updates
// to prevent future allocations. If a very large frame is rendered once,
// the render state will retain that much memory until deinit. To avoid
// waste, it is recommended that the caller `deinit` and start with an
// empty render state every so often.
type RenderState struct {
	// The current screen dimensions. It is possible that these don't match
	// the renderer's current dimensions in grid cells because resizing
	// can happen asynchronously. For example, for Metal, our NSView resizes
	// at a different time than when our internal terminal state resizes.
	// This can lead to a one or two frame mismatch a renderer needs to
	// handle.
	//
	// The viewport is always exactly equal to the active area size so this
	// is also the viewport size.
	rows SizeCellCountInt
	cols SizeCellCountInt

	// The color state for the terminal.
	colors RenderStateColors

	// Cursor state within the viewport.
	cursor RenderStateCursor

	// The rows (y=0 is top) of the viewport. Guaranteed to be `rows` length.
	//
	// This is a MultiArrayList because only the update cares about
	// the allocators. Callers care about all the other properties, and
	// this better optimizes cache locality for read access for those
	// use cases.
	//
	// Zig: std.MultiArrayList(Row). Go: a slice of the struct; see the file
	// header.
	row_data []RenderStateRow

	// The dirty state of the render state. This is set by the update method.
	// The renderer/caller should set this to false when it has handled
	// the dirty state.
	dirty RenderStateDirty

	// The screen type that this state represents. This is used primarily
	// to detect changes.
	screen ScreenSetKey

	// The last viewport pin used to generate this state. This is NOT
	// a tracked pin and is generally NOT safe to read other than the direct
	// values for comparison.
	//
	// Zig: ?PageList.Pin, default null.
	viewport_pin Optional[PageListPin]

	// The cached selection so we can avoid expensive selection calculations
	// if possible.
	//
	// Zig: ?SelectionCache, default null.
	selection_cache Optional[renderStateSelectionCache]
}

// RenderStateEmpty is render.RenderState.empty.
//
// Initial state.
//
// Zig: `.{ .rows = 0, .cols = 0, .colors = .{ .background = .{},
// .foreground = .{}, .cursor = null, .palette = color.default }, .cursor =
// .{ .active = .{ .x = 0, .y = 0 }, .viewport = null, .cell = .{}, .style =
// undefined, .visual_style = .block, .password_input = false, .visible =
// true, .blinking = false }, .row_data = .empty, .dirty = .false, .screen =
// .primary }` (viewport_pin and selection_cache null by default). Three
// fields are not the Go zero value: `colors.palette` (ColorDefault),
// `cursor.visual_style` (CursorStyleBlock) and `cursor.visible` (true);
// `cursor.style` is undefined in Zig and zero in Go. Wireframe: left zero,
// as ColorDefault is in color.go; the implementation initialises it once at
// start-up.
var RenderStateEmpty RenderState

// RenderStateColors is render.RenderState.Colors.
//
// The color state for the terminal.
//
// The background/foreground will be reversed if the terminal reverse
// color mode is on! You do not need to handle that manually!
type RenderStateColors struct {
	background ColorRGB
	foreground ColorRGB
	// Zig: ?color.RGB.
	cursor  Optional[ColorRGB]
	palette ColorPalette
}

// RenderStateCursor is render.RenderState.Cursor.
type RenderStateCursor struct {
	// The x/y position of the cursor within the active area.
	active PointCoordinate

	// The x/y position of the cursor within the viewport. This
	// may be null if the cursor is not visible within the viewport.
	//
	// Zig: ?Viewport.
	viewport Optional[RenderStateCursorViewport]

	// The cell data for the cursor position. Managed memory is not
	// safe to access from this.
	cell PageCell

	// The style, always valid even if the cell is default style.
	style Style

	// The visual style of the cursor itself, such as a block or
	// bar.
	//
	// Zig: cursor.Style.
	visual_style CursorStyle

	// True if the cursor is detected to be at a password input field.
	password_input bool

	// Cursor visibility state determined by the terminal mode.
	visible bool

	// Cursor blink state determined by the terminal mode.
	blinking bool
}

// RenderStateCursorViewport is render.RenderState.Cursor.Viewport.
type RenderStateCursorViewport struct {
	// The x/y position of the cursor within the viewport.
	x SizeCellCountInt
	y SizeCellCountInt

	// Whether the cursor is part of a wide character and
	// on the tail of it. If so, some renderers may use this
	// to move the cursor back one.
	wide_tail bool
}

// RenderStateRow is render.RenderState.Row.
//
// A row within the viewport.
//
// Zig's first field, `arena: ArenaAllocator.State`, is dropped (see the
// file header). Its doc comment was: Arena used for any heap allocations
// for cell contents in this row. Importantly, this is NOT used for the
// MultiArrayList itself. We do this on purpose so that we can easily clear
// rows, but retain cached MultiArrayList capacities since grid sizes don't
// change often.
type RenderStateRow struct {
	// The page pin. This is not safe to read unless you can guarantee
	// the terminal state hasn't changed since the last `update` call.
	pin PageListPin

	// Raw row data.
	//
	// Zig: page.Row.
	raw PageRow

	// The cells in this row. Guaranteed to be `cols` length.
	//
	// Zig: std.MultiArrayList(Cell). Go: a slice of the struct; see the file
	// header.
	cells []RenderStateCell

	// A dirty flag that can be used by the renderer to track
	// its own draw state. `update` will mark this true whenever
	// this row is changed, too.
	dirty bool

	// The x range of the selection within this row.
	//
	// Zig: ?[2]size.CellCountInt.
	selection Optional[[2]SizeCellCountInt]

	// The highlights within this row.
	//
	// Zig: std.ArrayList(Highlight). Go: a slice; see the file header.
	highlights []RenderStateHighlight
}

// RenderStateHighlight is render.RenderState.Highlight.
type RenderStateHighlight struct {
	// A special tag that can be used by the caller to differentiate
	// different highlight types. The value is opaque to the RenderState.
	tag uint8

	// The x ranges of highlights within this row.
	//
	// Zig: `range: [2]size.CellCountInt`. `range` is a Go keyword.
	range_ [2]SizeCellCountInt
}

// RenderStateCell is render.RenderState.Cell.
type RenderStateCell struct {
	// Always set, this is the raw copied cell data from page.Cell.
	// The managed memory (hyperlinks, graphames, etc.) is NOT safe
	// to access from here. It is duplicated into the other fields if
	// it exists.
	//
	// Zig: page.Cell.
	raw PageCell

	// Grapheme data for the cell. This is undefined unless the
	// raw cell's content_tag is `codepoint_grapheme`.
	//
	// Zig: []const u21.
	grapheme []uint32

	// The style data for the cell. This is undefined unless
	// the style_id is non-default on raw.
	style Style
}

// RenderStateDirty is render.RenderState.Dirty.
//
// Dirty state.
//
// Zig: enum, values numbered from zero in the order listed.
type RenderStateDirty uint8

const (
	// Not dirty at all. Can skip rendering if prior state was
	// already rendered.
	RenderStateDirtyFalse RenderStateDirty = iota

	// Partially dirty. Some rows changed but not all. None of the
	// global state changed such as colors.
	RenderStateDirtyPartial

	// Fully dirty. Global state changed or dimensions changed. All rows
	// should be redrawn.
	RenderStateDirtyFull
)

// String returns the Zig tag name (@tagName).
func (self RenderStateDirty) String() string { panic(notImplemented) }

// renderStateSelectionCache is render.RenderState.SelectionCache (private
// in Zig).
type renderStateSelectionCache struct {
	selection Selection
	tl_pin    PageListPin
	br_pin    PageListPin
}

// Deinit is render.RenderState.deinit.
//
// Zig: pub fn deinit(self: *RenderState, alloc: Allocator) void. The
// allocator is dropped; the stub is kept because every test calls it.
func (self *RenderState) Deinit() { panic(notImplemented) }

// Update is render.RenderState.update.
//
// Update the render state to the latest terminal state.
//
// This will reset the terminal dirty state since it is consumed
// by this render state update.
//
// Zig: pub fn update(self: *RenderState, alloc: Allocator, t: *Terminal)
// Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *RenderState) Update(t *Terminal) error { panic(notImplemented) }

// UpdateHighlightsFlattened is render.RenderState.updateHighlightsFlattened.
//
// Update the highlights in the render state from the given flattened
// highlights. Because this uses flattened highlights, it does not require
// reading from the terminal state so it should be done outside of
// any critical sections.
//
// This will not clear any previous highlights, so the caller must
// manually clear them if desired.
//
// Zig: pub fn updateHighlightsFlattened(self: *RenderState,
// alloc: Allocator, tag: u8, hls: []const highlight.Flattened)
// Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *RenderState) UpdateHighlightsFlattened(tag uint8, hls []HighlightFlattened) error {
	panic(notImplemented)
}

// RenderStateStringMap is render.RenderState.StringMap.
//
// Zig: std.ArrayListUnmanaged(point.Coordinate). Go: an alias of the slice
// type; `.empty` is the nil slice, `map.items` is the slice.
type RenderStateStringMap = []PointCoordinate

// RenderStateStringMapParam is NOT a named Zig declaration: it is the
// anonymous struct that is the payload of the optional parameter `map` of
// `string`. Zig's first field, `alloc: Allocator`, is dropped. Zig's
// `state.string(&w, .{ .alloc = alloc, .map = &map })` is
// `state.String(&w, Some(RenderStateStringMapParam{map_: &map_}))` with
// `var map_ RenderStateStringMap`; Zig's `null` is the zero
// `Optional[RenderStateStringMapParam]`.
type RenderStateStringMapParam struct {
	// Zig: `map: *StringMap`. `map` is a Go keyword. A pointer to the
	// caller's slice; the function appends through it.
	map_ *RenderStateStringMap
}

// String is render.RenderState.string.
//
// Convert the current render state contents to a UTF-8 encoded
// string written to the given writer. This will unwrap all the wrapped
// rows. This is useful for a minimal viewport search.
//
// This currently writes empty cell contents as \x00 and writes all
// blank lines. This is fine for our current usage (link search) but
// we can adjust this later.
//
// NOTE: There is a limitation in that wrapped lines before/after
// the the top/bottom line of the viewport are not included, since
// the render state cuts them off.
//
// Zig: pub fn string(self: *const RenderState, writer: *std.Io.Writer,
// map: ?struct { alloc: Allocator, map: *StringMap })
// (Allocator.Error || std.Io.Writer.Error)!void. The writer is the stand-in
// oscWriter (osc.go); the error is ErrWriteFailed (osc.go) or
// ErrOutOfMemory (dcs.go). This is not a fmt.Stringer `String() string`:
// the Zig name `string` capitalised.
func (self *RenderState) String(writer *oscWriter, map_ Optional[RenderStateStringMapParam]) error {
	panic(notImplemented)
}

// RenderStateCellSet is render.RenderState.CellSet.
//
// A set of coordinates representing cells.
//
// Zig: std.AutoArrayHashMapUnmanaged(point.Coordinate, void): a set that
// keeps insertion order. The methods below are std's, limited to the ones
// render.zig and the tests call; they are not declarations of render.zig.
// The zero value is the empty set (`.empty`).
type RenderStateCellSet struct {
	// The keys in insertion order; what std's `keys()` returns.
	keys []PointCoordinate
}

// Put is std's put with a void value: add key if it is not present.
//
// Zig: put(self, alloc, key, {}) Allocator.Error!void. The error is
// ErrOutOfMemory (dcs.go).
func (self *RenderStateCellSet) Put(key PointCoordinate) error { panic(notImplemented) }

// Deinit is std's deinit. The allocator is dropped; kept because the tests
// call it.
func (self *RenderStateCellSet) Deinit() { panic(notImplemented) }

// Count is std's count: the number of keys.
//
// Zig: usize.
func (self RenderStateCellSet) Count() uint { panic(notImplemented) }

// Contains is std's contains: whether key is in the set.
func (self RenderStateCellSet) Contains(key PointCoordinate) bool { panic(notImplemented) }

// LinkCells is render.RenderState.linkCells.
//
// Returns a map of the cells that match to an OSC8 hyperlink over the
// given point in the render state.
//
// IMPORTANT: The terminal must not have updated since the last call to
// `update`. If there is any chance the terminal has updated, the caller
// must first call `update` again to refresh the render state.
//
// For example, you may want to hold a lock for the duration of the
// update and hyperlink lookup to ensure no updates happen in between.
//
// Zig: pub fn linkCells(self: *const RenderState, alloc: Allocator,
// viewport_point: point.Coordinate) Allocator.Error!CellSet. The error is
// ErrOutOfMemory (dcs.go).
func (self *RenderState) LinkCells(viewport_point PointCoordinate) (RenderStateCellSet, error) {
	panic(notImplemented)
}
