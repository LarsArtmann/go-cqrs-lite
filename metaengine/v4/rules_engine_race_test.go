package metaengine

import (
	"context"
	"sync"
	"testing"
)

// A replan writes each query's assigned engine under the store lock (phase 1)
// and then runs the plan rules without it (phase 2). Two overlapping replans
// must therefore not let the rules of one read the engine the other writes.
func TestRules_ConcurrentReplansReadAssignedEngine(t *testing.T) {
	store, err := Plan([]Engine{NewMemoryEngine()}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	var (
		wg     sync.WaitGroup
		failed = make(chan error, 4)
	)

	for range 4 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 25 {
				if err := store.Replan(ctx); err != nil {
					failed <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(failed)

	for err := range failed {
		t.Errorf("Replan: %v", err)
	}
}
