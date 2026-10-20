package gpu

// What the screen hands to OpenGL ES 2.0 (es2.go), worked out in plain Go so
// that it is tested off the phone. ES 2.0 has no instancing: the GPU reads data per
// vertex, never per cell. So each cell is drawn as four vertices of its own,
// each carrying a copy of the cell's record (Expand), and its place in the
// grid comes from a second, static buffer (Corners). Two triangles join a
// cell's corners (Indices). Indices are 16-bit, which reach 65,536 vertices:
// the grid is drawn in chunks of ChunkCells cells, each chunk starting its
// vertices afresh, so the same index buffer serves every chunk and a grid of
// any size.

// SheetSize is the width and height of the character sheet in pixels.
const SheetSize = 2048

// RecordSize is the size of one cell's record in the table.
const RecordSize = 16

// ChunkCells is how many cells one draw call covers: 16-bit indices reach
// ChunkCells*4 vertices.
const ChunkCells = 65536 / 4

// CornerSize is the size of one vertex in the corner buffer: the cell's
// column and row (uint16 each, little-endian), then which corner it is (two
// bytes, 0 or 1, across and down), then two bytes unused.
const CornerSize = 8

// VertexSize is the size of one vertex in the record buffer: a copy of its
// cell's record.
const VertexSize = RecordSize

// The order of a cell's four corners: top left, top right, bottom left,
// bottom right.
var cornerOrder = [4][2]byte{{0, 0}, {1, 0}, {0, 1}, {1, 1}}

// Indices are the index buffer for one chunk: two triangles per cell, the
// cell's top left, top right and bottom left, then its bottom left, top
// right and bottom right.
func Indices() []uint16 {
	out := make([]uint16, 0, ChunkCells*6)
	for c := 0; c < ChunkCells; c++ {
		v := uint16(c * 4)
		out = append(out, v, v+1, v+2, v+2, v+1, v+3)
	}
	return out
}

// Corners is the corner buffer for a grid of cols by rows cells: four
// vertices per cell, cells row by row from the top left as in the table.
func Corners(cols, rows int) []byte {
	out := make([]byte, cols*rows*4*CornerSize)
	i := 0
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			for _, k := range cornerOrder {
				out[i], out[i+1] = byte(col), byte(col>>8)
				out[i+2], out[i+3] = byte(row), byte(row>>8)
				out[i+4], out[i+5] = k[0], k[1]
				i += CornerSize
			}
		}
	}
	return out
}

// Expand copies the cells of table that the bytes lo <= i < hi touch into
// vertices, four copies of each record, at the same cells' place in
// vertices, and returns the range of vertices' bytes written, [vlo, vhi);
// the two are equal when lo == hi.
func Expand(vertices, table []byte, lo, hi int) (vlo, vhi int) {
	if hi <= lo {
		return 0, 0
	}
	first, end := lo/RecordSize, (hi+RecordSize-1)/RecordSize
	for c := first; c < end; c++ {
		rec := table[c*RecordSize : (c+1)*RecordSize]
		for k := 0; k < 4; k++ {
			copy(vertices[(c*4+k)*VertexSize:], rec)
		}
	}
	return first * 4 * VertexSize, end * 4 * VertexSize
}

// Pack copies the rectangle x, y, w, h of the sheet, SheetSize pixels a row,
// into dst, row after row with no gap, and returns that part of dst: ES 2.0
// cannot take a rectangle out of a wider picture.
func Pack(dst, sheet []byte, x, y, w, h int) []byte {
	dst = dst[:w*h]
	for row := 0; row < h; row++ {
		copy(dst[row*w:(row+1)*w], sheet[(y+row)*SheetSize+x:])
	}
	return dst
}

// Chunks are the draw calls for a grid of cells: the first cell and the
// number of cells of each.
func Chunks(cells int) [][2]int {
	var out [][2]int
	for first := 0; first < cells; first += ChunkCells {
		out = append(out, [2]int{first, min(ChunkCells, cells-first)})
	}
	return out
}
