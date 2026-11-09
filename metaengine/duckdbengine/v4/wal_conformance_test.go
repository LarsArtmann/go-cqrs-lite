package duckdbengine_test

import (
	"path/filepath"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	adttest "github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestDuckDBEngine_WALConformance pins the journal-of-record identity
// contract (disk-truth A14): duplicate event appends and occupied
// (stream, version) slots CONFLICT via the typed WAL unique indexes instead
// of silently adding a second row. Scenario bodies live in adttest.
func TestDuckDBEngine_WALConformance(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New(filepath.Join(t.TempDir(), "wal_conf.duckdb"))
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	t.Cleanup(func() { metaengine.DeferClose(eng) })

	adttest.AssertWALDuplicateAppendConflicts(t, eng)
	adttest.AssertWALStreamVersionConflict(t, eng)
}
