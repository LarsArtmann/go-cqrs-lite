package metaengine_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// failingMapEngine delegates to the memory engine but fails MapSet on demand,
// so an apply can be made to fail after the store accepted it.
type failingMapEngine struct {
	metaengine.Engine

	fail atomic.Bool
}

func (e *failingMapEngine) MapSet(ctx context.Context, col string, key, value any) error {
	if e.fail.Load() {
		return errors.New("engine down")
	}

	return e.backend().MapSet(ctx, col, key, value) //nolint:wrapcheck
}

func (e *failingMapEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	return e.backend().MapGet(ctx, col, key) //nolint:wrapcheck
}

func (e *failingMapEngine) MapDelete(ctx context.Context, col string, key any) error {
	return e.backend().MapDelete(ctx, col, key) //nolint:wrapcheck
}

func (e *failingMapEngine) backend() metaengine.MapBackend {
	return e.Engine.(metaengine.MapBackend)
}

// TestApplyIdempotentFailedApplyIsRetried pins that a failed apply does not
// record its event ID: the retry of the same ID applies and is visible.
func TestApplyIdempotentFailedApplyIsRetried(t *testing.T) {
	t.Parallel()

	eng := &failingMapEngine{Engine: metaengine.NewMemoryEngine()}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	evt := TaskCreated{ID: "retry-1", Title: "t", Status: "open"}

	eng.fail.Store(true)
	if err := store.ApplyIdempotent(ctx, "evt-1", "TaskCreated", evt); err == nil {
		t.Fatal("ApplyIdempotent must surface the engine error")
	}

	eng.fail.Store(false)
	if err := store.ApplyIdempotent(ctx, "evt-1", "TaskCreated", evt); err != nil {
		t.Fatalf("retry ApplyIdempotent: %v", err)
	}

	reader := metaengine.NewReader[FindTaskResult](store, "find_task")

	if _, found, err := reader.Get(ctx, TaskID("retry-1")); err != nil || !found {
		t.Fatalf("after the retry the event must be projected (found=%v, err=%v)", found, err)
	}
}
