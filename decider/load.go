package decider

import (
	"context"
	"errors"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func (r *Repository[State]) loadFromStore(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType id.AggregateType,
) (State, event.Version, error) {
	ref := id.NewAggregateRef(aggregateType, aggregateID)

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
	ref id.AggregateRef,
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
	ref id.AggregateRef,
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
		return errorfamily.Newf(errorfamily.Infrastructure, "decider.op_error", fmtMsg, args...)
	}

	cause := errs[0]
	if len(errs) > 1 {
		cause = errors.Join(errs...)
	}

	return errorfamily.Wrapf(
		cause,
		errorfamily.Classify(cause),
		"decider.op_error",
		fmtMsg,
		args...,
	)
}

// LoadAtVersion reconstructs state from events up to and including maxVersion.
// Useful for time-travel queries: "what was the state at version N?".
func (r *Repository[State]) LoadAtVersion(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType id.AggregateType,
	maxVersion event.Version,
) (State, event.Version, error) {
	ref := id.NewAggregateRef(aggregateType, aggregateID)

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
	aggregateType id.AggregateType,
	maxTime time.Time,
) (State, event.Version, error) {
	ref := id.NewAggregateRef(aggregateType, aggregateID)

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
	ref id.AggregateRef,
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
	ref id.AggregateRef,
	version event.Version,
) bool {
	return snapshot.ShouldSnapshotFor(
		r.snapshotStrategy,
		r.snapshotStore,
		r.codec,
		ref,
		version,
	)
}

func (r *Repository[State]) loadFromSnapshot(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType id.AggregateType,
) (State, event.Version, error) {
	ref := id.NewAggregateRef(aggregateType, aggregateID)

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

// loadFromCache attempts an incremental load from the hot-state cache.
// On a hit, it loads only events after the cached version and folds them
// onto the cached state. Returns ok=false on miss, error, or fold failure
// (the caller falls back to the full load path).
func (r *Repository[State]) loadFromCache(
	ctx context.Context,
	aggregateID id.AggregateID,
	aggregateType id.AggregateType,
) (State, event.Version, bool) {
	ref := id.NewAggregateRef(aggregateType, aggregateID)

	cachedState, cachedVersion, ok := r.stateCache.Get(ref)
	if !ok {
		var zero State

		return zero, 0, false
	}

	events, err := r.store.LoadFromVersion(ctx, ref, cachedVersion)
	if err != nil {
		r.stateCache.Invalidate(ref)

		var zero State

		return zero, 0, false
	}

	if len(events) == 0 {
		return cachedState, cachedVersion, true
	}

	finalState, err := r.foldEvents(cachedState, events, ref)
	if err != nil {
		r.stateCache.Invalidate(ref)

		var zero State

		return zero, 0, false
	}

	finalVersion := cachedVersion.Add(uint(len(events)))
	r.stateCache.Put(ref, finalState, finalVersion)

	return finalState, finalVersion, true
}

// recordRead notifies the snapshot strategy of a read, enabling read-pressure
// strategies like ReadPressure to track load frequency.
func (r *Repository[State]) recordRead(ref id.AggregateRef, version event.Version) {
	if r.snapshotStrategy == nil {
		return
	}

	if tracker, ok := r.snapshotStrategy.(snapshot.ReadTracker); ok {
		tracker.RecordRead(ref, version)
	}
}
