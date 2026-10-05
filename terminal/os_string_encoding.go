// Port of src/os/string_encoding.zig (Ghostty v1.3.1). Within
// libghostty-vt only osc/parsers/semantic_prompt.zig uses it
// (Command.writeCommandLine calls printfQDecode and urlPercentDecode);
// urlPercentEncode is used only by src/synthetic, but the whole file is
// ported.
//
// File naming: string_encoding.zig is in src/os, outside src/terminal; it
// is in package terminal because semantic_prompt.zig uses it. As for
// datastruct_circ_buf.go, the directory is part of the name: the file is
// `os_string_encoding.go` and its prefix is `OsStringEncoding`. `pub fn`s
// are exported, private functions take the prefix in lower case
// (`osStringEncodingHex`, `osStringEncodingIsValidChar`).
//
// Declarations appear in the same order as in string_encoding.zig. The
// file's tests are in os_string_encoding_test.go.
//
// The writer is a *std.Io.Writer; see oscWriter in osc.go. The error set
// std.Io.Writer.Error || error{DecodeError} is ErrWriteFailed (osc.go) or
// ErrDecodeError (osc_parsers_semantic_prompt.go).

package terminal

import "bytes"

// OsStringEncodingPrintfQDecode is os.string_encoding.printfQDecode: decode
// date from the buffer that has been encoded in the same way that `bash`'s
// `printf %q` encodes a string and write it to the writer. If an error is
// returned garbage may have been written to the buffer.
func OsStringEncodingPrintfQDecode(writer *oscWriter, buf []uint8) error {
	var data []uint8
	{
		// Strip off `$''` quoting.
		if bytes.HasPrefix(buf, []uint8("$'")) {
			if len(buf) < 3 || !bytes.HasSuffix(buf, []uint8("'")) {
				return ErrDecodeError
			}
			data = buf[2 : len(buf)-1]
			goto data
		}
		// Strip off `''` quoting.
		if bytes.HasPrefix(buf, []uint8("'")) {
			if len(buf) < 2 || !bytes.HasSuffix(buf, []uint8("'")) {
				return ErrDecodeError
			}
			data = buf[1 : len(buf)-1]
			goto data
		}
		data = buf
	}
data:

	var src int

	for src < len(data) {
		switch data[src] {
		default:
			if err := writer.writeByte(data[src]); err != nil {
				return err
			}
			src += 1
		case '\\':
			if src+1 >= len(data) {
				return ErrDecodeError
			}
			switch c := data[src+1]; c {
			case ' ',
				'\\',
				'"',
				'\'',
				'$':
				if err := writer.writeByte(c); err != nil {
					return err
				}
				src += 2
			case 'e':
				if err := writer.writeByte(0x1b); err != nil { // std.ascii.control_code.esc
					return err
				}
				src += 2
			case 'n':
				if err := writer.writeByte(0x0a); err != nil { // std.ascii.control_code.lf
					return err
				}
				src += 2
			case 'r':
				if err := writer.writeByte(0x0d); err != nil { // std.ascii.control_code.cr
					return err
				}
				src += 2
			case 't':
				if err := writer.writeByte(0x09); err != nil { // std.ascii.control_code.ht
					return err
				}
				src += 2
			case 'v':
				if err := writer.writeByte(0x0b); err != nil { // std.ascii.control_code.vt
					return err
				}
				src += 2
			default:
				return ErrDecodeError
			}
		}
	}
	return nil
}

// OsStringEncodingUrlPercentDecode is os.string_encoding.urlPercentDecode:
// decode data from the buffer that has been URL percent encoded and write
// it to the given buffer. If an error is returned the garbage may have been
// written to the writer.
func OsStringEncodingUrlPercentDecode(writer *oscWriter, buf []uint8) error {
	var src int
	for src < len(buf) {
		switch buf[src] {
		default:
			if err := writer.writeByte(buf[src]); err != nil {
				return err
			}
			src += 1
		case '%':
			if src+2 >= len(buf) {
				return ErrDecodeError
			}
			switch c := buf[src+1]; {
			case '0' <= c && c <= '9', 'a' <= c && c <= 'f', 'A' <= c && c <= 'F':
				switch c := buf[src+2]; {
				case '0' <= c && c <= '9', 'a' <= c && c <= 'f', 'A' <= c && c <= 'F':
					// std.math.shl(u8, hex(...), 4): the u4 is widened to u8
					// before the shift, so no bit is lost.
					if err := writer.writeByte(osStringEncodingHex(buf[src+1])<<4 | osStringEncodingHex(buf[src+2])); err != nil {
						return err
					}
					src += 3
				default:
					return ErrDecodeError
				}
			default:
				return ErrDecodeError
			}
		}
	}
	return nil
}

// osStringEncodingHex is os.string_encoding.hex. Zig returns a u4; the
// result here is a uint8 that is always below 16.
func osStringEncodingHex(c uint8) uint8 {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	default:
		panic("unreachable")
	}
}

// osStringEncodingIsValidChar is os.string_encoding.isValidChar: is the
// given character valid in URI percent encoding?
func osStringEncodingIsValidChar(c uint8) bool {
	switch c {
	case ' ', ';', '=':
		return false
	default:
		// std.ascii.isPrint: isAscii(c) and !isControl(c).
		return c < 0x80 && !(c <= 0x1f || c == 0x7f)
	}
}

// OsStringEncodingUrlPercentEncode is os.string_encoding.urlPercentEncode:
// write data to the writer after URI percent encoding. Zig's error set is
// std.Io.Writer.Error, i.e. ErrWriteFailed (osc.go).
func OsStringEncodingUrlPercentEncode(writer *oscWriter, data []uint8) error {
	if err := osStringEncodingUriComponentPercentEncode(writer, data, osStringEncodingIsValidChar); err != nil {
		return err
	}
	return nil
}

// osStringEncodingUriComponentPercentEncode is std.Uri.Component.percentEncode
// (lib/std/Uri.zig), which urlPercentEncode calls. Not a function of
// string_encoding.zig: it is the port of the standard-library function.
// Zig's comptime function parameter isValidChar is a Go func value.
//
// Zig writes each escape with `w.print("{s}%{X:0>2}", .{ raw[start..index],
// char })`; the port writes the same bytes in the same order: the pending
// slice, '%', then the two upper-case hex digits (the zero padding).
func osStringEncodingUriComponentPercentEncode(w *oscWriter, raw []uint8, isValidChar func(uint8) bool) error {
	const hexDigits = "0123456789ABCDEF"
	var start int
	for index, char := range raw {
		if isValidChar(char) {
			continue
		}
		if err := w.writeAll(raw[start:index]); err != nil {
			return err
		}
		if err := w.writeAll([]uint8("%")); err != nil {
			return err
		}
		if err := w.writeAll([]uint8{hexDigits[char>>4], hexDigits[char&0xf]}); err != nil {
			return err
		}
		start = index + 1
	}
	if err := w.writeAll(raw[start:]); err != nil {
		return err
	}
	return nil
}
