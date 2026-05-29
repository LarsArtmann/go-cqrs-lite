package pebble

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event"
)

// AppendBatch implements event.Store.AppendBatch.
// Appends events without optimistic concurrency checks.
func (a *PebbleEventStore) AppendBatch(
	_ context.Context,
	ref event.AggregateRef,
	events []event.Event,
) error {
	if len(events) == 0 {
		return nil
	}

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	for _, evt := range events {
		key := a.eventKey(ref, evt.Version())

		err := a.serializeAndAddToBatch(batch, key, evt)
		if err != nil {
			return event.WrapCorruption(err, "pebble.serialize_event",
				fmt.Sprintf("serialize event %s for %s %s", evt.Type(), ref.Type, ref.ID))
		}
	}

	return a.commitAndLog(
		batch,
		"events appended in batch",
		ref,
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
	ref event.AggregateRef,
	count int,
) {
	a.logger.Debug(
		msg,
		slog.String("aggregate_type", string(ref.Type)),
		slog.String("aggregate_id", ref.ID.String()),
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

func (a *PebbleEventStore) writeOptions() *pebble.WriteOptions {
	if a.syncWrites {
		return pebble.Sync
	}

	return nil
}

// commitAndLog commits the batch and logs the operation.
func (a *PebbleEventStore) commitAndLog(
	batch *pebble.Batch,
	logMsg string,
	ref event.AggregateRef,
	count int,
) error {
	err := batch.Commit(a.writeOptions())
	if err != nil {
		return event.WrapInfrastructure(err, "pebble.commit_batch",
			fmt.Sprintf("failed to commit %d events (%s)", count, logMsg))
	}

	a.logEventOperation(logMsg, ref, count)

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
