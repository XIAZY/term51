package terminal

// Self-tests of stage 2a of the corpus replayer (split runner, keyboard and
// paste corpus, static tables): the plumbing that needs no port. Same style
// as corpus_replay_self_test.go; `go test ./terminal -run CorpusReplaySelf`.

import (
	"bytes"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// The split planner.

func TestCorpusReplaySelfSplitSeed(t *testing.T) {
	// FNV-1a 64 against the standard library.
	for _, s := range []struct {
		area, suite, c string
		op             int
	}{{"csi", "cup", "filled_H_c1", 0}, {"fuzz", "resize_scrolled", "x", 12}, {"", "", "", 0}} {
		h := fnv.New64a()
		fmt.Fprintf(h, "%s/%s/%s/%d", s.area, s.suite, s.c, s.op)
		if got := splitSeed(s.area, s.suite, s.c, s.op); got != h.Sum64() {
			t.Errorf("splitSeed(%v) = %#x, want %#x", s, got, h.Sum64())
		}
	}
	if a, b := splitSeed("csi", "cup", "c", 0), splitSeed("csi", "cup", "c", 1); a == b {
		t.Errorf("the seed ignores the op index: %#x", a)
	}

	// splitmix64 from state 0: the reference outputs.
	rng := splitmix64{}
	for i, want := range []uint64{0xe220a8397b1dcdaf, 0x6e789e6aa1b965f4, 0x06c45d188009454f} {
		if got := rng.next(); got != want {
			t.Errorf("splitmix64 output %d = %#x, want %#x", i, got, want)
		}
	}
}

func TestCorpusReplaySelfSplitConfig(t *testing.T) {
	env := func(kv ...string) func(string) string {
		m := map[string]string{}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return func(k string) string { return m[k] }
	}
	cfg, err := splitConfigFrom(env(), false)
	if err != nil || cfg != (splitConfig{allUpTo: 64, head: 64, seeded: 3}) {
		t.Errorf("default: %+v %v", cfg, err)
	}
	cfg, err = splitConfigFrom(env(), true)
	if err != nil || cfg != (splitConfig{allUpTo: 1000, head: 256, tail: 256, seeded: 1024}) {
		t.Errorf("full: %+v %v", cfg, err)
	}
	cfg, _ = splitConfigFrom(env("TERM51_SPLIT_MAX", "0"), true)
	if cfg.allUpTo != math.MaxInt {
		t.Errorf("SPLIT_MAX=0: %+v", cfg)
	}
	cfg, _ = splitConfigFrom(env("TERM51_SPLIT_MAX", "300"), false)
	if cfg.allUpTo != 300 || cfg.head != 64 {
		t.Errorf("SPLIT_MAX=300 in the default run: %+v", cfg)
	}
	cfg, err = splitConfigFrom(env("TERM51_SPLIT_AT", "op:2,at:17"), false)
	if err != nil || cfg.at == nil || *cfg.at != (splitVariant{2, 17}) {
		t.Errorf("SPLIT_AT: %+v %v", cfg, err)
	}
	cfg, _ = splitConfigFrom(env("TERM51_SPLIT_AT", "bytewise"), false)
	if cfg.at == nil || *cfg.at != splitBytewise {
		t.Errorf("SPLIT_AT=bytewise: %+v", cfg)
	}
	for _, bad := range [][]string{{"TERM51_SPLIT_MAX", "x"}, {"TERM51_SPLIT_MAX", "-1"}, {"TERM51_SPLIT_AT", "op:1"},
		{"TERM51_SPLIT_AT", "op:1,at:0"}, {"TERM51_SPLIT_AT", "op:-1,at:3"}, {"TERM51_SPLIT_AT", "x"}} {
		if _, err := splitConfigFrom(env(bad...), false); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	for _, v := range []splitVariant{splitBytewise, {0, 1}, {12, 345}} {
		if got, err := parseSplitVariant(v.String()); err != nil || got != v {
			t.Errorf("parseSplitVariant(%q) = %+v %v", v, got, err)
		}
	}
}

func TestCorpusReplaySelfSplitOffsets(t *testing.T) {
	def := splitConfig{allUpTo: 64, head: 64, seeded: 3}
	full := splitConfig{allUpTo: 1000, head: 256, tail: 256, seeded: 1024}
	all := splitConfig{allUpTo: math.MaxInt}
	seq := func(from, to int) []int {
		var out []int
		for i := from; i <= to; i++ {
			out = append(out, i)
		}
		return out
	}

	// Short feeds have every offset; empty and one-byte feeds none.
	for _, cfg := range []splitConfig{def, full, all} {
		for length, want := range map[int][]int{0: nil, 1: nil, 2: {1}, 3: {1, 2}, 64: seq(1, 63), 65: seq(1, 64)} {
			if got := splitOffsets(length, 7, cfg); !reflect.DeepEqual(got, want) {
				t.Errorf("%+v length %d: %v, want %v", cfg, length, got, want)
			}
		}
	}
	if got := splitOffsets(1000, 7, full); !reflect.DeepEqual(got, seq(1, 999)) {
		t.Errorf("full, 1000 bytes: not every offset (%d)", len(got))
	}
	if got := splitOffsets(100000, 7, all); !reflect.DeepEqual(got, seq(1, 99999)) {
		t.Errorf("SPLIT_MAX=0, 100000 bytes: not every offset (%d)", len(got))
	}

	// A long feed: the head, the seeded offsets above it, the tail.
	got := splitOffsets(5000, 11, def)
	if len(got) != 64+3 || !reflect.DeepEqual(got[:64], seq(1, 64)) || got[64] <= 64 || got[66] > 4999 {
		t.Errorf("default, 5000 bytes: %v", got)
	}
	got = splitOffsets(5000, 11, full)
	if len(got) != 256+256+1024 || !reflect.DeepEqual(got[:256], seq(1, 256)) ||
		!reflect.DeepEqual(got[len(got)-256:], seq(4744, 4999)) {
		t.Errorf("full, 5000 bytes: %d offsets, head %v, tail %v", len(got), got[:3], got[len(got)-3:])
	}
	for _, length := range []int{1001, 1500, 1537, 1538, 1600, 5000, 70000} {
		for _, cfg := range []splitConfig{def, full} {
			got := splitOffsets(length, uint64(length), cfg)
			if !sort.IntsAreSorted(got) || got[0] != 1 && cfg.head > 0 || got[len(got)-1] > length-1 || got[0] < 1 {
				t.Errorf("%+v length %d: %v...", cfg, length, got[:min(5, len(got))])
			}
			for i := 1; i < len(got); i++ {
				if got[i] == got[i-1] {
					t.Errorf("%+v length %d: offset %d twice", cfg, length, got[i])
				}
			}
			if !reflect.DeepEqual(got, splitOffsets(length, uint64(length), cfg)) {
				t.Errorf("%+v length %d: not deterministic", cfg, length)
			}
		}
	}
	// The seed changes the seeded offsets, not the edges.
	a, b := splitOffsets(5000, 1, full), splitOffsets(5000, 2, full)
	if reflect.DeepEqual(a, b) || !reflect.DeepEqual(a[:256], b[:256]) {
		t.Errorf("seeds 1 and 2: the same plan, or other edges")
	}
}

func TestCorpusReplaySelfSplitVariants(t *testing.T) {
	c := replayNamed{name: "c", ops: []replayOp{
		{Tag: replayOpInit, Init: replayOpInitArgs{10, 5, 0}},
		{Tag: replayOpFeed, Feed: []uint8{1, 2, 3}},
		{Tag: replayOpDump, Dump: "mid"},
		{Tag: replayOpFeed, Feed: []uint8{1}},
		{Tag: replayOpFeed, Feed: []uint8{1, 2}},
		{Tag: replayOpFeed},
	}}
	def := splitConfig{allUpTo: 64, head: 64, seeded: 3}
	// The bytewise run first, then per feed op (counting feed ops only, as
	// Zig's feed_idx) its offsets; ops of length 0 and 1 have none.
	want := []splitVariant{splitBytewise, {0, 1}, {0, 2}, {2, 1}}
	if got := splitVariants("a", "s", c, def); !reflect.DeepEqual(got, want) {
		t.Errorf("variants %v, want %v", got, want)
	}
	at := splitVariant{2, 1}
	if got := splitVariants("a", "s", c, splitConfig{at: &at}); !reflect.DeepEqual(got, []splitVariant{at}) {
		t.Errorf("SPLIT_AT: %v", got)
	}
	if got := splitVariants("a", "s", replayNamed{name: "e"}, def); !reflect.DeepEqual(got, []splitVariant{splitBytewise}) {
		t.Errorf("a case without feeds: %v", got)
	}

	if m := splitBytewise.mode(); m.Tag != feedModeBytewise {
		t.Errorf("bytewise mode %+v", m)
	}
	if m := (splitVariant{3, 9}).mode(); m != (feedMode{Tag: feedModeSplit, Op: 3, At: 9}) {
		t.Errorf("split mode %+v", m)
	}

	// The label of a report names the op, the offset and the seed of the op.
	label := splitLabel("csi", "cup", "c", splitVariant{2, 17})
	if want := fmt.Sprintf("op:2,at:17,seed=%#x", splitSeed("csi", "cup", "c", 2)); label != want {
		t.Errorf("label %q, want %q", label, want)
	}
	if label := splitLabel("csi", "cup", "c", splitBytewise); label != "bytewise" {
		t.Errorf("bytewise label %q", label)
	}
	m := corpusMismatch{area: "csi", suite: "cup", caseName: "c", kind: replayKindGolden, split: label,
		diff: corpusDiff{line: 9, got: "a", want: "b", label: "end"}}
	if got := corpusFormatMismatch(m); !strings.HasPrefix(got, "MISMATCH csi/cup case=c kind=golden split=op:2,at:17,seed=0x") ||
		!strings.Contains(got, " dump=end line 9\n") {
		t.Errorf("report %q", got)
	}
}

// The size of the plan on the real corpus, against docs/replayer-design.md
// section 7: all offsets of all feeds are 22.6 M variants, the offsets of the
// feeds of at most 1000 bytes 9.7 M, and 1,417 feeds are longer than 1000
// bytes.
func TestCorpusReplaySelfSplitSizes(t *testing.T) {
	full := splitConfig{allUpTo: 1000, head: 256, tail: 256, seeded: 1024}
	def := splitConfig{allUpTo: 64, head: 64, seeded: 3}
	var offsetsAll, offsetsShort, planFull, planDef, planCases int
	feeds, longFeeds, cases := 0, 0, 0
	for _, area := range corpusAreas {
		for _, suite := range corpusSuiteNames(t, area) {
			parsed := corpusReadSuite(t, area, suite)
			cases += len(parsed.cases)
			for _, c := range parsed.cases {
				idx := 0
				for _, op := range c.ops {
					if op.Tag != replayOpFeed {
						continue
					}
					feeds++
					n := max(len(op.Feed)-1, 0)
					offsetsAll += n
					if len(op.Feed) <= 1000 {
						offsetsShort += n
					} else {
						longFeeds++
					}
					seed := splitSeed(area.name, suite, c.name, idx)
					planFull += len(splitOffsets(len(op.Feed), seed, full))
					planDef += len(splitOffsets(len(op.Feed), seed, def))
					idx++
				}
				planCases++
			}
		}
	}
	t.Logf("%d cases, %d feeds (%d longer than 1000 bytes); offsets of all feeds %d, of feeds up to 1000 bytes %d; plan: full %d (+%d bytewise), default %d (+%d bytewise)",
		cases, feeds, longFeeds, offsetsAll, offsetsShort, planFull, planCases, planDef, planCases)
	if offsetsAll < 22_550_000 || offsetsAll > 22_650_000 {
		t.Errorf("all offsets %d, the design says 22.6 M", offsetsAll)
	}
	if offsetsShort < 9_650_000 || offsetsShort > 9_750_000 {
		t.Errorf("offsets of feeds up to 1000 bytes %d, the design says 9.7 M", offsetsShort)
	}
	if longFeeds != 1417 {
		t.Errorf("%d feeds longer than 1000 bytes, the design says 1,417", longFeeds)
	}
	if planFull < offsetsShort || planFull > offsetsShort+longFeeds*(256+256+1024) {
		t.Errorf("full plan %d outside [%d, %d]", planFull, offsetsShort, offsetsShort+longFeeds*(256+256+1024))
	}
	if planDef >= planFull {
		t.Errorf("default plan %d is not smaller than the full plan %d", planDef, planFull)
	}
}

func TestCorpusReplaySelfSplitKnown(t *testing.T) {
	text := "# comment\n\n" +
		"csi/cup c1 golden bytewise\n" +
		"csi/cup c2 actions op:3,at:17\n" +
		"  csi/cup c3 golden op:1,at:*  \n" +
		"osc/all c4 actions_noalloc *\n"
	k, err := parseSplitKnown(text)
	if err != nil || len(k) != 4 {
		t.Fatalf("%v %v", k, err)
	}
	for _, c := range []struct {
		area, suite, name string
		kind              replayKind
		v                 splitVariant
		want              bool
	}{
		{"csi", "cup", "c1", replayKindGolden, splitBytewise, true},
		{"csi", "cup", "c1", replayKindGolden, splitVariant{0, 1}, false},
		{"csi", "cup", "c1", replayKindActions, splitBytewise, false},
		{"csi", "cup", "c2", replayKindActions, splitVariant{3, 17}, true},
		{"csi", "cup", "c2", replayKindActions, splitVariant{3, 18}, false},
		{"csi", "cup", "c3", replayKindGolden, splitVariant{1, 99}, true},
		{"csi", "cup", "c3", replayKindGolden, splitVariant{2, 99}, false},
		{"csi", "cup", "c3", replayKindGolden, splitBytewise, false},
		{"osc", "all", "c4", replayKindActionsNoalloc, splitBytewise, true},
		{"osc", "all", "c4", replayKindActionsNoalloc, splitVariant{7, 2}, true},
		{"osc", "all", "c4", replayKindGolden, splitVariant{7, 2}, false},
		{"osc", "all", "c5", replayKindActionsNoalloc, splitBytewise, false},
		{"csi", "cub", "c1", replayKindGolden, splitBytewise, false},
	} {
		if got := k.accepts(c.area, c.suite, c.name, c.kind, c.v); got != c.want {
			t.Errorf("accepts(%v) = %v, want %v", c, got, c.want)
		}
	}
	for _, bad := range []string{"a b c\n", "csi/cup c nokind bytewise\n", "csi/cup c golden op:1\n", "csi/cup c golden op:x,at:*\n",
		"csi/cup c golden op:1,at:0\n", "csi/cup c golden bytewise extra\n"} {
		if _, err := parseSplitKnown(bad); err == nil {
			t.Errorf("parseSplitKnown(%q) succeeded", bad)
		}
	}
	// The real file parses and, today, accepts nothing.
	if known := splitKnownLoad(t); len(known) != 0 {
		t.Errorf("%s has %d entries; update this test with the reason", splitKnownFile, len(known))
	}
}

// ---------------------------------------------------------------------------
// The keyboard and paste corpus.

func TestCorpusReplaySelfInputRunErrors(t *testing.T) {
	// Errors that precede every name lookup: no port needed. The echo of the
	// request is written before the error, as in the Zig.
	for _, c := range []struct {
		text string
		out  string
		err  error
	}{
		{"# comment\n\n  \n", "", nil},
		{"frob 1\n", "frob 1\n", errReplayBadCase},
		{"k\n", "k\n", errReplayBadCase},
		{"k a\n", "k a\n", errReplayBadCase},
		{"k a press 0 0 0 0 - 0 0\n", "k a press 0 0 0 0 - 0 0\n", errReplayBadCase},
		{"p\n", "p\n", errReplayBadCase},
		{"p 1\n", "p 1\n", errReplayBadCase},
		{"p 1 abc\n", "p 1 abc\n", errInputInvalidLength},
		{"p 1 zz\n", "p 1 zz\n", errInputInvalidCharacter},
	} {
		got, err := inputRunText(c.text)
		if c.err == nil && err != nil || c.err != nil && !errors.Is(err, c.err) || got != c.out {
			t.Errorf("inputRun(%q) = %q, %v; want %q, %v", c.text, got, err, c.out, c.err)
		}
	}
	// Numbers that do not fit their Zig type are errors (u16, u1, u21, u8, u5).
	for _, bad := range []string{
		"k a press 65536 0 0 0 - 0 0 false", "k a press 0 65536 0 0 - 0 0 false", "k a press 0 0 2 0 - 0 0 false",
		"k a press 0 0 0 2097152 - 0 0 false", "k a press 0 0 0 0 - 256 0 false", "k a press 0 0 0 0 - 0 32 false",
		"k a press x 0 0 0 - 0 0 false", "k a press 0 0 0 0 abc 0 0 false", "k a press 0 0 0 0 zz 0 0 false",
		"p 2 -", "p x -",
	} {
		if _, err := inputRunText(bad + "\n"); err == nil {
			t.Errorf("inputRun(%q) succeeded", bad)
		}
	}
}

// inputRunText is inputRun into a string.
func inputRunText(text string) (string, error) {
	var buf bytes.Buffer
	err := inputRun(text, &buf)
	return buf.String(), err
}

func TestCorpusReplaySelfInputHex(t *testing.T) {
	if got, err := inputHexArg("-"); err != nil || got == nil || len(got) != 0 {
		t.Errorf("hexArg(-) = %v %v", got, err)
	}
	if got, err := inputHexArg("00ff1B7f"); err != nil || !reflect.DeepEqual(got, []uint8{0, 255, 0x1b, 0x7f}) {
		t.Errorf("hexArg = %v %v", got, err)
	}
	if got, err := inputHexArg(""); err != nil || len(got) != 0 {
		t.Errorf("hexArg(empty) = %v %v", got, err)
	}
	if _, err := inputHexArg("abc"); !errors.Is(err, errInputInvalidLength) {
		t.Errorf("odd length: %v", err)
	}
	if _, err := inputHexArg("0g"); !errors.Is(err, errInputInvalidCharacter) {
		t.Errorf("bad digit: %v", err)
	}
	var buf bytes.Buffer
	inputWriteHex(&buf, nil)
	inputWriteHex(&buf, []uint8{})
	if buf.String() != "--" {
		t.Errorf("writeHex of nothing: %q", buf.String())
	}
	buf.Reset()
	inputWriteHex(&buf, []uint8{0, 1, 0xab, 0xff})
	if buf.String() != "0001abff" {
		t.Errorf("writeHex: %q", buf.String())
	}
}

func TestCorpusReplaySelfInputKeyParse(t *testing.T) {
	line := "k key_a press 5 4 1 97 61 29 31 left"
	it := &replayTokens{rest: line[2:]}
	got, err := inputKeyParse(it)
	want := inputKeyArgs{keyName: "key_a", actionName: "press", mods: 5, consumed: 4, composing: 1, ucp: 97,
		utf8: []uint8{0x61}, opts: 29, kitty: 31, oaaName: "left"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("inputKeyParse = %+v %v, want %+v", got, err, want)
	}
	// Fields past the last are not read.
	it = &replayTokens{rest: line[2:] + " extra"}
	if got, err = inputKeyParse(it); err != nil || got.oaaName != "left" {
		t.Errorf("extra token: %+v %v", got, err)
	}
}

func TestCorpusReplaySelfInputFormat(t *testing.T) {
	r := inputRequest{line: 7, req: "p 1 61", want: "=> 1b|61|1c safe=1"}
	got := inputFormatMismatch("input/linux", "paste", 3, r, "p 1 61\n=> 1b|62|1c safe=1\n")
	want := "MISMATCH input/linux/paste req=3 line 7\n  req:  p 1 61\n  got:  => 1b|62|1c safe=1\n  want: => 1b|61|1c safe=1"
	if got != want {
		t.Errorf("report\n got  %q\n want %q", got, want)
	}
	if out := inputProduce("frob"); out != "frob\n!replay error=BadCase\n" {
		t.Errorf("inputProduce(frob) = %q", out)
	}
	// A request that reaches the port starts with its echo whatever happens.
	if out := inputProduce("p 1 61"); !strings.HasPrefix(out, "p 1 61\n") {
		t.Errorf("inputProduce(paste) = %q", out)
	}

	if _, err := inputPairs([]string{"a"}); err == nil {
		t.Errorf("odd line count accepted")
	}
	if _, err := inputPairs([]string{"a", "b"}); err == nil {
		t.Errorf("a pair without => accepted")
	}
	if _, err := inputPairs([]string{"=> a", "=> b"}); err == nil {
		t.Errorf("a request that looks like an output accepted")
	}
	pairs, err := inputPairs([]string{"p 0 -", "=> -|-|- safe=1", "p 1 -", "=> a|b|c safe=1"})
	if err != nil || len(pairs) != 2 || pairs[1].line != 3 || pairs[1].want != "=> a|b|c safe=1" {
		t.Errorf("inputPairs = %+v %v", pairs, err)
	}

	cfg, _ := corpusConfigFrom(func(string) string { return "" }, false)
	if got := inputSelect(cfg, false, "input/linux", "paste", 100000); len(got) != corpusInputSample || got[0] != 0 || got[499] != 99999 {
		t.Errorf("default selection: %d requests", len(got))
	}
	if got := inputSelect(cfg, true, "input/linux", "paste", 100000); len(got) != 20 {
		t.Errorf("TERM51_SAMPLE selection: %d", len(got))
	}
	if got := inputSelect(cfg, false, "input/linux", "paste", 40); len(got) != 40 {
		t.Errorf("a small file: %d", len(got))
	}
	cfg.full = true
	if got := inputSelect(cfg, false, "input/linux", "paste", 1000); len(got) != 1000 {
		t.Errorf("full selection: %d", len(got))
	}
	cfg, _ = corpusConfigFrom(func(k string) string {
		if k == "TERM51_ONLY" {
			return "^input/ios/paste/(3|40)$"
		}
		return ""
	}, false)
	if got := inputSelect(cfg, false, "input/ios", "paste", 1000); !reflect.DeepEqual(got, []int{3, 40}) {
		t.Errorf("TERM51_ONLY selection: %v", got)
	}
}

// Every file of the three input recordings: its pairs, the request fields
// against their Zig widths and name sets, the request echo as the canonical
// text of the parsed fields, the hex of the outputs, the totals and the
// differences between the three modes (docs/replayer-design.md appendix B and
// docs/macos.md).
func TestCorpusReplaySelfInputFiles(t *testing.T) {
	modes := []inputMode{inputModeLinux, inputModeMacos, inputModeIos}
	files := inputFiles(t, modes[0])
	if len(files) != 49 {
		t.Fatalf("%d files, want 49", len(files))
	}
	all := files
	if testing.Short() {
		// The 32-bit ARM run under qemu: five files, no totals.
		files = []string{"keys_kitty_01", "keys_legacy", "keys_platform_legacy", "keys_text", "paste"}
	}
	for _, m := range modes[1:] {
		if got := inputFiles(t, m); !reflect.DeepEqual(got, all) {
			t.Fatalf("%s has other files than linux: %v", m.name, got)
		}
	}
	identical := map[string]bool{"paste": true}
	for i := 1; i <= 15; i++ {
		identical[fmt.Sprintf("keys_kitty_%02d", i)] = true
	}
	for i := 1; i <= 9; i++ {
		identical[fmt.Sprintf("keys_platform_kitty_%02d", i)] = true
	}

	var total int
	var diff [3]int // linux/macos, linux/ios, macos/ios
	keys, actions, oaas := map[string]bool{}, map[string]bool{}, map[string]bool{}
	var kinds [2]int
	for _, file := range files {
		var reqs [3][]inputRequest
		for i, m := range modes {
			reqs[i] = inputReadFile(t, m, file)
		}
		if len(reqs[1]) != len(reqs[0]) || len(reqs[2]) != len(reqs[0]) {
			t.Fatalf("%s: %d, %d and %d requests", file, len(reqs[0]), len(reqs[1]), len(reqs[2]))
		}
		total += len(reqs[0])
		differs := 0
		for i, r := range reqs[0] {
			if reqs[1][i].req != r.req || reqs[2][i].req != r.req {
				t.Fatalf("%s request %d differs between the modes: %q", file, i, r.req)
			}
			for j, pair := range [3][2]int{{0, 1}, {0, 2}, {1, 2}} {
				if reqs[pair[0]][i].want != reqs[pair[1]][i].want {
					diff[j]++
					differs++
				}
			}
			inputSelfCheckRequest(t, file, r, keys, actions, oaas, &kinds)
		}
		if identical[file] && differs != 0 {
			t.Errorf("%s: %d outputs differ between the modes, the design says none", file, differs)
		}
	}
	if !testing.Short() && total != 1583876 {
		t.Errorf("%d requests, want 1,583,876", total)
	}
	if !testing.Short() && diff != [3]int{94650, 27750, 71576} {
		t.Errorf("outputs that differ (linux/macos, linux/ios, macos/ios) %v, want [94650 27750 71576]", diff)
	}
	if !reflect.DeepEqual(sortedKeys(actions), []string{"press", "release", "repeat"}) ||
		!reflect.DeepEqual(sortedKeys(oaas), []string{"false", "left", "right", "true"}) {
		t.Errorf("actions %v, option-as-alt %v", sortedKeys(actions), sortedKeys(oaas))
	}
	if len(keys) == 0 || len(keys) > 176 {
		t.Errorf("%d distinct key names", len(keys))
	}
	t.Logf("%d requests (%d key, %d paste), %d distinct key names", total, kinds[0], kinds[1], len(keys))
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// inputSelfCheckRequest parses a request with the recorder's own field
// parsers, and checks the recorded output against its form.
func inputSelfCheckRequest(t *testing.T, file string, r inputRequest, keys, actions, oaas map[string]bool, kinds *[2]int) {
	t.Helper()
	it := &replayTokens{rest: r.req}
	kind, _ := it.next()
	switch kind {
	case "k":
		kinds[0]++
		a, err := inputKeyParse(it)
		if err != nil {
			t.Fatalf("%s line %d: %q: %v", file, r.line, r.req, err)
		}
		if _, extra := it.next(); extra {
			t.Fatalf("%s line %d: %q: extra fields", file, r.line, r.req)
		}
		keys[a.keyName], actions[a.actionName], oaas[a.oaaName] = true, true, true
		var hexText strings.Builder
		if len(a.utf8) == 0 {
			hexText.WriteString("-")
		}
		for _, c := range a.utf8 {
			fmt.Fprintf(&hexText, "%02x", c)
		}
		again := fmt.Sprintf("k %s %s %d %d %d %d %s %d %d %s", a.keyName, a.actionName, a.mods, a.consumed,
			a.composing, a.ucp, hexText.String(), a.opts, a.kitty, a.oaaName)
		if again != r.req {
			t.Fatalf("%s line %d: %q is not the canonical form %q", file, r.line, r.req, again)
		}
		out := r.want[len("=> "):]
		if strings.HasPrefix(out, "!") {
			if out != "!WriteFailed" {
				t.Fatalf("%s line %d: %q", file, r.line, r.want)
			}
			return
		}
		inputSelfCheckHex(t, file, r, out)
	case "p":
		kinds[1]++
		if file != "paste" {
			t.Fatalf("%s line %d: a paste request: %q", file, r.line, r.req)
		}
		f := strings.Fields(r.req)
		if len(f) != 3 || f[1] != "0" && f[1] != "1" {
			t.Fatalf("%s line %d: %q", file, r.line, r.req)
		}
		if _, err := inputHexArg(f[2]); err != nil {
			t.Fatalf("%s line %d: %q: %v", file, r.line, r.req, err)
		}
		segs, safe, ok := strings.Cut(r.want[len("=> "):], " safe=")
		if !ok || safe != "0" && safe != "1" || strings.Count(segs, "|") != 2 {
			t.Fatalf("%s line %d: %q", file, r.line, r.want)
		}
		for _, seg := range strings.Split(segs, "|") {
			inputSelfCheckHex(t, file, r, seg)
		}
	default:
		t.Fatalf("%s line %d: %q", file, r.line, r.req)
	}
}

// inputSelfCheckHex checks that a recorded hex field is what inputWriteHex
// writes for its bytes.
func inputSelfCheckHex(t *testing.T, file string, r inputRequest, field string) {
	t.Helper()
	b, err := inputHexArg(field)
	var buf bytes.Buffer
	inputWriteHex(&buf, b)
	if err != nil || buf.String() != field {
		t.Fatalf("%s line %d: %q: hex field %q does not round-trip (%v)", file, r.line, r.want, field, err)
	}
}

// ---------------------------------------------------------------------------
// The static tables.

func TestCorpusReplaySelfTablesFiles(t *testing.T) {
	read := func(name string) []string {
		return corpusLines(t, filepath.Join("golden", "tables", name+".txt"))
	}

	parse := read("parse_table")
	if len(parse) != 256*parserStateCount {
		t.Fatalf("parse_table has %d lines, want %d", len(parse), 256*parserStateCount)
	}
	for i, line := range parse {
		s, c := i/256, i%256
		prefix := fmt.Sprintf("%s %02x -> ", ParserState(s), c)
		if !strings.HasPrefix(line, prefix) || len(strings.Fields(line)) != 5 {
			t.Fatalf("parse_table line %d: %q, want prefix %q and 5 fields", i+1, line, prefix)
		}
	}

	pal := read("default_palette")
	if len(pal) != 256 {
		t.Fatalf("default_palette has %d lines", len(pal))
	}
	for i, line := range pal {
		var idx int
		var rgb string
		if n, err := fmt.Sscanf(line, "%d %s", &idx, &rgb); n != 2 || err != nil || idx != i || len(rgb) != 6 ||
			fmt.Sprintf("%d %s", i, strings.ToLower(rgb)) != line {
			t.Fatalf("default_palette line %d: %q", i+1, line)
		}
	}

	modes := read("modes")
	if len(modes) != 40 {
		t.Fatalf("modes has %d lines, want 40", len(modes))
	}
	seen := map[string]bool{}
	for i, line := range modes {
		f := strings.Fields(line)
		if len(f) != 4 || f[2] != "ansi" && f[2] != "dec" || f[3] != "default=0" && f[3] != "default=1" || seen[f[2]+f[1]] {
			t.Fatalf("modes line %d: %q", i+1, line)
		}
		seen[f[2]+f[1]] = true
	}

	// The writers: a panic of the port is a line of the text.
	if out := tablesProduce(func(*bytes.Buffer) { panic("boom") }); string(out) != "!panic: boom\n" {
		t.Errorf("tablesProduce on a panic: %q", out)
	}
	if _, ok := corpusCompare([]byte("a\nb\n"), []byte("a\nc\n"), nil); ok {
		t.Errorf("tables compare: a difference was not seen")
	}
}
