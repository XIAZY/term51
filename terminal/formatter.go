// Ported from src/terminal/formatter.zig (Ghostty v1.3.1).
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
// # std.Io facilities
//
// The writer facilities the bodies need are in osc.go with Zig's names:
// `splatByteAll` (with `splatByte`, `writeSplat`, `countSplat`),
// `printUnicodeCodepoint` (what `{u}` prints) and
// `std.Io.Writer.Discarding` (`oscWriterDiscarding`; Zig's
// `discarding.count` is `discarding.writer.count`). The rest map as
// follows:
//   - `writer.print(fmt, args)` is one `writer.writeAll` of the same bytes,
//     built with `itoa` (`{d}`), `formatterHex2` (`{x:0>2}` of a u8) and
//     string concatenation (`{s}`, `{c}`), with Zig's print line in a
//     comment above (the style.go precedent: print writes its pieces with
//     writeAll one after the other, and for every writer here that is the
//     same as writing them at once, also when a fixed writer runs out).
//     `print("{f}", .{x})` is `writeAll(x.Format())`.
//   - `std.io.fixedBufferStream(&buf)` (formatWithState's HTML and VT
//     header) is `oscWriterFixed(buf[:])` and `getWritten()` is
//     `buffered()`: every write into it is `catch return
//     error.WriteFailed`, and a fixed writer's failure is ErrWriteFailed.
//   - `std.unicode.Utf8View.init(s) catch unreachable` and its iterator are
//     `utf8.Valid(s)` (false panics "unreachable") and `utf8.DecodeRune`:
//     both reject exactly what Zig's UTF-8 validation rejects (overlong
//     forms, surrogate halves, codepoints above U+10FFFF, truncated
//     sequences), and on valid input decode the same codepoints.
//   - `ArrayList.appendNTimes`, and `ensureUnusedCapacity` followed by
//     `appendNTimesAssumeCapacity`, are `formatterAppendNTimes`; `append`
//     is Go's append. Zig's `catch return error.WriteFailed` on them
//     cannot happen in Go (no allocation fails).
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

import "unicode/utf8"

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

// formatterFormatNames are the tag names of FormatterFormat, indexed by
// value (Zig's @tagName).
var formatterFormatNames = [...]string{"plain", "vt", "html"}

// String returns the Zig tag name (@tagName).
func (self FormatterFormat) String() string { return formatterFormatNames[self] }

// Styled is formatter.Format.styled.
//
// Zig: pub fn styled(self: Format) bool.
func (self FormatterFormat) Styled() bool {
	switch self {
	case FormatterFormatPlain:
		return false
	case FormatterFormatHtml, FormatterFormatVt:
		return true
	}
	panic("unreachable")
}

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

// formatterCodepointMapReplacementTagNames are the tag names of
// FormatterCodepointMapReplacementTag, indexed by value (Zig's @tagName).
var formatterCodepointMapReplacementTagNames = [...]string{"codepoint", "string"}

// String returns the Zig tag name (@tagName).
func (self FormatterCodepointMapReplacementTag) String() string {
	return formatterCodepointMapReplacementTagNames[self]
}

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
	Emit FormatterFormat

	// Whether to unwrap soft-wrapped lines. If false, this will emit the
	// screen contents as it is rendered on the page in the given size.
	//
	// Zig default: false.
	Unwrap bool

	// Trim trailing whitespace on lines with other text. Trailing blank
	// lines are always trimmed. This only affects trailing whitespace
	// on rows that have at least one other cell with text. Whitespace
	// is currently only space characters (0x20).
	//
	// Zig default: TRUE, which is NOT the Go zero value.
	Trim bool

	// Replace matching Unicode codepoints with some other values.
	// This will use the last matching range found in the list.
	//
	// Zig: ?std.MultiArrayList(CodepointMap), default `.{}` (an empty,
	// non-null list), which is NOT the Go zero value (null). Go: a slice of
	// the struct; see the file header.
	CodepointMap Optional[[]FormatterCodepointMap]

	// Set a background and foreground color to use for the "screen".
	// For styled formats, this will emit the proper sequences or styles.
	//
	// Zig: ?color.RGB, default null.
	Background Optional[ColorRGB]
	// Zig: ?color.RGB, default null.
	Foreground Optional[ColorRGB]

	// If set, then styled formats in `emit` will use this palette to
	// emit colors directly as RGB. If this is null, styled formats will
	// still work but will use deferred palette styling (e.g. CSS variables
	// for HTML or the actual palette indexes for VT).
	//
	// Zig: ?*const color.Palette, default null. nil is null.
	Palette *ColorPalette
}

// FormatterOptionsPlain is formatter.Options.plain: `.{ .emit = .plain }`,
// with the other fields at their Zig defaults (written out: `trim` true,
// `codepoint_map` empty and non-null).
var FormatterOptionsPlain = FormatterOptions{
	Emit:         FormatterFormatPlain,
	Trim:         true,
	CodepointMap: Optional[[]FormatterCodepointMap]{Valid: true},
}

// FormatterOptionsVt is formatter.Options.vt: `.{ .emit = .vt }`, with the
// other fields at their Zig defaults (written out as for
// FormatterOptionsPlain).
var FormatterOptionsVt = FormatterOptions{
	Emit:         FormatterFormatVt,
	Trim:         true,
	CodepointMap: Optional[[]FormatterCodepointMap]{Valid: true},
}

// FormatterOptionsHtml is formatter.Options.html: `.{ .emit = .html }`,
// with the other fields at their Zig defaults (written out as for
// FormatterOptionsPlain).
var FormatterOptionsHtml = FormatterOptions{
	Emit:         FormatterFormatHtml,
	Trim:         true,
	CodepointMap: Optional[[]FormatterCodepointMap]{Valid: true},
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
	Palette bool

	// Emit terminal modes that differ from their defaults using CSI h/l
	// sequences. Defaults are according to the Ghostty defaults which
	// are generally match most terminal defaults. This will include
	// things like current screen, bracketed mode, mouse event reporting,
	// etc.
	Modes bool

	// Emit scrolling region state using DECSTBM and DECSLRM sequences.
	ScrollingRegion bool

	// Emit tabstop positions by clearing all tabs (CSI 3 g) and setting
	// each configured tabstop with HTS.
	Tabstops bool

	// Emit the present working directory using OSC 7.
	Pwd bool

	// Emit keyboard modes such as ModifyOtherKeys using CSI > 4 m
	// sequences.
	Keyboard bool

	// The screen extras to emit. TerminalFormatter always only
	// emits data for the currently active screen. If you want to emit
	// data for all screens, you should manually construct a no-content
	// terminal formatter, followed by screen formatters.
	Screen FormatterScreenFormatterExtra
}

// FormatterTerminalFormatterExtraNone is
// formatter.TerminalFormatter.Extra.none.
//
// Emit nothing.
var FormatterTerminalFormatterExtraNone = FormatterTerminalFormatterExtra{
	Palette:         false,
	Modes:           false,
	ScrollingRegion: false,
	Tabstops:        false,
	Pwd:             false,
	Keyboard:        false,
	Screen: FormatterScreenFormatterExtra{
		Cursor:        false,
		Style:         false,
		Hyperlink:     false,
		Protection:    false,
		KittyKeyboard: false,
		Charsets:      false,
	},
}

// FormatterTerminalFormatterExtraStyles is
// formatter.TerminalFormatter.Extra.styles.
//
// Emit style-relevant information only such as palettes.
var FormatterTerminalFormatterExtraStyles = FormatterTerminalFormatterExtra{
	Palette:         true,
	Modes:           false,
	ScrollingRegion: false,
	Tabstops:        false,
	Pwd:             false,
	Keyboard:        false,
	Screen: FormatterScreenFormatterExtra{
		Cursor:        false,
		Style:         true,
		Hyperlink:     true,
		Protection:    false,
		KittyKeyboard: false,
		Charsets:      false,
	},
}

// FormatterTerminalFormatterExtraAll is
// formatter.TerminalFormatter.Extra.all.
//
// Emit everything. This reconstructs the terminal state as closely
// as possible.
var FormatterTerminalFormatterExtraAll = FormatterTerminalFormatterExtra{
	Palette:         true,
	Modes:           true,
	ScrollingRegion: true,
	Tabstops:        true,
	Pwd:             true,
	Keyboard:        true,
	Screen: FormatterScreenFormatterExtra{
		Cursor:        true,
		Style:         true,
		Hyperlink:     true,
		Protection:    true,
		KittyKeyboard: true,
		Charsets:      true,
	},
}

// FormatterTerminalFormatterInit is formatter.TerminalFormatter.init.
//
// Zig: pub fn init(terminal: *const Terminal, opts: Options)
// TerminalFormatter. It sets `content` to `.{ .selection = null }`,
// `extra` to `.styles` and `pin_map` to null.
func FormatterTerminalFormatterInit(terminal *Terminal, opts FormatterOptions) FormatterTerminalFormatter {
	return FormatterTerminalFormatter{
		terminal: terminal,
		opts:     opts,
		content: FormatterScreenFormatterContent{
			Tag:       FormatterScreenFormatterContentSelection,
			Selection: None[Selection](),
		},
		extra:   FormatterTerminalFormatterExtraStyles,
		pin_map: None[FormatterPinMap](),
	}
}

// Format is formatter.TerminalFormatter.format.
//
// Zig: pub fn format(self: TerminalFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The writer is the stand-in oscWriter (osc.go);
// the error is ErrWriteFailed (osc.go), also returned when appending to the
// pin map fails. See the file header for why this is not `Format() string`.
func (self FormatterTerminalFormatter) format(writer *oscWriter) error {
	// Emit palette before screen content if using VT format. Technically
	// we could do this after but this way if replay is slow for whatever
	// reason the colors will be right right away.
	if self.extra.Palette {
		switch self.opts.Emit {
		case FormatterFormatPlain:
			goto palette // break :palette

		case FormatterFormatVt:
			for i, rgb := range &self.terminal.Colors.Palette.Current {
				// writer.print(
				//     "\x1b]4;{d};rgb:{x:0>2}/{x:0>2}/{x:0>2}\x1b\\",
				//     .{ i, rgb.r, rgb.g, rgb.b },
				// )
				if err := writer.writeAll([]uint8("\x1b]4;" + itoa(uint64(i)) + ";rgb:" +
					formatterHex2(rgb.R) + "/" + formatterHex2(rgb.G) + "/" + formatterHex2(rgb.B) +
					"\x1b\\")); err != nil {
					return err
				}
			}

		// For HTML, we emit CSS to setup our palette variables.
		case FormatterFormatHtml:
			if err := writer.writeAll([]uint8("<style>:root{")); err != nil {
				return err
			}
			for i, rgb := range &self.terminal.Colors.Palette.Current {
				// writer.print(
				//     "--vt-palette-{d}: #{x:0>2}{x:0>2}{x:0>2};",
				//     .{ i, rgb.r, rgb.g, rgb.b },
				// )
				if err := writer.writeAll([]uint8("--vt-palette-" + itoa(uint64(i)) + ": #" +
					formatterHex2(rgb.R) + formatterHex2(rgb.G) + formatterHex2(rgb.B) + ";")); err != nil {
					return err
				}
			}
			if err := writer.writeAll([]uint8("}</style>")); err != nil {
				return err
			}

		default:
			panic("unreachable")
		}

		// If we have a pin_map, add the bytes we wrote to map.
		if self.pin_map.Valid {
			m := &self.pin_map.Value
			discarding := oscWriterDiscardingInit(nil)
			extra_formatter := self
			extra_formatter.content = FormatterScreenFormatterContent{Tag: FormatterScreenFormatterContentNone}
			extra_formatter.pin_map = None[FormatterPinMap]()
			extra_formatter.extra = FormatterTerminalFormatterExtraNone
			extra_formatter.extra.Palette = true
			if err := extra_formatter.format(&discarding.writer); err != nil {
				return err
			}

			// Map all those bytes to the same pin. Use the top left to ensure
			// the node pointer is always properly initialized.
			formatterAppendNTimes(
				m.map_,
				self.terminal.Screens.Active.Pages.GetTopLeft(PointTagScreen),
				discarding.writer.count,
			)
		}
	}
palette:

	// Emit terminal modes that differ from defaults. We probably have
	// some modes we want to emit before and some after, but for now for
	// simplicity we just emit them all before. If we make this more complex
	// later we should add test cases for it.
	if self.opts.Emit == FormatterFormatVt && self.extra.Modes {
		// inline for (@typeInfo(modespkg.Mode).@"enum".fields): the fields
		// built once at start-up (modes.go); field i is entry i, so
		// `@field(default, field.name)` is modesModePackedFields[i].
		for i, field := range modesModeFields {
			mode := ModesMode(field.value)
			current := self.terminal.Modes.Get(mode)
			default_val := *modesModePackedFields[i](&self.terminal.Modes.Default)

			if current != default_val {
				tag := ModesModeTagFromInt(ModesModeTagBacking(mode))
				prefix := "?"
				if tag.Ansi {
					prefix = ""
				}
				suffix := "l"
				if current {
					suffix = "h"
				}
				// writer.print("\x1b[{s}{d}{s}", .{ prefix, tag.value, suffix })
				if err := writer.writeAll([]uint8("\x1b[" + prefix + itoa(uint64(tag.Value)) + suffix)); err != nil {
					return err
				}
			}
		}

		// If we have a pin_map, add the bytes we wrote to map.
		if self.pin_map.Valid {
			m := &self.pin_map.Value
			discarding := oscWriterDiscardingInit(nil)
			extra_formatter := self
			extra_formatter.content = FormatterScreenFormatterContent{Tag: FormatterScreenFormatterContentNone}
			extra_formatter.pin_map = None[FormatterPinMap]()
			extra_formatter.extra = FormatterTerminalFormatterExtraNone
			extra_formatter.extra.Modes = true
			if err := extra_formatter.format(&discarding.writer); err != nil {
				return err
			}

			// Map all those bytes to the same pin. Use the top left to ensure
			// the node pointer is always properly initialized.
			formatterAppendNTimes(
				m.map_,
				self.terminal.Screens.Active.Pages.GetTopLeft(PointTagScreen),
				discarding.writer.count,
			)
		}
	}

	screen_formatter := FormatterScreenFormatterInit(self.terminal.Screens.Active, self.opts)
	screen_formatter.content = self.content
	screen_formatter.extra = self.extra.Screen
	screen_formatter.pin_map = self.pin_map
	if err := screen_formatter.format(writer); err != nil {
		return err
	}

	// Extra terminal state to emit after the screen contents so that
	// it doesn't impact the emitted contents.
	if self.opts.Emit == FormatterFormatVt {
		// Emit scrolling region using DECSTBM and DECSLRM
		if self.extra.ScrollingRegion {
			region := &self.terminal.ScrollingRegion

			// DECSTBM: top and bottom margins (1-indexed)
			// Only emit if not the full screen
			assert(self.terminal.Rows > 0) // Zig: `rows - 1` traps on 0
			if region.Top != 0 || region.Bottom != self.terminal.Rows-1 {
				// writer.print("\x1b[{d};{d}r", .{ region.top + 1, region.bottom + 1 })
				if err := writer.writeAll([]uint8("\x1b[" + itoa(uint64(formatterAddOneU16(region.Top))) + ";" +
					itoa(uint64(formatterAddOneU16(region.Bottom))) + "r")); err != nil {
					return err
				}
			}

			// DECSLRM: left and right margins (1-indexed)
			// Only emit if not the full width
			assert(self.terminal.Cols > 0) // Zig: `cols - 1` traps on 0
			if region.Left != 0 || region.Right != self.terminal.Cols-1 {
				// writer.print("\x1b[{d};{d}s", .{ region.left + 1, region.right + 1 })
				if err := writer.writeAll([]uint8("\x1b[" + itoa(uint64(formatterAddOneU16(region.Left))) + ";" +
					itoa(uint64(formatterAddOneU16(region.Right))) + "s")); err != nil {
					return err
				}
			}
		}

		// Emit tabstop positions
		if self.extra.Tabstops {
			// Clear all tabs (CSI 3 g)
			if err := writer.writeAll([]uint8("\x1b[3g")); err != nil {
				return err
			}

			// Set each configured tabstop by moving cursor and using HTS
			for col := uint(0); col < uint(self.terminal.Cols); col++ {
				if self.terminal.Tabstops.Get(col) {
					// Move cursor to the column (1-indexed)
					// writer.print("\x1b[{d}G", .{col + 1})
					if err := writer.writeAll([]uint8("\x1b[" + itoa(uint64(col+1)) + "G")); err != nil {
						return err
					}
					// Set tab (HTS)
					if err := writer.writeAll([]uint8("\x1bH")); err != nil {
						return err
					}
				}
			}
		}

		// Emit keyboard modes such as ModifyOtherKeys
		if self.extra.Keyboard {
			// Only emit if modify_other_keys_2 is true
			if self.terminal.Flags.ModifyOtherKeys2 {
				if err := writer.writeAll([]uint8("\x1b[>4;2m")); err != nil {
					return err
				}
			}
		}

		// Emit present working directory using OSC 7
		if self.extra.Pwd {
			pwd := self.terminal.Pwd
			if len(pwd) > 0 {
				// writer.print("\x1b]7;{s}\x1b\\", .{pwd})
				if err := writer.writeAll([]uint8("\x1b]7;" + string(pwd) + "\x1b\\")); err != nil {
					return err
				}
			}
		}

		// If we have a pin_map, add the bytes we wrote to map.
		if self.pin_map.Valid {
			m := &self.pin_map.Value
			discarding := oscWriterDiscardingInit(nil)
			extra_formatter := self
			extra_formatter.content = FormatterScreenFormatterContent{Tag: FormatterScreenFormatterContentNone}
			extra_formatter.pin_map = None[FormatterPinMap]()
			extra_formatter.extra = FormatterTerminalFormatterExtraNone
			extra_formatter.extra.ScrollingRegion = self.extra.ScrollingRegion
			extra_formatter.extra.Tabstops = self.extra.Tabstops
			extra_formatter.extra.Keyboard = self.extra.Keyboard
			extra_formatter.extra.Pwd = self.extra.Pwd
			if err := extra_formatter.format(&discarding.writer); err != nil {
				return err
			}

			var pin PageListPin
			if len(*m.map_) > 0 {
				last := (*m.map_)[len(*m.map_)-1]
				pin = PageListPin{
					Node: last.Node,
					X:    last.X,
					Y:    last.Y,
				}
			} else {
				pin = self.terminal.Screens.Active.Pages.GetTopLeft(PointTagScreen)
			}
			formatterAppendNTimes(m.map_, pin, discarding.writer.count)
		}
	}
	return nil
}

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

// formatterScreenFormatterContentTagNames are the tag names of
// FormatterScreenFormatterContentTag, indexed by value (Zig's @tagName).
var formatterScreenFormatterContentTagNames = [...]string{"none", "selection"}

// String returns the Zig tag name (@tagName).
func (self FormatterScreenFormatterContentTag) String() string {
	return formatterScreenFormatterContentTagNames[self]
}

// FormatterScreenFormatterExtra is formatter.ScreenFormatter.Extra.
//
// Zig: packed struct; no field has a default. In Go an ordinary struct;
// see the file header.
type FormatterScreenFormatterExtra struct {
	// Emit cursor position using CUP (CSI H).
	Cursor bool

	// Emit current SGR style state based on the cursor's active style_id.
	// This reconstructs the SGR attributes (bold, italic, colors, etc.) at
	// the cursor position.
	Style bool

	// Emit current hyperlink state using OSC 8 sequences.
	// This sets the active hyperlink based on cursor.hyperlink_id.
	Hyperlink bool

	// Emit character protection mode using DECSCA.
	Protection bool

	// Emit Kitty keyboard protocol state using CSI > u and CSI = sequences.
	KittyKeyboard bool

	// Emit character set designations and invocations.
	// This includes G0-G3 designations (ESC ( ) * +) and GL/GR invocations.
	Charsets bool
}

// FormatterScreenFormatterExtraNone is formatter.ScreenFormatter.Extra.none.
//
// Emit nothing.
var FormatterScreenFormatterExtraNone = FormatterScreenFormatterExtra{
	Cursor:        false,
	Style:         false,
	Hyperlink:     false,
	Protection:    false,
	KittyKeyboard: false,
	Charsets:      false,
}

// FormatterScreenFormatterExtraStyles is
// formatter.ScreenFormatter.Extra.styles.
//
// Emit style-relevant information only.
var FormatterScreenFormatterExtraStyles = FormatterScreenFormatterExtra{
	Cursor:        false,
	Style:         true,
	Hyperlink:     true,
	Protection:    false,
	KittyKeyboard: false,
	Charsets:      false,
}

// FormatterScreenFormatterExtraAll is formatter.ScreenFormatter.Extra.all.
//
// Emit everything. This reconstructs the screen state as closely
// as possible.
var FormatterScreenFormatterExtraAll = FormatterScreenFormatterExtra{
	Cursor:        true,
	Style:         true,
	Hyperlink:     true,
	Protection:    true,
	KittyKeyboard: true,
	Charsets:      true,
}

// isSet is formatter.ScreenFormatter.Extra.isSet (private in Zig).
//
// Zig: fn isSet(self: Extra) bool, true when the packed struct's backing
// integer is non-zero, i.e. when any field is true. Zig's `E.isSet(.none)`
// is `FormatterScreenFormatterExtraNone.isSet()`.
func (self FormatterScreenFormatterExtra) isSet() bool {
	// const Int = @typeInfo(Extra).@"struct".backing_integer.?;
	// const v: Int = @bitCast(self);
	// return v != 0;
	return self.Cursor || self.Style || self.Hyperlink || self.Protection ||
		self.KittyKeyboard || self.Charsets
}

// FormatterScreenFormatterInit is formatter.ScreenFormatter.init.
//
// Zig: pub fn init(screen: *const Screen, opts: Options) ScreenFormatter.
// It sets `content` to `.{ .selection = null }`, `extra` to `.none` and
// `pin_map` to null.
func FormatterScreenFormatterInit(screen *Screen, opts FormatterOptions) FormatterScreenFormatter {
	return FormatterScreenFormatter{
		screen: screen,
		opts:   opts,
		content: FormatterScreenFormatterContent{
			Tag:       FormatterScreenFormatterContentSelection,
			Selection: None[Selection](),
		},
		extra:   FormatterScreenFormatterExtraNone,
		pin_map: None[FormatterPinMap](),
	}
}

// Format is formatter.ScreenFormatter.format.
//
// Zig: pub fn format(self: ScreenFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The writer is the stand-in oscWriter (osc.go);
// the error is ErrWriteFailed (osc.go), also returned when appending to the
// pin map fails. See the file header for why this is not `Format() string`.
func (self FormatterScreenFormatter) format(writer *oscWriter) error {
	switch self.content.Tag {
	case FormatterScreenFormatterContentNone:

	case FormatterScreenFormatterContentSelection:
		selection_ := self.content.Selection
		// Emit our pagelist contents according to our selection.
		list_formatter := FormatterPageListFormatterInit(&self.screen.Pages, self.opts)
		list_formatter.pin_map = self.pin_map
		if selection_.Valid {
			sel := selection_.Value
			list_formatter.top_left = Some(sel.TopLeft(self.screen))
			list_formatter.bottom_right = Some(sel.BottomRight(self.screen))
			list_formatter.rectangle = sel.Rectangle
		}
		if err := list_formatter.format(writer); err != nil {
			return err
		}

	default:
		panic("unreachable")
	}

	// Emit extra screen state after content if we care. The state has
	// to be emitted after since some state such as cursor position and
	// style are impacted by content rendering.
	switch self.opts.Emit {
	case FormatterFormatPlain:
		return nil
	case FormatterFormatVt:
		if !self.extra.isSet() {
			return nil
		}

	// HTML doesn't preserve any screen state because it has
	// nothing to do with rendering.
	case FormatterFormatHtml:
		return nil

	default:
		panic("unreachable")
	}

	// Emit current SGR style state
	if self.extra.Style {
		cursor := &self.screen.Cursor
		// writer.print("{f}", .{cursor.style.formatterVt()})
		if err := writer.writeAll([]uint8(cursor.Style.FormatterVt().Format())); err != nil {
			return err
		}
	}

	// Emit current hyperlink state using OSC 8
	if self.extra.Hyperlink {
		cursor := &self.screen.Cursor
		if link := cursor.Hyperlink; link != nil {
			// Start hyperlink with uri (and explicit id if present)
			switch link.ID.Tag {
			case HyperlinkHyperlinkIdExplicit:
				id := link.ID.Explicit
				// writer.print(
				//     "\x1b]8;id={s};{s}\x1b\\",
				//     .{ id, link.uri },
				// )
				if err := writer.writeAll([]uint8("\x1b]8;id=" + string(id) + ";" + string(link.URI) + "\x1b\\")); err != nil {
					return err
				}
			case HyperlinkHyperlinkIdImplicit:
				// writer.print(
				//     "\x1b]8;;{s}\x1b\\",
				//     .{link.uri},
				// )
				if err := writer.writeAll([]uint8("\x1b]8;;" + string(link.URI) + "\x1b\\")); err != nil {
					return err
				}
			default:
				panic("unreachable")
			}
		}
	}

	// Emit character protection mode using DECSCA
	if self.extra.Protection {
		cursor := &self.screen.Cursor
		if cursor.Protected {
			// DEC protected mode
			if err := writer.writeAll([]uint8("\x1b[1\"q")); err != nil {
				return err
			}
		}
	}

	// Emit Kitty keyboard protocol state using CSI = u
	if self.extra.KittyKeyboard {
		current_flags := self.screen.KittyKeyboard.Current()
		if current_flags.Int() != KittyKeyFlagsDisabled.Int() {
			flags := current_flags.Int()
			// writer.print("\x1b[={d};1u", .{flags})
			if err := writer.writeAll([]uint8("\x1b[=" + itoa(uint64(flags)) + ";1u")); err != nil {
				return err
			}
		}
	}

	// Emit character set designations and invocations
	if self.extra.Charsets {
		charset := &self.screen.Charset

		// Emit G0-G3 designations
		for _, slot := range formatterCharsetsSlotsValues {
			cs := charset.charsets.Get(slot)
			if cs != CharsetsCharsetUtf8 { // Only emit non-default charsets
				var intermediate uint8
				switch slot {
				case CharsetsSlotsG0:
					intermediate = '('
				case CharsetsSlotsG1:
					intermediate = ')'
				case CharsetsSlotsG2:
					intermediate = '*'
				case CharsetsSlotsG3:
					intermediate = '+'
				default:
					panic("unreachable")
				}
				var final uint8
				switch cs {
				case CharsetsCharsetAscii:
					final = 'B'
				case CharsetsCharsetBritish:
					final = 'A'
				case CharsetsCharsetDecSpecial:
					final = '0'
				default:
					continue
				}
				// writer.print("\x1b{c}{c}", .{ intermediate, final })
				if err := writer.writeAll([]uint8{0x1b, intermediate, final}); err != nil {
					return err
				}
			}
		}

		// Emit GL invocation if not G0
		if charset.Gl != CharsetsSlotsG0 {
			var seq string
			switch charset.Gl {
			case CharsetsSlotsG0:
				panic("unreachable")
			case CharsetsSlotsG1:
				seq = "\x0e" // SO - Shift Out
			case CharsetsSlotsG2:
				seq = "\x1bn" // LS2
			case CharsetsSlotsG3:
				seq = "\x1bo" // LS3
			default:
				panic("unreachable")
			}
			// writer.print("{s}", .{seq})
			if err := writer.writeAll([]uint8(seq)); err != nil {
				return err
			}
		}

		// Emit GR invocation if not G2
		if charset.Gr != CharsetsSlotsG2 {
			var seq string
			switch charset.Gr {
			case CharsetsSlotsG0:
				panic("unreachable") // GR can't be G0
			case CharsetsSlotsG1:
				seq = "\x1b~" // LS1R
			case CharsetsSlotsG2:
				panic("unreachable")
			case CharsetsSlotsG3:
				seq = "\x1b|" // LS3R
			default:
				panic("unreachable")
			}
			// writer.print("{s}", .{seq})
			if err := writer.writeAll([]uint8(seq)); err != nil {
				return err
			}
		}
	}

	// Emit cursor position using CUP (CSI H)
	if self.extra.Cursor {
		cursor := &self.screen.Cursor
		// CUP is 1-indexed
		// writer.print("\x1b[{d};{d}H", .{ cursor.y + 1, cursor.x + 1 })
		if err := writer.writeAll([]uint8("\x1b[" + itoa(uint64(formatterAddOneU16(cursor.Y))) + ";" +
			itoa(uint64(formatterAddOneU16(cursor.X))) + "H")); err != nil {
			return err
		}
	}

	// If we have a pin_map, we need to count how many bytes the extras
	// will emit so we can map them all to the same pin. We do this by
	// formatting to a discarding writer with content=none.
	if self.pin_map.Valid {
		m := &self.pin_map.Value
		discarding := oscWriterDiscardingInit(nil)
		extra_formatter := self
		extra_formatter.content = FormatterScreenFormatterContent{Tag: FormatterScreenFormatterContentNone}
		extra_formatter.pin_map = None[FormatterPinMap]()
		if err := extra_formatter.format(&discarding.writer); err != nil {
			return err
		}

		// Map all those bytes to the same pin. Use the first page node
		// to ensure the node pointer is always properly initialized.
		var pin PageListPin
		if len(*m.map_) > 0 {
			// There is a weird Zig miscompilation here on 0.15.2.
			// If I return the m.map.items value directly then we
			// get undefined memory (even though we're copying a
			// Pin struct). If we duplicate here like this we do
			// not.
			last := (*m.map_)[len(*m.map_)-1]
			pin = PageListPin{
				Node: last.Node,
				X:    last.X,
				Y:    last.Y,
			}
		} else {
			pin = self.screen.Pages.GetTopLeft(PointTagScreen)
		}
		formatterAppendNTimes(m.map_, pin, discarding.writer.count)
	}
	return nil
}

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
	return FormatterPageListFormatter{
		list:         list,
		opts:         opts,
		top_left:     None[PageListPin](),
		bottom_right: None[PageListPin](),
		rectangle:    false,
		pin_map:      None[FormatterPinMap](),
	}
}

// Format is formatter.PageListFormatter.format.
//
// Zig: pub fn format(self: PageListFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The writer is the stand-in oscWriter (osc.go);
// the error is ErrWriteFailed (osc.go), also returned when appending to the
// pin map fails. See the file header for why this is not `Format() string`.
func (self FormatterPageListFormatter) format(writer *oscWriter) error {
	var tl PageListPin
	if self.top_left.Valid {
		tl = self.top_left.Value
	} else {
		tl = self.list.GetTopLeft(PointTagScreen)
	}
	var br PageListPin
	if self.bottom_right.Valid {
		br = self.bottom_right.Value
	} else {
		br_ := self.list.GetBottomRight(PointTagScreen)
		assert(br_.Valid) // .?
		br = br_.Value
	}

	// If we keep track of pins, we'll need this.
	var point_map []PointCoordinate // std.ArrayList(Coordinate) = .empty
	// Zig: defer if (self.pin_map) |*m| point_map.deinit(m.alloc); Go memory.

	page_state := None[FormatterPageFormatterTrailingState]()
	iter := tl.PageIterator(PageListDirectionRightDown, Some(br))
	for {
		chunk_ := iter.Next()
		if !chunk_.Valid {
			break
		}
		chunk := chunk_.Value
		assert(chunk.start < chunk.end)
		assert(chunk.end > 0)

		formatter := FormatterPageFormatterInit(&chunk.node.data, self.opts)
		formatter.start_y = chunk.start
		formatter.end_y = Some(chunk.end - 1)
		formatter.trailing_state = page_state
		formatter.rectangle = self.rectangle

		// For rectangle selection, apply start_x and end_x to all chunks
		if self.rectangle {
			formatter.start_x = tl.X
			formatter.end_x = Some(br.X)
		} else {
			// Otherwise only on the first/last, respectively.
			if chunk.node == tl.Node {
				formatter.start_x = tl.X
			}
			if chunk.node == br.Node {
				formatter.end_x = Some(br.X)
			}
		}

		// If we're tracking pins, then we setup a point map for the
		// page formatter (cause it can't track pins). And then we convert
		// this to pins later.
		if self.pin_map.Valid {
			point_map = point_map[:0]
			formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})
		}

		state, err := formatter.formatWithState(writer)
		if err != nil {
			return err
		}
		page_state = Some(state)

		// If we're tracking pins then grab our points and write them
		// to our pin map.
		if self.pin_map.Valid {
			m := &self.pin_map.Value
			for _, coord := range point_map {
				assert(coord.Y <= 0xFFFF) // Zig: @intCast(coord.y) to u16 traps
				*m.map_ = append(*m.map_, PageListPin{
					Node: chunk.node,
					X:    coord.X,
					Y:    SizeCellCountInt(coord.Y),
				})
			}
		}
	}
	return nil
}

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
	return FormatterPageFormatter{
		page:           page,
		opts:           opts,
		start_x:        0,
		start_y:        0,
		end_x:          None[SizeCellCountInt](),
		end_y:          None[SizeCellCountInt](),
		rectangle:      false,
		point_map:      None[FormatterPageFormatterPointMap](),
		trailing_state: None[FormatterPageFormatterTrailingState](),
	}
}

// Format is formatter.PageFormatter.format.
//
// Zig: pub fn format(self: PageFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void: formatWithState with the state discarded. The
// writer is the stand-in oscWriter (osc.go); the error is ErrWriteFailed
// (osc.go). See the file header for why this is not `Format() string`.
func (self FormatterPageFormatter) format(writer *oscWriter) error {
	_, err := self.formatWithState(writer)
	return err
}

// FormatWithState is formatter.PageFormatter.formatWithState.
//
// Zig: pub fn formatWithState(self: PageFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!TrailingState. The writer is the stand-in oscWriter
// (osc.go); the error is ErrWriteFailed (osc.go), also returned when
// appending to the point map fails.
func (self FormatterPageFormatter) formatWithState(writer *oscWriter) (FormatterPageFormatterTrailingState, error) {
	var blank_rows uint = 0
	var blank_cells uint = 0

	// Continue our prior trailing state if we have it, but only if we're
	// starting from the beginning (start_y and start_x are both 0).
	// If a non-zero start position is specified, ignore trailing state.
	if self.trailing_state.Valid {
		state := self.trailing_state.Value
		if self.start_y == 0 && self.start_x == 0 {
			blank_rows = state.rows
			blank_cells = state.cells
		}
	}

	// Setup our starting column and perform some validation for overflows.
	// Note: start_x only applies to the first row, end_x only applies to the last row.
	start_x := self.start_x
	if start_x >= self.page.size.cols {
		return FormatterPageFormatterTrailingState{rows: blank_rows, cells: blank_cells}, nil
	}
	// cols > start_x >= 0 here, so `cols - 1` cannot underflow.
	var end_x_unclamped SizeCellCountInt
	if self.end_x.Valid {
		end_x_unclamped = self.end_x.Value
	} else {
		end_x_unclamped = self.page.size.cols - 1
	}
	end_x := min(end_x_unclamped, self.page.size.cols-1)

	// Setup our starting row and perform some validation for overflows.
	start_y := self.start_y
	if start_y >= self.page.size.rows {
		return FormatterPageFormatterTrailingState{rows: blank_rows, cells: blank_cells}, nil
	}
	// rows > start_y >= 0 here, so `rows - 1` cannot underflow.
	var end_y_unclamped SizeCellCountInt
	if self.end_y.Valid {
		end_y_unclamped = self.end_y.Value
	} else {
		end_y_unclamped = self.page.size.rows - 1
	}
	if start_y > end_y_unclamped {
		return FormatterPageFormatterTrailingState{rows: blank_rows, cells: blank_cells}, nil
	}
	end_y := min(end_y_unclamped, self.page.size.rows-1)

	// Edge case: if our end x/y falls on a spacer head AND we're unwrapping,
	// then we move the x/y to the start of the next row (if available).
	if self.opts.Unwrap && !self.rectangle {
		final_row := self.page.GetRow(uint(end_y))
		cells := self.page.GetCells(final_row)
		switch cells[end_x].wide {
		case PageCellWideSpacerHead:
			// Move to next row if available
			//
			// TODO: if unavailable, we should add to our trailing state
			//
			// so the pagelist formatter can be aware and maybe add
			// another page
			if end_y < self.page.size.rows-1 {
				end_y += 1
				end_x = 0
			}

		default:
		}
	}

	// If we only have a single row, validate that start_x <= end_x
	if start_y == end_y && start_x > end_x {
		return FormatterPageFormatterTrailingState{rows: blank_rows, cells: blank_cells}, nil
	}

	// Wrap HTML output in monospace font styling
	switch self.opts.Emit {
	case FormatterFormatPlain:

	case FormatterFormatHtml:
		// Setup our div. We use a buffer here that should always
		// fit the stuff we need, in order to make counting bytes easier.
		var buf [1024]uint8
		stream := oscWriterFixed(buf[:]) // std.io.fixedBufferStream(&buf)
		buf_writer := &stream

		// Monospace and whitespace preserving
		if err := buf_writer.writeAll([]uint8("<div style=\"font-family: monospace; white-space: pre;")); err != nil {
			return FormatterPageFormatterTrailingState{}, ErrWriteFailed
		}

		// Background/foreground colors
		if self.opts.Background.Valid {
			bg := self.opts.Background.Value
			// buf_writer.print(
			//     "background-color: #{x:0>2}{x:0>2}{x:0>2};",
			//     .{ bg.r, bg.g, bg.b },
			// )
			if err := buf_writer.writeAll([]uint8("background-color: #" +
				formatterHex2(bg.R) + formatterHex2(bg.G) + formatterHex2(bg.B) + ";")); err != nil {
				return FormatterPageFormatterTrailingState{}, ErrWriteFailed
			}
		}
		if self.opts.Foreground.Valid {
			fg := self.opts.Foreground.Value
			// buf_writer.print(
			//     "color: #{x:0>2}{x:0>2}{x:0>2};",
			//     .{ fg.r, fg.g, fg.b },
			// )
			if err := buf_writer.writeAll([]uint8("color: #" +
				formatterHex2(fg.R) + formatterHex2(fg.G) + formatterHex2(fg.B) + ";")); err != nil {
				return FormatterPageFormatterTrailingState{}, ErrWriteFailed
			}
		}

		if err := buf_writer.writeAll([]uint8("\">")); err != nil {
			return FormatterPageFormatterTrailingState{}, ErrWriteFailed
		}

		header := stream.buffered() // stream.getWritten()
		if err := writer.writeAll(header); err != nil {
			return FormatterPageFormatterTrailingState{}, err
		}
		if self.point_map.Valid {
			map_ := &self.point_map.Value
			formatterAppendNTimes(
				map_.map_,
				PointCoordinate{X: 0, Y: 0},
				uint64(len(header)),
			)
		}

	case FormatterFormatVt:
		// OSC 10 sets foreground color, OSC 11 sets background color
		var buf [512]uint8
		stream := oscWriterFixed(buf[:]) // std.io.fixedBufferStream(&buf)
		buf_writer := &stream
		if self.opts.Foreground.Valid {
			fg := self.opts.Foreground.Value
			// buf_writer.print(
			//     "\x1b]10;rgb:{x:0>2}/{x:0>2}/{x:0>2}\x1b\\",
			//     .{ fg.r, fg.g, fg.b },
			// )
			if err := buf_writer.writeAll([]uint8("\x1b]10;rgb:" +
				formatterHex2(fg.R) + "/" + formatterHex2(fg.G) + "/" + formatterHex2(fg.B) +
				"\x1b\\")); err != nil {
				return FormatterPageFormatterTrailingState{}, ErrWriteFailed
			}
		}
		if self.opts.Background.Valid {
			bg := self.opts.Background.Value
			// buf_writer.print(
			//     "\x1b]11;rgb:{x:0>2}/{x:0>2}/{x:0>2}\x1b\\",
			//     .{ bg.r, bg.g, bg.b },
			// )
			if err := buf_writer.writeAll([]uint8("\x1b]11;rgb:" +
				formatterHex2(bg.R) + "/" + formatterHex2(bg.G) + "/" + formatterHex2(bg.B) +
				"\x1b\\")); err != nil {
				return FormatterPageFormatterTrailingState{}, ErrWriteFailed
			}
		}

		header := stream.buffered() // stream.getWritten()
		if err := writer.writeAll(header); err != nil {
			return FormatterPageFormatterTrailingState{}, err
		}
		if self.point_map.Valid {
			map_ := &self.point_map.Value
			formatterAppendNTimes(
				map_.map_,
				PointCoordinate{X: 0, Y: 0},
				uint64(len(header)),
			)
		}

	default:
		panic("unreachable")
	}

	// Our style for non-plain formats
	var style Style // .{}

	// Track hyperlink state for HTML output. We need to close </a> tags
	// when the hyperlink changes or ends.
	current_hyperlink_id := None[HyperlinkID]()

	// end_y < rows <= maxInt(u16), so `end_y + 1` cannot overflow.
	for y_usize := uint(start_y); y_usize < uint(end_y+1); y_usize++ {
		y := SizeCellCountInt(y_usize) // @intCast: y_usize <= end_y
		row := self.page.GetRow(uint(y))
		cells := self.page.GetCells(row)

		// Determine the x range for this row
		// - First row: start_x to end of row (or end_x if single row)
		// - Last row: start of row to end_x
		// - Middle rows: full width
		//
		// cells_subset: (labeled block; its `continue` continues the
		// row loop)

		// The end is always straightforward
		var row_end_x SizeCellCountInt
		if self.rectangle || y == end_y {
			row_end_x = end_x + 1
		} else {
			row_end_x = self.page.size.cols
		}

		// The first we have to check if our start X falls on the
		// tail of a wide character.
		var row_start_x SizeCellCountInt
		if start_x > 0 &&
			(self.rectangle || y == start_y) {
			// start_x:
			switch cells[start_x].wide {
			// Include the prior cell to get the full wide char
			case PageCellWideSpacerTail:
				row_start_x = start_x - 1

			// If we're a spacer head on our first row then we
			// skip this whole row.
			case PageCellWideSpacerHead:
				continue

			case PageCellWideNarrow, PageCellWideWide:
				row_start_x = start_x

			default:
				panic("unreachable")
			}
		} else {
			row_start_x = 0
		}

		// Zig traps when row_start_x > row_end_x (a rectangle with
		// start_x > end_x); so does Go's slice expression.
		cells_subset := cells[row_start_x:row_end_x]

		// If this row is blank, accumulate to avoid a bunch of extra
		// work later. If it isn't blank, make sure we dump all our
		// blanks.
		if !PageCellHasTextAny(cells_subset) {
			blank_rows += 1
			continue
		}

		if blank_rows > 0 {
			// Reset style before emitting newlines to prevent background
			// colors from bleeding into the next line's leading cells.
			if !style.Default() {
				if err := self.formatStyleClose(writer); err != nil {
					return FormatterPageFormatterTrailingState{}, err
				}
				style = Style{}
			}

			var sequence string
			switch self.opts.Emit {
			// Plaintext just uses standard newlines because newlines
			// on their own usually move the cursor back in anywhere
			// you type plaintext.
			case FormatterFormatPlain:
				sequence = "\n"

			// VT uses \r\n because in a raw pty, \n alone doesn't
			// guarantee moving the cursor back to column 0. \r
			// makes it work for sure.
			case FormatterFormatVt:
				sequence = "\r\n"

			// HTML uses just \n because HTML rendering will move
			// the cursor back.
			case FormatterFormatHtml:
				sequence = "\n"

			default:
				panic("unreachable")
			}

			for i := uint(0); i < blank_rows; i++ {
				if err := writer.writeAll([]uint8(sequence)); err != nil {
					return FormatterPageFormatterTrailingState{}, err
				}
			}

			// \r and \n map to the row that ends with this newline.
			// If we're continuing (trailing state) then this will be
			// in a prior page, so we just map to the first row of this
			// page.
			if self.point_map.Valid {
				map_ := &self.point_map.Value
				var start PointCoordinate
				if len(*map_.map_) > 0 {
					start = (*map_.map_)[len(*map_.map_)-1]
				} else {
					start = PointCoordinate{X: 0, Y: 0}
				}

				// The first one inherits the x value.
				formatterAppendNTimes(
					map_.map_,
					PointCoordinate{X: start.X, Y: start.Y},
					uint64(len(sequence)),
				)

				// All others have x = 0 since they reference their prior
				// blank line.
				for y_offset_usize := uint(1); y_offset_usize < blank_rows; y_offset_usize++ {
					assert(y_offset_usize <= 0xFFFF) // Zig: @intCast to u16 traps
					y_offset := SizeCellCountInt(y_offset_usize)
					// Zig: `start.y + y_offset` is a u32 addition that traps
					// on overflow.
					assert(uint64(start.Y)+uint64(y_offset) <= 0xFFFFFFFF)
					formatterAppendNTimes(
						map_.map_,
						PointCoordinate{X: 0, Y: start.Y + uint32(y_offset)},
						uint64(len(sequence)),
					)
				}
			}

			blank_rows = 0
		}

		// If we're not wrapped, we always add a newline so after
		// the row is printed we can add a newline.
		if !row.wrap || !self.opts.Unwrap {
			blank_rows += 1
		}

		// If the row doesn't continue a wrap then we need to reset
		// our blank cell count.
		if !row.wrap_continuation || !self.opts.Unwrap {
			blank_cells = 0
		}

		// Go through each cell and print it
		for i := range cells_subset {
			cell := &cells_subset[i]
			x_usize := uint(row_start_x) + uint(i)
			x := SizeCellCountInt(x_usize) // @intCast: x_usize < cols

			// Skip spacers. These happen naturally when wide characters
			// are printed again on the screen (for well-behaved terminals!)
			switch cell.wide {
			case PageCellWideNarrow, PageCellWideWide:
			case PageCellWideSpacerHead, PageCellWideSpacerTail:
				continue
			default:
				panic("unreachable")
			}

			// If we have a zero value, then we accumulate a counter. We
			// only want to turn zero values into spaces if we have a non-zero
			// char sometime later.
			{
				// If we're emitting styled output (not plaintext) and
				// the cell has some kind of styling or is not empty
				// then this isn't blank.
				if self.opts.Emit.Styled() &&
					(!cell.IsEmpty() || cell.HasStyling()) {
					goto blank // break :blank
				}

				// Cells with no text are blank
				if !cell.HasText() {
					blank_cells += 1
					continue
				}

				// Trailing spaces are blank. We know it is trailing
				// because if we get a non-empty cell later we'll
				// fill the blanks.
				if cell.Codepoint() == ' ' && self.opts.Trim {
					blank_cells += 1
					continue
				}
			}
		blank:

			// This cell is not blank. If we have accumulated blank cells
			// then we want to emit them now.
			if blank_cells > 0 {
				if err := writer.splatByteAll(' ', int(blank_cells)); err != nil {
					return FormatterPageFormatterTrailingState{}, err
				}

				if self.point_map.Valid {
					map_ := &self.point_map.Value
					// Map each blank cell to its coordinate. Blank cells can span
					// multiple rows if they carry over from wrap continuation.
					remaining_blanks := blank_cells
					blank_x := x
					blank_y := y
					for ; remaining_blanks > 0; remaining_blanks -= 1 {
						if blank_x > 0 {
							// We have space in this row
							blank_x -= 1
						} else if blank_y > 0 {
							// Wrap to previous row
							blank_y -= 1
							blank_x = self.page.size.cols - 1
						} else {
							// Can't go back further, just use (0, 0)
							blank_x = 0
							blank_y = 0
						}

						*map_.map_ = append(
							*map_.map_,
							PointCoordinate{X: blank_x, Y: uint32(blank_y)},
						)
					}
				}

				blank_cells = 0
			}

			{
				// If we aren't emitting styled output then we don't
				// have to worry about styles.
				if !self.opts.Emit.Styled() {
					goto style // break :style
				}

				// Get our cell style.
				cell_style := self.cellStyle(cell)

				// If the style hasn't changed, don't bloat output.
				if cell_style.Eql(style) {
					goto style // break :style
				}

				// If we had a previous style, we need to close it,
				// because we've confirmed we have some new style
				// (which is maybe default).
				if !style.Default() {
					switch self.opts.Emit {
					case FormatterFormatHtml:
						if err := self.formatStyleClose(writer); err != nil {
							return FormatterPageFormatterTrailingState{}, err
						}

					// For VT, we only close if we're switching to a default
					// style because any non-default style will emit
					// a \x1b[0m as the start of a VT coloring sequence.
					case FormatterFormatVt:
						if cell_style.Default() {
							if err := self.formatStyleClose(writer); err != nil {
								return FormatterPageFormatterTrailingState{}, err
							}
						}

					// Unreachable because of the styled() check at the
					// top of this block.
					case FormatterFormatPlain:
						panic("unreachable")

					default:
						panic("unreachable")
					}
				}

				// At this point, we can copy our style over
				style = cell_style

				// If we're just the default style now, we're done.
				if cell_style.Default() {
					goto style // break :style
				}

				// New style, emit it.
				if err := self.formatStyleOpen(
					writer,
					&style,
				); err != nil {
					return FormatterPageFormatterTrailingState{}, err
				}

				// If we have a point map, we map the style to
				// this cell.
				if self.point_map.Valid {
					map_ := &self.point_map.Value
					discarding := oscWriterDiscardingInit(nil)
					if err := self.formatStyleOpen(
						&discarding.writer,
						&style,
					); err != nil {
						return FormatterPageFormatterTrailingState{}, err
					}
					for j := uint64(0); j < discarding.writer.count; j++ {
						*map_.map_ = append(*map_.map_, PointCoordinate{
							X: x,
							Y: uint32(y),
						})
					}
				}
			}
		style:

			// Hyperlink state
			{
				// We currently only emit hyperlinks for HTML. In the
				// future we can support emitting OSC 8 hyperlinks for
				// VT output as well.
				if self.opts.Emit != FormatterFormatHtml {
					goto hyperlink // break :hyperlink
				}

				// Get the hyperlink ID. This ID is our internal ID,
				// not necessarily the OSC8 ID.
				var link_id_ Optional[HyperlinkID]
				if cell.hyperlink {
					link_id_ = self.page.LookupHyperlink(cell)
				} else {
					link_id_ = None[HyperlinkID]()
				}

				// If our hyperlink IDs match (even null) then we have
				// identical hyperlink state and we do nothing.
				if current_hyperlink_id.Valid == link_id_.Valid &&
					(!link_id_.Valid || current_hyperlink_id.Value == link_id_.Value) {
					goto hyperlink // break :hyperlink
				}

				// If our prior hyperlink ID was non-null, we need to
				// close it because the ID has changed.
				if current_hyperlink_id.Valid {
					if err := self.formatHyperlinkClose(writer); err != nil {
						return FormatterPageFormatterTrailingState{}, err
					}
					current_hyperlink_id = None[HyperlinkID]()
				}

				// Set our current hyperlink ID
				if !link_id_.Valid {
					goto hyperlink // orelse break :hyperlink
				}
				link_id := link_id_.Value
				current_hyperlink_id = Some(link_id)

				// Emit the opening hyperlink tag
				// uri:
				link := self.page.hyperlink_set.Get(link_id)
				uri := link.uri
				if err := self.formatHyperlinkOpen(
					writer,
					uri,
				); err != nil {
					return FormatterPageFormatterTrailingState{}, err
				}

				// If we have a point map, we map the hyperlink to
				// this cell.
				if self.point_map.Valid {
					map_ := &self.point_map.Value
					discarding := oscWriterDiscardingInit(nil)
					if err := self.formatHyperlinkOpen(
						&discarding.writer,
						uri,
					); err != nil {
						return FormatterPageFormatterTrailingState{}, err
					}
					for j := uint64(0); j < discarding.writer.count; j++ {
						*map_.map_ = append(*map_.map_, PointCoordinate{
							X: x,
							Y: uint32(y),
						})
					}
				}
			}
		hyperlink:

			switch cell.content_tag {
			// We combine codepoint and graphemes because both have
			// shared style handling. We use comptime to dup it.
			case PageCellContentTagCodepoint, PageCellContentTagCodepointGrapheme:
				tag := cell.content_tag
				if err := self.writeCell(tag, writer, cell); err != nil {
					return FormatterPageFormatterTrailingState{}, err
				}

				// If we have a point map, all codepoints map to this
				// cell.
				if self.point_map.Valid {
					map_ := &self.point_map.Value
					discarding := oscWriterDiscardingInit(nil)
					if err := self.writeCell(tag, &discarding.writer, cell); err != nil {
						return FormatterPageFormatterTrailingState{}, err
					}
					for j := uint64(0); j < discarding.writer.count; j++ {
						*map_.map_ = append(*map_.map_, PointCoordinate{
							X: x,
							Y: uint32(y),
						})
					}
				}

			// Cells with only background color (no text). Emit a space
			// with the appropriate background color SGR sequence.
			case PageCellContentTagBgColorPalette, PageCellContentTagBgColorRgb:
				if err := writer.writeByte(' '); err != nil {
					return FormatterPageFormatterTrailingState{}, err
				}
				if self.point_map.Valid {
					map_ := &self.point_map.Value
					*map_.map_ = append(
						*map_.map_,
						PointCoordinate{X: x, Y: uint32(y)},
					)
				}

			default:
				panic("unreachable")
			}
		}
	}

	// If the style is non-default, we need to close our style tag.
	if !style.Default() {
		if err := self.formatStyleClose(writer); err != nil {
			return FormatterPageFormatterTrailingState{}, err
		}
	}

	// Close any open hyperlink for HTML output
	if current_hyperlink_id.Valid {
		if err := self.formatHyperlinkClose(writer); err != nil {
			return FormatterPageFormatterTrailingState{}, err
		}
	}

	// Close the monospace wrapper for HTML output
	if self.opts.Emit == FormatterFormatHtml {
		const closing = "</div>"
		if err := writer.writeAll([]uint8(closing)); err != nil {
			return FormatterPageFormatterTrailingState{}, err
		}
		if self.point_map.Valid {
			map_ := &self.point_map.Value
			// ensureUnusedCapacity + appendNTimesAssumeCapacity. An empty
			// map traps in Zig (`items.len - 1`); Go's index panics.
			formatterAppendNTimes(
				map_.map_,
				(*map_.map_)[len(*map_.map_)-1],
				uint64(len(closing)),
			)
		}
	}

	return FormatterPageFormatterTrailingState{rows: blank_rows, cells: blank_cells}, nil
}

// writeCell is formatter.PageFormatter.writeCell.
//
// Zig: fn writeCell(self: PageFormatter, comptime tag: Cell.ContentTag,
// writer: *std.Io.Writer, cell: *const Cell) !void. The comptime parameter
// is an ordinary parameter; Zig instantiates it only for `.codepoint` and
// `.codepoint_grapheme`. The inferred error set is ErrWriteFailed
// (osc.go).
func (self FormatterPageFormatter) writeCell(tag PageCellContentTag, writer *oscWriter, cell *PageCell) error {
	// Blank cells get an empty space that isn't replaced by anything
	// because it isn't really a space. We do this so that formatting
	// is preserved if we're emitting styles.
	if !cell.HasText() {
		return writer.writeByte(' ')
	}

	if err := self.writeCodepointWithReplacement(writer, cell.content.codepoint); err != nil {
		return err
	}
	if tag == PageCellContentTagCodepointGrapheme {
		cps := self.page.LookupGrapheme(cell)
		assert(cps.Valid) // .?
		for _, cp := range cps.Value {
			if err := self.writeCodepointWithReplacement(writer, cp); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeCodepointWithReplacement is
// formatter.PageFormatter.writeCodepointWithReplacement.
//
// Zig: fn writeCodepointWithReplacement(self: PageFormatter,
// writer: *std.Io.Writer, codepoint: u21) !void. The inferred error set is
// ErrWriteFailed (osc.go).
func (self FormatterPageFormatter) writeCodepointWithReplacement(writer *oscWriter, codepoint uint32) error {
	// Search for our replacement
	//
	// replacement: (labeled block; each `break :replacement` sets r_ and
	// leaves the search)
	r_ := None[FormatterCodepointMapReplacement]()
	if self.opts.CodepointMap.Valid {
		map_ := self.opts.CodepointMap.Value
		for forward_i := 0; forward_i < len(map_); forward_i++ {
			i := len(map_) - forward_i - 1
			range_ := map_[i].range_
			if range_[0] <= codepoint && codepoint <= range_[1] {
				r_ = Some(map_[i].replacement)
				break
			}
		}
	}

	// If no replacement, write it directly.
	if !r_.Valid {
		return self.writeCodepoint(
			writer,
			codepoint,
		)
	}
	r := r_.Value

	switch r.Tag {
	case FormatterCodepointMapReplacementCodepoint:
		v := r.Codepoint
		return self.writeCodepoint(
			writer,
			v,
		)

	case FormatterCodepointMapReplacementString:
		s := r.String
		// std.unicode.Utf8View.init(s) catch unreachable
		if !utf8.Valid(s) {
			panic("unreachable")
		}
		for i := 0; i < len(s); {
			cp, size := utf8.DecodeRune(s[i:])
			i += size
			if err := self.writeCodepoint(
				writer,
				uint32(cp),
			); err != nil {
				return err
			}
		}

	default:
		panic("unreachable")
	}
	return nil
}

// writeCodepoint is formatter.PageFormatter.writeCodepoint.
//
// Zig: fn writeCodepoint(self: PageFormatter, writer: *std.Io.Writer,
// codepoint: u21) !void. The inferred error set is ErrWriteFailed
// (osc.go).
func (self FormatterPageFormatter) writeCodepoint(writer *oscWriter, codepoint uint32) error {
	switch self.opts.Emit {
	case FormatterFormatPlain, FormatterFormatVt:
		// writer.print("{u}", .{codepoint})
		return writer.printUnicodeCodepoint(codepoint)
	case FormatterFormatHtml:
		switch codepoint {
		case '<':
			return writer.writeAll([]uint8("&lt;"))
		case '>':
			return writer.writeAll([]uint8("&gt;"))
		case '&':
			return writer.writeAll([]uint8("&amp;"))
		case '"':
			return writer.writeAll([]uint8("&quot;"))
		case '\'':
			return writer.writeAll([]uint8("&#39;"))
		default:
			// For HTML, emit ASCII (< 0x80) directly, but encode
			// all non-ASCII as numeric entities to avoid encoding
			// detection issues (fixes #9426). We can't set the
			// meta tag because we emit partial HTML so this ensures
			// proper unicode handling.
			if codepoint < 0x80 {
				// writer.print("{u}", .{codepoint})
				return writer.printUnicodeCodepoint(codepoint)
			} else {
				// writer.print("&#{d};", .{codepoint})
				return writer.writeAll([]uint8("&#" + itoa(uint64(codepoint)) + ";"))
			}
		}
	}
	panic("unreachable")
}

// cellStyle is formatter.PageFormatter.cellStyle.
//
// Returns the style for the given cell. If there is no styling this
// will return the default style.
//
// Zig: fn cellStyle(self: *const PageFormatter, cell: *const Cell) Style.
func (self *FormatterPageFormatter) cellStyle(cell *PageCell) Style {
	switch cell.content_tag {
	case PageCellContentTagCodepoint, PageCellContentTagCodepointGrapheme:
		if !cell.HasStyling() {
			return Style{}
		}
		return *self.page.styles.Get(
			cell.style_id,
		)

	case PageCellContentTagBgColorPalette:
		return Style{
			BgColor: StyleColor{
				Tag:     StyleColorPalette,
				Palette: cell.content.color_palette,
			},
		}

	case PageCellContentTagBgColorRgb:
		return Style{
			BgColor: StyleColor{
				Tag: StyleColorRgb,
				Rgb: ColorRGB{
					R: cell.content.color_rgb.R,
					G: cell.content.color_rgb.G,
					B: cell.content.color_rgb.B,
				},
			},
		}
	}
	panic("unreachable")
}

// formatStyleOpen is formatter.PageFormatter.formatStyleOpen.
//
// Write a string with HTML escaping. Used for escaping href attributes
// and other HTML attribute values.
//
// Zig: fn formatStyleOpen(self: PageFormatter, writer: *std.Io.Writer,
// style: *const Style) std.Io.Writer.Error!void. The error is
// ErrWriteFailed (osc.go). Zig traps (`unreachable`) for `.plain`.
func (self FormatterPageFormatter) formatStyleOpen(writer *oscWriter, style *Style) error {
	switch self.opts.Emit {
	case FormatterFormatPlain:
		panic("unreachable")

	case FormatterFormatVt:
		formatter := style.FormatterVt()
		formatter.palette = self.opts.Palette
		// writer.print("{f}", .{formatter})
		return writer.writeAll([]uint8(formatter.Format()))

	// We use `display: inline` so that the div doesn't impact
	// layout since we're primarily using it as a CSS wrapper.
	case FormatterFormatHtml:
		formatter := style.FormatterHtml()
		formatter.palette = self.opts.Palette
		// writer.print(
		//     "<div style=\"display: inline;{f}\">",
		//     .{formatter},
		// )
		return writer.writeAll([]uint8("<div style=\"display: inline;" + formatter.Format() + "\">"))
	}
	panic("unreachable")
}

// formatStyleClose is formatter.PageFormatter.formatStyleClose.
//
// Zig: fn formatStyleClose(self: PageFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go), also
// returned when appending to the point map fails.
func (self FormatterPageFormatter) formatStyleClose(writer *oscWriter) error {
	var str string
	switch self.opts.Emit {
	case FormatterFormatPlain:
		return nil
	case FormatterFormatVt:
		str = "\x1b[0m"
	case FormatterFormatHtml:
		str = "</div>"
	default:
		panic("unreachable")
	}

	if err := writer.writeAll([]uint8(str)); err != nil {
		return err
	}
	if self.point_map.Valid {
		m := &self.point_map.Value
		assert(len(*m.map_) > 0)
		// ensureUnusedCapacity + appendNTimesAssumeCapacity
		formatterAppendNTimes(
			m.map_,
			(*m.map_)[len(*m.map_)-1],
			uint64(len(str)),
		)
	}
	return nil
}

// formatHyperlinkOpen is formatter.PageFormatter.formatHyperlinkOpen.
//
// Zig: fn formatHyperlinkOpen(self: PageFormatter, writer: *std.Io.Writer,
// uri: []const u8) std.Io.Writer.Error!void. The error is ErrWriteFailed
// (osc.go). Zig traps (`unreachable`) for `.plain` and `.vt`.
func (self FormatterPageFormatter) formatHyperlinkOpen(writer *oscWriter, uri []uint8) error {
	switch self.opts.Emit {
	case FormatterFormatPlain, FormatterFormatVt:
		panic("unreachable")

	// layout since we're primarily using it as a CSS wrapper.
	case FormatterFormatHtml:
		if err := writer.writeAll([]uint8("<a href=\"")); err != nil {
			return err
		}
		for _, byte_ := range uri {
			if err := self.writeCodepoint(
				writer,
				uint32(byte_),
			); err != nil {
				return err
			}
		}
		return writer.writeAll([]uint8("\">"))
	}
	panic("unreachable")
}

// formatHyperlinkClose is formatter.PageFormatter.formatHyperlinkClose.
//
// Zig: fn formatHyperlinkClose(self: PageFormatter, writer: *std.Io.Writer)
// std.Io.Writer.Error!void. The error is ErrWriteFailed (osc.go), also
// returned when appending to the point map fails.
func (self FormatterPageFormatter) formatHyperlinkClose(writer *oscWriter) error {
	var str string
	switch self.opts.Emit {
	case FormatterFormatHtml:
		str = "</a>"
	case FormatterFormatPlain, FormatterFormatVt:
		return nil
	default:
		panic("unreachable")
	}

	if err := writer.writeAll([]uint8(str)); err != nil {
		return err
	}
	if self.point_map.Valid {
		m := &self.point_map.Value
		assert(len(*m.map_) > 0)
		// ensureUnusedCapacity + appendNTimesAssumeCapacity
		formatterAppendNTimes(
			m.map_,
			(*m.map_)[len(*m.map_)-1],
			uint64(len(str)),
		)
	}
	return nil
}

// formatterCharsetsSlotsValues is `std.enums.values(charsets.Slots)` in
// ScreenFormatter.format: every slot in declaration order (a compile-time
// constant in Zig). Not a Zig declaration.
var formatterCharsetsSlotsValues = [...]CharsetsSlots{
	CharsetsSlotsG0,
	CharsetsSlotsG1,
	CharsetsSlotsG2,
	CharsetsSlotsG3,
}

// formatterHex2 is what Zig's `{x:0>2}` prints for a u8: two lowercase hex
// digits, zero-padded. Not a Zig declaration.
func formatterHex2(v uint8) string {
	const digits = "0123456789abcdef"
	return string([]uint8{digits[v>>4], digits[v&0xF]})
}

// formatterAddOneU16 is Zig's `v + 1` on a u16 (size.CellCountInt), which
// traps on overflow. Not a Zig declaration.
func formatterAddOneU16(v uint16) uint16 {
	assert(v < 0xFFFF) // Zig: integer overflow traps
	return v + 1
}

// formatterAppendNTimes is std.ArrayList(T).appendNTimes (and
// ensureUnusedCapacity followed by appendNTimesAssumeCapacity) on the
// caller's list: append value n times. The allocation failure Zig maps to
// error.WriteFailed cannot happen in Go. Not a Zig declaration of this
// file.
func formatterAppendNTimes[T any](list *[]T, value T, n uint64) {
	for i := uint64(0); i < n; i++ {
		*list = append(*list, value)
	}
}
