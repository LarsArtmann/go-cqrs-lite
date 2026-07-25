package analyzer

import "go/ast"

// extractHandlerFuncLit unwraps function literal arguments (closure handlers).
func extractHandlerFuncLit(expr ast.Expr) *ast.FuncLit {
	if fn, ok := expr.(*ast.FuncLit); ok {
		return fn
	}

	return nil
}

// recordTypeConstArg records a type-constant argument for post-pass resolution.
// The argument may be a bare identifier (GetVisitQueryType) or a selector
// expression (projection.GetVisitQueryType); only the final identifier is
// recorded, since const declarations are package-scoped and cross-package
// matching by bare name is sufficient for suppression (a collision would only
// cause a false negative — a suppressed finding — which is acceptable). No-op
// for composite literals, closures, and constructor calls — those are handled
// directly by handlerTypeFromCall.
func recordTypeConstArg(ctx *AnalysisContext, call *ast.CallExpr, argIndex int) {
	if argIndex < 0 || argIndex >= len(call.Args) {
		return
	}

	name := constNameFromExpr(call.Args[argIndex])
	if name != "" {
		ctx.Registry.registeredTypeConsts = append(ctx.Registry.registeredTypeConsts, name)
	}
}

// constNameFromExpr extracts a bare constant identifier name from an argument
// expression. Returns "" for expressions that are not simple constant references
// (composite literals, closures, constructor calls, string literals).
func constNameFromExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		// Bare constant: GetVisitQueryType
		return e.Name
	case *ast.SelectorExpr:
		// Package-qualified constant: projection.GetVisitQueryType
		return e.Sel.Name
	}

	return ""
}

// methodNameFromHandlerArg extracts the method name from a RegisterTyped/
// RegisterQuery handler argument when it is a method value (e.g.
// `h.handleCreateGame` → "handleCreateGame"). Returns "" for non-method-value
// handlers (closures, constructor calls). The handler is conventionally the
// LAST argument of RegisterTyped.
func methodNameFromHandlerArg(call *ast.CallExpr) string {
	if len(call.Args) < 3 {
		return ""
	}

	handler := call.Args[len(call.Args)-1]
	sel, ok := handler.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	// Only record if the qualifier is an *ast.Ident (a variable like `h`, not
	// a package qualifier or a call chain like `NewHandler(rm).Handle`).
	if _, ok := sel.X.(*ast.Ident); !ok {
		return ""
	}

	return sel.Sel.Name
}

// foldNameFromStrictApplyArg extracts the fold function name from the first
// argument of a decider.StrictApply call. The fold arg may be a bare function
// identifier (Fold) or a method value (aggregate.Fold, h.Fold). Returns the
// last identifier segment so it matches FoldInfo.FuncName's last segment
// regardless of receiver/package qualification.
func foldNameFromStrictApplyArg(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}

	return lastIdentSegment(call.Args[0])
}

// lastIdentSegment returns the trailing identifier of an expression: the Sel
// name for a SelectorExpr (a.b.Fold → Fold), the Name for an Ident, or "".
func lastIdentSegment(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.StarExpr:
		return lastIdentSegment(e.X)
	}

	return ""
}

// hasAsyncInBody checks if a function body contains a go statement (goroutine launch).
func hasAsyncInBody(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}

	hasGo := false

	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.GoStmt); ok {
			hasGo = true
			return false
		}
		return true
	})

	return hasGo
}
