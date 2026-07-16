package analyzer

import (
	"go/ast"
)

// scanCallExpr inspects call expressions for event.New/NewEvent,
// RegisterTyped, catalog.Event, and similar CQRS API calls.
func scanCallExpr(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	funcName := sel.Sel.Name
	pkgName := selectorPackage(sel)

	switch {
	case funcName == "New" && pkgName == "event":
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = gf.Path
			}
		}

	case funcName == "NewEvent" && pkgName == "event":
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = gf.Path
			}
		}

	case funcName == "RegisterTyped":
		ctx.Registry.CommandTypesRegistered[handlerTypeFromCall(call)] = true

	case funcName == "Event" && pkgName == "catalog":
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesInCatalog[eventTypeStr] = true
			}
		}
	}
}

func handlerTypeFromCall(call *ast.CallExpr) string {
	for _, arg := range call.Args {
		switch a := arg.(type) {
		case *ast.CompositeLit:
			if id, ok := a.Type.(*ast.Ident); ok {
				return id.Name
			}
		case *ast.CallExpr:
			return ExprString(a)
		}
	}

	return ""
}
