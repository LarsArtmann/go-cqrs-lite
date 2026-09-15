//go:build cgo

package duckdbengine_test

import (
	"context"
	"database/sql"
	"testing"
)

// Temporary probe: DuckDB core array distance functions + casts + param binding.
func TestProbeDuckDBArrayFunctions(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Skipf("duckdb driver unavailable: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	for _, q := range []string{
		`SELECT array_cosine_similarity([1.0,0.0]::FLOAT[2], [0.0,1.0]::FLOAT[2])`,
		`SELECT array_distance([1.0,0.0]::FLOAT[2], [0.0,1.0]::FLOAT[2])`,
		`SELECT array_inner_product([1.0,2.0]::FLOAT[2], [3.0,4.0]::FLOAT[2])`,
		`SELECT list_cosine_similarity([1.0,0.0], [0.0,1.0])`,
		`SELECT '[1.0, 2.0, 3.0]'::FLOAT[3]`,
		`SELECT '[1.0, 2.0, 3.0]'::FLOAT[]`,
	} {
		var out any
		err := db.QueryRowContext(ctx, q).Scan(&out)
		t.Logf("q=%q err=%v out=%v (%T)", q, err, out, out)
	}

	// Top-k shape: store FLOAT[] column, cast string params, ORDER BY + LIMIT.
	stmts := []string{
		`CREATE TABLE probe_vec (id VARCHAR PRIMARY KEY, v FLOAT[2])`,
		`INSERT INTO probe_vec VALUES ('a', [1.0,0.0]::FLOAT[2])`,
		`INSERT INTO probe_vec VALUES ('b', [0.0,1.0]::FLOAT[2])`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}

	var id string
	var dist any
	err = db.QueryRowContext(ctx,
		`SELECT id, array_distance(v, ?::FLOAT[2]) AS d FROM probe_vec ORDER BY d LIMIT 1`,
		"[1.0,0.0]").Scan(&id, &dist)
	t.Logf("topk-param err=%v id=%v dist=%v", err, id, dist)

	err = db.QueryRowContext(ctx,
		`SELECT id, 1 - array_cosine_similarity(v, ?::FLOAT[2]) AS d FROM probe_vec ORDER BY d LIMIT 1`,
		"[1.0,0.0]").Scan(&id, &dist)
	t.Logf("cosine-param err=%v id=%v dist=%v", err, id, dist)
}
