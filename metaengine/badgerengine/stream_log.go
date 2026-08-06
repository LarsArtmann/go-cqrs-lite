package badgerengine

import (
	"bytes"
	"context"
	"fmt"
	"iter"
	"slices"
	"sync/atomic"

	"github.com/dgraph-io/badger/v4"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- StreamLogBackend implementation ---
//
// Key scheme:
//   sl\x00{collection}\x00{streamID}\x00{seq:020d}  — per-stream entries
//   jl\x00{collection}\x00{gseq:020d}               — global journal index
//
// The per-stream entries store the actual values (JSON-encoded).
// The global journal stores (streamID + "\x00" + value) so JournalReadAll
// can reconstruct cross-stream ordering.

func streamKey(col, sid string, seq int64) []byte {
	return fmt.Appendf(nil, "sl%s%s%s%s%s%020d", sep, col, sep, sid, sep, seq)
}

func streamPrefix(col, sid string) []byte {
	return []byte("sl" + sep + col + sep + sid + sep)
}

func journalKey(col string, gseq int64) []byte {
	return fmt.Appendf(nil, "jl%s%s%s%020d", sep, col, sep, gseq)
}

func journalPrefix(col string) []byte {
	return []byte("jl" + sep + col + sep)
}

func streamSeqMapKey(col, sid string) string {
	return col + sep + sid
}

func (e *badgerEngine) nextStreamSeq(col, sid string) int64 {
	k := streamSeqMapKey(col, sid)
	actual, _ := e.streamSeq.LoadOrStore(k, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1)
}

func (e *badgerEngine) nextJournalSeq(col string) int64 {
	actual, _ := e.journalSeq.LoadOrStore(col, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1)
}

func (e *badgerEngine) StreamAppend(
	_ context.Context,
	col, sid string,
	values []any,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.db.Update(func(txn *badger.Txn) error {
		for _, v := range values {
			seq := e.nextStreamSeq(col, sid)
			gseq := e.nextJournalSeq(col)

			valBytes := encodeJSON(v)
			journalEntry := fmt.Sprintf("%s%s%s", sid, sep, string(valBytes))

			if err := txn.Set(streamKey(col, sid, seq), valBytes); err != nil {
				return err
			}

			if err := txn.Set(journalKey(col, gseq), []byte(journalEntry)); err != nil {
				return err
			}
		}

		return nil
	})
}

func (e *badgerEngine) StreamRead(
	_ context.Context,
	col, sid string,
) ([]any, error) {
	prefix := streamPrefix(col, sid)

	var result []any

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			val, err := iter.Item().ValueCopy(nil)
			if err != nil {
				return err
			}

			result = append(result, decodeJSON(val))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

func (e *badgerEngine) StreamVersion(
	_ context.Context,
	col, sid string,
) (int64, error) {
	prefix := streamPrefix(col, sid)

	var count int64

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			count++
		}

		return nil
	})

	return count, err
}

func (e *badgerEngine) JournalReadAll(
	_ context.Context,
	col string,
) ([]any, error) {
	prefix := journalPrefix(col)

	var result []any

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			val, err := iter.Item().ValueCopy(nil)
			if err != nil {
				return err
			}

			result = append(result, extractJournalValue(val))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

func (e *badgerEngine) JournalReadFrom(
	_ context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	prefix := journalPrefix(col)

	// Start from afterSeq+1 (exclusive lower bound).
	startKey := journalKey(col, afterSeq+1)

	var result []any

	count := 0

	err := e.db.View(func(txn *badger.Txn) error {
		opts := badger.IteratorOptions{}

		iter := txn.NewIterator(opts)
		defer iter.Close()

		for iter.Seek(startKey); iter.Valid(); iter.Next() {
			item := iter.Item()

			// Check prefix match.
			if !bytes.HasPrefix(item.Key(), prefix) {
				break
			}

			if limit > 0 && count >= limit {
				break
			}

			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			result = append(result, extractJournalValue(val))
			count++
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

// extractJournalValue parses a journal entry "streamID\x00value" and returns
// the decoded value part.
func extractJournalValue(raw []byte) any {
	_, after, ok := bytes.Cut(raw, []byte(sep))
	if ok {
		return decodeJSON(after)
	}

	return decodeJSON(raw)
}

// StreamAppendExpected implements AtomicAppender for optimistic concurrency.
func (e *badgerEngine) StreamAppendExpected(
	_ context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	current, err := e.countStreamEntries(col, sid)
	if err != nil {
		return err
	}

	if current != expectedVersion {
		return metaengine.ErrVersionConflict
	}

	return e.db.Update(func(txn *badger.Txn) error {
		for _, v := range values {
			seq := e.nextStreamSeq(col, sid)
			gseq := e.nextJournalSeq(col)

			valBytes := encodeJSON(v)
			journalEntry := fmt.Sprintf("%s%s%s", sid, sep, string(valBytes))

			if err := txn.Set(streamKey(col, sid, seq), valBytes); err != nil {
				return err
			}

			if err := txn.Set(journalKey(col, gseq), []byte(journalEntry)); err != nil {
				return err
			}
		}

		return nil
	})
}

func (e *badgerEngine) countStreamEntries(col, sid string) (int64, error) {
	prefix := streamPrefix(col, sid)

	var count int64

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			count++
		}

		return nil
	})

	return count, err
}

// --- StreamingScan ---

func (e *badgerEngine) StreamScan(
	_ context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		prefix := collectionPrefix(col)

		err := e.db.View(func(txn *badger.Txn) error {
			iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
			defer iter.Close()

			if sort == nil {
				for iter.Rewind(); iter.Valid(); iter.Next() {
					val, err := iter.Item().ValueCopy(nil)
					if err != nil {
						yield(nil, err)
						return nil
					}

					decoded := decodeJSON(val)

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

			for iter.Rewind(); iter.Valid(); iter.Next() {
				val, err := iter.Item().ValueCopy(nil)
				if err != nil {
					yield(nil, err)
					return nil
				}

				decoded := decodeJSON(val)

				if len(filters) > 0 && !metaengine.PassesFilterSpecs(decoded, filters) {
					continue
				}

				matched = append(matched, decoded)
			}

			sortMatchedFunc(matched, sort)

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

// sortMatchedFunc sorts a slice of decoded items by a named column.
func sortMatchedFunc(matched []any, sort *metaengine.SortSpec) {
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
