package pebbleengine

import (
	"fmt"
	"sync"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// seedSeqCounters scans existing keys and seeds all in-memory sequence counters
// to the max existing value per group. This ensures that reopening a persistent
// Pebble DB does not restart counters from zero and overwrite existing keys.
//
// Called once during construction when the engine wraps a persistent DB.
// The scan is O(N) in existing key count — acceptable as a one-time startup cost.
func (e *pebbleEngine) seedSeqCounters() error {
	if err := e.seedCollectionSeqs("sl", &e.streamSeq, true); err != nil {
		return fmt.Errorf("seed stream seqs: %w", err)
	}

	if err := e.seedCollectionSeqs("jl", &e.journalSeq, false); err != nil {
		return fmt.Errorf("seed journal seqs: %w", err)
	}

	if err := e.seedCollectionSeqs("l", &e.logSeq, false); err != nil {
		return fmt.Errorf("seed log seqs: %w", err)
	}

	if err := e.seedCollectionSeqs("mm", &e.mmSeq, false); err != nil {
		return fmt.Errorf("seed multimap seqs: %w", err)
	}

	return nil
}

// seedCollectionSeqs scans a tag prefix (e.g. "jl", "l", "mm") and seeds a
// per-group counter sync.Map via keycodec.SeedSeqMax. With wholeGroup the
// group is everything between tag and seq ("sl" uses "col\x00sid"); otherwise
// it is only the first segment (collection name).
func (e *pebbleEngine) seedCollectionSeqs(tag string, target *sync.Map, wholeGroup bool) error {
	iter, err := e.newPrefixIter([]byte(tag + sep))
	if err != nil {
		return err
	}

	defer metaengine.DeferClose(iter)

	tagLen := len(tag + sep)

	for iter.First(); iter.Valid(); iter.Next() {
		group, seq, ok := keycodec.SplitGroupAndSeq(iter.Key(), tagLen, wholeGroup)
		if !ok {
			continue
		}

		keycodec.SeedSeqMax(target, group, seq)
	}

	return iter.Error()
}
