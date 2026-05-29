package testhelpers

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/saga"
)

// SaveSagaState saves a saga state and fails the test on error.
// Use this instead of duplicating: if err := store.Save(ctx, state); err != nil { t.Fatalf(...) }
func SaveSagaState(t *testing.T, ctx context.Context, store saga.Store, state *saga.State) {
	t.Helper()
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}
}
