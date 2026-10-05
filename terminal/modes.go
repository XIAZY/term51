// Port of src/terminal/modes.zig (Ghostty v1.3.1).
//
// This file contains all the terminal modes that we support and various
// support types for them: an enum of supported modes, a packed struct to
// store mode values, a more generalized state struct to store values plus
// handle save/restore, and much more.
//
// There is pretty heavy comptime usage and type generation here. I don't
// love to have this sort of complexity but its a good way to ensure all our
// various types and logic remain in sync.
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
// What the generation computes besides the declarations is computed from
// `entries` in Go too:
//   - the field defaults of ModePacked (`.default_value_ptr = &entry.default`)
//     by ModesModePackedInit, which stands for Zig's `ModePacked{}`;
//   - the field list of the Mode enum (names and values) once at start-up
//     by modesModeInit (`modesModeFields`), which String() and
//     ModesModeValues read. The written-out constants equal the values it
//     computes (the tests check every one).
//
// Default initialisation: in Zig a default-initialised ModePacked (`.{}`)
// has every field at its entry's `default`, and six entries default to true
// (send_receive_mode, wraparound, cursor_visible, mouse_alternate_scroll,
// ignore_keypad_with_numlock, alt_esc_prefix). A Go zero value is
// all-false, so `.{}` is ModesModePackedInit() and `ModeState{}` is
// ModesModeStateInit(); a zero ModesModeState is NOT Zig's default state.
//
// `@field(packed, entry.name)` is modesModePackedField, which returns a
// pointer to the bool field Zig names entry.name. Zig resolves the
// `inline else` + `comptime entryForMode(...)` + `@field` of set, get, save
// and restore at compile time, so at run time each is a single field
// access. Following the rule for compile-time work (computed once at
// start-up, not on each call), two tables are built at start-up:
// modesModeIndex maps a Mode to the index of its entry, and
// modesModePackedFields holds, per entry, the field accessor `@field`
// resolves to (checked against modesModePackedField at start-up).
// set/get/save/restore, entryForMode, modeFromInt and String use them;
// modesEntryForMode and modesModePackedField keep their Zig meaning.

package terminal

// ModesModeState is modes.ModeState: a struct that maintains the state of
// all the settable modes.
//
// Zig gives every field the default `.{}`, a ModePacked with every mode at
// its default: ModesModeStateInit (see the file header).
type ModesModeState struct {
	// The values of the current modes.
	values ModesModePacked

	// The saved values. We only allow saving each mode once. This is in
	// line with other terminals that implement XTSAVE and XTRESTORE. We can
	// improve this in the future if it becomes a real-world issue but we
	// need to be aware of a DoS vector.
	saved ModesModePacked

	// The default values for the modes. This is used to reset the modes to
	// their default values during reset.
	//
	// Zig: `default`, a Go keyword.
	default_ ModesModePacked
}

// Reset is modes.ModeState.reset: reset the modes to their default values.
// This also clears the saved state.
func (self *ModesModeState) Reset() {
	self.values = self.default_
	self.saved = ModesModePackedInit()
}

// Set is modes.ModeState.set: set a mode to a value.
func (self *ModesModeState) Set(mode ModesMode, value bool) {
	// Zig: switch (mode) { inline else => |mode_comptime| ... }, resolved
	// at compile time; the tables are built at start-up.
	i := modesModeEntryIndex(mode)
	*modesModePackedFields[i](&self.values) = value
}

// Get is modes.ModeState.get: get the value of a mode.
//
// Zig takes `*const ModeState`; Go uses the pointer receiver of the other
// methods.
func (self *ModesModeState) Get(mode ModesMode) bool {
	// Zig: switch (mode) { inline else => |mode_comptime| ... }, resolved
	// at compile time; the tables are built at start-up.
	i := modesModeEntryIndex(mode)
	return *modesModePackedFields[i](&self.values)
}

// Save is modes.ModeState.save: save the state of the given mode. This can
// then be restored with restore. This will only be accurate if the previous
// mode was saved exactly once and not restored. Otherwise this will just
// keep restoring the last stored value in memory.
func (self *ModesModeState) Save(mode ModesMode) {
	// Zig: switch (mode) { inline else => |mode_comptime| ... }, resolved
	// at compile time; the tables are built at start-up.
	i := modesModeEntryIndex(mode)
	*modesModePackedFields[i](&self.saved) = *modesModePackedFields[i](&self.values)
}

// Restore is modes.ModeState.restore: see save. This will return the
// restored value.
func (self *ModesModeState) Restore(mode ModesMode) bool {
	// Zig: switch (mode) { inline else => |mode_comptime| ... }, resolved
	// at compile time; the tables are built at start-up.
	i := modesModeEntryIndex(mode)
	*modesModePackedFields[i](&self.values) = *modesModePackedFields[i](&self.saved)
	return *modesModePackedFields[i](&self.values)
}

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

// modesModePackedField stands for Zig's `@field(p, name)` on a ModePacked:
// a pointer to the field named name (an entry name). A name without a field
// is a compile error in Zig; here it panics.
func modesModePackedField(p *ModesModePacked, name []uint8) *bool {
	switch string(name) {
	// ANSI
	case "disable_keyboard":
		return &p.disable_keyboard
	case "insert":
		return &p.insert
	case "send_receive_mode":
		return &p.send_receive_mode
	case "linefeed":
		return &p.linefeed

	// DEC
	case "cursor_keys":
		return &p.cursor_keys
	case "132_column":
		return &p.x132_column
	case "slow_scroll":
		return &p.slow_scroll
	case "reverse_colors":
		return &p.reverse_colors
	case "origin":
		return &p.origin
	case "wraparound":
		return &p.wraparound
	case "autorepeat":
		return &p.autorepeat
	case "mouse_event_x10":
		return &p.mouse_event_x10
	case "cursor_blinking":
		return &p.cursor_blinking
	case "cursor_visible":
		return &p.cursor_visible
	case "enable_mode_3":
		return &p.enable_mode_3
	case "reverse_wrap":
		return &p.reverse_wrap
	case "alt_screen_legacy":
		return &p.alt_screen_legacy
	case "keypad_keys":
		return &p.keypad_keys
	case "enable_left_and_right_margin":
		return &p.enable_left_and_right_margin
	case "mouse_event_normal":
		return &p.mouse_event_normal
	case "mouse_event_button":
		return &p.mouse_event_button
	case "mouse_event_any":
		return &p.mouse_event_any
	case "focus_event":
		return &p.focus_event
	case "mouse_format_utf8":
		return &p.mouse_format_utf8
	case "mouse_format_sgr":
		return &p.mouse_format_sgr
	case "mouse_alternate_scroll":
		return &p.mouse_alternate_scroll
	case "mouse_format_urxvt":
		return &p.mouse_format_urxvt
	case "mouse_format_sgr_pixels":
		return &p.mouse_format_sgr_pixels
	case "ignore_keypad_with_numlock":
		return &p.ignore_keypad_with_numlock
	case "alt_esc_prefix":
		return &p.alt_esc_prefix
	case "alt_sends_escape":
		return &p.alt_sends_escape
	case "reverse_wrap_extended":
		return &p.reverse_wrap_extended
	case "alt_screen":
		return &p.alt_screen
	case "save_cursor":
		return &p.save_cursor
	case "alt_screen_save_cursor_clear_enter":
		return &p.alt_screen_save_cursor_clear_enter
	case "bracketed_paste":
		return &p.bracketed_paste
	case "synchronized_output":
		return &p.synchronized_output
	case "grapheme_cluster":
		return &p.grapheme_cluster
	case "report_color_scheme":
		return &p.report_color_scheme
	case "in_band_size_reports":
		return &p.in_band_size_reports
	}
	panic("ModePacked has no field " + string(name))
}

// modesModePackedFields is, for each entry of modesEntries (same index),
// the field `@field(packed, entry.name)` resolves to at compile time, as an
// accessor. modesModePackedFieldsCheck verifies at start-up that each one
// returns the same field as modesModePackedField(p, entry.name).
var modesModePackedFields = [...]func(p *ModesModePacked) *bool{
	func(p *ModesModePacked) *bool { return &p.disable_keyboard },                   // disable_keyboard
	func(p *ModesModePacked) *bool { return &p.insert },                             // insert
	func(p *ModesModePacked) *bool { return &p.send_receive_mode },                  // send_receive_mode
	func(p *ModesModePacked) *bool { return &p.linefeed },                           // linefeed
	func(p *ModesModePacked) *bool { return &p.cursor_keys },                        // cursor_keys
	func(p *ModesModePacked) *bool { return &p.x132_column },                        // 132_column
	func(p *ModesModePacked) *bool { return &p.slow_scroll },                        // slow_scroll
	func(p *ModesModePacked) *bool { return &p.reverse_colors },                     // reverse_colors
	func(p *ModesModePacked) *bool { return &p.origin },                             // origin
	func(p *ModesModePacked) *bool { return &p.wraparound },                         // wraparound
	func(p *ModesModePacked) *bool { return &p.autorepeat },                         // autorepeat
	func(p *ModesModePacked) *bool { return &p.mouse_event_x10 },                    // mouse_event_x10
	func(p *ModesModePacked) *bool { return &p.cursor_blinking },                    // cursor_blinking
	func(p *ModesModePacked) *bool { return &p.cursor_visible },                     // cursor_visible
	func(p *ModesModePacked) *bool { return &p.enable_mode_3 },                      // enable_mode_3
	func(p *ModesModePacked) *bool { return &p.reverse_wrap },                       // reverse_wrap
	func(p *ModesModePacked) *bool { return &p.alt_screen_legacy },                  // alt_screen_legacy
	func(p *ModesModePacked) *bool { return &p.keypad_keys },                        // keypad_keys
	func(p *ModesModePacked) *bool { return &p.enable_left_and_right_margin },       // enable_left_and_right_margin
	func(p *ModesModePacked) *bool { return &p.mouse_event_normal },                 // mouse_event_normal
	func(p *ModesModePacked) *bool { return &p.mouse_event_button },                 // mouse_event_button
	func(p *ModesModePacked) *bool { return &p.mouse_event_any },                    // mouse_event_any
	func(p *ModesModePacked) *bool { return &p.focus_event },                        // focus_event
	func(p *ModesModePacked) *bool { return &p.mouse_format_utf8 },                  // mouse_format_utf8
	func(p *ModesModePacked) *bool { return &p.mouse_format_sgr },                   // mouse_format_sgr
	func(p *ModesModePacked) *bool { return &p.mouse_alternate_scroll },             // mouse_alternate_scroll
	func(p *ModesModePacked) *bool { return &p.mouse_format_urxvt },                 // mouse_format_urxvt
	func(p *ModesModePacked) *bool { return &p.mouse_format_sgr_pixels },            // mouse_format_sgr_pixels
	func(p *ModesModePacked) *bool { return &p.ignore_keypad_with_numlock },         // ignore_keypad_with_numlock
	func(p *ModesModePacked) *bool { return &p.alt_esc_prefix },                     // alt_esc_prefix
	func(p *ModesModePacked) *bool { return &p.alt_sends_escape },                   // alt_sends_escape
	func(p *ModesModePacked) *bool { return &p.reverse_wrap_extended },              // reverse_wrap_extended
	func(p *ModesModePacked) *bool { return &p.alt_screen },                         // alt_screen
	func(p *ModesModePacked) *bool { return &p.save_cursor },                        // save_cursor
	func(p *ModesModePacked) *bool { return &p.alt_screen_save_cursor_clear_enter }, // alt_screen_save_cursor_clear_enter
	func(p *ModesModePacked) *bool { return &p.bracketed_paste },                    // bracketed_paste
	func(p *ModesModePacked) *bool { return &p.synchronized_output },                // synchronized_output
	func(p *ModesModePacked) *bool { return &p.grapheme_cluster },                   // grapheme_cluster
	func(p *ModesModePacked) *bool { return &p.report_color_scheme },                // report_color_scheme
	func(p *ModesModePacked) *bool { return &p.in_band_size_reports },               // in_band_size_reports
}

var _ = modesModePackedFieldsCheck()

// modesModePackedFieldsCheck asserts that modesModePackedFields matches
// modesEntries and modesModePackedField. It runs once at start-up.
func modesModePackedFieldsCheck() bool {
	assert(len(modesModePackedFields) == len(modesEntries))
	var p ModesModePacked
	for i, entry := range modesEntries {
		assert(modesModePackedFields[i](&p) == modesModePackedField(&p, entry.name))
	}
	return true
}

// ModesModeStateInit stands for Zig's `ModeState{}`: every field takes its
// declared default, which for `values`, `saved` and `default` is
// `ModePacked{}` (see ModesModePackedInit). Zig has no named constructor;
// Go needs one because its zero value is all false, and six modes default to
// true.
func ModesModeStateInit() ModesModeState {
	return ModesModeState{
		values:   ModesModePackedInit(),
		saved:    ModesModePackedInit(),
		default_: ModesModePackedInit(),
	}
}

// ModesModePackedInit stands for Zig's `ModePacked{}`: each mode at the
// default given by its entry in `entries`.
//
// Zig: the `packed_struct:` block gives field i of the generated struct
// `.default_value_ptr = &entry.default`; Go sets each field from its entry
// in the same loop over `entries`.
func ModesModePackedInit() ModesModePacked {
	var p ModesModePacked
	for _, entry := range modesEntries {
		*modesModePackedField(&p, entry.name) = entry.default_
	}
	return p
}

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

// modesModeField is std.builtin.Type.EnumField: one field of the generated
// Mode enum.
type modesModeField struct {
	name  []uint8
	value ModesModeTagBacking
}

// modesModeFields are the fields of the Mode enum, in the order of
// `entries`, built at start-up (Zig: at compile time).
var modesModeFields = modesModeInit()

// modesModeInit is the `mode_enum:` block of `Mode`: one enum field per
// entry, whose value is the entry's ModeTag bit-cast to ModeTag.Backing.
func modesModeInit() []modesModeField {
	fields := make([]modesModeField, len(modesEntries))
	for i, entry := range modesEntries {
		fields[i] = modesModeField{
			name: entry.name,
			value: ModesModeTag{
				value: uint16(entry.value),
				ansi:  entry.ansi,
			}.Int(),
		}
	}

	return fields
}

// String returns the Zig tag name (@tagName). The enum is exhaustive: a
// value outside it panics, as @tagName on an invalid value does.
func (self ModesMode) String() string {
	return string(modesModeFields[modesModeEntryIndex(self)].name)
}

// modesModeIndex maps a Mode (its u16 value) to the index of its entry in
// modesEntries (and of its field in modesModeFields), or
// modesModeIndexNone where no entry has that value. Built once at start-up
// from modesModeFields: Zig maps a Mode to its entry at compile time.
var modesModeIndex = modesModeIndexInit()

const modesModeIndexNone = 0xFF

func modesModeIndexInit() *[1 << 16]uint8 {
	var t [1 << 16]uint8
	for i := range t {
		t[i] = modesModeIndexNone
	}
	assert(len(modesModeFields) < modesModeIndexNone)
	for i, field := range modesModeFields {
		assert(t[field.value] == modesModeIndexNone)
		t[field.value] = uint8(i)
	}
	return &t
}

// modesModeEntryIndex is the index of mode's entry. A value that is not a
// Mode panics, as Zig's `unreachable` (an invalid enum value) would.
func modesModeEntryIndex(mode ModesMode) int {
	i := modesModeIndex[mode]
	if i == modesModeIndexNone {
		panic("unreachable")
	}
	return int(i)
}

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
//
// `value` is a u15 in Zig, so it cannot exceed 0x7FFF; the Go field is a
// uint16, and a larger value is asserted against instead of being dropped.
func (self ModesModeTag) Int() ModesModeTagBacking {
	assert(self.value <= 0x7FFF)
	var ansi ModesModeTagBacking = 0
	if self.ansi {
		ansi = 1
	}
	return self.value | ansi<<15
}

// ModesModeFromInt is modes.modeFromInt.
//
// Zig: `inline for (entries)`, unrolled at compile time into one
// comparison per enabled entry. Go looks the tag up in modesModeIndex
// (built at start-up) instead: an enabled entry with value v and this ansi
// bit exists exactly when the tag (v, ansi) is in the index and its entry
// is not disabled. No entry value exceeds a u15, so a larger v matches
// nothing.
func ModesModeFromInt(v uint16, ansi bool) Optional[ModesMode] {
	if v > 0x7FFF {
		return None[ModesMode]()
	}
	tag := ModesModeTag{ansi: ansi, value: v}
	int_ := tag.Int()
	if i := modesModeIndex[int_]; i != modesModeIndexNone && !modesEntries[i].disabled {
		return Some(ModesMode(int_))
	}

	return None[ModesMode]()
}

// ModesModeTagFromInt stands for Zig's `@bitCast` from the backing integer
// (ModeTag.Backing, a u16) to a ModeTag: the reverse of ModesModeTag.Int.
func ModesModeTagFromInt(v ModesModeTagBacking) ModesModeTag {
	return ModesModeTag{
		value: v & 0x7FFF,
		ansi:  v>>15 != 0,
	}
}

// ModesModeValues stands for Zig's `std.enums.values(Mode)`: every mode, in
// the order of `entries`. Each call returns a new slice.
func ModesModeValues() []ModesMode {
	values := make([]ModesMode, len(modesModeFields))
	for i, field := range modesModeFields {
		values[i] = ModesMode(field.value)
	}
	return values
}

// modesEntryForMode is modes.entryForMode.
//
// Zig: `mode` is a comptime parameter; an unknown mode is `unreachable`.
// `@setEvalBranchQuota(10_000)` has no Go counterpart.
func modesEntryForMode(mode ModesMode) modesModeEntry {
	return modesEntries[modesModeEntryIndex(mode)]
}

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
// It is not valuable to redocument them all here.
var modesEntries = []modesModeEntry{
	// ANSI
	{name: []uint8("disable_keyboard"), value: 2, ansi: true}, // KAM
	{name: []uint8("insert"), value: 4, ansi: true},
	{name: []uint8("send_receive_mode"), value: 12, ansi: true, default_: true}, // SRM
	{name: []uint8("linefeed"), value: 20, ansi: true},

	// DEC
	{name: []uint8("cursor_keys"), value: 1}, // DECCKM
	{name: []uint8("132_column"), value: 3},
	{name: []uint8("slow_scroll"), value: 4},
	{name: []uint8("reverse_colors"), value: 5},
	{name: []uint8("origin"), value: 6},
	{name: []uint8("wraparound"), value: 7, default_: true},
	{name: []uint8("autorepeat"), value: 8},
	{name: []uint8("mouse_event_x10"), value: 9},
	{name: []uint8("cursor_blinking"), value: 12},
	{name: []uint8("cursor_visible"), value: 25, default_: true},
	{name: []uint8("enable_mode_3"), value: 40},
	{name: []uint8("reverse_wrap"), value: 45},
	{name: []uint8("alt_screen_legacy"), value: 47},
	{name: []uint8("keypad_keys"), value: 66},
	{name: []uint8("enable_left_and_right_margin"), value: 69},
	{name: []uint8("mouse_event_normal"), value: 1000},
	{name: []uint8("mouse_event_button"), value: 1002},
	{name: []uint8("mouse_event_any"), value: 1003},
	{name: []uint8("focus_event"), value: 1004},
	{name: []uint8("mouse_format_utf8"), value: 1005},
	{name: []uint8("mouse_format_sgr"), value: 1006},
	{name: []uint8("mouse_alternate_scroll"), value: 1007, default_: true},
	{name: []uint8("mouse_format_urxvt"), value: 1015},
	{name: []uint8("mouse_format_sgr_pixels"), value: 1016},
	{name: []uint8("ignore_keypad_with_numlock"), value: 1035, default_: true},
	{name: []uint8("alt_esc_prefix"), value: 1036, default_: true},
	{name: []uint8("alt_sends_escape"), value: 1039},
	{name: []uint8("reverse_wrap_extended"), value: 1045},
	{name: []uint8("alt_screen"), value: 1047},
	{name: []uint8("save_cursor"), value: 1048},
	{name: []uint8("alt_screen_save_cursor_clear_enter"), value: 1049},
	{name: []uint8("bracketed_paste"), value: 2004},
	{name: []uint8("synchronized_output"), value: 2026},
	{name: []uint8("grapheme_cluster"), value: 2027},
	{name: []uint8("report_color_scheme"), value: 2031},
	{name: []uint8("in_band_size_reports"), value: 2048},
}
