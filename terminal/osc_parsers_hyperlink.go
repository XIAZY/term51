// Ported from src/terminal/osc/parsers/hyperlink.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/hyperlink.zig` becomes
// `osc_parsers_hyperlink.go` in package terminal, and its
// declarations take the prefix `OSCParsersHyperlink`. The file has no C ABI
// declarations.
//
// `const log = std.log.scoped(.osc_hyperlink)` has no declaration in Go: the
// port uses warn("osc_hyperlink", ...).
//
// std.mem.indexOfScalar is bytes.IndexByte (-1 is null), std.mem.eql is
// bytes.Equal; std.mem.indexOfScalarPos and std.mem.replaceScalar are
// written out where they are used.

package terminal

import "bytes"

// OSCParsersHyperlinkParse is osc.parsers.hyperlink.parse: parse OSC 8 hyperlinks.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersHyperlinkParse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
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
	s := bytes.IndexByte(data, ';')
	if s < 0 {
		parser.state = OSCParserStateInvalid
		return nil
	}

	// Zig: .uri = data[s + 1 .. data.len - 1 :0]; id keeps its default,
	// null. The Go slice keeps the NUL behind it in its capacity.
	parser.command = OSCCommand{
		Tag: OSCCommandKeyHyperlinkStart,
		HyperlinkStart: OSCCommandHyperlinkStartPayload{
			uri: data[s+1 : len(data)-1],
		},
	}

	data[s] = 0
	kvs := data[0 : s+1]
	// Zig: std.mem.replaceScalar(u8, kvs, ':', 0).
	for i := range kvs {
		if kvs[i] == ':' {
			kvs[i] = 0
		}
	}
	kv_start := 0
	for kv_start < len(kvs) {
		// Zig: std.mem.indexOfScalarPos(u8, kvs, kv_start + 1, 0) orelse
		// break. A start at or past the end finds nothing.
		if kv_start+1 >= len(kvs) {
			break
		}
		kv_end := bytes.IndexByte(kvs[kv_start+1:], 0)
		if kv_end < 0 {
			break
		}
		kv_end += kv_start + 1
		kv := data[kv_start : kv_end+1]
		v := bytes.IndexByte(kv, '=')
		if v < 0 {
			break
		}
		key := kv[0:v]
		// Zig: kv[v + 1 .. kv.len - 1 :0].
		value := kv[v+1 : len(kv)-1]
		if bytes.Equal(key, []uint8("id")) {
			if len(value) > 0 {
				parser.command.HyperlinkStart.id = Some(value)
			}
		} else {
			warn("osc_hyperlink", "unknown hyperlink option: '%s'", key)
		}
		kv_start = kv_end + 1
	}

	if len(parser.command.HyperlinkStart.uri) == 0 {
		if parser.command.HyperlinkStart.id.Valid {
			parser.state = OSCParserStateInvalid
			return nil
		}
		parser.command = OSCCommand{Tag: OSCCommandKeyHyperlinkEnd}
	}

	return &parser.command
}
