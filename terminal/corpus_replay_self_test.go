package terminal

// Tests of the corpus replayer's own plumbing (corpus_replay*_test.go). None
// of them needs the port: they run today. `go test ./terminal -run
// CorpusReplaySelf`.
//
// Where a check compares with an expected text, the text is a recorded line of
// the golden files (dumpSelfGoldenLines) as far as the port is not needed to
// reproduce it.
//
// TODO(recorded): the checks that still use hand-written values, and why:
//   - TestCorpusReplaySelfDumpAny: the types are made up for the test (a
//     dumper has no recorded input of its own); the bool sets, flags, scrolling
//     region and colors of the real types are in
//     TestCorpusReplaySelfDumpAgainstGolden, from recorded lines.
//   - TestCorpusReplaySelfWriters: isPlainAscii, cellIsDefault and
//     cellAttrsDefault on hand-built cells: the recorded lines only show the
//     result of these functions on whole rows, which needs the port's pages
//     (page.GetCells and the row iterator are not implemented). The `\x5c`
//     (backslash) and control-byte escapes of writeEscaped: the golden files
//     have only high bytes (`\x9c`, `\xfb`) in the first sections. The
//     `underline:` flag of writeStyle and the cell lines with `wide=` or
//     `sem=`: their text is an enum name, and the String() of the enums is
//     the port's. The cell lines with `style=`, `g=` and `link=` need a page
//     (dumpScreenCell reads the style, grapheme and hyperlink sets).
//   - TestCorpusReplaySelfNames: zigName and zigCamel on made-up names; the
//     recorded names are used through dumpSelfSetBools (zigName) and
//     TestCorpusReplaySelfUnionsAgainstZig (zigCamel, zigFieldName), against
//     the Zig source rather than the recorded dumps, because the dumps of
//     tagged unions are the action traces (kind actions), stage 2b.
//   - dumpUnwrap and boolInt (in TestCorpusReplaySelfWriters): no recorded
//     counterpart (they are Zig's `.?` and @intFromBool).
//   - the selection, report, config and parsing tests in this file and in
//     corpus_replay_self2_test.go build their inputs: they test the
//     replayer's own logic, which the recorder does not have (sampling,
//     reports), or the recorder's functions on inputs written to hit an edge
//     (parseSuite: the corpus case files, which the test also parses, are
//     covered by TestCorpusReplaySelfParseAllAreas).

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The corpus as counted by docs/replayer-design.md section 2: suites and
// cases per area.
var corpusSelfCounts = map[string][2]int{
	"csi":    {45, 52898},
	"dcs":    {1, 648},
	"esc":    {4, 2032},
	"fuzz":   {9, 30900},
	"modes":  {8, 4773},
	"osc":    {7, 4192},
	"parser": {27, 78397},
	"resize": {8, 4116},
	"scroll": {6, 5345},
	"text":   {13, 6803},
}

// Every case file of every area parses; its cases equal the cases= of every
// output file (golden, actions, actions_noalloc, parser) in golden/MANIFEST;
// the totals are the corpus's; the markers are the three kinds present.
func TestCorpusReplaySelfParseAllAreas(t *testing.T) {
	manifest := corpusManifest(t)
	if len(corpusAreas) != len(corpusSelfCounts) {
		t.Fatalf("%d areas, want %d", len(corpusAreas), len(corpusSelfCounts))
	}
	totalSuites, totalCases := 0, 0
	markers := map[replayOpTag]int{}
	kindFiles := [replayKindCount]int{}
	for _, area := range corpusAreas {
		suites, cases := 0, 0
		for _, suite := range corpusSuiteNames(t, area) {
			parsed := corpusReadSuite(t, area, suite)
			corpusCheckManifest(t, manifest, area, suite, parsed)
			suites++
			cases += len(parsed.cases)
			for _, k := range parsed.kinds.kinds() {
				kindFiles[k]++
			}
			for _, c := range parsed.cases {
				for _, op := range c.ops {
					switch op.Tag {
					case replayOpExpectCrash, replayOpUpstreamInvalid, replayOpTransientInvariant:
						markers[op.Tag]++
					}
				}
			}
		}
		want := corpusSelfCounts[area.name]
		if suites != want[0] || cases != want[1] {
			t.Errorf("%s: %d suites %d cases, want %d suites %d cases", area.name, suites, cases, want[0], want[1])
		}
		totalSuites += suites
		totalCases += cases
	}
	if totalSuites != 128 || totalCases != 190104 {
		t.Errorf("%d suites %d cases, want 128 and 190104", totalSuites, totalCases)
	}
	if markers[replayOpExpectCrash] != 114 || markers[replayOpTransientInvariant] != 951 || markers[replayOpUpstreamInvalid] != 0 {
		t.Errorf("markers expect_crash=%d transient_invariant=%d upstream_invalid=%d, want 114 951 0",
			markers[replayOpExpectCrash], markers[replayOpTransientInvariant], markers[replayOpUpstreamInvalid])
	}
	t.Logf("%d suites, %d cases; output files per kind %v", totalSuites, totalCases, kindFiles)
	// MANIFEST has no file that is not a suite output, but the files of the
	// other recorders.
	for path := range manifest {
		area, file, _ := strings.Cut(path, "/")
		if area == "input" || area == "tables" || area == "search" {
			// Not terminal areas: the key and paste encoders, the static
			// tables, and the search recorder (harness/src/search.zig).
			continue
		}
		known := false
		for _, a := range corpusAreas {
			known = known || a.name == area
		}
		if !known || !strings.HasSuffix(file, ".gz") {
			t.Errorf("MANIFEST entry %q is not in an area", path)
		}
	}
}

func TestCorpusReplaySelfParseSuite(t *testing.T) {
	text := "# comment\r\n" +
		"kinds golden actions parser\n" +
		"  init 10 5 10000  \n" +
		"feed 4142\n" +
		"\n" +
		"case a\n" +
		"transient_invariant\n" +
		"init 80 24 0\n" +
		"feed\n" +
		"feed 1b5B\n" +
		"resize 3 4\n" +
		"scroll top\n" +
		"scroll active\n" +
		"scroll delta -3\n" +
		"dump mid\n" +
		"dump\n" +
		"case b\n" +
		"expect_crash\n" +
		"upstream_invalid\n" +
		"case empty\n"
	got, err := parseSuite(text)
	if err != nil {
		t.Fatal(err)
	}
	wantKinds := replayKindSetOf(replayKindGolden, replayKindActions, replayKindParser)
	if got.kinds != wantKinds {
		t.Errorf("kinds %v, want %v", got.kinds.kinds(), wantKinds.kinds())
	}
	want := []replayNamed{
		{"default", []replayOp{
			{Tag: replayOpInit, Init: replayOpInitArgs{10, 5, 10000}},
			{Tag: replayOpFeed, Feed: []uint8("AB")},
		}},
		{"a", []replayOp{
			{Tag: replayOpTransientInvariant},
			{Tag: replayOpInit, Init: replayOpInitArgs{80, 24, 0}},
			{Tag: replayOpFeed, Feed: []uint8{}},
			{Tag: replayOpFeed, Feed: []uint8{0x1b, 0x5b}},
			{Tag: replayOpResize, Resize: replayOpResizeArgs{3, 4}},
			{Tag: replayOpScrollTop},
			{Tag: replayOpScrollActive},
			{Tag: replayOpScrollDelta, ScrollDelta: -3},
			{Tag: replayOpDump, Dump: "mid"},
			{Tag: replayOpDump, Dump: "end"},
		}},
		{"b", []replayOp{{Tag: replayOpExpectCrash}, {Tag: replayOpUpstreamInvalid}}},
		{"empty", nil},
	}
	if !reflect.DeepEqual(got.cases, want) {
		t.Errorf("cases\n got  %+v\n want %+v", got.cases, want)
	}

	// No cases at all: one empty "default" case, like the Zig. The default
	// kinds are golden and actions.
	got, err = parseSuite("# nothing\n")
	if err != nil || len(got.cases) != 1 || got.cases[0].name != "default" || len(got.cases[0].ops) != 0 {
		t.Errorf("empty suite: %+v, %v", got, err)
	}
	if got.kinds != replayKindSetOf(replayKindGolden, replayKindActions) {
		t.Errorf("default kinds %v", got.kinds.kinds())
	}

	for _, bad := range []string{
		"feed 123\n", "feed zz\n", "scroll sideways\n", "scroll\n", "scroll delta\n", "frob\n",
		"init 1 2\n", "init 70000 1 1\n", "init 1 1 -1\n", "resize 1\n", "case\n", "kinds nope\n",
	} {
		if _, err := parseSuite(bad); err == nil {
			t.Errorf("parseSuite(%q) succeeded", bad)
		}
	}
	if _, err := parseSuite("frob\n"); !errors.Is(err, errReplayBadCase) {
		t.Errorf("unknown op: %v, want BadCase", err)
	}
}

func TestCorpusReplaySelfSelect(t *testing.T) {
	// i*(total-1)/(n-1): the first and the last case are always in.
	want := []int{0, 69, 139, 209, 279, 348, 418, 488, 558, 628, 697, 767, 837, 907, 977, 1046, 1116, 1186, 1256, 1326}
	if got := corpusSelect(1327, 20, nil); !reflect.DeepEqual(got, want) {
		t.Errorf("corpusSelect(1327, 20) = %v, want %v", got, want)
	}
	if got := corpusSelect(1327, 3, nil); !reflect.DeepEqual(got, []int{0, 663, 1326}) {
		t.Errorf("corpusSelect(1327, 3) = %v", got)
	}
	// Deterministic, sorted, without duplicates, markers added.
	for _, c := range []struct{ total, n int }{{1, 20}, {2, 20}, {20, 20}, {21, 20}, {52898, 20}, {100, 1}, {100, 2}, {100, 0}} {
		a := corpusSelect(c.total, c.n, []int{5, 5, 99})
		b := corpusSelect(c.total, c.n, []int{99, 5, 5})
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%v: not deterministic: %v %v", c, a, b)
		}
		for i := 1; i < len(a); i++ {
			if a[i] <= a[i-1] {
				t.Errorf("%v: not increasing: %v", c, a)
			}
		}
		for _, v := range a {
			if v < 0 || v >= c.total {
				t.Errorf("%v: index %d out of range", c, v)
			}
		}
		if c.total > 5 && c.n >= 1 && c.n < c.total {
			found := false
			for _, v := range a {
				found = found || v == 5
			}
			if !found {
				t.Errorf("%v: marked case 5 missing from %v", c, a)
			}
		}
		if c.n >= 1 && c.total >= 1 && a[0] != 0 {
			t.Errorf("%v: first case missing from %v", c, a)
		}
	}
	if got := corpusSelect(5, 20, nil); !reflect.DeepEqual(got, []int{0, 1, 2, 3, 4}) {
		t.Errorf("n > total: %v", got)
	}
	if got := corpusSelect(5, 0, nil); len(got) != 5 {
		t.Errorf("n = 0 is all: %v", got)
	}
	if got := corpusSelect(0, 20, nil); got != nil {
		t.Errorf("empty suite: %v", got)
	}
	if got := corpusSelect(100, 1, []int{7, 200}); !reflect.DeepEqual(got, []int{0, 7}) {
		t.Errorf("one case and a marker: %v", got)
	}

	// The first case of each marker kind of a suite.
	cases := []replayNamed{
		{"c0", nil},
		{"c1", []replayOp{{Tag: replayOpTransientInvariant}}},
		{"c2", []replayOp{{Tag: replayOpExpectCrash}}},
		{"c3", []replayOp{{Tag: replayOpTransientInvariant}}},
		{"c4", []replayOp{{Tag: replayOpExpectCrash}}},
	}
	if got := corpusFirstMarked(cases); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Errorf("corpusFirstMarked = %v", got)
	}

	// On a real suite: the marked case of fuzz/resize_scrolled is selected.
	area := corpusAreaByName(t, "fuzz")
	parsed := corpusReadSuite(t, area, "resize_scrolled")
	marked := corpusFirstMarked(parsed.cases)
	if len(marked) != 1 {
		t.Fatalf("resize_scrolled marked cases %v", marked)
	}
	sel := corpusSelect(len(parsed.cases), 3, marked)
	found := false
	for _, v := range sel {
		found = found || v == marked[0]
	}
	if !found || len(sel) < 3 {
		t.Errorf("selection %v lacks the marked case %v", sel, marked)
	}
}

func TestCorpusReplaySelfConfig(t *testing.T) {
	env := func(kv ...string) func(string) string {
		m := map[string]string{}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return func(k string) string { return m[k] }
	}
	cfg, err := corpusConfigFrom(env(), false)
	if err != nil || cfg.sample != 20 || cfg.full || cfg.short || cfg.exhaustive() || cfg.suiteMax() != 10 || cfg.testMax() != 50 {
		t.Errorf("defaults: %+v %v", cfg, err)
	}
	if !reflect.DeepEqual(cfg.skip(), []string{"layout."}) {
		t.Errorf("default skip list %v", cfg.skip())
	}
	cfg, _ = corpusConfigFrom(env(), true)
	if cfg.sample != 3 {
		t.Errorf("-short sample %d", cfg.sample)
	}
	cfg, _ = corpusConfigFrom(env("TERM51_SAMPLE", "7"), true)
	if cfg.sample != 7 {
		t.Errorf("TERM51_SAMPLE with -short: %d", cfg.sample)
	}
	cfg, _ = corpusConfigFrom(env("TERM51_FULL", "1", "TERM51_STRICT", "1", "TERM51_REPORT_MAX", "4",
		"TERM51_FAIL_LIST", "f", "TERM51_DIFF_DIR", "d"), true)
	if !cfg.full || !cfg.exhaustive() || cfg.skip() != nil || cfg.suiteMax() != 4 || cfg.testMax() != 20 ||
		cfg.failList != "f" || cfg.diffDir != "d" {
		t.Errorf("full: %+v", cfg)
	}
	cfg, err = corpusConfigFrom(env("TERM51_ONLY", "^csi/cup/filled_.*_c1$"), false)
	if err != nil || !cfg.exhaustive() || !cfg.wants("csi", "cup", "filled_H_c1") || cfg.wants("csi", "cup", "filled_H_c10") ||
		cfg.wants("csi", "cub", "filled_H_c1") {
		t.Errorf("only regexp: %+v %v", cfg, err)
	}
	list := filepath.Join(t.TempDir(), "fails")
	if err := os.WriteFile(list, []byte("csi/cup:filled_H_c1:golden\ncsi/cup:a:b:actions\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = corpusConfigFrom(env("TERM51_ONLY", "@"+list), false)
	if err != nil || !cfg.exhaustive() || !cfg.wants("csi", "cup", "filled_H_c1") || !cfg.wants("csi", "cup", "a:b") ||
		cfg.wants("csi", "cup", "a") {
		t.Errorf("only list: %+v %v", cfg, err)
	}
	for _, bad := range [][]string{{"TERM51_SAMPLE", "x"}, {"TERM51_REPORT_MAX", "-1"}, {"TERM51_ONLY", "("}, {"TERM51_ONLY", "@/nonexistent/x"}} {
		if _, err := corpusConfigFrom(env(bad...), false); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
}

func TestCorpusReplaySelfCompare(t *testing.T) {
	want := "## case x\n== dump end\nreplies=\nterm.cols=10\n[screen primary]\nscreen.cursor.x=1\npages.cols=10\nlayout.pages.nodes=10x5/10x4246\nrow 0: dirty\n t 0: \"ABC\"\nrow 1:\n"
	skip := []string{"layout."}

	if d, ok := corpusCompare([]byte(want), []byte(want), skip); !ok {
		t.Errorf("equal texts differ: %+v", d)
	}

	// A difference in a skipped line only.
	layout := strings.Replace(want, "10x4246", "10x4000", 1)
	if d, ok := corpusCompare([]byte(layout), []byte(want), skip); !ok {
		t.Errorf("layout line not skipped: %+v", d)
	}
	if d, ok := corpusCompare([]byte(layout), []byte(want), nil); ok || d.line != 8 {
		t.Errorf("strict: %+v %v", d, ok)
	}
	if n := corpusSkippedDiffs([]byte(layout), []byte(want), skip); n != 1 {
		t.Errorf("skipped diffs %d, want 1", n)
	}
	if n := corpusSkippedDiffs([]byte(want), []byte(want), skip); n != 0 {
		t.Errorf("skipped diffs of equal texts %d", n)
	}
	if n := corpusSkippedDiffs([]byte("a\n"), []byte("layout.x\nlayout.y\n"), skip); n != 2 {
		t.Errorf("a missing skipped line counts: %d", n)
	}

	// A skipped line present on one side only.
	if d, ok := corpusCompare([]byte(strings.Replace(want, "layout.pages.nodes=10x5/10x4246\n", "", 1)), []byte(want), skip); !ok {
		t.Errorf("a missing layout line is not skipped: %+v", d)
	}

	// A real difference: line number counts the section lines, label and
	// nearest header come from want.
	got := strings.Replace(want, " t 0: \"ABC\"", " t 0: \"ABD\"", 1)
	d, ok := corpusCompare([]byte(got), []byte(want), skip)
	wantDiff := corpusDiff{line: 10, got: ` t 0: "ABD"`, want: ` t 0: "ABC"`, label: "end", near: "row 0: dirty"}
	if ok || d != wantDiff {
		t.Errorf("diff %+v, want %+v", d, wantDiff)
	}
	d, _ = corpusCompare([]byte(strings.Replace(want, "screen.cursor.x=1", "screen.cursor.x=2", 1)), []byte(want), skip)
	if d.line != 6 || d.near != "[screen primary]" || d.label != "end" {
		t.Errorf("diff in the screen header block: %+v", d)
	}
	d, _ = corpusCompare([]byte("## case y\n"), []byte(want), skip)
	if d.line != 1 || d.label != "-" || d.near != "" || d.got != "## case y" {
		t.Errorf("diff in the case line: %+v", d)
	}

	// A text that ends early or late.
	d, ok = corpusCompare([]byte(want[:len(want)-len("\nrow 1:\n")]), []byte(want), skip)
	if ok || d.line != 11 || d.got != "<end of text>" || d.want != "row 1:" {
		t.Errorf("short got: %+v", d)
	}
	d, ok = corpusCompare([]byte(want+"extra\n"), []byte(want), skip)
	if ok || d.line != 12 || d.got != "extra" || d.want != "" {
		t.Errorf("long got: %+v", d)
	}
	d, ok = corpusCompare([]byte(strings.TrimSuffix(want, "\n")), []byte(want), skip)
	if ok || d.got != "<end of text>" {
		t.Errorf("missing final newline: %+v %v", d, ok)
	}

	// Long lines are cut.
	long := strings.Repeat("x", 500)
	d, _ = corpusCompare([]byte("a\n"+long+"\n"), []byte("a\ny\n"), skip)
	if d.got != strings.Repeat("x", 300)+"..." || d.want != "y" {
		t.Errorf("cut line: %q", d.got)
	}
}

type corpusSelfReporter struct{ errors, fails int }

func (r *corpusSelfReporter) Helper()      {}
func (r *corpusSelfReporter) Error(...any) { r.errors++ }
func (r *corpusSelfReporter) Fail()        { r.fails++ }

func TestCorpusReplaySelfReport(t *testing.T) {
	m := corpusMismatch{
		area: "csi", suite: "cup", caseName: "filled_H_rnone_c0", kind: replayKindGolden,
		diff: corpusDiff{line: 57, got: "screen.cursor.x=2", want: "screen.cursor.x=1", label: "end", near: "[screen primary]"},
	}
	want := "MISMATCH csi/cup case=filled_H_rnone_c0 kind=golden dump=end line 57\n" +
		"  got:  screen.cursor.x=2\n" +
		"  want: screen.cursor.x=1\n" +
		"  near: [screen primary]"
	if got := corpusFormatMismatch(m); got != want {
		t.Errorf("report\n got  %q\n want %q", got, want)
	}
	m.split, m.kind, m.diff.near = "op:2,at:5", replayKindActionsNoalloc, ""
	want = "MISMATCH csi/cup case=filled_H_rnone_c0 kind=actions_noalloc split=op:2,at:5 dump=end line 57\n" +
		"  got:  screen.cursor.x=2\n" +
		"  want: screen.cursor.x=1"
	if got := corpusFormatMismatch(m); got != want {
		t.Errorf("split report\n got  %q\n want %q", got, want)
	}

	st := &corpusSuiteStats{cases: 1330, run: 20, skipped: 1, skippedLines: 4}
	st.mismatch[replayKindGolden] = 2
	st.mismatch[replayKindActions] = 1
	if got, want := corpusSummary("csi", "cup", st, false),
		"SUITE csi/cup cases=1330 run=20 skipped=1 mismatch=3 (golden=2 actions=1) skipped_lines=4"; got != want {
		t.Errorf("summary %q, want %q", got, want)
	}
	if got, want := corpusSummary("csi", "cup", &corpusSuiteStats{cases: 3, run: 3}, true),
		"SUITE csi/cup cases=3 run=3 skipped=0 mismatch=0 layout_lines_differ=0"; got != want {
		t.Errorf("summary %q, want %q", got, want)
	}

	// Caps: 10 reports per suite and 50 per test; the rest are only failures.
	run := &corpusRun{}
	var rep corpusSelfReporter
	for s := 0; s < 7; s++ {
		suite := &corpusSuiteStats{}
		for i := 0; i < 12; i++ {
			corpusReport(&rep, run, suite, m)
		}
		if suite.mismatches() != 12 || suite.reported > 10 {
			t.Errorf("suite %d: %d mismatches, %d reported", s, suite.mismatches(), suite.reported)
		}
	}
	if rep.errors != 50 || rep.fails != 7*12-50 {
		t.Errorf("%d reports and %d silent failures, want 50 and %d", rep.errors, rep.fails, 7*12-50)
	}
	// TERM51_REPORT_MAX.
	run = &corpusRun{cfg: corpusConfig{reportMax: 2}}
	rep = corpusSelfReporter{}
	for s := 0; s < 4; s++ {
		suite := &corpusSuiteStats{}
		for i := 0; i < 5; i++ {
			corpusReport(&rep, run, suite, m)
		}
	}
	if rep.errors != 8 || rep.fails != 12 {
		t.Errorf("REPORT_MAX=2: %d reports and %d silent failures, want 8 and 12", rep.errors, rep.fails)
	}
}

// A fail list and a diff dir are written as documented.
func TestCorpusReplaySelfFailListAndDiffDir(t *testing.T) {
	dir := t.TempDir()
	run := &corpusRun{cfg: corpusConfig{failList: filepath.Join(dir, "fails"), diffDir: filepath.Join(dir, "diff")}}
	m := corpusMismatch{area: "csi", suite: "cup", caseName: "a/b", kind: replayKindActions}
	run.appendFailList(m)
	m.kind = replayKindGolden
	run.appendFailList(m)
	data, err := os.ReadFile(run.cfg.failList)
	if err != nil || string(data) != "csi/cup:a/b:actions\ncsi/cup:a/b:golden\n" {
		t.Errorf("fail list %q, %v", data, err)
	}
	run.writeDiff(t, m, []byte("got"), []byte("want"))
	got, err1 := os.ReadFile(filepath.Join(run.cfg.diffDir, "csi.cup.a_b.golden.got"))
	want, err2 := os.ReadFile(filepath.Join(run.cfg.diffDir, "csi.cup.a_b.golden.want"))
	if err1 != nil || err2 != nil || string(got) != "got" || string(want) != "want" {
		t.Errorf("diff dir: %q %q %v %v", got, want, err1, err2)
	}
}

func TestCorpusReplaySelfSectionsReader(t *testing.T) {
	long := strings.Repeat("y", 3<<20) // longer than the reader's buffer
	data := "## case a\nl1\nl2\n## case b\n## case notstart\n" + long + "\n## case c\n" + long + "\n## case d\nlast"
	s := corpusSectionsReader(strings.NewReader(data))
	for i, want := range []struct{ name, text string }{
		{"a", "## case a\nl1\nl2\n"},
		{"b", "## case b\n"},
	} {
		name, text, err := s.section(i)
		if err != nil || name != want.name || string(text) != want.text {
			t.Errorf("section %d: %q %q %v", i, name, text, err)
		}
	}
	// "## case notstart" is a header like any other: sections are cut at
	// every line that starts with it.
	name, text, err := s.section(3)
	if err != nil || name != "c" || string(text) != "## case c\n"+long+"\n" {
		t.Errorf("section 3: %q %d %v", name, len(text), err)
	}
	if _, _, err := s.section(2); err == nil {
		t.Error("going back succeeded")
	}
	name, text, err = s.section(4)
	if err != nil || name != "d" || string(text) != "## case d\nlast" {
		t.Errorf("section 4: %q %q %v", name, text, err)
	}
	if _, _, err := s.section(5); !errors.Is(err, errCorpusSectionsEnd) {
		t.Errorf("section 5: %v", err)
	}

	// Skipping, counting.
	s = corpusSectionsReader(strings.NewReader(data))
	if n, err := s.count(); err != nil || n != 5 {
		t.Errorf("count %d, %v", n, err)
	}
	s = corpusSectionsReader(strings.NewReader(""))
	if n, err := s.count(); err != nil || n != 0 {
		t.Errorf("empty count %d, %v", n, err)
	}
	s = corpusSectionsReader(strings.NewReader("junk\n## case a\n"))
	if _, _, err := s.section(0); err == nil {
		t.Error("a file that does not start with a section was accepted")
	}
	// A body line that merely contains the prefix is not a header.
	s = corpusSectionsReader(strings.NewReader("## case a\n x## case b\n"))
	if n, _ := s.count(); n != 1 {
		t.Errorf("count %d, want 1", n)
	}
}

// The section files of every area and kind read back in the order of the case
// file, at least the first sections (the first 256 KB; all of them, counted
// against the case file, with TERM51_FULL=1). The files read whole (all of
// them with TERM51_FULL=1, the dcs and esc ones always) are checked against
// golden/MANIFEST: the number of sections and the sha256 of the uncompressed
// text.
func TestCorpusReplaySelfSectionFiles(t *testing.T) {
	full := os.Getenv("TERM51_FULL") != ""
	manifest := corpusManifest(t)
	files, sections, verified := 0, 0, 0
	for _, area := range corpusAreas {
		for _, suite := range corpusSuiteNames(t, area) {
			parsed := corpusReadSuite(t, area, suite)
			for _, kind := range parsed.kinds.kinds() {
				name := corpusGoldenName(suite, kind)
				var s *corpusSections
				whole := full || area.name == "dcs" || area.name == "esc"
				if whole {
					s = corpusSectionsFile(t, area, name)
				} else {
					s = corpusSectionsWindow(t, area, name, 256<<10)
				}
				n := 0
				for {
					got, _, ok, err := s.read(false)
					if err != nil {
						t.Fatalf("%s/%s: %v", area.name, name, err)
					}
					if !ok {
						break
					}
					if n >= len(parsed.cases) || got != parsed.cases[n].name {
						t.Fatalf("%s/%s: section %d is %q", area.name, name, n, got)
					}
					n++
				}
				if whole {
					key := area.name + "/" + name
					if err := s.verify(key, manifest[key], true); err != nil {
						t.Errorf("%v", err)
					}
					verified++
				}
				s.Close()
				if (whole || n == len(parsed.cases)) && n != len(parsed.cases) {
					t.Errorf("%s/%s: %d sections, the case file has %d", area.name, name, n, len(parsed.cases))
				}
				if n == 0 {
					t.Errorf("%s/%s: no section", area.name, name)
				}
				files++
				sections += n
			}
		}
	}
	t.Logf("%d output files, %d sections read (full=%v), %d checked against the MANIFEST", files, sections, full, verified)
	inManifest := 0
	for path := range corpusManifest(t) {
		area, _, _ := strings.Cut(path, "/")
		for _, a := range corpusAreas {
			if a.name == area {
				inManifest++
			}
		}
	}
	if files != inManifest {
		t.Errorf("%d output files read, MANIFEST has %d", files, inManifest)
	}
}

// produce on the cases that never reach the port: a case marked
// upstream_invalid, and the markers' first lines.
func TestCorpusReplaySelfProduce(t *testing.T) {
	var buf bytes.Buffer
	ops := []replayOp{{Tag: replayOpUpstreamInvalid}, {Tag: replayOpFeed, Feed: []uint8("x")}}
	if err := produceSuite([]replayNamed{{"inv", ops}}, replayKindGolden, &buf); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "## case inv\n!upstream-integrity-failure\n"; got != want {
		t.Errorf("upstream_invalid: %q, want %q", got, want)
	}
	buf.Reset()
	ops = []replayOp{{Tag: replayOpTransientInvariant}, {Tag: replayOpUpstreamInvalid}}
	if err := produce(ops, replayKindGolden, feedMode{}, &buf); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "!transient-upstream-invariant-failure-end-state-verified\n!upstream-integrity-failure\n"; got != want {
		t.Errorf("transient + invalid: %q, want %q", got, want)
	}
	if !expectsCrash([]replayOp{{Tag: replayOpFeed}, {Tag: replayOpExpectCrash}}) || expectsCrash(nil) ||
		!isUpstreamInvalid(ops) || isUpstreamInvalid([]replayOp{{Tag: replayOpFeed}}) {
		t.Error("expectsCrash or isUpstreamInvalid")
	}
	// The search kind is not a terminal kind: an error, which corpusProduce
	// turns into a line of the output.
	out := corpusProduce(replayNamed{"x", []replayOp{{Tag: replayOpFeed, Feed: []uint8("x")}}}, replayKindSearch)
	if got, want := string(out), "## case x\n!replay error=kind search is not a terminal kind\n"; got != want {
		t.Errorf("corpusProduce of the search kind: %q, want %q", got, want)
	}
	// The run of an upstream_invalid case is never reached.
	func() {
		defer func() {
			if r := recover(); r != "unreachable" {
				t.Errorf("runCaseUpstreamInvalid recovered %v", r)
			}
		}()
		runCaseUpstreamInvalid(&runCaseState{})
	}()
}

// ---------------------------------------------------------------------------
// The dump.

type dumpSelfTag uint8

const (
	dumpSelfTagNone dumpSelfTag = iota
	dumpSelfTagCursorUp
	dumpSelfTagPin
	dumpSelfTag8Bg
	dumpSelfTagOrphan
)

func (self dumpSelfTag) String() string {
	switch self {
	case dumpSelfTagNone:
		return "none"
	case dumpSelfTagCursorUp:
		return "cursor_up"
	case dumpSelfTagPin:
		return "pin"
	case dumpSelfTag8Bg:
		return "8_bg"
	case dumpSelfTagOrphan:
		return "orphan"
	}
	return ""
}

type dumpSelfEnum int8

func (self dumpSelfEnum) String() string {
	if self == 3 {
		return "three"
	}
	return ""
}

type dumpSelfBools struct {
	focused bool
	dirty   bool
	_pad    uint8
}

type dumpSelfOneBool struct {
	focused bool
}

type dumpSelfInner struct {
	x uint16
	y int32
}

type dumpSelfUnion struct {
	Tag      dumpSelfTag
	CursorUp uint8
	Pin      dumpSelfInner
	X8Bg     uint8
}

// dumpSelfUnregistered is a union type that is not in dumpVoidTags.
type dumpSelfUnregistered struct {
	Tag dumpSelfTag
}

type dumpSelfNames struct {
	x132_column bool
	default_    Optional[uint32]
	_padding    uint16
	fn          func()
	Other       dumpSelfEnum
}

func dumpSelfText(prefix string, v any) string {
	var buf bytes.Buffer
	dumpAny(&buf, prefix, v)
	return buf.String()
}

func TestCorpusReplaySelfDumpAny(t *testing.T) {
	// dumpSelfUnion: `none` is void; `orphan` is not and has no field.
	dumpVoidTags[reflect.TypeOf(dumpSelfUnion{})] = []string{"none"}
	defer delete(dumpVoidTags, reflect.TypeOf(dumpSelfUnion{}))
	n := 5
	for _, c := range []struct {
		name string
		v    any
		want string
	}{
		{"bool", true, "p=1\n"},
		{"false", false, "p=0\n"},
		{"uint", uint16(65535), "p=65535\n"},
		{"int", int64(-7), "p=-7\n"},
		{"enum", dumpSelfEnum(3), "p=three\n"},
		{"unnamed", dumpSelfEnum(-2), "p=#-2\n"},
		{"optional null", Optional[uint32]{}, "p=null\n"},
		{"optional", Some(uint32(90)), "p=90\n"},
		{"optional enum", Some(dumpSelfEnum(3)), "p=three\n"},
		{"optional struct", Some(dumpSelfInner{x: 1, y: -2}), "p.x=1\np.y=-2\n"},
		{"bools", dumpSelfBools{focused: true, dirty: true}, "p=focused|dirty\n"},
		{"bools one", dumpSelfBools{dirty: true}, "p=dirty\n"},
		{"bools none", dumpSelfBools{}, "p=none\n"},
		{"one bool is a plain struct", dumpSelfOneBool{focused: true}, "p.focused=1\n"},
		{"struct", dumpSelfInner{x: 3, y: 4}, "p.x=3\np.y=4\n"},
		{"union void", dumpSelfUnion{Tag: dumpSelfTagNone, CursorUp: 9}, "p=none\n"},
		{"union", dumpSelfUnion{Tag: dumpSelfTagCursorUp, CursorUp: 9}, "p=cursor_up\np.cursor_up=9\n"},
		{"union struct", dumpSelfUnion{Tag: dumpSelfTagPin, Pin: dumpSelfInner{x: 1}}, "p=pin\np.pin.x=1\np.pin.y=0\n"},
		{"union digit tag", dumpSelfUnion{Tag: dumpSelfTag8Bg, X8Bg: 3}, "p=8_bg\np.8_bg=3\n"},
		{"bytes", []uint8("a\\\x00\x7f\n~"), "p=a\\x5c\\x00\\x7f\\x0a~\n"},
		{"empty bytes", []uint8{}, "p=\n"},
		{"byte array", [3]uint8{0x41, 0x1b, 0x5c}, "p=A\\x1b\\x5c\n"},
		{"int slice", []uint16{1, 22, 65535}, "p=[1,22,65535]\n"},
		{"empty int slice", []int{}, "p=[]\n"},
		{"int array", [2]uint16{7, 8}, "p.len=2\np[0]:\np[]=7\np[1]:\np[]=8\n"},
		{"enum slice", []dumpSelfEnum{3, 1}, "p.len=2\np[0]:\np[]=three\np[1]:\np[]=#1\n"},
		{"struct slice", []dumpSelfInner{{1, 2}}, "p.len=1\np[0]:\np[].x=1\np[].y=2\n"},
		{"pointer", &dumpSelfInner{x: 5}, "p.*.x=5\np.*.y=0\n"},
		{"nil pointer", (*dumpSelfInner)(nil), "p=null\n"},
		{"pointer to int", &n, "p.*=5\n"},
		{"float", 1.5, "p=1.5\n"},
		{"float zero", float32(0), "p=0\n"},
		{"float small", 0.00001, "p=0.00001\n"},
		{"float big", 1e21, "p=1000000000000000000000\n"},
		{"names", dumpSelfNames{x132_column: true, default_: Some(uint32(2)), _padding: 7, Other: 3},
			"p.132_column=1\np.default=2\np.other=three\n"},
	} {
		if got := dumpSelfText("p", c.v); got != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.name, got, c.want)
		}
	}
	// A name that is a keyword in Zig and a bool struct with the mapped name.
	type flags struct {
		x132_column bool
		default_    bool
	}
	if got, want := dumpSelfText("m", flags{true, true}), "m=132_column|default\n"; got != want {
		t.Errorf("mapped bool names: %q, want %q", got, want)
	}
	// NaN and the infinities are Zig's.
	for f, want := range map[float64]string{nan(): "p=nan\n", inf(1): "p=inf\n", inf(-1): "p=-inf\n"} {
		if got := dumpSelfText("p", f); got != want {
			t.Errorf("%v: %q, want %q", f, got, want)
		}
	}
	func() {
		defer func() {
			if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "unsupported type") {
				t.Errorf("a string did not panic: %v", r)
			}
		}()
		dumpSelfText("p", "text")
	}()

	// A non-void tag without a payload field, and a union type that is not
	// listed, panic with the type and the tag: the payload line is never
	// silently left out.
	for _, c := range []struct {
		name string
		v    any
		want []string
	}{
		{"payload field missing", dumpSelfUnion{Tag: dumpSelfTagOrphan}, []string{"dumpSelfUnion", `"orphan"`, "Orphan"}},
		{"unlisted type", dumpSelfUnregistered{Tag: dumpSelfTagPin}, []string{"dumpSelfUnregistered", "dumpVoidTags", `"pin"`}},
	} {
		func() {
			defer func() {
				r := fmt.Sprint(recover())
				for _, w := range c.want {
					if !strings.Contains(r, w) {
						t.Errorf("%s: panic %q lacks %q", c.name, r, w)
					}
				}
			}()
			dumpSelfText("p", c.v)
			t.Errorf("%s: no panic", c.name)
		}()
	}
}

func nan() float64 { z := 0.0; return z / z }

func inf(sign int) float64 {
	z := 0.0
	if sign < 0 {
		return -1 / z
	}
	return 1 / z
}

func TestCorpusReplaySelfNames(t *testing.T) {
	for in, want := range map[string]string{
		"x132_column": "132_column", "default_": "default", "xerox": "xerox", "x": "x", "cols": "cols", "mouse_event_x10": "mouse_event_x10",
	} {
		if got := zigName(in); got != want {
			t.Errorf("zigName(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{"cursor_up": "CursorUp", "cl": "Cl", "pin": "Pin", "a_b_c": "ABC"} {
		if got := zigCamel(in); got != want {
			t.Errorf("zigCamel(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{"cursor_up": "CursorUp", "8_bg": "X8Bg", "256_underline_color": "X256UnderlineColor"} {
		if got := zigFieldName(in); got != want {
			t.Errorf("zigFieldName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCorpusReplaySelfWriters(t *testing.T) {
	// Hand-written (see the TODO at the top): the bytes that the first
	// sections of the golden files do not have.
	var buf bytes.Buffer
	writeEscaped(&buf, []uint8("a\\b\x00\xff~ "))
	if got, want := buf.String(), `a\x5cb\x00\xff~ `; got != want {
		t.Errorf("writeEscaped %q, want %q", got, want)
	}

	plain := PageCell{content_tag: PageCellContentTagCodepoint}
	plain.content.codepoint = 'A'
	empty := PageCell{}
	space := PageCell{}
	space.content.codepoint = 0x20
	ctrl := PageCell{}
	ctrl.content.codepoint = 0x1f
	del := PageCell{}
	del.content.codepoint = 0x7f
	styled := plain
	styled.style_id = 1
	wide := plain
	wide.wide = PageCellWideWide
	prot := plain
	prot.protected = true
	link := plain
	link.hyperlink = true
	sem := plain
	sem.semantic_content = PageCellSemanticContentInput
	bg := PageCell{content_tag: PageCellContentTagBgColorPalette}
	bg.content.color_palette = 9
	for _, c := range []struct {
		name                       string
		cell                       PageCell
		plainAscii, isDefault, att bool
	}{
		{"plain", plain, true, false, true},
		{"empty", empty, false, true, true},
		{"space", space, true, false, true},
		{"control", ctrl, false, false, true},
		{"delete", del, false, false, true},
		{"styled", styled, false, false, false},
		{"wide", wide, false, false, false},
		{"protected", prot, false, false, false},
		{"hyperlink", link, false, false, false},
		{"semantic", sem, false, false, false},
		{"bg palette", bg, false, false, true},
	} {
		if got := isPlainAscii(&c.cell); got != c.plainAscii {
			t.Errorf("isPlainAscii(%s) = %v", c.name, got)
		}
		if got := cellIsDefault(&c.cell); got != c.isDefault {
			t.Errorf("cellIsDefault(%s) = %v", c.name, got)
		}
		if got := cellAttrsDefault(&c.cell); got != c.att {
			t.Errorf("cellAttrsDefault(%s) = %v", c.name, got)
		}
	}

	if boolInt(true) != 1 || boolInt(false) != 0 {
		t.Error("boolInt")
	}
	// dumpUnwrap fails like Zig's `.?` on null.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("dumpUnwrap of null did not panic")
			}
		}()
		dumpUnwrap(None[int]())
	}()
}

// ---------------------------------------------------------------------------
// The dumper against the recorded text: the dump lines of the types whose
// values need no enum name (so they run without the port) are rebuilt from
// the recorded lines and dumped again.

// dumpSelfGoldenLines are the lines of the first sections (the first 256 KB)
// of a golden file.
func dumpSelfGoldenLines(t *testing.T, area, name string) []string {
	t.Helper()
	s := corpusSectionsWindow(t, corpusAreaByName(t, area), name, 256<<10)
	defer s.Close()
	var lines []string
	for {
		_, text, ok, err := s.read(true)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			return lines
		}
		lines = append(lines, strings.Split(string(text), "\n")...)
	}
}

// dumpSelfSetBools sets the bool fields named in `a|b|none` of the struct that
// ptr points to.
func dumpSelfSetBools(t *testing.T, ptr any, names string) {
	t.Helper()
	v := reflect.ValueOf(ptr).Elem()
	if names == "none" {
		return
	}
	for _, name := range strings.Split(names, "|") {
		found := false
		for i := 0; i < v.NumField(); i++ {
			if !strings.HasPrefix(v.Type().Field(i).Name, "_") && zigName(v.Type().Field(i).Name) == name {
				dumpField(v, i).SetBool(true)
				found = true
			}
		}
		if !found {
			t.Errorf("%T has no field %q", ptr, name)
		}
	}
}

func TestCorpusReplaySelfDumpAgainstGolden(t *testing.T) {
	// The bool sets: modes (values, saved, default), the dirty flags, the
	// kitty keyboard flags. Every distinct recorded line is rebuilt.
	seen := map[string]bool{}
	for _, suite := range corpusSuiteNames(t, corpusAreaByName(t, "modes")) {
		for _, line := range dumpSelfGoldenLines(t, "modes", suite+".golden.gz") {
			key, names, ok := strings.Cut(line, "=")
			if !ok || seen[line] {
				continue
			}
			var ptr any
			switch {
			case key == "term.modes.values" || key == "term.modes.saved" || key == "term.modes.default":
				ptr = &ModesModePacked{}
			case key == "term.flags.dirty":
				ptr = &TerminalDirty{}
			case key == "screen.dirty":
				ptr = &ScreenDirty{}
			case strings.HasPrefix(key, "screen.kitty_keyboard.flags["):
				ptr = &KittyKeyFlags{}
			default:
				continue
			}
			seen[line] = true
			dumpSelfSetBools(t, ptr, names)
			if got := dumpSelfText(key, reflect.ValueOf(ptr).Elem().Interface()); got != line+"\n" {
				t.Errorf("dump %q, recorded %q", got, line)
			}
		}
	}
	if len(seen) < 60 {
		t.Errorf("only %d distinct bool-set lines in the modes suites", len(seen))
	}

	lines := dumpSelfGoldenLines(t, "csi", "cup.golden.gz")
	var flags, region, colors []string
	for i, l := range lines { // the first section only
		if i > 0 && strings.HasPrefix(l, corpusSectionPrefix) {
			break
		}
		switch {
		case strings.HasPrefix(l, "term.flags."):
			flags = append(flags, strings.SplitN(l, "=", 2)[0][len("term.flags."):])
		case strings.HasPrefix(l, "term.scrolling_region."):
			region = append(region, l)
		case strings.HasPrefix(l, "term.colors.") && !strings.Contains(l, "palette"):
			colors = append(colors, l)
		}
	}
	// The order of the fields of Terminal.flags.
	var fields []string
	ft := reflect.TypeOf(TerminalFlags{})
	for i := 0; i < ft.NumField(); i++ {
		fields = append(fields, zigName(ft.Field(i).Name))
	}
	if !reflect.DeepEqual(flags, fields) {
		t.Errorf("term.flags fields\n recorded %v\n Go       %v", flags, fields)
	}
	// Scrolling region and the unset dynamic colors.
	// The region is rebuilt from the recorded values.
	var sr TerminalScrollingRegion
	for _, l := range region {
		name, v, _ := strings.Cut(strings.TrimPrefix(l, "term.scrolling_region."), "=")
		n, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			t.Fatalf("recorded line %q", l)
		}
		switch name {
		case "top":
			sr.top = SizeCellCountInt(n)
		case "bottom":
			sr.bottom = SizeCellCountInt(n)
		case "left":
			sr.left = SizeCellCountInt(n)
		case "right":
			sr.right = SizeCellCountInt(n)
		default:
			t.Fatalf("recorded line %q", l)
		}
	}
	if got := dumpSelfText("term.scrolling_region", sr); got != strings.Join(region, "\n")+"\n" {
		t.Errorf("scrolling region %q, recorded %q", got, region)
	}
	var unset string
	for _, name := range []string{"background", "foreground", "cursor"} {
		unset += dumpSelfText("term.colors."+name, ColorDynamicRGB{})
	}
	if unset != strings.Join(colors, "\n")+"\n" {
		t.Errorf("unset colors %q, recorded %q", unset, colors)
	}

	// A set dynamic color: r, g, b in this order, decimal, then the default.
	// Every distinct recorded set color of the first sections of the osc
	// suites is rebuilt from its r, g and b and dumped again.
	checked := map[string]bool{}
	for _, suite := range []string{"all", "color_state", "rgbi_float", "unterminated"} {
		lines := dumpSelfGoldenLines(t, "osc", suite+".golden.gz")
		for i, l := range lines {
			prefix, _, found := strings.Cut(l, ".override.r=")
			if !found || !strings.HasPrefix(prefix, "term.colors.") || i+3 >= len(lines) {
				continue
			}
			recorded := strings.Join(lines[i:i+4], "\n") + "\n"
			if checked[recorded] {
				continue
			}
			checked[recorded] = true
			var rgb [3]uint8
			for k, c := range []string{"r", "g", "b"} {
				v, found := strings.CutPrefix(lines[i+k], prefix+".override."+c+"=")
				n, err := strconv.ParseUint(v, 10, 8)
				if !found || err != nil {
					t.Fatalf("recorded color line %q", lines[i+k])
				}
				rgb[k] = uint8(n)
			}
			if lines[i+3] != prefix+".default=null" {
				t.Fatalf("recorded color: %q after the override", lines[i+3])
			}
			got := dumpSelfText(prefix, ColorDynamicRGB{override: Some(ColorRGB{r: rgb[0], g: rgb[1], b: rgb[2]})})
			if got != recorded {
				t.Errorf("set color %q, recorded %q", got, recorded)
			}
		}
	}
	if len(checked) < 2 {
		t.Errorf("only %d distinct set colors in the osc golden windows", len(checked))
	}
	t.Logf("%d distinct recorded set colors", len(checked))
}

// ---------------------------------------------------------------------------
// Tagged unions: the payload of every tag is dumped or void.

// zigUnionField is one field of a Zig union: the tag name, and whether its
// payload type is void.
type zigUnionField struct {
	name string
	void bool
}

// zigUnion is a tagged union of the Zig source.
type zigUnion struct {
	where  string // file:line
	fields []zigUnionField
}

// zigStripLine removes what is not code from a line of Zig: comments, the
// contents of string and character literals; a multi-line string line (`\\`)
// is empty.
func zigStripLine(line string) string {
	if strings.HasPrefix(strings.TrimSpace(line), `\\`) {
		return ""
	}
	var sb strings.Builder
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '/' && i+1 < len(line) && line[i+1] == '/':
			return sb.String()
		case c == '"' && i > 0 && line[i-1] == '@':
			// An @"quoted name" is code: keep it.
			end := strings.IndexByte(line[i+1:], '"')
			if end < 0 {
				return sb.String()
			}
			sb.WriteString(line[i : i+end+2])
			i += end + 1
		case c == '"' || c == '\'':
			sb.WriteByte(c)
			for i++; i < len(line) && line[i] != c; i++ {
				if line[i] == '\\' {
					i++
				}
			}
			sb.WriteByte(c)
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

var (
	zigUnionStart = regexp.MustCompile(`\bunion\s*\(\s*(enum(\s*\([^)]*\))?|[A-Za-z_][A-Za-z0-9_.]*)\s*\)\s*\{\s*$`)
	zigFieldStart = regexp.MustCompile(`^(@"[^"]+"|[A-Za-z_][A-Za-z0-9_]*)\s*(:|,|$)`)
	zigNotAField  = map[string]bool{"pub": true, "const": true, "var": true, "fn": true, "test": true, "comptime": true,
		"inline": true, "extern": true, "export": true, "usingnamespace": true}
)

// zigScanUnions finds the tagged unions of a Zig source text, with their
// fields. A field is a line at the top level of the union body that starts
// with a name and has `:` or `,` after it; its payload is void when it has no
// `:` or the type is `void`.
func zigScanUnions(where, text string) []zigUnion {
	lines := strings.Split(text, "\n")
	var out []zigUnion
	for i := 0; i < len(lines); i++ {
		if !zigUnionStart.MatchString(zigStripLine(lines[i])) {
			continue
		}
		u := zigUnion{where: fmt.Sprintf("%s:%d", where, i+1)}
		depth := 1
		for j := i + 1; j < len(lines) && depth > 0; j++ {
			code := strings.TrimSpace(zigStripLine(lines[j]))
			if depth == 1 {
				if m := zigFieldStart.FindStringSubmatch(code); m != nil && !zigNotAField[m[1]] {
					name := strings.TrimSuffix(strings.TrimPrefix(m[1], `@"`), `"`)
					typ := ""
					if m[2] == ":" {
						typ = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(code[len(m[1]):][1:]), ","))
					}
					u.fields = append(u.fields, zigUnionField{name: name, void: m[2] != ":" || typ == "void"})
				}
			}
			// Parentheses and brackets nest too: `lib.Enum(\n lib_target,` is
			// not a field.
			depth += strings.Count(code, "{") + strings.Count(code, "(") + strings.Count(code, "[") -
				strings.Count(code, "}") - strings.Count(code, ")") - strings.Count(code, "]")
		}
		out = append(out, u)
	}
	return out
}

// zigSourceUnions are the tagged unions of the Zig source of the terminal
// package of the pinned upstream (term51/third_party/ghostty/src/terminal).
func zigSourceUnions(t *testing.T) []zigUnion {
	t.Helper()
	root := filepath.Join(corpusDir(t), "third_party", "ghostty", "src", "terminal")
	var out []zigUnion
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".zig") {
			return err
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		out = append(out, zigScanUnions(rel, string(text))...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// goTagNames are the tags of the Go tag types of the package, from the
// declarations of their constants (String() is not usable before the port):
// the constants of type T named T<Value> (or without the `Tag` of T), as the
// suffixes <Value>.
func goTagNames(t *testing.T) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, f := range parsePackageSources(t) {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			typ := ""
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				if id, ok := vs.Type.(*ast.Ident); ok {
					typ = id.Name
				} else if vs.Type != nil || len(vs.Values) > 0 {
					typ = ""
				}
				for _, n := range vs.Names {
					if typ == "" {
						continue
					}
					// T<Value>, or without the `Tag` of the type name.
					for _, prefix := range []string{typ, strings.TrimSuffix(typ, "Tag")} {
						rest, ok := strings.CutPrefix(n.Name, prefix)
						if ok && rest != "" && (rest[0] >= 'A' && rest[0] <= 'Z' || rest[0] >= '0' && rest[0] <= '9') {
							out[typ] = append(out[typ], rest)
							break
						}
					}
				}
			}
		}
	}
	return out
}

// parsePackageSources parses the non-test .go files of the package.
func parsePackageSources(t *testing.T) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var out []*ast.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, f)
	}
	return out
}

// dumpUnionTypes are the tagged-union types reachable from a type by fields,
// pointers, slices and arrays, the type itself included.
func dumpUnionTypes(root reflect.Type) []reflect.Type {
	var out []reflect.Type
	seen := map[reflect.Type]bool{}
	var walk func(rt reflect.Type)
	walk = func(rt reflect.Type) {
		if seen[rt] {
			return
		}
		seen[rt] = true
		switch rt.Kind() {
		case reflect.Struct:
			if dumpIsUnion(rt) {
				out = append(out, rt)
			}
			for i := 0; i < rt.NumField(); i++ {
				walk(rt.Field(i).Type)
			}
		case reflect.Pointer, reflect.Slice, reflect.Array:
			walk(rt.Elem())
		}
	}
	walk(root)
	return out
}

// Every tagged union reachable from the stream action and the parser action: for each tag the Zig
// source says whether its payload is void, and the Go type has a payload
// field (named by zigFieldName) exactly for the others; dumpVoidTags lists
// exactly the void tags and exactly these types. A failure names the type and
// the tag.
func TestCorpusReplaySelfUnionsAgainstZig(t *testing.T) {
	zigUnions := zigSourceUnions(t)
	goTags := goTagNames(t)
	types := dumpUnionTypes(reflect.TypeOf(StreamAction{}))
	if len(types) < 6 {
		t.Errorf("only %d union types reachable from StreamAction: %v", len(types), types)
	}
	// The parser trace dumps Parser.Action (trace.zig:58 parserFeed).
	for _, ut := range dumpUnionTypes(reflect.TypeOf(ParserAction{})) {
		if !slices.Contains(types, ut) {
			types = append(types, ut)
		}
	}
	// The state dump (dump.zig dumpTerminal, dumpScreen) passes these fields
	// to dumpAny; the unions reachable from them must be registered too.
	for _, root := range dumpStateRootTypes(t) {
		for _, ut := range dumpUnionTypes(root) {
			if !slices.Contains(types, ut) {
				types = append(types, ut)
			}
		}
	}
	registered := map[reflect.Type]bool{}
	for _, ut := range types {
		registered[ut] = true
		tagType := ut.Field(0).Type.Name()
		tags := goTags[tagType]
		if len(tags) == 0 {
			t.Errorf("%s: no constants of its tag type %s found in the source", ut, tagType)
			continue
		}
		// The Zig unions whose tags are these.
		var match []zigUnion
		for _, zu := range zigUnions {
			names := make([]string, len(zu.fields))
			for i, f := range zu.fields {
				names[i] = zigCamel(f.name)
			}
			if slices.Equal(sortedCopy(names), sortedCopy(tags)) {
				match = append(match, zu)
			}
		}
		if len(match) == 0 {
			t.Errorf("%s: no union of the Zig source has the tags %v", ut, tags)
			continue
		}
		void := map[string]bool{}
		for _, f := range match[0].fields {
			void[f.name] = f.void
		}
		for _, zu := range match[1:] {
			for _, f := range zu.fields {
				if void[f.name] != f.void {
					t.Errorf("%s: the Zig unions %s and %s have the same tags and differ on the tag %q",
						ut, match[0].where, zu.where, f.name)
				}
			}
		}
		var voids []string
		for _, f := range match[0].fields {
			_, hasField := ut.FieldByName(zigFieldName(f.name))
			switch {
			case f.void && hasField:
				t.Errorf("%s: tag %q is void in %s, but the type has the field %s", ut, f.name, match[0].where, zigFieldName(f.name))
			case !f.void && !hasField:
				t.Errorf("%s: tag %q has a payload in %s, but the type has no field %s", ut, f.name, match[0].where, zigFieldName(f.name))
			}
			if f.void {
				voids = append(voids, f.name)
			}
		}
		// Fields that are no tag.
		for i := 1; i < ut.NumField(); i++ {
			ok := false
			for _, f := range match[0].fields {
				ok = ok || zigFieldName(f.name) == ut.Field(i).Name
			}
			if !ok {
				t.Errorf("%s: the field %s is the payload of no tag", ut, ut.Field(i).Name)
			}
		}
		reg, ok := dumpVoidTags[ut]
		if !ok {
			t.Errorf("%s: not in dumpVoidTags; its void tags are %q", ut, voids)
		} else if !slices.Equal(sortedCopy(reg), sortedCopy(voids)) {
			t.Errorf("%s: dumpVoidTags has %q, the Zig source %q", ut, sortedCopy(reg), sortedCopy(voids))
		}
	}
	for ut := range dumpVoidTags {
		if !registered[ut] {
			t.Errorf("dumpVoidTags has %s, which is not reachable from StreamAction, ParserAction or the state dump", ut)
		}
	}
}

// dumpStateRootTypes is the types of the Terminal and Screen fields that
// dumpTerminal and dumpScreen (corpus_replay_dump_test.go) pass to dumpAny.
// Only those fields: a walk of all of Terminal would reach page internals
// the dump prints by hand. Keep it in step with the dumpAny calls there.
func dumpStateRootTypes(t *testing.T) []reflect.Type {
	field := func(rt reflect.Type, names ...string) []reflect.Type {
		var out []reflect.Type
		for _, n := range names {
			f, ok := rt.FieldByName(n)
			if !ok {
				t.Fatalf("%s has no field %s", rt, n)
			}
			out = append(out, f.Type)
		}
		return out
	}
	var roots []reflect.Type
	roots = append(roots, field(reflect.TypeOf(Terminal{}), "colors", "flags", "modes", "mouse_shape",
		"previous_char", "scrolling_region", "status_display")...)
	roots = append(roots, field(reflect.TypeOf(Screen{}), "charset", "dirty", "kitty_keyboard",
		"protected_mode", "semantic_prompt")...)
	roots = append(roots, field(reflect.TypeOf(ScreenCursor{}), "cursor_style", "semantic_content",
		"semantic_content_clear_eol", "style_id", "hyperlink_id", "hyperlink_implicit_id", "protected",
		"pending_wrap")...)
	roots = append(roots, field(reflect.TypeOf(ScreenSavedCursor{}), "charset", "origin", "pending_wrap",
		"protected")...)
	return roots
}

func sortedCopy(s []string) []string {
	out := slices.Clone(s)
	sort.Strings(out)
	return out
}

func TestCorpusReplaySelfZigScanUnions(t *testing.T) {
	text := "pub const A = union(enum) {\n" +
		"    /// doc { comment\n" +
		"    plain,\n" +
		"    with_payload: u8,\n" +
		"    @\"8_bg\": u8,\n" +
		"    void_typed: void,\n" +
		"    inline_struct: struct {\n" +
		"        x: u8,\n" +
		"        y,\n" +
		"    },\n" +
		"    str: []const u8, // trailing \"{\" comment\n" +
		"\n" +
		"    pub const Key = enum { a, b };\n" +
		"    pub fn f() void { const z = \"}\"; }\n" +
		"};\n" +
		"const B = union(Key) { one, two: u8 };\n" +
		"const C = union { not_tagged: u8 };\n" +
		"pub const D = union(enum(u8)) {\n" +
		"    d1,\n" +
		"};\n"
	got := zigScanUnions("t.zig", text)
	want := [][]zigUnionField{
		{{"plain", true}, {"with_payload", false}, {"8_bg", false}, {"void_typed", true}, {"inline_struct", false}, {"str", false}},
		{{"d1", true}},
	}
	if len(got) != len(want) {
		t.Fatalf("%d unions, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if !reflect.DeepEqual(got[i].fields, want[i]) {
			t.Errorf("union %d: %+v, want %+v", i, got[i].fields, want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// The writers against the recorded lines.

// dumpSelfGoldenLinesN is dumpSelfGoldenLines with a window of `limit` bytes.
func dumpSelfGoldenLinesN(t *testing.T, area, name string, limit int64) []string {
	t.Helper()
	s := corpusSectionsWindow(t, corpusAreaByName(t, area), name, limit)
	defer s.Close()
	var lines []string
	for {
		_, text, ok, err := s.read(true)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			return lines
		}
		lines = append(lines, strings.Split(string(text), "\n")...)
	}
}

// dumpSelfUnescape is the inverse of writeEscaped: `\xNN` is the byte NN.
func dumpSelfUnescape(t *testing.T, s string) []byte {
	t.Helper()
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			out = append(out, s[i])
			continue
		}
		if i+4 > len(s) || s[i+1] != 'x' {
			t.Fatalf("bad escape in %q", s)
		}
		b, err := hex.DecodeString(s[i+2 : i+4])
		if err != nil {
			t.Fatalf("bad escape in %q: %v", s, err)
		}
		out = append(out, b...)
		i += 3
	}
	return out
}

// dumpSelfParseColor is the inverse of writeColor.
func dumpSelfParseColor(s string) (StyleColor, bool) {
	switch {
	case s == "none":
		return StyleColor{Tag: StyleColorNone}, true
	case strings.HasPrefix(s, "pal:"):
		n, err := strconv.ParseUint(s[len("pal:"):], 10, 8)
		return StyleColor{Tag: StyleColorPalette, Palette: uint8(n)}, err == nil
	case strings.HasPrefix(s, "rgb:") && len(s) == len("rgb:")+6:
		v, err := strconv.ParseUint(s[len("rgb:"):], 16, 32)
		return StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{r: uint8(v >> 16), g: uint8(v >> 8), b: uint8(v)}}, err == nil
	}
	return StyleColor{}, false
}

// dumpSelfParseStyle is the inverse of writeStyle, for the styles without an
// underline (the name of the underline is the port's String()).
func dumpSelfParseStyle(s string) (Style, bool) {
	body, ok := strings.CutPrefix(s, "{fg=")
	if !ok || !strings.HasSuffix(body, "}") {
		return Style{}, false
	}
	body = strings.TrimSuffix(body, "}")
	fg, rest, ok1 := strings.Cut(body, ",bg=")
	bg, rest, ok2 := strings.Cut(rest, ",ul=")
	ul, flags, ok3 := strings.Cut(rest, ",flags=")
	if !ok1 || !ok2 || !ok3 || strings.Contains(flags, "underline:") {
		return Style{}, false
	}
	var st Style
	var good [3]bool
	st.fg_color, good[0] = dumpSelfParseColor(fg)
	st.bg_color, good[1] = dumpSelfParseColor(bg)
	st.underline_color, good[2] = dumpSelfParseColor(ul)
	if !good[0] || !good[1] || !good[2] {
		return Style{}, false
	}
	if flags != "none" {
		for _, f := range strings.Split(flags, "|") {
			switch f {
			case "bold":
				st.flags.bold = true
			case "italic":
				st.flags.italic = true
			case "faint":
				st.flags.faint = true
			case "blink":
				st.flags.blink = true
			case "inverse":
				st.flags.inverse = true
			case "invisible":
				st.flags.invisible = true
			case "strikethrough":
				st.flags.strikethrough = true
			case "overline":
				st.flags.overline = true
			default:
				return Style{}, false
			}
		}
	}
	return st, true
}

// writeStyle (and writeColor) against the styles of the recorded dumps: every
// distinct style of the first sections of text/sgr_*, text/style_churn and
// csi/cbt that has no underline is parsed back and written again.
func TestCorpusReplaySelfWriteStyleRecorded(t *testing.T) {
	styleRe := regexp.MustCompile(`style=(\{[^}]*\})`)
	seen := map[string]bool{}
	skipped := 0
	for _, f := range []struct{ area, name string }{
		{"text", "sgr_single.golden.gz"}, {"text", "sgr_pairs.golden.gz"}, {"text", "sgr_extended.golden.gz"},
		{"text", "style_churn.golden.gz"}, {"csi", "cbt.golden.gz"},
	} {
		for _, line := range dumpSelfGoldenLines(t, f.area, f.name) {
			m := styleRe.FindStringSubmatch(line)
			if m == nil || seen[m[1]] {
				continue
			}
			seen[m[1]] = true
			st, ok := dumpSelfParseStyle(m[1])
			if !ok {
				skipped++ // an underline: see the TODO(recorded) list
				continue
			}
			var buf bytes.Buffer
			writeStyle(&buf, st)
			if buf.String() != m[1] {
				t.Errorf("writeStyle = %q, recorded %q", buf.String(), m[1])
			}
		}
	}
	t.Logf("%d distinct recorded styles, %d with an underline not checked", len(seen), skipped)
	// The three kinds of colors and a flag of each position are in the data.
	for _, want := range []string{"fg=pal:", "bg=pal:", "ul=pal:", "rgb:", "flags=bold", "|overline", "flags=none"} {
		found := false
		for st := range seen {
			found = found || strings.Contains(st, want)
		}
		if !found {
			t.Errorf("no recorded style contains %q: the test does not cover it", want)
		}
	}
	if len(seen)-skipped < 20 {
		t.Errorf("only %d recorded styles without an underline", len(seen)-skipped)
	}
}

// dumpScreenCell against the recorded ` c <x>:` lines that need no page
// lookup and no enum name: the cells with a codepoint, a palette or RGB
// background, and the protected flag.
func TestCorpusReplaySelfDumpScreenCellRecorded(t *testing.T) {
	cellRe := regexp.MustCompile(`^ c (\d+): (cp=([0-9a-f]+)|bgpal=(\d+)|bgrgb=([0-9a-f]{6}))( protected)?$`)
	shapes := map[string]bool{}
	checked := 0
	for _, f := range []struct {
		area, name string
		limit      int64
	}{
		{"csi", "cbt.golden.gz", 256 << 10}, {"csi", "decsca.golden.gz", 256 << 10},
		{"text", "style_churn.golden.gz", 256 << 10}, {"fuzz", "structured_medium.golden.gz", 2 << 20},
	} {
		for _, line := range dumpSelfGoldenLinesN(t, f.area, f.name, f.limit) {
			m := cellRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			x, _ := strconv.Atoi(m[1])
			var cell PageCell
			switch {
			case m[3] != "":
				v, _ := strconv.ParseUint(m[3], 16, 32)
				cell.content_tag = PageCellContentTagCodepoint
				cell.content.codepoint = uint32(v)
				shapes["cp"] = true
			case m[4] != "":
				v, _ := strconv.ParseUint(m[4], 10, 8)
				cell.content_tag = PageCellContentTagBgColorPalette
				cell.content.color_palette = uint8(v)
				shapes["bgpal"] = true
			default:
				v, _ := strconv.ParseUint(m[5], 16, 32)
				cell.content_tag = PageCellContentTagBgColorRgb
				cell.content.color_rgb = PageCellRGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}
				shapes["bgrgb"] = true
			}
			if m[6] != "" {
				cell.protected = true
				shapes["protected"] = true
			}
			var buf bytes.Buffer
			dumpScreenCell(&buf, x, &cell, nil)
			if buf.String() != line+"\n" {
				t.Errorf("dumpScreenCell = %q, recorded %q", buf.String(), line)
			}
			checked++
		}
	}
	for _, want := range []string{"cp", "bgpal", "bgrgb", "protected"} {
		if !shapes[want] {
			t.Errorf("no recorded cell line of the shape %q: the test does not cover it", want)
		}
	}
	t.Logf("%d recorded cell lines", checked)
}

// writeEscaped against the recorded escaped strings: the uri of the cursor
// hyperlink in the fuzz suites, which has bytes above 0x7e.
func TestCorpusReplaySelfWriteEscapedRecorded(t *testing.T) {
	n, escaped := 0, 0
	for _, name := range []string{"chunked.golden.gz", "mutated.golden.gz", "with_resize.golden.gz"} {
		for _, line := range dumpSelfGoldenLinesN(t, "fuzz", name, 1<<20) {
			uri, ok := strings.CutPrefix(line, "screen.cursor.hyperlink=explicit:q uri=")
			if !ok {
				continue
			}
			var buf bytes.Buffer
			writeEscaped(&buf, dumpSelfUnescape(t, uri))
			if buf.String() != uri {
				t.Errorf("writeEscaped = %q, recorded %q", buf.String(), uri)
			}
			n++
			if strings.Contains(uri, `\x`) {
				escaped++
			}
		}
	}
	if n == 0 || escaped == 0 {
		t.Errorf("%d recorded uri lines, %d with an escape", n, escaped)
	}
}
