package dgraphengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_Dgraph runs the AutoCRUDByConvention soak against the
// Dgraph engine to verify no memory leaks and CRUD lifecycle correctness
// under sustained write load. Dgraph's RAFT consensus write path (~2.5ms/write)
// makes this test slower than embedded engines (~115s for 46K events), so it
// skips in -short mode (handled inside RunAutoCRUDSoak).
func TestSoak_AutoCRUD_Dgraph(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	enginetest.RunAutoCRUDSoak(t, eng)
}
