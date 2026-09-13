// Package security implements security-related detection rules.
package security

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

const toolName = lintutil.ToolName

// S001: Hardcoded secrets.
// Detects fields with secret-like names assigned string literals. Coverage:
// plain assignments (cfg.Password = "…", m["api_key"] = "…"), package-level
// var/const declarations, and composite-literal fields (Config{Password:
// "…"}) — the last two were silently skipped by the original AssignStmt-only
// walk even though they are the most common placements.
//
// Values that are URLs or unfilled placeholder templates are allowlisted:
// documentation links and env-var/insertion templates trip the name
// heuristic without embedding a credential.
//
//nolint:ireturn // factory returns public interface
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

			check := func(pos token.Pos, fieldName string, expr ast.Expr) {
				lit, ok := expr.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return
				}

				val := strings.Trim(lit.Value, "\"`")
				if len(val) < 8 {
					return
				}

				if isURLOrPlaceholder(val) {
					return
				}

				lower := strings.ToLower(fieldName)

				for _, sf := range secretFields {
					if !strings.Contains(lower, sf) {
						continue
					}

					p := ctx.Fset.Position(pos)

					f, err := finding.NewBuilder(
						"S001", toolName,
						fmt.Sprintf("Potential hardcoded secret in field %q — use environment variables or a secret manager", lower),
						finding.SeverityCritical,
						finding.Pos(finding.FilePath(p.Filename), p.Line, p.Column),
					).
						WithCategory(finding.CategorySecurity).
						WithConfidence(finding.ConfidenceMedium).
						WithSuggestion("Load secrets from environment variables (os.Getenv) or a secret manager, never hardcode them").
						WithSnippet(ctx.SourceLine(p.Filename, p.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}

					break
				}
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					switch node := n.(type) {
					case *ast.AssignStmt:
						for i, lhs := range node.Lhs {
							if i >= len(node.Rhs) {
								break
							}

							check(node.Pos(), s001LHSName(lhs), node.Rhs[i])
						}
					case *ast.GenDecl:
						for _, spec := range node.Specs {
							vs, ok := spec.(*ast.ValueSpec)
							if !ok {
								continue
							}

							for i, name := range vs.Names {
								if i >= len(vs.Values) {
									break
								}

								check(node.Pos(), name.Name, vs.Values[i])
							}
						}
					case *ast.KeyValueExpr:
						if key, ok := node.Key.(*ast.Ident); ok {
							check(node.Pos(), key.Name, node.Value)
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// isURLOrPlaceholder reports whether a string literal value is a URL or an
// unfilled placeholder template rather than a real credential. Documentation
// links (apiKeyDocsURL = "https://…") and env-var/insertion templates
// ("${API_KEY}", "<your-token>") trip the secret-name heuristic without
// embedding a secret. Tradeoff: credential-bearing DSNs (postgres://…)
// are also skipped, but those live under dsn/connectionString-style names,
// not the secret keywords this rule matches on.
func isURLOrPlaceholder(val string) bool {
	if strings.Contains(val, "://") {
		return true
	}

	trimmed := strings.TrimSpace(val)

	return strings.HasPrefix(trimmed, "${") ||
		(strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">"))
}

// s001LHSName extracts the secret-name candidate from an assignment target:
// the identifier (x = …), the selector field (cfg.Password = …), or a string
// map key (m["api_key"] = …). Empty when the target carries no name.
func s001LHSName(lhs ast.Expr) string {
	switch t := lhs.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.IndexExpr:
		if key, ok := t.Index.(*ast.BasicLit); ok {
			return strings.Trim(key.Value, "\"`")
		}
	}

	return ""
}
