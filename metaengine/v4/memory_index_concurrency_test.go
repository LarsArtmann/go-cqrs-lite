package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestMemoryIndexConcurrency_Memory pins M05: the brute-force memory indexes
// (vector, spatial, full-text search) are safe under concurrent use. The
// assertions are race-detector bait — run with -race for the full contract
// (unsynchronized map access between readers and writers fails the suite).
func TestMemoryIndexConcurrency_Memory(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = eng.Close() })

	adttest.AssertConcurrentVectorInsert(t, eng)
	adttest.AssertConcurrentScanDuringWrite(t, eng)
}
