// WIREFRAME for src/terminal/osc/parsers/kitty_clipboard_protocol.zig
// (Ghostty v1.3.1). NOT YET PORTED.
//
// Kitty's clipboard protocol (OSC 5522)
// Specification: https://sw.kovidgoyal.net/kitty/clipboard/
// https://rockorager.dev/misc/bracketed-paste-mime/
//
// Types and signatures only; see the header of color.go for the rules.
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
//     warn("kitty_clipboard_protocol", ...).

package terminal

// OSCParsersKittyClipboardProtocolOSC is
// osc.parsers.kitty_clipboard_protocol.OSC.
type OSCParsersKittyClipboardProtocolOSC struct {
	// The raw metadata that was received. It can be parsed by using the
	// `readOption` method.
	metadata []uint8
	// The raw payload. It may be Base64 encoded, check the `e` option.
	//
	// Zig: ?[]const u8.
	payload Optional[[]uint8]
	// The terminator that was used in case we need to send a response.
	terminator OSCTerminator
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
	panic(notImplemented)
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
func (self OSCParsersKittyClipboardProtocolLocation) String() string { panic(notImplemented) }

// OSCParsersKittyClipboardProtocolLocationInit is
// osc.parsers.kitty_clipboard_protocol.Location.init:
// std.meta.stringToEnum(Location, str).
func OSCParsersKittyClipboardProtocolLocationInit(str []uint8) Optional[OSCParsersKittyClipboardProtocolLocation] {
	panic(notImplemented)
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
func (self OSCParsersKittyClipboardProtocolOperation) String() string { panic(notImplemented) }

// OSCParsersKittyClipboardProtocolOperationInit is
// osc.parsers.kitty_clipboard_protocol.Operation.init:
// std.meta.stringToEnum(Operation, str).
func OSCParsersKittyClipboardProtocolOperationInit(str []uint8) Optional[OSCParsersKittyClipboardProtocolOperation] {
	panic(notImplemented)
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
func (self OSCParsersKittyClipboardProtocolStatus) String() string { panic(notImplemented) }

// OSCParsersKittyClipboardProtocolStatusInit is
// osc.parsers.kitty_clipboard_protocol.Status.init:
// std.meta.stringToEnum(Status, str).
func OSCParsersKittyClipboardProtocolStatusInit(str []uint8) Optional[OSCParsersKittyClipboardProtocolStatus] {
	panic(notImplemented)
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
func (self OSCParsersKittyClipboardProtocolOption) String() string { panic(notImplemented) }

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

// Read is osc.parsers.kitty_clipboard_protocol.Option.read: read the option
// value from the raw metadata string.
//
// Zig: `key` is a comptime parameter and the result type is `?key.Type()`.
// Go: the result is in the field of the returned struct that is named after
// `key`; see OSCParsersKittyClipboardProtocolOptionValue.
func (key OSCParsersKittyClipboardProtocolOption) Read(
	metadata []uint8,
) OSCParsersKittyClipboardProtocolOptionValue {
	panic(notImplemented)
}

// oscParsersKittyClipboardProtocolValidIdentifierCharacters is
// osc.parsers.kitty_clipboard_protocol.valid_identifier_characters:
// characters that are valid in identifiers.
const oscParsersKittyClipboardProtocolValidIdentifierCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_+."

// oscParsersKittyClipboardProtocolIsValidIdentifier is
// osc.parsers.kitty_clipboard_protocol.isValidIdentifier.
func oscParsersKittyClipboardProtocolIsValidIdentifier(str []uint8) bool {
	panic(notImplemented)
}

// oscParsersKittyClipboardProtocolParseIdentifier is
// osc.parsers.kitty_clipboard_protocol.parseIdentifier: str if it is a valid
// identifier, otherwise null.
func oscParsersKittyClipboardProtocolParseIdentifier(str []uint8) Optional[[]uint8] {
	panic(notImplemented)
}

// OSCParsersKittyClipboardProtocolParse is
// osc.parsers.kitty_clipboard_protocol.parse: parse OSC 5522.
//
// The result is nil for Zig's null; otherwise it points at the parser's own
// command.
func OSCParsersKittyClipboardProtocolParse(parser *OSCParser, terminator_ch Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}
