package api

import (
	"context"
	"fmt"
	"go/ast"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A011: Inconsistent JSON key casing in event payloads.
// Detects event payload structs (named *Created, *Updated, *Deleted, *Event)
// with mixed camelCase and snake_case JSON tags.

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

						f, err := finding.NewBuilder(
							"A011",
							toolName,
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
		} else if len(jsonTag) > 0 && jsonTag[0] >= 'a' && jsonTag[0] <= 'z' {
			camel++
		}
	}

	return camel, snake
}

// A014: Deprecated API usage.
// Detects calls to deprecated APIs: event.NewEvent (use event.New),
// dispatcher.Register (use RegisterTyped).
func NewA014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	deprecatedAPIs := map[string]string{
		"NewEvent": "event.New (auto-marshaling, simpler API)",
		"Register": "RegisterTyped (type-safe handler registration)",
	}

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

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					replacement, deprecated := deprecatedAPIs[sel.Sel.Name]
					if !deprecated {
						return true
					}

					pkgName := analyzer.SelectorPackage(sel)
					if sel.Sel.Name == "NewEvent" && pkgName != "event" {
						return true
					}

					if sel.Sel.Name == "Register" && pkgName != "dispatcher" &&
						pkgName != "command" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A014",
						toolName,
						fmt.Sprintf(
							"Deprecated API %s.%s — use %s instead",
							pkgName,
							sel.Sel.Name,
							replacement,
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
// Detects repositories created without a snapshot store option.
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

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "NewRepository" {
						return true
					}

					hasSnapshot := false

					for _, arg := range call.Args {
						if fnCall, ok := arg.(*ast.CallExpr); ok {
							if fnSel, ok := fnCall.Fun.(*ast.SelectorExpr); ok {
								if fnSel.Sel.Name == "WithSnapshotStore" ||
									fnSel.Sel.Name == "WithSnapshotStrategy" ||
									fnSel.Sel.Name == "WithStateCache" {
									hasSnapshot = true
								}
							}
						}
					}

					if hasSnapshot {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A017", toolName,
						"Repository created without snapshot strategy — long event streams will cause slow loads",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceLow).
						WithSuggestion("Add decider.WithSnapshotStore(snapStore) or decider.WithStateCache(cache) for large aggregates").
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
