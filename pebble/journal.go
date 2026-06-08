package pebble

import (
	"context"
	"fmt"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// journalKey generates a globally-ordered key for the journal index.
// Pattern: cqrs_journal:{occurredAt_unix_nano_20d}:{eventID}
// The timestamp is zero-padded to 20 digits for lexicographic ordering.
// The event ID provides uniqueness for events occurring at the same nanosecond.
func (a *EventStore) journalKey(evt event.Event) []byte {
	return fmt.Appendf(
		nil, "%s%020d:%s", a.journalPrefix,
		evt.OccurredAt().UnixNano(), evt.ID().String(),
	)
}

// appendToJournal writes journal entries for each event into the batch.
func (a *EventStore) appendToJournal(batch *pebble.Batch, events []event.Event) error {
	for _, evt := range events {
		jKey := a.journalKey(evt)

		data, err := a.serializeEvent(evt)
		if err != nil {
			return event.WrapCorruption(err, "pebble.journal_serialize",
				"serialize event for journal "+evt.ID().String())
		}

		err = a.addToBatch(batch, jKey, data)
		if err != nil {
			return event.WrapInfrastructure(err, "pebble.journal_write",
				"write journal entry for "+evt.ID().String())
		}
	}

	return nil
}

// ReadAll retrieves all events across all aggregates, ordered by OccurredAt.
// Implements event.Journal by scanning the journal key prefix.
func (a *EventStore) ReadAll(_ context.Context) ([]event.Event, error) {
	lowerBound := []byte(a.journalPrefix)
	upperBound := []byte(a.journalPrefix + "\xff")

	return a.iterateEvents(lowerBound, upperBound, nil)
}

// ReadFrom retrieves events ordered by OccurredAt, starting after the given event ID.
// Implements event.SeekableJournal for efficient projection catch-up.
func (a *EventStore) ReadFrom(
	_ context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	var events []event.Event

	lowerBound := []byte(a.journalPrefix)
	upperBound := []byte(a.journalPrefix + "\xff")

	iter, err := a.db.NewIter(
		&pebble.IterOptions{ //nolint:exhaustruct // only Lower/Upper bound needed
			LowerBound: lowerBound,
			UpperBound: upperBound,
		},
	)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "pebble.read_from",
			"create iterator for ReadFrom")
	}

	defer func() { _ = iter.Close() }()

	skipping := !afterEventID.IsZero()

	for iter.First(); iter.Valid(); iter.Next() {
		evt, err := a.deserializeEvent(iter.Value())
		if err != nil {
			if a.logger != nil {
				a.logger.Warn("corrupt journal event in pebble store",
					"key", string(iter.Key()), "error", err)
			}

			return nil, event.WrapCorruption(err, "pebble.corrupt_journal_event",
				"corrupt journal event at key "+string(iter.Key()))
		}

		if skipping {
			if evt.ID() == afterEventID {
				skipping = false
			}

			continue
		}

		events = append(events, evt)

		if limit > 0 && len(events) >= limit {
			break
		}
	}

	err = checkIteratorError(iter)
	if err != nil {
		return nil, err
	}

	return events, nil
}
