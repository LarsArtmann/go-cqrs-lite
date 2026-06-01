package turso_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/turso"
)

func TestOpenSync_MemoryWithRemote(t *testing.T) {
	t.Parallel()

	_, err := turso.OpenSync(
		context.Background(),
		":memory:",
		"libsql://example.turso.io",
		"test-token",
	)
	if err == nil {
		t.Fatal("expected error for in-memory database with remote sync")
	}

	if !errors.Is(err, turso.ErrMemorySync) {
		t.Errorf("error = %v, want ErrMemorySync", err)
	}

	classification := event.Classify(err)
	if classification != event.Rejection {
		t.Errorf("Classify = %s, want Rejection", classification)
	}
}

func TestOpenSync_FilePathWithoutRemote(t *testing.T) {
	t.Parallel()

	dbPath := t.TempDir() + "/test.db"

	database, err := turso.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if database == nil {
		t.Fatal("expected non-nil db")
	}

	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestOpenInMemory(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}

	if database == nil {
		t.Fatal("expected non-nil db")
	}

	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestInitSchema(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}
	defer database.Close() //nolint:errcheck // test helper

	if err := turso.InitSchema(context.Background(), database); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}
}

func TestNewEventStore(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}
	defer database.Close() //nolint:errcheck // test helper

	store, err := turso.NewEventStore(database)
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewSnapshotStore(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}
	defer database.Close() //nolint:errcheck // test helper

	store, err := turso.NewSnapshotStore(database)
	if err != nil {
		t.Fatalf("NewSnapshotStore: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewCheckpointStore(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}
	defer database.Close() //nolint:errcheck // test helper

	store, err := turso.NewCheckpointStore(database)
	if err != nil {
		t.Fatalf("NewCheckpointStore: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBackwardCompatAliases(t *testing.T) {
	t.Parallel()

	if turso.OpenTurso == nil {
		t.Error("OpenTurso alias is nil")
	}

	if turso.OpenTursoInMemory == nil {
		t.Error("OpenTursoInMemory alias is nil")
	}

	if turso.TursoInitSchema == nil {
		t.Error("TursoInitSchema alias is nil")
	}

	if turso.NewTursoEventStore == nil {
		t.Error("NewTursoEventStore alias is nil")
	}

	if turso.NewTursoSnapshotStore == nil {
		t.Error("NewTursoSnapshotStore alias is nil")
	}

	if turso.NewTursoCheckpointStore == nil {
		t.Error("NewTursoCheckpointStore alias is nil")
	}

	if turso.OpenTursoSync == nil {
		t.Error("OpenTursoSync alias is nil")
	}
}
