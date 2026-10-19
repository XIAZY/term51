// Tests translated one for one from src/terminal/device_status.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order (helpers in Parser_test.go).
//   - `@typeInfo(Request).@"enum".fields` has no Go counterpart: a field name
//     is the String() of the constant, the field value is the integer
//     conversion, and the field count is the number of constants (3).
//   - `@bitCast(Tag{...})` is DeviceStatusTag.Int().
//   - The wireframe has no function for the reverse `@bitCast` from the
//     backing integer to a Tag (device_status.go has no counterpart of
//     modes.go's ModesModeTagFromInt); the one test that needs it keeps that
//     part as a marked comment (see the report).

package terminal

import (
	"reflect"
	"testing"
)

// Zig: test "order"
func TestDeviceStatus_order(t *testing.T) {
	tag := DeviceStatusTag{Value: 1}
	int_ := tag.Int()
	expectEqual(t, DeviceStatusTagBacking(1), int_)
}

// Zig: test "term51: Request: values and order"
func TestDeviceStatus_term51_Request_values_and_order(t *testing.T) {
	// Zig: info.fields: the constants of the enum, in declaration order.
	fields := [...]DeviceStatusRequest{
		DeviceStatusRequestOperatingStatus,
		DeviceStatusRequestCursorPosition,
		DeviceStatusRequestColorScheme,
	}
	expectEqual(t, 3, len(fields))
	expect(t, reflect.TypeOf(DeviceStatusRequest(0)).Kind() == reflect.Uint16, "info.tag_type == u16")
	// Zig-only: expect(info.is_exhaustive)
	expectEqualStrings(t, "operating_status", fields[0].String())
	expectEqualStrings(t, "cursor_position", fields[1].String())
	expectEqualStrings(t, "color_scheme", fields[2].String())
	expectEqual(t, uint16(5), uint16(DeviceStatusRequestOperatingStatus))
	expectEqual(t, uint16(6), uint16(DeviceStatusRequestCursorPosition))
	// The question bit is bit 15.
	expectEqual(t, uint16(0x8000|996), uint16(DeviceStatusRequestColorScheme))
}

// Zig: test "term51: Tag: layout of the value and question bit"
func TestDeviceStatus_term51_Tag_layout_of_the_value_and_question_bit(t *testing.T) {
	expect(t, reflect.TypeOf(DeviceStatusTagBacking(0)).Kind() == reflect.Uint16, "Tag.Backing == u16")
	expectEqual(t, uint16(5), DeviceStatusTag{Value: 5}.Int())
	expectEqual(t, uint16(0x8005), DeviceStatusTag{Value: 5, Question: true}.Int())
	expectEqual(t, uint16(0x7FFF), DeviceStatusTag{Value: 0x7FFF}.Int())
	expectEqual(t, uint16(0xFFFF), DeviceStatusTag{Value: 0x7FFF, Question: true}.Int())
	tag := DeviceStatusTagFromInt(uint16(0x83E4))
	expectEqual(t, uint16(996), tag.Value)
	expect(t, tag.Question, "t.question")
}

// Zig: test "term51: reqFromInt: supported requests"
func TestDeviceStatus_term51_reqFromInt_supported_requests(t *testing.T) {
	expectEqual(t, DeviceStatusRequestOperatingStatus, zigUnwrap(t, DeviceStatusReqFromInt(5, false)))
	expectEqual(t, DeviceStatusRequestCursorPosition, zigUnwrap(t, DeviceStatusReqFromInt(6, false)))
	expectEqual(t, DeviceStatusRequestColorScheme, zigUnwrap(t, DeviceStatusReqFromInt(996, true)))
}

// Zig: test "term51: reqFromInt: the question flag must match"
func TestDeviceStatus_term51_reqFromInt_the_question_flag_must_match(t *testing.T) {
	expect(t, !DeviceStatusReqFromInt(5, true).Valid, "reqFromInt(5, true) == null")
	expect(t, !DeviceStatusReqFromInt(6, true).Valid, "reqFromInt(6, true) == null")
	expect(t, !DeviceStatusReqFromInt(996, false).Valid, "reqFromInt(996, false) == null")
}

// Zig: test "term51: reqFromInt: values just outside the supported ones"
func TestDeviceStatus_term51_reqFromInt_values_just_outside_the_supported_ones(t *testing.T) {
	expect(t, !DeviceStatusReqFromInt(0, false).Valid, "reqFromInt(0, false) == null")
	expect(t, !DeviceStatusReqFromInt(0, true).Valid, "reqFromInt(0, true) == null")
	expect(t, !DeviceStatusReqFromInt(4, false).Valid, "reqFromInt(4, false) == null")
	expect(t, !DeviceStatusReqFromInt(7, false).Valid, "reqFromInt(7, false) == null")
	expect(t, !DeviceStatusReqFromInt(995, true).Valid, "reqFromInt(995, true) == null")
	expect(t, !DeviceStatusReqFromInt(997, true).Valid, "reqFromInt(997, true) == null")
	expect(t, !DeviceStatusReqFromInt(0x8005, false).Valid, "reqFromInt(0x8005, false) == null")
	expect(t, !DeviceStatusReqFromInt(0x8005, true).Valid, "reqFromInt(0x8005, true) == null")
	expect(t, !DeviceStatusReqFromInt(0x83E4, true).Valid, "reqFromInt(0x83E4, true) == null")
	expect(t, !DeviceStatusReqFromInt(65535, true).Valid, "reqFromInt(65535, true) == null")
	expect(t, !DeviceStatusReqFromInt(65535, false).Valid, "reqFromInt(65535, false) == null")
}

// Zig: test "term51: reqFromInt: exhaustive over every u16 and both prefixes"
func TestDeviceStatus_term51_reqFromInt_exhaustive_over_every_u16_and_both_prefixes(t *testing.T) {
	found := 0
	for v := uint32(0); v <= 0xFFFF; v++ {
		n := uint16(v)
		for _, question := range [...]bool{false, true} {
			var want Optional[DeviceStatusRequest]
			if !question && n == 5 {
				want = Some(DeviceStatusRequestOperatingStatus)
			} else if !question && n == 6 {
				want = Some(DeviceStatusRequestCursorPosition)
			} else if question && n == 996 {
				want = Some(DeviceStatusRequestColorScheme)
			} else {
				want = None[DeviceStatusRequest]()
			}
			expectEqual(t, want, DeviceStatusReqFromInt(n, question))
			if want.Valid {
				found += 1
			}
		}
	}
	expectEqual(t, 3, found)
}
