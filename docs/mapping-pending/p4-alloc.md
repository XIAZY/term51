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
