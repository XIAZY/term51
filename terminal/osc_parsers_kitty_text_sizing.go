// Ported from src/terminal/osc/parsers/kitty_text_sizing.zig (Ghostty
// v1.3.1).
//
// Kitty's text sizing protocol (OSC 66)
// Specification: https://sw.kovidgoyal.net/kitty/text-sizing-protocol/
//
// File naming: `osc/parsers/kitty_text_sizing.zig` becomes
// `osc_parsers_kitty_text_sizing.go` in package terminal, and its
// declarations take the prefix `OSCParsersKittyTextSizing`.
//
// Not ported, see docs/mapping.md:
//   - the C ABI declarations `OSC.C` (void) and `OSC.cval`, the C variants
//     of `VAlign` and `HAlign` (lib.Enum with the .c target), and
//     `lib_target`, which only selects between the two;
//   - `const log = std.log.scoped(.kitty_text_sizing)`: the port uses
//     warn("kitty_text_sizing", ...).

package terminal

import (
	"bytes"
	"errors"
)

// ErrUnknownKey is Zig's error.UnknownKey (kitty_text_sizing.OSC.update).
var ErrUnknownKey = errors.New("UnknownKey")

// ErrInvalidValue is Zig's error.InvalidValue (kitty_text_sizing.OSC.update).
var ErrInvalidValue = errors.New("InvalidValue")

// OSCParsersKittyTextSizingMaxPayloadLength is
// osc.parsers.kitty_text_sizing.max_payload_length.
const OSCParsersKittyTextSizingMaxPayloadLength = 4096

// OSCParsersKittyTextSizingVAlign is osc.parsers.kitty_text_sizing.VAlign.
//
// Zig: lib.Enum with the .zig target in the library build, i.e. a plain enum
// with these values in this order.
type OSCParsersKittyTextSizingVAlign uint8

const (
	OSCParsersKittyTextSizingVAlignTop    OSCParsersKittyTextSizingVAlign = 0
	OSCParsersKittyTextSizingVAlignBottom OSCParsersKittyTextSizingVAlign = 1
	OSCParsersKittyTextSizingVAlignCenter OSCParsersKittyTextSizingVAlign = 2
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersKittyTextSizingVAlign) String() string {
	return oscParsersKittyTextSizingVAlignNames[self]
}

var oscParsersKittyTextSizingVAlignNames = [...]string{
	OSCParsersKittyTextSizingVAlignTop:    "top",
	OSCParsersKittyTextSizingVAlignBottom: "bottom",
	OSCParsersKittyTextSizingVAlignCenter: "center",
}

// OSCParsersKittyTextSizingHAlign is osc.parsers.kitty_text_sizing.HAlign.
//
// Zig: lib.Enum with the .zig target in the library build, i.e. a plain enum
// with these values in this order.
type OSCParsersKittyTextSizingHAlign uint8

const (
	OSCParsersKittyTextSizingHAlignLeft   OSCParsersKittyTextSizingHAlign = 0
	OSCParsersKittyTextSizingHAlignRight  OSCParsersKittyTextSizingHAlign = 1
	OSCParsersKittyTextSizingHAlignCenter OSCParsersKittyTextSizingHAlign = 2
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersKittyTextSizingHAlign) String() string {
	return oscParsersKittyTextSizingHAlignNames[self]
}

var oscParsersKittyTextSizingHAlignNames = [...]string{
	OSCParsersKittyTextSizingHAlignLeft:   "left",
	OSCParsersKittyTextSizingHAlignRight:  "right",
	OSCParsersKittyTextSizingHAlignCenter: "center",
}

// OSCParsersKittyTextSizingOSC is osc.parsers.kitty_text_sizing.OSC.
//
// Zig gives every field but `text` a default value. They are all the Go zero
// value except `scale`, whose default is 1: the implementation must set it
// wherever Zig writes `.{ .text = ... }`.
type OSCParsersKittyTextSizingOSC struct {
	// Zig: u3, default 1. 1 - 7
	scale uint8
	// Zig: u3, default 0. 0 - 7 (0 means default)
	width uint8
	// Zig: u4, default 0.
	numerator uint8
	// Zig: u4, default 0.
	denominator uint8
	// Zig: default .top.
	valign OSCParsersKittyTextSizingVAlign
	// Zig: default .left.
	halign OSCParsersKittyTextSizingHAlign
	text   []uint8
}

// oscParsersKittyTextSizingParseIntU4 is std.fmt.parseInt(u4, buf, 10)
// (Zig 0.15.2, parseIntWithGenericCharacter and parseIntWithSign), with the
// error reduced to ok == false: the only caller turns every error into
// error.InvalidValue.
//
// It accepts what Zig accepts: a leading '+' or '-', '_' between digits
// (not first or last), leading zeros, and "-0" (any number of zero digits
// after '-'). The digits accumulate in a u8 (Zig's Accumulate is at least
// 8 bits) with overflow checks, and the result must fit in a u4.
func oscParsersKittyTextSizingParseIntU4(buf []uint8) (uint8, bool) {
	if len(buf) == 0 {
		return 0, false // error.InvalidCharacter
	}
	neg := false
	if buf[0] == '+' {
		buf = buf[1:]
	} else if buf[0] == '-' {
		buf = buf[1:]
		neg = true
	}

	// parseIntWithSign
	if len(buf) == 0 {
		return 0, false // error.InvalidCharacter
	}

	var accumulate uint8 = 0

	if buf[0] == '_' || buf[len(buf)-1] == '_' {
		return 0, false // error.InvalidCharacter
	}

	for _, c := range buf {
		if c == '_' {
			continue
		}
		// charToDigit(c, 10)
		var digit uint8
		switch {
		case c >= '0' && c <= '9':
			digit = c - '0'
		case c >= 'A' && c <= 'Z':
			digit = c - 'A' + 10
		case c >= 'a' && c <= 'z':
			digit = c - 'a' + 10
		default:
			return 0, false // error.InvalidCharacter
		}
		if digit >= 10 {
			return 0, false // error.InvalidCharacter
		}
		if accumulate != 0 {
			// math.mul(u8, accumulate, 10)
			if uint16(accumulate)*10 > 0xff {
				return 0, false // error.Overflow
			}
			accumulate = accumulate * 10
		} else if neg {
			// The first digit of a negative number.
			// math.cast(u8, -digit): only 0 fits.
			if digit != 0 {
				return 0, false // error.Overflow
			}
			accumulate = 0
			continue
		}
		if neg {
			// math.sub(u8, accumulate, digit)
			if accumulate < digit {
				return 0, false // error.Overflow
			}
			accumulate = accumulate - digit
		} else {
			// math.add(u8, accumulate, digit)
			if uint16(accumulate)+uint16(digit) > 0xff {
				return 0, false // error.Overflow
			}
			accumulate = accumulate + digit
		}
	}

	// math.cast(u4, accumulate)
	if accumulate > 0xf {
		return 0, false // error.Overflow
	}
	return accumulate, true
}

// update is osc.parsers.kitty_text_sizing.OSC.update: apply one key=value
// parameter. Returns ErrUnknownKey or ErrInvalidValue.
func (self *OSCParsersKittyTextSizingOSC) update(key uint8, value []uint8) error {
	// All values are numeric, so we can do a small hack here
	v, ok := oscParsersKittyTextSizingParseIntU4(value)
	if !ok {
		return ErrInvalidValue
	}

	switch key {
	case 's':
		if v == 0 {
			return ErrInvalidValue
		}
		// std.math.cast(u3, v) orelse return error.InvalidValue
		if v > 7 {
			return ErrInvalidValue
		}
		self.scale = v
	case 'w':
		// std.math.cast(u3, v) orelse return error.InvalidValue
		if v > 7 {
			return ErrInvalidValue
		}
		self.width = v
	case 'n':
		self.numerator = v
	case 'd':
		self.denominator = v
	case 'v':
		// std.enums.fromInt(VAlign, v) orelse return error.InvalidValue
		if v > uint8(OSCParsersKittyTextSizingVAlignCenter) {
			return ErrInvalidValue
		}
		self.valign = OSCParsersKittyTextSizingVAlign(v)
	case 'h':
		// std.enums.fromInt(HAlign, v) orelse return error.InvalidValue
		if v > uint8(OSCParsersKittyTextSizingHAlignCenter) {
			return ErrInvalidValue
		}
		self.halign = OSCParsersKittyTextSizingHAlign(v)
	default:
		return ErrUnknownKey
	}
	return nil
}

// OSCParsersKittyTextSizingParse is osc.parsers.kitty_text_sizing.parse:
// parse OSC 66.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersKittyTextSizingParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	assert(parser.state == OSCParserState66)

	writer := parser.writer
	if writer == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}

	// Write a NUL byte to ensure that `text` is NUL-terminated
	if err := writer.writeByte(0); err != nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	data := writer.buffered()

	payload_start := bytes.IndexByte(data, ';')
	if payload_start < 0 {
		warn("kitty_text_sizing", "missing semicolon before payload")
		parser.state = OSCParserStateInvalid
		return nil
	}
	// Zig: data[payload_start + 1 .. data.len - 1 :0]. The NUL written above
	// stays in the slice's capacity, right after its end.
	payload := data[payload_start+1 : len(data)-1]

	// Payload has to be a URL-safe UTF-8 string,
	// and be under the size limit.
	if len(payload) > OSCParsersKittyTextSizingMaxPayloadLength {
		warn("kitty_text_sizing", "payload is too long")
		parser.state = OSCParserStateInvalid
		return nil
	}
	if !OSCEncodingIsSafeUtf8(payload) {
		warn("kitty_text_sizing", "payload is not escape code safe UTF-8")
		parser.state = OSCParserStateInvalid
		return nil
	}

	parser.command = OSCCommand{
		Tag: OSCCommandKeyKittyTextSizing,
		// Zig: .{ .text = payload }; scale defaults to 1.
		KittyTextSizing: OSCParsersKittyTextSizingOSC{scale: 1, text: payload},
	}
	cmd := &parser.command.KittyTextSizing

	// Parse any arguments if given
	if payload_start > 0 {
		// Zig: std.mem.splitScalar(u8, data[0..payload_start], ':').
		// bytes.Split with a one-byte separator yields the same pieces,
		// empty ones included.
		for _, kv := range bytes.Split(data[0:payload_start], []uint8{':'}) {
			// Zig: std.mem.splitScalar(u8, kv, '='); `it.next()` is the
			// next element of it.
			it := bytes.Split(kv, []uint8{'='})
			if len(it) < 1 {
				warn("kitty_text_sizing", "missing key")
				continue
			}
			k := it[0]
			if len(k) != 1 {
				warn("kitty_text_sizing", "key must be a single character")
				continue
			}

			if len(it) < 2 {
				warn("kitty_text_sizing", "missing value")
				continue
			}
			value := it[1]

			if err := cmd.update(k[0], value); err != nil {
				switch err {
				case ErrUnknownKey:
					warn("kitty_text_sizing", "unknown key: '%c'", k[0])
				case ErrInvalidValue:
					warn("kitty_text_sizing", "invalid value for key '%c': error.%v", k[0], err)
				}
				continue
			}
		}
	}

	return &parser.command
}
