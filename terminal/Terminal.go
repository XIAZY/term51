// WIREFRAME for src/terminal/Terminal.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// The primary terminal emulation structure. This represents a single
// "terminal" containing a grid of characters and exposes various operations
// on that grid. This also maintains the scrollback buffer.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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
// word size, counts and indexes are `uint` (Zig `usize`), `isize` is `int`,
// and fixed Zig widths stay fixed. Byte quantity: `Options.max_scrollback`
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
//   - `print` (line 302) reads `unicode.table` (width, `emoji_vs_base`,
//     `width_zero_in_grapheme`, `grapheme_break`) and calls
//     `unicode.graphemeBreak` with a `uucode.grapheme.BreakState`
//     (src/unicode and the uucode dependency);
//   - `plainString` (line 3058) and `plainStringUnwrapped` (line 3063) call
//     Screen.dumpStringAlloc and Screen.dumpStringAllocUnwrapped, which are
//     written with `ScreenFormatter` (formatter.zig).
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

// String returns the Zig tag name (@tagName).
func (self TerminalFlagsMouseShiftCapture) String() string { panic(notImplemented) }

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
// ColorDynamicPaletteDefault. Wireframe: left zero, as
// ColorDynamicPaletteDefault is in color.go; the implementation initialises
// it once at start-up.
var TerminalColorsDefault TerminalColors

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

// String returns the Zig tag name (@tagName).
func (self TerminalMouseEvents) String() string { panic(notImplemented) }

// Motion is Terminal.MouseEvents.motion.
//
// Returns true if this event sends motion events.
func (self TerminalMouseEvents) Motion() bool { panic(notImplemented) }

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

// String returns the Zig tag name (@tagName).
func (self TerminalMouseFormat) String() string { panic(notImplemented) }

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
func TerminalInit(opts TerminalOptions) (Terminal, error) { panic(notImplemented) }

// Deinit is Terminal.deinit.
//
// Zig: pub fn deinit(self: *Terminal, alloc: Allocator) void. The allocator
// parameter is dropped. Kept: it deinitialises the tabstops and the
// screens, and every test calls it.
func (self *Terminal) Deinit() { panic(notImplemented) }

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
func (self *Terminal) VtStream() StreamReadonlyStream { panic(notImplemented) }

// VtHandler is Terminal.vtHandler.
//
// This is the handler-side only for vtStream.
//
// Zig: pub fn vtHandler(self: *Terminal) ReadonlyHandler.
func (self *Terminal) VtHandler() StreamReadonlyHandler { panic(notImplemented) }

// gpa is Terminal.gpa.
//
// The general allocator we should use for this terminal.
//
// Zig: fn gpa(self: *Terminal) Allocator, returning
// `self.screens.active.alloc`. Go has no allocator and Screen has no
// `alloc` field, so the stub returns nothing; it is kept because a project
// test calls it. See the file header.
func (self *Terminal) gpa() { panic(notImplemented) }

// PrintString is Terminal.printString.
//
// Print UTF-8 encoded string to the terminal.
//
// Zig: pub fn printString(self: *Terminal, str: []const u8) !void. The
// inferred error set is ErrInvalidUtf8 (Screen.go; std.unicode.Utf8View.init)
// and those of linefeed and print.
func (self *Terminal) PrintString(str []uint8) error { panic(notImplemented) }

// PrintRepeat is Terminal.printRepeat.
//
// Print the previous printed character a repeated amount of times.
//
// Zig: pub fn printRepeat(self: *Terminal, count_req: usize) !void. The
// inferred error set is that of print.
func (self *Terminal) PrintRepeat(count_req uint) error { panic(notImplemented) }

// Print is Terminal.print.
//
// Zig: pub fn print(self: *Terminal, c: u21) !void. The inferred error set
// is that of printWrap and Screen.appendGrapheme: ErrOutOfMemory (dcs.go),
// ErrOutOfSpace (PageList.go) and Page.CloneFromError (page.go).
func (self *Terminal) Print(c uint32) error { panic(notImplemented) }

// printCell is Terminal.printCell.
//
// Zig: fn printCell(self: *Terminal, unmapped_c: u21, wide: Cell.Wide)
// void.
func (self *Terminal) printCell(unmapped_c uint32, wide PageCellWide) {
	panic(notImplemented)
}

// printWrap is Terminal.printWrap.
//
// Zig: fn printWrap(self: *Terminal) !void. The inferred error set is that
// of index.
func (self *Terminal) printWrap() error { panic(notImplemented) }

// ConfigureCharset is Terminal.configureCharset.
//
// Set the charset into the given slot.
func (self *Terminal) ConfigureCharset(slot CharsetsSlots, set CharsetsCharset) {
	panic(notImplemented)
}

// InvokeCharset is Terminal.invokeCharset.
//
// Invoke the charset in slot into the active slot. If single is true,
// then this will only be invoked for a single character.
func (self *Terminal) InvokeCharset(active CharsetsActiveSlot, slot CharsetsSlots, single bool) {
	panic(notImplemented)
}

// CarriageReturn is Terminal.carriageReturn.
//
// Carriage return moves the cursor to the first column.
func (self *Terminal) CarriageReturn() { panic(notImplemented) }

// Linefeed is Terminal.linefeed.
//
// Linefeed moves the cursor to the next line.
//
// Zig: !void. The inferred error set is that of index.
func (self *Terminal) Linefeed() error { panic(notImplemented) }

// Backspace is Terminal.backspace.
//
// Backspace moves the cursor back a column (but not less than 0).
func (self *Terminal) Backspace() { panic(notImplemented) }

// CursorUp is Terminal.cursorUp.
//
// Move the cursor up amount lines. If amount is greater than the maximum
// move distance then it is internally adjusted to the maximum. If amount is
// 0, adjust it to 1.
//
// Zig: count_req is usize.
func (self *Terminal) CursorUp(count_req uint) { panic(notImplemented) }

// CursorDown is Terminal.cursorDown.
//
// Move the cursor down amount lines. If amount is greater than the maximum
// move distance then it is internally adjusted to the maximum. This sequence
// will not scroll the screen or scroll region. If amount is 0, adjust it to 1.
//
// Zig: count_req is usize.
func (self *Terminal) CursorDown(count_req uint) { panic(notImplemented) }

// CursorRight is Terminal.cursorRight.
//
// Move the cursor right amount columns. If amount is greater than the
// maximum move distance then it is internally adjusted to the maximum.
// This sequence will not scroll the screen or scroll region. If amount is
// 0, adjust it to 1.
//
// Zig: count_req is usize.
func (self *Terminal) CursorRight(count_req uint) { panic(notImplemented) }

// CursorLeft is Terminal.cursorLeft.
//
// Move the cursor to the left amount cells. If amount is 0, adjust it to 1.
//
// Zig: count_req is usize.
func (self *Terminal) CursorLeft(count_req uint) { panic(notImplemented) }

// SaveCursor is Terminal.saveCursor.
//
// Save cursor position and further state.
//
// The primary and alternate screen have distinct save state. One saved state
// is kept per screen (main / alternative). If for the current screen state
// was already saved it is overwritten.
func (self *Terminal) SaveCursor() { panic(notImplemented) }

// RestoreCursor is Terminal.restoreCursor.
//
// Restore cursor position and other state.
//
// The primary and alternate screen have distinct save state.
// If no save was done before values are reset to their initial values.
func (self *Terminal) RestoreCursor() { panic(notImplemented) }

// SetProtectedMode is Terminal.setProtectedMode.
//
// Set the character protection mode for the terminal.
func (self *Terminal) SetProtectedMode(mode ANSIProtectedMode) { panic(notImplemented) }

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
	panic(notImplemented)
}

// semanticPromptFreshLine is Terminal.semanticPromptFreshLine.
//
// OSC 133;L
//
// Zig: fn semanticPromptFreshLine(self: *Terminal) !void. The inferred
// error set is that of index.
func (self *Terminal) semanticPromptFreshLine() error { panic(notImplemented) }

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

// String returns the Zig tag name (@tagName).
func (self TerminalSemanticPrompt) String() string { panic(notImplemented) }

// CursorIsAtPrompt is Terminal.cursorIsAtPrompt.
//
// Returns true if the cursor is currently at a prompt. Another way to look
// at this is it returns false if the shell is currently outputting something.
// This requires shell integration (semantic prompt integration).
//
// If the shell integration doesn't exist, this will always return false.
func (self *Terminal) CursorIsAtPrompt() bool { panic(notImplemented) }

// HorizontalTab is Terminal.horizontalTab.
//
// Horizontal tab moves the cursor to the next tabstop, clearing
// the screen to the left the tabstop.
func (self *Terminal) HorizontalTab() { panic(notImplemented) }

// HorizontalTabBack is Terminal.horizontalTabBack.
//
// Same as horizontalTab but moves to the previous tabstop instead of the next.
func (self *Terminal) HorizontalTabBack() { panic(notImplemented) }

// TabClear is Terminal.tabClear.
//
// Clear tab stops.
func (self *Terminal) TabClear(cmd CSITabClear) { panic(notImplemented) }

// TabSet is Terminal.tabSet.
//
// Set a tab stop on the current cursor.
// TODO: test
func (self *Terminal) TabSet() { panic(notImplemented) }

// TabReset is Terminal.tabReset.
//
// TODO: test
func (self *Terminal) TabReset() { panic(notImplemented) }

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
func (self *Terminal) Index() error { panic(notImplemented) }

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
func (self *Terminal) ReverseIndex() { panic(notImplemented) }

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
func (self *Terminal) SetCursorPos(row_req uint, col_req uint) { panic(notImplemented) }

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
	panic(notImplemented)
}

// SetLeftAndRightMargin is Terminal.setLeftAndRightMargin.
//
// DECSLRM.
//
// Zig: left_req and right_req are usize.
func (self *Terminal) SetLeftAndRightMargin(left_req uint, right_req uint) {
	panic(notImplemented)
}

// ScrollDown is Terminal.scrollDown.
//
// Scroll the text down by one row.
//
// Zig: pub fn scrollDown(self: *Terminal, count: usize) void.
func (self *Terminal) ScrollDown(count uint) { panic(notImplemented) }

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
func (self *Terminal) ScrollUp(count uint) error { panic(notImplemented) }

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
	// Zig: isize.
	Delta int
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

// String returns the Zig tag name (@tagName).
func (self TerminalScrollViewportTag) String() string { panic(notImplemented) }

// ScrollViewport is Terminal.scrollViewport.
//
// Scroll the viewport of the terminal grid.
func (self *Terminal) ScrollViewport(behavior TerminalScrollViewport) { panic(notImplemented) }

// rowWillBeShifted is Terminal.rowWillBeShifted.
//
// To be called before shifting a row (as in insertLines and deleteLines)
//
// Takes care of boundary conditions such as potentially split wide chars
// across scrolling region boundaries and orphaned spacer heads at line
// ends.
//
// Zig: fn rowWillBeShifted(self: *Terminal, page: *Page, row: *Row) void.
func (self *Terminal) rowWillBeShifted(page *Page, row *PageRow) { panic(notImplemented) }

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
func (self *Terminal) InsertLines(count uint) { panic(notImplemented) }

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
func (self *Terminal) DeleteLines(count uint) { panic(notImplemented) }

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
func (self *Terminal) InsertBlanks(count uint) { panic(notImplemented) }

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
func (self *Terminal) DeleteChars(count_req uint) { panic(notImplemented) }

// EraseChars is Terminal.eraseChars.
//
// Zig: pub fn eraseChars(self: *Terminal, count_req: usize) void.
func (self *Terminal) EraseChars(count_req uint) { panic(notImplemented) }

// EraseLine is Terminal.eraseLine.
//
// Erase the line.
//
// Zig: pub fn eraseLine(self: *Terminal, mode: csi.EraseLine,
// protected_req: bool) void.
func (self *Terminal) EraseLine(mode CSIEraseLine, protected_req bool) {
	panic(notImplemented)
}

// EraseDisplay is Terminal.eraseDisplay.
//
// Erase the display.
//
// Zig: pub fn eraseDisplay(self: *Terminal, mode: csi.EraseDisplay,
// protected_req: bool) void.
func (self *Terminal) EraseDisplay(mode CSIEraseDisplay, protected_req bool) {
	panic(notImplemented)
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
func (self *Terminal) Decaln() error { panic(notImplemented) }

// Terminal.kittyGraphics (line 2681) is omitted; see the file header.

// SetAttribute is Terminal.setAttribute.
//
// Set a style attribute.
//
// Zig: pub fn setAttribute(self: *Terminal, attr: sgr.Attribute) !void. The
// inferred error set is that of Screen.setAttribute: ErrOutOfMemory
// (dcs.go) or ErrOutOfSpace (PageList.go).
func (self *Terminal) SetAttribute(attr SGRAttribute) error { panic(notImplemented) }

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
func (self *Terminal) PrintAttributes(buf []uint8) ([]uint8, error) { panic(notImplemented) }

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

// String returns the Zig tag name (@tagName).
func (self TerminalDeccolmMode) String() string { panic(notImplemented) }

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
func (self *Terminal) Deccolm(mode TerminalDeccolmMode) error { panic(notImplemented) }

// Resize is Terminal.resize.
//
// Resize the underlying terminal.
//
// Zig: pub fn resize(self: *Terminal, alloc: Allocator,
// cols: size.CellCountInt, rows: size.CellCountInt) !void. The allocator
// parameter is dropped. The inferred error set is that of Tabstops.init and
// Screen.resize: ErrOutOfMemory (dcs.go).
func (self *Terminal) Resize(cols SizeCellCountInt, rows SizeCellCountInt) error {
	panic(notImplemented)
}

// SetPwd is Terminal.setPwd.
//
// Set the pwd for the terminal.
//
// Zig: pub fn setPwd(self: *Terminal, pwd: []const u8) !void. The error is
// ErrOutOfMemory (dcs.go).
func (self *Terminal) SetPwd(pwd []uint8) error { panic(notImplemented) }

// GetPwd is Terminal.getPwd.
//
// Returns the pwd for the terminal, if any. The memory is owned by the
// Terminal and is not copied. It is safe until a reset or setPwd.
//
// Zig: pub fn getPwd(self: *const Terminal) ?[]const u8.
func (self *Terminal) GetPwd() Optional[[]uint8] { panic(notImplemented) }

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
func (self *Terminal) SwitchScreen(key ScreenSetKey) (*Screen, error) { panic(notImplemented) }

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
	panic(notImplemented)
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

// String returns the Zig tag name (@tagName).
func (self TerminalSwitchScreenMode) String() string { panic(notImplemented) }

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
func (self *Terminal) PlainString() ([]uint8, error) { panic(notImplemented) }

// PlainStringUnwrapped is Terminal.plainStringUnwrapped.
//
// Same as plainString, but respects row wrap state when building the string.
//
// Zig: pub fn plainStringUnwrapped(self: *Terminal, alloc: Allocator)
// ![]const u8. The allocator parameter is dropped. The inferred error set
// is that of Screen.dumpStringAllocUnwrapped: ErrUnknownPoint (Screen.go),
// ErrWriteFailed (osc.go) and ErrOutOfMemory (dcs.go).
func (self *Terminal) PlainStringUnwrapped() ([]uint8, error) { panic(notImplemented) }

// FullReset is Terminal.fullReset.
//
// Full reset.
//
// This will attempt to free the existing screen memory but if that fails
// this will reuse the existing memory. In the latter case, memory may
// be wasted (since its unused) but it isn't leaked.
func (self *Terminal) FullReset() { panic(notImplemented) }

// isDirty is Terminal.isDirty.
//
// Returns true if the point is dirty, used for testing.
//
// Zig: fn isDirty(t: *const Terminal, pt: point.Point) bool.
func (t *Terminal) isDirty(pt Point) bool { panic(notImplemented) }

// clearDirty is Terminal.clearDirty.
//
// Clear all dirty bits. Testing only.
//
// Zig: fn clearDirty(t: *Terminal) void.
func (t *Terminal) clearDirty() { panic(notImplemented) }
