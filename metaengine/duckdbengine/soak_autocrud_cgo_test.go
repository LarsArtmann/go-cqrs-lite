//go:build cgo

package duckdbengine_test

import (
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_DuckDB runs the AutoCRUDByConvention soak against the
// DuckDB engine (columnar OLAP backend) to verify no memory leaks and CRUD
// lifecycle correctness under sustained write load. DuckDB's columnar storage
// and memory management differ from both Memory and Pebble — this test
// catches OLAP-specific issues (e.g. unbounded append, MVCC bloat).
func TestSoak_AutoCRUD_DuckDB(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer metaengine.DeferClose(eng)

	enginetest.RunAutoCRUDSoak(t, eng)
}
