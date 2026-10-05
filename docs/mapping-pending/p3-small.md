# Pending mapping notes: worktree `p3-small`

Sections ready to merge into `docs/mapping.md`.

## Zig std hashing: `std.hash.Wyhash`, `std.hash.int`, `std.hash.autoHash`

Three new files port the parts of Zig 0.15.2's std (the toolchain that built
the recorded corpus) that `style.zig`, `hyperlink.zig` and `hash_map.zig`
reach. Nothing in them is ported from a Ghostty file. The Go standard
library has nothing equivalent (`hash/maphash` is a different, randomly
seeded function), and the hashes decide when a set or map reports that it
is full (`data/sets/style_hash.txt`, `hyperlink_hash.txt`), so they must be
bit-exact.

| Zig std file | Go file | Prefix |
|---|---|---|
| `lib/std/hash/wyhash.zig` | `terminal/std_hash_wyhash.go` | `stdHashWyhash` |
| `lib/std/hash.zig` (`int`) | `terminal/std_hash.go` | `stdHash` |
| `lib/std/hash/auto_hash.zig` | `terminal/std_hash_auto_hash.go` | `stdHashAutoHash` |

### `std_hash_wyhash.go` (whole of `Wyhash`)

- Struct `Wyhash` is the type `stdHashWyhash` with Zig's field names. Methods
  keep Zig's names in lower case (the type is package-private, as in
  `std_fmt_parse_float.go`): `update`, `final`, `shallowCopy`, `smallKey`,
  `round`, `final0`, `final1`, `final2`. Functions without `self`:
  `stdHashWyhashInit(seed)`, `stdHashWyhashHash(seed, input)`,
  `stdHashWyhashRead(bytes, data)`, `stdHashWyhashMum(a, b *uint64)`,
  `stdHashWyhashMix`. The constant `secret` is `stdHashWyhashSecret`.
- `total_len` (usize, enters the result in `final2`) is `uint64` on every
  platform: the width of the 64-bit build that recorded the data. `buf_len`
  and indexes are `uint`.
- `mum`'s `@as(u128, a) *% b` is `bits.Mul64` (the full product; it cannot
  wrap). `std.mem.readInt(.little)` is `encoding/binary.LittleEndian`.
  `read`'s comptime `bytes` is an ordinary parameter (4 or 8 at every call;
  a general loop covers other widths). `input[i..][0..48]` (a `*const
  [48]u8`) is the Go conversion `(*[48]uint8)(input[i:][:48])`.
- `std.debug.assert` is `assert`. `final1`'s `input_lb.len - start_pos`
  would trap on underflow in Zig; the port asserts `start_pos <= len` before
  it. `undefined` fields are Go zero values (always written before read).
- `inline fn`s are ordinary Go functions.

### `std_hash.go` (`std.hash.int`, u64 only)

- `int` is generic over its integer type and chooses the mixer by bit width
  at compile time. `Style.hash` uses it only on a u64, so the u64
  instantiation is written out: `stdHashIntU64(input uint64) uint64`, which
  calls the private mixer `uint64` (`stdHashUint64`). The unused `mult`
  constant, the `0...2` and `else` prongs and the `uint16`/`uint32` mixers
  are other instantiations and not ported. Go's `*` on uint64 is Zig's `*%`.
- `autoHash`, `autoHashStrat` and `Wyhash` are re-exports in hash.zig; their
  Go counterparts are in the other two files (`std.hash.autoHashStrat` is
  `auto_hash.hash`, i.e. a `stdHashAutoHashHash...` function).

### `std_hash_auto_hash.go` (instantiations reached by `PageEntry.hash`)

- `hash`, `hashPointer`, `hashArray` and `autoHash` dispatch on the key type
  at compile time. Each instantiation the library reaches is written out as
  one Go function named after the Zig function plus the key type, keeping
  the branches that key type takes; the comptime decisions
  (`hasUniqueRepresentation`, `@typeInfo` prong, `typeContainsSlice`) are
  comments.

  | Zig call (in `hyperlink.PageEntry.hash`) | Go |
  |---|---|
  | `autoHash(&h, std.meta.activeTag(self.id))` (enum(u1)) | `stdHashAutoHashAutoHashEnumU1(&h, uint8(tag))` -> `stdHashAutoHashHashEnumU1` |
  | `autoHash(&h, v)` (v: `size.OffsetInt` = u32) | `stdHashAutoHashAutoHashU32(&h, v)` -> `stdHashAutoHashHashU32` |
  | `autoHashStrat(&h, slice, .Deep)` (`[]const u8`) | `stdHashAutoHashHashSliceU8(&h, slice, stdHashAutoHashHashStrategyDeep)` -> `stdHashAutoHashHashPointerSliceU8` -> `stdHashAutoHashHashArraySliceU8` -> `stdHashAutoHashHashU8` per byte, then `stdHashAutoHashHashUsize(len)` |

- `HashStrategy` is `stdHashAutoHashHashStrategy` (`...Shallow`, `...Deep`,
  `...DeepRecursive`) with a `String()` from a name table.
- Bytes hashed: the enum(u1) tag is 1 byte (enums have a unique
  representation, `@sizeOf` is 1); a u32 is its 4 little-endian bytes; a
  `[]const u8` with `.Deep` is every byte through a separate `update` call,
  then the length as a usize. **usize is hashed as 8 bytes on every
  platform** (the recorded data's width); Zig built for 32-bit ARM would
  hash 4 bytes and give different hyperlink hashes (checked: e.g.
  `autoHashStrat("hello", .Deep)` is 0x759b537a1dec968b on aarch64,
  0x2c43e210a5850fd1 on arm).
- `std.mem.asBytes(&key)` is native byte order in Zig; every port target
  (arm64, 32-bit ARM) and the recording build are little-endian, and the Go
  code writes little-endian explicitly.
- The hasher (`anytype`) is always `*std.hash.Wyhash` in the library, so the
  Go parameter is `*stdHashWyhash`.
- `hashPointer` of a slice with `.Shallow` hashes the slice's address
  (`@intFromPtr`), which Go cannot take without `unsafe` and the library
  never does: the Go function panics `not ported` there.
- `stdHashAutoHashHashEnumU1` asserts its argument is 0 or 1 (a u1 cannot
  hold more).
- `hash_map.AutoContext` does not reach this file: every key type it is used
  with (u32, i32, u64, usize, `Offset(Cell)` = packed struct(u32)) has a
  unique representation, so `std.hash_map.getAutoHashFn` returns
  `Wyhash.hash(0, std.mem.asBytes(&key))`, i.e. `stdHashWyhashHash(0, <key
  bytes LE at the Zig width>)` (usize 8 bytes).
- Not ported: every other instantiation (pointers, optionals, arrays,
  vectors, structs, unions, error unions), `typeContainsSlice`.

### Tests

- `std_hash_wyhash_test.go`: all 5 tests of wyhash.zig ("test vectors",
  "test vectors at comptime", "smhasher", "iterative api", "iterative
  maintains last sixteen"). `verify.smhasher` and `verify.iterativeApi`
  (lib/std/hash/verify.zig) are ported into the test file for Wyhash. The
  comptime halves run a second time at run time.
- `std_hash_test.go`: `test int`, u64 line only (the other ten lines use
  other instantiations; they are kept as comments). The unnamed `test { _ =
  ...; }` block has no assertions and is not translated.
- `std_hash_auto_hash_test.go`: none of auto_hash.zig's 12 tests can be
  translated (they hash only key types that are not ported; the list with
  the type each needs is in the file header).
- Go-only tests (values printed by Zig 0.15.2, see "Cross-check"):
  `TestStdHashWyhash_go_only_seeded_inputs_match_zig`,
  `TestStdHash_go_only_int_u64_matches_zig`,
  `TestStdHashAutoHash_go_only_autoHash_u32`, `..._autoHash_enum_u1`,
  `..._enum_u1_out_of_range_panics`, `..._hash_usize_is_8_bytes`,
  `..._autoHashStrat_slice_u8`, `..._slice_shallow_panics`,
  `..._page_entry_sequence_matches_zig`.

### Cross-check against Zig 0.15.2

A Zig program (term51 toolchain, built for aarch64 and
arm-linux-musleabihf, Debug and ReleaseSafe; run natively and under
qemu-arm) generated inputs with SplitMix64 and printed digests (every
result folded into one Wyhash, 8 LE bytes per result). The Go-only tests
recompute the same digests:

- Wyhash: 5000 inputs, random seed, length 0..300; for each, `Wyhash.hash`,
  one `update` + `final`, and `update` in random chunks of 0..63 bytes with a
  `final` after every chunk (all folded). Same digest on all four Zig builds
  (0x314f82606e8f4b36) and in Go native and on 32-bit ARM.
- `std.hash.int(u64)`: 15000 values (x, x & 0xffff, x << 40 for 5000 random
  x) plus 0, max and 0xdeadbeef. Same on all Zig builds and in Go.
- `PageEntry.hash` sequence (tag, implicit u32 or explicit id slice, uri
  slice; 5000 random entries, ids 0..120 bytes, uris 0..300 bytes) plus three
  fixed entries: Go equals 64-bit Zig (0x118773d096266e81). 32-bit Zig
  differs, as expected (usize length is 4 bytes there).
