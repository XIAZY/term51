// WIREFRAME for src/terminal/osc/parsers/mouse_shape.zig (Ghostty v1.3.1).
// NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: `osc/parsers/mouse_shape.zig` becomes
// `osc_parsers_mouse_shape.go` in package terminal, and its
// declarations take the prefix `OSCParsersMouseShape`. The file has no C ABI
// declarations.

package terminal

// OSCParsersMouseShapeParse is osc.parsers.mouse_shape.parse: parse OSC 22.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersMouseShapeParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}
