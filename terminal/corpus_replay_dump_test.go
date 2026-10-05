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
//   - Zig's SegmentedList and ArrayList special cases of dumpAny do not exist
//     in Go (a slice is already the list of its items);
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

	dumpAny(w, "term.cols", t.cols)
	dumpAny(w, "term.rows", t.rows)
	dumpAny(w, "term.width_px", t.width_px)
	dumpAny(w, "term.height_px", t.height_px)
	dumpAny(w, "term.status_display", t.status_display)
	dumpAny(w, "term.scrolling_region", t.scrolling_region)
	dumpAny(w, "term.previous_char", t.previous_char)
	dumpAny(w, "term.mouse_shape", t.mouse_shape)
	dumpAny(w, "term.flags", t.flags)
	dumpAny(w, "term.modes.values", t.modes.values)
	dumpAny(w, "term.modes.saved", t.modes.saved)
	dumpAny(w, "term.modes.default", t.modes.default_)
	dumpAny(w, "term.colors.background", t.colors.background)
	dumpAny(w, "term.colors.foreground", t.colors.foreground)
	dumpAny(w, "term.colors.cursor", t.colors.cursor)

	// Palette: only entries that differ from the original, plus the mask.
	for i := 0; i < 256; i++ {
		cur := t.colors.palette.current[i]
		orig := t.colors.palette.original[i]
		masked := t.colors.palette.mask.isSet(i)
		if masked || cur.r != orig.r || cur.g != orig.g || cur.b != orig.b {
			fmt.Fprintf(w, "term.colors.palette[%d]=%02x%02x%02x mask=%d\n", i, cur.r, cur.g, cur.b, boolInt(masked))
		}
	}

	w.WriteString("term.pwd=")
	writeEscaped(w, t.pwd)
	w.WriteString("\n")

	w.WriteString("term.tabstops=")
	{
		first := true
		for x := uint(0); x < uint(t.cols); x++ {
			if !t.tabstops.Get(x) {
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

	fmt.Fprintf(w, "term.screens.active_key=%s\n", t.screens.active_key.String())

	for _, key := range []ScreenSetKey{ScreenSetKeyPrimary, ScreenSetKeyAlternate} {
		if screen := t.screens.Get(key); screen != nil {
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
	dumpAny(w, "screen.no_scrollback", s.no_scrollback)
	dumpAny(w, "screen.cursor.x", s.cursor.x)
	dumpAny(w, "screen.cursor.y", s.cursor.y)
	dumpAny(w, "screen.cursor.cursor_style", s.cursor.cursor_style)
	dumpAny(w, "screen.cursor.pending_wrap", s.cursor.pending_wrap)
	dumpAny(w, "screen.cursor.protected", s.cursor.protected)
	w.WriteString("screen.cursor.style=")
	writeStyle(w, s.cursor.style)
	w.WriteString("\n")
	dumpAny(w, "screen.cursor.semantic_content", s.cursor.semantic_content)
	dumpAny(w, "screen.cursor.semantic_content_clear_eol", s.cursor.semantic_content_clear_eol)
	if link := s.cursor.hyperlink; link != nil {
		w.WriteString("screen.cursor.hyperlink=")
		switch link.id.Tag {
		case HyperlinkHyperlinkIdExplicit:
			w.WriteString("explicit:")
			writeEscaped(w, link.id.Explicit)
		case HyperlinkHyperlinkIdImplicit:
			fmt.Fprintf(w, "implicit:%d", link.id.Implicit)
		}
		w.WriteString(" uri=")
		writeEscaped(w, link.uri)
		w.WriteString("\n")
	} else {
		w.WriteString("screen.cursor.hyperlink=null\n")
	}
	dumpAny(w, "ident.screen.cursor.style_id", s.cursor.style_id)
	dumpAny(w, "ident.screen.cursor.hyperlink_id", s.cursor.hyperlink_id)
	dumpAny(w, "ident.screen.cursor.hyperlink_implicit_id", s.cursor.hyperlink_implicit_id)

	if s.saved_cursor.Valid {
		sc := s.saved_cursor.Value
		dumpAny(w, "screen.saved_cursor.x", sc.x)
		dumpAny(w, "screen.saved_cursor.y", sc.y)
		w.WriteString("screen.saved_cursor.style=")
		writeStyle(w, sc.style)
		w.WriteString("\n")
		dumpAny(w, "screen.saved_cursor.protected", sc.protected)
		dumpAny(w, "screen.saved_cursor.pending_wrap", sc.pending_wrap)
		dumpAny(w, "screen.saved_cursor.origin", sc.origin)
		dumpAny(w, "screen.saved_cursor.charset", sc.charset)
	} else {
		w.WriteString("screen.saved_cursor=null\n")
	}

	dumpAny(w, "screen.charset", s.charset)
	dumpAny(w, "screen.protected_mode", s.protected_mode)
	dumpAny(w, "screen.kitty_keyboard.idx", s.kitty_keyboard.idx)
	for i := range s.kitty_keyboard.flags {
		dumpAny(w, fmt.Sprintf("screen.kitty_keyboard.flags[%d]", i), s.kitty_keyboard.flags[i])
	}
	dumpAny(w, "screen.semantic_prompt", s.semantic_prompt)
	dumpAny(w, "screen.dirty", s.dirty)
	selection := "set"
	if !s.selection.Valid {
		selection = "null"
	}
	fmt.Fprintf(w, "screen.selection=%s\n", selection)

	pages := &s.pages
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
		fmt.Fprintf(w, "pages.active_top=%d\n", pt.Screen.y)
	}
	{
		tl := pages.GetTopLeft(PointTagViewport)
		pt := dumpUnwrap(pages.PointFromPin(PointTagScreen, tl))
		fmt.Fprintf(w, "pages.viewport_top=%d\n", pt.Screen.y)
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
		page := &pin.node.data
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
			cell.content.color_rgb.r,
			cell.content.color_rgb.g,
			cell.content.color_rgb.b,
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
	writeColor(w, s.fg_color)
	w.WriteString(",bg=")
	writeColor(w, s.bg_color)
	w.WriteString(",ul=")
	writeColor(w, s.underline_color)
	w.WriteString(",flags=")
	any_ := false
	for _, flag := range []struct {
		name string
		set  bool
	}{
		{"bold", s.flags.bold},
		{"italic", s.flags.italic},
		{"faint", s.flags.faint},
		{"blink", s.flags.blink},
		{"inverse", s.flags.inverse},
		{"invisible", s.flags.invisible},
		{"strikethrough", s.flags.strikethrough},
		{"overline", s.flags.overline},
	} {
		if flag.set {
			if any_ {
				w.WriteString("|")
			}
			any_ = true
			w.WriteString(flag.name)
		}
	}
	if s.flags.underline != SGRAttributeUnderlineNone {
		if any_ {
			w.WriteString("|")
		}
		any_ = true
		fmt.Fprintf(w, "underline:%s", s.flags.underline.String())
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
		fmt.Fprintf(w, "rgb:%02x%02x%02x", c.Rgb.r, c.Rgb.g, c.Rgb.b)
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

// dumpIsEnum is the Go test for Zig's `.@"enum"`: an integer type with a
// String method.
func dumpIsEnum(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return t.Implements(dumpStringerType)
	}
	return false
}

// dumpIsInt is Zig's `.int` (and not an enum).
func dumpIsInt(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return !t.Implements(dumpStringerType)
	}
	return false
}

// dumpInteger is the integer value of v as text.
func dumpInteger(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	}
	return strconv.FormatUint(v.Uint(), 10)
}

// dumpTagName is std.enums.tagName: the name of an enum value, and false for
// a value that has none.
func dumpTagName(v reflect.Value) (string, bool) {
	name := v.Interface().(fmt.Stringer).String()
	return name, name != ""
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

// zigName is the Zig field name of a Go struct field: the Go port appends
// `_` to a name that is a Go keyword or collides (`default_` is `default`)
// and prefixes `x` to a name that starts with a digit (`x132_column` is
// `132_column`).
func zigName(name string) string {
	name = strings.TrimSuffix(name, "_")
	if len(name) > 1 && name[0] == 'x' && name[1] >= '0' && name[1] <= '9' {
		name = name[1:]
	}
	return name
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
	reflect.TypeOf(SGRAttribute{}): {
		"unset", "bold", "reset_bold", "italic", "reset_italic", "faint",
		"reset_underline_color", "overline", "reset_overline", "blink",
		"reset_blink", "inverse", "reset_inverse", "invisible",
		"reset_invisible", "strikethrough", "reset_strikethrough",
		"reset_fg", "reset_bg",
	},
	reflect.TypeOf(KittyColorOSCRequest{}):   {},
	reflect.TypeOf(KittyColorKind{}):         {},
	reflect.TypeOf(OSCParsersColorRequest{}): {"reset_palette", "reset_special"},
	reflect.TypeOf(OSCParsersColorTarget{}):  {},
}

// dumpUnion is the `.@"union"` arm: `prefix=<tag>`, then the payload of the
// tag as `prefix.<tag>` unless it is void (see dumpVoidTags).
func dumpUnion(w *bytes.Buffer, prefix string, v reflect.Value) {
	tag := dumpField(v, 0)
	name, ok := dumpTagName(tag)
	if !ok {
		panic("dumpAny: tagged union with an unnamed tag at " + prefix)
	}
	fmt.Fprintf(w, "%s=%s\n", prefix, name)
	voids, ok := dumpVoidTags[v.Type()]
	if !ok {
		panic(fmt.Sprintf("dumpAny: union type %s is not in dumpVoidTags (tag %q at %s)", v.Type(), name, prefix))
	}
	if slices.Contains(voids, name) {
		return
	}
	f, ok := v.Type().FieldByName(zigFieldName(name))
	if !ok {
		panic(fmt.Sprintf("dumpAny: union type %s: tag %q is not void in Zig but the type has no field %s (at %s)",
			v.Type(), name, zigFieldName(name), prefix))
	}
	dumpValue(w, prefix+"."+name, dumpField(v, f.Index[0]))
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

// dumpItems is the slice/array arm for items that are not bytes or integers:
// `prefix.len=N`, then `prefix[i]:` and the item as `prefix[]`.
func dumpItems(w *bytes.Buffer, prefix string, v reflect.Value) {
	fmt.Fprintf(w, "%s.len=%d\n", prefix, v.Len())
	for i := 0; i < v.Len(); i++ {
		fmt.Fprintf(w, "%s[%d]:\n", prefix, i)
		dumpValue(w, prefix+"[]", v.Index(i))
	}
}
