package consistency

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// D006: Missing errorfamily classification — plain errors.New in production code.
//
// Detects errors.New() and fmt.Errorf() without %w in non-test source files.
// The go-cqrs-lite error taxonomy (ADR: 6-family model — Rejection, Conflict,
// Transient, Infrastructure, Corruption, Orchestration) requires classified
// constructors: errorfamily.NewRejection, errorfamily.WrapConflict, etc.
// Plain errors.New produces an unclassified error that bypasses the taxonomy,
// making it impossible for consumers to distinguish business-rule rejections
// from infrastructure failures.
//
// Exceptions (not flagged):
//   - Test files (_test.go)
//   - Sentinel errors defined at package level (var ErrXxx = errors.New(...))
//     — sentinels are pattern-matched by errors.Is, not classified.
//   - fmt.Errorf with %w (wrapping preserves classification)
//   - fmt.Errorf in CQRS files (owned by C025 at warning severity)
//
//nolint:ireturn // factory returns public interface
func NewD006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D006-missing-errorfamily",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			sentinels := lintutil.CollectPkgLevelVarCalls(ctx)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				cqrsFile := lintutil.FileImportsCQRS(gf.AST)

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					// Check errors.New(...)
					if isErrorsNew(call) {
						// Skip package-level sentinel declarations:
						// var ErrXxx = errors.New(...)
						if sentinels[call.Pos()] {
							return true
						}

						reportUnclassified(ctx, &findings, call, "errors.New")
						return true
					}

					// Check fmt.Errorf without %w.
					// C025 owns fmt.Errorf in CQRS files (warning severity);
					// D006 only reports fmt.Errorf in non-CQRS files.
					if !cqrsFile && lintutil.IsFmtErrorf(call) && !lintutil.HasWrapVerb(call) {
						reportUnclassified(ctx, &findings, call, "fmt.Errorf (no %w)")
						return true
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

func isErrorsNew(call *ast.CallExpr) bool {
	return isPkgSelectorCall(call, "errors", "New")
}

// isPkgSelectorCall returns true if call is pkgName.methodName(...).
func isPkgSelectorCall(call *ast.CallExpr, pkgName, methodName string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)

	return ok && ident.Name == pkgName && sel.Sel.Name == methodName
}

func reportUnclassified(
	ctx *analyzer.AnalysisContext,
	findings *[]finding.Finding,
	call *ast.CallExpr,
	ctor string,
) {
	pos := ctx.Fset.Position(call.Pos())

	f, err := finding.NewBuilder(
		"D006",
		toolName,
		fmt.Sprintf(
			"%s at %s — unclassified error, bypasses the 6-family error taxonomy",
			ctor, pos.String(),
		),
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(finding.ConfidenceMedium).
		WithSuggestion(
			"Use errorfamily.NewRejection/NewConflict/NewTransient/NewInfrastructure/NewCorruption/NewOrchestration " +
				"for classified errors, or errorfamily.WrapConflict/WrapTransient/etc. to wrap an existing error. " +
				"For sentinel errors matched by errors.Is, package-level var ErrXxx = errors.New(...) is acceptable.",
		).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	lintutil.AppendBuild(findings, f, err)
}
