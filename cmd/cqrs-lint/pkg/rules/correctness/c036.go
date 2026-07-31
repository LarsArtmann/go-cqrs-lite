package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// C036: Persistent store backend mismatch.
// Detects checkpoint, snapshot, idempotency, or DLQ stores that use a
// different backend than the event store (e.g., SQLite event store with
// Pebble checkpoint store). Different backends cannot share a transaction,
// so crash-recovery guarantees break — a partial write can leave the system
// in an inconsistent state.
//
// This extends C017 (which only catches in-memory vs persistent) to also
// catch cross-backend mismatches among persistent stores.
//
//nolint:ireturn // factory returns public interface
func NewC036Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C036-store-backend-mismatch",
		func(_ context.Context) ([]finding.Finding, error) {
			eventBackend := ctx.FeatureProfile.Store
			if eventBackend == analyzer.StoreUnknown ||
				eventBackend == analyzer.StoreNone ||
				eventBackend == analyzer.StoreMemory {
				return nil, nil
			}

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

					fnName := sel.Sel.Name
					pkg := analyzer.SelectorPackage(sel)

					storeBackend := detectBackend(pkg, fnName)
					if storeBackend == "" || storeBackend == string(eventBackend) {
						return true
					}

					// Skip stack preset calls (e.g., sqlite.New) — those use the
					// bundle's backend consistently.
					if isStackPresetCall(pkg, fnName) {
						return true
					}

					what := describeMismatchStore(fnName)
					if what == "" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"C036", toolName,
						fmt.Sprintf(
							"%s uses %s backend but event store uses %s — backends cannot share a transaction, crash-recovery guarantees break",
							what, storeBackend, eventBackend,
						),
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion(fmt.Sprintf(
							"Use a %s-backed %s to match the event store backend",
							eventBackend, what,
						)).
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					lintutil.AppendBuild(&findings, f, err)

					return true
				})
			}

			return findings, nil
		},
	)
}

// detectBackend returns the backend type ("sqlite", "pebble", "postgres",
// "turso") for a store constructor call, or "" if not a store constructor.
func detectBackend(pkg, fnName string) string {
	switch {
	case pkg == "pebble" && strings.HasPrefix(fnName, "New"):
		return "pebble"
	case pkg == "storage" && strings.Contains(fnName, "SQLite"):
		return "sqlite"
	case pkg == "storage" && strings.Contains(fnName, "Postgres"):
		return "postgres"
	case strings.HasPrefix(pkg, "turso") && strings.HasPrefix(fnName, "New"):
		return "turso"
	}

	return ""
}

// isStackPresetCall returns true for stack bundle constructors that already
// use a consistent backend internally.
func isStackPresetCall(pkg, fnName string) bool {
	return pkg == "sqlite" || pkg == "postgres" || pkg == "pebble" ||
		pkg == "memory" || pkg == "turso" || pkg == "duckdb"
}

// describeMismatchStore returns a human-readable description for known
// store constructor names, or "" if not relevant.
func describeMismatchStore(fnName string) string {
	switch {
	case strings.Contains(fnName, "Checkpoint"):
		return "checkpoint store"
	case strings.Contains(fnName, "Snapshot"):
		return "snapshot store"
	case strings.Contains(fnName, "DeadLetter") || strings.Contains(fnName, "DLQ"):
		return "dead-letter store"
	case strings.Contains(fnName, "Idempotency") || strings.Contains(fnName, "Dedup"):
		return "idempotency store"
	case strings.Contains(fnName, "EventStore") || strings.Contains(fnName, "Event"):
		return "" // the event store itself — not a mismatch
	default:
		return "store"
	}
}
