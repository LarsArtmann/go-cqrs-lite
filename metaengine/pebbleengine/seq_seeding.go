package pebbleengine

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
)

// seedSeqCounters scans existing keys and seeds all in-memory sequence counters
// to the max existing value per group. This ensures that reopening a persistent
// Pebble DB does not restart counters from zero and overwrite existing keys.
//
// Called once during construction when the engine wraps a persistent DB.
// The scan is O(N) in existing key count — acceptable as a one-time startup cost.
func (e *pebbleEngine) seedSeqCounters() error {
	if err := e.seedStreamSeqs(); err != nil {
		return fmt.Errorf("seed stream seqs: %w", err)
	}

	if err := e.seedCollectionSeqs("jl", &e.journalSeq); err != nil {
		return fmt.Errorf("seed journal seqs: %w", err)
	}

	if err := e.seedCollectionSeqs("l", &e.logSeq); err != nil {
		return fmt.Errorf("seed log seqs: %w", err)
	}

	if err := e.seedMultimapSeqs(); err != nil {
		return fmt.Errorf("seed multimap seqs: %w", err)
	}

	return nil
}

// seedStreamSeqs scans the sl\x00 prefix and seeds streamSeq counters.
// The sync.Map key for streamSeq is "col\x00sid" (matching streamSeqKey).
func (e *pebbleEngine) seedStreamSeqs() error {
	tag := "sl"
	iter, err := e.newPrefixIter([]byte(tag + sep))
	if err != nil {
		return err
	}

	defer func() { _ = iter.Close() }()

	tagLen := len(tag + sep)

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		group, seq, ok := extractGroupAndSeq(key, tagLen)
		if !ok {
			continue
		}

		seedSyncMapMax(&e.streamSeq, group, seq)
	}

	return iter.Error()
}

// seedCollectionSeqs scans a tag prefix (e.g. "jl", "l") and seeds a
// per-collection counter sync.Map. The group key is the collection name only.
func (e *pebbleEngine) seedCollectionSeqs(tag string, target *sync.Map) error {
	prefix := []byte(tag + sep)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: nextKey(prefix),
	})
	if err != nil {
		return err
	}

	defer func() { _ = iter.Close() }()

	tagLen := len(tag + sep)

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		group, seq, ok := extractGroupAndSeq(key, tagLen)
		if !ok {
			continue
		}

		seedSyncMapMax(target, group, seq)
	}

	return iter.Error()
}

// seedMultimapSeqs scans the mm\x00 prefix and seeds mmSeq counters.
// The mmSeq counter is keyed by collection name only (not collection+key).
func (e *pebbleEngine) seedMultimapSeqs() error {
	prefix := []byte("mm" + sep)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: nextKey(prefix),
	})
	if err != nil {
		return err
	}

	defer func() { _ = iter.Close() }()

	tagLen := len("mm" + sep)

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < tagLen+22 {
			continue
		}

		rest := key[tagLen:]
		nulIdx := indexOfByte(rest, 0)
		if nulIdx < 0 {
			continue
		}

		col := string(rest[:nulIdx])

		if key[len(key)-21] != 0 {
			continue
		}

		seq, err := strconv.ParseInt(string(key[len(key)-20:]), 10, 64)
		if err != nil {
			continue
		}

		seedSyncMapMax(&e.mmSeq, col, seq)
	}

	return iter.Error()
}

// extractGroupAndSeq parses a Pebble key into its group identifier and seq.
// prefixLen is the number of bytes to strip from the front (tag + sep).
// The seq is always the last 20 zero-padded digits, preceded by a \x00.
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

// indexOfByte returns the index of the first occurrence of b in data, or -1.
func indexOfByte(data []byte, b byte) int {
	for i, c := range data {
		if c == b {
			return i
		}
	}

	return -1
}

// seedSyncMapMax seeds a sync.Map (storing *atomic.Int64) to at least seq.
// Uses CAS loop to handle concurrent seeding (shouldn't happen, but safe).
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
