// Package resilience contains cqrs-lint rules that detect missing resilience
// patterns: absent retry middleware, missing circuit breakers, and missing
// dead-letter configuration on projection hosts.
package resilience

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B029: Missing retry middleware.
// Detects a bus/dispatcher that is created and used without any retry
// middleware registered. B008 detects manual retry; this rule detects
// the absence of middleware-based retry entirely.
//
//nolint:ireturn // factory returns public interface
func NewB029Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B029-missing-retry-middleware",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.IsLibrarySelfLint() {
				return nil, nil
			}

			buses := findBusesWithoutRetry(ctx)

			return buses, nil
		},
	)
}

func findBusesWithoutRetry(ctx *analyzer.AnalysisContext) []finding.Finding {
	var findings []finding.Finding

	busVars := make(map[string]bool)
	hasRetry := make(map[string]bool)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				for _, expr := range node.Lhs {
					if ident, ok := expr.(*ast.Ident); ok {
						name := ident.Name
						if isBusName(name) {
							busVars[name] = true
						}
					}
				}

				for _, expr := range node.Rhs {
					if call, ok := expr.(*ast.CallExpr); ok {
						if hasRetryInCall(call) {
							for _, lhs := range node.Lhs {
								if ident, ok := lhs.(*ast.Ident); ok {
									hasRetry[ident.Name] = true
								}
							}
						}
					}
				}

			case *ast.ExprStmt:
				if call, ok := node.X.(*ast.CallExpr); ok {
					if isUseMiddlewareCall(call, "Retry", "retry") {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							if ident, ok := sel.X.(*ast.Ident); ok {
								hasRetry[ident.Name] = true
							}
						}
					}
				}
			}

			return true
		})
	}

	for bus := range busVars {
		if !hasRetry[bus] {
			findings = append(findings, finding.Finding{
				RuleID:     "B029",
				Title:      "Missing retry middleware on " + bus,
				Category:   "boilerplate",
				Severity:   "info",
				Confidence: "low",
				Summary: "Bus/dispatcher created without retry middleware — " +
					"transient failures will propagate to callers",
				Suggestion: "Add middleware.Retry() or retry.Do to " + bus +
					" for automatic transient failure recovery",
			})
		}
	}

	return findings
}

func isBusName(name string) bool {
	name = strings.ToLower(name)
	return strings.HasSuffix(name, "bus") ||
		strings.HasSuffix(name, "dispatcher") ||
		strings.HasSuffix(name, "disp")
}

func hasRetryInCall(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if ident, ok := arg.(*ast.Ident); ok {
			if strings.Contains(strings.ToLower(ident.Name), "retry") {
				return true
			}
		}
	}

	return false
}

func isUseMiddlewareCall(call *ast.CallExpr, names ...string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	methodName := sel.Sel.Name
	for _, n := range names {
		if methodName == n {
			return true
		}
	}

	return false
}
