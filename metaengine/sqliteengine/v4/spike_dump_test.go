package sqliteengine_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"database/sql"
	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestSpike_MF1_Dump(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "d.db"))
	if err != nil { t.Fatal(err) }
	defer db.Close()
	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil { t.Fatal(err) }
	q := metaengine.Query[spikeMF1List, spikeMF1Row]("spike_mf1_tracks",
		metaengine.Volume(1_000_000),
		metaengine.OnRecordTyped("track.recorded", spikeMF1Row{},
			func(_ record.Record, e spikeMF1Row) (int64, spikeMF1Row) { return e.ID, e }),
	)
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil { t.Fatal(err) }
	defer store.Close()
	t.Logf("plan: %+v", store.Plan())
	events := spikeMF1Events(3)
	start := time.Now()
	if err := store.ApplyBatch(context.Background(), events); err != nil { t.Fatal(err) }
	t.Logf("apply3: %v", time.Since(start))
	rows, err := db.Query("SELECT type, name FROM sqlite_master ORDER BY name")
	if err != nil { t.Fatal(err) }
	for rows.Next() {
		var typ, name string
		_ = rows.Scan(&typ, &name)
		t.Logf("%s %s", typ, name)
	}
	rows.Close()
	if r, ok := eng.(metaengine.PlannedTablesReporter); ok {
		infos, err := r.PlannedTables(context.Background())
		if err != nil { t.Fatal(err) }
		for _, i := range infos { t.Logf("planned: %+v", i) }
	}
}
