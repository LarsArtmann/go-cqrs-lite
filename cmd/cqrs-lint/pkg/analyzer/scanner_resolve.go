package analyzer

import (
	"go/ast"
	"strings"
)

// looksLikeStructName reports whether s is a plausible exported Go struct name
// (starts with an uppercase letter and contains only identifier characters).
// Used to decide whether a type-constant value should suppress E005/E007 even
// when the struct isn't in the Commands slice (queries aren't pre-registered).
func looksLikeStructName(s string) bool {
	if s == "" {
		return false
	}

	first := s[0]

	if first < 'A' || first > 'Z' {
		return false
	}

	for i := 1; i < len(s); i++ {
		c := s[i]
		isAlphaNum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '_'
		if !isAlphaNum {
			return false
		}
	}

	return true
}

// ResolveHandlerMethods resolves pending handler method names (recorded during
// scanCallExpr when RegisterTyped took a method value like `h.handleX`) to
// their target command/query struct types by finding the method's FuncDecl
// across all scanned files and extracting the type from its parameter list.
// Must run AFTER all files have been scanned, because the method declaration
// and the RegisterTyped call are typically in different files.
//
// A matching method is a FuncDecl whose Name matches a pending method name
// and whose parameter list contains a pointer-to-struct parameter whose name
// ends in "Command", "Cmd", or "Query". The first such parameter is the
// handler's command/query type.
func ResolveHandlerMethods(ctx *AnalysisContext) {
	if len(ctx.Registry.pendingHandlerMethods) == 0 {
		return
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest || gf.AST == nil {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}

			if !ctx.Registry.pendingHandlerMethods[fn.Name.Name] {
				continue
			}

			if t := handlerTypeFromFuncDecl(fn); t != "" {
				ctx.Registry.CommandTypesRegistered[t] = true
			}
		}
	}
}

// handlerTypeFromFuncDecl extracts the command/query struct name from a
// handler method's parameter list. Looks for the first parameter that is a
// pointer to a named type whose name ends in "Command", "Cmd", or "Query".
func handlerTypeFromFuncDecl(fn *ast.FuncDecl) string {
	if fn.Type == nil || fn.Type.Params == nil {
		return ""
	}

	for _, param := range fn.Type.Params.List {
		t, ok := param.Type.(*ast.StarExpr)
		if !ok {
			continue
		}

		name := lastIdentSegment(t.X)
		if name == "" {
			continue
		}

		if strings.HasSuffix(name, "Command") ||
			strings.HasSuffix(name, "Cmd") ||
			strings.HasSuffix(name, "Query") {
			return name
		}
	}

	return ""
}

// scanTypeAssertion detects type assertions on command/query structs inside
// handler closures and marks the asserted type as registered. This covers the
// opaque-closure pattern (SwettySwipper): handlers registered via
// RegisterAll(disp, map[Type]Handler{...}) use closures that take
// corecmd.Command (an interface) and type-assert internally:
//
//	func(ctx, cmd corecmd.Command) error {
//	    c, ok := cmd.(*CreateBattleCmd)
//	    ...
//	}
//
// The type assertion `cmd.(*CreateBattleCmd)` is the only place the command
// struct is visible. Scanning TypeAssertExpr nodes catches this pattern
// without needing to trace through the RegisterAll map indirection.
func scanTypeAssertion(ctx *AnalysisContext, expr *ast.TypeAssertExpr) {
	if expr.Type == nil {
		return
	}

	name := lastIdentSegment(expr.Type)
	if name == "" {
		return
	}

	if strings.HasSuffix(name, "Command") ||
		strings.HasSuffix(name, "Cmd") ||
		strings.HasSuffix(name, "Query") {
		ctx.Registry.CommandTypesRegistered[name] = true
	}
}
