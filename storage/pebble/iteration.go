package pebble

import (
	"context"
	"time"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// eventPredicate returns true when iteration should stop BEFORE appending
// the current event. nil means "never stop early" (collect all events).
type eventPredicate func(event.Event) bool

// iterateEvents iterates over events in the database using the provided
// iterator configuration. If shouldStop is non-nil, iteration stops before
// appending the first event for which shouldStop returns true.
func (a *EventStore) iterateEvents(
	lowerBound, upperBound []byte,
	shouldStop eventPredicate,
) ([]event.Event, error) {
	iter, err := a.db.NewIter(
		&pebble.IterOptions{
			LowerBound: lowerBound,
			UpperBound: upperBound,
		},
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "pebble.create_iterator",
			"failed to create iterator")
	}

	defer func() { _ = iter.Close() }()

	var events []event.Event

	for iter.First(); iter.Valid(); iter.Next() {
		evt, err := a.deserializeEvent(iter.Value())
		if err != nil {
			return nil, a.corruptEventErr(string(iter.Key()), err)
		}

		if shouldStop != nil && shouldStop(evt) {
			break
		}

		events = append(events, evt)
	}

	return events, checkIteratorError(iter)
}

// Load implements event.Store.Load.
func (a *EventStore) Load(
	ctx context.Context,
	ref id.StreamRef,
) ([]event.Event, error) {
	span, prefix, upperBound := a.startLoadSpan(ctx, "pebble.event.load", ref)
	defer span.End()

	events, err := a.iterateEvents(prefix, upperBound, nil)

	return finalizeScan(span, events, err, "pebble.event_load",
		"load events for aggregate", "event.count")
}

// LoadFromVersion implements event.Store.LoadFromVersion.
// Returns events with version strictly greater than the given version.
func (a *EventStore) LoadFromVersion(
	ctx context.Context,
	ref id.StreamRef,
	version event.Version,
) ([]event.Event, error) {
	span, lowerBound, upperBound := a.startLoadFromVersionSpan(
		ctx,
		"pebble.event.load_from_version",
		ref,
		version,
	)
	defer span.End()

	events, err := a.iterateEvents(lowerBound, upperBound, nil)

	return finalizeScan(span, events, err, "pebble.event_load_from_version",
		"load events from version", "event.count")
}

// loadFiltered iterates events and returns them filtered by predicate.
// Returns ErrStreamNotFound if no events match the filter.
func (a *EventStore) loadFiltered(
	ref id.StreamRef,
	upperBound []byte,
	predicate eventPredicate,
) ([]event.Event, error) {
	prefix := a.aggregatePrefix(ref)

	events, err := a.iterateEvents(prefix, upperBound, predicate)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "pebble.event_load_filtered",
			"load filtered events")
	}

	if len(events) == 0 {
		return nil, event.ErrStreamNotFound
	}

	return events, nil
}

// LoadToVersion retrieves events up to and including maxVersion.
func (a *EventStore) LoadToVersion(
	ctx context.Context,
	ref id.StreamRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	_, span := startAggregateSpan(ctx, "pebble.event.load_to_version", ref,
		cqrsotel.AttrInt(cqrsotel.AttrStreamVersion, maxVersion.Int()))
	defer span.End()

	upperBound := a.eventKey(ref, maxVersion+1)

	events, err := a.loadFiltered(ref, upperBound, nil)

	return finalizeScan(span, events, err, "pebble.event_load_to_version",
		"load events up to version", "event.count")
}

// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
// Uses early termination: since events are stored in version order and
// OccurredAt is monotonically increasing, the iterator stops as soon as it
// encounters an event past maxTime — avoiding a full aggregate scan.
func (a *EventStore) LoadToTimestamp(
	ctx context.Context,
	ref id.StreamRef,
	maxTime time.Time,
) ([]event.Event, error) {
	_, span := startAggregateSpan(ctx, "pebble.event.load_to_timestamp", ref)
	defer span.End()

	upperBound := a.aggregateUpperBound(ref)
	predicate := func(evt event.Event) bool {
		return evt.OccurredAt().After(maxTime)
	}

	events, err := a.loadFiltered(ref, upperBound, predicate)

	return finalizeScan(span, events, err, "pebble.event_load_to_timestamp",
		"load events up to timestamp", "event.count")
}
