package pebble

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// AppendBatch implements event.Store.AppendBatch.
// Appends events without optimistic concurrency checks.
// cqrs-lint:ignore(A021) library code or intentional pattern
func (a *EventStore) AppendBatch(
	_ context.Context,
	ref id.StreamRef,
	events []event.Event,
) error {
	if len(events) == 0 {
		return nil
	}

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	for _, evt := range events {
		key := a.eventKey(ref, evt.Version())

		err := a.serializeAndAddToBatchWithJournal(batch, key, evt)
		if err != nil {
			return errorfamily.WrapCorruption(err, "pebble.serialize_event",
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

// Close is a no-op; the underlying *pebble.DB is owned by the caller (or Backend).
// Implemented to satisfy io.Closer for event.EventSink/EventSource.
// Only [Backend.Close] closes the *pebble.DB.
func (a *EventStore) Close() error { return nil }

// logEventOperation logs a debug message for event operations.
func (a *EventStore) logEventOperation(
	msg string,
	ref id.StreamRef,
	count int,
) {
	if a.logger == nil {
		return
	}

	a.logger.Debug(
		msg,
		slog.String("aggregate_type", string(ref.Type)),
		slog.String("aggregate_id", ref.ID.String()),
		slog.Int("count", count),
	)
}

// serializeAndAddToBatchWithJournal serializes an event once and writes it to
// both the stream event key and the global journal key. This eliminates the
// duplicate serialization that previously occurred when appendToJournal
// re-serialized every event from scratch.
func (a *EventStore) serializeAndAddToBatchWithJournal(
	batch *pebble.Batch,
	eventKey []byte,
	evt event.Event,
) error {
	data, err := a.serializeEvent(evt)
	if err != nil {
		return errorfamily.WrapCorruption(err, "pebble.serialize_event",
			"failed to serialize event")
	}

	err = a.addToBatch(batch, eventKey, data)
	if err != nil {
		return err
	}

	journalKey := a.journalKey(evt)

	return a.addToBatch(batch, journalKey, data)
}

// addToBatch is a helper that adds a key-value pair to a batch with error handling.
func (a *EventStore) addToBatch(batch *pebble.Batch, key, data []byte) error {
	err := batch.Set(key, data, nil)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.add_to_batch",
			"failed to add event to batch")
	}

	return nil
}

// commitAndLog commits the batch and logs the operation.
func (a *EventStore) commitAndLog(
	batch *pebble.Batch,
	logMsg string,
	ref id.StreamRef,
	count int,
) error {
	err := batch.Commit(a.writeOptions())
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.commit_batch",
			fmt.Sprintf("failed to commit %d events (%s)", count, logMsg))
	}

	a.logEventOperation(logMsg, ref, count)

	return nil
}

// corruptEventErr logs a warning and returns a corruption error for a corrupt event.
func (a *EventStore) corruptEventErr(key string, err error) error {
	if a.logger != nil {
		a.logger.Warn("corrupt event in pebble store", "key", key, "error", err)
	}

	return errorfamily.WrapCorruption(err, "pebble.corrupt_event",
		"corrupt event at key "+key)
}

// checkIteratorError checks an iterator for errors and returns an appropriate error.
func checkIteratorError(iter *pebble.Iterator) error {
	return wrapInfraOrOK(iter.Error(), "pebble.iterator_error", "iterator error")
}

// keyExists reports whether the given key is present in the database. Returns
// true for any non-ErrNotFound error — the caller treats unknown errors as
// "key might exist, don't risk a duplicate write". Shared by CommandStore
// and QueryStore for their idempotent Save paths.
func keyExists(db *pebble.DB, key []byte) bool {
	_, closer, err := db.Get(key)
	if err == nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = closer.Close()

		return true
	}

	return !errors.Is(err, pebble.ErrNotFound)
}

// lastSegmentAfterByte returns the substring after the last occurrence of sep
// in key, or the whole key if sep is not found. Shared by journal-key ID extractors.
func lastSegmentAfterByte(key []byte, sep byte) string {
	if i := bytes.LastIndexByte(key, sep); i >= 0 {
		return string(key[i+1:])
	}

	return string(key)
}

// wrapInfraOrOK returns nil when err is nil, otherwise wraps err as an
// infrastructure error with the given code and message. Collapses the
// repeated "if err != nil { return WrapInfrastructure(...) }; return nil"
// boilerplate — the unique code stays a parameter.
func wrapInfraOrOK(err error, code, msg string) error {
	if err == nil {
		return nil
	}

	return errorfamily.WrapInfrastructure(err, code, msg)
}

// closeAndWrap closes db and wraps any failure with the given errorfamily code
// and message. Shared by KVAdapter.Close and Backend.Close.
func closeAndWrap(db *pebble.DB, code, msg string) error {
	return wrapInfraOrOK(db.Close(), code, msg)
}

// Ensure EventStore implements event.Store.
var _ event.Store = (*EventStore)(nil)

// Ensure EventStore implements event.Journal and event.SeekableJournal.
var (
	_ event.Journal         = (*EventStore)(nil)
	_ event.SeekableJournal = (*EventStore)(nil)
)
