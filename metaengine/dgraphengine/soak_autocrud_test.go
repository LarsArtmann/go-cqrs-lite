package dgraphengine_test

import (
	"os"

	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_Dgraph runs the AutoCRUDByConvention soak against the
// Dgraph engine to verify no memory leaks and CRUD lifecycle correctness
// under sustained write load. Dgraph's RAFT consensus write path (~2.5ms/write)
// makes this test slower than embedded engines (~115s for 46K events), so it
// skips in -short mode (handled inside RunAutoCRUDSoak).
//
// NOT parallel: RunAutoCRUDSoak asserts on the process-global heap.
func TestSoak_AutoCRUD_Dgraph(t *testing.T) {
	if os.Getenv("SOAK_SKIP_DGRAPH") == "1" {
		t.Skip("dgraph soak: skipped by SOAK_SKIP_DGRAPH=1 (~115s over the RAFT write path; this is why a plain #integration-dgraph run costs minutes while a -run filtered one is ~52s)")
	}

	eng := mustNewDgraphEngine(t)

	enginetest.RunAutoCRUDSoak(t, eng)
}
