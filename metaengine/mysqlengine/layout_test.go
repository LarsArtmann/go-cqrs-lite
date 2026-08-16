package mysqlengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// mariadbVersion reports the server VERSION() string, or skips the test when
// no DSN is configured or the server is not MariaDB.
func mariadbVersion(t *testing.T) string {
	t.Helper()

	dsn := mysqlTestDSN()
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MariaDB layout test")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	var version string
	if err := db.QueryRowContext(context.Background(), "SELECT VERSION()").
		Scan(&version); err != nil {
		t.Skipf("MySQL not reachable: %v", err)
	}

	if !strings.Contains(strings.ToUpper(version), "MARIADB") {
		t.Skipf("server %q is not MariaDB — generated-column layout is MariaDB-only", version)
	}

	return version
}

// chunks splits values into fixed-size chunks for multi-VALUES inserts.
func chunks(values []string, size int) [][]string {
	var out [][]string
	for len(values) > size {
		out = append(out, values[:size])
		values = values[size:]
	}

	return append(out, values)
}

// gcColumnFor discovers the generated column meta_map gained for a JSON
// field, via its generation expression referencing '$.<field>'.
func gcColumnFor(t *testing.T, db *sql.DB, field string) string {
	t.Helper()

	const query = `SELECT column_name FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = 'meta_map'
		  AND generation_expression LIKE ?`

	var column string
	if err := db.QueryRowContext(context.Background(), query, "%'$."+field+"')%").
		Scan(&column); err != nil {
		t.Fatalf("no generated column for field %q: %v", field, err)
	}

	return column
}

// TestMariaDBApplyLayout_GeneratedColumnFilter verifies the MariaDB layout
// path end to end: ApplyLayout creates a VIRTUAL generated column plus a
// composite (collection, gc(N)) index, PushdownMapScan filters through the
// generated column, the planner actually uses the index, and semantics
// (missing fields, >prefix-length values) stay exact.
func TestMariaDBApplyLayout_GeneratedColumnFilter(t *testing.T) {
	t.Parallel()

	mariadbVersion(t)

	// Fixed name is safe under reruns: MapSet keys are overwritten, so the
	// seeded state is idempotent and the count assertions stay exact.
	collection := "layout_mariadb"

	eng := mustNewMySQLEngine(t)

	mb := eng.(metaengine.MapBackend)
	ctx := context.Background()

	docs := map[string]map[string]any{
		"open1":  {"status": "open", "priority": float64(1)},
		"open2":  {"status": "open", "priority": float64(2)},
		"done1":  {"status": "done", "priority": float64(3)},
		"gap1":   {"priority": float64(4)}, // missing status: must never match
		"long1":  {"status": "L" + strings.Repeat("o", 400) + "ng"},
		"other1": {},
	}

	for key, doc := range docs {
		if err := mb.MapSet(ctx, collection, key, doc); err != nil {
			t.Fatalf("MapSet(%s): %v", key, err)
		}
	}

	planner := eng.(metaengine.LayoutPlanner)
	if err := planner.ApplyLayout(
		collection,
		[]string{"status"},
		[]string{"priority"},
	); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	db, err := sql.Open("mysql", mysqlTestDSN())
	if err != nil {
		t.Fatalf("open raw connection: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	// Bulk-fill the collection so the index beats a PK-prefix range scan:
	// at a handful of rows the optimizer legitimately prefers PRIMARY.
	// Chunked multi-VALUES INSERT keeps statements small and portable (no
	// dependence on MariaDB's seq schema or extra grants).
	values := make([]string, 0, 5000)
	for i := range 5000 {
		values = append(values, fmt.Sprintf(`('%s', 'bulk%d', '{"status":"bulk","priority":%d}')`,
			collection, i, i%100))
	}

	for _, chunk := range chunks(values, 500) {
		stmt := "INSERT INTO meta_map (collection, `key`, value) VALUES " +
			strings.Join(chunk, ",") +
			" ON DUPLICATE KEY UPDATE value = VALUES(value)"
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("bulk seed: %v", err)
		}
	}

	gc := gcColumnFor(t, db, "status")

	if !strings.HasPrefix(gc, "gc_") {
		t.Fatalf("generated column %q lacks gc_ prefix", gc)
	}

	// The composite index must serve the engine's filter shape: EXPLAIN on
	// the predicate filterExpr renders must report a ref access on it.
	explain := fmt.Sprintf(
		"EXPLAIN SELECT CAST(value AS CHAR) FROM meta_map WHERE collection = ? AND `%s` = ?", gc)

	var (
		selectType, table, accessType, possibleKeys, keyCol, keyLen, refCol string
		id                                                                  int
		rowsEst                                                             int64
		extra                                                               sql.NullString
	)

	if err := db.QueryRowContext(ctx, explain, collection, "open").
		Scan(&id, &selectType, &table, &accessType, &possibleKeys, &keyCol, &keyLen, &refCol, &rowsEst, &extra); err != nil {
		t.Fatalf("EXPLAIN: %v", err)
	}

	if keyCol == "" || !strings.Contains(keyCol, "idx_map_gc") {
		t.Fatalf("EXPLAIN did not use the generated-column index: access=%q key=%q possible=%q",
			accessType, keyCol, possibleKeys)
	}

	// Correctness through the pushdown path (filters now hit the gc column).
	ps := eng.(metaengine.PushdownScan)

	results, err := ps.PushdownMapScan(ctx, collection,
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}},
		nil, nil, 0)
	if err != nil {
		t.Fatalf("PushdownMapScan(status=open): %v", err)
	}

	if len(results.Items) != 2 {
		t.Fatalf("status=open matched %d items, want 2 (open1, open2)", len(results.Items))
	}

	// A value longer than both the VARCHAR hazard zone (255) and the index
	// prefix (190) must still match exactly — TEXT column, no truncation.
	long := docs["long1"]["status"].(string)

	results, err = ps.PushdownMapScan(ctx, collection,
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: long}},
		nil, nil, 0)
	if err != nil {
		t.Fatalf("PushdownMapScan(status=long): %v", err)
	}

	if len(results.Items) != 1 {
		t.Fatalf("long-value status matched %d items, want 1 (long1)", len(results.Items))
	}

	// Idempotency: re-applying the same layout must not error.
	if err := planner.ApplyLayout(
		collection,
		[]string{"status"},
		[]string{"priority"},
	); err != nil {
		t.Fatalf("ApplyLayout re-apply: %v", err)
	}
}
