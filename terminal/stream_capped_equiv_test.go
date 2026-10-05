package terminal

import (
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"
)

// Ruling (project lead, 2026-10-05): nextSliceCapped's defined
// result is what per-byte `next` gives. Upstream's scalar fallback
// (simd/vt.zig utf8DecodeUntilControlSeqScalar) is not ported (mapping rule
// 8); its stand-in streamUtf8DecodeUntilControlSeq must keep nextSliceCapped
// identical to the per-byte path. This test enforces that on seeded random
// inputs fed in random chunks: same printed codepoints, same action log, same
// parser state and same UTF-8 decoder state.
//
// TERM51_CAPPED_SEED=<n> replays one seed; TERM51_CAPPED_N=<n> changes the
// number of inputs (default 30,000).

// strCapEq_alphabet: ESC and the bytes that build CSI/OSC/DCS sequences, C0
// controls, ASCII, invalid UTF-8 bytes, lead bytes of every length,
// continuation bytes, and the starts of surrogates (ED A0), overlongs
// (E0 80, F0 80, C0/C1) and values above U+10FFFF (F4 90, F5).
var strCapEq_alphabet = []uint8{
	0x1B, '[', ']', 'P', '\\', '1', ';', 'A', 'm', '?', '0', 'h',
	0x07, 0x00, 0x08, 0x0A, 0x0D, 0x18, 0x1A, 'a', ' ', '(',
	0xFF, 0xFE, 0xC0, 0xC1, 0xF5, 0xF8,
	0xC2, 0xDF, 0xE0, 0xE2, 0xED, 0xEF, 0xF0, 0xF4,
	0x80, 0x82, 0x8F, 0x90, 0x9F, 0xA0, 0xBF,
}

// strCapEq_input builds one input of 0-23 bytes and its chunk boundaries.
func strCapEq_input(r *rand.Rand) ([]uint8, []int) {
	n := r.Intn(24)
	input := make([]uint8, n)
	for i := range input {
		input[i] = strCapEq_alphabet[r.Intn(len(strCapEq_alphabet))]
	}
	var chunks []int
	for i := 0; i < n; {
		c := 1 + r.Intn(n-i)
		chunks = append(chunks, c)
		i += c
	}
	return input, chunks
}

// strCapEq_parserState is the parser without its OSC parser, whose fixed
// writer points into its own buffer (so two parsers never compare equal);
// the OSC parser's state is compared separately and its results reach the
// action log.
func strCapEq_parserState(p *Parser) Parser {
	c := *p
	c.osc_parser = OSCParser{}
	return c
}

func strCapEq_one(t *testing.T, seed int64) bool {
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
		// Alternate between a 4096-entry buffer and one exactly as long as
		// the chunk (the smallest nextSliceCapped accepts).
		var cp_buf []uint32
		if r.Intn(2) == 0 {
			cp_buf = make([]uint32, 4096)
		} else {
			cp_buf = make([]uint32, c)
		}
		if err := b.nextSliceCapped(input[off:off+c], cp_buf); err != nil {
			t.Errorf("seed %d: nextSliceCapped: %v", seed, err)
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
		t.Errorf("seed %d (TERM51_CAPPED_SEED=%d): input % X chunks %v\n"+
			"  per byte: cps %X decoder %+v parser state %v\n%s"+
			"  capped:   cps %X decoder %+v parser state %v\n%s",
			seed, seed, input, chunks,
			recA.cps, a.utf8decoder, a.parser.state, strA_recText(&recA),
			recB.cps, b.utf8decoder, b.parser.state, strA_recText(&recB))
	}
	return ok
}

// Go-only: ruling (nextSliceCapped equals per-byte next; scalar SIMD fallback not ported, mapping rule 8)
func TestStream_go_only_nextSliceCapped_equals_per_byte_next(t *testing.T) {
	if s := os.Getenv("TERM51_CAPPED_SEED"); s != "" {
		seed, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("TERM51_CAPPED_SEED: %v", err)
		}
		strCapEq_one(t, seed)
		return
	}
	n := 30000
	if s := os.Getenv("TERM51_CAPPED_N"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("TERM51_CAPPED_N: %v", err)
		}
		n = v
	}
	failures := 0
	for seed := int64(1); seed <= int64(n); seed++ {
		if !strCapEq_one(t, seed) {
			failures++
			if failures >= 5 {
				t.Fatalf("stopping after %d mismatches", failures)
			}
		}
	}
}
