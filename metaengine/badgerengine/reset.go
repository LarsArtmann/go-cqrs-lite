package badgerengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// resetTagPrefixes are the engine-owned keycodec tag prefixes (see the
// keycodec package key shapes). ResetEngine drops exactly these ranges — a
// caller-owned badger.DB shared with foreign keys keeps them. When keycodec
// gains a tag, this list AND the reset test must grow together (the test
// walks every ADT, so a missed tag fails it).
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
}

// ResetEngine implements [metaengine.EngineResetter]: it drops every
// engine-owned key range, returning the engine to its empty post-construction
// state so a journal replay rebuilds every collection from zero. Foreign keys
// in a caller-supplied DB (NewBadgerEngineFromDB) are never touched — the
// drop is scoped to the keycodec tag prefixes above.
//
// In-memory sequence counters (log, multimap, stream, journal) deliberately
// KEEP advancing across a reset: sequence numbers must stay monotonic
// forever, so a consumer holding a pre-reset resumption token (journal seq >
// N) never skips replayed entries, and replayed keys can never collide with
// deleted ones.
//
// Serialized against MapUpdate/StreamAppend via mu. Badger drops each prefix
// range internally; a mid-drop failure surfaces as an error with prior ranges
// already gone — rerun the reset (it is idempotent).
func (e *badgerEngine) ResetEngine(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.db.DropPrefix(resetTagPrefixes...); err != nil {
		return fmt.Errorf("badgerengine.ResetEngine: drop key ranges: %w", err)
	}

	return nil
}

// Compile-time assertion: the engine satisfies the reset capability.
var _ metaengine.EngineResetter = (*badgerEngine)(nil)
