package resilience

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// accessorReturnsCQRSBus reports whether a matched Bus() call's static
// RESULT type is a go-cqrs-lite event bus — the engine journal tail.
// system.System.Bus() returns event.Bus, and consumer passthroughs (CV's
// LibraryEngine.Bus()) return the same interface, so the RESULT type is
// the discriminating signal: a user rabbit.Bus() returning its own
// downstream transport type never grants the journal-tail skip, whatever
// the accessor is named. Returns true when type info is unavailable or
// unresolved (conservative — keeps the legacy name-only match, which the
// syntax-only fixtures rely on).
func accessorReturnsCQRSBus(pkg *packages.Package, call *ast.CallExpr) bool {
	if pkg == nil || pkg.TypesInfo == nil {
		return true
	}

	tv, ok := pkg.TypesInfo.Types[call]
	if !ok || tv.Type == nil {
		return true
	}

	return strings.Contains(tv.Type.String(), "cqrs-lite/event/")
}

// busVariableFromAccessor reports whether varName is assigned from a Bus()
// accessor call — `bus := engine.Bus()` or `bus := s.library.Bus()`. The
// engine's Bus() returns the in-process journal/notification bus: the engine
// publishes there itself after appends, consumers tail it via SubscribeAll
// fan-out with drop counting, and durability comes from journal replay — not
// from transport retries. Publishing onto it feeds that same fan-out
// (best-effort, drop-counted) rather than calling downstream services, so
// retry/circuit-breaker middleware advice is category confusion whether the
// variable subscribes or feeds (CV feedback, 2026-10-03 — the journal-tail
// vs dispatch-bus distinction).
func busVariableFromAccessor(ctx *analyzer.AnalysisContext, v busVariable) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			matches := false

			for _, lhs := range assign.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && identIsVariable(gf.Pkg, ident, v) {
					matches = true

					break
				}
			}

			if !matches {
				return true
			}

			for _, rhs := range assign.Rhs {
				call, ok := rhs.(*ast.CallExpr)
				if !ok {
					continue
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Bus" {
					continue
				}

				if !accessorReturnsCQRSBus(gf.Pkg, call) {
					continue
				}

				found = true

				return false
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// busIsJournalTail combines the two journal-tail signals: a variable whose
// whole bus surface is subscriptions (read-only consumer) and one assigned
// from an engine Bus() accessor (subscribing and feeding the fan-out are
// both journal-tail acts). Neither dispatches to downstream services, so the
// B029/B030 resilience advice does not apply; a bus constructed by the
// project itself (newBus() and friends) never matches and stays fully armed.
func busIsJournalTail(ctx *analyzer.AnalysisContext, v busVariable) bool {
	return busIsReadOnlySubscriber(ctx, v) || busVariableFromAccessor(ctx, v)
}

// hasMiddlewareKeyword scans all non-test files for x.Use(...) or x.UsePublish(...)
// calls where any argument or the method name contains keyword (case-insensitive).
func hasMiddlewareKeyword(ctx *analyzer.AnalysisContext, v busVariable, keyword string) bool {
	keywordLower := strings.ToLower(keyword)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			exprStmt, ok := n.(*ast.ExprStmt)
			if !ok {
				return true
			}

			call, ok := exprStmt.X.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if !selectorOnVariable(gf.Pkg, sel, v) {
				return true
			}

			methodName := strings.ToLower(sel.Sel.Name)
			if methodName != "use" && methodName != "usepublish" &&
				methodName != "usemiddleware" && methodName != "addmiddleware" {
				return true
			}

			for _, arg := range call.Args {
				if callContainsKeyword(arg, keywordLower) {
					found = true
					return false
				}
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// callContainsKeyword checks whether an expression contains an identifier
// matching keyword (case-insensitive).
func callContainsKeyword(expr ast.Expr, keyword string) bool {
	found := false

	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}

		if ident, ok := n.(*ast.Ident); ok {
			if strings.Contains(strings.ToLower(ident.Name), keyword) {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

// findBusVariables returns the project's bus/dispatcher variables, collected
// from assignment statements in non-test files and keyed by IDENTITY: the
// variable's types.Object when type info resolves (two same-named `bus`
// variables in different scopes/files are distinct entries), the variable
// name otherwise (the legacy single-entry-per-name semantics).
// Only variables that also have CQRS bus method calls (Use, Publish, etc.)
// are included — this filters out lookalike names (errorBus.Notify).
func findBusVariables(ctx *analyzer.AnalysisContext) []busVariable {
	candidates := make([]busVariable, 0)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			for _, lhs := range assign.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || !isBusName(ident.Name) {
					continue
				}

				obj := types.Object(nil)
				if gf.Pkg != nil && gf.Pkg.TypesInfo != nil {
					if o := gf.Pkg.TypesInfo.ObjectOf(ident); o != nil {
						obj = o
					}
				}

				candidates = append(candidates, busVariable{
					name: ident.Name,
					pos:  ctx.Fset.Position(ident.Pos()),
					obj:  obj,
				})
			}

			return true
		})
	}

	// Dedupe by identity: same types.Object (typed loads) or same name
	// (syntax-only fallback — the legacy single-entry semantics).
	buses := make([]busVariable, 0, len(candidates))
	seenObj := make(map[types.Object]bool)
	seenName := make(map[string]bool)

	for _, c := range candidates {
		if !hasBusMethodCall(ctx, c) {
			continue
		}

		if c.obj != nil {
			if seenObj[c.obj] {
				continue
			}

			seenObj[c.obj] = true
			buses = append(buses, c)

			continue
		}

		if seenName[c.name] {
			continue
		}

		seenName[c.name] = true
		buses = append(buses, c)
	}

	return buses
}
