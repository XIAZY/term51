package session

import (
	"bytes"

	"github.com/XIAZY/libghostty-vt-go/vt"
	"github.com/XIAZY/libghostty-vt-go/vt/color"
	"github.com/XIAZY/libghostty-vt-go/vt/devicestatus"
	"github.com/XIAZY/libghostty-vt-go/vt/modes"
	"github.com/XIAZY/libghostty-vt-go/vt/osc"

	"term51/internal/replies"
)

// handler receives every action of the stream. Everything that changes the
// terminal goes to the library's own handler; the queries, which the library
// leaves to the application, are answered into out; titles and bells become
// events. It runs with the session's lock held.
type handler struct {
	s      *Session
	ro     vt.ReadonlyHandler
	out    bytes.Buffer
	events []Event
}

func (h *handler) Deinit() { h.ro.Deinit() }

func (h *handler) answer(q replies.Query) {
	h.out.Write(replies.Answer(h.s.cfg, state{h.s}, q))
}

func (h *handler) Vt(action vt.StreamActionKey, value *vt.StreamAction) error {
	switch action {
	case vt.StreamActionKeyDeviceAttributes:
		switch value.DeviceAttributes {
		case vt.DeviceAttributeReqPrimary:
			h.answer(replies.Query{Kind: replies.PrimaryDA})
		case vt.DeviceAttributeReqSecondary:
			h.answer(replies.Query{Kind: replies.SecondaryDA})
		case vt.DeviceAttributeReqTertiary:
			h.answer(replies.Query{Kind: replies.TertiaryDA})
		}
	case vt.StreamActionKeyXtversion:
		h.answer(replies.Query{Kind: replies.XTVersion})
	case vt.StreamActionKeyEnquiry:
		h.answer(replies.Query{Kind: replies.Enquiry})
	case vt.StreamActionKeyDeviceStatus:
		switch value.DeviceStatus.Request {
		case devicestatus.RequestOperatingStatus:
			h.answer(replies.Query{Kind: replies.StatusReport})
		case devicestatus.RequestCursorPosition:
			h.answer(replies.Query{Kind: replies.CursorReport})
		case devicestatus.RequestColorScheme:
			h.answer(replies.Query{Kind: replies.ColorScheme})
		}
	case vt.StreamActionKeyRequestMode:
		tag := modes.ModeTagFromInt(modes.ModeTagBacking(value.RequestMode.Mode))
		h.answer(replies.Query{Kind: replies.ModeQuery, Mode: int(tag.Value), DEC: !tag.Ansi})
	case vt.StreamActionKeyRequestModeUnknown:
		h.answer(replies.Query{Kind: replies.ModeQuery, Mode: int(value.RequestModeUnknown.Mode), DEC: !value.RequestModeUnknown.Ansi})
	case vt.StreamActionKeySizeReport:
		switch value.SizeReport {
		case vt.SizeReportStyleCsi14T:
			h.answer(replies.Query{Kind: replies.TextAreaPixels})
		case vt.SizeReportStyleCsi16T:
			h.answer(replies.Query{Kind: replies.CellPixels})
		case vt.SizeReportStyleCsi18T:
			h.answer(replies.Query{Kind: replies.TextAreaChars})
		}
	case vt.StreamActionKeyKittyKeyboardQuery:
		h.answer(replies.Query{Kind: replies.KittyFlags})
	case vt.StreamActionKeyColorOperation:
		// Queries are answered here; sets and resets are the library's.
		op := &value.ColorOperation
		bel := op.Terminator == osc.TerminatorBel
		for i := 0; i < op.Requests.Count(); i++ {
			req := op.Requests.At(i)
			if req.Tag != osc.ParsersColorRequestQuery {
				continue
			}
			switch req.Query.Tag {
			case osc.ParsersColorTargetPalette:
				h.answer(replies.Query{Kind: replies.ColorQuery, Target: replies.Palette, Index: int(req.Query.Palette), BEL: bel})
			case osc.ParsersColorTargetDynamic:
				switch req.Query.Dynamic {
				case color.DynamicForeground:
					h.answer(replies.Query{Kind: replies.ColorQuery, Target: replies.Foreground, BEL: bel})
				case color.DynamicBackground:
					h.answer(replies.Query{Kind: replies.ColorQuery, Target: replies.Background, BEL: bel})
				case color.DynamicCursor:
					h.answer(replies.Query{Kind: replies.ColorQuery, Target: replies.Cursor, BEL: bel})
				}
			}
		}
		return h.ro.Vt(action, value)
	case vt.StreamActionKeyWindowTitle:
		h.events = append(h.events, Event{Kind: EventTitle, Text: string(value.WindowTitle.Title)})
	case vt.StreamActionKeyBell:
		h.events = append(h.events, Event{Kind: EventBell})
	case vt.StreamActionKeyReportPwd:
		h.events = append(h.events, Event{Kind: EventPwd, Text: string(value.ReportPwd.URL)})
	case vt.StreamActionKeyClipboardContents:
		h.events = append(h.events, Event{Kind: EventClipboard, Text: string(value.ClipboardContents.Data)})
	default:
		return h.ro.Vt(action, value)
	}
	return nil
}

// state is what package replies needs to know about the terminal.
type state struct{ s *Session }

func (st state) CursorReportPos() (row, col int) {
	t := &st.s.term
	c := &t.Screens.Active.Cursor
	row, col = int(c.Y), int(c.X)
	if t.Modes.Get(vt.ModeOrigin) {
		row -= int(t.ScrollingRegion.Top)
		col -= int(t.ScrollingRegion.Left)
	}
	return row + 1, col + 1
}

func (st state) Size() (cols, rows int) { return int(st.s.term.Cols), int(st.s.term.Rows) }

func (st state) CellSize() (width, height int) { return st.s.cellW, st.s.cellH }

func (st state) Mode(mode int, dec bool) replies.ModeState {
	m := modes.ModeFromInt(uint16(mode), !dec)
	if !m.Valid {
		return replies.ModeUnknown
	}
	if st.s.term.Modes.Get(m.Value) {
		return replies.ModeSet
	}
	return replies.ModeReset
}

func (st state) KittyKeyboardFlags() int {
	return int(st.s.term.Screens.Active.KittyKeyboard.Current().Int())
}

func (st state) Color(target replies.ColorTarget, index int) (replies.RGB, bool) {
	t := &st.s.term
	rgb := func(c vt.Optional[color.RGB]) (replies.RGB, bool) {
		if !c.Valid {
			return replies.RGB{}, false
		}
		return replies.RGB{R: c.Value.R, G: c.Value.G, B: c.Value.B}, true
	}
	switch target {
	case replies.Palette:
		if index < 0 || index > 255 {
			return replies.RGB{}, false
		}
		c := t.Colors.Palette.Current[index]
		return replies.RGB{R: c.R, G: c.G, B: c.B}, true
	case replies.Foreground:
		return rgb(t.Colors.Foreground.Get())
	case replies.Background:
		return rgb(t.Colors.Background.Get())
	case replies.Cursor:
		return rgb(t.Colors.Cursor.Get())
	}
	return replies.RGB{}, false
}
