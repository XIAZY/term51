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
		osc_parser:        OSCParserInit(false),

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
	var result [3]Optional[ParserAction]
	self.nextInto(c, &result)
	return result
}

// nextInto is the body of Parser.next, writing the up to three actions into
// the caller's slots instead of returning them.
//
// Go: Zig returns `[3]?Action`, a few small tagged unions. The Go
// ParserAction is a struct with a field per payload (840 bytes, because one
// field is a whole OSC command), so returning three of them for every input
// byte meant clearing and copying about 2.5 KB per byte; that was most of the
// stream's time (docs/post-port-optimizations.md, item 1). The stream keeps
// one array for the life of the stream and passes it here. Each slot is
// written through parserActionNone / parserActionSome, which keep a slot
// exactly equal to the value Next would have returned: only the payload field
// of the slot's tag is ever non-zero.
func (self *Parser) nextInto(c uint8, result *[3]Optional[ParserAction]) {
	effect := table[c][int(self.state)]

	next_state := effect.state
	action := effect.action

	// After generating the actions, we set our next state.
	// Zig: `defer self.state = next_state`. The function has one exit, so
	// the assignment is written there instead of in a deferred closure,
	// which Go would run (and pay for) on every byte.

	// When going from one state to another, the actions take place in this order:
	//
	// 1. exit action from old state
	// 2. transition action
	// 3. entry action to new state
	// Go: the three slots are the caller's (see nextInto's comment).

	// Exit depends on current state
	if self.state == next_state {
		parserActionNone(&result[0])
	} else {
		switch self.state {
		case ParserStateOscString:
			if cmd := self.osc_parser.End(Some(c)); cmd != nil {
				parserActionSome(&result[0], ParserActionOscDispatch).OscDispatch = *cmd
			} else {
				parserActionNone(&result[0])
			}
		case ParserStateDcsPassthrough:
			parserActionSome(&result[0], ParserActionDcsUnhook)
		case ParserStateSosPmApcString:
			parserActionSome(&result[0], ParserActionApcEnd)
		default:
			parserActionNone(&result[0])
		}
	}

	self.doActionInto(action, c, &result[1])

	// Entry depends on new state
	if self.state == next_state {
		parserActionNone(&result[2])
	} else {
		switch next_state {
		case ParserStateEscape, ParserStateDcsEntry, ParserStateCsiEntry:
			self.Clear()
			parserActionNone(&result[2])
		case ParserStateOscString:
			self.osc_parser.Reset()
			parserActionNone(&result[2])
		case ParserStateDcsPassthrough:
			// Ignore too many parameters
			if self.params_idx >= MAX_PARAMS {
				parserActionNone(&result[2])
				break
			}

			// Finalize parameters
			if self.param_acc_idx > 0 {
				self.params[self.params_idx] = self.param_acc
				self.params_idx += 1
			}
			parserActionSome(&result[2], ParserActionDcsHook).DcsHook = ParserActionDCS{
				Intermediates: self.intermediates[0:self.intermediates_idx],
				Params:        self.params[0:self.params_idx],
				Final:         c,
			}
		case ParserStateSosPmApcString:
			parserActionSome(&result[2], ParserActionApcStart)
		default:
			parserActionNone(&result[2])
		}
	}

	self.state = next_state
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
	var out Optional[ParserAction]
	self.doActionInto(action, c, &out)
	return out
}

// doActionInto is the body of Parser.doAction, writing its result into the
// caller's slot (see nextInto).
func (self *Parser) doActionInto(action ParserTransitionAction, c uint8, out *Optional[ParserAction]) {
	switch action {
	case ParserTransitionActionNone, ParserTransitionActionIgnore:
		parserActionNone(out)
		return
	case ParserTransitionActionPrint:
		parserActionSome(out, ParserActionPrint).Print = uint32(c)
		return
	case ParserTransitionActionExecute:
		parserActionSome(out, ParserActionExecute).Execute = c
		return
	case ParserTransitionActionCollect:
		self.Collect(c)
		parserActionNone(out)
		return
	case ParserTransitionActionParam:
		// Semicolon separates parameters. If we encounter a semicolon
		// we need to store and move on to the next parameter.
		if c == ';' || c == ':' {
			// Ignore too many parameters
			if self.params_idx >= MAX_PARAMS {
				parserActionNone(out)
				return
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
			parserActionNone(out)
			return
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
		parserActionNone(out)
		return
	case ParserTransitionActionOscPut:
		self.osc_parser.Next(c)
		parserActionNone(out)
		return
	case ParserTransitionActionCsiDispatch:
		// Ignore too many parameters
		if self.params_idx >= MAX_PARAMS {
			parserActionNone(out)
			return
		}

		// Finalize parameters if we have one
		if self.param_acc_idx > 0 {
			self.params[self.params_idx] = self.param_acc
			self.params_idx += 1
		}

		// Go: built in the caller's slot; `result` is Zig's local.
		result := parserActionSome(out, ParserActionCsiDispatch)
		result.CsiDispatch = ParserActionCSI{
			Intermediates: self.intermediates[0:self.intermediates_idx],
			Params:        self.params[0:self.params_idx],
			ParamsSep:     self.params_sep,
			Final:         c,
		}

		// We only allow colon or mixed separators for the 'm' command.
		if c != 'm' && self.params_sep.count() > 0 {
			warn(
				"parser",
				"CSI colon or mixed separators only allowed for 'm' command, got: %s",
				result.Format(),
			)
			parserActionNone(out)
			return
		}

		return
	case ParserTransitionActionEscDispatch:
		parserActionSome(out, ParserActionEscDispatch).EscDispatch = ParserActionESC{
			Intermediates: self.intermediates[0:self.intermediates_idx],
			Final:         c,
		}
		return
	case ParserTransitionActionPut:
		parserActionSome(out, ParserActionDcsPut).DcsPut = c
		return
	case ParserTransitionActionApcPut:
		parserActionSome(out, ParserActionApcPut).ApcPut = c
		return
	}
	panic("unreachable")
}

// parserActionNone makes a slot Zig's `null`: afterwards it equals
// None[ParserAction](). Not a Zig declaration (see nextInto).
func parserActionNone(o *Optional[ParserAction]) {
	if !o.Valid {
		return
	}
	// Only the payload of the slot's tag can be non-zero.
	switch o.Value.Tag {
	case ParserActionPrint:
		o.Value.Print = 0
	case ParserActionExecute:
		o.Value.Execute = 0
	case ParserActionCsiDispatch:
		o.Value.CsiDispatch = ParserActionCSI{}
	case ParserActionEscDispatch:
		o.Value.EscDispatch = ParserActionESC{}
	case ParserActionOscDispatch:
		o.Value.OscDispatch = OSCCommand{}
	case ParserActionDcsHook:
		o.Value.DcsHook = ParserActionDCS{}
	case ParserActionDcsPut:
		o.Value.DcsPut = 0
	case ParserActionApcPut:
		o.Value.ApcPut = 0
	case ParserActionDcsUnhook, ParserActionApcStart, ParserActionApcEnd:
		// No payload.
	default:
		panic("unreachable")
	}
	o.Value.Tag = 0
	o.Valid = false
}

// parserActionSome makes a slot Zig's `Action{ .tag = ... }` with a zero
// payload and returns the action for the caller to set its payload field:
// afterwards the slot equals Some(ParserAction{Tag: tag, <payload>}). Not a
// Zig declaration (see nextInto).
func parserActionSome(o *Optional[ParserAction], tag ParserActionTag) *ParserAction {
	parserActionNone(o)
	o.Valid = true
	o.Value.Tag = tag
	return &o.Value
}

// Clear is Parser.clear.
func (self *Parser) Clear() {
	self.intermediates_idx = 0
	self.params_idx = 0
	self.params_sep = StaticBitSet24InitEmpty()
	self.param_acc = 0
	self.param_acc_idx = 0
}
