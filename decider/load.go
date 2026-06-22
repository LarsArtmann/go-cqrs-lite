package decider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
)

func (r *Repository[State]) loadFromStore(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
) (State, event.Version, error) {
	ref := event.NewAggregateRef(aggregateType, aggregateID)

	return r.loadByEvents(
		func() ([]event.Event, error) {
			if !r.loadCoalescing {
				return r.store.Load(ctx, ref)
			}

			key := ref.Type.String() + "/" + ref.ID.String()

			v, loadErr, _ := r.loadGroup.Do(key, func() (any, error) {
				return r.store.Load(ctx, ref)
			})
			if loadErr != nil {
				return nil, loadErr //nolint:wrapcheck // passthrough from our own store.Load via singleflight
			}

			events, _ := v.([]event.Event)

			return events, nil
		},
		ref,
	)
}

func (r *Repository[State]) foldEvents(
	state State,
	events []event.Event,
	ref event.AggregateRef,
) (State, error) {
	var err error

	for _, evt := range events {
		state, err = r.decider.Apply(state, evt)
		if err != nil {
			var zero State

			return zero, opError(
				ref,
				"%w (event %s): %w",
				ErrApplyFailed,
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
	prefix := ref.Type.String() + " " + ref.ID.String() + ": "
	fmtMsg := strings.ReplaceAll(prefix+msg, "%w", "%v")

	var errs []error

	for _, arg := range args {
		if e, ok := arg.(error); ok {
			errs = append(errs, e)
		}
	}

	if len(errs) == 0 {
		return event.Newf(event.Infrastructure, "decider.op_error", fmtMsg, args...)
	}

	cause := errs[0]
	if len(errs) > 1 {
		cause = event.Compose(errs...)
	}

	return event.Wrapf(cause, event.Classify(cause), "decider.op_error", fmtMsg, args...)
}

// LoadAtVersion reconstructs state from events up to and including maxVersion.
// Useful for time-travel queries: "what was the state at version N?".
func (r *Repository[State]) LoadAtVersion(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
	maxVersion event.Version,
) (State, event.Version, error) {
	ref := event.NewAggregateRef(aggregateType, aggregateID)

	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "decider.load_at_version",
		cqrsotel.SpanKindInternal,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrAggregateType, string(aggregateType)),
			cqrsotel.AttrString(cqrsotel.AttrAggregateID, aggregateID.String()),
			cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, maxVersion.Int()),
		),
	)
	defer span.End()

	state, ver, err := r.loadByEvents(
		func() ([]event.Event, error) {
			return r.store.LoadToVersion(ctx, ref, maxVersion)
		},
		ref,
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
	ref := event.NewAggregateRef(aggregateType, aggregateID)

	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "decider.load_at_time",
		cqrsotel.SpanKindInternal,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrAggregateType, string(aggregateType)),
			cqrsotel.AttrString(cqrsotel.AttrAggregateID, aggregateID.String()),
		),
	)
	defer span.End()

	state, ver, err := r.loadByEvents(
		func() ([]event.Event, error) {
			return r.store.LoadToTimestamp(ctx, ref, maxTime)
		},
		ref,
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

		return zero, 0, opError(ref, "%w: %w", ErrLoadFailed, err)
	}

	state, err := r.foldEvents(r.decider.Initial, events, ref)
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
	return snapshot.ShouldSnapshot(
		r.snapshotStrategy,
		r.snapshotStore,
		r.codec,
		aggregateType,
		version,
	)
}

func (r *Repository[State]) loadFromSnapshot(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
) (State, event.Version, error) {
	ref := event.NewAggregateRef(aggregateType, aggregateID)

	snap, err := r.snapshotStore.Load(ctx, ref)
	if err != nil {
		if !errors.Is(err, snapshot.ErrSnapshotNotFound) {
			var zero State

			return zero, 0, opError(ref, "load snapshot: %w", err)
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

		return zero, 0, opError(ref, "decode snapshot: %w", err)
	}

	events, err := r.store.LoadFromVersion(ctx, ref, snap.Version)
	if err != nil {
		var zero State

		return zero, 0, opError(ref, "%w: %w", ErrLoadFailed, err)
	}

	state, err = r.foldEvents(state, events, ref)
	if err != nil {
		var zero State

		return zero, 0, err
	}

	return state, snap.Version.Add(uint(len(events))), nil
}
