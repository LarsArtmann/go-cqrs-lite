package analyzer

import (
	"go/ast"
	"go/token"
	"strings"
)

func recvTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}

	return baseTypeName(fn.Recv.List[0].Type)
}

func baseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return baseTypeName(t.X)
	case *ast.IndexExpr:
		return baseTypeName(t.X)
	}

	return ""
}

func stringLit(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}

	return strings.Trim(lit.Value, `"`)
}

// Helper: extract the package/qualifier from a SelectorExpr.
// SelectorPackage extracts the package name from a SelectorExpr.
func SelectorPackage(sel *ast.SelectorExpr) string {
	switch x := sel.X.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return SelectorPackage(x)
	}

	return ""
}

// Helper: get function name including receiver.
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		return "(" + baseTypeName(fn.Recv.List[0].Type) + ")." + fn.Name.Name
	}

	return fn.Name.Name
}

// ExprString returns a best-effort string representation of an AST expression.
// This is used for heuristic type matching.
func ExprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case nil:
		return ""
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + ExprString(e.X)
	case *ast.SelectorExpr:
		return ExprString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + ExprString(e.Elt)
	case *ast.MapType:
		return "map[" + ExprString(e.Key) + "]" + ExprString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.StructType:
		return "struct{...}"
	case *ast.BasicLit:
		return e.Value
	case *ast.CallExpr:
		return ExprString(e.Fun) + "(...)"
	case *ast.IndexExpr:
		return ExprString(e.X) + "[" + ExprString(e.Index) + "]"
	case *ast.BinaryExpr:
		return ExprString(e.X) + " " + e.Op.String() + " " + ExprString(e.Y)
	case *ast.UnaryExpr:
		return e.Op.String() + ExprString(e.X)
	case *ast.ParenExpr:
		return "(" + ExprString(e.X) + ")"
	case *ast.CompositeLit:
		return ExprString(e.Type) + "{...}"
	case *ast.Ellipsis:
		return "..." + ExprString(e.Elt)
	default:
		return ""
	}
}

// nodeString returns a rough string for an AST node (for pattern matching).
func nodeString(n ast.Node) string {
	if n == nil {
		return ""
	}

	if expr, ok := n.(ast.Expr); ok {
		return ExprString(expr)
	}

	return ""
}
