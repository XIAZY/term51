// WIREFRAME for src/terminal/osc/parsers/kitty_text_sizing.zig (Ghostty
// v1.3.1). NOT YET PORTED.
//
// Kitty's text sizing protocol (OSC 66)
// Specification: https://sw.kovidgoyal.net/kitty/text-sizing-protocol/
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: `osc/parsers/kitty_text_sizing.zig` becomes
// `osc_parsers_kitty_text_sizing.go` in package terminal, and its
// declarations take the prefix `OSCParsersKittyTextSizing`.
//
// Not ported, see docs/mapping.md:
//   - the C ABI declarations `OSC.C` (void) and `OSC.cval`, the C variants
//     of `VAlign` and `HAlign` (lib.Enum with the .c target), and
//     `lib_target`, which only selects between the two;
//   - `const log = std.log.scoped(.kitty_text_sizing)`: the port uses
//     warn("kitty_text_sizing", ...).

package terminal

import "errors"

// ErrUnknownKey is Zig's error.UnknownKey (kitty_text_sizing.OSC.update).
var ErrUnknownKey = errors.New("UnknownKey")

// ErrInvalidValue is Zig's error.InvalidValue (kitty_text_sizing.OSC.update).
var ErrInvalidValue = errors.New("InvalidValue")

// OSCParsersKittyTextSizingMaxPayloadLength is
// osc.parsers.kitty_text_sizing.max_payload_length.
const OSCParsersKittyTextSizingMaxPayloadLength = 4096

// OSCParsersKittyTextSizingVAlign is osc.parsers.kitty_text_sizing.VAlign.
//
// Zig: lib.Enum with the .zig target in the library build, i.e. a plain enum
// with these values in this order.
type OSCParsersKittyTextSizingVAlign uint8

const (
	OSCParsersKittyTextSizingVAlignTop    OSCParsersKittyTextSizingVAlign = 0
	OSCParsersKittyTextSizingVAlignBottom OSCParsersKittyTextSizingVAlign = 1
	OSCParsersKittyTextSizingVAlignCenter OSCParsersKittyTextSizingVAlign = 2
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersKittyTextSizingVAlign) String() string { panic(notImplemented) }

// OSCParsersKittyTextSizingHAlign is osc.parsers.kitty_text_sizing.HAlign.
//
// Zig: lib.Enum with the .zig target in the library build, i.e. a plain enum
// with these values in this order.
type OSCParsersKittyTextSizingHAlign uint8

const (
	OSCParsersKittyTextSizingHAlignLeft   OSCParsersKittyTextSizingHAlign = 0
	OSCParsersKittyTextSizingHAlignRight  OSCParsersKittyTextSizingHAlign = 1
	OSCParsersKittyTextSizingHAlignCenter OSCParsersKittyTextSizingHAlign = 2
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersKittyTextSizingHAlign) String() string { panic(notImplemented) }

// OSCParsersKittyTextSizingOSC is osc.parsers.kitty_text_sizing.OSC.
//
// Zig gives every field but `text` a default value. They are all the Go zero
// value except `scale`, whose default is 1: the implementation must set it
// wherever Zig writes `.{ .text = ... }`.
type OSCParsersKittyTextSizingOSC struct {
	// Zig: u3, default 1. 1 - 7
	scale uint8
	// Zig: u3, default 0. 0 - 7 (0 means default)
	width uint8
	// Zig: u4, default 0.
	numerator uint8
	// Zig: u4, default 0.
	denominator uint8
	// Zig: default .top.
	valign OSCParsersKittyTextSizingVAlign
	// Zig: default .left.
	halign OSCParsersKittyTextSizingHAlign
	text   []uint8
}

// update is osc.parsers.kitty_text_sizing.OSC.update: apply one key=value
// parameter. Returns ErrUnknownKey or ErrInvalidValue.
func (self *OSCParsersKittyTextSizingOSC) update(key uint8, value []uint8) error {
	panic(notImplemented)
}

// OSCParsersKittyTextSizingParse is osc.parsers.kitty_text_sizing.parse:
// parse OSC 66.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersKittyTextSizingParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}
