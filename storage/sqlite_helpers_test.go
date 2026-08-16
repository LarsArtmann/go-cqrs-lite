package storage

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestOpenSQLiteInMemory_SharedCacheDatabase pins the shared-cache guarantee
// of OpenSQLiteInMemory. modernc.org/sqlite gives each pooled connection to
// "file::memory:" a private empty database; OpenSQLiteInMemory uses a unique
// named shared-cache DSN (file:<random>?mode=memory&cache=shared) so all
// pooled connections share one in-memory schema. This test verifies that a
// write on one connection is visible to a concurrent reader on another.
func TestOpenSQLiteInMemory_SharedCacheDatabase(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, "CREATE TABLE probe (k TEXT)"); err != nil {
		t.Fatalf("create probe table: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO probe (k) VALUES ('seed')"); err != nil {
		t.Fatalf("seed insert: %v", err)
	}

	// The transaction holds one connection; the concurrent write should
	// succeed on a different pooled connection AND see the same schema
	// (shared-cache DSN), not land on an empty private database.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	writeErr := make(chan error, 1)
	go func() {
		_, err := db.ExecContext(ctx, "INSERT INTO probe (k) VALUES ('concurrent')")
		writeErr <- err
	}()

	select {
	case err := <-writeErr:
		if err != nil {
			t.Fatalf("concurrent write on second connection failed: %v", err)
		}
		// Success — the second connection saw the probe table (shared cache).
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent write timed out — pool may be pinned to one connection")
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := db.ExecContext(ctx, "INSERT INTO probe (k) VALUES ('burst')"); err != nil {
				t.Errorf("burst write: %v", err)
			}
		}()
	}
	wg.Wait()

	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM probe").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if want := 2 + 8; n != want {
		t.Fatalf("expected %d rows in the shared database, got %d", want, n)
	}
}

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

func TestEnsureSQLiteDSNBusyTimeout(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		dsn  string
		want string
	}{
		{"plain_path", "/data/app.db", "/data/app.db?_pragma=busy_timeout(5000)"},
		{"file_uri", "file:test.db", "file:test.db?_pragma=busy_timeout(5000)"},
		{"memory", "file::memory:", "file::memory:?_pragma=busy_timeout(5000)"},
		{
			"existing_params",
			"file:db?_pragma=journal_mode(WAL)",
			"file:db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)",
		},
		{
			"already_has_busy_timeout",
			"db?_pragma=busy_timeout(10000)",
			"db?_pragma=busy_timeout(10000)",
		},
		{"custom_ms", "app.db", "app.db?_pragma=busy_timeout(15000)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ms := 5000
			if tc.name == "custom_ms" {
				ms = 15000
			}

			got := EnsureSQLiteDSNBusyTimeout(tc.dsn, ms)
			if got != tc.want {
				t.Errorf(
					"EnsureSQLiteDSNBusyTimeout(%q, %d) = %q, want %q",
					tc.dsn,
					ms,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestSQLiteEnableWAL_ClosedDB(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLite(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}

	if closeErr := db.Close(); closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}

	err = SQLiteEnableWAL(context.Background(), db)
	if err == nil {
		t.Fatal("expected error when enabling WAL on closed DB")
	}
}

func TestOpenSQLite_ClosedDB(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLite(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}

	if closeErr := db.Close(); closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}

	err = db.PingContext(context.Background())
	if err == nil {
		t.Fatal("expected error when pinging closed DB")
	}
}

func TestPostgresInitSchema_ExecError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	defer func() { _ = db.Close() }()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS events").
		WillReturnError(errors.New("connection refused"))

	err = PostgresInitSchema(context.Background(), db)
	if err == nil {
		t.Fatal("expected error from PostgresInitSchema")
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

	tables := []string{"events", "snapshots", "checkpoints"}

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

func TestSQLiteEnableWAL_BusyTimeout(t *testing.T) {
	db, err := OpenSQLite(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}

	defer func() { _ = db.Close() }()

	if err := SQLiteEnableWAL(context.Background(), db); err != nil {
		t.Fatalf("SQLiteEnableWAL: %v", err)
	}

	var busyTimeout int

	err = db.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}

	const expected = 5000

	if busyTimeout != expected {
		t.Errorf("busy_timeout = %d, want %d", busyTimeout, expected)
	}
}

func TestSQLiteEnableForeignKeys(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLite(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}

	defer func() { _ = db.Close() }()

	if err := SQLiteEnableForeignKeys(context.Background(), db); err != nil {
		t.Fatalf("SQLiteEnableForeignKeys: %v", err)
	}

	var fkEnabled int

	err = db.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}

	if fkEnabled != 1 {
		t.Errorf("foreign_keys = %d, want 1", fkEnabled)
	}
}
