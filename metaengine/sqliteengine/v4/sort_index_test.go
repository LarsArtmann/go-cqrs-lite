package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func newSortIndexEngine(t *testing.T, opts ...EngineOption) (*sqliteEngine, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:sqliteengine_sortidx_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	eng, err := NewSQLiteEngine(db, opts...)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	e, ok := eng.(*sqliteEngine)
	if !ok {
		t.Fatalf("engine is %T, want *sqliteEngine", eng)
	}

	return e, db
}

// TestWithSortIndex_CreatesExpressionIndex pins the emitted DDL: the same
// partial-index form the journal's idx_stream_log_event_id carries, over the
// declared collection and json_extract expression, honoring DESC.
func TestWithSortIndex_CreatesExpressionIndex(t *testing.T) {
	t.Parallel()

	e, db := newSortIndexEngine(t, WithSortIndex("captures", "$.last_seen", true))

	wantExpr := "json_extract(value, '$.last_seen')"
	var name, sqlText string
	err := db.QueryRow(
		`SELECT name, sql FROM sqlite_master WHERE type = 'index' AND name = ?`, "idx_map_sort_captures___last_seen",
	).Scan(&name, &sqlText)
	if err != nil {
		t.Fatalf("index not created: %v", err)
	}

	for _, want := range []string{"ON meta_map", wantExpr, "DESC", "WHERE json_valid(value)"} {
		if !strings.Contains(sqlText, want) {
			t.Errorf("index DDL missing %q: %s", want, sqlText)
		}
	}

	if len(e.sortIndexes) != 1 {
		t.Errorf("sortIndexes = %d, want 1", len(e.sortIndexes))
	}
}

// TestWithSortIndex_NormalizesBareFieldName pins that a bare field name
// ("last_seen") normalizes to the exact expression standard scans order by
// — an index whose expression differs by one byte is dead weight.
func TestWithSortIndex_NormalizesBareFieldName(t *testing.T) {
	t.Parallel()

	_, db := newSortIndexEngine(t, WithSortIndex("captures", "last_seen", false))

	var sqlText string
	if err := db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'idx_map_sort_captures___last_seen'`,
	).Scan(&sqlText); err != nil {
		t.Fatalf("index not created: %v", err)
	}

	if !strings.Contains(sqlText, "json_extract(value, '$.last_seen')") {
		t.Errorf("bare field must normalize to $.last_seen: %s", sqlText)
	}
	if strings.Contains(sqlText, "DESC") {
		t.Errorf("desc=false must not emit DESC: %s", sqlText)
	}
}

// TestWithSortIndex_PlannerUsesIndex is the reason the option exists: the
// standard sorted scan's ORDER BY must be SERVED by the index (no full
// collection scan + temp B-tree). EXPLAIN QUERY PLAN names the index when
// the planner picked it.
func TestWithSortIndex_PlannerUsesIndex(t *testing.T) {
	t.Parallel()

	_, db := newSortIndexEngine(t, WithSortIndex("captures", "$.last_seen", true))

	query, args := buildStandardScanQuery(
		"captures",
		nil,
		&metaengine.SortSpec{Column: "last_seen", Desc: true},
		nil,
		50,
	)

	plan, err := explainQueryPlan(t, db, query, args)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN: %v", err)
	}

	if !strings.Contains(plan, "idx_map_sort_captures___last_seen") {
		t.Errorf("planner did not use the sort index; plan: %s (query: %s)", plan, query)
	}
}

// TestWithSortIndex_ScanResultsIdentical pins correctness parity: the same
// keyset-paginated sorted scan returns identical pages with and without the
// index — the option may only change the plan, never the rows.
func TestWithSortIndex_ScanResultsIdentical(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plain, plainDB := newSortIndexEngine(t)
	indexed, _ := newSortIndexEngine(t, WithSortIndex("captures", "$.last_seen", true))

	seed := func(e metaengine.Engine) {
		mb := e.(metaengine.MapBackend) //nolint:forcetypeassert // engine under test implements it
		for i := 50; i >= 1; i-- {
			key := fmt.Sprintf("cap-%02d", i)
			value := fmt.Sprintf(`{"id":%q,"last_seen":%d}`, key, 1000-i)
			if err := mb.MapSet(ctx, "captures", key, value); err != nil {
				t.Fatalf("MapSet: %v", err)
			}
		}
	}
	seed(plain)
	seed(indexed)

	sortSpec := &metaengine.SortSpec{Column: "last_seen", Desc: true}

	drain := func(e metaengine.Engine) []string {
		var got []string

		var cursor any

		reader := e.(metaengine.RawScanReader) //nolint:forcetypeassert // engine under test implements it
		for {
			res, err := reader.ScanRawValues(ctx, "captures", nil, sortSpec, cursor, 7)
			if err != nil {
				t.Fatalf("ScanRawValues: %v", err)
			}

			for _, row := range res.Items {
				got = append(got, string(row))
			}

			if !res.HasMore || res.NextCursor == nil {
				return got
			}

			cursor = res.NextCursor
		}
	}

	plainRows := drain(plain)
	indexedRows := drain(indexed)

	if len(plainRows) != 50 || len(indexedRows) != 50 {
		t.Fatalf("page counts: plain=%d indexed=%d, want 50", len(plainRows), len(indexedRows))
	}

	for i := range plainRows {
		if plainRows[i] != indexedRows[i] {
			t.Fatalf("row %d differs: plain=%s indexed=%s", i, plainRows[i], indexedRows[i])
		}
	}

	// Newest-first: the first row carries last_seen 999 (cap-01).
	if !strings.Contains(indexedRows[0], `"last_seen":999`) {
		t.Errorf("first sorted row = %s, want the newest last_seen", indexedRows[0])
	}

	_ = plainDB
}

// TestWithSortIndex_IdempotentConstruction: re-creating the engine over the
// same database with the same declaration must not fail (IF NOT EXISTS).
func TestWithSortIndex_IdempotentConstruction(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:sqliteengine_sortidx_idem_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	for range 2 {
		eng, err := NewSQLiteEngine(db, WithSortIndex("captures", "$.last_seen", true))
		if err != nil {
			t.Fatalf("repeated construction: %v", err)
		}
		_ = eng.Close()
	}
}

// TestWithSortIndex_RejectsInvalidDeclarations: an empty collection or a
// path with no field fails at construction, not at first scan.
func TestWithSortIndex_RejectsInvalidDeclarations(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		collection string
		path       string
	}{
		"empty collection": {"", "$.last_seen"},
		"empty path":       {"captures", "$"},
	} {
		_, err := func() (metaengine.Engine, error) {
			db, dbErr := sql.Open("sqlite", "file::memory:?cache=shared")
			if dbErr != nil {
				return nil, dbErr
			}
			defer func() { _ = db.Close() }()

			return NewSQLiteEngine(db, WithSortIndex(tc.collection, tc.path, false))
		}()
		if err == nil {
			t.Errorf("%s: expected a construction error", name)
		}
	}
}

func explainQueryPlan(t *testing.T, db *sql.DB, query string, args []any) (string, error) {
	t.Helper()

	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		return "", err
	}
	defer func() { _ = rows.Close() }()

	var b strings.Builder

	for rows.Next() {
		var id, parent, notused int

		var detail string

		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			return "", err
		}

		b.WriteString(detail)
		b.WriteByte('\n')
	}

	return b.String(), rows.Err()
}
