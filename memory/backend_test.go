package memory_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/store"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func TestMemoryBackend(t *testing.T) {
	t.Parallel()
	store.RunBackendTests(t, func(t *testing.T) store.Backend {
		t.Helper()

		return memory.NewBackend()
	})
}
