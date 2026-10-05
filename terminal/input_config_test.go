// Tests translated one for one from src/input/config.zig (Ghostty v1.3.1 plus
// the term51 additional tests). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. Helpers are prefixed kp.
//
// Translation notes:
//   - `@intFromEnum` is the conversion to int32 (the Go type is
//     `enum(c_int)`'s int32); `@enumFromInt` is the conversion from an
//     integer.
//   - `@typeInfo(OptionAsAlt).@"enum".fields.len` is
//     inputConfigOptionAsAltCount and `fields[i].name` is the String method
//     of the value i (the wireframe declares both for this purpose).
//   - `std.meta.stringToEnum` is kpConfigStringToEnum, which matches the
//     String method of every value.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import "testing"

// kpConfigStringToEnum is `std.meta.stringToEnum(OptionAsAlt, s)`: the value
// whose tag name is s, if any.
func kpConfigStringToEnum(s string) Optional[InputConfigOptionAsAlt] {
	for v := InputConfigOptionAsAlt(0); v < inputConfigOptionAsAltCount; v++ {
		if v.String() == s {
			return Optional[InputConfigOptionAsAlt]{Valid: true, Value: v}
		}
	}
	return Optional[InputConfigOptionAsAlt]{}
}

// Zig: test "term51: OptionAsAlt: every value and its integer"
func TestKpConfig_OptionAsAlt_every_value_and_its_integer(t *testing.T) {
	expectEqual(t, int32(0), int32(InputConfigOptionAsAltFalse))
	expectEqual(t, int32(1), int32(InputConfigOptionAsAltTrue))
	expectEqual(t, int32(2), int32(InputConfigOptionAsAltLeft))
	expectEqual(t, int32(3), int32(InputConfigOptionAsAltRight))
}

// Zig: test "term51: OptionAsAlt: from integer"
func TestKpConfig_OptionAsAlt_from_integer(t *testing.T) {
	expectEqual(t, InputConfigOptionAsAltFalse, InputConfigOptionAsAlt(0))
	expectEqual(t, InputConfigOptionAsAltTrue, InputConfigOptionAsAlt(1))
	expectEqual(t, InputConfigOptionAsAltLeft, InputConfigOptionAsAlt(2))
	expectEqual(t, InputConfigOptionAsAltRight, InputConfigOptionAsAlt(3))
}

// Zig: test "term51: OptionAsAlt: names, count and order"
func TestKpConfig_OptionAsAlt_names_count_and_order(t *testing.T) {
	expectEqual(t, 4, inputConfigOptionAsAltCount)
	expectEqualStrings(t, "false", InputConfigOptionAsAlt(0).String())
	expectEqualStrings(t, "true", InputConfigOptionAsAlt(1).String())
	expectEqualStrings(t, "left", InputConfigOptionAsAlt(2).String())
	expectEqualStrings(t, "right", InputConfigOptionAsAlt(3).String())
	expectEqual(t, int32(0), int32(InputConfigOptionAsAlt(0)))
	expectEqual(t, int32(1), int32(InputConfigOptionAsAlt(1)))
	expectEqual(t, int32(2), int32(InputConfigOptionAsAlt(2)))
	expectEqual(t, int32(3), int32(InputConfigOptionAsAlt(3)))
}

// Zig: test "term51: OptionAsAlt: string conversion"
func TestKpConfig_OptionAsAlt_string_conversion(t *testing.T) {
	expectEqual(t, InputConfigOptionAsAltFalse, zigUnwrap(t, kpConfigStringToEnum("false")))
	expectEqual(t, InputConfigOptionAsAltTrue, zigUnwrap(t, kpConfigStringToEnum("true")))
	expectEqual(t, InputConfigOptionAsAltLeft, zigUnwrap(t, kpConfigStringToEnum("left")))
	expectEqual(t, InputConfigOptionAsAltRight, zigUnwrap(t, kpConfigStringToEnum("right")))

	// Case sensitive, and no other spellings.
	expect(t, !kpConfigStringToEnum("True").Valid, `stringToEnum(OptionAsAlt, "True") == null`)
	expect(t, !kpConfigStringToEnum("LEFT").Valid, `stringToEnum(OptionAsAlt, "LEFT") == null`)
	expect(t, !kpConfigStringToEnum("both").Valid, `stringToEnum(OptionAsAlt, "both") == null`)
	expect(t, !kpConfigStringToEnum("").Valid, `stringToEnum(OptionAsAlt, "") == null`)
	expect(t, !kpConfigStringToEnum("0").Valid, `stringToEnum(OptionAsAlt, "0") == null`)
}
