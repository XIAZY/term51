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
		Cols:         cols,
		Rows:         rows,
		Reflow:       true,
		PromptRedraw: OSCParsersSemanticPromptRedrawFalse,
	}
}

// screenCharsetStateDefault is `Screen.CharsetState{}`: gl G0, gr G2.
func screenCharsetStateDefault() ScreenCharsetState {
	return ScreenCharsetState{Gr: CharsetsSlotsG2}
}

// screenCursorDefault is a Screen.Cursor with the Zig default style, block.
func screenCursorDefault() ScreenCursor {
	return ScreenCursor{CursorStyle: ScreenCursorStyleBlock}
}

// screenCursorCopyOptionsDefault is `.{}` for cursorCopy: hyperlink = true.
func screenCursorCopyOptionsDefault() ScreenCursorCopyOptions {
	return ScreenCursorCopyOptions{Hyperlink: true}
}

// screenSelectionStringDefault is `.{ .sel = sel }` for selectionString:
// trim = true, no map.
func screenSelectionStringDefault(sel Selection) ScreenSelectionString {
	return ScreenSelectionString{Sel: sel, Trim: true}
}

// screenSelectLineDefault is `.{ .pin = pin }` for selectLine: whitespace
// NUL, space and tab, semantic_prompt_boundary = true.
func screenSelectLineDefault(pin PageListPin) ScreenSelectLine {
	return ScreenSelectLine{
		Pin:                    pin,
		Whitespace:             Some([]uint32{0, ' ', '\t'}),
		SemanticPromptBoundary: true,
	}
}

// screenDumpStringOptionsDefault is `.{ .tl = tl }` for dumpString: no br,
// unwrap = true.
func screenDumpStringOptionsDefault(tl PageListPin) ScreenDumpStringOptions {
	return ScreenDumpStringOptions{Tl: tl, Unwrap: true}
}

// terminalOptionsDefault is `Terminal.Options{ .cols = cols, .rows = rows }`
// with the Zig defaults: max_scrollback 10_000, colors .default and
// default_modes .{}. Tests that pass other fields modify the result.
func terminalOptionsDefault(cols SizeCellCountInt, rows SizeCellCountInt) TerminalOptions {
	return TerminalOptions{
		Cols:          cols,
		Rows:          rows,
		MaxScrollback: 10_000,
		Colors:        TerminalColorsDefault,
		DefaultModes:  ModesModePackedInit(),
	}
}

// terminalFlagsDefault is Terminal.Flags `.{}`: focused = true.
func terminalFlagsDefault() TerminalFlags {
	return TerminalFlags{Focused: true}
}

// testPlatform sets platformGOOS (the platform whose behaviour the keyboard
// encoders follow: "linux" for the non-Darwin branch, "darwin" for macOS,
// "ios" for the Apple-but-not-macOS mix) for the test and restores it when
// the test ends. Tests that use it must not run in parallel. Every encoder
// test calls it first, so no test depends on the machine it runs on; a test
// that needs another platform for a part of its body calls it again.
func testPlatform(t *testing.T, goos string) {
	t.Helper()
	old := platformGOOS
	platformGOOS = goos
	t.Cleanup(func() { platformGOOS = old })
}
