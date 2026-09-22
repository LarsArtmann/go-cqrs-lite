package claimkit_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // test-only: Postgres dialect leg

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
)

// pgDSN resolves the Postgres test DSN; skips unless POSTGRES_TEST_DSN is
// set (the ephemeral/integration legs export it).
func pgDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set — run via the integration leg")
	}

	return dsn
}

func newPGHost(t *testing.T) metaengine.Engine {
	t.Helper()

	db, err := sql.Open("pgx", pgDSN(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	claims, err := claimkit.New(context.Background(), db, claiming.DialectPostgres)
	if err != nil {
		t.Fatalf("claimkit.New: %v", err)
	}

	return host{db: db, Claims: claims}
}

// TestClaimKit_PostgresConformance pins the Postgres dialect (CTE FOR UPDATE
// SKIP LOCKED claim with composite-keyspace join) to the shared contract —
// including cross-collection isolation via the run-unique collection names.
func TestClaimKit_PostgresConformance(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "claimkit-postgres", Create: newPGHost}})
	adttest.AssertFactSink(t, []adttest.Factory{{Name: "claimkit-postgres", Create: newPGHost}})
}

// TestClaimKit_PostgresCrossKeyspaceIsolation pins the composite-keyspace
// join directly: claiming in one collection must never stamp or return rows
// of another collection sharing the same key.
func TestClaimKit_PostgresCrossKeyspaceIsolation(t *testing.T) {
	t.Parallel()

	eng := newPGHost(t)
	claimer := eng.(metaengine.DueClaimer)
	ctx := context.Background()

	// Run-unique collection names: reruns (-count>1) must not observe the
	// previous run's still-leased rows.
	col := fmt.Sprintf("iso_probe_%d", time.Now().UnixNano())

	for i := range 3 {
		if err := claimer.ClaimInsert(
			ctx,
			fmt.Sprintf("%s_%d", col, i),
			"shared-key",
			time.UnixMilli(1000).Add(-time.Minute),
			[]byte("x"),
		); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	got, err := claimer.ClaimDue(
		ctx,
		metaengine.ClaimDueRequest{Collection: col + "_0", Owner: "w", Now: time.UnixMilli(1000)},
	)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("cross-keyspace leak: claimed %d rows from one collection, want 1", len(got))
	}
}
