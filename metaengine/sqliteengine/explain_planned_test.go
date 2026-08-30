package sqliteengine_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestExplainScanQuery_PlannedUsesIndex pins the planned-path proof pattern on
// SQLite, uniform with the live pg/mysql EXPLAIN proofs: ExplainScanQuery must
// route through the planned builder (targeting meta_planned_*, never meta_map)
// and the emitted query must be index-backed under EXPLAIN QUERY PLAN — a bare
// SCAN of the planned table would mean index-name drift silently killed the
// planned-path performance contract.
func TestExplainScanQuery_PlannedUsesIndex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	defer eng.Close()

	applier, ok := eng.(metaengine.LayoutPlanApplier)
	if !ok {
		t.Fatal("sqliteEngine does not implement LayoutPlanApplier")
	}

	plan := metaengine.BuildLayoutPlanFromType[pushItem](
		"proof_items", []string{"status"}, []string{"price"},
	)
	if err := applier.ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	explainer, ok := eng.(metaengine.ExplainableScan)
	if !ok {
		t.Fatal("sqliteEngine does not implement ExplainableScan")
	}

	query, args := explainer.ExplainScanQuery(ctx, "proof_items", metaengine.ExplainOptions{
		Filters: []metaengine.FilterSpec{
			{Column: "status", Op: metaengine.FilterEq, Value: "open"},
		},
		Sort:  &metaengine.SortSpec{Column: "price", Desc: false},
		Limit: 10,
	})

	if !strings.Contains(query, "meta_planned_") {
		t.Errorf("planned query must target meta_planned_*, got: %s", query)
	}
	if strings.Contains(query, "meta_map") {
		t.Errorf("planned query must never fall back to meta_map, got: %s", query)
	}

	rows, err := db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN: %v", err)
	}
	defer rows.Close()

	var details []string
	for rows.Next() {
		var id, parent int
		var notused, detail string

		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan plan row: %v", err)
		}

		details = append(details, detail)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	indexBacked := false
	bareScan := false

	for _, detail := range details {
		if strings.Contains(detail, "USING INDEX") ||
			strings.Contains(detail, "USING COVERING INDEX") {
			indexBacked = true
		}

		if strings.Contains(detail, "SCAN") && !strings.Contains(detail, "INDEX") {
			bareScan = true
		}
	}

	if !indexBacked {
		t.Errorf("no index-backed plan node in %v — planned indexes are not being used", details)
	}

	if bareScan {
		t.Errorf("bare table scan in %v — query plan must not degrade to a full scan", details)
	}
}
