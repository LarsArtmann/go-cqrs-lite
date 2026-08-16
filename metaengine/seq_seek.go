package metaengine

import "context"

// StreamLogEntry pairs a journal value with its engine resume token.
//
// Seq is the engine's append sequence for this entry: an opaque,
// strictly-increasing, never-reused token, stable within a collection. It is
// NOT a position or a count — on engines with a shared autoincrement counter
// (SQLite, Postgres, MySQL, DuckDB) the seqs of one collection interleave with
// other collections' appends and contain gaps. Values start at 1.
type StreamLogEntry struct {
	// Seq is the engine resume token. Feed it back to
	// SeqSeekableStreamLog.JournalReadFromSeq to resume after this entry.
	// 0 means "from the beginning" when passed as a cursor.
	Seq int64
	// Value is the decoded journal payload (same content JournalReadAll
	// returns).
	Value any
}

// SeqSeekableStreamLog is the token-resumable journal capability. Engines
// implement it when they can resume a journal read via an index seek on the
// entry's own sequence, instead of skipping ahead by position.
//
// Unlike [StreamLogBackend.JournalReadFrom] — whose afterSeq parameter is a
// POSITION within the collection and therefore forces OFFSET-style skips on
// engines with shared seq counters — the cursor here is a token previously
// returned by this interface. A token resume is a pure
// `WHERE collection = ? AND seq > ?` index range seek: O(log n) per page
// instead of O(offset), and gap-tolerant by construction (deleted or
// rolled-back journal entries cannot shift the cursor).
//
// Tokens are engine-instance-local: never persist a Seq across engine
// migrations (sqlite → pg) or restarts of an engine whose seq counter resets.
// Long-lived checkpoints must store the domain cursor (e.g. event ID), not
// the raw Seq.
//
// Optional-capability pattern (same as [Transactional], [AtomicAppender]):
// engines adopt incrementally; callers type-assert.
type SeqSeekableStreamLog interface {
	// JournalReadAllWithSeq is JournalReadAll with each entry's resume
	// token attached. The entries are in the same order JournalReadAll
	// returns, and their Seq values are strictly increasing.
	JournalReadAllWithSeq(ctx context.Context, collection string) ([]StreamLogEntry, error)

	// JournalReadFromSeq returns up to limit entries with Seq > afterSeq,
	// in journal order. afterSeq 0 reads from the start; a token obtained
	// from a prior read resumes exactly after that entry. limit <= 0 means
	// no limit.
	JournalReadFromSeq(
		ctx context.Context,
		collection string,
		afterSeq int64,
		limit int,
	) ([]StreamLogEntry, error)
}
