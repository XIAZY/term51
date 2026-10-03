// WIREFRAME for src/terminal/osc/parsers/osc9.zig (Ghostty v1.3.1).
// NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: `osc/parsers/osc9.zig` becomes
// `osc_parsers_osc9.go` in package terminal, and its
// declarations take the prefix `OSCParsersOsc9`. The file has no C ABI
// declarations.

package terminal

// OSCParsersOsc9Parse is osc.parsers.osc9.parse: parse OSC 9, which
// could be an iTerm2 notification or a ConEmu extension.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersOsc9Parse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}
