// Ported from src/terminal/parse_table.zig (Ghostty v1.3.1).

// The primary export of this file is "table", which contains a
// comptime-generated state transition table for VT emulation.
//
// This is based on the vt100.net state machine:
// https://vt100.net/emu/dec_ansi_parser
// But has some modifications:
//
//   - csi_param accepts the colon character (':') since the SGR command
//     accepts colon as a valid parameter value.

package terminal

// parse_table.zig names these State and Action through imports of
// Parser.zig; in Go they are ParserState and ParserTransitionAction.

// The state transition table. The type is [u8][State]Transition but
// comptime-generated to be exactly-sized.
//
// Zig computes this at compile time; Go computes it once at start-up.
var table = genTable()

// Table is a state transition table.
//
// Zig: genTableType(false). See docs/mapping.md: a function returning a
// type has no Go counterpart, so the two instantiations are written out.
type Table [256][parserStateCount]Transition

// OptionalTable is a table that has optional transitions.
//
// Zig: genTableType(true).
type OptionalTable [256][parserStateCount]Optional[Transition]

// Transition is a transition to a state and an action to perform.
type Transition struct {
	state  ParserState
	action ParserTransitionAction
}

// genTable is the function that generates the table.
func genTable() Table {
	var result OptionalTable
	for i := 0; i < len(result); i++ {
		for j := 0; j < len(result[0]); j++ {
			result[i][j] = Optional[Transition]{}
		}
	}

	// anywhere transitions
	for value := 0; value < parserStateCount; value++ {
		source := ParserState(value)

		// anywhere => ground
		single(&result, 0x18, source, ParserStateGround, ParserTransitionActionExecute)
		single(&result, 0x1A, source, ParserStateGround, ParserTransitionActionExecute)
		range_(&result, 0x80, 0x8F, source, ParserStateGround, ParserTransitionActionExecute)
		range_(&result, 0x91, 0x97, source, ParserStateGround, ParserTransitionActionExecute)
		single(&result, 0x99, source, ParserStateGround, ParserTransitionActionExecute)
		single(&result, 0x9A, source, ParserStateGround, ParserTransitionActionExecute)
		single(&result, 0x9C, source, ParserStateGround, ParserTransitionActionNone)

		// anywhere => escape
		single(&result, 0x1B, source, ParserStateEscape, ParserTransitionActionNone)

		// anywhere => sos_pm_apc_string
		single(&result, 0x98, source, ParserStateSosPmApcString, ParserTransitionActionNone)
		single(&result, 0x9E, source, ParserStateSosPmApcString, ParserTransitionActionNone)
		single(&result, 0x9F, source, ParserStateSosPmApcString, ParserTransitionActionNone)

		// anywhere => csi_entry
		single(&result, 0x9B, source, ParserStateCsiEntry, ParserTransitionActionNone)

		// anywhere => dcs_entry
		single(&result, 0x90, source, ParserStateDcsEntry, ParserTransitionActionNone)

		// anywhere => osc_string
		single(&result, 0x9D, source, ParserStateOscString, ParserTransitionActionNone)
	}

	// ground
	{
		// events
		single(&result, 0x19, ParserStateGround, ParserStateGround, ParserTransitionActionExecute)
		range_(&result, 0, 0x17, ParserStateGround, ParserStateGround, ParserTransitionActionExecute)
		range_(&result, 0x1C, 0x1F, ParserStateGround, ParserStateGround, ParserTransitionActionExecute)
		range_(&result, 0x20, 0x7F, ParserStateGround, ParserStateGround, ParserTransitionActionPrint)
	}

	// escape_intermediate
	{
		const source = ParserStateEscapeIntermediate

		single(&result, 0x19, source, source, ParserTransitionActionExecute)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionExecute)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionExecute)
		range_(&result, 0x20, 0x2F, source, source, ParserTransitionActionCollect)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)

		// => ground
		range_(&result, 0x30, 0x7E, source, ParserStateGround, ParserTransitionActionEscDispatch)
	}

	// sos_pm_apc_string
	{
		const source = ParserStateSosPmApcString

		// events
		single(&result, 0x19, source, source, ParserTransitionActionApcPut)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionApcPut)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionApcPut)
		range_(&result, 0x20, 0x7F, source, source, ParserTransitionActionApcPut)
	}

	// escape
	{
		const source = ParserStateEscape

		single(&result, 0x19, source, source, ParserTransitionActionExecute)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionExecute)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionExecute)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)

		// => ground
		range_(&result, 0x30, 0x4F, source, ParserStateGround, ParserTransitionActionEscDispatch)
		range_(&result, 0x51, 0x57, source, ParserStateGround, ParserTransitionActionEscDispatch)
		range_(&result, 0x60, 0x7E, source, ParserStateGround, ParserTransitionActionEscDispatch)
		single(&result, 0x59, source, ParserStateGround, ParserTransitionActionEscDispatch)
		single(&result, 0x5A, source, ParserStateGround, ParserTransitionActionEscDispatch)
		single(&result, 0x5C, source, ParserStateGround, ParserTransitionActionEscDispatch)

		// => escape_intermediate
		range_(&result, 0x20, 0x2F, source, ParserStateEscapeIntermediate, ParserTransitionActionCollect)

		// => sos_pm_apc_string
		single(&result, 0x58, source, ParserStateSosPmApcString, ParserTransitionActionNone)
		single(&result, 0x5E, source, ParserStateSosPmApcString, ParserTransitionActionNone)
		single(&result, 0x5F, source, ParserStateSosPmApcString, ParserTransitionActionNone)

		// => dcs_entry
		single(&result, 0x50, source, ParserStateDcsEntry, ParserTransitionActionNone)

		// => csi_entry
		single(&result, 0x5B, source, ParserStateCsiEntry, ParserTransitionActionNone)

		// => osc_string
		single(&result, 0x5D, source, ParserStateOscString, ParserTransitionActionNone)
	}

	// dcs_entry
	{
		const source = ParserStateDcsEntry

		// events
		single(&result, 0x19, source, source, ParserTransitionActionIgnore)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionIgnore)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionIgnore)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)

		// => dcs_intermediate
		range_(&result, 0x20, 0x2F, source, ParserStateDcsIntermediate, ParserTransitionActionCollect)

		// => dcs_ignore
		single(&result, 0x3A, source, ParserStateDcsIgnore, ParserTransitionActionNone)

		// => dcs_param
		range_(&result, 0x30, 0x39, source, ParserStateDcsParam, ParserTransitionActionParam)
		single(&result, 0x3B, source, ParserStateDcsParam, ParserTransitionActionParam)
		range_(&result, 0x3C, 0x3F, source, ParserStateDcsParam, ParserTransitionActionCollect)

		// => dcs_passthrough
		range_(&result, 0x40, 0x7E, source, ParserStateDcsPassthrough, ParserTransitionActionNone)
	}

	// dcs_intermediate
	{
		const source = ParserStateDcsIntermediate

		// events
		single(&result, 0x19, source, source, ParserTransitionActionIgnore)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionIgnore)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionIgnore)
		range_(&result, 0x20, 0x2F, source, source, ParserTransitionActionCollect)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)

		// => dcs_ignore
		range_(&result, 0x30, 0x3F, source, ParserStateDcsIgnore, ParserTransitionActionNone)

		// => dcs_passthrough
		range_(&result, 0x40, 0x7E, source, ParserStateDcsPassthrough, ParserTransitionActionNone)
	}

	// dcs_ignore
	{
		const source = ParserStateDcsIgnore

		// events
		single(&result, 0x19, source, source, ParserTransitionActionIgnore)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionIgnore)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionIgnore)
	}

	// dcs_param
	{
		const source = ParserStateDcsParam

		// events
		single(&result, 0x19, source, source, ParserTransitionActionIgnore)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionIgnore)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionIgnore)
		range_(&result, 0x30, 0x39, source, source, ParserTransitionActionParam)
		single(&result, 0x3B, source, source, ParserTransitionActionParam)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)

		// => dcs_ignore
		single(&result, 0x3A, source, ParserStateDcsIgnore, ParserTransitionActionNone)
		range_(&result, 0x3C, 0x3F, source, ParserStateDcsIgnore, ParserTransitionActionNone)

		// => dcs_intermediate
		range_(&result, 0x20, 0x2F, source, ParserStateDcsIntermediate, ParserTransitionActionCollect)

		// => dcs_passthrough
		range_(&result, 0x40, 0x7E, source, ParserStateDcsPassthrough, ParserTransitionActionNone)
	}

	// dcs_passthrough
	{
		const source = ParserStateDcsPassthrough

		// events
		single(&result, 0x19, source, source, ParserTransitionActionPut)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionPut)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionPut)
		range_(&result, 0x20, 0x7E, source, source, ParserTransitionActionPut)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)
	}

	// csi_param
	{
		const source = ParserStateCsiParam

		// events
		single(&result, 0x19, source, source, ParserTransitionActionExecute)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionExecute)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionExecute)
		range_(&result, 0x30, 0x39, source, source, ParserTransitionActionParam)
		single(&result, 0x3A, source, source, ParserTransitionActionParam)
		single(&result, 0x3B, source, source, ParserTransitionActionParam)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)

		// => ground
		range_(&result, 0x40, 0x7E, source, ParserStateGround, ParserTransitionActionCsiDispatch)

		// => csi_ignore
		range_(&result, 0x3C, 0x3F, source, ParserStateCsiIgnore, ParserTransitionActionNone)

		// => csi_intermediate
		range_(&result, 0x20, 0x2F, source, ParserStateCsiIntermediate, ParserTransitionActionCollect)
	}

	// csi_ignore
	{
		const source = ParserStateCsiIgnore

		// events
		single(&result, 0x19, source, source, ParserTransitionActionExecute)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionExecute)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionExecute)
		range_(&result, 0x20, 0x3F, source, source, ParserTransitionActionIgnore)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)

		// => ground
		range_(&result, 0x40, 0x7E, source, ParserStateGround, ParserTransitionActionNone)
	}

	// csi_intermediate
	{
		const source = ParserStateCsiIntermediate

		// events
		single(&result, 0x19, source, source, ParserTransitionActionExecute)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionExecute)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionExecute)
		range_(&result, 0x20, 0x2F, source, source, ParserTransitionActionCollect)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)

		// => ground
		range_(&result, 0x40, 0x7E, source, ParserStateGround, ParserTransitionActionCsiDispatch)

		// => csi_ignore
		range_(&result, 0x30, 0x3F, source, ParserStateCsiIgnore, ParserTransitionActionNone)
	}

	// csi_entry
	{
		const source = ParserStateCsiEntry

		// events
		single(&result, 0x19, source, source, ParserTransitionActionExecute)
		range_(&result, 0, 0x17, source, source, ParserTransitionActionExecute)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionExecute)
		single(&result, 0x7F, source, source, ParserTransitionActionIgnore)

		// => ground
		range_(&result, 0x40, 0x7E, source, ParserStateGround, ParserTransitionActionCsiDispatch)

		// => csi_ignore
		single(&result, 0x3A, source, ParserStateCsiIgnore, ParserTransitionActionNone)

		// => csi_intermediate
		range_(&result, 0x20, 0x2F, source, ParserStateCsiIntermediate, ParserTransitionActionCollect)

		// => csi_param
		range_(&result, 0x30, 0x39, source, ParserStateCsiParam, ParserTransitionActionParam)
		single(&result, 0x3B, source, ParserStateCsiParam, ParserTransitionActionParam)
		range_(&result, 0x3C, 0x3F, source, ParserStateCsiParam, ParserTransitionActionCollect)
	}

	// osc_string
	{
		const source = ParserStateOscString

		// events
		single(&result, 0x19, source, source, ParserTransitionActionIgnore)
		range_(&result, 0, 0x06, source, source, ParserTransitionActionIgnore)
		range_(&result, 0x08, 0x17, source, source, ParserTransitionActionIgnore)
		range_(&result, 0x1C, 0x1F, source, source, ParserTransitionActionIgnore)
		range_(&result, 0x20, 0xFF, source, source, ParserTransitionActionOscPut)

		// XTerm accepts either BEL  or ST  for terminating OSC
		// sequences, and when returning information, uses the same
		// terminator used in a query.
		single(&result, 0x07, source, ParserStateGround, ParserTransitionActionNone)
	}

	// Create our immutable version
	var final Table
	for i := 0; i < len(final); i++ {
		for j := 0; j < len(final[0]); j++ {
			if result[i][j].Valid {
				final[i][j] = result[i][j].Value
			} else {
				final[i][j] = transition(ParserState(j), ParserTransitionActionNone)
			}
		}
	}

	return final
}

func single(t *OptionalTable, c uint8, s0 ParserState, s1 ParserState, a ParserTransitionAction) {
	s0_int := int(s0)

	t[c][s0_int] = Some(transition(s1, a))
}

// range_ is Zig's "range"; "range" is a Go keyword.
func range_(t *OptionalTable, from uint8, to uint8, s0 ParserState, s1 ParserState, a ParserTransitionAction) {
	i := from
	for i <= to {
		single(t, i, s0, s1, a)
		// If 'to' is 0xFF, our next pass will overflow. Return early to prevent
		// the loop from executing it's continue expression
		if i == to {
			break
		}
		i += 1
	}
}

func transition(state ParserState, action ParserTransitionAction) Transition {
	return Transition{state: state, action: action}
}
