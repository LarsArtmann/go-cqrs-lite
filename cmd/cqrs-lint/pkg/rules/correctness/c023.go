package correctness

import (
	"context"
	"go/ast"

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

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					assign, ok := n.(*ast.AssignStmt)
					if !ok || len(assign.Lhs) != 1 {
						return true
					}

					ident, ok := assign.Lhs[0].(*ast.Ident)
					if !ok || ident.Name != "_" {
						return true
					}

					if len(assign.Rhs) != 1 {
						return true
					}

					call, ok := assign.Rhs[0].(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					if !lifecycleMethods[sel.Sel.Name] {
						return true
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
						return true
					}

					findings = append(findings, f)
					return true
				})
			}

			return findings, nil
		},
	)
}
