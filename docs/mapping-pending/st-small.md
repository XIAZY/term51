### Small terminal files: `ansi.zig`, `charsets.zig`, `modes.zig`, `device_status.zig`, `kitty/key.zig`

Files: `ansi.go`, `charsets.go`, `modes.go`, `device_status.go`,
`kitty_key.go`. Recorded while porting them (2026-10-05).

#### Language features (additions to the table)

| Zig | Go | Notes |
|---|---|---|
| `packed struct(uN)` whose bit layout is observable (`ModeTag`, `device_status.Tag`, `kitty.key.Flags`) | ordinary Go struct plus the wireframe's conversion pair (`Int()` = `@bitCast` to the backing integer, `...FromInt` = `@bitCast` back) | the conversions place each field at Zig's bit offset (first field = bit 0). A field narrower than its Go type (`u15` value) and a `u5` argument are asserted to fit (`assert(value <= 0x7FFF)`, `assert(v <= 0x1F)`): such values cannot exist in Zig, so the port stops instead of dropping bits. The reverse cast from the full backing integer (`ModesModeTagFromInt`) needs no check: every u16 is a valid ModeTag. |
| `packed struct` used only as a set of bools (`ModePacked`) | ordinary Go struct of bools, fields in Zig's order | no Go code depends on the bit layout; the tests check the field order with reflect (the counterpart of `@bitOffsetOf`) |
| Type generated at compile time from a table (`Mode`, `ModePacked`, `device_status.Request`, via `@Type`) | the declarations written out (wireframe), plus the part of the generation that is data computed from the table at start-up | `Mode`/`Request`: the enum field list (name, value = the Tag bit-cast) is built by `modesModeInit` / `deviceStatusRequestInit` from `entries` into `modesModeFields` / `deviceStatusRequestFields`; `String()` and `ModesModeValues()` read it. The written-out constants equal what it computes (the tests check all of them). `ModePacked`'s field defaults (`.default_value_ptr = &entry.default`) are set by `ModesModePackedInit` in a loop over `entries`. |
| `@field(s, name)` with a comptime field name (`modes.zig`) | `modesModePackedField(p, name) *bool`, a `switch` on the name returning a pointer to that field | an unknown name is a compile error in Zig; it panics in Go (cannot happen: every entry has a field). `132_column` maps to the Go field `x132_column`. |
| `switch (mode) { inline else => \|m\| { const entry = comptime entryForMode(m); ... } }` | `entry := modesEntryForMode(mode)` at run time, then `modesModePackedField` | same result. Zig evaluates the lookup at compile time; Go does it per call (see optimisation note). An invalid mode value panics in `String()` (Zig: switch on a corrupt enum value traps). |
| `inline for (entries)` with `comptime` conditions (`modeFromInt`, `reqFromInt`) | ordinary `for` loop and `if` | |
| `@splat(.disabled)` (FlagStack) | loop assigning `KittyKeyFlagsDisabled` to every element | |
| `u3` counter with `+%=` / `-%=` (`FlagStack.idx`) | uint8, `(idx + 1) & 7` / `(idx - 1) & 7` | wraps exactly as the u3 |
| `~` on a `u5` (`FlagStack.set`, `.not`) | `^` on the uint8 | sets bits 5..7 too, but the `&` with the current flags (at most 31) clears them, so the result is the u5 result |
| `pub inline fn` (`charsets.table`) | ordinary function | inlining left to the Go compiler |

#### Enum `String()` methods

- Exhaustive enums (all `lib.Enum`s, `ProtectedMode`, `Slots`, `ActiveSlot`,
  `Charset`, `SetMode`) read a file-private names array (`ansiCursorStyleNames`,
  `charsetsSlotsNames`, `kittyKeySetModeNames`, ...); an out-of-range value
  panics, as in `mouse_shape.go` / `Parser.go`.
- Non-exhaustive enums (`ANSIC0`, `ANSIRenditionAspect`, `ANSIStatusLineType`)
  use a `switch` and return `""` for an unnamed value (dumps print `#N`), per
  "Enums that are not exhaustive print `#N`".
- Generated enums (`ModesMode`, `DeviceStatusRequest`) search the generated
  field list and panic("unreachable") for a value outside the enum.

#### Per-file notes

##### `ansi.zig`
- No functions; only the String() methods. Upstream's inline comments (the
  `// Blank`, `// >`, `// =` of DeviceAttributeReq, the "Non-exhaustive so
  that ..." comments) are carried over.

##### `charsets.zig`
- `ascii`, `british`, `dec_special` are package variables initialised at
  start-up; the `british:` and `tech:` blocks are function literals called
  in the initialiser.
- `table`: `.utf8 => unreachable` is `panic("unreachable")`, and a value
  outside the enum also panics (exhaustive switch). The returned slice
  aliases the package array, as Zig's points at the comptime constant (the
  storage-identity test depends on that); callers must not write through it.
- Upstream's doc comment of `table` says the slice has 255 chars; it has
  `table_len` = 256. The comment is carried over with a note.
- `initTable` keeps the `while` loop and `assert(i == table_len)`.

##### `modes.zig`
- `ModeState{}` / `ModePacked{}` are `ModesModeStateInit()` /
  `ModesModePackedInit()` (wireframe). A zero `ModesModeState` is NOT Zig's
  default state (six modes default to true).
- `reset`: `self.saved = .{}` is `ModesModePackedInit()` (the type's defaults,
  not `self.default`), as upstream.
- `get` takes `*const ModeState` in Zig; Go uses the pointer receiver.
- `entryForMode`: `std.mem.eql(u8, entry.name, @tagName(mode))` is
  `string(entry.name) == mode.String()`; `@setEvalBranchQuota` has no
  counterpart; the final `unreachable` is `panic("unreachable")`.
- `ModesModeValues` (std.enums.values) returns a new slice per call.
- `golden/tables/modes.txt` (TestCorpusReplayTables/modes) passes.

##### `device_status.zig`
- Same scheme as modes.zig for the generated `Request` enum and `Tag`.
- The wireframe has no reverse `@bitCast` for `Tag` (modes has
  `ModesModeTagFromInt`), so three lines of
  `term51: Tag: layout of the value and question bit` stay untranslated
  (marked MISSING-IN-WIREFRAME in the test). A `DeviceStatusTagFromInt`
  identical in shape to `ModesModeTagFromInt` would make them translatable;
  not added here (signatures are the lead's).

##### `kitty/key.zig`
- `Flags.int` builds the u5 from the five bools (bit 0 = disambiguate ...
  bit 4 = report_associated); `KittyKeyFlagsFromInt` is the reverse and
  asserts `v <= 31`. stream.zig only casts values already checked with
  `std.math.cast(u5, ...)`, so the assertion never fires from the stream.
- `FlagStack.set`: the exhaustive switch gets `default: panic("unreachable")`.
- `FlagStack.pop(n)`: `n` is `uint` (usize); `n >= self.flags.len` resets the
  stack exactly as upstream (idx = 0, every slot disabled).

#### Optimisation candidate (for docs/post-port-optimizations.md)

- `ModeState.set/get/save/restore` evaluate at run time what Zig evaluates
  at compile time: `String()` (scan of 40 generated fields), `entryForMode`
  (scan of 40 entries comparing names) and the `@field` name switch. Measured
  about 420 ns per `Get` on the arm64 build machine (Zig: one bit test).
  `Terminal.print` calls `modes.get` on its common path (Terminal.zig lines
  326, 339, 425) for every printed character.
  Candidate: precompute the mode -> field mapping at start-up (e.g. a table
  indexed by mode giving a field index or accessor) and use it in the four
  methods.
