package pebble

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
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
func (a *EventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.journal.read_all",
		cqrsotel.SpanKindClient)
	defer span.End()

	lowerBound := []byte(a.journalPrefix)
	upperBound := []byte(a.journalPrefix + "\xff")

	events, err := a.iterateEvents(lowerBound, upperBound, nil)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(cqrsotel.AttrInt("event.count", len(events)))

	return events, nil
}

// ReadFrom retrieves events ordered by OccurredAt, starting after the given event ID.
// Implements event.SeekableJournal for efficient projection catch-up.
//
// Optimization: when afterEventID is non-zero, the ULID timestamp embedded in
// the ID narrows the iterator's lower bound. This avoids scanning the entire
// journal on every catch-up page. A 1-minute buffer ensures correctness even
// when OccurredAt differs from the ULID generation time. If the target is not
// found in the narrowed range, a full scan is performed as fallback.
func (a *EventStore) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.journal.read_from",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("limit", limit)))
	defer span.End()

	upperBound := []byte(a.journalPrefix + "\xff")

	// Fast path: no afterEventID, collect from beginning.
	if afterEventID.IsZero() {
		events, _, err := a.scanJournalWithSkip([]byte(a.journalPrefix), upperBound, "", limit)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return nil, err
		}

		span.SetAttributes(cqrsotel.AttrInt("event.count", len(events)))

		return events, nil
	}

	// Fast path: narrow lower bound using ULID timestamp.
	targetID := afterEventID.String()

	ulidTime := id.ULID(afterEventID)
	narrowedLower := fmt.Appendf(nil, "%s%020d", a.journalPrefix,
		ulidTime.Add(-journalSeekBuffer).UnixNano())

	events, found, err := a.scanJournalWithSkip(narrowedLower, upperBound, targetID, limit)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	// Fallback: target not in narrowed range (rare edge case — backdated event).
	if !found {
		events, _, err = a.scanJournalWithSkip([]byte(a.journalPrefix), upperBound,
			targetID, limit)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return nil, err
		}
	}

	span.SetAttributes(cqrsotel.AttrInt("event.count", len(events)))

	return events, nil
}

// journalSeekBuffer is subtracted from the ULID timestamp to create a
// conservative lower bound for narrowed journal scans.
const journalSeekBuffer = time.Minute

// scanJournalWithSkip scans between bounds. If targetID is non-empty, events
// are skipped until the matching journal key is found.
// Returns (events, targetFound, error).
func (a *EventStore) scanJournalWithSkip(
	lowerBound, upperBound []byte,
	targetID string,
	limit int,
) ([]event.Event, bool, error) {
	iter, err := a.db.NewIter(
		&pebble.IterOptions{
			LowerBound: lowerBound,
			UpperBound: upperBound,
		},
	)
	if err != nil {
		return nil, false, event.WrapInfrastructure(err, "pebble.scan_journal",
			"create iterator")
	}

	defer func() { _ = iter.Close() }()

	skipping := targetID != ""
	found := false

	var events []event.Event

	for iter.First(); iter.Valid(); iter.Next() {
		if skipping {
			if journalKeyEventID(iter.Key()) == targetID {
				skipping = false
				found = true
			}

			continue
		}

		evt, err := a.deserializeEvent(iter.Value())
		if err != nil {
			return nil, found, event.Wrapf(
				a.corruptEventErr(string(iter.Key()), err),
				event.Corruption, "pebble.journal_corrupt_event",
				"corrupt event in journal (limit=%d)", limit,
			)
		}

		events = append(events, evt)

		if limit > 0 && len(events) >= limit {
			break
		}
	}

	err = checkIteratorError(iter)
	if err != nil {
		return nil, found, event.Wrapf(err, event.Infrastructure, "pebble.journal_iterator",
			"iterator error in journal (limit=%d)", limit)
	}

	return events, found, nil
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
