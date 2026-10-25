package system

// Version-bounded and by-ID event reads on EventAdapter: the temporal
// fast-path-then-fallback pattern and the EventByIDBackend delegate. Split
// from adapter_event.go (file-size ratchet).

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// loadVersioned implements the temporal-fast-path-then-fallback pattern shared
// by LoadFromVersion and LoadToVersion. temporalRead is only called when
// a.temporal is non-nil. sliceFallback slices the full-stream fallback result.
func (a *EventAdapter) loadVersioned(
	ctx context.Context,
	ref id.StreamRef,
	temporalRead func(collection, key string) ([]any, error),
	errLabel string,
	sliceFallback func(all []event.Event) []event.Event,
) ([]event.Event, error) {
	if a.temporal != nil {
		key, err := resolveStreamKey(ctx, a.Backend, a.Collection, ref)
		if err != nil {
			return nil, fmt.Errorf("event adapter: resolve stream key: %w", err)
		}

		values, err := temporalRead(a.Collection, key)
		if err != nil {
			return nil, fmt.Errorf("event adapter: %s: %w", errLabel, err)
		}

		return a.FromAny(values)
	}

	all, err := a.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	return sliceFallback(all), nil
}

func (a *EventAdapter) LoadFromVersion(
	ctx context.Context, ref id.StreamRef, version event.Version,
) ([]event.Event, error) {
	// StreamReadFromVersion is 1-indexed inclusive; LoadFromVersion's version
	// is 0-indexed (skip N events), so add 1.
	return a.loadVersioned(
		ctx, ref,
		func(collection, key string) ([]any, error) {
			return a.temporal.StreamReadFromVersion(
				ctx, collection, key, int64(version)+1,
			)
		},
		"load from version",
		func(all []event.Event) []event.Event {
			return all[min(int(version), len(all)):]
		},
	)
}

func (a *EventAdapter) LoadToVersion(
	ctx context.Context, ref id.StreamRef, maxVersion event.Version,
) ([]event.Event, error) {
	return a.loadVersioned(
		ctx, ref,
		func(collection, key string) ([]any, error) {
			return a.temporal.StreamReadAsOfVersion(
				ctx, collection, key, int64(maxVersion),
			)
		},
		"load to version",
		func(all []event.Event) []event.Event {
			return all[:min(int(maxVersion), len(all))]
		},
	)
}

func (a *EventAdapter) LoadToTimestamp(
	ctx context.Context, ref id.StreamRef, maxTime time.Time,
) ([]event.Event, error) {
	all, err := a.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	result := make([]event.Event, 0, len(all))
	for _, evt := range all {
		if !evt.OccurredAt().After(maxTime) {
			result = append(result, evt)
		}
	}

	return result, nil
}

// LoadByEventID resolves one event by its globally unique ID (#32). It
// delegates to the backend's EventByIDBackend capability (an indexed lookup
// where the engine knows where the ID lives — e.g. the SQLite engine's
// expression index over the JSON envelope) and returns
// [event.ErrEventNotFound] when no such event exists.
//
// Backends without the capability return [ErrLoadByEventIDUnsupported] —
// callers keep their existing degrade path (sequential journal reads); the
// capability is NOT silently emulated, because a full-journal scan hidden
// behind an O(1)-looking method is a latency trap.
func (a *EventAdapter) LoadByEventID(
	ctx context.Context,
	eventID id.EventID,
) (event.Event, error) {
	if a.byID == nil {
		return nil, ErrLoadByEventIDUnsupported
	}

	value, err := a.byID.StreamLoadByEventID(ctx, a.Collection, eventID.String())
	if err != nil {
		if errors.Is(err, metaengine.ErrNotFound) {
			return nil, event.ErrEventNotFound
		}

		return nil, fmt.Errorf("event adapter: load by event ID: %w", err)
	}

	evt, err := a.decodeValue(value)
	if err != nil {
		return nil, fmt.Errorf("event adapter: decode event %s: %w", eventID, err)
	}

	return evt, nil
}
