#version 100
// The colours are worked out in medium precision, as es3.frag does, so that
// both draw the same pixels. Only the place in the sheet, v_tex, which runs
// up to the sheet's 2,048 pixels in steps of 1/2,048, needs full precision
// (Open checks that the phone's GPU has it); u_cell is declared as the
// vertex shader has it, which a uniform shared by both must be.
precision mediump float;
#ifdef GL_FRAGMENT_PRECISION_HIGH
#define HIGH highp
#else
#define HIGH mediump
#endif
uniform sampler2D u_sheet;
uniform HIGH vec2 u_cell;
uniform float u_thick;     // the thickness of lines, in pixels
varying vec4 v_fg;
varying vec4 v_bg;
varying vec4 v_line;
varying vec2 v_local;
varying HIGH vec2 v_tex;
varying float v_glyph;
varying vec4 v_flags;
varying float v_box;
void main() {
	float a = 0.0;
	if (v_glyph > 0.5)
		a = texture2D(u_sheet, v_tex).r;
	float x = v_local.x, y = v_local.y, t = u_thick, line = 0.0;
	if (v_flags.y > 0.5 && y >= floor(u_cell.y * 0.55) && y < floor(u_cell.y * 0.55) + t) a = 1.0; // strikethrough
	if (v_flags.x > 0.5 && y >= u_cell.y - 2.0 * t && y < u_cell.y - t) line = 1.0;                // underline
	if (v_flags.z > 0.5 && x < 2.0 * t) line = 1.0;                                                 // cursor bar
	if (v_flags.w > 0.5 && y >= u_cell.y - 2.0 * t) line = 1.0;                                     // cursor underline
	if (v_box > 0.5 && (x < t || y < t || x >= u_cell.x - t || y >= u_cell.y - t)) line = 1.0;       // hollow cursor box
	gl_FragColor = vec4(mix(mix(v_bg.rgb, v_fg.rgb, a), v_line.rgb, line), 1.0);
}
