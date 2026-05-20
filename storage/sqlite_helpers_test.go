package storage

import (
	"context"
	"testing"
)

func TestOpenSQLite_InMemory(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	defer func() { _ = db.Close() }()

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestOpenSQLite_WithFile(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}

	defer func() { _ = db.Close() }()

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestSQLiteEnableWAL(t *testing.T) {
	db, err := OpenSQLite(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}

	defer func() { _ = db.Close() }()

	err = SQLiteEnableWAL(context.Background(), db)
	if err != nil {
		t.Fatalf("SQLiteEnableWAL: %v", err)
	}

	var mode string

	err = db.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}

	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}

func TestConfigureSQLitePool(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	defer func() { _ = db.Close() }()

	ConfigureSQLitePool(db)

	stats := db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Errorf("MaxOpenConnections = %d, want 1", stats.MaxOpenConnections)
	}
}

func TestConfigureTursoPool(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	defer func() { _ = db.Close() }()

	ConfigureTursoPool(db)

	stats := db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Errorf("MaxOpenConnections = %d, want 1", stats.MaxOpenConnections)
	}
}

func TestSQLiteInitSchema_CreatesTables(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	defer func() { _ = db.Close() }()

	err = SQLiteInitSchema(context.Background(), db)
	if err != nil {
		t.Fatalf("SQLiteInitSchema: %v", err)
	}

	tables := []string{"events", "snapshots", "checkpoints", "outbox"}

	for _, table := range tables {
		var name string

		err := db.QueryRowContext(
			context.Background(),
			"SELECT name FROM sqlite_master WHERE type='table' AND name = ?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}
