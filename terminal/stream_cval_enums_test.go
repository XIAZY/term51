package terminal

import (
	"testing"
)

// These two tests are not one-to-one translations. They keep the plain
// enum-value checks of the Zig tests
//
//	test "term51: Action.DeviceStatus.cval: is the packed request tag"
//	test "term51: Action.Mode.cval: is the packed mode tag"
//
// in src/terminal/stream.zig (section A), whose cval/C-binding parts are not
// ported. The integer values of the DeviceStatus and Mode tags are checked
// directly and through DeviceStatusTag.Int() and ModesModeTag.Int().

func TestStreamEnums_DeviceStatusValues(t *testing.T) {
	// Plain requests are the bare value.
	expectEqual(t, uint16(5), uint16(DeviceStatusRequestOperatingStatus))
	expectEqual(t, uint16(6), uint16(DeviceStatusRequestCursorPosition))
	// The "?" bit is the top bit of the u16.
	expectEqual(t, uint16(996|0x8000), uint16(DeviceStatusRequestColorScheme))
	expectEqual(t, uint16(0x83E4), uint16(DeviceStatusRequestColorScheme))
	tag := DeviceStatusTag{value: 996, question: true}
	expectEqual(t, tag.Int(), uint16(DeviceStatusRequestColorScheme))
}

func TestStreamEnums_ModeValues(t *testing.T) {
	// DEC private modes are the bare number.
	expectEqual(t, uint16(1), uint16(ModesModeCursorKeys))
	expectEqual(t, uint16(6), uint16(ModesModeOrigin))
	expectEqual(t, uint16(66), uint16(ModesModeKeypadKeys))
	expectEqual(t, uint16(1049), uint16(ModesModeAltScreenSaveCursorClearEnter))
	expectEqual(t, uint16(2048), uint16(ModesModeInBandSizeReports))
	// ANSI modes have the top bit set.
	expectEqual(t, uint16(4|0x8000), uint16(ModesModeInsert))
	expectEqual(t, uint16(2|0x8000), uint16(ModesModeDisableKeyboard))
	expectEqual(t, uint16(20|0x8000), uint16(ModesModeLinefeed))
	// DEC 4 (slow scroll) and ANSI 4 (insert) differ only by that bit.
	expectEqual(t, uint16(4), uint16(ModesModeSlowScroll))

	// The same values through the packed tag.
	expectEqual(t, ModesModeTag{value: 1}.Int(), uint16(ModesModeCursorKeys))
	expectEqual(t, ModesModeTag{value: 1049}.Int(), uint16(ModesModeAltScreenSaveCursorClearEnter))
	expectEqual(t, ModesModeTag{value: 4, ansi: true}.Int(), uint16(ModesModeInsert))
	expectEqual(t, ModesModeTag{value: 4}.Int(), uint16(ModesModeSlowScroll))
}
