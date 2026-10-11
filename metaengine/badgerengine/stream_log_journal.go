package badgerengine

import (
	"bytes"
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func (e *badgerEngine) JournalReadAll(
	_ context.Context,
	col string,
) ([]any, error) {
	return entryValues(e.journalEntries(col, 0, 0))
}

func (e *badgerEngine) JournalReadFrom(
	_ context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	return entryValues(e.journalEntries(col, afterSeq, limit))
}

// JournalReadAllWithSeq returns every journal entry with its resume token
// (the per-collection journal seq embedded in the journal key). Implements
// metaengine.SeqSeekableStreamLog.
func (e *badgerEngine) JournalReadAllWithSeq(
	_ context.Context,
	col string,
) ([]metaengine.StreamLogEntry, error) {
	return e.journalEntries(col, 0, 0)
}

// JournalReadFromSeq returns up to limit entries with Seq > afterSeq by
// seeking journalKey(col, afterSeq+1) — the same O(log n) Seek
// JournalReadFrom performs. Implements metaengine.SeqSeekableStreamLog. The
// token is read back out of the journal key, so callers resume on true
// engine seqs.
func (e *badgerEngine) JournalReadFromSeq(
	_ context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]metaengine.StreamLogEntry, error) {
	return e.journalEntries(col, afterSeq, limit)
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
