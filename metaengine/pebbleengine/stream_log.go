package pebbleengine

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"

	"github.com/cockroachdb/pebble"

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

// streamSeqKey builds the sync.Map key for per-stream sequence counters.
func streamSeqKey(col, sid string) string {
	return col + sep + sid
}

// nextStreamSeq returns the next sequence number for a stream.
func (e *pebbleEngine) nextStreamSeq(col, sid string) int64 {
	k := streamSeqKey(col, sid)
	actual, _ := e.streamSeq.LoadOrStore(k, &atomic.Int64{})
	return actual.(*atomic.Int64).Add(1)
}

// nextJournalSeq returns the next global journal sequence for a collection.
func (e *pebbleEngine) nextJournalSeq(col string) int64 {
	actual, _ := e.journalSeq.LoadOrStore(col, &atomic.Int64{})
	return actual.(*atomic.Int64).Add(1)
}

func (e *pebbleEngine) StreamAppend(_ context.Context, col, sid string, values []any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	batch := e.db.NewBatch()
	defer func() { _ = batch.Close() }()

	for _, v := range values {
		seq := e.nextStreamSeq(col, sid)
		gseq := e.nextJournalSeq(col)

		valBytes := encodeJSON(v)
		journalEntry := fmt.Sprintf("%s%s%s", sid, sep, string(valBytes))

		_ = batch.Set(streamKey(col, sid, seq), valBytes, nil)
		_ = batch.Set(journalKey(col, gseq), []byte(journalEntry), nil)
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("pebbleengine.StreamAppend: %w", err)
	}

	return nil
}

func (e *pebbleEngine) StreamRead(_ context.Context, col, sid string) ([]any, error) {
	iter, err := e.newPrefixIter(streamPrefix(col, sid))
	if err != nil {
		return nil, fmt.Errorf("pebbleengine.StreamRead: %w", err)
	}

	defer func() { _ = iter.Close() }()

	var result []any

	for iter.First(); iter.Valid(); iter.Next() {
		result = append(result, decodeJSON(iter.Value()))
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("pebbleengine.StreamRead iter: %w", err)
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

func (e *pebbleEngine) StreamVersion(_ context.Context, col, sid string) (int64, error) {
	iter, err := e.newPrefixIter(streamPrefix(col, sid))
	if err != nil {
		return 0, fmt.Errorf("pebbleengine.StreamVersion: %w", err)
	}

	defer func() { _ = iter.Close() }()

	var count int64
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, fmt.Errorf("pebbleengine.StreamVersion iter: %w", err)
	}

	return count, nil
}

func (e *pebbleEngine) JournalReadAll(_ context.Context, col string) ([]any, error) {
	iter, err := e.newPrefixIter(journalPrefix(col))
	if err != nil {
		return nil, fmt.Errorf("pebbleengine.JournalReadAll: %w", err)
	}

	defer func() { _ = iter.Close() }()

	var result []any

	for iter.First(); iter.Valid(); iter.Next() {
		// Journal entries are stored as "streamID\x00value".
		// Extract the value part.
		raw := iter.Value()
		idx := bytes.Index(raw, []byte(sep))
		if idx >= 0 {
			result = append(result, decodeJSON(raw[idx+1:]))
		} else {
			result = append(result, decodeJSON(raw))
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("pebbleengine.JournalReadAll iter: %w", err)
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

func (e *pebbleEngine) JournalReadFrom(
	_ context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	upperBound := nextKey(journalPrefix(col))

	// LowerBound is exclusive of afterSeq: start at afterSeq+1.
	startKey := journalKey(col, afterSeq+1)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: startKey,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, fmt.Errorf("pebbleengine.JournalReadFrom: %w", err)
	}

	defer func() { _ = iter.Close() }()

	var result []any
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		if limit > 0 && count >= limit {
			break
		}

		raw := iter.Value()
		idx := bytes.Index(raw, []byte(sep))
		if idx >= 0 {
			result = append(result, decodeJSON(raw[idx+1:]))
		} else {
			result = append(result, decodeJSON(raw))
		}

		count++
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("pebbleengine.JournalReadFrom iter: %w", err)
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

// StreamAppendExpected implements AtomicAppender for optimistic concurrency.
func (e *pebbleEngine) StreamAppendExpected(
	_ context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Count current version by scanning stream prefix.
	current, err := e.countStreamEntries(col, sid)
	if err != nil {
		return err
	}

	if current != expectedVersion {
		return metaengine.ErrVersionConflict
	}

	batch := e.db.NewBatch()
	defer func() { _ = batch.Close() }()

	for _, v := range values {
		seq := e.nextStreamSeq(col, sid)
		gseq := e.nextJournalSeq(col)

		valBytes := encodeJSON(v)
		journalEntry := fmt.Sprintf("%s%s%s", sid, sep, string(valBytes))

		_ = batch.Set(streamKey(col, sid, seq), valBytes, nil)
		_ = batch.Set(journalKey(col, gseq), []byte(journalEntry), nil)
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("pebbleengine.StreamAppendExpected: %w", err)
	}

	return nil
}

func (e *pebbleEngine) countStreamEntries(col, sid string) (int64, error) {
	iter, err := e.newPrefixIter(streamPrefix(col, sid))
	if err != nil {
		return 0, err
	}

	defer func() { _ = iter.Close() }()

	var count int64
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return count, iter.Error()
}

// Compile-time assertions.
var (
	_ metaengine.StreamLogBackend = (*pebbleEngine)(nil)
	_ metaengine.AtomicAppender   = (*pebbleEngine)(nil)
)
