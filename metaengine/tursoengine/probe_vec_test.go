package tursoengine_test

import (
	"context"
	"database/sql"
	"testing"
)

// Temporary probe: which vector SQL does embedded libSQL (turso driver) support?
func TestProbeLibSQLVectorFunctions(t *testing.T) {
	db, err := sql.Open("turso", ":memory:")
	if err != nil {
		t.Skipf("turso driver unavailable: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	for _, q := range []string{
		`SELECT vector_distance_cos('[1,0]', '[0,1]')`,
		`SELECT vector_distance_l2('[1,0]', '[0,1]')`,
		`SELECT vector_distance_l1('[1,0]', '[0,1]')`,
		`SELECT vector_distance_dot('[1,0]', '[0,1]')`,
		`SELECT vector32('[1,2]')`,
		`SELECT vector_top_k('nope', '[1]', 1)`,
	} {
		var out any
		err := db.QueryRowContext(ctx, q).Scan(&out)
		t.Logf("q=%q err=%v out=%v", q, err, out)
	}

	// F32_BLOB column + parameter binding + top-k pushdown shape.
	stmts := []string{
		`CREATE TABLE probe_vec (id TEXT PRIMARY KEY, v F32_BLOB)`,
		`INSERT INTO probe_vec VALUES ('a', vector32('[1,0]'))`,
		`INSERT INTO probe_vec VALUES ('b', vector32('[0,1]'))`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}

	for _, q := range []string{
		`SELECT id, vector_distance_cos(v, vector32('[1,0]')) AS d FROM probe_vec ORDER BY d LIMIT 1`,
		`SELECT id FROM probe_vec ORDER BY vector_distance_l2(v, vector32('[1,0]')) LIMIT 1`,
	} {
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			t.Logf("q=%q err=%v", q, err)
			continue
		}
		for rows.Next() {
			cols, _ := rows.Columns()
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err == nil {
				t.Logf("q=%q row=%v", q, vals)
			}
		}
		_ = rows.Close()
	}

	// Binding a Go string parameter into vector32().
	var d any
	err = db.QueryRowContext(ctx,
		`SELECT vector_distance_cos(v, vector32(?)) FROM probe_vec WHERE id='a'`, "[1,0]").Scan(&d)
	t.Logf("param-bind err=%v out=%v", err, d)

	// Storing a binary blob into F32_BLOB (little-endian float32s, no marker).
	if _, err := db.ExecContext(ctx,
		`INSERT INTO probe_vec VALUES ('c', vector32(?))`, "[0.5,0.5]"); err != nil {
		t.Logf("insert-param err=%v", err)
	}
}
