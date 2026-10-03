// WIREFRAME for src/terminal/kitty/color.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: a Zig file in a subdirectory, `kitty/color.zig`, becomes
// `kitty_color.go` in package terminal, and its declarations take the
// prefix `KittyColor`. The C ABI declarations (`C`, `cval`) are not ported.

package terminal

// KittyColorOSC is kitty.color.OSC.
type KittyColorOSC struct {
	// list of requests
	//
	// Zig: std.ArrayList(Request).
	list []KittyColorOSCRequest

	// We must reply with the same string terminator (ST) as used in the
	// request.
	//
	// Zig: osc.Terminator, default .st (the zero value).
	terminator OSCTerminator
}

// KittyColorOSCRequestTag is the tag of kitty.color.OSC.Request.
type KittyColorOSCRequestTag uint8

const (
	KittyColorOSCRequestQuery KittyColorOSCRequestTag = iota
	KittyColorOSCRequestSet
	KittyColorOSCRequestReset
)

// KittyColorOSCRequest is kitty.color.OSC.Request.
//
// Zig: a tagged union. Only the field named by Tag is meaningful.
type KittyColorOSCRequest struct {
	Tag KittyColorOSCRequestTag

	Query KittyColorKind
	Set   KittyColorOSCRequestSetPayload
	Reset KittyColorKind
}

// KittyColorOSCRequestSetPayload is the payload of Request.set:
// struct { key: Kind, color: RGB }.
type KittyColorOSCRequestSetPayload struct {
	key   KittyColorKind
	color ColorRGB
}

// Deinit is kitty.color.OSC.deinit. Zig frees the list; the allocator
// parameter is dropped.
func (self *KittyColorOSC) Deinit() { panic(notImplemented) }

// KittyColorSpecial is kitty.color.Special.
type KittyColorSpecial uint8

const (
	KittyColorSpecialForeground KittyColorSpecial = iota
	KittyColorSpecialBackground
	KittyColorSpecialSelectionForeground
	KittyColorSpecialSelectionBackground
	KittyColorSpecialCursor
	KittyColorSpecialCursorText
	KittyColorSpecialVisualBell
	KittyColorSpecialSecondTransparentBackground

	kittyColorSpecialCount = iota
)

// String returns the Zig tag name (@tagName).
func (self KittyColorSpecial) String() string { panic(notImplemented) }

// KittyColorKindTag is the tag of kitty.color.Kind.
type KittyColorKindTag uint8

const (
	KittyColorKindPalette KittyColorKindTag = iota
	KittyColorKindSpecial
)

// KittyColorKind is kitty.color.Kind.
//
// Zig: a tagged union. Only the field named by Tag is meaningful.
type KittyColorKind struct {
	Tag KittyColorKindTag

	Palette uint8
	Special KittyColorSpecial
}

// KittyColorKindMax is kitty.color.Kind.max:
// maxInt(u8) + the number of Special values.
const KittyColorKindMax = 255 + kittyColorSpecialCount

// KittyColorKindParse is kitty.color.Kind.parse.
func KittyColorKindParse(key []uint8) Optional[KittyColorKind] { panic(notImplemented) }

// Format is kitty.color.Kind.format. Per docs/mapping.md a Zig `format`
// method becomes `Format() string` with output identical to Zig's.
func (self KittyColorKind) Format() string { panic(notImplemented) }
