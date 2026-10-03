// WIREFRAME for src/terminal/cursor.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: cursor.zig is a namespace file, so its declarations take the
// prefix `Cursor`: `cursor.Style` is `CursorStyle`.
//
// Declarations appear in the same order as in cursor.zig. The file declares
// one enum and no function; it has no C ABI declarations and no tests.
//
// `Screen.CursorStyle` (Screen.zig line 30) is an alias of this enum; in Go
// it is `type ScreenCursorStyle = CursorStyle` with the constant aliases
// `ScreenCursorStyle<Value>`, in Screen.go.
//
// # Declarations that are not Zig declarations
//
//   - the `String` method.
//
// # Zig defaults a Go literal does not give
//
// The enum itself has no default. Its Go zero value is `.bar`, and the one
// field of this type, `Screen.Cursor.cursor_style`, defaults to `.block`; see
// the header of Screen.go.

package terminal

// CursorStyle is cursor.Style.
//
// The visual style of the cursor. Whether or not it blinks
// is determined by mode 12 (modes.zig). This mode is synchronized
// with CSI q, the same as xterm.
//
// Zig: enum { bar, block, underline, block_hollow }, values numbered from
// zero in the order listed.
type CursorStyle uint8

const (
	CursorStyleBar       CursorStyle = iota // DECSCUSR 5, 6
	CursorStyleBlock                        // DECSCUSR 1, 2
	CursorStyleUnderline                    // DECSCUSR 3, 4

	// The cursor styles below aren't known by DESCUSR and are custom
	// implemented in Ghostty. They are reported as some standard style
	// if requested, though.
	// Hollow block cursor. This is a block cursor with the center empty.
	// Reported as DECSCUSR 1 or 2 (block).
	CursorStyleBlockHollow
)

// String returns the Zig tag name (@tagName).
func (self CursorStyle) String() string { panic(notImplemented) }
