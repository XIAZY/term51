//go:build qnx

package gpu

/*
#include <stdlib.h>
#include <GLES2/gl2.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// program is a linked pair of shaders and the places of the uniforms the
// two pipelines share (shaders.go).
type program struct {
	id                                      C.GLuint
	uGrid, uCell, uView, uSheetCols, uThick C.GLint
}

// link compiles the two shaders and links them, with the attributes at the
// places 0, 1, ... in the order given, and makes the program current.
func link(vertex, fragment string, attributes []string) (*program, error) {
	p := &program{id: C.glCreateProgram()}
	for _, sh := range []struct {
		kind C.GLenum
		src  string
	}{{C.GL_VERTEX_SHADER, vertex}, {C.GL_FRAGMENT_SHADER, fragment}} {
		id, err := compile(sh.kind, sh.src)
		if err != nil {
			return nil, err
		}
		C.glAttachShader(p.id, id)
	}
	for i, name := range attributes {
		cname := C.CString(name)
		C.glBindAttribLocation(p.id, C.GLuint(i), (*C.GLchar)(cname))
		C.free(unsafe.Pointer(cname))
	}
	C.glLinkProgram(p.id)
	var ok C.GLint
	C.glGetProgramiv(p.id, C.GL_LINK_STATUS, &ok)
	if ok == 0 {
		var log [512]C.GLchar
		C.glGetProgramInfoLog(p.id, C.GLsizei(len(log)), nil, &log[0])
		return nil, fmt.Errorf("gpu: linking the program: %s", C.GoString((*C.char)(&log[0])))
	}
	C.glUseProgram(p.id)
	p.uGrid, p.uCell, p.uView = p.uniform("u_grid"), p.uniform("u_cell"), p.uniform("u_view")
	p.uSheetCols, p.uThick = p.uniform("u_sheet_cols"), p.uniform("u_thick")
	C.glUniform1i(p.uniform("u_sheet"), 0)
	return p, nil
}

// uniform is the place of the named uniform; -1, which OpenGL ignores when
// it is set, where the program has none.
func (p *program) uniform(name string) C.GLint {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return C.glGetUniformLocation(p.id, (*C.GLchar)(cname))
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

// view tells the program the window's size in pixels.
func (p *program) view(w, h int) {
	C.glUniform2f(p.uView, C.GLfloat(w), C.GLfloat(h))
}

// grid tells the program the grid: cols by rows cells of cellW by cellH
// pixels.
func (p *program) grid(cols, rows, cellW, cellH int) {
	C.glUniform2i(p.uGrid, C.GLint(cols), C.GLint(rows))
	C.glUniform2f(p.uCell, C.GLfloat(cellW), C.GLfloat(cellH))
	C.glUniform1i(p.uSheetCols, C.GLint(SheetSize/cellW))
	C.glUniform1f(p.uThick, C.GLfloat(max(cellH/14, 1)))
}
