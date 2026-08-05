package api

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A016: Missing idempotency middleware.
// Detects command dispatchers without idempotency middleware.
// Only flags when the dispatcher's module actually dispatches commands
// (CommandFlowCommands). Read-only systems (no dispatcher) and sync/batch
// systems are not at risk. Command-flow is evaluated per-module via
// ProfileForFile so a dispatcher in a read-only sub-module is not flagged.
//
//nolint:ireturn // factory returns public interface
func NewA016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A016-missing-idempotency-middleware",
		func(_ context.Context) ([]finding.Finding, error) {
			hasIdempotency := false
			dispFile := ""
			dispLine := 0

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

					if sel.Sel.Name == "CommandIdempotency" || sel.Sel.Name == "EventIdempotency" ||
						sel.Sel.Name == "QueryIdempotency" {
						hasIdempotency = true

						return false
					}

					// Direct idempotency.NewMemoryStore usage also counts.
					if sel.Sel.Name == "NewMemoryStore" &&
						analyzer.SelectorPackage(sel) == "idempotency" {
						hasIdempotency = true

						return false
					}

					if sel.Sel.Name == "NewDispatcher" || sel.Sel.Name == "Use" {
						if dispFile == "" {
							p := ctx.Fset.Position(call.Pos())
							dispFile = p.Filename
							dispLine = p.Line
						}
					}

					return true
				})
			}

			if hasIdempotency || dispFile == "" {
				return nil, nil
			}

			// Evaluate per-module: only flag dispatchers in modules that
			// actually dispatch commands. Using the primary profile would
			// flag a read-only sub-module when the primary module has
			// command flow.
			if ctx.ProfileForFile(dispFile).CommandFlow != analyzer.CommandFlowCommands {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"A016", toolName,
				"Command dispatcher lacks idempotency middleware — duplicate commands may execute twice",
				finding.SeverityWarning,
				finding.Pos(finding.FilePath(dispFile), dispLine, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion("Add middleware.CommandIdempotency(store, ttl, nil) to your dispatcher").
				WithSnippet(ctx.SourceLine(dispFile, dispLine)).
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// A018: No actual event sourcing.
// Detects projects that import event/ but never call store.Save or bus.Publish.
//
//nolint:ireturn // factory returns public interface
func NewA018Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A018-no-actual-event-sourcing",
		func(_ context.Context) ([]finding.Finding, error) {
			hasSaveOrPublish := false
			hasDispatch := false

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

					if sel.Sel.Name == "Save" || sel.Sel.Name == "Publish" ||
						sel.Sel.Name == "AppendBatch" {
						hasSaveOrPublish = true

						return false
					}

						if sel.Sel.Name == "Dispatch" || sel.Sel.Name == "DispatchTyped" ||
							sel.Sel.Name == "RegisterTyped" || sel.Sel.Name == "RegisterQuery" ||
							sel.Sel.Name == "NewDispatcher" {
							hasDispatch = true
						}

						return true
					})
			}

			if hasSaveOrPublish || len(ctx.Registry.Folds) > 0 {
				return nil, nil
			}

			// If command/query dispatch is actively used, the import is NOT dead —
			// the consumer is using CQRS-without-event-sourcing by design. A025
			// already covers the "consider adding event sourcing" coaching case.
			if hasDispatch {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"A018",
				toolName,
				"Project imports go-cqrs-lite but never calls Save/Publish/Dispatch — "+
					"possible dead import or missing wiring",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion("Wire up an event store and bus, or remove the unused import").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// A019: Vendored cqrs.
// Detects vendored copies of go-cqrs-lite instead of proper module imports.
//
//nolint:ireturn // factory returns public interface
func NewA019Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A019-vendored-cqrs",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, pkg := range ctx.Packages {
				for _, imp := range pkg.Imports {
					if imp == nil {
						continue
					}

					if strings.Contains(imp.PkgPath, "vendor/") &&
						strings.Contains(imp.PkgPath, "cqrs") {
						f, err := finding.NewBuilder(
							"A019", toolName,
							"Vendored copy of go-cqrs-lite detected — update lag and missing bug fixes",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceHigh).
							WithSuggestion("Remove vendor/ and use proper go.mod dependency for automatic updates").
							Build()
						if err == nil {
							findings = append(findings, f)
						}
					}
				}
			}

			return findings, nil
		},
	)
}
