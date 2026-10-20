#version 100
attribute vec2 a_cell;     // the cell's column and row
attribute vec2 a_corner;   // which corner: 0,0 1,0 0,1 1,1
attribute vec2 a_what;     // slot, flags
attribute vec4 a_fg;
attribute vec4 a_bg;
attribute vec4 a_line;
uniform vec2 u_cell;       // a cell in pixels
uniform vec2 u_view;       // the window in pixels
uniform int u_sheet_cols;  // slots per row of the sheet
varying vec4 v_fg;
varying vec4 v_bg;
varying vec4 v_line;
varying vec2 v_local;      // the pixel inside the cell
varying vec2 v_tex;        // the pixel in the sheet, as a texture coordinate
varying float v_glyph;     // 1 when the cell has a character
varying vec4 v_flags;      // underline, strikethrough, cursor bar, cursor underline
varying float v_box;       // hollow cursor box
// bit is 1 when bit n of the small whole number f is set, else 0.
float bit(float f, float n) {
	return mod(floor(f / exp2(n)), 2.0);
}
void main() {
	vec2 p = (a_cell + a_corner) * u_cell / u_view * 2.0 - 1.0;
	gl_Position = vec4(p.x, -p.y, 0.0, 1.0); // the first row is at the top
	v_fg = a_fg;
	v_bg = a_bg;
	v_line = a_line;
	v_local = a_corner * u_cell;
	float slot = a_what.x;
	// The slot's row and column; the half keeps a division that comes out a
	// whole number from landing a hair below it.
	float cols = float(u_sheet_cols);
	float row = floor((slot + 0.5) / cols);
	vec2 origin = vec2(slot - row * cols, row) * u_cell;
	v_tex = (origin + v_local) / 2048.0;
	v_glyph = slot > 0.5 ? 1.0 : 0.0;
	float flags = a_what.y;
	v_flags = vec4(bit(flags, 0.0), bit(flags, 1.0), bit(flags, 2.0), bit(flags, 3.0));
	v_box = bit(flags, 4.0);
}
