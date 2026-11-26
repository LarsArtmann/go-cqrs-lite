package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// mustPlanFromMemory plans q on a fresh in-memory engine, failing the test on
// a plan error, and closes the store when the test ends.
func mustPlanFromMemory(t *testing.T, args ...any) *metaengine.Store {
	t.Helper()

	store, err := metaengine.PlanFromMemory(args...)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	return store
}
