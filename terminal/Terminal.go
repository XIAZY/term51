// Port of src/terminal/Terminal.zig (Ghostty v1.3.1).
//
// The primary terminal emulation structure. This represents a single
// "terminal" containing a grid of characters and exposes various operations
// on that grid. This also maintains the scrollback buffer.
//
// File naming: Terminal.zig is a file-level struct, so the type is
// `Terminal` and its constructor is `TerminalInit`. What is nested in it
// takes the prefix `Terminal`: `Terminal.Colors` is `TerminalColors`,
// `Terminal.Dirty` is `TerminalDirty`, `Terminal.MouseEvents` is
// `TerminalMouseEvents`, `Terminal.MouseFormat` is `TerminalMouseFormat`,
// `Terminal.ScrollingRegion` is `TerminalScrollingRegion`,
// `Terminal.Options` is `TerminalOptions`, `Terminal.SemanticPrompt` is
// `TerminalSemanticPrompt`, `Terminal.ScrollViewport` is
// `TerminalScrollViewport`, `Terminal.DeccolmMode` is `TerminalDeccolmMode`,
// `Terminal.SwitchScreenMode` is `TerminalSwitchScreenMode`. The private
// file-level constant `TABSTOP_INTERVAL` is `terminalTabstopInterval`.
// Private methods keep their Zig name (`gpa`, `printCell`, `printWrap`,
// `semanticPromptFreshLine`, `rowWillBeShifted`, `isDirty`, `clearDirty`);
// `pub fn` methods are exported (`Print`, `SetCursorPos`, `Resize`).
//
// Declarations appear in the same order as in Terminal.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory.
//
//   - Every allocator parameter is dropped: `init`, `deinit`, `deccolm`,
//     `resize`, `plainString`, `plainStringUnwrapped`. `deinit` is kept; the
//     tests call it and it deinitialises the tabstops and the screens.
//   - `pwd` is a `std.ArrayList(u8)`: a `[]uint8`. `items.len` is
//     `len(pwd)`, `capacity` is `cap(pwd)`, `clearRetainingCapacity` is
//     `pwd = pwd[:0]`.
//   - `gpa` returns the allocator of the active screen
//     (`self.screens.active.alloc`). Screen has no `alloc` field in Go, so
//     there is nothing to return. The stub is kept, returning nothing,
//     because a project test calls it.
//   - `printAttributes` writes into a caller-supplied `[]u8` and returns
//     the written part of it: `buf []uint8` in, a sub-slice of `buf` out.
//   - `plainString` and `plainStringUnwrapped` return `[]const u8` owned by
//     the caller: a `[]uint8`.
//   - `getPwd` returns `?[]const u8` that aliases `pwd`:
//     `Optional[[]uint8]` whose value is `pwd` itself, not a copy.
//   - `switchScreen` returns `!?*Screen`: `(*Screen, error)`, nil is null.
//   - Cell pointers stepped in Zig (`[*]Cell` in insertBlanks, deleteChars,
//     eraseChars, eraseLine, rowWillBeShifted, decaln) are indexes into the
//     page's `cells` slice, as in Screen.go.
//
// # Widths
//
// As in Screen.go: every byte quantity is a `uint64` whatever the platform
// word size, counts and indexes are `uint` (Zig `usize`), `isize` is
// `int64` (the reference's width: `ScrollViewport.delta`), and fixed Zig
// widths stay fixed. Byte quantity: `Options.max_scrollback`
// (passed to `ScreenSetInit` as `ScreenOptions.max_scrollback`). Counts
// (`uint`): every `count`, `count_req`, `row_req`, `col_req`, `top_req`,
// `bottom_req`, `left_req`, `right_req` parameter. `u21` is `uint32`;
// `width_px` and `height_px` are `u32`.
//
// # Packed structs
//
// `Terminal.flags` and `Terminal.Dirty` are packed structs in Zig. In Go
// they are ordinary structs with the same field names; no non-test Zig code
// converts either to its integer, so no conversion is declared. A test that
// writes `@bitCast(t.flags)` to compare two states compares the structs.
//
// # Declarations that are not Zig declarations
//
//   - `TerminalFlags`: the anonymous packed struct that is the type of the
//     field `flags`;
//   - `TerminalFlagsMouseShiftCapture`: the anonymous `enum(u2) { null,
//     false, true }` that is the type of `flags.mouse_shift_capture`;
//   - `TerminalScrollViewportTag`: the tag type of the tagged union
//     `ScrollViewport`;
//   - the `String` methods of the enums.
//
// # Omitted
//
// Compiled out (`build_options.kitty_graphics` is false in the library
// build):
//   - `kittyGraphics` (line 2681): `pub fn kittyGraphics(self: *Terminal,
//     alloc: Allocator, cmd: *kitty.graphics.Command)
//     ?kitty.graphics.Response`. With the option false `kitty.graphics` is
//     an empty struct, so its parameter and result types do not exist and
//     the function cannot be instantiated;
//   - every `if (comptime build_options.kitty_graphics)` block: the
//     placeholder check in `printCell`, the `kitty_images.dirty` writes in
//     `index`, `scrollUp`, `insertLines`, `deleteLines`, `eraseDisplay` and
//     `switchScreen`, and the `kitty_image_storage_limit` inherited from the
//     primary screen in `switchScreen` (the `else` branch, 0, is what the
//     library build passes).
// Aliases and imports with nothing to declare: `Terminal` (@This), `std`,
// `build_options`, `assert`, `testing`, `Allocator`, `unicode`, `uucode`,
// `ansi`, `modespkg`, `charsets`, `csi`, `hyperlink`, `kitty`, `osc`,
// `point`, `sgr`, `Tabstops`, `color`, `mouse_shape_pkg`, `ReadonlyHandler`
// (StreamReadonlyHandler), `ReadonlyStream` (StreamReadonlyStream), `size`,
// `pagepkg`, `style`, `Screen`, `ScreenSet`, `Page`, `Cell`, `Row`, and the
// `log` scope (the port uses warn("terminal", ...)).
// Types local to a function body, which are logic and not declarations:
// `Prev` and the `desired_wide` enum in `print`, `WrapMode` in `cursorLeft`,
// the `params` struct in `setCursorPos`.
// No C ABI declaration exists in this file.
//
// # What the implementation needs from files that have no Go file yet
//
// No signature in this file needs a type from an unported file, so no
// function is left undeclared for that reason. These bodies do:
//   - `print` (line 302) reads `unicode.table` (UnicodeTable.Get: width,
//     `emoji_vs_base`, `width_zero_in_grapheme`, `grapheme_break`) and calls
//     `unicode.graphemeBreak` (UnicodeGraphemeBreak) with a
//     `uucode.grapheme.BreakState` (UucodeGraphemeBreakState), all in
//     unicode_*.go and uucode.go;
//   - `plainString` (line 3058) and `plainStringUnwrapped` (line 3063) call
//     Screen.dumpStringAlloc and Screen.dumpStringAllocUnwrapped, which are
//     written with `ScreenFormatter` (formatter.zig);
//   - `vtStream` and `vtHandler` build the stream_readonly.zig handler
//     (stream_readonly.go).
//
// # Implementation notes
//
//   - `defer self.screens.active.f()` is a deferred closure, so that
//     `self.screens.active` is read when the defer runs, as in Zig (Go would
//     evaluate the receiver when the defer statement runs). A `defer` inside
//     an inner Zig block (`pauseIntegrityChecks` in insertBlanks and
//     deleteChars) is a closure called in place, so it runs where Zig's does.
//   - Zig's `errdefer` in decaln is a deferred closure on the named error
//     result; `errdefer screen_set.deinit(alloc)` in init only frees memory
//     and is a comment.
//   - Labeled blocks: `grapheme:` and `transfer_graphemes:` in print are left
//     with `goto` to a label after the block; `break :wide` / `break :narrow`
//     leave a block that is a whole switch prong and are Go's `break`;
//     `click:` in semanticPrompt and `at_prompt:` in eraseDisplay are blocks
//     followed by nothing and are closures left with `return`; the
//     `while ... else` in `at_prompt:` is a labeled loop whose fall-through
//     returns.
//   - `[*]Cell` pointers stepped in Zig are indexes into the cursor page's
//     `cells` (Page.cellIndex); in insertBlanks the backwards walk uses a
//     signed index because Zig's pointer may step one before `left`.
//   - `@min` of a `usize` and a `CellCountInt` has type CellCountInt in Zig
//     (the result cannot exceed the smaller type); Go converts the result.
//   - Integer traps made explicit: `col + x_offset` and `row + y_offset`
//     (usize) in setCursorPos, `rows - 1` / `cols - 1` in init and resize,
//     `scrolling_region.left - 1` in rowWillBeShifted, `prev.left - 1` in
//     print's narrow branch, and `.?` unwraps (`assert(x.Valid)` or
//     `assert(p != nil)`).
//   - `log.warn` / `log.err` are `warn("terminal", ...)` (an unscoped
//     `std.log.warn` is the `default` scope); `log.debug` is a comment.
//
// # Zig defaults a Go literal does not give
//
// A Go composite literal leaves an omitted field at the Go zero value. The
// Zig defaults that are NOT the Go zero value, which a translated literal
// must write out:
//   - `Options.max_scrollback` defaults to 10_000 (bytes); the zero value
//     is 0, which means "no scrollback";
//   - `Options.colors` defaults to `Colors.default`
//     (`TerminalColorsDefault`), whose palette is
//     `ColorDynamicPaletteDefault`; the zero value has an all-black palette;
//   - `Options.default_modes` defaults to `ModePacked{}`
//     (`ModesModePackedInit()`), in which six modes are true; the zero value
//     is all false. So `Terminal.init(alloc, .{ .cols = 10, .rows = 10 })`
//     in Zig is `TerminalInit(TerminalOptions{cols: 10, rows: 10,
//     max_scrollback: 10_000, colors: TerminalColorsDefault,
//     default_modes: ModesModePackedInit()})` in Go;
//   - `flags.focused` defaults to TRUE. So Zig's `flags = .{}` (the field
//     default, and what `fullReset` assigns) is
//     `TerminalFlags{focused: true}` in Go.
// The fields of Terminal itself also have defaults that are not the Go zero
// value, but a Terminal is only made by TerminalInit, which must give them:
// `mouse_shape` defaults to `.text` (`MouseShapeText`; the zero value is
// `.default`), `modes` to `ModeState{}` (overwritten by `init` with the
// option `default_modes`) and `flags` as above.
// All other defaults in this file are the Go zero value (`null`, 0, false,
// the first enum value) and are noted on the fields. In particular
// `flags.shell_redraws_prompt` defaults to `.true`
// (`OSCParsersSemanticPromptRedrawTrue`), which is the zero value.

package terminal

import (
	"math"
	"reflect"
	"unicode/utf8"
)

// Default tabstop interval
//
// Zig: const TABSTOP_INTERVAL = 8 (private).
const terminalTabstopInterval = 8

// Terminal is the file-level struct of Terminal.zig.
//
// The Go zero value is not an initialised terminal; use TerminalInit.
type Terminal struct {
	// The set of screens behind this terminal (e.g. primary vs alternate).
	screens ScreenSet

	// Whether we're currently writing to the status line (DECSASD and DECSSDT).
	// We don't support a status line currently so we just black hole this
	// data so that it doesn't mess up our main display.
	//
	// Zig default: .main.
	status_display ANSIStatusDisplay

	// Where the tabstops are.
	tabstops Tabstops

	// The size of the terminal.
	rows SizeCellCountInt
	cols SizeCellCountInt

	// The size of the screen in pixels. This is used for pty events and images
	//
	// Zig: u32, default 0.
	width_px uint32
	// Zig: u32, default 0.
	height_px uint32

	// The current scrolling region.
	scrolling_region TerminalScrollingRegion

	// The last reported pwd, if any.
	//
	// Zig: std.ArrayList(u8). See the file header.
	pwd []uint8

	// The color state for this terminal.
	colors TerminalColors

	// The previous printed character. This is used for the repeat previous
	// char CSI (ESC [ <n> b).
	//
	// Zig: ?u21, default null.
	previous_char Optional[uint32]

	// The modes that this terminal currently has active.
	//
	// Zig default: `.{}`, which is NOT the Go zero value (see modes.go);
	// `init` sets `values` and `default` from the option `default_modes`.
	modes ModesModeState

	// The most recently set mouse shape for the terminal.
	//
	// Zig default: .text, which is NOT the Go zero value (.default).
	mouse_shape MouseShape

	// These are just a packed set of flags we may set on the terminal.
	//
	// Zig default: `.{}`, which is NOT the Go zero value (`focused` is
	// true); see TerminalFlags.
	flags TerminalFlags
}

// TerminalFlags is NOT a named Zig declaration: it is the anonymous packed
// struct that is the type of the field `Terminal.flags`.
//
// These are just a packed set of flags we may set on the terminal.
//
// Zig: packed struct. In Go it is an ordinary struct; no non-test Zig code
// converts it to its integer. The Go zero value is NOT Zig's `.{}`:
// `focused` defaults to true. See the file header.
type TerminalFlags struct {
	// This supports a Kitty extension where programs using semantic
	// prompts (OSC133) can annotate their new prompts with `redraw=0` to
	// disable clearing the prompt on resize.
	//
	// Zig default: .true (OSCParsersSemanticPromptRedrawTrue), the zero
	// value.
	shell_redraws_prompt OSCParsersSemanticPromptRedraw

	// This is set via ESC[4;2m. Any other modify key mode just sets
	// this to false and we act in mode 1 by default.
	//
	// Zig default: false.
	modify_other_keys_2 bool

	// The mouse event mode and format. These are set to the last
	// set mode in modes. You can't get the right event/format to use
	// based on modes alone because modes don't show you what order
	// this was called so we have to track it separately.
	//
	// Zig default: .none.
	mouse_event TerminalMouseEvents
	// Zig default: .x10.
	mouse_format TerminalMouseFormat

	// Set via the XTSHIFTESCAPE sequence. If true (XTSHIFTESCAPE = 1)
	// then we want to capture the shift key for the mouse protocol
	// if the configuration allows it.
	//
	// Zig default: .null.
	mouse_shift_capture TerminalFlagsMouseShiftCapture

	// True if the window is focused.
	//
	// Zig default: TRUE, which is NOT the Go zero value.
	focused bool

	// True if the terminal is in a password entry mode. This is set
	// to true based on termios state. This is set
	// to true based on termios state.
	//
	// Zig default: false.
	password_input bool

	// True if the terminal should perform selection scrolling.
	//
	// Zig default: false.
	selection_scroll bool

	// Dirty flag used only by the search thread. The renderer is expected
	// to set this to true if the viewport was dirty as it was rendering.
	// This is used by the search thread to more efficiently re-search the
	// viewport and active area.
	//
	// Since the renderer is going to inspect the viewport/active area ANYWAYS,
	// this lets our search thread do less work and hold the lock less time,
	// resulting in more throughput for everything.
	//
	// Zig default: false.
	search_viewport_dirty bool

	// Dirty flags for the renderer.
	//
	// Zig default: `.{}`.
	dirty TerminalDirty
}

// TerminalFlagsMouseShiftCapture is NOT a named Zig declaration: it is the
// anonymous enum that is the type of `Terminal.flags.mouse_shift_capture`.
//
// Zig: enum(u2) { null, false, true }, values numbered from zero in the
// order listed.
type TerminalFlagsMouseShiftCapture uint8

const (
	TerminalFlagsMouseShiftCaptureNull TerminalFlagsMouseShiftCapture = iota
	TerminalFlagsMouseShiftCaptureFalse
	TerminalFlagsMouseShiftCaptureTrue
)

// terminalFlagsMouseShiftCaptureNames are the tag names of
// TerminalFlagsMouseShiftCapture, in order.
var terminalFlagsMouseShiftCaptureNames = [...]string{"null", "false", "true"}

// String returns the Zig tag name (@tagName).
func (self TerminalFlagsMouseShiftCapture) String() string {
	return terminalFlagsMouseShiftCaptureNames[self]
}

// TerminalColors is Terminal.Colors.
//
// The various color configurations a terminal maintains and that can
// be set dynamically via OSC, with defaults usually coming from a
// configuration.
type TerminalColors struct {
	background ColorDynamicRGB
	foreground ColorDynamicRGB
	cursor     ColorDynamicRGB
	palette    ColorDynamicPalette
}

// TerminalColorsDefault is Terminal.Colors.default.
//
// Zig: `.{ .background = .unset, .foreground = .unset, .cursor = .unset,
// .palette = .default }`, that is ColorDynamicRGBUnset three times and
// ColorDynamicPaletteDefault, initialised once at start-up (Go's package
// initialisation runs ColorDynamicPaletteDefault's initialiser first).
var TerminalColorsDefault = TerminalColors{
	background: ColorDynamicRGBUnset,
	foreground: ColorDynamicRGBUnset,
	cursor:     ColorDynamicRGBUnset,
	palette:    ColorDynamicPaletteDefault,
}

// TerminalDirty is Terminal.Dirty.
//
// This is a set of dirty flags the renderer can use to determine
// what parts of the screen need to be redrawn. It is up to the renderer
// to clear these flags.
//
// This only contains dirty flags for terminal state, not for the screen
// state. The screen state has its own dirty flags.
//
// Zig: packed struct, every field default false. In Go it is an ordinary
// struct; no non-test Zig code converts it to its integer.
type TerminalDirty struct {
	// Set when the color palette is modified in any way.
	palette bool

	// Set when the reverse colors mode is modified.
	reverse_colors bool

	// Screen clear of some kind. This can be due to a screen change,
	// erase display, etc.
	clear bool

	// Set when the pre-edit is modified.
	preedit bool
}

// TerminalMouseEvents is Terminal.MouseEvents.
//
// The event types that can be reported for mouse-related activities.
// These are all mutually exclusive (hence in a single enum).
//
// Zig: enum(u3).
type TerminalMouseEvents uint8

const (
	TerminalMouseEventsNone   TerminalMouseEvents = 0
	TerminalMouseEventsX10    TerminalMouseEvents = 1 // 9
	TerminalMouseEventsNormal TerminalMouseEvents = 2 // 1000
	TerminalMouseEventsButton TerminalMouseEvents = 3 // 1002
	TerminalMouseEventsAny    TerminalMouseEvents = 4 // 1003
)

// terminalMouseEventsNames are the tag names of TerminalMouseEvents, by
// value.
var terminalMouseEventsNames = [...]string{"none", "x10", "normal", "button", "any"}

// String returns the Zig tag name (@tagName).
func (self TerminalMouseEvents) String() string { return terminalMouseEventsNames[self] }

// Motion is Terminal.MouseEvents.motion.
//
// Returns true if this event sends motion events.
func (self TerminalMouseEvents) Motion() bool {
	return self == TerminalMouseEventsButton || self == TerminalMouseEventsAny
}

// TerminalMouseFormat is Terminal.MouseFormat.
//
// The format of mouse events when enabled.
// These are all mutually exclusive (hence in a single enum).
//
// Zig: enum(u3).
type TerminalMouseFormat uint8

const (
	TerminalMouseFormatX10       TerminalMouseFormat = 0
	TerminalMouseFormatUtf8      TerminalMouseFormat = 1 // 1005
	TerminalMouseFormatSgr       TerminalMouseFormat = 2 // 1006
	TerminalMouseFormatUrxvt     TerminalMouseFormat = 3 // 1015
	TerminalMouseFormatSgrPixels TerminalMouseFormat = 4 // 1016
)

// terminalMouseFormatNames are the tag names of TerminalMouseFormat, by
// value.
var terminalMouseFormatNames = [...]string{"x10", "utf8", "sgr", "urxvt", "sgr_pixels"}

// String returns the Zig tag name (@tagName).
func (self TerminalMouseFormat) String() string { return terminalMouseFormatNames[self] }

// TerminalScrollingRegion is Terminal.ScrollingRegion.
//
// Scrolling region is the area of the screen designated where scrolling
// occurs. When scrolling the screen, only this viewport is scrolled.
type TerminalScrollingRegion struct {
	// Top and bottom of the scroll region (0-indexed)
	// Precondition: top < bottom
	top    SizeCellCountInt
	bottom SizeCellCountInt

	// Left/right scroll regions.
	// Precondition: right > left
	// Precondition: right <= cols - 1
	left  SizeCellCountInt
	right SizeCellCountInt
}

// TerminalOptions is Terminal.Options.
//
// The Go zero value of three fields is NOT the Zig default; see the file
// header.
type TerminalOptions struct {
	cols SizeCellCountInt
	rows SizeCellCountInt

	// Zig: usize, default 10_000, which is NOT the Go zero value. Bytes: it
	// is passed to ScreenSet.init as Screen.Options.max_scrollback.
	max_scrollback uint64

	// Zig default: .default (TerminalColorsDefault), which is NOT the Go
	// zero value.
	colors TerminalColors

	// The default mode state. When the terminal gets a reset, it
	// will revert back to this state.
	//
	// Zig default: `.{}` (ModesModePackedInit()), which is NOT the Go zero
	// value.
	default_modes ModesModePacked
}

// TerminalInit is Terminal.init.
//
// Initialize a new terminal.
//
// Zig: pub fn init(alloc: Allocator, opts: Options) !Terminal. The
// allocator parameter is dropped. The inferred error set is that of
// ScreenSet.init and Tabstops.init: ErrOutOfMemory (dcs.go).
func TerminalInit(opts TerminalOptions) (Terminal, error) {
	cols := opts.cols
	rows := opts.rows

	screen_set, err := ScreenSetInit(ScreenOptions{
		cols:           cols,
		rows:           rows,
		max_scrollback: opts.max_scrollback,

		// Zig: the field default, which is not the Go zero value.
		kitty_image_storage_limit: ScreenOptionsDefault.kitty_image_storage_limit,
	})
	if err != nil {
		return Terminal{}, err
	}
	// Zig: errdefer screen_set.deinit(alloc); Go memory.

	tabstops, err := TabstopsInit(uint(cols), terminalTabstopInterval)
	if err != nil {
		return Terminal{}, err
	}

	// Zig: `rows - 1` and `cols - 1` (u16) trap below 1.
	assert(rows >= 1)
	assert(cols >= 1)

	return Terminal{
		cols:     cols,
		rows:     rows,
		screens:  screen_set,
		tabstops: tabstops,
		scrolling_region: TerminalScrollingRegion{
			top:    0,
			bottom: rows - 1,
			left:   0,
			right:  cols - 1,
		},
		pwd:    nil, // Zig: .empty
		colors: opts.colors,
		modes: ModesModeState{
			values:   opts.default_modes,
			saved:    ModesModePackedInit(), // Zig: the field default `.{}`
			default_: opts.default_modes,
		},

		// The field defaults that are not the Go zero value.
		mouse_shape: MouseShapeText,
		flags:       TerminalFlags{focused: true},
	}, nil
}

// Deinit is Terminal.deinit.
//
// Zig: pub fn deinit(self: *Terminal, alloc: Allocator) void. The allocator
// parameter is dropped. Kept: it deinitialises the tabstops and the
// screens, and every test calls it.
func (self *Terminal) Deinit() {
	self.tabstops.Deinit()
	self.screens.Deinit()
	// Zig: self.pwd.deinit(alloc); Go memory.
	*self = Terminal{} // Zig: self.* = undefined
}

// VtStream is Terminal.vtStream.
//
// Return a terminal.Stream that can process VT streams and update this
// terminal state. The streams will only process read-only data that
// modifies terminal state. Sequences that query or otherwise require
// output will be ignored.
//
// Zig: pub fn vtStream(self: *Terminal) ReadonlyStream, which is
// `.initAlloc(self.gpa(), self.vtHandler())`. The result is a value; its
// methods have pointer receivers, so a caller assigns it to a variable
// (`s := t.VtStream()`) before calling `s.NextSlice(...)`.
func (self *Terminal) VtStream() StreamReadonlyStream {
	// Zig: `.initAlloc(self.gpa(), self.vtHandler())`. Go has no allocator
	// (gpa returns nothing); StreamInitAlloc only tells nil from non-nil, and
	// Zig always passes an allocator here, so a non-nil value stands for it.
	// The Zig stream holds the handler by value; Go's holds a pointer, here
	// to the stream's own copy.
	self.gpa()
	handler := self.VtHandler()
	return StreamInitAlloc[*StreamReadonlyHandler](terminalGpa{}, &handler)
}

// terminalGpa is NOT a Zig declaration: the non-nil value VtStream passes to
// StreamInitAlloc where Zig passes `self.gpa()`.
type terminalGpa struct{}

// VtHandler is Terminal.vtHandler.
//
// This is the handler-side only for vtStream.
//
// Zig: pub fn vtHandler(self: *Terminal) ReadonlyHandler.
func (self *Terminal) VtHandler() StreamReadonlyHandler {
	return StreamReadonlyHandlerInit(self)
}

// gpa is Terminal.gpa.
//
// The general allocator we should use for this terminal.
//
// Zig: fn gpa(self: *Terminal) Allocator, returning
// `self.screens.active.alloc`. Go has no allocator and Screen has no
// `alloc` field, so the stub returns nothing; it is kept because a project
// test calls it. See the file header.
//
// It returns nothing in Go (there is no allocator) and exists only so that
// the translated call (`t.gpa()`) compiles; the translated tests cannot
// compare its result.
func (self *Terminal) gpa() {
	// Zig: return self.screens.active.alloc; Go has no allocator.
}

// PrintString is Terminal.printString.
//
// Print UTF-8 encoded string to the terminal.
//
// Zig: pub fn printString(self: *Terminal, str: []const u8) !void. The
// inferred error set is ErrInvalidUtf8 (Screen.go; std.unicode.Utf8View.init)
// and those of linefeed and print.
func (self *Terminal) PrintString(str []uint8) error {
	// Zig: std.unicode.Utf8View.init validates the whole string first, then
	// its iterator yields the code points; utf8.Valid accepts exactly the
	// same strings and ranging over the validated string yields the same
	// code points (osc_encoding.go).
	if !utf8.Valid(str) {
		return ErrInvalidUtf8
	}
	for _, r := range string(str) {
		cp := uint32(r)
		switch cp {
		case '\n':
			self.CarriageReturn()
			if err := self.Linefeed(); err != nil {
				return err
			}

		default:
			if err := self.Print(cp); err != nil {
				return err
			}
		}
	}
	return nil
}

// PrintRepeat is Terminal.printRepeat.
//
// Print the previous printed character a repeated amount of times.
//
// Zig: pub fn printRepeat(self: *Terminal, count_req: usize) !void. The
// inferred error set is that of print.
func (self *Terminal) PrintRepeat(count_req uint) error {
	if self.previous_char.Valid {
		c := self.previous_char.Value
		count := max(count_req, 1)
		for range count {
			if err := self.Print(c); err != nil {
				return err
			}
		}
	}
	return nil
}

// Print is Terminal.print.
//
// Zig: pub fn print(self: *Terminal, c: u21) !void. The inferred error set
// is that of printWrap and Screen.appendGrapheme: ErrOutOfMemory (dcs.go),
// ErrOutOfSpace (PageList.go) and Page.CloneFromError (page.go).
func (self *Terminal) Print(c uint32) error {
	// log.debug("print={x} y={} x={}", .{ c, self.screens.active.cursor.y, self.screens.active.cursor.x });

	// If we're not on the main display, do nothing for now
	if self.status_display != ANSIStatusDisplayMain {
		return nil
	}

	// After doing any printing, wrapping, scrolling, etc. we want to ensure
	// that our screen remains in a consistent state.
	defer func() { self.screens.active.AssertIntegrity() }()

	// Our right margin depends where our cursor is now.
	var right_limit SizeCellCountInt
	if self.screens.active.cursor.x > self.scrolling_region.right {
		right_limit = self.cols
	} else {
		right_limit = self.scrolling_region.right + 1
	}

	// Perform grapheme clustering if grapheme support is enabled (mode 2027).
	// This is MUCH slower than the normal path so the conditional below is
	// purposely ordered in least-likely to most-likely so we can drop out
	// as quickly as possible.
	if c > 255 &&
		self.modes.Get(ModesModeGraphemeCluster) &&
		self.screens.active.cursor.x > 0 {
		// Zig: the labeled block `grapheme:`; `break :grapheme` is
		// `goto grapheme`, to the label after this block.

		// We need the previous cell to determine if we're at a grapheme
		// break or not. If we are NOT, then we are still combining the
		// same grapheme, and will be appending to prev.cell. Otherwise, we are
		// in a new cell.
		type Prev struct {
			cell *PageCell
			left SizeCellCountInt
		}
		var prev Prev
		{
			// Zig: the labeled block `prev:`.
			var left SizeCellCountInt
			{
				// Zig: the labeled block `left:`.

				// If we have wraparound, then we use the prev col unless
				// there's a pending wrap, in which case we use the current.
				if self.modes.Get(ModesModeWraparound) {
					// Zig: @intFromBool(!pending_wrap)
					if !self.screens.active.cursor.pending_wrap {
						left = 1
					} else {
						left = 0
					}
				} else if self.screens.active.cursor.x != right_limit-1 {
					// If we do not have wraparound, the logic is trickier. If
					// we're not on the last column, then we just use the previous
					// column. Otherwise, we need to check if there is text to
					// figure out if we're attaching to the prev or current.
					left = 1
				} else if self.screens.active.cursor.page_cell.Codepoint() == 0 {
					// Zig: @intFromBool(codepoint() == 0)
					left = 1
				} else {
					left = 0
				}
			}

			// If the previous cell is a wide spacer tail, then we actually
			// want to use the cell before that because that has the actual
			// content.
			immediate := self.screens.active.CursorCellLeft(left)
			switch immediate.wide {
			default:
				prev = Prev{cell: immediate, left: left}
			case PageCellWideSpacerTail:
				prev = Prev{
					cell: self.screens.active.CursorCellLeft(left + 1),
					left: left + 1,
				}
			}
		}

		// If our cell has no content, then this is a new cell and
		// necessarily a grapheme break.
		if prev.cell.Codepoint() == 0 {
			goto grapheme
		}

		{
			var grapheme_break bool
			{
				// Zig: the labeled block `brk:`.
				state := UucodeGraphemeBreakStateDefault
				cp1 := prev.cell.content.codepoint
				if prev.cell.HasGrapheme() {
					cps := self.screens.active.cursor.page_pin.node.data.LookupGrapheme(prev.cell)
					assert(cps.Valid) // .?
					for _, cp2 := range cps.Value {
						// log.debug("cp1={x} cp2={x}", .{ cp1, cp2 });
						assert(!UnicodeGraphemeBreak(cp1, cp2, &state))
						cp1 = cp2
					}
				}

				// log.debug("cp1={x} cp2={x} end", .{ cp1, c });
				grapheme_break = UnicodeGraphemeBreak(cp1, c, &state)
			}

			// If we can NOT break, this means that "c" is part of a grapheme
			// with the previous char.
			if !grapheme_break {
				// Zig: enum { no_change, wide, narrow }.
				const (
					desired_wide_no_change = iota
					desired_wide_wide
					desired_wide_narrow
				)
				desired_wide := desired_wide_no_change

				// If this is an emoji variation selector then we need to modify
				// the cell width accordingly. VS16 makes the character wide and
				// VS15 makes it narrow.
				if c == 0xFE0F || c == 0xFE0E {
					prev_props := UnicodeTable.Get(prev.cell.content.codepoint)
					// Check if it is a valid variation sequence in
					// emoji-variation-sequences.txt, and if not, ignore the char.
					if !prev_props.emoji_vs_base {
						return nil
					}

					switch c {
					case 0xFE0F:
						desired_wide = desired_wide_wide
					case 0xFE0E:
						desired_wide = desired_wide_narrow
					default:
						panic("unreachable")
					}
				} else if !UnicodeTable.Get(c).width_zero_in_grapheme {
					// If we have a code point that contributes to the width of a
					// grapheme, it necessarily means that we're at least at width
					// 2, since the first code point must be at least width 1 to
					// start. (Note that Prepend code points could effectively mean
					// the first code point should be width 0, but we don't handle
					// that yet.)
					desired_wide = desired_wide_wide
				}

				switch desired_wide {
				case desired_wide_wide:
					// Zig: the labeled block `wide:`; `break :wide` is
					// Go's break out of the switch.
					if prev.cell.wide == PageCellWideWide {
						break
					}

					// Move our cursor back to the previous. We'll move
					// the cursor within this block to the proper location.
					self.screens.active.CursorLeft(prev.left)

					// If we don't have space for the wide char, we need to
					// insert spacers and wrap. We need special handling if the
					// previous cell has grapheme data.
					if self.screens.active.cursor.x == right_limit-1 {
						if !self.modes.Get(ModesModeWraparound) {
							return nil
						}

						// This path can write a spacer_head before printWrap
						// which can trigger integrity violations so mark
						// the wrap first to keep the intermediary state valid
						// if we're wrapping.
						row_wrap := right_limit == self.cols
						if row_wrap {
							self.screens.active.cursor.page_row.wrap = true
						}

						prev_cp := prev.cell.content.codepoint
						if prev.cell.HasGrapheme() {
							// This is like printCell but without clearing the
							// grapheme data from the cell, so we can move it
							// later.
							if row_wrap {
								prev.cell.wide = PageCellWideSpacerHead
							} else {
								prev.cell.wide = PageCellWideNarrow
							}
							prev.cell.content.codepoint = 0

							if err := self.printWrap(); err != nil {
								return err
							}
							self.printCell(prev_cp, PageCellWideWide)

							new_pin := *self.screens.active.cursor.page_pin
							new_rac := new_pin.RowAndCell()

							{
								// Zig: the labeled block `transfer_graphemes:`;
								// `break :transfer_graphemes` is `goto
								// transfer_graphemes`, to the label after it.
								old_pin_ := self.screens.active.cursor.page_pin.Up(1)
								if !old_pin_.Valid {
									goto transfer_graphemes
								}
								old_pin := old_pin_.Value
								old_pin.x = right_limit - 1
								old_rac := old_pin.RowAndCell()

								if new_pin.node == old_pin.node {
									new_pin.node.data.MoveGrapheme(prev.cell, new_rac.cell)
									prev.cell.content_tag = PageCellContentTagCodepoint
									new_rac.cell.content_tag = PageCellContentTagCodepointGrapheme
									new_rac.row.grapheme = true
								} else {
									cps := old_pin.node.data.LookupGrapheme(old_rac.cell)
									assert(cps.Valid) // .?
									for _, cp := range cps.Value {
										if err := self.screens.active.AppendGrapheme(new_rac.cell, cp); err != nil {
											return err
										}
									}
									old_pin.node.data.ClearGrapheme(old_rac.cell)
								}

								old_pin.node.data.UpdateRowGraphemeFlag(old_rac.row)
							}
						transfer_graphemes:

							// Point prev.cell to our new previous cell that
							// we'll be appending graphemes to
							prev.cell = new_rac.cell
						} else {
							var wide PageCellWide
							if row_wrap {
								wide = PageCellWideSpacerHead
							} else {
								wide = PageCellWideNarrow
							}
							self.printCell(0, wide)
							if err := self.printWrap(); err != nil {
								return err
							}
							self.printCell(prev_cp, PageCellWideWide)

							// Point prev.cell to our new previous cell that
							// we'll be appending graphemes to
							prev.cell = self.screens.active.cursor.page_cell
						}
					} else {
						prev.cell.wide = PageCellWideWide
					}

					// Write our spacer, since prev.cell is now wide
					self.screens.active.CursorRight(1)
					self.printCell(0, PageCellWideSpacerTail)

					// Move the cursor again so we're beyond our spacer
					if self.screens.active.cursor.x == right_limit-1 {
						self.screens.active.cursor.pending_wrap = true
					} else {
						self.screens.active.CursorRight(1)
					}

				case desired_wide_narrow:
					// Zig: the labeled block `narrow:`; `break :narrow` is
					// Go's break out of the switch.

					// Prev cell is no longer wide
					if prev.cell.wide != PageCellWideWide {
						break
					}
					prev.cell.wide = PageCellWideNarrow

					// Remove the wide spacer tail
					assert(prev.left >= 1) // Zig: prev.left - 1 (u16) traps
					cell := self.screens.active.CursorCellLeft(prev.left - 1)
					cell.wide = PageCellWideNarrow

					// Back track the cursor so that we don't end up with
					// an extra space after the character. Since xterm is
					// not VS aware, it cannot be used as a reference for
					// this behavior; but it does follow the principle of
					// least surprise, and also matches the behavior that
					// can be observed in Kitty, which is one of the only
					// other VS aware terminals.
					if self.screens.active.cursor.x == right_limit-1 {
						// If we're already at the right edge, we stay
						// here and set the pending wrap to false since
						// when we pend a wrap, we only move our cursor once
						// even for wide chars (tests verify).
						self.screens.active.cursor.pending_wrap = false
					} else {
						// Otherwise, move back.
						self.screens.active.CursorLeft(1)
					}

				default:
				}

				// Zig: log.debug("c={X} grapheme attach to left={} primary_cp={X}", .{
				//     c, prev.left, prev.cell.codepoint() });
				self.screens.active.CursorMarkDirty()
				return self.screens.active.AppendGrapheme(prev.cell, c)
			}
		}
	}
grapheme:

	// Determine the width of this character so we can handle
	// non-single-width characters properly. We have a fast-path for
	// byte-sized characters since they're so common. We can ignore
	// control characters because they're always filtered prior.
	var width uint
	if c <= 0xFF {
		width = 1
	} else {
		width = uint(UnicodeTable.Get(c).width)
	}

	// Note: it is possible to have a width of "3" and a width of "-1" from
	// uucode.x's wcwidth. We should look into those cases and handle them
	// appropriately.
	assert(width <= 2)
	// log.debug("c={x} width={}", .{ c, width });

	// Attach zero-width characters to our cell as grapheme data.
	if width == 0 {
		// If we have grapheme clustering enabled, we don't blindly attach
		// any zero width character to our cells and we instead just ignore
		// it.
		if self.modes.Get(ModesModeGraphemeCluster) {
			return nil
		}

		// If we're at cell zero, then this is malformed data and we don't
		// print anything or even store this. Zero-width characters are ALWAYS
		// attached to some other non-zero-width character at the time of
		// writing.
		if self.screens.active.cursor.x == 0 {
			warn("terminal", "zero-width character with no prior character, ignoring")
			return nil
		}

		// Find our previous cell
		var prev *PageCell
		{
			// Zig: the labeled block `prev:`.
			immediate := self.screens.active.CursorCellLeft(1)
			if immediate.wide != PageCellWideSpacerTail {
				prev = immediate
			} else {
				prev = self.screens.active.CursorCellLeft(2)
			}
		}

		// If our previous cell has no text, just ignore the zero-width character
		if !prev.HasText() {
			warn("terminal", "zero-width character with no prior character, ignoring")
			return nil
		}

		// If this is a emoji variation selector, prev must be an emoji
		if c == 0xFE0F || c == 0xFE0E {
			prev_props := UnicodeTable.Get(prev.content.codepoint)
			emoji := prev_props.grapheme_break == UucodeXTypesGraphemeBreakNoControlExtendedPictographic
			if !emoji {
				return nil
			}
		}

		return self.screens.active.AppendGrapheme(prev, c)
	}

	// We have a printable character, save it
	self.previous_char = Some(c)

	// If we're soft-wrapping, then handle that first.
	if self.screens.active.cursor.pending_wrap && self.modes.Get(ModesModeWraparound) {
		if err := self.printWrap(); err != nil {
			return err
		}
	}

	// If we have insert mode enabled then we need to handle that. We
	// only do insert mode if we're not at the end of the line.
	if self.modes.Get(ModesModeInsert) &&
		uint(self.screens.active.cursor.x)+width < uint(self.cols) {
		self.InsertBlanks(width)
	}

	switch width {
	// Single cell is very easy: just write in the cell
	case 1:
		self.screens.active.CursorMarkDirty()
		self.printCell(c, PageCellWideNarrow) // Zig: @call(.always_inline, ...)

	// Wide character requires a spacer. We print this by
	// using two cells: the first is flagged "wide" and has the
	// wide char. The second is guaranteed to be a spacer if
	// we're not at the end of the line.
	case 2:
		assert(right_limit >= self.scrolling_region.left) // Zig: - (u16) traps
		if (right_limit - self.scrolling_region.left) > 1 {
			// If we don't have space for the wide char, we need
			// to insert spacers and wrap. Then we just print the wide
			// char as normal.
			if self.screens.active.cursor.x == right_limit-1 {
				// If we don't have wraparound enabled then we don't print
				// this character at all and don't move the cursor. This is
				// how xterm behaves.
				if !self.modes.Get(ModesModeWraparound) {
					return nil
				}

				// We only create a spacer head if we're at the real edge
				// of the screen. Otherwise, we clear the space with a narrow.
				// This allows soft wrapping to work correctly.
				if right_limit == self.cols {
					// Special-case: we need to set wrap to true even
					// though we call printWrap below because if there is
					// a page resize during printCell then it'll fail
					// integrity checks.
					self.screens.active.cursor.page_row.wrap = true
					self.printCell(0, PageCellWideSpacerHead)
				} else {
					self.printCell(0, PageCellWideNarrow)
				}
				if err := self.printWrap(); err != nil {
					return err
				}
			}

			self.screens.active.CursorMarkDirty()
			self.printCell(c, PageCellWideWide)
			self.screens.active.CursorRight(1)
			self.printCell(0, PageCellWideSpacerTail)
		} else {
			// This is pretty broken, terminals should never be only 1-wide.
			// We should prevent this downstream.
			self.screens.active.CursorMarkDirty()
			self.printCell(0, PageCellWideNarrow)
		}

	default:
		panic("unreachable")
	}

	// If we're at the column limit, then we need to wrap the next time.
	// In this case, we don't move the cursor.
	if self.screens.active.cursor.x == right_limit-1 {
		self.screens.active.cursor.pending_wrap = true
		return nil
	}

	// Move the cursor
	self.screens.active.CursorRight(1)
	return nil
}

// printCell is Terminal.printCell.
//
// Zig: fn printCell(self: *Terminal, unmapped_c: u21, wide: Cell.Wide)
// void.
func (self *Terminal) printCell(unmapped_c uint32, wide PageCellWide) {
	defer func() { self.screens.active.AssertIntegrity() }()

	// TODO: spacers should use a bgcolor only cell

	var c uint32
	{
		// Zig: the labeled block `c:`.

		// TODO: non-utf8 handling, gr

		// If we're single shifting, then we use the key exactly once.
		var key CharsetsSlots
		if self.screens.active.charset.single_shift.Valid {
			key_once := self.screens.active.charset.single_shift.Value
			self.screens.active.charset.single_shift = None[CharsetsSlots]()
			key = key_once
		} else {
			key = self.screens.active.charset.gl
		}

		set := self.screens.active.charset.charsets.Get(key)

		if set == CharsetsCharsetUtf8 || set == CharsetsCharsetAscii {
			// UTF-8 or ASCII is used as-is
			c = unmapped_c
		} else if unmapped_c > math.MaxUint8 {
			// If we're outside of ASCII range this is an invalid value in
			// this table so we just return space.
			c = ' '
		} else {
			// Get our lookup table and map it
			table := CharsetsTable(set)
			c = uint32(table[unmapped_c])
		}
	}

	cell := self.screens.active.cursor.page_cell

	// If the wide property of this cell is the same, then we don't
	// need to do the special handling here because the structure will
	// be the same. If it is NOT the same, then we may need to clear some
	// cells.
	if cell.wide != wide {
		switch cell.wide {
		// Previous cell was narrow. Do nothing.
		case PageCellWideNarrow:

		// Previous cell was wide. We need to clear the tail and head.
		case PageCellWideWide:
			// Zig: the labeled block `wide:`; `break :wide` is Go's break
			// out of the switch.
			if self.screens.active.cursor.x >= self.cols-1 {
				break
			}

			spacer_cell := self.screens.active.CursorCellRight(1)
			page := &self.screens.active.cursor.page_pin.node.data
			i := page.cellIndex(spacer_cell)
			self.screens.active.ClearCells(
				page,
				self.screens.active.cursor.page_row,
				page.cells[i:i+1],
			)

			// If we're near the left edge, a wide char may have
			// wrapped from the previous row, leaving a spacer_head
			// at the end of that row. Clear it so the previous row
			// doesn't keep a stale spacer_head.
			if self.screens.active.cursor.y > 0 && self.screens.active.cursor.x <= 1 {
				head_cell := self.screens.active.CursorCellEndOfPrev()
				if head_cell.wide == PageCellWideSpacerHead {
					head_cell.wide = PageCellWideNarrow
				}
			}

		case PageCellWideSpacerTail:
			assert(self.screens.active.cursor.x > 0)

			// So integrity checks pass. We fix this up later so we don't
			// need to do this without safety checks.
			//
			// Zig: `if (comptime std.debug.runtime_safety)`, true in the
			// safe build the port follows.
			cell.wide = PageCellWideNarrow

			wide_cell := self.screens.active.CursorCellLeft(1)
			page := &self.screens.active.cursor.page_pin.node.data
			i := page.cellIndex(wide_cell)
			self.screens.active.ClearCells(
				page,
				self.screens.active.cursor.page_row,
				page.cells[i:i+1],
			)
			// If we're near the left edge, a wide char may have
			// wrapped from the previous row, leaving a spacer_head
			// at the end of that row. Clear it so the previous row
			// doesn't keep a stale spacer_head.
			if self.screens.active.cursor.y > 0 && self.screens.active.cursor.x <= 1 {
				head_cell := self.screens.active.CursorCellEndOfPrev()
				if head_cell.wide == PageCellWideSpacerHead {
					head_cell.wide = PageCellWideNarrow
				}
			}

		// TODO: this case was not handled in the old terminal implementation
		// but it feels like we should do something. investigate other
		// terminals (xterm mainly) and see what's up.
		case PageCellWideSpacerHead:

		default:
			panic("unreachable")
		}
	}

	// If the prior value had graphemes, clear those
	if cell.HasGrapheme() {
		page := &self.screens.active.cursor.page_pin.node.data
		page.ClearGrapheme(cell)
		page.UpdateRowGraphemeFlag(self.screens.active.cursor.page_row)
	}

	// We don't need to update the style refs unless the
	// cell's new style will be different after writing.
	style_changed := cell.style_id != self.screens.active.cursor.style_id
	if style_changed {
		page := &self.screens.active.cursor.page_pin.node.data

		// Release the old style.
		if cell.style_id != StyleDefaultId {
			assert(self.screens.active.cursor.page_row.styled)
			page.styles.Release(cell.style_id)
		}
	}

	// Keep track if we had a hyperlink so we can unset it.
	had_hyperlink := cell.hyperlink

	// Write
	*cell = PageCell{
		content_tag:      PageCellContentTagCodepoint,
		content:          PageCellContent{codepoint: c},
		style_id:         self.screens.active.cursor.style_id,
		wide:             wide,
		protected:        self.screens.active.cursor.protected,
		semantic_content: self.screens.active.cursor.semantic_content,
	}

	if style_changed {
		page := &self.screens.active.cursor.page_pin.node.data

		// Use the new style.
		if cell.style_id != StyleDefaultId {
			page.styles.Use(cell.style_id)
			self.screens.active.cursor.page_row.styled = true
		}
	}

	// Zig: the Kitty unicode placeholder check is under
	// `build_options.kitty_graphics` (compiled out; see the file header).

	// We check for an active hyperlink first because setHyperlink
	// handles clearing the old hyperlink and an optimization if we're
	// overwriting the same hyperlink.
	if self.screens.active.cursor.hyperlink_id > 0 {
		if err := self.screens.active.CursorSetHyperlink(); err != nil {
			warn("terminal", "error reallocating for more hyperlink space, ignoring hyperlink err=error.%v", err)
			assert(!cell.hyperlink)
		}
	} else if had_hyperlink {
		// If the previous cell had a hyperlink then we need to clear it.
		page := &self.screens.active.cursor.page_pin.node.data
		page.ClearHyperlink(cell)
		page.UpdateRowHyperlinkFlag(self.screens.active.cursor.page_row)
	}
}

// printWrap is Terminal.printWrap.
//
// Zig: fn printWrap(self: *Terminal) !void. The inferred error set is that
// of index.
func (self *Terminal) printWrap() error {
	// We only mark that we soft-wrapped if we're at the edge of our
	// full screen. We don't mark the row as wrapped if we're in the
	// middle due to a right margin.
	cursor := &self.screens.active.cursor
	mark_wrap := cursor.x == self.cols-1
	if mark_wrap {
		cursor.page_row.wrap = true
	}

	// Get the old semantic prompt so we can extend it to the next
	// line. We need to do this before we index() because we may
	// modify memory.
	old_semantic := cursor.semantic_content
	old_semantic_clear := cursor.semantic_content_clear_eol

	// Move to the next line
	if err := self.Index(); err != nil {
		return err
	}
	self.screens.active.CursorHorizontalAbsolute(self.scrolling_region.left)

	// Our pointer should never move
	assert(cursor == &self.screens.active.cursor)

	// We always reset our semantic prompt state
	cursor.semantic_content = old_semantic
	cursor.semantic_content_clear_eol = old_semantic_clear
	switch old_semantic {
	case PageCellSemanticContentOutput, PageCellSemanticContentInput:
	case PageCellSemanticContentPrompt:
		cursor.page_row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	default:
		panic("unreachable")
	}

	if mark_wrap {
		row := self.screens.active.cursor.page_row
		// Always mark the row as a continuation
		row.wrap_continuation = true
	}

	// Assure that our screen is consistent
	self.screens.active.AssertIntegrity()
	return nil
}

// ConfigureCharset is Terminal.configureCharset.
//
// Set the charset into the given slot.
func (self *Terminal) ConfigureCharset(slot CharsetsSlots, set CharsetsCharset) {
	self.screens.active.charset.charsets.Set(slot, set)
}

// InvokeCharset is Terminal.invokeCharset.
//
// Invoke the charset in slot into the active slot. If single is true,
// then this will only be invoked for a single character.
func (self *Terminal) InvokeCharset(active CharsetsActiveSlot, slot CharsetsSlots, single bool) {
	if single {
		assert(active == CharsetsActiveSlotGL)
		self.screens.active.charset.single_shift = Some(slot)
		return
	}

	switch active {
	case CharsetsActiveSlotGL:
		self.screens.active.charset.gl = slot
	case CharsetsActiveSlotGR:
		self.screens.active.charset.gr = slot
	default:
		panic("unreachable")
	}
}

// CarriageReturn is Terminal.carriageReturn.
//
// Carriage return moves the cursor to the first column.
func (self *Terminal) CarriageReturn() {
	// Always reset pending wrap state
	self.screens.active.cursor.pending_wrap = false

	// In origin mode we always move to the left margin
	var x SizeCellCountInt
	if self.modes.Get(ModesModeOrigin) {
		x = self.scrolling_region.left
	} else if self.screens.active.cursor.x >= self.scrolling_region.left {
		x = self.scrolling_region.left
	} else {
		x = 0
	}
	self.screens.active.CursorHorizontalAbsolute(x)
}

// Linefeed is Terminal.linefeed.
//
// Linefeed moves the cursor to the next line.
//
// Zig: !void. The inferred error set is that of index.
func (self *Terminal) Linefeed() error {
	if err := self.Index(); err != nil {
		return err
	}
	if self.modes.Get(ModesModeLinefeed) {
		self.CarriageReturn()
	}
	return nil
}

// Backspace is Terminal.backspace.
//
// Backspace moves the cursor back a column (but not less than 0).
func (self *Terminal) Backspace() {
	self.CursorLeft(1)
}

// CursorUp is Terminal.cursorUp.
//
// Move the cursor up amount lines. If amount is greater than the maximum
// move distance then it is internally adjusted to the maximum. If amount is
// 0, adjust it to 1.
//
// Zig: count_req is usize.
func (self *Terminal) CursorUp(count_req uint) {
	// Always resets pending wrap
	self.screens.active.cursor.pending_wrap = false

	// The maximum amount the cursor can move up depends on scrolling regions
	var max_ SizeCellCountInt // Zig: max
	if self.screens.active.cursor.y >= self.scrolling_region.top {
		max_ = self.screens.active.cursor.y - self.scrolling_region.top
	} else {
		max_ = self.screens.active.cursor.y
	}
	count := min(uint(max_), max(count_req, 1))

	// We can safely intCast below because of the min/max clamping we did above.
	self.screens.active.CursorUp(SizeCellCountInt(count))
}

// CursorDown is Terminal.cursorDown.
//
// Move the cursor down amount lines. If amount is greater than the maximum
// move distance then it is internally adjusted to the maximum. This sequence
// will not scroll the screen or scroll region. If amount is 0, adjust it to 1.
//
// Zig: count_req is usize.
func (self *Terminal) CursorDown(count_req uint) {
	// Always resets pending wrap
	self.screens.active.cursor.pending_wrap = false

	// The max the cursor can move to depends where the cursor currently is
	var max_ SizeCellCountInt // Zig: max
	if self.screens.active.cursor.y <= self.scrolling_region.bottom {
		max_ = self.scrolling_region.bottom - self.screens.active.cursor.y
	} else {
		max_ = self.rows - self.screens.active.cursor.y - 1
	}
	count := min(uint(max_), max(count_req, 1))
	self.screens.active.CursorDown(SizeCellCountInt(count))
}

// CursorRight is Terminal.cursorRight.
//
// Move the cursor right amount columns. If amount is greater than the
// maximum move distance then it is internally adjusted to the maximum.
// This sequence will not scroll the screen or scroll region. If amount is
// 0, adjust it to 1.
//
// Zig: count_req is usize.
func (self *Terminal) CursorRight(count_req uint) {
	// Always resets pending wrap
	self.screens.active.cursor.pending_wrap = false

	// The max the cursor can move to depends where the cursor currently is
	var max_ SizeCellCountInt // Zig: max
	if self.screens.active.cursor.x <= self.scrolling_region.right {
		max_ = self.scrolling_region.right - self.screens.active.cursor.x
	} else {
		max_ = self.cols - self.screens.active.cursor.x - 1
	}
	count := min(uint(max_), max(count_req, 1))
	self.screens.active.CursorRight(SizeCellCountInt(count))
}

// CursorLeft is Terminal.cursorLeft.
//
// Move the cursor to the left amount cells. If amount is 0, adjust it to 1.
//
// Zig: count_req is usize.
func (self *Terminal) CursorLeft(count_req uint) {
	// Wrapping behavior depends on various terminal modes
	//
	// Zig: const WrapMode = enum { none, reverse, reverse_extended };
	const (
		wrap_mode_none = iota
		wrap_mode_reverse
		wrap_mode_reverse_extended
	)
	var wrap_mode int
	{
		// Zig: the labeled block `wrap_mode:`.
		if !self.modes.Get(ModesModeWraparound) {
			wrap_mode = wrap_mode_none
		} else if self.modes.Get(ModesModeReverseWrapExtended) {
			wrap_mode = wrap_mode_reverse_extended
		} else if self.modes.Get(ModesModeReverseWrap) {
			wrap_mode = wrap_mode_reverse
		} else {
			wrap_mode = wrap_mode_none
		}
	}

	count := max(count_req, 1)

	// If we are in no wrap mode, then we move the cursor left and exit
	// since this is the fastest and most typical path.
	if wrap_mode == wrap_mode_none {
		self.screens.active.CursorLeft(SizeCellCountInt(min(count, uint(self.screens.active.cursor.x))))
		self.screens.active.cursor.pending_wrap = false
		return
	}

	// If we have a pending wrap state and we are in either reverse wrap
	// modes then we decrement the amount we move by one to match xterm.
	if self.screens.active.cursor.pending_wrap {
		count -= 1
		self.screens.active.cursor.pending_wrap = false
	}

	// The margins we can move to.
	top := self.scrolling_region.top
	bottom := self.scrolling_region.bottom
	right_margin := self.scrolling_region.right
	var left_margin SizeCellCountInt
	if self.screens.active.cursor.x < self.scrolling_region.left {
		left_margin = 0
	} else {
		left_margin = self.scrolling_region.left
	}

	// Handle some edge cases when our cursor is already on the left margin.
	if self.screens.active.cursor.x == left_margin {
		switch wrap_mode {
		// In reverse mode, if we're already before the top margin
		// then we just set our cursor to the top-left and we're done.
		case wrap_mode_reverse:
			if self.screens.active.cursor.y <= top {
				self.screens.active.CursorAbsolute(left_margin, top)
				return
			}

		// Handled in while loop
		case wrap_mode_reverse_extended:

		// Handled above
		case wrap_mode_none:
			panic("unreachable")
		}
	}

	for {
		// We can move at most to the left margin.
		max_ := self.screens.active.cursor.x - left_margin // Zig: max

		// We want to move at most the number of columns we have left
		// or our remaining count. Do the move.
		amount := SizeCellCountInt(min(uint(max_), count))
		count -= uint(amount)
		self.screens.active.CursorLeft(amount)

		// If we have no more to move, then we're done.
		if count == 0 {
			break
		}

		// If we are at the top, then we are done.
		if self.screens.active.cursor.y == top {
			if wrap_mode != wrap_mode_reverse_extended {
				break
			}

			self.screens.active.CursorAbsolute(right_margin, bottom)
			count -= 1
			continue
		}

		// UNDEFINED TERMINAL BEHAVIOR. This situation is not handled in xterm
		// and currently results in a crash in xterm. Given no other known
		// terminal [to me] implements XTREVWRAP2, I decided to just mimic
		// the behavior of xterm up and not including the crash by wrapping
		// up to the (0, 0) and stopping there. My reasoning is that for an
		// appropriately sized value of "count" this is the behavior that xterm
		// would have. This is unit tested.
		if self.screens.active.cursor.y == 0 {
			assert(self.screens.active.cursor.x == left_margin)
			break
		}

		// If our previous line is not wrapped then we are done.
		if wrap_mode != wrap_mode_reverse_extended {
			prev_row := self.screens.active.CursorRowUp(1)
			if !prev_row.wrap {
				break
			}
		}

		self.screens.active.CursorAbsolute(right_margin, self.screens.active.cursor.y-1)
		count -= 1
	}
}

// SaveCursor is Terminal.saveCursor.
//
// Save cursor position and further state.
//
// The primary and alternate screen have distinct save state. One saved state
// is kept per screen (main / alternative). If for the current screen state
// was already saved it is overwritten.
func (self *Terminal) SaveCursor() {
	self.screens.active.saved_cursor = Some(ScreenSavedCursor{
		x:            self.screens.active.cursor.x,
		y:            self.screens.active.cursor.y,
		style:        self.screens.active.cursor.style,
		protected:    self.screens.active.cursor.protected,
		pending_wrap: self.screens.active.cursor.pending_wrap,
		origin:       self.modes.Get(ModesModeOrigin),
		charset:      self.screens.active.charset,
	})
}

// RestoreCursor is Terminal.restoreCursor.
//
// Restore cursor position and other state.
//
// The primary and alternate screen have distinct save state.
// If no save was done before values are reset to their initial values.
func (self *Terminal) RestoreCursor() {
	var saved ScreenSavedCursor
	if self.screens.active.saved_cursor.Valid {
		saved = self.screens.active.saved_cursor.Value
	} else {
		saved = ScreenSavedCursor{
			x:            0,
			y:            0,
			style:        Style{},
			protected:    false,
			pending_wrap: false,
			origin:       false,
			charset:      ScreenCharsetState{gr: CharsetsSlotsG2}, // Zig: .{}
		}
	}

	// Set the style first because it can fail
	self.screens.active.cursor.style = saved.style
	if err := self.screens.active.ManualStyleUpdate(); err != nil {
		// Regardless of the error here, we revert back to an unstyled
		// cursor. It is more important that the restore succeeds in
		// other attributes because terminals have no way to communicate
		// failure back.
		warn("terminal", "restoreCursor error updating style err=error.%v", err)
		screen := self.screens.active
		screen.cursor.style = Style{}
		if err := self.screens.active.ManualStyleUpdate(); err != nil {
			panic("unreachable") // Zig: catch unreachable
		}
	}

	self.screens.active.charset = saved.charset
	self.modes.Set(ModesModeOrigin, saved.origin)
	self.screens.active.cursor.pending_wrap = saved.pending_wrap
	self.screens.active.cursor.protected = saved.protected
	self.screens.active.CursorAbsolute(
		min(saved.x, self.cols-1),
		min(saved.y, self.rows-1),
	)

	// Ensure our screen is consistent
	self.screens.active.AssertIntegrity()
}

// SetProtectedMode is Terminal.setProtectedMode.
//
// Set the character protection mode for the terminal.
func (self *Terminal) SetProtectedMode(mode ANSIProtectedMode) {
	switch mode {
	case ANSIProtectedModeOff:
		self.screens.active.cursor.protected = false

		// screen.protected_mode is NEVER reset to ".off" because
		// logic such as eraseChars depends on knowing what the
		// _most recent_ mode was.

	case ANSIProtectedModeIso:
		self.screens.active.cursor.protected = true
		self.screens.active.protected_mode = ANSIProtectedModeIso

	case ANSIProtectedModeDec:
		self.screens.active.cursor.protected = true
		self.screens.active.protected_mode = ANSIProtectedModeDec

	default:
		panic("unreachable")
	}
}

// SemanticPrompt is Terminal.semanticPrompt.
//
// Perform a semantic prompt command.
//
// If there is an error, we do our best to get the terminal into
// some coherent state, since callers typically can't handle errors
// (since they're sending sequences via the pty).
//
// Zig: pub fn semanticPrompt(self: *Terminal,
// cmd: osc.Command.SemanticPrompt) !void. The inferred error set is that of
// semanticPromptFreshLine.
func (self *Terminal) SemanticPrompt(cmd OSCCommandSemanticPrompt) error {
	switch cmd.action {
	case OSCParsersSemanticPromptCommandActionFreshLine:
		return self.semanticPromptFreshLine()

	case OSCParsersSemanticPromptCommandActionFreshLineNewPrompt:
		// "First do a fresh-line."
		if err := self.semanticPromptFreshLine(); err != nil {
			return err
		}

		screen := self.screens.active

		// "Subsequent text (until a OSC "133;B" or OSC "133;I" command)
		// is a prompt string (as if followed by OSC 133;P;k=i\007)."
		// Zig: cmd.readOption(.prompt_kind) orelse .initial
		prompt_kind := OSCParsersSemanticPromptPromptKindInitial
		if v := cmd.ReadOption(OSCParsersSemanticPromptOptionPromptKind).PromptKind; v.Valid {
			prompt_kind = v.Value
		}
		screen.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
			Tag:    ScreenCursorSetSemanticContentTagPrompt,
			Prompt: prompt_kind,
		})

		// This is a kitty-specific flag that notes that the shell
		// is NOT capable of redraw. Redraw defaults to true so this
		// usually just disables it, but either is possible.
		if v := cmd.ReadOption(OSCParsersSemanticPromptOptionRedraw).Redraw; v.Valid {
			self.flags.shell_redraws_prompt = v.Value
		}

		func() {
			// Zig: the labeled block `click:`, followed by nothing;
			// `break :click` is `return`.

			// Handle click_events as a priority over cl. click_events
			// is another Kitty-specific extension that converts clicks
			// within a prompt area to SGR mouse events and defers to the
			// shell to handle them.
			if v := cmd.ReadOption(OSCParsersSemanticPromptOptionClickEvents).ClickEvents; v.Valid {
				if v.Value {
					screen.semantic_prompt.click = ScreenSemanticPromptSemanticClick{
						Tag: ScreenSemanticPromptSemanticClickClickEvents,
					}
					return
				}
			}

			// If click_events was not set or disabled, fallback to `cl`.
			if v := cmd.ReadOption(OSCParsersSemanticPromptOptionCl).Cl; v.Valid {
				screen.semantic_prompt.click = ScreenSemanticPromptSemanticClick{
					Tag: ScreenSemanticPromptSemanticClickCl,
					Cl:  v.Value,
				}
			}
		}()

		// The "aid" and "cl" options are also valid for this
		// command but we don't yet handle these in any meaningful way.

	case OSCParsersSemanticPromptCommandActionNewCommand:
		// Spec:
		// Same as OSC "133;A" but may first implicitly terminate a
		// previous command: if the options specify an aid and there
		// is an active (open) command with matching aid, finish the
		// innermost such command (as well as any other commands
		// nested more deeply). If no aid is specified, treat as an
		// aid whose value is the empty string.

		// Ghostty:
		// We don't currently do explicit command tracking in any way
		// so there is no need to terminate prior commands. We just
		// perform the `A` action.
		return self.SemanticPrompt(OSCCommandSemanticPrompt{
			action:              OSCParsersSemanticPromptCommandActionFreshLineNewPrompt,
			options_unvalidated: cmd.options_unvalidated,
		})

	case OSCParsersSemanticPromptCommandActionPromptStart:
		// Explicit start of prompt. Optional after an A or N command.
		// The k (kind) option specifies the type of prompt:
		// regular primary prompt (k=i or default),
		// right-side prompts (k=r), or prompts for continuation lines (k=c or k=s).
		//
		// Zig: cmd.readOption(.prompt_kind) orelse .initial
		prompt_kind := OSCParsersSemanticPromptPromptKindInitial
		if v := cmd.ReadOption(OSCParsersSemanticPromptOptionPromptKind).PromptKind; v.Valid {
			prompt_kind = v.Value
		}
		self.screens.active.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
			Tag:    ScreenCursorSetSemanticContentTagPrompt,
			Prompt: prompt_kind,
		})

	case OSCParsersSemanticPromptCommandActionEndPromptStartInput:
		// End of prompt and start of user input, terminated by a OSC
		// "133;C" or another prompt (OSC "133;P").
		self.screens.active.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
			Tag:   ScreenCursorSetSemanticContentTagInput,
			Input: ScreenCursorSetSemanticContentInputClearExplicit,
		})

	case OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol:
		// End of prompt and start of user input, terminated by end-of-line.
		self.screens.active.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
			Tag:   ScreenCursorSetSemanticContentTagInput,
			Input: ScreenCursorSetSemanticContentInputClearEol,
		})

	case OSCParsersSemanticPromptCommandActionEndInputStartOutput:
		// "End of input, and start of output."
		self.screens.active.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
			Tag: ScreenCursorSetSemanticContentTagOutput,
		})

		// If our current row is marked as a prompt and we're
		// at column zero then we assume we're un-prompting. This
		// is a heuristic to deal with fish, mostly. The issue that
		// fish brings up is that it has no PS2 equivalent and its
		// builtin OSC133 marking doesn't output continuation lines
		// as k=s. So, we assume when we get a newline with a prompt
		// cursor that the new line is also a prompt. But fish changes
		// to output on the newline. So if we're at col 0 we just assume
		// we're overwriting the prompt.
		if self.screens.active.cursor.page_row.semantic_prompt != PageRowSemanticPromptNone &&
			self.screens.active.cursor.x == 0 {
			self.screens.active.cursor.page_row.semantic_prompt = PageRowSemanticPromptNone
		}

	case OSCParsersSemanticPromptCommandActionEndCommand:
		// From a terminal state perspective, this doesn't really do
		// anything. Other terminals appear to do nothing here. I think
		// its reasonable at this point to reset our semantic content
		// state but the spec doesn't really say what to do.
		self.screens.active.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
			Tag: ScreenCursorSetSemanticContentTagOutput,
		})

	default:
		panic("unreachable")
	}
	return nil
}

// semanticPromptFreshLine is Terminal.semanticPromptFreshLine.
//
// OSC 133;L
//
// Zig: fn semanticPromptFreshLine(self: *Terminal) !void. The inferred
// error set is that of index.
func (self *Terminal) semanticPromptFreshLine() error {
	var left_margin SizeCellCountInt
	if self.screens.active.cursor.x < self.scrolling_region.left {
		left_margin = 0
	} else {
		left_margin = self.scrolling_region.left
	}

	// Spec: "If the cursor is the initial column (left, assuming
	// left-to-right writing), do nothing" This specification is very under
	// specified. We are taking the liberty to assume that in a left/right
	// margin context, if the cursor is outside of the left margin, we treat
	// it as being at the left margin for the purposes of this command.
	// This is arbitrary. If someone has a better reasonable idea we can
	// apply it.
	if self.screens.active.cursor.x == left_margin {
		return nil
	}

	self.CarriageReturn()
	return self.Index()
}

// TerminalSemanticPrompt is Terminal.SemanticPrompt.
//
// The semantic prompt type. This is used when tracking a line type and
// requires integration with the shell. By default, we mark a line as "none"
// meaning we don't know what type it is.
//
// See: https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md
//
// Zig: enum { prompt, prompt_continuation, input, command }, values
// numbered from zero in the order listed.
type TerminalSemanticPrompt uint8

const (
	TerminalSemanticPromptPrompt TerminalSemanticPrompt = iota
	TerminalSemanticPromptPromptContinuation
	TerminalSemanticPromptInput
	TerminalSemanticPromptCommand
)

// terminalSemanticPromptNames are the tag names of TerminalSemanticPrompt,
// in order.
var terminalSemanticPromptNames = [...]string{"prompt", "prompt_continuation", "input", "command"}

// String returns the Zig tag name (@tagName).
func (self TerminalSemanticPrompt) String() string { return terminalSemanticPromptNames[self] }

// CursorIsAtPrompt is Terminal.cursorIsAtPrompt.
//
// Returns true if the cursor is currently at a prompt. Another way to look
// at this is it returns false if the shell is currently outputting something.
// This requires shell integration (semantic prompt integration).
//
// If the shell integration doesn't exist, this will always return false.
func (self *Terminal) CursorIsAtPrompt() bool {
	// If we're on the secondary screen, we're never at a prompt.
	if self.screens.active_key == ScreenSetKeyAlternate {
		return false
	}

	// If our page row is a prompt then we're always at a prompt
	cursor := &self.screens.active.cursor
	if cursor.page_row.semantic_prompt != PageRowSemanticPromptNone {
		return true
	}

	// Otherwise, determine our cursor state
	switch cursor.semantic_content {
	case PageCellSemanticContentInput, PageCellSemanticContentPrompt:
		return true
	case PageCellSemanticContentOutput:
		return false
	default:
		panic("unreachable")
	}
}

// HorizontalTab is Terminal.horizontalTab.
//
// Horizontal tab moves the cursor to the next tabstop, clearing
// the screen to the left the tabstop.
func (self *Terminal) HorizontalTab() {
	for self.screens.active.cursor.x < self.scrolling_region.right {
		// Move the cursor right
		self.screens.active.CursorRight(1)

		// If the last cursor position was a tabstop we return. We do
		// "last cursor position" because we want a space to be written
		// at the tabstop unless we're at the end (the while condition).
		if self.tabstops.Get(uint(self.screens.active.cursor.x)) {
			return
		}
	}
}

// HorizontalTabBack is Terminal.horizontalTabBack.
//
// Same as horizontalTab but moves to the previous tabstop instead of the next.
func (self *Terminal) HorizontalTabBack() {
	// With origin mode enabled, our leftmost limit is the left margin.
	var left_limit SizeCellCountInt
	if self.modes.Get(ModesModeOrigin) {
		left_limit = self.scrolling_region.left
	} else {
		left_limit = 0
	}

	for {
		// If we're already at the edge of the screen, then we're done.
		if self.screens.active.cursor.x <= left_limit {
			return
		}

		// Move the cursor left
		self.screens.active.CursorLeft(1)
		if self.tabstops.Get(uint(self.screens.active.cursor.x)) {
			return
		}
	}
}

// TabClear is Terminal.tabClear.
//
// Clear tab stops.
func (self *Terminal) TabClear(cmd CSITabClear) {
	switch cmd {
	case CSITabClearCurrent:
		self.tabstops.Unset(uint(self.screens.active.cursor.x))
	case CSITabClearAll:
		self.tabstops.Reset(0)
	default:
		warn("terminal", "invalid or unknown tab clear setting: %v", cmd)
	}
}

// TabSet is Terminal.tabSet.
//
// Set a tab stop on the current cursor.
// TODO: test
func (self *Terminal) TabSet() {
	self.tabstops.Set(uint(self.screens.active.cursor.x))
}

// TabReset is Terminal.tabReset.
//
// TODO: test
func (self *Terminal) TabReset() {
	self.tabstops.Reset(terminalTabstopInterval)
}

// Index is Terminal.index.
//
// Move the cursor to the next line in the scrolling region, possibly scrolling.
//
// If the cursor is outside of the scrolling region: move the cursor one line
// down if it is not on the bottom-most line of the screen.
//
// If the cursor is inside the scrolling region:
//
//	If the cursor is on the bottom-most line of the scrolling region:
//	  invoke scroll up with amount=1
//	If the cursor is not on the bottom-most line of the scrolling region:
//	  move the cursor one line down
//
// This unsets the pending wrap state without wrapping.
//
// Zig: pub fn index(self: *Terminal) !void. The inferred error set is that
// of Screen.cursorScrollAbove, scrollUp and PageList.eraseRowBounded:
// ErrOutOfMemory (dcs.go) and Page.CloneFromError (page.go).
func (self *Terminal) Index() error {
	screen := self.screens.active

	// Unset pending wrap state
	screen.cursor.pending_wrap = false

	// We handle our cursor semantic prompt state AFTER doing the
	// scrolling, because we may need to apply to new rows.
	defer func() {
		if screen.cursor.semantic_content != PageCellSemanticContentOutput {
			// Always reset any semantic content clear-eol state.
			//
			// The specification is not clear what "end-of-line" means. If we
			// discover that there are more scenarios we should be unsetting
			// this we should document and test it.
			if screen.cursor.semantic_content_clear_eol {
				screen.cursor.semantic_content = PageCellSemanticContentOutput
				screen.cursor.semantic_content_clear_eol = false
			} else {
				// If we aren't clearing our state at EOL and we're not output,
				// then we mark the new row as a prompt continuation. This is
				// to work around shells that don't send OSC 133 k=s sequences
				// for continuations.
				//
				// This can be a false positive if the shell changes content
				// type later and outputs something. We handle that in the
				// semanticPrompt function.
				screen.cursor.page_row.semantic_prompt = PageRowSemanticPromptPromptContinuation
			}
		} else {
			// This should never be set in the output mode.
			assert(!screen.cursor.semantic_content_clear_eol)
		}
	}()

	// Outside of the scroll region we move the cursor one line down.
	if screen.cursor.y < self.scrolling_region.top ||
		screen.cursor.y > self.scrolling_region.bottom {
		// We only move down if we're not already at the bottom of
		// the screen.
		if screen.cursor.y < self.rows-1 {
			screen.CursorDown(1)
		}

		return nil
	}

	// If the cursor is inside the scrolling region and on the bottom-most
	// line, then we scroll up. If our scrolling region is the full screen
	// we create scrollback.
	if screen.cursor.y == self.scrolling_region.bottom &&
		screen.cursor.x >= self.scrolling_region.left &&
		screen.cursor.x <= self.scrolling_region.right {
		// Zig: `screen.kitty_images.dirty = true` under
		// `build_options.kitty_graphics` (compiled out).

		// If our scrolling region is at the top, we create scrollback.
		if self.scrolling_region.top == 0 &&
			self.scrolling_region.left == 0 &&
			self.scrolling_region.right == self.cols-1 {
			return screen.CursorScrollAbove()
		}

		// Slow path for left and right scrolling region margins.
		if self.scrolling_region.left != 0 ||
			self.scrolling_region.right != self.cols-1 ||

			// PERF(mitchellh): If we have an SGR background set then
			// we need to preserve that background in our erased rows.
			// scrollUp does that but eraseRowBounded below does not.
			// However, scrollUp is WAY slower. We should optimize this
			// case to work in the eraseRowBounded codepath and remove
			// this check.
			!screen.BlankCell().IsZero() {
			return self.ScrollUp(1)
		}

		// Otherwise use a fast path function from PageList to efficiently
		// scroll the contents of the scrolling region.

		// Preserve old cursor just for assertions
		old_cursor := screen.cursor

		if err := screen.pages.EraseRowBounded(
			Point{Tag: PointTagActive, Active: PointCoordinate{y: uint32(self.scrolling_region.top)}},
			uint(self.scrolling_region.bottom-self.scrolling_region.top),
		); err != nil {
			return err
		}

		// eraseRow and eraseRowBounded will end up moving the cursor pin
		// up by 1, so we need to move it back down. A `cursorReload`
		// would be better option but this is more efficient and this is
		// a super hot path so we do this instead.
		assert(screen.cursor.x == old_cursor.x)
		assert(screen.cursor.y == old_cursor.y)
		screen.cursor.y -= 1
		screen.CursorDown(1)

		// The operations above can prune our cursor style so we need to
		// update. This should never fail because the above can only FREE
		// memory.
		if err := screen.ManualStyleUpdate(); err != nil {
			warn("default", "deleteLines manualStyleUpdate err=error.%v", err) // Zig: std.log.warn
			screen.cursor.style = Style{}
			if err := screen.ManualStyleUpdate(); err != nil {
				panic("unreachable") // Zig: catch unreachable
			}
		}

		return nil
	}

	// Increase cursor by 1, maximum to bottom of scroll region
	if screen.cursor.y < self.scrolling_region.bottom {
		screen.CursorDown(1)
	}
	return nil
}

// ReverseIndex is Terminal.reverseIndex.
//
// Move the cursor to the previous line in the scrolling region, possibly
// scrolling.
//
// If the cursor is outside of the scrolling region, move the cursor one
// line up if it is not on the top-most line of the screen.
//
// If the cursor is inside the scrolling region:
//
//   - If the cursor is on the top-most line of the scrolling region:
//     invoke scroll down with amount=1
//   - If the cursor is not on the top-most line of the scrolling region:
//     move the cursor one line up
func (self *Terminal) ReverseIndex() {
	if self.screens.active.cursor.y != self.scrolling_region.top ||
		self.screens.active.cursor.x < self.scrolling_region.left ||
		self.screens.active.cursor.x > self.scrolling_region.right {
		self.CursorUp(1)
		return
	}

	self.ScrollDown(1)
}

// SetCursorPos is Terminal.setCursorPos.
//
// Set Cursor Position. Move cursor to the position indicated
// by row and column (1-indexed). If column is 0, it is adjusted to 1.
// If column is greater than the right-most column it is adjusted to
// the right-most column. If row is 0, it is adjusted to 1. If row is
// greater than the bottom-most row it is adjusted to the bottom-most
// row.
//
// Zig: pub fn setCursorPos(self: *Terminal, row_req: usize, col_req: usize)
// void.
func (self *Terminal) SetCursorPos(row_req uint, col_req uint) {
	// If cursor origin mode is set the cursor row will be moved relative to
	// the top margin row and adjusted to be above or at bottom-most row in
	// the current scroll region.
	//
	// If origin mode is set and left and right margin mode is set the cursor
	// will be moved relative to the left margin column and adjusted to be on
	// or left of the right margin column.
	var params struct {
		x_offset SizeCellCountInt // Zig default: 0
		y_offset SizeCellCountInt // Zig default: 0
		x_max    SizeCellCountInt
		y_max    SizeCellCountInt
	}
	if self.modes.Get(ModesModeOrigin) {
		params.x_offset = self.scrolling_region.left
		params.y_offset = self.scrolling_region.top
		params.x_max = self.scrolling_region.right + 1  // We need this 1-indexed
		params.y_max = self.scrolling_region.bottom + 1 // We need this 1-indexed
	} else {
		params.x_max = self.cols
		params.y_max = self.rows
	}

	// Unset pending wrap state
	self.screens.active.cursor.pending_wrap = false

	// Calculate our new x/y
	row := row_req
	if row_req == 0 {
		row = 1
	}
	col := col_req
	if col_req == 0 {
		col = 1
	}
	// Zig: `col + params.x_offset` and `row + params.y_offset` are usize
	// additions, which trap on overflow. usize is 64 bits on the reference,
	// so the sums are computed in uint64 on every platform: on 32-bit ARM a
	// huge column clamps to the margin as on the reference instead of
	// tripping a 32-bit overflow check.
	assert(uint64(col) <= math.MaxUint64-uint64(params.x_offset))
	assert(uint64(row) <= math.MaxUint64-uint64(params.y_offset))
	// Zig: @min(CellCountInt, usize) has type CellCountInt; `-| 1`
	// saturates at 0.
	x := SizeCellCountInt(min(uint64(params.x_max), uint64(col)+uint64(params.x_offset)))
	if x > 0 {
		x -= 1
	}
	y := SizeCellCountInt(min(uint64(params.y_max), uint64(row)+uint64(params.y_offset)))
	if y > 0 {
		y -= 1
	}

	// If the y is unchanged then this is fast pointer math
	if y == self.screens.active.cursor.y {
		if x > self.screens.active.cursor.x {
			self.screens.active.CursorRight(x - self.screens.active.cursor.x)
		} else {
			self.screens.active.CursorLeft(self.screens.active.cursor.x - x)
		}

		return
	}

	// If everything changed we do an absolute change which is slightly slower
	self.screens.active.CursorAbsolute(x, y)
	// log.info("set cursor position: col={} row={}", .{ self.screens.active.cursor.x, self.screens.active.cursor.y });
}

// SetTopAndBottomMargin is Terminal.setTopAndBottomMargin.
//
// Set Top and Bottom Margins If bottom is not specified, 0 or bigger than
// the number of the bottom-most row, it is adjusted to the number of the
// bottom most row.
//
// If top < bottom set the top and bottom row of the scroll region according
// to top and bottom and move the cursor to the top-left cell of the display
// (when in cursor origin mode is set to the top-left cell of the scroll region).
//
// Otherwise: Set the top and bottom row of the scroll region to the top-most
// and bottom-most line of the screen.
//
// Top and bottom are 1-indexed.
//
// Zig: top_req and bottom_req are usize.
func (self *Terminal) SetTopAndBottomMargin(top_req uint, bottom_req uint) {
	top := max(1, top_req)
	bottom_ := bottom_req
	if bottom_req == 0 {
		bottom_ = uint(self.rows)
	}
	// Zig: @min(CellCountInt, usize) has type CellCountInt.
	bottom := SizeCellCountInt(min(uint(self.rows), bottom_))
	if top >= uint(bottom) {
		return
	}

	self.scrolling_region.top = SizeCellCountInt(top - 1)
	self.scrolling_region.bottom = bottom - 1
	self.SetCursorPos(1, 1)
}

// SetLeftAndRightMargin is Terminal.setLeftAndRightMargin.
//
// DECSLRM.
//
// Zig: left_req and right_req are usize.
func (self *Terminal) SetLeftAndRightMargin(left_req uint, right_req uint) {
	// We must have this mode enabled to do anything
	if !self.modes.Get(ModesModeEnableLeftAndRightMargin) {
		return
	}

	left := max(1, left_req)
	right_ := right_req
	if right_req == 0 {
		right_ = uint(self.cols)
	}
	// Zig: @min(CellCountInt, usize) has type CellCountInt.
	right := SizeCellCountInt(min(uint(self.cols), right_))
	if left >= uint(right) {
		return
	}

	self.scrolling_region.left = SizeCellCountInt(left - 1)
	self.scrolling_region.right = right - 1
	self.SetCursorPos(1, 1)
}

// ScrollDown is Terminal.scrollDown.
//
// Scroll the text down by one row.
//
// Zig: pub fn scrollDown(self: *Terminal, count: usize) void.
func (self *Terminal) ScrollDown(count uint) {
	// Preserve our x/y to restore.
	old_x := self.screens.active.cursor.x
	old_y := self.screens.active.cursor.y
	old_wrap := self.screens.active.cursor.pending_wrap
	defer func() {
		self.screens.active.CursorAbsolute(old_x, old_y)
		self.screens.active.cursor.pending_wrap = old_wrap
	}()

	// Move to the top of the scroll region
	self.screens.active.CursorAbsolute(self.scrolling_region.left, self.scrolling_region.top)
	self.InsertLines(count)
}

// ScrollUp is Terminal.scrollUp.
//
// Removes amount lines from the top of the scroll region. The remaining lines
// to the bottom margin are shifted up and space from the bottom margin up
// is filled with empty lines.
//
// The new lines are created according to the current SGR state.
//
// Does not change the (absolute) cursor position.
//
// Zig: pub fn scrollUp(self: *Terminal, count: usize) !void. The inferred
// error set is that of Screen.cursorScrollAbove: ErrOutOfMemory (dcs.go)
// and Page.CloneFromError (page.go).
func (self *Terminal) ScrollUp(count uint) error {
	// Preserve our x/y to restore.
	old_x := self.screens.active.cursor.x
	old_y := self.screens.active.cursor.y
	old_wrap := self.screens.active.cursor.pending_wrap
	defer func() {
		self.screens.active.CursorAbsolute(old_x, old_y)
		self.screens.active.cursor.pending_wrap = old_wrap
	}()

	// If our scroll region is at the top and we have no left/right
	// margins then we move the scrolled out text into the scrollback.
	if self.scrolling_region.top == 0 &&
		self.scrolling_region.left == 0 &&
		self.scrolling_region.right == self.cols-1 {
		// Scrolling dirties the images because it updates their placements pins.
		//
		// Zig: under `build_options.kitty_graphics` (compiled out).

		// Clamp count to the scroll region height.
		region_height := self.scrolling_region.bottom + 1
		adjusted_count := min(count, uint(region_height))

		// TODO: Create an optimized version that can scroll N times
		// This isn't critical because in most cases, scrollUp is used
		// with count=1, but it's still a big optimization opportunity.

		// Move our cursor to the bottom of the scroll region so we can
		// use the cursorScrollAbove function to create scrollback
		self.screens.active.CursorAbsolute(0, self.scrolling_region.bottom)
		for range adjusted_count {
			if err := self.screens.active.CursorScrollAbove(); err != nil {
				return err
			}
		}
		return nil
	}

	// Move to the top of the scroll region
	self.screens.active.CursorAbsolute(self.scrolling_region.left, self.scrolling_region.top)
	self.DeleteLines(count)
	return nil
}

// TerminalScrollViewport is Terminal.ScrollViewport.
//
// Options for scrolling the viewport of the terminal grid.
//
// Zig: union(enum) { top, bottom, delta: isize }. Only the field named by
// Tag is meaningful; `top` and `bottom` have no payload and no field. The
// Go zero value is `.top`.
type TerminalScrollViewport struct {
	Tag TerminalScrollViewportTag

	// Scroll by some delta amount, up is negative.
	//
	// Zig: isize (int64, the reference's width).
	Delta int64
}

// TerminalScrollViewportTag is the tag of Terminal.ScrollViewport, values
// numbered from zero in the order listed.
type TerminalScrollViewportTag uint8

const (
	// Scroll to the top of the scrollback
	TerminalScrollViewportTop TerminalScrollViewportTag = iota

	// Scroll to the bottom, i.e. the top of the active area
	TerminalScrollViewportBottom

	// Scroll by some delta amount, up is negative.
	TerminalScrollViewportDelta
)

// terminalScrollViewportTagNames are the tag names of
// TerminalScrollViewportTag, in order.
var terminalScrollViewportTagNames = [...]string{"top", "bottom", "delta"}

// String returns the Zig tag name (@tagName).
func (self TerminalScrollViewportTag) String() string {
	return terminalScrollViewportTagNames[self]
}

// ScrollViewport is Terminal.scrollViewport.
//
// Scroll the viewport of the terminal grid.
func (self *Terminal) ScrollViewport(behavior TerminalScrollViewport) {
	var scroll ScreenScroll
	switch behavior.Tag {
	case TerminalScrollViewportTop:
		scroll = ScreenScroll{Tag: ScreenScrollTop}
	case TerminalScrollViewportBottom:
		scroll = ScreenScroll{Tag: ScreenScrollActive}
	case TerminalScrollViewportDelta:
		scroll = ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: behavior.Delta}
	default:
		panic("unreachable")
	}
	self.screens.active.Scroll(scroll)
}

// rowWillBeShifted is Terminal.rowWillBeShifted.
//
// To be called before shifting a row (as in insertLines and deleteLines)
//
// Takes care of boundary conditions such as potentially split wide chars
// across scrolling region boundaries and orphaned spacer heads at line
// ends.
//
// Zig: fn rowWillBeShifted(self: *Terminal, page: *Page, row: *Row) void.
func (self *Terminal) rowWillBeShifted(page *Page, row *PageRow) {
	cells := page.rowCells(row)

	// If our scrolling region includes the rightmost column then we
	// need to turn any spacer heads in to normal empty cells, since
	// once we move them they no longer correspond with soft-wrapped
	// wide characters.
	//
	// If it contains either of the 2 leftmost columns, then the wide
	// characters in the first column which may be associated with a
	// spacer head will be either moved or cleared, so we also need
	// to turn the spacer heads in to empty cells in that case.
	if self.scrolling_region.right == self.cols-1 ||
		self.scrolling_region.left < 2 {
		end_cell := &cells[page.size.cols-1]
		if end_cell.wide == PageCellWideSpacerHead {
			end_cell.wide = PageCellWideNarrow
		}
	}

	// If the leftmost or rightmost cells of our scrolling region
	// are parts of wide chars, we need to clear the cells' contents
	// since they'd be split by the move.
	left_cell := &cells[self.scrolling_region.left]
	right_cell := &cells[self.scrolling_region.right]

	if left_cell.wide == PageCellWideSpacerTail {
		assert(self.scrolling_region.left >= 1) // Zig: left - 1 (u16) traps
		wide_cell := &cells[self.scrolling_region.left-1]
		if wide_cell.HasGrapheme() {
			page.ClearGrapheme(wide_cell)
			page.UpdateRowGraphemeFlag(row)
		}
		wide_cell.content.codepoint = 0
		wide_cell.wide = PageCellWideNarrow
		left_cell.wide = PageCellWideNarrow
	}

	if right_cell.wide == PageCellWideWide {
		tail_cell := &cells[self.scrolling_region.right+1]
		if right_cell.HasGrapheme() {
			page.ClearGrapheme(right_cell)
			page.UpdateRowGraphemeFlag(row)
		}
		right_cell.content.codepoint = 0
		right_cell.wide = PageCellWideNarrow
		tail_cell.wide = PageCellWideNarrow
	}
}

// TODO(qwerasd): `insertLines` and `deleteLines` are 99% identical,
// the majority of their logic can (and should) be abstracted in to
// a single shared helper function, probably on `Screen` not here.
// I'm just too lazy to do that rn :p

// InsertLines is Terminal.insertLines.
//
// Insert amount lines at the current cursor row. The contents of the line
// at the current cursor row and below (to the bottom-most line in the
// scrolling region) are shifted down by amount lines. The contents of the
// amount bottom-most lines in the scroll region are lost.
//
// This unsets the pending wrap state without wrapping. If the current cursor
// position is outside of the current scroll region it does nothing.
//
// If amount is greater than the remaining number of lines in the scrolling
// region it is adjusted down (still allowing for scrolling out every remaining
// line in the scrolling region)
//
// In left and right margin mode the margins are respected; lines are only
// scrolled in the scroll region.
//
// All cleared space is colored according to the current SGR state.
//
// Moves the cursor to the left margin.
//
// Zig: pub fn insertLines(self: *Terminal, count: usize) void.
func (self *Terminal) InsertLines(count uint) {
	// Rare, but happens
	if count == 0 {
		return
	}

	// If the cursor is outside the scroll region we do nothing.
	if self.screens.active.cursor.y < self.scrolling_region.top ||
		self.screens.active.cursor.y > self.scrolling_region.bottom ||
		self.screens.active.cursor.x < self.scrolling_region.left ||
		self.screens.active.cursor.x > self.scrolling_region.right {
		return
	}

	// Zig: `self.screens.active.kitty_images.dirty = true` under
	// `build_options.kitty_graphics` (compiled out).

	// At the end we need to return the cursor to the row it started on.
	start_y := self.screens.active.cursor.y
	defer func() {
		self.screens.active.CursorAbsolute(self.scrolling_region.left, start_y)

		// Always unset pending wrap
		self.screens.active.cursor.pending_wrap = false
	}()

	// We have a slower path if we have left or right scroll margins.
	left_right := self.scrolling_region.left > 0 ||
		self.scrolling_region.right < self.cols-1

	// Remaining rows from our cursor to the bottom of the scroll region.
	rem := self.scrolling_region.bottom - self.screens.active.cursor.y + 1

	// We can only insert lines up to our remaining lines in the scroll
	// region. So we take whichever is smaller.
	//
	// Zig: @min(usize, CellCountInt) has type CellCountInt.
	adjusted_count := SizeCellCountInt(min(count, uint(rem)))

	// Create a new tracked pin which we'll use to navigate the page list
	// so that if we need to adjust capacity it will be properly tracked.
	down := self.screens.active.cursor.page_pin.Down(uint(rem - 1))
	assert(down.Valid) // .?
	cur_p, err := self.screens.active.pages.TrackPin(down.Value)
	if err != nil {
		// Zig: comptime assert(@TypeOf(err) == error{OutOfMemory});

		// This error scenario means that our GPA is OOM. This is not a
		// situation we can gracefully handle. We can't just ignore insertLines
		// because it'll result in a corrupted screen. Ideally in the future
		// we flag the state as broken and show an error message to the user.
		// For now, we panic.
		warn("terminal", "insertLines trackPin error err=error.%v", err) // Zig: log.err
		panic("insertLines trackPin OOM")
	}
	defer func() { self.screens.active.pages.UntrackPin(cur_p) }()

	// Our current y position relative to the cursor
	y := uint(rem)

	// Traverse from the bottom up
	for y > 0 {
		cur_rac := cur_p.RowAndCell()
		cur_row := cur_rac.row

		// If this is one of the lines we need to shift, do so
		if y > uint(adjusted_count) {
			off_p_ := cur_p.Up(uint(adjusted_count))
			assert(off_p_.Valid) // .?
			off_p := off_p_.Value
			off_rac := off_p.RowAndCell()
			off_row := off_rac.row

			self.rowWillBeShifted(&cur_p.node.data, cur_row)
			self.rowWillBeShifted(&off_p.node.data, off_row)

			// If our scrolling region is full width, then we unset wrap.
			if !left_right {
				off_row.wrap = false
				cur_row.wrap = false
				off_row.wrap_continuation = false
				cur_row.wrap_continuation = false
			}

			src_p := off_p
			src_row := off_row
			dst_p := cur_p
			dst_row := cur_row

			// If our page doesn't match, then we need to do a copy from
			// one page to another. This is the slow path.
			if src_p.node != dst_p.node {
				if err := dst_p.node.data.ClonePartialRowFrom(
					&src_p.node.data,
					dst_row,
					src_row,
					uint(self.scrolling_region.left),
					uint(self.scrolling_region.right)+1,
				); err != nil {
					// Adjust our page capacity to make
					// room for we didn't have space for
					if _, e := self.screens.active.IncreaseCapacity(
						dst_p.node,
						terminalCloneFromErrorAdjustment(err),
					); e != nil {
						switch e {
						// System OOM. We have no way to recover from this
						// currently. We should probably change insertLines
						// to raise an error here.
						case ErrOutOfMemory:
							panic("increaseCapacity system allocator OOM")

						// The page can't accommodate the managed memory required
						// for this operation. We previously just corrupted
						// memory here so a crash is better. The right long
						// term solution is to allocate a new page here
						// move this row to the new page, and start over.
						case ErrOutOfSpace:
							panic("increaseCapacity OutOfSpace")

						default:
							panic("unreachable")
						}
					}

					// Continue the loop to try handling this row again.
					continue
				}
			} else {
				if !left_right {
					// Swap the src/dst cells. This ensures that our dst gets the
					// proper shifted rows and src gets non-garbage cell data that
					// we can clear.
					dst := *dst_row
					*dst_row = *src_row
					*src_row = dst

					// Ensure what we did didn't corrupt the page
					cur_p.node.data.AssertIntegrity()
				} else {
					// Left/right scroll margins we have to
					// copy cells, which is much slower...
					page := &cur_p.node.data
					page.MoveCells(
						src_row,
						uint(self.scrolling_region.left),
						dst_row,
						uint(self.scrolling_region.left),
						uint(self.scrolling_region.right-self.scrolling_region.left)+1,
					)
				}
			}
		} else {
			// Clear the cells for this row, it has been shifted.
			self.rowWillBeShifted(&cur_p.node.data, cur_row)
			page := &cur_p.node.data
			cells := page.GetCells(cur_row)
			self.screens.active.ClearCells(
				page,
				cur_row,
				cells[self.scrolling_region.left:self.scrolling_region.right+1],
			)
		}

		// Mark the row as dirty
		cur_p.MarkDirty()

		// We have successfully processed a line.
		y -= 1
		// Move our pin up to the next row.
		if p := cur_p.Up(1); p.Valid {
			*cur_p = p.Value
		}
	}
}

// terminalCloneFromErrorAdjustment is NOT a Zig declaration: the `switch
// (err)` over Page.CloneFromError that insertLines and deleteLines both
// write inline as the second argument of increaseCapacity.
func terminalCloneFromErrorAdjustment(err error) Optional[PageListIncreaseCapacity] {
	switch err {
	// Rehash the sets
	case ErrStyleSetNeedsRehash,
		ErrHyperlinkSetNeedsRehash:
		return None[PageListIncreaseCapacity]()

	// Increase style memory
	case ErrStyleSetOutOfMemory:
		return Some(PageListIncreaseCapacityStyles)

	// Increase string memory
	case ErrStringAllocOutOfMemory:
		return Some(PageListIncreaseCapacityStringBytes)

	// Increase hyperlink memory
	case ErrHyperlinkSetOutOfMemory,
		ErrHyperlinkMapOutOfMemory:
		return Some(PageListIncreaseCapacityHyperlinkBytes)

	// Increase grapheme memory
	case ErrGraphemeMapOutOfMemory,
		ErrGraphemeAllocOutOfMemory:
		return Some(PageListIncreaseCapacityGraphemeBytes)

	default:
		panic("unreachable")
	}
}

// DeleteLines is Terminal.deleteLines.
//
// Removes amount lines from the current cursor row down. The remaining lines
// to the bottom margin are shifted up and space from the bottom margin up is
// filled with empty lines.
//
// If the current cursor position is outside of the current scroll region it
// does nothing. If amount is greater than the remaining number of lines in the
// scrolling region it is adjusted down.
//
// In left and right margin mode the margins are respected; lines are only
// scrolled in the scroll region.
//
// If the cell movement splits a multi cell character that character cleared,
// by replacing it by spaces, keeping its current attributes. All other
// cleared space is colored according to the current SGR state.
//
// Moves the cursor to the left margin.
//
// Zig: pub fn deleteLines(self: *Terminal, count: usize) void.
func (self *Terminal) DeleteLines(count uint) {
	// Rare, but happens
	if count == 0 {
		return
	}

	// If the cursor is outside the scroll region we do nothing.
	if self.screens.active.cursor.y < self.scrolling_region.top ||
		self.screens.active.cursor.y > self.scrolling_region.bottom ||
		self.screens.active.cursor.x < self.scrolling_region.left ||
		self.screens.active.cursor.x > self.scrolling_region.right {
		return
	}

	// Zig: `self.screens.active.kitty_images.dirty = true` under
	// `build_options.kitty_graphics` (compiled out).

	// At the end we need to return the cursor to the row it started on.
	start_y := self.screens.active.cursor.y
	defer func() {
		self.screens.active.CursorAbsolute(self.scrolling_region.left, start_y)
		// Always unset pending wrap
		self.screens.active.cursor.pending_wrap = false
	}()

	// We have a slower path if we have left or right scroll margins.
	left_right := self.scrolling_region.left > 0 ||
		self.scrolling_region.right < self.cols-1

	// Remaining rows from our cursor to the bottom of the scroll region.
	rem := self.scrolling_region.bottom - self.screens.active.cursor.y + 1

	// We can only insert lines up to our remaining lines in the scroll
	// region. So we take whichever is smaller.
	//
	// Zig: @min(usize, CellCountInt) has type CellCountInt.
	adjusted_count := SizeCellCountInt(min(count, uint(rem)))

	// Create a new tracked pin which we'll use to navigate the page list
	// so that if we need to adjust capacity it will be properly tracked.
	cur_p, err := self.screens.active.pages.TrackPin(*self.screens.active.cursor.page_pin)
	if err != nil {
		// See insertLines
		// Zig: comptime assert(@TypeOf(err) == error{OutOfMemory});
		warn("terminal", "deleteLines trackPin error err=error.%v", err) // Zig: log.err
		panic("deleteLines trackPin OOM")
	}
	defer func() { self.screens.active.pages.UntrackPin(cur_p) }()

	// Our current y position relative to the cursor
	var y uint = 0

	// Traverse from the top down
	for y < uint(rem) {
		cur_rac := cur_p.RowAndCell()
		cur_row := cur_rac.row

		// If this is one of the lines we need to shift, do so
		if y < uint(rem-adjusted_count) {
			off_p_ := cur_p.Down(uint(adjusted_count))
			assert(off_p_.Valid) // .?
			off_p := off_p_.Value
			off_rac := off_p.RowAndCell()
			off_row := off_rac.row

			self.rowWillBeShifted(&cur_p.node.data, cur_row)
			self.rowWillBeShifted(&off_p.node.data, off_row)

			// If our scrolling region is full width, then we unset wrap.
			if !left_right {
				off_row.wrap = false
				cur_row.wrap = false
				off_row.wrap_continuation = false
				cur_row.wrap_continuation = false
			}

			src_p := off_p
			src_row := off_row
			dst_p := cur_p
			dst_row := cur_row

			// If our page doesn't match, then we need to do a copy from
			// one page to another. This is the slow path.
			if src_p.node != dst_p.node {
				if err := dst_p.node.data.ClonePartialRowFrom(
					&src_p.node.data,
					dst_row,
					src_row,
					uint(self.scrolling_region.left),
					uint(self.scrolling_region.right)+1,
				); err != nil {
					// Adjust our page capacity to make
					// room for we didn't have space for
					if _, e := self.screens.active.IncreaseCapacity(
						dst_p.node,
						terminalCloneFromErrorAdjustment(err),
					); e != nil {
						switch e {
						// See insertLines
						case ErrOutOfMemory:
							panic("increaseCapacity system allocator OOM")

						case ErrOutOfSpace:
							panic("increaseCapacity OutOfSpace")

						default:
							panic("unreachable")
						}
					}

					// Continue the loop to try handling this row again.
					continue
				}
			} else {
				if !left_right {
					// Swap the src/dst cells. This ensures that our dst gets the
					// proper shifted rows and src gets non-garbage cell data that
					// we can clear.
					dst := *dst_row
					*dst_row = *src_row
					*src_row = dst

					// Ensure what we did didn't corrupt the page
					cur_p.node.data.AssertIntegrity()
				} else {
					// Left/right scroll margins we have to
					// copy cells, which is much slower...
					page := &cur_p.node.data
					page.MoveCells(
						src_row,
						uint(self.scrolling_region.left),
						dst_row,
						uint(self.scrolling_region.left),
						uint(self.scrolling_region.right-self.scrolling_region.left)+1,
					)
				}
			}
		} else {
			// Clear the cells for this row, it's from out of bounds.
			self.rowWillBeShifted(&cur_p.node.data, cur_row)
			page := &cur_p.node.data
			cells := page.GetCells(cur_row)
			self.screens.active.ClearCells(
				page,
				cur_row,
				cells[self.scrolling_region.left:self.scrolling_region.right+1],
			)
		}

		// Mark the row as dirty
		cur_p.MarkDirty()

		// We have successfully processed a line.
		y += 1
		// Move our pin down to the next row.
		if p := cur_p.Down(1); p.Valid {
			*cur_p = p.Value
		}
	}
}

// InsertBlanks is Terminal.insertBlanks.
//
// Inserts spaces at current cursor position moving existing cell contents
// to the right. The contents of the count right-most columns in the scroll
// region are lost. The cursor position is not changed.
//
// This unsets the pending wrap state without wrapping.
//
// The inserted cells are colored according to the current SGR state.
//
// Zig: pub fn insertBlanks(self: *Terminal, count: usize) void.
func (self *Terminal) InsertBlanks(count uint) {
	// Unset pending wrap state without wrapping. Note: this purposely
	// happens BEFORE the scroll region check below, because that's what
	// xterm does.
	self.screens.active.cursor.pending_wrap = false

	// If we're given a zero then we do nothing. The rest of this function
	// assumes count > 0 and will crash if zero so return early. Note that
	// this shouldn't be possible with real CSI sequences because the value
	// is clamped to 1 min.
	if count == 0 {
		return
	}

	// If our cursor is outside the margins then do nothing. We DO reset
	// wrap state still so this must remain below the above logic.
	if self.screens.active.cursor.x < self.scrolling_region.left ||
		self.screens.active.cursor.x > self.scrolling_region.right {
		return
	}

	// If our count is larger than the remaining amount, we just erase right.
	// We only do this if we can erase the entire line (no right margin).
	// if (right_limit == self.cols and
	//     count > right_limit - self.screens.active.cursor.x)
	// {
	//     self.eraseLine(.right, false);
	//     return;
	// }

	// left is just the cursor position but as a multi-pointer
	//
	// Go: the index of the cursor cell in the page's cells.
	page := &self.screens.active.cursor.page_pin.node.data
	left := page.cellIndex(self.screens.active.cursor.page_cell)

	// If our X is a wide spacer tail then we need to erase the
	// previous cell too so we don't split a multi-cell character.
	if self.screens.active.cursor.page_cell.wide == PageCellWideSpacerTail {
		assert(self.screens.active.cursor.x > 0)
		self.screens.active.ClearCells(page, self.screens.active.cursor.page_row, page.cells[left-1:left+1])
	}

	// Remaining cols from our cursor to the right margin.
	rem := self.scrolling_region.right - self.screens.active.cursor.x + 1

	// If the cell at the right margin is wide, its spacer tail is
	// outside the scroll region and would be orphaned by either the
	// shift or the clear. Clean up both halves up front.
	{
		right_cell := left + uint(rem-1)
		if page.cells[right_cell].wide == PageCellWideWide {
			self.screens.active.ClearCells(
				page,
				self.screens.active.cursor.page_row,
				page.cells[right_cell:right_cell+2],
			)
		}
	}

	// We can only insert blanks up to our remaining cols
	//
	// Zig: @min(usize, CellCountInt) has type CellCountInt.
	adjusted_count := SizeCellCountInt(min(count, uint(rem)))

	// This is the amount of space at the right of the scroll region
	// that will NOT be blank, so we need to shift the correct cols right.
	// "scroll_amount" is the number of such cols.
	scroll_amount := rem - adjusted_count
	if scroll_amount > 0 {
		func() {
			// Zig: a block-scoped defer; the closure runs it at the end of
			// the block.
			page.PauseIntegrityChecks(true)
			defer page.PauseIntegrityChecks(false)

			// Go: a signed index; Zig's pointer steps one before `left`
			// to end the loop below.
			x := int(left) + int(scroll_amount-1)

			// If our last cell we're shifting is wide, then we need to clear
			// it to be empty so we don't split the multi-cell char.
			end := x
			if page.cells[end].wide == PageCellWideWide {
				end_multi := end
				assert(page.cells[end_multi+1].wide == PageCellWideSpacerTail)
				self.screens.active.ClearCells(
					page,
					self.screens.active.cursor.page_row,
					page.cells[end_multi:end_multi+2],
				)
			}

			// We work backwards so we don't overwrite data.
			for ; x >= int(left); x -= 1 {
				src := &page.cells[x]
				dst := &page.cells[x+int(adjusted_count)]
				page.SwapCells(src, dst)
			}
		}()
	}

	// Insert blanks. The blanks preserve the background color.
	self.screens.active.ClearCells(page, self.screens.active.cursor.page_row, page.cells[left:left+uint(adjusted_count)])

	// Our row is always dirty
	self.screens.active.CursorMarkDirty()
}

// DeleteChars is Terminal.deleteChars.
//
// Removes amount characters from the current cursor position to the right.
// The remaining characters are shifted to the left and space from the right
// margin is filled with spaces.
//
// If amount is greater than the remaining number of characters in the
// scrolling region, it is adjusted down.
//
// Does not change the cursor position.
//
// Zig: pub fn deleteChars(self: *Terminal, count_req: usize) void.
func (self *Terminal) DeleteChars(count_req uint) {
	if count_req == 0 {
		return
	}

	// If our cursor is outside the margins then do nothing. We DO reset
	// wrap state still so this must remain below the above logic.
	if self.screens.active.cursor.x < self.scrolling_region.left ||
		self.screens.active.cursor.x > self.scrolling_region.right {
		return
	}

	// left is just the cursor position but as a multi-pointer
	//
	// Go: the index of the cursor cell in the page's cells.
	page := &self.screens.active.cursor.page_pin.node.data
	left := page.cellIndex(self.screens.active.cursor.page_cell)

	// Remaining cols from our cursor to the right margin.
	rem := self.scrolling_region.right - self.screens.active.cursor.x + 1

	// We can only insert blanks up to our remaining cols
	//
	// Zig: @min(usize, CellCountInt) has type CellCountInt.
	count := SizeCellCountInt(min(count_req, uint(rem)))

	self.screens.active.SplitCellBoundary(self.screens.active.cursor.x)
	self.screens.active.SplitCellBoundary(self.screens.active.cursor.x + count)
	self.screens.active.SplitCellBoundary(self.scrolling_region.right + 1)

	// This is the amount of space at the right of the scroll region
	// that will NOT be blank, so we need to shift the correct cols right.
	// "scroll_amount" is the number of such cols.
	scroll_amount := rem - count
	x := left
	if scroll_amount > 0 {
		func() {
			// Zig: a block-scoped defer; the closure runs it at the end of
			// the block.
			page.PauseIntegrityChecks(true)
			defer page.PauseIntegrityChecks(false)

			right := left + uint(scroll_amount-1)

			for ; x <= right; x += 1 {
				src := &page.cells[x+uint(count)]
				dst := &page.cells[x]
				page.SwapCells(src, dst)
			}
		}()
	}

	// Insert blanks. The blanks preserve the background color.
	self.screens.active.ClearCells(page, self.screens.active.cursor.page_row, page.cells[x:x+uint(rem-scroll_amount)])

	// Our row's soft-wrap is always reset.
	self.screens.active.CursorResetWrap()

	// Our row is always dirty
	self.screens.active.CursorMarkDirty()
}

// EraseChars is Terminal.eraseChars.
//
// Zig: pub fn eraseChars(self: *Terminal, count_req: usize) void.
func (self *Terminal) EraseChars(count_req uint) {
	var count SizeCellCountInt
	{
		// Zig: the labeled block `end:`.
		remaining := self.cols - self.screens.active.cursor.x
		// Zig: @min(CellCountInt, usize) has type CellCountInt.
		end := SizeCellCountInt(min(uint(remaining), max(count_req, 1)))

		// If our last cell is a wide char then we need to also clear the
		// cell beyond it since we can't just split a wide char.
		if end != remaining {
			last := self.screens.active.CursorCellRight(end - 1)
			if last.wide == PageCellWideWide {
				end += 1
			}
		}

		count = end
	}

	// Handle any boundary conditions on the edges of the erased area.
	//
	// TODO(qwerasd): This isn't actually correct if you take in to account
	// protected modes. We need to figure out how to make `clearCells` or at
	// least `clearUnprotectedCells` handle boundary conditions...
	self.screens.active.SplitCellBoundary(self.screens.active.cursor.x)
	self.screens.active.SplitCellBoundary(self.screens.active.cursor.x + count)

	// Reset our row's soft-wrap.
	self.screens.active.CursorResetWrap()

	// Mark our cursor row as dirty
	self.screens.active.CursorMarkDirty()

	// Clear the cells
	//
	// Go: the index of the cursor cell in the page's cells.
	page := &self.screens.active.cursor.page_pin.node.data
	cells := page.cellIndex(self.screens.active.cursor.page_cell)

	// If we never had a protection mode, then we can assume no cells
	// are protected and go with the fast path. If the last protection
	// mode was not ISO we also always ignore protection attributes.
	if self.screens.active.protected_mode != ANSIProtectedModeIso {
		self.screens.active.ClearCells(
			page,
			self.screens.active.cursor.page_row,
			page.cells[cells:cells+uint(count)],
		)
		return
	}

	self.screens.active.ClearUnprotectedCells(
		page,
		self.screens.active.cursor.page_row,
		page.cells[cells:cells+uint(count)],
	)
}

// EraseLine is Terminal.eraseLine.
//
// Erase the line.
//
// Zig: pub fn eraseLine(self: *Terminal, mode: csi.EraseLine,
// protected_req: bool) void.
func (self *Terminal) EraseLine(mode CSIEraseLine, protected_req bool) {
	// Get our start/end positions depending on mode.
	var start, end SizeCellCountInt
	switch mode {
	case CSIEraseLineRight:
		// Zig: the labeled block `right:`.
		x := self.screens.active.cursor.x

		// If our X is a wide spacer tail then we need to erase the
		// previous cell too so we don't split a multi-cell character.
		if x > 0 && self.screens.active.cursor.page_cell.wide == PageCellWideSpacerTail {
			x -= 1
		}

		// Reset our row's soft-wrap.
		self.screens.active.CursorResetWrap()

		start, end = x, self.cols

	case CSIEraseLineLeft:
		// Zig: the labeled block `left:`.
		x := self.screens.active.cursor.x

		// If our x is a wide char we need to delete the tail too.
		if self.screens.active.cursor.page_cell.wide == PageCellWideWide {
			x += 1
		}

		start, end = 0, x+1

	// Note that it seems like complete should reset the soft-wrap
	// state of the line but in xterm it does not.
	case CSIEraseLineComplete:
		start, end = 0, self.cols

	default:
		warn("terminal", "unimplemented erase line mode: %v", mode) // Zig: log.err
		return
	}

	// All modes will clear the pending wrap state and we know we have
	// a valid mode at this point.
	self.screens.active.cursor.pending_wrap = false

	// We always mark our row as dirty
	self.screens.active.CursorMarkDirty()

	// Start of our cells
	//
	// Go: the index in the page's cells of the cursor row's first cell.
	page := &self.screens.active.cursor.page_pin.node.data
	var cells uint
	{
		// Zig: the labeled block `cells:`.
		cells_ := page.cellIndex(self.screens.active.cursor.page_cell)
		cells = cells_ - uint(self.screens.active.cursor.x)
	}

	// We respect protected attributes if explicitly requested (probably
	// a DECSEL sequence) or if our last protected mode was ISO even if its
	// not currently set.
	protected := self.screens.active.protected_mode == ANSIProtectedModeIso || protected_req

	// If we're not respecting protected attributes, we can use a fast-path
	// to fill the entire line.
	if !protected {
		self.screens.active.ClearCells(
			page,
			self.screens.active.cursor.page_row,
			page.cells[cells+uint(start):cells+uint(end)],
		)
		return
	}

	self.screens.active.ClearUnprotectedCells(
		page,
		self.screens.active.cursor.page_row,
		page.cells[cells+uint(start):cells+uint(end)],
	)
}

// EraseDisplay is Terminal.eraseDisplay.
//
// Erase the display.
//
// Zig: pub fn eraseDisplay(self: *Terminal, mode: csi.EraseDisplay,
// protected_req: bool) void.
func (self *Terminal) EraseDisplay(mode CSIEraseDisplay, protected_req bool) {
	// We respect protected attributes if explicitly requested (probably
	// a DECSEL sequence) or if our last protected mode was ISO even if its
	// not currently set.
	protected := self.screens.active.protected_mode == ANSIProtectedModeIso || protected_req

	switch mode {
	case CSIEraseDisplayScrollComplete:
		if err := self.screens.active.ScrollClear(); err != nil {
			warn("terminal", "scroll clear failed, doing a normal clear err=error.%v", err)
			self.EraseDisplay(CSIEraseDisplayComplete, protected_req)
			return
		}

		// Unsets pending wrap state
		self.screens.active.cursor.pending_wrap = false

		// Zig: clearing all Kitty graphics state for this screen is under
		// `build_options.kitty_graphics` (compiled out).

	case CSIEraseDisplayComplete:
		// If we're on the primary screen and our last non-empty row is
		// a prompt, then we do a scroll_complete instead. This is a
		// heuristic to get the generally desirable behavior that ^L
		// at a prompt scrolls the screen contents prior to clearing.
		// Most shells send `ESC [ H ESC [ 2 J` so we can't just check
		// our current cursor position. See #905
		if self.screens.active_key == ScreenSetKeyPrimary {
			func() {
				// Zig: the labeled block `at_prompt:`, followed by
				// nothing; `break :at_prompt` is `return`.

				// Go from the bottom of the active up and see if we're
				// at a prompt.
				active_br_ := self.screens.active.pages.GetBottomRight(
					PointTagActive,
				)
				if !active_br_.Valid {
					return
				}
				active_br := active_br_.Value
				it := active_br.RowIterator(
					PageListDirectionLeftUp,
					Some(self.screens.active.pages.GetTopLeft(PointTagActive)),
				)
			loop:
				for {
					p := it.Next()
					if !p.Valid {
						// Zig: the `else` of the while: break :at_prompt.
						return
					}
					row := p.Value.RowAndCell().row
					switch row.semantic_prompt {
					// If we're at a prompt or input area, then we are at a prompt.
					case PageRowSemanticPromptPrompt,
						PageRowSemanticPromptPromptContinuation:
						break loop

					// If we have command output, then we're most certainly not
					// at a prompt.
					case PageRowSemanticPromptNone:
						return

					default:
						panic("unreachable")
					}
				}

				if err := self.screens.active.ScrollClear(); err != nil {
					// If we fail, we just fall back to doing a normal clear
					// so we don't worry about the error.
				}
			}()
		}

		// All active area
		self.screens.active.ClearRows(
			Point{Tag: PointTagActive},
			None[Point](),
			protected,
		)

		// Unsets pending wrap state
		self.screens.active.cursor.pending_wrap = false

		// Zig: clearing all Kitty graphics state for this screen is under
		// `build_options.kitty_graphics` (compiled out).

		// Cleared screen dirty bit
		self.flags.dirty.clear = true

	case CSIEraseDisplayBelow:
		// All lines to the right (including the cursor)
		self.EraseLine(CSIEraseLineRight, protected_req)

		// All lines below
		if self.screens.active.cursor.y+1 < self.rows {
			self.screens.active.ClearRows(
				Point{Tag: PointTagActive, Active: PointCoordinate{y: uint32(self.screens.active.cursor.y + 1)}},
				None[Point](),
				protected,
			)
		}

		// Unsets pending wrap state. Should be done by eraseLine.
		assert(!self.screens.active.cursor.pending_wrap)

	case CSIEraseDisplayAbove:
		// Erase to the left (including the cursor)
		self.EraseLine(CSIEraseLineLeft, protected_req)

		// All lines above
		if self.screens.active.cursor.y > 0 {
			self.screens.active.ClearRows(
				Point{Tag: PointTagActive, Active: PointCoordinate{y: 0}},
				Some(Point{Tag: PointTagActive, Active: PointCoordinate{y: uint32(self.screens.active.cursor.y - 1)}}),
				protected,
			)
		}

		// Unsets pending wrap state
		assert(!self.screens.active.cursor.pending_wrap)

	case CSIEraseDisplayScrollback:
		self.screens.active.EraseRows(Point{Tag: PointTagHistory}, None[Point]())

	default:
		panic("unreachable")
	}
}

// Decaln is Terminal.decaln.
//
// Resets all margins and fills the whole screen with the character 'E'
//
// Sets the cursor to the top left corner.
//
// Zig: pub fn decaln(self: *Terminal) !void. The inferred error set is that
// of Screen.manualStyleUpdate: ErrOutOfMemory (dcs.go) or ErrOutOfSpace
// (PageList.go).
func (self *Terminal) Decaln() (err error) {
	// Clear our stylistic attributes. This is the only thing that can
	// fail so we do it first so we can undo it.
	old_style := self.screens.active.cursor.style
	self.screens.active.cursor.style = Style{
		bg_color: self.screens.active.cursor.style.bg_color,
		fg_color: self.screens.active.cursor.style.fg_color,
	}
	// Zig: errdefer self.screens.active.cursor.style = old_style;
	defer func() {
		if err != nil {
			self.screens.active.cursor.style = old_style
		}
	}()
	if err = self.screens.active.ManualStyleUpdate(); err != nil {
		return err
	}

	// Reset margins, also sets cursor to top-left
	self.scrolling_region = TerminalScrollingRegion{
		top:    0,
		bottom: self.rows - 1,
		left:   0,
		right:  self.cols - 1,
	}

	// Origin mode is disabled
	self.modes.Set(ModesModeOrigin, false)

	// Move our cursor to the top-left
	self.SetCursorPos(1, 1)

	// Use clearRows instead of eraseDisplay because we must NOT respect
	// protected attributes here.
	self.screens.active.ClearRows(
		Point{Tag: PointTagActive},
		None[Point](),
		false,
	)

	// Fill with Es by moving the cursor but reset it after.
	for {
		page := &self.screens.active.cursor.page_pin.node.data
		row := self.screens.active.cursor.page_row
		cells_multi := page.rowCells(row)
		cells := cells_multi[0:page.size.cols]
		for i := range cells { // Zig: @memset
			cells[i] = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 'E'},
				style_id:    self.screens.active.cursor.style_id,

				// DECALN does not respect protected state. Verified with xterm.
				protected: false,
			}
		}

		// If we have a ref-counted style, increase
		if self.screens.active.cursor.style_id != StyleDefaultId {
			page.styles.UseMultiple(
				self.screens.active.cursor.style_id,
				SizeCellCountInt(len(cells)),
			)
			row.styled = true
		}

		// We messed with the page so assert its integrity here.
		page.AssertIntegrity()

		self.screens.active.CursorMarkDirty()
		if self.screens.active.cursor.y == self.rows-1 {
			break
		}
		self.screens.active.CursorDown(1)
	}

	// Reset the cursor to the top-left
	self.SetCursorPos(1, 1)
	return nil
}

// Terminal.kittyGraphics (line 2681) is omitted; see the file header.

// SetAttribute is Terminal.setAttribute.
//
// Set a style attribute.
//
// Zig: pub fn setAttribute(self: *Terminal, attr: sgr.Attribute) !void. The
// inferred error set is that of Screen.setAttribute: ErrOutOfMemory
// (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Terminal) SetAttribute(attr SGRAttribute) error {
	return self.screens.active.SetAttribute(attr)
}

// PrintAttributes is Terminal.printAttributes.
//
// Print the active attributes as a string. This is used to respond to DECRQSS
// requests.
//
// Boolean attributes are printed first, followed by foreground color, then
// background color. Each attribute is separated by a semicolon.
//
// Zig: pub fn printAttributes(self: *Terminal, buf: []u8) ![]const u8. The
// result is the written part of `buf`. The inferred error set is that of a
// std.io.fixedBufferStream writer: ErrNoSpaceLeft (color.go).
func (self *Terminal) PrintAttributes(buf []uint8) ([]uint8, error) {
	// Zig: `var stream = std.io.fixedBufferStream(buf); const writer =
	// stream.writer();`. Go: the write position `pos` and `writeAll`, which
	// copies what fits and fails with ErrNoSpaceLeft when not everything
	// does, as the fixed buffer stream's writeAll. `print` formats first
	// and then writes, which leaves the same bytes in buf.
	var pos uint
	writeAll := func(bytes []uint8) error {
		n := uint(copy(buf[pos:], bytes))
		pos += n
		if n < uint(len(bytes)) {
			return ErrNoSpaceLeft
		}
		return nil
	}

	// The SGR response always starts with a 0. See https://vt100.net/docs/vt510-rm/DECRPSS
	if err := writeAll([]uint8{'0'}); err != nil { // Zig: writer.writeByte('0')
		return nil, err
	}

	pen := self.screens.active.cursor.style
	var attrs [8]uint8 // Zig: @splat(0)
	var i uint = 0

	if pen.flags.bold {
		attrs[i] = '1'
		i += 1
	}

	if pen.flags.faint {
		attrs[i] = '2'
		i += 1
	}

	if pen.flags.italic {
		attrs[i] = '3'
		i += 1
	}

	if pen.flags.underline != SGRAttributeUnderlineNone {
		attrs[i] = '4'
		i += 1
	}

	if pen.flags.blink {
		attrs[i] = '5'
		i += 1
	}

	if pen.flags.inverse {
		attrs[i] = '7'
		i += 1
	}

	if pen.flags.invisible {
		attrs[i] = '8'
		i += 1
	}

	if pen.flags.strikethrough {
		attrs[i] = '9'
		i += 1
	}

	for _, c := range attrs[0:i] {
		if err := writeAll([]uint8{';', c}); err != nil { // Zig: writer.print(";{c}", .{c})
			return nil, err
		}
	}

	switch pen.fg_color.Tag {
	case StyleColorNone:
	case StyleColorPalette:
		idx := pen.fg_color.Palette
		var s string
		if idx >= 16 {
			s = ";38:5:" + itoa(uint64(idx))
		} else if idx >= 8 {
			s = ";9" + itoa(uint64(idx-8))
		} else {
			s = ";3" + itoa(uint64(idx))
		}
		if err := writeAll([]uint8(s)); err != nil {
			return nil, err
		}
	case StyleColorRgb:
		rgb := pen.fg_color.Rgb
		s := ";38:2::" + itoa(uint64(rgb.r)) + ":" + itoa(uint64(rgb.g)) + ":" + itoa(uint64(rgb.b))
		if err := writeAll([]uint8(s)); err != nil {
			return nil, err
		}
	default:
		panic("unreachable")
	}

	switch pen.bg_color.Tag {
	case StyleColorNone:
	case StyleColorPalette:
		idx := pen.bg_color.Palette
		var s string
		if idx >= 16 {
			s = ";48:5:" + itoa(uint64(idx))
		} else if idx >= 8 {
			s = ";10" + itoa(uint64(idx-8))
		} else {
			s = ";4" + itoa(uint64(idx))
		}
		if err := writeAll([]uint8(s)); err != nil {
			return nil, err
		}
	case StyleColorRgb:
		rgb := pen.bg_color.Rgb
		s := ";48:2::" + itoa(uint64(rgb.r)) + ":" + itoa(uint64(rgb.g)) + ":" + itoa(uint64(rgb.b))
		if err := writeAll([]uint8(s)); err != nil {
			return nil, err
		}
	default:
		panic("unreachable")
	}

	return buf[:pos], nil // Zig: stream.getWritten()
}

// TerminalDeccolmMode is Terminal.DeccolmMode.
//
// The modes for DECCOLM.
//
// Zig: enum(u1) { @"80_cols" = 0, @"132_cols" = 1 }.
type TerminalDeccolmMode uint8

const (
	TerminalDeccolmMode80Cols  TerminalDeccolmMode = 0
	TerminalDeccolmMode132Cols TerminalDeccolmMode = 1
)

// terminalDeccolmModeNames are the tag names of TerminalDeccolmMode, by
// value.
var terminalDeccolmModeNames = [...]string{"80_cols", "132_cols"}

// String returns the Zig tag name (@tagName).
func (self TerminalDeccolmMode) String() string { return terminalDeccolmModeNames[self] }

// Deccolm is Terminal.deccolm.
//
// DECCOLM changes the terminal width between 80 and 132 columns. This
// function call will do NOTHING unless `setDeccolmSupported` has been
// called with "true".
//
// This breaks the expectation around modern terminals that they resize
// with the window. This will fix the grid at either 80 or 132 columns.
// The rows will continue to be variable.
//
// Zig: pub fn deccolm(self: *Terminal, alloc: Allocator, mode: DeccolmMode)
// !void. The allocator parameter is dropped. The inferred error set is that
// of resize.
func (self *Terminal) Deccolm(mode TerminalDeccolmMode) error {
	// If DEC mode 40 isn't enabled, then this is ignored. We also make
	// sure that we don't have deccolm set because we want to fully ignore
	// set mode.
	if !self.modes.Get(ModesModeEnableMode3) {
		self.modes.Set(ModesMode132Column, false)
		return nil
	}

	// Enable it
	self.modes.Set(ModesMode132Column, mode == TerminalDeccolmMode132Cols)

	// Resize to the requested size
	var cols SizeCellCountInt
	switch mode {
	case TerminalDeccolmMode132Cols:
		cols = 132
	case TerminalDeccolmMode80Cols:
		cols = 80
	default:
		panic("unreachable")
	}
	if err := self.Resize(cols, self.rows); err != nil {
		return err
	}

	// Erase our display and move our cursor.
	self.EraseDisplay(CSIEraseDisplayComplete, false)
	self.SetCursorPos(1, 1)
	return nil
}

// Resize is Terminal.resize.
//
// Resize the underlying terminal.
//
// Zig: pub fn resize(self: *Terminal, alloc: Allocator,
// cols: size.CellCountInt, rows: size.CellCountInt) !void. The allocator
// parameter is dropped. The inferred error set is that of Tabstops.init and
// Screen.resize: ErrOutOfMemory (dcs.go).
func (self *Terminal) Resize(cols SizeCellCountInt, rows SizeCellCountInt) error {
	// If our cols/rows didn't change then we're done
	if self.cols == cols && self.rows == rows {
		return nil
	}

	// Resize our tabstops
	if self.cols != cols {
		self.tabstops.Deinit()
		tabstops, err := TabstopsInit(uint(cols), 8)
		if err != nil {
			return err
		}
		self.tabstops = tabstops
	}

	// Resize primary screen, which supports reflow
	primary := self.screens.Get(ScreenSetKeyPrimary)
	assert(primary != nil) // .?
	if err := primary.Resize(ScreenResize{
		cols:          cols,
		rows:          rows,
		reflow:        self.modes.Get(ModesModeWraparound),
		prompt_redraw: self.flags.shell_redraws_prompt,
	}); err != nil {
		return err
	}

	// Alternate screen, if it exists, doesn't reflow
	if alt := self.screens.Get(ScreenSetKeyAlternate); alt != nil {
		if err := alt.Resize(ScreenResize{
			cols:   cols,
			rows:   rows,
			reflow: false,

			// Zig: the field default, which is not the Go zero value.
			prompt_redraw: OSCParsersSemanticPromptRedrawFalse,
		}); err != nil {
			return err
		}
	}

	// Whenever we resize we just mark it as a screen clear
	self.flags.dirty.clear = true

	// Set our size
	self.cols = cols
	self.rows = rows

	// Reset the scrolling region
	//
	// Zig: `rows - 1` and `cols - 1` (u16) trap below 1.
	assert(rows >= 1)
	assert(cols >= 1)
	self.scrolling_region = TerminalScrollingRegion{
		top:    0,
		bottom: rows - 1,
		left:   0,
		right:  cols - 1,
	}
	return nil
}

// SetPwd is Terminal.setPwd.
//
// Set the pwd for the terminal.
//
// Zig: pub fn setPwd(self: *Terminal, pwd: []const u8) !void. The error is
// ErrOutOfMemory (dcs.go).
func (self *Terminal) SetPwd(pwd []uint8) error {
	self.pwd = self.pwd[:0] // Zig: clearRetainingCapacity
	// Zig: appendSlice copies with @memcpy into items[0..pwd.len]; a safe
	// build stops when the two ranges overlap ("@memcpy arguments alias"),
	// e.g. t.setPwd(t.getPwd().?). Go's append would copy anyway, so the
	// check is explicit. When the capacity is too small Zig grows the list
	// first and the ranges cannot overlap.
	if len(pwd) > 0 && cap(self.pwd) >= len(pwd) {
		dst := reflect.ValueOf(self.pwd[:len(pwd)]).Pointer()
		src := reflect.ValueOf(pwd).Pointer()
		n := uintptr(len(pwd))
		if src < dst+n && dst < src+n {
			panic("@memcpy arguments alias")
		}
	}
	self.pwd = append(self.pwd, pwd...) // Zig: appendSlice(self.gpa(), pwd)
	return nil
}

// GetPwd is Terminal.getPwd.
//
// Returns the pwd for the terminal, if any. The memory is owned by the
// Terminal and is not copied. It is safe until a reset or setPwd.
//
// Zig: pub fn getPwd(self: *const Terminal) ?[]const u8.
func (self *Terminal) GetPwd() Optional[[]uint8] {
	if len(self.pwd) == 0 {
		return None[[]uint8]()
	}
	return Some(self.pwd)
}

// SwitchScreen is Terminal.switchScreen.
//
// Switch to the given screen type (alternate or primary).
//
// This does NOT handle behaviors such as clearing the screen,
// copying the cursor, etc. This should be handled by downstream
// callers.
//
// After calling this function, the `self.screen` field will point
// to the current screen, and the returned value will be the previous
// screen. If the return value is null, then the screen was not
// switched because it was already the active screen.
//
// Note: This is written in a generic way so that we can support
// more than two screens in the future if needed. There isn't
// currently a spec for this, but it is something I think might
// be useful in the future.
//
// Zig: pub fn switchScreen(self: *Terminal, key: ScreenSet.Key) !?*Screen.
// Null is nil. The inferred error set is that of ScreenSet.getInit:
// ErrOutOfMemory (dcs.go).
func (self *Terminal) SwitchScreen(key ScreenSetKey) (*Screen, error) {
	// If we're already on the requested screen we do nothing.
	if self.screens.active_key == key {
		return nil, nil
	}
	old := self.screens.active

	// We always end hyperlink state when switching screens.
	// We need to do this on the original screen.
	old.EndHyperlink()

	// Switch the screens/
	new_ := self.screens.Get(key) // Zig: new
	if new_ == nil {
		// Zig: the labeled block `new:`.
		primary := self.screens.Get(ScreenSetKeyPrimary)
		assert(primary != nil) // .?
		var max_scrollback uint64
		switch key {
		case ScreenSetKeyPrimary:
			max_scrollback = primary.pages.explicit_max_size
		case ScreenSetKeyAlternate:
			max_scrollback = 0
		default:
			panic("unreachable")
		}
		s, err := self.screens.GetInit(
			key,
			ScreenOptions{
				cols:           self.cols,
				rows:           self.rows,
				max_scrollback: max_scrollback,

				// Inherit our Kitty image storage limit from the primary
				// screen if we have to initialize.
				//
				// Zig: `primary.kitty_images.total_limit` under
				// `build_options.kitty_graphics` (compiled out), else 0.
				kitty_image_storage_limit: 0,
			},
		)
		if err != nil {
			return nil, err
		}
		new_ = s
	}

	// The new screen should not have any hyperlinks set
	assert(new_.cursor.hyperlink_id == 0)

	// Bring our charset state with us
	new_.charset = old.charset

	// Clear our selection
	new_.ClearSelection()

	// Zig: `new.kitty_images.dirty = true` under
	// `build_options.kitty_graphics` (compiled out).

	// Mark our terminal as dirty to redraw the grid.
	self.flags.dirty.clear = true

	// Finalize the switch
	self.screens.SwitchTo(key)

	return old, nil
}

// SwitchScreenMode is Terminal.switchScreenMode.
//
// Switch screen via a mode switch (e.g. mode 47, 1047, 1049).
// This is a much more opinionated operation than `switchScreen`
// since it also handles the behaviors of the specific mode,
// such as clearing the screen, saving/restoring the cursor,
// etc.
//
// This should be used for legacy compatibility with VT protocols,
// but more modern usage should use `switchScreen` instead and handle
// details like clearing the screen, cursor saving, etc. manually.
//
// Zig: pub fn switchScreenMode(self: *Terminal, mode: SwitchScreenMode,
// enabled: bool) !void. The inferred error set is that of switchScreen.
func (self *Terminal) SwitchScreenMode(mode TerminalSwitchScreenMode, enabled bool) error {
	// The behavior in this function is completely based on reading
	// the xterm source, specifically "charproc.c" for
	// `srm_ALTBUF`, `srm_OPT_ALTBUF`, and `srm_OPT_ALTBUF_CURSOR`.
	// We shouldn't touch anything in here without adding a unit
	// test AND verifying the behavior with xterm.

	switch mode {
	case TerminalSwitchScreenMode47:

	// If we're disabling 1047 and we're on alt screen then
	// we clear the screen.
	case TerminalSwitchScreenMode1047:
		if !enabled && self.screens.active_key == ScreenSetKeyAlternate {
			self.EraseDisplay(CSIEraseDisplayComplete, false)
		}

	// 1049 unconditionally saves the cursor on enabling, even
	// if we're already on the alternate screen.
	case TerminalSwitchScreenMode1049:
		if enabled {
			self.SaveCursor()
		}

	default:
		panic("unreachable")
	}

	// Switch screens first to whatever we're going to.
	var to ScreenSetKey
	if enabled {
		to = ScreenSetKeyAlternate
	} else {
		to = ScreenSetKeyPrimary
	}
	old_, err := self.SwitchScreen(to)
	if err != nil {
		return err
	}

	switch mode {
	// For these modes, we need to copy the cursor. We only copy
	// the cursor if the screen actually changed, otherwise the
	// cursor is already copied. The cursor is copied regardless
	// of destination screen.
	case TerminalSwitchScreenMode47, TerminalSwitchScreenMode1047:
		if old := old_; old != nil {
			if err := self.screens.active.CursorCopy(old.cursor, ScreenCursorCopyOptions{
				hyperlink: false,
			}); err != nil {
				warn(
					"terminal",
					"cursor copy failed entering alt screen err=error.%v",
					err,
				)
			}
		}

	// Mode 1049 restores cursor on the primary screen when
	// we disable it.
	case TerminalSwitchScreenMode1049:
		if enabled {
			assert(self.screens.active_key == ScreenSetKeyAlternate)
			self.EraseDisplay(CSIEraseDisplayComplete, false)

			// When we enter alt screen with 1049, we always copy the
			// cursor from the primary screen (if we weren't already
			// on it).
			if old := old_; old != nil {
				if err := self.screens.active.CursorCopy(old.cursor, ScreenCursorCopyOptions{
					hyperlink: false,
				}); err != nil {
					warn(
						"terminal",
						"cursor copy failed entering alt screen err=error.%v",
						err,
					)
				}
			}
		} else {
			assert(self.screens.active_key == ScreenSetKeyPrimary)
			self.RestoreCursor()
		}

	default:
		panic("unreachable")
	}
	return nil
}

// TerminalSwitchScreenMode is Terminal.SwitchScreenMode.
//
// Modal screen changes. These map to the literal terminal
// modes to enable or disable alternate screen modes. They each
// have subtle behaviors so we define them as an enum here.
//
// Zig: enum { @"47", @"1047", @"1049" }, values numbered from zero in the
// order listed.
type TerminalSwitchScreenMode uint8

const (
	// Legacy alternate screen mode. This goes to the alternate
	// screen or primary screen and only copies the cursor. The
	// screen is not erased.
	TerminalSwitchScreenMode47 TerminalSwitchScreenMode = iota

	// Alternate screen mode where the alternate screen is cleared
	// on exit. The primary screen is never cleared. The cursor is
	// copied.
	TerminalSwitchScreenMode1047

	// Save primary screen cursor, switch to alternate screen,
	// and clear the alternate screen on entry. On exit,
	// do not clear the screen, and restore the cursor on the
	// primary screen.
	TerminalSwitchScreenMode1049
)

// terminalSwitchScreenModeNames are the tag names of
// TerminalSwitchScreenMode, in order.
var terminalSwitchScreenModeNames = [...]string{"47", "1047", "1049"}

// String returns the Zig tag name (@tagName).
func (self TerminalSwitchScreenMode) String() string {
	return terminalSwitchScreenModeNames[self]
}

// PlainString is Terminal.plainString.
//
// Return the current string value of the terminal. Newlines are
// encoded as "\n". This omits any formatting such as fg/bg.
//
// The caller must free the string.
//
// Zig: pub fn plainString(self: *Terminal, alloc: Allocator) ![]const u8.
// The allocator parameter is dropped. The inferred error set is that of
// Screen.dumpStringAlloc: ErrUnknownPoint (Screen.go), ErrWriteFailed
// (osc.go) and ErrOutOfMemory (dcs.go).
func (self *Terminal) PlainString() ([]uint8, error) {
	return self.screens.active.DumpStringAlloc(Point{Tag: PointTagViewport})
}

// PlainStringUnwrapped is Terminal.plainStringUnwrapped.
//
// Same as plainString, but respects row wrap state when building the string.
//
// Zig: pub fn plainStringUnwrapped(self: *Terminal, alloc: Allocator)
// ![]const u8. The allocator parameter is dropped. The inferred error set
// is that of Screen.dumpStringAllocUnwrapped: ErrUnknownPoint (Screen.go),
// ErrWriteFailed (osc.go) and ErrOutOfMemory (dcs.go).
func (self *Terminal) PlainStringUnwrapped() ([]uint8, error) {
	return self.screens.active.DumpStringAllocUnwrapped(Point{Tag: PointTagViewport})
}

// FullReset is Terminal.fullReset.
//
// Full reset.
//
// This will attempt to free the existing screen memory but if that fails
// this will reuse the existing memory. In the latter case, memory may
// be wasted (since its unused) but it isn't leaked.
func (self *Terminal) FullReset() {
	// Ensure we're back on primary screen
	self.screens.SwitchTo(ScreenSetKeyPrimary)
	self.screens.Remove(
		ScreenSetKeyAlternate,
	)

	// Reset our screens
	self.screens.active.Reset()

	// Rest our basic state
	self.modes.Reset()
	self.flags = TerminalFlags{focused: true} // Zig: .{}
	self.tabstops.Reset(terminalTabstopInterval)
	self.previous_char = None[uint32]()
	self.pwd = self.pwd[:0] // Zig: clearRetainingCapacity
	self.status_display = ANSIStatusDisplayMain
	self.scrolling_region = TerminalScrollingRegion{
		top:    0,
		bottom: self.rows - 1,
		left:   0,
		right:  self.cols - 1,
	}

	// Always mark dirty so we redraw everything
	self.flags.dirty.clear = true
}

// isDirty is Terminal.isDirty.
//
// Returns true if the point is dirty, used for testing.
//
// Zig: fn isDirty(t: *const Terminal, pt: point.Point) bool.
func (t *Terminal) isDirty(pt Point) bool {
	cell := t.screens.active.pages.GetCell(pt)
	assert(cell.Valid) // .?
	return cell.Value.IsDirty()
}

// clearDirty is Terminal.clearDirty.
//
// Clear all dirty bits. Testing only.
//
// Zig: fn clearDirty(t: *Terminal) void.
func (t *Terminal) clearDirty() {
	t.screens.active.pages.ClearDirty()
}
