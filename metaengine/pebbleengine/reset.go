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
//
// The journal prefixes (l, sl, jl) are deliberately ABSENT: journal entries
// are facts on the ADR-0136 ladder (ADR-0143) — the replay source a reset
// rebuilds FROM, never derived data a reset clears.
var resetTagPrefixes = [][]byte{
	[]byte("m\x00"),     // map
	[]byte("s\x00"),     // set
	[]byte("c\x00"),     // counter
	[]byte("mm\x00"),    // multimap
	[]byte("vec\x00"),   // vector embeddings
	[]byte("vecm\x00"),  // vector metadata
	[]byte("edge\x00"),  // graph forward adjacency
	[]byte("edger\x00"), // graph reverse adjacency
	[]byte("i\x00"),     // layout filter index
	[]byte("o\x00"),     // layout sort index
}

// ResetEngine implements [metaengine.EngineResetter]: it deletes every
// engine-owned materialized key range in ONE atomic batch. Layout
// declarations (e.layouts) survive — a reset reverts the Store to its
// post-Plan state, so secondary indexes are rebuilt by the replay, not
// forgotten. The JOURNAL (l/sl/jl prefixes) survives: its entries are facts
// (ADR-0136 top rung / ADR-0143) — the replay source, never derived data.
//
// Serialized against counter/multimap/log seq operations via mu.
func (e *pebbleEngine) ResetEngine(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	batch := e.db.NewBatch()
	defer func() { _ = batch.Close() }()

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
