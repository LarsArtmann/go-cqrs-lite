package bboltengine

import (
	"bytes"
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	bolt "go.etcd.io/bbolt"
)

// resetTagPrefixes are the engine-owned MATERIALIZED keycodec prefixes (see
// the keycodec package key shapes) plus the two layout secondary index
// families ('i' filter, 'o' sort). The journal prefixes (l, sl, jl) are
// deliberately ABSENT: journal entries are facts on the ADR-0136 ladder
// (ADR-0143) — the replay source a reset rebuilds FROM, never derived data a
// reset clears. When a new tag or index family is added, this list AND the
// reset test must grow together.
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
// engine-owned materialized key prefix from the single cqrs_meta bucket in
// one write transaction. All engine data lives inside that one bucket, so
// foreign buckets in a caller-supplied DB (NewBboltEngineFromDB) are never
// touched. The JOURNAL (l/sl/jl prefixes) survives: its entries are facts
// (ADR-0136 top rung / ADR-0143) — the replay source, never derived data.
//
// bbolt serializes write transactions internally; the single-transaction
// sweep commits atomically (a partial reset cannot be observed or committed).
func (e *bboltEngine) ResetEngine(_ context.Context) error {
	if err := e.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}

		for _, prefix := range resetTagPrefixes {
			c := b.Cursor()

			for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
				if err := c.Delete(); err != nil {
					return fmt.Errorf("delete prefix %q: %w", prefix, err)
				}
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("bboltengine.ResetEngine: %w", err)
	}

	return nil
}

// Compile-time assertion: the engine satisfies the reset capability.
var _ metaengine.EngineResetter = (*bboltEngine)(nil)
