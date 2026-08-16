package mysqlengine

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

// Sort-pushdown dialect benchmarks: numeric ORDER BY over meta_map.
//
// MariaDB cannot text-sort JSON numbers ("10" < "2"), so jsonSortExprs
// emits a dual key (CAST(JSON_EXTRACT(...) AS DECIMAL(65,10)) primary,
// JSON_UNQUOTE(JSON_EXTRACT(...)) tiebreak). These benchmarks quantify the
// dialect overhead versus the single-expression forms each server could
// otherwise use, over a deterministic shuffled numeric dataset.
//
// Run against each server:
//
//	MYSQL_TEST_DSN="mariadb..." go test -bench BenchmarkSortPushdown -benchtime 20x -run '^$'
//	MYSQL_TEST_DSN="mysql..."   go test -bench BenchmarkSortPushdown -benchtime 20x -run '^$'
//
// Forms:
//   - dual:    the engine's MariaDB sort shape (CAST + UNQUOTE)
//   - unquote: single JSON_UNQUOTE control (valid on both servers)
//   - arrow:   single value->'$.p' control (MySQL JSON type; MariaDB 1064)
const sortBenchRows = 50_000

const (
	sortBenchDual = `SELECT ` + "`key`" + ` FROM meta_map WHERE collection = ?
		ORDER BY CAST(JSON_EXTRACT(value, '$.p') AS DECIMAL(65,10)),
		         JSON_UNQUOTE(JSON_EXTRACT(value, '$.p'))
		LIMIT 100`

	sortBenchUnquote = `SELECT ` + "`key`" + ` FROM meta_map WHERE collection = ?
		ORDER BY JSON_UNQUOTE(JSON_EXTRACT(value, '$.p'))
		LIMIT 100`

	sortBenchArrow = "SELECT `key` FROM meta_map WHERE collection = ?" + `
		ORDER BY value->'$.p'
		LIMIT 100`
)

// seedSortBench bulk-inserts shuffled numeric-priority rows.
func seedSortBench(tb testing.TB, e *mysqlEngine, col string) {
	tb.Helper()

	priorities := rand.Perm(sortBenchRows)

	var b strings.Builder
	written := 0

	flush := func() {
		if written == 0 {
			return
		}

		stmt := "INSERT INTO meta_map (collection, `key`, value) VALUES " +
			strings.TrimSuffix(b.String(), ",") +
			" ON DUPLICATE KEY UPDATE value = VALUES(value)"
		if _, err := e.conn().ExecContext(context.Background(), stmt); err != nil {
			tb.Fatalf("seed sort bench: %v", err)
		}

		b.Reset()
		written = 0
	}

	for i, p := range priorities {
		fmt.Fprintf(&b, "('%s','k%d','{\"p\":%d}'),", col, i, p)
		written++

		if written >= 1_000 {
			flush()
		}
	}

	flush()
}

func runSortBench(b *testing.B, query, label string) {
	e := newInternalEngine(b)

	col := "bench_sort"
	seedSortBench(b, e, col)

	b.Run(label, func(b *testing.B) {
		ctx := context.Background()

		b.ResetTimer()

		for range b.N {
			rows, err := e.conn().QueryContext(ctx, query, col)
			if err != nil {
				b.Skipf("sort form unsupported on this server: %v", err)
			}

			count := 0
			for rows.Next() {
				count++
			}

			if err := rows.Err(); err != nil {
				b.Fatalf("sort bench rows: %v", err)
			}

			if err := rows.Close(); err != nil {
				b.Fatalf("sort bench close: %v", err)
			}

			if count != 100 {
				b.Fatalf("sort form returned %d rows, want 100", count)
			}
		}
	})
}

// BenchmarkSortPushdown_Dual measures the engine's MariaDB dual-key sort.
func BenchmarkSortPushdown_Dual(b *testing.B) { runSortBench(b, sortBenchDual, "dual-key") }

// BenchmarkSortPushdown_Unquote measures the single JSON_UNQUOTE control.
func BenchmarkSortPushdown_Unquote(b *testing.B) { runSortBench(b, sortBenchUnquote, "unquote") }

// BenchmarkSortPushdown_Arrow measures the MySQL JSON-typed control.
func BenchmarkSortPushdown_Arrow(b *testing.B) { runSortBench(b, sortBenchArrow, "arrow") }
