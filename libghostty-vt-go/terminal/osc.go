// Ported from src/terminal/osc.zig (Ghostty v1.3.1).
//
// OSC (Operating System Command) related functions and types.
//
// OSC is another set of control sequences for terminal programs that start
// with "ESC ]". Unlike CSI or standard ESC sequences, they may contain
// strings and other irregular formatting so a dedicated parser is created to
// handle it.
//
// File naming: osc.zig is a namespace file, so its declarations take the
// prefix `OSC`. Declarations appear in the same order as in osc.zig.
//
// The stand-ins for std.Io.Writer and std.Io.Writer.Allocating (oscWriter,
// oscWriterAllocating) come first: osc.zig and every file under osc/ use
// them, as do Screen.go, formatter.go and render.go.
//
// Not ported, see docs/mapping.md:
//   - the C ABI declarations: `Command.ProgressReport.C` and `cval`,
//     `Terminator.C` and `cval`, the C variant of `Command.Key` (LibEnum with
//     the .c target) and the "ghostty.h" enum check;
//   - the comptime size assertion on Command (memory layout only);
//   - the namespace re-exports `pub const color = parsers.color` and
//     `pub const semantic_prompt = parsers.semantic_prompt` (Go has one
//     package; the names are OSCParsersColor... and
//     OSCParsersSemanticPrompt...);
//   - `const log = std.log.scoped(.osc)`: the port uses warn("osc", ...);
//   - the Valgrind blocks of Parser.init and Parser.reset (they only mark
//     `buffer` as undefined for Valgrind).

package terminal

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ErrWriteFailed is Zig's error.WriteFailed (std.Io.Writer.Error): a write
// did not fit in the writer's buffer. osc.Parser.next turns it into the
// invalid state.
var ErrWriteFailed = errors.New("WriteFailed")

// oscWriter stands in for std.Io.Writer as osc.zig and the files under osc/
// use it: a buffer and the number of bytes written to it. Zig's
// `writer.buffered()` is buffer[:end]. The methods keep the names of
// std.Io.Writer's (writeByte, write, writeAll) and follow its code: bytes go
// into buffer[end:] while they fit, and the rest goes to `drain`, which for
// a fixed writer fails with ErrWriteFailed and for an allocating writer
// grows the buffer.
type oscWriter struct {
	// Zig: buffer: []u8. For a fixed writer this is the parser's own
	// `buffer` array.
	buffer []uint8

	// Zig: end: usize. The number of bytes of buffer in use.
	end int

	// Zig: vtable: *const VTable. Which of the two implementations of
	// std.Io.Writer this writer is: the one of `Writer.fixed` or the one
	// of `Writer.Allocating`. Go cannot recover the Allocating that embeds
	// a writer (Zig's @fieldParentPtr), so the writer itself says which it
	// is. The zero value is decided by the buffer (see isAllocating), so
	// that both `oscWriter{buffer: buf[:]}` (a fixed writer) and the zero
	// oscWriterAllocating (Zig's `Allocating.init`, whose buffer is empty)
	// behave as in Zig.
	vtable oscWriterVTable

	// Zig: Discarding.count: u64, the field of the std.Io.Writer.Discarding
	// that embeds this writer. Go cannot recover the Discarding from its
	// writer in drain (Zig's @fieldParentPtr), so the count lives here and
	// only the Discarding vtable uses it; see oscWriterDiscarding.
	count uint64
}

// oscWriterVTable selects the implementation of std.Io.Writer (Zig's
// vtable pointer).
type oscWriterVTable uint8

const (
	// Not set: a writer with a nil buffer is an Allocating writer that has
	// not allocated yet, any other is a fixed writer. This is the zero
	// value, which a Go composite literal gets.
	oscWriterVTableUnset oscWriterVTable = iota

	// std.Io.Writer.fixed: drain is fixedDrain.
	oscWriterVTableFixed

	// std.Io.Writer.Allocating: drain is Allocating.drain.
	oscWriterVTableAllocating

	// std.Io.Writer.Discarding: drain is Discarding.drain.
	oscWriterVTableDiscarding
)

// isAllocating reports whether the writer's vtable is Allocating's.
func (self *oscWriter) isAllocating() bool {
	switch self.vtable {
	case oscWriterVTableFixed:
		return false
	case oscWriterVTableAllocating:
		return true
	}
	return self.buffer == nil
}

// oscWriterAllocating stands in for std.Io.Writer.Allocating: a writer whose
// buffer grows. The allocator field is dropped. The zero value is
// `Allocating.init`'s: an empty writer that grows on demand.
type oscWriterAllocating struct {
	// Zig: writer: std.Io.Writer.
	writer oscWriter
}

// oscWriterAllocatingInit is std.Io.Writer.Allocating.init: an empty
// allocating writer. The allocator parameter is dropped.
func oscWriterAllocatingInit() oscWriterAllocating {
	return oscWriterAllocating{
		writer: oscWriter{
			buffer: nil,
			vtable: oscWriterVTableAllocating,
		},
	}
}

// oscWriterAllocatingInitCapacity is std.Io.Writer.Allocating.initCapacity:
// an allocating writer whose buffer starts with room for capacity bytes.
// Zig: error{OutOfMemory}!Allocating; Go's make does not fail, so the error
// is always nil.
func oscWriterAllocatingInitCapacity(capacity int) (oscWriterAllocating, error) {
	return oscWriterAllocating{
		writer: oscWriter{
			buffer: make([]uint8, capacity),
			vtable: oscWriterVTableAllocating,
		},
	}, nil
}

// deinit is std.Io.Writer.Allocating.deinit. Zig frees the buffer and
// leaves the writer undefined; Go drops the buffer.
func (self *oscWriterAllocating) deinit() {
	*self = oscWriterAllocating{}
}

// toOwnedSlice is std.Io.Writer.Allocating.toOwnedSlice: the bytes written
// so far, handed to the caller, leaving the writer empty. The allocator
// parameter is dropped. Zig: Error![]u8; the error would be ErrOutOfMemory
// (dcs.go), which Go never returns.
//
// Zig goes through toArrayList, ArrayList.toOwnedSlice (a remap or a copy
// of exactly items.len bytes, after which the list is empty) and
// setArrayList; the result is the same as here.
func (self *oscWriterAllocating) toOwnedSlice() ([]uint8, error) {
	w := &self.writer
	result := make([]uint8, w.end)
	copy(result, w.buffer[0:w.end])
	w.buffer = nil
	w.end = 0
	return result, nil
}

// written is std.Io.Writer.Allocating.written: the bytes written so far.
func (self *oscWriterAllocating) written() []uint8 {
	return self.writer.buffered()
}

// clearRetainingCapacity is std.Io.Writer.Allocating.clearRetainingCapacity:
// discard the bytes written so far (written() becomes empty) and keep the
// buffer for reuse. Zig calls shrinkRetainingCapacity(0), which sets end.
func (self *oscWriterAllocating) clearRetainingCapacity() {
	self.writer.end = 0
}

// oscWriterAllocatingDrain is std.Io.Writer.Allocating.drain: append every
// slice of data, the last one splat times, growing the buffer as needed.
// Zig works on the writer as an ArrayList(u8) (toArrayList and
// setArrayList): items are buffer[0:end], the capacity is len(buffer).
func oscWriterAllocatingDrain(w *oscWriter, data [][]uint8, splat int) (int, error) {
	pattern := data[len(data)-1]
	splat_len := len(pattern) * splat
	start_len := w.end
	assert(len(data) != 0)
	for _, bytes := range data {
		oscWriterArrayListEnsureUnusedCapacity(w, len(bytes)+splat_len+1)
		copy(w.buffer[w.end:], bytes)
		w.end += len(bytes)
	}
	if splat == 0 {
		w.end -= len(pattern)
	} else {
		switch len(pattern) {
		case 0:
		case 1:
			for i := 0; i < splat-1; i++ {
				w.buffer[w.end] = pattern[0]
				w.end += 1
			}
		default:
			for i := 0; i < splat-1; i++ {
				copy(w.buffer[w.end:], pattern)
				w.end += len(pattern)
			}
		}
	}
	return w.end - start_len, nil
}

// oscWriterArrayListInitCapacity is ArrayList(u8)'s init_capacity:
// @max(1, std.atomic.cache_line / @sizeOf(u8)), with the cache line of the
// 64-bit targets the corpus was recorded on (128 on x86_64 and aarch64).
const oscWriterArrayListInitCapacity = 128

// oscWriterArrayListEnsureUnusedCapacity is ArrayList(u8).ensureUnusedCapacity
// on the writer's buffer seen as a list: make room for additional_count more
// bytes, growing the capacity as ArrayList.growCapacity does. Zig's
// allocation failure (error.WriteFailed in drain) cannot happen in Go.
func oscWriterArrayListEnsureUnusedCapacity(w *oscWriter, additional_count int) {
	new_capacity := w.end + additional_count
	if len(w.buffer) >= new_capacity {
		return
	}
	// growCapacity: a capacity larger than minimum that grows
	// super-linearly.
	better_capacity := len(w.buffer)
	for {
		better_capacity += better_capacity/2 + oscWriterArrayListInitCapacity
		if better_capacity >= new_capacity {
			break
		}
	}
	grown := make([]uint8, better_capacity)
	copy(grown, w.buffer[0:w.end])
	w.buffer = grown
	// Once it has allocated, an Allocating writer is no longer told by
	// its nil buffer.
	w.vtable = oscWriterVTableAllocating
}

// oscWriterFixed is std.Io.Writer.fixed: a writer that writes into buf and
// fails with ErrWriteFailed once buf is full. Zig: `var w: std.Io.Writer =
// .fixed(&buf);` is `w := oscWriterFixed(buf[:])`.
func oscWriterFixed(buf []uint8) oscWriter {
	return oscWriter{
		buffer: buf,
		vtable: oscWriterVTableFixed,
	}
}

// oscWriterFixedDrain is std.Io.Writer.fixedDrain: copy what fits of data
// (the last slice splat times) into the rest of the buffer and fail with
// ErrWriteFailed when it does not all fit.
func oscWriterFixedDrain(w *oscWriter, data [][]uint8, splat int) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	for _, bytes := range data[0 : len(data)-1] {
		dest := w.buffer[w.end:]
		n := min(len(bytes), len(dest))
		copy(dest[0:n], bytes[0:n])
		w.end += n
		if len(bytes) > len(dest) {
			return 0, ErrWriteFailed
		}
	}
	pattern := data[len(data)-1]
	dest := w.buffer[w.end:]
	switch len(pattern) {
	case 0:
		return 0, nil
	case 1:
		assert(splat >= len(dest))
		for i := range dest {
			dest[i] = pattern[0]
		}
		w.end += len(dest)
		return 0, ErrWriteFailed
	default:
		for i := 0; i < splat; i++ {
			remaining := dest[i*len(pattern):]
			n := min(len(pattern), len(remaining))
			copy(remaining[0:n], pattern[0:n])
			w.end += n
			if len(pattern) > len(remaining) {
				return 0, ErrWriteFailed
			}
		}
		panic("unreachable")
	}
}

// drain is the writer's vtable.drain: oscWriterFixedDrain,
// oscWriterAllocatingDrain or oscWriterDiscardingDrain.
func (self *oscWriter) drain(data [][]uint8, splat int) (int, error) {
	if self.vtable == oscWriterVTableDiscarding {
		return oscWriterDiscardingDrain(self, data, splat)
	}
	if self.isAllocating() {
		return oscWriterAllocatingDrain(self, data, splat)
	}
	return oscWriterFixedDrain(self, data, splat)
}

// buffered is std.Io.Writer.buffered: the bytes written into the writer's
// buffer so far, buffer[:end]. For an allocating writer Zig's
// `builder.writer.buffered()` is `builder.writer.buffered()`.
func (self *oscWriter) buffered() []uint8 {
	return self.buffer[0:self.end]
}

// write is std.Io.Writer.write: write as many bytes as possible, returning
// how many were written.
func (self *oscWriter) write(bytes []uint8) (int, error) {
	if self.end+len(bytes) <= len(self.buffer) {
		copy(self.buffer[self.end:][0:len(bytes)], bytes)
		self.end += len(bytes)
		return len(bytes), nil
	}
	return self.drain([][]uint8{bytes}, 1)
}

// writeAll is std.Io.Writer.writeAll: calls drain as many times as
// necessary such that all of bytes are transferred.
func (self *oscWriter) writeAll(bytes []uint8) error {
	index := 0
	for index < len(bytes) {
		n, err := self.write(bytes[index:])
		if err != nil {
			return err
		}
		index += n
	}
	return nil
}

// writeByte is std.Io.Writer.writeByte.
func (self *oscWriter) writeByte(byte_ uint8) error {
	for len(self.buffer)-self.end == 0 {
		n, err := self.drain([][]uint8{{byte_}}, 1)
		if err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
	}
	self.buffer[self.end] = byte_
	self.end += 1
	return nil
}

// printUnicodeCodepoint is std.Io.Writer.printUnicodeCodepoint (what `{u}`
// prints): the UTF-8 encoding of c, or U+FFFD for a surrogate half or a
// codepoint above U+10FFFF. Zig's std.unicode.utf8Encode is Go's
// utf8.EncodeRune, which encodes and replaces exactly those.
//
// Zig: c is a u21.
func (self *oscWriter) printUnicodeCodepoint(c uint32) error {
	var buf [4]uint8
	len_ := utf8.EncodeRune(buf[:], rune(c))
	return self.writeAll(buf[0:len_])
}

// countSplat is std.Io.Writer.countSplat: the number of bytes of data, the
// last slice counted splat times.
func oscWriterCountSplat(data [][]uint8, splat int) int {
	total := 0
	for _, buf := range data[0 : len(data)-1] {
		total += len(buf)
	}
	total += len(data[len(data)-1]) * splat
	return total
}

// writeSplat is std.Io.Writer.writeSplat: if the number of bytes to write
// based on `data` and `splat` fits inside the unused capacity, this does
// not fail, does not call into the vtable, and returns the full number of
// bytes.
func (self *oscWriter) writeSplat(data [][]uint8, splat int) (int, error) {
	assert(len(data) > 0)
	buffer := self.buffer
	count := oscWriterCountSplat(data, splat)
	if self.end+count > len(buffer) {
		return self.drain(data, splat)
	}
	for _, bytes := range data[0 : len(data)-1] {
		copy(buffer[self.end:][0:len(bytes)], bytes)
		self.end += len(bytes)
	}
	pattern := data[len(data)-1]
	switch len(pattern) {
	case 0:
	case 1:
		for i := range buffer[self.end:][0:splat] {
			buffer[self.end+i] = pattern[0]
		}
		self.end += splat
	default:
		for i := 0; i < splat; i++ {
			copy(buffer[self.end:][0:len(pattern)], pattern)
			self.end += len(pattern)
		}
	}
	return count, nil
}

// splatByteAll is std.Io.Writer.splatByteAll: writes the same byte many
// times, performing the underlying write call as many times as necessary.
//
// Zig: n is a usize.
func (self *oscWriter) splatByteAll(byte_ uint8, n int) error {
	remaining := n
	for remaining > 0 {
		written, err := self.splatByte(byte_, remaining)
		if err != nil {
			return err
		}
		remaining -= written
	}
	return nil
}

// splatByte is std.Io.Writer.splatByte: writes the same byte many times,
// allowing short writes. Does maximum of one underlying `VTable.drain`.
func (self *oscWriter) splatByte(byte_ uint8, n int) (int, error) {
	if self.end+n <= len(self.buffer) {
		for i := range self.buffer[self.end:][0:n] {
			self.buffer[self.end+i] = byte_
		}
		self.end += n
		return n, nil
	}
	return self.writeSplat([][]uint8{{byte_}}, n)
}

// oscWriterDiscarding stands in for std.Io.Writer.Discarding: a writer
// that discards everything written to it and counts the bytes. Zig's
// `discarding.count` is `discarding.writer.count` (see oscWriter.count).
// Only `init`, `count` and `drain` are ported; `fullCount` and `sendFile`
// have no caller.
type oscWriterDiscarding struct {
	// Zig: writer: std.Io.Writer. Its `count` field is Discarding.count.
	writer oscWriter
}

// oscWriterDiscardingInit is std.Io.Writer.Discarding.init. Zig:
// `var discarding: std.Io.Writer.Discarding = .init(&.{});` is
// `discarding := oscWriterDiscardingInit(nil)`.
func oscWriterDiscardingInit(buffer []uint8) oscWriterDiscarding {
	return oscWriterDiscarding{
		writer: oscWriter{
			vtable: oscWriterVTableDiscarding,
			buffer: buffer,
			count:  0,
		},
	}
}

// oscWriterDiscardingDrain is std.Io.Writer.Discarding.drain: count the
// buffered bytes and every byte of data (the last slice splat times), then
// empty the buffer.
func oscWriterDiscardingDrain(w *oscWriter, data [][]uint8, splat int) (int, error) {
	slice := data[0 : len(data)-1]
	pattern := data[len(slice)]
	written := len(pattern) * splat
	for _, bytes := range slice {
		written += len(bytes)
	}
	w.count += uint64(w.end + written)
	w.end = 0
	return written, nil
}

// OSCCommand is osc.Command.
//
// Zig: union(Key). Only the field named by Tag is meaningful. The variants
// invalid, hyperlink_end and conemu_wait_input carry no payload and have no
// field.
type OSCCommand struct {
	Tag OSCCommandKey

	// Set the window title of the terminal
	//
	// If title mode 0 is set text is expect to be hex encoded (i.e. utf-8
	// with each code unit further encoded with two hex digits).
	//
	// If title mode 2 is set or the terminal is setup for unconditional
	// utf-8 titles text is interpreted as utf-8. Else text is interpreted
	// as latin1.
	ChangeWindowTitle []uint8

	// Set the icon of the terminal window. The name of the icon is not
	// well defined, so this is currently ignored by Ghostty at the time
	// of writing this. We just parse it so that we don't get parse errors
	// in the log.
	ChangeWindowIcon []uint8

	// Semantic prompt command: https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md
	SemanticPrompt OSCCommandSemanticPrompt

	// Set or get clipboard contents. If data is null, then the current
	// clipboard contents are sent to the pty. If data is set, this
	// contents is set on the clipboard.
	ClipboardContents OSCCommandClipboardContentsPayload

	// OSC 7. Reports the current working directory of the shell. This is
	// a moderately flawed escape sequence but one that many major terminals
	// support so we also support it. To understand the flaws, read through
	// this terminal-wg issue: https://gitlab.freedesktop.org/terminal-wg/specifications/-/issues/20
	ReportPwd OSCCommandReportPwdPayload

	// OSC 22. Set the mouse shape. There doesn't seem to be a standard
	// naming scheme for cursors but it looks like terminals such as Foot
	// are moving towards using the W3C CSS cursor names. For OSC parsing,
	// we just parse whatever string is given.
	MouseShape OSCCommandMouseShapePayload

	// OSC color operations to set, reset, or report color settings. Some
	// OSCs allow multiple operations to be specified in a single OSC so we
	// need a list-like datastructure to manage them.
	//
	// Currently, these OSCs are handled by `color_operation`:
	//
	// 4, 5, 10-19, 104, 105, 110-119
	ColorOperation OSCCommandColorOperationPayload

	// Kitty color protocol, OSC 21
	// https://sw.kovidgoyal.net/kitty/color-stack/#id1
	KittyColorProtocol KittyColorOSC

	// Show a desktop notification (OSC 9 or OSC 777)
	ShowDesktopNotification OSCCommandShowDesktopNotificationPayload

	// Start a hyperlink (OSC 8)
	HyperlinkStart OSCCommandHyperlinkStartPayload

	// ConEmu sleep (OSC 9;1)
	ConemuSleep OSCCommandConemuSleepPayload

	// ConEmu show GUI message box (OSC 9;2)
	ConemuShowMessageBox []uint8

	// ConEmu change tab title (OSC 9;3)
	ConemuChangeTabTitle OSCCommandConemuChangeTabTitlePayload

	// ConEmu progress report (OSC 9;4)
	ConemuProgressReport OSCCommandProgressReport

	// ConEmu GUI macro (OSC 9;6)
	ConemuGuimacro []uint8

	// ConEmu run process (OSC 9;7)
	ConemuRunProcess []uint8

	// ConEmu output environment variable (OSC 9;8)
	ConemuOutputEnvironmentVariable []uint8

	// ConEmu XTerm keyboard and output emulation (OSC 9;10)
	// https://conemu.github.io/en/TerminalModes.html
	ConemuXtermEmulation OSCCommandConemuXtermEmulationPayload

	// ConEmu comment (OSC 9;11)
	ConemuComment []uint8

	// Kitty text sizing protocol (OSC 66)
	KittyTextSizing OSCParsersKittyTextSizingOSC

	KittyClipboardProtocol OSCCommandKittyClipboardProtocol

	// OSC 3008. Hierarchical context signalling (UAPI spec).
	// https://uapi-group.org/specifications/specs/osc_context/
	ContextSignal OSCParsersContextSignalCommand
}

// zigFmtAny is not a declaration of osc.zig: it is the port's rendering of
// Zig's std.fmt "{any}" for a Command, which Parser.go calls when it formats
// an osc_dispatch action (Action.format prints the payload with
// `writer.printValue("any", .{}, value, 3)`).
//
// It follows std.Io.Writer.printValue (Zig 0.15.2) for each variant's
// payload type, at the depths that printValue reaches: the union is
// printed at max_depth 3, its payload at 2, the payload's fields at 1 and
// their fields at 0, where a struct or union prints ".{ ... }" and a slice
// or array "{ ... }"; integers, bools, enums and null are printed at every
// depth. A union prints `.{ .<tag> = <payload> }`, a struct
// `.{ .<field> = <value>, ... }`, a byte slice `{ 97, 98 }` (`{  }` when
// empty), an enum `.<tag name>`, a void payload `void`, a null optional
// `null` and a non-null one as its payload. Format methods (such as
// Terminator.format) are skipped by "{any}". The output was checked against
// upstream's Action.format for every variant.
func (self OSCCommand) zigFmtAny() string {
	// The union itself is at max_depth 3; its payload is printed at 2.
	const depth = 3
	var b strings.Builder
	b.WriteString(".{ .")
	b.WriteString(self.Tag.String())
	b.WriteString(" = ")
	switch self.Tag {
	case OSCCommandKeyInvalid, OSCCommandKeyHyperlinkEnd, OSCCommandKeyConemuWaitInput:
		b.WriteString("void")
	case OSCCommandKeyChangeWindowTitle:
		b.WriteString(zigFmtAnyBytes(self.ChangeWindowTitle, depth-1))
	case OSCCommandKeyChangeWindowIcon:
		b.WriteString(zigFmtAnyBytes(self.ChangeWindowIcon, depth-1))
	case OSCCommandKeySemanticPrompt:
		v := self.SemanticPrompt
		zigFmtAnyStruct(&b, depth-1,
			"action", func(int) string { return "." + v.Action.String() },
			"options_unvalidated", func(d int) string { return zigFmtAnyBytes(v.OptionsUnvalidated, d) },
		)
	case OSCCommandKeyClipboardContents:
		v := self.ClipboardContents
		zigFmtAnyStruct(&b, depth-1,
			"kind", func(int) string { return strconv.FormatUint(uint64(v.Kind), 10) },
			"data", func(d int) string { return zigFmtAnyBytes(v.Data, d) },
		)
	case OSCCommandKeyReportPwd:
		v := self.ReportPwd
		zigFmtAnyStruct(&b, depth-1,
			"value", func(d int) string { return zigFmtAnyBytes(v.Value, d) },
		)
	case OSCCommandKeyMouseShape:
		v := self.MouseShape
		zigFmtAnyStruct(&b, depth-1,
			"value", func(d int) string { return zigFmtAnyBytes(v.Value, d) },
		)
	case OSCCommandKeyColorOperation:
		v := self.ColorOperation
		zigFmtAnyStruct(&b, depth-1,
			"op", func(int) string { return "." + v.Op.String() },
			// color.List is std.SegmentedList(Request, 2): fields
			// prealloc_segment ([2]Request), dynamic_segments
			// ([][*]Request) and len (usize).
			"requests", func(d int) string {
				var l strings.Builder
				zigFmtAnyStruct(&l, d,
					"prealloc_segment", func(d int) string { return zigFmtAnyElided(d, "{ ... }") },
					"dynamic_segments", func(d int) string { return zigFmtAnyElided(d, "{ ... }") },
					"len", func(int) string { return strconv.Itoa(v.Requests.Count()) },
				)
				return l.String()
			},
			"terminator", func(int) string { return "." + v.Terminator.TagName() },
		)
	case OSCCommandKeyKittyColorProtocol:
		v := self.KittyColorProtocol
		zigFmtAnyStruct(&b, depth-1,
			// std.ArrayList(Request): fields items ([]Request) and
			// capacity (usize).
			"list", func(d int) string {
				var l strings.Builder
				zigFmtAnyStruct(&l, d,
					"items", func(d int) string { return zigFmtAnyElided(d, "{ ... }") },
					"capacity", func(int) string { return strconv.Itoa(oscKittyColorListCapacity(len(v.List))) },
				)
				return l.String()
			},
			"terminator", func(int) string { return "." + v.Terminator.TagName() },
		)
	case OSCCommandKeyShowDesktopNotification:
		v := self.ShowDesktopNotification
		zigFmtAnyStruct(&b, depth-1,
			"title", func(d int) string { return zigFmtAnyBytes(v.Title, d) },
			"body", func(d int) string { return zigFmtAnyBytes(v.Body, d) },
		)
	case OSCCommandKeyHyperlinkStart:
		v := self.HyperlinkStart
		zigFmtAnyStruct(&b, depth-1,
			"id", func(d int) string {
				if !v.ID.Valid {
					return "null"
				}
				return zigFmtAnyBytes(v.ID.Value, d)
			},
			"uri", func(d int) string { return zigFmtAnyBytes(v.URI, d) },
		)
	case OSCCommandKeyConemuSleep:
		v := self.ConemuSleep
		zigFmtAnyStruct(&b, depth-1,
			"duration_ms", func(int) string { return strconv.FormatUint(uint64(v.DurationMs), 10) },
		)
	case OSCCommandKeyConemuShowMessageBox:
		b.WriteString(zigFmtAnyBytes(self.ConemuShowMessageBox, depth-1))
	case OSCCommandKeyConemuChangeTabTitle:
		// A union(enum) { reset, value: [:0]const u8 } at depth 2.
		v := self.ConemuChangeTabTitle
		b.WriteString(".{ .")
		b.WriteString(v.Tag.String())
		b.WriteString(" = ")
		switch v.Tag {
		case OSCCommandConemuChangeTabTitlePayloadReset:
			b.WriteString("void")
		case OSCCommandConemuChangeTabTitlePayloadValue:
			b.WriteString(zigFmtAnyBytes(v.Value, depth-2))
		}
		b.WriteString(" }")
	case OSCCommandKeyConemuProgressReport:
		v := self.ConemuProgressReport
		zigFmtAnyStruct(&b, depth-1,
			"state", func(int) string { return "." + v.State.String() },
			"progress", func(int) string {
				if !v.Progress.Valid {
					return "null"
				}
				return strconv.FormatUint(uint64(v.Progress.Value), 10)
			},
		)
	case OSCCommandKeyConemuGuimacro:
		b.WriteString(zigFmtAnyBytes(self.ConemuGuimacro, depth-1))
	case OSCCommandKeyConemuRunProcess:
		b.WriteString(zigFmtAnyBytes(self.ConemuRunProcess, depth-1))
	case OSCCommandKeyConemuOutputEnvironmentVariable:
		b.WriteString(zigFmtAnyBytes(self.ConemuOutputEnvironmentVariable, depth-1))
	case OSCCommandKeyConemuXtermEmulation:
		v := self.ConemuXtermEmulation
		zigFmtAnyStruct(&b, depth-1,
			"keyboard", func(int) string { return zigFmtAnyOptionalBool(v.Keyboard) },
			"output", func(int) string { return zigFmtAnyOptionalBool(v.Output) },
		)
	case OSCCommandKeyConemuComment:
		b.WriteString(zigFmtAnyBytes(self.ConemuComment, depth-1))
	case OSCCommandKeyKittyTextSizing:
		v := self.KittyTextSizing
		zigFmtAnyStruct(&b, depth-1,
			"scale", func(int) string { return strconv.FormatUint(uint64(v.Scale), 10) },
			"width", func(int) string { return strconv.FormatUint(uint64(v.Width), 10) },
			"numerator", func(int) string { return strconv.FormatUint(uint64(v.Numerator), 10) },
			"denominator", func(int) string { return strconv.FormatUint(uint64(v.Denominator), 10) },
			"valign", func(int) string { return "." + v.Valign.String() },
			"halign", func(int) string { return "." + v.Halign.String() },
			"text", func(d int) string { return zigFmtAnyBytes(v.Text, d) },
		)
	case OSCCommandKeyKittyClipboardProtocol:
		v := self.KittyClipboardProtocol
		zigFmtAnyStruct(&b, depth-1,
			"metadata", func(d int) string { return zigFmtAnyBytes(v.Metadata, d) },
			"payload", func(d int) string {
				if !v.Payload.Valid {
					return "null"
				}
				return zigFmtAnyBytes(v.Payload.Value, d)
			},
			"terminator", func(int) string { return "." + v.Terminator.TagName() },
		)
	case OSCCommandKeyContextSignal:
		v := self.ContextSignal
		zigFmtAnyStruct(&b, depth-1,
			"action", func(int) string { return "." + v.Action.String() },
			"id", func(d int) string { return zigFmtAnyBytes(v.ID, d) },
			"metadata", func(d int) string { return zigFmtAnyBytes(v.Metadata, d) },
		)
	default:
		panic("unreachable")
	}
	b.WriteString(" }")
	return b.String()
}

// zigFmtAnyStruct is printValue's `.@"struct"` arm: ".{ ... }" at depth 0,
// otherwise ".{ .a = x, .b = y }" with each field printed at depth-1.
// fields alternates a field name (string) and its printer
// (func(depth int) string).
func zigFmtAnyStruct(b *strings.Builder, depth int, fields ...any) {
	if depth == 0 {
		b.WriteString(".{ ... }")
		return
	}
	b.WriteString(".{")
	for i := 0; i < len(fields); i += 2 {
		if i == 0 {
			b.WriteString(" .")
		} else {
			b.WriteString(", .")
		}
		b.WriteString(fields[i].(string))
		b.WriteString(" = ")
		b.WriteString(fields[i+1].(func(int) string)(depth - 1))
	}
	b.WriteString(" }")
}

// zigFmtAnyBytes is printValue's slice arm for a []const u8: "{ ... }" at
// depth 0, otherwise each byte as a decimal integer.
func zigFmtAnyBytes(s []uint8, depth int) string {
	if depth == 0 {
		return "{ ... }"
	}
	return zigFmtAnySlice(s)
}

// zigFmtAnyElided is what printValue prints for a slice or array (elided
// "{ ... }") or a struct (".{ ... }") at depth 0. It is only reached at
// depth 0 here; any other depth would need the element type's printer.
func zigFmtAnyElided(depth int, elided string) string {
	assert(depth == 0)
	return elided
}

// zigFmtAnyOptionalBool is printValue for a ?bool.
func zigFmtAnyOptionalBool(v Optional[bool]) string {
	if !v.Valid {
		return "null"
	}
	if v.Value {
		return "true"
	}
	return "false"
}

// oscKittyColorListCapacity is the capacity of the std.ArrayList(Request)
// of a kitty_color_protocol command holding n requests, which "{any}"
// prints. The parser (osc/parsers/kitty_color.zig) starts from `.empty` and
// only appends one request at a time, so the capacity is the first value
// of ArrayList.growCapacity's sequence (new +|= new / 2 + init_capacity)
// that reaches n. init_capacity is @max(1, std.atomic.cache_line /
// @sizeOf(Request)) = 128 / 12 = 10 on the 64-bit targets the corpus was
// recorded on (checked against upstream). Go's slice keeps its own capacity,
// which is not Zig's.
func oscKittyColorListCapacity(n int) int {
	const init_capacity = 10
	capacity := 0
	for capacity < n {
		capacity += capacity/2 + init_capacity
	}
	return capacity
}

// OSCCommandClipboardContentsPayload is the payload of
// Command.clipboard_contents: struct { kind: u8, data: [:0]const u8 }.
type OSCCommandClipboardContentsPayload struct {
	Kind uint8
	Data []uint8
}

// OSCCommandReportPwdPayload is the payload of Command.report_pwd:
// struct { value: [:0]const u8 }.
type OSCCommandReportPwdPayload struct {
	// The reported pwd value. This is not checked for validity. It should
	// be a file URL but it is up to the caller to utilize this value.
	Value []uint8
}

// OSCCommandMouseShapePayload is the payload of Command.mouse_shape:
// struct { value: [:0]const u8 }.
type OSCCommandMouseShapePayload struct {
	Value []uint8
}

// OSCCommandColorOperationPayload is the payload of Command.color_operation.
type OSCCommandColorOperationPayload struct {
	Op OSCParsersColorOperation

	// Zig: color.List, default .{} (empty). The zero value is the empty
	// list.
	Requests OSCParsersColorList

	// Zig: default .st, which is the zero value.
	Terminator OSCTerminator
}

// OSCCommandShowDesktopNotificationPayload is the payload of
// Command.show_desktop_notification.
type OSCCommandShowDesktopNotificationPayload struct {
	Title []uint8
	Body  []uint8
}

// OSCCommandHyperlinkStartPayload is the payload of Command.hyperlink_start.
type OSCCommandHyperlinkStartPayload struct {
	// Zig: ?[:0]const u8, default null.
	ID  Optional[[]uint8]
	URI []uint8
}

// OSCCommandConemuSleepPayload is the payload of Command.conemu_sleep.
type OSCCommandConemuSleepPayload struct {
	DurationMs uint16
}

// OSCCommandConemuChangeTabTitlePayloadTag is the tag of the payload of
// Command.conemu_change_tab_title.
type OSCCommandConemuChangeTabTitlePayloadTag uint8

const (
	OSCCommandConemuChangeTabTitlePayloadReset OSCCommandConemuChangeTabTitlePayloadTag = iota
	OSCCommandConemuChangeTabTitlePayloadValue
)

// String returns the Zig tag name (@tagName).
func (self OSCCommandConemuChangeTabTitlePayloadTag) String() string {
	return oscCommandConemuChangeTabTitlePayloadTagNames[self]
}

var oscCommandConemuChangeTabTitlePayloadTagNames = [...]string{
	"reset",
	"value",
}

// OSCCommandConemuChangeTabTitlePayload is the payload of
// Command.conemu_change_tab_title:
// union(enum) { reset, value: [:0]const u8 }.
//
// Zig: a tagged union. Only the field named by Tag is meaningful; reset
// carries no payload.
type OSCCommandConemuChangeTabTitlePayload struct {
	Tag OSCCommandConemuChangeTabTitlePayloadTag

	Value []uint8
}

// OSCCommandConemuXtermEmulationPayload is the payload of
// Command.conemu_xterm_emulation.
type OSCCommandConemuXtermEmulationPayload struct {
	// null => do not change
	// false => turn off
	// true => turn on
	Keyboard Optional[bool]
	// null => do not change
	// false => turn off
	// true => turn on
	Output Optional[bool]
}

// OSCCommandSemanticPrompt is osc.Command.SemanticPrompt:
// parsers.semantic_prompt.Command.
type OSCCommandSemanticPrompt = OSCParsersSemanticPromptCommand

// OSCCommandKittyClipboardProtocol is osc.Command.KittyClipboardProtocol:
// parsers.kitty_clipboard_protocol.OSC.
type OSCCommandKittyClipboardProtocol = OSCParsersKittyClipboardProtocolOSC

// OSCCommandKey is osc.Command.Key: the tag type of Command.
//
// Zig: LibEnum with the .zig target in the library build, i.e. a plain enum
// with these values in this order. NOTE: Order matters, see LibEnum
// documentation.
//
// The constants are named OSCCommandKey<Value> (the enum rule) rather than
// OSCCommand<Value>, because OSCCommandSemanticPrompt and
// OSCCommandKittyClipboardProtocol are already the names of the two type
// aliases above.
type OSCCommandKey uint8

const (
	OSCCommandKeyInvalid OSCCommandKey = iota
	OSCCommandKeyChangeWindowTitle
	OSCCommandKeyChangeWindowIcon
	OSCCommandKeySemanticPrompt
	OSCCommandKeyClipboardContents
	OSCCommandKeyReportPwd
	OSCCommandKeyMouseShape
	OSCCommandKeyColorOperation
	OSCCommandKeyKittyColorProtocol
	OSCCommandKeyShowDesktopNotification
	OSCCommandKeyHyperlinkStart
	OSCCommandKeyHyperlinkEnd
	OSCCommandKeyConemuSleep
	OSCCommandKeyConemuShowMessageBox
	OSCCommandKeyConemuChangeTabTitle
	OSCCommandKeyConemuProgressReport
	OSCCommandKeyConemuWaitInput
	OSCCommandKeyConemuGuimacro
	OSCCommandKeyConemuRunProcess
	OSCCommandKeyConemuOutputEnvironmentVariable
	OSCCommandKeyConemuXtermEmulation
	OSCCommandKeyConemuComment
	OSCCommandKeyKittyTextSizing
	OSCCommandKeyKittyClipboardProtocol
	OSCCommandKeyContextSignal

	oscCommandKeyCount = iota
)

// OSCCommandTag is std.meta.Tag(Command), which is Key. It exists so that
// the tag type of this tagged union can also be named the way every other
// tagged union's is.
type OSCCommandTag = OSCCommandKey

// String returns the Zig tag name (@tagName).
func (self OSCCommandKey) String() string {
	return oscCommandKeyNames[self]
}

var oscCommandKeyNames = [oscCommandKeyCount]string{
	"invalid",
	"change_window_title",
	"change_window_icon",
	"semantic_prompt",
	"clipboard_contents",
	"report_pwd",
	"mouse_shape",
	"color_operation",
	"kitty_color_protocol",
	"show_desktop_notification",
	"hyperlink_start",
	"hyperlink_end",
	"conemu_sleep",
	"conemu_show_message_box",
	"conemu_change_tab_title",
	"conemu_progress_report",
	"conemu_wait_input",
	"conemu_guimacro",
	"conemu_run_process",
	"conemu_output_environment_variable",
	"conemu_xterm_emulation",
	"conemu_comment",
	"kitty_text_sizing",
	"kitty_clipboard_protocol",
	"context_signal",
}

// OSCCommandProgressReport is osc.Command.ProgressReport.
//
// The C ABI declarations (`C`, `cval`) are not ported.
type OSCCommandProgressReport struct {
	State OSCCommandProgressReportState

	// Zig: ?u8, default null.
	Progress Optional[uint8]
}

// OSCCommandProgressReportState is osc.Command.ProgressReport.State.
//
// Zig: enum(c_int).
type OSCCommandProgressReportState int32

const (
	OSCCommandProgressReportStateRemove        OSCCommandProgressReportState = 0
	OSCCommandProgressReportStateSet           OSCCommandProgressReportState = 1
	OSCCommandProgressReportStateError         OSCCommandProgressReportState = 2
	OSCCommandProgressReportStateIndeterminate OSCCommandProgressReportState = 3
	OSCCommandProgressReportStatePause         OSCCommandProgressReportState = 4
)

// String returns the Zig tag name (@tagName).
func (self OSCCommandProgressReportState) String() string {
	return oscCommandProgressReportStateNames[self]
}

var oscCommandProgressReportStateNames = [...]string{
	"remove",
	"set",
	"error",
	"indeterminate",
	"pause",
}

// OSCTerminator is osc.Terminator: the terminator used to end an OSC
// command. For OSC commands that demand a response, we try to match the
// terminator used in the request since that is most likely to be accepted by
// the calling program.
//
// The C ABI declarations (`C`, `cval`) are not ported.
type OSCTerminator uint8

const (
	// The preferred string terminator is ESC followed by \
	OSCTerminatorSt OSCTerminator = 0

	// Some applications and terminals use BELL (0x07) as the string
	// terminator.
	OSCTerminatorBel OSCTerminator = 1
)

// OSCTerminatorInit is osc.Terminator.init: initialize the terminator based
// on the last byte seen. If the last byte is a BEL then we use BEL,
// otherwise we just assume ST.
func OSCTerminatorInit(ch Optional[uint8]) OSCTerminator {
	if !ch.Valid {
		return OSCTerminatorSt
	}
	switch ch.Value {
	case 0x07:
		return OSCTerminatorBel
	default:
		return OSCTerminatorSt
	}
}

// String is osc.Terminator.string: the terminator as a string. This is
// static memory so it doesn't need to be freed.
//
// Note: this is Zig's `string` method (the bytes that end a reply), not the
// tag name; it returns []uint8 like every other `[]const u8`.
func (self OSCTerminator) String() []uint8 {
	switch self {
	case OSCTerminatorSt:
		return []uint8("\x1b\\")
	case OSCTerminatorBel:
		return []uint8("\x07")
	}
	panic("unreachable")
}

// TagName returns the Zig tag name (@tagName): "st" or "bel". Every other
// enum uses String() for this; here String is taken by Zig's own `string`
// function above, which returns the terminator bytes.
func (self OSCTerminator) TagName() string {
	return oscTerminatorNames[self]
}

var oscTerminatorNames = [...]string{
	"st",
	"bel",
}

// Format is osc.Terminator.format: writes the terminator bytes. Per
// docs/mapping.md a Zig `format` method becomes `Format() string` with
// output identical to Zig's.
func (self OSCTerminator) Format() string {
	return string(self.String())
}

// OSCParser is osc.Parser.
type OSCParser struct {
	// Optional allocator used to accept data longer than MAX_BUF.
	// This only applies to some commands (e.g. OSC 52) that can
	// reasonably exceed MAX_BUF.
	//
	// Zig: ?Allocator. Go allocates without an allocator, but whether
	// upstream's is null changes what the parser accepts (ensureAllocator,
	// writeToAllocating, the colour parsers), so the choice is kept as a
	// bool: false is Zig's null (commands that need to grow a buffer are
	// refused), true is "an allocator was provided".
	alloc bool

	// Current state of the parser.
	state OSCParserState

	// Buffer for temporary storage of OSC data
	buffer [OSCParserMAX_BUF]uint8
	// Fixed writer for accumulating OSC data
	//
	// Zig: ?std.Io.Writer.
	fixed Optional[oscWriter]
	// Allocating writer for accumulating OSC data
	//
	// Zig: ?std.Io.Writer.Allocating.
	allocating Optional[oscWriterAllocating]
	// Pointer to the active writer for accumulating OSC data
	//
	// Zig: ?*std.Io.Writer; nil is null. It points at fixed.Value or at
	// allocating.Value.writer.
	writer *oscWriter

	// The command that is the result of parsing.
	command OSCCommand
}

// OSCParserMAX_BUF is osc.Parser.MAX_BUF: maximum size of a "normal" OSC.
const OSCParserMAX_BUF = 2048

// OSCParserState is osc.Parser.State.
type OSCParserState uint8

const (
	OSCParserStateStart OSCParserState = iota
	OSCParserStateInvalid

	// OSC command prefixes. Not all of these are valid OSCs, but may be
	// needed to "bridge" to a valid OSC (e.g. to support OSC 777 we need to
	// have a state "77" even though there is no OSC 77).
	OSCParserState0
	OSCParserState1
	OSCParserState2
	OSCParserState3
	OSCParserState4
	OSCParserState5
	OSCParserState6
	OSCParserState7
	OSCParserState8
	OSCParserState9
	OSCParserState30
	OSCParserState300
	OSCParserState3008
	OSCParserState10
	OSCParserState11
	OSCParserState12
	OSCParserState13
	OSCParserState14
	OSCParserState15
	OSCParserState16
	OSCParserState17
	OSCParserState18
	OSCParserState19
	OSCParserState21
	OSCParserState22
	OSCParserState52
	OSCParserState55
	OSCParserState66
	OSCParserState77
	OSCParserState104
	OSCParserState110
	OSCParserState111
	OSCParserState112
	OSCParserState113
	OSCParserState114
	OSCParserState115
	OSCParserState116
	OSCParserState117
	OSCParserState118
	OSCParserState119
	OSCParserState133
	OSCParserState552
	OSCParserState777
	OSCParserState1337
	OSCParserState5522

	oscParserStateCount = iota
)

// String returns the Zig tag name (@tagName; the "{t}" format in the
// parser's log lines).
func (self OSCParserState) String() string {
	return oscParserStateNames[self]
}

var oscParserStateNames = [oscParserStateCount]string{
	"start",
	"invalid",
	"0",
	"1",
	"2",
	"3",
	"4",
	"5",
	"6",
	"7",
	"8",
	"9",
	"30",
	"300",
	"3008",
	"10",
	"11",
	"12",
	"13",
	"14",
	"15",
	"16",
	"17",
	"18",
	"19",
	"21",
	"22",
	"52",
	"55",
	"66",
	"77",
	"104",
	"110",
	"111",
	"112",
	"113",
	"114",
	"115",
	"116",
	"117",
	"118",
	"119",
	"133",
	"552",
	"777",
	"1337",
	"5522",
}

// OSCParserInit is osc.Parser.init.
//
// Zig leaves `buffer` undefined; Go zeroes it.
func OSCParserInit(alloc bool) OSCParser {
	var result = OSCParser{
		alloc:      alloc,
		state:      OSCParserStateStart,
		fixed:      None[oscWriter](),
		allocating: None[oscWriterAllocating](),
		writer:     nil,
		command:    OSCCommand{Tag: OSCCommandKeyInvalid},
	}
	return result
}

// Deinit is osc.Parser.deinit: this must be called to clean up any allocated
// memory.
func (self *OSCParser) Deinit() {
	self.Reset()
}

// Reset is osc.Parser.reset: reset the parser state.
func (self *OSCParser) Reset() {
	// If we set up an allocating writer, free up that memory.
	if self.allocating.Valid {
		self.allocating.Value.deinit()
	}

	// Handle any cleanup that individual OSCs require.
	switch self.command.Tag {
	case OSCCommandKeyKittyColorProtocol:
		// Zig: v.deinit(self.alloc orelse break :kitty_color_protocol).
		// The allocator argument is dropped; without an allocator
		// nothing is freed.
		if self.alloc {
			self.command.KittyColorProtocol.Deinit()
		}
	case OSCCommandKeyChangeWindowIcon,
		OSCCommandKeyChangeWindowTitle,
		OSCCommandKeyClipboardContents,
		OSCCommandKeyColorOperation,
		OSCCommandKeyConemuChangeTabTitle,
		OSCCommandKeyConemuComment,
		OSCCommandKeyConemuGuimacro,
		OSCCommandKeyConemuOutputEnvironmentVariable,
		OSCCommandKeyConemuProgressReport,
		OSCCommandKeyConemuRunProcess,
		OSCCommandKeyConemuShowMessageBox,
		OSCCommandKeyConemuSleep,
		OSCCommandKeyConemuWaitInput,
		OSCCommandKeyConemuXtermEmulation,
		OSCCommandKeyHyperlinkEnd,
		OSCCommandKeyHyperlinkStart,
		OSCCommandKeyInvalid,
		OSCCommandKeyMouseShape,
		OSCCommandKeyReportPwd,
		OSCCommandKeySemanticPrompt,
		OSCCommandKeyShowDesktopNotification,
		OSCCommandKeyKittyTextSizing,
		OSCCommandKeyKittyClipboardProtocol,
		OSCCommandKeyContextSignal:
	default:
		// Zig's switch is exhaustive; a tag outside the enum traps.
		panic("unreachable")
	}

	self.state = OSCParserStateStart
	self.fixed = None[oscWriter]()
	self.allocating = None[oscWriterAllocating]()
	self.writer = nil
	self.command = OSCCommand{Tag: OSCCommandKeyInvalid}
}

// ensureAllocator is osc.Parser.ensureAllocator: make sure that we have an
// allocator. If we don't, set the state to invalid so that any additional
// OSC data is discarded.
func (self *OSCParser) ensureAllocator() bool {
	if self.alloc {
		return true
	}
	warn("osc", "An allocator is required to process OSC %s but none was provided.", self.state.String())
	self.state = OSCParserStateInvalid
	return false
}

// writeToFixed is osc.Parser.writeToFixed: set up a fixed Writer to collect
// the rest of the OSC data.
func (self *OSCParser) writeToFixed() {
	self.fixed = Some(oscWriterFixed(self.buffer[:]))
	self.writer = &self.fixed.Value
}

// writeToAllocating is osc.Parser.writeToAllocating: set up an allocating
// Writer to collect the rest of the OSC data. If we don't have an allocator
// or setting up the allocator fails, fall back to writing to a fixed buffer
// and hope that it's big enough.
func (self *OSCParser) writeToAllocating() {
	if !self.alloc {
		// We don't have an allocator - fall back to a fixed buffer and hope
		// that it's big enough.
		self.writeToFixed()
		return
	}

	allocating, err := oscWriterAllocatingInitCapacity(2048)
	if err != nil {
		// The allocator failed for some reason, fall back to a fixed buffer
		// and hope that it's big enough.
		self.writeToFixed()
		return
	}
	self.allocating = Some(allocating)

	self.writer = &self.allocating.Value.writer
}

// Next is osc.Parser.next: consume the next character c and advance the
// parser state.
func (self *OSCParser) Next(c uint8) {
	// If the state becomes invalid for any reason, just discard
	// any further input.
	if self.state == OSCParserStateInvalid {
		return
	}

	// If a writer has been initialized, we just accumulate the rest of the
	// OSC sequence in the writer's buffer and skip the state machine.
	if writer := self.writer; writer != nil {
		if err := writer.writeByte(c); err != nil {
			switch err {
			// We have overflowed our buffer or had some other error, set the
			// state to invalid so that we discard any further input.
			case ErrWriteFailed:
				self.state = OSCParserStateInvalid
			default:
				panic("unreachable")
			}
		}
		return
	}

	switch self.state {
	// handled above, so should never be here
	case OSCParserStateInvalid:
		panic("unreachable")

	case OSCParserStateStart:
		switch c {
		case '0':
			self.state = OSCParserState0
		case '1':
			self.state = OSCParserState1
		case '2':
			self.state = OSCParserState2
		case '3':
			self.state = OSCParserState3
		case '4':
			self.state = OSCParserState4
		case '5':
			self.state = OSCParserState5
		case '6':
			self.state = OSCParserState6
		case '7':
			self.state = OSCParserState7
		case '8':
			self.state = OSCParserState8
		case '9':
			self.state = OSCParserState9
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState3:
		switch c {
		case '0':
			self.state = OSCParserState30
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState30:
		switch c {
		case '0':
			self.state = OSCParserState300
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState300:
		switch c {
		case '8':
			self.state = OSCParserState3008
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState3008:
		switch c {
		case ';':
			self.writeToFixed()
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState1:
		switch c {
		case ';':
			self.writeToFixed()
		case '0':
			self.state = OSCParserState10
		case '1':
			self.state = OSCParserState11
		case '2':
			self.state = OSCParserState12
		case '3':
			self.state = OSCParserState13
		case '4':
			self.state = OSCParserState14
		case '5':
			self.state = OSCParserState15
		case '6':
			self.state = OSCParserState16
		case '7':
			self.state = OSCParserState17
		case '8':
			self.state = OSCParserState18
		case '9':
			self.state = OSCParserState19
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState10:
		switch c {
		case ';':
			if self.ensureAllocator() {
				self.writeToFixed()
			}
		case '4':
			self.state = OSCParserState104
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState104:
		switch c {
		case ';':
			if self.ensureAllocator() {
				self.writeToFixed()
			}
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState11:
		switch c {
		case ';':
			if self.ensureAllocator() {
				self.writeToFixed()
			}
		case '0':
			self.state = OSCParserState110
		case '1':
			self.state = OSCParserState111
		case '2':
			self.state = OSCParserState112
		case '3':
			self.state = OSCParserState113
		case '4':
			self.state = OSCParserState114
		case '5':
			self.state = OSCParserState115
		case '6':
			self.state = OSCParserState116
		case '7':
			self.state = OSCParserState117
		case '8':
			self.state = OSCParserState118
		case '9':
			self.state = OSCParserState119
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState4,
		OSCParserState12,
		OSCParserState14,
		OSCParserState15,
		OSCParserState16,
		OSCParserState17,
		OSCParserState18,
		OSCParserState19,
		OSCParserState21,
		OSCParserState110,
		OSCParserState111,
		OSCParserState112,
		OSCParserState113,
		OSCParserState114,
		OSCParserState115,
		OSCParserState116,
		OSCParserState117,
		OSCParserState118,
		OSCParserState119:
		switch c {
		case ';':
			if self.ensureAllocator() {
				self.writeToFixed()
			}
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState13:
		switch c {
		case ';':
			if self.ensureAllocator() {
				self.writeToFixed()
			}
		case '3':
			self.state = OSCParserState133
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState2:
		switch c {
		case ';':
			self.writeToFixed()
		case '1':
			self.state = OSCParserState21
		case '2':
			self.state = OSCParserState22
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState5:
		switch c {
		case ';':
			if self.ensureAllocator() {
				self.writeToFixed()
			}
		case '2':
			self.state = OSCParserState52
		case '5':
			self.state = OSCParserState55
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState6:
		switch c {
		case '6':
			self.state = OSCParserState66
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState52,
		OSCParserState66:
		switch c {
		case ';':
			self.writeToAllocating()
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState55:
		switch c {
		case '2':
			self.state = OSCParserState552
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState7:
		switch c {
		case ';':
			self.writeToFixed()
		case '7':
			self.state = OSCParserState77
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState77:
		switch c {
		case '7':
			self.state = OSCParserState777
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState133:
		switch c {
		case ';':
			self.writeToFixed()
		case '7':
			self.state = OSCParserState1337
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState552:
		switch c {
		case '2':
			self.state = OSCParserState5522
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState1337:
		switch c {
		case ';':
			self.writeToFixed()
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState5522:
		switch c {
		case ';':
			self.writeToAllocating()
		default:
			self.state = OSCParserStateInvalid
		}

	case OSCParserState0,
		OSCParserState22,
		OSCParserState777,
		OSCParserState8,
		OSCParserState9:
		switch c {
		case ';':
			self.writeToFixed()
		default:
			self.state = OSCParserStateInvalid
		}

	default:
		// Zig's switch is exhaustive; a state outside the enum traps.
		panic("unreachable")
	}
}

// End is osc.Parser.end: end the sequence and return the command, if any.
// If the return value is null, then no valid command was found. The optional
// terminator_ch is the final character in the OSC sequence. This is used to
// determine the response terminator.
//
// The returned pointer is only valid until the next call to the parser.
// Callers should copy out any data they wish to retain across calls.
func (self *OSCParser) End(terminator_ch Optional[uint8]) *OSCCommand {
	switch self.state {
	case OSCParserStateStart:
		return nil

	case OSCParserStateInvalid:
		return nil

	case OSCParserState0,
		OSCParserState2:
		return OSCParsersChangeWindowTitleParse(self, terminator_ch)

	case OSCParserState1:
		return OSCParsersChangeWindowIconParse(self, terminator_ch)

	case OSCParserState4,
		OSCParserState5,
		OSCParserState10,
		OSCParserState11,
		OSCParserState12,
		OSCParserState13,
		OSCParserState14,
		OSCParserState15,
		OSCParserState16,
		OSCParserState17,
		OSCParserState18,
		OSCParserState19,
		OSCParserState104,
		OSCParserState110,
		OSCParserState111,
		OSCParserState112,
		OSCParserState113,
		OSCParserState114,
		OSCParserState115,
		OSCParserState116,
		OSCParserState117,
		OSCParserState118,
		OSCParserState119:
		return OSCParsersColorParse(self, terminator_ch)

	case OSCParserState7:
		return OSCParsersReportPwdParse(self, terminator_ch)

	case OSCParserState8:
		return OSCParsersHyperlinkParse(self, terminator_ch)

	case OSCParserState9:
		return OSCParsersOsc9Parse(self, terminator_ch)

	case OSCParserState21:
		return OSCParsersKittyColorParse(self, terminator_ch)

	case OSCParserState22:
		return OSCParsersMouseShapeParse(self, terminator_ch)

	case OSCParserState52:
		return OSCParsersClipboardOperationParse(self, terminator_ch)

	case OSCParserState55:
		return nil

	case OSCParserState3,
		OSCParserState30,
		OSCParserState300:
		return nil

	case OSCParserState3008:
		return OSCParsersContextSignalParse(self, terminator_ch)

	case OSCParserState6:
		return nil

	case OSCParserState66:
		return OSCParsersKittyTextSizingParse(self, terminator_ch)

	case OSCParserState77:
		return nil

	case OSCParserState133:
		return OSCParsersSemanticPromptParse(self, terminator_ch)

	case OSCParserState552:
		return nil

	case OSCParserState777:
		return OSCParsersRxvtExtensionParse(self, terminator_ch)

	case OSCParserState1337:
		return OSCParsersIterm2Parse(self, terminator_ch)

	case OSCParserState5522:
		return OSCParsersKittyClipboardProtocolParse(self, terminator_ch)
	}
	// Zig's switch is exhaustive; a state outside the enum traps.
	panic("unreachable")
}
