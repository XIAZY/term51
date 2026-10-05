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
// # What the implementation needs from files that are not ported yet
//
//   - `selectionString` (line 2448) and `dumpString` (line 3129) are
//     written against the formatter.go wireframe (`ScreenFormatter`); they
//     panic "not implemented" until formatter.go is ported;
//   - `testWriteString` (line 3207) reads the width of a codepoint above
//     0xFF from `unicode.table` (src/unicode), which has no Go port yet:
//     that one call site is `panic(notImplemented) // needs unicode`.
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
func (self *ScreenCursor) Deinit() {
	if self.hyperlink != nil {
		link := self.hyperlink
		link.Deinit()
		// Zig: alloc.destroy(link); Go memory.
	}
}

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
func ScreenInit(opts ScreenOptions) (_ Screen, err error) {
	// Initialize our backing pages.
	pages, err := PageListInit(
		opts.cols,
		opts.rows,
		Some(opts.max_scrollback),
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
	page_pin, err := pages.TrackPin(PageListPin{node: pages.pages.first})
	if err != nil {
		return Screen{}, err
	}
	// Zig: errdefer pages.untrackPin(page_pin): nothing fails after it.
	page_rac := page_pin.RowAndCell()

	result := Screen{
		pages:         pages,
		no_scrollback: opts.max_scrollback == 0,
		cursor: ScreenCursor{
			x:            0,
			y:            0,
			cursor_style: ScreenCursorStyleBlock, // Zig default
			page_pin:     page_pin,
			page_row:     page_rac.row,
			page_cell:    page_rac.cell,
		},
		charset:         ScreenCharsetState{gr: CharsetsSlotsG2}, // Zig default .{}
		semantic_prompt: ScreenSemanticPromptDisabled,
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
	self.cursor.Deinit()
	self.pages.Deinit()
}

// AssertIntegrity is Screen.assertIntegrity.
//
// Assert that the screen is in a consistent state. This doesn't check
// all pages in the page list because that is SO SLOW even just for
// tests. This only asserts the screen specific data so callers should
// ensure they're also calling page integrity checks if necessary.
//
// Zig: self is *const Screen.
func (self *Screen) AssertIntegrity() {
	if pageSlowRuntimeSafety {
		// We don't run integrity checks on Valgrind because its soooooo slow,
		// Valgrind is our integrity checker, and we run these during unit
		// tests (non-Valgrind) anyways so we're verifying anyways.
		//
		// Zig: `if (std.valgrind.runningOnValgrind() > 0) return;`; no Go
		// counterpart (file header).

		assert(self.cursor.x < self.pages.cols)
		assert(self.cursor.y < self.pages.rows)

		// Our cursor x/y should always match the pin. If this doesn't
		// match then it indicates that the tracked pin moved and we didn't
		// account for it by either calling cursorReload or manually
		// adjusting.
		pt_ := self.pages.PointFromPin(
			PointTagActive,
			*self.cursor.page_pin,
		)
		if !pt_.Valid {
			panic("unreachable") // orelse unreachable
		}
		pt := pt_.Value
		assert(self.cursor.x == pt.Active.x)
		assert(uint32(self.cursor.y) == pt.Active.y)
	}
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
	self.pages.Reset()

	// The above reset preserves tracked pins so we can still use
	// our cursor pin, which should be at the top-left already.
	cursor_pin := self.cursor.page_pin
	assert(self.pages.pages.first != nil) // .?
	assert(cursor_pin.node == self.pages.pages.first)
	assert(cursor_pin.x == 0)
	assert(cursor_pin.y == 0)
	cursor_rac := cursor_pin.RowAndCell()
	self.cursor.Deinit()
	self.cursor = ScreenCursor{
		cursor_style: ScreenCursorStyleBlock, // Zig default
		page_pin:     cursor_pin,
		page_row:     cursor_rac.row,
		page_cell:    cursor_rac.cell,
	}

	// Zig: `if (comptime build_options.kitty_graphics)` block (reset
	// kitty graphics storage): compiled out; see the file header.

	// Reset our basic state
	self.saved_cursor = None[ScreenSavedCursor]()
	self.charset = ScreenCharsetState{gr: CharsetsSlotsG2} // Zig: .{}
	self.kitty_keyboard = KittyKeyFlagStack{}
	self.protected_mode = ANSIProtectedModeOff
	self.semantic_prompt = ScreenSemanticPromptDisabled
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

	pages, err := self.pages.Clone(PageListClone{
		top:          top,
		bot:          bot,
		tracked_pins: &pin_remap,
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
		if p := pin_remap[self.cursor.page_pin]; p != nil {
			page_rac := p.RowAndCell()
			if pt := pages.PointFromPin(PointTagActive, *p); pt.Valid {
				// Zig: @intCast(pt.active.y) to u16.
				assert(pt.Value.Active.y <= 0xFFFF)
				return ScreenCursor{
					x:            pt.Value.Active.x,
					y:            SizeCellCountInt(pt.Value.Active.y),
					cursor_style: ScreenCursorStyleBlock, // Zig default
					page_pin:     p,
					page_row:     page_rac.row,
					page_cell:    page_rac.cell,
				}, nil
			}
			// break :remap
		}

		assert(pages.pages.first != nil) // .?
		page_pin, err := pages.TrackPin(PageListPin{node: pages.pages.first})
		if err != nil {
			return ScreenCursor{}, err
		}
		page_rac := page_pin.RowAndCell()
		return ScreenCursor{
			x:            0,
			y:            0,
			cursor_style: ScreenCursorStyleBlock, // Zig default
			page_pin:     page_pin,
			page_row:     page_rac.row,
			page_cell:    page_rac.cell,
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
		if !self.selection.Valid {
			return None[Selection](), nil
		}
		sel := self.selection.Value
		assert(sel.Tracked())

		var ordered struct {
			tl *PageListPin
			br *PageListPin
		}
		switch sel.Order(self) {
		case SelectionOrderForward, SelectionOrderMirroredForward:
			ordered.tl = sel.bounds.Tracked.start
			ordered.br = sel.bounds.Tracked.end
		case SelectionOrderReverse, SelectionOrderMirroredReverse:
			ordered.tl = sel.bounds.Tracked.end
			ordered.br = sel.bounds.Tracked.start
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
				clone_top_ := self.pages.Pin(top)
				if !clone_top_.Valid {
					return None[Selection](), nil
				}
				clone_top := clone_top_.Value
				clone_top_y_ := self.pages.PointFromPin(
					PointTagScreen,
					clone_top,
				)
				assert(clone_top_y_.Valid) // .?
				clone_top_y := clone_top_y_.Value.Screen.y
				br_pt := self.pages.PointFromPin(
					PointTagScreen,
					*ordered.br,
				)
				assert(br_pt.Valid) // .?
				if br_pt.Value.Screen.y < clone_top_y {
					return None[Selection](), nil
				}
				tl_pt := self.pages.PointFromPin(
					PointTagScreen,
					*ordered.tl,
				)
				assert(tl_pt.Valid) // .?
				if tl_pt.Value.Screen.y > clone_top_y {
					return None[Selection](), nil
				}
			}

			// We move the top pin back in bounds to the top row.
			assert(pages.pages.first != nil) // .?
			var x SizeCellCountInt = 0
			if sel.rectangle {
				x = ordered.tl.x
			}
			start_pin, err = pages.TrackPin(PageListPin{
				node: pages.pages.first,
				x:    x,
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
			if sel.rectangle {
				x = ordered.br.x
			} else {
				assert(pages.cols > 0) // Zig: - traps
				x = pages.cols - 1
			}
			assert(pages.pages.last.data.size.rows > 0) // Zig: - traps
			end_pin, err = pages.TrackPin(PageListPin{
				node: pages.pages.last,
				x:    x,
				y:    pages.pages.last.data.size.rows - 1,
			})
			if err != nil {
				return None[Selection](), err
			}
		}

		return Some(Selection{
			bounds: SelectionBounds{
				Tag: SelectionBoundsTagTracked,
				Tracked: SelectionBoundsTracked{
					start: start_pin,
					end:   end_pin,
				},
			},
			rectangle: sel.rectangle,
		}), nil
	}()
	if err != nil {
		return Screen{}, err
	}

	result := Screen{
		pages:           pages,
		no_scrollback:   self.no_scrollback,
		cursor:          cursor,
		selection:       sel,
		charset:         ScreenCharsetState{gr: CharsetsSlotsG2}, // Zig default .{}
		semantic_prompt: ScreenSemanticPromptDisabled,
		dirty:           self.dirty,
	}

	// term51 (docs/mapping.md, "Page-layer bookkeeping"; term51
	// docs/upstream-crashes.md and upstream-findings.md section 15, Screen
	// item 1): when the cursor was outside the cloned region and the clone
	// has more rows than the active area, the fallback cursor pin above is
	// not in the active area and upstream aborts in the assertIntegrity
	// below (`pointFromPin(...) orelse unreachable`). The port returns its
	// own error instead; the errdefer above deinits the clone.
	if !result.pages.PointFromPin(PointTagActive, *result.cursor.page_pin).Valid {
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
	if node != self.cursor.page_pin.node {
		return self.pages.IncreaseCapacity(
			node,
			adjustment,
		)
	}

	// We're modifying the cursor page. When we increase the
	// capacity below it will be short the ref count on our
	// current style and hyperlink, so we need to init those.
	new_node, err := self.pages.IncreaseCapacity(node, adjustment)
	if err != nil {
		return nil, err
	}
	new_page := &new_node.data

	// Re-add the style, if the page somehow doesn't have enough
	// memory to add it, we emit a warning and gracefully degrade
	// to the default style for the cursor.
	if self.cursor.style_id != StyleDefaultId {
		id, err := new_page.styles.Add(
			self.cursor.style,
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
			self.cursor.style = Style{}
			id = StyleDefaultId
		}
		self.cursor.style_id = id
	}

	// Re-add the hyperlink, if the page somehow doesn't have enough
	// memory to add it, we emit a warning and gracefully degrade to
	// no hyperlink.
	if self.cursor.hyperlink != nil {
		link := self.cursor.hyperlink

		// So we don't attempt to free any memory in the replaced page.
		self.cursor.hyperlink_id = 0
		self.cursor.hyperlink = nil

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
	assert(uint32(self.cursor.x)+uint32(n) < uint32(self.pages.cols))
	// Zig: `@as([*]Cell, @ptrCast(self.cursor.page_cell)) + n`.
	page := &self.cursor.page_pin.node.data
	return &page.cells[page.cellIndex(self.cursor.page_cell)+uint(n)]
}

// CursorCellLeft is Screen.cursorCellLeft.
//
// Zig: pub inline fn cursorCellLeft(self: *Screen, n: size.CellCountInt)
// *pagepkg.Cell. The result points into the cursor row's cells.
func (self *Screen) CursorCellLeft(n SizeCellCountInt) *PageCell {
	assert(self.cursor.x >= n)
	// Zig: `@as([*]Cell, @ptrCast(self.cursor.page_cell)) - n`.
	page := &self.cursor.page_pin.node.data
	return &page.cells[page.cellIndex(self.cursor.page_cell)-uint(n)]
}

// CursorCellEndOfPrev is Screen.cursorCellEndOfPrev.
func (self *Screen) CursorCellEndOfPrev() *PageCell {
	assert(self.cursor.y > 0)

	page_pin_ := self.cursor.page_pin.Up(1)
	assert(page_pin_.Valid) // .?
	page_pin := page_pin_.Value
	page_pin.x = self.pages.cols - 1
	page_rac := page_pin.RowAndCell()
	return page_rac.cell
}

// CursorRight is Screen.cursorRight.
//
// Move the cursor right. This is a specialized function that is very fast
// if the caller can guarantee we have space to move right (no wrapping).
func (self *Screen) CursorRight(n SizeCellCountInt) {
	// Zig: `self.cursor.x + n` is u16 and traps on overflow.
	assert(uint32(self.cursor.x)+uint32(n) < uint32(self.pages.cols))
	defer self.AssertIntegrity()

	// Zig: `cell + n` on the many-item pointer.
	page := &self.cursor.page_pin.node.data
	self.cursor.page_cell = &page.cells[page.cellIndex(self.cursor.page_cell)+uint(n)]
	self.cursor.page_pin.x += n
	self.cursor.x += n
}

// CursorLeft is Screen.cursorLeft.
//
// Move the cursor left.
func (self *Screen) CursorLeft(n SizeCellCountInt) {
	assert(self.cursor.x >= n)
	defer self.AssertIntegrity()

	// Zig: `cell - n` on the many-item pointer.
	page := &self.cursor.page_pin.node.data
	self.cursor.page_cell = &page.cells[page.cellIndex(self.cursor.page_cell)-uint(n)]
	// Zig: `-=` on u16 traps on underflow; the pin's x is the cursor's x
	// (assertIntegrity), so the assertion above covers it.
	self.cursor.page_pin.x -= n
	self.cursor.x -= n
}

// CursorUp is Screen.cursorUp.
//
// Move the cursor up.
//
// Precondition: The cursor is not at the top of the screen.
func (self *Screen) CursorUp(n SizeCellCountInt) {
	assert(self.cursor.y >= n)
	defer self.AssertIntegrity()

	self.cursor.y -= n // Must be set before cursorChangePin
	up := self.cursor.page_pin.Up(uint(n))
	assert(up.Valid) // .?
	self.cursorChangePin(up.Value)
	page_rac := self.cursor.page_pin.RowAndCell()
	self.cursor.page_row = page_rac.row
	self.cursor.page_cell = page_rac.cell
}

// CursorRowUp is Screen.cursorRowUp.
//
// Zig: pub fn cursorRowUp(self: *Screen, n: size.CellCountInt)
// *pagepkg.Row.
func (self *Screen) CursorRowUp(n SizeCellCountInt) *PageRow {
	assert(self.cursor.y >= n)
	defer self.AssertIntegrity()

	page_pin_ := self.cursor.page_pin.Up(uint(n))
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
	assert(uint32(self.cursor.y)+uint32(n) < uint32(self.pages.rows))
	defer self.AssertIntegrity()

	self.cursor.y += n // Must be set before cursorChangePin

	// We move the offset into our page list to the next row and then
	// get the pointers to the row/cell and set all the cursor state up.
	down := self.cursor.page_pin.Down(uint(n))
	assert(down.Valid) // .?
	self.cursorChangePin(down.Value)
	page_rac := self.cursor.page_pin.RowAndCell()
	self.cursor.page_row = page_rac.row
	self.cursor.page_cell = page_rac.cell
}

// CursorHorizontalAbsolute is Screen.cursorHorizontalAbsolute.
//
// Move the cursor to some absolute horizontal position.
func (self *Screen) CursorHorizontalAbsolute(x SizeCellCountInt) {
	assert(x < self.pages.cols)
	defer self.AssertIntegrity()

	self.cursor.page_pin.x = x
	page_rac := self.cursor.page_pin.RowAndCell()
	self.cursor.page_cell = page_rac.cell
	self.cursor.x = x
}

// CursorAbsolute is Screen.cursorAbsolute.
//
// Move the cursor to some absolute position.
func (self *Screen) CursorAbsolute(x SizeCellCountInt, y SizeCellCountInt) {
	assert(x < self.pages.cols)
	assert(y < self.pages.rows)
	defer self.AssertIntegrity()

	var page_pin PageListPin
	if y < self.cursor.y {
		up := self.cursor.page_pin.Up(uint(self.cursor.y - y))
		assert(up.Valid) // .?
		page_pin = up.Value
	} else if y > self.cursor.y {
		down := self.cursor.page_pin.Down(uint(y - self.cursor.y))
		assert(down.Valid) // .?
		page_pin = down.Value
	} else {
		page_pin = *self.cursor.page_pin
	}
	page_pin.x = x
	self.cursor.x = x // Must be set before cursorChangePin
	self.cursor.y = y
	self.cursorChangePin(page_pin)
	page_rac := self.cursor.page_pin.RowAndCell()
	self.cursor.page_row = page_rac.row
	self.cursor.page_cell = page_rac.cell
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
	pt_ := self.pages.PointFromPin(
		PointTagActive,
		*self.cursor.page_pin,
	)
	if !pt_.Valid {
		// reset:
		pin_ := self.pages.Pin(Point{Tag: PointTagActive})
		assert(pin_.Valid) // .?
		pin := pin_.Value
		*self.cursor.page_pin = pin
		pt_ = self.pages.PointFromPin(PointTagActive, pin)
		assert(pt_.Valid) // .?
	}
	pt := pt_.Value

	self.cursor.x = pt.Active.x
	// Zig: @intCast(pt.active.y) to u16.
	assert(pt.Active.y <= 0xFFFF)
	self.cursor.y = SizeCellCountInt(pt.Active.y)
	page_rac := self.cursor.page_pin.RowAndCell()
	self.cursor.page_row = page_rac.row
	self.cursor.page_cell = page_rac.cell

	// If we have a style, we need to ensure it is in the page because this
	// method may also be called after a page change.
	if self.cursor.style_id != StyleDefaultId {
		if err := self.ManualStyleUpdate(); err != nil {
			// This failure should not happen because manualStyleUpdate
			// handles page splitting, overflow, and more. This should only
			// happen if we're out of RAM. In this case, we'll just degrade
			// gracefully back to the default style.
			warn("screen", "failed to update style on cursor reload err=error.%v", err) // Zig: log.err
			self.cursor.style = Style{}
			self.cursor.style_id = 0
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
	assert(self.cursor.y == self.pages.rows-1)
	defer self.AssertIntegrity()

	// Zig: `if (comptime build_options.kitty_graphics)` block (scrolling
	// dirties the images): compiled out; see the file header.

	// If we have no scrollback, then we shift all our rows instead.
	if self.no_scrollback {
		// If we have a single-row screen, we have no rows to shift
		// so our cursor is in the correct place we just have to clear
		// the cells.
		if self.pages.rows == 1 {
			page := &self.cursor.page_pin.node.data
			self.ClearCells(
				page,
				self.cursor.page_row,
				page.GetCells(self.cursor.page_row),
			)
			self.CursorMarkDirty()
		} else {
			// The call to `eraseRow` will move the tracked cursor pin up by one
			// row, but we don't actually want that, so we keep the old pin and
			// put it back after calling `eraseRow`.
			old_pin := *self.cursor.page_pin

			// eraseRow will shift everything below it up.
			if err := self.pages.EraseRow(Point{Tag: PointTagActive}); err != nil {
				return err
			}

			// Note we don't need to mark anything dirty in this branch
			// because eraseRow will mark all the rotated rows as dirty
			// in the entire page.

			// We don't use `cursorChangePin` here because we aren't
			// actually changing the pin, we're keeping it the same.
			*self.cursor.page_pin = old_pin

			// We do, however, need to refresh the cached page row
			// and cell, because `eraseRow` will have moved the row.
			page_rac := self.cursor.page_pin.RowAndCell()
			self.cursor.page_row = page_rac.row
			self.cursor.page_cell = page_rac.cell
		}
	} else {
		old_pin := *self.cursor.page_pin

		// Grow our pages by one row. The PageList will handle if we need to
		// allocate, prune scrollback, whatever.
		if _, err := self.pages.Grow(); err != nil {
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
			if old_pin.node == self.cursor.page_pin.node {
				down := self.cursor.page_pin.Down(1)
				assert(down.Valid) // .?
				page_pin = down.Value
			} else {
				// reuse:
				pin := *self.cursor.page_pin
				pin.x = self.cursor.x
				page_pin = pin
			}

			// These assertions help catch some pagelist math errors. Our
			// x/y should be unchanged after the grow.
			if pageSlowRuntimeSafety {
				active_ := self.pages.PointFromPin(
					PointTagActive,
					page_pin,
				)
				assert(active_.Valid) // .?
				active := active_.Value.Active
				assert(active.x == self.cursor.x)
				assert(active.y == uint32(self.cursor.y))
			}

			new_pin = page_pin
		}
		self.cursorChangePin(new_pin)
		page_rac := self.cursor.page_pin.RowAndCell()
		self.cursor.page_row = page_rac.row
		self.cursor.page_cell = page_rac.cell

		// Our new row is always dirty
		self.CursorMarkDirty()

		// Clear the new row so it gets our bg color. We only do this
		// if we have a bg color at all.
		if self.cursor.style.bg_color.Tag != StyleColorNone {
			page := &self.cursor.page_pin.node.data
			self.ClearCells(
				page,
				self.cursor.page_row,
				page.GetCells(self.cursor.page_row),
			)
		}
	}

	if self.cursor.style_id != StyleDefaultId {
		// The newly created line needs to be styled according to
		// the bg color if it is set.
		if blank_cell := self.cursor.style.BgCell(); blank_cell.Valid {
			// Zig: `cell_current - self.cursor.x` on the many-item
			// pointer, then `@memset(cells[0..self.pages.cols], blank_cell)`.
			page := &self.cursor.page_pin.node.data
			start := page.cellIndex(self.cursor.page_cell) - uint(self.cursor.x)
			cells := page.cells[start : start+uint(self.pages.cols)]
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
	if self.cursor.y == self.pages.rows-1 {
		return self.CursorDownScroll()
	}

	defer self.AssertIntegrity()

	// Logic below assumes we always have at least one row that isn't moving
	assert(self.cursor.y < self.pages.rows-1)

	// Explanation:
	//  We don't actually move everything that's at or above the cursor row,
	//  since this would require us to shift up our ENTIRE scrollback, which
	//  would be ridiculously expensive. Instead, we insert a new row at the
	//  end of the pagelist (`grow()`), and move everything BELOW the cursor
	//  DOWN by one row. This has the same practical result but it's a whole
	//  lot cheaper in 99% of cases.

	old_pin := *self.cursor.page_pin
	grown, err := self.pages.Grow()
	if err != nil {
		return err
	}
	if grown != nil {
		if err := self.cursorScrollAboveRotate(); err != nil {
			return err
		}
	} else {
		// In this case, it means grow() didn't allocate a new page.

		if self.cursor.page_pin.node == self.pages.pages.last {
			// If we're on the last page we can do a very fast path because
			// all the rows we need to move around are within a single page.

			// Note: we don't need to call cursorChangePin here because
			// the pin page is the same so there is no accounting to do
			// for styles or any of that.
			assert(old_pin.node == self.cursor.page_pin.node)
			down := self.cursor.page_pin.Down(1)
			assert(down.Valid) // .?
			*self.cursor.page_pin = down.Value

			pin := self.cursor.page_pin
			page := &self.cursor.page_pin.node.data

			// Rotate the rows so that the newly created empty row is at the
			// beginning. e.g. [ 0 1 2 3 ] in to [ 3 0 1 2 ].
			rows := page.rows
			screenRotateOnceR(rows[pin.y:page.size.rows])

			// Mark the whole page as dirty.
			//
			// Technically we only need to mark from the cursor row to the
			// end but this is a hot function, so we want to minimize work.
			page.dirty = true

			// Setup our cursor caches after the rotation so it points to the
			// correct data
			page_rac := self.cursor.page_pin.RowAndCell()
			self.cursor.page_row = page_rac.row
			self.cursor.page_cell = page_rac.cell
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

	if self.cursor.style_id != StyleDefaultId {
		// The newly created line needs to be styled according to
		// the bg color if it is set.
		if blank_cell := self.cursor.style.BgCell(); blank_cell.Valid {
			// Zig: `cell_current - self.cursor.x` on the many-item
			// pointer, then `@memset(cells[0..self.pages.cols], blank_cell)`.
			page := &self.cursor.page_pin.node.data
			start := page.cellIndex(self.cursor.page_cell) - uint(self.cursor.x)
			cells := page.cells[start : start+uint(self.pages.cols)]
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
	down := self.cursor.page_pin.Down(1)
	assert(down.Valid) // .?
	self.cursorChangePin(down.Value)

	// Go through each of the pages following our pin, shift all rows
	// down by one, and copy the last row of the previous page.
	current := self.pages.pages.last
	assert(current != nil) // .?
	for current != self.cursor.page_pin.node {
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
	assert(current == self.cursor.page_pin.node)
	cur_page := &current.data
	cur_rows := cur_page.rows
	screenRotateOnceR(cur_rows[self.cursor.page_pin.y:cur_page.size.rows])
	self.ClearCells(
		cur_page,
		&cur_rows[self.cursor.page_pin.y],
		cur_page.GetCells(&cur_rows[self.cursor.page_pin.y]),
	)

	// Mark the whole page as dirty.
	//
	// Technically we only need to mark from the cursor row to the
	// end but this is a hot function, so we want to minimize work.
	cur_page.dirty = true

	// Setup cursor cache data after all the rotations so our
	// row is valid.
	page_rac := self.cursor.page_pin.RowAndCell()
	self.cursor.page_row = page_rac.row
	self.cursor.page_cell = page_rac.cell

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
	if uint32(self.cursor.y)+1 < uint32(self.pages.rows) {
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
func (self *Screen) CursorCopy(other ScreenCursor, opts ScreenCursorCopyOptions) (err error) {
	assert(other.x < self.pages.cols)
	assert(other.y < self.pages.rows)

	// End any currently active hyperlink on our cursor.
	self.EndHyperlink()

	old := self.cursor
	self.cursor = other
	defer func() {
		// errdefer self.cursor = old;
		if err != nil {
			self.cursor = old
		}
	}()

	// Keep our old style ID so it can be properly cleaned up below.
	self.cursor.style_id = old.style_id

	// Hyperlinks will be managed separately below.
	self.cursor.hyperlink_id = 0
	self.cursor.hyperlink = nil

	// Keep our old page pin and X/Y because:
	// 1. The old style will need to be cleaned up from the page it's from.
	// 2. The new position navigated to by `cursorAbsolute` needs to be in our
	//    own screen.
	self.cursor.page_pin = old.page_pin
	self.cursor.x = old.x
	self.cursor.y = old.y

	// Call manual style update in order to clean up our old style, if we have
	// one, and also to load the style from the other cursor, if it had one.
	if err := self.ManualStyleUpdate(); err != nil {
		return err
	}

	// Move to the correct location to match the other cursor.
	self.CursorAbsolute(other.x, other.y)

	// If the other cursor had a hyperlink, add it to ours.
	if opts.hyperlink && other.hyperlink_id != 0 {
		// Get the hyperlink from the other cursor's page.
		other_page := &other.page_pin.node.data
		other_link := other_page.hyperlink_set.Get(other.hyperlink_id)

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
	if !self.cursor.page_pin.Eql(new) {
		self.CursorMarkDirty()
		new.MarkDirty()
	}

	// If our pin is on the same page, then we can just update the pin.
	// We don't need to migrate any state.
	if self.cursor.page_pin.node == new.node {
		*self.cursor.page_pin = new
		return
	}

	// If we have an old style then we need to release it from the old page.
	var old_style_ Optional[Style]
	if self.cursor.style_id == StyleDefaultId {
		old_style_ = None[Style]()
	} else {
		old_style_ = Some(self.cursor.style)
	}
	if old_style_.Valid {
		// Release the style directly from the old page instead of going through
		// manualStyleUpdate, because the cursor position may have already been
		// updated but the pin has not, which would fail integrity checks.
		old_page := &self.cursor.page_pin.node.data
		old_page.styles.Release(self.cursor.style_id)
		self.cursor.style = Style{}
		self.cursor.style_id = StyleDefaultId
	}

	// If we have a hyperlink then we need to release it from the old page.
	if self.cursor.hyperlink != nil {
		old_page := &self.cursor.page_pin.node.data
		old_page.hyperlink_set.Release(self.cursor.hyperlink_id)
	}

	// Update our pin to the new page
	*self.cursor.page_pin = new

	// On the new page, we need to migrate our style
	if old_style_.Valid {
		old_style := old_style_.Value
		self.cursor.style = old_style
		if err := self.ManualStyleUpdate(); err != nil {
			// This failure should not happen because manualStyleUpdate
			// handles page splitting, overflow, and more. This should only
			// happen if we're out of RAM. In this case, we'll just degrade
			// gracefully back to the default style.
			warn("screen", "failed to update style on cursor change err=error.%v", err) // Zig: log.err
			self.cursor.style = Style{}
			self.cursor.style_id = 0
		}
	}

	// On the new page, we need to migrate our hyperlink
	if self.cursor.hyperlink != nil {
		link := self.cursor.hyperlink

		// So we don't attempt to free any memory in the replaced page.
		self.cursor.hyperlink_id = 0
		self.cursor.hyperlink = nil

		// Re-add
		var id_ Optional[[]uint8]
		switch link.id.Tag {
		case HyperlinkHyperlinkIdExplicit:
			id_ = Some(link.id.Explicit)
		case HyperlinkHyperlinkIdImplicit:
			id_ = None[[]uint8]()
		default:
			panic("unreachable")
		}
		if err := self.StartHyperlink(link.uri, id_); err != nil {
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
	self.cursor.page_row.dirty = true
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
	self.cursor.pending_wrap = false

	page_row := self.cursor.page_row

	if !page_row.wrap {
		return
	}

	// This row does not wrap and the next row is not wrapped to
	page_row.wrap = false

	if next_row := self.cursor.page_pin.Down(1); next_row.Valid {
		next_row.Value.RowAndCell().row.wrap_continuation = false
	}

	// If the last cell in the row is a spacer head we need to clear it.
	cells := self.cursor.page_pin.Cells(PageListPinCellSubsetAll)
	cell := cells[self.cursor.page_pin.node.data.size.cols-1]
	if cell.wide == PageCellWideSpacerHead {
		self.ClearCells(
			&self.cursor.page_pin.node.data,
			page_row,
			cells[self.cursor.page_pin.node.data.size.cols-1:][0:1],
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
		self.pages.Scroll(PageListScroll{Tag: PageListScrollActive})
	case ScreenScrollTop:
		self.pages.Scroll(PageListScroll{Tag: PageListScrollTop})
	case ScreenScrollPin:
		self.pages.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: behavior.Pin})
	case ScreenScrollRow:
		self.pages.Scroll(PageListScroll{Tag: PageListScrollRow, Row: behavior.Row})
	case ScreenScrollDeltaRow:
		self.pages.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: behavior.DeltaRow})
	case ScreenScrollDeltaPrompt:
		self.pages.Scroll(PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: behavior.DeltaPrompt})
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

	if err := self.pages.ScrollClear(); err != nil {
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
	return self.pages.viewport.Tag == PageListViewportActive
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
	self.pages.EraseRows(tl, bl)

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

	it := self.pages.PageIterator(PageListDirectionRightDown, tl, bl)
	for chunk_ := it.Next(); chunk_.Valid; chunk_ = it.Next() {
		chunk := chunk_.Value
		rows := chunk.Rows()
		for i := range rows {
			row := &rows[i]
			cells_offset := row.cells
			// Zig: `row.cells.ptr(chunk.node.data.memory)`, sliced.
			cells_multi := chunk.node.data.rowCells(row)
			cells := cells_multi[0:self.pages.cols]

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
		if uint(len(cells)) == uint(self.pages.cols) {
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
		if uint(len(cells)) == uint(self.pages.cols) {
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
		if uint(len(cells)) == uint(self.pages.cols) {
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
	page := &self.cursor.page_pin.node.data

	page.PauseIntegrityChecks(true)
	defer page.PauseIntegrityChecks(false)

	cols := self.cursor.page_pin.node.data.size.cols

	// `x` may be up to an INCLUDING `cols`, since that signifies splitting
	// the boundary to the right of the final cell in the row.
	assert(x <= cols)

	// [ A B C D E F|]
	//              ^ Boundary between final cell and row end.
	if x == cols {
		if !self.cursor.page_row.wrap {
			return
		}

		cells := self.cursor.page_pin.Cells(PageListPinCellSubsetAll)

		// Spacer head at end of wrapped row.
		if cells[cols-1].wide == PageCellWideSpacerHead {
			self.ClearCells(
				page,
				self.cursor.page_row,
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
	if (x == 0 || x == 1) && self.cursor.page_row.wrap_continuation {
		cells := self.cursor.page_pin.Cells(PageListPinCellSubsetAll)

		// If the first cell in a row is wide the previous row
		// may have a spacer head which needs to be cleared.
		if cells[0].wide == PageCellWideWide {
			if p_row_ := self.cursor.page_pin.Up(1); p_row_.Valid {
				p_row := p_row_.Value
				p_rac := p_row.RowAndCell()
				p_cells := p_row.Cells(PageListPinCellSubsetAll)
				p_cell := p_cells[p_row.node.data.size.cols-1]
				if p_cell.wide == PageCellWideSpacerHead {
					self.ClearCells(
						&p_row.node.data,
						p_rac.row,
						p_cells[p_row.node.data.size.cols-1:][0:1],
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

		cells := self.cursor.page_pin.Cells(PageListPinCellSubsetAll)

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
				self.cursor.page_row,
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
	if self.cursor.style_id == StyleDefaultId {
		return PageCell{}
	}
	if bg := self.cursor.style.BgCell(); bg.Valid {
		return bg.Value
	}
	return PageCell{}
}

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
func (self *Screen) Resize(opts ScreenResize) (err error) {
	defer self.AssertIntegrity()

	// Zig: `if (comptime build_options.kitty_graphics)` block (mark the
	// image state dirty): compiled out; see the file header.

	// Release the cursor style while resizing just
	// in case the cursor ends up on a different page.
	cursor_style := self.cursor.style
	self.cursor.style = Style{}
	if err := self.ManualStyleUpdate(); err != nil {
		panic("unreachable") // catch unreachable
	}
	defer func() {
		// Restore the cursor style.
		self.cursor.style = cursor_style
		if err := self.ManualStyleUpdate(); err != nil {
			// This failure should not happen because manualStyleUpdate
			// handles page splitting, overflow, and more. This should only
			// happen if we're out of RAM. In this case, we'll just degrade
			// gracefully back to the default style.
			warn("screen", "failed to update style on cursor reload err=error.%v", err) // Zig: log.err
			self.cursor.style = Style{}
			self.cursor.style_id = 0
		}
	}()

	// If we have a hyperlink, release it from the old page
	// and then we need to re-add it to the new page. This needs
	// to happen because resize below typically reallocates a
	// new page so the old hyperlink is invalid.
	hyperlink_ := self.cursor.hyperlink
	if self.cursor.hyperlink_id != 0 {
		// Note we do NOT use endHyperlink because we want to keep
		// our allocated self.cursor.hyperlink valid.
		page := &self.cursor.page_pin.node.data
		page.hyperlink_set.Release(self.cursor.hyperlink_id)
		self.cursor.hyperlink_id = 0
		self.cursor.hyperlink = nil
	}

	// We need to insert a tracked pin for our saved cursor so we can
	// modify its X/Y for reflow.
	//
	// Zig: the labeled block `saved_cursor:`.
	var saved_cursor_pin *PageListPin
	if self.saved_cursor.Valid {
		sc := self.saved_cursor.Value
		pin := self.pages.Pin(Point{Tag: PointTagActive, Active: PointCoordinate{
			x: sc.x,
			y: uint32(sc.y),
		}})
		if pin.Valid {
			saved_cursor_pin, err = self.pages.TrackPin(pin.Value)
			if err != nil {
				return err
			}
		}
	}
	defer func() {
		if saved_cursor_pin != nil {
			self.pages.UntrackPin(saved_cursor_pin)
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
	if opts.prompt_redraw != OSCParsersSemanticPromptRedrawFalse &&
		self.cursor.semantic_content != PageCellSemanticContentOutput {
		func() {
			switch opts.prompt_redraw {
			case OSCParsersSemanticPromptRedrawFalse:
				panic("unreachable")

			// For `.last`, only clear the current line where the cursor is.
			// For `.true`, clear all prompt lines starting from the beginning.
			case OSCParsersSemanticPromptRedrawLast:
				page := &self.cursor.page_pin.node.data
				row := self.cursor.page_row
				cells := page.GetCells(row)
				self.ClearCells(page, row, cells)

			case OSCParsersSemanticPromptRedrawTrue:
				// start:
				var start PageListPin
				{
					it := self.cursor.page_pin.PromptIterator(
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
					page := &pin.node.data
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
	if err := self.pages.Resize(PageListResize{
		rows:   Some(opts.rows),
		cols:   Some(opts.cols),
		reflow: opts.reflow,
		cursor: Some(PageListResizeCursor{x: self.cursor.x, y: self.cursor.y}),
	}); err != nil {
		return err
	}

	// If we have no scrollback and we shrunk our rows, we must explicitly
	// erase our history. This is because PageList always keeps at least
	// a page size of history.
	if self.no_scrollback {
		self.pages.EraseRows(Point{Tag: PointTagHistory}, None[Point]())
	}

	// If our cursor was updated, we do a full reload so all our cursor
	// state is correct.
	self.CursorReload()

	// If we reflowed a saved cursor, update it.
	if saved_cursor_pin != nil {
		p := saved_cursor_pin
		// This should never fail because a non-null saved_cursor_pin
		// implies a non-null saved_cursor.
		assert(self.saved_cursor.Valid) // .?
		sc := &self.saved_cursor.Value
		if pt_ := self.pages.PointFromPin(PointTagActive, *p); pt_.Valid {
			pt := pt_.Value
			sc.x = pt.Active.x
			// Zig: @intCast(pt.active.y) to u16.
			assert(pt.Active.y <= 0xFFFF)
			sc.y = SizeCellCountInt(pt.Active.y)

			// If we had pending wrap set and we're no longer at the end of
			// the line, we unset the pending wrap and move the cursor to
			// reflect the correct next position.
			if sc.pending_wrap && sc.x != opts.cols-1 {
				sc.pending_wrap = false
				sc.x += 1
			}
		} else {
			// I think this can happen if the screen is resized to be
			// less rows or less cols and our saved cursor moves outside
			// the active area. In this case, there isn't anything really
			// reasonable we can do so we just move the cursor to the
			// top-left. It may be reasonable to also move the cursor to
			// match the primary cursor. Any behavior is fine since this is
			// totally unspecified.
			sc.x = 0
			sc.y = 0
			sc.pending_wrap = false
		}
	}

	// Fix up our hyperlink if we had one.
	if hyperlink_ != nil {
		link := hyperlink_
		var id_ Optional[[]uint8]
		switch link.id.Tag {
		case HyperlinkHyperlinkIdExplicit:
			id_ = Some(link.id.Explicit)
		case HyperlinkHyperlinkIdImplicit:
			id_ = None[[]uint8]()
		default:
			panic("unreachable")
		}
		if err := self.StartHyperlink(link.uri, id_); err != nil {
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
	old_style := self.cursor.style
	defer func() {
		// errdefer
		if err != nil {
			self.cursor.style = old_style
			if err := self.ManualStyleUpdate(); err != nil {
				warn("screen", "setAttribute error restoring old style after failure err=error.%v", err)
				self.cursor.style = Style{}
				if err := self.ManualStyleUpdate(); err != nil {
					panic("unreachable") // catch unreachable
				}
			}
		}
	}()

	switch attr.Tag {
	case SGRAttributeTagUnset:
		self.cursor.style = Style{}

	case SGRAttributeTagBold:
		self.cursor.style.flags.bold = true

	case SGRAttributeTagResetBold:
		// Bold and faint share the same SGR code for this
		self.cursor.style.flags.bold = false
		self.cursor.style.flags.faint = false

	case SGRAttributeTagItalic:
		self.cursor.style.flags.italic = true

	case SGRAttributeTagResetItalic:
		self.cursor.style.flags.italic = false

	case SGRAttributeTagFaint:
		self.cursor.style.flags.faint = true

	case SGRAttributeTagUnderline:
		self.cursor.style.flags.underline = attr.Underline

	case SGRAttributeTagUnderlineColor:
		rgb := attr.UnderlineColor
		self.cursor.style.underline_color = StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
			r: rgb.r,
			g: rgb.g,
			b: rgb.b,
		}}

	case SGRAttributeTag256UnderlineColor:
		self.cursor.style.underline_color = StyleColor{Tag: StyleColorPalette, Palette: attr.X256UnderlineColor}

	case SGRAttributeTagResetUnderlineColor:
		self.cursor.style.underline_color = StyleColor{Tag: StyleColorNone}

	case SGRAttributeTagOverline:
		self.cursor.style.flags.overline = true

	case SGRAttributeTagResetOverline:
		self.cursor.style.flags.overline = false

	case SGRAttributeTagBlink:
		self.cursor.style.flags.blink = true

	case SGRAttributeTagResetBlink:
		self.cursor.style.flags.blink = false

	case SGRAttributeTagInverse:
		self.cursor.style.flags.inverse = true

	case SGRAttributeTagResetInverse:
		self.cursor.style.flags.inverse = false

	case SGRAttributeTagInvisible:
		self.cursor.style.flags.invisible = true

	case SGRAttributeTagResetInvisible:
		self.cursor.style.flags.invisible = false

	case SGRAttributeTagStrikethrough:
		self.cursor.style.flags.strikethrough = true

	case SGRAttributeTagResetStrikethrough:
		self.cursor.style.flags.strikethrough = false

	case SGRAttributeTagDirectColorFg:
		rgb := attr.DirectColorFg
		self.cursor.style.fg_color = StyleColor{
			Tag: StyleColorRgb,
			Rgb: ColorRGB{
				r: rgb.r,
				g: rgb.g,
				b: rgb.b,
			},
		}

	case SGRAttributeTagDirectColorBg:
		rgb := attr.DirectColorBg
		self.cursor.style.bg_color = StyleColor{
			Tag: StyleColorRgb,
			Rgb: ColorRGB{
				r: rgb.r,
				g: rgb.g,
				b: rgb.b,
			},
		}

	case SGRAttributeTag8Fg:
		self.cursor.style.fg_color = StyleColor{Tag: StyleColorPalette, Palette: uint8(attr.X8Fg)}

	case SGRAttributeTag8Bg:
		self.cursor.style.bg_color = StyleColor{Tag: StyleColorPalette, Palette: uint8(attr.X8Bg)}

	case SGRAttributeTagResetFg:
		self.cursor.style.fg_color = StyleColor{Tag: StyleColorNone}

	case SGRAttributeTagResetBg:
		self.cursor.style.bg_color = StyleColor{Tag: StyleColorNone}

	case SGRAttributeTag8BrightFg:
		self.cursor.style.fg_color = StyleColor{Tag: StyleColorPalette, Palette: uint8(attr.X8BrightFg)}

	case SGRAttributeTag8BrightBg:
		self.cursor.style.bg_color = StyleColor{Tag: StyleColorPalette, Palette: uint8(attr.X8BrightBg)}

	case SGRAttributeTag256Fg:
		self.cursor.style.fg_color = StyleColor{Tag: StyleColorPalette, Palette: attr.X256Fg}

	case SGRAttributeTag256Bg:
		self.cursor.style.bg_color = StyleColor{Tag: StyleColorPalette, Palette: attr.X256Bg}

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
	page := &self.cursor.page_pin.node.data

	// std.log.warn("active styles={}", .{page.styles.count()});

	// Release our previous style if it was not default.
	if self.cursor.style_id != StyleDefaultId {
		page.styles.Release(self.cursor.style_id)
	}

	// If our new style is the default, just reset to that
	if self.cursor.style.Default() {
		self.cursor.style_id = StyleDefaultId
		return nil
	}

	// Clear the cursor style ID to prevent weird things from happening
	// if the page capacity has to be adjusted which would end up calling
	// manualStyleUpdate again.
	//
	// This also ensures that if anything fails below, we fall back to
	// clearing our style.
	self.cursor.style_id = StyleDefaultId

	// After setting the style, we need to update our style map.
	// Note that we COULD lazily do this in print. We should look into
	// if that makes a meaningful difference. Our priority is to keep print
	// fast because setting a ton of styles that do nothing is uncommon
	// and weird.
	id, err := page.styles.Add(
		self.cursor.style,
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
			self.cursor.page_pin.node,
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
				if err := self.splitForCapacity(*self.cursor.page_pin); err != nil {
					return err
				}
				node = self.cursor.page_pin.node
			default:
				panic("unreachable")
			}
		}

		page = &node.data
		id2, err2 := page.styles.Add(
			self.cursor.style,
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

	self.cursor.style_id = id
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
	bytes_above := PageLayoutOf(pin.node.data.ExactRowCapacity(
		0,
		uint(pin.y)+1,
	)).total_size
	bytes_below := PageLayoutOf(pin.node.data.ExactRowCapacity(
		uint(pin.y),
		uint(pin.node.data.size.rows),
	)).total_size

	// We need to track the old cursor pin because if our split
	// moves the cursor pin we need to update our accounting.
	old_cursor := *self.cursor.page_pin

	// If our bytes above are less than bytes below, we move the pin
	// to split down one since splitting includes the pinned row in
	// the new node.
	split_pin := pin
	if bytes_above < bytes_below {
		if down := pin.Down(1); down.Valid {
			split_pin = down.Value
		}
	}
	if err := self.pages.Split(split_pin); err != nil {
		return err
	}

	// Cursor didn't change nodes, we're done.
	if self.cursor.page_pin.node == old_cursor.node {
		return nil
	}

	// Cursor changed, we need to restore the old pin then use
	// cursorChangePin to move to the new pin. The old node is guaranteed
	// to still exist, just not the row.
	//
	// Note that page_row and all that will be invalid, it points to the
	// new node, but at the time of writing this we don't need any of that
	// to be right in cursorChangePin.
	new_cursor := *self.cursor.page_pin
	*self.cursor.page_pin = old_cursor
	self.cursorChangePin(new_cursor)
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
	defer func() { self.cursor.page_pin.node.data.AssertIntegrity() }()
	if err := self.cursor.page_pin.node.data.AppendGrapheme(
		self.cursor.page_row,
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
				page := &self.cursor.page_pin.node.data
				zero := page.cellIndex(self.cursor.page_cell) - uint(self.cursor.x)
				target := page.cellIndex(cell)
				assert(target >= zero) // Zig: - traps
				cell_idx = target - zero
			}

			// Adjust our capacity. This will update our cursor page pin and
			// force us to reload.
			if _, err := self.IncreaseCapacity(
				self.cursor.page_pin.node,
				Some(PageListIncreaseCapacityGraphemeBytes),
			); err != nil {
				return err
			}

			// The cell pointer is now invalid, so we need to get it from
			// the reloaded cursor pointers.
			var reloaded_cell *PageCell
			switch {
			case cell_idx == uint(self.cursor.x):
				reloaded_cell = self.cursor.page_cell
			case cell_idx < uint(self.cursor.x):
				reloaded_cell = self.CursorCellLeft(SizeCellCountInt(uint(self.cursor.x) - cell_idx))
			default:
				// Zig: @intCast to u16.
				assert(cell_idx-uint(self.cursor.x) <= 0xFFFF)
				reloaded_cell = self.CursorCellRight(SizeCellCountInt(cell_idx - uint(self.cursor.x)))
			}

			if err2 := self.cursor.page_pin.node.data.AppendGrapheme(
				self.cursor.page_row,
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
	link.uri = uri
	if id_.Valid {
		link.id = HyperlinkHyperlinkId{
			Tag:      HyperlinkHyperlinkIdExplicit,
			Explicit: id_.Value,
		}
	} else {
		// implicit:
		link.id = HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: self.cursor.hyperlink_implicit_id}
		// defer self.cursor.hyperlink_implicit_id += 1; (u32, traps on
		// overflow)
		assert(self.cursor.hyperlink_implicit_id != 0xFFFFFFFF)
		self.cursor.hyperlink_implicit_id += 1
	}
	defer func() {
		// errdefer
		if err != nil {
			switch link.id.Tag {
			case HyperlinkHyperlinkIdExplicit:
			case HyperlinkHyperlinkIdImplicit:
				// Cannot underflow: it was incremented above.
				self.cursor.hyperlink_implicit_id -= 1
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
				self.cursor.page_pin.node,
				Some(PageListIncreaseCapacityStringBytes),
			); err != nil {
				return err
			}

		// hyperlink set is out of memory, adjust it up
		case ErrSetOutOfMemory:
			if _, err := self.IncreaseCapacity(
				self.cursor.page_pin.node,
				Some(PageListIncreaseCapacityHyperlinkBytes),
			); err != nil {
				return err
			}

		// hyperlink set is too full, rehash it
		case ErrSetNeedsRehash:
			if _, err := self.IncreaseCapacity(
				self.cursor.page_pin.node,
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
	page := &self.cursor.page_pin.node.data
	id, err := page.InsertHyperlink(*link)
	if err != nil {
		return err
	}

	// Save it all
	self.cursor.hyperlink = link
	self.cursor.hyperlink_id = id
	return nil
}

// EndHyperlink is Screen.endHyperlink.
//
// End the hyperlink state so that future cells aren't part of the
// current hyperlink (if any). This is safe to call multiple times.
func (self *Screen) EndHyperlink() {
	// If we have no hyperlink state then do nothing
	if self.cursor.hyperlink_id == 0 {
		assert(self.cursor.hyperlink == nil)
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
	page := &self.cursor.page_pin.node.data
	page.hyperlink_set.Release(self.cursor.hyperlink_id)
	assert(self.cursor.hyperlink != nil) // .?
	self.cursor.hyperlink.Deinit()
	// Zig: self.alloc.destroy(self.cursor.hyperlink.?); Go memory.
	self.cursor.hyperlink_id = 0
	self.cursor.hyperlink = nil
}

// CursorSetHyperlink is Screen.cursorSetHyperlink.
//
// Set the current hyperlink state on the current cell.
//
// Zig: PageList.IncreaseCapacityError!void. The error is ErrOutOfMemory
// (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Screen) CursorSetHyperlink() error {
	assert(self.cursor.hyperlink_id != 0)

	page := &self.cursor.page_pin.node.data
	err := page.SetHyperlink(
		self.cursor.page_row,
		self.cursor.page_cell,
		self.cursor.hyperlink_id,
	)
	if err == nil {
		// Success, increase the refcount for the hyperlink.
		page.hyperlink_set.Use(self.cursor.hyperlink_id)
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
		for self.cursor.hyperlink != nil {
			link := self.cursor.hyperlink
			if slice, err := page.string_alloc.Alloc(
				uint(len(link.uri)),
			); err == nil {
				// We don't bother freeing because we're
				// about to free the entire page anyway.
				_ = slice
				break
			}

			// We didn't have enough room, let's increase string bytes
			new_node, err := self.IncreaseCapacity(
				self.cursor.page_pin.node,
				Some(PageListIncreaseCapacityStringBytes),
			)
			if err != nil {
				return err
			}
			assert(new_node == self.cursor.page_pin.node)
			page = &new_node.data
		}

		if _, err := self.IncreaseCapacity(
			self.cursor.page_pin.node,
			Some(PageListIncreaseCapacityHyperlinkBytes),
		); err != nil {
			return err
		}

		// Retry
		//
		// We check that the cursor hyperlink hasn't been destroyed
		// by the capacity adjustment first though- since despite the
		// terrible code above, that can still apparently happen ._.
		if self.cursor.hyperlink_id > 0 {
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
	cursor := &self.cursor

	switch t.Tag {
	case ScreenCursorSetSemanticContentTagOutput:
		cursor.semantic_content = PageCellSemanticContentOutput
		cursor.semantic_content_clear_eol = false

	case ScreenCursorSetSemanticContentTagInput:
		clear := t.Input
		cursor.semantic_content = PageCellSemanticContentInput
		switch clear {
		case ScreenCursorSetSemanticContentInputClearExplicit:
			cursor.semantic_content_clear_eol = false
		case ScreenCursorSetSemanticContentInputClearEol:
			cursor.semantic_content_clear_eol = true
		default:
			panic("unreachable")
		}

	case ScreenCursorSetSemanticContentTagPrompt:
		kind := t.Prompt
		self.semantic_prompt.seen = true
		cursor.semantic_content = PageCellSemanticContentPrompt
		cursor.semantic_content_clear_eol = false
		switch kind {
		case OSCParsersSemanticPromptPromptKindInitial, OSCParsersSemanticPromptPromptKindRight:
			cursor.page_row.semantic_prompt = PageRowSemanticPromptPrompt
		case OSCParsersSemanticPromptPromptKindContinuation, OSCParsersSemanticPromptPromptKindSecondary:
			cursor.page_row.semantic_prompt = PageRowSemanticPromptPromptContinuation
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
	if self.selection.Valid {
		old := &self.selection.Value
		old.Deinit(self)
	}
	self.selection = Some(tracked_sel)
	self.dirty.selection = true
	return nil
}

// ClearSelection is Screen.clearSelection.
//
// Same as select(null) but can't fail.
func (self *Screen) ClearSelection() {
	if self.selection.Valid {
		sel := &self.selection.Value
		sel.Deinit(self)
		self.dirty.selection = true
	}
	self.selection = None[Selection]()
}

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
	// We'll use this as our buffer to build our string.
	aw := oscWriterAllocatingInit()
	defer aw.deinit()

	// Create a formatter and use that to emit our text.
	formatter := FormatterScreenFormatterInit(
		self,
		FormatterOptions{
			emit:   FormatterFormatPlain,
			unwrap: true,
			trim:   opts.trim,
			// Zig default: `.{}`, an empty non-null list.
			codepoint_map: Optional[[]FormatterCodepointMap]{Valid: true},
		},
	)
	formatter.content = FormatterScreenFormatterContent{
		Tag:       FormatterScreenFormatterContentSelection,
		Selection: Some(opts.sel),
	}

	// If we have a string map, we need to set that up.
	var pins []PageListPin // std.ArrayList(Pin) = .empty
	// Zig: defer pins.deinit(alloc); Go memory.
	if opts.map_ != nil {
		formatter.pin_map = Some(FormatterPinMap{
			map_: &pins,
		})
	}

	// Emit. Since this is an allocating writer, a failed write
	// just becomes an OOM.
	if err := formatter.Format(&aw.writer); err != nil {
		return nil, ErrOutOfMemory
	}

	// Build our final text and if we have a string map set that up.
	text, err := aw.toOwnedSliceSentinel(0)
	if err != nil {
		return nil, err
	}
	// Zig: errdefer alloc.free(text); Go memory.
	if opts.map_ != nil {
		map_ := opts.map_
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
			string: map_string,
			map_:   map_pins,
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
	_ = self

	// Get the current point semantic prompt state since that determines
	// boundary conditions too. This makes it so that line selection can
	// only happen within the same prompt state. For example, if you triple
	// click output, but the shell uses spaces to soft-wrap to the prompt
	// then the selection will stop prior to the prompt. See issue #1329.
	var semantic_prompt_state Optional[PageCellSemanticContent]
	if opts.semantic_prompt_boundary {
		rac := opts.pin.RowAndCell()
		semantic_prompt_state = Some(rac.cell.semantic_content)
	}

	// The real start of the row is the first row in the soft-wrap.
	//
	// Zig: the labeled block `start_pin:`.
	start_pin := func() PageListPin {
		it := opts.pin.RowIterator(PageListDirectionLeftUp, None[PageListPin]())
		it_prev_ := it.Next()  // skip self
		assert(it_prev_.Valid) // .?
		it_prev := it_prev_.Value

		// First, check the current row for semantic boundaries before the clicked position.
		if semantic_prompt_state.Valid {
			v := semantic_prompt_state.Value
			row := it_prev.RowAndCell().row
			cells := it_prev.node.data.GetCells(row)
			// Scan backwards from clicked position to find where our content starts
			n := uint(opts.pin.x) + 1
			for i := uint(0); i < n; i++ {
				x_rev := uint(opts.pin.x) - i
				if cells[x_rev].semantic_content != v {
					copy_ := it_prev
					// Zig: @intCast to u16; x_rev < pin.x here.
					copy_.x = SizeCellCountInt(x_rev + 1)
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
				copy_.x = 0
				return copy_
			}

			if semantic_prompt_state.Valid {
				v := semantic_prompt_state.Value
				// We need to check every cell in this row in reverse
				// order since we're going up and back.
				cells := p.node.data.GetCells(row)
				for x := 0; x < len(cells); x++ {
					x_rev := len(cells) - 1 - x
					cell := cells[x_rev]
					if cell.semantic_content != v {
						return it_prev
					}
					it_prev = p
					it_prev.x = SizeCellCountInt(x_rev)
				}

				continue
			}

			it_prev = p
		}
		// Zig: the `else` of the while loop.
		copy_ := it_prev
		copy_.x = 0
		return copy_
	}()

	// The real end of the row is the final row in the soft-wrap.
	//
	// Zig: the labeled block `end_pin:`; its `return null` returns from
	// selectLine.
	end_pin_ := func() Optional[PageListPin] {
		it := opts.pin.RowIterator(PageListDirectionRightDown, None[PageListPin]())
		for p_ := it.Next(); p_.Valid; p_ = it.Next() {
			p := p_.Value
			row := p.RowAndCell().row

			if semantic_prompt_state.Valid {
				v := semantic_prompt_state.Value
				// We need to check every cell in this row
				cells := p.node.data.GetCells(row)

				// If this is our pin row we can start from our x because
				// the start_pin logic already found the real start.
				var start_offset uint = 0
				if p.node == opts.pin.node &&
					p.y == opts.pin.y {
					start_offset = uint(opts.pin.x)
				}

				// Handle the zero case specially because if the first
				// col doesn't match then we end at the end of the prior
				// row. But if this is the first row, we can't go back,
				// so we scan forward to find where our content ends.
				if start_offset == 0 && cells[0].semantic_content != v {
					prev_ := p.Up(1)
					assert(prev_.Valid) // .?
					prev := prev_.Value
					prev.x = p.node.data.size.cols - 1
					return Some(prev)
				}

				// For every other case, we end at the prior cell.
				for x := start_offset; x < uint(len(cells)); x++ {
					cell := cells[x]
					if cell.semantic_content != v {
						copy_ := p
						assert(x >= 1) // Zig: - traps
						copy_.x = SizeCellCountInt(x - 1)
						return Some(copy_)
					}
				}
			}

			if !row.wrap {
				copy_ := p
				copy_.x = p.node.data.size.cols - 1
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
	if !opts.whitespace.Valid {
		start = start_pin
	} else {
		whitespace := opts.whitespace.Value
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
	if !opts.whitespace.Valid {
		end = end_pin
	} else {
		whitespace := opts.whitespace.Value
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
		it := self.pages.CellIterator(
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
		it := self.pages.CellIterator(
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
			if p.x == p.node.data.size.cols-1 && !rac.row.wrap {
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
			if p.x == p.node.data.size.cols-1 && !rac.row.wrap {
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
		start_pin := self.pages.GetTopLeft(PointTagScreen)
		end_pin_ := next.Up(1)
		if !end_pin_.Valid {
			return PageListPin{}, None[Selection](), true
		}
		end_pin := end_pin_.Value
		end_pin.x = end_pin.node.data.size.cols - 1
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
	hl_ := self.pages.HighlightSemanticContent(
		prompt_pin,
		PageCellSemanticContentOutput,
	)
	if !hl_.Valid {
		return None[Selection]()
	}
	hl := hl_.Value

	// Trim our trailing whitespace
	cell_it := hl.end.CellIterator(PageListDirectionLeftUp, Some(hl.start))
	for p_ := cell_it.Next(); p_.Valid; p_ = cell_it.Next() {
		p := p_.Value
		cell := p.RowAndCell().cell
		hl.end = p
		if cell.HasText() {
			break
		}
	}

	return Some(SelectionInit(hl.start, hl.end, false))
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
		pin:                      current,
		whitespace:               None[[]uint32](),
		semantic_prompt_boundary: false,
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
	// If we're not at an input cell with our cursor, no movement will
	// ever be possible.
	if self.cursor.semantic_content != PageCellSemanticContentInput &&
		self.cursor.page_cell.semantic_content != PageCellSemanticContentInput {
		return ScreenPromptClickMoveZero
	}

	switch self.semantic_prompt.click.Tag {
	// None doesn't support movement and click_events must use a
	// different mechanism (SGR mouse events) that callers must handle.
	case ScreenSemanticPromptSemanticClickNone, ScreenSemanticPromptSemanticClickClickEvents:
		return ScreenPromptClickMoveZero
	case ScreenSemanticPromptSemanticClickCl:
		cl := self.semantic_prompt.click.Cl
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
	cursor_pin := *self.cursor.page_pin
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
			cells := row_pin.node.data.GetCells(rac.row)

			// Determine if this row is our cursor.
			is_cursor_row := row_pin.node == cursor_pin.node &&
				row_pin.y == cursor_pin.y

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
					return uint(cursor_pin.x) + 1
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
				if row_pin.node == click_pin.node &&
					row_pin.y == click_pin.y &&
					x == uint(click_pin.x) {
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
				if self.cursor.page_cell.semantic_content == PageCellSemanticContentInput {
					count += 1
				}

				break
			}
		}

		return ScreenPromptClickMove{left: 0, right: count}
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
		cells := row_pin.node.data.GetCells(rac.row)

		// Determine the length of the cells we look at in this row.
		//
		// Zig: the labeled block `end_len:`.
		var end_len uint
		if row_pin.node == cursor_pin.node &&
			row_pin.y == cursor_pin.y {
			// If this is our cursor row then we end before the cursor.
			end_len = uint(cursor_pin.x)
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
			if row_pin.node == click_pin.node &&
				row_pin.y == click_pin.y &&
				x == uint(click_pin.x) {
				break left_row_it
			}
		}

		// If this row is not a wrap continuation, then break out
		if !rac.row.wrap_continuation {
			break
		}
	}

	return ScreenPromptClickMove{left: count, right: 0}
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
	// Create a formatter and use that to emit our text.
	formatter := FormatterScreenFormatterInit(self, FormatterOptions{
		emit:   FormatterFormatPlain,
		unwrap: opts.unwrap,
		trim:   false,
		// Zig default: `.{}`, an empty non-null list.
		codepoint_map: Optional[[]FormatterCodepointMap]{Valid: true},
	})

	// Set up the selection based on the pins
	tl := opts.tl
	var br PageListPin
	if opts.br.Valid {
		br = opts.br.Value
	} else {
		br_ := self.pages.GetBottomRight(PointTagScreen)
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
	return formatter.Format(writer)
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

	tl_pin := self.pages.GetTopLeft(tl.Tag)
	br := self.pages.GetBottomRight(tl.Tag)
	if !br.Valid {
		return nil, ErrUnknownPoint
	}
	if err := self.DumpString(&builder.writer, ScreenDumpStringOptions{
		tl:     tl_pin,
		br:     Some(br.Value),
		unwrap: false,
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

	tl_pin := self.pages.GetTopLeft(tl.Tag)
	br := self.pages.GetBottomRight(tl.Tag)
	if !br.Valid {
		return nil, ErrUnknownPoint
	}
	if err := self.DumpString(&builder.writer, ScreenDumpStringOptions{
		tl:     tl_pin,
		br:     Some(br.Value),
		unwrap: true,
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
			self.cursor.pending_wrap = false
			if self.cursor.semantic_content_clear_eol {
				self.CursorSetSemanticContent(ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagOutput})
			} else {
				switch self.cursor.semantic_content {
				case PageCellSemanticContentOutput:
				case PageCellSemanticContentPrompt, PageCellSemanticContentInput:
					self.cursor.page_row.semantic_prompt = PageRowSemanticPromptPromptContinuation
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
			// Zig: @intCast(unicode.table.get(c).width)
			panic(notImplemented) // needs unicode
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

			if err := self.cursor.page_pin.node.data.AppendGrapheme(
				self.cursor.page_row,
				cell,
				c,
			); err != nil {
				return err
			}
			continue
		}

		if self.cursor.pending_wrap {
			assert(self.cursor.x == self.pages.cols-1)
			self.cursor.pending_wrap = false
			self.cursor.page_row.wrap = true
			if err := self.cursorDownOrScroll(); err != nil {
				return err
			}
			self.CursorHorizontalAbsolute(0)
			self.cursor.page_row.wrap_continuation = true
			switch self.cursor.semantic_content {
			case PageCellSemanticContentOutput:
			case PageCellSemanticContentInput, PageCellSemanticContentPrompt:
				self.cursor.page_row.semantic_prompt = PageRowSemanticPromptPromptContinuation
			default:
				panic("unreachable")
			}
		}

		assert(width == 1 || width == 2)
		switch width {
		case 1:
			*self.cursor.page_cell = PageCell{
				content_tag:      PageCellContentTagCodepoint,
				content:          PageCellContent{codepoint: c},
				style_id:         self.cursor.style_id,
				protected:        self.cursor.protected,
				semantic_content: self.cursor.semantic_content,
			}

			// If we have a ref-counted style, increase.
			if self.cursor.style_id != StyleDefaultId {
				// Zig: `const page = self.cursor.page_pin.node.data;`, a
				// copy of the Page struct; `use` only writes the item's
				// ref count, which the copy shares.
				page := self.cursor.page_pin.node.data
				page.styles.Use(self.cursor.style_id)
				self.cursor.page_row.styled = true
			}

			// If we have a hyperlink, add it to the cell.
			if self.cursor.hyperlink_id > 0 {
				if err := self.CursorSetHyperlink(); err != nil {
					return err
				}
			}

		case 2:
			// Need a wide spacer head
			if self.cursor.x == self.pages.cols-1 {
				*self.cursor.page_cell = PageCell{
					content_tag:      PageCellContentTagCodepoint,
					content:          PageCellContent{codepoint: 0},
					wide:             PageCellWideSpacerHead,
					protected:        self.cursor.protected,
					semantic_content: self.cursor.semantic_content,
				}

				// If we have a hyperlink, add it to the cell.
				if self.cursor.hyperlink_id > 0 {
					if err := self.CursorSetHyperlink(); err != nil {
						return err
					}
				}

				self.cursor.page_row.wrap = true
				if err := self.cursorDownOrScroll(); err != nil {
					return err
				}
				self.CursorHorizontalAbsolute(0)
				self.cursor.page_row.wrap_continuation = true
			}

			// Write our wide char
			*self.cursor.page_cell = PageCell{
				content_tag:      PageCellContentTagCodepoint,
				content:          PageCellContent{codepoint: c},
				style_id:         self.cursor.style_id,
				wide:             PageCellWideWide,
				protected:        self.cursor.protected,
				semantic_content: self.cursor.semantic_content,
			}

			// If we have a hyperlink, add it to the cell.
			if self.cursor.hyperlink_id > 0 {
				if err := self.CursorSetHyperlink(); err != nil {
					return err
				}
			}

			// Write our tail
			self.CursorRight(1)
			*self.cursor.page_cell = PageCell{
				content_tag:      PageCellContentTagCodepoint,
				content:          PageCellContent{codepoint: 0},
				wide:             PageCellWideSpacerTail,
				protected:        self.cursor.protected,
				semantic_content: self.cursor.semantic_content,
			}

			// If we have a hyperlink, add it to the cell.
			if self.cursor.hyperlink_id > 0 {
				if err := self.CursorSetHyperlink(); err != nil {
					return err
				}
			}

			// If we have a ref-counted style, increase twice.
			if self.cursor.style_id != StyleDefaultId {
				// Zig: a copy of the Page struct, as above.
				page := self.cursor.page_pin.node.data
				page.styles.Use(self.cursor.style_id)
				page.styles.Use(self.cursor.style_id)
				self.cursor.page_row.styled = true
			}

		default:
			panic("unreachable")
		}

		if uint32(self.cursor.x)+1 < uint32(self.pages.cols) {
			self.CursorRight(1)
		} else {
			self.cursor.pending_wrap = true
		}
	}
	return nil
}
