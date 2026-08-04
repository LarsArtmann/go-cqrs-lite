package correctness

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects in-memory snapshot/checkpoint/DLQ stores paired with a persistent
// event store (SQLite/Postgres/Pebble). In-memory stores lose their data on
// restart, making the snapshot/checkpoint optimization useless and causing
// full projection replays every time.
//
// C017: In-memory snapshot/checkpoint/dead-letter/timer store with persistent
// event store.
//
//nolint:ireturn // factory returns public interface
func NewC017Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C017-inmem-store-persistent-eventstore",
		func(_ context.Context) ([]finding.Finding, error) {
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

				// Evaluate the store backend per-module: an in-memory store is
				// only a problem when THIS file's module uses a persistent event
				// store. Using the primary profile would miss a persistent
				// sub-module or, conversely, flag files in a purely in-memory
				// example module.
				profile := ctx.ProfileForFile(gf.Path)
				if !isPersistentStore(profile.Store) {
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
					if pkg != "memory" && pkg != "projectionhost" && pkg != "scheduling" {
						return true
					}

					// Skip if the enclosing function also creates a memory
					// event store — the entire setup is in-memory within
					// that function scope.
					if enclosingFunctionUsesMemoryStore(gf.AST, call.Pos()) {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())
					what := describeInMemStore(fnName)

					f, err := finding.NewBuilder(
						"C017",
						toolName,
						"In-memory "+what+" paired with persistent event store ("+string(
							profile.Store,
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

// enclosingFunctionUsesMemoryStore finds the innermost function enclosing
// pos and checks whether it also creates a memory event store
// (memory.NewMemoryStore). This replaces the old file-level heuristic
// (fileUsesMemoryEventStore) which skipped entire files even when only one
// function used in-memory setup while another used a persistent store.
func enclosingFunctionUsesMemoryStore(root ast.Node, pos token.Pos) bool {
	var funcNode ast.Node

	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		if n.Pos() <= pos && n.End() >= pos {
			switch n.(type) {
			case *ast.FuncDecl, *ast.FuncLit:
				funcNode = n
			}
			return true
		}

		return false
	})

	if funcNode == nil {
		return false
	}

	found := false

	ast.Inspect(funcNode, func(n ast.Node) bool {
		if found || n == nil {
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
