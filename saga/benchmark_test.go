package saga_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

func BenchmarkRunner_Register(b *testing.B) {
	store := saga.NewMemoryStore()
	dispatcher := nopDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	b.ResetTimer()

	for i := range b.N {
		def := testDefinition{
			sagaType: "bench-saga-" + string(rune(i)),
			steps:    []saga.Step{{Name: "step-1"}},
		}
		_ = runner.Register(def)
	}
}

func BenchmarkRunner_ExecuteStep(b *testing.B) {
	store := saga.NewMemoryStore()
	dispatcher := nopDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	def := testDefinition{
		sagaType: "bench-saga",
		steps: []saga.Step{
			{Name: "step-1", Action: func(_ context.Context, _ id.AggregateID) command.Command {
				cmd, _ := command.New("noop", id.NewAggregateID())

				return cmd
			}},
		},
	}

	_ = runner.Register(def)

	instanceID := id.NewAggregateID()
	state := &saga.State{
		ID:          instanceID,
		SagaType:    "bench-saga",
		CurrentStep: 0,
		Status:      saga.StatusRunning,
	}
	_ = store.Save(context.Background(), state)

	b.ResetTimer()

	for range b.N {
		_ = runner.ExecuteStep(context.Background(), instanceID)
	}
}

func BenchmarkMemoryStore_Save(b *testing.B) {
	store := saga.NewMemoryStore()

	b.ResetTimer()

	for range b.N {
		state := &saga.State{
			ID:          id.NewAggregateID(),
			SagaType:    "bench-saga",
			CurrentStep: 0,
			Status:      saga.StatusRunning,
		}
		_ = store.Save(context.Background(), state)
	}
}

func BenchmarkMemoryStore_Load(b *testing.B) {
	store := saga.NewMemoryStore()
	instanceID := id.NewAggregateID()

	state := &saga.State{
		ID:          instanceID,
		SagaType:    "bench-saga",
		CurrentStep: 0,
		Status:      saga.StatusRunning,
	}
	_ = store.Save(context.Background(), state)

	b.ResetTimer()

	for range b.N {
		_, _ = store.Load(context.Background(), instanceID)
	}
}
