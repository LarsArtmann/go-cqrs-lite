package analyzer

import (
	"go/ast"
)

// IsInsideUpcasterClosure checks if the given call expression is inside a
// function literal passed to schema.NewUpcaster. In that context,
// event.NewEvent and json.Unmarshal on event payloads are the correct APIs
// (upcasters transform raw bytes and reconstruct events).
//
// This is used to suppress A014 and C005 findings in upcaster context.
func IsInsideUpcasterClosure(gf *GoFile, call *ast.CallExpr) bool {
	var inside bool

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		if inside {
			return false
		}

		upcasterCall, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := SelectorFromExpr(upcasterCall.Fun)
		if !ok {
			return true
		}

		if sel.Sel.Name != "NewUpcaster" {
			return true
		}

		pkgName := SelectorPackage(sel)
		if pkgName != "schema" {
			return true
		}

		for _, arg := range upcasterCall.Args {
			fnLit, ok := arg.(*ast.FuncLit)
			if !ok || fnLit.Body == nil {
				continue
			}

			if fnLit.Body.Pos() <= call.Pos() && call.Pos() <= fnLit.Body.End() {
				inside = true
				return false
			}
		}

		return true
	})

	return inside
}
