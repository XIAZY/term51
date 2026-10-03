// Tests translated one for one from src/terminal/mouse_shape.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualStrings and zigUnwrap are defined in Parser_test.go; helpers
// that do not exist there are prefixed col_.
//
// Translation notes:
//   - `@typeInfo(MouseShape).@"enum".fields.len` is mouseShapeCount, and the
//     enum tag name (`@tagName`) is String().
//   - `string_map.keys().len` is len(mouseShapeStringMap).
//   - `@intFromEnum(x)` on a MouseShape (enum(c_int)) is int32(x).

package terminal

import (
	"strings"
	"testing"
)

// Zig: test "ghostty.h MouseShape"
func TestMouseShape_ghostty_h_MouseShape(t *testing.T) {
	// Zig: the test is skipped for the library artifact (`.lib => return
	// error.SkipZigTest`): it needs application code (lib.checkGhosttyHEnum)
	// that is not part of libghostty-vt.
	t.Skip("term51: needs application code that is not part of libghostty-vt (error.SkipZigTest for the lib artifact)")
}

// Zig: test "cursor shape from string"
func TestMouseShape_cursor_shape_from_string(t *testing.T) {
	expectEqual(t, MouseShapeDefault, zigUnwrap(t, MouseShapeFromString([]uint8("default"))))
}

// Zig: test (unnamed, mouse_shape.zig:145)
func TestMouseShape_unnamed(t *testing.T) {
	// Zig-only: `_ = std.testing.refAllDeclsRecursive(@This());` forces the
	// compiler to analyse every declaration of the file; Go compiles all of
	// them anyway.
	_ = mouseShapeStringMap
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// term51_w3c_names is the term51 test helper: the canonical (W3C) name of
// every shape, indexed by the integer value of the enum.
var term51_w3c_names = [...]string{
	"default", "context-menu", "help", "pointer",
	"progress", "wait", "cell", "crosshair",
	"text", "vertical-text", "alias", "copy",
	"move", "no-drop", "not-allowed", "grab",
	"grabbing", "all-scroll", "col-resize", "row-resize",
	"n-resize", "e-resize", "s-resize", "w-resize",
	"ne-resize", "nw-resize", "se-resize", "sw-resize",
	"ew-resize", "ns-resize", "nesw-resize", "nwse-resize",
	"zoom-in", "zoom-out",
}

// Zig: test "term51: MouseShape.fromString: every W3C name maps to its enum value in order"
func TestMouseShape_term51_MouseShape_fromString_every_W3C_name_maps_to_its_enum_value_in_order(t *testing.T) {
	expectEqual(t, 34, len(term51_w3c_names))
	expectEqual(t, 34, mouseShapeCount) // @typeInfo(MouseShape).@"enum".fields.len

	for i, name := range term51_w3c_names {
		got := zigUnwrap(t, MouseShapeFromString([]uint8(name)))
		expectEqual(t, int32(i), int32(got))
		// The enum tag name is the W3C name with '-' replaced by '_'.
		// Zig-only: var buf: [32]u8, @memcpy and std.mem.replaceScalar;
		// Go: strings.ReplaceAll.
		expectEqualStrings(t, strings.ReplaceAll(name, "-", "_"), got.String())
	}
}

// Zig: test "term51: MouseShape.fromString: first and last enum values and explicit integers"
func TestMouseShape_term51_MouseShape_fromString_first_and_last_enum_values_and_explicit_integers(t *testing.T) {
	expectEqual(t, int32(0), int32(MouseShapeDefault))
	expectEqual(t, int32(33), int32(MouseShapeZoomOut))
	expectEqual(t, MouseShapeDefault, zigUnwrap(t, MouseShapeFromString([]uint8("default"))))
	expectEqual(t, MouseShapeZoomOut, zigUnwrap(t, MouseShapeFromString([]uint8("zoom-out"))))
	expectEqual(t, MouseShapeContextMenu, zigUnwrap(t, MouseShapeFromString([]uint8("context-menu"))))
	expectEqual(t, MouseShapeNwseResize, zigUnwrap(t, MouseShapeFromString([]uint8("nwse-resize"))))
}

// Zig: test "term51: MouseShape.fromString: every xterm/foot alias"
func TestMouseShape_term51_MouseShape_fromString_every_xterm_foot_alias(t *testing.T) {
	aliases := [...]struct {
		name  string
		shape MouseShape
	}{
		{"left_ptr", MouseShapeDefault},
		{"question_arrow", MouseShapeHelp},
		{"hand", MouseShapePointer},
		{"left_ptr_watch", MouseShapeProgress},
		{"watch", MouseShapeWait},
		{"cross", MouseShapeCrosshair},
		{"xterm", MouseShapeText},
		{"dnd-link", MouseShapeAlias},
		{"dnd-copy", MouseShapeCopy},
		{"dnd-move", MouseShapeMove},
		{"dnd-no-drop", MouseShapeNoDrop},
		{"crossed_circle", MouseShapeNotAllowed},
		{"hand1", MouseShapeGrab},
		{"right_side", MouseShapeEResize},
		{"top_side", MouseShapeNResize},
		{"top_right_corner", MouseShapeNeResize},
		{"top_left_corner", MouseShapeNwResize},
		{"bottom_side", MouseShapeSResize},
		{"bottom_right_corner", MouseShapeSeResize},
		{"bottom_left_corner", MouseShapeSwResize},
		{"left_side", MouseShapeWResize},
		{"fleur", MouseShapeAllScroll},
	}
	expectEqual(t, 22, len(aliases))
	for _, a := range aliases {
		expectEqual(t, a.shape, zigUnwrap(t, MouseShapeFromString([]uint8(a.name))))
	}

	// 34 W3C names plus 22 aliases.
	expectEqual(t, 56, len(mouseShapeStringMap)) // string_map.keys().len
}

// Zig: test "term51: MouseShape.fromString: unknown, empty, wrong case, and near-miss names"
func TestMouseShape_term51_MouseShape_fromString_unknown_empty_wrong_case_and_near_miss_names(t *testing.T) {
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("nosuchshape")))
	// Matching is case sensitive.
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("Default")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("DEFAULT")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("Context-Menu")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("XTERM")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("Fleur")))
	// Enum tag spelling (underscores) is not a W3C name.
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("context_menu")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("zoom_in")))
	// Alias spelling with the other separator is not accepted.
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("left-ptr")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("dnd_link")))
	// Whitespace and prefixes/suffixes are not trimmed or matched loosely.
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8(" default")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("default ")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("defaul")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("defaultt")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("n-resiz")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("hand2")))
	expectEqual(t, None[MouseShape](), MouseShapeFromString([]uint8("default\x00")))
}
