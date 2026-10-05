// Port of src/terminal/device_status.zig (Ghostty v1.3.1).
//
// File naming: device_status.zig is a namespace file, so its declarations
// take the prefix `DeviceStatus` (`deviceStatus` for private ones).
//
// Declarations appear in the same order as in device_status.zig. The file
// has no C ABI declarations.
//
// Zig generates the `Request` enum at compile time from `entries` (the
// `dsr_enum:` block). Go has no type generation: the constants are written
// out (their values are what the block computes, which the tests check),
// and the field list the block builds (names and values, in the order of
// `entries`) is computed once at start-up by deviceStatusRequestInit, from
// `entries`, the same way. String() reads it.

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

// deviceStatusRequestField is std.builtin.Type.EnumField: one field of the
// generated Request enum.
type deviceStatusRequestField struct {
	name  []uint8
	value DeviceStatusTagBacking
}

// deviceStatusRequestFields are the fields of the Request enum, built at
// start-up from `entries` (Zig: at compile time).
var deviceStatusRequestFields = deviceStatusRequestInit()

// deviceStatusRequestInit is the `dsr_enum:` block of `Request`: one enum
// field per entry, whose value is the entry's Tag bit-cast to Tag.Backing.
func deviceStatusRequestInit() []deviceStatusRequestField {
	fields := make([]deviceStatusRequestField, len(deviceStatusEntries))
	for i, entry := range deviceStatusEntries {
		fields[i] = deviceStatusRequestField{
			name: entry.name,
			value: DeviceStatusTag{
				value:    uint16(entry.value),
				question: entry.question,
			}.Int(),
		}
	}

	return fields
}

// String returns the Zig tag name (@tagName). The enum is exhaustive: a
// value outside it panics, as @tagName on an invalid value does.
func (self DeviceStatusRequest) String() string {
	for _, field := range deviceStatusRequestFields {
		if field.value == DeviceStatusTagBacking(self) {
			return string(field.name)
		}
	}
	panic("unreachable")
}

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
//
// `value` is a u15 in Zig, so it cannot exceed 0x7FFF; the Go field is a
// uint16, and a larger value is asserted against instead of being dropped.
func (self DeviceStatusTag) Int() DeviceStatusTagBacking {
	assert(self.value <= 0x7FFF)
	var question DeviceStatusTagBacking = 0
	if self.question {
		question = 1
	}
	return self.value | question<<15
}

// DeviceStatusTagFromInt stands for Zig's `@bitCast` from the backing
// integer (Tag.Backing, a u16) to a Tag: the reverse of DeviceStatusTag.Int.
// Every u16 is a valid Tag.
func DeviceStatusTagFromInt(v DeviceStatusTagBacking) DeviceStatusTag {
	return DeviceStatusTag{
		value:    v & 0x7FFF,
		question: v>>15 != 0,
	}
}

// DeviceStatusReqFromInt is device_status.reqFromInt.
//
// Zig: `inline for (entries)`; a loop over the entries.
func DeviceStatusReqFromInt(v uint16, question bool) Optional[DeviceStatusRequest] {
	for _, entry := range deviceStatusEntries {
		if entry.value == int(v) && entry.question == question {
			tag := DeviceStatusTag{question: question, value: uint16(entry.value)}
			int_ := tag.Int()
			return Some(DeviceStatusRequest(int_))
		}
	}

	return None[DeviceStatusRequest]()
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
var deviceStatusEntries = []deviceStatusEntry{
	{name: []uint8("operating_status"), value: 5},
	{name: []uint8("cursor_position"), value: 6},
	{name: []uint8("color_scheme"), value: 996, question: true},
}
