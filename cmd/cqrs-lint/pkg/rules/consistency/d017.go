package consistency

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// D017: Raw errors in domain decider files.
//
// Extends D006 with stricter detection in domain files. While D006 reports
// unclassified errors at info severity across all files, D017 escalates to
// warning severity when the unclassified error appears in a file that contains
// a fold/decide function — the domain core where business rule violations
// should use errorfamily.NewRejection/NewConflict, not plain errors.New.
//
// A rejection error built with errors.New bypasses the 6-family taxonomy,
// making it impossible for middleware to distinguish business-rule rejections
// (retry = no-op, alert = false positive) from infrastructure failures
// (retry = yes, alert = true).
//
//nolint:ireturn // factory returns public interface
func NewD017Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D017-raw-errors-in-domain",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			domainFiles := make(map[string]bool)
			for _, fold := range ctx.Registry.Folds {
				domainFiles[fold.File] = true
			}

			sentinels := lintutil.CollectPkgLevelVarCalls(ctx)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if !domainFiles[gf.Path] {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					if sentinels[call.Pos()] {
						return true
					}

					if isRawErrorInDomain(call) {
						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"D017", toolName,
							fmt.Sprintf(
								"unclassified error in domain file at %s — "+
									"business rule violations must use errorfamily.NewRejection/NewConflict",
								pos.String(),
							),
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryStyle).
							WithConfidence(finding.ConfidenceHigh).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion(
								"Use errorfamily.NewRejection(code, msg) for business rule violations, " +
									"errorfamily.NewConflict(code, msg) for state conflicts, or " +
									"errorfamily.WrapTransient(err, code, msg) for retryable failures",
							).
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						lintutil.AppendBuild(&findings, f, err)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// isRawErrorInDomain returns true for errors.New or fmt.Errorf (without %w)
// calls — the patterns D017 owns in domain files.
func isRawErrorInDomain(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	if ident.Name == "errors" && sel.Sel.Name == "New" {
		return true
	}

	if ident.Name == "fmt" && sel.Sel.Name == "Errorf" && !hasWrapVerb(call) {
		return true
	}

	return false
}

// hasWrapVerb returns true if the format string contains %w.
func hasWrapVerb(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok {
		return true
	}

	return strings.Contains(lit.Value, "%w")
}
