package dgraphengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDgraph_RecordStamping verifies that Record metadata (StreamID, Version)
// is correctly stamped into result fields when using AutoInsert through the
// Dgraph engine. Skips when no Dgraph instance is available (DGRAPH_ADDR env
// or localhost:9080 by default). This completes all-engine record-stamp parity
// for engines that implement MapBackend (7/8 — graphadapter is graph-only).
func TestDgraph_RecordStamping(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	enginetest.RunRecordStampTest(t, eng)
}
