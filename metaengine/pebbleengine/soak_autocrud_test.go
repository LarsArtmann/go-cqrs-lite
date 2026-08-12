package pebbleengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_Pebble runs the AutoCRUDByConvention soak against the
// Pebble engine (LSM backend) to verify no memory leaks and CRUD lifecycle
// correctness under sustained write load. Pebble's LSM compaction and SST
// accumulation behave differently from the Memory engine's map — this test
// catches backend-specific leaks.
func TestSoak_AutoCRUD_Pebble(t *testing.T) {
	t.Parallel()

	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		t.Fatalf("NewPebbleEngine: %v", err)
	}

	// store.Close() inside RunAutoCRUDSoak closes the engine.
	enginetest.RunAutoCRUDSoak(t, eng)
}
