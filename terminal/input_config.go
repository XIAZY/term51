// Port of src/input/config.zig (Ghostty v1.3.1).
//
// File naming: config.zig lives in src/input/, a sibling of src/terminal/,
// so the subdirectory is part of the name: `input_config.go`, and every
// declaration takes the prefix `InputConfig`: `config.OptionAsAlt` is
// `InputConfigOptionAsAlt`.
//
// The file has one declaration and no C ABI declarations (`OptionAsAlt` is
// `enum(c_int)` only so that the embedding API can use it as is; the Go type
// is int32, as `OSCCommandProgressReportState` in osc.go).
//
// # Declarations that are not Zig declarations
//
//   - `inputConfigOptionAsAltCount`, the number of values (Zig's
//     `@typeInfo(OptionAsAlt).@"enum".fields.len`), and the `String` method.
//
// # Zig defaults a Go literal does not give
//
// None: the file has no struct.

package terminal

// InputConfigOptionAsAlt is config.OptionAsAlt.
//
// Determines the macOS option key behavior. See the config
// `macos-option-as-alt` for a lot more details.
//
// Zig: enum(c_int), values numbered from zero in the order listed.
type InputConfigOptionAsAlt int32

const (
	InputConfigOptionAsAltFalse InputConfigOptionAsAlt = iota
	InputConfigOptionAsAltTrue
	InputConfigOptionAsAltLeft
	InputConfigOptionAsAltRight

	inputConfigOptionAsAltCount = iota
)

// String returns the Zig tag name (@tagName): "false", "true", "left",
// "right". It is also the name std.meta.stringToEnum matches. A value
// outside the enum panics (index out of range), as @tagName of an invalid
// value of an exhaustive enum does.
func (self InputConfigOptionAsAlt) String() string { return inputConfigOptionAsAltNames[self] }

// inputConfigOptionAsAltNames holds the Zig tag names, indexed by value.
var inputConfigOptionAsAltNames = [inputConfigOptionAsAltCount]string{
	"false",
	"true",
	"left",
	"right",
}
