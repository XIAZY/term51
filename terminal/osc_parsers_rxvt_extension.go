// WIREFRAME for src/terminal/osc/parsers/rxvt_extension.zig (Ghostty v1.3.1).
// NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: `osc/parsers/rxvt_extension.zig` becomes
// `osc_parsers_rxvt_extension.go` in package terminal, and its
// declarations take the prefix `OSCParsersRxvtExtension`. The file has no C ABI
// declarations.
//
// `const log = std.log.scoped(.osc_rxvt_extension)` has no declaration in Go:
// the port uses warn("osc_rxvt_extension", ...).

package terminal

// OSCParsersRxvtExtensionParse is osc.parsers.rxvt_extension.parse: parse OSC 777.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersRxvtExtensionParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}
