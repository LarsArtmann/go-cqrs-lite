package aggregate_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

type testState struct{ Name string }

func testFold(state testState, evt event.Event) (testState, error) {
	return state, nil
}

func TestTypeAliases_Compile(t *testing.T) {
	t.Parallel()

	var _ aggregate.Decider[testState] = decider.Decider[testState]{}
	var _ aggregate.Repository[testState] = decider.Repository[testState]{}
	var _ aggregate.DecideFunc[testState] = func(testState, event.Version) ([]event.Event, error) {
		return nil, nil
	}
}

func TestNewRepository_DelegatesToDecider(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	d := aggregate.Decider[testState]{
		Initial: testState{},
		Fold:    testFold,
	}

	repo, err := aggregate.NewRepository(store, bus, d)
	require.NoError(t, err)
	require.NotNil(t, repo)
}

func TestExecute_DelegatesToDecider(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	d := aggregate.Decider[testState]{
		Initial: testState{},
		Fold:    testFold,
	}

	repo, err := aggregate.NewRepository(store, bus, d)
	require.NoError(t, err)

	aggID := id.NewAggregateID()
	err = aggregate.Execute(
		repo,
		context.Background(),
		aggID,
		"Test",
		func(state testState, v event.Version) ([]event.Event, error) {
			evt, evErr := event.NewEvent("test.created", aggID, "Test", v.Add(1), nil)
			if evErr != nil {
				return nil, evErr
			}

			return []event.Event{evt}, nil
		},
	)
	require.NoError(t, err)
}
