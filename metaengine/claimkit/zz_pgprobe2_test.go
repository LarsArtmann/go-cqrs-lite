package claimkit_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPGProbeRaw(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://cqrs@127.0.0.1:39095/cqrs_test?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	// fresh isolated collection
	col := fmt.Sprintf("rawprobe_%d", timeNowNano())
	if _, err := db.ExecContext(ctx,
		`INSERT INTO meta_due_claims (collection, key, due_at, payload) VALUES ($1,'k1',NOW() - INTERVAL '1 hour','x') ON CONFLICT DO NOTHING`, col); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO meta_due_claims (collection, key, due_at, payload) VALUES ('other_'+$1,'k1',NOW() - INTERVAL '1 hour','x') ON CONFLICT DO NOTHING`, col); err != nil {
		t.Fatalf("seed2: %v", err)
	}

	query := `WITH due AS (
SELECT key FROM meta_due_claims
WHERE due_at <= $1 AND (lease_until IS NULL OR lease_until <= $1) AND collection = $4
ORDER BY due_at ASC, key ASC LIMIT $5
FOR UPDATE SKIP LOCKED
)
UPDATE meta_due_claims t SET lease_until = $2, owner = $3 FROM due WHERE t.key = due.key
RETURNING t.key, t.due_at, t.lease_until, t.payload`

	rows, err := db.QueryContext(ctx, query,
		timeNowPlus(0), timeNowPlus(60_000_000_000), "w", col, 10)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		n++
	}
	t.Logf("raw claim rows=%d (want 1)", n)
}
