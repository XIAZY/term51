// Ported from src/terminal/osc/parsers/kitty_color.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/kitty_color.zig` becomes
// `osc_parsers_kitty_color.go` in package terminal, and its declarations
// take the prefix `OSCParsersKittyColor`. This is the OSC 21 sub-parser; the
// types it fills (KittyColorOSC, KittyColorKind) are kitty/color.zig's and
// live in kitty_color.go. The file has no C ABI declarations.
//
// `const log = std.log.scoped(.osc_kitty_color)` has no declaration in Go:
// the port uses warn("osc_kitty_color", ...).
//
// std.mem.splitScalar is oscParsersKittyColorSplitScalar (declared after
// parse, with Zig's next and rest), std.mem.eql is bytes.Equal and
// std.mem.trim(u8, s, " ") is bytes.Trim(s, " ").

package terminal

import "bytes"

// OSCParsersKittyColorParse is osc.parsers.kitty_color.parse: parse OSC 21,
// the Kitty Color Protocol.
//
// The result is nil for Zig's null; otherwise it points at the parser's own
// command.
func OSCParsersKittyColorParse(parser *OSCParser, terminator_ch Optional[uint8]) *OSCCommand {
	assert(parser.state == OSCParserState21)

	// Zig: `const alloc = parser.alloc orelse ...`. The allocator itself is
	// not used in Go (list.append does not take one).
	if parser.alloc == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	writer := parser.writer
	if writer == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	parser.command = OSCCommand{
		Tag: OSCCommandKeyKittyColorProtocol,
		KittyColorProtocol: KittyColorOSC{
			// Zig: .empty.
			list:       nil,
			terminator: OSCTerminatorInit(terminator_ch),
		},
	}
	list := &parser.command.KittyColorProtocol.list
	data := writer.buffered()
	kv_it := oscParsersKittyColorSplitScalar(data, ';')
	for {
		kv_opt := kv_it.next()
		if !kv_opt.Valid {
			break
		}
		kv := kv_opt.Value
		if len(*list) >= int(KittyColorKindMax)*2 {
			warn("osc_kitty_color", "exceeded limit for number of keys in kitty color protocol, ignoring")
			parser.state = OSCParserStateInvalid
			return nil
		}
		it := oscParsersKittyColorSplitScalar(kv, '=')
		k_opt := it.next()
		if !k_opt.Valid {
			continue
		}
		k := k_opt.Value
		if len(k) == 0 {
			warn("osc_kitty_color", "zero length key in kitty color protocol")
			continue
		}
		key_opt := KittyColorKindParse(k)
		if !key_opt.Valid {
			warn("osc_kitty_color", "unknown key in kitty color protocol: %s", k)
			continue
		}
		key := key_opt.Value
		value := bytes.Trim(it.rest(), " ")
		// Zig: each `list.append(alloc, ...)` is followed by
		// `catch |err| { log.warn("unable to append kitty color protocol
		// option: {}", .{err}); continue; }`. Go's append does not fail, so
		// that branch has no counterpart.
		if len(value) == 0 {
			*list = append(*list, KittyColorOSCRequest{Tag: KittyColorOSCRequestReset, Reset: key})
		} else if bytes.Equal([]uint8("?"), value) {
			*list = append(*list, KittyColorOSCRequest{Tag: KittyColorOSCRequestQuery, Query: key})
		} else {
			color, err := ColorRGBParse(value)
			if err != nil {
				switch err {
				case ErrInvalidFormat:
					warn("osc_kitty_color", "invalid color format in kitty color protocol: %s", value)
					continue
				default:
					// Zig's error switch is exhaustive over RGB.parse's
					// error set, which is error{InvalidFormat}.
					panic("unreachable")
				}
			}
			*list = append(*list, KittyColorOSCRequest{
				Tag: KittyColorOSCRequestSet,
				Set: KittyColorOSCRequestSetPayload{
					key:   key,
					color: color,
				},
			})
		}
	}
	return &parser.command
}

// oscParsersKittyColorSplitIterator stands in for
// std.mem.SplitIterator(u8, .scalar), the iterator returned by
// std.mem.splitScalar: it yields the runs of buffer between delimiter
// bytes, empty runs included. Only next and rest, which parse uses, are
// provided, with Zig's names.
type oscParsersKittyColorSplitIterator struct {
	buffer []uint8
	// Zig: index: ?usize; null once the last run has been returned.
	index     Optional[int]
	delimiter uint8
}

// oscParsersKittyColorSplitScalar is std.mem.splitScalar(u8, buffer,
// delimiter).
func oscParsersKittyColorSplitScalar(buffer []uint8, delimiter uint8) oscParsersKittyColorSplitIterator {
	return oscParsersKittyColorSplitIterator{
		buffer:    buffer,
		index:     Some(0),
		delimiter: delimiter,
	}
}

// next is std.mem.SplitIterator.next: returns a slice of the next field, or
// null if splitting is complete.
func (self *oscParsersKittyColorSplitIterator) next() Optional[[]uint8] {
	if !self.index.Valid {
		return None[[]uint8]()
	}
	start := self.index.Value
	// Zig: indexOfScalarPos(T, self.buffer, start, self.delimiter).
	var end int
	if delim_start := bytes.IndexByte(self.buffer[start:], self.delimiter); delim_start >= 0 {
		self.index = Some(start + delim_start + 1)
		end = start + delim_start
	} else {
		self.index = None[int]()
		end = len(self.buffer)
	}
	return Some(self.buffer[start:end])
}

// rest is std.mem.SplitIterator.rest: returns a slice of the remaining
// bytes. Does not affect iterator state.
func (self *oscParsersKittyColorSplitIterator) rest() []uint8 {
	end := len(self.buffer)
	start := end
	if self.index.Valid {
		start = self.index.Value
	}
	return self.buffer[start:end]
}
