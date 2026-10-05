package terminal

// Corpus test for page.zig: Page.layout, Capacity.maxCols and
// Capacity.adjust over the recorded capacities of data/layout/page_layout.txt.
// The generator is the Zig dump test `term51: Page.layout: dump exact data
// when TERM51_LAYOUT_DUMP_DIR is set` near the end of page.zig (term51 build;
// helpers term51DumpCapacity and term51LayoutLines), which is not translated
// (scripts/not-translated.txt).
//
// The file has two kinds of records, all numbers decimal:
//
//   - a capacity: a header line
//
//     ## <name> cols=C rows=R styles=S hyperlink_bytes=H grapheme_bytes=G string_bytes=B
//
//     followed by one line per field of `Page.layout(cap)`, in declaration
//     order, nested layouts flattened (`layout.total_size=N`,
//     `layout.styles_layout.cap=N`, ..., `layout.capacity.string_bytes=N`;
//     45 lines), and then `maxCols=N` or `maxCols=null`. The names are
//     std_capacity_test_build (grapheme_bytes 512), std_capacity_production
//     (8192), default_80x24 (`.{ .cols = 80, .rows = 24 }`), and min_<field>,
//     max_<field>, pow_<field>, grid for capacities derived from it; the
//     header line carries every field, so a record stands on its own.
//   - an adjustment:
//
//     adjust <base> cols=N -> cols=C rows=R styles=S hyperlink_bytes=H grapheme_bytes=G string_bytes=B
//     adjust <base> cols=N error=OutOfMemory
//
//     the result of `base.adjust(.{ .cols = N })`. The five bases are
//     std_capacity_test_build, std_capacity_production and default_80x24 (the
//     capacities of the headers of those names) and styles_1000
//     (`.{ .cols = 80, .rows = 24, .styles = 1000 }`) and big_strings
//     (`.{ .cols = 80, .rows = 24, .string_bytes = 100000 }`), which have no
//     header and are written out below.
//
// The test regenerates every line from the Go port and compares the whole
// file line by line.

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// cpl_parseCapacity parses the `key=value` fields of a header line.
func cpl_parseCapacity(t *testing.T, fields []string) PageCapacity {
	t.Helper()
	if len(fields) != 6 {
		t.Fatalf("capacity: %d fields, want 6: %q", len(fields), fields)
	}
	var c PageCapacity
	for _, f := range fields {
		kv := strings.SplitN(f, "=", 2)
		if len(kv) != 2 {
			t.Fatalf("bad field %q", f)
		}
		v, err := strconv.ParseUint(kv[1], 10, 32)
		if err != nil {
			t.Fatalf("bad field %q: %v", f, err)
		}
		switch kv[0] {
		case "cols":
			c.cols = SizeCellCountInt(v)
		case "rows":
			c.rows = SizeCellCountInt(v)
		case "styles":
			c.styles = SizeStyleCountInt(v)
		case "hyperlink_bytes":
			c.hyperlink_bytes = SizeHyperlinkCountInt(v)
		case "grapheme_bytes":
			c.grapheme_bytes = SizeGraphemeBytesInt(v)
		case "string_bytes":
			c.string_bytes = SizeStringBytesInt(v)
		default:
			t.Fatalf("unknown field %q", f)
		}
	}
	return c
}

// cpl_capacity formats a capacity as the dump does.
func cpl_capacity(c PageCapacity) string {
	return fmt.Sprintf("cols=%d rows=%d styles=%d hyperlink_bytes=%d grapheme_bytes=%d string_bytes=%d",
		c.cols, c.rows, c.styles, c.hyperlink_bytes, c.grapheme_bytes, c.string_bytes)
}

// cpl_layoutLines is term51LayoutLines(w, "layout", l).
func cpl_layoutLines(l PageLayout) []string {
	type kv struct {
		k string
		v uint64
	}
	fields := []kv{
		{"total_size", l.total_size},
		{"rows_start", l.rows_start},
		{"rows_size", l.rows_size},
		{"cells_start", l.cells_start},
		{"cells_size", l.cells_size},
		{"styles_start", l.styles_start},
		{"styles_layout.cap", uint64(l.styles_layout.cap)},
		{"styles_layout.table_cap", uint64(l.styles_layout.table_cap)},
		{"styles_layout.table_mask", uint64(l.styles_layout.table_mask)},
		{"styles_layout.table_start", l.styles_layout.table_start},
		{"styles_layout.items_start", l.styles_layout.items_start},
		{"styles_layout.total_size", l.styles_layout.total_size},
		{"grapheme_alloc_start", l.grapheme_alloc_start},
		{"grapheme_alloc_layout.total_size", l.grapheme_alloc_layout.total_size},
		{"grapheme_alloc_layout.bitmap_count", uint64(l.grapheme_alloc_layout.bitmap_count)},
		{"grapheme_alloc_layout.bitmap_start", l.grapheme_alloc_layout.bitmap_start},
		{"grapheme_alloc_layout.chunks_start", l.grapheme_alloc_layout.chunks_start},
		{"grapheme_map_start", l.grapheme_map_start},
		{"grapheme_map_layout.total_size", l.grapheme_map_layout.total_size},
		{"grapheme_map_layout.keys_start", l.grapheme_map_layout.keys_start},
		{"grapheme_map_layout.vals_start", l.grapheme_map_layout.vals_start},
		{"grapheme_map_layout.capacity", uint64(l.grapheme_map_layout.capacity)},
		{"string_alloc_start", l.string_alloc_start},
		{"string_alloc_layout.total_size", l.string_alloc_layout.total_size},
		{"string_alloc_layout.bitmap_count", uint64(l.string_alloc_layout.bitmap_count)},
		{"string_alloc_layout.bitmap_start", l.string_alloc_layout.bitmap_start},
		{"string_alloc_layout.chunks_start", l.string_alloc_layout.chunks_start},
		{"hyperlink_map_start", l.hyperlink_map_start},
		{"hyperlink_map_layout.total_size", l.hyperlink_map_layout.total_size},
		{"hyperlink_map_layout.keys_start", l.hyperlink_map_layout.keys_start},
		{"hyperlink_map_layout.vals_start", l.hyperlink_map_layout.vals_start},
		{"hyperlink_map_layout.capacity", uint64(l.hyperlink_map_layout.capacity)},
		{"hyperlink_set_start", l.hyperlink_set_start},
		{"hyperlink_set_layout.cap", uint64(l.hyperlink_set_layout.cap)},
		{"hyperlink_set_layout.table_cap", uint64(l.hyperlink_set_layout.table_cap)},
		{"hyperlink_set_layout.table_mask", uint64(l.hyperlink_set_layout.table_mask)},
		{"hyperlink_set_layout.table_start", l.hyperlink_set_layout.table_start},
		{"hyperlink_set_layout.items_start", l.hyperlink_set_layout.items_start},
		{"hyperlink_set_layout.total_size", l.hyperlink_set_layout.total_size},
		{"capacity.cols", uint64(l.capacity.cols)},
		{"capacity.rows", uint64(l.capacity.rows)},
		{"capacity.styles", uint64(l.capacity.styles)},
		{"capacity.hyperlink_bytes", uint64(l.capacity.hyperlink_bytes)},
		{"capacity.grapheme_bytes", uint64(l.capacity.grapheme_bytes)},
		{"capacity.string_bytes", uint64(l.capacity.string_bytes)},
	}
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, fmt.Sprintf("layout.%s=%d", f.k, f.v))
	}
	return out
}

// Go-only: corpus (Page.layout, Capacity.maxCols and Capacity.adjust must reproduce every line of data/layout/page_layout.txt; the Zig dump test is not translated)
func TestPage_corpus_layout(t *testing.T) {
	want := corpusLines(t, "data/layout/page_layout.txt")

	// The adjust bases without a header (see the file comment).
	bases := map[string]PageCapacity{}
	styles_1000 := PageCapacityInit(80, 24)
	styles_1000.styles = 1000
	bases["styles_1000"] = styles_1000
	big_strings := PageCapacityInit(80, 24)
	big_strings.string_bytes = 100000
	bases["big_strings"] = big_strings

	var got []string
	headers, adjusts := 0, 0
	for i := 0; i < len(want); {
		line := want[i]
		switch {
		case strings.HasPrefix(line, "## "):
			f := strings.Fields(line[3:])
			if len(f) < 1 {
				t.Fatalf("line %d: bad header %q", i+1, line)
			}
			c := cpl_parseCapacity(t, f[1:])
			switch f[0] {
			case "std_capacity_test_build", "std_capacity_production", "default_80x24":
				bases[f[0]] = c
			}
			got = append(got, "## "+f[0]+" "+cpl_capacity(c))
			got = append(got, cpl_layoutLines(PageLayoutOf(c))...)
			if m := c.MaxCols(); m.Valid {
				got = append(got, fmt.Sprintf("maxCols=%d", m.Value))
			} else {
				got = append(got, "maxCols=null")
			}
			headers++
			// The layout lines and maxCols follow the header.
			i += 1 + len(cpl_layoutLines(PageLayout{})) + 1
		case strings.HasPrefix(line, "adjust "):
			f := strings.Fields(line)
			if len(f) < 3 || !strings.HasPrefix(f[2], "cols=") {
				t.Fatalf("line %d: bad adjust line %q", i+1, line)
			}
			base, ok := bases[f[1]]
			if !ok {
				t.Fatalf("line %d: unknown adjust base %q", i+1, f[1])
			}
			cols, err := strconv.ParseUint(strings.TrimPrefix(f[2], "cols="), 10, 16)
			if err != nil {
				t.Fatalf("line %d: bad cols %q: %v", i+1, f[2], err)
			}
			r, err := base.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(cols))})
			if err != nil {
				got = append(got, fmt.Sprintf("adjust %s cols=%d error=%s", f[1], cols, zigErrorName(err)))
			} else {
				got = append(got, fmt.Sprintf("adjust %s cols=%d -> %s", f[1], cols, cpl_capacity(r)))
			}
			adjusts++
			i++
		default:
			t.Fatalf("line %d: unexpected line %q", i+1, line)
		}
	}

	if len(got) != len(want) {
		t.Fatalf("generated %d lines, the file has %d", len(got), len(want))
	}
	failures := 0
	for i := range want {
		if got[i] != want[i] {
			failures++
			if failures <= 10 {
				t.Errorf("line %d: got %q, want %q", i+1, got[i], want[i])
			}
		}
	}
	if failures > 0 {
		t.Errorf("%d of %d lines differ", failures, len(want))
	}
	if headers != 505 || adjusts != 3325 {
		t.Errorf("got %d capacities and %d adjustments, want 505 and 3325", headers, adjusts)
	}
}
