// WIREFRAME for src/terminal/device_status.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented) and every table is left at its zero value. The
// implementation is added only after the project owner and the project lead
// both approve it. Do not fill in bodies before then. See the header of
// color.go for the rules.
//
// File naming: device_status.zig is a namespace file, so its declarations
// take the prefix `DeviceStatus` (`deviceStatus` for private ones).
//
// Declarations appear in the same order as in device_status.zig. The file
// has no C ABI declarations.

package terminal

// DeviceStatusRequest is device_status.Request: an enum(u16) of the
// available device status requests.
//
// Zig builds the enum at compile time from `entries`: one value per entry,
// whose integer is the entry's Tag (value in the low 15 bits, the question
// bit in bit 15) cast to u16. The values are written out here.
type DeviceStatusRequest uint16

const (
	DeviceStatusRequestOperatingStatus DeviceStatusRequest = 5
	DeviceStatusRequestCursorPosition  DeviceStatusRequest = 6
	DeviceStatusRequestColorScheme     DeviceStatusRequest = 996 | 0x8000 // "?" request
)

// String returns the Zig tag name (@tagName).
func (self DeviceStatusRequest) String() string { panic(notImplemented) }

// DeviceStatusTag is device_status.Tag: the tag type for our enum is a u16
// but we use a packed struct in order to pack the question bit into the
// tag. The "u16" size is chosen somewhat arbitrarily to match the largest
// expected size we see as a multiple of 8 bits.
//
// Zig: packed struct(u16). In Go it is an ordinary struct; Int is the
// conversion Zig writes as @bitCast.
type DeviceStatusTag struct {
	// Zig: u15.
	value uint16
	// Zig: default false.
	question bool
}

// DeviceStatusTagBacking is device_status.Tag.Backing: the backing integer
// of Tag.
type DeviceStatusTagBacking = uint16

// Int is Zig's `@bitCast(tag)` to Tag.Backing: value in bits 0 to 14,
// question in bit 15. Zig has no named function for it.
func (self DeviceStatusTag) Int() DeviceStatusTagBacking { panic(notImplemented) }

// DeviceStatusReqFromInt is device_status.reqFromInt.
func DeviceStatusReqFromInt(v uint16, question bool) Optional[DeviceStatusRequest] {
	panic(notImplemented)
}

// deviceStatusEntry is device_status.Entry: a single entry of a possible
// device status request we support. The "question" field determines if it
// is valid with or without the "?" prefix.
type deviceStatusEntry struct {
	// Zig: [:0]const u8.
	name []uint8
	// Zig: comptime_int.
	value int
	// "?" request. Zig: default false.
	question bool
}

// deviceStatusEntries is device_status.entries: the full list of device
// status request entries.
//
// Wireframe: left nil; the implementation fills it with upstream's three
// entries (operating_status 5, cursor_position 6, color_scheme 996 with
// question) in upstream's order.
var deviceStatusEntries []deviceStatusEntry
