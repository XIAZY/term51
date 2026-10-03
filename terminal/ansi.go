// WIREFRAME for src/terminal/ansi.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: ansi.zig is a namespace file, so its declarations take the
// prefix `ANSI`.
//
// Declarations appear in the same order as in ansi.zig. The C ABI selection
// (`lib_target`, and the C variant of every `lib.Enum`) is not ported; see
// docs/mapping.md. A `lib.Enum` with the Zig target is an ordinary enum
// whose values are numbered from zero in the order listed.

package terminal

// ANSIC0 is ansi.C0: C0 (7-bit) control characters from ANSI.
//
// This is not complete, control characters are only added to this as the
// terminal emulator handles them.
//
// Zig: enum(u7), non-exhaustive so that @enumFromInt never fails since the
// inputs are user-generated. A Go uint8 type has the same property.
type ANSIC0 uint8

const (
	// Null
	ANSIC0NUL ANSIC0 = 0x00
	// Start of heading
	ANSIC0SOH ANSIC0 = 0x01
	// Start of text
	ANSIC0STX ANSIC0 = 0x02
	// Enquiry
	ANSIC0ENQ ANSIC0 = 0x05
	// Bell
	ANSIC0BEL ANSIC0 = 0x07
	// Backspace
	ANSIC0BS ANSIC0 = 0x08
	// Horizontal tab
	ANSIC0HT ANSIC0 = 0x09
	// Line feed
	ANSIC0LF ANSIC0 = 0x0A
	// Vertical Tab
	ANSIC0VT ANSIC0 = 0x0B
	// Form feed
	ANSIC0FF ANSIC0 = 0x0C
	// Carriage return
	ANSIC0CR ANSIC0 = 0x0D
	// Shift out
	ANSIC0SO ANSIC0 = 0x0E
	// Shift in
	ANSIC0SI ANSIC0 = 0x0F
)

// String returns the Zig tag name (@tagName).
func (self ANSIC0) String() string { panic(notImplemented) }

// ANSIRenditionAspect is ansi.RenditionAspect: the SGR rendition aspects
// that can be set, sometimes known as attributes. The value corresponds to
// the parameter value for the SGR command (ESC [ m).
//
// Zig: enum(u16), non-exhaustive.
type ANSIRenditionAspect uint16

const (
	ANSIRenditionAspectDefault   ANSIRenditionAspect = 0
	ANSIRenditionAspectBold      ANSIRenditionAspect = 1
	ANSIRenditionAspectDefaultFg ANSIRenditionAspect = 39
	ANSIRenditionAspectDefaultBg ANSIRenditionAspect = 49
)

// String returns the Zig tag name (@tagName).
func (self ANSIRenditionAspect) String() string { panic(notImplemented) }

// ANSIDeviceAttributeReq is ansi.DeviceAttributeReq: the device attribute
// request type (ESC [ c).
//
// Zig: lib.Enum(lib_target, ...).
type ANSIDeviceAttributeReq uint8

const (
	ANSIDeviceAttributeReqPrimary   ANSIDeviceAttributeReq = iota // Blank
	ANSIDeviceAttributeReqSecondary                               // >
	ANSIDeviceAttributeReqTertiary                                // =
)

// String returns the Zig tag name (@tagName).
func (self ANSIDeviceAttributeReq) String() string { panic(notImplemented) }

// ANSICursorStyle is ansi.CursorStyle: possible cursor styles (ESC [ q).
//
// Zig: lib.Enum(lib_target, ...).
type ANSICursorStyle uint8

const (
	ANSICursorStyleDefault ANSICursorStyle = iota
	ANSICursorStyleBlinkingBlock
	ANSICursorStyleSteadyBlock
	ANSICursorStyleBlinkingUnderline
	ANSICursorStyleSteadyUnderline
	ANSICursorStyleBlinkingBar
	ANSICursorStyleSteadyBar
)

// String returns the Zig tag name (@tagName).
func (self ANSICursorStyle) String() string { panic(notImplemented) }

// ANSIStatusLineType is ansi.StatusLineType: the status line type for
// DECSSDT.
//
// Zig: enum(u16), non-exhaustive so that @enumFromInt never fails for
// unsupported values.
type ANSIStatusLineType uint16

const (
	ANSIStatusLineTypeNone         ANSIStatusLineType = 0
	ANSIStatusLineTypeIndicator    ANSIStatusLineType = 1
	ANSIStatusLineTypeHostWritable ANSIStatusLineType = 2
)

// String returns the Zig tag name (@tagName).
func (self ANSIStatusLineType) String() string { panic(notImplemented) }

// ANSIStatusDisplay is ansi.StatusDisplay: the display to target for status
// updates (DECSASD).
//
// Zig: lib.Enum(lib_target, ...).
type ANSIStatusDisplay uint8

const (
	ANSIStatusDisplayMain ANSIStatusDisplay = iota
	ANSIStatusDisplayStatusLine
)

// String returns the Zig tag name (@tagName).
func (self ANSIStatusDisplay) String() string { panic(notImplemented) }

// ANSIModifyKeyFormat is ansi.ModifyKeyFormat: the possible modify key
// formats to ESC[>{a};{b}m.
// Note: this is not complete, we should add more as we support more.
//
// Zig: lib.Enum(lib_target, ...).
type ANSIModifyKeyFormat uint8

const (
	ANSIModifyKeyFormatLegacy ANSIModifyKeyFormat = iota
	ANSIModifyKeyFormatCursorKeys
	ANSIModifyKeyFormatFunctionKeys
	ANSIModifyKeyFormatOtherKeysNone
	ANSIModifyKeyFormatOtherKeysNumericExcept
	ANSIModifyKeyFormatOtherKeysNumeric
)

// String returns the Zig tag name (@tagName).
func (self ANSIModifyKeyFormat) String() string { panic(notImplemented) }

// ANSIProtectedMode is ansi.ProtectedMode: the protection modes that can be
// set for the terminal. See DECSCA and ESC V, W.
type ANSIProtectedMode uint8

const (
	ANSIProtectedModeOff ANSIProtectedMode = iota
	ANSIProtectedModeIso                   // ESC V, W
	ANSIProtectedModeDec                   // CSI Ps " q
)

// String returns the Zig tag name (@tagName).
func (self ANSIProtectedMode) String() string { panic(notImplemented) }
