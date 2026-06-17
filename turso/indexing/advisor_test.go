package indexing_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/turso/v2"
	"github.com/larsartmann/go-cqrs-lite/turso/v2/indexing"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := turso.InitSchema(context.Background(), db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	return db
}

func TestAdvisor_AnalyzeQuery_NoScan(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	advisor := indexing.NewAdvisor(db)

	// Query that uses the existing primary key index — no scan.
	recs, err := advisor.AnalyzeQuery(context.Background(),
		"SELECT * FROM events WHERE id = ?", "dummy-id")
	if err != nil {
		t.Fatalf("AnalyzeQuery: %v", err)
	}

	if len(recs) != 0 {
		t.Errorf("expected 0 recommendations for PK lookup, got %d", len(recs))
	}
}

func TestAdvisor_AnalyzeQuery_DetectsScan(t *testing.T) {
	t.Parallel()

	// Use a fresh schema WITHOUT CQRS indexes.
	db, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := turso.InitSchema(context.Background(), db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	advisor := indexing.NewAdvisor(db)

	// Cursor pagination query — the base schema has no index on
	// (occurred_at, id), so this produces a SCAN. The advisor should
	// detect it and recommend idx_events_cursor.
	recs, err := advisor.AnalyzeQuery(context.Background(),
		"SELECT * FROM events ORDER BY occurred_at ASC, id ASC LIMIT ?",
		100)
	if err != nil {
		t.Fatalf("AnalyzeQuery: %v", err)
	}

	if len(recs) == 0 {
		t.Fatal("expected at least 1 recommendation for cursor pagination scan")
	}

	// Verify the correct index was recommended.
	found := false
	for _, r := range recs {
		if r.Index.Name == "idx_events_cursor" {
			found = true
			if r.Priority != indexing.PriorityCritical {
				t.Errorf("expected Critical priority, got %s", r.Priority)
			}
		}
	}

	if !found {
		names := make([]string, 0, len(recs))
		for _, r := range recs {
			names = append(names, r.Index.Name)
		}
		t.Errorf("expected recommendation for idx_events_cursor, got: %v", names)
	}
}

func TestAdvisor_AnalyzeTable(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	advisor := indexing.NewAdvisor(db)

	recs, err := advisor.AnalyzeTable(context.Background(), "events")
	if err != nil {
		t.Fatalf("AnalyzeTable: %v", err)
	}

	// With the base schema, some patterns may already be covered.
	// We just verify the function runs and returns deduplicated results.
	seen := make(map[string]bool)
	for _, r := range recs {
		if seen[r.Index.Name] {
			t.Errorf("duplicate recommendation: %s", r.Index.Name)
		}
		seen[r.Index.Name] = true
	}
}

func TestAdvisor_MissingIndexes(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	advisor := indexing.NewAdvisor(db)

	recs, err := advisor.MissingIndexes(context.Background())
	if err != nil {
		t.Fatalf("MissingIndexes: %v", err)
	}

	// Verify deduplication.
	seen := make(map[string]bool)
	for _, r := range recs {
		if seen[r.Index.Name] {
			t.Errorf("duplicate recommendation: %s", r.Index.Name)
		}
		seen[r.Index.Name] = true
	}
}

func TestAdvisor_ExistingIndexes(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	advisor := indexing.NewAdvisor(db)

	if err := advisor.ExistingIndexes(context.Background()); err != nil {
		t.Fatalf("ExistingIndexes: %v", err)
	}

	// idx_events_aggregate is created by InitSchema.
	if !advisor.HasIndex("idx_events_aggregate") {
		t.Error("expected idx_events_aggregate to exist")
	}
}

func TestAdvisor_HasIndex_Unknown(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	advisor := indexing.NewAdvisor(db)

	if advisor.HasIndex("nonexistent_index_xyz") {
		t.Error("expected false for unknown index before cache refresh")
	}
}

func TestAdvisor_recommendationFromDetail_nil(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	advisor := indexing.NewAdvisor(db)

	recs, err := advisor.AnalyzeQuery(context.Background(),
		"SELECT 1")
	if err != nil {
		t.Fatalf("AnalyzeQuery: %v", err)
	}

	// SELECT 1 should produce no recommendations.
	for _, r := range recs {
		if r.Index.Table != "" {
			t.Errorf("unexpected recommendation for SELECT 1: %+v", r)
		}
	}
}

func TestAdvisor_WithExcludedTables(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)

	// Insert a custom table that we will exclude.
	_, err := database.ExecContext(context.Background(),
		"CREATE TABLE audit_log (id TEXT PRIMARY KEY, message TEXT, created_at TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	advisor := indexing.NewAdvisor(database, indexing.WithExcludedTables("audit_log"))

	recs, err := advisor.MissingIndexes(context.Background())
	if err != nil {
		t.Fatalf("MissingIndexes: %v", err)
	}

	// audit_log should not appear in the missing index recommendations.
	for _, r := range recs {
		if r.Index.Table == "audit_log" {
			t.Errorf("audit_log should be excluded but found recommendation: %+v", r)
		}
	}
}

func benchAdvisor(b *testing.B, withIndexes bool) *sql.DB {
	b.Helper()

	db, err := turso.OpenInMemory()
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = db.Close() })

	if err := turso.InitSchema(context.Background(), db); err != nil {
		b.Fatal(err)
	}

	if withIndexes {
		_ = turso.ApplyCQRSIndexes(context.Background(), db)
	}

	// Seed events for the same aggregate to make the queries realistic.
	for i := 0; i < 1000; i++ {
		_, err := db.ExecContext(
			context.Background(),
			`INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at, created_at)
			 VALUES (?, 'TestEvent', 'Test', ?, ?, 1, '{}', 'json', '{}', datetime('now'), datetime('now'))`,
			fmt.Sprintf("evt-%d-%d", i, i%10),
			fmt.Sprintf("agg-%d", i%10),
			i+1,
		)
		if err != nil {
			b.Fatal(err)
		}
	}

	return db
}

func BenchmarkReadFrom_WithIndexes(b *testing.B) {
	db := benchAdvisor(b, true)
	ctx := context.Background()
	aggID := "agg-5"

	b.ResetTimer()

	for b.Loop() {
		rows, err := db.QueryContext(ctx,
			`SELECT * FROM events
			 WHERE aggregate_type = ? AND aggregate_id = ? AND occurred_at > '2020-01-01' AND id > 'evt-0-0'
			 ORDER BY occurred_at ASC, id ASC LIMIT 100`,
			"Test", aggID)
		if err == nil {
			_ = rows.Close()
		}
	}
}

func BenchmarkReadFrom_WithoutIndexes(b *testing.B) {
	db := benchAdvisor(b, false)
	ctx := context.Background()
	aggID := "agg-5"

	b.ResetTimer()

	for b.Loop() {
		rows, err := db.QueryContext(ctx,
			`SELECT * FROM events
			 WHERE aggregate_type = ? AND aggregate_id = ? AND occurred_at > '2020-01-01' AND id > 'evt-0-0'
			 ORDER BY occurred_at ASC, id ASC LIMIT 100`,
			"Test", aggID)
		if err == nil {
			_ = rows.Close()
		}
	}
}

func BenchmarkAdvisor_MissingIndexes(b *testing.B) {
	db := benchAdvisor(b, true)
	ctx := context.Background()
	advisor := indexing.NewAdvisor(db)

	b.ResetTimer()

	for b.Loop() {
		_, _ = advisor.MissingIndexes(ctx)
	}
}
