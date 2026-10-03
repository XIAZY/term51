// WIREFRAME for src/terminal/osc/encoding.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// Specialized encodings used in some OSC protocols.
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: `osc/encoding.zig` becomes `osc_encoding.go` in package
// terminal, and its declarations take the prefix `OSCEncoding`. The file has
// no C ABI declarations.

package terminal

// OSCEncodingIsSafeUtf8 is osc.encoding.isSafeUtf8.
//
// Kitty defines "Escape code safe UTF-8" as valid UTF-8 with the additional
// requirement of not containing any C0 escape codes (0x00-0x1f), DEL (0x7f)
// and C1 escape codes (0x80-0x9f).
//
// Used by OSC 66 (text sizing) and OSC 99 (Kitty notifications).
//
// See: https://sw.kovidgoyal.net/kitty/desktop-notifications/#safe-utf8
func OSCEncodingIsSafeUtf8(s []uint8) bool { panic(notImplemented) }
