//go:build qnx

// Package gpu is the screen: a window of our own inside the Cascades page,
// drawn with OpenGL. It knows nothing of the terminal. It is given a table
// with one record per cell and a sheet of character shapes, and draws every
// cell from them.
//
// The cell table: cols*rows records of 16 bytes, row by row from the top
// left, little-endian:
//
//	0  uint16  slot in the character sheet; 0 draws no character
//	2  uint16  flags: 1 underline, 2 strikethrough, 4 cursor bar,
//	           8 cursor underline, 16 hollow cursor box
//	4  4 bytes text colour R, G, B, 255
//	8  4 bytes background colour R, G, B, 255
//	12 4 bytes line colour R, G, B, 255
//
// The underline and the three cursor shapes are drawn in the line colour,
// the strikethrough in the text colour. Selection, reverse video and the
// block cursor are already in the colours.
//
// The character sheet: one 8-bit picture of SheetSize by SheetSize, cut into
// slots the size of a cell; slot n is at column n % (SheetSize/cellW), row
// n / (SheetSize/cellW). Its value is how much of the text colour a pixel
// gets.
//
// The window draws with OpenGL ES 3.0 where the phone has it (the Passport)
// and with 2.0 where it does not (the Classic, Q10 and Z10); Open picks.
// What differs between the two is how the table reaches the GPU and is
// drawn, which is a pipeline (es3.go, es2.go) with its own pair of shaders
// (shaders.go); the window, EGL, the program's uniforms and everything the
// app sees are shared.
//
// Go crosses into C once per frame (Frame), and once more only when part of
// the sheet is new (Sheet). The frame is the one piece written in C, so that
// it is one crossing and not many. Everything else, which runs once or on a
// resize, is Go calling the window system's, EGL's and OpenGL's functions
// directly. Everything must be called from one goroutine locked to its
// thread: an OpenGL context belongs to a thread.
package gpu

/*
#cgo LDFLAGS: -lscreen -lEGL -lGLESv2
#include <stdlib.h>
#include <screen/screen.h>
#include <EGL/egl.h>
#include <GLES2/gl2.h>
*/
import "C"

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unsafe"
)

// EGL_OPENGL_ES3_BIT_KHR, which the phone's egl.h does not name.
const eglOpenGLES3Bit = 0x0040

// pipeline is how the cell table reaches the GPU and is drawn: es3 or es2.
type pipeline interface {
	// grid makes the GPU's buffers for a grid of cols by rows cells.
	grid(cols, rows int)
	// sheet hands over the rectangle x, y, w, h of the character sheet.
	sheet(sheet []byte, x, y, w, h int)
	// frame hands over the bytes lo <= i < hi of the table, which changed,
	// draws it and shows it.
	frame(table []byte, lo, hi int) error
}

// Screen is the window and what OpenGL holds for it.
type Screen struct {
	ctx     C.screen_context_t
	win     C.screen_window_t
	dpy     C.EGLDisplay
	config  C.EGLConfig
	glctx   C.EGLContext
	surf    C.EGLSurface
	w, h    int
	es      int  // the OpenGL ES version drawing: 3 or 2
	has3    bool // the phone has OpenGL ES 3.0
	prog    *program
	drawing pipeline
	readDir string // where the next frame writes what it drew; "" for none
	read    []byte // that frame's pixels
}

// fail is an error naming the step, with what the three libraries say.
func (s *Screen) fail(what string, errno error) error {
	glErr := C.GLenum(0)
	if s.glctx != nil {
		glErr = C.glGetError()
	}
	return fmt.Errorf("gpu: %s (%v, EGL %#x, GL %#x)", what, errno, uint(C.eglGetError()), uint(glErr))
}

// Open creates the window, w by h pixels, for the page's ForeignWindowControl
// with the given window group and id: a child window in the Cascades
// window's group, EGL, an OpenGL ES context, the program and its buffers.
// es is the OpenGL ES version to draw with, 3 or 2, or 0 for 3.0 where the
// phone has it and 2.0 where it does not. A phone without 3.0 draws with
// 2.0 whatever es says, so that the window never stays blank.
func Open(group, id string, w, h int, es int) (_ *Screen, err error) {
	s := new(Screen)
	// A window that could not be opened leaves nothing behind: no child
	// window in the page's group, which would confuse the next open, and
	// no EGL.
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	cgroup, cid := C.CString(group), C.CString(id)
	defer C.free(unsafe.Pointer(cgroup))
	defer C.free(unsafe.Pointer(cid))

	s.dpy = C.eglGetDisplay(C.EGLNativeDisplayType(0)) // EGL_DEFAULT_DISPLAY
	if s.dpy == 0 || C.eglInitialize(s.dpy, nil, nil) == 0 || C.eglBindAPI(C.EGL_OPENGL_ES_API) == 0 {
		return nil, s.fail("starting EGL", nil)
	}
	s.has3 = s.choose(3)
	if !s.has3 {
		log.Printf("gpu: this phone has no OpenGL ES 3.0 (EGL %#x)", uint(C.eglGetError()))
	}
	switch {
	case s.has3 && es != 2 && s.context(3):
	case s.choose(2) && s.context(2):
	default:
		return nil, s.fail("finding an OpenGL ES 3.0 or 2.0 setup", nil)
	}
	// The window's usage matches: ES 3.0 windows as before this package
	// also drew with 2.0, ES 2.0 alone where that is all there is.
	usage := C.int(C.SCREEN_USAGE_OPENGL_ES2)
	if s.es == 3 {
		usage |= C.SCREEN_USAGE_OPENGL_ES3
	}

	if r, err := C.screen_create_context(&s.ctx, C.SCREEN_APPLICATION_CONTEXT); r != 0 {
		return nil, s.fail("screen_create_context", err)
	}
	if r, err := C.screen_create_window_type(&s.win, s.ctx, C.SCREEN_CHILD_WINDOW); r != 0 {
		return nil, s.fail("screen_create_window_type", err)
	}
	if r, err := C.screen_join_window_group(s.win, cgroup); r != 0 {
		return nil, s.fail("joining the page's window group", err)
	}
	if r, err := C.screen_set_window_property_cv(s.win, C.SCREEN_PROPERTY_ID_STRING, C.int(len(id)), cid); r != 0 {
		return nil, s.fail("setting the window's id", err)
	}
	// Below the Cascades window (Z-order 0): the ForeignWindowControl is a
	// hole in the page through which this window shows.
	for _, p := range []struct{ name, value C.int }{
		{C.SCREEN_PROPERTY_FORMAT, C.SCREEN_FORMAT_RGBX8888},
		{C.SCREEN_PROPERTY_USAGE, usage},
		{C.SCREEN_PROPERTY_ZORDER, -5},
	} {
		value := p.value
		if r, err := C.screen_set_window_property_iv(s.win, p.name, &value); r != 0 {
			return nil, s.fail("setting the window's properties", err)
		}
	}
	if err := s.surface(w, h); err != nil {
		return nil, err
	}

	var maxTex C.GLint
	C.glGetIntegerv(C.GL_MAX_TEXTURE_SIZE, &maxTex)
	if maxTex < SheetSize {
		return nil, fmt.Errorf("gpu: this phone's GPU takes textures of at most %d pixels across; the character sheet is %d", int(maxTex), SheetSize)
	}
	if s.es == 3 {
		if s.prog, err = link(es3Vertex, es3Fragment, es3Attributes); err != nil {
			return nil, err
		}
		s.drawing = newES3(s)
	} else {
		// Before linking: without full precision the two shaders' u_cell
		// differ, and the link would fail with the driver's words instead.
		if err := checkES2(); err != nil {
			return nil, err
		}
		if s.prog, err = link(es2Vertex, es2Fragment, es2Attributes); err != nil {
			return nil, err
		}
		s.drawing = newES2(s)
	}
	log.Printf("gpu: drawing with %s", s.Drawing())

	s.viewport()
	visible := C.int(1)
	if r, err := C.screen_set_window_property_iv(s.win, C.SCREEN_PROPERTY_VISIBLE, &visible); r != 0 {
		return nil, s.fail("showing the window", err)
	}
	return s, nil
}

// choose finds an EGL setup for OpenGL ES version es (3 or 2); false when
// the phone has none. A phone without ES 3.0 can refuse the ES 3.0 setup
// outright (the Classic answers EGL_BAD_ATTRIBUTE); the caller reads that
// error, so that the next call starts clean.
func (s *Screen) choose(es int) bool {
	renderable := C.EGLint(C.EGL_OPENGL_ES2_BIT)
	if es == 3 {
		renderable = eglOpenGLES3Bit
	}
	attribs := []C.EGLint{C.EGL_RED_SIZE, 8, C.EGL_GREEN_SIZE, 8, C.EGL_BLUE_SIZE, 8,
		C.EGL_SURFACE_TYPE, C.EGL_WINDOW_BIT, C.EGL_RENDERABLE_TYPE, renderable, C.EGL_NONE}
	var n C.EGLint
	return C.eglChooseConfig(s.dpy, &attribs[0], &s.config, 1, &n) != 0 && n > 0
}

// context makes a context of OpenGL ES version es on the setup choose
// found last; false when the phone refuses it.
func (s *Screen) context(es int) bool {
	version := []C.EGLint{C.EGL_CONTEXT_CLIENT_VERSION, C.EGLint(es), C.EGL_NONE}
	s.glctx = C.eglCreateContext(s.dpy, s.config, nil, &version[0])
	if s.glctx == nil {
		log.Printf("gpu: no OpenGL ES %d.0 context (EGL %#x)", es, uint(C.eglGetError()))
		return false
	}
	s.es = es
	return true
}

// Drawing names the OpenGL ES version the window draws with.
func (s *Screen) Drawing() string {
	return fmt.Sprintf("OpenGL ES %d.0", s.es)
}

// HasES3 says whether the phone has OpenGL ES 3.0, so that a choice of
// version means anything.
func (s *Screen) HasES3() bool { return s.has3 }

// surface makes the window's buffers and the EGL surface on them, for w by h
// pixels, and makes the context current on it.
func (s *Screen) surface(w, h int) error {
	size := [2]C.int{C.int(w), C.int(h)}
	if r, err := C.screen_set_window_property_iv(s.win, C.SCREEN_PROPERTY_BUFFER_SIZE, &size[0]); r != 0 {
		return s.fail("setting the window's buffer size", err)
	}
	if r, err := C.screen_create_window_buffers(s.win, 2); r != 0 {
		return s.fail("creating the window's buffers", err)
	}
	// The part of the buffer shown is all of it. The system sets this with
	// a window's first buffers only: after a resize it stays the old size.
	if r, err := C.screen_set_window_property_iv(s.win, C.SCREEN_PROPERTY_SOURCE_SIZE, &size[0]); r != 0 {
		return s.fail("setting the part of the buffer shown", err)
	}
	s.surf = C.eglCreateWindowSurface(s.dpy, s.config, C.EGLNativeWindowType(unsafe.Pointer(s.win)), nil)
	if s.surf == nil {
		return s.fail("eglCreateWindowSurface", nil)
	}
	if C.eglMakeCurrent(s.dpy, s.surf, s.surf, s.glctx) == 0 {
		return s.fail("eglMakeCurrent", nil)
	}
	s.w, s.h = w, h
	return nil
}

func (s *Screen) viewport() {
	C.glViewport(0, 0, C.GLsizei(s.w), C.GLsizei(s.h))
	s.prog.view(s.w, s.h)
}

// Resize gives the window a new size in pixels: its buffers and the EGL
// surface are made again.
func (s *Screen) Resize(w, h int) error {
	C.eglMakeCurrent(s.dpy, nil, nil, nil)
	C.eglDestroySurface(s.dpy, s.surf)
	if r, err := C.screen_destroy_window_buffers(s.win); r != 0 {
		return s.fail("destroying the window's buffers", err)
	}
	if err := s.surface(w, h); err != nil {
		return err
	}
	s.viewport()
	return nil
}

// DPI is the dots per inch of the display the window is on.
func (s *Screen) DPI() int {
	var display unsafe.Pointer
	var dpi C.int
	C.screen_get_window_property_pv(s.win, C.SCREEN_PROPERTY_DISPLAY, &display)
	C.screen_get_display_property_iv(C.screen_display_t(display), C.SCREEN_PROPERTY_DPI, &dpi)
	return int(dpi)
}

// Grid sets the grid: cols by rows cells of cellW by cellH pixels, and the
// colour of the window outside the grid. The next Frame must hand over the
// whole table.
func (s *Screen) Grid(cols, rows, cellW, cellH int, r, g, b uint8) {
	s.prog.grid(cols, rows, cellW, cellH)
	s.drawing.grid(cols, rows)
	C.glClearColor(C.GLfloat(r)/255, C.GLfloat(g)/255, C.GLfloat(b)/255, 1)
}

// Sheet hands over the rectangle x, y, w, h of the character sheet; sheet is
// the whole picture, SheetSize*SheetSize bytes.
func (s *Screen) Sheet(sheet []byte, x, y, w, h int) {
	s.drawing.sheet(sheet, x, y, w, h)
}

// Frame draws the table and shows it; the bytes lo <= i < hi of table have
// changed since the last frame. It is the one call from Go into C per frame.
// The table is never empty: the settings limit the font size so that even
// the largest font has a row in the phone's smallest window (the one with
// the on-screen keyboard rows up).
func (s *Screen) Frame(table []byte, lo, hi int) error {
	err := s.drawing.frame(table, lo, hi)
	if s.readDir != "" {
		name := filepath.Join(s.readDir, fmt.Sprintf("gpu-readback-es%d-%dx%d.rgba", s.es, s.w, s.h))
		if werr := os.WriteFile(name, s.read, 0o644); werr != nil {
			log.Printf("gpu: %v", werr)
		} else {
			log.Printf("gpu: wrote %s", name)
		}
		s.readDir, s.read = "", nil
	}
	return err
}

// ReadNext makes the next frame write the pixels it drew to the folder dir,
// for comparing what the two OpenGL ES versions draw: raw RGBA, 4 bytes a
// pixel, the bottom row first, in gpu-readback-es<version>-<w>x<h>.rgba.
func (s *Screen) ReadNext(dir string) {
	s.readDir = dir
}

// readTarget is where the frame being drawn reads its pixels to, and their
// size; nil when it reads none.
func (s *Screen) readTarget() (*C.uchar, C.int, C.int) {
	if s.readDir == "" {
		return nil, 0, 0
	}
	// Made here, at the window's size now, which a resize since ReadNext
	// may have changed.
	s.read = make([]byte, s.w*s.h*4)
	return (*C.uchar)(unsafe.Pointer(&s.read[0])), C.int(s.w), C.int(s.h)
}

// Close removes the window, and whatever of it Open had made when it
// failed.
func (s *Screen) Close() {
	if s.dpy != 0 {
		C.eglMakeCurrent(s.dpy, nil, nil, nil)
		if s.surf != nil {
			C.eglDestroySurface(s.dpy, s.surf)
		}
		if s.glctx != nil {
			C.eglDestroyContext(s.dpy, s.glctx)
		}
		C.eglTerminate(s.dpy)
	}
	if s.win != nil {
		C.screen_destroy_window(s.win)
	}
	if s.ctx != nil {
		C.screen_destroy_context(s.ctx)
	}
}
