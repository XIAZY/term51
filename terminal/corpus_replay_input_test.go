package terminal

// The keyboard and paste corpus of the replayer: the Go side of the recorder's
// harness/src/input.zig. golden/input/*.out.gz (Linux), golden-macos/input
// (darwin) and golden-ios/input (ios) hold, per file, pairs of lines: the
// request as the recorder echoed it, then `=> ...` with its output. The
// request files themselves (cases/input/*.req) are not in the repository:
// the echoed lines are the requests.
//
// inputRun is fed one request at a time and its text (request line, then
// output line) is compared with the two recorded lines. A file has up to
// hundreds of thousands of requests; the default run checks corpusInputSample
// (500) of them, spread evenly (corpusSelect), the full run all of them.
//
//	TERM51_FULL=1          every request
//	TERM51_SAMPLE=n        n requests per file (default 500)
//	TERM51_ONLY=<regexp>   on `input/<mode>/<file>/<index>` (index from 0)
//	TERM51_FAIL_LIST=file  appends `input/<mode>/<file>:<index>:input`
//	TERM51_REPORT_MAX=n    reports per file (default 10); per test 5n
//	TERM51_ALLOW_EMPTY=1   a test that checked no request is not a failure
//
// A report:
//
//	MISMATCH input/linux/keys_legacy req=123 line 247
//	  req:  k ...
//	  got:  => ...
//	  want: => ...

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const corpusInputSample = 500

// The errors of input.zig that are not Zig's std.fmt ones: error.BadCase and
// std.fmt.hexToBytes' errors.
var (
	errInputInvalidLength    = errors.New("InvalidLength")
	errInputInvalidCharacter = errors.New("InvalidCharacter")
)

// Mirrors harness/src/input.zig:28 run
//
// Input file, one request per line ('#' starts a comment line).
func inputRun(text string, w *bytes.Buffer) error {
	for text != "" {
		var raw string
		raw, text, _ = strings.Cut(text, "\n")
		line := strings.Trim(raw, " \t\r")
		if line == "" || line[0] == '#' {
			continue
		}
		fmt.Fprintf(w, "%s\n", line)
		it := &replayTokens{rest: line}
		kind, _ := it.next()
		switch kind {
		case "k":
			if err := inputKey(it, w); err != nil {
				return err
			}
		case "p":
			if err := inputPaste(it, w); err != nil {
				return err
			}
		default:
			return errReplayBadCase
		}
	}
	return nil
}

// Mirrors harness/src/input.zig:44 hexArg
func inputHexArg(arg string) ([]uint8, error) {
	if arg == "-" {
		return []uint8{}, nil
	}
	// std.fmt.hexToBytes: the length is checked before the digits.
	if len(arg)%2 != 0 {
		return nil, errInputInvalidLength
	}
	buf, err := hex.DecodeString(arg)
	if err != nil {
		return nil, errInputInvalidCharacter
	}
	return buf, nil
}

// Mirrors harness/src/input.zig:52 writeHex
func inputWriteHex(w *bytes.Buffer, bytes_ []uint8) {
	if len(bytes_) == 0 {
		w.WriteString("-")
		return
	}
	for _, c := range bytes_ {
		fmt.Fprintf(w, "%02x", c)
	}
}

// inputKeyArgs are the fields of a `k` request, as inputKey reads them
// (input.zig:58-68), the names still unresolved.
type inputKeyArgs struct {
	keyName    string
	actionName string
	mods       uint16
	consumed   uint16
	composing  uint8 // Zig: u1
	ucp        uint32
	utf8       []uint8
	opts       uint8
	kitty      uint8 // Zig: u5
	oaaName    string
}

// inputKeyParse is the first half of inputKey (input.zig:58-68): the fields
// in order, every missing one is error.BadCase, every number is parsed with
// the width of its Zig type.
func inputKeyParse(it *replayTokens) (inputKeyArgs, error) {
	var a inputKeyArgs
	var err error
	word := func() (string, error) {
		s, ok := it.next()
		if !ok {
			return "", errReplayBadCase
		}
		return s, nil
	}
	number := func(bits int) (uint64, error) {
		s, err := word()
		if err != nil {
			return 0, err
		}
		return zigParseUint(s, bits)
	}
	if a.keyName, err = word(); err != nil {
		return a, err
	}
	if a.actionName, err = word(); err != nil {
		return a, err
	}
	var v uint64
	if v, err = number(16); err != nil {
		return a, err
	}
	a.mods = uint16(v)
	if v, err = number(16); err != nil {
		return a, err
	}
	a.consumed = uint16(v)
	if v, err = number(1); err != nil {
		return a, err
	}
	a.composing = uint8(v)
	if v, err = number(21); err != nil {
		return a, err
	}
	a.ucp = uint32(v)
	s, err := word()
	if err != nil {
		return a, err
	}
	if a.utf8, err = inputHexArg(s); err != nil {
		return a, err
	}
	if v, err = number(8); err != nil {
		return a, err
	}
	a.opts = uint8(v)
	if v, err = number(5); err != nil {
		return a, err
	}
	a.kitty = uint8(v)
	a.oaaName, err = word()
	return a, err
}

// The name maps of std.meta.stringToEnum(input.KeyAction | input.Key |
// OptionAsAlt, name): built once from the String methods of the enums, which
// return the Zig tag names. The build is retried until it succeeds (a
// not-yet-ported String panics, and the panic must not be remembered).
var inputNames struct {
	sync.Mutex
	key    map[string]InputKeyKey
	action map[string]InputKeyAction
	oaa    map[string]InputConfigOptionAsAlt
}

func inputNamesBuild() {
	inputNames.Lock()
	defer inputNames.Unlock()
	if inputNames.key != nil {
		return
	}
	key := map[string]InputKeyKey{}
	for _, k := range InputKeyKeyValues() {
		key[k.String()] = k
	}
	action := map[string]InputKeyAction{}
	for a := InputKeyAction(0); a < inputKeyActionCount; a++ {
		action[a.String()] = a
	}
	oaa := map[string]InputConfigOptionAsAlt{}
	for o := InputConfigOptionAsAlt(0); o < inputConfigOptionAsAltCount; o++ {
		oaa[o.String()] = o
	}
	inputNames.action, inputNames.oaa, inputNames.key = action, oaa, key
}

// Mirrors harness/src/input.zig:57 key
func inputKey(it *replayTokens, w *bytes.Buffer) error {
	a, err := inputKeyParse(it)
	if err != nil {
		return err
	}
	inputNamesBuild()
	action, ok := inputNames.action[a.actionName]
	if !ok {
		return errReplayBadCase
	}
	key, ok := inputNames.key[a.keyName]
	if !ok {
		return errReplayBadCase
	}

	// The request names the action: `action` is set explicitly (the zero
	// value of InputKeyAction is release, Zig's default is press).
	event := InputKeyKeyEvent{
		action:              action,
		key:                 key,
		mods:                InputKeyModsModsFromInt(a.mods),
		consumed_mods:       InputKeyModsModsFromInt(a.consumed),
		composing:           a.composing == 1,
		utf8:                a.utf8,
		unshifted_codepoint: a.ucp,
	}

	opts := InputKeyEncodeOptionsDefault
	opts.cursor_key_application = a.opts&1 != 0
	opts.keypad_key_application = a.opts&2 != 0
	opts.ignore_keypad_with_numlock = a.opts&4 != 0
	opts.alt_esc_prefix = a.opts&8 != 0
	opts.modify_other_keys_state_2 = a.opts&16 != 0
	opts.kitty_flags = KittyKeyFlagsFromInt(a.kitty)
	if opts.macos_option_as_alt, ok = inputNames.oaa[a.oaaName]; !ok {
		return errReplayBadCase
	}

	aw := oscWriterAllocatingInit()
	defer aw.deinit()
	if err := InputKeyEncodeEncode(&aw.writer, event, opts); err != nil {
		fmt.Fprintf(w, "=> !%s\n", zigErrorName(err))
		return nil
	}
	w.WriteString("=> ")
	inputWriteHex(w, aw.written())
	w.WriteString("\n")
	return nil
}

// Mirrors harness/src/input.zig:103 paste
func inputPaste(it *replayTokens, w *bytes.Buffer) error {
	s, ok := it.next()
	if !ok {
		return errReplayBadCase
	}
	bracketed, err := zigParseUint(s, 1)
	if err != nil {
		return err
	}
	s, ok = it.next()
	if !ok {
		return errReplayBadCase
	}
	data, err := inputHexArg(s)
	if err != nil {
		return err
	}

	safe := InputPasteIsSafe(data)

	// The mutable form never fails: it rewrites unsafe bytes in place. Go
	// passes a copy, so the aliasing of the Zig (`@as([]u8, data)`) is
	// explicit and `data` stays intact.
	segs := InputPasteEncodeMutable(append([]uint8(nil), data...), InputPasteOptions{bracketed: bracketed == 1})
	w.WriteString("=> ")
	for i, seg := range segs {
		if i > 0 {
			w.WriteString("|")
		}
		inputWriteHex(w, seg)
	}
	safeInt := 0
	if safe {
		safeInt = 1
	}
	fmt.Fprintf(w, " safe=%d\n", safeInt)
	return nil
}

// ---------------------------------------------------------------------------
// The files of the input corpus and their requests.

// inputMode is one recording of the input corpus: the platform the Go
// encoder is told it runs on (testPlatform) and the directory below the
// corpus.
type inputMode struct {
	name string // linux, macos, ios: the test and the report name
	goos string // the platform the recording was made for
	dir  string // below the corpus
}

var (
	inputModeLinux = inputMode{"linux", "linux", filepath.Join("golden", "input")}
	inputModeMacos = inputMode{"macos", "darwin", filepath.Join("golden-macos", "input")}
	inputModeIos   = inputMode{"ios", "ios", filepath.Join("golden-ios", "input")}
)

// inputRequest is one request of a file with its recorded output line.
type inputRequest struct {
	line int    // line number of the request in the file, 1-based
	req  string // the echoed request line
	want string // the line `=> ...`
}

// inputFiles are the names of the files of a mode (without .out.gz), sorted.
func inputFiles(t testing.TB, mode inputMode) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(corpusDir(t), mode.dir))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".out.gz"); ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// inputPairs splits the lines of an output file into requests: the request
// and its `=>` line alternate.
func inputPairs(lines []string) ([]inputRequest, error) {
	if len(lines)%2 != 0 {
		return nil, fmt.Errorf("%d lines: not pairs", len(lines))
	}
	out := make([]inputRequest, 0, len(lines)/2)
	for i := 0; i < len(lines); i += 2 {
		if !strings.HasPrefix(lines[i+1], "=> ") || strings.HasPrefix(lines[i], "=> ") {
			return nil, fmt.Errorf("line %d: %q is not a request followed by a `=> ` line", i+1, lines[i])
		}
		out = append(out, inputRequest{line: i + 1, req: lines[i], want: lines[i+1]})
	}
	return out, nil
}

// inputManifests caches the MANIFEST of each mode (golden*/MANIFEST).
var inputManifests struct {
	sync.Mutex
	m map[string]map[string]corpusManifestEntry
}

// inputManifest is the MANIFEST that lists the files of a mode.
func inputManifest(t testing.TB, mode inputMode) map[string]corpusManifestEntry {
	t.Helper()
	inputManifests.Lock()
	defer inputManifests.Unlock()
	rel := filepath.Join(filepath.Dir(mode.dir), "MANIFEST")
	if inputManifests.m == nil {
		inputManifests.m = map[string]map[string]corpusManifestEntry{}
	}
	if inputManifests.m[rel] == nil {
		inputManifests.m[rel] = corpusManifestAt(t, rel)
	}
	return inputManifests.m[rel]
}

// inputReadFile reads the requests of a file of a mode. The whole file is
// read, so its sha256 and its number of requests are checked against the
// MANIFEST of the mode (the golden files of the areas are checked in
// corpusSections.verify): a truncated file must not shrink the corpus.
func inputReadFile(t testing.TB, mode inputMode, file string) []inputRequest {
	t.Helper()
	rel := filepath.Join(mode.dir, file+".out.gz")
	text := corpusReadAll(t, rel)
	key := "input/" + file + ".out.gz"
	want, ok := inputManifest(t, mode)[key]
	if !ok || want.unit != "requests" {
		t.Fatalf("%s is not in the MANIFEST of %s with a request count", key, mode.name)
	}
	if sum := sha256.Sum256(text); hex.EncodeToString(sum[:]) != want.sha256 {
		t.Fatalf("%s: sha256 of the uncompressed text is %x, MANIFEST says %s", rel, sum, want.sha256)
	}
	var lines []string
	if len(text) > 0 {
		lines = strings.Split(strings.TrimSuffix(string(text), "\n"), "\n")
	}
	reqs, err := inputPairs(lines)
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	if len(reqs) != want.count {
		t.Fatalf("%s: %d requests, MANIFEST says requests=%d", rel, len(reqs), want.count)
	}
	return reqs
}

// inputProduce is inputRun for one request; a panic of the port becomes a
// line of the text, so it is a mismatch like any other.
func inputProduce(req string) (out string) {
	var buf bytes.Buffer
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprintf("%s\n!panic: %v\n", req, r)
		}
	}()
	if err := inputRun(req+"\n", &buf); err != nil {
		fmt.Fprintf(&buf, "!replay error=%s\n", err.Error())
	}
	return buf.String()
}

// inputSelect chooses the requests of a file to check (see the file header).
func inputSelect(cfg corpusConfig, sampleSet bool, area, file string, total int) []int {
	if cfg.exhaustive() {
		var out []int
		for i := 0; i < total; i++ {
			if cfg.wants(area, file, strconv.Itoa(i)) {
				out = append(out, i)
			}
		}
		return out
	}
	n := corpusInputSample
	if sampleSet {
		n = cfg.sample
	}
	return corpusSelect(total, n, nil)
}

// inputFormatMismatch is the report of one request.
func inputFormatMismatch(area, file string, index int, r inputRequest, got string) string {
	gotLine := "<end of text>"
	if _, rest, ok := strings.Cut(got, "\n"); ok {
		gotLine = corpusCut([]byte(strings.TrimSuffix(rest, "\n")))
	}
	return fmt.Sprintf("MISMATCH %s/%s req=%d line %d\n  req:  %s\n  got:  %s\n  want: %s",
		area, file, index, r.line, corpusCut([]byte(r.req)), gotLine, corpusCut([]byte(r.want)))
}

// corpusReplayInputFile checks the selected requests of one file.
func corpusReplayInputFile(t *testing.T, run *corpusRun, mode inputMode, file string) {
	area := "input/" + mode.name
	reqs := inputReadFile(t, mode, file)
	_, sampleSet := os.LookupEnv("TERM51_SAMPLE")
	selected := inputSelect(run.cfg, sampleSet, area, file, len(reqs))
	run.ran.Add(int64(len(selected)))
	reported, mismatch := 0, 0
	for _, i := range selected {
		r := reqs[i]
		got := inputProduce(r.req)
		if got == r.req+"\n"+r.want+"\n" {
			continue
		}
		mismatch++
		t.Fail()
		if reported < run.cfg.suiteMax() && run.reported.Add(1) <= int64(run.cfg.testMax()) {
			reported++
			t.Error(inputFormatMismatch(area, file, i, r, got))
		}
		run.appendLine(run.cfg.failList, fmt.Sprintf("%s/%s:%d:input", area, file, i))
	}
	t.Logf("SUITE %s/%s requests=%d run=%d mismatch=%d", area, file, len(reqs), len(selected), mismatch)
}

// appendLine appends a line to a file (TERM51_FAIL_LIST); errors are ignored.
func (run *corpusRun) appendLine(path, line string) {
	if path == "" {
		return
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, line)
}

// corpusReplayInput is the body of TestCorpusReplayInput<Mode>: a subtest per
// file, sequential (the platform is a global).
func corpusReplayInput(t *testing.T, mode inputMode) {
	testPlatform(t, mode.goos)
	cfg, err := corpusConfigFrom(os.Getenv, testing.Short())
	if err != nil {
		t.Fatal(err)
	}
	run := &corpusRun{cfg: cfg}
	// Cleanup runs when the subtests are done.
	t.Cleanup(func() { corpusFailIfEmpty(t, cfg, "input replay "+mode.name, int(run.ran.Load())) })
	for _, file := range inputFiles(t, mode) {
		t.Run(file, func(t *testing.T) { corpusReplayInputFile(t, run, mode, file) })
	}
}

func TestCorpusReplayInputLinux(t *testing.T) { corpusReplayInput(t, inputModeLinux) }
func TestCorpusReplayInputMacos(t *testing.T) { corpusReplayInput(t, inputModeMacos) }
func TestCorpusReplayInputIos(t *testing.T)   { corpusReplayInput(t, inputModeIos) }
