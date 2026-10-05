// Tests for std_hash_auto_hash.go, the port of the reached part of Zig
// 0.15.2's lib/std/hash/auto_hash.zig. scripts/check-test-mapping.py does
// not check this file: it maps Ghostty source files, not std files.
//
// None of auto_hash.zig's own tests can be translated: each one hashes key
// types whose instantiations the library never reaches and which are not
// ported (std_hash_auto_hash.go keeps only u8, u32, usize, an enum(u1) and
// []const u8):
//
//	"typeContainsSlice"                 compile-time type checks (comptime block)
//	"hash pointer"                      *const u32 (Shallow hashes the address)
//	"hash slice shallow"                []u32, *[6]u32, hashed by address
//	"hash slice deep"                   []u32 and *[6]u32
//	"hash struct deep"                  struct { u32, u16, *bool }
//	"testHash optional"                 ?u32
//	"testHash array"                    [3]u32
//	"testHash multi-dimensional array"  [2][]const u32
//	"testHash struct"                   struct { u32, u32, u32 }
//	"testHash union"                    union(enum) { u32, bool, u32, void }
//	"testHash vector"                   @Vector(4, u32), @Vector(4, u31)
//	"testHash error union"              error{Test}!struct
//
// The Go-only tests below pin every ported instantiation to values printed by
// Zig 0.15.2 on aarch64 (64-bit usize, the recorded data's width); see
// docs/mapping-pending/p3-small.md for the cross-check.

package terminal

import (
	"encoding/binary"
	"testing"
)

// ah_testHashSlice is auto_hash.zig's testHashDeep (and
// testHashDeepRecursive) for a []const u8 key: Wyhash seed 0,
// hash(&hasher, key, strat), final.
func ah_testHashSlice(key []uint8, strat stdHashAutoHashHashStrategy) uint64 {
	hasher := stdHashWyhashInit(0)
	stdHashAutoHashHashSliceU8(&hasher, key, strat)
	return hasher.final()
}

// ah_pageEntryHash is the hashing sequence of hyperlink.PageEntry.hash on a
// plain id and uri (no page memory), as written in the Zig cross-check
// program: autoHash of the tag, autoHash of the implicit u32 or autoHashStrat
// .Deep of the explicit slice, then autoHashStrat .Deep of the uri.
func ah_pageEntryHash(explicit bool, id []uint8, implicit uint32, uri []uint8) uint64 {
	hasher := stdHashWyhashInit(0)
	if explicit {
		stdHashAutoHashAutoHashEnumU1(&hasher, 0)
		stdHashAutoHashHashSliceU8(&hasher, id, stdHashAutoHashHashStrategyDeep)
	} else {
		stdHashAutoHashAutoHashEnumU1(&hasher, 1)
		stdHashAutoHashAutoHashU32(&hasher, implicit)
	}
	stdHashAutoHashHashSliceU8(&hasher, uri, stdHashAutoHashHashStrategyDeep)
	return hasher.final()
}

// Go-only. Zig 0.15.2: Wyhash.init(0); std.hash.autoHash(&h, @as(u32,
// 0x12345678)); h.final().
func TestStdHashAutoHash_go_only_autoHash_u32(t *testing.T) {
	h := stdHashWyhashInit(0)
	stdHashAutoHashAutoHashU32(&h, 0x12345678)
	expectEqual(t, uint64(0x1a5c7e44722ad15), h.final())
	// autoHash of a u32 is its 4 little-endian bytes.
	expectEqual(t, stdHashWyhashHash(0, []uint8{0x78, 0x56, 0x34, 0x12}), uint64(0x1a5c7e44722ad15))
}

// Go-only. Zig 0.15.2: std.hash.autoHash of both values of an enum(u1).
func TestStdHashAutoHash_go_only_autoHash_enum_u1(t *testing.T) {
	h := stdHashWyhashInit(0)
	stdHashAutoHashAutoHashEnumU1(&h, 0)
	expectEqual(t, uint64(0xfbe5af10e5f8bd85), h.final())
	h = stdHashWyhashInit(0)
	stdHashAutoHashAutoHashEnumU1(&h, 1)
	expectEqual(t, uint64(0xd5b26ee485f2a92d), h.final())
	// The tag is one byte.
	expectEqual(t, stdHashWyhashHash(0, []uint8{1}), uint64(0xd5b26ee485f2a92d))
}

// Go-only. A u1 holds only 0 and 1.
func TestStdHashAutoHash_go_only_enum_u1_out_of_range_panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected a panic")
		}
	}()
	h := stdHashWyhashInit(0)
	stdHashAutoHashAutoHashEnumU1(&h, 2)
}

// Go-only. Zig 0.15.2 (64-bit): std.hash.autoHash(&h, @as(usize,
// 0x05060708)). The port hashes 8 bytes on every platform.
func TestStdHashAutoHash_go_only_hash_usize_is_8_bytes(t *testing.T) {
	h := stdHashWyhashInit(0)
	stdHashAutoHashHashUsize(&h, 0x05060708, stdHashAutoHashHashStrategyShallow)
	expectEqual(t, uint64(0xe3e482a9cb2d6be8), h.final())
	expectEqual(t, stdHashWyhashHash(0, []uint8{8, 7, 6, 5, 0, 0, 0, 0}), uint64(0xe3e482a9cb2d6be8))
}

// Go-only. Zig 0.15.2 (64-bit): std.hash.autoHashStrat(&h, @as([]const u8,
// s), .Deep / .DeepRecursive).
func TestStdHashAutoHash_go_only_autoHashStrat_slice_u8(t *testing.T) {
	expectEqual(t, uint64(0x759b537a1dec968b), ah_testHashSlice([]uint8("hello"), stdHashAutoHashHashStrategyDeep))
	expectEqual(t, uint64(0x759b537a1dec968b), ah_testHashSlice([]uint8("hello"), stdHashAutoHashHashStrategyDeepRecursive))
	expectEqual(t, uint64(0xe6b7875d0487a132), ah_testHashSlice([]uint8(""), stdHashAutoHashHashStrategyDeep))
	expectEqual(t, uint64(0xe6b7875d0487a132), ah_testHashSlice(nil, stdHashAutoHashHashStrategyDeep))
	// Deep is the bytes followed by the length as 8 little-endian bytes.
	expectEqual(t, stdHashWyhashHash(0, []uint8{'h', 'e', 'l', 'l', 'o', 5, 0, 0, 0, 0, 0, 0, 0}), uint64(0x759b537a1dec968b))
}

// Go-only. Shallow hashing of a slice hashes its address in Zig and is not
// ported.
func TestStdHashAutoHash_go_only_slice_shallow_panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected a panic")
		}
	}()
	ah_testHashSlice([]uint8("hello"), stdHashAutoHashHashStrategyShallow)
}

// Go-only. Zig 0.15.2 (64-bit): the PageEntry.hash sequence on three fixed
// entries and on 5000 SplitMix64 entries (explicit id of 0..120 bytes or
// implicit u32, uri of 0..300 bytes), every result folded into one Wyhash
// (seed 0, 8 little-endian bytes per result).
func TestStdHashAutoHash_go_only_page_entry_sequence_matches_zig(t *testing.T) {
	uri := []uint8("http://example.com")
	expectEqual(t, uint64(0x263eea5e59df0fd7), ah_pageEntryHash(true, []uint8("id"), 0, uri))
	expectEqual(t, uint64(0x78d1442917c3f784), ah_pageEntryHash(false, nil, 7, uri))
	expectEqual(t, uint64(0xec264b8ab043dc8e), ah_pageEntryHash(true, nil, 0, nil))

	rng := wy_splitmix64{s: 0xA070}
	digest := stdHashWyhashInit(0)
	var word [8]uint8
	var b1 [120]uint8
	var b2 [300]uint8
	for n := 0; n < 5000; n++ {
		l1 := int(rng.next() % 121)
		for i := 0; i < l1; i++ {
			b1[i] = uint8(rng.next())
		}
		l2 := int(rng.next() % 301)
		for i := 0; i < l2; i++ {
			b2[i] = uint8(rng.next())
		}
		imp := uint32(rng.next())
		which := rng.next() & 1
		h := ah_pageEntryHash(which == 0, b1[0:l1], imp, b2[0:l2])
		binary.LittleEndian.PutUint64(word[:], h)
		digest.update(word[:])
	}
	expectEqual(t, uint64(0x118773d096266e81), digest.final())
}
