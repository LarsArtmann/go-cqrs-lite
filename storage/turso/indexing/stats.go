package indexing

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// IndexUsageStats reports per-index statistics from the query planner.
type IndexUsageStats struct {
	Name      string
	Table     string
	Columns   []string
	HasStats  bool  // whether sqlite_stat1 entry exists
	RowEst    int64 // estimated row count
	SizeBytes int64 // approximate index size
}

// Stats scans sqlite_master and sqlite_stat1 to report index usage
// statistics. Useful for finding unused indexes that can be dropped.
//
// Requires the database to have been ANALYZEd at least once; otherwise
// the HasStats field will be false and RowEst will be zero.
func Stats(ctx context.Context, db *sql.DB) ([]IndexUsageStats, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT name, tbl_name, sql
		FROM sqlite_master
		WHERE type = 'index' AND sql IS NOT NULL
	`)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "indexing.list_indexes",
			"list indexes for stats")
	}

	defer func() { _ = rows.Close() }()

	var stats []IndexUsageStats

	for rows.Next() {
		var stat IndexUsageStats
		var sqlDDL string

		if err := rows.Scan(&stat.Name, &stat.Table, &sqlDDL); err != nil {
			return nil, event.WrapInfrastructure(err, "indexing.scan_index",
				"scan index for stats")
		}

		stat.Columns = extractIndexedColumns(sqlDDL)
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, event.WrapInfrastructure(err, "indexing.iterate_indexes",
			"iterate indexes for stats")
	}

	// Augment with sqlite_stat1 row estimates where available.
	stats = augmentWithStat1(ctx, db, stats)

	return stats, nil
}

// UnusedIndexes returns indexes that haven't been used by the query
// planner (no sqlite_stat1 entry). Requires prior ANALYZE to be useful.
func UnusedIndexes(ctx context.Context, db *sql.DB) ([]Index, error) {
	all, err := Stats(ctx, db)
	if err != nil {
		return nil, err //nolint:wrapcheck // transparent delegation
	}

	var unused []Index

	for _, s := range all {
		if !s.HasStats {
			unused = append(unused, Index{
				Name:    s.Name,
				Table:   s.Table,
				Columns: s.Columns,
				Reason:  "no sqlite_stat1 row — planner has not seen this index",
			})
		}
	}

	return unused, nil
}

func extractIndexedColumns(sqlDDL string) []string {
	open := strings.Index(sqlDDL, "(")
	close := strings.Index(sqlDDL, ")")
	if open < 0 || close < 0 || close <= open {
		return nil
	}

	cols := strings.Split(sqlDDL[open+1:close], ",")
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		c = strings.TrimSpace(c)
		if c != "" {
			out = append(out, c)
		}
	}

	return out
}

func augmentWithStat1(
	ctx context.Context,
	db *sql.DB,
	stats []IndexUsageStats,
) []IndexUsageStats {
	stat1, err := queryStat1(ctx, db)
	if err != nil {
		// Non-fatal; return stats as-is.
		return stats
	}

	byName := make(map[string]stat1Row, len(stat1))
	for _, r := range stat1 {
		byName[r.Name] = r
	}

	for i := range stats {
		if row, ok := byName[stats[i].Name]; ok {
			stats[i].HasStats = true
			stats[i].RowEst = row.Rows
		}
	}

	return stats
}

type stat1Row struct {
	Name string
	Rows int64
}

func queryStat1(ctx context.Context, db *sql.DB) ([]stat1Row, error) {
	rows, err := db.QueryContext(ctx, "SELECT idx, stat FROM sqlite_stat1")
	if err != nil {
		// sqlite_stat1 may not exist if ANALYZE has not been run.
		return nil, event.Wrapf(err, event.Infrastructure, "turso.stat1_query", "sqlite_stat1")
	}
	defer func() { _ = rows.Close() }()

	var out []stat1Row

	for rows.Next() {
		var name, stat string

		if err := rows.Scan(&name, &stat); err != nil {
			return nil, event.Wrapf(err, event.Infrastructure, "turso.stat1_scan", "stat=%v", stat)
		}

		rows := parseStat1Rows(stat)
		out = append(out, stat1Row{Name: name, Rows: rows})
	}

	if err := rows.Err(); err != nil {
		return out, event.Wrapf(
			err,
			event.Infrastructure,
			"turso.stat1_iter",
			"sqlite_stat1 iteration",
		)
	}

	return out, nil
}

func parseStat1Rows(stat string) int64 {
	// Format: "rows=12345 ...". Take the first integer.
	for part := range strings.FieldsSeq(stat) {
		if strings.HasPrefix(part, "rows=") {
			var n int64
			_, _ = fmt.Sscanf(part, "rows=%d", &n)
			return n
		}
	}

	return 0
}
