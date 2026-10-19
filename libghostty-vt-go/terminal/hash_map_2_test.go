// Tests translated one for one from the second batch of the `term51:` tests of
// src/terminal/hash_map.zig (banner "term51: additional tests (AG)"): the
// direct tests of the functions that nothing in the library calls
// (HashMapUnmanaged.FieldIterator, keyIterator, valueIterator,
// fetchPutContext, fetchPutAssumeCapacityContext, fetchRemoveContext,
// fetchRemoveAdapted, getKeyPtrContext, getKeyPtrAdapted, getKey,
// getKeyContext, getKeyAdapted, getOrPutAdapted, getOrPutAssumeCapacity,
// getOrPutValueContext). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. Tests that expect a panic carry "// Go-only: panic
// test:" (Zig tests cannot test an assert or a safety-checked panic: the test
// would abort).
//
// Option A (keep upstream's byte-offset bookkeeping on Go slices): confirmed by the project owner 2026-10-04.
//
// Translation notes: the conventions and fixtures of hash_map_test.go apply
// (hmBox, hmK, hmCtx, hmSl, hmDesc, hmExpectSl, hmCtl, hmSlot, hmUsedCtl,
// hmCountCtl, the key chains and hmFull16 are used from there). In addition:
//   - `term51AGLooseCtx` (keys equal when offset >> 4 agrees, so that the
//     stored key and the key looked up differ) is hm2LooseCtx, its box is
//     hm2LBox (a map of HyperlinkId values like term51XHMap).
//   - The `...Adapted` functions take the map's own context type in Go, so
//     the raw u32 key of term51XRawCtx is the key type and hashMapAutoContext
//     stands for term51XRawCtx (as in hash_map_test.go).
//   - `@intFromPtr(a) == @intFromPtr(b)` is `a == b` of Go pointers.
//   - `try m.fetchPutContext(...)` returns the pair (Optional, error) in Go;
//     hm2FetchPut and hm2GetOrPutValue unwrap the error with zigTry.
//   - Zig-only: `defer box.deinit()` is dropped in every test.

package terminal

import (
	"math"
	"testing"
)

// hm2ExpectPanic runs f, which must panic. A panic with the wireframe's "not
// implemented" value does not count: it only means the function under test
// does not exist yet. The setup of the map is done outside f.
func hm2ExpectPanic(t *testing.T, f func()) {
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

// hm2LooseCtx is term51AGLooseCtx: two keys are equal when their offsets agree
// above the low four bits (offset >> 4).
type hm2LooseCtx struct{}

func (hm2LooseCtx) Hash(k SizeOffsetInt) uint64 { return hmWyhash(0, hmLE32(k>>4)) }

func (hm2LooseCtx) Eql(a SizeOffsetInt, b SizeOffsetInt) bool { return a>>4 == b>>4 }

// hm2Loose is term51AGLoose.
var hm2Loose = hm2LooseCtx{}

// hm2LBox is term51AGLBox: a map of HyperlinkId values under hm2LooseCtx.
type hm2LBox struct {
	offset_map HashMapOffsetHashMap[SizeOffsetInt, HyperlinkID, hm2LooseCtx]
}

func hm2LBoxInit(cap uint32) hm2LBox {
	l := HashMapOffsetHashMapLayoutOf(cap, hyperlinkMapEntrySizes)
	return hm2LBox{offset_map: HashMapOffsetHashMapInit[SizeOffsetInt, HyperlinkID, hm2LooseCtx](l)}
}

func (self *hm2LBox) map_() hashMapUnmanaged[SizeOffsetInt, HyperlinkID, hm2LooseCtx] {
	return self.offset_map.Map()
}

// hm2Order16 is term51AGOrder16: the keys of hmFull16 in slot order.
var hm2Order16 = [16]uint32{65640, 65656, 65568, 65576, 65616, 65592, 65600, 65536, 65552, 65624, 65648, 65544, 65584, 65560, 65608, 65632}

// hm2Fill16 is term51AGFill16.
func hm2Fill16(t *testing.T, m hmHU) {
	t.Helper()
	for i, k := range hmFull16 {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i)))
	}
}

// hm2Snap is term51AGSnap: the control bytes of a map.
func hm2Snap(m hmHU) []uint8 {
	out := make([]uint8, m.Capacity())
	for i := range out {
		out[i] = hmCtl(m, uint(i))
	}
	return out
}

// hm2FetchPut is `try m.fetchPutContext(key, value, ctx)`.
func hm2FetchPut[V any, C HashMapContext[SizeOffsetInt]](t *testing.T, m hashMapUnmanaged[SizeOffsetInt, V, C], key SizeOffsetInt, value V, ctx C) Optional[HashMapKV[SizeOffsetInt, V]] {
	t.Helper()
	r, err := m.FetchPutContext(key, value, ctx)
	zigTry(t, err)
	return r
}

// hm2GetOrPutValue is `try m.getOrPutValueContext(key, value, ctx)`.
func hm2GetOrPutValue[V any, C HashMapContext[SizeOffsetInt]](t *testing.T, m hashMapUnmanaged[SizeOffsetInt, V, C], key SizeOffsetInt, value V, ctx C) HashMapEntry[SizeOffsetInt, V] {
	t.Helper()
	r, err := m.GetOrPutValueContext(key, value, ctx)
	zigTry(t, err)
	return r
}

// Zig: test "term51: HashMapUnmanaged.FieldIterator: yields the items of used slots in order and counts len down"
func TestHashMap_term51_HashMapUnmanaged_FieldIterator_yields_the_items_of_used_slots_in_order_and_counts_len_down(t *testing.T) {
	items := [8]uint32{10, 11, 12, 13, 14, 15, 16, 17}
	// free, used, tombstone, used (fp 0), free, used (fp 0x7f), tombstone, used (fp 1)
	meta := [8]hashMapMetadata{
		hmMeta(0x00), hmMeta(0x85), hmMeta(0x01), hmMeta(0x80),
		hmMeta(0x00), hmMeta(0xff), hmMeta(0x01), hmMeta(0x81),
	}
	it := hashMapFieldIterator[uint32]{len: uint(len(meta)), metadata: meta[:], items: items[:]}
	slots := [...]uint{1, 3, 5, 7}
	for _, s := range slots {
		p := it.Next()
		expect(t, p == &items[s], "@intFromPtr(&items[s]) == @intFromPtr(p)")
		expectEqual(t, uint32(10)+uint32(s), *p)
		// The iterator is positioned one slot past the one it returned.
		expectEqual(t, uint(len(meta))-(s+1), it.len)
		// Zig-only: @intFromPtr(&items) + 4 * (s + 1) == @intFromPtr(it.items) and
		// @intFromPtr(&meta) + (s + 1) == @intFromPtr(it.metadata): the advanced
		// pointers have no Go counterpart (the Go fields are slices).
	}
	expect(t, it.Next() == nil, "it.next() == null")
	expectEqual(t, uint(0), it.len)
	expect(t, it.Next() == nil, "it.next() == null")
	expectEqual(t, uint(0), it.len)
	// Zig-only: @intFromPtr(&items) + 32 == @intFromPtr(it.items).
}

// Zig: test "term51: HashMapUnmanaged.FieldIterator.next: len limits the scan, unused ranges and empty ranges yield nothing"
func TestHashMap_term51_HashMapUnmanaged_FieldIterator_next_len_limits_the_scan_unused_ranges_and_empty_ranges_yield_nothing(t *testing.T) {
	items := [4]uint16{1, 2, 3, 4}
	used := [4]hashMapMetadata{hmMeta(0x80), hmMeta(0x81), hmMeta(0x82), hmMeta(0xff)}
	unused := [4]hashMapMetadata{hmMeta(0x00), hmMeta(0x01), hmMeta(0x01), hmMeta(0x00)}

	// len shorter than the arrays: the slot at len is never looked at.
	{
		it := hashMapFieldIterator[uint16]{len: 3, metadata: used[:], items: items[:]}
		for s := range 3 {
			expect(t, it.Next() == &items[s], "@intFromPtr(&items[s]) == @intFromPtr(it.next().?)")
		}
		expect(t, it.Next() == nil, "it.next() == null")
		expectEqual(t, uint(0), it.len)
	}
	// len zero reads nothing (a map without metadata makes this iterator).
	{
		it := hashMapFieldIterator[uint16]{len: 0}
		expect(t, it.Next() == nil, "it.next() == null")
		expectEqual(t, uint(0), it.len)
	}
	// Only free slots and tombstones: nothing, and the whole range is consumed.
	{
		it := hashMapFieldIterator[uint16]{len: 4, metadata: unused[:], items: items[:]}
		expect(t, it.Next() == nil, "it.next() == null")
		expectEqual(t, uint(0), it.len)
		expect(t, it.Next() == nil, "it.next() == null")
	}
	// A copy of an iterator keeps its own position.
	{
		it := hashMapFieldIterator[uint16]{len: 4, metadata: used[:], items: items[:]}
		expect(t, it.Next() == &items[0], "@intFromPtr(&items[0]) == @intFromPtr(it.next().?)")
		cp := it
		expect(t, cp.Next() == &items[1], "@intFromPtr(&items[1]) == @intFromPtr(copy.next().?)")
		expect(t, cp.Next() == &items[2], "@intFromPtr(&items[2]) == @intFromPtr(copy.next().?)")
		expectEqual(t, uint(1), cp.len)
		expectEqual(t, uint(3), it.len)
		expect(t, it.Next() == &items[1], "@intFromPtr(&items[1]) == @intFromPtr(it.next().?)")
	}
	// The pointers write through to the items.
	{
		it := hashMapFieldIterator[uint16]{len: 4, metadata: used[:], items: items[:]}
		for p := it.Next(); p != nil; p = it.Next() {
			*p *= 10
		}
		expectEqualSlices(t, []uint16{10, 20, 30, 40}, items[:])
	}
}

// Zig: test "term51: HashMapUnmanaged.keyIterator: a map without metadata and a zero-capacity map yield nothing"
func TestHashMap_term51_HashMapUnmanaged_keyIterator_a_map_without_metadata_and_a_zero_capacity_map_yield_nothing(t *testing.T) {
	var m0 hmHU
	it0 := m0.KeyIterator()
	expectEqual(t, uint(0), it0.len)
	expect(t, it0.Next() == nil, "it0.next() == null")
	expectEqual(t, uint(0), it0.len)

	z := hmHBoxInit(0)
	zm := z.map_()
	itz := zm.KeyIterator()
	expectEqual(t, uint(0), itz.len)
	expect(t, itz.Next() == nil, "itz.next() == null")
}

// Zig: test "term51: HashMapUnmanaged.keyIterator: an empty map is scanned over its whole capacity"
func TestHashMap_term51_HashMapUnmanaged_keyIterator_an_empty_map_is_scanned_over_its_whole_capacity(t *testing.T) {
	for _, cap := range []uint32{1, 2, 8, 16, 64} {
		box := hmHBoxInit(cap)
		m := box.map_()
		it := m.KeyIterator()
		expectEqual(t, uint(cap), it.len)
		expect(t, &it.items[0] == &m.keys()[0], "@intFromPtr(m.keys()) == @intFromPtr(it.items)")
		expect(t, &it.metadata[0] == &m.metadata[0], "@intFromPtr(m.metadata.?) == @intFromPtr(it.metadata)")
		expect(t, it.Next() == nil, "it.next() == null")
		expectEqual(t, uint(0), it.len)
		expect(t, it.Next() == nil, "it.next() == null")
	}
}

// Zig: test "term51: HashMapUnmanaged.keyIterator: keys in slot order with pointers to the key slots on a full map"
func TestHashMap_term51_HashMapUnmanaged_keyIterator_keys_in_slot_order_with_pointers_to_the_key_slots_on_a_full_map(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	hm2Fill16(t, m)
	it := m.KeyIterator()
	expectEqual(t, uint(16), it.len)
	for s, k := range hm2Order16 {
		p := it.Next()
		expect(t, p != nil, "it.next().? != null")
		expectEqual(t, k, *p)
		expect(t, &m.keys()[s] == p, "@intFromPtr(&m.keys()[s]) == @intFromPtr(p)")
		expectEqual(t, uint(16-(s+1)), it.len)
	}
	expect(t, it.Next() == nil, "it.next() == null")
	expectEqual(t, uint(0), it.len)

	// Restart: a new iterator starts over, one in the middle is independent.
	again := m.KeyIterator()
	expectEqual(t, uint32(65640), *again.Next())
	cp := again
	expectEqual(t, uint32(65656), *again.Next())
	expectEqual(t, uint32(65656), *cp.Next())
	n := uint(2)
	for p := again.Next(); p != nil; p = again.Next() {
		n++
	}
	expectEqual(t, uint(16), n)
	expectEqual(t, m.Count(), uint32(n))
}

// Zig: test "term51: HashMapUnmanaged.keyIterator: tombstones are skipped and reinsertion changes the order"
func TestHashMap_term51_HashMapUnmanaged_keyIterator_tombstones_are_skipped_and_reinsertion_changes_the_order(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain14
	for _, k := range c[0:4] {
		zigTry(t, m.Put(hmK(k), 1))
	}
	// Wrapped chain: slots 14,15,0,1 are visited 0,1,14,15.
	{
		it := m.KeyIterator()
		want := [...]uint32{c[2], c[3], c[0], c[1]}
		lens := [...]uint{15, 14, 1, 0}
		for i, k := range want {
			expectEqual(t, k, *it.Next())
			expectEqual(t, lens[i], it.len)
		}
		expect(t, it.Next() == nil, "it.next() == null")
	}
	_ = m.Remove(hmK(c[1]))
	_ = m.Remove(hmK(c[2]))
	{
		it := m.KeyIterator()
		expectEqual(t, c[3], *it.Next())
		expectEqual(t, uint(14), it.len)
		expectEqual(t, c[0], *it.Next())
		expect(t, it.Next() == nil, "it.next() == null")
	}
	// c[4] recycles slot 15, c[5] slot 0.
	zigTry(t, m.Put(hmK(c[4]), 1))
	zigTry(t, m.Put(hmK(c[5]), 1))
	{
		it := m.KeyIterator()
		want := [...]uint32{c[5], c[3], c[0], c[4]}
		for _, k := range want {
			expectEqual(t, k, *it.Next())
		}
		expect(t, it.Next() == nil, "it.next() == null")
	}
	// Only tombstones: nothing, though the capacity is still scanned.
	for _, k := range c[3:6] {
		_ = m.Remove(hmK(k))
	}
	_ = m.Remove(hmK(c[0]))
	expectEqual(t, uint32(0), m.Count())
	{
		it := m.KeyIterator()
		expectEqual(t, uint(16), it.len)
		expect(t, it.Next() == nil, "it.next() == null")
		expectEqual(t, uint(0), it.len)
	}
	// After clearRetainingCapacity as well.
	zigTry(t, m.Put(hmK(c[0]), 1))
	m.ClearRetainingCapacity()
	it := m.KeyIterator()
	expect(t, it.Next() == nil, "it.next() == null")
}

// Zig: test "term51: HashMapUnmanaged.keyIterator: shares the map's storage with later changes"
func TestHashMap_term51_HashMapUnmanaged_keyIterator_shares_the_map_s_storage_with_later_changes(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	it := m.KeyIterator()
	// Entries added after the iterator was made are seen.
	zigTry(t, m.Put(hmK(c[0]), 1))
	zigTry(t, m.Put(hmK(c[1]), 2))
	p := it.Next()
	expect(t, p != nil, "it.next().? != null")
	expect(t, &m.keys()[5] == p, "@intFromPtr(&m.keys()[5]) == @intFromPtr(p)")
	expectEqual(t, c[0], *p)
	// A copy continues from the same point.
	cp := it
	expectEqual(t, c[1], *it.Next())
	expectEqual(t, c[1], *cp.Next())
	expect(t, it.Next() == nil, "it.next() == null")
	expect(t, cp.Next() == nil, "copy.next() == null")
	// Writing through the pointer writes the key slot.
	*p = hmK(7777)
	expectEqual(t, uint32(7777), m.keys()[5])
	*p = hmK(c[0])
	expect(t, m.Contains(hmK(c[0])), "m.contains(term51XK(c[0]))")
	// A new iterator starts again at slot 0.
	fresh := m.KeyIterator()
	expectEqual(t, c[0], *fresh.Next())
}

// Zig: test "term51: HashMapUnmanaged.valueIterator: a map without metadata and a zero-capacity map yield nothing"
func TestHashMap_term51_HashMapUnmanaged_valueIterator_a_map_without_metadata_and_a_zero_capacity_map_yield_nothing(t *testing.T) {
	var m0 hmHU
	it0 := m0.ValueIterator()
	expectEqual(t, uint(0), it0.len)
	expect(t, it0.Next() == nil, "it0.next() == null")

	z := hmGBoxInit(0)
	zm := z.map_()
	itz := zm.ValueIterator()
	expectEqual(t, uint(0), itz.len)
	expect(t, itz.Next() == nil, "itz.next() == null")
}

// Zig: test "term51: HashMapUnmanaged.valueIterator: an empty map is scanned over its whole capacity"
func TestHashMap_term51_HashMapUnmanaged_valueIterator_an_empty_map_is_scanned_over_its_whole_capacity(t *testing.T) {
	for _, cap := range []uint32{1, 2, 8, 16, 64} {
		box := hmGBoxInit(cap)
		m := box.map_()
		it := m.ValueIterator()
		expectEqual(t, uint(cap), it.len)
		expect(t, &it.items[0] == &m.values()[0], "@intFromPtr(m.values()) == @intFromPtr(it.items)")
		expect(t, &it.metadata[0] == &m.metadata[0], "@intFromPtr(m.metadata.?) == @intFromPtr(it.metadata)")
		expect(t, it.Next() == nil, "it.next() == null")
		expectEqual(t, uint(0), it.len)
	}
}

// Zig: test "term51: HashMapUnmanaged.valueIterator: values in slot order, pointers into the value slots and write through"
func TestHashMap_term51_HashMapUnmanaged_valueIterator_values_in_slot_order_pointers_into_the_value_slots_and_write_through(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	for i, k := range hmFull16 {
		zigTry(t, m.Put(hmK(k), hmSl(uint32(i), uint(i+1))))
	}
	it := m.ValueIterator()
	for s, k := range hm2Order16 {
		p := it.Next()
		expect(t, p != nil, "it.next().? != null")
		expect(t, &m.values()[s] == p, "@intFromPtr(&m.values()[s]) == @intFromPtr(p)")
		hmExpectSl(t, zigUnwrap(t, m.Get(hmK(k))), *p)
		expect(t, m.GetPtr(hmK(k)) == p, "@intFromPtr(m.getPtr(term51XK(k)).?) == @intFromPtr(p)")
		expectEqual(t, uint(16-(s+1)), it.len)
	}
	expect(t, it.Next() == nil, "it.next() == null")

	// The key and value iterators run in lockstep.
	ki := m.KeyIterator()
	vi := m.ValueIterator()
	n := uint(0)
	for kp := ki.Next(); kp != nil; kp = ki.Next() {
		vp := vi.Next()
		expect(t, vp != nil, "vi.next().? != null")
		expect(t, m.GetPtr(*kp) == vp, "@intFromPtr(m.getPtr(kp.*).?) == @intFromPtr(vp)")
		n++
	}
	expectEqual(t, uint(16), n)
	expect(t, vi.Next() == nil, "vi.next() == null")

	// Writing through the iterator changes the map's values.
	wi := m.ValueIterator()
	for p := wi.Next(); p != nil; p = wi.Next() {
		*p = hmSl(hmDesc(*p).offset, hmDesc(*p).len+100) // p.len += 100
	}
	for i, k := range hmFull16 {
		hmExpectSl(t, hmSl(uint32(i), uint(i+101)), zigUnwrap(t, m.Get(hmK(k))))
	}
}

// Zig: test "term51: HashMapUnmanaged.valueIterator: removed entries are skipped"
func TestHashMap_term51_HashMapUnmanaged_valueIterator_removed_entries_are_skipped(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	hm2Fill16(t, m)
	// Remove the entries with an odd index.
	for i, k := range hmFull16 {
		if i%2 == 1 {
			expect(t, m.Remove(hmK(k)), "m.remove(term51XK(k))")
		}
	}
	expectEqual(t, uint32(8), m.Count())
	it := m.ValueIterator()
	seen := 0
	for s := range 16 {
		i := -1
		for j, sl := range hmFull16Slots {
			if sl == uint(s) {
				i = j
			}
		}
		expect(t, i >= 0, "std.mem.indexOfScalar(usize, &term51XFull16Slots, s).? != null")
		if i%2 == 0 {
			p := it.Next()
			expect(t, p != nil, "it.next().? != null")
			expectEqual(t, uint16(i), *p)
			expect(t, &m.values()[s] == p, "@intFromPtr(&m.values()[s]) == @intFromPtr(p)")
			seen++
		}
	}
	expectEqual(t, 8, seen)
	expect(t, it.Next() == nil, "it.next() == null")
	expectEqual(t, uint(0), it.len)

	// Cleared: nothing, the capacity is still scanned.
	m.ClearRetainingCapacity()
	empty := m.ValueIterator()
	expectEqual(t, uint(16), empty.len)
	expect(t, empty.Next() == nil, "empty.next() == null")
}

// Zig: test "term51: HashMapUnmanaged.fetchPutContext: a new key returns null and is inserted"
func TestHashMap_term51_HashMapUnmanaged_fetchPutContext_a_new_key_returns_null_and_is_inserted(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c[0:6] {
		expect(t, !hm2FetchPut(t, m, hmK(k), HyperlinkID(i+10), hmCtx).Valid, "(try m.fetchPutContext(term51XK(k), @intCast(i + 10), term51XCtx)) == null")
		expectEqual(t, uint32(i+1), m.Count())
		expectEqual(t, uint(5+i), zigUnwrap(t, hmSlot(m, k)))
		expectEqual(t, k, m.keys()[5+i])
		expectEqual(t, uint16(i+10), m.values()[5+i])
		expectEqual(t, hmUsedCtl(k), hmCtl(m, uint(5+i)))
	}
	// The wrapper without a context.
	r, err := m.FetchPut(hmK(c[6]), 99)
	zigTry(t, err)
	expect(t, !r.Valid, "(try m.fetchPut(term51XK(c[6]), 99)) == null")
	expectEqual(t, uint(11), zigUnwrap(t, hmSlot(m, c[6])))
	expectEqual(t, uint16(99), zigUnwrap(t, m.Get(hmK(c[6]))))
	expectEqual(t, uint32(7), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.fetchPutContext: an existing key returns its old key and value and replaces the value"
func TestHashMap_term51_HashMapUnmanaged_fetchPutContext_an_existing_key_returns_its_old_key_and_value_and_replaces_the_value(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	for i := range 5 {
		zigTry(t, m.Put(hmK(uint32(4096+8*i)), hmSl(uint32(i), uint(i+1))))
	}
	key := hmK(4096 + 16)
	slot := zigUnwrap(t, hmSlot(m, 4096+16))

	old := zigUnwrap(t, hm2FetchPut(t, m, key, hmSl(100, 7), hmCtx))
	expectEqual(t, uint32(4096+16), old.key)
	hmExpectSl(t, hmSl(2, 3), old.value)
	hmExpectSl(t, hmSl(100, 7), zigUnwrap(t, m.Get(key)))
	expectEqual(t, uint32(5), m.Count())
	expectEqual(t, slot, zigUnwrap(t, hmSlot(m, 4096+16)))

	// A second replacement returns the first replacement.
	old2 := zigUnwrap(t, hm2FetchPut(t, m, key, hmSl(200, 9), hmCtx))
	hmExpectSl(t, hmSl(100, 7), old2.value)
	hmExpectSl(t, hmSl(200, 9), zigUnwrap(t, m.Get(key)))

	// The wrapper without a context, and a zero length slice.
	r3, err := m.FetchPut(key, hmSl(0, 0))
	zigTry(t, err)
	old3 := zigUnwrap(t, r3)
	hmExpectSl(t, hmSl(200, 9), old3.value)
	hmExpectSl(t, hmSl(0, 0), zigUnwrap(t, m.Get(key)))

	// The other entries are untouched.
	for i := range 5 {
		if i == 2 {
			continue
		}
		hmExpectSl(t, hmSl(uint32(i), uint(i+1)), zigUnwrap(t, m.Get(hmK(uint32(4096+8*i)))))
	}
	expectEqual(t, uint32(5), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.fetchPutContext: a full map still replaces existing keys and refuses new ones"
func TestHashMap_term51_HashMapUnmanaged_fetchPutContext_a_full_map_still_replaces_existing_keys_and_refuses_new_ones(t *testing.T) {
	box := hmHBoxInit(4)
	m := box.map_()
	keys := [...]uint32{8, 16, 24, 32}
	for i, k := range keys {
		expect(t, !hm2FetchPut(t, m, hmK(k), HyperlinkID(i+1), hmCtx).Valid, "(try m.fetchPutContext(term51XK(k), @intCast(i + 1), term51XCtx)) == null")
	}
	expectEqual(t, uint32(4), m.Count())
	for i, k := range keys {
		old := zigUnwrap(t, hm2FetchPut(t, m, hmK(k), HyperlinkID(i+50), hmCtx))
		expectEqual(t, k, old.key)
		expectEqual(t, uint16(i+1), old.value)
	}
	expectEqual(t, uint32(4), m.Count())
	_, err := m.FetchPutContext(hmK(40), 1, hmCtx)
	hmExpectError(t, ErrOutOfMemory, err)
	_, err = m.FetchPut(hmK(48), 1)
	hmExpectError(t, ErrOutOfMemory, err)
	// Nothing changed.
	expectEqual(t, uint32(4), m.Count())
	for i, k := range keys {
		expectEqual(t, uint16(i+50), zigUnwrap(t, m.Get(hmK(k))))
	}
	expect(t, !m.Get(hmK(40)).Valid, "m.get(term51XK(40)) == null")
	expect(t, !m.Get(hmK(48)).Valid, "m.get(term51XK(48)) == null")

	// Capacity one.
	one := hmHBoxInit(1)
	om := one.map_()
	expect(t, !hm2FetchPut(t, om, hmK(8), 1, hmCtx).Valid, "(try om.fetchPutContext(term51XK(8), 1, term51XCtx)) == null")
	expectEqual(t, uint16(1), zigUnwrap(t, hm2FetchPut(t, om, hmK(8), 2, hmCtx)).value)
	_, err = om.FetchPutContext(hmK(16), 3, hmCtx)
	hmExpectError(t, ErrOutOfMemory, err)
	expectEqual(t, uint16(2), zigUnwrap(t, om.Get(hmK(8))))

	// Capacity zero.
	z := hmHBoxInit(0)
	zm := z.map_()
	_, err = zm.FetchPutContext(hmK(8), 1, hmCtx)
	hmExpectError(t, ErrOutOfMemory, err)
	expectEqual(t, uint32(0), zm.Count())
}

// Zig: test "term51: HashMapUnmanaged.fetchPutContext: tombstones are recycled and keys behind them are found"
func TestHashMap_term51_HashMapUnmanaged_fetchPutContext_tombstones_are_recycled_and_keys_behind_them_are_found(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c[0:6] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i)))
	}
	_ = m.Remove(hmK(c[1]))
	_ = m.Remove(hmK(c[3]))
	expectEqual(t, uint32(4), m.Count())
	expectEqual(t, uint8(1), hmCtl(m, 6))
	expectEqual(t, uint8(1), hmCtl(m, 8))

	// A new key takes the first tombstone in probe order.
	expect(t, !hm2FetchPut(t, m, hmK(c[6]), 60, hmCtx).Valid, "(try m.fetchPutContext(term51XK(c[6]), 60, term51XCtx)) == null")
	expectEqual(t, uint(6), zigUnwrap(t, hmSlot(m, c[6])))
	expectEqual(t, hmUsedCtl(c[6]), hmCtl(m, 6))
	expectEqual(t, uint8(1), hmCtl(m, 8))
	expectEqual(t, uint32(5), m.Count())

	// A key behind a tombstone is found.
	old := zigUnwrap(t, hm2FetchPut(t, m, hmK(c[4]), 40, hmCtx))
	expectEqual(t, uint16(4), old.value)
	expectEqual(t, uint(9), zigUnwrap(t, hmSlot(m, c[4])))

	// A removed key is new again (its old value is gone) and takes the other tombstone.
	expect(t, !hm2FetchPut(t, m, hmK(c[1]), 11, hmCtx).Valid, "(try m.fetchPutContext(term51XK(c[1]), 11, term51XCtx)) == null")
	expectEqual(t, uint(8), zigUnwrap(t, hmSlot(m, c[1])))
	expectEqual(t, uint32(6), m.Count())
	expect(t, !m.Get(hmK(c[3])).Valid, "m.get(term51XK(c[3])) == null")

	// A map of nothing but tombstones accepts new keys until it is full.
	b4 := hmHBoxInit(4)
	m4 := b4.map_()
	keys := [...]uint32{8, 16, 24, 32}
	for _, k := range keys {
		zigTry(t, m4.Put(hmK(k), 1))
	}
	for _, k := range keys {
		expect(t, m4.Remove(hmK(k)), "m4.remove(term51XK(k))")
	}
	expectEqual(t, uint32(0), m4.Count())
	expectEqual(t, uint(4), hmCountCtl(m4, 1))
	home := uint(hmCtx.Hash(hmK(40)) & 3)
	expect(t, !hm2FetchPut(t, m4, hmK(40), 7, hmCtx).Valid, "(try m4.fetchPutContext(term51XK(40), 7, term51XCtx)) == null")
	expectEqual(t, home, zigUnwrap(t, hmSlot(m4, 40)))
	for _, k := range [...]uint32{48, 56, 64} {
		expect(t, !hm2FetchPut(t, m4, hmK(k), 7, hmCtx).Valid, "(try m4.fetchPutContext(term51XK(k), 7, term51XCtx)) == null")
	}
	expectEqual(t, uint32(4), m4.Count())
	expectEqual(t, uint(0), hmCountCtl(m4, 1))
	_, err := m4.FetchPutContext(hmK(72), 7, hmCtx)
	hmExpectError(t, ErrOutOfMemory, err)
}

// Zig: test "term51: HashMapUnmanaged.fetchPutContext: the stored key is kept, a new entry stores the given key"
func TestHashMap_term51_HashMapUnmanaged_fetchPutContext_the_stored_key_is_kept_a_new_entry_stores_the_given_key(t *testing.T) {
	box := hm2LBoxInit(16)
	m := box.map_()
	// 4096 and 4100 are the same key to this context (offset >> 4), 4112 is another.
	expect(t, !hm2FetchPut(t, m, hmK(4100), 1, hm2Loose).Valid, "(try m.fetchPutContext(term51XK(4100), 1, term51AGLoose)) == null")
	slot := zigUnwrap(t, m.getIndex(hmK(4100), hm2Loose))
	expectEqual(t, uint32(4100), m.keys()[slot])

	old := zigUnwrap(t, hm2FetchPut(t, m, hmK(4097), 2, hm2Loose))
	expectEqual(t, uint32(4100), old.key)
	expectEqual(t, uint16(1), old.value)
	// The stored key was not replaced by the given one.
	expectEqual(t, uint32(4100), m.keys()[slot])
	expectEqual(t, uint16(2), m.values()[slot])
	expectEqual(t, uint32(1), m.Count())

	r, err := m.FetchPut(hmK(4112), 3)
	zigTry(t, err)
	expect(t, !r.Valid, "(try m.fetchPut(term51XK(4112), 3)) == null")
	expectEqual(t, uint32(2), m.Count())
	slot2 := zigUnwrap(t, m.getIndex(hmK(4112), hm2Loose))
	expectEqual(t, uint32(4112), m.keys()[slot2])
}

// Zig: test "term51: HashMapUnmanaged.fetchPutAssumeCapacityContext: new keys return null, existing keys return the old entry"
func TestHashMap_term51_HashMapUnmanaged_fetchPutAssumeCapacityContext_new_keys_return_null_existing_keys_return_the_old_entry(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c[0:6] {
		expect(t, !m.FetchPutAssumeCapacityContext(hmK(k), HyperlinkID(i+10), hmCtx).Valid, "m.fetchPutAssumeCapacityContext(term51XK(k), @intCast(i + 10), term51XCtx) == null")
		expectEqual(t, uint32(i+1), m.Count())
		expectEqual(t, uint(5+i), zigUnwrap(t, hmSlot(m, k)))
		expectEqual(t, k, m.keys()[5+i])
		expectEqual(t, uint16(i+10), m.values()[5+i])
		expectEqual(t, hmUsedCtl(k), hmCtl(m, uint(5+i)))
	}
	for i, k := range c[0:6] {
		old := zigUnwrap(t, m.FetchPutAssumeCapacityContext(hmK(k), HyperlinkID(i+100), hmCtx))
		expectEqual(t, k, old.key)
		expectEqual(t, uint16(i+10), old.value)
		expectEqual(t, uint(5+i), zigUnwrap(t, hmSlot(m, k)))
		expectEqual(t, uint16(i+100), zigUnwrap(t, m.Get(hmK(k))))
	}
	expectEqual(t, uint32(6), m.Count())
	// The wrapper without a context.
	expectEqual(t, uint16(100), zigUnwrap(t, m.FetchPutAssumeCapacity(hmK(c[0]), 7)).value)
	expect(t, !m.FetchPutAssumeCapacity(hmK(c[6]), 1).Valid, "m.fetchPutAssumeCapacity(term51XK(c[6]), 1) == null")
	expectEqual(t, uint32(7), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.fetchPutAssumeCapacityContext: fills and replaces every slot of a full map"
func TestHashMap_term51_HashMapUnmanaged_fetchPutAssumeCapacityContext_fills_and_replaces_every_slot_of_a_full_map(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	for i, k := range hmFull16 {
		expect(t, !m.FetchPutAssumeCapacityContext(hmK(k), hmSl(uint32(i), 1), hmCtx).Valid, "m.fetchPutAssumeCapacityContext(term51XK(k), term51XSl(@intCast(i), 1), term51XCtx) == null")
		expectEqual(t, hmFull16Slots[i], zigUnwrap(t, hmSlot(m, k)))
		expectEqual(t, hmFull16Fps[i]|0x80, hmCtl(m, hmFull16Slots[i]))
	}
	expectEqual(t, uint32(16), m.Count())
	// Replace in reverse: each old entry comes back, the count stays.
	for i := 16; i > 0; {
		i--
		k := hmFull16[i]
		old := zigUnwrap(t, m.FetchPutAssumeCapacityContext(hmK(k), hmSl(uint32(i), 2), hmCtx))
		expectEqual(t, k, old.key)
		hmExpectSl(t, hmSl(uint32(i), 1), old.value)
		expectEqual(t, uint32(16), m.Count())
	}
	for j, k := range hmFull16 {
		hmExpectSl(t, hmSl(uint32(j), 2), zigUnwrap(t, m.Get(hmK(k))))
	}
}

// Zig: test "term51: HashMapUnmanaged.fetchPutAssumeCapacityContext: tombstones are recycled in probe order"
func TestHashMap_term51_HashMapUnmanaged_fetchPutAssumeCapacityContext_tombstones_are_recycled_in_probe_order(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain14
	for i, k := range c[0:4] {
		expect(t, !m.FetchPutAssumeCapacityContext(hmK(k), HyperlinkID(i), hmCtx).Valid, "m.fetchPutAssumeCapacityContext(term51XK(k), @intCast(i), term51XCtx) == null")
	}
	// Slots 14, 15, 0, 1.
	_ = m.Remove(hmK(c[1]))
	_ = m.Remove(hmK(c[2]))
	expect(t, !m.FetchPutAssumeCapacityContext(hmK(c[4]), 4, hmCtx).Valid, "m.fetchPutAssumeCapacityContext(term51XK(c[4]), 4, term51XCtx) == null")
	expectEqual(t, uint(15), zigUnwrap(t, hmSlot(m, c[4])))
	expect(t, !m.FetchPutAssumeCapacityContext(hmK(c[5]), 5, hmCtx).Valid, "m.fetchPutAssumeCapacityContext(term51XK(c[5]), 5, term51XCtx) == null")
	expectEqual(t, uint(0), zigUnwrap(t, hmSlot(m, c[5])))
	// The key behind the tombstones is found.
	expectEqual(t, uint16(3), zigUnwrap(t, m.FetchPutAssumeCapacityContext(hmK(c[3]), 33, hmCtx)).value)
	expectEqual(t, uint32(4), m.Count())
	// A removed key is new.
	expect(t, !m.FetchPutAssumeCapacityContext(hmK(c[1]), 1, hmCtx).Valid, "m.fetchPutAssumeCapacityContext(term51XK(c[1]), 1, term51XCtx) == null")
	expectEqual(t, uint32(5), m.Count())

	// A map of nothing but tombstones takes as many new keys as it has slots.
	b4 := hmHBoxInit(4)
	m4 := b4.map_()
	keys := [...]uint32{8, 16, 24, 32}
	for _, k := range keys {
		zigTry(t, m4.Put(hmK(k), 1))
	}
	for _, k := range keys {
		_ = m4.Remove(hmK(k))
	}
	for i, k := range [...]uint32{40, 48, 56, 64} {
		expect(t, !m4.FetchPutAssumeCapacityContext(hmK(k), 9, hmCtx).Valid, "m4.fetchPutAssumeCapacityContext(term51XK(k), 9, term51XCtx) == null")
		expectEqual(t, uint32(i+1), m4.Count())
	}
	expectEqual(t, uint(0), hmCountCtl(m4, 1))
}

// Zig: test "term51: HashMapUnmanaged.fetchPutAssumeCapacityContext: a new key in a full map overwrites its home slot without any check"
func TestHashMap_term51_HashMapUnmanaged_fetchPutAssumeCapacityContext_a_new_key_in_a_full_map_overwrites_its_home_slot_without_any_check(t *testing.T) {
	// Upstream: the capacity precondition is documented but not asserted. With
	// no free slot and no tombstone the probe comes back to the home slot and
	// fills it: the previous occupant is lost and the count exceeds the capacity.
	box := hmHBoxInit(16)
	m := box.map_()
	hm2Fill16(t, m)
	const nk uint32 = 65536 + 8*16
	home := uint(hmCtx.Hash(hmK(nk)) & 15)
	victim := m.keys()[home]
	expect(t, !m.FetchPutAssumeCapacityContext(hmK(nk), 999, hmCtx).Valid, "m.fetchPutAssumeCapacityContext(term51XK(nk), 999, term51XCtx) == null")
	expectEqual(t, uint32(17), m.Count())
	expectEqual(t, uint32(16), m.Capacity())
	expectEqual(t, nk, m.keys()[home])
	expectEqual(t, uint16(999), m.values()[home])
	expectEqual(t, hmUsedCtl(nk), hmCtl(m, home))
	expect(t, !m.Get(hmK(victim)).Valid, "m.get(term51XK(victim)) == null")
	for _, k := range hmFull16 {
		if k != victim {
			expect(t, m.Get(hmK(k)).Valid, "m.get(term51XK(k)) != null")
		}
	}
	expectEqual(t, uint16(999), zigUnwrap(t, m.Get(hmK(nk))))
}

// Zig: test "term51: HashMapUnmanaged.fetchRemoveContext: returns the removed key and value and leaves a tombstone"
func TestHashMap_term51_HashMapUnmanaged_fetchRemoveContext_returns_the_removed_key_and_value_and_leaves_a_tombstone(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	hm2Fill16(t, m)
	for i, k := range hmFull16 {
		kv := zigUnwrap(t, m.FetchRemoveContext(hmK(k), hmCtx))
		expectEqual(t, k, kv.key)
		expectEqual(t, uint16(i), kv.value)
		expectEqual(t, uint32(15-i), m.Count())
		expectEqual(t, uint8(1), hmCtl(m, hmFull16Slots[i]))
		expect(t, !m.Get(hmK(k)).Valid, "m.get(term51XK(k)) == null")
		// The entries not yet removed are still found, the removed ones are not.
		for j, k2 := range hmFull16 {
			if j > i {
				expectEqual(t, uint16(j), zigUnwrap(t, m.Get(hmK(k2))))
			} else {
				expect(t, !m.Get(hmK(k2)).Valid, "m.get(term51XK(k2)) == null")
			}
		}
	}
	expectEqual(t, uint(16), hmCountCtl(m, 1))
	for _, k := range hmFull16 {
		expect(t, !m.FetchRemoveContext(hmK(k), hmCtx).Valid, "m.fetchRemoveContext(term51XK(k), term51XCtx) == null")
	}
	it := m.Iterator()
	expect(t, !it.Next().Valid, "it.next() == null")
	// The first key goes back to its own (tombstone) slot.
	zigTry(t, m.Put(hmK(hmFull16[0]), 5))
	expectEqual(t, uint(7), zigUnwrap(t, hmSlot(m, hmFull16[0])))
}

// Zig: test "term51: HashMapUnmanaged.fetchRemoveContext: absent keys and empty maps return null and change nothing"
func TestHashMap_term51_HashMapUnmanaged_fetchRemoveContext_absent_keys_and_empty_maps_return_null_and_change_nothing(t *testing.T) {
	z := hmHBoxInit(0)
	zm := z.map_()
	expect(t, !zm.FetchRemoveContext(hmK(8), hmCtx).Valid, "zm.fetchRemoveContext(term51XK(8), term51XCtx) == null")
	expect(t, !zm.FetchRemove(hmK(8)).Valid, "zm.fetchRemove(term51XK(8)) == null")

	e := hmHBoxInit(8)
	em := e.map_()
	expect(t, !em.FetchRemoveContext(hmK(8), hmCtx).Valid, "em.fetchRemoveContext(term51XK(8), term51XCtx) == null")
	expect(t, !em.FetchRemove(hmK(0)).Valid, "em.fetchRemove(term51XK(0)) == null")
	expectEqual(t, uint(8), hmCountCtl(em, 0))

	box := hmHBoxInit(8)
	m := box.map_()
	c := hmChain7of8
	for i, k := range c[0:3] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i)))
	}
	// Slots 7, 0, 1.
	before := hm2Snap(m)
	expect(t, !m.FetchRemoveContext(hmK(c[3]), hmCtx).Valid, "m.fetchRemoveContext(term51XK(c[3]), term51XCtx) == null")
	expect(t, !m.FetchRemoveContext(hmK(12345), hmCtx).Valid, "m.fetchRemoveContext(term51XK(12345), term51XCtx) == null")
	expectEqual(t, uint32(3), m.Count())
	expectEqualSlices(t, before, hm2Snap(m))

	// A probe that passes a tombstone before it ends at a free slot.
	expect(t, m.Remove(hmK(c[1])), "m.remove(term51XK(c[1]))")
	before = hm2Snap(m)
	expect(t, !m.FetchRemoveContext(hmK(c[4]), hmCtx).Valid, "m.fetchRemoveContext(term51XK(c[4]), term51XCtx) == null")
	expect(t, !m.FetchRemoveContext(hmK(c[1]), hmCtx).Valid, "m.fetchRemoveContext(term51XK(c[1]), term51XCtx) == null")
	expectEqual(t, uint32(2), m.Count())
	expectEqualSlices(t, before, hm2Snap(m))
}

// Zig: test "term51: HashMapUnmanaged.fetchRemoveContext: the result is a copy, keys behind the tombstone stay reachable"
func TestHashMap_term51_HashMapUnmanaged_fetchRemoveContext_the_result_is_a_copy_keys_behind_the_tombstone_stay_reachable(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c {
		zigTry(t, m.Put(hmK(k), hmSl(uint32(i), uint(i+1))))
	}
	// Slots 5..12.
	kv := zigUnwrap(t, m.FetchRemoveContext(hmK(c[2]), hmCtx))
	expectEqual(t, c[2], kv.key)
	hmExpectSl(t, hmSl(2, 3), kv.value)
	kv.value = hmSl(hmDesc(kv.value).offset, 99) // kv.value.len = 99
	_ = kv
	expectEqual(t, uint8(1), hmCtl(m, 7))
	expectEqual(t, uint32(7), m.Count())
	for i, k := range c {
		if i == 2 {
			expect(t, !m.Get(hmK(k)).Valid, "m.get(term51XK(k)) == null")
		} else {
			hmExpectSl(t, hmSl(uint32(i), uint(i+1)), zigUnwrap(t, m.Get(hmK(k))))
		}
	}
	// The last key of the chain: its slot becomes a tombstone before a free slot.
	last := zigUnwrap(t, m.FetchRemove(hmK(c[7])))
	hmExpectSl(t, hmSl(7, 8), last.value)
	expectEqual(t, uint8(1), hmCtl(m, 12))
	expectEqual(t, uint8(0), hmCtl(m, 13))
	// Put it back: the first tombstone in probe order is slot 7.
	zigTry(t, m.Put(hmK(c[2]), hmSl(50, 5)))
	expectEqual(t, uint(7), zigUnwrap(t, hmSlot(m, c[2])))
	hmExpectSl(t, hmSl(50, 5), zigUnwrap(t, m.Get(hmK(c[2]))))
}

// Zig: test "term51: HashMapUnmanaged.fetchRemoveContext: returns the stored key, not the one given"
func TestHashMap_term51_HashMapUnmanaged_fetchRemoveContext_returns_the_stored_key_not_the_one_given(t *testing.T) {
	box := hm2LBoxInit(16)
	m := box.map_()
	zigTry(t, m.Put(hmK(4096), 7))
	zigTry(t, m.Put(hmK(4112), 8))
	kv := zigUnwrap(t, m.FetchRemoveContext(hmK(4101), hm2Loose))
	expectEqual(t, uint32(4096), kv.key)
	expectEqual(t, uint16(7), kv.value)
	expectEqual(t, uint32(1), m.Count())
	expect(t, !m.getIndex(hmK(4096), hm2Loose).Valid, "m.getIndex(term51XK(4096), term51AGLoose) == null")
	kv2 := zigUnwrap(t, m.FetchRemove(hmK(4127)))
	expectEqual(t, uint32(4112), kv2.key)
	expectEqual(t, uint32(0), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.fetchRemoveAdapted: remove by an adapted key and return the entry"
func TestHashMap_term51_HashMapUnmanaged_fetchRemoveAdapted_remove_by_an_adapted_key_and_return_the_entry(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig: `const raw: term51XRawCtx = .{}`; see the file header.
	raw := hmCtx
	c := hmChain5
	for i, k := range c[0:6] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i+10)))
	}
	kv := zigUnwrap(t, m.FetchRemoveAdapted(c[2], raw))
	expectEqual(t, c[2], kv.key)
	expectEqual(t, uint16(12), kv.value)
	expectEqual(t, uint8(1), hmCtl(m, 7))
	expectEqual(t, uint32(5), m.Count())
	expect(t, !m.FetchRemoveAdapted(c[2], raw).Valid, "m.fetchRemoveAdapted(c[2], raw) == null")
	expect(t, !m.FetchRemoveAdapted(c[7], raw).Valid, "m.fetchRemoveAdapted(c[7], raw) == null")
	expect(t, !m.FetchRemoveAdapted(1, raw).Valid, "m.fetchRemoveAdapted(@as(u32, 1), raw) == null")
	expectEqual(t, uint32(5), m.Count())
	// Keys behind the tombstone, the first and the last of the chain.
	expectEqual(t, uint16(13), zigUnwrap(t, m.FetchRemoveAdapted(c[3], raw)).value)
	expectEqual(t, uint16(10), zigUnwrap(t, m.FetchRemoveAdapted(c[0], raw)).value)
	expectEqual(t, uint16(15), zigUnwrap(t, m.FetchRemoveAdapted(c[5], raw)).value)
	expectEqual(t, uint16(14), zigUnwrap(t, m.FetchRemoveAdapted(c[4], raw)).value)
	expectEqual(t, uint32(1), m.Count())
	expectEqual(t, uint16(11), zigUnwrap(t, m.FetchRemoveAdapted(c[1], raw)).value)
	expectEqual(t, uint32(0), m.Count())
	expect(t, !m.FetchRemoveAdapted(c[1], raw).Valid, "m.fetchRemoveAdapted(c[1], raw) == null")

	// A full map, removed in reverse order.
	fb := hmHBoxInit(16)
	fm := fb.map_()
	hm2Fill16(t, fm)
	for i := 16; i > 0; {
		i--
		r := zigUnwrap(t, fm.FetchRemoveAdapted(hmFull16[i], raw))
		expectEqual(t, hmFull16[i], r.key)
		expectEqual(t, uint16(i), r.value)
	}
	expectEqual(t, uint32(0), fm.Count())
	expect(t, !fm.FetchRemoveAdapted(hmFull16[0], raw).Valid, "fm.fetchRemoveAdapted(term51XFull16[0], raw) == null")
}

// Zig: test "term51: HashMapUnmanaged.getKeyPtrContext: pointer to the stored key of the slot, null when absent"
func TestHashMap_term51_HashMapUnmanaged_getKeyPtrContext_pointer_to_the_stored_key_of_the_slot_null_when_absent(t *testing.T) {
	z := hmHBoxInit(0)
	zm := z.map_()
	expect(t, zm.GetKeyPtrContext(hmK(8), hmCtx) == nil, "zm.getKeyPtrContext(term51XK(8), term51XCtx) == null")
	e := hmHBoxInit(8)
	em := e.map_()
	expect(t, em.GetKeyPtrContext(hmK(8), hmCtx) == nil, "em.getKeyPtrContext(term51XK(8), term51XCtx) == null")

	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c[0:6] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i)))
	}
	for i, k := range c[0:6] {
		p := m.GetKeyPtrContext(hmK(k), hmCtx)
		expect(t, p != nil, "m.getKeyPtrContext(term51XK(k), term51XCtx) != null")
		expect(t, &m.keys()[5+i] == p, "@intFromPtr(&m.keys()[5 + i]) == @intFromPtr(p)")
		expectEqual(t, k, *p)
		expect(t, zigUnwrap(t, m.GetEntryContext(hmK(k), hmCtx)).key_ptr == p, "@intFromPtr(m.getEntryContext(term51XK(k), term51XCtx).?.key_ptr) == @intFromPtr(p)")
		expect(t, m.GetKeyPtr(hmK(k)) == p, "@intFromPtr(m.getKeyPtr(term51XK(k)).?) == @intFromPtr(p)")
	}
	expect(t, m.GetKeyPtrContext(hmK(c[6]), hmCtx) == nil, "m.getKeyPtrContext(term51XK(c[6]), term51XCtx) == null")
	expect(t, m.GetKeyPtrContext(hmK(c[7]), hmCtx) == nil, "m.getKeyPtrContext(term51XK(c[7]), term51XCtx) == null")
	expect(t, m.GetKeyPtrContext(hmK(0), hmCtx) == nil, "m.getKeyPtrContext(term51XK(0), term51XCtx) == null")
	expect(t, m.GetKeyPtr(hmK(1<<24)) == nil, "m.getKeyPtr(term51XK(1 << 24)) == null")
	// Updating the value leaves the key slot where it is.
	p0 := m.GetKeyPtrContext(hmK(c[0]), hmCtx)
	zigTry(t, m.Put(hmK(c[0]), 77))
	expect(t, p0 == m.GetKeyPtrContext(hmK(c[0]), hmCtx), "@intFromPtr(p0) == @intFromPtr(m.getKeyPtrContext(term51XK(c[0]), term51XCtx).?)")
	expectEqual(t, uint32(6), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.getKeyPtrContext: full map, removals and the slot reused by a new key"
func TestHashMap_term51_HashMapUnmanaged_getKeyPtrContext_full_map_removals_and_the_slot_reused_by_a_new_key(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	hm2Fill16(t, m)
	for i, k := range hmFull16 {
		p := m.GetKeyPtrContext(hmK(k), hmCtx)
		expect(t, p != nil, "m.getKeyPtrContext(term51XK(k), term51XCtx) != null")
		expect(t, &m.keys()[hmFull16Slots[i]] == p, "@intFromPtr(&m.keys()[term51XFull16Slots[i]]) == @intFromPtr(p)")
		expectEqual(t, k, *p)
	}
	// Absent keys end the probe although there is no free slot.
	for j := range 20 {
		expect(t, m.GetKeyPtrContext(hmK(uint32(65664+8*j)), hmCtx) == nil, "m.getKeyPtrContext(term51XK(@intCast(65664 + 8 * j)), term51XCtx) == null")
	}
	expect(t, m.GetKeyPtrContext(hmK(0), hmCtx) == nil, "m.getKeyPtrContext(term51XK(0), term51XCtx) == null")

	// Removed keys are not found, the others keep their pointers.
	keep := m.GetKeyPtrContext(hmK(hmFull16[12]), hmCtx)
	p0 := m.GetKeyPtrContext(hmK(hmFull16[0]), hmCtx)
	for _, i := range [...]int{0, 5, 9} {
		expect(t, m.Remove(hmK(hmFull16[i])), "m.remove(term51XK(term51XFull16[i]))")
	}
	for _, i := range [...]int{0, 5, 9} {
		expect(t, m.GetKeyPtrContext(hmK(hmFull16[i]), hmCtx) == nil, "m.getKeyPtrContext(term51XK(term51XFull16[i]), term51XCtx) == null")
	}
	expect(t, keep == m.GetKeyPtrContext(hmK(hmFull16[12]), hmCtx), "@intFromPtr(keep) == @intFromPtr(m.getKeyPtrContext(term51XK(term51XFull16[12]), term51XCtx).?)")
	expect(t, &m.keys()[hmFull16Slots[12]] == keep, "@intFromPtr(&m.keys()[term51XFull16Slots[12]]) == @intFromPtr(keep)")
	// The first key goes back to the slot it had (a tombstone is recycled): the old
	// pointer is the pointer of the new entry.
	zigTry(t, m.Put(hmK(hmFull16[0]), 40))
	expect(t, p0 == m.GetKeyPtrContext(hmK(hmFull16[0]), hmCtx), "@intFromPtr(p0) == @intFromPtr(m.getKeyPtrContext(term51XK(term51XFull16[0]), term51XCtx).?)")
	// A different key takes a tombstone slot: the pointer of the new key is its slot.
	zigTry(t, m.Put(hmK(65664), 41))
	found := m.GetKeyPtrContext(hmK(65664), hmCtx)
	expect(t, found != nil, "m.getKeyPtrContext(term51XK(65664), term51XCtx) != null")
	expectEqual(t, uint32(65664), *found)
	expect(t, &m.keys()[zigUnwrap(t, hmSlot(m, 65664))] == found, "@intFromPtr(&m.keys()[term51XSlot(m, 65664).?]) == @intFromPtr(found)")
}

// Zig: test "term51: HashMapUnmanaged.getKeyPtrContext: the key can be rewritten in place under a looser context"
func TestHashMap_term51_HashMapUnmanaged_getKeyPtrContext_the_key_can_be_rewritten_in_place_under_a_looser_context(t *testing.T) {
	box := hm2LBoxInit(16)
	m := box.map_()
	zigTry(t, m.Put(hmK(4100), 1))
	p := m.GetKeyPtrContext(hmK(4096), hm2Loose)
	expect(t, p != nil, "m.getKeyPtrContext(term51XK(4096), term51AGLoose) != null")
	expectEqual(t, uint32(4100), *p)
	expect(t, &m.keys()[zigUnwrap(t, m.getIndex(hmK(4100), hm2Loose))] == p, "@intFromPtr(&m.keys()[m.getIndex(term51XK(4100), term51AGLoose).?]) == @intFromPtr(p)")
	// Another representative of the same key: lookups still succeed.
	*p = hmK(4111)
	expectEqual(t, uint32(4111), *m.GetKeyPtrContext(hmK(4096), hm2Loose))
	expectEqual(t, uint16(1), zigUnwrap(t, m.Get(hmK(4100))))
	expect(t, m.GetKeyPtr(hmK(4112)) == nil, "m.getKeyPtr(term51XK(4112)) == null")
}

// Zig: test "term51: HashMapUnmanaged.getKeyPtrAdapted: pointer by an adapted key"
func TestHashMap_term51_HashMapUnmanaged_getKeyPtrAdapted_pointer_by_an_adapted_key(t *testing.T) {
	z := hmHBoxInit(0)
	zm := z.map_()
	// Zig: `const raw: term51XRawCtx = .{}`; see the file header.
	raw := hmCtx
	expect(t, zm.GetKeyPtrAdapted(8, raw) == nil, "zm.getKeyPtrAdapted(@as(u32, 8), raw) == null")

	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	expect(t, m.GetKeyPtrAdapted(c[0], raw) == nil, "m.getKeyPtrAdapted(c[0], raw) == null")
	for i, k := range c[0:4] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i)))
	}
	for i, k := range c[0:4] {
		p := m.GetKeyPtrAdapted(k, raw)
		expect(t, p != nil, "m.getKeyPtrAdapted(k, raw) != null")
		expect(t, &m.keys()[5+i] == p, "@intFromPtr(&m.keys()[5 + i]) == @intFromPtr(p)")
		expectEqual(t, k, *p)
	}
	expect(t, m.GetKeyPtrAdapted(c[4], raw) == nil, "m.getKeyPtrAdapted(c[4], raw) == null")
	expect(t, m.GetKeyPtrAdapted(1, raw) == nil, "m.getKeyPtrAdapted(@as(u32, 1), raw) == null")
	// Behind a tombstone.
	_ = m.Remove(hmK(c[1]))
	expect(t, m.GetKeyPtrAdapted(c[1], raw) == nil, "m.getKeyPtrAdapted(c[1], raw) == null")
	expect(t, &m.keys()[7] == m.GetKeyPtrAdapted(c[2], raw), "@intFromPtr(&m.keys()[7]) == @intFromPtr(m.getKeyPtrAdapted(c[2], raw).?)")
}

// Zig: test "term51: HashMapUnmanaged.getKey: a copy of the stored key, null when absent"
func TestHashMap_term51_HashMapUnmanaged_getKey_a_copy_of_the_stored_key_null_when_absent(t *testing.T) {
	z := hmHBoxInit(0)
	zm := z.map_()
	expect(t, !zm.GetKey(hmK(8)).Valid, "zm.getKey(term51XK(8)) == null")
	e := hmHBoxInit(8)
	em := e.map_()
	expect(t, !em.GetKey(hmK(8)).Valid, "em.getKey(term51XK(8)) == null")

	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c[0:6] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i)))
	}
	for _, k := range c[0:6] {
		expectEqual(t, k, zigUnwrap(t, m.GetKey(hmK(k))))
	}
	expect(t, !m.GetKey(hmK(c[6])).Valid, "m.getKey(term51XK(c[6])) == null")
	expect(t, !m.GetKey(hmK(0)).Valid, "m.getKey(term51XK(0)) == null")
	// A copy: changing it does not change the map.
	k := zigUnwrap(t, m.GetKey(hmK(c[0])))
	k = 1 // k.offset = 1
	_ = k
	expectEqual(t, c[0], m.keys()[5])
	// A removed key is gone, the keys behind its tombstone are not.
	_ = m.Remove(hmK(c[1]))
	expect(t, !m.GetKey(hmK(c[1])).Valid, "m.getKey(term51XK(c[1])) == null")
	expectEqual(t, c[2], zigUnwrap(t, m.GetKey(hmK(c[2]))))
	zigTry(t, m.Put(hmK(c[1]), 5))
	expectEqual(t, c[1], zigUnwrap(t, m.GetKey(hmK(c[1]))))
}

// Zig: test "term51: HashMapUnmanaged.getKey: the stored key is returned, not the one given"
func TestHashMap_term51_HashMapUnmanaged_getKey_the_stored_key_is_returned_not_the_one_given(t *testing.T) {
	box := hm2LBoxInit(16)
	m := box.map_()
	zigTry(t, m.Put(hmK(4100), 1))
	expectEqual(t, uint32(4100), zigUnwrap(t, m.GetKey(hmK(4096))))
	expectEqual(t, uint32(4100), zigUnwrap(t, m.GetKey(hmK(4111))))
	expect(t, !m.GetKey(hmK(4112)).Valid, "m.getKey(term51XK(4112)) == null")
	expect(t, !m.GetKey(hmK(4095)).Valid, "m.getKey(term51XK(4095)) == null")
}

// Zig: test "term51: HashMapUnmanaged.getKeyContext: full map, removals and the stored key under a looser context"
func TestHashMap_term51_HashMapUnmanaged_getKeyContext_full_map_removals_and_the_stored_key_under_a_looser_context(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	hm2Fill16(t, m)
	for _, k := range hmFull16 {
		expectEqual(t, k, zigUnwrap(t, m.GetKeyContext(hmK(k), hmCtx)))
	}
	for j := range 20 {
		expect(t, !m.GetKeyContext(hmK(uint32(65664+8*j)), hmCtx).Valid, "m.getKeyContext(term51XK(@intCast(65664 + 8 * j)), term51XCtx) == null")
	}
	for _, i := range [...]int{1, 4, 8, 15} {
		_ = m.Remove(hmK(hmFull16[i]))
	}
	for i, k := range hmFull16 {
		gone := i == 1 || i == 4 || i == 8 || i == 15
		expectEqual(t, !gone, m.GetKeyContext(hmK(k), hmCtx).Valid)
	}
	z := hmHBoxInit(0)
	zm := z.map_()
	expect(t, !zm.GetKeyContext(hmK(8), hmCtx).Valid, "z.map().getKeyContext(term51XK(8), term51XCtx) == null")

	lb := hm2LBoxInit(8)
	lm := lb.map_()
	zigTry(t, lm.Put(hmK(4100), 1))
	expectEqual(t, uint32(4100), zigUnwrap(t, lm.GetKeyContext(hmK(4097), hm2Loose)))
	expect(t, !lm.GetKeyContext(hmK(4128), hm2Loose).Valid, "lm.getKeyContext(term51XK(4128), term51AGLoose) == null")
}

// Zig: test "term51: HashMapUnmanaged.getKeyAdapted: key by an adapted key"
func TestHashMap_term51_HashMapUnmanaged_getKeyAdapted_key_by_an_adapted_key(t *testing.T) {
	z := hmHBoxInit(0)
	zm := z.map_()
	// Zig: `const raw: term51XRawCtx = .{}`; see the file header.
	raw := hmCtx
	expect(t, !zm.GetKeyAdapted(8, raw).Valid, "z.map().getKeyAdapted(@as(u32, 8), raw) == null")

	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	expect(t, !m.GetKeyAdapted(c[0], raw).Valid, "m.getKeyAdapted(c[0], raw) == null")
	for i, k := range c[0:5] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i)))
	}
	for _, k := range c[0:5] {
		expectEqual(t, k, zigUnwrap(t, m.GetKeyAdapted(k, raw)))
	}
	expect(t, !m.GetKeyAdapted(c[5], raw).Valid, "m.getKeyAdapted(c[5], raw) == null")
	expect(t, !m.GetKeyAdapted(1, raw).Valid, "m.getKeyAdapted(@as(u32, 1), raw) == null")
	_ = m.Remove(hmK(c[0]))
	_ = m.Remove(hmK(c[2]))
	expect(t, !m.GetKeyAdapted(c[0], raw).Valid, "m.getKeyAdapted(c[0], raw) == null")
	expect(t, !m.GetKeyAdapted(c[2], raw).Valid, "m.getKeyAdapted(c[2], raw) == null")
	expectEqual(t, c[1], zigUnwrap(t, m.GetKeyAdapted(c[1], raw)))
	expectEqual(t, c[4], zigUnwrap(t, m.GetKeyAdapted(c[4], raw)))
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAdapted: a new key reserves a slot and leaves the key to the caller"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAdapted_a_new_key_reserves_a_slot_and_leaves_the_key_to_the_caller(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig: `const raw: term51XRawCtx = .{}`; see the file header.
	raw := hmCtx
	c := hmChain5
	for i, k := range c[0:6] {
		gop, err := m.GetOrPutAdapted(k, raw)
		zigTry(t, err)
		expect(t, !gop.found_existing, "!gop.found_existing")
		expect(t, &m.keys()[5+i] == gop.key_ptr, "@intFromPtr(&m.keys()[5 + i]) == @intFromPtr(gop.key_ptr)")
		expect(t, &m.values()[5+i] == gop.value_ptr, "@intFromPtr(&m.values()[5 + i]) == @intFromPtr(gop.value_ptr)")
		// The slot is already counted and marked with the fingerprint.
		expectEqual(t, uint32(i+1), m.Count())
		expectEqual(t, hmUsedCtl(k), hmCtl(m, uint(5+i)))
		*gop.key_ptr = hmK(k)
		*gop.value_ptr = HyperlinkID(i + 10)
		expectEqual(t, uint(5+i), zigUnwrap(t, hmSlot(m, k)))
		expectEqual(t, uint16(i+10), zigUnwrap(t, m.Get(hmK(k))))
	}
	expectEqual(t, uint32(6), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAdapted: an existing key is found with pointers into its slot"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAdapted_an_existing_key_is_found_with_pointers_into_its_slot(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig: `const raw: term51XRawCtx = .{}`; see the file header.
	raw := hmCtx
	c := hmChain5
	for i, k := range c[0:4] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i+10)))
	}
	for i, k := range c[0:4] {
		gop, err := m.GetOrPutAdapted(k, raw)
		zigTry(t, err)
		expect(t, gop.found_existing, "gop.found_existing")
		expect(t, &m.keys()[5+i] == gop.key_ptr, "@intFromPtr(&m.keys()[5 + i]) == @intFromPtr(gop.key_ptr)")
		expectEqual(t, k, *gop.key_ptr)
		expectEqual(t, uint16(i+10), *gop.value_ptr)
		entry := zigUnwrap(t, m.GetEntry(hmK(k)))
		expect(t, entry.key_ptr == gop.key_ptr, "@intFromPtr(entry.key_ptr) == @intFromPtr(gop.key_ptr)")
		expect(t, entry.value_ptr == gop.value_ptr, "@intFromPtr(entry.value_ptr) == @intFromPtr(gop.value_ptr)")
		expect(t, m.GetPtr(hmK(k)) == gop.value_ptr, "@intFromPtr(m.getPtr(term51XK(k)).?) == @intFromPtr(gop.value_ptr)")
		expect(t, m.GetKeyPtr(hmK(k)) == gop.key_ptr, "@intFromPtr(m.getKeyPtr(term51XK(k)).?) == @intFromPtr(gop.key_ptr)")
		expectEqual(t, uint32(4), m.Count())
	}
	// A write through the value pointer is the map's value; a second call
	// returns the same pointers.
	first, err := m.GetOrPutAdapted(c[1], raw)
	zigTry(t, err)
	*first.value_ptr = 99
	second, err := m.GetOrPutAdapted(c[1], raw)
	zigTry(t, err)
	expect(t, second.found_existing, "second.found_existing")
	expect(t, first.key_ptr == second.key_ptr, "@intFromPtr(first.key_ptr) == @intFromPtr(second.key_ptr)")
	expect(t, first.value_ptr == second.value_ptr, "@intFromPtr(first.value_ptr) == @intFromPtr(second.value_ptr)")
	expectEqual(t, uint16(99), zigUnwrap(t, m.Get(hmK(c[1]))))
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAdapted: a full map finds existing keys and refuses new ones"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAdapted_a_full_map_finds_existing_keys_and_refuses_new_ones(t *testing.T) {
	box := hmHBoxInit(8)
	m := box.map_()
	// Zig: `const raw: term51XRawCtx = .{}`; see the file header.
	raw := hmCtx
	// Eight new keys fit, the last one fills the map.
	for i := range 8 {
		k := uint32(8 * i)
		gop, err := m.GetOrPutAdapted(k, raw)
		zigTry(t, err)
		expect(t, !gop.found_existing, "!gop.found_existing")
		*gop.key_ptr = hmK(k)
		*gop.value_ptr = HyperlinkID(i + 1)
	}
	expectEqual(t, uint32(8), m.Count())
	for i := range 8 {
		k := uint32(8 * i)
		gop, err := m.GetOrPutAdapted(k, raw)
		zigTry(t, err)
		expect(t, gop.found_existing, "gop.found_existing")
		expectEqual(t, uint16(i+1), *gop.value_ptr)
		expect(t, &m.keys()[zigUnwrap(t, hmSlot(m, k))] == gop.key_ptr, "@intFromPtr(&m.keys()[term51XSlot(m, k).?]) == @intFromPtr(gop.key_ptr)")
	}
	_, err := m.GetOrPutAdapted(800, raw)
	hmExpectError(t, ErrOutOfMemory, err)
	expectEqual(t, uint32(8), m.Count())
	expect(t, !m.Get(hmK(800)).Valid, "m.get(term51XK(800)) == null")
	// After a removal a new key fits and takes the tombstone.
	slot := zigUnwrap(t, hmSlot(m, 16))
	expect(t, m.Remove(hmK(16)), "m.remove(term51XK(16))")
	gop, err := m.GetOrPutAdapted(800, raw)
	zigTry(t, err)
	expect(t, !gop.found_existing, "!gop.found_existing")
	expect(t, &m.keys()[slot] == gop.key_ptr, "@intFromPtr(&m.keys()[slot]) == @intFromPtr(gop.key_ptr)")
	*gop.key_ptr = hmK(800)
	*gop.value_ptr = 9
	expectEqual(t, uint32(8), m.Count())
	_, err = m.GetOrPutAdapted(808, raw)
	hmExpectError(t, ErrOutOfMemory, err)

	z := hmHBoxInit(0)
	zm := z.map_()
	_, err = zm.GetOrPutAdapted(8, raw)
	hmExpectError(t, ErrOutOfMemory, err)
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAdapted: tombstones are recycled in probe order and keys behind them are found"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAdapted_tombstones_are_recycled_in_probe_order_and_keys_behind_them_are_found(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	// Zig: `const raw: term51XRawCtx = .{}`; see the file header.
	raw := hmCtx
	c := hmChain14
	for i, k := range c[0:4] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i)))
	}
	// Slots 14, 15, 0, 1; tombstones at 15 and 0.
	_ = m.Remove(hmK(c[1]))
	_ = m.Remove(hmK(c[2]))
	g4, err := m.GetOrPutAdapted(c[4], raw)
	zigTry(t, err)
	expect(t, !g4.found_existing, "!g4.found_existing")
	expect(t, &m.keys()[15] == g4.key_ptr, "@intFromPtr(&m.keys()[15]) == @intFromPtr(g4.key_ptr)")
	*g4.key_ptr = hmK(c[4])
	*g4.value_ptr = 4
	g5, err := m.GetOrPutAdapted(c[5], raw)
	zigTry(t, err)
	expect(t, !g5.found_existing, "!g5.found_existing")
	expect(t, &m.keys()[0] == g5.key_ptr, "@intFromPtr(&m.keys()[0]) == @intFromPtr(g5.key_ptr)")
	*g5.key_ptr = hmK(c[5])
	*g5.value_ptr = 5
	g3, err := m.GetOrPutAdapted(c[3], raw)
	zigTry(t, err)
	expect(t, g3.found_existing, "g3.found_existing")
	expect(t, &m.keys()[1] == g3.key_ptr, "@intFromPtr(&m.keys()[1]) == @intFromPtr(g3.key_ptr)")
	g0, err := m.GetOrPutAdapted(c[0], raw)
	zigTry(t, err)
	expect(t, g0.found_existing, "g0.found_existing")
	expect(t, &m.keys()[14] == g0.key_ptr, "@intFromPtr(&m.keys()[14]) == @intFromPtr(g0.key_ptr)")
	expectEqual(t, uint32(4), m.Count())

	// No free slot left but a tombstone: a new key recycles it, then the map is full.
	b4 := hmHBoxInit(4)
	m4 := b4.map_()
	keys := [...]uint32{8, 16, 24, 32}
	for i, k := range keys {
		zigTry(t, m4.Put(hmK(k), HyperlinkID(i)))
	}
	tomb := zigUnwrap(t, hmSlot(m4, 32))
	_ = m4.Remove(hmK(32))
	expectEqual(t, uint(0), hmCountCtl(m4, 0))
	f, err := m4.GetOrPutAdapted(16, raw)
	zigTry(t, err)
	expect(t, f.found_existing, "f.found_existing")
	n, err := m4.GetOrPutAdapted(40, raw)
	zigTry(t, err)
	expect(t, !n.found_existing, "!n.found_existing")
	expect(t, &m4.keys()[tomb] == n.key_ptr, "@intFromPtr(&m4.keys()[tomb]) == @intFromPtr(n.key_ptr)")
	*n.key_ptr = hmK(40)
	expectEqual(t, uint32(4), m4.Count())
	_, err = m4.GetOrPutAdapted(48, raw)
	hmExpectError(t, ErrOutOfMemory, err)
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacity: a new key is stored by the call"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacity_a_new_key_is_stored_by_the_call(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c {
		gop := m.GetOrPutAssumeCapacity(hmK(k))
		expect(t, !gop.found_existing, "!gop.found_existing")
		expectEqual(t, k, *gop.key_ptr)
		expect(t, &m.keys()[5+i] == gop.key_ptr, "@intFromPtr(&m.keys()[5 + i]) == @intFromPtr(gop.key_ptr)")
		expect(t, &m.values()[5+i] == gop.value_ptr, "@intFromPtr(&m.values()[5 + i]) == @intFromPtr(gop.value_ptr)")
		*gop.value_ptr = HyperlinkID(i)
		expectEqual(t, uint32(i+1), m.Count())
		expectEqual(t, hmUsedCtl(k), hmCtl(m, uint(5+i)))
	}
	for i, k := range c {
		expectEqual(t, uint16(i), zigUnwrap(t, m.Get(hmK(k))))
	}
	expectEqual(t, uint32(8), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacity: an existing key is found, its key and value are untouched"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacity_an_existing_key_is_found_its_key_and_value_are_untouched(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c[0:4] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i+10)))
	}
	for i, k := range c[0:4] {
		gop := m.GetOrPutAssumeCapacity(hmK(k))
		expect(t, gop.found_existing, "gop.found_existing")
		expect(t, &m.keys()[5+i] == gop.key_ptr, "@intFromPtr(&m.keys()[5 + i]) == @intFromPtr(gop.key_ptr)")
		expect(t, &m.values()[5+i] == gop.value_ptr, "@intFromPtr(&m.values()[5 + i]) == @intFromPtr(gop.value_ptr)")
		expectEqual(t, k, *gop.key_ptr)
		expectEqual(t, uint16(i+10), *gop.value_ptr)
		again := m.GetOrPutAssumeCapacity(hmK(k))
		expect(t, gop.key_ptr == again.key_ptr, "@intFromPtr(gop.key_ptr) == @intFromPtr(again.key_ptr)")
		expect(t, gop.value_ptr == again.value_ptr, "@intFromPtr(gop.value_ptr) == @intFromPtr(again.value_ptr)")
	}
	expectEqual(t, uint32(4), m.Count())
	// Under a looser context the stored key stays, a new entry gets the given key.
	lb := hm2LBoxInit(16)
	lm := lb.map_()
	a := lm.GetOrPutAssumeCapacity(hmK(4100))
	expect(t, !a.found_existing, "!a.found_existing")
	*a.value_ptr = 3
	b := lm.GetOrPutAssumeCapacity(hmK(4097))
	expect(t, b.found_existing, "b.found_existing")
	expectEqual(t, uint32(4100), *b.key_ptr)
	expect(t, a.key_ptr == b.key_ptr, "@intFromPtr(a.key_ptr) == @intFromPtr(b.key_ptr)")
	expectEqual(t, uint16(3), *b.value_ptr)
	expectEqual(t, uint32(1), lm.Count())
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacity: tombstones are recycled in probe order across the wrap"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacity_tombstones_are_recycled_in_probe_order_across_the_wrap(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain14
	for i, k := range c[0:4] {
		*m.GetOrPutAssumeCapacity(hmK(k)).value_ptr = HyperlinkID(i)
	}
	_ = m.Remove(hmK(c[1]))
	_ = m.Remove(hmK(c[2]))
	g4 := m.GetOrPutAssumeCapacity(hmK(c[4]))
	expect(t, !g4.found_existing, "!g4.found_existing")
	expect(t, &m.keys()[15] == g4.key_ptr, "@intFromPtr(&m.keys()[15]) == @intFromPtr(g4.key_ptr)")
	g5 := m.GetOrPutAssumeCapacity(hmK(c[5]))
	expect(t, !g5.found_existing, "!g5.found_existing")
	expect(t, &m.keys()[0] == g5.key_ptr, "@intFromPtr(&m.keys()[0]) == @intFromPtr(g5.key_ptr)")
	// Found behind tombstones.
	g3 := m.GetOrPutAssumeCapacity(hmK(c[3]))
	expect(t, g3.found_existing, "g3.found_existing")
	expectEqual(t, uint16(3), *g3.value_ptr)
	expectEqual(t, uint32(4), m.Count())
	// A removed key comes back at the first tombstone on its path.
	_ = m.Remove(hmK(c[0]))
	g0 := m.GetOrPutAssumeCapacity(hmK(c[0]))
	expect(t, !g0.found_existing, "!g0.found_existing")
	expect(t, &m.keys()[14] == g0.key_ptr, "@intFromPtr(&m.keys()[14]) == @intFromPtr(g0.key_ptr)")
	expectEqual(t, uint32(4), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacity: fills a map completely and finds every key in it"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacity_fills_a_map_completely_and_finds_every_key_in_it(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	for i, k := range hmFull16 {
		gop := m.GetOrPutAssumeCapacity(hmK(k))
		expect(t, !gop.found_existing, "!gop.found_existing")
		*gop.value_ptr = HyperlinkID(i)
		expect(t, &m.keys()[hmFull16Slots[i]] == gop.key_ptr, "@intFromPtr(&m.keys()[term51XFull16Slots[i]]) == @intFromPtr(gop.key_ptr)")
	}
	expectEqual(t, uint32(16), m.Count())
	for i, k := range hmFull16 {
		gop := m.GetOrPutAssumeCapacity(hmK(k))
		expect(t, gop.found_existing, "gop.found_existing")
		expectEqual(t, uint16(i), *gop.value_ptr)
		expect(t, &m.keys()[hmFull16Slots[i]] == gop.key_ptr, "@intFromPtr(&m.keys()[term51XFull16Slots[i]]) == @intFromPtr(gop.key_ptr)")
	}
	expectEqual(t, uint32(16), m.Count())

	// Capacity one.
	one := hmHBoxInit(1)
	om := one.map_()
	g := om.GetOrPutAssumeCapacity(hmK(8))
	expect(t, !g.found_existing, "!g.found_existing")
	*g.value_ptr = 4
	h := om.GetOrPutAssumeCapacity(hmK(8))
	expect(t, h.found_existing, "h.found_existing")
	expect(t, g.key_ptr == h.key_ptr, "@intFromPtr(g.key_ptr) == @intFromPtr(h.key_ptr)")
	expectEqual(t, uint16(4), *h.value_ptr)
	expectEqual(t, uint32(1), om.Count())
}

// Zig: test "term51: HashMapUnmanaged.getOrPutAssumeCapacity: a new key in a full map overwrites its home slot without any check"
func TestHashMap_term51_HashMapUnmanaged_getOrPutAssumeCapacity_a_new_key_in_a_full_map_overwrites_its_home_slot_without_any_check(t *testing.T) {
	// Upstream: like fetchPutAssumeCapacityContext, the capacity precondition is
	// not asserted: the new key lands on its home slot, the previous occupant
	// is lost and the count passes the capacity.
	box := hmHBoxInit(16)
	m := box.map_()
	hm2Fill16(t, m)
	const nk uint32 = 65536 + 8*16
	home := uint(hmCtx.Hash(hmK(nk)) & 15)
	victim := m.keys()[home]
	gop := m.GetOrPutAssumeCapacity(hmK(nk))
	expect(t, !gop.found_existing, "!gop.found_existing")
	expect(t, &m.keys()[home] == gop.key_ptr, "@intFromPtr(&m.keys()[home]) == @intFromPtr(gop.key_ptr)")
	expectEqual(t, nk, *gop.key_ptr)
	*gop.value_ptr = 5
	expectEqual(t, uint32(17), m.Count())
	expectEqual(t, hmUsedCtl(nk), hmCtl(m, home))
	expect(t, !m.Get(hmK(victim)).Valid, "m.get(term51XK(victim)) == null")
	for _, k := range hmFull16 {
		if k != victim {
			expect(t, m.Get(hmK(k)).Valid, "m.get(term51XK(k)) != null")
		}
	}
	expectEqual(t, uint16(5), zigUnwrap(t, m.Get(hmK(nk))))
}

// Zig: test "term51: HashMapUnmanaged.getOrPutValueContext: a new key is inserted with the given value"
func TestHashMap_term51_HashMapUnmanaged_getOrPutValueContext_a_new_key_is_inserted_with_the_given_value(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c[0:6] {
		e := hm2GetOrPutValue(t, m, hmK(k), HyperlinkID(i+10), hmCtx)
		expect(t, &m.keys()[5+i] == e.key_ptr, "@intFromPtr(&m.keys()[5 + i]) == @intFromPtr(e.key_ptr)")
		expect(t, &m.values()[5+i] == e.value_ptr, "@intFromPtr(&m.values()[5 + i]) == @intFromPtr(e.value_ptr)")
		expectEqual(t, k, *e.key_ptr)
		expectEqual(t, uint16(i+10), *e.value_ptr)
		expectEqual(t, uint32(i+1), m.Count())
		expectEqual(t, hmUsedCtl(k), hmCtl(m, uint(5+i)))
		expectEqual(t, uint16(i+10), zigUnwrap(t, m.Get(hmK(k))))
	}
	// The wrapper without a context.
	w, err := m.GetOrPutValue(hmK(c[6]), 77)
	zigTry(t, err)
	expectEqual(t, uint16(77), *w.value_ptr)
	expectEqual(t, uint32(7), m.Count())
}

// Zig: test "term51: HashMapUnmanaged.getOrPutValueContext: an existing entry is returned unchanged"
func TestHashMap_term51_HashMapUnmanaged_getOrPutValueContext_an_existing_entry_is_returned_unchanged(t *testing.T) {
	box := hmGBoxInit(16)
	m := box.map_()
	for i := range 4 {
		zigTry(t, m.Put(hmK(uint32(4096+8*i)), hmSl(uint32(i), uint(i+1))))
	}
	for i := range 4 {
		k := hmK(uint32(4096 + 8*i))
		e := hm2GetOrPutValue(t, m, k, hmSl(100, 100), hmCtx)
		hmExpectSl(t, hmSl(uint32(i), uint(i+1)), *e.value_ptr)
		expectEqual(t, k, *e.key_ptr)
		entry := zigUnwrap(t, m.GetEntry(k))
		expect(t, entry.key_ptr == e.key_ptr, "@intFromPtr(entry.key_ptr) == @intFromPtr(e.key_ptr)")
		expect(t, entry.value_ptr == e.value_ptr, "@intFromPtr(entry.value_ptr) == @intFromPtr(e.value_ptr)")
		w, err := m.GetOrPutValue(k, hmSl(7, 7))
		zigTry(t, err)
		expect(t, e.value_ptr == w.value_ptr, "@intFromPtr(e.value_ptr) == @intFromPtr(w.value_ptr)")
		hmExpectSl(t, hmSl(uint32(i), uint(i+1)), zigUnwrap(t, m.Get(k)))
	}
	expectEqual(t, uint32(4), m.Count())
	// The returned pointer writes the map's value.
	e := hm2GetOrPutValue(t, m, hmK(4096), hmSl(0, 0), hmCtx)
	*e.value_ptr = hmSl(hmDesc(*e.value_ptr).offset, 40) // e.value_ptr.len = 40
	expectEqual(t, uint(40), hmDesc(zigUnwrap(t, m.Get(hmK(4096)))).len)
}

// Zig: test "term51: HashMapUnmanaged.getOrPutValueContext: a full map returns existing entries and refuses new ones"
func TestHashMap_term51_HashMapUnmanaged_getOrPutValueContext_a_full_map_returns_existing_entries_and_refuses_new_ones(t *testing.T) {
	box := hmHBoxInit(4)
	m := box.map_()
	keys := [...]uint32{8, 16, 24, 32}
	for i, k := range keys {
		hm2GetOrPutValue(t, m, hmK(k), HyperlinkID(i+1), hmCtx)
	}
	expectEqual(t, uint32(4), m.Count())
	for i, k := range keys {
		e := hm2GetOrPutValue(t, m, hmK(k), 99, hmCtx)
		expectEqual(t, uint16(i+1), *e.value_ptr)
		expect(t, &m.keys()[zigUnwrap(t, hmSlot(m, k))] == e.key_ptr, "@intFromPtr(&m.keys()[term51XSlot(m, k).?]) == @intFromPtr(e.key_ptr)")
	}
	_, err := m.GetOrPutValueContext(hmK(40), 5, hmCtx)
	hmExpectError(t, ErrOutOfMemory, err)
	_, err = m.GetOrPutValue(hmK(48), 5)
	hmExpectError(t, ErrOutOfMemory, err)
	expectEqual(t, uint32(4), m.Count())
	expect(t, !m.Get(hmK(40)).Valid, "m.get(term51XK(40)) == null")
	for i, k := range keys {
		expectEqual(t, uint16(i+1), zigUnwrap(t, m.Get(hmK(k))))
	}
	// Room again after a removal.
	expect(t, m.Remove(hmK(16)), "m.remove(term51XK(16))")
	e := hm2GetOrPutValue(t, m, hmK(40), 5, hmCtx)
	expectEqual(t, uint16(5), *e.value_ptr)
	expectEqual(t, uint32(4), m.Count())

	z := hmHBoxInit(0)
	zm := z.map_()
	_, err = zm.GetOrPutValueContext(hmK(8), 1, hmCtx)
	hmExpectError(t, ErrOutOfMemory, err)
}

// Zig: test "term51: HashMapUnmanaged.getOrPutValueContext: tombstones are recycled and removed keys get the new value"
func TestHashMap_term51_HashMapUnmanaged_getOrPutValueContext_tombstones_are_recycled_and_removed_keys_get_the_new_value(t *testing.T) {
	box := hmHBoxInit(16)
	m := box.map_()
	c := hmChain5
	for i, k := range c[0:6] {
		zigTry(t, m.Put(hmK(k), HyperlinkID(i)))
	}
	_ = m.Remove(hmK(c[1]))
	_ = m.Remove(hmK(c[3]))
	// New key: first tombstone in probe order.
	e6 := hm2GetOrPutValue(t, m, hmK(c[6]), 60, hmCtx)
	expect(t, &m.keys()[6] == e6.key_ptr, "@intFromPtr(&m.keys()[6]) == @intFromPtr(e6.key_ptr)")
	expectEqual(t, uint16(60), *e6.value_ptr)
	// Existing key behind a tombstone: its own value.
	e4 := hm2GetOrPutValue(t, m, hmK(c[4]), 40, hmCtx)
	expect(t, &m.keys()[9] == e4.key_ptr, "@intFromPtr(&m.keys()[9]) == @intFromPtr(e4.key_ptr)")
	expectEqual(t, uint16(4), *e4.value_ptr)
	// A removed key is new: it gets the value and the other tombstone.
	e1 := hm2GetOrPutValue(t, m, hmK(c[1]), 11, hmCtx)
	expect(t, &m.keys()[8] == e1.key_ptr, "@intFromPtr(&m.keys()[8]) == @intFromPtr(e1.key_ptr)")
	expectEqual(t, uint16(11), zigUnwrap(t, m.Get(hmK(c[1]))))
	expectEqual(t, uint32(6), m.Count())
	expect(t, !m.Get(hmK(c[3])).Valid, "m.get(term51XK(c[3])) == null")
}

// Zig: test "term51: HashMapUnmanaged.getOrPutValueContext: the stored key is kept, a new entry stores the given key"
func TestHashMap_term51_HashMapUnmanaged_getOrPutValueContext_the_stored_key_is_kept_a_new_entry_stores_the_given_key(t *testing.T) {
	box := hm2LBoxInit(16)
	m := box.map_()
	a := hm2GetOrPutValue(t, m, hmK(4100), 1, hm2Loose)
	expectEqual(t, uint32(4100), *a.key_ptr)
	b := hm2GetOrPutValue(t, m, hmK(4097), 2, hm2Loose)
	expectEqual(t, uint32(4100), *b.key_ptr)
	expectEqual(t, uint16(1), *b.value_ptr)
	expect(t, a.key_ptr == b.key_ptr, "@intFromPtr(a.key_ptr) == @intFromPtr(b.key_ptr)")
	expectEqual(t, uint32(1), m.Count())
	c, err := m.GetOrPutValue(hmK(4113), 3)
	zigTry(t, err)
	expectEqual(t, uint32(4113), *c.key_ptr)
	expectEqual(t, uint32(2), m.Count())
}

// Go-only: panic test: fetchPutContext, getOrPutAdapted and getOrPutValueContext: the map has no metadata
func TestHashMap_panic_fetchPutContext_getOrPutAdapted_and_getOrPutValueContext_the_map_has_no_metadata(t *testing.T) {
	// Zig: getOrPutContextAdapted starts with growIfNeeded, which reads
	// self.header().size: self.metadata.? is null in a default map.
	var m hmHU
	hm2ExpectPanic(t, func() { _, _ = m.FetchPutContext(hmK(1), 1, hmCtx) })
	hm2ExpectPanic(t, func() { _, _ = m.FetchPut(hmK(1), 1) })
	hm2ExpectPanic(t, func() { _, _ = m.GetOrPutAdapted(1, hmCtx) })
	hm2ExpectPanic(t, func() { _, _ = m.GetOrPutValueContext(hmK(1), 1, hmCtx) })
	hm2ExpectPanic(t, func() { _, _ = m.GetOrPutValue(hmK(1), 1) })
}

// Go-only: panic test: fetchRemoveContext, fetchRemoveAdapted, getKeyPtrContext, getKeyPtrAdapted, getKey, getKeyContext, getKeyAdapted: the map has no metadata
func TestHashMap_panic_fetchRemove_and_getKey_functions_the_map_has_no_metadata(t *testing.T) {
	// Zig: all of them go through getIndex, which reads self.header().size
	// first: self.metadata.? is null in a default map.
	var m hmHU
	hm2ExpectPanic(t, func() { m.FetchRemoveContext(hmK(1), hmCtx) })
	hm2ExpectPanic(t, func() { m.FetchRemove(hmK(1)) })
	hm2ExpectPanic(t, func() { m.FetchRemoveAdapted(1, hmCtx) })
	hm2ExpectPanic(t, func() { m.GetKeyPtrContext(hmK(1), hmCtx) })
	hm2ExpectPanic(t, func() { m.GetKeyPtr(hmK(1)) })
	hm2ExpectPanic(t, func() { m.GetKeyPtrAdapted(1, hmCtx) })
	hm2ExpectPanic(t, func() { m.GetKey(hmK(1)) })
	hm2ExpectPanic(t, func() { m.GetKeyContext(hmK(1), hmCtx) })
	hm2ExpectPanic(t, func() { m.GetKeyAdapted(1, hmCtx) })
}

// Go-only: panic test: fetchPutAssumeCapacityContext and getOrPutAssumeCapacity: capacity 0 or no metadata (capacity() - 1 overflows)
func TestHashMap_panic_fetchPutAssumeCapacityContext_and_getOrPutAssumeCapacity_capacity_0_or_no_metadata_capacity_1_overflows(t *testing.T) {
	// Zig: const mask = self.capacity() - 1; is a u32 subtraction that
	// overflows when the capacity is 0 (a zero-capacity map and a default map).
	box := hmHBoxInit(0)
	m := box.map_()
	hm2ExpectPanic(t, func() { m.FetchPutAssumeCapacityContext(hmK(1), 1, hmCtx) })
	hm2ExpectPanic(t, func() { m.GetOrPutAssumeCapacity(hmK(1)) })
	hm2ExpectPanic(t, func() { m.GetOrPutAssumeCapacityContext(hmK(1), hmCtx) })
	var d hmHU
	hm2ExpectPanic(t, func() { d.FetchPutAssumeCapacityContext(hmK(1), 1, hmCtx) })
	hm2ExpectPanic(t, func() { d.GetOrPutAssumeCapacity(hmK(1)) })
}

// Go-only: panic test: growIfNeeded: the size passed the capacity through a full map (capacity() - size underflows)
func TestHashMap_panic_growIfNeeded_the_size_passed_the_capacity_through_a_full_map_capacity_size_underflows(t *testing.T) {
	// Zig: a new key put by an assume-capacity function into a full map
	// overwrites its home slot and increments the size past the capacity (see
	// the two "overwrites its home slot" tests). The next growIfNeeded computes
	// `self.capacity() - self.header().size` on u32 values: 16 - 17 overflows.
	box := hmHBoxInit(16)
	m := box.map_()
	hm2Fill16(t, m)
	m.GetOrPutAssumeCapacity(hmK(65664))
	expectEqual(t, uint32(17), m.Count())
	hm2ExpectPanic(t, func() { _ = m.Put(hmK(1), 1) })
	hm2ExpectPanic(t, func() { _, _ = m.FetchPut(hmK(1), 1) })
	hm2ExpectPanic(t, func() { _, _ = m.GetOrPutAdapted(1, hmCtx) })
	hm2ExpectPanic(t, func() { _ = m.EnsureTotalCapacity(math.MaxUint32) })
}
