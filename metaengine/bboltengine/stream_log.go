package bboltengine

import (
	"bytes"
	"context"
	"fmt"
	"iter"
	"slices"
	"sync/atomic"

	bolt "go.etcd.io/bbolt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- StreamLogBackend implementation ---
//
// Key scheme (shared with pebbleengine/badgerengine via keycodec):
//
//	sl\x00{collection}\x00{streamID}\x00{seq:020d}  — per-stream entries
//	jl\x00{collection}\x00{gseq:020d}               — global journal index
//
// The per-stream entries store the actual values (JSON-encoded).
// The global journal stores (streamID + "\x00" + value) so JournalReadAll
// can reconstruct cross-stream ordering.

func (e *bboltEngine) nextStreamSeq(col, sid string) int64 {
	k := streamSeqMapKey(col, sid)
	actual, _ := e.streamSeq.LoadOrStore(k, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1)
}

func (e *bboltEngine) nextJournalSeq(col string) int64 {
	actual, _ := e.journalSeq.LoadOrStore(col, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1)
}

func (e *bboltEngine) StreamAppend(
	_ context.Context,
	col, sid string,
	values []any,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		for _, v := range values {
			seq := e.nextStreamSeq(col, sid)
			gseq := e.nextJournalSeq(col)

			valBytes := encodeJSON(v)
			journalEntry := fmt.Sprintf("%s%s%s", sid, sep, string(valBytes))

			if err := bucket.Put(streamKey(col, sid, seq), valBytes); err != nil {
				return err //nolint:wrapcheck // bbolt error is self-describing
			}

			if err := bucket.Put(journalKey(col, gseq), []byte(journalEntry)); err != nil {
				return err //nolint:wrapcheck // bbolt error is self-describing
			}
		}

		return nil
	})
}

// StreamAppendExpected implements AtomicAppender for optimistic concurrency.
// bbolt's Update transaction provides exclusive access, so the count-then-append
// is atomic without additional locking beyond e.mu (which serializes seq
// generation across concurrent calls).
func (e *bboltEngine) StreamAppendExpected(
	_ context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		current, err := countStreamEntriesBBolt(bucket, col, sid)
		if err != nil {
			return err
		}

		if current != expectedVersion {
			return metaengine.ErrVersionConflict
		}

		for _, v := range values {
			seq := e.nextStreamSeq(col, sid)
			gseq := e.nextJournalSeq(col)

			valBytes := encodeJSON(v)
			journalEntry := fmt.Sprintf("%s%s%s", sid, sep, string(valBytes))

			if err := bucket.Put(streamKey(col, sid, seq), valBytes); err != nil {
				return err //nolint:wrapcheck // bbolt error is self-describing
			}

			if err := bucket.Put(journalKey(col, gseq), []byte(journalEntry)); err != nil {
				return err //nolint:wrapcheck // bbolt error is self-describing
			}
		}

		return nil
	})
}

func (e *bboltEngine) StreamRead(
	_ context.Context,
	col, sid string,
) ([]any, error) {
	prefix := streamPrefix(col, sid)

	var result []any

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			result = append(result, decodeJSON(slices.Clone(v)))
		}

		return nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

func (e *bboltEngine) StreamVersion(
	_ context.Context,
	col, sid string,
) (int64, error) {
	prefix := streamPrefix(col, sid)

	var count int64

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			count++
		}

		return nil
	})

	return count, err //nolint:wrapcheck // passthrough
}

func (e *bboltEngine) JournalReadAll(
	_ context.Context,
	col string,
) ([]any, error) {
	prefix := journalPrefix(col)

	var result []any

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			result = append(result, extractJournalValue(cloneBytes(v)))
		}

		return nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

func (e *bboltEngine) JournalReadFrom(
	_ context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	prefix := journalPrefix(col)

	startKey := journalKey(col, afterSeq+1)

	var result []any

	count := 0

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, v := c.Seek(startKey); k != nil; k, v = c.Next() {
			if !bytes.HasPrefix(k, prefix) {
				break
			}

			if limit > 0 && count >= limit {
				break
			}

			result = append(result, extractJournalValue(cloneBytes(v)))
			count++
		}

		return nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

// --- StreamingScan ---

func (e *bboltEngine) StreamScan(
	_ context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		prefix := collectionPrefix(col)

		err := e.db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketName))
			c := bucket.Cursor()

			if sort == nil {
				for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
					decoded := decodeJSON(slices.Clone(v))

					if len(filters) > 0 && !metaengine.PassesFilterSpecs(decoded, filters) {
						continue
					}

					if !yield(decoded, nil) {
						return nil
					}
				}

				return nil
			}

			// Sorted path: collect, sort, then stream.
			var matched []any

			for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
				decoded := decodeJSON(slices.Clone(v))

				if len(filters) > 0 && !metaengine.PassesFilterSpecs(decoded, filters) {
					continue
				}

				matched = append(matched, decoded)
			}

			sortMatchedFuncKV(matched, sort)

			for _, val := range matched {
				if !yield(val, nil) {
					return nil
				}
			}

			return nil
		})
		if err != nil {
			yield(nil, err)
		}
	}
}

// sortMatchedFuncKV sorts a slice of decoded items by a named column.
// art-dupl:accept cross-module KV engine pattern — separate go.mod
func sortMatchedFuncKV(matched []any, sort *metaengine.SortSpec) {
	slices.SortFunc(matched, func(a, b any) int {
		av := metaengine.ItemFieldByName(a, sort.Column)
		bv := metaengine.ItemFieldByName(b, sort.Column)

		cmp := metaengine.CompareValues(av, bv)
		if sort.Desc {
			return -cmp
		}

		return cmp
	})
}

// countStreamEntriesBBolt counts entries matching a stream prefix.
// art-dupl:accept cross-module KV engine pattern — separate go.mod
func countStreamEntriesBBolt(
	bucket *bolt.Bucket,
	col, sid string,
) (int64, error) { //nolint:unparam // consistent KV engine pattern
	prefix := streamPrefix(col, sid)

	var count int64

	c := bucket.Cursor()

	for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
		count++
	}

	return count, nil
}

// extractJournalValue parses a journal entry "streamID\x00value" and returns
// the decoded value part.
// art-dupl:accept cross-module KV engine pattern — separate go.mod
func extractJournalValue(raw []byte) any {
	_, after, ok := bytes.Cut(raw, []byte(sep))
	if ok {
		return decodeJSON(after)
	}

	return decodeJSON(raw)
}
