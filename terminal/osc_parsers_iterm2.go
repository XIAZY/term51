// Ported from src/terminal/osc/parsers/iterm2.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/iterm2.zig` becomes `osc_parsers_iterm2.go` in
// package terminal, and its declarations take the prefix `OSCParsersIterm2`
// (`oscParsersIterm2` for private ones). The file has no C ABI declarations.
//
// `const log = std.log.scoped(.osc_iterm2)` has no declaration in Go: the
// port uses warn("osc_iterm2", ...); the file's one log line is a
// `log.debug`, which the port does not print. The `simd` import is unused
// upstream.

package terminal

import "bytes"

// oscParsersIterm2Key is osc.parsers.iterm2.Key (private in Zig): the keys
// of OSC 1337. The Zig value names are already CamelCase and are kept.
type oscParsersIterm2Key uint8

const (
	oscParsersIterm2KeyAddAnnotation oscParsersIterm2Key = iota
	oscParsersIterm2KeyAddHiddenAnnotation
	oscParsersIterm2KeyBlock
	oscParsersIterm2KeyButton
	oscParsersIterm2KeyClearCapturedOutput
	oscParsersIterm2KeyClearScrollback
	oscParsersIterm2KeyCopy
	oscParsersIterm2KeyCopyToClipboard
	oscParsersIterm2KeyCurrentDir
	oscParsersIterm2KeyCursorShape
	oscParsersIterm2KeyCustom
	oscParsersIterm2KeyDisinter
	oscParsersIterm2KeyEndCopy
	oscParsersIterm2KeyFile
	oscParsersIterm2KeyFileEnd
	oscParsersIterm2KeyFilePart
	oscParsersIterm2KeyHighlightCursorLine
	oscParsersIterm2KeyMultipartFile
	oscParsersIterm2KeyOpenURL
	oscParsersIterm2KeyPopKeyLabels
	oscParsersIterm2KeyPushKeyLabels
	oscParsersIterm2KeyRemoteHost
	oscParsersIterm2KeyReportCellSize
	oscParsersIterm2KeyReportVariable
	oscParsersIterm2KeyRequestAttention
	oscParsersIterm2KeyRequestUpload
	oscParsersIterm2KeySetBackgroundImageFile
	oscParsersIterm2KeySetBadgeFormat
	oscParsersIterm2KeySetColors
	oscParsersIterm2KeySetKeyLabel
	oscParsersIterm2KeySetMark
	oscParsersIterm2KeySetProfile
	oscParsersIterm2KeySetUserVar
	oscParsersIterm2KeyShellIntegrationVersion
	oscParsersIterm2KeyStealFocus
	oscParsersIterm2KeyUnicodeVersion

	oscParsersIterm2KeyCount = iota
)

// String returns the Zig tag name (@tagName; the "{t}" format in the
// "unimplemented OSC 1337" log line, and the names the map is built from).
func (self oscParsersIterm2Key) String() string {
	return oscParsersIterm2KeyNames[self]
}

var oscParsersIterm2KeyNames = [oscParsersIterm2KeyCount]string{
	oscParsersIterm2KeyAddAnnotation:           "AddAnnotation",
	oscParsersIterm2KeyAddHiddenAnnotation:     "AddHiddenAnnotation",
	oscParsersIterm2KeyBlock:                   "Block",
	oscParsersIterm2KeyButton:                  "Button",
	oscParsersIterm2KeyClearCapturedOutput:     "ClearCapturedOutput",
	oscParsersIterm2KeyClearScrollback:         "ClearScrollback",
	oscParsersIterm2KeyCopy:                    "Copy",
	oscParsersIterm2KeyCopyToClipboard:         "CopyToClipboard",
	oscParsersIterm2KeyCurrentDir:              "CurrentDir",
	oscParsersIterm2KeyCursorShape:             "CursorShape",
	oscParsersIterm2KeyCustom:                  "Custom",
	oscParsersIterm2KeyDisinter:                "Disinter",
	oscParsersIterm2KeyEndCopy:                 "EndCopy",
	oscParsersIterm2KeyFile:                    "File",
	oscParsersIterm2KeyFileEnd:                 "FileEnd",
	oscParsersIterm2KeyFilePart:                "FilePart",
	oscParsersIterm2KeyHighlightCursorLine:     "HighlightCursorLine",
	oscParsersIterm2KeyMultipartFile:           "MultipartFile",
	oscParsersIterm2KeyOpenURL:                 "OpenURL",
	oscParsersIterm2KeyPopKeyLabels:            "PopKeyLabels",
	oscParsersIterm2KeyPushKeyLabels:           "PushKeyLabels",
	oscParsersIterm2KeyRemoteHost:              "RemoteHost",
	oscParsersIterm2KeyReportCellSize:          "ReportCellSize",
	oscParsersIterm2KeyReportVariable:          "ReportVariable",
	oscParsersIterm2KeyRequestAttention:        "RequestAttention",
	oscParsersIterm2KeyRequestUpload:           "RequestUpload",
	oscParsersIterm2KeySetBackgroundImageFile:  "SetBackgroundImageFile",
	oscParsersIterm2KeySetBadgeFormat:          "SetBadgeFormat",
	oscParsersIterm2KeySetColors:               "SetColors",
	oscParsersIterm2KeySetKeyLabel:             "SetKeyLabel",
	oscParsersIterm2KeySetMark:                 "SetMark",
	oscParsersIterm2KeySetProfile:              "SetProfile",
	oscParsersIterm2KeySetUserVar:              "SetUserVar",
	oscParsersIterm2KeyShellIntegrationVersion: "ShellIntegrationVersion",
	oscParsersIterm2KeyStealFocus:              "StealFocus",
	oscParsersIterm2KeyUnicodeVersion:          "UnicodeVersion",
}

// oscParsersIterm2Map is osc.parsers.iterm2.Map (private in Zig):
// std.StaticStringMapWithEql(Key, std.ascii.eqlIgnoreCase).
//
// Instead of using `std.meta.stringToEnum` we set up a StaticStringMap so
// that we can get ASCII case-insensitive lookups.
type oscParsersIterm2Map struct {
	kvs map[string]oscParsersIterm2Key
}

// get is StaticStringMap.get: the value for key, or none. Lookup ignores
// ASCII case and is exact otherwise.
//
// Port: kvs is keyed by the ASCII-lowercased name, so one lookup of the
// lowercased key finds what Zig's length-indexed scan with eqlIgnoreCase
// finds. The two agree because no two names of Key differ only in case
// (oscParsersIterm2MapInitComptime checks this).
func (self oscParsersIterm2Map) get(key []uint8) Optional[oscParsersIterm2Key] {
	if v, ok := self.kvs[oscParsersIterm2AsciiLower(key)]; ok {
		return Some(v)
	}
	return None[oscParsersIterm2Key]()
}

// oscParsersIterm2AsciiLower is std.ascii.toLower applied to every byte of
// s: only 'A'...'Z' change, every other byte (including non-ASCII ones) is
// kept. The comparison std.ascii.eqlIgnoreCase makes.
func oscParsersIterm2AsciiLower(s []uint8) string {
	lower := make([]uint8, len(s))
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			c |= 0x20
		}
		lower[i] = c
	}
	return string(lower)
}

// oscParsersIterm2MapInitComptime is Map.initComptime with the block that
// builds the list of pairs: one pair per field of Key, (field.name,
// @enumFromInt(field.value)), in Key order.
func oscParsersIterm2MapInitComptime() oscParsersIterm2Map {
	result := oscParsersIterm2Map{kvs: make(map[string]oscParsersIterm2Key, oscParsersIterm2KeyCount)}
	for k := oscParsersIterm2Key(0); k < oscParsersIterm2KeyCount; k++ {
		name := oscParsersIterm2AsciiLower([]uint8(k.String()))
		_, dup := result.kvs[name]
		assert(!dup)
		result.kvs[name] = k
	}
	return result
}

// oscParsersIterm2Map_ is osc.parsers.iterm2.map (private in Zig): one entry
// per Key value, keyed by the value's name. The trailing underscore is there
// only because Go cannot tell `map` from `Map` once both carry the file
// prefix.
//
// Zig builds it at compile time from the fields of Key; Go fills it once
// at start-up, in Key order.
var oscParsersIterm2Map_ oscParsersIterm2Map = oscParsersIterm2MapInitComptime()

// OSCParsersIterm2Parse is osc.parsers.iterm2.parse: parse OSC 1337.
// https://iterm2.com/documentation-escape-codes.html
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersIterm2Parse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	assert(parser.state == OSCParserState1337)

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

	// Zig: key_str is [:0]u8 and value_ is ?[:0]u8. Each Go slice ends right
	// before a NUL byte of the buffer (the one written above, or the '='
	// overwritten below), which stays in the slice's capacity.
	var key_str []uint8
	var value_ Optional[[]uint8]
	if index := bytes.IndexByte(data, '='); index < 0 {
		key_str, value_ = data[0:len(data)-1], None[[]uint8]()
	} else {
		data[index] = 0
		key_str, value_ = data[0:index], Some(data[index+1:len(data)-1])
	}

	key_ := oscParsersIterm2Map_.get(key_str)
	if !key_.Valid {
		parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
		return nil
	}
	key := key_.Value

	switch key {
	case oscParsersIterm2KeyCopy:
		if !value_.Valid {
			parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
			return nil
		}
		value := value_.Value

		// Sending a blank entry to clear the clipboard is an OSC 52-ism,
		// make sure that is invalid here.
		if len(value) == 0 {
			parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
			return nil
		}

		// base64 value must be prefixed by a colon
		if value[0] != ':' {
			parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
			return nil
		}

		value = value[1:len(value)]

		// Sending a blank entry to clear the clipboard is an OSC 52-ism,
		// make sure that is invalid here.
		if len(value) == 0 {
			parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
			return nil
		}

		// Sending a '?' to query the clipboard is an OSC 52-ism, make sure
		// that is invalid here.
		if len(value) == 1 && value[0] == '?' {
			parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
			return nil
		}

		// It would be better to check for valid base64 data here, but that
		// would mean parsing the base64 data twice in the "normal" case.

		parser.command = OSCCommand{
			Tag: OSCCommandKeyClipboardContents,
			ClipboardContents: OSCCommandClipboardContentsPayload{
				kind: 'c',
				data: value,
			},
		}
		return &parser.command

	case oscParsersIterm2KeyCurrentDir:
		if !value_.Valid {
			parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
			return nil
		}
		value := value_.Value
		if len(value) == 0 {
			parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
			return nil
		}
		parser.command = OSCCommand{
			Tag: OSCCommandKeyReportPwd,
			ReportPwd: OSCCommandReportPwdPayload{
				value: value,
			},
		}
		return &parser.command

	case oscParsersIterm2KeyAddAnnotation,
		oscParsersIterm2KeyAddHiddenAnnotation,
		oscParsersIterm2KeyBlock,
		oscParsersIterm2KeyButton,
		oscParsersIterm2KeyClearCapturedOutput,
		oscParsersIterm2KeyClearScrollback,
		oscParsersIterm2KeyCopyToClipboard,
		oscParsersIterm2KeyCursorShape,
		oscParsersIterm2KeyCustom,
		oscParsersIterm2KeyDisinter,
		oscParsersIterm2KeyEndCopy,
		oscParsersIterm2KeyFile,
		oscParsersIterm2KeyFileEnd,
		oscParsersIterm2KeyFilePart,
		oscParsersIterm2KeyHighlightCursorLine,
		oscParsersIterm2KeyMultipartFile,
		oscParsersIterm2KeyOpenURL,
		oscParsersIterm2KeyPopKeyLabels,
		oscParsersIterm2KeyPushKeyLabels,
		oscParsersIterm2KeyRemoteHost,
		oscParsersIterm2KeyReportCellSize,
		oscParsersIterm2KeyReportVariable,
		oscParsersIterm2KeyRequestAttention,
		oscParsersIterm2KeyRequestUpload,
		oscParsersIterm2KeySetBackgroundImageFile,
		oscParsersIterm2KeySetBadgeFormat,
		oscParsersIterm2KeySetColors,
		oscParsersIterm2KeySetKeyLabel,
		oscParsersIterm2KeySetMark,
		oscParsersIterm2KeySetProfile,
		oscParsersIterm2KeySetUserVar,
		oscParsersIterm2KeyShellIntegrationVersion,
		oscParsersIterm2KeyStealFocus,
		oscParsersIterm2KeyUnicodeVersion:
		// Zig: log.debug("unimplemented OSC 1337: {t}", .{key}); the port
		// does not print debug logs.
		parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
		return nil
	}
	return &parser.command
}
