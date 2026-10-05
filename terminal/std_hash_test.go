// Tests translated from Zig 0.15.2's lib/std/hash.zig (the toolchain that
// built the recorded corpus), one Go test per Zig test, each with the exact
// Zig title in a "// Zig: test" comment. scripts/check-test-mapping.py does
// not check this file: it maps Ghostty source files, not std files.
//
// Only the u64 instantiation of std.hash.int is ported (stdHashIntU64,
// see std_hash.go), so `test int` keeps its u64 line only. Its lines for
// u1, u2, u3, u8, u16, i16, u24, u32, i32 and i64 call other
// instantiations and are not translated. The unnamed `test { _ = ...; }`
// block only references the sub-files for the Zig test runner; it has no
// assertions and is not translated.
//
// The Go-only test at the end compares the port with values printed by Zig
// 0.15.2's std.hash.int(u64) (see docs/mapping-pending/p3-small.md).

package terminal

import (
	"encoding/binary"
	"testing"
)

// Zig: test int (lib/std/hash.zig)
func TestStdHash_int(t *testing.T) {
	// Not translated (other instantiations, see the file header):
	//   expectEqual(0x1, int(@as(u1, 1)))
	//   expectEqual(0x3, int(@as(u2, 1)))
	//   expectEqual(0x4, int(@as(u3, 1)))
	//   expectEqual(0xD6, int(@as(u8, 1)))
	//   expectEqual(0x2880, int(@as(u16, 1)))
	//   expectEqual(0x2880, int(@as(i16, 1)))
	//   expectEqual(0x838380, int(@as(u24, 1)))
	//   expectEqual(0x42741D6, int(@as(u32, 1)))
	//   expectEqual(0x42741D6, int(@as(i32, 1)))
	expectEqual(t, uint64(0x71894DE00D9981F), stdHashIntU64(uint64(1)))
	//   expectEqual(0x71894DE00D9981F, int(@as(i64, 1)))
}

// Go-only. Values and digest printed by Zig 0.15.2's std.hash.int(u64)
// (aarch64 and arm, Debug and ReleaseSafe): 5000 SplitMix64 values x, each
// hashed as x, x & 0xffff and x << 40, every result folded into one Wyhash
// (seed 0, 8 little-endian bytes per result).
func TestStdHash_go_only_int_u64_matches_zig(t *testing.T) {
	expectEqual(t, uint64(0x0), stdHashIntU64(0))
	expectEqual(t, uint64(0x96c7cbb7179e89f6), stdHashIntU64(0xffffffffffffffff))
	expectEqual(t, uint64(0x5f7ab967f16dcf2f), stdHashIntU64(0xdeadbeef))

	rng := wy_splitmix64{s: 0x1A7}
	digest := stdHashWyhashInit(0)
	var word [8]uint8
	fold := func(h uint64) {
		binary.LittleEndian.PutUint64(word[:], h)
		digest.update(word[:])
	}
	for n := 0; n < 5000; n++ {
		x := rng.next()
		fold(stdHashIntU64(x))
		fold(stdHashIntU64(x & 0xffff))
		fold(stdHashIntU64(x << 40))
	}
	expectEqual(t, uint64(0xf6a99ebbb747d413), digest.final())
}
