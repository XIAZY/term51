// WIREFRAME for src/terminal/osc/parsers/color.zig (Ghostty v1.3.1).
// NOT YET PORTED.
//
// Types and signatures only; see the header of color.go for the rules.
//
// File naming: `osc/parsers/color.zig` becomes `osc_parsers_color.go` in
// package terminal, and its declarations take the prefix `OSCParsersColor`
// (`oscParsersColor` for private ones). The file has no C ABI declarations.
//
// Not ported:
//   - `ParseError` (`Allocator.Error || error{MissingOperation}`): an error
//     set is not a type in Go. Its members are the sentinels ErrOutOfMemory
//     (dcs.go) and ErrMissingOperation (below).
//   - `const log = std.log.scoped(.osc_color)`: the port uses
//     warn("osc_color", ...).

package terminal

import "errors"

// ErrMissingOperation is Zig's error.MissingOperation, a member of
// osc.parsers.color.ParseError.
var ErrMissingOperation = errors.New("MissingOperation")

// OSCParsersColorOperation is osc.parsers.color.Operation: the possible
// operations we support for colors.
type OSCParsersColorOperation uint8

const (
	OSCParsersColorOperationOsc4 OSCParsersColorOperation = iota
	OSCParsersColorOperationOsc5
	OSCParsersColorOperationOsc10
	OSCParsersColorOperationOsc11
	OSCParsersColorOperationOsc12
	OSCParsersColorOperationOsc13
	OSCParsersColorOperationOsc14
	OSCParsersColorOperationOsc15
	OSCParsersColorOperationOsc16
	OSCParsersColorOperationOsc17
	OSCParsersColorOperationOsc18
	OSCParsersColorOperationOsc19
	OSCParsersColorOperationOsc104
	OSCParsersColorOperationOsc105
	OSCParsersColorOperationOsc110
	OSCParsersColorOperationOsc111
	OSCParsersColorOperationOsc112
	OSCParsersColorOperationOsc113
	OSCParsersColorOperationOsc114
	OSCParsersColorOperationOsc115
	OSCParsersColorOperationOsc116
	OSCParsersColorOperationOsc117
	OSCParsersColorOperationOsc118
	OSCParsersColorOperationOsc119

	oscParsersColorOperationCount = iota
)

// String returns the Zig tag name (@tagName).
func (self OSCParsersColorOperation) String() string { panic(notImplemented) }

// OSCParsersColorParse is osc.parsers.color.parse: parse OSCs 4, 5, 10-19,
// 104, 110-119.
//
// The result is nil for Zig's null; otherwise it points at the parser's own
// command.
func OSCParsersColorParse(parser *OSCParser, terminator_ch Optional[uint8]) *OSCCommand {
	panic(notImplemented)
}

// oscParsersColorParseColor is osc.parsers.color.parseColor: parse any color
// operation string. This should NOT include the operation itself, but only
// the body of the operation. e.g. for "4;a;b;c" the body should be "a;b;c"
// and the operation should be set accordingly.
//
// Color parsing is fairly complicated so we pull this out to a specialized
// function rather than go through our OSC parsing state machine. This is
// much slower and requires more memory (since we need to buffer the full
// request) but grants us an easier to understand and testable
// implementation.
//
// If color changing ends up being a bottleneck we can optimize this later.
//
// Zig: returns ParseError!List. The allocator parameter is dropped.
func oscParsersColorParseColor(
	op OSCParsersColorOperation,
	buf []uint8,
) (OSCParsersColorList, error) {
	panic(notImplemented)
}

// oscParsersColorTokenIterator stands in for
// std.mem.TokenIterator(u8, .scalar), the iterator returned by
// std.mem.tokenizeScalar(u8, buf, ';') that parseColor hands to its helpers:
// it yields the non-empty runs of buffer between delimiter bytes. Wireframe:
// fields only, with Zig's names.
type oscParsersColorTokenIterator struct {
	buffer    []uint8
	delimiter uint8
	index     int
}

// oscParsersColorTokenizeScalar is std.mem.tokenizeScalar(u8, buffer,
// delimiter): an iterator over the non-empty runs of buffer between
// delimiter bytes.
func oscParsersColorTokenizeScalar(buffer []uint8, delimiter uint8) oscParsersColorTokenIterator {
	panic(notImplemented)
}

// next is std.mem.TokenIterator.next: the next token, or none at the end.
func (self *oscParsersColorTokenIterator) next() Optional[[]uint8] { panic(notImplemented) }

// oscParsersColorParseGetSetAnsiColor is
// osc.parsers.color.parseGetSetAnsiColor: OSC 4/5.
//
// Zig: `op` is a comptime parameter (.osc_4 or .osc_5; anything else does
// not compile) and the result is Allocator.Error!List.
func oscParsersColorParseGetSetAnsiColor(
	op OSCParsersColorOperation,
	it *oscParsersColorTokenIterator,
) (OSCParsersColorList, error) {
	panic(notImplemented)
}

// oscParsersColorParseResetAnsiColor is
// osc.parsers.color.parseResetAnsiColor: OSC 104/105: Reset ANSI Colors.
//
// Zig: `op` is a comptime parameter (.osc_104 or .osc_105; anything else
// does not compile) and the result is Allocator.Error!List.
func oscParsersColorParseResetAnsiColor(
	op OSCParsersColorOperation,
	it *oscParsersColorTokenIterator,
) (OSCParsersColorList, error) {
	panic(notImplemented)
}

// oscParsersColorParseGetSetDynamicColor is
// osc.parsers.color.parseGetSetDynamicColor: OSC 10-19: Get/Set Dynamic
// Colors.
//
// Zig: the result is Allocator.Error!List.
func oscParsersColorParseGetSetDynamicColor(
	start ColorDynamic,
	it *oscParsersColorTokenIterator,
) (OSCParsersColorList, error) {
	panic(notImplemented)
}

// oscParsersColorParseResetDynamicColor is
// osc.parsers.color.parseResetDynamicColor: OSC 110-119: Reset Dynamic
// Colors.
//
// Zig: the result is Allocator.Error!List.
func oscParsersColorParseResetDynamicColor(
	color ColorDynamic,
	it *oscParsersColorTokenIterator,
) (OSCParsersColorList, error) {
	panic(notImplemented)
}

// OSCParsersColorList is osc.parsers.color.List:
// std.SegmentedList(Request, 2).
//
// A segmented list is used to avoid copying when many operations are given
// in a single OSC. In most cases, OSC 4/104/etc. send very few so the
// prealloc is optimized for that.
//
// In Go the segments are not observable, so the list is one slice. The
// methods keep Zig's names (the way StaticBitSet256 does in color.go); the
// zero value is Zig's `.{}`, the empty list.
type OSCParsersColorList struct {
	items []OSCParsersColorRequest
}

// append is std.SegmentedList.append: add one request at the end. Zig
// returns Allocator.Error!void.
func (self *OSCParsersColorList) append(item OSCParsersColorRequest) error {
	panic(notImplemented)
}

// count is std.SegmentedList.count: the number of requests.
func (self OSCParsersColorList) count() int { panic(notImplemented) }

// at is std.SegmentedList.at: a pointer to the request at index i. Zig
// asserts i < count().
func (self *OSCParsersColorList) at(i int) *OSCParsersColorRequest { panic(notImplemented) }

// constIterator is std.SegmentedList.constIterator: an iterator positioned
// at start_index.
func (self *OSCParsersColorList) constIterator(start_index int) OSCParsersColorListConstIterator {
	panic(notImplemented)
}

// deinit is std.SegmentedList.deinit. Zig frees the list; the allocator
// parameter is dropped.
func (self *OSCParsersColorList) deinit() { panic(notImplemented) }

// OSCParsersColorListConstIterator is the iterator returned by
// OSCParsersColorList.constIterator (std.SegmentedList.ConstIterator).
type OSCParsersColorListConstIterator struct {
	list  *OSCParsersColorList
	index int
}

// next returns the next request, or nil at the end. Zig: next() ?*const T.
func (self *OSCParsersColorListConstIterator) next() *OSCParsersColorRequest {
	panic(notImplemented)
}

// OSCParsersColorRequestTag is the tag of osc.parsers.color.Request.
type OSCParsersColorRequestTag uint8

const (
	OSCParsersColorRequestSet OSCParsersColorRequestTag = iota
	OSCParsersColorRequestQuery
	OSCParsersColorRequestReset
	OSCParsersColorRequestResetPalette
	OSCParsersColorRequestResetSpecial
)

// OSCParsersColorRequest is osc.parsers.color.Request: a single operation
// related to the terminal color palette.
//
// Zig: a tagged union. Only the field named by Tag is meaningful;
// reset_palette and reset_special carry no payload.
type OSCParsersColorRequest struct {
	Tag OSCParsersColorRequestTag

	Set   OSCParsersColorColoredTarget
	Query OSCParsersColorTarget
	Reset OSCParsersColorTarget
}

// OSCParsersColorTargetTag is the tag of osc.parsers.color.Target.
type OSCParsersColorTargetTag uint8

const (
	OSCParsersColorTargetPalette OSCParsersColorTargetTag = iota
	OSCParsersColorTargetSpecial
	OSCParsersColorTargetDynamic
)

// OSCParsersColorTarget is osc.parsers.color.Target.
//
// Zig: a tagged union. Only the field named by Tag is meaningful.
type OSCParsersColorTarget struct {
	Tag OSCParsersColorTargetTag

	Palette uint8
	Special ColorSpecial
	Dynamic ColorDynamic
}

// OSCParsersColorColoredTarget is osc.parsers.color.ColoredTarget.
type OSCParsersColorColoredTarget struct {
	target OSCParsersColorTarget
	color  ColorRGB
}
