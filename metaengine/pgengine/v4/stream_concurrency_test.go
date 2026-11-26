package pgengine_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Two standalone StreamAppendExpected calls on one stream with the same
// expected version: exactly one may succeed. Plain values carry no typed
// version, so the unique indexes cannot catch the second one — only the
// serialization of check-then-append can.
func TestPostgresEngine_StreamAppendExpected_Concurrent(t *testing.T) {
	t.Parallel()

	// Skips (via mustNewPgEngine) when no Postgres is reachable: needs
	// POSTGRES_TEST_DSN or Docker for the testcontainer.
	eng := mustNewPgEngine(t)
	ctx := context.Background()

	ap := eng.(metaengine.AtomicAppender)
	slb := eng.(metaengine.StreamLogBackend)

	const callers = 4

	for round := range 25 {
		stream := fmt.Sprintf("concurrent-%d", round)

		var (
			wg        sync.WaitGroup
			mu        sync.Mutex
			successes int
		)

		start := make(chan struct{})

		for i := range callers {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()
				<-start

				err := ap.StreamAppendExpected(ctx, "concurrency", stream, 0, []any{fmt.Sprintf("w%d", i)})

				mu.Lock()
				defer mu.Unlock()

				switch {
				case err == nil:
					successes++
				case errors.Is(err, metaengine.ErrVersionConflict):
				default:
					t.Errorf("round %d caller %d: unexpected error %v", round, i, err)
				}
			}(i)
		}

		close(start)
		wg.Wait()

		if successes != 1 {
			t.Fatalf("round %d: %d callers passed the version check, want exactly 1", round, successes)
		}

		ver, err := slb.StreamVersion(ctx, "concurrency", stream)
		if err != nil {
			t.Fatalf("round %d: StreamVersion: %v", round, err)
		}

		if ver != 1 {
			t.Fatalf("round %d: version = %d, want 1", round, ver)
		}
	}
}
