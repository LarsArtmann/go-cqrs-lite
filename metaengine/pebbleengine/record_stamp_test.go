package pebbleengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestPebble_RecordStamping verifies that Record metadata (StreamID, Version)
// is correctly stamped into result fields when using AutoInsert through the
// Pebble engine.
func TestPebble_RecordStamping(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)

	enginetest.RunRecordStampTest(t, eng)
}
