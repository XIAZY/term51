//go:build qnx

// Package gpu is the screen: a window of our own inside the Cascades page,
// drawn with OpenGL ES 3.0. It knows nothing of the terminal. It is given a
// table with one record per cell and a sheet of character shapes, and draws
// every cell from them in one draw call.
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
// Go crosses into C once per frame (Frame), and once more only when part of
// the sheet is new (Sheet). The frame is the one piece written in C
// (t51_frame below), so that it is one crossing and not four. Everything
// else, which runs once or on a resize, is Go calling the window system's,
// EGL's and OpenGL's functions directly. Everything must be called from one
// goroutine locked to its thread: an OpenGL context belongs to a thread.
package gpu

/*
#cgo LDFLAGS: -lscreen -lEGL -lGLESv2
#include <stdlib.h>
#include <screen/screen.h>
#include <EGL/egl.h>
#include <GLES3/gl3.h>

// One frame: the bytes lo <= i < hi of the table are new. Returns 0 when
// the swap failed.
static int t51_frame(EGLDisplay dpy, EGLSurface surf, const unsigned char *table, int lo, int hi, int cells) {
	glClear(GL_COLOR_BUFFER_BIT);
	if (hi > lo)
		glBufferSubData(GL_ARRAY_BUFFER, lo, hi - lo, table + lo);
	glDrawArraysInstanced(GL_TRIANGLE_STRIP, 0, 4, cells);
	return eglSwapBuffers(dpy, surf);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// SheetSize is the width and height of the character sheet in pixels.
const SheetSize = 2048

// RecordSize is the size of one cell's record in the table.
const RecordSize = 16

const vertexShader = `#version 300 es
layout(location = 0) in vec2 a_corner; // of the cell: 0,0 1,0 0,1 1,1
layout(location = 1) in uvec2 a_what;  // slot, flags
layout(location = 2) in vec4 a_fg;
layout(location = 3) in vec4 a_bg;
layout(location = 4) in vec4 a_line;
uniform ivec2 u_grid;      // columns, rows
uniform vec2 u_cell;       // a cell in pixels
uniform vec2 u_view;       // the window in pixels
uniform int u_sheet_cols;  // slots per row of the sheet
out vec4 v_fg;
out vec4 v_bg;
out vec4 v_line;
out vec2 v_local;          // the pixel inside the cell
flat out ivec2 v_origin;   // the slot's corner in the sheet
flat out uint v_slot;
flat out uint v_flags;
void main() {
	vec2 at = vec2(float(gl_InstanceID % u_grid.x), float(gl_InstanceID / u_grid.x));
	vec2 p = (at + a_corner) * u_cell / u_view * 2.0 - 1.0;
	gl_Position = vec4(p.x, -p.y, 0.0, 1.0); // record 0 is the top left
	v_fg = a_fg;
	v_bg = a_bg;
	v_line = a_line;
	v_local = a_corner * u_cell;
	int slot = int(a_what.x);
	v_origin = ivec2(slot % u_sheet_cols, slot / u_sheet_cols) * ivec2(u_cell);
	v_slot = a_what.x;
	v_flags = a_what.y;
}
`

const fragmentShader = `#version 300 es
precision mediump float;
uniform mediump sampler2D u_sheet;
uniform vec2 u_cell;
uniform float u_thick;     // the thickness of lines, in pixels
in vec4 v_fg;
in vec4 v_bg;
in vec4 v_line;
in vec2 v_local;
flat in ivec2 v_origin;
flat in uint v_slot;
flat in uint v_flags;
out vec4 o_col;
void main() {
	float a = 0.0;
	if (v_slot != 0u)
		a = texelFetch(u_sheet, v_origin + ivec2(v_local), 0).r;
	float x = v_local.x, y = v_local.y, t = u_thick, line = 0.0;
	if ((v_flags & 2u) != 0u && y >= floor(u_cell.y * 0.55) && y < floor(u_cell.y * 0.55) + t) a = 1.0; // strikethrough
	if ((v_flags & 1u) != 0u && y >= u_cell.y - 2.0 * t && y < u_cell.y - t) line = 1.0;                // underline
	if ((v_flags & 4u) != 0u && x < 2.0 * t) line = 1.0;                                                 // cursor bar
	if ((v_flags & 8u) != 0u && y >= u_cell.y - 2.0 * t) line = 1.0;                                     // cursor underline
	if ((v_flags & 16u) != 0u && (x < t || y < t || x >= u_cell.x - t || y >= u_cell.y - t)) line = 1.0; // hollow cursor box
	o_col = vec4(mix(mix(v_bg.rgb, v_fg.rgb, a), v_line.rgb, line), 1.0);
}
`

// EGL_OPENGL_ES3_BIT_KHR, which the phone's egl.h does not name.
const eglOpenGLES3Bit = 0x0040

// Screen is the window and what OpenGL holds for it.
type Screen struct {
	ctx    C.screen_context_t
	win    C.screen_window_t
	dpy    C.EGLDisplay
	config C.EGLConfig
	glctx  C.EGLContext
	surf   C.EGLSurface
	w, h   int
	cells  int

	uGrid, uCell, uView, uSheetCols, uThick C.GLint
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
// window's group, EGL, an OpenGL ES 3.0 context, the program and its buffers.
func Open(group, id string, w, h int) (*Screen, error) {
	s := new(Screen)
	cgroup, cid := C.CString(group), C.CString(id)
	defer C.free(unsafe.Pointer(cgroup))
	defer C.free(unsafe.Pointer(cid))

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
		{C.SCREEN_PROPERTY_USAGE, C.SCREEN_USAGE_OPENGL_ES2 | C.SCREEN_USAGE_OPENGL_ES3},
		{C.SCREEN_PROPERTY_ZORDER, -5},
	} {
		value := p.value
		if r, err := C.screen_set_window_property_iv(s.win, p.name, &value); r != 0 {
			return nil, s.fail("setting the window's properties", err)
		}
	}

	s.dpy = C.eglGetDisplay(C.EGLNativeDisplayType(0)) // EGL_DEFAULT_DISPLAY
	if s.dpy == 0 || C.eglInitialize(s.dpy, nil, nil) == 0 || C.eglBindAPI(C.EGL_OPENGL_ES_API) == 0 {
		return nil, s.fail("starting EGL", nil)
	}
	attribs := []C.EGLint{C.EGL_RED_SIZE, 8, C.EGL_GREEN_SIZE, 8, C.EGL_BLUE_SIZE, 8,
		C.EGL_SURFACE_TYPE, C.EGL_WINDOW_BIT, C.EGL_RENDERABLE_TYPE, eglOpenGLES3Bit, C.EGL_NONE}
	var n C.EGLint
	if C.eglChooseConfig(s.dpy, &attribs[0], &s.config, 1, &n) == 0 || n == 0 {
		return nil, s.fail("eglChooseConfig", nil)
	}
	version := []C.EGLint{C.EGL_CONTEXT_CLIENT_VERSION, 3, C.EGL_NONE}
	s.glctx = C.eglCreateContext(s.dpy, s.config, nil, &version[0])
	if s.glctx == nil {
		return nil, s.fail("creating an OpenGL ES 3.0 context", nil)
	}
	if err := s.surface(w, h); err != nil {
		return nil, err
	}

	prog := C.glCreateProgram()
	for _, sh := range []struct {
		kind C.GLenum
		src  string
	}{{C.GL_VERTEX_SHADER, vertexShader}, {C.GL_FRAGMENT_SHADER, fragmentShader}} {
		id, err := compile(sh.kind, sh.src)
		if err != nil {
			return nil, err
		}
		C.glAttachShader(prog, id)
	}
	C.glLinkProgram(prog)
	var ok C.GLint
	C.glGetProgramiv(prog, C.GL_LINK_STATUS, &ok)
	if ok == 0 {
		var log [512]C.GLchar
		C.glGetProgramInfoLog(prog, C.GLsizei(len(log)), nil, &log[0])
		return nil, fmt.Errorf("gpu: linking the program: %s", C.GoString((*C.char)(&log[0])))
	}
	C.glUseProgram(prog)
	uniform := func(name string) C.GLint {
		cname := C.CString(name)
		defer C.free(unsafe.Pointer(cname))
		return C.glGetUniformLocation(prog, (*C.GLchar)(cname))
	}
	s.uGrid, s.uCell, s.uView = uniform("u_grid"), uniform("u_cell"), uniform("u_view")
	s.uSheetCols, s.uThick = uniform("u_sheet_cols"), uniform("u_thick")
	C.glUniform1i(uniform("u_sheet"), 0)

	// A cell's four corners, the same for every cell.
	var vao, quad, buf, tex C.GLuint
	C.glGenVertexArrays(1, &vao)
	C.glBindVertexArray(vao)
	corners := [8]uint8{0, 0, 1, 0, 0, 1, 1, 1}
	C.glGenBuffers(1, &quad)
	C.glBindBuffer(C.GL_ARRAY_BUFFER, quad)
	C.glBufferData(C.GL_ARRAY_BUFFER, C.GLsizeiptr(len(corners)), unsafe.Pointer(&corners[0]), C.GL_STATIC_DRAW)
	C.glEnableVertexAttribArray(0)
	C.glVertexAttribPointer(0, 2, C.GL_UNSIGNED_BYTE, C.GL_FALSE, 2, nil)

	// The cell table: one record per instance. Its buffer stays bound.
	C.glGenBuffers(1, &buf)
	C.glBindBuffer(C.GL_ARRAY_BUFFER, buf)
	C.glEnableVertexAttribArray(1)
	C.glVertexAttribIPointer(1, 2, C.GL_UNSIGNED_SHORT, RecordSize, nil)
	C.glVertexAttribDivisor(1, 1)
	for i, offset := range []int{4, 8, 12} { // text, background and line colours
		at := C.GLuint(i + 2)
		C.glEnableVertexAttribArray(at)
		C.glVertexAttribPointer(at, 4, C.GL_UNSIGNED_BYTE, C.GL_TRUE, RecordSize, unsafe.Add(nil, offset))
		C.glVertexAttribDivisor(at, 1)
	}

	// The character sheet, empty until Sheet is called.
	C.glGenTextures(1, &tex)
	C.glActiveTexture(C.GL_TEXTURE0)
	C.glBindTexture(C.GL_TEXTURE_2D, tex)
	C.glTexStorage2D(C.GL_TEXTURE_2D, 1, C.GL_R8, SheetSize, SheetSize)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_NEAREST)
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 1)
	C.glPixelStorei(C.GL_UNPACK_ROW_LENGTH, SheetSize)

	s.viewport()
	visible := C.int(1)
	if r, err := C.screen_set_window_property_iv(s.win, C.SCREEN_PROPERTY_VISIBLE, &visible); r != 0 {
		return nil, s.fail("showing the window", err)
	}
	return s, nil
}

func compile(kind C.GLenum, src string) (C.GLuint, error) {
	id := C.glCreateShader(kind)
	csrc := (*C.GLchar)(C.CString(src))
	defer C.free(unsafe.Pointer(csrc))
	C.glShaderSource(id, 1, &csrc, nil)
	C.glCompileShader(id)
	var ok C.GLint
	C.glGetShaderiv(id, C.GL_COMPILE_STATUS, &ok)
	if ok == 0 {
		var log [512]C.GLchar
		C.glGetShaderInfoLog(id, C.GLsizei(len(log)), nil, &log[0])
		return 0, fmt.Errorf("gpu: compiling a shader: %s", C.GoString((*C.char)(&log[0])))
	}
	return id, nil
}

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
	C.glUniform2f(s.uView, C.GLfloat(s.w), C.GLfloat(s.h))
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
	s.cells = cols * rows
	C.glUniform2i(s.uGrid, C.GLint(cols), C.GLint(rows))
	C.glUniform2f(s.uCell, C.GLfloat(cellW), C.GLfloat(cellH))
	C.glUniform1i(s.uSheetCols, C.GLint(SheetSize/cellW))
	C.glUniform1f(s.uThick, C.GLfloat(max(cellH/14, 1)))
	C.glBufferData(C.GL_ARRAY_BUFFER, C.GLsizeiptr(s.cells*RecordSize), nil, C.GL_DYNAMIC_DRAW)
	C.glClearColor(C.GLfloat(r)/255, C.GLfloat(g)/255, C.GLfloat(b)/255, 1)
}

// Sheet hands over the rectangle x, y, w, h of the character sheet; sheet is
// the whole picture, SheetSize*SheetSize bytes.
func (s *Screen) Sheet(sheet []byte, x, y, w, h int) {
	C.glTexSubImage2D(C.GL_TEXTURE_2D, 0, C.GLint(x), C.GLint(y), C.GLsizei(w), C.GLsizei(h),
		C.GL_RED, C.GL_UNSIGNED_BYTE, unsafe.Pointer(&sheet[y*SheetSize+x]))
}

// Frame draws the table and shows it; the bytes lo <= i < hi of table have
// changed since the last frame. It is the one call from Go into C per frame.
// The table is never empty: the settings limit the font size so that even
// the largest font has a row in the phone's smallest window (the one with
// the on-screen keyboard rows up).
func (s *Screen) Frame(table []byte, lo, hi int) error {
	if C.t51_frame(s.dpy, s.surf, (*C.uchar)(unsafe.Pointer(&table[0])), C.int(lo), C.int(hi), C.int(s.cells)) == 0 {
		return s.fail("eglSwapBuffers", nil)
	}
	return nil
}

// Close removes the window.
func (s *Screen) Close() {
	C.eglMakeCurrent(s.dpy, nil, nil, nil)
	C.eglDestroySurface(s.dpy, s.surf)
	C.eglDestroyContext(s.dpy, s.glctx)
	C.eglTerminate(s.dpy)
	C.screen_destroy_window(s.win)
	C.screen_destroy_context(s.ctx)
}
