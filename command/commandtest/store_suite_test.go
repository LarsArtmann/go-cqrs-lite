package commandtest_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4/commandtest"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestStoreSuite validates the commandtest suite itself by running it against
// the reference in-memory implementation. This mirrors how eventtest is tested.
func TestStoreSuite(t *testing.T) {
	t.Parallel()

	commandtest.RunStoreSuite(t, func(t *testing.T) commandtest.StoreSuite {
		store := memory.NewMemoryCommandStore()
		t.Cleanup(func() { _ = store.Close() })

		return store
	})
}
