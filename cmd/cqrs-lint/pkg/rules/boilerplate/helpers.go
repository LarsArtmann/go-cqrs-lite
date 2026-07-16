package boilerplate

import "go/ast"

func selectorPackage(sel *ast.SelectorExpr) string {
	if ident, ok := sel.X.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}
