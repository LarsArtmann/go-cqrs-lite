package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects time.Now() calls inside decider decide functions.
func NewC007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C007-time-now-in-decider",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						continue
					}

					if !isLikelyDecider(fn) {
						continue
					}

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := analyzer.SelectorFromExpr(call.Fun)
						if !ok {
							return true
						}

						pkgIdent, ok := sel.X.(*ast.Ident)
						if !ok || pkgIdent.Name != "time" || sel.Sel.Name != "Now" {
							return true
						}

						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"C007", toolName,
							"time.Now() inside decider — non-deterministic, makes testing impossible",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceMedium).
							WithSuggestion("Pass time as a parameter or inject a clock interface for deterministic testing").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)

						return true
					})
				}
			}

			return findings, nil
		},
	)
}
