package badgerengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBadger_RecordStamping verifies that Record metadata (StreamID, Version)
// is correctly stamped into result fields when using AutoInsert through the
// Badger engine. This completes all-engine record-stamp parity alongside
// Memory, SQLite, Pebble, DuckDB, and Postgres.
func TestBadger_RecordStamping(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)

	enginetest.RunRecordStampTest(t, eng)
}
