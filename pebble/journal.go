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
	targetID := afterEventID.String()

	for iter.First(); iter.Valid(); iter.Next() {
		if skipping {
			if journalKeyEventID(iter.Key()) == targetID {
				skipping = false
			}

			continue
		}

		evt, err := a.deserializeEvent(iter.Value())
		if err != nil {
			return nil, fmt.Errorf(
				"corrupt event in journal (limit=%d, after=%s): %w",
				limit,
				afterEventID,
				a.corruptEventErr(string(iter.Key()), err),
			)
		}

		events = append(events, evt)

		if limit > 0 && len(events) >= limit {
			break
		}
	}

	err = checkIteratorError(iter)
	if err != nil {
		return nil, fmt.Errorf(
			"iterator error in journal (limit=%d, after=%s): %w",
			limit,
			afterEventID,
			err,
		)
	}

	return events, nil
}

// journalKeyEventID extracts the event ID portion from a journal key.
// Journal key format: {prefix}{020d_timestamp}:{eventID}.
func journalKeyEventID(key []byte) string {
	for i := len(key) - 1; i >= 0; i-- { //nolint:modernize // reverse scan is clearer here
		if key[i] == ':' {
			return string(key[i+1:])
		}
	}

	return string(key)
}
