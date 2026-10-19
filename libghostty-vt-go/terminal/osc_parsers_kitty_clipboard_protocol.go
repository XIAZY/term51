// Ported from src/terminal/osc/parsers/kitty_clipboard_protocol.zig
// (Ghostty v1.3.1).
//
// Kitty's clipboard protocol (OSC 5522)
// Specification: https://sw.kovidgoyal.net/kitty/clipboard/
// https://rockorager.dev/misc/bracketed-paste-mime/
//
// File naming: `osc/parsers/kitty_clipboard_protocol.zig` becomes
// `osc_parsers_kitty_clipboard_protocol.go` in package terminal, and its
// declarations take the prefix `OSCParsersKittyClipboardProtocol`
// (`oscParsersKittyClipboardProtocol` for private ones). The file has no C
// ABI declarations.
//
// Not ported:
//   - `Option.Type`, a comptime function that returns a type. Go has no such
//     thing; its counterpart is the struct
//     OSCParsersKittyClipboardProtocolOptionValue, which holds one result
//     field per Option value.
//   - `const log = std.log.scoped(.kitty_clipboard_protocol)`: the port uses
//     warn("kitty_clipboard_protocol", ...). Upstream never logs in this
//     file.
//   - The imports `build_options` and `encoding`, unused upstream.

package terminal

import "bytes"

// OSCParsersKittyClipboardProtocolOSC is
// osc.parsers.kitty_clipboard_protocol.OSC.
type OSCParsersKittyClipboardProtocolOSC struct {
	// The raw metadata that was received. It can be parsed by using the
	// `readOption` method.
	Metadata []uint8
	// The raw payload. It may be Base64 encoded, check the `e` option.
	//
	// Zig: ?[]const u8.
	Payload Optional[[]uint8]
	// The terminator that was used in case we need to send a response.
	Terminator OSCTerminator
}

// ReadOption is osc.parsers.kitty_clipboard_protocol.OSC.readOption: decode
// an option from the metadata.
//
// Zig: `key` is a comptime parameter and the result type is `?key.Type()`.
// Go: the result is in the field of the returned struct that is named after
// `key`; see OSCParsersKittyClipboardProtocolOptionValue.
func (self OSCParsersKittyClipboardProtocolOSC) ReadOption(
	key OSCParsersKittyClipboardProtocolOption,
) OSCParsersKittyClipboardProtocolOptionValue {
	return key.Read(self.Metadata)
}

// OSCParsersKittyClipboardProtocolLocation is
// osc.parsers.kitty_clipboard_protocol.Location.
type OSCParsersKittyClipboardProtocolLocation uint8

const (
	OSCParsersKittyClipboardProtocolLocationPrimary OSCParsersKittyClipboardProtocolLocation = iota

	oscParsersKittyClipboardProtocolLocationCount = iota
)

// String returns the Zig tag name (@tagName), the name that
// std.meta.stringToEnum matches in Location.init.
func (self OSCParsersKittyClipboardProtocolLocation) String() string {
	return oscParsersKittyClipboardProtocolLocationNames[self]
}

var oscParsersKittyClipboardProtocolLocationNames = [oscParsersKittyClipboardProtocolLocationCount]string{
	OSCParsersKittyClipboardProtocolLocationPrimary: "primary",
}

// OSCParsersKittyClipboardProtocolLocationInit is
// osc.parsers.kitty_clipboard_protocol.Location.init:
// std.meta.stringToEnum(Location, str).
func OSCParsersKittyClipboardProtocolLocationInit(str []uint8) Optional[OSCParsersKittyClipboardProtocolLocation] {
	// std.meta.stringToEnum: the value whose tag name equals str exactly.
	for i := OSCParsersKittyClipboardProtocolLocation(0); i < oscParsersKittyClipboardProtocolLocationCount; i++ {
		if string(str) == i.String() {
			return Some(i)
		}
	}
	return None[OSCParsersKittyClipboardProtocolLocation]()
}

// OSCParsersKittyClipboardProtocolOperation is
// osc.parsers.kitty_clipboard_protocol.Operation.
type OSCParsersKittyClipboardProtocolOperation uint8

const (
	OSCParsersKittyClipboardProtocolOperationRead OSCParsersKittyClipboardProtocolOperation = iota
	OSCParsersKittyClipboardProtocolOperationWalias
	OSCParsersKittyClipboardProtocolOperationWdata
	OSCParsersKittyClipboardProtocolOperationWrite

	oscParsersKittyClipboardProtocolOperationCount = iota
)

// String returns the Zig tag name (@tagName), the name that
// std.meta.stringToEnum matches in Operation.init.
func (self OSCParsersKittyClipboardProtocolOperation) String() string {
	return oscParsersKittyClipboardProtocolOperationNames[self]
}

var oscParsersKittyClipboardProtocolOperationNames = [oscParsersKittyClipboardProtocolOperationCount]string{
	OSCParsersKittyClipboardProtocolOperationRead:   "read",
	OSCParsersKittyClipboardProtocolOperationWalias: "walias",
	OSCParsersKittyClipboardProtocolOperationWdata:  "wdata",
	OSCParsersKittyClipboardProtocolOperationWrite:  "write",
}

// OSCParsersKittyClipboardProtocolOperationInit is
// osc.parsers.kitty_clipboard_protocol.Operation.init:
// std.meta.stringToEnum(Operation, str).
func OSCParsersKittyClipboardProtocolOperationInit(str []uint8) Optional[OSCParsersKittyClipboardProtocolOperation] {
	// std.meta.stringToEnum: the value whose tag name equals str exactly.
	for i := OSCParsersKittyClipboardProtocolOperation(0); i < oscParsersKittyClipboardProtocolOperationCount; i++ {
		if string(str) == i.String() {
			return Some(i)
		}
	}
	return None[OSCParsersKittyClipboardProtocolOperation]()
}

// OSCParsersKittyClipboardProtocolStatus is
// osc.parsers.kitty_clipboard_protocol.Status. The Zig value names are upper
// case (they are the strings of the protocol) and are kept as they are.
type OSCParsersKittyClipboardProtocolStatus uint8

const (
	OSCParsersKittyClipboardProtocolStatusDATA OSCParsersKittyClipboardProtocolStatus = iota
	OSCParsersKittyClipboardProtocolStatusDONE
	OSCParsersKittyClipboardProtocolStatusEBUSY
	OSCParsersKittyClipboardProtocolStatusEINVAL
	OSCParsersKittyClipboardProtocolStatusEIO
	OSCParsersKittyClipboardProtocolStatusENOSYS
	OSCParsersKittyClipboardProtocolStatusEPERM
	OSCParsersKittyClipboardProtocolStatusOK

	oscParsersKittyClipboardProtocolStatusCount = iota
)

// String returns the Zig tag name (@tagName), the name that
// std.meta.stringToEnum matches in Status.init.
func (self OSCParsersKittyClipboardProtocolStatus) String() string {
	return oscParsersKittyClipboardProtocolStatusNames[self]
}

var oscParsersKittyClipboardProtocolStatusNames = [oscParsersKittyClipboardProtocolStatusCount]string{
	OSCParsersKittyClipboardProtocolStatusDATA:   "DATA",
	OSCParsersKittyClipboardProtocolStatusDONE:   "DONE",
	OSCParsersKittyClipboardProtocolStatusEBUSY:  "EBUSY",
	OSCParsersKittyClipboardProtocolStatusEINVAL: "EINVAL",
	OSCParsersKittyClipboardProtocolStatusEIO:    "EIO",
	OSCParsersKittyClipboardProtocolStatusENOSYS: "ENOSYS",
	OSCParsersKittyClipboardProtocolStatusEPERM:  "EPERM",
	OSCParsersKittyClipboardProtocolStatusOK:     "OK",
}

// OSCParsersKittyClipboardProtocolStatusInit is
// osc.parsers.kitty_clipboard_protocol.Status.init:
// std.meta.stringToEnum(Status, str).
func OSCParsersKittyClipboardProtocolStatusInit(str []uint8) Optional[OSCParsersKittyClipboardProtocolStatus] {
	// std.meta.stringToEnum: the value whose tag name equals str exactly.
	for i := OSCParsersKittyClipboardProtocolStatus(0); i < oscParsersKittyClipboardProtocolStatusCount; i++ {
		if string(str) == i.String() {
			return Some(i)
		}
	}
	return None[OSCParsersKittyClipboardProtocolStatus]()
}

// OSCParsersKittyClipboardProtocolOption is
// osc.parsers.kitty_clipboard_protocol.Option.
type OSCParsersKittyClipboardProtocolOption uint8

const (
	OSCParsersKittyClipboardProtocolOptionId OSCParsersKittyClipboardProtocolOption = iota
	OSCParsersKittyClipboardProtocolOptionLoc
	OSCParsersKittyClipboardProtocolOptionMime
	OSCParsersKittyClipboardProtocolOptionName
	OSCParsersKittyClipboardProtocolOptionPassword
	OSCParsersKittyClipboardProtocolOptionPw
	OSCParsersKittyClipboardProtocolOptionStatus
	OSCParsersKittyClipboardProtocolOptionType

	oscParsersKittyClipboardProtocolOptionCount = iota
)

// String returns the Zig tag name (@tagName), which Option.read uses as the
// key to look for in the metadata.
func (self OSCParsersKittyClipboardProtocolOption) String() string {
	return oscParsersKittyClipboardProtocolOptionNames[self]
}

var oscParsersKittyClipboardProtocolOptionNames = [oscParsersKittyClipboardProtocolOptionCount]string{
	OSCParsersKittyClipboardProtocolOptionId:       "id",
	OSCParsersKittyClipboardProtocolOptionLoc:      "loc",
	OSCParsersKittyClipboardProtocolOptionMime:     "mime",
	OSCParsersKittyClipboardProtocolOptionName:     "name",
	OSCParsersKittyClipboardProtocolOptionPassword: "password",
	OSCParsersKittyClipboardProtocolOptionPw:       "pw",
	OSCParsersKittyClipboardProtocolOptionStatus:   "status",
	OSCParsersKittyClipboardProtocolOptionType:     "type",
}

// OSCParsersKittyClipboardProtocolOptionValue is the Go counterpart of
// osc.parsers.kitty_clipboard_protocol.Option.Type, the comptime function
// that gives the type `read` and `readOption` return (as an optional) for
// each option:
//
//	.id, .mime, .name, .password, .pw  []const u8
//	.loc                               Location
//	.status                            Status
//	.type                              Operation
//
// Go cannot make a result type depend on an argument value, so `Read` and
// `ReadOption` return this struct. It works like a tagged union whose tag is
// the Option that was asked for: only the field named after that Option is
// meaningful, and it is the null Optional where Zig returns null.
type OSCParsersKittyClipboardProtocolOptionValue struct {
	Id       Optional[[]uint8]
	Loc      Optional[OSCParsersKittyClipboardProtocolLocation]
	Mime     Optional[[]uint8]
	Name     Optional[[]uint8]
	Password Optional[[]uint8]
	Pw       Optional[[]uint8]
	Status   Optional[OSCParsersKittyClipboardProtocolStatus]
	Type     Optional[OSCParsersKittyClipboardProtocolOperation]
}

// oscParsersKittyClipboardProtocolIsWhitespace is std.ascii.isWhitespace:
// ' ' and '\t'...'\r'.
func oscParsersKittyClipboardProtocolIsWhitespace(c uint8) bool {
	return c == ' ' || (c >= '\t' && c <= '\r')
}

// oscParsersKittyClipboardProtocolAsciiWhitespace is std.ascii.whitespace,
// as a cutset for bytes.Trim (std.mem.trim). Every byte is ASCII, so
// bytes.Trim compares byte by byte.
const oscParsersKittyClipboardProtocolAsciiWhitespace = " \t\n\r\x0b\x0c"

// Read is osc.parsers.kitty_clipboard_protocol.Option.read: read the option
// value from the raw metadata string.
//
// Zig: `key` is a comptime parameter and the result type is `?key.Type()`.
// Go: the result is in the field of the returned struct that is named after
// `key`; see OSCParsersKittyClipboardProtocolOptionValue.
func (key OSCParsersKittyClipboardProtocolOption) Read(
	metadata []uint8,
) OSCParsersKittyClipboardProtocolOptionValue {
	var result OSCParsersKittyClipboardProtocolOptionValue

	// Zig: a labeled block `value:` that breaks with the value or returns
	// null from the function.
	var value []uint8
	found := false
	pos := 0
	for pos < len(metadata) {
		// skip any whitespace
		for pos < len(metadata) && oscParsersKittyClipboardProtocolIsWhitespace(metadata[pos]) {
			pos += 1
		}
		// bail if we are out of metadata
		if pos >= len(metadata) {
			return result
		}
		if !bytes.HasPrefix(metadata[pos:], []uint8(key.String())) {
			// this isn't the key we are looking for, skip to the next option, or bail if
			// there is no next option
			//
			// Zig: std.mem.indexOfScalarPos(u8, metadata, pos, ':').
			idx := bytes.IndexByte(metadata[pos:], ':')
			if idx < 0 {
				return result
			}
			pos = pos + idx
			pos += 1
			continue
		}
		// skip past the key
		pos += len(key.String())
		// skip any whitespace
		for pos < len(metadata) && oscParsersKittyClipboardProtocolIsWhitespace(metadata[pos]) {
			pos += 1
		}
		// bail if we are out of metadata
		if pos >= len(metadata) {
			return result
		}
		// a valid option has an '='
		if metadata[pos] != '=' {
			return result
		}
		// the end of the value is bounded by a ':' or the end of the metadata
		end := len(metadata)
		if idx := bytes.IndexByte(metadata[pos:], ':'); idx >= 0 {
			end = pos + idx
		}
		start := pos + 1
		// strip any leading or trailing whitespace
		value = bytes.Trim(metadata[start:end], oscParsersKittyClipboardProtocolAsciiWhitespace)
		found = true
		break
	}
	if !found {
		// the key was not found
		return result
	}

	// return the parsed value
	switch key {
	case OSCParsersKittyClipboardProtocolOptionId:
		result.Id = oscParsersKittyClipboardProtocolParseIdentifier(value)
	case OSCParsersKittyClipboardProtocolOptionLoc:
		result.Loc = OSCParsersKittyClipboardProtocolLocationInit(value)
	case OSCParsersKittyClipboardProtocolOptionMime:
		result.Mime = Some(value)
	case OSCParsersKittyClipboardProtocolOptionName:
		result.Name = Some(value)
	case OSCParsersKittyClipboardProtocolOptionPassword:
		result.Password = Some(value)
	case OSCParsersKittyClipboardProtocolOptionPw:
		result.Pw = Some(value)
	case OSCParsersKittyClipboardProtocolOptionStatus:
		result.Status = OSCParsersKittyClipboardProtocolStatusInit(value)
	case OSCParsersKittyClipboardProtocolOptionType:
		result.Type = OSCParsersKittyClipboardProtocolOperationInit(value)
	default:
		// Zig's switch is exhaustive over a comptime Option.
		panic("unreachable")
	}
	return result
}

// oscParsersKittyClipboardProtocolValidIdentifierCharacters is
// osc.parsers.kitty_clipboard_protocol.valid_identifier_characters:
// characters that are valid in identifiers.
const oscParsersKittyClipboardProtocolValidIdentifierCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_+."

// oscParsersKittyClipboardProtocolIsValidIdentifier is
// osc.parsers.kitty_clipboard_protocol.isValidIdentifier.
func oscParsersKittyClipboardProtocolIsValidIdentifier(str []uint8) bool {
	if len(str) == 0 {
		return false
	}
	// Zig: std.mem.indexOfNone(u8, str, valid_identifier_characters) == null.
	for _, c := range str {
		if bytes.IndexByte([]uint8(oscParsersKittyClipboardProtocolValidIdentifierCharacters), c) < 0 {
			return false
		}
	}
	return true
}

// oscParsersKittyClipboardProtocolParseIdentifier is
// osc.parsers.kitty_clipboard_protocol.parseIdentifier: str if it is a valid
// identifier, otherwise null.
func oscParsersKittyClipboardProtocolParseIdentifier(str []uint8) Optional[[]uint8] {
	if oscParsersKittyClipboardProtocolIsValidIdentifier(str) {
		return Some(str)
	}
	return None[[]uint8]()
}

// OSCParsersKittyClipboardProtocolParse is
// osc.parsers.kitty_clipboard_protocol.parse: parse OSC 5522.
//
// The result is nil for Zig's null; otherwise it points at the parser's own
// command.
func OSCParsersKittyClipboardProtocolParse(parser *OSCParser, terminator_ch Optional[uint8]) *OSCCommand {
	assert(parser.state == OSCParserState5522)

	writer := parser.writer
	if writer == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}

	data := writer.buffered()

	var metadata []uint8
	var payload Optional[[]uint8]
	if start := bytes.IndexByte(data, ';'); start < 0 {
		metadata, payload = data, None[[]uint8]()
	} else {
		metadata, payload = data[0:start], Some(data[start+1:len(data)])
	}

	parser.command = OSCCommand{
		Tag: OSCCommandKeyKittyClipboardProtocol,
		KittyClipboardProtocol: OSCParsersKittyClipboardProtocolOSC{
			Metadata:   metadata,
			Payload:    payload,
			Terminator: OSCTerminatorInit(terminator_ch),
		},
	}

	return &parser.command
}
