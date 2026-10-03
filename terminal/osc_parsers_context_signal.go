// WIREFRAME for src/terminal/osc/parsers/context_signal.zig (Ghostty
// v1.3.1). NOT YET PORTED.
//
// OSC 3008: Hierarchical Context Signalling (UAPI spec)
// Specification: https://uapi-group.org/specifications/specs/osc_context/
//
// OSC 3008 allows programs to signal context changes to the terminal
// emulator. Each context has an identifier and metadata fields. Contexts are
// hierarchical and form a stack.
//
// Types and signatures only; see the header of color.go for the rules.
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
	panic(notImplemented)
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
func (self OSCParsersContextSignalContextType) String() string { panic(notImplemented) }

// OSCParsersContextSignalContextTypeParse is
// osc.parsers.context_signal.ContextType.parse:
// std.meta.stringToEnum(ContextType, value).
func OSCParsersContextSignalContextTypeParse(value []uint8) Optional[OSCParsersContextSignalContextType] {
	panic(notImplemented)
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
func (self OSCParsersContextSignalExitStatus) String() string { panic(notImplemented) }

// OSCParsersContextSignalExitStatusParse is
// osc.parsers.context_signal.ExitStatus.parse:
// std.meta.stringToEnum(ExitStatus, value).
func OSCParsersContextSignalExitStatusParse(value []uint8) Optional[OSCParsersContextSignalExitStatus] {
	panic(notImplemented)
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
func (self OSCParsersContextSignalField) String() string { panic(notImplemented) }

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
func (self OSCParsersContextSignalField) key() []uint8 { panic(notImplemented) }

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
	panic(notImplemented)
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
	panic(notImplemented)
}
