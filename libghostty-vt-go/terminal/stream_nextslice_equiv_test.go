package terminal

import (
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"
)

// NextSlice prints a printable ASCII byte directly when the parser is in the
// ground state and no UTF-8 sequence is open (stream.go). Its defined result
// is what per-byte `next` gives. This test enforces that on seeded random
// inputs fed in random chunks: same printed codepoints, same action log, same parser state and same
// UTF-8 decoder state. The inputs are those of the nextSliceCapped test
// (stream_capped_equiv_test.go): escape sequences, C0 controls, ASCII, and
// valid, truncated and invalid UTF-8, so that a chunk often ends inside a
// sequence.
//
// TERM51_SLICE_SEED=<n> replays one seed; TERM51_SLICE_N=<n> changes the
// number of inputs (default 30,000).

func strSliceEq_one(t *testing.T, seed int64) bool {
	t.Helper()
	r := rand.New(rand.NewSource(seed))
	input, chunks := strCapEq_input(r)

	recA := strA_recorderInit(false)
	a := StreamInit(&recA)
	defer a.Deinit()
	for _, c := range input {
		if err := a.Next(c); err != nil {
			t.Errorf("seed %d: next: %v", seed, err)
			return false
		}
	}

	recB := strA_recorderInit(false)
	b := StreamInit(&recB)
	defer b.Deinit()
	off := 0
	for _, c := range chunks {
		if err := b.NextSlice(input[off : off+c]); err != nil {
			t.Errorf("seed %d: NextSlice: %v", seed, err)
			return false
		}
		off += c
	}

	ok := reflect.DeepEqual(recA.cps, recB.cps) &&
		strA_recText(&recA) == strA_recText(&recB) &&
		a.utf8decoder == b.utf8decoder &&
		a.parser.osc_parser.state == b.parser.osc_parser.state &&
		reflect.DeepEqual(strCapEq_parserState(&a.parser), strCapEq_parserState(&b.parser))
	if !ok {
		t.Errorf("seed %d (TERM51_SLICE_SEED=%d): input % X chunks %v\n"+
			"  per byte:  cps %X decoder %+v parser state %v\n%s"+
			"  NextSlice: cps %X decoder %+v parser state %v\n%s",
			seed, seed, input, chunks,
			recA.cps, a.utf8decoder, a.parser.state, strA_recText(&recA),
			recB.cps, b.utf8decoder, b.parser.state, strA_recText(&recB))
	}
	return ok
}

// Go-only: ruling (NextSlice's direct print of printable ASCII equals per-byte next)
func TestStream_go_only_NextSlice_equals_per_byte_next(t *testing.T) {
	if s := os.Getenv("TERM51_SLICE_SEED"); s != "" {
		seed, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("TERM51_SLICE_SEED: %v", err)
		}
		strSliceEq_one(t, seed)
		return
	}
	n := 30000
	if s := os.Getenv("TERM51_SLICE_N"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("TERM51_SLICE_N: %v", err)
		}
		n = v
	}
	failures := 0
	for seed := int64(1); seed <= int64(n); seed++ {
		if !strSliceEq_one(t, seed) {
			failures++
			if failures >= 5 {
				t.Fatalf("stopping after %d mismatches", failures)
			}
		}
	}
}

// Go-only: ruling (NextSlice's direct print is not taken inside a UTF-8 sequence or outside the ground state)
func TestStream_go_only_NextSlice_direct_print_guard(t *testing.T) {
	feed := func(chunks ...string) *strA_recorder {
		rec := strA_recorderInit(false)
		s := StreamInit(&rec)
		defer s.Deinit()
		for _, c := range chunks {
			if err := s.NextSlice([]uint8(c)); err != nil {
				t.Fatal(err)
			}
		}
		return &rec
	}

	// An open UTF-8 sequence: 'a' after the lead byte E4 is not a
	// continuation byte, so the sequence is replaced by U+FFFD and 'a' is
	// printed after it. Printing 'a' directly would lose the U+FFFD.
	for _, chunks := range [][]string{{"\xE4a"}, {"\xE4", "a"}} {
		rec := feed(chunks...)
		if want := []uint32{0xFFFD, 'a'}; !reflect.DeepEqual(rec.cps, want) {
			t.Errorf("%q: printed %X, want %X", chunks, rec.cps, want)
		}
	}

	// Outside the ground state: the bytes after ESC belong to the
	// sequence (CSI 5 A, cursor up) and are not printed.
	for _, chunks := range [][]string{{"\x1b[5A"}, {"\x1b", "[5A"}, {"\x1b[", "5", "A"}} {
		rec := feed(chunks...)
		if len(rec.cps) != 0 {
			t.Errorf("%q: printed %X, want nothing", chunks, rec.cps)
		}
		if got := strA_recText(rec); got == "" {
			t.Errorf("%q: no action recorded", chunks)
		}
	}

	// In the ground state with no sequence open, the bytes are printed.
	rec := feed("ab", "~\x7f")
	if want := []uint32{'a', 'b', '~', 0x7F}; !reflect.DeepEqual(rec.cps, want) {
		t.Errorf("printed %X, want %X", rec.cps, want)
	}
}
