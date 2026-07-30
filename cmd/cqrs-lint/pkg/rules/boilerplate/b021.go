package boilerplate

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects decider fold/apply functions that don't use decider.StrictApply.
// Without StrictApply, unknown event types are silently ignored, hiding
// bugs when new event types are added.
//
// B021: Missing StrictApply in fold functions.
//
//nolint:ireturn // factory returns public interface
func NewB021Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B021-fold-without-strictapply",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				if foldHasStrictApply(ctx, fold.FuncName) {
					continue
				}

				// Only flag folds whose default case returns nil (silently ignores).
				if !fold.DefaultNil {
					continue
				}

				f, err := finding.NewBuilder(
					"B021", toolName,
					"Fold function silently ignores unknown event types — "+
						"use decider.StrictApply for compile-time safety",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(fold.File), fold.Pos.Line, fold.Pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Replace the switch/default pattern with decider.StrictApply " +
						"to get compile-time exhaustiveness checking").
					WithSnippet(ctx.SourceLine(fold.File, fold.Pos.Line)).
					Build()
				if err != nil {
					continue
				}

				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// foldHasStrictApply checks if a fold function's body contains a call to
// StrictApply.
func foldHasStrictApply(ctx *analyzer.AnalysisContext, funcName string) bool {
	fn := findFoldFunc(ctx, funcName)
	if fn == nil || fn.Body == nil {
		return false
	}

	found := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok {
			return true
		}

		if sel.Sel.Name == "StrictApply" {
			found = true
			return false
		}

		return true
	})

	return found
}

func findFoldFunc(ctx *analyzer.AnalysisContext, name string) *ast.FuncDecl {
	for _, gf := range ctx.GoFiles {
		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}

			if fn.Name.Name == name {
				return fn
			}
		}
	}

	return nil
}
