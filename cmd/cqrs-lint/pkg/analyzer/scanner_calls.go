package analyzer

import (
	"go/ast"
	"slices"
)

// scanCallExpr inspects call expressions for event.New/NewEvent,
// RegisterTyped, catalog.Event, and similar CQRS API calls.
func scanCallExpr(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	funcName := sel.Sel.Name
	pkgName := SelectorPackage(sel)

	switch {
	case funcName == "New" && pkgName == "event":
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = gf.Path
			}
		}

		capturePayloadType(ctx, call)

	case funcName == "NewEvent" && pkgName == "event":
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = gf.Path
			}
		}

		capturePayloadType(ctx, call)

	case funcName == "RegisterTyped":
		ctx.Registry.CommandTypesRegistered[handlerTypeFromCall(call)] = true

	case funcName == "Event" && pkgName == "catalog":
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesInCatalog[eventTypeStr] = true
			}
		}

	case funcName == "NewProjection":
		scanProjectionRegistration(ctx, gf, call)

	case funcName == "Subscribe":
		scanProjectionSubscription(ctx, gf, call, pkgName)
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

func capturePayloadType(ctx *AnalysisContext, call *ast.CallExpr) {
	for _, arg := range call.Args {
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
	}

	ctx.Registry.Projections = append(ctx.Registry.Projections, info)
}

func scanProjectionSubscription(
	ctx *AnalysisContext,
	gf *GoFile,
	call *ast.CallExpr,
	pkgName string,
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
