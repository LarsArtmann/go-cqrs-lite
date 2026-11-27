package indexing_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4/indexing"
)

func TestStats_RowEstAfterAnalyze(t *testing.T) {
	t.Parallel()

	db, err := turso.OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	for _, stmt := range []string{
		"CREATE TABLE stat_probe (a TEXT, b TEXT)",
		"CREATE INDEX idx_stat_probe_a ON stat_probe(a)",
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	for i := 0; i < 50; i++ {
		if _, err := db.ExecContext(ctx, "INSERT INTO stat_probe (a, b) VALUES (?, ?)", "a", "b"); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if err := indexing.Analyze(ctx, db); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	stats, err := indexing.Stats(ctx, db)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	for _, s := range stats {
		if s.Name != "idx_stat_probe_a" {
			continue
		}
		if !s.HasStats {
			t.Fatal("idx_stat_probe_a has no sqlite_stat1 row after ANALYZE")
		}
		if s.RowEst != 50 {
			t.Errorf("RowEst = %d, want 50", s.RowEst)
		}
		return
	}

	t.Fatal("idx_stat_probe_a missing from Stats")
}
