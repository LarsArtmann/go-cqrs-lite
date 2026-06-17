package pebble

import (
	"fmt"
	"time"

	"github.com/cockroachdb/pebble"
)

// Checkpoint creates a point-in-time snapshot of the entire database at the
// given directory. Useful for backups, online compaction, and disaster recovery.
// The checkpoint directory contains a complete, consistent copy of the DB at
// the moment Checkpoint is called — writes continue normally during checkpointing.
//
//	dir := fmt.Sprintf("backups/%s", time.Now().Format("2006-01-02-150405"))
//	err := backend.Checkpoint(dir)
//	defer os.RemoveAll(dir) // or upload to S3/GCS
func (b *Backend) Checkpoint(dir string) error {
	err := b.database.Checkpoint(dir)
	if err != nil {
		return fmt.Errorf("pebble: checkpoint to %s: %w", dir, err)
	}

	return nil
}

// NewSnapshot returns a point-in-time consistent read view of the database.
// Iterators created from the snapshot see a consistent state as of the call —
// writes after NewSnapshot are invisible. Always call Close() when done to
// release the resources held by the snapshot (prevents compaction of old files).
//
//	snap := backend.NewSnapshot()
//	defer snap.Close()
//	iter := snap.NewIter(&pebble.IterOptions{
//	    LowerBound: []byte("cqrs_journal:"),
//	    UpperBound: []byte("cqrs_journal:\xff"),
//	})
//	// iterate consistent view of all events...
func (b *Backend) NewSnapshot() *pebble.Snapshot {
	return b.database.NewSnapshot()
}

// DeleteEventsBefore deletes journal entries older than the given timestamp.
// This is a bulk retention operation — all journal keys with a timestamp
// component before cutoff are removed in a single range tombstone. The
// per-aggregate event log (cqrs_event:) is NOT affected — only the global
// journal index (cqrs_journal:) is pruned.
//
// Use this for time-based retention policies:
//
//	// Delete journal entries older than 90 days
//	cutoff := time.Now().AddDate(0, 0, -90)
//	err := backend.DeleteEventsBefore(cutoff)
//
// Note: This does not reclaim disk space immediately. Space is reclaimed
// when Pebble compacts the range tombstone. Use Flush() afterward to trigger
// a flush, or wait for background compaction.
func (b *Backend) DeleteEventsBefore(cutoff time.Time) error {
	nanos := cutoff.UnixNano()

	// Journal keys are formatted as cqrs_journal:{020d_unix_nano}:{eventID}
	// We construct a prefix that sorts before any key with a timestamp >= cutoff.
	lowerBound := []byte("cqrs_journal:")
	upperBound := fmt.Appendf([]byte("cqrs_journal:"), "%020d", nanos)

	err := b.database.DeleteRange(lowerBound, upperBound, nil)
	if err != nil {
		return fmt.Errorf("pebble: delete journal before %s: %w", cutoff.Format(time.RFC3339), err)
	}

	return nil
}

// Flush forces a flush of the memtable to disk, triggering a level-0
// compaction. Useful after DeleteEventsBefore to ensure range tombstones
// are persisted and eligible for compaction.
func (b *Backend) Flush() error {
	err := b.database.Flush()
	if err != nil {
		return fmt.Errorf("pebble: flush: %w", err)
	}

	return nil
}
