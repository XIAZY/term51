// Tests translated one for one from src/terminal/hash_map.zig (Ghostty
// v1.3.1 plus the term51 additional tests): the 22 upstream tests and the
// `term51:` tests. Each Go test carries the exact Zig title in a "// Zig: test"
// comment. Tests that expect a panic carry "// Go-only: panic test:" (Zig tests
// cannot test an assert or a safety-checked panic: the test would abort).
//
// Option A (keep upstream's byte-offset bookkeeping on Go slices): confirmed by the project owner 2026-10-04.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual keeps Zig's (expected, actual) argument order. `try
//     testing.expectError(error.OutOfMemory, call)` is hmExpectError(t,
//     ErrOutOfMemory, <the error of the call>). `try f()` of a function that
//     returns only an error is zigTry(t, f()).
//   - The Zig maps. `AutoHashMapUnmanaged(K, V)` is
//     hashMapAutoHashMapUnmanaged[K, V], made by hmInitAuto(layout), and
//     `Map.layoutForCapacity(cap)` is hashMapLayoutForCapacity(cap, sizes)
//     with the sizes of K and V in bytes (hmSizes...). The page's two maps
//     are term51XGMap = pageGraphemeMap (hmGU is its unmanaged form) and
//     term51XHMap = HyperlinkMap (hmHU); their key is Offset(Cell), the
//     number SizeOffsetInt (uint32), so term51XK(n) is hmK(n) and the key of
//     `term51XRawCtx` (a bare u32) is the same Go type.
//   - Pointers are Go pointers to elements of the map's keys and values
//     slices (hash_map.go, option A), so `@intFromPtr(&m.keys()[s]) ==
//     @intFromPtr(p)` is `&m.keys()[s] == p`.
//   - `Offset(u21).Slice` (the grapheme map's value) is a Go []uint32 (see
//     page.go). term51XSl(o, l) is hmSl(o, l): a sub-slice of one shared
//     backing array, so that "offset o, length l" can be told apart and
//     compared (hmDesc, hmExpectSl). Setting `.len` or `.offset.offset` of a
//     value replaces the slice by hmSl(...).
//   - term51XBox is hmBox: Go has no aligned byte buffer, so `box.buf` is
//     gone; `initAt(cap, at)` takes `at` and ignores it (the map has no
//     position in a larger buffer), and memory starts zeroed, not 0xAA.
//   - The control byte of a slot (`@as(u8, @bitCast(m.metadata.?[idx]))`) is
//     hmCtl: fingerprint in bits 0..6, used in bit 7.
//   - term51XRawCtx (a context that hashes like AutoContext but is passed a
//     bare u32): the `...Adapted` functions take the map's own context type
//     in Go (hash_map.go), and the raw u32 key is the same Go type as the
//     map's key, so these tests pass hashMapAutoContext, whose hash and eql
//     behave exactly like term51XRawCtx's.
//   - std.hash.Wyhash is not in Go's standard library. hmWyhash is a test-side
//     reference for inputs of up to 16 bytes (the 4 and 8 byte keys here); it
//     is checked against the 200 pinned hashes by the tests that use it.
//   - Random numbers: std.Random.Xoshiro256 / DefaultPrng and shuffle are
//     replaced by math/rand/v2 (PCG, fixed seeds). Both tests that use them
//     check the map against a model or against invariants that hold for any
//     random stream.
//   - Zig-only: `const alloc = testing.allocator;`, `alloc.alignedAlloc`,
//     `defer alloc.free(buf)`, `defer box.deinit()`, `defer keys.deinit(alloc)`
//     are allocator bookkeeping; they are dropped in every test without a
//     per-test note. Raw address and byte-offset arithmetic (`@intFromPtr`
//     sums, `.offset` fields of Offset values, reads of box.buf bytes) has no
//     Go counterpart and is dropped with a per-test `// Zig-only:` comment.
//   - A zero-capacity map made by init has an empty, non-nil metadata slice;
//     a nil metadata slice is Zig's `metadata == null`.

package terminal

import (
	"encoding/binary"
	"errors"
	"math"
	"math/bits"
	"math/rand/v2"
	"reflect"
	"testing"
)

// hmExpectPanic runs f, which must panic. A panic with the wireframe's "not
// implemented" value does not count: it only means the function under test
// does not exist yet. The setup of the map is done outside f.
func hmExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, got none")
		}
		if s, ok := r.(string); ok && s == notImplemented {
			t.Fatalf("expected a panic from the assertion, got: %v", r)
		}
	}()
	f()
}

// hmExpectError is Zig's `try testing.expectError(expected, actual)`.
func hmExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// Sizes in bytes of the key and value types of the maps the tests build:
// @sizeOf and @alignOf of K and V (a size of 0 has alignment 1).
var (
	hmSizesU32U32  = hashMapEntrySizes{key_size: 4, key_align: 4, val_size: 4, val_align: 4}
	hmSizesUsize   = hashMapEntrySizes{key_size: 8, key_align: 8, val_size: 8, val_align: 8}
	hmSizesU64U64  = hashMapEntrySizes{key_size: 8, key_align: 8, val_size: 8, val_align: 8}
	hmSizesI32U64  = hashMapEntrySizes{key_size: 4, key_align: 4, val_size: 8, val_align: 8}
	hmSizesU64Void = hashMapEntrySizes{key_size: 8, key_align: 8, val_size: 0, val_align: 1}
	hmHeaderSize   = uint64(16) // @sizeOf(Header)
	hmHeaderAlign  = uint64(4)  // @alignOf(Header)
	hmMetadataSize = uint64(1)  // @sizeOf(Metadata)
)

// hmInitAuto is `AutoHashMapUnmanaged(K, V).init(.init(buf), layout)`.
func hmInitAuto[K comparable, V any](layout hashMapLayout) hashMapAutoHashMapUnmanaged[K, V] {
	return hashMapUnmanagedInit[K, V, hashMapAutoContext[K]](layout)
}

// hmBaseAlign is Unmanaged.base_align: the maximum of the header, key and
// value alignments (a type of size 0 has alignment 1).
func hmBaseAlign(s hashMapEntrySizes) uint64 {
	key_align, val_align := s.key_align, s.val_align
	if s.key_size == 0 {
		key_align = 1
	}
	if s.val_size == 0 {
		val_align = 1
	}
	return max(hmHeaderAlign, key_align, val_align)
}

// hmAlignForward is std.mem.alignForward(u64, n, alignment).
func hmAlignForward(n uint64, alignment uint64) uint64 {
	return (n + alignment - 1) / alignment * alignment
}

// hmWyhash is std.hash.Wyhash.hash(seed, input) for inputs of up to 16
// bytes. Go's standard library has no Wyhash (final version 4, as Zig's).
func hmWyhash(seed uint64, input []byte) uint64 {
	if len(input) > 16 {
		panic("hmWyhash: only inputs of up to 16 bytes")
	}
	secret := [3]uint64{0xa0761d6478bd642f, 0xe7037ed1a0b428db, 0x8ebc6af09c88c6e3}
	mum := func(a uint64, b uint64) (uint64, uint64) {
		hi, lo := bits.Mul64(a, b)
		return lo, hi
	}
	mix := func(a uint64, b uint64) uint64 {
		lo, hi := mum(a, b)
		return lo ^ hi
	}
	r4 := func(i int) uint64 { return uint64(binary.LittleEndian.Uint32(input[i:])) }
	state := seed ^ mix(seed^secret[0], secret[1])
	var a, b uint64
	n := len(input)
	if n >= 4 {
		end := n - 4
		quarter := (n >> 3) << 2
		a = r4(0)<<32 | r4(quarter)
		b = r4(end)<<32 | r4(end-quarter)
	} else if n > 0 {
		a = uint64(input[0])<<16 | uint64(input[n>>1])<<8 | uint64(input[n-1])
	}
	a ^= secret[1]
	b ^= state
	a, b = mum(a, b)
	return mix(a^secret[0]^uint64(n), b^secret[1])
}

// hmLE32 is std.mem.asBytes(&off) of a u32 (little endian).
func hmLE32(off uint32) []byte {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], off)
	return b[:]
}

// The types of the page's two maps.
type (
	hmCtxT = hashMapAutoContext[SizeOffsetInt]
	hmGU   = hashMapUnmanaged[SizeOffsetInt, []uint32, hmCtxT]
	hmHU   = hashMapUnmanaged[SizeOffsetInt, HyperlinkId, hmCtxT]
	hmGBox = hmBox[[]uint32]
	hmHBox = hmBox[HyperlinkId]
)

// hmCtx is term51XCtx.
var hmCtx = hmCtxT{}

// hmK is term51XK: Offset(Cell){ .offset = n }. The key is the number.
func hmK(n uint32) SizeOffsetInt { return n }

// hmBacking is the array that every hmSl value is a sub-slice of.
var hmBackingArray []uint32

func hmBacking() []uint32 {
	if hmBackingArray == nil {
		hmBackingArray = make([]uint32, 1<<20+64)
	}
	return hmBackingArray
}

// hmSl is term51XSl: the slice with offset o and length l.
func hmSl(o uint32, l uint) []uint32 {
	return hmBacking()[o : uint(o)+l]
}

// hmSlv is the offset and length of a value made by hmSl.
type hmSlv struct {
	offset uint32
	len    uint
}

// hmDesc is `.offset.offset` and `.len` of a slice made by hmSl.
func hmDesc(s []uint32) hmSlv {
	return hmSlv{offset: uint32(len(hmBacking()) - cap(s)), len: uint(len(s))}
}

// hmExpectSl is `try term51XEq(want, got)` of two slice values.
func hmExpectSl(t *testing.T, want []uint32, got []uint32) {
	t.Helper()
	expectEqual(t, hmDesc(want), hmDesc(got))
}

// hmBox is term51XBox: a map as a page makes it.
type hmBox[V any] struct {
	sizes      hashMapEntrySizes
	offset_map HashMapOffsetHashMap[SizeOffsetInt, V, hmCtxT]
}

// hmBoxInit is Box.init(cap) and hmBoxInitAt is Box.initAt(cap, at). In Go
// `at` has no meaning (see the file header).
func hmBoxInit[V any](sizes hashMapEntrySizes, cap uint32) hmBox[V] {
	return hmBoxInitAt[V](sizes, cap, 0)
}

func hmBoxInitAt[V any](sizes hashMapEntrySizes, cap uint32, at uint) hmBox[V] {
	_ = at
	l := HashMapOffsetHashMapLayoutOf(cap, sizes)
	return hmBox[V]{sizes: sizes, offset_map: HashMapOffsetHashMapInit[SizeOffsetInt, V, hmCtxT](l)}
}

// map_ is Box.map (`map` is a Go keyword).
func (self *hmBox[V]) map_() hashMapUnmanaged[SizeOffsetInt, V, hmCtxT] {
	return self.offset_map.Map()
}

func hmGBoxInit(cap uint32) hmGBox {
	return hmBoxInit[[]uint32](pageGraphemeMapEntrySizes, cap)
}

func hmGBoxInitAt(cap uint32, at uint) hmGBox {
	return hmBoxInitAt[[]uint32](pageGraphemeMapEntrySizes, cap, at)
}

func hmHBoxInit(cap uint32) hmHBox {
	return hmBoxInit[HyperlinkId](hyperlinkMapEntrySizes, cap)
}

func hmHBoxInitAt(cap uint32, at uint) hmHBox {
	return hmBoxInitAt[HyperlinkId](hyperlinkMapEntrySizes, cap, at)
}

// hmGLayout is term51XGMap.layout(cap) and hmHLayout is term51XHMap.layout(cap).
func hmGLayout(cap uint32) HashMapOffsetHashMapLayout {
	return HashMapOffsetHashMapLayoutOf(cap, pageGraphemeMapEntrySizes)
}

func hmHLayout(cap uint32) HashMapOffsetHashMapLayout {
	return HashMapOffsetHashMapLayoutOf(cap, hyperlinkMapEntrySizes)
}

// hmMeta is `@as(Metadata, @bitCast(b))` and hmMetaByte is
// `@as(u8, @bitCast(m))`: fingerprint in bits 0..6, used in bit 7.
func hmMeta(b uint8) hashMapMetadata {
	return hashMapMetadata{fingerprint: b & 0x7f, used: b >> 7}
}

func hmMetaByte(m hashMapMetadata) uint8 {
	return m.fingerprint | m.used<<7
}

// hmCtl is term51XCtl: the raw control byte of a slot.
func hmCtl[V any](m hashMapUnmanaged[SizeOffsetInt, V, hmCtxT], idx uint) uint8 {
	return hmMetaByte(m.metadata[idx])
}

// hmSlot is term51XSlot: the slot index holding a key (null if absent).
func hmSlot[V any](m hashMapUnmanaged[SizeOffsetInt, V, hmCtxT], key uint32) Optional[uint] {
	return m.getIndex(hmK(key), hmCtx)
}

// hmUsedCtl is term51XUsedCtl: the expected control byte of a used slot
// holding `key`.
func hmUsedCtl(key uint32) uint8 {
	return 0x80 | uint8(hmCtx.Hash(hmK(key))>>57)
}

// hmCountCtl is term51XCountCtl: the number of slots whose control byte is
// exactly `byte`.
func hmCountCtl[V any](m hashMapUnmanaged[SizeOffsetInt, V, hmCtxT], byte uint8) uint {
	var n uint
	for i := range uint(m.Capacity()) {
		if hmCtl(m, i) == byte {
			n++
		}
	}
	return n
}

// hmCeilPowerOfTwo is std.math.ceilPowerOfTwoAssert(u32, n).
func hmCeilPowerOfTwo(n uint32) uint32 {
	assert(n > 0)
	return uint32(1) << bits.Len32(n-1)
}

// Keys whose hash & 15 is the same (found by scanning offsets 8*j with the
// Python port of Wyhash; verified against the hash here).
var hmChain5 = [8]uint32{1144, 1224, 1264, 1392, 1408, 1504, 1536, 1600}
var hmChain14 = [6]uint32{1096, 1184, 1360, 1520, 1608, 1616}

// hash & 7 == 7
var hmChain7of8 = [5]uint32{1040, 1120, 1312, 1400, 1480}

// 16 keys 65536 + 8*i that fill a capacity-16 map completely.
var hmFull16 = [16]uint32{
	65536, 65544, 65552, 65560, 65568, 65576, 65584, 65592,
	65600, 65608, 65616, 65624, 65632, 65640, 65648, 65656,
}

// Their slots when inserted in this order into an empty capacity-16 map.
var hmFull16Slots = [16]uint{7, 11, 8, 13, 2, 3, 12, 5, 6, 14, 4, 9, 15, 0, 10, 1}

// The fingerprints (hash >> 57) of those keys.
var hmFull16Fps = [16]uint8{
	0x77, 0x4f, 0x0b, 0x5a, 0x7c, 0x5b, 0x4a, 0x3d,
	0x3d, 0x61, 0x42, 0x46, 0x7e, 0x01, 0x6b, 0x75,
}

var hmHashes = [200]uint64{
	0x34c9efbe84f17de4, 0x1a95fa9658107418, 0x90cc9a1c28c31278, 0x7be171c573d70336,
	0x9934360743077031, 0x32382f24ebfbac6b, 0xd4728dbb94a866a2, 0x89c98d2770d81b01,
	0xcdb0f4b695ee38e4, 0xc15579f29ce6f3a4, 0x2f8a72a0d90247bd, 0x3299aff1d29ebdf2,
	0x781784b3cd4f8cd5, 0xd4227f5fb3b14a46, 0x19bb0eb6c882716a, 0x22ae0b46e4421e68,
	0xf269e0fe078f8b6b, 0x98d5ce30c68cfb6f, 0x1723da36de344527, 0x3ffdf78f94b2d26f,
	0x4ab59a0927ab26d8, 0xc7e8f660ed38aa21, 0x72e9b89604cb088b, 0xb19e3718bf1af173,
	0x42a847c0382e4983, 0x3e98a5671bb814a0, 0xd23fe772e0d2cf25, 0x1e7c09a3f5bc01fd,
	0x3fd4798c6918888a, 0x48766629a266a7c8, 0x91cb32ffb3d99fa9, 0x89b50bb964448022,
	0xc303e3f83cbba40f, 0xe4f85dac219979ac, 0x499abb81773667c2, 0xb87f7b8c0f0fe608,
	0xe7b473607d566be7, 0x894d0353c24eb354, 0x9ce9aa9a43c2539f, 0xe96bed30c581cfab,
	0xdb31b22812e51ae2, 0x6b31fd5a1565b949, 0x5f6234fd3beba368, 0xd70075121fedad4d,
	0x39e81a79623ee616, 0x9fe2c0c486959f7f, 0x88ebb24031c150c8, 0x5cf17ecefabdf69f,
	0xc6a29d6308d3edc5, 0xf158a9090b7a4bb1, 0x3a5e2c1c9aea4ae3, 0x287b33d435f037ef,
	0xc43fd24dfc2d65cf, 0x689fa9371c1d6aee, 0x09ffcfb3142f83f3, 0x18e709979bc7845a,
	0x883d8db62b63f9a3, 0x4c465953efda0edf, 0x75cc1d618413e1e4, 0x3d8e64fc7eac5688,
	0xddc88d7653475032, 0x2726717a4fa588e7, 0xad94ab4310c929e2, 0xca272d134175b0a2,
	0xd3132a2e8863ccb1, 0x7060aff319cf2b20, 0xfad6f071d44db968, 0x63d5876e3ea116b5,
	0xbe893af438ae444a, 0x812fcefd681b4a14, 0x8d380beeb705b0e2, 0x81d00afefc535a7d,
	0x31af54f4954f50d2, 0x62719c3e614cb01e, 0x8762f27bce294c44, 0x7747813ccd3a1c3a,
	0x2093eb658cf07597, 0x49020fc4b8fd38a2, 0xdcc761408f72e619, 0xdb2199c8590aec9d,
	0x3aa809270f63ed44, 0x0e13fbbf7ac7702a, 0xc61a440481827ced, 0x99b97856d536d1a4,
	0xb49dc001d5b25d58, 0x557d88baaf8cee23, 0x857f5caaf373bba4, 0x02f813309e8a8848,
	0x797538396a82396f, 0x57b86d20139a6a11, 0x17823fa61ca32379, 0xab0f7db575446d8c,
	0x951795e773ca30ef, 0x2c5da1d0d9a58610, 0x13d36ece5e4fab61, 0x76eb9568012c2bf2,
	0x0468ee8d4e011d66, 0x8ca0035df99bca9e, 0x41b8235070879af0, 0x8ecbaf5000f3e7b5,
	0x96086f2e5e7ad02b, 0xf34fe367766d7418, 0x5ab2502dbc48c297, 0x2a15ea6abad748e5,
	0xd0a4149f2432a4aa, 0xca611713c4979eb3, 0xa98b6a3569a08453, 0xf2394e7c31525fd3,
	0xe369fc1ff943ab07, 0xec372c736fe92928, 0xe60269acad029a0e, 0x6e7cf8ba1102ddb9,
	0x2c7c276a130d483a, 0xec56c861ccf12def, 0x34b406012467d493, 0xdfd6c05900c6e5bd,
	0x7942584f5a071732, 0xeac6e99d009ae1be, 0x4b30aec165f27c65, 0x0149886d18608cae,
	0xeab6f3ba6fbd4acd, 0x49c873fc7ae5f139, 0x30b2e8f020c1323d, 0x50592f122c01d144,
	0xbe19c6b75a3efc43, 0x3325bd4c6a948351, 0x6f55bfa496fedd4a, 0x1e1420b5a2d720a9,
	0x3cbe40f74ecf7002, 0x49b6a7184ce743c1, 0x008a83eedcdd157b, 0x2e29673c09249335,
	0xc24dbb3ccd374de2, 0x1ddd6e793cd3b705, 0x80c5364d45ab64f6, 0xc728dedef1674445,
	0xacef1a2b52806307, 0x207360ee3749d6e2, 0x43b1943e56ee0eb6, 0x45dbea2d6b16bc6e,
	0x8e2a06365bb3423e, 0x49bdb3f552d3ed48, 0x6ce2a91f7999e43f, 0x19a9a2e671983839,
	0x2379b1c5c159274b, 0x9640093d5cf6c3c6, 0x4233edc4f4ad87cf, 0x8cab71e73c8ab9f6,
	0x1b3ac7404ed724cc, 0x19ade07d1f2c12aa, 0x59c62ef52876b159, 0xdb316bf70a03169d,
	0x5b8cd8f6a97943f5, 0xecbefc143bc0af20, 0xc64bfc9b91cd069e, 0x40ad8fa79db4bdaf,
	0xfdf9ceb1f0708cc6, 0x0fc8d93cc6f41a67, 0x6dc8f6933cc12105, 0x4faba1957a8f5586,
	0x187a0a3aba21599c, 0x0f8e9152adb84fd6, 0x1e5e9fb773102aed, 0x35e0e484b46d32a4,
	0xc21c1311fa177d2e, 0x5254b5fd6e85b9b4, 0xffa3a7961cbc27b8, 0x4d3c10cb66184c58,
	0xfcddb7c769d792c3, 0x29c223516caed7b3, 0xd245bf6c5dfe7de0, 0xe34a0ac5c4327745,
	0xc4096a6ce25c51ce, 0xcfefb1d9c9d9a6be, 0x287f5f2cac160ff3, 0x5157fc964ec90190,
	0xfe694678ef930149, 0xdcc0090eb2712f4e, 0xa8e5ac9d39f7312e, 0x1f2457a2c50ec21a,
	0x1bf2b4f6b309b5db, 0x1a844e0e5a9256ee, 0x25a9b0ec37495921, 0x50d0c20903f8d40d,
	0x95c4612f4764064b, 0x1da427939ade458f, 0x5acd790a4d02f121, 0xe725dff6df17039b,
	0xc60c92c584e76f78, 0x0f12a3e83b1f4bf5, 0xae8430d9f38bdcae, 0xe916ffe1d78a8ffa,
	0x2d1381400491d83b, 0x19e6209599df432a, 0x4dac168a25cc782e, 0x4594a1eac9b65084,
	0x7ca3c77b19d8317e, 0x1afd03475f2372fd, 0x888047a4f23deb1a, 0xf96dfcc32c3b9eb1,
}

var hmSlots256 = [200]uint16{
	228, 24, 120, 54, 49, 107, 162, 1, 229, 164,
	189, 242, 213, 70, 106, 104, 108, 111, 39, 112,
	216, 33, 139, 115, 131, 160, 37, 253, 138, 200,
	169, 34, 15, 172, 194, 8, 231, 84, 159, 171,
	226, 73, 105, 77, 22, 127, 201, 161, 197, 177,
	227, 239, 207, 238, 243, 90, 163, 223, 230, 136,
	50, 232, 233, 165, 178, 32, 109, 181, 74, 20,
	234, 125, 210, 30, 68, 58, 151, 166, 25, 157,
	69, 42, 237, 167, 88, 35, 168, 72, 113, 17,
	121, 140, 240, 16, 97, 244, 102, 158, 241, 182,
	43, 26, 152, 235, 170, 179, 83, 211, 7, 40,
	14, 185, 59, 245, 147, 190, 51, 191, 101, 174,
	205, 57, 61, 71, 67, 81, 75, 173, 2, 193,
	123, 53, 236, 5, 246, 76, 9, 247, 183, 110,
	62, 78, 63, 60, 79, 198, 208, 248, 204, 175,
	89, 176, 249, 36, 180, 184, 199, 103, 6, 134,
	156, 214, 250, 186, 46, 187, 188, 91, 195, 192,
	224, 80, 206, 196, 251, 144, 82, 85, 47, 27,
	219, 252, 38, 13, 86, 143, 41, 155, 122, 254,
	202, 255, 64, 44, 48, 132, 126, 0, 28, 203,
}

// hmGLayouts: [total_size, keys_start, vals_start] for the grapheme map; entry 0 is capacity 0, entry i is capacity 2^(i-1).
var hmGLayouts = [33][3]uint64{
	{16, 0, 0},                             // cap 0
	{40, 4, 8},                             // cap 1
	{64, 4, 16},                            // cap 2
	{104, 4, 24},                           // cap 4
	{184, 8, 40},                           // cap 8
	{352, 16, 80},                          // cap 16
	{688, 32, 160},                         // cap 32
	{1360, 64, 320},                        // cap 64
	{2704, 128, 640},                       // cap 128
	{5392, 256, 1280},                      // cap 256
	{10768, 512, 2560},                     // cap 512
	{21520, 1024, 5120},                    // cap 1024
	{43024, 2048, 10240},                   // cap 2048
	{86032, 4096, 20480},                   // cap 4096
	{172048, 8192, 40960},                  // cap 8192
	{344080, 16384, 81920},                 // cap 16384
	{688144, 32768, 163840},                // cap 32768
	{1376272, 65536, 327680},               // cap 65536
	{2752528, 131072, 655360},              // cap 131072
	{5505040, 262144, 1310720},             // cap 262144
	{11010064, 524288, 2621440},            // cap 524288
	{22020112, 1048576, 5242880},           // cap 1048576
	{44040208, 2097152, 10485760},          // cap 2097152
	{88080400, 4194304, 20971520},          // cap 4194304
	{176160784, 8388608, 41943040},         // cap 8388608
	{352321552, 16777216, 83886080},        // cap 16777216
	{704643088, 33554432, 167772160},       // cap 33554432
	{1409286160, 67108864, 335544320},      // cap 67108864
	{2818572304, 134217728, 671088640},     // cap 134217728
	{5637144592, 268435456, 1342177280},    // cap 268435456
	{11274289168, 536870912, 2684354560},   // cap 536870912
	{22548578320, 1073741824, 5368709120},  // cap 1073741824
	{45097156624, 2147483648, 10737418240}, // cap 2147483648
}

// hmHLayouts is the same for the hyperlink map.
var hmHLayouts = [33][3]uint64{
	{16, 0, 0},                             // cap 0
	{28, 4, 8},                             // cap 1
	{32, 4, 12},                            // cap 2
	{44, 4, 20},                            // cap 4
	{72, 8, 40},                            // cap 8
	{128, 16, 80},                          // cap 16
	{240, 32, 160},                         // cap 32
	{464, 64, 320},                         // cap 64
	{912, 128, 640},                        // cap 128
	{1808, 256, 1280},                      // cap 256
	{3600, 512, 2560},                      // cap 512
	{7184, 1024, 5120},                     // cap 1024
	{14352, 2048, 10240},                   // cap 2048
	{28688, 4096, 20480},                   // cap 4096
	{57360, 8192, 40960},                   // cap 8192
	{114704, 16384, 81920},                 // cap 16384
	{229392, 32768, 163840},                // cap 32768
	{458768, 65536, 327680},                // cap 65536
	{917520, 131072, 655360},               // cap 131072
	{1835024, 262144, 1310720},             // cap 262144
	{3670032, 524288, 2621440},             // cap 524288
	{7340048, 1048576, 5242880},            // cap 1048576
	{14680080, 2097152, 10485760},          // cap 2097152
	{29360144, 4194304, 20971520},          // cap 4194304
	{58720272, 8388608, 41943040},          // cap 8388608
	{117440528, 16777216, 83886080},        // cap 16777216
	{234881040, 33554432, 167772160},       // cap 33554432
	{469762064, 67108864, 335544320},       // cap 67108864
	{939524112, 134217728, 671088640},      // cap 134217728
	{1879048208, 268435456, 1342177280},    // cap 268435456
	{3758096400, 536870912, 2684354560},    // cap 536870912
	{7516192784, 1073741824, 5368709120},   // cap 1073741824
	{15032385552, 2147483648, 10737418240}, // cap 2147483648
}

// ---------------------------------------------------------------------------
// Upstream tests.
// ---------------------------------------------------------------------------

// Zig: test "HashMap basic usage"
func TestHashMap_HashMap_basic_usage(t *testing.T) {
	// Zig-only: const alloc = testing.allocator;
	const cap = 16
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);

	m := hmInitAuto[uint32, uint32](layout)

	const count = 5
	var i uint32 = 0
	var total uint32 = 0
	for ; i < count; i++ {
		zigTry(t, m.Put(i, i))
		total += i
	}

	var sum uint32 = 0
	it := m.Iterator()
	for kv := it.Next(); kv.Valid; kv = it.Next() {
		sum += *kv.Value.key_ptr
	}
	expectEqual(t, total, sum)

	i = 0
	sum = 0
	for ; i < count; i++ {
		expectEqual(t, i, zigUnwrap(t, m.Get(i)))
		sum += zigUnwrap(t, m.Get(i))
	}
	expectEqual(t, total, sum)
}

// Zig: test "HashMap ensureTotalCapacity"
func TestHashMap_HashMap_ensureTotalCapacity(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[int32, int32](layout)

	initial_capacity := m.Capacity()
	expect(t, initial_capacity >= 20, "initial_capacity >= 20")
	var i int32 = 0
	for ; i < 20; i++ {
		expect(t, !m.FetchPutAssumeCapacity(i, i+10).Valid, "map.fetchPutAssumeCapacity(i, i + 10) == null")
	}
	// shouldn't resize from putAssumeCapacity
	expect(t, initial_capacity == m.Capacity(), "initial_capacity == map.capacity()")
}

// Zig: test "HashMap ensureUnusedCapacity with tombstones"
func TestHashMap_HashMap_ensureUnusedCapacity_with_tombstones(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[int32, int32](layout)

	var i int32 = 0
	for ; i < 100; i++ {
		zigTry(t, m.EnsureUnusedCapacity(1))
		m.PutAssumeCapacity(i, i)
		_ = m.Remove(i)
	}
}

// Zig: test "HashMap clearRetainingCapacity"
func TestHashMap_HashMap_clearRetainingCapacity(t *testing.T) {
	const cap = 16

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	m.ClearRetainingCapacity()

	zigTry(t, m.Put(1, 1))
	expectEqual(t, zigUnwrap(t, m.Get(1)), 1)
	expectEqual(t, m.Count(), 1)

	m.ClearRetainingCapacity()
	m.PutAssumeCapacity(1, 1)
	expectEqual(t, zigUnwrap(t, m.Get(1)), 1)
	expectEqual(t, m.Count(), 1)

	actual_cap := m.Capacity()
	expect(t, actual_cap > 0, "actual_cap > 0")

	m.ClearRetainingCapacity()
	m.ClearRetainingCapacity()
	expectEqual(t, m.Count(), 0)
	expectEqual(t, m.Capacity(), actual_cap)
	expect(t, !m.Contains(1), "!map.contains(1)")
}

// Zig: test "HashMap ensureTotalCapacity with existing elements"
func TestHashMap_HashMap_ensureTotalCapacity_with_existing_elements(t *testing.T) {
	const cap = hashMapMinimalCapacity

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	zigTry(t, m.Put(0, 0))
	expectEqual(t, m.Count(), 1)
	expectEqual(t, m.Capacity(), hashMapMinimalCapacity)

	hmExpectError(t, ErrOutOfMemory, m.EnsureTotalCapacity(65))
	expectEqual(t, m.Count(), 1)
	expectEqual(t, m.Capacity(), hashMapMinimalCapacity)
}

// Zig: test "HashMap remove"
func TestHashMap_HashMap_remove(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	var i uint32 = 0
	for ; i < 16; i++ {
		zigTry(t, m.Put(i, i))
	}

	i = 0
	for ; i < 16; i++ {
		if i%3 == 0 {
			_ = m.Remove(i)
		}
	}
	expectEqual(t, m.Count(), 10)
	it := m.Iterator()
	for kv := it.Next(); kv.Valid; kv = it.Next() {
		expectEqual(t, *kv.Value.key_ptr, *kv.Value.value_ptr)
		expect(t, *kv.Value.key_ptr%3 != 0, "kv.key_ptr.* % 3 != 0")
	}

	i = 0
	for ; i < 16; i++ {
		if i%3 == 0 {
			expect(t, !m.Contains(i), "!map.contains(i)")
		} else {
			expectEqual(t, zigUnwrap(t, m.Get(i)), i)
		}
	}
}

// Zig: test "HashMap reverse removes"
func TestHashMap_HashMap_reverse_removes(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	var i uint32 = 0
	for ; i < 16; i++ {
		zigTry(t, m.PutNoClobber(i, i))
	}

	i = 16
	for ; i > 0; i-- {
		_ = m.Remove(i - 1)
		expect(t, !m.Contains(i-1), "!map.contains(i - 1)")
		var j uint32 = 0
		for ; j < i-1; j++ {
			expectEqual(t, zigUnwrap(t, m.Get(j)), j)
		}
	}

	expectEqual(t, m.Count(), 0)
}

// Zig: test "HashMap multiple removes on same metadata"
func TestHashMap_HashMap_multiple_removes_on_same_metadata(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	var i uint32 = 0
	for ; i < 16; i++ {
		zigTry(t, m.Put(i, i))
	}

	_ = m.Remove(7)
	_ = m.Remove(15)
	_ = m.Remove(14)
	_ = m.Remove(13)
	expect(t, !m.Contains(7), "!map.contains(7)")
	expect(t, !m.Contains(15), "!map.contains(15)")
	expect(t, !m.Contains(14), "!map.contains(14)")
	expect(t, !m.Contains(13), "!map.contains(13)")

	i = 0
	for ; i < 13; i++ {
		if i == 7 {
			expect(t, !m.Contains(i), "!map.contains(i)")
		} else {
			expectEqual(t, zigUnwrap(t, m.Get(i)), i)
		}
	}

	zigTry(t, m.Put(15, 15))
	zigTry(t, m.Put(13, 13))
	zigTry(t, m.Put(14, 14))
	zigTry(t, m.Put(7, 7))
	i = 0
	for ; i < 16; i++ {
		expectEqual(t, zigUnwrap(t, m.Get(i)), i)
	}
}

// Zig: test "HashMap put and remove loop in random order"
func TestHashMap_HashMap_put_and_remove_loop_in_random_order(t *testing.T) {
	const cap = 64

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	var keys []uint32
	// Zig-only: defer keys.deinit(alloc);

	const size = 32
	const iterations = 100

	var i uint32 = 0
	for ; i < size; i++ {
		keys = append(keys, i)
	}
	// Zig: std.Random.DefaultPrng.init(0); see the file header.
	random := rand.New(rand.NewPCG(0, 0))

	for ; i < iterations; i++ {
		random.Shuffle(len(keys), func(a int, b int) { keys[a], keys[b] = keys[b], keys[a] })

		for _, key := range keys {
			zigTry(t, m.Put(key, key))
		}
		expectEqual(t, m.Count(), size)

		for _, key := range keys {
			_ = m.Remove(key)
		}
		expectEqual(t, m.Count(), 0)
	}
}

// Zig: test "HashMap put"
func TestHashMap_HashMap_put(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	var i uint32 = 0
	for ; i < 16; i++ {
		zigTry(t, m.Put(i, i))
	}

	i = 0
	for ; i < 16; i++ {
		expectEqual(t, zigUnwrap(t, m.Get(i)), i)
	}

	i = 0
	for ; i < 16; i++ {
		zigTry(t, m.Put(i, i*16+1))
	}

	i = 0
	for ; i < 16; i++ {
		expectEqual(t, zigUnwrap(t, m.Get(i)), i*16+1)
	}
}

// Zig: test "HashMap put full load"
func TestHashMap_HashMap_put_full_load(t *testing.T) {
	const cap = 16

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesUsize)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint, uint](layout)

	for i := range uint(cap) {
		zigTry(t, m.Put(i, i))
	}
	for i := range uint(cap) {
		expectEqual(t, zigUnwrap(t, m.Get(i)), i)
	}

	hmExpectError(t, ErrOutOfMemory, m.Put(cap, cap))
}

// Zig: test "HashMap putAssumeCapacity"
func TestHashMap_HashMap_putAssumeCapacity(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	var i uint32 = 0
	for ; i < 20; i++ {
		m.PutAssumeCapacityNoClobber(i, i)
	}

	i = 0
	sum := i
	for ; i < 20; i++ {
		sum += *m.GetPtr(i)
	}
	expectEqual(t, sum, 190)

	i = 0
	for ; i < 20; i++ {
		m.PutAssumeCapacity(i, 1)
	}

	i = 0
	sum = i
	for ; i < 20; i++ {
		sum += zigUnwrap(t, m.Get(i))
	}
	expectEqual(t, sum, 20)
}

// Zig: test "HashMap repeat putAssumeCapacity/remove"
func TestHashMap_HashMap_repeat_putAssumeCapacity_remove(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	const limit = cap

	var i uint32 = 0
	for ; i < limit; i++ {
		m.PutAssumeCapacityNoClobber(i, i)
	}

	// Repeatedly delete/insert an entry without resizing the map.
	// Put to different keys so entries don't land in the just-freed slot.
	i = 0
	for ; i < 10*limit; i++ {
		expect(t, m.Remove(i), "map.remove(i)")
		if i%2 == 0 {
			m.PutAssumeCapacityNoClobber(limit+i, i)
		} else {
			m.PutAssumeCapacity(limit+i, i)
		}
	}

	i = 9 * limit
	for ; i < 10*limit; i++ {
		expectEqual(t, zigUnwrap(t, m.Get(limit+i)), i)
	}
	expectEqual(t, m.Count(), limit)
}

// Zig: test "HashMap getOrPut"
func TestHashMap_HashMap_getOrPut(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint32, uint32](layout)

	var i uint32 = 0
	for ; i < 10; i++ {
		zigTry(t, m.Put(i*2, 2))
	}

	i = 0
	for ; i < 20; i++ {
		_, err := m.GetOrPutValue(i, 1)
		zigTry(t, err)
	}

	i = 0
	sum := i
	for ; i < 20; i++ {
		sum += zigUnwrap(t, m.Get(i))
	}

	expectEqual(t, sum, 30)
}

// Zig: test "HashMap basic hash map usage"
func TestHashMap_HashMap_basic_hash_map_usage(t *testing.T) {
	const cap = 32

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[int32, int32](layout)

	fetchPut := func(key int32, value int32) Optional[HashMapKV[int32, int32]] {
		t.Helper()
		r, err := m.FetchPut(key, value)
		zigTry(t, err)
		return r
	}

	expect(t, !fetchPut(1, 11).Valid, "(try map.fetchPut(1, 11)) == null")
	expect(t, !fetchPut(2, 22).Valid, "(try map.fetchPut(2, 22)) == null")
	expect(t, !fetchPut(3, 33).Valid, "(try map.fetchPut(3, 33)) == null")
	expect(t, !fetchPut(4, 44).Valid, "(try map.fetchPut(4, 44)) == null")

	zigTry(t, m.PutNoClobber(5, 55))
	expect(t, zigUnwrap(t, fetchPut(5, 66)).value == 55, "(try map.fetchPut(5, 66)).?.value == 55")
	expect(t, zigUnwrap(t, fetchPut(5, 55)).value == 66, "(try map.fetchPut(5, 55)).?.value == 66")

	gop1, err := m.GetOrPut(5)
	zigTry(t, err)
	expect(t, gop1.found_existing == true, "gop1.found_existing == true")
	expect(t, *gop1.value_ptr == 55, "gop1.value_ptr.* == 55")
	*gop1.value_ptr = 77
	expect(t, *zigUnwrap(t, m.GetEntry(5)).value_ptr == 77, "map.getEntry(5).?.value_ptr.* == 77")

	gop2, err := m.GetOrPut(99)
	zigTry(t, err)
	expect(t, gop2.found_existing == false, "gop2.found_existing == false")
	*gop2.value_ptr = 42
	expect(t, *zigUnwrap(t, m.GetEntry(99)).value_ptr == 42, "map.getEntry(99).?.value_ptr.* == 42")

	gop3, err := m.GetOrPutValue(5, 5)
	zigTry(t, err)
	expect(t, *gop3.value_ptr == 77, "gop3.value_ptr.* == 77")

	gop4, err := m.GetOrPutValue(100, 41)
	zigTry(t, err)
	expect(t, *gop4.value_ptr == 41, "gop4.value_ptr.* == 41")

	expect(t, m.Contains(2), "map.contains(2)")
	expect(t, *zigUnwrap(t, m.GetEntry(2)).value_ptr == 22, "map.getEntry(2).?.value_ptr.* == 22")
	expect(t, zigUnwrap(t, m.Get(2)) == 22, "map.get(2).? == 22")

	rmv1 := m.FetchRemove(2)
	expect(t, zigUnwrap(t, rmv1).key == 2, "rmv1.?.key == 2")
	expect(t, zigUnwrap(t, rmv1).value == 22, "rmv1.?.value == 22")
	expect(t, !m.FetchRemove(2).Valid, "map.fetchRemove(2) == null")
	expect(t, m.Remove(2) == false, "map.remove(2) == false")
	expect(t, !m.GetEntry(2).Valid, "map.getEntry(2) == null")
	expect(t, !m.Get(2).Valid, "map.get(2) == null")

	expect(t, m.Remove(3) == true, "map.remove(3) == true")
}

// Zig: test "HashMap ensureUnusedCapacity"
func TestHashMap_HashMap_ensureUnusedCapacity(t *testing.T) {
	const cap = 64

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU64U64)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint64, uint64](layout)

	zigTry(t, m.EnsureUnusedCapacity(32))
	hmExpectError(t, ErrOutOfMemory, m.EnsureUnusedCapacity(cap+1))
}

// Zig: test "HashMap removeByPtr"
func TestHashMap_HashMap_removeByPtr(t *testing.T) {
	const cap = 64

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesI32U64)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[int32, uint64](layout)

	var i int32
	i = 0
	for ; i < 10; i++ {
		zigTry(t, m.Put(i, 0))
	}

	expect(t, m.Count() == 10, "map.count() == 10")

	i = 0
	for ; i < 10; i++ {
		key_ptr := m.GetKeyPtr(i)
		expect(t, key_ptr != nil, "key_ptr != null")

		if key_ptr != nil {
			m.RemoveByPtr(key_ptr)
		}
	}

	expect(t, m.Count() == 0, "map.count() == 0")
}

// Zig: test "HashMap removeByPtr 0 sized key"
func TestHashMap_HashMap_removeByPtr_0_sized_key(t *testing.T) {
	const cap = 64

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesI32U64)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[int32, uint64](layout)

	zigTry(t, m.Put(0, 0))

	expect(t, m.Count() == 1, "map.count() == 1")

	key_ptr := m.GetKeyPtr(0)
	expect(t, key_ptr != nil, "key_ptr != null")

	if key_ptr != nil {
		m.RemoveByPtr(key_ptr)
	}

	expect(t, m.Count() == 0, "map.count() == 0")
}

// Zig: test "HashMap repeat fetchRemove"
func TestHashMap_HashMap_repeat_fetchRemove(t *testing.T) {
	const cap = 64

	// Zig-only: const alloc = testing.allocator;
	layout := hashMapLayoutForCapacity(cap, hmSizesU64Void)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, Map.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	m := hmInitAuto[uint64, struct{}](layout)

	m.PutAssumeCapacity(0, struct{}{})
	m.PutAssumeCapacity(1, struct{}{})
	m.PutAssumeCapacity(2, struct{}{})
	m.PutAssumeCapacity(3, struct{}{})

	// fetchRemove() should make slots available.
	var i uint = 0
	for ; i < 10; i++ {
		expect(t, m.FetchRemove(3).Valid, "map.fetchRemove(3) != null")
		m.PutAssumeCapacity(3, struct{}{})
	}

	expect(t, m.Get(0).Valid, "map.get(0) != null")
	expect(t, m.Get(1).Valid, "map.get(1) != null")
	expect(t, m.Get(2).Valid, "map.get(2) != null")
	expect(t, m.Get(3).Valid, "map.get(3) != null")
}

// Zig: test "OffsetHashMap basic usage"
func TestHashMap_OffsetHashMap_basic_usage(t *testing.T) {
	const cap = 16

	// Zig-only: const alloc = testing.allocator;
	layout := HashMapOffsetHashMapLayoutOf(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, OffsetMap.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	offset_map := HashMapOffsetHashMapInit[uint32, uint32, hashMapAutoContext[uint32]](layout)
	m := offset_map.Map()

	const count = 5
	var i uint32 = 0
	var total uint32 = 0
	for ; i < count; i++ {
		zigTry(t, m.Put(i, i))
		total += i
	}

	var sum uint32 = 0
	it := m.Iterator()
	for kv := it.Next(); kv.Valid; kv = it.Next() {
		sum += *kv.Value.key_ptr
	}
	expectEqual(t, total, sum)

	i = 0
	sum = 0
	for ; i < count; i++ {
		expectEqual(t, i, zigUnwrap(t, m.Get(i)))
		sum += zigUnwrap(t, m.Get(i))
	}
	expectEqual(t, total, sum)
}

// Zig: test "OffsetHashMap remake map"
func TestHashMap_OffsetHashMap_remake_map(t *testing.T) {
	const cap = 16

	// Zig-only: const alloc = testing.allocator;
	layout := HashMapOffsetHashMapLayoutOf(cap, hmSizesU32U32)
	// Zig-only: const buf = try alloc.alignedAlloc(u8, OffsetMap.base_align, layout.total_size);
	// Zig-only: defer alloc.free(buf);
	offset_map := HashMapOffsetHashMapInit[uint32, uint32, hashMapAutoContext[uint32]](layout)

	{
		m := offset_map.Map()
		zigTry(t, m.Put(5, 5))
	}

	{
		m := offset_map.Map()
		expectEqual(t, 5, zigUnwrap(t, m.Get(5)))
	}
}

// Zig: test "layoutForCapacity no overflow for large capacity"
func TestHashMap_layoutForCapacity_no_overflow_for_large_capacity(t *testing.T) {
	// Test that layoutForCapacity correctly handles large capacities without overflow.
	// Prior to the fix, new_capacity (u32) was multiplied before widening to usize,
	// causing overflow when new_capacity * @sizeOf(K) exceeded 2^32.
	// See: https://github.com/ghostty-org/ghostty/issues/9862

	// Use 2^30 capacity - this would overflow in u32 when multiplied by @sizeOf(u64)=8
	// 0x40000000 * 8 = 0x2_0000_0000 which wraps to 0 in u32
	var large_cap HashMapSize = 1 << 30
	layout := hashMapLayoutForCapacity(large_cap, hmSizesU64U64)

	// With the fix, total_size should be at least cap * (sizeof(K) + sizeof(V))
	// = 2^30 * 16 = 2^34 bytes = 16 GiB
	// Without the fix, this would wrap and produce a much smaller value.
	var min_expected uint64 = uint64(large_cap) * (8 + 8)
	expect(t, layout.total_size >= min_expected, "layout.total_size >= min_expected")

	// Also verify the individual offsets don't wrap
	expect(t, layout.keys_start > 0, "layout.keys_start > 0")
	expect(t, layout.vals_start > layout.keys_start, "layout.vals_start > layout.keys_start")
}

// ---------------------------------------------------------------------------
// term51: additional tests (X). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: AutoOffsetHashMap: instantiation, sizes and alignments"
func TestHashMap_term51_AutoOffsetHashMap_instantiation_sizes_and_alignments(t *testing.T) {
	type G = pageGraphemeMap
	type H = HyperlinkMap
	type Ctx = hashMapAutoContext[SizeOffsetInt]
	typeOf := func(v any) reflect.Type { return reflect.TypeOf(v) }

	expect(t, typeOf(G{}) == typeOf(HashMapOffsetHashMap[SizeOffsetInt, []uint32, Ctx]{}), "G == OffsetHashMap(term51XKey, term51XSlice, Ctx)")
	expect(t, reflect.TypeOf(G.Map).Out(0) == typeOf(hashMapUnmanaged[SizeOffsetInt, []uint32, Ctx]{}), "G.Unmanaged == HashMapUnmanaged(term51XKey, term51XSlice, Ctx)")
	expect(t, typeOf(H{}) == typeOf(HashMapAutoOffsetHashMap[SizeOffsetInt, HyperlinkId]{}), "H == AutoOffsetHashMap(term51XKey, u16)")
	expect(t, reflect.TypeOf(H.Map).Out(0) == typeOf(hashMapUnmanaged[SizeOffsetInt, uint16, Ctx]{}), "H.Unmanaged == HashMapUnmanaged(term51XKey, u16, Ctx)")
	expect(t, typeOf(HashMapOffsetHashMapLayout{}) == typeOf(hashMapLayout{}), "G.Layout == G.Unmanaged.Layout")

	// Key and value shapes the page relies on: the sizes the layout
	// arithmetic takes (hmGLayout, hmHLayout), key 4 and 4, slice 16 and 8.
	expectEqual(t, 4, int(pageGraphemeMapEntrySizes.key_size))
	expectEqual(t, 4, int(pageGraphemeMapEntrySizes.key_align))
	expectEqual(t, 16, int(pageGraphemeMapEntrySizes.val_size))
	expectEqual(t, 8, int(pageGraphemeMapEntrySizes.val_align))

	// The wrapper holds only the 32 bit offset of the metadata.
	// Zig-only: try term51XEq(@as(usize, 4), @sizeOf(G));
	// Zig-only: try term51XEq(@as(usize, 4), @sizeOf(H));
	// Zig: `g.metadata.offset` is 0; the Go metadata is nil.
	var g G
	expect(t, g.metadata == nil, "g.metadata.offset == 0")

	// Base alignment is the max of the header, key and value alignments.
	expectEqual(t, 8, int(hmBaseAlign(pageGraphemeMapEntrySizes)))
	expectEqual(t, 4, int(hmBaseAlign(hyperlinkMapEntrySizes)))
	expectEqual(t, 4, int(hmHeaderAlign))
	// Zig-only: try term51XEq(@as(usize, 4), @alignOf(G.Unmanaged.Header));
	// Zig: G.Unmanaged.header_align is not a Go declaration; hmHeaderAlign above.
	expectEqual(t, 4, int(pageGraphemeMapEntrySizes.key_align))
	expectEqual(t, 8, int(pageGraphemeMapEntrySizes.val_align))
	expectEqual(t, 2, int(hyperlinkMapEntrySizes.val_align))
}

// Zig: test "term51: HashMapUnmanaged: type constants, header and default state"
func TestHashMap_term51_HashMapUnmanaged_type_constants_header_and_default_state(t *testing.T) {
	type U = hashMapUnmanaged[SizeOffsetInt, []uint32, hashMapAutoContext[SizeOffsetInt]]
	expect(t, reflect.TypeFor[HashMapSize]() == reflect.TypeFor[uint32](), "U.Size == u32")
	expect(t, reflect.TypeFor[HashMapHash]() == reflect.TypeFor[uint64](), "U.Hash == u64")
	expectEqual(t, 8, hashMapMinimalCapacity)
	// Zig-only: try term51XEq(@as(usize, 16), @sizeOf(U.Header));
	// Zig-only: try term51XEq(@as(usize, 1), @sizeOf(U.Metadata));
	// Zig-only: try term51XEq(@as(usize, 1), @alignOf(U.Metadata));
	// Zig-only: try std.testing.expect(U.Metadata.FingerPrint == u7); (Go: uint8)
	expectEqual(t, 0, int(hashMapMetadataFree))
	expectEqual(t, 1, int(hashMapMetadataTombstone))
	expectEqual(t, 0, int(hashMapMetadataSlotFree))
	expectEqual(t, 1, int(hashMapMetadataSlotTombstone))

	// Header field order (values, keys, capacity, size) and types.
	fields := reflect.VisibleFields(reflect.TypeFor[hashMapHeader[SizeOffsetInt, []uint32]]())
	expectEqual(t, 4, len(fields))
	expectEqualStrings(t, "values", fields[0].Name)
	expectEqualStrings(t, "keys", fields[1].Name)
	expectEqualStrings(t, "capacity", fields[2].Name)
	expectEqualStrings(t, "size", fields[3].Name)
	expect(t, fields[2].Type == reflect.TypeFor[uint32](), "fields[2].type == u32")
	expect(t, fields[3].Type == reflect.TypeFor[uint32](), "fields[3].type == u32")

	// Layout fields.
	lf := reflect.VisibleFields(reflect.TypeFor[hashMapLayout]())
	expectEqual(t, 4, len(lf))
	expectEqualStrings(t, "total_size", lf[0].Name)
	expectEqualStrings(t, "keys_start", lf[1].Name)
	expectEqualStrings(t, "vals_start", lf[2].Name)
	expectEqualStrings(t, "capacity", lf[3].Name)

	// A default map has no metadata and reports capacity zero.
	var m U
	expect(t, m.metadata == nil, "m.metadata == null")
	expectEqual(t, 0, m.Capacity())
	// Zig-only: try term51XEq(@as(usize, 8), @sizeOf(U));
}

// Zig: test "term51: AutoContext: hash of Offset(Cell) keys is Wyhash seed 0 of the 4 key bytes"
func TestHashMap_term51_AutoContext_hash_of_Offset_Cell_keys_is_Wyhash_seed_0_of_the_4_key_bytes(t *testing.T) {
	// 200 pinned values for keys 4096 + 8 * i (computed independently with a
	// port of Wyhash; they depend on autoHash hashing the 32 bit offset as
	// four little-endian bytes with seed 0).
	expectEqual(t, len(hmHashes), 200)
	for i, expected := range hmHashes {
		off := uint32(4096 + 8*i)
		key := hmK(off)
		expectEqual(t, expected, hmCtx.Hash(key))
		expectEqual(t, expected, hmWyhash(0, hmLE32(off)))
		// std.hash.Wyhash.init(0), std.hash.autoHash(&hasher, key) and
		// hasher.final(): autoHash of the key hashes its 4 bytes.
		expectEqual(t, expected, hmWyhash(0, hmLE32(key)))
	}

	// Key zero, small and large offsets.
	expectEqual(t, uint64(0x14b83016dc460955), hmCtx.Hash(hmK(0)))
	expectEqual(t, hmWyhash(0, []byte{1, 0, 0, 0}), hmCtx.Hash(hmK(1)))
	expectEqual(t, hmWyhash(0, []byte{0, 0, 0, 1}), hmCtx.Hash(hmK(1<<24)))
	expectEqual(t, hmWyhash(0, []byte{0xff, 0xff, 0xff, 0xff}), hmCtx.Hash(hmK(math.MaxUint32)))
	expect(t, hmCtx.Hash(hmK(8)) != hmCtx.Hash(hmK(16)), "term51XCtx.hash(term51XK(8)) != term51XCtx.hash(term51XK(16))")
}

// Zig: test "term51: AutoContext: eql compares the offsets"
func TestHashMap_term51_AutoContext_eql_compares_the_offsets(t *testing.T) {
	expect(t, hmCtx.Eql(hmK(0), hmK(0)), "term51XCtx.eql(term51XK(0), term51XK(0))")
	expect(t, hmCtx.Eql(hmK(4096), hmK(4096)), "term51XCtx.eql(term51XK(4096), term51XK(4096))")
	expect(t, !hmCtx.Eql(hmK(4096), hmK(4097)), "!term51XCtx.eql(term51XK(4096), term51XK(4097))")
	expect(t, !hmCtx.Eql(hmK(0), hmK(math.MaxUint32)), "!term51XCtx.eql(term51XK(0), term51XK(maxInt(u32)))")
	expect(t, hmCtx.Eql(hmK(math.MaxUint32), hmK(math.MaxUint32)), "term51XCtx.eql(term51XK(maxInt(u32)), term51XK(maxInt(u32)))")
	// Zig-only: try term51XEq(@as(usize, 0), @sizeOf(AutoContext(term51XKey)));
}

// Zig: test "term51: AutoContext: slot index and fingerprint derived from the hash"
func TestHashMap_term51_AutoContext_slot_index_and_fingerprint_derived_from_the_hash(t *testing.T) {
	// slot = hash & (capacity - 1); fingerprint = top 7 bits. Pinned for a
	// few capacities and the first keys; chains used below depend on this.
	for _, k := range hmChain5 {
		h := hmCtx.Hash(hmK(k))
		expectEqual(t, uint64(5), h&15)
	}
	for _, k := range hmChain14 {
		expectEqual(t, uint64(14), hmCtx.Hash(hmK(k))&15)
	}
	for _, k := range hmChain7of8 {
		expectEqual(t, uint64(7), hmCtx.Hash(hmK(k))&7)
	}
	for i, k := range hmFull16 {
		fp := hmFull16Fps[i]
		h := hmCtx.Hash(hmK(k))
		expectEqual(t, fp, hashMapMetadataTakeFingerprint(h))
	}
	// 200 key table: the slot in a capacity-256 table is the low byte
	// whenever probing did not displace the key.
	var displaced uint = 0
	for i, h := range hmHashes {
		s := hmSlots256[i]
		if uint64(s) != (h & 255) {
			displaced += 1
		}
	}
	expectEqual(t, uint(78), displaced)
}

// Zig: test "term51: HashMapUnmanaged.Metadata.isUsed: bit 7 of the control byte"
func TestHashMap_term51_HashMapUnmanaged_Metadata_isUsed_bit_7_of_the_control_byte(t *testing.T) {
	for b := range 256 {
		m := hmMeta(uint8(b))
		expectEqual(t, (b&0x80) != 0, m.IsUsed())
		expectEqual(t, uint8(b>>7), m.used)
		expectEqual(t, uint8(b&0x7f), m.fingerprint)
	}
}

// Zig: test "term51: HashMapUnmanaged.Metadata.isTombstone: only the exact byte 0x01"
func TestHashMap_term51_HashMapUnmanaged_Metadata_isTombstone_only_the_exact_byte_0x01(t *testing.T) {
	for b := range 256 {
		m := hmMeta(uint8(b))
		expectEqual(t, b == 0x01, m.IsTombstone())
	}
	// A used slot whose fingerprint is 1 is not a tombstone.
	var m hashMapMetadata
	m.Fill(1)
	expectEqual(t, uint8(0x81), hmMetaByte(m))
	expect(t, !m.IsTombstone(), "!m.isTombstone()")
}

// Zig: test "term51: HashMapUnmanaged.Metadata.isFree: only the exact byte 0x00"
func TestHashMap_term51_HashMapUnmanaged_Metadata_isFree_only_the_exact_byte_0x00(t *testing.T) {
	for b := range 256 {
		m := hmMeta(uint8(b))
		expectEqual(t, b == 0x00, m.IsFree())
	}
	// A used slot whose fingerprint is 0 is not free.
	var m hashMapMetadata
	expect(t, m.IsFree(), "m.isFree()")
	m.Fill(0)
	expectEqual(t, uint8(0x80), hmMetaByte(m))
	expect(t, !m.IsFree(), "!m.isFree()")
	expect(t, m.IsUsed(), "m.isUsed()")
}

// Zig: test "term51: HashMapUnmanaged.Metadata.takeFingerprint: top 7 bits of the hash"
func TestHashMap_term51_HashMapUnmanaged_Metadata_takeFingerprint_top_7_bits_of_the_hash(t *testing.T) {
	expectEqual(t, 0, int(hashMapMetadataTakeFingerprint(0)))
	expectEqual(t, 0, int(hashMapMetadataTakeFingerprint((1<<57)-1)))
	expectEqual(t, 1, int(hashMapMetadataTakeFingerprint(1<<57)))
	expectEqual(t, 64, int(hashMapMetadataTakeFingerprint(1<<63)))
	expectEqual(t, 127, int(hashMapMetadataTakeFingerprint(math.MaxUint64)))
	expectEqual(t, 127, int(hashMapMetadataTakeFingerprint(uint64(127)<<57)))
	expectEqual(t, 0x2a, int(hashMapMetadataTakeFingerprint(0x5400_0000_0000_0000)))
	var v uint64 = 0
	for ; v < 128; v++ {
		// The low 57 bits never matter.
		expectEqual(t, int(v), int(hashMapMetadataTakeFingerprint(v<<57)))
		expectEqual(t, int(v), int(hashMapMetadataTakeFingerprint((v<<57)|((1<<57)-1))))
		expectEqual(t, int(v), int(hashMapMetadataTakeFingerprint((v<<57)|(0x0123_4567_89ab_cdef&((1<<57)-1)))))
	}
	// Hashes of real keys.
	expectEqual(t, 0x1a, int(hashMapMetadataTakeFingerprint(hmHashes[0])))
	expectEqual(t, 0x0d, int(hashMapMetadataTakeFingerprint(hmHashes[1])))
}

// Zig: test "term51: HashMapUnmanaged.Metadata.fill: sets used and the fingerprint"
func TestHashMap_term51_HashMapUnmanaged_Metadata_fill_sets_used_and_the_fingerprint(t *testing.T) {
	var fp uint8 = 0
	for ; fp < 128; fp++ {
		var m hashMapMetadata
		m.Fill(fp)
		expectEqual(t, uint8(0x80)|fp, hmMetaByte(m))
		expect(t, m.IsUsed(), "m.isUsed()")
		expect(t, !m.IsFree(), "!m.isFree()")
		expect(t, !m.IsTombstone(), "!m.isTombstone()")

		// Fill overwrites a tombstone and a previously used slot.
		m.Remove()
		expect(t, m.IsTombstone(), "m.isTombstone()")
		m.Fill(127 - fp)
		expectEqual(t, uint8(0x80)|(127-fp), hmMetaByte(m))
		m.Fill(fp)
		expectEqual(t, uint8(0x80)|fp, hmMetaByte(m))
	}
}

// Zig: test "term51: HashMapUnmanaged.Metadata.remove: makes a tombstone from any state"
func TestHashMap_term51_HashMapUnmanaged_Metadata_remove_makes_a_tombstone_from_any_state(t *testing.T) {
	for b := range 256 {
		m := hmMeta(uint8(b))
		m.Remove()
		expectEqual(t, uint8(0x01), hmMetaByte(m))
		expect(t, m.IsTombstone(), "m.isTombstone()")
		expect(t, !m.IsUsed(), "!m.isUsed()")
		expect(t, !m.IsFree(), "!m.isFree()")
	}
}

// Zig: test "term51: HashMapUnmanaged.layoutForCapacity: layout totals and offsets for every power of two"
func TestHashMap_term51_HashMapUnmanaged_layoutForCapacity_layout_totals_and_offsets_for_every_power_of_two(t *testing.T) {
	// Pinned bytes for the grapheme map (key 4, value 16, header 16,
	// base alignment 8) and the hyperlink map (key 4, value 2, base
	// alignment 4). Entry 0 is capacity 0; entry i is capacity 2^(i-1).
	expectEqual(t, 33, len(hmGLayouts))
	expectEqual(t, 33, len(hmHLayouts))
	for i, row := range hmGLayouts {
		var cap uint32
		if i != 0 {
			cap = uint32(1) << (i - 1)
		}
		l := hmGLayout(cap)
		expectEqual(t, cap, l.capacity)
		expectEqual(t, row[0], l.total_size)
		expectEqual(t, row[1], l.keys_start)
		expectEqual(t, row[2], l.vals_start)
	}
	for i, row := range hmHLayouts {
		var cap uint32
		if i != 0 {
			cap = uint32(1) << (i - 1)
		}
		l := hmHLayout(cap)
		expectEqual(t, cap, l.capacity)
		expectEqual(t, row[0], l.total_size)
		expectEqual(t, row[1], l.keys_start)
		expectEqual(t, row[2], l.vals_start)
	}
}

// Zig: test "term51: HashMapUnmanaged.layoutForCapacity: layout formula and alignment invariants"
func TestHashMap_term51_HashMapUnmanaged_layoutForCapacity_layout_formula_and_alignment_invariants(t *testing.T) {
	// Re-derive every number from the formula, independent of the table.
	for _, sizes := range []hashMapEntrySizes{pageGraphemeMapEntrySizes, hyperlinkMapEntrySizes} {
		key_align, val_align := sizes.key_align, sizes.val_align
		base_align := hmBaseAlign(sizes)
		ksz := sizes.key_size
		vsz := sizes.val_size
		layout := func(cap uint32) hashMapLayout {
			return HashMapOffsetHashMapLayoutOf(cap, sizes)
		}
		for shift := uint(0); shift < 32; shift++ {
			cap := uint64(1) << shift
			l := layout(uint32(cap))
			keys_start := hmAlignForward(16+cap, key_align)
			vals_start := hmAlignForward(keys_start+cap*ksz, val_align)
			total := hmAlignForward(vals_start+cap*vsz, base_align)
			expectEqual(t, keys_start-16, l.keys_start)
			expectEqual(t, vals_start-16, l.vals_start)
			expectEqual(t, total, l.total_size)
			expectEqual(t, uint64(0), (l.keys_start+16)%key_align)
			expectEqual(t, uint64(0), (l.vals_start+16)%val_align)
			expectEqual(t, uint64(0), l.total_size%base_align)
			expect(t, l.total_size >= 16+cap*(1+ksz+vsz), "l.total_size >= 16 + cap * (1 + ksz + vsz)")
		}
		// Capacity 0: header only.
		z := layout(0)
		expectEqual(t, 16, int(z.total_size))
		expectEqual(t, 0, int(z.keys_start))
		expectEqual(t, 0, int(z.vals_start))
		expectEqual(t, uint32(0), z.capacity)
	}
}

// Zig: test "term51: HashMapUnmanaged.layoutForCapacity: capacity echoes the requested power of two over a wide sweep"
func TestHashMap_term51_HashMapUnmanaged_layoutForCapacity_capacity_echoes_the_requested_power_of_two_over_a_wide_sweep(t *testing.T) {
	// The caller (page.zig) rounds a requested count up to a power of two;
	// sweep every request 0..70000 and check what the layout reports.
	for n := range uint32(70001) {
		var cap uint32
		if n != 0 {
			cap = hmCeilPowerOfTwo(n)
		}
		lg := hmGLayout(cap)
		lh := hmHLayout(cap)
		expectEqual(t, cap, lg.capacity)
		expectEqual(t, cap, lh.capacity)
		expect(t, cap >= n, "cap >= n")
		if n > 1 {
			expect(t, cap < 2*n, "cap < 2 * n")
		}
		if n > 0 {
			expect(t, bits.OnesCount32(cap) == 1, "std.math.isPowerOfTwo(cap)")
		}

		var idx int
		if cap != 0 {
			idx = bits.Len32(cap) - 1 + 1
		}
		expectEqual(t, hmGLayouts[idx][0], lg.total_size)
		expectEqual(t, hmGLayouts[idx][1], lg.keys_start)
		expectEqual(t, hmGLayouts[idx][2], lg.vals_start)
		expectEqual(t, hmHLayouts[idx][0], lh.total_size)
		expectEqual(t, hmHLayouts[idx][1], lh.keys_start)
		expectEqual(t, hmHLayouts[idx][2], lh.vals_start)
	}
}

// Zig: test "term51: HashMapUnmanaged.layoutForCapacity: layout of the largest capacities does not overflow"
func TestHashMap_term51_HashMapUnmanaged_layoutForCapacity_layout_of_the_largest_capacities_does_not_overflow(t *testing.T) {
	const max_cap uint32 = 1 << 31
	g := hmGLayout(max_cap)
	expectEqual(t, uint64(45097156624), g.total_size)
	expectEqual(t, uint64(2147483648), g.keys_start)
	expectEqual(t, uint64(10737418240), g.vals_start)
	h := hmHLayout(max_cap)
	expectEqual(t, uint64(15032385552), h.total_size)
	expect(t, h.vals_start > h.keys_start, "h.vals_start > h.keys_start")
	expectEqual(t, max_cap, h.capacity)
}

// Zig: test "term51: OffsetHashMap.layout: is layoutForCapacity of the unmanaged map"
func TestHashMap_term51_OffsetHashMap_layout_is_layoutForCapacity_of_the_unmanaged_map(t *testing.T) {
	for shift := uint(0); shift < 32; shift++ {
		cap := uint32(uint64(1) << shift)
		a := hmGLayout(cap)
		b := hashMapLayoutForCapacity(cap, pageGraphemeMapEntrySizes)
		expectEqual(t, b.total_size, a.total_size)
		expectEqual(t, b.keys_start, a.keys_start)
		expectEqual(t, b.vals_start, a.vals_start)
		expectEqual(t, b.capacity, a.capacity)
		c := hmHLayout(cap)
		d := hashMapLayoutForCapacity(cap, hyperlinkMapEntrySizes)
		expectEqual(t, d.total_size, c.total_size)
		expectEqual(t, d.keys_start, c.keys_start)
		expectEqual(t, d.vals_start, c.vals_start)
	}
	expectEqual(t, 16, int(hmGLayout(0).total_size))
}

// Zig: test "term51: OffsetHashMap: relocating the whole buffer keeps the map working"
func TestHashMap_term51_OffsetHashMap_relocating_the_whole_buffer_keeps_the_map_working(t *testing.T) {
	// The map stores only offsets, so a byte copy of the backing memory
	// (what a page clone does) yields an equal, independent map.
	a := hmGBoxInit(32)
	ma := a.map_()
	for i := range 20 {
		zigTry(t, ma.Put(hmK(uint32(4096+8*i)), hmSl(uint32(i*2), uint(i+1))))
	}

	b := hmGBoxInit(32)
	// Zig-only: @memcpy(b.buf, a.buf); b.offset_map = a.offset_map;
	// Go has no byte buffer: the copy is made store by store (the control
	// bytes, the keys, the values and the header's size).
	{
		mb0 := b.map_()
		copy(mb0.metadata, ma.metadata)
		copy(mb0.keys(), ma.keys())
		copy(mb0.values(), ma.values())
		mb0.header().size = ma.header().size
	}
	mb := b.map_()
	expectEqual(t, uint32(20), mb.Count())
	for i := range 20 {
		v := zigUnwrap(t, mb.Get(hmK(uint32(4096+8*i))))
		hmExpectSl(t, hmSl(uint32(i*2), uint(i+1)), v)
		expectEqual(t, hmSlot(ma, uint32(4096+8*i)), hmSlot(mb, uint32(4096+8*i)))
	}
	// Zig-only: keys and values live inside each buffer, not the other one
	// (kp = @intFromPtr(mb.keys()) is inside b.buf).
	expect(t, &ma.keys()[0] != &mb.keys()[0], "keys are not shared")

	// Mutating the copy leaves the original alone.
	expect(t, mb.Remove(hmK(4096)), "mb.remove(term51XK(4096))")
	zigTry(t, mb.Put(hmK(1), hmSl(7, 7)))
	expectEqual(t, uint32(20), mb.Count())
	expect(t, ma.Contains(hmK(4096)), "ma.contains(term51XK(4096))")
	expect(t, !ma.Contains(hmK(1)), "!ma.contains(term51XK(1))")
	expect(t, !mb.Contains(hmK(4096)), "!mb.contains(term51XK(4096))")
	hmExpectSl(t, hmSl(7, 7), zigUnwrap(t, mb.Get(hmK(1))))
}

// Zig: test "term51: OffsetHashMap.init: stores the metadata offset against the true base"
func TestHashMap_term51_OffsetHashMap_init_stores_the_metadata_offset_against_the_true_base(t *testing.T) {
	// layout header offsets only: the offset is 16 (sizeof Header) past the
	// start of the map's own memory, relative to the base of the allocation.
	check := func(m func(at uint) (capacity uint32, count uint32)) {
		for _, at := range []uint{0, 8, 64, 1024} {
			// Zig-only: box.offset_map.metadata.offset == at + 16 and
			// @intFromPtr(m.metadata.?) == @intFromPtr(box.buf.ptr) + at + 16
			// (byte offsets and addresses).
			capacity, count := m(at)
			expectEqual(t, uint32(16), capacity)
			expectEqual(t, uint32(0), count)
		}
	}
	// inline for (.{ term51XGBox, term51XHBox }) |Box|
	check(func(at uint) (uint32, uint32) {
		box := hmGBoxInitAt(16, at)
		m := box.map_()
		return m.Capacity(), m.Count()
	})
	check(func(at uint) (uint32, uint32) {
		box := hmHBoxInitAt(16, at)
		m := box.map_()
		return m.Capacity(), m.Count()
	})
}

// hmInitEmptyChecks is the body of "OffsetHashMap.init: map starts empty and
// ignores dirty memory" for one of the two boxes.
func hmInitEmptyChecks[V any](t *testing.T, newBox func(cap uint32) hmBox[V]) {
	t.Helper()
	for _, cap := range []uint32{0, 1, 2, 8, 64, 1024} {
		box := newBox(cap)
		m := box.map_()
		expectEqual(t, cap, m.Capacity())
		expectEqual(t, uint32(0), m.Count())
		for i := range uint(cap) {
			expectEqual(t, uint8(0), hmCtl(m, i))
		}
		expect(t, !m.Contains(hmK(4096)), "!m.contains(term51XK(4096))")
		expect(t, !m.Get(hmK(4096)).Valid, "m.get(term51XK(4096)) == null")
		it := m.Iterator()
		expect(t, !it.Next().Valid, "it.next() == null")
	}
}

// Zig: test "term51: OffsetHashMap.init: map starts empty and ignores dirty memory"
func TestHashMap_term51_OffsetHashMap_init_map_starts_empty_and_ignores_dirty_memory(t *testing.T) {
	// Zig: the buffer starts dirty (0xAA); Go memory starts zeroed.
	hmInitEmptyChecks(t, hmGBoxInit)
	hmInitEmptyChecks(t, hmHBoxInit)
}

// Zig: test "term51: OffsetHashMap.map: pointer map for a base address"
func TestHashMap_term51_OffsetHashMap_map_pointer_map_for_a_base_address(t *testing.T) {
	box := hmHBoxInitAt(8, 16)
	a := box.offset_map.Map()
	b := box.map_()
	expect(t, &a.metadata[0] == &b.metadata[0], "a.metadata == b.metadata")
	// Zig-only: try term51XEq(@intFromPtr(box.buf.ptr) + 16 + 16, @intFromPtr(a.metadata.?));
	// A different base address (an exact copy of the memory) maps there.
	// Zig-only: const copy = try std.testing.allocator.alignedAlloc(u8, term51XHMap.base_align, box.buf.len);
	// Zig-only: @memcpy(copy, box.buf);
	// Zig: `box.offset_map.map(copy.ptr)` has no Go counterpart (no base); the
	// map of the box itself is used for the capacity check.
	c := box.offset_map.Map()
	// Zig-only: try term51XEq(@intFromPtr(copy.ptr) + 32, @intFromPtr(c.metadata.?));
	expectEqual(t, uint32(8), c.Capacity())
	// Zig-only: default offset (0) maps to the base itself:
	// const d = (term51XHMap{}).map(copy.ptr); d.metadata == copy.ptr.
}

// Zig: test "term51: HashMapUnmanaged.init: layout header offsets, capacity and size"
func TestHashMap_term51_HashMapUnmanaged_init_layout_header_offsets_capacity_and_size(t *testing.T) {
	for _, cap := range []uint32{0, 1, 2, 4, 8, 16, 256} {
		for _, at := range []uint{0, 24, 128} {
			box := hmGBoxInitAt(cap, at)
			// Zig-only: const l = term51XGMap.layout(cap); (used by the
			// offset checks below)
			m := box.map_()
			hdr := m.header()
			expectEqual(t, cap, hdr.capacity)
			expectEqual(t, uint32(0), hdr.size)
			// Zig-only: try term51XEq(@as(u32, @intCast(l.keys_start)), hdr.keys.offset);
			// Zig-only: try term51XEq(@as(u32, @intCast(l.vals_start)), hdr.values.offset);
			// keys()/values() are the layout offsets from the metadata.
			// Zig-only: try term51XEq(@intFromPtr(m.metadata.?) + l.keys_start, @intFromPtr(m.keys()));
			// Zig-only: try term51XEq(@intFromPtr(m.metadata.?) + l.vals_start, @intFromPtr(m.values()));
		}
	}
}

// Zig: test "term51: HashMapUnmanaged.init: clears only the control bytes"
func TestHashMap_term51_HashMapUnmanaged_init_clears_only_the_control_bytes(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Control bytes are the 16 bytes after the 16 byte header; keys follow.
	// Zig: `box.buf[16 + i]` is the control byte of slot i.
	for i := range uint(16) {
		expectEqual(t, uint8(0), hmCtl(m, i))
	}
	// Zig-only: try term51XEq(@as(u8, 0xAA), box.buf[32]); (dirty memory
	// past the control bytes: Go memory starts zeroed)
	// Header bytes after capacity/size were not touched: size is 0 and
	// capacity 16 (header words 2 and 3).
	expectEqual(t, uint32(16), m.Capacity())
	expectEqual(t, uint32(0), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.count: number of live entries"
func TestHashMap_term51_HashMapUnmanaged_count_number_of_live_entries(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	expectEqual(t, uint32(0), m.Count())
	for i := range 16 {
		zigTry(t, m.Put(hmK(uint32(8*i)), HyperlinkId(i)))
		expectEqual(t, uint32(i+1), m.Count())
	}
	// Updating an existing key does not change the count.
	zigTry(t, m.Put(hmK(0), 99))
	expectEqual(t, uint32(16), m.Count())
	// Removing present keys decrements, absent keys do not.
	expect(t, m.Remove(hmK(0)), "m.remove(term51XK(0))")
	expectEqual(t, uint32(15), m.Count())
	expect(t, !m.Remove(hmK(0)), "!m.remove(term51XK(0))")
	expectEqual(t, uint32(15), m.Count())
	expect(t, !m.Remove(hmK(12345)), "!m.remove(term51XK(12345))")
	expectEqual(t, uint32(15), m.Count())
	// The count is the header size.
	expectEqual(t, uint32(15), m.header().size)
}

// Zig: test "term51: HashMapUnmanaged.capacity: fixed capacity, available accounting"
func TestHashMap_term51_HashMapUnmanaged_capacity_fixed_capacity_available_accounting(t *testing.T) {
	box := hmHBoxInit(8)
	m := box.map_()
	expectEqual(t, uint32(8), m.Capacity())
	// available == capacity - count in every state; tombstones do not count.
	expectEqual(t, uint32(8), m.Capacity()-m.Count())
	for i := range 8 {
		zigTry(t, m.Put(hmK(uint32(8*i)), 1))
		expectEqual(t, uint32(7-i), m.Capacity()-m.Count())
		expectEqual(t, uint32(8), m.Capacity())
	}
	hmExpectError(t, ErrOutOfMemory, m.Put(hmK(999), 1))
	expectEqual(t, uint32(8), m.Capacity())
	expectEqual(t, uint32(0), m.Capacity()-m.Count())
	// Removing frees availability even though the slot becomes a tombstone.
	expect(t, m.Remove(hmK(8)), "m.remove(term51XK(8))")
	expectEqual(t, uint32(1), m.Capacity()-m.Count())
	expect(t, m.Remove(hmK(16)), "m.remove(term51XK(16))")
	expectEqual(t, uint32(2), m.Capacity()-m.Count())
	zigTry(t, m.Put(hmK(999), 1))
	expectEqual(t, uint32(1), m.Capacity()-m.Count())
	// Updates leave availability alone.
	zigTry(t, m.Put(hmK(999), 2))
	expectEqual(t, uint32(1), m.Capacity()-m.Count())
	// The capacity of a map without metadata is zero.
	var empty hmHU
	expectEqual(t, uint32(0), empty.Capacity())
	// Capacity zero maps.
	z := hmHBoxInit(0)
	zm := z.map_()
	expectEqual(t, uint32(0), zm.Capacity())
	expectEqual(t, uint32(0), zm.Count())
}

// Zig: test "term51: HashMapUnmanaged.header: capacity and size words, shared by copies of the map"
func TestHashMap_term51_HashMapUnmanaged_header_capacity_and_size_words_shared_by_copies_of_the_map(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	expectEqual(t, uint32(16), m.header().capacity)
	expectEqual(t, uint32(0), m.header().size)
	zigTry(t, m.Put(hmK(8), hmSl(1, 1)))
	// Another map value on the same memory sees the same header.
	m2 := box.map_()
	expectEqual(t, uint32(1), m2.header().size)
	expect(t, m.header() == m2.header(), "@intFromPtr(m.header()) == @intFromPtr(m2.header())")
	// The header sits immediately before the metadata.
	// Zig-only: try term51XEq(@intFromPtr(m.metadata.?) - 16, @intFromPtr(m.header()));
	// Zig-only: try term51XEq(@intFromPtr(box.buf.ptr), @intFromPtr(m.header()));
}

// Zig: test "term51: HashMapUnmanaged.keys: key array start and slot storage"
func TestHashMap_term51_HashMapUnmanaged_keys_key_array_start_and_slot_storage(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig-only: const l = term51XHMap.layout(16);
	// Zig-only: try term51XEq(@intFromPtr(box.buf.ptr) + 16 + l.keys_start, @intFromPtr(m.keys()));
	for i, k := range hmChain5[0:3] {
		zigTry(t, m.Put(hmK(k), HyperlinkId(i)))
		// The key is stored at the slot index of the key array.
		slot := zigUnwrap(t, hmSlot(m, k))
		expectEqual(t, uint(5+i), slot)
		expectEqual(t, k, m.keys()[slot])
		expectEqual(t, uint16(i), m.values()[slot])
	}
}

// Zig: test "term51: HashMapUnmanaged.values: value array start follows the keys"
func TestHashMap_term51_HashMapUnmanaged_values_value_array_start_follows_the_keys(t *testing.T) {
	box := hmGBoxInit(32)
	m := box.map_()
	// Zig-only: const l = term51XGMap.layout(32);
	// Zig-only: try term51XEq(@intFromPtr(box.buf.ptr) + 16 + l.vals_start, @intFromPtr(m.values()));
	// Zig-only: try term51XEq(l.vals_start - l.keys_start, @intFromPtr(m.values()) - @intFromPtr(m.keys()));
	zigTry(t, m.Put(hmK(4096), hmSl(12, 3)))
	slot := zigUnwrap(t, hmSlot(m, 4096))
	hmExpectSl(t, hmSl(12, 3), m.values()[slot])
	expectEqual(t, uint32(4096), m.keys()[slot])
	// Pointers from a lookup are the array elements.
	expect(t, &m.values()[slot] == m.GetPtr(hmK(4096)), "@intFromPtr(&m.values()[slot]) == @intFromPtr(m.getPtr(term51XK(4096)).?)")
	expect(t, &m.keys()[slot] == zigUnwrap(t, m.GetEntry(hmK(4096))).key_ptr, "@intFromPtr(&m.keys()[slot]) == @intFromPtr(m.getEntry(term51XK(4096)).?.key_ptr)")
}

// Zig: test "term51: HashMapUnmanaged.initMetadatas: zeroes exactly capacity control bytes"
func TestHashMap_term51_HashMapUnmanaged_initMetadatas_zeroes_exactly_capacity_control_bytes(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Dirty all control bytes and set a size in the header.
	for i := range 16 {
		m.metadata[i] = hmMeta(0xFF)
	}
	m.header().size = 3
	m.initMetadatas()
	for i := range uint(16) {
		expectEqual(t, uint8(0), hmCtl(m, i))
	}
	// The header is untouched (only clearRetainingCapacity resets the size)
	// and so are the bytes after the control bytes (the key array).
	expectEqual(t, uint32(3), m.Count())
	expectEqual(t, uint32(16), m.Capacity())
	// Zig-only: for (32..40) |i| try term51XEq(@as(u8, 0xAA), box.buf[i]);
	// (the bytes after the control bytes: Go memory starts zeroed)

	// Capacity zero: nothing to clear.
	z := hmHBoxInit(0)
	zm := z.map_()
	zm.initMetadatas()
	expectEqual(t, uint32(0), zm.Capacity())
}

// Zig: test "term51: HashMapUnmanaged.growIfNeeded: refuses when more than the free slots are requested"
func TestHashMap_term51_HashMapUnmanaged_growIfNeeded_refuses_when_more_than_the_free_slots_are_requested(t *testing.T) {
	box := hmHBoxInit(8)
	m := box.map_()
	// Empty: 0..8 fit, 9 does not.
	for n := range 9 {
		zigTry(t, m.growIfNeeded(uint32(n)))
	}
	hmExpectError(t, ErrOutOfMemory, m.growIfNeeded(9))
	hmExpectError(t, ErrOutOfMemory, m.growIfNeeded(math.MaxUint32))
	// Five live entries: three more fit.
	for i := range 5 {
		zigTry(t, m.Put(hmK(uint32(8*i)), 0))
	}
	zigTry(t, m.growIfNeeded(0))
	zigTry(t, m.growIfNeeded(3))
	hmExpectError(t, ErrOutOfMemory, m.growIfNeeded(4))
	// Removing two (tombstones) makes five available.
	_ = m.Remove(hmK(0))
	_ = m.Remove(hmK(8))
	zigTry(t, m.growIfNeeded(5))
	hmExpectError(t, ErrOutOfMemory, m.growIfNeeded(6))
	// Full.
	zigTry(t, m.Put(hmK(100), 0))
	zigTry(t, m.Put(hmK(108), 0))
	zigTry(t, m.Put(hmK(116), 0))
	zigTry(t, m.Put(hmK(124), 0))
	zigTry(t, m.Put(hmK(132), 0))
	expectEqual(t, uint32(8), m.Count())
	zigTry(t, m.growIfNeeded(0))
	hmExpectError(t, ErrOutOfMemory, m.growIfNeeded(1))
	// The state did not change.
	expectEqual(t, uint32(8), m.Count())
	expectEqual(t, uint32(8), m.Capacity())

	// Capacity zero.
	z := hmHBoxInit(0)
	zm := z.map_()
	zigTry(t, zm.growIfNeeded(0))
	hmExpectError(t, ErrOutOfMemory, zm.growIfNeeded(1))
}

// Zig: test "term51: HashMapUnmanaged.putAssumeCapacityNoClobberContext: slot and control byte of 200 keys"
func TestHashMap_term51_HashMapUnmanaged_putAssumeCapacityNoClobberContext_slot_and_control_byte_of_200_keys(t *testing.T) {
	box := hmGBoxInit(256)
	m := box.map_()
	for i := range 200 {
		k := uint32(4096 + 8*i)
		m.PutAssumeCapacityNoClobberContext(hmK(k), hmSl(uint32(3*i), uint(i%7+1)), hmCtx)
		expectEqual(t, uint32(i+1), m.Count())
	}
	var used uint = 0
	for s := range uint(256) {
		if hmCtl(m, s) != 0 {
			used += 1
		}
	}
	expectEqual(t, uint(200), used)
	expectEqual(t, uint32(56), m.Capacity()-m.Count())
	for i := range 200 {
		k := uint32(4096 + 8*i)
		slot := zigUnwrap(t, hmSlot(m, k))
		expectEqual(t, uint(hmSlots256[i]), slot)
		expectEqual(t, hmUsedCtl(k), hmCtl(m, slot))
		expectEqual(t, uint8(0x80)|uint8(hmHashes[i]>>57), hmCtl(m, slot))
		expectEqual(t, k, m.keys()[slot])
		hmExpectSl(t, hmSl(uint32(3*i), uint(i%7+1)), m.values()[slot])
		hmExpectSl(t, hmSl(uint32(3*i), uint(i%7+1)), zigUnwrap(t, m.Get(hmK(k))))
	}
}

// Zig: test "term51: HashMapUnmanaged.putAssumeCapacityNoClobberContext: probes linearly past used slots and wraps"
func TestHashMap_term51_HashMapUnmanaged_putAssumeCapacityNoClobberContext_probes_linearly_past_used_slots_and_wraps(t *testing.T) {
	// Six keys with home slot 5 land on 5..10.
	{
		box := hmHBoxInit(16)
		m := box.map_()
		for i, k := range hmChain5[0:6] {
			m.PutAssumeCapacityNoClobberContext(hmK(k), HyperlinkId(i), hmCtx)
			expectEqual(t, uint(5+i), zigUnwrap(t, hmSlot(m, k)))
			expectEqual(t, hmUsedCtl(k), hmCtl(m, uint(5+i)))
		}
		expectEqual(t, uint(10), hmCountCtl(m, 0))
		// A key with another home slot is not displaced.
		for s := range uint(5) {
			expectEqual(t, uint8(0), hmCtl(m, s))
		}
		for s := uint(11); s < 16; s++ {
			expectEqual(t, uint8(0), hmCtl(m, s))
		}
	}
	// Home slot 14 of 16: 14, 15, then wraps to 0, 1.
	{
		box := hmHBoxInit(16)
		m := box.map_()
		want := [...]uint{14, 15, 0, 1}
		for i, k := range hmChain14[0:4] {
			s := want[i]
			m.PutAssumeCapacityNoClobberContext(hmK(k), 1, hmCtx)
			expectEqual(t, s, zigUnwrap(t, hmSlot(m, k)))
		}
	}
	// Home slot 7 of 8: 7, 0, 1, 2, 3.
	{
		box := hmHBoxInit(8)
		m := box.map_()
		want := [...]uint{7, 0, 1, 2, 3}
		for i, k := range hmChain7of8 {
			s := want[i]
			m.PutAssumeCapacityNoClobberContext(hmK(k), 1, hmCtx)
			expectEqual(t, s, zigUnwrap(t, hmSlot(m, k)))
		}
		expectEqual(t, uint32(5), m.Count())
	}
}

// Zig: test "term51: HashMapUnmanaged.putAssumeCapacityNoClobberContext: fills the first unused slot, tombstones included"
func TestHashMap_term51_HashMapUnmanaged_putAssumeCapacityNoClobberContext_fills_the_first_unused_slot_tombstones_included(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	for i, k := range hmChain5[0:6] {
		m.PutAssumeCapacityNoClobber(hmK(k), HyperlinkId(i))
	}
	_ = m.Remove(hmK(hmChain5[1]))
	expectEqual(t, uint8(1), hmCtl(m, 6))
	// The tombstone at slot 6 is the first unused slot of the chain.
	m.PutAssumeCapacityNoClobberContext(hmK(hmChain5[6]), 60, hmCtx)
	expectEqual(t, uint(6), zigUnwrap(t, hmSlot(m, hmChain5[6])))
	expectEqual(t, uint32(6), m.Count())
	// Next goes after the end of the chain.
	m.PutAssumeCapacityNoClobber(hmK(hmChain5[7]), 70)
	expectEqual(t, uint(11), zigUnwrap(t, hmSlot(m, hmChain5[7])))
	expectEqual(t, uint16(60), zigUnwrap(t, m.Get(hmK(hmChain5[6]))))
	expectEqual(t, uint16(70), zigUnwrap(t, m.Get(hmK(hmChain5[7]))))
}

// Zig: test "term51: HashMapUnmanaged.putNoClobberContext: inserts new keys and refuses past capacity"
func TestHashMap_term51_HashMapUnmanaged_putNoClobberContext_inserts_new_keys_and_refuses_past_capacity(t *testing.T) {
	box := hmGBoxInit(8)
	m := box.map_()
	for i := range 8 {
		k := hmK(uint32(4096 + 8*i))
		if i%2 == 0 {
			zigTry(t, m.PutNoClobberContext(k, hmSl(uint32(i), 1), hmCtx))
		} else {
			zigTry(t, m.PutNoClobber(k, hmSl(uint32(i), 1)))
		}
		expectEqual(t, uint32(i+1), m.Count())
	}
	// One past capacity: OutOfMemory, nothing changed.
	hmExpectError(t, ErrOutOfMemory, m.PutNoClobberContext(hmK(1), hmSl(9, 9), hmCtx))
	hmExpectError(t, ErrOutOfMemory, m.PutNoClobber(hmK(1), hmSl(9, 9)))
	expectEqual(t, uint32(8), m.Count())
	expect(t, !m.Contains(hmK(1)), "!m.contains(term51XK(1))")
	for i := range 8 {
		hmExpectSl(t, hmSl(uint32(i), 1), zigUnwrap(t, m.Get(hmK(uint32(4096+8*i)))))
	}
	// A removal makes room again (into the tombstone).
	gone := zigUnwrap(t, hmSlot(m, 4096+8*3))
	expect(t, m.Remove(hmK(4096+8*3)), "m.remove(term51XK(4096 + 8 * 3))")
	expectEqual(t, uint8(1), hmCtl(m, gone))
	zigTry(t, m.PutNoClobber(hmK(1), hmSl(9, 9)))
	expectEqual(t, uint32(8), m.Count())
	hmExpectSl(t, hmSl(9, 9), zigUnwrap(t, m.Get(hmK(1))))
	// With no free slot the tombstone is the only place it can go.
	expectEqual(t, gone, zigUnwrap(t, hmSlot(m, 1)))
	// The map is full again (the tombstone was reused): refuses again.
	hmExpectError(t, ErrOutOfMemory, m.PutNoClobber(hmK(2), hmSl(9, 9)))

	// Capacity zero refuses everything.
	z := hmGBoxInit(0)
	zm := z.map_()
	hmExpectError(t, ErrOutOfMemory, zm.PutNoClobberContext(hmK(1), hmSl(1, 1), hmCtx))
	expectEqual(t, uint32(0), zm.Count())
}

// Zig: test "term51: HashMapUnmanaged.putAssumeCapacityContext: inserts or overwrites in place"
func TestHashMap_term51_HashMapUnmanaged_putAssumeCapacityContext_inserts_or_overwrites_in_place(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	for i, k := range hmChain5[0:4] {
		m.PutAssumeCapacityContext(hmK(k), hmSl(uint32(i), 1), hmCtx)
	}
	expectEqual(t, uint32(4), m.Count())
	slots := [...]uint{5, 6, 7, 8}
	for i, k := range hmChain5[0:4] {
		s := slots[i]
		expectEqual(t, s, zigUnwrap(t, hmSlot(m, k)))
	}

	// Overwriting keeps the slot, count and control byte.
	before := make([]hashMapMetadata, 16)
	copy(before, m.metadata[0:16])
	for i, k := range hmChain5[0:4] {
		m.PutAssumeCapacityContext(hmK(k), hmSl(uint32(100+i), 2), hmCtx)
	}
	expectEqual(t, uint32(4), m.Count())
	for i, k := range hmChain5[0:4] {
		s := slots[i]
		expectEqual(t, s, zigUnwrap(t, hmSlot(m, k)))
		hmExpectSl(t, hmSl(uint32(100+i), 2), zigUnwrap(t, m.Get(hmK(k))))
	}
	expectEqualSlices(t, before, m.metadata[0:16])

	// The wrapper is the same operation.
	m.PutAssumeCapacity(hmK(hmChain5[4]), hmSl(5, 5))
	expectEqual(t, uint(9), zigUnwrap(t, hmSlot(m, hmChain5[4])))
	m.PutAssumeCapacity(hmK(hmChain5[4]), hmSl(6, 6))
	expectEqual(t, uint32(5), m.Count())
	hmExpectSl(t, hmSl(6, 6), zigUnwrap(t, m.Get(hmK(hmChain5[4]))))

	// Existing keys behind a tombstone are updated, not duplicated.
	_ = m.Remove(hmK(hmChain5[0]))
	m.PutAssumeCapacity(hmK(hmChain5[3]), hmSl(77, 7))
	expectEqual(t, uint32(4), m.Count())
	expectEqual(t, uint8(1), hmCtl(m, 5))
	hmExpectSl(t, hmSl(77, 7), zigUnwrap(t, m.Get(hmK(hmChain5[3]))))
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacityAdapted: a new key recycles the first tombstone in probe order"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacityAdapted_a_new_key_recycles_the_first_tombstone_in_probe_order(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	// Zig: `const ctx: term51XRawCtx = .{}`; see the file header.
	ctx := hmCtx
	for i, k := range c[0:6] {
		gop := m.GetOrPutAssumeCapacityAdapted(k, ctx)
		expect(t, !gop.found_existing, "!gop.found_existing")
		// The adapted variant leaves writing the key to the caller.
		*gop.key_ptr = hmK(k)
		*gop.value_ptr = HyperlinkId(i)
		expectEqual(t, uint(5+i), zigUnwrap(t, hmSlot(m, k)))
		expect(t, &m.keys()[5+i] == gop.key_ptr, "@intFromPtr(&m.keys()[5 + i]) == @intFromPtr(gop.key_ptr)")
		expect(t, &m.values()[5+i] == gop.value_ptr, "@intFromPtr(&m.values()[5 + i]) == @intFromPtr(gop.value_ptr)")
	}
	expectEqual(t, uint32(6), m.Count())

	// Existing key: found, count unchanged, pointers to its slot.
	{
		gop := m.GetOrPutAssumeCapacityAdapted(c[3], ctx)
		expect(t, gop.found_existing, "gop.found_existing")
		expect(t, &m.keys()[8] == gop.key_ptr, "@intFromPtr(&m.keys()[8]) == @intFromPtr(gop.key_ptr)")
		expectEqual(t, uint16(3), *gop.value_ptr)
		expectEqual(t, uint32(6), m.Count())
	}

	// A tombstone in the chain is recycled by a new key (not the free slot
	// at the end of the chain).
	_ = m.Remove(hmK(c[2]))
	expectEqual(t, uint8(1), hmCtl(m, 7))
	{
		gop := m.GetOrPutAssumeCapacityAdapted(c[6], ctx)
		expect(t, !gop.found_existing, "!gop.found_existing")
		*gop.key_ptr = hmK(c[6])
		*gop.value_ptr = 6
		expect(t, &m.keys()[7] == gop.key_ptr, "@intFromPtr(&m.keys()[7]) == @intFromPtr(gop.key_ptr)")
		expectEqual(t, hmUsedCtl(c[6]), hmCtl(m, 7))
		expectEqual(t, uint32(6), m.Count())
		expectEqual(t, uint8(0), hmCtl(m, 11))
	}

	// Two tombstones: the first one met while probing is used.
	_ = m.Remove(hmK(c[0]))
	_ = m.Remove(hmK(c[1]))
	expectEqual(t, uint8(1), hmCtl(m, 5))
	expectEqual(t, uint8(1), hmCtl(m, 6))
	zigTry(t, m.Put(hmK(c[7]), 7))
	expectEqual(t, uint(5), zigUnwrap(t, hmSlot(m, c[7])))
	zigTry(t, m.Put(hmK(c[0]), 0))
	expectEqual(t, uint(6), zigUnwrap(t, hmSlot(m, c[0])))
	expectEqual(t, uint32(6), m.Count())

	// An existing key behind tombstones is found, tombstones are kept.
	_ = m.Remove(hmK(c[7]))
	expectEqual(t, uint8(1), hmCtl(m, 5))
	gop := m.GetOrPutAssumeCapacityAdapted(c[4], ctx)
	expect(t, gop.found_existing, "gop.found_existing")
	expect(t, &m.keys()[9] == gop.key_ptr, "@intFromPtr(&m.keys()[9]) == @intFromPtr(gop.key_ptr)")
	expectEqual(t, uint8(1), hmCtl(m, 5))
	expectEqual(t, uint32(5), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacityAdapted: tombstones are recycled in probe order across the wrap"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacityAdapted_tombstones_are_recycled_in_probe_order_across_the_wrap(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain14
	for _, k := range c[0:4] {
		zigTry(t, m.Put(hmK(k), 1))
	}
	expectEqual(t, uint(14), zigUnwrap(t, hmSlot(m, c[0])))
	expectEqual(t, uint(15), zigUnwrap(t, hmSlot(m, c[1])))
	expectEqual(t, uint(0), zigUnwrap(t, hmSlot(m, c[2])))
	expectEqual(t, uint(1), zigUnwrap(t, hmSlot(m, c[3])))

	// Tombstones at 15 and 0: probing from 14 meets 15 first, even though
	// 0 is the lower index.
	_ = m.Remove(hmK(c[1]))
	_ = m.Remove(hmK(c[2]))
	expectEqual(t, uint8(1), hmCtl(m, 15))
	expectEqual(t, uint8(1), hmCtl(m, 0))
	zigTry(t, m.Put(hmK(c[4]), 4))
	expectEqual(t, uint(15), zigUnwrap(t, hmSlot(m, c[4])))
	zigTry(t, m.Put(hmK(c[5]), 5))
	expectEqual(t, uint(0), zigUnwrap(t, hmSlot(m, c[5])))
	expectEqual(t, uint32(4), m.Count())
	// The key behind the tombstone is still found.
	expectEqual(t, uint(1), zigUnwrap(t, hmSlot(m, c[3])))
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacityContext: sets the key of a new entry"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacityContext_sets_the_key_of_a_new_entry(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	gop := m.GetOrPutAssumeCapacityContext(hmK(4096), hmCtx)
	expect(t, !gop.found_existing, "!gop.found_existing")
	expectEqual(t, uint32(4096), *gop.key_ptr)
	*gop.value_ptr = 5
	expectEqual(t, uint32(1), m.Count())
	expectEqual(t, uint16(5), zigUnwrap(t, m.Get(hmK(4096))))
	slot := zigUnwrap(t, hmSlot(m, 4096))
	expect(t, &m.keys()[slot] == gop.key_ptr, "@intFromPtr(&m.keys()[slot]) == @intFromPtr(gop.key_ptr)")
	expectEqual(t, hmUsedCtl(4096), hmCtl(m, slot))

	// Existing: same slot, key untouched, count the same.
	again := m.GetOrPutAssumeCapacityContext(hmK(4096), hmCtx)
	expect(t, again.found_existing, "again.found_existing")
	expect(t, gop.key_ptr == again.key_ptr, "@intFromPtr(gop.key_ptr) == @intFromPtr(again.key_ptr)")
	expectEqual(t, uint16(5), *again.value_ptr)
	expectEqual(t, uint32(1), m.Count())

	// Wrapper without context.
	w := m.GetOrPutAssumeCapacity(hmK(4104))
	expect(t, !w.found_existing, "!w.found_existing")
	expectEqual(t, uint32(4104), *w.key_ptr)
	expectEqual(t, uint32(2), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacityAdapted: equal fingerprints are told apart by key equality"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacityAdapted_equal_fingerprints_are_told_apart_by_key_equality(t *testing.T) {
	// 2168 and 2976 share home slot 3 and fingerprint 0x05 in a capacity-16
	// table; the second must skip the first although the fingerprints match.
	box := hmHBoxInit(16)
	m := box.map_()
	expectEqual(t, uint64(3), hmCtx.Hash(hmK(2168))&15)
	expectEqual(t, uint64(3), hmCtx.Hash(hmK(2976))&15)
	expectEqual(t, hmUsedCtl(2168), hmUsedCtl(2976))
	expectEqual(t, uint8(0x85), hmUsedCtl(2168))
	zigTry(t, m.Put(hmK(2168), 1))
	zigTry(t, m.Put(hmK(2976), 2))
	expectEqual(t, uint(3), zigUnwrap(t, hmSlot(m, 2168)))
	expectEqual(t, uint(4), zigUnwrap(t, hmSlot(m, 2976)))
	expectEqual(t, uint8(0x85), hmCtl(m, 3))
	expectEqual(t, uint8(0x85), hmCtl(m, 4))
	expectEqual(t, uint16(1), zigUnwrap(t, m.Get(hmK(2168))))
	expectEqual(t, uint16(2), zigUnwrap(t, m.Get(hmK(2976))))
	// Updating the second does not touch the first.
	zigTry(t, m.Put(hmK(2976), 22))
	expectEqual(t, uint16(1), zigUnwrap(t, m.Get(hmK(2168))))
	expectEqual(t, uint16(22), zigUnwrap(t, m.Get(hmK(2976))))
	expectEqual(t, uint32(2), m.Count())
	// A third key with this fingerprint that is absent is not found.
	expect(t, !m.Get(hmK(2976+8)).Valid, "m.get(term51XK(2976 + 8)) == null")
	// Remove the first: the second is still found behind the tombstone and
	// the first can be re-inserted into its own slot.
	expect(t, m.Remove(hmK(2168)), "m.remove(term51XK(2168))")
	expectEqual(t, uint(4), zigUnwrap(t, hmSlot(m, 2976)))
	expect(t, !hmSlot(m, 2168).Valid, "term51XSlot(m, 2168) == null")
	zigTry(t, m.Put(hmK(2168), 3))
	expectEqual(t, uint(3), zigUnwrap(t, hmSlot(m, 2168)))
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacityAdapted: fingerprints 0 and 1 of used slots are not free or tombstone"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacityAdapted_fingerprints_0_and_1_of_used_slots_are_not_free_or_tombstone(t *testing.T) {
	// Key 1136 has fingerprint 0 (home 13), key 1584 fingerprint 1 (home 7).
	box := hmHBoxInit(16)
	m := box.map_()
	zigTry(t, m.Put(hmK(1136), 10))
	zigTry(t, m.Put(hmK(1584), 11))
	expectEqual(t, uint(13), zigUnwrap(t, hmSlot(m, 1136)))
	expectEqual(t, uint(7), zigUnwrap(t, hmSlot(m, 1584)))
	expectEqual(t, uint8(0x80), hmCtl(m, 13))
	expectEqual(t, uint8(0x81), hmCtl(m, 7))
	// A new key homed at slot 7 skips the 0x81 slot (it is not a tombstone).
	expectEqual(t, uint64(7), hmCtx.Hash(hmK(1576))&15)
	zigTry(t, m.Put(hmK(1576), 12))
	expectEqual(t, uint(8), zigUnwrap(t, hmSlot(m, 1576)))
	expectEqual(t, uint16(11), zigUnwrap(t, m.Get(hmK(1584))))
	expectEqual(t, uint16(10), zigUnwrap(t, m.Get(hmK(1136))))
	// Removal turns both into the single tombstone byte 0x01.
	expect(t, m.Remove(hmK(1136)), "m.remove(term51XK(1136))")
	expect(t, m.Remove(hmK(1584)), "m.remove(term51XK(1584))")
	expectEqual(t, uint8(1), hmCtl(m, 13))
	expectEqual(t, uint8(1), hmCtl(m, 7))
	// 1576 is found behind the new tombstone; then a new key homed at 7
	// takes the tombstone at 7.
	expectEqual(t, uint(8), zigUnwrap(t, hmSlot(m, 1576)))
	expectEqual(t, uint64(7), hmCtx.Hash(hmK(1824))&15)
	zigTry(t, m.Put(hmK(1824), 13))
	expectEqual(t, uint(7), zigUnwrap(t, hmSlot(m, 1824)))
}

// Zig: test "term51: HashMapUnmanaged.getIndex: hits, misses and termination without a free slot"
func TestHashMap_term51_HashMapUnmanaged_getIndex_hits_misses_and_termination_without_a_free_slot(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Empty map: always null (even for keys whose home is a tombstone).
	expect(t, !hmSlot(m, 65536).Valid, "term51XSlot(m, 65536) == null")

	for i, k := range hmFull16 {
		zigTry(t, m.Put(hmK(k), HyperlinkId(i)))
	}
	expectEqual(t, uint32(16), m.Count())
	// Every slot used, slots and control bytes pinned.
	for i, k := range hmFull16 {
		s := hmFull16Slots[i]
		fp := hmFull16Fps[i]
		expectEqual(t, s, zigUnwrap(t, hmSlot(m, k)))
		expectEqual(t, uint8(0x80)|fp, hmCtl(m, s))
		expectEqual(t, k, m.keys()[s])
	}
	expectEqual(t, uint(0), hmCountCtl(m, 0))
	// A missing key terminates after visiting every slot once.
	expect(t, !hmSlot(m, 70000).Valid, "term51XSlot(m, 70000) == null")
	expect(t, !hmSlot(m, 0).Valid, "term51XSlot(m, 0) == null")
	// Equal fingerprints (0x3d) but different keys.
	expectEqual(t, uint(5), zigUnwrap(t, hmSlot(m, 65592)))
	expectEqual(t, uint(6), zigUnwrap(t, hmSlot(m, 65600)))
	// Fingerprint 1 slot (65640) wrapped from home 11 to slot 0.
	expectEqual(t, uint8(0x81), hmCtl(m, 0))

	// One tombstone and no free slot: lookups still finish and others stay.
	expect(t, m.Remove(hmK(65576)), "m.remove(term51XK(65576))")
	expectEqual(t, uint8(1), hmCtl(m, 3))
	expectEqual(t, uint(0), hmCountCtl(m, 0))
	expect(t, !hmSlot(m, 70000).Valid, "term51XSlot(m, 70000) == null")
	expect(t, !hmSlot(m, 65576).Valid, "term51XSlot(m, 65576) == null")
	for _, k := range hmFull16 {
		if k == 65576 {
			continue
		}
		expect(t, hmSlot(m, k).Valid, "term51XSlot(m, k) != null")
	}

	// Everything removed but one key: all tombstones except that slot.
	for _, k := range hmFull16 {
		if k != 65536 {
			_ = m.Remove(hmK(k))
		}
	}
	expectEqual(t, uint32(1), m.Count())
	expectEqual(t, uint(15), hmCountCtl(m, 1))
	expectEqual(t, uint(7), zigUnwrap(t, hmSlot(m, 65536)))
	expect(t, !hmSlot(m, 70000).Valid, "term51XSlot(m, 70000) == null")
	expect(t, !hmSlot(m, 65544).Valid, "term51XSlot(m, 65544) == null")

	// Remove the last: size 0 returns null at once; all 16 slots are
	// tombstones.
	expect(t, m.Remove(hmK(65536)), "m.remove(term51XK(65536))")
	expectEqual(t, uint(16), hmCountCtl(m, 1))
	expect(t, !hmSlot(m, 65536).Valid, "term51XSlot(m, 65536) == null")

	// Capacity zero.
	z := hmHBoxInit(0)
	zm := z.map_()
	expect(t, !hmSlot(zm, 1).Valid, "term51XSlot(zm, 1) == null")
}

// Zig: test "term51: HashMapUnmanaged.getIndex: probing sequence under collisions with removals"
func TestHashMap_term51_HashMapUnmanaged_getIndex_probing_sequence_under_collisions_with_removals(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c[0:6] {
		zigTry(t, m.Put(hmK(k), HyperlinkId(i)))
	}
	// Lookup walks 5,6,7,...; found at its own slot.
	for i, k := range c[0:6] {
		expectEqual(t, uint(5+i), zigUnwrap(t, hmSlot(m, k)))
	}
	// Absent key with the same home: walks to the free slot 11, null.
	expect(t, !hmSlot(m, c[6]).Valid, "term51XSlot(m, c[6]) == null")
	// Tombstone in the middle: later keys still found, the removed one not.
	_ = m.Remove(hmK(c[2]))
	expectEqual(t, uint(8), zigUnwrap(t, hmSlot(m, c[3])))
	expectEqual(t, uint(10), zigUnwrap(t, hmSlot(m, c[5])))
	expect(t, !hmSlot(m, c[2]).Valid, "term51XSlot(m, c[2]) == null")
	// Tombstone does not stop a miss before the chain ends.
	expect(t, !hmSlot(m, c[6]).Valid, "term51XSlot(m, c[6]) == null")
	// Removing the rest of the chain leaves tombstones, which never end a
	// search: the first two keys are still found and a miss walks past them.
	_ = m.Remove(hmK(c[5]))
	_ = m.Remove(hmK(c[4]))
	_ = m.Remove(hmK(c[3]))
	expectEqual(t, uint(4), hmCountCtl(m, 1))
	expectEqual(t, uint(10), hmCountCtl(m, 0))
	expect(t, !hmSlot(m, c[3]).Valid, "term51XSlot(m, c[3]) == null")
	expectEqual(t, uint(5), zigUnwrap(t, hmSlot(m, c[0])))
	expectEqual(t, uint(6), zigUnwrap(t, hmSlot(m, c[1])))
	// The next new key takes the first tombstone of the chain.
	zigTry(t, m.Put(hmK(c[6]), 6))
	expectEqual(t, uint(7), zigUnwrap(t, hmSlot(m, c[6])))
}

// Zig: test "term51: HashMapUnmanaged.getContext: value copy of a present key"
func TestHashMap_term51_HashMapUnmanaged_getContext_value_copy_of_a_present_key(t *testing.T) {
	box := hmGBoxInit(32)
	m := box.map_()
	expect(t, !m.GetContext(hmK(4096), hmCtx).Valid, "m.getContext(term51XK(4096), term51XCtx) == null")
	for i := range 20 {
		zigTry(t, m.Put(hmK(uint32(4096+8*i)), hmSl(uint32(i), uint(i+1))))
	}
	for i := range 20 {
		k := hmK(uint32(4096 + 8*i))
		hmExpectSl(t, hmSl(uint32(i), uint(i+1)), zigUnwrap(t, m.GetContext(k, hmCtx)))
		hmExpectSl(t, hmSl(uint32(i), uint(i+1)), zigUnwrap(t, m.Get(k)))
	}
	expect(t, !m.GetContext(hmK(4096+8*20), hmCtx).Valid, "m.getContext(term51XK(4096 + 8 * 20), term51XCtx) == null")
	expect(t, !m.Get(hmK(0)).Valid, "m.get(term51XK(0)) == null")
	// The result is a copy.
	v := zigUnwrap(t, m.Get(hmK(4096)))
	v = hmSl(hmDesc(v).offset, 99) // v.len = 99
	expectEqual(t, uint(1), hmDesc(zigUnwrap(t, m.Get(hmK(4096)))).len)
	_ = v
	// Removed keys are gone.
	_ = m.Remove(hmK(4096))
	expect(t, !m.GetContext(hmK(4096), hmCtx).Valid, "m.getContext(term51XK(4096), term51XCtx) == null")
}

// Zig: test "term51: HashMapUnmanaged.getAdapted: lookup by an adapted key"
func TestHashMap_term51_HashMapUnmanaged_getAdapted_lookup_by_an_adapted_key(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig: `const ctx: term51XRawCtx = .{}`; see the file header.
	ctx := hmCtx
	expect(t, !m.GetAdapted(5, ctx).Valid, "m.getAdapted(@as(u32, 5), ctx) == null")
	for i, k := range hmChain5[0:4] {
		zigTry(t, m.Put(hmK(k), HyperlinkId(i+10)))
	}
	for i, k := range hmChain5[0:4] {
		expectEqual(t, uint16(i+10), zigUnwrap(t, m.GetAdapted(k, ctx)))
	}
	expect(t, !m.GetAdapted(hmChain5[4], ctx).Valid, "m.getAdapted(term51XChain5[4], ctx) == null")
	expect(t, !m.GetAdapted(1, ctx).Valid, "m.getAdapted(@as(u32, 1), ctx) == null")
}

// Zig: test "term51: HashMapUnmanaged.getPtrContext: pointer to the value slot"
func TestHashMap_term51_HashMapUnmanaged_getPtrContext_pointer_to_the_value_slot(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	expect(t, m.GetPtrContext(hmK(8), hmCtx) == nil, "m.getPtrContext(term51XK(8), term51XCtx) == null")
	zigTry(t, m.Put(hmK(4096), hmSl(1, 1)))
	zigTry(t, m.Put(hmK(4104), hmSl(2, 2)))
	p := m.GetPtrContext(hmK(4096), hmCtx)
	expect(t, p != nil, "m.getPtrContext(term51XK(4096), term51XCtx) != null")
	expect(t, &m.values()[zigUnwrap(t, hmSlot(m, 4096))] == p, "@intFromPtr(&m.values()[term51XSlot(m, 4096).?]) == @intFromPtr(p)")
	*p = hmSl(50, 5)
	hmExpectSl(t, hmSl(50, 5), zigUnwrap(t, m.Get(hmK(4096))))
	// Page usage: grow a slice in place through the pointer.
	q := m.GetPtr(hmK(4104))
	expect(t, q != nil, "m.getPtr(term51XK(4104)) != null")
	*q = hmSl(hmDesc(*q).offset, hmDesc(*q).len+3) // q.len += 3
	*q = hmSl(77, hmDesc(*q).len)                  // q.offset.offset = 77
	hmExpectSl(t, hmSl(77, 5), zigUnwrap(t, m.Get(hmK(4104))))
	expect(t, m.GetPtrContext(hmK(4112), hmCtx) == nil, "m.getPtrContext(term51XK(4112), term51XCtx) == null")
	expectEqual(t, uint32(2), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.getPtrAdapted: pointer by adapted key"
func TestHashMap_term51_HashMapUnmanaged_getPtrAdapted_pointer_by_adapted_key(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig: `const ctx: term51XRawCtx = .{}`; see the file header.
	ctx := hmCtx
	zigTry(t, m.Put(hmK(4096), 1))
	p := m.GetPtrAdapted(4096, ctx)
	expect(t, p != nil, "m.getPtrAdapted(@as(u32, 4096), ctx) != null")
	*p = 9
	expectEqual(t, uint16(9), zigUnwrap(t, m.Get(hmK(4096))))
	expect(t, m.GetPtrAdapted(4097, ctx) == nil, "m.getPtrAdapted(@as(u32, 4097), ctx) == null")
}

// Zig: test "term51: HashMapUnmanaged.getEntryContext: key and value pointers of the slot"
func TestHashMap_term51_HashMapUnmanaged_getEntryContext_key_and_value_pointers_of_the_slot(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	expect(t, !m.GetEntryContext(hmK(8), hmCtx).Valid, "m.getEntryContext(term51XK(8), term51XCtx) == null")
	for i := range 10 {
		zigTry(t, m.Put(hmK(uint32(4096+8*i)), hmSl(uint32(i), 1)))
	}
	for i := range 10 {
		k := uint32(4096 + 8*i)
		e := zigUnwrap(t, m.GetEntryContext(hmK(k), hmCtx))
		slot := zigUnwrap(t, hmSlot(m, k))
		expect(t, &m.keys()[slot] == e.key_ptr, "@intFromPtr(&m.keys()[slot]) == @intFromPtr(e.key_ptr)")
		expect(t, &m.values()[slot] == e.value_ptr, "@intFromPtr(&m.values()[slot]) == @intFromPtr(e.value_ptr)")
		expectEqual(t, k, *e.key_ptr)
		hmExpectSl(t, hmSl(uint32(i), 1), *e.value_ptr)
		w := zigUnwrap(t, m.GetEntry(hmK(k)))
		expect(t, e.key_ptr == w.key_ptr, "@intFromPtr(e.key_ptr) == @intFromPtr(w.key_ptr)")
	}
	expect(t, !m.GetEntryContext(hmK(4096+80), hmCtx).Valid, "m.getEntryContext(term51XK(4096 + 80), term51XCtx) == null")
	// The value pointer is writable.
	ev := zigUnwrap(t, m.GetEntry(hmK(4096))).value_ptr
	*ev = hmSl(hmDesc(*ev).offset, 42) // .len = 42
	expectEqual(t, uint(42), hmDesc(zigUnwrap(t, m.Get(hmK(4096)))).len)
}

// Zig: test "term51: HashMapUnmanaged.getEntryAdapted: entry by adapted key"
func TestHashMap_term51_HashMapUnmanaged_getEntryAdapted_entry_by_adapted_key(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig: `const ctx: term51XRawCtx = .{}`; see the file header.
	ctx := hmCtx
	zigTry(t, m.Put(hmK(4096), 3))
	e := zigUnwrap(t, m.GetEntryAdapted(4096, ctx))
	expectEqual(t, uint32(4096), *e.key_ptr)
	expectEqual(t, uint16(3), *e.value_ptr)
	expect(t, !m.GetEntryAdapted(1, ctx).Valid, "m.getEntryAdapted(@as(u32, 1), ctx) == null")
}

// Zig: test "term51: HashMapUnmanaged.containsContext: presence by key"
func TestHashMap_term51_HashMapUnmanaged_containsContext_presence_by_key(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	expect(t, !m.ContainsContext(hmK(4096), hmCtx), "!m.containsContext(term51XK(4096), term51XCtx)")
	zigTry(t, m.Put(hmK(4096), 1))
	expect(t, m.ContainsContext(hmK(4096), hmCtx), "m.containsContext(term51XK(4096), term51XCtx)")
	expect(t, m.Contains(hmK(4096)), "m.contains(term51XK(4096))")
	expect(t, !m.ContainsContext(hmK(4097), hmCtx), "!m.containsContext(term51XK(4097), term51XCtx)")
	_ = m.Remove(hmK(4096))
	expect(t, !m.ContainsContext(hmK(4096), hmCtx), "!m.containsContext(term51XK(4096), term51XCtx)")
	expect(t, !m.Contains(hmK(4096)), "!m.contains(term51XK(4096))")
	// The value does not matter (zero value).
	zigTry(t, m.Put(hmK(8), 0))
	expect(t, m.ContainsContext(hmK(8), hmCtx), "m.containsContext(term51XK(8), term51XCtx)")
}

// Zig: test "term51: HashMapUnmanaged.containsAdapted: presence by adapted key"
func TestHashMap_term51_HashMapUnmanaged_containsAdapted_presence_by_adapted_key(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig: `const ctx: term51XRawCtx = .{}`; see the file header.
	ctx := hmCtx
	expect(t, !m.ContainsAdapted(8, ctx), "!m.containsAdapted(@as(u32, 8), ctx)")
	zigTry(t, m.Put(hmK(8), 0))
	expect(t, m.ContainsAdapted(8, ctx), "m.containsAdapted(@as(u32, 8), ctx)")
	expect(t, !m.ContainsAdapted(9, ctx), "!m.containsAdapted(@as(u32, 9), ctx)")
}

// Zig: test "term51: HashMapUnmanaged.putContext: inserts, updates, refuses when full"
func TestHashMap_term51_HashMapUnmanaged_putContext_inserts_updates_refuses_when_full(t *testing.T) {
	box := hmGBoxInit(8)
	m := box.map_()
	for i := range 8 {
		k := hmK(uint32(4096 + 8*i))
		if i%2 == 0 {
			zigTry(t, m.PutContext(k, hmSl(uint32(i), 1), hmCtx))
		} else {
			zigTry(t, m.Put(k, hmSl(uint32(i), 1)))
		}
	}
	expectEqual(t, uint32(8), m.Count())
	// Full: a new key is refused (both forms), nothing changes.
	hmExpectError(t, ErrOutOfMemory, m.PutContext(hmK(1), hmSl(9, 9), hmCtx))
	hmExpectError(t, ErrOutOfMemory, m.Put(hmK(1), hmSl(9, 9)))
	expectEqual(t, uint32(8), m.Count())
	expect(t, !m.Get(hmK(1)).Valid, "m.get(term51XK(1)) == null")
	// Full: an existing key is updated, in place.
	slot := zigUnwrap(t, hmSlot(m, 4096+24))
	zigTry(t, m.PutContext(hmK(4096+24), hmSl(77, 7), hmCtx))
	zigTry(t, m.Put(hmK(4096), hmSl(78, 8)))
	expectEqual(t, uint32(8), m.Count())
	expectEqual(t, slot, zigUnwrap(t, hmSlot(m, 4096+24)))
	hmExpectSl(t, hmSl(77, 7), zigUnwrap(t, m.Get(hmK(4096+24))))
	hmExpectSl(t, hmSl(78, 8), zigUnwrap(t, m.Get(hmK(4096))))
	// The stored key is the same value.
	expectEqual(t, uint32(4096+24), m.keys()[slot])

	// Capacity zero refuses everything.
	z := hmGBoxInit(0)
	zm := z.map_()
	hmExpectError(t, ErrOutOfMemory, zm.PutContext(hmK(1), hmSl(1, 1), hmCtx))
	hmExpectError(t, ErrOutOfMemory, zm.Put(hmK(1), hmSl(1, 1)))
	expectEqual(t, uint32(0), zm.Count())
}

// Zig: test "term51: HashMapUnmanaged.getOrPutContext: sets the key and reports existing entries"
func TestHashMap_term51_HashMapUnmanaged_getOrPutContext_sets_the_key_and_reports_existing_entries(t *testing.T) {
	box := hmHBoxInit(4)
	m := box.map_()
	{
		gop, err := m.GetOrPutContext(hmK(4096), hmCtx)
		zigTry(t, err)
		expect(t, !gop.found_existing, "!gop.found_existing")
		expectEqual(t, uint32(4096), *gop.key_ptr)
		*gop.value_ptr = 1
	}
	{
		gop, err := m.GetOrPut(hmK(4096))
		zigTry(t, err)
		expect(t, gop.found_existing, "gop.found_existing")
		expectEqual(t, uint16(1), *gop.value_ptr)
		*gop.value_ptr = 2
	}
	expectEqual(t, uint16(2), zigUnwrap(t, m.Get(hmK(4096))))
	expectEqual(t, uint32(1), m.Count())
	for i := 1; i < 4; i++ {
		gop, err := m.GetOrPutContext(hmK(uint32(4096+8*i)), hmCtx)
		zigTry(t, err)
		expect(t, !gop.found_existing, "!gop.found_existing")
		*gop.value_ptr = HyperlinkId(i)
	}
	expectEqual(t, uint32(4), m.Count())
	// Full: existing found without error, new key is refused.
	gop, err := m.GetOrPutContext(hmK(4096+16), hmCtx)
	zigTry(t, err)
	expect(t, gop.found_existing, "gop.found_existing")
	expectEqual(t, uint16(2), *gop.value_ptr)
	_, err = m.GetOrPutContext(hmK(1), hmCtx)
	hmExpectError(t, ErrOutOfMemory, err)
	_, err = m.GetOrPut(hmK(1))
	hmExpectError(t, ErrOutOfMemory, err)
	expectEqual(t, uint32(4), m.Count())
	expect(t, !m.Contains(hmK(1)), "!m.contains(term51XK(1))")

	// Capacity zero.
	z := hmHBoxInit(0)
	zm := z.map_()
	_, err = zm.GetOrPutContext(hmK(1), hmCtx)
	hmExpectError(t, ErrOutOfMemory, err)
}

// Zig: test "term51: HashMapUnmanaged.getOrPutContextAdapted: lookup fallback when the map is full"
func TestHashMap_term51_HashMapUnmanaged_getOrPutContextAdapted_lookup_fallback_when_the_map_is_full(t *testing.T) {
	box := hmHBoxInit(4)
	m := box.map_()
	// Zig: `const ctx: term51XRawCtx = .{}`; see the file header.
	ctx := hmCtx
	for i := range 4 {
		gop, err := m.GetOrPutContextAdapted(uint32(8*i), ctx)
		zigTry(t, err)
		expect(t, !gop.found_existing, "!gop.found_existing")
		*gop.key_ptr = hmK(uint32(8 * i))
		*gop.value_ptr = HyperlinkId(i + 1)
	}
	expectEqual(t, uint32(4), m.Count())
	// Full: existing key still reported via the lookup fallback.
	again, err := m.GetOrPutContextAdapted(16, ctx)
	zigTry(t, err)
	expect(t, again.found_existing, "again.found_existing")
	expectEqual(t, uint16(3), *again.value_ptr)
	expect(t, &m.keys()[zigUnwrap(t, hmSlot(m, 16))] == again.key_ptr, "@intFromPtr(&m.keys()[term51XSlot(m, 16).?]) == @intFromPtr(again.key_ptr)")
	// Full: missing key reports the allocation error.
	_, err = m.GetOrPutContextAdapted(99, ctx)
	hmExpectError(t, ErrOutOfMemory, err)
	// Room again after a removal: found keys still found, new key placed.
	expect(t, m.Remove(hmK(8)), "m.remove(term51XK(8))")
	found, err := m.GetOrPutContextAdapted(24, ctx)
	zigTry(t, err)
	expect(t, found.found_existing, "found.found_existing")
	fresh, err := m.GetOrPutContextAdapted(99, ctx)
	zigTry(t, err)
	expect(t, !fresh.found_existing, "!fresh.found_existing")
	expectEqual(t, uint32(4), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.removeByIndex: slot becomes a tombstone and the size drops"
func TestHashMap_term51_HashMapUnmanaged_removeByIndex_slot_becomes_a_tombstone_and_the_size_drops(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	for i, k := range hmChain5[0:4] {
		zigTry(t, m.Put(hmK(k), HyperlinkId(i)))
	}
	m.removeByIndex(6)
	expectEqual(t, uint8(1), hmCtl(m, 6))
	expectEqual(t, uint32(3), m.Count())
	expect(t, !hmSlot(m, hmChain5[1]).Valid, "term51XSlot(m, term51XChain5[1]) == null")
	expectEqual(t, uint(7), zigUnwrap(t, hmSlot(m, hmChain5[2])))
	m.removeByIndex(5)
	m.removeByIndex(8)
	m.removeByIndex(7)
	expectEqual(t, uint32(0), m.Count())
	for s := uint(5); s < 9; s++ {
		expectEqual(t, uint8(1), hmCtl(m, s))
	}
	for s := range uint(5) {
		expectEqual(t, uint8(0), hmCtl(m, s))
	}
	for s := uint(9); s < 16; s++ {
		expectEqual(t, uint8(0), hmCtl(m, s))
	}
	// The slots are reusable (probe order from the home slot).
	zigTry(t, m.Put(hmK(hmChain5[5]), 5))
	expectEqual(t, uint(5), zigUnwrap(t, hmSlot(m, hmChain5[5])))
}

// Zig: test "term51: HashMapUnmanaged.removeContext: true when removed, false when absent"
func TestHashMap_term51_HashMapUnmanaged_removeContext_true_when_removed_false_when_absent(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	// Empty map.
	expect(t, !m.RemoveContext(hmK(4096), hmCtx), "!m.removeContext(term51XK(4096), term51XCtx)")
	for i, k := range hmChain5[0:5] {
		zigTry(t, m.Put(hmK(k), hmSl(uint32(i), 1)))
	}
	expect(t, m.RemoveContext(hmK(hmChain5[2]), hmCtx), "m.removeContext(term51XK(term51XChain5[2]), term51XCtx)")
	expectEqual(t, uint32(4), m.Count())
	expectEqual(t, uint8(1), hmCtl(m, 7))
	// Second removal and absent keys: false, nothing changes.
	expect(t, !m.RemoveContext(hmK(hmChain5[2]), hmCtx), "!m.removeContext(term51XK(term51XChain5[2]), term51XCtx)")
	expect(t, !m.RemoveContext(hmK(hmChain5[7]), hmCtx), "!m.removeContext(term51XK(term51XChain5[7]), term51XCtx)")
	expectEqual(t, uint32(4), m.Count())
	// Keys behind the tombstone can still be removed.
	expect(t, m.RemoveContext(hmK(hmChain5[4]), hmCtx), "m.removeContext(term51XK(term51XChain5[4]), term51XCtx)")
	expectEqual(t, uint8(1), hmCtl(m, 9))
	expectEqual(t, uint32(3), m.Count())
	// The wrapper.
	expect(t, m.Remove(hmK(hmChain5[0])), "m.remove(term51XK(term51XChain5[0]))")
	expect(t, !m.Remove(hmK(hmChain5[0])), "!m.remove(term51XK(term51XChain5[0]))")
	expectEqual(t, uint32(2), m.Count())
	// Remaining entries intact.
	hmExpectSl(t, hmSl(1, 1), zigUnwrap(t, m.Get(hmK(hmChain5[1]))))
	hmExpectSl(t, hmSl(3, 1), zigUnwrap(t, m.Get(hmK(hmChain5[3]))))
	// A map emptied to zero entries.
	expect(t, m.Remove(hmK(hmChain5[1])), "m.remove(term51XK(term51XChain5[1]))")
	expect(t, m.Remove(hmK(hmChain5[3])), "m.remove(term51XK(term51XChain5[3]))")
	expectEqual(t, uint32(0), m.Count())
	expect(t, !m.Remove(hmK(hmChain5[3])), "!m.remove(term51XK(term51XChain5[3]))")
}

// Zig: test "term51: HashMapUnmanaged.removeAdapted: remove by an adapted key"
func TestHashMap_term51_HashMapUnmanaged_removeAdapted_remove_by_an_adapted_key(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig: `const ctx: term51XRawCtx = .{}`; see the file header.
	ctx := hmCtx
	expect(t, !m.RemoveAdapted(4096, ctx), "!m.removeAdapted(@as(u32, 4096), ctx)")
	zigTry(t, m.Put(hmK(4096), 1))
	zigTry(t, m.Put(hmK(4104), 2))
	expect(t, m.RemoveAdapted(4096, ctx), "m.removeAdapted(@as(u32, 4096), ctx)")
	expect(t, !m.RemoveAdapted(4096, ctx), "!m.removeAdapted(@as(u32, 4096), ctx)")
	expectEqual(t, uint32(1), m.Count())
	expect(t, !m.Contains(hmK(4096)), "!m.contains(term51XK(4096))")
	expectEqual(t, uint16(2), zigUnwrap(t, m.Get(hmK(4104))))
}

// Zig: test "term51: HashMapUnmanaged.removeByPtr: the slot index comes from the key pointer"
func TestHashMap_term51_HashMapUnmanaged_removeByPtr_the_slot_index_comes_from_the_key_pointer(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	for i, k := range hmFull16 {
		zigTry(t, m.Put(hmK(k), HyperlinkId(i)))
	}
	// Remove through getEntry's key pointer in key order; the removed slot is
	// exactly the one the key occupied.
	for i, k := range hmFull16 {
		s := hmFull16Slots[i]
		e := zigUnwrap(t, m.GetEntry(hmK(k)))
		expect(t, &m.keys()[s] == e.key_ptr, "@intFromPtr(&m.keys()[s]) == @intFromPtr(e.key_ptr)")
		m.RemoveByPtr(e.key_ptr)
		expectEqual(t, uint8(1), hmCtl(m, s))
		expectEqual(t, uint32(15-i), m.Count())
		expect(t, !hmSlot(m, k).Valid, "term51XSlot(m, k) == null")
		// Later keys are untouched.
		for j, k2 := range hmFull16[i+1:] {
			expectEqual(t, uint16(i+1+j), zigUnwrap(t, m.Get(hmK(k2))))
		}
	}
	expectEqual(t, uint(16), hmCountCtl(m, 1))
}

// Zig: test "term51: HashMapUnmanaged.removeByPtr: move pattern used by the page (remove then put at the destination)"
func TestHashMap_term51_HashMapUnmanaged_removeByPtr_move_pattern_used_by_the_page_remove_then_put_at_the_destination(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	zigTry(t, m.Put(hmK(4096), hmSl(10, 2)))
	zigTry(t, m.Put(hmK(4104), hmSl(20, 3)))
	before := m.Capacity() - m.Count()
	entry := zigUnwrap(t, m.GetEntry(hmK(4096)))
	value := *entry.value_ptr
	m.RemoveByPtr(entry.key_ptr)
	m.PutAssumeCapacity(hmK(8192), value)
	expect(t, !m.Contains(hmK(4096)), "!m.contains(term51XK(4096))")
	hmExpectSl(t, hmSl(10, 2), zigUnwrap(t, m.Get(hmK(8192))))
	hmExpectSl(t, hmSl(20, 3), zigUnwrap(t, m.Get(hmK(4104))))
	expectEqual(t, uint32(2), m.Count())
	expectEqual(t, before, m.Capacity()-m.Count())
	// On a full map the move still fits: the removal frees the slot.
	full := hmGBoxInit(4)
	fm := full.map_()
	for i := range 4 {
		zigTry(t, fm.Put(hmK(uint32(8*i)), hmSl(uint32(i), 1)))
	}
	e := zigUnwrap(t, fm.GetEntry(hmK(16)))
	v := *e.value_ptr
	fm.RemoveByPtr(e.key_ptr)
	fm.PutAssumeCapacity(hmK(4096), v)
	expectEqual(t, uint32(4), fm.Count())
	hmExpectSl(t, hmSl(2, 1), zigUnwrap(t, fm.Get(hmK(4096))))
	expect(t, !fm.Contains(hmK(16)), "!fm.contains(term51XK(16))")
}

// Zig: test "term51: HashMapUnmanaged.iterator: starts at slot zero over the same map"
func TestHashMap_term51_HashMapUnmanaged_iterator_starts_at_slot_zero_over_the_same_map(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	it := m.Iterator()
	expectEqual(t, uint32(0), it.index)
	// Zig-only: try term51XEq(@intFromPtr(&m), @intFromPtr(it.hm)); Iterator()
	// has a value receiver in Go, so it.hm points at a copy of the handle; it
	// is the same map (same header and same metadata).
	expect(t, it.hm.hdr == m.hdr, "it.hm is the same map as m")
	expect(t, &it.hm.metadata[0] == &m.metadata[0], "it.hm is the same map as m")
}

// Zig: test "term51: HashMapUnmanaged.Iterator.next: entries come in slot order"
func TestHashMap_term51_HashMapUnmanaged_Iterator_next_entries_come_in_slot_order(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	for i, k := range hmFull16 {
		zigTry(t, m.Put(hmK(k), HyperlinkId(i)))
	}
	// Slot order, not insertion order.
	order := [...]uint32{65640, 65656, 65568, 65576, 65616, 65592, 65600, 65536, 65552, 65624, 65648, 65544, 65584, 65560, 65608, 65632}
	it := m.Iterator()
	for s, k := range order {
		e := zigUnwrap(t, it.Next())
		expectEqual(t, k, *e.key_ptr)
		expect(t, &m.keys()[s] == e.key_ptr, "@intFromPtr(&m.keys()[s]) == @intFromPtr(e.key_ptr)")
		expect(t, &m.values()[s] == e.value_ptr, "@intFromPtr(&m.values()[s]) == @intFromPtr(e.value_ptr)")
		// index is one past the slot just returned
		expectEqual(t, uint32(s+1), it.index)
	}
	expect(t, !it.Next().Valid, "it.next() == null")
	expectEqual(t, uint32(16), it.index)
	expect(t, !it.Next().Valid, "it.next() == null")
	expectEqual(t, uint32(16), it.index)
}

// Zig: test "term51: HashMapUnmanaged.Iterator.next: order after removals and reinsertions"
func TestHashMap_term51_HashMapUnmanaged_Iterator_next_order_after_removals_and_reinsertions(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain14
	for _, k := range c[0:4] {
		zigTry(t, m.Put(hmK(k), HyperlinkId(k)))
	}
	// Wrapped chain: slots 14,15,0,1 are visited 0,1,14,15.
	{
		it := m.Iterator()
		want := [...]uint32{c[2], c[3], c[0], c[1]}
		for _, k := range want {
			expectEqual(t, k, *zigUnwrap(t, it.Next()).key_ptr)
		}
		expect(t, !it.Next().Valid, "it.next() == null")
	}
	// Removed entries disappear; tombstones are skipped.
	_ = m.Remove(hmK(c[1]))
	_ = m.Remove(hmK(c[2]))
	{
		it := m.Iterator()
		expectEqual(t, c[3], *zigUnwrap(t, it.Next()).key_ptr)
		expectEqual(t, c[0], *zigUnwrap(t, it.Next()).key_ptr)
		expect(t, !it.Next().Valid, "it.next() == null")
	}
	// Reinsertion recycles tombstones in probe order, which changes the
	// iteration order: c[4] takes slot 15, c[5] takes slot 0.
	zigTry(t, m.Put(hmK(c[4]), HyperlinkId(c[4])))
	zigTry(t, m.Put(hmK(c[5]), HyperlinkId(c[5])))
	{
		it := m.Iterator()
		want := [...]uint32{c[5], c[3], c[0], c[4]}
		for _, k := range want {
			e := zigUnwrap(t, it.Next())
			expectEqual(t, k, *e.key_ptr)
			expectEqual(t, uint16(k), *e.value_ptr)
		}
		expect(t, !it.Next().Valid, "it.next() == null")
	}
	// Values can be modified through the iterator.
	it := m.Iterator()
	for e := it.Next(); e.Valid; e = it.Next() {
		*e.Value.value_ptr += 1
	}
	expectEqual(t, uint16(c[0]+1), zigUnwrap(t, m.Get(hmK(c[0]))))
}

// Zig: test "term51: HashMapUnmanaged.Iterator.next: empty maps and tombstone-only maps yield nothing"
func TestHashMap_term51_HashMapUnmanaged_Iterator_next_empty_maps_and_tombstone_only_maps_yield_nothing(t *testing.T) {
	z := hmHBoxInit(0)
	zm := z.map_()
	zit := zm.Iterator()
	expect(t, !zit.Next().Valid, "zit.next() == null")
	expectEqual(t, uint32(0), zit.index)

	box := hmHBoxInit(8)
	m := box.map_()
	it0 := m.Iterator()
	expect(t, !it0.Next().Valid, "it0.next() == null")
	expectEqual(t, uint32(0), it0.index)

	// Size zero returns before scanning, even with tombstones around.
	zigTry(t, m.Put(hmK(8), 1))
	zigTry(t, m.Put(hmK(16), 2))
	_ = m.Remove(hmK(8))
	_ = m.Remove(hmK(16))
	it1 := m.Iterator()
	expect(t, !it1.Next().Valid, "it1.next() == null")
	expectEqual(t, uint32(0), it1.index)

	// A single entry in the middle: index skips the leading free slots.
	zigTry(t, m.Put(hmK(8), 5))
	slot := zigUnwrap(t, hmSlot(m, 8))
	it2 := m.Iterator()
	e := zigUnwrap(t, it2.Next())
	expectEqual(t, uint16(5), *e.value_ptr)
	expectEqual(t, uint32(slot+1), it2.index)
	expect(t, !it2.Next().Valid, "it2.next() == null")
	expectEqual(t, uint32(8), it2.index)
}

// hmRef is term51XRef: a slot-level reference model of the map.
type hmRef struct {
	ctl  [64]uint8
	keys [64]uint32
	vals [64]uint16
	cap  uint
	size uint
}

func (self *hmRef) home(key uint32) uint {
	return uint(hmCtx.Hash(hmK(key)) & uint64(self.cap-1))
}

func (self *hmRef) find(key uint32) Optional[uint] {
	i := self.home(key)
	for range self.cap {
		if self.ctl[i] == 0 {
			return None[uint]()
		}
		if self.ctl[i]&0x80 != 0 && self.keys[i] == key {
			return Some(i)
		}
		i = (i + 1) % self.cap
	}
	return None[uint]()
}

func (self *hmRef) put(key uint32, val uint16) error {
	if i := self.find(key); i.Valid {
		self.vals[i.Value] = val
		return nil
	}
	if self.size == self.cap {
		return ErrOutOfMemory
	}
	i := self.home(key)
	tomb := None[uint]()
	for self.ctl[i] != 0 {
		if self.ctl[i] == 1 && !tomb.Valid {
			tomb = Some(i)
		}
		i = (i + 1) % self.cap
		if i == self.home(key) {
			break
		}
	}
	at := i
	if tomb.Valid {
		at = tomb.Value
	}
	self.ctl[at] = hmUsedCtl(key)
	self.keys[at] = key
	self.vals[at] = val
	self.size += 1
	return nil
}

func (self *hmRef) remove(key uint32) bool {
	i := self.find(key)
	if !i.Valid {
		return false
	}
	self.ctl[i.Value] = 1
	self.size -= 1
	return true
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacityAdapted: randomized operations match a slot-level reference"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacityAdapted_randomized_operations_match_a_slot_level_reference(t *testing.T) {
	for _, cfg := range [...][2]uint64{{8, 0xa11ce}, {16, 0xb0b}, {64, 0xc0ffee}, {32, 0xd00d}} {
		cap := uint32(cfg[0])
		// Zig: std.Random.Xoshiro256.init(cfg[1]); see the file header.
		r := rand.New(rand.NewPCG(cfg[1], cfg[1]))
		box := hmHBoxInit(cap)
		m := box.map_()
		ref := hmRef{cap: uint(cap)}
		pool := cap + cap/2
		for step := range 4000 {
			key := 4096 + 8*r.Uint32N(pool)
			val := uint16(step & 0xffff)
			switch n := r.IntN(20); {
			case n <= 9:
				if err := ref.put(key, val); err == nil {
					zigTry(t, m.Put(hmK(key), val))
				} else {
					hmExpectError(t, ErrOutOfMemory, m.Put(hmK(key), val))
				}
			case n <= 16:
				expectEqual(t, ref.remove(key), m.Remove(hmK(key)))
			default:
				want := None[uint16]()
				if i := ref.find(key); i.Valid {
					want = Some(ref.vals[i.Value])
				}
				expectEqual(t, want, m.Get(hmK(key)))
			}
			expectEqual(t, uint32(ref.size), m.Count())
			for s := range uint(cap) {
				expectEqual(t, ref.ctl[s], hmCtl(m, s))
				if ref.ctl[s]&0x80 != 0 {
					expectEqual(t, ref.keys[s], m.keys()[s])
					expectEqual(t, ref.vals[s], m.values()[s])
				}
			}
		}
	}
}

// Zig: the term51XDump* helpers and the test "term51: OffsetHashMap: dump
// exact data when TERM51_HASHMAP_DUMP_DIR is set" are not translated (see
// scripts/not-translated-hm.txt); corpus_hashmap_test.go replays the data
// file it writes.

// ---------------------------------------------------------------------------
// Go-only: panic tests. Zig's tests cannot test an assert, an integer
// overflow or an unwrap of null (the test would abort), so each of these
// paths of hash_map.zig is checked here.
// ---------------------------------------------------------------------------

// Go-only: panic test: layoutForCapacity: new_capacity is neither 0 nor a power of two
func TestHashMap_panic_layoutForCapacity_capacity_not_a_power_of_two(t *testing.T) {
	// Zig: assert(new_capacity == 0 or std.math.isPowerOfTwo(new_capacity));
	for _, cap := range []uint32{3, 5, 6, 12, 100, 1<<31 + 1, math.MaxUint32} {
		hmExpectPanic(t, func() { hashMapLayoutForCapacity(cap, hmSizesU32U32) })
	}
}

// Go-only: panic test: OffsetHashMap.layout: capacity is neither 0 nor a power of two
func TestHashMap_panic_OffsetHashMap_layout_capacity_not_a_power_of_two(t *testing.T) {
	// OffsetHashMap.layout is Unmanaged.layoutForCapacity.
	for _, cap := range []uint32{3, 6, 1000} {
		hmExpectPanic(t, func() { HashMapOffsetHashMapLayoutOf(cap, pageGraphemeMapEntrySizes) })
	}
}

// Go-only: panic test: putNoClobberContext: the key is already present
func TestHashMap_panic_putNoClobberContext_key_already_present(t *testing.T) {
	// Zig: assert(!self.containsContext(key, ctx));
	box := hmHBoxInit(8)
	m := box.map_()
	zigTry(t, m.Put(hmK(4096), 1))
	hmExpectPanic(t, func() { _ = m.PutNoClobberContext(hmK(4096), 2, hmCtx) })
	hmExpectPanic(t, func() { _ = m.PutNoClobber(hmK(4096), 2) })
}

// Go-only: panic test: putAssumeCapacityNoClobberContext: the key is already present
func TestHashMap_panic_putAssumeCapacityNoClobberContext_key_already_present(t *testing.T) {
	// Zig: assert(!self.containsContext(key, ctx));
	box := hmHBoxInit(8)
	m := box.map_()
	m.PutAssumeCapacityNoClobber(hmK(4096), 1)
	hmExpectPanic(t, func() { m.PutAssumeCapacityNoClobberContext(hmK(4096), 2, hmCtx) })
	hmExpectPanic(t, func() { m.PutAssumeCapacityNoClobber(hmK(4096), 2) })
}

// Go-only: panic test: putAssumeCapacityNoClobberContext: capacity 0 (capacity() - 1 overflows)
func TestHashMap_panic_putAssumeCapacityNoClobberContext_capacity_zero(t *testing.T) {
	// Zig: const mask = self.capacity() - 1; with capacity 0 this is a u32
	// subtraction that overflows (a safety-checked panic).
	box := hmHBoxInit(0)
	m := box.map_()
	hmExpectPanic(t, func() { m.PutAssumeCapacityNoClobber(hmK(1), 1) })
	hmExpectPanic(t, func() { m.PutAssumeCapacityNoClobberContext(hmK(1), 1, hmCtx) })
}

// Go-only: panic test: getOrPutAssumeCapacityAdapted: capacity 0 (capacity() - 1 overflows)
func TestHashMap_panic_getOrPutAssumeCapacityAdapted_capacity_zero(t *testing.T) {
	// Zig: const mask = self.capacity() - 1; with capacity 0 this is a u32
	// subtraction that overflows. putAssumeCapacity, getOrPutAssumeCapacity
	// and the Context forms go through it.
	box := hmHBoxInit(0)
	m := box.map_()
	hmExpectPanic(t, func() { m.GetOrPutAssumeCapacityAdapted(hmK(1), hmCtx) })
	hmExpectPanic(t, func() { m.GetOrPutAssumeCapacity(hmK(1)) })
	hmExpectPanic(t, func() { m.PutAssumeCapacity(hmK(1), 1) })
	hmExpectPanic(t, func() { m.FetchPutAssumeCapacity(hmK(1), 1) })
}

// Go-only: panic test: ensureUnusedCapacity: count + additional_size overflows u32
func TestHashMap_panic_ensureUnusedCapacity_size_overflow(t *testing.T) {
	// Zig: ensureTotalCapacity(self, self.count() + additional_size) adds two
	// u32 values.
	box := hmHBoxInit(8)
	m := box.map_()
	zigTry(t, m.Put(hmK(8), 1))
	hmExpectPanic(t, func() { _ = m.EnsureUnusedCapacity(math.MaxUint32) })
	// The sum that just fits does not panic (it is refused as too big).
	hmExpectError(t, ErrOutOfMemory, m.EnsureUnusedCapacity(math.MaxUint32-1))
}

// Go-only: panic test: removeByIndex: size is already 0 (the size decrement underflows)
func TestHashMap_panic_removeByIndex_size_underflow(t *testing.T) {
	// Zig: self.header().size -= 1 on a u32 that is 0.
	box := hmHBoxInit(8)
	m := box.map_()
	hmExpectPanic(t, func() { m.removeByIndex(0) })
}

// Go-only: panic test: Iterator.next: index is past the capacity
func TestHashMap_panic_Iterator_next_index_past_capacity(t *testing.T) {
	// Zig: assert(it.index <= it.hm.capacity());
	box := hmHBoxInit(8)
	m := box.map_()
	it := HashMapIterator[SizeOffsetInt, HyperlinkId, hmCtxT]{hm: &m, index: 9}
	hmExpectPanic(t, func() { it.Next() })
	// An index equal to the capacity is allowed.
	ok := HashMapIterator[SizeOffsetInt, HyperlinkId, hmCtxT]{hm: &m, index: 8}
	expect(t, !ok.Next().Valid, "it.next() == null")
}

// Go-only: panic test: count: the map has no metadata
func TestHashMap_panic_count_null_metadata(t *testing.T) {
	// Zig: header() unwraps self.metadata.?, which is null in a default map.
	var m hmHU
	hmExpectPanic(t, func() { m.Count() })
}

// Go-only: panic test: header: the map has no metadata
func TestHashMap_panic_header_null_metadata(t *testing.T) {
	// Zig: self.metadata.? is null in a default map.
	var m hmHU
	hmExpectPanic(t, func() { m.header() })
}

// Go-only: panic test: keys and values: the map has no metadata
func TestHashMap_panic_keys_values_null_metadata(t *testing.T) {
	// Zig: both go through header(), which unwraps self.metadata.?.
	var m hmHU
	hmExpectPanic(t, func() { m.keys() })
	hmExpectPanic(t, func() { m.values() })
}

// Go-only: panic test: getIndex: the map has no metadata
func TestHashMap_panic_getIndex_null_metadata(t *testing.T) {
	// Zig: getIndex reads self.header().size first; get, getPtr, getEntry,
	// contains and remove go through it.
	var m hmHU
	hmExpectPanic(t, func() { m.getIndex(hmK(1), hmCtx) })
	hmExpectPanic(t, func() { m.Get(hmK(1)) })
	hmExpectPanic(t, func() { m.Contains(hmK(1)) })
	hmExpectPanic(t, func() { m.Remove(hmK(1)) })
}

// Go-only: panic test: initMetadatas: the map has no metadata
func TestHashMap_panic_initMetadatas_null_metadata(t *testing.T) {
	// Zig: @memset(@as([*]u8, @ptrCast(self.metadata.?))[0 ..]) unwraps null.
	var m hmHU
	hmExpectPanic(t, func() { m.initMetadatas() })
	// clearRetainingCapacity tests for null first and does nothing.
	m.ClearRetainingCapacity()
}

// Go-only: panic test: growIfNeeded: the map has no metadata
func TestHashMap_panic_growIfNeeded_null_metadata(t *testing.T) {
	// Zig: self.capacity() is 0 for a default map, then self.header().size
	// unwraps null; put, getOrPut and ensureTotalCapacity go through it.
	var m hmHU
	hmExpectPanic(t, func() { _ = m.growIfNeeded(1) })
	hmExpectPanic(t, func() { _ = m.Put(hmK(1), 1) })
}

// Go-only: panic test: Iterator.next: the map has no metadata
func TestHashMap_panic_Iterator_next_null_metadata(t *testing.T) {
	// Zig: the index assertion holds (0 <= 0), then it.hm.header() unwraps null.
	var m hmHU
	it := m.Iterator()
	hmExpectPanic(t, func() { it.Next() })
}

// Go-only: panic test: removeByPtr: the key pointer is not an element of the map's keys
func TestHashMap_panic_removeByPtr_foreign_pointer(t *testing.T) {
	// Zig: key_ptr is assumed to be a valid pointer to a key of the map and
	// the index is found by pointer subtraction; in Go keyIndex must panic
	// for a pointer that is not an element of keys (hash_map.go, keyIndex).
	box := hmHBoxInit(8)
	m := box.map_()
	zigTry(t, m.Put(hmK(8), 1))
	var foreign uint32 = 8
	hmExpectPanic(t, func() { m.RemoveByPtr(&foreign) })
	other := hmHBoxInit(8)
	om := other.map_()
	zigTry(t, om.Put(hmK(8), 1))
	hmExpectPanic(t, func() { m.RemoveByPtr(&om.keys()[0]) })
	expectEqual(t, uint32(1), m.Count())
}
