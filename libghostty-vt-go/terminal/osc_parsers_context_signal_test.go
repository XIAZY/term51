// Tests translated one for one from src/terminal/osc/parsers/context_signal.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers (expect,
// expectEqual, ...) are in Parser_test.go; oscA_term51Feed, oscA_unwrapPtr,
// oscA_ptrAt and oscA_terms are in
// osc_parsers_change_window_icon_test.go, where the translation notes are.
//
// Not translated (compile-time types, see scripts/not-translated.txt):
//   - test "term51: Field.Type: result type of every field"
//
// Translation notes:
//   - `Field.read(raw)` and `Command.readOption(option)` have a comptime
//     `self`/`option` and a result type that depends on it. In Go they return
//     an OSCParsersContextSignalFieldValue and the result is in the field
//     named after the requested option (.type is .Type, .user is .User, ...).
//     `x == null` is !x.Valid, `x.?` is zigUnwrap(t, x), and
//     `expectEqual(@as(?T, v), x)` is expectEqual(t, Some(v), x) for the
//     comparable payloads.
//   - `cmd.context_signal.action/.id/.metadata` are cmd.ContextSignal.action,
//     .id and .metadata (unexported fields keep Zig's names).
//   - `@tagName(field)` is field.String().

package terminal

import (
	"strings"
	"testing"
)

// oscA_csRead is `Field.read(raw)` with a Go string argument.
func oscA_csRead(f OSCParsersContextSignalField, raw string) OSCParsersContextSignalFieldValue {
	return f.Read([]uint8(raw))
}

// oscA_csStrField picks the result of a string-valued field out of a
// OSCParsersContextSignalFieldValue: the field named after the Field that was
// read. It stands for the Zig result type `?[]const u8`.
func oscA_csStrField(v OSCParsersContextSignalFieldValue, f OSCParsersContextSignalField) Optional[[]uint8] {
	switch f {
	case OSCParsersContextSignalFieldUser:
		return v.User
	case OSCParsersContextSignalFieldHostname:
		return v.Hostname
	case OSCParsersContextSignalFieldMachineid:
		return v.Machineid
	case OSCParsersContextSignalFieldBootid:
		return v.Bootid
	case OSCParsersContextSignalFieldComm:
		return v.Comm
	case OSCParsersContextSignalFieldCwd:
		return v.Cwd
	case OSCParsersContextSignalFieldCmdline:
		return v.Cmdline
	case OSCParsersContextSignalFieldVm:
		return v.Vm
	case OSCParsersContextSignalFieldContainer:
		return v.Container
	case OSCParsersContextSignalFieldTargetuser:
		return v.Targetuser
	case OSCParsersContextSignalFieldTargethost:
		return v.Targethost
	case OSCParsersContextSignalFieldSessionid:
		return v.Sessionid
	case OSCParsersContextSignalFieldSignal:
		return v.Signal
	}
	panic("not a string field")
}

// oscA_csNumField picks the result of a numeric field (?u64) out of a
// OSCParsersContextSignalFieldValue.
func oscA_csNumField(v OSCParsersContextSignalFieldValue, f OSCParsersContextSignalField) Optional[uint64] {
	switch f {
	case OSCParsersContextSignalFieldPid:
		return v.Pid
	case OSCParsersContextSignalFieldPidfdid:
		return v.Pidfdid
	case OSCParsersContextSignalFieldStatus:
		return v.Status
	}
	panic("not a numeric field")
}

// Zig: test "OSC 3008: basic start command"
func TestOSCParsersContextSignal_OSC_3008_basic_start_command(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;start=abc123"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, cmd.ContextSignal.Action == OSCParsersContextSignalCommandActionStart, "cmd.context_signal.action == .start")
	expectEqualStrings(t, "abc123", string(cmd.ContextSignal.ID))
	expectEqualStrings(t, "", string(cmd.ContextSignal.Metadata))
}

// Zig: test "OSC 3008: basic end command"
func TestOSCParsersContextSignal_OSC_3008_basic_end_command(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;end=abc123"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, cmd.ContextSignal.Action == OSCParsersContextSignalCommandActionEnd, "cmd.context_signal.action == .end")
	expectEqualStrings(t, "abc123", string(cmd.ContextSignal.ID))
	expectEqualStrings(t, "", string(cmd.ContextSignal.Metadata))
}

// Zig: test "OSC 3008: start with metadata fields"
func TestOSCParsersContextSignal_OSC_3008_start_with_metadata_fields(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;start=bed86fab93af4328bbed0a1224af6d40;type=container;user=lennart;hostname=zeta"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, cmd.ContextSignal.Action == OSCParsersContextSignalCommandActionStart, "cmd.context_signal.action == .start")
	expectEqualStrings(t, "bed86fab93af4328bbed0a1224af6d40", string(cmd.ContextSignal.ID))

	// Read individual fields
	expect(t, zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldType).Type) == OSCParsersContextSignalContextTypeContainer, "cmd.context_signal.readOption(.type).? == .container")
	expectEqualStrings(t, "lennart", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldUser).User)))
	expectEqualStrings(t, "zeta", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldHostname).Hostname)))
}

// Zig: test "OSC 3008: start with all common fields"
func TestOSCParsersContextSignal_OSC_3008_start_with_all_common_fields(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;start=myctx;type=shell;user=root;hostname=myhost;machineid=3deb5353d3ba43d08201c136a47ead7b;bootid=d4a3d0fdf2e24fdea6d971ce73f4fbf2;pid=1062862;pidfdid=1063162;comm=bash"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldType).Type) == OSCParsersContextSignalContextTypeShell, "cmd.context_signal.readOption(.type).? == .shell")
	expectEqualStrings(t, "root", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldUser).User)))
	expectEqualStrings(t, "myhost", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldHostname).Hostname)))
	expectEqualStrings(t, "3deb5353d3ba43d08201c136a47ead7b", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldMachineid).Machineid)))
	expectEqualStrings(t, "d4a3d0fdf2e24fdea6d971ce73f4fbf2", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldBootid).Bootid)))
	expectEqual(t, uint64(1062862), zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldPid).Pid))
	expectEqual(t, uint64(1063162), zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldPidfdid).Pidfdid))
	expectEqualStrings(t, "bash", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldComm).Comm)))
}

// Zig: test "OSC 3008: end with exit metadata"
func TestOSCParsersContextSignal_OSC_3008_end_with_exit_metadata(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;end=myctx;exit=success;status=0"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, cmd.ContextSignal.Action == OSCParsersContextSignalCommandActionEnd, "cmd.context_signal.action == .end")
	expectEqualStrings(t, "myctx", string(cmd.ContextSignal.ID))
	expect(t, zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldExit).Exit) == OSCParsersContextSignalExitStatusSuccess, "cmd.context_signal.readOption(.exit).? == .success")
	expectEqual(t, uint64(0), zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldStatus).Status))
}

// Zig: test "OSC 3008: end with failure exit"
func TestOSCParsersContextSignal_OSC_3008_end_with_failure_exit(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;end=myctx;exit=failure;status=1;signal=SIGKILL"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldExit).Exit) == OSCParsersContextSignalExitStatusFailure, "cmd.context_signal.readOption(.exit).? == .failure")
	expectEqual(t, uint64(1), zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldStatus).Status))
	expectEqualStrings(t, "SIGKILL", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldSignal).Signal)))
}

// Zig: test "OSC 3008: unknown fields are ignored"
func TestOSCParsersContextSignal_OSC_3008_unknown_fields_are_ignored(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;start=myctx;type=shell;unknownfield=value;user=root"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldType).Type) == OSCParsersContextSignalContextTypeShell, "cmd.context_signal.readOption(.type).? == .shell")
	expectEqualStrings(t, "root", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldUser).User)))
}

// Zig: test "OSC 3008: missing field returns null"
func TestOSCParsersContextSignal_OSC_3008_missing_field_returns_null(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;start=myctx;user=lennart"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, !cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldType).Type.Valid, "cmd.context_signal.readOption(.type) == null")
	expect(t, !cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldHostname).Hostname.Valid, "cmd.context_signal.readOption(.hostname) == null")
	expect(t, !cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldPid).Pid.Valid, "cmd.context_signal.readOption(.pid) == null")
}

// Zig: test "OSC 3008: invalid prefix"
func TestOSCParsersContextSignal_OSC_3008_invalid_prefix(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;bogus=abc123"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 3008: empty data"
func TestOSCParsersContextSignal_OSC_3008_empty_data(t *testing.T) {
	// Can't really produce empty data after "3008;" because the state machine
	// won't write a writer for that case, but we test the edge case where
	// only "start=" is present with no ID.
	p := OSCParserInit(false)
	input := "3008;start="
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 3008: max length context ID"
func TestOSCParsersContextSignal_OSC_3008_max_length_context_ID(t *testing.T) {
	p := OSCParserInit(false)
	id := strings.Repeat("a", 64)
	input := "3008;start=" + id
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expectEqualStrings(t, id, string(cmd.ContextSignal.ID))
}

// Zig: test "OSC 3008: over-length context ID"
func TestOSCParsersContextSignal_OSC_3008_over_length_context_ID(t *testing.T) {
	p := OSCParserInit(false)
	id := strings.Repeat("a", 65)
	input := "3008;start=" + id
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 3008: context type enum coverage"
func TestOSCParsersContextSignal_OSC_3008_context_type_enum_coverage(t *testing.T) {
	types := []struct {
		str      string
		expected OSCParsersContextSignalContextType
	}{
		{"boot", OSCParsersContextSignalContextTypeBoot},
		{"container", OSCParsersContextSignalContextTypeContainer},
		{"vm", OSCParsersContextSignalContextTypeVm},
		{"elevate", OSCParsersContextSignalContextTypeElevate},
		{"chpriv", OSCParsersContextSignalContextTypeChpriv},
		{"subcontext", OSCParsersContextSignalContextTypeSubcontext},
		{"remote", OSCParsersContextSignalContextTypeRemote},
		{"shell", OSCParsersContextSignalContextTypeShell},
		{"command", OSCParsersContextSignalContextTypeCommand},
		{"app", OSCParsersContextSignalContextTypeApp},
		{"service", OSCParsersContextSignalContextTypeService},
		{"session", OSCParsersContextSignalContextTypeSession},
	}

	for _, tt := range types {
		expectEqual(t, tt.expected, zigUnwrap(t, OSCParsersContextSignalContextTypeParse([]uint8(tt.str))))
	}

	expect(t, !OSCParsersContextSignalContextTypeParse([]uint8("invalid")).Valid, "ContextType.parse(\"invalid\") == null")
}

// Zig: test "OSC 3008: exit status enum coverage"
func TestOSCParsersContextSignal_OSC_3008_exit_status_enum_coverage(t *testing.T) {
	expect(t, zigUnwrap(t, OSCParsersContextSignalExitStatusParse([]uint8("success"))) == OSCParsersContextSignalExitStatusSuccess, "ExitStatus.parse(\"success\").? == .success")
	expect(t, zigUnwrap(t, OSCParsersContextSignalExitStatusParse([]uint8("failure"))) == OSCParsersContextSignalExitStatusFailure, "ExitStatus.parse(\"failure\").? == .failure")
	expect(t, zigUnwrap(t, OSCParsersContextSignalExitStatusParse([]uint8("crash"))) == OSCParsersContextSignalExitStatusCrash, "ExitStatus.parse(\"crash\").? == .crash")
	expect(t, zigUnwrap(t, OSCParsersContextSignalExitStatusParse([]uint8("interrupt"))) == OSCParsersContextSignalExitStatusInterrupt, "ExitStatus.parse(\"interrupt\").? == .interrupt")
	expect(t, !OSCParsersContextSignalExitStatusParse([]uint8("invalid")).Valid, "ExitStatus.parse(\"invalid\") == null")
}

// Zig: test "OSC 3008: spec example - container start"
func TestOSCParsersContextSignal_OSC_3008_spec_example_container_start(t *testing.T) {
	// From the spec: a new container "foobar" invoked by user "lennart" on host "zeta"
	p := OSCParserInit(false)
	input := "3008;start=bed86fab93af4328bbed0a1224af6d40;type=container;user=lennart;hostname=zeta;machineid=3deb5353d3ba43d08201c136a47ead7b;bootid=d4a3d0fdf2e24fdea6d971ce73f4fbf2;pid=1062862;pidfdid=1063162;comm=systemd-nspawn;container=foobar"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, cmd.ContextSignal.Action == OSCParsersContextSignalCommandActionStart, "cmd.context_signal.action == .start")
	expectEqualStrings(t, "bed86fab93af4328bbed0a1224af6d40", string(cmd.ContextSignal.ID))
	expect(t, zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldType).Type) == OSCParsersContextSignalContextTypeContainer, "cmd.context_signal.readOption(.type).? == .container")
	expectEqualStrings(t, "lennart", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldUser).User)))
	expectEqualStrings(t, "zeta", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldHostname).Hostname)))
	expectEqualStrings(t, "systemd-nspawn", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldComm).Comm)))
	expectEqualStrings(t, "foobar", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldContainer).Container)))
	expectEqual(t, uint64(1062862), zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldPid).Pid))
}

// Zig: test "OSC 3008: spec example - context end"
func TestOSCParsersContextSignal_OSC_3008_spec_example_context_end(t *testing.T) {
	// From the spec: context end
	p := OSCParserInit(false)
	input := "3008;end=bed86fab93af4328bbed0a1224af6d40"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, cmd.ContextSignal.Action == OSCParsersContextSignalCommandActionEnd, "cmd.context_signal.action == .end")
	expectEqualStrings(t, "bed86fab93af4328bbed0a1224af6d40", string(cmd.ContextSignal.ID))
}

// Zig: test "OSC 3008: cwd and cmdline fields"
func TestOSCParsersContextSignal_OSC_3008_cwd_and_cmdline_fields(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;start=myctx;type=command;cwd=/home/user;cmdline=ls -la"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expectEqualStrings(t, "/home/user", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldCwd).Cwd)))
	expectEqualStrings(t, "ls -la", string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldCmdline).Cmdline)))
}

// Zig: test "OSC 3008: start command with no fields"
func TestOSCParsersContextSignal_OSC_3008_start_command_with_no_fields(t *testing.T) {
	p := OSCParserInit(false)
	input := "3008;start=simpleid"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
	expect(t, cmd.ContextSignal.Action == OSCParsersContextSignalCommandActionStart, "cmd.context_signal.action == .start")
	expectEqualStrings(t, "simpleid", string(cmd.ContextSignal.ID))
	expect(t, !cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldType).Type.Valid, "cmd.context_signal.readOption(.type) == null")
	expect(t, !cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldUser).User.Valid, "cmd.context_signal.readOption(.user) == null")
	expect(t, !cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldExit).Exit.Valid, "cmd.context_signal.readOption(.exit) == null")
}

// term51_string_fields is the term51 constant of the same name.
var oscA_csTerm51StringFields = []OSCParsersContextSignalField{
	OSCParsersContextSignalFieldUser,
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
	OSCParsersContextSignalFieldSignal,
}

// Zig: test "term51: Field.key: the key is the field name"
func TestOSCParsersContextSignal_term51_Field_key_the_key_is_the_field_name(t *testing.T) {
	expectEqualStrings(t, "type", string(OSCParsersContextSignalFieldType.key()))
	expectEqualStrings(t, "user", string(OSCParsersContextSignalFieldUser.key()))
	expectEqualStrings(t, "hostname", string(OSCParsersContextSignalFieldHostname.key()))
	expectEqualStrings(t, "machineid", string(OSCParsersContextSignalFieldMachineid.key()))
	expectEqualStrings(t, "bootid", string(OSCParsersContextSignalFieldBootid.key()))
	expectEqualStrings(t, "pid", string(OSCParsersContextSignalFieldPid.key()))
	expectEqualStrings(t, "pidfdid", string(OSCParsersContextSignalFieldPidfdid.key()))
	expectEqualStrings(t, "comm", string(OSCParsersContextSignalFieldComm.key()))
	expectEqualStrings(t, "cwd", string(OSCParsersContextSignalFieldCwd.key()))
	expectEqualStrings(t, "cmdline", string(OSCParsersContextSignalFieldCmdline.key()))
	expectEqualStrings(t, "vm", string(OSCParsersContextSignalFieldVm.key()))
	expectEqualStrings(t, "container", string(OSCParsersContextSignalFieldContainer.key()))
	expectEqualStrings(t, "targetuser", string(OSCParsersContextSignalFieldTargetuser.key()))
	expectEqualStrings(t, "targethost", string(OSCParsersContextSignalFieldTargethost.key()))
	expectEqualStrings(t, "sessionid", string(OSCParsersContextSignalFieldSessionid.key()))
	expectEqualStrings(t, "exit", string(OSCParsersContextSignalFieldExit.key()))
	expectEqualStrings(t, "status", string(OSCParsersContextSignalFieldStatus.key()))
	expectEqualStrings(t, "signal", string(OSCParsersContextSignalFieldSignal.key()))
}

// Zig: test "term51: Field.read: empty input and missing field"
func TestOSCParsersContextSignal_term51_Field_read_empty_input_and_missing_field(t *testing.T) {
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, "").User.Valid, "Field.user.read(\"\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldPid, "").Pid.Valid, "Field.pid.read(\"\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldType, "").Type.Valid, "Field.type.read(\"\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldExit, "").Exit.Valid, "Field.exit.read(\"\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, ";;;").User.Valid, "Field.user.read(\";;;\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, "hostname=zeta;pid=1").User.Valid, "Field.user.read(\"hostname=zeta;pid=1\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldPid, "hostname=zeta;user=x").Pid.Valid, "Field.pid.read(\"hostname=zeta;user=x\") == null")
}

// Zig: test "term51: Field.read: every string field"
func TestOSCParsersContextSignal_term51_Field_read_every_string_field(t *testing.T) {
	for _, field := range oscA_csTerm51StringFields {
		name := field.String()
		expectEqualStrings(t, "v", string(zigUnwrap(t, oscA_csStrField(oscA_csRead(field, name+"=v"), field))))
		expectEqualStrings(t, "v", string(zigUnwrap(t, oscA_csStrField(oscA_csRead(field, "zzz=1;"+name+"=v;yyy=2"), field))))
		// An empty string value is reported as missing.
		expect(t, !oscA_csStrField(oscA_csRead(field, name+"="), field).Valid, "field.read(name ++ \"=\") == null")
		// A key without '=' is not a field.
		expect(t, !oscA_csStrField(oscA_csRead(field, name), field).Valid, "field.read(name) == null")
	}
}

// Zig: test "term51: Field.read: string values are returned verbatim"
func TestOSCParsersContextSignal_term51_Field_read_string_values_are_returned_verbatim(t *testing.T) {
	// Everything after the first '=' up to the next ';' is the value.
	expectEqualStrings(t, "a=b=c", string(zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldCmdline, "cmdline=a=b=c").Cmdline)))
	expectEqualStrings(t, " x ", string(zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldUser, "user= x ").User)))
	expectEqualStrings(t, "/tmp/a b", string(zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldCwd, "cwd=/tmp/a b;user=x").Cwd)))
	expectEqualSlices(t, []uint8("\x01\xff\x00"), zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldUser, "user=\x01\xff\x00").User))
	expectEqualStrings(t, "=", string(zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldUser, "user==").User)))
}

// Zig: test "term51: Field.read: keys must match exactly"
func TestOSCParsersContextSignal_term51_Field_read_keys_must_match_exactly(t *testing.T) {
	// case sensitive
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, "User=x").User.Valid, "Field.user.read(\"User=x\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, "USER=x").User.Valid, "Field.user.read(\"USER=x\") == null")
	// no trimming
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, " user=x").User.Valid, "Field.user.read(\" user=x\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, "user =x").User.Valid, "Field.user.read(\"user =x\") == null")
	// no prefix or suffix matching
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, "username=x").User.Valid, "Field.user.read(\"username=x\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, "targetuser=x").User.Valid, "Field.user.read(\"targetuser=x\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldPid, "pidfdid=5").Pid.Valid, "Field.pid.read(\"pidfdid=5\") == null")
	expectEqual(t, Some(uint64(5)), oscA_csRead(OSCParsersContextSignalFieldPidfdid, "pidfdid=5").Pidfdid)
	expectEqualStrings(t, "y", string(zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldUser, "username=x;user=y").User)))
	expectEqualStrings(t, "y", string(zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldVm, "type=vm;vm=y").Vm)))
}

// Zig: test "term51: Field.read: empty segments are skipped"
func TestOSCParsersContextSignal_term51_Field_read_empty_segments_are_skipped(t *testing.T) {
	expectEqualStrings(t, "x", string(zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldUser, ";;user=x;;").User)))
	expectEqualStrings(t, "x", string(zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldUser, "=;=y;user=x").User)))
}

// Zig: test "term51: Field.read: the first occurrence of a key decides"
func TestOSCParsersContextSignal_term51_Field_read_the_first_occurrence_of_a_key_decides(t *testing.T) {
	expectEqualStrings(t, "a", string(zigUnwrap(t, oscA_csRead(OSCParsersContextSignalFieldUser, "user=a;user=b").User)))
	// Even if the first occurrence is empty or malformed, later ones are
	// not consulted.
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldUser, "user=;user=b").User.Valid, "Field.user.read(\"user=;user=b\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldPid, "pid=x;pid=5").Pid.Valid, "Field.pid.read(\"pid=x;pid=5\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldType, "type=bogus;type=shell").Type.Valid, "Field.type.read(\"type=bogus;type=shell\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldExit, "exit=;exit=success").Exit.Valid, "Field.exit.read(\"exit=;exit=success\") == null")
}

// Zig: test "term51: Field.read: numeric fields"
func TestOSCParsersContextSignal_term51_Field_read_numeric_fields(t *testing.T) {
	for _, field := range []OSCParsersContextSignalField{
		OSCParsersContextSignalFieldPid,
		OSCParsersContextSignalFieldPidfdid,
		OSCParsersContextSignalFieldStatus,
	} {
		name := field.String()
		num := func(raw string) Optional[uint64] {
			return oscA_csNumField(oscA_csRead(field, raw), field)
		}
		expectEqual(t, Some(uint64(0)), num(name+"=0"))
		expectEqual(t, Some(uint64(1234)), num(name+"=1234"))
		expectEqual(t, Some(uint64(7)), num(name+"=007"))
		// largest u64 and one past it
		expectEqual(t, Some(uint64(18446744073709551615)), num(name+"=18446744073709551615"))
		expectEqual(t, Some(uint64(18446744073709551615)), num(name+"=00018446744073709551615"))
		expect(t, !num(name+"=18446744073709551616").Valid, "field.read(name ++ \"=18446744073709551616\") == null")
		expect(t, !num(name+"=99999999999999999999").Valid, "field.read(name ++ \"=99999999999999999999\") == null")
		// Only ASCII digits are allowed: no sign, no '_', no whitespace,
		// no hex.
		expect(t, !num(name+"=").Valid, "field.read(name ++ \"=\") == null")
		expect(t, !num(name+"=+1").Valid, "field.read(name ++ \"=+1\") == null")
		expect(t, !num(name+"=-1").Valid, "field.read(name ++ \"=-1\") == null")
		expect(t, !num(name+"=-0").Valid, "field.read(name ++ \"=-0\") == null")
		expect(t, !num(name+"=1_0").Valid, "field.read(name ++ \"=1_0\") == null")
		expect(t, !num(name+"= 1").Valid, "field.read(name ++ \"= 1\") == null")
		expect(t, !num(name+"=1 ").Valid, "field.read(name ++ \"=1 \") == null")
		expect(t, !num(name+"=0x10").Valid, "field.read(name ++ \"=0x10\") == null")
		expect(t, !num(name+"=12a").Valid, "field.read(name ++ \"=12a\") == null")
		expect(t, !num(name+"=1.5").Valid, "field.read(name ++ \"=1.5\") == null")
		expect(t, !num(name+"=1=2").Valid, "field.read(name ++ \"=1=2\") == null")
	}
}

// Zig: test "term51: Field.read: type field"
func TestOSCParsersContextSignal_term51_Field_read_type_field(t *testing.T) {
	expectEqual(t, Some(OSCParsersContextSignalContextTypeBoot), oscA_csRead(OSCParsersContextSignalFieldType, "type=boot").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeContainer), oscA_csRead(OSCParsersContextSignalFieldType, "type=container").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeVm), oscA_csRead(OSCParsersContextSignalFieldType, "type=vm").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeElevate), oscA_csRead(OSCParsersContextSignalFieldType, "type=elevate").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeChpriv), oscA_csRead(OSCParsersContextSignalFieldType, "type=chpriv").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeSubcontext), oscA_csRead(OSCParsersContextSignalFieldType, "type=subcontext").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeRemote), oscA_csRead(OSCParsersContextSignalFieldType, "type=remote").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeShell), oscA_csRead(OSCParsersContextSignalFieldType, "type=shell").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeCommand), oscA_csRead(OSCParsersContextSignalFieldType, "type=command").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeApp), oscA_csRead(OSCParsersContextSignalFieldType, "type=app").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeService), oscA_csRead(OSCParsersContextSignalFieldType, "type=service").Type)
	expectEqual(t, Some(OSCParsersContextSignalContextTypeSession), oscA_csRead(OSCParsersContextSignalFieldType, "user=x;type=session;pid=1").Type)

	expect(t, !oscA_csRead(OSCParsersContextSignalFieldType, "type=").Type.Valid, "Field.type.read(\"type=\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldType, "type=bogus").Type.Valid, "Field.type.read(\"type=bogus\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldType, "type=Shell").Type.Valid, "Field.type.read(\"type=Shell\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldType, "type=SHELL").Type.Valid, "Field.type.read(\"type=SHELL\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldType, "type= shell").Type.Valid, "Field.type.read(\"type= shell\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldType, "type=shell ").Type.Valid, "Field.type.read(\"type=shell \") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldType, "type=shell=x").Type.Valid, "Field.type.read(\"type=shell=x\") == null")
}

// Zig: test "term51: Field.read: exit field"
func TestOSCParsersContextSignal_term51_Field_read_exit_field(t *testing.T) {
	expectEqual(t, Some(OSCParsersContextSignalExitStatusSuccess), oscA_csRead(OSCParsersContextSignalFieldExit, "exit=success").Exit)
	expectEqual(t, Some(OSCParsersContextSignalExitStatusFailure), oscA_csRead(OSCParsersContextSignalFieldExit, "exit=failure").Exit)
	expectEqual(t, Some(OSCParsersContextSignalExitStatusCrash), oscA_csRead(OSCParsersContextSignalFieldExit, "exit=crash").Exit)
	expectEqual(t, Some(OSCParsersContextSignalExitStatusInterrupt), oscA_csRead(OSCParsersContextSignalFieldExit, "status=1;exit=interrupt").Exit)

	expect(t, !oscA_csRead(OSCParsersContextSignalFieldExit, "exit=").Exit.Valid, "Field.exit.read(\"exit=\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldExit, "exit=0").Exit.Valid, "Field.exit.read(\"exit=0\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldExit, "exit=Success").Exit.Valid, "Field.exit.read(\"exit=Success\") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldExit, "exit=success ").Exit.Valid, "Field.exit.read(\"exit=success \") == null")
	expect(t, !oscA_csRead(OSCParsersContextSignalFieldExit, "exit").Exit.Valid, "Field.exit.read(\"exit\") == null")
}

// Zig: test "term51: ContextType.parse: names are exact and case sensitive"
func TestOSCParsersContextSignal_term51_ContextType_parse_names_are_exact_and_case_sensitive(t *testing.T) {
	expectEqual(t, Some(OSCParsersContextSignalContextTypeBoot), OSCParsersContextSignalContextTypeParse([]uint8("boot")))
	expectEqual(t, Some(OSCParsersContextSignalContextTypeSession), OSCParsersContextSignalContextTypeParse([]uint8("session")))
	expect(t, !OSCParsersContextSignalContextTypeParse([]uint8("")).Valid, "ContextType.parse(\"\") == null")
	expect(t, !OSCParsersContextSignalContextTypeParse([]uint8("Boot")).Valid, "ContextType.parse(\"Boot\") == null")
	expect(t, !OSCParsersContextSignalContextTypeParse([]uint8("boot ")).Valid, "ContextType.parse(\"boot \") == null")
	expect(t, !OSCParsersContextSignalContextTypeParse([]uint8("boo")).Valid, "ContextType.parse(\"boo\") == null")
	expect(t, !OSCParsersContextSignalContextTypeParse([]uint8("boots")).Valid, "ContextType.parse(\"boots\") == null")
	expectEqual(t, 12, int(oscParsersContextSignalContextTypeCount))
}

// Zig: test "term51: ExitStatus.parse: names are exact and case sensitive"
func TestOSCParsersContextSignal_term51_ExitStatus_parse_names_are_exact_and_case_sensitive(t *testing.T) {
	expectEqual(t, Some(OSCParsersContextSignalExitStatusSuccess), OSCParsersContextSignalExitStatusParse([]uint8("success")))
	expectEqual(t, Some(OSCParsersContextSignalExitStatusInterrupt), OSCParsersContextSignalExitStatusParse([]uint8("interrupt")))
	expect(t, !OSCParsersContextSignalExitStatusParse([]uint8("")).Valid, "ExitStatus.parse(\"\") == null")
	expect(t, !OSCParsersContextSignalExitStatusParse([]uint8("SUCCESS")).Valid, "ExitStatus.parse(\"SUCCESS\") == null")
	expect(t, !OSCParsersContextSignalExitStatusParse([]uint8("succes")).Valid, "ExitStatus.parse(\"succes\") == null")
	expect(t, !OSCParsersContextSignalExitStatusParse([]uint8("0")).Valid, "ExitStatus.parse(\"0\") == null")
	expectEqual(t, 4, int(oscParsersContextSignalExitStatusCount))
}

// Zig: test "term51: Command.readOption: reads from the metadata"
func TestOSCParsersContextSignal_term51_Command_readOption_reads_from_the_metadata(t *testing.T) {
	cmd := OSCParsersContextSignalCommand{
		Action:   OSCParsersContextSignalCommandActionEnd,
		ID:       []uint8("x"),
		Metadata: []uint8("exit=crash;status=139;signal=SIGSEGV"),
	}
	expectEqual(t, Some(OSCParsersContextSignalExitStatusCrash), cmd.ReadOption(OSCParsersContextSignalFieldExit).Exit)
	expectEqual(t, Some(uint64(139)), cmd.ReadOption(OSCParsersContextSignalFieldStatus).Status)
	expectEqualStrings(t, "SIGSEGV", string(zigUnwrap(t, cmd.ReadOption(OSCParsersContextSignalFieldSignal).Signal)))
	expect(t, !cmd.ReadOption(OSCParsersContextSignalFieldUser).User.Valid, "cmd.readOption(.user) == null")

	empty := OSCParsersContextSignalCommand{Action: OSCParsersContextSignalCommandActionStart, ID: []uint8("x"), Metadata: []uint8("")}
	expect(t, !empty.ReadOption(OSCParsersContextSignalFieldType).Type.Valid, "empty.readOption(.type) == null")
	expect(t, !empty.ReadOption(OSCParsersContextSignalFieldPid).Pid.Valid, "empty.readOption(.pid) == null")
	expect(t, !empty.ReadOption(OSCParsersContextSignalFieldCwd).Cwd.Valid, "empty.readOption(.cwd) == null")
}

// Zig: test "term51: parse: missing separator after 3008 is invalid"
func TestOSCParsersContextSignal_term51_parse_missing_separator_after_3008_is_invalid(t *testing.T) {
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "3008")
	expect(t, p.state == OSCParserState3008, "p.state == .@\"3008\"")
	expect(t, OSCParsersContextSignalParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
}

// Zig: test "term51: parse: invalid prefixes"
func TestOSCParsersContextSignal_term51_parse_invalid_prefixes(t *testing.T) {
	for _, input := range []string{
		"3008;",
		"3008;start",
		"3008;end",
		"3008;start=",
		"3008;end=",
		"3008;START=abc",
		"3008;Start=abc",
		"3008;End=abc",
		"3008; start=abc",
		"3008;stop=abc",
		"3008;abc",
		"3008;;start=abc",
		// an empty id followed by metadata
		"3008;start=;type=shell",
	} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, input)
		expect(t, OSCParsersContextSignalParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: action, id and metadata"
func TestOSCParsersContextSignal_term51_parse_action_id_and_metadata(t *testing.T) {
	for _, c := range []struct {
		input    string
		action   OSCParsersContextSignalCommandAction
		id       string
		metadata string
	}{
		{"3008;start=a", OSCParsersContextSignalCommandActionStart, "a", ""},
		{"3008;end=a", OSCParsersContextSignalCommandActionEnd, "a", ""},
		// a trailing ';' gives empty metadata
		{"3008;end=a;", OSCParsersContextSignalCommandActionEnd, "a", ""},
		// metadata is everything after the first ';', unparsed
		{"3008;start=a;;b=c;", OSCParsersContextSignalCommandActionStart, "a", ";b=c;"},
		{"3008;start=a;not a field\x01\xff", OSCParsersContextSignalCommandActionStart, "a", "not a field\x01\xff"},
		// the id may contain '=' and spaces
		{"3008;start=a=b c;x=y", OSCParsersContextSignalCommandActionStart, "a=b c", "x=y"},
		{"3008;end=start=x", OSCParsersContextSignalCommandActionEnd, "start=x", ""},
	} {
		for _, term := range oscA_terms {
			p := OSCParserInit(false)
			oscA_term51Feed(&p, c.input)
			cmd := *oscA_unwrapPtr(t, OSCParsersContextSignalParse(&p, term))
			expect(t, cmd.Tag == OSCCommandKeyContextSignal, "cmd == .context_signal")
			expectEqual(t, c.action, cmd.ContextSignal.Action)
			expectEqualStrings(t, c.id, string(cmd.ContextSignal.ID))
			expectEqualSlices(t, []uint8(c.metadata), cmd.ContextSignal.Metadata)
		}
	}
}

// Zig: test "term51: parse: id length limits"
func TestOSCParsersContextSignal_term51_parse_id_length_limits(t *testing.T) {
	for _, n := range []int{1, 63, 64} {
		id := strings.Repeat("i", n)
		for _, suffix := range []string{"", ";type=shell"} {
			p := OSCParserInit(false)
			oscA_term51Feed(&p, "3008;end="+id+suffix)
			cmd := *oscA_unwrapPtr(t, OSCParsersContextSignalParse(&p, None[uint8]()))
			expectEqualStrings(t, id, string(cmd.ContextSignal.ID))
		}
	}
	for _, n := range []int{65, 100} {
		id := strings.Repeat("i", n)
		for _, suffix := range []string{"", ";type=shell"} {
			p := OSCParserInit(false)
			oscA_term51Feed(&p, "3008;end="+id+suffix)
			expect(t, OSCParsersContextSignalParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
			expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		}
	}
}

// Zig: test "term51: parse: id byte range"
func TestOSCParsersContextSignal_term51_parse_id_byte_range(t *testing.T) {
	// 0x20 (space) and 0x7e ('~') are the smallest and largest valid bytes.
	for _, id := range []string{" ", "~", " ~ ", "a b"} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "3008;start="+id)
		cmd := *oscA_unwrapPtr(t, OSCParsersContextSignalParse(&p, None[uint8]()))
		expectEqualStrings(t, id, string(cmd.ContextSignal.ID))
	}
	for _, id := range []string{"\x1f", "\x7f", "\x00", "a\x1fb", "a\x7f", "\x80", "\xc3\xa9", "\xff"} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "3008;start="+id)
		expect(t, OSCParsersContextSignalParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		// The metadata is not subject to the same restriction.
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "3008;start=a;user=\x1f\x7f\xff")
		cmd := *oscA_unwrapPtr(t, OSCParsersContextSignalParse(&p, None[uint8]()))
		expectEqualSlices(t, []uint8("\x1f\x7f\xff"), zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldUser).User))
	}
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersContextSignal_term51_parse_buffer_limit(t *testing.T) {
	// No NUL terminator is needed, so the data may fill the whole buffer.
	const prefix = "start=a;cwd="
	{
		value := strings.Repeat("d", OSCParserMAX_BUF-len(prefix))
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "3008;"+prefix+value)
		expect(t, p.state == OSCParserState3008, "p.state == .@\"3008\"")
		cmd := *oscA_unwrapPtr(t, OSCParsersContextSignalParse(&p, None[uint8]()))
		expectEqualStrings(t, "a", string(cmd.ContextSignal.ID))
		expectEqual(t, OSCParserMAX_BUF-len("start=a;"), len(cmd.ContextSignal.Metadata))
		expectEqualStrings(t, value, string(zigUnwrap(t, cmd.ContextSignal.ReadOption(OSCParsersContextSignalFieldCwd).Cwd)))
	}
	{
		// One more byte overflows while feeding; an allocator does not help.
		value := strings.Repeat("d", OSCParserMAX_BUF-len(prefix)+1)
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "3008;"+prefix+value)
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}
}
