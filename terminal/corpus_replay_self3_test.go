package terminal

// Tests of the plumbing of stage 2b of the corpus replayer: the trace kinds
// (corpus_replay_trace_test.go), the markers, and the search area
// (corpus_replay_search_test.go). None of them needs the port: they run today.
// `go test ./terminal -run CorpusReplaySelf`.
//
// Where a check compares with an expected text, the text is a recorded line or
// block of the golden files as far as the port is not needed to reproduce it.
//
// TODO(recorded): the checks that still use hand-written values, and why:
//   - TestCorpusReplaySelfActionHandler: the flush of an empty run and the start
//     of a second run after a flush (a recorded section ends with its flush, and
//     the print action is the only one that does not need StreamActionKey.String,
//     the port's);
//   - TestCorpusReplaySelfMarkerProduce: the combinations of the markers in one
//     case (the corpus has no case with two markers);
//   - TestCorpusReplaySelfSearchParseSuite: bad inputs (parseSuite of search.zig
//     on lines that the corpus does not have);
//   - TestCorpusReplaySelfSearchSelect: the selection of made-up cases.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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

// corpusSelfSections reads every section of a golden file of an area (the
// text of each, header line included) with the name of its case.
func corpusSelfSections(t *testing.T, area corpusArea, file string) (names []string, texts []string) {
	t.Helper()
	s := corpusSectionsFile(t, area, file)
	defer s.Close()
	for {
		name, text, ok, err := s.read(true)
		if err != nil {
			t.Fatalf("%s/%s: %v", area.name, file, err)
		}
		if !ok {
			return names, texts
		}
		names = append(names, name)
		texts = append(texts, string(text))
	}
}

// corpusSelfBody is the text of a section without its `## case` line.
func corpusSelfBody(text string) string {
	_, body, _ := strings.Cut(text, "\n")
	return body
}

// ---------------------------------------------------------------------------
// The trace kinds.

// The parser kind needs only the parser of the port (Parser.go): every trace
// of every parser suite is replayed through corpusProduce (runTrace) and
// compared with the recorded one, all cases, except the traces that contain an
// `@osc_dispatch`: the parser of the port yields no OSC command until osc.go
// is ported, so those traces cannot match yet. This is a limit of THIS
// plumbing self-test only (it must pass today); TestCorpusReplayParser has no
// exclusion and fails for them until osc.go is ported.
func TestCorpusReplaySelfParserTrace(t *testing.T) {
	area := corpusAreaByName(t, "parser")
	compared, needOSC, bad := 0, 0, 0
	suitesRead := 0
	for _, suite := range corpusSuiteNames(t, area) {
		parsed := corpusReadSuite(t, area, suite)
		if !parsed.kinds.contains(replayKindParser) {
			continue
		}
		// With -short only the first five suites (a few seconds under qemu).
		if suitesRead++; testing.Short() && suitesRead > 5 {
			continue
		}
		names, texts := corpusSelfSections(t, area, corpusGoldenName(suite, replayKindParser))
		if len(texts) != len(parsed.cases) {
			t.Fatalf("%s: %d sections, %d cases", suite, len(texts), len(parsed.cases))
		}
		for i, c := range parsed.cases {
			if names[i] != c.name {
				t.Fatalf("%s: section %d is %q, the case is %q", suite, i, names[i], c.name)
			}
			if strings.Contains(texts[i], "\n@osc_dispatch\n") {
				needOSC++
				continue
			}
			compared++
			got := corpusProduce(c, replayKindParser)
			if diff, equal := corpusCompare(got, []byte(texts[i]), nil); !equal {
				bad++
				if bad <= 5 {
					t.Errorf("%s/%s: line %d\n got  %s\n want %s", suite, c.name, diff.line, diff.got, diff.want)
				}
			}
		}
	}
	t.Logf("parser traces: %d compared, %d with an @osc_dispatch (need osc.go), %d differ", compared, needOSC, bad)
	if !testing.Short() && (compared+needOSC != 9010 || needOSC != 393) {
		// 25 of the 27 suites have the parser kind; 9,010 traces in all.
		t.Errorf("%d compared and %d with an @osc_dispatch, want 9010 in all and 393 with one", compared, needOSC)
	}
	if bad > 0 {
		t.Errorf("%d traces differ", bad)
	}
}

// The print run of the action handler against the recorded lines: every
// `@print` line of the first sections of two suites is rebuilt from its code
// points by Vt and flush. The other actions go through StreamActionKey.String
// (the port's): dumpActionPayload is tested below.
func TestCorpusReplaySelfActionHandler(t *testing.T) {
	lines := 0
	for _, f := range []struct{ area, name string }{{"csi", "cup.actions.gz"}, {"parser", "utf8_decode.actions.gz"}} {
		for _, l := range dumpSelfGoldenLinesN(t, f.area, f.name, 256<<10) {
			rest, ok := strings.CutPrefix(l, "@print ")
			if !ok {
				continue
			}
			var buf bytes.Buffer
			h := &actionHandler{w: &buf}
			for _, hexcp := range strings.Split(rest, ",") {
				cp, err := strconv.ParseUint(hexcp, 16, 32)
				if err != nil {
					t.Fatalf("%q: %v", l, err)
				}
				if err := h.Vt(StreamActionKeyPrint, StreamAction{Tag: StreamActionKeyPrint, Print: StreamActionPrint{cp: uint32(cp)}}); err != nil {
					t.Fatal(err)
				}
			}
			if buf.String() != l {
				t.Errorf("Vt wrote %q before the flush, the recorded line is %q", buf.String(), l)
			}
			h.flush()
			if buf.String() != l+"\n" {
				t.Fatalf("after flush %q, want %q", buf.String(), l+"\n")
			}
			lines++
		}
	}
	if lines < 100 {
		t.Errorf("only %d recorded @print lines", lines)
	}

	// An empty run writes nothing; after a flush a print starts a new run.
	var buf bytes.Buffer
	h := &actionHandler{w: &buf}
	h.flush()
	h.Vt(StreamActionKeyPrint, StreamAction{Print: StreamActionPrint{cp: 0x41}})
	h.flush()
	h.flush()
	h.Vt(StreamActionKeyPrint, StreamAction{Print: StreamActionPrint{cp: 0x1f600}})
	h.Vt(StreamActionKeyPrint, StreamAction{Print: StreamActionPrint{cp: 0x7e}})
	h.flush()
	if got, want := buf.String(), "@print 41\n@print 1f600,7e\n"; got != want {
		t.Errorf("runs %q, want %q", got, want)
	}
}

// dumpActionPayload against the recorded blocks: `@<tag>` and the lines of the
// payload, for the payloads that need no enum name (the String methods of the
// port's enums are the port's).
func TestCorpusReplaySelfActionPayloads(t *testing.T) {
	kfl := func(f KittyKeyFlags) StreamActionKittyKeyboardFlags { return StreamActionKittyKeyboardFlags{flags: f} }
	for _, c := range []struct {
		name  string
		value StreamAction
		file  string // the recorded file (below golden/) that has the block
	}{
		{"cursor_pos", StreamAction{CursorPos: StreamActionCursorPos{row: 1, col: 1}}, "csi/misc"},
		{"cursor_up", StreamAction{CursorUp: StreamActionCursorMovement{value: 1}}, "parser/nextslicecapped_inputs"},
		{"print_repeat", StreamAction{PrintRepeat: 3}, "text/rep"},
		{"horizontal_tab", StreamAction{HorizontalTab: 1}, "parser/state_csi_ignore"},
		{"erase_display_below", StreamAction{EraseDisplayBelow: true}, "parser/csi_intermediates"},
		{"top_and_bottom_margin", StreamAction{TopAndBottomMargin: StreamActionMargin{top_left: 2, bottom_right: 4}}, "esc/reset"},
		{"title_push", StreamAction{TitlePush: 0}, "csi/misc"},
		{"kitty_keyboard_set", StreamAction{KittyKeyboardSet: kfl(KittyKeyFlags{})}, "modes/kitty_keyboard"},
		{"kitty_keyboard_set", StreamAction{KittyKeyboardSet: kfl(KittyKeyFlags{disambiguate: true, report_alternates: true})}, "fuzz/structured_short"},
		{"kitty_keyboard_pop", StreamAction{KittyKeyboardPop: 3}, "modes/kitty_keyboard"},
		{"start_hyperlink", StreamAction{StartHyperlink: StreamActionStartHyperlink{uri: []uint8("http://example.com"), id: Some([]uint8("a"))}}, "esc/reset"},
		{"report_pwd", StreamAction{ReportPwd: StreamActionReportPwd{url: []uint8("file:///tmp")}}, "esc/reset"},
		{"dcs_hook", StreamAction{DcsHook: ParserActionDCS{Intermediates: []uint8("$"), Params: []uint16{}, Final: 113}}, "dcs/all"},
		{"dcs_put", StreamAction{DcsPut: 109}, "dcs/all"},
		{"window_title", StreamAction{WindowTitle: StreamActionWindowTitle{title: []uint8("hello world")}}, "osc/all"},
		{"window_title", StreamAction{WindowTitle: StreamActionWindowTitle{title: []uint8("t\u00eftl\u00e9 \u4e2d \U0001f600")}}, "osc/all"},
		{"mouse_shift_capture", StreamAction{MouseShiftCapture: false}, "csi/misc"},
		// void: nothing but the tag line
		{"decaln", StreamAction{}, "esc/reset"},
	} {
		var buf bytes.Buffer
		buf.WriteString("@" + c.name + "\n")
		dumpActionPayload(&buf, " v", c.value, c.name)
		recorded := string(corpusReadAll(t, "golden/"+c.file+".actions.gz"))
		if !strings.Contains(recorded, buf.String()) {
			t.Errorf("%s: %q is not in golden/%s.actions.gz", c.name, buf.String(), c.file)
		}
	}
	// A tag that is not void in Zig but has no field, and a void tag that is
	// not listed, are loud.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("dumpActionPayload of an unknown tag did not panic")
			}
		}()
		dumpActionPayload(&bytes.Buffer{}, " v", StreamAction{}, "no_such_tag")
	}()
}

// ---------------------------------------------------------------------------
// The markers.

// corpusSelfMarkerPrefix is what produce writes before the output of a case
// whose ops have these markers (main.zig:322): the lines in order, each marker
// at most once; upstream_invalid ends the case.
func corpusSelfMarkerPrefix(ops []replayOp) string {
	var out string
	for _, op := range ops {
		if op.Tag == replayOpTransientInvariant {
			out += "!transient-upstream-invariant-failure-end-state-verified\n"
			break
		}
	}
	for _, op := range ops {
		if op.Tag == replayOpUpstreamInvalid {
			return out + "!upstream-integrity-failure\n"
		}
	}
	for _, op := range ops {
		if op.Tag == replayOpExpectCrash {
			out += "!recorded-from-corrected-upstream\n"
			break
		}
	}
	return out
}

// The marked cases of the corpus: the marker is the first op of the case, every
// golden section of a marked case starts with its lines and no other section
// starts with a `!` line, in every kind file; the numbers are the corpus's
// (114 expect_crash and 951 transient_invariant cases, 340 and 1,902 sections).
// Which files are read: the files of the suites that have a marked case, every
// file of the trace kinds (not with -short), and with TERM51_FULL every file.
func TestCorpusReplaySelfMarkerFirstLines(t *testing.T) {
	full := os.Getenv("TERM51_FULL") != ""
	type key struct{ suite, kind string }
	wantCases := map[string][2]int{ // area/suite: expect_crash, transient_invariant
		"osc/all": {72, 0}, "osc/unterminated": {40, 0}, "text/grapheme_churn": {2, 0},
		"fuzz/resize_scrolled": {0, 949}, "fuzz/with_resize": {0, 2},
	}
	gotCases := map[string][2]int{}
	sections := map[key][2]int{} // sections whose first line is the marker's
	files, read := 0, 0
	for _, area := range corpusAreas {
		for _, suite := range corpusSuiteNames(t, area) {
			parsed := corpusReadSuite(t, area, suite)
			id := area.name + "/" + suite
			var counts [2]int
			for _, c := range parsed.cases {
				prefix := corpusSelfMarkerPrefix(c.ops)
				if strings.Contains(prefix, "!upstream-integrity-failure") {
					t.Errorf("%s/%s: upstream_invalid, which the corpus has 0 of", id, c.name)
				}
				for k, tag := range []replayOpTag{replayOpExpectCrash, replayOpTransientInvariant} {
					n := 0
					for _, op := range c.ops {
						if op.Tag == tag {
							n++
						}
					}
					if n > 1 || n == 1 && c.ops[0].Tag != tag {
						t.Errorf("%s/%s: marker %d times, first op is %d: a marker must be the first op, once", id, c.name, n, c.ops[0].Tag)
					}
					counts[k] += n
				}
			}
			if counts != [2]int{} {
				gotCases[id] = counts
			}
			for _, kind := range parsed.kinds.kinds() {
				// With -short the trace files of the suites without a marked case
				// are not read (the big areas take seconds under qemu).
				if counts == [2]int{} && !full && (kind == replayKindGolden || testing.Short()) {
					continue
				}
				names, texts := corpusSelfSections(t, area, corpusGoldenName(suite, kind))
				files++
				read += len(texts)
				if len(texts) != len(parsed.cases) {
					t.Fatalf("%s.%s: %d sections, %d cases", id, kind, len(texts), len(parsed.cases))
				}
				var found [2]int
				for i, c := range parsed.cases {
					if names[i] != c.name {
						t.Fatalf("%s.%s: section %d is %q, the case is %q", id, kind, i, names[i], c.name)
					}
					body := corpusSelfBody(texts[i])
					prefix := corpusSelfMarkerPrefix(c.ops)
					switch {
					case prefix != "" && !strings.HasPrefix(body, prefix):
						t.Errorf("%s.%s %s: section starts with %q, want the lines %q", id, kind, c.name, firstLine(body), prefix)
					case prefix == "" && strings.HasPrefix(body, "!"):
						t.Errorf("%s.%s %s: unmarked case, but the section starts with %q", id, kind, c.name, firstLine(body))
					}
					for k, line := range []string{"!recorded-from-corrected-upstream", "!transient-upstream-invariant-failure-end-state-verified"} {
						if strings.Contains(prefix, line) {
							found[k]++
						}
					}
				}
				sections[key{id, kind.String()}] = found
			}
		}
	}
	if !reflect.DeepEqual(gotCases, wantCases) {
		t.Errorf("marked cases %v, want %v", gotCases, wantCases)
	}
	var crash, transient int
	for _, f := range sections {
		crash += f[0]
		transient += f[1]
	}
	if crash != 340 || transient != 1902 {
		t.Errorf("%d expect_crash and %d transient_invariant sections, want 340 and 1902", crash, transient)
	}
	// Per golden file: the number of marked sections is the number of marked
	// cases, in every kind file the suite has.
	for k, f := range sections {
		w := wantCases[k.suite]
		if f != w {
			t.Errorf("%s.%s: %v sections with a marker line, the suite has %v marked cases", k.suite, k.kind, f, w)
		}
	}
	t.Logf("%d files, %d sections read; expect_crash sections %d, transient_invariant sections %d", files, read, crash, transient)
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// produce writes the marker lines first and then, for upstream_invalid, nothing
// else (main.zig:322), for every kind. The parser kind runs today, so the
// combinations are checked on it: the output with markers is the lines and then
// the output without.
func TestCorpusReplaySelfMarkerProduce(t *testing.T) {
	feedOp := replayOp{Tag: replayOpFeed, Feed: []uint8("\x1b[2;3H\x1b]0;x\x07a")}
	var base bytes.Buffer
	if err := produce([]replayOp{feedOp}, replayKindParser, feedMode{}, &base); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(base.String(), "> 1b state=escape\n") {
		t.Fatalf("trace %q", base.String())
	}
	markers := []replayOpTag{replayOpExpectCrash, replayOpTransientInvariant, replayOpUpstreamInvalid}
	for mask := 1; mask < 8; mask++ {
		var ops []replayOp
		for i, m := range markers {
			if mask&(1<<i) != 0 {
				ops = append(ops, replayOp{Tag: m})
			}
		}
		ops = append(ops, feedOp)
		prefix := corpusSelfMarkerPrefix(ops)
		invalid := mask&4 != 0
		for _, kind := range []replayKind{replayKindGolden, replayKindActions, replayKindActionsNoalloc, replayKindParser} {
			if !invalid && kind != replayKindParser {
				continue // needs the port
			}
			var buf bytes.Buffer
			if err := produce(ops, kind, feedMode{}, &buf); err != nil {
				t.Fatalf("mask %d kind %s: %v", mask, kind, err)
			}
			want := prefix + base.String()
			if invalid {
				want = prefix
			}
			if buf.String() != want {
				t.Errorf("mask %d kind %s: %q, want %q", mask, kind, buf.String(), want)
			}
		}
	}

	// The lines are the ones of main.zig, in its order.
	text := string(corpusReadAll(t, "harness/src/main.zig"))
	i := strings.Index(text, "fn produce(")
	if i < 0 {
		t.Fatal("produce not found in main.zig")
	}
	last := i
	for _, line := range []string{
		"!transient-upstream-invariant-failure-end-state-verified", "!upstream-integrity-failure",
		"!upstream-crash", "!recorded-from-corrected-upstream",
	} {
		j := strings.Index(text[i:], line)
		if j < 0 || i+j < last {
			t.Errorf("main.zig produce: %q missing or out of order", line)
		}
		last = i + j
	}

	// The marker op of a transient_invariant case switches the integrity checks
	// on, the others do nothing to the state (the Go arm of main.zig:273).
	st := &runCaseState{}
	runCaseExpectCrash(st)
	if st.integrity {
		t.Error("expect_crash switched the integrity checks on")
	}
	runCaseTransientInvariant(st)
	if !st.integrity {
		t.Error("transient_invariant did not switch the integrity checks on")
	}
}

// The integrity check of a transient_invariant case after each resize op
// (runCaseResize returns its error) must not be dropped by runCase: the call is
// not an expression statement. A silent no-op would pass every case.
func TestCorpusReplaySelfRunCaseResizeChecked(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "corpus_replay_run_test.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	calls, dropped := 0, 0
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "runCase" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ExprStmt:
				if call, ok := n.X.(*ast.CallExpr); ok {
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "runCaseResize" {
						dropped++
					}
				}
			case *ast.IfStmt:
				if as, ok := n.Init.(*ast.AssignStmt); ok {
					if call, ok := as.Rhs[0].(*ast.CallExpr); ok {
						if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "runCaseResize" {
							calls++
						}
					}
				}
			}
			return true
		})
	}
	if calls != 1 || dropped != 0 {
		t.Errorf("runCase: %d checked calls of runCaseResize and %d with the error dropped, want 1 and 0", calls, dropped)
	}
}

// ---------------------------------------------------------------------------
// The search area: case files, marks, markers, echo and records.

// searchSelfSuites parses every case file of the search area.
func searchSelfSuites(t *testing.T) map[string]searchSuite {
	t.Helper()
	out := map[string]searchSuite{}
	for _, suite := range corpusSuiteNames(t, searchArea) {
		out[suite] = corpusReadSearchSuite(t, suite)
	}
	return out
}

func TestCorpusReplaySelfSearchParseSuite(t *testing.T) {
	text := "# comment\r\n" +
		"kinds search\n" +
		"case a\n" +
		"unchecked_only\n" +
		"init 1000 4 10000000\n" +
		"write -\n" +
		"repeat 45 0d0a\n" +
		"resize 10 5\n" +
		"scroll_viewport -3\n" +
		"scroll_top\n" +
		"scroll_active\n" +
		"reset\n" +
		"open screen 6162\n" +
		"open pagelist 61 start=2 screen=alternate\n" +
		"open pagelist 61 start=last\n" +
		"open window 61 dir=reverse\n" +
		"open window 61 dir=forward\n" +
		"open viewport 61 dirty=true\n" +
		"open viewport 61 dirty=null\n" +
		"close\n" +
		"update\n" +
		"next\n" +
		"drain\n" +
		"drain 7\n" +
		"feed\n" +
		"tick\n" +
		"search_all\n" +
		"select_next\n" +
		"select_prev\n" +
		"update_active\n" +
		"matches\n" +
		"selected\n" +
		"append 3\n" +
		"append_all\n" +
		"append_all_rev\n" +
		"clear\n" +
		"viewport_reset\n" +
		"viewport_dirty false\n" +
		"no_result\n" +
		"case b\n" +
		"no_result abort\n" +
		"case empty\n"
	got, err := parseSearchSuite(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.cases) != 3 || got.cases[0].name != "a" || got.cases[1].name != "b" || got.cases[2].name != "empty" || len(got.cases[2].ops) != 0 {
		t.Fatalf("cases %+v", got.cases)
	}
	ops := got.cases[0].ops
	wantTags := []searchOpTag{
		searchOpUncheckedOnly, searchOpInit, searchOpWrite, searchOpRepeat, searchOpResize, searchOpScrollViewport,
		searchOpScrollTop, searchOpScrollActive, searchOpReset, searchOpOpen, searchOpOpen, searchOpOpen, searchOpOpen,
		searchOpOpen, searchOpOpen, searchOpOpen, searchOpClose, searchOpUpdate, searchOpNext, searchOpDrain, searchOpDrain,
		searchOpFeed, searchOpTick, searchOpSearchAll, searchOpSelectNext, searchOpSelectPrev, searchOpUpdateActive,
		searchOpMatches, searchOpSelected, searchOpAppend, searchOpAppendAll, searchOpAppendAllRev, searchOpClear,
		searchOpViewportReset, searchOpViewportDirty, searchOpNoResult,
	}
	var gotTags []searchOpTag
	for _, op := range ops {
		gotTags = append(gotTags, op.Tag)
	}
	if !slices.Equal(gotTags, wantTags) {
		t.Fatalf("tags %v, want %v", gotTags, wantTags)
	}
	if ops[0].Reason != "underflow" || ops[len(ops)-1].Reason != "unknown" || got.cases[1].ops[0].Reason != "abort" {
		t.Errorf("reasons %q %q %q", ops[0].Reason, ops[len(ops)-1].Reason, got.cases[1].ops[0].Reason)
	}
	if ops[1].Init != (replayOpInitArgs{cols: 1000, rows: 4, max_scrollback: 10000000}) ||
		ops[2].Write == nil || len(ops[2].Write) != 0 ||
		ops[3].Repeat.n != 45 || !bytes.Equal(ops[3].Repeat.bytes, []byte("\r\n")) ||
		ops[4].Resize != (replayOpResizeArgs{cols: 10, rows: 5}) || ops[5].ScrollViewport != -3 ||
		ops[19].Drain != 100000 || ops[20].Drain != 7 || ops[29].Append != 3 ||
		ops[34].ViewportDirty != Some(false) {
		t.Errorf("payloads: %+v", ops)
	}
	open := func(i int) searchOpen { return ops[i].Open }
	if o := open(9); o.kind != searchKindScreen || string(o.needle) != "ab" || o.screen != ScreenSetKeyPrimary || o.start.Valid || o.reverse || o.dirty.Valid {
		t.Errorf("open screen: %+v", o)
	}
	if o := open(10); o.kind != searchKindPagelist || o.start != Some(uint(2)) || o.screen != ScreenSetKeyAlternate {
		t.Errorf("open pagelist start=2: %+v", o)
	}
	if o := open(11); o.start.Valid {
		t.Errorf("open pagelist start=last: %+v", o)
	}
	if !open(12).reverse || open(13).reverse || open(14).dirty != Some(true) || open(15).dirty.Valid || open(15).kind != searchKindViewport {
		t.Errorf("open window/viewport options")
	}

	for _, bad := range []string{
		"init 1 1 1\n", "case a\nwrite 123\n", "case a\nwrite zz\n", "case a\nwrite\n", "case a\nrepeat x 61\n", "case a\nrepeat 2\n",
		"case a\nopen nope 61\n", "case a\nopen screen\n", "case a\nopen screen 6\n", "case a\nopen screen 61 foo\n",
		"case a\nopen screen 61 screen=middle\n", "case a\nopen window 61 dir=up\n", "case a\nopen viewport 61 dirty=maybe\n",
		"case a\nopen pagelist 61 start=x\n", "case a\nopen screen 61 bogus=1\n", "case a\nviewport_dirty maybe\n", "case a\nviewport_dirty\n",
		"case a\nfrob\n", "case\n", "case a\nappend\n", "case a\ndrain x\n", "case a\nscroll_viewport 1.5\n",
		"case a\ninit 70000 1 1\n", "case a\ninit 1 1 -1\n", "case a\ninit 1 1\n", "case a\nresize 1\n",
	} {
		if _, err := parseSearchSuite(bad); err == nil {
			t.Errorf("parseSearchSuite(%q) succeeded", bad)
		}
	}
	if _, err := parseSearchSuite("frob\n"); err != errReplayBadCase {
		t.Errorf("op before the first case: %v, want BadCase", err)
	}
	if _, err := parseSearchSuite("case a\ninit 70000 1 1\n"); err != errZigOverflow {
		t.Errorf("init 70000: %v, want Overflow", err)
	}

	// isSearchSuite: the first line that is neither empty nor a comment.
	for text, want := range map[string]bool{
		"# c\n\nkinds search\ncase a\n": true, "kinds search": true, "kinds golden\n": false, "case a\nkinds search\n": false, "": false,
		"kinds search extra\n": false,
	} {
		if got := isSearchSuite(text); got != want {
			t.Errorf("isSearchSuite(%q) = %v, want %v", text, got, want)
		}
	}
}

// Every case file of the search area is a search suite that parses, with the
// number of cases of golden/MANIFEST; the totals and the marks are the corpus's
// (docs/search-corpus.md).
func TestCorpusReplaySelfSearchParseAll(t *testing.T) {
	manifest := corpusManifest(t)
	suites := searchSelfSuites(t)
	total := 0
	marks := map[string]int{}
	tags := map[searchOpTag]int{}
	for suite, parsed := range suites {
		searchCheckManifest(t, manifest, suite, parsed)
		total += len(parsed.cases)
		for _, c := range parsed.cases {
			if len(c.ops) == 0 {
				t.Errorf("%s/%s: no ops", suite, c.name)
			}
			nMarks := 0
			for i, op := range c.ops {
				tags[op.Tag]++
				switch op.Tag {
				case searchOpUncheckedOnly, searchOpNoResult:
					nMarks++
					marks[op.Tag.String()+" "+op.Reason]++
					if i != 0 {
						t.Errorf("%s/%s: marker %s is op %d, not the first", suite, c.name, op.Tag, i)
					}
				}
			}
			if nMarks > 1 {
				t.Errorf("%s/%s: %d markers", suite, c.name, nMarks)
			}
		}
	}
	// The case files and the MANIFEST have the same suites.
	inManifest := 0
	for path := range manifest {
		if strings.HasPrefix(path, "search/") {
			inManifest++
			if _, ok := suites[strings.TrimSuffix(strings.TrimPrefix(path, "search/"), ".search.gz")]; !ok {
				t.Errorf("MANIFEST has %s, no case file", path)
			}
		}
	}
	wantMarks := map[string]int{
		"unchecked_only underflow": 138, "unchecked_only empty_needle": 2,
		"no_result abort": 48, "no_result empty_needle_segv": 180,
	}
	if len(suites) != 74 || inManifest != 74 || total != 8100 || !reflect.DeepEqual(marks, wantMarks) {
		t.Errorf("%d suites (%d in the MANIFEST), %d cases, marks %v; want 74, 74, 8100, %v", len(suites), inManifest, total, marks, wantMarks)
	}
	// Every op of the Zig is used by some case, but the markers and the ones
	// that only the Go tests use.
	for tag := searchOpInit; tag <= searchOpNoResult; tag++ {
		if tags[tag] == 0 {
			t.Errorf("no case has the op %s", tag)
		}
	}
	t.Logf("%d suites, %d cases (%d with no recorded result), marks %v", len(suites), total, marks["no_result abort"]+marks["no_result empty_needle_segv"], marks)
}

// tools/search_marks.txt: every rule matches cases whose first op is the
// marker with the reason of the rule, and the rules cover exactly the markers of
// the empty_* suites; the other markers are the recorders' (underflow and
// abort), in the suites that docs/search-corpus.md names.
func TestCorpusReplaySelfSearchMarks(t *testing.T) {
	suites := searchSelfSuites(t)
	covered := map[string]bool{} // suite/case
	rules := 0
	for n, line := range corpusLines(t, "tools/search_marks.txt") {
		if line = strings.TrimSpace(line); line == "" || line[0] == '#' {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 4 || f[2] != "unchecked_only" && f[2] != "no_result" {
			t.Fatalf("search_marks.txt line %d: %q", n+1, line)
		}
		rules++
		suite, ok := suites[f[0]]
		if !ok {
			t.Errorf("rule %q: no suite %s", line, f[0])
			continue
		}
		matched := 0
		for _, c := range suite.cases {
			if f[1] != "*" && f[1] != c.name {
				continue
			}
			matched++
			covered[f[0]+"/"+c.name] = true
			tag := searchOpNoResult
			if f[2] == "unchecked_only" {
				tag = searchOpUncheckedOnly
			}
			if len(c.ops) == 0 || c.ops[0].Tag != tag || c.ops[0].Reason != f[3] {
				t.Errorf("rule %q: the case %s does not start with the marker", line, c.name)
			}
		}
		if matched == 0 || f[1] == "*" && matched != len(suite.cases) {
			t.Errorf("rule %q matched %d of %d cases", line, matched, len(suite.cases))
		}
	}
	// The rules cover every marker of the empty_* suites, and only those.
	for name, suite := range suites {
		for _, c := range suite.cases {
			marked := len(c.ops) > 0 && (c.ops[0].Tag == searchOpNoResult || c.ops[0].Tag == searchOpUncheckedOnly)
			isEmpty := strings.HasPrefix(name, "empty_")
			switch {
			case isEmpty && marked != covered[name+"/"+c.name] && marked:
				t.Errorf("%s/%s: marked, but no rule of search_marks.txt covers it", name, c.name)
			case !isEmpty && covered[name+"/"+c.name]:
				t.Errorf("%s/%s: covered by a rule, but not an empty_* suite", name, c.name)
			case !isEmpty && marked:
				// Decided by the recorders: underflow (unchecked_only) in the
				// underflow suites and the window/viewport/pagelist/active
				// suites that hit sliding_window.zig:279, abort in changes_prune.
				if reason := c.ops[0].Reason; reason != "underflow" && reason != "abort" {
					t.Errorf("%s/%s: reason %q, a recorder decides underflow or abort", name, c.name, reason)
				}
			}
		}
	}
	if rules != 22 {
		t.Errorf("%d rules in search_marks.txt, want 22", rules)
	}
}

// The first lines of the recorded sections, the echo of the ops and the
// records, against the case files and the formats of the replayer, in every
// golden file of the search area (21 MB inflated):
//   - a no_result case: its section is what runCase writes, the status line
//     (the port is not needed);
//   - an unchecked_only case: the status line of the unchecked recorder first,
//     never the safe recorder's `!needs-unchecked-release`;
//   - any other case: `op 1` first; the `op <n> <text>` lines are the echo of the
//     ops, in order;
//   - every layout, hl, win and st line is parsed and written again by the
//     format methods of the records.
func TestCorpusReplaySelfSearchMarkers(t *testing.T) {
	suites := searchSelfSuites(t)
	cases, noResult, unchecked, opLines, records := 0, 0, 0, 0, map[string]int{}
	for _, suite := range corpusSuiteNames(t, searchArea) {
		parsed := suites[suite]
		names, texts := corpusSelfSections(t, searchArea, corpusGoldenName(suite, replayKindSearch))
		if len(texts) != len(parsed.cases) {
			t.Fatalf("%s: %d sections, %d cases", suite, len(texts), len(parsed.cases))
		}
		for i, c := range parsed.cases {
			cases++
			if names[i] != c.name {
				t.Fatalf("%s: section %d is %q, the case is %q", suite, i, names[i], c.name)
			}
			text := texts[i]
			body := corpusSelfBody(text)
			if strings.HasPrefix(body, "!needs-unchecked-release") {
				t.Errorf("%s/%s: recorded by the safe recorder", suite, c.name)
			}
			if r := searchMarker(c, searchOpNoResult); r.Valid {
				noResult++
				// The status line of runCase, without the port.
				var buf bytes.Buffer
				if _, err := searchProduceSuite([]searchCase{c}, &buf); err != nil || buf.String() != text {
					t.Errorf("%s/%s: runCase wrote %q (%v), the section is %q", suite, c.name, buf.String(), err, text)
				}
				if text != "## case "+c.name+"\n!no-recorded-result "+r.Value+"\n" {
					t.Errorf("%s/%s: section %q", suite, c.name, text)
				}
				continue
			}
			if r := searchMarker(c, searchOpUncheckedOnly); r.Valid {
				unchecked++
				if want := "!recorded-from-unchecked-release reason=" + r.Value; firstLine(body) != want {
					t.Errorf("%s/%s: first line %q, want %q", suite, c.name, firstLine(body), want)
				}
				body = corpusSelfBody(body)
			}
			if !strings.HasPrefix(body, "op 1 ") {
				t.Errorf("%s/%s: the first record is %q, not op 1", suite, c.name, firstLine(body))
			}

			// The echo of the ops.
			var echo bytes.Buffer
			ctx := &searchCtx{w: &echo}
			n := 0
			for _, op := range c.ops {
				if op.Tag == searchOpUncheckedOnly || op.Tag == searchOpNoResult {
					continue
				}
				n++
				ctx.echo(n, op)
			}
			var gotOps []string
			for _, l := range strings.Split(body, "\n") {
				if strings.HasPrefix(l, "op ") {
					gotOps = append(gotOps, l)
				}
				if line, kind := searchSelfReformat(l); kind != "" {
					records[kind]++
					if line != l {
						t.Errorf("%s/%s: the record\n  %s\nis written as\n  %s", suite, c.name, l, line)
					}
				}
			}
			opLines += len(gotOps)
			if want := strings.TrimSuffix(echo.String(), "\n"); strings.Join(gotOps, "\n") != want {
				t.Errorf("%s/%s: the op lines of the section differ from the echo of the ops:\n got  %q\n want %q", suite, c.name, gotOps, want)
			}
		}
	}
	if cases != 8100 || noResult != 228 || unchecked != 140 {
		t.Errorf("%d cases, %d with no result, %d unchecked_only; want 8100, 228, 140", cases, noResult, unchecked)
	}
	for _, k := range []string{"layout", "hl", "win", "st"} {
		if records[k] == 0 {
			t.Errorf("no %s record in the golden files", k)
		}
	}
	t.Logf("%d cases, %d no_result, %d unchecked_only, %d op lines, records %v", cases, noResult, unchecked, opLines, records)
}

// searchSelfReformat parses a layout, hl, win or st line of a recorded section
// and writes it again; kind is "" for another line.
func searchSelfReformat(l string) (line string, kind string) {
	switch {
	case strings.HasPrefix(l, "layout "):
		kind = "layout"
	case strings.HasPrefix(l, "hl "):
		kind = "hl"
	case strings.HasPrefix(l, "win "):
		kind = "win"
	case strings.HasPrefix(l, "st "):
		kind = "st"
	default:
		return "", ""
	}
	var out string
	var ok bool
	switch kind {
	case "layout":
		var r searchLayoutRec
		if r, ok = searchSelfParseLayout(l); ok {
			out = r.format()
		}
	case "hl":
		var r searchHlRec
		if r, ok = searchSelfParseHl(l); ok {
			out = r.format()
		}
	case "win":
		var r searchWinRec
		if r, ok = searchSelfParseWin(l); ok {
			out = r.format()
		}
	case "st":
		var r searchStateRec
		if r, ok = searchSelfParseState(l); ok {
			out = r.format()
		}
	}
	if !ok {
		return "(unparsed)", kind
	}
	return strings.TrimSuffix(out, "\n"), kind
}

// searchSelfKV reads `key=value` fields in order.
func searchSelfKV(fields []string, keys ...string) ([]string, bool) {
	if len(fields) != len(keys) {
		return nil, false
	}
	out := make([]string, len(keys))
	for i, k := range keys {
		v, ok := strings.CutPrefix(fields[i], k+"=")
		if !ok {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}

func searchSelfUint(s string) uint64 {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return n
}

func searchSelfUints(s string) []uint64 {
	if s == "-" || s == "" {
		return nil
	}
	var out []uint64
	for _, p := range strings.Split(s, ",") {
		out = append(out, searchSelfUint(p))
	}
	return out
}

func searchSelfOptY(s string) Optional[uint32] {
	if s == "none" {
		return None[uint32]()
	}
	return Some(uint32(searchSelfUint(s)))
}

func searchSelfParseLayout(l string) (searchLayoutRec, bool) {
	f := strings.Fields(l)
	if len(f) < 3 {
		return searchLayoutRec{}, false
	}
	v, ok := searchSelfKV(f[2:], "pages", "total_rows", "serial_min", "next_serial", "viewport", "viewport_top",
		"active_top", "cols", "rows", "page_rows", "serials")
	if !ok {
		return searchLayoutRec{}, false
	}
	r := searchLayoutRec{
		name: f[1], pages: uint(searchSelfUint(v[0])), totalRows: uint(searchSelfUint(v[1])),
		serialMin: searchSelfUint(v[2]), nextSerial: searchSelfUint(v[3]), viewport: v[4],
		viewportTop: searchSelfOptY(v[5]), activeTop: searchSelfOptY(v[6]),
		cols: uint16(searchSelfUint(v[7])), rows: uint16(searchSelfUint(v[8])),
		serials: searchSelfUints(v[10]),
	}
	for _, n := range searchSelfUints(v[9]) {
		r.pageRows = append(r.pageRows, uint16(n))
	}
	return r, true
}

func searchSelfParsePoint(s string) Optional[PointCoordinate] {
	if s == "none" {
		return None[PointCoordinate]()
	}
	x, y, _ := strings.Cut(s, ",")
	return Some(PointCoordinate{x: uint16(searchSelfUint(x)), y: uint32(searchSelfUint(y))})
}

func searchSelfParseHl(l string) (searchHlRec, bool) {
	f := strings.Fields(l)
	// The label is `next`, `selected` or `match <i>`.
	i := 1
	for i < len(f) && !strings.HasPrefix(f[i], "top_x=") {
		i++
	}
	if i >= len(f) || i < 2 || len(f) < i+5 {
		return searchHlRec{}, false
	}
	v, ok := searchSelfKV(f[i:i+5], "top_x", "bot_x", "tl", "br", "chunks")
	if !ok {
		return searchHlRec{}, false
	}
	r := searchHlRec{
		label: strings.Join(f[1:i], " "), topX: uint16(searchSelfUint(v[0])), botX: uint16(searchSelfUint(v[1])),
		tl: searchSelfParsePoint(v[2]), br: searchSelfParsePoint(v[3]),
	}
	chunks := f[i+5:]
	if uint64(len(chunks)) != searchSelfUint(v[4]) {
		return searchHlRec{}, false
	}
	for _, c := range chunks {
		parts := strings.Split(c, ":")
		if len(parts) != 3 {
			return searchHlRec{}, false
		}
		span := strings.Split(parts[1], "-")
		if len(span) != 2 {
			return searchHlRec{}, false
		}
		page := None[uint]()
		if parts[0] != "-" {
			page = Some(uint(searchSelfUint(parts[0])))
		}
		r.chunks = append(r.chunks, searchChunkRec{page: page, start: uint16(searchSelfUint(span[0])),
			end: uint16(searchSelfUint(span[1])), serial: searchSelfUint(parts[2])})
	}
	return r, true
}

func searchSelfParseWin(l string) (searchWinRec, bool) {
	f := strings.Fields(l)
	if len(f) != 4 {
		return searchWinRec{}, false
	}
	metas, lens, _ := strings.Cut(strings.TrimPrefix(f[3], "metas="), ":")
	v, ok := searchSelfKV([]string{f[1], f[2], "metas=" + metas}, "data_len", "data_offset", "metas")
	if !ok {
		return searchWinRec{}, false
	}
	r := searchWinRec{dataLen: uint(searchSelfUint(v[0])), dataOffset: uint(searchSelfUint(v[1])), metaLen: uint(searchSelfUint(v[2]))}
	for _, n := range searchSelfUints(lens) {
		r.metas = append(r.metas, uint(n))
	}
	if uint64(len(r.metas)) != searchSelfUint(v[2]) {
		return searchWinRec{}, false
	}
	return r, true
}

// searchSelfParseWinSummary reads `<d>/<o>/<m>`: the lengths of the cell maps
// are not in it.
func searchSelfParseWinSummary(s string) (searchWinRec, bool) {
	p := strings.Split(s, "/")
	if len(p) != 3 {
		return searchWinRec{}, false
	}
	return searchWinRec{dataLen: uint(searchSelfUint(p[0])), dataOffset: uint(searchSelfUint(p[1])),
		metaLen: uint(searchSelfUint(p[2]))}, true
}

func searchSelfParseState(l string) (searchStateRec, bool) {
	f := strings.Fields(l)
	if len(f) != 9 {
		return searchStateRec{}, false
	}
	v, ok := searchSelfKV(f[1:], "state", "total", "active", "history", "sel", "hist", "act_win", "hist_win")
	if !ok {
		return searchStateRec{}, false
	}
	r := searchStateRec{state: v[0], total: uint(searchSelfUint(v[1])), active: uint(searchSelfUint(v[2])),
		history: uint(searchSelfUint(v[3])), hist: v[5] == "1"}
	if v[4] != "none" {
		r.sel = Some(uint(searchSelfUint(v[4])))
	}
	if r.actWin, ok = searchSelfParseWinSummary(v[6]); !ok {
		return searchStateRec{}, false
	}
	if v[7] != "-" {
		w, ok := searchSelfParseWinSummary(v[7])
		if !ok {
			return searchStateRec{}, false
		}
		r.histWin = Some(w)
	}
	return r, true
}

// The records against hand-written lines of the recorder's documented format
// (search.zig header): a record is written as the format says.
func TestCorpusReplaySelfSearchRecords(t *testing.T) {
	for _, c := range []struct{ got, want string }{
		{searchLayoutRec{name: "primary", pages: 0, viewport: "active", viewportTop: None[uint32](), activeTop: None[uint32](),
			cols: 80, rows: 24}.format(),
			"layout primary pages=0 total_rows=0 serial_min=0 next_serial=0 viewport=active viewport_top=none active_top=none cols=80 rows=24 page_rows=- serials=-\n"},
		{searchHlRec{label: "next", topX: 1, botX: 2}.format(), "hl next top_x=1 bot_x=2 tl=none br=none chunks=0\n"},
		{searchHlRec{label: "match 2", topX: 3, botX: 4, tl: Some(PointCoordinate{x: 3, y: 9}), br: None[PointCoordinate](),
			chunks: []searchChunkRec{{page: Some(uint(1)), start: 2, end: 3, serial: 5}, {start: 0, end: 1, serial: 6}}}.format(),
			"hl match 2 top_x=3 bot_x=4 tl=3,9 br=none chunks=2 1:2-3:5 -:0-1:6\n"},
		{searchWinRec{dataLen: 20, dataOffset: 0, metaLen: 1, metas: []uint{5}}.format(), "win data_len=20 data_offset=0 metas=1:5\n"},
		{searchWinRec{}.format(), "win data_len=0 data_offset=0 metas=0\n"},
		// metas=N is meta.Len(), not the length of the iterated list.
		{searchWinRec{metaLen: 2, metas: []uint{5}}.format(), "win data_len=0 data_offset=0 metas=2:5\n"},
		{searchWinRec{metaLen: 3, metas: []uint{1, 2, 3}}.format(), "win data_len=0 data_offset=0 metas=3:1,2,3\n"},
		{searchStateRec{state: "history", total: 1, active: 1, hist: true, actWin: searchWinRec{},
			histWin: Some(searchWinRec{dataLen: 20, metaLen: 1, metas: []uint{9}})}.format(),
			"st state=history total=1 active=1 history=0 sel=none hist=1 act_win=0/0/0 hist_win=20/0/1\n"},
		{searchStateRec{state: "complete", sel: Some(uint(3))}.format(),
			"st state=complete total=0 active=0 history=0 sel=3 hist=0 act_win=0/0/0 hist_win=-\n"},
	} {
		if c.got != c.want {
			t.Errorf("record\n got  %q\n want %q", c.got, c.want)
		}
	}
	// ret and the other lines are not records: not reformatted.
	if _, kind := searchSelfReformat("ret ok"); kind != "" {
		t.Errorf("ret is a record: %q", kind)
	}
	if line, kind := searchSelfReformat("layout primary pages=x"); kind != "layout" || line != "(unparsed)" {
		t.Errorf("a bad layout line: %q %q", line, kind)
	}
}

// The replayer's selection of the search cases (searchSelect): never a
// no_result case, always the first unchecked_only one, the first and the last
// replayable cases, the same result for the same arguments.
func TestCorpusReplaySelfSearchSelect(t *testing.T) {
	mk := func(name string, marker searchOpTag, with bool) searchCase {
		c := searchCase{name: name, ops: []searchOp{{Tag: searchOpInit}}}
		if with {
			c.ops = append([]searchOp{{Tag: marker, Reason: "r"}}, c.ops...)
		}
		return c
	}
	var cases []searchCase
	for i := 0; i < 100; i++ {
		switch {
		case i%10 == 0:
			cases = append(cases, mk(fmt.Sprint("c", i), searchOpNoResult, true))
		case i == 77 || i == 88:
			cases = append(cases, mk(fmt.Sprint("c", i), searchOpUncheckedOnly, true))
		default:
			cases = append(cases, mk(fmt.Sprint("c", i), 0, false))
		}
	}
	cfg := corpusConfig{sample: 5}
	got := searchSelect(cfg, "s", cases, len(cases))
	if len(got) != 6 || got[0] != 1 || got[len(got)-1] != 99 || !slices.Contains(got, 77) || slices.Contains(got, 88) {
		t.Errorf("sample of 5: %v", got)
	}
	for _, i := range got {
		if searchMarker(cases[i], searchOpNoResult).Valid {
			t.Errorf("selected the no_result case %d", i)
		}
	}
	if !slices.Equal(got, searchSelect(cfg, "s", cases, len(cases))) || !sort.IntsAreSorted(got) {
		t.Errorf("not deterministic or not sorted: %v", got)
	}
	// A window of 30 sections: nothing beyond; the first unchecked case is
	// outside, so there is no marked pick.
	for _, i := range searchSelect(cfg, "s", cases, 30) {
		if i >= 30 {
			t.Errorf("window of 30: selected %d", i)
		}
	}
	// n = 0 selects all replayable cases.
	if all := searchSelect(corpusConfig{}, "s", cases, len(cases)); len(all) != 90 {
		t.Errorf("all: %d cases, want the 90 that are not no_result", len(all))
	}
	// TERM51_ONLY: the matching cases, still not the no_result ones.
	cfg.only = regexp.MustCompile(`^search/s/c1[01]$`)
	if got := searchSelect(cfg, "s", cases, len(cases)); !slices.Equal(got, []int{11}) {
		t.Errorf("only c10|c11: %v, want [11] (c10 is no_result)", got)
	}
}

// The replayer never uses the test capacity of the unit tests (512 grapheme
// bytes): no corpus_replay*_test.go file calls testStdCapacity, and the
// production capacity is in place.
func TestCorpusReplaySelfSearchProductionCapacity(t *testing.T) {
	if PageStdCapacity.grapheme_bytes != 8192 {
		t.Errorf("PageStdCapacity.grapheme_bytes = %d, want the production 8192", PageStdCapacity.grapheme_bytes)
	}
	files, err := filepath.Glob("corpus_replay*_test.go")
	if err != nil || len(files) < 8 {
		t.Fatalf("%d files, %v", len(files), err)
	}
	for _, file := range files {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "testStdCapacity" {
					t.Errorf("%s calls testStdCapacity: the replayer uses the production page capacity", file)
				}
			}
			return true
		})
	}
}
