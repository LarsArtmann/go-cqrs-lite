//go:build integration

package sqlstore_test

// Shared helpers for the Postgres and MySQL integration suites. Build-tagged
// so the always-compiled test binary carries none of the driver imports.

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"

	"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
)

// openStore opens a *sql.DB with driver/dsn, pings it, and wraps it via new
// (NewPostgresStore / NewMySQLStore — each runs its dialect DDL). The db is
// closed via t.Cleanup.
func openStore(
	t *testing.T,
	driver, dsn string,
	new func(context.Context, *sql.DB) (*sqlstore.Store, error),
) (*sql.DB, *sqlstore.Store) {
	t.Helper()

	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatalf("open %s: %v", driver, err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping %s: %v", driver, err)
	}

	store, err := new(ctx, db)
	if err != nil {
		t.Fatalf("create %s store: %v", driver, err)
	}

	return db, store
}

// concurrentClaimExactlyOnce fires n concurrent CheckAndRecord calls for key
// and asserts exactly one success and n-1 ErrDuplicate — the row-level
// serialization guarantee the single-statement upsert must provide on a real
// server under real connection parallelism.
func concurrentClaimExactlyOnce(t *testing.T, store *sqlstore.Store, key string, n int) {
	t.Helper()

	results := make(chan error, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()
			results <- store.CheckAndRecord(context.Background(), key, time.Minute)
		}()
	}

	wg.Wait()
	close(results)

	wins, dups := 0, 0

	for err := range results {
		switch {
		case err == nil:
			wins++
		case errors.Is(err, idempotency.ErrDuplicate):
			dups++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if wins != 1 || dups != n-1 {
		t.Fatalf(
			"atomic claim: want 1 win + %d duplicates, got %d wins + %d duplicates",
			n-1,
			wins,
			dups,
		)
	}
}

// assertIntegrationSeen asserts store.Seen(key) == want for integration tests.
func assertIntegrationSeen(t *testing.T, store *sqlstore.Store, key string, want bool) {
	t.Helper()

	seen, err := store.Seen(context.Background(), key)
	if err != nil {
		t.Fatalf("Seen(%q): %v", key, err)
	}

	if seen != want {
		t.Fatalf("Seen(%q): want %v, got %v", key, want, seen)
	}
}
