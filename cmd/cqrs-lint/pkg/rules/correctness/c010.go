package correctness

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects `_, _ = decode(evt)` or `_, := decode(evt); _ = err` in fold
// functions AND inline closures assigned to CQRS callback fields
// (OnCreate, OnUpdate, OnTombstone, Apply, Fold, Handle).
//
//nolint:ireturn // factory returns public interface
func NewC010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C010-swallowed-error-in-fold",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				for _, gf := range ctx.GoFiles {
					if gf.Path != fold.File || gf.IsTest {
						continue
					}

					ast.Inspect(gf.AST, func(n ast.Node) bool {
						fn, ok := n.(*ast.FuncDecl)
						if !ok || fn.Name == nil {
							return true
						}

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

			// Also scan inline closures assigned to CQRS callback fields.
			cqrsCallbackFields := map[string]bool{
				"OnCreate": true, "OnUpdate": true, "OnTombstone": true,
				"Apply": true, "Fold": true, "Handle": true,
				"HandleEvent": true, "HandleFunc": true,
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					assign, ok := n.(*ast.AssignStmt)
					if !ok {
						return true
					}

					for i, lhs := range assign.Lhs {
						if i >= len(assign.Rhs) {
							break
						}

						sel, ok := lhs.(*ast.SelectorExpr)
						if !ok || !cqrsCallbackFields[sel.Sel.Name] {
							continue
						}

						lit, ok := assign.Rhs[i].(*ast.FuncLit)
						if !ok || lit.Body == nil {
							continue
						}

						inspectBodyForSwallowedError(ctx, lit.Body, &findings)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}
