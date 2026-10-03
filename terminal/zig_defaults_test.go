package terminal

// Shared helpers for translated tests that depend on Zig field defaults a Go
// literal does not give (see the wireframe notes in Screen.go and PageList.go).
// Every translated test that writes `.{}` or `.default` for one of these
// structs starts from the helper here, so a missing field is one place to fix.
// Tests that use testStdCapacity must not run in parallel.

import "testing"

// testStdCapacity sets PageStdCapacity.grapheme_bytes to 512 for the test,
// the value Zig has under builtin.is_test (page.std_capacity), and restores
// it when the test ends. The recorded corpus tests do not call it.
func testStdCapacity(t *testing.T) {
	t.Helper()
	old := PageStdCapacity.grapheme_bytes
	PageStdCapacity.grapheme_bytes = 512
	t.Cleanup(func() { PageStdCapacity.grapheme_bytes = old })
}

// screenResizeDefault is `Screen.Resize{ .cols = cols, .rows = rows }` with the
// Zig defaults reflow = true and prompt_redraw = .false.
func screenResizeDefault(cols SizeCellCountInt, rows SizeCellCountInt) ScreenResize {
	return ScreenResize{
		cols:          cols,
		rows:          rows,
		reflow:        true,
		prompt_redraw: OSCParsersSemanticPromptRedrawFalse,
	}
}

// screenCharsetStateDefault is `Screen.CharsetState{}`: gl G0, gr G2.
func screenCharsetStateDefault() ScreenCharsetState {
	return ScreenCharsetState{gr: CharsetsSlotsG2}
}

// screenCursorDefault is a Screen.Cursor with the Zig default style, block.
func screenCursorDefault() ScreenCursor {
	return ScreenCursor{cursor_style: ScreenCursorStyleBlock}
}

// screenCursorCopyOptionsDefault is `.{}` for cursorCopy: hyperlink = true.
func screenCursorCopyOptionsDefault() ScreenCursorCopyOptions {
	return ScreenCursorCopyOptions{hyperlink: true}
}

// screenSelectionStringDefault is `.{ .sel = sel }` for selectionString:
// trim = true, no map.
func screenSelectionStringDefault(sel Selection) ScreenSelectionString {
	return ScreenSelectionString{sel: sel, trim: true}
}

// screenSelectLineDefault is `.{ .pin = pin }` for selectLine: whitespace
// NUL, space and tab, semantic_prompt_boundary = true.
func screenSelectLineDefault(pin PageListPin) ScreenSelectLine {
	return ScreenSelectLine{
		pin:                      pin,
		whitespace:               Some([]uint32{0, ' ', '\t'}),
		semantic_prompt_boundary: true,
	}
}

// screenDumpStringOptionsDefault is `.{ .tl = tl }` for dumpString: no br,
// unwrap = true.
func screenDumpStringOptionsDefault(tl PageListPin) ScreenDumpStringOptions {
	return ScreenDumpStringOptions{tl: tl, unwrap: true}
}
