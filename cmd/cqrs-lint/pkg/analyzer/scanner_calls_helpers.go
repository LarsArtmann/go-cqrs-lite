package analyzer

import (
	"go/ast"
)

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

	name := ExprIdentName(call.Args[argIndex])
	if name != "" {
		ctx.Registry.registeredTypeConsts = append(ctx.Registry.registeredTypeConsts, name)
	}
}

// ExprIdentName extracts the bare identifier name from an AST expression,
// unwrapping package qualifiers (e.g. models.UserState → UserState,
// projection.GetVisitQueryType → GetVisitQueryType). Returns "" for
// expressions that are not simple identifier references.
func ExprIdentName(expr ast.Expr) string {
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

// eventMetadataTypeExpr returns the value expression of the "Type" field when
// call's single argument is a composite literal carrying one (the
// event-metadata registration shape, e.g.
// Register(EventMetadata{Type: string(identitymodel.EventX), ...})), and nil
// otherwise.
func eventMetadataTypeExpr(call *ast.CallExpr) ast.Expr {
	if len(call.Args) != 1 {
		return nil
	}

	lit, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return nil
	}

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Type" {
			return kv.Value
		}
	}

	return nil
}

// handlerTypeFromCall extracts the handler type name from a RegisterTyped or
// RegisterQuery call. It handles two registration patterns:
//
//  1. Composite literal:     RegisterTyped(d, MyCommand{})      → "MyCommand"
//  2. Closure handler:       RegisterTyped(d, type, func(ctx, c *MyCommand) error {...})
//
// Constructor-call handlers (NewMyCommand(bus)) are NOT type names — they are
// recorded separately via constructorHandlerText (T20-4).
func handlerTypeFromCall(call *ast.CallExpr) string {
	for _, arg := range call.Args {
		switch a := arg.(type) {
		case *ast.CompositeLit:
			if id, ok := a.Type.(*ast.Ident); ok {
				return id.Name
			}
		case *ast.FuncLit:
			return handlerTypeFromClosure(a)
		}
	}

	return ""
}

// constructorHandlerText returns the call text of the first constructor-call
// handler argument (e.g. "NewMyCommand(bus)"), or "" when no handler arg is a
// call expression. T20-4: these records live in Registry.ConstructorHandlers,
// never in CommandTypesRegistered (keys there must be struct type names).
func constructorHandlerText(call *ast.CallExpr) string {
	for _, arg := range call.Args {
		if _, ok := arg.(*ast.CallExpr); ok {
			return ExprString(arg)
		}
	}

	return ""
}

// handlerTypeFromClosure extracts the handler type from a function literal's
// parameter list. It finds the first parameter that is a pointer to a named
// type (e.g., *MyCommand) or a named type (e.g., MyQuery), skipping
// context.Context parameters. Handles bare identifiers (*MyCommand),
// package-qualified types (*pkg.MyCommand → "MyCommand"), and unqualified
// cross-package params (cmd deal.LoseCmd → "LoseCmd").
func handlerTypeFromClosure(fn *ast.FuncLit) string {
	if fn.Type == nil || fn.Type.Params == nil {
		return ""
	}

	for _, param := range fn.Type.Params.List {
		switch t := param.Type.(type) {
		case *ast.StarExpr:
			if name := lastIdentSegment(t.X); name != "" {
				return name
			}
		case *ast.Ident:
			return t.Name
		case *ast.SelectorExpr:
			// context.Context is a SelectorExpr, not a handler type; skip it
			// and keep looking at the remaining parameters.
			if isContextSelector(t) {
				continue
			}

			return t.Sel.Name
		}
	}

	return ""
}

// isContextSelector reports whether expr is a reference to context.Context.
func isContextSelector(sel *ast.SelectorExpr) bool {
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "context" && sel.Sel.Name == "Context"
}
