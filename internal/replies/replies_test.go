package replies

import "testing"

type fakeState struct {
	row, col   int
	cols, rows int
	cw, ch     int
	modes      map[[2]int]ModeState
	kitty      int
	colors     map[[2]int]RGB
}

func (f *fakeState) CursorReportPos() (int, int) { return f.row, f.col }
func (f *fakeState) Size() (int, int)            { return f.cols, f.rows }
func (f *fakeState) CellSize() (int, int)        { return f.cw, f.ch }
func (f *fakeState) Mode(m int, dec bool) ModeState {
	d := 0
	if dec {
		d = 1
	}
	return f.modes[[2]int{m, d}]
}
func (f *fakeState) KittyKeyboardFlags() int { return f.kitty }
func (f *fakeState) Color(t ColorTarget, i int) (RGB, bool) {
	c, ok := f.colors[[2]int{int(t), i}]
	return c, ok
}

func newFake() *fakeState {
	return &fakeState{
		row: 3, col: 7, cols: 80, rows: 24, cw: 9, ch: 18,
		modes: map[[2]int]ModeState{
			{25, 1}: ModeSet,   // DECTCEM
			{4, 0}:  ModeReset, // IRM
		},
		kitty: 5,
		colors: map[[2]int]RGB{
			{int(Palette), 1}:    {0xcd, 0x00, 0x00},
			{int(Background), 0}: {0x1d, 0x1f, 0x21},
		},
	}
}

func TestAnswers(t *testing.T) {
	cfg := DefaultConfig()
	s := newFake()
	cases := []struct {
		name string
		q    Query
		want string
	}{
		{"primary DA, VT220", Query{Kind: PrimaryDA}, "\x1b[?62;22c"},
		{"secondary DA", Query{Kind: SecondaryDA}, "\x1b[>1;1;0c"},
		{"tertiary DA", Query{Kind: TertiaryDA}, "\x1bP!|00000000\x1b\\"},
		{"XTVERSION", Query{Kind: XTVersion}, "\x1bP>|term51 0.1\x1b\\"},
		{"DSR status", Query{Kind: StatusReport}, "\x1b[0n"},
		{"DSR cursor", Query{Kind: CursorReport}, "\x1b[3;7R"},
		{"colour scheme dark", Query{Kind: ColorScheme}, "\x1b[?997;1n"},
		{"DECRQM DEC set", Query{Kind: ModeQuery, Mode: 25, DEC: true}, "\x1b[?25;1$y"},
		{"DECRQM ANSI reset", Query{Kind: ModeQuery, Mode: 4}, "\x1b[4;2$y"},
		{"DECRQM unknown", Query{Kind: ModeQuery, Mode: 9999, DEC: true}, "\x1b[?9999;0$y"},
		{"OSC 4 query ST", Query{Kind: ColorQuery, Target: Palette, Index: 1}, "\x1b]4;1;rgb:cdcd/0000/0000\x1b\\"},
		{"OSC 11 query BEL", Query{Kind: ColorQuery, Target: Background, BEL: true}, "\x1b]11;rgb:1d1d/1f1f/2121\a"},
		{"size in cells", Query{Kind: TextAreaChars}, "\x1b[8;24;80t"},
		{"size in pixels", Query{Kind: TextAreaPixels}, "\x1b[4;432;720t"},
		{"cell in pixels", Query{Kind: CellPixels}, "\x1b[6;18;9t"},
		{"kitty flags", Query{Kind: KittyFlags}, "\x1b[?5u"},
	}
	for _, c := range cases {
		got := string(Answer(cfg, s, c.q))
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestIdentities(t *testing.T) {
	s := newFake()
	for _, c := range []struct {
		id        Identity
		primary   string
		secondary string
	}{
		{VT100, "\x1b[?1;2c", "\x1b[>0;1;0c"},
		{VT220, "\x1b[?62;22c", "\x1b[>1;1;0c"},
		{XTerm, "\x1b[?64;1;2;6;9;15;16;17;18;21;22;28c", "\x1b[>41;1;0c"},
	} {
		cfg := DefaultConfig()
		cfg.Identity = c.id
		if got := string(Answer(cfg, s, Query{Kind: PrimaryDA})); got != c.primary {
			t.Errorf("identity %d primary: got %q, want %q", c.id, got, c.primary)
		}
		if got := string(Answer(cfg, s, Query{Kind: SecondaryDA})); got != c.secondary {
			t.Errorf("identity %d secondary: got %q, want %q", c.id, got, c.secondary)
		}
	}
}

// Every kind of answer can be switched off, and then nothing is sent.
func TestSwitchedOff(t *testing.T) {
	s := newFake()
	off := Config{} // everything off, no answerback
	for k := PrimaryDA; k <= Enquiry; k++ {
		q := Query{Kind: k, Mode: 25, DEC: true, Target: Palette, Index: 1}
		if got := Answer(off, s, q); got != nil {
			t.Errorf("kind %d answered %q with everything off", k, got)
		}
	}
}

func TestNoAnswerWithoutData(t *testing.T) {
	cfg := DefaultConfig()
	s := newFake()
	if got := Answer(cfg, s, Query{Kind: ColorQuery, Target: Foreground}); got != nil {
		t.Errorf("unset colour answered %q", got)
	}
	s.cw, s.ch = 0, 0
	if got := Answer(cfg, s, Query{Kind: CellPixels}); got != nil {
		t.Errorf("unknown cell size answered %q", got)
	}
	if got := Answer(cfg, s, Query{Kind: Enquiry}); got != nil {
		t.Errorf("empty answerback sent %q", got)
	}
	cfg.Answerback = "term51"
	if got := string(Answer(cfg, s, Query{Kind: Enquiry})); got != "term51" {
		t.Errorf("answerback: got %q", got)
	}
	cfg.Dark = false
	if got := string(Answer(cfg, s, Query{Kind: ColorScheme})); got != "\x1b[?997;2n" {
		t.Errorf("light scheme: got %q", got)
	}
}
