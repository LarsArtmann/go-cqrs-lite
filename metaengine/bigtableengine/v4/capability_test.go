package bigtableengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestCapabilityConformance verifies this engine's Profile() declarations
// against its implemented backend interfaces (declared-vs-implemented
// table), including the ADR-0142 write-side refusal entries (DueClaim /
// Dedup are refused with the missing-RMW reason, never silent).
func TestCapabilityConformance(t *testing.T) {
	t.Parallel()

	adttest.RunCapabilityConformance(t, "bigtable", newFakeEngine(t), nil)
}
