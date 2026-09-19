package claimkit_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPGProbeTwoCollections(t *testing.T) {
	dsn := "postgres://cqrs@127.0.0.1:39095/cqrs_test?sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	c, err := claimkit.New(ctx, db, claiming.DialectPostgres)
	if err != nil {
		t.Fatal(err)
	}

	now := time.UnixMilli(1000)
	for i := range 3 {
		if err := c.ClaimInsert(ctx, fmt.Sprintf("probeA_%d", i), "t1", now.Add(-time.Minute), []byte("a")); err != nil {
			t.Fatal(err)
		}
		if err := c.ClaimInsert(ctx, fmt.Sprintf("probeB_%d", i), "t1", now.Add(-time.Minute), []byte("b")); err != nil {
			t.Fatal(err)
		}
	}

	got, err := c.ClaimDue(ctx, metaengine.ClaimDueRequest{Collection: "probeA_0", Owner: "w", Now: now})
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	t.Logf("claimed %d from probeA_0", len(got))
	if len(got) != 1 {
		t.Fatalf("collection filter broken: got %d, want 1", len(got))
	}
}
