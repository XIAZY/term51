// Tests translated one for one from src/terminal/Tabstops.zig (Ghostty v1.3.1
// plus the term51 additional tests). Each Go test carries the exact Zig title
// in a "// Zig: test" comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual keeps Zig's name and Zig's (expected, actual) argument
//     order (helpers in Parser_test.go). `try x` on an error is zigTry
//     (dcs_test.go).
//   - The Zig variable `t` (a Tabstops) is `ts` in Go, because `t` is the
//     *testing.T.
//   - Zig's `defer t.deinit(alloc)` is `defer ts.Deinit()`. Inside a loop
//     body it runs at the end of each iteration, so Go wraps the body in a
//     function literal.
//   - Allocator arguments are dropped; checks on allocator behaviour
//     (std.testing.FailingAllocator, leak detection) are `// Zig-only:`.
//   - Zig's `const Unit = u8` is tabstopsUnit, `masks` is tabstopsMasks,
//     `entry` and `index` are tabstopsEntry and tabstopsIndex.

package terminal

import (
	"testing"
)

// sm1_tabstopsInit is `try init(testing.allocator, cols, interval)`.
func sm1_tabstopsInit(t *testing.T, cols uint, interval uint) Tabstops {
	t.Helper()
	ts, err := TabstopsInit(cols, interval)
	zigTry(t, err)
	return ts
}

// Zig: test "Tabstops: basic"
func TestTabstops_Tabstops_basic(t *testing.T) {
	ts := Tabstops{}
	defer ts.Deinit()
	expectEqual(t, uint(0), tabstopsEntry(4))
	expectEqual(t, uint(1), tabstopsEntry(8))
	expectEqual(t, uint(0), tabstopsIndex(0))
	expectEqual(t, uint(1), tabstopsIndex(1))
	expectEqual(t, uint(1), tabstopsIndex(9))

	expectEqual(t, tabstopsUnit(0b00001000), tabstopsMasks[3])
	expectEqual(t, tabstopsUnit(0b00010000), tabstopsMasks[4])

	expect(t, !ts.Get(4), "!t.get(4)")
	ts.Set(4)
	expect(t, ts.Get(4), "t.get(4)")
	expect(t, !ts.Get(3), "!t.get(3)")

	ts.Reset(0)
	expect(t, !ts.Get(4), "!t.get(4)")

	ts.Set(4)
	expect(t, ts.Get(4), "t.get(4)")
	ts.Unset(4)
	expect(t, !ts.Get(4), "!t.get(4)")
}

// Zig: test "Tabstops: dynamic allocations"
func TestTabstops_Tabstops_dynamic_allocations(t *testing.T) {
	ts := Tabstops{}
	defer ts.Deinit()

	// Grow the capacity by 2.
	cap := ts.Capacity()
	zigTry(t, ts.Resize(cap*2))

	// Set something that was out of range of the first
	ts.Set(cap + 5)
	expect(t, ts.Get(cap+5), "t.get(cap + 5)")
	expect(t, !ts.Get(cap+4), "!t.get(cap + 4)")

	// Prealloc still works
	expect(t, !ts.Get(5), "!t.get(5)")
}

// Zig: test "Tabstops: interval"
func TestTabstops_Tabstops_interval(t *testing.T) {
	ts := sm1_tabstopsInit(t, 80, 4)
	defer ts.Deinit()
	expect(t, !ts.Get(0), "!t.get(0)")
	expect(t, ts.Get(4), "t.get(4)")
	expect(t, !ts.Get(5), "!t.get(5)")
	expect(t, ts.Get(8), "t.get(8)")
}

// Zig: test "Tabstops: count on 80"
func TestTabstops_Tabstops_count_on_80(t *testing.T) {
	// https://superuser.com/questions/710019/why-there-are-11-tabstops-on-a-80-column-console

	ts := sm1_tabstopsInit(t, 80, 8)
	defer ts.Deinit()

	// Count the tabstops
	count := func() uint {
		var v uint = 0
		var i uint = 0
		for ; i < 80; i++ {
			if ts.Get(i) {
				v += 1
			}
		}

		return v
	}()

	expectEqual(t, uint(9), count)
}

// Not translated, Zig test "Tabstops: resize alloc failure preserves state": see scripts/not-translated.txt (tripwire fault
// injection; no Go counterpart).

// Zig: test "term51: prealloc_columns: constants"
func TestTabstops_term51_prealloc_columns_constants(t *testing.T) {
	expectEqual(t, uint(512), uint(tabstopsPreallocColumns))
	expectEqual(t, uint(64), uint(tabstopsPreallocCount))
	expectEqual(t, uint(8), uint(tabstopsUnitBits))
	var unit tabstopsUnit
	var u8 uint8 = unit // Unit == u8
	_ = u8
	expectEqual(t, uint(64), uint(len(Tabstops{}.prealloc_stops)))
}

// Zig: test "term51: masks: one distinct bit per index"
func TestTabstops_term51_masks_one_distinct_bit_per_index(t *testing.T) {
	expectEqual(t, uint(8), uint(len(tabstopsMasks)))
	for i, m := range tabstopsMasks {
		expectEqual(t, tabstopsUnit(1)<<uint(i), m)
	}
	expectEqual(t, tabstopsUnit(0x01), tabstopsMasks[0])
	expectEqual(t, tabstopsUnit(0x80), tabstopsMasks[7])
}

// Zig: test "term51: entry: column divided by eight"
func TestTabstops_term51_entry_column_divided_by_eight(t *testing.T) {
	for col := uint(0); col < 1100; col++ {
		expectEqual(t, col/8, tabstopsEntry(col))
	}
	expectEqual(t, uint(0), tabstopsEntry(0))
	expectEqual(t, uint(0), tabstopsEntry(7))
	expectEqual(t, uint(1), tabstopsEntry(8))
	expectEqual(t, uint(63), tabstopsEntry(511))
	expectEqual(t, uint(64), tabstopsEntry(512))
}

// Zig: test "term51: index: column modulo eight"
func TestTabstops_term51_index_column_modulo_eight(t *testing.T) {
	for col := uint(0); col < 1100; col++ {
		expectEqual(t, col%8, tabstopsIndex(col))
	}
	expectEqual(t, uint(7), tabstopsIndex(7))
	expectEqual(t, uint(0), tabstopsIndex(8))
	expectEqual(t, uint(7), tabstopsIndex(511))
	expectEqual(t, uint(0), tabstopsIndex(512))
}

// Zig: test "term51: Tabstops: layout default value"
func TestTabstops_term51_Tabstops_layout_default_value(t *testing.T) {
	ts := Tabstops{}
	expectEqual(t, uint(0), ts.cols)
	expectEqual(t, uint(0), uint(len(ts.dynamic_stops)))
	expectEqual(t, uint(512), ts.Capacity())
	for _, u := range ts.prealloc_stops {
		expectEqual(t, tabstopsUnit(0), u)
	}
}

// Zig: test "term51: Tabstops.set: bit layout is column / 8 byte and column % 8 bit"
func TestTabstops_term51_Tabstops_set_bit_layout_is_column_8_byte_and_column_8_bit(t *testing.T) {
	ts := Tabstops{}
	defer ts.Deinit()

	ts.Set(0)
	expectEqual(t, tabstopsUnit(0b0000_0001), ts.prealloc_stops[0])
	ts.Set(7)
	expectEqual(t, tabstopsUnit(0b1000_0001), ts.prealloc_stops[0])
	expectEqual(t, tabstopsUnit(0), ts.prealloc_stops[1])
	ts.Set(8)
	expectEqual(t, tabstopsUnit(0b0000_0001), ts.prealloc_stops[1])
	ts.Set(511)
	expectEqual(t, tabstopsUnit(0b1000_0000), ts.prealloc_stops[63])

	// Setting again is idempotent.
	ts.Set(511)
	ts.Set(0)
	expectEqual(t, tabstopsUnit(0b1000_0000), ts.prealloc_stops[63])
	expectEqual(t, tabstopsUnit(0b1000_0001), ts.prealloc_stops[0])
}

// Zig: test "term51: Tabstops.set: every prealloc column sets exactly its own bit"
func TestTabstops_term51_Tabstops_set_every_prealloc_column_sets_exactly_its_own_bit(t *testing.T) {
	ts := Tabstops{}
	defer ts.Deinit()
	for col := uint(0); col < 512; col++ {
		ts.Reset(0)
		ts.Set(col)
		for other := uint(0); other < 512; other++ {
			expectEqual(t, other == col, ts.Get(other))
		}
	}
}

// Zig: test "term51: Tabstops.set: dynamic columns use the dynamic storage"
func TestTabstops_term51_Tabstops_set_dynamic_columns_use_the_dynamic_storage(t *testing.T) {
	ts := sm1_tabstopsInit(t, 600, 0)
	defer ts.Deinit()

	ts.Set(512)
	expectEqual(t, tabstopsUnit(0b0000_0001), ts.dynamic_stops[0])
	ts.Set(519)
	expectEqual(t, tabstopsUnit(0b1000_0001), ts.dynamic_stops[0])
	ts.Set(520)
	expectEqual(t, tabstopsUnit(0b0000_0001), ts.dynamic_stops[1])
	ts.Set(599)
	expectEqual(t, tabstopsUnit(0b1000_0000), ts.dynamic_stops[74-64])

	// Preallocated storage is untouched.
	for _, u := range ts.prealloc_stops {
		expectEqual(t, tabstopsUnit(0), u)
	}
	expect(t, ts.Get(512), "t.get(512)")
	expect(t, ts.Get(519), "t.get(519)")
	expect(t, ts.Get(520), "t.get(520)")
	expect(t, ts.Get(599), "t.get(599)")
	expect(t, !ts.Get(513), "!t.get(513)")
	expect(t, !ts.Get(598), "!t.get(598)")
}

// Zig: test "term51: Tabstops.get: columns past cols but inside capacity are addressable"
func TestTabstops_term51_Tabstops_get_columns_past_cols_but_inside_capacity_are_addressable(t *testing.T) {
	// cols is not consulted by set/get; only the storage capacity matters.
	ts := Tabstops{}
	defer ts.Deinit()
	expectEqual(t, uint(0), ts.cols)
	expect(t, !ts.Get(0), "!t.get(0)")
	expect(t, !ts.Get(511), "!t.get(511)")
	ts.Set(511)
	expect(t, ts.Get(511), "t.get(511)")
	expect(t, !ts.Get(510), "!t.get(510)")
	expectEqual(t, uint(0), ts.cols)
}

// Zig: test "term51: Tabstops.get: reads each unit boundary independently"
func TestTabstops_term51_Tabstops_get_reads_each_unit_boundary_independently(t *testing.T) {
	ts := Tabstops{}
	defer ts.Deinit()
	ts.Set(7)
	expect(t, ts.Get(7), "t.get(7)")
	expect(t, !ts.Get(6), "!t.get(6)")
	expect(t, !ts.Get(8), "!t.get(8)")
	ts.Set(8)
	expect(t, ts.Get(8), "t.get(8)")
	expect(t, !ts.Get(9), "!t.get(9)")
	expect(t, ts.Get(7), "t.get(7)")
}

// Zig: test "term51: Tabstops.unset: clears a set column"
func TestTabstops_term51_Tabstops_unset_clears_a_set_column(t *testing.T) {
	ts := Tabstops{}
	defer ts.Deinit()
	ts.Set(3)
	ts.Set(4)
	ts.Unset(3)
	expect(t, !ts.Get(3), "!t.get(3)")
	expect(t, ts.Get(4), "t.get(4)")
	expectEqual(t, tabstopsUnit(0b0001_0000), ts.prealloc_stops[0])
}

// Zig: test "term51: Tabstops.unset: defect, unsetting a column with no stop toggles it on"
func TestTabstops_term51_Tabstops_unset_defect_unsetting_a_column_with_no_stop_toggles_it_on(t *testing.T) {
	// Upstream implements unset with XOR, so clearing a column that has no
	// tabstop SETS one. Pinned as-is.
	ts := Tabstops{}
	defer ts.Deinit()

	expect(t, !ts.Get(10), "!t.get(10)")
	ts.Unset(10)
	expect(t, ts.Get(10), "t.get(10)")
	expectEqual(t, tabstopsUnit(0b0000_0100), ts.prealloc_stops[1])

	// A second unset toggles it back off.
	ts.Unset(10)
	expect(t, !ts.Get(10), "!t.get(10)")
	expectEqual(t, tabstopsUnit(0), ts.prealloc_stops[1])

	// Neighbours are unaffected by the toggle.
	ts.Set(9)
	ts.Unset(10)
	expect(t, ts.Get(9), "t.get(9)")
	expect(t, ts.Get(10), "t.get(10)")
	expect(t, !ts.Get(11), "!t.get(11)")

	// Set then unset twice: off, then on again.
	ts.Reset(0)
	ts.Set(20)
	ts.Unset(20)
	expect(t, !ts.Get(20), "!t.get(20)")
	ts.Unset(20)
	expect(t, ts.Get(20), "t.get(20)")
}

// Zig: test "term51: Tabstops.unset: defect also applies to dynamic storage"
func TestTabstops_term51_Tabstops_unset_defect_also_applies_to_dynamic_storage(t *testing.T) {
	ts := sm1_tabstopsInit(t, 600, 0)
	defer ts.Deinit()

	expect(t, !ts.Get(530), "!t.get(530)")
	ts.Unset(530)
	expect(t, ts.Get(530), "t.get(530)")
	ts.Unset(530)
	expect(t, !ts.Get(530), "!t.get(530)")

	ts.Set(531)
	ts.Unset(531)
	expect(t, !ts.Get(531), "!t.get(531)")
	expectEqual(t, tabstopsUnit(0), ts.dynamic_stops[(531-512)/8])
}

// Zig: test "term51: Tabstops.init: sets cols and tabstops at the interval"
func TestTabstops_term51_Tabstops_init_sets_cols_and_tabstops_at_the_interval(t *testing.T) {
	ts := sm1_tabstopsInit(t, 80, 8)
	defer ts.Deinit()
	expectEqual(t, uint(80), ts.cols)
	expectEqual(t, uint(0), uint(len(ts.dynamic_stops)))
	expect(t, !ts.Get(0), "!t.get(0)")
	var c uint = 0
	for ; c < 80; c++ {
		expectEqual(t, c > 0 && c%8 == 0, ts.Get(c))
	}
}

// Zig: test "term51: Tabstops.init: zero columns with interval zero"
func TestTabstops_term51_Tabstops_init_zero_columns_with_interval_zero(t *testing.T) {
	ts := sm1_tabstopsInit(t, 0, 0)
	defer ts.Deinit()
	expectEqual(t, uint(0), ts.cols)
	expectEqual(t, uint(512), ts.Capacity())
	for col := uint(0); col < 512; col++ {
		expect(t, !ts.Get(col), "!t.get(col)")
	}
}

// Zig: test "term51: Tabstops.init: interval one, never at column zero or the last column"
func TestTabstops_term51_Tabstops_init_interval_one_never_at_column_zero_or_the_last_column(t *testing.T) {
	ts := sm1_tabstopsInit(t, 5, 1)
	defer ts.Deinit()
	expect(t, !ts.Get(0), "!t.get(0)")
	expect(t, ts.Get(1), "t.get(1)")
	expect(t, ts.Get(2), "t.get(2)")
	expect(t, ts.Get(3), "t.get(3)")
	expect(t, !ts.Get(4), "!t.get(4)")
}

// Not translated, Zig test "term51: Tabstops.init: out of memory when dynamic storage is needed": see scripts/not-translated.txt (FailingAllocator).

// Zig: test "term51: Tabstops.deinit: frees dynamic storage only when present"
func TestTabstops_term51_Tabstops_deinit_frees_dynamic_storage_only_when_present(t *testing.T) {
	// Zig-only: leak detection by the testing allocator is the check for the free.
	ts := sm1_tabstopsInit(t, 700, 8)
	expectEqual(t, uint(188), uint(len(ts.dynamic_stops)))
	ts.Deinit()

	// Without dynamic storage, deinit does not call free.
	// Zig-only: var failing = std.testing.FailingAllocator.init(testing.allocator, .{ .fail_index = 0 });
	u := sm1_tabstopsInit(t, 80, 8)
	u.Deinit()
	// Zig-only: try testing.expectEqual(@as(usize, 0), failing.freed_bytes);
}

// Zig: test "term51: Tabstops.resize: no allocation up to the preallocated column count"
func TestTabstops_term51_Tabstops_resize_no_allocation_up_to_the_preallocated_column_count(t *testing.T) {
	// Zig-only: var failing = std.testing.FailingAllocator.init(testing.allocator, .{ .fail_index = 0 });
	ts := Tabstops{}
	defer ts.Deinit()

	zigTry(t, ts.Resize(0))
	expectEqual(t, uint(0), ts.cols)
	zigTry(t, ts.Resize(1))
	expectEqual(t, uint(1), ts.cols)
	zigTry(t, ts.Resize(512))
	expectEqual(t, uint(512), ts.cols)
	expectEqual(t, uint(0), uint(len(ts.dynamic_stops)))
	expectEqual(t, uint(512), ts.Capacity())
	// Shrinking below the earlier size also works.
	zigTry(t, ts.Resize(3))
	expectEqual(t, uint(3), ts.cols)
}

// Zig: test "term51: Tabstops.resize: one past the preallocated count allocates dynamic storage"
func TestTabstops_term51_Tabstops_resize_one_past_the_preallocated_count_allocates_dynamic_storage(t *testing.T) {
	ts := Tabstops{}
	defer ts.Deinit()

	zigTry(t, ts.Resize(513))
	expectEqual(t, uint(513), ts.cols)
	// Upstream sizes the dynamic array in bytes as (cols - 512), although
	// each byte holds 8 columns, so it over-allocates 8x.
	expectEqual(t, uint(1), uint(len(ts.dynamic_stops)))
	expectEqual(t, uint((64+1)*8), ts.Capacity())
	for _, u := range ts.dynamic_stops {
		expectEqual(t, tabstopsUnit(0), u)
	}

	ts.Set(512)
	expect(t, ts.Get(512), "t.get(512)")
	ts.Set(519)
	expect(t, ts.Get(519), "t.get(519)")
}

// Zig: test "term51: Tabstops.resize: dynamic length is cols minus 512 bytes (over-allocation)"
func TestTabstops_term51_Tabstops_resize_dynamic_length_is_cols_minus_512_bytes_over_allocation(t *testing.T) {
	ts := Tabstops{}
	defer ts.Deinit()

	zigTry(t, ts.Resize(520))
	expectEqual(t, uint(8), uint(len(ts.dynamic_stops)))
	expectEqual(t, uint((64+8)*8), ts.Capacity())

	zigTry(t, ts.Resize(600))
	expectEqual(t, uint(88), uint(len(ts.dynamic_stops)))
	expectEqual(t, uint((64+88)*8), ts.Capacity())
	expectEqual(t, uint(600), ts.cols)
}

// Zig: test "term51: Tabstops.resize: growing preserves existing stops and zeroes new storage"
func TestTabstops_term51_Tabstops_resize_growing_preserves_existing_stops_and_zeroes_new_storage(t *testing.T) {
	ts := sm1_tabstopsInit(t, 600, 0)
	defer ts.Deinit()
	ts.Set(5)
	ts.Set(512)
	ts.Set(599)

	zigTry(t, ts.Resize(700))
	expectEqual(t, uint(700), ts.cols)
	expectEqual(t, uint(188), uint(len(ts.dynamic_stops)))
	expect(t, ts.Get(5), "t.get(5)")
	expect(t, ts.Get(512), "t.get(512)")
	expect(t, ts.Get(599), "t.get(599)")
	// New storage is zero.
	for _, u := range ts.dynamic_stops[88:] {
		expectEqual(t, tabstopsUnit(0), u)
	}
	expect(t, !ts.Get(600), "!t.get(600)")
	expect(t, !ts.Get(699), "!t.get(699)")
}

// Zig: test "term51: Tabstops.resize: growth does not add default tabstops"
func TestTabstops_term51_Tabstops_resize_growth_does_not_add_default_tabstops(t *testing.T) {
	// Upstream has a TODO about this: resize never applies an interval.
	ts := sm1_tabstopsInit(t, 80, 8)
	defer ts.Deinit()
	expect(t, ts.Get(72), "t.get(72)")
	expect(t, !ts.Get(80), "!t.get(80)")

	zigTry(t, ts.Resize(200))
	expectEqual(t, uint(200), ts.cols)
	expect(t, ts.Get(72), "t.get(72)")
	expect(t, !ts.Get(80), "!t.get(80)")
	expect(t, !ts.Get(88), "!t.get(88)")
	expect(t, !ts.Get(192), "!t.get(192)")
}

// Zig: test "term51: Tabstops.resize: same size as the current dynamic length reallocates and keeps data"
func TestTabstops_term51_Tabstops_resize_same_size_as_the_current_dynamic_length_reallocates_and_keeps_data(t *testing.T) {
	ts := sm1_tabstopsInit(t, 600, 0)
	defer ts.Deinit()
	ts.Set(550)
	expectEqual(t, uint(88), uint(len(ts.dynamic_stops)))

	// size (88) is not < dynamic len (88), so a fresh copy is made.
	zigTry(t, ts.Resize(600))
	expectEqual(t, uint(88), uint(len(ts.dynamic_stops)))
	expectEqual(t, uint(600), ts.cols)
	expect(t, ts.Get(550), "t.get(550)")
}

// Zig: test "term51: Tabstops.resize: smaller than the dynamic length keeps the storage"
func TestTabstops_term51_Tabstops_resize_smaller_than_the_dynamic_length_keeps_the_storage(t *testing.T) {
	ts := sm1_tabstopsInit(t, 600, 0)
	defer ts.Deinit()
	ts.Set(590)
	before := &ts.dynamic_stops[0] // Zig: t.dynamic_stops.ptr

	zigTry(t, ts.Resize(550))
	expectEqual(t, uint(550), ts.cols)
	expectEqual(t, uint(88), uint(len(ts.dynamic_stops)))
	expect(t, &ts.dynamic_stops[0] == before, "t.dynamic_stops.ptr == before")
	// The capacity does not shrink and the stop is retained.
	expectEqual(t, uint((64+88)*8), ts.Capacity())
	expect(t, ts.Get(590), "t.get(590)")

	// Shrinking to the preallocated range keeps the dynamic storage too.
	zigTry(t, ts.Resize(100))
	expectEqual(t, uint(100), ts.cols)
	expectEqual(t, uint(88), uint(len(ts.dynamic_stops)))
	expect(t, ts.Get(590), "t.get(590)")

	// Growing again within the existing storage does not allocate.
	// Zig-only: var failing = std.testing.FailingAllocator.init(testing.allocator, .{ .fail_index = 0 });
	zigTry(t, ts.Resize(580))
	expectEqual(t, uint(580), ts.cols)
	expect(t, ts.Get(590), "t.get(590)")
}

// Not translated, Zig test "term51: Tabstops.resize: allocation failure on growth preserves state and data": see scripts/not-translated.txt (FailingAllocator).

// Not translated, Zig test "term51: Tabstops.resize: failure when growing from the preallocated range": see scripts/not-translated.txt (FailingAllocator).

// Zig: test "term51: Tabstops.capacity: depends on the dynamic storage only"
func TestTabstops_term51_Tabstops_capacity_depends_on_the_dynamic_storage_only(t *testing.T) {
	ts := Tabstops{}
	defer ts.Deinit()
	expectEqual(t, uint(512), ts.Capacity())

	zigTry(t, ts.Resize(100))
	expectEqual(t, uint(512), ts.Capacity())
	zigTry(t, ts.Resize(512))
	expectEqual(t, uint(512), ts.Capacity())
	zigTry(t, ts.Resize(513))
	expectEqual(t, uint(520), ts.Capacity())
	zigTry(t, ts.Resize(1000))
	expectEqual(t, uint((64+488)*8), ts.Capacity())
}

// Zig: test "term51: Tabstops.reset: clears every stop including dynamic ones"
func TestTabstops_term51_Tabstops_reset_clears_every_stop_including_dynamic_ones(t *testing.T) {
	ts := sm1_tabstopsInit(t, 600, 0)
	defer ts.Deinit()
	ts.Set(0)
	ts.Set(100)
	ts.Set(511)
	ts.Set(512)
	ts.Set(599)

	ts.Reset(0)
	for col := uint(0); col < ts.Capacity(); col++ {
		expect(t, !ts.Get(col), "!t.get(col)")
	}
	expectEqual(t, uint(600), ts.cols)
}

// Zig: test "term51: Tabstops.reset: interval boundary, last column never gets a stop"
func TestTabstops_term51_Tabstops_reset_interval_boundary_last_column_never_gets_a_stop(t *testing.T) {
	// Stops land on multiples of the interval strictly below cols - 1.
	func() {
		ts := sm1_tabstopsInit(t, 9, 8)
		defer ts.Deinit()
		for c := uint(0); c < 16; c++ {
			expect(t, !ts.Get(c), "!t.get(c)")
		}
	}()
	func() {
		ts := sm1_tabstopsInit(t, 10, 8)
		defer ts.Deinit()
		expect(t, ts.Get(8), "t.get(8)")
		expect(t, !ts.Get(9), "!t.get(9)")
		expect(t, !ts.Get(16), "!t.get(16)")
	}()
	func() {
		// interval == cols - 1: i < cols - 1 fails immediately.
		ts := sm1_tabstopsInit(t, 10, 9)
		defer ts.Deinit()
		for c := uint(0); c < 16; c++ {
			expect(t, !ts.Get(c), "!t.get(c)")
		}
	}()
	func() {
		// interval larger than cols.
		ts := sm1_tabstopsInit(t, 10, 100)
		defer ts.Deinit()
		for c := uint(0); c < 16; c++ {
			expect(t, !ts.Get(c), "!t.get(c)")
		}
	}()
	func() {
		// cols of 1: cols - 1 == 0, no stops for any interval.
		ts := sm1_tabstopsInit(t, 1, 1)
		defer ts.Deinit()
		expect(t, !ts.Get(0), "!t.get(0)")
		expect(t, !ts.Get(1), "!t.get(1)")
	}()
	func() {
		// cols of 2: cols - 1 == 1, i starts at interval 1 and 1 < 1 is false.
		ts := sm1_tabstopsInit(t, 2, 1)
		defer ts.Deinit()
		expect(t, !ts.Get(0), "!t.get(0)")
		expect(t, !ts.Get(1), "!t.get(1)")
	}()
	func() {
		ts := sm1_tabstopsInit(t, 3, 1)
		defer ts.Deinit()
		expect(t, ts.Get(1), "t.get(1)")
		expect(t, !ts.Get(2), "!t.get(2)")
	}()
}

// Zig: test "term51: Tabstops.reset: replaces earlier stops and spans the dynamic boundary"
func TestTabstops_term51_Tabstops_reset_replaces_earlier_stops_and_spans_the_dynamic_boundary(t *testing.T) {
	ts := sm1_tabstopsInit(t, 600, 8)
	defer ts.Deinit()
	ts.Set(3)
	ts.Set(513)

	ts.Reset(8)
	expect(t, !ts.Get(3), "!t.get(3)")
	expect(t, !ts.Get(513), "!t.get(513)")
	// Stops across the boundary at 512 and up to the last multiple below 599.
	expect(t, ts.Get(504), "t.get(504)")
	expect(t, ts.Get(512), "t.get(512)")
	expect(t, ts.Get(520), "t.get(520)")
	expect(t, ts.Get(592), "t.get(592)")
	expect(t, !ts.Get(600), "!t.get(600)")

	// A different interval.
	ts.Reset(100)
	var count uint = 0
	for col := uint(0); col < ts.Capacity(); col++ {
		if ts.Get(col) {
			count += 1
			expect(t, col%100 == 0, "col % 100 == 0")
		}
	}
	expectEqual(t, uint(5), count) // 100..500
	expect(t, ts.Get(500), "t.get(500)")
	expect(t, !ts.Get(600), "!t.get(600)")
}

// Zig: test "term51: Tabstops.reset: uses the current cols after a resize"
func TestTabstops_term51_Tabstops_reset_uses_the_current_cols_after_a_resize(t *testing.T) {
	ts := sm1_tabstopsInit(t, 20, 8)
	defer ts.Deinit()
	expect(t, ts.Get(16), "t.get(16)")
	expect(t, !ts.Get(24), "!t.get(24)")

	zigTry(t, ts.Resize(40))
	ts.Reset(8)
	expect(t, ts.Get(8), "t.get(8)")
	expect(t, ts.Get(16), "t.get(16)")
	expect(t, ts.Get(24), "t.get(24)")
	expect(t, ts.Get(32), "t.get(32)")
	expect(t, !ts.Get(39), "!t.get(39)")
	expect(t, !ts.Get(40), "!t.get(40)")
}

// Zig: test "term51: Tabstops: width table around the dynamic storage boundary"
func TestTabstops_term51_Tabstops_width_table_around_the_dynamic_storage_boundary(t *testing.T) {
	// For every width 1..600 and a few intervals check the stored geometry
	// and the exact set of stops.
	intervals := [...]uint{0, 1, 3, 8}
	for w := uint(1); w < 601; w++ {
		for _, interval := range intervals {
			func() {
				ts := sm1_tabstopsInit(t, w, interval)
				defer ts.Deinit()

				expectEqual(t, w, ts.cols)
				var dyn uint = 0
				if w > tabstopsPreallocColumns {
					dyn = w - tabstopsPreallocColumns
				}
				expectEqual(t, dyn, uint(len(ts.dynamic_stops)))
				expectEqual(t, (64+dyn)*8, ts.Capacity())
				expect(t, ts.Capacity() >= w, "t.capacity() >= w")

				for col := uint(0); col < ts.Capacity(); col++ {
					want := interval > 0 &&
						col >= interval &&
						col%interval == 0 &&
						col < w-1
					if ts.Get(col) != want {
						t.Fatalf("w=%d interval=%d col=%d: TestExpectedEqual", w, interval, col)
					}
				}
			}()
		}
	}
}

// Zig: test "term51: Tabstops: boundary widths 511 512 513 and 514"
func TestTabstops_term51_Tabstops_boundary_widths_511_512_513_and_514(t *testing.T) {
	widths := [...]uint{511, 512, 513, 514}
	dyn_len := [...]uint{0, 0, 1, 2}
	caps := [...]uint{512, 512, 520, 528}
	for i := range widths {
		w, d, cap := widths[i], dyn_len[i], caps[i]
		func() {
			ts := sm1_tabstopsInit(t, w, 8)
			defer ts.Deinit()
			expectEqual(t, d, uint(len(ts.dynamic_stops)))
			expectEqual(t, cap, ts.Capacity())
			// Last addressable column works in each.
			ts.Set(w - 1)
			expect(t, ts.Get(w-1), "t.get(w - 1)")
			ts.Unset(w - 1)
			expect(t, !ts.Get(w-1), "!t.get(w - 1)")
		}()
	}

	// Default stops need col < width - 1: 512 is a stop only from width 514.
	ts514 := sm1_tabstopsInit(t, 514, 8)
	defer ts514.Deinit()
	expect(t, ts514.Get(504), "t514.get(504)")
	expect(t, ts514.Get(512), "t514.get(512)")

	ts513 := sm1_tabstopsInit(t, 513, 8)
	defer ts513.Deinit()
	expect(t, ts513.Get(504), "t513.get(504)")
	expect(t, !ts513.Get(512), "!t513.get(512)")

	ts512 := sm1_tabstopsInit(t, 512, 8)
	defer ts512.Deinit()
	expect(t, ts512.Get(504), "t512.get(504)")
	expect(t, !ts512.Get(511), "!t512.get(511)")

	ts505 := sm1_tabstopsInit(t, 505, 8)
	defer ts505.Deinit()
	expect(t, ts505.Get(496), "t505.get(496)")
	expect(t, !ts505.Get(504), "!t505.get(504)")
	ts506 := sm1_tabstopsInit(t, 506, 8)
	defer ts506.Deinit()
	expect(t, ts506.Get(504), "t506.get(504)")
}

// tsExpectPanic fails the test unless f panics.
func tsExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// Go-only: panic test: Tabstops.reset: cols is 0 and interval above 0 (Zig: `self.cols - 1` underflows usize and traps)
func TestTabstops_panic_reset_cols_zero(t *testing.T) {
	tsExpectPanic(t, func() { _, _ = TabstopsInit(0, 8) })
}
