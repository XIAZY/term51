// Package textgrid turns the terminal's render state into rich text, one
// string per row, for a front end that shows the terminal as a column of
// Cascades text labels (TextFormat.Html) in a monospace font instead of
// drawing pixels.
//
// A row is a sequence of <span> elements, one per run of cells with the same
// look. Cascades collapses ordinary spaces, so every space is a no-break
// space, and every row is padded to the terminal's width, which keeps the
// labels the same width and the columns aligned as far as the font allows.
package textgrid

import (
	"strconv"
	"strings"

	"github.com/XIAZY/libghostty-vt-go/vt"
	"github.com/XIAZY/libghostty-vt-go/vt/page"
	"github.com/XIAZY/libghostty-vt-go/vt/sgr"
)

// Row is one row's rich text.
type Row struct {
	Y    int
	HTML string
}

// Grid remembers what each row last looked like, so that only rows whose
// text or look changed are sent to the labels again.
type Grid struct {
	rows []string
}

// Update returns the rows that differ from the last Update (all of them the
// first time, and after the number of rows changed).
func (g *Grid) Update(rs *vt.RenderState) []Row {
	n := len(rs.RowData)
	if len(g.rows) != n {
		g.rows = make([]string, n)
		for i := range g.rows {
			g.rows[i] = "\x00" // differs from any row
		}
	}
	var out []Row
	var sb strings.Builder
	for y := range rs.RowData {
		sb.Reset()
		row(&sb, rs, y)
		if html := sb.String(); html != g.rows[y] {
			g.rows[y] = html
			out = append(out, Row{Y: y, HTML: html})
		}
	}
	return out
}

// look is how a cell is shown.
type look struct {
	fg, bg                          uint32
	bold, italic, underline, strike bool
}

func rgb(r, g, b uint8) uint32 { return uint32(r)<<16 | uint32(g)<<8 | uint32(b) }

func row(sb *strings.Builder, rs *vt.RenderState, y int) {
	r := &rs.RowData[y]
	defBg := rgb(rs.Colors.Background.R, rs.Colors.Background.G, rs.Colors.Background.B)
	cursorX := -1
	if c := &rs.Cursor; c.Visible && c.Viewport.Valid && int(c.Viewport.Value.Y) == y {
		cursorX = int(c.Viewport.Value.X)
		if c.Viewport.Value.WideTail && cursorX > 0 {
			cursorX--
		}
	}

	var cur look
	open := false
	var text strings.Builder
	flush := func() {
		if !open {
			return
		}
		sb.WriteString(`<span style="color:#`)
		hex6(sb, cur.fg)
		if cur.bg != defBg {
			sb.WriteString(`;background-color:#`)
			hex6(sb, cur.bg)
		}
		if cur.bold {
			sb.WriteString(`;font-weight:bold`)
		}
		if cur.italic {
			sb.WriteString(`;font-style:italic`)
		}
		if cur.underline && cur.strike {
			sb.WriteString(`;text-decoration:underline line-through`)
		} else if cur.underline {
			sb.WriteString(`;text-decoration:underline`)
		} else if cur.strike {
			sb.WriteString(`;text-decoration:line-through`)
		}
		sb.WriteString(`">`)
		sb.WriteString(text.String())
		sb.WriteString(`</span>`)
		text.Reset()
		open = false
	}

	for x := range r.Cells {
		cell := &r.Cells[x]
		wide := cell.Raw.Wide()
		if wide == page.CellWideSpacerTail {
			continue // the wide character before it fills this cell
		}
		var style vt.Style
		if cell.Raw.HasStyling() {
			style = cell.Style
		}
		fgc := style.Fg(vt.StyleFg{Default: rs.Colors.Foreground, Palette: &rs.Colors.Palette})
		l := look{
			fg: rgb(fgc.R, fgc.G, fgc.B), bg: defBg,
			bold: style.Flags.Bold, italic: style.Flags.Italic,
			underline: style.Flags.Underline != sgr.AttributeUnderlineNone,
			strike:    style.Flags.Strikethrough,
		}
		if c := style.Bg(&cell.Raw, &rs.Colors.Palette); c.Valid {
			l.bg = rgb(c.Value.R, c.Value.G, c.Value.B)
		}
		inverse := style.Flags.Inverse
		if sel := r.Selection; sel.Valid && x >= int(sel.Value[0]) && x <= int(sel.Value[1]) {
			inverse = !inverse
		}
		if x == cursorX {
			inverse = !inverse // a block cursor: the cell inverted
		}
		if inverse {
			l.fg, l.bg = l.bg, l.fg
		}
		if style.Flags.Faint {
			l.fg = mix(l.fg, l.bg)
		}
		if l != cur || !open {
			flush()
			cur, open = l, true
		}
		if style.Flags.Invisible || !cell.Raw.HasText() {
			text.WriteString(" ")
			if wide == page.CellWideWide {
				text.WriteString(" ")
			}
			continue
		}
		escape(&text, rune(cell.Raw.Codepoint()))
		for _, cp := range cell.Grapheme {
			escape(&text, rune(cp))
		}
	}
	flush()
}

// escape writes one character of text: markup characters as entities, a
// space as a no-break space, and control characters (which a cell should not
// hold) as the replacement character.
func escape(sb *strings.Builder, r rune) {
	switch {
	case r == ' ':
		sb.WriteString(" ")
	case r == '<':
		sb.WriteString("&lt;")
	case r == '>':
		sb.WriteString("&gt;")
	case r == '&':
		sb.WriteString("&amp;")
	case r == '"':
		sb.WriteString("&quot;")
	case r < 0x20 || r == 0x7F:
		sb.WriteString("�")
	default:
		sb.WriteRune(r)
	}
}

func hex6(sb *strings.Builder, c uint32) {
	s := strconv.FormatUint(uint64(c), 16)
	sb.WriteString("000000"[len(s):])
	sb.WriteString(s)
}

// mix is the colour half way between a and b.
func mix(a, b uint32) uint32 {
	return (a>>1)&0x7F7F7F + (b>>1)&0x7F7F7F + (a & b & 0x010101)
}
