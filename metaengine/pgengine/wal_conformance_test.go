package pgengine_test

import (
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	adttest "github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestPGEngine_WALConformance pins the journal-of-record identity contract
// (disk-truth A14): duplicate event appends and occupied (stream, version)
// slots CONFLICT via the typed WAL unique indexes instead of silently adding
// a second row. Scenario bodies live in adttest.
func TestPGEngine_WALConformance(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	t.Cleanup(func() { metaengine.DeferClose(eng) })

	adttest.AssertWALDuplicateAppendConflicts(t, eng)
	adttest.AssertWALStreamVersionConflict(t, eng)
}
