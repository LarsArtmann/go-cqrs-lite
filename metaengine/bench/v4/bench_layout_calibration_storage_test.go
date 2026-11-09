package bench_test

import (
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"os"
	"strings"
	"testing"
)

// bench_layout_calibration_storage_test.go holds the shared storage-size
// calibration helpers and the Row-engine storage bench. Embed duplicates the
// aggregate across projections (measured as 3 copies, the CQRS norm of
// summary + history + search index); normalize stores each fact once.

const calibChunk = 500

// phTuple renders the placeholder tuple for row `row` (0-based) of a
// multi-VALUES insert with `cols` columns. Question-mark styles repeat `?`
// per position; dollar styles number parameters cumulatively across rows —
// repeating "$1, $2" would bind the SAME values to every row (DuckDB/PG).
type phTuple func(row, cols int) string

func qTuple(_, cols int) string {
	return strings.TrimSuffix(strings.Repeat("?,", cols), ",")
}

func dollarTuple(row, cols int) string {
	parts := make([]string, cols)
	for i := range cols {
		parts[i] = fmt.Sprintf("$%d", row*cols+i+1)
	}
	return strings.Join(parts, ",")
}

// chunkedInsert executes multi-VALUES inserts in calibChunk-sized batches.
// Batching matters: DuckDB and remote engines pay per-statement overhead that
// would dominate seeding time at 20K+ rows.
func chunkedInsert(
	db *sql.DB,
	tuple phTuple,
	prefix string,
	rows [][]any,
) error {
	if len(rows) == 0 {
		return nil
	}

	cols := len(rows[0])
	for start := 0; start < len(rows); start += calibChunk {
		end := min(start+calibChunk, len(rows))
		batch := rows[start:end]

		var b strings.Builder
		args := make([]any, 0, len(batch)*cols)
		b.WriteString(prefix)

		for j, row := range batch {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('(')
			b.WriteString(tuple(j, cols))
			b.WriteByte(')')
			args = append(args, row...)
		}

		if _, err := db.Exec(b.String(), args...); err != nil {
			return fmt.Errorf("chunked insert: %w", err)
		}
	}

	return nil
}

// embedValueType maps a storage-kind hint to the physical value column type:
// "jsonb" (Postgres), "json" (MySQL), anything else = "VARCHAR".
func embedValueType(kind string) string {
	switch kind {
	case "jsonb":
		return "JSONB NOT NULL"
	case "json":
		return "JSON NOT NULL"
	default:
		return "VARCHAR NOT NULL"
	}
}

// createEmbedTable creates the embed-standin table (one row per aggregate).
func createEmbedTable(db *sql.DB, table, kind string) error {
	_, err := db.Exec(fmt.Sprintf(
		"CREATE TABLE %s (k VARCHAR(64) PRIMARY KEY, v %s)", table, embedValueType(kind)))
	return err
}

// seedEmbedTable bulk-inserts n serialized orders into an embed-standin table.
func seedEmbedTable(db *sql.DB, tuple phTuple, table string, n int) error {
	rows := make([][]any, 0, n)
	for i := range n {
		order := makeDiskCalibOrder(i)
		raw, err := json.Marshal(order)
		if err != nil {
			return err
		}
		rows = append(rows, []any{order.ID, string(raw)})
	}
	return chunkedInsert(db, tuple, "INSERT INTO "+table+" (k, v) VALUES ", rows)
}

// fileSize stats a database file.
func fileSize(tb testing.TB, path string) int64 {
	tb.Helper()

	info, err := os.Stat(path)
	if err != nil {
		tb.Fatalf("stat %s: %v", path, err)
	}
	return info.Size()
}

// reportStorageRatio reports the embed(3 projections) vs normalize byte sizes.
func reportStorageRatio(b *testing.B, embedOne, norm int64) {
	b.Helper()

	embed3 := 3 * embedOne
	b.ReportMetric(float64(norm)/float64(embed3), "norm/embed-bytes")
	b.ReportMetric(float64(embed3), "embed-bytes-3x")
	b.ReportMetric(float64(norm), "norm-bytes")
}

// BenchmarkRowLayoutCalibration_Storage measures on-disk bytes of the embed vs
// normalize layouts on the row engines. SQLite measures whole files (one per
// layout); Postgres/MySQL measure catalog table sizes (DSN-gated). Runs once —
// use -benchtime=1x.
func BenchmarkRowLayoutCalibration_Storage(b *testing.B) {
	const n = 20_000

	b.Run("sqlite", func(b *testing.B) {
		embedOne, norm := rowStorageSizes(b, b.TempDir(), n)
		reportStorageRatio(b, embedOne, norm)
	})

	if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		b.Run("postgres", func(b *testing.B) {
			db, err := sql.Open("pgx", dsn)
			if err != nil {
				b.Fatalf("open pgx: %v", err)
			}
			defer func() { _ = db.Close() }()

			embedOne, norm := pgStorageSizes(b, db, n)
			reportStorageRatio(b, embedOne, norm)
		})
	}

	if dsn := os.Getenv("MYSQL_TEST_DSN"); dsn != "" {
		b.Run("mysql", func(b *testing.B) {
			db, err := sql.Open("mysql", dsn)
			if err != nil {
				b.Fatalf("open mysql: %v", err)
			}
			defer func() { _ = db.Close() }()

			embedOne, norm := mysqlStorageSizes(b, db, n)
			reportStorageRatio(b, embedOne, norm)
		})
	}
}

// rowStorageSizes builds standalone SQLite files for each layout and returns
// (embed-single-copy bytes, normalize bytes).
func rowStorageSizes(
	tb testing.TB,
	dir string,
	n int,
) (int64, int64) {
	tb.Helper()

	embedPath := dir + "/embed.db"
	edb, err := sql.Open("sqlite", embedPath)
	if err != nil {
		tb.Fatalf("open sqlite: %v", err)
	}
	if err := createEmbedTable(edb, "calib_embed_st", "varchar"); err != nil {
		tb.Fatal(err)
	}
	if err := seedEmbedTable(edb, qTuple, "calib_embed_st", n); err != nil {
		tb.Fatal(err)
	}
	if err := edb.Close(); err != nil {
		tb.Fatal(err)
	}

	normPath := dir + "/norm.db"
	ndb, err := sql.Open("sqlite", normPath)
	if err != nil {
		tb.Fatalf("open sqlite: %v", err)
	}
	h := rowCalibHarness{db: ndb, ph: qPH, tuple: qTuple, double: "DOUBLE"}
	if err := h.createNormTables("calib_p_st", "calib_c_st"); err != nil {
		tb.Fatal(err)
	}
	if err := h.seedNormChunked("calib_p_st", "calib_c_st", n); err != nil {
		tb.Fatal(err)
	}
	if err := ndb.Close(); err != nil {
		tb.Fatal(err)
	}

	return fileSize(tb, embedPath), fileSize(tb, normPath)
}

// pgStorageSizes seeds nonce tables on Postgres and measures them via
// pg_total_relation_size (heap + indexes + toast).
func pgStorageSizes(tb testing.TB, db *sql.DB, n int) (int64, int64) {
	tb.Helper()

	embed := "calib_embed_st_" + rowCalibNonce("pgs")
	parent := "calib_p_st_" + rowCalibNonce("pgs")
	child := "calib_c_st_" + rowCalibNonce("pgs")

	if err := createEmbedTable(db, embed, "jsonb"); err != nil {
		tb.Fatal(err)
	}
	if err := seedEmbedTable(db, dollarTuple, embed, n); err != nil {
		tb.Fatal(err)
	}

	h := rowCalibHarness{db: db, ph: dollarPH, tuple: dollarTuple, double: "DOUBLE PRECISION"}
	if err := h.createNormTables(parent, child); err != nil {
		tb.Fatal(err)
	}
	if err := h.seedNormChunked(parent, child, n); err != nil {
		tb.Fatal(err)
	}

	defer func() { dropAll(db, embed, parent, child) }()

	embedOne := pgTableSize(tb, db, embed)
	norm := pgTableSize(tb, db, parent) + pgTableSize(tb, db, child)

	return embedOne, norm
}

func pgTableSize(tb testing.TB, db *sql.DB, table string) int64 {
	tb.Helper()

	var size int64
	if err := db.QueryRow("SELECT pg_total_relation_size($1)", table).Scan(&size); err != nil {
		tb.Fatalf("pg size %s: %v", table, err)
	}
	return size
}

// mysqlStorageSizes seeds nonce tables on MySQL and measures them via
// information_schema (data_length + index_length).
func mysqlStorageSizes(tb testing.TB, db *sql.DB, n int) (int64, int64) {
	tb.Helper()

	embed := "calib_embed_st_" + rowCalibNonce("mys")
	parent := "calib_p_st_" + rowCalibNonce("mys")
	child := "calib_c_st_" + rowCalibNonce("mys")

	if err := createEmbedTable(db, embed, "json"); err != nil {
		tb.Fatal(err)
	}
	if err := seedEmbedTable(db, qTuple, embed, n); err != nil {
		tb.Fatal(err)
	}

	h := rowCalibHarness{db: db, ph: qPH, tuple: qTuple, double: "DOUBLE"}
	if err := h.createNormTables(parent, child); err != nil {
		tb.Fatal(err)
	}
	if err := h.seedNormChunked(parent, child, n); err != nil {
		tb.Fatal(err)
	}

	defer func() { dropAll(db, embed, parent, child) }()

	for _, t := range []string{embed, parent, child} {
		if _, err := db.Exec("ANALYZE TABLE " + t); err != nil {
			tb.Fatalf("analyze %s: %v", t, err)
		}
	}

	embedOne := mysqlTableSize(tb, db, embed)
	norm := mysqlTableSize(tb, db, parent) + mysqlTableSize(tb, db, child)

	return embedOne, norm
}

func mysqlTableSize(tb testing.TB, db *sql.DB, table string) int64 {
	tb.Helper()

	var size sql.NullInt64
	err := db.QueryRow(`SELECT data_length + index_length
		FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&size)
	if err != nil {
		tb.Fatalf("mysql size %s: %v", table, err)
	}
	return size.Int64
}

func dropAll(db *sql.DB, tables ...string) {
	for _, t := range tables {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + t)
	}
}
