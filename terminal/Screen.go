// WIREFRAME for src/terminal/Screen.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: Screen.zig is a file-level struct, so the type is `Screen`
// and its constructor is `ScreenInit`. What is nested in it takes the
// prefix `Screen`: `Screen.Cursor` is `ScreenCursor`, `Screen.CursorStyle`
// is `ScreenCursorStyle`, `Screen.SavedCursor` is `ScreenSavedCursor`,
// `Screen.CharsetState` is `ScreenCharsetState`, `Screen.Options` is
// `ScreenOptions`, `Screen.Resize` is `ScreenResize`, `Screen.Scroll` is
// `ScreenScroll`, `Screen.Dirty` is `ScreenDirty`, `Screen.SemanticPrompt`
// is `ScreenSemanticPrompt`, `Screen.SemanticPrompt.SemanticClick` is
// `ScreenSemanticPromptSemanticClick`, `Screen.SelectionString` is
// `ScreenSelectionString`, `Screen.SelectLine` is `ScreenSelectLine`,
// `Screen.LineIterator` is `ScreenLineIterator`, `Screen.PromptClickMove`
// is `ScreenPromptClickMove`. The private type
// `Screen.CharsetState.CharsetArray` is `screenCharsetStateCharsetArray`;
// its methods are `pub` and so are exported (`Get`, `Set`). Private methods
// keep their Zig name (`cursorScrollAboveRotate`, `cursorDownOrScroll`,
// `cursorChangePin`, `splitForCapacity`, `startHyperlinkOnce`,
// `promptClickLine`); `pub fn` methods are exported (`CursorRight`,
// `Resize`, `Select`).
//
// Declarations appear in the same order as in Screen.zig.
//
// # Go-memory rule, option A
//
// (docs/mapping.md, governing rule 3, as refined for page.go and
// PageList.go.) Ordinary Go memory; upstream's bookkeeping is kept.
//
//   - `alloc: Allocator`, the first field of Screen, is dropped, and so is
//     every allocator parameter (`init`, `clone`, `Cursor.deinit`,
//     `selectionString`, `dumpStringAlloc`, `dumpStringAllocUnwrapped`).
//     `deinit` stubs are kept; the tests call them.
//   - `cursor.page_pin` is a `*PageListPin` tracked by `pages`;
//     `cursor.page_row` and `cursor.page_cell` are Go pointers to elements
//     of the cursor page's `rows` and `cells` slices. Where Zig steps such a
//     pointer (`cell + n`, `cell - n` in cursorCellRight, cursorCellLeft,
//     cursorRight, cursorLeft, cursorHorizontalAbsolute; `rows - n` in
//     cursorUp, cursorRowUp, cursorDown) the Go implementation indexes the
//     slice; `Page.cellIndex` and `Page.rowIndex` (page.go) give the index
//     of a pointer.
//   - `cursor.hyperlink` is `?*hyperlink.Hyperlink`, heap allocated in Zig;
//     it is a `*Hyperlink`, nil when there is none.
//   - Slices of cells (`clearCells`, `clearUnprotectedCells`) are Go
//     sub-slices of the page's `cells`.
//   - `selectionString` returns `[:0]const u8` owned by the caller: a
//     `[]uint8`. `dumpStringAlloc` and `dumpStringAllocUnwrapped` return
//     `[]const u8`: a `[]uint8`.
//   - `std.ArrayList(Pin)` (used inside selectionString) is a
//     `[]PageListPin`.
//
// # Widths
//
// As in page.go and PageList.go: every byte quantity is a `uint64` whatever
// the platform word size, counts and indexes are `uint` (Zig `usize`),
// `isize` is `int`, and fixed Zig widths stay fixed. Byte quantities, both
// `uint64`: `Options.max_scrollback` (passed to `PageListInit` as
// `max_size`) and `Options.kitty_image_storage_limit`. Counts (`uint`):
// `Scroll.row`, `PromptClickMove.left` and `right`. `u21` is `uint32`.
//
// # std.Io.Writer
//
// `dumpString` takes a `*std.Io.Writer`. The stand-in is `*oscWriter`
// (osc.go), the package's existing stand-in for `std.Io.Writer`;
// `oscWriterAllocating` is `std.Io.Writer.Allocating`, and Zig's
// `&builder.writer` is `&builder.writer`. The error of a failed write is
// ErrWriteFailed (osc.go).
//
// # Declarations that are not Zig declarations
//
//   - `ScreenCursorCopyOptions`: the anonymous struct `cursorCopy` takes;
//   - `ScreenCursorSetSemanticContent`, its tag and
//     `ScreenCursorSetSemanticContentInput`: the anonymous union(enum)
//     `cursorSetSemanticContent` takes and the anonymous enum that is the
//     payload of its `input` variant;
//   - `ScreenDumpStringOptions`: the anonymous struct `dumpString` takes;
//   - the `...Tag` types of the tagged unions and the `String` methods;
//   - `ErrUnknownPoint` and `ErrInvalidUtf8`: members of inferred error
//     sets, declared at their first use.
//
// # ScreenCursorStyle
//
// Zig: `pub const CursorStyle = @import("cursor.zig").Style;`, an alias of
// the one enum cursor.zig declares. cursor.go declares the enum
// (`CursorStyle`), its four values and its String method. The project lead
// named this declaration `ScreenCursorStyle`, so it is an alias here
// (`type ScreenCursorStyle = CursorStyle`), as `HyperlinkId` is an alias in
// hyperlink.go, and the four values keep constant aliases under the
// `ScreenCursorStyle...` names because Zig code names them through Screen.
//
// # Omitted
//
// Compiled out (`build_options.kitty_graphics` is false in the library
// build):
//   - the field `kitty_images` (line 72). With the option false its type
//     is `struct {}`; there is nothing to store. `kitty_keyboard` is NOT
//     graphics and is kept;
//   - every `if (comptime build_options.kitty_graphics)` block: in `init`
//     (setLimit), `deinit`, `reset`, `cursorDownScroll`, `scroll`,
//     `scrollClear`, `resize` (dirty flag), and the virtual placeholder
//     check in `clearCells`; and the `kitty` import's `graphics` part.
//   `Options.kitty_image_storage_limit` is an ordinary field, not under the
//   option, and is kept; with graphics compiled out nothing reads it.
// Test-only fault injection, as in PageList.go:
//   - `selectionString_tw` (line 2439, `tripwire.module(...)`) and the
//     `selectionString_tw.check(.copy_map)` call in `selectionString`.
//     There is no allocator in the Go port to fail.
// Aliases and imports with nothing to declare: `Offset`, `Page`, `Row`,
// `Cell`, `Pin` (use `PageListPin`), `ansi`, `charsets`, `fastmem`, `sgr`,
// `unicode`, `Selection`, `PageList`, `StringMap`, `ScreenFormatter`, `osc`,
// `pagepkg`, `point`, `size`, `style`, `hyperlink`, and the `log` scope
// (the port uses warn("screen", ...)).
// No C ABI declaration exists in this file.
//
// # What the implementation needs from files that have no Go file yet
//
// No signature in this file needs a type from an unported file, so nothing
// is left undeclared. Three bodies do:
//   - `selectionString` (line 2448) and `dumpString` (line 3129) are
//     written with `ScreenFormatter` (formatter.zig);
//   - `testWriteString` (line 3207) reads the width of a codepoint above
//     0xFF from `unicode.table` (src/unicode).
//
// # Build options
//
// `build_options.slow_runtime_safety` is the constant
// `pageSlowRuntimeSafety` (page.go); `assertIntegrity` does nothing without
// it. The `std.valgrind.runningOnValgrind()` check has no Go counterpart.
//
// # Zig defaults a Go literal does not give
//
// A Go composite literal leaves an omitted field at the Go zero value. The
// Zig defaults that are NOT the Go zero value, which a translated literal
// must write out:
//   - `Cursor.cursor_style` defaults to `.block`
//     (`ScreenCursorStyleBlock`); the zero value is `.bar`;
//   - `CharsetState.gr` defaults to `.G2` (`CharsetsSlotsG2`); the zero
//     value is `.G0`. So `CharsetState{}` in Zig is
//     `ScreenCharsetState{gr: CharsetsSlotsG2}` in Go;
//   - `Options.kitty_image_storage_limit` defaults to 320 * 1000 * 1000;
//   - `Resize.reflow` defaults to TRUE;
//   - `Resize.prompt_redraw` defaults to `.false`
//     (`OSCParsersSemanticPromptRedrawFalse`, value 1); the zero value is
//     `.true`. A translated `s.resize(.{ .cols = 5, .rows = 3 })` must
//     write `reflow: true, prompt_redraw: OSCParsersSemanticPromptRedrawFalse`;
//   - `SelectionString.trim` defaults to TRUE;
//   - `SelectLine.whitespace` defaults to the three codepoints 0, ' ' and
//     '\t' (`Some([]uint32{0, ' ', '\t'})`); the zero value is null, which
//     means "trim nothing";
//   - `SelectLine.semantic_prompt_boundary` defaults to TRUE;
//   - the `hyperlink` option of `cursorCopy` defaults to TRUE;
//   - the `unwrap` option of `dumpString` defaults to TRUE.
// All other defaults in this file are the Go zero value (`null`, 0, false,
// the first enum value) and are noted on the fields.

package terminal

import "errors"

// ScreenCursorStyle is Screen.CursorStyle, which is cursor.Style
// (src/terminal/cursor.zig): an alias of CursorStyle (cursor.go), where the
// enum, its values and its String method are declared. See the file header.
type ScreenCursorStyle = CursorStyle

// The values of Screen.CursorStyle, for Zig code that names them through
// Screen (`CursorStyle.block` inside Screen.zig, `Screen.CursorStyle` in
// stream_readonly.zig). Each is the CursorStyle constant of the same value.
const (
	ScreenCursorStyleBar         = CursorStyleBar
	ScreenCursorStyleBlock       = CursorStyleBlock
	ScreenCursorStyleUnderline   = CursorStyleUnderline
	ScreenCursorStyleBlockHollow = CursorStyleBlockHollow
)

// Screen is the file-level struct of Screen.zig.
//
// The Go zero value is not an initialised screen; use ScreenInit.
//
// Zig's first field, `alloc: Allocator` ("The general purpose allocator to
// use for all memory allocations. Unfortunately some screen operations do
// require allocation."), is dropped.
type Screen struct {
	// The list of pages in the screen.
	pages PageList

	// Special-case where we want no scrollback whatsoever. We have to flag
	// this because max_size 0 in PageList gets rounded up to two pages so
	// we can always have an active screen.
	//
	// Zig default: false.
	no_scrollback bool

	// The current cursor position
	cursor ScreenCursor

	// The saved cursor
	//
	// Zig: ?SavedCursor, default null.
	saved_cursor Optional[ScreenSavedCursor]

	// The selection for this screen (if any). This MUST be a tracked selection
	// otherwise the selection will become invalid. Instead of accessing this
	// directly to set it, use the `select` function which will assert and
	// automatically setup tracking.
	//
	// Zig: ?Selection, default null.
	selection Optional[Selection]

	// The charset state
	//
	// Zig default: `.{}`, which is NOT the Go zero value (`gr` is G2); see
	// ScreenCharsetState.
	charset ScreenCharsetState

	// The current or most recent protected mode. Once a protection mode is
	// set, this will never become "off" again until the screen is reset.
	// The current state of whether protection attributes should be set is
	// set on the Cell pen; this is only used to determine the most recent
	// protection mode since some sequences such as ECH depend on this.
	//
	// Zig default: .off.
	protected_mode ANSIProtectedMode

	// The kitty keyboard settings.
	//
	// Zig default: `.{}`.
	kitty_keyboard KittyKeyFlagStack

	// Screen.kitty_images (line 72), the Kitty graphics protocol state, is
	// omitted; see the file header.

	// Semantic prompt (OSC133) state.
	//
	// Zig default: .disabled (ScreenSemanticPromptDisabled).
	semantic_prompt ScreenSemanticPrompt

	// Dirty flags for the renderer.
	//
	// Zig default: `.{}`.
	dirty ScreenDirty
}

// ScreenDirty is Screen.Dirty.
//
// See Terminal.Dirty. This behaves the same way.
//
// Zig: packed struct, every field default false. In Go it is an ordinary
// struct; no non-test Zig code converts it to its integer.
type ScreenDirty struct {
	// Set when the selection is set or unset, regardless of if the
	// selection is changed or not.
	selection bool

	// When an OSC8 hyperlink is hovered, we set the full screen as dirty
	// because links can span multiple lines.
	hyperlink_hover bool
}

// ScreenSemanticPrompt is Screen.SemanticPrompt.
type ScreenSemanticPrompt struct {
	// This is flipped to true when any sort of semantic content is
	// seen. In particular, this is set to true only when a `prompt` type
	// is ever set on our cursor.
	//
	// This is used to optimize away semantic content operations if we know
	// we've never seen them.
	seen bool

	// This is set on any `cl` or `click_events` option set on the
	// most recent OSC 133 commands to specify how click handling in a
	// prompt is handling.
	click ScreenSemanticPromptSemanticClick
}

// ScreenSemanticPromptDisabled is Screen.SemanticPrompt.disabled.
var ScreenSemanticPromptDisabled = ScreenSemanticPrompt{
	seen:  false,
	click: ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickNone},
}

// ScreenSemanticPromptSemanticClick is
// Screen.SemanticPrompt.SemanticClick.
//
// Zig: union(enum) { none, click_events, cl: osc.semantic_prompt.Click }.
// Only the field named by Tag is meaningful; `none` and `click_events` have
// no payload and no field. The Go zero value is `.none`.
type ScreenSemanticPromptSemanticClick struct {
	Tag ScreenSemanticPromptSemanticClickTag

	Cl OSCParsersSemanticPromptClick
}

// ScreenSemanticPromptSemanticClickTag is the tag of
// Screen.SemanticPrompt.SemanticClick, values numbered from zero in the
// order listed.
type ScreenSemanticPromptSemanticClickTag uint8

const (
	ScreenSemanticPromptSemanticClickNone ScreenSemanticPromptSemanticClickTag = iota
	ScreenSemanticPromptSemanticClickClickEvents
	ScreenSemanticPromptSemanticClickCl
)

// String returns the Zig tag name (@tagName).
func (self ScreenSemanticPromptSemanticClickTag) String() string { panic(notImplemented) }

// ScreenCursor is Screen.Cursor: the cursor position and style.
type ScreenCursor struct {
	// The x/y position within the active area.
	//
	// Zig default: 0.
	x SizeCellCountInt
	// Zig default: 0.
	y SizeCellCountInt

	// The visual style of the cursor. This defaults to block because
	// it has to default to something, but users of this struct are
	// encouraged to set their own default.
	//
	// Zig default: .block, which is NOT the Go zero value (.bar); see the
	// file header.
	cursor_style ScreenCursorStyle

	// The "last column flag (LCF)" as its called. If this is set then the
	// next character print will force a soft-wrap.
	//
	// Zig default: false.
	pending_wrap bool

	// The protected mode state of the cursor. If this is true then
	// all new characters printed will have the protected state set.
	//
	// Zig default: false.
	protected bool

	// The currently active style. This is the concrete style value
	// that should be kept up to date. The style ID to use for cell writing
	// is below.
	//
	// Zig default: `.{}`.
	style Style

	// The currently active style ID. The style is page-specific so when
	// we change pages we need to ensure that we update that page with
	// our style when used.
	//
	// Zig default: style.default_id (StyleDefaultId, 0).
	style_id StyleId

	// The hyperlink ID that is currently active for the cursor. A value
	// of zero means no hyperlink is active. (Implements OSC8, saying that
	// so code search can find it.).
	//
	// Zig default: 0.
	hyperlink_id HyperlinkId

	// This is the implicit ID to use for hyperlinks that don't specify
	// an ID. We do an overflowing add to this so repeats can technically
	// happen with carefully crafted inputs but for real workloads its
	// highly unlikely -- and the fix is for the TUI program to use explicit
	// IDs.
	//
	// Zig default: 0.
	hyperlink_implicit_id SizeOffsetInt

	// Heap-allocated hyperlink state so that we can recreate it when
	// the cursor page pin changes. We can't get it from the old screen
	// state because the page may be cleared. This is heap allocated
	// because its most likely null.
	//
	// Zig: ?*hyperlink.Hyperlink, default null.
	hyperlink *Hyperlink

	// The current semantic content type for the cursor that will be
	// applied to any newly written cells.
	//
	// Zig default: .output.
	semantic_content PageCellSemanticContent
	// Zig default: false.
	semantic_content_clear_eol bool

	// The pointers into the page list where the cursor is currently
	// located. This makes it faster to move the cursor.
	page_pin  *PageListPin
	page_row  *PageRow
	page_cell *PageCell
}

// Deinit is Screen.Cursor.deinit.
//
// Zig: pub fn deinit(self: *Cursor, alloc: Allocator) void. The allocator
// parameter is dropped. It deinitialises and frees the cursor's hyperlink,
// if any; the stub is kept because the tests call it.
func (self *ScreenCursor) Deinit() { panic(notImplemented) }

// ScreenSavedCursor is Screen.SavedCursor: saved cursor state.
type ScreenSavedCursor struct {
	x            SizeCellCountInt
	y            SizeCellCountInt
	style        Style
	protected    bool
	pending_wrap bool
	origin       bool
	charset      ScreenCharsetState
}

// ScreenCharsetState is Screen.CharsetState: state required for all
// charset operations.
//
// The Go zero value is NOT Zig's `.{}`: `gr` defaults to G2. See the file
// header.
type ScreenCharsetState struct {
	// The list of graphical charsets by slot
	//
	// Zig default: `.{}`, all four slots `.utf8`, which is the Go zero
	// value.
	charsets screenCharsetStateCharsetArray

	// GL is the slot to use when using a 7-bit printable char (up to 127)
	// GR used for 8-bit printable chars.
	//
	// Zig default: .G0.
	gl CharsetsSlots
	// Zig default: .G2, which is NOT the Go zero value.
	gr CharsetsSlots

	// Single shift where a slot is used for exactly one char.
	//
	// Zig: ?charsets.Slots, default null.
	single_shift Optional[CharsetsSlots]
}

// screenCharsetStateCharsetArray is Screen.CharsetState.CharsetArray
// (private in Zig).
//
// An array to map a charset slot to a lookup table.
//
// We use this bespoke struct instead of `std.EnumArray` because
// accessing these slots is very performance critical since it's
// done for every single print. This benchmarks faster.
type screenCharsetStateCharsetArray struct {
	// Zig default for each: .utf8.
	g0 CharsetsCharset
	g1 CharsetsCharset
	g2 CharsetsCharset
	g3 CharsetsCharset
}

// Get is Screen.CharsetState.CharsetArray.get.
//
// Zig: pub inline fn get(self: *const CharsetArray, slot: charsets.Slots)
// charsets.Charset.
func (self *screenCharsetStateCharsetArray) Get(slot CharsetsSlots) CharsetsCharset {
	panic(notImplemented)
}

// Set is Screen.CharsetState.CharsetArray.set.
//
// Zig: pub inline fn set(self: *CharsetArray, slot: charsets.Slots,
// charset: charsets.Charset) void.
func (self *screenCharsetStateCharsetArray) Set(slot CharsetsSlots, charset CharsetsCharset) {
	panic(notImplemented)
}

// ScreenOptions is Screen.Options.
type ScreenOptions struct {
	cols SizeCellCountInt
	rows SizeCellCountInt

	// The maximum size of scrollback in bytes. Zero means unlimited. Any
	// other value will be clamped to support a minimum of the active area.
	//
	// Zig: usize, default 0. Bytes: it is passed to PageList.init as
	// max_size. (Whatever the comment above says, `init` treats zero as "no
	// scrollback": it sets `no_scrollback` and passes a max_size of 0.)
	max_scrollback uint64

	// The total storage limit for Kitty images in bytes for this
	// screen. Kitty image storage is per-screen.
	//
	// Zig: usize, default 320 * 1000 * 1000 (320MB), which is NOT the Go
	// zero value. Bytes. Nothing reads it in the library build (Kitty
	// graphics are compiled out).
	kitty_image_storage_limit uint64
}

// ScreenOptionsDefault is Screen.Options.default.
//
// A simple, default terminal. If you rely on specific dimensions or
// scrollback (or lack of) then do not use this directly. This is just
// for callers that need some defaults.
var ScreenOptionsDefault = ScreenOptions{
	cols:                      80,
	rows:                      24,
	max_scrollback:            0,
	kitty_image_storage_limit: 320 * 1000 * 1000,
}

// ScreenInit is Screen.init.
//
// Initialize a new screen.
//
// max_scrollback is the amount of scrollback to keep in bytes. This
// will be rounded UP to the nearest page size because our minimum allocation
// size is that anyways.
//
// If max scrollback is 0, then no scrollback is kept at all.
//
// Zig: pub fn init(alloc: Allocator, opts: Options) Allocator.Error!Screen.
// The allocator parameter is dropped. The error is ErrOutOfMemory (dcs.go).
func ScreenInit(opts ScreenOptions) (Screen, error) { panic(notImplemented) }

// Deinit is Screen.deinit.
//
// Kept although it mostly frees: it deinitialises the cursor and the page
// list (which asserts integrity), and every test calls it.
func (self *Screen) Deinit() { panic(notImplemented) }

// AssertIntegrity is Screen.assertIntegrity.
//
// Assert that the screen is in a consistent state. This doesn't check
// all pages in the page list because that is SO SLOW even just for
// tests. This only asserts the screen specific data so callers should
// ensure they're also calling page integrity checks if necessary.
//
// Zig: self is *const Screen.
func (self *Screen) AssertIntegrity() { panic(notImplemented) }

// Reset is Screen.reset.
//
// Reset the screen according to the logic of a DEC RIS sequence.
//
//   - Clears the screen and attempts to reclaim memory.
//   - Moves the cursor to the top-left.
//   - Clears any cursor state: style, hyperlink, etc.
//   - Resets the charset
//   - Clears the selection
//   - Deletes all Kitty graphics
//   - Resets Kitty Keyboard settings
//   - Disables protection mode
func (self *Screen) Reset() { panic(notImplemented) }

// Clone is Screen.clone.
//
// Clone the screen.
//
// This will copy:
//
//   - Screen dimensions
//   - Screen data (cell state, etc.) for the region
//
// Anything not mentioned above is NOT copied. Some of this is for
// very good reason:
//
//   - Kitty images have a LOT of data. This is not efficient to copy.
//     Use a lock and access the image data. The dirty bit is there for
//     a reason.
//   - Cursor location can be expensive to calculate with respect to the
//     specified region. It is faster to grab the cursor from the old
//     screen and then move it to the new screen.
//   - Current hyperlink cursor state has heap allocations. Since clone
//     is only for read-only operations, it is better to not have any
//     hyperlink state. Note that already-written hyperlinks are cloned.
//
// If not mentioned above, then there isn't a specific reason right now
// to not copy some data other than we probably didn't need it and it
// isn't necessary for screen coherency.
//
// Other notes:
//
//   - The viewport will always be set to the active area of the new
//     screen. This is the bottom "rows" rows.
//   - If the clone region is smaller than a viewport area, blanks will
//     be filled in at the bottom.
//
// Zig: pub fn clone(self: *const Screen, alloc: Allocator, top: point.Point,
// bot: ?point.Point) !Screen. The allocator parameter is dropped. The
// inferred error set is that of PageList.clone and PageList.trackPin:
// ErrOutOfMemory (dcs.go) and Page.CloneFromError (page.go).
func (self *Screen) Clone(top Point, bot Optional[Point]) (Screen, error) {
	panic(notImplemented)
}

// IncreaseCapacity is Screen.increaseCapacity.
//
// Zig: pub fn increaseCapacity(self: *Screen, node: *PageList.List.Node,
// adjustment: ?PageList.IncreaseCapacity)
// PageList.IncreaseCapacityError!*PageList.List.Node. The error is
// ErrOutOfMemory (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) IncreaseCapacity(
	node *PageListNode,
	adjustment Optional[PageListIncreaseCapacity],
) (*PageListNode, error) {
	panic(notImplemented)
}

// CursorCellRight is Screen.cursorCellRight.
//
// Zig: pub inline fn cursorCellRight(self: *Screen, n: size.CellCountInt)
// *pagepkg.Cell. The result points into the cursor row's cells.
func (self *Screen) CursorCellRight(n SizeCellCountInt) *PageCell { panic(notImplemented) }

// CursorCellLeft is Screen.cursorCellLeft.
//
// Zig: pub inline fn cursorCellLeft(self: *Screen, n: size.CellCountInt)
// *pagepkg.Cell. The result points into the cursor row's cells.
func (self *Screen) CursorCellLeft(n SizeCellCountInt) *PageCell { panic(notImplemented) }

// CursorCellEndOfPrev is Screen.cursorCellEndOfPrev.
func (self *Screen) CursorCellEndOfPrev() *PageCell { panic(notImplemented) }

// CursorRight is Screen.cursorRight.
//
// Move the cursor right. This is a specialized function that is very fast
// if the caller can guarantee we have space to move right (no wrapping).
func (self *Screen) CursorRight(n SizeCellCountInt) { panic(notImplemented) }

// CursorLeft is Screen.cursorLeft.
//
// Move the cursor left.
func (self *Screen) CursorLeft(n SizeCellCountInt) { panic(notImplemented) }

// CursorUp is Screen.cursorUp.
//
// Move the cursor up.
//
// Precondition: The cursor is not at the top of the screen.
func (self *Screen) CursorUp(n SizeCellCountInt) { panic(notImplemented) }

// CursorRowUp is Screen.cursorRowUp.
//
// Zig: pub fn cursorRowUp(self: *Screen, n: size.CellCountInt)
// *pagepkg.Row.
func (self *Screen) CursorRowUp(n SizeCellCountInt) *PageRow { panic(notImplemented) }

// CursorDown is Screen.cursorDown.
//
// Move the cursor down.
//
// Precondition: The cursor is not at the bottom of the screen.
func (self *Screen) CursorDown(n SizeCellCountInt) { panic(notImplemented) }

// CursorHorizontalAbsolute is Screen.cursorHorizontalAbsolute.
//
// Move the cursor to some absolute horizontal position.
func (self *Screen) CursorHorizontalAbsolute(x SizeCellCountInt) { panic(notImplemented) }

// CursorAbsolute is Screen.cursorAbsolute.
//
// Move the cursor to some absolute position.
func (self *Screen) CursorAbsolute(x SizeCellCountInt, y SizeCellCountInt) {
	panic(notImplemented)
}

// CursorReload is Screen.cursorReload.
//
// Reloads the cursor pointer information into the screen. This is expensive
// so it should only be done in cases where the pointers are invalidated
// in such a way that its difficult to recover otherwise.
func (self *Screen) CursorReload() { panic(notImplemented) }

// CursorDownScroll is Screen.cursorDownScroll.
//
// Scroll the active area and keep the cursor at the bottom of the screen.
// This is a very specialized function but it keeps it fast.
//
// Zig: !void. The inferred error set is that of PageList.eraseRow and
// PageList.grow: ErrOutOfMemory (dcs.go) and Page.CloneFromError (page.go).
func (self *Screen) CursorDownScroll() error { panic(notImplemented) }

// CursorScrollAbove is Screen.cursorScrollAbove.
//
// This scrolls the active area at and above the cursor.
// The lines below the cursor are not scrolled.
//
// Zig: !void. The inferred error set is that of cursorDownScroll,
// PageList.grow and cursorScrollAboveRotate: ErrOutOfMemory (dcs.go) and
// Page.CloneFromError (page.go).
func (self *Screen) CursorScrollAbove() error { panic(notImplemented) }

// cursorScrollAboveRotate is Screen.cursorScrollAboveRotate.
//
// Zig: !void. The inferred error set is that of Page.cloneRowFrom:
// Page.CloneFromError (page.go).
func (self *Screen) cursorScrollAboveRotate() error { panic(notImplemented) }

// cursorDownOrScroll is Screen.cursorDownOrScroll.
//
// Move the cursor down if we're not at the bottom of the screen. Otherwise
// scroll. Currently only used for testing.
//
// Zig: !void. The inferred error set is that of cursorDownScroll.
func (self *Screen) cursorDownOrScroll() error { panic(notImplemented) }

// ScreenCursorCopyOptions is NOT a named Zig declaration: it is the
// anonymous struct that is the `opts` parameter of Screen.cursorCopy.
type ScreenCursorCopyOptions struct {
	// Copy the hyperlink from the other cursor. If not set, this will
	// clear our current hyperlink.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default (`.{}`) must set it.
	hyperlink bool
}

// CursorCopy is Screen.cursorCopy.
//
// Copy another cursor. The cursor can be on any screen but the x/y
// must be within our screen bounds.
//
// Zig: pub fn cursorCopy(self: *Screen, other: Cursor, opts: struct {
// hyperlink: bool = true }) !void. The inferred error set is that of
// manualStyleUpdate: ErrOutOfMemory (dcs.go) or ErrOutOfSpace
// (PageList.go).
func (self *Screen) CursorCopy(other ScreenCursor, opts ScreenCursorCopyOptions) error {
	panic(notImplemented)
}

// cursorChangePin is Screen.cursorChangePin.
//
// Always use this to write to cursor.page_pin.*.
//
// This specifically handles the case when the new pin is on a different
// page than the old AND we have a style or hyperlink set. In that case,
// we must release our old one and insert the new one, since styles are
// stored per-page.
//
// Note that this can change the cursor pin AGAIN if the process of
// setting up our cursor forces a capacity adjustment of the underlying
// cursor page, so any references to the page pin should be re-read
// from `self.cursor.page_pin` after calling this.
//
// Zig: inline fn cursorChangePin(self: *Screen, new: Pin) void.
func (self *Screen) cursorChangePin(new PageListPin) { panic(notImplemented) }

// CursorMarkDirty is Screen.cursorMarkDirty.
//
// Mark the cursor position as dirty.
// TODO: test
func (self *Screen) CursorMarkDirty() { panic(notImplemented) }

// CursorResetWrap is Screen.cursorResetWrap.
//
// Reset the cursor row's soft-wrap state and the cursor's pending wrap.
// Also handles clearing the spacer head on the cursor row and resetting
// the wrap_continuation flag on the next row if necessary.
//
// NOTE(qwerasd): This method is not scrolling region aware, and cannot be
// since it's on Screen not Terminal. This needs to be addressed down the
// line. Not an extremely urgent issue since it's an edge case of an edge
// case, but not ideal.
func (self *Screen) CursorResetWrap() { panic(notImplemented) }

// ScreenScroll is Screen.Scroll.
//
// Options for scrolling the viewport of the terminal grid. The reason
// we have this in addition to PageList.Scroll is because we have additional
// scroll behaviors that are not part of the PageList.Scroll enum.
//
// Zig: union(enum) { active, top, pin: Pin, row: usize, delta_row: isize,
// delta_prompt: isize }. Only the field named by Tag is meaningful;
// `active` and `top` have no payload and no field. The Go zero value is
// `.active`.
type ScreenScroll struct {
	Tag ScreenScrollTag

	// For all of these, see PageList.Scroll.
	Pin PageListPin
	// Zig: usize. A count of rows.
	Row uint
	// Zig: isize.
	DeltaRow int
	// Zig: isize.
	DeltaPrompt int
}

// ScreenScrollTag is the tag of Screen.Scroll, values numbered from zero
// in the order listed.
type ScreenScrollTag uint8

const (
	ScreenScrollActive ScreenScrollTag = iota
	ScreenScrollTop
	ScreenScrollPin
	ScreenScrollRow
	ScreenScrollDeltaRow
	ScreenScrollDeltaPrompt
)

// String returns the Zig tag name (@tagName).
func (self ScreenScrollTag) String() string { panic(notImplemented) }

// Scroll is Screen.scroll.
//
// Scroll the viewport of the terminal grid.
func (self *Screen) Scroll(behavior ScreenScroll) { panic(notImplemented) }

// ScrollClear is Screen.scrollClear.
//
// See PageList.scrollClear. In addition to that, we reset the cursor
// to be on top.
//
// Zig: !void. The inferred error set is that of PageList.scrollClear:
// ErrOutOfMemory (dcs.go).
func (self *Screen) ScrollClear() error { panic(notImplemented) }

// ViewportIsBottom is Screen.viewportIsBottom.
//
// Returns true if the viewport is scrolled to the bottom of the screen.
//
// Zig: pub inline fn viewportIsBottom(self: Screen) bool. Zig passes the
// screen by value; Go uses a pointer receiver like every other method of
// Screen (a Screen is not meant to be copied).
func (self *Screen) ViewportIsBottom() bool { panic(notImplemented) }

// EraseRows is Screen.eraseRows.
//
// Erase the region specified by tl and br, inclusive. This will physically
// erase the rows meaning the memory will be reclaimed (if the underlying
// page is empty) and other rows will be shifted up.
//
// Zig: pub inline fn eraseRows(self: *Screen, tl: point.Point,
// bl: ?point.Point) void.
func (self *Screen) EraseRows(tl Point, bl Optional[Point]) { panic(notImplemented) }

// ClearRows is Screen.clearRows.
//
// Clear the region specified by tl and bl, inclusive. Cleared cells are
// colored with the current style background color. This will clear all
// cells in the rows.
//
// If protected is true, the protected flag will be respected and only
// unprotected cells will be cleared. Otherwise, all cells will be cleared.
//
// Zig: pub fn clearRows(self: *Screen, tl: point.Point, bl: ?point.Point,
// protected: bool) void.
func (self *Screen) ClearRows(tl Point, bl Optional[Point], protected bool) {
	panic(notImplemented)
}

// ClearCells is Screen.clearCells.
//
// Clear the cells with the blank cell.
//
// This takes care to handle cleaning up graphemes and styles.
//
// Zig: pub fn clearCells(self: *Screen, page: *Page, row: *Row,
// cells: []Cell) void. `cells` is a sub-slice of the page's cells.
func (self *Screen) ClearCells(page *Page, row *PageRow, cells []PageCell) {
	panic(notImplemented)
}

// ClearUnprotectedCells is Screen.clearUnprotectedCells.
//
// Clear cells but only if they are not protected.
//
// Zig: pub fn clearUnprotectedCells(self: *Screen, page: *Page, row: *Row,
// cells: []Cell) void. `cells` is a sub-slice of the page's cells.
func (self *Screen) ClearUnprotectedCells(page *Page, row *PageRow, cells []PageCell) {
	panic(notImplemented)
}

// SplitCellBoundary is Screen.splitCellBoundary.
//
// Clean up boundary conditions where a cell will become discontiguous with
// a neighboring cell because either one of them will be moved and/or cleared.
//
// For performance reasons this is specialized to operate on the cursor row.
//
// Handles the boundary between the cell at `x` and the cell at `x - 1`.
//
// So, for example, when moving a region of cells [a, b] (inclusive), call this
// function with `x = a` and `x = b + 1`. It is okay if `x` is out of bounds by
// 1, this will be interpreted correctly.
//
// DOES NOT MODIFY ROW WRAP STATE! See `cursorResetWrap` for that.
//
// The following boundary conditions are handled:
//
//   - `x - 1` is a wide character and `x` is a spacer tail:
//     o Both cells will be cleared.
//     o If `x - 1` is the start of the row and was wrapped from a previous row
//     then the previous row is checked for a spacer head, which is cleared if
//     present.
//
//   - `x == 0` and is a wide character:
//     o If the row is a wrap continuation then the previous row will be checked
//     for a spacer head, which is cleared if present.
//
//   - `x == cols` and `x - 1` is a spacer head:
//     o `x - 1` will be cleared.
//
// NOTE(qwerasd): This method is not scrolling region aware, and cannot be
// since it's on Screen not Terminal. This needs to be addressed down the
// line. Not an extremely urgent issue since it's an edge case of an edge
// case, but not ideal.
func (self *Screen) SplitCellBoundary(x SizeCellCountInt) { panic(notImplemented) }

// BlankCell is Screen.blankCell.
//
// Returns the blank cell to use when doing terminal operations that
// require preserving the bg color.
//
// Zig: self is *const Screen.
func (self *Screen) BlankCell() PageCell { panic(notImplemented) }

// ScreenResize is Screen.Resize: the options of resize.
type ScreenResize struct {
	// The new size to resize to
	cols SizeCellCountInt
	rows SizeCellCountInt

	// Whether to reflow soft-wrapped text.
	//
	// This will reflow soft-wrapped text. If the screen size is getting
	// smaller and the maximum scrollback size is exceeded, data will be
	// lost from the top of the scrollback.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default must set it.
	reflow bool

	// Set this to enable prompt redraw on resize. This signals
	// that the running program can redraw the prompt if the cursor is
	// currently at a prompt. This detects OSC133 prompts lines and clears
	// them. If set to `.last`, only the most recent prompt line is cleared.
	//
	// Zig default: .false (OSCParsersSemanticPromptRedrawFalse). The Go
	// zero value is .true; a caller that relies on the default must set
	// it.
	prompt_redraw OSCParsersSemanticPromptRedraw
}

// Resize is Screen.resize.
//
// Resize the screen. The rows or cols can be bigger or smaller.
//
// If this returns an error, the screen is left in a likely garbage state.
// It is very hard to undo this operation without blowing up our memory
// usage. The only way to recover is to reset the screen. The only way
// this really fails is if page allocation is required and fails, which
// probably means the system is in trouble anyways. I'd like to improve this
// in the future but it is not a priority particularly because this scenario
// (resize) is difficult.
//
// Zig: pub inline fn resize(self: *Screen, opts: Resize) !void. The
// inferred error set is that of PageList.trackPin and PageList.resize:
// ErrOutOfMemory (dcs.go).
func (self *Screen) Resize(opts ScreenResize) error { panic(notImplemented) }

// SetAttribute is Screen.setAttribute.
//
// Set a style attribute for the current cursor.
//
// If the style can't be set due to any internal errors (memory-related),
// then this will revert back to the existing style and return an error.
//
// Zig: pub fn setAttribute(self: *Screen, attr: sgr.Attribute)
// PageList.IncreaseCapacityError!void. The error is ErrOutOfMemory
// (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) SetAttribute(attr SGRAttribute) error { panic(notImplemented) }

// ManualStyleUpdate is Screen.manualStyleUpdate.
//
// Call this whenever you manually change the cursor style.
//
// This function can NOT fail if the cursor style is changing to the
// default style.
//
// If this returns an error, the style change did not take effect and
// the cursor style is reverted back to the default. The only scenario
// this returns an error is if there is a physical memory allocation failure
// or if there is no possible way to increase style capacity to store
// the style.
//
// This function WILL split pages as necessary to accommodate the new style.
// So if OutOfSpace is returned, it means that even after splitting the page
// there was still no room for the new style.
//
// Zig: PageList.IncreaseCapacityError!void. The error is ErrOutOfMemory
// (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) ManualStyleUpdate() error { panic(notImplemented) }

// splitForCapacity is Screen.splitForCapacity.
//
// Split at the given pin so that the pinned row moves to the page
// with less used capacity after the split.
//
// The primary use case for this is to handle IncreaseCapacityError
// OutOfSpace conditions where we need to split the page in order
// to make room for more managed memory.
//
// If the caller cares about where the pin moves to, they should
// setup a tracked pin before calling this and then check that.
// In many calling cases, the input pin is tracked (e.g. the cursor
// pin).
//
// If this returns OOM then its a system OOM. If this returns OutOfSpace
// then it means the page can't be split further.
//
// Zig: fn splitForCapacity(self: *Screen, pin: Pin)
// PageList.SplitError!void. The error is ErrOutOfMemory (dcs.go) or
// ErrOutOfSpace (PageList.go).
func (self *Screen) splitForCapacity(pin PageListPin) error { panic(notImplemented) }

// AppendGrapheme is Screen.appendGrapheme.
//
// Append a grapheme to the given cell within the current cursor row.
//
// Zig: pub fn appendGrapheme(self: *Screen, cell: *Cell, cp: u21)
// PageList.IncreaseCapacityError!void. The error is ErrOutOfMemory
// (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) AppendGrapheme(cell *PageCell, cp uint32) error { panic(notImplemented) }

// StartHyperlink is Screen.startHyperlink.
//
// Start the hyperlink state. Future cells will be marked as hyperlinks with
// this state. Note that various terminal operations may clear the hyperlink
// state, such as switching screens (alt screen).
//
// Zig: pub fn startHyperlink(self: *Screen, uri: []const u8,
// id_: ?[]const u8) PageList.IncreaseCapacityError!void. The error is
// ErrOutOfMemory (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) StartHyperlink(uri []uint8, id_ Optional[[]uint8]) error {
	panic(notImplemented)
}

// startHyperlinkOnce is Screen.startHyperlinkOnce.
//
// This is like startHyperlink but if we have to adjust page capacities
// this returns error.PageAdjusted. This is useful so that we unwind
// all the previous state and try again.
//
// Zig: fn startHyperlinkOnce(self: *Screen, source: hyperlink.Hyperlink)
// (Allocator.Error || Page.InsertHyperlinkError)!void. The error is
// ErrOutOfMemory (dcs.go) or one of the Page.InsertHyperlinkError
// sentinels (page.go): ErrStringsOutOfMemory, ErrSetOutOfMemory,
// ErrSetNeedsRehash. (No `PageAdjusted` error exists in this version; the
// comment above is upstream's.)
func (self *Screen) startHyperlinkOnce(source Hyperlink) error { panic(notImplemented) }

// EndHyperlink is Screen.endHyperlink.
//
// End the hyperlink state so that future cells aren't part of the
// current hyperlink (if any). This is safe to call multiple times.
func (self *Screen) EndHyperlink() { panic(notImplemented) }

// CursorSetHyperlink is Screen.cursorSetHyperlink.
//
// Set the current hyperlink state on the current cell.
//
// Zig: PageList.IncreaseCapacityError!void. The error is ErrOutOfMemory
// (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) CursorSetHyperlink() error { panic(notImplemented) }

// ScreenCursorSetSemanticContent is NOT a named Zig declaration: it is the
// anonymous union that is the `t` parameter of
// Screen.cursorSetSemanticContent.
//
// Zig: union(enum) { prompt: osc.semantic_prompt.PromptKind, output,
// input: enum { clear_explicit, clear_eol } }. Only the field named by Tag
// is meaningful; `output` has no payload and no field. The Go zero value
// is `.prompt` with kind `.initial`.
type ScreenCursorSetSemanticContent struct {
	Tag ScreenCursorSetSemanticContentTag

	Prompt OSCParsersSemanticPromptPromptKind
	Input  ScreenCursorSetSemanticContentInput
}

// ScreenCursorSetSemanticContentTag is the tag of the union above, values
// numbered from zero in the order listed. The constants are
// `ScreenCursorSetSemanticContentTag<Variant>` because
// `ScreenCursorSetSemanticContentInput` is the name of a payload type.
type ScreenCursorSetSemanticContentTag uint8

const (
	ScreenCursorSetSemanticContentTagPrompt ScreenCursorSetSemanticContentTag = iota
	ScreenCursorSetSemanticContentTagOutput
	ScreenCursorSetSemanticContentTagInput
)

// String returns the Zig tag name (@tagName).
func (self ScreenCursorSetSemanticContentTag) String() string { panic(notImplemented) }

// ScreenCursorSetSemanticContentInput is NOT a named Zig declaration: it is
// the anonymous enum that is the payload of the `input` variant above.
//
// Zig: enum { clear_explicit, clear_eol }, values numbered from zero in
// the order listed.
type ScreenCursorSetSemanticContentInput uint8

const (
	ScreenCursorSetSemanticContentInputClearExplicit ScreenCursorSetSemanticContentInput = iota
	ScreenCursorSetSemanticContentInputClearEol
)

// String returns the Zig tag name (@tagName).
func (self ScreenCursorSetSemanticContentInput) String() string { panic(notImplemented) }

// CursorSetSemanticContent is Screen.cursorSetSemanticContent.
//
// Modify the semantic content type of the cursor. This should
// be preferred over setting it manually since it handles all the
// proper accounting.
func (self *Screen) CursorSetSemanticContent(t ScreenCursorSetSemanticContent) {
	panic(notImplemented)
}

// Select is Screen.select.
//
// Set the selection to the given selection. If this is a tracked selection
// then the screen will take ownership of the selection. If this is untracked
// then the screen will convert it to tracked internally. This will automatically
// untrack the prior selection (if any).
//
// Set the selection to null to clear any previous selection.
//
// This is always recommended over setting `selection` directly. Beyond
// managing memory for you, it also performs safety checks that the selection
// is always tracked.
//
// Zig: pub fn select(self: *Screen, sel_: ?Selection) Allocator.Error!void.
// The error is ErrOutOfMemory (dcs.go).
func (self *Screen) Select(sel_ Optional[Selection]) error { panic(notImplemented) }

// ClearSelection is Screen.clearSelection.
//
// Same as select(null) but can't fail.
func (self *Screen) ClearSelection() { panic(notImplemented) }

// ScreenSelectionString is Screen.SelectionString: the options of
// selectionString.
type ScreenSelectionString struct {
	// The selection to convert to a string.
	sel Selection

	// If true, trim whitespace around the selection.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default must set it.
	trim bool

	// If non-null, a stringmap will be written here. This will use
	// the same allocator as the call to selectionString. The string will
	// be duplicated here and in the return value so both must be freed.
	//
	// Zig: `map: ?*StringMap`, default null. `map` is a Go keyword.
	map_ *StringMap
}

// Screen.selectionString_tw (line 2439) is omitted; see the file header.

// SelectionString is Screen.selectionString.
//
// Returns the raw text associated with a selection. This will unwrap
// soft-wrapped edges. The returned slice is owned by the caller and allocated
// using alloc, not the allocator associated with the screen (unless they match).
//
// For more flexibility, use a ScreenFormatter directly.
//
// Zig: pub fn selectionString(self: *Screen, alloc: Allocator,
// opts: SelectionString) Allocator.Error![:0]const u8. The allocator
// parameter is dropped. The error is ErrOutOfMemory (dcs.go).
func (self *Screen) SelectionString(opts ScreenSelectionString) ([]uint8, error) {
	panic(notImplemented)
}

// ScreenSelectLine is Screen.SelectLine: the options of selectLine.
type ScreenSelectLine struct {
	// The pin of some part of the line to select.
	pin PageListPin

	// These are the codepoints to consider whitespace to trim
	// from the ends of the selection.
	//
	// Zig: ?[]const u21, default `&.{ 0, ' ', '\t' }`, which is NOT the Go
	// zero value (null); a caller that relies on the default must write
	// `Some([]uint32{0, ' ', '\t'})`.
	whitespace Optional[[]uint32]

	// If true, line selection will consider semantic prompt
	// state changing a boundary. State changing is ANY state
	// change.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default must set it.
	semantic_prompt_boundary bool
}

// SelectLine is Screen.selectLine.
//
// Select the line under the given point. This will select across soft-wrapped
// lines and will omit the leading and trailing whitespace. If the point is
// over whitespace but the line has non-whitespace characters elsewhere, the
// line will be selected.
//
// Zig: pub fn selectLine(self: *const Screen, opts: SelectLine) ?Selection.
func (self *Screen) SelectLine(opts ScreenSelectLine) Optional[Selection] {
	panic(notImplemented)
}

// SelectAll is Screen.selectAll.
//
// Return the selection for all contents on the screen. Surrounding
// whitespace is omitted. If there is no selection, this returns null.
//
// Zig: returns ?Selection.
func (self *Screen) SelectAll() Optional[Selection] { panic(notImplemented) }

// SelectWordBetween is Screen.selectWordBetween.
//
// Select the nearest word to start point that is between start_pt and
// end_pt (inclusive). Because it selects "nearest" to start point, start
// point can be before or after end point.
//
// The boundary_codepoints parameter should be a slice of u21 codepoints that
// mark word boundaries, passed through to selectWord.
//
// TODO: test this
//
// Zig: pub fn selectWordBetween(self: *Screen, start: Pin, end: Pin,
// boundary_codepoints: []const u21) ?Selection.
func (self *Screen) SelectWordBetween(
	start PageListPin,
	end PageListPin,
	boundary_codepoints []uint32,
) Optional[Selection] {
	panic(notImplemented)
}

// SelectWord is Screen.selectWord.
//
// Select the word under the given point. A word is any consecutive series
// of characters that are exclusively whitespace or exclusively non-whitespace.
// A selection can span multiple physical lines if they are soft-wrapped.
//
// This will return null if a selection is impossible. The only scenario
// this happens is if the point pt is outside of the written screen space.
//
// The boundary_codepoints parameter should be a slice of u21 codepoints that
// mark word boundaries. This is expected to be pre-parsed from the config.
//
// Zig: pub fn selectWord(self: *Screen, pin: Pin,
// boundary_codepoints: []const u21) ?Selection.
func (self *Screen) SelectWord(pin PageListPin, boundary_codepoints []uint32) Optional[Selection] {
	panic(notImplemented)
}

// SelectOutput is Screen.selectOutput.
//
// Select the command output under the given point. The limits of the output
// are determined by semantic prompt information provided by shell integration.
// A selection can span multiple physical lines if they are soft-wrapped.
//
// This will return null if a selection is impossible:
//   - the point pt is outside of the written screen space.
//   - the point pt is on a prompt / input line.
//
// Zig: pub fn selectOutput(self: *Screen, pin: Pin) ?Selection.
func (self *Screen) SelectOutput(pin PageListPin) Optional[Selection] { panic(notImplemented) }

// ScreenLineIterator is Screen.LineIterator.
type ScreenLineIterator struct {
	// Zig: *const Screen.
	screen *Screen
	// Zig: ?Pin, default null.
	current Optional[PageListPin]
}

// Next is Screen.LineIterator.next.
//
// Zig: pub fn next(self: *LineIterator) ?Selection.
func (self *ScreenLineIterator) Next() Optional[Selection] { panic(notImplemented) }

// LineIterator is Screen.lineIterator.
//
// Returns an iterator to move through the soft-wrapped lines starting
// from pin.
//
// Zig: pub fn lineIterator(self: *const Screen, start: Pin) LineIterator.
func (self *Screen) LineIterator(start PageListPin) ScreenLineIterator { panic(notImplemented) }

// ScreenPromptClickMove is Screen.PromptClickMove.
type ScreenPromptClickMove struct {
	// Zig: usize. A count.
	left uint
	// Zig: usize. A count.
	right uint
}

// ScreenPromptClickMoveZero is Screen.PromptClickMove.zero.
var ScreenPromptClickMoveZero = ScreenPromptClickMove{
	left:  0,
	right: 0,
}

// PromptClickMove is Screen.promptClickMove.
//
// Determine the inputs necessary to move the cursor to the given
// click location within a prompt input area.
//
// If the cursor isn't currently at a prompt input location, this
// returns no movement.
//
// This feature depends on well-behaved OSC133 shell integration. Specifically,
// this only moves over designated input areas (OSC 133 B). It is assumed
// that the shell will only move the cursor to input cells, so prompt cells
// and other blank cells are ignored as part of the movement calculation.
func (self *Screen) PromptClickMove(click_pin PageListPin) ScreenPromptClickMove {
	panic(notImplemented)
}

// promptClickLine is Screen.promptClickLine.
//
// Determine the inputs required to move from the cursor to the given
// click location. If the cursor isn't currently at a prompt input
// location, this will return zero.
//
// This currently only supports moving a single line.
func (self *Screen) promptClickLine(click_pin PageListPin) ScreenPromptClickMove {
	panic(notImplemented)
}

// ScreenDumpStringOptions is NOT a named Zig declaration: it is the
// anonymous struct that is the `opts` parameter of Screen.dumpString.
type ScreenDumpStringOptions struct {
	// The start and end points of the dump, both inclusive. The x will
	// be ignored and the full row will always be dumped.
	tl PageListPin
	// Zig: ?Pin, default null.
	br Optional[PageListPin]

	// If true, this will unwrap soft-wrapped lines. If false, this will
	// dump the screen as it is visually seen in a rendered window.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default must set it.
	unwrap bool
}

// DumpString is Screen.dumpString.
//
// Dump the screen to a string. The writer given should be buffered;
// this function does not attempt to efficiently write and generally writes
// one byte at a time.
//
// Zig: pub fn dumpString(self: *const Screen, writer: *std.Io.Writer,
// opts: struct { tl: Pin, br: ?Pin = null, unwrap: bool = true })
// std.Io.Writer.Error!void. The writer is the stand-in oscWriter (osc.go);
// the error is ErrWriteFailed (osc.go).
func (self *Screen) DumpString(writer *oscWriter, opts ScreenDumpStringOptions) error {
	panic(notImplemented)
}

// ErrUnknownPoint is Zig's error.UnknownPoint (dumpStringAlloc and
// dumpStringAllocUnwrapped: the point has no bottom-right pin).
var ErrUnknownPoint = errors.New("UnknownPoint")

// DumpStringAlloc is Screen.dumpStringAlloc.
//
// You should use dumpString, this is a restricted version mostly for
// legacy and convenience reasons for unit tests.
//
// Zig: pub fn dumpStringAlloc(self: *const Screen, alloc: Allocator,
// tl: point.Point) ![]const u8. The allocator parameter is dropped. `tl`
// is a point of which only the tag is used. The inferred error set is
// ErrUnknownPoint, ErrWriteFailed (osc.go) and ErrOutOfMemory (dcs.go).
func (self *Screen) DumpStringAlloc(tl Point) ([]uint8, error) { panic(notImplemented) }

// DumpStringAllocUnwrapped is Screen.dumpStringAllocUnwrapped.
//
// You should use dumpString, this is a restricted version mostly for
// legacy and convenience reasons for unit tests.
//
// Zig: pub fn dumpStringAllocUnwrapped(self: *const Screen,
// alloc: Allocator, tl: point.Point) ![]const u8. The allocator parameter
// is dropped. `tl` is a point of which only the tag is used. The inferred
// error set is ErrUnknownPoint, ErrWriteFailed (osc.go) and ErrOutOfMemory
// (dcs.go).
func (self *Screen) DumpStringAllocUnwrapped(tl Point) ([]uint8, error) {
	panic(notImplemented)
}

// ErrInvalidUtf8 is Zig's error.InvalidUtf8 (std.unicode.Utf8View.init, in
// testWriteString).
var ErrInvalidUtf8 = errors.New("InvalidUtf8")

// TestWriteString is Screen.testWriteString.
//
// This is basically a really jank version of Terminal.printString. We
// have to reimplement it here because we want a way to print to the screen
// to test it but don't want all the features of Terminal.
//
// Zig: pub fn testWriteString(self: *Screen, text: []const u8) !void. The
// inferred error set is ErrInvalidUtf8, that of cursorDownOrScroll
// (ErrOutOfMemory, Page.CloneFromError), that of Page.appendGrapheme
// (ErrOutOfMemory) and that of cursorSetHyperlink (ErrOutOfMemory,
// ErrOutOfSpace).
func (self *Screen) TestWriteString(text []uint8) error { panic(notImplemented) }
