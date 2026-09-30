package mysqlengine_test

// D3 slice 3: EXPLAIN-based index-usage proofs for the planned-table
// pushdown on MySQL/MariaDB. Asserts the planner serves the planned scan
// from an index (access type != ALL with a named key), never a full scan.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMySQLPlannedExplain_IndexUsageProofs pins the index-usage contract:
// the planned pushdown scan is index-backed (EXPLAIN type != ALL, key set)
// and the EXPLAIN reflects the planned table, not meta_map.
func TestMySQLPlannedExplain_IndexUsageProofs(t *testing.T) {
	//art-dupl:accept test prologue — capability-assert setup is intentionally uniform across planned-table tests
	mariadbVersion(t)

	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := mustNewMySQLEngine(t)
	mb := eng.(metaengine.MapBackend)
	lpa := eng.(metaengine.LayoutPlanApplier)
	ex, ok := eng.(metaengine.ExplainableScan)
	g.Expect(ok).To(gomega.BeTrue(), "mysqlEngine must implement ExplainableScan")

	g.Expect(lpa.ApplyLayoutPlan(
		metaengine.BuildLayoutPlan("planned_explain", []string{"code"}, []string{"priority"}),
	)).To(gomega.Succeed())

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

	// Selective filter on the indexed `code` column.
	sqlText, args := ex.ExplainScanQuery(ctx, "planned_explain",
		metaengine.ExplainOptions{
			Filters: []metaengine.FilterSpec{
				{Column: "code", Op: metaengine.FilterEq, Value: "code-007"},
			},
			Limit: 10,
		})

	g.Expect(sqlText).To(gomega.ContainSubstring("meta_planned_planned_explain"))
	g.Expect(sqlText).NotTo(gomega.ContainSubstring("meta_map"))

	db, err := sql.Open("mysql", mysqlTestDSN())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	defer db.Close()

	rows, err := db.QueryContext(ctx, "EXPLAIN "+sqlText, args...)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	defer rows.Close()

	cols, err := rows.Columns()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	typeIdx, keyIdx := -1, -1
	for i, c := range cols {
		switch c {
		case "type":
			typeIdx = i
		case "key":
			keyIdx = i
		}
	}

	g.Expect(typeIdx).To(gomega.BeNumerically(">=", 0), "EXPLAIN must report 'type'")
	g.Expect(keyIdx).To(gomega.BeNumerically(">=", 0), "EXPLAIN must report 'key'")

	foundIndex := false

	for rows.Next() {
		acc := make([]any, len(cols))
		for i := range acc {
			acc[i] = new(sql.RawBytes)
		}

		g.Expect(rows.Scan(acc...)).To(gomega.Succeed())

		accessType := string(*acc[typeIdx].(*sql.RawBytes))
		key := string(*acc[keyIdx].(*sql.RawBytes))

		g.Expect(accessType).NotTo(gomega.Equal("ALL"),
			"planned scan must not full-scan (EXPLAIN type=ALL)")
		g.Expect(key).NotTo(gomega.BeEmpty(),
			"planned scan must name its index")

		foundIndex = true
	}

	g.Expect(rows.Err()).NotTo(gomega.HaveOccurred())
	g.Expect(foundIndex).To(gomega.BeTrue(), "EXPLAIN returned no rows")
}
