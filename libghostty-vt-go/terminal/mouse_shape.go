// Port of src/terminal/mouse_shape.zig (Ghostty v1.3.1).
//
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

// mouseShapeNames are the tag names of MouseShape, in order.
var mouseShapeNames = [mouseShapeCount]string{
	"default",
	"context_menu",
	"help",
	"pointer",
	"progress",
	"wait",
	"cell",
	"crosshair",
	"text",
	"vertical_text",
	"alias",
	"copy",
	"move",
	"no_drop",
	"not_allowed",
	"grab",
	"grabbing",
	"all_scroll",
	"col_resize",
	"row_resize",
	"n_resize",
	"e_resize",
	"s_resize",
	"w_resize",
	"ne_resize",
	"nw_resize",
	"se_resize",
	"sw_resize",
	"ew_resize",
	"ns_resize",
	"nesw_resize",
	"nwse_resize",
	"zoom_in",
	"zoom_out",
}

// String returns the Zig tag name (@tagName).
func (self MouseShape) String() string { return mouseShapeNames[self] }

// MouseShapeFromString is MouseShape.fromString: build cursor shape from
// string or null if its unknown.
func MouseShapeFromString(v []uint8) Optional[MouseShape] {
	if shape, ok := mouseShapeStringMap[string(v)]; ok {
		return Some(shape)
	}
	return None[MouseShape]()
}

// mouseShapeStringMap is mouse_shape.string_map:
// std.StaticStringMap(MouseShape), the W3C names followed by the xterm/foot
// aliases, in upstream's order. Lookup is an exact match; the keys are
// distinct, so a Go map behaves as the static map does.
var mouseShapeStringMap = map[string]MouseShape{
	// W3C
	"default":       MouseShapeDefault,
	"context-menu":  MouseShapeContextMenu,
	"help":          MouseShapeHelp,
	"pointer":       MouseShapePointer,
	"progress":      MouseShapeProgress,
	"wait":          MouseShapeWait,
	"cell":          MouseShapeCell,
	"crosshair":     MouseShapeCrosshair,
	"text":          MouseShapeText,
	"vertical-text": MouseShapeVerticalText,
	"alias":         MouseShapeAlias,
	"copy":          MouseShapeCopy,
	"move":          MouseShapeMove,
	"no-drop":       MouseShapeNoDrop,
	"not-allowed":   MouseShapeNotAllowed,
	"grab":          MouseShapeGrab,
	"grabbing":      MouseShapeGrabbing,
	"all-scroll":    MouseShapeAllScroll,
	"col-resize":    MouseShapeColResize,
	"row-resize":    MouseShapeRowResize,
	"n-resize":      MouseShapeNResize,
	"e-resize":      MouseShapeEResize,
	"s-resize":      MouseShapeSResize,
	"w-resize":      MouseShapeWResize,
	"ne-resize":     MouseShapeNeResize,
	"nw-resize":     MouseShapeNwResize,
	"se-resize":     MouseShapeSeResize,
	"sw-resize":     MouseShapeSwResize,
	"ew-resize":     MouseShapeEwResize,
	"ns-resize":     MouseShapeNsResize,
	"nesw-resize":   MouseShapeNeswResize,
	"nwse-resize":   MouseShapeNwseResize,
	"zoom-in":       MouseShapeZoomIn,
	"zoom-out":      MouseShapeZoomOut,

	// xterm/foot
	"left_ptr":            MouseShapeDefault,
	"question_arrow":      MouseShapeHelp,
	"hand":                MouseShapePointer,
	"left_ptr_watch":      MouseShapeProgress,
	"watch":               MouseShapeWait,
	"cross":               MouseShapeCrosshair,
	"xterm":               MouseShapeText,
	"dnd-link":            MouseShapeAlias,
	"dnd-copy":            MouseShapeCopy,
	"dnd-move":            MouseShapeMove,
	"dnd-no-drop":         MouseShapeNoDrop,
	"crossed_circle":      MouseShapeNotAllowed,
	"hand1":               MouseShapeGrab,
	"right_side":          MouseShapeEResize,
	"top_side":            MouseShapeNResize,
	"top_right_corner":    MouseShapeNeResize,
	"top_left_corner":     MouseShapeNwResize,
	"bottom_side":         MouseShapeSResize,
	"bottom_right_corner": MouseShapeSeResize,
	"bottom_left_corner":  MouseShapeSwResize,
	"left_side":           MouseShapeWResize,
	"fleur":               MouseShapeAllScroll,
}
