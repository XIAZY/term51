package terminal

// The state dump of the corpus replayer: the Go side of the recorder's
// harness/src/dump.zig. The text it writes must equal the Zig text byte for
// byte. Every function names the Zig function it mirrors; the control flow
// follows the Zig.
//
// Differences from the Zig, all forced by the language:
//   - the writer is a *bytes.Buffer (writes never fail, so there is no `try`);
//   - Zig's `dumpAny` is comptime reflection over struct fields; the Go one is
//     run-time reflection (reflect), which cannot read unexported fields with
//     Interface(), so every field is reached through dumpField (one test-only
//     use of unsafe);
//   - Go has no enum type: an integer type that has a String method is an
//     enum (every wireframe enum has one) and String() == "" stands for the
//     unnamed value that Zig's std.enums.tagName reports as null;
//   - Go has no tagged union: a struct whose first field is `Tag` (an integer
//     type) is one, and its payload is the field named by the CamelCase of
//     the tag (`cursor_up` is `CursorUp`, `8_bg` is `X8Bg`, see zigFieldName). A tag whose Zig
//     payload is void has no field. Which tags are void is not visible to
//     reflection, so every union type that dumpAny reaches is listed in
//     dumpVoidTags with its void tags (checked against the Zig source by
//     TestCorpusReplaySelfUnionsAgainstZig); dumpUnion panics, with the type
//     and the tag, for a union that is not listed and for a non-void tag
//     without a payload field, instead of leaving the line out;
//   - Go field names that differ from the Zig ones are mapped by zigName;
//   - Zig's ArrayList special case of dumpAny does not exist in Go (a slice is
//     already the list of its items); its SegmentedList special case is the
//     Go type OSCParsersColorList (dumpIsSegmentedList), a struct that holds
//     the items as a slice;
//   - an enum whose type has no String method, or whose String is taken by
//     another function (OSCTerminator: String is Zig's `string`, the tag name
//     is TagName), is named through TagName when the type has it; an integer
//     type that is neither is a wireframe gap and dumpAny panics with the type
//     and the field instead of printing the number (see dumpTagName);
//   - the VTGOLD_RAW debugging aid (`raw_cells`) is not ported: the recorder
//     never uses it for the corpus.

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unsafe"
)

// Mirrors harness/src/dump.zig:24 dumpTerminal
func dumpTerminal(w *bytes.Buffer, t *Terminal, label string) {
	fmt.Fprintf(w, "== dump %s\n", label)

	// Bytes the terminal wants written back to the pty (answers to
	// queries). The read-only stream handler never produces any, so this is
	// always empty.
	w.WriteString("replies=\n")

	dumpAny(w, "term.cols", t.Cols)
	dumpAny(w, "term.rows", t.Rows)
	dumpAny(w, "term.width_px", t.WidthPx)
	dumpAny(w, "term.height_px", t.HeightPx)
	dumpAny(w, "term.status_display", t.StatusDisplay)
	dumpAny(w, "term.scrolling_region", t.ScrollingRegion)
	dumpAny(w, "term.previous_char", t.PreviousChar)
	dumpAny(w, "term.mouse_shape", t.MouseShape)
	dumpAny(w, "term.flags", t.Flags)
	dumpAny(w, "term.modes.values", t.Modes.Values)
	dumpAny(w, "term.modes.saved", t.Modes.Saved)
	dumpAny(w, "term.modes.default", t.Modes.Default)
	dumpAny(w, "term.colors.background", t.Colors.Background)
	dumpAny(w, "term.colors.foreground", t.Colors.Foreground)
	dumpAny(w, "term.colors.cursor", t.Colors.Cursor)

	// Palette: only entries that differ from the original, plus the mask.
	for i := 0; i < 256; i++ {
		cur := t.Colors.Palette.Current[i]
		orig := t.Colors.Palette.Original[i]
		masked := t.Colors.Palette.Mask.isSet(i)
		if masked || cur.R != orig.R || cur.G != orig.G || cur.B != orig.B {
			fmt.Fprintf(w, "term.colors.palette[%d]=%02x%02x%02x mask=%d\n", i, cur.R, cur.G, cur.B, boolInt(masked))
		}
	}

	w.WriteString("term.pwd=")
	writeEscaped(w, t.Pwd)
	w.WriteString("\n")

	w.WriteString("term.tabstops=")
	{
		first := true
		for x := uint(0); x < uint(t.Cols); x++ {
			if !t.Tabstops.Get(x) {
				continue
			}
			if !first {
				w.WriteString(",")
			}
			first = false
			fmt.Fprintf(w, "%d", x)
		}
	}
	w.WriteString("\n")

	fmt.Fprintf(w, "term.screens.active_key=%s\n", t.Screens.ActiveKey.String())

	for _, key := range []ScreenSetKey{ScreenSetKeyPrimary, ScreenSetKeyAlternate} {
		if screen := t.Screens.Get(key); screen != nil {
			fmt.Fprintf(w, "[screen %s]\n", key.String())
			dumpScreen(w, screen)
		} else {
			fmt.Fprintf(w, "[screen %s] absent\n", key.String())
		}
	}
}

// boolInt is Zig's @intFromBool.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Mirrors harness/src/dump.zig:89 dumpScreen
//
// The Zig function is one body; here the pages block and the row loop are
// separate functions (dumpScreenPages, dumpScreenRows), called in the same
// order.
func dumpScreen(w *bytes.Buffer, s *Screen) {
	dumpAny(w, "screen.no_scrollback", s.NoScrollback)
	dumpAny(w, "screen.cursor.x", s.Cursor.X)
	dumpAny(w, "screen.cursor.y", s.Cursor.Y)
	dumpAny(w, "screen.cursor.cursor_style", s.Cursor.CursorStyle)
	dumpAny(w, "screen.cursor.pending_wrap", s.Cursor.PendingWrap)
	dumpAny(w, "screen.cursor.protected", s.Cursor.Protected)
	w.WriteString("screen.cursor.style=")
	writeStyle(w, s.Cursor.Style)
	w.WriteString("\n")
	dumpAny(w, "screen.cursor.semantic_content", s.Cursor.SemanticContent)
	dumpAny(w, "screen.cursor.semantic_content_clear_eol", s.Cursor.SemanticContentClearEol)
	if link := s.Cursor.Hyperlink; link != nil {
		w.WriteString("screen.cursor.hyperlink=")
		switch link.ID.Tag {
		case HyperlinkHyperlinkIdExplicit:
			w.WriteString("explicit:")
			writeEscaped(w, link.ID.Explicit)
		case HyperlinkHyperlinkIdImplicit:
			fmt.Fprintf(w, "implicit:%d", link.ID.Implicit)
		}
		w.WriteString(" uri=")
		writeEscaped(w, link.URI)
		w.WriteString("\n")
	} else {
		w.WriteString("screen.cursor.hyperlink=null\n")
	}
	dumpAny(w, "ident.screen.cursor.style_id", s.Cursor.StyleID)
	dumpAny(w, "ident.screen.cursor.hyperlink_id", s.Cursor.HyperlinkID)
	dumpAny(w, "ident.screen.cursor.hyperlink_implicit_id", s.Cursor.HyperlinkImplicitID)

	if s.SavedCursor.Valid {
		sc := s.SavedCursor.Value
		dumpAny(w, "screen.saved_cursor.x", sc.X)
		dumpAny(w, "screen.saved_cursor.y", sc.Y)
		w.WriteString("screen.saved_cursor.style=")
		writeStyle(w, sc.Style)
		w.WriteString("\n")
		dumpAny(w, "screen.saved_cursor.protected", sc.Protected)
		dumpAny(w, "screen.saved_cursor.pending_wrap", sc.PendingWrap)
		dumpAny(w, "screen.saved_cursor.origin", sc.Origin)
		dumpAny(w, "screen.saved_cursor.charset", sc.Charset)
	} else {
		w.WriteString("screen.saved_cursor=null\n")
	}

	dumpAny(w, "screen.charset", s.Charset)
	dumpAny(w, "screen.protected_mode", s.ProtectedMode)
	dumpAny(w, "screen.kitty_keyboard.idx", s.KittyKeyboard.idx)
	for i := range s.KittyKeyboard.flags {
		dumpAny(w, fmt.Sprintf("screen.kitty_keyboard.flags[%d]", i), s.KittyKeyboard.flags[i])
	}
	dumpAny(w, "screen.semantic_prompt", s.SemanticPrompt)
	dumpAny(w, "screen.dirty", s.Dirty)
	selection := "set"
	if !s.Selection.Valid {
		selection = "null"
	}
	fmt.Fprintf(w, "screen.selection=%s\n", selection)

	pages := &s.Pages
	dumpScreenPages(w, pages)
	dumpScreenRows(w, pages)
}

// Mirrors harness/src/dump.zig:148 dumpScreen, the "// Pages" block (lines
// 148-192).
func dumpScreenPages(w *bytes.Buffer, pages *PageList) {
	dumpAny(w, "pages.cols", pages.cols)
	dumpAny(w, "pages.rows", pages.rows)
	fmt.Fprintf(w, "pages.viewport=%s\n", pages.viewport.Tag.String())
	{
		total := uint(0)
		for n := pages.pages.first; n != nil; n = n.next {
			total += uint(n.data.size.rows)
		}
		fmt.Fprintf(w, "pages.total_rows=%d\n", total)
		// The page list's running byte count of its pages. The scrollback
		// limit is measured against it, and upstream's accounting drifts, so
		// a port must reproduce it exactly (these three lines are not in
		// corpusSkipPrefixes).
		fmt.Fprintf(w, "pages.page_size=%d\n", pages.page_size)
		fmt.Fprintf(w, "pages.explicit_max_size=%d\n", pages.explicit_max_size)
		fmt.Fprintf(w, "pages.min_max_size=%d\n", pages.min_max_size)
	}
	{
		tl := pages.GetTopLeft(PointTagActive)
		pt := dumpUnwrap(pages.PointFromPin(PointTagScreen, tl))
		fmt.Fprintf(w, "pages.active_top=%d\n", pt.Screen.Y)
	}
	{
		tl := pages.GetTopLeft(PointTagViewport)
		pt := dumpUnwrap(pages.PointFromPin(PointTagScreen, tl))
		fmt.Fprintf(w, "pages.viewport_top=%d\n", pt.Screen.Y)
	}
	{
		w.WriteString("layout.pages.nodes=")
		first := true
		for n := pages.pages.first; n != nil; n = n.next {
			if !first {
				w.WriteString(",")
			}
			first = false
			fmt.Fprintf(w, "%dx%d/%dx%d",
				n.data.size.cols,
				n.data.size.rows,
				n.data.capacity.cols,
				n.data.capacity.rows,
			)
		}
		w.WriteString("\n")
	}
}

// dumpUnwrap is Zig's `o.?` outside a test: a null optional is a bug of the
// port, reported by a panic that the replayer turns into a mismatch.
func dumpUnwrap[T any](o Optional[T]) T {
	if !o.Valid {
		panic("attempt to use null value")
	}
	return o.Value
}

// Mirrors harness/src/dump.zig:194 dumpScreen, the rows block (lines
// 194-278).
func dumpScreenRows(w *bytes.Buffer, pages *PageList) {
	// Every row is listed; cells are listed only when some field differs
	// from an empty cell.
	y := 0
	it := pages.RowIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	for {
		next := it.Next()
		if !next.Valid {
			break
		}
		pin := next.Value
		rac := pin.RowAndCell()
		row := rac.row
		page := &pin.Node.data
		dumpScreenRow(w, y, row, page)
		y++
	}
}

// Mirrors harness/src/dump.zig:202 dumpScreen, the body of the row loop
// (lines 199-277).
func dumpScreenRow(w *bytes.Buffer, y int, row *PageRow, page *Page) {
	fmt.Fprintf(w, "row %d:", y)
	if row.wrap {
		w.WriteString(" wrap")
	}
	if row.wrap_continuation {
		w.WriteString(" wrap_continuation")
	}
	if row.grapheme {
		w.WriteString(" grapheme")
	}
	if row.styled {
		w.WriteString(" styled")
	}
	if row.hyperlink {
		w.WriteString(" hyperlink")
	}
	if row.semantic_prompt != PageRowSemanticPromptNone {
		fmt.Fprintf(w, " semantic_prompt=%s", row.semantic_prompt.String())
	}
	if row.kitty_virtual_placeholder {
		w.WriteString(" kitty_virtual_placeholder")
	}
	if row.dirty {
		w.WriteString(" dirty")
	}
	w.WriteString("\n")

	cells := page.GetCells(row)
	runStart := -1 // Zig: ?usize
	for x := range cells {
		cell := &cells[x]
		// Runs of plain printable-ASCII cells (no style, width, link or
		// other attribute) are written as one line: t <x>: "<text>".
		// The text runs to the last quote on the line; it is not escaped.
		if isPlainAscii(cell) {
			if runStart < 0 {
				runStart = x
				fmt.Fprintf(w, " t %d: \"", x)
			}
			w.WriteByte(byte(cell.content.codepoint))
			continue
		}
		if runStart >= 0 {
			runStart = -1
			w.WriteString("\"\n")
		}
		if cellIsDefault(cell) {
			continue
		}
		dumpScreenCell(w, x, cell, page)
	}
	if runStart >= 0 {
		w.WriteString("\"\n")
	}
}

// Mirrors harness/src/dump.zig:232 dumpScreen, the " c <x>:" line of one
// cell (lines 232-275). The page is only read for the grapheme, style and
// hyperlink lookups.
func dumpScreenCell(w *bytes.Buffer, x int, cell *PageCell, page *Page) {
	fmt.Fprintf(w, " c %d:", x)
	switch cell.content_tag {
	case PageCellContentTagCodepoint:
		fmt.Fprintf(w, " cp=%x", cell.content.codepoint)
	case PageCellContentTagCodepointGrapheme:
		fmt.Fprintf(w, " cp=%x", cell.content.codepoint)
		if cps := page.LookupGrapheme(cell); cps.Valid {
			w.WriteString(" g=")
			for i, cp := range cps.Value {
				if i > 0 {
					w.WriteString("+")
				}
				fmt.Fprintf(w, "%x", cp)
			}
		} else {
			w.WriteString(" g=MISSING")
		}
	case PageCellContentTagBgColorPalette:
		fmt.Fprintf(w, " bgpal=%d", cell.content.color_palette)
	case PageCellContentTagBgColorRgb:
		fmt.Fprintf(w, " bgrgb=%02x%02x%02x",
			cell.content.color_rgb.R,
			cell.content.color_rgb.G,
			cell.content.color_rgb.B,
		)
	}
	if cell.wide != PageCellWideNarrow {
		fmt.Fprintf(w, " wide=%s", cell.wide.String())
	}
	if cell.protected {
		w.WriteString(" protected")
	}
	if cell.semantic_content != PageCellSemanticContentOutput {
		fmt.Fprintf(w, " sem=%s", cell.semantic_content.String())
	}
	if cell.style_id != 0 {
		w.WriteString(" style=")
		writeStyle(w, *page.styles.Get(cell.style_id))
	}
	if cell.hyperlink {
		if id := page.LookupHyperlink(cell); id.Valid {
			entry := page.hyperlink_set.Get(id.Value)
			w.WriteString(" link=")
			switch entry.id.Tag {
			case HyperlinkPageEntryIdExplicit:
				w.WriteString("explicit:")
				writeEscaped(w, entry.id.Explicit)
			case HyperlinkPageEntryIdImplicit:
				fmt.Fprintf(w, "implicit:%d", entry.id.Implicit)
			}
			w.WriteString(" uri=")
			writeEscaped(w, entry.uri)
		} else {
			w.WriteString(" link=MISSING")
		}
	}
	w.WriteString("\n")
}

// Mirrors harness/src/dump.zig:286 cellIsDefault
//
// True if every field of the cell that carries meaning has its default
// value. It does not look at the raw bits (see the Zig comment).
func cellIsDefault(cell *PageCell) bool {
	return cell.content_tag == PageCellContentTagCodepoint &&
		cell.content.codepoint == 0 &&
		cellAttrsDefault(cell)
}

// Mirrors harness/src/dump.zig:292 cellAttrsDefault
func cellAttrsDefault(cell *PageCell) bool {
	return cell.style_id == 0 &&
		cell.wide == PageCellWideNarrow &&
		!cell.protected &&
		!cell.hyperlink &&
		cell.semantic_content == PageCellSemanticContentOutput
}

// Mirrors harness/src/dump.zig:300 isPlainAscii
func isPlainAscii(cell *PageCell) bool {
	if cell.content_tag != PageCellContentTagCodepoint {
		return false
	}
	cp := cell.content.codepoint
	if cp < 0x20 || cp > 0x7e {
		return false
	}
	return cellAttrsDefault(cell)
}

// Mirrors harness/src/dump.zig:307 writeStyle
func writeStyle(w *bytes.Buffer, s Style) {
	w.WriteString("{fg=")
	writeColor(w, s.FgColor)
	w.WriteString(",bg=")
	writeColor(w, s.BgColor)
	w.WriteString(",ul=")
	writeColor(w, s.underline_color)
	w.WriteString(",flags=")
	any_ := false
	for _, flag := range []struct {
		name string
		set  bool
	}{
		{"bold", s.Flags.Bold},
		{"italic", s.Flags.Italic},
		{"faint", s.Flags.Faint},
		{"blink", s.Flags.Blink},
		{"inverse", s.Flags.Inverse},
		{"invisible", s.Flags.Invisible},
		{"strikethrough", s.Flags.Strikethrough},
		{"overline", s.Flags.Overline},
	} {
		if flag.set {
			if any_ {
				w.WriteString("|")
			}
			any_ = true
			w.WriteString(flag.name)
		}
	}
	if s.Flags.Underline != SGRAttributeUnderlineNone {
		if any_ {
			w.WriteString("|")
		}
		any_ = true
		fmt.Fprintf(w, "underline:%s", s.Flags.Underline.String())
	}
	if !any_ {
		w.WriteString("none")
	}
	w.WriteString("}")
}

// Mirrors harness/src/dump.zig:332 writeColor
func writeColor(w *bytes.Buffer, c StyleColor) {
	switch c.Tag {
	case StyleColorNone:
		w.WriteString("none")
	case StyleColorPalette:
		fmt.Fprintf(w, "pal:%d", c.Palette)
	case StyleColorRgb:
		fmt.Fprintf(w, "rgb:%02x%02x%02x", c.Rgb.R, c.Rgb.G, c.Rgb.B)
	}
}

// Mirrors harness/src/dump.zig:341 writeEscaped
//
// Printable ASCII verbatim, everything else (and backslash) as \xNN. The
// escaping itself is dumpEscaped (corpus_test.go).
func writeEscaped(w *bytes.Buffer, b []uint8) {
	w.WriteString(dumpEscaped(b))
}

// Mirrors harness/src/dump.zig:353 dumpAny
//
// Reflectively dump a value as "prefix=value" lines. Structs and tagged
// unions recurse with dotted names; padding fields are skipped. See the file
// header for the Go counterparts of the Zig type kinds.
func dumpAny(w *bytes.Buffer, prefix string, v any) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		panic("dumpAny: nil value at " + prefix)
	}
	// An addressable copy, so that dumpField can reach unexported fields.
	addr := reflect.New(rv.Type()).Elem()
	addr.Set(rv)
	dumpValue(w, prefix, addr)
}

var dumpStringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()

// dumpTagNamerType is the interface of an enum whose String method is not the
// tag name (OSCTerminator: String is Zig's `string`, TagName is @tagName).
var dumpTagNamerType = reflect.TypeOf((*interface{ TagName() string })(nil)).Elem()

// dumpHasName is true for a type that can name its values: a String or a
// TagName method.
func dumpHasName(t reflect.Type) bool {
	return t.Implements(dumpTagNamerType) || t.Implements(dumpStringerType)
}

// dumpIsInteger is true for the Go integer kinds.
func dumpIsInteger(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	}
	return false
}

// dumpIsEnum is the Go test for Zig's `.@"enum"`: an integer type that can
// name its values (dumpHasName).
func dumpIsEnum(t reflect.Type) bool {
	_, listed := dumpEnumNames[t]
	return dumpIsInteger(t) && (dumpHasName(t) || listed)
}

// dumpUnnamedEnum panics for a defined integer type that is not an enum for
// dumpIsEnum: see dumpIsInt. t is the type of the value at prefix.
func dumpUnnamedEnum(t reflect.Type, prefix string) {
	if dumpIsInteger(t) && !dumpIsEnum(t) && t.PkgPath() != "" {
		panic("dumpAny: the integer type " + t.String() + " has no String or TagName method, " +
			"Zig prints the name of its enum value (at " + prefix + ")")
	}
}

// dumpIsInt is Zig's `.int` (and not an enum). Every Zig integer is a plain
// Go integer type; a defined integer type without a name method is a Zig
// enum whose wireframe lacks String, and dumpAny would print the number
// where Zig prints the name: dumpValue panics for it (dumpUnnamedEnum).
func dumpIsInt(t reflect.Type) bool {
	return dumpIsInteger(t) && !dumpIsEnum(t)
}

// dumpInteger is the integer value of v as text.
func dumpInteger(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	}
	return strconv.FormatUint(v.Uint(), 10)
}

// dumpEnumNames replaces the name methods of the listed enum types, by value.
// It is nil in the replayer: the names come from the port. Only the self tests
// of the dumper (corpus_replay_osc_self_test.go) set it, to dump values of
// types whose String is still the wireframe's panic or is missing.
var dumpEnumNames map[reflect.Type]map[int64]string

// dumpTagName is std.enums.tagName: the name of an enum value, and false for
// a value that has none. The name is TagName() when the type has it, else
// String(). A type with neither is a union tag or an enum that the wireframe
// leaves without a name: panic, naming the type.
func dumpTagName(v reflect.Value) (string, bool) {
	if names, ok := dumpEnumNames[v.Type()]; ok {
		name, ok := names[dumpInt64(v)]
		return name, ok
	}
	var name string
	switch e := v.Interface().(type) {
	case interface{ TagName() string }:
		name = e.TagName()
	case fmt.Stringer:
		name = e.String()
	default:
		panic("dumpAny: " + v.Type().String() + " has no String or TagName method, its value names cannot be dumped")
	}
	return name, name != ""
}

// dumpInt64 is the integer value of v.
func dumpInt64(v reflect.Value) int64 {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	}
	return int64(v.Uint())
}

// dumpField is the field i of the struct v, readable even when the field is
// unexported. v must be addressable.
func dumpField(v reflect.Value, i int) reflect.Value {
	f := v.Field(i)
	if f.CanInterface() {
		return f
	}
	return reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
}

// zigName is the Zig field name of a Go struct field. An unexported field
// keeps the Zig name, with `_` appended when the name is a Go keyword or
// collides (`default_` is `default`) and `x` in front when it starts with a
// digit (`x132_column` is `132_column`). An exported field (the types listed
// in docs/public-api.md, and the parser actions) is the CamelCase of the Zig
// name: `ParamsSep` is `params_sep`, `StyleID` is `style_id`. CamelCase
// loses the underscore in front of a digit, so the names that have one are
// listed in zigNameDigits.
func zigName(name string) string {
	if z, ok := zigNameDigits[name]; ok {
		return z
	}
	if name != "" && name[0] >= 'A' && name[0] <= 'Z' {
		name = zigSnake(name)
	} else {
		name = strings.TrimSuffix(name, "_")
	}
	if len(name) > 1 && name[0] == 'x' && name[1] >= '0' && name[1] <= '9' {
		name = name[1:]
	}
	return name
}

// zigNameDigits is the exported fields whose Zig name has an underscore in
// front of a digit (scripts/exportfields reports them as "no round trip").
var zigNameDigits = map[string]string{
	"ModifyOtherKeys2":      "modify_other_keys_2",
	"ModifyOtherKeysState2": "modify_other_keys_state_2",
}

// zigSnake is the inverse of zigCamel and of scripts/exportfields' naming: a
// new word starts at a capital that follows a lower-case letter or digit, or
// that is followed by a lower-case letter. `ParamsSep` is `params_sep`,
// `StyleID` is `style_id`.
func zigSnake(name string) string {
	r := []rune(name)
	var sb strings.Builder
	for i, c := range r {
		if c >= 'A' && c <= 'Z' {
			prevLower := i > 0 && !(r[i-1] >= 'A' && r[i-1] <= 'Z')
			nextLower := i+1 < len(r) && r[i+1] >= 'a' && r[i+1] <= 'z'
			if i > 0 && (prevLower || nextLower) {
				sb.WriteByte('_')
			}
			c += 'a' - 'A'
		}
		sb.WriteRune(c)
	}
	return sb.String()
}

// zigCamel is the CamelCase of a tag: `cursor_up` is `CursorUp`.
func zigCamel(tag string) string {
	var sb strings.Builder
	for _, part := range strings.Split(tag, "_") {
		if part == "" {
			continue
		}
		sb.WriteString(strings.ToUpper(part[:1]))
		sb.WriteString(part[1:])
	}
	return sb.String()
}

// zigFieldName is the name of the payload field of a tagged union for the tag
// `tag`: zigCamel, with an `X` in front when the tag starts with a digit
// (docs/mapping.md: `8_bg` is `X8Bg`, `256_fg` is `X256Fg`).
func zigFieldName(tag string) string {
	name := zigCamel(tag)
	if name != "" && name[0] >= '0' && name[0] <= '9' {
		return "X" + name
	}
	return name
}

// dumpIsOptional is Zig's `.optional`: Optional[T].
func dumpIsOptional(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && strings.HasPrefix(t.Name(), "Optional[") &&
		t.NumField() == 2 && t.Field(0).Name == "Value" && t.Field(1).Name == "Valid"
}

// dumpIsUnion is Zig's tagged `.@"union"`: a struct whose first field is Tag.
func dumpIsUnion(t reflect.Type) bool {
	if t.Kind() != reflect.Struct || t.NumField() == 0 || t.Field(0).Name != "Tag" {
		return false
	}
	switch t.Field(0).Type.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return true
	}
	return false
}

// dumpValue is the body of dumpAny: the switch over the type kinds of
// dump.zig:353-466.
func dumpValue(w *bytes.Buffer, prefix string, v reflect.Value) {
	t := v.Type()
	dumpUnnamedEnum(t, prefix)
	switch v.Kind() {
	case reflect.Bool:
		fmt.Fprintf(w, "%s=%d\n", prefix, boolInt(v.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if dumpIsEnum(t) {
			dumpEnum(w, prefix, v)
		} else {
			fmt.Fprintf(w, "%s=%s\n", prefix, dumpInteger(v))
		}
	case reflect.Float32:
		fmt.Fprintf(w, "%s=%s\n", prefix, dumpFloat(v.Float(), 32))
	case reflect.Float64:
		fmt.Fprintf(w, "%s=%s\n", prefix, dumpFloat(v.Float(), 64))
	case reflect.Struct:
		switch {
		case dumpIsOptional(t):
			if dumpField(v, 1).Bool() {
				dumpValue(w, prefix, dumpField(v, 0))
			} else {
				w.WriteString(prefix + "=null\n")
			}
		case dumpIsUnion(t):
			dumpUnion(w, prefix, v)
		case dumpIsSegmentedList(t):
			dumpSegmentedList(w, prefix, v)
		case dumpAllBools(t):
			dumpBools(w, prefix, v)
		default:
			for i := 0; i < t.NumField(); i++ {
				if strings.HasPrefix(t.Field(i).Name, "_") {
					continue
				}
				dumpValue(w, prefix+"."+zigName(t.Field(i).Name), dumpField(v, i))
			}
		}
	case reflect.Array:
		dumpUnnamedEnum(t.Elem(), prefix+"[]")
		if t.Elem().Kind() == reflect.Uint8 && !dumpIsEnum(t.Elem()) {
			b := make([]byte, v.Len())
			for i := range b {
				b[i] = byte(v.Index(i).Uint())
			}
			w.WriteString(prefix + "=")
			writeEscaped(w, b)
			w.WriteString("\n")
		} else {
			dumpItems(w, prefix, v)
		}
	case reflect.Slice:
		dumpUnnamedEnum(t.Elem(), prefix+"[]")
		switch {
		case t.Elem().Kind() == reflect.Uint8 && !dumpIsEnum(t.Elem()):
			w.WriteString(prefix + "=")
			writeEscaped(w, v.Bytes())
			w.WriteString("\n")
		case dumpIsInt(t.Elem()):
			w.WriteString(prefix + "=[")
			for i := 0; i < v.Len(); i++ {
				if i > 0 {
					w.WriteString(",")
				}
				w.WriteString(dumpInteger(v.Index(i)))
			}
			w.WriteString("]\n")
		default:
			dumpItems(w, prefix, v)
		}
	case reflect.Pointer:
		// Zig has no null pointer: a nil Go pointer is a Zig `?*T`.
		if v.IsNil() {
			w.WriteString(prefix + "=null\n")
		} else {
			dumpValue(w, prefix+".*", v.Elem())
		}
	case reflect.Func:
		// Pointers to opaque machinery are not part of the value.
	default:
		panic("dumpAny: unsupported type " + t.String() + " at " + prefix)
	}
}

// dumpEnum is the `.@"enum"` arm: the tag name, or `#N` for an unnamed value.
func dumpEnum(w *bytes.Buffer, prefix string, v reflect.Value) {
	if name, ok := dumpTagName(v); ok {
		fmt.Fprintf(w, "%s=%s\n", prefix, name)
	} else {
		fmt.Fprintf(w, "%s=#%s\n", prefix, dumpInteger(v))
	}
}

// dumpFloat is Zig's `{d}` for a float: the shortest decimal that reads back
// to the same value, without an exponent.
func dumpFloat(f float64, bits int) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	return strconv.FormatFloat(f, 'f', -1, bits)
}

// dumpVoidTags lists every tagged-union type that dumpAny can reach, with the
// tags (Zig names) whose payload type is void: for those the Zig prints the
// `prefix=<tag>` line and nothing else. The union types of the stream action
// (StreamAction and the unions of its payloads) are listed; a union of another
// recorded value is added here when its dump is ported.
// TestCorpusReplaySelfUnionsAgainstZig compares the table with the unions of
// the Zig source, tag by tag.
var dumpVoidTags = map[reflect.Type][]string{
	reflect.TypeOf(StreamAction{}): {
		"bell", "backspace", "linefeed", "carriage_return", "enquiry",
		"tab_clear_current", "tab_clear_all", "tab_set", "tab_reset",
		"index", "next_line", "reverse_index", "full_reset",
		"left_and_right_margin_ambiguous", "save_cursor", "restore_cursor",
		"protected_mode_off", "protected_mode_iso", "protected_mode_dec",
		"xtversion", "kitty_keyboard_query", "dcs_unhook",
		"apc_start", "apc_end", "end_hyperlink", "decaln",
	},
	// Screen.zig:113 SemanticClick, reached through screen.semantic_prompt.click.
	reflect.TypeOf(ScreenSemanticPromptSemanticClick{}): {"none", "click_events"},
	reflect.TypeOf(SGRAttribute{}): {
		"unset", "bold", "reset_bold", "italic", "reset_italic", "faint",
		"reset_underline_color", "overline", "reset_overline", "blink",
		"reset_blink", "inverse", "reset_inverse", "invisible",
		"reset_invisible", "strikethrough", "reset_strikethrough",
		"reset_fg", "reset_bg",
	},
	// Parser.Action (the parser trace, trace.zig:58 parserFeed), and its
	// osc_dispatch payload, osc.Command, with the unions that the OSC and
	// colour payloads of the stream actions and of osc.Command contain.
	reflect.TypeOf(ParserAction{}):                          {"dcs_unhook", "apc_start", "apc_end"},
	reflect.TypeOf(OSCCommand{}):                            {"invalid", "hyperlink_end", "conemu_wait_input"},
	reflect.TypeOf(OSCCommandConemuChangeTabTitlePayload{}): {"reset"},
	reflect.TypeOf(KittyColorOSCRequest{}):                  {},
	reflect.TypeOf(KittyColorKind{}):                        {},
	reflect.TypeOf(OSCParsersColorRequest{}):                {"reset_palette", "reset_special"},
	reflect.TypeOf(OSCParsersColorTarget{}):                 {},
}

// dumpUnion is the `.@"union"` arm: `prefix=<tag>`, then the payload of the
// tag as `prefix.<tag>` unless it is void (see dumpVoidTags).
func dumpUnion(w *bytes.Buffer, prefix string, v reflect.Value) {
	tag := dumpField(v, 0)
	if !dumpIsEnum(tag.Type()) {
		panic(fmt.Sprintf("dumpAny: union type %s: its tag type %s has no String or TagName method (at %s)",
			v.Type(), tag.Type(), prefix))
	}
	name, ok := dumpTagName(tag)
	if !ok {
		panic("dumpAny: tagged union with an unnamed tag at " + prefix)
	}
	fmt.Fprintf(w, "%s=%s\n", prefix, name)
	if payload, ok := dumpUnionPayload(v, name, prefix); ok {
		dumpValue(w, prefix+"."+name, payload)
	}
}

// dumpUnionPayload is the payload of the union v for the tag `name`: the
// field named by zigFieldName, and false for a tag whose Zig payload is void
// (dumpVoidTags). It panics for a union type that is not listed and for a
// non-void tag without a field, instead of leaving the line out. v must be
// addressable.
func dumpUnionPayload(v reflect.Value, name string, prefix string) (reflect.Value, bool) {
	voids, ok := dumpVoidTags[v.Type()]
	if !ok {
		panic(fmt.Sprintf("dumpAny: union type %s is not in dumpVoidTags (tag %q at %s)", v.Type(), name, prefix))
	}
	if slices.Contains(voids, name) {
		return reflect.Value{}, false
	}
	f, ok := v.Type().FieldByName(zigFieldName(name))
	if !ok {
		panic(fmt.Sprintf("dumpAny: union type %s: tag %q is not void in Zig but the type has no field %s (at %s)",
			v.Type(), name, zigFieldName(name), prefix))
	}
	return dumpField(v, f.Index[0]), true
}

// dumpAllBools is Zig's allBools (dump.zig:470): more than one field that
// does not start with `_`, and every such field a bool.
func dumpAllBools(t reflect.Type) bool {
	n := 0
	for i := 0; i < t.NumField(); i++ {
		if strings.HasPrefix(t.Field(i).Name, "_") {
			continue
		}
		if t.Field(i).Type.Kind() != reflect.Bool {
			return false
		}
		n++
	}
	return n > 1
}

// dumpBools is the allBools arm: a struct made only of bools is printed on
// one line as the names that are set.
func dumpBools(w *bytes.Buffer, prefix string, v reflect.Value) {
	w.WriteString(prefix + "=")
	any_ := false
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		if strings.HasPrefix(name, "_") {
			continue
		}
		if dumpField(v, i).Bool() {
			if any_ {
				w.WriteString("|")
			}
			any_ = true
			w.WriteString(zigName(name))
		}
	}
	if !any_ {
		w.WriteString("none")
	}
	w.WriteString("\n")
}

// dumpIsSegmentedList is Zig's `@hasField(T, "prealloc_segment")` test
// (dump.zig:375): the one SegmentedList of the recorded values, osc.color.List.
func dumpIsSegmentedList(t reflect.Type) bool {
	return t == reflect.TypeOf(OSCParsersColorList{})
}

// dumpSegmentedList is the std.SegmentedList arm (dump.zig:375): `prefix.len`
// and every item as `prefix[i]:` and `prefix[]`. Like Zig's, it is the
// logical contents; the Go list is a struct whose one field is the slice of
// its items, which dumpItems writes in the same format.
func dumpSegmentedList(w *bytes.Buffer, prefix string, v reflect.Value) {
	dumpItems(w, prefix, dumpField(v, 0))
}

// dumpItems is the slice/array arm for items that are not bytes or integers:
// `prefix.len=N`, then `prefix[i]:` and the item as `prefix[]`.
func dumpItems(w *bytes.Buffer, prefix string, v reflect.Value) {
	fmt.Fprintf(w, "%s.len=%d\n", prefix, v.Len())
	for i := 0; i < v.Len(); i++ {
		fmt.Fprintf(w, "%s[%d]:\n", prefix, i)
		dumpValue(w, prefix+"[]", v.Index(i))
	}
}
