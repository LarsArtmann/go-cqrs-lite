package mysqlengine_test

import (
	"testing"

	adttest "github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestMySQLEngine_WALConformance pins the journal-of-record identity
// contract (disk-truth A14): duplicate event appends and occupied
// (stream, version) slots CONFLICT via the typed WAL unique indexes instead
// of silently adding a second row. Scenario bodies live in adttest.
func TestMySQLEngine_WALConformance(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)

	adttest.AssertWALDuplicateAppendConflicts(t, eng)
	adttest.AssertWALStreamVersionConflict(t, eng)
}
