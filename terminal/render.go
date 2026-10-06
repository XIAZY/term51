// Ported from src/terminal/render.zig (Ghostty v1.3.1).
//
// Developer note: this is in src/terminal and not src/renderer because
// the goal is that this remains generic to multiple renderers. This can
// aid specifically with libghostty-vt with converting terminal state to
// a renderable form.
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
// `std.unicode.utf8CodepointSequenceLength`, ported as
// stdUnicodeUtf8CodepointSequenceLength (std_unicode.go): Go's
// `unicode/utf8.RuneLen` returns -1 for surrogate halves, for which Zig
// returns 3 (and `{u}` then prints U+FFFD, also 3 bytes). `print("{u}", cp)`
// is oscWriter.printUnicodeCodepoint and `writeAll` is oscWriter.writeAll
// (osc.go).
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
// `cursor.style` is undefined in Zig and zero in Go. Initialised once at
// start-up (Go's package initialisation runs ColorDefault's initialiser
// first).
var RenderStateEmpty = RenderState{
	rows: 0,
	cols: 0,
	colors: RenderStateColors{
		background: ColorRGB{},
		foreground: ColorRGB{},
		cursor:     None[ColorRGB](),
		palette:    ColorDefault,
	},
	cursor: RenderStateCursor{
		active:         PointCoordinate{x: 0, y: 0},
		viewport:       None[RenderStateCursorViewport](),
		cell:           PageCell{},
		style:          Style{}, // Zig: undefined.
		visual_style:   CursorStyleBlock,
		password_input: false,
		visible:        true,
		blinking:       false,
	},
	row_data: nil,
	dirty:    RenderStateDirtyFalse,
	screen:   ScreenSetKeyPrimary,
}

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
func (self RenderStateDirty) String() string { return renderStateDirtyNames[self] }

// renderStateDirtyNames are the tag names of RenderStateDirty, in order.
var renderStateDirtyNames = [...]string{
	"false",
	"partial",
	"full",
}

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
func (self *RenderState) Deinit() {
	// Zig: each row's arena is promoted and deinitialised and its cells
	// list deinitialised, then row_data is deinitialised. The memory is
	// Go's; the lists are left at their zero value (docs/mapping.md:
	// deinit leaves a list undefined, which is the Go zero value).
	for i := range self.row_data {
		self.row_data[i].cells = nil
	}
	self.row_data = nil
}

// Update is render.RenderState.update.
//
// Update the render state to the latest terminal state.
//
// This will reset the terminal dirty state since it is consumed
// by this render state update.
//
// Zig: pub fn update(self: *RenderState, alloc: Allocator, t: *Terminal)
// Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *RenderState) Update(t *Terminal) error {
	s := t.screens.active
	viewport_pin := s.pages.GetTopLeft(PointTagViewport)
	var redraw bool
	// redraw:
	{
		// If our screen key changed, we need to do a full rebuild
		// because our render state is viewport-specific.
		if t.screens.active_key != self.screen {
			redraw = true
			goto redraw_done
		}

		// If our terminal is dirty at all, we do a full rebuild. These
		// dirty values are full-terminal dirty values.
		//
		// Zig: the packed struct's backing integer > 0, that is any flag
		// set.
		if t.flags.dirty != (TerminalDirty{}) {
			redraw = true
			goto redraw_done
		}

		// If our screen is dirty at all, we do a full rebuild. This is
		// a full screen dirty tracker.
		//
		// Zig: the packed struct's backing integer > 0, that is any flag
		// set.
		if t.screens.active.dirty != (ScreenDirty{}) {
			redraw = true
			goto redraw_done
		}

		// If our dimensions changed, we do a full rebuild.
		if self.rows != s.pages.rows ||
			self.cols != s.pages.cols {
			redraw = true
			goto redraw_done
		}

		// If our viewport pin changed, we do a full rebuild.
		if self.viewport_pin.Valid {
			old := self.viewport_pin.Value
			if !old.Eql(viewport_pin) {
				redraw = true
				goto redraw_done
			}
		}

		redraw = false
	}
redraw_done:

	// Always set our cheap fields, its more expensive to compare
	self.rows = s.pages.rows
	self.cols = s.pages.cols
	self.viewport_pin = Some(viewport_pin)
	self.cursor.active = PointCoordinate{x: s.cursor.x, y: uint32(s.cursor.y)}
	self.cursor.cell = *s.cursor.page_cell
	self.cursor.style = s.cursor.style
	self.cursor.visual_style = s.cursor.cursor_style
	self.cursor.password_input = t.flags.password_input
	self.cursor.visible = t.modes.Get(ModesModeCursorVisible)
	self.cursor.blinking = t.modes.Get(ModesModeCursorBlinking)

	// Always reset the cursor viewport position. In the future we can
	// probably cache this by comparing the cursor pin and viewport pin
	// but may not be worth it.
	self.cursor.viewport = None[RenderStateCursorViewport]()

	// Colors.
	self.colors.cursor = t.colors.cursor.Get()
	self.colors.palette = t.colors.palette.current
	// bg_fg:
	{
		// Background/foreground can be unset initially which would
		// depend on "default" background/foreground. The expected use
		// case of Terminal is that the caller set their own configured
		// defaults on load so this doesn't happen.
		bg := t.colors.background.Get()
		if !bg.Valid {
			goto bg_fg_done
		}
		fg := t.colors.foreground.Get()
		if !fg.Valid {
			goto bg_fg_done
		}
		if t.modes.Get(ModesModeReverseColors) {
			self.colors.background = fg.Value
			self.colors.foreground = bg.Value
		} else {
			self.colors.background = bg.Value
			self.colors.foreground = fg.Value
		}
	}
bg_fg_done:

	// Ensure our row length is exactly our height, freeing or allocating
	// data as necessary. In most cases we'll have a perfectly matching
	// size.
	if len(self.row_data) != int(self.rows) {
		if len(self.row_data) < int(self.rows) {
			// Resize our rows to the desired length, marking any added
			// values undefined.
			old_len := len(self.row_data)
			self.row_data = renderResize(self.row_data, int(self.rows))

			// Initialize all our values. Its faster to use slice() + set()
			// because appendAssumeCapacity does this multiple times.
			row_data := self.row_data
			for y := old_len; y < int(self.rows); y++ {
				row_data[y] = RenderStateRow{
					// Zig: .arena = .{} (dropped).
					pin:        PageListPin{}, // Zig: undefined.
					raw:        PageRow{},     // Zig: undefined.
					cells:      nil,
					dirty:      true,
					selection:  None[[2]SizeCellCountInt](),
					highlights: nil,
				}
			}
		} else {
			row_data := self.row_data
			for i := range row_data[self.rows:] {
				// Zig: the row's arena is promoted and deinitialised and
				// its cells list deinitialised.
				row_data[int(self.rows)+i].cells = nil
			}
			self.row_data = self.row_data[:self.rows]
		}
	}

	// Break down our row data
	//
	// Zig: row_data.items(.<field>) for arena, pin, raw, cells,
	// selection, highlights and dirty. Go: row_data[y].<field>; the arena
	// is dropped.
	row_data := self.row_data

	// Track the last page that we know was dirty. This lets us
	// more quickly do the full-page dirty check.
	var last_dirty_page *Page = nil

	// Go through and setup our rows.
	row_it := s.pages.RowIterator(
		PageListDirectionRightDown,
		Point{Tag: PointTagViewport, Viewport: PointCoordinate{}},
		None[Point](),
	)
	var y SizeCellCountInt = 0
	var any_dirty bool = false
	for row_pin_ := row_it.Next(); row_pin_.Valid; row_pin_, y = row_it.Next(), renderAddOneU16(y) {
		row_pin := row_pin_.Value

		// Find our cursor if we haven't found it yet. We do this even
		// if the row is not dirty because the cursor is unrelated.
		if !self.cursor.viewport.Valid &&
			row_pin.node == s.cursor.page_pin.node &&
			row_pin.y == s.cursor.page_pin.y {
			var wide_tail bool
			if s.cursor.x > 0 {
				wide_tail = s.CursorCellLeft(1).wide == PageCellWideWide
			} else {
				wide_tail = false
			}
			self.cursor.viewport = Some(RenderStateCursorViewport{
				y: y,
				x: s.cursor.x,

				// Future: we should use our own state here to look this
				// up rather than calling this.
				wide_tail: wide_tail,
			})
		}

		// Store our pin. We have to store these even if we're not dirty
		// because dirty is only a renderer optimization. It doesn't
		// apply to memory movement. This will let us remap any cell
		// pins back to an exact entry in our RenderState.
		row_data[y].pin = row_pin

		// Get all our cells in the page.
		p := &row_pin.node.data
		page_rac := row_pin.RowAndCell()

		// dirty:
		{
			// If we're redrawing then we're definitely dirty.
			if redraw {
				goto dirty
			}

			// If our page is the same as last time then its dirty.
			if p == last_dirty_page {
				goto dirty
			}
			if p.dirty {
				// If this page is dirty then clear the dirty flag
				// of the last page and then store this one. This benchmarks
				// faster than iterating pages again later.
				if last_dirty_page != nil {
					last_dirty_page.dirty = false
				}
				last_dirty_page = p
				goto dirty
			}

			// If our row is dirty then we're dirty.
			if page_rac.row.dirty {
				goto dirty
			}

			// Not dirty!
			continue
		}
	dirty:

		// Set that at least one row was dirty.
		any_dirty = true

		// Clear our row dirty, we'll clear our page dirty later.
		// We can't clear it now because we have more rows to go through.
		page_rac.row.dirty = false

		// Promote our arena. State is copied by value so we need to
		// restore it on all exit paths so we don't leak memory.
		//
		// Zig: `var arena = row_arenas[y].promote(alloc); defer
		// row_arenas[y] = arena.state;`. The arena is dropped.

		// Reset our cells if we're rebuilding this row.
		if len(row_data[y].cells) > 0 {
			// Zig: _ = arena.reset(.retain_capacity) (dropped).
			row_data[y].cells = row_data[y].cells[:0]
			row_data[y].selection = None[[2]SizeCellCountInt]()
			row_data[y].highlights = nil
		}
		row_data[y].dirty = true

		// Get all our cells in the page.
		page_cells := p.GetCells(page_rac.row)
		assert(len(page_cells) == int(self.cols))

		// Copy our raw row data
		row_data[y].raw = *page_rac.row

		// Note: our cells MultiArrayList uses our general allocator.
		// We do this on purpose because as rows become dirty, we do
		// not want to reallocate space for cells (which are large). This
		// was a source of huge slowdown.
		//
		// Our per-row arena is only used for temporary allocations
		// pertaining to cells directly (e.g. graphemes, hyperlinks).
		cells := &row_data[y].cells
		*cells = renderResize(*cells, int(self.cols))

		// We always copy our raw cell data. In the case we have no
		// managed memory, we can skip setting any other fields.
		//
		// This is an important optimization. For plain-text screens
		// this ends up being something around 300% faster based on
		// the `screen-clone` benchmark.
		cells_slice := *cells
		// Zig: fastmem.copy(page.Cell, cells_slice.items(.raw), page_cells).
		for x := range page_cells {
			cells_slice[x].raw = page_cells[x]
		}
		if !page_rac.row.ManagedMemory() {
			continue
		}

		for x := range page_cells {
			page_cell := &page_cells[x]
			// Append assuming its a single-codepoint, styled cell
			// (most common by far).
			if page_cell.style_id > 0 {
				cells_slice[x].style = *p.styles.Get(
					page_cell.style_id,
				)
			}

			// Switch on our content tag to handle less likely cases.
			switch page_cell.content_tag {
			case PageCellContentTagCodepoint:
				// Primary codepoint goes into `raw` field.

			// If we have a multi-codepoint grapheme, look it up and
			// set our content type.
			case PageCellContentTagCodepointGrapheme:
				// Zig: arena_alloc.dupe(u21, p.lookupGrapheme(page_cell)
				// orelse &.{}).
				var src []uint32
				if g := p.LookupGrapheme(page_cell); g.Valid {
					src = g.Value
				} else {
					src = []uint32{}
				}
				dup := make([]uint32, len(src))
				copy(dup, src)
				cells_slice[x].grapheme = dup

			case PageCellContentTagBgColorRgb:
				cells_slice[x].style = Style{bg_color: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
					r: page_cell.content.color_rgb.r,
					g: page_cell.content.color_rgb.g,
					b: page_cell.content.color_rgb.b,
				}}}

			case PageCellContentTagBgColorPalette:
				cells_slice[x].style = Style{bg_color: StyleColor{
					Tag:     StyleColorPalette,
					Palette: page_cell.content.color_palette,
				}}

			default:
				panic("unreachable")
			}
		}
	}
	assert(y == self.rows)

	// If our screen has a selection, then mark the rows with the
	// selection. We do this outside of the loop above because its unlikely
	// a selection exists and because the way our selections are structured
	// today is very inefficient.
	//
	// NOTE: To improve the performance of the block below, we'll need
	// to rethink how we model selections in general.
	//
	// There are performance improvements that can be made here, though.
	// For example, `containedRow` recalculates a bunch of information
	// we can cache.
	if s.selection.Valid {
		sel := &s.selection.Value
		// selection:
		{
			// Populate our selection cache to avoid some expensive
			// recalculation.
			var cache *renderStateSelectionCache
			// cache:
			{
				if self.selection_cache.Valid {
					c := &self.selection_cache.Value
					// cache_check:
					{
						// If we're redrawing, we recalculate the cache just to
						// be safe.
						if redraw {
							goto cache_check_done
						}

						// If our selection isn't equal, we aren't cached!
						if !c.selection.Eql(*sel) {
							goto cache_check_done
						}

						// If we have no dirty rows, we can not recalculate.
						if !any_dirty {
							goto selection_done
						}

						// We have dirty rows, we can utilize the cache.
						cache = c
						goto cache_done
					}
				cache_check_done:
				}

				// Create a new cache
				tl_pin := sel.TopLeft(s)
				br_pin := sel.BottomRight(s)
				self.selection_cache = Some(renderStateSelectionCache{
					selection: SelectionInit(tl_pin, br_pin, sel.rectangle),
					tl_pin:    tl_pin,
					br_pin:    br_pin,
				})
				cache = &self.selection_cache.Value
			}
		cache_done:

			// Grab the inefficient data we need from the selection. At
			// least we can cache it.
			tl_ := s.pages.PointFromPin(PointTagScreen, cache.tl_pin)
			assert(tl_.Valid) // .?
			tl := tl_.Value.Screen
			br_ := s.pages.PointFromPin(PointTagScreen, cache.br_pin)
			assert(br_.Valid) // .?
			br := br_.Value.Screen

			// We need to determine if our selection is within the viewport.
			// The viewport is generally very small so the efficient way to
			// do this is to traverse the viewport pages and check for the
			// matching selection pages.
			for i := range row_data {
				pin := row_data[i].pin
				sel_bounds := &row_data[i].selection
				p_ := s.pages.PointFromPin(PointTagScreen, pin)
				assert(p_.Valid) // .?
				p := p_.Value.Screen
				row_sel_ := sel.ContainedRowCached(
					s,
					cache.tl_pin,
					cache.br_pin,
					pin,
					tl,
					br,
					p,
				)
				if !row_sel_.Valid {
					continue
				}
				row_sel := row_sel_.Value
				start := row_sel.Start()
				end := row_sel.End()
				assert(start.node == end.node)
				assert(start.x <= end.x)
				assert(start.y == end.y)
				*sel_bounds = Some([2]SizeCellCountInt{start.x, end.x})
			}
		}
	selection_done:
	}

	// Handle dirty state.
	if redraw {
		// Fully redraw resets some other state.
		self.screen = t.screens.active_key
		self.dirty = RenderStateDirtyFull

		// Note: we don't clear any row_data here because our rebuild
		// above did this.
	} else if any_dirty && self.dirty == RenderStateDirtyFalse {
		self.dirty = RenderStateDirtyPartial
	}

	// Finalize our final dirty page
	if last_dirty_page != nil {
		last_dirty_page.dirty = false
	}

	// Clear our dirty flags
	t.flags.dirty = TerminalDirty{}
	s.dirty = ScreenDirty{}
	return nil
}

// renderResize is std.MultiArrayList.resize on the Go slice that stands in
// for it (not a declaration of render.zig): set the length to new_len,
// keeping the capacity. As in Zig, the items between the old and the new
// length are undefined: within the capacity they keep whatever the slice
// held (Zig keeps the old bytes too), beyond it they are the Go zero value.
func renderResize[T any](list []T, new_len int) []T {
	if new_len <= cap(list) {
		return list[:new_len]
	}
	grown := make([]T, new_len)
	copy(grown, list)
	return grown
}

// renderAddOneU16 is Zig's `y + 1` on a u16, which traps on overflow (not a
// declaration of render.zig).
func renderAddOneU16(v SizeCellCountInt) SizeCellCountInt {
	if v == 0xFFFF {
		panic("integer overflow")
	}
	return v + 1
}

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
	// Fast path, we have no highlights!
	if len(hls) == 0 {
		return nil
	}

	// This is, admittedly, horrendous. This is some low hanging fruit
	// to optimize. In my defense, screens are usually small, the number
	// of highlights is usually small, and this only happens on the
	// viewport outside of a locked area. Still, I'd love to see this
	// improved someday.

	// We need to track whether any row had a match so we can mark
	// the dirty state.
	var any_dirty bool = false

	// Zig: row_data.items(.arena), (.dirty), (.pin), (.highlights). Go:
	// row_data[r].<field>; the arena is dropped.
	row_data := self.row_data
	for r := range row_data {
		row_pin := row_data[r].pin
		row_highlights := &row_data[r].highlights
		dirty := &row_data[r].dirty
		for _, hl := range hls {
			// Zig: hl.chunks.slice().items(.node), (.start), (.end).
			chunks := hl.chunks
			for i := range chunks {
				node := chunks[i].node
				// If this node doesn't match or we're not within
				// the row range, skip it.
				if node != row_pin.node ||
					row_pin.y < chunks[i].start ||
					row_pin.y >= chunks[i].end {
					continue
				}

				// We're a match!
				//
				// Zig: `var arena = row_arena.promote(alloc); defer
				// row_arena.* = arena.state;` (dropped).
				var lo, hi SizeCellCountInt
				if i == 0 &&
					row_pin.y == chunks[0].start {
					lo = hl.top_x
				} else {
					lo = 0
				}
				// Zig: `ends[nodes.len - 1] - 1` on u16 cannot trap here:
				// this chunk's end is above row_pin.y.
				if i == len(chunks)-1 &&
					row_pin.y == chunks[len(chunks)-1].end-1 {
					hi = hl.bot_x
				} else {
					// Zig: `self.cols - 1` on u16; cols > 0 when there
					// are rows.
					hi = self.cols - 1
				}
				*row_highlights = append(*row_highlights, RenderStateHighlight{
					tag:    tag,
					range_: [2]SizeCellCountInt{lo, hi},
				})

				*dirty = true
				any_dirty = true
			}
		}
	}

	// Mark our dirty state.
	if any_dirty && self.dirty == RenderStateDirtyFalse {
		self.dirty = RenderStateDirtyPartial
	}
	return nil
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
	// Zig: row_slice.items(.raw), (.cells). Go: row_data[y].raw, .cells.
	row_data := self.row_data

	for y := range row_data {
		row := row_data[y].raw
		cells := row_data[y].cells
		cells_slice := cells
		for x := range cells_slice {
			cell := cells_slice[x].raw
			graphemes := cells_slice[x].grapheme
			len__, err := stdUnicodeUtf8CodepointSequenceLength(cell.Codepoint())
			if err != nil {
				return ErrWriteFailed
			}
			var len_ uint = uint(len__)
			// Zig: try writer.print("{u}", .{cell.codepoint()}).
			if err := writer.printUnicodeCodepoint(cell.Codepoint()); err != nil {
				return err
			}
			if cell.HasGrapheme() {
				for _, cp := range graphemes {
					n, err := stdUnicodeUtf8CodepointSequenceLength(cp)
					if err != nil {
						return ErrWriteFailed
					}
					len_ += uint(n)
					// Zig: try writer.print("{u}", .{cp}).
					if err := writer.printUnicodeCodepoint(cp); err != nil {
						return err
					}
				}
			}

			if map_.Valid {
				m := map_.Value
				// Zig: @intCast(x) to u16 and @intCast(y) to u32; x < cols
				// and y < rows, so both fit.
				formatterAppendNTimes(m.map_, PointCoordinate{
					x: SizeCellCountInt(x),
					y: uint32(y),
				}, uint64(len_))
			}
		}

		if !row.wrap {
			if err := writer.writeAll([]uint8("\n")); err != nil {
				return err
			}
			if map_.Valid {
				m := map_.Value
				*m.map_ = append(*m.map_, PointCoordinate{
					x: SizeCellCountInt(len(cells_slice)),
					y: uint32(y),
				})
			}
		}
	}
	return nil
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
func (self *RenderStateCellSet) Put(key PointCoordinate) error {
	if self.Contains(key) {
		return nil
	}
	self.keys = append(self.keys, key)
	return nil
}

// Deinit is std's deinit. The allocator is dropped; kept because the tests
// call it.
func (self *RenderStateCellSet) Deinit() {
	self.keys = nil
}

// Count is std's count: the number of keys.
//
// Zig: usize.
func (self RenderStateCellSet) Count() uint {
	return uint(len(self.keys))
}

// Contains is std's contains: whether key is in the set.
func (self RenderStateCellSet) Contains(key PointCoordinate) bool {
	for _, k := range self.keys {
		if k == key {
			return true
		}
	}
	return false
}

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
	var result RenderStateCellSet = RenderStateCellSet{}
	// Zig: errdefer result.deinit(alloc) (memory only).

	// Zig: row_slice.items(.pin), (.cells). Go: row_data[y].pin, .cells.
	row_data := self.row_data

	// Our viewport point is sent in by the caller and can't be trusted.
	// If it is outside the valid area then just return empty because
	// we can't possibly have a link there.
	if viewport_point.x >= self.cols ||
		uint(viewport_point.y) >= uint(len(row_data)) {
		return result, nil
	}

	// Grab our link ID
	link_pin := row_data[viewport_point.y].pin
	link_page := &link_pin.node.data
	var link *HyperlinkPageEntry
	// link:
	{
		rac := link_page.GetRowAndCell(
			uint(viewport_point.x),
			uint(link_pin.y),
		)

		// The likely scenario is that our mouse isn't even over a link.
		if !rac.cell.hyperlink {
			return result, nil
		}

		link_id := link_page.LookupHyperlink(rac.cell)
		if !link_id.Valid {
			return result, nil
		}
		link = link_page.hyperlink_set.Get(
			link_id.Value,
		)
	}

	for y := range row_data {
		pin := row_data[y].pin
		cells := row_data[y].cells
		for x := range cells {
			cell := cells[x].raw
			if !cell.hyperlink {
				continue
			}

			other_page := &pin.node.data
			var other *HyperlinkPageEntry
			// link:
			{
				rac := other_page.GetRowAndCell(uint(x), uint(pin.y))
				link_id := other_page.LookupHyperlink(rac.cell)
				if !link_id.Valid {
					continue
				}
				other = other_page.hyperlink_set.Get(
					link_id.Value,
				)
			}

			// Zig: link.eql(link_page.memory, other, other_page.memory).
			if link.Eql(
				other,
			) {
				// Zig: @intCast(y) to u32 and @intCast(x) to u16; both fit.
				if err := result.Put(PointCoordinate{
					y: uint32(y),
					x: SizeCellCountInt(x),
				}); err != nil {
					return RenderStateCellSet{}, err
				}
			}
		}
	}

	return result, nil
}
