package terminal

// The search area of the corpus replayer: the Go side of the recorder's
// harness/src/search.zig. It replays cases/search/<suite>.case through the Go
// search code (search_screen.go, search_active.go, search_viewport.go,
// search_pagelist.go, search_sliding_window.go) and compares the trace with
// golden/search/<suite>.search.gz byte for byte. The format of the case file
// and of the dump is documented in search.zig and in
// term51/docs/search-corpus.md; the Go code mirrors search.zig, one Go function
// per Zig function or op, each with `// Mirrors harness/src/search.zig:<line>
// <ZigName>`. The control flow follows the Zig.
//
// It reuses the machinery of the terminal areas: the case selection
// (corpusSelect), the golden section files and their manifest check
// (corpusSections), the comparison and the mismatch report (corpusCompare,
// corpusReport), the environment variables (corpus_replay_test.go header). The
// kind of the area is replayKindSearch (`kind=search` in a report; the file
// `<suite>.search.gz`); a report names the op that the first differing line
// belongs to, `dump=op <n>`, and `near:` is empty.
//
// # Marks
//
// tools/record.py writes a marker op as the first op of a case
// (docs/search-corpus.md, tools/search_marks.txt):
//
//	unchecked_only <reason>  upstream aborts in a safe build on this case
//	                         (reason underflow: sliding_window.zig:279;
//	                         empty_needle). The golden section is the output of the
//	                         UNCHECKED release recorder: the case is replayed and
//	                         compared like any other. The status line
//	                         `!recorded-from-unchecked-release reason=<r>` is its
//	                         first line (searchRuns).
//	no_result <reason>       upstream has no valid result in any build (abort,
//	                         hang, undefined behaviour); the golden section is the
//	                         single line `!no-recorded-result <reason>`. The case is
//	                         NOT replayed: it is skipped and counted (no_result=N in
//	                         the suite line); the Go-only ruling tests of
//	                         search_rulings_test.go cover these cases (the empty
//	                         needle: 180 cases; ScreenSearch.feed on a completed
//	                         search: 48 cases). The skipped sections are not
//	                         selected by the sampling either. runCase still
//	                         writes the status line for such a case, as the Zig
//	                         does, and TestCorpusReplaySelfSearchMarkers compares
//	                         it with every recorded section.
//
// # Page capacity
//
// The corpus is recorded with the production page capacity (grapheme_bytes
// 8192: 46 rows per page at 1000 columns). The replayer never calls
// testStdCapacity; TestCorpusReplaySearch fails if PageStdCapacity does not
// have the production value.
//
// # Differences from the Zig, forced by the language
//
//   - the writer is a *bytes.Buffer; a Zig `try` that can only fail on the
//     writer disappears, an error of the terminal's creation is returned and
//     becomes a line of the output (corpusProduce style);
//   - Zig's `Searcher` union holds the searcher by value and every use is by
//     pointer: searchSearcher holds a pointer to it;
//   - the dump records (layout, hl, win, st) are built by the print functions
//     and written by their `format` methods, so that the formats can be tested
//     against the recorded lines without the port (the self tests);
//   - an error that Zig prints with @errorName is printed by zigErrorName;
//     `@tagName` of the recorder's own enums is the table of names of this
//     file (the String methods of the port's enums are the port's).

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// searchRuns is `recorder_options.search_runs` (search.zig:898): which recorder
// runs the cases marked unchecked_only. The golden files hold the section of
// the "unchecked" recorder (ReleaseFast), so every such case is run.
const searchRuns = "unchecked"

// searchArea is the row of the search area. It is not in corpusAreas, whose
// rows are the terminal suites parsed by parseSuite.
var searchArea = corpusArea{
	name:      "search",
	casesDir:  "cases/search",
	goldenDir: "golden/search",
	kinds:     replayKindSetOf(replayKindSearch),
}

// ---------------------------------------------------------------------------
// Case files: search.zig Kind, Open, Op, Case, Suite, parseSuite.

// Mirrors harness/src/search.zig:147 Kind
type searchKind uint8

const (
	searchKindScreen searchKind = iota
	searchKindActive
	searchKindViewport
	searchKindPagelist
	searchKindWindow
)

var searchKindNames = [...]string{"screen", "active", "viewport", "pagelist", "window"}

func (k searchKind) String() string { return searchKindNames[k] }

// searchScreenNames are the tag names of ScreenSet.Key (@tagName(o.screen)).
var searchScreenNames = [...]string{"primary", "alternate"}

// Mirrors harness/src/search.zig:149 Open
type searchOpen struct {
	kind   searchKind
	needle []uint8
	screen ScreenSetKey // default primary
	// null = last
	start   Optional[uint]
	reverse bool
	dirty   Optional[bool]
}

// searchOpTag is the tag of Op (search.zig:158).
type searchOpTag uint8

const (
	searchOpInit searchOpTag = iota
	searchOpWrite
	searchOpRepeat
	searchOpResize
	searchOpScrollViewport
	searchOpScrollTop
	searchOpScrollActive
	searchOpReset
	searchOpOpen
	searchOpClose
	searchOpUpdate
	searchOpNext
	searchOpDrain
	searchOpFeed
	searchOpTick
	searchOpSearchAll
	searchOpSelectNext
	searchOpSelectPrev
	searchOpUpdateActive
	searchOpMatches
	searchOpSelected
	searchOpAppend
	searchOpAppendAll
	searchOpAppendAllRev
	searchOpClear
	searchOpViewportReset
	searchOpViewportDirty
	searchOpUncheckedOnly
	searchOpNoResult
)

var searchOpNames = [...]string{
	"init", "write", "repeat", "resize", "scroll_viewport", "scroll_top", "scroll_active", "reset",
	"open", "close", "update", "next", "drain", "feed", "tick", "search_all", "select_next",
	"select_prev", "update_active", "matches", "selected", "append", "append_all", "append_all_rev",
	"clear", "viewport_reset", "viewport_dirty", "unchecked_only", "no_result",
}

// String is @tagName(op).
func (t searchOpTag) String() string { return searchOpNames[t] }

// searchOpRepeatArgs is the payload of Op.repeat.
type searchOpRepeatArgs struct {
	n     uint64 // Zig: usize
	bytes []uint8
}

// searchOp is Op, a tagged union: only the field named by Tag is meaningful.
// Init and Resize are the payloads of the terminal ops' replayOp.
type searchOp struct {
	Tag searchOpTag

	Init           replayOpInitArgs
	Write          []uint8
	Repeat         searchOpRepeatArgs
	Resize         replayOpResizeArgs
	ScrollViewport int64 // Zig: isize (the 64-bit reference)
	Open           searchOpen
	Drain          uint // Zig: usize
	Append         uint // Zig: usize
	ViewportDirty  Optional[bool]
	// unchecked_only and no_result
	Reason string
}

// Mirrors harness/src/search.zig:190 Case
type searchCase struct {
	name string
	ops  []searchOp
}

// Mirrors harness/src/search.zig:192 Suite (without the arena)
type searchSuite struct {
	cases []searchCase
}

// Mirrors harness/src/search.zig:203 isSearchSuite
//
// True if the file is a search suite: its first line that is neither empty
// nor a comment is `kinds search`.
func isSearchSuite(text string) bool {
	for text != "" {
		var raw string
		raw, text, _ = strings.Cut(text, "\n")
		line := strings.Trim(raw, " \t\r")
		if line == "" || line[0] == '#' {
			continue
		}
		return line == "kinds search"
	}
	return false
}

// Mirrors harness/src/search.zig:213 hexBytes
func hexBytes(tok string) ([]uint8, error) {
	if tok == "-" {
		return []uint8{}, nil
	}
	if len(tok)%2 != 0 {
		return nil, errReplayBadCase
	}
	buf, err := hex.DecodeString(tok)
	if err != nil {
		return nil, errReplayBadCase
	}
	return buf, nil
}

// Mirrors harness/src/search.zig:221 parseOptBool
func parseOptBool(tok string) (Optional[bool], error) {
	switch tok {
	case "null":
		return None[bool](), nil
	case "true":
		return Some(true), nil
	case "false":
		return Some(false), nil
	}
	return None[bool](), errReplayBadCase
}

// Mirrors harness/src/search.zig:228 parseSuite
//
// Lines of a search case file; `kinds` is skipped; every other line before the
// first `case` is a BadCase. The arms are parseSearchOp.
func parseSearchSuite(text string) (searchSuite, error) {
	var cases []searchCase
	var name Optional[string]
	var ops []searchOp
	for text != "" {
		var raw string
		raw, text, _ = strings.Cut(text, "\n")
		line := strings.Trim(raw, " \t\r")
		if line == "" || line[0] == '#' {
			continue
		}
		it := &replayTokens{rest: line}
		cmd, _ := it.next()
		if cmd == "kinds" {
			continue
		} else if cmd == "case" {
			if name.Valid {
				cases = append(cases, searchCase{name: name.Value, ops: ops})
			}
			n, ok := it.next()
			if !ok {
				return searchSuite{}, errReplayBadCase
			}
			name = Some(n)
			ops = nil
			continue
		}
		if !name.Valid {
			return searchSuite{}, errReplayBadCase
		}
		op, err := parseSearchOp(cmd, it)
		if err != nil {
			return searchSuite{}, err
		}
		ops = append(ops, op)
	}
	if name.Valid {
		cases = append(cases, searchCase{name: name.Value, ops: ops})
	}
	return searchSuite{cases: cases}, nil
}

// searchNextToken is `it.next() orelse return error.BadCase`.
func searchNextToken(it *replayTokens) (string, error) {
	tok, ok := it.next()
	if !ok {
		return "", errReplayBadCase
	}
	return tok, nil
}

// searchParseUints reads the next tokens as std.fmt.parseInt of the unsigned
// types of `bits` bits.
func searchParseUints(it *replayTokens, bits ...int) ([]uint64, error) {
	out := make([]uint64, len(bits))
	for i, b := range bits {
		tok, err := searchNextToken(it)
		if err != nil {
			return nil, err
		}
		if out[i], err = zigParseUint(tok, b); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Mirrors harness/src/search.zig:252 parseSuite, the labeled block `op:` that
// follows the chain of `eq(u8, cmd, ...)`.
func parseSearchOp(cmd string, it *replayTokens) (searchOp, error) {
	switch cmd {
	case "init":
		v, err := searchParseUints(it, 16, 16, 64)
		if err != nil {
			return searchOp{}, err
		}
		return searchOp{Tag: searchOpInit, Init: replayOpInitArgs{cols: uint16(v[0]), rows: uint16(v[1]), max_scrollback: v[2]}}, nil
	case "write":
		tok, err := searchNextToken(it)
		if err != nil {
			return searchOp{}, err
		}
		b, err := hexBytes(tok)
		return searchOp{Tag: searchOpWrite, Write: b}, err
	case "repeat":
		v, err := searchParseUints(it, 64)
		if err != nil {
			return searchOp{}, err
		}
		tok, err := searchNextToken(it)
		if err != nil {
			return searchOp{}, err
		}
		b, err := hexBytes(tok)
		return searchOp{Tag: searchOpRepeat, Repeat: searchOpRepeatArgs{n: v[0], bytes: b}}, err
	case "resize":
		v, err := searchParseUints(it, 16, 16)
		if err != nil {
			return searchOp{}, err
		}
		return searchOp{Tag: searchOpResize, Resize: replayOpResizeArgs{cols: uint16(v[0]), rows: uint16(v[1])}}, nil
	case "scroll_viewport":
		tok, err := searchNextToken(it)
		if err != nil {
			return searchOp{}, err
		}
		d, err := zigParseSigned(tok, 64)
		return searchOp{Tag: searchOpScrollViewport, ScrollViewport: d}, err
	case "scroll_top":
		return searchOp{Tag: searchOpScrollTop}, nil
	case "scroll_active":
		return searchOp{Tag: searchOpScrollActive}, nil
	case "reset":
		return searchOp{Tag: searchOpReset}, nil
	case "open":
		return parseSearchOpen(it)
	case "close":
		return searchOp{Tag: searchOpClose}, nil
	case "update":
		return searchOp{Tag: searchOpUpdate}, nil
	case "next":
		return searchOp{Tag: searchOpNext}, nil
	case "drain":
		limit := uint64(100000)
		if tok, ok := it.next(); ok {
			var err error
			if limit, err = zigParseUint(tok, 64); err != nil {
				return searchOp{}, err
			}
		}
		return searchOp{Tag: searchOpDrain, Drain: uint(limit)}, nil
	case "feed":
		return searchOp{Tag: searchOpFeed}, nil
	case "tick":
		return searchOp{Tag: searchOpTick}, nil
	case "search_all":
		return searchOp{Tag: searchOpSearchAll}, nil
	case "select_next":
		return searchOp{Tag: searchOpSelectNext}, nil
	case "select_prev":
		return searchOp{Tag: searchOpSelectPrev}, nil
	case "update_active":
		return searchOp{Tag: searchOpUpdateActive}, nil
	case "matches":
		return searchOp{Tag: searchOpMatches}, nil
	case "selected":
		return searchOp{Tag: searchOpSelected}, nil
	case "append":
		v, err := searchParseUints(it, 64)
		if err != nil {
			return searchOp{}, err
		}
		return searchOp{Tag: searchOpAppend, Append: uint(v[0])}, nil
	case "append_all":
		return searchOp{Tag: searchOpAppendAll}, nil
	case "append_all_rev":
		return searchOp{Tag: searchOpAppendAllRev}, nil
	case "clear":
		return searchOp{Tag: searchOpClear}, nil
	case "viewport_reset":
		return searchOp{Tag: searchOpViewportReset}, nil
	case "viewport_dirty":
		tok, err := searchNextToken(it)
		if err != nil {
			return searchOp{}, err
		}
		d, err := parseOptBool(tok)
		return searchOp{Tag: searchOpViewportDirty, ViewportDirty: d}, err
	case "unchecked_only":
		reason, ok := it.next()
		if !ok {
			reason = "underflow"
		}
		return searchOp{Tag: searchOpUncheckedOnly, Reason: reason}, nil
	case "no_result":
		reason, ok := it.next()
		if !ok {
			reason = "unknown"
		}
		return searchOp{Tag: searchOpNoResult, Reason: reason}, nil
	}
	return searchOp{}, errReplayBadCase
}

// Mirrors harness/src/search.zig:275 parseSuite, arm "open"
func parseSearchOpen(it *replayTokens) (searchOp, error) {
	kindName, err := searchNextToken(it)
	if err != nil {
		return searchOp{}, err
	}
	kind := -1
	for i, name := range searchKindNames {
		if name == kindName {
			kind = i
		}
	}
	if kind < 0 {
		return searchOp{}, errReplayBadCase
	}
	tok, err := searchNextToken(it)
	if err != nil {
		return searchOp{}, err
	}
	needle, err := hexBytes(tok)
	if err != nil {
		return searchOp{}, err
	}
	o := searchOpen{kind: searchKind(kind), needle: needle}
	for {
		opt, ok := it.next()
		if !ok {
			break
		}
		k, v, found := strings.Cut(opt, "=")
		if !found {
			return searchOp{}, errReplayBadCase
		}
		switch k {
		case "screen":
			switch v {
			case "primary":
				o.screen = ScreenSetKeyPrimary
			case "alternate":
				o.screen = ScreenSetKeyAlternate
			default:
				return searchOp{}, errReplayBadCase
			}
		case "start":
			if v == "last" {
				o.start = None[uint]()
			} else {
				n, err := zigParseUint(v, 64)
				if err != nil {
					return searchOp{}, err
				}
				o.start = Some(uint(n))
			}
		case "dir":
			if v == "reverse" {
				o.reverse = true
			} else if v != "forward" {
				return searchOp{}, errReplayBadCase
			}
		case "dirty":
			if o.dirty, err = parseOptBool(v); err != nil {
				return searchOp{}, err
			}
		default:
			return searchOp{}, errReplayBadCase
		}
	}
	return searchOp{Tag: searchOpOpen, Open: o}, nil
}

// ---------------------------------------------------------------------------
// The records of the dump: the data of a layout, hl, win and st line, and
// their text (see the file header).

// searchLayoutRec is a `layout` line.
type searchLayoutRec struct {
	name                   string
	pages                  uint
	totalRows              uint
	serialMin              uint64
	nextSerial             uint64
	viewport               string
	viewportTop, activeTop Optional[uint32]
	cols, rows             uint16
	pageRows               []uint16
	serials                []uint64
}

func searchJoin[T any](list []T) string {
	if len(list) == 0 {
		return "-"
	}
	parts := make([]string, len(list))
	for i, x := range list {
		parts[i] = fmt.Sprint(x)
	}
	return strings.Join(parts, ",")
}

func searchOptY(y Optional[uint32]) string {
	if !y.Valid {
		return "none"
	}
	return strconv.FormatUint(uint64(y.Value), 10)
}

func (r searchLayoutRec) format() string {
	return fmt.Sprintf("layout %s pages=%d total_rows=%d serial_min=%d next_serial=%d viewport=%s viewport_top=%s active_top=%s cols=%d rows=%d page_rows=%s serials=%s\n",
		r.name, r.pages, r.totalRows, r.serialMin, r.nextSerial, r.viewport,
		searchOptY(r.viewportTop), searchOptY(r.activeTop), r.cols, r.rows,
		searchJoin(r.pageRows), searchJoin(r.serials))
}

// searchChunkRec is one `<page>:<start>-<end>:<serial>` of an hl line.
type searchChunkRec struct {
	page   Optional[uint] // the node's index in the list, none if it is not in it
	start  uint16
	end    uint16
	serial uint64
}

// searchHlRec is an `hl` line.
type searchHlRec struct {
	label      string
	topX, botX uint16
	tl, br     Optional[PointCoordinate]
	chunks     []searchChunkRec
}

func searchOptPoint(p Optional[PointCoordinate]) string {
	if !p.Valid {
		return "none"
	}
	return fmt.Sprintf("%d,%d", p.Value.x, p.Value.y)
}

func (r searchHlRec) format() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "hl %s top_x=%d bot_x=%d tl=%s br=%s chunks=%d", r.label, r.topX, r.botX,
		searchOptPoint(r.tl), searchOptPoint(r.br), len(r.chunks))
	for _, c := range r.chunks {
		sb.WriteString(" ")
		if c.page.Valid {
			fmt.Fprintf(&sb, "%d", c.page.Value)
		} else {
			sb.WriteString("-")
		}
		fmt.Fprintf(&sb, ":%d-%d:%d", c.start, c.end, c.serial)
	}
	sb.WriteString("\n")
	return sb.String()
}

// searchWinRec is the content of a `win` line, and of a window summary of an
// `st` line.
type searchWinRec struct {
	dataLen    uint
	dataOffset uint
	// win.meta.Len(): the number written as `metas=N`, as the Zig writes it
	// (it is NOT the length of metas, so that a wrong CircBuf.Len() with a
	// right iterator is a mismatch).
	metaLen uint
	metas   []uint // the cell_map length of every meta, from the iterator
}

// Mirrors harness/src/search.zig:520 Ctx.winSummary
//
// `{d}{c}{d}{c}{d}`: the bytes in data, data_offset and the number of metas.
func (r searchWinRec) summary(sep byte) string {
	return fmt.Sprintf("%d%c%d%c%d", r.dataLen, sep, r.dataOffset, sep, r.metaLen)
}

func (r searchWinRec) format() string {
	s := fmt.Sprintf("win data_len=%d data_offset=%d metas=%d", r.dataLen, r.dataOffset, r.metaLen)
	for i, m := range r.metas {
		if i == 0 {
			s += ":"
		} else {
			s += ","
		}
		s += strconv.FormatUint(uint64(m), 10)
	}
	return s + "\n"
}

// searchStateRec is an `st` line.
type searchStateRec struct {
	state   string
	total   uint
	active  uint
	history uint
	sel     Optional[uint]
	hist    bool
	actWin  searchWinRec
	histWin Optional[searchWinRec]
}

func (r searchStateRec) format() string {
	sel := "none"
	if r.sel.Valid {
		sel = strconv.FormatUint(uint64(r.sel.Value), 10)
	}
	s := fmt.Sprintf("st state=%s total=%d active=%d history=%d sel=%s hist=%d act_win=%s hist_win=",
		r.state, r.total, r.active, r.history, sel, boolInt(r.hist), r.actWin.summary('/'))
	if r.histWin.Valid {
		s += r.histWin.Value.summary('/')
	} else {
		s += "-"
	}
	return s + "\n"
}

// ---------------------------------------------------------------------------
// The context: search.zig Searcher and Ctx.

type searchSearcherTag uint8

const (
	searchSearcherNone searchSearcherTag = iota
	searchSearcherScreen
	searchSearcherActive
	searchSearcherViewport
	searchSearcherPagelist
	searchSearcherWindow
)

// Mirrors harness/src/search.zig:343 Searcher
//
// A union(enum): only the pointer named by Tag is set.
type searchSearcher struct {
	Tag      searchSearcherTag
	screen   *SearchScreen
	active   *SearchActive
	viewport *SearchViewport
	pagelist *SearchPagelist
	window   *SearchSlidingWindow
}

// Mirrors harness/src/search.zig:352 Ctx
type searchCtx struct {
	w        *bytes.Buffer
	term     *Terminal
	stream   *StreamReadonlyStream
	searcher searchSearcher
	// The screen whose page list the open searcher works on.
	target ScreenSetKey
}

// Mirrors harness/src/search.zig:361 Ctx.closeSearcher
func (self *searchCtx) closeSearcher() {
	switch self.searcher.Tag {
	case searchSearcherNone:
	case searchSearcherScreen:
		self.searcher.screen.Deinit()
	case searchSearcherActive:
		self.searcher.active.Deinit()
	case searchSearcherViewport:
		self.searcher.viewport.Deinit()
	case searchSearcherPagelist:
		self.searcher.pagelist.Deinit()
	case searchSearcherWindow:
		self.searcher.window.Deinit()
	}
	self.searcher = searchSearcher{Tag: searchSearcherNone}
}

// Mirrors harness/src/search.zig:373 Ctx.closeTerminal
func (self *searchCtx) closeTerminal() {
	self.closeSearcher()
	if self.stream != nil {
		self.stream.Deinit()
	}
	self.stream = nil
	if self.term != nil {
		self.term.Deinit()
	}
	self.term = nil
}

// Mirrors harness/src/search.zig:384 Ctx.ensureTerminal
func (self *searchCtx) ensureTerminal() (*Terminal, error) {
	if self.term != nil {
		return self.term, nil
	}
	if err := self.initTerminal(80, 24, 0); err != nil {
		return nil, err
	}
	return self.term, nil
}

// Mirrors harness/src/search.zig:390 Ctx.initTerminal
//
// The terminal has the production page capacity (the replayer never calls
// testStdCapacity) and the Zig defaults of Terminal.Options but for the three
// given.
func (self *searchCtx) initTerminal(cols, rows uint16, sb uint64) error {
	self.closeTerminal()
	opts := terminalOptionsDefault(cols, rows)
	opts.max_scrollback = sb
	t := new(Terminal)
	var err error
	*t, err = TerminalInit(opts)
	if err != nil {
		return err
	}
	self.term = t
	stream := t.VtStream()
	self.stream = &stream
	return nil
}

// Mirrors harness/src/search.zig:399 Ctx.pages
func (self *searchCtx) pages() *PageList {
	t := self.term
	if t == nil {
		return nil
	}
	s := t.screens.Get(self.target)
	if s == nil {
		return nil
	}
	return &s.pages
}

// Mirrors harness/src/search.zig:405 Ctx.pageIndex
func searchPageIndex(list *PageList, node *PageListNode) Optional[uint] {
	if list == nil {
		return None[uint]()
	}
	var i uint
	for cur := list.pages.first; cur != nil; cur = cur.next {
		if cur == node {
			return Some(i)
		}
		i++
	}
	return None[uint]()
}

// Mirrors harness/src/search.zig:416 Ctx.nodeAt
func searchNodeAt(list *PageList, idx uint) *PageListNode {
	var i uint
	for cur := list.pages.first; cur != nil; cur = cur.next {
		if i == idx {
			return cur
		}
		i++
	}
	return nil
}

// Mirrors harness/src/search.zig:426 Ctx.winOf
func (self *searchCtx) winOf() *SearchSlidingWindow {
	switch self.searcher.Tag {
	case searchSearcherActive:
		return &self.searcher.active.window
	case searchSearcherViewport:
		return &self.searcher.viewport.window
	case searchSearcherPagelist:
		return &self.searcher.pagelist.window
	case searchSearcherWindow:
		return self.searcher.window
	}
	return nil
}

// ---- records ----

// Mirrors harness/src/search.zig:438 Ctx.printLayoutOf
func (self *searchCtx) printLayoutOf(name string, screen *Screen) {
	p := &screen.pages
	rec := searchLayoutRec{name: name, serialMin: p.page_serial_min, nextSerial: p.page_serial, viewport: p.viewport.Tag.String()}
	for node := p.pages.first; node != nil; node = node.next {
		rec.pages++
		rec.totalRows += uint(node.data.size.rows)
		rec.pageRows = append(rec.pageRows, uint16(node.data.size.rows))
		rec.serials = append(rec.serials, node.serial)
	}
	vtTop := p.PointFromPin(PointTagScreen, p.GetTopLeft(PointTagViewport))
	acTop := p.PointFromPin(PointTagScreen, p.GetTopLeft(PointTagActive))
	if vtTop.Valid {
		rec.viewportTop = Some(vtTop.Value.Screen.y)
	}
	if acTop.Valid {
		rec.activeTop = Some(acTop.Value.Screen.y)
	}
	rec.cols, rec.rows = uint16(p.cols), uint16(p.rows)
	self.w.WriteString(rec.format())
}

// Mirrors harness/src/search.zig:478 Ctx.printLayout
//
// The page list of the primary screen, then of the alternate screen if it
// exists.
func (self *searchCtx) printLayout() {
	t := self.term
	if t == nil {
		return
	}
	if s := t.screens.Get(ScreenSetKeyPrimary); s != nil {
		self.printLayoutOf("primary", s)
	}
	if s := t.screens.Get(ScreenSetKeyAlternate); s != nil {
		self.printLayoutOf("alternate", s)
	}
}

// Mirrors harness/src/search.zig:484 Ctx.printHl
func (self *searchCtx) printHl(label string, hl HighlightFlattened) {
	list := self.pages()
	rec := searchHlRec{label: label, topX: uint16(hl.top_x), botX: uint16(hl.bot_x)}
	k := len(hl.chunks)
	if k == 0 {
		self.w.WriteString(rec.format())
		return
	}
	// The start pin is (top_x, starts[0]) in nodes[0], the end pin
	// (bot_x, ends[k-1] - 1) in nodes[k-1].
	if list != nil {
		pt := list.PointFromPin(PointTagScreen, PageListPin{node: hl.chunks[0].node, x: hl.top_x, y: hl.chunks[0].start})
		if pt.Valid {
			rec.tl = Some(pt.Value.Screen)
		}
	}
	if list != nil && hl.chunks[k-1].end > 0 {
		pt := list.PointFromPin(PointTagScreen, PageListPin{node: hl.chunks[k-1].node, x: hl.bot_x, y: hl.chunks[k-1].end - 1})
		if pt.Valid {
			rec.br = Some(pt.Value.Screen)
		}
	}
	for _, c := range hl.chunks {
		rec.chunks = append(rec.chunks, searchChunkRec{page: searchPageIndex(list, c.node), start: uint16(c.start), end: uint16(c.end), serial: c.serial})
	}
	self.w.WriteString(rec.format())
}

// searchWinRecOf is the window's numbers: bytes in data, data_offset, the
// length of meta and the cell_map length of every meta.
func searchWinRecOf(win *SearchSlidingWindow) searchWinRec {
	rec := searchWinRec{dataLen: win.data.Len(), dataOffset: win.data_offset, metaLen: win.meta.Len()}
	it := win.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	for m := it.Next(); m != nil; m = it.Next() {
		rec.metas = append(rec.metas, uint(len(m.cell_map)))
	}
	return rec
}

// Mirrors harness/src/search.zig:524 Ctx.printWin
func (self *searchCtx) printWin() {
	win := self.winOf()
	if win == nil {
		return
	}
	self.w.WriteString(searchWinRecOf(win).format())
}

// Mirrors harness/src/search.zig:538 Ctx.printState
func (self *searchCtx) printState() {
	if self.searcher.Tag != searchSearcherScreen {
		return
	}
	s := self.searcher.screen
	rec := searchStateRec{
		state: s.state.String(), total: s.MatchesLen(),
		active: uint(len(s.active_results)), history: uint(len(s.history_results)),
		hist: s.history.Valid, actWin: searchWinRecOf(&s.active.window),
	}
	if s.selected.Valid {
		rec.sel = Some(s.selected.Value.idx)
	}
	if s.history.Valid {
		rec.histWin = Some(searchWinRecOf(&s.history.Value.searcher.window))
	}
	self.w.WriteString(rec.format())
}

// Mirrors harness/src/search.zig:555 Ctx.ret
func (self *searchCtx) ret(format string, args ...any) {
	fmt.Fprintf(self.w, "ret "+format+"\n", args...)
}

// Mirrors harness/src/search.zig:559 Ctx.retErr
func (self *searchCtx) retErr(err error) {
	fmt.Fprintf(self.w, "ret err %s\n", zigErrorName(err))
}

// Mirrors harness/src/search.zig:563 Ctx.notApplicable
func (self *searchCtx) notApplicable() {
	self.w.WriteString("ret err NotApplicable\n")
}

// ---- ops ----

// Mirrors harness/src/search.zig:569 Ctx.echo
func (self *searchCtx) echo(n int, op searchOp) {
	w := self.w
	fmt.Fprintf(w, "op %d ", n)
	switch op.Tag {
	case searchOpInit:
		v := op.Init
		fmt.Fprintf(w, "init %d %d %d\n", v.cols, v.rows, v.max_scrollback)
	case searchOpWrite:
		fmt.Fprintf(w, "write len=%d\n", len(op.Write))
	case searchOpRepeat:
		fmt.Fprintf(w, "repeat %d len=%d\n", op.Repeat.n, len(op.Repeat.bytes))
	case searchOpResize:
		fmt.Fprintf(w, "resize %d %d\n", op.Resize.cols, op.Resize.rows)
	case searchOpScrollViewport:
		fmt.Fprintf(w, "scroll_viewport %d\n", op.ScrollViewport)
	case searchOpOpen:
		o := op.Open
		fmt.Fprintf(w, "open %s needle_len=%d", o.kind, len(o.needle))
		fmt.Fprintf(w, " screen=%s", searchScreenNames[o.screen])
		if o.kind == searchKindPagelist {
			if o.start.Valid {
				fmt.Fprintf(w, " start=%d", o.start.Value)
			} else {
				w.WriteString(" start=last")
			}
		}
		if o.kind == searchKindWindow {
			dir := "forward"
			if o.reverse {
				dir = "reverse"
			}
			fmt.Fprintf(w, " dir=%s", dir)
		}
		if o.kind == searchKindViewport {
			if o.dirty.Valid {
				fmt.Fprintf(w, " dirty=%t", o.dirty.Value)
			} else {
				w.WriteString(" dirty=null")
			}
		}
		w.WriteString("\n")
	case searchOpDrain:
		fmt.Fprintf(w, "drain %d\n", op.Drain)
	case searchOpAppend:
		fmt.Fprintf(w, "append %d\n", op.Append)
	case searchOpViewportDirty:
		if op.ViewportDirty.Valid {
			fmt.Fprintf(w, "viewport_dirty %t\n", op.ViewportDirty.Value)
		} else {
			w.WriteString("viewport_dirty null\n")
		}
	default:
		fmt.Fprintf(w, "%s\n", op.Tag)
	}
}

// Mirrors harness/src/search.zig:599 Ctx.run
//
// The error is the creation of the terminal failing, which the Zig `try`
// propagates out of runCase.
func (self *searchCtx) run(n int, op searchOp) error {
	self.echo(n, op)
	switch op.Tag {
	case searchOpInit:
		return self.runInit(op.Init)
	case searchOpWrite:
		return self.runWrite(op.Write)
	case searchOpRepeat:
		return self.runRepeat(op.Repeat)
	case searchOpResize:
		return self.runResize(op.Resize)
	case searchOpScrollViewport:
		return self.runScrollViewport(op.ScrollViewport)
	case searchOpScrollTop:
		return self.runScrollTop()
	case searchOpScrollActive:
		return self.runScrollActive()
	case searchOpReset:
		return self.runReset()
	case searchOpOpen:
		return self.open(op.Open)
	case searchOpClose:
		self.runClose()
	case searchOpUpdate:
		self.runUpdate()
	case searchOpNext:
		self.runNext()
	case searchOpDrain:
		self.runDrain(op.Drain)
	case searchOpFeed:
		self.runFeed()
	case searchOpTick:
		self.runTick()
	case searchOpSearchAll:
		self.runSearchAll()
	case searchOpSelectNext:
		self.runSelect(SearchScreenSelectNext)
	case searchOpSelectPrev:
		self.runSelect(SearchScreenSelectPrev)
	case searchOpUpdateActive:
		self.runUpdateActive()
	case searchOpMatches:
		self.runMatches()
	case searchOpSelected:
		self.runSelected()
	case searchOpAppend:
		self.runAppend(op.Append)
	case searchOpAppendAll:
		self.runAppendAll(true)
	case searchOpAppendAllRev:
		self.runAppendAll(false)
	case searchOpClear:
		self.runClear()
	case searchOpViewportReset:
		self.runViewportReset()
	case searchOpViewportDirty:
		self.runViewportDirty(op.ViewportDirty)
	case searchOpUncheckedOnly, searchOpNoResult:
	}
	return nil
}

// Mirrors harness/src/search.zig:602 Ctx.run, arm .init
func (self *searchCtx) runInit(v replayOpInitArgs) error {
	if err := self.initTerminal(v.cols, v.rows, v.max_scrollback); err != nil {
		return err
	}
	self.printLayout()
	return nil
}

// Mirrors harness/src/search.zig:606 Ctx.run, arm .write
func (self *searchCtx) runWrite(bytes_ []uint8) error {
	if _, err := self.ensureTerminal(); err != nil {
		return err
	}
	if err := self.stream.NextSlice(bytes_); err != nil {
		self.retErr(err)
	}
	self.printLayout()
	return nil
}

// Mirrors harness/src/search.zig:611 Ctx.run, arm .repeat
func (self *searchCtx) runRepeat(v searchOpRepeatArgs) error {
	if _, err := self.ensureTerminal(); err != nil {
		return err
	}
	for i := uint64(0); i < v.n; i++ {
		if err := self.stream.NextSlice(v.bytes); err != nil {
			self.retErr(err)
		}
	}
	self.printLayout()
	return nil
}

// Mirrors harness/src/search.zig:616 Ctx.run, arm .resize
func (self *searchCtx) runResize(v replayOpResizeArgs) error {
	t, err := self.ensureTerminal()
	if err != nil {
		return err
	}
	if err := t.Resize(v.cols, v.rows); err != nil {
		self.retErr(err)
	}
	self.printLayout()
	return nil
}

// Mirrors harness/src/search.zig:621 Ctx.run, arm .scroll_viewport
func (self *searchCtx) runScrollViewport(d int64) error {
	t, err := self.ensureTerminal()
	if err != nil {
		return err
	}
	t.screens.active.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: d})
	self.printLayout()
	return nil
}

// Mirrors harness/src/search.zig:626 Ctx.run, arm .scroll_top
func (self *searchCtx) runScrollTop() error {
	t, err := self.ensureTerminal()
	if err != nil {
		return err
	}
	t.screens.active.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	self.printLayout()
	return nil
}

// Mirrors harness/src/search.zig:631 Ctx.run, arm .scroll_active
func (self *searchCtx) runScrollActive() error {
	t, err := self.ensureTerminal()
	if err != nil {
		return err
	}
	t.screens.active.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	self.printLayout()
	return nil
}

// Mirrors harness/src/search.zig:636 Ctx.run, arm .reset
func (self *searchCtx) runReset() error {
	t, err := self.ensureTerminal()
	if err != nil {
		return err
	}
	t.FullReset()
	self.printLayout()
	return nil
}

// Mirrors harness/src/search.zig:642 Ctx.run, arm .close
func (self *searchCtx) runClose() {
	self.closeSearcher()
	self.ret("ok")
}

// Mirrors harness/src/search.zig:646 Ctx.run, arm .update
func (self *searchCtx) runUpdate() {
	switch self.searcher.Tag {
	case searchSearcherActive:
		list := self.pages()
		node, err := self.searcher.active.Update(list)
		if err == nil {
			if node != nil {
				if i := searchPageIndex(list, node); i.Valid {
					self.ret("node %d", i.Value)
				} else {
					self.ret("node -")
				}
			} else {
				self.ret("node null")
			}
		} else {
			self.retErr(err)
		}
		self.printWin()
	case searchSearcherViewport:
		list := self.pages()
		if changed, err := self.searcher.viewport.Update(list); err == nil {
			self.ret("bool %t", changed)
		} else {
			self.retErr(err)
		}
		self.printWin()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:663 Ctx.run, arm .next
func (self *searchCtx) runNext() {
	self.next()
	self.printWin()
}

// Mirrors harness/src/search.zig:667 Ctx.run, arm .drain
func (self *searchCtx) runDrain(limit uint) {
	var count uint
	hitLimit := false
	for {
		if count >= limit {
			hitLimit = true
			break
		}
		hl := self.nextHl()
		if !hl.Valid {
			break
		}
		self.printHl("next", hl.Value)
		count++
	}
	if hitLimit {
		self.ret("drain_limit count=%d", count)
	} else {
		self.ret("drain count=%d", count)
	}
	self.printWin()
}

// Mirrors harness/src/search.zig:682 Ctx.run, arm .feed
func (self *searchCtx) runFeed() {
	switch self.searcher.Tag {
	case searchSearcherPagelist:
		if fed, err := self.searcher.pagelist.Feed(); err == nil {
			self.ret("bool %t", fed)
		} else {
			self.retErr(err)
		}
		self.printWin()
	case searchSearcherScreen:
		if err := self.searcher.screen.Feed(); err == nil {
			self.ret("ok")
		} else {
			self.retErr(err)
		}
		self.printState()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:693 Ctx.run, arm .tick
func (self *searchCtx) runTick() {
	switch self.searcher.Tag {
	case searchSearcherScreen:
		if err := self.searcher.screen.Tick(); err == nil {
			self.ret("tick ok")
		} else {
			self.ret("tick %s", zigErrorName(err))
		}
		self.printState()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:700 Ctx.run, arm .search_all
func (self *searchCtx) runSearchAll() {
	switch self.searcher.Tag {
	case searchSearcherScreen:
		if err := self.searcher.screen.SearchAll(); err == nil {
			self.ret("ok")
		} else {
			self.retErr(err)
		}
		self.printState()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:707 Ctx.run, arm .select_next, .select_prev
func (self *searchCtx) runSelect(to SearchScreenSelect) {
	switch self.searcher.Tag {
	case searchSearcherScreen:
		s := self.searcher.screen
		if moved, err := s.Select(to); err == nil {
			self.ret("bool %t", moved)
		} else {
			self.retErr(err)
		}
		self.printState()
		if hl := s.SelectedMatch(); hl.Valid {
			self.printHl("selected", hl.Value)
		}
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:716 Ctx.run, arm .update_active
func (self *searchCtx) runUpdateActive() {
	switch self.searcher.Tag {
	case searchSearcherScreen:
		if err := self.searcher.screen.ReloadActive(); err == nil {
			self.ret("ok")
		} else {
			self.retErr(err)
		}
		self.printState()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:723 Ctx.run, arm .matches
func (self *searchCtx) runMatches() {
	switch self.searcher.Tag {
	case searchSearcherScreen:
		if ms, err := self.searcher.screen.Matches(); err == nil {
			self.ret("matches n=%d", len(ms))
			for i, hl := range ms {
				self.printHl(fmt.Sprintf("match %d", i), hl)
			}
		} else {
			self.retErr(err)
		}
		self.printState()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:738 Ctx.run, arm .selected
func (self *searchCtx) runSelected() {
	switch self.searcher.Tag {
	case searchSearcherScreen:
		if hl := self.searcher.screen.SelectedMatch(); hl.Valid {
			self.ret("selected hit")
			self.printHl("selected", hl.Value)
		} else {
			self.ret("selected null")
		}
		self.printState()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:748 Ctx.run, arm .append
func (self *searchCtx) runAppend(idx uint) {
	switch self.searcher.Tag {
	case searchSearcherWindow:
		list := self.pages()
		if node := searchNodeAt(list, idx); node != nil {
			if n, err := self.searcher.window.Append(node); err == nil {
				self.ret("bytes %d", n)
			} else {
				self.retErr(err)
			}
		} else {
			self.ret("err NoSuchPage")
		}
		self.printWin()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:758 Ctx.run, arm .append_all, .append_all_rev
//
// forward is `op == .append_all`.
func (self *searchCtx) runAppendAll(forward bool) {
	switch self.searcher.Tag {
	case searchSearcherWindow:
		list := self.pages()
		var total uint
		var failed error
		node := list.pages.last
		if forward {
			node = list.pages.first
		}
		for node != nil {
			cur := node
			n, err := self.searcher.window.Append(cur)
			if err != nil {
				failed = err
				break
			}
			total += n
			if forward {
				node = cur.next
			} else {
				node = cur.prev
			}
		}
		if failed != nil {
			self.retErr(failed)
		} else {
			self.ret("bytes %d", total)
		}
		self.printWin()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:775 Ctx.run, arm .clear
func (self *searchCtx) runClear() {
	if self.searcher.Tag == searchSearcherScreen {
		self.notApplicable()
	} else if win := self.winOf(); win != nil {
		win.ClearAndRetainCapacity()
		self.ret("ok")
		self.printWin()
	} else {
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:784 Ctx.run, arm .viewport_reset
func (self *searchCtx) runViewportReset() {
	switch self.searcher.Tag {
	case searchSearcherViewport:
		self.searcher.viewport.Reset()
		self.ret("ok")
		self.printWin()
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:792 Ctx.run, arm .viewport_dirty
func (self *searchCtx) runViewportDirty(d Optional[bool]) {
	switch self.searcher.Tag {
	case searchSearcherViewport:
		self.searcher.viewport.active_dirty = d
		self.ret("ok")
	default:
		self.notApplicable()
	}
}

// Mirrors harness/src/search.zig:803 Ctx.open
func (self *searchCtx) open(o searchOpen) error {
	t, err := self.ensureTerminal()
	if err != nil {
		return err
	}
	self.closeSearcher()
	self.target = o.screen
	screen := t.screens.Get(o.screen)
	if screen == nil {
		self.ret("err NoSuchScreen")
		return nil
	}
	list := &screen.pages
	switch o.kind {
	case searchKindScreen:
		if s, err := SearchScreenInit(screen, o.needle); err == nil {
			self.searcher = searchSearcher{Tag: searchSearcherScreen, screen: &s}
			self.ret("ok")
		} else {
			self.retErr(err)
		}
	case searchKindActive:
		if s, err := SearchActiveInit(o.needle); err == nil {
			self.searcher = searchSearcher{Tag: searchSearcherActive, active: &s}
			self.ret("ok")
		} else {
			self.retErr(err)
		}
	case searchKindViewport:
		if s, err := SearchViewportInit(o.needle); err == nil {
			self.searcher = searchSearcher{Tag: searchSearcherViewport, viewport: &s}
			self.searcher.viewport.active_dirty = o.dirty
			self.ret("ok")
		} else {
			self.retErr(err)
		}
	case searchKindPagelist:
		start := list.pages.last
		if o.start.Valid {
			start = searchNodeAt(list, o.start.Value)
		}
		if start != nil {
			if s, err := SearchPagelistInit(o.needle, list, start); err == nil {
				self.searcher = searchSearcher{Tag: searchSearcherPagelist, pagelist: &s}
				self.ret("ok")
			} else {
				self.retErr(err)
			}
		} else {
			self.ret("err NoSuchPage")
		}
	case searchKindWindow:
		dir := searchSlidingWindowDirectionForward
		if o.reverse {
			dir = searchSlidingWindowDirectionReverse
		}
		if s, err := SearchSlidingWindowInit(dir, o.needle); err == nil {
			self.searcher = searchSearcher{Tag: searchSearcherWindow, window: &s}
			self.ret("ok")
		} else {
			self.retErr(err)
		}
	}
	self.printLayout()
	self.printWin()
	self.printState()
	return nil
}

// Mirrors harness/src/search.zig:856 Ctx.nextHl
//
// One next() call on the open searcher. The returned highlight aliases the
// window's chunk buffer: print it before the next call.
func (self *searchCtx) nextHl() Optional[HighlightFlattened] {
	switch self.searcher.Tag {
	case searchSearcherActive:
		return self.searcher.active.Next()
	case searchSearcherViewport:
		return self.searcher.viewport.Next()
	case searchSearcherPagelist:
		return self.searcher.pagelist.Next()
	case searchSearcherWindow:
		return self.searcher.window.Next()
	}
	return None[HighlightFlattened]()
}

// Mirrors harness/src/search.zig:866 Ctx.next
func (self *searchCtx) next() {
	switch self.searcher.Tag {
	case searchSearcherNone, searchSearcherScreen:
		self.notApplicable()
	default:
		if hl := self.nextHl(); hl.Valid {
			self.ret("next hit")
			self.printHl("next", hl.Value)
		} else {
			self.ret("next null")
		}
	}
}

// Mirrors harness/src/search.zig:880 runCase
//
// Run one case and append its trace to `w`. The error is the creation of the
// terminal failing (see Ctx.run).
func searchRunCase(c searchCase, w *bytes.Buffer) error {
	// Markers.
	var unchecked, noResult Optional[string]
	for _, op := range c.ops {
		switch op.Tag {
		case searchOpUncheckedOnly:
			unchecked = Some(op.Reason)
		case searchOpNoResult:
			noResult = Some(op.Reason)
		}
	}
	if noResult.Valid {
		fmt.Fprintf(w, "!no-recorded-result %s\n", noResult.Value)
		return nil
	}
	if unchecked.Valid {
		mode := searchRuns
		runs := mode == "unchecked" || (mode == "candidate" && unchecked.Value == "underflow")
		if !runs {
			fmt.Fprintf(w, "!needs-unchecked-release reason=%s\n", unchecked.Value)
			return nil
		}
		fmt.Fprintf(w, "!recorded-from-unchecked-release reason=%s\n", unchecked.Value)
	}

	ctx := &searchCtx{w: w}
	defer ctx.closeTerminal()
	n := 0
	for _, op := range c.ops {
		switch op.Tag {
		case searchOpUncheckedOnly, searchOpNoResult:
			continue
		}
		n++
		if err := ctx.run(n, op); err != nil {
			return err
		}
	}
	return nil
}

// Mirrors harness/src/search.zig:921 produceSuite
//
// The suite is parsed already (parseSearchSuite); the result is the number of
// cases written.
func searchProduceSuite(cases []searchCase, w *bytes.Buffer) (int, error) {
	n := 0
	for _, c := range cases {
		fmt.Fprintf(w, "## case %s\n", c.name)
		if err := searchRunCase(c, w); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// searchCorpusProduce is searchProduceSuite for one case, as the text of one
// section: corpusProduce for the search area. A panic of the port or an error
// of the replay becomes a line of the text, so it is a mismatch like any other
// and the run goes on.
func searchCorpusProduce(c searchCase) (out []byte) {
	var buf bytes.Buffer
	defer func() {
		if r := recover(); r != nil {
			buf.Reset()
			fmt.Fprintf(&buf, "## case %s\n!panic: %v\n", c.name, r)
			out = buf.Bytes()
		}
	}()
	if _, err := searchProduceSuite([]searchCase{c}, &buf); err != nil {
		fmt.Fprintf(&buf, "!replay error=%s\n", err.Error())
	}
	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// The suite driver and the test function.

// searchMarker returns the reason of the marker op of the tag, if the case has
// one.
func searchMarker(c searchCase, tag searchOpTag) Optional[string] {
	for _, op := range c.ops {
		if op.Tag == tag {
			return Some(op.Reason)
		}
	}
	return None[string]()
}

// searchCheckManifest compares the number of cases of a suite with the cases=
// of golden/MANIFEST (corpusCheckManifest for a search suite).
func searchCheckManifest(t testing.TB, manifest map[string]corpusManifestEntry, suite string, parsed searchSuite) {
	t.Helper()
	key := searchArea.name + "/" + corpusGoldenName(suite, replayKindSearch)
	e, ok := manifest[key]
	switch {
	case !ok || e.unit != "cases":
		t.Errorf("%s is not in golden/MANIFEST with a case count", key)
	case e.count != len(parsed.cases):
		t.Errorf("%s: MANIFEST says cases=%d, the case file has %d", key, e.count, len(parsed.cases))
	}
}

// corpusReadSearchSuite parses cases/search/<suite>.case.
func corpusReadSearchSuite(t testing.TB, suite string) searchSuite {
	t.Helper()
	text := corpusReadAll(t, "cases/search/"+suite+".case")
	if !isSearchSuite(string(text)) {
		t.Fatalf("cases/search/%s.case is not a search suite (`kinds search`)", suite)
	}
	parsed, err := parseSearchSuite(string(text))
	if err != nil {
		t.Fatalf("bad search suite %s: %v", suite, err)
	}
	return parsed
}

// searchReplayable returns the indexes of the cases that are replayed: all but
// the ones marked no_result.
func searchReplayable(cases []searchCase) []int {
	var out []int
	for i, c := range cases {
		if !searchMarker(c, searchOpNoResult).Valid {
			out = append(out, i)
		}
	}
	return out
}

// searchSelect is the selection of the cases of a suite to run
// (selectCases): corpusSelect over the replayable cases below `avail`, plus the
// first case marked unchecked_only; the result is a sorted list of indexes into
// `cases`. With exhaustive, every replayable case that the filter wants.
func searchSelect(cfg corpusConfig, suite string, cases []searchCase, avail int) []int {
	var replay []int
	for _, i := range searchReplayable(cases) {
		if i < avail {
			replay = append(replay, i)
		}
	}
	if cfg.exhaustive() {
		var out []int
		for _, i := range replay {
			if cfg.wants(searchArea.name, suite, cases[i].name) {
				out = append(out, i)
			}
		}
		return out
	}
	var marked []int
	for k, i := range replay {
		if searchMarker(cases[i], searchOpUncheckedOnly).Valid {
			marked = append(marked, k)
			break
		}
	}
	var out []int
	for _, k := range corpusSelect(len(replay), cfg.sample, marked) {
		out = append(out, replay[k])
	}
	return out
}

// searchCompare is corpusCompare for a search trace: no line is skipped, and the
// label is the op that the first differing line of want belongs to.
func searchCompare(got, want []byte) (corpusDiff, bool) {
	d, equal := corpusCompare(got, want, nil)
	if equal {
		return d, true
	}
	wl := corpusSplitLines(want)
	for i := min(d.line, len(wl)) - 1; i >= 0; i-- {
		if f := strings.Fields(string(wl[i])); len(f) >= 2 && f[0] == "op" {
			d.label = "op " + f[1]
			break
		}
	}
	return d, false
}

// selectSearchCases chooses the cases of a suite to run: searchSelect with the
// window of the golden file under -short.
func (run *corpusRun) selectSearchCases(t testing.TB, suite string, cases []searchCase) []int {
	avail := len(cases)
	if run.cfg.short && !run.cfg.exhaustive() {
		s := run.openSections(t, suite, replayKindSearch)
		n, err := s.count()
		s.Close()
		if err != nil {
			t.Fatalf("%s: %v", corpusGoldenName(suite, replayKindSearch), err)
		}
		avail = min(avail, n)
	}
	return searchSelect(run.cfg, suite, cases, avail)
}

// replaySearchSuite runs the selected cases of a search suite, a subtest per
// case (replaySuite for the search area).
func (run *corpusRun) replaySearchSuite(t *testing.T, suite string) {
	parsed := corpusReadSearchSuite(t, suite)
	searchCheckManifest(t, run.manifest, suite, parsed)
	st := &corpusSuiteStats{cases: len(parsed.cases)}
	for _, c := range parsed.cases {
		if searchMarker(c, searchOpNoResult).Valid {
			st.noResult++
		}
	}
	selected := run.selectSearchCases(t, suite, parsed.cases)
	sections := run.openSections(t, suite, replayKindSearch)
	defer sections.Close()
	for _, idx := range selected {
		c := parsed.cases[idx]
		t.Run(c.name, func(t *testing.T) {
			st.run++
			run.ran.Add(1)
			run.replaySearchCase(t, st, suite, c, idx, sections)
		})
	}
	run.verifySections(t, suite, []replayKind{replayKindSearch}, []*corpusSections{sections})
	t.Log(searchSummary(suite, st))
}

// replaySearchCase produces one case and compares it with its recorded
// section (replayCase for the search area).
func (run *corpusRun) replaySearchCase(t *testing.T, st *corpusSuiteStats, suite string, c searchCase, idx int, sections *corpusSections) {
	t.Helper()
	name, want, err := sections.section(idx)
	if err != nil {
		t.Errorf("search/%s: section %d: %v", suite, idx, err)
		return
	}
	if name != c.name {
		t.Errorf("search/%s: section %d is %q, the case file has %q", suite, idx, name, c.name)
		return
	}
	if bytes.HasPrefix(want, []byte(corpusSectionPrefix+name+"\n!needs-unchecked-release")) {
		t.Errorf("search/%s case=%s: recorded by the safe recorder (!needs-unchecked-release)", suite, c.name)
		return
	}
	got := searchCorpusProduce(c)
	diff, equal := searchCompare(got, want)
	if equal {
		return
	}
	m := corpusMismatch{area: searchArea.name, suite: suite, caseName: c.name, kind: replayKindSearch, diff: diff}
	corpusReport(t, run, st, m)
	run.writeDiff(t, m, got, want)
}

// searchSummary is the SUITE line of a search suite: cases in the case file,
// cases run, the cases with no recorded result (never replayed), differences.
func searchSummary(suite string, st *corpusSuiteStats) string {
	line := fmt.Sprintf("SUITE search/%s cases=%d run=%d no_result=%d mismatch=%d",
		suite, st.cases, st.run, st.noResult, st.mismatches())
	return line
}

// TestCorpusReplaySearch replays the search area: a subtest per suite
// (parallel), in it a subtest per selected case (sequential).
func TestCorpusReplaySearch(t *testing.T) {
	if PageStdCapacity.grapheme_bytes != 8192 {
		t.Fatalf("PageStdCapacity.grapheme_bytes is %d: the corpus is recorded with the production capacity 8192", PageStdCapacity.grapheme_bytes)
	}
	run := newCorpusRun(t, searchArea)
	// Cleanup runs when the parallel subtests are done.
	t.Cleanup(func() { corpusFailIfEmpty(t, run.cfg, "replay of area search", int(run.ran.Load())) })
	for _, suite := range corpusSuiteNames(t, searchArea) {
		t.Run(suite, func(t *testing.T) {
			t.Parallel()
			run.replaySearchSuite(t, suite)
		})
	}
}
