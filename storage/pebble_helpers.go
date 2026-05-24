package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Delete implements event.Store.Delete.
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
		return fmt.Errorf("failed to create iterator: %w", err)
	}

	defer func() { _ = iter.Close() }()

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		err := batch.Delete(iter.Key(), nil)
		if err != nil {
			return fmt.Errorf("failed to delete event: %w", err)
		}

		count++
	}

	commitErr := batch.Commit(pebble.Sync)
	if commitErr != nil {
		return fmt.Errorf("failed to commit deletions: %w", commitErr)
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
			return fmt.Errorf("serialize event %s for %s %s: %w", evt.Type(), aggregateType, aggregateID, err)
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
			return fmt.Errorf("close pebble db: %w", err)
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
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	return a.addToBatch(batch, key, data)
}

// addToBatch is a helper that adds a key-value pair to a batch with error handling.
func (a *PebbleEventStore) addToBatch(batch *pebble.Batch, key, data []byte) error {
	err := batch.Set(key, data, nil)
	if err != nil {
		return fmt.Errorf("failed to add event to batch: %w", err)
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
		return fmt.Errorf("failed to commit %d events (%s): %w", count, logMsg, err)
	}

	a.logEventOperation(logMsg, aggregateType, aggregateID, count)

	return nil
}

// checkIteratorError checks an iterator for errors and returns an appropriate error.
func checkIteratorError(iter *pebble.Iterator) error {
	err := iter.Error()
	if err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return nil
}

// Ensure PebbleEventStore implements event.Store.
var _ event.Store = (*PebbleEventStore)(nil)
