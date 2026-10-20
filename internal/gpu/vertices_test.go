package gpu

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Each chunk's index buffer joins each cell's four corners with two
// triangles, and its largest index is the last vertex 16-bit indices reach.
func TestIndices(t *testing.T) {
	ix := Indices()
	if len(ix) != ChunkCells*6 {
		t.Fatalf("%d indices, want %d", len(ix), ChunkCells*6)
	}
	if got := ix[:6]; !equal(got, []uint16{0, 1, 2, 2, 1, 3}) {
		t.Errorf("the first cell's triangles are %v", got)
	}
	if got := ix[6:12]; !equal(got, []uint16{4, 5, 6, 6, 5, 7}) {
		t.Errorf("the second cell's triangles are %v", got)
	}
	var top uint16
	for _, i := range ix {
		top = max(top, i)
	}
	if top != 65535 {
		t.Errorf("the largest index is %d, want 65535", top)
	}
}

func equal(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The corner buffer holds each cell's column and row, four times, with its
// four corners in the index buffer's order; the columns and rows go past
// 255.
func TestCorners(t *testing.T) {
	cols, rows := 300, 3
	c := Corners(cols, rows)
	if len(c) != cols*rows*4*CornerSize {
		t.Fatalf("%d bytes, want %d", len(c), cols*rows*4*CornerSize)
	}
	vertex := func(cell, k int) (col, row int, cx, cy byte) {
		v := c[(cell*4+k)*CornerSize:]
		return int(binary.LittleEndian.Uint16(v)), int(binary.LittleEndian.Uint16(v[2:])), v[4], v[5]
	}
	for _, cell := range []int{0, 1, 299, 300, 899} {
		for k, want := range cornerOrder {
			col, row, cx, cy := vertex(cell, k)
			if col != cell%cols || row != cell/cols || cx != want[0] || cy != want[1] {
				t.Errorf("cell %d corner %d is col %d row %d corner %d,%d", cell, k, col, row, cx, cy)
			}
		}
	}
}

// Expand copies each changed cell's record onto its four vertices, rounds a
// change inside a record out to the whole record, and leaves other cells
// alone.
func TestExpand(t *testing.T) {
	cells := 5
	table := make([]byte, cells*RecordSize)
	for i := range table {
		table[i] = byte(i)
	}
	vertices := bytes.Repeat([]byte{0xee}, cells*4*VertexSize)
	// A change inside cell 1's record and the start of cell 2's.
	vlo, vhi := Expand(vertices, table, RecordSize+3, 2*RecordSize+1)
	if vlo != 1*4*VertexSize || vhi != 3*4*VertexSize {
		t.Errorf("the range written is [%d, %d), want [%d, %d)", vlo, vhi, 4*VertexSize, 12*VertexSize)
	}
	for cell := 0; cell < cells; cell++ {
		for k := 0; k < 4; k++ {
			got := vertices[(cell*4+k)*VertexSize:][:VertexSize]
			want := table[cell*RecordSize:][:RecordSize]
			if cell != 1 && cell != 2 {
				want = bytes.Repeat([]byte{0xee}, VertexSize)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("cell %d vertex %d is %x, want %x", cell, k, got, want)
			}
		}
	}
	if lo, hi := Expand(vertices, table, 7, 7); lo != hi {
		t.Errorf("no change wrote [%d, %d)", lo, hi)
	}
}

// A grid larger than one chunk is drawn in several, covering every cell
// once; one at most a chunk is one draw.
func TestChunks(t *testing.T) {
	// 1440 by 1440 pixels in cells of 6 by 12: 240 by 120 cells.
	for _, cells := range []int{1, ChunkCells, ChunkCells + 1, 240 * 120} {
		next := 0
		for _, ch := range Chunks(cells) {
			if ch[0] != next || ch[1] < 1 || ch[1] > ChunkCells {
				t.Errorf("%d cells: chunk %v after cell %d", cells, ch, next)
			}
			next = ch[0] + ch[1]
		}
		if next != cells {
			t.Errorf("%d cells: the chunks cover %d", cells, next)
		}
	}
	if n := len(Chunks(240 * 120)); n != 2 {
		t.Errorf("a grid of 240 by 120 is %d draws, want 2", n)
	}
}

// Pack lifts a rectangle out of the sheet, row after row with no gap.
func TestPack(t *testing.T) {
	sheet := make([]byte, SheetSize*SheetSize)
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			sheet[(100+y)*SheetSize+200+x] = byte(10*y + x + 1)
		}
	}
	got := Pack(make([]byte, 64), sheet, 201, 101, 4, 2)
	want := []byte{12, 13, 14, 15, 22, 23, 24, 25}
	if !bytes.Equal(got, want) {
		t.Errorf("Pack gave %v, want %v", got, want)
	}
}
