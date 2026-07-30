package api

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// projectImportsModule returns true if any non-test file in the project imports
// a go-cqrs-lite module matching the given suffix (e.g., "event", "decider",
// "command", "query", "watermill").
func projectImportsModule(ctx *analyzer.AnalysisContext, moduleSuffix string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}

			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "go-cqrs-lite/"+moduleSuffix) {
				return true
			}
		}
	}

	return false
}

// projectCallsFunction returns true if any non-test file in the project contains
// a call to a function with the given name (any package qualifier).
func projectCallsFunction(ctx *analyzer.AnalysisContext, fnName string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if sel.Sel.Name == fnName {
				found = true
				return false
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// A024: Decorative event sourcing (decider shape, no wiring).
// Detects projects that import event/ and decider/ but never create events
// (event.New, event.NewEvent) or wire a repository (decider.NewRepository,
// decider.NewTypedRepository). The decider pattern is imported but the
// pipeline is never connected — "decorative" event sourcing.
//
//nolint:ireturn // factory returns public interface
func NewA024Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A024-decorative-event-sourcing",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectImportsModule(ctx, "event") || !projectImportsModule(ctx, "decider") {
				return nil, nil
			}

			hasNewEvent := projectCallsFunction(ctx, "New") || projectCallsFunction(ctx, "NewEvent")
			hasRepository := projectCallsFunction(ctx, "NewRepository") ||
				projectCallsFunction(ctx, "NewTypedRepository")

			if hasNewEvent || hasRepository {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"A024", toolName,
				"Project imports event/ and decider/ but never creates events or wires a repository — decorative event sourcing",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceHigh).
				WithSuggestion("Wire up an event store, bus, and decider.Repository to complete the event sourcing pipeline, or remove the unused imports").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// A025: Command/query only, no events.
// Detects projects that import command and query but have no event sourcing,
// no decider, and no event store. The project uses CQRS dispatchers as a thin
// service layer without the event sourcing half.
//
// This may be intentional (CQRS without ES), so the finding is informational.
//
//nolint:ireturn // factory returns public interface
func NewA025Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A025-command-query-no-events",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectImportsModule(ctx, "command") || !projectImportsModule(ctx, "query") {
				return nil, nil
			}

			if projectImportsModule(ctx, "event") || projectImportsModule(ctx, "decider") {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"A025", toolName,
				"Project uses command/query dispatchers but has no event sourcing — CQRS without event sourcing may miss audit trail and replay capabilities",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion("Consider adding event sourcing (event/ + decider/) for audit trail, replay, and temporal queries, or keep as-is if CQRS-without-ES is intentional").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// A026: Event bus only, no CQRS pipeline.
// Detects projects that import event and watermill but have no command
// dispatcher, decider, or query handler. The project uses go-cqrs-lite as a
// bare event bus without the CQRS separation.
//
//nolint:ireturn // factory returns public interface
func NewA026Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A026-event-bus-only",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectImportsModule(ctx, "event") || !projectImportsModule(ctx, "watermill") {
				return nil, nil
			}

			if projectImportsModule(ctx, "command") || projectImportsModule(ctx, "decider") ||
				projectImportsModule(ctx, "query") {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"A026", toolName,
				"Project uses event/ and watermill but has no command, decider, or query — using go-cqrs-lite as a bare event bus without CQRS separation",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion("Consider adding command/query separation (command/ + query/ + decider/) for a fuller CQRS architecture, or keep as-is if event bus is the intended use").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
