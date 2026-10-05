// Ported from src/terminal/osc/parsers/rxvt_extension.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/rxvt_extension.zig` becomes
// `osc_parsers_rxvt_extension.go` in package terminal, and its
// declarations take the prefix `OSCParsersRxvtExtension`. The file has no C ABI
// declarations.
//
// `const log = std.log.scoped(.osc_rxvt_extension)` has no declaration in Go:
// the port uses warn("osc_rxvt_extension", ...).
//
// std.mem.indexOfScalar is bytes.IndexByte (-1 is null), std.mem.eql is
// bytes.Equal.

package terminal

import "bytes"

// OSCParsersRxvtExtensionParse is osc.parsers.rxvt_extension.parse: parse OSC 777.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersRxvtExtensionParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	writer := parser.writer
	if writer == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	// ensure that we are sentinel terminated
	if err := writer.writeByte(0); err != nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	data := writer.buffered()
	k := bytes.IndexByte(data, ';')
	if k < 0 {
		parser.state = OSCParserStateInvalid
		return nil
	}
	ext := data[0:k]
	if !bytes.Equal(ext, []uint8("notify")) {
		warn("osc_rxvt_extension", "unknown rxvt extension: %s", ext)
		parser.state = OSCParserStateInvalid
		return nil
	}
	// Zig: std.mem.indexOfScalarPos(u8, data, k + 1, ';'). k + 1 is at most
	// data.len - 1 because data ends with the NUL written above.
	t := bytes.IndexByte(data[k+1:], ';')
	if t < 0 {
		warn("osc_rxvt_extension", "rxvt notify extension is missing the title")
		parser.state = OSCParserStateInvalid
		return nil
	}
	t += k + 1
	data[t] = 0
	// Zig: data[k + 1 .. t :0] and data[t + 1 .. data.len - 1 :0]. The Go
	// slices keep the NUL behind them in their capacity.
	title := data[k+1 : t]
	body := data[t+1 : len(data)-1]
	parser.command = OSCCommand{
		Tag: OSCCommandKeyShowDesktopNotification,
		ShowDesktopNotification: OSCCommandShowDesktopNotificationPayload{
			title: title,
			body:  body,
		},
	}
	return &parser.command
}
