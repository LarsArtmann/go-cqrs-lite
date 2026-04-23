package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

func TestNewStoreFromConfig_MemoryBackendReturnsError(t *testing.T) {
	t.Parallel()

	cfg := event.NewStoreConfig(event.WithBackend(event.BackendMemory))

	_, err := event.NewStoreFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error: memory store moved to memory module")
	}
}

func TestNewStoreFromConfig_DefaultReturnsError(t *testing.T) {
	t.Parallel()

	cfg := event.NewStoreConfig()

	_, err := event.NewStoreFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error: memory store moved to memory module")
	}
}

func TestNewStoreFromConfig_UnknownBackend(t *testing.T) {
	t.Parallel()

	cfg := event.NewStoreConfig(event.WithBackend("nonexistent"))

	_, err := event.NewStoreFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}
