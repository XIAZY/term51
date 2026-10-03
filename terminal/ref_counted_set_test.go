// Tests translated one for one from src/terminal/ref_counted_set.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The file has no upstream
// tests; all of them are the project's `term51:` tests.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual keeps Zig's (expected, actual) argument order. `try
//     testing.expectError(error.X, call)` is sm2_expectError(t, ErrX,
//     <the error of the call>).
//   - Zig's test helpers are the sm2_ types: sm2_Val (term51GVal), sm2_Spec
//     (term51GSpec), sm2_Rec (term51GRec), sm2_Ctx / sm2_CtxDel (term51GCtx /
//     term51GCtxDel) and the generic fixture sm2_Fix (term51GFix). The Zig
//     sets term51GSet, term51GSetDel, term51GSet8 and term51GSetMix are the
//     instantiations sm2_Set, sm2_SetDel, sm2_Set8 and sm2_SetMix; the style
//     set is StyleSet.
//   - Zig's struct field defaults (`.{}` for a context or spec: div 1, mul 1,
//     add 0) are not Go zero values, so contexts are built with sm2_ctx /
//     sm2_ctxDel and specs with sm2_spec; sm2_specDefault is `.{}`.
//   - The `base` argument of every set function is gone (Go memory).
//     `try f.set.add(f.buf, v)` is f.add(v), which stops the test on an
//     error like `try`; f.addErr(v) is the call whose error a test inspects.
//   - Zig-only: `const alloc = testing.allocator;`, `defer f.deinit(alloc)`,
//     `defer snap.deinit(alloc)` and the allocation inside Fix.init are
//     allocator bookkeeping with no Go counterpart; they are dropped in
//     every test without a per-test note.
//   - `*const S` receivers: Go has no const pointers, so a test that calls
//     the same function through `const cs: *const S = &f.set` calls it
//     through the same pointer.
//   - A zero-capacity set is empty in Go: lookups find nothing and adds
//     return ErrOutOfMemory (see the two tests that need it).

package terminal

import (
	"errors"
	"math"
	"math/bits"
	"reflect"
	"testing"
)

// sm2_expectError is Zig's `try testing.expectError(expected, actual)`.
func sm2_expectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// sm2_ceilPowerOfTwo is std.math.ceilPowerOfTwoAssert(usize, n).
func sm2_ceilPowerOfTwo(n uint) uint {
	assert(n > 0)
	return uint(1) << bits.Len(n-1)
}

// sm2_Val is term51GVal: test item. Only `key` takes part in hashing and
// equality, `tag` lets a test tell apart two values that the set considers
// equal.
type sm2_Val struct {
	key uint32
	tag uint32
}

// sm2_K is term51GK.
func sm2_K(key uint32) sm2_Val {
	return sm2_Val{key: key}
}

// sm2_KT is term51GKT.
func sm2_KT(key uint32, tag uint32) sm2_Val {
	return sm2_Val{key: key, tag: tag}
}

// sm2_Spec is term51GSpec: controllable hash: hash(key) = (key / div) *% mul
// +% add.
//   - identity (default):      div 1, mul 1, add 0
//   - constant c (collisions): mul 0, add c
//   - groups of `div` keys that share one bucket start: div = group size
type sm2_Spec struct {
	div uint32
	mul uint64
	add uint64
}

// sm2_specDefault is `.{}`: div 1, mul 1, add 0.
var sm2_specDefault = sm2_Spec{div: 1, mul: 1, add: 0}

// sm2_spec builds a spec with every field given (Zig names only the fields
// that differ from the defaults).
func sm2_spec(div uint32, mul uint64, add uint64) sm2_Spec {
	return sm2_Spec{div: div, mul: mul, add: add}
}

func (self sm2_Spec) apply(key uint32) uint64 {
	return uint64(key/self.div)*self.mul + self.add
}

// sm2_Rec is term51GRec: records what the context was asked to do.
type sm2_Rec struct {
	eqls   uint
	last_a sm2_Val
	last_b sm2_Val
	dels   [128]sm2_Val
	del_n  uint
}

func (self *sm2_Rec) expectDeleted(t *testing.T, want ...sm2_Val) {
	t.Helper()
	expectEqualSlices(t, want, self.dels[0:self.del_n])
}

// sm2_Ctx is term51GCtx: context without a `deleted` hook.
type sm2_Ctx struct {
	spec sm2_Spec
	rec  *sm2_Rec
}

func sm2_ctx(spec sm2_Spec, rec *sm2_Rec) sm2_Ctx {
	return sm2_Ctx{spec: spec, rec: rec}
}

func (self sm2_Ctx) Hash(v sm2_Val) uint64 {
	return self.spec.apply(v.key)
}

func (self sm2_Ctx) Eql(a sm2_Val, b sm2_Val) bool {
	if self.rec != nil {
		self.rec.eqls += 1
		self.rec.last_a = a
		self.rec.last_b = b
	}
	return a.key == b.key
}

// sm2_CtxDel is term51GCtxDel: context with a `deleted` hook.
type sm2_CtxDel struct {
	spec sm2_Spec
	rec  *sm2_Rec
}

func sm2_ctxDel(spec sm2_Spec, rec *sm2_Rec) sm2_CtxDel {
	return sm2_CtxDel{spec: spec, rec: rec}
}

func (self sm2_CtxDel) Hash(v sm2_Val) uint64 {
	return self.spec.apply(v.key)
}

func (self sm2_CtxDel) Eql(a sm2_Val, b sm2_Val) bool {
	if self.rec != nil {
		self.rec.eqls += 1
		self.rec.last_a = a
		self.rec.last_b = b
	}
	return a.key == b.key
}

func (self sm2_CtxDel) Deleted(v sm2_Val) {
	if self.rec != nil {
		self.rec.dels[self.rec.del_n] = v
		self.rec.del_n += 1
	}
}

var _ RefCountedSetContextDeleted[sm2_Val] = sm2_CtxDel{}

type sm2_Set = RefCountedSet[sm2_Val, uint16, uint16, sm2_Ctx]
type sm2_SetDel = RefCountedSet[sm2_Val, uint16, uint16, sm2_CtxDel]
type sm2_Set8 = RefCountedSet[sm2_Val, uint8, uint8, sm2_Ctx]
type sm2_SetMix = RefCountedSet[sm2_Val, uint16, uint32, sm2_Ctx]

// sm2_Fix is term51GFix(S): an initialized set. Zig's fixture also owns the
// buffer behind the set; Go memory needs none.
type sm2_Fix[T comparable, Id RefCountedSetInt, Ref RefCountedSetInt, C RefCountedSetContext[T]] struct {
	t   *testing.T
	set RefCountedSet[T, Id, Ref, C]
}

type sm2_FixSet = sm2_Fix[sm2_Val, uint16, uint16, sm2_Ctx]
type sm2_FixSetDel = sm2_Fix[sm2_Val, uint16, uint16, sm2_CtxDel]
type sm2_FixSet8 = sm2_Fix[sm2_Val, uint8, uint8, sm2_Ctx]
type sm2_FixSetMix = sm2_Fix[sm2_Val, uint16, uint32, sm2_Ctx]
type sm2_FixStyleSet = sm2_Fix[Style, StyleId, SizeCellCountInt, StyleSetContext]

// sm2_fixInit is Fix.init: Layout.init(cap) and S.init.
func sm2_fixInit[T comparable, Id RefCountedSetInt, Ref RefCountedSetInt, C RefCountedSetContext[T]](t *testing.T, cap uint, ctx C) *sm2_Fix[T, Id, Ref, C] {
	layout := RefCountedSetLayoutInit[Id](cap)
	return &sm2_Fix[T, Id, Ref, C]{t: t, set: RefCountedSetInit[T, Id, Ref, C](layout, ctx)}
}

func sm2_newSet(t *testing.T, cap uint, ctx sm2_Ctx) *sm2_FixSet {
	return sm2_fixInit[sm2_Val, uint16, uint16, sm2_Ctx](t, cap, ctx)
}

func sm2_newSetDel(t *testing.T, cap uint, ctx sm2_CtxDel) *sm2_FixSetDel {
	return sm2_fixInit[sm2_Val, uint16, uint16, sm2_CtxDel](t, cap, ctx)
}

func sm2_newSet8(t *testing.T, cap uint, ctx sm2_Ctx) *sm2_FixSet8 {
	return sm2_fixInit[sm2_Val, uint8, uint8, sm2_Ctx](t, cap, ctx)
}

func sm2_newSetMix(t *testing.T, cap uint, ctx sm2_Ctx) *sm2_FixSetMix {
	return sm2_fixInit[sm2_Val, uint16, uint32, sm2_Ctx](t, cap, ctx)
}

func sm2_newStyleSet(t *testing.T, cap uint) *sm2_FixStyleSet {
	return sm2_fixInit[Style, StyleId, SizeCellCountInt, StyleSetContext](t, cap, StyleSetContext{})
}

// add is `try f.set.add(f.buf, v)`.
func (f *sm2_Fix[T, Id, Ref, C]) add(v T) Id {
	f.t.Helper()
	id, err := f.set.Add(v)
	zigTry(f.t, err)
	return id
}

// addErr is `f.set.add(f.buf, v)` where the test looks at the error.
func (f *sm2_Fix[T, Id, Ref, C]) addErr(v T) error {
	_, err := f.set.Add(v)
	return err
}

// addContext is `try f.set.addContext(f.buf, v, ctx)`.
func (f *sm2_Fix[T, Id, Ref, C]) addContext(v T, ctx C) Id {
	f.t.Helper()
	id, err := f.set.AddContext(v, ctx)
	zigTry(f.t, err)
	return id
}

// addWithId is `try f.set.addWithId(f.buf, v, id)`.
func (f *sm2_Fix[T, Id, Ref, C]) addWithId(v T, id Id) Optional[Id] {
	f.t.Helper()
	got, err := f.set.AddWithId(v, id)
	zigTry(f.t, err)
	return got
}

// addWithIdErr is `f.set.addWithId(f.buf, v, id)` where the test looks at
// the error.
func (f *sm2_Fix[T, Id, Ref, C]) addWithIdErr(v T, id Id) error {
	_, err := f.set.AddWithId(v, id)
	return err
}

// addWithIdContext is `try f.set.addWithIdContext(f.buf, v, id, ctx)`.
func (f *sm2_Fix[T, Id, Ref, C]) addWithIdContext(v T, id Id, ctx C) Optional[Id] {
	f.t.Helper()
	got, err := f.set.AddWithIdContext(v, id, ctx)
	zigTry(f.t, err)
	return got
}

func (f *sm2_Fix[T, Id, Ref, C]) tbl() []Id {
	return f.set.table[0:f.set.layout.table_cap]
}

func (f *sm2_Fix[T, Id, Ref, C]) itm(id uint) *RefCountedSetItem[T, Id, Ref] {
	return &f.set.items[id]
}

// expectItem asserts bucket, psl and ref of an item and that its bucket
// holds the id.
func (f *sm2_Fix[T, Id, Ref, C]) expectItem(id Id, bucket Id, psl Id, ref Ref) {
	f.t.Helper()
	it := f.itm(uint(id))
	expectEqual(f.t, uint(bucket), uint(it.meta.bucket))
	expectEqual(f.t, uint(psl), uint(it.meta.psl))
	expectEqual(f.t, uint(ref), uint(it.meta.ref))
	expectEqual(f.t, uint(id), uint(f.tbl()[bucket]))
}

// expectHole asserts an item slot is in the default (unused) state.
func (f *sm2_Fix[T, Id, Ref, C]) expectHole(id Id) {
	f.t.Helper()
	it := f.itm(uint(id))
	var maxId Id
	maxId--
	expectEqual(f.t, uint(maxId), uint(it.meta.bucket))
	expectEqual(f.t, uint(0), uint(it.meta.psl))
	expectEqual(f.t, uint(0), uint(it.meta.ref))
}

// expectStats asserts the psl_stats prefix; everything after it must be 0.
func (f *sm2_Fix[T, Id, Ref, C]) expectStats(want ...uint) {
	f.t.Helper()
	for i, got := range f.set.psl_stats {
		var w uint
		if i < len(want) {
			w = want[i]
		}
		expectEqual(f.t, w, uint(got))
	}
}

// expectTable asserts the whole table content (bucket -> id, 0 = empty).
func (f *sm2_Fix[T, Id, Ref, C]) expectTable(want ...uint) {
	f.t.Helper()
	for i, got := range f.tbl() {
		var w uint
		if i < len(want) {
			w = want[i]
		}
		expectEqual(f.t, w, uint(got))
	}
}

// sm2_Snap is Fix.Snap: a copy of the table, the items and the set's own
// fields. Zig copies the bytes of the whole buffer.
type sm2_Snap[T comparable, Id RefCountedSetInt, Ref RefCountedSetInt, C RefCountedSetContext[T]] struct {
	table []Id
	items []RefCountedSetItem[T, Id, Ref]
	set   RefCountedSet[T, Id, Ref, C]
}

func (f *sm2_Fix[T, Id, Ref, C]) snapshot() sm2_Snap[T, Id, Ref, C] {
	snap := sm2_Snap[T, Id, Ref, C]{
		table: append([]Id(nil), f.set.table...),
		items: append([]RefCountedSetItem[T, Id, Ref](nil), f.set.items...),
		set:   f.set,
	}
	snap.set.table = nil
	snap.set.items = nil
	return snap
}

func (self sm2_Snap[T, Id, Ref, C]) expectSame(f *sm2_Fix[T, Id, Ref, C]) {
	f.t.Helper()
	expectEqualSlices(f.t, self.table, f.set.table)
	expectEqualSlices(f.t, self.items, f.set.items)
	expectEqual(f.t, self.set.max_psl, f.set.max_psl)
	expectEqual(f.t, self.set.psl_stats, f.set.psl_stats)
	expectEqual(f.t, self.set.living, f.set.living)
	expectEqual(f.t, self.set.next_id, f.set.next_id)
	expectEqual(f.t, self.set.layout, f.set.layout)
	expectEqual(f.t, any(self.set.context), any(f.set.context))
}

// sm2_palette and sm2_rgb build `.{ .palette = p }` and
// `.{ .rgb = .{ .r = r, .g = g, .b = b } }` of Style.Color.
func sm2_palette(p uint8) StyleColor {
	return StyleColor{Tag: StyleColorPalette, Palette: p}
}

func sm2_rgb(r uint8, g uint8, b uint8) StyleColor {
	return StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{r: r, g: g, b: b}}
}

// Zig: test "term51: RefCountedSet: constants, re-exported types, item defaults"
func TestRefCountedSet_term51_RefCountedSet_constants_re_exported_types_item_defaults(t *testing.T) {
	expect(t, RefCountedSetLoadFactor == 0.8125, "S.load_factor == 0.8125")
	// Zig-only: `S.Id == u16` and `S.Context == term51GCtx`: the re-exports
	// are the type parameters themselves.
	// Zig-only: `S.base_align.toByteUnits() == 8`: base_align is omitted
	// under the Go-memory rule.

	it := RefCountedSetItemInit[sm2_Val, uint16, uint16]()
	expectEqual(t, uint16(math.MaxUint16), it.meta.bucket)
	expectEqual(t, uint16(0), it.meta.psl)
	expectEqual(t, uint16(0), it.meta.ref)

	// The metadata field types follow the Id and ref count parameters.
	m := RefCountedSetItemInit[sm2_Val, uint16, uint32]()
	expect(t, reflect.TypeOf(m.meta.bucket) == reflect.TypeOf(uint16(0)), "@TypeOf(@as(M.Item, .{}).meta.bucket) == u16")
	expect(t, reflect.TypeOf(m.meta.psl) == reflect.TypeOf(uint16(0)), "@TypeOf(@as(M.Item, .{}).meta.psl) == u16")
	expect(t, reflect.TypeOf(m.meta.ref) == reflect.TypeOf(uint32(0)), "@TypeOf(@as(M.Item, .{}).meta.ref) == u32")
	s8 := RefCountedSetItemInit[sm2_Val, uint8, uint8]()
	expect(t, reflect.TypeOf(s8.meta.bucket) == reflect.TypeOf(uint8(0)), "@TypeOf(@as(S8.Item, .{}).meta.bucket) == u8")
	expectEqual(t, uint8(255), s8.meta.bucket)

	// The style set used by Page is a 16-bit id / 16-bit count set.
	ss := RefCountedSetItemInit[Style, StyleId, SizeCellCountInt]()
	expect(t, reflect.TypeOf(StyleId(0)) == reflect.TypeOf(uint16(0)), "SS.Id == u16")
	expect(t, reflect.TypeOf(ss.meta.ref) == reflect.TypeOf(uint16(0)), "@TypeOf(@as(SS.Item, .{}).meta.ref) == u16")
	expect(t, RefCountedSetLoadFactor == 0.8125, "SS.load_factor == 0.8125")
	// Zig-only: `SS.base_align.toByteUnits() >= 8`: base_align is omitted.
}

// Zig: test "term51: RefCountedSet.capacityForCount: zero and small counts"
func TestRefCountedSet_term51_RefCountedSet_capacityForCount_zero_and_small_counts(t *testing.T) {
	// ceil((n + 1) / 0.8125); n = 0 is special-cased to 0.
	expectEqual(t, uint(0), RefCountedSetCapacityForCount(0))
	expectEqual(t, uint(3), RefCountedSetCapacityForCount(1))
	expectEqual(t, uint(4), RefCountedSetCapacityForCount(2))
	expectEqual(t, uint(5), RefCountedSetCapacityForCount(3))
	expectEqual(t, uint(7), RefCountedSetCapacityForCount(4))
	expectEqual(t, uint(8), RefCountedSetCapacityForCount(5))
	expectEqual(t, uint(9), RefCountedSetCapacityForCount(6))
}

// Zig: test "term51: RefCountedSet.capacityForCount: exact and just-over multiples of 13/16"
func TestRefCountedSet_term51_RefCountedSet_capacityForCount_exact_and_just_over_multiples_of_13_16(t *testing.T) {
	// (n + 1) / 0.8125 is an exact integer when n + 1 is a multiple of 13.
	expectEqual(t, uint(16), RefCountedSetCapacityForCount(12))
	expectEqual(t, uint(18), RefCountedSetCapacityForCount(13))
	expectEqual(t, uint(32), RefCountedSetCapacityForCount(25))
	expectEqual(t, uint(34), RefCountedSetCapacityForCount(26))
	expectEqual(t, uint(64), RefCountedSetCapacityForCount(51))
	expectEqual(t, uint(66), RefCountedSetCapacityForCount(52))
	expectEqual(t, uint(125), RefCountedSetCapacityForCount(100))
	expectEqual(t, uint(128), RefCountedSetCapacityForCount(103))
}

// Zig: test "term51: RefCountedSet.capacityForCount: largest count for 16-bit ids"
func TestRefCountedSet_term51_RefCountedSet_capacityForCount_largest_count_for_16_bit_ids(t *testing.T) {
	// 65536 is the largest cap Layout.init accepts for a u16 Id.
	expectEqual(t, uint(65536), RefCountedSetCapacityForCount(53247))
	expectEqual(t, uint(65538), RefCountedSetCapacityForCount(53248))
	expectEqual(t, uint(53248), RefCountedSetLayoutInit[uint16](RefCountedSetCapacityForCount(53247)).cap)
}

// Zig: test "term51: RefCountedSet.capacityForCount: layout capacity always holds n items plus reserved id 0"
func TestRefCountedSet_term51_RefCountedSet_capacityForCount_layout_capacity_always_holds_n_items_plus_reserved_id_0(t *testing.T) {
	for n := uint(1); n <= 2000; n++ {
		c := RefCountedSetCapacityForCount(n)
		expect(t, c >= n+1, "c >= n + 1")
		// Layout.cap counts ids including the reserved 0, so n items need n + 1.
		expect(t, RefCountedSetLayoutInit[uint16](c).cap >= n+1, "S.Layout.init(c).cap >= n + 1")
	}
}

// Zig: test "term51: RefCountedSet.capacityForCount: a set sized for n holds n items and exact fits hold no more"
func TestRefCountedSet_term51_RefCountedSet_capacityForCount_a_set_sized_for_n_holds_n_items_and_exact_fits_hold_no_more(t *testing.T) {
	// n + 1 is a multiple of 13 here, so the capacity is exactly full.
	for _, n := range []uint32{12, 25, 51, 103} {
		f := sm2_newSet(t, RefCountedSetCapacityForCount(uint(n)), sm2_ctx(sm2_specDefault, nil))
		expectEqual(t, uint(n)+1, f.set.layout.cap)
		for k := uint32(0); k < n; k++ {
			expectEqual(t, uint16(k+1), f.add(sm2_K(k)))
		}
		expectEqual(t, uint(n), f.set.Count())
		sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(n)))
		expectEqual(t, uint(n), f.set.Count())
	}

	// Other counts have at least n slots (rounding up to a power of two).
	for _, n := range []uint32{1, 2, 3, 4, 5, 13, 26, 52, 100} {
		f := sm2_newSet(t, RefCountedSetCapacityForCount(uint(n)), sm2_ctx(sm2_specDefault, nil))
		for k := uint32(0); k < n; k++ {
			f.add(sm2_K(k))
		}
		expectEqual(t, uint(n), f.set.Count())
	}
}

// Zig: test "term51: RefCountedSet.Layout.init: layout zero capacity"
func TestRefCountedSet_term51_RefCountedSet_Layout_init_layout_zero_capacity(t *testing.T) {
	l := RefCountedSetLayoutInit[uint16](0)
	expectEqual(t, uint(0), l.cap)
	expectEqual(t, uint(0), l.table_cap)
	expectEqual(t, uint16(0), l.table_mask)
	// Zig-only: l.table_start, l.items_start and l.total_size (byte
	// offsets and size) are not in the Go layout.
}

// Zig: test "term51: RefCountedSet.Layout.init: layout offsets and total size for the u16 set"
func TestRefCountedSet_term51_RefCountedSet_Layout_init_layout_offsets_and_total_size_for_the_u16_set(t *testing.T) {
	// Item size 16, alignment 4, Id size 2.
	// Zig-only: each Zig row also has items_start and total (byte offsets
	// and sizes: 4/4, 4/20, 8/56, 8/56, 16/112, 16/112, 32/240, 32/240,
	// 64/480, 256/1920, 2048/15360, 32768/245760, 131072/983040 twice).
	type row struct {
		cap       uint
		table_cap uint
		items_cap uint
		mask      uint16
	}
	rows := []row{
		// table is a single 2-byte entry, padded to 4 for the items.
		{cap: 1, table_cap: 1, items_cap: 0, mask: 0},
		{cap: 2, table_cap: 2, items_cap: 1, mask: 1},
		{cap: 3, table_cap: 4, items_cap: 3, mask: 3},
		{cap: 4, table_cap: 4, items_cap: 3, mask: 3},
		{cap: 5, table_cap: 8, items_cap: 6, mask: 7},
		{cap: 8, table_cap: 8, items_cap: 6, mask: 7},
		{cap: 9, table_cap: 16, items_cap: 13, mask: 15},
		{cap: 16, table_cap: 16, items_cap: 13, mask: 15},
		{cap: 17, table_cap: 32, items_cap: 26, mask: 31},
		{cap: 100, table_cap: 128, items_cap: 104, mask: 127},
		{cap: 1000, table_cap: 1024, items_cap: 832, mask: 1023},
		{cap: 16384, table_cap: 16384, items_cap: 13312, mask: 16383},
		{cap: 65535, table_cap: 65536, items_cap: 53248, mask: 65535},
		{cap: 65536, table_cap: 65536, items_cap: 53248, mask: 65535},
	}
	for _, r := range rows {
		l := RefCountedSetLayoutInit[uint16](r.cap)
		expectEqual(t, r.items_cap, l.cap)
		expectEqual(t, r.table_cap, l.table_cap)
		expectEqual(t, r.mask, l.table_mask)
		// Zig-only: l.table_start == 0, l.items_start == r.items_start and
		// l.total_size == r.total.
	}
}

// Zig: test "term51: RefCountedSet.Layout.init: layout offsets and total size for the u8 set"
func TestRefCountedSet_term51_RefCountedSet_Layout_init_layout_offsets_and_total_size_for_the_u8_set(t *testing.T) {
	// Item size 12, alignment 4, Id size 1.
	// Zig-only: each Zig row also has items_start and total (byte offsets
	// and sizes: 4/4, 4/40, 16/172, 256/2752 twice).
	type row struct {
		cap       uint
		table_cap uint
		items_cap uint
		mask      uint8
	}
	rows := []row{
		{cap: 1, table_cap: 1, items_cap: 0, mask: 0},
		{cap: 3, table_cap: 4, items_cap: 3, mask: 3},
		{cap: 16, table_cap: 16, items_cap: 13, mask: 15},
		{cap: 255, table_cap: 256, items_cap: 208, mask: 255},
		{cap: 256, table_cap: 256, items_cap: 208, mask: 255},
	}
	for _, r := range rows {
		l := RefCountedSetLayoutInit[uint8](r.cap)
		expectEqual(t, r.items_cap, l.cap)
		expectEqual(t, r.table_cap, l.table_cap)
		expectEqual(t, r.mask, l.table_mask)
		// Zig-only: l.table_start == 0, l.items_start == r.items_start and
		// l.total_size == r.total.
	}
}

// Zig: test "term51: RefCountedSet.Layout.init: layout style set offsets follow item size and alignment"
func TestRefCountedSet_term51_RefCountedSet_Layout_init_layout_style_set_offsets_follow_item_size_and_alignment(t *testing.T) {
	// Zig-only: isz = @sizeOf(SS.Item) and ial = @alignOf(SS.Item).
	for _, cap := range []uint{1, 2, 3, 16, 17, 1000, 16384} {
		l := RefCountedSetLayoutInit[StyleId](cap)
		tc := sm2_ceilPowerOfTwo(cap)
		expectEqual(t, tc, l.table_cap)
		expectEqual(t, tc-1, uint(l.table_mask))
		// Zig-only: l.table_start == 0, l.items_start ==
		// alignForward(tc * 2, ial) and l.total_size == l.items_start +
		// l.cap * isz.
	}
}

// Zig: test "term51: RefCountedSet.Layout.init: capacity is a power of two table and 13/16 floor of it"
func TestRefCountedSet_term51_RefCountedSet_Layout_init_capacity_is_a_power_of_two_table_and_13_16_floor_of_it(t *testing.T) {
	// Each entry is Layout.init(cap) of one of the sets term51GSet,
	// term51GSet8 and the style set, as (table_cap, cap).
	for _, layout := range []func(cap uint) (uint, uint){
		func(cap uint) (uint, uint) {
			l := RefCountedSetLayoutInit[uint16](cap)
			return l.table_cap, l.cap
		},
		func(cap uint) (uint, uint) {
			l := RefCountedSetLayoutInit[uint8](cap)
			return l.table_cap, l.cap
		},
		func(cap uint) (uint, uint) {
			l := RefCountedSetLayoutInit[StyleId](cap)
			return l.table_cap, l.cap
		},
	} {
		// Item capacity (which includes the reserved id 0) is
		// floor(0.8125 * table_cap) with table_cap = next power of two.
		rows := [][3]uint{
			{0, 0, 0},
			{1, 1, 0},
			{2, 2, 1},
			{3, 4, 3},
			{4, 4, 3},
			{5, 8, 6},
			{7, 8, 6},
			{8, 8, 6},
			{9, 16, 13},
			{16, 16, 13},
			{17, 32, 26},
			{32, 32, 26},
			{33, 64, 52},
			{64, 64, 52},
			{65, 128, 104},
			{128, 128, 104},
			{129, 256, 208},
		}
		for _, r := range rows {
			table_cap, cap := layout(r[0])
			expectEqual(t, r[1], table_cap)
			expectEqual(t, r[2], cap)
		}
	}
	// 16-bit ids: the largest accepted capacity is 65536 (one past maxInt).
	big := RefCountedSetLayoutInit[uint16](65536)
	expectEqual(t, uint(65536), big.table_cap)
	expectEqual(t, uint(53248), big.cap)
	expectEqual(t, uint16(65535), big.table_mask)
	// 8-bit ids: 256.
	big8 := RefCountedSetLayoutInit[uint8](256)
	expectEqual(t, uint(256), big8.table_cap)
	expectEqual(t, uint(208), big8.cap)
	expectEqual(t, uint8(255), big8.table_mask)
}

// Zig: test "term51: RefCountedSet.init: fresh state with table zeroed and items defaulted"
func TestRefCountedSet_term51_RefCountedSet_init_fresh_state_with_table_zeroed_and_items_defaulted(t *testing.T) {
	layout := RefCountedSetLayoutInit[uint16](16)
	// Zig-only: buf (aligned allocation of layout.total_size + 32 bytes)
	// filled with 0xAA.

	set := RefCountedSetInit[sm2_Val, uint16, uint16, sm2_Ctx](layout, sm2_ctx(sm2_specDefault, nil))

	expectEqual(t, uint16(0), set.max_psl)
	expectEqual(t, uint(0), set.living)
	expectEqual(t, uint16(1), set.next_id)
	for _, s := range set.psl_stats {
		expectEqual(t, uint16(0), s)
	}
	expectEqual(t, layout, set.layout)
	expectEqual(t, uint(0), set.Count())
	// Zig-only: set.table.offset == 0 and set.items.offset ==
	// layout.items_start (byte offsets).

	// Go counterpart of slicing the buffer to table_cap / cap entries: the
	// set owns slices of exactly those lengths.
	expectEqual(t, int(layout.table_cap), len(set.table))
	expectEqual(t, int(layout.cap), len(set.items))
	for _, id := range set.table {
		expectEqual(t, uint16(0), id)
	}
	for _, it := range set.items {
		expectEqual(t, uint16(math.MaxUint16), it.meta.bucket)
		expectEqual(t, uint16(0), it.meta.psl)
		expectEqual(t, uint16(0), it.meta.ref)
	}
	// Zig-only: nothing past the layout is written (bytes after
	// layout.total_size are still 0xAA).
}

// Zig: test "term51: RefCountedSet.init: layout offsets are relative to the true base"
func TestRefCountedSet_term51_RefCountedSet_init_layout_offsets_are_relative_to_the_true_base(t *testing.T) {
	layout := RefCountedSetLayoutInit[uint16](16)
	// Zig-only: buf (64 + layout.total_size + 16 bytes of 0xAA) and the
	// base `.initOffset(buf, 64)`: Go sets own their memory and have no base.

	set := RefCountedSetInit[sm2_Val, uint16, uint16, sm2_Ctx](layout, sm2_ctx(sm2_specDefault, nil))
	// Zig-only: set.table.offset == 64, set.items.offset == 64 +
	// layout.items_start, the 0xAA bytes before and after the owned region
	// and the 32 zero bytes of the table at the offset.

	// And the set works.
	id, err := set.Add(sm2_K(5))
	zigTry(t, err)
	expectEqual(t, uint16(1), id)
	expectEqual(t, uint16(1), zigUnwrap(t, set.Lookup(sm2_K(5))))
	expectEqual(t, uint16(1), set.table[5])
	// Zig-only: the bytes before the region are untouched and table entry 5
	// is at byte 64 + 10 of the buffer (buf[64 + 10] == 1).
}

// Zig: test "term51: RefCountedSet.init: zero capacity set is empty and rejects adds"
func TestRefCountedSet_term51_RefCountedSet_init_zero_capacity_set_is_empty_and_rejects_adds(t *testing.T) {
	f := sm2_newSet(t, 0, sm2_ctx(sm2_specDefault, nil))

	expectEqual(t, uint(0), f.set.layout.cap)
	expectEqual(t, uint(0), f.set.layout.table_cap)
	expectEqual(t, uint(0), f.set.Count())
	expectEqual(t, uint16(1), f.set.next_id)
	// Zig-only: Zig's lookup reads table[0], which is outside the empty
	// region; the fixture keeps 16 spare zero bytes so that it reads "empty
	// bucket". In Go a zero-capacity set is empty: lookups find nothing and
	// adds return ErrOutOfMemory.
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(1)))
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	expectEqual(t, uint(0), f.set.Count())
	expectEqual(t, uint16(1), f.set.next_id)
}

// Zig: test "term51: RefCountedSet.init: capacities 1 and 2 hold no items, 3 holds two"
func TestRefCountedSet_term51_RefCountedSet_init_capacities_1_and_2_hold_no_items_3_holds_two(t *testing.T) {
	// cap 1 -> items cap 0, cap 2 -> items cap 1 (only the reserved id 0).
	for _, cap := range []uint{1, 2} {
		f := sm2_newSet(t, cap, sm2_ctx(sm2_specDefault, nil))
		sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(0)))
		sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(1)))
		expectEqual(t, uint(0), f.set.Count())
		expectEqual(t, uint16(1), f.set.next_id)
	}

	// cap 3 -> table 4, items cap 3, so two items fit.
	f := sm2_newSet(t, 3, sm2_ctx(sm2_specDefault, nil))
	expectEqual(t, uint(4), f.set.layout.table_cap)
	expectEqual(t, uint16(1), f.add(sm2_K(10)))
	expectEqual(t, uint16(2), f.add(sm2_K(11)))
	sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(12)))
	expectEqual(t, uint(2), f.set.Count())
	expectEqual(t, Some(uint16(1)), f.set.Lookup(sm2_K(10)))
	expectEqual(t, Some(uint16(2)), f.set.Lookup(sm2_K(11)))
}

// Zig: test "term51: RefCountedSet.add: ids start at 1 and increase, duplicates return the same id"
func TestRefCountedSet_term51_RefCountedSet_add_ids_start_at_1_and_increase_duplicates_return_the_same_id(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	expectEqual(t, uint16(1), f.add(sm2_K(10)))
	expectEqual(t, uint16(2), f.add(sm2_K(11)))
	expectEqual(t, uint16(3), f.add(sm2_K(12)))
	expectEqual(t, uint(3), f.set.Count())
	expectEqual(t, uint16(4), f.set.next_id)
	// Identity hash: bucket == key & 15, no collisions.
	f.expectItem(1, 10, 0, 1)
	f.expectItem(2, 11, 0, 1)
	f.expectItem(3, 12, 0, 1)
	f.expectStats(3)
	expectEqual(t, uint16(0), f.set.max_psl)
	expectEqual(t, uint(0), rec.del_n)

	// Re-adding bumps the ref count only; the id and set size don't change.
	expectEqual(t, uint16(2), f.add(sm2_K(11)))
	expectEqual(t, uint16(2), f.set.RefCount(2))
	expectEqual(t, uint(3), f.set.Count())
	expectEqual(t, uint16(4), f.set.next_id)
	f.expectStats(3)
}

// Zig: test "term51: RefCountedSet.add: a duplicate keeps the resident value and reports the new one as deleted"
func TestRefCountedSet_term51_RefCountedSet_add_a_duplicate_keeps_the_resident_value_and_reports_the_new_one_as_deleted(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	id := f.add(sm2_KT(7, 100))
	rec.eqls = 0
	expectEqual(t, id, f.add(sm2_KT(7, 200)))
	// The context is told the incoming copy is deleted...
	rec.expectDeleted(t, sm2_KT(7, 200))
	// ...the stored value is still the first one...
	expectEqual(t, uint32(100), f.set.Get(id).tag)
	// ...and eql was called with (incoming, resident), once.
	expectEqual(t, uint(1), rec.eqls)
	expectEqual(t, uint32(200), rec.last_a.tag)
	expectEqual(t, uint32(100), rec.last_b.tag)

	// A third add reports again.
	f.add(sm2_KT(7, 300))
	rec.expectDeleted(t, sm2_KT(7, 200), sm2_KT(7, 300))
	expectEqual(t, uint16(3), f.set.RefCount(id))
}

// Zig: test "term51: RefCountedSet.add: works without a deleted hook"
func TestRefCountedSet_term51_RefCountedSet_add_works_without_a_deleted_hook(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_specDefault, nil))

	expectEqual(t, uint16(1), f.add(sm2_K(1)))
	expectEqual(t, uint16(2), f.add(sm2_K(2)))
	expectEqual(t, uint16(1), f.add(sm2_K(1)))
	f.set.Release(2)
	// Tail trim then reuse of id 2.
	expectEqual(t, uint16(2), f.add(sm2_K(3)))
	f.expectItem(2, 3, 0, 1)
	// Resurrection of a non-tail dead item.
	f.set.Release(1)
	f.set.Release(1)
	expectEqual(t, uint16(1), f.add(sm2_K(1)))
	f.expectItem(1, 1, 0, 1)
	expectEqual(t, uint(2), f.set.Count())
}

// Zig: test "term51: RefCountedSet.add: dead tail items are trimmed and their ids reused"
func TestRefCountedSet_term51_RefCountedSet_add_dead_tail_items_are_trimmed_and_their_ids_reused(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(1))
	f.add(sm2_K(2))
	f.add(sm2_KT(3, 33))

	// Releasing keeps the entry: still in the table, value intact, no hook.
	f.set.Release(3)
	expectEqual(t, uint(2), f.set.Count())
	expectEqual(t, uint16(4), f.set.next_id)
	expectEqual(t, uint16(0), f.set.RefCount(3))
	f.expectItem(3, 3, 0, 0)
	expectEqual(t, uint32(33), f.itm(3).value.tag)
	expectEqual(t, uint(0), rec.del_n)
	// A dead item is not found by lookup.
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(3)))

	// The next add trims it (hook called, bucket cleared) and reuses id 3.
	expectEqual(t, uint16(3), f.add(sm2_K(4)))
	rec.expectDeleted(t, sm2_KT(3, 33))
	expectEqual(t, uint16(4), f.set.next_id)
	expectEqual(t, uint(3), f.set.Count())
	expectEqual(t, uint16(0), f.tbl()[3])
	f.expectItem(3, 4, 0, 1)
	expectEqual(t, uint32(4), f.itm(3).value.key)
}

// Zig: test "term51: RefCountedSet.add: re-adding the value of a dead tail item deletes then inserts afresh"
func TestRefCountedSet_term51_RefCountedSet_add_re_adding_the_value_of_a_dead_tail_item_deletes_then_inserts_afresh(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(1))
	f.add(sm2_KT(2, 5))
	f.set.Release(2)

	// Trimming runs before the lookup, so the dead item is gone, the hook
	// sees the old value, and the new value is a fresh insert at id 2.
	expectEqual(t, uint16(2), f.add(sm2_KT(2, 9)))
	rec.expectDeleted(t, sm2_KT(2, 5))
	expectEqual(t, uint32(9), f.itm(2).value.tag)
	expectEqual(t, uint16(1), f.set.RefCount(2))
	expectEqual(t, uint16(3), f.set.next_id)
	expectEqual(t, uint(2), f.set.Count())
}

// Zig: test "term51: RefCountedSet.add: trimming cascades through several dead tail items down to id 1"
func TestRefCountedSet_term51_RefCountedSet_add_trimming_cascades_through_several_dead_tail_items_down_to_id_1(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(1))
	f.add(sm2_K(2))
	f.add(sm2_K(3))
	// Release the lower ids first: they cannot be trimmed until 3 is dead.
	f.set.Release(1)
	f.set.Release(2)
	expectEqual(t, uint(1), f.set.Count())
	// A duplicate add of the live tail trims nothing.
	expectEqual(t, uint16(3), f.add(sm2_K(3)))
	rec.expectDeleted(t, sm2_K(3)) // the duplicate-value report only
	f.set.Release(3)
	f.set.Release(3)
	expectEqual(t, uint(0), f.set.Count())
	expectEqual(t, uint16(4), f.set.next_id)

	// Now everything is dead: 3, 2, 1 are trimmed in that order, and the
	// loop stops at next_id == 1 without touching item 0.
	expectEqual(t, uint16(1), f.add(sm2_K(9)))
	rec.expectDeleted(t, sm2_K(3), sm2_K(3), sm2_K(2), sm2_K(1))
	expectEqual(t, uint16(2), f.set.next_id)
	expectEqual(t, uint(1), f.set.Count())
	f.expectHole(0)
	f.expectHole(2)
	f.expectHole(3)
	f.expectItem(1, 9, 0, 1)
	f.expectTable(0, 0, 0, 0, 0, 0, 0, 0, 0, 1)
}

// Zig: test "term51: RefCountedSet.add: trimming also happens when the add is a duplicate"
func TestRefCountedSet_term51_RefCountedSet_add_trimming_also_happens_when_the_add_is_a_duplicate(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(1))
	f.add(sm2_K(2))
	f.add(sm2_K(3))
	f.set.Release(3)
	expectEqual(t, uint16(1), f.add(sm2_K(1)))
	// Order: trim hook for the dead tail first, then the duplicate report.
	rec.expectDeleted(t, sm2_K(3), sm2_K(1))
	expectEqual(t, uint16(3), f.set.next_id)
	f.expectHole(3)
}

// Zig: test "term51: RefCountedSet.add: a released non-tail item is resurrected with its own id"
func TestRefCountedSet_term51_RefCountedSet_add_a_released_non_tail_item_is_resurrected_with_its_own_id(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_KT(1, 7))
	f.add(sm2_K(2))
	f.add(sm2_K(3))
	f.set.Release(1)
	expectEqual(t, uint(2), f.set.Count())
	// Still in the table with its old value.
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	f.expectItem(1, 1, 0, 0)

	expectEqual(t, uint16(1), f.add(sm2_KT(1, 9)))
	// The dead value was reaped (hook), the new value stored under id 1,
	// and next_id was not consumed.
	rec.expectDeleted(t, sm2_KT(1, 7))
	expectEqual(t, uint32(9), f.itm(1).value.tag)
	f.expectItem(1, 1, 0, 1)
	expectEqual(t, uint16(4), f.set.next_id)
	expectEqual(t, uint(3), f.set.Count())
	f.expectStats(3)
}

// Zig: test "term51: RefCountedSet.add: a dead non-tail item outside the probe path stays until reached"
func TestRefCountedSet_term51_RefCountedSet_add_a_dead_non_tail_item_outside_the_probe_path_stays_until_reached(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(1))
	f.add(sm2_K(2))
	f.add(sm2_K(3))
	f.set.Release(1)

	// A value hashing elsewhere gets the next fresh id; the dead item stays.
	expectEqual(t, uint16(4), f.add(sm2_K(5)))
	expectEqual(t, uint16(5), f.set.next_id)
	expectEqual(t, uint(0), rec.del_n)
	f.expectItem(1, 1, 0, 0)
	expectEqual(t, uint(3), f.set.Count())
	f.expectStats(4)

	// Re-adding value 1 later still resurrects id 1.
	expectEqual(t, uint16(1), f.add(sm2_KT(1, 4)))
	rec.expectDeleted(t, sm2_K(1))
	expectEqual(t, uint16(5), f.set.next_id)
}

// Zig: test "term51: RefCountedSet.add: reaping a dead item in the probe path of a collision takes its smaller id"
func TestRefCountedSet_term51_RefCountedSet_add_reaping_a_dead_item_in_the_probe_path_of_a_collision_takes_its_smaller_id(t *testing.T) {
	var rec sm2_Rec
	// Every key hashes to bucket 3.
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_spec(1, 0, 3), &rec))

	f.add(sm2_K(1)) // id 1 bucket 3 psl 0
	f.add(sm2_K(2)) // id 2 bucket 4 psl 1
	f.add(sm2_K(3)) // id 3 bucket 5 psl 2
	f.expectItem(1, 3, 0, 1)
	f.expectItem(2, 4, 1, 1)
	f.expectItem(3, 5, 2, 1)
	f.expectStats(1, 1, 1)
	expectEqual(t, uint16(2), f.set.max_psl)

	// Kill the head of the chain (not the tail) and add a fourth key: it
	// lands in the dead item's bucket and takes its id 1.
	f.set.Release(1)
	expectEqual(t, uint16(1), f.add(sm2_KT(4, 44)))
	rec.expectDeleted(t, sm2_K(1))
	f.expectItem(1, 3, 0, 1)
	expectEqual(t, uint32(4), f.itm(1).value.key)
	expectEqual(t, uint32(44), f.itm(1).value.tag)
	f.expectItem(2, 4, 1, 1)
	f.expectItem(3, 5, 2, 1)
	expectEqual(t, uint16(4), f.set.next_id)
	expectEqual(t, uint(3), f.set.Count())
	f.expectStats(1, 1, 1)
	f.expectHole(4)

	// Kill the middle of the chain: the new key (psl 1 at bucket 4)
	// takes id 2 as it reaps that dead item.
	f.set.Release(2)
	expectEqual(t, uint16(2), f.add(sm2_KT(5, 55)))
	rec.expectDeleted(t, sm2_K(1), sm2_K(2))
	f.expectItem(2, 4, 1, 1)
	expectEqual(t, uint32(5), f.itm(2).value.key)
	expectEqual(t, uint16(4), f.set.next_id)
	f.expectStats(1, 1, 1)
}

// Zig: test "term51: RefCountedSet.add: full set returns OutOfMemory or NeedsRehash and leaves state untouched"
func TestRefCountedSet_term51_RefCountedSet_add_full_set_returns_OutOfMemory_or_NeedsRehash_and_leaves_state_untouched(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	// 16 -> table 16, item cap 13 -> ids 1..12 usable = 12 items.
	for k := uint32(0); k < 12; k++ {
		expectEqual(t, uint16(k+1), f.add(sm2_K(k)))
	}
	expectEqual(t, uint16(13), f.set.next_id)
	expectEqual(t, uint(12), f.set.Count())

	// All 12 alive: 12 < floor(13 * 0.9) = 11 is false -> OutOfMemory.
	{
		snap := f.snapshot()
		sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(12)))
		snap.expectSame(f)
	}

	// One dead (non-tail) id: living 11 < 11 is false -> still OutOfMemory.
	f.set.Release(1)
	expectEqual(t, uint(11), f.set.Count())
	{
		snap := f.snapshot()
		sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(12)))
		snap.expectSame(f)
	}

	// Two dead ids: living 10 < 11 -> NeedsRehash.
	f.set.Release(2)
	expectEqual(t, uint(10), f.set.Count())
	{
		snap := f.snapshot()
		sm2_expectError(t, ErrNeedsRehash, f.addErr(sm2_K(12)))
		snap.expectSame(f)
		// Even the value of a dead item cannot be resurrected: the id
		// check comes before the insert.
		sm2_expectError(t, ErrNeedsRehash, f.addErr(sm2_K(0)))
		snap.expectSame(f)
	}
	expectEqual(t, uint(0), rec.del_n)

	// Existing live values can still be added (lookup comes first).
	expectEqual(t, uint16(6), f.add(sm2_K(5)))
	expectEqual(t, uint16(2), f.set.RefCount(6))
	rec.expectDeleted(t, sm2_K(5))

	// Releasing the live tail makes room again: it is trimmed and its id reused.
	f.set.Release(12)
	expectEqual(t, uint16(12), f.add(sm2_K(12)))
	rec.expectDeleted(t, sm2_K(5), sm2_K(11))
	expectEqual(t, uint16(13), f.set.next_id)
	expectEqual(t, uint(10), f.set.Count())
	// ...and now it's full again with 2 dead ids: NeedsRehash.
	sm2_expectError(t, ErrNeedsRehash, f.addErr(sm2_K(13)))
}

// Zig: test "term51: RefCountedSet.add: maximum number of items with 16-bit ids"
func TestRefCountedSet_term51_RefCountedSet_add_maximum_number_of_items_with_16_bit_ids(t *testing.T) {
	// Largest layout: 65536 table entries, 53248 item slots, id 0 reserved.
	f := sm2_newSet(t, 65536, sm2_ctx(sm2_specDefault, nil))
	expectEqual(t, uint(53248), f.set.layout.cap)

	const max_items uint32 = 53247
	for k := uint32(0); k < max_items; k++ {
		expectEqual(t, uint16(k+1), f.add(sm2_K(k)))
	}
	expectEqual(t, uint(max_items), f.set.Count())
	expectEqual(t, uint16(53248), f.set.next_id)
	expectEqual(t, uint16(0), f.set.max_psl)

	// One past the maximum.
	sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(max_items)))
	expectEqual(t, uint(max_items), f.set.Count())
	expectEqual(t, uint16(53248), f.set.next_id)
	// The last item and an existing one are still reachable.
	expectEqual(t, Some(uint16(53247)), f.set.Lookup(sm2_K(53246)))
	expectEqual(t, uint16(101), f.add(sm2_K(100)))
	expectEqual(t, uint16(2), f.set.RefCount(101))
}

// Zig: test "term51: RefCountedSet.add: maximum number of items with 8-bit ids and rehash threshold"
func TestRefCountedSet_term51_RefCountedSet_add_maximum_number_of_items_with_8_bit_ids_and_rehash_threshold(t *testing.T) {
	f := sm2_newSet8(t, 256, sm2_ctx(sm2_specDefault, nil))
	expectEqual(t, uint(208), f.set.layout.cap)

	for k := uint32(0); k < 207; k++ {
		expectEqual(t, uint8(k+1), f.add(sm2_K(k)))
	}
	expectEqual(t, uint8(208), f.set.next_id)
	expectEqual(t, uint(207), f.set.Count())
	sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(207)))

	// Threshold: floor(208 * 0.9) = 187. living == 187 -> OutOfMemory,
	// living == 186 -> NeedsRehash.
	for id := uint8(1); id <= 20; id++ {
		f.set.Release(id)
	}
	expectEqual(t, uint(187), f.set.Count())
	sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(207)))
	f.set.Release(21)
	expectEqual(t, uint(186), f.set.Count())
	sm2_expectError(t, ErrNeedsRehash, f.addErr(sm2_K(207)))
}

// Zig: test "term51: RefCountedSet.add: probe length 31 is allowed but a set holding one rejects new values"
func TestRefCountedSet_term51_RefCountedSet_add_probe_length_31_is_allowed_but_a_set_holding_one_rejects_new_values(t *testing.T) {
	var rec sm2_Rec
	// Keys 0..63 all hash to bucket 0.
	f := sm2_newSetDel(t, 64, sm2_ctxDel(sm2_spec(64, 40, 0), &rec))

	// 31 colliders get psl 0..30; the stats array's last slot is unused.
	for k := uint32(0); k < 31; k++ {
		expectEqual(t, uint16(k+1), f.add(sm2_K(k)))
		f.expectItem(uint16(k+1), uint16(k), uint16(k), 1)
	}
	expectEqual(t, uint16(30), f.set.max_psl)
	expectEqual(t, uint16(0), f.set.psl_stats[31])

	// The 32nd collider ends up with psl 31 (the last stats slot): allowed.
	expectEqual(t, uint16(32), f.add(sm2_K(31)))
	f.expectItem(32, 31, 31, 1)
	expectEqual(t, uint16(31), f.set.max_psl)
	expectEqual(t, uint16(1), f.set.psl_stats[31])

	// The 33rd would need psl 32: refused as OutOfMemory, even though
	// there is plenty of room, and nothing changes.
	{
		snap := f.snapshot()
		sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(32)))
		snap.expectSame(f)
		// A value hashing elsewhere is refused too.
		sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(64)))
		snap.expectSame(f)
	}
	expectEqual(t, uint(32), f.set.Count())
	expectEqual(t, uint16(33), f.set.next_id)

	// Values already present are still found and ref counted: lookup
	// walks all 32 probe positions.
	expectEqual(t, uint16(32), f.add(sm2_K(31)))
	expectEqual(t, uint16(2), f.set.RefCount(32))
	expectEqual(t, uint16(1), f.add(sm2_K(0)))
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(32)))
	rec.expectDeleted(t, sm2_K(31), sm2_K(0))
}

// Zig: test "term51: RefCountedSet.add: probe length check comes before the id exhaustion check"
func TestRefCountedSet_term51_RefCountedSet_add_probe_length_check_comes_before_the_id_exhaustion_check(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 64, sm2_ctxDel(sm2_spec(64, 40, 0), &rec))

	// 31 colliders (ids 1..31, psl 0..30), 19 keys at bucket 40.. (ids
	// 32..50, psl 0..18), then the 32nd collider (id 51, psl 31).
	for k := uint32(0); k < 31; k++ {
		f.add(sm2_K(k))
	}
	for k := uint32(64); k < 83; k++ {
		f.add(sm2_K(k))
	}
	expectEqual(t, uint16(51), f.add(sm2_K(31)))
	expectEqual(t, uint16(52), f.set.next_id)
	expectEqual(t, uint(51), f.set.Count())
	f.expectItem(32, 40, 0, 1)
	f.expectItem(50, 58, 18, 1)
	f.expectItem(51, 31, 31, 1)
	expectEqual(t, uint16(2), f.set.psl_stats[0])
	expectEqual(t, uint16(2), f.set.psl_stats[18])
	expectEqual(t, uint16(1), f.set.psl_stats[19])
	expectEqual(t, uint16(1), f.set.psl_stats[30])
	expectEqual(t, uint16(1), f.set.psl_stats[31])

	// Make 13 non-tail ids dead: living 38 < floor(52 * 0.9) = 46, which
	// alone would say NeedsRehash.
	for id := uint16(32); id <= 44; id++ {
		f.set.Release(id)
	}
	expectEqual(t, uint(38), f.set.Count())

	{
		snap := f.snapshot()
		sm2_expectError(t, ErrOutOfMemory, f.addErr(sm2_K(200)))
		snap.expectSame(f)
		// addWithId on a dead id hits the same check, before deleting anything.
		sm2_expectError(t, ErrOutOfMemory, f.addWithIdErr(sm2_K(300), 32))
		snap.expectSame(f)
		sm2_expectError(t, ErrOutOfMemory, f.addWithIdErr(sm2_K(300), 60))
		snap.expectSame(f)
	}
	expectEqual(t, uint(0), rec.del_n)

	// With the last stats slot cleared the same add reports NeedsRehash.
	f.set.psl_stats[31] = 0
	sm2_expectError(t, ErrNeedsRehash, f.addErr(sm2_K(200)))
}

// Zig: test "term51: RefCountedSet.addContext: uses the given context instead of the stored one"
func TestRefCountedSet_term51_RefCountedSet_addContext_uses_the_given_context_instead_of_the_stored_one(t *testing.T) {
	var rec1 sm2_Rec
	var rec2 sm2_Rec
	// Stored context: every key hashes to bucket 5.
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_spec(1, 0, 5), &rec1))
	// Alternative context: identity hash.
	alt := sm2_ctxDel(sm2_specDefault, &rec2)

	expectEqual(t, uint16(1), f.addContext(sm2_K(1), alt))
	f.expectItem(1, 1, 0, 1)
	// The stored context hashes elsewhere, so it does not find it...
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	expectEqual(t, Some(uint16(1)), f.set.LookupContext(sm2_K(1), alt))
	// ...and `add` makes a second entry for the same key at bucket 5.
	expectEqual(t, uint16(2), f.add(sm2_K(1)))
	f.expectItem(2, 5, 0, 1)
	expectEqual(t, Some(uint16(2)), f.set.Lookup(sm2_K(1)))

	// Duplicate reports go to the context that was passed in.
	expectEqual(t, uint16(1), f.addContext(sm2_KT(1, 8), alt))
	rec2.expectDeleted(t, sm2_KT(1, 8))
	expectEqual(t, uint(0), rec1.del_n)
	expectEqual(t, uint16(2), f.add(sm2_KT(1, 9)))
	rec1.expectDeleted(t, sm2_KT(1, 9))
	expectEqual(t, uint(1), rec2.del_n)
	expectEqual(t, uint16(2), f.set.RefCount(1))
	expectEqual(t, uint16(2), f.set.RefCount(2))

	// Trimming a dead tail item reports to the passed context as well.
	f.set.Release(2)
	f.set.Release(2)
	expectEqual(t, uint16(2), f.addContext(sm2_K(2), alt))
	rec2.expectDeleted(t, sm2_KT(1, 8), sm2_K(1))
	expectEqual(t, uint(1), rec1.del_n)
}

// Zig: test "term51: RefCountedSet.addWithId: an id at or past next_id is ignored and a fresh id is returned"
func TestRefCountedSet_term51_RefCountedSet_addWithId_an_id_at_or_past_next_id_is_ignored_and_a_fresh_id_is_returned(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	// id == next_id (1): falls through to add, which returns the id it
	// used. The result is non-null even though the requested id was used.
	expectEqual(t, Some(uint16(1)), f.addWithId(sm2_K(5), 1))
	// id past next_id: ignored, the next sequential id is assigned.
	expectEqual(t, Some(uint16(2)), f.addWithId(sm2_K(6), 7))
	expectEqual(t, Some(uint16(3)), f.addWithId(sm2_K(7), 3))
	expectEqual(t, uint16(4), f.set.next_id)
	expectEqual(t, uint(3), f.set.Count())
	// Largest id value is accepted too.
	expectEqual(t, Some(uint16(4)), f.addWithId(sm2_K(8), math.MaxUint16))
	// A duplicate reached through the fall-through is just ref counted.
	expectEqual(t, Some(uint16(2)), f.addWithId(sm2_K(6), 9))
	expectEqual(t, uint16(2), f.set.RefCount(2))
	rec.expectDeleted(t, sm2_K(6))
}

// Zig: test "term51: RefCountedSet.addWithId: a live id holding an equal value is ref counted and null returned"
func TestRefCountedSet_term51_RefCountedSet_addWithId_a_live_id_holding_an_equal_value_is_ref_counted_and_null_returned(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_KT(5, 50))
	f.add(sm2_KT(6, 60))
	rec.eqls = 0

	expectEqual(t, None[uint16](), f.addWithId(sm2_KT(6, 61), 2))
	expectEqual(t, uint16(2), f.set.RefCount(2))
	expectEqual(t, uint16(1), f.set.RefCount(1))
	// eql(incoming, resident), then the context is told the incoming copy is deleted.
	expectEqual(t, uint(1), rec.eqls)
	expectEqual(t, uint32(61), rec.last_a.tag)
	expectEqual(t, uint32(60), rec.last_b.tag)
	rec.expectDeleted(t, sm2_KT(6, 61))
	expectEqual(t, uint32(60), f.itm(2).value.tag)
	expectEqual(t, uint(2), f.set.Count())
	expectEqual(t, uint16(3), f.set.next_id)

	// The lowest valid id (1) takes the same branch.
	expectEqual(t, None[uint16](), f.addWithId(sm2_K(5), 1))
	expectEqual(t, uint16(2), f.set.RefCount(1))
}

// Zig: test "term51: RefCountedSet.addWithId: a live id holding a different value falls back to add"
func TestRefCountedSet_term51_RefCountedSet_addWithId_a_live_id_holding_a_different_value_falls_back_to_add(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(5))
	f.add(sm2_K(6))

	// New value: gets the next id, the requested one is untouched.
	expectEqual(t, Some(uint16(3)), f.addWithId(sm2_K(9), 1))
	expectEqual(t, uint16(1), f.set.RefCount(1))
	expectEqual(t, uint32(5), f.itm(1).value.key)
	expectEqual(t, uint16(4), f.set.next_id)

	// Value that lives under another id: that id is returned (non-null).
	expectEqual(t, Some(uint16(2)), f.addWithId(sm2_KT(6, 3), 1))
	expectEqual(t, uint16(2), f.set.RefCount(2))
	expectEqual(t, uint16(1), f.set.RefCount(1))
	rec.expectDeleted(t, sm2_KT(6, 3))
	expectEqual(t, uint(3), f.set.Count())
}

// Zig: test "term51: RefCountedSet.addWithId: a dead id is cleaned and reused for a new value"
func TestRefCountedSet_term51_RefCountedSet_addWithId_a_dead_id_is_cleaned_and_reused_for_a_new_value(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(5))
	f.add(sm2_KT(6, 60))
	f.add(sm2_K(7))
	f.set.Release(2)
	expectEqual(t, uint(2), f.set.Count())

	expectEqual(t, None[uint16](), f.addWithId(sm2_K(9), 2))
	// The dead value was reported deleted, its bucket (6) is free, and the
	// new value sits in id 2 at bucket 9.
	rec.expectDeleted(t, sm2_KT(6, 60))
	expectEqual(t, uint16(0), f.tbl()[6])
	f.expectItem(2, 9, 0, 1)
	expectEqual(t, uint32(9), f.itm(2).value.key)
	expectEqual(t, uint(3), f.set.Count())
	expectEqual(t, uint16(4), f.set.next_id)
	f.expectStats(3)
}

// Zig: test "term51: RefCountedSet.addWithId: a dead tail id is reused without trimming next_id"
func TestRefCountedSet_term51_RefCountedSet_addWithId_a_dead_tail_id_is_reused_without_trimming_next_id(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(1))
	f.add(sm2_K(2))
	f.add(sm2_K(3))
	f.set.Release(3)

	expectEqual(t, None[uint16](), f.addWithId(sm2_K(9), 3))
	rec.expectDeleted(t, sm2_K(3))
	expectEqual(t, uint16(4), f.set.next_id)
	f.expectItem(3, 9, 0, 1)
	expectEqual(t, uint(3), f.set.Count())

	// The same value again under the same id is the "equal live" branch.
	expectEqual(t, None[uint16](), f.addWithId(sm2_K(9), 3))
	expectEqual(t, uint16(2), f.set.RefCount(3))
	expectEqual(t, uint(3), f.set.Count())
}

// Zig: test "term51: RefCountedSet.addWithId: the dead item's own value can be re-added under its id"
func TestRefCountedSet_term51_RefCountedSet_addWithId_the_dead_item_s_own_value_can_be_re_added_under_its_id(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_KT(1, 10))
	f.add(sm2_K(2))
	f.set.Release(1)
	expectEqual(t, None[uint16](), f.addWithId(sm2_KT(1, 11), 1))
	rec.expectDeleted(t, sm2_KT(1, 10))
	f.expectItem(1, 1, 0, 1)
	expectEqual(t, uint32(11), f.itm(1).value.tag)
	expectEqual(t, uint(2), f.set.Count())
}

// Zig: test "term51: RefCountedSet.addWithId: a value already live under another id returns that id and overcounts living"
func TestRefCountedSet_term51_RefCountedSet_addWithId_a_value_already_live_under_another_id_returns_that_id_and_overcounts_living(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(5))
	f.add(sm2_K(6))
	f.add(sm2_KT(7, 70))
	f.set.Release(1)
	expectEqual(t, uint(2), f.set.Count())

	// Dead id 1 is wiped, then upsert finds the live value 7 under id 3.
	expectEqual(t, Some(uint16(3)), f.addWithId(sm2_KT(7, 71), 1))
	rec.expectDeleted(t, sm2_K(5), sm2_KT(7, 71))
	expectEqual(t, uint16(2), f.set.RefCount(3))
	f.expectHole(1)
	expectEqual(t, uint32(70), f.itm(3).value.tag)
	// Defect pinned: living is incremented although no item came alive.
	// Only ids 2 and 3 are live, yet count() says 3.
	expectEqual(t, uint(3), f.set.Count())

	// Releasing both references of id 3 drops living only by one.
	f.set.Release(3)
	expectEqual(t, uint(3), f.set.Count())
	f.set.Release(3)
	expectEqual(t, uint(2), f.set.Count())
	f.set.Release(2)
	expectEqual(t, uint(1), f.set.Count())
}

// Zig: test "term51: RefCountedSet.addWithId: a dead id may be satisfied by a smaller dead id in the probe path"
func TestRefCountedSet_term51_RefCountedSet_addWithId_a_dead_id_may_be_satisfied_by_a_smaller_dead_id_in_the_probe_path(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_spec(1, 0, 3), &rec))

	f.add(sm2_K(1)) // id 1 bucket 3 psl 0
	f.add(sm2_K(2)) // id 2 bucket 4 psl 1
	f.add(sm2_K(3)) // id 3 bucket 5 psl 2
	f.set.Release(1)
	f.set.Release(2)
	expectEqual(t, uint(1), f.set.Count())

	// Id 2 is deleted first (id 3 shifts back to bucket 4, psl 1), then the
	// insert runs into dead id 1 at bucket 3 and takes that smaller id.
	expectEqual(t, Some(uint16(1)), f.addWithId(sm2_K(4), 2))
	rec.expectDeleted(t, sm2_K(2), sm2_K(1))
	f.expectItem(1, 3, 0, 1)
	expectEqual(t, uint32(4), f.itm(1).value.key)
	f.expectItem(3, 4, 1, 1)
	f.expectHole(2)
	f.expectTable(0, 0, 0, 1, 3)
	f.expectStats(1, 1)
	expectEqual(t, uint16(1), f.set.max_psl)
	expectEqual(t, uint(2), f.set.Count())
	expectEqual(t, uint16(4), f.set.next_id)

	// Id 2 is now a hole: asking for it deletes nothing (no hook) and
	// gets the id.
	expectEqual(t, None[uint16](), f.addWithId(sm2_K(5), 2))
	expectEqual(t, uint(2), rec.del_n)
	f.expectItem(2, 5, 2, 1)
	expectEqual(t, uint(3), f.set.Count())
}

// Zig: test "term51: RefCountedSet.addWithId: a dead id is kept when the probe path holds a dead larger id"
func TestRefCountedSet_term51_RefCountedSet_addWithId_a_dead_id_is_kept_when_the_probe_path_holds_a_dead_larger_id(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_spec(1, 0, 3), &rec))

	f.add(sm2_K(1)) // id 1 bucket 3 psl 0
	f.add(sm2_K(2)) // id 2 bucket 4 psl 1
	f.add(sm2_K(3)) // id 3 bucket 5 psl 2
	f.add(sm2_K(4)) // id 4 bucket 6 psl 3
	f.set.Release(2)
	f.set.Release(3)
	f.expectStats(1, 1, 1, 1)

	// Delete id 2 (ids 3, 4 shift back one bucket, dead id 3 included),
	// then the new value (psl 1, bucket 4) reaps dead id 3 but keeps id 2
	// because 3 > 2.
	expectEqual(t, None[uint16](), f.addWithId(sm2_K(9), 2))
	rec.expectDeleted(t, sm2_K(2), sm2_K(3))
	f.expectItem(1, 3, 0, 1)
	f.expectItem(2, 4, 1, 1)
	expectEqual(t, uint32(9), f.itm(2).value.key)
	f.expectHole(3)
	f.expectItem(4, 5, 2, 1)
	f.expectTable(0, 0, 0, 1, 2, 4)
	f.expectStats(1, 1, 1)
	expectEqual(t, uint16(2), f.set.max_psl)
	expectEqual(t, uint(3), f.set.Count())
	expectEqual(t, uint16(5), f.set.next_id)
}

// Zig: test "term51: RefCountedSet.addWithIdContext: uses the given context"
func TestRefCountedSet_term51_RefCountedSet_addWithIdContext_uses_the_given_context(t *testing.T) {
	var rec1 sm2_Rec
	var rec2 sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_spec(1, 0, 5), &rec1))
	alt := sm2_ctxDel(sm2_specDefault, &rec2)

	expectEqual(t, Some(uint16(1)), f.addWithIdContext(sm2_K(1), 1, alt))
	expectEqual(t, Some(uint16(2)), f.addWithIdContext(sm2_K(2), 2, alt))
	f.expectItem(1, 1, 0, 1)
	f.expectItem(2, 2, 0, 1)

	// Equal live branch: the hook goes to the passed context.
	expectEqual(t, None[uint16](), f.addWithIdContext(sm2_KT(2, 4), 2, alt))
	rec2.expectDeleted(t, sm2_KT(2, 4))

	// Dead id branch: both the cleanup of the old value and the
	// insertion use the passed context (bucket 3 = identity hash).
	f.set.Release(1)
	expectEqual(t, None[uint16](), f.addWithIdContext(sm2_K(3), 1, alt))
	rec2.expectDeleted(t, sm2_KT(2, 4), sm2_K(1))
	f.expectItem(1, 3, 0, 1)
	expectEqual(t, uint(0), rec1.del_n)

	// `addWithId` goes through the stored context (everything at bucket 5).
	expectEqual(t, Some(uint16(3)), f.addWithId(sm2_K(4), 9))
	f.expectItem(3, 5, 0, 1)
}

// Zig: test "term51: RefCountedSet.use: increments the ref count by one"
func TestRefCountedSet_term51_RefCountedSet_use_increments_the_ref_count_by_one(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_specDefault, nil))

	a := f.add(sm2_K(1))
	b := f.add(sm2_K(2))
	f.set.Use(a)
	expectEqual(t, uint16(2), f.set.RefCount(a))
	// Also callable through a const pointer.
	// Zig-only: `const cs: *const S = &f.set;` Go has no const pointers;
	// cs is the same pointer.
	cs := &f.set
	cs.Use(a)
	cs.Use(a)
	expectEqual(t, uint16(4), f.set.RefCount(a))
	expectEqual(t, uint16(1), f.set.RefCount(b))
	// Doesn't touch the number of living items or ids.
	expectEqual(t, uint(2), f.set.Count())
	expectEqual(t, uint16(3), f.set.next_id)
	// The highest valid id (cap - 1 = 12) can be used once it holds an item.
	for k := uint32(3); k <= 12; k++ {
		f.add(sm2_K(k))
	}
	f.set.Use(12)
	expectEqual(t, uint16(2), f.set.RefCount(12))
}

// Zig: test "term51: RefCountedSet.use: counts up to the largest value of the count type"
func TestRefCountedSet_term51_RefCountedSet_use_counts_up_to_the_largest_value_of_the_count_type(t *testing.T) {
	f := sm2_newSet8(t, 16, sm2_ctx(sm2_specDefault, nil))

	id := f.add(sm2_K(1))
	for n := 1; n < 255; n++ {
		f.set.Use(id)
	}
	expectEqual(t, uint8(255), f.set.RefCount(id))
	// Re-adding it once more would overflow the u8 counter (a runtime
	// panic in safe builds), so we stop at the maximum.
}

// Zig: test "term51: RefCountedSet.useMultiple: adds n to the ref count"
func TestRefCountedSet_term51_RefCountedSet_useMultiple_adds_n_to_the_ref_count(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_specDefault, nil))

	a := f.add(sm2_K(1))
	b := f.add(sm2_K(2))
	f.set.UseMultiple(a, 0)
	expectEqual(t, uint16(1), f.set.RefCount(a))
	f.set.UseMultiple(a, 1)
	expectEqual(t, uint16(2), f.set.RefCount(a))
	f.set.UseMultiple(a, 5)
	expectEqual(t, uint16(7), f.set.RefCount(a))
	f.set.UseMultiple(a, 65535-7)
	expectEqual(t, uint16(65535), f.set.RefCount(a))
	expectEqual(t, uint16(1), f.set.RefCount(b))
	expectEqual(t, uint(2), f.set.Count())

	// Const pointer receiver.
	// Zig-only: `const cs: *const S = &f.set;` Go has no const pointers.
	cs := &f.set
	cs.UseMultiple(b, 3)
	expectEqual(t, uint16(4), f.set.RefCount(b))
}

// Zig: test "term51: RefCountedSet.useMultiple: count type is independent of the id type"
func TestRefCountedSet_term51_RefCountedSet_useMultiple_count_type_is_independent_of_the_id_type(t *testing.T) {
	f := sm2_newSetMix(t, 16, sm2_ctx(sm2_specDefault, nil))

	id := f.add(sm2_K(1))
	f.set.UseMultiple(id, 100_000)
	expectEqual(t, uint32(100_001), f.set.RefCount(id))
	f.set.ReleaseMultiple(id, 65535)
	expectEqual(t, uint32(100_001-65535), f.set.RefCount(id))
	expectEqual(t, uint(1), f.set.Count())
}

// Zig: test "term51: RefCountedSet.get: returns a pointer to the stored value"
func TestRefCountedSet_term51_RefCountedSet_get_returns_a_pointer_to_the_stored_value(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_specDefault, nil))

	a := f.add(sm2_KT(1, 10))
	b := f.add(sm2_KT(2, 20))
	pa := f.set.Get(a)
	expectEqual(t, uint32(1), pa.key)
	expectEqual(t, uint32(10), pa.tag)
	expectEqual(t, uint32(2), f.set.Get(b).key)
	expect(t, pa == &f.itm(uint(a)).value, "pa == &f.itm(a).value")
	expect(t, pa == f.set.Get(a), "pa == f.set.get(f.buf, a)")
	// get does not change the ref count.
	expectEqual(t, uint16(1), f.set.RefCount(a))

	// The pointer is mutable and writes the stored value (tag is not part
	// of eql/hash so the item is still found).
	pa.tag = 99
	expectEqual(t, uint32(99), f.set.Get(a).tag)
	expectEqual(t, Some(a), f.set.Lookup(sm2_K(1)))
	// Const pointer receiver.
	// Zig-only: `const cs: *const S = &f.set;` Go has no const pointers.
	cs := &f.set
	expectEqual(t, uint32(20), cs.Get(b).tag)
}

// Zig: test "term51: RefCountedSet.release: decrements, and the item stays at zero references"
func TestRefCountedSet_term51_RefCountedSet_release_decrements_and_the_item_stays_at_zero_references(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	a := f.add(sm2_KT(1, 7))
	f.add(sm2_K(1))
	f.add(sm2_K(1))
	b := f.add(sm2_K(2))
	expectEqual(t, uint16(3), f.set.RefCount(a))
	expectEqual(t, uint(2), f.set.Count())
	rec.del_n = 0

	f.set.Release(a)
	expectEqual(t, uint16(2), f.set.RefCount(a))
	expectEqual(t, uint(2), f.set.Count())
	f.set.Release(a)
	expectEqual(t, uint16(1), f.set.RefCount(a))
	expectEqual(t, uint(2), f.set.Count())
	// Last reference: living drops by one.
	f.set.Release(a)
	expectEqual(t, uint16(0), f.set.RefCount(a))
	expectEqual(t, uint(1), f.set.Count())
	expectEqual(t, uint16(1), f.set.RefCount(b))

	// The entry stays: bucket, value and id are kept, no hook ran, the
	// next id is unchanged, but the value is not found any more.
	f.expectItem(a, 1, 0, 0)
	expectEqual(t, uint32(7), f.itm(uint(a)).value.tag)
	expectEqual(t, uint(0), rec.del_n)
	expectEqual(t, uint16(3), f.set.next_id)
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	expectEqual(t, Some(b), f.set.Lookup(sm2_K(2)))

	f.set.Release(b)
	expectEqual(t, uint(0), f.set.Count())
}

// Zig: test "term51: RefCountedSet.releaseMultiple: subtracts n and only the last reference lowers living"
func TestRefCountedSet_term51_RefCountedSet_releaseMultiple_subtracts_n_and_only_the_last_reference_lowers_living(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_specDefault, nil))

	a := f.add(sm2_K(1))
	b := f.add(sm2_K(2))
	f.set.UseMultiple(a, 9) // 10
	f.set.ReleaseMultiple(a, 3)
	expectEqual(t, uint16(7), f.set.RefCount(a))
	expectEqual(t, uint(2), f.set.Count())
	// n = 0 on a live item changes nothing.
	f.set.ReleaseMultiple(a, 0)
	expectEqual(t, uint16(7), f.set.RefCount(a))
	expectEqual(t, uint(2), f.set.Count())
	// n = ref - 1 leaves one reference.
	f.set.ReleaseMultiple(a, 6)
	expectEqual(t, uint16(1), f.set.RefCount(a))
	expectEqual(t, uint(2), f.set.Count())
	// n == ref releases everything at once.
	f.set.UseMultiple(a, 4) // 5
	f.set.ReleaseMultiple(a, 5)
	expectEqual(t, uint16(0), f.set.RefCount(a))
	expectEqual(t, uint(1), f.set.Count())
	// The entry stays, like with release.
	f.expectItem(a, 1, 0, 0)
	expectEqual(t, uint16(1), f.set.RefCount(b))
	f.set.ReleaseMultiple(b, 1)
	expectEqual(t, uint(0), f.set.Count())
}

// Zig: test "term51: RefCountedSet.releaseMultiple: releasing zero from a dead item lowers living again"
func TestRefCountedSet_term51_RefCountedSet_releaseMultiple_releasing_zero_from_a_dead_item_lowers_living_again(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_specDefault, nil))

	a := f.add(sm2_K(1))
	f.add(sm2_K(2))
	f.set.Release(a)
	expectEqual(t, uint(1), f.set.Count())
	// The assertion `ref >= n` holds for n == 0, and the zero check then
	// treats the item as having just reached 0 references, so living is
	// decremented a second time (defect pinned: id 2 is still alive).
	f.set.ReleaseMultiple(a, 0)
	expectEqual(t, uint16(0), f.set.RefCount(a))
	expectEqual(t, uint(0), f.set.Count())
}

// Zig: test "term51: RefCountedSet.refCount: reads the count of any slot below the capacity"
func TestRefCountedSet_term51_RefCountedSet_refCount_reads_the_count_of_any_slot_below_the_capacity(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_specDefault, nil))

	// Unused slots read 0, including the last valid id (cap - 1 = 12).
	expectEqual(t, uint16(0), f.set.RefCount(1))
	expectEqual(t, uint16(0), f.set.RefCount(12))
	a := f.add(sm2_K(1))
	expectEqual(t, uint16(1), f.set.RefCount(a))
	f.add(sm2_K(1))
	expectEqual(t, uint16(2), f.set.RefCount(a))
	f.set.ReleaseMultiple(a, 2)
	// Dead items read 0 (no assertion on refCount).
	expectEqual(t, uint16(0), f.set.RefCount(a))
	// Const pointer receiver.
	// Zig-only: `const cs: *const S = &f.set;` Go has no const pointers.
	cs := &f.set
	expectEqual(t, uint16(0), cs.RefCount(12))
}

// Zig: test "term51: RefCountedSet.count: counts living items only"
func TestRefCountedSet_term51_RefCountedSet_count_counts_living_items_only(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	expectEqual(t, uint(0), f.set.Count())
	f.add(sm2_K(1))
	expectEqual(t, uint(1), f.set.Count())
	f.add(sm2_K(1)) // duplicate
	expectEqual(t, uint(1), f.set.Count())
	f.add(sm2_K(2))
	f.add(sm2_K(3))
	expectEqual(t, uint(3), f.set.Count())
	f.set.Release(3) // dead, still in the table
	expectEqual(t, uint(2), f.set.Count())
	f.set.Release(2)
	expectEqual(t, uint(1), f.set.Count())
	// Trimming dead items doesn't change it, reusing an id raises it.
	f.add(sm2_K(1))
	expectEqual(t, uint(1), f.set.Count())
	f.add(sm2_K(9))
	expectEqual(t, uint(2), f.set.Count())
	// Const pointer receiver.
	// Zig-only: `const cs: *const S = &f.set;` Go has no const pointers.
	cs := &f.set
	expectEqual(t, uint(2), cs.Count())
}

// Zig: test "term51: RefCountedSet.lookup: finds live values only"
func TestRefCountedSet_term51_RefCountedSet_lookup_finds_live_values_only(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_specDefault, nil))

	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	f.add(sm2_K(1))
	f.add(sm2_K(2))
	expectEqual(t, Some(uint16(1)), f.set.Lookup(sm2_K(1)))
	expectEqual(t, Some(uint16(2)), f.set.Lookup(sm2_K(2)))
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(3)))
	// Lookup does not change ref counts or the count.
	expectEqual(t, uint16(1), f.set.RefCount(1))
	expectEqual(t, uint(2), f.set.Count())
	// A key hashing to the same bucket as a present one (17 & 15 == 1) but different.
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(17)))
	// Dead: not found, found again once resurrected.
	f.set.Release(1)
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	expectEqual(t, Some(uint16(2)), f.set.Lookup(sm2_K(2)))
	expectEqual(t, uint16(1), f.add(sm2_K(1)))
	expectEqual(t, Some(uint16(1)), f.set.Lookup(sm2_K(1)))
}

// Zig: test "term51: RefCountedSet.lookupContext: eql gets the searched value first and the resident second"
func TestRefCountedSet_term51_RefCountedSet_lookupContext_eql_gets_the_searched_value_first_and_the_resident_second(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSet(t, 16, sm2_ctx(sm2_specDefault, &rec))

	f.add(sm2_KT(4, 40))
	rec.eqls = 0
	expectEqual(t, Some(uint16(1)), f.set.Lookup(sm2_KT(4, 41)))
	expectEqual(t, uint(1), rec.eqls)
	expectEqual(t, uint32(41), rec.last_a.tag)
	expectEqual(t, uint32(40), rec.last_b.tag)

	// A different context is used for both hash and eql.
	var rec2 sm2_Rec
	alt := sm2_ctx(sm2_spec(1, 0, 4), &rec2)
	expectEqual(t, Some(uint16(1)), f.set.LookupContext(sm2_KT(4, 42), alt))
	expectEqual(t, uint(1), rec2.eqls)
	expectEqual(t, uint32(42), rec2.last_a.tag)
	expectEqual(t, uint(1), rec.eqls)
	// A context that hashes to an empty bucket finds nothing.
	alt2 := sm2_ctx(sm2_spec(1, 0, 9), nil)
	expectEqual(t, None[uint16](), f.set.LookupContext(sm2_K(4), alt2))
}

// Zig: test "term51: RefCountedSet.lookupContext: probe sequence is bounded by max_psl and skips other lengths"
func TestRefCountedSet_term51_RefCountedSet_lookupContext_probe_sequence_is_bounded_by_max_psl_and_skips_other_lengths(t *testing.T) {
	var rec sm2_Rec
	// Every key hashes to bucket 3.
	f := sm2_newSet(t, 16, sm2_ctx(sm2_spec(1, 0, 3), &rec))

	for k := uint32(1); k <= 4; k++ {
		f.add(sm2_K(k))
	}
	f.expectTable(0, 0, 0, 1, 2, 3, 4)
	f.expectStats(1, 1, 1, 1)
	expectEqual(t, uint16(3), f.set.max_psl)

	// A missing value is compared with each of the 4 residents (psl 0..3
	// = max_psl + 1 positions), then null.
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(99)))
	expectEqual(t, uint(4), rec.eqls)
	// A present value stops at its position: 3rd bucket -> 3 compares.
	rec.eqls = 0
	expectEqual(t, Some(uint16(3)), f.set.Lookup(sm2_K(3)))
	expectEqual(t, uint(3), rec.eqls)
	// Dead residents are not compared.
	f.set.Release(2)
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(99)))
	expectEqual(t, uint(3), rec.eqls)
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(2)))
	expectEqual(t, uint(3), rec.eqls)
	expectEqual(t, Some(uint16(4)), f.set.Lookup(sm2_K(4)))
}

// Zig: test "term51: RefCountedSet.lookupContext: residents whose psl differs from the probe step are not compared"
func TestRefCountedSet_term51_RefCountedSet_lookupContext_residents_whose_psl_differs_from_the_probe_step_are_not_compared(t *testing.T) {
	var rec sm2_Rec
	// Keys 0..9 hash to 3, keys 10..19 to 4, keys 20..29 to 5, ...
	f := sm2_newSet(t, 64, sm2_ctx(sm2_spec(10, 1, 3), &rec))

	f.add(sm2_K(1))  // id 1, hash 3 -> bucket 3 psl 0
	f.add(sm2_K(2))  // id 2, hash 3 -> bucket 4 psl 1
	f.add(sm2_K(10)) // id 3, hash 4 -> bucket 5 psl 1
	f.expectItem(1, 3, 0, 1)
	f.expectItem(2, 4, 1, 1)
	f.expectItem(3, 5, 1, 1)
	expectEqual(t, uint16(1), f.set.max_psl)

	// Key 10: i = 0 -> bucket 4 holds psl 1 (skipped, no eql), i = 1 ->
	// bucket 5 holds psl 1 (compared, equal).
	rec.eqls = 0
	expectEqual(t, Some(uint16(3)), f.set.Lookup(sm2_K(10)))
	expectEqual(t, uint(1), rec.eqls)
	// Key 11 (hash 4, absent): same walk, one compare, null.
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(11)))
	expectEqual(t, uint(1), rec.eqls)
	// Key 20 (hash 5, absent): i = 0 bucket 5 has psl 1 (skip); i = 1 bucket 6 is empty.
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(20)))
	expectEqual(t, uint(0), rec.eqls)
}

// Zig: test "term51: RefCountedSet.lookupContext: a resident with a shorter probe length ends the search early"
func TestRefCountedSet_term51_RefCountedSet_lookupContext_a_resident_with_a_shorter_probe_length_ends_the_search_early(t *testing.T) {
	var rec sm2_Rec
	// Group g of 10 keys hashes to g + 3 (buckets 3, 4, 5, ...).
	f := sm2_newSet(t, 64, sm2_ctx(sm2_spec(10, 1, 3), &rec))

	// A long chain far away raises max_psl to 5.
	for k := uint32(100); k < 106; k++ { // group 10 -> hash 13
		f.add(sm2_K(k))
	}
	expectEqual(t, uint16(5), f.set.max_psl)
	// Key 0 at bucket 3, key 20 at bucket 5 (psl 0).
	f.add(sm2_K(0))
	f.add(sm2_K(20))
	f.expectItem(7, 3, 0, 1)
	f.expectItem(8, 5, 0, 1)

	// Key 1 hashes to 3: i = 0 compares with key 0; i = 1 is bucket 4
	// (empty) -> null, with max_psl 5 not reached.
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	expectEqual(t, uint(1), rec.eqls)
	// Key 21 hashes to 5: i = 0 compares with key 20.
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(21)))
	expectEqual(t, uint(1), rec.eqls)
	// Key 30 hashes to 6 (empty).
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(30)))
	expectEqual(t, uint(0), rec.eqls)
	// Key 21 collides with key 20 and lands at bucket 6 with psl 1.
	expectEqual(t, uint16(9), f.add(sm2_K(21)))
	f.expectItem(9, 6, 1, 1)
	expectEqual(t, Some(uint16(9)), f.set.Lookup(sm2_K(21)))
	// Key 22 (hash 5, absent): i = 0 -> key 20 compared; i = 1 -> key 21
	// (psl 1) compared; i = 2 -> bucket 7 empty.
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(22)))
	expectEqual(t, uint(2), rec.eqls)
	// Key 40 (hash 7) sees an empty bucket first.
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(40)))
	// A lookup whose hash lands in the middle of the far chain (bucket
	// 15, psl 2): every resident has a psl different from the step, so
	// no compare happens, and the walk ends at an empty bucket.
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(120))) // hash 15
	expectEqual(t, uint(0), rec.eqls)

	// Early exit: key 10 sits at bucket 4 with psl 0. Looking up key 1
	// (hash 3): i = 0 compares with key 0; at i = 1 the resident at
	// bucket 4 has psl 0 < 1 so the search ends, although max_psl is 5.
	expectEqual(t, uint16(10), f.add(sm2_K(10)))
	f.expectItem(10, 4, 0, 1)
	rec.eqls = 0
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	expectEqual(t, uint(1), rec.eqls)
}

// Zig: test "term51: RefCountedSet.lookupContext: probing wraps around the end of the table"
func TestRefCountedSet_term51_RefCountedSet_lookupContext_probing_wraps_around_the_end_of_the_table(t *testing.T) {
	// Every key hashes to bucket 14 of 16.
	f := sm2_newSet(t, 16, sm2_ctx(sm2_spec(1, 0, 14), nil))

	for k := uint32(1); k <= 4; k++ {
		f.add(sm2_K(k))
	}
	f.expectItem(1, 14, 0, 1)
	f.expectItem(2, 15, 1, 1)
	f.expectItem(3, 0, 2, 1)
	f.expectItem(4, 1, 3, 1)
	for k := uint32(1); k <= 4; k++ {
		expectEqual(t, Some(uint16(k)), f.set.Lookup(sm2_K(k)))
	}
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(5)))
}

// Zig: test "term51: RefCountedSet.lookupContext: a 64-bit hash plus the probe step wraps"
func TestRefCountedSet_term51_RefCountedSet_lookupContext_a_64_bit_hash_plus_the_probe_step_wraps(t *testing.T) {
	// hash = maxInt(u64): bucket 15, then (hash +% 1) = 0 -> bucket 0, ...
	f := sm2_newSet(t, 16, sm2_ctx(sm2_spec(1, 0, math.MaxUint64), nil))

	for k := uint32(1); k <= 3; k++ {
		f.add(sm2_K(k))
	}
	f.expectItem(1, 15, 0, 1)
	f.expectItem(2, 0, 1, 1)
	f.expectItem(3, 1, 2, 1)
	for k := uint32(1); k <= 3; k++ {
		expectEqual(t, Some(uint16(k)), f.set.Lookup(sm2_K(k)))
	}
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(4)))
}

// Zig: test "term51: RefCountedSet.lookupContext: zero and one slot tables"
func TestRefCountedSet_term51_RefCountedSet_lookupContext_zero_and_one_slot_tables(t *testing.T) {
	// cap 1: a 1-entry table (mask 0) with no item slots.
	f1 := sm2_newSet(t, 1, sm2_ctx(sm2_specDefault, nil))
	expectEqual(t, uint(1), f1.set.layout.table_cap)
	expectEqual(t, None[uint16](), f1.set.Lookup(sm2_K(0)))
	expectEqual(t, None[uint16](), f1.set.Lookup(sm2_K(12345)))

	// cap 0: Zig's lookup reads one table entry past the (empty) table;
	// with zero bytes there it reports "empty bucket".
	// Zig-only: that read is outside the layout. In Go a zero-capacity set
	// is empty: lookups find nothing.
	f0 := sm2_newSet(t, 0, sm2_ctx(sm2_specDefault, nil))
	expectEqual(t, None[uint16](), f0.set.Lookup(sm2_K(0)))
	expectEqual(t, None[uint16](), f0.set.Lookup(sm2_K(7)))
}

// Zig: test "term51: RefCountedSet.insert: a resident with a shorter probe length is displaced by a longer one"
func TestRefCountedSet_term51_RefCountedSet_insert_a_resident_with_a_shorter_probe_length_is_displaced_by_a_longer_one(t *testing.T) {
	var rec sm2_Rec
	// Keys 0..9 hash to 3, keys 10..19 to 4, keys 20..29 to 5, ...
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_spec(10, 1, 3), &rec))

	x := f.add(sm2_K(10)) // hash 4 -> bucket 4 psl 0
	y := f.add(sm2_K(1))  // hash 3 -> bucket 3 psl 0
	expectEqual(t, uint16(1), x)
	expectEqual(t, uint16(2), y)
	f.expectItem(x, 4, 0, 1)
	f.expectItem(y, 3, 0, 1)
	f.expectStats(2)

	// Z (hash 3) finds bucket 3 taken by an equal-psl item, moves on to
	// bucket 4 with psl 1 and displaces X (psl 0), which is pushed to
	// bucket 5 with psl 1.
	z := f.add(sm2_K(2))
	expectEqual(t, uint16(3), z)
	f.expectItem(y, 3, 0, 1)
	f.expectItem(z, 4, 1, 1)
	f.expectItem(x, 5, 1, 1)
	f.expectTable(0, 0, 0, 2, 3, 1)
	f.expectStats(1, 2)
	expectEqual(t, uint16(1), f.set.max_psl)
	expectEqual(t, Some(x), f.set.Lookup(sm2_K(10)))
	expectEqual(t, Some(y), f.set.Lookup(sm2_K(1)))
	expectEqual(t, Some(z), f.set.Lookup(sm2_K(2)))
	expectEqual(t, uint(0), rec.del_n)

	// Removing Z (dead tail, trimmed by the next add) shifts X back to
	// its home bucket and max_psl shrinks to 0; the new key goes to
	// bucket 6 and reuses id 3.
	f.set.Release(z)
	expectEqual(t, uint16(3), f.add(sm2_K(30))) // hash 6
	rec.expectDeleted(t, sm2_K(2))
	f.expectItem(x, 4, 0, 1)
	f.expectItem(y, 3, 0, 1)
	f.expectItem(3, 6, 0, 1)
	f.expectTable(0, 0, 0, 2, 1, 0, 3)
	f.expectStats(3)
	expectEqual(t, uint16(0), f.set.max_psl)
	expectEqual(t, Some(x), f.set.Lookup(sm2_K(10)))
}

// Zig: test "term51: RefCountedSet.insert: a displaced item with a higher ref count takes a position from an equal-psl item"
func TestRefCountedSet_term51_RefCountedSet_insert_a_displaced_item_with_a_higher_ref_count_takes_a_position_from_an_equal_psl_item(t *testing.T) {
	var rec sm2_Rec
	// Keys 30..39 hash to 3, keys 40..49 to 4.
	f := sm2_newSet(t, 16, sm2_ctx(sm2_spec(10, 1, 0), &rec))

	h := f.add(sm2_K(40)) // id 1, bucket 4 psl 0
	r := f.add(sm2_K(41)) // id 2, bucket 5 psl 1
	s := f.add(sm2_K(30)) // id 3, bucket 3 psl 0
	f.expectItem(h, 4, 0, 1)
	f.expectItem(r, 5, 1, 1)
	f.expectItem(s, 3, 0, 1)
	// H gets two more references.
	f.add(sm2_K(40))
	f.add(sm2_K(40))
	expectEqual(t, uint16(3), f.set.RefCount(h))

	// N (hash 3) probes: bucket 3 equal psl no swap; bucket 4 has H with
	// psl 0 < 1: N takes it, H moves on with psl 1 and meets R (psl 1,
	// 1 ref) at bucket 5: equal psl and lower ref count, so H takes the
	// bucket and R continues to bucket 6 with psl 2.
	n := f.add(sm2_K(31))
	expectEqual(t, uint16(4), n)
	f.expectItem(s, 3, 0, 1)
	f.expectItem(n, 4, 1, 1)
	f.expectItem(h, 5, 1, 3)
	f.expectItem(r, 6, 2, 1)
	f.expectTable(0, 0, 0, 3, 4, 1, 2)
	f.expectStats(1, 2, 1)
	expectEqual(t, uint16(2), f.set.max_psl)

	// Everything is still reachable. R needs 3 steps: N at i=0 (psl 1,
	// skipped), H at i=1 (compared), R at i=2 (compared).
	rec.eqls = 0
	expectEqual(t, Some(r), f.set.Lookup(sm2_K(41)))
	expectEqual(t, uint(2), rec.eqls)
	expectEqual(t, Some(h), f.set.Lookup(sm2_K(40)))
	expectEqual(t, Some(n), f.set.Lookup(sm2_K(31)))
	expectEqual(t, Some(s), f.set.Lookup(sm2_K(30)))
}

// Zig: test "term51: RefCountedSet.insert: equal ref counts do not swap equal-psl items"
func TestRefCountedSet_term51_RefCountedSet_insert_equal_ref_counts_do_not_swap_equal_psl_items(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_spec(10, 1, 0), nil))

	h := f.add(sm2_K(40))
	r := f.add(sm2_K(41))
	s := f.add(sm2_K(30))
	// Same as the previous test but H keeps a single reference.
	n := f.add(sm2_K(31))
	f.expectItem(s, 3, 0, 1)
	f.expectItem(n, 4, 1, 1)
	// R keeps bucket 5, H moves on to bucket 6 with psl 2.
	f.expectItem(r, 5, 1, 1)
	f.expectItem(h, 6, 2, 1)
	f.expectTable(0, 0, 0, 3, 4, 2, 1)
	f.expectStats(1, 2, 1)

	// A lower ref count on the held item does not swap either.
	f2 := sm2_newSet(t, 16, sm2_ctx(sm2_spec(10, 1, 0), nil))
	h2 := f2.add(sm2_K(40))
	r2 := f2.add(sm2_K(41))
	f2.add(sm2_K(30))
	f2.set.Use(r2)
	f2.set.Use(r2)
	f2.add(sm2_K(31))
	f2.expectItem(r2, 5, 1, 3)
	f2.expectItem(h2, 6, 2, 1)
}

// Zig: test "term51: RefCountedSet.insert: probing and displacement wrap around the table end"
func TestRefCountedSet_term51_RefCountedSet_insert_probing_and_displacement_wrap_around_the_table_end(t *testing.T) {
	// hash = key / 10 + 14: keys 0..9 -> 14, keys 10..19 -> 15,
	// keys 20..29 -> 16 (bucket 0), keys 30..39 -> 17 (bucket 1).
	f := sm2_newSet(t, 16, sm2_ctx(sm2_spec(10, 1, 14), nil))

	f.add(sm2_K(1)) // id 1 bucket 14 psl 0
	f.add(sm2_K(2)) // id 2 bucket 15 psl 1
	f.add(sm2_K(3)) // id 3 bucket 0 psl 2
	f.expectItem(1, 14, 0, 1)
	f.expectItem(2, 15, 1, 1)
	f.expectItem(3, 0, 2, 1)
	// A key with home bucket 15: bucket 15 is held by a psl 1 item, so
	// it ends after the cluster at bucket 1 with psl 2.
	f.add(sm2_K(10))
	f.expectItem(4, 1, 2, 1)
	// A key with home bucket 0 (hash 16): i = 0 bucket 0 psl 2 not lower,
	// i = 1 bucket 1 psl 2 not lower than 1, i = 2 bucket 2 empty.
	f.add(sm2_K(20))
	f.expectItem(5, 2, 2, 1)
	expectEqual(t, uint16(2), f.set.max_psl)
	for i, key := range []uint32{1, 2, 3, 10, 20} {
		id := i + 1
		expectEqual(t, Some(uint16(id)), f.set.Lookup(sm2_K(key)))
	}
}

// Zig: test "term51: RefCountedSet.insert: places a value with the given id and leaves the ref count at zero"
func TestRefCountedSet_term51_RefCountedSet_insert_places_a_value_with_the_given_id_and_leaves_the_ref_count_at_zero(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	id := f.set.insert(sm2_KT(5, 50), 7, f.set.context)
	expectEqual(t, uint16(7), id)
	f.expectItem(7, 5, 0, 0)
	expectEqual(t, uint32(5), f.itm(7).value.key)
	expectEqual(t, uint32(50), f.itm(7).value.tag)
	f.expectStats(1)
	expectEqual(t, uint16(0), f.set.max_psl)
	// insert alone touches neither next_id nor living.
	expectEqual(t, uint16(1), f.set.next_id)
	expectEqual(t, uint(0), f.set.Count())
	expectEqual(t, uint(0), rec.del_n)
	// Its ref is 0, so lookup does not see it.
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(5)))
	f.itm(7).meta.ref = 1
	expectEqual(t, Some(uint16(7)), f.set.Lookup(sm2_K(5)))

	// The highest id (cap - 1) is not special to insert.
	expectEqual(t, uint16(12), f.set.insert(sm2_K(6), 12, f.set.context))
	f.expectItem(12, 6, 0, 0)
}

// Zig: test "term51: RefCountedSet.insert: collisions probe forward and raise max_psl"
func TestRefCountedSet_term51_RefCountedSet_insert_collisions_probe_forward_and_raise_max_psl(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_spec(1, 0, 3), nil))

	expectEqual(t, uint16(1), f.set.insert(sm2_K(1), 1, f.set.context))
	f.itm(1).meta.ref = 1
	expectEqual(t, uint16(2), f.set.insert(sm2_K(2), 2, f.set.context))
	f.itm(2).meta.ref = 1
	expectEqual(t, uint16(3), f.set.insert(sm2_K(3), 3, f.set.context))
	f.expectItem(1, 3, 0, 1)
	f.expectItem(2, 4, 1, 1)
	f.expectItem(3, 5, 2, 0)
	f.expectStats(1, 1, 1)
	expectEqual(t, uint16(2), f.set.max_psl)
}

// Zig: test "term51: RefCountedSet.insert: a dead item in the probe path is reaped and its smaller id taken"
func TestRefCountedSet_term51_RefCountedSet_insert_a_dead_item_in_the_probe_path_is_reaped_and_its_smaller_id_taken(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(1))
	f.add(sm2_KT(2, 20))
	f.add(sm2_K(3))
	f.set.Release(2)

	// Direct insert of a value hashing to the dead item's bucket with a
	// larger requested id: the dead item's smaller id is chosen.
	id := f.set.insert(sm2_KT(2, 21), 7, f.set.context)
	expectEqual(t, uint16(2), id)
	rec.expectDeleted(t, sm2_KT(2, 20))
	f.expectItem(2, 2, 0, 0)
	expectEqual(t, uint32(21), f.itm(2).value.tag)
	f.expectHole(7)
	f.expectStats(3)
	expectEqual(t, uint16(4), f.set.next_id)
}

// Zig: test "term51: RefCountedSet.upsert: returns the existing id of a live value without changing it"
func TestRefCountedSet_term51_RefCountedSet_upsert_returns_the_existing_id_of_a_live_value_without_changing_it(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_KT(5, 50))
	f.add(sm2_KT(6, 60))

	snap := f.snapshot()
	expectEqual(t, uint16(2), f.set.upsert(sm2_KT(6, 61), 9, f.set.context))
	// Context told that the offered copy is deleted; nothing else changed
	// (no ref increment, offered id ignored).
	rec.expectDeleted(t, sm2_KT(6, 61))
	snap.expectSame(f)
	expectEqual(t, uint16(1), f.set.RefCount(2))
	f.expectHole(9)
}

// Zig: test "term51: RefCountedSet.upsert: inserts a missing value under the given id with zero refs"
func TestRefCountedSet_term51_RefCountedSet_upsert_inserts_a_missing_value_under_the_given_id_with_zero_refs(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(5))
	expectEqual(t, uint16(9), f.set.upsert(sm2_KT(6, 60), 9, f.set.context))
	expectEqual(t, uint(0), rec.del_n)
	f.expectItem(9, 6, 0, 0)
	expectEqual(t, uint32(60), f.itm(9).value.tag)
	// upsert does not touch counters.
	expectEqual(t, uint(1), f.set.Count())
	expectEqual(t, uint16(2), f.set.next_id)
}

// Zig: test "term51: RefCountedSet.upsert: a dead item with an equal value is not matched and gets reaped"
func TestRefCountedSet_term51_RefCountedSet_upsert_a_dead_item_with_an_equal_value_is_not_matched_and_gets_reaped(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_KT(1, 10))
	f.add(sm2_K(2))
	f.set.Release(1)

	// The dead id 1 is reaped from the probe path and re-used (1 < 9).
	expectEqual(t, uint16(1), f.set.upsert(sm2_KT(1, 11), 9, f.set.context))
	rec.expectDeleted(t, sm2_KT(1, 10))
	expectEqual(t, uint32(11), f.itm(1).value.tag)
	f.expectItem(1, 1, 0, 0)
	f.expectHole(9)
}

// Zig: test "term51: RefCountedSet.deleteItem: an item that is not in the table is left alone"
func TestRefCountedSet_term51_RefCountedSet_deleteItem_an_item_that_is_not_in_the_table_is_left_alone(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_K(1))
	snap := f.snapshot()
	// Slots 0, 2 and the last one are default items (bucket = maxInt).
	f.set.deleteItem(0, f.set.context)
	f.set.deleteItem(2, f.set.context)
	f.set.deleteItem(12, f.set.context)
	snap.expectSame(f)
	expectEqual(t, uint(0), rec.del_n)
}

// Zig: test "term51: RefCountedSet.deleteItem: removes the item, calls the hook and keeps counters"
func TestRefCountedSet_term51_RefCountedSet_deleteItem_removes_the_item_calls_the_hook_and_keeps_counters(t *testing.T) {
	var rec sm2_Rec
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_specDefault, &rec))

	f.add(sm2_KT(1, 10))
	f.add(sm2_K(2))
	f.expectStats(2)

	// Works on a live item too: deleteItem does not look at the ref count.
	f.set.deleteItem(1, f.set.context)
	rec.expectDeleted(t, sm2_KT(1, 10))
	f.expectHole(1)
	expectEqual(t, uint16(0), f.tbl()[1])
	f.expectStats(1)
	expectEqual(t, uint16(3), f.set.next_id)
	// living is not adjusted by deleteItem.
	expectEqual(t, uint(2), f.set.Count())
	f.expectItem(2, 2, 0, 1)

	// Deleting it again is a no-op (bucket is maxInt now).
	f.set.deleteItem(1, f.set.context)
	expectEqual(t, uint(1), rec.del_n)
}

// Zig: test "term51: RefCountedSet.deleteItem: following items with a probe length are shifted back, wrapping at the table end"
func TestRefCountedSet_term51_RefCountedSet_deleteItem_following_items_with_a_probe_length_are_shifted_back_wrapping_at_the_table_end(t *testing.T) {
	var rec sm2_Rec
	// Keys 0..9 -> hash 14, keys 30..39 -> hash 17 (bucket 1).
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_spec(10, 1, 14), &rec))

	f.add(sm2_K(1))  // A id 1 bucket 14 psl 0
	f.add(sm2_K(2))  // B id 2 bucket 15 psl 1
	f.add(sm2_K(3))  // C id 3 bucket 0 psl 2
	f.add(sm2_K(4))  // D id 4 bucket 1 psl 3
	f.add(sm2_K(30)) // E id 5 home bucket 1 -> bucket 2 psl 1
	f.expectItem(1, 14, 0, 1)
	f.expectItem(2, 15, 1, 1)
	f.expectItem(3, 0, 2, 1)
	f.expectItem(4, 1, 3, 1)
	f.expectItem(5, 2, 1, 1)
	f.expectStats(1, 2, 1, 1)
	expectEqual(t, uint16(3), f.set.max_psl)

	f.set.deleteItem(1, f.set.context)
	rec.expectDeleted(t, sm2_K(1))
	f.expectHole(1)
	// B..E each moved back by one bucket and lost one psl.
	f.expectItem(2, 14, 0, 1)
	f.expectItem(3, 15, 1, 1)
	f.expectItem(4, 0, 2, 1)
	f.expectItem(5, 1, 0, 1)
	f.expectTable(4, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 3)
	f.expectStats(2, 1, 1)
	expectEqual(t, uint16(2), f.set.max_psl)
	expectEqual(t, uint(5), f.set.Count())
	// All remaining values are still found.
	expectEqual(t, None[uint16](), f.set.Lookup(sm2_K(1)))
	expectEqual(t, Some(uint16(2)), f.set.Lookup(sm2_K(2)))
	expectEqual(t, Some(uint16(3)), f.set.Lookup(sm2_K(3)))
	expectEqual(t, Some(uint16(4)), f.set.Lookup(sm2_K(4)))
	expectEqual(t, Some(uint16(5)), f.set.Lookup(sm2_K(30)))
}

// Zig: test "term51: RefCountedSet.deleteItem: shifting stops at an item in its home bucket"
func TestRefCountedSet_term51_RefCountedSet_deleteItem_shifting_stops_at_an_item_in_its_home_bucket(t *testing.T) {
	var rec sm2_Rec
	// Keys 0..9 -> hash 3, keys 20..29 -> hash 5.
	f := sm2_newSetDel(t, 16, sm2_ctxDel(sm2_spec(10, 1, 3), &rec))

	f.add(sm2_K(1))  // A id 1 bucket 3
	f.add(sm2_K(2))  // B id 2 bucket 4 psl 1
	f.add(sm2_K(20)) // C id 3 bucket 5 psl 0
	f.expectStats(2, 1)
	expectEqual(t, uint16(1), f.set.max_psl)

	f.set.deleteItem(1, f.set.context)
	f.expectItem(2, 3, 0, 1)
	f.expectItem(3, 5, 0, 1) // psl 0: not moved
	f.expectTable(0, 0, 0, 2, 0, 3)
	f.expectStats(2)
	expectEqual(t, uint16(0), f.set.max_psl)

	// Deleting an item whose successor bucket is empty just clears it.
	f.set.deleteItem(3, f.set.context)
	f.expectTable(0, 0, 0, 2)
	f.expectStats(1)
}

// Zig: test "term51: RefCountedSet.deleteItem: max_psl only shrinks to the longest remaining probe"
func TestRefCountedSet_term51_RefCountedSet_deleteItem_max_psl_only_shrinks_to_the_longest_remaining_probe(t *testing.T) {
	f := sm2_newSet(t, 64, sm2_ctx(sm2_spec(10, 1, 3), nil))

	// Chain at hash 3 (keys 0..3: psl 0..3) and one far chain at hash 13
	// (keys 100, 101: psl 0, 1).
	for k := uint32(0); k < 4; k++ {
		f.add(sm2_K(k))
	}
	f.add(sm2_K(100))
	f.add(sm2_K(101))
	expectEqual(t, uint16(3), f.set.max_psl)
	f.expectStats(2, 2, 1, 1)

	// Delete the first of the main chain: the others shift, psl 3 -> 2.
	f.set.deleteItem(1, f.set.context)
	expectEqual(t, uint16(2), f.set.max_psl)
	f.expectStats(2, 2, 1)
	// Delete the head of the far chain: 101 shifts to psl 0; still 2 from the main chain.
	f.set.deleteItem(5, f.set.context)
	expectEqual(t, uint16(2), f.set.max_psl)
	f.expectStats(2, 1, 1)
	// Delete the end of the main chain (the psl 2 item): max_psl drops by one, to 1.
	f.set.deleteItem(4, f.set.context)
	expectEqual(t, uint16(1), f.set.max_psl)
	f.expectStats(2, 1)
	f.set.deleteItem(3, f.set.context)
	expectEqual(t, uint16(0), f.set.max_psl)
	f.expectStats(2)
}

// Zig: test "term51: RefCountedSet.assertIntegrity: is a no-op in every state"
func TestRefCountedSet_term51_RefCountedSet_assertIntegrity_is_a_no_op_in_every_state(t *testing.T) {
	f := sm2_newSet(t, 16, sm2_ctx(sm2_spec(1, 0, 3), nil))

	f.add(sm2_K(1))
	f.add(sm2_K(2))
	{
		snap := f.snapshot()
		f.set.assertIntegrity(f.set.context)
		snap.expectSame(f)
	}

	// The checks are compiled out (`if (false and ...)`), so even an
	// inconsistent state does not trip anything.
	f.set.psl_stats[7] = 9
	f.tbl()[9] = 2
	f.itm(1).meta.bucket = 12
	{
		snap := f.snapshot()
		f.set.assertIntegrity(f.set.context)
		snap.expectSame(f)
	}
}

// Zig: test "term51: RefCountedSet.add: style set separates styles by every field and merges equal ones"
func TestRefCountedSet_term51_RefCountedSet_add_style_set_separates_styles_by_every_field_and_merges_equal_ones(t *testing.T) {
	f := sm2_newStyleSet(t, 32)

	styles := []Style{
		{}, // the default style is an ordinary value to the set
		{flags: styleFlags{bold: true}},
		{flags: styleFlags{italic: true}},
		{flags: styleFlags{underline: SGRAttributeUnderlineSingle}},
		{flags: styleFlags{underline: SGRAttributeUnderlineCurly}},
		{fg_color: sm2_palette(1)},
		{bg_color: sm2_palette(1)},
		{underline_color: sm2_palette(1)},
		{fg_color: sm2_rgb(1, 2, 3)},
		{fg_color: sm2_rgb(1, 2, 4)},
		{bg_color: sm2_rgb(1, 2, 3)},
		{underline_color: sm2_rgb(1, 2, 3)},
		{fg_color: sm2_palette(0)}, // palette 0 differs from none
	}
	for i, style := range styles {
		expectEqual(t, uint16(i+1), f.add(style))
	}
	expectEqual(t, uint(len(styles)), f.set.Count())

	// Every style is found under its own id; adding again counts a reference.
	for i, style := range styles {
		i := i + 1
		expectEqual(t, Some(uint16(i)), f.set.Lookup(style))
		expectEqual(t, uint16(i), f.add(style))
		expectEqual(t, uint16(2), f.set.RefCount(uint16(i)))
		expect(t, f.set.Get(uint16(i)).Eql(style), "f.set.get(f.buf, @intCast(i)).eql(style)")
	}
	expectEqual(t, uint(len(styles)), f.set.Count())
	expectEqual(t, None[uint16](), f.set.Lookup(Style{flags: styleFlags{blink: true}}))
}

// Zig: test "term51: RefCountedSet.release: style set entries stay until reused"
func TestRefCountedSet_term51_RefCountedSet_release_style_set_entries_stay_until_reused(t *testing.T) {
	f := sm2_newStyleSet(t, 16)

	a := Style{flags: styleFlags{bold: true}}
	b := Style{flags: styleFlags{italic: true}}
	c := Style{flags: styleFlags{faint: true}}
	d := Style{flags: styleFlags{inverse: true}}
	expectEqual(t, uint16(1), f.add(a))
	expectEqual(t, uint16(2), f.add(b))
	expectEqual(t, uint16(3), f.add(c))

	// Releasing a non-tail style leaves it resurrectable with the same id.
	f.set.Release(1)
	expectEqual(t, uint(2), f.set.Count())
	expectEqual(t, None[uint16](), f.set.Lookup(a))
	expectEqual(t, uint16(1), f.add(a))
	expectEqual(t, uint(3), f.set.Count())
	expectEqual(t, uint16(1), f.set.RefCount(1))
	expectEqual(t, uint16(4), f.set.next_id)

	// Releasing the tail style frees its id for the next new style.
	f.set.Release(3)
	expectEqual(t, uint16(3), f.add(d))
	expect(t, f.set.Get(3).flags.inverse, "f.set.get(f.buf, 3).flags.inverse")
	expectEqual(t, None[uint16](), f.set.Lookup(c))
	expectEqual(t, uint16(4), f.set.next_id)
}

// Zig: test "term51: RefCountedSet.add: style set holds exactly capacity minus one items"
func TestRefCountedSet_term51_RefCountedSet_add_style_set_holds_exactly_capacity_minus_one_items(t *testing.T) {
	// 16 -> 13 slots, id 0 reserved -> 12 styles.
	f := sm2_newStyleSet(t, 16)

	for i := uint8(0); i < 12; i++ {
		style := Style{fg_color: sm2_palette(i)}
		expectEqual(t, uint16(i)+1, f.add(style))
	}
	expectEqual(t, uint(12), f.set.Count())
	extra := Style{fg_color: sm2_palette(12)}
	sm2_expectError(t, ErrOutOfMemory, f.addErr(extra))
	expectEqual(t, uint(12), f.set.Count())
	expectEqual(t, None[uint16](), f.set.Lookup(extra))

	// Dead styles below the rehash threshold turn it into NeedsRehash.
	f.set.Release(1)
	sm2_expectError(t, ErrOutOfMemory, f.addErr(extra))
	f.set.Release(2)
	sm2_expectError(t, ErrNeedsRehash, f.addErr(extra))
	// An existing style can still be added.
	expectEqual(t, uint16(5), f.add(Style{fg_color: sm2_palette(4)}))
}

// Zig: test "term51: RefCountedSet.add: style set maximum number of styles for the page capacity"
func TestRefCountedSet_term51_RefCountedSet_add_style_set_maximum_number_of_styles_for_the_page_capacity(t *testing.T) {
	// The capacity upstream's "Set capacities" test makes sure is available.
	f := sm2_newStyleSet(t, 16384)
	expectEqual(t, uint(13312), f.set.layout.cap)

	const max_items uint32 = 13311
	for i := uint32(0); i < max_items; i++ {
		style := Style{fg_color: sm2_rgb(uint8(i), uint8(i>>8), 9)}
		expectEqual(t, uint16(i+1), f.add(style))
	}
	expectEqual(t, uint(max_items), f.set.Count())
	expectEqual(t, uint16(13312), f.set.next_id)
	extra := Style{fg_color: sm2_rgb(255, 255, 9)}
	sm2_expectError(t, ErrOutOfMemory, f.addErr(extra))
	expectEqual(t, uint(max_items), f.set.Count())
	// Lookups of the first and last style still work.
	expectEqual(t, Some(uint16(1)), f.set.Lookup(Style{fg_color: sm2_rgb(0, 0, 9)}))
	expectEqual(t, Some(uint16(13311)), f.set.Lookup(Style{fg_color: sm2_rgb(uint8(13310&0xff), uint8(13310>>8), 9)}))
}
