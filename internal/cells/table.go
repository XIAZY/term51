package cells

import (
	"github.com/XIAZY/libghostty-vt-go/vt"
	"github.com/XIAZY/libghostty-vt-go/vt/color"
	"github.com/XIAZY/libghostty-vt-go/vt/page"
	"github.com/XIAZY/libghostty-vt-go/vt/sgr"
)

// RecordSize is the size of one cell's record in the table, in bytes:
//
//	0   uint16, little-endian: the character's slot in the sheet, 0 for none
//	2   uint16, little-endian: flags
//	4   the text's colour: red, green, blue, 255
//	8   the background's colour: red, green, blue, 255
//	12  the lines' colour: red, green, blue, 255
//
// The underline and the cursor shapes are drawn in the lines' colour, the
// strikethrough in the text's. Selection, reverse video and the block cursor
// are already in the colours.
const RecordSize = 16

// The flags of a record.
const (
	FlagUnderline       = 1
	FlagStrikethrough   = 2
	FlagCursorBar       = 4
	FlagCursorUnderline = 8
	FlagCursorHollow    = 16
)

// selectionTint is mixed into the background of selected text that was
// already in reverse video, so that the selection is never invisible.
var selectionTint = color.RGB{R: 0x30, G: 0x70, B: 0xC0}

// Table is the cell table: Cols*Rows records, row by row from the top left.
type Table struct {
	Cols, Rows int
	Data       []byte

	gen int
	rec []byte
}

// Update brings the table up to date with the render state and returns the
// range of bytes of Data that changed, [first, end); the two are equal when
// nothing did. After the grid's size changed, and the first time, that is
// the whole of a new Data.
func (t *Table) Update(rs *vt.RenderState, sheet *Sheet) (first, end int) {
	rows := len(rs.RowData)
	cols := 0
	if rows > 0 {
		cols = len(rs.RowData[0].Cells)
	}
	all := false
	if cols != t.Cols || rows != t.Rows || t.Data == nil {
		t.Cols, t.Rows = cols, rows
		t.Data = make([]byte, cols*rows*RecordSize)
		t.rec = make([]byte, cols*RecordSize)
		all = true
	}
	for {
		gen := sheet.Generation()
		if gen != t.gen {
			t.gen = gen
			all = true
		}
		first, end = t.fill(rs, sheet)
		if sheet.Generation() == gen {
			break
		}
		// The sheet filled up and started again while this pass asked for
		// characters, so slots written earlier in the pass are stale. A
		// screen's characters always fit in an empty sheet: the pass after
		// this one ends it.
	}
	if all {
		return 0, len(t.Data)
	}
	return first, end
}

// fill writes every row's records and returns the range that differs from
// what Data held.
func (t *Table) fill(rs *vt.RenderState, sheet *Sheet) (first, end int) {
	first, end = len(t.Data), 0
	n := t.Cols * RecordSize
	for y := range rs.RowData {
		t.row(rs, sheet, y)
		old := t.Data[y*n : y*n+n]
		for i := 0; i < n; i += RecordSize {
			if string(old[i:i+RecordSize]) != string(t.rec[i:i+RecordSize]) {
				first = min(first, y*n+i)
				end = max(end, y*n+i+RecordSize)
			}
		}
		copy(old, t.rec)
	}
	if end == 0 {
		return 0, 0
	}
	return first, end
}

// row writes row y's records into t.rec.
func (t *Table) row(rs *vt.RenderState, sheet *Sheet, y int) {
	r := &rs.RowData[y]
	defBg, defFg := rs.Colors.Background, rs.Colors.Foreground

	var runes []rune
	for x := range r.Cells {
		cell := &r.Cells[x]
		rec := t.rec[x*RecordSize : (x+1)*RecordSize]
		if cell.Raw.Wide() == page.CellWideSpacerTail && x > 0 {
			// The right half of the wide character before it: the same
			// look, and the slot after its slot.
			prev := t.rec[(x-1)*RecordSize : x*RecordSize]
			copy(rec, prev)
			if slot := uint16(prev[0]) | uint16(prev[1])<<8; slot != 0 {
				put16(rec, slot+1)
			}
			continue
		}
		var style vt.Style
		if cell.Raw.HasStyling() {
			style = cell.Style
		}
		fg := style.Fg(vt.StyleFg{Default: defFg, Palette: &rs.Colors.Palette})
		bg := defBg
		if c := style.Bg(&cell.Raw, &rs.Colors.Palette); c.Valid {
			bg = c.Value
		}
		// Selected text is shown inverted. Text already in reverse video
		// stays so when selected and gets a tint.
		selected := false
		if sel := r.Selection; sel.Valid && x >= int(sel.Value[0]) && x <= int(sel.Value[1]) {
			selected = true
		}
		if style.Flags.Inverse || selected {
			fg, bg = bg, fg
		}
		if style.Flags.Inverse && selected {
			bg = mix(bg, selectionTint)
		}
		if style.Flags.Faint {
			fg = mix(fg, bg)
		}
		line := fg
		var flags uint16
		// Curly, dotted, dashed and double underlines are drawn as one line.
		if style.Flags.Underline != sgr.AttributeUnderlineNone {
			flags |= FlagUnderline
			if c := style.UnderlineColor(&rs.Colors.Palette); c.Valid {
				line = c.Value
			}
		}
		if style.Flags.Strikethrough {
			flags |= FlagStrikethrough
		}
		var slot uint16
		if !style.Flags.Invisible && cell.Raw.HasText() {
			if cp := rune(cell.Raw.Codepoint()); cp != ' ' {
				runes = append(runes[:0], cp)
				for _, g := range cell.Grapheme {
					runes = append(runes, rune(g))
				}
				var face uint8
				if style.Flags.Bold {
					face |= 1
				}
				if style.Flags.Italic {
					face |= 2
				}
				slot = sheet.slot(runes, face, cell.Raw.Wide() == page.CellWideWide)
			}
		}
		put16(rec, slot)
		put16(rec[2:], flags)
		putRGB(rec[4:], fg)
		putRGB(rec[8:], bg)
		putRGB(rec[12:], line)
	}

	c := &rs.Cursor
	if !c.Visible || !c.Viewport.Valid || int(c.Viewport.Value.Y) != y {
		return
	}
	x := int(c.Viewport.Value.X)
	if c.Viewport.Value.WideTail && x > 0 {
		x--
	}
	n := 1
	if c.Cell.Wide() == page.CellWideWide {
		n = 2
	}
	cursor := defFg
	if rs.Colors.Cursor.Valid {
		cursor = rs.Colors.Cursor.Value
	}
	for i := 0; i < n && x+i < len(r.Cells); i++ {
		rec := t.rec[(x+i)*RecordSize : (x+i+1)*RecordSize]
		flags := uint16(rec[2]) | uint16(rec[3])<<8
		switch c.VisualStyle {
		case vt.CursorStyleBar:
			if i == 0 {
				flags |= FlagCursorBar
			}
			putRGB(rec[12:], cursor)
		case vt.CursorStyleUnderline:
			// It takes the place of the text's own underline.
			flags = flags&^FlagUnderline | FlagCursorUnderline
			putRGB(rec[12:], cursor)
		case vt.CursorStyleBlockHollow:
			flags |= FlagCursorHollow
			putRGB(rec[12:], cursor)
		default:
			// A block: the cell in the cursor's colour, its character in the
			// background's.
			putRGB(rec[4:], defBg)
			putRGB(rec[8:], cursor)
			putRGB(rec[12:], defBg)
		}
		put16(rec[2:], flags)
	}
}

func put16(b []byte, v uint16) { b[0], b[1] = byte(v), byte(v>>8) }

func putRGB(b []byte, c color.RGB) { b[0], b[1], b[2], b[3] = c.R, c.G, c.B, 255 }

// mix is the colour half way between a and b.
func mix(a, b color.RGB) color.RGB {
	return color.RGB{
		R: uint8((uint(a.R) + uint(b.R)) / 2),
		G: uint8((uint(a.G) + uint(b.G)) / 2),
		B: uint8((uint(a.B) + uint(b.B)) / 2),
	}
}
