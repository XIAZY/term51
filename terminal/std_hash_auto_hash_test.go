// Tests for std_hash_auto_hash.go, the port of the reached part of Zig
// 0.15.2's lib/std/hash/auto_hash.zig. scripts/check-test-mapping.py does
// not check this file: it maps Ghostty source files, not std files.
//
// None of auto_hash.zig's own tests can be translated: each one hashes key
// types whose instantiations the library never reaches and which are not
// ported (std_hash_auto_hash.go keeps only u8, u32, usize, u21, an enum(u1),
// an enum(c_int), a packed struct(u16) hashed shallowly and []const u8):
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
// Zig 0.15.2: the PageEntry.hash ones (u8, u32, usize, enum(u1), []const u8)
// on aarch64 (64-bit usize, the recorded data's width), the
// KeyEvent.bindingHash ones (enum(c_int), u21, packed struct(u16)) on
// aarch64 and on 32-bit ARM, which agree (no usize is hashed). The
// assertions and the not-ported strategies are checked to panic. See
// docs/mapping.md, section "Zig std hashing: `std.hash.Wyhash`,
// `std.hash.int`, `std.hash.autoHash`", for the cross-check.

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

// ah_bindingRow is one row printed by the Zig cross-check program for the
// KeyEvent.bindingHash instantiations: a Wyhash seed, a key (enum(c_int),
// as its integer), a u21 codepoint, a Mods (packed struct(u16), as its u16
// bits), then the results a = autoHash(key), b = autoHash(cp),
// c = autoHash(mods), each alone from Wyhash.init(seed);
// d = autoHash of key, cp, mods, cp, key in turn from Wyhash.init(seed);
// e = KeyEvent.bindingHash (seed 0, mods.binding()).
type ah_bindingRow struct {
	seed       uint64
	key        int32
	cp         uint32
	mods       uint16
	a, b, c, d uint64
	e          uint64
}

// ah_bindingRows are the first five rows printed by Zig 0.15.2 (aarch64
// and arm-linux-musleabihf give the same values). Rows 0 to 2 are fixed:
// .unidentified with codepoint 0 and no mods; .paste with
// maxInt(u21) and mods 0xffff; .key_a with 0x41 and mods 0x24f3.
var ah_bindingRows = []ah_bindingRow{
	{0x161922c645ce50e8, 0, 0, 0x0000, 0x63bc323b421a716d, 0xbda27073ed41f684, 0x4fa8066b343cdc95, 0x8bfb3b1437206995, 0xda83d4271fcd766d},
	{0x66239e8c309a282b, 175, 0x1fffff, 0xffff, 0x8f33e4537b78262c, 0xa906c46566eefe37, 0x0ad26b52d54b240f, 0xd7e67864fc0852d2, 0x812652d9327a7a50},
	{0x4d7760d307367afa, 20, 0x41, 0x24f3, 0x71f98e2ac19fbf58, 0x59c9f4607afe4338, 0x50a429993f6719d9, 0xcd45d01ae9be1191, 0x5d10bb1a32970b2f},
	{0xb7dc9a7f27f0b595, 133, 0x195bce, 0xa756, 0xfbf3bfcb572246f9, 0x4e12de99dcaf23c7, 0xa9133dce4956d574, 0xaa585afa65ffaf2a, 0x6f8b20758dca359a},
	{0x19a42fc02250ff9d, 1, 0x13997b, 0x7253, 0xc89060edb03f67ff, 0x42b1eaf464a558fd, 0xd7bcec42f2a0b1ea, 0xba8bc757a8db1b42, 0xa839ec44947fcc81},
}

// ah_bindingResults computes a, b, c, d and e of ah_bindingRow for the
// given inputs with the ported instantiations and KeyEvent.BindingHash.
func ah_bindingResults(seed uint64, key int32, cp uint32, mods uint16, action InputKeyAction, consumed uint16) [5]uint64 {
	var out [5]uint64
	h := stdHashWyhashInit(seed)
	stdHashAutoHashAutoHashEnumCInt(&h, key)
	out[0] = h.final()
	h = stdHashWyhashInit(seed)
	stdHashAutoHashAutoHashU21(&h, cp)
	out[1] = h.final()
	h = stdHashWyhashInit(seed)
	stdHashAutoHashAutoHashPackedStructU16(&h, mods)
	out[2] = h.final()
	h = stdHashWyhashInit(seed)
	stdHashAutoHashAutoHashEnumCInt(&h, key)
	stdHashAutoHashAutoHashU21(&h, cp)
	stdHashAutoHashAutoHashPackedStructU16(&h, mods)
	stdHashAutoHashAutoHashU21(&h, cp)
	stdHashAutoHashAutoHashEnumCInt(&h, key)
	out[3] = h.final()
	ev := InputKeyKeyEvent{
		action:              action,
		key:                 InputKeyKey(key),
		mods:                InputKeyModsModsFromInt(mods),
		consumed_mods:       InputKeyModsModsFromInt(consumed),
		unshifted_codepoint: cp,
		utf8:                []uint8("x"),
	}
	out[4] = ev.BindingHash()
	return out
}

// Go-only. Zig 0.15.2: KeyEvent.bindingHash of (.unidentified, codepoint 0,
// mods 0), (.paste, codepoint 0x1FFFFF, mods 0xffff) and (.key_a, codepoint
// 0x41, mods 0x24f3), mods given as their u16 bits.
func TestStdHashAutoHash_go_only_bindingHash_matches_zig(t *testing.T) {
	// The key values Zig printed for the three tags.
	expectEqual(t, int32(0), int32(InputKeyKeyUnidentified))
	expectEqual(t, int32(175), int32(InputKeyKeyPaste))
	expectEqual(t, int32(20), int32(InputKeyKeyKeyA))

	ev := InputKeyKeyEvent{key: InputKeyKeyUnidentified, unshifted_codepoint: 0, mods: InputKeyModsModsFromInt(0)}
	expectEqual(t, uint64(0xda83d4271fcd766d), ev.BindingHash())
	ev = InputKeyKeyEvent{key: InputKeyKeyPaste, unshifted_codepoint: 0x1FFFFF, mods: InputKeyModsModsFromInt(0xffff)}
	expectEqual(t, uint64(0x812652d9327a7a50), ev.BindingHash())
	ev = InputKeyKeyEvent{key: InputKeyKeyKeyA, unshifted_codepoint: 0x41, mods: InputKeyModsModsFromInt(0x24f3)}
	expectEqual(t, uint64(0x5d10bb1a32970b2f), ev.BindingHash())

	// The same as the autoHash sequence bindingHash is made of: the key's 4
	// bytes, the codepoint's 3 low bytes, mods.binding()'s 2 bytes.
	expectEqual(t, stdHashWyhashHash(0, []uint8{20, 0, 0, 0, 0x41, 0, 0, 0xf3 & 0x0f, 0}), uint64(0x5d10bb1a32970b2f))
}

// Go-only. Zig 0.15.2 (aarch64 and arm): std.hash.autoHash of an enum(c_int)
// (input.key.Key), a u21 and a packed struct(u16) (input.key_mods.Mods), each
// alone and in sequence, and KeyEvent.bindingHash, on five fixed rows and
// on 800 SplitMix64 rows (the five results of every row folded into one
// Wyhash, seed 0, 8 little-endian bytes per result).
func TestStdHashAutoHash_go_only_bindingHash_instantiations_match_zig(t *testing.T) {
	for i, row := range ah_bindingRows {
		got := ah_bindingResults(row.seed, row.key, row.cp, row.mods, InputKeyActionPress, 0)
		want := [5]uint64{row.a, row.b, row.c, row.d, row.e}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("row %d result %d: expected 0x%x, found 0x%x", i, j, want[j], got[j])
			}
		}
	}

	// The Zig program's generator and draws, in its order: seed, key
	// (mod the number of Key tags), codepoint (@truncate to u21), mods and
	// consumed mods (@truncate to u16), action (mod 3). Rows 0 to 2 replace
	// key, codepoint and mods (row 2: key and codepoint) by the fixed values.
	expectEqual(t, 176, int(inputKeyKeyCount))
	rng := wy_splitmix64{s: 0x123456789abcdef0}
	digest := stdHashWyhashInit(0)
	var word [8]uint8
	for i := 0; i < 800; i++ {
		seed := rng.next()
		key := int32(rng.next() % uint64(inputKeyKeyCount))
		cp := uint32(rng.next()) & 0x1FFFFF
		mods := uint16(rng.next())
		consumed := uint16(rng.next())
		action := InputKeyAction(rng.next() % 3)
		switch i {
		case 0:
			key, cp, mods = int32(InputKeyKeyUnidentified), 0, 0
		case 1:
			key, cp, mods = int32(InputKeyKeyPaste), 0x1FFFFF, 0xffff
		case 2:
			key, cp = int32(InputKeyKeyKeyA), 0x41
		}
		if i < len(ah_bindingRows) {
			row := ah_bindingRows[i]
			expectEqual(t, row.seed, seed)
			expectEqual(t, row.key, key)
			expectEqual(t, row.cp, cp)
			expectEqual(t, row.mods, mods)
		}
		for _, v := range ah_bindingResults(seed, key, cp, mods, action, consumed) {
			binary.LittleEndian.PutUint64(word[:], v)
			digest.update(word[:])
		}
	}
	expectEqual(t, uint64(0x980e85effae8addf), digest.final())
}

// Go-only. A u21 holds at most 0x1FFFFF.
func TestStdHashAutoHash_go_only_u21_out_of_range_panics(t *testing.T) {
	h := stdHashWyhashInit(0)
	stdHashAutoHashAutoHashU21(&h, 0x1FFFFF) // the largest u21 does not panic
	defer func() {
		if recover() == nil {
			t.Fatalf("expected a panic")
		}
	}()
	h = stdHashWyhashInit(0)
	stdHashAutoHashAutoHashU21(&h, 0x200000)
}

// Go-only. Hashing a packed struct(u16) with .Deep or .DeepRecursive hashes
// it field by field in Zig and is not ported.
func TestStdHashAutoHash_go_only_packed_struct_u16_deep_panics(t *testing.T) {
	for _, strat := range []stdHashAutoHashHashStrategy{stdHashAutoHashHashStrategyDeep, stdHashAutoHashHashStrategyDeepRecursive} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected a panic with %v", strat)
				}
			}()
			h := stdHashWyhashInit(0)
			stdHashAutoHashHashPackedStructU16(&h, 0x24f3, strat)
		}()
	}
}
