package boilerplate

import (
	"context"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B016: Manual checkpoint replay table.
// Detects SQL tables named "checkpoint" or "projection_offset" combined with
// manual ReadFrom/ReadAll journal loops — logic that duplicates
// projectionhost.Host.
//
//nolint:ireturn // factory returns public interface
func NewB016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B016-manual-checkpoint-replay",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				hasCheckpointTable := false
				hasJournalLoop := false
				var reportPos ast.Node

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					// Detect SQL string literals mentioning checkpoint/projection_offset tables.
					if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						val := strings.ToLower(lit.Value)
						if strings.Contains(val, "checkpoint") || strings.Contains(val, "projection_offset") {
							if strings.Contains(val, "create table") {
								hasCheckpointTable = true
								if reportPos == nil {
									reportPos = lit
								}
							}
						}
					}

					// Detect ReadFrom / ReadAll journal iteration calls.
					if call, ok := n.(*ast.CallExpr); ok {
						sel, ok := analyzer.SelectorFromExpr(call.Fun)
						if ok {
							method := sel.Sel.Name
							if method == "ReadFrom" || method == "ReadAll" {
								hasJournalLoop = true
								if reportPos == nil {
									reportPos = call
								}
							}
						}
					}

					return true
				})

				if hasCheckpointTable && hasJournalLoop && reportPos != nil {
					pos := ctx.Fset.Position(reportPos.Pos())

					f, err := finding.NewBuilder(
						"B016", toolName,
						"Manual checkpoint table + journal replay loop — "+
							"use projectionhost.Host for crash-restart lifecycle",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Replace manual checkpoint logic with projectionhost.New(journal, cpStore) — "+
							"it handles checkpoint persistence, replay, and crash-restart automatically").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}

// B017: Manual read model rebuild from scratch.
// Detects Rehydrate/Rebuild/Replay methods that load ALL events from the store
// on every startup instead of using incremental catch-up with checkpoints.
//
//nolint:ireturn // factory returns public interface
func NewB017Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B017-manual-readmodel-rebuild",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			rebuildNames := map[string]bool{
				"Rehydrate": true, "Rebuild": true, "Replay": true,
				"RebuildAll": true, "RehydrateAll": true, "ReplayAll": true,
				"RebuildReadModels": true, "RebuildProjections": true,
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Name == nil || fn.Body == nil {
						continue
					}

					if !rebuildNames[fn.Name.Name] {
						continue
					}

					// Check if the function calls ReadAll (loads ALL events).
					hasReadAll := false
					ast.Inspect(fn.Body, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := analyzer.SelectorFromExpr(call.Fun)
						if !ok {
							return true
						}

						if sel.Sel.Name == "ReadAll" {
							hasReadAll = true
							return false
						}

						return true
					})

					if !hasReadAll {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"B017", toolName,
						"Read model rebuilt from scratch on startup ("+fn.Name.Name+
							" loads ALL events) — use incremental catch-up with checkpoints",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Use projectionhost.Host with a checkpoint store — "+
							"it replays only new events on restart instead of rebuilding everything").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}
