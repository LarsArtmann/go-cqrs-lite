package analyzer

import (
	"go/ast"
)

// newEventsTypeArgIndex is the position of the []event.Type argument in
// event.NewEvents(streamID, streamType, version, eventTypes, payloads, opts...).
const newEventsTypeArgIndex = 3

// scanNewEventsCall records the event types of an event.NewEvents call. The
// batch constructor carries its types as a slice literal, so only a composite
// literal argument can be read; a slice held in a variable (or built by a
// call) stays unrecorded.
func scanNewEventsCall(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	if len(call.Args) <= newEventsTypeArgIndex {
		return
	}

	lit, ok := call.Args[newEventsTypeArgIndex].(*ast.CompositeLit)
	if !ok {
		return
	}

	for _, elt := range lit.Elts {
		// Elements are commonly written as event.Type(Const) conversions,
		// which preserve the constant's identity.
		recordEmittedEventExpr(ctx, gf, unwrapIdConversions(gf, elt))
	}
}

// recordEmittedEventExpr records one event-type expression as emitted: a
// string literal directly, a constant reference for the post-scan const
// resolution pass.
func recordEmittedEventExpr(ctx *AnalysisContext, gf *GoFile, expr ast.Expr) {
	if expr == nil {
		return
	}

	pos := ctx.Fset.Position(expr.Pos())

	if eventTypeStr := StringLit(expr); eventTypeStr != "" {
		ctx.Registry.EventTypesEmitted[eventTypeStr] = EventEmission{
			File: gf.Path,
			Line: pos.Line,
		}

		return
	}

	if name := ExprIdentName(expr); name != "" {
		ctx.Registry.pendingEmittedEventTypeRefs = append(
			ctx.Registry.pendingEmittedEventTypeRefs,
			pendingEventTypeRef{constName: name, file: gf.Path, line: pos.Line},
		)
	}
}
