//go:build qnx

package gpu

/*
#include <EGL/egl.h>
#include <GLES3/gl3.h>

// One frame: the bytes lo <= i < hi of the table are new. Returns 0 when
// the swap failed.
static int t51_frame3(EGLDisplay dpy, EGLSurface surf, const unsigned char *table, int lo, int hi, int cells) {
	glClear(GL_COLOR_BUFFER_BIT);
	if (hi > lo)
		glBufferSubData(GL_ARRAY_BUFFER, lo, hi - lo, table + lo);
	glDrawArraysInstanced(GL_TRIANGLE_STRIP, 0, 4, cells);
	return eglSwapBuffers(dpy, surf);
}
*/
import "C"

import "unsafe"

// es3 draws the grid with OpenGL ES 3.0: one rectangle drawn once per cell
// (instancing), each copy reading its cell's record from the table, which
// goes to the GPU as it is.
type es3 struct {
	s     *Screen
	cells int
}

// es3Attributes are es3.vert's attributes, in the order of their places.
var es3Attributes = []string{"a_corner", "a_what", "a_fg", "a_bg", "a_line"}

// newES3 sets up the buffers and the character sheet; the program is
// current.
func newES3(s *Screen) *es3 {
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

	// The character sheet, empty until sheet is called.
	C.glGenTextures(1, &tex)
	C.glActiveTexture(C.GL_TEXTURE0)
	C.glBindTexture(C.GL_TEXTURE_2D, tex)
	C.glTexStorage2D(C.GL_TEXTURE_2D, 1, C.GL_R8, SheetSize, SheetSize)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_NEAREST)
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 1)
	C.glPixelStorei(C.GL_UNPACK_ROW_LENGTH, SheetSize)
	return &es3{s: s}
}

func (p *es3) grid(cols, rows int) {
	p.cells = cols * rows
	C.glBufferData(C.GL_ARRAY_BUFFER, C.GLsizeiptr(p.cells*RecordSize), nil, C.GL_DYNAMIC_DRAW)
}

func (p *es3) sheet(sheet []byte, x, y, w, h int) {
	C.glTexSubImage2D(C.GL_TEXTURE_2D, 0, C.GLint(x), C.GLint(y), C.GLsizei(w), C.GLsizei(h),
		C.GL_RED, C.GL_UNSIGNED_BYTE, unsafe.Pointer(&sheet[y*SheetSize+x]))
}

func (p *es3) frame(table []byte, lo, hi int) error {
	if C.t51_frame3(p.s.dpy, p.s.surf, (*C.uchar)(unsafe.Pointer(&table[0])), C.int(lo), C.int(hi), C.int(p.cells)) == 0 {
		return p.s.fail("eglSwapBuffers", nil)
	}
	return nil
}
