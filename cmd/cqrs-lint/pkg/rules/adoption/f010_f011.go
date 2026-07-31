package adoption

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F010 detects projects with graph-traversal patterns (recursive queries,
// ancestry, path-finding) that do not use the graph projection module.
// The graph tier provides MergeNode/MergeEdge semantics and native traversal.
//
//nolint:ireturn // factory returns public interface
func NewF010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F010-no-graph-projections",
		func(_ context.Context) ([]finding.Finding, error) {
			if importsPath(ctx, "go-cqrs-lite/graph") {
				return nil, nil
			}

			pos, ok := hasTraversalPatterns(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F010",
				"Graph-traversal patterns detected (recursive queries, ancestry, "+
					"path-finding) but graph module is not used — recursive SQL "+
					"CTEs are slow for deep traversals",
				"Import the graph module and use graph.NewGraphProjection to build "+
					"node/edge read models. The MemoryDriver supports Traverse, "+
					"Neighbors, and ShortestPath in Go-native code.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// F011 detects projects with SQL projections that do multi-table writes per
// event without using storage.RelationalProjection. The relational tier
// handles multi-table atomic writes, junction tables, and rollup counters.
//
//nolint:ireturn // factory returns public interface
func NewF011Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F011-no-relational-projections",
		func(_ context.Context) ([]finding.Finding, error) {
			if projectHasCall(ctx, "storage", "NewRelationalProjection") {
				return nil, nil
			}

			// Need SQL usage + projection/event handling + multiple Exec calls.
			if !importsPath(ctx, "database/sql") &&
				!importsPath(ctx, "go-cqrs-lite/storage") {
				return nil, nil
			}

			if eventCount(ctx) == 0 {
				return nil, nil
			}

			execCount := countSQLExec(ctx)
			if execCount < 3 {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F011",
				"Multiple SQL Exec calls in event-handling code but "+
					"storage.RelationalProjection is not used — manual "+
					"multi-table writes lack atomicity guarantees",
				"Use storage.NewRelationalProjection for multi-table atomic "+
					"writes per event. Provides Ensure, Upsert, Increment "+
					"(rollup counters), and junction-table support with "+
					"automatic transaction management.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// countSQLExec counts calls to Exec/ExecContext on *sql.DB, *sql.Tx, or
// *sql.Conn variables across non-test files. Uses type info to avoid counting
// unrelated .Exec() calls (e.g., os/exec, custom types).
func countSQLExec(ctx *analyzer.AnalysisContext) int {
	count := 0

	execMethods := map[string]bool{
		"Exec": true, "ExecContext": true,
	}

	// SQL types whose Exec/ExecContext methods we care about.
	sqlTypes := map[string]bool{
		"*database/sql.DB":   true,
		"*database/sql.Tx":   true,
		"*database/sql.Conn": true,
		"*github.com/larsartmann/go-cqrs-lite/storage/sql.DBCloser": true,
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		astInspectCalls(gf.AST, func(call *ast.CallExpr) bool {
			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if !execMethods[sel.Sel.Name] {
				return true
			}

			// Use type info to verify the receiver is a *sql.DB/Tx/Conn.
			if gf.Pkg != nil && gf.Pkg.TypesInfo != nil {
				if tv, ok := gf.Pkg.TypesInfo.Types[sel.X]; ok {
					typeStr := tv.Type.String()
					if sqlTypes[typeStr] {
						count++
					}

					return true
				}
			}

			// Fallback: no type info available (AST-only mode in unit tests).
			// Use variable-name heuristic: db, tx, conn, stmt.
			if ident, ok := sel.X.(*ast.Ident); ok {
				switch ident.Name {
				case "db", "tx", "conn", "stmt", "store":
					count++
				}
			}

			return true
		})
	}

	return count
}
