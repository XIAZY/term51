// Ported from src/terminal/UTF8Decoder.zig (Ghostty v1.3.1).

// DFA-based non-allocating error-replacing UTF-8 decoder.
//
// This implementation is based largely on the excellent work of
// Bjoern Hoehrmann, with slight modifications to support error-
// replacement.
//
// For details on Bjoern's DFA-based UTF-8 decoder, see
// http://bjoern.hoehrmann.de/utf-8/decoder/dfa (MIT licensed)

package terminal

var char_classes = [256]uint8{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	8, 8, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	10, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 3, 3, 11, 6, 6, 6, 5, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
}

var transitions = [108]uint8{
	0, 12, 24, 36, 60, 96, 84, 12, 12, 12, 48, 72, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12,
	12, 0, 12, 12, 12, 12, 12, 0, 12, 0, 12, 12, 12, 24, 12, 12, 12, 12, 12, 24, 12, 24, 12, 12,
	12, 12, 12, 12, 12, 12, 12, 24, 12, 12, 12, 12, 12, 24, 12, 12, 12, 12, 12, 12, 12, 24, 12, 12,
	12, 12, 12, 12, 12, 12, 12, 36, 12, 36, 12, 12, 12, 36, 12, 12, 12, 12, 12, 36, 12, 36, 12, 12,
	12, 36, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12,
}

// DFA states
const ACCEPT_STATE = 0
const REJECT_STATE = 12

// UTF8Decoder is the decoder state. The zero value is ready to use.
type UTF8Decoder struct {
	// This is where we accumulate our current codepoint.
	// Zig: u21.
	accumulator uint32
	// The internal state of the DFA.
	state uint8
}

// Next takes the next byte in the utf-8 sequence and emits a tuple of
//   - The codepoint that was generated, if there is one.
//   - A boolean that indicates whether the provided byte was consumed.
//
// The only case where the byte is not consumed is if an ill-formed
// sequence is reached, in which case a replacement character will be
// emitted and the byte will not be consumed.
//
// If the byte is not consumed, the caller is responsible for calling
// again with the same byte before continuing.
//
// Zig returns struct { ?u21, bool }; the optional codepoint is the
// Optional[uint32] here.
func (self *UTF8Decoder) Next(byte_ uint8) (Optional[uint32], bool) {
	char_class := char_classes[byte_]

	initial_state := self.state

	if self.state != ACCEPT_STATE {
		self.accumulator <<= 6
		// Zig's accumulator is a u21: bits shifted past bit 20 are lost.
		self.accumulator &= 0x1FFFFF
		self.accumulator |= uint32(byte_ & 0x3F)
	} else {
		self.accumulator = (uint32(0xFF) >> char_class) & uint32(byte_)
	}

	self.state = transitions[self.state+char_class]

	if self.state == ACCEPT_STATE {
		cp := self.accumulator
		self.accumulator = 0

		// Emit the fully decoded codepoint.
		return Some(cp), true
	} else if self.state == REJECT_STATE {
		self.accumulator = 0
		self.state = ACCEPT_STATE
		// Emit a replacement character. If we rejected the first byte
		// in a sequence, then it was consumed, otherwise it was not.
		return Some[uint32](0xFFFD), initial_state == ACCEPT_STATE
	} else {
		// Emit nothing, we're in the middle of a sequence.
		return None[uint32](), true
	}
}
