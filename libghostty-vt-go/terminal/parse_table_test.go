// Tests translated one for one from src/terminal/parse_table.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment.
//
// The helpers expect, expectEqual and zigUnwrap are in Parser_test.go; they
// stand for Zig's testing functions and for `x.?`.
//
// In parse_table.zig, State is Parser.State (Go: ParserState) and Action is
// Parser.TransitionAction (Go: ParserTransitionAction).
//
// Several Zig tests name their table `t`; that is the *testing.T here, so
// the table is `tbl`.

package terminal

import (
	"testing"
)

// Zig: test (unnamed, parse_table.zig:384)
func TestParseTable_unnamed(t *testing.T) {
	// This forces comptime-evaluation of table, so we're just testing
	// that it succeeds in creation.
	_ = table
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

func term51EmptyTable() OptionalTable {
	var t OptionalTable
	for i := 0; i < len(t); i++ {
		for j := 0; j < len(t[0]); j++ {
			t[i][j] = None[Transition]()
		}
	}
	return t
}

// Zig: test "term51: genTableType: one row per byte, one column per state"
func TestParseTable_term51_genTableType_one_row_per_byte_one_column_per_state(t *testing.T) {
	// Zig: const n_states = @typeInfo(State).@"enum".fields.len;
	const n_states = parserStateCount
	expectEqual(t, 14, n_states)
	var tbl Table
	// Zig: try term51_testing.expectEqual(256, @typeInfo(Table).array.len);
	expectEqual(t, 256, len(tbl))
	// Zig: try term51_testing.expectEqual(n_states, @typeInfo(@typeInfo(Table).array.child).array.len);
	expectEqual(t, n_states, len(tbl[0]))
	// Zig: try term51_testing.expect(@typeInfo(@typeInfo(Table).array.child).array.child == Transition);
	// A typed assignment that does not compile unless the element type is
	// Transition.
	var elem Transition = tbl[0][0]
	_ = elem
}

// Zig: test "term51: genTableType: optional flag selects the element type"
func TestParseTable_term51_genTableType_optional_flag_selects_the_element_type(t *testing.T) {
	var tbl Table
	var opt OptionalTable
	// Zig: try term51_testing.expect(genTableType(false) == Table);
	// genTableType has no Go counterpart (docs/mapping.md); the array type
	// it returns for `false` is written out here. The assignment does not
	// compile unless Table is that type.
	var not_optional [256][parserStateCount]Transition = tbl
	_ = not_optional
	// Zig: try term51_testing.expect(genTableType(true) == OptionalTable);
	var optional [256][parserStateCount]Optional[Transition] = opt
	_ = optional
	// Zig: try term51_testing.expect(@typeInfo(@typeInfo(OptionalTable).array.child).array.child == ?Transition);
	var elem Optional[Transition] = opt[0][0]
	_ = elem
	// Zig: try term51_testing.expectEqual(@typeInfo(Table).array.len, @typeInfo(OptionalTable).array.len);
	expectEqual(t, len(tbl), len(opt))
}

// Zig: test "term51: genTable: deterministic and equal to the published table"
func TestParseTable_term51_genTable_deterministic_and_equal_to_the_published_table(t *testing.T) {
	// Zig: const again = comptime genTable();
	again := genTable()
	for c := 0; c < 256; c++ {
		for s := 0; s < len(again[0]); s++ {
			expectEqual(t, table[c][s].state, again[c][s].state)
			expectEqual(t, table[c][s].action, again[c][s].action)
		}
	}
}

// Zig: test "term51: genTable: transitions that apply in every state"
func TestParseTable_term51_genTable_transitions_that_apply_in_every_state(t *testing.T) {
	// Zig: inline for (@typeInfo(State).@"enum".fields) |field|
	for s := 0; s < parserStateCount; s++ {
		// CAN and SUB abort to ground and execute.
		expectEqual(t, ParserStateGround, table[0x18][s].state)
		expectEqual(t, ParserTransitionActionExecute, table[0x18][s].action)
		expectEqual(t, ParserStateGround, table[0x1A][s].state)
		expectEqual(t, ParserTransitionActionExecute, table[0x1A][s].action)
		// ESC always starts a new escape sequence.
		expectEqual(t, ParserStateEscape, table[0x1B][s].state)
	}
}

// Zig: test "term51: genTable: ground"
func TestParseTable_term51_genTable_ground(t *testing.T) {
	g := int(ParserStateGround)
	// printable bytes, and DEL, print
	for c := 0x20; c <= 0x7F; c += 1 {
		expectEqual(t, ParserStateGround, table[c][g].state)
		expectEqual(t, ParserTransitionActionPrint, table[c][g].action)
	}
	for _, ctl := range []int{0x00, 0x07, 0x08, 0x09, 0x0A, 0x0D, 0x17, 0x19, 0x1C, 0x1F} {
		expectEqual(t, ParserStateGround, table[ctl][g].state)
		expectEqual(t, ParserTransitionActionExecute, table[ctl][g].action)
	}
}

// Zig: test "term51: genTable: escape"
func TestParseTable_term51_genTable_escape(t *testing.T) {
	e := int(ParserStateEscape)
	expectEqual(t, ParserStateCsiEntry, table['['][e].state)
	expectEqual(t, ParserStateOscString, table[']'][e].state)
	expectEqual(t, ParserStateDcsEntry, table['P'][e].state)
	expectEqual(t, ParserStateSosPmApcString, table['X'][e].state)
	expectEqual(t, ParserStateSosPmApcString, table['^'][e].state)
	expectEqual(t, ParserStateSosPmApcString, table['_'][e].state)
	// intermediates
	for c := 0x20; c <= 0x2F; c += 1 {
		expectEqual(t, ParserStateEscapeIntermediate, table[c][e].state)
		expectEqual(t, ParserTransitionActionCollect, table[c][e].action)
	}
	// finals
	for _, f := range []int{0x30, '7', '8', 'D', 'M', 'O', 'Q', 'W', 'Y', 'Z', '\\', '`', 'c', '~'} {
		expectEqual(t, ParserStateGround, table[f][e].state)
		expectEqual(t, ParserTransitionActionEscDispatch, table[f][e].action)
	}
	// DEL is ignored
	expectEqual(t, ParserStateEscape, table[0x7F][e].state)
	expectEqual(t, ParserTransitionActionIgnore, table[0x7F][e].action)
}

// Zig: test "term51: genTable: csi"
func TestParseTable_term51_genTable_csi(t *testing.T) {
	entry := int(ParserStateCsiEntry)
	param := int(ParserStateCsiParam)
	inter := int(ParserStateCsiIntermediate)
	ignore := int(ParserStateCsiIgnore)

	var c int = '0'
	for ; c <= '9'; c += 1 {
		expectEqual(t, ParserStateCsiParam, table[c][entry].state)
		expectEqual(t, ParserTransitionActionParam, table[c][entry].action)
		expectEqual(t, ParserStateCsiParam, table[c][param].state)
		expectEqual(t, ParserTransitionActionParam, table[c][param].action)
		// a digit after an intermediate invalidates the sequence
		expectEqual(t, ParserStateCsiIgnore, table[c][inter].state)
	}
	// ';' separates parameters from entry and from param
	expectEqual(t, ParserStateCsiParam, table[';'][entry].state)
	expectEqual(t, ParserTransitionActionParam, table[';'][entry].action)
	expectEqual(t, ParserStateCsiParam, table[';'][param].state)
	expectEqual(t, ParserTransitionActionParam, table[';'][param].action)
	// ':' is a separator only once a parameter has started; at entry and
	// after an intermediate it invalidates the sequence
	expectEqual(t, ParserStateCsiIgnore, table[':'][entry].state)
	expectEqual(t, ParserTransitionActionNone, table[':'][entry].action)
	expectEqual(t, ParserStateCsiParam, table[':'][param].state)
	expectEqual(t, ParserTransitionActionParam, table[':'][param].action)
	expectEqual(t, ParserStateCsiIgnore, table[':'][inter].state)
	// private markers are collected at entry and invalid afterwards
	c = 0x3C
	for ; c <= 0x3F; c += 1 {
		expectEqual(t, ParserStateCsiParam, table[c][entry].state)
		expectEqual(t, ParserTransitionActionCollect, table[c][entry].action)
		expectEqual(t, ParserStateCsiIgnore, table[c][param].state)
		expectEqual(t, ParserStateCsiIgnore, table[c][inter].state)
	}
	// intermediates
	c = 0x20
	for ; c <= 0x2F; c += 1 {
		expectEqual(t, ParserStateCsiIntermediate, table[c][entry].state)
		expectEqual(t, ParserStateCsiIntermediate, table[c][param].state)
		expectEqual(t, ParserStateCsiIntermediate, table[c][inter].state)
		expectEqual(t, ParserTransitionActionCollect, table[c][inter].action)
		expectEqual(t, ParserStateCsiIgnore, table[c][ignore].state)
	}
	// finals dispatch from entry, param and intermediate; ignore just ends
	c = 0x40
	for ; c <= 0x7E; c += 1 {
		expectEqual(t, ParserStateGround, table[c][entry].state)
		expectEqual(t, ParserTransitionActionCsiDispatch, table[c][entry].action)
		expectEqual(t, ParserStateGround, table[c][param].state)
		expectEqual(t, ParserTransitionActionCsiDispatch, table[c][param].action)
		expectEqual(t, ParserStateGround, table[c][inter].state)
		expectEqual(t, ParserTransitionActionCsiDispatch, table[c][inter].action)
		expectEqual(t, ParserStateGround, table[c][ignore].state)
		expectEqual(t, ParserTransitionActionNone, table[c][ignore].action)
	}
	// C0 controls execute inside a CSI sequence without leaving it
	for _, ctl := range []int{0x00, 0x07, 0x0A, 0x17, 0x19, 0x1C, 0x1F} {
		expectEqual(t, ParserStateCsiEntry, table[ctl][entry].state)
		expectEqual(t, ParserTransitionActionExecute, table[ctl][entry].action)
		expectEqual(t, ParserStateCsiParam, table[ctl][param].state)
		expectEqual(t, ParserTransitionActionExecute, table[ctl][param].action)
		expectEqual(t, ParserStateCsiIgnore, table[ctl][ignore].state)
		expectEqual(t, ParserTransitionActionExecute, table[ctl][ignore].action)
	}
}

// Zig: test "term51: genTable: dcs"
func TestParseTable_term51_genTable_dcs(t *testing.T) {
	entry := int(ParserStateDcsEntry)
	param := int(ParserStateDcsParam)
	inter := int(ParserStateDcsIntermediate)
	pass := int(ParserStateDcsPassthrough)
	ignore := int(ParserStateDcsIgnore)

	var c int = 0x40
	for ; c <= 0x7E; c += 1 {
		expectEqual(t, ParserStateDcsPassthrough, table[c][entry].state)
		expectEqual(t, ParserStateDcsPassthrough, table[c][param].state)
		expectEqual(t, ParserStateDcsPassthrough, table[c][inter].state)
		expectEqual(t, ParserStateDcsIgnore, table[c][ignore].state)
	}
	c = '0'
	for ; c <= '9'; c += 1 {
		expectEqual(t, ParserStateDcsParam, table[c][entry].state)
		expectEqual(t, ParserTransitionActionParam, table[c][entry].action)
		expectEqual(t, ParserStateDcsIgnore, table[c][inter].state)
	}
	// everything printable is passed through
	c = 0x20
	for ; c <= 0x7E; c += 1 {
		expectEqual(t, ParserStateDcsPassthrough, table[c][pass].state)
		expectEqual(t, ParserTransitionActionPut, table[c][pass].action)
	}
	// DEL is ignored in passthrough
	expectEqual(t, ParserTransitionActionIgnore, table[0x7F][pass].action)
	// C0 controls are passed through too
	expectEqual(t, ParserTransitionActionPut, table[0x0A][pass].action)
}

// Zig: test "term51: genTable: osc string"
func TestParseTable_term51_genTable_osc_string(t *testing.T) {
	o := int(ParserStateOscString)
	for c := 0x20; c <= 0xFF; c += 1 {
		expectEqual(t, ParserStateOscString, table[c][o].state)
		expectEqual(t, ParserTransitionActionOscPut, table[c][o].action)
	}
	// BEL terminates
	expectEqual(t, ParserStateGround, table[0x07][o].state)
	expectEqual(t, ParserTransitionActionNone, table[0x07][o].action)
	// other C0 controls are ignored
	for _, ctl := range []int{0x00, 0x06, 0x08, 0x0A, 0x17, 0x19, 0x1C, 0x1F} {
		expectEqual(t, ParserStateOscString, table[ctl][o].state)
		expectEqual(t, ParserTransitionActionIgnore, table[ctl][o].action)
	}
}

// Zig: test "term51: genTable: sos/pm/apc string"
func TestParseTable_term51_genTable_sos_pm_apc_string(t *testing.T) {
	s := int(ParserStateSosPmApcString)
	for c := 0x20; c <= 0x7F; c += 1 {
		expectEqual(t, ParserStateSosPmApcString, table[c][s].state)
		expectEqual(t, ParserTransitionActionApcPut, table[c][s].action)
	}
}

// Zig: test "term51: genTable: every cell is populated with a valid state"
func TestParseTable_term51_genTable_every_cell_is_populated_with_a_valid_state(t *testing.T) {
	// Zig: const n_states = @typeInfo(State).@"enum".fields.len;
	const n_states = parserStateCount
	for c := 0; c < 256; c++ {
		for s := 0; s < n_states; s++ {
			expect(t, int(table[c][s].state) < n_states, "@intFromEnum(table[c][s].state) < n_states")
		}
	}
}

// Zig: test "term51: single: sets exactly one cell"
func TestParseTable_term51_single_sets_exactly_one_cell(t *testing.T) {
	tbl := term51EmptyTable()
	single(&tbl, 0x41, ParserStateGround, ParserStateEscape, ParserTransitionActionPrint)
	cell := zigUnwrap(t, tbl[0x41][int(ParserStateGround)])
	expectEqual(t, ParserStateEscape, cell.state)
	expectEqual(t, ParserTransitionActionPrint, cell.action)
	set := 0
	for i := 0; i < len(tbl); i++ {
		for j := 0; j < len(tbl[0]); j++ {
			if tbl[i][j].Valid {
				set += 1
			}
		}
	}
	expectEqual(t, 1, set)
}

// Zig: test "term51: single: a later call overwrites an earlier one"
func TestParseTable_term51_single_a_later_call_overwrites_an_earlier_one(t *testing.T) {
	tbl := term51EmptyTable()
	single(&tbl, 0x00, ParserStateCsiEntry, ParserStateGround, ParserTransitionActionExecute)
	single(&tbl, 0x00, ParserStateCsiEntry, ParserStateCsiIgnore, ParserTransitionActionIgnore)
	cell := zigUnwrap(t, tbl[0x00][int(ParserStateCsiEntry)])
	expectEqual(t, ParserStateCsiIgnore, cell.state)
	expectEqual(t, ParserTransitionActionIgnore, cell.action)
}

// Zig: test "term51: single: first and last byte, first and last state"
func TestParseTable_term51_single_first_and_last_byte_first_and_last_state(t *testing.T) {
	tbl := term51EmptyTable()
	single(&tbl, 0x00, ParserStateGround, ParserStateGround, ParserTransitionActionNone)
	single(&tbl, 0xFF, ParserStateSosPmApcString, ParserStateGround, ParserTransitionActionApcPut)
	expect(t, tbl[0x00][int(ParserStateGround)].Valid, "t[0x00][@intFromEnum(State.ground)] != null")
	expect(t, tbl[0xFF][int(ParserStateSosPmApcString)].Valid, "t[0xFF][@intFromEnum(State.sos_pm_apc_string)] != null")
}

// Zig: test "term51: range: inclusive on both ends"
func TestParseTable_term51_range_inclusive_on_both_ends(t *testing.T) {
	tbl := term51EmptyTable()
	range_(&tbl, 0x30, 0x39, ParserStateCsiEntry, ParserStateCsiParam, ParserTransitionActionParam)
	s := int(ParserStateCsiEntry)
	expect(t, !tbl[0x2F][s].Valid, "t[0x2F][s] == null")
	for c := 0x30; c <= 0x39; c += 1 {
		expectEqual(t, ParserStateCsiParam, zigUnwrap(t, tbl[c][s]).state)
		expectEqual(t, ParserTransitionActionParam, zigUnwrap(t, tbl[c][s]).action)
	}
	expect(t, !tbl[0x3A][s].Valid, "t[0x3A][s] == null")
}

// Zig: test "term51: range: a single-element range"
func TestParseTable_term51_range_a_single_element_range(t *testing.T) {
	tbl := term51EmptyTable()
	range_(&tbl, 0x7F, 0x7F, ParserStateGround, ParserStateGround, ParserTransitionActionIgnore)
	s := int(ParserStateGround)
	expect(t, !tbl[0x7E][s].Valid, "t[0x7E][s] == null")
	expect(t, tbl[0x7F][s].Valid, "t[0x7F][s] != null")
	expect(t, !tbl[0x80][s].Valid, "t[0x80][s] == null")
}

// Zig: test "term51: range: ending at 0xFF does not overflow"
func TestParseTable_term51_range_ending_at_0xFF_does_not_overflow(t *testing.T) {
	tbl := term51EmptyTable()
	range_(&tbl, 0xFE, 0xFF, ParserStateOscString, ParserStateOscString, ParserTransitionActionOscPut)
	s := int(ParserStateOscString)
	expect(t, !tbl[0xFD][s].Valid, "t[0xFD][s] == null")
	expect(t, tbl[0xFE][s].Valid, "t[0xFE][s] != null")
	expect(t, tbl[0xFF][s].Valid, "t[0xFF][s] != null")
	// nothing wrapped around to the start
	expect(t, !tbl[0x00][s].Valid, "t[0x00][s] == null")
}

// Zig: test "term51: range: the full byte range"
func TestParseTable_term51_range_the_full_byte_range(t *testing.T) {
	tbl := term51EmptyTable()
	range_(&tbl, 0x00, 0xFF, ParserStateDcsIgnore, ParserStateDcsIgnore, ParserTransitionActionIgnore)
	s := int(ParserStateDcsIgnore)
	for c := 0; c < 256; c++ {
		expect(t, tbl[c][s].Valid, "t[c][s] != null")
	}
}

// Zig: test "term51: transition: carries state and action"
func TestParseTable_term51_transition_carries_state_and_action(t *testing.T) {
	tr := transition(ParserStateCsiParam, ParserTransitionActionCollect)
	expectEqual(t, ParserStateCsiParam, tr.state)
	expectEqual(t, ParserTransitionActionCollect, tr.action)
	tr2 := transition(ParserStateGround, ParserTransitionActionNone)
	expectEqual(t, ParserStateGround, tr2.state)
	expectEqual(t, ParserTransitionActionNone, tr2.action)
}
