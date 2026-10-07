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
// the sheet is new (Sheet). Everything must be called from one goroutine
// locked to its thread: an OpenGL context belongs to a thread.
package gpu

/*
#cgo LDFLAGS: -lscreen -lEGL -lGLESv2
#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <screen/screen.h>
#include <EGL/egl.h>
#include <GLES3/gl3.h>

#define T51_SHEET 2048

typedef struct t51_gpu {
	screen_context_t ctx;
	screen_window_t win;
	EGLDisplay dpy;
	EGLConfig config;
	EGLContext glctx;
	EGLSurface surf;
	GLuint prog, buf, tex, vao;
	GLint u_grid, u_cell, u_view, u_sheet_cols, u_thick;
	int w, h, cells;
	char err[256];
} t51_gpu;

static const char *t51_vs =
	"#version 300 es\n"
	"layout(location = 0) in uvec2 a_what;\n" // slot, flags
	"layout(location = 1) in vec4 a_fg;\n"
	"layout(location = 2) in vec4 a_bg;\n"
	"layout(location = 3) in vec4 a_line;\n"
	"uniform ivec2 u_grid;\n"       // columns, rows
	"uniform vec2 u_cell;\n"        // a cell in pixels
	"uniform vec2 u_view;\n"        // the window in pixels
	"uniform int u_sheet_cols;\n"   // slots per row of the sheet
	"out vec4 v_fg;\n"
	"out vec4 v_bg;\n"
	"out vec4 v_line;\n"
	"out vec2 v_local;\n"           // the pixel inside the cell
	"flat out ivec2 v_origin;\n"    // the slot's corner in the sheet
	"flat out uint v_slot;\n"
	"flat out uint v_flags;\n"
	"void main() {\n"
	"	vec2 corner = vec2(float(gl_VertexID & 1), float(gl_VertexID >> 1));\n"
	"	vec2 at = vec2(float(gl_InstanceID % u_grid.x), float(gl_InstanceID / u_grid.x));\n"
	"	vec2 p = (at + corner) * u_cell / u_view * 2.0 - 1.0;\n"
	"	gl_Position = vec4(p.x, -p.y, 0.0, 1.0);\n" // record 0 is the top left
	"	v_fg = a_fg;\n"
	"	v_bg = a_bg;\n"
	"	v_line = a_line;\n"
	"	v_local = corner * u_cell;\n"
	"	int slot = int(a_what.x);\n"
	"	v_origin = ivec2(slot % u_sheet_cols, slot / u_sheet_cols) * ivec2(u_cell);\n"
	"	v_slot = a_what.x;\n"
	"	v_flags = a_what.y;\n"
	"}\n";

static const char *t51_fs =
	"#version 300 es\n"
	"precision mediump float;\n"
	"uniform mediump sampler2D u_sheet;\n"
	"uniform vec2 u_cell;\n"
	"uniform float u_thick;\n"      // the thickness of lines, in pixels
	"in vec4 v_fg;\n"
	"in vec4 v_bg;\n"
	"in vec4 v_line;\n"
	"in vec2 v_local;\n"
	"flat in ivec2 v_origin;\n"
	"flat in uint v_slot;\n"
	"flat in uint v_flags;\n"
	"out vec4 o_col;\n"
	"void main() {\n"
	"	float a = 0.0;\n"
	"	if (v_slot != 0u)\n"
	"		a = texelFetch(u_sheet, v_origin + ivec2(v_local), 0).r;\n"
	"	float x = v_local.x, y = v_local.y, t = u_thick, line = 0.0;\n"
	"	if ((v_flags & 2u) != 0u && y >= floor(u_cell.y * 0.55) && y < floor(u_cell.y * 0.55) + t) a = 1.0;\n" // strikethrough
	"	if ((v_flags & 1u) != 0u && y >= u_cell.y - 2.0 * t && y < u_cell.y - t) line = 1.0;\n"          // underline
	"	if ((v_flags & 4u) != 0u && x < 2.0 * t) line = 1.0;\n"                                           // cursor bar
	"	if ((v_flags & 8u) != 0u && y >= u_cell.y - 2.0 * t) line = 1.0;\n"                               // cursor underline
	"	if ((v_flags & 16u) != 0u && (x < t || y < t || x >= u_cell.x - t || y >= u_cell.y - t)) line = 1.0;\n" // hollow cursor box
	"	o_col = vec4(mix(mix(v_bg.rgb, v_fg.rgb, a), v_line.rgb, line), 1.0);\n"
	"}\n";

static int t51_fail(t51_gpu *g, const char *what) {
	snprintf(g->err, sizeof g->err, "%s (errno %d, EGL %#x, GL %#x)", what, errno, (unsigned)eglGetError(),
		g->glctx != EGL_NO_CONTEXT ? (unsigned)glGetError() : 0u);
	return -1;
}

static GLuint t51_shader(t51_gpu *g, GLenum kind, const char *src) {
	GLuint s = glCreateShader(kind);
	glShaderSource(s, 1, &src, NULL);
	glCompileShader(s);
	GLint ok = 0;
	glGetShaderiv(s, GL_COMPILE_STATUS, &ok);
	if (!ok) {
		char log[200];
		glGetShaderInfoLog(s, sizeof log, NULL, log);
		snprintf(g->err, sizeof g->err, "compiling a shader: %s", log);
		return 0;
	}
	return s;
}

// The window's buffers and the EGL surface on them, for w by h pixels.
static int t51_surface(t51_gpu *g, int w, int h) {
	int size[2] = {w, h};
	if (screen_set_window_property_iv(g->win, SCREEN_PROPERTY_BUFFER_SIZE, size) != 0)
		return t51_fail(g, "setting the window's buffer size");
	if (screen_create_window_buffers(g->win, 2) != 0)
		return t51_fail(g, "creating the window's buffers");
	g->surf = eglCreateWindowSurface(g->dpy, g->config, g->win, NULL);
	if (g->surf == EGL_NO_SURFACE)
		return t51_fail(g, "eglCreateWindowSurface");
	if (!eglMakeCurrent(g->dpy, g->surf, g->surf, g->glctx))
		return t51_fail(g, "eglMakeCurrent");
	g->w = w;
	g->h = h;
	return 0;
}

// Creates the window in the Cascades window's group, with the id the page's
// ForeignWindowControl looks for, and everything OpenGL needs.
static int t51_gpu_open(t51_gpu *g, const char *group, const char *id, int w, int h) {
	if (screen_create_context(&g->ctx, SCREEN_APPLICATION_CONTEXT) != 0)
		return t51_fail(g, "screen_create_context");
	if (screen_create_window_type(&g->win, g->ctx, SCREEN_CHILD_WINDOW) != 0)
		return t51_fail(g, "screen_create_window_type");
	if (screen_join_window_group(g->win, group) != 0)
		return t51_fail(g, "joining the page's window group");
	if (screen_set_window_property_cv(g->win, SCREEN_PROPERTY_ID_STRING, strlen(id), id) != 0)
		return t51_fail(g, "setting the window's id");
	int format = SCREEN_FORMAT_RGBX8888;
	int usage = SCREEN_USAGE_OPENGL_ES2 | SCREEN_USAGE_OPENGL_ES3;
	// Below the Cascades window (Z-order 0): the ForeignWindowControl is a
	// hole in the page through which this window shows.
	int z = -5;
	if (screen_set_window_property_iv(g->win, SCREEN_PROPERTY_FORMAT, &format) != 0 ||
		screen_set_window_property_iv(g->win, SCREEN_PROPERTY_USAGE, &usage) != 0 ||
		screen_set_window_property_iv(g->win, SCREEN_PROPERTY_ZORDER, &z) != 0)
		return t51_fail(g, "setting the window's properties");

	g->dpy = eglGetDisplay(EGL_DEFAULT_DISPLAY);
	if (g->dpy == EGL_NO_DISPLAY || !eglInitialize(g->dpy, NULL, NULL) || !eglBindAPI(EGL_OPENGL_ES_API))
		return t51_fail(g, "starting EGL");
	EGLint attribs[] = {EGL_RED_SIZE, 8, EGL_GREEN_SIZE, 8, EGL_BLUE_SIZE, 8,
		EGL_SURFACE_TYPE, EGL_WINDOW_BIT, EGL_RENDERABLE_TYPE, 0x0040, EGL_NONE}; // EGL_OPENGL_ES3_BIT_KHR
	EGLint n = 0;
	if (!eglChooseConfig(g->dpy, attribs, &g->config, 1, &n) || n == 0)
		return t51_fail(g, "eglChooseConfig");
	EGLint version[] = {EGL_CONTEXT_CLIENT_VERSION, 3, EGL_NONE};
	g->glctx = eglCreateContext(g->dpy, g->config, EGL_NO_CONTEXT, version);
	if (g->glctx == EGL_NO_CONTEXT)
		return t51_fail(g, "creating an OpenGL ES 3.0 context");
	if (t51_surface(g, w, h) != 0)
		return -1;

	GLuint vs = t51_shader(g, GL_VERTEX_SHADER, t51_vs), fs = t51_shader(g, GL_FRAGMENT_SHADER, t51_fs);
	if (!vs || !fs)
		return -1;
	g->prog = glCreateProgram();
	glAttachShader(g->prog, vs);
	glAttachShader(g->prog, fs);
	glLinkProgram(g->prog);
	GLint ok = 0;
	glGetProgramiv(g->prog, GL_LINK_STATUS, &ok);
	if (!ok) {
		char log[200];
		glGetProgramInfoLog(g->prog, sizeof log, NULL, log);
		snprintf(g->err, sizeof g->err, "linking the program: %s", log);
		return -1;
	}
	glUseProgram(g->prog);
	g->u_grid = glGetUniformLocation(g->prog, "u_grid");
	g->u_cell = glGetUniformLocation(g->prog, "u_cell");
	g->u_view = glGetUniformLocation(g->prog, "u_view");
	g->u_sheet_cols = glGetUniformLocation(g->prog, "u_sheet_cols");
	g->u_thick = glGetUniformLocation(g->prog, "u_thick");
	glUniform1i(glGetUniformLocation(g->prog, "u_sheet"), 0);

	// The cell table: one record per instance.
	glGenVertexArrays(1, &g->vao);
	glBindVertexArray(g->vao);
	glGenBuffers(1, &g->buf);
	glBindBuffer(GL_ARRAY_BUFFER, g->buf);
	glEnableVertexAttribArray(0);
	glVertexAttribIPointer(0, 2, GL_UNSIGNED_SHORT, 16, (void *)0);
	glVertexAttribDivisor(0, 1);
	glEnableVertexAttribArray(1);
	glVertexAttribPointer(1, 4, GL_UNSIGNED_BYTE, GL_TRUE, 16, (void *)4);
	glVertexAttribDivisor(1, 1);
	glEnableVertexAttribArray(2);
	glVertexAttribPointer(2, 4, GL_UNSIGNED_BYTE, GL_TRUE, 16, (void *)8);
	glVertexAttribDivisor(2, 1);
	glEnableVertexAttribArray(3);
	glVertexAttribPointer(3, 4, GL_UNSIGNED_BYTE, GL_TRUE, 16, (void *)12);
	glVertexAttribDivisor(3, 1);

	// The character sheet, empty until Sheet is called.
	glGenTextures(1, &g->tex);
	glActiveTexture(GL_TEXTURE0);
	glBindTexture(GL_TEXTURE_2D, g->tex);
	glTexStorage2D(GL_TEXTURE_2D, 1, GL_R8, T51_SHEET, T51_SHEET);
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
	glPixelStorei(GL_UNPACK_ALIGNMENT, 1);
	glPixelStorei(GL_UNPACK_ROW_LENGTH, T51_SHEET);

	glViewport(0, 0, w, h);
	glUniform2f(g->u_view, (float)w, (float)h);
	int visible = 1;
	if (screen_set_window_property_iv(g->win, SCREEN_PROPERTY_VISIBLE, &visible) != 0)
		return t51_fail(g, "showing the window");
	return 0;
}

// The window's buffers again, for a new size.
static int t51_gpu_resize(t51_gpu *g, int w, int h) {
	eglMakeCurrent(g->dpy, EGL_NO_SURFACE, EGL_NO_SURFACE, EGL_NO_CONTEXT);
	eglDestroySurface(g->dpy, g->surf);
	if (screen_destroy_window_buffers(g->win) != 0)
		return t51_fail(g, "destroying the window's buffers");
	if (t51_surface(g, w, h) != 0)
		return -1;
	glViewport(0, 0, w, h);
	glUniform2f(g->u_view, (float)w, (float)h);
	return 0;
}

// The dots per inch of the display the window is on.
static int t51_gpu_dpi(t51_gpu *g) {
	screen_display_t display = NULL;
	int dpi = 0;
	screen_get_window_property_pv(g->win, SCREEN_PROPERTY_DISPLAY, (void **)&display);
	screen_get_display_property_iv(display, SCREEN_PROPERTY_DPI, &dpi);
	return dpi;
}

// The grid's shape and the colour of what lies outside it; the table's
// storage on the GPU.
static void t51_gpu_grid(t51_gpu *g, int cols, int rows, int cell_w, int cell_h, int r, int gr, int b) {
	g->cells = cols * rows;
	glUniform2i(g->u_grid, cols, rows);
	glUniform2f(g->u_cell, (float)cell_w, (float)cell_h);
	glUniform1i(g->u_sheet_cols, T51_SHEET / cell_w);
	int thick = cell_h / 14;
	glUniform1f(g->u_thick, (float)(thick < 1 ? 1 : thick));
	glBufferData(GL_ARRAY_BUFFER, g->cells * 16, NULL, GL_DYNAMIC_DRAW);
	glClearColor(r / 255.0f, gr / 255.0f, b / 255.0f, 1.0f);
}

// A new part of the character sheet: the rectangle x, y, w, h of the whole
// picture at pix.
static void t51_gpu_sheet(t51_gpu *g, const unsigned char *pix, int x, int y, int w, int h) {
	glTexSubImage2D(GL_TEXTURE_2D, 0, x, y, w, h, GL_RED, GL_UNSIGNED_BYTE, pix + (size_t)y * T51_SHEET + x);
}

// One frame: the bytes lo <= i < hi of the table are new. With shot not
// NULL the picture is also read back into it before it is shown: 4 bytes
// (R, G, B, A) per pixel, the bottom row first.
static int t51_gpu_frame(t51_gpu *g, const unsigned char *table, int lo, int hi, unsigned char *shot) {
	glClear(GL_COLOR_BUFFER_BIT);
	if (hi > lo)
		glBufferSubData(GL_ARRAY_BUFFER, lo, hi - lo, table + lo);
	glDrawArraysInstanced(GL_TRIANGLE_STRIP, 0, 4, g->cells);
	if (shot)
		glReadPixels(0, 0, g->w, g->h, GL_RGBA, GL_UNSIGNED_BYTE, shot);
	if (!eglSwapBuffers(g->dpy, g->surf))
		return t51_fail(g, "eglSwapBuffers");
	return 0;
}

static void t51_gpu_close(t51_gpu *g) {
	eglMakeCurrent(g->dpy, EGL_NO_SURFACE, EGL_NO_SURFACE, EGL_NO_CONTEXT);
	eglDestroySurface(g->dpy, g->surf);
	eglDestroyContext(g->dpy, g->glctx);
	eglTerminate(g->dpy);
	screen_destroy_window(g->win);
	screen_destroy_context(g->ctx);
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// SheetSize is the width and height of the character sheet in pixels.
const SheetSize = C.T51_SHEET

// RecordSize is the size of one cell's record in the table.
const RecordSize = 16

// Screen is the window and what OpenGL holds for it.
type Screen struct {
	g C.t51_gpu
}

func (s *Screen) err() error { return errors.New("gpu: " + C.GoString(&s.g.err[0])) }

// Open creates the window, w by h pixels, for the page's ForeignWindowControl
// with the given window group and id.
func Open(group, id string, w, h int) (*Screen, error) {
	s := new(Screen)
	cg, ci := C.CString(group), C.CString(id)
	defer C.free(unsafe.Pointer(cg))
	defer C.free(unsafe.Pointer(ci))
	if C.t51_gpu_open(&s.g, cg, ci, C.int(w), C.int(h)) != 0 {
		return nil, s.err()
	}
	return s, nil
}

// Resize gives the window a new size in pixels.
func (s *Screen) Resize(w, h int) error {
	if C.t51_gpu_resize(&s.g, C.int(w), C.int(h)) != 0 {
		return s.err()
	}
	return nil
}

// DPI is the dots per inch of the display the window is on.
func (s *Screen) DPI() int { return int(C.t51_gpu_dpi(&s.g)) }

// Size is the window's size in pixels.
func (s *Screen) Size() (w, h int) { return int(s.g.w), int(s.g.h) }

// Grid sets the grid: cols by rows cells of cellW by cellH pixels, and the
// colour of the window outside the grid. The next Frame must hand over the
// whole table.
func (s *Screen) Grid(cols, rows, cellW, cellH int, r, g, b uint8) {
	C.t51_gpu_grid(&s.g, C.int(cols), C.int(rows), C.int(cellW), C.int(cellH), C.int(r), C.int(g), C.int(b))
}

// Sheet hands over the rectangle x, y, w, h of the character sheet; sheet is
// the whole picture, SheetSize*SheetSize bytes.
func (s *Screen) Sheet(sheet []byte, x, y, w, h int) {
	C.t51_gpu_sheet(&s.g, (*C.uchar)(unsafe.Pointer(&sheet[0])), C.int(x), C.int(y), C.int(w), C.int(h))
}

// Frame draws the table and shows it; the bytes lo <= i < hi of table have
// changed since the last frame. It is the one call from Go into C per frame.
// A shot that is not nil receives the picture as drawn: 4 bytes (R, G, B,
// A) per pixel of the window, the bottom row first.
func (s *Screen) Frame(table []byte, lo, hi int, shot []byte) error {
	var out *C.uchar
	if shot != nil {
		out = (*C.uchar)(unsafe.Pointer(&shot[0]))
	}
	if C.t51_gpu_frame(&s.g, (*C.uchar)(unsafe.Pointer(&table[0])), C.int(lo), C.int(hi), out) != 0 {
		return s.err()
	}
	return nil
}

// Close removes the window.
func (s *Screen) Close() { C.t51_gpu_close(&s.g) }
