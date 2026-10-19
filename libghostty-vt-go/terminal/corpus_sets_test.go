package terminal

// Data-comparison test for ref_counted_set.go against the recorded corpus
// file data/sets/capacities.txt, written by the Zig dump test in style.zig
// (`term51: Style.hash: dump exact data when TERM51_SET_DUMP_DIR is set`).
// The file has, for the u16-id set,
//
//	c <n> <capacityForCount(n)>                       for n in 0..65536
//	l <cap> <items_cap> <table_cap> <table_mask>      for cap in 0..65536
//
// where the `l` values are Layout.init(cap).cap, .table_cap and .table_mask.

import (
	"strconv"
	"strings"
	"testing"
)

func TestRefCountedSet_corpus_capacities(t *testing.T) {
	lines := corpusLines(t, "data/sets/capacities.txt")
	atoi := func(s string) uint {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			t.Fatalf("bad number %q: %v", s, err)
		}
		return uint(v)
	}
	nc, nl, bad := 0, 0, 0
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		f := strings.Fields(line)
		switch f[0] {
		case "c":
			if len(f) != 3 {
				t.Fatalf("bad line %q", line)
			}
			n, want := atoi(f[1]), atoi(f[2])
			nc++
			if got := RefCountedSetCapacityForCount(n); got != want {
				t.Errorf("capacityForCount(%d) = %d, want %d", n, got, want)
				bad++
			}
		case "l":
			if len(f) != 5 {
				t.Fatalf("bad line %q", line)
			}
			cap_ := atoi(f[1])
			itemsCap, tableCap, tableMask := atoi(f[2]), atoi(f[3]), atoi(f[4])
			nl++
			l := RefCountedSetLayoutInit[uint16](cap_, sm2_itemSizes16)
			if l.cap != itemsCap || l.table_cap != tableCap || uint(l.table_mask) != tableMask {
				t.Errorf("Layout.init(%d) = cap %d table_cap %d table_mask %d, want %d %d %d",
					cap_, l.cap, l.table_cap, l.table_mask, itemsCap, tableCap, tableMask)
				bad++
			}
		default:
			t.Fatalf("bad line %q", line)
		}
		if bad >= 20 {
			t.Fatal("too many differences")
		}
	}
	// Every capacity from 0 to 65536 inclusive is recorded.
	if nc != 65537 || nl != 65537 {
		t.Fatalf("got %d c lines and %d l lines, want 65537 of each", nc, nl)
	}
}
