package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects in-memory snapshot/checkpoint/DLQ stores paired with a persistent
// event store (SQLite/Postgres/Pebble). In-memory stores lose their data on
// restart, making the snapshot/checkpoint optimization useless and causing
// full projection replays every time.
//
// C017: In-memory snapshot store with persistent event store.
//
//nolint:ireturn // factory returns public interface
func NewC017Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C017-inmem-store-persistent-eventstore",
		func(_ context.Context) ([]finding.Finding, error) {
			if !isPersistentStore(ctx.FeatureProfile.Store) {
				return nil, nil
			}

			var findings []finding.Finding

			memoryStoreFns := map[string]bool{
				"NewMemorySnapshotStore":   true,
				"NewMemoryCheckpointStore": true,
				"NewMemoryDeadLetterStore": true,
				"NewMemoryTimerStore":      true,
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				// If this file also uses memory.NewMemoryStore() for the event
				// store, the setup is entirely in-memory — no mismatch.
				if fileUsesMemoryEventStore(gf.AST) {
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
					if !memoryStoreFns[fnName] {
						return true
					}

					pkg := analyzer.SelectorPackage(sel)
					if pkg != "memory" && pkg != "projectionhost" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())
					what := describeInMemStore(fnName)

					f, err := finding.NewBuilder(
						"C017",
						toolName,
						"In-memory "+what+" paired with persistent event store ("+string(
							ctx.FeatureProfile.Store,
						)+
							") — lost on restart",
						finding.SeverityError,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Use a persistent " + what +
							" (SQLite/Postgres/Pebble) matching the event store backend").
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

func isPersistentStore(s analyzer.StoreKind) bool {
	// Any non-memory, non-unknown, non-none store is considered persistent.
	// Custom stores typically wrap SQLite/Postgres and would also lose data.
	return s != analyzer.StoreMemory && s != analyzer.StoreUnknown && s != analyzer.StoreNone
}

func describeInMemStore(fnName string) string {
	switch fnName {
	case "NewMemorySnapshotStore":
		return "snapshot store"
	case "NewMemoryCheckpointStore":
		return "checkpoint store"
	case "NewMemoryDeadLetterStore":
		return "dead-letter store"
	case "NewMemoryTimerStore":
		return "timer store"
	default:
		return "store"
	}
}

// fileUsesMemoryEventStore returns true if the file contains a call to
// memory.NewMemoryStore(), indicating the event store itself is in-memory.
// In that case C017 should not fire — the entire setup is in-memory.
func fileUsesMemoryEventStore(root ast.Node) bool {
	found := false

	ast.Inspect(root, func(n ast.Node) bool {
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

		if sel.Sel.Name == "NewMemoryStore" && analyzer.SelectorPackage(sel) == "memory" {
			found = true
			return false
		}

		return true
	})

	return found
}
