// Stand-ins for Zig language and standard-library features that Go lacks.
// Each one is listed in docs/mapping.md. Nothing here is ported from a
// Ghostty source file.

package terminal

import "math/bits"

// Optional is Zig's optional type `?T` for the places where an optional is
// stored in a data structure or returned alongside other values. A null
// optional is the zero value.
type Optional[T any] struct {
	Value T
	Valid bool
}

// Some returns an Optional holding v.
func Some[T any](v T) Optional[T] {
	return Optional[T]{Value: v, Valid: true}
}

// None returns the null Optional.
func None[T any]() Optional[T] {
	return Optional[T]{}
}

// assert is Zig's assert (quirks.inlineAssert, std.debug.assert): it stops
// the program when cond is false. Zig also traps on out-of-range indexes,
// integer overflow and `unreachable`; wherever Go would silently continue in
// one of those situations the ported code calls assert (or panics with
// "unreachable") so that the port stops exactly where upstream would.
func assert(cond bool) {
	if !cond {
		panic("assertion failed")
	}
}

// StaticBitSet24 is std.StaticBitSet(24): a fixed-size bit set backed by one
// integer. Only the methods the ported code uses are provided, with Zig's
// names.
type StaticBitSet24 struct {
	mask uint32
}

// StaticBitSet24BitLength is std.StaticBitSet(24).bit_length.
const StaticBitSet24BitLength = 24

// initEmpty returns a set with no bits set.
func StaticBitSet24InitEmpty() StaticBitSet24 {
	return StaticBitSet24{}
}

// isSet reports whether the bit at index is set. Like Zig's, it requires
// index < bit_length.
func (self StaticBitSet24) isSet(index int) bool {
	assert(index >= 0 && index < StaticBitSet24BitLength)
	return self.mask&(1<<uint(index)) != 0
}

// set sets the bit at index.
func (self *StaticBitSet24) set(index int) {
	assert(index >= 0 && index < StaticBitSet24BitLength)
	self.mask |= 1 << uint(index)
}

// count returns the number of set bits.
func (self StaticBitSet24) count() int {
	return bits.OnesCount32(self.mask)
}

// logWarn is the hook behind Zig's scoped `log.warn`. It is nil by default,
// so logging costs nothing; tests or an embedding program may set it.
var logWarn func(scope string, format string, args ...any)

func warn(scope string, format string, args ...any) {
	if logWarn != nil {
		logWarn(scope, format, args...)
	}
}

// saturatingMulU16 is Zig's `*|` on u16.
func saturatingMulU16(a, b uint16) uint16 {
	r := uint32(a) * uint32(b)
	if r > 0xFFFF {
		return 0xFFFF
	}
	return uint16(r)
}

// saturatingAddU16 is Zig's `+|` on u16.
func saturatingAddU16(a, b uint16) uint16 {
	r := uint32(a) + uint32(b)
	if r > 0xFFFF {
		return 0xFFFF
	}
	return uint16(r)
}

// zigFmtAnySlice renders a slice the way Zig's std.fmt "{any}" does:
// "{ 1, 22 }", and "{  }" when empty.
func zigFmtAnySlice[T uint8 | uint16](s []T) string {
	out := "{ "
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += itoa(uint64(v))
	}
	return out + " }"
}

// zigFmtUnicode renders a codepoint the way Zig's std.fmt "{u}" does: the
// UTF-8 encoding of the codepoint.
func zigFmtUnicode(cp uint32) string {
	return string(rune(cp))
}

func itoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
