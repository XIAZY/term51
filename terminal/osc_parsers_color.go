// Ported from src/terminal/osc/parsers/color.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/color.zig` becomes `osc_parsers_color.go` in
// package terminal, and its declarations take the prefix `OSCParsersColor`
// (`oscParsersColor` for private ones). The file has no C ABI declarations.
//
// Not ported:
//   - `ParseError` (`Allocator.Error || error{MissingOperation}`): an error
//     set is not a type in Go. Its members are the sentinels ErrOutOfMemory
//     (dcs.go) and ErrMissingOperation (below).
//   - `const log = std.log.scoped(.osc_color)`: the port uses
//     warn("osc_color", ...). The file's only log call is a `log.info`,
//     which the port does not print.
//
// Stand-ins for the standard library, declared next to their first use:
// std.mem.tokenizeScalar and its TokenIterator (after parseColor),
// std.fmt.parseInt(u9, s, 10) (after parseGetSetAnsiColor) and
// std.SegmentedList(Request, 2) (List). std.mem.eql is bytes.Equal.

package terminal

import (
	"bytes"
	"errors"
)

// ErrMissingOperation is Zig's error.MissingOperation, a member of
// osc.parsers.color.ParseError.
var ErrMissingOperation = errors.New("MissingOperation")

// OSCParsersColorOperation is osc.parsers.color.Operation: the possible
// operations we support for colors.
type OSCParsersColorOperation uint8

const (
	OSCParsersColorOperationOsc4 OSCParsersColorOperation = iota
	OSCParsersColorOperationOsc5
	OSCParsersColorOperationOsc10
	OSCParsersColorOperationOsc11
	OSCParsersColorOperationOsc12
	OSCParsersColorOperationOsc13
	OSCParsersColorOperationOsc14
	OSCParsersColorOperationOsc15
	OSCParsersColorOperationOsc16
	OSCParsersColorOperationOsc17
	OSCParsersColorOperationOsc18
	OSCParsersColorOperationOsc19
	OSCParsersColorOperationOsc104
	OSCParsersColorOperationOsc105
	OSCParsersColorOperationOsc110
	OSCParsersColorOperationOsc111
	OSCParsersColorOperationOsc112
	OSCParsersColorOperationOsc113
	OSCParsersColorOperationOsc114
	OSCParsersColorOperationOsc115
	OSCParsersColorOperationOsc116
	OSCParsersColorOperationOsc117
	OSCParsersColorOperationOsc118
	OSCParsersColorOperationOsc119

	oscParsersColorOperationCount = iota
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersColorOperation) String() string {
	return oscParsersColorOperationNames[self]
}

var oscParsersColorOperationNames = [oscParsersColorOperationCount]string{
	OSCParsersColorOperationOsc4:   "osc_4",
	OSCParsersColorOperationOsc5:   "osc_5",
	OSCParsersColorOperationOsc10:  "osc_10",
	OSCParsersColorOperationOsc11:  "osc_11",
	OSCParsersColorOperationOsc12:  "osc_12",
	OSCParsersColorOperationOsc13:  "osc_13",
	OSCParsersColorOperationOsc14:  "osc_14",
	OSCParsersColorOperationOsc15:  "osc_15",
	OSCParsersColorOperationOsc16:  "osc_16",
	OSCParsersColorOperationOsc17:  "osc_17",
	OSCParsersColorOperationOsc18:  "osc_18",
	OSCParsersColorOperationOsc19:  "osc_19",
	OSCParsersColorOperationOsc104: "osc_104",
	OSCParsersColorOperationOsc105: "osc_105",
	OSCParsersColorOperationOsc110: "osc_110",
	OSCParsersColorOperationOsc111: "osc_111",
	OSCParsersColorOperationOsc112: "osc_112",
	OSCParsersColorOperationOsc113: "osc_113",
	OSCParsersColorOperationOsc114: "osc_114",
	OSCParsersColorOperationOsc115: "osc_115",
	OSCParsersColorOperationOsc116: "osc_116",
	OSCParsersColorOperationOsc117: "osc_117",
	OSCParsersColorOperationOsc118: "osc_118",
	OSCParsersColorOperationOsc119: "osc_119",
}

// OSCParsersColorParse is osc.parsers.color.parse: parse OSCs 4, 5, 10-19,
// 104, 110-119.
//
// The result is nil for Zig's null; otherwise it points at the parser's own
// command.
func OSCParsersColorParse(parser *OSCParser, terminator_ch Optional[uint8]) *OSCCommand {
	// Zig: `const alloc = parser.alloc orelse ...`. The allocator itself is
	// not passed on (parseColor drops its allocator parameter).
	if parser.alloc == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	// If we've collected any extra data parse that, otherwise use an empty
	// string.
	var data []uint8
	if writer := parser.writer; writer != nil {
		data = writer.buffered()
	} else {
		data = []uint8("")
	}
	// Check and make sure that we're parsing the correct OSCs
	var op OSCParsersColorOperation
	switch parser.state {
	case OSCParserState4:
		op = OSCParsersColorOperationOsc4
	case OSCParserState5:
		op = OSCParsersColorOperationOsc5
	case OSCParserState10:
		op = OSCParsersColorOperationOsc10
	case OSCParserState11:
		op = OSCParsersColorOperationOsc11
	case OSCParserState12:
		op = OSCParsersColorOperationOsc12
	case OSCParserState13:
		op = OSCParsersColorOperationOsc13
	case OSCParserState14:
		op = OSCParsersColorOperationOsc14
	case OSCParserState15:
		op = OSCParsersColorOperationOsc15
	case OSCParserState16:
		op = OSCParsersColorOperationOsc16
	case OSCParserState17:
		op = OSCParsersColorOperationOsc17
	case OSCParserState18:
		op = OSCParsersColorOperationOsc18
	case OSCParserState19:
		op = OSCParsersColorOperationOsc19
	case OSCParserState104:
		op = OSCParsersColorOperationOsc104
	case OSCParserState110:
		op = OSCParsersColorOperationOsc110
	case OSCParserState111:
		op = OSCParsersColorOperationOsc111
	case OSCParserState112:
		op = OSCParsersColorOperationOsc112
	case OSCParserState113:
		op = OSCParsersColorOperationOsc113
	case OSCParserState114:
		op = OSCParsersColorOperationOsc114
	case OSCParserState115:
		op = OSCParsersColorOperationOsc115
	case OSCParserState116:
		op = OSCParsersColorOperationOsc116
	case OSCParserState117:
		op = OSCParsersColorOperationOsc117
	case OSCParserState118:
		op = OSCParsersColorOperationOsc118
	case OSCParserState119:
		op = OSCParsersColorOperationOsc119
	default:
		parser.state = OSCParserStateInvalid
		return nil
	}
	requests, err := oscParsersColorParseColor(op, data)
	if err != nil {
		// Zig: log.info("failed to parse OSC {t} color request err={}
		// data={s}", .{ parser.state, err, data }); not printed by the
		// port.
		requests = OSCParsersColorList{}
	}
	parser.command = OSCCommand{
		Tag: OSCCommandKeyColorOperation,
		ColorOperation: OSCCommandColorOperationPayload{
			op:         op,
			requests:   requests,
			terminator: OSCTerminatorInit(terminator_ch),
		},
	}
	return &parser.command
}

// oscParsersColorParseColor is osc.parsers.color.parseColor: parse any color
// operation string. This should NOT include the operation itself, but only
// the body of the operation. e.g. for "4;a;b;c" the body should be "a;b;c"
// and the operation should be set accordingly.
//
// Color parsing is fairly complicated so we pull this out to a specialized
// function rather than go through our OSC parsing state machine. This is
// much slower and requires more memory (since we need to buffer the full
// request) but grants us an easier to understand and testable
// implementation.
//
// If color changing ends up being a bottleneck we can optimize this later.
//
// Zig: returns ParseError!List. The allocator parameter is dropped.
func oscParsersColorParseColor(
	op OSCParsersColorOperation,
	buf []uint8,
) (OSCParsersColorList, error) {
	it := oscParsersColorTokenizeScalar(buf, ';')
	switch op {
	case OSCParsersColorOperationOsc4:
		return oscParsersColorParseGetSetAnsiColor(OSCParsersColorOperationOsc4, &it)
	case OSCParsersColorOperationOsc5:
		return oscParsersColorParseGetSetAnsiColor(OSCParsersColorOperationOsc5, &it)
	case OSCParsersColorOperationOsc104:
		return oscParsersColorParseResetAnsiColor(OSCParsersColorOperationOsc104, &it)
	case OSCParsersColorOperationOsc105:
		return oscParsersColorParseResetAnsiColor(OSCParsersColorOperationOsc105, &it)
	case OSCParsersColorOperationOsc10:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicForeground, &it)
	case OSCParsersColorOperationOsc11:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicBackground, &it)
	case OSCParsersColorOperationOsc12:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicCursor, &it)
	case OSCParsersColorOperationOsc13:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicPointerForeground, &it)
	case OSCParsersColorOperationOsc14:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicPointerBackground, &it)
	case OSCParsersColorOperationOsc15:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicTektronixForeground, &it)
	case OSCParsersColorOperationOsc16:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicTektronixBackground, &it)
	case OSCParsersColorOperationOsc17:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicHighlightBackground, &it)
	case OSCParsersColorOperationOsc18:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicTektronixCursor, &it)
	case OSCParsersColorOperationOsc19:
		return oscParsersColorParseGetSetDynamicColor(ColorDynamicHighlightForeground, &it)
	case OSCParsersColorOperationOsc110:
		return oscParsersColorParseResetDynamicColor(ColorDynamicForeground, &it)
	case OSCParsersColorOperationOsc111:
		return oscParsersColorParseResetDynamicColor(ColorDynamicBackground, &it)
	case OSCParsersColorOperationOsc112:
		return oscParsersColorParseResetDynamicColor(ColorDynamicCursor, &it)
	case OSCParsersColorOperationOsc113:
		return oscParsersColorParseResetDynamicColor(ColorDynamicPointerForeground, &it)
	case OSCParsersColorOperationOsc114:
		return oscParsersColorParseResetDynamicColor(ColorDynamicPointerBackground, &it)
	case OSCParsersColorOperationOsc115:
		return oscParsersColorParseResetDynamicColor(ColorDynamicTektronixForeground, &it)
	case OSCParsersColorOperationOsc116:
		return oscParsersColorParseResetDynamicColor(ColorDynamicTektronixBackground, &it)
	case OSCParsersColorOperationOsc117:
		return oscParsersColorParseResetDynamicColor(ColorDynamicHighlightBackground, &it)
	case OSCParsersColorOperationOsc118:
		return oscParsersColorParseResetDynamicColor(ColorDynamicTektronixCursor, &it)
	case OSCParsersColorOperationOsc119:
		return oscParsersColorParseResetDynamicColor(ColorDynamicHighlightForeground, &it)
	}
	// Zig's switch is exhaustive; a value outside the enum traps.
	panic("unreachable")
}

// oscParsersColorTokenIterator stands in for
// std.mem.TokenIterator(u8, .scalar), the iterator returned by
// std.mem.tokenizeScalar(u8, buf, ';') that parseColor hands to its helpers:
// it yields the non-empty runs of buffer between delimiter bytes.
type oscParsersColorTokenIterator struct {
	buffer    []uint8
	delimiter uint8
	index     int
}

// oscParsersColorTokenizeScalar is std.mem.tokenizeScalar(u8, buffer,
// delimiter): an iterator over the non-empty runs of buffer between
// delimiter bytes.
func oscParsersColorTokenizeScalar(buffer []uint8, delimiter uint8) oscParsersColorTokenIterator {
	return oscParsersColorTokenIterator{
		buffer:    buffer,
		delimiter: delimiter,
		index:     0,
	}
}

// next is std.mem.TokenIterator.next: returns a slice of the current token,
// or null if tokenization is complete, and advances to the next token.
//
// Zig's next is `peek` followed by advancing the index past the token; peek
// is written out here (the port has no other use for it).
func (self *oscParsersColorTokenIterator) next() Optional[[]uint8] {
	// Zig: const result = self.peek() orelse return null;

	// move to beginning of token
	for self.index < len(self.buffer) && self.buffer[self.index] == self.delimiter {
		self.index += 1
	}
	start := self.index
	if start == len(self.buffer) {
		return None[[]uint8]()
	}

	// move to end of token
	end := start
	for end < len(self.buffer) && self.buffer[end] != self.delimiter {
		end += 1
	}

	result := self.buffer[start:end]
	self.index += len(result)
	return Some(result)
}

// oscParsersColorParseGetSetAnsiColor is
// osc.parsers.color.parseGetSetAnsiColor: OSC 4/5.
//
// Zig: `op` is a comptime parameter (.osc_4 or .osc_5; anything else does
// not compile) and the result is Allocator.Error!List.
func oscParsersColorParseGetSetAnsiColor(
	op OSCParsersColorOperation,
	it *oscParsersColorTokenIterator,
) (OSCParsersColorList, error) {
	// Note: in ANY error scenario below we return the accumulated results.
	// This matches the xterm behavior (see misc.c ChangeAnsiColorRequest)

	var result OSCParsersColorList
	for {
		// We expect a `c; spec` pair. If either doesn't exist then
		// we return the results up to this point.
		color_str := it.next()
		if !color_str.Valid {
			return result, nil
		}
		spec_str := it.next()
		if !spec_str.Valid {
			return result, nil
		}

		// Color must be numeric. u9 because that'll fit our palette + special
		color_opt := oscParsersColorParseIntU9(color_str.Value)
		if !color_opt.Valid {
			return result, nil
		}
		color := color_opt.Value

		// Parse the color.
		var target OSCParsersColorTarget
		switch op {
		// OSC5 maps directly to the Special enum.
		case OSCParsersColorOperationOsc5:
			// Zig: std.math.cast(u3, color) orelse return result
			if color > 7 {
				return result, nil
			}
			// Zig: std.meta.intToEnum(SpecialColor, ...) catch return
			// result. The values of SpecialColor are 0 through 4.
			if color > uint16(ColorSpecialItalic) {
				return result, nil
			}
			target = OSCParsersColorTarget{Tag: OSCParsersColorTargetSpecial, Special: ColorSpecial(color)}

		// OSC4 maps 0-255 to palette, 256-259 to special offset
		// by the palette count.
		case OSCParsersColorOperationOsc4:
			// Zig: if (std.math.cast(u8, color)) |idx|
			if color <= 255 {
				target = OSCParsersColorTarget{Tag: OSCParsersColorTargetPalette, Palette: uint8(color)}
			} else {
				// Zig: std.math.cast(u3, color - 256) orelse return
				// result. color is at least 256 here.
				if color-256 > 7 {
					return result, nil
				}
				// Zig: std.meta.intToEnum(SpecialColor, ...) catch
				// return result. The values of SpecialColor are 0
				// through 4.
				if color-256 > uint16(ColorSpecialItalic) {
					return result, nil
				}
				target = OSCParsersColorTarget{Tag: OSCParsersColorTargetSpecial, Special: ColorSpecial(color - 256)}
			}

		default:
			// Zig: else => comptime unreachable
			panic("unreachable")
		}

		// "?" always results in a query.
		if bytes.Equal(spec_str.Value, []uint8("?")) {
			req, err := result.addOne()
			if err != nil {
				// Zig: errdefer result.deinit(alloc)
				result.deinit()
				return OSCParsersColorList{}, err
			}
			*req = OSCParsersColorRequest{Tag: OSCParsersColorRequestQuery, Query: target}
			continue
		}

		rgb, err := ColorRGBParse(spec_str.Value)
		if err != nil {
			return result, nil
		}
		req, err := result.addOne()
		if err != nil {
			// Zig: errdefer result.deinit(alloc)
			result.deinit()
			return OSCParsersColorList{}, err
		}
		*req = OSCParsersColorRequest{Tag: OSCParsersColorRequestSet, Set: OSCParsersColorColoredTarget{
			target: target,
			color:  rgb,
		}}
	}
}

// oscParsersColorParseIntU9 stands in for std.fmt.parseInt(u9, buf, 10)
// (parseIntWithGenericCharacter): an optional leading '+' or '-', then
// parseIntWithSign. The callers discard the error (`catch`), so the
// result is null for both of Zig's errors, InvalidCharacter and Overflow.
func oscParsersColorParseIntU9(buf []uint8) Optional[uint16] {
	if len(buf) == 0 {
		return None[uint16]()
	}
	if buf[0] == '+' {
		return oscParsersColorParseIntWithSignU9(buf[1:], false)
	}
	if buf[0] == '-' {
		return oscParsersColorParseIntWithSignU9(buf[1:], true)
	}
	return oscParsersColorParseIntWithSignU9(buf, false)
}

// oscParsersColorParseIntWithSignU9 stands in for std.fmt.parseIntWithSign
// with Result = Accumulate = u9 and base 10 ('_' between digits is
// ignored). neg is Zig's `sign == .neg`: digits are subtracted, which for
// an unsigned type only lets zeros through ("-0" is 0, "-1" overflows).
func oscParsersColorParseIntWithSignU9(buf []uint8, neg bool) Optional[uint16] {
	// maxInt(u9)
	const max = 511

	if len(buf) == 0 {
		return None[uint16]()
	}

	var accumulate uint16 = 0

	if buf[0] == '_' || buf[len(buf)-1] == '_' {
		return None[uint16]()
	}

	for _, c := range buf {
		if c == '_' {
			continue
		}
		// Zig: charToDigit(c, 10)
		var digit uint16
		switch {
		case c >= '0' && c <= '9':
			digit = uint16(c - '0')
		case c >= 'A' && c <= 'Z':
			digit = uint16(c-'A') + 10
		case c >= 'a' && c <= 'z':
			digit = uint16(c-'a') + 10
		default:
			return None[uint16]()
		}
		if digit >= 10 {
			return None[uint16]()
		}
		if accumulate != 0 {
			// Zig: math.mul(Accumulate, accumulate, 10)
			accumulate *= 10
			if accumulate > max {
				return None[uint16]()
			}
		} else if neg {
			// The first digit of a negative number.
			// Zig: math.cast(Accumulate, -@as(i8, digit)) orelse
			// return error.Overflow; only zero fits a u9.
			if digit != 0 {
				return None[uint16]()
			}
			accumulate = 0
			continue
		}
		if neg {
			// Zig: math.sub(Accumulate, accumulate, digit)
			if digit > accumulate {
				return None[uint16]()
			}
			accumulate -= digit
		} else {
			// Zig: math.add(Accumulate, accumulate, digit)
			accumulate += digit
			if accumulate > max {
				return None[uint16]()
			}
		}
	}

	return Some(accumulate)
}

// oscParsersColorParseResetAnsiColor is
// osc.parsers.color.parseResetAnsiColor: OSC 104/105: Reset ANSI Colors.
//
// Zig: `op` is a comptime parameter (.osc_104 or .osc_105; anything else
// does not compile) and the result is Allocator.Error!List.
func oscParsersColorParseResetAnsiColor(
	op OSCParsersColorOperation,
	it *oscParsersColorTokenIterator,
) (OSCParsersColorList, error) {
	// Note: xterm stops parsing the reset list on any error, but we're
	// more flexible and try the next value. This matches the behavior of
	// Kitty and I don't see a downside to being more flexible here. Hopefully
	// no one depends on the exact behavior of xterm.

	var result OSCParsersColorList
	for {
		color_str_opt := it.next()
		if !color_str_opt.Valid {
			// If no parameters are given, we reset the full table.
			if result.count() == 0 {
				req, err := result.addOne()
				if err != nil {
					// Zig: errdefer result.deinit(alloc)
					result.deinit()
					return OSCParsersColorList{}, err
				}
				switch op {
				case OSCParsersColorOperationOsc104:
					*req = OSCParsersColorRequest{Tag: OSCParsersColorRequestResetPalette}
				case OSCParsersColorOperationOsc105:
					*req = OSCParsersColorRequest{Tag: OSCParsersColorRequestResetSpecial}
				default:
					// Zig: else => comptime unreachable
					panic("unreachable")
				}
			}
			return result, nil
		}
		color_str := color_str_opt.Value

		// Empty color strings are ignored, not treated as an error.
		if len(color_str) == 0 {
			continue
		}

		// Color must be numeric. u9 because that'll fit our palette + special
		color_opt := oscParsersColorParseIntU9(color_str)
		if !color_opt.Valid {
			continue
		}
		color := color_opt.Value

		// Parse the color.
		var target OSCParsersColorTarget
		switch op {
		// OSC105 maps directly to the Special enum.
		case OSCParsersColorOperationOsc105:
			// Zig: std.math.cast(u3, color) orelse continue
			if color > 7 {
				continue
			}
			// Zig: std.meta.intToEnum(SpecialColor, ...) catch
			// continue. The values of SpecialColor are 0 through 4.
			if color > uint16(ColorSpecialItalic) {
				continue
			}
			target = OSCParsersColorTarget{Tag: OSCParsersColorTargetSpecial, Special: ColorSpecial(color)}

		// OSC104 maps 0-255 to palette, 256-259 to special offset
		// by the palette count.
		case OSCParsersColorOperationOsc104:
			// Zig: if (std.math.cast(u8, color)) |idx|
			if color <= 255 {
				target = OSCParsersColorTarget{Tag: OSCParsersColorTargetPalette, Palette: uint8(color)}
			} else {
				// Zig: std.math.cast(u3, color - 256) orelse continue.
				// color is at least 256 here.
				if color-256 > 7 {
					continue
				}
				// Zig: std.meta.intToEnum(SpecialColor, ...) catch
				// continue. The values of SpecialColor are 0 through 4.
				if color-256 > uint16(ColorSpecialItalic) {
					continue
				}
				target = OSCParsersColorTarget{Tag: OSCParsersColorTargetSpecial, Special: ColorSpecial(color - 256)}
			}

		default:
			// Zig: else => comptime unreachable
			panic("unreachable")
		}

		req, err := result.addOne()
		if err != nil {
			// Zig: errdefer result.deinit(alloc)
			result.deinit()
			return OSCParsersColorList{}, err
		}
		*req = OSCParsersColorRequest{Tag: OSCParsersColorRequestReset, Reset: target}
	}
}

// oscParsersColorParseGetSetDynamicColor is
// osc.parsers.color.parseGetSetDynamicColor: OSC 10-19: Get/Set Dynamic
// Colors.
//
// Zig: the result is Allocator.Error!List.
func oscParsersColorParseGetSetDynamicColor(
	start ColorDynamic,
	it *oscParsersColorTokenIterator,
) (OSCParsersColorList, error) {
	// Note: in ANY error scenario below we return the accumulated results.
	// This matches the xterm behavior (see misc.c ChangeColorsRequest)

	// Zig has no errdefer here: on an allocation failure the partial list
	// is not freed.
	var result OSCParsersColorList
	color := start
	for {
		spec_str := it.next()
		if !spec_str.Valid {
			return result, nil
		}

		if bytes.Equal(spec_str.Value, []uint8("?")) {
			req, err := result.addOne()
			if err != nil {
				return OSCParsersColorList{}, err
			}
			*req = OSCParsersColorRequest{Tag: OSCParsersColorRequestQuery, Query: OSCParsersColorTarget{
				Tag: OSCParsersColorTargetDynamic, Dynamic: color,
			}}
		} else {
			rgb, err := ColorRGBParse(spec_str.Value)
			if err != nil {
				return result, nil
			}
			req, err := result.addOne()
			if err != nil {
				return OSCParsersColorList{}, err
			}
			*req = OSCParsersColorRequest{Tag: OSCParsersColorRequestSet, Set: OSCParsersColorColoredTarget{
				target: OSCParsersColorTarget{Tag: OSCParsersColorTargetDynamic, Dynamic: color},
				color:  rgb,
			}}
		}

		// Each successive value uses the next color so long as it exists.
		next := color.Next()
		if !next.Valid {
			return result, nil
		}
		color = next.Value
	}
}

// oscParsersColorParseResetDynamicColor is
// osc.parsers.color.parseResetDynamicColor: OSC 110-119: Reset Dynamic
// Colors.
//
// Zig: the result is Allocator.Error!List.
func oscParsersColorParseResetDynamicColor(
	color ColorDynamic,
	it *oscParsersColorTokenIterator,
) (OSCParsersColorList, error) {
	var result OSCParsersColorList
	if it.next().Valid {
		return result, nil
	}
	req, err := result.addOne()
	if err != nil {
		// Zig: errdefer result.deinit(alloc)
		result.deinit()
		return OSCParsersColorList{}, err
	}
	*req = OSCParsersColorRequest{Tag: OSCParsersColorRequestReset, Reset: OSCParsersColorTarget{
		Tag: OSCParsersColorTargetDynamic, Dynamic: color,
	}}
	return result, nil
}

// OSCParsersColorList is osc.parsers.color.List:
// std.SegmentedList(Request, 2).
//
// A segmented list is used to avoid copying when many operations
// are given in a single OSC. In most cases, OSC 4/104/etc. send
// very few so the prealloc is optimized for that.
//
// The exact prealloc value is chosen arbitrarily assuming most
// color ops have very few. If we can get empirical data on more
// typical values we can switch to that.
//
// In Go the segments are not observable, so the list is one slice. The
// methods keep Zig's names (the way StaticBitSet256 does in color.go); the
// zero value is Zig's `.{}`, the empty list.
type OSCParsersColorList struct {
	items []OSCParsersColorRequest
}

// append is std.SegmentedList.append: add one request at the end. Zig
// returns Allocator.Error!void; Go's append does not fail, so the error is
// always nil.
func (self *OSCParsersColorList) append(item OSCParsersColorRequest) error {
	new_item_ptr, err := self.addOne()
	if err != nil {
		return err
	}
	*new_item_ptr = item
	return nil
}

// addOne is std.SegmentedList.addOne: add one element at the end and
// return a pointer to it. Zig's new element is undefined; Go's is the zero
// value. Zig returns Allocator.Error!*T; Go's append does not fail, so the
// error is always nil.
func (self *OSCParsersColorList) addOne() (*OSCParsersColorRequest, error) {
	self.items = append(self.items, OSCParsersColorRequest{})
	return &self.items[len(self.items)-1], nil
}

// count is std.SegmentedList.count: the number of requests.
func (self OSCParsersColorList) count() int {
	return len(self.items)
}

// at is std.SegmentedList.at: a pointer to the request at index i. Zig
// asserts i < count().
func (self *OSCParsersColorList) at(i int) *OSCParsersColorRequest {
	assert(i < len(self.items))
	return &self.items[i]
}

// constIterator is std.SegmentedList.constIterator: an iterator positioned
// at start_index.
func (self *OSCParsersColorList) constIterator(start_index int) OSCParsersColorListConstIterator {
	return OSCParsersColorListConstIterator{
		list:  self,
		index: start_index,
	}
}

// deinit is std.SegmentedList.deinit. Zig frees the list and leaves it
// undefined; Go drops the items. The allocator parameter is dropped.
func (self *OSCParsersColorList) deinit() {
	*self = OSCParsersColorList{}
}

// OSCParsersColorListConstIterator is the iterator returned by
// OSCParsersColorList.constIterator (std.SegmentedList.ConstIterator).
type OSCParsersColorListConstIterator struct {
	list  *OSCParsersColorList
	index int
}

// next returns the next request, or nil at the end. Zig: next() ?*const T.
func (self *OSCParsersColorListConstIterator) next() *OSCParsersColorRequest {
	if self.index >= len(self.list.items) {
		return nil
	}
	ptr := &self.list.items[self.index]
	self.index += 1
	return ptr
}

// OSCParsersColorRequestTag is the tag of osc.parsers.color.Request.
type OSCParsersColorRequestTag uint8

const (
	OSCParsersColorRequestSet OSCParsersColorRequestTag = iota
	OSCParsersColorRequestQuery
	OSCParsersColorRequestReset
	OSCParsersColorRequestResetPalette
	OSCParsersColorRequestResetSpecial
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersColorRequestTag) String() string {
	return oscParsersColorRequestTagNames[self]
}

var oscParsersColorRequestTagNames = [...]string{
	OSCParsersColorRequestSet:          "set",
	OSCParsersColorRequestQuery:        "query",
	OSCParsersColorRequestReset:        "reset",
	OSCParsersColorRequestResetPalette: "reset_palette",
	OSCParsersColorRequestResetSpecial: "reset_special",
}

// OSCParsersColorRequest is osc.parsers.color.Request: a single operation
// related to the terminal color palette.
//
// Zig: a tagged union. Only the field named by Tag is meaningful;
// reset_palette and reset_special carry no payload.
type OSCParsersColorRequest struct {
	Tag OSCParsersColorRequestTag

	Set   OSCParsersColorColoredTarget
	Query OSCParsersColorTarget
	Reset OSCParsersColorTarget
}

// OSCParsersColorTargetTag is the tag of osc.parsers.color.Target.
type OSCParsersColorTargetTag uint8

const (
	OSCParsersColorTargetPalette OSCParsersColorTargetTag = iota
	OSCParsersColorTargetSpecial
	OSCParsersColorTargetDynamic
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersColorTargetTag) String() string {
	return oscParsersColorTargetTagNames[self]
}

var oscParsersColorTargetTagNames = [...]string{
	OSCParsersColorTargetPalette: "palette",
	OSCParsersColorTargetSpecial: "special",
	OSCParsersColorTargetDynamic: "dynamic",
}

// OSCParsersColorTarget is osc.parsers.color.Target.
//
// Zig: a tagged union. Only the field named by Tag is meaningful.
type OSCParsersColorTarget struct {
	Tag OSCParsersColorTargetTag

	Palette uint8
	Special ColorSpecial
	Dynamic ColorDynamic
}

// OSCParsersColorColoredTarget is osc.parsers.color.ColoredTarget.
type OSCParsersColorColoredTarget struct {
	target OSCParsersColorTarget
	color  ColorRGB
}
