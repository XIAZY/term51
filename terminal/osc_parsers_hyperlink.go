// WIREFRAME for src/terminal/osc/parsers/hyperlink.zig (Ghostty v1.3.1).
// NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: `osc/parsers/hyperlink.zig` becomes
// `osc_parsers_hyperlink.go` in package terminal, and its
// declarations take the prefix `OSCParsersHyperlink`. The file has no C ABI
// declarations.
//
// `const log = std.log.scoped(.osc_hyperlink)` has no declaration in Go: the
// port uses warn("osc_hyperlink", ...).

package terminal

// OSCParsersHyperlinkParse is osc.parsers.hyperlink.parse: parse OSC 8 hyperlinks.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersHyperlinkParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}
