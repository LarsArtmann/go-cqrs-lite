package pgengine_test

import (
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
)

// TestNew_ConcurrentConstruction pins the advisory-lock serialization of
// pgengine schema setup (M16.2 sweep): N concurrent pgengine.New calls on ONE
// shared database must all succeed. Engine construction eagerly runs the
// meta_* CREATE TABLE IF NOT EXISTS set plus the claimkit tables; without the
// advisory lock, concurrent construction intermittently collides in
// PostgreSQL's catalog — the class t.Parallel suites and CI -count=2 legs
// would multiply.
func TestNew_ConcurrentConstruction(t *testing.T) {
	t.Parallel()

	dsn := pgDSN(t)

	// Pre-flight one construction so skip-class unavailability is handled on
	// the test goroutine; the DSN helper caches per test name, so all workers
	// below hit the same per-test database.
	first, err := pgengine.New(dsn)
	if err != nil {
		if pgSkipClass(err) {
			t.Skipf("Postgres not available: %v", err)
		}

		t.Fatalf("pgengine.New pre-flight: %v", err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close pre-flight engine: %v", err)
	}

	const workers = 8

	errs := make(chan error, workers)
	var wg sync.WaitGroup

	for range workers {

		wg.Go(func() {

			eng, err := pgengine.New(dsn)
			if err != nil {
				errs <- err

				return
			}

			errs <- eng.Close()
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent pgengine.New/Close: %v", err)
		}
	}
}
