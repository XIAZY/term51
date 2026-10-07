package terminal

import (
	"reflect"
	"testing"
	"unsafe"
)

// streamActionKeys returns every StreamActionKey: one per name in the
// table StreamActionKey.String reads.
func streamActionKeys(t *testing.T) []StreamActionKey {
	t.Helper()
	var keys []StreamActionKey
	for k := range streamActionKeyNames {
		keys = append(keys, StreamActionKey(k))
	}
	if len(keys) < 90 {
		t.Fatalf("only %d action keys found", len(keys))
	}
	return keys
}

// fillNonZero sets every part of v to a non-zero value, unexported fields
// included.
func fillNonZero(v reflect.Value) {
	if !v.CanSet() {
		v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
	}
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1)
	case reflect.String:
		v.SetString("x")
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		fillNonZero(s.Index(0))
		v.Set(s)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			fillNonZero(v.Index(i))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			fillNonZero(v.Field(i))
		}
	case reflect.Pointer:
		p := reflect.New(v.Type().Elem())
		fillNonZero(p.Elem())
		v.Set(p)
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
	}
}

// Go-only: ruling (the stream reuses one StreamAction; for every action key, a reused action must equal a freshly built one)
func TestStream_go_only_action_reused_equals_fresh(t *testing.T) {
	// For every key: an action of that key with its whole payload non-zero
	// is cleared by streamActionClear to exactly the zero StreamAction, so
	// the next action built in the same storage equals a freshly built one.
	// The payload field of a key is found by name here, independently of the
	// switch in streamActionClear.
	rt := reflect.TypeOf(StreamAction{})
	withPayload := 0
	for _, key := range streamActionKeys(t) {
		var a StreamAction
		a.Tag = key
		name := zigFieldName(key.String())
		if f, ok := rt.FieldByName(name); ok {
			fv := reflect.ValueOf(&a).Elem().FieldByIndex(f.Index)
			fillNonZero(fv)
			if f.Type.Size() > 0 && fv.IsZero() {
				t.Fatalf("%s: could not make the payload field %s non-zero", key, name)
			}
			withPayload++
		}
		streamActionClear(&a)
		if !reflect.DeepEqual(a, StreamAction{}) {
			t.Errorf("%s: after streamActionClear the action is not zero: %+v", key, a)
		}
	}
	// Every payload field belongs to a key (Tag is the one other field).
	if withPayload != rt.NumField()-1 {
		t.Errorf("%d keys have a payload field, StreamAction has %d payload fields", withPayload, rt.NumField()-1)
	}
}

// streamStaleCheckH checks that every action it receives has no payload but
// its own.
type streamStaleCheckH struct {
	t    *testing.T
	seen map[StreamActionKey]int
}

func (h *streamStaleCheckH) Deinit() {}

func (h *streamStaleCheckH) Vt(action StreamActionKey, value *StreamAction) error {
	h.seen[action]++
	if value.Tag != action {
		h.t.Fatalf("%s: value.Tag is %s", action, value.Tag)
	}
	// A copy with its own payload cleared must be the zero action.
	c := *value
	streamActionClear(&c)
	if !reflect.DeepEqual(c, StreamAction{}) {
		h.t.Fatalf("%s: the action carries another action's payload: %+v", action, c)
	}
	return nil
}

// Go-only: ruling (the stream reuses one StreamAction; an action must never carry a stale payload of an earlier one)
func TestStream_go_only_action_has_no_stale_payload(t *testing.T) {
	h := &streamStaleCheckH{t: t, seen: map[StreamActionKey]int{}}
	s := StreamInit(h)
	defer s.Deinit()
	inputs := [][]byte{
		benchInputASCII()[:2048],
		benchInputSGR()[:4096],
		benchInputMixed()[:4096],
		[]byte("\x1b]0;title\x07\x1b]2;other\x1b\\\x1b]8;id=a;https://example.com\x1b\\link\x1b]8;;\x1b\\" +
			"\x1b]4;1;?\x07\x1b]10;?\x1b\\\x1b]52;c;aGVsbG8=\x07\x1b]7;file://host/tmp\x07\x1b]133;A\x07" +
			"\x1bP+q544e\x1b\\\x1b_Gf=24\x1b\\\x1b[c\x1b[>c\x1b[5n\x1b[6n\x1b[?25$p\x1b[4$p\x1b[18t\x1b[?u\x1b[>1u\x1b[<u" +
			"\x1b[?1049h\x1b[?1049l\x1b[1;5r\x1b[s\x1b[u\x1b7\x1b8\x1bM\x1bD\x1bE\x1bH\x1b[g\x1b[3g\x1b(0\x0e\x0f\x05\x07\x08\t\n\r" +
			"\x1b[2 q\x1b[1\"q\x1b[38;2;1;2;3;48;5;4;4:3;58:2::5:6:7m\x1b[0m\x1b[4h\x1b[4l\x1b[?2004h\x1b[=1;1u\x1b[20;10H\x1b[3@\x1b[2P\x1b[2X\x1b[2L\x1b[2M"),
	}
	for _, in := range inputs {
		zigTry(t, s.NextSlice(in))
	}
	if len(h.seen) < 40 {
		t.Errorf("only %d different actions seen; the inputs should exercise more", len(h.seen))
	}
}
