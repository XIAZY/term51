// NOT YET PORTED. Placeholder for src/terminal/osc.zig.
//
// Parser.zig embeds an osc.Parser and returns osc.Command values, so the
// parser cannot compile without these names. This file declares only the
// surface Parser.go touches, with an OSC parser that recognises nothing.
// It is replaced by the real port of osc.zig when that file is signed off;
// until then every Parser test that involves an OSC sequence is pending
// (docs/port-status.md).

package terminal

// OSCCommand is osc.Command. Placeholder.
type OSCCommand struct{}

func (self OSCCommand) zigFmtAny() string {
	return "<osc command: not yet ported>"
}

// OSCParser is osc.Parser. Placeholder.
type OSCParser struct{}

// OSCParserInit is osc.Parser.init. Placeholder.
func OSCParserInit(alloc any) OSCParser {
	return OSCParser{}
}

// Deinit is osc.Parser.deinit. Placeholder.
func (self *OSCParser) Deinit() {}

// Reset is osc.Parser.reset. Placeholder.
func (self *OSCParser) Reset() {}

// Next is osc.Parser.next. Placeholder.
func (self *OSCParser) Next(c uint8) {}

// End is osc.Parser.end. Placeholder: never yields a command.
func (self *OSCParser) End(terminator_ch Optional[uint8]) *OSCCommand {
	return nil
}
