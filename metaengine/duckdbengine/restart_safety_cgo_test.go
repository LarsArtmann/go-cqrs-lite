//go:build cgo

package duckdbengine_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDuckDBRestartSafety_StreamAndJournal verifies that reopening a persistent
// DuckDB database does NOT reset seq counters — which would cause silent key
// collisions and data loss (see enginetest.RunRestartSafetyTest).
func TestDuckDBRestartSafety_StreamAndJournal(t *testing.T) {
	t.Parallel()

	probe, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	if err := probe.Close(); err != nil {
		t.Fatalf("probe close: %v", err)
	}

	enginetest.RunRestartSafetyTest(t, func(path string) (metaengine.Engine, error) {
		return duckdbengine.New(path)
	})
}

// TestDuckDBRestartSafety_FromDB verifies seq seeding when using
// NewFromDB (caller-owned *sql.DB path).
func TestDuckDBRestartSafety_FromDB(t *testing.T) {
	t.Parallel()

	probe, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	if err := probe.Close(); err != nil {
		t.Fatalf("probe close: %v", err)
	}

	enginetest.RunRestartSafetyFromDBTest(t,
		func(dir string) (metaengine.Engine, error) {
			return duckdbengine.New(filepath.Join(dir, "test.duckdb"))
		},
		func(dir string) (metaengine.Engine, error) {
			db, err := sql.Open("duckdb", filepath.Join(dir, "test.duckdb"))
			if err != nil {
				return nil, fmt.Errorf("raw duckdb open: %w", err)
			}

			return duckdbengine.NewFromDB(db)
		},
	)
}
