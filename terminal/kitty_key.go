// WIREFRAME for src/terminal/kitty/key.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// Kitty keyboard protocol support.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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
func (self KittyKeyFlagStack) Current() KittyKeyFlags { panic(notImplemented) }

// Set is kitty.key.FlagStack.set: perform the "set" operation as described
// in the spec for the CSI = u sequence.
func (self *KittyKeyFlagStack) Set(mode KittyKeySetMode, v KittyKeyFlags) {
	panic(notImplemented)
}

// Push is kitty.key.FlagStack.push: push a new set of flags onto the stack.
// If the stack is full then the oldest entry is evicted.
func (self *KittyKeyFlagStack) Push(flags KittyKeyFlags) { panic(notImplemented) }

// Pop is kitty.key.FlagStack.pop: pop `n` entries from the stack. This will
// just wrap around if `n` is greater than the amount in the stack.
//
// Zig: n is usize.
func (self *KittyKeyFlagStack) Pop(n uint) { panic(notImplemented) }

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
// Zig: returns u5.
func (self KittyKeyFlags) Int() uint8 { panic(notImplemented) }

// KittyKeyFlagsFromInt is Zig's `@bitCast` from u5 to Flags, the inverse of
// Int. Zig has no named function for it; FlagStack.set and stream.zig (the
// CSI u handlers) use the cast directly.
func KittyKeyFlagsFromInt(v uint8) KittyKeyFlags { panic(notImplemented) }

// KittyKeySetMode is kitty.key.SetMode: the possible modes for setting the
// key flags.
type KittyKeySetMode uint8

const (
	KittyKeySetModeSet KittyKeySetMode = iota
	KittyKeySetModeOr
	KittyKeySetModeNot
)

// String returns the Zig tag name (@tagName).
func (self KittyKeySetMode) String() string { panic(notImplemented) }
