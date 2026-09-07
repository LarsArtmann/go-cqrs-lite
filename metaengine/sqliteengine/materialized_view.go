package sqliteengine

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// EngineOption configures optional sqliteEngine capabilities at construction.
type EngineOption func(*sqliteEngine)

// matView pairs a validated operator spec with its derived view name.
type matView struct {
	spec metaengine.MaterializedViewSpec
	name string
}

// WithMaterializedViews registers operator-declared materialized views
// (metaengine.MaterializedViewSpec). NewSQLiteEngine derives each view's DDL
// and executes CREATE MATERIALIZED VIEW IF NOT EXISTS after the base tables
// are created, so construction fails loudly when the underlying server cannot
// maintain materialized views (plain SQLite lacks the feature — Turso/libSQL
// with the "views" experimental feature provides it).
//
// The views live over meta_map for unplanned collections. If a collection is
// later migrated to a planned table (ApplyLayout), aggregate queries fall
// through to the base path — the view over meta_map would otherwise go stale.
func WithMaterializedViews(specs []metaengine.MaterializedViewSpec) EngineOption {
	return func(e *sqliteEngine) {
		for _, spec := range specs {
			if err := spec.Validate(); err != nil {
				e.matViewErr = err

				return
			}

			e.matViews = append(e.matViews, matView{spec: spec, name: spec.ViewName()})
		}
	}
}

// createMatViews executes the derived DDL for every registered spec.
func (e *sqliteEngine) createMatViews(ctx context.Context) error {
	if e.matViewErr != nil {
		return fmt.Errorf("metaengine: materialized view spec: %w", e.matViewErr)
	}

	for _, mv := range e.matViews {
		ddl := matViewDDL(mv.spec)

		if _, err := e.db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf(
				"metaengine: create materialized view %s: %w (materialized views require Turso/libSQL with the \"views\" experimental feature, e.g. DSN \"file:app.db?experimental=views\" or server flag --experimental-views)",
				mv.name, err)
		}
	}

	return nil
}

// matViewDDL derives the Turso/libSQL DDL for a spec. The view reads meta_map
// filtered to the spec's collection; AVG views store SUM and COUNT columns so
// per-group and scalar averages stay exact (sum-of-sums / sum-of-counts).
func matViewDDL(spec metaengine.MaterializedViewSpec) string {
	coll := strings.ReplaceAll(spec.Collection, "'", "''")

	var aggPart string

	switch spec.Fn {
	case metaengine.MatViewCount:
		aggPart = "COUNT(*) AS agg"
	case metaengine.MatViewAvg:
		expr := fmt.Sprintf("json_extract(value, '%s')", jsonPath(spec.Column))
		aggPart = fmt.Sprintf("SUM(%s) AS agg, COUNT(%s) AS cnt", expr, expr)
	default:
		aggPart = fmt.Sprintf("%s(json_extract(value, '%s')) AS agg", spec.Fn, jsonPath(spec.Column))
	}

	var b strings.Builder

	fmt.Fprintf(&b, "CREATE MATERIALIZED VIEW IF NOT EXISTS %s AS SELECT ", metaengine.QuoteIdent(spec.ViewName()))

	if spec.GroupBy != "" {
		grpExpr := fmt.Sprintf("json_extract(value, '%s')", jsonPath(spec.GroupBy))
		fmt.Fprintf(&b, "%s AS grp, %s FROM meta_map WHERE collection = '%s' GROUP BY grp", grpExpr, aggPart, coll)

		return b.String()
	}

	fmt.Fprintf(&b, "%s FROM meta_map WHERE collection = '%s'", aggPart, coll)

	return b.String()
}

// matViewExact returns the view whose spec matches (col, fn, column, groupBy)
// exactly, provided serving is possible: the collection must still be
// unplanned (views over meta_map go stale once a collection migrates to a
// planned table) and the query must be unfiltered (views do not carry the
// base rows needed for filtered aggregates).
func (e *sqliteEngine) matViewExact(col string, fn metaengine.AggregateFn, column, groupBy string) *matView {
	if len(e.matViews) == 0 {
		return nil
	}

	if _, planned := e.plans[col]; planned {
		return nil
	}

	for i := range e.matViews {
		mv := &e.matViews[i]
		if mv.spec.Collection == col && mv.spec.Fn == fn && mv.spec.Column == column && mv.spec.GroupBy == groupBy {
			return mv
		}
	}

	return nil
}

// matViewGrouped returns a grouped view accelerating (col, fn, column) for any
// group-by field, enabling exact scalar derivations (e.g. scalar SUM of a
// grouped SUM view equals SUM over the group sums).
func (e *sqliteEngine) matViewGrouped(col string, fn metaengine.AggregateFn, column string) *matView {
	if len(e.matViews) == 0 {
		return nil
	}

	if _, planned := e.plans[col]; planned {
		return nil
	}

	for i := range e.matViews {
		mv := &e.matViews[i]
		if mv.spec.Collection == col && mv.spec.Fn == fn && mv.spec.Column == column && mv.spec.GroupBy != "" {
			return mv
		}
	}

	return nil
}

// matViewScalarAgg serves an unfiltered scalar aggregate from its own scalar
// view. AVG views store SUM and COUNT columns; the average is the quotient.
func (e *sqliteEngine) matViewScalarAgg(ctx context.Context, mv *matView) (float64, error) {
	selectExpr := "agg"

	if mv.spec.Fn == metaengine.MatViewAvg {
		selectExpr = "agg / cnt"
	}

	var raw any

	query := fmt.Sprintf("SELECT %s FROM %s", selectExpr, metaengine.QuoteIdent(mv.name))

	if err := e.xd().QueryRowContext(ctx, query).Scan(&raw); err != nil {
		return 0, fmt.Errorf("matview aggregate %s(%s): %w", mv.spec.Fn, mv.spec.Column, err)
	}

	return metaengine.DecodeFloat(raw)
}

// matViewScalarViaGrouped serves an unfiltered scalar aggregate from a grouped
// view by algebraic derivation. Each derivation is exact over NULL handling:
// aggregates ignore NULLs on both sides.
func (e *sqliteEngine) matViewScalarViaGrouped(ctx context.Context, mv *matView) (float64, error) {
	var expr string

	switch mv.spec.Fn {
	case metaengine.MatViewAvg:
		expr = "SUM(agg) / SUM(cnt)"
	case metaengine.MatViewCount, metaengine.MatViewSum:
		expr = "SUM(agg)"
	case metaengine.MatViewMin:
		expr = "MIN(agg)"
	case metaengine.MatViewMax:
		expr = "MAX(agg)"
	}

	var raw any

	query := fmt.Sprintf("SELECT %s FROM %s", expr, metaengine.QuoteIdent(mv.name))

	if err := e.xd().QueryRowContext(ctx, query).Scan(&raw); err != nil {
		return 0, fmt.Errorf("matview scalar derivation %s(%s): %w", mv.spec.Fn, mv.spec.Column, err)
	}

	return metaengine.DecodeFloat(raw)
}

// matViewGroupedAgg serves an unfiltered grouped aggregate from its exact
// grouped view. AVG views divide the per-group SUM by the per-group COUNT.
func (e *sqliteEngine) matViewGroupedAgg(ctx context.Context, mv *matView) (map[string]float64, error) {
	selectExpr := "grp, agg"

	if mv.spec.Fn == metaengine.MatViewAvg {
		selectExpr = "grp, agg / cnt"
	}

	query := fmt.Sprintf("SELECT %s FROM %s", selectExpr, metaengine.QuoteIdent(mv.name))

	result, err := e.scanGroupedSQLite(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("matview grouped %s(%s by %s): %w",
			mv.spec.Fn, mv.spec.Column, mv.spec.GroupBy, err)
	}

	return result, nil
}

// serveScalarMatView returns the view-accelerated read for an unfiltered
// scalar aggregate, or nil when no view applies (caller falls through).
func (e *sqliteEngine) serveScalarMatView(
	ctx context.Context,
	col string,
	fn metaengine.AggregateFn,
	column string,
) (float64, bool, error) {
	if mv := e.matViewExact(col, fn, column, ""); mv != nil {
		v, err := e.matViewScalarAgg(ctx, mv)

		return v, true, err
	}

	if gv := e.matViewGrouped(col, fn, column); gv != nil {
		v, err := e.matViewScalarViaGrouped(ctx, gv)

		return v, true, err
	}

	return 0, false, nil
}

// matViewExplainSQL returns the SQL the serving path would run, keeping
// EXPLAIN honest about view-accelerated reads.
func (e *sqliteEngine) matViewExplainSQL(fn metaengine.AggregateFn, mv *matView, grouped bool) string {
	if !grouped {
		if mv.spec.Fn == metaengine.MatViewAvg {
			return fmt.Sprintf("SELECT agg / cnt FROM %s", metaengine.QuoteIdent(mv.name))
		}

		return "SELECT agg FROM " + metaengine.QuoteIdent(mv.name)
	}

	if fn == metaengine.MatViewAvg {
		return fmt.Sprintf("SELECT grp, agg / cnt FROM %s", metaengine.QuoteIdent(mv.name))
	}

	return fmt.Sprintf("SELECT grp, agg FROM %s", metaengine.QuoteIdent(mv.name))
}

// MaterializedViews implements metaengine.MaterializedViewsReporter.
func (e *sqliteEngine) MaterializedViews() []metaengine.MaterializedViewInfo {
	if len(e.matViews) == 0 {
		return nil
	}

	out := make([]metaengine.MaterializedViewInfo, 0, len(e.matViews))

	for _, mv := range e.matViews {
		rows := int64(-1)

		var count any

		query := "SELECT COUNT(*) FROM " + metaengine.QuoteIdent(mv.name)

		if err := e.db.QueryRow(query).Scan(&count); err == nil {
			if f, convErr := metaengine.DecodeFloat(count); convErr == nil {
				rows = int64(f)
			}
		}

		out = append(out, metaengine.MaterializedViewInfo{Spec: mv.spec, Name: mv.name, Rows: rows})
	}

	return out
}
