package metaengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUDByConvention processes ~46K events through the auto-projection
// path (AutoCRUDByConvention — zero-boilerplate CRUD folds derived from Go
// struct naming conventions) and verifies:
//
//  1. No memory leaks — heap growth is O(unique keys), not O(total events).
//  2. CRUD lifecycle correctness: created items exist, updates applied,
//     deleted items gone, re-created items restored.
//
// The soak logic lives in enginetest.RunAutoCRUDSoak so that Pebble, DuckDB,
// and PG engine modules can run the same workload against their backends.
func TestSoak_AutoCRUDByConvention(t *testing.T) {
	t.Parallel()

	enginetest.RunAutoCRUDSoak(t, metaengine.NewMemoryEngine())
}
