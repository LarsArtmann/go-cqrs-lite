package tursoengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "turso.tech/database/tursogo"
)

func TestProbeBigTx(t *testing.T) {
	ctx := context.Background()

	for _, n := range []int{1000, 2000, 5000, 10000} {
		db, err := sql.Open("turso", ":memory:")
		if err != nil {
			t.Fatal(err)
		}

		_, _ = db.ExecContext(ctx, `CREATE TABLE t (k TEXT PRIMARY KEY, v TEXT)`)

		err = bigTx(ctx, db, n)
		t.Logf("n=%d err=%v", n, err)
		_ = db.Close()
	}
}

func bigTx(ctx context.Context, db *sql.DB, n int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	for i := range n {
		if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO t VALUES (?, ?)`, fmt.Sprintf("k%d", i), "v"); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
