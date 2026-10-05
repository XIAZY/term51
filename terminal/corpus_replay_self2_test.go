package terminal

// Self-tests of stage 2a of the corpus replayer (split runner, keyboard and
// paste corpus, static tables): the plumbing that needs no port. Same style
// as corpus_replay_self_test.go; `go test ./terminal -run CorpusReplaySelf`.

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"hash/fnv"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
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

// ---------------------------------------------------------------------------
// std.fmt.parseInt.

func TestCorpusReplaySelfZigParseInt(t *testing.T) {
	type tc struct {
		s      string
		bits   int
		signed bool
		want   int64 // for an unsigned type that fits in int64
		err    error
	}
	u := func(s string, bits int, want int64, err error) tc { return tc{s, bits, false, want, err} }
	i := func(s string, bits int, want int64, err error) tc { return tc{s, bits, true, want, err} }
	ov, ic := errZigOverflow, errZigInvalidCharacter
	for _, c := range []tc{
		// The tests of parseInt in fmt.zig (base 10 ones).
		i("-10", 32, -10, nil), i("+10", 32, 10, nil), u("+10", 32, 10, nil), u("-10", 32, 0, ov),
		u(" 10", 32, 0, ic), u("10 ", 32, 0, ic), u("_10_", 32, 0, ic), u("0x_10_", 32, 0, ic),
		u("0x10_", 32, 0, ic), u("0x_10", 32, 0, ic), u("0x10", 32, 0, ic), u("0b1", 32, 0, ic), u("0o7", 32, 0, ic),
		u("255", 8, 255, nil), u("256", 8, 0, ov),
		u("-0", 8, 0, nil), u("+0", 8, 0, nil), u("-00", 8, 0, nil), u("-0_0", 8, 0, nil), u("-01", 8, 0, ov), u("-1", 8, 0, ov),
		i("-1", 1, -1, nil), i("-128", 8, -128, nil), i("-129", 8, 0, ov), i("127", 8, 127, nil), i("128", 8, 0, ov),
		i("-4398046511104", 43, -4398046511104, nil), i("-4398046511105", 43, 0, ov), i("4398046511103", 43, 4398046511103, nil),
		u("", 32, 0, ic), i("", 32, 0, ic), u("+", 32, 0, ic), i("+", 32, 0, ic), u("-", 32, 0, ic), i("-", 32, 0, ic),
		// parseUnsigned's: digits with separators, leading zeros.
		u("050124", 16, 50124, nil), u("65535", 16, 65535, nil), u("65_535", 16, 65535, nil), u("65536", 16, 0, ov),
		u("1_1_1", 8, 111, nil), u("1__1", 8, 11, nil),
		// Separators first or last, after a sign too.
		u("_1", 8, 0, ic), u("1_", 8, 0, ic), u("_", 8, 0, ic), u("+_1", 8, 0, ic), u("-_1", 8, 0, ic), i("-1_", 8, 0, ic),
		// Signs.
		u("+-1", 8, 0, ic), u("-+1", 8, 0, ic), u("--1", 8, 0, ic), u("++1", 8, 0, ic), u("1-", 8, 0, ic), u("1+1", 8, 0, ic),
		// The error is the first one in text order, with the accumulator of
		// max(8, bits) bits.
		u("300x", 8, 0, ov), u("x300", 8, 0, ic), u("25x", 8, 0, ic), u("2", 1, 0, ov), u("1", 1, 1, nil), u("0", 1, 0, nil),
		u("10", 1, 0, ov), u("01", 1, 1, nil), u("31", 5, 31, nil), u("32", 5, 0, ov), u("32x", 5, 0, ic), u("100", 5, 0, ov),
		u("2097151", 21, 2097151, nil), u("2097152", 21, 0, ov), u("1048576", 21, 1048576, nil),
		// Other characters.
		u("1\x00", 8, 0, ic), u("é", 8, 0, ic), u("1.5", 8, 0, ic), u("٣", 8, 0, ic),
		// 64 bits.
		u("9223372036854775807", 64, 9223372036854775807, nil), i("9223372036854775807", 64, 9223372036854775807, nil),
		i("9223372036854775808", 64, 0, ov), i("-9223372036854775808", 64, -9223372036854775808, nil),
		i("-9223372036854775809", 64, 0, ov), u("18446744073709551616", 64, 0, ov), u("99999999999999999999", 64, 0, ov),
		u("000000000000000000000000000000001", 64, 1, nil),
	} {
		var got int64
		var err error
		if c.signed {
			got, err = zigParseSigned(c.s, c.bits)
		} else {
			var v uint64
			v, err = zigParseUint(c.s, c.bits)
			got = int64(v)
		}
		if err != c.err || err == nil && got != c.want {
			t.Errorf("parseInt(%q, bits=%d, signed=%v) = %d, %v; want %d, %v", c.s, c.bits, c.signed, got, err, c.want, c.err)
		}
	}
	if v, err := zigParseUint("18446744073709551615", 64); err != nil || v != math.MaxUint64 {
		t.Errorf("max u64: %d %v", v, err)
	}
	if z, err := zigParseInt("-0", 8, false); err != nil || z.Neg || z.Mag != 0 {
		t.Errorf("-0: %+v %v", z, err)
	}
	if z, err := zigParseInt("-7", 8, true); err != nil || !z.Neg || z.Mag != 7 || z.Int() != -7 {
		t.Errorf("-7: %+v %v", z, err)
	}
}

// The recorder passes base 10 to every std.fmt.parseInt, with these types, so
// zigParseInt needs no base prefixes (docs: harness/src/main.zig,
// input.zig). If the recorder starts to use base 0, or another type, this
// test fails: the replayer's parsers must follow.
func TestCorpusReplaySelfZigParseIntSites(t *testing.T) {
	site := regexp.MustCompile(`std\.fmt\.parseInt\(\s*([a-z0-9]+)\s*,(.*),\s*([0-9a-z]+)\s*\)`)
	for file, want := range map[string]string{
		"main.zig":  "u16 u16 usize u16 u16 isize",
		"input.zig": "u16 u16 u1 u21 u8 u5 u1",
	} {
		text, err := os.ReadFile(filepath.Join(corpusDir(t), "harness", "src", file))
		if err != nil {
			t.Fatal(err)
		}
		var types []string
		for _, m := range site.FindAllStringSubmatch(string(text), -1) {
			if m[3] != "10" {
				t.Errorf("%s: parseInt(%s, ...) with base %s: zigParseInt only does base 10", file, m[1], m[3])
			}
			types = append(types, m[1])
		}
		if got := strings.Join(types, " "); got != want {
			t.Errorf("%s: parseInt types %q, the replayer is written for %q", file, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Error names.

// zigErrorNameExceptions are the sentinels of the port whose text is not the
// identifier without its `Err` prefix: name -> the Zig error name it carries.
// Each entry says why. None today: every sentinel is `Err` + the Zig name.
var zigErrorNameExceptions = map[string]string{}

// Every `Err*` variable of the non-test files of the package is declared as
// `var ErrX = errors.New("<Zig name>")`, so that err.Error() is Zig's
// @errorName (docs/mapping.md: the port never wraps errors); zigSentinels
// lists exactly these variables; nothing else in the package creates an error
// with errors.New, wraps with fmt.Errorf("...%w...") or joins.
func TestCorpusReplaySelfErrorSentinels(t *testing.T) {
	declared := map[string]string{} // Err name -> text
	news := 0
	for _, f := range parsePackageSources(t) {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			switch pkg.Name + "." + sel.Sel.Name {
			case "errors.New":
				news++
			case "errors.Join":
				t.Errorf("%s: errors.Join: the port never wraps errors", f.Name.Name)
			case "fmt.Errorf":
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && strings.Contains(lit.Value, "%w") {
					t.Errorf("fmt.Errorf with %%w: the port never wraps errors: %s", lit.Value)
				}
			}
			return true
		})
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, name := range vs.Names {
					if !strings.HasPrefix(name.Name, "Err") {
						continue
					}
					text, ok := errorsNewText(vs, i)
					if !ok {
						t.Errorf("%s is not declared as errors.New(\"<text>\")", name.Name)
						continue
					}
					declared[name.Name] = text
				}
			}
		}
	}
	if news != len(declared) {
		t.Errorf("%d errors.New calls in the package, %d Err* declarations: an error is created elsewhere", news, len(declared))
	}
	usedException := map[string]bool{}
	for name, text := range declared {
		want := strings.TrimPrefix(name, "Err")
		if ex, ok := zigErrorNameExceptions[name]; ok {
			want = ex
			usedException[name] = true
		}
		if text != want {
			t.Errorf("%s carries the text %q; the Zig error name is %q", name, text, want)
		}
	}
	for name := range zigErrorNameExceptions {
		if !usedException[name] {
			t.Errorf("zigErrorNameExceptions has %s, which is not declared", name)
		}
	}
	// The registry of zigErrorName: the same names, and the variables.
	listed := map[string]bool{}
	for _, s := range zigSentinels {
		listed[s.name] = true
		text, ok := declared[s.name]
		switch {
		case !ok:
			t.Errorf("zigSentinels lists %s, which is not declared", s.name)
		case s.err == nil || s.err.Error() != text:
			t.Errorf("zigSentinels: %s is not the variable of that name (%v)", s.name, s.err)
		}
	}
	for name := range declared {
		if !listed[name] {
			t.Errorf("%s is not in zigSentinels (corpus_replay_run_test.go): add it", name)
		}
	}
	if len(declared) < 40 {
		t.Errorf("only %d sentinels found", len(declared))
	}
}

// errorsNewText is the string of `errors.New("...")`, the value i of a var
// spec.
func errorsNewText(vs *ast.ValueSpec, i int) (string, bool) {
	if i >= len(vs.Values) {
		return "", false
	}
	call, ok := vs.Values[i].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "errors" || sel.Sel.Name != "New" {
		return "", false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	text, err := strconv.Unquote(lit.Value)
	return text, err == nil
}

func TestCorpusReplaySelfZigErrorName(t *testing.T) {
	if got := zigErrorName(ErrOutOfMemory); got != "OutOfMemory" {
		t.Errorf("zigErrorName(ErrOutOfMemory) = %q", got)
	}
	if got := zigErrorName(ErrViewportPinInsufficientRows); got != "ViewportPinInsufficientRows" {
		t.Errorf("zigErrorName(ErrViewportPinInsufficientRows) = %q", got)
	}
	for _, c := range []struct {
		name string
		err  error
		want []string
	}{
		{"wrapped", fmt.Errorf("resize: %w", ErrOutOfMemory), []string{"*fmt.wrapError", "resize: OutOfMemory", "wraps *errors.errorString"}},
		{"foreign", errors.New("OutOfMemory"), []string{"*errors.errorString", `"OutOfMemory"`, "not a bare sentinel"}},
		{"replayer", errReplayBadCase, []string{"BadCase"}},
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
			zigErrorName(c.err)
			t.Errorf("%s: no panic", c.name)
		}()
	}
}

// ---------------------------------------------------------------------------
// Empty runs.

type corpusSelfErrorer struct{ errs []string }

func (e *corpusSelfErrorer) Helper() {}
func (e *corpusSelfErrorer) Errorf(format string, args ...any) {
	e.errs = append(e.errs, fmt.Sprintf(format, args...))
}

func TestCorpusReplaySelfFailIfEmpty(t *testing.T) {
	var e corpusSelfErrorer
	corpusFailIfEmpty(&e, corpusConfig{}, "the test", 3)
	if len(e.errs) != 0 {
		t.Errorf("a test that ran 3 cases failed: %v", e.errs)
	}
	corpusFailIfEmpty(&e, corpusConfig{}, "the test", 0)
	if len(e.errs) != 1 || !strings.Contains(e.errs[0], "the test ran no case") || !strings.Contains(e.errs[0], "TERM51_ALLOW_EMPTY") {
		t.Errorf("an empty test: %v", e.errs)
	}
	e = corpusSelfErrorer{}
	cfg, err := corpusConfigFrom(func(k string) string {
		if k == "TERM51_ALLOW_EMPTY" {
			return "1"
		}
		return ""
	}, false)
	if err != nil || !cfg.allowEmpty {
		t.Fatalf("TERM51_ALLOW_EMPTY: %+v %v", cfg, err)
	}
	corpusFailIfEmpty(&e, cfg, "the test", 0)
	if len(e.errs) != 0 {
		t.Errorf("TERM51_ALLOW_EMPTY did not allow: %v", e.errs)
	}

	// A filter that matches nothing, on the real corpus: no case is selected.
	cfg, err = corpusConfigFrom(func(k string) string {
		if k == "TERM51_ONLY" {
			return "^csi/cup/no_such_case$"
		}
		return ""
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	area := corpusAreaByName(t, "csi")
	run := &corpusRun{cfg: cfg, area: area}
	parsed := corpusReadSuite(t, area, "cup")
	if sel := run.selectCases(t, "cup", parsed, []replayKind{replayKindGolden}); len(sel) != 0 {
		t.Errorf("a filter that matches nothing selected %v", sel)
	}
	// Sampling n = 0 means all, so an empty selection only comes from a filter
	// or an empty suite: corpusSelect of nothing.
	if got := corpusSelect(0, 20, nil); len(got) != 0 {
		t.Errorf("corpusSelect(0, ...) = %v", got)
	}
}

// ---------------------------------------------------------------------------
// The golden files against the MANIFEST.

func TestCorpusReplaySelfSectionsVerify(t *testing.T) {
	text := "## case a\nl1\n## case b\nl2\n## case c\n"
	sum := sha256.Sum256([]byte(text))
	good := corpusManifestEntry{sha256: hex.EncodeToString(sum[:]), unit: "cases", count: 3}
	open := func(data string) *corpusSections {
		h := sha256.New()
		s := corpusSectionsReader(io.TeeReader(strings.NewReader(data), h))
		s.hash = h
		return s
	}

	s := open(text)
	if err := s.verify("x", good, true); err != nil {
		t.Errorf("a good file: %v", err)
	}
	// Reading some sections and then draining reads the rest.
	s = open(text)
	if _, _, err := s.section(1); err != nil {
		t.Fatal(err)
	}
	if err := s.verify("x", good, true); err != nil {
		t.Errorf("a good file, drained after a partial read: %v", err)
	}
	// Without draining, a file that was not read to its end is not checked,
	// one that was is.
	s = open(text)
	s.section(1)
	if err := s.verify("x", corpusManifestEntry{sha256: "bad", unit: "cases", count: 9}, false); err != nil {
		t.Errorf("a partial read without drain was checked: %v", err)
	}
	s = open(text)
	s.count()
	if err := s.verify("x", corpusManifestEntry{sha256: "bad", unit: "cases", count: 3}, false); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Errorf("a file read to its end with a bad hash: %v", err)
	}
	// A truncated file: the sha256 and the count differ.
	s = open(text[:len(text)-len("## case c\n")])
	if err := s.verify("x", good, true); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Errorf("a truncated file: %v", err)
	}
	// The right text with a wrong count in the MANIFEST.
	bad := good
	bad.count = 4
	if err := open(text).verify("x", bad, true); err == nil || !strings.Contains(err.Error(), "3 `## case` sections, MANIFEST says cases=4") {
		t.Errorf("a wrong count: %v", err)
	}
	// An entry without a case count (not a golden file of an area).
	if err := open(text).verify("x", corpusManifestEntry{sha256: good.sha256}, true); err == nil {
		t.Errorf("an entry without cases= was accepted")
	}
	// A window, and an in-memory reader without a hash, are not checked.
	if err := corpusSectionsReader(strings.NewReader(text)).verify("x", bad, true); err != nil {
		t.Errorf("a reader without a hash was checked: %v", err)
	}
	// An edited file with the same number of sections.
	if err := open(strings.Replace(text, "l1", "l9", 1)).verify("x", good, true); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Errorf("an edited file: %v", err)
	}
	// A gzip stream that is cut: the error comes from the reader.
	var gzbuf bytes.Buffer
	zw := gzip.NewWriter(&gzbuf)
	zw.Write([]byte(text))
	zw.Close()
	cut := gzbuf.Bytes()[:gzbuf.Len()-8]
	zr, err := gzip.NewReader(bytes.NewReader(cut))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	s = corpusSectionsReader(io.TeeReader(zr, h))
	s.hash = h
	if err := s.verify("x", good, true); err == nil {
		t.Errorf("a cut gzip stream was accepted")
	}

	// The tuning of the drain: a TERM51_ONLY filter without TERM51_FULL does
	// not read to the end.
	for env, want := range map[string]bool{"": true, "TERM51_FULL=1": true, "TERM51_ONLY=x": false, "TERM51_ONLY=x TERM51_FULL=1": true} {
		vars := map[string]string{}
		for _, kv := range strings.Fields(env) {
			k, v, _ := strings.Cut(kv, "=")
			vars[k] = v
		}
		cfg, err := corpusConfigFrom(func(k string) string { return vars[k] }, false)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.drains() != want {
			t.Errorf("drains() with %q = %v, want %v", env, cfg.drains(), want)
		}
	}
}

// The MANIFEST lines of the three kinds of files parse into hash, unit and
// count.
func TestCorpusReplaySelfManifestEntries(t *testing.T) {
	m := corpusManifest(t)
	if e := m["csi/cbt.golden.gz"]; e.unit != "cases" || e.count != 1115 || len(e.sha256) != 64 {
		t.Errorf("csi/cbt.golden.gz: %+v", e)
	}
	if e := m["tables/modes.txt"]; e.unit != "lines" || e.count != 40 {
		t.Errorf("tables/modes.txt: %+v", e)
	}
	if e := m["input/paste.out.gz"]; e.unit != "requests" || e.count == 0 {
		t.Errorf("input/paste.out.gz: %+v", e)
	}
	for _, mode := range []inputMode{inputModeMacos, inputModeIos} {
		if e := inputManifest(t, mode)["input/paste.out.gz"]; e.unit != "requests" || e.sha256 != m["input/paste.out.gz"].sha256 {
			t.Errorf("%s input/paste.out.gz: %+v", mode.name, e)
		}
	}
}

// ---------------------------------------------------------------------------
// The split runner: the whole run first.

type splitSelfReporter struct {
	errs  []string
	fails int
}

func (r *splitSelfReporter) Helper()             {}
func (r *splitSelfReporter) Error(args ...any)   { r.errs = append(r.errs, fmt.Sprint(args...)) }
func (r *splitSelfReporter) Fail()               { r.fails++ }
func (r *splitSelfReporter) Logf(string, ...any) {}

func TestCorpusReplaySelfSplitCheckWholeFirst(t *testing.T) {
	c := replayNamed{name: "c", ops: []replayOp{{Tag: replayOpFeed, Feed: []uint8("ABCDE")}}}
	want := []byte("## case c\nx\n")
	differ := []byte("## case c\ny\n")
	// Variants of the case: bytewise and the offsets 1..4.
	nVariants := len(splitVariants("csi", "cup", c, splitConfig{allUpTo: 64, head: 64, seeded: 3}))
	if nVariants != 5 {
		t.Fatalf("%d variants", nVariants)
	}
	defer func(f func(replayNamed, replayKind, feedMode) []byte) { splitProduceFn = f }(splitProduceFn)

	check := func(produce func(mode feedMode) []byte) (*splitSelfReporter, *splitStats, []feedMode, int) {
		var modes []feedMode
		splitProduceFn = func(_ replayNamed, _ replayKind, mode feedMode) []byte {
			modes = append(modes, mode)
			return produce(mode)
		}
		rep := &splitSelfReporter{}
		run := &corpusRun{area: corpusArea{name: "csi"}}
		st := &splitStats{}
		n := splitCheck(rep, run, st, nil, splitConfig{allUpTo: 64, head: 64, seeded: 3}, "cup", c, replayKindGolden, want)
		return rep, st, modes, n
	}

	// Everything equal: the whole run first, then the five variants.
	rep, st, modes, n := check(func(feedMode) []byte { return want })
	if n != 0 || len(rep.errs) != 0 || rep.fails != 0 || st.variants != 5 || st.skipped != 0 || st.whole != 0 || len(modes) != 6 ||
		modes[0].Tag != feedModeWhole || modes[1].Tag != feedModeBytewise {
		t.Errorf("all equal: n=%d errs=%v st=%+v modes=%v", n, rep.errs, st, modes)
	}

	// The whole run differs: reported once, as whole, the variants skipped
	// (not even run) and counted.
	rep, st, modes, n = check(func(feedMode) []byte { return differ })
	if n != 1 || len(rep.errs) != 1 || st.variants != 0 || st.skipped != 5 || st.whole != 1 || len(modes) != 1 || modes[0].Tag != feedModeWhole ||
		st.mismatch[replayKindGolden] != 1 {
		t.Errorf("whole differs: n=%d errs=%v st=%+v modes=%v", n, rep.errs, st, modes)
	}
	if len(rep.errs) == 1 && !strings.HasPrefix(rep.errs[0], "MISMATCH csi/cup case=c kind=golden split=whole dump=- line 2\n  got:  y\n  want: x") {
		t.Errorf("whole-run report %q", rep.errs[0])
	}

	// The whole run matches and one variant differs: the report has the split
	// and the seed; the other variants are still replayed.
	rep, st, _, n = check(func(m feedMode) []byte {
		if m.Tag == feedModeSplit && m.At == 2 {
			return differ
		}
		return want
	})
	if n != 1 || len(rep.errs) != 1 || st.variants != 5 || st.skipped != 0 || st.whole != 0 ||
		!strings.HasPrefix(rep.errs[0], "MISMATCH csi/cup case=c kind=golden split=op:0,at:2,seed=0x") {
		t.Errorf("one variant differs: n=%d errs=%v st=%+v", n, rep.errs, st)
	}

	// The summary line names the skipped variants and the whole runs.
	st = &splitStats{corpusSuiteStats: corpusSuiteStats{cases: 9, run: 3}, variants: 40, known: 1, skipped: 12, whole: 2}
	st.mismatch[replayKindGolden] = 3
	if got, want := splitSummary("csi", "cup", st),
		"SPLIT csi/cup cases=9 run=3 variants=40 known=1 skipped=12 whole=2 mismatch=3 (golden=3)"; got != want {
		t.Errorf("summary %q, want %q", got, want)
	}
}
