package decider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

func (r *Repository[State]) loadFromStore(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
) (State, event.Version, error) {
	return r.loadByEvents(
		func() ([]event.Event, error) { return r.store.Load(ctx, aggregateType, aggregateID) },
		aggregateType,
		aggregateID,
	)
}

func (r *Repository[State]) foldEvents(
	state State,
	events []event.Event,
	ref event.AggregateRef,
) (State, error) {
	var err error

	for _, evt := range events {
		state, err = r.decider.Fold(state, evt)
		if err != nil {
			var zero State

			return zero, opError(
				aggregateType,
				aggregateID,
				"%w (event %s): %w",
				ErrFoldFailed,
				evt.Type(),
				err,
			)
		}
	}

	return state, nil
}

func opError(
	ref event.AggregateRef,
	msg string,
	args ...any,
) error {
	prefix := aggregateType.String() + " " + aggregateID.String() + ": "

	return fmt.Errorf(prefix+msg, args...) //nolint:err113
}

// LoadAtVersion reconstructs state from events up to and including maxVersion.
// Useful for time-travel queries: "what was the state at version N?".
func (r *Repository[State]) LoadAtVersion(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
	maxVersion event.Version,
) (State, event.Version, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "decider.load_at_version",
		trace.SpanKindInternal,
		trace.WithAttributes(
			attribute.String(cqrsotel.AttrAggregateType, string(aggregateType)),
			attribute.String(cqrsotel.AttrAggregateID, aggregateID.String()),
			attribute.Int(cqrsotel.AttrAggregateVersion, maxVersion.Int()),
		),
	)
	defer span.End()

	state, ver, err := r.loadByEvents(
		func() ([]event.Event, error) {
			return r.store.LoadToVersion(ctx, aggregateType, aggregateID, maxVersion)
		},
		aggregateType, aggregateID,
	)
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return state, ver, err
}

// LoadAtTime reconstructs state from events up to and including maxTime.
// Useful for temporal queries: "what was the state at this point in time?".
func (r *Repository[State]) LoadAtTime(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
	maxTime time.Time,
) (State, event.Version, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "decider.load_at_time",
		trace.SpanKindInternal,
		trace.WithAttributes(
			attribute.String(cqrsotel.AttrAggregateType, string(aggregateType)),
			attribute.String(cqrsotel.AttrAggregateID, aggregateID.String()),
		),
	)
	defer span.End()

	state, ver, err := r.loadByEvents(
		func() ([]event.Event, error) {
			return r.store.LoadToTimestamp(ctx, aggregateType, aggregateID, maxTime)
		},
		aggregateType, aggregateID,
	)
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return state, ver, err
}

func (r *Repository[State]) loadByEvents(
	loadFn func() ([]event.Event, error),
	ref event.AggregateRef,
) (State, event.Version, error) {
	events, err := loadFn()
	if err != nil {
		if errors.Is(err, event.ErrAggregateNotFound) {
			return r.decider.Initial, 0, nil
		}

		var zero State

		return zero, 0, opError(aggregateType, aggregateID, "%w: %w", ErrLoadFailed, err)
	}

	state, err := r.foldEvents(r.decider.Initial, events, aggregateType, aggregateID)
	if err != nil {
		var zero State

		return zero, 0, err
	}

	return state, event.Version(len(events)), nil
}

func (r *Repository[State]) shouldSnapshot(
	aggregateType event.AggregateType,
	version event.Version,
) bool {
	return event.ShouldSnapshot(r.snapshotStrategy, r.snapshotStore, r.codec, aggregateType, version)
}

func (r *Repository[State]) loadFromSnapshot(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
) (State, event.Version, error) {
	snap, err := r.snapshotStore.Load(ctx, aggregateType, aggregateID)
	if err != nil {
		if !errors.Is(err, event.ErrSnapshotNotFound) {
			var zero State

			return zero, 0, opError(aggregateType, aggregateID, "load snapshot: %w", err)
		}

		return r.loadFromStore(ctx, aggregateID, aggregateType)
	}

	if snap == nil {
		return r.loadFromStore(ctx, aggregateID, aggregateType)
	}

	var state State

	err = r.codec.Decode(snap.State, &state)
	if err != nil {
		var zero State

		return zero, 0, opError(aggregateType, aggregateID, "decode snapshot: %w", err)
	}

	events, err := r.store.LoadFromVersion(ctx, aggregateType, aggregateID, snap.Version)
	if err != nil {
		var zero State

		return zero, 0, opError(aggregateType, aggregateID, "%w: %w", ErrLoadFailed, err)
	}

	state, err = r.foldEvents(state, events, aggregateType, aggregateID)
	if err != nil {
		var zero State

		return zero, 0, err
	}

	return state, snap.Version.Add(len(events)), nil
}
