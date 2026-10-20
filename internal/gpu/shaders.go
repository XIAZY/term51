package gpu

import _ "embed"

// The two pairs of shaders, one for each pipeline. They draw the same
// picture and share the names and types of their uniforms, so that setting
// them (program.go) is one piece of code; ES 3.0's u_grid, which ES 2.0 does
// not need, is set on both and ignored where a program does not have it.

//go:embed shaders/es3.vert
var es3Vertex string

//go:embed shaders/es3.frag
var es3Fragment string

//go:embed shaders/es2.vert
var es2Vertex string

//go:embed shaders/es2.frag
var es2Fragment string
