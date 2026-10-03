package terminal

// Data-comparison tests for bitmap_allocator.go against the recorded corpus
// files data/bitmap/bitmap_layout.txt and data/bitmap/bitmap_trace.txt,
// written by the Zig dump test in bitmap_allocator.zig (`term51:
// BitmapAllocator.layout: dump exact data when TERM51_BITMAP_DUMP_DIR is
// set`, not translated). The files are plain text.
//
// PROVISIONAL: option A (see mapping doc); the tests stand if the choice is reversed only for the parts that do not read byte offsets
//
// bitmap_layout.txt has, for the chunk sizes 16 (grapheme) and 32 (string),
// a header `# chunk_size=<c>: ...` and then one line per capacity of a set
// of capacities in 0..70000 (every cap up to 3000, then every 7th, plus every
// cap within 3 of a multiple of 64 * chunk_size):
//
//	l <chunk> <cap> <total_size> <bitmap_count> <bitmap_start> <chunks_start> <bytesRequired(u8,cap)> <bytesRequired(u21,cap)>
//
// bitmap_trace.txt has four traces, each introduced by
//
//	# trace chunk_size=<c> cap=<cap> seed=<seed> bitmap_count=<n> chunks_start=<n> total_size=<n>
//
// and 300 operations on a fresh allocator of u8 (Layout.init(cap)):
//
//	<op> a <size> null <used>
//	<op> a <size> <chunk_offset> <base_offset> <used>
//	<op> f <i> <size> <chunk_offset> <base_offset> <used>
//
// `a` allocates <size> bytes (null: out of memory), `f` frees live
// allocation i (swap-remove: the last live allocation takes its place);
// <used> is usedBytes after the operation. chunk_offset is in bytes from the
// chunk area, base_offset in bytes from the start of the Zig buffer, which is
// chunks_start further (the Go allocator has no buffer: base_offset is
// checked as chunks_start + chunk_offset). The operations are drawn from
// Xoshiro256(seed): for each of 300 ops, draw uintLessThan(u8,100); if live
// allocations exist and the draw is < 40 free live[i] with i =
// uintLessThan(usize, live_len), else alloc intRangeAtMost(usize,1,200)
// bytes.

import (
	"strconv"
	"strings"
	"testing"
)

// bmCorpusNum parses a decimal field of a corpus line.
func bmCorpusNum(t *testing.T, s string) uint64 {
	t.Helper()
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		t.Fatalf("bad number %q: %v", s, err)
	}
	return v
}

// bmCorpusKV parses the key=value fields of a header line.
func bmCorpusKV(t *testing.T, fields []string) map[string]uint64 {
	t.Helper()
	m := map[string]uint64{}
	for _, f := range fields {
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			continue
		}
		m[k] = bmCorpusNum(t, v)
	}
	return m
}

// bmDumpCap is term51WDumpCap: whether the dump records this capacity.
func bmDumpCap(cap_ uint64, chunk_size uint64) bool {
	if cap_ <= 3000 {
		return true
	}
	if cap_%7 == 0 {
		return true
	}
	period := 64 * chunk_size
	r := cap_ % period
	return r <= 3 || period-r <= 3
}

// bmDumpNextCap returns the first recorded capacity >= from, or 70001 when
// there is none.
func bmDumpNextCap(from uint64, chunk_size uint64) uint64 {
	for c := from; c <= 70000; c++ {
		if bmDumpCap(c, chunk_size) {
			return c
		}
	}
	return 70001
}

func TestBitmapAllocator_corpus_layout(t *testing.T) {
	lines := bmCorpusLines(t, "data/bitmap/bitmap_layout.txt")

	var chunks []uint64 // the chunk sizes in the order of the headers
	var chunk_size uint64
	var next uint64 // the next capacity expected for chunk_size
	finish := func() {
		if len(chunks) > 0 && next <= 70000 {
			t.Fatalf("chunk size %d: lines stop before capacity %d", chunk_size, next)
		}
	}
	nl, bad := 0, 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			finish()
			f := strings.Fields(strings.TrimPrefix(line, "#"))
			if len(f) == 0 || !strings.HasPrefix(f[0], "chunk_size=") {
				t.Fatalf("bad header %q", line)
			}
			chunk_size = bmCorpusNum(t, strings.TrimSuffix(strings.TrimPrefix(f[0], "chunk_size="), ":"))
			chunks = append(chunks, chunk_size)
			next = bmDumpNextCap(0, chunk_size)
			continue
		}
		f := strings.Fields(line)
		if len(f) != 9 || f[0] != "l" {
			t.Fatalf("bad line %q", line)
		}
		chunk := bmCorpusNum(t, f[1])
		cap_ := bmCorpusNum(t, f[2])
		total_size := bmCorpusNum(t, f[3])
		bitmap_count := bmCorpusNum(t, f[4])
		bitmap_start := bmCorpusNum(t, f[5])
		chunks_start := bmCorpusNum(t, f[6])
		br8 := bmCorpusNum(t, f[7])
		br21 := bmCorpusNum(t, f[8])
		nl++

		// Every recorded capacity, in order, and no other.
		if chunk != chunk_size || cap_ != next {
			t.Fatalf("line %q: expected chunk %d cap %d", line, chunk_size, next)
		}
		next = bmDumpNextCap(cap_+1, chunk_size)

		l := BitmapAllocatorLayoutOf(chunk, cap_)
		if l.total_size != total_size || uint64(l.bitmap_count) != bitmap_count ||
			l.bitmap_start != bitmap_start || l.chunks_start != chunks_start {
			t.Errorf("layout(%d, %d) = total %d count %d bitmap_start %d chunks_start %d, want %d %d %d %d",
				chunk, cap_, l.total_size, l.bitmap_count, l.bitmap_start, l.chunks_start,
				total_size, bitmap_count, bitmap_start, chunks_start)
			bad++
		}
		if got := BitmapAllocatorBytesRequired[uint8](chunk, uint(cap_)); got != br8 {
			t.Errorf("bytesRequired(u8, %d) with chunk %d = %d, want %d", cap_, chunk, got, br8)
			bad++
		}
		// u21 is uint32 (@sizeOf 4).
		if got := BitmapAllocatorBytesRequired[uint32](chunk, uint(cap_)); got != br21 {
			t.Errorf("bytesRequired(u21, %d) with chunk %d = %d, want %d", cap_, chunk, got, br21)
			bad++
		}
		if bad >= 20 {
			t.Fatal("too many differences")
		}
	}
	finish()
	if len(chunks) != 2 || chunks[0] != 16 || chunks[1] != 32 {
		t.Fatalf("got the chunk sizes %v, want [16 32]", chunks)
	}
	if nl == 0 {
		t.Fatal("no layout lines")
	}
}

// bmTraceHeader is one `# trace ...` header of bitmap_trace.txt.
type bmTraceHeader struct {
	chunk_size, cap_, seed, bitmap_count, chunks_start, total_size uint64
}

// bmTraces reads bitmap_trace.txt: the header of each trace and the lines of
// its operations.
func bmTraces(t *testing.T) ([]bmTraceHeader, [][]string) {
	t.Helper()
	var headers []bmTraceHeader
	var ops [][]string
	for _, line := range bmCorpusLines(t, "data/bitmap/bitmap_trace.txt") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			f := strings.Fields(line)
			if len(f) < 3 || f[1] != "trace" {
				t.Fatalf("bad header %q", line)
			}
			kv := bmCorpusKV(t, f[2:])
			headers = append(headers, bmTraceHeader{
				chunk_size:   kv["chunk_size"],
				cap_:         kv["cap"],
				seed:         kv["seed"],
				bitmap_count: kv["bitmap_count"],
				chunks_start: kv["chunks_start"],
				total_size:   kv["total_size"],
			})
			ops = append(ops, nil)
			continue
		}
		if len(ops) == 0 {
			t.Fatalf("operation before the first header: %q", line)
		}
		ops[len(ops)-1] = append(ops[len(ops)-1], line)
	}
	// The four traces of the dump test: chunk sizes 16 and 32, capacities
	// 2048 and 8192, seeds 0x7e570001..4, 300 operations each.
	want := []bmTraceHeader{
		{chunk_size: 16, cap_: 2048, seed: 0x7e57_0001},
		{chunk_size: 16, cap_: 8192, seed: 0x7e57_0002},
		{chunk_size: 32, cap_: 2048, seed: 0x7e57_0003},
		{chunk_size: 32, cap_: 8192, seed: 0x7e57_0004},
	}
	if len(headers) != len(want) {
		t.Fatalf("got %d traces, want %d", len(headers), len(want))
	}
	for i, w := range want {
		h := headers[i]
		if h.chunk_size != w.chunk_size || h.cap_ != w.cap_ || h.seed != w.seed {
			t.Fatalf("trace %d: header %+v, want chunk_size %d cap %d seed %d",
				i, h, w.chunk_size, w.cap_, w.seed)
		}
		if len(ops[i]) != 300 {
			t.Fatalf("trace %d: %d operations, want 300", i, len(ops[i]))
		}
	}
	return headers, ops
}

// bmCorpusLines is corpusLines for the plain-text files of data/bitmap.
func bmCorpusLines(t *testing.T, rel string) []string {
	t.Helper()
	return corpusLines(t, rel)
}

func TestBitmapAllocator_corpus_trace(t *testing.T) {
	headers, traces := bmTraces(t)
	for ti, h := range headers {
		fx := bmFixtureInit[uint8](t, h.chunk_size, h.cap_)
		// The header records Layout.init(cap).
		if uint64(fx.layout.bitmap_count) != h.bitmap_count || fx.layout.chunks_start != h.chunks_start ||
			fx.layout.total_size != h.total_size {
			t.Fatalf("trace %d: layout count %d chunks_start %d total_size %d, want %d %d %d", ti,
				fx.layout.bitmap_count, fx.layout.chunks_start, fx.layout.total_size,
				h.bitmap_count, h.chunks_start, h.total_size)
		}

		var live [][]uint8
		for op, line := range traces[ti] {
			f := strings.Fields(line)
			if len(f) < 2 || bmCorpusNum(t, f[0]) != uint64(op) {
				t.Fatalf("trace %d: bad line %q (op %d)", ti, line, op)
			}
			switch f[1] {
			case "a":
				// <op> a <size> null <used> or
				// <op> a <size> <chunk_offset> <base_offset> <used>
				if len(f) != 5 && len(f) != 6 {
					t.Fatalf("trace %d: bad line %q", ti, line)
				}
				nbytes := bmCorpusNum(t, f[2])
				s, err := fx.allocErr(uint(nbytes))
				if f[3] == "null" {
					if len(f) != 5 {
						t.Fatalf("trace %d: bad line %q", ti, line)
					}
					if err != ErrOutOfMemory {
						t.Fatalf("trace %d op %d: alloc(%d) = %v, want out of memory", ti, op, nbytes, err)
					}
					if got, want := fx.usedBytes(), bmCorpusNum(t, f[4]); got != want {
						t.Fatalf("trace %d op %d: used %d, want %d", ti, op, got, want)
					}
					continue
				}
				if len(f) != 6 {
					t.Fatalf("trace %d: bad line %q", ti, line)
				}
				if err != nil {
					t.Fatalf("trace %d op %d: alloc(%d) failed: %v", ti, op, nbytes, err)
				}
				if uint64(len(s)) != nbytes {
					t.Fatalf("trace %d op %d: alloc(%d) has length %d", ti, op, nbytes, len(s))
				}
				chunk_off := uint64(fx.chunkIndex(s)) * h.chunk_size
				if want := bmCorpusNum(t, f[3]); chunk_off != want {
					t.Fatalf("trace %d op %d: chunk offset %d, want %d", ti, op, chunk_off, want)
				}
				if want := bmCorpusNum(t, f[4]); h.chunks_start+chunk_off != want {
					t.Fatalf("trace %d op %d: base offset %d, want %d", ti, op, h.chunks_start+chunk_off, want)
				}
				if got, want := fx.usedBytes(), bmCorpusNum(t, f[5]); got != want {
					t.Fatalf("trace %d op %d: used %d, want %d", ti, op, got, want)
				}
				live = append(live, s)
			case "f":
				// <op> f <i> <size> <chunk_offset> <base_offset> <used>
				if len(f) != 7 {
					t.Fatalf("trace %d: bad line %q", ti, line)
				}
				i := bmCorpusNum(t, f[2])
				if i >= uint64(len(live)) {
					t.Fatalf("trace %d op %d: free of live[%d] with %d live", ti, op, i, len(live))
				}
				s := live[i]
				live[i] = live[len(live)-1]
				live = live[:len(live)-1]
				if uint64(len(s)) != bmCorpusNum(t, f[3]) {
					t.Fatalf("trace %d op %d: live[%d] has length %d, want %s", ti, op, i, len(s), f[3])
				}
				chunk_off := uint64(fx.chunkIndex(s)) * h.chunk_size
				if want := bmCorpusNum(t, f[4]); chunk_off != want {
					t.Fatalf("trace %d op %d: chunk offset %d, want %d", ti, op, chunk_off, want)
				}
				if want := bmCorpusNum(t, f[5]); h.chunks_start+chunk_off != want {
					t.Fatalf("trace %d op %d: base offset %d, want %d", ti, op, h.chunks_start+chunk_off, want)
				}
				fx.free(s)
				if got, want := fx.usedBytes(), bmCorpusNum(t, f[6]); got != want {
					t.Fatalf("trace %d op %d: used %d, want %d", ti, op, got, want)
				}
			default:
				t.Fatalf("trace %d: bad line %q", ti, line)
			}
		}
	}
}

// Not a test of the port: the operations of each recorded trace are the ones
// Zig's Xoshiro256(seed) draws, which checks bmXoshiro (used by the random
// findFreeChunks test) against Zig's generator.
func TestBitmapAllocator_corpus_trace_ops_match_xoshiro(t *testing.T) {
	headers, traces := bmTraces(t)
	for ti, h := range headers {
		prng := bmXoshiroInit(h.seed)
		r := &prng
		var live []uint64 // sizes of the live allocations
		for op, line := range traces[ti] {
			f := strings.Fields(line)
			pick := r.uintLessThanU8(100)
			if len(live) > 0 && pick < 40 {
				i := r.uintLessThanU64(uint64(len(live)))
				if f[1] != "f" || bmCorpusNum(t, f[2]) != i || bmCorpusNum(t, f[3]) != live[i] {
					t.Fatalf("trace %d op %d: line %q, the generator draws a free of live[%d] (size %d)",
						ti, op, line, i, live[i])
				}
				live[i] = live[len(live)-1]
				live = live[:len(live)-1]
			} else {
				nbytes := r.intRangeAtMostUsize(1, 200)
				if f[1] != "a" || bmCorpusNum(t, f[2]) != nbytes {
					t.Fatalf("trace %d op %d: line %q, the generator draws an alloc of %d", ti, op, line, nbytes)
				}
				if f[3] != "null" {
					live = append(live, nbytes)
				}
			}
		}
	}
}
