// Ported from src/terminal/osc/parsers/context_signal.zig (Ghostty v1.3.1).
//
// OSC 3008: Hierarchical Context Signalling (UAPI spec)
// Specification: https://uapi-group.org/specifications/specs/osc_context/
//
// OSC 3008 allows programs to signal context changes to the terminal
// emulator. Each context has an identifier and metadata fields. Contexts are
// hierarchical and form a stack.
//
// File naming: `osc/parsers/context_signal.zig` becomes
// `osc_parsers_context_signal.go` in package terminal, and its declarations
// take the prefix `OSCParsersContextSignal` (`oscParsersContextSignal` for
// private ones). The file has no C ABI declarations.
//
// Not ported:
//   - `Field.Type`, a comptime function that returns a type. Go has no such
//     thing; its counterpart is the struct OSCParsersContextSignalFieldValue,
//     which holds one result field per Field value.
//   - `const log = std.log.scoped(.osc_context_signal)`: the port uses
//     warn("osc_context_signal", ...).

package terminal

import (
	"bytes"
	"strconv"
)

// oscParsersContextSignalMaxContextIdLen is
// osc.parsers.context_signal.max_context_id_len: maximum length of a context
// identifier (per spec).
const oscParsersContextSignalMaxContextIdLen = 64

// OSCParsersContextSignalCommand is osc.parsers.context_signal.Command: a
// single OSC 3008 context signal command.
type OSCParsersContextSignalCommand struct {
	action OSCParsersContextSignalCommandAction
	// The context identifier. Must be 1-64 characters in the 32..126 byte
	// range.
	id []uint8
	// Raw unparsed metadata fields after the context ID.
	// Fields are semicolon-separated key=value pairs.
	// Parsed lazily via `readOption`.
	metadata []uint8
}

// OSCParsersContextSignalCommandAction is
// osc.parsers.context_signal.Command.Action.
type OSCParsersContextSignalCommandAction uint8

const (
	// OSC 3008;start=<id> — initiates, updates, or returns to a context.
	OSCParsersContextSignalCommandActionStart OSCParsersContextSignalCommandAction = iota
	// OSC 3008;end=<id> — terminates a context.
	OSCParsersContextSignalCommandActionEnd
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersContextSignalCommandAction) String() string {
	return oscParsersContextSignalCommandActionNames[self]
}

var oscParsersContextSignalCommandActionNames = [...]string{
	OSCParsersContextSignalCommandActionStart: "start",
	OSCParsersContextSignalCommandActionEnd:   "end",
}

// ReadOption is osc.parsers.context_signal.Command.readOption: read a
// metadata field value from the raw fields string. Returns null if the field
// is not present or malformed.
//
// Zig: `option` is a comptime parameter and the result type is
// `option.Type()`. Go: the result is in the field of the returned struct
// that is named after `option`; see OSCParsersContextSignalFieldValue.
func (self OSCParsersContextSignalCommand) ReadOption(
	option OSCParsersContextSignalField,
) OSCParsersContextSignalFieldValue {
	return option.Read(self.metadata)
}

// OSCParsersContextSignalContextType is
// osc.parsers.context_signal.ContextType: context types defined by the
// specification.
type OSCParsersContextSignalContextType uint8

const (
	OSCParsersContextSignalContextTypeBoot OSCParsersContextSignalContextType = iota
	OSCParsersContextSignalContextTypeContainer
	OSCParsersContextSignalContextTypeVm
	OSCParsersContextSignalContextTypeElevate
	OSCParsersContextSignalContextTypeChpriv
	OSCParsersContextSignalContextTypeSubcontext
	OSCParsersContextSignalContextTypeRemote
	OSCParsersContextSignalContextTypeShell
	OSCParsersContextSignalContextTypeCommand
	OSCParsersContextSignalContextTypeApp
	OSCParsersContextSignalContextTypeService
	OSCParsersContextSignalContextTypeSession

	oscParsersContextSignalContextTypeCount = iota
)

// String returns the Zig tag name (@tagName), the name that
// std.meta.stringToEnum matches in ContextType.parse.
func (self OSCParsersContextSignalContextType) String() string {
	return oscParsersContextSignalContextTypeNames[self]
}

var oscParsersContextSignalContextTypeNames = [oscParsersContextSignalContextTypeCount]string{
	OSCParsersContextSignalContextTypeBoot:       "boot",
	OSCParsersContextSignalContextTypeContainer:  "container",
	OSCParsersContextSignalContextTypeVm:         "vm",
	OSCParsersContextSignalContextTypeElevate:    "elevate",
	OSCParsersContextSignalContextTypeChpriv:     "chpriv",
	OSCParsersContextSignalContextTypeSubcontext: "subcontext",
	OSCParsersContextSignalContextTypeRemote:     "remote",
	OSCParsersContextSignalContextTypeShell:      "shell",
	OSCParsersContextSignalContextTypeCommand:    "command",
	OSCParsersContextSignalContextTypeApp:        "app",
	OSCParsersContextSignalContextTypeService:    "service",
	OSCParsersContextSignalContextTypeSession:    "session",
}

// OSCParsersContextSignalContextTypeParse is
// osc.parsers.context_signal.ContextType.parse:
// std.meta.stringToEnum(ContextType, value).
func OSCParsersContextSignalContextTypeParse(value []uint8) Optional[OSCParsersContextSignalContextType] {
	// std.meta.stringToEnum: the value whose tag name equals value exactly.
	for i := OSCParsersContextSignalContextType(0); i < oscParsersContextSignalContextTypeCount; i++ {
		if string(value) == i.String() {
			return Some(i)
		}
	}
	return None[OSCParsersContextSignalContextType]()
}

// OSCParsersContextSignalExitStatus is
// osc.parsers.context_signal.ExitStatus: exit status for the `exit`
// end-sequence field.
type OSCParsersContextSignalExitStatus uint8

const (
	OSCParsersContextSignalExitStatusSuccess OSCParsersContextSignalExitStatus = iota
	OSCParsersContextSignalExitStatusFailure
	OSCParsersContextSignalExitStatusCrash
	OSCParsersContextSignalExitStatusInterrupt

	oscParsersContextSignalExitStatusCount = iota
)

// String returns the Zig tag name (@tagName), the name that
// std.meta.stringToEnum matches in ExitStatus.parse.
func (self OSCParsersContextSignalExitStatus) String() string {
	return oscParsersContextSignalExitStatusNames[self]
}

var oscParsersContextSignalExitStatusNames = [oscParsersContextSignalExitStatusCount]string{
	OSCParsersContextSignalExitStatusSuccess:   "success",
	OSCParsersContextSignalExitStatusFailure:   "failure",
	OSCParsersContextSignalExitStatusCrash:     "crash",
	OSCParsersContextSignalExitStatusInterrupt: "interrupt",
}

// OSCParsersContextSignalExitStatusParse is
// osc.parsers.context_signal.ExitStatus.parse:
// std.meta.stringToEnum(ExitStatus, value).
func OSCParsersContextSignalExitStatusParse(value []uint8) Optional[OSCParsersContextSignalExitStatus] {
	// std.meta.stringToEnum: the value whose tag name equals value exactly.
	for i := OSCParsersContextSignalExitStatus(0); i < oscParsersContextSignalExitStatusCount; i++ {
		if string(value) == i.String() {
			return Some(i)
		}
	}
	return None[OSCParsersContextSignalExitStatus]()
}

// OSCParsersContextSignalField is osc.parsers.context_signal.Field: metadata
// fields that can appear in OSC 3008 sequences. Fields are read lazily from
// the raw string using the `read` method.
type OSCParsersContextSignalField uint8

const (
	// Start sequence fields
	OSCParsersContextSignalFieldType OSCParsersContextSignalField = iota
	OSCParsersContextSignalFieldUser
	OSCParsersContextSignalFieldHostname
	OSCParsersContextSignalFieldMachineid
	OSCParsersContextSignalFieldBootid
	OSCParsersContextSignalFieldPid
	OSCParsersContextSignalFieldPidfdid
	OSCParsersContextSignalFieldComm
	OSCParsersContextSignalFieldCwd
	OSCParsersContextSignalFieldCmdline
	OSCParsersContextSignalFieldVm
	OSCParsersContextSignalFieldContainer
	OSCParsersContextSignalFieldTargetuser
	OSCParsersContextSignalFieldTargethost
	OSCParsersContextSignalFieldSessionid

	// End sequence fields
	OSCParsersContextSignalFieldExit
	OSCParsersContextSignalFieldStatus
	OSCParsersContextSignalFieldSignal

	oscParsersContextSignalFieldCount = iota
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersContextSignalField) String() string {
	return oscParsersContextSignalFieldNames[self]
}

var oscParsersContextSignalFieldNames = [oscParsersContextSignalFieldCount]string{
	OSCParsersContextSignalFieldType:       "type",
	OSCParsersContextSignalFieldUser:       "user",
	OSCParsersContextSignalFieldHostname:   "hostname",
	OSCParsersContextSignalFieldMachineid:  "machineid",
	OSCParsersContextSignalFieldBootid:     "bootid",
	OSCParsersContextSignalFieldPid:        "pid",
	OSCParsersContextSignalFieldPidfdid:    "pidfdid",
	OSCParsersContextSignalFieldComm:       "comm",
	OSCParsersContextSignalFieldCwd:        "cwd",
	OSCParsersContextSignalFieldCmdline:    "cmdline",
	OSCParsersContextSignalFieldVm:         "vm",
	OSCParsersContextSignalFieldContainer:  "container",
	OSCParsersContextSignalFieldTargetuser: "targetuser",
	OSCParsersContextSignalFieldTargethost: "targethost",
	OSCParsersContextSignalFieldSessionid:  "sessionid",
	OSCParsersContextSignalFieldExit:       "exit",
	OSCParsersContextSignalFieldStatus:     "status",
	OSCParsersContextSignalFieldSignal:     "signal",
}

// OSCParsersContextSignalFieldValue is the Go counterpart of
// osc.parsers.context_signal.Field.Type, the comptime function that gives
// the result type of `read` and `readOption` for each field:
//
//	.type                    ?ContextType
//	.exit                    ?ExitStatus
//	.pid, .pidfdid, .status  ?u64
//	every other field        ?[]const u8
//
// Go cannot make a result type depend on an argument value, so `Read` and
// `ReadOption` return this struct. It works like a tagged union whose tag is
// the Field that was asked for: only the field named after that Field is
// meaningful, and it is the null Optional where Zig returns null.
type OSCParsersContextSignalFieldValue struct {
	Type       Optional[OSCParsersContextSignalContextType]
	User       Optional[[]uint8]
	Hostname   Optional[[]uint8]
	Machineid  Optional[[]uint8]
	Bootid     Optional[[]uint8]
	Pid        Optional[uint64]
	Pidfdid    Optional[uint64]
	Comm       Optional[[]uint8]
	Cwd        Optional[[]uint8]
	Cmdline    Optional[[]uint8]
	Vm         Optional[[]uint8]
	Container  Optional[[]uint8]
	Targetuser Optional[[]uint8]
	Targethost Optional[[]uint8]
	Sessionid  Optional[[]uint8]

	Exit   Optional[OSCParsersContextSignalExitStatus]
	Status Optional[uint64]
	Signal Optional[[]uint8]
}

// key is osc.parsers.context_signal.Field.key: @tagName(self).
func (self OSCParsersContextSignalField) key() []uint8 {
	return []uint8(self.String())
}

// Read is osc.parsers.context_signal.Field.read: read the field value from
// the raw fields string.
//
// The raw fields string contains semicolon-separated key=value pairs
// e.g. "type=container;user=lennart;hostname=zeta".
//
// Unknown or malformed fields are ignored per the specification.
//
// Zig: `self` is a comptime parameter and the result type is `self.Type()`.
// Go: the result is in the field of the returned struct that is named after
// `self`; see OSCParsersContextSignalFieldValue.
func (self OSCParsersContextSignalField) Read(raw []uint8) OSCParsersContextSignalFieldValue {
	var result OSCParsersContextSignalFieldValue

	// Zig: std.mem.splitScalar(u8, raw, ';'). bytes.Split with a one-byte
	// separator yields the same pieces, empty ones included.
	for _, full := range bytes.Split(raw, []uint8{';'}) {
		// Parse key=value
		eql_idx := bytes.IndexByte(full, '=')
		if eql_idx < 0 || !bytes.Equal(full[:eql_idx], self.key()) {
			continue
		}
		value := full[eql_idx+1:]

		switch self {
		case OSCParsersContextSignalFieldType:
			result.Type = OSCParsersContextSignalContextTypeParse(value)
		case OSCParsersContextSignalFieldExit:
			result.Exit = OSCParsersContextSignalExitStatusParse(value)
		case OSCParsersContextSignalFieldPid,
			OSCParsersContextSignalFieldPidfdid,
			OSCParsersContextSignalFieldStatus:
			// Zig: a labeled block that breaks with null on the first byte
			// that is not a decimal digit, else with
			// std.fmt.parseInt(u64, value, 10) catch null. After the digit
			// check the input has no sign, no '_' and no base prefix, so
			// strconv.ParseUint(value, 10, 64) accepts and rejects exactly
			// what Zig's parseInt does: it fails on the empty string and on
			// overflow.
			number := None[uint64]()
			digits := true
			for _, c := range value {
				if c < '0' || c > '9' {
					digits = false
					break
				}
			}
			if digits {
				if v, err := strconv.ParseUint(string(value), 10, 64); err == nil {
					number = Some(v)
				}
			}
			switch self {
			case OSCParsersContextSignalFieldPid:
				result.Pid = number
			case OSCParsersContextSignalFieldPidfdid:
				result.Pidfdid = number
			case OSCParsersContextSignalFieldStatus:
				result.Status = number
			}
		// String fields
		case OSCParsersContextSignalFieldUser,
			OSCParsersContextSignalFieldHostname,
			OSCParsersContextSignalFieldMachineid,
			OSCParsersContextSignalFieldBootid,
			OSCParsersContextSignalFieldComm,
			OSCParsersContextSignalFieldCwd,
			OSCParsersContextSignalFieldCmdline,
			OSCParsersContextSignalFieldVm,
			OSCParsersContextSignalFieldContainer,
			OSCParsersContextSignalFieldTargetuser,
			OSCParsersContextSignalFieldTargethost,
			OSCParsersContextSignalFieldSessionid,
			OSCParsersContextSignalFieldSignal:
			var str Optional[[]uint8]
			if len(value) > 0 {
				str = Some(value)
			}
			switch self {
			case OSCParsersContextSignalFieldUser:
				result.User = str
			case OSCParsersContextSignalFieldHostname:
				result.Hostname = str
			case OSCParsersContextSignalFieldMachineid:
				result.Machineid = str
			case OSCParsersContextSignalFieldBootid:
				result.Bootid = str
			case OSCParsersContextSignalFieldComm:
				result.Comm = str
			case OSCParsersContextSignalFieldCwd:
				result.Cwd = str
			case OSCParsersContextSignalFieldCmdline:
				result.Cmdline = str
			case OSCParsersContextSignalFieldVm:
				result.Vm = str
			case OSCParsersContextSignalFieldContainer:
				result.Container = str
			case OSCParsersContextSignalFieldTargetuser:
				result.Targetuser = str
			case OSCParsersContextSignalFieldTargethost:
				result.Targethost = str
			case OSCParsersContextSignalFieldSessionid:
				result.Sessionid = str
			case OSCParsersContextSignalFieldSignal:
				result.Signal = str
			}
		default:
			// Zig's switch is exhaustive over a comptime Field.
			panic("unreachable")
		}
		return result
	}

	// Not found
	return result
}

// OSCParsersContextSignalParse is osc.parsers.context_signal.parse: parse
// OSC 3008: hierarchical context signalling.
//
// Expected data format (after "3008;" prefix has been consumed by the state
// machine):
//
//	start=<id>[;<field>=<value>]*
//	end=<id>[;<field>=<value>]*
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*OSCCommand`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersContextSignalParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	writer := parser.writer
	if writer == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	data := writer.buffered()
	if len(data) == 0 {
		parser.state = OSCParserStateInvalid
		return nil
	}

	// Determine the action (start= or end=)
	var action OSCParsersContextSignalCommandAction
	if bytes.HasPrefix(data, []uint8("start=")) {
		action = OSCParsersContextSignalCommandActionStart
	} else if bytes.HasPrefix(data, []uint8("end=")) {
		action = OSCParsersContextSignalCommandActionEnd
	} else {
		warn("osc_context_signal", "OSC 3008: expected 'start=' or 'end=' prefix, got: %s",
			data[:min(len(data), 10)])
		parser.state = OSCParserStateInvalid
		return nil
	}

	// Skip past the "start=" or "end=" prefix
	var prefix_len int
	switch action {
	case OSCParsersContextSignalCommandActionStart:
		prefix_len = len("start=")
	case OSCParsersContextSignalCommandActionEnd:
		prefix_len = len("end=")
	}
	rest := data[prefix_len:]

	if len(rest) == 0 {
		warn("osc_context_signal", "OSC 3008: missing context ID")
		parser.state = OSCParserStateInvalid
		return nil
	}

	// Extract the context ID (up to the first semicolon or end of data)
	id_end := bytes.IndexByte(rest, ';')
	if id_end < 0 {
		id_end = len(rest)
	}
	id := rest[:id_end]

	// Validate context ID length (1-64 chars per spec)
	if len(id) == 0 || len(id) > oscParsersContextSignalMaxContextIdLen {
		warn("osc_context_signal", "OSC 3008: context ID length %d out of range (1-%d)",
			len(id),
			oscParsersContextSignalMaxContextIdLen)
		parser.state = OSCParserStateInvalid
		return nil
	}

	// Validate context ID characters (32..126 byte range per spec)
	for _, c := range id {
		if c < 0x20 || c > 0x7e {
			warn("osc_context_signal", "OSC 3008: invalid character 0x%02x in context ID", c)
			parser.state = OSCParserStateInvalid
			return nil
		}
	}

	// Extract raw metadata fields (everything after the ID)
	var metadata []uint8
	if id_end < len(rest) {
		metadata = rest[id_end+1:]
	} else {
		metadata = []uint8("")
	}

	parser.command = OSCCommand{
		Tag: OSCCommandKeyContextSignal,
		ContextSignal: OSCParsersContextSignalCommand{
			action:   action,
			id:       id,
			metadata: metadata,
		},
	}

	return &parser.command
}
