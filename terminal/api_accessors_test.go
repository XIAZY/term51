package terminal

import "testing"

// Go-only tests of the accessors (api_accessors.go): each returns the field
// of the same name.

func TestApiAccessors_cell_and_row(t *testing.T) {
	// "hello " then bold red "world", a new line, "line 2 " and two wide
	// characters.
	term := apiWriterTerminal(t)
	defer term.Deinit()
	pages := &term.Screens.Active.Pages

	at := func(x, y SizeCellCountInt) PageListCell {
		c := pages.GetCell(Point{Tag: PointTagActive, Active: PointCoordinate{X: x, Y: uint32(y)}})
		if !c.Valid {
			t.Fatalf("no cell at %d,%d", x, y)
		}
		return c.Value
	}

	plain := at(0, 0)
	if plain.Cell.Codepoint() != 'h' || plain.Cell.StyleID() != StyleDefaultID || plain.Cell.Wide() != PageCellWideNarrow {
		t.Errorf("plain cell: %+v", *plain.Cell)
	}
	bold := at(6, 0)
	if bold.Cell.Codepoint() != 'w' || bold.Cell.StyleID() == StyleDefaultID {
		t.Errorf("bold cell: %+v", *bold.Cell)
	}
	if !plain.Row.Styled() || plain.Row.Wrap() || plain.Row.WrapContinuation() {
		t.Errorf("row 0: %+v", *plain.Row)
	}
	wide := at(7, 1)
	if wide.Cell.Wide() != PageCellWideWide || at(8, 1).Cell.Wide() != PageCellWideSpacerTail {
		t.Errorf("wide cell: %+v, then %+v", *wide.Cell, *at(8, 1).Cell)
	}

	// Every accessor against its field, on every cell of the two rows.
	for y := SizeCellCountInt(0); y < 2; y++ {
		for x := SizeCellCountInt(0); x < 20; x++ {
			c, r := at(x, y).Cell, at(x, y).Row
			if c.ContentTag() != c.ContentTag() || c.StyleID() != c.StyleID() || c.Wide() != c.Wide() ||
				c.Protected() != c.Protected() || c.Hyperlink() != c.Hyperlink() ||
				c.SemanticContent() != c.SemanticContent() ||
				c.ColorPalette() != c.content().colorPalette() || c.ColorRGB() != c.content().colorRGB() {
				t.Errorf("cell %d,%d: an accessor differs from its field", x, y)
			}
			if r.Wrap() != r.wrap || r.WrapContinuation() != r.wrap_continuation || r.Grapheme() != r.grapheme ||
				r.Styled() != r.styled || r.Hyperlink() != r.hyperlink || r.SemanticPrompt() != r.semantic_prompt ||
				r.KittyVirtualPlaceholder() != r.kitty_virtual_placeholder || r.Dirty() != r.dirty {
				t.Errorf("row %d: an accessor differs from its field", y)
			}
		}
	}
}

// ModePacked.Get and Set read and write the same field as ModeState's.
func TestApiAccessors_ModePacked(t *testing.T) {
	for _, mode := range ModesModeValues() {
		p := ModesModePackedInit()
		st := ModesModeState{Values: p, Default: p}
		if p.Get(mode) != st.Get(mode) {
			t.Errorf("%v: default differs", mode)
		}
		want := !p.Get(mode)
		p.Set(mode, want)
		st.Set(mode, want)
		if p.Get(mode) != want || p != st.Values {
			t.Errorf("%v: set to %v, got %v", mode, want, p.Get(mode))
		}
	}
}
