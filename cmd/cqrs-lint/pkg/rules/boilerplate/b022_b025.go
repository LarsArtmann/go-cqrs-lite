package boilerplate

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B022: Manual correlation enricher instead of CommandCausalityEnricher.
// Detects custom enricher functions passed to decider.NewRepository that are
// not decider.CommandCausalityEnricher. Custom enrichers miss the typed
// command causality metadata that CommandCausalityEnricher provides.
//
//nolint:ireturn // factory returns public interface
func NewB022Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B022-manual-correlation-enricher",
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

					// Find NewRepository calls from the decider package.
					if sel.Sel.Name != "NewRepository" {
						return true
					}

					pkg := analyzer.SelectorPackage(sel)
					if pkg != "decider" {
						return true
					}

					// Scan arguments for enricher calls that are not
					// CommandCausalityEnricher.
					for _, arg := range call.Args {
						argCall, ok := arg.(*ast.CallExpr)
						if !ok {
							continue
						}

						argSel, ok := analyzer.SelectorFromExpr(argCall.Fun)
						if !ok {
							continue
						}

						argName := argSel.Sel.Name
						if !containsEnricher(argName) {
							continue
						}

						// CommandCausalityEnricher is the recommended enricher.
						if argName == "CommandCausalityEnricher" {
							continue
						}

						// WithCommandCausality is also acceptable (the option
						// form that wraps the enricher).
						if argName == "WithCommandCausality" {
							continue
						}

						pos := ctx.Fset.Position(argCall.Pos())

						f, err := finding.NewBuilder(
							"B022", toolName,
							"Custom enricher ("+argName+") passed to decider.NewRepository — "+
								"use decider.CommandCausalityEnricher for typed command causality",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Replace the custom enricher with decider.CommandCausalityEnricher — "+
								"it stamps metadata.command.type and metadata.command.id on every event").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err == nil {
							findings = append(findings, f)
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

func containsEnricher(s string) bool {
	return strings.Contains(strings.ToLower(s), "enrich")
}

// B025: Missing state cache on repository.
// Detects decider.NewRepository calls without the WithStateCache option.
// For hot streams, incremental loads via state cache are 7.4x faster.
//
//nolint:ireturn // factory returns public interface
func NewB025Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B025-missing-state-cache",
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

					// Find NewRepository calls from the decider package.
					if sel.Sel.Name != "NewRepository" {
						return true
					}

					pkg := analyzer.SelectorPackage(sel)
					if pkg != "decider" {
						return true
					}

					// Check if WithStateCache is among the arguments.
					hasStateCache := false
					for _, arg := range call.Args {
						argSel, ok := analyzer.SelectorFromExpr(arg)
						if !ok {
							continue
						}

						if argSel.Sel.Name == "WithStateCache" {
							hasStateCache = true
							break
						}
					}

					if hasStateCache {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"B025", toolName,
						"Repository created without decider.WithStateCache — "+
							"hot streams benefit from incremental loads (7.4x faster)",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceLow).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Add decider.WithStateCache(decider.NewStateCache[State](256)) to "+
							"NewRepository options for incremental event loading on hot streams").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}
