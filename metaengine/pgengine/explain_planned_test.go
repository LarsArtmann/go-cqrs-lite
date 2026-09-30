package pgengine_test

// D3 slice 3: EXPLAIN-based index-usage proofs for the planned-table
// pushdown. The tests assert the planner serves the planned scan from an
// INDEX, not a sequential scan — proving the extracted-column layout is
// actually doing its job (not just "routing differently").

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4"
)

// explainPlannedScan runs EXPLAIN (FORMAT JSON) for the planned collection's
// pushdown scan and returns the raw plan tree.
func explainPlannedScan(
	t *testing.T,
	dsn string,
	sqlText string,
	args []any,
) []any {
	t.Helper()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer db.Close()

	rows, err := db.QueryContext(context.Background(),
		"EXPLAIN (FORMAT JSON) "+sqlText, args...)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}

	defer rows.Close()

	if !rows.Next() {
		t.Fatal("EXPLAIN returned no rows")
	}

	var raw []byte
	if err := rows.Scan(&raw); err != nil {
		t.Fatalf("scan: %v", err)
	}

	var plan []any
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatalf("unmarshal plan: %v", err)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	return plan
}

// walkPlanNodes depth-first walks the EXPLAIN JSON tree, visiting every
// plan node (each has a "Node Type" key).
func walkPlanNodes(node any, visit func(nodeType, indexName string)) {
	m, ok := node.(map[string]any)
	if !ok {
		return
	}

	nodeType, _ := m["Node Type"].(string)
	indexName, _ := m["Index Name"].(string)

	visit(nodeType, indexName)

	if child, ok := m["Plan"]; ok {
		walkPlanNodes(child, visit)
	}

	if children, ok := m["Plans"].([]any); ok {
		for _, c := range children {
			walkPlanNodes(c, visit)
		}
	}
}

// TestPgPlannedExplain_IndexUsageProofs pins the index-usage contract for
// planned pushdown scans: a selective filter and a sorted+limited scan are
// served from the extracted column's index (never a sequential scan).
func TestPgPlannedExplain_IndexUsageProofs(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	dsn := pgtestcontainer.DSN(t)

	eng := mustNewPgEngine(t)
	mb := eng.(metaengine.MapBackend)
	lpa := eng.(metaengine.LayoutPlanApplier)
	ex, ok := eng.(metaengine.ExplainableScan)
	g.Expect(ok).To(gomega.BeTrue(), "pgEngine must implement ExplainableScan")

	g.Expect(lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_explain", []string{"code"}, []string{"priority"}),
	)).To(gomega.Succeed())

	// Seed enough rows with high cardinality that the planner must choose
	// the index (a 500-row table with one matching row cannot rationally
	// seq-scan when an index exists).
	for i := 0; i < 500; i++ {
		g.Expect(mb.MapSet(
			ctx,
			"planned_explain",
			fmt.Sprintf("k%d", i),
			map[string]any{
				"priority": float64(i),
				"code":     fmt.Sprintf("code-%03d", i%200),
			},
		)).To(gomega.Succeed())
	}

	assertNoSeqScan := func(sqlText string, args []any) []string {
		t.Helper()

		plan := explainPlannedScan(t, dsn, sqlText, args)

		var nodeTypes, indexNames []string

		for _, root := range plan {
			walkPlanNodes(root, func(nodeType, indexName string) {
				nodeTypes = append(nodeTypes, nodeType)

				if indexName != "" {
					indexNames = append(indexNames, indexName)
				}
			})
		}

		for _, nt := range nodeTypes {
			g.Expect(nt).NotTo(gomega.Equal("Seq Scan"),
				"planned scan must not seq-scan\nplan nodes: %v", nodeTypes)
		}

		foundIndex := false
		for _, nt := range nodeTypes {
			if strings.Contains(nt, "Index") || strings.Contains(nt, "Bitmap") {
				foundIndex = true
			}
		}

		g.Expect(foundIndex).To(gomega.BeTrue(),
			"expected an index-backed node, got %v", nodeTypes)

		return indexNames
	}

	// Proof 1: selective filter on the indexed `code` column.
	sqlText, args := ex.ExplainScanQuery(ctx, "planned_explain",
		metaengine.ExplainOptions{
			Filters: []metaengine.FilterSpec{
				{Column: "code", Op: metaengine.FilterEq, Value: "code-007"},
			},
			Limit: 10,
		})
	indexNames := assertNoSeqScan(sqlText, args)
	g.Expect(indexNames).NotTo(gomega.BeEmpty(), "filter proof must name its index")

	// Proof 2: sorted + limited scan uses the sort column's index.
	sqlText, args = ex.ExplainScanQuery(ctx, "planned_explain",
		metaengine.ExplainOptions{
			Sort:  &metaengine.SortSpec{Column: "priority", Desc: true},
			Limit: 5,
		})
	assertNoSeqScan(sqlText, args)

	// Proof 3: the EXPLAIN reflects the planned table (not meta_map).
	g.Expect(strings.Contains(sqlText, "meta_planned_planned_explain")).To(gomega.BeTrue())
	g.Expect(strings.Contains(sqlText, "meta_map")).To(gomega.BeFalse())
}
