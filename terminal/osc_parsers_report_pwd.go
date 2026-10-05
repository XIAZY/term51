// Ported from src/terminal/osc/parsers/report_pwd.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/report_pwd.zig` becomes
// `osc_parsers_report_pwd.go` in package terminal, and its
// declarations take the prefix `OSCParsersReportPwd`. The file has no C ABI
// declarations.

package terminal

// OSCParsersReportPwdParse is osc.parsers.report_pwd.parse: parse OSC 7.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersReportPwdParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	writer := parser.writer
	if writer == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	if err := writer.writeByte(0); err != nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	data := writer.buffered()
	// Zig: data[0 .. data.len - 1 :0]. The Go slice keeps the NUL behind it
	// in its capacity.
	parser.command = OSCCommand{
		Tag: OSCCommandKeyReportPwd,
		ReportPwd: OSCCommandReportPwdPayload{
			value: data[0 : len(data)-1],
		},
	}
	return &parser.command
}
