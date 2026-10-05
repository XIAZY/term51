package terminal

// Self tests of the OSC and colour payloads of dumpAny (corpus_replay_dump_test.go):
// the Go values that the OSC wireframes (osc.go, osc_parsers_*.go, color.go,
// kitty_color.go) declare are dumped in the format of harness/src/dump.zig
// dumpAny, and the text is compared with the recorded actions and parser
// traces. None of them needs the port: they run today.
//
// What the port is needed for is the name of an enum value (the String
// methods of the wireframes panic with notImplemented, five tag types have no
// String at all). The replayer takes the names from the port; these tests set
// dumpEnumNames, the table of the names of every enum type of the OSC
// payloads, which they read from the Zig source (oscSelfEnumSpecs) and check
// against the constants of the Go types (TestCorpusReplaySelfOSCEnumNames).
//
// Coverage (see the report of TestCorpusReplaySelfOSCRoundTrip, run with -v):
// every recorded block of an OSC-derived action and every recorded
// `@osc_dispatch` of the parser traces is read back into a Go value by a
// reader that is the inverse of dumpValue (oscSelfReader) and dumped again:
// the text must be the recorded one. The values that the corpus never records
// (the Command variants other than change_window_title, reset_special, ...)
// are built by hand in TestCorpusReplaySelfOSCVariants, with the text that
// the Zig rules of dumpAny give, and the test checks that the two sets
// together cover every variant of osc.Command and of the colour requests.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// The names of the enum values, from the Zig source.

// oscSelfEnumSpec says where the Zig source declares an enum type of the OSC
// payloads: the names, in order, are the Go constants' names.
type oscSelfEnumSpec struct {
	goType reflect.Type
	file   string // below third_party/ghostty/src/terminal
	// anchor is a text before the declaration (a nested enum is found after
	// the declaration of its container); name is the declared name. For a tag
	// type (a tagged union's tag) name is empty: the tags are the fields of
	// the union of that file that has exactly the tags of the Go type.
	anchor string
	name   string
}

var oscSelfEnumSpecs = []oscSelfEnumSpec{
	{reflect.TypeOf(OSCParsersColorOperation(0)), "osc/parsers/color.zig", "", "Operation"},
	{reflect.TypeOf(ColorSpecial(0)), "color.zig", "", "Special"},
	{reflect.TypeOf(ColorDynamic(0)), "color.zig", "", "Dynamic"},
	{reflect.TypeOf(OSCTerminator(0)), "osc.zig", "", "Terminator"},
	{reflect.TypeOf(OSCCommandProgressReportState(0)), "osc.zig", "pub const ProgressReport = struct", "State"},
	{reflect.TypeOf(MouseShape(0)), "mouse_shape.zig", "", "MouseShape"},
	{reflect.TypeOf(OSCParsersSemanticPromptCommandAction(0)), "osc/parsers/semantic_prompt.zig", "pub const Command = struct", "Action"},
	{reflect.TypeOf(KittyColorSpecial(0)), "kitty/color.zig", "", "Special"},
	{reflect.TypeOf(OSCParsersKittyTextSizingVAlign(0)), "osc/parsers/kitty_text_sizing.zig", "", "VAlign"},
	{reflect.TypeOf(OSCParsersKittyTextSizingHAlign(0)), "osc/parsers/kitty_text_sizing.zig", "", "HAlign"},
	{reflect.TypeOf(OSCParsersContextSignalCommandAction(0)), "osc/parsers/context_signal.zig", "pub const Command = struct", "Action"},
	// The tags of the unions.
	{reflect.TypeOf(OSCCommandKey(0)), "osc.zig", "", ""},
	{reflect.TypeOf(OSCCommandConemuChangeTabTitlePayloadTag(0)), "osc.zig", "", ""},
	{reflect.TypeOf(OSCParsersColorRequestTag(0)), "osc/parsers/color.zig", "", ""},
	{reflect.TypeOf(OSCParsersColorTargetTag(0)), "osc/parsers/color.zig", "", ""},
	{reflect.TypeOf(KittyColorOSCRequestTag(0)), "kitty/color.zig", "", ""},
	{reflect.TypeOf(KittyColorKindTag(0)), "kitty/color.zig", "", ""},
}

// oscSelfZigEnumField is a name of an enum of the Zig source with its value.
type oscSelfZigEnumField struct {
	name  string
	value int64
}

var (
	oscSelfEnumStart = regexp.MustCompile(`^(?:pub\s+)?const\s+(\w+)\s*=\s*(enum(?:\s*\(\s*\w+\s*\))?\s*\{|(?:lib\.Enum|LibEnum)\()`)
	oscSelfEnumField = regexp.MustCompile(`^(@"[^"]+"|[A-Za-z_]\w*)\s*(?:=\s*(\d+))?\s*,?\s*$`)
	oscSelfQuoted    = regexp.MustCompile(`"([A-Za-z_0-9]+)"`)
)

// oscSelfScanEnum finds `const <name> = enum {...}` (or the lib.Enum list of
// names) after the first `anchor` of the Zig text, and returns its names with
// their values (an explicit `= N`, else the one after the previous).
func oscSelfScanEnum(text, anchor, name string) ([]oscSelfZigEnumField, bool) {
	lines := strings.Split(text, "\n")
	i := 0
	if anchor != "" {
		for ; i < len(lines) && !strings.Contains(lines[i], anchor); i++ {
		}
	}
	for ; i < len(lines); i++ {
		m := oscSelfEnumStart.FindStringSubmatch(strings.TrimSpace(zigStripLine(lines[i])))
		if m == nil || m[1] != name {
			continue
		}
		var out []oscSelfZigEnumField
		if strings.HasPrefix(m[2], "enum") {
			depth := 1
			next := int64(0)
			for j := i + 1; j < len(lines) && depth > 0; j++ {
				code := strings.TrimSpace(zigStripLine(lines[j]))
				if depth == 1 {
					if f := oscSelfEnumField.FindStringSubmatch(code); f != nil {
						if f[2] != "" {
							next, _ = strconv.ParseInt(f[2], 10, 64)
						}
						out = append(out, oscSelfZigEnumField{strings.TrimSuffix(strings.TrimPrefix(f[1], `@"`), `"`), next})
						next++
					}
				}
				depth += strings.Count(code, "{") + strings.Count(code, "(") - strings.Count(code, "}") - strings.Count(code, ")")
			}
			return out, true
		}
		// lib.Enum(target, &.{ "a", "b" }): the quoted names up to `})`.
		var body strings.Builder
		for j := i; j < len(lines); j++ {
			body.WriteString(lines[j] + "\n")
			if strings.Contains(lines[j], "})") {
				break
			}
		}
		for k, q := range oscSelfQuoted.FindAllStringSubmatch(body.String(), -1) {
			out = append(out, oscSelfZigEnumField{q[1], int64(k)})
		}
		return out, true
	}
	return nil, false
}

// oscSelfZigText is a file of the Zig source of the terminal package.
func oscSelfZigText(t testing.TB, file string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(corpusDir(t), "third_party", "ghostty", "src", "terminal", file))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// oscSelfGoConsts are the constants of the Go types of the package that are
// declared as `T<Value>` (or without the `Tag` of T) with their values: the
// constant blocks of the wireframes use literals, iota and its repetition.
func oscSelfGoConsts(t *testing.T) map[string]map[string]int64 {
	t.Helper()
	out := map[string]map[string]int64{}
	var eval func(e ast.Expr, iota int64) (int64, bool)
	eval = func(e ast.Expr, iota int64) (int64, bool) {
		switch e := e.(type) {
		case *ast.Ident:
			return iota, e.Name == "iota"
		case *ast.BasicLit:
			n, err := strconv.ParseInt(e.Value, 0, 64)
			return n, err == nil && e.Kind == token.INT
		case *ast.ParenExpr:
			return eval(e.X, iota)
		case *ast.BinaryExpr:
			x, ok1 := eval(e.X, iota)
			y, ok2 := eval(e.Y, iota)
			switch e.Op {
			case token.ADD:
				return x + y, ok1 && ok2
			case token.SUB:
				return x - y, ok1 && ok2
			}
		}
		return 0, false
	}
	for _, f := range parsePackageSources(t) {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			typ := ""
			var expr ast.Expr
			for i, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				if len(vs.Values) > 0 {
					expr = vs.Values[0]
					typ = ""
					if id, ok := vs.Type.(*ast.Ident); ok {
						typ = id.Name
					}
				}
				if typ == "" || expr == nil {
					continue
				}
				value, ok := eval(expr, int64(i))
				if !ok {
					continue
				}
				for _, n := range vs.Names {
					for _, prefix := range []string{typ, strings.TrimSuffix(typ, "Tag")} {
						rest, ok := strings.CutPrefix(n.Name, prefix)
						if ok && rest != "" && (rest[0] >= 'A' && rest[0] <= 'Z' || rest[0] >= '0' && rest[0] <= '9') {
							if out[typ] == nil {
								out[typ] = map[string]int64{}
							}
							out[typ][rest] = value
							break
						}
					}
				}
			}
		}
	}
	return out
}

// oscSelfEnumNames reads the names of every enum type of oscSelfEnumSpecs from
// the Zig source and checks the constants of the Go type: the same names
// (zigCamel) with the same values, no other constant. It returns the table for
// dumpEnumNames.
func oscSelfEnumNames(t *testing.T) map[reflect.Type]map[int64]string {
	t.Helper()
	consts := oscSelfGoConsts(t)
	out := map[reflect.Type]map[int64]string{}
	unions := map[string][]zigUnion{}
	for _, spec := range oscSelfEnumSpecs {
		text := oscSelfZigText(t, spec.file)
		var fields []oscSelfZigEnumField
		goConsts := consts[spec.goType.Name()]
		if spec.name != "" {
			var ok bool
			if fields, ok = oscSelfScanEnum(text, spec.anchor, spec.name); !ok {
				t.Errorf("%s: no enum %s in %s", spec.goType, spec.name, spec.file)
				continue
			}
		} else {
			// A tag type: the union of the file with the same tags.
			if unions[spec.file] == nil {
				unions[spec.file] = zigScanUnions(spec.file, text)
			}
			var goNames []string
			for n := range goConsts {
				goNames = append(goNames, n)
			}
			var match []zigUnion
			for _, zu := range unions[spec.file] {
				var names []string
				for _, f := range zu.fields {
					names = append(names, zigCamel(f.name))
				}
				if slices.Equal(sortedCopy(names), sortedCopy(goNames)) {
					match = append(match, zu)
				}
			}
			if len(match) == 0 {
				t.Errorf("%s: no union of %s has the tags %v", spec.goType, spec.file, sortedCopy(goNames))
				continue
			}
			for _, zu := range match[1:] {
				for i := range zu.fields {
					if zu.fields[i].name != match[0].fields[i].name {
						t.Errorf("%s: the unions %s and %s have the same tags in another order", spec.goType, match[0].where, zu.where)
					}
				}
			}
			for i, f := range match[0].fields {
				fields = append(fields, oscSelfZigEnumField{f.name, int64(i)})
			}
		}
		names := map[int64]string{}
		for _, f := range fields {
			names[f.value] = f.name
			if v, ok := goConsts[zigCamel(f.name)]; !ok || v != f.value {
				t.Errorf("%s: Zig %s = %d, the Go constant %s is %d (declared: %v)", spec.goType, f.name, f.value, zigCamel(f.name), v, ok)
			}
		}
		if len(goConsts) != len(fields) {
			t.Errorf("%s: %d Go constants, %d names in the Zig source", spec.goType, len(goConsts), len(fields))
		}
		out[spec.goType] = names
	}
	return out
}

// oscSelfUseNames installs the names of oscSelfEnumNames as dumpEnumNames for
// the rest of the test.
func oscSelfUseNames(t *testing.T) {
	t.Helper()
	dumpEnumNames = oscSelfEnumNames(t)
	t.Cleanup(func() { dumpEnumNames = nil })
}

func TestCorpusReplaySelfOSCEnumNames(t *testing.T) {
	names := oscSelfEnumNames(t)
	if len(names) != len(oscSelfEnumSpecs) {
		t.Errorf("%d tables for %d specs", len(names), len(oscSelfEnumSpecs))
	}
	// Spot checks against the Zig source (the names are the recorded ones).
	for _, c := range []struct {
		typ   reflect.Type
		value int64
		name  string
	}{
		{reflect.TypeOf(OSCParsersColorOperation(0)), 0, "osc_4"},
		{reflect.TypeOf(OSCParsersColorOperation(0)), 23, "osc_119"},
		{reflect.TypeOf(ColorDynamic(0)), 10, "foreground"},
		{reflect.TypeOf(ColorDynamic(0)), 19, "highlight_foreground"},
		{reflect.TypeOf(ColorSpecial(0)), 4, "italic"},
		{reflect.TypeOf(OSCTerminator(0)), 1, "bel"},
		{reflect.TypeOf(OSCCommandProgressReportState(0)), 2, "error"},
		{reflect.TypeOf(OSCCommandKey(0)), 0, "invalid"},
		{reflect.TypeOf(OSCCommandKey(0)), 24, "context_signal"},
		{reflect.TypeOf(OSCParsersColorRequestTag(0)), 3, "reset_palette"},
		{reflect.TypeOf(KittyColorOSCRequestTag(0)), 1, "set"},
		{reflect.TypeOf(MouseShape(0)), 3, "pointer"},
		{reflect.TypeOf(OSCParsersKittyTextSizingHAlign(0)), 2, "center"},
	} {
		if got := names[c.typ][c.value]; got != c.name {
			t.Errorf("%s value %d is %q, want %q", c.typ, c.value, got, c.name)
		}
	}
}

// The enum types that dumpAny reaches in the OSC payloads and the stream
// actions that the wireframes leave without a name method: the report of the
// gaps for the lead (a Go type that dumpAny cannot name panics, see
// dumpUnnamedEnum and dumpUnion). The test logs them, it does not fail.
func TestCorpusReplaySelfOSCWireframeGaps(t *testing.T) {
	var gaps []string
	seen := map[reflect.Type]bool{}
	var walk func(rt reflect.Type, where string)
	walk = func(rt reflect.Type, where string) {
		if seen[rt] {
			return
		}
		seen[rt] = true
		switch {
		case rt.Kind() == reflect.Struct && dumpIsOptional(rt):
			walk(rt.Field(0).Type, where)
		case rt.Kind() == reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				if strings.HasPrefix(f.Name, "_") {
					continue
				}
				if i == 0 && dumpIsUnion(rt) && !dumpHasName(f.Type) {
					gaps = append(gaps, fmt.Sprintf("%s (tag of %s): no String", f.Type, rt))
				}
				walk(f.Type, where)
			}
		case rt.Kind() == reflect.Pointer || rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array:
			walk(rt.Elem(), where)
		case dumpIsInteger(rt) && rt.PkgPath() != "" && !dumpHasName(rt):
			gaps = append(gaps, fmt.Sprintf("%s (in %s): no String", rt, where))
		}
	}
	for _, root := range []any{OSCCommand{}, StreamAction{}, ParserAction{}} {
		walk(reflect.TypeOf(root), reflect.TypeOf(root).String())
	}
	sort.Strings(gaps)
	for _, g := range gaps {
		t.Log("wireframe gap: " + g)
	}
}

// ---------------------------------------------------------------------------
// The order of the fields: the dump writes the fields of a struct in the order
// of the Go struct, which must be the order of the Zig struct.

var (
	oscSelfStructField = regexp.MustCompile(`^(@"[^"]+"|[A-Za-z_]\w*)\s*:`)
)

// oscSelfScanStruct returns the field names, in order, of the struct declared
// by the first line of the Zig text that matches start (`name: struct {`,
// `const Name = struct {`): the fields at its top level, or the one-line form
// `struct { a: A, b: B }`.
func oscSelfScanStruct(text string, start *regexp.Regexp) ([]string, bool) {
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		code := strings.TrimSpace(zigStripLine(lines[i]))
		if !start.MatchString(code) {
			continue
		}
		var out []string
		if strings.HasSuffix(code, "},") || strings.HasSuffix(code, "}") {
			body := code[strings.Index(code, "{")+1 : strings.LastIndex(code, "}")]
			for _, part := range strings.Split(body, ",") {
				if m := oscSelfStructField.FindStringSubmatch(strings.TrimSpace(part)); m != nil {
					out = append(out, m[1])
				}
			}
			return out, true
		}
		depth := 1
		for j := i + 1; j < len(lines) && depth > 0; j++ {
			code := strings.TrimSpace(zigStripLine(lines[j]))
			if depth == 1 {
				if m := oscSelfStructField.FindStringSubmatch(code); m != nil && !zigNotAField[m[1]] {
					out = append(out, strings.TrimSuffix(strings.TrimPrefix(m[1], `@"`), `"`))
				}
			}
			depth += strings.Count(code, "{") + strings.Count(code, "(") + strings.Count(code, "[") -
				strings.Count(code, "}") - strings.Count(code, ")") - strings.Count(code, "]")
		}
		return out, true
	}
	return nil, false
}

func TestCorpusReplaySelfOSCFieldOrder(t *testing.T) {
	re := regexp.MustCompile
	for _, c := range []struct {
		v     any
		file  string
		start string
	}{
		// osc.Command: the payload structs, in the union.
		{OSCCommandClipboardContentsPayload{}, "osc.zig", `^clipboard_contents: struct \{`},
		{OSCCommandReportPwdPayload{}, "osc.zig", `^report_pwd: struct \{`},
		{OSCCommandMouseShapePayload{}, "osc.zig", `^mouse_shape: struct \{`},
		{OSCCommandColorOperationPayload{}, "osc.zig", `^color_operation: struct \{`},
		{OSCCommandShowDesktopNotificationPayload{}, "osc.zig", `^show_desktop_notification: struct \{`},
		{OSCCommandHyperlinkStartPayload{}, "osc.zig", `^hyperlink_start: struct \{`},
		{OSCCommandConemuSleepPayload{}, "osc.zig", `^conemu_sleep: struct \{`},
		{OSCCommandConemuXtermEmulationPayload{}, "osc.zig", `^conemu_xterm_emulation: struct \{`},
		{OSCCommandProgressReport{}, "osc.zig", `^pub const ProgressReport = struct \{`},
		{OSCCommandSemanticPrompt{}, "osc/parsers/semantic_prompt.zig", `^pub const Command = struct \{`},
		{OSCParsersContextSignalCommand{}, "osc/parsers/context_signal.zig", `^pub const Command = struct \{`},
		{OSCParsersKittyTextSizingOSC{}, "osc/parsers/kitty_text_sizing.zig", `^pub const OSC = struct \{`},
		{OSCParsersKittyClipboardProtocolOSC{}, "osc/parsers/kitty_clipboard_protocol.zig", `^pub const OSC = struct \{`},
		// The colour requests.
		{KittyColorOSC{}, "kitty/color.zig", `^pub const OSC = struct \{`},
		{KittyColorOSCRequestSetPayload{}, "kitty/color.zig", `^set: struct \{`},
		{OSCParsersColorColoredTarget{}, "osc/parsers/color.zig", `^pub const ColoredTarget = struct \{`},
		{ColorRGB{}, "color.zig", `^pub const RGB = packed struct`},
		// The payloads of the stream actions.
		{StreamActionWindowTitle{}, "stream.zig", `^pub const WindowTitle = struct \{`},
		{StreamActionReportPwd{}, "stream.zig", `^pub const ReportPwd = struct \{`},
		{StreamActionShowDesktopNotification{}, "stream.zig", `^pub const ShowDesktopNotification = struct \{`},
		{StreamActionStartHyperlink{}, "stream.zig", `^pub const StartHyperlink = struct \{`},
		{StreamActionClipboardContents{}, "stream.zig", `^pub const ClipboardContents = struct \{`},
		{StreamActionColorOperation{}, "stream.zig", `^pub const ColorOperation = struct \{`},
	} {
		rt := reflect.TypeOf(c.v)
		want, ok := oscSelfScanStruct(oscSelfZigText(t, c.file), re(c.start))
		if !ok || len(want) == 0 {
			t.Errorf("%s: no struct %q in %s", rt, c.start, c.file)
			continue
		}
		var got []string
		for i := 0; i < rt.NumField(); i++ {
			if !strings.HasPrefix(rt.Field(i).Name, "_") {
				got = append(got, zigName(rt.Field(i).Name))
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: fields %v, the Zig struct has %v", rt, got, want)
		}
	}
	// A scan of an inline struct and of the one-line form.
	if got, _ := oscSelfScanStruct("x: struct {\n    a: u8,\n    /// c\n    b: ?bool,\n},\n", re(`^x: struct`)); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("scan: %v", got)
	}
	if got, _ := oscSelfScanStruct("    set: struct { key: Kind, color: RGB },", re(`^set: struct`)); !slices.Equal(got, []string{"key", "color"}) {
		t.Errorf("one-line scan: %v", got)
	}
}

// The scanners on text, so that a failure of the tests above is the Zig
// source's and not the scanner's.
func TestCorpusReplaySelfOSCScanEnum(t *testing.T) {
	text := "pub const Outer = struct {\n" +
		"    pub const State = enum(c_int) {\n" +
		"        remove,\n" +
		"        set,\n" +
		"        @\"error\",\n" +
		"        pause = 7,\n" +
		"        next,\n" +
		"        pub fn f() void {}\n" +
		"        test \"x\" {\n" +
		"            a,\n" +
		"        }\n" +
		"    };\n" +
		"};\n" +
		"pub const L = lib.Enum(lib_target, &.{\n" +
		"    \"top\",\n" +
		"    \"bottom\",\n" +
		"});\n"
	got, ok := oscSelfScanEnum(text, "pub const Outer", "State")
	want := []oscSelfZigEnumField{{"remove", 0}, {"set", 1}, {"error", 2}, {"pause", 7}, {"next", 8}}
	if !ok || !slices.Equal(got, want) {
		t.Errorf("enum %v %v, want %v", got, ok, want)
	}
	got, ok = oscSelfScanEnum(text, "", "L")
	if want := []oscSelfZigEnumField{{"top", 0}, {"bottom", 1}}; !ok || !slices.Equal(got, want) {
		t.Errorf("lib enum %v %v, want %v", got, ok, want)
	}
	if _, ok := oscSelfScanEnum(text, "", "Nope"); ok {
		t.Error("a missing enum was found")
	}
}

// ---------------------------------------------------------------------------
// The reader: the inverse of dumpValue, for the recorded blocks.

// oscSelfReader reads the lines of one recorded block back into a Go value.
// It follows the shape of the Go type exactly as dumpValue writes it, so a
// recorded line that the Go type cannot hold is an error naming the line.
type oscSelfReader struct {
	lines []string
	pos   int
}

func (r *oscSelfReader) peek() string {
	if r.pos < len(r.lines) {
		return r.lines[r.pos]
	}
	return ""
}

// take consumes the next line, which must start with prefix, and returns the rest.
func (r *oscSelfReader) take(prefix string) (string, error) {
	if r.pos >= len(r.lines) {
		return "", fmt.Errorf("end of block, expected %q", prefix)
	}
	rest, ok := strings.CutPrefix(r.lines[r.pos], prefix)
	if !ok {
		return "", fmt.Errorf("line %q, expected %q", r.lines[r.pos], prefix)
	}
	r.pos++
	return rest, nil
}

// oscSelfEnumValue is the value of the enum type t named name: `#N` or a name
// of the table (dumpEnumNames) or of String().
func oscSelfEnumValue(t reflect.Type, name string) (int64, error) {
	if n, ok := strings.CutPrefix(name, "#"); ok {
		return strconv.ParseInt(n, 10, 64)
	}
	for v := int64(0); v < 256; v++ {
		rv := reflect.New(t).Elem()
		if rv.CanInt() {
			rv.SetInt(v)
		} else {
			rv.SetUint(uint64(v))
		}
		if got, ok := dumpTagName(rv); ok && got == name {
			return v, nil
		}
	}
	return 0, fmt.Errorf("%s has no value named %q", t, name)
}

func oscSelfSetInt(v reflect.Value, n int64) error {
	if v.CanInt() {
		if v.OverflowInt(n) {
			return fmt.Errorf("%d overflows %s", n, v.Type())
		}
		v.SetInt(n)
		return nil
	}
	if n < 0 || v.OverflowUint(uint64(n)) {
		return fmt.Errorf("%d overflows %s", n, v.Type())
	}
	v.SetUint(uint64(n))
	return nil
}

// value reads the value of v (addressable) that dumpValue wrote at prefix.
func (r *oscSelfReader) value(prefix string, v reflect.Value) error {
	t := v.Type()
	switch {
	case t.Kind() == reflect.Bool:
		rest, err := r.take(prefix + "=")
		if err != nil {
			return err
		}
		v.SetBool(rest == "1")
		return nil
	case dumpIsEnum(t):
		rest, err := r.take(prefix + "=")
		if err != nil {
			return err
		}
		n, err := oscSelfEnumValue(t, rest)
		if err != nil {
			return err
		}
		return oscSelfSetInt(v, n)
	case dumpIsInteger(t):
		rest, err := r.take(prefix + "=")
		if err != nil {
			return err
		}
		n, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			return err
		}
		return oscSelfSetInt(v, n)
	case t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 && !dumpIsEnum(t.Elem()):
		rest, err := r.take(prefix + "=")
		if err != nil {
			return err
		}
		b, err := oscSelfUnescape(rest)
		if err != nil {
			return err
		}
		v.SetBytes(b)
		return nil
	case t.Kind() == reflect.Slice:
		return r.items(prefix, v)
	case t.Kind() != reflect.Struct:
		return fmt.Errorf("%s at %s: not readable", t, prefix)
	case dumpIsOptional(t):
		// A null Optional and an Optional bytes whose value is `null` are the
		// same line; the reader takes null.
		if r.peek() == prefix+"=null" {
			r.pos++
			return nil
		}
		dumpField(v, 1).SetBool(true)
		return r.value(prefix, dumpField(v, 0))
	case dumpIsUnion(t):
		rest, err := r.take(prefix + "=")
		if err != nil {
			return err
		}
		tag := dumpField(v, 0)
		n, err := oscSelfEnumValue(tag.Type(), rest)
		if err != nil {
			return err
		}
		if err := oscSelfSetInt(tag, n); err != nil {
			return err
		}
		if slices.Contains(dumpVoidTags[t], rest) {
			return nil
		}
		f, ok := t.FieldByName(zigFieldName(rest))
		if !ok {
			return fmt.Errorf("%s has no field for the tag %q", t, rest)
		}
		return r.value(prefix+"."+rest, dumpField(v, f.Index[0]))
	case dumpIsSegmentedList(t):
		return r.items(prefix, dumpField(v, 0))
	}
	for i := 0; i < t.NumField(); i++ {
		if strings.HasPrefix(t.Field(i).Name, "_") {
			continue
		}
		if err := r.value(prefix+"."+zigName(t.Field(i).Name), dumpField(v, i)); err != nil {
			return err
		}
	}
	return nil
}

// items reads `prefix.len=N` and the items `prefix[i]:` `prefix[]...`.
func (r *oscSelfReader) items(prefix string, v reflect.Value) error {
	rest, err := r.take(prefix + ".len=")
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(rest)
	if err != nil {
		return err
	}
	v.Set(reflect.MakeSlice(v.Type(), n, n))
	for i := 0; i < n; i++ {
		if _, err := r.take(fmt.Sprintf("%s[%d]:", prefix, i)); err != nil {
			return err
		}
		if err := r.value(prefix+"[]", v.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

// oscSelfUnescape is the inverse of writeEscaped.
func oscSelfUnescape(s string) ([]byte, error) {
	out := []byte{}
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			out = append(out, s[i])
			continue
		}
		if i+4 > len(s) || s[i+1] != 'x' {
			return nil, fmt.Errorf("bad escape in %q", s)
		}
		n, err := strconv.ParseUint(s[i+2:i+4], 16, 8)
		if err != nil {
			return nil, err
		}
		out = append(out, byte(n))
		i += 3
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// The recorded blocks.

// oscSelfActionTags are the actions of stream.zig oscDispatch that call
// handler.vt, and osc_dispatch is the parser action of the Command.
var oscSelfActionTags = []string{
	"semantic_prompt", "window_title", "clipboard_contents", "report_pwd",
	"mouse_shape", "color_operation", "kitty_color_report",
	"show_desktop_notification", "start_hyperlink", "end_hyperlink",
	"progress_report",
}

// oscSelfFiles are the recorded files (below golden/) that are read whole: the
// OSC suites, the other suites with an OSC action in their first sections and
// the parser traces that have an @osc_dispatch.
func oscSelfFiles(t *testing.T) []string {
	t.Helper()
	files := []string{"esc/reset.actions.gz", "esc/save_restore.actions.gz", "csi/misc.actions.gz", "modes/alt_screen.actions.gz"}
	for _, suite := range []string{"all", "color_state", "hyperlink_text", "lengths", "semantic_prompt_text", "unterminated"} {
		files = append(files, "osc/"+suite+".actions.gz", "osc/"+suite+".actions_noalloc.gz")
	}
	for _, suite := range []string{"abort", "c0_embedded", "nextslicecapped_inputs", "state_osc_string", "state_then_final", "utf8"} {
		files = append(files, "parser/"+suite+".parser.gz")
	}
	if !testing.Short() {
		for _, suite := range []string{"chunked", "mutated", "structured_short", "structured_medium", "structured_long", "with_resize"} {
			files = append(files, "fuzz/"+suite+".actions.gz")
		}
	}
	return files
}

// oscSelfBlock is a recorded block of an OSC action: `@tag` and its ` v` lines.
type oscSelfBlock struct {
	tag   string
	lines []string // the payload lines
	count int
}

// oscSelfBlocks are the distinct blocks of the OSC actions (and of the parser
// action osc_dispatch) of the files, by their text.
func oscSelfBlocks(t *testing.T, files []string) map[string]*oscSelfBlock {
	t.Helper()
	out := map[string]*oscSelfBlock{}
	for _, f := range files {
		lines := strings.Split(string(corpusReadAll(t, "golden/"+f)), "\n")
		for i := 0; i < len(lines); i++ {
			tag, ok := strings.CutPrefix(lines[i], "@")
			if !ok || !(slices.Contains(oscSelfActionTags, tag) || tag == "osc_dispatch") {
				continue
			}
			j := i + 1
			for j < len(lines) && strings.HasPrefix(lines[j], " v") {
				j++
			}
			text := strings.Join(lines[i:j], "\n") + "\n"
			if b := out[text]; b != nil {
				b.count++
			} else {
				out[text] = &oscSelfBlock{tag: tag, lines: lines[i+1 : j], count: 1}
			}
			i = j - 1
		}
	}
	return out
}

// oscSelfDump is what the trace writes for the block: the action tag and the
// dump of the payload (trace.zig:35 Vt) or of the parser action (:58 parserFeed).
func oscSelfDump(b *oscSelfBlock, v reflect.Value) string {
	var buf bytes.Buffer
	buf.WriteString("@" + b.tag + "\n")
	if b.tag == "osc_dispatch" {
		dumpAny(&buf, " v", v.Interface())
	} else {
		dumpActionPayload(&buf, " v", v.Interface().(StreamAction), b.tag)
	}
	return buf.String()
}

// oscSelfRead reads the block into a StreamAction (the payload field of the tag)
// or a ParserAction.
func oscSelfRead(b *oscSelfBlock) (reflect.Value, error) {
	r := &oscSelfReader{lines: b.lines}
	if b.tag == "osc_dispatch" {
		v := reflect.New(reflect.TypeOf(ParserAction{})).Elem()
		err := r.value(" v", v)
		return v, err
	}
	v := reflect.New(reflect.TypeOf(StreamAction{})).Elem()
	f, ok := v.Type().FieldByName(zigFieldName(b.tag))
	if ok && len(b.lines) > 0 {
		if err := r.value(" v", dumpField(v, f.Index[0])); err != nil {
			return v, err
		}
	}
	if !ok && !slices.Contains(dumpVoidTags[v.Type()], b.tag) {
		return v, fmt.Errorf("no payload field for %s", b.tag)
	}
	if r.pos != len(b.lines) {
		return v, fmt.Errorf("%d lines left over: %q", len(b.lines)-r.pos, b.lines[r.pos])
	}
	return v, nil
}

// oscSelfVariants lists the variants that the value (a StreamAction or a
// ParserAction of the block) has: `command:<tag>` for osc_dispatch,
// `color:<request>[/<target>]`, `color_op:<op>`, `kitty:<request>/<kind>`,
// `progress:<state>`, `semantic:<action>`, `terminator:<t>`.
func oscSelfVariants(b *oscSelfBlock, v reflect.Value) []string {
	var out []string
	name := func(x any) string {
		n, _ := dumpTagName(reflect.ValueOf(x))
		return n
	}
	color := func(c OSCParsersColorList, term OSCTerminator, op OSCParsersColorOperation) {
		out = append(out, "color_op:"+name(op), "terminator:"+name(term))
		for _, r := range c.items {
			s := "color:" + name(r.Tag)
			switch r.Tag {
			case OSCParsersColorRequestSet:
				s += "/" + name(r.Set.target.Tag)
			case OSCParsersColorRequestQuery:
				s += "/" + name(r.Query.Tag)
			case OSCParsersColorRequestReset:
				s += "/" + name(r.Reset.Tag)
			}
			out = append(out, s)
		}
	}
	kitty := func(k KittyColorOSC) {
		out = append(out, "terminator:"+name(k.terminator))
		for _, r := range k.list {
			s := "kitty:" + name(r.Tag) + "/"
			switch r.Tag {
			case KittyColorOSCRequestQuery:
				s += name(r.Query.Tag)
			case KittyColorOSCRequestSet:
				s += name(r.Set.key.Tag)
			case KittyColorOSCRequestReset:
				s += name(r.Reset.Tag)
			}
			out = append(out, s)
		}
	}
	switch a := v.Interface().(type) {
	case ParserAction:
		out = append(out, "command:"+name(a.OscDispatch.Tag))
	case StreamAction:
		switch b.tag {
		case "color_operation":
			color(a.ColorOperation.requests, a.ColorOperation.terminator, a.ColorOperation.op)
		case "kitty_color_report":
			kitty(a.KittyColorReport)
		case "progress_report":
			out = append(out, "progress:"+name(a.ProgressReport.state))
		case "semantic_prompt":
			out = append(out, "semantic:"+name(a.SemanticPrompt.action))
		}
	}
	return out
}

// Every recorded block of an OSC action or of the parser action osc_dispatch
// is read into a Go value and dumped again: the text is the recorded one.
func TestCorpusReplaySelfOSCRoundTrip(t *testing.T) {
	oscSelfUseNames(t)
	blocks := oscSelfBlocks(t, oscSelfFiles(t))
	perTag := map[string]int{}
	covered := map[string]int{}
	bad := 0
	for text, b := range blocks {
		perTag[b.tag]++
		v, err := oscSelfRead(b)
		if err != nil {
			if bad++; bad <= 10 {
				t.Errorf("block %q: %v", text, err)
			}
			continue
		}
		var got string
		func() {
			defer func() {
				if r := recover(); r != nil {
					got = fmt.Sprint("panic: ", r)
				}
			}()
			got = oscSelfDump(b, v)
		}()
		if got != text {
			if bad++; bad <= 10 {
				t.Errorf("dump\n got  %q\n want %q", got, text)
			}
			continue
		}
		for _, x := range oscSelfVariants(b, v) {
			covered[x]++
		}
	}
	for _, tag := range append(slices.Clone(oscSelfActionTags), "osc_dispatch") {
		if perTag[tag] == 0 {
			t.Errorf("no recorded block of %s", tag)
		}
	}
	var keys []string
	for k := range covered {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("%d distinct recorded blocks; per tag %v", len(blocks), perTag)
	t.Logf("recorded variants: %v", keys)
	// What the corpus must have (the variants of the recorded dumps).
	for _, want := range []string{
		"command:change_window_title",
		"color:set/palette", "color:set/dynamic", "color:query/palette", "color:query/dynamic", "color:query/special",
		"color:reset/palette", "color:reset/dynamic", "color:reset_palette",
		"kitty:query/special", "kitty:set/special", "kitty:reset/special", "kitty:set/palette",
		"color_op:osc_4", "color_op:osc_104", "color_op:osc_119", "terminator:st", "terminator:bel",
		"progress:set", "progress:pause", "semantic:end_command",
	} {
		if covered[want] == 0 {
			t.Errorf("the recorded blocks have no %s", want)
		}
	}
}

// ---------------------------------------------------------------------------
// The values that the corpus does not record, built by hand.

// oscSelfCmd is the text that the parser trace writes for a Parser.Action
// osc_dispatch of the command `tag` with the payload lines (the part after
// ` v.osc_dispatch.<tag>`; none for a void tag): trace.zig:58 parserFeed.
func oscSelfCmd(tag string, payload ...string) string {
	out := " v=osc_dispatch\n v.osc_dispatch=" + tag + "\n"
	for _, l := range payload {
		out += " v.osc_dispatch." + tag + l + "\n"
	}
	return out
}

func oscSelfCmdDump(c OSCCommand) string {
	var buf bytes.Buffer
	dumpAny(&buf, " v", ParserAction{Tag: ParserActionOscDispatch, OscDispatch: c})
	return buf.String()
}

func oscSelfRGB(r, g, b uint8) ColorRGB { return ColorRGB{r: r, g: g, b: b} }

// Every variant of osc.Command, dumped as the parser trace dumps it. Where the
// corpus records the same payload (a stream action of the same type), the
// payload text is the recorded one, and the command text is that text under
// `v.osc_dispatch.<tag>`; the others (the corpus has no OSC sequence that
// yields them in the parser kind but change_window_title) have the text that
// the rules of dumpAny give, written out here from dump.zig.
func TestCorpusReplaySelfOSCVariants(t *testing.T) {
	oscSelfUseNames(t)
	str := func(s string) []uint8 { return []uint8(s) }
	cases := []struct {
		tag  string
		cmd  OSCCommand
		want string
	}{
		{"invalid", OSCCommand{Tag: OSCCommandKeyInvalid}, oscSelfCmd("invalid")},
		{"change_window_title", OSCCommand{Tag: OSCCommandKeyChangeWindowTitle, ChangeWindowTitle: str("tA")}, oscSelfCmd("change_window_title", "=tA")},
		{"change_window_icon", OSCCommand{Tag: OSCCommandKeyChangeWindowIcon, ChangeWindowIcon: str("i\\c\x01")}, oscSelfCmd("change_window_icon", "=i\\x5cc\\x01")},
		{"semantic_prompt", OSCCommand{Tag: OSCCommandKeySemanticPrompt, SemanticPrompt: OSCCommandSemanticPrompt{
			action: OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, options_unvalidated: str("k=c")}},
			oscSelfCmd("semantic_prompt", ".action=fresh_line_new_prompt", ".options_unvalidated=k=c")},
		{"clipboard_contents", OSCCommand{Tag: OSCCommandKeyClipboardContents, ClipboardContents: OSCCommandClipboardContentsPayload{kind: 'c', data: str("aGk=")}},
			oscSelfCmd("clipboard_contents", ".kind=99", ".data=aGk=")},
		{"report_pwd", OSCCommand{Tag: OSCCommandKeyReportPwd, ReportPwd: OSCCommandReportPwdPayload{value: str("file:///tmp")}},
			oscSelfCmd("report_pwd", ".value=file:///tmp")},
		{"mouse_shape", OSCCommand{Tag: OSCCommandKeyMouseShape, MouseShape: OSCCommandMouseShapePayload{value: str("pointer")}},
			oscSelfCmd("mouse_shape", ".value=pointer")},
		{"color_operation", OSCCommand{Tag: OSCCommandKeyColorOperation, ColorOperation: OSCCommandColorOperationPayload{
			op: OSCParsersColorOperationOsc4, terminator: OSCTerminatorBel,
			requests: OSCParsersColorList{items: []OSCParsersColorRequest{
				{Tag: OSCParsersColorRequestSet, Set: OSCParsersColorColoredTarget{target: OSCParsersColorTarget{Tag: OSCParsersColorTargetPalette, Palette: 1}, color: oscSelfRGB(0x12, 0x34, 0x56)}},
				{Tag: OSCParsersColorRequestResetSpecial},
			}}}},
			oscSelfCmd("color_operation", ".op=osc_4", ".requests.len=2", ".requests[0]:", ".requests[]=set",
				".requests[].set.target=palette", ".requests[].set.target.palette=1",
				".requests[].set.color.r=18", ".requests[].set.color.g=52", ".requests[].set.color.b=86",
				".requests[1]:", ".requests[]=reset_special", ".terminator=bel")},
		{"kitty_color_protocol", OSCCommand{Tag: OSCCommandKeyKittyColorProtocol, KittyColorProtocol: KittyColorOSC{terminator: OSCTerminatorSt,
			list: []KittyColorOSCRequest{{Tag: KittyColorOSCRequestQuery, Query: KittyColorKind{Tag: KittyColorKindSpecial, Special: KittyColorSpecialForeground}}}}},
			oscSelfCmd("kitty_color_protocol", ".list.len=1", ".list[0]:", ".list[]=query", ".list[].query=special",
				".list[].query.special=foreground", ".terminator=st")},
		{"show_desktop_notification", OSCCommand{Tag: OSCCommandKeyShowDesktopNotification, ShowDesktopNotification: OSCCommandShowDesktopNotificationPayload{title: str("a"), body: str("b")}},
			oscSelfCmd("show_desktop_notification", ".title=a", ".body=b")},
		{"hyperlink_start", OSCCommand{Tag: OSCCommandKeyHyperlinkStart, HyperlinkStart: OSCCommandHyperlinkStartPayload{id: Some(str("a")), uri: str("http://example.com")}},
			oscSelfCmd("hyperlink_start", ".id=a", ".uri=http://example.com")},
		{"hyperlink_start null id", OSCCommand{Tag: OSCCommandKeyHyperlinkStart, HyperlinkStart: OSCCommandHyperlinkStartPayload{uri: str("u")}},
			oscSelfCmd("hyperlink_start", ".id=null", ".uri=u")},
		{"hyperlink_end", OSCCommand{Tag: OSCCommandKeyHyperlinkEnd}, oscSelfCmd("hyperlink_end")},
		{"conemu_sleep", OSCCommand{Tag: OSCCommandKeyConemuSleep, ConemuSleep: OSCCommandConemuSleepPayload{duration_ms: 65535}},
			oscSelfCmd("conemu_sleep", ".duration_ms=65535")},
		{"conemu_show_message_box", OSCCommand{Tag: OSCCommandKeyConemuShowMessageBox, ConemuShowMessageBox: str("m")}, oscSelfCmd("conemu_show_message_box", "=m")},
		{"conemu_change_tab_title value", OSCCommand{Tag: OSCCommandKeyConemuChangeTabTitle, ConemuChangeTabTitle: OSCCommandConemuChangeTabTitlePayload{Tag: OSCCommandConemuChangeTabTitlePayloadValue, Value: str("tab")}},
			oscSelfCmd("conemu_change_tab_title", "=value", ".value=tab")},
		{"conemu_change_tab_title reset", OSCCommand{Tag: OSCCommandKeyConemuChangeTabTitle, ConemuChangeTabTitle: OSCCommandConemuChangeTabTitlePayload{Tag: OSCCommandConemuChangeTabTitlePayloadReset}},
			oscSelfCmd("conemu_change_tab_title", "=reset")},
		{"conemu_progress_report", OSCCommand{Tag: OSCCommandKeyConemuProgressReport, ConemuProgressReport: OSCCommandProgressReport{state: OSCCommandProgressReportStateSet, progress: Some(uint8(50))}},
			oscSelfCmd("conemu_progress_report", ".state=set", ".progress=50")},
		{"conemu_progress_report null", OSCCommand{Tag: OSCCommandKeyConemuProgressReport, ConemuProgressReport: OSCCommandProgressReport{state: OSCCommandProgressReportStateError}},
			oscSelfCmd("conemu_progress_report", ".state=error", ".progress=null")},
		{"conemu_progress_report unnamed", OSCCommand{Tag: OSCCommandKeyConemuProgressReport, ConemuProgressReport: OSCCommandProgressReport{state: 9}},
			oscSelfCmd("conemu_progress_report", ".state=#9", ".progress=null")},
		{"conemu_wait_input", OSCCommand{Tag: OSCCommandKeyConemuWaitInput}, oscSelfCmd("conemu_wait_input")},
		{"conemu_guimacro", OSCCommand{Tag: OSCCommandKeyConemuGuimacro, ConemuGuimacro: str("g")}, oscSelfCmd("conemu_guimacro", "=g")},
		{"conemu_run_process", OSCCommand{Tag: OSCCommandKeyConemuRunProcess, ConemuRunProcess: str("p")}, oscSelfCmd("conemu_run_process", "=p")},
		{"conemu_output_environment_variable", OSCCommand{Tag: OSCCommandKeyConemuOutputEnvironmentVariable, ConemuOutputEnvironmentVariable: str("e")},
			oscSelfCmd("conemu_output_environment_variable", "=e")},
		{"conemu_xterm_emulation", OSCCommand{Tag: OSCCommandKeyConemuXtermEmulation, ConemuXtermEmulation: OSCCommandConemuXtermEmulationPayload{keyboard: Some(true), output: Some(false)}},
			oscSelfCmd("conemu_xterm_emulation", ".keyboard=1", ".output=0")},
		{"conemu_xterm_emulation null", OSCCommand{Tag: OSCCommandKeyConemuXtermEmulation},
			oscSelfCmd("conemu_xterm_emulation", ".keyboard=null", ".output=null")},
		{"conemu_comment", OSCCommand{Tag: OSCCommandKeyConemuComment, ConemuComment: str("c")}, oscSelfCmd("conemu_comment", "=c")},
		{"kitty_text_sizing", OSCCommand{Tag: OSCCommandKeyKittyTextSizing, KittyTextSizing: OSCParsersKittyTextSizingOSC{
			scale: 2, width: 7, numerator: 1, denominator: 15,
			valign: OSCParsersKittyTextSizingVAlignCenter, halign: OSCParsersKittyTextSizingHAlignRight, text: str("hi")}},
			oscSelfCmd("kitty_text_sizing", ".scale=2", ".width=7", ".numerator=1", ".denominator=15", ".valign=center", ".halign=right", ".text=hi")},
		{"kitty_clipboard_protocol", OSCCommand{Tag: OSCCommandKeyKittyClipboardProtocol, KittyClipboardProtocol: OSCCommandKittyClipboardProtocol{
			metadata: str("type=read"), payload: Some(str("abc")), terminator: OSCTerminatorBel}},
			oscSelfCmd("kitty_clipboard_protocol", ".metadata=type=read", ".payload=abc", ".terminator=bel")},
		{"kitty_clipboard_protocol null", OSCCommand{Tag: OSCCommandKeyKittyClipboardProtocol, KittyClipboardProtocol: OSCCommandKittyClipboardProtocol{metadata: str("m")}},
			oscSelfCmd("kitty_clipboard_protocol", ".metadata=m", ".payload=null", ".terminator=st")},
		{"context_signal", OSCCommand{Tag: OSCCommandKeyContextSignal, ContextSignal: OSCParsersContextSignalCommand{
			action: OSCParsersContextSignalCommandActionEnd, id: str("abc"), metadata: str("type=shell")}},
			oscSelfCmd("context_signal", ".action=end", ".id=abc", ".metadata=type=shell")},
	}
	seen := map[string]bool{}
	for _, c := range cases {
		if got := oscSelfCmdDump(c.cmd); got != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.tag, got, c.want)
		}
		seen[strings.Fields(c.tag)[0]] = true
	}
	// Every variant of osc.Command is built above.
	for _, zu := range zigScanUnions("osc.zig", oscSelfZigText(t, "osc.zig")) {
		if len(zu.fields) > 0 && zu.fields[0].name == "invalid" {
			for _, f := range zu.fields {
				if !seen[f.name] {
					t.Errorf("osc.Command variant %s: no hand-built value", f.name)
				}
			}
		}
	}

	// The same payloads as stream actions, with the recorded text.
	recorded := map[string]string{}
	for _, f := range oscSelfFiles(t) {
		if strings.HasSuffix(f, ".actions.gz") {
			recorded[f] = string(corpusReadAll(t, "golden/"+f))
		}
	}
	has := func(block string) bool {
		for _, r := range recorded {
			if strings.Contains(r, block) {
				return true
			}
		}
		return false
	}
	for _, c := range []struct {
		tag   string
		value StreamAction
		text  string
	}{
		{"semantic_prompt", StreamAction{SemanticPrompt: OSCCommandSemanticPrompt{action: OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, options_unvalidated: str("k=c")}},
			" v.action=fresh_line_new_prompt\n v.options_unvalidated=k=c\n"},
		{"progress_report", StreamAction{ProgressReport: OSCCommandProgressReport{state: OSCCommandProgressReportStateSet, progress: Some(uint8(50))}},
			" v.state=set\n v.progress=50\n"},
		{"mouse_shape", StreamAction{MouseShape: MouseShapePointer}, " v=pointer\n"},
		{"clipboard_contents", StreamAction{ClipboardContents: StreamActionClipboardContents{kind: 'c', data: str("aGk=")}}, " v.kind=99\n v.data=aGk=\n"},
		{"show_desktop_notification", StreamAction{ShowDesktopNotification: StreamActionShowDesktopNotification{title: str("a"), body: str("b")}}, " v.title=a\n v.body=b\n"},
		{"end_hyperlink", StreamAction{}, ""},
		{"kitty_color_report", StreamAction{KittyColorReport: KittyColorOSC{terminator: OSCTerminatorSt,
			list: []KittyColorOSCRequest{{Tag: KittyColorOSCRequestQuery, Query: KittyColorKind{Tag: KittyColorKindSpecial, Special: KittyColorSpecialForeground}}}}},
			" v.list.len=1\n v.list[0]:\n v.list[]=query\n v.list[].query=special\n v.list[].query.special=foreground\n v.terminator=st\n"},
		{"color_operation", StreamAction{ColorOperation: StreamActionColorOperation{op: OSCParsersColorOperationOsc4, terminator: OSCTerminatorSt,
			requests: OSCParsersColorList{items: []OSCParsersColorRequest{{Tag: OSCParsersColorRequestSet,
				Set: OSCParsersColorColoredTarget{target: OSCParsersColorTarget{Tag: OSCParsersColorTargetPalette, Palette: 1}, color: oscSelfRGB(0x12, 0x34, 0x56)}}}}}},
			" v.op=osc_4\n v.requests.len=1\n v.requests[0]:\n v.requests[]=set\n v.requests[].set.target=palette\n v.requests[].set.target.palette=1\n" +
				" v.requests[].set.color.r=18\n v.requests[].set.color.g=52\n v.requests[].set.color.b=86\n v.terminator=st\n"},
	} {
		var buf bytes.Buffer
		buf.WriteString("@" + c.tag + "\n")
		dumpActionPayload(&buf, " v", c.value, c.tag)
		if got := buf.String(); got != "@"+c.tag+"\n"+c.text {
			t.Errorf("%s: dump %q, want %q", c.tag, got, c.text)
		}
		if !has(buf.String()) {
			t.Errorf("%s: %q is not in the recorded actions", c.tag, buf.String())
		}
	}

	// The colour requests that the corpus does not record (reset_special, and
	// osc_105): the lines follow the rules of the union and enum arms.
	var buf bytes.Buffer
	dumpAny(&buf, " v", StreamActionColorOperation{op: OSCParsersColorOperationOsc105, terminator: OSCTerminatorBel,
		requests: OSCParsersColorList{items: []OSCParsersColorRequest{
			{Tag: OSCParsersColorRequestResetSpecial},
			{Tag: OSCParsersColorRequestQuery, Query: OSCParsersColorTarget{Tag: OSCParsersColorTargetSpecial, Special: ColorSpecialItalic}},
			{Tag: OSCParsersColorRequestReset, Reset: OSCParsersColorTarget{Tag: OSCParsersColorTargetDynamic, Dynamic: ColorDynamicHighlightForeground}},
		}}})
	want := " v.op=osc_105\n v.requests.len=3\n v.requests[0]:\n v.requests[]=reset_special\n v.requests[1]:\n v.requests[]=query\n" +
		" v.requests[].query=special\n v.requests[].query.special=italic\n v.requests[2]:\n v.requests[]=reset\n" +
		" v.requests[].reset=dynamic\n v.requests[].reset.dynamic=highlight_foreground\n v.terminator=bel\n"
	if buf.String() != want {
		t.Errorf("unrecorded colour requests:\n got  %q\n want %q", buf.String(), want)
	}
	// An empty list and an unnamed enum value.
	buf.Reset()
	dumpAny(&buf, " v", StreamActionColorOperation{op: 77})
	if got, want := buf.String(), " v.op=#77\n v.requests.len=0\n v.terminator=st\n"; got != want {
		t.Errorf("empty list %q, want %q", got, want)
	}
}

// Types the self-test uses to check that a type which cannot be mapped fails
// loudly. They stand in for wireframe types without String: the OSC types
// that once lacked it now have it (dde4530), so they no longer exercise
// this path.
type (
	oscSelfNoStringTag   uint8
	oscSelfNoStringEnum  uint8
	oscSelfNoStringUnion struct {
		Tag     oscSelfNoStringTag
		Palette uint8
	}
	oscSelfNoStringHolder struct {
		action oscSelfNoStringEnum
	}
)

const oscSelfNoStringTagPalette oscSelfNoStringTag = 1

// A wireframe type that cannot be mapped fails loudly, naming the type: a tag
// type without String, and an enum type without String.
func TestCorpusReplaySelfOSCLoudFailures(t *testing.T) {
	for _, c := range []struct {
		name string
		v    any
		want string
	}{
		{"tag without String", oscSelfNoStringUnion{Tag: oscSelfNoStringTagPalette}, "oscSelfNoStringTag"},
		{"enum without String", oscSelfNoStringHolder{}, "oscSelfNoStringEnum"},
		{"enum slice without String", []oscSelfNoStringEnum{1}, "oscSelfNoStringEnum"},
	} {
		func() {
			defer func() {
				r := fmt.Sprint(recover())
				if !strings.Contains(r, c.want) {
					t.Errorf("%s: panic %q lacks %q", c.name, r, c.want)
				}
			}()
			var buf bytes.Buffer
			dumpAny(&buf, "p", c.v)
			t.Errorf("%s: no panic", c.name)
		}()
	}
	// An unnamed tag of a union is loud too (dumpUnion).
	func() {
		defer func() {
			if r := fmt.Sprint(recover()); !strings.Contains(r, "unnamed tag") {
				t.Errorf("panic %q", r)
			}
		}()
		oscSelfUseNames(t)
		var buf bytes.Buffer
		dumpAny(&buf, "p", OSCCommand{Tag: 99})
		t.Error("no panic for an unnamed tag")
	}()
}
