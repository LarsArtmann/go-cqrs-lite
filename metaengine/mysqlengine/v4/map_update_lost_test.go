package mysqlengine_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMySQLEngine_MapUpdateMissingKeySerializes pins the create-on-update case
// of the MapUpdater contract: two concurrent MapUpdate calls on a key that
// does not exist yet must serialize, so an incrementing update applied 2*N
// times ends at 2*N. SELECT ... FOR UPDATE locks no missing row; without the named key lock
// taken before the read, the second upsert overwrote the first.
func TestMySQLEngine_MapUpdateMissingKeySerializes(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)
	mu := eng.(metaengine.MapUpdater)
	ctx := context.Background()

	const keys, perGoroutine = 4, 100

	for k := range keys {
		key := fmt.Sprintf("lost-update-%d-%d", time.Now().UnixNano(), k)

		var (
			wg    sync.WaitGroup
			errs  = make(chan error, 2)
			start = make(chan struct{})
		)

		for range 2 {
			wg.Add(1)

			go func() {
				defer wg.Done()
				<-start

				for range perGoroutine {
					err := mu.MapUpdate(ctx, "map_update_lost_test", key, func(prev any) any {
						n := 0.0

						if m, ok := prev.(map[string]any); ok {
							if v, ok := m["n"].(float64); ok {
								n = v
							}
						}

						return map[string]any{"n": n + 1}
					})
					if err != nil {
						errs <- err

						return
					}
				}
			}()
		}

		close(start)
		wg.Wait()
		close(errs)

		for err := range errs {
			t.Fatalf("MapUpdate: %v", err)
		}

		got, ok, err := eng.(metaengine.MapBackend).MapGet(ctx, "map_update_lost_test", key)
		if err != nil || !ok {
			t.Fatalf("MapGet after updates: ok=%v err=%v", ok, err)
		}

		if n := got.(map[string]any)["n"]; n != float64(2*perGoroutine) { //nolint:forcetypeassert // written above
			t.Errorf("key %s: counter = %v, want %d (lost update on a missing key)", key, n, 2*perGoroutine)
		}
	}
}
