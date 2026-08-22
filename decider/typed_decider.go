package decider

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TypedDecider binds the command type at compile time (ADR-0001 evolution).
//
// Unlike [Decider], which requires the consumer to pass a separate DecideFunc
// to Execute on every call, TypedDecider carries its Decide function as a
// field. This is the Eventuous-style pattern: the stream's decision logic
// is part of the type, not a loose parameter.
//
// Usage:
//
//	d := decider.TypedDecider[CounterState, IncrementCmd]{
//		Initial: CounterState{},
//		Decide:  decideIncrement, // func(CounterState, IncrementCmd) ([]event.Event, error)
//		Apply:    applyCounter,     // func(CounterState, event.Event) (CounterState, error)
//	}
//	repo, _ := decider.NewTypedRepository(store, bus, d)
//	err := repo.ExecuteCommand(ctx, aggID, "Counter", IncrementCmd{Amount: 5})
type TypedDecider[State any, Cmd any] struct {
	Initial State
	Decide  func(state State, cmd Cmd) ([]event.Event, error)
	Apply   func(state State, evt event.Event) (State, error)
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
		Apply:   d.Apply,
	}

	inner, err := NewRepository(store, publisher, legacyDecider, opts...)
	if err != nil {
		return nil, err
	}

	return &TypedRepository[State, Cmd]{decider: d, inner: inner}, nil
}

// ExecuteCommandRef loads the stream, folds its history, calls the typed
// Decide function with the command, and persists any resulting events.
//
// The stream is addressed by a single [id.StreamRef].
func (r *TypedRepository[State, Cmd]) ExecuteCommandRef(
	ctx context.Context,
	ref id.StreamRef,
	cmd Cmd,
) error {
	return r.inner.ExecuteRef(
		ctx, ref,
		func(state State, _ event.Version) ([]event.Event, error) {
			return r.decider.Decide(state, cmd)
		},
	)
}

// ExecuteCommand loads the stream, folds its history, calls the typed
// Decide function with the command, and persists any resulting events.
//
// Deprecated: removed in v5. Use [TypedRepository.ExecuteCommandRef] with
// [id.NewStreamRef]; this pair form forwards to it unchanged.
func (r *TypedRepository[State, Cmd]) ExecuteCommand(
	ctx context.Context,
	streamID id.StreamID,
	streamType id.StreamType,
	cmd Cmd,
) error {
	return r.ExecuteCommandRef(ctx, id.NewStreamRef(streamType, streamID), cmd)
}

// LoadRef delegates to the underlying [Repository.LoadRef].
func (r *TypedRepository[State, Cmd]) LoadRef(
	ctx context.Context,
	ref id.StreamRef,
) (State, event.Version, error) {
	return r.inner.LoadRef(ctx, ref)
}

// Load delegates to the underlying [Repository.Load].
//
// Deprecated: removed in v5. Use [TypedRepository.LoadRef] with
// [id.NewStreamRef]; this pair form forwards to it unchanged.
func (r *TypedRepository[State, Cmd]) Load(
	ctx context.Context,
	streamID id.StreamID,
	streamType id.StreamType,
) (State, event.Version, error) {
	return r.LoadRef(ctx, id.NewStreamRef(streamType, streamID))
}
