// Tests translated from Zig 0.15.2's lib/std/hash/wyhash.zig (the toolchain
// that built the recorded corpus), one Go test per Zig test, each with the
// exact Zig title in a "// Zig: test" comment. scripts/check-test-mapping.py
// does not check this file: it maps Ghostty source files, not std files.
//
// Translation notes:
//   - The helpers of lib/std/hash/verify.zig that the tests call are ported
//     here as wy_smhasher (verify.smhasher) and wy_iterativeApi
//     (verify.iterativeApi), for Wyhash only (hashMaybeSeed and
//     initMaybeSeed pick the seeded forms `hash(seed, buf)` and
//     `init(seed)`, which is what they resolve to for Wyhash).
//   - Go has no comptime evaluation. "test vectors at comptime" runs the
//     same checks at run time, and the `comptime Test.do()` halves of
//     "smhasher" and "iterative api" are a second run-time call. Both are
//     kept so the test list matches Zig's.
//
// The Go-only tests at the end compare the port with values printed by Zig
// 0.15.2's std.hash.Wyhash on seeded inputs (see
// docs/mapping-pending/p3-small.md for the cross-check).

package terminal

import (
	"encoding/binary"
	"errors"
	"testing"
)

// wy_TestVector is wyhash.zig's TestVector.
type wy_TestVector struct {
	expected uint64
	seed     uint64
	input    string
}

// wy_vectors is wyhash.zig's `vectors`.
//
// Run https://github.com/wangyi-fudan/wyhash/blob/77e50f267fbc7b8e2d09f2d455219adb70ad4749/test_vector.cpp directly.
var wy_vectors = []wy_TestVector{
	{seed: 0, expected: 0x409638ee2bde459, input: ""},
	{seed: 1, expected: 0xa8412d091b5fe0a9, input: "a"},
	{seed: 2, expected: 0x32dd92e4b2915153, input: "abc"},
	{seed: 3, expected: 0x8619124089a3a16b, input: "message digest"},
	{seed: 4, expected: 0x7a43afb61d7f5f40, input: "abcdefghijklmnopqrstuvwxyz"},
	{seed: 5, expected: 0xff42329b90e50d58, input: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"},
	{seed: 6, expected: 0xc39cab13b115aad3, input: "12345678901234567890123456789012345678901234567890123456789012345678901234567890"},
}

// wy_smhasher is verify.smhasher(Wyhash.hash): returns a verification
// code, the same as used by SMHasher.
//
// Hash keys of the form {0}, {0,1}, {0,1,2}... up to N=255, using 256-N as seed.
// First four-bytes of the hash, interpreted as little-endian is the verification code.
func wy_smhasher() uint32 {
	const hash_size = 8 // @sizeOf(u64)

	var buf [256]uint8
	var buf_all [256 * hash_size]uint8

	for i := 0; i < 256; i++ {
		buf[i] = uint8(i)
		h := stdHashWyhashHash(uint64(256-i), buf[0:i])
		binary.LittleEndian.PutUint64(buf_all[i*hash_size:][:hash_size], h)
	}

	return uint32(stdHashWyhashHash(0, buf_all[0:]))
}

var (
	wy_errIterativeHashWasNotIdempotent  = errors.New("IterativeHashWasNotIdempotent")
	wy_errIterativeHashDidNotMatchDirect = errors.New("IterativeHashDidNotMatchDirect")
)

// wy_iterativeApi is verify.iterativeApi(Wyhash).
func wy_iterativeApi() error {
	// Sum(1..32) = 528
	var buf [528]uint8
	len_ := 0
	const seed = 0

	hasher := stdHashWyhashInit(seed)
	for i := 1; i < 32; i++ {
		r := stdHashWyhashHash(seed, buf[0:len_+i])
		hasher.update(buf[len_:][:i])
		f1 := hasher.final()
		f2 := hasher.final()
		if f1 != f2 {
			return wy_errIterativeHashWasNotIdempotent
		}
		if f1 != r {
			return wy_errIterativeHashDidNotMatchDirect
		}
		len_ += i
	}
	return nil
}

// Zig: test "test vectors" (lib/std/hash/wyhash.zig)
func TestStdHashWyhash_test_vectors(t *testing.T) {
	for _, e := range wy_vectors {
		expectEqual(t, e.expected, stdHashWyhashHash(e.seed, []uint8(e.input)))
	}
}

// Zig: test "test vectors at comptime" (lib/std/hash/wyhash.zig)
func TestStdHashWyhash_test_vectors_at_comptime(t *testing.T) {
	// Zig: comptime { ... }. Run at run time (see the file header).
	for _, e := range wy_vectors {
		expectEqual(t, e.expected, stdHashWyhashHash(e.seed, []uint8(e.input)))
	}
}

// Zig: test "smhasher" (lib/std/hash/wyhash.zig)
func TestStdHashWyhash_smhasher(t *testing.T) {
	do := func() {
		t.Helper()
		expectEqual(t, uint32(0xBD5E840C), wy_smhasher())
	}
	do()
	// Zig: @setEvalBranchQuota(50000); try comptime Test.do();
	do()
}

// Zig: test "iterative api" (lib/std/hash/wyhash.zig)
func TestStdHashWyhash_iterative_api(t *testing.T) {
	do := func() {
		t.Helper()
		if err := wy_iterativeApi(); err != nil {
			t.Fatalf("verify.iterativeApi(Wyhash): error.%s", err)
		}
	}
	do()
	// Zig: @setEvalBranchQuota(50000); try comptime Test.do();
	do()
}

// Zig: test "iterative maintains last sixteen" (lib/std/hash/wyhash.zig)
func TestStdHashWyhash_iterative_maintains_last_sixteen(t *testing.T) {
	input := []uint8("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ" + "01234567890abcdefg") // "Z" ** 48 ++ ...
	expectEqual(t, 48+18, len(input))
	const seed = 0

	for i := 0; i < 17; i++ {
		payload := input[0 : len(input)-i]
		non_iterative_hash := stdHashWyhashHash(seed, payload)

		wh := stdHashWyhashInit(seed)
		wh.update(payload)
		iterative_hash := wh.final()

		expectEqual(t, non_iterative_hash, iterative_hash)
	}
}

// ---------------------------------------------------------------------------
// Go-only tests: cross-check against Zig 0.15.2's std.hash.Wyhash.

// wy_splitmix64 is the generator of the seeded cross-check inputs
// (SplitMix64; the Zig program that produced the pinned digests uses the
// same generator).
type wy_splitmix64 struct{ s uint64 }

func (self *wy_splitmix64) next() uint64 {
	self.s += 0x9e3779b97f4a7c15
	z := self.s
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// wy_crossCheckDigest hashes `count` seeded inputs of length 0..300 three
// ways (Wyhash.hash, one update + final, and updates in random chunks with
// a final after every chunk) and folds every result into one Wyhash, so a
// single number pins all of them. It fails the test as soon as the three
// ways disagree.
func wy_crossCheckDigest(t *testing.T, count int) uint64 {
	t.Helper()
	rng := wy_splitmix64{s: 0x5eed}
	digest := stdHashWyhashInit(0)
	var buf [300]uint8
	var word [8]uint8
	fold := func(h uint64) {
		binary.LittleEndian.PutUint64(word[:], h)
		digest.update(word[:])
	}
	for n := 0; n < count; n++ {
		seed := rng.next()
		length := int(rng.next() % 301)
		for i := 0; i < length; i++ {
			buf[i] = uint8(rng.next())
		}
		input := buf[0:length]

		h1 := stdHashWyhashHash(seed, input)

		w := stdHashWyhashInit(seed)
		w.update(input)
		h2 := w.final()

		w = stdHashWyhashInit(seed)
		pos := 0
		for pos < length {
			chunk := int(rng.next() % 64)
			if chunk > length-pos {
				chunk = length - pos
			}
			w.update(input[pos : pos+chunk])
			pos += chunk
			fold(w.final())
		}
		h3 := w.final()

		if h1 != h2 || h1 != h3 {
			t.Fatalf("input %d (seed %#x, len %d): hash %#x, update %#x, chunked %#x", n, seed, length, h1, h2, h3)
		}
		fold(h1)
	}
	return digest.final()
}

// Go-only. The digest was computed by the same procedure with Zig 0.15.2's
// std.hash.Wyhash (aarch64 and arm, ReleaseSafe and Debug).
func TestStdHashWyhash_go_only_seeded_inputs_match_zig(t *testing.T) {
	expectEqual(t, uint64(wy_crossCheckDigestZig), wy_crossCheckDigest(t, 5000))
}

// wy_crossCheckDigestZig is the digest Zig 0.15.2 prints for
// wy_crossCheckDigest(5000).
const wy_crossCheckDigestZig = 0x314f82606e8f4b36
