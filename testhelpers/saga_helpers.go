package testhelpers

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

// NewSagaState creates a saga.State with sensible defaults for tests.
func NewSagaState(sagaType string, status saga.Status, currentStep int, errMsg string) *saga.State {
	now := time.Now()

	return &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    sagaType,
		Status:      status,
		CurrentStep: currentStep,
		ErrMsg:      errMsg,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SaveSagaState saves a saga state and fails the test on error.
// Use this instead of duplicating: if err := store.Save(ctx, state); err != nil { t.Fatalf(...) }.
func SaveSagaState(t *testing.T, ctx context.Context, store saga.Store, state *saga.State) {
	t.Helper()

	err := store.Save(ctx, state)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
}
