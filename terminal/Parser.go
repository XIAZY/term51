// Ported from src/terminal/Parser.zig (Ghostty v1.3.1).

// VT-series parser for escape and control sequences.
//
// This is implemented directly as the state machine described on
// vt100.net: https://vt100.net/emu/dec_ansi_parser

package terminal

import (
	"fmt"
	"strings"
)

// ParserState is Parser.State: states for the state machine.
type ParserState uint8

const (
	ParserStateGround ParserState = iota
	ParserStateEscape
	ParserStateEscapeIntermediate
	ParserStateCsiEntry
	ParserStateCsiIntermediate
	ParserStateCsiParam
	ParserStateCsiIgnore
	ParserStateDcsEntry
	ParserStateDcsParam
	ParserStateDcsIntermediate
	ParserStateDcsPassthrough
	ParserStateDcsIgnore
	ParserStateOscString
	ParserStateSosPmApcString

	// parserStateCount is the number of states: Zig's
	// @typeInfo(State).@"enum".fields.len.
	parserStateCount = iota
)

var parserStateNames = [parserStateCount]string{
	"ground",
	"escape",
	"escape_intermediate",
	"csi_entry",
	"csi_intermediate",
	"csi_param",
	"csi_ignore",
	"dcs_entry",
	"dcs_param",
	"dcs_intermediate",
	"dcs_passthrough",
	"dcs_ignore",
	"osc_string",
	"sos_pm_apc_string",
}

// String returns the Zig tag name (@tagName).
func (s ParserState) String() string {
	return parserStateNames[s]
}

// ParserTransitionAction is Parser.TransitionAction: transition action is an
// action that can be taken during a state transition. This is more of an
// internal action, not one used by end users, typically.
type ParserTransitionAction uint8

const (
	ParserTransitionActionNone ParserTransitionAction = iota
	ParserTransitionActionIgnore
	ParserTransitionActionPrint
	ParserTransitionActionExecute
	ParserTransitionActionCollect
	ParserTransitionActionParam
	ParserTransitionActionEscDispatch
	ParserTransitionActionCsiDispatch
	ParserTransitionActionPut
	ParserTransitionActionOscPut
	ParserTransitionActionApcPut

	parserTransitionActionCount = iota
)

var parserTransitionActionNames = [parserTransitionActionCount]string{
	"none",
	"ignore",
	"print",
	"execute",
	"collect",
	"param",
	"esc_dispatch",
	"csi_dispatch",
	"put",
	"osc_put",
	"apc_put",
}

// String returns the Zig tag name (@tagName).
func (a ParserTransitionAction) String() string {
	return parserTransitionActionNames[a]
}

// ParserActionTag is Parser.Action.Tag.
type ParserActionTag uint8

const (
	// Draw character to the screen. This is a unicode codepoint.
	ParserActionPrint ParserActionTag = iota

	// Execute the C0 or C1 function.
	ParserActionExecute

	// Execute the CSI command. Note that pointers within this
	// structure are only valid until the next call to "next".
	ParserActionCsiDispatch

	// Execute the ESC command.
	ParserActionEscDispatch

	// Execute the OSC command.
	ParserActionOscDispatch

	// DCS-related events.
	ParserActionDcsHook
	ParserActionDcsPut
	ParserActionDcsUnhook

	// APC data
	ParserActionApcStart
	ParserActionApcPut
	ParserActionApcEnd

	parserActionTagCount = iota
)

var parserActionTagNames = [parserActionTagCount]string{
	"print",
	"execute",
	"csi_dispatch",
	"esc_dispatch",
	"osc_dispatch",
	"dcs_hook",
	"dcs_put",
	"dcs_unhook",
	"apc_start",
	"apc_put",
	"apc_end",
}

// String returns the Zig tag name (@tagName).
func (t ParserActionTag) String() string {
	return parserActionTagNames[t]
}

// ParserAction is Parser.Action: action is the action that a caller of the
// parser is expected to take as a result of some input character.
//
// Zig: a tagged union. In Go it is a struct holding the tag and one field
// per payload; only the field named by Tag is meaningful.
type ParserAction struct {
	Tag ParserActionTag

	// Zig: u21.
	Print       uint32
	Execute     uint8
	CsiDispatch ParserActionCSI
	EscDispatch ParserActionESC
	OscDispatch OSCCommand
	DcsHook     ParserActionDCS
	DcsPut      uint8
	ApcPut      uint8
}

// ParserActionCSI is Parser.Action.CSI.
type ParserActionCSI struct {
	Intermediates []uint8
	Params        []uint16
	ParamsSep     ParserActionCSISepList
	Final         uint8
}

// ParserActionCSISepList is Parser.Action.CSI.SepList: the list of
// separators used for CSI params. The value of the bit can be mapped to Sep.
// The index of this bit set specifies the separator AFTER that param. For
// example: 0;4:3 would have index 1 set.
type ParserActionCSISepList = StaticBitSet24

// ParserActionCSISep is Parser.Action.CSI.Sep: the separator used for CSI
// params.
type ParserActionCSISep uint8

const (
	ParserActionCSISepSemicolon ParserActionCSISep = 0
	ParserActionCSISepColon     ParserActionCSISep = 1
)

// Format implements formatter for logging.
//
// Zig: Action.CSI.format, writing "ESC [ {s} {any} {c}".
func (self ParserActionCSI) Format() string {
	return fmt.Sprintf("ESC [ %s %s %c", self.Intermediates, zigFmtAnySlice(self.Params), self.Final)
}

// ParserActionESC is Parser.Action.ESC.
type ParserActionESC struct {
	Intermediates []uint8
	Final         uint8
}

// Format implements formatter for logging.
//
// Zig: Action.ESC.format, writing "ESC {s} {c}".
func (self ParserActionESC) Format() string {
	return fmt.Sprintf("ESC %s %c", self.Intermediates, self.Final)
}

// ParserActionDCS is Parser.Action.DCS.
//
// Zig also declares DCS.C, the C ABI layout; the C bindings are not ported.
type ParserActionDCS struct {
	Intermediates []uint8
	Params        []uint16
	Final         uint8
}

// Format implements formatter for logging. This is mostly copied from the
// std.fmt implementation, but we modify it slightly so that we can
// print out custom formats for some of our primitives.
//
// Zig: Action.format. The output matches what Zig's std.fmt produces for
// this union so that log lines stay comparable.
func (self ParserAction) Format() string {
	var b strings.Builder
	b.WriteString("terminal.Parser.Action")
	b.WriteString("{ .")
	b.WriteString(self.Tag.String())
	b.WriteString(" = ")

	switch self.Tag {
	// Unicode
	case ParserActionPrint:
		fmt.Fprintf(&b, "'%s' (U+%X)", zigFmtUnicode(self.Print), self.Print)

	// Byte
	case ParserActionExecute:
		fmt.Fprintf(&b, "0x%x", self.Execute)
	case ParserActionDcsPut:
		fmt.Fprintf(&b, "0x%x", self.DcsPut)
	case ParserActionApcPut:
		fmt.Fprintf(&b, "0x%x", self.ApcPut)

	// All others do the default behavior
	case ParserActionCsiDispatch:
		v := self.CsiDispatch
		fmt.Fprintf(&b, ".{ .intermediates = %s, .params = %s, .params_sep = .{ .mask = %d }, .final = %d }",
			zigFmtAnySlice(v.Intermediates), zigFmtAnySlice(v.Params), v.ParamsSep.mask, v.Final)
	case ParserActionEscDispatch:
		v := self.EscDispatch
		fmt.Fprintf(&b, ".{ .intermediates = %s, .final = %d }", zigFmtAnySlice(v.Intermediates), v.Final)
	case ParserActionDcsHook:
		v := self.DcsHook
		fmt.Fprintf(&b, ".{ .intermediates = %s, .params = %s, .final = %d }",
			zigFmtAnySlice(v.Intermediates), zigFmtAnySlice(v.Params), v.Final)
	case ParserActionOscDispatch:
		b.WriteString(self.OscDispatch.zigFmtAny())
	case ParserActionDcsUnhook, ParserActionApcStart, ParserActionApcEnd:
		b.WriteString("void")
	}

	b.WriteString(" }")
	return b.String()
}

// Maximum number of intermediate characters during parsing. This is
// 4 because we also use the intermediates array for UTF8 decoding which
// can be at most 4 bytes.
const MAX_INTERMEDIATE = 4

// Maximum number of CSI parameters. This is arbitrary. Practically, the
// only CSI command that uses more than 3 parameters is the SGR command
// which can be infinitely long. 24 is a reasonable limit based on empirical
// data. This used to be 16 but Kakoune has a SGR command that uses 17
// parameters.
//
// We could in the future make this the static limit and then allocate after
// but that's a lot more work and practically its so rare to exceed this
// number. I implore TUI authors to not use more than this number of CSI
// params, but I suspect we'll introduce a slow path with heap allocation
// one day.
const MAX_PARAMS = 24

// Parser is the VT parser. Zig: the file-level struct of Parser.zig.
type Parser struct {
	// Current state of the state machine
	state ParserState

	// Intermediate tracking.
	intermediates     [MAX_INTERMEDIATE]uint8
	intermediates_idx uint8

	// Param tracking, building
	params        [MAX_PARAMS]uint16
	params_sep    ParserActionCSISepList
	params_idx    uint8
	param_acc     uint16
	param_acc_idx uint8

	// Parser for OSC sequences
	osc_parser OSCParser
}

// ParserInit is Parser.init.
func ParserInit() Parser {
	var result = Parser{
		state:             ParserStateGround,
		intermediates_idx: 0,
		params_sep:        StaticBitSet24InitEmpty(),
		params_idx:        0,
		param_acc:         0,
		param_acc_idx:     0,
		osc_parser:        OSCParserInit(nil),

		// Zig leaves intermediates and params undefined; Go zeroes them.
	}
	return result
}

// Deinit is Parser.deinit.
func (self *Parser) Deinit() {
	self.osc_parser.Deinit()
}

// Next consumes the next byte c and returns the actions to execute.
// Up to 3 actions may need to be executed -- in order -- representing
// the state exit, transition, and entry actions.
func (self *Parser) Next(c uint8) [3]Optional[ParserAction] {
	effect := table[c][int(self.state)]

	next_state := effect.state
	action := effect.action

	// After generating the actions, we set our next state.
	// Zig: `defer self.state = next_state`.
	defer func() { self.state = next_state }()

	// When going from one state to another, the actions take place in this order:
	//
	// 1. exit action from old state
	// 2. transition action
	// 3. entry action to new state
	var result [3]Optional[ParserAction]

	// Exit depends on current state
	if self.state == next_state {
		result[0] = None[ParserAction]()
	} else {
		switch self.state {
		case ParserStateOscString:
			if cmd := self.osc_parser.End(Some(c)); cmd != nil {
				result[0] = Some(ParserAction{Tag: ParserActionOscDispatch, OscDispatch: *cmd})
			} else {
				result[0] = None[ParserAction]()
			}
		case ParserStateDcsPassthrough:
			result[0] = Some(ParserAction{Tag: ParserActionDcsUnhook})
		case ParserStateSosPmApcString:
			result[0] = Some(ParserAction{Tag: ParserActionApcEnd})
		default:
			result[0] = None[ParserAction]()
		}
	}

	result[1] = self.doAction(action, c)

	// Entry depends on new state
	if self.state == next_state {
		result[2] = None[ParserAction]()
	} else {
		switch next_state {
		case ParserStateEscape, ParserStateDcsEntry, ParserStateCsiEntry:
			self.Clear()
			result[2] = None[ParserAction]()
		case ParserStateOscString:
			self.osc_parser.Reset()
			result[2] = None[ParserAction]()
		case ParserStateDcsPassthrough:
			// Ignore too many parameters
			if self.params_idx >= MAX_PARAMS {
				result[2] = None[ParserAction]()
				break
			}

			// Finalize parameters
			if self.param_acc_idx > 0 {
				self.params[self.params_idx] = self.param_acc
				self.params_idx += 1
			}
			result[2] = Some(ParserAction{
				Tag: ParserActionDcsHook,
				DcsHook: ParserActionDCS{
					Intermediates: self.intermediates[0:self.intermediates_idx],
					Params:        self.params[0:self.params_idx],
					Final:         c,
				},
			})
		case ParserStateSosPmApcString:
			result[2] = Some(ParserAction{Tag: ParserActionApcStart})
		default:
			result[2] = None[ParserAction]()
		}
	}

	return result
}

// Collect is Parser.collect.
func (self *Parser) Collect(c uint8) {
	if self.intermediates_idx >= MAX_INTERMEDIATE {
		warn("parser", "invalid intermediates count")
		return
	}

	self.intermediates[self.intermediates_idx] = c
	self.intermediates_idx += 1
}

func (self *Parser) doAction(action ParserTransitionAction, c uint8) Optional[ParserAction] {
	switch action {
	case ParserTransitionActionNone, ParserTransitionActionIgnore:
		return None[ParserAction]()
	case ParserTransitionActionPrint:
		return Some(ParserAction{Tag: ParserActionPrint, Print: uint32(c)})
	case ParserTransitionActionExecute:
		return Some(ParserAction{Tag: ParserActionExecute, Execute: c})
	case ParserTransitionActionCollect:
		self.Collect(c)
		return None[ParserAction]()
	case ParserTransitionActionParam:
		// Semicolon separates parameters. If we encounter a semicolon
		// we need to store and move on to the next parameter.
		if c == ';' || c == ':' {
			// Ignore too many parameters
			if self.params_idx >= MAX_PARAMS {
				return None[ParserAction]()
			}

			// Set param final value
			self.params[self.params_idx] = self.param_acc
			if c == ':' {
				self.params_sep.set(int(self.params_idx))
			}
			self.params_idx += 1

			// Reset current param value to 0
			self.param_acc = 0
			self.param_acc_idx = 0
			return None[ParserAction]()
		}

		// A numeric value. Add it to our accumulator.
		// Zig: `*|=` and `+|=`, saturating arithmetic on a u16.
		self.param_acc = saturatingMulU16(self.param_acc, 10)
		self.param_acc = saturatingAddU16(self.param_acc, uint16(c-'0'))

		// Increment our accumulator index. If we overflow then
		// we're out of bounds and we exit immediately.
		//
		// Zig: @addWithOverflow on a u8. The counter wraps either way; the
		// overflow flag only chooses between two branches that both return
		// null. Go's uint8 addition wraps identically.
		self.param_acc_idx += 1

		// The client is expected to perform no action.
		return None[ParserAction]()
	case ParserTransitionActionOscPut:
		self.osc_parser.Next(c)
		return None[ParserAction]()
	case ParserTransitionActionCsiDispatch:
		// Ignore too many parameters
		if self.params_idx >= MAX_PARAMS {
			return None[ParserAction]()
		}

		// Finalize parameters if we have one
		if self.param_acc_idx > 0 {
			self.params[self.params_idx] = self.param_acc
			self.params_idx += 1
		}

		result := ParserAction{
			Tag: ParserActionCsiDispatch,
			CsiDispatch: ParserActionCSI{
				Intermediates: self.intermediates[0:self.intermediates_idx],
				Params:        self.params[0:self.params_idx],
				ParamsSep:     self.params_sep,
				Final:         c,
			},
		}

		// We only allow colon or mixed separators for the 'm' command.
		if c != 'm' && self.params_sep.count() > 0 {
			warn(
				"parser",
				"CSI colon or mixed separators only allowed for 'm' command, got: %s",
				result.Format(),
			)
			return None[ParserAction]()
		}

		return Some(result)
	case ParserTransitionActionEscDispatch:
		return Some(ParserAction{
			Tag: ParserActionEscDispatch,
			EscDispatch: ParserActionESC{
				Intermediates: self.intermediates[0:self.intermediates_idx],
				Final:         c,
			},
		})
	case ParserTransitionActionPut:
		return Some(ParserAction{Tag: ParserActionDcsPut, DcsPut: c})
	case ParserTransitionActionApcPut:
		return Some(ParserAction{Tag: ParserActionApcPut, ApcPut: c})
	}
	panic("unreachable")
}

// Clear is Parser.clear.
func (self *Parser) Clear() {
	self.intermediates_idx = 0
	self.params_idx = 0
	self.params_sep = StaticBitSet24InitEmpty()
	self.param_acc = 0
	self.param_acc_idx = 0
}
