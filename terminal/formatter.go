// WIREFRAME for src/terminal/formatter.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: formatter.zig is a namespace file, so its declarations take
// the prefix `Formatter`: `formatter.Format` is `FormatterFormat`,
// `formatter.CodepointMap` is `FormatterCodepointMap`,
// `formatter.CodepointMap.Replacement` is
// `FormatterCodepointMapReplacement`, `formatter.Options` is
// `FormatterOptions` (its presets `Options.plain`, `.vt`, `.html` are
// `FormatterOptionsPlain`, `FormatterOptionsVt`, `FormatterOptionsHtml`),
// `formatter.PinMap` is `FormatterPinMap`, `formatter.TerminalFormatter` is
// `FormatterTerminalFormatter`, `formatter.ScreenFormatter` is
// `FormatterScreenFormatter`, `formatter.PageListFormatter` is
// `FormatterPageListFormatter`, `formatter.PageFormatter` is
// `FormatterPageFormatter`. What is nested in those takes their name as
// prefix: `TerminalFormatter.Extra` is `FormatterTerminalFormatterExtra`
// (presets `FormatterTerminalFormatterExtraNone`, `...Styles`, `...All`),
// `ScreenFormatter.Content` is `FormatterScreenFormatterContent`,
// `ScreenFormatter.Extra` is `FormatterScreenFormatterExtra` (presets
// `FormatterScreenFormatterExtraNone`, `...Styles`, `...All`),
// `PageFormatter.TrailingState` is `FormatterPageFormatterTrailingState`
// (`.empty` is `FormatterPageFormatterTrailingStateEmpty`). No declaration
// has the name of the file, so none is undoubled. `init` functions are
// `<Type>Init`: `FormatterTerminalFormatterInit`,
// `FormatterScreenFormatterInit`, `FormatterPageListFormatterInit`,
// `FormatterPageFormatterInit`. `pub fn` methods are exported (`Styled`,
// `Format`, `FormatWithState`); private methods keep their Zig name
// (`isSet`, `writeCell`, `writeCodepointWithReplacement`, `writeCodepoint`,
// `cellStyle`, `formatStyleOpen`, `formatStyleClose`,
// `formatHyperlinkOpen`, `formatHyperlinkClose`).
//
// Declarations appear in the same order as in formatter.zig.
//
// # The `format` methods
//
// docs/mapping.md maps a Zig `format` used with `{f}` to `Format() string`.
// The four formatters' `format(self, writer: *std.Io.Writer)
// std.Io.Writer.Error!void` are NOT used with `{f}` anywhere: every caller
// (Screen.selectionString, Screen.dumpString, the formatters themselves and
// all 180 upstream and project tests) calls `format(&writer)` directly,
// reads the bytes from the writer, checks `error.WriteFailed` from small
// fixed writers, and relies on the side effects on `pin_map`/`point_map`.
// So `format` keeps its writer parameter and error:
// `Format(writer *oscWriter) error`. This is the one deviation from the
// mapping table in this file.
//
// # std.Io.Writer
//
// Every `*std.Io.Writer` parameter is `*oscWriter` (osc.go), the package's
// stand-in for std.Io.Writer; `std.Io.Writer.Allocating` is
// `oscWriterAllocating`, `std.Io.Writer.fixed(buf)` is `oscWriterFixed(buf)`
// and `writer.buffered()` is `writer.buffered()` (osc.go). The error of a
// failed write, `std.Io.Writer.Error`, is ErrWriteFailed (osc.go).
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory.
//
//   - Every allocator is dropped: the `alloc` field of `PinMap` and of the
//     anonymous `point_map` struct. No function in this file takes one.
//   - `PinMap.map` is `*std.ArrayList(Pin)`: a `*[]PageListPin`. The caller
//     owns the slice variable; the formatter appends through the pointer.
//     Zig's `map.items.len` is `len(*map_)`, `clearRetainingCapacity` is
//     `*map_ = (*map_)[:0]`.
//   - `PageFormatter.point_map.map` is `*std.ArrayList(Coordinate)`: a
//     `*[]PointCoordinate`, used the same way.
//   - `Options.codepoint_map` is `?std.MultiArrayList(CodepointMap)`, a
//     struct-of-arrays list of CodepointMap: an
//     `Optional[[]FormatterCodepointMap]` (slice of the struct). Zig's
//     `map.items(.range)[i]` is `map[i].range_`, `map.len` is `len(map)`,
//     `map.append(alloc, e)` is `map = append(map, e)`.
//   - Pointers to the formatted object (`*const Terminal`, `*const Screen`,
//     `*const PageList`, `*const Page`) are Go pointers.
//
// # Widths
//
// `u21` is `uint32`; `TrailingState.rows` and `.cells` are counts (`usize`)
// and so `uint`; cell coordinates are `SizeCellCountInt` (u16).
//
// # Packed structs
//
// `TerminalFormatter.Extra` and `ScreenFormatter.Extra` are packed structs
// in Zig. In Go they are ordinary structs with the same field names, and
// comparable with `==` (as `expectEqual` compares them). `isSet` is Zig's
// "backing integer != 0", i.e. "any field is true". A test that walks the
// fields with `@typeInfo` lists them by hand in Go.
//
// # Declarations that are not Zig declarations
//
//   - `FormatterPageFormatterPointMap`: the anonymous struct that is the
//     payload of the optional field `PageFormatter.point_map`;
//   - the `...Tag` types of the tagged unions (`CodepointMap.Replacement`,
//     `ScreenFormatter.Content`) and every `String` method.
//
// # Omitted
//
// Aliases and imports with nothing to declare: `std`, `assert`,
// `Allocator`, `color`, `size`, `charsets`, `hyperlink`, `kitty`,
// `modespkg`, `Screen`, `Terminal`, `Cell` (use `PageCell`), `Coordinate`
// (use `PointCoordinate`), `Page`, `PageList`, `Pin` (use `PageListPin`),
// `Row` (use `PageRow`), `Selection`, `Style`.
// No declaration is compiled out: formatter.zig has no
// `build_options.kitty_graphics`, tmux or oniguruma code (`kitty` is used
// only for `kitty.KeyFlags`, the keyboard protocol, which is ported).
// No C ABI declaration exists in this file.
//
// # What the implementation needs from files that have no Go file yet
//
// No signature in this file needs a type from an unported file, so no
// function is blocked. The bodies need std.Io facilities that the writer
// stand-in does not have yet, to be added to osc.go with Zig's names when
// implemented: `print` with `{d}`, `{x:0>2}`, `{s}`, `{c}` and `{u}`,
// `writeAll`, `writeByte`, `splatByteAll`, `std.Io.Writer.Discarding` (a
// writer that only counts bytes: `.count`), `std.io.fixedBufferStream`
// (formatWithState's HTML/VT header), and `std.unicode.Utf8View` (string
// replacements in writeCodepointWithReplacement).
//
// # Zig defaults a Go literal does not give
//
// A Go composite literal leaves an omitted field at the Go zero value. The
// Zig defaults that are NOT the Go zero value, which a translated literal
// must write out:
//   - `Options.trim` defaults to TRUE;
//   - `Options.codepoint_map` defaults to `.{}`, an EMPTY NON-NULL list;
//     the zero value is null. Write
//     `codepoint_map: Optional[[]FormatterCodepointMap]{Valid: true}`.
//     (Null and empty behave the same in formatWithState; a test that
//     checks `codepoint_map != null` sees the difference.)
// So Zig's `.{ .emit = .plain }` is
// `FormatterOptions{emit: FormatterFormatPlain, trim: true,
// codepoint_map: Optional[[]FormatterCodepointMap]{Valid: true}}`, which is
// `FormatterOptionsPlain`; start from a preset and set fields
// (`opts := FormatterOptionsPlain; opts.unwrap = true`).
// The formatter structs have no field defaults in Zig; they are made by
// their `init`, which sets fields that are NOT the Go zero value:
//   - `TerminalFormatter.init` sets `content` to `.{ .selection = null }`
//     (the zero value is `.none`) and `extra` to `.styles`;
//   - `ScreenFormatter.init` sets `content` to `.{ .selection = null }`
//     (the zero value is `.none`).
// `PageListFormatter.init` and `PageFormatter.init` set only zero values.
// So a translated test makes a formatter with its Init function, never
// with a bare literal. The Extra structs have no defaults (every field is
// required in Zig). All other defaults in this file (`unwrap` false;
// `background`, `foreground`, `palette` null; `TrailingState` 0, 0) are the
// Go zero value and are noted on the fields.

package terminal

// FormatterFormat is formatter.Format.
//
// Formats available.
//
// Zig: enum, values numbered from zero in the order listed.
type FormatterFormat uint8

const (
	// Plain text.
	FormatterFormatPlain FormatterFormat = iota

	// Include VT sequences to preserve colors, styles, URLs, etc.
	// This is predominantly SGR sequences but may contain others as needed.
	//
	// Note that for reference colors, like palette indices, this will
	// vary based on the formatter and you should see the docs. For example,
	// PageFormatter with VT will emit SGR sequences with palette indices,
	// not the color itself.
	//
	// For VT, newlines will be emitted as `\r\n` so that the cursor properly
	// moves back to the beginning prior emitting follow-up lines.
	FormatterFormatVt

	// HTML output.
	//
	// This will emit inline styles for as much styling as possible,
	// in the interest of simplicity and ease of editing. This isn't meant
	// to build the most beautiful or efficient HTML, but rather to be
	// stylistically correct.
	//
	// For colors, RGB values are emitted as inline CSS (#RRGGBB) while palette
	// indices use CSS variables (var(--vt-palette-N)). The palette colors are
	// emitted by TerminalFormatter.Extra.palette as a <style> block if you
	// want to also include that. But if you only format a screen or lower,
	// the formatter doesn't have access to the current palette to render it.
	//
	// Newlines are emitted as actual '\n' characters. Consumers should use
	// CSS white-space: pre or pre-wrap to preserve spacing and alignment.
	FormatterFormatHtml
)

// String returns the Zig tag name (@tagName).
func (self FormatterFormat) String() string { panic(notImplemented) }

// Styled is formatter.Format.styled.
//
// Zig: pub fn styled(self: Format) bool.
func (self FormatterFormat) Styled() bool { panic(notImplemented) }

// FormatterCodepointMap is formatter.CodepointMap.
type FormatterCodepointMap struct {
	// Unicode codepoint range to replace.
	// Asserts: range[0] <= range[1]
	//
	// Zig: `range: [2]u21`. `range` is a Go keyword.
	range_ [2]uint32

	// Replacement value for this range.
	replacement FormatterCodepointMapReplacement
}

// FormatterCodepointMapReplacement is formatter.CodepointMap.Replacement.
//
// Zig: union(enum) { codepoint: u21, string: []const u8 }. Only the field
// named by Tag is meaningful. The Go zero value is `.codepoint` 0.
type FormatterCodepointMapReplacement struct {
	Tag FormatterCodepointMapReplacementTag

	// A single replacement codepoint.
	//
	// Zig: u21.
	Codepoint uint32

	// A UTF-8 encoded string to replace with. Asserts the
	// UTF-8 encoding (must be valid).
	//
	// Zig: []const u8.
	String []uint8
}

// FormatterCodepointMapReplacementTag is the tag of
// formatter.CodepointMap.Replacement, values numbered from zero in the
// order listed.
type FormatterCodepointMapReplacementTag uint8

const (
	FormatterCodepointMapReplacementCodepoint FormatterCodepointMapReplacementTag = iota
	FormatterCodepointMapReplacementString
)

// String returns the Zig tag name (@tagName).
func (self FormatterCodepointMapReplacementTag) String() string { panic(notImplemented) }

// FormatterOptions is formatter.Options.
//
// Common encoding options regardless of what exact formatter is used.
//
// Two Zig defaults are NOT the Go zero value (`trim`, `codepoint_map`); see
// the file header. Start from FormatterOptionsPlain, FormatterOptionsVt or
// FormatterOptionsHtml.
type FormatterOptions struct {
	// The format to emit.
	//
	// Zig: no default (required).
	emit FormatterFormat

	// Whether to unwrap soft-wrapped lines. If false, this will emit the
	// screen contents as it is rendered on the page in the given size.
	//
	// Zig default: false.
	unwrap bool

	// Trim trailing whitespace on lines with other text. Trailing blank
	// lines are always trimmed. This only affects trailing whitespace
	// on rows that have at least one other cell with text. Whitespace
	// is currently only space characters (0x20).
	//
	// Zig default: TRUE, which is NOT the Go zero value.
	trim bool

	// Replace matching Unicode codepoints with some other values.
	// This will use the last matching range found in the list.
	//
	// Zig: ?std.MultiArrayList(CodepointMap), default `.{}` (an empty,
	// non-null list), which is NOT the Go zero value (null). Go: a slice of
	// the struct; see the file header.
	codepoint_map Optional[[]FormatterCodepointMap]

	// Set a background and foreground color to use for the "screen".
	// For styled formats, this will emit the proper sequences or styles.
	//
	// Zig: ?color.RGB, default null.
	background Optional[ColorRGB]
	// Zig: ?color.RGB, default null.
	foreground Optional[ColorRGB]

	// If set, then styled formats in `emit` will use this palette to
	// emit colors directly as RGB. If this is null, styled formats will
	// still work but will use deferred palette styling (e.g. CSS variables
	// for HTML or the actual palette indexes for VT).
	//
	// Zig: ?*const color.Palette, default null. nil is null.
	palette *ColorPalette
}

// FormatterOptionsPlain is formatter.Options.plain: `.{ .emit = .plain }`,
// with the other fields at their Zig defaults (written out: `trim` true,
// `codepoint_map` empty and non-null).
var FormatterOptionsPlain = FormatterOptions{
	emit:          FormatterFormatPlain,
	trim:          true,
	codepoint_map: Optional[[]FormatterCodepointMap]{Valid: true},
}

// FormatterOptionsVt is formatter.Options.vt: `.{ .emit = .vt }`, with the
// other fields at their Zig defaults (written out as for
// FormatterOptionsPlain).
var FormatterOptionsVt = FormatterOptions{
	emit:          FormatterFormatVt,
	trim:          true,
	codepoint_map: Optional[[]FormatterCodepointMap]{Valid: true},
}

// FormatterOptionsHtml is formatter.Options.html: `.{ .emit = .html }`,
// with the other fields at their Zig defaults (written out as for
// FormatterOptionsPlain).
var FormatterOptionsHtml = FormatterOptions{
	emit:          FormatterFormatHtml,
	trim:          true,
	codepoint_map: Optional[[]FormatterCodepointMap]{Valid: true},
}

// FormatterPinMap is formatter.PinMap.
//
// Maps byte positions in formatted output to PageList pins.
//
// Used by formatters that operate on PageLists to track the source position
// of each byte written. The caller is responsible for freeing the map.
//
// Zig's first field, `alloc: Allocator`, is dropped.
type FormatterPinMap struct {
	// Zig: `map: *std.ArrayList(Pin)`. `map` is a Go keyword. A pointer to
	// the caller's slice; see the file header.
	map_ *[]PageListPin
}

// FormatterTerminalFormatter is formatter.TerminalFormatter.
//
// Terminal formatter formats the active terminal screen.
//
// This will always only emit data related to the currently active screen.
// If you want to emit data for a specific screen (e.g. primary vs alt), then
// switch to that screen in the terminal prior to using this.
//
// If you want to emit data for all screens (a less common operation), then
// you must create a no-content TerminalFormatter followed by multiple
// explicit ScreenFormatter calls. This isn't a common operation so this
// little extra work should be acceptable.
//
// For styled formatting, this will emit the palette colors at the
// beginning so that the output can be rendered properly according to
// the current terminal state.
//
// Zig: no field has a default; use FormatterTerminalFormatterInit (see the
// file header).
type FormatterTerminalFormatter struct {
	// The terminal to format.
	//
	// Zig: *const Terminal.
	terminal *Terminal

	// The common options
	opts FormatterOptions

	// The content to include.
	content FormatterScreenFormatterContent

	// Extra stuff to emit, such as terminal modes, palette, cursor, etc.
	// This information is ONLY emitted when the format is "vt".
	extra FormatterTerminalFormatterExtra

	// If non-null, then `map` will contain the Pin of every byte
	// byte written to the writer offset by the byte index. It is the
	// caller's responsibility to free the map.
	//
	// Note that some emitted bytes may not correspond to any Pin, such as
	// the extra data around terminal state (palette, modes, etc.). For these,
	// we'll map it to the most previous pin so there is some continuity but
	// its an arbitrary choice.
	//
	// Warning: there is a significant performance hit to track this
	//
	// Zig: ?PinMap.
	pin_map Optional[FormatterPinMap]
}

// FormatterTerminalFormatterExtra is formatter.TerminalFormatter.Extra.
//
// Zig: packed struct; no field has a default. In Go an ordinary struct;
// see the file header.
type FormatterTerminalFormatterExtra struct {
	// Emit the palette using OSC 4 sequences.
	palette bool

	// Emit terminal modes that differ from their defaults using CSI h/l
	// sequences. Defaults are according to the Ghostty defaults which
	// are generally match most terminal defaults. This will include
	// things like current screen, bracketed mode, mouse event reporting,
	// etc.
	modes bool

	// Emit scrolling region state using DECSTBM and DECSLRM sequences.
	scrolling_region bool

	// Emit tabstop positions by clearing all tabs (CSI 3 g) and setting
	// each configured tabstop with HTS.
	tabstops bool

	// Emit the present working directory using OSC 7.
	pwd bool

	// Emit keyboard modes such as ModifyOtherKeys using CSI > 4 m
	// sequences.
	keyboard bool

	// The screen extras to emit. TerminalFormatter always only
	// emits data for the currently active screen. If you want to emit
	// data for all screens, you should manually construct a no-content
	// terminal formatter, followed by screen formatters.
	screen FormatterScreenFormatterExtra
}

// FormatterTerminalFormatterExtraNone is
// formatter.TerminalFormatter.Extra.none.
//
// Emit nothing.
var FormatterTerminalFormatterExtraNone = FormatterTerminalFormatterExtra{
	palette:          false,
	modes:            false,
	scrolling_region: false,
	tabstops:         false,
	pwd:              false,
	keyboard:         false,
	screen: FormatterScreenFormatterExtra{
		cursor:         false,
		style:          false,
		hyperlink:      false,
		protection:     false,
		kitty_keyboard: false,
		charsets:       false,
	},
}

// FormatterTerminalFormatterExtraStyles is
// formatter.TerminalFormatter.Extra.styles.
//
// Emit style-relevant information only such as palettes.
var FormatterTerminalFormatterExtraStyles = FormatterTerminalFormatterExtra{
	palette:          true,
	modes:            false,
	scrolling_region: false,
	tabstops:         false,
	pwd:              false,
	keyboard:         false,
	screen: FormatterScreenFormatterExtra{
		cursor:         false,
		style:          true,
		hyperlink:      true,
		protection:     false,
		kitty_keyboard: false,
		charsets:       false,
	},
}

// FormatterTerminalFormatterExtraAll is
// formatter.TerminalFormatter.Extra.all.
//
// Emit everything. This reconstructs the terminal state as closely
// as possible.
var FormatterTerminalFormatterExtraAll = FormatterTerminalFormatterExtra{
	palette:          true,
	modes:            true,
	scrolling_region: true,
	tabstops:         true,
	pwd:              true,
	keyboard:         true,
	screen: FormatterScreenFormatterExtra{
		cursor:         true,
		style:          true,
		hyperlink:      true,
		protection:     true,
		kitty_keyboard: true,
		charsets:       true,
	},
}

// FormatterTerminalFormatterInit is formatter.TerminalFormatter.init.
//
// Zig: pub fn init(terminal: *const Terminal, opts: Options)
// TerminalFormatter. It sets `content` to `.{ .selection = null }`,
// `extra` to `.styles` and `pin_map` to null.
func FormatterTerminalFormatterInit(terminal *Terminal, opts FormatterOptions) FormatterTerminalFormatter {
	panic(notImplemented)
}

// Format is formatter.TerminalFormatter.format.
//
// Zig: pub fn format(self: TerminalFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The writer is the stand-in oscWriter (osc.go);
// the error is ErrWriteFailed (osc.go), also returned when appending to the
// pin map fails. See the file header for why this is not `Format() string`.
func (self FormatterTerminalFormatter) Format(writer *oscWriter) error { panic(notImplemented) }

// FormatterScreenFormatter is formatter.ScreenFormatter.
//
// Screen formatter formats a single terminal screen (e.g. primary vs alt).
//
// Zig: no field has a default; use FormatterScreenFormatterInit (see the
// file header).
type FormatterScreenFormatter struct {
	// The screen to format.
	//
	// Zig: *const Screen.
	screen *Screen

	// The common options
	opts FormatterOptions

	// The content to include.
	content FormatterScreenFormatterContent

	// Extra stuff to emit, such as cursor, style, hyperlinks, etc.
	// This information is ONLY emitted when the format is "vt".
	extra FormatterScreenFormatterExtra

	// If non-null, then `map` will contain the Pin of every byte
	// byte written to the writer offset by the byte index. It is the
	// caller's responsibility to free the map.
	//
	// Note that some emitted bytes may not correspond to any Pin, such as
	// the extra data around screen state. For these, we'll map it to the
	// most previous pin so there is some continuity but its an arbitrary
	// choice.
	//
	// Warning: there is a significant performance hit to track this
	//
	// Zig: ?PinMap.
	pin_map Optional[FormatterPinMap]
}

// FormatterScreenFormatterContent is formatter.ScreenFormatter.Content.
//
// Zig: union(enum) { none, selection: ?Selection }. Only the field named by
// Tag is meaningful; `none` has no payload and no field. The Go zero value
// is `.none`, which is NOT what `init` sets (`.{ .selection = null }`).
type FormatterScreenFormatterContent struct {
	Tag FormatterScreenFormatterContentTag

	// Emit the content specified by the selection. Null for all.
	// The selection is inclusive on both ends.
	//
	// Zig: ?Selection.
	Selection Optional[Selection]
}

// FormatterScreenFormatterContentTag is the tag of
// formatter.ScreenFormatter.Content, values numbered from zero in the order
// listed.
type FormatterScreenFormatterContentTag uint8

const (
	// Emit no content, only terminal state such as modes, palette, etc.
	// via extra.
	FormatterScreenFormatterContentNone FormatterScreenFormatterContentTag = iota

	// Emit the content specified by the selection.
	FormatterScreenFormatterContentSelection
)

// String returns the Zig tag name (@tagName).
func (self FormatterScreenFormatterContentTag) String() string { panic(notImplemented) }

// FormatterScreenFormatterExtra is formatter.ScreenFormatter.Extra.
//
// Zig: packed struct; no field has a default. In Go an ordinary struct;
// see the file header.
type FormatterScreenFormatterExtra struct {
	// Emit cursor position using CUP (CSI H).
	cursor bool

	// Emit current SGR style state based on the cursor's active style_id.
	// This reconstructs the SGR attributes (bold, italic, colors, etc.) at
	// the cursor position.
	style bool

	// Emit current hyperlink state using OSC 8 sequences.
	// This sets the active hyperlink based on cursor.hyperlink_id.
	hyperlink bool

	// Emit character protection mode using DECSCA.
	protection bool

	// Emit Kitty keyboard protocol state using CSI > u and CSI = sequences.
	kitty_keyboard bool

	// Emit character set designations and invocations.
	// This includes G0-G3 designations (ESC ( ) * +) and GL/GR invocations.
	charsets bool
}

// FormatterScreenFormatterExtraNone is formatter.ScreenFormatter.Extra.none.
//
// Emit nothing.
var FormatterScreenFormatterExtraNone = FormatterScreenFormatterExtra{
	cursor:         false,
	style:          false,
	hyperlink:      false,
	protection:     false,
	kitty_keyboard: false,
	charsets:       false,
}

// FormatterScreenFormatterExtraStyles is
// formatter.ScreenFormatter.Extra.styles.
//
// Emit style-relevant information only.
var FormatterScreenFormatterExtraStyles = FormatterScreenFormatterExtra{
	cursor:         false,
	style:          true,
	hyperlink:      true,
	protection:     false,
	kitty_keyboard: false,
	charsets:       false,
}

// FormatterScreenFormatterExtraAll is formatter.ScreenFormatter.Extra.all.
//
// Emit everything. This reconstructs the screen state as closely
// as possible.
var FormatterScreenFormatterExtraAll = FormatterScreenFormatterExtra{
	cursor:         true,
	style:          true,
	hyperlink:      true,
	protection:     true,
	kitty_keyboard: true,
	charsets:       true,
}

// isSet is formatter.ScreenFormatter.Extra.isSet (private in Zig).
//
// Zig: fn isSet(self: Extra) bool, true when the packed struct's backing
// integer is non-zero, i.e. when any field is true. Zig's `E.isSet(.none)`
// is `FormatterScreenFormatterExtraNone.isSet()`.
func (self FormatterScreenFormatterExtra) isSet() bool { panic(notImplemented) }

// FormatterScreenFormatterInit is formatter.ScreenFormatter.init.
//
// Zig: pub fn init(screen: *const Screen, opts: Options) ScreenFormatter.
// It sets `content` to `.{ .selection = null }`, `extra` to `.none` and
// `pin_map` to null.
func FormatterScreenFormatterInit(screen *Screen, opts FormatterOptions) FormatterScreenFormatter {
	panic(notImplemented)
}

// Format is formatter.ScreenFormatter.format.
//
// Zig: pub fn format(self: ScreenFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The writer is the stand-in oscWriter (osc.go);
// the error is ErrWriteFailed (osc.go), also returned when appending to the
// pin map fails. See the file header for why this is not `Format() string`.
func (self FormatterScreenFormatter) Format(writer *oscWriter) error { panic(notImplemented) }

// FormatterPageListFormatter is formatter.PageListFormatter.
//
// PageList formatter formats multiple pages as represented by a PageList.
//
// Zig: no field has a default; use FormatterPageListFormatterInit.
type FormatterPageListFormatter struct {
	// The pagelist to format.
	//
	// Zig: *const PageList.
	list *PageList

	// The common options
	opts FormatterOptions

	// The bounds of the PageList to format. The top left and bottom right
	// MUST be ordered properly.
	//
	// Zig: ?PageList.Pin.
	top_left Optional[PageListPin]
	// Zig: ?PageList.Pin.
	bottom_right Optional[PageListPin]

	// If true, the boundaries define a rectangle selection where start_x
	// and end_x apply to every row, not just the first and last.
	rectangle bool

	// If non-null, then `map` will contain the Pin of every byte
	// byte written to the writer offset by the byte index. It is the
	// caller's responsibility to free the map.
	//
	// Warning: there is a significant performance hit to track this
	//
	// Zig: ?PinMap.
	pin_map Optional[FormatterPinMap]
}

// FormatterPageListFormatterInit is formatter.PageListFormatter.init.
//
// Zig: pub fn init(list: *const PageList, opts: Options) PageListFormatter.
// Every other field is null or false.
func FormatterPageListFormatterInit(list *PageList, opts FormatterOptions) FormatterPageListFormatter {
	panic(notImplemented)
}

// Format is formatter.PageListFormatter.format.
//
// Zig: pub fn format(self: PageListFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The writer is the stand-in oscWriter (osc.go);
// the error is ErrWriteFailed (osc.go), also returned when appending to the
// pin map fails. See the file header for why this is not `Format() string`.
func (self FormatterPageListFormatter) Format(writer *oscWriter) error { panic(notImplemented) }

// FormatterPageFormatter is formatter.PageFormatter.
//
// Page formatter.
//
// For styled formatting such as VT, this will emit references for palette
// colors. If you want to capture the palette as-is at the type of formatting,
// you'll have to emit the sequences for setting up the palette prior to
// this formatting. (TODO: A function to do this)
//
// Zig: no field has a default; use FormatterPageFormatterInit.
type FormatterPageFormatter struct {
	// The page to format.
	//
	// Zig: *const Page.
	page *Page

	// The common options
	opts FormatterOptions

	// Start and end points within the page to format. If end x is not given
	// then it will be the full width. If end y is not given then it will be
	// the full height.
	//
	// The start and end are both inclusive, so equal values will still
	// return a non-empty result (i.e. a single cell or row).
	//
	// The start x is considered the X in the first row and end X is
	// X in the final row. This isn't a rectangle selection by default.
	//
	// If start X falls on the second column of a wide character, then
	// the entire character will be included (as if you specified the
	// previous column).
	start_x SizeCellCountInt
	start_y SizeCellCountInt
	// Zig: ?size.CellCountInt.
	end_x Optional[SizeCellCountInt]
	// Zig: ?size.CellCountInt.
	end_y Optional[SizeCellCountInt]

	// If true, the start x/y and end x/y define a rectangle selection.
	// In this case, the boundaries will apply to every row, not just
	// the first and last.
	rectangle bool

	// If non-null, then `map` will contain the x/y coordinate of every
	// byte written to the writer offset by the byte index. It is the
	// caller's responsibility to free the map.
	//
	// The x/y coordinate will be the coordinates within the page.
	//
	// Warning: there is a significant performance hit to track this
	//
	// Zig: ?struct { alloc: Allocator, map: *std.ArrayList(Coordinate) }.
	point_map Optional[FormatterPageFormatterPointMap]

	// The previous trailing state from the prior page. If you're iterating
	// over multiple pages this helps ensure that unwrapping and other
	// accounting works properly.
	//
	// Zig: ?TrailingState.
	trailing_state Optional[FormatterPageFormatterTrailingState]
}

// FormatterPageFormatterPointMap is NOT a named Zig declaration: it is the
// anonymous struct that is the payload of the optional field
// PageFormatter.point_map. Zig's first field, `alloc: Allocator`, is
// dropped. Zig's `formatter.point_map = .{ .alloc = alloc, .map =
// &point_map }` is `formatter.point_map =
// Some(FormatterPageFormatterPointMap{map_: &point_map})` with
// `var point_map []PointCoordinate`.
type FormatterPageFormatterPointMap struct {
	// Zig: `map: *std.ArrayList(Coordinate)`. `map` is a Go keyword.
	map_ *[]PointCoordinate
}

// FormatterPageFormatterTrailingState is
// formatter.PageFormatter.TrailingState.
//
// Trailing state. This is used to ensure that rows wrapped across
// multiple pages are unwrapped properly, as well as other accounting
// we may do in the future.
type FormatterPageFormatterTrailingState struct {
	// Zig: usize, default 0. A count of rows.
	rows uint
	// Zig: usize, default 0. A count of cells.
	cells uint
}

// FormatterPageFormatterTrailingStateEmpty is
// formatter.PageFormatter.TrailingState.empty.
var FormatterPageFormatterTrailingStateEmpty = FormatterPageFormatterTrailingState{
	rows:  0,
	cells: 0,
}

// FormatterPageFormatterInit is formatter.PageFormatter.init.
//
// Initializes a page formatter. Other options can be set directly on the
// struct after initialization and before calling `format()`.
//
// Zig: pub fn init(page: *const Page, opts: Options) PageFormatter. Every
// other field is 0, null or false.
func FormatterPageFormatterInit(page *Page, opts FormatterOptions) FormatterPageFormatter {
	panic(notImplemented)
}

// Format is formatter.PageFormatter.format.
//
// Zig: pub fn format(self: PageFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void: formatWithState with the state discarded. The
// writer is the stand-in oscWriter (osc.go); the error is ErrWriteFailed
// (osc.go). See the file header for why this is not `Format() string`.
func (self FormatterPageFormatter) Format(writer *oscWriter) error { panic(notImplemented) }

// FormatWithState is formatter.PageFormatter.formatWithState.
//
// Zig: pub fn formatWithState(self: PageFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!TrailingState. The writer is the stand-in oscWriter
// (osc.go); the error is ErrWriteFailed (osc.go), also returned when
// appending to the point map fails.
func (self FormatterPageFormatter) FormatWithState(writer *oscWriter) (FormatterPageFormatterTrailingState, error) {
	panic(notImplemented)
}

// writeCell is formatter.PageFormatter.writeCell.
//
// Zig: fn writeCell(self: PageFormatter, comptime tag: Cell.ContentTag,
// writer: *std.Io.Writer, cell: *const Cell) !void. The comptime parameter
// is an ordinary parameter; Zig instantiates it only for `.codepoint` and
// `.codepoint_grapheme`. The inferred error set is ErrWriteFailed
// (osc.go).
func (self FormatterPageFormatter) writeCell(tag PageCellContentTag, writer *oscWriter, cell *PageCell) error {
	panic(notImplemented)
}

// writeCodepointWithReplacement is
// formatter.PageFormatter.writeCodepointWithReplacement.
//
// Zig: fn writeCodepointWithReplacement(self: PageFormatter,
// writer: *std.Io.Writer, codepoint: u21) !void. The inferred error set is
// ErrWriteFailed (osc.go).
func (self FormatterPageFormatter) writeCodepointWithReplacement(writer *oscWriter, codepoint uint32) error {
	panic(notImplemented)
}

// writeCodepoint is formatter.PageFormatter.writeCodepoint.
//
// Zig: fn writeCodepoint(self: PageFormatter, writer: *std.Io.Writer,
// codepoint: u21) !void. The inferred error set is ErrWriteFailed
// (osc.go).
func (self FormatterPageFormatter) writeCodepoint(writer *oscWriter, codepoint uint32) error {
	panic(notImplemented)
}

// cellStyle is formatter.PageFormatter.cellStyle.
//
// Returns the style for the given cell. If there is no styling this
// will return the default style.
//
// Zig: fn cellStyle(self: *const PageFormatter, cell: *const Cell) Style.
func (self *FormatterPageFormatter) cellStyle(cell *PageCell) Style { panic(notImplemented) }

// formatStyleOpen is formatter.PageFormatter.formatStyleOpen.
//
// Write a string with HTML escaping. Used for escaping href attributes
// and other HTML attribute values.
//
// Zig: fn formatStyleOpen(self: PageFormatter, writer: *std.Io.Writer,
// style: *const Style) std.Io.Writer.Error!void. The error is
// ErrWriteFailed (osc.go). Zig traps (`unreachable`) for `.plain`.
func (self FormatterPageFormatter) formatStyleOpen(writer *oscWriter, style *Style) error {
	panic(notImplemented)
}

// formatStyleClose is formatter.PageFormatter.formatStyleClose.
//
// Zig: fn formatStyleClose(self: PageFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go), also
// returned when appending to the point map fails.
func (self FormatterPageFormatter) formatStyleClose(writer *oscWriter) error {
	panic(notImplemented)
}

// formatHyperlinkOpen is formatter.PageFormatter.formatHyperlinkOpen.
//
// Zig: fn formatHyperlinkOpen(self: PageFormatter, writer: *std.Io.Writer,
// uri: []const u8) std.Io.Writer.Error!void. The error is ErrWriteFailed
// (osc.go). Zig traps (`unreachable`) for `.plain` and `.vt`.
func (self FormatterPageFormatter) formatHyperlinkOpen(writer *oscWriter, uri []uint8) error {
	panic(notImplemented)
}

// formatHyperlinkClose is formatter.PageFormatter.formatHyperlinkClose.
//
// Zig: fn formatHyperlinkClose(self: PageFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go), also
// returned when appending to the point map fails.
func (self FormatterPageFormatter) formatHyperlinkClose(writer *oscWriter) error {
	panic(notImplemented)
}
