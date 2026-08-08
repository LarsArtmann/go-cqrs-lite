package resilience

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B031: Missing dead-letter queue configuration.
// Detects projectionhost.New() calls that do not configure
// WithDeadLetterStore. Without a DLQ, poison events cause terminal
// worker failure with no recovery path.
//
//nolint:ireturn // factory returns public interface
func NewB031Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B031-missing-dead-letter-config",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.IsLibrarySelfLint() {
				return nil, nil
			}

			var findings []finding.Finding

			projectHasDLQ := projectUsesDeadLetterStore(ctx)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					pkg, ok := sel.X.(*ast.Ident)
					if !ok || pkg.Name != "projectionhost" {
						return true
					}

					if sel.Sel.Name != "New" {
						return true
					}

					if projectHasDLQ {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					findings = append(findings, singleInfoFinding(
						ctx,
						"B031",
						"projectionhost.New() called without WithDeadLetterStore — "+
							"poison events will cause terminal worker failure",
						"Add projectionhost.WithDeadLetterStore(store, maxRetries) "+
							"to isolate poison events for manual inspection and replay",
						pos,
						finding.ConfidenceLow,
					...)...)

					return true
				})
			}

			return findings, nil
		},
	)
}

// projectUsesDeadLetterStore reports whether any non-test file references
// WithDeadLetterStore or DeadLetterStore.
func projectUsesDeadLetterStore(ctx *analyzer.AnalysisContext) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			if ident, ok := n.(*ast.Ident); ok {
				name := strings.ToLower(ident.Name)
				if strings.Contains(name, "deadletter") || strings.Contains(name, "dead_letter") {
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
