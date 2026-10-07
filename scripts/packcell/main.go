// packcell rewrites the uses of the fields of PageCell and PageCellContent
// in a package for the packed cell (one 64-bit value, as upstream's
// `packed struct(u64)`): a field read becomes a getter call, a field
// assignment a setter call, and a composite literal becomes a literal of the
// plain "fields" struct followed by .pack(). It resolves every name with the
// type checker and reports what it cannot rewrite.
//
// usage: packcell [-w] <package dir>
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

var cellGetter = map[string]string{
	"content_tag":      "ContentTag",
	"content":          "content",
	"style_id":         "StyleID",
	"wide":             "Wide",
	"protected":        "Protected",
	"hyperlink":        "Hyperlink",
	"semantic_content": "SemanticContent",
}
var cellSetter = map[string]string{
	"content_tag":      "setContentTag",
	"content":          "setContent",
	"style_id":         "setStyleID",
	"wide":             "setWide",
	"protected":        "setProtected",
	"hyperlink":        "setHyperlink",
	"semantic_content": "setSemanticContent",
}
var contentGetter = map[string]string{
	"codepoint":     "codepoint",
	"color_palette": "colorPalette",
	"color_rgb":     "colorRGB",
}
var contentSetter = map[string]string{
	"codepoint":     "setContentCodepoint",
	"color_palette": "setContentColorPalette",
	"color_rgb":     "setContentColorRGB",
}
var litType = map[string]string{
	"PageCell":        "pageCellFields",
	"PageCellContent": "pageCellContentFields",
}

type edit struct {
	start, end int
	text       string
}

func main() {
	write := flag.Bool("w", false, "write the files")
	flag.Parse()
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo, Dir: flag.Arg(0), Tests: true}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		panic(err)
	}
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
	fset, info := pkg.Fset, pkg.TypesInfo

	// The field objects of the two structs.
	owner := map[*types.Var]string{}
	for _, name := range []string{"PageCell", "PageCellContent"} {
		st := pkg.Types.Scope().Lookup(name).Type().Underlying().(*types.Struct)
		for i := 0; i < st.NumFields(); i++ {
			owner[st.Field(i)] = name
		}
	}
	namedOf := func(t types.Type) string {
		if p, ok := t.(*types.Pointer); ok {
			t = p.Elem()
		}
		if n, ok := t.(*types.Named); ok && n.Obj().Pkg() == pkg.Types {
			return n.Obj().Name()
		}
		return ""
	}

	edits := map[string][]edit{}
	var manual []string
	counts := map[string]int{}
	off := func(p token.Pos) int { return fset.Position(p).Offset }
	add := func(file string, s, e token.Pos, text string) {
		edits[file] = append(edits[file], edit{off(s), off(e), text})
	}
	report := func(n ast.Node, why string) {
		manual = append(manual, fmt.Sprintf("%s: %s", fset.Position(n.Pos()), why))
	}

	for _, f := range pkg.Syntax {
		file := fset.Position(f.Pos()).Filename
		// Parents, to know the context of each selector.
		parent := map[ast.Node]ast.Node{}
		var stack []ast.Node
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			if len(stack) > 0 {
				parent[n] = stack[len(stack)-1]
			}
			stack = append(stack, n)
			return true
		})
		handledSel := map[*ast.SelectorExpr]bool{}

		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CompositeLit:
				t := info.TypeOf(x)
				if t == nil {
					return true
				}
				name := namedOf(t)
				if _, isPtr := t.(*types.Pointer); isPtr {
					name = ""
				}
				nt, ok := litType[name]
				if !ok {
					return true
				}
				if x.Type == nil {
					report(x, "composite literal of "+name+" with an elided type")
					return true
				}
				// `&T{...}`: a packed value has no address of its own.
				if u, ok := parent[x].(*ast.UnaryExpr); ok && u.Op == token.AND {
					if len(x.Elts) != 0 {
						report(x, "address of a composite literal of "+name)
						return true
					}
					add(file, u.Pos(), x.End(), "new("+name+")")
					counts["zero literal"]++
					return true
				}
				if len(x.Elts) == 0 {
					add(file, x.Pos(), x.End(), name+"(0)")
					counts["zero literal"]++
					return true
				}
				for _, e := range x.Elts {
					if _, ok := e.(*ast.KeyValueExpr); !ok {
						report(x, "composite literal of "+name+" without field names")
						return true
					}
				}
				add(file, x.Type.Pos(), x.Type.End(), nt)
				add(file, x.End(), x.End(), ".pack()")
				counts["literal"]++

			case *ast.SelectorExpr:
				v, ok := info.Uses[x.Sel].(*types.Var)
				if !ok {
					return true
				}
				own, ok := owner[v]
				if !ok || handledSel[x] {
					return true
				}
				field := x.Sel.Name
				if field == "_padding" {
					report(x, "use of _padding")
					return true
				}
				p := parent[x]
				// `cell.content.g`: handled at the outer selector when it
				// is assigned to; otherwise each selector becomes a getter.
				if own == "PageCellContent" {
					if as, ok := p.(*ast.AssignStmt); ok && isLHS(as, x) {
						inner, ok := x.X.(*ast.SelectorExpr)
						iv, _ := info.Uses[selIdent(inner)].(*types.Var)
						if !ok || owner[iv] != "PageCell" || iv.Name() != "content" {
							report(x, "assignment to a field of a PageCellContent that is not `cell.content`")
							return true
						}
						if as.Tok != token.ASSIGN || len(as.Lhs) != 1 {
							report(x, "assignment form "+as.Tok.String())
							return true
						}
						handledSel[inner] = true
						add(file, inner.Sel.Pos(), as.Rhs[0].Pos(), contentSetter[field]+"(")
						add(file, as.Rhs[0].End(), as.Rhs[0].End(), ")")
						counts["content field write"]++
						return true
					}
					if bad := badContext(p, x); bad != "" {
						report(x, bad)
						return true
					}
					add(file, x.Sel.Pos(), x.Sel.End(), contentGetter[field]+"()")
					counts["content field read"]++
					return true
				}
				// PageCell field.
				if as, ok := p.(*ast.AssignStmt); ok && isLHS(as, x) {
					if as.Tok != token.ASSIGN || len(as.Lhs) != 1 {
						report(x, "assignment form "+as.Tok.String()+" or multiple targets")
						return true
					}
					add(file, x.Sel.Pos(), as.Rhs[0].Pos(), cellSetter[field]+"(")
					add(file, as.Rhs[0].End(), as.Rhs[0].End(), ")")
					counts["cell field write"]++
					return true
				}
				if bad := badContext(p, x); bad != "" {
					report(x, bad)
					return true
				}
				add(file, x.Sel.Pos(), x.Sel.End(), cellGetter[field]+"()")
				counts["cell field read"]++
			}
			return true
		})
	}

	for file, es := range edits {
		sort.Slice(es, func(i, j int) bool {
			if es[i].start != es[j].start {
				return es[i].start < es[j].start
			}
			if es[i].end != es[j].end {
				return es[i].end < es[j].end
			}
			// Two insertions at one place: a literal assigned to a field,
			// `x.f = T{...}`. The literal's `.pack()` goes inside the
			// setter's closing parenthesis.
			return es[i].text == ".pack()" && es[j].text != ".pack()"
		})
		for i := 1; i < len(es); i++ {
			if es[i].start < es[i-1].end {
				fmt.Fprintf(os.Stderr, "%s: overlapping edits at offset %d (%q, %q)\n", file, es[i].start, es[i-1].text, es[i].text)
				os.Exit(1)
			}
		}
		if !*write {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		var out []byte
		last := 0
		for _, e := range es {
			out = append(out, src[last:e.start]...)
			out = append(out, e.text...)
			last = e.end
		}
		out = append(out, src[last:]...)
		if err := os.WriteFile(file, out, 0o644); err != nil {
			panic(err)
		}
	}
	var ks []string
	for k := range counts {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		fmt.Printf("%-22s %d\n", k, counts[k])
	}
	fmt.Printf("files: %d\n", len(edits))
	sort.Strings(manual)
	fmt.Printf("manual: %d\n", len(manual))
	for _, m := range manual {
		fmt.Println("  " + m)
	}
}

func selIdent(s *ast.SelectorExpr) *ast.Ident {
	if s == nil {
		return nil
	}
	return s.Sel
}

func isLHS(as *ast.AssignStmt, x ast.Expr) bool {
	for _, l := range as.Lhs {
		if l == x {
			return true
		}
	}
	return false
}

// badContext reports a use that is neither a plain read nor a plain
// assignment: the address of a field, ++/--, or a nested field write.
func badContext(p ast.Node, x *ast.SelectorExpr) string {
	switch q := p.(type) {
	case *ast.UnaryExpr:
		if q.Op == token.AND {
			return "address of a field"
		}
	case *ast.IncDecStmt:
		return "++ or -- on a field"
	case *ast.AssignStmt:
		// Not an LHS (checked by the caller): a read on the right side.
	}
	return ""
}
