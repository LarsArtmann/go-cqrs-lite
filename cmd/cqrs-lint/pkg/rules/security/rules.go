// Package security implements security-related detection rules.
package security

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

const toolName finding.ToolName = "cqrs-lint"

// S001: Hardcoded secrets.
// Detects fields with secret-like names assigned string literals.
func NewS001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S001-hardcoded-secrets",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			secretFields := []string{
				"secret",
				"password",
				"passwd",
				"apikey",
				"api_key",
				"token",
				"privatekey",
				"private_key",
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					assign, ok := n.(*ast.AssignStmt)
					if !ok {
						return true
					}

					for i, lhs := range assign.Lhs {
						if i >= len(assign.Rhs) {
							break
						}

						lit, ok := assign.Rhs[i].(*ast.BasicLit)
						if !ok || lit.Kind != token.STRING {
							continue
						}

						val := strings.Trim(lit.Value, `"`)
						if len(val) < 8 {
							continue
						}

						var fieldName string
						if id, ok := lhs.(*ast.Ident); ok {
							fieldName = strings.ToLower(id.Name)
						} else if sel, ok := lhs.(*ast.SelectorExpr); ok {
							fieldName = strings.ToLower(sel.Sel.Name)
						}

						for _, sf := range secretFields {
							if strings.Contains(fieldName, sf) {
								pos := ctx.Fset.Position(assign.Pos())

								f, err := finding.NewBuilder(
									"S001", toolName,
									fmt.Sprintf("Potential hardcoded secret in field %q — use environment variables or a secret manager", fieldName),
									finding.SeverityCritical,
									finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
								).
									WithCategory(finding.CategorySecurity).
									WithConfidence(finding.ConfidenceMedium).
									WithSuggestion("Load secrets from environment variables (os.Getenv) or a secret manager, never hardcode them").
									WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
									Build()
								if err == nil {
									findings = append(findings, f)
								}

								break
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
