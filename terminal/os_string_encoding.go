// Port of src/os/string_encoding.zig (Ghostty v1.3.1), the two decoders
// that osc/parsers/semantic_prompt.zig calls (Command.writeCommandLine).
//
// File naming: `os/string_encoding.zig` lies outside src/terminal; the
// directory goes into the name as for the subdirectories of src/terminal,
// so the file is `os_string_encoding.go` and its declarations take the
// prefix `OsStringEncoding`.
//
// Not ported: `isValidChar` and `urlPercentEncode` (only
// src/synthetic/Osc.zig uses them, which is not part of libghostty-vt), and
// the file's tests (printf_q 1-10, singles percent, percent 1-7), which are
// not yet translated.
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
