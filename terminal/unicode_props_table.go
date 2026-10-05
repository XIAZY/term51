// Port of src/unicode/props_table.zig (Ghostty v1.3.1).
//
// Naming: callers use the table through the unicode namespace
// (unicode/main.zig: `pub const table = @import("props_table.zig").table`),
// so it is `UnicodeTable`, the namespace name; `unicode.table.get(cp)` is
// `UnicodeTable.Get(cp)`.
//
// The data: in Zig the three stages come from the generated module
// `unicode_tables` (src/build/UnicodeTables.zig runs
// src/unicode/props_uucode.zig at build time, which fills lut.Generator
// from the uucode 0.2.0 package's data and Ghostty's width rule in
// src/build/uucode_config.zig, and writes the result with
// lut.Tables.writeZig). The Go port does not regenerate them (project lead
// decision): res/unicode_tables.bin is the build's stage1, stage2 and
// stage3, dumped from the pinned Zig build by term51's
// tools/dump_unicode.sh (gated test "term51: table: dump exact data when
// TERM51_UNICODE_DUMP_DIR is set" in props_table.zig) and embedded byte for
// byte. Its layout, for stage1, stage2 and stage3 in turn: a u32 little
// endian length, then each entry as a u16 little endian (stage3: the
// Properties bit pattern, UnicodeProperties.Int). The corpus tests compare
// the file with term51's data/unicode/unicode_tables.bin, table.get for
// every u21 codepoint with props_runs.txt and props_sha256.txt, and
// WriteZig with the generated Zig file (unicode_tables.zig.txt).
//
// Zig builds the table at compile time; Go decodes the file once at
// start-up (package-level var, mapping.md: comptime tables).

package terminal

import (
	_ "embed"
	"encoding/binary"
)

// unicodePropsTableData is res/unicode_tables.bin (see the file comment).
//
//go:embed res/unicode_tables.bin
var unicodePropsTableData []byte

// UnicodeTable is unicode.table: the lookup tables for Ghostty.
var UnicodeTable = unicodePropsTableInit(unicodePropsTableData)

// unicodePropsTableInit decodes res/unicode_tables.bin. It is the Go
// counterpart of `@import("unicode_tables").Tables(Properties)` bound to
// lut.Tables(Properties); a malformed file stops the program at start-up.
func unicodePropsTableInit(data []byte) UnicodeLutTables[UnicodeProperties] {
	stage := func() []uint16 {
		if len(data) < 4 {
			panic("unicode_props_table: truncated res/unicode_tables.bin")
		}
		n := int(binary.LittleEndian.Uint32(data))
		data = data[4:]
		if len(data) < 2*n {
			panic("unicode_props_table: truncated res/unicode_tables.bin")
		}
		s := make([]uint16, n)
		for i := range s {
			s[i] = binary.LittleEndian.Uint16(data[2*i:])
		}
		data = data[2*n:]
		return s
	}
	stage1 := stage()
	stage2 := stage()
	raw3 := stage()
	if len(data) != 0 {
		panic("unicode_props_table: trailing bytes in res/unicode_tables.bin")
	}
	// The generator covers every u21 codepoint in blocks of 256
	// (lut.Generator), and every index must be in range: Zig's arrays are
	// bounds-checked at compile time against the generated sizes.
	if len(stage1) != 0x200000/256 {
		panic("unicode_props_table: stage1 does not have 8192 entries")
	}
	for _, v := range stage1 {
		if int(v)+256 > len(stage2) {
			panic("unicode_props_table: stage1 entry out of range")
		}
	}
	stage3 := make([]UnicodeProperties, len(raw3))
	for i, v := range raw3 {
		if v > 0x1FF || UucodeXTypesGraphemeBreakNoControl(v>>3&0x1F) > UucodeXTypesGraphemeBreakNoControlIndicConjunctBreakConsonant {
			panic("unicode_props_table: invalid Properties in res/unicode_tables.bin")
		}
		stage3[i] = UnicodePropertiesFromInt(v)
	}
	for _, v := range stage2 {
		if int(v) >= len(stage3) {
			panic("unicode_props_table: stage2 entry out of range")
		}
	}
	return UnicodeLutTables[UnicodeProperties]{
		stage1: stage1,
		stage2: stage2,
		stage3: stage3,
	}
}
