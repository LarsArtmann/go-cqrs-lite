package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Delete removes all events for an aggregate from the Pebble store.
// Not part of the event.Store interface — kept as a utility for testing.
func (a *PebbleEventStore) Delete(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	prefix := a.aggregatePrefix(aggregateType, aggregateID)
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, aggregateType, aggregateID)

	iter, err := a.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return event.WrapInfrastructure(err, "pebble.create_iterator",
			"failed to create iterator")
	}

	defer func() { _ = iter.Close() }()

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		err := batch.Delete(iter.Key(), nil)
		if err != nil {
			return event.WrapInfrastructure(err, "pebble.delete_event",
				"failed to delete event")
		}

		count++
	}

	commitErr := batch.Commit(pebble.Sync)
	if commitErr != nil {
		return event.WrapInfrastructure(commitErr, "pebble.commit_deletions",
			"failed to commit deletions")
	}

	a.logger.Debug(
		"events deleted",
		slog.String("aggregate_type", string(aggregateType)),
		slog.String("aggregate_id", aggregateID.String()),
		slog.Int("count", count),
	)

	return nil
}

// AppendBatch implements event.Store.AppendBatch.
// Appends events without optimistic concurrency checks.
func (a *PebbleEventStore) AppendBatch(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	if len(events) == 0 {
		return nil
	}

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	for _, evt := range events {
		key := a.eventKey(aggregateType, aggregateID, evt.Version())

		err := a.serializeAndAddToBatch(batch, key, evt)
		if err != nil {
			return event.WrapCorruption(err, "pebble.serialize_event",
				fmt.Sprintf("serialize event %s for %s %s", evt.Type(), aggregateType, aggregateID))
		}
	}

	return a.commitAndLog(
		batch,
		"events appended in batch",
		aggregateType,
		aggregateID,
		len(events),
	)
}

// Close releases the Pebble database.
func (a *PebbleEventStore) Close() error {
	if a.db != nil {
		err := a.db.Close()
		if err != nil {
			return event.WrapInfrastructure(err, "pebble.close_db",
				"close pebble db")
		}
	}

	return nil
}

// logEventOperation logs a debug message for event operations.
func (a *PebbleEventStore) logEventOperation(
	msg string,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	count int,
) {
	a.logger.Debug(
		msg,
		slog.String("aggregate_type", string(aggregateType)),
		slog.String("aggregate_id", aggregateID.String()),
		slog.Int("count", count),
	)
}

// serializeAndAddToBatch serializes an event and adds it to the batch.
func (a *PebbleEventStore) serializeAndAddToBatch(
	batch *pebble.Batch,
	key []byte,
	evt event.Event,
) error {
	data, err := a.serializeEvent(evt)
	if err != nil {
		return event.WrapCorruption(err, "pebble.serialize_event",
			"failed to serialize event")
	}

	return a.addToBatch(batch, key, data)
}

// addToBatch is a helper that adds a key-value pair to a batch with error handling.
func (a *PebbleEventStore) addToBatch(batch *pebble.Batch, key, data []byte) error {
	err := batch.Set(key, data, nil)
	if err != nil {
		return event.WrapInfrastructure(err, "pebble.add_to_batch",
			"failed to add event to batch")
	}

	return nil
}

// commitAndLog commits the batch and logs the operation.
func (a *PebbleEventStore) commitAndLog(
	batch *pebble.Batch,
	logMsg string,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	count int,
) error {
	err := batch.Commit(pebble.Sync)
	if err != nil {
		return event.WrapInfrastructure(err, "pebble.commit_batch",
			fmt.Sprintf("failed to commit %d events (%s)", count, logMsg))
	}

	a.logEventOperation(logMsg, aggregateType, aggregateID, count)

	return nil
}

// checkIteratorError checks an iterator for errors and returns an appropriate error.
func checkIteratorError(iter *pebble.Iterator) error {
	err := iter.Error()
	if err != nil {
		return event.WrapInfrastructure(err, "pebble.iterator_error",
			"iterator error")
	}

	return nil
}

// Ensure PebbleEventStore implements event.Store.
var _ event.Store = (*PebbleEventStore)(nil)
