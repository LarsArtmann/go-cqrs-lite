package analyzer

import (
	"go/ast"
	"go/token"
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

// recordSchemaDeclaredEvent records one schema-declaration event type
// (schema.Event/EventOf or the Schemas-builder .Event method): the first
// argument is the event type as a string literal, or a constant reference
// that resolves in the post-pass (ResolveEmittedEventTypeConsts).
func recordSchemaDeclaredEvent(
	ctx *AnalysisContext,
	gf *GoFile,
	call *ast.CallExpr,
	pos token.Position,
) {
	if len(call.Args) == 0 {
		return
	}

	if eventTypeStr := StringLit(call.Args[0]); eventTypeStr != "" {
		ctx.Registry.EventTypesInSchemaDecl[eventTypeStr] = EventEmission{
			File: gf.Path,
			Line: pos.Line,
		}
	} else if name := ExprIdentName(call.Args[0]); name != "" {
		ctx.Registry.pendingSchemaEventTypeRefs = append(
			ctx.Registry.pendingSchemaEventTypeRefs,
			pendingEventTypeRef{constName: name, file: gf.Path, line: pos.Line},
		)
	}
}

// isSchemaBuilderEventCall reports whether an .Event method call targets the
// system.Schemas() fluent builder: the receiver is a system.Schemas() call
// (directly or through a chain of builder .Event calls) or a local variable
// initialized from one (tracked per file by trackSchemaBuilderAssignments).
// A false positive only over-declares — suppressing an E021 finding — which
// is the safe direction; a false negative keeps the finding alive.
func isSchemaBuilderEventCall(ctx *AnalysisContext, gf *GoFile, sel *ast.SelectorExpr) bool {
	switch recv := sel.X.(type) {
	case *ast.Ident:
		return ctx.Registry.schemaBuilderIdents[gf.Path][recv.Name]
	case *ast.CallExpr:
		return isSchemaBuilderChainCall(ctx, gf, recv)
	}

	return false
}

// isSchemaBuilderChainCall walks a receiver call chain that must end in a
// system-qualified Schemas() call or a tracked builder variable:
// system.Schemas().Event[a](...).Event[b](...).
func isSchemaBuilderChainCall(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) bool {
	chainSel, ok := SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}

	switch chainSel.Sel.Name {
	case "Schemas":
		return IsQualifierFor(gf, chainSel, "go-cqrs-lite/system")
	case "Event":
		switch recv := chainSel.X.(type) {
		case *ast.CallExpr:
			return isSchemaBuilderChainCall(ctx, gf, recv)
		case *ast.Ident:
			return ctx.Registry.schemaBuilderIdents[gf.Path][recv.Name]
		}
	}

	return false
}

// trackSchemaBuilderAssignments records `schemas := system.Schemas()` (and
// the rare re-assignment form) so later builder-method calls through the
// variable resolve to the schema declaration tier. Same-file only: the fluent
// builder is declared and consumed in one place; a cross-file builder var is
// a missed declaration (safe direction).
func trackSchemaBuilderAssignments(ctx *AnalysisContext, gf *GoFile, stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}

		sel, ok := SelectorFromExpr(call.Fun)
		if !ok || sel.Sel.Name != "Schemas" || !IsQualifierFor(gf, sel, "go-cqrs-lite/system") {
			continue
		}

		if i >= len(stmt.Lhs) {
			continue
		}

		if ident, ok := stmt.Lhs[i].(*ast.Ident); ok {
			if ctx.Registry.schemaBuilderIdents == nil {
				ctx.Registry.schemaBuilderIdents = make(map[string]map[string]bool)
			}

			if ctx.Registry.schemaBuilderIdents[gf.Path] == nil {
				ctx.Registry.schemaBuilderIdents[gf.Path] = make(map[string]bool)
			}

			ctx.Registry.schemaBuilderIdents[gf.Path][ident.Name] = true
		}
	}
}
