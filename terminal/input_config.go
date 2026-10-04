// WIREFRAME for src/input/config.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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
// "right". It is also the name std.meta.stringToEnum matches.
func (self InputConfigOptionAsAlt) String() string { panic(notImplemented) }
