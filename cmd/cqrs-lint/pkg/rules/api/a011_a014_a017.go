package api

import (
	"context"
	"fmt"
	"go/ast"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
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
// The qualifier is resolved through the file's import declarations
// (lintutil.QualifierResolvesTo), so aliased imports are detected and a
// consumer's own package named "event" no longer false-positives.

// deprecatedAPIEntry pairs a deprecated call target with its replacement.
type deprecatedAPIEntry struct {
	pathFragment string // go-cqrs-lite import path fragment that must own the qualifier
	symbol       string // selector name being called
	replacement  string // sanctioned replacement, shown in the suggestion
}

var deprecatedAPIEntries = []deprecatedAPIEntry{ //nolint:gochecknoglobals // static lookup table
	{
		pathFragment: "go-cqrs-lite/event",
		symbol:       "NewEvent",
		replacement:  "event.New (auto-marshaling, simpler API)",
	},
	{
		pathFragment: "go-cqrs-lite/dispatcher",
		symbol:       "Register",
		replacement:  "RegisterTyped (type-safe handler registration)",
	},
	{
		pathFragment: "go-cqrs-lite/command",
		symbol:       "Register",
		replacement:  "RegisterTyped (type-safe handler registration)",
	},
}

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

					// event.NewEvent inside schema.NewUpcaster closures is the
					// correct API — upcasters reconstruct events from raw bytes.
					if entry.symbol == "NewEvent" && analyzer.IsInsideUpcasterClosure(gf, call) {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A014",
						toolName,
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

// matchDeprecatedAPI reports the deprecated-API entry matching a call of
// selName through qualifier, where the qualifier must resolve (via the file's
// imports) to the entry's go-cqrs-lite path fragment.
func matchDeprecatedAPI(file *ast.File, qualifier, selName string) (deprecatedAPIEntry, bool) {
	for _, e := range deprecatedAPIEntries {
		if e.symbol == selName && lintutil.QualifierResolvesTo(file, qualifier, e.pathFragment) {
			return e, true
		}
	}

	return deprecatedAPIEntry{}, false
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
						f, err := finding.NewBuilder(
							"A017", toolName,
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
						f, err := finding.NewBuilder(
							"A017", toolName,
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
