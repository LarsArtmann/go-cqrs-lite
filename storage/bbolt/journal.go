package bbolt

import (
	"context"

	bolt "go.etcd.io/bbolt"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ReadAll retrieves all events across all streams, ordered by OccurredAt.
// Implements event.Journal by scanning the journal bucket.
func (s *EventStore) ReadAll(_ context.Context) ([]event.Event, error) {
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

	return events, err
}

// ReadFrom retrieves events ordered by OccurredAt, starting after afterEventID.
// Implements event.SeekableJournal for projection catch-up.
func (s *EventStore) ReadFrom(
	_ context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
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

	return events, err
}
