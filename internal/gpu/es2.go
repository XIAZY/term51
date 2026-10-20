//go:build qnx

package gpu

/*
#include <stddef.h>
#include <EGL/egl.h>
#include <GLES2/gl2.h>

// The attributes' places: es2Attributes, in order.
enum { T51_CELL, T51_CORNER, T51_WHAT, T51_FG, T51_BG, T51_LINE };

// Points the attributes at the vertices of the cells from first on, in the
// corner buffer and in the record buffer (vertices.go has their layout).
static void t51_point(GLuint corners, GLuint records, int first) {
	size_t c = (size_t)first * 4 * 8, r = (size_t)first * 4 * 16;
	glBindBuffer(GL_ARRAY_BUFFER, corners);
	glVertexAttribPointer(T51_CELL, 2, GL_UNSIGNED_SHORT, GL_FALSE, 8, (const void *)c);
	glVertexAttribPointer(T51_CORNER, 2, GL_UNSIGNED_BYTE, GL_FALSE, 8, (const void *)(c + 4));
	glBindBuffer(GL_ARRAY_BUFFER, records);
	glVertexAttribPointer(T51_WHAT, 2, GL_UNSIGNED_SHORT, GL_FALSE, 16, (const void *)r);
	glVertexAttribPointer(T51_FG, 4, GL_UNSIGNED_BYTE, GL_TRUE, 16, (const void *)(r + 4));
	glVertexAttribPointer(T51_BG, 4, GL_UNSIGNED_BYTE, GL_TRUE, 16, (const void *)(r + 8));
	glVertexAttribPointer(T51_LINE, 4, GL_UNSIGNED_BYTE, GL_TRUE, 16, (const void *)(r + 12));
}

// One frame: the bytes lo <= i < hi of the record buffer are new, in
// vertices; the grid is drawn in n chunks, chunk i being the chunks[2i+1]
// cells from cell chunks[2i]. With read, the w by h pixels drawn are read
// into it before they are shown. Returns 0 when the swap failed.
static int t51_frame2(EGLDisplay dpy, EGLSurface surf, GLuint corners, GLuint records,
		const unsigned char *vertices, int lo, int hi, const int *chunks, int n,
		unsigned char *read, int w, int h) {
	glClear(GL_COLOR_BUFFER_BIT);
	if (hi > lo) {
		glBindBuffer(GL_ARRAY_BUFFER, records);
		glBufferSubData(GL_ARRAY_BUFFER, lo, hi - lo, vertices + lo);
	}
	for (int i = 0; i < n; i++) {
		t51_point(corners, records, chunks[2 * i]);
		glDrawElements(GL_TRIANGLES, chunks[2 * i + 1] * 6, GL_UNSIGNED_SHORT, 0);
	}
	if (read)
		glReadPixels(0, 0, w, h, GL_RGBA, GL_UNSIGNED_BYTE, read);
	return eglSwapBuffers(dpy, surf);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// es2 draws the grid with OpenGL ES 2.0, which reads data per vertex only:
// each cell is four vertices, each with a copy of its record, its place in
// the grid comes from a static buffer of corners, and the grid is drawn in
// chunks of ChunkCells cells (vertices.go).
type es2 struct {
	s                *Screen
	chunks           []C.int // Chunks of the grid, first cell and count after each other
	corners, records C.GLuint
	vertices         []byte // the record buffer's contents
	packed           []byte // a rectangle of the sheet on its way to the GPU
}

// es2Attributes are es2.vert's attributes, in the order of their places.
var es2Attributes = []string{"a_cell", "a_corner", "a_what", "a_fg", "a_bg", "a_line"}

// checkES2 fails when the GPU lacks the precision es2.frag needs: full
// precision for the place in the character sheet.
func checkES2() error {
	var span [2]C.GLint
	var bits C.GLint
	C.glGetShaderPrecisionFormat(C.GL_FRAGMENT_SHADER, C.GL_HIGH_FLOAT, &span[0], &bits)
	if bits < 16 {
		return fmt.Errorf("gpu: this phone's GPU has %d bits of precision in fragment shaders; the character sheet needs 16", int(bits))
	}
	return nil
}

// newES2 sets up the buffers and the character sheet; the program is
// current.
func newES2(s *Screen) *es2 {
	p := &es2{s: s, packed: make([]byte, SheetSize*SheetSize)}
	for i := range es2Attributes {
		C.glEnableVertexAttribArray(C.GLuint(i))
	}

	// Two triangles per cell, for one chunk of cells; it stays bound and
	// serves every chunk.
	indices := Indices()
	var index C.GLuint
	C.glGenBuffers(1, &index)
	C.glBindBuffer(C.GL_ELEMENT_ARRAY_BUFFER, index)
	C.glBufferData(C.GL_ELEMENT_ARRAY_BUFFER, C.GLsizeiptr(len(indices)*2), unsafe.Pointer(&indices[0]), C.GL_STATIC_DRAW)

	// The corners of the grid's cells, and the cells' records, four copies
	// each; grid fills them.
	C.glGenBuffers(1, &p.corners)
	C.glGenBuffers(1, &p.records)

	// The character sheet, empty until sheet is called.
	var tex C.GLuint
	C.glGenTextures(1, &tex)
	C.glActiveTexture(C.GL_TEXTURE0)
	C.glBindTexture(C.GL_TEXTURE_2D, tex)
	C.glTexImage2D(C.GL_TEXTURE_2D, 0, C.GL_LUMINANCE, SheetSize, SheetSize, 0, C.GL_LUMINANCE, C.GL_UNSIGNED_BYTE, nil)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_WRAP_S, C.GL_CLAMP_TO_EDGE)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_WRAP_T, C.GL_CLAMP_TO_EDGE)
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 1)
	return p
}

func (p *es2) grid(cols, rows int) {
	cells := cols * rows
	p.chunks = p.chunks[:0]
	for _, ch := range Chunks(cells) {
		p.chunks = append(p.chunks, C.int(ch[0]), C.int(ch[1]))
	}
	corners := Corners(cols, rows)
	C.glBindBuffer(C.GL_ARRAY_BUFFER, p.corners)
	C.glBufferData(C.GL_ARRAY_BUFFER, C.GLsizeiptr(len(corners)), unsafe.Pointer(&corners[0]), C.GL_STATIC_DRAW)
	p.vertices = make([]byte, cells*4*VertexSize)
	C.glBindBuffer(C.GL_ARRAY_BUFFER, p.records)
	C.glBufferData(C.GL_ARRAY_BUFFER, C.GLsizeiptr(len(p.vertices)), nil, C.GL_DYNAMIC_DRAW)
}

func (p *es2) sheet(sheet []byte, x, y, w, h int) {
	rect := Pack(p.packed, sheet, x, y, w, h)
	C.glTexSubImage2D(C.GL_TEXTURE_2D, 0, C.GLint(x), C.GLint(y), C.GLsizei(w), C.GLsizei(h),
		C.GL_LUMINANCE, C.GL_UNSIGNED_BYTE, unsafe.Pointer(&rect[0]))
}

func (p *es2) frame(table []byte, lo, hi int) error {
	vlo, vhi := Expand(p.vertices, table, lo, hi)
	read, w, h := p.s.readTarget()
	if C.t51_frame2(p.s.dpy, p.s.surf, p.corners, p.records, (*C.uchar)(unsafe.Pointer(&p.vertices[0])),
		C.int(vlo), C.int(vhi), &p.chunks[0], C.int(len(p.chunks)/2), read, w, h) == 0 {
		return p.s.fail("eglSwapBuffers", nil)
	}
	return nil
}
