package sqliteengine_test

// Spike MF1b: same fold-lane measurement under the DEPLOYED pragma posture
// (WAL + synchronous=NORMAL, dnsblockd's tracking DSN), where per-commit
// fsync is amortized by WAL. Isolates the tx-overhead vs fsync share of the
// per-event cost.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestSpike_MF1b_FoldLaneWAL(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "wal.db") +
		"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatal(err)
	}

	q := metaengine.Query[spikeMF1List, spikeMF1Row]("spike_mf1_tracks",
		metaengine.Volume(1_000_000),
		metaengine.FilterOnField[spikeMF1Row]("domain", metaengine.FilterEq),
		metaengine.FilterOnField[spikeMF1Row]("blocked", metaengine.FilterEq),
		metaengine.SortOnField[spikeMF1Row]("createdUnix", true),
		metaengine.OnRecordTyped("track.recorded", spikeMF1Row{},
			func(_ record.Record, e spikeMF1Row) (int64, spikeMF1Row) { return e.ID, e }),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = store.Close() }()

	ctx := context.Background()
	start := time.Now()
	if err := store.ApplyBatch(ctx, spikeMF1Events(1000)); err != nil {
		t.Fatal(err)
	}
	t.Logf("WAL+NORMAL fold lane ApplyBatch(1000): %v", time.Since(start))

	sqlTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	txStart := time.Now()
	for i := range 1000 {
		if _, err := sqlTx.ExecContext(ctx,
			`INSERT INTO meta_planned_spike_mf1_tracks (key, value, domain, blocked, createdUnix)
			 VALUES (?, ?, ?, ?, ?)`,
			int64(50_000+i), "{}", "x", 0, 1); err != nil {
			t.Fatal(err)
		}
	}

	if err := sqlTx.Commit(); err != nil {
		t.Fatal(err)
	}

	t.Logf("WAL+NORMAL one-tx x1000 ctrl: %v", time.Since(txStart))
}
