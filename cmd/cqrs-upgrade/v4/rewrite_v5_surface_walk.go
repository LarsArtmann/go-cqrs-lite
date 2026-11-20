package main

// rewrite_v5_surface_walk.go holds the AST walk half of the M14 surface
// pass (see rewrite_v5_surface.go for the rules and the tables): call-site
// classification, the parent map, and the collapsed-field hints.

import (
	"fmt"
	"go/ast"
	"go/token"
)

// callName renders a qualified call's "pkg.Fn" name, if the callee is a
// plain selector on an identifier.
func callName(ce *ast.CallExpr) (string, bool) {
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	return ident.Name + "." + sel.Sel.Name, true
}

// twoValueContext reports whether the call sits where a two-value
// expression is consumed (the unchanged v5 shape).
func twoValueContext(parents map[ast.Node]ast.Node, n ast.Node) bool {
	switch p := parents[n].(type) {
	case *ast.AssignStmt:
		if len(p.Rhs) == 1 && len(p.Lhs) == 2 {
			return true
		}
	case *ast.ValueSpec:
		if len(p.Values) == 1 && len(p.Names) == 2 {
			return true
		}
	}

	return false
}

// surfaceSites is the AST walk's harvest.
type surfaceSites struct {
	edits          []edit
	changes        []v5SurfaceChange
	manual         []string
	anyCtorRewrite bool
	sawStringCtor  bool
}

// collectSurfaceSites walks the file and classifies every R1/R2/R3 site.
func collectSurfaceSites(
	f *ast.File,
	fset *token.FileSet,
	parents map[ast.Node]ast.Node,
	path string,
) surfaceSites {
	var s surfaceSites

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			visitCall(node, fset, parents, path, &s)
		case *ast.CompositeLit:
			visitLiteral(node, fset, path, &s)
		}

		return true
	})

	return s
}

// visitCall applies the R1/R2 constructor rules and the R3 Split hint.
func visitCall(
	node *ast.CallExpr,
	fset *token.FileSet,
	parents map[ast.Node]ast.Node,
	path string,
	s *surfaceSites,
) {
	name, ok := callName(node)
	if !ok {
		return
	}

	sel, _ := node.Fun.(*ast.SelectorExpr)
	if sel == nil {
		return
	}

	if to, isCtor := singleValueStreamRefCtors[name]; isCtor && !twoValueContext(parents, node) {
		pos := fset.Position(sel.Pos())
		end := fset.Position(sel.End())
		s.edits = append(s.edits, edit{pos.Offset, end.Offset, to})
		s.changes = append(s.changes, v5SurfaceChange{
			Kind: "ctor-single-value",
			From: name + "(...)",
			To:   to + "(...)",
		})
		s.manual = append(s.manual, fmt.Sprintf(
			"%s:%d: %s(...) rewritten to %s — review the zero-ref arm",
			path, pos.Line, name, to))
		s.anyCtorRewrite = true

		return
	}

	if stringFormRecordCtors[name] && len(node.Args) == 2 {
		pos := fset.Position(sel.Pos())
		end := fset.Position(sel.End())
		s.edits = append(s.edits, edit{pos.Offset, end.Offset, v5HelperName})
		s.changes = append(s.changes, v5SurfaceChange{
			Kind: "record-string-ctor",
			From: name + "(...)",
			To:   v5HelperName + "(...)",
		})
		s.sawStringCtor = true

		return
	}

	// .Split() died with the string form (ADR-0166).
	if sel.Sel.Name == "Split" && len(node.Args) == 0 {
		pos := fset.Position(node.Pos())
		s.manual = append(s.manual, fmt.Sprintf(
			"%s:%d: .Split() call — the string form died (ADR-0166); read .Type/.ID from the struct ref",
			path, pos.Line))
	}
}

// visitLiteral applies the R3 collapsed-field hints.
func visitLiteral(node *ast.CompositeLit, fset *token.FileSet, path string, s *surfaceSites) {
	for _, elt := range node.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		pos := fset.Position(kv.Pos())
		switch key.Name {
		case "StreamID", "StreamType":
			s.manual = append(s.manual, fmt.Sprintf(
				"%s:%d: record literal key %s collapsed into Stream (M04) — use a StreamRef",
				path, pos.Line, key.Name))
		case "Encoding":
			s.manual = append(s.manual, fmt.Sprintf(
				"%s:%d: record literal key Encoding fused into Payload (M05) — use PayloadFrom",
				path, pos.Line))
		}
	}
}
