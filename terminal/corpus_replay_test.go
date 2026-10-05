package terminal

// The corpus replayer: replays the recorded corpus of term51 (cases/ and
// golden/) through the Go port and compares the result byte for byte with
// what the pinned upstream recorded. Design: term51/docs/replayer-design.md.
// The Go code mirrors the recorder, harness/src/{main,dump}.zig (later also
// trace, input, tables), one Go function per Zig function or op, each with a
// "Mirrors harness/src/<file>.zig:<line> <ZigName>" comment.
//
// # File layout
//
//   - corpus_test.go          corpusDir, corpusLines, dumpEscaped, the older
//     parser/UTF-8/table tests (unchanged);
//   - corpus_replay_test.go   this file: case loading (parseSuite), the area
//     table (corpusAreas), the golden section files (corpusSections), the
//     manifest, sampling (corpusSelect), comparison and mismatch reports
//     (corpusCompare, corpusReport), the suite driver and the test functions
//     (TestCorpusReplay<Area>);
//   - corpus_replay_run_test.go   main.zig: runCase and its ops, feed,
//     produce, produceSuite;
//   - corpus_replay_dump_test.go  dump.zig: dumpTerminal, dumpScreen,
//     dumpAny and the helpers;
//   - corpus_replay_self_test.go  TestCorpusReplaySelf*: tests of the
//     plumbing that pass without the port (case files against the manifest,
//     sampling, reports, dumpAny, the section loader).
//
// # Corpus
//
// cases/<area>/<suite>.case are the inputs and golden/<area>/<suite>.<kind>.gz
// the outputs, one file per kind named by the suite's `kinds` line (golden,
// actions, actions_noalloc, parser); a golden file is `## case <name>` and the
// output of the case, in the order of the case file. golden/MANIFEST has
// `cases=N` per file. The location is LIBGHOSTTY_VT_CORPUS (corpusDir); a
// missing corpus fails the tests.
//
// # Which kinds are replayed
//
// A kind is compared only when its row of corpusAreas lists it and the suite
// has it. Stage 1: the golden kind of csi (TestCorpusReplayCsi). The other
// areas are registered but have no test function yet.
//
// # Environment variables
//
//	TERM51_FULL=1         every case of every suite (the gate; native only)
//	TERM51_SAMPLE=n       n cases per suite (default 20; with -short 3)
//	-short                n=3 and only the first 2 MB of each golden file
//	TERM51_ONLY=<regexp>  only the cases whose `area/suite/case` match, all of
//	                      them (no sampling); `@file` reads a fail list
//	                      (lines `area/suite:case:kind`)
//	TERM51_FAIL_LIST=file append `area/suite:case:kind` of every mismatch
//	TERM51_DIFF_DIR=dir   write the full got and want of every mismatch
//	TERM51_REPORT_MAX=n   reports per suite (default 10); per test 5n (50)
//	TERM51_STRICT=1       compare the lines of corpusSkipPrefixes too
//	LIBGHOSTTY_VT_CORPUS  the term51 checkout
//
// The selection of cases is deterministic (corpusSelect): the cases i*(n-1)/(N-1),
// plus the first case of each marker kind (expect_crash, upstream_invalid,
// transient_invariant) of the suite.
//
// # Reports
//
//	MISMATCH csi/cup case=<c> kind=<k> [split=op:i,at:n] dump=<label> line <n>
//	  got:  <first differing line, cut at 300 bytes>
//	  want: <the same>
//	  near: <nearest preceding `row` or `[screen` line of want>
//
// `line n` counts the lines of the section, whose first line is `## case`.
// At the end of every suite one line, always logged:
//
//	SUITE csi/cup cases=1330 run=20 skipped=0 mismatch=3 (golden=2) skipped_lines=0
//
// Hooks for stage 2: the other kinds (runTrace in corpus_replay_run_test.go),
// the split runs (feedMode already has bytewise and split), TERM51_SPLIT_MAX.

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// ---------------------------------------------------------------------------
// Case files: harness/src/main.zig Op, Suite, parseSuite.

// replayKind is main.zig:292 Kind: the outputs a suite records.
type replayKind uint8

const (
	replayKindGolden replayKind = iota
	replayKindActions
	// Action trace from a Stream created without an allocator.
	replayKindActionsNoalloc
	replayKindParser

	replayKindCount = iota
)

// replayKindNames are the Zig tag names of Kind, which are also the file
// extensions of the golden files.
var replayKindNames = [replayKindCount]string{"golden", "actions", "actions_noalloc", "parser"}

func (k replayKind) String() string { return replayKindNames[k] }

// replayKindFromString is std.meta.stringToEnum(Kind, s).
func replayKindFromString(s string) (replayKind, bool) {
	for i, name := range replayKindNames {
		if name == s {
			return replayKind(i), true
		}
	}
	return 0, false
}

// replayKindSet is std.EnumSet(Kind).
type replayKindSet uint8

func replayKindSetOf(kinds ...replayKind) replayKindSet {
	var s replayKindSet
	for _, k := range kinds {
		s |= 1 << k
	}
	return s
}

func (s replayKindSet) contains(k replayKind) bool { return s&(1<<k) != 0 }

// kinds lists the members in Kind order.
func (s replayKindSet) kinds() []replayKind {
	var out []replayKind
	for k := replayKind(0); k < replayKindCount; k++ {
		if s.contains(k) {
			out = append(out, k)
		}
	}
	return out
}

// replayOpTag is the tag of main.zig:50 Op.
type replayOpTag uint8

const (
	replayOpInit replayOpTag = iota
	replayOpFeed
	replayOpResize
	replayOpScrollTop
	replayOpScrollActive
	replayOpScrollDelta
	replayOpDump
	// The pinned upstream crashes on this case; the corrected recorder runs it.
	replayOpExpectCrash
	// Upstream fails its own integrity checks on this case; never replayed.
	replayOpUpstreamInvalid
	// Upstream trips an integrity check in the middle of this case but ends in
	// the same state as the release build.
	replayOpTransientInvariant
)

// replayOpInitArgs is the payload of Op.init.
type replayOpInitArgs struct {
	cols           uint16
	rows           uint16
	max_scrollback uint64 // Zig: usize
}

// replayOpResizeArgs is the payload of Op.resize.
type replayOpResizeArgs struct {
	cols uint16
	rows uint16
}

// replayOp is main.zig:50 Op, a tagged union: only the field named by Tag is
// meaningful (Init, Feed, Resize, ScrollDelta, Dump).
type replayOp struct {
	Tag replayOpTag

	Init        replayOpInitArgs
	Feed        []uint8
	Resize      replayOpResizeArgs
	ScrollDelta int // Zig: isize
	Dump        string
}

// replayNamed is main.zig:84 Suite.Named.
type replayNamed struct {
	name string
	ops  []replayOp
}

// replaySuite is main.zig:81 Suite (without the arena).
type replaySuite struct {
	cases []replayNamed
	// Which outputs to record, from a "kinds ..." line.
	kinds replayKindSet
}

// errReplayBadCase is Zig's error.BadCase.
var errReplayBadCase = errors.New("BadCase")

// replayTokens is std.mem.tokenizeScalar(u8, line, ' ').
type replayTokens struct{ rest string }

func (it *replayTokens) next() (string, bool) {
	s := strings.TrimLeft(it.rest, " ")
	if s == "" {
		it.rest = ""
		return "", false
	}
	end := strings.IndexByte(s, ' ')
	if end < 0 {
		end = len(s)
	}
	it.rest = s[end:]
	return s[:end], true
}

// replayParseUint is std.fmt.parseInt for an unsigned type, base 10: an
// optional leading '+' is accepted.
func replayParseUint(s string, bits int) (uint64, error) {
	return strconv.ParseUint(strings.TrimPrefix(s, "+"), 10, bits)
}

// Mirrors harness/src/main.zig:95 parseSuite
//
// A suite file holds many cases; each starts with a "case <name>" line. Ops
// before the first "case" line belong to a case named "default". The arms are
// one function per op (parseOp...), called in the order of the Zig chain.
func parseSuite(text string) (replaySuite, error) {
	var cases []replayNamed
	kinds := replayKindSetOf(replayKindGolden, replayKindActions)
	name := "default"
	var ops []replayOp
	for text != "" {
		var raw string
		raw, text, _ = strings.Cut(text, "\n")
		line := strings.Trim(raw, " \t\r")
		if line == "" || line[0] == '#' {
			continue
		}
		it := &replayTokens{rest: line}
		cmd, _ := it.next()
		switch cmd {
		case "case":
			if len(ops) > 0 || len(cases) > 0 || name != "default" {
				cases = append(cases, replayNamed{name: name, ops: ops})
			}
			n, ok := it.next()
			if !ok {
				return replaySuite{}, errReplayBadCase
			}
			name = n
			ops = nil
		case "init":
			op, err := parseOpInit(it)
			if err != nil {
				return replaySuite{}, err
			}
			ops = append(ops, op)
		case "feed":
			op, err := parseOpFeed(it)
			if err != nil {
				return replaySuite{}, err
			}
			ops = append(ops, op)
		case "resize":
			op, err := parseOpResize(it)
			if err != nil {
				return replaySuite{}, err
			}
			ops = append(ops, op)
		case "scroll":
			op, err := parseOpScroll(it)
			if err != nil {
				return replaySuite{}, err
			}
			ops = append(ops, op)
		case "expect_crash":
			ops = append(ops, replayOp{Tag: replayOpExpectCrash})
		case "upstream_invalid":
			ops = append(ops, replayOp{Tag: replayOpUpstreamInvalid})
		case "transient_invariant":
			ops = append(ops, replayOp{Tag: replayOpTransientInvariant})
		case "kinds":
			var err error
			if kinds, err = parseKinds(it); err != nil {
				return replaySuite{}, err
			}
		case "dump":
			label, ok := it.next()
			if !ok {
				label = "end"
			}
			ops = append(ops, replayOp{Tag: replayOpDump, Dump: label})
		default:
			return replaySuite{}, errReplayBadCase
		}
	}

	cases = append(cases, replayNamed{name: name, ops: ops})
	return replaySuite{cases: cases, kinds: kinds}, nil
}

// Mirrors harness/src/main.zig:116 parseSuite, arm "init"
func parseOpInit(it *replayTokens) (replayOp, error) {
	var v [3]uint64
	for i, bits := range [3]int{16, 16, 64} {
		s, ok := it.next()
		if !ok {
			return replayOp{}, errReplayBadCase
		}
		n, err := replayParseUint(s, bits)
		if err != nil {
			return replayOp{}, err
		}
		v[i] = n
	}
	return replayOp{Tag: replayOpInit, Init: replayOpInitArgs{
		cols: uint16(v[0]), rows: uint16(v[1]), max_scrollback: v[2],
	}}, nil
}

// Mirrors harness/src/main.zig:121 parseSuite, arm "feed"
func parseOpFeed(it *replayTokens) (replayOp, error) {
	hexText, _ := it.next()
	if len(hexText)%2 != 0 {
		return replayOp{}, errReplayBadCase
	}
	buf, err := hex.DecodeString(hexText)
	if err != nil {
		return replayOp{}, err
	}
	return replayOp{Tag: replayOpFeed, Feed: buf}, nil
}

// Mirrors harness/src/main.zig:127 parseSuite, arm "resize"
func parseOpResize(it *replayTokens) (replayOp, error) {
	var v [2]uint64
	for i := range v {
		s, ok := it.next()
		if !ok {
			return replayOp{}, errReplayBadCase
		}
		n, err := replayParseUint(s, 16)
		if err != nil {
			return replayOp{}, err
		}
		v[i] = n
	}
	return replayOp{Tag: replayOpResize, Resize: replayOpResizeArgs{cols: uint16(v[0]), rows: uint16(v[1])}}, nil
}

// Mirrors harness/src/main.zig:131 parseSuite, arm "scroll"
func parseOpScroll(it *replayTokens) (replayOp, error) {
	what, ok := it.next()
	if !ok {
		return replayOp{}, errReplayBadCase
	}
	switch what {
	case "top":
		return replayOp{Tag: replayOpScrollTop}, nil
	case "active":
		return replayOp{Tag: replayOpScrollActive}, nil
	case "delta":
		s, ok := it.next()
		if !ok {
			return replayOp{}, errReplayBadCase
		}
		n, err := strconv.ParseInt(s, 10, strconv.IntSize)
		if err != nil {
			return replayOp{}, err
		}
		return replayOp{Tag: replayOpScrollDelta, ScrollDelta: int(n)}, nil
	}
	return replayOp{}, errReplayBadCase
}

// Mirrors harness/src/main.zig:147 parseSuite, arm "kinds"
func parseKinds(it *replayTokens) (replayKindSet, error) {
	var kinds replayKindSet
	for {
		k, ok := it.next()
		if !ok {
			return kinds, nil
		}
		kind, ok := replayKindFromString(k)
		if !ok {
			return 0, errReplayBadCase
		}
		kinds |= replayKindSetOf(kind)
	}
}

// ---------------------------------------------------------------------------
// The areas of the corpus.

// corpusArea is one row of corpusAreas: a directory of suites of cases/ with
// the matching directory of golden/. A new area is a new row (more fields,
// for the ops, kinds and dumpers of a different recorder, come with it).
type corpusArea struct {
	name      string
	casesDir  string // relative to the corpus
	goldenDir string // relative to the corpus
	// The kinds the test of this area compares (the suites may have more).
	kinds replayKindSet
}

// corpusAreaRow is the row of an area whose test compares the golden kind.
func corpusAreaRow(name string) corpusArea {
	return corpusArea{
		name:      name,
		casesDir:  filepath.Join("cases", name),
		goldenDir: filepath.Join("golden", name),
		kinds:     replayKindSetOf(replayKindGolden),
	}
}

// corpusAreas are the terminal areas of the corpus. Only csi has a test
// function (TestCorpusReplayCsi); the others are the same code with another
// row and get theirs when the first works.
var corpusAreas = []corpusArea{
	corpusAreaRow("csi"),
	corpusAreaRow("dcs"),
	corpusAreaRow("esc"),
	corpusAreaRow("fuzz"),
	corpusAreaRow("modes"),
	corpusAreaRow("osc"),
	corpusAreaRow("parser"),
	corpusAreaRow("resize"),
	corpusAreaRow("scroll"),
	corpusAreaRow("text"),
}

// corpusAreaByName is the row of an area.
func corpusAreaByName(t testing.TB, name string) corpusArea {
	t.Helper()
	for _, a := range corpusAreas {
		if a.name == name {
			return a
		}
	}
	t.Fatalf("no area %q in corpusAreas", name)
	return corpusArea{}
}

// corpusSuiteNames are the suites of an area: the .case files of its
// directory, sorted (listCases, main.zig:402).
func corpusSuiteNames(t testing.TB, area corpusArea) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(corpusDir(t), area.casesDir))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".case") {
			out = append(out, strings.TrimSuffix(e.Name(), ".case"))
		}
	}
	sort.Strings(out)
	return out
}

// corpusReadSuite parses cases/<area>/<suite>.case.
func corpusReadSuite(t testing.TB, area corpusArea, suite string) replaySuite {
	t.Helper()
	text, err := os.ReadFile(filepath.Join(corpusDir(t), area.casesDir, suite+".case"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseSuite(string(text))
	if err != nil {
		t.Fatalf("bad suite %s/%s: %v", area.name, suite, err)
	}
	return parsed
}

// corpusGoldenName is the manifest key and the path below golden/<area> of
// the output of a kind: goldenName (main.zig:420) with the .gz suffix.
func corpusGoldenName(suite string, kind replayKind) string {
	return suite + "." + kind.String() + ".gz"
}

// ---------------------------------------------------------------------------
// The manifest: golden/MANIFEST, `<sha256>  <path>  [cases=N]`.

// corpusManifest maps the path of a file below golden/ to its number of
// cases; files without a count (the tables) are not in the map.
func corpusManifest(t testing.TB) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, line := range corpusLines(t, filepath.Join("golden", "MANIFEST")) {
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			t.Fatalf("bad MANIFEST line %q", line)
		}
		if len(fields) == 3 && strings.HasPrefix(fields[2], "cases=") {
			n, err := strconv.Atoi(fields[2][len("cases="):])
			if err != nil {
				t.Fatalf("bad MANIFEST line %q: %v", line, err)
			}
			out[fields[1]] = n
		}
	}
	return out
}

// corpusCheckManifest checks that the number of cases of every output file of
// a suite equals the number of parsed cases: it guards against sampling a
// truncated file.
func corpusCheckManifest(t testing.TB, manifest map[string]int, area corpusArea, suite string, parsed replaySuite) {
	t.Helper()
	for _, kind := range parsed.kinds.kinds() {
		key := area.name + "/" + corpusGoldenName(suite, kind)
		n, ok := manifest[key]
		switch {
		case !ok:
			t.Errorf("%s is not in golden/MANIFEST", key)
		case n != len(parsed.cases):
			t.Errorf("%s: MANIFEST says cases=%d, the case file has %d", key, n, len(parsed.cases))
		}
	}
}

// ---------------------------------------------------------------------------
// Golden section files.

const corpusSectionPrefix = "## case "

// corpusSections reads the sections of an output file (`## case <name>` and
// the lines of the case) one after the other.
type corpusSections struct {
	r      *bufio.Reader
	closer io.Closer // nil for an in-memory window

	header  []byte // the pending `## case` line, without the newline
	have    bool   // header holds the next section
	started bool
	next    int // index of the next section to read
}

// errCorpusSectionsEnd is returned by section when the file has no more
// sections.
var errCorpusSectionsEnd = errors.New("no more sections")

// corpusSectionsReader reads the sections of an uncompressed stream.
func corpusSectionsReader(r io.Reader) *corpusSections {
	return &corpusSections{r: bufio.NewReaderSize(r, 1<<20)}
}

type corpusGzipFile struct {
	gz *gzip.Reader
	f  *os.File
}

func (c *corpusGzipFile) Close() error {
	c.gz.Close()
	return c.f.Close()
}

// corpusOpenGzip opens a .gz file below the corpus.
func corpusOpenGzip(t testing.TB, rel string) (io.Reader, io.Closer) {
	t.Helper()
	f, err := os.Open(filepath.Join(corpusDir(t), rel))
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		t.Fatalf("%s: %v", rel, err)
	}
	return gz, &corpusGzipFile{gz: gz, f: f}
}

// corpusSectionsFile reads the sections of golden/<dir>/<name>.gz from the
// start; the caller closes it.
func corpusSectionsFile(t testing.TB, area corpusArea, name string) *corpusSections {
	t.Helper()
	r, c := corpusOpenGzip(t, filepath.Join(area.goldenDir, name))
	s := corpusSectionsReader(r)
	s.closer = c
	return s
}

// corpusSectionsWindow reads the sections that lie completely within the
// first `limit` bytes of golden/<dir>/<name>.gz (the -short run).
func corpusSectionsWindow(t testing.TB, area corpusArea, name string, limit int64) *corpusSections {
	t.Helper()
	r, c := corpusOpenGzip(t, filepath.Join(area.goldenDir, name))
	defer c.Close()
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if int64(len(data)) > limit {
		// Cut after the last complete section.
		i := bytes.LastIndex(data, []byte("\n"+corpusSectionPrefix))
		if i < 0 {
			data = nil
		} else {
			data = data[:i+1]
		}
	}
	return corpusSectionsReader(bytes.NewReader(data))
}

func (s *corpusSections) Close() {
	if s.closer != nil {
		s.closer.Close()
	}
}

// fillHeader reads the first line of the file, which must be a header.
func (s *corpusSections) fillHeader() error {
	chunk, err := s.r.ReadSlice('\n')
	switch {
	case err == io.EOF && len(chunk) == 0:
		return nil
	case err != nil || !bytes.HasPrefix(chunk, []byte(corpusSectionPrefix)):
		return fmt.Errorf("the file does not start with %q", corpusSectionPrefix)
	}
	s.header = append(s.header[:0], chunk[:len(chunk)-1]...)
	s.have = true
	return nil
}

// read returns the next section: its case name and, when keep, its text (the
// header line and the lines of the case, each with its newline). A line
// starts a section when it begins with `## case ` (a case output never does).
func (s *corpusSections) read(keep bool) (name string, text []byte, ok bool, err error) {
	if !s.started {
		s.started = true
		if err := s.fillHeader(); err != nil {
			return "", nil, false, err
		}
	}
	if !s.have {
		return "", nil, false, nil
	}
	name = string(s.header[len(corpusSectionPrefix):])
	if keep {
		text = append(text, s.header...)
		text = append(text, '\n')
	}
	s.have = false
	atLineStart := true
	for {
		chunk, err := s.r.ReadSlice('\n')
		complete := err == nil
		if err != nil && err != bufio.ErrBufferFull && err != io.EOF {
			return "", nil, false, err
		}
		if len(chunk) > 0 {
			if atLineStart && complete && bytes.HasPrefix(chunk, []byte(corpusSectionPrefix)) {
				s.header = append(s.header[:0], chunk[:len(chunk)-1]...)
				s.have = true
				return name, text, true, nil
			}
			if keep {
				text = append(text, chunk...)
			}
		}
		atLineStart = complete
		if err == io.EOF {
			return name, text, true, nil
		}
	}
}

// section returns the section with index idx, skipping the ones before it.
// The indexes must not decrease.
func (s *corpusSections) section(idx int) (name string, text []byte, err error) {
	if idx < s.next {
		return "", nil, fmt.Errorf("section %d requested after section %d", idx, s.next-1)
	}
	for {
		keep := s.next == idx
		name, text, ok, err := s.read(keep)
		if err != nil {
			return "", nil, err
		}
		if !ok {
			return "", nil, errCorpusSectionsEnd
		}
		s.next++
		if keep {
			return name, text, nil
		}
	}
}

// count reads the rest of the file and returns the number of sections left.
func (s *corpusSections) count() (int, error) {
	n := 0
	for {
		_, _, ok, err := s.read(false)
		if err != nil || !ok {
			return n, err
		}
		n++
	}
}

// ---------------------------------------------------------------------------
// Configuration and sampling.

const (
	corpusSampleDefault = 20
	corpusSampleShort   = 3
	// With -short, only this many inflated bytes of a golden file are used.
	corpusShortWindow = 2 << 20
	corpusLineMax     = 300 // a differing line is cut at this length in a report
	corpusReportSuite = 10
	corpusReportTest  = 50
)

// corpusSkipPrefixes is the one list of the dump lines that are not compared:
// the layout-only lines (how rows are divided among page nodes and the
// byte-derived capacity of each node), which a port with other page memory
// cannot reproduce. The lines `ident.*` and `pages.page_size`,
// `explicit_max_size`, `min_max_size` are compared. If the Go page accounting
// cannot reproduce those three, add them here and nowhere else.
var corpusSkipPrefixes = []string{"layout."}

// corpusConfig is the environment of a run.
type corpusConfig struct {
	full      bool // TERM51_FULL: every case
	short     bool // -short
	sample    int  // cases per suite
	only      *regexp.Regexp
	onlyList  map[string]bool // `area/suite:case`, from TERM51_ONLY=@file
	failList  string
	diffDir   string
	reportMax int // TERM51_REPORT_MAX, 0 for the defaults
	strict    bool
}

// corpusConfigFrom reads the configuration with getenv (os.Getenv in a run).
func corpusConfigFrom(getenv func(string) string, short bool) (corpusConfig, error) {
	cfg := corpusConfig{
		full:     getenv("TERM51_FULL") != "",
		short:    short,
		sample:   corpusSampleDefault,
		failList: getenv("TERM51_FAIL_LIST"),
		diffDir:  getenv("TERM51_DIFF_DIR"),
		strict:   getenv("TERM51_STRICT") != "",
	}
	if short {
		cfg.sample = corpusSampleShort
	}
	for _, v := range []struct {
		name string
		dst  *int
	}{{"TERM51_SAMPLE", &cfg.sample}, {"TERM51_REPORT_MAX", &cfg.reportMax}} {
		if s := getenv(v.name); s != "" {
			n, err := strconv.Atoi(s)
			if err != nil || n < 0 {
				return cfg, fmt.Errorf("%s=%q: not a number", v.name, s)
			}
			*v.dst = n
		}
	}
	if only := getenv("TERM51_ONLY"); strings.HasPrefix(only, "@") {
		list, err := corpusReadFailList(only[1:])
		if err != nil {
			return cfg, err
		}
		cfg.onlyList = list
	} else if only != "" {
		re, err := regexp.Compile(only)
		if err != nil {
			return cfg, fmt.Errorf("TERM51_ONLY: %v", err)
		}
		cfg.only = re
	}
	return cfg, nil
}

// corpusReadFailList reads a TERM51_FAIL_LIST file: `area/suite:case:kind`
// per line. The kind is ignored: every replayed kind of the case is run.
func corpusReadFailList(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		i := strings.LastIndexByte(line, ':')
		if i < 0 {
			return nil, fmt.Errorf("bad fail list line %q", line)
		}
		out[line[:i]] = true
	}
	return out, nil
}

// exhaustive reports whether every case of a suite that passes wants is run
// (no sampling, no window): TERM51_FULL or a TERM51_ONLY filter.
func (c corpusConfig) exhaustive() bool {
	return c.full || c.only != nil || c.onlyList != nil
}

// wants reports whether the TERM51_ONLY filter lets a case through.
func (c corpusConfig) wants(area, suite, caseName string) bool {
	switch {
	case c.only != nil:
		return c.only.MatchString(area + "/" + suite + "/" + caseName)
	case c.onlyList != nil:
		return c.onlyList[area+"/"+suite+":"+caseName]
	}
	return true
}

// skip is the list of the line prefixes that corpusCompare ignores.
func (c corpusConfig) skip() []string {
	if c.strict {
		return nil
	}
	return corpusSkipPrefixes
}

// suiteMax and testMax are the report caps.
func (c corpusConfig) suiteMax() int {
	if c.reportMax > 0 {
		return c.reportMax
	}
	return corpusReportSuite
}

func (c corpusConfig) testMax() int {
	if c.reportMax > 0 {
		return c.reportMax * (corpusReportTest / corpusReportSuite)
	}
	return corpusReportTest
}

// corpusFirstMarked returns the index of the first case that carries each
// marker op, in increasing order: these cases are always selected.
func corpusFirstMarked(cases []replayNamed) []int {
	var out []int
	for _, tag := range []replayOpTag{replayOpExpectCrash, replayOpUpstreamInvalid, replayOpTransientInvariant} {
		for i, c := range cases {
			found := false
			for _, op := range c.ops {
				if op.Tag == tag {
					found = true
					break
				}
			}
			if found {
				out = append(out, i)
				break
			}
		}
	}
	sort.Ints(out)
	return out
}

// corpusSelect returns the indexes of the cases to run of a suite of `total`
// cases: n of them spread evenly, i*(total-1)/(n-1) for i in 0..n-1 (so the
// first and the last are always in), plus the given marked cases. n <= 0 or
// n >= total selects all. The result is sorted without duplicates; the same
// arguments always give the same result.
func corpusSelect(total, n int, marked []int) []int {
	var picks []int
	switch {
	case total <= 0:
		return nil
	case n <= 0 || n >= total:
		for i := 0; i < total; i++ {
			picks = append(picks, i)
		}
		return picks
	case n == 1:
		picks = []int{0}
	default:
		for i := 0; i < n; i++ {
			picks = append(picks, int(int64(i)*int64(total-1)/int64(n-1)))
		}
	}
	for _, m := range marked {
		if m >= 0 && m < total {
			picks = append(picks, m)
		}
	}
	sort.Ints(picks)
	out := picks[:0]
	for i, p := range picks {
		if i == 0 || p != picks[i-1] {
			out = append(out, p)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Comparison and reports.

// corpusDiff is the first difference of two outputs.
type corpusDiff struct {
	line  int    // 1-based line of the section (line 1 is the `## case` line)
	got   string // the differing lines, cut at corpusLineMax
	want  string
	label string // the label of the dump the line is in, "-" if none
	near  string // the nearest row or [screen line at or before it, in want
}

type corpusLine struct {
	no   int // 1-based line number in the text
	text []byte
}

// corpusSplitLines splits a text at newlines; a text that ends with a newline
// has a last empty line.
func corpusSplitLines(b []byte) [][]byte { return bytes.Split(b, []byte("\n")) }

func corpusHasPrefix(line []byte, prefixes []string) bool {
	for _, p := range prefixes {
		if bytes.HasPrefix(line, []byte(p)) {
			return true
		}
	}
	return false
}

// corpusFilter returns the lines that do not start with a skip prefix.
func corpusFilter(lines [][]byte, skip []string) []corpusLine {
	out := make([]corpusLine, 0, len(lines))
	for i, l := range lines {
		if !corpusHasPrefix(l, skip) {
			out = append(out, corpusLine{no: i + 1, text: l})
		}
	}
	return out
}

func corpusCut(b []byte) string {
	if len(b) > corpusLineMax {
		return string(b[:corpusLineMax]) + "..."
	}
	return string(b)
}

// corpusCompare compares an output with the recorded one, ignoring the lines
// that start with a prefix of skip (on both sides). It returns the first
// difference, and true when there is none.
func corpusCompare(got, want []byte, skip []string) (corpusDiff, bool) {
	if bytes.Equal(got, want) {
		return corpusDiff{}, true
	}
	gl, wl := corpusSplitLines(got), corpusSplitLines(want)
	g, w := corpusFilter(gl, skip), corpusFilter(wl, skip)
	k := 0
	for k < len(g) && k < len(w) && bytes.Equal(g[k].text, w[k].text) {
		k++
	}
	if k == len(g) && k == len(w) {
		return corpusDiff{}, true
	}
	d := corpusDiff{got: "<end of text>", want: "<end of text>"}
	if k < len(g) {
		d.got = corpusCut(g[k].text)
		d.line = g[k].no
	}
	if k < len(w) {
		d.want = corpusCut(w[k].text)
		d.line = w[k].no
	}
	// The label and the header are looked up in want, before the line.
	at := len(wl)
	if k < len(w) {
		at = w[k].no
	}
	d.label = "-"
	for i := at - 1; i >= 0; i-- {
		l := wl[i]
		if d.near == "" && (bytes.HasPrefix(l, []byte("row ")) || bytes.HasPrefix(l, []byte("[screen"))) {
			d.near = corpusCut(l)
		}
		if bytes.HasPrefix(l, []byte("== dump ")) {
			d.label = string(l[len("== dump "):])
			break
		}
	}
	return d, false
}

// corpusSkippedDiffs counts the lines starting with a skip prefix that differ
// between got and want (compared in order of appearance; a line missing on
// one side counts).
func corpusSkippedDiffs(got, want []byte, skip []string) int {
	only := func(b []byte) [][]byte {
		var out [][]byte
		for _, l := range corpusSplitLines(b) {
			if corpusHasPrefix(l, skip) {
				out = append(out, l)
			}
		}
		return out
	}
	g, w := only(got), only(want)
	n := 0
	for i := 0; i < len(g) || i < len(w); i++ {
		if i >= len(g) || i >= len(w) || !bytes.Equal(g[i], w[i]) {
			n++
		}
	}
	return n
}

// corpusMismatch is one mismatch of one case and kind.
type corpusMismatch struct {
	area, suite, caseName string
	kind                  replayKind
	split                 string // "op:i,at:n" of a split run, else ""
	diff                  corpusDiff
}

// corpusFormatMismatch is the report of a mismatch (see the file header).
func corpusFormatMismatch(m corpusMismatch) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "MISMATCH %s/%s case=%s kind=%s", m.area, m.suite, m.caseName, m.kind)
	if m.split != "" {
		fmt.Fprintf(&sb, " split=%s", m.split)
	}
	fmt.Fprintf(&sb, " dump=%s line %d\n", m.diff.label, m.diff.line)
	fmt.Fprintf(&sb, "  got:  %s\n  want: %s", m.diff.got, m.diff.want)
	if m.diff.near != "" {
		fmt.Fprintf(&sb, "\n  near: %s", m.diff.near)
	}
	return sb.String()
}

// corpusSuiteStats are the counters of one suite.
type corpusSuiteStats struct {
	cases        int // in the case file
	run          int
	skipped      int // upstream_invalid: never replayed
	mismatch     [replayKindCount]int
	reported     int // reports logged in full
	skippedLines int // differing lines of corpusSkipPrefixes
}

func (s *corpusSuiteStats) mismatches() int {
	n := 0
	for _, m := range s.mismatch {
		n += m
	}
	return n
}

// corpusSummary is the SUITE line.
func corpusSummary(area, suite string, st *corpusSuiteStats, strict bool) string {
	var kinds []string
	for k := replayKind(0); k < replayKindCount; k++ {
		if st.mismatch[k] > 0 {
			kinds = append(kinds, fmt.Sprintf("%s=%d", k, st.mismatch[k]))
		}
	}
	line := fmt.Sprintf("SUITE %s/%s cases=%d run=%d skipped=%d mismatch=%d", area, suite, st.cases, st.run, st.skipped, st.mismatches())
	if len(kinds) > 0 {
		line += " (" + strings.Join(kinds, " ") + ")"
	}
	name := "skipped_lines"
	if strict {
		name = "layout_lines_differ"
	}
	return fmt.Sprintf("%s %s=%d", line, name, st.skippedLines)
}

// corpusRun is the state shared by the suites of one test function.
type corpusRun struct {
	cfg      corpusConfig
	area     corpusArea
	manifest map[string]int

	reported atomic.Int64 // reports logged in full, all suites
	mu       sync.Mutex   // the fail list file
}

func newCorpusRun(t testing.TB, area corpusArea) *corpusRun {
	t.Helper()
	cfg, err := corpusConfigFrom(os.Getenv, testing.Short())
	if err != nil {
		t.Fatal(err)
	}
	return &corpusRun{cfg: cfg, area: area, manifest: corpusManifest(t)}
}

// corpusReporter is the part of *testing.T that corpusReport uses.
type corpusReporter interface {
	Helper()
	Error(args ...any)
	Fail()
}

// corpusReport counts a mismatch and reports it: in full while the caps of
// the suite and of the test allow, otherwise only as a failure.
func corpusReport(t corpusReporter, run *corpusRun, st *corpusSuiteStats, m corpusMismatch) {
	t.Helper()
	st.mismatch[m.kind]++
	if st.reported < run.cfg.suiteMax() && run.reported.Add(1) <= int64(run.cfg.testMax()) {
		st.reported++
		t.Error(corpusFormatMismatch(m))
	} else {
		t.Fail()
	}
	run.appendFailList(m)
}

func (run *corpusRun) appendFailList(m corpusMismatch) {
	if run.cfg.failList == "" {
		return
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	f, err := os.OpenFile(run.cfg.failList, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s/%s:%s:%s\n", m.area, m.suite, m.caseName, m.kind)
}

// writeDiff writes the full got and want of a mismatch to TERM51_DIFF_DIR.
func (run *corpusRun) writeDiff(t *testing.T, m corpusMismatch, got, want []byte) {
	dir := run.cfg.diffDir
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Logf("TERM51_DIFF_DIR: %v", err)
		return
	}
	base := strings.NewReplacer("/", "_", "\\", "_").Replace(fmt.Sprintf("%s.%s.%s.%s", m.area, m.suite, m.caseName, m.kind))
	for ext, data := range map[string][]byte{"got": got, "want": want} {
		if err := os.WriteFile(filepath.Join(dir, base+"."+ext), data, 0o644); err != nil {
			t.Logf("TERM51_DIFF_DIR: %v", err)
		}
	}
}

// corpusProduce is produceSuite for one case, as the text of one section
// (`## case` line included). A panic of the port or an error of the replay
// becomes a line of the text, so it is a mismatch like any other and the run
// goes on.
func corpusProduce(c replayNamed, kind replayKind) (out []byte) {
	var buf bytes.Buffer
	defer func() {
		if r := recover(); r != nil {
			buf.Reset()
			fmt.Fprintf(&buf, "## case %s\n!panic: %v\n", c.name, r)
			out = buf.Bytes()
		}
	}()
	if err := produceSuite([]replayNamed{c}, kind, &buf); err != nil {
		fmt.Fprintf(&buf, "!replay error=%s\n", err.Error())
	}
	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// The suite driver and the test functions.

// openSections opens the golden file of a kind of a suite: the first
// corpusShortWindow bytes with -short, else the whole file.
func (run *corpusRun) openSections(t testing.TB, suite string, kind replayKind) *corpusSections {
	name := corpusGoldenName(suite, kind)
	if run.cfg.short && !run.cfg.exhaustive() {
		return corpusSectionsWindow(t, run.area, name, corpusShortWindow)
	}
	return corpusSectionsFile(t, run.area, name)
}

// selectCases chooses the cases of a suite to run (see the file header).
func (run *corpusRun) selectCases(t testing.TB, suite string, parsed replaySuite, kinds []replayKind) []int {
	cfg := run.cfg
	if cfg.exhaustive() {
		var out []int
		for i, c := range parsed.cases {
			if cfg.wants(run.area.name, suite, c.name) {
				out = append(out, i)
			}
		}
		return out
	}
	avail := len(parsed.cases)
	if cfg.short {
		for _, kind := range kinds {
			s := run.openSections(t, suite, kind)
			n, err := s.count()
			s.Close()
			if err != nil {
				t.Fatalf("%s: %v", corpusGoldenName(suite, kind), err)
			}
			avail = min(avail, n)
		}
	}
	return corpusSelect(avail, cfg.sample, corpusFirstMarked(parsed.cases[:avail]))
}

// replaySuite runs the selected cases of a suite, a subtest per case.
func (run *corpusRun) replaySuite(t *testing.T, suite string) {
	area := run.area
	parsed := corpusReadSuite(t, area, suite)
	corpusCheckManifest(t, run.manifest, area, suite, parsed)
	var kinds []replayKind
	for _, k := range parsed.kinds.kinds() {
		if area.kinds.contains(k) {
			kinds = append(kinds, k)
		}
	}
	if len(kinds) == 0 {
		t.Skipf("%s/%s: no kind to replay", area.name, suite)
	}
	st := &corpusSuiteStats{cases: len(parsed.cases)}
	selected := run.selectCases(t, suite, parsed, kinds)
	sections := make([]*corpusSections, len(kinds))
	for i, kind := range kinds {
		sections[i] = run.openSections(t, suite, kind)
		defer sections[i].Close()
	}

	for _, idx := range selected {
		c := parsed.cases[idx]
		t.Run(c.name, func(t *testing.T) {
			st.run++
			if isUpstreamInvalid(c.ops) {
				st.skipped++
			}
			for i, kind := range kinds {
				run.replayCase(t, st, suite, c, idx, kind, sections[i])
			}
		})
	}
	t.Log(corpusSummary(area.name, suite, st, run.cfg.strict))
}

// replayCase produces one case and compares it with its recorded section.
func (run *corpusRun) replayCase(t *testing.T, st *corpusSuiteStats, suite string, c replayNamed, idx int, kind replayKind, sections *corpusSections) {
	t.Helper()
	name, want, err := sections.section(idx)
	if err != nil {
		t.Errorf("%s/%s kind=%s: section %d: %v", run.area.name, suite, kind, idx, err)
		return
	}
	if name != c.name {
		t.Errorf("%s/%s kind=%s: section %d is %q, the case file has %q", run.area.name, suite, kind, idx, name, c.name)
		return
	}
	if bytes.HasPrefix(want, []byte(corpusSectionPrefix+name+"\n!upstream-crash\n")) {
		t.Errorf("%s/%s case=%s kind=%s: recorded by the uncorrected recorder (!upstream-crash)", run.area.name, suite, c.name, kind)
		return
	}
	got := corpusProduce(c, kind)
	m := corpusMismatch{area: run.area.name, suite: suite, caseName: c.name, kind: kind}
	if !bytes.Equal(got, want) {
		st.skippedLines += corpusSkippedDiffs(got, want, corpusSkipPrefixes)
	}
	diff, equal := corpusCompare(got, want, run.cfg.skip())
	if equal {
		return
	}
	m.diff = diff
	corpusReport(t, run, st, m)
	run.writeDiff(t, m, got, want)
}

// corpusReplayArea is the body of TestCorpusReplay<Area>: a subtest per suite
// (parallel), in it a subtest per selected case (sequential).
func corpusReplayArea(t *testing.T, name string) {
	area := corpusAreaByName(t, name)
	run := newCorpusRun(t, area)
	for _, suite := range corpusSuiteNames(t, area) {
		t.Run(suite, func(t *testing.T) {
			t.Parallel()
			run.replaySuite(t, suite)
		})
	}
}

func TestCorpusReplayCsi(t *testing.T) { corpusReplayArea(t, "csi") }
