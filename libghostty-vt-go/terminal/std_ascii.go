// Ported from Zig 0.15.2's lib/std/ascii.zig: only the functions the port
// reaches (sliding_window.zig's next calls std.ascii.indexOfIgnoreCase).
// Go's standard library has no ASCII-only case-insensitive search
// (strings.EqualFold and bytes.EqualFold fold Unicode), and the match index
// decides every search result, so the functions are ported, including the
// Boyer-Moore-Horspool path std takes for large inputs. Their std test
// blocks are in std_ascii_test.go.

package terminal

// stdASCIIIsUpper is std.ascii.isUpper.
//
// Returns whether the character is an uppercase letter.
func stdASCIIIsUpper(c uint8) bool {
	switch {
	case c >= 'A' && c <= 'Z':
		return true
	default:
		return false
	}
}

// stdASCIIToLower is std.ascii.toLower.
//
// Lowercases the character and returns it as-is if already lowercase or not a letter.
func stdASCIIToLower(c uint8) uint8 {
	var mask uint8
	if stdASCIIIsUpper(c) {
		mask = 1 << 5
	}
	return c | mask
}

// stdASCIIEqlIgnoreCase is std.ascii.eqlIgnoreCase.
//
// Compares strings `a` and `b` case-insensitively and returns whether they are equal.
func stdASCIIEqlIgnoreCase(a []uint8, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i, a_c := range a {
		if stdASCIIToLower(a_c) != stdASCIIToLower(b[i]) {
			return false
		}
	}
	return true
}

// stdASCIIIndexOfIgnoreCase is std.ascii.indexOfIgnoreCase.
//
// Finds `needle` in `haystack`, ignoring case, starting at index 0.
//
// Zig: ?usize.
func stdASCIIIndexOfIgnoreCase(haystack []uint8, needle []uint8) Optional[uint] {
	return stdASCIIIndexOfIgnoreCasePos(haystack, 0, needle)
}

// stdASCIIIndexOfIgnoreCasePos is std.ascii.indexOfIgnoreCasePos.
//
// Finds `needle` in `haystack`, ignoring case, starting at `start_index`.
// Uses Boyer-Moore-Horspool algorithm on large inputs; `indexOfIgnoreCasePosLinear` on small inputs.
//
// Zig: ?usize.
func stdASCIIIndexOfIgnoreCasePos(haystack []uint8, start_index uint, needle []uint8) Optional[uint] {
	if len(needle) > len(haystack) {
		return None[uint]()
	}
	if len(needle) == 0 {
		return Some(start_index)
	}

	if len(haystack) < 52 || len(needle) <= 4 {
		return stdASCIIIndexOfIgnoreCasePosLinear(haystack, start_index, needle)
	}

	var skip_table [256]uint
	stdASCIIBoyerMooreHorspoolPreprocessIgnoreCase(needle, &skip_table)

	i := start_index
	for i <= uint(len(haystack)-len(needle)) {
		if stdASCIIEqlIgnoreCase(haystack[i:i+uint(len(needle))], needle) {
			return Some(i)
		}
		i += skip_table[stdASCIIToLower(haystack[i+uint(len(needle))-1])]
	}

	return None[uint]()
}

// stdASCIIIndexOfIgnoreCasePosLinear is std.ascii.indexOfIgnoreCasePosLinear.
//
// Consider using `indexOfIgnoreCasePos` instead of this, which will automatically use a
// more sophisticated algorithm on larger inputs.
//
// Zig: ?usize. `haystack.len - needle.len` traps on underflow.
func stdASCIIIndexOfIgnoreCasePosLinear(haystack []uint8, start_index uint, needle []uint8) Optional[uint] {
	i := start_index
	assert(len(needle) <= len(haystack)) // Zig: usize subtraction traps
	end := uint(len(haystack) - len(needle))
	for i <= end {
		if stdASCIIEqlIgnoreCase(haystack[i:i+uint(len(needle))], needle) {
			return Some(i)
		}
		i += 1
	}
	return None[uint]()
}

// stdASCIIBoyerMooreHorspoolPreprocessIgnoreCase is
// std.ascii.boyerMooreHorspoolPreprocessIgnoreCase.
//
// Zig: `pattern.len - 1` traps on an empty pattern (the caller never
// passes one).
func stdASCIIBoyerMooreHorspoolPreprocessIgnoreCase(pattern []uint8, table *[256]uint) {
	for i := range table {
		table[i] = uint(len(pattern))
	}

	assert(len(pattern) > 0) // Zig: usize subtraction traps
	var i uint = 0
	// The last item is intentionally ignored and the skip size will be pattern.len.
	// This is the standard way Boyer-Moore-Horspool is implemented.
	for i < uint(len(pattern))-1 {
		table[stdASCIIToLower(pattern[i])] = uint(len(pattern)) - 1 - i
		i += 1
	}
}
