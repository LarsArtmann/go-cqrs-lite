package bboltengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	bolt "go.etcd.io/bbolt"
)

// ResetEngine implements [metaengine.EngineResetter]: it drops and recreates
// the engine's single cqrs_meta bucket in one write transaction, returning
// the engine to its empty post-construction state so a journal replay
// rebuilds every collection from zero. All engine data lives inside that one
// bucket under keycodec prefixes, so the drop is complete by construction;
// foreign buckets in a caller-supplied DB (NewBboltEngineFromDB) are never
// touched.
//
// In-memory sequence counters (log, multimap, stream, journal) deliberately
// KEEP advancing across a reset: sequence numbers must stay monotonic
// forever, so a consumer holding a pre-reset resumption token (journal seq >
// N) never skips replayed entries, and replayed keys can never collide with
// deleted ones.
//
// bbolt serializes write transactions internally; the drop-recreate pair
// commits atomically (a partial reset cannot be observed or committed).
func (e *bboltEngine) ResetEngine(_ context.Context) error {
	if err := e.db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket([]byte(bucketName)); err != nil {
			return fmt.Errorf("delete bucket: %w", err)
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
			return fmt.Errorf("recreate bucket: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("bboltengine.ResetEngine: %w", err)
	}

	return nil
}

// Compile-time assertion: the engine satisfies the reset capability.
var _ metaengine.EngineResetter = (*bboltEngine)(nil)
