//go:build cgo

package duckdbengine_test

import (
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDuckDB_RecordStamping verifies that Record metadata (StreamID, Version)
// is correctly stamped into result fields when using AutoInsert through the
// DuckDB engine.
func TestDuckDB_RecordStamping(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer metaengine.DeferClose(eng)

	enginetest.RunRecordStampTest(t, eng)
}
