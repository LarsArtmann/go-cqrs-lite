package sql_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// benchJournalReader wires a JournalReader[string] over a minimal two-column
// table for scan-path benchmarking. Scan forwards the capacity hint to
// ScanSlice, mirroring the event/command/query store scan functions.
type benchJournalReader struct {
	db *sql.DB
	r  *sqlpkg.JournalReader[string]
}

func newBenchJournalReader(b *testing.B, rows int) *benchJournalReader {
	b.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(
		`CREATE TABLE bench_events (id TEXT NOT NULL, ts TEXT NOT NULL)`,
	); err != nil {
		b.Fatalf("create table: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("begin: %v", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO bench_events (id, ts) VALUES (?, ?)`)
	if err != nil {
		b.Fatalf("prepare: %v", err)
	}

	for i := range rows {
		if _, err := stmt.Exec(fmt.Sprintf("evt-%06d", i), fmt.Sprintf("%06d", i)); err != nil {
			b.Fatalf("insert %d: %v", i, err)
		}
	}

	if err := stmt.Close(); err != nil {
		b.Fatalf("close stmt: %v", err)
	}

	if err := tx.Commit(); err != nil {
		b.Fatalf("commit: %v", err)
	}

	scan := func(rows *sql.Rows, capacityHint int) ([]string, error) {
		return sqlpkg.ScanSlice(rows, func(r *sql.Rows) (string, error) {
			var id string
			if err := r.Scan(&id); err != nil {
				return "", err
			}

			return id, nil
		}, capacityHint)
	}

	return &benchJournalReader{
		db: db,
		r: &sqlpkg.JournalReader[string]{
			DB:                db,
			Dialect:           sqlpkg.SQLiteDialect{},
			CheckClosed:       func() error { return nil },
			SpanNameAll:       "bench.read_all",
			SpanNameFrom:      "bench.read_from",
			CountAttr:         "bench.count",
			ErrCodeAll:        "bench.read_all",
			ErrCodeReadFrom:   "bench.read_from",
			ErrCodeFromStart:  "bench.from_start",
			ErrCodeQueryStart: "bench.query_start",
			ErrCodeScan:       "bench.scan",
			EntityNoun:        "event",
			EntityNounPlural:  "events",
			Table:             "bench_events",
			AllColumns:        "id",
			TimestampColumn:   "ts",
			Scan:              scan,
		},
	}
}

// BenchmarkJournalReader_ScanPrealloc contrasts a capacity-hinted drain
// (limit-bounded ReadFrom, hint = limit capped at maxScanPrealloc) against an
// unbounded ReadAll (default growth from 64).
// Run: go test -bench BenchmarkJournalReader_ScanPrealloc -run '^$' ./storage/sql/.
func BenchmarkJournalReader_ScanPrealloc(b *testing.B) {
	const totalRows = 10_000

	ctx := context.Background()
	br := newBenchJournalReader(b, totalRows)

	b.Run("hinted-from-start-4096", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			items, err := br.r.ReadFrom(ctx, "", 4096)
			if err != nil {
				b.Fatalf("ReadFrom: %v", err)
			}

			if len(items) != 4096 {
				b.Fatalf("got %d rows, want %d", len(items), 4096)
			}
		}
	})

	b.Run("default-read-all", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			items, err := br.r.ReadAll(ctx)
			if err != nil {
				b.Fatalf("ReadAll: %v", err)
			}

			if len(items) != totalRows {
				b.Fatalf("got %d rows, want %d", len(items), totalRows)
			}
		}
	})
}
