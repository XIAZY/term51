// exportfields renames the unexported fields of exported struct types in a
// package to exported names (snake_case -> CamelCase), updating every use,
// tests included. Zig struct fields are always public; this gives the Go
// port the same visibility.
//
// usage: exportfields [-w] <package dir>
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

func camel(s string) string {
	trail := strings.HasSuffix(s, "_") && len(s) > 1
	parts := strings.Split(strings.Trim(s, "_"), "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(p)
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	out := b.String()
	_ = trail
	return out
}

// snake is the inverse the corpus replayer uses (zigSnake).
func snake(name string) string {
	var sb strings.Builder
	for i, c := range name {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				sb.WriteByte('_')
			}
			c += 'a' - 'A'
		}
		sb.WriteRune(c)
	}
	return sb.String()
}

type edit struct {
	off int
	old string
	new string
}

func main() {
	write := flag.Bool("w", false, "write the files")
	allowFile := flag.String("types", "", "file listing the types whose fields are exported, one name or prefix* per line (default: every exported struct type)")
	flag.Parse()
	dir := flag.Arg(0)
	var allowExact = map[string]bool{}
	var allowPrefix []string
	if *allowFile != "" {
		b, err := os.ReadFile(*allowFile)
		if err != nil {
			panic(err)
		}
		for _, l := range strings.Split(string(b), "\n") {
			l = strings.TrimSpace(l)
			if l == "" || strings.HasPrefix(l, "#") {
				continue
			}
			if strings.HasSuffix(l, "*") {
				allowPrefix = append(allowPrefix, strings.TrimSuffix(l, "*"))
			} else {
				allowExact[l] = true
			}
		}
	}
	allowed := func(name string) bool {
		if *allowFile == "" || allowExact[name] {
			return true
		}
		for _, p := range allowPrefix {
			if strings.HasPrefix(name, p) {
				return true
			}
		}
		return false
	}
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedCompiledGoFiles, Dir: dir, Tests: true}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		panic(err)
	}
	// Pick the package variant that includes the in-package tests.
	var pkg *packages.Package
	for _, p := range pkgs {
		if len(p.Errors) > 0 {
			for _, e := range p.Errors {
				fmt.Fprintln(os.Stderr, e)
			}
			os.Exit(1)
		}
		if strings.HasSuffix(p.ID, ".test") {
			continue
		}
		if pkg == nil || len(p.Syntax) > len(pkg.Syntax) {
			pkg = p
		}
	}
	fset := pkg.Fset
	info := pkg.TypesInfo

	// Collect the fields to rename: unexported fields of structs reachable
	// from exported named types of this package (nested anonymous structs
	// included).
	rename := map[*types.Var]string{}
	var collisions []string
	seen := map[types.Type]bool{}
	var visitStruct func(owner *types.Named, st *types.Struct, path string)
	visitStruct = func(owner *types.Named, st *types.Struct, path string) {
		if seen[st] {
			return
		}
		seen[st] = true
		names := map[string]bool{}
		for i := 0; i < st.NumFields(); i++ {
			names[st.Field(i).Name()] = true
		}
		if owner != nil {
			for i := 0; i < owner.NumMethods(); i++ {
				names[owner.Method(i).Name()] = true
			}
		}
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			if f.Pkg() != pkg.Types {
				continue
			}
			if !f.Exported() && !f.Embedded() && !strings.HasPrefix(f.Name(), "_") {
				n := camel(f.Name())
				if snake(n) != strings.TrimSuffix(f.Name(), "_") {
					fmt.Printf("no round trip: %s.%s -> %s -> %s\n", path, f.Name(), n, snake(n))
				}
				if names[n] {
					// A method already has the name: the field stays
					// unexported and the method is the access.
					collisions = append(collisions, fmt.Sprintf("%s.%s -> %s (taken)", path, f.Name(), n))
				} else {
					names[n] = true
					rename[f] = n
				}
			}
			// Nested anonymous struct types (directly, or behind pointer,
			// slice, array, map).
			t := f.Type()
			for {
				switch u := t.(type) {
				case *types.Pointer:
					t = u.Elem()
					continue
				case *types.Slice:
					t = u.Elem()
					continue
				case *types.Array:
					t = u.Elem()
					continue
				}
				break
			}
			if s, ok := t.(*types.Struct); ok {
				visitStruct(nil, s, path+"."+f.Name())
			}
		}
	}
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		tn, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || !tn.Exported() || tn.IsAlias() || !allowed(name) {
			continue
		}
		named, ok := tn.Type().(*types.Named)
		if !ok {
			continue
		}
		if st, ok := named.Underlying().(*types.Struct); ok {
			visitStruct(named, st, name)
		}
	}

	// Every identifier that defines or uses one of those fields.
	edits := map[string][]edit{}
	add := func(id *ast.Ident, obj types.Object) {
		v, ok := obj.(*types.Var)
		if !ok || !v.IsField() {
			return
		}
		v = v.Origin()
		n, ok := rename[v]
		if !ok {
			return
		}
		pos := fset.Position(id.Pos())
		edits[pos.Filename] = append(edits[pos.Filename], edit{pos.Offset, id.Name, n})
	}
	for id, obj := range info.Defs {
		if obj != nil {
			add(id, obj)
		}
	}
	for id, obj := range info.Uses {
		add(id, obj)
	}

	total := 0
	for file, es := range edits {
		sort.Slice(es, func(i, j int) bool { return es[i].off > es[j].off })
		src, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		last := -1
		for _, e := range es {
			if e.off == last {
				continue
			}
			last = e.off
			if string(src[e.off:e.off+len(e.old)]) != e.old {
				panic(fmt.Sprintf("%s:%d: expected %q", file, e.off, e.old))
			}
			src = append(src[:e.off:e.off], append([]byte(e.new), src[e.off+len(e.old):]...)...)
			total++
		}
		if *write {
			if err := os.WriteFile(file, src, 0o644); err != nil {
				panic(err)
			}
		}
	}
	sort.Strings(collisions)
	for _, c := range collisions {
		fmt.Println("collision:", c)
	}
	fmt.Printf("fields renamed: %d, identifiers rewritten: %d, files: %d, collisions: %d\n", len(rename), total, len(edits), len(collisions))
}
