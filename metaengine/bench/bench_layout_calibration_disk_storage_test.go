package bench_test

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"

	"github.com/cockroachdb/pebble"

	bboltengine "github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	pebbleengine "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_layout_calibration_disk_storage_test.go measures REAL on-disk bytes of
// the embed vs normalize layouts on Pebble and bbolt — the LSM-family
// counterpart of BenchmarkRowLayoutCalibration_Storage (2026-08-15 precedent:
// measure per-engine bytes instead of inheriting an engine-independent JSON
// size model).
//
// The 3-projection CQRS model (summary + history + search) is REALIZED
// physically instead of computed arithmetically:
//
//	embed DB: every projection collection stores the full aggregate
//	          (3 x n MapSet of the whole order)
//	norm DB:  every projection collection stores the header, and child facts
//	          go once into a shared multimap (3 x n MapSet + 3n MultiAdd)
//
// Engine-specific effects the JSON model cannot see ARE captured: bbolt stores
// every multimap child under its own seq-suffixed key
// (mm\x00col\x00key\x00<20-digit-seq>, ~41 bytes) inside B+Tree pages; Pebble
// compresses SSTable blocks (repeated field names inside one embed blob
// compress better than many small child values spread across keys).
//
// Pebble needs the raw *pebble.DB handle: Close alone does NOT flush the
// memtable (data would stay in the WAL and SSTable compression would never
// apply), so writes go through the engine (NewPebbleEngineFromDB) and the
// bench calls db.Flush() before Close. bbolt writes pages on every commit;
// NoSync only skips fsyncs (file size is unaffected, seeding is fast).
//
// Runs once — use -benchtime=1x:
//
//	cd metaengine/bench
//	GOWORK=off go test -tags "goexperiment.jsonv2" -run '^$' \
//	  -bench 'BenchmarkDiskLayoutCalibration_Storage' -benchtime 1x .

// diskStorageProjections are the three projection collection names shared by
// both layouts (the CQRS norm: order summary + history + search).
var diskStorageProjections = [3]string{"order_summary", "order_history", "order_search"}

// diskStorageItemsCol is the shared child-fact multimap of the normalize
// layout.
const diskStorageItemsCol = "order_items"

// diskStorageSeedEmbed writes n full aggregates into every projection
// collection (embed layout: each projection duplicates the aggregate).
func diskStorageSeedEmbed(tb testing.TB, eng metaengine.Engine, n int) {
	tb.Helper()

	mb := eng.(metaengine.MapBackend)
	ctx := context.Background()

	for i := range n {
		order := makeDiskCalibOrder(i)
		for _, col := range diskStorageProjections {
			if err := mb.MapSet(ctx, col, order.ID, order); err != nil {
				tb.Fatalf("storage calib: seed embed %s/%s: %v", col, order.ID, err)
			}
		}
	}

	// Self-verification: a silent seeding failure (e.g. a swallowed error)
	// would shrink both sides and corrupt the ratio.
	if _, found, err := mb.MapGet(ctx, diskStorageProjections[0], "order-0"); err != nil || !found {
		tb.Fatalf("storage calib: embed seed verification failed: found=%v err=%v", found, err)
	}
}

// diskStorageSeedNorm writes n headers into every projection collection plus
// each child fact exactly once into the shared multimap (normalize layout).
func diskStorageSeedNorm(tb testing.TB, eng metaengine.Engine, n int) {
	tb.Helper()

	mb := eng.(metaengine.MapBackend)
	mm := eng.(metaengine.MultimapBackend)
	ctx := context.Background()

	for i := range n {
		order := makeDiskCalibOrder(i)
		header := diskCalibOrderHeader{ID: order.ID, Total: order.Total, Status: order.Status}
		for _, col := range diskStorageProjections {
			if err := mb.MapSet(ctx, col, order.ID, header); err != nil {
				tb.Fatalf("storage calib: seed norm header %s/%s: %v", col, order.ID, err)
			}
		}
		for _, item := range order.Items {
			if err := mm.MultiAdd(ctx, diskStorageItemsCol, order.ID, item); err != nil {
				tb.Fatalf("storage calib: seed norm item %s: %v", order.ID, err)
			}
		}
	}

	// Self-verification: order-0 must carry its three child facts.
	items, err := mm.MultiGet(ctx, diskStorageItemsCol, "order-0")
	if err != nil || len(items) != 3 {
		tb.Fatalf("storage calib: norm seed verification failed: items=%d err=%v", len(items), err)
	}
}

// dirSizeBytes sums every regular file under dir. Pebble stores a database as
// a directory (MANIFEST, OPTIONS, SSTables, WAL).
func dirSizeBytes(tb testing.TB, dir string) int64 {
	tb.Helper()

	var total int64
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		tb.Fatalf("storage calib: walk %s: %v", dir, err)
	}
	return total
}

// reportDiskStorageRatio reports the realized normalize/embed byte sizes. The
// embed side is already the 3-projection realization, so no x3 arithmetic —
// metric names match reportStorageRatio for comparable output.
func reportDiskStorageRatio(b *testing.B, embed, norm int64) {
	b.Helper()

	b.ReportMetric(float64(norm)/float64(embed), "norm/embed-bytes")
	b.ReportMetric(float64(embed), "embed-bytes-3x")
	b.ReportMetric(float64(norm), "norm-bytes")
}

// BenchmarkDiskLayoutCalibration_Storage measures on-disk bytes of the embed
// vs normalize layouts on the disk KV engines (Pebble, bbolt). Ratios feed the
// LSM storage constants in metaengine/layout_scoring.go. Runs once — use
// -benchtime=1x.
func BenchmarkDiskLayoutCalibration_Storage(b *testing.B) {
	const n = 10_000

	b.Run("pebbleDisk", func(b *testing.B) {
		embedDir := b.TempDir()
		embedDB, err := pebble.Open(filepath.Join(embedDir, "db"), &pebble.Options{})
		if err != nil {
			b.Fatalf("storage calib: pebble open embed: %v", err)
		}
		embedEng, err := pebbleengine.NewPebbleEngineFromDB(embedDB)
		if err != nil {
			b.Fatalf("storage calib: pebble engine embed: %v", err)
		}
		diskStorageSeedEmbed(b, embedEng, n)
		if err := embedDB.Flush(); err != nil {
			b.Fatalf("storage calib: pebble flush embed: %v", err)
		}
		if err := embedDB.Close(); err != nil {
			b.Fatalf("storage calib: pebble close embed: %v", err)
		}

		normDir := b.TempDir()
		normDB, err := pebble.Open(filepath.Join(normDir, "db"), &pebble.Options{})
		if err != nil {
			b.Fatalf("storage calib: pebble open norm: %v", err)
		}
		normEng, err := pebbleengine.NewPebbleEngineFromDB(normDB)
		if err != nil {
			b.Fatalf("storage calib: pebble engine norm: %v", err)
		}
		diskStorageSeedNorm(b, normEng, n)
		if err := normDB.Flush(); err != nil {
			b.Fatalf("storage calib: pebble flush norm: %v", err)
		}
		if err := normDB.Close(); err != nil {
			b.Fatalf("storage calib: pebble close norm: %v", err)
		}

		reportDiskStorageRatio(b, dirSizeBytes(b, embedDir), dirSizeBytes(b, normDir))
	})

	b.Run("bboltDisk", func(b *testing.B) {
		embedPath := filepath.Join(b.TempDir(), "embed.db")
		embedDB, err := bolt.Open(embedPath, 0o600, &bolt.Options{NoSync: true})
		if err != nil {
			b.Fatalf("storage calib: bbolt open embed: %v", err)
		}
		embedEng, err := bboltengine.NewBboltEngineFromDB(embedDB)
		if err != nil {
			b.Fatalf("storage calib: bbolt engine embed: %v", err)
		}
		diskStorageSeedEmbed(b, embedEng, n)
		embedSize := bboltUsedBytes(b, embedDB)
		if err := embedDB.Close(); err != nil {
			b.Fatalf("storage calib: bbolt close embed: %v", err)
		}

		normPath := filepath.Join(b.TempDir(), "norm.db")
		normDB, err := bolt.Open(normPath, 0o600, &bolt.Options{NoSync: true})
		if err != nil {
			b.Fatalf("storage calib: bbolt open norm: %v", err)
		}
		normEng, err := bboltengine.NewBboltEngineFromDB(normDB)
		if err != nil {
			b.Fatalf("storage calib: bbolt engine norm: %v", err)
		}
		diskStorageSeedNorm(b, normEng, n)
		normSize := bboltUsedBytes(b, normDB)
		if err := normDB.Close(); err != nil {
			b.Fatalf("storage calib: bbolt close norm: %v", err)
		}

		reportDiskStorageRatio(b, embedSize, normSize)
	})
}

// bboltUsedBytes returns the database size as seen by a read transaction:
// highest allocated page x page size. The FILE size is unusable — bbolt grows
// the mmap in power-of-two steps, so both sides quantize to the same round
// number (observed: 16 MiB for both layouts at n=10K, ratio 1.000).
func bboltUsedBytes(tb testing.TB, db *bolt.DB) int64 {
	tb.Helper()

	var size int64
	err := db.View(func(tx *bolt.Tx) error {
		size = tx.Size()
		return nil
	})
	if err != nil {
		tb.Fatalf("storage calib: bbolt size: %v", err)
	}
	return size
}
