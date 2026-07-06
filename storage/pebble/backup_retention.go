package pebble

import (
	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"
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
		return errorfamily.WrapInfrastructure(err, "pebble.checkpoint",
			"checkpoint to "+dir)
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

// Flush forces a flush of the memtable to disk, triggering a level-0
// compaction. Call after batch writes or before a checkpoint to ensure
// all data is persisted to SST files.
func (b *Backend) Flush() error {
	err := b.database.Flush()
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.flush", "flush database")
	}

	return nil
}
