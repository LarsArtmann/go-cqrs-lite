package api

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// A016: Missing idempotency middleware.
// Detects command dispatchers without idempotency middleware.
// Only flags when the dispatcher's module actually dispatches commands
// (CommandFlowCommands). Read-only systems (no dispatcher) and sync/batch
// systems are not at risk. Only go-cqrs-lite dispatcher wiring counts (see
// a016_dispatcher.go). Command-flow is evaluated per-module via
// ProfileForFile, one finding per module that dispatches commands, so a
// dispatcher in a read-only sub-module is not flagged and a real one
// elsewhere is not hidden by it.
//
//nolint:ireturn // factory returns public interface
func NewA016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A016-missing-idempotency-middleware",
		func(_ context.Context) ([]finding.Finding, error) {
			hasIdempotency := false
			dispatchers := newDispatcherSites()

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

					if hasIdempotencySignal(sel) {
						hasIdempotency = true

						return false
					}

					// `.Use` is also a router/logger method: only go-cqrs-lite
					// dispatcher wiring decides the verdict.
					if isDispatcherWiring(gf, call) {
						dispatchers.add(ctx, gf, call)
					}

					return true
				})
			}

			if hasIdempotency {
				return nil, nil
			}

			var findings []finding.Finding

			for _, site := range dispatchers.sites {
				if ctx.ProfileForFile(site.pos.Filename).CommandFlow != analyzer.CommandFlowCommands {
					continue
				}

				f, err := a016Finding(ctx, site)
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}

// hasIdempotencySignal reports whether sel is a library idempotency
// middleware constructor or a direct idempotency.NewMemoryStore call.
func hasIdempotencySignal(sel *ast.SelectorExpr) bool {
	switch sel.Sel.Name {
	case "CommandIdempotency", "EventIdempotency", "QueryIdempotency":
		return true
	case "NewMemoryStore":
		return analyzer.SelectorPackage(sel) == "idempotency"
	default:
		return false
	}
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

			f, err := findingTemplate.Builder(
				"A018",
				"Project imports go-cqrs-lite but never calls Save/Publish/Dispatch — "+
					"possible dead import or missing wiring",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceHigh).
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
						f, err := findingTemplate.Builder(
							"A019",
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
