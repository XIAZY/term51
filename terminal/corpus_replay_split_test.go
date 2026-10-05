package terminal

// The split runs of the corpus replayer: the Go side of the recorder's
// `vtgold split` (harness/src/main.zig splitCheck). A case must give the
// same output however the stream is cut, so every case is replayed
//
//   - bytewise: one NextSlice per byte, and
//   - for every feed op and an offset `at` of it: that op alone cut in two
//     (bytes[:at], bytes[at:]), the other ops whole,
//
// and every variant is compared with the recorded golden section of the case
// (the Zig compares with its own whole run; the golden section is that run,
// recorded). The area test is TestCorpusReplaySplit<Area>; it replays the
// kinds of the area's row of corpusAreas (golden now; actions and
// actions_noalloc with stage 2b).
//
// The WHOLE run of a case is replayed and compared with the golden section
// first. If it already differs, that is reported once, as a whole-run
// mismatch (`split=whole`, the same difference TestCorpusReplay<Area> reports)
// and the variants of the case are skipped: so a report with a split in it is
// a difference that comes from the cut. The skipped variants are counted in the
// suite line.
//
// # Which offsets
//
// All offsets of all feeds are 22.6 million variants and 1.0e12 byte-feeds
// (docs/replayer-design.md section 7), 99.9 percent of it in the 1,417 feeds
// longer than 1,000 bytes. Feeds with L bytes have the offsets 1..L-1. Per
// feed op:
//
//	default             the offsets 1..64 of every feed and 3 seeded offsets
//	                    above 64 (a feed with at most 67 offsets: all of them)
//	TERM51_FULL=1       (or TERM51_ONLY) a feed of at most TERM51_SPLIT_MAX
//	                    bytes (default 1000): every offset; a longer one: the
//	                    first 256 offsets, the last 256 and 1,024 seeded ones
//	                    in between
//	TERM51_SPLIT_MAX=n  the length up to which every offset is cut, in the
//	                    default run too; 0 means every offset of every feed
//	TERM51_SPLIT_AT=v   only that variant, `bytewise` or `op:<i>,at:<n>` (a
//	                    reproduction of a report)
//
// The seeded offsets are drawn from a splitmix64 stream seeded with the FNV-1a
// hash of `area/suite/case/opindex` (splitSeed), so they do not depend on the
// kind, the run mode or the order of the run. Every split report names the
// seed, the op and the offset:
//
//	MISMATCH csi/cup case=c kind=golden split=op:2,at:17,seed=0x... dump=end line 9
//
// Rerun one variant with TERM51_ONLY='^csi/cup/c$' TERM51_SPLIT_AT=op:2,at:17.
//
// # Known exceptions
//
// corpus_split_known.txt (next to this file) lists variants accepted as
// upstream behaviour (a split run of the pinned upstream that differs from its
// whole run, arbitrated with `vtgold split cases/<area>`). One line each,
// fields separated by spaces, `#` starts a comment:
//
//	<area>/<suite> <case> <kind> <split>
//
// where split is `bytewise`, `op:<i>,at:<n>`, `op:<i>,at:*` (every offset of
// op i) or `*` (every variant of the case and kind). A listed variant that
// differs is not a failure; the suite line counts them as known=N. The file
// is empty today.
//
// The suite line:
//
//	SPLIT csi/cup cases=1330 run=20 variants=1234 known=0 skipped=57 whole=1 mismatch=3
//
// (cases in the suite, cases run, variants replayed, variants accepted by the
// known file, variants not replayed because the whole run of their case
// differs, cases whose whole run differs, differences: the whole runs that
// differ and the variants that differ). It is always logged. A test that runs
// no case fails (corpusFailIfEmpty).

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// The plan: which variants of a case.

// splitVariant is one variant of a case: the bytewise run (op < 0), or the
// feed op with index op (counting feed ops only, Zig `feed_idx`) cut at
// offset at.
type splitVariant struct {
	op int
	at int
}

var splitBytewise = splitVariant{op: -1}

func (v splitVariant) String() string {
	if v.op < 0 {
		return "bytewise"
	}
	return fmt.Sprintf("op:%d,at:%d", v.op, v.at)
}

// parseSplitVariant is the inverse of String.
func parseSplitVariant(s string) (splitVariant, error) {
	if s == "bytewise" {
		return splitBytewise, nil
	}
	var v splitVariant
	if n, err := fmt.Sscanf(s, "op:%d,at:%d", &v.op, &v.at); err != nil || n != 2 || v.op < 0 || v.at < 1 {
		return splitVariant{}, fmt.Errorf("bad split variant %q: want bytewise or op:<i>,at:<n>", s)
	}
	return v, nil
}

// splitConfig is the choice of offsets (see the file header).
type splitConfig struct {
	// A feed of at most allUpTo bytes has every offset cut; math.MaxInt for
	// TERM51_SPLIT_MAX=0.
	allUpTo int
	// Of a longer feed: offsets 1..head, the last tail offsets, and `seeded`
	// seeded ones in between.
	head, tail, seeded int
	// Only this variant (TERM51_SPLIT_AT).
	at *splitVariant
}

const (
	splitDefaultAllUpTo = 64
	splitDefaultSeeded  = 3
	splitFullAllUpTo    = 1000
	splitFullEdge       = 256
	splitFullSeeded     = 1024
)

// splitConfigFrom reads the configuration with getenv. exhaustive is
// corpusConfig.exhaustive: TERM51_FULL or a TERM51_ONLY filter.
func splitConfigFrom(getenv func(string) string, exhaustive bool) (splitConfig, error) {
	cfg := splitConfig{allUpTo: splitDefaultAllUpTo, head: splitDefaultAllUpTo, seeded: splitDefaultSeeded}
	if exhaustive {
		cfg = splitConfig{allUpTo: splitFullAllUpTo, head: splitFullEdge, tail: splitFullEdge, seeded: splitFullSeeded}
	}
	if s := getenv("TERM51_SPLIT_MAX"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return cfg, fmt.Errorf("TERM51_SPLIT_MAX=%q: not a number", s)
		}
		cfg.allUpTo = n
		if n == 0 {
			cfg.allUpTo = math.MaxInt
		}
	}
	if s := getenv("TERM51_SPLIT_AT"); s != "" {
		v, err := parseSplitVariant(s)
		if err != nil {
			return cfg, fmt.Errorf("TERM51_SPLIT_AT: %v", err)
		}
		cfg.at = &v
	}
	return cfg, nil
}

// splitSeed is the seed of the offsets of one feed op: the FNV-1a (64 bit)
// hash of `area/suite/case/opindex`.
func splitSeed(area, suite, caseName string, op int) uint64 {
	h := uint64(14695981039346656037)
	for _, c := range []byte(fmt.Sprintf("%s/%s/%s/%d", area, suite, caseName, op)) {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return h
}

// splitmix64 is the generator of Steele, Lea and Flood (the reference
// implementation by Vigna).
type splitmix64 struct{ state uint64 }

func (s *splitmix64) next() uint64 {
	s.state += 0x9e3779b97f4a7c15
	z := s.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// splitOffsets returns the offsets to cut of a feed of length bytes, in
// increasing order: a subset of 1..length-1.
func splitOffsets(length int, seed uint64, cfg splitConfig) []int {
	n := length - 1 // the offsets 1..n
	if n < 1 {
		return nil
	}
	if length <= cfg.allUpTo || n <= cfg.head+cfg.tail+cfg.seeded {
		out := make([]int, n)
		for i := range out {
			out[i] = i + 1
		}
		return out
	}
	var out []int
	for at := 1; at <= cfg.head; at++ {
		out = append(out, at)
	}
	// Seeded offsets: distinct, in head+1..n-tail.
	lo, span := cfg.head+1, n-cfg.tail-cfg.head
	rng := splitmix64{state: seed}
	picked := map[int]bool{}
	for len(picked) < min(cfg.seeded, span) {
		picked[lo+int(rng.next()%uint64(span))] = true
	}
	for at := range picked {
		out = append(out, at)
	}
	for at := n - cfg.tail + 1; at <= n; at++ {
		out = append(out, at)
	}
	sort.Ints(out)
	return out
}

// splitVariants lists the variants of a case, in the order of splitCheck
// (main.zig:360): the bytewise run first, then per feed op its offsets.
func splitVariants(area, suite string, c replayNamed, cfg splitConfig) []splitVariant {
	if cfg.at != nil {
		return []splitVariant{*cfg.at}
	}
	out := []splitVariant{splitBytewise}
	feedIdx := 0
	for _, op := range c.ops {
		if op.Tag != replayOpFeed {
			continue
		}
		seed := splitSeed(area, suite, c.name, feedIdx)
		for _, at := range splitOffsets(len(op.Feed), seed, cfg) {
			out = append(out, splitVariant{op: feedIdx, at: at})
		}
		feedIdx++
	}
	return out
}

// splitMode is the feedMode of a variant.
func (v splitVariant) mode() feedMode {
	if v.op < 0 {
		return feedMode{Tag: feedModeBytewise}
	}
	return feedMode{Tag: feedModeSplit, Op: uint(v.op), At: uint(v.at)}
}

// splitLabel is the `split=` of a report: the variant and the seed of its op.
func splitLabel(area, suite, caseName string, v splitVariant) string {
	if v.op < 0 {
		return "bytewise"
	}
	return fmt.Sprintf("%s,seed=%#x", v, splitSeed(area, suite, caseName, v.op))
}

// ---------------------------------------------------------------------------
// Known exceptions: corpus_split_known.txt.

// splitKnownFile is the file of accepted variants, in the package directory.
const splitKnownFile = "corpus_split_known.txt"

// splitKnownEntry is one line of the file; at is -1 for `at:*` and op is -2
// for `*`.
type splitKnownEntry struct {
	suite, caseName string // `area/suite` and the case
	kind            replayKind
	v               splitVariant
}

// splitKnown is the set of accepted variants.
type splitKnown []splitKnownEntry

// parseSplitKnown parses the text of the file (see the file header).
func parseSplitKnown(text string) (splitKnown, error) {
	var out splitKnown
	for n, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 4 {
			return nil, fmt.Errorf("line %d: want `<area>/<suite> <case> <kind> <split>`: %q", n+1, line)
		}
		kind, ok := replayKindFromString(f[2])
		if !ok {
			return nil, fmt.Errorf("line %d: unknown kind %q", n+1, f[2])
		}
		e := splitKnownEntry{suite: f[0], caseName: f[1], kind: kind}
		switch {
		case f[3] == "*":
			e.v = splitVariant{op: -2}
		case strings.HasSuffix(f[3], ",at:*"):
			var op int
			if _, err := fmt.Sscanf(strings.TrimSuffix(f[3], ",at:*"), "op:%d", &op); err != nil || op < 0 {
				return nil, fmt.Errorf("line %d: bad split %q", n+1, f[3])
			}
			e.v = splitVariant{op: op, at: -1}
		default:
			v, err := parseSplitVariant(f[3])
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", n+1, err)
			}
			e.v = v
		}
		out = append(out, e)
	}
	return out, nil
}

// splitKnownLoad reads corpus_split_known.txt.
func splitKnownLoad(t testing.TB) splitKnown {
	t.Helper()
	text, err := os.ReadFile(splitKnownFile)
	if err != nil {
		t.Fatal(err)
	}
	known, err := parseSplitKnown(string(text))
	if err != nil {
		t.Fatalf("%s: %v", splitKnownFile, err)
	}
	return known
}

// accepts is the one place where a known exception is honoured: the variant
// v of the case of `area/suite` and kind is accepted upstream behaviour.
func (k splitKnown) accepts(area, suite, caseName string, kind replayKind, v splitVariant) bool {
	for _, e := range k {
		if e.suite != area+"/"+suite || e.caseName != caseName || e.kind != kind {
			continue
		}
		switch {
		case e.v.op == -2, e.v == v, e.v.op == v.op && e.v.at == -1:
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// The runner.

// splitProduceFn is the producer of splitCheck; the self-tests replace it.
var splitProduceFn = splitProduce

// splitProduce is produceSuite for one case and one variant, as the text of
// one section; a panic of the port becomes a line of the text (as in
// corpusProduce).
func splitProduce(c replayNamed, kind replayKind, mode feedMode) (out []byte) {
	var buf bytes.Buffer
	defer func() {
		if r := recover(); r != nil {
			buf.Reset()
			fmt.Fprintf(&buf, "## case %s\n!panic: %v\n", c.name, r)
			out = buf.Bytes()
		}
	}()
	fmt.Fprintf(&buf, "## case %s\n", c.name)
	if err := produce(c.ops, kind, mode, &buf); err != nil {
		fmt.Fprintf(&buf, "!replay error=%s\n", err.Error())
	}
	return buf.Bytes()
}

// splitReport is corpusReport for a variant: the same counting and caps, but
// the fail list names the case once (first is true for its first differing
// variant), not every variant.
func splitReport(t corpusReporter, run *corpusRun, st *splitStats, m corpusMismatch, first bool) {
	t.Helper()
	st.mismatch[m.kind]++
	if st.reported < run.cfg.suiteMax() && run.reported.Add(1) <= int64(run.cfg.testMax()) {
		st.reported++
		t.Error(corpusFormatMismatch(m))
	} else {
		t.Fail()
	}
	if first {
		run.appendFailList(m)
	}
}

// splitStats are the counters of one suite of a split run.
type splitStats struct {
	corpusSuiteStats
	variants int // replayed (the whole runs are not counted)
	known    int
	// Cases whose whole run differs from the golden section, and the variants
	// of those cases that were not replayed for it.
	whole   int
	skipped int
}

func splitSummary(area, suite string, st *splitStats) string {
	var kinds []string
	for k := replayKind(0); k < replayKindCount; k++ {
		if st.mismatch[k] > 0 {
			kinds = append(kinds, fmt.Sprintf("%s=%d", k, st.mismatch[k]))
		}
	}
	line := fmt.Sprintf("SPLIT %s/%s cases=%d run=%d variants=%d known=%d skipped=%d whole=%d mismatch=%d",
		area, suite, st.cases, st.run, st.variants, st.known, st.skipped, st.whole, st.mismatches())
	if len(kinds) > 0 {
		line += " (" + strings.Join(kinds, " ") + ")"
	}
	return line
}

// splitReporter is the part of *testing.T that splitCheck uses.
type splitReporter interface {
	corpusReporter
	Logf(format string, args ...any)
}

// Mirrors harness/src/main.zig:349 splitCheck
//
// The replay of one case and kind: first the whole run, which is compared with
// the recorded section `want` (in the Zig the whole run is the base of the
// comparison and is not itself compared); then, if it matches, all the
// variants of the plan, each compared with `want`. It returns the number of
// differences: 1 for a whole run that differs (its variants are skipped and
// counted), else the number of variants that differ. The report of a variant
// has the variant and its seed.
func splitCheck(t splitReporter, run *corpusRun, st *splitStats, known splitKnown, cfg splitConfig,
	suite string, c replayNamed, kind replayKind, want []byte) int {
	t.Helper()
	area := run.area.name
	variants := splitVariants(area, suite, c, cfg)

	whole := splitProduceFn(c, kind, feedMode{Tag: feedModeWhole})
	if diff, equal := corpusCompare(whole, want, run.cfg.skip()); !equal {
		m := corpusMismatch{area: area, suite: suite, caseName: c.name, kind: kind, split: "whole", diff: diff}
		splitReport(t, run, st, m, true)
		run.writeDiff(t, m, whole, want)
		st.whole++
		st.skipped += len(variants)
		return 1
	}

	failed := 0
	for _, v := range variants {
		got := splitProduceFn(c, kind, v.mode())
		st.variants++
		diff, equal := corpusCompare(got, want, run.cfg.skip())
		if equal {
			continue
		}
		if known.accepts(area, suite, c.name, kind, v) {
			st.known++
			continue
		}
		m := corpusMismatch{area: area, suite: suite, caseName: c.name, kind: kind,
			split: splitLabel(area, suite, c.name, v), diff: diff}
		splitReport(t, run, st, m, failed == 0)
		if failed == 0 {
			run.writeDiff(t, m, got, want)
		}
		failed++
	}
	return failed
}

// replaySplitSuite runs the split variants of the selected cases of a suite,
// a subtest per case.
func (run *corpusRun) replaySplitSuite(t *testing.T, cfg splitConfig, known splitKnown, suite string) {
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
	st := &splitStats{corpusSuiteStats: corpusSuiteStats{cases: len(parsed.cases)}}
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
			run.ran.Add(1)
			for i, kind := range kinds {
				name, want, err := sections[i].section(idx)
				switch {
				case err != nil:
					t.Errorf("%s/%s kind=%s: section %d: %v", area.name, suite, kind, idx, err)
				case name != c.name:
					t.Errorf("%s/%s kind=%s: section %d is %q, the case file has %q", area.name, suite, kind, idx, name, c.name)
				default:
					splitCheck(t, run, st, known, cfg, suite, c, kind, want)
				}
			}
		})
	}
	run.verifySections(t, suite, kinds, sections)
	t.Log(splitSummary(area.name, suite, st))
}

// corpusSplitArea is the body of TestCorpusReplaySplit<Area>: a subtest per
// suite (parallel), in it a subtest per selected case.
func corpusSplitArea(t *testing.T, name string) {
	area := corpusAreaByName(t, name)
	run := newCorpusRun(t, area)
	t.Cleanup(func() { corpusFailIfEmpty(t, run.cfg, "split replay of area "+name, int(run.ran.Load())) })
	cfg, err := splitConfigFrom(os.Getenv, run.cfg.exhaustive())
	if err != nil {
		t.Fatal(err)
	}
	known := splitKnownLoad(t)
	for _, suite := range corpusSuiteNames(t, area) {
		t.Run(suite, func(t *testing.T) {
			t.Parallel()
			run.replaySplitSuite(t, cfg, known, suite)
		})
	}
}

func TestCorpusReplaySplitCsi(t *testing.T)    { corpusSplitArea(t, "csi") }
func TestCorpusReplaySplitDcs(t *testing.T)    { corpusSplitArea(t, "dcs") }
func TestCorpusReplaySplitEsc(t *testing.T)    { corpusSplitArea(t, "esc") }
func TestCorpusReplaySplitFuzz(t *testing.T)   { corpusSplitArea(t, "fuzz") }
func TestCorpusReplaySplitModes(t *testing.T)  { corpusSplitArea(t, "modes") }
func TestCorpusReplaySplitOsc(t *testing.T)    { corpusSplitArea(t, "osc") }
func TestCorpusReplaySplitParser(t *testing.T) { corpusSplitArea(t, "parser") }
func TestCorpusReplaySplitResize(t *testing.T) { corpusSplitArea(t, "resize") }
func TestCorpusReplaySplitScroll(t *testing.T) { corpusSplitArea(t, "scroll") }
func TestCorpusReplaySplitText(t *testing.T)   { corpusSplitArea(t, "text") }
