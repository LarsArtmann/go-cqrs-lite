package pebbleengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// resetTagPrefixes are the engine-owned key prefixes: the shared keycodec tag
// ranges (see the keycodec package key shapes) plus the two layout secondary
// index families ('i' filter, 'o' sort — layout_planner.go and sort_index.go).
// ResetEngine deletes exactly these ranges, so foreign keys in a
// caller-supplied pebble.DB survive. When a new tag or index family is added,
// this list AND the reset test must grow together.
var resetTagPrefixes = [][]byte{
	[]byte("m\x00"),     // map
	[]byte("s\x00"),     // set
	[]byte("c\x00"),     // counter
	[]byte("mm\x00"),    // multimap
	[]byte("l\x00"),     // log
	[]byte("sl\x00"),    // stream log
	[]byte("jl\x00"),    // journal index
	[]byte("vec\x00"),   // vector embeddings
	[]byte("vecm\x00"),  // vector metadata
	[]byte("edge\x00"),  // graph forward adjacency
	[]byte("edger\x00"), // graph reverse adjacency
	[]byte("i\x00"),     // layout filter index
	[]byte("o\x00"),     // layout sort index
}

// ResetEngine implements [metaengine.EngineResetter]: it deletes every
// engine-owned key range in ONE atomic batch, returning the engine to its
// empty post-construction state so a journal replay rebuilds every collection
// from zero. Layout declarations (e.layouts) survive — a reset reverts the
// Store to its post-Plan state, so secondary indexes are rebuilt by the
// replay, not forgotten.
//
// In-memory sequence counters (log, multimap, stream, journal) deliberately
// KEEP advancing across a reset: sequence numbers must stay monotonic
// forever, so a consumer holding a pre-reset resumption token (journal seq >
// N) never skips replayed entries, and replayed keys can never collide with
// deleted ones.
//
// Serialized against counter/multimap/log seq operations via mu.
func (e *pebbleEngine) ResetEngine(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	batch := e.db.NewBatch()
	defer batch.Close()

	for _, prefix := range resetTagPrefixes {
		if err := batch.DeleteRange(prefix, nextKey(prefix), nil); err != nil {
			return fmt.Errorf("pebbleengine.ResetEngine: queue delete range %q: %w", prefix, err)
		}
	}

	if err := batch.Commit(e.writeOptions()); err != nil {
		return fmt.Errorf("pebbleengine.ResetEngine: commit: %w", err)
	}

	return nil
}

// Compile-time assertion: the engine satisfies the reset capability.
var _ metaengine.EngineResetter = (*pebbleEngine)(nil)
