package sqliteengine_test

// Spike MF1 (dnsblockd metaflood plan, m1.1a): flood-lane write mechanics on
// planned tables over sqliteengine.
//
// The dnsblockd telemetry flood targets planned-table collections written
// through the fold lane (Store.Apply/ApplyBatch/ApplyRecord -> dispatch ->
// mapSetPlanned -> single-row INSERT OR REPLACE). This spike answers:
//
//	S1: does ApplyBatch work on a planned-table fold today, and what does a
//	    read-back see (N events = N rows)?
//	S2: does Store.InTransaction + Apply work? (expected: nested RunInTx
//	    rejection — dispatch wraps every event's folds in RunInTx itself)
//	S3: baseline wall-times at N=1000 on a FILE-backed DB, fold lane vs
//	    MapSet lane vs raw SQL controls (autocommit singles, one-tx singles,
//	    one multi-row INSERT) — sizing the MF1.2/MF1.3 win.
//
// Findings recorded at the bottom of the file.

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type spikeMF1Row struct {
	ID          int64  `json:"id"`
	Domain      string `json:"domain"`
	ClientIP    string `json:"clientIp"`
	Blocked     bool   `json:"blocked"`
	CreatedUnix int64  `json:"createdUnix"`
}

type spikeMF1List struct{ Limit int }

func newSpikeMF1Store(t *testing.T) (*metaengine.Store, metaengine.Engine, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "spike_mf1.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	q := metaengine.Query[spikeMF1List, spikeMF1Row]("spike_mf1_tracks",
		metaengine.Volume(1_000_000),
		metaengine.FilterOnField[spikeMF1Row]("domain", metaengine.FilterEq),
		metaengine.FilterOnField[spikeMF1Row]("blocked", metaengine.FilterEq),
		metaengine.SortOnField[spikeMF1Row]("createdUnix", true),
		metaengine.OnRecordTyped("track.recorded", spikeMF1Row{},
			func(_ record.Record, e spikeMF1Row) (int64, spikeMF1Row) {
				return e.ID, e
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store, eng, db
}

func spikeMF1Events(n int) []metaengine.EventInput {
	events := make([]metaengine.EventInput, 0, n)
	for i := range n {
		events = append(events, metaengine.EventInput{
			Type: "track.recorded",
			Payload: spikeMF1Row{
				ID:          int64(i),
				Domain:      fmt.Sprintf("host%d.example.net", i%97),
				ClientIP:    fmt.Sprintf("192.168.1.%d", i%251),
				Blocked:     i%3 == 0,
				CreatedUnix: 1_700_000_000 + int64(i),
			},
		})
	}

	return events
}

func spikeMF1Count(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	return n
}

func TestSpike_MF1_FloodWritePath(t *testing.T) {
	store, eng, db := newSpikeMF1Store(t)
	ctx := context.Background()
	events := spikeMF1Events(1000)

	// S1: ApplyBatch works; read-back sees all rows.
	start := time.Now()
	if err := store.ApplyBatch(ctx, events); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}
	foldLane := time.Since(start)

	var table string
	if err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'meta_planned%'`,
	).Scan(&table); err != nil {
		t.Fatalf("planned table not found (planner did not materialize it?): %v", err)
	}

	if got := spikeMF1Count(t, db, table); got != 1000 {
		t.Fatalf("S1: %d rows after ApplyBatch(1000), want 1000", got)
	}

	// S2: InTransaction + Apply — the documented batch-tx route.
	txErr := store.InTransaction(ctx, func(ctx context.Context) error {
		return store.Apply(ctx, "track.recorded", spikeMF1Row{ID: 999999, CreatedUnix: 1})
	})
	t.Logf("S2: InTransaction+Apply err=%v", txErr)

	// S3: MapSet lane (direct engine write, autocommit per call).
	mb, _ := eng.(metaengine.MapBackend)
	mbStart := time.Now()
	for i := range 1000 {
		if err := mb.MapSet(ctx, "spike_mf1_tracks", int64(10_000+i), spikeMF1Row{
			ID: int64(10_000 + i), Domain: "mapset.example.net", CreatedUnix: 2_000_000_000,
		}); err != nil {
			t.Fatalf("MapSet %d: %v", i, err)
		}
	}
	mapSetLane := time.Since(mbStart)
	if got := spikeMF1Count(t, db, table); got != 2000 {
		t.Fatalf("S3 MapSet: %d rows, want 2000", got)
	}

	// S3 controls: raw SQL on a scratch table.
	if _, err := db.Exec(
		`CREATE TABLE spike_mf1_ctrl (id INTEGER PRIMARY KEY, domain TEXT, blocked INT, created INTEGER)`,
	); err != nil {
		t.Fatalf("create ctrl: %v", err)
	}

	ins := func(exec interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	}, id int64) error {
		_, err := exec.ExecContext(ctx,
			`INSERT INTO spike_mf1_ctrl (id, domain, blocked, created) VALUES (?, ?, ?, ?)`,
			id, "ctrl.example.net", 0, 1)
		return err
	}

	autoStart := time.Now()
	for i := range 1000 {
		if err := ins(db, int64(i)); err != nil {
			t.Fatalf("ctrl autocommit %d: %v", i, err)
		}
	}
	ctrlAutocommit := time.Since(autoStart)

	sqlTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("ctrl begin: %v", err)
	}

	oneTxStart := time.Now()
	for i := range 1000 {
		if err := ins(sqlTx, int64(10_000+i)); err != nil {
			t.Fatalf("ctrl one-tx %d: %v", i, err)
		}
	}

	if err := sqlTx.Commit(); err != nil {
		t.Fatalf("ctrl one-tx commit: %v", err)
	}

	ctrlOneTx := time.Since(oneTxStart)

	if _, err := db.Exec(
		`CREATE TABLE spike_mf1_ctrl2 (id INTEGER PRIMARY KEY, domain TEXT, blocked INT, created INTEGER)`,
	); err != nil {
		t.Fatalf("create ctrl2: %v", err)
	}

	multiStart := time.Now()
	var sb strings.Builder
	sb.WriteString(`INSERT INTO spike_mf1_ctrl2 (id, domain, blocked, created) VALUES `)
	args := make([]any, 0, 4*1000)
	for i := range 1000 {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?,?,?)")
		args = append(args, int64(i), "ctrl.example.net", 0, 1)
	}
	if _, err := db.Exec(sb.String(), args...); err != nil {
		t.Fatalf("ctrl multi-row: %v", err)
	}
	ctrlMultiRow := time.Since(multiStart)

	t.Logf("S3 timings N=1000 (file db):\n"+
		"  fold lane ApplyBatch : %v (per-event tx, single-row stmt)\n"+
		"  MapSet lane          : %v (autocommit per call)\n"+
		"  ctrl autocommit xN   : %v\n"+
		"  ctrl one-tx xN       : %v\n"+
		"  ctrl one multi-row   : %v",
		foldLane, mapSetLane, ctrlAutocommit, ctrlOneTx, ctrlMultiRow)
}

// ── SPIKE FINDINGS (filled after the run) ──
