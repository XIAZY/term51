// Tests translated one for one from src/terminal/style.zig (Ghostty v1.3.1
// plus the term51 additional tests): the six `Style.bg` and `Style.bgCell`
// tests, which style_test.go left out because they need the page wireframe
// (the Cell type). Each Go test carries the exact Zig title in a "// Zig: test"
// comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual keeps Zig's (expected, actual) argument order. A `?color.RGB`
//     is Optional[ColorRGB]; `@as(?color.RGB, null)` is None[ColorRGB]().
//   - A `page.Cell` literal is a PageCell literal; the untagged union
//     `content` is PageCellContent with the field that the content tag names
//     (page.go). The Go zero value of a PageCell is Zig's `.{}`.
//   - Zig's `inline for` over the two content tags is a plain loop.

package terminal

import "testing"

// sbgPalette is term51FPalette: a palette where every entry is distinct and
// derived from its index.
func sbgPalette() ColorPalette {
	var p ColorPalette
	for i := range p {
		p[i] = ColorRGB{
			R: uint8(i),
			G: uint8(255 - i),
			B: 100,
		}
	}
	return p
}

// sbgPal is the Zig literal `.{ .palette = idx }` of a Style.Color.
func sbgPal(idx uint8) StyleColor {
	return StyleColor{Tag: StyleColorPalette, Palette: idx}
}

// sbgRgb is the Zig literal `.{ .rgb = .{ .r = r, .g = g, .b = b } }` of a
// Style.Color.
func sbgRgb(r uint8, g uint8, b uint8) StyleColor {
	return StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{R: r, G: g, B: b}}
}

// Zig: test "term51: Style.bg: cell content tag decides the source"
func TestStyle_term51_Style_bg_cell_content_tag_decides_the_source(t *testing.T) {
	pal := sbgPalette()

	style_none := Style{}
	style_pal := Style{BgColor: sbgPal(20)}
	style_rgb := Style{BgColor: sbgRgb(7, 8, 9)}

	// Text cells use the style's bg color.
	for _, tag := range []PageCellContentTag{PageCellContentTagCodepoint, PageCellContentTagCodepointGrapheme} {
		cell := PageCell{
			content_tag: tag,
			content:     PageCellContent{codepoint: 'a'},
		}
		expectEqual(t, None[ColorRGB](), style_none.Bg(&cell, &pal))
		expectEqual(t, Some(pal[20]), style_pal.Bg(&cell, &pal))
		expectEqual(t,
			Some(ColorRGB{R: 7, G: 8, B: 9}),
			style_rgb.Bg(&cell, &pal),
		)
	}

	// A palette bg-only cell ignores the style entirely.
	{
		cell := PageCell{
			content_tag: PageCellContentTagBgColorPalette,
			content:     PageCellContent{color_palette: 33},
		}
		expectEqual(t, Some(pal[33]), style_none.Bg(&cell, &pal))
		expectEqual(t, Some(pal[33]), style_pal.Bg(&cell, &pal))
		expectEqual(t, Some(pal[33]), style_rgb.Bg(&cell, &pal))
	}

	// An rgb bg-only cell ignores the style entirely.
	{
		cell := PageCell{
			content_tag: PageCellContentTagBgColorRgb,
			content:     PageCellContent{color_rgb: PageCellRGB{R: 11, G: 12, B: 13}},
		}
		want := Some(ColorRGB{R: 11, G: 12, B: 13})
		expectEqual(t, want, style_none.Bg(&cell, &pal))
		expectEqual(t, want, style_pal.Bg(&cell, &pal))
		expectEqual(t, want, style_rgb.Bg(&cell, &pal))
	}
}

// Zig: test "term51: Style.bg: palette boundaries"
func TestStyle_term51_Style_bg_palette_boundaries(t *testing.T) {
	pal := sbgPalette()
	cell := PageCell{}

	s0 := Style{BgColor: sbgPal(0)}
	expectEqual(t, Some(pal[0]), s0.Bg(&cell, &pal))
	s255 := Style{BgColor: sbgPal(255)}
	expectEqual(t, Some(pal[255]), s255.Bg(&cell, &pal))

	c0 := PageCell{content_tag: PageCellContentTagBgColorPalette, content: PageCellContent{color_palette: 0}}
	expectEqual(t, Some(pal[0]), (Style{}).Bg(&c0, &pal))
	c255 := PageCell{content_tag: PageCellContentTagBgColorPalette, content: PageCellContent{color_palette: 255}}
	expectEqual(t, Some(pal[255]), (Style{}).Bg(&c255, &pal))
}

// Zig: test "term51: Style.bg: flags and other colors do not matter"
func TestStyle_term51_Style_bg_flags_and_other_colors_do_not_matter(t *testing.T) {
	pal := sbgPalette()
	cell := PageCell{}
	s := Style{
		FgColor:         sbgPal(1),
		underline_color: sbgPal(2),
		Flags:           StyleFlags{Inverse: true, Bold: true},
	}
	expectEqual(t, None[ColorRGB](), s.Bg(&cell, &pal))
}

// Zig: test "term51: Style.bgCell: no bg"
func TestStyle_term51_Style_bgCell_no_bg(t *testing.T) {
	s := Style{
		FgColor:         sbgPal(1),
		underline_color: sbgPal(2),
		Flags:           StyleFlags{Bold: true},
	}
	expect(t, !s.BgCell().Valid, "s.bgCell() == null")
	expect(t, !(Style{}).BgCell().Valid, "(Style{}).bgCell() == null")
}

// Zig: test "term51: Style.bgCell: palette bg"
func TestStyle_term51_Style_bgCell_palette_bg(t *testing.T) {
	for _, idx := range []uint8{0, 1, 128, 255} {
		s := Style{BgColor: sbgPal(idx)}
		cell := zigUnwrap(t, s.BgCell())
		expectEqual(t, PageCellContentTagBgColorPalette, cell.content_tag)
		expectEqual(t, idx, cell.content.color_palette)
		expectEqual(t, StyleID(0), cell.style_id)
		expectEqual(t, PageCellWideNarrow, cell.wide)
		expect(t, !cell.protected, "!cell.protected")
		expect(t, !cell.hyperlink, "!cell.hyperlink")
		expectEqual(t, PageCellSemanticContentOutput, cell.semantic_content)
	}
}

// Zig: test "term51: Style.bgCell: rgb bg"
func TestStyle_term51_Style_bgCell_rgb_bg(t *testing.T) {
	s := Style{BgColor: sbgRgb(10, 20, 30)}
	cell := zigUnwrap(t, s.BgCell())
	expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
	expectEqual(t, uint8(10), cell.content.color_rgb.R)
	expectEqual(t, uint8(20), cell.content.color_rgb.G)
	expectEqual(t, uint8(30), cell.content.color_rgb.B)
	expectEqual(t, StyleID(0), cell.style_id)
	expectEqual(t, PageCellWideNarrow, cell.wide)

	// Round trip through Style.bg.
	pal := sbgPalette()
	expectEqual(t,
		Some(ColorRGB{R: 10, G: 20, B: 30}),
		(Style{}).Bg(&cell, &pal),
	)
}
