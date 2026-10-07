// Package view shows one session: it turns the session's terminal into
// frames, redrawing only the rows that changed, and turns the size of the
// drawing area into a terminal size.
//
// It knows nothing about the window: a frame is drawn into whatever pixel
// buffer it is handed, so the same code runs on the phone and in tests.
package view

import (
	"github.com/XIAZY/libghostty-vt-go/vt"

	"term51/internal/render"
	"term51/internal/session"
)

// View draws one session with one renderer. It is used from one goroutine
// (the one that owns the window).
type View struct {
	sess *session.Session
	r    *render.Renderer
	rs   vt.RenderState

	cols, rows int

	// A window hands out its buffers in turn. Each has its own list of
	// rows that are stale in it: a row that changed has to be redrawn in
	// every buffer, the next time that buffer comes round. The caller
	// names the buffer it hands over (Frame's id).
	bufs map[int]*buffer

	changed []bool // reused between frames

	// The cursor as it was drawn in the last frame.
	cursor cursorState
}

// cursorState is what the cursor looks like and where; row is -1 when it is
// not shown.
type cursorState struct {
	row, col int
	style    vt.CursorStyle
	color    uint32 // 0x01RRGGBB when the cursor has its own colour, else 0
}

func cursorOf(rs *vt.RenderState) cursorState {
	c := &rs.Cursor
	if !c.Visible || !c.Viewport.Valid {
		return cursorState{row: -1}
	}
	st := cursorState{row: int(c.Viewport.Value.Y), col: int(c.Viewport.Value.X), style: c.VisualStyle}
	if cc := rs.Colors.Cursor; cc.Valid {
		st.color = 1<<24 | uint32(cc.Value.R)<<16 | uint32(cc.Value.G)<<8 | uint32(cc.Value.B)
	}
	return st
}

type buffer struct {
	w, h, stride int
	stale        []bool // by row; nil: everything
}

// New makes the view of sess, drawn with r.
func New(sess *session.Session, r *render.Renderer) *View {
	return &View{sess: sess, r: r, rs: vt.RenderStateEmpty(), bufs: map[int]*buffer{}, cursor: cursorState{row: -1}}
}

// Resize fits the terminal to a drawing area of w by h pixels. It reports
// the new size in cells; nothing happens when that has not changed.
func (v *View) Resize(w, h int) (cols, rows int, err error) {
	cols, rows = v.r.Grid(w, h)
	if cols == v.cols && rows == v.rows {
		return cols, rows, nil
	}
	cw, ch := v.r.CellSize()
	if err := v.sess.Resize(cols, rows, cw, ch); err != nil {
		return v.cols, v.rows, err
	}
	v.cols, v.rows = cols, rows
	v.Invalidate()
	return cols, rows, nil
}

// Invalidate makes the next frame of every buffer a full redraw. The window
// code calls it whenever its buffers were created anew (a resize, a
// rotation): their contents are then unknown, whatever their ids.
func (v *View) Invalidate() {
	clear(v.bufs)
}

// Frame draws the session's current state into pix (w by h pixels, stride
// pixels per row), the window buffer the caller calls id (a window with two
// buffers passes 0 and 1). It returns the band of pixel rows it painted,
// y0 <= y < y1, across the whole width, so that the caller can show just
// that; y0 == y1 means nothing was painted and the frame need not be shown.
func (v *View) Frame(id int, pix []uint32, stride, w, h int) (y0, y1 int, err error) {
	if w <= 0 || h <= 0 || len(pix) == 0 {
		return 0, 0, nil
	}
	if err := v.sess.Snapshot(&v.rs); err != nil {
		return 0, 0, err
	}
	nrows := len(v.rs.RowData)

	// The rows that changed since the last frame: those the terminal
	// marked, and the rows the cursor left and arrived on.
	if cap(v.changed) < nrows {
		v.changed = make([]bool, nrows)
	}
	changed := v.changed[:nrows]
	all := v.rs.Dirty == vt.RenderStateDirtyFull
	if !all {
		for y := range v.rs.RowData {
			changed[y] = v.rs.RowData[y].Dirty
		}
		if cur := cursorOf(&v.rs); cur != v.cursor {
			for _, y := range []int{v.cursor.row, cur.row} {
				if y >= 0 && y < nrows {
					changed[y] = true
				}
			}
		}
	}
	// They are stale in every buffer.
	for _, b := range v.bufs {
		if all || len(b.stale) != nrows {
			b.stale = nil
			continue
		}
		if b.stale == nil {
			continue
		}
		for y, c := range changed {
			b.stale[y] = b.stale[y] || c
		}
	}
	// The terminal's marks are taken; the next snapshot marks anew.
	for y := range v.rs.RowData {
		v.rs.RowData[y].Dirty = false
	}
	v.rs.Dirty = vt.RenderStateDirtyFalse
	v.cursor = cursorOf(&v.rs)

	b := v.bufs[id]
	if b == nil || b.w != w || b.h != h || b.stride != stride {
		// A buffer not seen before, or one that changed shape: everything.
		b = &buffer{w: w, h: h, stride: stride}
		v.bufs[id] = b
	}
	if b.stale == nil {
		v.r.Draw(&v.rs, pix, stride, w, h)
		b.stale = make([]bool, nrows)
		return 0, h, nil
	}
	first, last := -1, -1
	for y, s := range b.stale {
		if s {
			if first < 0 {
				first = y
			}
			last = y
		}
	}
	if first < 0 {
		return 0, 0, nil
	}
	v.r.DrawRows(&v.rs, pix, stride, w, h, b.stale)
	clear(b.stale)
	_, ch := v.r.CellSize()
	return min(first*ch, h), min((last+1)*ch, h), nil
}
