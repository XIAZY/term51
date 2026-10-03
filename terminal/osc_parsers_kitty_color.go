// WIREFRAME for src/terminal/osc/parsers/kitty_color.zig (Ghostty v1.3.1).
// NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: `osc/parsers/kitty_color.zig` becomes
// `osc_parsers_kitty_color.go` in package terminal, and its declarations
// take the prefix `OSCParsersKittyColor`. This is the OSC 21 sub-parser; the
// types it fills (KittyColorOSC, KittyColorKind) are kitty/color.zig's and
// live in kitty_color.go. The file has no C ABI declarations.
//
// `const log = std.log.scoped(.osc_kitty_color)` has no declaration in Go:
// the port uses warn("osc_kitty_color", ...).

package terminal

// OSCParsersKittyColorParse is osc.parsers.kitty_color.parse: parse OSC 21,
// the Kitty Color Protocol.
//
// The result is nil for Zig's null; otherwise it points at the parser's own
// command.
func OSCParsersKittyColorParse(parser *OSCParser, terminator_ch Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}
