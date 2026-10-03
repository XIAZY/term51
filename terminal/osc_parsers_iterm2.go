// WIREFRAME for src/terminal/osc/parsers/iterm2.zig (Ghostty v1.3.1).
// NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: `osc/parsers/iterm2.zig` becomes `osc_parsers_iterm2.go` in
// package terminal, and its declarations take the prefix `OSCParsersIterm2`
// (`oscParsersIterm2` for private ones). The file has no C ABI declarations.
//
// `const log = std.log.scoped(.osc_iterm2)` has no declaration in Go: the
// port uses warn("osc_iterm2", ...). The `simd` import is unused upstream.

package terminal

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
func (self oscParsersIterm2Key) String() string { panic(notImplemented) }

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
func (self oscParsersIterm2Map) get(key []uint8) Optional[oscParsersIterm2Key] {
	panic(notImplemented)
}

// oscParsersIterm2Map_ is osc.parsers.iterm2.map (private in Zig): one entry
// per Key value, keyed by the value's name. The trailing underscore is there
// only because Go cannot tell `map` from `Map` once both carry the file
// prefix.
//
// Zig builds it at compile time from the fields of Key. Wireframe: left
// zero; the implementation fills it once at start-up, in Key order.
var oscParsersIterm2Map_ oscParsersIterm2Map

// OSCParsersIterm2Parse is osc.parsers.iterm2.parse: parse OSC 1337.
// https://iterm2.com/documentation-escape-codes.html
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersIterm2Parse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}
