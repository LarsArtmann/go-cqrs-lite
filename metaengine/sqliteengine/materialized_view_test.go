package sqliteengine

import (
	"database/sql"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Plain SQLite has no CREATE MATERIALIZED VIEW — declaring specs must fail
// construction loudly (not silently) with a hint pointing at the feature.
func TestNewSQLiteEngine_MaterializedViewsFailLoudly(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer func() { _ = db.Close() }()

	_, err = NewSQLiteEngine(db, WithMaterializedViews([]metaengine.MaterializedViewSpec{
		{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"},
	}))
	if err == nil {
		t.Fatal("expected construction error for materialized views on plain SQLite, got nil")
	}

	if !strings.Contains(err.Error(), "experimental") {
		t.Fatalf("error should hint at the experimental feature, got: %v", err)
	}
}

// A spec that fails validation must fail construction before any DDL runs.
func TestNewSQLiteEngine_InvalidSpecFails(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer func() { _ = db.Close() }()

	_, err = NewSQLiteEngine(db, WithMaterializedViews([]metaengine.MaterializedViewSpec{
		{Collection: "orders", Fn: "MEDIAN", Column: "amount"},
	}))
	if err == nil {
		t.Fatal("expected validation error for unsupported fn, got nil")
	}
}
