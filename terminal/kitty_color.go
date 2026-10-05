// Port of src/terminal/kitty/color.zig (Ghostty v1.3.1).
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

// kittyColorOSCRequestTagNames are the tag names of KittyColorOSCRequestTag,
// in order.
var kittyColorOSCRequestTagNames = [...]string{
	"query",
	"set",
	"reset",
}

// String returns the Zig tag name (@tagName).
func (self KittyColorOSCRequestTag) String() string { return kittyColorOSCRequestTagNames[self] }

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
//
// Zig's ArrayList.deinit leaves the list undefined; Go sets it to its zero
// value (docs/mapping.md: `undefined` memory is the Go zero value).
func (self *KittyColorOSC) Deinit() {
	self.list = nil
}

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

// kittyColorSpecialNames are the tag names of KittyColorSpecial, in order.
var kittyColorSpecialNames = [kittyColorSpecialCount]string{
	"foreground",
	"background",
	"selection_foreground",
	"selection_background",
	"cursor",
	"cursor_text",
	"visual_bell",
	"second_transparent_background",
}

// String returns the Zig tag name (@tagName).
func (self KittyColorSpecial) String() string { return kittyColorSpecialNames[self] }

// KittyColorKindTag is the tag of kitty.color.Kind.
type KittyColorKindTag uint8

const (
	KittyColorKindPalette KittyColorKindTag = iota
	KittyColorKindSpecial
)

// kittyColorKindTagNames are the tag names of KittyColorKindTag, in order.
var kittyColorKindTagNames = [...]string{
	"palette",
	"special",
}

// String returns the Zig tag name (@tagName).
func (self KittyColorKindTag) String() string { return kittyColorKindTagNames[self] }

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
func KittyColorKindParse(key []uint8) Optional[KittyColorKind] {
	// Zig: std.meta.stringToEnum(Special, key), an exact match of a tag
	// name.
	for i := KittyColorSpecial(0); i < kittyColorSpecialCount; i++ {
		if string(key) == kittyColorSpecialNames[i] {
			return Some(KittyColorKind{Tag: KittyColorKindSpecial, Special: i})
		}
	}
	p, ok := colorStdFmtParseUnsigned(key, 10, 8)
	if !ok {
		return None[KittyColorKind]()
	}
	return Some(KittyColorKind{Tag: KittyColorKindPalette, Palette: uint8(p)})
}

// Format is kitty.color.Kind.format. Per docs/mapping.md a Zig `format`
// method becomes `Format() string` with output identical to Zig's.
func (self KittyColorKind) Format() string {
	switch self.Tag {
	case KittyColorKindPalette:
		return itoa(uint64(self.Palette))
	case KittyColorKindSpecial:
		return self.Special.String()
	}
	panic("unreachable")
}
