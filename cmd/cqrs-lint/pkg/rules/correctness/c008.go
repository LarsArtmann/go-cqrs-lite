package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C008: float64 for money.
// Detects float64 fields with monetary names in struct types.
func NewC008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C008-float64-for-money",
		func(_ context.Context) ([]finding.Finding, error) {
			moneyFields := []string{
				"amount",
				"price",
				"cost",
				"balance",
				"total",
				"fee",
				"charge",
				"payment",
				"salary",
				"value",
			}

			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					st, ok := n.(*ast.StructType)
					if !ok || st.Fields == nil {
						return true
					}

					for _, field := range st.Fields.List {
						if !isFloat64(field.Type) {
							continue
						}

						for _, name := range field.Names {
							lowerName := strings.ToLower(name.Name)
							for _, mf := range moneyFields {
								if strings.Contains(lowerName, mf) {
									pos := ctx.Fset.Position(name.Pos())

									f, err := finding.NewBuilder(
										"C008", toolName,
										fmt.Sprintf("Field %s is float64 — use decimal or integer cents for money to avoid rounding errors", name.Name),
										finding.SeverityWarning,
										finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
									).
										WithCategory(finding.CategoryCorrectness).
										WithConfidence(finding.ConfidenceMedium).
										WithSuggestion("Use shopspring/decimal or int64 cents instead of float64 for monetary values").
										WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
										Build()
									if err != nil {
										return true
									}

									findings = append(findings, f)

									break
								}
							}
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}
