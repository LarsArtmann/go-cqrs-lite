package bboltengine

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	bolt "go.etcd.io/bbolt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// seedSeqCounters scans existing keys and seeds all in-memory sequence counters
// to the max existing value per group. This ensures that reopening a persistent
// bbolt DB does not restart counters from zero and overwrite existing keys.
//
// Called once during construction when the engine wraps a persistent DB.
// The scan is O(N) in existing key count — acceptable as a one-time startup cost.
func (e *bboltEngine) seedSeqCounters() error {
	if e.persistence == metaengine.PersistenceVolatile {
		return nil
	}

	return e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return nil
		}

		if err := e.seedCollectionSeqs(bucket, "sl", &e.streamSeq); err != nil {
			return fmt.Errorf("seed stream seqs: %w", err)
		}

		if err := e.seedCollectionSeqs(bucket, "jl", &e.journalSeq); err != nil {
			return fmt.Errorf("seed journal seqs: %w", err)
		}

		if err := e.seedCollectionSeqs(bucket, "l", &e.logSeq); err != nil {
			return fmt.Errorf("seed log seqs: %w", err)
		}

		if err := e.seedMultimapSeqs(bucket); err != nil {
			return fmt.Errorf("seed multimap seqs: %w", err)
		}

		return nil
	})
}

// seedCollectionSeqs scans a tag prefix (e.g. "jl", "l", "sl") and seeds a
// per-collection (or per-stream) counter sync.Map.
// art-dupl:accept cross-module KV engine pattern — separate go.mod
func (e *bboltEngine) seedCollectionSeqs(
	bucket *bolt.Bucket,
	tag string,
	target *sync.Map,
) error { //nolint:unparam // consistent KV engine pattern
	prefix := []byte(tag + sep)
	tagLen := len(tag + sep)

	c := bucket.Cursor()

	for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
		group, seq, ok := extractGroupAndSeq(k, tagLen)
		if !ok {
			continue
		}

		seedSyncMapMax(target, group, seq)
	}

	return nil
}

// seedMultimapSeqs scans the mm prefix and seeds mmSeq counters.
// art-dupl:accept cross-module KV engine pattern — separate go.mod
func (e *bboltEngine) seedMultimapSeqs(
	bucket *bolt.Bucket,
) error { //nolint:unparam // consistent KV engine pattern
	tag := "mm"
	prefix := []byte(tag + sep)
	tagLen := len(tag + sep)

	c := bucket.Cursor()

	for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
		if len(k) < tagLen+22 {
			continue
		}

		rest := k[tagLen:]
		before, _, ok := bytes.Cut(rest, []byte{0})
		if !ok {
			continue
		}

		col := string(before)

		if k[len(k)-21] != 0 {
			continue
		}

		seq, err := strconv.ParseInt(string(k[len(k)-20:]), 10, 64)
		if err != nil {
			continue
		}

		seedSyncMapMax(&e.mmSeq, col, seq)
	}

	return nil
}

// extractGroupAndSeq parses a bbolt key into its group identifier and seq.
// art-dupl:accept cross-module KV engine pattern — separate go.mod
func extractGroupAndSeq(key []byte, prefixLen int) (string, int64, bool) {
	if len(key) < prefixLen+22 {
		return "", 0, false
	}

	if key[len(key)-21] != 0 {
		return "", 0, false
	}

	seq, err := strconv.ParseInt(string(key[len(key)-20:]), 10, 64)
	if err != nil {
		return "", 0, false
	}

	return string(key[prefixLen : len(key)-21]), seq, true
}

// seedSyncMapMax seeds a sync.Map (storing *atomic.Int64) to at least seq.
// art-dupl:accept cross-module KV engine pattern — separate go.mod
func seedSyncMapMax(m *sync.Map, key string, seq int64) {
	actual, _ := m.LoadOrStore(key, new(atomic.Int64))
	counter := actual.(*atomic.Int64)

	for {
		existing := counter.Load()
		if existing >= seq {
			return
		}

		if counter.CompareAndSwap(existing, seq) {
			return
		}
	}
}
