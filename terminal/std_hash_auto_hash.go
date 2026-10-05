// Port of the part of Zig 0.15.2's lib/std/hash/auto_hash.zig (the toolchain
// that built the recorded corpus) that the library reaches. Nothing here is
// ported from a Ghostty source file. See docs/mapping-pending/p3-small.md.
//
// auto_hash.zig's `hash`, `hashPointer`, `hashArray` and `autoHash` are
// generic over the key type (`anytype`) and dispatch on it at compile time
// (`@typeInfo`, std.meta.hasUniqueRepresentation). Go has no such
// dispatch, so every instantiation the library reaches is written out as its
// own function, named after the Zig function plus the key type (as the F64
// suffix of std_fmt_parse_float.go). Each one keeps the branches of the Zig
// function that its key type takes; the comptime decisions are comments.
//
// Instantiations reached (hyperlink.PageEntry.hash):
//
//	autoHash(&hasher, std.meta.activeTag(self.id))   stdHashAutoHashAutoHashEnumU1
//	  hash(hasher, <enum(u1)>, .Shallow)             stdHashAutoHashHashEnumU1
//	autoHash(&hasher, v)  (v: u32)                   stdHashAutoHashAutoHashU32
//	  hash(hasher, <u32>, .Shallow)                  stdHashAutoHashHashU32
//	autoHashStrat(&hasher, slice, .Deep)             stdHashAutoHashHashSliceU8
//	  (std.hash.autoHashStrat is auto_hash.hash; slice: []const u8)
//	  hashPointer(hasher, <[]const u8>, .Deep)       stdHashAutoHashHashPointerSliceU8
//	  hashArray(hasher, <[]const u8>, .Shallow)      stdHashAutoHashHashArraySliceU8
//	  hash(hasher, <u8>, .Shallow)                   stdHashAutoHashHashU8
//	  hash(hasher, <usize>, .Shallow)                stdHashAutoHashHashUsize
//
// hash_map.AutoContext does not reach this file: every key type it is used
// with has a unique representation, so std.hash_map.getAutoHashFn calls
// Wyhash.hash(0, std.mem.asBytes(&key)) directly (stdHashWyhashHash).
//
// The hasher. Zig's `hasher: anytype` is always a *std.hash.Wyhash in the
// library, so the parameter is *stdHashWyhash; `Hasher.update` is its
// `update` method.
//
// Bytes. `std.mem.asBytes(&key)` is the key's memory in native byte order,
// which is little-endian on every port target (arm64, 32-bit ARM) and on the
// 64-bit build that recorded the data; the Go code writes little-endian
// explicitly. usize is 8 bytes (the recorded data come from a 64-bit build):
// stdHashAutoHashHashUsize hashes 8 bytes on every platform, including
// 32-bit ARM where Zig's own usize, and so its hash, would be 4 bytes.
//
// Not ported: the instantiations for other types (pointers hashed by
// address, optionals, arrays, vectors, structs, unions, error unions,
// floats), `typeContainsSlice` (a compile-time check that only makes
// autoHash refuse slices; none of the instantiations above contains one).

package terminal

import "encoding/binary"

// stdHashAutoHashHashStrategy is auto_hash.HashStrategy (std.hash.Strategy).
//
// Describes how pointer types should be hashed.
type stdHashAutoHashHashStrategy uint8

const (
	// Do not follow pointers, only hash their value.
	stdHashAutoHashHashStrategyShallow stdHashAutoHashHashStrategy = iota

	// Follow pointers, hash the pointee content.
	// Only dereferences one level, ie. it is changed into .Shallow when a
	// pointer type is encountered.
	stdHashAutoHashHashStrategyDeep

	// Follow pointers, hash the pointee content.
	// Dereferences all pointers encountered.
	// Assumes no cycle.
	stdHashAutoHashHashStrategyDeepRecursive
)

// stdHashAutoHashHashStrategyNames holds the Zig tag names, indexed by value.
var stdHashAutoHashHashStrategyNames = [...]string{
	"Shallow",
	"Deep",
	"DeepRecursive",
}

// String returns the Zig tag name (@tagName).
func (self stdHashAutoHashHashStrategy) String() string {
	if int(self) >= len(stdHashAutoHashHashStrategyNames) {
		panic("unreachable")
	}
	return stdHashAutoHashHashStrategyNames[self]
}

// stdHashAutoHashHashPointerSliceU8 is auto_hash.hashPointer(hasher, key,
// strat) with @TypeOf(key) == []const u8.
//
// Helper function to hash a pointer and mutate the strategy if needed.
func stdHashAutoHashHashPointerSliceU8(hasher *stdHashWyhash, key []uint8, strat stdHashAutoHashHashStrategy) {
	// Zig: switch (info.pointer.size): a []const u8 is `.slice`; the `.one`,
	// `.many` and `.c` prongs belong to other instantiations.
	switch strat {
	case stdHashAutoHashHashStrategyShallow:
		// Zig: hashPointer(hasher, key.ptr, .Shallow), which hashes the
		// address (@intFromPtr). Go has no address of a slice without
		// `unsafe`, and the library never hashes a slice shallowly.
		panic("not ported: auto_hash.hashPointer of a slice with .Shallow (hashes the address)")
	case stdHashAutoHashHashStrategyDeep:
		stdHashAutoHashHashArraySliceU8(hasher, key, stdHashAutoHashHashStrategyShallow)
	case stdHashAutoHashHashStrategyDeepRecursive:
		stdHashAutoHashHashArraySliceU8(hasher, key, stdHashAutoHashHashStrategyDeepRecursive)
	default:
		panic("unreachable")
	}
	stdHashAutoHashHashUsize(hasher, uint(len(key)), stdHashAutoHashHashStrategyShallow)
}

// stdHashAutoHashHashArraySliceU8 is auto_hash.hashArray(hasher, key,
// strat) with @TypeOf(key) == []const u8.
//
// Helper function to hash a set of contiguous objects, from an array or slice.
func stdHashAutoHashHashArraySliceU8(hasher *stdHashWyhash, key []uint8, strat stdHashAutoHashHashStrategy) {
	for _, element := range key {
		stdHashAutoHashHashU8(hasher, element, strat)
	}
}

// stdHashAutoHashHashU8 is auto_hash.hash(hasher, key, strat) with
// @TypeOf(key) == u8.
//
// Provides generic hashing for any eligible type.
// Strategy is provided to determine if pointers should be followed or not.
func stdHashAutoHashHashU8(hasher *stdHashWyhash, key uint8, strat stdHashAutoHashHashStrategy) {
	// Zig: std.meta.hasUniqueRepresentation(u8) is true (8 bits in 1 byte).
	bytes := [1]uint8{key} // std.mem.asBytes(&key)
	if strat == stdHashAutoHashHashStrategyShallow {
		hasher.update(bytes[:])
		return
	}

	// Zig: switch (@typeInfo(Key)) { .int => unsigned, unique representation }.
	hasher.update(bytes[:])
}

// stdHashAutoHashHashU32 is auto_hash.hash(hasher, key, strat) with
// @TypeOf(key) == u32.
//
// Provides generic hashing for any eligible type.
// Strategy is provided to determine if pointers should be followed or not.
func stdHashAutoHashHashU32(hasher *stdHashWyhash, key uint32, strat stdHashAutoHashHashStrategy) {
	// Zig: std.meta.hasUniqueRepresentation(u32) is true (32 bits in 4 bytes).
	var bytes [4]uint8 // std.mem.asBytes(&key)
	binary.LittleEndian.PutUint32(bytes[:], key)
	if strat == stdHashAutoHashHashStrategyShallow {
		hasher.update(bytes[:])
		return
	}

	// Zig: switch (@typeInfo(Key)) { .int => unsigned, unique representation }.
	hasher.update(bytes[:])
}

// stdHashAutoHashHashUsize is auto_hash.hash(hasher, key, strat) with
// @TypeOf(key) == usize, hashed as the 8 bytes of a 64-bit usize on every
// platform (see the file header).
//
// Provides generic hashing for any eligible type.
// Strategy is provided to determine if pointers should be followed or not.
func stdHashAutoHashHashUsize(hasher *stdHashWyhash, key uint, strat stdHashAutoHashHashStrategy) {
	// Zig: std.meta.hasUniqueRepresentation(usize) is true (64 bits in 8 bytes).
	var bytes [8]uint8 // std.mem.asBytes(&key)
	binary.LittleEndian.PutUint64(bytes[:], uint64(key))
	if strat == stdHashAutoHashHashStrategyShallow {
		hasher.update(bytes[:])
		return
	}

	// Zig: switch (@typeInfo(Key)) { .int => unsigned, unique representation }.
	hasher.update(bytes[:])
}

// stdHashAutoHashHashEnumU1 is auto_hash.hash(hasher, key, strat) with
// @TypeOf(key) an enum whose tag type is u1 (the tag of a two-field
// `union(enum)`, such as hyperlink.PageEntry.Id). key is @intFromEnum of
// the value, 0 or 1.
//
// Provides generic hashing for any eligible type.
// Strategy is provided to determine if pointers should be followed or not.
func stdHashAutoHashHashEnumU1(hasher *stdHashWyhash, key uint8, strat stdHashAutoHashHashStrategy) {
	// A u1 value is 0 or 1; Zig cannot hold anything else in the enum.
	assert(key <= 1)

	// Zig: std.meta.hasUniqueRepresentation is true for every enum, so the
	// enum's memory (@sizeOf is 1: the u1 in one byte, upper bits zero) is
	// hashed.
	bytes := [1]uint8{key} // std.mem.asBytes(&key)
	if strat == stdHashAutoHashHashStrategyShallow {
		hasher.update(bytes[:])
		return
	}

	// Zig: switch (@typeInfo(Key)) { .@"enum" => hash(hasher,
	// @intFromEnum(key), strat) }: the u1 instantiation, an unsigned integer
	// without a unique representation (1 bit in 1 byte), hashes
	// std.mem.asBytes(&key)[0..byte_size] with byte_size = divCeil(1, 8) = 1:
	// the same byte.
	hasher.update(bytes[0:1])
}

// stdHashAutoHashHashSliceU8 is auto_hash.hash(hasher, key, strat) with
// @TypeOf(key) == []const u8. std.hash.autoHashStrat is auto_hash.hash, so
// this is also `std.hash.autoHashStrat(hasher, key, strat)` for a
// []const u8.
//
// Provides generic hashing for any eligible type.
// Strategy is provided to determine if pointers should be followed or not.
func stdHashAutoHashHashSliceU8(hasher *stdHashWyhash, key []uint8, strat stdHashAutoHashHashStrategy) {
	// Zig: `if (strat == .Shallow and std.meta.hasUniqueRepresentation(Key))`
	// is false: hasUniqueRepresentation is false for a slice.
	//
	// Zig: switch (@typeInfo(Key)) { .pointer => hashPointer(hasher, key, strat) }.
	stdHashAutoHashHashPointerSliceU8(hasher, key, strat)
}

// stdHashAutoHashAutoHashEnumU1 is auto_hash.autoHash(hasher, key)
// (std.hash.autoHash) with @TypeOf(key) an enum whose tag type is u1; key is
// @intFromEnum of the value.
//
// Provides generic hashing for any eligible type.
// Only hashes `key` itself, pointers are not followed.
// Slices as well as unions and structs containing slices are rejected to avoid
// ambiguity on the user's intention.
func stdHashAutoHashAutoHashEnumU1(hasher *stdHashWyhash, key uint8) {
	// Zig: `comptime typeContainsSlice(Key)` is false for an enum.
	stdHashAutoHashHashEnumU1(hasher, key, stdHashAutoHashHashStrategyShallow)
}

// stdHashAutoHashAutoHashU32 is auto_hash.autoHash(hasher, key)
// (std.hash.autoHash) with @TypeOf(key) == u32.
//
// Provides generic hashing for any eligible type.
// Only hashes `key` itself, pointers are not followed.
// Slices as well as unions and structs containing slices are rejected to avoid
// ambiguity on the user's intention.
func stdHashAutoHashAutoHashU32(hasher *stdHashWyhash, key uint32) {
	// Zig: `comptime typeContainsSlice(Key)` is false for an integer.
	stdHashAutoHashHashU32(hasher, key, stdHashAutoHashHashStrategyShallow)
}
