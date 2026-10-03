// WIREFRAME for src/terminal/mouse_shape.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
// `getGObjectType` is void in the library build and is not ported.

package terminal

// MouseShape is mouse_shape.MouseShape: the possible cursor shapes. Not all
// app runtimes support these shapes. The shapes are always based on the W3C
// supported cursor styles so we can have a cross platform list.
//
// Zig: enum(c_int).
type MouseShape int32

const (
	MouseShapeDefault MouseShape = iota
	MouseShapeContextMenu
	MouseShapeHelp
	MouseShapePointer
	MouseShapeProgress
	MouseShapeWait
	MouseShapeCell
	MouseShapeCrosshair
	MouseShapeText
	MouseShapeVerticalText
	MouseShapeAlias
	MouseShapeCopy
	MouseShapeMove
	MouseShapeNoDrop
	MouseShapeNotAllowed
	MouseShapeGrab
	MouseShapeGrabbing
	MouseShapeAllScroll
	MouseShapeColResize
	MouseShapeRowResize
	MouseShapeNResize
	MouseShapeEResize
	MouseShapeSResize
	MouseShapeWResize
	MouseShapeNeResize
	MouseShapeNwResize
	MouseShapeSeResize
	MouseShapeSwResize
	MouseShapeEwResize
	MouseShapeNsResize
	MouseShapeNeswResize
	MouseShapeNwseResize
	MouseShapeZoomIn
	MouseShapeZoomOut

	mouseShapeCount = iota
)

// String returns the Zig tag name (@tagName).
func (self MouseShape) String() string { panic(notImplemented) }

// MouseShapeFromString is MouseShape.fromString: build cursor shape from
// string or null if its unknown.
func MouseShapeFromString(v []uint8) Optional[MouseShape] { panic(notImplemented) }

// mouseShapeStringMap is mouse_shape.string_map:
// std.StaticStringMap(MouseShape), the W3C names followed by the xterm/foot
// aliases. Wireframe: left nil; the implementation fills it with upstream's
// entries in upstream's order.
var mouseShapeStringMap map[string]MouseShape
