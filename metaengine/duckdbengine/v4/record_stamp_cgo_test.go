//go:build cgo

package duckdbengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDuckDB_RecordStamping verifies that Record metadata (StreamID, Version)
// is correctly stamped into result fields when using AutoInsert through the
// DuckDB engine.
func TestDuckDB_RecordStamping(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)

	enginetest.RunRecordStampTest(t, eng)
}
