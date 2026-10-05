package terminal

// The case runner of the corpus replayer: the Go side of the replay half of
// the recorder's harness/src/main.zig (runCase and its ops, feed, produce,
// produceSuite). One Go function per Zig function or op, same name in Go
// style, each with the Zig line it mirrors; the control flow follows the Zig.
//
// Differences from the Zig, forced by the language or by the replay:
//   - the writer is a *bytes.Buffer; a Zig `try` that can only fail on the
//     writer disappears, the other errors are returned;
//   - there is no allocator argument;
//   - `recorder_options.corrected` is the constant replayCorrected, true: the
//     corpus is recorded by the corrected recorder, so a case marked
//     expect_crash is replayed and its output starts with
//     "!recorded-from-corrected-upstream";
//   - a case marked transient_invariant is replayed the release way (the Go
//     Terminal.Resize defers the pin check) and PageList.verifyIntegrity, the
//     function that returns the error, is called on every present screen's
//     pages after each resize op and at the end; any error fails the case
//     (runCaseTransientInvariant sets the flag, verifyScreensIntegrity does
//     it). It is NOT assertIntegrity: upstream gates that behind
//     build_options.slow_runtime_safety, and a port that copies the gating
//     would make the check of the 951 transient cases a silent no-op. This is
//     the Go counterpart of the Debug recorder's "enforce when resize()
//     returns" (docs/replayer-design.md section 6);
//   - an error that Zig prints with @errorName (a feed or resize error, a key
//     encoder error) is printed by zigErrorName, which accepts only the bare
//     sentinels of the package (docs/mapping.md: the port never wraps
//     errors);
//   - the trace kinds (actions, actions_noalloc, parser: runTrace) are not
//     replayed yet (stage 2): produce returns errReplayKindNotRun for them.

import (
	"bytes"
	"errors"
	"fmt"
)

// replayCorrected is `recorder_options.corrected` (main.zig:332).
const replayCorrected = true

// errReplayKindNotRun is returned by produce for a kind that the replayer
// does not run yet.
var errReplayKindNotRun = errors.New("replay kind not implemented yet")

// Mirrors harness/src/main.zig:161 FeedMode
//
// How feed ops are delivered to the stream.
type feedMode struct {
	Tag feedModeTag

	// The feed op with index Op (counting feed ops only) is cut in two at
	// byte offset At (Zig: .split).
	Op uint
	At uint
}

type feedModeTag uint8

const (
	// One NextSlice per feed op, as written.
	feedModeWhole feedModeTag = iota
	// One NextSlice per byte.
	feedModeBytewise
	// One feed op is cut in two.
	feedModeSplit
)

// runCaseState is the set of locals of Zig's runCase that the ops share:
// t_opt, stream_opt, dumped and feed_idx, and the parameters mode and w.
// Nil is Zig's null for the two pointers.
type runCaseState struct {
	t      *Terminal
	stream *StreamReadonlyStream

	dumped    bool
	feedIdx   uint
	mode      feedMode
	w         *bytes.Buffer
	integrity bool // transient_invariant: verify the integrity (Go only)
}

// Mirrors harness/src/main.zig:211 runCase
//
// Run a case and append its dump(s) to `w`.
func runCase(ops []replayOp, mode feedMode, w *bytes.Buffer) error {
	st := &runCaseState{mode: mode, w: w}
	defer runCaseDeinitTerminal(st)

	// One stream for the lifetime of the terminal, so parser state carries
	// across feed ops exactly as it would across reads from a pty.
	defer runCaseDeinitStream(st)

	for _, op := range ops {
		if st.t == nil && op.Tag != replayOpInit {
			if err := runCaseTerminalInit(st, 80, 24, 10_000); err != nil {
				return err
			}
		}
		switch op.Tag {
		case replayOpInit:
			if err := runCaseInit(st, op.Init); err != nil {
				return err
			}
		case replayOpFeed:
			runCaseFeed(st, op.Feed)
		case replayOpResize:
			runCaseResize(st, op.Resize)
		case replayOpScrollTop:
			runCaseScrollTop(st)
		case replayOpScrollActive:
			runCaseScrollActive(st)
		case replayOpScrollDelta:
			runCaseScrollDelta(st, op.ScrollDelta)
		case replayOpDump:
			runCaseDump(st, op.Dump)
		case replayOpExpectCrash:
			runCaseExpectCrash(st)
		case replayOpUpstreamInvalid:
			runCaseUpstreamInvalid(st)
		case replayOpTransientInvariant:
			runCaseTransientInvariant(st)
		}
	}

	if !st.dumped {
		if st.t == nil {
			if err := runCaseTerminalInit(st, 80, 24, 10_000); err != nil {
				return err
			}
		}
		dumpTerminal(w, st.t, "end")
	}
	if st.integrity {
		return verifyScreensIntegrity(st.t)
	}
	return nil
}

// runCaseTerminalInit is the pair of lines that create the terminal and its
// stream in runCase (main.zig:230-231, :239-244, :279-280):
// `Terminal.init(gpa, .{ .cols, .rows, .max_scrollback })` and
// `t_opt.?.vtStream()`. The other options have their Zig defaults.
func runCaseTerminalInit(st *runCaseState, cols, rows uint16, maxScrollback uint64) error {
	opts := terminalOptionsDefault(cols, rows)
	opts.max_scrollback = maxScrollback
	t := new(Terminal)
	var err error
	*t, err = TerminalInit(opts)
	if err != nil {
		return err
	}
	st.t = t
	stream := t.VtStream()
	st.stream = &stream
	return nil
}

// runCaseDeinitStream is `defer if (stream_opt) |*s| s.deinit()` and the
// first line of the init arm (main.zig:223, :235).
func runCaseDeinitStream(st *runCaseState) {
	if st.stream != nil {
		st.stream.Deinit()
	}
	st.stream = nil
}

// runCaseDeinitTerminal is `defer if (t_opt) |*t| t.deinit(gpa)` and the
// third line of the init arm (main.zig:218, :237).
func runCaseDeinitTerminal(st *runCaseState) {
	if st.t != nil {
		st.t.Deinit()
	}
	st.t = nil
}

// Mirrors harness/src/main.zig:234 runCase, arm .init
func runCaseInit(st *runCaseState, v replayOpInitArgs) error {
	runCaseDeinitStream(st)
	runCaseDeinitTerminal(st)
	return runCaseTerminalInit(st, v.cols, v.rows, v.max_scrollback)
}

// Mirrors harness/src/main.zig:246 runCase, arm .feed
func runCaseFeed(st *runCaseState, bytes_ []uint8) {
	stream := st.stream
	switch st.mode.Tag {
	case feedModeWhole:
		feed(stream, bytes_, st.w)
	case feedModeBytewise:
		for i := range bytes_ {
			feed(stream, bytes_[i:i+1], st.w)
		}
	case feedModeSplit:
		if st.mode.Op == st.feedIdx && st.mode.At > 0 && st.mode.At < uint(len(bytes_)) {
			feed(stream, bytes_[:st.mode.At], st.w)
			feed(stream, bytes_[st.mode.At:], st.w)
		} else {
			feed(stream, bytes_, st.w)
		}
	}
	st.feedIdx++
}

// Mirrors harness/src/main.zig:258 runCase, arm .resize
//
// The returned error is Go only: a failed integrity check of a
// transient_invariant case (the resize error itself is part of the output).
func runCaseResize(st *runCaseState, v replayOpResizeArgs) error {
	if err := st.t.Resize(v.cols, v.rows); err != nil {
		fmt.Fprintf(st.w, "!resize error=%s\n", zigErrorName(err))
	}
	if st.integrity {
		return verifyScreensIntegrity(st.t)
	}
	return nil
}

// Mirrors harness/src/main.zig:264 runCase, arm .scroll_top
func runCaseScrollTop(st *runCaseState) {
	st.t.screens.active.Scroll(ScreenScroll{Tag: ScreenScrollTop})
}

// Mirrors harness/src/main.zig:265 runCase, arm .scroll_active
func runCaseScrollActive(st *runCaseState) {
	st.t.screens.active.Scroll(ScreenScroll{Tag: ScreenScrollActive})
}

// Mirrors harness/src/main.zig:266 runCase, arm .scroll_delta
func runCaseScrollDelta(st *runCaseState, n int) {
	st.t.screens.active.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: n})
}

// Mirrors harness/src/main.zig:267 runCase, arm .dump
func runCaseDump(st *runCaseState, label string) {
	dumpTerminal(st.w, st.t, label)
	st.dumped = true
}

// Mirrors harness/src/main.zig:271 runCase, arm .expect_crash
//
// Only reached in the corrected recorder: nothing to do.
func runCaseExpectCrash(st *runCaseState) {}

// Mirrors harness/src/main.zig:272 runCase, arm .upstream_invalid
//
// Filtered by produce(): never reached.
func runCaseUpstreamInvalid(st *runCaseState) {
	panic("unreachable")
}

// Mirrors harness/src/main.zig:273 runCase, arm .transient_invariant
//
// Go only: the Zig arm is empty. Here it switches on the integrity checks
// (see the file header). The marker is the first op of its case.
func runCaseTransientInvariant(st *runCaseState) {
	st.integrity = true
}

// verifyScreensIntegrity calls PageList.verifyIntegrity on the pages of every
// present screen: the check of the transient_invariant cases (see the file
// header). The first error is returned, with the screen and the Zig name of
// the error.
func verifyScreensIntegrity(t *Terminal) error {
	for _, key := range []ScreenSetKey{ScreenSetKeyPrimary, ScreenSetKeyAlternate} {
		if screen := t.screens.Get(key); screen != nil {
			if err := screen.pages.verifyIntegrity(); err != nil {
				return fmt.Errorf("integrity check failed on screen %s: %s", key.String(), zigErrorName(err))
			}
		}
	}
	return nil
}

// zigErrorName is `@errorName(err)` for an error of the port. The port never
// wraps errors (docs/mapping.md): it returns the bare sentinel, whose text is
// the Zig error name (TestCorpusReplaySelfErrorSentinels checks every
// sentinel). Anything else, a wrapped sentinel or an error of another origin,
// is a bug of the port or of the replayer and fails loudly, with its type, by a
// panic that corpusProduce turns into a `!panic:` line of the output.
func zigErrorName(err error) string {
	for _, s := range zigSentinels {
		if err == s.err {
			return err.Error()
		}
	}
	panic(fmt.Sprintf("zigErrorName: %T %q is not a bare sentinel of the package%s", err, err.Error(), wrappedNote(err)))
}

// wrappedNote says if an error that is not a sentinel wraps one.
func wrappedNote(err error) string {
	if u := errors.Unwrap(err); u != nil {
		return fmt.Sprintf(" (wraps %T %q)", u, u.Error())
	}
	return ""
}

// zigSentinels lists every `Err*` variable of the non-test files of the
// package, by name. TestCorpusReplaySelfErrorSentinels parses the source and
// compares this list with the declarations, so a new sentinel that is not
// added here fails the test, and each text with its Zig error name.
var zigSentinels = []struct {
	name string
	err  error
}{
	{"ErrOutOfMemory", ErrOutOfMemory},
	{"ErrMissingOperation", ErrMissingOperation},
	{"ErrNoDefaultValue", ErrNoDefaultValue},
	{"ErrInvalidFormat", ErrInvalidFormat},
	{"ErrNoSpaceLeft", ErrNoSpaceLeft},
	{"ErrUnknownKey", ErrUnknownKey},
	{"ErrInvalidValue", ErrInvalidValue},
	{"ErrDecodeError", ErrDecodeError},
	{"ErrNeedsRehash", ErrNeedsRehash},
	{"ErrZeroRowCount", ErrZeroRowCount},
	{"ErrZeroColCount", ErrZeroColCount},
	{"ErrUnmarkedGraphemeRow", ErrUnmarkedGraphemeRow},
	{"ErrMissingGraphemeData", ErrMissingGraphemeData},
	{"ErrInvalidGraphemeCount", ErrInvalidGraphemeCount},
	{"ErrUnmarkedGraphemeCell", ErrUnmarkedGraphemeCell},
	{"ErrMissingStyle", ErrMissingStyle},
	{"ErrUnmarkedStyleRow", ErrUnmarkedStyleRow},
	{"ErrMismatchedStyleRef", ErrMismatchedStyleRef},
	{"ErrInvalidStyleCount", ErrInvalidStyleCount},
	{"ErrMissingHyperlinkData", ErrMissingHyperlinkData},
	{"ErrMismatchedHyperlinkRef", ErrMismatchedHyperlinkRef},
	{"ErrUnmarkedHyperlinkCell", ErrUnmarkedHyperlinkCell},
	{"ErrUnmarkedHyperlinkRow", ErrUnmarkedHyperlinkRow},
	{"ErrInvalidSpacerTailLocation", ErrInvalidSpacerTailLocation},
	{"ErrInvalidSpacerHeadLocation", ErrInvalidSpacerHeadLocation},
	{"ErrUnwrappedSpacerHead", ErrUnwrappedSpacerHead},
	{"ErrStyleSetOutOfMemory", ErrStyleSetOutOfMemory},
	{"ErrStyleSetNeedsRehash", ErrStyleSetNeedsRehash},
	{"ErrStringAllocOutOfMemory", ErrStringAllocOutOfMemory},
	{"ErrHyperlinkSetOutOfMemory", ErrHyperlinkSetOutOfMemory},
	{"ErrHyperlinkSetNeedsRehash", ErrHyperlinkSetNeedsRehash},
	{"ErrHyperlinkMapOutOfMemory", ErrHyperlinkMapOutOfMemory},
	{"ErrGraphemeMapOutOfMemory", ErrGraphemeMapOutOfMemory},
	{"ErrGraphemeAllocOutOfMemory", ErrGraphemeAllocOutOfMemory},
	{"ErrStringsOutOfMemory", ErrStringsOutOfMemory},
	{"ErrSetOutOfMemory", ErrSetOutOfMemory},
	{"ErrSetNeedsRehash", ErrSetNeedsRehash},
	{"ErrMissingAssignment", ErrMissingAssignment},
	{"ErrInvalidMod", ErrInvalidMod},
	{"ErrWriteFailed", ErrWriteFailed},
	{"ErrPageSerialInvalid", ErrPageSerialInvalid},
	{"ErrTotalRowsMismatch", ErrTotalRowsMismatch},
	{"ErrTrackedPinInvalid", ErrTrackedPinInvalid},
	{"ErrViewportPinOffsetMismatch", ErrViewportPinOffsetMismatch},
	{"ErrViewportPinInsufficientRows", ErrViewportPinInsufficientRows},
	{"ErrOutOfSpace", ErrOutOfSpace},
	{"ErrFeedRequired", ErrFeedRequired},
	{"ErrSearchComplete", ErrSearchComplete},
	{"ErrUnknownPoint", ErrUnknownPoint},
	{"ErrInvalidUtf8", ErrInvalidUtf8},
	{"ErrMutableRequired", ErrMutableRequired},
}

// Mirrors harness/src/main.zig:286 feed
//
// A feed error is part of the recorded output.
func feed[H StreamHandler](stream *Stream[H], bytes_ []uint8, w *bytes.Buffer) {
	if err := stream.NextSlice(bytes_); err != nil {
		fmt.Fprintf(w, "!feed error=%s\n", zigErrorName(err))
	}
}

// Mirrors harness/src/main.zig:312 expectsCrash
func expectsCrash(ops []replayOp) bool {
	for _, op := range ops {
		if op.Tag == replayOpExpectCrash {
			return true
		}
	}
	return false
}

// Mirrors harness/src/main.zig:317 isUpstreamInvalid
func isUpstreamInvalid(ops []replayOp) bool {
	for _, op := range ops {
		if op.Tag == replayOpUpstreamInvalid {
			return true
		}
	}
	return false
}

// Mirrors harness/src/main.zig:322 produce
func produce(ops []replayOp, kind replayKind, mode feedMode, w *bytes.Buffer) error {
	for _, op := range ops {
		if op.Tag == replayOpTransientInvariant {
			w.WriteString("!transient-upstream-invariant-failure-end-state-verified\n")
			break
		}
	}
	if isUpstreamInvalid(ops) {
		w.WriteString("!upstream-integrity-failure\n")
		return nil
	}
	if expectsCrash(ops) {
		if !replayCorrected {
			w.WriteString("!upstream-crash\n")
			return nil
		}
		// Built against the corrected upstream: record what upstream
		// intends, and say where the output came from.
		w.WriteString("!recorded-from-corrected-upstream\n")
	}
	switch kind {
	case replayKindGolden:
		return runCase(ops, mode, w)
	case replayKindActions, replayKindActionsNoalloc, replayKindParser:
		// Stage 2: runTrace (main.zig:173).
		return errReplayKindNotRun
	}
	return errReplayKindNotRun
}

// Mirrors harness/src/main.zig:391 produceSuite
func produceSuite(cases []replayNamed, kind replayKind, w *bytes.Buffer) error {
	for _, c := range cases {
		fmt.Fprintf(w, "## case %s\n", c.name)
		if err := produce(c.ops, kind, feedMode{Tag: feedModeWhole}, w); err != nil {
			return err
		}
	}
	return nil
}
