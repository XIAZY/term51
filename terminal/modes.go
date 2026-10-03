// WIREFRAME for src/terminal/modes.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file contains all the terminal modes that we support and various
// support types for them: an enum of supported modes, a packed struct to
// store mode values, a more generalized state struct to store values plus
// handle save/restore, and much more.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented) and every table is left at its zero value. The
// implementation is added only after the project owner and the project lead
// both approve it. Do not fill in bodies before then. See the header of
// color.go for the rules.
//
// File naming: modes.zig is a namespace file, so its declarations take the
// prefix `Modes` (`modes` for private ones): `modes.Mode` is `ModesMode`,
// `modes.ModeState` is `ModesModeState`.
//
// Declarations appear in the same order as in modes.zig. Zig generates the
// `Mode` enum and the `ModePacked` struct at compile time from `entries`;
// Go has no type generation, so both are written out, one constant and one
// field per entry, in the order of `entries`. The file has no C ABI
// declarations.
//
// OPEN POINT for the implementation: in Zig a default-initialised
// ModePacked (`.{}`) has every field at its entry's `default`, and six
// entries default to true (send_receive_mode, wraparound, cursor_visible,
// mouse_alternate_scroll, ignore_keypad_with_numlock, alt_esc_prefix). So
// `var state: ModeState = .{}` does not start all-false. A Go zero value is
// all-false, and modes.zig has no constructor to map. The wireframe does not
// decide how this is solved.

package terminal

// ModesModeState is modes.ModeState: a struct that maintains the state of
// all the settable modes.
//
// Zig gives every field the default `.{}`, a ModePacked with every mode at
// its default. See the open point in the file header.
type ModesModeState struct {
	// The values of the current modes.
	values ModesModePacked

	// The saved values. We only allow saving each mode once. This is in
	// line with other terminals that implement XTSAVE and XTRESTORE.
	saved ModesModePacked

	// The default values for the modes. This is used to reset the modes to
	// their default values during reset.
	//
	// Zig: `default`, a Go keyword.
	default_ ModesModePacked
}

// Reset is modes.ModeState.reset: reset the modes to their default values.
// This also clears the saved state.
func (self *ModesModeState) Reset() { panic(notImplemented) }

// Set is modes.ModeState.set: set a mode to a value.
func (self *ModesModeState) Set(mode ModesMode, value bool) { panic(notImplemented) }

// Get is modes.ModeState.get: get the value of a mode.
func (self *ModesModeState) Get(mode ModesMode) bool { panic(notImplemented) }

// Save is modes.ModeState.save: save the state of the given mode. This can
// then be restored with restore. This will only be accurate if the previous
// mode was saved exactly once and not restored. Otherwise this will just
// keep restoring the last stored value in memory.
func (self *ModesModeState) Save(mode ModesMode) { panic(notImplemented) }

// Restore is modes.ModeState.restore: see save. This will return the
// restored value.
func (self *ModesModeState) Restore(mode ModesMode) bool { panic(notImplemented) }

// ModesModePacked is modes.ModePacked: a packed struct of all the settable
// modes. This shouldn't be used directly but rather through the ModeState
// struct.
//
// Zig: a packed struct generated from `entries`, one bool per entry, named
// after the entry, default the entry's `default`. In Go it is an ordinary
// struct with the same fields in the same order. `132_column` begins with a
// digit, which a Go identifier cannot, so that field takes the prefix `X`
// (as in sgr.go).
type ModesModePacked struct {
	// ANSI
	disable_keyboard  bool
	insert            bool
	send_receive_mode bool // Zig: default true.
	linefeed          bool

	// DEC
	cursor_keys                        bool
	x132_column                        bool
	slow_scroll                        bool
	reverse_colors                     bool
	origin                             bool
	wraparound                         bool // Zig: default true.
	autorepeat                         bool
	mouse_event_x10                    bool
	cursor_blinking                    bool
	cursor_visible                     bool // Zig: default true.
	enable_mode_3                      bool
	reverse_wrap                       bool
	alt_screen_legacy                  bool
	keypad_keys                        bool
	enable_left_and_right_margin       bool
	mouse_event_normal                 bool
	mouse_event_button                 bool
	mouse_event_any                    bool
	focus_event                        bool
	mouse_format_utf8                  bool
	mouse_format_sgr                   bool
	mouse_alternate_scroll             bool // Zig: default true.
	mouse_format_urxvt                 bool
	mouse_format_sgr_pixels            bool
	ignore_keypad_with_numlock         bool // Zig: default true.
	alt_esc_prefix                     bool // Zig: default true.
	alt_sends_escape                   bool
	reverse_wrap_extended              bool
	alt_screen                         bool
	save_cursor                        bool
	alt_screen_save_cursor_clear_enter bool
	bracketed_paste                    bool
	synchronized_output                bool
	grapheme_cluster                   bool
	report_color_scheme                bool
	in_band_size_reports               bool
}

// ModesModeStateInit stands for Zig's `ModeState{}`: every field takes its
// declared default, which for `values`, `saved` and `default` is
// `ModePacked{}` (see ModesModePackedInit). Zig has no named constructor;
// Go needs one because its zero value is all false, and six modes default to
// true.
func ModesModeStateInit() ModesModeState { panic(notImplemented) }

// ModesModePackedInit stands for Zig's `ModePacked{}`: each mode at the
// default given by its entry in `entries`.
func ModesModePackedInit() ModesModePacked { panic(notImplemented) }

// ModesMode is modes.Mode: an enum(u16) of the available modes. See entries
// for available values.
//
// Zig builds the enum at compile time from `entries`: one value per entry,
// whose integer is the entry's ModeTag (value in the low 15 bits, the ansi
// bit in bit 15) cast to u16. The values are written out here, in the order
// of `entries`.
type ModesMode uint16

const (
	// ANSI
	ModesModeDisableKeyboard ModesMode = 2 | 0x8000 // KAM
	ModesModeInsert          ModesMode = 4 | 0x8000
	ModesModeSendReceiveMode ModesMode = 12 | 0x8000 // SRM
	ModesModeLinefeed        ModesMode = 20 | 0x8000

	// DEC
	ModesModeCursorKeys                    ModesMode = 1 // DECCKM
	ModesMode132Column                     ModesMode = 3
	ModesModeSlowScroll                    ModesMode = 4
	ModesModeReverseColors                 ModesMode = 5
	ModesModeOrigin                        ModesMode = 6
	ModesModeWraparound                    ModesMode = 7
	ModesModeAutorepeat                    ModesMode = 8
	ModesModeMouseEventX10                 ModesMode = 9
	ModesModeCursorBlinking                ModesMode = 12
	ModesModeCursorVisible                 ModesMode = 25
	ModesModeEnableMode3                   ModesMode = 40
	ModesModeReverseWrap                   ModesMode = 45
	ModesModeAltScreenLegacy               ModesMode = 47
	ModesModeKeypadKeys                    ModesMode = 66
	ModesModeEnableLeftAndRightMargin      ModesMode = 69
	ModesModeMouseEventNormal              ModesMode = 1000
	ModesModeMouseEventButton              ModesMode = 1002
	ModesModeMouseEventAny                 ModesMode = 1003
	ModesModeFocusEvent                    ModesMode = 1004
	ModesModeMouseFormatUtf8               ModesMode = 1005
	ModesModeMouseFormatSgr                ModesMode = 1006
	ModesModeMouseAlternateScroll          ModesMode = 1007
	ModesModeMouseFormatUrxvt              ModesMode = 1015
	ModesModeMouseFormatSgrPixels          ModesMode = 1016
	ModesModeIgnoreKeypadWithNumlock       ModesMode = 1035
	ModesModeAltEscPrefix                  ModesMode = 1036
	ModesModeAltSendsEscape                ModesMode = 1039
	ModesModeReverseWrapExtended           ModesMode = 1045
	ModesModeAltScreen                     ModesMode = 1047
	ModesModeSaveCursor                    ModesMode = 1048
	ModesModeAltScreenSaveCursorClearEnter ModesMode = 1049
	ModesModeBracketedPaste                ModesMode = 2004
	ModesModeSynchronizedOutput            ModesMode = 2026
	ModesModeGraphemeCluster               ModesMode = 2027
	ModesModeReportColorScheme             ModesMode = 2031
	ModesModeInBandSizeReports             ModesMode = 2048
)

// String returns the Zig tag name (@tagName).
func (self ModesMode) String() string { panic(notImplemented) }

// ModesModeTag is modes.ModeTag: the tag type for our enum is a u16 but we
// use a packed struct in order to pack the ansi bit into the tag.
//
// Zig: packed struct(u16). In Go it is an ordinary struct; Int is the
// conversion Zig writes as @bitCast.
type ModesModeTag struct {
	// Zig: u15.
	value uint16
	// Zig: default false.
	ansi bool
}

// ModesModeTagBacking is modes.ModeTag.Backing: the backing integer of
// ModeTag.
type ModesModeTagBacking = uint16

// Int is Zig's `@bitCast(tag)` to ModeTag.Backing: value in bits 0 to 14,
// ansi in bit 15. Zig has no named function for it.
func (self ModesModeTag) Int() ModesModeTagBacking { panic(notImplemented) }

// ModesModeFromInt is modes.modeFromInt.
func ModesModeFromInt(v uint16, ansi bool) Optional[ModesMode] { panic(notImplemented) }

// ModesModeTagFromInt stands for Zig's `@bitCast` from the backing integer
// (ModeTag.Backing, a u16) to a ModeTag: the reverse of ModesModeTag.Int.
func ModesModeTagFromInt(v ModesModeTagBacking) ModesModeTag { panic(notImplemented) }

// ModesModeValues stands for Zig's `std.enums.values(Mode)`: every mode, in
// the order of `entries`.
func ModesModeValues() []ModesMode { panic(notImplemented) }

// modesEntryForMode is modes.entryForMode.
//
// Zig: `mode` is a comptime parameter; an unknown mode is `unreachable`.
func modesEntryForMode(mode ModesMode) modesModeEntry { panic(notImplemented) }

// modesModeEntry is modes.ModeEntry: a single entry of a possible mode we
// support. This is used to dynamically define the enum and other tables.
type modesModeEntry struct {
	// Zig: [:0]const u8.
	name []uint8
	// Zig: comptime_int.
	value int
	// Zig: `default` (a Go keyword), default false.
	default_ bool

	// True if this is an ANSI mode, false if its a DEC mode (?-prefixed).
	ansi bool

	// If true, this mode is disabled and Ghostty will not allow it to be
	// set or queried. The mode enum still has it, allowing Ghostty
	// developers to develop a mode without exposing it to real users.
	disabled bool
}

// modesEntries is modes.entries: the full list of available entries. For
// documentation see how they're used within Ghostty or google their values.
//
// Wireframe: left nil; the implementation fills it with upstream's 40
// entries in upstream's order (the order of the ModesMode constants above).
var modesEntries []modesModeEntry
