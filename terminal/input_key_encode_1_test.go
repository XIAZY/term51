// Tests translated one for one from src/input/key_encode.zig (Ghostty v1.3.1
// plus the term51 additional tests), lines 1080 to 2318: the KittySequence
// tests, the kitty tests and the first legacy tests (up to "legacy:
// right_shift+tab"). Each Go test carries the exact Zig title in a "// Zig:
// test" comment. Helpers are prefixed ke1.
//
// Translation notes:
//   - `try testing.expectEqualStrings(want, writer.buffered())` is
//     expectEqualStrings(t, want, string(writer.buffered())); `try` on a call
//     that returns an error is zigTry(t, err).
//   - `var writer: std.Io.Writer = .fixed(&buf)` is
//     `writer := oscWriterFixed(buf[:])`.
//   - A Zig KeyEvent literal has `action` defaulting to `.press`, which is not
//     the Go zero value: every literal here writes `action: InputKeyActionPress`
//     (or `InputKeyActionRelease` / `InputKeyActionRepeat` where Zig says so).
//   - Zig's `kitty` and `legacy` are inputKeyEncodeKitty and
//     inputKeyEncodeLegacy; Zig's `Options` literal `.{}` is
//     InputKeyEncodeOptions{}, whose every field is the Zig default.
//   - Darwin-only tests (the SkipZigTest ones that run only on macOS) are not
//     translated; they are listed in scripts/not-translated-ke1.txt.

package terminal

import (
	"testing"
)

// ke1Kitty runs inputKeyEncodeKitty into a fresh 128-byte fixed writer and
// returns what was written. It stands for Zig's three lines
// `var buf: [128]u8 = undefined; var writer: std.Io.Writer = .fixed(&buf);
// try kitty(&writer, event, opts);` followed by `writer.buffered()`.
func ke1Kitty(t *testing.T, event InputKeyKeyEvent, opts InputKeyEncodeOptions) string {
	t.Helper()
	var buf [128]uint8
	writer := oscWriterFixed(buf[:])
	zigTry(t, inputKeyEncodeKitty(&writer, event, opts))
	return string(writer.buffered())
}

// ke1Legacy is ke1Kitty for inputKeyEncodeLegacy.
func ke1Legacy(t *testing.T, event InputKeyKeyEvent, opts InputKeyEncodeOptions) string {
	t.Helper()
	var buf [128]uint8
	writer := oscWriterFixed(buf[:])
	zigTry(t, inputKeyEncodeLegacy(&writer, event, opts))
	return string(writer.buffered())
}

// ke1Seq encodes a KittySequence into a fresh 128-byte fixed writer and
// returns what was written.
func ke1Seq(t *testing.T, seq inputKeyEncodeKittySequence) string {
	t.Helper()
	var buf [128]uint8
	writer := oscWriterFixed(buf[:])
	zigTry(t, seq.Encode(&writer))
	return string(writer.buffered())
}

// ke1FlagsAll is the flag set the Zig tests spell out as
// `.{ .disambiguate = true, .report_events = true, .report_alternates = true,
// .report_all = true, .report_associated = true }`.
func ke1FlagsAll() KittyKeyFlags {
	return KittyKeyFlags{
		disambiguate:      true,
		report_events:     true,
		report_alternates: true,
		report_all:        true,
		report_associated: true,
	}
}

// Zig: test "KittySequence: backspace"
func TestKe1_KittySequence_backspace(t *testing.T) {
	testPlatform(t, "linux")
	// Plain
	{
		seq := inputKeyEncodeKittySequence{key: 127, final: 'u'}
		expectEqualStrings(t, "\x1B[127u", ke1Seq(t, seq))
	}

	// Release event
	{
		seq := inputKeyEncodeKittySequence{key: 127, final: 'u', event: inputKeyEncodeKittySequenceEventRelease}
		expectEqualStrings(t, "\x1B[127;1:3u", ke1Seq(t, seq))
	}

	// Shift
	{
		seq := inputKeyEncodeKittySequence{
			key:   127,
			final: 'u',
			mods:  inputKeyEncodeKittyMods{shift: true},
		}
		expectEqualStrings(t, "\x1B[127;2u", ke1Seq(t, seq))
	}
}

// Zig: test "KittySequence: text"
func TestKe1_KittySequence_text(t *testing.T) {
	testPlatform(t, "linux")
	// Plain
	{
		seq := inputKeyEncodeKittySequence{
			key:   127,
			final: 'u',
			text:  []uint8("A"),
		}
		expectEqualStrings(t, "\x1B[127;;65u", ke1Seq(t, seq))
	}

	// Release
	{
		seq := inputKeyEncodeKittySequence{
			key:   127,
			final: 'u',
			event: inputKeyEncodeKittySequenceEventRelease,
			text:  []uint8("A"),
		}
		expectEqualStrings(t, "\x1B[127;1:3;65u", ke1Seq(t, seq))
	}

	// Shift
	{
		seq := inputKeyEncodeKittySequence{
			key:   127,
			final: 'u',
			mods:  inputKeyEncodeKittyMods{shift: true},
			text:  []uint8("A"),
		}
		expectEqualStrings(t, "\x1B[127;2;65u", ke1Seq(t, seq))
	}
}

// Zig: test "KittySequence: text with control characters"
func TestKe1_KittySequence_text_with_control_characters(t *testing.T) {
	testPlatform(t, "linux")
	// By itself
	{
		seq := inputKeyEncodeKittySequence{
			key:   127,
			final: 'u',
			text:  []uint8("\n"),
		}
		expectEqualStrings(t, "\x1b[127u", ke1Seq(t, seq))
	}

	// With other printables
	{
		seq := inputKeyEncodeKittySequence{
			key:   127,
			final: 'u',
			text:  []uint8("A\n"),
		}
		expectEqualStrings(t, "\x1b[127;;65u", ke1Seq(t, seq))
	}
}

// Zig: test "KittySequence: special no mods"
func TestKe1_KittySequence_special_no_mods(t *testing.T) {
	testPlatform(t, "linux")
	seq := inputKeyEncodeKittySequence{key: 1, final: 'A'}
	expectEqualStrings(t, "\x1B[A", ke1Seq(t, seq))
}

// Zig: test "KittySequence: special mods only"
func TestKe1_KittySequence_special_mods_only(t *testing.T) {
	testPlatform(t, "linux")
	seq := inputKeyEncodeKittySequence{key: 1, final: 'A', mods: inputKeyEncodeKittyMods{shift: true}}
	expectEqualStrings(t, "\x1B[1;2A", ke1Seq(t, seq))
}

// Zig: test "KittySequence: special mods and event"
func TestKe1_KittySequence_special_mods_and_event(t *testing.T) {
	testPlatform(t, "linux")
	seq := inputKeyEncodeKittySequence{
		key:   1,
		final: 'A',
		event: inputKeyEncodeKittySequenceEventRelease,
		mods:  inputKeyEncodeKittyMods{shift: true},
	}
	expectEqualStrings(t, "\x1B[1;2:3A", ke1Seq(t, seq))
}

// Zig: test "kitty: plain text"
func TestKe1_kitty_plain_text(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyKeyA,
		mods:   InputKeyMods{},
		utf8:   []uint8("abcd"),
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true},
	})
	expectEqualStrings(t, "abcd", got)
}

// Zig: test "kitty: repeat with just disambiguate"
func TestKe1_kitty_repeat_with_just_disambiguate(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		key:    InputKeyKeyKeyA,
		action: InputKeyActionRepeat,
		mods:   InputKeyMods{},
		utf8:   []uint8("a"),
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true},
	})
	expectEqualStrings(t, "a", got)
}

// Zig: test "kitty: enter, backspace, tab"
func TestKe1_kitty_enter_backspace_tab(t *testing.T) {
	testPlatform(t, "linux")
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{disambiguate: true},
		})
		expectEqualStrings(t, "\r", got)
	}
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{disambiguate: true},
		})
		expectEqualStrings(t, "\x7f", got)
	}
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{disambiguate: true},
		})
		expectEqualStrings(t, "\t", got)
	}

	// No release events if "report_all" is not set
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEnter, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{disambiguate: true, report_events: true},
		})
		expectEqualStrings(t, "", got)
	}
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyBackspace, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{disambiguate: true, report_events: true},
		})
		expectEqualStrings(t, "", got)
	}
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyTab, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{disambiguate: true, report_events: true},
		})
		expectEqualStrings(t, "", got)
	}

	// Release events if "report_all" is set
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEnter, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{
				disambiguate:  true,
				report_events: true,
				report_all:    true,
			},
		})
		expectEqualStrings(t, "\x1b[13;1:3u", got)
	}
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyBackspace, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{
				disambiguate:  true,
				report_events: true,
				report_all:    true,
			},
		})
		expectEqualStrings(t, "\x1b[127;1:3u", got)
	}
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyTab, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{
				disambiguate:  true,
				report_events: true,
				report_all:    true,
			},
		})
		expectEqualStrings(t, "\x1b[9;1:3u", got)
	}
}

// Zig: test "kitty: shift+backspace emits CSI u"
func TestKe1_kitty_shift_backspace_emits_CSI_u(t *testing.T) {
	testPlatform(t, "linux")
	// Backspace with shift modifier should emit CSI u sequence, not raw 0x7F.
	// This is important for programs that want to distinguish shift+backspace.
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyBackspace,
		mods:   InputKeyMods{shift: true},
		utf8:   []uint8(""),
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true},
	})
	expectEqualStrings(t, "\x1b[127;2u", got)
}

// Zig: test "kitty: shift+enter emits CSI u"
func TestKe1_kitty_shift_enter_emits_CSI_u(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyEnter,
		mods:   InputKeyMods{shift: true},
		utf8:   []uint8(""),
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true},
	})
	expectEqualStrings(t, "\x1b[13;2u", got)
}

// Zig: test "kitty: shift+tab emits CSI u"
func TestKe1_kitty_shift_tab_emits_CSI_u(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyTab,
		mods:   InputKeyMods{shift: true},
		utf8:   []uint8(""),
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true},
	})
	expectEqualStrings(t, "\x1b[9;2u", got)
}

// Zig: test "kitty: enter with all flags"
func TestKe1_kitty_enter_with_all_flags(t *testing.T) {
	testPlatform(t, "linux")
	actual := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{}, utf8: []uint8("")}, InputKeyEncodeOptions{
		kitty_flags: ke1FlagsAll(),
	})
	expectEqualStrings(t, "[13u", actual[1:])
}

// Zig: test "kitty: ctrl with all flags"
func TestKe1_kitty_ctrl_with_all_flags(t *testing.T) {
	testPlatform(t, "linux")
	actual := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyControlLeft, mods: InputKeyMods{ctrl: true}, utf8: []uint8("")}, InputKeyEncodeOptions{
		kitty_flags: ke1FlagsAll(),
	})
	expectEqualStrings(t, "[57442;5u", actual[1:])
}

// Zig: test "kitty: ctrl release with ctrl mod set"
func TestKe1_kitty_ctrl_release_with_ctrl_mod_set(t *testing.T) {
	testPlatform(t, "linux")
	actual := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionRelease,
		key:    InputKeyKeyControlLeft,
		mods:   InputKeyMods{ctrl: true},
		utf8:   []uint8(""),
	}, InputKeyEncodeOptions{
		kitty_flags: ke1FlagsAll(),
	})
	expectEqualStrings(t, "[57442;5:3u", actual[1:])
}

// Zig: test "kitty: delete"
func TestKe1_kitty_delete(t *testing.T) {
	testPlatform(t, "linux")
	{
		got := ke1Kitty(t, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyDelete, mods: InputKeyMods{}, utf8: []uint8("\x7F")}, InputKeyEncodeOptions{
			kitty_flags: KittyKeyFlags{disambiguate: true},
		})
		expectEqualStrings(t, "\x1b[3~", got)
	}
}

// Zig: test "kitty: composing with no modifier"
func TestKe1_kitty_composing_with_no_modifier(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:    InputKeyActionPress,
		key:       InputKeyKeyKeyA,
		mods:      InputKeyMods{shift: true},
		composing: true,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true},
	})
	expectEqualStrings(t, "", got)
}

// Zig: test "kitty: composing with modifier"
func TestKe1_kitty_composing_with_modifier(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:    InputKeyActionPress,
		key:       InputKeyKeyShiftLeft,
		mods:      InputKeyMods{shift: true},
		composing: true,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true, report_all: true},
	})
	expectEqualStrings(t, "\x1b[57441;2u", got)
}

// Zig: test "kitty: composed text with report all"
func TestKe1_kitty_composed_text_with_report_all(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyUnidentified,
		mods:   InputKeyMods{},
		utf8:   []uint8("\xc3\xbb"), // û
	}, InputKeyEncodeOptions{
		kitty_flags: ke1FlagsAll(),
	})
	expectEqualStrings(t, "\xc3\xbb", got)
}

// Zig: test "kitty: shift+a on US keyboard"
func TestKe1_kitty_shift_a_on_US_keyboard(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyKeyA,
		mods:                InputKeyMods{shift: true},
		utf8:                []uint8("A"),
		unshifted_codepoint: 97, // lowercase A
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_alternates: true,
		},
	})
	expectEqualStrings(t, "\x1b[97:65;2u", got)
}

// Zig: test "kitty: matching unshifted codepoint"
func TestKe1_kitty_matching_unshifted_codepoint(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyKeyA,
		mods:                InputKeyMods{shift: true},
		utf8:                []uint8("A"),
		unshifted_codepoint: 65,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_alternates: true,
		},
	})
	// WARNING: This is not a valid encoding. This is a hypothetical encoding
	// just to test that our logic is correct around matching unshifted
	// codepoints. We get an alternate here because the unshifted_codepoint does
	// not match the base key
	expectEqualStrings(t, "\x1b[65::97;2u", got)
}

// Zig: test "kitty: report alternates with caps"
func TestKe1_kitty_report_alternates_with_caps(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyKeyJ,
		mods:                InputKeyMods{caps_lock: true},
		utf8:                []uint8("J"),
		unshifted_codepoint: 106,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_all:        true,
			report_alternates: true,
			report_associated: true,
		},
	})
	expectEqualStrings(t, "\x1b[106;65;74u", got)
}

// Zig: test "kitty: report alternates colon (shift+';')"
func TestKe1_kitty_report_alternates_colon_shift_semicolon(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeySemicolon,
		mods:                InputKeyMods{shift: true},
		utf8:                []uint8(":"),
		unshifted_codepoint: ';',
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_all:        true,
			report_alternates: true,
			report_associated: true,
		},
	})
	expectEqualStrings(t, "\x1b[59:58;2;58u", got)
}

// Zig: test "kitty: report alternates with ru layout"
func TestKe1_kitty_report_alternates_with_ru_layout(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeySemicolon,
		mods:                InputKeyMods{},
		utf8:                []uint8("ч"),
		unshifted_codepoint: 1095,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_all:        true,
			report_alternates: true,
			report_associated: true,
		},
	})
	expectEqualStrings(t, "\x1b[1095::59;;1095u", got)
}

// Zig: test "kitty: report alternates with ru layout shifted"
func TestKe1_kitty_report_alternates_with_ru_layout_shifted(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeySemicolon,
		mods:                InputKeyMods{shift: true},
		utf8:                []uint8("Ч"),
		unshifted_codepoint: 1095,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_all:        true,
			report_alternates: true,
			report_associated: true,
		},
	})
	expectEqualStrings(t, "\x1b[1095:1063:59;2;1063u", got)
}

// Zig: test "kitty: report alternates with ru layout caps lock"
func TestKe1_kitty_report_alternates_with_ru_layout_caps_lock(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeySemicolon,
		mods:                InputKeyMods{caps_lock: true},
		utf8:                []uint8("Ч"),
		unshifted_codepoint: 1095,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_all:        true,
			report_alternates: true,
			report_associated: true,
		},
	})
	expectEqualStrings(t, "\x1b[1095::59;65;1063u", got)
}

// Zig: test "kitty: report alternates with hu layout release"
func TestKe1_kitty_report_alternates_with_hu_layout_release(t *testing.T) {
	testPlatform(t, "linux")
	actual := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionRelease,
		key:                 InputKeyKeyBracketLeft,
		mods:                InputKeyMods{ctrl: true},
		utf8:                []uint8(""),
		unshifted_codepoint: 337,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_all:        true,
			report_alternates: true,
			report_associated: true,
			report_events:     true,
		},
	})
	expectEqualStrings(t, "[337::91;5:3u", actual[1:])
}

// macOS generates utf8 text for arrow keys.
//
// Zig: test "kitty: up arrow with utf8"
func TestKe1_kitty_up_arrow_with_utf8(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyArrowUp,
		mods:   InputKeyMods{},
		utf8:   []uint8{30},
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true},
	})
	expectEqualStrings(t, "\x1b[A", got)
}

// Zig: test "kitty: shift+tab"
func TestKe1_kitty_shift_tab(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyTab,
		mods:   InputKeyMods{shift: true},
		utf8:   []uint8(""), // tab
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true, report_alternates: true},
	})
	expectEqualStrings(t, "\x1b[9;2u", got)
}

// Zig: test "kitty: left shift"
func TestKe1_kitty_left_shift(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyShiftLeft,
		mods:   InputKeyMods{},
		utf8:   []uint8(""),
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true, report_alternates: true},
	})
	expectEqualStrings(t, "", got)
}

// Zig: test "kitty: left shift with report all"
func TestKe1_kitty_left_shift_with_report_all(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyShiftLeft,
		mods:   InputKeyMods{},
		utf8:   []uint8(""),
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{disambiguate: true, report_all: true},
	})
	expectEqualStrings(t, "\x1b[57441u", got)
}

// Zig: test "kitty: report associated with modifiers"
func TestKe1_kitty_report_associated_with_modifiers(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyKeyJ,
		mods:                InputKeyMods{ctrl: true},
		utf8:                []uint8("j"),
		unshifted_codepoint: 106,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_all:        true,
			report_alternates: true,
			report_associated: true,
		},
	})
	expectEqualStrings(t, "\x1b[106;5u", got)
}

// Zig: test "kitty: report associated"
func TestKe1_kitty_report_associated(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyKeyJ,
		mods:                InputKeyMods{shift: true},
		utf8:                []uint8("J"),
		unshifted_codepoint: 106,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_all:        true,
			report_alternates: true,
			report_associated: true,
		},
	})
	expectEqualStrings(t, "\x1b[106:74;2;74u", got)
}

// Zig: test "kitty: report associated on release"
func TestKe1_kitty_report_associated_on_release(t *testing.T) {
	testPlatform(t, "linux")
	actual := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionRelease,
		key:                 InputKeyKeyKeyJ,
		mods:                InputKeyMods{shift: true},
		utf8:                []uint8("J"),
		unshifted_codepoint: 106,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_all:        true,
			report_alternates: true,
			report_associated: true,
			report_events:     true,
		},
	})
	expectEqualStrings(t, "[106:74;2:3u", actual[1:])
}

// Zig: test "kitty: alternates omit control characters"
func TestKe1_kitty_alternates_omit_control_characters(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyDelete,
		mods:   InputKeyMods{},
		utf8:   []uint8{0x7F},
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_alternates: true,
			report_all:        true,
		},
	})
	expectEqualStrings(t, "\x1b[3~", got)
}

// Zig: test "kitty: enter with utf8 (dead key state)"
func TestKe1_kitty_enter_with_utf8_dead_key_state(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyEnter,
		utf8:                []uint8("A"),
		unshifted_codepoint: 0x0D,
	}, InputKeyEncodeOptions{
		kitty_flags: KittyKeyFlags{
			disambiguate:      true,
			report_alternates: true,
			report_all:        true,
		},
	})
	expectEqualStrings(t, "A", got)
}

// Zig: test "kitty: keypad number"
func TestKe1_kitty_keypad_number(t *testing.T) {
	testPlatform(t, "linux")
	actual := ke1Kitty(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyNumpad1,
		mods:   InputKeyMods{},
		utf8:   []uint8("1"),
	}, InputKeyEncodeOptions{
		kitty_flags: ke1FlagsAll(),
	})
	expectEqualStrings(t, "[57400;;49u", actual[1:])
}

// Zig: test "kitty: backspace with utf8 (dead key state)"
func TestKe1_kitty_backspace_with_utf8_dead_key_state(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Kitty(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyBackspace,
		utf8:                []uint8("A"),
		unshifted_codepoint: 0x0D,
	}, InputKeyEncodeOptions{
		kitty_flags: ke1FlagsAll(),
	})
	expectEqualStrings(t, "", got)
}

// Zig: test "legacy: backspace with utf8 (dead key state)"
func TestKe1_legacy_backspace_with_utf8_dead_key_state(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyBackspace,
		utf8:                []uint8("A"),
		unshifted_codepoint: 0x0D,
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "", got)
}

// Zig: test "legacy: enter with utf8 (dead key state)"
func TestKe1_legacy_enter_with_utf8_dead_key_state(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyEnter,
		utf8:                []uint8("A"),
		unshifted_codepoint: 0x0D,
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "A", got)
}

// Zig: test "legacy: esc with utf8 (dead key state)"
func TestKe1_legacy_esc_with_utf8_dead_key_state(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyEscape,
		utf8:                []uint8("A"),
		unshifted_codepoint: 0x0D,
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "A", got)
}

// Zig: test "legacy: ctrl+shift+minus (underscore on US)"
func TestKe1_legacy_ctrl_shift_minus_underscore_on_US(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyMinus,
		mods:   InputKeyMods{ctrl: true, shift: true},
		utf8:   []uint8("_"),
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "\x1F", got)
}

// Zig: test "legacy: ctrl+alt+c"
func TestKe1_legacy_ctrl_alt_c(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyKeyC,
		mods:   InputKeyMods{ctrl: true, alt: true},
		utf8:   []uint8("c"),
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "\x1b\x03", got)
}

// Zig: test "legacy: alt+c"
func TestKe1_legacy_alt_c(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyKeyC,
		utf8:   []uint8("c"),
		mods:   InputKeyMods{alt: true},
	}, InputKeyEncodeOptions{
		alt_esc_prefix:      true,
		macos_option_as_alt: InputConfigOptionAsAltTrue,
	})
	expectEqualStrings(t, "\x1Bc", got)
}

// Zig: test "legacy: alt+e only unshifted"
func TestKe1_legacy_alt_e_only_unshifted(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyKeyE,
		unshifted_codepoint: 'e',
		mods:                InputKeyMods{alt: true},
	}, InputKeyEncodeOptions{
		alt_esc_prefix:      true,
		macos_option_as_alt: InputConfigOptionAsAltTrue,
	})
	expectEqualStrings(t, "\x1Be", got)
}

// Zig: test "legacy: alt+ф"
func TestKe1_legacy_alt_ф(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyKeyF,
		utf8:   []uint8("ф"),
		mods:   InputKeyMods{alt: true},
	}, InputKeyEncodeOptions{
		alt_esc_prefix: true,
	})
	expectEqualStrings(t, "ф", got)
}

// Zig: test "legacy: ctrl+c"
func TestKe1_legacy_ctrl_c(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyKeyC,
		mods:   InputKeyMods{ctrl: true},
		utf8:   []uint8("c"),
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "\x03", got)
}

// Zig: test "legacy: ctrl+space"
func TestKe1_legacy_ctrl_space(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeySpace,
		mods:   InputKeyMods{ctrl: true},
		utf8:   []uint8(" "),
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "\x00", got)
}

// Zig: test "legacy: ctrl+shift+backspace"
func TestKe1_legacy_ctrl_shift_backspace(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyBackspace,
		mods:   InputKeyMods{ctrl: true, shift: true},
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "\x08", got)
}

// Zig: test "legacy: ctrl+shift+char with modify other state 2"
func TestKe1_legacy_ctrl_shift_char_with_modify_other_state_2(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyKeyH,
		mods:   InputKeyMods{ctrl: true, shift: true},
		utf8:   []uint8("H"),
	}, InputKeyEncodeOptions{
		modify_other_keys_state_2: true,
	})
	expectEqualStrings(t, "\x1b[27;6;72~", got)
}

// Zig: test "legacy: ctrl+shift+char with modify other state 2 and consumed mods"
func TestKe1_legacy_ctrl_shift_char_with_modify_other_state_2_and_consumed_mods(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:        InputKeyActionPress,
		key:           InputKeyKeyKeyH,
		mods:          InputKeyMods{ctrl: true, shift: true},
		consumed_mods: InputKeyMods{shift: true},
		utf8:          []uint8("H"),
	}, InputKeyEncodeOptions{
		modify_other_keys_state_2: true,
	})
	expectEqualStrings(t, "\x1b[27;6;72~", got)
}

// Zig: test "legacy: alt+digit with modify other state 2"
func TestKe1_legacy_alt_digit_with_modify_other_state_2(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:        InputKeyActionPress,
		key:           InputKeyKeyDigit8,
		mods:          InputKeyMods{alt: true},
		consumed_mods: InputKeyMods{},
		utf8:          []uint8("8"),
	}, InputKeyEncodeOptions{
		modify_other_keys_state_2: true,
		macos_option_as_alt:       InputConfigOptionAsAltTrue,
	})
	expectEqualStrings(t, "\x1b[27;3;56~", got)
}

// Zig: test "legacy: fixterm awkward letters"
func TestKe1_legacy_fixterm_awkward_letters(t *testing.T) {
	testPlatform(t, "linux")
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action: InputKeyActionPress,
			key:    InputKeyKeyKeyI,
			mods:   InputKeyMods{ctrl: true},
			utf8:   []uint8("i"),
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[105;5u", got)
	}
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action: InputKeyActionPress,
			key:    InputKeyKeyKeyM,
			mods:   InputKeyMods{ctrl: true},
			utf8:   []uint8("m"),
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[109;5u", got)
	}
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action: InputKeyActionPress,
			key:    InputKeyKeyBracketLeft,
			mods:   InputKeyMods{ctrl: true},
			utf8:   []uint8("["),
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[91;5u", got)
	}
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action:              InputKeyActionPress,
			key:                 InputKeyKeyDigit2,
			mods:                InputKeyMods{ctrl: true, shift: true},
			utf8:                []uint8("@"),
			unshifted_codepoint: '2',
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[64;5u", got)
	}
}

// These tests apply Kitty's behavior to CSIu where ctrl+shift+letter
// is sent as the unshifted letter with the shift modifier present.
//
// Zig: test "legacy: ctrl+shift+letter ascii"
func TestKe1_legacy_ctrl_shift_letter_ascii(t *testing.T) {
	testPlatform(t, "linux")
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action:              InputKeyActionPress,
			key:                 InputKeyKeyKeyM,
			mods:                InputKeyMods{ctrl: true, shift: true},
			utf8:                []uint8("M"),
			unshifted_codepoint: 'm',
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[109;6u", got)
	}
}

// Zig: test "legacy: shift+function key should use all mods"
func TestKe1_legacy_shift_function_key_should_use_all_mods(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:        InputKeyActionPress,
		key:           InputKeyKeyArrowUp,
		mods:          InputKeyMods{shift: true},
		consumed_mods: InputKeyMods{shift: true},
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "\x1b[1;2A", got)
}

// Zig: test "legacy: keypad enter"
func TestKe1_legacy_keypad_enter(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:        InputKeyActionPress,
		key:           InputKeyKeyNumpadEnter,
		mods:          InputKeyMods{},
		consumed_mods: InputKeyMods{},
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "\r", got)
}

// Zig: test "legacy: keypad 1"
func TestKe1_legacy_keypad_1(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:        InputKeyActionPress,
		key:           InputKeyKeyNumpad1,
		mods:          InputKeyMods{},
		consumed_mods: InputKeyMods{},
		utf8:          []uint8("1"),
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "1", got)
}

// Zig: test "legacy: keypad 1 with application keypad"
func TestKe1_legacy_keypad_1_with_application_keypad(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:        InputKeyActionPress,
		key:           InputKeyKeyNumpad1,
		mods:          InputKeyMods{},
		consumed_mods: InputKeyMods{},
		utf8:          []uint8("1"),
	}, InputKeyEncodeOptions{
		keypad_key_application: true,
	})
	expectEqualStrings(t, "\x1bOq", got)
}

// Zig: test "legacy: keypad 1 with application keypad and numlock"
func TestKe1_legacy_keypad_1_with_application_keypad_and_numlock(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:        InputKeyActionPress,
		key:           InputKeyKeyNumpad1,
		mods:          InputKeyMods{num_lock: true},
		consumed_mods: InputKeyMods{},
		utf8:          []uint8("1"),
	}, InputKeyEncodeOptions{
		keypad_key_application: true,
	})
	expectEqualStrings(t, "\x1bOq", got)
}

// Zig: test "legacy: keypad 1 with application keypad and numlock ignore"
func TestKe1_legacy_keypad_1_with_application_keypad_and_numlock_ignore(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action:        InputKeyActionPress,
		key:           InputKeyKeyNumpad1,
		mods:          InputKeyMods{num_lock: false},
		consumed_mods: InputKeyMods{},
		utf8:          []uint8("1"),
	}, InputKeyEncodeOptions{
		keypad_key_application:     true,
		ignore_keypad_with_numlock: true,
	})
	expectEqualStrings(t, "1", got)
}

// Zig: test "legacy: f1"
func TestKe1_legacy_f1(t *testing.T) {
	testPlatform(t, "linux")
	// F1
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action:        InputKeyActionPress,
			key:           InputKeyKeyF1,
			mods:          InputKeyMods{ctrl: true},
			consumed_mods: InputKeyMods{},
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[1;5P", got)
	}

	// F2
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action:        InputKeyActionPress,
			key:           InputKeyKeyF2,
			mods:          InputKeyMods{ctrl: true},
			consumed_mods: InputKeyMods{},
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[1;5Q", got)
	}

	// F3
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action:        InputKeyActionPress,
			key:           InputKeyKeyF3,
			mods:          InputKeyMods{ctrl: true},
			consumed_mods: InputKeyMods{},
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[13;5~", got)
	}

	// F4
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action:        InputKeyActionPress,
			key:           InputKeyKeyF4,
			mods:          InputKeyMods{ctrl: true},
			consumed_mods: InputKeyMods{},
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[1;5S", got)
	}

	// F5 uses new encoding
	{
		got := ke1Legacy(t, InputKeyKeyEvent{
			action:        InputKeyActionPress,
			key:           InputKeyKeyF5,
			mods:          InputKeyMods{ctrl: true},
			consumed_mods: InputKeyMods{},
		}, InputKeyEncodeOptions{})
		expectEqualStrings(t, "\x1b[15;5~", got)
	}
}

// Zig: test "legacy: left_shift+tab"
func TestKe1_legacy_left_shift_tab(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyTab,
		mods: InputKeyMods{
			shift: true,
			sides: InputKeyModsModsSide{shift: InputKeyModsModSideLeft},
		},
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "\x1b[Z", got)
}

// Zig: test "legacy: right_shift+tab"
func TestKe1_legacy_right_shift_tab(t *testing.T) {
	testPlatform(t, "linux")
	got := ke1Legacy(t, InputKeyKeyEvent{
		action: InputKeyActionPress,
		key:    InputKeyKeyTab,
		mods: InputKeyMods{
			shift: true,
			sides: InputKeyModsModsSide{shift: InputKeyModsModSideRight},
		},
	}, InputKeyEncodeOptions{})
	expectEqualStrings(t, "\x1b[Z", got)
}
