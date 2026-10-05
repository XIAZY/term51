// Ported from src/terminal/osc/parsers/change_window_title.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/change_window_title.zig` becomes
// `osc_parsers_change_window_title.go` in package terminal, and its
// declarations take the prefix `OSCParsersChangeWindowTitle`. The file has no C ABI
// declarations.

package terminal

// OSCParsersChangeWindowTitleParse is osc.parsers.change_window_title.parse: parse OSC 0 and OSC 2.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersChangeWindowTitleParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	writer := parser.writer
	if writer == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	if err := writer.writeByte(0); err != nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	data := writer.buffered()
	// Zig: data[0 .. data.len - 1 :0]. The Go slice keeps the NUL behind it
	// in its capacity.
	parser.command = OSCCommand{
		Tag:               OSCCommandKeyChangeWindowTitle,
		ChangeWindowTitle: data[0 : len(data)-1],
	}
	return &parser.command
}
