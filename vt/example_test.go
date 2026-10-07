package vt_test

import (
	"bytes"
	"fmt"

	"github.com/XIAZY/libghostty-vt-go/vt"
	"github.com/XIAZY/libghostty-vt-go/vt/color"
	"github.com/XIAZY/libghostty-vt-go/vt/input"
)

// A terminal is fed the bytes a program writes, and its screen is read back:
// as text, or cell by cell through a render state.
func Example() {
	colors := vt.TerminalColorsDefault()
	// The default colours are unset until the application gives its own.
	colors.Foreground = color.NewDynamicRGB(color.RGB{R: 0xD0, G: 0xD0, B: 0xD0})
	colors.Background = color.NewDynamicRGB(color.RGB{R: 0x10, G: 0x14, B: 0x18})

	term, err := vt.NewTerminal(vt.TerminalOptions{
		Cols:          40,
		Rows:          5,
		MaxScrollback: 1 << 20, // bytes
		Colors:        colors,
		DefaultModes:  vt.NewModePacked(),
	})
	if err != nil {
		panic(err)
	}
	defer term.Deinit()

	// The stream parses what a program writes and applies it to the
	// terminal. (This one ignores queries; see ExampleNewStream.)
	stream := term.VtStream()
	defer stream.Deinit()
	if err := stream.NextSlice([]byte("hello \x1b[1;31mworld\x1b[0m\r\nsecond line")); err != nil {
		panic(err)
	}

	text, _ := term.PlainString()
	fmt.Printf("%q\n", text)

	// A render state is a snapshot of the visible screen for drawing.
	rs := vt.RenderStateEmpty()
	if err := rs.Update(&term); err != nil {
		panic(err)
	}
	cell := rs.RowData[0].Cells[6] // the "w" of "world"
	fg := cell.Style.Fg(vt.StyleFg{Default: rs.Colors.Foreground, Palette: &rs.Colors.Palette})
	fmt.Printf("%c bold=%v fg=#%02x%02x%02x\n", cell.Raw.Codepoint(), cell.Style.Flags.Bold, fg.R, fg.G, fg.B)
	fmt.Printf("cursor at column %d, row %d\n", rs.Cursor.Viewport.Value.X, rs.Cursor.Viewport.Value.Y)

	// Output:
	// "hello world\nsecond line"
	// w bold=true fg=#cc6666
	// cursor at column 11, row 1
}

// A key press becomes the bytes to send to the program, according to the
// modes the program has set.
func ExampleTerminal_keys() {
	term, err := vt.NewTerminal(vt.TerminalOptions{Cols: 80, Rows: 24, Colors: vt.TerminalColorsDefault(), DefaultModes: vt.NewModePacked()})
	if err != nil {
		panic(err)
	}
	defer term.Deinit()
	stream := term.VtStream()
	defer stream.Deinit()

	up := input.KeyEvent{Action: input.KeyActionPress, Key: input.KeyArrowUp}
	var out bytes.Buffer
	input.EncodeKey(&out, up, input.KeyEncodeOptionsFromTerminal(&term))
	fmt.Printf("%q\n", out.String())

	// The program switches the cursor keys to application mode.
	stream.NextSlice([]byte("\x1b[?1h"))
	out.Reset()
	input.EncodeKey(&out, up, input.KeyEncodeOptionsFromTerminal(&term))
	fmt.Printf("%q\n", out.String())

	// Output:
	// "\x1b[A"
	// "\x1bOA"
}

// answering is a stream handler that passes everything to the library's own
// handler and answers one kind of query itself.
type answering struct {
	ro    vt.ReadonlyHandler
	term  *vt.Terminal
	reply bytes.Buffer
}

func (h *answering) Deinit() {}

func (h *answering) Vt(action vt.StreamActionKey, value *vt.StreamAction) error {
	if action == vt.StreamActionKeyDeviceStatus {
		// "Where is the cursor?" The answer goes back to the program.
		c := h.term.Screens.Active.Cursor
		fmt.Fprintf(&h.reply, "\x1b[%d;%dR", c.Y+1, c.X+1)
		return nil
	}
	return h.ro.Vt(action, value)
}

// The library reports the queries a program sends and leaves the answers to
// the application: a handler of its own receives every action.
func ExampleNewStream() {
	term, err := vt.NewTerminal(vt.TerminalOptions{Cols: 80, Rows: 24, Colors: vt.TerminalColorsDefault(), DefaultModes: vt.NewModePacked()})
	if err != nil {
		panic(err)
	}
	defer term.Deinit()
	h := &answering{ro: vt.NewReadonlyHandler(&term), term: &term}
	stream := vt.NewStream(h)
	defer stream.Deinit()

	stream.NextSlice([]byte("abc\x1b[6n"))
	fmt.Printf("%q\n", h.reply.String())

	// Output:
	// "\x1b[1;4R"
}
