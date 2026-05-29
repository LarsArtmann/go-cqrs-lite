package decider

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
	"github.com/larsartmann/go-cqrs-lite/snapshot"
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
// It wraps event.Store (persistence) and event.Publisher (publishing) behind a
// Decider[State], providing load → fold → decide → save → publish semantics
// without requiring the consumer to implement a mutable aggregate root
// interface.
type Repository[State any] struct {
	store            event.Store
	publisher        event.Publisher
	snapshotStore    snapshot.SnapshotStore
	codec            codec.Codec
	snapshotStrategy snapshot.SnapshotStrategy
	enricher         event.ContextEnricher
	decider          Decider[State]
}

// NewRepository creates a decider-backed repository.
//
// Returns an error if store, publisher, or decider.Fold is nil.
func NewRepository[State any](
	store event.Store,
	publisher event.Publisher,
	decider Decider[State],
	opts ...RepositoryOption[State],
) (*Repository[State], error) {
	if store == nil {
		return nil, ErrNilStore
	}

	if publisher == nil {
		return nil, ErrNilBus
	}

	if decider.Fold == nil {
		return nil, ErrNilFold
	}

	r := &Repository[State]{ //nolint:exhaustruct // options fill remaining fields
		store:     store,
		publisher: publisher,
		decider:   decider,
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
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
	decide DecideFunc[State],
) error {
	ref := event.NewAggregateRef(aggregateType, aggregateID)

	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "decider.execute",
		trace.SpanKindInternal,
		trace.WithAttributes(cqrsotel.AggregateAttrs(aggregateType, aggregateID)...),
	)
	defer span.End()

	state, currentVersion, err := r.Load(ctx, aggregateID, aggregateType)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return err
	}

	newEvents, err := decide(state, currentVersion)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return err
	}

	if len(newEvents) == 0 {
		return nil
	}

	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(newEvents)))

	r.applyEnricher(ctx, newEvents)

	err = r.store.Save(ctx, ref, newEvents, currentVersion)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return opError(ref, "%w: %w", ErrSaveFailed, err)
	}

	err = r.publisher.Publish(ctx, newEvents...)
	if err != nil {
		cqrsotel.RecordError(span, err)
		wrapErr := event.WrapInfrastructure(err, "event.publish_failed", "publish events")

		return opError(ref, "%w", wrapErr)
	}

	newVersion := currentVersion.Add(len(newEvents))

	r.saveSnapshotAfterEvents(ctx, ref, newVersion, state, newEvents)

	return nil
}

// saveSnapshotAfterEvents folds new events onto state to get the final state,
// then attempts to save a snapshot. Errors are logged and swallowed — snapshots
// are best-effort and must not block the write path.
func (r *Repository[State]) saveSnapshotAfterEvents(
	ctx context.Context,
	ref event.AggregateRef,
	newVersion event.Version,
	state State,
	newEvents []event.Event,
) {
	if !r.shouldSnapshot(ref.Type, newVersion) {
		return
	}

	finalState := state

	for _, evt := range newEvents {
		var foldErr error

		finalState, foldErr = r.decider.Fold(finalState, evt)
		if foldErr != nil {
			_ = opError(ref, "fold event %s for snapshot: %w", evt.Type(), foldErr)

			return
		}
	}

	encoded, encErr := r.codec.Encode(finalState)
	if encErr != nil {
		_ = opError(ref, "encode snapshot: %w", encErr)

		return
	}

	_ = snapshot.SaveSnapshot(ctx, r.snapshotStore, ref.Type, ref.ID, newVersion, encoded)
}

// Load reconstructs state from the aggregate's event history without any
// side effects. Useful for read-only state access or debugging.
func (r *Repository[State]) Load(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
) (State, event.Version, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "decider.load",
		trace.SpanKindInternal,
		trace.WithAttributes(cqrsotel.AggregateAttrs(aggregateType, aggregateID)...),
	)
	defer span.End()

	var (
		state State
		ver   event.Version
		err   error
	)

	if r.snapshotStore != nil && r.codec != nil {
		state, ver, err = r.loadFromSnapshot(ctx, aggregateID, aggregateType)
	} else {
		state, ver, err = r.loadFromStore(ctx, aggregateID, aggregateType)
	}

	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return state, ver, err
}

func (r *Repository[State]) applyEnricher(ctx context.Context, events []event.Event) {
	if r.enricher == nil {
		return
	}

	opts := r.enricher(ctx)
	if len(opts) == 0 {
		return
	}

	for _, evt := range events {
		immutable, ok := evt.(*event.ImmutableEvent)
		if !ok {
			continue
		}

		for _, opt := range opts {
			opt(immutable)
		}
	}
}
