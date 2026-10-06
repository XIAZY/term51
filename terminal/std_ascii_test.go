// Tests translated from Zig 0.15.2's lib/std/ascii.zig (the toolchain that
// built the recorded corpus), one Go test per Zig test of a ported
// function, each with the exact Zig title in a "// Zig: test" comment.
// scripts/check-test-mapping.py does not check this file: it maps Ghostty
// source files, not std files.
//
// Ported (std_ascii.go): isUpper, toLower, eqlIgnoreCase,
// indexOfIgnoreCase, indexOfIgnoreCasePos, indexOfIgnoreCasePosLinear,
// boyerMooreHorspoolPreprocessIgnoreCase. Of their test blocks, `test
// eqlIgnoreCase` and `test indexOfIgnoreCase` are translated. `test "ASCII
// character classes"` also covers isUpper and toLower but mostly the
// other, unported classifiers; its isUpper/toLower lines are translated in
// TestStdASCII_ASCII_character_classes and the others are listed there.

package terminal

import "testing"

// Zig: test "ASCII character classes" (lib/std/ascii.zig)
func TestStdASCII_ASCII_character_classes(t *testing.T) {
	// Only the lines of the ported functions (isUpper, toLower); the lines
	// of isControl, toUpper, isLower, isAlphanumeric, isAlphabetic,
	// isWhitespace and the other classifiers are not translated (not
	// ported).
	expect(t, !stdASCIIIsUpper('z'), "!isUpper('z')")
	expect(t, !stdASCIIIsUpper(0x80), "!isUpper(0x80)")
	expect(t, !stdASCIIIsUpper(0xff), "!isUpper(0xff)")

	expect(t, 'c' == stdASCIIToLower('C'), "'c' == toLower('C')")
	expect(t, ':' == stdASCIIToLower(':'), "':' == toLower(':')")
	expect(t, '\xab' == stdASCIIToLower('\xab'), "'\\xab' == toLower('\\xab')")
}

// Zig: test eqlIgnoreCase (lib/std/ascii.zig)
func TestStdASCII_eqlIgnoreCase(t *testing.T) {
	expect(t, stdASCIIEqlIgnoreCase([]uint8("HEl💩Lo!"), []uint8("hel💩lo!")), `eqlIgnoreCase("HEl💩Lo!", "hel💩lo!")`)
	expect(t, !stdASCIIEqlIgnoreCase([]uint8("hElLo!"), []uint8("hello! ")), `!eqlIgnoreCase("hElLo!", "hello! ")`)
	expect(t, !stdASCIIEqlIgnoreCase([]uint8("hElLo!"), []uint8("helro!")), `!eqlIgnoreCase("hElLo!", "helro!")`)
}

// Zig: test indexOfIgnoreCase (lib/std/ascii.zig)
func TestStdASCII_indexOfIgnoreCase(t *testing.T) {
	idx := func(h, n string) Optional[uint] { return stdASCIIIndexOfIgnoreCase([]uint8(h), []uint8(n)) }
	expect(t, idx("one Two Three Four", "foUr") == Some(uint(14)), `indexOfIgnoreCase("one Two Three Four", "foUr").? == 14`)
	expect(t, !idx("one two three FouR", "gOur").Valid, `indexOfIgnoreCase("one two three FouR", "gOur") == null`)
	expect(t, idx("foO", "Foo") == Some(uint(0)), `indexOfIgnoreCase("foO", "Foo").? == 0`)
	expect(t, !idx("foo", "fool").Valid, `indexOfIgnoreCase("foo", "fool") == null`)
	expect(t, idx("FOO foo", "fOo") == Some(uint(0)), `indexOfIgnoreCase("FOO foo", "fOo").? == 0`)

	expect(t, idx("one two three four five six seven eight nine ten eleven", "ThReE fOUr") == Some(uint(8)), `indexOfIgnoreCase(..., "ThReE fOUr").? == 8`)
	expect(t, !idx("one two three four five six seven eight nine ten eleven", "Two tWo").Valid, `indexOfIgnoreCase(..., "Two tWo") == null`)
}
