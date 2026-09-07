package tursoengine_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "turso.tech/database/tursogo"
)

func TestProbeBigTxFile(t *testing.T) {
	ctx := context.Background()

	dsn := filepath.Join(t.TempDir(), "bigtx.db")
	db, err := sql.Open("turso", dsn+"?experimental=views")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `CREATE TABLE t (k TEXT PRIMARY KEY, v TEXT)`)
	if err != nil {
		t.Fatal(err)
	}

	for _, n := range []int{1000, 5000, 10000} {
		err = bigTx(ctx, db, n)
		t.Logf("n=%d err=%v", n, err)
	}
}
