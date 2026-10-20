#version 300 es
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
