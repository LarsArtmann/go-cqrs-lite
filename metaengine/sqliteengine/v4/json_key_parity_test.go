package sqliteengine_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Issue #63: declarative filter/sort columns must resolve identically on the
// memory engine (closure fallback, struct rows) and the SQL engines (JSON
// keys). This parity leg runs the SAME json-key filter and sort against both
// engines and requires identical counts and identical row order.

type sessionRow struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"started_at"`
	Region    string    `json:"region"`
}

func sessionQuery() metaengine.QueryDecl[struct{}, sessionRow] {
	return metaengine.Query[struct{}, sessionRow](
		"sessions",
		metaengine.OnRecord(sessionRow{}, func(_ record.Record, s sessionRow) (string, sessionRow) {
			return s.ID, s
		}),
	)
}

func seedSessions(t *testing.T, eng metaengine.Engine) *metaengine.Store {
	t.Helper()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, sessionQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	base := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	rows := []sessionRow{
		{ID: "late", StartedAt: base.Add(18 * time.Hour), Region: "eu"},
		{ID: "early", StartedAt: base.Add(6 * time.Hour), Region: "us"},
		{ID: "mid", StartedAt: base.Add(12 * time.Hour), Region: "eu"},
	}
	for _, r := range rows {
		if err := store.Apply(context.Background(), "sessionRow", r); err != nil {
			t.Fatalf("Apply %s: %v", r.ID, err)
		}
	}

	return store
}

// TestJSONKeyParity_MemoryVsSQLite pins engine parity on the closure path.
func TestJSONKeyParity_MemoryVsSQLite(t *testing.T) { //nolint:tparallel
	t.Parallel()

	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	sqlEng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	defer func() { _ = sqlEng.Close() }()

	day := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	dayEnd := day.Add(24 * time.Hour)

	memStore := seedSessions(t, metaengine.NewMemoryEngine())
	sqlStore := seedSessions(t, sqlEng)

	memReader := metaengine.NewReader[sessionRow](memStore, "sessions")
	sqlReader := metaengine.NewReader[sessionRow](sqlStore, "sessions")

	memCount, err := memReader.Count(
		ctx,
		metaengine.WithFilter("started_at", metaengine.FilterGe, day),
	)
	if err != nil {
		t.Fatalf("memory Count: %v", err)
	}
	sqlCount, err := sqlReader.Count(
		ctx,
		metaengine.WithFilter("started_at", metaengine.FilterGe, day),
	)
	if err != nil {
		t.Fatalf("sqlite Count: %v", err)
	}
	if memCount != sqlCount {
		t.Errorf(
			"memory Count = %d, sqlite Count = %d — engines disagree on the same json-key filter",
			memCount,
			sqlCount,
		)
	}

	memScan, err := memReader.Scan(ctx,
		metaengine.WithFilter("started_at", metaengine.FilterLt, dayEnd),
		metaengine.WithSort("started_at", false),
		metaengine.WithLimit(3))
	if err != nil {
		t.Fatalf("memory Scan: %v", err)
	}
	sqlScan, err := sqlReader.Scan(ctx,
		metaengine.WithFilter("started_at", metaengine.FilterLt, dayEnd),
		metaengine.WithSort("started_at", false),
		metaengine.WithLimit(3))
	if err != nil {
		t.Fatalf("sqlite Scan: %v", err)
	}
	if len(memScan) != len(sqlScan) {
		t.Fatalf("memory Scan len = %d, sqlite = %d", len(memScan), len(sqlScan))
	}
	for i := range memScan {
		if memScan[i].ID != sqlScan[i].ID {
			t.Errorf(
				"row %d memory = %q, sqlite = %q — sort by json key disagrees",
				i,
				memScan[i].ID,
				sqlScan[i].ID,
			)
		}
	}
	if len(memScan) != 3 || memScan[0].ID != "early" || memScan[1].ID != "mid" ||
		memScan[2].ID != "late" {
		t.Errorf("expected [early mid late] ascending by started_at, got %v", memScan)
	}
}
