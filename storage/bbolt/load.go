package bbolt

import (
	"bytes"
	"context"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Load implements event.Store.Load — returns all events for a stream.
func (s *EventStore) Load(
	_ context.Context,
	ref id.StreamRef,
) ([]event.Event, error) {
	var events []event.Event

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketEvents))
		if bucket == nil {
			return nil
		}

		prefix := streamPrefix(ref)
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			evt, err := deserializeEvent(v)
			if err != nil {
				return errorfamily.WrapCorruption(err, "bbolt.load_corrupt",
					"corrupt event at key "+string(k))
			}

			events = append(events, evt)
		}

		return nil
	})

	return events, wrapBucketErr(err, "bbolt.load", "load events")
}

// LoadFromVersion returns events with version strictly greater than version.
func (s *EventStore) LoadFromVersion(
	_ context.Context,
	ref id.StreamRef,
	version event.Version,
) ([]event.Event, error) {
	lower := eventKey(ref, version+1)
	prefix := streamPrefix(ref)

	var events []event.Event

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketEvents))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()

		for k, v := c.Seek(lower); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			evt, err := deserializeEvent(v)
			if err != nil {
				return errorfamily.WrapCorruption(err, "bbolt.load_from_version_corrupt",
					"corrupt event at key "+string(k))
			}

			events = append(events, evt)
		}

		return nil
	})

	return events, wrapBucketErr(err, "bbolt.load_from_version", "load events from version")
} returns events up to and including maxVersion.
func (s *EventStore) LoadToVersion(
	_ context.Context,
	ref id.StreamRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	upper := eventKey(ref, maxVersion+1)
	prefix := streamPrefix(ref)

	var events []event.Event

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketEvents))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			if string(k) >= string(upper) {
				break
			}

			evt, err := deserializeEvent(v)
			if err != nil {
				return errorfamily.WrapCorruption(err, "bbolt.load_to_version_corrupt",
					"corrupt event at key "+string(k))
			}

			events = append(events, evt)
		}

		return nil
	})
	if err != nil {
		return nil, wrapBucketErr(err, "bbolt.load_to_version",
			"load events to version")
	} returns events where OccurredAt <= maxTime.
func (s *EventStore) LoadToTimestamp(
	_ context.Context,
	ref id.StreamRef,
	maxTime time.Time,
) ([]event.Event, error) {
	prefix := streamPrefix(ref)

	var events []event.Event

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketEvents))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			evt, err := deserializeEvent(v)
			if err != nil {
				return errorfamily.WrapCorruption(err, "bbolt.load_to_ts_corrupt",
					"corrupt event at key "+string(k))
			}

			if evt.OccurredAt().After(maxTime) {
				break
			}

			events = append(events, evt)
		}

		return nil
	})
	if err != nil {
		return nil, wrapBucketErr(err, "bbolt.load_to_timestamp",
			"load events to timestamp")
	}

	if len(events) == 0 {
		return nil, event.ErrStreamNotFound
	}

	return events, nil
}
