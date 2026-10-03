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

		argName := ExprIdentName(call.Args[0])
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

			argName := ExprIdentName(call.Args[idx])
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

// callTargetIndex resolves the called function's bare name for a call whose
// Fun is a plain identifier or a selector (pkg.helper / obj.Method), and looks
// it up in the helper table.
func callTargetIndex(call *ast.CallExpr, helpers map[string]int) (int, bool) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		idx, ok := helpers[fun.Name]
		return idx, ok
	case *ast.SelectorExpr:
		idx, ok := helpers[fun.Sel.Name]
		return idx, ok
	default:
		return 0, false
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
