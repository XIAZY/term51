package terminal

// The trace kinds of the corpus replayer: the Go side of the recorder's
// harness/src/trace.zig (the action handler and the parser trace) and of
// runTrace in harness/src/main.zig. One Go function per Zig function, same
// name in Go style, each with the Zig line it mirrors.
//
// The three kinds (`kinds` line of a suite, golden/<area>/<suite>.<kind>.gz):
//
//	actions          every Action that Stream dispatches, tag and payload, for a
//	                 stream made by StreamInitAlloc (nil-ok allocator given);
//	actions_noalloc  the same for a stream made by StreamInit (no allocator: an
//	                 OSC or DCS payload that needs one is not dispatched);
//	parser           every raw Parser action, byte by byte, with the parser state
//	                 after each byte; the parser's osc_parser has an allocator.
//
// Differences from the Zig, forced by the language:
//   - the writer is a *bytes.Buffer: writes never fail, so flush and parserFeed
//     return nothing and Vt returns nil;
//   - Zig's Vt receives the payload of the action (Action.Value(action)); Go's
//     StreamHandler.Vt receives the whole StreamAction and the payload is its
//     field named by the tag (dumpUnionPayload), nothing for a void tag;
//   - `gpa` (the allocator) is replayAllocator: StreamInitAlloc and
//     osc_parser.alloc only distinguish nil from non-nil.
//
// # Pending osc.go
//
// The OSC parser of the port is still a placeholder (osc.go), so a trace that
// contains an action that only an OSC sequence produces cannot be compared yet:
// corpusPendingOSC is the ONE place that says which traces those are (the
// action tags of stream.zig oscDispatch for the action kinds, `@osc_dispatch`
// for the parser kind). The replayers (corpus_replay_test.go replayCase and
// corpus_replay_split_test.go splitCheck) skip such a case and kind and count it
// as pending_osc in the suite line; nothing else is excluded. When osc.go is
// ported, empty corpusPendingOSCTags and the parser branch of corpusPendingOSC:
// TestCorpusReplaySelfPendingOSC then fails until the counts are updated.

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

// replayAllocator stands for `gpa`, the allocator that the recorder hands to
// StreamInitAlloc and sets as parser.osc_parser.alloc: for the port only
// "an allocator was provided" (non-nil) matters (stream.go, osc.go).
type replayAllocator struct{}

// corpusPendingOSCTags are the tags (Zig names) of the stream actions that only
// an OSC command produces: the arms of stream.zig:1942 oscDispatch that call
// `handler.vt(...)`. TestCorpusReplaySelfPendingOSC compares the list with the
// Zig source.
var corpusPendingOSCTags = []string{
	"semantic_prompt", "window_title", "clipboard_contents", "report_pwd",
	"mouse_shape", "color_operation", "kitty_color_report",
	"show_desktop_notification", "start_hyperlink", "end_hyperlink",
	"progress_report",
}

// corpusPendingOSC is the exclusion of the traces that are pending osc.go (see
// the file header): whether the recorded trace `want` (the text of a section)
// of the kind contains such an action. The kinds golden and search are never
// pending.
func corpusPendingOSC(kind replayKind, want []byte) bool {
	switch kind {
	case replayKindActions, replayKindActionsNoalloc:
		for _, line := range strings.Split(string(want), "\n") {
			tag, ok := strings.CutPrefix(line, "@")
			if ok && slicesContainsString(corpusPendingOSCTags, tag) {
				return true
			}
		}
	case replayKindParser:
		for _, line := range strings.Split(string(want), "\n") {
			if line == "@osc_dispatch" {
				return true
			}
		}
	}
	return false
}

func slicesContainsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Mirrors harness/src/trace.zig:16 ActionHandler
type actionHandler struct {
	w *bytes.Buffer

	// True while a run of consecutive print actions is being written on
	// one line: "@print <hex cp>,<hex cp>,...".
	inPrintRun bool
}

// Mirrors harness/src/trace.zig:23 ActionHandler.deinit
func (self *actionHandler) Deinit() {}

// Mirrors harness/src/trace.zig:28 ActionHandler.flush
//
// End any open print run. Must be called once after the last input.
func (self *actionHandler) flush() {
	if self.inPrintRun {
		self.inPrintRun = false
		self.w.WriteString("\n")
	}
}

// Mirrors harness/src/trace.zig:35 ActionHandler.vt
func (self *actionHandler) Vt(action StreamActionKey, value StreamAction) error {
	if action == StreamActionKeyPrint {
		if self.inPrintRun {
			fmt.Fprintf(self.w, ",%x", value.Print.cp)
		} else {
			self.inPrintRun = true
			fmt.Fprintf(self.w, "@print %x", value.Print.cp)
		}
		return nil
	}
	self.flush()
	name := action.String()
	self.w.WriteString("@" + name + "\n")
	dumpActionPayload(self.w, " v", value, name)
	return nil
}

// dumpActionPayload is `dumpAny(w, prefix, value)` for the payload of the
// action `value`, whose tag is `name` (see the file header): nothing for a void
// tag.
func dumpActionPayload(w *bytes.Buffer, prefix string, value StreamAction, name string) {
	// An addressable copy, so that dumpField can reach unexported fields.
	addr := reflect.New(reflect.TypeOf(value)).Elem()
	addr.Set(reflect.ValueOf(value))
	if payload, ok := dumpUnionPayload(addr, name, prefix); ok {
		dumpValue(w, prefix, payload)
	}
}

// Mirrors harness/src/trace.zig:58 parserFeed
//
// Feed bytes to a raw Parser and print each action.
func parserFeed(w *bytes.Buffer, p *Parser, bytes_ []uint8) {
	for _, c := range bytes_ {
		actions := p.Next(c)
		fmt.Fprintf(w, "> %02x state=%s\n", c, p.state.String())
		for _, actionOpt := range actions {
			if !actionOpt.Valid {
				continue
			}
			action := actionOpt.Value
			fmt.Fprintf(w, "@%s\n", action.Tag.String())
			dumpAny(w, " v", action)
		}
	}
}

// Mirrors harness/src/main.zig:173 runTrace
//
// Run a case through the action-trace stream (kind actions, actions_noalloc)
// or the raw parser (kind parser). Only feed ops matter; the rest are ignored.
func runTrace(ops []replayOp, mode feedMode, kind replayKind, w *bytes.Buffer) error {
	if kind != replayKindActions && kind != replayKindActionsNoalloc && kind != replayKindParser {
		panic("unreachable")
	}
	handler := &actionHandler{w: w}
	// Go difference: Zig creates the stream for every kind; the stream of the
	// parser kind is never fed, and the port's Stream is not needed to replay
	// that kind, so it is not created for it (TestCorpusReplaySelfParserTrace
	// replays the parser kind without the port's stream).
	var stream Stream[*actionHandler]
	if kind != replayKindParser {
		if kind == replayKindActionsNoalloc {
			stream = StreamInit(handler)
		} else {
			stream = StreamInitAlloc(replayAllocator{}, handler)
		}
		defer stream.Deinit()
	}
	defer handler.flush()
	parser := ParserInit()
	defer parser.Deinit()
	parser.osc_parser.alloc = replayAllocator{}

	var feedIdx uint
	for _, op := range ops {
		if op.Tag != replayOpFeed {
			continue
		}
		bytes_ := op.Feed
		switch kind {
		case replayKindParser:
			parserFeed(w, &parser, bytes_)
		case replayKindActions, replayKindActionsNoalloc:
			switch mode.Tag {
			case feedModeWhole:
				feed(&stream, bytes_, w)
			case feedModeBytewise:
				for i := range bytes_ {
					feed(&stream, bytes_[i:i+1], w)
				}
			case feedModeSplit:
				if mode.Op == feedIdx && mode.At > 0 && mode.At < uint(len(bytes_)) {
					feed(&stream, bytes_[:mode.At], w)
					feed(&stream, bytes_[mode.At:], w)
				} else {
					feed(&stream, bytes_, w)
				}
			}
		}
		feedIdx++
	}
	return nil
}
