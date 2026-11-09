package bboltengine_test

import (
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_Bbolt runs the AutoCRUDByConvention soak against the bbolt
// engine. Measured 509-1145s under load (2026-08-16; 1145s with ~20 concurrent
// agent processes) — above the full verify gate's 8m per-package timeout, which
// is why that gate exports SOAK_SKIP_BOLT=1 and this soak is covered by
// dedicated runs instead.
//
// Skips in -short mode. Skips when SOAK_SKIP_BOLT=1.
//
// NOT parallel: RunAutoCRUDSoak asserts on the process-global heap.
func TestSoak_AutoCRUD_Bbolt(t *testing.T) {
	if os.Getenv("SOAK_SKIP_BOLT") == "1" {
		t.Skip("bbolt soak: skipped by SOAK_SKIP_BOLT=1")
	}

	eng := mustNewBboltEngine(t)

	enginetest.RunAutoCRUDSoak(t, eng)
}
