package boilerplate

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B004: Command constructor boilerplate.
// Detects manual command construction with repetitive field assignment.
func NewB004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B004-command-constructor-boilerplate",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, cmd := range ctx.Registry.Commands {
				if len(cmd.Fields) < 3 {
					continue
				}

				f, err := finding.NewBuilder(
					"B004",
					toolName,
					fmt.Sprintf(
						"Command %s has %d fields — consider using cqrs-gen to generate constructors",
						cmd.Name,
						len(cmd.Fields),
					),
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(cmd.File), cmd.Pos.Line, cmd.Pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion("Run cqrs-gen to auto-generate typed constructors from struct tags").
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

// B005: Fold switch boilerplate.
// Detects fold functions with large switch statements that could use a dispatch map.
func NewB005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B005-fold-switch-boilerplate",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				if !fold.HasSwitch {
					continue
				}

				f, err := finding.NewBuilder(
					"B005", toolName,
					fmt.Sprintf("Fold %s uses a switch statement — consider decider.StrictApply for compile-time exhaustiveness", fold.FuncName),
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(fold.File), fold.Pos.Line, fold.Pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion("Use decider.StrictApply[T](fold, initial) to error on unknown event types at runtime").
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

// B008: Manual retry implementation.
// Detects hand-written retry loops instead of using retry.Do.
func NewB008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B008-manual-retry-implementation",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					fn, ok := n.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						return true
					}

					ast.Inspect(fn.Body, func(nn ast.Node) bool {
						loop, ok := nn.(*ast.ForStmt)
						if !ok {
							return true
						}

						hasRetry := false

						ast.Inspect(loop, func(nnn ast.Node) bool {
							ident, ok := nnn.(*ast.Ident)
							if !ok {
								return true
							}

							if strings.Contains(strings.ToLower(ident.Name), "retry") ||
								strings.Contains(strings.ToLower(ident.Name), "attempt") {
								hasRetry = true

								return false
							}

							return true
						})

						if !hasRetry {
							return true
						}

						hasSleep := false

						ast.Inspect(loop, func(nnn ast.Node) bool {
							call, ok := nnn.(*ast.CallExpr)
							if !ok {
								return true
							}

							sel, ok := call.Fun.(*ast.SelectorExpr)
							if !ok {
								return true
							}

							if sel.Sel.Name == "Sleep" {
								hasSleep = true

								return false
							}

							return true
						})

						if !hasSleep {
							return true
						}

						pos := ctx.Fset.Position(loop.Pos())

						f, err := finding.NewBuilder(
							"B008",
							toolName,
							fmt.Sprintf(
								"Manual retry loop in %s — use retry.Do for exponential backoff with jitter",
								fn.Name.Name,
							),
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceHigh).
							WithSuggestion("Replace with retry.Do(ctx, func() error { ... }, retry.WithMaxRetries(3))").
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)

						return true
					})

					return true
				})
			}

			return findings, nil
		},
	)
}
