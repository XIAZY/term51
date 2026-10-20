package gpu

import (
	"regexp"
	"testing"
)

// uniforms are the uniforms a pair of shaders declares, by name, with their
// types.
func uniforms(srcs ...string) map[string]string {
	re := regexp.MustCompile(`(?m)^uniform\s+(?:(?:lowp|mediump|highp|HIGH)\s+)?(\w+)\s+(\w+)\s*;`)
	out := map[string]string{}
	for _, src := range srcs {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			out[m[2]] = m[1]
		}
	}
	return out
}

// Both pairs declare the uniforms program.go sets, with the same types;
// only ES 3.0 has u_grid.
func TestShadersShareUniforms(t *testing.T) {
	es3 := uniforms(es3Vertex, es3Fragment)
	es2 := uniforms(es2Vertex, es2Fragment)
	for _, name := range []string{"u_cell", "u_view", "u_sheet_cols", "u_thick", "u_sheet"} {
		if es3[name] == "" || es3[name] != es2[name] {
			t.Errorf("%s: ES 3.0 declares %q, ES 2.0 %q", name, es3[name], es2[name])
		}
	}
	if es3["u_grid"] == "" || es2["u_grid"] != "" {
		t.Errorf("u_grid: ES 3.0 %q, ES 2.0 %q; it belongs to ES 3.0 only", es3["u_grid"], es2["u_grid"])
	}
	for name := range es2 {
		if es3[name] == "" {
			t.Errorf("ES 2.0 has a uniform %s that ES 3.0 lacks", name)
		}
	}
}

// Each shader names its language version on its first line.
func TestShaderVersions(t *testing.T) {
	for name, want := range map[string]string{"es3Vertex": "#version 300 es", "es3Fragment": "#version 300 es",
		"es2Vertex": "#version 100", "es2Fragment": "#version 100"} {
		src := map[string]string{"es3Vertex": es3Vertex, "es3Fragment": es3Fragment, "es2Vertex": es2Vertex, "es2Fragment": es2Fragment}[name]
		if got := regexp.MustCompile(`^[^\n]*`).FindString(src); got != want {
			t.Errorf("%s starts %q, want %q", name, got, want)
		}
	}
}
