package terminal

// Tests against the recorded data of the pinned upstream libghostty-vt for
// color.zig and x11_color.zig. The files under data/colors/ are produced by
// the Zig test "term51: LAB.fromRgb: dump exact data when
// TERM51_COLOR_DUMP_DIR is set" (not translated: it only writes them); these
// tests regenerate the same values with the Go port and compare the lines
// exactly. Floats are compared as f32 bit patterns. The location of the
// corpus is corpusDir (corpus_test.go); a missing corpus fails the tests.
//
// The test names use a lower-case "corpus" so that the corpus tests of the
// other files (-run Corpus) do not select them.
//
// The generic helpers expect, expectEqual and zigTry are defined in
// Parser_test.go and dcs_test.go; term51Rgb and term51_lab_levels are in
// color_test.go; col_allocLowerString, col_allocUpperString and col_x11Get
// are in x11_color_test.go.

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// corpusColorLines returns the lines of data/colors/<name>.
func corpusColorLines(t *testing.T, name string) []string {
	t.Helper()
	return corpusLines(t, filepath.Join("data", "colors", name))
}

// col_corpusExpectLines compares the regenerated lines with the file's, one
// by one, reporting the first difference.
func col_corpusExpectLines(t *testing.T, file string, want []string, got []string) {
	t.Helper()
	n := len(want)
	if len(got) < n {
		n = len(got)
	}
	for i := 0; i < n; i++ {
		if want[i] != got[i] {
			t.Fatalf("%s line %d:\n  file: %s\n  port: %s", file, i+1, want[i], got[i])
		}
	}
	if len(want) != len(got) {
		t.Fatalf("%s: file has %d lines, the port produced %d", file, len(want), len(got))
	}
}

// The grid of lab_grid.txt: line = "r g b l a b -> r g b" with the floats as
// f32 bit patterns, for every combination of term51_lab_levels in r, g, b
// order (b fastest), of LAB.fromRgb and LAB.toRgb.
func TestColor_corpus_lab_grid(t *testing.T) {
	want := corpusColorLines(t, "lab_grid.txt")
	var got []string
	for _, r := range term51_lab_levels {
		for _, g := range term51_lab_levels {
			for _, b := range term51_lab_levels {
				lab := colorLABFromRgb(ColorRGB{R: r, G: g, B: b})
				back := lab.toRgb()
				got = append(got, fmt.Sprintf("%d %d %d %08x %08x %08x -> %d %d %d",
					r, g, b,
					math.Float32bits(lab.l), math.Float32bits(lab.a), math.Float32bits(lab.b),
					back.R, back.G, back.B))
			}
		}
	}
	expectEqual(t, 22*22*22, len(want))
	col_corpusExpectLines(t, "lab_grid.txt", want, got)
}

// lab_channels.txt: every channel value, each channel taken alone (r, then
// g, then b), then gray (v, v, v); same line format as lab_grid.txt. Added
// after review found that Go fused multiply-adds on arm64 changed
// pow(x, 2.4) for channel values the 22-level grid does not contain (180).
func TestColor_corpus_lab_channels(t *testing.T) {
	want := corpusColorLines(t, "lab_channels.txt")
	var got []string
	for which := 0; which < 4; which++ {
		for i := 0; i < 256; i++ {
			v := uint8(i)
			var c ColorRGB
			switch which {
			case 0:
				c = ColorRGB{R: v, G: 0, B: 0}
			case 1:
				c = ColorRGB{R: 0, G: v, B: 0}
			case 2:
				c = ColorRGB{R: 0, G: 0, B: v}
			default:
				c = ColorRGB{R: v, G: v, B: v}
			}
			lab := colorLABFromRgb(c)
			back := lab.toRgb()
			got = append(got, fmt.Sprintf("%d %d %d %08x %08x %08x -> %d %d %d",
				c.R, c.G, c.B,
				math.Float32bits(lab.l), math.Float32bits(lab.a), math.Float32bits(lab.b),
				back.R, back.G, back.B))
		}
	}
	expectEqual(t, 4*256, len(want))
	col_corpusExpectLines(t, "lab_channels.txt", want, got)
}

// lab_cube.txt: the whole RGB cube through LAB.fromRgb and LAB.toRgb, one
// SHA-256 per red value over the 65,536 colours with that red (g outer, b
// inner), each hashed as the little-endian bit patterns of l, a and b
// followed by toRgb's r, g and b. Line = "<r> <sha256 hex>". The red
// values are hashed in parallel. A mismatch names the red value whose slice
// differs; the file cannot say which colour in it.
func TestColor_corpus_lab_cube(t *testing.T) {
	want := corpusColorLines(t, "lab_cube.txt")
	expectEqual(t, 256, len(want))
	got := make([]string, 256)
	var wg sync.WaitGroup
	reds := make(chan int)
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ri := range reds {
				h := sha256.New()
				var rec [15]uint8
				for gi := 0; gi < 256; gi++ {
					for bi := 0; bi < 256; bi++ {
						lab := colorLABFromRgb(ColorRGB{R: uint8(ri), G: uint8(gi), B: uint8(bi)})
						back := lab.toRgb()
						binary.LittleEndian.PutUint32(rec[0:4], math.Float32bits(lab.l))
						binary.LittleEndian.PutUint32(rec[4:8], math.Float32bits(lab.a))
						binary.LittleEndian.PutUint32(rec[8:12], math.Float32bits(lab.b))
						rec[12] = back.R
						rec[13] = back.G
						rec[14] = back.B
						h.Write(rec[:])
					}
				}
				got[ri] = fmt.Sprintf("%d %x", ri, h.Sum(nil))
			}
		}()
	}
	for ri := 0; ri < 256; ri++ {
		reds <- ri
	}
	close(reds)
	wg.Wait()
	col_corpusExpectLines(t, "lab_cube.txt", want, got)
}

// lab_lerp.txt: a grid of t values between two colors, exact float bits:
// "t l a b" as 8 hex digits each.
func TestColor_corpus_lab_lerp(t *testing.T) {
	want := corpusColorLines(t, "lab_lerp.txt")
	a := colorLABFromRgb(ColorRGB{R: 10, G: 20, B: 200})
	b := colorLABFromRgb(ColorRGB{R: 240, G: 230, B: 15})
	ts := [...]float32{0.0, 0.2, 0.4, 0.6, 0.8, 1.0, 1.0 / 3.0, 2.0 / 3.0, 0.5, 0.1, 0.9}
	var got []string
	for _, t_ := range ts {
		l := colorLABLerp(t_, a, b)
		got = append(got, fmt.Sprintf("%08x %08x %08x %08x",
			math.Float32bits(t_),
			math.Float32bits(l.l),
			math.Float32bits(l.a),
			math.Float32bits(l.b)))
	}
	expectEqual(t, 11, len(want))
	col_corpusExpectLines(t, "lab_lerp.txt", want, got)
}

// generate256.txt: "<case> <index> <rrggbb>" for all 256 entries of
// generate256Color over every theme, base palette, skip mask and
// harmonious flag, in the order the dump writes them.
func TestColor_corpus_generate256(t *testing.T) {
	want := corpusColorLines(t, "generate256.txt")

	themes := [...]struct {
		name string
		bg   ColorRGB
		fg   ColorRGB
	}{
		{"black_white", ColorRGB{R: 0, G: 0, B: 0}, ColorRGB{R: 255, G: 255, B: 255}},
		{"white_black", ColorRGB{R: 255, G: 255, B: 255}, ColorRGB{R: 0, G: 0, B: 0}},
		{"black_black", ColorRGB{R: 0, G: 0, B: 0}, ColorRGB{R: 0, G: 0, B: 0}},
		{"white_white", ColorRGB{R: 255, G: 255, B: 255}, ColorRGB{R: 255, G: 255, B: 255}},
		{"gray_gray", ColorRGB{R: 128, G: 128, B: 128}, ColorRGB{R: 128, G: 128, B: 128}},
		{"near_equal_lo", ColorRGB{R: 127, G: 127, B: 127}, ColorRGB{R: 128, G: 128, B: 128}},
		{"near_equal_hi", ColorRGB{R: 128, G: 128, B: 128}, ColorRGB{R: 127, G: 127, B: 127}},
		{"solarized_dark", ColorRGB{R: 0x00, G: 0x2b, B: 0x36}, ColorRGB{R: 0x83, G: 0x94, B: 0x96}},
		{"solarized_light", ColorRGB{R: 0xfd, G: 0xf6, B: 0xe3}, ColorRGB{R: 0x65, G: 0x7b, B: 0x83}},
		{"dracula", ColorRGB{R: 0x28, G: 0x2a, B: 0x36}, ColorRGB{R: 0xf8, G: 0xf8, B: 0xf2}},
		{"gruvbox_dark", ColorRGB{R: 0x28, G: 0x28, B: 0x28}, ColorRGB{R: 0xeb, G: 0xdb, B: 0xb2}},
		{"nord", ColorRGB{R: 0x2e, G: 0x34, B: 0x40}, ColorRGB{R: 0xd8, G: 0xde, B: 0xe9}},
		{"github_light", ColorRGB{R: 0xff, G: 0xff, B: 0xff}, ColorRGB{R: 0x24, G: 0x29, B: 0x2f}},
		{"red_on_blue", ColorRGB{R: 255, G: 0, B: 0}, ColorRGB{R: 0, G: 0, B: 255}},
		{"dark_gray_on_light_gray", ColorRGB{R: 200, G: 200, B: 200}, ColorRGB{R: 60, G: 60, B: 60}},
		{"yellow_on_navy", ColorRGB{R: 0, G: 0, B: 80}, ColorRGB{R: 255, G: 255, B: 0}},
	}

	theme_base := ColorDefault
	// a second base: every first-16 entry replaced by a distinct value
	for i := 0; i < 16; i++ {
		theme_base[i] = ColorRGB{R: uint8(i * 16), G: uint8(255 - i*16), B: uint8((i * 37) % 256)}
	}
	bases := [...]struct {
		name string
		pal  ColorPalette
	}{
		{"default", ColorDefault},
		{"themed", theme_base},
	}

	some := StaticBitSet256InitEmpty()
	some.set(20)
	some.set(100)
	some.set(231)
	some.set(232)
	masks := [...]struct {
		name string
		mask ColorPaletteMask
	}{
		{"mask_none", StaticBitSet256InitEmpty()},
		{"mask_some", some},
	}

	var got []string
	for _, theme := range themes {
		for _, base := range bases {
			for _, mask := range masks {
				for _, harmonious := range [...]bool{false, true} {
					result := ColorGenerate256Color(base.pal, mask.mask, theme.bg, theme.fg, harmonious)
					mode := "inv"
					if harmonious {
						mode = "harm"
					}
					for i, c := range result {
						got = append(got, fmt.Sprintf("%s/%s/%s/%s %d %02x%02x%02x",
							theme.name, base.name, mask.name, mode, i, c.R, c.G, c.B))
					}
				}
			}
		}
	}
	expectEqual(t, 16*2*2*2*256, len(want))
	col_corpusExpectLines(t, "generate256.txt", want, got)
}

// The X11 color table: x11ColorData must be upstream's res/rgb.txt byte for
// byte, and every line of that file, re-read here independently of
// colorMap(), must resolve in X11ColorMap to its recorded value as written,
// upper-cased and lower-cased.
func TestX11Color_corpus_rgb_txt(t *testing.T) {
	path := filepath.Join(corpusDir(t), "third_party", "ghostty", "src", "terminal", "res", "rgb.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file := string(raw)
	expect(t, x11ColorData == file, "x11ColorData == res/rgb.txt of the pinned upstream")

	count := 0
	for _, line := range strings.Split(file, "\n") {
		if len(line) == 0 {
			continue
		}
		r, err := strconv.ParseUint(strings.Trim(line[0:3], " "), 10, 8)
		zigTry(t, err)
		g, err := strconv.ParseUint(strings.Trim(line[4:7], " "), 10, 8)
		zigTry(t, err)
		b, err := strconv.ParseUint(strings.Trim(line[8:11], " "), 10, 8)
		zigTry(t, err)
		name := strings.Trim(line[12:], " \t\n")
		want := ColorRGB{R: uint8(r), G: uint8(g), B: uint8(b)}

		expectEqual(t, want, zigUnwrap(t, col_x11Get(name)))
		expectEqual(t, want, zigUnwrap(t, col_x11Get(col_allocLowerString(name))))
		expectEqual(t, want, zigUnwrap(t, col_x11Get(col_allocUpperString(name))))
		count += 1
	}
	expectEqual(t, 782, count)
	expectEqual(t, 782, len(X11ColorMap.kvs))
}
