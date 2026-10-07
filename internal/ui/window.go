//go:build qnx

package ui

/*
#cgo LDFLAGS: -lscreen
#include <stdlib.h>
#include "window.h"
*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"
)

// Window is the native Screen window the terminal is drawn into. Every
// method must be called from the goroutine that opened it, locked to its OS
// thread (runtime.LockOSThread) for the window's whole life.
type Window struct {
	w             *C.t51_win
	Width, Height int
}

// EventType is the kind of a window Event.
type EventType int

const (
	EventNone  EventType = C.T51_EV_NONE
	EventKey   EventType = C.T51_EV_KEY
	EventSize  EventType = C.T51_EV_SIZE
	EventTouch EventType = C.T51_EV_TOUCH
	EventClose EventType = C.T51_EV_CLOSE
	EventError EventType = C.T51_EV_ERROR
)

// Event is a Screen event on the window.
type Event struct {
	Type EventType
	// EventKey: the KEY_* values of sys/keycodes.h.
	KeyFlags, KeySym, KeyCap, KeyMods, KeyScan int
	// EventSize: the new size. EventTouch: the position and the phase
	// (1 down, 2 move, 3 up).
	W, H, X, Y, Touch int
}

// OpenWindow creates the window in the Cascades window group group with the
// ID string id, which the QML ForeignWindowControl binds to.
func OpenWindow(group, id string, width, height int) (*Window, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("window: bad size %dx%d", width, height)
	}
	g, i := C.CString(group), C.CString(id)
	defer C.free(unsafe.Pointer(g))
	defer C.free(unsafe.Pointer(i))
	var w *C.t51_win
	if rc, err := C.t51_win_create(&w, g, i, C.int(width), C.int(height)); rc != 0 {
		return nil, fmt.Errorf("window: create: %v", err)
	}
	return &Window{w: w, Width: width, Height: height}, nil
}

// Resize reallocates the window's buffers for a new size.
func (w *Window) Resize(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("window: bad size %dx%d", width, height)
	}
	if rc, err := C.t51_win_resize(w.w, C.int(width), C.int(height)); rc != 0 {
		return fmt.Errorf("window: resize: %v", err)
	}
	w.Width, w.Height = width, height
	return nil
}

// Frame calls draw with the next buffer: its id (0 or 1, which of the
// window's two buffers it is, stable until the next Resize), its pixels as
// 0x00RRGGBB and stride in pixels per row. draw returns the band of rows it
// painted, y0 <= y < y1, which is then shown; with y0 == y1 nothing is shown
// and the same buffer is handed out again next time.
func (w *Window) Frame(draw func(id int, pix []uint32, stride int) (y0, y1 int)) error {
	var strideBytes, index C.int
	p := C.t51_win_buffer(w.w, &strideBytes, &index)
	if p == nil {
		return fmt.Errorf("window: no buffer")
	}
	stride := int(strideBytes) / 4
	y0, y1 := draw(int(index), unsafe.Slice((*uint32)(p), stride*w.Height), stride)
	if rc, err := C.t51_win_post(w.w, C.int(y0), C.int(y1)); rc != 0 {
		return fmt.Errorf("window: post: %v", err)
	}
	return nil
}

// NextEvent waits up to timeout (negative: forever) for the next event.
func (w *Window) NextEvent(timeout time.Duration) Event {
	var ev C.t51_event
	C.t51_win_next_event(w.w, C.longlong(timeout), &ev)
	return Event{
		Type:     EventType(ev._type),
		KeyFlags: int(ev.key_flags), KeySym: int(ev.key_sym), KeyCap: int(ev.key_cap),
		KeyMods: int(ev.key_mods), KeyScan: int(ev.key_scan),
		W: int(ev.w), H: int(ev.h), X: int(ev.x), Y: int(ev.y), Touch: int(ev.touch_type),
	}
}

// Close destroys the window.
func (w *Window) Close() {
	C.t51_win_destroy(w.w)
	w.w = nil
}
