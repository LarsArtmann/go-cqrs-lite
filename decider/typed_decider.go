package decider

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// TypedDecider binds the command type at compile time (ADR-0001 evolution).
//
// Unlike [Decider], which requires the consumer to pass a separate DecideFunc
// to Execute on every call, TypedDecider carries its Decide function as a
// field. This is the Eventuous-style pattern: the aggregate's decision logic
// is part of the type, not a loose parameter.
//
// Usage:
//
//	d := decider.TypedDecider[CounterState, IncrementCmd]{
//		Initial: CounterState{},
//		Decide:  decideIncrement, // func(CounterState, IncrementCmd) ([]event.Event, error)
//		Fold:    foldCounter,     // func(CounterState, event.Event) (CounterState, error)
//	}
//	repo, _ := decider.NewTypedRepository(store, bus, d)
//	err := repo.ExecuteCommand(ctx, aggID, "Counter", IncrementCmd{Amount: 5})
type TypedDecider[State any, Cmd any] struct {
	Initial State
	Decide  func(state State, cmd Cmd) ([]event.Event, error)
	Fold    func(state State, evt event.Event) (State, error)
}

// TypedRepository is a command-bound repository that uses [TypedDecider].
// It wraps [Repository] and adds [TypedRepository.ExecuteCommand], which
// accepts the typed command directly — no DecideFunc needed.
type TypedRepository[State any, Cmd any] struct {
	decider TypedDecider[State, Cmd]
	inner   *Repository[State]
}

// NewTypedRepository creates a repository bound to a [TypedDecider].
// The publisher may be nil for pure event-sourcing mode.
func NewTypedRepository[State, Cmd any](
	store event.Store,
	publisher event.Publisher,
	d TypedDecider[State, Cmd],
	opts ...RepositoryOption[State],
) (*TypedRepository[State, Cmd], error) {
	legacyDecider := Decider[State]{
		Initial: d.Initial,
		Fold:    d.Fold,
	}

	inner, err := NewRepository(store, publisher, legacyDecider, opts...)
	if err != nil {
		return nil, err
	}

	return &TypedRepository[State, Cmd]{decider: d, inner: inner}, nil
}

// ExecuteCommand loads the aggregate, folds its history, calls the typed
// Decide function with the command, and persists any resulting events.
func (r *TypedRepository[State, Cmd]) ExecuteCommand(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
	cmd Cmd,
) error {
	return r.inner.Execute(
		ctx, aggregateID, aggregateType,
		func(state State, _ event.Version) ([]event.Event, error) {
			return r.decider.Decide(state, cmd)
		},
	)
}

// Load delegates to the underlying [Repository.Load].
func (r *TypedRepository[State, Cmd]) Load(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
) (State, event.Version, error) {
	return r.inner.Load(ctx, aggregateID, aggregateType)
}
