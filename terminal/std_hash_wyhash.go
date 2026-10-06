// Port of Zig 0.15.2's std.hash.Wyhash (lib/std/hash/wyhash.zig), the
// toolchain that built the recorded corpus. Nothing here is ported from a
// Ghostty source file. Callers: hyperlink.PageEntry.hash (Wyhash.init,
// update through autoHash, final) and hash_map.AutoContext.hash
// (Wyhash.hash with seed 0 over the key's bytes); see docs/mapping.md,
// section "Zig std hashing: `std.hash.Wyhash`, `std.hash.int`,
// `std.hash.autoHash`".
//
// The Go standard library has no Wyhash (hash/maphash is a different,
// randomly seeded function), so the whole of wyhash.zig's Wyhash is ported,
// function by function in the file's order.
//
// Names. Every declaration carries the prefix stdHashWyhash. The struct
// Wyhash is the type stdHashWyhash; its methods keep Zig's names in lower
// case because the type is package-private (`update`, `final`, `smallKey`,
// `round`, `final0`, ...). Functions without `self` follow the `<Type>Init`
// rule: `init` is stdHashWyhashInit, `hash` stdHashWyhashHash, `read`
// stdHashWyhashRead, `mum` stdHashWyhashMum, `mix` stdHashWyhashMix. The
// file-level constant `secret` is stdHashWyhashSecret.
//
// Widths. `total_len` is usize in Zig and enters the result (final2): it is
// a uint64 here, the width of the 64-bit build that recorded the data, on
// every platform. `buf_len` and the local indexes are Go `uint` (they never
// exceed 48 or the input length). `@as(u128, a) *% b` in `mum` is
// math/bits.Mul64 (the full 128-bit product, which cannot wrap).
//
// `std.mem.readInt(T, ..., .little)` is encoding/binary.LittleEndian.
// `undefined` fields (a, b, state in init; buf and buf_len in shallowCopy)
// are Go zero values; they are always written before they are read.

package terminal

import (
	"encoding/binary"
	"math/bits"
)

// stdHashWyhash is std.hash.Wyhash.
type stdHashWyhash struct {
	a     uint64
	b     uint64
	state [3]uint64

	// Zig: usize. 64 bits on every platform (see the file header).
	total_len uint64

	buf [48]uint8

	// Zig: usize.
	buf_len uint
}

// stdHashWyhashSecret is Wyhash.secret.
var stdHashWyhashSecret = [4]uint64{
	0xa0761d6478bd642f,
	0xe7037ed1a0b428db,
	0x8ebc6af09c88c6e3,
	0x589965cc75374cc3,
}

// stdHashWyhashInit is Wyhash.init.
func stdHashWyhashInit(seed uint64) stdHashWyhash {
	// Zig: a, b, state and buf are undefined; Go zeroes them.
	self := stdHashWyhash{
		total_len: 0,
		buf_len:   0,
	}

	self.state[0] = seed ^ stdHashWyhashMix(seed^stdHashWyhashSecret[0], stdHashWyhashSecret[1])
	self.state[1] = self.state[0]
	self.state[2] = self.state[0]
	return self
}

// update is Wyhash.update.
//
// This is subtly different from other hash function update calls. Wyhash requires the last
// full 48-byte block to be run through final1 if is exactly aligned to 48-bytes.
func (self *stdHashWyhash) update(input []uint8) {
	// Zig: usize addition; 2^64 bytes cannot be hashed, so it cannot trap.
	self.total_len += uint64(len(input))

	if uint(len(input)) <= 48-self.buf_len {
		copy(self.buf[self.buf_len:][:len(input)], input)
		self.buf_len += uint(len(input))
		return
	}

	var i uint = 0

	if self.buf_len > 0 {
		i = 48 - self.buf_len
		copy(self.buf[self.buf_len:][:i], input[0:i])
		self.round(&self.buf)
		self.buf_len = 0
	}

	for i+48 < uint(len(input)) {
		self.round((*[48]uint8)(input[i:][:48]))
		i += 48
	}

	remaining_bytes := input[i:]
	if len(remaining_bytes) < 16 && i >= 48 {
		rem := 16 - uint(len(remaining_bytes))
		copy(self.buf[uint(len(self.buf))-rem:], input[i-rem:i])
	}
	copy(self.buf[0:len(remaining_bytes)], remaining_bytes)
	self.buf_len = uint(len(remaining_bytes))
}

// final is Wyhash.final.
func (self *stdHashWyhash) final() uint64 {
	var input []uint8 = self.buf[0:self.buf_len]
	newSelf := self.shallowCopy() // ensure idempotency

	if self.total_len <= 16 {
		newSelf.smallKey(input)
	} else {
		var offset uint = 0
		var scratch [16]uint8
		if self.buf_len < 16 {
			rem := 16 - self.buf_len
			copy(scratch[0:rem], self.buf[uint(len(self.buf))-rem:][:rem])
			copy(scratch[rem:][:self.buf_len], self.buf[0:self.buf_len])

			// Same as input but with additional bytes preceding start in case of a short buffer
			input = scratch[:]
			offset = rem
		}

		newSelf.final0()
		newSelf.final1(input, offset)
	}

	return newSelf.final2()
}

// shallowCopy is Wyhash.shallowCopy.
//
// Copies the core wyhash state but not any internal buffers.
func (self *stdHashWyhash) shallowCopy() stdHashWyhash {
	// Zig: buf and buf_len are undefined; Go zeroes them.
	return stdHashWyhash{
		a:         self.a,
		b:         self.b,
		state:     self.state,
		total_len: self.total_len,
	}
}

// smallKey is Wyhash.smallKey.
func (self *stdHashWyhash) smallKey(input []uint8) {
	assert(len(input) <= 16)

	if len(input) >= 4 {
		end := uint(len(input)) - 4
		quarter := (uint(len(input)) >> 3) << 2
		self.a = (stdHashWyhashRead(4, input[0:]) << 32) | stdHashWyhashRead(4, input[quarter:])
		self.b = (stdHashWyhashRead(4, input[end:]) << 32) | stdHashWyhashRead(4, input[end-quarter:])
	} else if len(input) > 0 {
		self.a = (uint64(input[0]) << 16) | (uint64(input[len(input)>>1]) << 8) | uint64(input[len(input)-1])
		self.b = 0
	} else {
		self.a = 0
		self.b = 0
	}
}

// round is Wyhash.round.
func (self *stdHashWyhash) round(input *[48]uint8) {
	// Zig: inline for (0..3).
	for i := 0; i < 3; i++ {
		a := stdHashWyhashRead(8, input[8*(2*i):])
		b := stdHashWyhashRead(8, input[8*(2*i+1):])
		self.state[i] = stdHashWyhashMix(a^stdHashWyhashSecret[i+1], b^self.state[i])
	}
}

// stdHashWyhashRead is Wyhash.read: the first `bytes` bytes of data as a
// little-endian unsigned integer. `bytes` is comptime in Zig; it is 4 or 8
// at every call.
func stdHashWyhashRead(bytes uint, data []uint8) uint64 {
	assert(bytes <= 8)
	// Zig: data[0..bytes] traps if data is shorter; so does the Go slice.
	switch bytes {
	case 4:
		return uint64(binary.LittleEndian.Uint32(data[0:bytes]))
	case 8:
		return binary.LittleEndian.Uint64(data[0:bytes])
	default:
		// std.meta.Int(.unsigned, 8 * bytes) for the other widths.
		var v uint64 = 0
		for i, c := range data[0:bytes] {
			v |= uint64(c) << (8 * uint(i))
		}
		return v
	}
}

// stdHashWyhashMum is Wyhash.mum: the 128-bit product of *a and *b, low
// half to *a and high half to *b.
func stdHashWyhashMum(a *uint64, b *uint64) {
	hi, lo := bits.Mul64(*a, *b)
	*a = lo
	*b = hi
}

// stdHashWyhashMix is Wyhash.mix.
func stdHashWyhashMix(a_ uint64, b_ uint64) uint64 {
	a := a_
	b := b_
	stdHashWyhashMum(&a, &b)
	return a ^ b
}

// final0 is Wyhash.final0.
func (self *stdHashWyhash) final0() {
	self.state[0] ^= self.state[1] ^ self.state[2]
}

// final1 is Wyhash.final1.
//
// input_lb must be at least 16-bytes long (in shorter key cases the smallKey function will be
// used instead). We use an index into a slice to for comptime processing as opposed to if we
// used pointers.
func (self *stdHashWyhash) final1(input_lb []uint8, start_pos uint) {
	assert(len(input_lb) >= 16)
	// Zig: `input_lb.len - start_pos` traps if start_pos > input_lb.len.
	assert(start_pos <= uint(len(input_lb)))
	assert(uint(len(input_lb))-start_pos <= 48)
	input := input_lb[start_pos:]

	var i uint = 0
	for i+16 < uint(len(input)) {
		self.state[0] = stdHashWyhashMix(stdHashWyhashRead(8, input[i:])^stdHashWyhashSecret[1], stdHashWyhashRead(8, input[i+8:])^self.state[0])
		i += 16
	}

	self.a = stdHashWyhashRead(8, input_lb[len(input_lb)-16:][:8])
	self.b = stdHashWyhashRead(8, input_lb[len(input_lb)-8:][:8])
}

// final2 is Wyhash.final2.
func (self *stdHashWyhash) final2() uint64 {
	self.a ^= stdHashWyhashSecret[1]
	self.b ^= self.state[0]
	stdHashWyhashMum(&self.a, &self.b)
	return stdHashWyhashMix(self.a^stdHashWyhashSecret[0]^self.total_len, self.b^stdHashWyhashSecret[1])
}

// stdHashWyhashHash is Wyhash.hash: the one-shot hash of input.
func stdHashWyhashHash(seed uint64, input []uint8) uint64 {
	self := stdHashWyhashInit(seed)

	if len(input) <= 16 {
		self.smallKey(input)
	} else {
		var i uint = 0
		if len(input) >= 48 {
			for i+48 < uint(len(input)) {
				self.round((*[48]uint8)(input[i:][:48]))
				i += 48
			}
			self.final0()
		}
		self.final1(input, i)
	}

	self.total_len = uint64(len(input))
	return self.final2()
}
