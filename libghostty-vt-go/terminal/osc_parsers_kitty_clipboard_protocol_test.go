// Tests translated one for one from src/terminal/osc/parsers/kitty_clipboard_protocol.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualSlices, expectEqualStrings and zigUnwrap are defined in
// Parser_test.go.
//
// Translation notes:
//   - `Parser.init(testing.allocator)` is OSCParserInit(true),
//     any non-nil value ("an allocator was provided"); `Parser.init(null)` is
//     OSCParserInit(false). The leak check of testing.allocator has no Go
//     counterpart; `defer p.deinit()` is kept as defer p.Deinit(). Zig's
//     `defer` inside a loop body runs at the end of each iteration; Go's
//     defer runs when the test function returns.
//   - `p.end(ch).?.*` is oscB_kcp_end: it fails the test when End returns nil,
//     otherwise yields the command by value.
//   - `x.readOption(.key)` / `Option.key.read(m)` return an
//     OSCParsersKittyClipboardProtocolOptionValue; the result is in the field
//     named after the option (.id is Id, .type is Type, ...). `== null` is
//     `!<field>.Valid`, `.?` is zigUnwrap, and comparing an optional with an
//     enum value is comparing with Some(value).
//   - `[]const u8` results are []uint8; they are compared as strings.
//   - The term51 test "Option.Type: result type of every option" checks
//     compile-time types and is not translated (see
//     scripts/not-translated-oscB_kcp.txt).

package terminal

import (
	"strings"
	"testing"
)

// oscB_kcp_feed is term51Feed: feed every byte of input to the parser.
func oscB_kcp_feed(p *OSCParser, input string) {
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}
}

// oscB_kcp_deref is Zig's `x.?.*` on an optional pointer to a command.
func oscB_kcp_deref(t *testing.T, cmd *OSCCommand) OSCCommand {
	t.Helper()
	if cmd == nil {
		t.Fatalf("unwrapped a null command")
	}
	return *cmd
}

// oscB_kcp_end is `p.end(terminator_ch).?.*`.
func oscB_kcp_end(t *testing.T, p *OSCParser, terminator_ch Optional[uint8]) OSCCommand {
	t.Helper()
	return oscB_kcp_deref(t, p.End(terminator_ch))
}

// oscB_kcp_read is `key.read(metadata)` with a string literal as metadata.
func oscB_kcp_read(key OSCParsersKittyClipboardProtocolOption, metadata string) OSCParsersKittyClipboardProtocolOptionValue {
	return key.Read([]uint8(metadata))
}

// oscB_kcp_str is `o.?` on a `?[]const u8`, as a string.
func oscB_kcp_str(t *testing.T, o Optional[[]uint8]) string {
	t.Helper()
	return string(zigUnwrap(t, o))
}

// Zig: test "OSC: 5522: empty metadata and missing payload"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_empty_metadata_and_missing_payload(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "", string(cmd.KittyClipboardProtocol.Metadata))
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type.Valid, "cmd.kitty_clipboard_protocol.readOption(.type) == null")
}

// Zig: test "OSC: 5522: empty metadata and empty payload"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_empty_metadata_and_empty_payload(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;;"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "", string(cmd.KittyClipboardProtocol.Metadata))
	expectEqualStrings(t, "", string(zigUnwrap(t, cmd.KittyClipboardProtocol.Payload)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type.Valid, "cmd.kitty_clipboard_protocol.readOption(.type) == null")
}

// Zig: test "OSC: 5522: non-empty metadata and payload"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_non_empty_metadata_and_payload(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read;dGV4dC9wbGFpbg=="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "type=read", string(cmd.KittyClipboardProtocol.Metadata))
	expectEqualStrings(t, "dGV4dC9wbGFpbg==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.Payload)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationRead), cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type)
}

// Zig: test "OSC: 5522: empty id"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_empty_id(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;id="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
}

// Zig: test "OSC: 5522: valid id"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_valid_id(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;id=5c076ad9-d36f-4705-847b-d4dbf356cc0d"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "5c076ad9-d36f-4705-847b-d4dbf356cc0d", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id)))
}

// Zig: test "OSC: 5522: invalid id"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_invalid_id(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;id=*42*"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
}

// Zig: test "OSC: 5522: invalid status"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_invalid_status(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;status=BOBR"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
}

// Zig: test "OSC: 5522: valid status"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_valid_status(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;status=DONE"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusDONE, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
}

// Zig: test "OSC: 5522: invalid location"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_invalid_location(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;loc=bobr"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
}

// Zig: test "OSC: 5522: valid location"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_valid_location(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;loc=primary"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqual(t, OSCParsersKittyClipboardProtocolLocationPrimary, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc))
}

// Zig: test "OSC: 5522: password 1"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_password_1(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;pw=R2hvc3R0eQ==:name=Qk9CUiBLVVJXQQ=="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "R2hvc3R0eQ==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw)))
	expectEqualStrings(t, "Qk9CUiBLVVJXQQ==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name)))
}

// Zig: test "OSC: 5522: password 2"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_password_2(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;password=R2hvc3R0eQ=="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "R2hvc3R0eQ==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password)))
}

// Zig: test "OSC: 5522: example 1"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_1(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read:status=OK"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusOK, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationRead, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 2"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_2(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read:mime=dGV4dC9wbGFpbg==;R2hvc3R0eQ=="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "R2hvc3R0eQ==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.Payload)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expectEqualStrings(t, "dGV4dC9wbGFpbg==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationRead, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 3"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_3(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read:status=OK"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusOK, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationRead, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 4"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_4(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=write"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationWrite, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 5"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_5(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=wdata:mime=dGV4dC9wbGFpbg==;R2hvc3R0eQ=="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "R2hvc3R0eQ==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.Payload)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expectEqualStrings(t, "dGV4dC9wbGFpbg==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationWdata, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 6"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_6(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=wdata"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationWdata, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 7"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_7(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=write:status=DONE"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusDONE, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationWrite, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 8"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_8(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=write:status=EPERM"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusEPERM, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationWrite, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 9"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_9(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=walias:mime=dGV4dC9wbGFpbg==;dGV4dC9odG1sIGFwcGxpY2F0aW9uL2pzb24="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "dGV4dC9odG1sIGFwcGxpY2F0aW9uL2pzb24=", string(zigUnwrap(t, cmd.KittyClipboardProtocol.Payload)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expectEqualStrings(t, "dGV4dC9wbGFpbg==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationWalias, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 10"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_10(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read:status=OK:password=Qk9CUiBLVVJXQQ=="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expectEqualStrings(t, "Qk9CUiBLVVJXQQ==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusOK, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationRead, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 11"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_11(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read:status=DATA:mime=dGV4dC9wbGFpbg=="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expectEqualStrings(t, "dGV4dC9wbGFpbg==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusDATA, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationRead, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 12"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_12(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read:mime=dGV4dC9wbGFpbg==:password=Qk9CUiBLVVJXQQ=="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expectEqualStrings(t, "dGV4dC9wbGFpbg==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expectEqualStrings(t, "Qk9CUiBLVVJXQQ==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "cmd.kitty_clipboard_protocol.readOption(.status) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationRead, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 13"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_13(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read:status=OK"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusOK, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationRead, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 14"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_14(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read:status=DATA:mime=dGV4dC9wbGFpbg==;Qk9CUiBLVVJXQQ=="
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expectEqualStrings(t, "Qk9CUiBLVVJXQQ==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.Payload)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expectEqualStrings(t, "dGV4dC9wbGFpbg==", string(zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime)))
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusDATA, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationRead, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// Zig: test "OSC: 5522: example 15"
func TestOSCParsersKittyClipboardProtocol_OSC_5522_example_15(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	input := "5522;type=read:status=OK"
	oscB_kcp_feed(&p, input)
	cmd := oscB_kcp_end(t, &p, Some[uint8](0x1b))
	expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
	expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "cmd.kitty_clipboard_protocol.readOption(.id) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionLoc).Loc.Valid, "cmd.kitty_clipboard_protocol.readOption(.loc) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime.Valid, "cmd.kitty_clipboard_protocol.readOption(.mime) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionName).Name.Valid, "cmd.kitty_clipboard_protocol.readOption(.name) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPassword).Password.Valid, "cmd.kitty_clipboard_protocol.readOption(.password) == null")
	expect(t, !cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionPw).Pw.Valid, "cmd.kitty_clipboard_protocol.readOption(.pw) == null")
	expectEqual(t, OSCParsersKittyClipboardProtocolStatusOK, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status))
	expectEqual(t, OSCParsersKittyClipboardProtocolOperationRead, zigUnwrap(t, cmd.KittyClipboardProtocol.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type))
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: Option.read: empty and whitespace-only metadata"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_empty_and_whitespace_only_metadata(t *testing.T) {
	for _, metadata := range []string{"", " ", " \t\r\n\x0b\x0c", ":", "::", " : : "} {
		expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, metadata).Id.Valid, "Option.id.read(metadata) == null")
		expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionLoc, metadata).Loc.Valid, "Option.loc.read(metadata) == null")
		expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, metadata).Mime.Valid, "Option.mime.read(metadata) == null")
		expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, metadata).Name.Valid, "Option.name.read(metadata) == null")
		expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPassword, metadata).Password.Valid, "Option.password.read(metadata) == null")
		expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPw, metadata).Pw.Valid, "Option.pw.read(metadata) == null")
		expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, metadata).Status.Valid, "Option.status.read(metadata) == null")
		expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, metadata).Type.Valid, "Option.type.read(metadata) == null")
	}
}

// Zig: test "term51: Option.read: string options"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_string_options(t *testing.T) {
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=abc").Id))
	expectEqualStrings(t, "text/plain", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, "mime=text/plain").Mime))
	expectEqualStrings(t, "n", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, "name=n").Name))
	expectEqualStrings(t, "secret", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPassword, "password=secret").Password))
	expectEqualStrings(t, "secret", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPw, "pw=secret").Pw))
}

// Zig: test "term51: Option.read: option position"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_option_position(t *testing.T) {
	metadata := "type=write:id=abc:mime=text/plain:name=n:password=p1:pw=p2:status=OK:loc=primary"
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationWrite), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, metadata).Type)
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, metadata).Id))
	expectEqualStrings(t, "text/plain", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, metadata).Mime))
	expectEqualStrings(t, "n", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, metadata).Name))
	expectEqualStrings(t, "p1", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPassword, metadata).Password))
	expectEqualStrings(t, "p2", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPw, metadata).Pw))
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusOK), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, metadata).Status)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolLocationPrimary), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionLoc, metadata).Loc)

	// A trailing or doubled ':' is harmless.
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=abc:").Id))
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "type=read::id=abc").Id))
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, ":id=abc").Id))
	// A missing option after other options.
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "type=read:mime=a").Id.Valid, "Option.id.read(\"type=read:mime=a\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "type=read:").Id.Valid, "Option.id.read(\"type=read:\") == null")
}

// Zig: test "term51: Option.read: whitespace is skipped around keys and trimmed from values"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_whitespace_is_skipped_around_keys_and_trimmed_from_values(t *testing.T) {
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "  id  =  abc  ").Id))
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "\t\r\n\x0b\x0cid\t\r\n\x0b\x0c=\t\r\n\x0b\x0cabc\t\r\n\x0b\x0c").Id))
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "type=read :\tid = abc : mime=x").Id))
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationRead), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, " type = read : id=abc").Type)
	// Whitespace inside a value is kept.
	expectEqualStrings(t, "a b\tc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, "name=  a b\tc  ").Name))
	expectEqualStrings(t, "s3 cr3t", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPassword, "password= s3 cr3t ").Password))
	// A value that is only whitespace is empty.
	expectEqualStrings(t, "", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, "name=   ").Name))
	// Other control bytes are not whitespace.
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "\x00id=abc").Id.Valid, "Option.id.read(\"\\x00id=abc\") == null")
	expectEqualSlices(t, []uint8("\x00a\x1f"), zigUnwrap(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, "name=\x00a\x1f").Name))
}

// Zig: test "term51: Option.read: empty values"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_empty_values(t *testing.T) {
	// Plain string options return the empty string...
	expectEqualStrings(t, "", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, "mime=").Mime))
	expectEqualStrings(t, "", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, "name=").Name))
	expectEqualStrings(t, "", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPassword, "password=").Password))
	expectEqualStrings(t, "", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPw, "pw=").Pw))
	expectEqualStrings(t, "", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, "mime=:id=a").Mime))
	// ...validated options return null.
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=").Id.Valid, "Option.id.read(\"id=\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=:mime=a").Id.Valid, "Option.id.read(\"id=:mime=a\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionLoc, "loc=").Loc.Valid, "Option.loc.read(\"loc=\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=").Status.Valid, "Option.status.read(\"status=\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=").Type.Valid, "Option.type.read(\"type=\") == null")
}

// Zig: test "term51: Option.read: key without equals"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_key_without_equals(t *testing.T) {
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id").Id.Valid, "Option.id.read(\"id\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id ").Id.Valid, "Option.id.read(\"id \") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id:").Id.Valid, "Option.id.read(\"id:\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, "mime").Mime.Valid, "Option.mime.read(\"mime\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, "mime:mime=a").Mime.Valid, "Option.mime.read(\"mime:mime=a\") == null")
	// The search is abandoned at the first malformed occurrence, even if a
	// well-formed one follows.
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id:id=abc").Id.Valid, "Option.id.read(\"id:id=abc\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id abc:id=abc").Id.Valid, "Option.id.read(\"id abc:id=abc\") == null")
}

// Zig: test "term51: Option.read: keys are matched by prefix"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_keys_are_matched_by_prefix(t *testing.T) {
	// A longer key that merely starts with the wanted key is taken for the
	// wanted key, and as it is not followed by '=' the search is abandoned:
	// the real option after it is never found.
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "idx=1:id=abc").Id.Valid, "Option.id.read(\"idx=1:id=abc\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "identifier=1:id=abc").Id.Valid, "Option.id.read(\"identifier=1:id=abc\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, "mimetype=a:mime=b").Mime.Valid, "Option.mime.read(\"mimetype=a:mime=b\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "types=read:type=read").Type.Valid, "Option.type.read(\"types=read:type=read\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPw, "pwd=x:pw=y").Pw.Valid, "Option.pw.read(\"pwd=x:pw=y\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id-x=5").Id.Valid, "Option.id.read(\"id-x=5\") == null")
	// In the other order the real option is found first.
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=abc:idx=1").Id))

	// "password" does not start with "pw" and vice versa, so these two do
	// not shadow each other.
	expectEqualStrings(t, "b", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPw, "password=a:pw=b").Pw))
	expectEqualStrings(t, "a", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPassword, "pw=b:password=a").Password))

	// A key is only looked for at the start of an option, not inside a
	// key or a value.
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "valid=1").Id.Valid, "Option.id.read(\"valid=1\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "name=id=5").Id.Valid, "Option.id.read(\"name=id=5\") == null")
	expectEqualStrings(t, "5", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "name=id=7:id=5").Id))
	expectEqualStrings(t, "id=7", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, "name=id=7:id=5").Name))
}

// Zig: test "term51: Option.read: keys are case sensitive"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_keys_are_case_sensitive(t *testing.T) {
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "ID=abc").Id.Valid, "Option.id.read(\"ID=abc\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "Id=abc").Id.Valid, "Option.id.read(\"Id=abc\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, "MIME=a").Mime.Valid, "Option.mime.read(\"MIME=a\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "Type=read").Type.Valid, "Option.type.read(\"Type=read\") == null")
	expectEqualStrings(t, "abc", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "ID=x:id=abc").Id))
}

// Zig: test "term51: Option.read: the first occurrence of a key decides"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_the_first_occurrence_of_a_key_decides(t *testing.T) {
	expectEqualStrings(t, "a", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=a:id=b").Id))
	expectEqualStrings(t, "a", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionMime, "mime=a:mime=b").Mime))
	// Even if the first occurrence is invalid.
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=*:id=b").Id.Valid, "Option.id.read(\"id=*:id=b\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=bogus:type=read").Type.Valid, "Option.type.read(\"type=bogus:type=read\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=:status=OK").Status.Valid, "Option.status.read(\"status=:status=OK\") == null")
}

// Zig: test "term51: Option.read: values keep equals signs and end at the first colon"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_values_keep_equals_signs_and_end_at_the_first_colon(t *testing.T) {
	expectEqualStrings(t, "a=b=c", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, "name=a=b=c").Name))
	expectEqualStrings(t, "YQ==", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionPassword, "password=YQ==:id=a").Password))
	expectEqualStrings(t, "a", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, "name=a:b").Name))
	expectEqualStrings(t, "=", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionName, "name==").Name))
}

// Zig: test "term51: Option.read: id is validated"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_id_is_validated(t *testing.T) {
	expectEqualStrings(t, "aZ09-_+.", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=aZ09-_+.").Id))
	expectEqualStrings(t, "a", oscB_kcp_str(t, oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id= a ").Id))
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=a b").Id.Valid, "Option.id.read(\"id=a b\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=a/b").Id.Valid, "Option.id.read(\"id=a/b\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=a=b").Id.Valid, "Option.id.read(\"id=a=b\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionId, "id=\xc3\xa9").Id.Valid, "Option.id.read(\"id=\\xc3\\xa9\") == null")
}

// Zig: test "term51: Option.read: loc values"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_loc_values(t *testing.T) {
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolLocationPrimary), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionLoc, "loc=primary").Loc)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolLocationPrimary), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionLoc, "loc = primary ").Loc)
	// "primary" is the only known location.
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionLoc, "loc=clipboard").Loc.Valid, "Option.loc.read(\"loc=clipboard\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionLoc, "loc=Primary").Loc.Valid, "Option.loc.read(\"loc=Primary\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionLoc, "loc=PRIMARY").Loc.Valid, "Option.loc.read(\"loc=PRIMARY\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionLoc, "loc=primary2").Loc.Valid, "Option.loc.read(\"loc=primary2\") == null")
}

// Zig: test "term51: Option.read: type values"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_type_values(t *testing.T) {
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationRead), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=read").Type)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationWalias), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=walias").Type)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationWdata), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=wdata").Type)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationWrite), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=write").Type)
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=copy").Type.Valid, "Option.type.read(\"type=copy\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=Read").Type.Valid, "Option.type.read(\"type=Read\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=READ").Type.Valid, "Option.type.read(\"type=READ\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=rea").Type.Valid, "Option.type.read(\"type=rea\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionType, "type=read=x").Type.Valid, "Option.type.read(\"type=read=x\") == null")
}

// Zig: test "term51: Option.read: status values"
func TestOSCParsersKittyClipboardProtocol_term51_Option_read_status_values(t *testing.T) {
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusDATA), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=DATA").Status)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusDONE), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=DONE").Status)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusEBUSY), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=EBUSY").Status)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusEINVAL), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=EINVAL").Status)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusEIO), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=EIO").Status)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusENOSYS), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=ENOSYS").Status)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusEPERM), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=EPERM").Status)
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusOK), oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=OK").Status)
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=ok").Status.Valid, "Option.status.read(\"status=ok\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=Done").Status.Valid, "Option.status.read(\"status=Done\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=ENOENT").Status.Valid, "Option.status.read(\"status=ENOENT\") == null")
	expect(t, !oscB_kcp_read(OSCParsersKittyClipboardProtocolOptionStatus, "status=0").Status.Valid, "Option.status.read(\"status=0\") == null")
}

// Zig: test "term51: isValidIdentifier: empty string"
func TestOSCParsersKittyClipboardProtocol_term51_isValidIdentifier_empty_string(t *testing.T) {
	expect(t, !oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("")), "!isValidIdentifier(\"\")")
}

// Zig: test "term51: isValidIdentifier: every byte value"
func TestOSCParsersKittyClipboardProtocol_term51_isValidIdentifier_every_byte_value(t *testing.T) {
	// Exactly the 66 bytes a-z, A-Z, 0-9, '-', '_', '+' and '.' are valid.
	valid := 0
	for i := 0; i < 256; i++ {
		c := uint8(i)
		var expected bool
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '+', c == '.':
			expected = true
		default:
			expected = false
		}
		expectEqual(t, expected, oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8{c}))
		// One invalid byte anywhere makes the whole string invalid.
		expectEqual(t, expected, oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8{'a', c}))
		expectEqual(t, expected, oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8{c, 'a'}))
		if expected {
			valid += 1
		}
	}
	expectEqual(t, 66, valid)
	expectEqual(t, 66, len(oscParsersKittyClipboardProtocolValidIdentifierCharacters))
}

// Zig: test "term51: isValidIdentifier: typical values"
func TestOSCParsersKittyClipboardProtocol_term51_isValidIdentifier_typical_values(t *testing.T) {
	expect(t, oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("a")), "isValidIdentifier(\"a\")")
	expect(t, oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("5c076ad9-d36f-4705-847b-d4dbf356cc0d")), "isValidIdentifier(\"5c076ad9-d36f-4705-847b-d4dbf356cc0d\")")
	expect(t, oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_+.")), "isValidIdentifier(all valid characters)")
	// There is no length limit.
	expect(t, oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8(strings.Repeat("a", 5000))), "isValidIdentifier(\"a\" ** 5000)")
	expect(t, !oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8(" a")), "!isValidIdentifier(\" a\")")
	expect(t, !oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("a ")), "!isValidIdentifier(\"a \")")
	expect(t, !oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("a b")), "!isValidIdentifier(\"a b\")")
	expect(t, !oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("*42*")), "!isValidIdentifier(\"*42*\")")
	expect(t, !oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("a:b")), "!isValidIdentifier(\"a:b\")")
	expect(t, !oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("a\x00")), "!isValidIdentifier(\"a\\x00\")")
	expect(t, !oscParsersKittyClipboardProtocolIsValidIdentifier([]uint8("\xc3\xa9")), "!isValidIdentifier(\"\\xc3\\xa9\")")
}

// Zig: test "term51: parseIdentifier: valid input is returned unchanged"
func TestOSCParsersKittyClipboardProtocol_term51_parseIdentifier_valid_input_is_returned_unchanged(t *testing.T) {
	input := []uint8("abc-DEF_0.9+")
	result := zigUnwrap(t, oscParsersKittyClipboardProtocolParseIdentifier(input))
	expectEqualStrings(t, string(input), string(result))
	// It is the same slice, not a copy.
	expect(t, &result[0] == &input[0], "result.ptr == input.ptr")
	expectEqual(t, len(input), len(result))
}

// Zig: test "term51: parseIdentifier: invalid input is null"
func TestOSCParsersKittyClipboardProtocol_term51_parseIdentifier_invalid_input_is_null(t *testing.T) {
	expect(t, !oscParsersKittyClipboardProtocolParseIdentifier([]uint8("")).Valid, "parseIdentifier(\"\") == null")
	expect(t, !oscParsersKittyClipboardProtocolParseIdentifier([]uint8(" ")).Valid, "parseIdentifier(\" \") == null")
	expect(t, !oscParsersKittyClipboardProtocolParseIdentifier([]uint8("a b")).Valid, "parseIdentifier(\"a b\") == null")
	expect(t, !oscParsersKittyClipboardProtocolParseIdentifier([]uint8("a/b")).Valid, "parseIdentifier(\"a/b\") == null")
	expect(t, !oscParsersKittyClipboardProtocolParseIdentifier([]uint8("a=b")).Valid, "parseIdentifier(\"a=b\") == null")
	expect(t, !oscParsersKittyClipboardProtocolParseIdentifier([]uint8("a\n")).Valid, "parseIdentifier(\"a\\n\") == null")
	expect(t, !oscParsersKittyClipboardProtocolParseIdentifier([]uint8("\xff")).Valid, "parseIdentifier(\"\\xff\") == null")
}

// Zig: test "term51: Location.init: names are exact and case sensitive"
func TestOSCParsersKittyClipboardProtocol_term51_Location_init_names_are_exact_and_case_sensitive(t *testing.T) {
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolLocationPrimary), OSCParsersKittyClipboardProtocolLocationInit([]uint8("primary")))
	expect(t, !OSCParsersKittyClipboardProtocolLocationInit([]uint8("")).Valid, "Location.init(\"\") == null")
	expect(t, !OSCParsersKittyClipboardProtocolLocationInit([]uint8("Primary")).Valid, "Location.init(\"Primary\") == null")
	expect(t, !OSCParsersKittyClipboardProtocolLocationInit([]uint8("primary ")).Valid, "Location.init(\"primary \") == null")
	expect(t, !OSCParsersKittyClipboardProtocolLocationInit([]uint8("clipboard")).Valid, "Location.init(\"clipboard\") == null")
}

// Zig: test "term51: Operation.init: names are exact and case sensitive"
func TestOSCParsersKittyClipboardProtocol_term51_Operation_init_names_are_exact_and_case_sensitive(t *testing.T) {
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationRead), OSCParsersKittyClipboardProtocolOperationInit([]uint8("read")))
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationWalias), OSCParsersKittyClipboardProtocolOperationInit([]uint8("walias")))
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationWdata), OSCParsersKittyClipboardProtocolOperationInit([]uint8("wdata")))
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationWrite), OSCParsersKittyClipboardProtocolOperationInit([]uint8("write")))
	expect(t, !OSCParsersKittyClipboardProtocolOperationInit([]uint8("")).Valid, "Operation.init(\"\") == null")
	expect(t, !OSCParsersKittyClipboardProtocolOperationInit([]uint8("Read")).Valid, "Operation.init(\"Read\") == null")
	expect(t, !OSCParsersKittyClipboardProtocolOperationInit([]uint8("read ")).Valid, "Operation.init(\"read \") == null")
	expect(t, !OSCParsersKittyClipboardProtocolOperationInit([]uint8("writ")).Valid, "Operation.init(\"writ\") == null")
}

// Zig: test "term51: Status.init: names are exact and case sensitive"
func TestOSCParsersKittyClipboardProtocol_term51_Status_init_names_are_exact_and_case_sensitive(t *testing.T) {
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusDATA), OSCParsersKittyClipboardProtocolStatusInit([]uint8("DATA")))
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolStatusOK), OSCParsersKittyClipboardProtocolStatusInit([]uint8("OK")))
	expect(t, !OSCParsersKittyClipboardProtocolStatusInit([]uint8("")).Valid, "Status.init(\"\") == null")
	expect(t, !OSCParsersKittyClipboardProtocolStatusInit([]uint8("ok")).Valid, "Status.init(\"ok\") == null")
	expect(t, !OSCParsersKittyClipboardProtocolStatusInit([]uint8("OK ")).Valid, "Status.init(\"OK \") == null")
	expect(t, !OSCParsersKittyClipboardProtocolStatusInit([]uint8("EACCES")).Valid, "Status.init(\"EACCES\") == null")
}

// Zig: test "term51: OSC.readOption: reads from the metadata"
func TestOSCParsersKittyClipboardProtocol_term51_OSC_readOption_reads_from_the_metadata(t *testing.T) {
	osc := OSCParsersKittyClipboardProtocolOSC{
		Metadata:   []uint8("type=wdata:mime=text/plain"),
		Payload:    None[[]uint8](),
		Terminator: OSCTerminatorSt,
	}
	expectEqual(t, Some(OSCParsersKittyClipboardProtocolOperationWdata), osc.ReadOption(OSCParsersKittyClipboardProtocolOptionType).Type)
	expectEqualStrings(t, "text/plain", oscB_kcp_str(t, osc.ReadOption(OSCParsersKittyClipboardProtocolOptionMime).Mime))
	expect(t, !osc.ReadOption(OSCParsersKittyClipboardProtocolOptionId).Id.Valid, "osc.readOption(.id) == null")
	expect(t, !osc.ReadOption(OSCParsersKittyClipboardProtocolOptionStatus).Status.Valid, "osc.readOption(.status) == null")
}

// Zig: test "term51: parse: missing separator after 5522 is invalid"
func TestOSCParsersKittyClipboardProtocol_term51_parse_missing_separator_after_5522_is_invalid(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscB_kcp_feed(&p, "5522")
	expect(t, p.state == OSCParserState5522, "p.state == .@\"5522\"")
	expect(t, OSCParsersKittyClipboardProtocolParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
}

// Zig: test "term51: parse: metadata and payload split at the first semicolon"
func TestOSCParsersKittyClipboardProtocol_term51_parse_metadata_and_payload_split_at_the_first_semicolon(t *testing.T) {
	for _, c := range []struct {
		input    string
		metadata string
		payload  Optional[string]
	}{
		{"5522;", "", None[string]()},
		{"5522;;", "", Some("")},
		{"5522;type=read", "type=read", None[string]()},
		{"5522;type=read;", "type=read", Some("")},
		{"5522;type=read;YQ==", "type=read", Some("YQ==")},
		// later semicolons belong to the payload
		{"5522;a;b;c;", "a", Some("b;c;")},
		{"5522;;;", "", Some(";")},
		// neither part is validated
		{"5522;\x01\xff;\x00\x80", "\x01\xff", Some("\x00\x80")},
	} {
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kcp_feed(&p, c.input)
		cmd := oscB_kcp_deref(t, OSCParsersKittyClipboardProtocolParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyKittyClipboardProtocol, "cmd == .kitty_clipboard_protocol")
		expectEqualSlices(t, []uint8(c.metadata), cmd.KittyClipboardProtocol.Metadata)
		if !c.payload.Valid {
			expect(t, !cmd.KittyClipboardProtocol.Payload.Valid, "cmd.kitty_clipboard_protocol.payload == null")
		} else {
			expectEqualSlices(t, []uint8(c.payload.Value), zigUnwrap(t, cmd.KittyClipboardProtocol.Payload))
		}
	}
}

// Zig: test "term51: parse: terminator is recorded"
func TestOSCParsersKittyClipboardProtocol_term51_parse_terminator_is_recorded(t *testing.T) {
	for _, c := range []struct {
		ch   Optional[uint8]
		want OSCTerminator
	}{
		{Some[uint8](0x07), OSCTerminatorBel},
		{Some[uint8](0x1b), OSCTerminatorSt},
		{None[uint8](), OSCTerminatorSt},
		// Anything that is not BEL is treated as ST.
		{Some[uint8]('\\'), OSCTerminatorSt},
		{Some[uint8](0x9c), OSCTerminatorSt},
	} {
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kcp_feed(&p, "5522;type=read;x")
		cmd := oscB_kcp_deref(t, OSCParsersKittyClipboardProtocolParse(&p, c.ch))
		expect(t, cmd.KittyClipboardProtocol.Terminator == c.want, "cmd.kitty_clipboard_protocol.terminator == case[1]")
	}
}

// Zig: test "term51: parse: buffer limit without allocator"
func TestOSCParsersKittyClipboardProtocol_term51_parse_buffer_limit_without_allocator(t *testing.T) {
	// Without an allocator the fixed buffer is used. No NUL terminator is
	// needed, so the data may fill the whole buffer.
	prefix := "type=wdata;"
	{
		payload := strings.Repeat("A", OSCParserMAX_BUF-len(prefix))
		p := OSCParserInit(false)
		oscB_kcp_feed(&p, "5522;"+prefix+payload)
		expect(t, p.state == OSCParserState5522, "p.state == .@\"5522\"")
		cmd := oscB_kcp_deref(t, OSCParsersKittyClipboardProtocolParse(&p, None[uint8]()))
		expectEqualStrings(t, "type=wdata", string(cmd.KittyClipboardProtocol.Metadata))
		expectEqualStrings(t, payload, oscB_kcp_str(t, cmd.KittyClipboardProtocol.Payload))
	}
	{
		payload := strings.Repeat("A", OSCParserMAX_BUF-len(prefix)+1)
		p := OSCParserInit(false)
		oscB_kcp_feed(&p, "5522;"+prefix+payload)
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}
}

// Zig: test "term51: parse: no buffer limit with allocator"
func TestOSCParsersKittyClipboardProtocol_term51_parse_no_buffer_limit_with_allocator(t *testing.T) {
	prefix := "type=wdata;"
	for _, n := range []int{OSCParserMAX_BUF - len(prefix) + 1, OSCParserMAX_BUF, OSCParserMAX_BUF*4 + 1} {
		payload := strings.Repeat("A", n)
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kcp_feed(&p, "5522;"+prefix+payload)
		cmd := oscB_kcp_deref(t, OSCParsersKittyClipboardProtocolParse(&p, Some[uint8](0x07)))
		expectEqualStrings(t, "type=wdata", string(cmd.KittyClipboardProtocol.Metadata))
		expectEqualStrings(t, payload, oscB_kcp_str(t, cmd.KittyClipboardProtocol.Payload))
		expect(t, cmd.KittyClipboardProtocol.Terminator == OSCTerminatorBel, "cmd.kitty_clipboard_protocol.terminator == .bel")
	}
}
