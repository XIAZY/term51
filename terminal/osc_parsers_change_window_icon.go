// WIREFRAME for src/terminal/osc/parsers/change_window_icon.zig (Ghostty v1.3.1).
// NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
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
	panic(notImplemented)
}
