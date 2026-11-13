package tursoengine

import (
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestNew_WithCellVersioning pins the turso option surface for ADR-0141
// temporal cells: WithCellVersioning forwards to the SQLite storage core,
// so the engine reports versioned cells and retention is applied.
func TestNew_WithCellVersioning(t *testing.T) {
	eng, err := New(":memory:", WithCellVersioning(metaengine.RetentionPolicy{MaxVersions: 3}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer eng.Close()

	if !metaengine.EngineVersionsCells(eng) {
		t.Fatal("EngineVersionsCells = false, want WithCellVersioning to enable versioning")
	}

	writer, ok := eng.(metaengine.VersionedWriter)
	if !ok {
		t.Fatal("turso engine does not implement VersionedWriter")
	}

	now := time.Now()
	if err := writer.MapSetAt(t.Context(), "users", "u1", "v1", now.Add(-time.Minute)); err != nil {
		t.Fatalf("MapSetAt: %v", err)
	}

	if err := writer.MapSetAt(t.Context(), "users", "u1", "v2", now); err != nil {
		t.Fatalf("MapSetAt: %v", err)
	}

	reader, ok := eng.(metaengine.CellHistoryReader)
	if !ok {
		t.Fatal("turso engine does not implement CellHistoryReader")
	}

	history, err := reader.MapHistory(t.Context(), "users", "u1", time.Time{}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("MapHistory: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2 (MaxVersions=3 keeps both)", len(history))
	}
}
