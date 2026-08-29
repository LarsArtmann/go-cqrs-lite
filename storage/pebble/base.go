package pebble

import (
	"log/slog"
	"slices"
	"sync"

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

	// shards serializes duplicate-check + commit per key shard, closing the
	// check-then-commit race in CommandStore/QueryStore Save paths. Locks
	// must be taken in ascending index order (see lockShards).
	shards [shardCount]sync.Mutex
}

// shardCount is the number of duplicate-check lock shards. Power of two.
const shardCount = 64

// shardIndex maps a key to its lock shard via FNV-1a. Own function so the
// uint32→int conversion stays isolated (gosec G115).
func shardIndex(key []byte) int {
	h := uint32(2166136261)
	for _, c := range key {
		h ^= uint32(c)
		h *= 16777619
	}

	return int(h & (shardCount - 1)) //nolint:gosec // masked to 6 bits
}

// lockShards locks the shards for the given keys in ascending index order
// (deadlock-free) and returns the locked indices for unlockShards.
func (b *storeBase) lockShards(keys ...[]byte) []int {
	idx := make([]int, 0, len(keys))
	for _, k := range keys {
		idx = append(idx, shardIndex(k))
	}

	slices.Sort(idx)
	idx = slices.Compact(idx)

	for _, i := range idx {
		b.shards[i].Lock()
	}

	return idx
}

// unlockShards releases locks taken by lockShards (any order).
func (b *storeBase) unlockShards(idx []int) {
	for _, i := range idx {
		b.shards[i].Unlock()
	}
}

// writeOptions returns pebble.Sync when sync writes are enabled, otherwise nil
// (Pebble's default asynchronous write semantics).
func (b *storeBase) writeOptions() *pebble.WriteOptions {
	if b.syncWrites {
		return pebble.Sync
	}

	return nil
}
