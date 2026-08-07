package bbolt

import (
	"bytes"
	"context"
	"io"

	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// bboltEventIterator lazily yields events from a long-lived bbolt read
// transaction. Close rolls back the transaction, releasing the read lock.
type bboltEventIterator struct {
	tx        *bolt.Tx
	cursor    *bolt.Cursor
	prefix    []byte
	upper     []byte
	k, v      []byte
	skipUntil string
	skipping  bool
	limit     int
	yielded   int
	started   bool
	finished  bool
	firstErr  error
}

func (it *bboltEventIterator) Next() (event.Event, error) {
	if it.finished {
		return nil, io.EOF
	}

	if it.firstErr != nil {
		it.finished = true
		return nil, it.firstErr
	}

	if it.limit > 0 && it.yielded >= it.limit {
		it.finished = true
		return nil, io.EOF
	}

	for {
		if !it.started {
			it.started = true
			// On first call, k/v were captured during Seek/First in the
			// constructor.
		} else {
			it.k, it.v = it.cursor.Next()
		}

		if it.k == nil {
			it.finished = true
			return nil, io.EOF
		}

		if len(it.prefix) > 0 && !bytes.HasPrefix(it.k, it.prefix) {
			it.finished = true
			return nil, io.EOF
		}

		if len(it.upper) > 0 && string(it.k) >= string(it.upper) {
			it.finished = true
			return nil, io.EOF
		}

		if it.skipping {
			if journalKeyEventID(it.k) == it.skipUntil {
				it.skipping = false
			}

			continue
		}

		evt, err := deserializeEvent(it.v)
		if err != nil {
			it.finished = true
			it.firstErr = err
			return nil, err
		}

		it.yielded++
		return evt, nil
	}
}

func (it *bboltEventIterator) Close() error {
	it.finished = true
	if it.tx != nil {
		return it.tx.Rollback()
	}

	return nil
}

// newJournalIterator opens a read-only transaction against the journal bucket
// (global ordering by OccurredAt).
func (s *EventStore) newJournalIterator(
	skipID string,
	limit int,
) (event.EventIterator, error) {
	tx, err := s.db.Begin(false)
	if err != nil {
		return nil, wrapBucketErr(err, "bbolt.stream_iterator", "begin read transaction")
	}

	bucket := tx.Bucket([]byte(bucketJournal))
	if bucket == nil {
		_ = tx.Rollback()
		return &bboltEventIterator{finished: true}, nil
	}

	cursor := bucket.Cursor()
	k, v := cursor.First()

	iter := &bboltEventIterator{
		tx:        tx,
		cursor:    cursor,
		k:         k,
		v:         v,
		skipUntil: skipID,
		skipping:  skipID != "",
		limit:     limit,
	}

	if k == nil {
		iter.finished = true
	}

	return iter, nil
}

// newStreamEventIterator opens a read-only transaction against the events
// bucket (per-stream ordering by version).
func (s *EventStore) newStreamEventIterator(
	prefix, lowerBound, upper []byte,
) (event.EventIterator, error) {
	tx, err := s.db.Begin(false)
	if err != nil {
		return nil, wrapBucketErr(err, "bbolt.stream_iterator", "begin read transaction")
	}

	bucket := tx.Bucket([]byte(bucketEvents))
	if bucket == nil {
		_ = tx.Rollback()
		return &bboltEventIterator{finished: true}, nil
	}

	cursor := bucket.Cursor()

	var k, v []byte
	if len(lowerBound) > 0 {
		k, v = cursor.Seek(lowerBound)
	} else {
		k, v = cursor.Seek(prefix)
	}

	iter := &bboltEventIterator{
		tx:     tx,
		cursor: cursor,
		prefix: prefix,
		upper:  upper,
		k:      k,
		v:      v,
	}

	if k == nil || (len(prefix) > 0 && !bytes.HasPrefix(k, prefix)) {
		iter.finished = true
	}

	return iter, nil
}

// LoadStream is the streaming equivalent of Load — all events for one stream.
func (s *EventStore) LoadStream(
	_ context.Context,
	ref id.StreamRef,
) (event.EventIterator, error) {
	prefix := streamPrefix(ref)
	upper := streamUpperBound(ref)
	return s.newStreamEventIterator(prefix, nil, upper)
}

// LoadStreamFromVersion is the streaming equivalent of LoadFromVersion.
func (s *EventStore) LoadStreamFromVersion(
	_ context.Context,
	ref id.StreamRef,
	version event.Version,
) (event.EventIterator, error) {
	lower := eventKey(ref, version+1)
	prefix := streamPrefix(ref)
	upper := streamUpperBound(ref)
	return s.newStreamEventIterator(prefix, lower, upper)
}

// ReadStream is the streaming equivalent of ReadAll — all events globally.
func (s *EventStore) ReadStream(_ context.Context) (event.EventIterator, error) {
	return s.newJournalIterator("", 0)
}

// ReadStreamFrom is the streaming equivalent of ReadFrom — events after a
// given event ID, optionally limited.
func (s *EventStore) ReadStreamFrom(
	_ context.Context,
	afterEventID id.EventID,
	limit int,
) (event.EventIterator, error) {
	skipID := ""
	if !afterEventID.IsZero() {
		skipID = afterEventID.String()
	}

	return s.newJournalIterator(skipID, limit)
}

var (
	_ event.StreamingSource  = (*EventStore)(nil)
	_ event.StreamingJournal = (*EventStore)(nil)
)
