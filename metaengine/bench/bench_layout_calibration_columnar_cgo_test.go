//go:build cgo

package bench_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2" // register duckdb driver for calibration

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_layout_calibration_columnar_cgo_test.go calibrates the LayoutColumnar
// cost constants (scoreEmbed / scoreNormalize in metaengine/layout_scoring.go)
// on a real file-backed DuckDB database.
//
// Embed side is measured through the engine's public MapBackend API (meta_map
// point reads and read-modify-write updates). Normalize side is measured
// against dedicated parent/child tables in a separate DuckDB file (DuckDB
// allows one writer per file): LEFT JOIN read and O(1) child-row insert.
//
// Run:
//
//	cd metaengine/bench
//	GOWORK=off CGO_ENABLED=1 go test -tags "goexperiment.jsonv2" -run '^$' \
//	  -bench 'BenchmarkColumnarLayoutCalibration' -benchtime 2s ./...

// newDuckEngineFile opens a DuckDB engine on its own file (embed-side ops).
func newDuckEngineFile(tb testing.TB) (metaengine.Engine, string) {
	tb.Helper()

	path := filepath.Join(tb.TempDir(), "embed.duckdb")
	eng, err := duckdbengine.New(path)
	if err != nil {
		tb.Skipf("DuckDB not available: %v", err)
	}
	tb.Cleanup(func() { _ = eng.Close() })

	return eng, path
}

// newDuckRawFile opens a raw DuckDB handle on its own file (normalize side).
func newDuckRawFile(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := sql.Open("duckdb", filepath.Join(tb.TempDir(), "norm.duckdb"))
	if err != nil {
		tb.Skipf("duckdb open: %v", err)
	}
	tb.Cleanup(func() { _ = db.Close() })

	return db
}

// BenchmarkColumnarLayoutCalibration_EmbedRead: single-key lookup returning
// the whole aggregate JSON (DuckDB meta_map point read).
func BenchmarkColumnarLayoutCalibration_EmbedRead(b *testing.B) {
	eng, _ := newDuckEngineFile(b)
	col := "calib_embed_" + rowCalibNonce("der")
	seedEmbed(b, eng, col, 1000)

	mb := eng.(metaengine.MapBackend)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := range b.N {
		_, _, _ = mb.MapGet(ctx, col, fmt.Sprintf("order-%d", i%1000))
	}
}

// BenchmarkColumnarLayoutCalibration_EmbedWrite: child mutation =
// read-modify-write the parent JSON.
func BenchmarkColumnarLayoutCalibration_EmbedWrite(b *testing.B) {
	eng, _ := newDuckEngineFile(b)
	col := "calib_embed_" + rowCalibNonce("dew")
	seedEmbed(b, eng, col, 1000)

	mb := eng.(metaengine.MapBackend)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := range b.N {
		key := fmt.Sprintf("order-%d", i%1000)
		prev, _, err := mb.MapGet(ctx, col, key)
		if err != nil {
			b.Fatalf("get: %v", err)
		}
		if err := mb.MapSet(ctx, col, key, rmwReplaceChildren(prev)); err != nil {
			b.Fatalf("set: %v", err)
		}
	}
}

// BenchmarkColumnarLayoutCalibration_NormalizeRead: parent + children in one
// LEFT JOIN — columnar-native child table read.
func BenchmarkColumnarLayoutCalibration_NormalizeRead(b *testing.B) {
	h := rowCalibHarness{name: "duckdbDisk", db: newDuckRawFile(b), ph: dollarPH, tuple: dollarTuple, double: "DOUBLE"}
	parent := "calib_p_" + rowCalibNonce("dnr")
	child := "calib_c_" + rowCalibNonce("dnr")
	if err := h.createNormTables(parent, child); err != nil {
		b.Fatal(err)
	}
	if err := h.seedNorm(parent, child, 1000); err != nil {
		b.Fatal(err)
	}

	query := fmt.Sprintf(`SELECT p.total, p.status, c.sku, c.qty, c.price
		FROM %s p LEFT JOIN %s c ON c.parent_id = p.id
		WHERE p.id = %s`, parent, child, h.ph(1))
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := range b.N {
		if err := scanJoinRows(h.db.QueryContext(ctx, query, fmt.Sprintf("order-%d", i%1000))); err != nil {
			b.Fatalf("query: %v", err)
		}
	}
}

// BenchmarkColumnarLayoutCalibration_NormalizeWrite: single child-row insert.
func BenchmarkColumnarLayoutCalibration_NormalizeWrite(b *testing.B) {
	h := rowCalibHarness{name: "duckdbDisk", db: newDuckRawFile(b), ph: dollarPH, tuple: dollarTuple, double: "DOUBLE"}
	parent := "calib_p_" + rowCalibNonce("dnw")
	child := "calib_c_" + rowCalibNonce("dnw")
	if err := h.createNormTables(parent, child); err != nil {
		b.Fatal(err)
	}
	if err := h.seedNorm(parent, child, 1000); err != nil {
		b.Fatal(err)
	}

	insert := fmt.Sprintf(
		"INSERT INTO %s (parent_id, sku, qty, price) VALUES (%s)", child, h.ph(4))
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := range b.N {
		if _, err := h.db.ExecContext(ctx, insert,
			fmt.Sprintf("order-%d", i%1000), "EXTRA-999", 1, 5.0,
		); err != nil {
			b.Fatalf("insert: %v", err)
		}
	}
}

// BenchmarkColumnarLayoutCalibration_Storage measures on-disk bytes of the
// embed vs normalize layouts in standalone DuckDB files (run once — use
// -benchtime=1x). Embed duplicates the aggregate across 3 projections.
func BenchmarkColumnarLayoutCalibration_Storage(b *testing.B) {
	dir := b.TempDir()

	embedPath := filepath.Join(dir, "embed.duckdb")
	edb, err := sql.Open("duckdb", embedPath)
	if err != nil {
		b.Skipf("duckdb open: %v", err)
	}
	if err := createEmbedTable(edb, "calib_embed_st", "varchar"); err != nil {
		b.Fatal(err)
	}
	if err := seedEmbedTable(edb, dollarTuple, "calib_embed_st", 20_000); err != nil {
		b.Fatal(err)
	}
	if err := edb.Close(); err != nil {
		b.Fatal(err)
	}

	normPath := filepath.Join(dir, "norm.duckdb")
	ndb, err := sql.Open("duckdb", normPath)
	if err != nil {
		b.Skipf("duckdb open: %v", err)
	}
	h := rowCalibHarness{db: ndb, ph: dollarPH, tuple: dollarTuple, double: "DOUBLE"}
	if err := h.createNormTables("calib_p_st", "calib_c_st"); err != nil {
		b.Fatal(err)
	}
	if err := h.seedNormChunked("calib_p_st", "calib_c_st", 20_000); err != nil {
		b.Fatal(err)
	}
	if err := ndb.Close(); err != nil {
		b.Fatal(err)
	}

	embedBytes := fileSize(b, embedPath)
	normBytes := fileSize(b, normPath)
	reportStorageRatio(b, embedBytes, normBytes)
	_ = os.Remove(embedPath)
	_ = os.Remove(normPath)
}
