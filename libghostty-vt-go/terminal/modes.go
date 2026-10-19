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
// ModePacked is one bit per entry, bit i for entry i, as Zig's packed struct
// is; `@field(packed, entry.name)` is that bit (modesModePackedFieldBit gives
// it for a name). Zig resolves the `inline else` + `comptime
// entryForMode(...)` + `@field` of set, get, save and restore at compile
// time, so at run time each is a single bit access. Following the rule for
// compile-time work (computed once at start-up, not on each call), the table
// modesModeIndex, built at start-up, maps a Mode to the index of its entry,
// which is its bit. set/get/save/restore, entryForMode, modeFromInt and
// String use it; modesEntryForMode keeps its Zig meaning.

package terminal

// ModesModeState is modes.ModeState: a struct that maintains the state of
// all the settable modes.
//
// Zig gives every field the default `.{}`, a ModePacked with every mode at
// its default: ModesModeStateInit (see the file header).
type ModesModeState struct {
	// The values of the current modes.
	Values ModesModePacked

	// The saved values. We only allow saving each mode once. This is in
	// line with other terminals that implement XTSAVE and XTRESTORE. We can
	// improve this in the future if it becomes a real-world issue but we
	// need to be aware of a DoS vector.
	Saved ModesModePacked

	// The default values for the modes. This is used to reset the modes to
	// their default values during reset.
	//
	// Zig: `default`, a Go keyword.
	Default ModesModePacked
}

// Reset is modes.ModeState.reset: reset the modes to their default values.
// This also clears the saved state.
func (self *ModesModeState) Reset() {
	self.Values = self.Default
	self.Saved = ModesModePackedInit()
}

// Set is modes.ModeState.set: set a mode to a value.
func (self *ModesModeState) Set(mode ModesMode, value bool) {
	// Zig: switch (mode) { inline else => |mode_comptime| ... }, resolved
	// at compile time; the tables are built at start-up.
	self.Values.set(modesModeEntryIndex(mode), value)
}

// Get is modes.ModeState.get: get the value of a mode.
//
// Zig takes `*const ModeState`; Go uses the pointer receiver of the other
// methods.
func (self *ModesModeState) Get(mode ModesMode) bool {
	// Zig: switch (mode) { inline else => |mode_comptime| ... }, resolved
	// at compile time; the tables are built at start-up.
	return self.Values.get(modesModeEntryIndex(mode))
}

// Save is modes.ModeState.save: save the state of the given mode. This can
// then be restored with restore. This will only be accurate if the previous
// mode was saved exactly once and not restored. Otherwise this will just
// keep restoring the last stored value in memory.
func (self *ModesModeState) Save(mode ModesMode) {
	// Zig: switch (mode) { inline else => |mode_comptime| ... }, resolved
	// at compile time; the tables are built at start-up.
	i := modesModeEntryIndex(mode)
	self.Saved.set(i, self.Values.get(i))
}

// Restore is modes.ModeState.restore: see save. This will return the
// restored value.
func (self *ModesModeState) Restore(mode ModesMode) bool {
	// Zig: switch (mode) { inline else => |mode_comptime| ... }, resolved
	// at compile time; the tables are built at start-up.
	i := modesModeEntryIndex(mode)
	self.Values.set(i, self.Saved.get(i))
	return self.Values.get(i)
}

// ModesModePacked is modes.ModePacked: a packed struct of all the settable
// modes. This shouldn't be used directly but rather through the ModeState
// struct.
//
// Zig: a packed struct generated from `entries`, one bool per entry, named
// after the entry, default the entry's `default`. In Go it is the same thing
// written by hand: one bit per entry, bit i for entry i of `entries` (the
// order of the Zig fields), read and written with get and set. The mode
// called `x` in Zig (`p.wraparound`) is bit modesModePackedFieldBit("x").
//
// Go: the first port had one bool field per entry and reached a field
// through a table of accessor functions; a mode check, several per printed
// character, was a table lookup and an indirect call
// (docs/post-port-optimizations.md item 35).
type ModesModePacked struct {
	bits uint64
}

// get is `@field(p, name)` read for the entry with index i.
func (p *ModesModePacked) get(i uint8) bool {
	return p.bits>>i&1 != 0
}

// set is `@field(p, name) = value` for the entry with index i.
func (p *ModesModePacked) set(i uint8, value bool) {
	if value {
		p.bits |= 1 << i
	} else {
		p.bits &^= 1 << i
	}
}

// modesModePackedFieldBit stands for the name in Zig's `@field(p, name)` on
// a ModePacked: the bit of the field named name (an entry name), which is the
// index of that entry. A name without a field is a compile error in Zig;
// here it panics.
func modesModePackedFieldBit(name []uint8) uint8 {
	for i, entry := range modesEntries {
		if string(entry.name) == string(name) {
			return uint8(i)
		}
	}
	panic("ModePacked has no field " + string(name))
}

var _ = modesModePackedCheck()

// modesModePackedCheck asserts that every entry has a bit in ModePacked. It
// runs once at start-up.
func modesModePackedCheck() bool {
	assert(len(modesEntries) <= 64)
	return true
}

// ModesModeStateInit stands for Zig's `ModeState{}`: every field takes its
// declared default, which for `values`, `saved` and `default` is
// `ModePacked{}` (see ModesModePackedInit). Zig has no named constructor;
// Go needs one because its zero value is all false, and six modes default to
// true.
func ModesModeStateInit() ModesModeState {
	return ModesModeState{
		Values:  ModesModePackedInit(),
		Saved:   ModesModePackedInit(),
		Default: ModesModePackedInit(),
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
	for i, entry := range modesEntries {
		p.set(uint8(i), entry.default_)
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
				Value: uint16(entry.value),
				Ansi:  entry.ansi,
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
func modesModeEntryIndex(mode ModesMode) uint8 {
	i := modesModeIndex[mode]
	if i == modesModeIndexNone {
		panic("unreachable")
	}
	return i
}

// ModesModeTag is modes.ModeTag: the tag type for our enum is a u16 but we
// use a packed struct in order to pack the ansi bit into the tag.
//
// Zig: packed struct(u16). In Go it is an ordinary struct; Int is the
// conversion Zig writes as @bitCast.
type ModesModeTag struct {
	// Zig: u15.
	Value uint16
	// Zig: default false.
	Ansi bool
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
	assert(self.Value <= 0x7FFF)
	var ansi ModesModeTagBacking = 0
	if self.Ansi {
		ansi = 1
	}
	return self.Value | ansi<<15
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
	tag := ModesModeTag{Ansi: ansi, Value: v}
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
		Value: v & 0x7FFF,
		Ansi:  v>>15 != 0,
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
