package sagatest

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

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

func SaveSagaState(t *testing.T, ctx context.Context, store saga.Store, state *saga.State) {
	t.Helper()

	err := store.Save(ctx, state)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
}
