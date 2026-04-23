package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func TestNewStoreFromConfig_Memory(t *testing.T) {
	t.Parallel()

	cfg := event.NewStoreConfig(event.WithBackend(event.BackendMemory))

	store, err := event.NewStoreFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewStoreFromConfig_DefaultIsMemory(t *testing.T) {
	t.Parallel()

	cfg := event.NewStoreConfig()

	store, err := event.NewStoreFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
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
