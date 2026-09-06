package bbolt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	bolt "go.etcd.io/bbolt"
)

// ReadAll retrieves all events across all streams, ordered by OccurredAt.
// Implements event.Journal by scanning the journal bucket.
func (s *EventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	span := startReadSpan(ctx, "bbolt.journal.read_all")
	defer span.End()

	var events []event.Event

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketJournal))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			evt, err := deserializeEvent(v)
			if err != nil {
				return errorfamily.WrapCorruption(err, "bbolt.journal_corrupt",
					"corrupt event at journal key "+string(k))
			}

			events = append(events, evt)
		}

		return nil
	})

	return finalizeScan(span, events, err, "bbolt.journal_read_all",
		"read all events", "event.count")
}

// ReadFrom retrieves events ordered by OccurredAt, starting after afterEventID.
// Implements event.SeekableJournal for projection catch-up.
func (s *EventStore) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	span := startLimitSpan(ctx, "bbolt.journal.read_from", limit)
	defer span.End()

	var events []event.Event

	skipTarget := ""

	if !afterEventID.IsZero() {
		skipTarget = afterEventID.String()
	}

	skipping := skipTarget != ""

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketJournal))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			if skipping {
				if journalKeyEventID(k) == skipTarget {
					skipping = false
				}

				continue
			}

			evt, err := deserializeEvent(v)
			if err != nil {
				return errorfamily.WrapCorruption(err, "bbolt.journal_corrupt",
					"corrupt event at journal key "+string(k))
			}

			events = append(events, evt)

			if limit > 0 && len(events) >= limit {
				break
			}
		}

		return nil
	})

	return finalizeScan(span, events, err, "bbolt.journal_read_from",
		"read events from position", "event.count")
}
