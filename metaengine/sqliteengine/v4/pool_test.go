package sqliteengine_test

import (
	"path/filepath"
	"testing"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
)

// WithMaxOpenConns must raise the pool ceiling above the constructors'
// serialized default of 1 — the read-heavy escape hatch (issue #112).
func TestWithMaxOpenConns_RaisesCeiling(t *testing.T) {
	eng, err := sqliteengine.NewSQLiteEngineFromDSNWith(
		filepath.Join(t.TempDir(), "pool.db"), nil,
		sqliteengine.WithMaxOpenConns(4),
	)
	if err != nil {
		t.Fatalf("NewSQLiteEngineFromDSNWith: %v", err)
	}

	defer eng.Close() //nolint:errcheck // engine owns the db

	if got := sqliteengine.DBStats(eng).MaxOpenConnections; got != 4 {
		t.Fatalf("MaxOpenConnections = %d, want 4", got)
	}
}

// n <= 0 keeps the serialized default: 0 means "unlimited" in database/sql,
// which is a footgun behind an option named "max".
func TestWithMaxOpenConns_NonPositiveKeepsDefault(t *testing.T) {
	for _, n := range []int{0, -1} {
		eng, err := sqliteengine.NewSQLiteEngineFromDSNWith(
			filepath.Join(t.TempDir(), "pool.db"), nil,
			sqliteengine.WithMaxOpenConns(n),
		)
		if err != nil {
			t.Fatalf("n=%d: NewSQLiteEngineFromDSNWith: %v", n, err)
		}

		if got := sqliteengine.DBStats(eng).MaxOpenConnections; got != 1 {
			t.Errorf("n=%d: MaxOpenConnections = %d, want 1 (default kept)", n, got)
		}

		if err := eng.Close(); err != nil {
			t.Errorf("n=%d: close: %v", n, err)
		}
	}
}

// NewFromDSN stays backward-compatible (no options = ceiling 1) and accepts
// the option for read-heavy deployments.
func TestNewFromDSN_PoolOption(t *testing.T) {
	eng, db, err := sqliteengine.NewFromDSN(filepath.Join(t.TempDir(), "pool.db"))
	if err != nil {
		t.Fatalf("NewFromDSN: %v", err)
	}

	if got := db.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("default MaxOpenConnections = %d, want 1", got)
	}

	_ = eng.Close()
	_ = db.Close()

	eng, db, err = sqliteengine.NewFromDSN(
		filepath.Join(t.TempDir(), "pool2.db"),
		sqliteengine.WithMaxOpenConns(8),
	)
	if err != nil {
		t.Fatalf("NewFromDSN with option: %v", err)
	}

	if got := db.Stats().MaxOpenConnections; got != 8 {
		t.Fatalf("optioned MaxOpenConnections = %d, want 8", got)
	}

	_ = eng.Close()
	_ = db.Close()
}
