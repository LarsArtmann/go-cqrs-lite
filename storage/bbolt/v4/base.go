package bbolt

import (
	"log/slog"

	bolt "go.etcd.io/bbolt"
)

// Bucket names for the CQRS stores sharing a single *bbolt.DB.
const (
	bucketEvents      = "cqrs_events"
	bucketJournal     = "cqrs_journal"
	bucketSnapshots   = "cqrs_snapshots"
	bucketCheckpoints = "cqrs_checkpoints"
	bucketKV          = "cqrs_kv"
	bucketCommands    = "cqrs_commands"
	bucketCmdJournal  = "cqrs_cmd_journal"
	bucketQueries     = "cqrs_queries"
)

// storeBase holds the bbolt DB handle and logger shared by every store in
// this package. Each store embeds storeBase.
type storeBase struct {
	db     *bolt.DB
	logger *slog.Logger
	batch  bool
}

// writeTx runs fn in a write transaction: db.Batch (group commit across
// concurrent writers, one fsync per group) when batch mode is enabled via
// [WithBatchCommit], db.Update (the default) otherwise. fn must be
// idempotent: bbolt re-invokes it when a combined batch-mate fails and the
// transaction is retried — all closures in this package only mutate the
// transaction, never external state.
func (s *storeBase) writeTx(fn func(*bolt.Tx) error) error {
	if s.batch {
		return s.db.Batch(fn)
	}

	return s.db.Update(fn)
}

// enableBatchCommit switches this store's write transactions to group commit.
// Unexported: enabled via [WithBatchCommit] on the Backend constructors.
func (s *storeBase) enableBatchCommit() {
	s.batch = true
}

// createBuckets creates all CQRS buckets inside a write transaction.
// Called once during Backend construction.
func createBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		for _, name := range []string{
			bucketEvents, bucketJournal, bucketSnapshots,
			bucketCheckpoints, bucketKV, bucketCommands,
			bucketCmdJournal, bucketQueries,
		} {
			if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
				return err
			}
		}

		return nil
	})
}
