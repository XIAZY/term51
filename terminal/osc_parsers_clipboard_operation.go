// Ported from src/terminal/osc/parsers/clipboard_operation.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/clipboard_operation.zig` becomes
// `osc_parsers_clipboard_operation.go` in package terminal, and its
// declarations take the prefix `OSCParsersClipboardOperation`. The file has no C ABI
// declarations.

package terminal

// OSCParsersClipboardOperationParse is osc.parsers.clipboard_operation.parse: parse OSC 52.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersClipboardOperationParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	assert(parser.state == OSCParserState52)
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
	if len(data) == 1 {
		parser.state = OSCParserStateInvalid
		return nil
	}
	if data[0] == ';' {
		// Zig: data[1 .. data.len - 1 :0].
		parser.command = OSCCommand{
			Tag: OSCCommandKeyClipboardContents,
			ClipboardContents: OSCCommandClipboardContentsPayload{
				Kind: 'c',
				Data: data[1 : len(data)-1],
			},
		}
	} else {
		if len(data) < 2 {
			parser.state = OSCParserStateInvalid
			return nil
		}
		if data[1] != ';' {
			parser.state = OSCParserStateInvalid
			return nil
		}
		// Zig: data[2 .. data.len - 1 :0].
		parser.command = OSCCommand{
			Tag: OSCCommandKeyClipboardContents,
			ClipboardContents: OSCCommandClipboardContentsPayload{
				Kind: data[0],
				Data: data[2 : len(data)-1],
			},
		}
	}
	return &parser.command
}
