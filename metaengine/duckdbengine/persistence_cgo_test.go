//go:build cgo

package duckdbengine_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
)

func TestDuckDBPersistence_InMemoryIsVolatile(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	if !eng.Profile().IsVolatile() {
		t.Error("in-memory DuckDB engine (dsn=\"\") should be volatile")
	}

	if eng.Profile().IsPersistent() {
		t.Error("in-memory DuckDB engine should not be persistent")
	}
}

func TestDuckDBPersistence_OnDiskIsPersistent(t *testing.T) {
	t.Parallel()

	dsn := filepath.Join(t.TempDir(), "test.duckdb")
	eng, err := duckdbengine.New(dsn)
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	if !eng.Profile().IsPersistent() {
		t.Error("on-disk DuckDB engine should be persistent")
	}

	if eng.Profile().IsVolatile() {
		t.Error("on-disk DuckDB engine should not be volatile")
	}
}

func TestDuckDBPersistence_FromDBIsPersistent(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer db.Close()

	eng, err := duckdbengine.NewFromDB(db)
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	if !eng.Profile().IsPersistent() {
		t.Error("DuckDB engine from a caller-owned DB should be persistent")
	}
}
