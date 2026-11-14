package sqliteengine_test

// BenchmarkPlannedBatchFlood is the MF1.5 flood-lane profile: batched
// planned-table ingestion through the fold path (Store.ApplyBatch) under the
// dnsblockd deployed pragma posture (WAL + synchronous=NORMAL). One batch =
// one engine transaction + chunked multi-row INSERT OR REPLACE (MF1.2/MF1.3).
//
// Compare against dnsblockd's BatchWriter baseline
// (benchmarks/io-baseline.env, BenchmarkMixedIOThroughput medians) — the
// BatchWriter flushes a 500-command mix in one SQLite tx, so the comparable
// shape is b.N batches of batchRows events.
//
// MANUAL-ONLY (like BenchmarkGatewayStreams in dnsblockd): wall-clock
// sensitive; annotate co-tenant load (session census + uptime) when reporting.

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func BenchmarkPlannedBatchFlood(b *testing.B) {
	for _, batchSize := range []int{100, 500, 1000} {
		b.Run(fmt.Sprintf("batch=%d", batchSize), func(b *testing.B) {
			dsn := "file:" + filepath.Join(b.TempDir(), "flood.db") +
				"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)"

			db, err := sql.Open("sqlite", dsn)
			if err != nil {
				b.Fatal(err)
			}

			defer db.Close()

			eng, err := sqliteengine.NewSQLiteEngine(db)
			if err != nil {
				b.Fatal(err)
			}

			q := metaengine.Query[spikeMF1List, spikeMF1Row]("bench_flood",
				metaengine.Volume(1_000_000),
				metaengine.FilterOnField[spikeMF1Row]("domain", metaengine.FilterEq),
				metaengine.FilterOnField[spikeMF1Row]("blocked", metaengine.FilterEq),
				metaengine.SortOnField[spikeMF1Row]("createdUnix", true),
				metaengine.OnRecordTyped("track.recorded", spikeMF1Row{},
					func(_ record.Record, e spikeMF1Row) (int64, spikeMF1Row) { return e.ID, e }),
			)

			store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
			if err != nil {
				b.Fatal(err)
			}

			defer func() { _ = store.Close() }()

			ctx := context.Background()

			b.ResetTimer()

			for i := range b.N {
				events := make([]metaengine.EventInput, 0, batchSize)
				for j := range batchSize {
					events = append(events, metaengine.EventInput{
						Type: "track.recorded",
						Payload: spikeMF1Row{
							ID:          int64(i*batchSize + j),
							Domain:      fmt.Sprintf("host%d.example.net", j%97),
							ClientIP:    fmt.Sprintf("10.0.0.%d", j%251),
							Blocked:     j%3 == 0,
							CreatedUnix: 1_700_000_000 + int64(i*batchSize+j),
						},
					})
				}

				if err := store.ApplyBatch(ctx, events); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
