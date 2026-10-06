package terminal

import (
	"math/rand"
	"reflect"
	"testing"
)

// Go-only: ruling (Parser.nextInto reuses the caller's slots; every slot must equal what Parser.next returns)
func TestParser_go_only_nextInto_slots_equal_next(t *testing.T) {
	// Two parsers are fed the same bytes: one through Next (three fresh
	// actions per byte, Zig's shape), one through nextInto with one array
	// reused for the whole input, as Stream does. After every byte the reused
	// slots must be deeply equal to Next's result, stale payloads included.
	alphabet := []uint8{
		0x1B, '[', ']', 'P', '_', '^', 'X', '\\', 0x07, 0x18, 0x1A, 0x9C,
		'0', '1', '9', ';', ':', '?', '>', '$', ' ', '!',
		'm', 'H', 'A', 'c', 'q', 'p', 'h', 'l', 'u', 't',
		'a', 'Z', '~', 0x00, 0x0A, 0x0D, 0x7F, 0x80, 0x90, 0x9B, 0x9D, 0xC3, 0xFF,
	}
	for seed := int64(1); seed <= 300; seed++ {
		r := rand.New(rand.NewSource(seed))
		a := ParserInit()
		b := ParserInit()
		var slots [3]Optional[ParserAction]
		for i := 0; i < 400; i++ {
			c := alphabet[r.Intn(len(alphabet))]
			want := a.Next(c)
			b.nextInto(c, &slots)
			if !reflect.DeepEqual(want, slots) {
				t.Fatalf("seed %d byte %d (0x%02X): slots differ\n next:     %+v\n nextInto: %+v", seed, i, c, want, slots)
			}
			if a.state != b.state {
				t.Fatalf("seed %d byte %d: state %v, want %v", seed, i, b.state, a.state)
			}
		}
		a.Deinit()
		b.Deinit()
	}
}

// streamReentrantH feeds its own stream from inside a callback.
type streamReentrantH struct {
	s     *Stream[*streamReentrantH]
	log   []string
	depth int
}

func (h *streamReentrantH) Deinit() {}

func (h *streamReentrantH) Vt(action StreamActionKey, value StreamAction) error {
	h.log = append(h.log, action.String())
	// On the window title, feed a complete CSI sequence to the same stream
	// from inside the handler.
	if action == StreamActionKeyWindowTitle && h.depth == 0 {
		h.depth++
		err := h.s.NextSlice([]uint8("\x1b[2A"))
		h.depth--
		return err
	}
	return nil
}

// Go-only: ruling (the stream reuses one action array; a handler that feeds the same stream from a callback must not corrupt the outer call)
func TestStream_go_only_reentrant_feed_from_handler(t *testing.T) {
	// `OSC 0 ; t` ended by CAN gives two actions for the one byte CAN: the
	// parser's exit action is the OSC dispatch (slot 0) and its transition
	// action executes CAN (slot 1), which the stream ignores. The handler
	// re-enters on the title, while the outer call still has slot 1 to
	// process. With one shared array and no guard, the inner `CSI 2 A`
	// would overwrite slot 1 and the outer call would dispatch cursor_up a
	// second time. Zig's local array cannot be overwritten.
	h := &streamReentrantH{}
	s := StreamInit(h)
	h.s = &s
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("\x1b]0;t\x18")))
	want := []string{"window_title", "cursor_up"}
	if !reflect.DeepEqual(h.log, want) {
		t.Fatalf("got %v, want %v", h.log, want)
	}
	if s.actions_busy {
		t.Fatalf("actions_busy left set")
	}
}
