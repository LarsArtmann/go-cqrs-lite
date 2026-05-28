package storage

import (
	"fmt"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func (a *PebbleEventStore) checkVersion(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	expectedVersion event.Version,
) error {
	prefix := a.aggregatePrefix(aggregateType, aggregateID)
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, aggregateType, aggregateID)

	existing, err := a.iterateEvents(prefix, upperBound)
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

func (a *PebbleEventStore) writeEventsToBatch(
	batch *pebble.Batch,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	for i, evt := range events {
		err := validateEventOwnership(evt, aggregateType, aggregateID)
		if err != nil {
			return event.WrapCorruption(err, "pebble.validate_event",
			fmt.Sprintf("validate event %d", i))
		}

		expectedEventVersion := expectedVersion.Int() + i + 1
		if evt.Version() != event.Version(expectedEventVersion) {
			return event.WrapConflict(ErrVersionMismatch, "pebble.version_mismatch",
				fmt.Sprintf("expected %d, got %d", expectedEventVersion, evt.Version()))
		}

		key := a.eventKey(aggregateType, aggregateID, event.Version(expectedEventVersion))

		err = a.serializeAndAddToBatch(batch, key, evt)
		if err != nil {
			return event.WrapCorruption(err, "pebble.serialize_event",
				fmt.Sprintf("serialize event %d for %s %s", i, aggregateType, aggregateID))
		}
	}

	return nil
}

func validateEventOwnership(
	evt event.Event,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	if evt.AggregateType() != aggregateType {
		return event.WrapConflict(ErrAggregateTypeMismatch, "pebble.aggregate_type_mismatch",
			fmt.Sprintf("expected %s, got %s", aggregateType, evt.AggregateType()))
	}

	if evt.AggregateID() != aggregateID {
		return event.WrapConflict(ErrAggregateIDMismatch, "pebble.aggregate_id_mismatch",
			fmt.Sprintf("expected %s, got %s", aggregateID, evt.AggregateID()))
	}

	return nil
}
