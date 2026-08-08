package pgengine_test

import (
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestPostgres_RecordStamping verifies that Record metadata (StreamID,
// Version) is correctly stamped into result fields when using AutoInsert
// through the Postgres engine.
func TestPostgres_RecordStamping(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	enginetest.RunRecordStampTest(t, eng)
}
