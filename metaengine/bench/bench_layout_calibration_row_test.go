package bench_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	_ "github.com/go-sql-driver/mysql" // register mysql driver for calibration
	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver for calibration

	mysqlengine "github.com/larsartmann/go-cqrs-lite/metaengine/mysqlengine/v4"
	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_layout_calibration_row_test.go calibrates the LayoutRow cost constants
// (scoreEmbed / scoreNormalize in metaengine/layout_scoring.go) on real
// row-store engines: SQLite (file-backed B-Tree), Postgres, and MySQL/MariaDB.
//
// Embed side is measured through the engine's public MapBackend API (meta_map
// JSON-column reads and read-modify-write updates). Normalize side is measured
// against dedicated parent/child tables — the normalized layout: LEFT JOIN
// read and O(1) child-row insert.
//
// Postgres requires POSTGRES_TEST_DSN; MySQL requires MYSQL_TEST_DSN (see
// scripts/ephemeral-pg.sh and scripts/vm-mysql*.sh). Engines whose DSN is
// unset are skipped so the SQLite baseline always runs.
//
// Run:
//
//	cd metaengine/bench
//	GOWORK=off CGO_ENABLED=1 go test -tags "goexperiment.jsonv2" -run '^$' \
//	  -bench 'BenchmarkRowLayoutCalibration' -benchtime 2s ./...

// rowCalibSeq makes collection/table names unique per benchmark invocation so
// repeated runs against a persistent DSN never collide.
var rowCalibSeq atomic.Uint64

func rowCalibNonce(op string) string {
	return fmt.Sprintf("%s_%d", op, rowCalibSeq.Add(1))
}

// rowCalibHarness bundles an engine (embed-side ops) with a raw SQL handle
// (normalize-side tables) and the engine's placeholder/DDL dialect.
type rowCalibHarness struct {
	name   string
	eng    metaengine.Engine
	db     *sql.DB
	ph     func(n int) string
	tuple  phTuple
	double string // DOUBLE (sqlite/mysql/duckdb) or DOUBLE PRECISION (pg)
}

func qPH(n int) string { return strings.TrimSuffix(strings.Repeat("?,", n), ",") }

func dollarPH(n int) string {
	parts := make([]string, n)
	for i := range n {
		parts[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(parts, ",")
}

// newRowCalibHarnesses builds every available row-store harness.
func newRowCalibHarnesses(tb testing.TB) []rowCalibHarness {
	tb.Helper()

	var out []rowCalibHarness

	dir := tb.TempDir()
	sdb, err := sql.Open("sqlite", filepath.Join(dir, "calib.db"))
	if err != nil {
		tb.Fatalf("open sqlite: %v", err)
	}
	sdb.SetMaxOpenConns(1)

	seng, err := sqliteengine.NewSQLiteEngine(sdb)
	if err != nil {
		tb.Fatalf("sqlite engine: %v", err)
	}
	tb.Cleanup(func() { _ = seng.Close() })
	out = append(out, rowCalibHarness{
		name: "sqliteDisk", eng: seng, db: sdb,
		ph: qPH, tuple: qTuple, double: "DOUBLE",
	})

	if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		peng, err := pgengine.New(dsn)
		if err != nil {
			tb.Fatalf("pg engine: %v", err)
		}
		pdb, err := sql.Open("pgx", dsn)
		if err != nil {
			tb.Fatalf("open pgx: %v", err)
		}
		tb.Cleanup(func() { _ = peng.Close(); _ = pdb.Close() })
		out = append(out, rowCalibHarness{
			name: "postgres", eng: peng, db: pdb,
			ph: dollarPH, tuple: dollarTuple, double: "DOUBLE PRECISION",
		})
	}

	if dsn := os.Getenv("MYSQL_TEST_DSN"); dsn != "" {
		meng, err := mysqlengine.New(dsn)
		if err != nil {
			tb.Fatalf("mysql engine: %v", err)
		}
		mdb, err := sql.Open("mysql", dsn)
		if err != nil {
			tb.Fatalf("open mysql: %v", err)
		}
		tb.Cleanup(func() { _ = meng.Close(); _ = mdb.Close() })
		out = append(out, rowCalibHarness{
			name: "mysql", eng: meng, db: mdb,
			ph: qPH, tuple: qTuple, double: "DOUBLE",
		})
	}

	return out
}

// createNormTables creates parent/child tables for the normalize layout.
func (h rowCalibHarness) createNormTables(parent, child string) error {
	stmts := []string{
		fmt.Sprintf(
			"CREATE TABLE %s (id VARCHAR(64) PRIMARY KEY, total %s, status VARCHAR(32))",
			parent, h.double),
		fmt.Sprintf(
			"CREATE TABLE %s (\n\t\t\tparent_id VARCHAR(64) NOT NULL, sku VARCHAR(64), qty BIGINT, price %s)",
			child,
			h.double,
		),
		fmt.Sprintf("CREATE INDEX %s_pid ON %s (parent_id)", child, child),
	}
	for _, s := range stmts {
		if _, err := h.db.Exec(s); err != nil {
			return fmt.Errorf("ddl: %w", err)
		}
	}
	return nil
}

// seedNorm populates parent headers and child rows via raw inserts.
func (h rowCalibHarness) seedNorm(parent, child string, n int) error {
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for i := range n {
		order := makeDiskCalibOrder(i)
		if _, err := tx.Exec(
			fmt.Sprintf("INSERT INTO %s (id, total, status) VALUES (%s)", parent, h.ph(3)),
			order.ID, order.Total, order.Status,
		); err != nil {
			return fmt.Errorf("seed parent: %w", err)
		}
		for _, item := range order.Items {
			if _, err := tx.Exec(
				fmt.Sprintf(
					"INSERT INTO %s (parent_id, sku, qty, price) VALUES (%s)",
					child,
					h.ph(4),
				),
				order.ID,
				item.SKU,
				item.Qty,
				item.Price,
			); err != nil {
				return fmt.Errorf("seed child: %w", err)
			}
		}
	}
	return tx.Commit()
}

// seedNormChunked bulk-seeds parent/child rows with multi-VALUES inserts
// (fast path for large storage-calibration datasets).
func (h rowCalibHarness) seedNormChunked(parent, child string, n int) error {
	parents := make([][]any, 0, n)
	children := make([][]any, 0, 4*n)

	for i := range n {
		order := makeDiskCalibOrder(i)
		parents = append(parents, []any{order.ID, order.Total, order.Status})
		for _, item := range order.Items {
			children = append(children, []any{order.ID, item.SKU, item.Qty, item.Price})
		}
	}

	if err := chunkedInsert(
		h.db,
		h.tuple,
		"INSERT INTO "+parent+" (id, total, status) VALUES ",
		parents,
	); err != nil {
		return err
	}

	return chunkedInsert(h.db, h.tuple,
		"INSERT INTO "+child+" (parent_id, sku, qty, price) VALUES ", children)
}

// seedEmbed populates the embed collection through the engine's MapBackend.
func seedEmbed(tb testing.TB, eng metaengine.Engine, col string, n int) {
	tb.Helper()
	mb := eng.(metaengine.MapBackend)
	ctx := context.Background()
	for i := range n {
		order := makeDiskCalibOrder(i)
		if err := mb.MapSet(ctx, col, order.ID, order); err != nil {
			tb.Fatalf("seed embed: %v", err)
		}
	}
}

// rmwReplaceChildren rewrites a decoded embed value (map form) with a fresh
// fixed-size child slice — the read-modify-write every embed child mutation
// pays (full decode + mutate + re-encode), at a constant value size so the
// per-op cost stays steady across iterations.
func rmwReplaceChildren(prev any) any {
	m, ok := prev.(map[string]any)
	if !ok {
		return prev
	}
	m["items"] = []any{
		map[string]any{"sku": "WIDGET-001", "qty": 2, "price": 10.25},
		map[string]any{"sku": "GADGET-002", "qty": 1, "price": 22.0},
		map[string]any{"sku": "GIZMO-003", "qty": 3, "price": 3.42},
		map[string]any{"sku": "EXTRA-999", "qty": 1, "price": 5.0},
	}
	return m
}

// BenchmarkRowLayoutCalibration_EmbedRead: single-key lookup returning the
// whole aggregate JSON (engine meta_map point read).
func BenchmarkRowLayoutCalibration_EmbedRead(b *testing.B) {
	for _, h := range newRowCalibHarnesses(b) {
		b.Run(h.name, func(b *testing.B) {
			col := "calib_embed_" + rowCalibNonce("er")
			seedEmbed(b, h.eng, col, 1000)

			mb := h.eng.(metaengine.MapBackend)
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()
			for i := range b.N {
				_, _, _ = mb.MapGet(ctx, col, fmt.Sprintf("order-%d", i%1000))
			}
		})
	}
}

// BenchmarkRowLayoutCalibration_EmbedWrite: child mutation = read-modify-write
// the parent JSON (MapGet + MapSet through the engine).
func BenchmarkRowLayoutCalibration_EmbedWrite(b *testing.B) {
	for _, h := range newRowCalibHarnesses(b) {
		b.Run(h.name, func(b *testing.B) {
			col := "calib_embed_" + rowCalibNonce("ew")
			seedEmbed(b, h.eng, col, 1000)

			mb := h.eng.(metaengine.MapBackend)
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
		})
	}
}

// scanJoinRows drains a LEFT JOIN result set (parent + child columns).
func scanJoinRows(rows *sql.Rows, err error) error {
	if err != nil {
		_ = rows.Close()
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var total, price float64
		var qty int64
		var status, sku sql.NullString
		if err := rows.Scan(&total, &status, &sku, &qty, &price); err != nil {
			return err
		}
	}
	return rows.Err()
}

// BenchmarkRowLayoutCalibration_NormalizeRead: parent + children fetched with
// one LEFT JOIN — the SQL-native normalized aggregate read.
func BenchmarkRowLayoutCalibration_NormalizeRead(b *testing.B) {
	for _, h := range newRowCalibHarnesses(b) {
		b.Run(h.name, func(b *testing.B) {
			parent := "calib_p_" + rowCalibNonce("nr")
			child := "calib_c_" + rowCalibNonce("nr")
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
				if err := scanJoinRows(h.db.QueryContext(ctx, query,
					fmt.Sprintf("order-%d", i%1000))); err != nil {
					b.Fatalf("query: %v", err)
				}
			}
		})
	}
}

// BenchmarkRowLayoutCalibration_NormalizeWrite: single O(1) child-row insert.
func BenchmarkRowLayoutCalibration_NormalizeWrite(b *testing.B) {
	for _, h := range newRowCalibHarnesses(b) {
		b.Run(h.name, func(b *testing.B) {
			parent := "calib_p_" + rowCalibNonce("nw")
			child := "calib_c_" + rowCalibNonce("nw")
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
		})
	}
}
