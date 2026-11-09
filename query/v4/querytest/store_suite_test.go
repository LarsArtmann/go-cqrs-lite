package querytest_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v4/querytest"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestStoreSuite validates the querytest suite itself by running it against
// the reference in-memory implementation, mirroring the commandtest self-test.
func TestStoreSuite(t *testing.T) {
	t.Parallel()

	querytest.RunStoreSuite(t, func(t *testing.T) querytest.StoreSuite {
		store := memory.NewMemoryQueryStore()
		t.Cleanup(func() { _ = store.Close() })

		return store
	})
}
