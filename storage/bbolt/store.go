package bbolt

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strconv"

	errorfamily "github.com/larsartmann/go-error-family"
	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// EventStore implements event.Store using an embedded bbolt database.
//
// bbolt uses a single-writer model (one read-write transaction at a time),
// so Save does not need per-stream locking — the DB serializes all writes
// and the version check + write happen atomically inside one transaction.
type EventStore struct {
	storeBase
}

// StoreOption configures an EventStore.
type StoreOption func(*EventStore)

// NewStore wraps an existing *bbolt.DB into an EventStore.
// Returns ErrNilDatabase if db is nil.
func NewStore(database *bolt.DB, logger *slog.Logger, _ ...StoreOption) (*EventStore, error) {
	if database == nil {
		return nil, ErrNilDatabase
	}

	return &EventStore{storeBase: storeBase{db: database, logger: logger}}, nil
}

// eventKey builds the in-bucket key for an event.
// Pattern: {streamType}:{streamID}:{010d_version}
func eventKey(ref id.StreamRef, version event.Version) []byte {
	return fmt.Appendf(nil, "%s:%s:%010d", ref.Type, ref.ID, version.Int())
}

// streamPrefix returns the byte prefix for all events of one stream.
func streamPrefix(ref id.StreamRef) []byte {
	return fmt.Appendf(nil, "%s:%s:", ref.Type, ref.ID)
}

// streamUpperBound returns the exclusive upper bound for all events of a
// stream. The 0xff byte sorts after any version digit.
func streamUpperBound(ref id.StreamRef) []byte {
	return fmt.Appendf(nil, "%s:%s:\xff", ref.Type, ref.ID)
}

// journalKey builds the globally-ordered key for the journal index.
// Pattern: {020d_unixnano}:{eventID}
func journalKey(evt event.Event) []byte {
	return fmt.Appendf(nil, "%020d:%s", evt.OccurredAt().UnixNano(), evt.ID().String())
}

func journalKeyEventID(key []byte) string {
	if i := bytes.LastIndexByte(key, ':'); i >= 0 {
		return string(key[i+1:])
	}

	return string(key)
}

// Save implements event.Store.Save. Version checking and event writing happen
// in a single atomic bbolt write transaction — no per-stream lock needed.
func (s *EventStore) Save(
	_ context.Context,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if len(events) == 0 {
		return nil
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		currentVersion, err := s.currentVersion(tx, ref)
		if err != nil {
			return err
		}

		if err := event.CheckVersionConflict(currentVersion, expectedVersion); err != nil {
			return errorfamily.WrapConflict(err, "bbolt.version_conflict", "concurrency check")
		}

		eventsBucket := tx.Bucket([]byte(bucketEvents))
		journalBucket := tx.Bucket([]byte(bucketJournal))

		for i, evt := range events {
			if err := validateEventOwnership(evt, ref); err != nil {
				return errorfamily.WrapCorruption(err, "bbolt.validate_event",
					fmt.Sprintf("validate event %d", i))
			}

			expectedEvtVersion := expectedVersion.Int() + i + 1
			if evt.Version() != event.Version(expectedEvtVersion) {
				return errorfamily.WrapConflict(ErrVersionMismatch, "bbolt.version_mismatch",
					fmt.Sprintf("expected %d, got %d", expectedEvtVersion, evt.Version()))
			}

			data, err := serializeEvent(evt)
			if err != nil {
				return errorfamily.WrapCorruption(err, "bbolt.serialize_event",
					fmt.Sprintf("serialize event %d for %s %s", i, ref.Type, ref.ID))
			}

			ek := eventKey(ref, event.Version(expectedEvtVersion))
			if err := eventsBucket.Put(ek, data); err != nil {
				return errorfamily.WrapInfrastructure(err, "bbolt.put_event",
					fmt.Sprintf("put event %d", i))
			}

			jk := journalKey(evt)
			if err := journalBucket.Put(jk, data); err != nil {
				return errorfamily.WrapInfrastructure(err, "bbolt.put_journal",
					fmt.Sprintf("put journal entry %d", i))
			}
		}

		return nil
	})
}

// AppendBatch implements event.Store.AppendBatch (no version check).
// cqrs-lint:ignore(A021) library code or intentional pattern
func (s *EventStore) AppendBatch(
	_ context.Context,
	ref id.StreamRef,
	events []event.Event,
) error {
	if len(events) == 0 {
		return nil
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		eventsBucket := tx.Bucket([]byte(bucketEvents))
		journalBucket := tx.Bucket([]byte(bucketJournal))

		for _, evt := range events {
			data, err := serializeEvent(evt)
			if err != nil {
				return errorfamily.WrapCorruption(err, "bbolt.serialize_event",
					"serialize event "+evt.ID().String())
			}

			ek := eventKey(ref, evt.Version())
			if err := eventsBucket.Put(ek, data); err != nil {
				return errorfamily.WrapInfrastructure(err, "bbolt.put_event",
					"put event "+evt.ID().String())
			}

			jk := journalKey(evt)
			if err := journalBucket.Put(jk, data); err != nil {
				return errorfamily.WrapInfrastructure(err, "bbolt.put_journal",
					"put journal "+evt.ID().String())
			}
		}

		return nil
	})
}

// Close is a no-op; the underlying *bbolt.DB is owned by the caller or Backend.
func (s *EventStore) Close() error { return nil }

// currentVersion returns the highest event version for a stream, or 0 if none.
// Uses cursor.Seek to the stream upper bound, then Prev() for O(log N) lookup.
func (s *EventStore) currentVersion(tx *bolt.Tx, ref id.StreamRef) (int, error) {
	bucket := tx.Bucket([]byte(bucketEvents))
	if bucket == nil {
		return 0, nil
	}

	c := bucket.Cursor()
	c.Seek(streamUpperBound(ref))
	k, _ := c.Prev()

	if k == nil {
		return 0, nil
	}

	prefix := streamPrefix(ref)
	if len(k) < len(prefix) || string(k[:len(prefix)]) != string(prefix) {
		return 0, nil
	}

	return parseVersionFromKey(k)
}

func parseVersionFromKey(key []byte) (int, error) {
	str := string(key)
	lastColon := len(str) - (versionDigits + 1)
	if lastColon < 0 || str[lastColon] != ':' {
		return 0, errorfamily.NewCorruption("bbolt.invalid_key_format",
			"invalid key format: "+str)
	}

	n, err := strconv.Atoi(str[lastColon+1:])
	if err != nil {
		return 0, errorfamily.WrapCorruption(err, "bbolt.parse_version",
			"parse version from key")
	}

	return n, nil
}

func validateEventOwnership(evt event.Event, ref id.StreamRef) error {
	if evt.StreamType() != ref.Type {
		return errorfamily.WrapConflict(ErrStreamTypeMismatch, "bbolt.stream_type_mismatch",
			fmt.Sprintf("expected %s, got %s", ref.Type, evt.StreamType()))
	}

	if evt.StreamID() != ref.ID {
		return errorfamily.WrapConflict(ErrStreamIDMismatch, "bbolt.stream_id_mismatch",
			fmt.Sprintf("expected %s, got %s", ref.ID, evt.StreamID()))
	}

	return nil
}

// Ensure EventStore implements event.Store, Journal, and SeekableJournal.
var (
	_ event.Store           = (*EventStore)(nil)
	_ event.Journal         = (*EventStore)(nil)
	_ event.SeekableJournal = (*EventStore)(nil)
)

const versionDigits = 10
