// Tests translated one for one from src/terminal/kitty/key.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order (helpers in Parser_test.go).
//   - Zig's `u5` (the packed Flags) is a uint8 and `u3` (FlagStack.idx) is a
//     uint8 that the implementation keeps below 8. `term51EFlags` (the
//     `@bitCast` from u5) is sm1_flags, which calls KittyKeyFlagsFromInt.
//   - Zig's `idx +%= 1` and `idx -%= 1` on a u3 are written as explicit
//     arithmetic modulo 8 in the one test that does them in the test body.
//   - `Flags.disabled` and `Flags.true` are KittyKeyFlagsDisabled and
//     KittyKeyFlagsTrue; `.set`, `.@"or"`, `.not` are the
//     KittyKeySetMode constants.
//   - `@typeInfo(SetMode).@"enum".fields` has no Go counterpart: a field name
//     is the String() of the value, and the field count is the last value
//     plus one.

package terminal

import (
	"testing"
)

// sm1_flags is term51EFlags: build flags from their 5-bit integer form.
func sm1_flags(n uint8) KittyKeyFlags {
	return KittyKeyFlagsFromInt(n)
}

// Zig: test (unnamed, key.zig:65)
func TestKittyKey_unnamed_65(t *testing.T) {
	// Make sure we the overflow works as expected
	stack := KittyKeyFlagStack{}
	stack.idx = uint8(len(stack.flags) - 1)
	stack.idx = (stack.idx + 1) % 8 // Zig: stack.idx +%= 1, a u3
	expect(t, stack.idx == 0, "stack.idx == 0")

	stack.idx = 0
	stack.idx = (stack.idx + 7) % 8 // Zig: stack.idx -%= 1, a u3
	expect(t, stack.idx == uint8(len(stack.flags)-1), "stack.idx == stack.flags.len - 1")
}

// Zig: test (unnamed, key.zig:109)
func TestKittyKey_unnamed_109(t *testing.T) {
	// Its easy to get packed struct ordering wrong so this test checks.
	expectEqual(t, uint8(0b1), KittyKeyFlags{Disambiguate: true}.Int())
	expectEqual(t, uint8(0b10), KittyKeyFlags{ReportEvents: true}.Int())
}

// Zig: test "FlagStack: push pop"
func TestKittyKey_FlagStack_push_pop(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Push(KittyKeyFlags{Disambiguate: true})
	expectEqual(t, KittyKeyFlags{Disambiguate: true}, stack.Current())

	stack.Pop(1)
	expectEqual(t, KittyKeyFlags{}, stack.Current())
}

// Zig: test "FlagStack: pop big number"
func TestKittyKey_FlagStack_pop_big_number(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Pop(100)
	expectEqual(t, KittyKeyFlags{}, stack.Current())
}

// Zig: test "FlagStack: set"
func TestKittyKey_FlagStack_set(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Set(KittyKeySetModeSet, KittyKeyFlags{Disambiguate: true})
	expectEqual(t, KittyKeyFlags{Disambiguate: true}, stack.Current())

	stack.Set(KittyKeySetModeOr, KittyKeyFlags{ReportEvents: true})
	expectEqual(t, KittyKeyFlags{
		Disambiguate: true,
		ReportEvents: true,
	}, stack.Current())

	stack.Set(KittyKeySetModeNot, KittyKeyFlags{ReportEvents: true})
	expectEqual(t, KittyKeyFlags{Disambiguate: true}, stack.Current())
}

// Zig: test "term51: Flags.int: each flag has its own bit"
func TestKittyKey_term51_Flags_int_each_flag_has_its_own_bit(t *testing.T) {
	expectEqual(t, uint8(0b00001), KittyKeyFlags{Disambiguate: true}.Int())
	expectEqual(t, uint8(0b00010), KittyKeyFlags{ReportEvents: true}.Int())
	expectEqual(t, uint8(0b00100), KittyKeyFlags{ReportAlternates: true}.Int())
	expectEqual(t, uint8(0b01000), KittyKeyFlags{ReportAll: true}.Int())
	expectEqual(t, uint8(0b10000), KittyKeyFlags{ReportAssociated: true}.Int())
	expectEqual(t, uint8(0), KittyKeyFlags{}.Int())
}

// Zig: test "term51: Flags.int: constants and round trip over every value"
func TestKittyKey_term51_Flags_int_constants_and_round_trip_over_every_value(t *testing.T) {
	expectEqual(t, uint8(0), KittyKeyFlagsDisabled.Int())
	expectEqual(t, uint8(31), KittyKeyFlagsTrue.Int())
	expectEqual(t, KittyKeyFlags{}, KittyKeyFlagsDisabled)
	// Zig-only: try testing.expectEqual(@as(usize, 5), @bitSizeOf(Flags)); (layout)

	var n uint8 = 0
	for ; n < 32; n++ {
		f := sm1_flags(n)
		expectEqual(t, n, f.Int())
		expectEqual(t, n&1 != 0, f.Disambiguate)
		expectEqual(t, n&2 != 0, f.ReportEvents)
		expectEqual(t, n&4 != 0, f.ReportAlternates)
		expectEqual(t, n&8 != 0, f.ReportAll)
		expectEqual(t, n&16 != 0, f.ReportAssociated)
	}
}

// Zig: test "term51: SetMode: names and order"
func TestKittyKey_term51_SetMode_names_and_order(t *testing.T) {
	expectEqual(t, 3, int(KittyKeySetModeNot)+1) // f.len
	expectEqualStrings(t, "set", KittyKeySetMode(0).String())
	expectEqualStrings(t, "or", KittyKeySetMode(1).String())
	expectEqualStrings(t, "not", KittyKeySetMode(2).String())
	expectEqual(t, 0, int(KittyKeySetModeSet))
	expectEqual(t, 1, int(KittyKeySetModeOr))
	expectEqual(t, 2, int(KittyKeySetModeNot))
}

// Zig: test "term51: FlagStack: layout and default state"
func TestKittyKey_term51_FlagStack_layout_and_default_state(t *testing.T) {
	stack := KittyKeyFlagStack{}
	expectEqual(t, 8, kittyKeyFlagStackLen)
	expectEqual(t, 8, len(stack.flags))
	// Zig-only: expect(@TypeOf(stack.idx) == u3) (Go has no u3; idx is a uint8)
	expectEqual(t, uint8(0), stack.idx)
	for _, f := range stack.flags {
		expectEqual(t, KittyKeyFlagsDisabled, f)
	}
	expectEqual(t, KittyKeyFlagsDisabled, stack.Current())
}

// Zig: test "term51: FlagStack.current: reads the slot at idx"
func TestKittyKey_term51_FlagStack_current_reads_the_slot_at_idx(t *testing.T) {
	stack := KittyKeyFlagStack{}
	for i := range stack.flags {
		stack.flags[i] = sm1_flags(uint8(i + 1))
	}
	var i uint8 = 0
	for ; i < 8; i++ {
		stack.idx = i
		expectEqual(t, sm1_flags(i+1), stack.Current())
	}
}

// Zig: test "term51: FlagStack.set: set replaces the current flags"
func TestKittyKey_term51_FlagStack_set_set_replaces_the_current_flags(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Set(KittyKeySetModeSet, sm1_flags(0b10101))
	expectEqual(t, sm1_flags(0b10101), stack.Current())
	stack.Set(KittyKeySetModeSet, sm1_flags(0b00010))
	expectEqual(t, sm1_flags(0b00010), stack.Current())
	stack.Set(KittyKeySetModeSet, KittyKeyFlagsDisabled)
	expectEqual(t, KittyKeyFlagsDisabled, stack.Current())
	stack.Set(KittyKeySetModeSet, KittyKeyFlagsTrue)
	expectEqual(t, KittyKeyFlagsTrue, stack.Current())
}

// Zig: test "term51: FlagStack.set: or adds bits and not removes bits"
func TestKittyKey_term51_FlagStack_set_or_adds_bits_and_not_removes_bits(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Set(KittyKeySetModeSet, sm1_flags(0b00110))

	stack.Set(KittyKeySetModeOr, sm1_flags(0b01100))
	expectEqual(t, sm1_flags(0b01110), stack.Current())
	// Or with bits already set and with nothing.
	stack.Set(KittyKeySetModeOr, sm1_flags(0b00100))
	expectEqual(t, sm1_flags(0b01110), stack.Current())
	stack.Set(KittyKeySetModeOr, KittyKeyFlagsDisabled)
	expectEqual(t, sm1_flags(0b01110), stack.Current())

	stack.Set(KittyKeySetModeNot, sm1_flags(0b00110))
	expectEqual(t, sm1_flags(0b01000), stack.Current())
	// Not of bits already clear and of nothing.
	stack.Set(KittyKeySetModeNot, sm1_flags(0b00111))
	expectEqual(t, sm1_flags(0b01000), stack.Current())
	stack.Set(KittyKeySetModeNot, KittyKeyFlagsDisabled)
	expectEqual(t, sm1_flags(0b01000), stack.Current())
	stack.Set(KittyKeySetModeNot, KittyKeyFlagsTrue)
	expectEqual(t, KittyKeyFlagsDisabled, stack.Current())
}

// Zig: test "term51: FlagStack.set: exhaustive over all pairs of 5-bit values"
func TestKittyKey_term51_FlagStack_set_exhaustive_over_all_pairs_of_5_bit_values(t *testing.T) {
	var a uint8 = 0
	for ; a < 32; a++ {
		var b uint8 = 0
		for ; b < 32; b++ {
			cur := sm1_flags(a)
			v := sm1_flags(b)
			stack := KittyKeyFlagStack{}

			stack.flags[0] = cur
			stack.Set(KittyKeySetModeSet, v)
			expectEqual(t, b, stack.Current().Int())

			stack.flags[0] = cur
			stack.Set(KittyKeySetModeOr, v)
			expectEqual(t, a|b, stack.Current().Int())

			stack.flags[0] = cur
			stack.Set(KittyKeySetModeNot, v)
			expectEqual(t, a&^b&31, stack.Current().Int())
		}
	}
}

// Zig: test "term51: FlagStack.set: only the slot at idx changes"
func TestKittyKey_term51_FlagStack_set_only_the_slot_at_idx_changes(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.idx = 5
	stack.Set(KittyKeySetModeSet, sm1_flags(3))
	for i, f := range stack.flags {
		if i == 5 {
			expectEqual(t, sm1_flags(3), f)
		} else {
			expectEqual(t, KittyKeyFlagsDisabled, f)
		}
	}
	stack.Set(KittyKeySetModeOr, sm1_flags(8))
	expectEqual(t, sm1_flags(11), stack.flags[5])
	stack.Set(KittyKeySetModeNot, sm1_flags(1))
	expectEqual(t, sm1_flags(10), stack.flags[5])
	expectEqual(t, uint8(5), stack.idx)
}

// Zig: test "term51: FlagStack.push: moves idx up and stores the given flags"
func TestKittyKey_term51_FlagStack_push_moves_idx_up_and_stores_the_given_flags(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Set(KittyKeySetModeSet, sm1_flags(1))

	stack.Push(sm1_flags(2))
	expectEqual(t, uint8(1), stack.idx)
	expectEqual(t, sm1_flags(2), stack.Current())
	// The lower entry is retained, and the new entry is not derived from it.
	expectEqual(t, sm1_flags(1), stack.flags[0])

	stack.Push(KittyKeyFlagsDisabled)
	expectEqual(t, uint8(2), stack.idx)
	expectEqual(t, KittyKeyFlagsDisabled, stack.Current())
	expectEqual(t, sm1_flags(2), stack.flags[1])
}

// Zig: test "term51: FlagStack.push: depth of eight then the oldest entry is evicted"
func TestKittyKey_term51_FlagStack_push_depth_of_eight_then_the_oldest_entry_is_evicted(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Set(KittyKeySetModeSet, sm1_flags(1)) // slot 0

	// Seven pushes fill slots 1..7.
	var n uint8 = 2
	for ; n <= 8; n++ {
		stack.Push(sm1_flags(n))
		expectEqual(t, n-1, stack.idx)
		expectEqual(t, sm1_flags(n), stack.Current())
	}
	for i, f := range stack.flags {
		expectEqual(t, sm1_flags(uint8(i+1)), f)
	}

	// The eighth push wraps to slot 0 and evicts the original entry.
	stack.Push(sm1_flags(9))
	expectEqual(t, uint8(0), stack.idx)
	expectEqual(t, sm1_flags(9), stack.Current())
	expectEqual(t, sm1_flags(9), stack.flags[0])
	expectEqual(t, sm1_flags(2), stack.flags[1])
	expectEqual(t, sm1_flags(8), stack.flags[7])

	// The next evicts slot 1.
	stack.Push(sm1_flags(10))
	expectEqual(t, uint8(1), stack.idx)
	expectEqual(t, sm1_flags(10), stack.flags[1])
	expectEqual(t, sm1_flags(9), stack.flags[0])

	// A full cycle of eight more returns idx to where it started.
	var i int = 0
	for ; i < 8; i++ {
		stack.Push(sm1_flags(31))
	}
	expectEqual(t, uint8(1), stack.idx)
}

// Zig: test "term51: FlagStack.pop: zero pops nothing"
func TestKittyKey_term51_FlagStack_pop_zero_pops_nothing(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Push(sm1_flags(3))
	stack.Pop(0)
	expectEqual(t, uint8(1), stack.idx)
	expectEqual(t, sm1_flags(3), stack.Current())
	expectEqual(t, sm1_flags(3), stack.flags[1])
}

// Zig: test "term51: FlagStack.pop: clears the popped slot and moves idx down"
func TestKittyKey_term51_FlagStack_pop_clears_the_popped_slot_and_moves_idx_down(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Set(KittyKeySetModeSet, sm1_flags(1))
	stack.Push(sm1_flags(2))
	stack.Push(sm1_flags(3))
	stack.Push(sm1_flags(4))
	expectEqual(t, uint8(3), stack.idx)

	stack.Pop(1)
	expectEqual(t, uint8(2), stack.idx)
	expectEqual(t, sm1_flags(3), stack.Current())
	expectEqual(t, KittyKeyFlagsDisabled, stack.flags[3])

	stack.Pop(2)
	expectEqual(t, uint8(0), stack.idx)
	expectEqual(t, sm1_flags(1), stack.Current())
	expectEqual(t, KittyKeyFlagsDisabled, stack.flags[2])
	expectEqual(t, KittyKeyFlagsDisabled, stack.flags[1])
}

// Zig: test "term51: FlagStack.pop: popping the last entry clears slot zero and wraps idx"
func TestKittyKey_term51_FlagStack_pop_popping_the_last_entry_clears_slot_zero_and_wraps_idx(t *testing.T) {
	// Empty stack: slot 0 is cleared and idx wraps to 7, whose flags are
	// whatever is stored there (disabled for a fresh stack).
	fresh := KittyKeyFlagStack{}
	fresh.Set(KittyKeySetModeSet, sm1_flags(5))
	fresh.Pop(1)
	expectEqual(t, uint8(7), fresh.idx)
	expectEqual(t, KittyKeyFlagsDisabled, fresh.flags[0])
	expectEqual(t, KittyKeyFlagsDisabled, fresh.Current())

	// After eight pushes the stack is full and idx is back at 0. Popping
	// once clears slot 0 and exposes the stale entry in slot 7.
	full := KittyKeyFlagStack{}
	var n uint8 = 1
	for ; n <= 8; n++ {
		full.Push(sm1_flags(n))
	}
	expectEqual(t, uint8(0), full.idx)
	expectEqual(t, sm1_flags(8), full.Current())
	full.Pop(1)
	expectEqual(t, uint8(7), full.idx)
	expectEqual(t, KittyKeyFlagsDisabled, full.flags[0])
	expectEqual(t, sm1_flags(7), full.Current())
	full.Pop(1)
	expectEqual(t, uint8(6), full.idx)
	expectEqual(t, KittyKeyFlagsDisabled, full.flags[7])
	expectEqual(t, sm1_flags(6), full.Current())
}

// Zig: test "term51: FlagStack.pop: seven pops from slot zero leave slot one untouched"
func TestKittyKey_term51_FlagStack_pop_seven_pops_from_slot_zero_leave_slot_one_untouched(t *testing.T) {
	stack := KittyKeyFlagStack{}
	var n uint8 = 1
	for ; n <= 8; n++ {
		stack.Push(sm1_flags(n))
	}
	// Slot 0 holds the eighth push (8); slot i holds the value i otherwise.
	for i, f := range stack.flags {
		var want uint8 = uint8(i)
		if i == 0 {
			want = 8
		}
		expectEqual(t, sm1_flags(want), f)
	}
	expectEqual(t, uint8(0), stack.idx)

	stack.Pop(7)
	// Slots 0, 7, 6, 5, 4, 3, 2 are cleared; slot 1 is not.
	expectEqual(t, uint8(1), stack.idx)
	for i, f := range stack.flags {
		if i == 1 {
			expectEqual(t, sm1_flags(1), f)
		} else {
			expectEqual(t, KittyKeyFlagsDisabled, f)
		}
	}
	expectEqual(t, sm1_flags(1), stack.Current())
}

// Zig: test "term51: FlagStack.pop: n of at least the stack length resets everything"
func TestKittyKey_term51_FlagStack_pop_n_of_at_least_the_stack_length_resets_everything(t *testing.T) {
	counts := [...]uint{8, 9, 15, 16, 100, ^uint(0)} // Zig: std.math.maxInt(usize)
	for _, count := range counts {
		stack := KittyKeyFlagStack{}
		var n uint8 = 1
		for ; n <= 5; n++ {
			stack.Push(sm1_flags(n))
		}
		expectEqual(t, uint8(5), stack.idx)

		stack.Pop(count)
		expectEqual(t, uint8(0), stack.idx)
		for _, f := range stack.flags {
			expectEqual(t, KittyKeyFlagsDisabled, f)
		}
		expectEqual(t, KittyKeyFlagsDisabled, stack.Current())
	}
}

// Zig: test "term51: FlagStack.pop: a count of one below the length unwinds step by step"
func TestKittyKey_term51_FlagStack_pop_a_count_of_one_below_the_length_unwinds_step_by_step(t *testing.T) {
	// Start at idx 5 with every slot set.
	stack := KittyKeyFlagStack{}
	for i := range stack.flags {
		stack.flags[i] = sm1_flags(uint8(i + 1))
	}
	stack.idx = 5

	stack.Pop(7)
	// idx moves 5 -> 6 (mod 8: 5 - 7 = -2 = 6).
	expectEqual(t, uint8(6), stack.idx)
	// Cleared slots: 5,4,3,2,1,0,7. Slot 6 survives.
	for i, f := range stack.flags {
		if i == 6 {
			expectEqual(t, sm1_flags(7), f)
		} else {
			expectEqual(t, KittyKeyFlagsDisabled, f)
		}
	}
}

// Zig: test "term51: FlagStack.pop: pop count table from idx zero"
func TestKittyKey_term51_FlagStack_pop_pop_count_table_from_idx_zero(t *testing.T) {
	// For pops 0..7 from idx 0 with all slots set: idx becomes (8 - n) mod 8
	// and slots 0, 7, 6, ... (n of them) are cleared.
	var n uint = 0
	for ; n < 8; n++ {
		stack := KittyKeyFlagStack{}
		for i := range stack.flags {
			stack.flags[i] = sm1_flags(uint8(i + 1))
		}
		stack.Pop(n)
		expectEqual(t, uint8((8-n)%8), stack.idx)
		for i, f := range stack.flags {
			// Slot i is cleared if it is one of 0, 7, 6, ... visited n times.
			steps_from_zero := (8 - uint(i)) % 8 // 0 for slot 0, 1 for slot 7, ...
			cleared := steps_from_zero < n
			if cleared {
				expectEqual(t, KittyKeyFlagsDisabled, f)
			} else {
				expectEqual(t, sm1_flags(uint8(i+1)), f)
			}
		}
	}
}

// Zig: test "term51: FlagStack: push and pop round trip keeps lower entries"
func TestKittyKey_term51_FlagStack_push_and_pop_round_trip_keeps_lower_entries(t *testing.T) {
	stack := KittyKeyFlagStack{}
	stack.Set(KittyKeySetModeSet, sm1_flags(7))
	stack.Push(sm1_flags(1))
	stack.Push(sm1_flags(2))
	stack.Pop(2)
	expectEqual(t, uint8(0), stack.idx)
	expectEqual(t, sm1_flags(7), stack.Current())

	// The popped slots are cleared, so pushing again behaves identically.
	stack.Push(sm1_flags(3))
	expectEqual(t, sm1_flags(3), stack.Current())
	expectEqual(t, KittyKeyFlagsDisabled, stack.flags[2])
}

// Zig: test "term51: FlagStack: idx is a three-bit counter that wraps both ways"
func TestKittyKey_term51_FlagStack_idx_is_a_three_bit_counter_that_wraps_both_ways(t *testing.T) {
	stack := KittyKeyFlagStack{}
	var i int = 0
	for ; i < 20; i++ {
		stack.Push(sm1_flags(1))
		expectEqual(t, uint8((i+1)%8), stack.idx)
	}
	// Draining one at a time wraps downward.
	stack.idx = 0
	stack.Pop(1)
	expectEqual(t, uint8(7), stack.idx)
	stack.Pop(1)
	expectEqual(t, uint8(6), stack.idx)
}
