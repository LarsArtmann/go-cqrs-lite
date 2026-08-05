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
//
//nolint:ireturn // factory returns public interface
func NewB004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B004-command-constructor-boilerplate",
		func(_ context.Context) ([]finding.Finding, error) {
			constructorNames := collectConstructorNames(ctx)

			var findings []finding.Finding

			for _, cmd := range ctx.Registry.Commands {
				if len(cmd.Fields) < 3 {
					continue
				}

				// Skip if a hand-written constructor already exists for this
				// command type (NewXxx or NewXxxCommand). Generated constructors
				// would lack any custom validation the hand-written one has.
				if constructorNames["New"+cmd.Name] ||
					constructorNames["New"+cmd.Name+"Command"] {
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
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Run cqrs-gen to auto-generate typed constructors from struct tags").
					WithSnippet(ctx.SourceLine(cmd.File, cmd.Pos.Line)).
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

// collectConstructorNames scans all non-test Go files for top-level function
// declarations matching New* patterns. This lets B004 skip commands that
// already have hand-written constructors (with validation logic that
// generated code would lack).
func collectConstructorNames(ctx *analyzer.AnalysisContext) map[string]bool {
	names := make(map[string]bool)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if strings.HasPrefix(fn.Name.Name, "New") {
				names[fn.Name.Name] = true
			}
		}
	}

	return names
}

// B005: Fold switch boilerplate.
// Detects fold functions with large switch statements that could use a dispatch map.
//
// Suppression: if the fold function is already wrapped in a
// decider.StrictApply call (tracked in Registry.StrictApplyFolds), the
// suggestion is already implemented and the finding is suppressed. The fold
// is matched by the last identifier segment of its FuncName, so a method
// receiver or package qualifier does not prevent suppression. See
// browser-history feedback (B005 latent gap).
//
//nolint:ireturn // factory returns public interface
func NewB005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B005-fold-switch-boilerplate",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				if !fold.HasSwitch {
					continue
				}

				if ctx.Registry.StrictApplyFolds[fold.FuncName] ||
					ctx.Registry.StrictApplyFolds[lastSegmentOfFoldName(fold.FuncName)] {
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

// B008: Manual retry implementation.
// Detects hand-written retry loops instead of using retry.Do.
//
//nolint:ireturn // factory returns public interface
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

							sel, ok := analyzer.SelectorFromExpr(call.Fun)
							if !ok {
								return true
							}

							if sel.Sel.Name == "Sleep" || sel.Sel.Name == "After" ||
								sel.Sel.Name == "NewTimer" ||
								sel.Sel.Name == "NewTicker" {
								hasSleep = true

								return false
							}

							return true
						})

						if !hasSleep {
							return true
						}

						// Check for bitshift backoff bug (item 134).
						hasBitshift := false
						ast.Inspect(loop, func(nnn ast.Node) bool {
							bin, ok := nnn.(*ast.BinaryExpr)
							if !ok {
								return true
							}
							if bin.Op.String() == "<<" || bin.Op.String() == ">>" {
								hasBitshift = true
								return false
							}
							return true
						})

						pos := ctx.Fset.Position(loop.Pos())
						msg := fmt.Sprintf(
							"Manual retry loop in %s — use retry.Do for exponential backoff with jitter",
							fn.Name.Name,
						)
						severity := finding.SeverityWarning
						suggestion := "Replace with retry.Do(ctx, func() error { ... }, retry.WithMaxRetries(3))"
						if hasBitshift {
							msg = fmt.Sprintf(
								"Manual retry loop in %s with bitshift backoff — bitshifting time.Duration produces garbage values",
								fn.Name.Name,
							)
							severity = finding.SeverityError
							suggestion = "Bitshifting Duration values is always wrong — use retry.Do with exponential backoff"
						}

						f, err := finding.NewBuilder(
							"B008",
							toolName,
							msg,
							severity,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceHigh).
							WithSuggestion(suggestion).
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
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

// lastSegmentOfFoldName returns the trailing identifier of a fold FuncName as
// produced by analyzer.funcName. FuncName for a method is "(Recv).name"; for a
// package-qualified call it is collapsed by ExprString. We split on the last
// "." to get the trailing identifier, so StrictApplyFolds (which stores the
// trailing identifier of the decider.StrictApply arg) matches regardless of
// how the fold was named.
func lastSegmentOfFoldName(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}

	return name
}
