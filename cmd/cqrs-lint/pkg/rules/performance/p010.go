package performance

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// P010: No snapshot strategy on large aggregates.
// Detects decider.NewRepository/NewTypedRepository calls for state types
// that have slice or map fields (unbounded growth potential) without
// WithSnapshotStrategy and without WithStateCache. Aggregates whose state
// accumulates collections grow linearly with event count, making full-stream
// reloads increasingly expensive without snapshotting or caching.
//
//nolint:ireturn // factory returns public interface
func NewP010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P010-no-snapshot-large-aggregate",
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

					method := sel.Sel.Name
					if method != "NewRepository" && method != "NewTypedRepository" {
						return true
					}

					if analyzer.SelectorPackage(sel) != "decider" {
						return true
					}

					// Skip if snapshot strategy or state cache is configured.
					if callHasOption(call, "WithSnapshotStrategy") ||
						callHasOption(call, "WithStateCache") {
						return true
					}

					// Determine the state type to check for collection fields.
					stateType := extractStateTypeFromCall(call)
					if stateType == "" {
						return true
					}

					st := findStructType(ctx, stateType)
					if st == nil || !hasCollectionField(st) {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())
					fieldCount := structFieldCount(st)

					f, err := finding.NewBuilder(
						"P010", toolName,
						fmt.Sprintf(
							"Repository for %s (state has collection fields — "+
								"unbounded growth) without WithSnapshotStrategy or WithStateCache — "+
								"full stream reload on every command becomes expensive",
							stateType,
						),
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryPerformance).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion(fmt.Sprintf(
							"Add decider.WithSnapshotStore(snapStore) + "+
								"decider.WithSnapshotStrategy(snapshot.EveryNEvents(%d)), "+
								"or decider.WithStateCache(decider.NewStateCache[%s](256))",
							snapshotThreshold(fieldCount), stateType,
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

// snapshotThreshold returns a suggested EveryNEvents threshold based on the
// state struct's field count. More complex states benefit from more frequent
// snapshots.
func snapshotThreshold(fieldCount int) int {
	if fieldCount >= 10 {
		return 50
	}

	return 100
}
