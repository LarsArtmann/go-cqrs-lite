package pebble

import (
	"log/slog"

	"github.com/cockroachdb/pebble"
)

// storeBase holds the Pebble DB handle and configuration shared by every store
// in this package (EventStore, SnapshotStore, CheckpointStore). Each store
// embeds storeBase, which promotes the db, logger, prefix and syncWrites
// fields along with the writeOptions method — without coupling the stores to
// each other. They remain independently constructible and usable.
type storeBase struct {
	db         *pebble.DB
	logger     *slog.Logger
	prefix     string
	syncWrites bool
}

// writeOptions returns pebble.Sync when sync writes are enabled, otherwise nil
// (Pebble's default asynchronous write semantics).
func (b *storeBase) writeOptions() *pebble.WriteOptions {
	if b.syncWrites {
		return pebble.Sync
	}

	return nil
}
