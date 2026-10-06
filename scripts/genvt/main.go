// genvt generates the public packages of the module from package terminal.
//
// Upstream's public API is src/lib_vt.zig: a list of top-level names (Terminal,
// Screen, Cell = page.Cell, ...) and of namespaces (osc, sgr, modes, page,
// ...). In the Go port every declaration lives in package terminal with its
// Zig namespace as a name prefix (OSCParser, SGRAttribute, ModesMode). This
// tool gives each of upstream's public names its Go name:
//
//   - package vt: the top-level names of lib_vt.zig (table `top` below);
//   - packages vt/osc, vt/sgr, ...: the namespaces (table `namespaces`).
//
// A terminal declaration whose name starts with a table entry's prefix is
// re-exported under the entry's new prefix: types as aliases, constants as
// constants, functions as wrappers (`<Type>Init` becomes `New<Type>`),
// error values as variables, and other package variables (default values) as
// functions returning the value. A declaration whose signature needs a type
// that is not exported is left out and reported; nothing is changed in
// package terminal.
//
// usage: go run ./scripts/genvt   (from the module root)
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

const termPath = "github.com/XIAZY/libghostty-vt-go/terminal"

// entry maps the declarations named Prefix... to New... in package Pkg.
type entry struct{ Prefix, New string }

// top is lib_vt.zig's list of top-level names: Zig name <- terminal name.
var top = []entry{
	{"CharsetsCharset", "Charset"}, {"CharsetsSlots", "CharsetSlot"}, {"CharsetsActiveSlot", "CharsetActiveSlot"},
	{"PageCell", "Cell"}, {"PointCoordinate", "Coordinate"},
	{"ParserActionCSI", "CSI"}, {"ParserActionDCS", "DCS"},
	{"MouseShape", "MouseShape"}, {"Page", "Page"}, {"PageList", "PageList"}, {"Parser", "Parser"},
	{"PageListPin", "Pin"}, {"Point", "Point"}, {"RenderState", "RenderState"},
	{"Screen", "Screen"}, {"ScreenSet", "ScreenSet"}, {"Selection", "Selection"},
	{"CSISizeReportStyle", "SizeReportStyle"}, {"StringMap", "StringMap"}, {"Style", "Style"},
	{"Terminal", "Terminal"}, {"Stream", "Stream"}, {"StreamAction", "StreamAction"},
	{"StreamReadonlyStream", "ReadonlyStream"}, {"StreamReadonlyHandler", "ReadonlyHandler"},
	{"ScreenCursor", "Cursor"}, {"ScreenCursorStyle", "CursorStyle"},
	{"ANSICursorStyle", "CursorStyleReq"}, {"ANSIDeviceAttributeReq", "DeviceAttributeReq"},
	{"ModesMode", "Mode"}, {"ModesModePacked", "ModePacked"},
	{"ANSIModifyKeyFormat", "ModifyKeyFormat"}, {"ANSIProtectedMode", "ProtectedMode"},
	{"ANSIStatusLineType", "StatusLineType"}, {"ANSIStatusDisplay", "StatusDisplay"},
	{"CSIEraseDisplay", "EraseDisplay"}, {"CSIEraseLine", "EraseLine"}, {"CSITabClear", "TabClear"},
	{"SGRAttribute", "Attribute"},
	// The values and helpers the names above need.
	{"Optional", "Optional"}, {"Some", "Some"}, {"None", "None"}, {"Err", "Err"},
}

// namespaces is lib_vt.zig's list of namespaces: Go package <- terminal
// prefix. `input` is upstream's nested struct of the same name.
var namespaces = map[string][]entry{
	"apc":          {{"APC", ""}},
	"dcs":          {{"DCS", ""}},
	"osc":          {{"OSC", ""}},
	"point":        {{"Point", ""}},
	"color":        {{"Color", ""}},
	"devicestatus": {{"DeviceStatus", ""}},
	"formatter":    {{"Formatter", ""}},
	"highlight":    {{"Highlight", ""}},
	"kitty":        {{"Kitty", ""}},
	"modes":        {{"Modes", ""}},
	"page":         {{"Page", ""}},
	"search":       {{"Search", ""}},
	"sgr":          {{"SGR", ""}},
	"size":         {{"Size", ""}},
	"x11color":     {{"X11Color", ""}},
	"input": {
		{"InputPasteOptions", "PasteOptions"}, {"InputKeyKey", "Key"}, {"InputKeyAction", "KeyAction"},
		{"InputKeyKeyEvent", "KeyEvent"}, {"InputKeyMods", "KeyMods"}, {"InputKeyModsMods", "KeyMods"},
		{"InputKeyEncodeOptions", "KeyEncodeOptions"},
		{"InputKeyEncodeEncode", "EncodeKey"}, {"InputPasteIsSafe", "IsSafePaste"}, {"InputPasteEncode", "EncodePaste"},
	},
}

// exclude lists prefixes that a namespace's own prefix would wrongly take
// (PageList is not part of page.zig) and the switches only tests change.
var exclude = map[string][]string{
	"page": {"PageList", "PageStdCapacity"},
	"vt":   {"PageStdCapacity"},
}

// hand lists names written by hand in the package (the generator skips
// them).
var hand = map[string]map[string]bool{}

// lowerCamel turns a Zig parameter name (max_size) into a Go one (maxSize).
func lowerCamel(s string) string {
	parts := strings.Split(strings.Trim(s, "_"), "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

type gen struct {
	pkg     *types.Package
	skipped []string
}

func main() {
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax}
	pkgs, err := packages.Load(cfg, termPath)
	if err != nil || len(pkgs) != 1 || len(pkgs[0].Errors) > 0 {
		fmt.Fprintln(os.Stderr, "load:", err, pkgs[0].Errors)
		os.Exit(1)
	}
	g := &gen{pkg: pkgs[0].Types}
	g.emit("vt", "vt", top, "Package vt is the public API of libghostty-vt-go: the top-level names of\n// upstream's src/lib_vt.zig. Its namespaces are the packages below it\n// (vt/osc, vt/sgr, vt/modes, ...).")
	var names []string
	for n := range namespaces {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		g.emit(filepath.Join("vt", n), n, namespaces[n], fmt.Sprintf("Package %s is upstream's `%s` namespace of the libghostty-vt public API\n// (src/lib_vt.zig).", n, n))
	}
	sort.Strings(g.skipped)
	if err := os.WriteFile(filepath.Join("vt", "zz_skipped.txt"), []byte(strings.Join(g.skipped, "\n")+"\n"), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("skipped %d declarations (vt/zz_skipped.txt)\n", len(g.skipped))
}

// nameable reports whether t can be written outside package terminal.
func (g *gen) nameable(t types.Type, seen map[types.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	switch t := t.(type) {
	case *types.Named:
		if o := t.Obj(); o.Pkg() == g.pkg && !o.Exported() {
			return false
		}
		if ta := t.TypeArgs(); ta != nil {
			for i := 0; i < ta.Len(); i++ {
				if !g.nameable(ta.At(i), seen) {
					return false
				}
			}
		}
		return true
	case *types.Alias:
		if o := t.Obj(); o.Pkg() == g.pkg && !o.Exported() {
			return false
		}
		return true
	case *types.Pointer:
		return g.nameable(t.Elem(), seen)
	case *types.Slice:
		return g.nameable(t.Elem(), seen)
	case *types.Array:
		return g.nameable(t.Elem(), seen)
	case *types.Map:
		return g.nameable(t.Key(), seen) && g.nameable(t.Elem(), seen)
	case *types.Chan:
		return g.nameable(t.Elem(), seen)
	case *types.Signature:
		for _, tup := range []*types.Tuple{t.Params(), t.Results()} {
			for i := 0; i < tup.Len(); i++ {
				if !g.nameable(tup.At(i).Type(), seen) {
					return false
				}
			}
		}
		return true
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if !t.Field(i).Exported() || !g.nameable(t.Field(i).Type(), seen) {
				return false
			}
		}
		return true
	case *types.Interface:
		for i := 0; i < t.NumExplicitMethods(); i++ {
			if !t.ExplicitMethod(i).Exported() {
				return false
			}
		}
		return true
	case *types.TypeParam:
		return true
	}
	return true
}

func (g *gen) emit(dir, name string, entries []entry, doc string) {
	qual := func(p *types.Package) string {
		if p == g.pkg {
			return "terminal"
		}
		return p.Name()
	}
	type decl struct {
		name, text string
		typ        bool
	}
	var decls []decl
	used := map[string]string{}
	scope := g.pkg.Scope()
	imports := map[string]bool{}
	for _, on := range scope.Names() {
		obj := scope.Lookup(on)
		if !obj.Exported() {
			continue
		}
		skipIt := false
		for _, x := range exclude[name] {
			if strings.HasPrefix(on, x) {
				skipIt = true
			}
		}
		if skipIt {
			continue
		}
		// The longest matching prefix decides the new name.
		best := -1
		for i, e := range entries {
			if strings.HasPrefix(on, e.Prefix) && (best < 0 || len(e.Prefix) > len(entries[best].Prefix)) {
				// A prefix must end at a word boundary: "Page" does not
				// take "Pager", "Point" does not take "Pointer".
				rest := on[len(e.Prefix):]
				if rest != "" && rest[0] >= 'a' && rest[0] <= 'z' {
					continue
				}
				best = i
			}
		}
		if best < 0 {
			continue
		}
		e := entries[best]
		rest := on[len(e.Prefix):]
		nn := e.New + rest
		if nn == "" {
			nn = on // point.Point, page.Page
		}
		if nn[0] >= '0' && nn[0] <= '9' {
			nn = "X" + nn
		}
		if _, isFunc := obj.(*types.Func); isFunc {
			if strings.HasSuffix(rest, "Init") && (e.New+strings.TrimSuffix(rest, "Init")) != "" {
				nn = "New" + e.New + strings.TrimSuffix(rest, "Init")
			} else if rest == "Init" {
				nn = "New"
			}
		}
		if hand[name][nn] {
			continue
		}
		where := name + "." + nn + " = terminal." + on
		if prev, dup := used[nn]; dup {
			g.skipped = append(g.skipped, where+": name already taken by terminal."+prev)
			continue
		}
		var b strings.Builder
		switch o := obj.(type) {
		case *types.TypeName:
			if named, ok := o.Type().(*types.Named); ok && named.TypeParams().Len() > 0 {
				tp := named.TypeParams()
				var ps, as []string
				ok := true
				for i := 0; i < tp.Len(); i++ {
					c := tp.At(i).Constraint()
					if !g.nameable(c, map[types.Type]bool{}) {
						ok = false
					}
					ps = append(ps, tp.At(i).Obj().Name()+" "+types.TypeString(c, qual))
					as = append(as, tp.At(i).Obj().Name())
				}
				if !ok {
					g.skipped = append(g.skipped, where+": type parameter constraint is not exported")
					continue
				}
				fmt.Fprintf(&b, "// %s is [terminal.%s].\ntype %s[%s] = terminal.%s[%s]\n", nn, on, nn, strings.Join(ps, ", "), on, strings.Join(as, ", "))
			} else {
				fmt.Fprintf(&b, "// %s is [terminal.%s].\ntype %s = terminal.%s\n", nn, on, nn, on)
			}
		case *types.Const:
			fmt.Fprintf(&b, "// %s is [terminal.%s].\nconst %s = terminal.%s\n", nn, on, nn, on)
		case *types.Var:
			if !g.nameable(o.Type(), map[types.Type]bool{}) {
				g.skipped = append(g.skipped, where+": its type is not exported")
				continue
			}
			if types.Implements(o.Type(), types.Universe.Lookup("error").Type().Underlying().(*types.Interface)) {
				fmt.Fprintf(&b, "// %s is [terminal.%s].\nvar %s = terminal.%s\n", nn, on, nn, on)
			} else {
				fmt.Fprintf(&b, "// %s returns [terminal.%s].\nfunc %s() %s { return terminal.%s }\n", nn, on, nn, types.TypeString(o.Type(), qual), on)
			}
		case *types.Func:
			sig := o.Type().(*types.Signature)
			if !g.nameable(sig, map[types.Type]bool{}) {
				g.skipped = append(g.skipped, where+": its signature needs a type that is not exported")
				continue
			}
			var tps, tas string
			if tp := sig.TypeParams(); tp.Len() > 0 {
				var ps, as []string
				ok := true
				for i := 0; i < tp.Len(); i++ {
					c := tp.At(i).Constraint()
					if !g.nameable(c, map[types.Type]bool{}) {
						ok = false
					}
					ps = append(ps, tp.At(i).Obj().Name()+" "+types.TypeString(c, qual))
					as = append(as, tp.At(i).Obj().Name())
				}
				if !ok {
					g.skipped = append(g.skipped, where+": type parameter constraint is not exported")
					continue
				}
				tps, tas = "["+strings.Join(ps, ", ")+"]", "["+strings.Join(as, ", ")+"]"
			}
			var params, args []string
			for i := 0; i < sig.Params().Len(); i++ {
				p := sig.Params().At(i)
				pn := p.Name()
				pn = lowerCamel(pn)
				if pn == "" || pn == "_" || pn == "terminal" || token.Lookup(pn).IsKeyword() {
					pn = fmt.Sprintf("a%d", i)
				}
				ts := types.TypeString(p.Type(), qual)
				arg := pn
				if sig.Variadic() && i == sig.Params().Len()-1 {
					ts = "..." + types.TypeString(p.Type().(*types.Slice).Elem(), qual)
					arg += "..."
				}
				params = append(params, pn+" "+ts)
				args = append(args, arg)
			}
			res := ""
			if n := sig.Results().Len(); n > 0 {
				var rs []string
				for i := 0; i < n; i++ {
					rs = append(rs, types.TypeString(sig.Results().At(i).Type(), qual))
				}
				res = " (" + strings.Join(rs, ", ") + ")"
			}
			ret := "return "
			if sig.Results().Len() == 0 {
				ret = ""
			}
			fmt.Fprintf(&b, "// %s is [terminal.%s].\nfunc %s%s(%s)%s { %sterminal.%s%s(%s) }\n", nn, on, nn, tps, strings.Join(params, ", "), res, ret, on, tas, strings.Join(args, ", "))
		default:
			continue
		}
		used[nn] = on
		_, isType := obj.(*types.TypeName)
		decls = append(decls, decl{nn, b.String(), isType})
	}
	// Signatures name the package's own aliases (Cell, not
	// terminal.PageCell) where there is one.
	local := map[string]string{}
	for _, d := range decls {
		if d.typ {
			local[used[d.name]] = d.name
		}
	}
	re := regexp.MustCompile(`\bterminal\.([A-Za-z0-9_]+)`)
	for i, d := range decls {
		if d.typ {
			continue
		}
		head, body, found := strings.Cut(d.text, " { ")
		if !found {
			continue
		}
		lines := strings.SplitN(head, "\n", 2) // the doc line keeps the terminal name
		lines[1] = re.ReplaceAllStringFunc(lines[1], func(m string) string {
			if n, ok := local[strings.TrimPrefix(m, "terminal.")]; ok {
				return n
			}
			return m
		})
		decls[i].text = strings.Join(lines, "\n") + " { " + body
	}
	sort.Slice(decls, func(i, j int) bool { return decls[i].name < decls[j].name })
	var out bytes.Buffer
	fmt.Fprintf(&out, "// Code generated by scripts/genvt from package terminal; DO NOT EDIT.\n\n// %s\npackage %s\n\nimport (\n", doc, name)
	_ = imports
	fmt.Fprintf(&out, "\t%q\n)\n\n", termPath)
	for _, d := range decls {
		out.WriteString(d.text)
		out.WriteString("\n")
	}
	src, err := format.Source(out.Bytes())
	if err != nil {
		os.WriteFile("genvt_failed.go.txt", out.Bytes(), 0o644)
		panic(fmt.Sprintf("%s: %v", dir, err))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "zz_generated.go"), src, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("%s: %d declarations\n", dir, len(decls))
}
