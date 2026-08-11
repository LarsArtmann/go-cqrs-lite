package bench_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	bboltengine "github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	pebbleengine "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_layout_calibration_disk_test.go calibrates the layout-scoring cost
// multipliers (scoreEmbed / scoreNormalize in metaengine/layout_scoring.go)
// with REAL on-disk LSM/B+Tree engines.
//
// The Memory-only calibration in metaengine/layout_calibration_bench_test.go
// cannot measure disk effects (fsync, SSTable compaction, B+Tree page reads),
// so this bench drives Pebble and bbolt against a real disk directory.
//
// Run:
//   cd metaengine/bench
//   GOWORK=off go test -run='^$' -bench='BenchmarkDiskLayoutCalibration' -benchtime=0.5s ./...
//
// The per-engine ratios (normalize/embed for read and write) are what feed
// layout_scoring.go. Storage ratios are engine-independent and measured once.

// diskCalibOrder simulates an aggregate root with embedded child items,
// mirroring the Memory calibration fixture ("Embed" layout: one value per key).
type diskCalibOrder struct {
	ID     string          `json:"id"`
	Total  float64         `json:"total"`
	Status string          `json:"status"`
	Items  []diskCalibItem `json:"items"`
}

type diskCalibItem struct {
	SKU   string  `json:"sku"`
	Qty   int     `json:"qty"`
	Price float64 `json:"price"`
}

// diskCalibOrderHeader is the parent-only value in the "Normalize" layout.
type diskCalibOrderHeader struct {
	ID     string  `json:"id"`
	Total  float64 `json:"total"`
	Status string  `json:"status"`
}

func makeDiskCalibOrder(i int) diskCalibOrder {
	return diskCalibOrder{
		ID:     fmt.Sprintf("order-%d", i),
		Total:  42.50,
		Status: "pending",
		Items: []diskCalibItem{
			{SKU: "WIDGET-001", Qty: 2, Price: 10.25},
			{SKU: "GADGET-002", Qty: 1, Price: 22.00},
			{SKU: "GIZMO-003", Qty: 3, Price: 3.42},
		},
	}
}

func mustDiskJSON(b *testing.B, v any) []byte {
	b.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		b.Fatal(err)
	}
	return raw
}

// diskCalibEngines holds the two disk engines under test.
type diskCalibEngines struct {
	pebble metaengine.Engine
	bbolt  metaengine.Engine
}

// newDiskCalibEngines creates Pebble and bbolt engines on real on-disk temp
// directories. It registers cleanup to close engines and remove temp dirs.
func newDiskCalibEngines(b *testing.B) diskCalibEngines {
	b.Helper()

	engs := diskCalibEngines{}

	pebbleDir, err := os.MkdirTemp("", "metaengine-diskcalib-pebble-*")
	if err != nil {
		b.Fatalf("diskcalib: pebble mkdir: %v", err)
	}
	b.Cleanup(func() { _ = os.RemoveAll(pebbleDir) })

	peb, err := pebbleengine.NewPebbleEngine(pebbleDir)
	if err != nil {
		b.Fatalf("diskcalib: pebble: %v", err)
	}
	b.Cleanup(func() { _ = peb.Close() })
	engs.pebble = peb

	bboltDir, err := os.MkdirTemp("", "metaengine-diskcalib-bbolt-*")
	if err != nil {
		b.Fatalf("diskcalib: bbolt mkdir: %v", err)
	}
	b.Cleanup(func() { _ = os.RemoveAll(bboltDir) })

	bb, err := bboltengine.NewBboltEngine(filepath.Join(bboltDir, "calib.db"))
	if err != nil {
		b.Fatalf("diskcalib: bbolt: %v", err)
	}
	b.Cleanup(func() { _ = bb.Close() })
	engs.bbolt = bb

	return engs
}

// seedDiskCalibPopulates both engines with N embed-layout and normalize-layout
// rows so the measured ops hit warm in-memory caches where appropriate.
func seedDiskCalibPopulate(b *testing.B, eng metaengine.Engine, n int) {
	b.Helper()
	mb := eng.(metaengine.MapBackend)
	mm := eng.(metaengine.MultimapBackend)
	ctx := context.Background()

	for i := 0; i < n; i++ {
		order := makeDiskCalibOrder(i)
		if err := mb.MapSet(ctx, "orders_embed", order.ID, order); err != nil {
			b.Fatalf("seed embed: %v", err)
		}

		header := diskCalibOrderHeader{ID: order.ID, Total: order.Total, Status: order.Status}
		if err := mb.MapSet(ctx, "orders_norm", order.ID, header); err != nil {
			b.Fatalf("seed norm header: %v", err)
		}
		for _, item := range order.Items {
			if err := mm.MultiAdd(ctx, "items_norm", order.ID, item); err != nil {
				b.Fatalf("seed norm item: %v", err)
			}
		}
	}
}

// BenchmarkDiskLayoutCalibration_EmbedRead: single-key lookup returning the
// whole aggregate. This is what "Embed" costs on a real disk engine.
func BenchmarkDiskLayoutCalibration_EmbedRead(b *testing.B) {
	for _, eng := range []struct {
		name string
		eng  metaengine.Engine
	}{{"pebbleDisk", nil}, {"bboltDisk", nil}} {
		b.Run(eng.name, func(b *testing.B) {
			engs := newDiskCalibEngines(b)
			var e metaengine.Engine
			if eng.name == "pebbleDisk" {
				e = engs.pebble
			} else {
				e = engs.bbolt
			}
			seedDiskCalibPopulate(b, e, 1000)

			mb := e.(metaengine.MapBackend)
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _, _ = mb.MapGet(ctx, "orders_embed", fmt.Sprintf("order-%d", i%1000))
			}
		})
	}
}

// BenchmarkDiskLayoutCalibration_EmbedWrite: child mutation = read-modify-write
// the parent value (MapUpdate = Get + mutate + Set).
func BenchmarkDiskLayoutCalibration_EmbedWrite(b *testing.B) {
	for _, eng := range []struct {
		name string
		eng  metaengine.Engine
	}{{"pebbleDisk", nil}, {"bboltDisk", nil}} {
		b.Run(eng.name, func(b *testing.B) {
			engs := newDiskCalibEngines(b)
			var e metaengine.Engine
			if eng.name == "pebbleDisk" {
				e = engs.pebble
			} else {
				e = engs.bbolt
			}
			seedDiskCalibPopulate(b, e, 1000)

			mu := e.(metaengine.MapUpdater)
			ctx := context.Background()
			newItem := diskCalibItem{SKU: "EXTRA-999", Qty: 1, Price: 5.00}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("order-%d", i%1000)
				_ = mu.MapUpdate(ctx, "orders_embed", key, func(prev any) any {
					order, ok := prev.(diskCalibOrder)
					if !ok {
						return prev
					}
					order.Items = append(order.Items, newItem)
					return order
				})
			}
		})
	}
}

// BenchmarkDiskLayoutCalibration_NormalizeRead: parent lookup + child
// collection lookup + in-memory merge. This is what "Normalize" costs.
func BenchmarkDiskLayoutCalibration_NormalizeRead(b *testing.B) {
	for _, eng := range []struct {
		name string
		eng  metaengine.Engine
	}{{"pebbleDisk", nil}, {"bboltDisk", nil}} {
		b.Run(eng.name, func(b *testing.B) {
			engs := newDiskCalibEngines(b)
			var e metaengine.Engine
			if eng.name == "pebbleDisk" {
				e = engs.pebble
			} else {
				e = engs.bbolt
			}
			seedDiskCalibPopulate(b, e, 1000)

			mb := e.(metaengine.MapBackend)
			mm := e.(metaengine.MultimapBackend)
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("order-%d", i%1000)
				_, _, _ = mb.MapGet(ctx, "orders_norm", key)
				_, _ = mm.MultiGet(ctx, "items_norm", key)
			}
		})
	}
}

// BenchmarkDiskLayoutCalibration_NormalizeWrite: single O(1) append to the
// child collection (no parent rewrite).
func BenchmarkDiskLayoutCalibration_NormalizeWrite(b *testing.B) {
	for _, eng := range []struct {
		name string
		eng  metaengine.Engine
	}{{"pebbleDisk", nil}, {"bboltDisk", nil}} {
		b.Run(eng.name, func(b *testing.B) {
			engs := newDiskCalibEngines(b)
			var e metaengine.Engine
			if eng.name == "pebbleDisk" {
				e = engs.pebble
			} else {
				e = engs.bbolt
			}
			seedDiskCalibPopulate(b, e, 1000)

			mm := e.(metaengine.MultimapBackend)
			ctx := context.Background()
			newItem := diskCalibItem{SKU: "EXTRA-999", Qty: 1, Price: 5.00}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("order-%d", i%1000)
				_ = mm.MultiAdd(ctx, "items_norm", key, newItem)
			}
		})
	}
}
