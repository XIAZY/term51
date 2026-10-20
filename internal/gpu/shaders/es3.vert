#version 300 es
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
