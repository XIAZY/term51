# Mapping notes from the osc-core port (osc.zig, osc/encoding.zig, osc/parsers.zig)

Sections ready to merge into docs/mapping.md.

## Per-file notes: `osc.zig`

- Replaces the `Parser.zig` note "Until `osc.zig` is ported, `osc.go` is a
  marked placeholder that recognises nothing": `osc.go` is ported;
  `Parser.go` needed no change (its OSC paths, `osc_put`, the exit action of
  `osc_string` calling `end` and the entry action calling `reset`, already
  matched `Parser.zig`).
- `Parser.alloc` (`?Allocator`) is kept as `alloc any`: nil is null, any
  other value means "an allocator was provided". Nothing is allocated
  through it; it only decides what the parser accepts (`ensureAllocator`,
  `writeToAllocating`, the reset of a kitty colour command).
- `Parser.fixed` (`?std.Io.Writer`) and `Parser.allocating`
  (`?std.Io.Writer.Allocating`) are `Optional[...]`; `Parser.writer`
  (`?*std.Io.Writer`) is a Go pointer to `fixed.Value` or
  `allocating.Value.writer`, as in Zig (a copy of the parser after a writer
  was set up still points into the old copy, as in Zig).
- `std.Io.Writer.Allocating.initCapacity` cannot fail in Go; its `catch`
  branch in `writeToAllocating` (fall back to the fixed buffer) is kept
  but unreachable.
- `reset`: `v.deinit(self.alloc orelse break)` on a kitty colour command is
  `if self.alloc != nil { ...KittyColorProtocol.Deinit() }` (the allocator
  argument is dropped).
- Zig's exhaustive `switch` statements over `State` and `Command` get a
  `default: panic("unreachable")` (a value outside the enum traps in Zig).
- The Valgrind blocks of `init` and `reset` are not ported (they only mark
  `buffer` undefined); Go zeroes `buffer`.
- `ensureAllocator` logs with `warn("osc", ...)`; Zig's `{t}` of the state
  is `OSCParserState.String()`.
- `Terminator` keeps Zig's `string` as `String() []uint8`; the tag name is
  `TagName()`; `format` is `Format() string`, the terminator bytes.

## `std.Io.Writer` and `std.Io.Writer.Allocating` stand-ins (`oscWriter`)

- `oscWriter` has Zig's `buffer` and `end` and one more field, `vtable`,
  which stands for Zig's vtable pointer (fixed or Allocating). Go cannot
  recover the Allocating that embeds a writer (`@fieldParentPtr`), so the
  writer itself says which drain it uses. Three values: `Fixed` (set by
  `oscWriterFixed`), `Allocating` (set by `oscWriterAllocatingInit`,
  `...InitCapacity` and on the first growth) and `Unset`, the zero value a
  composite literal gets: then a nil buffer means "an Allocating writer
  that has not allocated yet" and any other buffer means a fixed writer.
  This is what makes both test idioms behave as in Zig:
  `oscWriter{buffer: buf[:]}` is `.fixed(&buf)` and the zero
  `oscWriterAllocating` is `Allocating.init`. Limit: an
  `oscWriter{}` with a nil buffer is treated as an empty Allocating writer
  (in Zig `.fixed(&.{})` fails every write); use `oscWriterFixed(nil)` for
  that.
- Methods ported from std.Io.Writer with Zig's code and names: `write`,
  `writeAll`, `writeByte` (including the `while ... else` of writeByte),
  `buffered`, and `drain` dispatching to `oscWriterFixedDrain`
  (`Writer.fixedDrain`, including its partial copy before WriteFailed and
  its `assert(splat >= dest.len)`) or `oscWriterAllocatingDrain`
  (`Allocating.drain`). Others (`print`, `splatByteAll`, ...) are for the
  files that need them to add, same rule.
- `Allocating` methods: `init`, `initCapacity` (returns a nil error),
  `deinit` (drops the buffer; Zig leaves the value undefined),
  `toOwnedSlice` (copy of exactly the written bytes, writer left empty),
  `written`, `clearRetainingCapacity` (`end = 0`, Zig's
  `shrinkRetainingCapacity(0)`).
- Growth follows `ArrayList(u8).ensureUnusedCapacity` / `growCapacity`
  (`new +|= new / 2 + init_capacity`, init_capacity 128 = the 64-bit cache
  line). Not observable except through the buffer length; kept so the
  buffer always has the spare capacity the NUL-terminated (`[:0]`) results
  need, exactly when Zig has it. The `+|=` cannot saturate on any reachable
  size and is a plain `+=`.

## `{any}` of an OSC command (`OSCCommand.zigFmtAny`)

- `Action.format` prints an `osc_dispatch` payload with
  `printValue("any", .{}, cmd, 3)`. `zigFmtAny` reproduces Zig 0.15.2's
  `printValue` for each variant at the depths it reaches: union at 3,
  payload at 2, payload fields at 1, their fields at 0 (struct ->
  `.{ ... }`, slice/array -> `{ ... }`; integers, bools, enums, null at any
  depth). Helpers: `zigFmtAnyStruct`, `zigFmtAnyBytes`, `zigFmtAnyElided`,
  `zigFmtAnyOptionalBool`.
- Two std containers show internals: `color.List` (`SegmentedList(Request,
  2)`) prints `.{ .prealloc_segment = { ... }, .dynamic_segments = { ... },
  .len = N }`; the kitty colour `std.ArrayList(Request)` prints
  `.{ .items = { ... }, .capacity = C }`. Go's slice capacity is not Zig's,
  so `oscKittyColorListCapacity(n)` computes Zig's: the parser starts from
  `.empty` and appends one at a time, so C is the first value of
  growCapacity's sequence with init_capacity = 128 / @sizeOf(Request) =
  128 / 12 = 10 that reaches n (0, 10, 25, 47, ...).
- Checked against upstream's `Action.format` output (built from the pinned
  source with the term51 Zig 0.15.2 toolchain) for every variant whose
  payload enums are already ported, and for the list capacities at 0, 2,
  11 and 26 requests. The type-name prefix is `terminal.Parser.Action`
  (Parser.go's existing choice), upstream prints
  `ghostty.terminal.Parser.Action`.

## `osc/encoding.zig`

- `isSafeUtf8`: `std.unicode.Utf8View.init` is `utf8.Valid` (mapping rule
  5: a standard-library equivalent behind upstream's name), then a range
  over the string yields Utf8View's code points. Checked against the Zig
  function on all 16,777,216 three-byte strings and 16,777,216 four-byte
  strings (lead bytes F0-F7, a spread of continuation and non-continuation
  bytes): identical.

## `osc/parsers.zig`

- Namespace file only; nothing to port (header updated).
