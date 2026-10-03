// WIREFRAME for src/terminal/osc/parsers/clipboard_operation.zig (Ghostty v1.3.1).
// NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
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
	panic(notImplemented)
}
