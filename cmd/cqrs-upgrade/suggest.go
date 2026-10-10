package main

import (
	"go/ast"
	"sort"

	cqrsanalyzer "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// ruleSuggestThenQuery tags hand-rolled eventually loops that the
// systemscenario harness replaces with a tested assertion.
const ruleSuggestThenQuery = "suggest:then-query"

// suggestionFindings walks the analyzed test files for hand-rolled
// eventually loops — a `for time.Now().Before(deadline)` (or After) loop
// whose body sleeps between probes — and emits one advisory per loop
// suggesting the harness's poll-based assertions instead:
// [WhenPhase.ThenQuery] awaits the projection with built-in timeout
// diagnostics, [WhenPhase.ThenQueryEventuallyFails] awaits a vanishing row.
// Suggestions are report-only: cqrs-upgrade never rewrites code, and
// --strict ignores them (they are migration hints, not v5 blockers).
func suggestionFindings(ctx *cqrsanalyzer.AnalysisContext) []findingJSON {
	var out []findingJSON

	for _, gf := range ctx.GoFiles {
		if !gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			loop, ok := n.(*ast.ForStmt)
			if !ok || !isEventuallyLoop(gf, loop) {
				return true
			}

			pos := ctx.Fset.Position(loop.Pos())
			out = append(out, findingJSON{
				Position: pos.String(),
				Rule:     ruleSuggestThenQuery,
				Message: "hand-rolled eventually loop (deadline condition + time.Sleep polling) — " +
					"systemscenario.ThenQuery / ThenQueryEventuallyFails provide the poll, " +
					"timeout, and last-mismatch diagnostics as tested assertions",
			})

			return true
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })

	return out
}

// isEventuallyLoop reports whether loop is the eventually idiom: the
// condition mentions time.Now (a deadline comparison) and the body sleeps
// between probes. Both signals together keep ordinary retry/backoff loops
// in tests (sleep without a deadline condition) silent.
func isEventuallyLoop(gf *cqrsanalyzer.GoFile, loop *ast.ForStmt) bool {
	if loop.Cond == nil || loop.Body == nil {
		return false
	}

	return mentionsTimeNow(gf, loop.Cond) && containsTimeSleep(gf, loop.Body)
}

// mentionsTimeNow reports whether expr contains a time.Now selector call.
func mentionsTimeNow(gf *cqrsanalyzer.GoFile, expr ast.Expr) bool {
	found := false

	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := cqrsanalyzer.SelectorFromExpr(call.Fun)
		if ok && sel.Sel.Name == "Now" && qualifierIsTime(gf, sel) {
			found = true
		}

		return !found
	})

	return found
}

// containsTimeSleep reports whether stmt's subtree calls time.Sleep.
func containsTimeSleep(gf *cqrsanalyzer.GoFile, stmt ast.Stmt) bool {
	found := false

	ast.Inspect(stmt, func(n ast.Node) bool {
		if found {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := cqrsanalyzer.SelectorFromExpr(call.Fun)
		if ok && sel.Sel.Name == "Sleep" && qualifierIsTime(gf, sel) {
			found = true
		}

		return !found
	})

	return found
}

// qualifierIsTime reports whether sel's qualifier refers to the stdlib
// time package. Typed resolution wins (exact import path, alias-aware);
// the bare-name fallback covers syntax-only loads (the stdlib package name
// is always "time" unless aliased, which this idiom does not do).
func qualifierIsTime(gf *cqrsanalyzer.GoFile, sel *ast.SelectorExpr) bool {
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	if path, resolved := cqrsanalyzer.ResolveQualifierTyped(gf, ident); resolved {
		return path == "time"
	}

	return ident.Name == "time"
}
