// Tests translated one for one from the two tests nested in the types
// CsiUMods and KittyMods of src/input/key_encode.zig (lines 855 and 938; both
// are titled "modifier sequence values"). They sit before the first
// top-level test of the file. Each Go test carries the exact Zig title in a
// "// Zig: test" comment. Helpers are prefixed ke0.

package terminal

import "testing"

// Zig: test "modifier sequence values"
func TestKe0_modifier_sequence_values_CsiUMods(t *testing.T) {
	// This is all sort of trivially seen by looking at the code but
	// we want to make sure we never regress this.
	mods := inputKeyEncodeCsiUMods{}
	expectEqual(t, uint8(1), mods.SeqInt())

	mods = inputKeyEncodeCsiUMods{shift: true}
	expectEqual(t, uint8(2), mods.SeqInt())

	mods = inputKeyEncodeCsiUMods{alt: true}
	expectEqual(t, uint8(3), mods.SeqInt())

	mods = inputKeyEncodeCsiUMods{ctrl: true}
	expectEqual(t, uint8(5), mods.SeqInt())

	mods = inputKeyEncodeCsiUMods{alt: true, shift: true}
	expectEqual(t, uint8(4), mods.SeqInt())

	mods = inputKeyEncodeCsiUMods{ctrl: true, shift: true}
	expectEqual(t, uint8(6), mods.SeqInt())

	mods = inputKeyEncodeCsiUMods{alt: true, ctrl: true}
	expectEqual(t, uint8(7), mods.SeqInt())

	mods = inputKeyEncodeCsiUMods{alt: true, ctrl: true, shift: true}
	expectEqual(t, uint8(8), mods.SeqInt())
}

// Zig: test "modifier sequence values"
func TestKe0_modifier_sequence_values_KittyMods(t *testing.T) {
	// This is all sort of trivially seen by looking at the code but
	// we want to make sure we never regress this.
	mods := inputKeyEncodeKittyMods{}
	expectEqual(t, uint16(1), mods.SeqInt())

	mods = inputKeyEncodeKittyMods{shift: true}
	expectEqual(t, uint16(2), mods.SeqInt())

	mods = inputKeyEncodeKittyMods{alt: true}
	expectEqual(t, uint16(3), mods.SeqInt())

	mods = inputKeyEncodeKittyMods{ctrl: true}
	expectEqual(t, uint16(5), mods.SeqInt())

	mods = inputKeyEncodeKittyMods{alt: true, shift: true}
	expectEqual(t, uint16(4), mods.SeqInt())

	mods = inputKeyEncodeKittyMods{ctrl: true, shift: true}
	expectEqual(t, uint16(6), mods.SeqInt())

	mods = inputKeyEncodeKittyMods{alt: true, ctrl: true}
	expectEqual(t, uint16(7), mods.SeqInt())

	mods = inputKeyEncodeKittyMods{alt: true, ctrl: true, shift: true}
	expectEqual(t, uint16(8), mods.SeqInt())
}
