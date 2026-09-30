//go:build integration

package storage_test

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// TestPostgresInitSchema_Concurrent pins the advisory-lock serialization of
// PostgreSQL schema setup (M16.2 sweep): N concurrent PostgresInitSchema calls
// on ONE shared database must all succeed. Without the advisory lock,
// concurrent CREATE TABLE IF NOT EXISTS intermittently collides in
// PostgreSQL's catalog (pg_type unique-index violations) — the class CI
// -count=2 legs would multiply.
func TestPostgresInitSchema_Concurrent(t *testing.T) {
	t.Parallel()

	db := pgDB(t)

	const workers = 8

	errs := make(chan error, workers)
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			errs <- storage.PostgresInitSchema(context.Background(), db)
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent PostgresInitSchema: %v", err)
		}
	}
}
