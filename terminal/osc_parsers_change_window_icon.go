// Ported from src/terminal/osc/parsers/change_window_icon.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/change_window_icon.zig` becomes
// `osc_parsers_change_window_icon.go` in package terminal, and its
// declarations take the prefix `OSCParsersChangeWindowIcon`. The file has no C ABI
// declarations.

package terminal

// OSCParsersChangeWindowIconParse is osc.parsers.change_window_icon.parse: parse OSC 1.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersChangeWindowIconParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
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
		Tag:              OSCCommandKeyChangeWindowIcon,
		ChangeWindowIcon: data[0 : len(data)-1],
	}
	return &parser.command
}
