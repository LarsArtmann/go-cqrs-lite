package indexing

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// Pragma is a SQLite/LibSQL PRAGMA setting.
type Pragma struct {
	Name  string
	Value string
}

// DefaultOptimizations returns the recommended PRAGMAs for CQRS workloads
// on Turso/LibSQL databases.
func DefaultOptimizations() []Pragma {
	return []Pragma{
		{Name: "journal_mode", Value: "WAL"},
		{Name: "synchronous", Value: "NORMAL"},
		{Name: "cache_size", Value: "-64000"}, // 64 MB page cache
		{Name: "temp_store", Value: "MEMORY"},
		{Name: "page_size", Value: "4096"},
	}
}

// ApplyOptimizations sets all DefaultOptimizations PRAGMAs on the database.
func ApplyOptimizations(ctx context.Context, db *sql.DB) error {
	return applyPragmas(ctx, db, DefaultOptimizations())
}

// ApplyWAL enables Write-Ahead Logging for better read concurrency.
func ApplyWAL(ctx context.Context, db *sql.DB) error {
	return applyPragmas(ctx, db, []Pragma{{Name: "journal_mode", Value: "WAL"}})
}

// SetCacheSize sets the SQLite page cache size in pages (negative = KiB).
func SetCacheSize(ctx context.Context, db *sql.DB, pages int) error {
	return applyPragmas(ctx, db, []Pragma{
		{Name: "cache_size", Value: fmt.Sprintf("%d", pages)},
	})
}

// SetMemoryMap enables memory-mapped I/O for faster reads.
// Size is in bytes; 0 disables. Typical: 256MB (268435456).
func SetMemoryMap(ctx context.Context, db *sql.DB, size int64) error {
	return applyPragmas(ctx, db, []Pragma{
		{Name: "mmap_size", Value: fmt.Sprintf("%d", size)},
	})
}

// RunOptimize runs PRAGMA optimize to let SQLite update statistics.
// Silently skipped on LibSQL/Turso variants that do not support this pragma.
func RunOptimize(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "PRAGMA optimize")
	if err != nil && !isUnsupportedPragma(err) {
		return event.WrapInfrastructure(err, "indexing.optimize",
			"run PRAGMA optimize")
	}

	return nil
}

// Analyze runs ANALYZE to update table statistics for the query planner.
func Analyze(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "ANALYZE")
	if err != nil {
		return event.WrapInfrastructure(err, "indexing.analyze",
			"run ANALYZE")
	}

	return nil
}

// AnalyzeTable runs ANALYZE on a specific table.
func AnalyzeTable(ctx context.Context, db *sql.DB, table string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf("ANALYZE %s", table))
	if err != nil {
		return event.WrapInfrastructure(err, "indexing.analyze_table",
			"run ANALYZE "+table)
	}

	return nil
}

func applyPragmas(ctx context.Context, db *sql.DB, pragmas []Pragma) error {
	for _, p := range pragmas {
		sqlStr := "PRAGMA " + p.Name + " = " + p.Value // #nosec G202 — values come from trusted Pragma constants

		_, err := db.ExecContext(ctx, sqlStr)
		if err != nil && !isUnsupportedPragma(err) {
			return event.WrapInfrastructure(err, "indexing.pragma",
				"set PRAGMA "+p.Name)
		}
	}

	return nil
}

func isUnsupportedPragma(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "not a valid pragma") ||
		strings.Contains(msg, "unknown pragma") ||
		strings.Contains(msg, "unrecognized pragma")
}
