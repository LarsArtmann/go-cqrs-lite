package pebbleengine

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
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
	defer metaengine.DeferClose(batch)

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

	defer metaengine.DeferClose(iter)

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
	n, err := e.countStreamEntries(col, sid)
	if err != nil {
		return 0, fmt.Errorf("pebbleengine.StreamVersion: %w", err)
	}

	return n, nil
}

func (e *pebbleEngine) JournalReadAll(_ context.Context, col string) ([]any, error) {
	iter, err := e.newPrefixIter(journalPrefix(col))
	if err != nil {
		return nil, fmt.Errorf("pebbleengine.JournalReadAll: %w", err)
	}

	defer metaengine.DeferClose(iter)

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

	defer metaengine.DeferClose(iter)

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

// JournalReadAllWithSeq returns every journal entry with its resume token
// (the per-collection journal seq embedded in the journal key). Implements
// metaengine.SeqSeekableStreamLog.
func (e *pebbleEngine) JournalReadAllWithSeq(
	_ context.Context,
	col string,
) ([]metaengine.StreamLogEntry, error) {
	iter, err := e.newPrefixIter(journalPrefix(col))
	if err != nil {
		return nil, fmt.Errorf("pebbleengine.JournalReadAllWithSeq: %w", err)
	}

	defer metaengine.DeferClose(iter)

	var result []metaengine.StreamLogEntry

	for iter.First(); iter.Valid(); iter.Next() {
		if entry, ok := journalEntryFromKey(iter.Key(), iter.Value()); ok {
			result = append(result, entry)
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("pebbleengine.JournalReadAllWithSeq iter: %w", err)
	}

	if result == nil {
		result = []metaengine.StreamLogEntry{}
	}

	return result, nil
}

// JournalReadFromSeq returns up to limit entries with Seq > afterSeq by
// seeking journalKey(col, afterSeq+1) — the same O(log n) seek JournalReadFrom
// performs. Implements metaengine.SeqSeekableStreamLog. The token is read back
// out of the journal key, so callers resume on true engine seqs.
func (e *pebbleEngine) JournalReadFromSeq(
	_ context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]metaengine.StreamLogEntry, error) {
	upperBound := nextKey(journalPrefix(col))

	// LowerBound is exclusive of afterSeq: start at afterSeq+1.
	startKey := journalKey(col, afterSeq+1)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: startKey,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, fmt.Errorf("pebbleengine.JournalReadFromSeq: %w", err)
	}

	defer metaengine.DeferClose(iter)

	var result []metaengine.StreamLogEntry
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		if limit > 0 && count >= limit {
			break
		}

		if entry, ok := journalEntryFromKey(iter.Key(), iter.Value()); ok {
			result = append(result, entry)
			count++
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("pebbleengine.JournalReadFromSeq iter: %w", err)
	}

	if result == nil {
		result = []metaengine.StreamLogEntry{}
	}

	return result, nil
}

// journalEntryFromKey builds a StreamLogEntry from a journal key and its raw
// "streamID\x00value" payload. The seq token is the zero-padded decimal in
// the key's tail. Returns false when the key carries no parseable seq.
func journalEntryFromKey(key, raw []byte) (metaengine.StreamLogEntry, bool) {
	seq, ok := keycodec.JournalSeq(key)
	if !ok {
		return metaengine.StreamLogEntry{}, false
	}

	idx := bytes.Index(raw, []byte(sep))
	if idx >= 0 {
		return metaengine.StreamLogEntry{Seq: seq, Value: decodeJSON(raw[idx+1:])}, true
	}

	return metaengine.StreamLogEntry{Seq: seq, Value: decodeJSON(raw)}, true
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
	defer metaengine.DeferClose(batch)

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

	defer metaengine.DeferClose(iter)

	var count int64
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return count, iter.Error()
}

// Compile-time assertions.
var (
	_ metaengine.StreamLogBackend     = (*pebbleEngine)(nil)
	_ metaengine.SeqSeekableStreamLog = (*pebbleEngine)(nil)
	_ metaengine.AtomicAppender       = (*pebbleEngine)(nil)
)
