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

	const workers = 8

	engines := make(chan any, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			eng, err := pgengine.New(pgDSN(t))
			if err != nil {
				if pgSkipClass(err) {
					engines <- nil
					return
				}

				errs <- err

				return
			}

			engines <- eng
		}()
	}

	wg.Wait()
	close(engines)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent pgengine.New: %v", err)
		}
	}

	for eng := range engines {
		if eng == nil {
			t.Skip("Postgres not available for all workers")
		}

		if closer, ok := eng.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				t.Fatalf("close engine: %v", err)
			}
		}
	}
}
