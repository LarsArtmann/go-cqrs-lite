package turso_test

import (
	"context"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4"
)

func TestOpenSync_MemoryWithRemote(t *testing.T) {
	t.Parallel()

	_, err := turso.OpenSync(
		context.Background(),
		turso.DbPath(":memory:"),
		turso.RemoteURL("libsql://example.turso.io"),
		turso.AuthToken("test-token"),
	)
	if err == nil {
		t.Fatal("expected error for in-memory database with remote sync")
	}

	if !errors.Is(err, turso.ErrMemorySync) {
		t.Errorf("error = %v, want ErrMemorySync", err)
	}

	classification := errorfamily.Classify(err)
	if classification != errorfamily.Rejection {
		t.Errorf("Classify = %s, want Rejection", classification)
	}
}

func TestOpenSync_FilePathWithoutRemote(t *testing.T) {
	t.Parallel()

	dbPath := t.TempDir() + "/test.db"

	database, err := turso.Open(turso.DbPath(dbPath))
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

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
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

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	defer database.Close()

	if err := turso.InitSchema(context.Background(), database); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}
}

func TestNewEventStore(t *testing.T) {
	t.Parallel()

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	defer database.Close()

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

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	defer database.Close()

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

	database, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	defer database.Close()

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

func TestDbPath_IsZero(t *testing.T) {
	t.Parallel()

	if turso.DbPath("").IsZero() != true {
		t.Error("empty DbPath should be zero")
	}
	if turso.DbPath("/tmp/test.db").IsZero() != false {
		t.Error("non-empty DbPath should not be zero")
	}
}

func TestDbPath_String(t *testing.T) {
	t.Parallel()

	p := turso.DbPath("/tmp/test.db")
	if p.String() != "/tmp/test.db" {
		t.Errorf("DbPath.String() = %q, want %q", p.String(), "/tmp/test.db")
	}
}

func TestRemoteURL_IsZero(t *testing.T) {
	t.Parallel()

	if turso.RemoteURL("").IsZero() != true {
		t.Error("empty RemoteURL should be zero")
	}
	if turso.RemoteURL("libsql://example.turso.io").IsZero() != false {
		t.Error("non-empty RemoteURL should not be zero")
	}
}

func TestRemoteURL_String(t *testing.T) {
	t.Parallel()

	u := turso.RemoteURL("libsql://example.turso.io")
	if u.String() != "libsql://example.turso.io" {
		t.Errorf("RemoteURL.String() = %q, want %q", u.String(), "libsql://example.turso.io")
	}
}

func TestAuthToken_IsZero(t *testing.T) {
	t.Parallel()

	if turso.AuthToken("").IsZero() != true {
		t.Error("empty AuthToken should be zero")
	}
	if turso.AuthToken("secret").IsZero() != false {
		t.Error("non-empty AuthToken should not be zero")
	}
}

func TestAuthToken_String(t *testing.T) {
	t.Parallel()

	a := turso.AuthToken("secret")
	if a.String() != "secret" {
		t.Errorf("AuthToken.String() = %q, want %q", a.String(), "secret")
	}
}
