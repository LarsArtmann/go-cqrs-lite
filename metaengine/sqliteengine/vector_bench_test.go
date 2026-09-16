package sqliteengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// benchVectorCorpus builds n deterministic embeddings of dim dimensions.
func benchVectorCorpus(n, dim int) []metaengine.Embedding {
	embs := make([]metaengine.Embedding, n)
	for i := range embs {
		values := make([]float32, dim)
		for d := range values {
			values[d] = float32((i*7 + d*13) % 97) / 97
		}
		embs[i] = metaengine.Embedding{ID: fmt.Sprintf("v%d", i), Values: values}
	}

	return embs
}

func newBenchSQLiteEngine(b *testing.B) metaengine.Engine {
	b.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	b.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		b.Fatalf("NewSQLiteEngine: %v", err)
	}
	b.Cleanup(func() { _ = eng.Close() })

	return eng
}

// setupVectorCorpus inserts n embeddings and returns the query vector.
func setupVectorCorpus(b *testing.B, eng metaengine.Engine, n, dim int) []float32 {
	b.Helper()

	ctx := context.Background()
	vb := eng.(metaengine.VectorBackend)

	for _, emb := range benchVectorCorpus(n, dim) {
		if err := vb.VectorInsert(ctx, "bench", emb); err != nil {
			b.Fatalf("VectorInsert: %v", err)
		}
	}

	query := make([]float32, dim)
	for d := range query {
		query[d] = 0.5
	}

	return query
}

// BenchmarkVectorSearch_GoScan measures the modernc path: rows scanned,
// scored in Go (O(N·D) per query).
func BenchmarkVectorSearch_GoScan(b *testing.B) {
	eng := newBenchSQLiteEngine(b)
	query := setupVectorCorpus(b, eng, 1000, 64)
	ctx := context.Background()
	vb := eng.(metaengine.VectorBackend)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := vb.VectorSearch(ctx, "bench", query, 10, "cosine"); err != nil {
			b.Fatalf("VectorSearch: %v", err)
		}
	}
}

// BenchmarkVectorInsert_GoScan measures insert cost including the
// dimension-lock probe (one indexed LIMIT 1 query per insert).
func BenchmarkVectorInsert_GoScan(b *testing.B) {
	eng := newBenchSQLiteEngine(b)
	ctx := context.Background()
	vb := eng.(metaengine.VectorBackend)

	if err := vb.VectorInsert(ctx, "ins", metaengine.Embedding{
		ID: "seed", Values: make([]float32, 64),
	}); err != nil {
		b.Fatalf("seed insert: %v", err)
	}

	emb := metaengine.Embedding{Values: make([]float32, 64)}

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		emb.ID = fmt.Sprintf("v%d", i)
		if err := vb.VectorInsert(ctx, "ins", emb); err != nil {
			b.Fatalf("VectorInsert: %v", err)
		}
	}
}
