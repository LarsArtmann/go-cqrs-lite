package pgengine_test

import (
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_Postgres runs the AutoCRUDByConvention soak against the
// Postgres engine (JSONB + B-tree backend) to verify no memory leaks and CRUD
// lifecycle correctness under sustained write load. Postgres's MVCC bloat and
// JSONB storage differ from SQLite and Pebble — this test catches server-side
// issues (e.g. connection lifecycle, transaction leaks).
//
// NOT parallel: RunAutoCRUDSoak asserts on the process-global heap.
func TestSoak_AutoCRUD_Postgres(t *testing.T) {
	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	// store.Close() inside RunAutoCRUDSoak closes the engine.
	enginetest.RunAutoCRUDSoak(t, eng)
}
