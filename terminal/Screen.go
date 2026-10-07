// Ported from src/terminal/Screen.zig (Ghostty v1.3.1).
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
// `isize` is `int64` (the reference's width: `Scroll.delta_row` and
// `delta_prompt`), and fixed Zig widths stay fixed. Byte quantities, both
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
// Added by the implementation (each says what it stands for): the names
// arrays of the four enums; `ErrScreenCloneCursorOutside` (clone returns
// it where upstream aborts; docs/mapping.md "Page-layer bookkeeping");
// `screenRotateOnceR` (fastmem.rotateOnceR on rows); the method
// `oscWriterAllocating.toOwnedSliceSentinel` (std.Io.Writer.Allocating,
// used only by selectionString; it could move to osc.go). The std.mem
// functions selectLine/selectAll/selectWord call are in std_mem.go.
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
// # What the implementation needs from other files
//
//   - `selectionString` (line 2448) and `dumpString` (line 3129) are
//     written against the formatter.go wireframe (`ScreenFormatter`); they
//     panic "not implemented" until formatter.go is ported;
//   - `testWriteString` (line 3207) reads the width of a codepoint above
//     0xFF from `unicode.table` (`UnicodeTable.Get`, unicode_props_table.go).
//
// # Build options
//
// `build_options.slow_runtime_safety` is the package variable
// `pageSlowRuntimeSafety` (page.go), false by default (upstream's release
// builds) and switched on by the test binary; `assertIntegrity` does nothing
// without it. The `std.valgrind.runningOnValgrind()` check has no Go
// counterpart.
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

import (
	"errors"
	"unicode/utf8"
)

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
	Pages PageList

	// Special-case where we want no scrollback whatsoever. We have to flag
	// this because max_size 0 in PageList gets rounded up to two pages so
	// we can always have an active screen.
	//
	// Zig default: false.
	NoScrollback bool

	// The current cursor position
	Cursor ScreenCursor

	// The saved cursor
	//
	// Zig: ?SavedCursor, default null.
	SavedCursor Optional[ScreenSavedCursor]

	// The selection for this screen (if any). This MUST be a tracked selection
	// otherwise the selection will become invalid. Instead of accessing this
	// directly to set it, use the `select` function which will assert and
	// automatically setup tracking.
	//
	// Zig: ?Selection, default null.
	Selection Optional[Selection]

	// The charset state
	//
	// Zig default: `.{}`, which is NOT the Go zero value (`gr` is G2); see
	// ScreenCharsetState.
	Charset ScreenCharsetState

	// The current or most recent protected mode. Once a protection mode is
	// set, this will never become "off" again until the screen is reset.
	// The current state of whether protection attributes should be set is
	// set on the Cell pen; this is only used to determine the most recent
	// protection mode since some sequences such as ECH depend on this.
	//
	// Zig default: .off.
	ProtectedMode ANSIProtectedMode

	// The kitty keyboard settings.
	//
	// Zig default: `.{}`.
	KittyKeyboard KittyKeyFlagStack

	// Screen.kitty_images (line 72), the Kitty graphics protocol state, is
	// omitted; see the file header.

	// Semantic prompt (OSC133) state.
	//
	// Zig default: .disabled (ScreenSemanticPromptDisabled).
	SemanticPrompt ScreenSemanticPrompt

	// Dirty flags for the renderer.
	//
	// Zig default: `.{}`.
	Dirty ScreenDirty
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
	Selection bool

	// When an OSC8 hyperlink is hovered, we set the full screen as dirty
	// because links can span multiple lines.
	HyperlinkHover bool
}

// ScreenSemanticPrompt is Screen.SemanticPrompt.
type ScreenSemanticPrompt struct {
	// This is flipped to true when any sort of semantic content is
	// seen. In particular, this is set to true only when a `prompt` type
	// is ever set on our cursor.
	//
	// This is used to optimize away semantic content operations if we know
	// we've never seen them.
	Seen bool

	// This is set on any `cl` or `click_events` option set on the
	// most recent OSC 133 commands to specify how click handling in a
	// prompt is handling.
	Click ScreenSemanticPromptSemanticClick
}

// ScreenSemanticPromptDisabled is Screen.SemanticPrompt.disabled.
var ScreenSemanticPromptDisabled = ScreenSemanticPrompt{
	Seen:  false,
	Click: ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickNone},
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
func (self ScreenSemanticPromptSemanticClickTag) String() string {
	return screenSemanticPromptSemanticClickTagNames[self]
}

// screenSemanticPromptSemanticClickTagNames are the tag names of
// ScreenSemanticPromptSemanticClickTag, indexed by value (Zig's @tagName).
var screenSemanticPromptSemanticClickTagNames = [...]string{"none", "click_events", "cl"}

// ScreenCursor is Screen.Cursor: the cursor position and style.
type ScreenCursor struct {
	// The x/y position within the active area.
	//
	// Zig default: 0.
	X SizeCellCountInt
	// Zig default: 0.
	Y SizeCellCountInt

	// The visual style of the cursor. This defaults to block because
	// it has to default to something, but users of this struct are
	// encouraged to set their own default.
	//
	// Zig default: .block, which is NOT the Go zero value (.bar); see the
	// file header.
	CursorStyle ScreenCursorStyle

	// The "last column flag (LCF)" as its called. If this is set then the
	// next character print will force a soft-wrap.
	//
	// Zig default: false.
	PendingWrap bool

	// The protected mode state of the cursor. If this is true then
	// all new characters printed will have the protected state set.
	//
	// Zig default: false.
	Protected bool

	// The currently active style. This is the concrete style value
	// that should be kept up to date. The style ID to use for cell writing
	// is below.
	//
	// Zig default: `.{}`.
	Style Style

	// The currently active style ID. The style is page-specific so when
	// we change pages we need to ensure that we update that page with
	// our style when used.
	//
	// Zig default: style.default_id (StyleDefaultId, 0).
	StyleID StyleID

	// The hyperlink ID that is currently active for the cursor. A value
	// of zero means no hyperlink is active. (Implements OSC8, saying that
	// so code search can find it.).
	//
	// Zig default: 0.
	HyperlinkID HyperlinkID

	// This is the implicit ID to use for hyperlinks that don't specify
	// an ID. We do an overflowing add to this so repeats can technically
	// happen with carefully crafted inputs but for real workloads its
	// highly unlikely -- and the fix is for the TUI program to use explicit
	// IDs.
	//
	// Zig default: 0.
	HyperlinkImplicitID SizeOffsetInt

	// Heap-allocated hyperlink state so that we can recreate it when
	// the cursor page pin changes. We can't get it from the old screen
	// state because the page may be cleared. This is heap allocated
	// because its most likely null.
	//
	// Zig: ?*hyperlink.Hyperlink, default null.
	Hyperlink *Hyperlink

	// The current semantic content type for the cursor that will be
	// applied to any newly written cells.
	//
	// Zig default: .output.
	SemanticContent PageCellSemanticContent
	// Zig default: false.
	SemanticContentClearEol bool

	// The pointers into the page list where the cursor is currently
	// located. This makes it faster to move the cursor.
	PagePin  *PageListPin
	PageRow  *PageRow
	PageCell *PageCell
}

// Deinit is Screen.Cursor.deinit.
//
// Zig: pub fn deinit(self: *Cursor, alloc: Allocator) void. The allocator
// parameter is dropped. It deinitialises and frees the cursor's hyperlink,
// if any; the stub is kept because the tests call it.
func (self *ScreenCursor) Deinit() {
	if self.Hyperlink != nil {
		link := self.Hyperlink
		link.Deinit()
		// Zig: alloc.destroy(link); Go memory.
	}
}

// ScreenSavedCursor is Screen.SavedCursor: saved cursor state.
type ScreenSavedCursor struct {
	X           SizeCellCountInt
	Y           SizeCellCountInt
	Style       Style
	Protected   bool
	PendingWrap bool
	Origin      bool
	Charset     ScreenCharsetState
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
	Gl CharsetsSlots
	// Zig default: .G2, which is NOT the Go zero value.
	Gr CharsetsSlots

	// Single shift where a slot is used for exactly one char.
	//
	// Zig: ?charsets.Slots, default null.
	SingleShift Optional[CharsetsSlots]
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
	switch slot {
	case CharsetsSlotsG0:
		return self.g0
	case CharsetsSlotsG1:
		return self.g1
	case CharsetsSlotsG2:
		return self.g2
	case CharsetsSlotsG3:
		return self.g3
	default:
		panic("unreachable")
	}
}

// Set is Screen.CharsetState.CharsetArray.set.
//
// Zig: pub inline fn set(self: *CharsetArray, slot: charsets.Slots,
// charset: charsets.Charset) void.
func (self *screenCharsetStateCharsetArray) Set(slot CharsetsSlots, charset CharsetsCharset) {
	switch slot {
	case CharsetsSlotsG0:
		self.g0 = charset
	case CharsetsSlotsG1:
		self.g1 = charset
	case CharsetsSlotsG2:
		self.g2 = charset
	case CharsetsSlotsG3:
		self.g3 = charset
	default:
		panic("unreachable")
	}
}

// ScreenOptions is Screen.Options.
type ScreenOptions struct {
	Cols SizeCellCountInt
	Rows SizeCellCountInt

	// The maximum size of scrollback in bytes. Zero means unlimited. Any
	// other value will be clamped to support a minimum of the active area.
	//
	// Zig: usize, default 0. Bytes: it is passed to PageList.init as
	// max_size. (Whatever the comment above says, `init` treats zero as "no
	// scrollback": it sets `no_scrollback` and passes a max_size of 0.)
	MaxScrollback uint64

	// The total storage limit for Kitty images in bytes for this
	// screen. Kitty image storage is per-screen.
	//
	// Zig: usize, default 320 * 1000 * 1000 (320MB), which is NOT the Go
	// zero value. Bytes. Nothing reads it in the library build (Kitty
	// graphics are compiled out).
	KittyImageStorageLimit uint64
}

// ScreenOptionsDefault is Screen.Options.default.
//
// A simple, default terminal. If you rely on specific dimensions or
// scrollback (or lack of) then do not use this directly. This is just
// for callers that need some defaults.
var ScreenOptionsDefault = ScreenOptions{
	Cols:                   80,
	Rows:                   24,
	MaxScrollback:          0,
	KittyImageStorageLimit: 320 * 1000 * 1000,
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
func ScreenInit(opts ScreenOptions) (_ Screen, err error) {
	// Initialize our backing pages.
	pages, err := PageListInit(
		opts.Cols,
		opts.Rows,
		Some(opts.MaxScrollback),
	)
	if err != nil {
		return Screen{}, err
	}
	defer func() {
		// errdefer pages.deinit();
		if err != nil {
			pages.Deinit()
		}
	}()

	// Create our tracked pin for the cursor.
	assert(pages.pages.first != nil) // .?
	page_pin, err := pages.TrackPin(PageListPin{Node: pages.pages.first})
	if err != nil {
		return Screen{}, err
	}
	// Zig: errdefer pages.untrackPin(page_pin): nothing fails after it.
	page_rac := page_pin.RowAndCell()

	result := Screen{
		Pages:        pages,
		NoScrollback: opts.MaxScrollback == 0,
		Cursor: ScreenCursor{
			X:           0,
			Y:           0,
			CursorStyle: ScreenCursorStyleBlock, // Zig default
			PagePin:     page_pin,
			PageRow:     page_rac.row,
			PageCell:    page_rac.cell,
		},
		Charset:        ScreenCharsetState{Gr: CharsetsSlotsG2}, // Zig default .{}
		SemanticPrompt: ScreenSemanticPromptDisabled,
	}

	// Zig: `if (comptime build_options.kitty_graphics)` block
	// (kitty_images.setLimit): compiled out; see the file header.

	return result, nil
}

// Deinit is Screen.deinit.
//
// Kept although it mostly frees: it deinitialises the cursor and the page
// list (which asserts integrity), and every test calls it.
func (self *Screen) Deinit() {
	// Zig: `if (comptime build_options.kitty_graphics)` block
	// (kitty_images.deinit): compiled out; see the file header.
	self.Cursor.Deinit()
	self.Pages.Deinit()
}

// AssertIntegrity is Screen.assertIntegrity.
//
// Assert that the screen is in a consistent state. This doesn't check
// all pages in the page list because that is SO SLOW even just for
// tests. This only asserts the screen specific data so callers should
// ensure they're also calling page integrity checks if necessary.
//
// Zig: self is *const Screen.
//
// Go: the body is assertIntegrityChecked, so that this wrapper is small
// enough for the compiler to inline: with the checks off (the default, as in
// upstream's release builds) a call costs one test of the flag, as Zig's
// comptime `if` costs nothing (docs/post-port-optimizations.md item 33).
func (self *Screen) AssertIntegrity() {
	if pageSlowRuntimeSafety {
		self.assertIntegrityChecked()
	}
}

// assertIntegrityChecked is the body of AssertIntegrity, run when the
// integrity checks are on. Not a Zig declaration.
func (self *Screen) assertIntegrityChecked() {
	// We don't run integrity checks on Valgrind because its soooooo slow,
	// Valgrind is our integrity checker, and we run these during unit
	// tests (non-Valgrind) anyways so we're verifying anyways.
	//
	// Zig: `if (std.valgrind.runningOnValgrind() > 0) return;`; no Go
	// counterpart (file header).

	assert(self.Cursor.X < self.Pages.cols)
	assert(self.Cursor.Y < self.Pages.rows)

	// Our cursor x/y should always match the pin. If this doesn't
	// match then it indicates that the tracked pin moved and we didn't
	// account for it by either calling cursorReload or manually
	// adjusting.
	pt_ := self.Pages.PointFromPin(
		PointTagActive,
		*self.Cursor.PagePin,
	)
	if !pt_.Valid {
		panic("unreachable") // orelse unreachable
	}
	pt := pt_.Value
	assert(self.Cursor.X == pt.Active.X)
	assert(uint32(self.Cursor.Y) == pt.Active.Y)
}

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
func (self *Screen) Reset() {
	// Reset our pages
	self.Pages.Reset()

	// The above reset preserves tracked pins so we can still use
	// our cursor pin, which should be at the top-left already.
	cursor_pin := self.Cursor.PagePin
	assert(self.Pages.pages.first != nil) // .?
	assert(cursor_pin.Node == self.Pages.pages.first)
	assert(cursor_pin.X == 0)
	assert(cursor_pin.Y == 0)
	cursor_rac := cursor_pin.RowAndCell()
	self.Cursor.Deinit()
	self.Cursor = ScreenCursor{
		CursorStyle: ScreenCursorStyleBlock, // Zig default
		PagePin:     cursor_pin,
		PageRow:     cursor_rac.row,
		PageCell:    cursor_rac.cell,
	}

	// Zig: `if (comptime build_options.kitty_graphics)` block (reset
	// kitty graphics storage): compiled out; see the file header.

	// Reset our basic state
	self.SavedCursor = None[ScreenSavedCursor]()
	self.Charset = ScreenCharsetState{Gr: CharsetsSlotsG2} // Zig: .{}
	self.KittyKeyboard = KittyKeyFlagStack{}
	self.ProtectedMode = ANSIProtectedModeOff
	self.SemanticPrompt = ScreenSemanticPromptDisabled
	self.ClearSelection()
}

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
func (self *Screen) Clone(top Point, bot Optional[Point]) (_ Screen, err error) {
	// Create a tracked pin remapper for our selection and cursor. Note
	// that we may want to expose this generally in the future but at the
	// time of doing this we don't need to.
	pin_remap := PageListCloneTrackedPinsRemap{}
	// Zig: defer pin_remap.deinit(); Go memory.

	pages, err := self.Pages.Clone(PageListClone{
		Top:         top,
		Bot:         bot,
		TrackedPins: &pin_remap,
	})
	if err != nil {
		return Screen{}, err
	}
	defer func() {
		// errdefer pages.deinit();
		if err != nil {
			pages.Deinit()
		}
	}()

	// Find our cursor. If the cursor isn't in the cloned area, we move it
	// to the top-left arbitrarily because a screen must have SOME cursor.
	//
	// Zig: the labeled block `cursor:`, with the inner block `remap:`.
	cursor, err := func() (ScreenCursor, error) {
		if p := pin_remap[self.Cursor.PagePin]; p != nil {
			page_rac := p.RowAndCell()
			if pt := pages.PointFromPin(PointTagActive, *p); pt.Valid {
				// Zig: @intCast(pt.active.y) to u16.
				assert(pt.Value.Active.Y <= 0xFFFF)
				return ScreenCursor{
					X:           pt.Value.Active.X,
					Y:           SizeCellCountInt(pt.Value.Active.Y),
					CursorStyle: ScreenCursorStyleBlock, // Zig default
					PagePin:     p,
					PageRow:     page_rac.row,
					PageCell:    page_rac.cell,
				}, nil
			}
			// break :remap
		}

		assert(pages.pages.first != nil) // .?
		page_pin, err := pages.TrackPin(PageListPin{Node: pages.pages.first})
		if err != nil {
			return ScreenCursor{}, err
		}
		page_rac := page_pin.RowAndCell()
		return ScreenCursor{
			X:           0,
			Y:           0,
			CursorStyle: ScreenCursorStyleBlock, // Zig default
			PagePin:     page_pin,
			PageRow:     page_rac.row,
			PageCell:    page_rac.cell,
		}, nil
	}()
	if err != nil {
		return Screen{}, err
	}

	// Preserve our selection if we have one.
	//
	// Zig: `if (self.selection) |sel| sel: { ... } else null`; `break :sel
	// null` is a return of null from the closure.
	sel, err := func() (Optional[Selection], error) {
		if !self.Selection.Valid {
			return None[Selection](), nil
		}
		sel := self.Selection.Value
		assert(sel.Tracked())

		var ordered struct {
			tl *PageListPin
			br *PageListPin
		}
		switch sel.Order(self) {
		case SelectionOrderForward, SelectionOrderMirroredForward:
			ordered.tl = sel.Bounds.Tracked.Start
			ordered.br = sel.Bounds.Tracked.End
		case SelectionOrderReverse, SelectionOrderMirroredReverse:
			ordered.tl = sel.Bounds.Tracked.End
			ordered.br = sel.Bounds.Tracked.Start
		default:
			panic("unreachable")
		}

		start_pin := pin_remap[ordered.tl]
		if start_pin == nil {
			// start:
			// No start means it is outside the cloned area.

			// If we have no end pin then either
			// (1) our whole selection is outside the cloned area or
			// (2) our cloned area is within the selection
			if pin_remap[ordered.br] == nil {
				// We check if the selection bottom right pin is above
				// the cloned area or if the top left pin is below the
				// cloned area, in either of these cases it means that
				// the selection is fully out of bounds, so we have no
				// selection in the cloned area and break out now.
				clone_top_ := self.Pages.Pin(top)
				if !clone_top_.Valid {
					return None[Selection](), nil
				}
				clone_top := clone_top_.Value
				clone_top_y_ := self.Pages.PointFromPin(
					PointTagScreen,
					clone_top,
				)
				assert(clone_top_y_.Valid) // .?
				clone_top_y := clone_top_y_.Value.Screen.Y
				br_pt := self.Pages.PointFromPin(
					PointTagScreen,
					*ordered.br,
				)
				assert(br_pt.Valid) // .?
				if br_pt.Value.Screen.Y < clone_top_y {
					return None[Selection](), nil
				}
				tl_pt := self.Pages.PointFromPin(
					PointTagScreen,
					*ordered.tl,
				)
				assert(tl_pt.Valid) // .?
				if tl_pt.Value.Screen.Y > clone_top_y {
					return None[Selection](), nil
				}
			}

			// We move the top pin back in bounds to the top row.
			assert(pages.pages.first != nil) // .?
			var x SizeCellCountInt = 0
			if sel.Rectangle {
				x = ordered.tl.X
			}
			start_pin, err = pages.TrackPin(PageListPin{
				Node: pages.pages.first,
				X:    x,
			})
			if err != nil {
				return None[Selection](), err
			}
		}

		// If we got to this point it means that the selection is not
		// fully out of bounds, so we move the bottom right pin back
		// in bounds if it isn't already.
		end_pin := pin_remap[ordered.br]
		if end_pin == nil {
			assert(pages.pages.last != nil) // .?
			var x SizeCellCountInt
			if sel.Rectangle {
				x = ordered.br.X
			} else {
				assert(pages.cols > 0) // Zig: - traps
				x = pages.cols - 1
			}
			assert(pages.pages.last.data.size.rows > 0) // Zig: - traps
			end_pin, err = pages.TrackPin(PageListPin{
				Node: pages.pages.last,
				X:    x,
				Y:    pages.pages.last.data.size.rows - 1,
			})
			if err != nil {
				return None[Selection](), err
			}
		}

		return Some(Selection{
			Bounds: SelectionBounds{
				Tag: SelectionBoundsTagTracked,
				Tracked: SelectionBoundsTracked{
					Start: start_pin,
					End:   end_pin,
				},
			},
			Rectangle: sel.Rectangle,
		}), nil
	}()
	if err != nil {
		return Screen{}, err
	}

	result := Screen{
		Pages:          pages,
		NoScrollback:   self.NoScrollback,
		Cursor:         cursor,
		Selection:      sel,
		Charset:        ScreenCharsetState{Gr: CharsetsSlotsG2}, // Zig default .{}
		SemanticPrompt: ScreenSemanticPromptDisabled,
		Dirty:          self.Dirty,
	}

	// term51 (docs/mapping.md, "Page-layer bookkeeping"; term51
	// docs/upstream-crashes.md and upstream-findings.md section 15, Screen
	// item 1): when the cursor was outside the cloned region and the clone
	// has more rows than the active area, the fallback cursor pin above is
	// not in the active area and upstream aborts in the assertIntegrity
	// below (`pointFromPin(...) orelse unreachable`). The port returns its
	// own error instead; the errdefer above deinits the clone.
	if !result.Pages.PointFromPin(PointTagActive, *result.Cursor.PagePin).Valid {
		return Screen{}, ErrScreenCloneCursorOutside
	}

	result.AssertIntegrity()
	return result, nil
}

// ErrScreenCloneCursorOutside is NOT a Zig error: Screen.clone returns it
// where upstream aborts (the cursor is outside the cloned region and the
// clone has more rows than the active area). See docs/mapping.md
// ("Page-layer bookkeeping") and the comment at its use.
var ErrScreenCloneCursorOutside = errors.New("ScreenCloneCursorOutside")

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
	// If the page being modified isn't our cursor page then
	// this is a quick operation because we have no additional
	// accounting. We have to do this check here BEFORE calling
	// increaseCapacity because increaseCapacity will update all
	// our tracked pins (including our cursor).
	if node != self.Cursor.PagePin.Node {
		return self.Pages.IncreaseCapacity(
			node,
			adjustment,
		)
	}

	// We're modifying the cursor page. When we increase the
	// capacity below it will be short the ref count on our
	// current style and hyperlink, so we need to init those.
	new_node, err := self.Pages.IncreaseCapacity(node, adjustment)
	if err != nil {
		return nil, err
	}
	new_page := &new_node.data

	// Re-add the style, if the page somehow doesn't have enough
	// memory to add it, we emit a warning and gracefully degrade
	// to the default style for the cursor.
	if self.Cursor.StyleID != StyleDefaultID {
		id, err := new_page.styles.Add(
			self.Cursor.Style,
		)
		if err != nil {
			// id:
			// TODO: Should we increase the capacity further in this case?
			warn(
				"screen",
				"(Screen.increaseCapacity) Failed to add cursor style back to page, err=error.%v",
				err,
			)

			// Reset the cursor style.
			self.Cursor.Style = Style{}
			id = StyleDefaultID
		}
		self.Cursor.StyleID = id
	}

	// Re-add the hyperlink, if the page somehow doesn't have enough
	// memory to add it, we emit a warning and gracefully degrade to
	// no hyperlink.
	if self.Cursor.Hyperlink != nil {
		link := self.Cursor.Hyperlink

		// So we don't attempt to free any memory in the replaced page.
		self.Cursor.HyperlinkID = 0
		self.Cursor.Hyperlink = nil

		// Re-add
		if err := self.startHyperlinkOnce(*link); err != nil {
			// TODO: Should we increase the capacity further in this case?
			warn(
				"screen",
				"(Screen.increaseCapacity) Failed to add cursor hyperlink back to page, err=error.%v",
				err,
			)
		}

		// Remove our old link
		link.Deinit()
		// Zig: self.alloc.destroy(link); Go memory.
	}

	// Reload the cursor information because the pin changed.
	// So our page row/cell and so on are all off.
	self.CursorReload()

	return new_node, nil
}

// CursorCellRight is Screen.cursorCellRight.
//
// Zig: pub inline fn cursorCellRight(self: *Screen, n: size.CellCountInt)
// *pagepkg.Cell. The result points into the cursor row's cells.
func (self *Screen) CursorCellRight(n SizeCellCountInt) *PageCell {
	// Zig: `self.cursor.x + n` is u16 and traps on overflow; the sum is
	// taken wider so an overflow fails the assertion instead of wrapping.
	assert(uint32(self.Cursor.X)+uint32(n) < uint32(self.Pages.cols))
	// Zig: `@as([*]Cell, @ptrCast(self.cursor.page_cell)) + n`.
	page := &self.Cursor.PagePin.Node.data
	return &page.cells[page.cellIndex(self.Cursor.PageCell)+uint(n)]
}

// CursorCellLeft is Screen.cursorCellLeft.
//
// Zig: pub inline fn cursorCellLeft(self: *Screen, n: size.CellCountInt)
// *pagepkg.Cell. The result points into the cursor row's cells.
func (self *Screen) CursorCellLeft(n SizeCellCountInt) *PageCell {
	assert(self.Cursor.X >= n)
	// Zig: `@as([*]Cell, @ptrCast(self.cursor.page_cell)) - n`.
	page := &self.Cursor.PagePin.Node.data
	return &page.cells[page.cellIndex(self.Cursor.PageCell)-uint(n)]
}

// CursorCellEndOfPrev is Screen.cursorCellEndOfPrev.
func (self *Screen) CursorCellEndOfPrev() *PageCell {
	assert(self.Cursor.Y > 0)

	page_pin_ := self.Cursor.PagePin.Up(1)
	assert(page_pin_.Valid) // .?
	page_pin := page_pin_.Value
	page_pin.X = self.Pages.cols - 1
	page_rac := page_pin.RowAndCell()
	return page_rac.cell
}

// CursorRight is Screen.cursorRight.
//
// Move the cursor right. This is a specialized function that is very fast
// if the caller can guarantee we have space to move right (no wrapping).
func (self *Screen) CursorRight(n SizeCellCountInt) {
	// Zig: `self.cursor.x + n` is u16 and traps on overflow.
	assert(uint32(self.Cursor.X)+uint32(n) < uint32(self.Pages.cols))
	// Zig: defer self.assertIntegrity(); Go: the call at the end of the
	// function, which has no other exit (Terminal.Print has the reason).

	// Zig: `cell + n` on the many-item pointer.
	page := &self.Cursor.PagePin.Node.data
	self.Cursor.PageCell = &page.cells[page.cellIndex(self.Cursor.PageCell)+uint(n)]
	self.Cursor.PagePin.X += n
	self.Cursor.X += n

	self.AssertIntegrity()
}

// CursorLeft is Screen.cursorLeft.
//
// Move the cursor left.
func (self *Screen) CursorLeft(n SizeCellCountInt) {
	assert(self.Cursor.X >= n)
	defer self.AssertIntegrity()

	// Zig: `cell - n` on the many-item pointer.
	page := &self.Cursor.PagePin.Node.data
	self.Cursor.PageCell = &page.cells[page.cellIndex(self.Cursor.PageCell)-uint(n)]
	// Zig: `-=` on u16 traps on underflow; the pin's x is the cursor's x
	// (assertIntegrity), so the assertion above covers it.
	self.Cursor.PagePin.X -= n
	self.Cursor.X -= n
}

// CursorUp is Screen.cursorUp.
//
// Move the cursor up.
//
// Precondition: The cursor is not at the top of the screen.
func (self *Screen) CursorUp(n SizeCellCountInt) {
	assert(self.Cursor.Y >= n)
	defer self.AssertIntegrity()

	self.Cursor.Y -= n // Must be set before cursorChangePin
	up := self.Cursor.PagePin.Up(uint(n))
	assert(up.Valid) // .?
	self.cursorChangePin(up.Value)
	page_rac := self.Cursor.PagePin.RowAndCell()
	self.Cursor.PageRow = page_rac.row
	self.Cursor.PageCell = page_rac.cell
}

// CursorRowUp is Screen.cursorRowUp.
//
// Zig: pub fn cursorRowUp(self: *Screen, n: size.CellCountInt)
// *pagepkg.Row.
func (self *Screen) CursorRowUp(n SizeCellCountInt) *PageRow {
	assert(self.Cursor.Y >= n)
	defer self.AssertIntegrity()

	page_pin_ := self.Cursor.PagePin.Up(uint(n))
	assert(page_pin_.Valid) // .?
	page_rac := page_pin_.Value.RowAndCell()
	return page_rac.row
}

// CursorDown is Screen.cursorDown.
//
// Move the cursor down.
//
// Precondition: The cursor is not at the bottom of the screen.
func (self *Screen) CursorDown(n SizeCellCountInt) {
	// Zig: `self.cursor.y + n` is u16 and traps on overflow.
	assert(uint32(self.Cursor.Y)+uint32(n) < uint32(self.Pages.rows))
	defer self.AssertIntegrity()

	self.Cursor.Y += n // Must be set before cursorChangePin

	// We move the offset into our page list to the next row and then
	// get the pointers to the row/cell and set all the cursor state up.
	down := self.Cursor.PagePin.Down(uint(n))
	assert(down.Valid) // .?
	self.cursorChangePin(down.Value)
	page_rac := self.Cursor.PagePin.RowAndCell()
	self.Cursor.PageRow = page_rac.row
	self.Cursor.PageCell = page_rac.cell
}

// CursorHorizontalAbsolute is Screen.cursorHorizontalAbsolute.
//
// Move the cursor to some absolute horizontal position.
func (self *Screen) CursorHorizontalAbsolute(x SizeCellCountInt) {
	assert(x < self.Pages.cols)
	defer self.AssertIntegrity()

	self.Cursor.PagePin.X = x
	page_rac := self.Cursor.PagePin.RowAndCell()
	self.Cursor.PageCell = page_rac.cell
	self.Cursor.X = x
}

// CursorAbsolute is Screen.cursorAbsolute.
//
// Move the cursor to some absolute position.
func (self *Screen) CursorAbsolute(x SizeCellCountInt, y SizeCellCountInt) {
	assert(x < self.Pages.cols)
	assert(y < self.Pages.rows)
	defer self.AssertIntegrity()

	var page_pin PageListPin
	if y < self.Cursor.Y {
		up := self.Cursor.PagePin.Up(uint(self.Cursor.Y - y))
		assert(up.Valid) // .?
		page_pin = up.Value
	} else if y > self.Cursor.Y {
		down := self.Cursor.PagePin.Down(uint(y - self.Cursor.Y))
		assert(down.Valid) // .?
		page_pin = down.Value
	} else {
		page_pin = *self.Cursor.PagePin
	}
	page_pin.X = x
	self.Cursor.X = x // Must be set before cursorChangePin
	self.Cursor.Y = y
	self.cursorChangePin(page_pin)
	page_rac := self.Cursor.PagePin.RowAndCell()
	self.Cursor.PageRow = page_rac.row
	self.Cursor.PageCell = page_rac.cell
}

// CursorReload is Screen.cursorReload.
//
// Reloads the cursor pointer information into the screen. This is expensive
// so it should only be done in cases where the pointers are invalidated
// in such a way that its difficult to recover otherwise.
func (self *Screen) CursorReload() {
	defer self.AssertIntegrity()

	// Our tracked pin is ALWAYS accurate, so we derive the active
	// point from the pin. If this returns null it means our pin
	// points outside the active area. In that case, we update the
	// pin to be the top-left.
	pt_ := self.Pages.PointFromPin(
		PointTagActive,
		*self.Cursor.PagePin,
	)
	if !pt_.Valid {
		// reset:
		pin_ := self.Pages.Pin(Point{Tag: PointTagActive})
		assert(pin_.Valid) // .?
		pin := pin_.Value
		*self.Cursor.PagePin = pin
		pt_ = self.Pages.PointFromPin(PointTagActive, pin)
		assert(pt_.Valid) // .?
	}
	pt := pt_.Value

	self.Cursor.X = pt.Active.X
	// Zig: @intCast(pt.active.y) to u16.
	assert(pt.Active.Y <= 0xFFFF)
	self.Cursor.Y = SizeCellCountInt(pt.Active.Y)
	page_rac := self.Cursor.PagePin.RowAndCell()
	self.Cursor.PageRow = page_rac.row
	self.Cursor.PageCell = page_rac.cell

	// If we have a style, we need to ensure it is in the page because this
	// method may also be called after a page change.
	if self.Cursor.StyleID != StyleDefaultID {
		if err := self.ManualStyleUpdate(); err != nil {
			// This failure should not happen because manualStyleUpdate
			// handles page splitting, overflow, and more. This should only
			// happen if we're out of RAM. In this case, we'll just degrade
			// gracefully back to the default style.
			warn("screen", "failed to update style on cursor reload err=error.%v", err) // Zig: log.err
			self.Cursor.Style = Style{}
			self.Cursor.StyleID = 0
		}
	}
}

// CursorDownScroll is Screen.cursorDownScroll.
//
// Scroll the active area and keep the cursor at the bottom of the screen.
// This is a very specialized function but it keeps it fast.
//
// Zig: !void. The inferred error set is that of PageList.eraseRow and
// PageList.grow: ErrOutOfMemory (dcs.go) and Page.CloneFromError (page.go).
func (self *Screen) CursorDownScroll() error {
	assert(self.Cursor.Y == self.Pages.rows-1)
	defer self.AssertIntegrity()

	// Zig: `if (comptime build_options.kitty_graphics)` block (scrolling
	// dirties the images): compiled out; see the file header.

	// If we have no scrollback, then we shift all our rows instead.
	if self.NoScrollback {
		// If we have a single-row screen, we have no rows to shift
		// so our cursor is in the correct place we just have to clear
		// the cells.
		if self.Pages.rows == 1 {
			page := &self.Cursor.PagePin.Node.data
			self.ClearCells(
				page,
				self.Cursor.PageRow,
				page.GetCells(self.Cursor.PageRow),
			)
			self.CursorMarkDirty()
		} else {
			// The call to `eraseRow` will move the tracked cursor pin up by one
			// row, but we don't actually want that, so we keep the old pin and
			// put it back after calling `eraseRow`.
			old_pin := *self.Cursor.PagePin

			// eraseRow will shift everything below it up.
			if err := self.Pages.EraseRow(Point{Tag: PointTagActive}); err != nil {
				return err
			}

			// Note we don't need to mark anything dirty in this branch
			// because eraseRow will mark all the rotated rows as dirty
			// in the entire page.

			// We don't use `cursorChangePin` here because we aren't
			// actually changing the pin, we're keeping it the same.
			*self.Cursor.PagePin = old_pin

			// We do, however, need to refresh the cached page row
			// and cell, because `eraseRow` will have moved the row.
			page_rac := self.Cursor.PagePin.RowAndCell()
			self.Cursor.PageRow = page_rac.row
			self.Cursor.PageCell = page_rac.cell
		}
	} else {
		old_pin := *self.Cursor.PagePin

		// Grow our pages by one row. The PageList will handle if we need to
		// allocate, prune scrollback, whatever.
		if _, err := self.Pages.Grow(); err != nil {
			return err
		}

		// Zig: the labeled block `new_pin:` is the argument of
		// cursorChangePin.
		var new_pin PageListPin
		{
			// We do this all in a block here because referencing this pin
			// after cursorChangePin is unsafe, and we want to keep it out
			// of scope.

			// If our pin page change it means that the page that the pin
			// was on was pruned. In this case, grow() moves the pin to
			// the top-left of the new page. This effectively moves it by
			// one already, we just need to fix up the x value.
			var page_pin PageListPin
			if old_pin.Node == self.Cursor.PagePin.Node {
				down := self.Cursor.PagePin.Down(1)
				assert(down.Valid) // .?
				page_pin = down.Value
			} else {
				// reuse:
				pin := *self.Cursor.PagePin
				pin.X = self.Cursor.X
				page_pin = pin
			}

			// These assertions help catch some pagelist math errors. Our
			// x/y should be unchanged after the grow.
			if pageSlowRuntimeSafety {
				active_ := self.Pages.PointFromPin(
					PointTagActive,
					page_pin,
				)
				assert(active_.Valid) // .?
				active := active_.Value.Active
				assert(active.X == self.Cursor.X)
				assert(active.Y == uint32(self.Cursor.Y))
			}

			new_pin = page_pin
		}
		self.cursorChangePin(new_pin)
		page_rac := self.Cursor.PagePin.RowAndCell()
		self.Cursor.PageRow = page_rac.row
		self.Cursor.PageCell = page_rac.cell

		// Our new row is always dirty
		self.CursorMarkDirty()

		// Clear the new row so it gets our bg color. We only do this
		// if we have a bg color at all.
		if self.Cursor.Style.BgColor.Tag != StyleColorNone {
			page := &self.Cursor.PagePin.Node.data
			self.ClearCells(
				page,
				self.Cursor.PageRow,
				page.GetCells(self.Cursor.PageRow),
			)
		}
	}

	if self.Cursor.StyleID != StyleDefaultID {
		// The newly created line needs to be styled according to
		// the bg color if it is set.
		if blank_cell := self.Cursor.Style.BgCell(); blank_cell.Valid {
			// Zig: `cell_current - self.cursor.x` on the many-item
			// pointer, then `@memset(cells[0..self.pages.cols], blank_cell)`.
			page := &self.Cursor.PagePin.Node.data
			start := page.cellIndex(self.Cursor.PageCell) - uint(self.Cursor.X)
			cells := page.cells[start : start+uint(self.Pages.cols)]
			for i := range cells {
				cells[i] = blank_cell.Value
			}
		}
	}

	return nil
}

// CursorScrollAbove is Screen.cursorScrollAbove.
//
// This scrolls the active area at and above the cursor.
// The lines below the cursor are not scrolled.
//
// Zig: !void. The inferred error set is that of cursorDownScroll,
// PageList.grow and cursorScrollAboveRotate: ErrOutOfMemory (dcs.go) and
// Page.CloneFromError (page.go).
func (self *Screen) CursorScrollAbove() error {
	// We unconditionally mark the cursor row as dirty here because
	// the cursor always changes page rows inside this function, and
	// when that happens it can mean the text in the old row needs to
	// be re-shaped because the cursor splits runs to break ligatures.
	self.CursorMarkDirty()

	// If the cursor is on the bottom of the screen, its faster to use
	// our specialized function for that case.
	if self.Cursor.Y == self.Pages.rows-1 {
		return self.CursorDownScroll()
	}

	defer self.AssertIntegrity()

	// Logic below assumes we always have at least one row that isn't moving
	assert(self.Cursor.Y < self.Pages.rows-1)

	// Explanation:
	//  We don't actually move everything that's at or above the cursor row,
	//  since this would require us to shift up our ENTIRE scrollback, which
	//  would be ridiculously expensive. Instead, we insert a new row at the
	//  end of the pagelist (`grow()`), and move everything BELOW the cursor
	//  DOWN by one row. This has the same practical result but it's a whole
	//  lot cheaper in 99% of cases.

	old_pin := *self.Cursor.PagePin
	grown, err := self.Pages.Grow()
	if err != nil {
		return err
	}
	if grown != nil {
		if err := self.cursorScrollAboveRotate(); err != nil {
			return err
		}
	} else {
		// In this case, it means grow() didn't allocate a new page.

		if self.Cursor.PagePin.Node == self.Pages.pages.last {
			// If we're on the last page we can do a very fast path because
			// all the rows we need to move around are within a single page.

			// Note: we don't need to call cursorChangePin here because
			// the pin page is the same so there is no accounting to do
			// for styles or any of that.
			assert(old_pin.Node == self.Cursor.PagePin.Node)
			down := self.Cursor.PagePin.Down(1)
			assert(down.Valid) // .?
			*self.Cursor.PagePin = down.Value

			pin := self.Cursor.PagePin
			page := &self.Cursor.PagePin.Node.data

			// Rotate the rows so that the newly created empty row is at the
			// beginning. e.g. [ 0 1 2 3 ] in to [ 3 0 1 2 ].
			rows := page.rows
			screenRotateOnceR(rows[pin.Y:page.size.rows])

			// Mark the whole page as dirty.
			//
			// Technically we only need to mark from the cursor row to the
			// end but this is a hot function, so we want to minimize work.
			page.dirty = true

			// Setup our cursor caches after the rotation so it points to the
			// correct data
			page_rac := self.Cursor.PagePin.RowAndCell()
			self.Cursor.PageRow = page_rac.row
			self.Cursor.PageCell = page_rac.cell
		} else {
			// We didn't grow pages but our cursor isn't on the last page.
			// In this case we need to do more work because we need to copy
			// elements between pages.
			//
			// An example scenario of this is shown below:
			//
			//      +----------+ = PAGE 0
			//  ... :          :
			//     +-------------+ ACTIVE
			// 4302 |1A00000000| | 0
			// 4303 |2B00000000| | 1
			//      :^         : : = PIN 0
			// 4304 |3C00000000| | 2
			//      +----------+ :
			//      +----------+ : = PAGE 1
			//    0 |4D00000000| | 3
			//    1 |5E00000000| | 4
			//      +----------+ :
			//     +-------------+
			if err := self.cursorScrollAboveRotate(); err != nil {
				return err
			}
		}
	}

	if self.Cursor.StyleID != StyleDefaultID {
		// The newly created line needs to be styled according to
		// the bg color if it is set.
		if blank_cell := self.Cursor.Style.BgCell(); blank_cell.Valid {
			// Zig: `cell_current - self.cursor.x` on the many-item
			// pointer, then `@memset(cells[0..self.pages.cols], blank_cell)`.
			page := &self.Cursor.PagePin.Node.data
			start := page.cellIndex(self.Cursor.PageCell) - uint(self.Cursor.X)
			cells := page.cells[start : start+uint(self.Pages.cols)]
			for i := range cells {
				cells[i] = blank_cell.Value
			}
		}
	}

	return nil
}

// cursorScrollAboveRotate is Screen.cursorScrollAboveRotate.
//
// Zig: !void. The inferred error set is that of Page.cloneRowFrom:
// Page.CloneFromError (page.go).
func (self *Screen) cursorScrollAboveRotate() error {
	down := self.Cursor.PagePin.Down(1)
	assert(down.Valid) // .?
	self.cursorChangePin(down.Value)

	// Go through each of the pages following our pin, shift all rows
	// down by one, and copy the last row of the previous page.
	current := self.Pages.pages.last
	assert(current != nil) // .?
	for current != self.Cursor.PagePin.Node {
		prev := current.prev
		assert(prev != nil) // .?
		prev_page := &prev.data
		cur_page := &current.data
		prev_rows := prev_page.rows
		cur_rows := cur_page.rows

		// Rotate the pages down: [ 0 1 2 3 ] => [ 3 0 1 2 ]
		screenRotateOnceR(cur_rows[0:cur_page.size.rows])

		// Copy the last row of the previous page to the top of current.
		assert(prev_page.size.rows > 0) // Zig: - traps
		if err := cur_page.CloneRowFrom(
			prev_page,
			&cur_rows[0],
			&prev_rows[prev_page.size.rows-1],
		); err != nil {
			return err
		}

		// Mark dirty on the page, since we are dirtying all rows with this.
		cur_page.dirty = true

		// Zig: the continue expression `current = current.prev.?`.
		current = current.prev
		assert(current != nil) // .?
	}

	// Our current is our cursor page, we need to rotate down from
	// our cursor and clear our row.
	assert(current == self.Cursor.PagePin.Node)
	cur_page := &current.data
	cur_rows := cur_page.rows
	screenRotateOnceR(cur_rows[self.Cursor.PagePin.Y:cur_page.size.rows])
	self.ClearCells(
		cur_page,
		&cur_rows[self.Cursor.PagePin.Y],
		cur_page.GetCells(&cur_rows[self.Cursor.PagePin.Y]),
	)

	// Mark the whole page as dirty.
	//
	// Technically we only need to mark from the cursor row to the
	// end but this is a hot function, so we want to minimize work.
	cur_page.dirty = true

	// Setup cursor cache data after all the rotations so our
	// row is valid.
	page_rac := self.Cursor.PagePin.RowAndCell()
	self.Cursor.PageRow = page_rac.row
	self.Cursor.PageCell = page_rac.cell

	return nil
}

// screenRotateOnceR is NOT a Zig declaration: it is fastmem.rotateOnceR
// (src/fastmem.zig) for rows.
//
// Moves the last item to the start.
// Reverse operation of `fastmem.rotateOnce`.
//
// e.g. `0 1 2 3` -> `3 0 1 2`.
//
// An empty slice panics on `items[len - 1]` as Zig traps on the
// `items.len - 1` underflow.
func screenRotateOnceR(items []PageRow) {
	tmp := items[len(items)-1]
	copy(items[1:], items[0:len(items)-1]) // fastmem.move: a memmove
	items[0] = tmp
}

// cursorDownOrScroll is Screen.cursorDownOrScroll.
//
// Move the cursor down if we're not at the bottom of the screen. Otherwise
// scroll. Currently only used for testing.
//
// Zig: !void. The inferred error set is that of cursorDownScroll.
func (self *Screen) cursorDownOrScroll() error {
	// Zig: `self.cursor.y + 1` is u16; y < rows, so it cannot overflow.
	if uint32(self.Cursor.Y)+1 < uint32(self.Pages.rows) {
		self.CursorDown(1)
	} else {
		if err := self.CursorDownScroll(); err != nil {
			return err
		}
	}
	return nil
}

// ScreenCursorCopyOptions is NOT a named Zig declaration: it is the
// anonymous struct that is the `opts` parameter of Screen.cursorCopy.
type ScreenCursorCopyOptions struct {
	// Copy the hyperlink from the other cursor. If not set, this will
	// clear our current hyperlink.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default (`.{}`) must set it.
	Hyperlink bool
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
func (self *Screen) CursorCopy(other ScreenCursor, opts ScreenCursorCopyOptions) (err error) {
	assert(other.X < self.Pages.cols)
	assert(other.Y < self.Pages.rows)

	// End any currently active hyperlink on our cursor.
	self.EndHyperlink()

	old := self.Cursor
	self.Cursor = other
	defer func() {
		// errdefer self.cursor = old;
		if err != nil {
			self.Cursor = old
		}
	}()

	// Keep our old style ID so it can be properly cleaned up below.
	self.Cursor.StyleID = old.StyleID

	// Hyperlinks will be managed separately below.
	self.Cursor.HyperlinkID = 0
	self.Cursor.Hyperlink = nil

	// Keep our old page pin and X/Y because:
	// 1. The old style will need to be cleaned up from the page it's from.
	// 2. The new position navigated to by `cursorAbsolute` needs to be in our
	//    own screen.
	self.Cursor.PagePin = old.PagePin
	self.Cursor.X = old.X
	self.Cursor.Y = old.Y

	// Call manual style update in order to clean up our old style, if we have
	// one, and also to load the style from the other cursor, if it had one.
	if err := self.ManualStyleUpdate(); err != nil {
		return err
	}

	// Move to the correct location to match the other cursor.
	self.CursorAbsolute(other.X, other.Y)

	// If the other cursor had a hyperlink, add it to ours.
	if opts.Hyperlink && other.HyperlinkID != 0 {
		// Get the hyperlink from the other cursor's page.
		other_page := &other.PagePin.Node.data
		other_link := other_page.hyperlink_set.Get(other.HyperlinkID)

		uri := other_link.uri
		var id_ Optional[[]uint8]
		switch other_link.id.Tag {
		case HyperlinkPageEntryIdExplicit:
			id_ = Some(other_link.id.Explicit)
		case HyperlinkPageEntryIdImplicit:
			id_ = None[[]uint8]()
		default:
			panic("unreachable")
		}

		// And it to our cursor.
		if err := self.StartHyperlink(uri, id_); err != nil {
			// This shouldn't happen because startHyperlink should handle
			// resizing. This only happens if we're truly out of RAM. Degrade
			// to forgetting the hyperlink.
			warn("screen", "failed to update hyperlink on cursor change err=error.%v", err) // Zig: log.err
		}
	}

	return nil
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
func (self *Screen) cursorChangePin(new PageListPin) {
	// Moving the cursor affects text run splitting (ligatures) so
	// we must mark the old and new page dirty. We do this as long
	// as the pins are not equal
	if !self.Cursor.PagePin.Eql(new) {
		self.CursorMarkDirty()
		new.MarkDirty()
	}

	// If our pin is on the same page, then we can just update the pin.
	// We don't need to migrate any state.
	if self.Cursor.PagePin.Node == new.Node {
		*self.Cursor.PagePin = new
		return
	}

	// If we have an old style then we need to release it from the old page.
	var old_style_ Optional[Style]
	if self.Cursor.StyleID == StyleDefaultID {
		old_style_ = None[Style]()
	} else {
		old_style_ = Some(self.Cursor.Style)
	}
	if old_style_.Valid {
		// Release the style directly from the old page instead of going through
		// manualStyleUpdate, because the cursor position may have already been
		// updated but the pin has not, which would fail integrity checks.
		old_page := &self.Cursor.PagePin.Node.data
		old_page.styles.Release(self.Cursor.StyleID)
		self.Cursor.Style = Style{}
		self.Cursor.StyleID = StyleDefaultID
	}

	// If we have a hyperlink then we need to release it from the old page.
	if self.Cursor.Hyperlink != nil {
		old_page := &self.Cursor.PagePin.Node.data
		old_page.hyperlink_set.Release(self.Cursor.HyperlinkID)
	}

	// Update our pin to the new page
	*self.Cursor.PagePin = new

	// On the new page, we need to migrate our style
	if old_style_.Valid {
		old_style := old_style_.Value
		self.Cursor.Style = old_style
		if err := self.ManualStyleUpdate(); err != nil {
			// This failure should not happen because manualStyleUpdate
			// handles page splitting, overflow, and more. This should only
			// happen if we're out of RAM. In this case, we'll just degrade
			// gracefully back to the default style.
			warn("screen", "failed to update style on cursor change err=error.%v", err) // Zig: log.err
			self.Cursor.Style = Style{}
			self.Cursor.StyleID = 0
		}
	}

	// On the new page, we need to migrate our hyperlink
	if self.Cursor.Hyperlink != nil {
		link := self.Cursor.Hyperlink

		// So we don't attempt to free any memory in the replaced page.
		self.Cursor.HyperlinkID = 0
		self.Cursor.Hyperlink = nil

		// Re-add
		var id_ Optional[[]uint8]
		switch link.ID.Tag {
		case HyperlinkHyperlinkIdExplicit:
			id_ = Some(link.ID.Explicit)
		case HyperlinkHyperlinkIdImplicit:
			id_ = None[[]uint8]()
		default:
			panic("unreachable")
		}
		if err := self.StartHyperlink(link.URI, id_); err != nil {
			// This shouldn't happen because startHyperlink should handle
			// resizing. This only happens if we're truly out of RAM. Degrade
			// to forgetting the hyperlink.
			warn("screen", "failed to update hyperlink on cursor change err=error.%v", err) // Zig: log.err
		}

		// Remove our old link
		link.Deinit()
		// Zig: self.alloc.destroy(link); Go memory.
	}
}

// CursorMarkDirty is Screen.cursorMarkDirty.
//
// Mark the cursor position as dirty.
// TODO: test
func (self *Screen) CursorMarkDirty() {
	self.Cursor.PageRow.dirty = true
}

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
func (self *Screen) CursorResetWrap() {
	// Reset the cursor's pending wrap state
	self.Cursor.PendingWrap = false

	page_row := self.Cursor.PageRow

	if !page_row.wrap {
		return
	}

	// This row does not wrap and the next row is not wrapped to
	page_row.wrap = false

	if next_row := self.Cursor.PagePin.Down(1); next_row.Valid {
		next_row.Value.RowAndCell().row.wrap_continuation = false
	}

	// If the last cell in the row is a spacer head we need to clear it.
	cells := self.Cursor.PagePin.Cells(PageListPinCellSubsetAll)
	cell := cells[self.Cursor.PagePin.Node.data.size.cols-1]
	if cell.wide == PageCellWideSpacerHead {
		self.ClearCells(
			&self.Cursor.PagePin.Node.data,
			page_row,
			cells[self.Cursor.PagePin.Node.data.size.cols-1:][0:1],
		)
	}
}

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
	// Zig: isize (int64, the reference's width).
	DeltaRow int64
	// Zig: isize (int64, the reference's width).
	DeltaPrompt int64
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
func (self ScreenScrollTag) String() string {
	return screenScrollTagNames[self]
}

// screenScrollTagNames are the tag names of ScreenScrollTag, indexed by
// value (Zig's @tagName).
var screenScrollTagNames = [...]string{"active", "top", "pin", "row", "delta_row", "delta_prompt"}

// Scroll is Screen.scroll.
//
// Scroll the viewport of the terminal grid.
func (self *Screen) Scroll(behavior ScreenScroll) {
	defer self.AssertIntegrity()

	// Zig: `if (comptime build_options.kitty_graphics)` block (scrolling
	// marks the image state dirty): compiled out; see the file header.

	switch behavior.Tag {
	case ScreenScrollActive:
		self.Pages.Scroll(PageListScroll{Tag: PageListScrollActive})
	case ScreenScrollTop:
		self.Pages.Scroll(PageListScroll{Tag: PageListScrollTop})
	case ScreenScrollPin:
		self.Pages.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: behavior.Pin})
	case ScreenScrollRow:
		self.Pages.Scroll(PageListScroll{Tag: PageListScrollRow, Row: behavior.Row})
	case ScreenScrollDeltaRow:
		self.Pages.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: behavior.DeltaRow})
	case ScreenScrollDeltaPrompt:
		self.Pages.Scroll(PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: behavior.DeltaPrompt})
	default:
		panic("unreachable")
	}
}

// ScrollClear is Screen.scrollClear.
//
// See PageList.scrollClear. In addition to that, we reset the cursor
// to be on top.
//
// Zig: !void. The inferred error set is that of PageList.scrollClear:
// ErrOutOfMemory (dcs.go).
func (self *Screen) ScrollClear() error {
	defer self.AssertIntegrity()

	if err := self.Pages.ScrollClear(); err != nil {
		return err
	}
	self.CursorReload()

	// Zig: `if (comptime build_options.kitty_graphics)` block (scrolling
	// marks the image state dirty): compiled out; see the file header.

	return nil
}

// ViewportIsBottom is Screen.viewportIsBottom.
//
// Returns true if the viewport is scrolled to the bottom of the screen.
//
// Zig: pub inline fn viewportIsBottom(self: Screen) bool. Zig passes the
// screen by value; Go uses a pointer receiver like every other method of
// Screen (a Screen is not meant to be copied).
func (self *Screen) ViewportIsBottom() bool {
	return self.Pages.viewport.Tag == PageListViewportActive
}

// EraseRows is Screen.eraseRows.
//
// Erase the region specified by tl and br, inclusive. This will physically
// erase the rows meaning the memory will be reclaimed (if the underlying
// page is empty) and other rows will be shifted up.
//
// Zig: pub inline fn eraseRows(self: *Screen, tl: point.Point,
// bl: ?point.Point) void.
func (self *Screen) EraseRows(tl Point, bl Optional[Point]) {
	defer self.AssertIntegrity()

	// Erase the rows
	self.Pages.EraseRows(tl, bl)

	// Just to be safe, reset our cursor since it is possible depending
	// on the points that our active area shifted so our pointers are
	// invalid.
	self.CursorReload()
}

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
	defer self.AssertIntegrity()

	it := self.Pages.PageIterator(PageListDirectionRightDown, tl, bl)
	for chunk_ := it.Next(); chunk_.Valid; chunk_ = it.Next() {
		chunk := chunk_.Value
		rows := chunk.Rows()
		for i := range rows {
			row := &rows[i]
			cells_offset := row.cells
			// Zig: `row.cells.ptr(chunk.node.data.memory)`, sliced.
			cells_multi := chunk.node.data.rowCells(row)
			cells := cells_multi[0:self.Pages.cols]

			// Clear all cells
			if protected {
				self.ClearUnprotectedCells(&chunk.node.data, row, cells)
				// We need to preserve other row attributes since we only
				// cleared unprotected cells.
				row.cells = cells_offset
			} else {
				self.ClearCells(&chunk.node.data, row, cells)
				*row = PageRow{cells: cells_offset}
			}

			row.dirty = true
		}
	}
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
	// This whole operation does unsafe things, so we just want to assert
	// the end state.
	page.PauseIntegrityChecks(true)
	defer func() {
		page.PauseIntegrityChecks(false)
		page.AssertIntegrity()
		self.AssertIntegrity()
	}()

	// Zig: `if (comptime std.debug.runtime_safety)`; the reference is a
	// safe build.
	{
		// Our row and cells should be within the page.
		//
		// Zig compares addresses: the row at or after `&page_rows[0]` and
		// at or before `&page_rows[page.size.rows - 1]`, the first and
		// last cell within the row's cells. rowIndex and cellIndex
		// (page.go) panic for a pointer that is not in the page at all.
		row_idx := page.rowIndex(row)
		assert(page.size.rows > 0) // Zig: - traps
		assert(row_idx <= uint(page.size.rows-1))

		row_cells := page.GetCells(row)
		assert(page.cellIndex(&cells[0]) >= page.cellIndex(&row_cells[0]))
		assert(page.cellIndex(&cells[len(cells)-1]) <= page.cellIndex(&row_cells[len(row_cells)-1]))
	}

	// If we have managed memory (styles, graphemes, or hyperlinks)
	// in this row then we go cell by cell and clear them if present.
	if row.grapheme {
		for i := range cells {
			cell := &cells[i]
			if cell.HasGrapheme() {
				page.ClearGrapheme(cell)
			}
		}

		// If we have no left/right scroll region we can be sure
		// that we've cleared all the graphemes, so we clear the
		// flag, otherwise we ask the page to update the flag.
		if uint(len(cells)) == uint(self.Pages.cols) {
			row.grapheme = false
		} else {
			page.UpdateRowGraphemeFlag(row)
		}
	}

	if row.hyperlink {
		for i := range cells {
			cell := &cells[i]
			if cell.hyperlink {
				page.ClearHyperlink(cell)
			}
		}

		// If we have no left/right scroll region we can be sure
		// that we've cleared all the hyperlinks, so we clear the
		// flag, otherwise we ask the page to update the flag.
		if uint(len(cells)) == uint(self.Pages.cols) {
			row.hyperlink = false
		} else {
			page.UpdateRowHyperlinkFlag(row)
		}
	}

	if row.styled {
		for i := range cells {
			cell := &cells[i]
			if cell.HasStyling() {
				page.styles.Release(cell.style_id)
			}
		}

		// If we have no left/right scroll region we can be sure
		// that we've cleared all the styles, so we clear the
		// flag, otherwise we ask the page to update the flag.
		if uint(len(cells)) == uint(self.Pages.cols) {
			row.styled = false
		} else {
			page.UpdateRowStyledFlag(row)
		}
	}

	// Zig: `if (comptime build_options.kitty_graphics)` block (the virtual
	// placeholder flag): compiled out; see the file header.

	// @memset(cells, self.blankCell())
	blank := self.BlankCell()
	for i := range cells {
		cells[i] = blank
	}
}

// ClearUnprotectedCells is Screen.clearUnprotectedCells.
//
// Clear cells but only if they are not protected.
//
// Zig: pub fn clearUnprotectedCells(self: *Screen, page: *Page, row: *Row,
// cells: []Cell) void. `cells` is a sub-slice of the page's cells.
func (self *Screen) ClearUnprotectedCells(page *Page, row *PageRow, cells []PageCell) {
	var x0 uint = 0
	var x1 uint = 0

	// Zig: `while (x0 < cells.len) clear: { ... }`; `break :clear` leaves
	// the loop body, so it is `continue`.
	for x0 < uint(len(cells)) {
		for cells[x0].protected {
			x0 += 1
			if x0 >= uint(len(cells)) {
				break
			}
		}
		if x0 >= uint(len(cells)) {
			continue // break :clear
		}
		x1 = x0 + 1
		for x1 < uint(len(cells)) && !cells[x1].protected {
			x1 += 1
		}
		self.ClearCells(page, row, cells[x0:x1])
		x0 = x1
	}

	page.AssertIntegrity()
	self.AssertIntegrity()
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
func (self *Screen) SplitCellBoundary(x SizeCellCountInt) {
	page := &self.Cursor.PagePin.Node.data

	page.PauseIntegrityChecks(true)
	defer page.PauseIntegrityChecks(false)

	cols := self.Cursor.PagePin.Node.data.size.cols

	// `x` may be up to an INCLUDING `cols`, since that signifies splitting
	// the boundary to the right of the final cell in the row.
	assert(x <= cols)

	// [ A B C D E F|]
	//              ^ Boundary between final cell and row end.
	if x == cols {
		if !self.Cursor.PageRow.wrap {
			return
		}

		cells := self.Cursor.PagePin.Cells(PageListPinCellSubsetAll)

		// Spacer head at end of wrapped row.
		if cells[cols-1].wide == PageCellWideSpacerHead {
			self.ClearCells(
				page,
				self.Cursor.PageRow,
				cells[cols-1:][0:1],
			)
		}

		return
	}

	// [|A B C D E F ]
	//  ^ Boundary between first cell and row start.
	//
	//  OR
	//
	// [ A|B C D E F ]
	//    ^ Boundary between first cell and second cell.
	//
	// First cell may be a wrapped wide cell with a spacer
	// head on the previous row that needs to be cleared.
	if (x == 0 || x == 1) && self.Cursor.PageRow.wrap_continuation {
		cells := self.Cursor.PagePin.Cells(PageListPinCellSubsetAll)

		// If the first cell in a row is wide the previous row
		// may have a spacer head which needs to be cleared.
		if cells[0].wide == PageCellWideWide {
			if p_row_ := self.Cursor.PagePin.Up(1); p_row_.Valid {
				p_row := p_row_.Value
				p_rac := p_row.RowAndCell()
				p_cells := p_row.Cells(PageListPinCellSubsetAll)
				p_cell := p_cells[p_row.Node.data.size.cols-1]
				if p_cell.wide == PageCellWideSpacerHead {
					self.ClearCells(
						&p_row.Node.data,
						p_rac.row,
						p_cells[p_row.Node.data.size.cols-1:][0:1],
					)
				}
			}
		}
	}

	// If x is 0 then we're done.
	if x == 0 {
		return
	}

	// [ ... X|Y ... ]
	//        ^ Boundary between two cells in the middle of the row.
	{
		assert(x > 0)
		assert(x < cols)

		cells := self.Cursor.PagePin.Cells(PageListPinCellSubsetAll)

		left := cells[x-1]
		switch left.wide {
		// There should not be spacer heads in the middle of the row.
		case PageCellWideSpacerHead:
			panic("unreachable")

		// We don't need to do anything for narrow cells or spacer tails.
		case PageCellWideNarrow, PageCellWideSpacerTail:

		// A wide char would be split, so must be cleared.
		case PageCellWideWide:
			self.ClearCells(
				page,
				self.Cursor.PageRow,
				cells[x-1:][0:2],
			)

		default:
			panic("unreachable")
		}
	}
}

// BlankCell is Screen.blankCell.
//
// Returns the blank cell to use when doing terminal operations that
// require preserving the bg color.
//
// Zig: self is *const Screen.
func (self *Screen) BlankCell() PageCell {
	if self.Cursor.StyleID == StyleDefaultID {
		return PageCell{}
	}
	if bg := self.Cursor.Style.BgCell(); bg.Valid {
		return bg.Value
	}
	return PageCell{}
}

// ScreenResize is Screen.Resize: the options of resize.
type ScreenResize struct {
	// The new size to resize to
	Cols SizeCellCountInt
	Rows SizeCellCountInt

	// Whether to reflow soft-wrapped text.
	//
	// This will reflow soft-wrapped text. If the screen size is getting
	// smaller and the maximum scrollback size is exceeded, data will be
	// lost from the top of the scrollback.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default must set it.
	Reflow bool

	// Set this to enable prompt redraw on resize. This signals
	// that the running program can redraw the prompt if the cursor is
	// currently at a prompt. This detects OSC133 prompts lines and clears
	// them. If set to `.last`, only the most recent prompt line is cleared.
	//
	// Zig default: .false (OSCParsersSemanticPromptRedrawFalse). The Go
	// zero value is .true; a caller that relies on the default must set
	// it.
	PromptRedraw OSCParsersSemanticPromptRedraw
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
func (self *Screen) Resize(opts ScreenResize) (err error) {
	// Go: a size of zero columns or rows is an error, before anything
	// changes (see Terminal.Resize).
	if opts.Cols == 0 {
		return ErrZeroColCount
	}
	if opts.Rows == 0 {
		return ErrZeroRowCount
	}

	defer self.AssertIntegrity()

	// Zig: `if (comptime build_options.kitty_graphics)` block (mark the
	// image state dirty): compiled out; see the file header.

	// Release the cursor style while resizing just
	// in case the cursor ends up on a different page.
	cursor_style := self.Cursor.Style
	self.Cursor.Style = Style{}
	if err := self.ManualStyleUpdate(); err != nil {
		panic("unreachable") // catch unreachable
	}
	defer func() {
		// Restore the cursor style.
		self.Cursor.Style = cursor_style
		if err := self.ManualStyleUpdate(); err != nil {
			// This failure should not happen because manualStyleUpdate
			// handles page splitting, overflow, and more. This should only
			// happen if we're out of RAM. In this case, we'll just degrade
			// gracefully back to the default style.
			warn("screen", "failed to update style on cursor reload err=error.%v", err) // Zig: log.err
			self.Cursor.Style = Style{}
			self.Cursor.StyleID = 0
		}
	}()

	// If we have a hyperlink, release it from the old page
	// and then we need to re-add it to the new page. This needs
	// to happen because resize below typically reallocates a
	// new page so the old hyperlink is invalid.
	hyperlink_ := self.Cursor.Hyperlink
	if self.Cursor.HyperlinkID != 0 {
		// Note we do NOT use endHyperlink because we want to keep
		// our allocated self.cursor.hyperlink valid.
		page := &self.Cursor.PagePin.Node.data
		page.hyperlink_set.Release(self.Cursor.HyperlinkID)
		self.Cursor.HyperlinkID = 0
		self.Cursor.Hyperlink = nil
	}

	// We need to insert a tracked pin for our saved cursor so we can
	// modify its X/Y for reflow.
	//
	// Zig: the labeled block `saved_cursor:`.
	var saved_cursor_pin *PageListPin
	if self.SavedCursor.Valid {
		sc := self.SavedCursor.Value
		pin := self.Pages.Pin(Point{Tag: PointTagActive, Active: PointCoordinate{
			X: sc.X,
			Y: uint32(sc.Y),
		}})
		if pin.Valid {
			saved_cursor_pin, err = self.Pages.TrackPin(pin.Value)
			if err != nil {
				return err
			}
		}
	}
	defer func() {
		if saved_cursor_pin != nil {
			self.Pages.UntrackPin(saved_cursor_pin)
		}
	}()

	// If our cursor is on a prompt or input line, clear it so the shell can
	// redraw it. This works with OSC 133 semantic prompts.
	//
	// We check cursor.semantic_content rather than page_row.semantic_prompt
	// because some shells (e.g., Nu) mark input areas with OSC 133 B but don't
	// mark continuation lines with k=s. If the input spans multiple lines and
	// continuation lines are unmarked, checking only page_row.semantic_prompt
	// would miss them. By checking semantic_content, we assume that if the
	// cursor is on anything other than command output, we're at a prompt/input
	// line and should clear from there.
	//
	// Zig: the labeled block `prompt:`; `break :prompt` is a return from the
	// closure.
	if opts.PromptRedraw != OSCParsersSemanticPromptRedrawFalse &&
		self.Cursor.SemanticContent != PageCellSemanticContentOutput {
		func() {
			switch opts.PromptRedraw {
			case OSCParsersSemanticPromptRedrawFalse:
				panic("unreachable")

			// For `.last`, only clear the current line where the cursor is.
			// For `.true`, clear all prompt lines starting from the beginning.
			case OSCParsersSemanticPromptRedrawLast:
				page := &self.Cursor.PagePin.Node.data
				row := self.Cursor.PageRow
				cells := page.GetCells(row)
				self.ClearCells(page, row, cells)

			case OSCParsersSemanticPromptRedrawTrue:
				// start:
				var start PageListPin
				{
					it := self.Cursor.PagePin.PromptIterator(
						PageListDirectionLeftUp,
						None[PageListPin](),
					)
					next := it.Next()
					if !next.Valid {
						// This should never happen because promptIterator should always
						// find a prompt if we already verified our row is some kind of
						// prompt.
						warn("screen", "cursor on prompt line but promptIterator found no prompt")
						return // break :prompt
					}
					start = next.Value
				}

				// Clear cells from our start down. We replace it with spaces,
				// and do not physically erase the rows (eraseRows) because the
				// shell is going to expect this space to be available.
				it := start.RowIterator(PageListDirectionRightDown, None[PageListPin]())
				for pin_ := it.Next(); pin_.Valid; pin_ = it.Next() {
					pin := pin_.Value
					page := &pin.Node.data
					row := pin.RowAndCell().row
					cells := page.GetCells(row)
					self.ClearCells(page, row, cells)
				}

			default:
				panic("unreachable")
			}
		}()
	}

	// Perform the resize operation.
	if err := self.Pages.Resize(PageListResize{
		Rows:   Some(opts.Rows),
		Cols:   Some(opts.Cols),
		Reflow: opts.Reflow,
		Cursor: Some(PageListResizeCursor{x: self.Cursor.X, y: self.Cursor.Y}),
	}); err != nil {
		return err
	}

	// If we have no scrollback and we shrunk our rows, we must explicitly
	// erase our history. This is because PageList always keeps at least
	// a page size of history.
	if self.NoScrollback {
		self.Pages.EraseRows(Point{Tag: PointTagHistory}, None[Point]())
	}

	// If our cursor was updated, we do a full reload so all our cursor
	// state is correct.
	self.CursorReload()

	// If we reflowed a saved cursor, update it.
	if saved_cursor_pin != nil {
		p := saved_cursor_pin
		// This should never fail because a non-null saved_cursor_pin
		// implies a non-null saved_cursor.
		assert(self.SavedCursor.Valid) // .?
		sc := &self.SavedCursor.Value
		if pt_ := self.Pages.PointFromPin(PointTagActive, *p); pt_.Valid {
			pt := pt_.Value
			sc.X = pt.Active.X
			// Zig: @intCast(pt.active.y) to u16.
			assert(pt.Active.Y <= 0xFFFF)
			sc.Y = SizeCellCountInt(pt.Active.Y)

			// If we had pending wrap set and we're no longer at the end of
			// the line, we unset the pending wrap and move the cursor to
			// reflect the correct next position.
			if sc.PendingWrap {
				// Zig: `opts.cols - 1` is a u16 subtraction, evaluated
				// only when pending_wrap is set; it traps at cols 0.
				assert(opts.Cols > 0)
				if sc.X != opts.Cols-1 {
					sc.PendingWrap = false
					sc.X += 1
				}
			}
		} else {
			// I think this can happen if the screen is resized to be
			// less rows or less cols and our saved cursor moves outside
			// the active area. In this case, there isn't anything really
			// reasonable we can do so we just move the cursor to the
			// top-left. It may be reasonable to also move the cursor to
			// match the primary cursor. Any behavior is fine since this is
			// totally unspecified.
			sc.X = 0
			sc.Y = 0
			sc.PendingWrap = false
		}
	}

	// Fix up our hyperlink if we had one.
	if hyperlink_ != nil {
		link := hyperlink_
		var id_ Optional[[]uint8]
		switch link.ID.Tag {
		case HyperlinkHyperlinkIdExplicit:
			id_ = Some(link.ID.Explicit)
		case HyperlinkHyperlinkIdImplicit:
			id_ = None[[]uint8]()
		default:
			panic("unreachable")
		}
		if err := self.StartHyperlink(link.URI, id_); err != nil {
			// This shouldn't happen because startHyperlink should handle
			// resizing. This only happens if we're truly out of RAM. Degrade
			// to forgetting the hyperlink.
			warn("screen", "failed to update hyperlink on resize err=error.%v", err) // Zig: log.err
		}

		// Remove our old link
		link.Deinit()
		// Zig: self.alloc.destroy(link); Go memory.
	}

	return nil
}

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
func (self *Screen) SetAttribute(attr SGRAttribute) (err error) {
	// If we fail to set our style for any reason, we should revert
	// back to the old style. If we fail to do that, we revert back to
	// the default style.
	old_style := self.Cursor.Style
	defer func() {
		// errdefer
		if err != nil {
			self.Cursor.Style = old_style
			if err := self.ManualStyleUpdate(); err != nil {
				warn("screen", "setAttribute error restoring old style after failure err=error.%v", err)
				self.Cursor.Style = Style{}
				if err := self.ManualStyleUpdate(); err != nil {
					panic("unreachable") // catch unreachable
				}
			}
		}
	}()

	switch attr.Tag {
	case SGRAttributeTagUnset:
		self.Cursor.Style = Style{}

	case SGRAttributeTagBold:
		self.Cursor.Style.Flags.Bold = true

	case SGRAttributeTagResetBold:
		// Bold and faint share the same SGR code for this
		self.Cursor.Style.Flags.Bold = false
		self.Cursor.Style.Flags.Faint = false

	case SGRAttributeTagItalic:
		self.Cursor.Style.Flags.Italic = true

	case SGRAttributeTagResetItalic:
		self.Cursor.Style.Flags.Italic = false

	case SGRAttributeTagFaint:
		self.Cursor.Style.Flags.Faint = true

	case SGRAttributeTagUnderline:
		self.Cursor.Style.Flags.Underline = attr.Underline

	case SGRAttributeTagUnderlineColor:
		rgb := attr.UnderlineColor
		self.Cursor.Style.underline_color = StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
			R: rgb.R,
			G: rgb.G,
			B: rgb.B,
		}}

	case SGRAttributeTag256UnderlineColor:
		self.Cursor.Style.underline_color = StyleColor{Tag: StyleColorPalette, Palette: attr.X256UnderlineColor}

	case SGRAttributeTagResetUnderlineColor:
		self.Cursor.Style.underline_color = StyleColor{Tag: StyleColorNone}

	case SGRAttributeTagOverline:
		self.Cursor.Style.Flags.Overline = true

	case SGRAttributeTagResetOverline:
		self.Cursor.Style.Flags.Overline = false

	case SGRAttributeTagBlink:
		self.Cursor.Style.Flags.Blink = true

	case SGRAttributeTagResetBlink:
		self.Cursor.Style.Flags.Blink = false

	case SGRAttributeTagInverse:
		self.Cursor.Style.Flags.Inverse = true

	case SGRAttributeTagResetInverse:
		self.Cursor.Style.Flags.Inverse = false

	case SGRAttributeTagInvisible:
		self.Cursor.Style.Flags.Invisible = true

	case SGRAttributeTagResetInvisible:
		self.Cursor.Style.Flags.Invisible = false

	case SGRAttributeTagStrikethrough:
		self.Cursor.Style.Flags.Strikethrough = true

	case SGRAttributeTagResetStrikethrough:
		self.Cursor.Style.Flags.Strikethrough = false

	case SGRAttributeTagDirectColorFg:
		rgb := attr.DirectColorFg
		self.Cursor.Style.FgColor = StyleColor{
			Tag: StyleColorRgb,
			Rgb: ColorRGB{
				R: rgb.R,
				G: rgb.G,
				B: rgb.B,
			},
		}

	case SGRAttributeTagDirectColorBg:
		rgb := attr.DirectColorBg
		self.Cursor.Style.BgColor = StyleColor{
			Tag: StyleColorRgb,
			Rgb: ColorRGB{
				R: rgb.R,
				G: rgb.G,
				B: rgb.B,
			},
		}

	case SGRAttributeTag8Fg:
		self.Cursor.Style.FgColor = StyleColor{Tag: StyleColorPalette, Palette: uint8(attr.X8Fg)}

	case SGRAttributeTag8Bg:
		self.Cursor.Style.BgColor = StyleColor{Tag: StyleColorPalette, Palette: uint8(attr.X8Bg)}

	case SGRAttributeTagResetFg:
		self.Cursor.Style.FgColor = StyleColor{Tag: StyleColorNone}

	case SGRAttributeTagResetBg:
		self.Cursor.Style.BgColor = StyleColor{Tag: StyleColorNone}

	case SGRAttributeTag8BrightFg:
		self.Cursor.Style.FgColor = StyleColor{Tag: StyleColorPalette, Palette: uint8(attr.X8BrightFg)}

	case SGRAttributeTag8BrightBg:
		self.Cursor.Style.BgColor = StyleColor{Tag: StyleColorPalette, Palette: uint8(attr.X8BrightBg)}

	case SGRAttributeTag256Fg:
		self.Cursor.Style.FgColor = StyleColor{Tag: StyleColorPalette, Palette: attr.X256Fg}

	case SGRAttributeTag256Bg:
		self.Cursor.Style.BgColor = StyleColor{Tag: StyleColorPalette, Palette: attr.X256Bg}

	case SGRAttributeTagUnknown:
		return nil

	default:
		panic("unreachable")
	}

	return self.ManualStyleUpdate()
}

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
func (self *Screen) ManualStyleUpdate() error {
	defer self.AssertIntegrity()
	page := &self.Cursor.PagePin.Node.data

	// std.log.warn("active styles={}", .{page.styles.count()});

	// Release our previous style if it was not default.
	if self.Cursor.StyleID != StyleDefaultID {
		page.styles.Release(self.Cursor.StyleID)
	}

	// If our new style is the default, just reset to that
	if self.Cursor.Style.Default() {
		self.Cursor.StyleID = StyleDefaultID
		return nil
	}

	// Clear the cursor style ID to prevent weird things from happening
	// if the page capacity has to be adjusted which would end up calling
	// manualStyleUpdate again.
	//
	// This also ensures that if anything fails below, we fall back to
	// clearing our style.
	self.Cursor.StyleID = StyleDefaultID

	// After setting the style, we need to update our style map.
	// Note that we COULD lazily do this in print. We should look into
	// if that makes a meaningful difference. Our priority is to keep print
	// fast because setting a ton of styles that do nothing is uncommon
	// and weird.
	id, err := page.styles.Add(
		self.Cursor.Style,
	)
	if err != nil {
		// id:
		// Our style map is full or needs to be rehashed, so we need to
		// increase style capacity (or rehash).
		var adjustment Optional[PageListIncreaseCapacity]
		switch err {
		case ErrOutOfMemory:
			adjustment = Some(PageListIncreaseCapacityStyles)
		case ErrNeedsRehash:
			adjustment = None[PageListIncreaseCapacity]()
		default:
			panic("unreachable")
		}
		node, increase_err := self.IncreaseCapacity(
			self.Cursor.PagePin.Node,
			adjustment,
		)
		if increase_err != nil {
			switch increase_err {
			case ErrOutOfMemory:
				return ErrOutOfMemory
			case ErrOutOfSpace:
				// space:
				// Out of space, we need to split the page. Split wherever
				// is using less capacity and hope that works. If it doesn't
				// work, we tried.
				if err := self.splitForCapacity(*self.Cursor.PagePin); err != nil {
					return err
				}
				node = self.Cursor.PagePin.Node
			default:
				panic("unreachable")
			}
		}

		page = &node.data
		id2, err2 := page.styles.Add(
			self.Cursor.Style,
		)
		if err2 != nil {
			switch err2 {
			case ErrOutOfMemory:
				// This shouldn't happen because increaseCapacity is
				// guaranteed to increase our capacity by at least one and
				// we only need one space, but again, I don't want to crash
				// here so let's log loudly and reset.
				warn("screen", "style addition failed after capacity increase") // Zig: log.err
				return ErrOutOfMemory
			case ErrNeedsRehash:
				// This should be impossible because we rehash above
				// and rehashing should never result in a duplicate. But
				// we don't want to simply hard crash so log it and
				// clear our style.
				warn("screen", "style rehash resulted in needs rehash") // Zig: log.err
				return nil
			default:
				panic("unreachable")
			}
		}
		id = id2
	}
	// Zig: errdefer page.styles.release(page.memory, id): nothing fails
	// after it.

	self.Cursor.StyleID = id
	return nil
}

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
func (self *Screen) splitForCapacity(pin PageListPin) error {
	// Get our capacities. We include our target row because its
	// capacity will be preserved.
	//
	// term51 (docs/mapping.md, "Page-layer bookkeeping"; term51
	// docs/upstream-crashes.md): upstream traps in exactRowCapacity's
	// `.styles = @intCast(styles_cap)` when one of the two ranges references
	// every id of a full style set of the maximum capacity. The port returns
	// ErrOutOfSpace (the page cannot be split) before anything has changed.
	cap_above, err := pin.Node.data.exactRowCapacityOrError(
		0,
		uint(pin.Y)+1,
	)
	if err != nil {
		return err
	}
	bytes_above := PageLayoutOf(cap_above).total_size
	cap_below, err := pin.Node.data.exactRowCapacityOrError(
		uint(pin.Y),
		uint(pin.Node.data.size.rows),
	)
	if err != nil {
		return err
	}
	bytes_below := PageLayoutOf(cap_below).total_size

	// We need to track the old cursor pin because if our split
	// moves the cursor pin we need to update our accounting.
	old_cursor := *self.Cursor.PagePin

	// If our bytes above are less than bytes below, we move the pin
	// to split down one since splitting includes the pinned row in
	// the new node.
	split_pin := pin
	if bytes_above < bytes_below {
		if down := pin.Down(1); down.Valid {
			split_pin = down.Value
		}
	}
	if err := self.Pages.Split(split_pin); err != nil {
		return err
	}

	// Cursor didn't change nodes, we're done.
	if self.Cursor.PagePin.Node == old_cursor.Node {
		return nil
	}

	// Cursor changed, we need to restore the old pin then use
	// cursorChangePin to move to the new pin. The old node is guaranteed
	// to still exist, just not the row.
	//
	// Note that page_row and all that will be invalid, it points to the
	// new node, but at the time of writing this we don't need any of that
	// to be right in cursorChangePin.
	new_cursor := *self.Cursor.PagePin
	*self.Cursor.PagePin = old_cursor
	self.cursorChangePin(new_cursor)

	// term51 (ruled departure, docs/mapping.md "Page-layer bookkeeping";
	// term51 docs/upstream-crashes.md and upstream-findings.md section 15,
	// Screen items 6 and 33): upstream stops here and leaves page_row and
	// page_cell pointing at the row of the OLD page, which the split cleared
	// and hid. Its release build then prints into that hidden row and later
	// leaves a styled cell in a row not marked styled, a state its own
	// integrity check rejects (its Debug build aborts). The port reloads the
	// cached row and cell from the moved pin, as cursorAbsolute does after
	// cursorChangePin; x and y are unchanged (the split does not move the
	// cursor on screen).
	page_rac := self.Cursor.PagePin.RowAndCell()
	self.Cursor.PageRow = page_rac.row
	self.Cursor.PageCell = page_rac.cell
	return nil
}

// AppendGrapheme is Screen.appendGrapheme.
//
// Append a grapheme to the given cell within the current cursor row.
//
// Zig: pub fn appendGrapheme(self: *Screen, cell: *Cell, cp: u21)
// PageList.IncreaseCapacityError!void. The error is ErrOutOfMemory
// (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) AppendGrapheme(cell *PageCell, cp uint32) error {
	// Zig's defer evaluates the expression when it runs; the cursor page
	// may have changed by then.
	defer func() { self.Cursor.PagePin.Node.data.AssertIntegrity() }()
	if err := self.Cursor.PagePin.Node.data.AppendGrapheme(
		self.Cursor.PageRow,
		cell,
		cp,
	); err != nil {
		switch err {
		case ErrOutOfMemory:
			// We need to determine the actual cell index of the cell so
			// that after we adjust the capacity we can reload the cell.
			var cell_idx uint
			{
				// Zig: pointer arithmetic from the cursor cell back to the
				// row's first cell, then `(@intFromPtr(target) -
				// @intFromPtr(zero)) / @sizeOf(Cell)`, which traps when the
				// cell is before the row's first cell.
				page := &self.Cursor.PagePin.Node.data
				zero := page.cellIndex(self.Cursor.PageCell) - uint(self.Cursor.X)
				target := page.cellIndex(cell)
				assert(target >= zero) // Zig: - traps
				cell_idx = target - zero
			}

			// Adjust our capacity. This will update our cursor page pin and
			// force us to reload.
			if _, err := self.IncreaseCapacity(
				self.Cursor.PagePin.Node,
				Some(PageListIncreaseCapacityGraphemeBytes),
			); err != nil {
				return err
			}

			// The cell pointer is now invalid, so we need to get it from
			// the reloaded cursor pointers.
			var reloaded_cell *PageCell
			switch {
			case cell_idx == uint(self.Cursor.X):
				reloaded_cell = self.Cursor.PageCell
			case cell_idx < uint(self.Cursor.X):
				reloaded_cell = self.CursorCellLeft(SizeCellCountInt(uint(self.Cursor.X) - cell_idx))
			default:
				// Zig: @intCast to u16.
				assert(cell_idx-uint(self.Cursor.X) <= 0xFFFF)
				reloaded_cell = self.CursorCellRight(SizeCellCountInt(cell_idx - uint(self.Cursor.X)))
			}

			if err2 := self.Cursor.PagePin.Node.data.AppendGrapheme(
				self.Cursor.PageRow,
				reloaded_cell,
				cp,
			); err2 != nil {
				// Zig: comptime assert(@TypeOf(err2) == error{OutOfMemory});
				// This should never happen because we just increased capacity.
				// Log loudly but still return an error so we don't just
				// crash.
				warn("screen", "grapheme append failed after capacity increase") // Zig: log.err
				return err2
			}
		default:
			panic("unreachable")
		}
	}
	return nil
}

// StartHyperlink is Screen.startHyperlink.
//
// Start the hyperlink state. Future cells will be marked as hyperlinks with
// this state. Note that various terminal operations may clear the hyperlink
// state, such as switching screens (alt screen).
//
// Zig: pub fn startHyperlink(self: *Screen, uri: []const u8,
// id_: ?[]const u8) PageList.IncreaseCapacityError!void. The error is
// ErrOutOfMemory (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) StartHyperlink(uri []uint8, id_ Optional[[]uint8]) (err error) {
	// Create our pending entry.
	var link Hyperlink
	link.URI = uri
	if id_.Valid {
		link.ID = HyperlinkHyperlinkId{
			Tag:      HyperlinkHyperlinkIdExplicit,
			Explicit: id_.Value,
		}
	} else {
		// implicit:
		link.ID = HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: self.Cursor.HyperlinkImplicitID}
		// defer self.cursor.hyperlink_implicit_id += 1; (u32, traps on
		// overflow)
		assert(self.Cursor.HyperlinkImplicitID != 0xFFFFFFFF)
		self.Cursor.HyperlinkImplicitID += 1
	}
	defer func() {
		// errdefer
		if err != nil {
			switch link.ID.Tag {
			case HyperlinkHyperlinkIdExplicit:
			case HyperlinkHyperlinkIdImplicit:
				// Cannot underflow: it was incremented above.
				self.Cursor.HyperlinkImplicitID -= 1
			default:
				panic("unreachable")
			}
		}
	}()

	// Loop until we have enough page memory to add the hyperlink
	for {
		err := self.startHyperlinkOnce(link)
		if err == nil {
			return nil
		}
		switch err {
		// An actual self.alloc OOM is a fatal error.
		case ErrOutOfMemory:
			return ErrOutOfMemory

		// strings table is out of memory, adjust it up
		case ErrStringsOutOfMemory:
			if _, err := self.IncreaseCapacity(
				self.Cursor.PagePin.Node,
				Some(PageListIncreaseCapacityStringBytes),
			); err != nil {
				return err
			}

		// hyperlink set is out of memory, adjust it up
		case ErrSetOutOfMemory:
			if _, err := self.IncreaseCapacity(
				self.Cursor.PagePin.Node,
				Some(PageListIncreaseCapacityHyperlinkBytes),
			); err != nil {
				return err
			}

		// hyperlink set is too full, rehash it
		case ErrSetNeedsRehash:
			if _, err := self.IncreaseCapacity(
				self.Cursor.PagePin.Node,
				None[PageListIncreaseCapacity](),
			); err != nil {
				return err
			}

		default:
			panic("unreachable")
		}

		self.AssertIntegrity()
	}
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
func (self *Screen) startHyperlinkOnce(source Hyperlink) error {
	// End any prior hyperlink
	self.EndHyperlink()

	// Allocate our new Hyperlink entry in non-page memory. This
	// lets us quickly get access to URI, ID.
	link := new(Hyperlink) // self.alloc.create
	// Zig: errdefer self.alloc.destroy(link); Go memory.
	dup, err := source.Dupe()
	if err != nil {
		return err
	}
	*link = dup
	// Zig: errdefer link.deinit(self.alloc); it only frees.

	// Insert the hyperlink into page memory
	page := &self.Cursor.PagePin.Node.data
	id, err := page.InsertHyperlink(*link)
	if err != nil {
		return err
	}

	// Save it all
	self.Cursor.Hyperlink = link
	self.Cursor.HyperlinkID = id
	return nil
}

// EndHyperlink is Screen.endHyperlink.
//
// End the hyperlink state so that future cells aren't part of the
// current hyperlink (if any). This is safe to call multiple times.
func (self *Screen) EndHyperlink() {
	// If we have no hyperlink state then do nothing
	if self.Cursor.HyperlinkID == 0 {
		assert(self.Cursor.Hyperlink == nil)
		return
	}

	// Release the old hyperlink state. If there are cells using the
	// hyperlink this will work because the creation creates a reference
	// and all additional cells create a new reference. This release will
	// just release our initial reference.
	//
	// If the ref count reaches zero the set will not delete the item
	// immediately; it is kept around in case it is used again (this is
	// how RefCountedSet works). This causes some memory fragmentation but
	// is fine because if it is ever pruned the context deleted callback
	// will be called.
	page := &self.Cursor.PagePin.Node.data
	page.hyperlink_set.Release(self.Cursor.HyperlinkID)
	assert(self.Cursor.Hyperlink != nil) // .?
	self.Cursor.Hyperlink.Deinit()
	// Zig: self.alloc.destroy(self.cursor.hyperlink.?); Go memory.
	self.Cursor.HyperlinkID = 0
	self.Cursor.Hyperlink = nil
}

// CursorSetHyperlink is Screen.cursorSetHyperlink.
//
// Set the current hyperlink state on the current cell.
//
// Zig: PageList.IncreaseCapacityError!void. The error is ErrOutOfMemory
// (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) CursorSetHyperlink() error {
	assert(self.Cursor.HyperlinkID != 0)

	page := &self.Cursor.PagePin.Node.data
	err := page.SetHyperlink(
		self.Cursor.PageRow,
		self.Cursor.PageCell,
		self.Cursor.HyperlinkID,
	)
	if err == nil {
		// Success, increase the refcount for the hyperlink.
		page.hyperlink_set.Use(self.Cursor.HyperlinkID)
		return nil
	}
	switch err {
	// hyperlink_map is out of space, realloc the page to be larger
	case ErrHyperlinkMapOutOfMemory:
		// Attempt to allocate the space that would be required to
		// insert a new copy of the cursor hyperlink uri in to the
		// string alloc, since right now increaseCapacity always just
		// adds an extra copy even if one already exists in the page.
		// If this alloc fails then we know we also need to grow our
		// string bytes.
		//
		// FIXME: increaseCapacity should not do this.
		for self.Cursor.Hyperlink != nil {
			link := self.Cursor.Hyperlink
			if slice, err := page.string_alloc.Alloc(
				uint(len(link.URI)),
			); err == nil {
				// We don't bother freeing because we're
				// about to free the entire page anyway.
				_ = slice
				break
			}

			// We didn't have enough room, let's increase string bytes
			new_node, err := self.IncreaseCapacity(
				self.Cursor.PagePin.Node,
				Some(PageListIncreaseCapacityStringBytes),
			)
			if err != nil {
				return err
			}
			assert(new_node == self.Cursor.PagePin.Node)
			page = &new_node.data
		}

		if _, err := self.IncreaseCapacity(
			self.Cursor.PagePin.Node,
			Some(PageListIncreaseCapacityHyperlinkBytes),
		); err != nil {
			return err
		}

		// Retry
		//
		// We check that the cursor hyperlink hasn't been destroyed
		// by the capacity adjustment first though- since despite the
		// terrible code above, that can still apparently happen ._.
		if self.Cursor.HyperlinkID > 0 {
			return self.CursorSetHyperlink()
		}

	default:
		panic("unreachable")
	}
	return nil
}

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
func (self ScreenCursorSetSemanticContentTag) String() string {
	return screenCursorSetSemanticContentTagNames[self]
}

// screenCursorSetSemanticContentTagNames are the tag names of
// ScreenCursorSetSemanticContentTag, indexed by value (Zig's @tagName).
var screenCursorSetSemanticContentTagNames = [...]string{"prompt", "output", "input"}

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
func (self ScreenCursorSetSemanticContentInput) String() string {
	return screenCursorSetSemanticContentInputNames[self]
}

// screenCursorSetSemanticContentInputNames are the tag names of
// ScreenCursorSetSemanticContentInput, indexed by value (Zig's @tagName).
var screenCursorSetSemanticContentInputNames = [...]string{"clear_explicit", "clear_eol"}

// CursorSetSemanticContent is Screen.cursorSetSemanticContent.
//
// Modify the semantic content type of the cursor. This should
// be preferred over setting it manually since it handles all the
// proper accounting.
func (self *Screen) CursorSetSemanticContent(t ScreenCursorSetSemanticContent) {
	cursor := &self.Cursor

	switch t.Tag {
	case ScreenCursorSetSemanticContentTagOutput:
		cursor.SemanticContent = PageCellSemanticContentOutput
		cursor.SemanticContentClearEol = false

	case ScreenCursorSetSemanticContentTagInput:
		clear := t.Input
		cursor.SemanticContent = PageCellSemanticContentInput
		switch clear {
		case ScreenCursorSetSemanticContentInputClearExplicit:
			cursor.SemanticContentClearEol = false
		case ScreenCursorSetSemanticContentInputClearEol:
			cursor.SemanticContentClearEol = true
		default:
			panic("unreachable")
		}

	case ScreenCursorSetSemanticContentTagPrompt:
		kind := t.Prompt
		self.SemanticPrompt.Seen = true
		cursor.SemanticContent = PageCellSemanticContentPrompt
		cursor.SemanticContentClearEol = false
		switch kind {
		case OSCParsersSemanticPromptPromptKindInitial, OSCParsersSemanticPromptPromptKindRight:
			cursor.PageRow.semantic_prompt = PageRowSemanticPromptPrompt
		case OSCParsersSemanticPromptPromptKindContinuation, OSCParsersSemanticPromptPromptKindSecondary:
			cursor.PageRow.semantic_prompt = PageRowSemanticPromptPromptContinuation
		default:
			panic("unreachable")
		}

	default:
		panic("unreachable")
	}
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
func (self *Screen) Select(sel_ Optional[Selection]) error {
	if !sel_.Valid {
		self.ClearSelection()
		return nil
	}
	sel := sel_.Value

	// If this selection is untracked then we track it.
	var tracked_sel Selection
	if sel.Tracked() {
		tracked_sel = sel
	} else {
		var err error
		tracked_sel, err = sel.Track(self)
		if err != nil {
			return err
		}
	}
	// Zig: errdefer if (!sel.tracked()) tracked_sel.deinit(self): nothing
	// fails after it.

	// Untrack prior selection
	if self.Selection.Valid {
		old := &self.Selection.Value
		old.Deinit(self)
	}
	self.Selection = Some(tracked_sel)
	self.Dirty.Selection = true
	return nil
}

// ClearSelection is Screen.clearSelection.
//
// Same as select(null) but can't fail.
func (self *Screen) ClearSelection() {
	if self.Selection.Valid {
		sel := &self.Selection.Value
		sel.Deinit(self)
		self.Dirty.Selection = true
	}
	self.Selection = None[Selection]()
}

// ScreenSelectionString is Screen.SelectionString: the options of
// selectionString.
type ScreenSelectionString struct {
	// The selection to convert to a string.
	Sel Selection

	// If true, trim whitespace around the selection.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default must set it.
	Trim bool

	// If non-null, a stringmap will be written here. This will use
	// the same allocator as the call to selectionString. The string will
	// be duplicated here and in the return value so both must be freed.
	//
	// Zig: `map: ?*StringMap`, default null. `map` is a Go keyword.
	Map *StringMap
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
	// We'll use this as our buffer to build our string.
	aw := oscWriterAllocatingInit()
	defer aw.deinit()

	// Create a formatter and use that to emit our text.
	formatter := FormatterScreenFormatterInit(
		self,
		FormatterOptions{
			Emit:   FormatterFormatPlain,
			Unwrap: true,
			Trim:   opts.Trim,
			// Zig default: `.{}`, an empty non-null list.
			CodepointMap: Optional[[]FormatterCodepointMap]{Valid: true},
		},
	)
	formatter.content = FormatterScreenFormatterContent{
		Tag:       FormatterScreenFormatterContentSelection,
		Selection: Some(opts.Sel),
	}

	// If we have a string map, we need to set that up.
	var pins []PageListPin // std.ArrayList(Pin) = .empty
	// Zig: defer pins.deinit(alloc); Go memory.
	if opts.Map != nil {
		formatter.pin_map = Some(FormatterPinMap{
			map_: &pins,
		})
	}

	// Emit. Since this is an allocating writer, a failed write
	// just becomes an OOM.
	if err := formatter.format(&aw.writer); err != nil {
		return nil, ErrOutOfMemory
	}

	// Build our final text and if we have a string map set that up.
	text, err := aw.toOwnedSliceSentinel(0)
	if err != nil {
		return nil, err
	}
	// Zig: errdefer alloc.free(text); Go memory.
	if opts.Map != nil {
		map_ := opts.Map
		// alloc.dupeZ(u8, text)
		map_string := make([]uint8, len(text), len(text)+1)
		copy(map_string, text)
		map_string = append(map_string, 0)[:len(text)]
		// Zig: errdefer alloc.free(map_string); Go memory.
		// Zig: try selectionString_tw.check(.copy_map); omitted (file
		// header).
		// pins.toOwnedSlice(alloc): exactly the items.
		map_pins := make([]PageListPin, len(pins))
		copy(map_pins, pins)
		*map_ = StringMap{
			String: map_string,
			Map:    map_pins,
		}
	}

	return text, nil
}

// toOwnedSliceSentinel is std.Io.Writer.Allocating.toOwnedSliceSentinel
// (Zig 0.15.2), here because selectionString is its only user: the bytes
// written so far followed by the sentinel, handed to the caller, leaving the
// writer empty. The allocator parameter is dropped. Zig: Error![:0]u8; the
// error would be ErrOutOfMemory (dcs.go), which Go never returns.
//
// The result follows the package's `[:0]const u8` convention (StringMap.go):
// len is the text length and the sentinel is at index len, inside cap.
func (self *oscWriterAllocating) toOwnedSliceSentinel(sentinel uint8) ([]uint8, error) {
	w := &self.writer
	result := make([]uint8, w.end, w.end+1)
	copy(result, w.buffer[0:w.end])
	result = append(result, sentinel)[:w.end]
	w.buffer = nil
	w.end = 0
	return result, nil
}

// ScreenSelectLine is Screen.SelectLine: the options of selectLine.
type ScreenSelectLine struct {
	// The pin of some part of the line to select.
	Pin PageListPin

	// These are the codepoints to consider whitespace to trim
	// from the ends of the selection.
	//
	// Zig: ?[]const u21, default `&.{ 0, ' ', '\t' }`, which is NOT the Go
	// zero value (null); a caller that relies on the default must write
	// `Some([]uint32{0, ' ', '\t'})`.
	Whitespace Optional[[]uint32]

	// If true, line selection will consider semantic prompt
	// state changing a boundary. State changing is ANY state
	// change.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default must set it.
	SemanticPromptBoundary bool
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
	_ = self

	// Get the current point semantic prompt state since that determines
	// boundary conditions too. This makes it so that line selection can
	// only happen within the same prompt state. For example, if you triple
	// click output, but the shell uses spaces to soft-wrap to the prompt
	// then the selection will stop prior to the prompt. See issue #1329.
	var semantic_prompt_state Optional[PageCellSemanticContent]
	if opts.SemanticPromptBoundary {
		rac := opts.Pin.RowAndCell()
		semantic_prompt_state = Some(rac.cell.semantic_content)
	}

	// The real start of the row is the first row in the soft-wrap.
	//
	// Zig: the labeled block `start_pin:`.
	start_pin := func() PageListPin {
		it := opts.Pin.RowIterator(PageListDirectionLeftUp, None[PageListPin]())
		it_prev_ := it.Next()  // skip self
		assert(it_prev_.Valid) // .?
		it_prev := it_prev_.Value

		// First, check the current row for semantic boundaries before the clicked position.
		if semantic_prompt_state.Valid {
			v := semantic_prompt_state.Value
			row := it_prev.RowAndCell().row
			cells := it_prev.Node.data.GetCells(row)
			// Scan backwards from clicked position to find where our content starts
			n := uint(opts.Pin.X) + 1
			for i := uint(0); i < n; i++ {
				x_rev := uint(opts.Pin.X) - i
				if cells[x_rev].semantic_content != v {
					copy_ := it_prev
					// Zig: @intCast to u16; x_rev < pin.x here.
					copy_.X = SizeCellCountInt(x_rev + 1)
					return copy_
				}
			}

			// No boundary found before clicked position on current row.
			// If row doesn't wrap from above, start is at column 0.
			// Otherwise, continue checking previous rows.
		}

		for p_ := it.Next(); p_.Valid; p_ = it.Next() {
			p := p_.Value
			row := p.RowAndCell().row

			if !row.wrap {
				copy_ := it_prev
				copy_.X = 0
				return copy_
			}

			if semantic_prompt_state.Valid {
				v := semantic_prompt_state.Value
				// We need to check every cell in this row in reverse
				// order since we're going up and back.
				cells := p.Node.data.GetCells(row)
				for x := 0; x < len(cells); x++ {
					x_rev := len(cells) - 1 - x
					cell := cells[x_rev]
					if cell.semantic_content != v {
						return it_prev
					}
					it_prev = p
					it_prev.X = SizeCellCountInt(x_rev)
				}

				continue
			}

			it_prev = p
		}
		// Zig: the `else` of the while loop.
		copy_ := it_prev
		copy_.X = 0
		return copy_
	}()

	// The real end of the row is the final row in the soft-wrap.
	//
	// Zig: the labeled block `end_pin:`; its `return null` returns from
	// selectLine.
	end_pin_ := func() Optional[PageListPin] {
		it := opts.Pin.RowIterator(PageListDirectionRightDown, None[PageListPin]())
		for p_ := it.Next(); p_.Valid; p_ = it.Next() {
			p := p_.Value
			row := p.RowAndCell().row

			if semantic_prompt_state.Valid {
				v := semantic_prompt_state.Value
				// We need to check every cell in this row
				cells := p.Node.data.GetCells(row)

				// If this is our pin row we can start from our x because
				// the start_pin logic already found the real start.
				var start_offset uint = 0
				if p.Node == opts.Pin.Node &&
					p.Y == opts.Pin.Y {
					start_offset = uint(opts.Pin.X)
				}

				// Handle the zero case specially because if the first
				// col doesn't match then we end at the end of the prior
				// row. But if this is the first row, we can't go back,
				// so we scan forward to find where our content ends.
				if start_offset == 0 && cells[0].semantic_content != v {
					prev_ := p.Up(1)
					assert(prev_.Valid) // .?
					prev := prev_.Value
					prev.X = p.Node.data.size.cols - 1
					return Some(prev)
				}

				// For every other case, we end at the prior cell.
				for x := start_offset; x < uint(len(cells)); x++ {
					cell := cells[x]
					if cell.semantic_content != v {
						copy_ := p
						assert(x >= 1) // Zig: - traps
						copy_.X = SizeCellCountInt(x - 1)
						return Some(copy_)
					}
				}
			}

			if !row.wrap {
				copy_ := p
				copy_.X = p.Node.data.size.cols - 1
				return Some(copy_)
			}
		}

		return None[PageListPin]()
	}()
	if !end_pin_.Valid {
		return None[Selection]()
	}
	end_pin := end_pin_.Value

	// Go forward from the start to find the first non-whitespace character.
	//
	// Zig: the labeled block `start:`.
	var start PageListPin
	if !opts.Whitespace.Valid {
		start = start_pin
	} else {
		whitespace := opts.Whitespace.Value
		found := false
		it := start_pin.CellIterator(PageListDirectionRightDown, Some(end_pin))
		for p_ := it.Next(); p_.Valid; p_ = it.Next() {
			p := p_.Value
			cell := p.RowAndCell().cell
			if !cell.HasText() {
				continue
			}

			// Non-empty means we found it.
			this_whitespace := stdMemIndexOfAny(
				whitespace,
				[]uint32{cell.content.codepoint},
			).Valid
			if this_whitespace {
				continue
			}

			start = p
			found = true
			break
		}

		if !found {
			return None[Selection]()
		}
	}

	// Go backward from the end to find the first non-whitespace character.
	//
	// Zig: the labeled block `end:`.
	var end PageListPin
	if !opts.Whitespace.Valid {
		end = end_pin
	} else {
		whitespace := opts.Whitespace.Value
		found := false
		it := end_pin.CellIterator(PageListDirectionLeftUp, Some(start_pin))
		for p_ := it.Next(); p_.Valid; p_ = it.Next() {
			p := p_.Value
			cell := p.RowAndCell().cell
			if !cell.HasText() {
				continue
			}

			// Non-empty means we found it.
			this_whitespace := stdMemIndexOfAny(
				whitespace,
				[]uint32{cell.content.codepoint},
			).Valid
			if this_whitespace {
				continue
			}

			end = p
			found = true
			break
		}

		if !found {
			return None[Selection]()
		}
	}

	return Some(SelectionInit(start, end, false))
}

// SelectAll is Screen.selectAll.
//
// Return the selection for all contents on the screen. Surrounding
// whitespace is omitted. If there is no selection, this returns null.
//
// Zig: returns ?Selection.
func (self *Screen) SelectAll() Optional[Selection] {
	whitespace := []uint32{0, ' ', '\t'}

	// Zig: the labeled block `start:`.
	var start PageListPin
	{
		found := false
		it := self.Pages.CellIterator(
			PageListDirectionRightDown,
			Point{Tag: PointTagScreen},
			None[Point](),
		)
		for p_ := it.Next(); p_.Valid; p_ = it.Next() {
			p := p_.Value
			cell := p.RowAndCell().cell
			if !cell.HasText() {
				continue
			}

			// Non-empty means we found it.
			this_whitespace := stdMemIndexOfAny(
				whitespace,
				[]uint32{cell.content.codepoint},
			).Valid
			if this_whitespace {
				continue
			}

			start = p
			found = true
			break
		}

		if !found {
			return None[Selection]()
		}
	}

	// Zig: the labeled block `end:`.
	var end PageListPin
	{
		found := false
		it := self.Pages.CellIterator(
			PageListDirectionLeftUp,
			Point{Tag: PointTagScreen},
			None[Point](),
		)
		for p_ := it.Next(); p_.Valid; p_ = it.Next() {
			p := p_.Value
			cell := p.RowAndCell().cell
			if !cell.HasText() {
				continue
			}

			// Non-empty means we found it.
			this_whitespace := stdMemIndexOfAny(
				whitespace,
				[]uint32{cell.content.codepoint},
			).Valid
			if this_whitespace {
				continue
			}

			end = p
			found = true
			break
		}

		if !found {
			return None[Selection]()
		}
	}

	return Some(SelectionInit(start, end, false))
}

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
	var dir PageListDirection
	if start.Before(end) {
		dir = PageListDirectionRightDown
	} else {
		dir = PageListDirectionLeftUp
	}
	it := start.CellIterator(dir, Some(end))
	for pin_ := it.Next(); pin_.Valid; pin_ = it.Next() {
		pin := pin_.Value
		// Boundary conditions
		switch dir {
		case PageListDirectionRightDown:
			if end.Before(pin) {
				return None[Selection]()
			}
		case PageListDirectionLeftUp:
			if pin.Before(end) {
				return None[Selection]()
			}
		default:
			panic("unreachable")
		}

		// If we found a word, then return it
		if sel := self.SelectWord(pin, boundary_codepoints); sel.Valid {
			return sel
		}
	}

	return None[Selection]()
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
	_ = self

	// If our cell is empty we can't select a word, because we can't select
	// areas where the screen is not yet written.
	start_cell := pin.RowAndCell().cell
	if !start_cell.HasText() {
		return None[Selection]()
	}

	// Determine if we are a boundary or not to determine what our boundary is.
	expect_boundary := stdMemIndexOfAny(
		boundary_codepoints,
		[]uint32{start_cell.content.codepoint},
	).Valid

	// Go forwards to find our end boundary
	//
	// Zig: the labeled block `end:`.
	end := func() PageListPin {
		it := pin.CellIterator(PageListDirectionRightDown, None[PageListPin]())
		prev_ := it.Next()  // Consume one, our start
		assert(prev_.Valid) // .?
		prev := prev_.Value
		for p_ := it.Next(); p_.Valid; p_ = it.Next() {
			p := p_.Value
			rac := p.RowAndCell()
			cell := rac.cell

			// If we reached an empty cell its always a boundary
			if !cell.HasText() {
				return prev
			}

			// If we do not match our expected set, we hit a boundary
			this_boundary := stdMemIndexOfAny(
				boundary_codepoints,
				[]uint32{cell.content.codepoint},
			).Valid
			if this_boundary != expect_boundary {
				return prev
			}

			// If we are going to the next row and it isn't wrapped, we
			// return the previous.
			if p.X == p.Node.data.size.cols-1 && !rac.row.wrap {
				return p
			}

			prev = p
		}

		return prev
	}()

	// Go backwards to find our start boundary
	//
	// Zig: the labeled block `start:`.
	start := func() PageListPin {
		it := pin.CellIterator(PageListDirectionLeftUp, None[PageListPin]())
		prev_ := it.Next()  // Consume one, our start
		assert(prev_.Valid) // .?
		prev := prev_.Value
		for p_ := it.Next(); p_.Valid; p_ = it.Next() {
			p := p_.Value
			rac := p.RowAndCell()
			cell := rac.cell

			// If we are going to the next row and it isn't wrapped, we
			// return the previous.
			if p.X == p.Node.data.size.cols-1 && !rac.row.wrap {
				return prev
			}

			// If we reached an empty cell its always a boundary
			if !cell.HasText() {
				return prev
			}

			// If we do not match our expected set, we hit a boundary
			this_boundary := stdMemIndexOfAny(
				boundary_codepoints,
				[]uint32{cell.content.codepoint},
			).Valid
			if this_boundary != expect_boundary {
				return prev
			}

			prev = p
		}

		return prev
	}()

	return Some(SelectionInit(start, end, false))
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
func (self *Screen) SelectOutput(pin PageListPin) Optional[Selection] {
	// If our pin right now is not on output, then we return nothing.
	if pin.RowAndCell().cell.semantic_content != PageCellSemanticContentOutput {
		return None[Selection]()
	}

	// Get the post prior prompt from this pin. This is the prompt whose
	// output we'll be capturing.
	//
	// Zig: the labeled block `prompt:`, whose `return` statements return
	// from selectOutput: the closure reports them with `done`.
	prompt_pin, result, done := func() (PageListPin, Optional[Selection], bool) {
		// If we have a prompt above this point (including this point),
		// then thats the prompt we want to capture output from.
		it := pin.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
		if p := it.Next(); p.Valid {
			return p.Value, None[Selection](), false
		}

		// If we don't have a prompt, then we assume that we're
		// capturing all the output up to the next prompt.
		it = pin.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
		next_ := it.Next()
		if !next_.Valid {
			return PageListPin{}, None[Selection](), true
		}
		next := next_.Value

		// We'll capture from the start of the screen to just above
		// the prompt and will trim the trailing whitespace.
		start_pin := self.Pages.GetTopLeft(PointTagScreen)
		end_pin_ := next.Up(1)
		if !end_pin_.Valid {
			return PageListPin{}, None[Selection](), true
		}
		end_pin := end_pin_.Value
		end_pin.X = end_pin.Node.data.size.cols - 1
		cell_it := end_pin.CellIterator(PageListDirectionLeftUp, Some(start_pin))
		for p_ := cell_it.Next(); p_.Valid; p_ = cell_it.Next() {
			p := p_.Value
			cell := p.RowAndCell().cell
			end_pin = p
			if cell.HasText() {
				break
			}
		}

		return PageListPin{}, Some(SelectionInit(
			start_pin,
			end_pin,
			false,
		)), true
	}()
	if done {
		return result
	}

	// Grab our content
	hl_ := self.Pages.HighlightSemanticContent(
		prompt_pin,
		PageCellSemanticContentOutput,
	)
	if !hl_.Valid {
		return None[Selection]()
	}
	hl := hl_.Value

	// Trim our trailing whitespace
	cell_it := hl.End.CellIterator(PageListDirectionLeftUp, Some(hl.Start))
	for p_ := cell_it.Next(); p_.Valid; p_ = cell_it.Next() {
		p := p_.Value
		cell := p.RowAndCell().cell
		hl.End = p
		if cell.HasText() {
			break
		}
	}

	return Some(SelectionInit(hl.Start, hl.End, false))
}

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
func (self *ScreenLineIterator) Next() Optional[Selection] {
	if !self.current.Valid {
		return None[Selection]()
	}
	current := self.current.Value
	result_ := self.screen.SelectLine(ScreenSelectLine{
		Pin:                    current,
		Whitespace:             None[[]uint32](),
		SemanticPromptBoundary: false,
	})
	if !result_.Valid {
		self.current = None[PageListPin]()
		return None[Selection]()
	}
	result := result_.Value

	self.current = result.End().Down(1)
	return Some(result)
}

// LineIterator is Screen.lineIterator.
//
// Returns an iterator to move through the soft-wrapped lines starting
// from pin.
//
// Zig: pub fn lineIterator(self: *const Screen, start: Pin) LineIterator.
func (self *Screen) LineIterator(start PageListPin) ScreenLineIterator {
	return ScreenLineIterator{
		screen:  self,
		current: Some(start),
	}
}

// ScreenPromptClickMove is Screen.PromptClickMove.
type ScreenPromptClickMove struct {
	// Zig: usize. A count.
	Left uint
	// Zig: usize. A count.
	Right uint
}

// ScreenPromptClickMoveZero is Screen.PromptClickMove.zero.
var ScreenPromptClickMoveZero = ScreenPromptClickMove{
	Left:  0,
	Right: 0,
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
	// If we're not at an input cell with our cursor, no movement will
	// ever be possible.
	if self.Cursor.SemanticContent != PageCellSemanticContentInput &&
		self.Cursor.PageCell.semantic_content != PageCellSemanticContentInput {
		return ScreenPromptClickMoveZero
	}

	switch self.SemanticPrompt.Click.Tag {
	// None doesn't support movement and click_events must use a
	// different mechanism (SGR mouse events) that callers must handle.
	case ScreenSemanticPromptSemanticClickNone, ScreenSemanticPromptSemanticClickClickEvents:
		return ScreenPromptClickMoveZero
	case ScreenSemanticPromptSemanticClickCl:
		cl := self.SemanticPrompt.Click.Cl
		switch cl {
		// All of these currently use dumb line-based navigation.
		// But eventually we'll support more.
		case OSCParsersSemanticPromptClickLine,
			OSCParsersSemanticPromptClickMultiple,
			OSCParsersSemanticPromptClickConservativeVertical,
			OSCParsersSemanticPromptClickSmartVertical:
			return self.promptClickLine(click_pin)
		default:
			panic("unreachable")
		}
	default:
		panic("unreachable")
	}
}

// promptClickLine is Screen.promptClickLine.
//
// Determine the inputs required to move from the cursor to the given
// click location. If the cursor isn't currently at a prompt input
// location, this will return zero.
//
// This currently only supports moving a single line.
func (self *Screen) promptClickLine(click_pin PageListPin) ScreenPromptClickMove {
	// If our click pin is our cursor pin, no movement is needed.
	// Do this early so we can assume later that they are different.
	cursor_pin := *self.Cursor.PagePin
	if cursor_pin.Eql(click_pin) {
		return ScreenPromptClickMoveZero
	}

	// If our cursor is before our click, we're only emitting right inputs.
	if cursor_pin.Before(click_pin) {
		var count uint = 0

		// We go row-by-row because soft-wrapped rows are still a single
		// line to a shell, so we can't just look at our page row.
		row_it := cursor_pin.RowIterator(
			PageListDirectionRightDown,
			Some(click_pin),
		)
	right_row_it:
		for row_pin_ := row_it.Next(); row_pin_.Valid; row_pin_ = row_it.Next() {
			row_pin := row_pin_.Value
			rac := row_pin.RowAndCell()
			cells := row_pin.Node.data.GetCells(rac.row)

			// Determine if this row is our cursor.
			is_cursor_row := row_pin.Node == cursor_pin.Node &&
				row_pin.Y == cursor_pin.Y

			// If this is not the cursor row, verify it's still part of the
			// continuation of our starting prompt.
			if !is_cursor_row &&
				rac.row.semantic_prompt != PageRowSemanticPromptPromptContinuation {
				break
			}

			// Determine where our input starts.
			//
			// Zig: the labeled block `start_x:`.
			start_x := func() uint {
				// If this is our cursor row then we start after the cursor.
				if is_cursor_row {
					return uint(cursor_pin.X) + 1
				}

				// Otherwise, we start at the first input cell, because
				// we expect the shell to properly translate arrows across
				// lines to the start of the input. Some shells indent
				// where input starts on subsequent lines so we must do
				// this.
				for x, cell := range cells {
					if cell.semantic_content == PageCellSemanticContentInput {
						return uint(x)
					}
				}

				// We never found an input cell, so we need to move to the
				// next row.
				return uint(len(cells))
			}()

			// Iterate over the input cells and assume arrow keys only
			// jump to input cells.
			for x := start_x; x < uint(len(cells)); x++ {
				cell := cells[x]
				// Ignore non-input cells, but allow breaks. We assume
				// the shell will translate arrow keys to only input
				// areas.
				if cell.semantic_content != PageCellSemanticContentInput {
					continue
				}

				// Increment our input count
				count += 1

				// If this is our target, we're done.
				if row_pin.Node == click_pin.Node &&
					row_pin.Y == click_pin.Y &&
					x == uint(click_pin.X) {
					break right_row_it
				}
			}

			// If this row isn't soft-wrapped, we need to break out
			// because line based moving only handles single lines.
			// We're done!
			if !rac.row.wrap {
				// If we never found our pin, that means we clicked further
				// right/beyond it. If we're already on a non-empty input cell
				// then we add one so we can move to the newest, empty cell
				// at the end, matching typical editor behavior.
				if self.Cursor.PageCell.semantic_content == PageCellSemanticContentInput {
					count += 1
				}

				break
			}
		}

		return ScreenPromptClickMove{Left: 0, Right: count}
	}

	// Otherwise, cursor is after click, so we're emitting left inputs.
	var count uint = 0

	// We go row-by-row because soft-wrapped rows are still a single
	// line to a shell, so we can't just look at our page row.
	row_it := cursor_pin.RowIterator(
		PageListDirectionLeftUp,
		Some(click_pin),
	)
left_row_it:
	for row_pin_ := row_it.Next(); row_pin_.Valid; row_pin_ = row_it.Next() {
		row_pin := row_pin_.Value
		rac := row_pin.RowAndCell()
		cells := row_pin.Node.data.GetCells(rac.row)

		// Determine the length of the cells we look at in this row.
		//
		// Zig: the labeled block `end_len:`.
		var end_len uint
		if row_pin.Node == cursor_pin.Node &&
			row_pin.Y == cursor_pin.Y {
			// If this is our cursor row then we end before the cursor.
			end_len = uint(cursor_pin.X)
		} else {
			// Otherwise, we end at the last cell in the row.
			end_len = uint(len(cells))
		}

		// Iterate backwards over the input cells.
		for rev_x := uint(0); rev_x < end_len; rev_x++ {
			x := end_len - 1 - rev_x
			cell := cells[x]

			// Ignore non-input cells.
			if cell.semantic_content != PageCellSemanticContentInput {
				continue
			}

			// Increment our input count
			count += 1

			// If this is our target, we're done.
			if row_pin.Node == click_pin.Node &&
				row_pin.Y == click_pin.Y &&
				x == uint(click_pin.X) {
				break left_row_it
			}
		}

		// If this row is not a wrap continuation, then break out
		if !rac.row.wrap_continuation {
			break
		}
	}

	return ScreenPromptClickMove{Left: count, Right: 0}
}

// ScreenDumpStringOptions is NOT a named Zig declaration: it is the
// anonymous struct that is the `opts` parameter of Screen.dumpString.
type ScreenDumpStringOptions struct {
	// The start and end points of the dump, both inclusive. The x will
	// be ignored and the full row will always be dumped.
	Tl PageListPin
	// Zig: ?Pin, default null.
	Br Optional[PageListPin]

	// If true, this will unwrap soft-wrapped lines. If false, this will
	// dump the screen as it is visually seen in a rendered window.
	//
	// Zig default: TRUE. The Go zero value is false; a caller that relies
	// on the default must set it.
	Unwrap bool
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
func (self *Screen) dumpString(writer *oscWriter, opts ScreenDumpStringOptions) error {
	// Create a formatter and use that to emit our text.
	formatter := FormatterScreenFormatterInit(self, FormatterOptions{
		Emit:   FormatterFormatPlain,
		Unwrap: opts.Unwrap,
		Trim:   false,
		// Zig default: `.{}`, an empty non-null list.
		CodepointMap: Optional[[]FormatterCodepointMap]{Valid: true},
	})

	// Set up the selection based on the pins
	tl := opts.Tl
	var br PageListPin
	if opts.Br.Valid {
		br = opts.Br.Value
	} else {
		br_ := self.Pages.GetBottomRight(PointTagScreen)
		assert(br_.Valid) // .?
		br = br_.Value
	}

	formatter.content = FormatterScreenFormatterContent{
		Tag: FormatterScreenFormatterContentSelection,
		Selection: Some(SelectionInit(
			tl,
			br,
			false, // not rectangle
		)),
	}

	// Emit
	return formatter.format(writer)
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
func (self *Screen) DumpStringAlloc(tl Point) ([]uint8, error) {
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	tl_pin := self.Pages.GetTopLeft(tl.Tag)
	br := self.Pages.GetBottomRight(tl.Tag)
	if !br.Valid {
		return nil, ErrUnknownPoint
	}
	if err := self.dumpString(&builder.writer, ScreenDumpStringOptions{
		Tl:     tl_pin,
		Br:     Some(br.Value),
		Unwrap: false,
	}); err != nil {
		return nil, err
	}

	return builder.toOwnedSlice()
}

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
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	tl_pin := self.Pages.GetTopLeft(tl.Tag)
	br := self.Pages.GetBottomRight(tl.Tag)
	if !br.Valid {
		return nil, ErrUnknownPoint
	}
	if err := self.dumpString(&builder.writer, ScreenDumpStringOptions{
		Tl:     tl_pin,
		Br:     Some(br.Value),
		Unwrap: true,
	}); err != nil {
		return nil, err
	}

	return builder.toOwnedSlice()
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
func (self *Screen) TestWriteString(text []uint8) error {
	// Zig: std.unicode.Utf8View.init validates the whole text first (no
	// overlong forms, no surrogates, nothing above U+10FFFF), which is
	// what Go's utf8.Valid accepts; the iterator then decodes valid UTF-8.
	if !utf8.Valid(text) {
		return ErrInvalidUtf8
	}
	iter := text
	for len(iter) > 0 {
		r, size := utf8.DecodeRune(iter) // iter.nextCodepoint()
		iter = iter[size:]
		c := uint32(r)

		// Explicit newline forces a new row
		if c == '\n' {
			if err := self.cursorDownOrScroll(); err != nil {
				return err
			}
			self.CursorHorizontalAbsolute(0)
			self.Cursor.PendingWrap = false
			if self.Cursor.SemanticContentClearEol {
				self.CursorSetSemanticContent(ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagOutput})
			} else {
				switch self.Cursor.SemanticContent {
				case PageCellSemanticContentOutput:
				case PageCellSemanticContentPrompt, PageCellSemanticContentInput:
					self.Cursor.PageRow.semantic_prompt = PageRowSemanticPromptPromptContinuation
				default:
					panic("unreachable")
				}
			}
			continue
		}

		var width uint
		if c <= 0xFF {
			width = 1
		} else {
			// Zig: @intCast(unicode.table.get(c).width), a u2 to usize.
			width = uint(UnicodeTable.Get(c).width)
		}
		if width == 0 {
			// Zig: the labeled block `cell:`.
			cell := self.CursorCellLeft(1)
			switch cell.wide {
			case PageCellWideNarrow:
			case PageCellWideWide:
			case PageCellWideSpacerHead:
				panic("unreachable")
			case PageCellWideSpacerTail:
				cell = self.CursorCellLeft(2)
			default:
				panic("unreachable")
			}

			if err := self.Cursor.PagePin.Node.data.AppendGrapheme(
				self.Cursor.PageRow,
				cell,
				c,
			); err != nil {
				return err
			}
			continue
		}

		if self.Cursor.PendingWrap {
			assert(self.Cursor.X == self.Pages.cols-1)
			self.Cursor.PendingWrap = false
			self.Cursor.PageRow.wrap = true
			if err := self.cursorDownOrScroll(); err != nil {
				return err
			}
			self.CursorHorizontalAbsolute(0)
			self.Cursor.PageRow.wrap_continuation = true
			switch self.Cursor.SemanticContent {
			case PageCellSemanticContentOutput:
			case PageCellSemanticContentInput, PageCellSemanticContentPrompt:
				self.Cursor.PageRow.semantic_prompt = PageRowSemanticPromptPromptContinuation
			default:
				panic("unreachable")
			}
		}

		assert(width == 1 || width == 2)
		switch width {
		case 1:
			*self.Cursor.PageCell = PageCell{
				content_tag:      PageCellContentTagCodepoint,
				content:          PageCellContent{codepoint: c},
				style_id:         self.Cursor.StyleID,
				protected:        self.Cursor.Protected,
				semantic_content: self.Cursor.SemanticContent,
			}

			// If we have a ref-counted style, increase.
			if self.Cursor.StyleID != StyleDefaultID {
				// Zig: `const page = self.cursor.page_pin.node.data;`, a
				// copy of the Page struct; `use` only writes the item's
				// ref count, which the copy shares.
				page := self.Cursor.PagePin.Node.data
				page.styles.Use(self.Cursor.StyleID)
				self.Cursor.PageRow.styled = true
			}

			// If we have a hyperlink, add it to the cell.
			if self.Cursor.HyperlinkID > 0 {
				if err := self.CursorSetHyperlink(); err != nil {
					return err
				}
			}

		case 2:
			// Need a wide spacer head
			if self.Cursor.X == self.Pages.cols-1 {
				*self.Cursor.PageCell = PageCell{
					content_tag:      PageCellContentTagCodepoint,
					content:          PageCellContent{codepoint: 0},
					wide:             PageCellWideSpacerHead,
					protected:        self.Cursor.Protected,
					semantic_content: self.Cursor.SemanticContent,
				}

				// If we have a hyperlink, add it to the cell.
				if self.Cursor.HyperlinkID > 0 {
					if err := self.CursorSetHyperlink(); err != nil {
						return err
					}
				}

				self.Cursor.PageRow.wrap = true
				if err := self.cursorDownOrScroll(); err != nil {
					return err
				}
				self.CursorHorizontalAbsolute(0)
				self.Cursor.PageRow.wrap_continuation = true
			}

			// Write our wide char
			*self.Cursor.PageCell = PageCell{
				content_tag:      PageCellContentTagCodepoint,
				content:          PageCellContent{codepoint: c},
				style_id:         self.Cursor.StyleID,
				wide:             PageCellWideWide,
				protected:        self.Cursor.Protected,
				semantic_content: self.Cursor.SemanticContent,
			}

			// If we have a hyperlink, add it to the cell.
			if self.Cursor.HyperlinkID > 0 {
				if err := self.CursorSetHyperlink(); err != nil {
					return err
				}
			}

			// Write our tail
			self.CursorRight(1)
			*self.Cursor.PageCell = PageCell{
				content_tag:      PageCellContentTagCodepoint,
				content:          PageCellContent{codepoint: 0},
				wide:             PageCellWideSpacerTail,
				protected:        self.Cursor.Protected,
				semantic_content: self.Cursor.SemanticContent,
			}

			// If we have a hyperlink, add it to the cell.
			if self.Cursor.HyperlinkID > 0 {
				if err := self.CursorSetHyperlink(); err != nil {
					return err
				}
			}

			// If we have a ref-counted style, increase twice.
			if self.Cursor.StyleID != StyleDefaultID {
				// Zig: a copy of the Page struct, as above.
				page := self.Cursor.PagePin.Node.data
				page.styles.Use(self.Cursor.StyleID)
				page.styles.Use(self.Cursor.StyleID)
				self.Cursor.PageRow.styled = true
			}

		default:
			panic("unreachable")
		}

		if uint32(self.Cursor.X)+1 < uint32(self.Pages.cols) {
			self.CursorRight(1)
		} else {
			self.Cursor.PendingWrap = true
		}
	}
	return nil
}
