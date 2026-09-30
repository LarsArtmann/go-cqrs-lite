package analyzer

import (
	"go/ast"
	"strings"
)

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
