package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C004: Checkpoint before async complete.
// Detects goroutine launches inside projection Handle methods — the checkpoint
// may be saved before the async work finishes, causing data loss on crash.
func NewC004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C004-checkpoint-before-async-complete",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, proj := range ctx.Registry.Projections {
				if !proj.HasAsync {
					continue
				}

				f, err := finding.NewBuilder(
					"C004",
					toolName,
					"Projection launches async work — checkpoint may be saved before completion, causing data loss on crash",
					finding.SeverityError,
					finding.Pos(finding.FilePath(proj.File), proj.Pos.Line, proj.Pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Make Handle synchronous, or use projectionhost with ordered delivery and retry").
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

// C011: Nondeterministic decider.
// Detects rand.* calls and global variable access inside decider decide functions.
func NewC011Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C011-nondeterministic-decider",
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

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						pkgIdent, ok := sel.X.(*ast.Ident)
						if !ok {
							return true
						}

						if pkgIdent.Name != "rand" && pkgIdent.Name != "mathrand" {
							return true
						}

						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"C011", toolName,
							"rand.* call inside decider — non-deterministic, breaks event sourcing replay",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceLow).
							WithSuggestion("Inject randomness via command parameters or a clock/seed interface").
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
