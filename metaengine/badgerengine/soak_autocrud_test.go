package badgerengine_test

import (
	"testing"

	badgerengine "github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_Badger runs the AutoCRUDByConvention soak against the
// Badger engine (LSM backend) to verify no memory leaks and CRUD lifecycle
// correctness under sustained write load. Badger's LSM compaction and GC
// behavior differ from Pebble — this test catches LSM-specific issues
// (e.g. value log growth, compaction stall). Completes the LSM soak matrix
// alongside Pebble.
func TestSoak_AutoCRUD_Badger(t *testing.T) {
	t.Parallel()

	eng, err := badgerengine.NewBadgerEngine("")
	if err != nil {
		t.Fatalf("NewBadgerEngine: %v", err)
	}

	enginetest.RunAutoCRUDSoak(t, eng)
}
