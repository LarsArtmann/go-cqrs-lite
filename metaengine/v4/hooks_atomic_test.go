package metaengine

import (
	"bytes"
	"context"
	"sync"
	"testing"
)

// Attaching and detaching a trace while the store serves must not race with
// the fold/apply/execute dispatch sites.
func TestHooks_SwapWhileServing(t *testing.T) {
	store, _ := newTraceStore(t)

	var buf bytes.Buffer

	stop := make(chan struct{})

	var serving sync.WaitGroup

	serving.Add(1)

	go func() {
		defer serving.Done()

		ctx := context.Background()

		for {
			select {
			case <-stop:
				return
			default:
			}

			_, _ = store.Execute(roleFindItem{ID: "x"})
			_ = store.Apply(ctx, "roleItemCreated", roleItemCreated{ID: "x", Name: "x"})
		}
	}()

	for range 50 {
		RecordTrace(store, &buf).Close()
	}

	close(stop)
	serving.Wait()
}

// RecordTrace must keep every previously installed callback, not only the
// apply/fold ones: losing OnQuarantined while diagnosing an unhealthy store
// hides exactly the signal the operator is looking for.
func TestTrace_ChainsHealthHooks(t *testing.T) {
	t.Parallel()

	store, _ := newTraceStore(t)

	calls := 0

	WithHooks(store, Hooks{
		OnQuarantined: func(string, int, string) { calls++ },
		OnReactivated: func(string, string) { calls++ },
		OnProbe:       func(string, error) { calls++ },
		OnCatchUp:     func(string, int, error) { calls++ },
	})

	rec := RecordTrace(store, &bytes.Buffer{})

	store.emitQuarantined("e", 1, "boom")
	store.emitReactivated("e", "manual")
	store.emitProbe("e", nil)
	store.emitCatchUp("e", 1, nil)

	rec.Close()

	if calls != 4 {
		t.Fatalf("previous health hooks should fire 4 times while tracing, fired %d", calls)
	}
}
