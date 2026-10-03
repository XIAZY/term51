// Tests translated one for one from src/terminal/style.zig (Ghostty v1.3.1
// plus the term51 additional tests). Each Go test carries the exact Zig title
// in a "// Zig: test" comment. The reference-counted set tests of style.zig are
// here too, instantiated as StyleSet.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order (Parser_test.go).
//   - Zig's `std.Io.Writer.Allocating` that collects `{f}` output is not
//     needed where Go's `Format() string` returns the text directly (Style
//     formatters, Color.format, Style.format). The formatColor tests, which
//     take a writer, use oscWriterAllocating (osc.go).
//   - `const buf = try alloc.alignedAlloc(...)` and its `defer alloc.free(buf)`
//     have no counterpart: the Go set owns its memory and every set method
//     drops its `buf` parameter.
//   - `@bitCast` of a PackedStyle to a u128 is stylePackedStyle.bitCast(),
//     which returns the two 64-bit halves (low half first); the expected
//     128-bit values are built with sm3_u128.
//   - Style.bg and Style.bgCell wait for the page wireframe (Cell type): see
//     scripts/not-translated-sm3.txt.

package terminal

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// sm3_pal is the Zig literal `.{ .palette = idx }` of a Style.Color.
func sm3_pal(idx uint8) StyleColor {
	return StyleColor{Tag: StyleColorPalette, Palette: idx}
}

// sm3_rgb is the Zig literal `.{ .rgb = .{ .r = r, .g = g, .b = b } }` of a
// Style.Color.
func sm3_rgb(r uint8, g uint8, b uint8) StyleColor {
	return StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{r: r, g: g, b: b}}
}

// sm3_bold is the Zig literal `.{ .color = .{ .r = r, .g = g, .b = b } }` of a
// config.BoldColor, as an optional.
func sm3_bold(r uint8, g uint8, b uint8) Optional[ConfigBoldColor] {
	return Some(ConfigBoldColor{Tag: ConfigBoldColorColor, Color: ConfigColor{r: r, g: g, b: b}})
}

// sm3_bright is the Zig literal `.bright` of a config.BoldColor, as an
// optional.
func sm3_bright() Optional[ConfigBoldColor] {
	return Some(ConfigBoldColor{Tag: ConfigBoldColorBright})
}

// sm3_expectError is Zig's `try testing.expectError(want, got)`.
func sm3_expectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// sm3_buffered is Zig's `writer.buffered()`: buffer[:end].
func sm3_buffered(w *oscWriter) string {
	return string(w.buffer[:w.end])
}

// sm3_hashInt is std.hash.int for a u64 (Zig 0.15.2, lib/std/hash.zig, the
// `uint64` function, source https://github.com/jonmaiga/mx3).
func sm3_hashInt(input uint64) uint64 {
	x := input
	const c = 0xbea225f9eb34556d
	x = (x ^ (x >> 32)) * c
	x = (x ^ (x >> 29)) * c
	x = (x ^ (x >> 32)) * c
	x = x ^ (x >> 29)
	return x
}

// sm3_u128 is `@as(u128, v) << shift` as two 64-bit halves, low half first.
func sm3_u128(v uint64, shift uint) [2]uint64 {
	if shift >= 128 {
		panic("shift out of range")
	}
	if shift == 0 {
		return [2]uint64{v, 0}
	}
	if shift >= 64 {
		return [2]uint64{0, v << (shift - 64)}
	}
	return [2]uint64{v << shift, v >> (64 - shift)}
}

// sm3_or128 is `a | b` on u128 values held as two 64-bit halves.
func sm3_or128(a [2]uint64, b [2]uint64) [2]uint64 {
	return [2]uint64{a[0] | b[0], a[1] | b[1]}
}

// sm3_newSet is `Set.init(.init(buf), layout, .{})`.
func sm3_newSet(layout RefCountedSetLayout[StyleId]) StyleSet {
	return RefCountedSetInit[Style, StyleId, SizeCellCountInt, StyleSetContext](layout, StyleSetContext{})
}

// sm3_add is `try set.add(buf, value)`.
func sm3_add(t *testing.T, set *StyleSet, value Style) StyleId {
	t.Helper()
	id, err := set.Add(value)
	zigTry(t, err)
	return id
}

// sm3_addWithId is `try set.addWithId(buf, value, id)`.
func sm3_addWithId(t *testing.T, set *StyleSet, value Style, id StyleId) Optional[StyleId] {
	t.Helper()
	r, err := set.AddWithId(value, id)
	zigTry(t, err)
	return r
}

// sm3_palette is term51FPalette: a palette where every entry is distinct and
// derived from its index.
func sm3_palette() ColorPalette {
	var p ColorPalette
	for i := range p {
		p[i] = ColorRGB{
			r: uint8(i),
			g: uint8(255 - i),
			b: 100,
		}
	}
	return p
}

// sm3_rgbStyle is term51FRgbStyle: a style with an rgb foreground that is
// unique for every u16.
func sm3_rgbStyle(i uint16) Style {
	return Style{fg_color: sm3_rgb(uint8(i), uint8(i>>8), 0)}
}

// sm3_variants is term51FVariants: every style that differs from the default
// style in exactly one field value. The default style itself is NOT
// included.
func sm3_variants() []Style {
	var list []Style

	colors := []StyleColor{
		sm3_pal(0),
		sm3_pal(1),
		sm3_pal(255),
		sm3_rgb(0, 0, 0),
		sm3_rgb(1, 0, 0),
		sm3_rgb(0, 1, 0),
		sm3_rgb(0, 0, 1),
	}
	for _, c := range colors {
		list = append(list, Style{fg_color: c})
		list = append(list, Style{bg_color: c})
		list = append(list, Style{underline_color: c})
	}

	list = append(list, Style{flags: styleFlags{bold: true}})
	list = append(list, Style{flags: styleFlags{italic: true}})
	list = append(list, Style{flags: styleFlags{faint: true}})
	list = append(list, Style{flags: styleFlags{blink: true}})
	list = append(list, Style{flags: styleFlags{inverse: true}})
	list = append(list, Style{flags: styleFlags{invisible: true}})
	list = append(list, Style{flags: styleFlags{strikethrough: true}})
	list = append(list, Style{flags: styleFlags{overline: true}})
	list = append(list, Style{flags: styleFlags{underline: SGRAttributeUnderlineSingle}})
	list = append(list, Style{flags: styleFlags{underline: SGRAttributeUnderlineDouble}})
	list = append(list, Style{flags: styleFlags{underline: SGRAttributeUnderlineCurly}})
	list = append(list, Style{flags: styleFlags{underline: SGRAttributeUnderlineDotted}})
	list = append(list, Style{flags: styleFlags{underline: SGRAttributeUnderlineDashed}})
	list = append(list, Style{flags: styleFlags{_padding: 1}})
	return list
}

// Zig: test "Style VT formatting empty"
func TestStyle_Style_VT_formatting_empty(t *testing.T) {
	style := Style{}
	expectEqualStrings(t, "\x1b[0m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting bold"
func TestStyle_Style_VT_formatting_bold(t *testing.T) {
	style := Style{flags: styleFlags{bold: true}}
	expectEqualStrings(t, "\x1b[0m\x1b[1m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting faint"
func TestStyle_Style_VT_formatting_faint(t *testing.T) {
	style := Style{flags: styleFlags{faint: true}}
	expectEqualStrings(t, "\x1b[0m\x1b[2m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting italic"
func TestStyle_Style_VT_formatting_italic(t *testing.T) {
	style := Style{flags: styleFlags{italic: true}}
	expectEqualStrings(t, "\x1b[0m\x1b[3m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting blink"
func TestStyle_Style_VT_formatting_blink(t *testing.T) {
	style := Style{flags: styleFlags{blink: true}}
	expectEqualStrings(t, "\x1b[0m\x1b[5m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting inverse"
func TestStyle_Style_VT_formatting_inverse(t *testing.T) {
	style := Style{flags: styleFlags{inverse: true}}
	expectEqualStrings(t, "\x1b[0m\x1b[7m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting invisible"
func TestStyle_Style_VT_formatting_invisible(t *testing.T) {
	style := Style{flags: styleFlags{invisible: true}}
	expectEqualStrings(t, "\x1b[0m\x1b[8m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting strikethrough"
func TestStyle_Style_VT_formatting_strikethrough(t *testing.T) {
	style := Style{flags: styleFlags{strikethrough: true}}
	expectEqualStrings(t, "\x1b[0m\x1b[9m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting overline"
func TestStyle_Style_VT_formatting_overline(t *testing.T) {
	style := Style{flags: styleFlags{overline: true}}
	expectEqualStrings(t, "\x1b[0m\x1b[53m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting underline single"
func TestStyle_Style_VT_formatting_underline_single(t *testing.T) {
	style := Style{flags: styleFlags{underline: SGRAttributeUnderlineSingle}}
	expectEqualStrings(t, "\x1b[0m\x1b[4m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting underline double"
func TestStyle_Style_VT_formatting_underline_double(t *testing.T) {
	style := Style{flags: styleFlags{underline: SGRAttributeUnderlineDouble}}
	expectEqualStrings(t, "\x1b[0m\x1b[4:2m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting underline curly"
func TestStyle_Style_VT_formatting_underline_curly(t *testing.T) {
	style := Style{flags: styleFlags{underline: SGRAttributeUnderlineCurly}}
	expectEqualStrings(t, "\x1b[0m\x1b[4:3m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting underline dotted"
func TestStyle_Style_VT_formatting_underline_dotted(t *testing.T) {
	style := Style{flags: styleFlags{underline: SGRAttributeUnderlineDotted}}
	expectEqualStrings(t, "\x1b[0m\x1b[4:4m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting underline dashed"
func TestStyle_Style_VT_formatting_underline_dashed(t *testing.T) {
	style := Style{flags: styleFlags{underline: SGRAttributeUnderlineDashed}}
	expectEqualStrings(t, "\x1b[0m\x1b[4:5m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting fg palette"
func TestStyle_Style_VT_formatting_fg_palette(t *testing.T) {
	style := Style{fg_color: sm3_pal(42)}
	expectEqualStrings(t, "\x1b[0m\x1b[38;5;42m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting fg rgb"
func TestStyle_Style_VT_formatting_fg_rgb(t *testing.T) {
	style := Style{fg_color: sm3_rgb(255, 128, 64)}
	expectEqualStrings(t, "\x1b[0m\x1b[38;2;255;128;64m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting bg palette"
func TestStyle_Style_VT_formatting_bg_palette(t *testing.T) {
	style := Style{bg_color: sm3_pal(7)}
	expectEqualStrings(t, "\x1b[0m\x1b[48;5;7m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting bg rgb"
func TestStyle_Style_VT_formatting_bg_rgb(t *testing.T) {
	style := Style{bg_color: sm3_rgb(32, 64, 96)}
	expectEqualStrings(t, "\x1b[0m\x1b[48;2;32;64;96m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting underline_color palette"
func TestStyle_Style_VT_formatting_underline_color_palette(t *testing.T) {
	style := Style{underline_color: sm3_pal(15)}
	expectEqualStrings(t, "\x1b[0m\x1b[58;5;15m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting underline_color rgb"
func TestStyle_Style_VT_formatting_underline_color_rgb(t *testing.T) {
	style := Style{underline_color: sm3_rgb(200, 100, 50)}
	expectEqualStrings(t, "\x1b[0m\x1b[58;2;200;100;50m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting multiple flags"
func TestStyle_Style_VT_formatting_multiple_flags(t *testing.T) {
	style := Style{flags: styleFlags{bold: true, italic: true, underline: SGRAttributeUnderlineSingle}}
	expectEqualStrings(t, "\x1b[0m\x1b[1m\x1b[3m\x1b[4m", style.FormatterVt().Format())
}

// Zig: test "Style VT formatting all flags"
func TestStyle_Style_VT_formatting_all_flags(t *testing.T) {
	style := Style{flags: styleFlags{
		bold:          true,
		faint:         true,
		italic:        true,
		blink:         true,
		inverse:       true,
		invisible:     true,
		strikethrough: true,
		overline:      true,
		underline:     SGRAttributeUnderlineCurly,
	}}
	expectEqualStrings(t,
		"\x1b[0m\x1b[1m\x1b[2m\x1b[3m\x1b[5m\x1b[7m\x1b[8m\x1b[9m\x1b[53m\x1b[4:3m",
		style.FormatterVt().Format(),
	)
}

// Zig: test "Style VT formatting combined colors and flags"
func TestStyle_Style_VT_formatting_combined_colors_and_flags(t *testing.T) {
	style := Style{
		fg_color:        sm3_rgb(255, 0, 0),
		bg_color:        sm3_pal(8),
		underline_color: sm3_rgb(0, 255, 0),
		flags:           styleFlags{bold: true, italic: true, underline: SGRAttributeUnderlineDouble},
	}
	expectEqualStrings(t,
		"\x1b[0m\x1b[1m\x1b[3m\x1b[4:2m\x1b[38;2;255;0;0m\x1b[48;5;8m\x1b[58;2;0;255;0m",
		style.FormatterVt().Format(),
	)
}

// Zig: test "Style VT formatting all colors rgb"
func TestStyle_Style_VT_formatting_all_colors_rgb(t *testing.T) {
	style := Style{
		fg_color:        sm3_rgb(10, 20, 30),
		bg_color:        sm3_rgb(40, 50, 60),
		underline_color: sm3_rgb(70, 80, 90),
	}
	expectEqualStrings(t,
		"\x1b[0m\x1b[38;2;10;20;30m\x1b[48;2;40;50;60m\x1b[58;2;70;80;90m",
		style.FormatterVt().Format(),
	)
}

// Zig: test "Style VT formatting all colors palette"
func TestStyle_Style_VT_formatting_all_colors_palette(t *testing.T) {
	style := Style{
		fg_color:        sm3_pal(1),
		bg_color:        sm3_pal(2),
		underline_color: sm3_pal(3),
	}
	expectEqualStrings(t,
		"\x1b[0m\x1b[38;5;1m\x1b[48;5;2m\x1b[58;5;3m",
		style.FormatterVt().Format(),
	)
}

// Zig: test "Style VT formatting palette with palette set emits rgb"
func TestStyle_Style_VT_formatting_palette_with_palette_set_emits_rgb(t *testing.T) {
	style := Style{fg_color: sm3_pal(1)}
	formatter := style.FormatterVt()
	formatter.palette = &ColorDefault
	expectEqualStrings(t, "\x1b[0m\x1b[38;2;204;102;102m", formatter.Format())
}

// Zig: test "Style VT formatting all palette colors with palette set"
func TestStyle_Style_VT_formatting_all_palette_colors_with_palette_set(t *testing.T) {
	style := Style{
		fg_color:        sm3_pal(1),
		bg_color:        sm3_pal(2),
		underline_color: sm3_pal(3),
	}
	formatter := style.FormatterVt()
	formatter.palette = &ColorDefault
	expectEqualStrings(t,
		"\x1b[0m\x1b[38;2;204;102;102m\x1b[48;2;181;189;104m\x1b[58;2;240;198;116m",
		formatter.Format(),
	)
}

// Zig: test "Set basic usage"
func TestStyle_Set_basic_usage(t *testing.T) {
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);

	style := Style{flags: styleFlags{bold: true}}
	style2 := Style{flags: styleFlags{italic: true}}

	set := sm3_newSet(layout)

	// Add style
	id := sm3_add(t, &set, style)
	expect(t, id > 0, "id > 0")

	// Second add should return the same metadata.
	{
		id2 := sm3_add(t, &set, style)
		expectEqual(t, id, id2)
	}

	// Look it up
	{
		v := set.Get(id)
		expect(t, v.flags.bold, "v.flags.bold")

		v2 := set.Get(id)
		expectEqual(t, v, v2)
	}

	// Add a second style
	id2 := sm3_add(t, &set, style2)

	// Look it up
	{
		v := set.Get(id2)
		expect(t, v.flags.italic, "v.flags.italic")
	}

	// Ref count
	expect(t, set.RefCount(id) == 2, "set.refCount(buf, id) == 2")
	expect(t, set.RefCount(id2) == 1, "set.refCount(buf, id2) == 1")

	// Release
	set.Release(id)
	expect(t, set.RefCount(id) == 1, "set.refCount(buf, id) == 1")
	set.Release(id2)
	expect(t, set.RefCount(id2) == 0, "set.refCount(buf, id2) == 0")

	// We added the first one twice, so
	set.Release(id)
	expect(t, set.RefCount(id) == 0, "set.refCount(buf, id) == 0")
}

// Zig: test "Set capacities"
func TestStyle_Set_capacities(t *testing.T) {
	// We want to support at least this many styles without overflowing.
	_ = RefCountedSetLayoutInit[StyleId](16384)
}

// Zig: test "Style HTML formatting basic bold"
func TestStyle_Style_HTML_formatting_basic_bold(t *testing.T) {
	style := Style{flags: styleFlags{bold: true}}
	expectEqualStrings(t, "font-weight: bold;", style.FormatterHtml().Format())
}

// Zig: test "Style HTML formatting fg color rgb"
func TestStyle_Style_HTML_formatting_fg_color_rgb(t *testing.T) {
	style := Style{fg_color: sm3_rgb(255, 128, 64)}
	expectEqualStrings(t, "color: rgb(255, 128, 64);", style.FormatterHtml().Format())
}

// Zig: test "Style HTML formatting bg color palette"
func TestStyle_Style_HTML_formatting_bg_color_palette(t *testing.T) {
	style := Style{bg_color: sm3_pal(7)}
	expectEqualStrings(t, "background-color: var(--vt-palette-7);", style.FormatterHtml().Format())
}

// Zig: test "Style HTML formatting combined colors and flags"
func TestStyle_Style_HTML_formatting_combined_colors_and_flags(t *testing.T) {
	style := Style{
		fg_color: sm3_rgb(255, 0, 0),
		bg_color: sm3_rgb(0, 0, 255),
		flags:    styleFlags{bold: true, italic: true},
	}
	result := style.FormatterHtml().Format()
	expect(t, strings.Contains(result, "color: rgb(255, 0, 0);"), `std.mem.indexOf(u8, result, "color: rgb(255, 0, 0);") != null`)
	expect(t, strings.Contains(result, "background-color: rgb(0, 0, 255);"), `std.mem.indexOf(u8, result, "background-color: rgb(0, 0, 255);") != null`)
	expect(t, strings.Contains(result, "font-weight: bold;"), `std.mem.indexOf(u8, result, "font-weight: bold;") != null`)
	expect(t, strings.Contains(result, "font-style: italic;"), `std.mem.indexOf(u8, result, "font-style: italic;") != null`)
}

// Zig: test "Style HTML formatting single decoration line"
func TestStyle_Style_HTML_formatting_single_decoration_line(t *testing.T) {
	style := Style{flags: styleFlags{underline: SGRAttributeUnderlineSingle}}
	result := style.FormatterHtml().Format()
	expect(t, strings.Contains(result, "text-decoration-line: underline;"), `std.mem.indexOf(u8, result, "text-decoration-line: underline;") != null`)
	expect(t, strings.Contains(result, "text-decoration-style: solid;"), `std.mem.indexOf(u8, result, "text-decoration-style: solid;") != null`)
}

// Zig: test "Style HTML formatting multiple decoration lines"
func TestStyle_Style_HTML_formatting_multiple_decoration_lines(t *testing.T) {
	style := Style{flags: styleFlags{underline: SGRAttributeUnderlineCurly, strikethrough: true, overline: true}}
	result := style.FormatterHtml().Format()
	expect(t, strings.Contains(result, "text-decoration-line: underline line-through overline;"), `std.mem.indexOf(u8, result, "text-decoration-line: underline line-through overline;") != null`)
	expect(t, strings.Contains(result, "text-decoration-style: wavy;"), `std.mem.indexOf(u8, result, "text-decoration-style: wavy;") != null`)
}

// Zig: test "Style HTML formatting palette with palette set emits rgb"
func TestStyle_Style_HTML_formatting_palette_with_palette_set_emits_rgb(t *testing.T) {
	style := Style{bg_color: sm3_pal(7)}
	formatter := style.FormatterHtml()
	formatter.palette = &ColorDefault
	expectEqualStrings(t, "background-color: rgb(197, 200, 198);", formatter.Format())
}

// Zig: test "Style HTML formatting all palette colors with palette set"
func TestStyle_Style_HTML_formatting_all_palette_colors_with_palette_set(t *testing.T) {
	style := Style{
		fg_color:        sm3_pal(1),
		bg_color:        sm3_pal(2),
		underline_color: sm3_pal(3),
	}
	formatter := style.FormatterHtml()
	formatter.palette = &ColorDefault
	expectEqualStrings(t,
		"color: rgb(204, 102, 102);background-color: rgb(181, 189, 104);text-decoration-color: rgb(240, 198, 116);",
		formatter.Format(),
	)
}

// ---------------------------------------------------------------------------
// term51: additional tests (F). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: Style.Color.eql: same tag and value"
func TestStyle_term51_Style_Color_eql_same_tag_and_value(t *testing.T) {
	none := StyleColor{}
	expect(t, none.Eql(StyleColor{}), "none.eql(.none)")

	p0 := sm3_pal(0)
	expect(t, p0.Eql(sm3_pal(0)), "p0.eql(.{ .palette = 0 })")
	p255 := sm3_pal(255)
	expect(t, p255.Eql(sm3_pal(255)), "p255.eql(.{ .palette = 255 })")

	rgb := sm3_rgb(1, 2, 3)
	expect(t, rgb.Eql(sm3_rgb(1, 2, 3)), "rgb.eql(.{ .rgb = .{ .r = 1, .g = 2, .b = 3 } })")
	black := sm3_rgb(0, 0, 0)
	// Zig: `.{ .rgb = .{} }`, the default color.RGB (all channels 0).
	expect(t, black.Eql(StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{}}), "black.eql(.{ .rgb = .{} })")
	white := sm3_rgb(255, 255, 255)
	expect(t, white.Eql(sm3_rgb(255, 255, 255)), "white.eql(.{ .rgb = .{ .r = 255, .g = 255, .b = 255 } })")
}

// Zig: test "term51: Style.Color.eql: different value same tag"
func TestStyle_term51_Style_Color_eql_different_value_same_tag(t *testing.T) {
	p0 := sm3_pal(0)
	expect(t, !p0.Eql(sm3_pal(1)), "!p0.eql(.{ .palette = 1 })")
	p255 := sm3_pal(255)
	expect(t, !p255.Eql(sm3_pal(254)), "!p255.eql(.{ .palette = 254 })")

	// Each rgb channel is compared.
	rgb := sm3_rgb(1, 2, 3)
	expect(t, !rgb.Eql(sm3_rgb(9, 2, 3)), "!rgb.eql(.{ .rgb = .{ .r = 9, .g = 2, .b = 3 } })")
	expect(t, !rgb.Eql(sm3_rgb(1, 9, 3)), "!rgb.eql(.{ .rgb = .{ .r = 1, .g = 9, .b = 3 } })")
	expect(t, !rgb.Eql(sm3_rgb(1, 2, 9)), "!rgb.eql(.{ .rgb = .{ .r = 1, .g = 2, .b = 9 } })")
}

// Zig: test "term51: Style.Color.eql: different tags never equal"
func TestStyle_term51_Style_Color_eql_different_tags_never_equal(t *testing.T) {
	all := []StyleColor{
		{},
		sm3_pal(0),
		sm3_rgb(0, 0, 0),
	}
	for i, a := range all {
		for j, b := range all {
			expectEqual(t, i == j, a.Eql(b))
		}
	}

	// palette 5 and rgb (5,0,0) share a numeric payload but not a tag.
	p := sm3_pal(5)
	expect(t, !p.Eql(sm3_rgb(5, 0, 0)), "!p.eql(.{ .rgb = .{ .r = 5, .g = 0, .b = 0 } })")
}

// Zig: test "term51: Style.Color.format: each tag"
func TestStyle_term51_Style_Color_format_each_tag(t *testing.T) {
	// Zig-only: var builder: std.Io.Writer.Allocating = .init(alloc); defer builder.deinit(); builder.writer.end = 0 (Format returns the string)
	none := StyleColor{}
	expectEqualStrings(t, "Color.none", none.Format())

	p := sm3_pal(255)
	expectEqualStrings(t, "Color.palette{ 255 }", p.Format())

	p0 := sm3_pal(0)
	expectEqualStrings(t, "Color.palette{ 0 }", p0.Format())

	rgb := sm3_rgb(1, 22, 255)
	expectEqualStrings(t, "Color.rgb{ 1, 22, 255 }", rgb.Format())
}

// Zig: test "term51: Style.default: default style and one-field variants"
func TestStyle_term51_Style_default_default_style_and_one_field_variants(t *testing.T) {
	d := Style{}
	expect(t, d.Default(), "d.default()")

	variants := sm3_variants()
	// Zig-only: defer variants.deinit(alloc)
	expectEqual(t, uint(35), uint(len(variants)))
	for _, v := range variants {
		expect(t, !v.Default(), "!v.default()")
	}
}

// Zig: test "term51: Style.default: default id and id type"
func TestStyle_term51_Style_default_default_id_and_id_type(t *testing.T) {
	expectEqual(t, StyleId(0), StyleDefaultId)
	expect(t, reflect.TypeOf(StyleId(0)) == reflect.TypeOf(uint16(0)), "Id == u16")
	expect(t, reflect.TypeOf(StyleId(0)) == reflect.TypeOf(SizeStyleCountInt(0)), "Id == size.StyleCountInt")
}

// Zig: test "term51: Style.eql: equal styles"
func TestStyle_term51_Style_eql_equal_styles(t *testing.T) {
	a := Style{}
	expect(t, a.Eql(Style{}), "a.eql(.{})")

	full := Style{
		fg_color:        sm3_rgb(1, 2, 3),
		bg_color:        sm3_pal(4),
		underline_color: sm3_rgb(5, 6, 7),
		flags: styleFlags{
			bold:          true,
			italic:        true,
			faint:         true,
			blink:         true,
			inverse:       true,
			invisible:     true,
			strikethrough: true,
			overline:      true,
			underline:     SGRAttributeUnderlineDashed,
		},
	}
	copy := full
	expect(t, full.Eql(copy), "full.eql(copy)")
	expect(t, copy.Eql(full), "copy.eql(full)")
	copy.flags.underline = SGRAttributeUnderlineDotted
	expect(t, !full.Eql(copy), "!full.eql(copy)")
}

// Zig: test "term51: Style.eql: every one-field difference is unequal"
func TestStyle_term51_Style_eql_every_one_field_difference_is_unequal(t *testing.T) {
	variants := sm3_variants()
	// Zig-only: defer variants.deinit(alloc)

	d := Style{}
	for i, a := range variants {
		expect(t, !a.Eql(d), "!a.eql(d)")
		expect(t, !d.Eql(a), "!d.eql(a)")
		expect(t, a.Eql(a), "a.eql(a)")
		for j, b := range variants {
			expectEqual(t, i == j, a.Eql(b))
		}
	}
}

// Zig: test "term51: Style.eql: same colors in different slots are unequal"
func TestStyle_term51_Style_eql_same_colors_in_different_slots_are_unequal(t *testing.T) {
	c := sm3_pal(7)
	fg := Style{fg_color: c}
	bg := Style{bg_color: c}
	ul := Style{underline_color: c}
	expect(t, !fg.Eql(bg), "!fg.eql(bg)")
	expect(t, !fg.Eql(ul), "!fg.eql(ul)")
	expect(t, !bg.Eql(ul), "!bg.eql(ul)")
}

// Zig: test "term51: Style.eql: flag padding participates"
func TestStyle_term51_Style_eql_flag_padding_participates(t *testing.T) {
	a := Style{}
	b := Style{flags: styleFlags{_padding: 0b10000}}
	expect(t, !a.Eql(b), "!a.eql(b)")
	expect(t, !a.Default() || a.Default(), "!a.default() or a.default()")
	expect(t, !b.Default(), "!b.default()")
}

// Not translated (pending: waits for the page wireframe, Cell type): see scripts/not-translated-sm3.txt.
//   Zig: test "term51: Style.bg: cell content tag decides the source"
//   Zig: test "term51: Style.bg: palette boundaries"
//   Zig: test "term51: Style.bg: flags and other colors do not matter"

// Zig: test "term51: Style.underlineColor: each tag"
func TestStyle_term51_Style_underlineColor_each_tag(t *testing.T) {
	pal := sm3_palette()

	none := Style{}
	expectEqual(t, None[ColorRGB](), none.UnderlineColor(&pal))

	for _, idx := range []uint8{0, 1, 255} {
		s := Style{underline_color: sm3_pal(idx)}
		expectEqual(t, Some(pal[idx]), s.UnderlineColor(&pal))
	}

	rgb := Style{underline_color: sm3_rgb(4, 5, 6)}
	expectEqual(t,
		Some(ColorRGB{r: 4, g: 5, b: 6}),
		rgb.UnderlineColor(&pal),
	)

	// Other fields do not matter.
	other := Style{
		fg_color: sm3_pal(1),
		bg_color: sm3_pal(2),
		flags:    styleFlags{underline: SGRAttributeUnderlineCurly},
	}
	expectEqual(t, None[ColorRGB](), other.UnderlineColor(&pal))
}

// Not translated (pending: waits for the page wireframe, Cell type): see scripts/not-translated-sm3.txt.
//   Zig: test "term51: Style.bgCell: no bg"
//   Zig: test "term51: Style.bgCell: palette bg"
//   Zig: test "term51: Style.bgCell: rgb bg"

// Zig: test "term51: Style.format: default and flags"
func TestStyle_term51_Style_format_default_and_flags(t *testing.T) {
	// Zig-only: var builder: std.Io.Writer.Allocating = .init(alloc); defer builder.deinit(); builder.writer.end = 0 (Format returns the string)
	d := Style{}
	expectEqualStrings(t, "Style{ flags={  } }", d.Format())

	bold := Style{flags: styleFlags{bold: true}}
	expectEqualStrings(t, "Style{ flags={ bold } }", bold.Format())

	all := Style{flags: styleFlags{
		bold:          true,
		italic:        true,
		faint:         true,
		blink:         true,
		inverse:       true,
		invisible:     true,
		strikethrough: true,
		overline:      true,
		underline:     SGRAttributeUnderlineDouble,
	}}
	expectEqualStrings(t,
		"Style{ flags={ bold, italic, faint, blink, inverse, invisible, strikethrough, overline, underline=.double } }",
		all.Format(),
	)

	ul := Style{flags: styleFlags{underline: SGRAttributeUnderlineDashed}}
	expectEqualStrings(t, "Style{ flags={ underline=.dashed } }", ul.Format())
}

// Zig: test "term51: Style.format: colors"
func TestStyle_term51_Style_format_colors(t *testing.T) {
	// Zig-only: var builder: std.Io.Writer.Allocating = .init(alloc); defer builder.deinit(); builder.writer.end = 0 (Format returns the string)
	fg := Style{fg_color: sm3_pal(5)}
	expectEqualStrings(t,
		"Style{ fg_color=.{ .palette = 5 }, flags={  } }",
		fg.Format(),
	)

	all := Style{
		fg_color:        sm3_rgb(1, 2, 3),
		bg_color:        sm3_pal(0),
		underline_color: sm3_rgb(0, 0, 0),
		flags:           styleFlags{italic: true},
	}
	expectEqualStrings(t,
		"Style{ fg_color=.{ .rgb = .{ .r = 1, .g = 2, .b = 3 } }, "+
			"bg_color=.{ .palette = 0 }, "+
			"underline_color=.{ .rgb = .{ .r = 0, .g = 0, .b = 0 } }, "+
			"flags={ italic } }",
		all.Format(),
	)
}

// Zig: test "term51: Style.formatterVt: references the style and has no palette"
func TestStyle_term51_Style_formatterVt_references_the_style_and_has_no_palette(t *testing.T) {
	style := Style{flags: styleFlags{bold: true}}
	f := style.FormatterVt()
	expect(t, f.style == &style, "f.style == &style")
	expect(t, f.palette == nil, "f.palette == null")
}

// Zig: test "term51: Style.formatterHtml: references the style and has no palette"
func TestStyle_term51_Style_formatterHtml_references_the_style_and_has_no_palette(t *testing.T) {
	style := Style{flags: styleFlags{bold: true}}
	f := style.FormatterHtml()
	expect(t, f.style == &style, "f.style == &style")
	expect(t, f.palette == nil, "f.palette == null")
}

// Zig: test "term51: Style.VTFormatter.format: empty style with palette still resets"
func TestStyle_term51_Style_VTFormatter_format_empty_style_with_palette_still_resets(t *testing.T) {
	style := Style{}
	f := style.FormatterVt()
	f.palette = &ColorDefault
	expectEqualStrings(t, "\x1b[0m", f.Format())
}

// Zig: test "term51: Style.VTFormatter.format: palette leaves rgb and none alone"
func TestStyle_term51_Style_VTFormatter_format_palette_leaves_rgb_and_none_alone(t *testing.T) {
	pal := sm3_palette()
	style := Style{
		fg_color:        sm3_rgb(1, 2, 3),
		bg_color:        StyleColor{},
		underline_color: sm3_pal(0),
	}
	f := style.FormatterVt()
	f.palette = &pal
	expectEqualStrings(t,
		"\x1b[0m\x1b[38;2;1;2;3m\x1b[58;2;0;255;100m",
		f.Format(),
	)
}

// Zig: test "term51: Style.VTFormatter.format: full ordering without palette"
func TestStyle_term51_Style_VTFormatter_format_full_ordering_without_palette(t *testing.T) {
	style := Style{
		fg_color:        sm3_pal(0),
		bg_color:        sm3_pal(255),
		underline_color: sm3_rgb(255, 255, 255),
		flags: styleFlags{
			bold:          true,
			faint:         true,
			italic:        true,
			blink:         true,
			inverse:       true,
			invisible:     true,
			strikethrough: true,
			overline:      true,
			underline:     SGRAttributeUnderlineDashed,
		},
	}
	expectEqualStrings(t,
		"\x1b[0m\x1b[1m\x1b[2m\x1b[3m\x1b[5m\x1b[7m\x1b[8m\x1b[9m\x1b[53m\x1b[4:5m"+
			"\x1b[38;5;0m\x1b[48;5;255m\x1b[58;2;255;255;255m",
		style.FormatterVt().Format(),
	)
}

// Zig: test "term51: Style.VTFormatter.format: full ordering with palette"
func TestStyle_term51_Style_VTFormatter_format_full_ordering_with_palette(t *testing.T) {
	pal := sm3_palette()
	style := Style{
		fg_color:        sm3_pal(0),
		bg_color:        sm3_pal(255),
		underline_color: sm3_pal(128),
		flags:           styleFlags{underline: SGRAttributeUnderlineSingle},
	}
	f := style.FormatterVt()
	f.palette = &pal
	expectEqualStrings(t,
		"\x1b[0m\x1b[4m\x1b[38;2;0;255;100m\x1b[48;2;255;0;100m\x1b[58;2;128;127;100m",
		f.Format(),
	)
}

// Zig: test "term51: Style.VTFormatter.format: each underline style alone"
func TestStyle_term51_Style_VTFormatter_format_each_underline_style_alone(t *testing.T) {
	cases := []struct {
		u    SGRAttributeUnderline
		want string
	}{
		{u: SGRAttributeUnderlineNone, want: "\x1b[0m"},
		{u: SGRAttributeUnderlineSingle, want: "\x1b[0m\x1b[4m"},
		{u: SGRAttributeUnderlineDouble, want: "\x1b[0m\x1b[4:2m"},
		{u: SGRAttributeUnderlineCurly, want: "\x1b[0m\x1b[4:3m"},
		{u: SGRAttributeUnderlineDotted, want: "\x1b[0m\x1b[4:4m"},
		{u: SGRAttributeUnderlineDashed, want: "\x1b[0m\x1b[4:5m"},
	}
	for _, c := range cases {
		style := Style{flags: styleFlags{underline: c.u}}
		expectEqualStrings(t, c.want, style.FormatterVt().Format())
	}
}

// Zig: test "term51: Style.VTFormatter.formatColor: none emits nothing"
func TestStyle_term51_Style_VTFormatter_formatColor_none_emits_nothing(t *testing.T) {
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	style := Style{}
	pal := sm3_palette()
	f := style.FormatterVt()
	zigTry(t, f.formatColor(&builder.writer, 38, StyleColor{}))
	expectEqualStrings(t, "", sm3_buffered(&builder.writer))
	f.palette = &pal
	zigTry(t, f.formatColor(&builder.writer, 38, StyleColor{}))
	expectEqualStrings(t, "", sm3_buffered(&builder.writer))
}

// Zig: test "term51: Style.VTFormatter.formatColor: palette index without palette"
func TestStyle_term51_Style_VTFormatter_formatColor_palette_index_without_palette(t *testing.T) {
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	style := Style{}
	f := style.FormatterVt()
	zigTry(t, f.formatColor(&builder.writer, 38, sm3_pal(0)))
	expectEqualStrings(t, "\x1b[38;5;0m", sm3_buffered(&builder.writer))
	builder.writer.end = 0
	zigTry(t, f.formatColor(&builder.writer, 48, sm3_pal(255)))
	expectEqualStrings(t, "\x1b[48;5;255m", sm3_buffered(&builder.writer))
	builder.writer.end = 0
	zigTry(t, f.formatColor(&builder.writer, 58, sm3_pal(17)))
	expectEqualStrings(t, "\x1b[58;5;17m", sm3_buffered(&builder.writer))
	builder.writer.end = 0
	// The prefix is used verbatim.
	zigTry(t, f.formatColor(&builder.writer, 99, sm3_pal(1)))
	expectEqualStrings(t, "\x1b[99;5;1m", sm3_buffered(&builder.writer))
}

// Zig: test "term51: Style.VTFormatter.formatColor: palette index with palette"
func TestStyle_term51_Style_VTFormatter_formatColor_palette_index_with_palette(t *testing.T) {
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	style := Style{}
	pal := sm3_palette()
	f := style.FormatterVt()
	f.palette = &pal
	zigTry(t, f.formatColor(&builder.writer, 38, sm3_pal(0)))
	expectEqualStrings(t, "\x1b[38;2;0;255;100m", sm3_buffered(&builder.writer))
	builder.writer.end = 0
	zigTry(t, f.formatColor(&builder.writer, 48, sm3_pal(255)))
	expectEqualStrings(t, "\x1b[48;2;255;0;100m", sm3_buffered(&builder.writer))
	builder.writer.end = 0
	zigTry(t, f.formatColor(&builder.writer, 58, sm3_pal(1)))
	expectEqualStrings(t, "\x1b[58;2;1;254;100m", sm3_buffered(&builder.writer))
}

// Zig: test "term51: Style.VTFormatter.formatColor: rgb"
func TestStyle_term51_Style_VTFormatter_formatColor_rgb(t *testing.T) {
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	style := Style{}
	pal := sm3_palette()
	f := style.FormatterVt()
	zigTry(t, f.formatColor(&builder.writer, 38, sm3_rgb(0, 0, 0)))
	expectEqualStrings(t, "\x1b[38;2;0;0;0m", sm3_buffered(&builder.writer))
	builder.writer.end = 0
	f.palette = &pal
	zigTry(t, f.formatColor(&builder.writer, 48, sm3_rgb(255, 254, 253)))
	expectEqualStrings(t, "\x1b[48;2;255;254;253m", sm3_buffered(&builder.writer))
}

// Zig: test "term51: Style.HtmlFormatter.format: empty style"
func TestStyle_term51_Style_HtmlFormatter_format_empty_style(t *testing.T) {
	style := Style{}
	expectEqualStrings(t, "", style.FormatterHtml().Format())
}

// Zig: test "term51: Style.HtmlFormatter.format: each flag alone"
func TestStyle_term51_Style_HtmlFormatter_format_each_flag_alone(t *testing.T) {
	cases := []struct {
		flags styleFlags
		want  string
	}{
		{flags: styleFlags{bold: true}, want: "font-weight: bold;"},
		{flags: styleFlags{italic: true}, want: "font-style: italic;"},
		{flags: styleFlags{faint: true}, want: "opacity: 0.5;"},
		{flags: styleFlags{invisible: true}, want: "visibility: hidden;"},
		{flags: styleFlags{inverse: true}, want: "filter: invert(100%);"},
		{flags: styleFlags{blink: true}, want: "text-decoration-line: blink;"},
		{flags: styleFlags{strikethrough: true}, want: "text-decoration-line: line-through;"},
		{flags: styleFlags{overline: true}, want: "text-decoration-line: overline;"},
	}
	for _, c := range cases {
		style := Style{flags: c.flags}
		expectEqualStrings(t, c.want, style.FormatterHtml().Format())
	}
}

// Zig: test "term51: Style.HtmlFormatter.format: each underline style alone"
func TestStyle_term51_Style_HtmlFormatter_format_each_underline_style_alone(t *testing.T) {
	cases := []struct {
		u    SGRAttributeUnderline
		want string
	}{
		{u: SGRAttributeUnderlineNone, want: ""},
		{u: SGRAttributeUnderlineSingle, want: "text-decoration-line: underline;text-decoration-style: solid;"},
		{u: SGRAttributeUnderlineDouble, want: "text-decoration-line: underline;text-decoration-style: double;"},
		{u: SGRAttributeUnderlineCurly, want: "text-decoration-line: underline;text-decoration-style: wavy;"},
		{u: SGRAttributeUnderlineDotted, want: "text-decoration-line: underline;text-decoration-style: dotted;"},
		{u: SGRAttributeUnderlineDashed, want: "text-decoration-line: underline;text-decoration-style: dashed;"},
	}
	for _, c := range cases {
		style := Style{flags: styleFlags{underline: c.u}}
		expectEqualStrings(t, c.want, style.FormatterHtml().Format())
	}
}

// Zig: test "term51: Style.HtmlFormatter.format: decoration line combinations"
func TestStyle_term51_Style_HtmlFormatter_format_decoration_line_combinations(t *testing.T) {
	// All four decoration lines, in fixed order.
	all := Style{flags: styleFlags{
		underline:     SGRAttributeUnderlineSingle,
		strikethrough: true,
		overline:      true,
		blink:         true,
	}}
	expectEqualStrings(t,
		"text-decoration-line: underline line-through overline blink;text-decoration-style: solid;",
		all.FormatterHtml().Format(),
	)

	sb := Style{flags: styleFlags{strikethrough: true, blink: true}}
	expectEqualStrings(t,
		"text-decoration-line: line-through blink;",
		sb.FormatterHtml().Format(),
	)

	ob := Style{flags: styleFlags{overline: true, blink: true}}
	expectEqualStrings(t,
		"text-decoration-line: overline blink;",
		ob.FormatterHtml().Format(),
	)

	us := Style{flags: styleFlags{underline: SGRAttributeUnderlineCurly, strikethrough: true}}
	expectEqualStrings(t,
		"text-decoration-line: underline line-through;text-decoration-style: wavy;",
		us.FormatterHtml().Format(),
	)
}

// Zig: test "term51: Style.HtmlFormatter.format: full ordering without palette"
func TestStyle_term51_Style_HtmlFormatter_format_full_ordering_without_palette(t *testing.T) {
	style := Style{
		fg_color:        sm3_pal(1),
		bg_color:        sm3_rgb(2, 3, 4),
		underline_color: sm3_pal(255),
		flags: styleFlags{
			bold:          true,
			italic:        true,
			faint:         true,
			blink:         true,
			inverse:       true,
			invisible:     true,
			strikethrough: true,
			overline:      true,
			underline:     SGRAttributeUnderlineDotted,
		},
	}
	expectEqualStrings(t,
		"color: var(--vt-palette-1);"+
			"background-color: rgb(2, 3, 4);"+
			"text-decoration-color: var(--vt-palette-255);"+
			"text-decoration-line: underline line-through overline blink;"+
			"text-decoration-style: dotted;"+
			"font-weight: bold;"+
			"font-style: italic;"+
			"opacity: 0.5;"+
			"visibility: hidden;"+
			"filter: invert(100%);",
		style.FormatterHtml().Format(),
	)
}

// Zig: test "term51: Style.HtmlFormatter.format: palette set leaves rgb alone"
func TestStyle_term51_Style_HtmlFormatter_format_palette_set_leaves_rgb_alone(t *testing.T) {
	pal := sm3_palette()
	style := Style{
		fg_color:        sm3_pal(0),
		bg_color:        sm3_rgb(2, 3, 4),
		underline_color: sm3_pal(255),
	}
	f := style.FormatterHtml()
	f.palette = &pal
	expectEqualStrings(t,
		"color: rgb(0, 255, 100);"+
			"background-color: rgb(2, 3, 4);"+
			"text-decoration-color: rgb(255, 0, 100);",
		f.Format(),
	)
}

// Zig: test "term51: Style.HtmlFormatter.formatColor: none emits nothing"
func TestStyle_term51_Style_HtmlFormatter_formatColor_none_emits_nothing(t *testing.T) {
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	style := Style{}
	pal := sm3_palette()
	f := style.FormatterHtml()
	zigTry(t, f.formatColor(&builder.writer, []uint8("color"), StyleColor{}))
	f.palette = &pal
	zigTry(t, f.formatColor(&builder.writer, []uint8("color"), StyleColor{}))
	expectEqualStrings(t, "", sm3_buffered(&builder.writer))
}

// Zig: test "term51: Style.HtmlFormatter.formatColor: palette index"
func TestStyle_term51_Style_HtmlFormatter_formatColor_palette_index(t *testing.T) {
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	style := Style{}
	pal := sm3_palette()
	f := style.FormatterHtml()
	zigTry(t, f.formatColor(&builder.writer, []uint8("color"), sm3_pal(0)))
	expectEqualStrings(t, "color: var(--vt-palette-0);", sm3_buffered(&builder.writer))
	builder.writer.end = 0
	zigTry(t, f.formatColor(&builder.writer, []uint8("x-prop"), sm3_pal(255)))
	expectEqualStrings(t, "x-prop: var(--vt-palette-255);", sm3_buffered(&builder.writer))

	builder.writer.end = 0
	f.palette = &pal
	zigTry(t, f.formatColor(&builder.writer, []uint8("color"), sm3_pal(0)))
	expectEqualStrings(t, "color: rgb(0, 255, 100);", sm3_buffered(&builder.writer))
	builder.writer.end = 0
	zigTry(t, f.formatColor(&builder.writer, []uint8("background-color"), sm3_pal(200)))
	expectEqualStrings(t, "background-color: rgb(200, 55, 100);", sm3_buffered(&builder.writer))
}

// Zig: test "term51: Style.HtmlFormatter.formatColor: rgb"
func TestStyle_term51_Style_HtmlFormatter_formatColor_rgb(t *testing.T) {
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	style := Style{}
	f := style.FormatterHtml()
	zigTry(t, f.formatColor(&builder.writer, []uint8("color"), sm3_rgb(0, 0, 0)))
	expectEqualStrings(t, "color: rgb(0, 0, 0);", sm3_buffered(&builder.writer))
	builder.writer.end = 0
	pal := sm3_palette()
	f.palette = &pal
	zigTry(t, f.formatColor(&builder.writer, []uint8("color"), sm3_rgb(255, 128, 1)))
	expectEqualStrings(t, "color: rgb(255, 128, 1);", sm3_buffered(&builder.writer))
}

// Zig: test "term51: Style.PackedStyle.Data.fromColor: each tag"
func TestStyle_term51_Style_PackedStyle_Data_fromColor_each_tag(t *testing.T) {
	// Zig: `@bitCast` of the packed union to a u24; the Go Data is the 24-bit
	// integer itself.
	none := stylePackedStyleDataFromColor(StyleColor{})
	expectEqual(t, stylePackedStyleData(0), none)

	for _, idx := range []uint8{0, 1, 200, 255} {
		bits := stylePackedStyleDataFromColor(sm3_pal(idx))
		expectEqual(t, stylePackedStyleData(idx), bits)
	}

	rgb := stylePackedStyleDataFromColor(sm3_rgb(0x11, 0x22, 0x33))
	expectEqual(t, stylePackedStyleData(0x332211), rgb)

	white := stylePackedStyleDataFromColor(sm3_rgb(255, 255, 255))
	expectEqual(t, stylePackedStyleData(0xFFFFFF), white)
}

// Zig: test "term51: Style.PackedStyle.fromStyle: layout bit positions"
func TestStyle_term51_Style_PackedStyle_fromStyle_layout_bit_positions(t *testing.T) {
	// Layout (LSB first): 3 tag bytes (fg, bg, underline), 3 24-bit data
	// fields (fg, bg, underline), the 16 flag bits, 16 zero bits.

	// Zig-only: expect(@sizeOf(P) == 16) and expect(@bitSizeOf(P) == 128): memory layout of the packed struct.

	{
		bits := stylePackedStyleFromStyle(Style{}).bitCast()
		expectEqual(t, [2]uint64{0, 0}, bits)
	}

	{
		s := Style{fg_color: sm3_pal(5)}
		bits := stylePackedStyleFromStyle(s).bitCast()
		expectEqual(t, sm3_or128(sm3_u128(1, 0), sm3_u128(5, 24)), bits)
	}

	{
		s := Style{bg_color: sm3_rgb(0x11, 0x22, 0x33)}
		bits := stylePackedStyleFromStyle(s).bitCast()
		expectEqual(t,
			sm3_or128(sm3_u128(2, 8), sm3_u128(0x332211, 48)),
			bits,
		)
	}

	{
		s := Style{underline_color: sm3_rgb(0xAA, 0xBB, 0xCC)}
		bits := stylePackedStyleFromStyle(s).bitCast()
		expectEqual(t,
			sm3_or128(sm3_u128(2, 16), sm3_u128(0xCCBBAA, 72)),
			bits,
		)
	}

	{
		// bold = bit 0, underline curly = 3 in bits 8..10 of the flags.
		s := Style{flags: styleFlags{bold: true, underline: SGRAttributeUnderlineCurly}}
		bits := stylePackedStyleFromStyle(s).bitCast()
		expectEqual(t, sm3_u128(0x301, 96), bits)
	}

	{
		// Every flag bit, in declaration order.
		s := Style{flags: styleFlags{
			bold:          true,
			italic:        true,
			faint:         true,
			blink:         true,
			inverse:       true,
			invisible:     true,
			strikethrough: true,
			overline:      true,
			underline:     SGRAttributeUnderlineDashed,
		}}
		bits := stylePackedStyleFromStyle(s).bitCast()
		expectEqual(t, sm3_u128(0x5FF, 96), bits)
	}

	{
		// Padding bits of the flags are carried over; the trailing 16 are 0.
		s := Style{flags: styleFlags{_padding: 0b11111}}
		bits := stylePackedStyleFromStyle(s).bitCast()
		expectEqual(t, sm3_u128(0xF800, 96), bits)
	}
}

// Zig: test "term51: Style.hash: pinned values"
func TestStyle_term51_Style_hash_pinned_values(t *testing.T) {
	h := sm3_hashInt

	d := Style{}
	expectEqual(t, h(0), d.Hash())

	// wide[0] holds bits 0..63, wide[1] holds bits 64..127; the hash is
	// std.hash.int(wide[0] ^ wide[1]).
	fg := Style{fg_color: sm3_pal(5)}
	expectEqual(t, h(uint64(1)|(uint64(5)<<24)), fg.Hash())

	bg := Style{bg_color: sm3_rgb(0x11, 0x22, 0x33)}
	expectEqual(t,
		h(((uint64(2)<<8)|(uint64(0x11)<<48)|(uint64(0x22)<<56))^0x33),
		bg.Hash(),
	)

	ul := Style{underline_color: sm3_rgb(0xAA, 0xBB, 0xCC)}
	expectEqual(t,
		h((uint64(2)<<16)^((uint64(0xAA)<<8)|(uint64(0xBB)<<16)|(uint64(0xCC)<<24))),
		ul.Hash(),
	)

	fl := Style{flags: styleFlags{bold: true, underline: SGRAttributeUnderlineCurly}}
	expectEqual(t, h(uint64(0x301)<<32), fl.Hash())
}

// Zig: test "term51: Style.hash: equal styles hash equal"
func TestStyle_term51_Style_hash_equal_styles_hash_equal(t *testing.T) {
	a := Style{
		fg_color: sm3_rgb(1, 2, 3),
		bg_color: sm3_pal(9),
		flags:    styleFlags{italic: true, underline: SGRAttributeUnderlineSingle},
	}
	b := Style{}
	b.flags.underline = SGRAttributeUnderlineSingle
	b.flags.italic = true
	b.bg_color = sm3_pal(9)
	b.fg_color = sm3_rgb(1, 2, 3)
	expect(t, a.Eql(b), "a.eql(b)")
	expectEqual(t, a.Hash(), b.Hash())
	expectEqual(t, a.Hash(), a.Hash())
}

// Zig: test "term51: Style.hash: every one-field difference hashes differently"
func TestStyle_term51_Style_hash_every_one_field_difference_hashes_differently(t *testing.T) {
	variants := sm3_variants()
	// Zig-only: defer variants.deinit(alloc)
	variants = append(variants, Style{})

	for i, a := range variants {
		for _, b := range variants[i+1:] {
			expect(t, a.Hash() != b.Hash(), "a.hash() != b.hash()")
		}
	}
}

// Zig: test "term51: Style.hash: folding 128 to 64 bits can collide"
func TestStyle_term51_Style_hash_folding_128_to_64_bits_can_collide(t *testing.T) {
	// Pins a weakness: the hash XORs the two halves, so bit 96 (bold)
	// cancels against bit 32 (the low bit of the fg rgb green channel).
	// The styles are distinct, only the set's eql tells them apart.
	a := Style{fg_color: sm3_rgb(0, 1, 0)}
	b := Style{
		fg_color: sm3_rgb(0, 0, 0),
		flags:    styleFlags{bold: true},
	}
	expect(t, !a.Eql(b), "!a.eql(b)")
	expectEqual(t, a.Hash(), b.Hash())
}

// Zig: test "term51: hash: set context hash is Style.hash"
func TestStyle_term51_hash_set_context_hash_is_Style_hash(t *testing.T) {
	ctx := StyleSetContext{}
	styles := []Style{
		{},
		{flags: styleFlags{bold: true}},
		{fg_color: sm3_pal(3)},
		{bg_color: sm3_rgb(1, 2, 3)},
	}
	for _, s := range styles {
		expectEqual(t, s.Hash(), ctx.Hash(s))
	}
}

// Zig: test "term51: eql: set context eql is Style.eql"
func TestStyle_term51_eql_set_context_eql_is_Style_eql(t *testing.T) {
	ctx := StyleSetContext{}

	variants := sm3_variants()
	// Zig-only: defer variants.deinit(alloc)
	variants = append(variants, Style{})
	for i, a := range variants {
		for j, b := range variants {
			expectEqual(t, i == j, ctx.Eql(a, b))
		}
	}
}

// Zig: test "term51: eql: set instantiation types and layout"
func TestStyle_term51_eql_set_instantiation_types_and_layout(t *testing.T) {
	// Set.Id is the third type argument of StyleSet; the field next_id has
	// that type.
	var inst StyleSet
	expect(t, reflect.TypeOf(inst.next_id) == reflect.TypeOf(StyleId(0)), "Set.Id == Id")
	expect(t, reflect.TypeOf(inst.next_id) == reflect.TypeOf(uint16(0)), "Set.Id == u16")
	expectEqual(t, float64(0.8125), float64(RefCountedSetLoadFactor))
	expectEqual(t, uint(0), RefCountedSetCapacityForCount(0))
	expectEqual(t, uint(3), RefCountedSetCapacityForCount(1))
	expectEqual(t, uint(16), RefCountedSetCapacityForCount(12))
	expectEqual(t, uint(18), RefCountedSetCapacityForCount(13))
	expectEqual(t, uint(80), RefCountedSetCapacityForCount(64))

	// Zero capacity.
	{
		l := RefCountedSetLayoutInit[StyleId](0)
		expectEqual(t, uint(0), l.cap)
		expectEqual(t, uint(0), l.table_cap)
		expectEqual(t, StyleId(0), l.table_mask)
		// Zig-only: expectEqual(@as(usize, 0), l.total_size)
	}

	// The cap is the rounded-up power of two times the load factor,
	// rounded down. IDs 1..cap-1 are usable.
	cases := []struct {
		req   uint
		table uint
		cap   uint
	}{
		{req: 1, table: 1, cap: 0},
		{req: 2, table: 2, cap: 1},
		{req: 3, table: 4, cap: 3},
		{req: 4, table: 4, cap: 3},
		{req: 5, table: 8, cap: 6},
		{req: 8, table: 8, cap: 6},
		{req: 16, table: 16, cap: 13},
		{req: 17, table: 32, cap: 26},
		{req: 128, table: 128, cap: 104},
		{req: 16384, table: 16384, cap: 13312},
		{req: 32769, table: 65536, cap: 53248},
		{req: 65535, table: 65536, cap: 53248},
		{req: 65536, table: 65536, cap: 53248},
	}
	for _, c := range cases {
		l := RefCountedSetLayoutInit[StyleId](c.req)
		expectEqual(t, c.table, l.table_cap)
		expectEqual(t, c.cap, l.cap)
		expectEqual(t, StyleId(c.table-1), l.table_mask)
		// Zig-only: expectEqual(@as(usize, 0), l.table_start)
		// Zig-only: expect(l.items_start >= l.table_cap * @sizeOf(Id))
		// Zig-only: expect(l.items_start % @alignOf(Set.Item) == 0)
		// Zig-only: expectEqual(l.items_start + l.cap * @sizeOf(Set.Item), l.total_size)
	}
}

// Zig: test "term51: eql: set first id is 1 and default style is storable"
func TestStyle_term51_eql_set_first_id_is_1_and_default_style_is_storable(t *testing.T) {
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	expectEqual(t, uint(0), set.Count())
	expectEqual(t, StyleId(1), set.next_id)

	// The set does not special-case the default style: it is just a
	// value that gets id 1 (id 0 is reserved and never assigned).
	id := sm3_add(t, &set, Style{})
	expectEqual(t, StyleId(1), id)
	expectEqual(t, uint(1), set.Count())
	expect(t, set.Get(id).Default(), "set.get(buf, id).default()")
	expectEqual(t, StyleId(1), zigUnwrap(t, set.Lookup(Style{})))

	id2 := sm3_add(t, &set, Style{flags: styleFlags{bold: true}})
	expectEqual(t, StyleId(2), id2)
}

// Zig: test "term51: eql: set lookup and add distinguish every one-field difference"
func TestStyle_term51_eql_set_lookup_and_add_distinguish_every_one_field_difference(t *testing.T) {
	variants := sm3_variants()
	// Zig-only: defer variants.deinit(alloc)
	variants = append(variants, Style{})

	// 36 variants: table of 64 gives cap 52, so 51 usable ids.
	layout := RefCountedSetLayoutInit[StyleId](64)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	for _, v := range variants {
		expectEqual(t, None[StyleId](), set.Lookup(v))
	}

	var ids []StyleId
	// Zig-only: defer ids.deinit(alloc)
	for i, v := range variants {
		id := sm3_add(t, &set, v)
		expectEqual(t, StyleId(i+1), id)
		ids = append(ids, id)
	}
	expectEqual(t, uint(len(variants)), set.Count())

	for i, v := range variants {
		id := ids[i]
		expectEqual(t, Some(id), set.Lookup(v))
		expectEqual(t, uint16(1), set.RefCount(id))
		expect(t, set.Get(id).Eql(v), "set.get(buf, id).eql(v)")
	}
}

// Zig: test "term51: eql: set add of a hash-colliding but unequal style"
func TestStyle_term51_eql_set_add_of_a_hash_colliding_but_unequal_style(t *testing.T) {
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	a := Style{fg_color: sm3_rgb(0, 1, 0)}
	b := Style{
		fg_color: sm3_rgb(0, 0, 0),
		flags:    styleFlags{bold: true},
	}
	expectEqual(t, a.Hash(), b.Hash())

	ia := sm3_add(t, &set, a)
	ib := sm3_add(t, &set, b)
	expect(t, ia != ib, "ia != ib")
	expectEqual(t, ia, zigUnwrap(t, set.Lookup(a)))
	expectEqual(t, ib, zigUnwrap(t, set.Lookup(b)))
	expectEqual(t, uint(2), set.Count())
}

// Zig: test "term51: eql: set fills to capacity then reports OutOfMemory"
func TestStyle_term51_eql_set_fills_to_capacity_then_reports_OutOfMemory(t *testing.T) {
	// Layout.init(16): table 16, cap 13, so ids 1..12 are usable.
	layout := RefCountedSetLayoutInit[StyleId](16)
	expectEqual(t, uint(13), layout.cap)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	for i := 1; i < 13; i++ {
		id := sm3_add(t, &set, sm3_rgbStyle(uint16(i)))
		expectEqual(t, StyleId(i), id)
	}
	expectEqual(t, uint(12), set.Count())

	// One past the limit, with every item alive: OutOfMemory (not rehash).
	_, err := set.Add(sm3_rgbStyle(13))
	sm3_expectError(t, ErrOutOfMemory, err)
	expectEqual(t, uint(12), set.Count())
	expectEqual(t, None[StyleId](), set.Lookup(sm3_rgbStyle(13)))

	// Re-adding an existing style still works at capacity and bumps the ref.
	expectEqual(t, StyleId(5), sm3_add(t, &set, sm3_rgbStyle(5)))
	expectEqual(t, uint16(2), set.RefCount(5))
	expectEqual(t, uint(12), set.Count())
}

// Zig: test "term51: eql: set full with dead items reports NeedsRehash"
func TestStyle_term51_eql_set_full_with_dead_items_reports_NeedsRehash(t *testing.T) {
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	for i := 1; i < 13; i++ {
		_ = sm3_add(t, &set, sm3_rgbStyle(uint16(i)))
	}

	// Threshold: living < floor(13 * 0.9) = 11 yields NeedsRehash.
	// 11 living is still OutOfMemory.
	set.Release(1)
	expectEqual(t, uint(11), set.Count())
	_, err := set.Add(sm3_rgbStyle(100))
	sm3_expectError(t, ErrOutOfMemory, err)

	set.Release(2)
	expectEqual(t, uint(10), set.Count())
	_, err = set.Add(sm3_rgbStyle(100))
	sm3_expectError(t, ErrNeedsRehash, err)

	// Dead items stay in the set, so no new ids become available.
	expectEqual(t, StyleId(13), set.next_id)
}

// Zig: test "term51: eql: set release of the last id frees its id for reuse"
func TestStyle_term51_eql_set_release_of_the_last_id_frees_its_id_for_reuse(t *testing.T) {
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	for i := 1; i < 13; i++ {
		_ = sm3_add(t, &set, sm3_rgbStyle(uint16(i)))
	}
	set.Release(12)
	expectEqual(t, uint(11), set.Count())

	// The trailing dead item is trimmed, which frees id 12 again.
	id := sm3_add(t, &set, sm3_rgbStyle(500))
	expectEqual(t, StyleId(12), id)
	expectEqual(t, uint(12), set.Count())
	expectEqual(t, None[StyleId](), set.Lookup(sm3_rgbStyle(12)))
}

// Zig: test "term51: eql: set tiny layouts have no usable ids"
func TestStyle_term51_eql_set_tiny_layouts_have_no_usable_ids(t *testing.T) {
	for _, req := range []uint{1, 2} {
		layout := RefCountedSetLayoutInit[StyleId](req)
		// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
		set := sm3_newSet(layout)
		_, err := set.Add(Style{})
		sm3_expectError(t, ErrOutOfMemory, err)
		expectEqual(t, uint(0), set.Count())
	}

	// table 4: cap 3, two usable ids.
	{
		layout := RefCountedSetLayoutInit[StyleId](4)
		// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
		set := sm3_newSet(layout)
		expectEqual(t, StyleId(1), sm3_add(t, &set, sm3_rgbStyle(1)))
		expectEqual(t, StyleId(2), sm3_add(t, &set, sm3_rgbStyle(2)))
		_, err := set.Add(sm3_rgbStyle(3))
		sm3_expectError(t, ErrOutOfMemory, err)
	}
}

// Zig: test "term51: eql: set release and resurrect a dead item"
func TestStyle_term51_eql_set_release_and_resurrect_a_dead_item(t *testing.T) {
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	a := sm3_rgbStyle(1)
	b := sm3_rgbStyle(2)
	ia := sm3_add(t, &set, a)
	ib := sm3_add(t, &set, b)
	expectEqual(t, uint(2), set.Count())

	// Releasing to zero makes the item dead: not found, not counted.
	set.Release(ia)
	expectEqual(t, uint16(0), set.RefCount(ia))
	expectEqual(t, uint(1), set.Count())
	expectEqual(t, None[StyleId](), set.Lookup(a))
	expectEqual(t, Some(ib), set.Lookup(b))

	// Adding it again resurrects it under the same id.
	expectEqual(t, ia, sm3_add(t, &set, a))
	expectEqual(t, uint16(1), set.RefCount(ia))
	expectEqual(t, uint(2), set.Count())
	expectEqual(t, StyleId(3), set.next_id)
}

// Zig: test "term51: eql: set use, useMultiple, release, releaseMultiple"
func TestStyle_term51_eql_set_use_useMultiple_release_releaseMultiple(t *testing.T) {
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	id := sm3_add(t, &set, Style{flags: styleFlags{bold: true}})
	expectEqual(t, uint16(1), set.RefCount(id))
	set.Use(id)
	expectEqual(t, uint16(2), set.RefCount(id))
	set.UseMultiple(id, 10)
	expectEqual(t, uint16(12), set.RefCount(id))
	expectEqual(t, uint(1), set.Count())

	set.Release(id)
	expectEqual(t, uint16(11), set.RefCount(id))
	set.ReleaseMultiple(id, 10)
	expectEqual(t, uint16(1), set.RefCount(id))
	expectEqual(t, uint(1), set.Count())
	set.ReleaseMultiple(id, 1)
	expectEqual(t, uint16(0), set.RefCount(id))
	expectEqual(t, uint(0), set.Count())
}

// Zig: test "term51: eql: set get returns a pointer to the stored style"
func TestStyle_term51_eql_set_get_returns_a_pointer_to_the_stored_style(t *testing.T) {
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	id := sm3_add(t, &set, Style{fg_color: sm3_pal(4)})
	p1 := set.Get(id)
	p2 := set.Get(id)
	expect(t, p1 == p2, "p1 == p2")
	// Zig compares the u16 ref count with a usize literal.
	expectEqual(t, uint16(1), set.RefCount(id))
	expect(t, p1.Eql(Style{fg_color: sm3_pal(4)}), "p1.eql(.{ .fg_color = .{ .palette = 4 } })")
}

// Zig: test "term51: eql: set addWithId"
func TestStyle_term51_eql_set_addWithId(t *testing.T) {
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	a := sm3_rgbStyle(1)
	b := sm3_rgbStyle(2)
	c := sm3_rgbStyle(3)

	// An id beyond next_id is ignored: falls back to a plain add. The
	// add path returns the id it assigned, which is returned as non-null
	// even when it equals the requested id.
	expectEqual(t, Some(StyleId(1)), sm3_addWithId(t, &set, a, 1))
	expectEqual(t, Some(StyleId(2)), sm3_addWithId(t, &set, b, 9))
	expectEqual(t, uint(2), set.Count())

	// Existing live item with an equal value: ref bumped, null returned.
	expectEqual(t, None[StyleId](), sm3_addWithId(t, &set, a, 1))
	expectEqual(t, uint16(2), set.RefCount(1))

	// Existing live item at that id with a different value: plain add.
	expectEqual(t, Some(StyleId(3)), sm3_addWithId(t, &set, c, 1))
	expectEqual(t, uint16(2), set.RefCount(1))
	expectEqual(t, uint(3), set.Count())

	// Dead item at that id: it is replaced and null is returned.
	set.Release(2)
	d := sm3_rgbStyle(4)
	expectEqual(t, None[StyleId](), sm3_addWithId(t, &set, d, 2))
	expect(t, set.Get(2).Eql(d), "set.get(buf, 2).eql(d)")
	expectEqual(t, uint16(1), set.RefCount(2))
	expectEqual(t, None[StyleId](), set.Lookup(b))
	expectEqual(t, uint(3), set.Count())
}

// Zig: test "term51: eql: set addWithId reusing a dead id for an existing value overcounts"
func TestStyle_term51_eql_set_addWithId_reusing_a_dead_id_for_an_existing_value_overcounts(t *testing.T) {
	// Pins a defect in upstream: when the dead id's replacement value
	// already lives under another id, upsert returns that other id but
	// `living` is still incremented, so count() drifts above the number
	// of items that are alive.
	layout := RefCountedSetLayoutInit[StyleId](16)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	a := sm3_rgbStyle(1)
	b := sm3_rgbStyle(2)
	expectEqual(t, StyleId(1), sm3_add(t, &set, a))
	expectEqual(t, StyleId(2), sm3_add(t, &set, b))
	set.Release(2)
	expectEqual(t, uint(1), set.Count())

	// `a` already lives at id 1, so the result is id 1, not null.
	expectEqual(t, Some(StyleId(1)), sm3_addWithId(t, &set, a, 2))
	expectEqual(t, uint16(2), set.RefCount(1))
	expectEqual(t, uint(2), set.Count())

	set.Release(1)
	set.Release(1)
	expectEqual(t, uint16(0), set.RefCount(1))
	// Nothing is alive but the counter says one item is.
	expectEqual(t, uint(1), set.Count())
}

// Zig: test "term51: eql: set of 16 bit ids holds 53247 styles"
func TestStyle_term51_eql_set_of_16_bit_ids_holds_53247_styles(t *testing.T) {
	// The largest layout a u16 id can address: table 65536, cap 53248.
	// Id 0 is reserved so 53247 styles fit, which is short of 65535.
	layout := RefCountedSetLayoutInit[StyleId](65535)
	expectEqual(t, uint(53248), layout.cap)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Set.base_align, layout.total_size); defer alloc.free(buf);
	set := sm3_newSet(layout)

	var n uint
	var last StyleId
	for n < 53247 {
		last = sm3_add(t, &set, sm3_rgbStyle(uint16(n)))
		n++
	}
	expectEqual(t, StyleId(53247), last)
	expectEqual(t, uint(53247), set.Count())
	_, err := set.Add(sm3_rgbStyle(60000))
	sm3_expectError(t, ErrOutOfMemory, err)
}

// Tests of `Style.fg`. They run in the library-only build because patch 0001
// declares `config.BoldColor` and `config.Color` locally there (copied from
// `config/Config.zig`). In Zig they sit in the struct term51FFgTests; in Go
// they are ordinary tests.

// Zig: test "term51: Style.fg: none color without bold"
func TestStyle_term51_Style_fg_none_color_without_bold(t *testing.T) {
	pal := sm3_palette()
	dflt := ColorRGB{r: 1, g: 2, b: 3}
	s := Style{}

	expectEqual(t, dflt, s.Fg(StyleFg{default_: dflt, palette: &pal}))
	expectEqual(t, dflt, s.Fg(StyleFg{default_: dflt, palette: &pal, bold: sm3_bright()}))
	expectEqual(t, dflt, s.Fg(StyleFg{
		default_: dflt,
		palette:  &pal,
		bold:     sm3_bold(9, 8, 7),
	}))
}

// Zig: test "term51: Style.fg: none color with bold"
func TestStyle_term51_Style_fg_none_color_with_bold(t *testing.T) {
	pal := sm3_palette()
	dflt := ColorRGB{r: 1, g: 2, b: 3}
	s := Style{flags: styleFlags{bold: true}}

	// No bold config, and bright, both leave the default untouched.
	expectEqual(t, dflt, s.Fg(StyleFg{default_: dflt, palette: &pal}))
	expectEqual(t, dflt, s.Fg(StyleFg{default_: dflt, palette: &pal, bold: sm3_bright()}))

	// A bold color replaces the default.
	expectEqual(t,
		ColorRGB{r: 9, g: 8, b: 7},
		s.Fg(StyleFg{
			default_: dflt,
			palette:  &pal,
			bold:     sm3_bold(9, 8, 7),
		}),
	)
}

// Zig: test "term51: Style.fg: palette color without bold"
func TestStyle_term51_Style_fg_palette_color_without_bold(t *testing.T) {
	pal := sm3_palette()
	dflt := ColorRGB{r: 1, g: 2, b: 3}

	for _, idx := range []uint8{0, 7, 8, 15, 16, 255} {
		s := Style{fg_color: sm3_pal(idx)}
		expectEqual(t, pal[idx], s.Fg(StyleFg{default_: dflt, palette: &pal}))
		expectEqual(t, pal[idx], s.Fg(StyleFg{default_: dflt, palette: &pal, bold: sm3_bright()}))
		expectEqual(t, pal[idx], s.Fg(StyleFg{
			default_: dflt,
			palette:  &pal,
			bold:     sm3_bold(9, 8, 7),
		}))
	}
}

// Zig: test "term51: Style.fg: palette color with bold and no bold config"
func TestStyle_term51_Style_fg_palette_color_with_bold_and_no_bold_config(t *testing.T) {
	pal := sm3_palette()
	dflt := ColorRGB{r: 1, g: 2, b: 3}

	for _, idx := range []uint8{0, 7, 8, 15, 16, 255} {
		s := Style{
			fg_color: sm3_pal(idx),
			flags:    styleFlags{bold: true},
		}
		expectEqual(t, pal[idx], s.Fg(StyleFg{default_: dflt, palette: &pal}))
	}
}

// Zig: test "term51: Style.fg: palette color with bold and bright"
func TestStyle_term51_Style_fg_palette_color_with_bold_and_bright(t *testing.T) {
	pal := sm3_palette()
	dflt := ColorRGB{r: 1, g: 2, b: 3}

	// 0..7 are promoted to 8..15.
	var idx uint8 = 0
	for idx < 8 {
		s := Style{
			fg_color: sm3_pal(idx),
			flags:    styleFlags{bold: true},
		}
		expectEqual(t,
			pal[idx+8],
			s.Fg(StyleFg{default_: dflt, palette: &pal, bold: sm3_bright()}),
		)
		idx++
	}

	// 8 and above are not promoted.
	for _, i := range []uint8{8, 9, 15, 16, 100, 255} {
		s := Style{
			fg_color: sm3_pal(i),
			flags:    styleFlags{bold: true},
		}
		expectEqual(t,
			pal[i],
			s.Fg(StyleFg{default_: dflt, palette: &pal, bold: sm3_bright()}),
		)
	}
}

// Zig: test "term51: Style.fg: palette color with bold and bold color config"
func TestStyle_term51_Style_fg_palette_color_with_bold_and_bold_color_config(t *testing.T) {
	// Pins upstream behaviour: any non-null bold option (including a
	// `.color`) promotes palette 0..7 to the bright variant; the color is
	// only used for the default and exact-default-rgb cases.
	pal := sm3_palette()
	dflt := ColorRGB{r: 1, g: 2, b: 3}

	s := Style{
		fg_color: sm3_pal(3),
		flags:    styleFlags{bold: true},
	}
	expectEqual(t, pal[11], s.Fg(StyleFg{
		default_: dflt,
		palette:  &pal,
		bold:     sm3_bold(9, 8, 7),
	}))

	s8 := Style{
		fg_color: sm3_pal(8),
		flags:    styleFlags{bold: true},
	}
	expectEqual(t, pal[8], s8.Fg(StyleFg{
		default_: dflt,
		palette:  &pal,
		bold:     sm3_bold(9, 8, 7),
	}))
}

// Zig: test "term51: Style.fg: non-bold palette color ignores bright"
func TestStyle_term51_Style_fg_non_bold_palette_color_ignores_bright(t *testing.T) {
	pal := sm3_palette()
	dflt := ColorRGB{r: 1, g: 2, b: 3}
	s := Style{fg_color: sm3_pal(2), flags: styleFlags{italic: true}}
	expectEqual(t, pal[2], s.Fg(StyleFg{default_: dflt, palette: &pal, bold: sm3_bright()}))
}

// Zig: test "term51: Style.fg: rgb color"
func TestStyle_term51_Style_fg_rgb_color(t *testing.T) {
	pal := sm3_palette()
	dflt := ColorRGB{r: 1, g: 2, b: 3}
	other := ColorRGB{r: 50, g: 60, b: 70}
	boldColor := sm3_bold(9, 8, 7)

	// Not bold: always the rgb value, even when equal to the default.
	{
		s := Style{fg_color: StyleColor{Tag: StyleColorRgb, Rgb: other}}
		expectEqual(t, other, s.Fg(StyleFg{default_: dflt, palette: &pal}))
		expectEqual(t, other, s.Fg(StyleFg{default_: dflt, palette: &pal, bold: sm3_bright()}))
		expectEqual(t, other, s.Fg(StyleFg{default_: dflt, palette: &pal, bold: boldColor}))

		same := Style{fg_color: StyleColor{Tag: StyleColorRgb, Rgb: dflt}}
		expectEqual(t, dflt, same.Fg(StyleFg{default_: dflt, palette: &pal, bold: boldColor}))
	}

	// Bold, rgb differing from the default: unchanged.
	{
		s := Style{fg_color: StyleColor{Tag: StyleColorRgb, Rgb: other}, flags: styleFlags{bold: true}}
		expectEqual(t, other, s.Fg(StyleFg{default_: dflt, palette: &pal}))
		expectEqual(t, other, s.Fg(StyleFg{default_: dflt, palette: &pal, bold: sm3_bright()}))
		expectEqual(t, other, s.Fg(StyleFg{default_: dflt, palette: &pal, bold: boldColor}))
	}

	// Bold, rgb equal to the default: replaced only by a bold color.
	{
		s := Style{fg_color: StyleColor{Tag: StyleColorRgb, Rgb: dflt}, flags: styleFlags{bold: true}}
		expectEqual(t, dflt, s.Fg(StyleFg{default_: dflt, palette: &pal}))
		expectEqual(t, dflt, s.Fg(StyleFg{default_: dflt, palette: &pal, bold: sm3_bright()}))
		expectEqual(t,
			ColorRGB{r: 9, g: 8, b: 7},
			s.Fg(StyleFg{default_: dflt, palette: &pal, bold: boldColor}),
		)
	}

	// Each channel of the default matters for the equality check.
	{
		channels := []ColorRGB{
			{r: 0, g: 2, b: 3},
			{r: 1, g: 0, b: 3},
			{r: 1, g: 2, b: 0},
		}
		for _, c := range channels {
			s := Style{fg_color: StyleColor{Tag: StyleColorRgb, Rgb: c}, flags: styleFlags{bold: true}}
			expectEqual(t, c, s.Fg(StyleFg{default_: dflt, palette: &pal, bold: boldColor}))
		}
	}
}

// Not translated (the unnamed test at style.zig:2739, `_ = term51FFgTests;`): see scripts/not-translated-sm3.txt.

// Not translated (Zig: test "term51: Style.hash: dump exact data when TERM51_SET_DUMP_DIR is set"): see scripts/not-translated-sm3.txt. corpus_style_test.go compares data/sets/style_hash.txt instead.
