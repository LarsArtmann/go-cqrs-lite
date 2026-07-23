package correctness

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects `_, _ = decode(evt)` or `_, := decode(evt); _ = err` in fold functions.
//nolint:ireturn // factory returns public interface
func NewC010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C010-swallowed-error-in-fold",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				// We need to find the function in the AST to check for swallowed errors.
				for _, gf := range ctx.GoFiles {
					if gf.Path != fold.File || gf.IsTest {
						continue
					}

					ast.Inspect(gf.AST, func(n ast.Node) bool {
						fn, ok := n.(*ast.FuncDecl)
						if !ok || fn.Name == nil {
							return true
						}

						// fold.FuncName may include a receiver prefix (e.g.
						// "MyAggregate.Fold"); strip it to match fn.Name.Name.
						name := fold.FuncName
						if idx := strings.LastIndex(name, "."); idx >= 0 {
							name = name[idx+1:]
						}

						if fn.Name.Name != name {
							return true
						}

						return inspectForSwallowedError(ctx, fn, &findings)
					})
				}
			}

			return findings, nil
		},
	)
}
