// Tests translated one for one from src/terminal/ansi.zig (Ghostty v1.3.1
// plus the term51 additional tests; ansi.zig has no upstream tests). Each Go
// test carries the exact Zig title in a "// Zig: test" comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order (helpers in Parser_test.go).
//   - `@typeInfo(E).@"enum".fields` has no Go counterpart: a field name is
//     the String() of the value, a field value is the integer conversion of
//     the constant, and the field count is the value of the last constant
//     plus one (the values are numbered from zero, in order).
//   - Zig's `u7` (C0) is a uint8 in Go, so `@intFromEnum(C0.X)` compares as
//     uint8. The enums that Zig declares non-exhaustive (`_`) are plain
//     integer types in Go, where any value converts; `is_exhaustive` has no
//     counterpart and is `// Zig-only:`.

package terminal

import (
	"testing"
)

// Zig: test "term51: C0: control character values"
func TestANSI_term51_C0_control_character_values(t *testing.T) {
	expectEqual(t, uint8(0x00), uint8(ANSIC0NUL))
	expectEqual(t, uint8(0x01), uint8(ANSIC0SOH))
	expectEqual(t, uint8(0x02), uint8(ANSIC0STX))
	expectEqual(t, uint8(0x05), uint8(ANSIC0ENQ))
	expectEqual(t, uint8(0x07), uint8(ANSIC0BEL))
	expectEqual(t, uint8(0x08), uint8(ANSIC0BS))
	expectEqual(t, uint8(0x09), uint8(ANSIC0HT))
	expectEqual(t, uint8(0x0A), uint8(ANSIC0LF))
	expectEqual(t, uint8(0x0B), uint8(ANSIC0VT))
	expectEqual(t, uint8(0x0C), uint8(ANSIC0FF))
	expectEqual(t, uint8(0x0D), uint8(ANSIC0CR))
	expectEqual(t, uint8(0x0E), uint8(ANSIC0SO))
	expectEqual(t, uint8(0x0F), uint8(ANSIC0SI))
	// Zig-only: try testing.expectEqual(@as(usize, 13), @typeInfo(C0).@"enum".fields.len);
	// (the C0 values are not consecutive, so Go has no field count)
}

// Zig: test "term51: C0: non-exhaustive for unnamed values"
func TestANSI_term51_C0_non_exhaustive_for_unnamed_values(t *testing.T) {
	// Zig-only: expect(!@typeInfo(C0).@"enum".is_exhaustive) and
	// expect(@typeInfo(C0).@"enum".tag_type == u7) (Go has no u7; the type is
	// uint8).
	unnamed := ANSIC0(0x03)
	expectEqual(t, uint8(3), uint8(unnamed))
	expect(t, unnamed != ANSIC0NUL, "unnamed != .NUL")
	max := ANSIC0(0x7F)
	expectEqual(t, uint8(0x7F), uint8(max))
	// Named values still compare equal when built from ints.
	bel := ANSIC0(7)
	expect(t, bel == ANSIC0BEL, "bel == .BEL")
}

// Zig: test "term51: RenditionAspect: values and non-exhaustive"
func TestANSI_term51_RenditionAspect_values_and_non_exhaustive(t *testing.T) {
	expectEqual(t, uint16(0), uint16(ANSIRenditionAspectDefault))
	expectEqual(t, uint16(1), uint16(ANSIRenditionAspectBold))
	expectEqual(t, uint16(39), uint16(ANSIRenditionAspectDefaultFg))
	expectEqual(t, uint16(49), uint16(ANSIRenditionAspectDefaultBg))
	// Zig-only: expect(!@typeInfo(RenditionAspect).@"enum".is_exhaustive)
	other := ANSIRenditionAspect(65535)
	expectEqual(t, uint16(65535), uint16(other))
	bold := ANSIRenditionAspect(1)
	expect(t, bold == ANSIRenditionAspectBold, "bold == .bold")
}

// Zig: test "term51: DeviceAttributeReq: names and order"
func TestANSI_term51_DeviceAttributeReq_names_and_order(t *testing.T) {
	expectEqual(t, 3, int(ANSIDeviceAttributeReqTertiary)+1) // f.len
	expectEqualStrings(t, "primary", ANSIDeviceAttributeReq(0).String())
	expectEqualStrings(t, "secondary", ANSIDeviceAttributeReq(1).String())
	expectEqualStrings(t, "tertiary", ANSIDeviceAttributeReq(2).String())
	expectEqual(t, 0, int(ANSIDeviceAttributeReqPrimary))
	expectEqual(t, 1, int(ANSIDeviceAttributeReqSecondary))
	expectEqual(t, 2, int(ANSIDeviceAttributeReqTertiary))
}

// Zig: test "term51: CursorStyle: names and order"
func TestANSI_term51_CursorStyle_names_and_order(t *testing.T) {
	names := [...]string{
		"default",
		"blinking_block",
		"steady_block",
		"blinking_underline",
		"steady_underline",
		"blinking_bar",
		"steady_bar",
	}
	// f[i].value: the constants in declaration order.
	values := [...]ANSICursorStyle{
		ANSICursorStyleDefault,
		ANSICursorStyleBlinkingBlock,
		ANSICursorStyleSteadyBlock,
		ANSICursorStyleBlinkingUnderline,
		ANSICursorStyleSteadyUnderline,
		ANSICursorStyleBlinkingBar,
		ANSICursorStyleSteadyBar,
	}
	expectEqual(t, len(names), int(ANSICursorStyleSteadyBar)+1) // f.len
	for i, name := range names {
		expectEqualStrings(t, name, ANSICursorStyle(i).String())
		expectEqual(t, i, int(values[i]))
	}
}

// Zig: test "term51: StatusLineType: values and non-exhaustive"
func TestANSI_term51_StatusLineType_values_and_non_exhaustive(t *testing.T) {
	expectEqual(t, uint16(0), uint16(ANSIStatusLineTypeNone))
	expectEqual(t, uint16(1), uint16(ANSIStatusLineTypeIndicator))
	expectEqual(t, uint16(2), uint16(ANSIStatusLineTypeHostWritable))
	// Zig-only: expect(!@typeInfo(StatusLineType).@"enum".is_exhaustive)
	other := ANSIStatusLineType(3)
	expectEqual(t, uint16(3), uint16(other))
}

// Zig: test "term51: StatusDisplay: names and order"
func TestANSI_term51_StatusDisplay_names_and_order(t *testing.T) {
	expectEqual(t, 2, int(ANSIStatusDisplayStatusLine)+1) // f.len
	expectEqualStrings(t, "main", ANSIStatusDisplay(0).String())
	expectEqualStrings(t, "status_line", ANSIStatusDisplay(1).String())
	expectEqual(t, 0, int(ANSIStatusDisplayMain))
	expectEqual(t, 1, int(ANSIStatusDisplayStatusLine))
}

// Zig: test "term51: ModifyKeyFormat: names and order"
func TestANSI_term51_ModifyKeyFormat_names_and_order(t *testing.T) {
	names := [...]string{
		"legacy",
		"cursor_keys",
		"function_keys",
		"other_keys_none",
		"other_keys_numeric_except",
		"other_keys_numeric",
	}
	// f[i].value: the constants in declaration order.
	values := [...]ANSIModifyKeyFormat{
		ANSIModifyKeyFormatLegacy,
		ANSIModifyKeyFormatCursorKeys,
		ANSIModifyKeyFormatFunctionKeys,
		ANSIModifyKeyFormatOtherKeysNone,
		ANSIModifyKeyFormatOtherKeysNumericExcept,
		ANSIModifyKeyFormatOtherKeysNumeric,
	}
	expectEqual(t, len(names), int(ANSIModifyKeyFormatOtherKeysNumeric)+1) // f.len
	for i, name := range names {
		expectEqualStrings(t, name, ANSIModifyKeyFormat(i).String())
		expectEqual(t, i, int(values[i]))
	}
}

// Zig: test "term51: ProtectedMode: names and order"
func TestANSI_term51_ProtectedMode_names_and_order(t *testing.T) {
	expectEqual(t, 3, int(ANSIProtectedModeDec)+1) // f.len
	expectEqualStrings(t, "off", ANSIProtectedMode(0).String())
	expectEqualStrings(t, "iso", ANSIProtectedMode(1).String())
	expectEqualStrings(t, "dec", ANSIProtectedMode(2).String())
	expectEqual(t, 0, int(ANSIProtectedModeOff))
	expectEqual(t, 1, int(ANSIProtectedModeIso))
	expectEqual(t, 2, int(ANSIProtectedModeDec))
}
