package analyzer

import (
	"go/ast"
	"slices"
	"strings"
)

// scanCallExpr inspects call expressions for event.New/NewEvent,
// RegisterTyped, catalog.Event, and similar CQRS API calls.
func scanCallExpr(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	// Generic type-instantiation calls (requireCommandType[*MyCommand](cmd),
	// requireQueryType[*MyQuery](q)) have call.Fun as *ast.IndexExpr, not
	// *ast.SelectorExpr. Handle them before the SelectorExpr path. These calls
	// unambiguously identify handler bodies: a command/query struct that
	// appears as a generic type argument is being handled. This suppresses
	// E005/E007 for handlers registered via string-typed APIs
	// (dispatcher.Register with command.Type constants) whose handler→struct
	// link the analyzer cannot otherwise trace.
	scanGenericHandlerCall(ctx, call)

	sel, ok := SelectorFromExpr(call.Fun)
	if !ok {
		return
	}

	funcName := sel.Sel.Name
	pkgName := SelectorPackage(sel)
	pos := ctx.Fset.Position(call.Pos())

	switch {
	case funcName == "New" && pkgName == "event":
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = EventEmission{
					File: gf.Path,
					Line: pos.Line,
				}
			}
		}

		capturePayloadType(ctx, call)

	case funcName == "NewEvent" && pkgName == "event":
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = EventEmission{
					File: gf.Path,
					Line: pos.Line,
				}
			}
		}

		capturePayloadType(ctx, call)

	case funcName == "RegisterTyped" || funcName == "RegisterQuery":
		if handlerType := handlerTypeFromCall(call); handlerType != "" {
			ctx.Registry.CommandTypesRegistered[handlerType] = true
		} else {
			// Handler type could not be extracted from the call args directly.
			// Try two fallback strategies:
			//   1. If the handler arg is a method value (h.handleX), record the
			//      method name for a post-pass that finds the FuncDecl and
			//      extracts the param type. Covers SEC's typed handler methods.
			//   2. Record the type-constant arg for const-value resolution.
			//      Covers consumers whose const values are struct names.
			if methodName := methodNameFromHandlerArg(call); methodName != "" {
				ctx.Registry.pendingHandlerMethods[methodName] = true
			}

			recordTypeConstArg(ctx, call, 1)
		}

	case funcName == "Register" && pkgName != "event":
		// Plain dispatcher.Register(typeConst, handler) — the string-type-based
		// command registration API. The handler type is not visible in the call
		// (it lives inside the handler body), so record the type-constant arg
		// for post-pass resolution. pkgName != "event" excludes event bus
		// Subscribe-style calls that happen to be named "Register" — though
		// such collisions are rare, the guard is cheap. See browser-history
		// feedback (E005 false positives on dispatcher.Register).
		recordTypeConstArg(ctx, call, 0)

	case funcName == "Event" && pkgName == "catalog":
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesInCatalog[eventTypeStr] = true
			}
		}

	case funcName == "NewProjection":
		scanProjectionRegistration(ctx, gf, call)

	case funcName == "Subscribe":
		scanProjectionSubscription(ctx, gf, call)

	case funcName == "StrictApply" && pkgName == "decider":
		// decider.StrictApply(foldFunc, knownTypes) — record the fold function
		// name so B005 can suppress its "use decider.StrictApply" suggestion
		// when the suggestion is already implemented. See browser-history
		// feedback (B005 latent gap).
		if name := foldNameFromStrictApplyArg(call); name != "" {
			ctx.Registry.StrictApplyFolds[name] = true
		}
	}
}

// handlerTypeFromCall extracts the handler type name from a RegisterTyped or
// RegisterQuery call. It handles three registration patterns:
//
//  1. Composite literal:     RegisterTyped(d, MyCommand{})      → "MyCommand"
//  2. Constructor call:      RegisterTyped(d, NewMyCommand())    → "NewMyCommand(...)"
//  3. Closure handler:       RegisterTyped(d, type, func(ctx, c *MyCommand) error {...})
//
// For closures, the handler type is extracted from the first non-context
// parameter of the function literal's signature.
func handlerTypeFromCall(call *ast.CallExpr) string {
	for _, arg := range call.Args {
		switch a := arg.(type) {
		case *ast.CompositeLit:
			if id, ok := a.Type.(*ast.Ident); ok {
				return id.Name
			}
		case *ast.FuncLit:
			return handlerTypeFromClosure(a)
		case *ast.CallExpr:
			return ExprString(a)
		}
	}

	return ""
}

// handlerTypeFromClosure extracts the handler type from a function literal's
// parameter list. It finds the first parameter that is a pointer to a named
// type (e.g., *MyCommand) or a named type (e.g., MyQuery), skipping
// context.Context parameters. Handles both bare identifiers (*MyCommand) and
// package-qualified types (*pkg.MyCommand → "MyCommand").
func handlerTypeFromClosure(fn *ast.FuncLit) string {
	if fn.Type == nil || fn.Type.Params == nil {
		return ""
	}

	for _, param := range fn.Type.Params.List {
		switch t := param.Type.(type) {
		case *ast.StarExpr:
			return lastIdentSegment(t.X)
		case *ast.Ident:
			// Skip context.Context-like params that are just Idents
			// (context.Context itself is a SelectorExpr, so it won't match here)
			return t.Name
		}
	}

	return ""
}

// capturePayloadType records the struct type name used as the event payload.
// In event.New/NewEvent, the payload is always the 5th argument (index 4):
// event.New(type, aggregateID, aggregateType, version, payload, opts...).
func capturePayloadType(ctx *AnalysisContext, call *ast.CallExpr) {
	for i := 4; i < len(call.Args); i++ {
		arg := call.Args[i]

		switch a := arg.(type) {
		case *ast.CompositeLit:
			if id, ok := a.Type.(*ast.Ident); ok {
				ctx.Registry.EventPayloadTypes[id.Name] = true
				return
			}
		case *ast.Ident:
			ctx.Registry.EventPayloadTypes[a.Name] = true
			return
		}
	}
}

func scanProjectionRegistration(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	pos := ctx.Fset.Position(call.Pos())
	info := ProjectionInfo{
		Package: gf.Pkg.PkgPath,
		File:    gf.Path,
		Pos:     pos,
	}

	if len(call.Args) > 0 {
		info.Name = stringLit(call.Args[0])
	}

	for _, arg := range call.Args {
		if cl, ok := arg.(*ast.CompositeLit); ok {
			for _, elt := range cl.Elts {
				if eventTypeStr := stringLit(elt); eventTypeStr != "" {
					info.EventTypes = append(info.EventTypes, eventTypeStr)
				}
			}
		}

		// Check if any handler argument is a function literal that launches goroutines.
		if fn := extractHandlerFuncLit(arg); fn != nil {
			info.HasAsync = hasAsyncInBody(fn.Body)
		}
	}

	ctx.Registry.Projections = append(ctx.Registry.Projections, info)
}

func scanProjectionSubscription(
	ctx *AnalysisContext,
	gf *GoFile,
	call *ast.CallExpr,
) {
	eventTypeStr := ""
	if len(call.Args) > 0 {
		eventTypeStr = stringLit(call.Args[0])
	}

	if eventTypeStr == "" {
		return
	}

	found := false

	for i := range ctx.Registry.Projections {
		if slices.Contains(ctx.Registry.Projections[i].EventTypes, eventTypeStr) {
			found = true
		}
	}

	if !found {
		pos := ctx.Fset.Position(call.Pos())
		ctx.Registry.Projections = append(ctx.Registry.Projections, ProjectionInfo{
			Package:    gf.Pkg.PkgPath,
			File:       gf.Path,
			Pos:        pos,
			EventTypes: []string{eventTypeStr},
		})
	}
}

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

// scanGenericHandlerCall detects generic type-instantiation calls whose type
// argument is a command or query struct — e.g.
//
//	requireCommandType[*MyCommand](cmd)
//	requireQueryType[*MyQuery](q)
//
// These calls appear inside handler method bodies and unambiguously identify
// which struct the handler processes. This suppresses E005/E007 even when
// registration uses a string-typed API (dispatcher.Register with
// command.Type constants whose value is an event-style string like
// "browser_history.extract_history") because the handler→struct link is
// recovered from the generic type argument, not from the const value.
//
// The match is intentionally general — any generic instantiation X[*T](...)
// or X[T](...) where T ends in "Command" or "Query" is treated as evidence
// that T is handled. This avoids hard-coding consumer-specific helper names
// (requireCommandType, mustCommand, castQuery, …) while keeping the false
// positive risk negligible: a Command/Query-suffixed type used as a generic
// argument is overwhelmingly a handler type assertion.
func scanGenericHandlerCall(ctx *AnalysisContext, call *ast.CallExpr) {
	var typeArgs []ast.Expr

	switch fn := call.Fun.(type) {
	case *ast.IndexExpr:
		typeArgs = []ast.Expr{fn.Index}
	case *ast.IndexListExpr:
		typeArgs = fn.Indices
	default:
		return
	}

	for _, ta := range typeArgs {
		name := typeNameFromGenericArg(ta)
		if name == "" {
			continue
		}

		if strings.HasSuffix(name, "Command") || strings.HasSuffix(name, "Query") {
			ctx.Registry.CommandTypesRegistered[name] = true
		}
	}
}

// typeNameFromGenericArg extracts the struct name from a generic type argument
// expression, stripping a leading pointer indirection: *MyCommand → MyCommand.
func typeNameFromGenericArg(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}

	return ""
}
