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

			check := func(pos token.Pos, fieldName, display string, expr ast.Expr, isConst bool) {
				lit, ok := expr.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return
				}

				val := strings.Trim(lit.Value, "\"`")
				if len(val) < 8 {
					return
				}

				if lintutil.IsURLOrPlaceholder(val) {
					return
				}

				// Name-shaped literals (pure lowercase snake_case, the shape of
				// schema vocabulary like "enroll_tokens") are only skipped when
				// the binding itself says "identifier": a const declaration or a
				// structural name suffix. Applied unconditionally this would
				// also skip lowercase API keys, which have the same shape.
				if lintutil.IsNameShapedValue(val) &&
					(isConst || isSchemaVocabularyName(fieldName)) {
					return
				}

				lower := strings.ToLower(fieldName)

				for _, sf := range secretFields {
					if !strings.Contains(lower, sf) {
						continue
					}

					p := ctx.Fset.Position(pos)
					severity := finding.SeverityCritical
					suggestion := "Load secrets from environment variables (os.Getenv) or a secret manager, never hardcode them"

					if isConst {
						// A const cannot be loaded at runtime, so "use env vars"
						// is unactionable advice. Keep the finding visible, but
						// out of the build-gating critical band.
						severity = finding.SeverityWarning
						suggestion = "A const cannot be read from the environment. If this is schema " +
							"vocabulary (a collection, table, or queue name), no action is needed. If it " +
							"is real key material, move it to a var loaded from the environment or a " +
							"secret manager."
					}

					f, err := findingTemplate.Builder(
						"S001",
						fmt.Sprintf(
							"Potential hardcoded secret in field %q — use environment variables or a secret manager",
							display,
						),
						severity,
						finding.Pos(finding.FilePath(p.Filename), p.Line, p.Column),
					).
						WithConfidence(finding.ConfidenceMedium).
						WithSuggestion(suggestion).
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

							check(
								node.Pos(),
								s001LHSName(lhs),
								s001LHSDisplay(lhs),
								node.Rhs[i],
								false,
							)
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

								check(
									node.Pos(),
									name.Name,
									name.Name,
									vs.Values[i],
									node.Tok == token.CONST,
								)
							}
						}
					case *ast.KeyValueExpr:
						if key, ok := node.Key.(*ast.Ident); ok {
							check(node.Pos(), key.Name, key.Name, node.Value, false)
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
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

// s001LHSDisplay renders the assignment target for finding MESSAGES: the
// full selector path ("cfg.Password", not just "Password") so the report
// names the receiver a reader must fix. Falls back to the bare name when
// the receiver is not a simple identifier.
func s001LHSDisplay(lhs ast.Expr) string {
	switch t := lhs.(type) {
	case *ast.SelectorExpr:
		if recv, ok := t.X.(*ast.Ident); ok {
			return recv.Name + "." + t.Sel.Name
		}
	case *ast.IndexExpr:
		if recv, ok := t.X.(*ast.Ident); ok {
			if key, ok := t.Index.(*ast.BasicLit); ok {
				return recv.Name + "[" + strings.Trim(key.Value, "\"`") + "]"
			}
		}
	}

	return s001LHSName(lhs)
}

// isSchemaVocabularyName reports whether an identifier names a storage or
// schema artifact. A name-shaped literal bound to one of these is vocabulary,
// not credential material.
func isSchemaVocabularyName(name string) bool {
	switch {
	case strings.HasSuffix(name, "Collection"), strings.HasSuffix(name, "CollectionName"),
		strings.HasSuffix(name, "Table"), strings.HasSuffix(name, "TableName"),
		strings.HasSuffix(name, "Queue"), strings.HasSuffix(name, "QueueName"),
		strings.HasSuffix(name, "Bucket"), strings.HasSuffix(name, "BucketName"),
		strings.HasSuffix(name, "Topic"), strings.HasSuffix(name, "TopicName"),
		strings.HasSuffix(name, "Index"), strings.HasSuffix(name, "IndexName"),
		strings.HasSuffix(name, "Schema"), strings.HasSuffix(name, "ViewName"):
		return true
	default:
		return false
	}
}
