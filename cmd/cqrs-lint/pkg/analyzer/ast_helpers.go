package analyzer

import (
	"go/ast"
	"go/token"
	"strings"
)

// unwrapSelector unwraps generic instantiation wrappers (IndexExpr, IndexListExpr)
// to find the underlying SelectorExpr. This handles calls like:
//
//	decider.WithSnapshotStore[State](store)   → IndexExpr wrapping SelectorExpr
//	query.RegisterTyped[Q, R](d, t, handler)   → IndexListExpr wrapping SelectorExpr
//
// Returns nil if no SelectorExpr is found.
func unwrapSelector(expr ast.Expr) *ast.SelectorExpr {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return e
	case *ast.IndexExpr: // generic: X[T]
		return unwrapSelector(e.X)
	case *ast.IndexListExpr: // generic: X[T, U]
		return unwrapSelector(e.X)
	default:
		return nil
	}
}

// SelectorFromExpr extracts a SelectorExpr from an expression, unwrapping
// generic instantiation wrappers. Returns (nil, false) if not found.
// This is a drop-in replacement for call.Fun.(*ast.SelectorExpr) that also
// handles generic function calls like pkg.Func[T](args).
func SelectorFromExpr(expr ast.Expr) (*ast.SelectorExpr, bool) {
	sel := unwrapSelector(expr)
	if sel == nil {
		return nil, false
	}

	return sel, true
}

func recvTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}

	return BaseTypeName(fn.Recv.List[0].Type)
}

// BaseTypeName extracts the underlying type name from an AST expression,
// unwrapping pointer and generic instantiation wrappers.
func BaseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return BaseTypeName(t.X)
	case *ast.IndexExpr:
		return BaseTypeName(t.X)
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

// selectorNameAndPkg extracts the function/method name and package qualifier
// from a call expression. Returns ok=false when the call target is not a
// selector (e.g. a bare ident or builtin).
func selectorNameAndPkg(call *ast.CallExpr) (funcName, pkgName string, ok bool) {
	sel, alright := SelectorFromExpr(call.Fun)
	if !alright {
		return "", "", false
	}

	return sel.Sel.Name, SelectorPackage(sel), true
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
		return "(" + BaseTypeName(fn.Recv.List[0].Type) + ")." + fn.Name.Name
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

// ExtractJSONTag extracts the JSON field name from a struct tag string.
// Returns empty string if no json tag is present.
func ExtractJSONTag(tag string) string {
	idx := strings.Index(tag, `json:"`)
	if idx < 0 {
		return ""
	}

	start := idx + len(`json:"`)

	end := strings.Index(tag[start:], `"`)
	if end < 0 {
		return ""
	}

	value := tag[start : start+end]
	if comma := strings.Index(value, ","); comma >= 0 {
		value = value[:comma]
	}

	return value
}
