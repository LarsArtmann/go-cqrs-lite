package analyzer

import "go/ast"

// scanEmitHelperFunc registers constructor helpers whose event.New/event.NewEvent
// call passes one of the helper's own parameters as the event type:
//
//	func newRoomEvent(t event.Type, actor id.ActorID) (event.Event, error) {
//		return event.New(t, ...)
//	}
//
// The parameter identifier cannot be resolved at the emission site — the
// constant arrives at CALL sites of the helper. ResolveHelperEmitCalls walks
// those call sites after all files are scanned and feeds the resolved type
// strings into EventTypesEmitted exactly as a direct emission would
// (nsfw-classifier feedback, 2026-10-03: helper indirection left soft-delete
// detection reporting false).
func scanEmitHelperFunc(ctx *AnalysisContext, gf *GoFile, fn *ast.FuncDecl) {
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
		if !ok {
			return true
		}

		if sel.Sel.Name != "New" && sel.Sel.Name != "NewEvent" {
			return true
		}

		if !IsQualifierFor(gf, sel, "go-cqrs-lite/event") || len(call.Args) == 0 {
			return true
		}

		// Unwrap value-preserving conversions: string(t) and event.Type(t)
		// both forward the parameter's identity — the wrapped form must
		// register the helper too (M37/F233, mirrors the catalog twin).
		argName := ExprIdentName(unwrapIdConversions(gf, call.Args[0]))
		if argName == "" {
			return true
		}

		for i, p := range params {
			if p == argName {
				ctx.Registry.emitHelperParams[fn.Name.Name] = i
				return false
			}
		}

		return true
	})
}

// ResolveHelperEmitCalls walks call sites of every registered emit helper and
// records the helper-argument constants as pending emission refs. Must run
// AFTER all files are scanned (helpers may be declared in any file) and
// BEFORE ResolveEmittedEventTypeConsts (which drains the pending refs).
func ResolveHelperEmitCalls(ctx *AnalysisContext) {
	if len(ctx.Registry.emitHelperParams) == 0 {
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

			idx, ok := callTargetIndex(call, ctx.Registry.emitHelperParams)
			if !ok || idx >= len(call.Args) {
				return true
			}

			// Unwrap value-preserving converters at the call site —
			// string(const) and event.Type(...) wrappers forward the same
			// constant — and replay them instead of dropping (M37/F233,
			// 20-24 item 15).
			argName := ExprIdentName(unwrapIdConversions(gf, call.Args[idx]))
			if argName == "" {
				return true
			}

			pos := ctx.Fset.Position(call.Pos())
			ctx.Registry.pendingEmittedEventTypeRefs = append(
				ctx.Registry.pendingEmittedEventTypeRefs,
				pendingEventTypeRef{constName: argName, file: gf.Path, line: pos.Line},
			)

			return true
		})
	}
}

// unwrapEventTypeDef unwraps a single-argument event.Type(x) conversion —
// the package's own type conversion over a string-kind constant, which
// preserves the constant's identity for emission-type resolution. Unlike
// unwrapStringConv this needs the file's import table (IsQualifierFor),
// because a selector call could otherwise be any function named Type.
func unwrapEventTypeDef(gf *GoFile, expr ast.Expr) ast.Expr {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return expr
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Type" {
		return expr
	}

	if !IsQualifierFor(gf, sel, "go-cqrs-lite/event") {
		return expr
	}

	return call.Args[0]
}

// unwrapIdConversions unwraps any nesting of value-preserving conversions
// (string(x), event.Type(x)) down to the innermost expression, so a
// converter-wrapped constant still resolves to its identifier.
func unwrapIdConversions(gf *GoFile, expr ast.Expr) ast.Expr {
	for range 8 {
		unwrapped := unwrapEventTypeDef(gf, unwrapStringConv(expr))
		if unwrapped == expr {
			return expr
		}

		expr = unwrapped
	}

	return expr
}

// callTargetIndex resolves the called function's bare name for a call whose
// Fun is a plain identifier, a selector (pkg.helper / obj.Method), a generic
// instantiation (helper[T](…)), or a parenthesized form of those, and looks it
// up in the helper table.
func callTargetIndex(call *ast.CallExpr, helpers map[string]int) (int, bool) {
	name := calleeBaseName(call.Fun)
	if name == "" {
		return 0, false
	}

	idx, ok := helpers[name]

	return idx, ok
}

// calleeBaseName returns the bare identifier a call target bottoms out at
// (helper, pkg.helper, helper[T]), or "" for more complex callees.
func calleeBaseName(expr ast.Expr) string {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if ok {
			expr = paren.X

			continue
		}

		break
	}

	switch fun := expr.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		return fun.Sel.Name
	case *ast.IndexExpr:
		return calleeBaseName(fun.X)
	case *ast.IndexListExpr:
		return calleeBaseName(fun.X)
	default:
		return ""
	}
}

// paramIdents returns the names of a function's positional parameters
// (variadic parameter included, at its declared index).
func paramIdents(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}

	var names []string

	for _, field := range fields.List {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}

		if len(field.Names) == 0 {
			// Unnamed parameter still occupies a position.
			names = append(names, "")
		}
	}

	return names
}

// collectSwitchCaseValues renders the raw values of a switch's case clauses:
// string literals verbatim, identifier/selector arguments by their last
// identifier segment. Const identifiers are resolved to string values later
// by ResolveFoldTombstoneCases.
func collectSwitchCaseValues(sw *ast.SwitchStmt) []string {
	var values []string

	for _, stmt := range sw.Body.List {
		cc, ok := stmt.(*ast.CaseClause)
		if !ok {
			continue
		}

		for _, expr := range cc.List {
			if lit := StringLit(expr); lit != "" {
				values = append(values, lit)
				continue
			}

			if name := ExprIdentName(expr); name != "" {
				values = append(values, name)
			}
		}
	}

	return values
}

// ResolveFoldTombstoneCases marks folds whose switch handles a tombstone-like
// event type. Case values recorded as identifiers are resolved against
// TypeConstValues (with alias expansion) first. Must run AFTER
// ResolveEmittedEventTypeConsts so alias chains are expanded.
func ResolveFoldTombstoneCases(reg *CQRSRegistry) {
	for i := range reg.Folds {
		for _, raw := range reg.Folds[i].SwitchCaseValues {
			value := raw
			if resolved, ok := reg.TypeConstValues[raw]; ok && resolved != "" {
				value = resolved
			}

			if IsTombstoneLikeEventType(value) {
				reg.Folds[i].HandlesTombstoneEvent = true
				break
			}
		}
	}
}
