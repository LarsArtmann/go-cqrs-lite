package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects _ = ctx or ctx = _ patterns inside functions with a context.Context
// parameter. Explicitly discarding the context breaks cancellation, timeouts,
// and tracing propagation.
//
//nolint:ireturn // factory returns public interface
func NewC022Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C022-context-discarded",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil || fn.Type == nil {
						continue
					}

					if !hasContextParam(fn.Type.Params) {
						continue
					}

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						assign, ok := n.(*ast.AssignStmt)
						if !ok {
							return true
						}

						if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
							return true
						}

						ident, ok := assign.Lhs[0].(*ast.Ident)
						if !ok || ident.Name != "_" {
							return true
						}

						rhs, ok := assign.Rhs[0].(*ast.Ident)
						if !ok || rhs.Name != "ctx" {
							return true
						}

						pos := ctx.Fset.Position(assign.Pos())

						f, err := finding.NewBuilder(
							"C022", toolName,
							"Context explicitly discarded (_ = ctx) — "+
								"breaks cancellation, timeouts, and tracing",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceHigh).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Pass ctx to downstream calls or store it for later use").
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

func hasContextParam(params *ast.FieldList) bool {
	if params == nil {
		return false
	}

	for _, field := range params.List {
		sel, ok := field.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "context" {
			continue
		}

		if sel.Sel.Name == "Context" {
			return true
		}
	}

	return false
}
