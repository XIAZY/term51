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

import (
	"bytes"
	"fmt"
	"reflect"
)

// replayAllocator stands for `gpa`, the allocator that the recorder hands to
// StreamInitAlloc and sets as parser.osc_parser.alloc: for the port only
// "an allocator was provided" (non-nil) matters (stream.go, osc.go).
type replayAllocator struct{}

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
func (self *actionHandler) Vt(action StreamActionKey, value *StreamAction) error {
	if action == StreamActionKeyPrint {
		if self.inPrintRun {
			fmt.Fprintf(self.w, ",%x", value.Print.Cp)
		} else {
			self.inPrintRun = true
			fmt.Fprintf(self.w, "@print %x", value.Print.Cp)
		}
		return nil
	}
	self.flush()
	name := action.String()
	self.w.WriteString("@" + name + "\n")
	// The dump is written during the call; dumpActionPayload takes a copy.
	dumpActionPayload(self.w, " v", *value, name)
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
			stream = StreamInitAlloc(handler)
		}
		defer stream.Deinit()
	}
	defer handler.flush()
	parser := ParserInit()
	defer parser.Deinit()
	parser.osc_parser.alloc = true

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
