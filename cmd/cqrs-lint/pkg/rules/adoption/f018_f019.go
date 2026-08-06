package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F018 detects projects using metaengine.FilterOn (closure-based filter)
// where metaengine.FilterOnField (declarative) enables SQL pushdown.
// Closure-based filters force in-memory evaluation; declarative filters
// enable WHERE-clause pushdown for O(logN) instead of O(N) scans.
//
//nolint:ireturn // factory returns public interface
func NewF018Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F018-metaengine-filteron-pushdown",
		func(_ context.Context) ([]finding.Finding, error) {
			if !usesMetaengine(ctx) {
				return nil, nil
			}

			pos, ok := firstCallPos(ctx, "metaengine", "FilterOn")
			if !ok {
				return nil, nil
			}

			suggestion := "Use metaengine.FilterOnField for declarative filters that enable " +
				"WHERE-clause pushdown (O(logN) indexed lookup instead of O(N) scan). " +
				"FilterOnField accepts a column name and comparison operator, " +
				"allowing the SQLite/Postgres engine to use json_extract() indexes."

			if projectHasCall(ctx, "metaengine", "FilterOnField") {
				return singleInfoFinding(
					ctx,
					"F018",
					"mixed metaengine usage: FilterOnField (pushdown) and FilterOn (closure) "+
						"— the FilterOn call still forces in-memory filtering for that query",
					suggestion, pos, finding.ConfidenceLow,
				), nil
			}

			return singleInfoFinding(
				ctx,
				"F018",
				"metaengine.FilterOn uses closure-based filtering which prevents "+
					"SQL pushdown — queries scan all rows and filter in Go memory",
				suggestion, pos, finding.ConfidenceMedium,
			), nil
		},
	)
}

// F019 detects metaengine query declarations that lack a Volume hint.
// The Volume hint tells the cost-based planner how many rows to expect,
// enabling better engine selection (memory vs SQLite) and layout planning.
// Without it, the planner defaults to conservative estimates.
//
//nolint:ireturn // factory returns public interface
func NewF019Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F019-metaengine-missing-volume-hint",
		func(_ context.Context) ([]finding.Finding, error) {
			if !usesMetaengine(ctx) {
				return nil, nil
			}

			if !projectHasCall(ctx, "metaengine", "Query") &&
				!projectHasCall(ctx, "metaengine", "On") &&
				!projectHasCall(ctx, "metaengine", "OnTyped") {
				return nil, nil
			}

			if projectHasCall(ctx, "metaengine", "Volume") {
				return nil, nil
			}

			pos, ok := firstCallPos(ctx, "metaengine", "Query")
			if !ok {
				pos, ok = firstCallPos(ctx, "metaengine", "On")
			}

			if !ok {
				pos, ok = firstFilePos(ctx)
				if !ok {
					return nil, nil
				}
			}

			return singleInfoFinding(
				ctx,
				"F019",
				"metaengine query declarations lack a Volume hint — the cost-based "+
					"planner cannot distinguish high-volume collections from point-lookups",
				"Add .Volume(N) to query declarations (e.g. Volume(100000) for a "+
					"large collection, Volume(100) for a small one). This enables the "+
					"planner to choose memory engines for small collections and SQLite "+
					"for large ones, and to prioritize index creation.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}
