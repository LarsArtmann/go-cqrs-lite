package tursoengine_test

import (
	"context"
	"database/sql"
	"testing"
)

func probeOpen(tb testing.TB, dsn string) *sql.DB {
	tb.Helper()

	db, err := sql.Open("turso", dsn)
	if err != nil {
		tb.Fatalf("open %q: %v", dsn, err)
	}

	tb.Cleanup(func() { _ = db.Close() })

	return db
}

func TestProbeMatView(t *testing.T) {
	ctx := context.Background()
	db := probeOpen(t, ":memory:?experimental=views")

	_, err := db.ExecContext(ctx, `CREATE TABLE meta_map (collection TEXT NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL, PRIMARY KEY (collection, key))`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = db.ExecContext(ctx, `CREATE MATERIALIZED VIEW IF NOT EXISTS cqrs_mv_orders AS
SELECT json_extract(value, '$.customer') AS grp,
       SUM(json_extract(value, '$.amount')) AS agg,
       COUNT(json_extract(value, '$.amount')) AS cnt
FROM meta_map WHERE collection = 'orders'
GROUP BY json_extract(value, '$.customer')`)
	if err != nil {
		t.Fatalf("create matview: %v", err)
	}

	seed := []string{
		`INSERT INTO meta_map VALUES ('orders', 'k1', '{"customer":"alice","amount":10.0}')`,
		`INSERT INTO meta_map VALUES ('orders', 'k2', '{"customer":"alice","amount":20.0}')`,
		`INSERT INTO meta_map VALUES ('orders', 'k3', '{"customer":"bob","amount":5.0}')`,
	}
	for _, stmt := range seed {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	rows, err := db.QueryContext(ctx, `SELECT grp, agg, cnt FROM cqrs_mv_orders ORDER BY grp`)
	if err != nil {
		t.Fatalf("query matview: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var grp string
		var agg float64
		var cnt int64
		if err := rows.Scan(&grp, &agg, &cnt); err != nil {
			t.Fatalf("scan: %v", err)
		}
		t.Logf("group=%s sum=%v count=%d", grp, agg, cnt)
	}

	// IVM: insert after matview creation must be reflected.
	if _, err := db.ExecContext(ctx, `INSERT INTO meta_map VALUES ('orders', 'k4', '{"customer":"bob","amount":7.0}')`); err != nil {
		t.Fatalf("post-create insert: %v", err)
	}

	var bobSum float64
	if err := db.QueryRowContext(ctx, `SELECT agg FROM cqrs_mv_orders WHERE grp = 'bob'`).Scan(&bobSum); err != nil {
		t.Fatalf("ivm check: %v", err)
	}
	t.Logf("bob sum after IVM insert = %v (want 12)", bobSum)

	// UPDATE + DELETE propagation.
	if _, err := db.ExecContext(ctx, `UPDATE meta_map SET value = '{"customer":"bob","amount":100.0}' WHERE key = 'k4'`); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT agg FROM cqrs_mv_orders WHERE grp = 'bob'`).Scan(&bobSum); err != nil {
		t.Fatalf("ivm update check: %v", err)
	}
	t.Logf("bob sum after IVM update = %v (want 105)", bobSum)

	if _, err := db.ExecContext(ctx, `DELETE FROM meta_map WHERE key = 'k4'`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT agg FROM cqrs_mv_orders WHERE grp = 'bob'`).Scan(&bobSum); err != nil {
		t.Fatalf("ivm delete check: %v", err)
	}
	t.Logf("bob sum after IVM delete = %v (want 5)", bobSum)
}

func TestProbeMatViewWithoutFlag(t *testing.T) {
	ctx := context.Background()
	db := probeOpen(t, ":memory:")

	_, err := db.ExecContext(ctx, `CREATE MATERIALIZED VIEW mv AS SELECT 1 AS one`)
	if err != nil {
		t.Logf("expected failure without experimental flag: %v", err)
	} else {
		t.Log("UNEXPECTED: matview created without flag")
	}
}
