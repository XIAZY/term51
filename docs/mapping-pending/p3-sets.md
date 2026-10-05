# Mapping notes: `ref_counted_set.zig`, `style.zig`, `hyperlink.zig` (branch impl/p3-sets)

Ready to merge into docs/mapping.md (suggested place: a new "Sets module"
section after the stream module). The generic form, the Go-memory rule and
the byte arithmetic are the wireframe's (headers of the three files);
unchanged except where noted.

## `ref_counted_set.zig`

File: `terminal/ref_counted_set.go`.

| Zig | Go | Notes |
|---|---|---|
| `if (comptime @hasDecl(Context, "deleted")) ctx.deleted(v)` | `if self.has_deleted { any(ctx).(RefCountedSetContextDeleted[T]).Deleted(v) }` | **added field** `has_deleted bool`, decided once in `RefCountedSetInit` (same ruling as `vt_raw` in stream.go). The interface conversion at the call is what Go needs to reach a method outside the constraint; it runs only when the context has the hook (hyperlink set), never for the style set |
| `.{}` of an Item (`items[id] = .{}`, `item.* = .{}`, `@memset(items, .{})`) | `RefCountedSetItemInit[T, Id, Ref]()` | bucket = `^Id(0)` (maxInt) |
| `std.math.maxInt(Id)` | `^Id(0)` | |
| `@sizeOf(Id)` | `bits.Len64(uint64(^Id(0))) / 8` | 1, 2 or 4 |
| `std.math.ceilPowerOfTwoAssert(usize, cap)` | `uint(1) << bits.Len(cap-1)` after `assert(bits.Len(cap-1) < bits.UintSize)` | the assert can only fire on a 32-bit uint with a u32 Id |
| `std.math.log2_int(usize, x)` | `bits.Len(x) - 1` | |
| `std.mem.alignForward(usize, x, a)` | `(x + a - 1) &^ (a - 1)` after asserting `a` is a power of two | std asserts the alignment |
| `@intFromFloat(@ceil(@as(f64, n + 1) / load_factor))` | `uint(math.Ceil(float64(n+1) / RefCountedSetLoadFactor))` | float kept as in Zig: 0.8125 = 13/16 exactly, IEEE division is correctly rounded in both languages, so the integer results are identical (`data/sets/capacities.txt`, all 65,537 `c` and `l` lines, pass natively and on 32-bit ARM). Asserts for Zig's traps: `n + 1` overflow, result not fitting usize |
| `@intFromFloat(load_factor * @as(f64, table_cap))` | `uint(float64(RefCountedSetLoadFactor * float64(table_cap)))` | one product, explicit float64() against fusion (none possible: no add) |
| `@as(Id, @intFromFloat(@as(f64, cap) * 0.9))` (rehash threshold) | `Id(float64(float64(cap) * rehash_threshold))` with an assert that it fits Id | the untyped 0.9 rounds to the same f64 as Zig's comptime_float coerced to f64 |
| `(p +% 1) & mask`, `(hash +% i) & mask` | plain Go `+` on the same-width unsigned type | wrapping as in Zig |
| `for (0..self.max_psl + 1) \|i\|` | `for i := uint(0); i < uint(self.max_psl)+1; i++` | max_psl <= 31 (psl_stats bound), so the Id addition cannot overflow |
| `if (comptime std.debug.runtime_safety) assert(lookup == null)` in `insert` | `assert(!self.LookupContext(value, ctx).Valid)` | the reference is a safe build; costs one extra lookup per insert (see optimisation list) |
| `assertIntegrity` | empty body, `_ = ctx` | upstream's body is `if (false and ...)`, never compiled |
| `@branchHint(.cold)` / `.unlikely` | comment `// Zig: @branchHint(...)` | |
| `self.psl_stats.len - 1` | `len(self.psl_stats)-1` | |

Run-time checks added for Zig's overflow traps (each `assert` sits right
before the operation and is commented `// Zig: += traps on overflow`):
`meta.ref += 1` (add, addWithId, use), `meta.ref += n` (useMultiple),
`living -= 1` (release, releaseMultiple), `held_item.meta.psl += 1`
(insert). Not added where the value is bounded by an invariant of the set
itself: `next_id += 1` (`next_id < layout.cap`), `psl_stats[...] +=/-= 1`,
`items[...].meta.psl -= 1` (guarded by `> 0`), `max_psl -= 1` (guarded).
`releaseMultiple`: Zig's `item.meta.ref -= n` mixes `RefCountInt` and `Id`;
Go compares as uint64 and subtracts `Ref(n)` (n fits Ref: Zig rejects the
instantiation otherwise).

Zero capacity: `LookupContext` returns null when `layout.table_cap == 0`
(Go-only guard, comment in place). Zig reads `table[0]` from the memory
that follows (the test fixture keeps spare zero bytes). Every add on such a
set returns `ErrOutOfMemory` through upstream's own `next_id >= cap` check
(`living 0 < @intFromFloat(0 * 0.9)` is false), so no other guard is
needed; `insert` (whose `table_cap - 1` would underflow) is never reached.

Upstream quirk kept: `deleteItem` returns early only when
`bucket > table_cap` (not `>=`), so an item in the default state with
`bucket == maxInt(Id) == table_cap - 1` (u16 set laid out for 65,536) would
go on to `assert(table[bucket] == id)`.

Tests: ref_counted_set_test.go 79 + corpus_sets_test.go 1
(TestRefCountedSet_corpus_capacities), all pass natively and on 32-bit ARM.

## `style.zig`

File: `terminal/style.go`.

| Zig | Go | Notes |
|---|---|---|
| `Color.eql`: `@as(Tag, self) == @as(Tag, other) and switch (self) {...}` | `if self.Tag != other.Tag { return false }` then the switch | same short-circuit |
| `Color.format(self, fmt, options, writer)` (old 4-argument signature, so Zig 0.15 never calls it through `{f}`/`{any}`) | `Format() string` | `{}` of a u8 is decimal (`itoa`) |
| `Style.format`: `inline for (std.meta.fields(Style))` / `(std.meta.fields(Flags))` | the fields written out in declaration order: a loop over the three colours, then the `flags` block (eight bools in order, then `underline` and `_padding`, which are not bool) | `!std.meta.eql(value, d_val)` on a Color is `Color.Eql` (tag, then active payload: what std.meta.eql does for a tagged union); for the enum and the u5 it is `!=` |
| `{any}` of a Color (`printValue`, Zig 0.15.2, depth 3) | local closure `fmtAny` in `Style.Format` | `.{ .palette = 5 }`, `.{ .rgb = .{ .r = 1, .g = 2, .b = 3 } }`, `.{ .none = void }` (never printed: none is the default) |
| `{any}` of `sgr.Attribute.Underline` | `"." + underline.String()` | `printEnumExhaustive`; an out-of-range Go value panics in String() (Zig: unreachable) |
| `{any}` of the u5 `_padding` | `itoa` | |
| `VTFormatter.format(self, writer) !void` / `HtmlFormatter.format` | `Format() string` (wireframe) whose body is the Zig body as a local `format := func(writer *oscWriter) error`, run on an `oscWriterAllocating`; a non-nil error is `panic("unreachable")` (an allocating writer never fails) | keeps every `try` as `if err := ...; err != nil { return err }` |
| `writer.print("\x1b[{d};5;{d}m", .{...})` etc. in the two `formatColor` | one `writer.writeAll` of the same bytes built with `itoa` (Zig's print line in a comment above) | std's print formats each integer into a local buffer and writes it with writeAll, so the byte stream is the same; with a fixed writer that runs out, `fixedDrain` fills the buffer to the brim either way, so even the partial result is identical |
| `switch (bold) { .bright => {}, .color => \|v\| break :default v.toTerminalRGB() }` (labeled blocks `default:`, `palette:`, `rgb:` in `fg`) | `return` from the Go switch arm, `// break :default` | each labeled block is a whole prong of the `return switch` |
| `@intFromEnum(color.Name.bright_black)` | `uint8(ColorNameBrightBlack)` in a local const | |
| `PackedStyle.Data.fromColor`: `inline else => \|v, t\| @unionInit(Data, @tagName(t), ...)` | switch on the tag returning the 24-bit value of that view | none 0; palette idx in bits 0..7; rgb r, g<<8, b<<16 |
| `@bitCast(packed_style)` to `[2]u64` | `stylePackedStyle.bitCast` (wireframe stub) | the fields shifted into place; `underline` masked to 3 bits, `_padding` to 5 and each Data to 24, so the Go values that the Zig widths cannot hold are dropped as the packed layout would (they cannot exist in Zig) |
| `bgCell` `.content = .{ .color_palette = idx }` | `PageCellContent{color_palette: idx}` | the other views zero (Zig: the rest of the 24 bits of the packed union are not set) |
| `Style.hash`: `@call(.always_inline, std.hash.int, .{wide[0] ^ wide[1]})` | `stdHashIntU64(wide[0] ^ wide[1])` (std_hash.go) | `data/sets/style_hash.txt`: all 4,096 styles match, natively and on 32-bit ARM (TestStyle_corpus_hash) |

`StyleColorTag.String` uses a names array (`styleColorTagNames`), as sgr.go.

## `hyperlink.zig`

File: `terminal/hyperlink.go`.

| Zig | Go | Notes |
|---|---|---|
| `Hyperlink.deinit(alloc)` | `Deinit()`: the switch kept, nothing freed | stub kept because the tests call it (wireframe) |
| `alloc.dupe(u8, s)` | `make([]uint8, len(s))` + `copy` | never fails; a duplicated empty slice is empty and non-nil |
| `errdefer` after a successful step | the cleanup written before the later error return, `// Zig: errdefer ...` (the `osc_parsers_color.go` idiom) | `PageEntry.dupe`: an allocation failure of the explicit id frees the uri copy. The errdefers with nothing failing after them are comments only |
| `.{ .offset = size.getOffset(u8, dst_page.memory, &buf[0]), .len = uri.len }` | `buf[0:len(uri)]` | the allocation from its first byte; its capacity reaches the end of the last chunk (BitmapAllocator requirement), which nothing here uses |
| `x.slice(base)` / `offset.ptr(base)[0..len]` | `x` itself | Go-memory rule |
| `self.page.?` / `(self.src_page orelse self.page.?)` in the Set context | `assert(self.page != nil)` (for `hash`, only when `src_page` is nil) | Zig's "attempt to use null value" panic; Go-only panic tests `TestHyperlink_panic_SetContext_*` |
| `std.meta.activeTag(x)` | `x.Tag` | |
| `std.mem.eql(u8, a, b)` | `bytes.Equal(a, b)` | |
| `PageEntry.hash`: `Wyhash.init(0)`, `autoHash(&h, activeTag(id))`, `autoHash(&h, v)` (implicit, u32), `autoHashStrat(&h, slice, .Deep)` (explicit id, then uri), `final()` | `stdHashWyhashInit(0)`, `stdHashAutoHashAutoHashEnumU1(&h, uint8(tag))` (the tag is an enum(u1): explicit 0, implicit 1), `stdHashAutoHashAutoHashU32`, `stdHashAutoHashHashSliceU8(&h, s, stdHashAutoHashHashStrategyDeep)`, `h.final()` | `data/sets/hyperlink_hash.txt` has no Go test (the Zig dump test is not translated, and no corpus test reads the file); checked with a throw-away test (not committed): all 3,000 lines match natively and on 32-bit ARM. Suggest adding a corpus test like TestStyle_corpus_hash |

Tag `String()` methods use names arrays (`hyperlinkHyperlinkIdTagNames`,
`hyperlinkPageEntryIdTagNames`).
