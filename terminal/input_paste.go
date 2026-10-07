// Port of src/input/paste.zig (Ghostty v1.3.1).
//
// File naming: paste.zig lives in src/input/, so the file is
// `input_paste.go` and every declaration takes the prefix `InputPaste`:
// `paste.Options` is `InputPasteOptions`, `paste.Options.fromTerminal` is
// `InputPasteOptionsFromTerminal`, `paste.encode` is `InputPasteEncode`
// (and `InputPasteEncodeMutable`, below), `paste.isSafe` is
// `InputPasteIsSafe`, `paste.Error` is the sentinel ErrMutableRequired.
//
// Declarations appear in the same order as in paste.zig.
//
// # encode
//
// Zig's `encode(data: anytype, opts)` is generic over the type of `data`
// and has two instantiations with different return types:
//   - `[]const u8`: `Error![3][]const u8`. It never modifies `data`; when
//     stripping or newline replacement is needed it returns
//     error.MutableRequired. Go: `InputPasteEncode(data []uint8, opts)
//     ([3][]uint8, error)`, the error being ErrMutableRequired.
//   - `[]u8`: `[3][]const u8`, no error. It replaces the stripped bytes with
//     ' ' and (unbracketed) '\n' with '\r' in place, in `data`. Go:
//     `InputPasteEncodeMutable(data []uint8, opts) [3][]uint8`.
// Go has no const slices, so the instantiation is chosen by name: a Zig
// call with `@as([]const u8, ...)` or a string literal is InputPasteEncode,
// a call with a `[]u8` (a dupe) is InputPasteEncodeMutable. In both, the
// middle slice of the result is `data` itself (same backing array) and the
// outer two are static memory ("" or the bracketed paste fences).
//
// # Errors
//
// `paste.Error` is `error{ MutableRequired }`. Go has no error-set types:
// it is the sentinel ErrMutableRequired.
//
// # Declarations that are not Zig declarations
//
//   - `InputPasteEncodeMutable` (the second instantiation, above);
//   - ErrMutableRequired.
//
// # Omitted
//
// The import aliases `std` and `Terminal` (use `Terminal`).
//
// `encode`'s local constant `strip` is the package-level inputPasteStrip,
// shared by the two instantiations (it is the same comptime constant in
// both). `std.mem.indexOfAny` is stdMemIndexOfAny (std_mem.go),
// `std.mem.indexOfScalar` is bytes.IndexByte, `std.mem.indexOf` with a
// non-empty needle is bytes.Index, `std.mem.replaceScalar` is a loop.
// No C ABI declaration exists in this file.
//
// # Zig defaults a Go literal does not give
//
// None: `Options.bracketed` has no default (it must always be given).

package terminal

import (
	"bytes"
	"errors"
)

// ErrMutableRequired is Zig's error.MutableRequired (paste.Error): returned
// if encoding requires a mutable copy of the data. This can only be
// returned if the input data type is const.
var ErrMutableRequired = errors.New("MutableRequired")

// InputPasteOptions is paste.Options.
type InputPasteOptions struct {
	// True if bracketed paste mode is on.
	//
	// Zig: no default.
	Bracketed bool
}

// InputPasteOptionsFromTerminal is paste.Options.fromTerminal: return the
// encoding options based on the current terminal state.
func InputPasteOptionsFromTerminal(t *Terminal) InputPasteOptions {
	return InputPasteOptions{
		Bracketed: t.Modes.Get(ModesModeBracketedPaste),
	}
}

// inputPasteStrip is encode's local `strip`.
//
// These are the set of byte values that are always replaced by
// a space (per xterm's behavior) for any text insertion method e.g.
// a paste, drag and drop, etc. These are copied directly from xterm's
// source.
var inputPasteStrip = []uint8{
	0x00, // NUL
	0x08, // BS
	0x05, // ENQ
	0x04, // EOT
	0x1B, // ESC
	0x7F, // DEL

	// These can be overridden by the running terminal program
	// via tcsetattr, so they aren't totally safe to hardcode like
	// this. In practice, I haven't seen modern programs change these
	// and its a much bigger architectural change to pass these through
	// so for now they're hardcoded.
	0x03, // VINTR (Ctrl+C)
	0x1C, // VQUIT (Ctrl+\)
	0x15, // VKILL (Ctrl+U)
	0x1A, // VSUSP (Ctrl+Z)
	0x11, // VSTART (Ctrl+Q)
	0x13, // VSTOP (Ctrl+S)
	0x17, // VWERASE (Ctrl+W)
	0x16, // VLNEXT (Ctrl+V)
	0x12, // VREPRINT (Ctrl+R)
	0x0F, // VDISCARD (Ctrl+O)
}

// InputPasteEncode is paste.encode instantiated with `data: []const u8`:
// encode the given data for pasting. The resulting value can be written to
// the pty to perform a paste of the input data.
//
// The data can be either a `[]u8` or a `[]const u8`. If the data
// type is const then `EncodeError` may be returned. If the data type
// is mutable then this function can't return an error.
//
// This slightly complex calling style allows for initially const
// data to be passed in without an allocation, since it is rare in normal
// use cases that the data will need to be modified. In the unlikely case
// data does need to be modified, the caller can make a mutable copy
// after seeing the error.
//
// The data is returned as a set of slices to limit allocations. The caller
// can combine the slices into a single buffer if desired.
//
// WARNING: The input data is not checked for safety. See the `isSafe`
// function to check if the data is safe to paste.
//
// Zig: Error![3][]const u8. Never modifies data; the error is
// ErrMutableRequired. See the file header.
func InputPasteEncode(data []uint8, opts InputPasteOptions) ([3][]uint8, error) {
	// Zig: const mutable = @TypeOf(data) == []u8; false here.

	result := [3][]uint8{[]uint8(""), data, []uint8("")}

	// If we have any of the strip values, then we need to replace them
	// with spaces. This is what xterm does and it does it regardless
	// of bracketed paste mode. This is a security measure to prevent pastes
	// from containing bytes that could be used to inject commands.
	if stdMemIndexOfAny(data, inputPasteStrip).Valid {
		// Zig: if (comptime !mutable) return Error.MutableRequired; the
		// replacement loop after it is the mutable instantiation's only.
		return [3][]uint8{}, ErrMutableRequired
	}

	// Bracketed paste mode (mode 2004) wraps pasted data in
	// fenceposts so that the terminal can ignore things like newlines.
	if opts.Bracketed {
		result[0] = []uint8("\x1b[200~")
		result[2] = []uint8("\x1b[201~")
		return result, nil
	}

	// Non-bracketed. We have to replace newline with `\r`. This matches
	// the behavior of xterm and other terminals. For `\r\n` this will
	// result in `\r\r` which does match xterm.
	//
	// Zig: `if (comptime mutable) { ... } else if (...)`: the const branch.
	if bytes.IndexByte(data, '\n') >= 0 {
		return [3][]uint8{}, ErrMutableRequired
	}

	return result, nil
}

// InputPasteEncodeMutable is paste.encode instantiated with `data: []u8`:
// as InputPasteEncode, but data is modified in place (stripped bytes become
// ' '; unbracketed, '\n' becomes '\r') and there is no error. Not a separate
// Zig declaration; see the file header.
//
// Zig: [3][]const u8.
func InputPasteEncodeMutable(data []uint8, opts InputPasteOptions) [3][]uint8 {
	// Zig: const mutable = @TypeOf(data) == []u8; true here.

	result := [3][]uint8{[]uint8(""), data, []uint8("")}

	// If we have any of the strip values, then we need to replace them
	// with spaces. This is what xterm does and it does it regardless
	// of bracketed paste mode. This is a security measure to prevent pastes
	// from containing bytes that could be used to inject commands.
	if stdMemIndexOfAny(data, inputPasteStrip).Valid {
		// Zig: `if (comptime !mutable) return ...` is false here.
		var offset uint = 0
		for {
			idx := stdMemIndexOfAny(data[offset:], inputPasteStrip)
			if !idx.Valid {
				break
			}
			offset += idx.Value
			data[offset] = ' '
			offset += 1
		}
	}

	// Bracketed paste mode (mode 2004) wraps pasted data in
	// fenceposts so that the terminal can ignore things like newlines.
	if opts.Bracketed {
		result[0] = []uint8("\x1b[200~")
		result[2] = []uint8("\x1b[201~")
		return result
	}

	// Non-bracketed. We have to replace newline with `\r`. This matches
	// the behavior of xterm and other terminals. For `\r\n` this will
	// result in `\r\r` which does match xterm.
	//
	// Zig: `if (comptime mutable)`: std.mem.replaceScalar(u8, data, '\n', '\r').
	for i := range data {
		if data[i] == '\n' {
			data[i] = '\r'
		}
	}

	return result
}

// InputPasteIsSafe is paste.isSafe: returns true if the data looks safe to
// paste. Data is considered unsafe if it contains any of the following:
//
//   - `\n`: Newlines can be used to inject commands.
//   - `\x1b[201~`: This is the end of a bracketed paste. This cane be used
//     to exit a bracketed paste and inject commands.
//
// We consider any scenario unsafe regardless of current terminal state.
// For example, even if bracketed paste mode is not active, we still
// consider `\x1b[201~` unsafe. The existence of these types of bytes
// should raise suspicion that the producer of the paste data is
// acting strangely.
func InputPasteIsSafe(data []uint8) bool {
	return bytes.Index(data, []uint8("\n")) < 0 &&
		bytes.Index(data, []uint8("\x1b[201~")) < 0
}
