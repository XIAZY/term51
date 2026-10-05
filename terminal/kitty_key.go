// Port of src/terminal/kitty/key.zig (Ghostty v1.3.1).
//
// Kitty keyboard protocol support.
//
// File naming: a Zig file in a subdirectory, `kitty/key.zig`, becomes
// `kitty_key.go` in package terminal, and its declarations take the prefix
// `KittyKey`. Declarations appear in the same order as in key.zig. The file
// has no C ABI declarations.
//
// kitty.zig has no Go file. Everything it declares that survives in the
// library build is a re-export, and each re-export already has the same Go
// name as the thing it re-exports: `kitty.KeyFlags` (key.Flags),
// `kitty.KeyFlagStack` (key.FlagStack) and `kitty.KeySetMode` (key.SetMode)
// are KittyKeyFlags, KittyKeyFlagStack and KittyKeySetMode below;
// `kitty.color` is the KittyColor... names of kitty_color.go;
// `kitty.graphics` is an empty struct because build_options.kitty_graphics
// is false.
//
// Odd-width integers: FlagStack.idx is a u3 and Flags a packed u5. Both are
// uint8 in Go. `idx +%= 1` / `idx -%= 1` wrap modulo 8 and are written as
// the uint8 operation masked with 7. The packed integer of Flags never has
// bits above bit 4: Int builds it from the five bools, and
// KittyKeyFlagsFromInt asserts its argument fits in a u5 (a larger value
// cannot exist in Zig).

package terminal

// KittyKeyFlagStack is kitty.key.FlagStack: stack for the key flags. This
// implements the push/pop behavior of the CSI > u and CSI < u sequences. We
// implement the stack as fixed size to avoid heap allocation.
//
// Zig gives every field a default: `flags = @splat(.disabled)` and
// `idx = 0`. The zero value of each field is the same state.
type KittyKeyFlagStack struct {
	flags [kittyKeyFlagStackLen]KittyKeyFlags
	// Zig: u3, which wraps at 8 (`+%=`, `-%=`).
	idx uint8
}

// kittyKeyFlagStackLen is kitty.key.FlagStack.len.
const kittyKeyFlagStackLen = 8

// Current is kitty.key.FlagStack.current: return the current stack value.
func (self KittyKeyFlagStack) Current() KittyKeyFlags {
	return self.flags[self.idx]
}

// Set is kitty.key.FlagStack.set: perform the "set" operation as described
// in the spec for the CSI = u sequence.
func (self *KittyKeyFlagStack) Set(mode KittyKeySetMode, v KittyKeyFlags) {
	switch mode {
	case KittyKeySetModeSet:
		self.flags[self.idx] = v
	case KittyKeySetModeOr:
		self.flags[self.idx] = KittyKeyFlagsFromInt(
			self.flags[self.idx].Int() | v.Int(),
		)
	case KittyKeySetModeNot:
		// Zig's `~` is on a u5; Go's on a uint8 also sets bits 5 to 7,
		// which the `&` with a u5 value clears again.
		self.flags[self.idx] = KittyKeyFlagsFromInt(
			self.flags[self.idx].Int() & ^v.Int(),
		)
	default:
		// Zig's switch is exhaustive; a value outside the enum traps.
		panic("unreachable")
	}
}

// Push is kitty.key.FlagStack.push: push a new set of flags onto the stack.
// If the stack is full then the oldest entry is evicted.
func (self *KittyKeyFlagStack) Push(flags KittyKeyFlags) {
	// Overflow and wrap around if we're full, which evicts
	// the oldest entry.
	self.idx = (self.idx + 1) & 7 // Zig: self.idx +%= 1 on a u3
	self.flags[self.idx] = flags
}

// Pop is kitty.key.FlagStack.pop: pop `n` entries from the stack. This will
// just wrap around if `n` is greater than the amount in the stack.
//
// Zig: n is usize.
func (self *KittyKeyFlagStack) Pop(n uint) {
	// If n is more than our length then we just reset the stack.
	// This also avoids a DoS vector where a malicious client
	// could send a huge number of pop commands to waste cpu.
	if n >= uint(len(self.flags)) {
		self.idx = 0
		for i := range self.flags {
			self.flags[i] = KittyKeyFlagsDisabled
		}
		return
	}

	for range n {
		self.flags[self.idx] = KittyKeyFlagsDisabled
		self.idx = (self.idx - 1) & 7 // Zig: self.idx -%= 1 on a u3
	}
}

// KittyKeyFlags is kitty.key.Flags: the possible flags for the Kitty
// keyboard protocol.
//
// Zig: packed struct(u5), every field default false. In Go it is an
// ordinary struct; Int and KittyKeyFlagsFromInt are the conversions to and
// from the packed integer.
type KittyKeyFlags struct {
	disambiguate      bool
	report_events     bool
	report_alternates bool
	report_all        bool
	report_associated bool
}

// KittyKeyFlagsDisabled is kitty.key.Flags.disabled: Kitty keyboard protocol
// disabled (all flags off). The zero value is already that state.
var KittyKeyFlagsDisabled = KittyKeyFlags{}

// KittyKeyFlagsTrue is kitty.key.Flags.true: sets all modes on.
//
// A constant value, so it is given here; it is not logic.
var KittyKeyFlagsTrue = KittyKeyFlags{
	disambiguate:      true,
	report_events:     true,
	report_alternates: true,
	report_all:        true,
	report_associated: true,
}

// Int is kitty.key.Flags.int: the packed integer, disambiguate in bit 0
// through report_associated in bit 4.
//
// Zig: returns u5, `@bitCast(self)`.
func (self KittyKeyFlags) Int() uint8 {
	var v uint8 = 0
	if self.disambiguate {
		v |= 1 << 0
	}
	if self.report_events {
		v |= 1 << 1
	}
	if self.report_alternates {
		v |= 1 << 2
	}
	if self.report_all {
		v |= 1 << 3
	}
	if self.report_associated {
		v |= 1 << 4
	}
	return v
}

// KittyKeyFlagsFromInt is Zig's `@bitCast` from u5 to Flags, the inverse of
// Int. Zig has no named function for it; FlagStack.set and stream.zig (the
// CSI u handlers) use the cast directly.
//
// The argument is a u5 in Zig; a Go value above 31 is asserted against.
func KittyKeyFlagsFromInt(v uint8) KittyKeyFlags {
	assert(v <= 0x1F)
	return KittyKeyFlags{
		disambiguate:      v&(1<<0) != 0,
		report_events:     v&(1<<1) != 0,
		report_alternates: v&(1<<2) != 0,
		report_all:        v&(1<<3) != 0,
		report_associated: v&(1<<4) != 0,
	}
}

// KittyKeySetMode is kitty.key.SetMode: the possible modes for setting the
// key flags.
type KittyKeySetMode uint8

const (
	KittyKeySetModeSet KittyKeySetMode = iota
	KittyKeySetModeOr
	KittyKeySetModeNot
)

// kittyKeySetModeNames are the tag names of KittyKeySetMode, in order.
var kittyKeySetModeNames = [...]string{"set", "or", "not"}

// String returns the Zig tag name (@tagName).
func (self KittyKeySetMode) String() string { return kittyKeySetModeNames[self] }
