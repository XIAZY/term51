// Port of the part of Zig 0.15.2's lib/std/hash.zig (the toolchain that
// built the recorded corpus) that the library reaches: `std.hash.int` for a
// u64, which style.Style.hash calls. Nothing here is ported from a Ghostty
// source file. See docs/mapping-pending/p3-small.md.
//
// `int` is generic over its integer type (`anytype`) and picks the mixer by
// the bit width at compile time. Only the u64 instantiation is used, so it is
// written out as stdHashIntU64 (suffix U64, as std_fmt_parse_float.go's F64)
// together with the one private mixer it reaches, `uint64`
// (stdHashUint64). `uint16`, `uint32` and the generic `else` prong (widths
// without a dedicated mixer) are other instantiations and not ported.
//
// The rest of hash.zig only re-exports other files: `autoHash` and
// `autoHashStrat` are std_hash_auto_hash.go's stdHashAutoHashAutoHash... and
// stdHashAutoHashHash... functions, `Wyhash` is std_hash_wyhash.go's
// stdHashWyhash. The other hash functions it re-exports (Adler32, crc, Fnv,
// SipHash, Murmur, CityHash, XxHash) are not used by the library.
//
// Go's `*` on uint64 wraps, as Zig's `*%`.

package terminal

// stdHashIntU64 is std.hash.int(input) with @TypeOf(input) == u64.
//
// Integer-to-integer hashing for bit widths <= 256.
func stdHashIntU64(input uint64) uint64 {
	// This function is only intended for integer types
	// Zig: info = @typeInfo(u64).int; bits = 64.
	// Convert input to unsigned integer (easier to deal with)
	// Zig: Uint is u64, so the @bitCast is the identity.
	u_input := input
	// Zig: `if (bits > 256) @compileError(...)`: 64 <= 256.
	//
	// For bit widths that don't have a dedicated function, use a heuristic
	// construction with a multiplier suited to diffusion -
	// a mod 2^bits where a^2 - 46 * a + 1 = 0 mod 2^(bits + 4),
	// on Mathematica: bits = 256; BaseForm[Solve[1 - 46 a + a^2 == 0, a, Modulus -> 2^(bits + 4)][[-1]][[1]][[2]], 16]
	// Zig: `mult` is only read by the `0...2` and `else` prongs; for u64 the
	// switch below takes the `64` prong, so it is not declared here.
	//
	// The bit width of the input integer determines how to hash it
	// Zig: switch (bits) { 0...2 => ..., 16 => uint16(u_input),
	// 32 => uint32(u_input), 64 => uint64(u_input), else => ... }.
	output := stdHashUint64(u_input)
	// Zig: @bitCast to u64, the identity.
	return output
}

// stdHashUint64 is hash.zig's private `uint64`.
//
// Source: https://github.com/jonmaiga/mx3
func stdHashUint64(input uint64) uint64 {
	x := input
	const c = 0xbea225f9eb34556d
	x = (x ^ (x >> 32)) * c
	x = (x ^ (x >> 29)) * c
	x = (x ^ (x >> 32)) * c
	x = x ^ (x >> 29)
	return x
}
