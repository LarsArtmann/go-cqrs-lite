package api

import (
	"context"
	"fmt"
	"go/ast"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// A011: Inconsistent JSON key casing in event payloads.
// Detects event payload structs (named *Created, *Updated, *Deleted, *Event)
// with mixed camelCase and snake_case JSON tags.

//nolint:ireturn // factory returns public interface
func NewA011Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	suffixes := []string{"Created", "Updated", "Deleted", "Removed", "Added", "Changed", "Event"}

	return finding.NamedDetectorFunc(
		"A011-inconsistent-json-key-casing-event-payloads",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					ts, ok := n.(*ast.TypeSpec)
					if !ok {
						return true
					}

					name := ts.Name.Name
					if !slices.ContainsFunc(
						suffixes,
						func(s string) bool { return strings.HasSuffix(name, s) },
					) {
						return true
					}

					st, ok := ts.Type.(*ast.StructType)
					if !ok || st.Fields == nil {
						return true
					}

					camelCount, snakeCount := countJSONKeyCasings(st)
					if camelCount > 0 && snakeCount > 0 {
						pos := ctx.Fset.Position(ts.Pos())

						f, err := findingTemplate.Builder(
							"A011",
							fmt.Sprintf(
								"Event payload %s has mixed JSON key casing (%d camelCase, %d snake_case)",
								name,
								camelCount,
								snakeCount,
							),
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceLow).
							WithSuggestion("Standardize on one JSON key casing convention for event payloads").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

func countJSONKeyCasings(st *ast.StructType) (int, int) {
	var camel, snake int
	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}

		tag := field.Tag.Value

		jsonTag := analyzer.ExtractJSONTag(tag)
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		if strings.Contains(jsonTag, "_") {
			snake++
		} else if hasCamelHump(jsonTag) {
			camel++
		}
	}

	return camel, snake
}

// hasCamelHump reports whether the JSON key contains a lowercase→uppercase
// transition (the camelCase hump, as in "createdAt"). Single-word keys ("id",
// "kind") and screaming-case keys ("ID") carry no case convention — counting
// them as camelCase manufactures a mixed-casing signal where none exists
// (nsfw-classifier feedback, 2026-10-03).
func hasCamelHump(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' && s[i-1] >= 'a' && s[i-1] <= 'z' {
			return true
		}
	}

	return false
}

// A014: Deprecated API usage.
// Detects calls to APIs the library itself marks Deprecated (see
// deprecatedAPIEntries in a014_deprecated.go). The qualifier is resolved
// through the file's import declarations (lintutil.QualifierToImportPath),
// so aliased imports are detected and a consumer's own package named
// "event" no longer false-positives.
//
// dispatcher.Register and command.Register are deliberately NOT in the
// table: neither carries a Deprecated marker (dispatcher/v4/dispatcher.go,
// command/v4/dispatcher.go) and command.Bus is a kept surface — preferring
// RegisterTyped is coaching (A004), not a deprecation. Same ruling as the
// event.NewEvent removal (2026-10-08).

//
//nolint:ireturn // factory returns public interface
func NewA014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A014-deprecated-api-usage",
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

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					qualifier := analyzer.SelectorPackage(sel)
					if qualifier == "" {
						return true
					}

					entry, deprecated := matchDeprecatedAPI(gf.AST, qualifier, sel.Sel.Name)
					if !deprecated {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := findingTemplate.Builder(
						"A014",
						fmt.Sprintf(
							"Deprecated API %s.%s — use %s instead",
							qualifier,
							sel.Sel.Name,
							entry.replacement,
						),
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Migrate to the recommended replacement API").
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

// A017: Missing snapshot strategy.
// Detects repositories created with WithSnapshotStore but without
// WithSnapshotStrategy — the store is useless without a strategy.
// Also flags repositories with neither snapshot store nor state cache.
//
//nolint:ireturn // factory returns public interface
func NewA017Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A017-missing-snapshot-strategy",
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

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok || sel.Sel.Name != "NewRepository" {
						return true
					}

					hasSnapshotStore := false
					hasSnapshotStrategy := false
					hasStateCache := false

					for _, arg := range call.Args {
						if fnCall, ok := arg.(*ast.CallExpr); ok {
							if fnSel, ok := analyzer.SelectorFromExpr(fnCall.Fun); ok {
								switch fnSel.Sel.Name {
								case "WithSnapshotStore":
									hasSnapshotStore = true
								case "WithSnapshotStrategy":
									hasSnapshotStrategy = true
								case "WithStateCache":
									hasStateCache = true
								}
							}
						}
					}

					pos := ctx.Fset.Position(call.Pos())

					// WithSnapshotStore without WithSnapshotStrategy — store is useless.
					if hasSnapshotStore && !hasSnapshotStrategy {
						f, err := findingTemplate.Builder(
							"A017",
							"WithSnapshotStore without WithSnapshotStrategy — "+
								"snapshot store is never triggered, snapshots are never taken",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceHigh).
							WithSuggestion("Add decider.WithSnapshotStrategy(snapshot.EveryNEvents(n)) " +
								"to actually trigger snapshot creation").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)
						return true
					}

					// No snapshot store and no state cache — slow loads on long streams.
					if !hasSnapshotStore && !hasStateCache {
						f, err := findingTemplate.Builder(
							"A017",
							"Repository created without snapshot strategy — "+
								"long event streams will cause slow loads",
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceLow).
							WithSuggestion("Add decider.WithSnapshotStore(snapStore) " +
								"with decider.WithSnapshotStrategy(snapshot.EveryNEvents(n)), " +
								"or decider.WithStateCache(cache) for large aggregates").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}
