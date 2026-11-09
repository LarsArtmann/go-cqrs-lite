package analyzer

// catalog/simple builder registrations. The builder API puts the *Builder at
// argument 0, so the message id lives at index 1; these helpers keep that
// shape out of the main call scanner.

import (
	"go/ast"
	"go/token"
)

// catalogSimpleMessageIDArg returns the message-id expression of a
// catalog/simple builder call. Event[T](b *Builder, id string, ...) puts the
// builder first, so the id lives at index 1.
func catalogSimpleMessageIDArg(call *ast.CallExpr) ast.Expr {
	if len(call.Args) < 2 {
		return nil
	}

	return call.Args[1]
}

// recordCatalogEventType records a catalog message declaration: a string
// literal (directly or through a string(...) conversion) is stored
// immediately; a constant reference is resolved after scanning by
// ResolveEmittedEventTypeConsts.
func recordCatalogEventType(ctx *AnalysisContext, gf *GoFile, pos token.Position, expr ast.Expr) {
	if expr == nil {
		return
	}

	expr = unwrapStringConv(expr)

	if eventTypeStr := StringLit(expr); eventTypeStr != "" {
		ctx.Registry.EventTypesInCatalog[eventTypeStr] = true

		return
	}

	if name := ExprIdentName(expr); name != "" {
		ctx.Registry.pendingCatalogEventTypeRefs = append(
			ctx.Registry.pendingCatalogEventTypeRefs,
			pendingEventTypeRef{constName: name, file: gf.Path, line: pos.Line},
		)
	}
}
