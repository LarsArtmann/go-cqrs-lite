package correctness

import (
	"context"
	"go/ast"
	"slices"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects ignored errors from lifecycle methods (Stop, Close, Shutdown,
// GracefulClose). Ignoring these can lose pending events or leak resources.
//
//nolint:ireturn // factory returns public interface
func NewC023Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C023-shutdown-error-ignored",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			lifecycleMethods := map[string]bool{
				"Stop":          true,
				"Close":         true,
				"Shutdown":      true,
				"GracefulClose": true,
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				findings = append(findings, scanLifecycleIgnores(ctx, gf, lifecycleMethods)...)
			}

			return findings, nil
		},
	)
}

// scanLifecycleIgnores walks a file in a single O(N) pass, maintaining an
// ancestor stack. When it finds an assignment that discards a lifecycle-method
// error (`_ = x.Stop()`), it checks the ancestor chain for a DeferStmt —
// deferred close/stop is the standard Go cleanup pattern where errors are
// conventionally ignored.
func scanLifecycleIgnores(
	ctx *analyzer.AnalysisContext,
	gf *analyzer.GoFile,
	methods map[string]bool,
) []finding.Finding {
	var ancestors []ast.Node
	var findings []finding.Finding

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		if n == nil {
			if len(ancestors) > 0 {
				ancestors = ancestors[:len(ancestors)-1]
			}

			return false
		}

		assign, ok := n.(*ast.AssignStmt)
		if ok && isLifecycleIgnore(assign, methods) && !hasDeferAncestor(ancestors) {
			emitC023(ctx, assign, &findings)
		}

		ancestors = append(ancestors, n)

		return true
	})

	return findings
}

func isLifecycleIgnore(assign *ast.AssignStmt, methods map[string]bool) bool {
	if len(assign.Lhs) != 1 {
		return false
	}

	ident, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || ident.Name != "_" {
		return false
	}

	if len(assign.Rhs) != 1 {
		return false
	}

	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	return methods[sel.Sel.Name]
}

func hasDeferAncestor(ancestors []ast.Node) bool {
	for _, v := range slices.Backward(ancestors) {
		if _, ok := v.(*ast.DeferStmt); ok {
			return true
		}
	}

	return false
}

func emitC023(ctx *analyzer.AnalysisContext, assign *ast.AssignStmt, findings *[]finding.Finding) {
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	pos := ctx.Fset.Position(assign.Pos())

	f, err := finding.NewBuilder(
		"C023", toolName,
		sel.Sel.Name+"() error ignored — pending events or resources may be lost",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceMedium).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion("Check the error from " + sel.Sel.Name +
			"() and log/handle failures during shutdown").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return
	}

	*findings = append(*findings, f)
}
