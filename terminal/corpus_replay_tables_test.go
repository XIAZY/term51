package terminal

// The static tables of the corpus replayer: the Go side of the recorder's
// harness/src/tables.zig. Each function writes the same text as its Zig
// original from the Go tables, and TestCorpusReplayTables compares it with
// golden/tables/<name>.txt byte for byte.
//
// TestCorpusParseTable (corpus_test.go) compares the same parse table line by
// line; tablesParseTable is its lifted form, and TestCorpusReplayTables
// subsumes it.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Mirrors harness/src/tables.zig:10 parseTable
//
// The parser state table: one line per (state, byte) in state order then
// byte order: "<state> <byte hex> -> <next state> <action>".
func tablesParseTable(w *bytes.Buffer) {
	for s := 0; s < parserStateCount; s++ {
		for c := 0; c < 256; c++ {
			t := table[c][s]
			fmt.Fprintf(w, "%s %02x -> %s %s\n", ParserState(s), c, t.state, t.action)
		}
	}
}

// Mirrors harness/src/tables.zig:26 defaultPalette
//
// The default 256-colour palette: "<index> <rrggbb>".
func tablesDefaultPalette(w *bytes.Buffer) {
	for i, rgb := range ColorDefault {
		fmt.Fprintf(w, "%d %02x%02x%02x\n", i, rgb.r, rgb.g, rgb.b)
	}
}

// Mirrors harness/src/tables.zig:33 modes
//
// Every mode upstream knows: "<name> <number> <ansi|dec> default=<0|1>".
//
// Zig reads the default from `ModePacked{}` by field name; the Go field is
// not reachable by name, so the default is read through a fresh
// ModesModeState (its `values` is ModesModePackedInit()).
func tablesModes(w *bytes.Buffer) {
	defaults := ModesModeStateInit()
	for _, mode := range ModesModeValues() {
		tag := ModesModeTagFromInt(uint16(mode))
		kind := "dec"
		if tag.ansi {
			kind = "ansi"
		}
		def := 0
		if defaults.Get(mode) {
			def = 1
		}
		fmt.Fprintf(w, "%s %d %s default=%d\n", mode, tag.value, kind, def)
	}
}

// tablesProduce runs a table writer; a panic of the port becomes a line of
// the text, so it is a mismatch like any other.
func tablesProduce(write func(*bytes.Buffer)) (out []byte) {
	var buf bytes.Buffer
	defer func() {
		if r := recover(); r != nil {
			out = []byte(fmt.Sprintf("!panic: %v\n", r))
		}
	}()
	write(&buf)
	return buf.Bytes()
}

// tablesCompare compares a table with its recorded file and reports the
// first differing line (corpusCompare: no skip list, the tables are exact).
func tablesCompare(t *testing.T, name string, got, want []byte) {
	t.Helper()
	diff, equal := corpusCompare(got, want, nil)
	if equal {
		return
	}
	t.Errorf("MISMATCH tables/%s line %d\n  got:  %s\n  want: %s", name, diff.line, diff.got, diff.want)
}

// tablesCheckManifest checks a table file against golden/MANIFEST: its sha256
// and its number of lines.
func tablesCheckManifest(t *testing.T, name string, text []byte) {
	t.Helper()
	key := "tables/" + name + ".txt"
	want, ok := corpusManifest(t)[key]
	if !ok || want.unit != "lines" {
		t.Fatalf("%s is not in golden/MANIFEST with a line count", key)
	}
	if sum := sha256.Sum256(text); hex.EncodeToString(sum[:]) != want.sha256 {
		t.Errorf("%s: sha256 is %x, MANIFEST says %s", key, sum, want.sha256)
	}
	if n := bytes.Count(text, []byte("\n")); n != want.count {
		t.Errorf("%s: %d lines, MANIFEST says lines=%d", key, n, want.count)
	}
}

// TestCorpusReplayTables compares the three static tables with
// golden/tables/{parse_table,default_palette,modes}.txt.
func TestCorpusReplayTables(t *testing.T) {
	cfg, err := corpusConfigFrom(os.Getenv, testing.Short())
	if err != nil {
		t.Fatal(err)
	}
	compared := 0
	for _, tab := range []struct {
		name  string
		write func(*bytes.Buffer)
	}{
		{"parse_table", tablesParseTable},
		{"default_palette", tablesDefaultPalette},
		{"modes", tablesModes},
	} {
		t.Run(tab.name, func(t *testing.T) {
			want := corpusReadAll(t, filepath.Join("golden", "tables", tab.name+".txt"))
			tablesCheckManifest(t, tab.name, want)
			tablesCompare(t, tab.name, tablesProduce(tab.write), want)
			compared++
		})
	}
	corpusFailIfEmpty(t, cfg, "the tables test", compared)
}
