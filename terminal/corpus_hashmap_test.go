package terminal

// Option A (keep upstream's byte-offset bookkeeping on Go slices): confirmed by the project owner 2026-10-04.

// Data-comparison test for hash_map.go against the recorded corpus file
// data/hashmap/hashmap_data.txt, written by the Zig dump test in hash_map.zig
// (`term51: OffsetHashMap: dump exact data when TERM51_HASHMAP_DUMP_DIR is
// set`). The file has
//
//	# header_size=<n> metadata_size=<n>
//	# grapheme map: key_size=<n> value_size=<n> base_align=<n>
//	# hyperlink map: key_size=<n> value_size=<n> base_align=<n>
//
// then a layout section,
//
//	L <n> <capacity> g <total_size> <keys_start> <vals_start> h <total_size> <keys_start> <vals_start>
//
// for the requested counts n = 0..3000 and every 7th after that up to 70000
// (capacity is n rounded up to a power of two, 0 for 0; g is the grapheme
// map's layout, h the hyperlink map's), then three scenarios,
//
//	scenario <name> cap=<cap> seed=0x<seed> keys=<kind>
//	op put <key> <value> | op remove <key>
//	phase <name> count=<count> capacity=<cap> available=<cap-count>
//	K <key> <hash hex> <slot> <control byte hex> <value>   live keys
//	T <n> <slot>...                                        tombstone slots
//	I <n> <slot> <key>                                     iterator order
//
// where the value is <offset>:<len> (grapheme map, scenario names starting
// with "grapheme") or <id> (hyperlink map). Every operation is replayed on a
// map of capacity <cap>; after each phase line the count, capacity and
// availability, the hash, slot, control byte and value of every K key, the
// tombstone slots and the iterator order are compared.

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// hmRunner is a map of one of the two types, driven by text.
type hmRunner interface {
	put(key uint32, value string) error
	remove(key uint32) bool
	count() uint32
	capacity() uint32
	slotOf(key uint32) (uint, bool)
	ctl(slot uint) uint8
	valueOf(key uint32) (string, bool)
	tombstones() []uint
	startIter()
	next() (slot uint, key uint32, ok bool)
}

type hmRun[V any] struct {
	box    hmBox[V]
	m      hashMapUnmanaged[SizeOffsetInt, V, hmCtxT]
	it     HashMapIterator[SizeOffsetInt, V, hmCtxT]
	parse  func(s string) (V, error)
	format func(v V) string
}

func hmNewRun[V any](sizes hashMapEntrySizes, cap uint32, parse func(s string) (V, error), format func(v V) string) *hmRun[V] {
	r := &hmRun[V]{box: hmBoxInit[V](sizes, cap), parse: parse, format: format}
	r.m = r.box.map_()
	return r
}

func (r *hmRun[V]) put(key uint32, value string) error {
	v, err := r.parse(value)
	if err != nil {
		return err
	}
	return r.m.Put(hmK(key), v)
}

func (r *hmRun[V]) remove(key uint32) bool { return r.m.Remove(hmK(key)) }
func (r *hmRun[V]) count() uint32          { return r.m.Count() }
func (r *hmRun[V]) capacity() uint32       { return r.m.Capacity() }

func (r *hmRun[V]) slotOf(key uint32) (uint, bool) {
	s := r.m.getIndex(hmK(key), hmCtx)
	return s.Value, s.Valid
}

func (r *hmRun[V]) ctl(slot uint) uint8 { return hmCtl(r.m, slot) }

func (r *hmRun[V]) valueOf(key uint32) (string, bool) {
	v := r.m.Get(hmK(key))
	if !v.Valid {
		return "", false
	}
	return r.format(v.Value), true
}

func (r *hmRun[V]) tombstones() []uint {
	var out []uint
	for s := range uint(r.m.Capacity()) {
		if hmCtl(r.m, s) == 1 {
			out = append(out, s)
		}
	}
	return out
}

func (r *hmRun[V]) startIter() { r.it = r.m.Iterator() }

func (r *hmRun[V]) next() (uint, uint32, bool) {
	e := r.it.Next()
	if !e.Valid {
		return 0, 0, false
	}
	return r.m.keyIndex(e.Value.key_ptr), *e.Value.key_ptr, true
}

func hmNewGraphemeRun(cap uint32) hmRunner {
	return hmNewRun(pageGraphemeMapEntrySizes, cap,
		func(s string) ([]uint32, error) {
			o, l, ok := strings.Cut(s, ":")
			if !ok {
				return nil, fmt.Errorf("bad grapheme value %q", s)
			}
			off, err1 := strconv.ParseUint(o, 10, 32)
			n, err2 := strconv.ParseUint(l, 10, 32)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("bad grapheme value %q", s)
			}
			return hmSl(uint32(off), uint(n)), nil
		},
		func(v []uint32) string {
			d := hmDesc(v)
			return fmt.Sprintf("%d:%d", d.offset, d.len)
		})
}

func hmNewHyperlinkRun(cap uint32) hmRunner {
	return hmNewRun(hyperlinkMapEntrySizes, cap,
		func(s string) (HyperlinkID, error) {
			v, err := strconv.ParseUint(s, 10, 16)
			return HyperlinkID(v), err
		},
		func(v HyperlinkID) string { return strconv.FormatUint(uint64(v), 10) })
}

func TestHashMap_corpus_hashmap_data(t *testing.T) {
	lines := corpusLines(t, "data/hashmap/hashmap_data.txt")
	atoi := func(s string) uint64 {
		t.Helper()
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			t.Fatalf("bad number %q: %v", s, err)
		}
		return v
	}
	bad := 0
	fail := func(lineno int, format string, args ...any) {
		t.Helper()
		t.Errorf("line %d: %s", lineno, fmt.Sprintf(format, args...))
		bad++
		if bad >= 20 {
			t.Fatal("too many differences")
		}
	}

	var (
		run     hmRunner
		nL      int
		nScen   int
		nOps    int
		nPhases int
		// the phase being checked
		phase       string
		wantCount   uint32
		kLines      uint32
		iLines      uint32
		sawT        bool
		iterStarted bool
	)
	// finishPhase checks what the lines of the phase before it must have
	// covered: every live key listed, the iterator exhausted.
	finishPhase := func(lineno int) {
		if phase == "" {
			return
		}
		if kLines != wantCount {
			fail(lineno, "phase %s: %d K lines, count %d", phase, kLines, wantCount)
		}
		if !sawT {
			fail(lineno, "phase %s: no T line", phase)
		}
		if iLines != wantCount {
			fail(lineno, "phase %s: %d I lines, count %d", phase, iLines, wantCount)
		}
		if iterStarted {
			if _, _, ok := run.next(); ok {
				fail(lineno, "phase %s: the iterator yields more entries than recorded", phase)
			}
		}
		phase = ""
	}

	for i, line := range lines {
		lineno := i + 1
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			f := strings.Fields(line[1:])
			switch {
			case len(f) == 2 && strings.HasPrefix(f[0], "header_size="):
				// # header_size=16 metadata_size=1
				if f[0] != "header_size="+strconv.Itoa(int(hmHeaderSize)) || f[1] != "metadata_size="+strconv.Itoa(int(hmMetadataSize)) {
					fail(lineno, "header line %q", line)
				}
			case len(f) == 5 && (f[0] == "grapheme" || f[0] == "hyperlink") && f[1] == "map:":
				// # grapheme map: key_size=4 value_size=16 base_align=8
				s := pageGraphemeMapEntrySizes
				if f[0] == "hyperlink" {
					s = hyperlinkMapEntrySizes
				}
				want := fmt.Sprintf("key_size=%d value_size=%d base_align=%d", s.key_size, s.val_size, hmBaseAlign(s))
				if got := strings.Join(f[2:], " "); got != want {
					fail(lineno, "%s map sizes: file has %q, Go has %q", f[0], got, want)
				}
			}
			continue
		}
		f := strings.Fields(line)
		switch f[0] {
		case "L":
			// L n cap g total keys vals h total keys vals
			if len(f) != 11 || f[3] != "g" || f[7] != "h" {
				t.Fatalf("line %d: bad line %q", lineno, line)
			}
			nL++
			n := atoi(f[1])
			cap_ := uint32(atoi(f[2]))
			var wantCap uint32
			if n != 0 {
				wantCap = hmCeilPowerOfTwo(uint32(n))
			}
			if cap_ != wantCap {
				fail(lineno, "capacity of %d is %d in the file, ceilPowerOfTwo gives %d", n, cap_, wantCap)
			}
			g := hmGLayout(cap_)
			h := hmHLayout(cap_)
			if g.capacity != cap_ || h.capacity != cap_ {
				fail(lineno, "layout(%d).capacity = %d, %d", cap_, g.capacity, h.capacity)
			}
			if g.total_size != atoi(f[4]) || g.keys_start != atoi(f[5]) || g.vals_start != atoi(f[6]) {
				fail(lineno, "grapheme layout(%d) = %d %d %d, want %s %s %s", cap_, g.total_size, g.keys_start, g.vals_start, f[4], f[5], f[6])
			}
			if h.total_size != atoi(f[8]) || h.keys_start != atoi(f[9]) || h.vals_start != atoi(f[10]) {
				fail(lineno, "hyperlink layout(%d) = %d %d %d, want %s %s %s", cap_, h.total_size, h.keys_start, h.vals_start, f[8], f[9], f[10])
			}
		case "scenario":
			// scenario <name> cap=<cap> seed=0x<seed> keys=<kind>
			finishPhase(lineno)
			if len(f) != 5 || !strings.HasPrefix(f[2], "cap=") {
				t.Fatalf("line %d: bad line %q", lineno, line)
			}
			nScen++
			cap_ := uint32(atoi(strings.TrimPrefix(f[2], "cap=")))
			switch {
			case strings.HasPrefix(f[1], "grapheme"):
				run = hmNewGraphemeRun(cap_)
			case strings.HasPrefix(f[1], "hyperlink"):
				run = hmNewHyperlinkRun(cap_)
			default:
				t.Fatalf("line %d: unknown scenario %q", lineno, f[1])
			}
		case "op":
			if run == nil {
				t.Fatalf("line %d: op before a scenario", lineno)
			}
			finishPhase(lineno)
			nOps++
			switch {
			case len(f) == 4 && f[1] == "put":
				if err := run.put(uint32(atoi(f[2])), f[3]); err != nil {
					fail(lineno, "put %s %s: %v", f[2], f[3], err)
				}
			case len(f) == 3 && f[1] == "remove":
				if !run.remove(uint32(atoi(f[2]))) {
					fail(lineno, "remove %s: the key is not in the map", f[2])
				}
			default:
				t.Fatalf("line %d: bad line %q", lineno, line)
			}
		case "phase":
			// phase <name> count=<count> capacity=<cap> available=<cap-count>
			if run == nil || len(f) != 5 {
				t.Fatalf("line %d: bad line %q", lineno, line)
			}
			finishPhase(lineno)
			phase = f[1]
			wantCount = uint32(atoi(strings.TrimPrefix(f[2], "count=")))
			wantCap := uint32(atoi(strings.TrimPrefix(f[3], "capacity=")))
			wantAvail := uint32(atoi(strings.TrimPrefix(f[4], "available=")))
			nPhases++
			kLines, iLines, sawT, iterStarted = 0, 0, false, false
			if run.count() != wantCount {
				fail(lineno, "phase %s: count %d, want %d", phase, run.count(), wantCount)
			}
			if run.capacity() != wantCap {
				fail(lineno, "phase %s: capacity %d, want %d", phase, run.capacity(), wantCap)
			}
			if run.capacity()-run.count() != wantAvail {
				fail(lineno, "phase %s: available %d, want %d", phase, run.capacity()-run.count(), wantAvail)
			}
		case "K":
			// K <key> <hash> <slot> <control byte> <value>
			if run == nil || phase == "" || len(f) != 6 {
				t.Fatalf("line %d: bad line %q", lineno, line)
			}
			kLines++
			key := uint32(atoi(f[1]))
			hash, err := strconv.ParseUint(f[2], 16, 64)
			if err != nil {
				t.Fatalf("line %d: bad hash %q", lineno, f[2])
			}
			ctl, err := strconv.ParseUint(f[4], 16, 8)
			if err != nil {
				t.Fatalf("line %d: bad control byte %q", lineno, f[4])
			}
			if got := hmCtx.Hash(hmK(key)); got != hash {
				fail(lineno, "hash(%d) = %016x, want %016x", key, got, hash)
			}
			slot, ok := run.slotOf(key)
			if !ok {
				fail(lineno, "key %d is not in the map", key)
				continue
			}
			if uint64(slot) != atoi(f[3]) {
				fail(lineno, "slot of key %d is %d, want %s", key, slot, f[3])
			}
			if got := run.ctl(slot); uint64(got) != ctl {
				fail(lineno, "control byte of key %d is %02x, want %02x", key, got, ctl)
			}
			if got, _ := run.valueOf(key); got != f[5] {
				fail(lineno, "value of key %d is %s, want %s", key, got, f[5])
			}
		case "T":
			// T <n> <slots>
			if run == nil || phase == "" || len(f) < 2 {
				t.Fatalf("line %d: bad line %q", lineno, line)
			}
			sawT = true
			got := run.tombstones()
			if uint64(len(got)) != atoi(f[1]) || len(f) != 2+len(got) {
				fail(lineno, "phase %s: %d tombstones, the file has %s (%d slots listed)", phase, len(got), f[1], len(f)-2)
				continue
			}
			for j, s := range got {
				if uint64(s) != atoi(f[2+j]) {
					fail(lineno, "phase %s: tombstone %d is slot %d, want %s", phase, j, s, f[2+j])
					break
				}
			}
		case "I":
			// I <n> <slot> <key>
			if run == nil || phase == "" || len(f) != 4 {
				t.Fatalf("line %d: bad line %q", lineno, line)
			}
			if !iterStarted {
				run.startIter()
				iterStarted = true
			}
			if uint64(iLines) != atoi(f[1]) {
				fail(lineno, "iterator line %s out of order, expected %d", f[1], iLines)
			}
			iLines++
			slot, key, ok := run.next()
			if !ok {
				fail(lineno, "the iterator ended before entry %s", f[1])
				continue
			}
			if uint64(slot) != atoi(f[2]) || uint64(key) != atoi(f[3]) {
				fail(lineno, "iterator entry %s is slot %d key %d, want slot %s key %s", f[1], slot, key, f[2], f[3])
			}
		default:
			t.Fatalf("line %d: bad line %q", lineno, line)
		}
	}
	finishPhase(len(lines))

	// 0..3000 one by one, then every 7th up to 70000; 3 scenarios of 600
	// puts, 200 removes and 200 puts, with 3 phases each.
	if nL != 12572 {
		t.Errorf("got %d L lines, want 12572", nL)
	}
	if nScen != 3 || nPhases != 9 || nOps != 3000 {
		t.Errorf("got %d scenarios, %d phases and %d operations, want 3, 9 and 3000", nScen, nPhases, nOps)
	}
}
