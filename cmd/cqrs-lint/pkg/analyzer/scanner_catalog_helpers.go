package analyzer

// Catalog registration WRAPPERS. Consumers rarely call catalog.Event /
// simple.Event directly for every message; the idiomatic shape is one
// generic helper that forwards a parameter as the id:
//
//	func registerEvent[P any](b *simple.Builder, agg string, t event.Type, s string) {
//		simple.Event[P](b, string(t), catalog.Sends, catalog.WithSummary(s))
//	}
//
//	registerEvent[contact.CreatedPayload](b, "contact", contact.EventCreated, "…")
//
// The id is unresolvable inside the helper body; the constant arrives at the
// call sites. This mirrors scanner_emit_helpers.go (the same indirection for
// event.New) because the blind spot is identical: without replay the catalog
// looks empty and E004 flags every emitted event (issue #79, CRM: 21 false
// positives on a catalog its completeness test pins as exhaustive).

import "go/ast"

// scanCatalogHelperFunc records a catalog registration wrapper in
// Registry.catalogHelperParams. Runs per FuncDecl during scanning, before any
// call site is known.
func scanCatalogHelperFunc(ctx *AnalysisContext, gf *GoFile, fn *ast.FuncDecl) {
	if fn.Name == nil || fn.Body == nil || fn.Recv != nil || fn.Type == nil {
		return
	}

	params := paramIdents(fn.Type.Params)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := SelectorFromExpr(call.Fun)
		if !ok || sel.Sel.Name != "Event" {
			return true
		}

		arg := catalogEventIDArg(gf, sel, call)
		argName := ExprIdentName(unwrapStringConv(arg))
		if argName == "" {
			return true
		}

		for i, p := range params {
			if p == argName {
				ctx.Registry.catalogHelperParams[fn.Name.Name] = i

				return false
			}
		}

		return true
	})
}

// catalogEventIDArg returns the message-id argument of a catalog.Event or
// simple.Event call, or nil when the call is neither. The simple builder keeps
// its id at index 1 because the *Builder comes first.
func catalogEventIDArg(gf *GoFile, sel *ast.SelectorExpr, call *ast.CallExpr) ast.Expr {
	if IsQualifierFor(gf, sel, "go-cqrs-lite/catalog/simple") {
		return catalogSimpleMessageIDArg(call)
	}

	if IsQualifierFor(gf, sel, "go-cqrs-lite/catalog") && len(call.Args) > 0 {
		return call.Args[0]
	}

	return nil
}

// ResolveCatalogHelperCalls replays the helper call sites' id arguments into
// EventTypesInCatalog (literals immediately, constants as pending refs). Must
// run AFTER all files are scanned and BEFORE
// ResolveEmittedEventTypeConsts, which drains the pending refs.
func ResolveCatalogHelperCalls(ctx *AnalysisContext) {
	if len(ctx.Registry.catalogHelperParams) == 0 {
		return
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			idx, ok := callTargetIndex(call, ctx.Registry.catalogHelperParams)
			if !ok || idx >= len(call.Args) {
				return true
			}

			recordCatalogEventType(ctx, gf, ctx.Fset.Position(call.Pos()), call.Args[idx])

			return true
		})
	}
}
