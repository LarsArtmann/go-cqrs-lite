package sqliteengine_test

import (
	"context"
	"database/sql"
	"testing"
)

// Temporary probe: does modernc.org/sqlite know libSQL vector functions?
func TestProbeVectorFunctions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	for _, q := range []string{
		`SELECT vector_distance_cos('[1,0]', '[0,1]')`,
		`SELECT vector32('[1,2]')`,
	} {
		var out any
		err := db.QueryRowContext(context.Background(), q).Scan(&out)
		t.Logf("q=%q err=%v out=%v", q, err, out)
	}
}
