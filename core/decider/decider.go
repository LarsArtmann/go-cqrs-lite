package decider

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

var (
	ErrNilStore   = errors.New("event store is required")
	ErrNilBus     = errors.New("event bus is required")
	ErrNilFold    = errors.New("fold function is required")
	ErrLoadFailed = errors.New("failed to load events")
	ErrFoldFailed = errors.New("failed to fold events")
	ErrSaveFailed = errors.New("failed to save events")
)

// Decider defines how to reconstruct state from events.
//
// State is the aggregate's domain state. Fold applies a single event to the
// state, returning the updated state. Fold must be a pure function — it should
// not perform I/O or have side effects.
//
// If Fold returns an error (e.g. corrupted payload), Execute aborts and
// returns ErrFoldFailed wrapping the cause.
type Decider[State any] struct {
	Initial State
	Fold    func(state State, evt event.Event) (State, error)
}

// Repository loads and saves aggregates using pure functions.
//
// It wraps event.Store (persistence) and event.Bus (publishing) behind a
// Decider[State], providing load → fold → decide → save → publish semantics
// without requiring the consumer to implement a mutable aggregate root
// interface.
type Repository[State any] struct {
	store   event.Store
	bus     event.Bus
	outbox  event.Outbox
	decider Decider[State]
}

// NewRepository creates a decider-backed repository.
//
// Returns an error if store, bus, or decider.Fold is nil.
func NewRepository[State any](
	store event.Store,
	bus event.Bus,
	decider Decider[State],
	opts ...RepositoryOption[State],
) (*Repository[State], error) {
	if store == nil {
		return nil, ErrNilStore
	}

	if bus == nil {
		return nil, ErrNilBus
	}

	if decider.Fold == nil {
		return nil, ErrNilFold
	}

	r := &Repository[State]{ //nolint:exhaustruct // options fill remaining fields
		store:   store,
		bus:     bus,
		decider: decider,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// DecideFunc is the signature for a decision function.
//
// It receives the current state and version, and returns the events to
// persist. Return an error to reject the command (no events will be saved).
type DecideFunc[State any] func(state State, currentVersion event.Version) ([]event.Event, error)

// Execute loads the aggregate's event history, folds it into state, calls
// decide, and if decide returns events, persists them to the store and
// publishes them to the bus.
//
// The decide function receives the reconstructed state and the current version
// (derived from the number of loaded events). Use currentVersion + 1, + 2,
// etc. when creating new events via event.NewEvent.
//
// If decide returns an error, no events are saved or published.
// If store.Save succeeds but bus.Publish fails, events are persisted but not
// published — the caller can retry publishing via the bus directly.
func (r *Repository[State]) Execute(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
	decide DecideFunc[State],
) error {
	state, currentVersion, err := r.loadState(ctx, aggID, aggType)
	if err != nil {
		return err
	}

	newEvents, err := decide(state, currentVersion)
	if err != nil {
		return err
	}

	if len(newEvents) == 0 {
		return nil
	}

	err = r.store.Save(ctx, aggType, aggID, newEvents, currentVersion)
	if err != nil {
		return opError(aggType, aggID, "%w: %w", ErrSaveFailed, err)
	}

	err = r.publishChanges(ctx, newEvents, aggType, aggID)
	if err != nil {
		return err
	}

	return nil
}

// Load reconstructs state from the aggregate's event history without any
// side effects. Useful for read-only state access or debugging.
func (r *Repository[State]) Load(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
) (State, event.Version, error) {
	return r.loadState(ctx, aggID, aggType)
}

func (r *Repository[State]) loadState(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
) (State, event.Version, error) {
	events, err := r.store.Load(ctx, aggType, aggID)
	if err != nil {
		if errors.Is(err, event.ErrAggregateNotFound) {
			return r.decider.Initial, 0, nil
		}

		var zero State
		return zero, 0, opError(aggType, aggID, "%w: %w", ErrLoadFailed, err)
	}

	state := r.decider.Initial
	for _, evt := range events {
		state, err = r.decider.Fold(state, evt)
		if err != nil {
			var zero State
			return zero, 0, opError(
				aggType,
				aggID,
				"%w (event %s): %w",
				ErrFoldFailed,
				evt.Type(),
				err,
			)
		}
	}

	return state, event.Version(len(events)), nil
}

// Delete removes all events for the aggregate from the store.
func (r *Repository[State]) Delete(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
) error {
	err := r.store.Delete(ctx, aggType, aggID)
	if err != nil {
		return opError(aggType, aggID, "delete: %w", err)
	}

	return nil
}

func (r *Repository[State]) publishChanges(
	ctx context.Context,
	events []event.Event,
	aggType event.AggregateType,
	aggID id.AggregateID,
) error {
	if r.outbox != nil {
		err := r.outbox.Append(ctx, events)
		if err != nil {
			return opError(aggType, aggID, "stage events in outbox: %w", err)
		}
	} else {
		err := r.bus.Publish(ctx, events...)
		if err != nil {
			return opError(aggType, aggID, "publish events: %w", err)
		}
	}

	return nil
}

func opError(aggType event.AggregateType, aggID id.AggregateID, format string, args ...any) error {
	prefix := aggType.String() + " " + aggID.String() + ": "

	return fmt.Errorf(prefix+format, args...)
}
