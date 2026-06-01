package pebble

import (
	"fmt"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func (a *EventStore) checkVersion(
	ref event.AggregateRef,
	expectedVersion event.Version,
) error {
	prefix := a.aggregatePrefix(ref)
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, ref.Type, ref.ID)

	existing, err := a.iterateEvents(prefix, upperBound, nil)
	if err != nil {
		return event.WrapInfrastructure(err, "pebble.concurrency_check",
			"concurrency check")
	}

	err = event.CheckVersionConflict(len(existing), expectedVersion)
	if err != nil {
		return event.WrapConflict(err, "pebble.version_conflict",
			"concurrency check")
	}

	return nil
}

func (a *EventStore) writeEventsToBatch(
	batch *pebble.Batch,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	for i, evt := range events {
		err := validateEventOwnership(evt, ref)
		if err != nil {
			return event.WrapCorruption(err, "pebble.validate_event",
				fmt.Sprintf("validate event %d", i))
		}

		expectedEventVersion := expectedVersion.Int() + i + 1
		if evt.Version() != event.Version(expectedEventVersion) {
			return event.WrapConflict(ErrVersionMismatch, "pebble.version_mismatch",
				fmt.Sprintf("expected %d, got %d", expectedEventVersion, evt.Version()))
		}

		key := a.eventKey(ref, event.Version(expectedEventVersion))

		err = a.serializeAndAddToBatch(batch, key, evt)
		if err != nil {
			return event.WrapCorruption(err, "pebble.serialize_event",
				fmt.Sprintf("serialize event %d for %s %s", i, ref.Type, ref.ID))
		}
	}

	return nil
}

func validateEventOwnership(
	evt event.Event,
	ref event.AggregateRef,
) error {
	if evt.AggregateType() != ref.Type {
		return event.WrapConflict(ErrAggregateTypeMismatch, "pebble.aggregate_type_mismatch",
			fmt.Sprintf("expected %s, got %s", ref.Type, evt.AggregateType()))
	}

	if evt.AggregateID() != ref.ID {
		return event.WrapConflict(ErrAggregateIDMismatch, "pebble.aggregate_id_mismatch",
			fmt.Sprintf("expected %s, got %s", ref.ID, evt.AggregateID()))
	}

	return nil
}
