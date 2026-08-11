package dgraphengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDgraph_ScanBackend exercises the standard ScanBackend.MapScan contract:
// unfiltered scan, filter by category, sort by price descending, and keyset
// pagination. This is the only engine contract test dgraphengine was missing
// — MapScan was implemented but never validated against the shared harness.
func TestDgraph_ScanBackend(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	enginetest.RunScanBackendTest(t, eng, "products")
}
