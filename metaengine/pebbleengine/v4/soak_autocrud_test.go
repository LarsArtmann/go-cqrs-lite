package pebbleengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_Pebble runs the AutoCRUDByConvention soak against the
// Pebble engine (LSM backend) to verify no memory leaks and CRUD lifecycle
// correctness under sustained write load. Pebble's LSM compaction and SST
// accumulation behave differently from the Memory engine's map — this test
// catches backend-specific leaks.
//
// NOT parallel: RunAutoCRUDSoak asserts on the process-global heap.
func TestSoak_AutoCRUD_Pebble(t *testing.T) {
	eng := newPebbleEngineOrSkip(t)

	// store.Close() inside RunAutoCRUDSoak closes the engine.
	enginetest.RunAutoCRUDSoak(t, eng)
}
