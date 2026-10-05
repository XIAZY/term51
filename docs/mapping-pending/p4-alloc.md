# Mapping notes from impl/p4-alloc (bitmap_allocator, hash_map)

## `bitmap_allocator.zig`

File: `terminal/bitmap_allocator.go`.

| Zig | Go | Notes |
|---|---|---|
| `comptime assert(std.math.isPowerOfTwo(chunk_size))` | `assert(chunk_size != 0 && chunk_size&(chunk_size-1) == 0)` in `BitmapAllocatorInit` | run time in Go (Go-only panic test) |
| `@sizeOf(T)`, `@alignOf(T)` (comptime) | `reflect.TypeFor[T]().Size()` / `.Align()`, read ONCE in `BitmapAllocatorInit` into the **added fields** `elem_size`, `elem_align` (the `has_deleted` precedent) | `BitmapAllocatorBytesRequired` has no receiver and reads `@sizeOf(T)` per call (`bitmapAllocatorSizeOf`). Go sizes equal Zig's for every element type used (u8, u16, u21 = uint32, u64, [3]u8). Go's alignment differs only for 8-byte scalars on 32-bit ARM (Go 4, Zig 8), read only by alloc's `chunk_size % @alignOf(T)` assertion with a chunk size below 8; the page uses 16 and 32 |
| `@intFromPtr(slice.ptr) - @intFromPtr(chunks)` (free, isAllocated) | `reflect.ValueOf(slice).Pointer()` minus the same for `chunks` (`bitmapAllocatorAddr`), in the helper `chunkIndexOf` (not a Zig declaration; the shared expression of free and isAllocated) | addresses are only subtracted, never turned back into pointers; no package `unsafe`. Zig's traps kept: `-` underflow (slice before `chunks`, or a nil slice) and `@divExact` (not on a chunk boundary). Go-only guard: a start past the end of `chunks` panics (Zig would write bitmap words through the unchecked `[*]u64`) |
| `bitmaps[i] \|= ~@as(u64, 0) >> @intCast(64 - bits) << @intCast(bit)` with `bits == 0` (empty slice) | `assert(bits > 0)` before the shift | Zig's `@intCast(64)` to u6 traps; Go would shift by 64 and continue (Go-only panic test) |
| `for (1..n)` in findFreeChunks with `n == 0` | `assert(n >= 1)` inside the per-bitmap loop, where Zig's `for` is | so an empty bitmap list still returns null for n = 0, as in Zig |
| findFreeChunks, large path: the read of `bitmaps[i]` after the middle loop | `if i >= len(bitmaps) { return None }` | the term51 reference fix (`patches-reference/0001`); the pinned upstream aborts (out-of-bounds read at bitmap_allocator.zig:281). Tests `TestBitmapAllocator_findFreeChunks_large_request_past_last_bitmap` and `..._random_bitmaps_skipped_inputs_no_space` |
| `continue :search` from the inner `while (rem > 64) : (i += 1)` | labeled `continue search` | Go's labeled continue skips the inner post statement, as Zig's does |
| `std.math.mul(usize, @sizeOf(T), n) catch return error.OutOfMemory` | `bits.Mul64`, high word non-zero is `ErrOutOfMemory` | 64-bit as the reference |
| `std.math.divCeil(usize, byte_count, chunk_size)` | `(byte_count-1)/chunk_size + 1` for a positive numerator, else `byte_count/chunk_size` | std's unsigned branch; its error (zero denominator) cannot happen |
| `findFreeChunks(..., chunk_count)` with `chunk_count` a 64-bit usize | Go-only: a count above `math.MaxUint` returns `ErrOutOfMemory` before the call | only on a 32-bit target; such a count exceeds every chunk the bitmaps hold, so findFreeChunks would return null |
| `ptr[0..n]` at `&chunks[idx * chunk_size]` | `chunks[start : start+n : end]`, `start`/`end` the byte offsets of the first chunk and of the end of the last chunk divided by `@sizeOf(T)` | capacity reaches the end of the last chunk (wireframe REQUIREMENT). Go-only assert: the chunk starts on a whole element (an element size that does not divide the chunk size cannot be a Go `[]T`; never with the page's types) |
| `std.mem.alignForward(usize, x, a)` | `bitmapAllocatorAlignForward` (not a Zig declaration) | asserts `a` is a power of two (std) and that `x + (a - 1)` does not overflow (Zig's `+` traps) |
| `a * b` on usize in layout/bytesRequired | `bitmapAllocatorMul` (not a Zig declaration) | asserts no overflow (Zig's `*` traps) |
| `@divExact` whose exactness follows from an alignForward just before | plain `/`, comment `// @divExact: exact by construction` | |
| `@popCount`, `@clz`, `@ctz` | `bits.OnesCount64`, `bits.LeadingZeros64`, `bits.TrailingZeros64` | |
| `std.log.warn("bm={b} idx={}", ...)` (dumpBitmaps) | `warn("default", "bm=%b idx=%d", ...)` | unscoped std.log is the `default` scope |
| `comptime assert(@import("builtin").is_test)` (isAllocated) | not checked | the function is unexported and only tests call it |
| `init`: `.chunks = buf.member(u8, l.chunks_start)` | `chunks = make([]T, bitmap_count * 64 * chunk_size / @sizeOf(T))` | what the bitmaps can hand out (file header) |

Tests: bitmap_allocator_test.go 74 + corpus_bitmap_test.go 3, all pass
natively and on 32-bit ARM (qemu).

## `hash_map.zig`

File: `terminal/hash_map.go`. The wireframe's structure (metadata slice +
shared `hdr` pointer, value receivers, `hashMapEntrySizes`, `keyIndex`) is
kept unchanged; no signature changed.

| Zig | Go | Notes |
|---|---|---|
| `AutoContext.hash` = `getAutoHashFn(K)`: `Wyhash.hash(0, std.mem.asBytes(&key))` (every key type used has a unique representation) | type switch on `any(key)` writing the key little-endian at its Zig width into an 8-byte stack buffer, then `stdHashWyhashHash(0, buf[:n])` | u8/i8 1, u16/i16 2, u32/i32 4, u64/i64 8 bytes; Go `uint`/`int` (usize/isize) always 8 (int sign-extended). Any other key type panics `not ported`. The switch is a per-call run-time step standing for Zig's comptime width: Go cannot specialise a method of the empty generic struct `hashMapAutoContext[K]` per type argument, and the struct has no field to hold a choice made once. A defined type (e.g. `type X uint32`) would hit the default; the library's keys are `SizeOffsetInt` (alias of uint32) |
| `AutoContext.eql` = `std.meta.eql(a, b)` | `a == b` | integers |
| `Metadata` packed struct, `@as(u8, @bitCast(self))` in isFree / isTombstone | method `bitCast()` (not a Zig declaration): `fingerprint&0x7f \| (used&1)<<7` | the masks drop what a u7 / u1 cannot hold |
| `takeFingerprint`: `@truncate(hash >> (64 - 7))` | `uint8(hash >> 57)` | |
| `self.metadata.?` (null unwrap, a safety panic) | `assert(self.metadata != nil)` in `header()` and `initMetadatas()`; `removeByIndex` and the probe loops index the slice (a nil slice panics on the index) | Go-only panic tests `TestHashMap_panic_*_null_metadata` |
| `self.metadata.? + idx` then `metadata[0]` (unchecked many-pointer) | `self.metadata[idx]` | idx is masked below capacity = len(metadata) |
| `const mask = self.capacity() - 1` (u32) | `assert(capacity >= 1)` before the subtraction | in putAssumeCapacityNoClobberContext, getIndex, getOrPutAssumeCapacityAdapted (capacity-0 panic tests) |
| `self.capacity() - self.header().size` in growIfNeeded | `assert(capacity >= size)` | Go-only test: a full map overfilled by an assume-capacity put |
| `self.count() + additional_size` in ensureUnusedCapacity | `assert(additional_size <= MaxUint32 - count)` | |
| `header().size += 1` / `-= 1` (u32) | assert before (`< MaxUint32` / `> 0`) | removeByIndex panic test |
| `x.* = undefined` (fetchRemoveAdapted, removeByIndex, getOrPutAssumeCapacityAdapted) | Go zero value of K / V | mapping table: `undefined` is the zero value |
| `@memset(metadata[0 .. capacity], 0)` (initMetadatas) | `clear(self.metadata[0:self.Capacity()])` | |
| Iterator `while (metadata != end) : ({ metadata += 1; it.index += 1; })` | `for ; len(metadata) != 0; metadata, it.index = metadata[1:], it.index+1` over `metadata[it.index:cap]` | the return inside the body skips the post statement as in Zig |
| FieldIterator `metadata += 1; items += 1` | reslice `[1:]` of both | |
| keyIterator / valueIterator with null metadata: `metadata` and `items` `undefined` | nil slices, `len: 0` | |
| `init`: metadata at `buf + @sizeOf(Header)`, keys/values at `layout.keys_start` / `vals_start` | `make` of each, `layout.capacity` long; header allocated and pointed at by `hdr` | capacity 0 gives a non-nil empty metadata slice (wireframe REQUIREMENT) |
| `OffsetHashMap.init`: `getOffset(..., m.metadata.?)`; `map(base)`: `self.metadata.ptr(base)` | copy `metadata` and `hdr` from / to the unmanaged map | |
| `removeByPtr`: `if (@sizeOf(K) > 0) (@intFromPtr(key_ptr) - @intFromPtr(self.keys())) / @sizeOf(K) else 0` | `keyIndex` holds the whole expression, with package `unsafe` as the wireframe allows (`unsafe.Sizeof(*key_ptr)`, `unsafe.Pointer`, `unsafe.SliceData`) | panics when the pointer is before `keys` (Zig's `-` trap) or the element at the computed index is not `key_ptr` (wireframe REQUIREMENT; Go-only foreign-pointer test) |
| comptime `key_align` / `val_align` (`1` for a zero-sized type), `header_align` (4), `base_align` (max) | computed at the top of `hashMapLayoutForCapacity` from `hashMapEntrySizes` | the function is the only user; a page computes it once per layout |
| `std.mem.alignForward(usize, x, a)` in layoutForCapacity | local closure `alignForward`, asserting a power-of-two alignment (std) | no overflow possible: cap <= 2^31 |
| `assert(new_capacity == 0 or std.math.isPowerOfTwo(new_capacity))` | `assert(new_capacity == 0 \|\| new_capacity&(new_capacity-1) == 0)` | Go-only panic tests (layoutForCapacity, OffsetHashMap.layout) |
| The Context-less wrappers (`put`, `get`, ... : compile error unless `@sizeOf(Context) == 0`, then `undefined` context) | `var ctx C` (zero value) | |
| `getOrPutAdapted`'s compile-time check | comment only | |

Tests: hash_map_test.go 101 + hash_map_2_test.go 51 +
corpus_hashmap_test.go 1 (data/hashmap/hashmap_data.txt), all pass natively
and on 32-bit ARM (qemu).
