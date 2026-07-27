package consistency

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
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
//
//nolint:ireturn // factory returns public interface
func NewD006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D006-missing-errorfamily",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					// Check errors.New(...)
					if isErrorsNew(call) {
						// Skip package-level sentinel declarations:
						// var ErrXxx = errors.New(...)
						if isPackageLevelVar(ctx, call) {
							return true
						}

						reportUnclassified(ctx, &findings, call, "errors.New")
						return true
					}

					// Check fmt.Errorf without %w
					if isFmtErrorf(call) && !hasWrapVerb(call) {
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

func isFmtErrorf(call *ast.CallExpr) bool {
	return isPkgSelectorCall(call, "fmt", "Errorf")
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

// hasWrapVerb returns true if the format string contains %w (error wrapping).
func hasWrapVerb(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok {
		// Non-literal format string — can't analyze statically.
		return true
	}

	// Check if the string value contains %w.
	return containsString(lit.Value, "%w")
}

func containsString(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// isPackageLevelVar checks if the call is the RHS of a package-level var
// declaration (sentinel error pattern).
func isPackageLevelVar(ctx *analyzer.AnalysisContext, call *ast.CallExpr) bool {
	for _, gf := range ctx.GoFiles {
		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok.String() != "var" {
				continue
			}

			for _, spec := range genDecl.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for _, val := range valSpec.Values {
					if c, ok := val.(*ast.CallExpr); ok && c.Pos() == call.Pos() {
						return true
					}
				}
			}
		}
	}

	return false
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
	if err != nil {
		return
	}

	*findings = append(*findings, f)
}
