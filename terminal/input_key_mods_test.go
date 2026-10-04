// Tests of input_key_mods.go, translated from src/input/key_mods.zig.
//
// Translation notes (project lead's decisions, see
// scripts/not-translated-km.txt):
//
//   - Packed-integer conversions (`@bitCast`, `.int()`) use the wireframe's
//     `Int` and `...FromInt` stubs. Zig's u12 (RemapSet.Mask) and u4 values
//     are uint16 and uint8 in Go.
//   - `set.map` is `set.map_`; `map.count()` is `Count()` (uint), `map.get(k)`
//     is `Get(k)` (Optional), `map.keys()` is `Keys()`.
//   - Allocator parameters are dropped. A `defer set.deinit(alloc)` inside a
//     Zig loop body is run per iteration here by wrapping the body in a
//     closure.
//   - Platform branches follow the non-Darwin branch; the Darwin lines are
//     kept as `// Zig-only (Darwin):` comments.
//   - `std.testing.expectError` is kmExpectError (errors.Is).

package terminal

import (
	"errors"
	"testing"
)

// Short names for the types and constants the tests write most often.
type kmMods = InputKeyModsMods
type kmSides = InputKeyModsModsSide

const (
	kmLeft  = InputKeyModsModSideLeft
	kmRight = InputKeyModsModSideRight
)

// kmErrBoom is Zig's error.Term51Boom of term51ADFailFormatter.
var kmErrBoom = errors.New("Term51Boom")

// kmExpectError is Zig's `try testing.expectError(expected, actual)`.
func kmExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// kmExpectPanic runs f, which must panic. A panic with the wireframe's "not
// implemented" value does not count: it only means the function under test
// does not exist yet. The setup is done outside f.
func kmExpectPanic(t *testing.T, f func()) {
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

// kmParse is `try set.parse(alloc, input)`.
func kmParse(t *testing.T, set *InputKeyModsRemapSet, input string) {
	t.Helper()
	zigTry(t, set.Parse([]uint8(input)))
}

// kmParseCLI is `try set.parseCLI(alloc, input)` with a non-null input.
func kmParseCLI(t *testing.T, set *InputKeyModsRemapSet, input string) {
	t.Helper()
	zigTry(t, set.ParseCLI(Some([]uint8(input))))
}

// kmMask is `@as(u12, @bitCast(set.mask))`.
func kmMask(set *InputKeyModsRemapSet) uint16 {
	return set.mask.Int()
}

// term51ADModsFromInt: Mods from a raw u16, for iterating over every bit.
func kmModsFromInt(v uint16) kmMods {
	return InputKeyModsModsFromInt(v)
}

// term51ADMods4 and term51ADSides2: all eight possible froms of a set, as
// (mod, side) pairs.
var kmMods4 = [...]InputKeyModsMod{
	InputKeyModsModShift, InputKeyModsModCtrl, InputKeyModsModAlt, InputKeyModsModSuper,
}
var kmSides2 = [...]InputKeyModsModSide{kmLeft, kmRight}

// kmRecorder is term51ADRecorder: a formatter that records everything
// formatEntry gives it.
type kmRecorder struct {
	items      [][]uint8
	void_count uint
}

func (self *kmRecorder) formatEntry(value Optional[[]uint8]) error {
	if !value.Valid {
		self.void_count += 1
		return nil
	}
	self.items = append(self.items, append([]uint8(nil), value.Value...))
	return nil
}

// kmFailFormatter is term51ADFailFormatter: a formatter whose formatEntry
// always fails.
type kmFailFormatter struct{}

func (self *kmFailFormatter) formatEntry(value Optional[[]uint8]) error {
	return kmErrBoom
}

// kmModName is `@tagName(mod)`.
func kmModName(mod InputKeyModsMod) string {
	return mod.String()
}

// Zig: test (unnamed, key_mods.zig:160)
func TestKm_unnamed_key_mods_zig_160(t *testing.T) {
	// For our own understanding
	expectEqual(t, uint16(0b0), (kmMods{}).Int())
	expectEqual(t, uint16(0b0000_0001), (kmMods{shift: true}).Int())
}

// Zig: test "RemapSet: unsided remap creates both left and right mappings"
func TestKm_RemapSet_unsided_remap_creates_both_left_and_right_mappings(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "ctrl=super")
	set.Finalize()
	expectEqual(t,
		kmMods{
			super: true,
			sides: kmSides{super: kmLeft},
		},
		set.Apply(kmMods{
			ctrl:  true,
			sides: kmSides{ctrl: kmLeft},
		}),
	)
	expectEqual(t,
		kmMods{
			super: true,
			sides: kmSides{super: kmLeft},
		},
		set.Apply(kmMods{
			ctrl:  true,
			sides: kmSides{ctrl: kmRight},
		}),
	)
}

// Zig: test "RemapSet: sided from only maps that side"
func TestKm_RemapSet_sided_from_only_maps_that_side(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "left_alt=ctrl")
	set.Finalize()

	left_alt := kmMods{alt: true, sides: kmSides{alt: kmLeft}}
	left_ctrl := kmMods{ctrl: true, sides: kmSides{ctrl: kmLeft}}
	expectEqual(t, left_ctrl, set.Apply(left_alt))

	right_alt := kmMods{alt: true, sides: kmSides{alt: kmRight}}
	expectEqual(t, right_alt, set.Apply(right_alt))
}

// Zig: test "RemapSet: sided to"
func TestKm_RemapSet_sided_to(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "ctrl=right_super")
	set.Finalize()

	left_ctrl := kmMods{ctrl: true, sides: kmSides{ctrl: kmLeft}}
	right_super := kmMods{super: true, sides: kmSides{super: kmRight}}
	expectEqual(t, right_super, set.Apply(left_ctrl))
}

// Zig: test "RemapSet: both sides specified"
func TestKm_RemapSet_both_sides_specified(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "left_shift=right_ctrl")
	set.Finalize()

	left_shift := kmMods{shift: true, sides: kmSides{shift: kmLeft}}
	right_ctrl := kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}
	expectEqual(t, right_ctrl, set.Apply(left_shift))
}

// Zig: test "RemapSet: multiple parses accumulate"
func TestKm_RemapSet_multiple_parses_accumulate(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "left_ctrl=super")
	kmParse(t, &set, "left_alt=ctrl")
	set.Finalize()

	left_ctrl := kmMods{ctrl: true, sides: kmSides{ctrl: kmLeft}}
	left_super := kmMods{super: true, sides: kmSides{super: kmLeft}}
	expectEqual(t, left_super, set.Apply(left_ctrl))

	left_alt := kmMods{alt: true, sides: kmSides{alt: kmLeft}}
	left_ctrl_result := kmMods{ctrl: true, sides: kmSides{ctrl: kmLeft}}
	expectEqual(t, left_ctrl_result, set.Apply(left_alt))
}

// Zig: test "RemapSet: error on missing assignment"
func TestKm_RemapSet_error_on_missing_assignment(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmExpectError(t, ErrMissingAssignment, set.Parse([]uint8("ctrl")))
	kmExpectError(t, ErrMissingAssignment, set.Parse([]uint8("")))
}

// Zig: test "RemapSet: error on invalid modifier"
func TestKm_RemapSet_error_on_invalid_modifier(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("invalid=ctrl")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("ctrl=invalid")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("middle_ctrl=super")))
}

// Zig: test "RemapSet: isRemapped checks mask"
func TestKm_RemapSet_isRemapped_checks_mask(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "ctrl=super")
	set.Finalize()

	expect(t, set.IsRemapped(kmMods{ctrl: true}), "set.isRemapped(.{ .ctrl = true })")
	expect(t, !set.IsRemapped(kmMods{alt: true}), "!set.isRemapped(.{ .alt = true })")
	expect(t, !set.IsRemapped(kmMods{shift: true}), "!set.isRemapped(.{ .shift = true })")
}

// Zig: test "RemapSet: clone creates independent copy"
func TestKm_RemapSet_clone_creates_independent_copy(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "ctrl=super")
	set.Finalize()

	cloned, err := set.Clone()
	zigTry(t, err)
	defer cloned.Deinit()

	expect(t, set.Equal(cloned), "set.equal(cloned)")
	expect(t, cloned.IsRemapped(kmMods{ctrl: true}), "cloned.isRemapped(.{ .ctrl = true })")
}

// Zig: test "RemapSet: equal compares correctly"
func TestKm_RemapSet_equal_compares_correctly(t *testing.T) {
	set1 := InputKeyModsRemapSetEmpty
	defer set1.Deinit()

	set2 := InputKeyModsRemapSetEmpty
	defer set2.Deinit()

	expect(t, set1.Equal(set2), "set1.equal(set2)")

	kmParse(t, &set1, "ctrl=super")
	expect(t, !set1.Equal(set2), "!set1.equal(set2)")

	kmParse(t, &set2, "ctrl=super")
	expect(t, set1.Equal(set2), "set1.equal(set2)")

	kmParse(t, &set1, "alt=shift")
	expect(t, !set1.Equal(set2), "!set1.equal(set2)")
}

// Zig: test "RemapSet: parseCLI basic"
func TestKm_RemapSet_parseCLI_basic(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParseCLI(t, &set, "ctrl=super")
	expectEqual(t, uint(2), set.map_.Count())
}

// Zig: test "RemapSet: parseCLI empty clears"
func TestKm_RemapSet_parseCLI_empty_clears(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParseCLI(t, &set, "ctrl=super")
	expectEqual(t, uint(2), set.map_.Count())

	kmParseCLI(t, &set, "")
	expectEqual(t, uint(0), set.map_.Count())
}

// Zig: test "RemapSet: parseCLI invalid"
func TestKm_RemapSet_parseCLI_invalid(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmExpectError(t, ErrInvalidValue, set.ParseCLI(Some([]uint8("foo=bar"))))
	kmExpectError(t, ErrInvalidValue, set.ParseCLI(Some([]uint8("ctrl"))))
}

// Zig: test "RemapSet: parse aliased modifiers"
func TestKm_RemapSet_parse_aliased_modifiers(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "cmd=ctrl")
	set.Finalize()

	left_super := kmMods{super: true, sides: kmSides{super: kmLeft}}
	left_ctrl := kmMods{ctrl: true, sides: kmSides{ctrl: kmLeft}}
	expectEqual(t, left_ctrl, set.Apply(left_super))
}

// Zig: test "RemapSet: parse aliased modifiers command"
func TestKm_RemapSet_parse_aliased_modifiers_command(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "command=alt")
	set.Finalize()

	left_super := kmMods{super: true, sides: kmSides{super: kmLeft}}
	left_alt := kmMods{alt: true, sides: kmSides{alt: kmLeft}}
	expectEqual(t, left_alt, set.Apply(left_super))
}

// Zig: test "RemapSet: parse aliased modifiers opt and option"
func TestKm_RemapSet_parse_aliased_modifiers_opt_and_option(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "opt=super")
	set.Finalize()

	left_alt := kmMods{alt: true, sides: kmSides{alt: kmLeft}}
	left_super := kmMods{super: true, sides: kmSides{super: kmLeft}}
	expectEqual(t, left_super, set.Apply(left_alt))

	set.Deinit()
	set = InputKeyModsRemapSetEmpty

	kmParse(t, &set, "option=shift")
	set.Finalize()

	left_shift := kmMods{shift: true, sides: kmSides{shift: kmLeft}}
	expectEqual(t, left_shift, set.Apply(left_alt))
}

// Zig: test "RemapSet: parse aliased modifiers control"
func TestKm_RemapSet_parse_aliased_modifiers_control(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "control=super")
	set.Finalize()

	left_ctrl := kmMods{ctrl: true, sides: kmSides{ctrl: kmLeft}}
	left_super := kmMods{super: true, sides: kmSides{super: kmLeft}}
	expectEqual(t, left_super, set.Apply(left_ctrl))
}

// Zig: test "RemapSet: parse aliased modifiers on target side"
func TestKm_RemapSet_parse_aliased_modifiers_on_target_side(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "alt=cmd")
	set.Finalize()

	left_alt := kmMods{alt: true, sides: kmSides{alt: kmLeft}}
	left_super := kmMods{super: true, sides: kmSides{super: kmLeft}}
	expectEqual(t, left_super, set.Apply(left_alt))
}

// The four upstream "RemapSet: formatEntry ..." tests (empty, single sided,
// unsided creates two entries, right sided) are not translated: they need
// config/formatter.zig (app code) and are skipped in the library build. See
// scripts/not-translated-km.txt.

// Zig: test "term51: Mod: values and Side"
func TestKm_term51_Mod_values_and_Side(t *testing.T) {
	// Zig: @typeInfo(Mod).@"enum".fields; here the count constant and String.
	expectEqual(t, 4, inputKeyModsModCount)
	expectEqualStrings(t, "shift", InputKeyModsMod(0).String())
	expectEqualStrings(t, "ctrl", InputKeyModsMod(1).String())
	expectEqualStrings(t, "alt", InputKeyModsMod(2).String())
	expectEqualStrings(t, "super", InputKeyModsMod(3).String())
	expectEqual(t, InputKeyModsModShift, InputKeyModsMod(0))
	expectEqual(t, InputKeyModsModCtrl, InputKeyModsMod(1))
	expectEqual(t, InputKeyModsModAlt, InputKeyModsMod(2))
	expectEqual(t, InputKeyModsModSuper, InputKeyModsMod(3))

	expectEqual(t, uint8(0), uint8(InputKeyModsModSideLeft))
	expectEqual(t, uint8(1), uint8(InputKeyModsModSideRight))
	expectEqual(t, 2, inputKeyModsModSideCount)
	// Zig-only: expect(@typeInfo(Mod.Side).@"enum".tag_type == u1)
}

// Zig: test "term51: alias: table contents"
func TestKm_term51_alias_table_contents(t *testing.T) {
	expectEqual(t, 5, len(InputKeyModsAlias))
	expectEqualStrings(t, "cmd", string(InputKeyModsAlias[0].x0))
	expectEqual(t, InputKeyModsModSuper, InputKeyModsAlias[0].x1)
	expectEqualStrings(t, "command", string(InputKeyModsAlias[1].x0))
	expectEqual(t, InputKeyModsModSuper, InputKeyModsAlias[1].x1)
	expectEqualStrings(t, "opt", string(InputKeyModsAlias[2].x0))
	expectEqual(t, InputKeyModsModAlt, InputKeyModsAlias[2].x1)
	expectEqualStrings(t, "option", string(InputKeyModsAlias[3].x0))
	expectEqual(t, InputKeyModsModAlt, InputKeyModsAlias[3].x1)
	expectEqualStrings(t, "control", string(InputKeyModsAlias[4].x0))
	expectEqual(t, InputKeyModsModCtrl, InputKeyModsAlias[4].x1)
}

// "term51: Mods: layout and backing types" is not translated (@bitSizeOf and
// Backing type identity checks). See scripts/not-translated-km.txt.

// Zig: test "term51: Mods.int: bit position of every field"
func TestKm_term51_Mods_int_bit_position_of_every_field(t *testing.T) {
	expectEqual(t, uint16(0), (kmMods{}).Int())
	// Zig: inline for over the six field names, with @field(m, name) = true.
	fields := [...]struct {
		name string
		set  func(m *kmMods)
	}{
		{"shift", func(m *kmMods) { m.shift = true }},
		{"ctrl", func(m *kmMods) { m.ctrl = true }},
		{"alt", func(m *kmMods) { m.alt = true }},
		{"super", func(m *kmMods) { m.super = true }},
		{"caps_lock", func(m *kmMods) { m.caps_lock = true }},
		{"num_lock", func(m *kmMods) { m.num_lock = true }},
	}
	for i, f := range fields {
		var m kmMods
		f.set(&m)
		expectEqual(t, uint16(1)<<i, m.Int())
	}

	// Side bits sit at 6 through 9.
	expectEqual(t, uint16(1<<6), (kmMods{sides: kmSides{shift: kmRight}}).Int())
	expectEqual(t, uint16(1<<7), (kmMods{sides: kmSides{ctrl: kmRight}}).Int())
	expectEqual(t, uint16(1<<8), (kmMods{sides: kmSides{alt: kmRight}}).Int())
	expectEqual(t, uint16(1<<9), (kmMods{sides: kmSides{super: kmRight}}).Int())

	// Padding sits at 10 through 15.
	expectEqual(t, uint16(1<<10), (kmMods{_padding: 1}).Int())
	expectEqual(t, uint16(0xFC00), (kmMods{_padding: 0x3f}).Int())

	// Everything set.
	all := kmMods{
		shift:     true,
		ctrl:      true,
		alt:       true,
		super:     true,
		caps_lock: true,
		num_lock:  true,
		sides:     kmSides{shift: kmRight, ctrl: kmRight, alt: kmRight, super: kmRight},
		_padding:  0x3f,
	}
	expectEqual(t, uint16(0xFFFF), all.Int())
	expectEqual(t, all, kmModsFromInt(0xFFFF))
}

// Zig: test "term51: Mods: side_mask"
func TestKm_term51_Mods_side_mask(t *testing.T) {
	expectEqual(t, uint16(0x3C0), InputKeyModsModsSideMask.Int())
	expect(t, !InputKeyModsModsSideMask.shift, "!Mods.side_mask.shift")
	expect(t, !InputKeyModsModsSideMask.ctrl, "!Mods.side_mask.ctrl")
	expect(t, !InputKeyModsModsSideMask.alt, "!Mods.side_mask.alt")
	expect(t, !InputKeyModsModsSideMask.super, "!Mods.side_mask.super")
	expect(t, !InputKeyModsModsSideMask.caps_lock, "!Mods.side_mask.caps_lock")
	expect(t, !InputKeyModsModsSideMask.num_lock, "!Mods.side_mask.num_lock")
	expectEqual(t, kmRight, InputKeyModsModsSideMask.sides.shift)
	expectEqual(t, kmRight, InputKeyModsModsSideMask.sides.ctrl)
	expectEqual(t, kmRight, InputKeyModsModsSideMask.sides.alt)
	expectEqual(t, kmRight, InputKeyModsModsSideMask.sides.super)
}

// Zig: test "term51: Mods.Side: defaults and bit layout"
func TestKm_term51_Mods_Side_defaults_and_bit_layout(t *testing.T) {
	d := kmSides{}
	expectEqual(t, uint8(0), d.Int())
	expectEqual(t, kmLeft, d.shift)
	expectEqual(t, kmLeft, d.ctrl)
	expectEqual(t, kmLeft, d.alt)
	expectEqual(t, kmLeft, d.super)
	expectEqual(t, uint8(0b0001), (kmSides{shift: kmRight}).Int())
	expectEqual(t, uint8(0b0010), (kmSides{ctrl: kmRight}).Int())
	expectEqual(t, uint8(0b0100), (kmSides{alt: kmRight}).Int())
	expectEqual(t, uint8(0b1000), (kmSides{super: kmRight}).Int())
	expectEqual(t,
		uint8(0b1111),
		InputKeyModsModsSideMask.sides.Int(),
	)
}

// Zig: test "term51: Mods.Keys.int: every bit"
func TestKm_term51_Mods_Keys_int_every_bit(t *testing.T) {
	expectEqual(t, uint8(0), (InputKeyModsModsKeys{}).Int())
	expectEqual(t, uint8(0b0001), (InputKeyModsModsKeys{shift: true}).Int())
	expectEqual(t, uint8(0b0010), (InputKeyModsModsKeys{ctrl: true}).Int())
	expectEqual(t, uint8(0b0100), (InputKeyModsModsKeys{alt: true}).Int())
	expectEqual(t, uint8(0b1000), (InputKeyModsModsKeys{super: true}).Int())
	expectEqual(t, uint8(0b1111), (InputKeyModsModsKeys{
		shift: true,
		ctrl:  true,
		alt:   true,
		super: true,
	}).Int())
}

// Zig: test "term51: Mods.empty: defaults and every bit"
func TestKm_term51_Mods_empty_defaults_and_every_bit(t *testing.T) {
	expect(t, (kmMods{}).Empty(), "(Mods{}).empty()")
	for i := uint(0); i < 16; i++ {
		m := kmModsFromInt(uint16(1) << i)
		expect(t, !m.Empty(), "!m.empty()")
	}
	// A right side bit counts even if no key bit is set.
	expect(t, !(kmMods{sides: kmSides{alt: kmRight}}).Empty(), "!(Mods{ .sides = .{ .alt = .right } }).empty()")
	// Locks count.
	expect(t, !(kmMods{caps_lock: true}).Empty(), "!(Mods{ .caps_lock = true }).empty()")
	expect(t, !(kmMods{num_lock: true}).Empty(), "!(Mods{ .num_lock = true }).empty()")
	// Padding counts.
	expect(t, !(kmMods{_padding: 1}).Empty(), "!(Mods{ ._padding = 1 }).empty()")
	expect(t, !kmModsFromInt(0xFFFF).Empty(), "!term51ADModsFromInt(0xFFFF).empty()")
	expect(t, kmModsFromInt(0).Empty(), "term51ADModsFromInt(0).empty()")
}

// Zig: test "term51: Mods.equal: every bit and symmetry"
func TestKm_term51_Mods_equal_every_bit_and_symmetry(t *testing.T) {
	for i := uint(0); i < 16; i++ {
		a := kmModsFromInt(uint16(1) << i)
		expect(t, a.Equal(a), "a.equal(a)")
		expect(t, !a.Equal(kmMods{}), "!a.equal(.{})")
		expect(t, !(kmMods{}).Equal(a), "!(Mods{}).equal(a)")
		for j := uint(0); j < 16; j++ {
			if i == j {
				continue
			}
			b := kmModsFromInt(uint16(1) << j)
			expect(t, !a.Equal(b), "!a.equal(b)")
		}
	}
	expect(t, (kmMods{}).Equal(kmMods{}), "(Mods{}).equal(.{})")
	expect(t, (kmMods{ctrl: true, caps_lock: true}).Equal(kmMods{ctrl: true, caps_lock: true}), "(Mods{ .ctrl = true, .caps_lock = true }).equal(.{ .ctrl = true, .caps_lock = true })")
	expect(t, !(kmMods{ctrl: true}).Equal(kmMods{ctrl: true, caps_lock: true}), "!(Mods{ .ctrl = true }).equal(.{ .ctrl = true, .caps_lock = true })")
	expect(t, !(kmMods{ctrl: true}).Equal(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}), "!(Mods{ .ctrl = true }).equal(.{ .ctrl = true, .sides = .{ .ctrl = .right } })")
}

// Zig: test "term51: Mods.keys: only the low four bits"
func TestKm_term51_Mods_keys_only_the_low_four_bits(t *testing.T) {
	for i := uint(0); i < 16; i++ {
		m := kmModsFromInt(uint16(1) << i)
		var expected uint8
		if i < 4 {
			expected = uint8(1) << i
		}
		expectEqual(t, expected, m.Keys().Int())
	}
	all := kmModsFromInt(0xFFFF)
	expectEqual(t, InputKeyModsModsKeys{shift: true, ctrl: true, alt: true, super: true}, all.Keys())
	expectEqual(t, InputKeyModsModsKeys{}, (kmMods{caps_lock: true, num_lock: true}).Keys())
	expectEqual(t,
		InputKeyModsModsKeys{alt: true},
		(kmMods{alt: true, caps_lock: true, sides: kmSides{alt: kmRight}}).Keys(),
	)
}

// Zig: test "term51: Mods.binding: drops locks, sides and padding"
func TestKm_term51_Mods_binding_drops_locks_sides_and_padding(t *testing.T) {
	for i := uint(0); i < 16; i++ {
		m := kmModsFromInt(uint16(1) << i)
		var expected uint16
		if i < 4 {
			expected = uint16(1) << i
		}
		expectEqual(t, expected, m.Binding().Int())
	}
	expectEqual(t,
		kmMods{shift: true, ctrl: true, alt: true, super: true},
		kmModsFromInt(0xFFFF).Binding(),
	)
	expectEqual(t,
		kmMods{ctrl: true},
		(kmMods{ctrl: true, caps_lock: true, num_lock: true, sides: kmSides{ctrl: kmRight}}).Binding(),
	)
	expectEqual(t, kmMods{}, (kmMods{}).Binding())
}

// Zig: test "term51: Mods.unset: self and not other"
func TestKm_term51_Mods_unset_self_and_not_other(t *testing.T) {
	all := kmModsFromInt(0xFFFF)
	expectEqual(t, kmMods{}, all.Unset(all))
	expectEqual(t, all, all.Unset(kmMods{}))
	expectEqual(t, kmMods{}, (kmMods{}).Unset(all))
	expectEqual(t,
		kmMods{shift: true, alt: true},
		(kmMods{shift: true, ctrl: true, alt: true}).Unset(kmMods{ctrl: true}),
	)
	// Bits only in other are ignored.
	expectEqual(t,
		kmMods{shift: true},
		(kmMods{shift: true}).Unset(kmMods{ctrl: true, alt: true}),
	)
	// Locks and sides are plain bits too.
	expectEqual(t,
		kmMods{ctrl: true},
		(kmMods{ctrl: true, caps_lock: true, num_lock: true}).Unset(kmMods{caps_lock: true, num_lock: true}),
	)
	expectEqual(t,
		kmMods{alt: true},
		(kmMods{alt: true, sides: kmSides{alt: kmRight}}).Unset(kmMods{sides: kmSides{alt: kmRight}}),
	)
	// Removing the key bit leaves the side bit.
	expectEqual(t,
		kmMods{sides: kmSides{alt: kmRight}},
		(kmMods{alt: true, sides: kmSides{alt: kmRight}}).Unset(kmMods{alt: true}),
	)
	// Every single bit against the all-ones value.
	for i := uint(0); i < 16; i++ {
		bit := uint16(1) << i
		expectEqual(t, uint16(0xFFFF)&^bit, all.Unset(kmModsFromInt(bit)).Int())
	}
}

// Zig: test "term51: Mods.withoutLocks: clears only caps and num lock"
func TestKm_term51_Mods_withoutLocks_clears_only_caps_and_num_lock(t *testing.T) {
	expectEqual(t, kmMods{}, (kmMods{caps_lock: true, num_lock: true}).WithoutLocks())
	expectEqual(t, kmMods{}, (kmMods{caps_lock: true}).WithoutLocks())
	expectEqual(t, kmMods{}, (kmMods{num_lock: true}).WithoutLocks())
	expectEqual(t, kmMods{}, (kmMods{}).WithoutLocks())
	all := kmModsFromInt(0xFFFF)
	// 0xFFFF minus bits 4 and 5.
	expectEqual(t, uint16(0xFFCF), all.WithoutLocks().Int())
	expectEqual(t,
		kmMods{shift: true, alt: true, sides: kmSides{alt: kmRight}},
		(kmMods{shift: true, alt: true, caps_lock: true, num_lock: true, sides: kmSides{alt: kmRight}}).WithoutLocks(),
	)
}

// Zig: test "term51: Mods.translation: option as alt"
func TestKm_term51_Mods_translation_option_as_alt(t *testing.T) {
	// Zig: const is_darwin = comptime builtin.target.os.tag.isDarwin();
	// Go follows the non-Darwin branch.
	options := [...]InputConfigOptionAsAlt{
		InputConfigOptionAsAltFalse,
		InputConfigOptionAsAltTrue,
		InputConfigOptionAsAltLeft,
		InputConfigOptionAsAltRight,
	}

	inputs := [...]kmMods{
		{},
		{alt: true},
		{alt: true, sides: kmSides{alt: kmRight}},
		{alt: true, shift: true, caps_lock: true},
		{shift: true, ctrl: true, super: true},
		{sides: kmSides{alt: kmRight}},
	}
	for _, mods := range inputs {
		for _, opt := range options {
			expected := mods
			// Zig-only (Darwin): if (is_darwin) { unset := switch (opt) { .false => false, .true => true, .left => mods.sides.alt != .right, .right => mods.sides.alt != .left }; if (unset) expected.alt = false; }
			expectEqual(t, expected, mods.Translation(opt))
		}
	}

	// Only the alt flag is cleared: the side bit is retained.
	// Zig-only (Darwin): if (is_darwin) { r := (Mods{ .alt = true, .sides = .{ .alt = .right } }).translation(.right); expectEqual(Mods{ .sides = .{ .alt = .right } }, r) } else {
	// Not macOS: translation never changes anything.
	m := kmMods{alt: true}
	expectEqual(t, m, m.Translation(InputConfigOptionAsAltTrue))
	expectEqual(t, m, m.Translation(InputConfigOptionAsAltLeft))
	expectEqual(t, m, m.Translation(InputConfigOptionAsAltRight))
	expectEqual(t, m, m.Translation(InputConfigOptionAsAltFalse))
}

// Zig: test "term51: Mods.ctrlOrSuper: checks ctrl or super per platform"
func TestKm_term51_Mods_ctrlOrSuper_checks_ctrl_or_super_per_platform(t *testing.T) {
	// Zig: const is_darwin = comptime builtin.target.os.tag.isDarwin();
	// Go follows the non-Darwin branch.
	const is_darwin = false
	expect(t, !(kmMods{}).CtrlOrSuper(), "!(Mods{}).ctrlOrSuper()")
	expectEqual(t, !is_darwin, (kmMods{ctrl: true}).CtrlOrSuper())
	expectEqual(t, is_darwin, (kmMods{super: true}).CtrlOrSuper())
	expect(t, (kmMods{ctrl: true, super: true}).CtrlOrSuper(), "(Mods{ .ctrl = true, .super = true }).ctrlOrSuper()")
	expect(t, !(kmMods{shift: true, alt: true, caps_lock: true, num_lock: true}).CtrlOrSuper(), "!(Mods{ .shift = true, .alt = true, .caps_lock = true, .num_lock = true }).ctrlOrSuper()")
	expectEqual(t,
		!is_darwin,
		(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}).CtrlOrSuper(),
	)
}

// Zig: test "term51: RemapSet: empty value"
func TestKm_term51_RemapSet_empty_value(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	expectEqual(t, uint(0), set.map_.Count())
	expectEqual(t, uint16(0), kmMask(&set))
	expect(t, !set.IsRemapped(kmMods{ctrl: true}), "!set.isRemapped(.{ .ctrl = true })")
	expect(t, set.Equal(InputKeyModsRemapSetEmpty), "set.equal(.empty)")
}

// Zig: test "term51: RemapSet.deinit: frees the map"
func TestKm_term51_RemapSet_deinit_frees_the_map(t *testing.T) {
	// Zig: std.testing.allocator fails the test on a leak; Go has no leak
	// check, the calls are kept.
	set := InputKeyModsRemapSetEmpty
	kmParse(t, &set, "ctrl=super")
	kmParse(t, &set, "alt=shift")
	set.Deinit()

	// Deinit of an empty set allocates and frees nothing.
	empty := InputKeyModsRemapSetEmpty
	empty.Deinit()
}

// Zig: test "term51: RemapSet.initMods: every mod and side"
func TestKm_term51_RemapSet_initMods_every_mod_and_side(t *testing.T) {
	expectEqual(t, uint16(0x001), inputKeyModsRemapSetInitMods(InputKeyModsModShift, kmLeft).Int())
	expectEqual(t, uint16(0x002), inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmLeft).Int())
	expectEqual(t, uint16(0x004), inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft).Int())
	expectEqual(t, uint16(0x008), inputKeyModsRemapSetInitMods(InputKeyModsModSuper, kmLeft).Int())
	expectEqual(t, uint16(0x041), inputKeyModsRemapSetInitMods(InputKeyModsModShift, kmRight).Int())
	expectEqual(t, uint16(0x082), inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmRight).Int())
	expectEqual(t, uint16(0x104), inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmRight).Int())
	expectEqual(t, uint16(0x208), inputKeyModsRemapSetInitMods(InputKeyModsModSuper, kmRight).Int())
	expectEqual(t,
		kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}},
		inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmRight),
	)
	expectEqual(t, kmMods{alt: true}, inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft))
}

// kmExpectParsedMod is term51ADExpectParsedMod.
func kmExpectParsedMod(t *testing.T, input string, mod InputKeyModsMod, side Optional[InputKeyModsModSide]) {
	t.Helper()
	r0, r1, err := inputKeyModsRemapSetParseMod([]uint8(input))
	zigTry(t, err)
	expectEqual(t, mod, r0)
	expectEqual(t, side, r1)
}

// Zig: test "term51: RemapSet.parseMod: unsided names and aliases"
func TestKm_term51_RemapSet_parseMod_unsided_names_and_aliases(t *testing.T) {
	none := None[InputKeyModsModSide]()
	kmExpectParsedMod(t, "shift", InputKeyModsModShift, none)
	kmExpectParsedMod(t, "ctrl", InputKeyModsModCtrl, none)
	kmExpectParsedMod(t, "alt", InputKeyModsModAlt, none)
	kmExpectParsedMod(t, "super", InputKeyModsModSuper, none)
	kmExpectParsedMod(t, "cmd", InputKeyModsModSuper, none)
	kmExpectParsedMod(t, "command", InputKeyModsModSuper, none)
	kmExpectParsedMod(t, "opt", InputKeyModsModAlt, none)
	kmExpectParsedMod(t, "option", InputKeyModsModAlt, none)
	kmExpectParsedMod(t, "control", InputKeyModsModCtrl, none)
}

// Zig: test "term51: RemapSet.parseMod: sided names and aliases"
func TestKm_term51_RemapSet_parseMod_sided_names_and_aliases(t *testing.T) {
	// Zig: inline for (.{ "left", "right" }) |side_name|
	sides := [...]struct {
		name string
		side InputKeyModsModSide
	}{{"left", kmLeft}, {"right", kmRight}}
	for _, s := range sides {
		side := Some(s.side)
		kmExpectParsedMod(t, s.name+"_shift", InputKeyModsModShift, side)
		kmExpectParsedMod(t, s.name+"_ctrl", InputKeyModsModCtrl, side)
		kmExpectParsedMod(t, s.name+"_alt", InputKeyModsModAlt, side)
		kmExpectParsedMod(t, s.name+"_super", InputKeyModsModSuper, side)
		kmExpectParsedMod(t, s.name+"_cmd", InputKeyModsModSuper, side)
		kmExpectParsedMod(t, s.name+"_command", InputKeyModsModSuper, side)
		kmExpectParsedMod(t, s.name+"_opt", InputKeyModsModAlt, side)
		kmExpectParsedMod(t, s.name+"_option", InputKeyModsModAlt, side)
		kmExpectParsedMod(t, s.name+"_control", InputKeyModsModCtrl, side)
	}
}

// Zig: test "term51: RemapSet.parseMod: invalid input"
func TestKm_term51_RemapSet_parseMod_invalid_input(t *testing.T) {
	bad := [...]string{
		"",      // empty
		"_",     // empty side and empty mod
		"left_", // empty mod
		"right_",
		"super_",          // trailing underscore: mod is empty
		"left__ctrl",      // mod is "_ctrl"
		"left_right_ctrl", // mod is "right_ctrl"
		"ctrl_left",       // side is "ctrl", mod is "left"
		"middle_ctrl",     // unknown side
		"bogus_shift",
		"control_ctrl", // alias name used as a side
		"Left_ctrl",    // case sensitive
		"left_CTRL",
		"CTRL",
		"Shift",
		"ctrl ", // whitespace is not trimmed
		" ctrl",
		"left_ ctrl",
		"meta",
		"win",
		"left_meta",
		"caps_lock",
		"alt_gr",
		"fn",
	}
	for _, input := range bad {
		_, _, err := inputKeyModsRemapSetParseMod([]uint8(input))
		kmExpectError(t, ErrInvalidMod, err)
	}
}

// Zig: test "term51: RemapSet.parseMod: empty side prefix is accepted as unsided"
func TestKm_term51_RemapSet_parseMod_empty_side_prefix_is_accepted_as_unsided(t *testing.T) {
	// "_ctrl" splits at the underscore into an empty side and "ctrl".
	// An empty side string means "no side", so this parses like "ctrl".
	kmExpectParsedMod(t, "_ctrl", InputKeyModsModCtrl, None[InputKeyModsModSide]())
	kmExpectParsedMod(t, "_cmd", InputKeyModsModSuper, None[InputKeyModsModSide]())
}

// Zig: test "term51: RemapSet.parse: unsided from creates left and right entries"
func TestKm_term51_RemapSet_parse_unsided_from_creates_left_and_right_entries(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "ctrl=super")
	expectEqual(t, uint(2), set.map_.Count())

	left_ctrl := inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmLeft)
	right_ctrl := inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmRight)
	left_super := inputKeyModsRemapSetInitMods(InputKeyModsModSuper, kmLeft)
	expectEqual(t, left_super, zigUnwrap(t, set.map_.Get(left_ctrl)))
	expectEqual(t, left_super, zigUnwrap(t, set.map_.Get(right_ctrl)))

	// Insertion order: left first, then right.
	expectEqual(t, left_ctrl, set.map_.Keys()[0])
	expectEqual(t, right_ctrl, set.map_.Keys()[1])

	// Mask: ctrl, both sides.
	expectEqual(t, uint16(0x222), kmMask(&set))
}

// Zig: test "term51: RemapSet.parse: sided from creates one entry"
func TestKm_term51_RemapSet_parse_sided_from_creates_one_entry(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "left_alt=ctrl")
	expectEqual(t, uint(1), set.map_.Count())
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft))),
	)
	expect(t, !set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmRight)).Valid, "set.map.get(RemapSet.initMods(.alt, .right)) == null")
	// keys=alt (0b0100), left_sides=alt, right_sides=none.
	expectEqual(t, uint16(0x044), kmMask(&set))

	kmParse(t, &set, "right_shift=left_ctrl")
	expectEqual(t, uint(2), set.map_.Count())
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModShift, kmRight))),
	)
	// shift is added on the right: keys 0b0101, left 0b0100, right 0b0001.
	expectEqual(t, uint16(0x145), kmMask(&set))
}

// Zig: test "term51: RemapSet.parse: to side defaults to left"
func TestKm_term51_RemapSet_parse_to_side_defaults_to_left(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "right_ctrl=alt")
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmRight))),
	)

	kmParse(t, &set, "left_shift=right_alt")
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmRight),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModShift, kmLeft))),
	)

	kmParse(t, &set, "left_super=left_alt")
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModSuper, kmLeft))),
	)
}

// Zig: test "term51: RemapSet.parse: mask only tracks the from side"
func TestKm_term51_RemapSet_parse_mask_only_tracks_the_from_side(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "left_ctrl=right_super")
	expectEqual(t, uint16(0x022), kmMask(&set))
	expect(t, !set.mask.keys.super, "!set.mask.keys.super")
	expect(t, !set.IsRemapped(kmMods{super: true}), "!set.isRemapped(.{ .super = true })")
}

// Zig: test "term51: RemapSet.parse: same from replaces the previous target"
func TestKm_term51_RemapSet_parse_same_from_replaces_the_previous_target(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "ctrl=super")
	kmParse(t, &set, "ctrl=alt")
	expectEqual(t, uint(2), set.map_.Count())
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmLeft))),
	)
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmRight))),
	)

	// A sided entry only replaces its own side.
	kmParse(t, &set, "right_ctrl=shift")
	expectEqual(t, uint(2), set.map_.Count())
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmLeft))),
	)
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModShift, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmRight))),
	)
}

// Zig: test "term51: RemapSet.parse: every from and to combination"
func TestKm_term51_RemapSet_parse_every_from_and_to_combination(t *testing.T) {
	for _, from := range kmMods4 {
		for _, from_side := range kmSides2 {
			for _, to := range kmMods4 {
				for _, to_side := range kmSides2 {
					// The closure runs the deferred Deinit per iteration.
					func() {
						set := InputKeyModsRemapSetEmpty
						defer set.Deinit()

						input := from_side.String() + "_" + kmModName(from) + "=" + to_side.String() + "_" + kmModName(to)
						kmParse(t, &set, input)
						expectEqual(t, uint(1), set.map_.Count())
						expectEqual(t,
							inputKeyModsRemapSetInitMods(to, to_side),
							zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(from, from_side))),
						)
					}()
				}
			}
		}
	}
}

// Zig: test "term51: RemapSet.parse: errors"
func TestKm_term51_RemapSet_parse_errors(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	// Missing assignment wins over any other problem.
	kmExpectError(t, ErrMissingAssignment, set.Parse([]uint8("")))
	kmExpectError(t, ErrMissingAssignment, set.Parse([]uint8("ctrl")))
	kmExpectError(t, ErrMissingAssignment, set.Parse([]uint8("bogus")))
	kmExpectError(t, ErrMissingAssignment, set.Parse([]uint8("ctrl:super")))

	// Empty or invalid either side.
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("=")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("ctrl=")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("=ctrl")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("ctrl==super")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("ctrl=super=alt")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("ctrl=super ")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8(" ctrl=super")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("ctrl = super")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("meta=super")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("ctrl=meta")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("up_ctrl=alt")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("ctrl=up_alt")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("left_=alt")))
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("ctrl=Left_alt")))

	// Nothing was added by any failed parse.
	expectEqual(t, uint(0), set.map_.Count())
	expectEqual(t, uint16(0), kmMask(&set))
}

// Zig: test "term51: RemapSet.parse: failed parse keeps previous entries"
func TestKm_term51_RemapSet_parse_failed_parse_keeps_previous_entries(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParse(t, &set, "left_ctrl=super")
	mask_before := kmMask(&set)
	kmExpectError(t, ErrInvalidMod, set.Parse([]uint8("alt=bogus")))
	kmExpectError(t, ErrMissingAssignment, set.Parse([]uint8("alt")))
	expectEqual(t, uint(1), set.map_.Count())
	expectEqual(t, mask_before, kmMask(&set))
}

// "term51: RemapSet.parse: out of memory leaves the set untouched" is not
// translated (FailingAllocator). See scripts/not-translated-km.txt.

// Zig: test "term51: RemapSet.parseCLI: null and empty reset the set"
func TestKm_term51_RemapSet_parseCLI_null_and_empty_reset_the_set(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParseCLI(t, &set, "ctrl=super")
	expectEqual(t, uint(2), set.map_.Count())
	expect(t, set.IsRemapped(kmMods{ctrl: true}), "set.isRemapped(.{ .ctrl = true })")

	zigTry(t, set.ParseCLI(None[[]uint8]()))
	expectEqual(t, uint(0), set.map_.Count())
	expectEqual(t, uint16(0), kmMask(&set))
	expect(t, !set.IsRemapped(kmMods{ctrl: true}), "!set.isRemapped(.{ .ctrl = true })")

	kmParseCLI(t, &set, "alt=shift")
	expectEqual(t, uint(2), set.map_.Count())
	kmParseCLI(t, &set, "")
	expectEqual(t, uint(0), set.map_.Count())
	expectEqual(t, uint16(0), kmMask(&set))

	// Resetting an already empty set is fine.
	zigTry(t, set.ParseCLI(None[[]uint8]()))
	expectEqual(t, uint(0), set.map_.Count())
}

// Zig: test "term51: RemapSet.parseCLI: accumulates and maps errors to InvalidValue"
func TestKm_term51_RemapSet_parseCLI_accumulates_and_maps_errors_to_InvalidValue(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	kmParseCLI(t, &set, "ctrl=super")
	kmParseCLI(t, &set, "left_alt=shift")
	expectEqual(t, uint(3), set.map_.Count())

	kmExpectError(t, ErrInvalidValue, set.ParseCLI(Some([]uint8("ctrl"))))
	kmExpectError(t, ErrInvalidValue, set.ParseCLI(Some([]uint8("foo=bar"))))
	kmExpectError(t, ErrInvalidValue, set.ParseCLI(Some([]uint8("ctrl=foo"))))
	kmExpectError(t, ErrInvalidValue, set.ParseCLI(Some([]uint8("="))))

	// Invalid input does not clear what was there.
	expectEqual(t, uint(3), set.map_.Count())
}

// "term51: RemapSet.parseCLI: out of memory is passed through" is not
// translated (FailingAllocator). See scripts/not-translated-km.txt.

// Zig: test "term51: RemapSet.finalize: right sided entries come first"
func TestKm_term51_RemapSet_finalize_right_sided_entries_come_first(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	// Insertion order: left_ctrl, right_ctrl, left_alt, right_shift.
	kmParse(t, &set, "ctrl=super")
	kmParse(t, &set, "left_alt=shift")
	kmParse(t, &set, "right_shift=alt")
	expectEqual(t, uint(4), set.map_.Count())
	expect(t, set.map_.Keys()[0].Equal(inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmLeft)), "set.map.keys()[0].equal(RemapSet.initMods(.ctrl, .left))")

	set.Finalize()
	expectEqual(t, uint(4), set.map_.Count())

	// All entries with any right side bit precede the others.
	keys := set.map_.Keys()
	side_mask := InputKeyModsModsSideMask.Int()
	expect(t, keys[0].Int()&side_mask != 0, "keys[0].int() & Mods.side_mask.int() != 0")
	expect(t, keys[1].Int()&side_mask != 0, "keys[1].int() & Mods.side_mask.int() != 0")
	expect(t, keys[2].Int()&side_mask == 0, "keys[2].int() & Mods.side_mask.int() == 0")
	expect(t, keys[3].Int()&side_mask == 0, "keys[3].int() & Mods.side_mask.int() == 0")

	// Lookups still work after sorting (index rebuilt) and keep their values.
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModSuper, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmLeft))),
	)
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModSuper, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmRight))),
	)
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModShift, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft))),
	)
	expectEqual(t,
		inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft),
		zigUnwrap(t, set.map_.Get(inputKeyModsRemapSetInitMods(InputKeyModsModShift, kmRight))),
	)
}

// Zig: test "term51: RemapSet.finalize: empty and single element"
func TestKm_term51_RemapSet_finalize_empty_and_single_element(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	set.Finalize()
	expectEqual(t, uint(0), set.map_.Count())

	kmParse(t, &set, "right_alt=ctrl")
	set.Finalize()
	expectEqual(t, uint(1), set.map_.Count())
	expectEqual(t, inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmRight), set.map_.Keys()[0])
}

// Zig: test "term51: RemapSet.clone: deep and independent"
func TestKm_term51_RemapSet_clone_deep_and_independent(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "ctrl=super")
	kmParse(t, &set, "right_alt=shift")
	set.Finalize()

	cloned, err := set.Clone()
	zigTry(t, err)
	defer cloned.Deinit()

	expect(t, set.Equal(cloned), "set.equal(cloned)")
	expect(t, cloned.Equal(set), "cloned.equal(set)")
	expectEqual(t, uint(3), cloned.map_.Count())
	expectEqual(t, kmMask(&set), kmMask(&cloned))
	// Same order as the source.
	setKeys := set.map_.Keys()
	clonedKeys := cloned.map_.Keys()
	for i := range setKeys {
		expectEqual(t, setKeys[i], clonedKeys[i])
	}

	// Mutating the original does not touch the clone.
	kmParse(t, &set, "left_shift=alt")
	expectEqual(t, uint(4), set.map_.Count())
	expectEqual(t, uint(3), cloned.map_.Count())
	expect(t, !set.Equal(cloned), "!set.equal(cloned)")
	expect(t, !cloned.IsRemapped(kmMods{shift: true}), "!cloned.isRemapped(.{ .shift = true })")

	// And the other way around.
	zigTry(t, cloned.ParseCLI(Some([]uint8(""))))
	expectEqual(t, uint(4), set.map_.Count())
	expectEqual(t, uint(0), cloned.map_.Count())
}

// Zig: test "term51: RemapSet.clone: empty set"
func TestKm_term51_RemapSet_clone_empty_set(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	cloned, err := set.Clone()
	zigTry(t, err)
	defer cloned.Deinit()
	expectEqual(t, uint(0), cloned.map_.Count())
	expectEqual(t, uint16(0), kmMask(&cloned))
}

// "term51: RemapSet.clone: out of memory" is not translated
// (FailingAllocator). See scripts/not-translated-km.txt.

// Zig: test "term51: RemapSet.equal: counts keys and values"
func TestKm_term51_RemapSet_equal_counts_keys_and_values(t *testing.T) {
	a := InputKeyModsRemapSetEmpty
	defer a.Deinit()
	b := InputKeyModsRemapSetEmpty
	defer b.Deinit()

	expect(t, a.Equal(b), "a.equal(b)")

	// Different count.
	kmParse(t, &a, "ctrl=super")
	expect(t, !a.Equal(b), "!a.equal(b)")
	expect(t, !b.Equal(a), "!b.equal(a)")

	// Same keys, same values.
	kmParse(t, &b, "ctrl=super")
	expect(t, a.Equal(b), "a.equal(b)")
	expect(t, b.Equal(a), "b.equal(a)")

	// Same keys, different value.
	c := InputKeyModsRemapSetEmpty
	defer c.Deinit()
	kmParse(t, &c, "ctrl=alt")
	expect(t, !a.Equal(c), "!a.equal(c)")
	expect(t, !c.Equal(a), "!c.equal(a)")

	// Same count, different keys.
	d := InputKeyModsRemapSetEmpty
	defer d.Deinit()
	kmParse(t, &d, "alt=super")
	expect(t, !a.Equal(d), "!a.equal(d)")
	expect(t, !d.Equal(a), "!d.equal(a)")

	// Same count, partly different keys: left_ctrl + left_alt vs ctrl.
	e := InputKeyModsRemapSetEmpty
	defer e.Deinit()
	kmParse(t, &e, "left_ctrl=super")
	kmParse(t, &e, "left_alt=super")
	expect(t, !a.Equal(e), "!a.equal(e)")
	expect(t, !e.Equal(a), "!e.equal(a)")

	// Target side matters.
	f := InputKeyModsRemapSetEmpty
	defer f.Deinit()
	kmParse(t, &f, "ctrl=right_super")
	expect(t, !a.Equal(f), "!a.equal(f)")
}

// Zig: test "term51: RemapSet.equal: insertion order and mask are not compared"
func TestKm_term51_RemapSet_equal_insertion_order_and_mask_are_not_compared(t *testing.T) {
	a := InputKeyModsRemapSetEmpty
	defer a.Deinit()
	b := InputKeyModsRemapSetEmpty
	defer b.Deinit()

	kmParse(t, &a, "ctrl=super")
	kmParse(t, &a, "alt=shift")
	kmParse(t, &b, "alt=shift")
	kmParse(t, &b, "ctrl=super")
	expect(t, a.Equal(b), "a.equal(b)")
	expect(t, b.Equal(a), "b.equal(a)")

	// Finalizing one of them changes the order but not equality.
	kmParse(t, &a, "right_shift=ctrl")
	kmParse(t, &b, "right_shift=ctrl")
	a.Finalize()
	expect(t, a.Equal(b), "a.equal(b)")

	// The mask is not part of equality.
	b.mask = InputKeyModsRemapSetMask{}
	expect(t, a.Equal(b), "a.equal(b)")
}

// kmExpectFormatMod is term51ADExpectFormatMod.
func kmExpectFormatMod(t *testing.T, mods kmMods, expected string) {
	t.Helper()
	var buf [64]uint8
	fbs := inputKeyModsFixedBufferStreamInit(buf[:])
	zigTry(t, inputKeyModsRemapSetFormatMod(&fbs, mods))
	expectEqualStrings(t, expected, string(fbs.getWritten()))
}

// Zig: test "term51: RemapSet.formatMod: every mod and side"
func TestKm_term51_RemapSet_formatMod_every_mod_and_side(t *testing.T) {
	kmExpectFormatMod(t, inputKeyModsRemapSetInitMods(InputKeyModsModShift, kmLeft), "left_shift")
	kmExpectFormatMod(t, inputKeyModsRemapSetInitMods(InputKeyModsModShift, kmRight), "right_shift")
	kmExpectFormatMod(t, inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmLeft), "left_ctrl")
	kmExpectFormatMod(t, inputKeyModsRemapSetInitMods(InputKeyModsModCtrl, kmRight), "right_ctrl")
	kmExpectFormatMod(t, inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmLeft), "left_alt")
	kmExpectFormatMod(t, inputKeyModsRemapSetInitMods(InputKeyModsModAlt, kmRight), "right_alt")
	kmExpectFormatMod(t, inputKeyModsRemapSetInitMods(InputKeyModsModSuper, kmLeft), "left_super")
	kmExpectFormatMod(t, inputKeyModsRemapSetInitMods(InputKeyModsModSuper, kmRight), "right_super")
}

// Zig: test "term51: RemapSet.formatMod: only the first set mod is written"
func TestKm_term51_RemapSet_formatMod_only_the_first_set_mod_is_written(t *testing.T) {
	// Priority order is shift, ctrl, alt, super.
	kmExpectFormatMod(t, kmMods{shift: true, ctrl: true, alt: true, super: true}, "left_shift")
	kmExpectFormatMod(t, kmMods{ctrl: true, alt: true, super: true}, "left_ctrl")
	kmExpectFormatMod(t, kmMods{alt: true, super: true}, "left_alt")
	kmExpectFormatMod(t, kmMods{super: true}, "left_super")
	// The side of the written mod is the one used.
	kmExpectFormatMod(t,
		kmMods{ctrl: true, alt: true, sides: kmSides{ctrl: kmRight, alt: kmLeft}},
		"right_ctrl",
	)
	kmExpectFormatMod(t,
		kmMods{ctrl: true, alt: true, sides: kmSides{ctrl: kmLeft, alt: kmRight}},
		"left_ctrl",
	)
}

// Zig: test "term51: RemapSet.formatMod: nothing set writes nothing"
func TestKm_term51_RemapSet_formatMod_nothing_set_writes_nothing(t *testing.T) {
	kmExpectFormatMod(t, kmMods{}, "")
	// Locks and side bits alone do not produce output.
	kmExpectFormatMod(t, kmMods{caps_lock: true, num_lock: true}, "")
	kmExpectFormatMod(t, InputKeyModsModsSideMask, "")
}

// Zig: test "term51: RemapSet.formatMod: writer errors are returned"
func TestKm_term51_RemapSet_formatMod_writer_errors_are_returned(t *testing.T) {
	var buf [4]uint8
	fbs := inputKeyModsFixedBufferStreamInit(buf[:])
	kmExpectError(t,
		ErrNoSpaceLeft,
		inputKeyModsRemapSetFormatMod(&fbs, kmMods{shift: true}),
	)

	// Not an error when there is nothing to write.
	fbs2 := inputKeyModsFixedBufferStreamInit(buf[:])
	zigTry(t, inputKeyModsRemapSetFormatMod(&fbs2, kmMods{}))
}

// Zig: test "term51: RemapSet.formatEntry: empty set emits a void entry"
func TestKm_term51_RemapSet_formatEntry_empty_set_emits_a_void_entry(t *testing.T) {
	var rec kmRecorder

	set := InputKeyModsRemapSetEmpty
	zigTry(t, set.FormatEntry(&rec))
	expectEqual(t, uint(1), rec.void_count)
	expectEqual(t, 0, len(rec.items))
}

// Zig: test "term51: RemapSet.formatEntry: one line per mapping in map order"
func TestKm_term51_RemapSet_formatEntry_one_line_per_mapping_in_map_order(t *testing.T) {
	var rec kmRecorder

	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "ctrl=super")
	kmParse(t, &set, "left_alt=right_shift")
	zigTry(t, set.FormatEntry(&rec))

	expectEqual(t, uint(0), rec.void_count)
	expectEqual(t, 3, len(rec.items))
	expectEqualStrings(t, "left_ctrl=left_super", string(rec.items[0]))
	expectEqualStrings(t, "right_ctrl=left_super", string(rec.items[1]))
	expectEqualStrings(t, "left_alt=right_shift", string(rec.items[2]))
}

// Zig: test "term51: RemapSet.formatEntry: single and right sided entries"
func TestKm_term51_RemapSet_formatEntry_single_and_right_sided_entries(t *testing.T) {
	var rec kmRecorder

	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "left_ctrl=super")
	set.Finalize()
	zigTry(t, set.FormatEntry(&rec))
	expectEqual(t, 1, len(rec.items))
	expectEqualStrings(t, "left_ctrl=left_super", string(rec.items[0]))

	var rec2 kmRecorder
	set2 := InputKeyModsRemapSetEmpty
	defer set2.Deinit()
	kmParse(t, &set2, "left_alt=right_ctrl")
	zigTry(t, set2.FormatEntry(&rec2))
	expectEqual(t, 1, len(rec2.items))
	expectEqualStrings(t, "left_alt=right_ctrl", string(rec2.items[0]))
}

// Zig: test "term51: RemapSet.formatEntry: round trip of every combination"
func TestKm_term51_RemapSet_formatEntry_round_trip_of_every_combination(t *testing.T) {
	for _, from := range kmMods4 {
		for _, from_side := range kmSides2 {
			for _, to := range kmMods4 {
				for _, to_side := range kmSides2 {
					input := from_side.String() + "_" + kmModName(from) + "=" + to_side.String() + "_" + kmModName(to)

					// The closure runs the deferred Deinit per iteration.
					func() {
						set := InputKeyModsRemapSetEmpty
						defer set.Deinit()
						kmParse(t, &set, input)

						var rec kmRecorder
						zigTry(t, set.FormatEntry(&rec))
						expectEqual(t, 1, len(rec.items))
						expectEqualStrings(t, input, string(rec.items[0]))
					}()
				}
			}
		}
	}
}

// Zig: test "term51: RemapSet.formatEntry: formatter errors are returned"
func TestKm_term51_RemapSet_formatEntry_formatter_errors_are_returned(t *testing.T) {
	var fail kmFailFormatter

	empty := InputKeyModsRemapSetEmpty
	kmExpectError(t, kmErrBoom, empty.FormatEntry(&fail))

	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "ctrl=super")
	kmExpectError(t, kmErrBoom, set.FormatEntry(&fail))
}

// Zig: test "term51: RemapSet.isRemapped: depends on the mask only"
func TestKm_term51_RemapSet_isRemapped_depends_on_the_mask_only(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	expect(t, !set.IsRemapped(kmMods{}), "!set.isRemapped(.{})")
	expect(t, !set.IsRemapped(kmMods{ctrl: true}), "!set.isRemapped(.{ .ctrl = true })")

	kmParse(t, &set, "left_ctrl=super")
	expect(t, set.IsRemapped(kmMods{ctrl: true}), "set.isRemapped(.{ .ctrl = true })")
	expect(t, set.IsRemapped(kmMods{ctrl: true, shift: true}), "set.isRemapped(.{ .ctrl = true, .shift = true })")
	expect(t, set.IsRemapped(kmMods{ctrl: true, caps_lock: true}), "set.isRemapped(.{ .ctrl = true, .caps_lock = true })")
	// Right ctrl is not remapped by a left only remap.
	expect(t, !set.IsRemapped(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}), "!set.isRemapped(.{ .ctrl = true, .sides = .{ .ctrl = .right } })")
	expect(t, !set.IsRemapped(kmMods{alt: true}), "!set.isRemapped(.{ .alt = true })")
	expect(t, !set.IsRemapped(kmMods{super: true}), "!set.isRemapped(.{ .super = true })")
	expect(t, !set.IsRemapped(kmMods{caps_lock: true}), "!set.isRemapped(.{ .caps_lock = true })")
	expect(t, !set.IsRemapped(kmMods{}), "!set.isRemapped(.{})")

	// Direct mask edits are seen.
	set.mask = InputKeyModsRemapSetMask{}
	expect(t, !set.IsRemapped(kmMods{ctrl: true}), "!set.isRemapped(.{ .ctrl = true })")
}

// Zig: test "term51: RemapSet.apply: unremapped mods are returned unchanged"
func TestKm_term51_RemapSet_apply_unremapped_mods_are_returned_unchanged(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()

	mods := kmMods{ctrl: true, caps_lock: true, _padding: 0x2a}
	expectEqual(t, mods, set.Apply(mods))

	kmParse(t, &set, "left_alt=super")
	set.Finalize()
	expectEqual(t, mods, set.Apply(mods))
	expectEqual(t, kmMods{}, set.Apply(kmMods{}))
	right_alt := kmMods{alt: true, sides: kmSides{alt: kmRight}}
	expectEqual(t, right_alt, set.Apply(right_alt))
}

// Zig: test "term51: RemapSet.apply: keeps other mods, locks and sides"
func TestKm_term51_RemapSet_apply_keeps_other_mods_locks_and_sides(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "ctrl=super")
	set.Finalize()

	input := kmMods{
		ctrl:      true,
		shift:     true,
		caps_lock: true,
		num_lock:  true,
		sides:     kmSides{shift: kmRight, alt: kmRight},
	}
	expectEqual(t,
		kmMods{
			super:     true,
			shift:     true,
			caps_lock: true,
			num_lock:  true,
			sides:     kmSides{shift: kmRight, alt: kmRight},
		},
		set.Apply(input),
	)
}

// Zig: test "term51: RemapSet.apply: right sided from and to"
func TestKm_term51_RemapSet_apply_right_sided_from_and_to(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "right_ctrl=right_alt")
	set.Finalize()

	// Right ctrl becomes right alt: the ctrl key and side bits are cleared
	// and the alt key and side bits set.
	expectEqual(t,
		kmMods{alt: true, sides: kmSides{alt: kmRight}},
		set.Apply(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}),
	)
	// Left ctrl is untouched.
	left_ctrl := kmMods{ctrl: true}
	expectEqual(t, left_ctrl, set.Apply(left_ctrl))
}

// Zig: test "term51: RemapSet.apply: right from to a left target"
func TestKm_term51_RemapSet_apply_right_from_to_a_left_target(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "right_ctrl=shift")
	set.Finalize()
	expectEqual(t,
		kmMods{shift: true},
		set.Apply(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}),
	)
}

// Zig: test "term51: RemapSet.apply: left from to a right target"
func TestKm_term51_RemapSet_apply_left_from_to_a_right_target(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "left_ctrl=right_shift")
	set.Finalize()
	expectEqual(t,
		kmMods{shift: true, sides: kmSides{shift: kmRight}},
		set.Apply(kmMods{ctrl: true}),
	)
}

// Zig: test "term51: RemapSet.apply: remaps are not chained"
func TestKm_term51_RemapSet_apply_remaps_are_not_chained(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "ctrl=alt")
	kmParse(t, &set, "alt=super")
	set.Finalize()

	expectEqual(t, kmMods{alt: true}, set.Apply(kmMods{ctrl: true}))
	expectEqual(t, kmMods{super: true}, set.Apply(kmMods{alt: true}))
}

// Zig: test "term51: RemapSet.apply: every unsided single mod"
func TestKm_term51_RemapSet_apply_every_unsided_single_mod(t *testing.T) {
	for _, from := range kmMods4 {
		for _, to := range kmMods4 {
			// The closure runs the deferred Deinit per iteration.
			func() {
				set := InputKeyModsRemapSetEmpty
				defer set.Deinit()

				input := kmModName(from) + "=" + kmModName(to)
				kmParse(t, &set, input)
				set.Finalize()

				for _, side := range kmSides2 {
					got := set.Apply(inputKeyModsRemapSetInitMods(from, side))
					// Both sides of from map to the left sided to.
					expectEqual(t, inputKeyModsRemapSetInitMods(to, kmLeft), got)
				}
			}()
		}
	}
}

// Zig: test "term51: RemapSet.apply: pressed target keeps its own side bit"
func TestKm_term51_RemapSet_apply_pressed_target_keeps_its_own_side_bit(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "left_ctrl=alt")
	set.Finalize()

	// Right alt is already pressed: ctrl becomes (left) alt, but the side
	// bit of alt stays right because the to bits are OR-ed in.
	input := kmMods{ctrl: true, alt: true, sides: kmSides{alt: kmRight}}
	expectEqual(t,
		kmMods{alt: true, sides: kmSides{alt: kmRight}},
		set.Apply(input),
	)
}

// Zig: test "term51: RemapSet.apply: first matching entry wins and finalize sets the order"
func TestKm_term51_RemapSet_apply_first_matching_entry_wins_and_finalize_sets_the_order(t *testing.T) {
	// ctrl=alt matches left ctrl, right_alt=shift matches right alt.
	// Both match {left ctrl, right alt}; the first entry in map order wins.
	input := kmMods{ctrl: true, alt: true, sides: kmSides{alt: kmRight}}

	// Without finalize the insertion order is used: left_ctrl first.
	{
		set := InputKeyModsRemapSetEmpty
		defer set.Deinit()
		kmParse(t, &set, "ctrl=alt")
		kmParse(t, &set, "right_alt=shift")
		expectEqual(t,
			kmMods{alt: true, sides: kmSides{alt: kmRight}},
			set.Apply(input),
		)
	}

	// With finalize the right sided entry moves to the front.
	{
		set := InputKeyModsRemapSetEmpty
		defer set.Deinit()
		kmParse(t, &set, "ctrl=alt")
		kmParse(t, &set, "right_alt=shift")
		set.Finalize()
		expectEqual(t,
			kmMods{ctrl: true, shift: true},
			set.Apply(input),
		)
	}
}

// Zig: test "term51: RemapSet.apply: swap of two mods"
func TestKm_term51_RemapSet_apply_swap_of_two_mods(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "ctrl=alt")
	kmParse(t, &set, "alt=ctrl")
	set.Finalize()

	expectEqual(t, kmMods{alt: true}, set.Apply(kmMods{ctrl: true}))
	expectEqual(t, kmMods{ctrl: true}, set.Apply(kmMods{alt: true}))
	// Only one entry is applied when both are pressed (the first in map
	// order, left_ctrl): ctrl is replaced by alt which is already set.
	expectEqual(t, kmMods{alt: true}, set.Apply(kmMods{ctrl: true, alt: true}))
}

// Zig: test "term51: RemapSet.apply: same mod target is the identity"
func TestKm_term51_RemapSet_apply_same_mod_target_is_the_identity(t *testing.T) {
	set := InputKeyModsRemapSetEmpty
	defer set.Deinit()
	kmParse(t, &set, "ctrl=ctrl")
	set.Finalize()
	expectEqual(t, kmMods{ctrl: true}, set.Apply(kmMods{ctrl: true}))

	// Right ctrl to (left) ctrl clears the right side bit.
	kmParse(t, &set, "right_ctrl=left_ctrl")
	set.Finalize()
	expectEqual(t,
		kmMods{ctrl: true},
		set.Apply(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}),
	)
}

// Zig: test "term51: RemapSet.Mask: default and layout"
func TestKm_term51_RemapSet_Mask_default_and_layout(t *testing.T) {
	mask := InputKeyModsRemapSetMask{}
	expectEqual(t, uint16(0), mask.Int())
	expectEqual(t, InputKeyModsModsKeys{}, mask.keys)
	expectEqual(t, kmSides{}, mask.left_sides)
	expectEqual(t, kmSides{}, mask.right_sides)

	// keys: bits 0-3, left_sides: 4-7, right_sides: 8-11.
	expectEqual(t, uint16(0x001), (InputKeyModsRemapSetMask{keys: InputKeyModsModsKeys{shift: true}}).Int())
	expectEqual(t, uint16(0x010), (InputKeyModsRemapSetMask{left_sides: kmSides{shift: kmRight}}).Int())
	expectEqual(t, uint16(0x100), (InputKeyModsRemapSetMask{right_sides: kmSides{shift: kmRight}}).Int())
}

// Zig: test "term51: RemapSet.Mask.update: left sided mod"
func TestKm_term51_RemapSet_Mask_update_left_sided_mod(t *testing.T) {
	mask := InputKeyModsRemapSetMask{}
	mask.Update(kmMods{ctrl: true})
	expectEqual(t, uint16(0x022), mask.Int())
	expectEqual(t, InputKeyModsModsKeys{ctrl: true}, mask.keys)
	expectEqual(t, kmSides{ctrl: kmRight}, mask.left_sides)
	expectEqual(t, kmSides{}, mask.right_sides)
}

// Zig: test "term51: RemapSet.Mask.update: right sided mod"
func TestKm_term51_RemapSet_Mask_update_right_sided_mod(t *testing.T) {
	mask := InputKeyModsRemapSetMask{}
	mask.Update(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}})
	expectEqual(t, uint16(0x202), mask.Int())
	expectEqual(t, InputKeyModsModsKeys{ctrl: true}, mask.keys)
	expectEqual(t, kmSides{}, mask.left_sides)
	expectEqual(t, kmSides{ctrl: kmRight}, mask.right_sides)
}

// Zig: test "term51: RemapSet.Mask.update: every mod and side"
func TestKm_term51_RemapSet_Mask_update_every_mod_and_side(t *testing.T) {
	for i, mod := range kmMods4 {
		bit := uint16(1) << i
		left := InputKeyModsRemapSetMask{}
		left.Update(inputKeyModsRemapSetInitMods(mod, kmLeft))
		expectEqual(t, bit|(bit<<4), left.Int())

		right := InputKeyModsRemapSetMask{}
		right.Update(inputKeyModsRemapSetInitMods(mod, kmRight))
		expectEqual(t, bit|(bit<<8), right.Int())
	}
}

// Zig: test "term51: RemapSet.Mask.update: accumulates with OR"
func TestKm_term51_RemapSet_Mask_update_accumulates_with_OR(t *testing.T) {
	mask := InputKeyModsRemapSetMask{}
	mask.Update(kmMods{ctrl: true})
	mask.Update(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}})
	expectEqual(t, uint16(0x222), mask.Int())

	// Updating again changes nothing.
	mask.Update(kmMods{ctrl: true})
	mask.Update(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}})
	expectEqual(t, uint16(0x222), mask.Int())

	// Another mod only adds its own bits.
	mask.Update(kmMods{super: true, sides: kmSides{super: kmRight}})
	expectEqual(t, uint16(0xA22|0x008|0x000), mask.Int())
}

// Zig: test "term51: RemapSet.Mask.update: multiple keys in one value"
func TestKm_term51_RemapSet_Mask_update_multiple_keys_in_one_value(t *testing.T) {
	mask := InputKeyModsRemapSetMask{}
	mask.Update(kmMods{shift: true, alt: true, sides: kmSides{alt: kmRight}})
	// keys 0b0101, left 0b0001 (shift on the left), right 0b0100 (alt).
	expectEqual(t, uint16(0x415), mask.Int())
}

// Zig: test "term51: RemapSet.Mask.update: ignores locks, stray sides and empty mods"
func TestKm_term51_RemapSet_Mask_update_ignores_locks_stray_sides_and_empty_mods(t *testing.T) {
	mask := InputKeyModsRemapSetMask{}
	mask.Update(kmMods{})
	expectEqual(t, uint16(0), mask.Int())

	// Side bits without a key bit do nothing.
	mask.Update(InputKeyModsModsSideMask)
	expectEqual(t, uint16(0), mask.Int())

	// Locks are not keys.
	mask.Update(kmMods{caps_lock: true, num_lock: true})
	expectEqual(t, uint16(0), mask.Int())

	// A stray side bit of a mod that is not set does not leak in.
	mask.Update(kmMods{ctrl: true, sides: kmSides{alt: kmRight}})
	expectEqual(t, uint16(0x022), mask.Int())
}

// Zig: test "term51: RemapSet.Mask.match: left sided remap"
func TestKm_term51_RemapSet_Mask_match_left_sided_remap(t *testing.T) {
	mask := InputKeyModsRemapSetMask{}
	mask.Update(kmMods{ctrl: true})

	expect(t, mask.Match(kmMods{ctrl: true}), "mask.match(.{ .ctrl = true })")
	expect(t, mask.Match(kmMods{ctrl: true, shift: true}), "mask.match(.{ .ctrl = true, .shift = true })")
	expect(t, mask.Match(kmMods{ctrl: true, caps_lock: true}), "mask.match(.{ .ctrl = true, .caps_lock = true })")
	expect(t, !mask.Match(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}), "!mask.match(.{ .ctrl = true, .sides = .{ .ctrl = .right } })")
	expect(t, !mask.Match(kmMods{alt: true}), "!mask.match(.{ .alt = true })")
	expect(t, !mask.Match(kmMods{shift: true, super: true}), "!mask.match(.{ .shift = true, .super = true })")
	expect(t, !mask.Match(kmMods{}), "!mask.match(.{})")
	expect(t, !mask.Match(kmMods{caps_lock: true, num_lock: true}), "!mask.match(.{ .caps_lock = true, .num_lock = true })")
	// The side bit of ctrl without ctrl pressed.
	expect(t, !mask.Match(kmMods{sides: kmSides{ctrl: kmRight}}), "!mask.match(.{ .sides = .{ .ctrl = .right } })")
}

// Zig: test "term51: RemapSet.Mask.match: right sided remap"
func TestKm_term51_RemapSet_Mask_match_right_sided_remap(t *testing.T) {
	mask := InputKeyModsRemapSetMask{}
	mask.Update(kmMods{alt: true, sides: kmSides{alt: kmRight}})

	expect(t, mask.Match(kmMods{alt: true, sides: kmSides{alt: kmRight}}), "mask.match(.{ .alt = true, .sides = .{ .alt = .right } })")
	expect(t, !mask.Match(kmMods{alt: true}), "!mask.match(.{ .alt = true })")
	expect(t, !mask.Match(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}), "!mask.match(.{ .ctrl = true, .sides = .{ .ctrl = .right } })")
	expect(t, mask.Match(kmMods{alt: true, ctrl: true, sides: kmSides{alt: kmRight}}), "mask.match(.{ .alt = true, .ctrl = true, .sides = .{ .alt = .right } })")
}

// Zig: test "term51: RemapSet.Mask.match: both sides remapped"
func TestKm_term51_RemapSet_Mask_match_both_sides_remapped(t *testing.T) {
	mask := InputKeyModsRemapSetMask{}
	mask.Update(kmMods{super: true})
	mask.Update(kmMods{super: true, sides: kmSides{super: kmRight}})

	expect(t, mask.Match(kmMods{super: true}), "mask.match(.{ .super = true })")
	expect(t, mask.Match(kmMods{super: true, sides: kmSides{super: kmRight}}), "mask.match(.{ .super = true, .sides = .{ .super = .right } })")
	expect(t, !mask.Match(kmMods{ctrl: true}), "!mask.match(.{ .ctrl = true })")
}

// Zig: test "term51: RemapSet.Mask.match: sides are tracked per mod"
func TestKm_term51_RemapSet_Mask_match_sides_are_tracked_per_mod(t *testing.T) {
	// left ctrl and right alt are remapped.
	mask := InputKeyModsRemapSetMask{}
	mask.Update(kmMods{ctrl: true})
	mask.Update(kmMods{alt: true, sides: kmSides{alt: kmRight}})

	expect(t, mask.Match(kmMods{ctrl: true}), "mask.match(.{ .ctrl = true })")
	expect(t, mask.Match(kmMods{alt: true, sides: kmSides{alt: kmRight}}), "mask.match(.{ .alt = true, .sides = .{ .alt = .right } })")
	expect(t, mask.Match(kmMods{
		ctrl:  true,
		alt:   true,
		sides: kmSides{alt: kmRight},
	}), "mask.match(.{ .ctrl = true, .alt = true, .sides = .{ .alt = .right } })")
	// Right ctrl and left alt: neither remapped.
	expect(t, !mask.Match(kmMods{
		ctrl:  true,
		alt:   true,
		sides: kmSides{ctrl: kmRight},
	}), "!mask.match(.{ .ctrl = true, .alt = true, .sides = .{ .ctrl = .right } })")
	expect(t, !mask.Match(kmMods{alt: true}), "!mask.match(.{ .alt = true })")
	// Right ctrl with right alt: alt matches.
	expect(t, mask.Match(kmMods{
		ctrl:  true,
		alt:   true,
		sides: kmSides{ctrl: kmRight, alt: kmRight},
	}), "mask.match(.{ .ctrl = true, .alt = true, .sides = .{ .ctrl = .right, .alt = .right } })")
	// Left ctrl with left alt: ctrl matches.
	expect(t, mask.Match(kmMods{ctrl: true, alt: true}), "mask.match(.{ .ctrl = true, .alt = true })")
}

// Zig: test "term51: RemapSet.Mask.match: every mod and side against every mask"
func TestKm_term51_RemapSet_Mask_match_every_mod_and_side_against_every_mask(t *testing.T) {
	for _, mask_mod := range kmMods4 {
		for _, mask_side := range kmSides2 {
			mask := InputKeyModsRemapSetMask{}
			mask.Update(inputKeyModsRemapSetInitMods(mask_mod, mask_side))
			for _, mod := range kmMods4 {
				for _, side := range kmSides2 {
					expected := mod == mask_mod && side == mask_side
					expectEqual(t,
						expected,
						mask.Match(inputKeyModsRemapSetInitMods(mod, side)),
					)
				}
			}
		}
	}
}

// Zig: test "term51: RemapSet.Mask.match: empty mask and inconsistent mask"
func TestKm_term51_RemapSet_Mask_match_empty_mask_and_inconsistent_mask(t *testing.T) {
	empty := InputKeyModsRemapSetMask{}
	expect(t, !empty.Match(kmMods{}), "!empty.match(.{})")
	expect(t, !empty.Match(kmMods{ctrl: true}), "!empty.match(.{ .ctrl = true })")
	expect(t, !empty.Match(kmModsFromInt(0xFFFF)), "!empty.match(term51ADModsFromInt(0xFFFF))")

	// A key is set but no side is: nothing can match.
	keys_only := InputKeyModsRemapSetMask{keys: InputKeyModsModsKeys{ctrl: true}}
	expect(t, !keys_only.Match(kmMods{ctrl: true}), "!keys_only.match(.{ .ctrl = true })")
	expect(t, !keys_only.Match(kmMods{ctrl: true, sides: kmSides{ctrl: kmRight}}), "!keys_only.match(.{ .ctrl = true, .sides = .{ .ctrl = .right } })")

	// A side is set but the key is not.
	sides_only := InputKeyModsRemapSetMask{left_sides: kmSides{ctrl: kmRight}}
	expect(t, !sides_only.Match(kmMods{ctrl: true}), "!sides_only.match(.{ .ctrl = true })")
}

// Go-only: panic test: RemapSet.apply: the mask matched but no entry did (unreachable)
func TestKm_panic_RemapSet_apply_mask_matched_but_no_entry_did(t *testing.T) {
	// The mask says ctrl is remapped but the map is empty: Zig falls off the
	// end of the loop into `unreachable`.
	set := InputKeyModsRemapSetEmpty
	set.mask.Update(kmMods{ctrl: true})
	kmExpectPanic(t, func() {
		set.Apply(kmMods{ctrl: true})
	})
}
