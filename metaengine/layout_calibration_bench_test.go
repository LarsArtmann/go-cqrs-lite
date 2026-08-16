package metaengine_test

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// This file calibrates the layout-scoring cost multipliers in
// layout_scoring.go by measuring embed-vs-normalize read/write patterns on the
// memory engine (LayoutKV). The measured ratios replace placeholder constants.
// Disk engines (LayoutLSM) are calibrated separately by
// BenchmarkDiskLayoutCalibration_* in metaengine/bench.
//
// All write benches are SIZE-STABLE: the embed mutation REPLACES the child
// collection with a fixed-size slice (never appends), so the value shape is
// constant and per-op cost cannot drift as the run progresses. The pre-
// 2026-08-15 shape appended one child per iteration and grew values
// unboundedly mid-run.
//
// Run: GOWORK=off go test -tags "goexperiment.jsonv2" -run '^$' \
//	-bench 'BenchmarkLayoutCalibration' -benchtime 2s .

// calibOrder simulates an aggregate root with embedded child items.
// This is the "Embed" layout: the whole aggregate is one value.
type calibOrder struct {
	ID     string      `json:"id"`
	Total  float64     `json:"total"`
	Status string      `json:"status"`
	Items  []calibItem `json:"items"`
}

type calibItem struct {
	SKU   string  `json:"sku"`
	Qty   int     `json:"qty"`
	Price float64 `json:"price"`
}

// calibOrderHeader is the parent-only value in the "Normalize" layout.
type calibOrderHeader struct {
	ID     string  `json:"id"`
	Total  float64 `json:"total"`
	Status string  `json:"status"`
}

func makeCalibOrder(i int) calibOrder {
	return calibOrder{
		ID:     fmt.Sprintf("order-%d", i),
		Total:  42.50,
		Status: "pending",
		Items: []calibItem{
			{SKU: "WIDGET-001", Qty: 2, Price: 10.25},
			{SKU: "GADGET-002", Qty: 1, Price: 22.00},
			{SKU: "GIZMO-003", Qty: 3, Price: 3.42},
		},
	}
}

func mustJSONEncode(t *testing.B, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// BenchmarkLayoutCalibration_EmbedRead measures a single-key lookup returning
// the entire aggregate (embed layout read). This is the baseline for KV reads.
func BenchmarkLayoutCalibration_EmbedRead(b *testing.B) {
	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)
	ctx := context.Background()
	const N = 1000

	for i := 0; i < N; i++ {
		_ = mb.MapSet(ctx, "orders_embed", fmt.Sprintf("order-%d", i), makeCalibOrder(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = mb.MapGet(ctx, "orders_embed", fmt.Sprintf("order-%d", i%N))
	}
}

// calibFixedChildren is the post-mutation child slice: the three seeded items
// plus one changed child. Replacing items with this FIXED-size slice models a
// child mutation at constant value size (size-stable measurement).
var calibFixedChildren = []calibItem{
	{SKU: "WIDGET-001", Qty: 2, Price: 10.25},
	{SKU: "GADGET-002", Qty: 1, Price: 22.00},
	{SKU: "GIZMO-003", Qty: 3, Price: 3.42},
	{SKU: "EXTRA-999", Qty: 1, Price: 5.00},
}

// BenchmarkLayoutCalibration_EmbedWrite measures a child mutation under the
// embed layout: read-modify-write the parent (Get + replace child slice + Set).
// Replace-only keeps the value size constant so per-op cost cannot drift.
func BenchmarkLayoutCalibration_EmbedWrite(b *testing.B) {
	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)
	mu := eng.(metaengine.MapUpdater)
	ctx := context.Background()
	const N = 1000

	for i := 0; i < N; i++ {
		_ = mb.MapSet(ctx, "orders_embed", fmt.Sprintf("order-%d", i), makeCalibOrder(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("order-%d", i%N)
		_ = mu.MapUpdate(ctx, "orders_embed", key, func(prev any) any {
			order, ok := prev.(calibOrder)
			if !ok {
				return prev
			}
			order.Items = calibFixedChildren
			return order
		})
	}
}

// BenchmarkLayoutCalibration_NormalizeRead measures a multi-key read+merge
// under the normalize layout: parent lookup + child collection lookup.
func BenchmarkLayoutCalibration_NormalizeRead(b *testing.B) {
	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)
	mm := eng.(metaengine.MultimapBackend)
	ctx := context.Background()
	const N = 1000

	for i := 0; i < N; i++ {
		order := makeCalibOrder(i)
		header := calibOrderHeader{ID: order.ID, Total: order.Total, Status: order.Status}
		_ = mb.MapSet(ctx, "orders_norm", order.ID, header)
		for _, item := range order.Items {
			_ = mm.MultiAdd(ctx, "items_norm", order.ID, item)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("order-%d", i%N)
		_, _, _ = mb.MapGet(ctx, "orders_norm", key)
		_, _ = mm.MultiGet(ctx, "items_norm", key)
	}
}

// BenchmarkLayoutCalibration_NormalizeWrite measures a child mutation under the
// normalize layout: a single O(1) append to the child collection. MultiAdd is
// O(1) on every engine (memory: slice append; disk: one seq-keyed entry), so
// per-op cost is stable even though the collection gains one entry per op —
// the same shape as the Row/Columnar child-insert benches.
func BenchmarkLayoutCalibration_NormalizeWrite(b *testing.B) {
	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)
	mm := eng.(metaengine.MultimapBackend)
	ctx := context.Background()
	const N = 1000

	for i := 0; i < N; i++ {
		order := makeCalibOrder(i)
		header := calibOrderHeader{ID: order.ID, Total: order.Total, Status: order.Status}
		_ = mb.MapSet(ctx, "orders_norm", order.ID, header)
		for _, item := range order.Items {
			_ = mm.MultiAdd(ctx, "items_norm", order.ID, item)
		}
	}

	newItem := calibItem{SKU: "EXTRA-999", Qty: 1, Price: 5.00}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("order-%d", i%N)
		_ = mm.MultiAdd(ctx, "items_norm", key, newItem)
	}
}

// BenchmarkLayoutCalibration_StorageOverhead measures the relative byte size
// of embed vs normalize storage. Embed duplicates the parent header in each
// projection that references the aggregate; normalize stores it once.
// Prints calibration details to stderr.
func BenchmarkLayoutCalibration_StorageOverhead(b *testing.B) {
	order := makeCalibOrder(0)
	header := calibOrderHeader{ID: order.ID, Total: order.Total, Status: order.Status}

	embedBytes := mustJSONEncode(b, order)
	headerBytes := mustJSONEncode(b, header)

	itemBytesTotal := 0
	for _, item := range order.Items {
		itemBytesTotal += len(mustJSONEncode(b, item))
	}

	normalizeBytes := len(headerBytes) + itemBytesTotal
	embedSingle := len(embedBytes)

	// Simulate 3 projections referencing the same aggregate (realistic for
	// CQRS: order summary + order history + order search index).
	const numProjections = 3
	embedTotal := embedSingle * numProjections
	normalizeTotal := len(headerBytes)*numProjections + itemBytesTotal
	storageRatio := float64(embedTotal) / float64(normalizeTotal)

	for range b.N {
		_ = storageRatio
	}

	b.StopTimer()
	fmt.Fprintf(os.Stderr, "\n=== Layout Storage Calibration ===\n")
	fmt.Fprintf(os.Stderr, "Embed single proj:  %d bytes\n", embedSingle)
	fmt.Fprintf(os.Stderr, "Normalize single:   %d bytes\n", normalizeBytes)
	fmt.Fprintf(os.Stderr, "Embed x%d projs:     %d bytes\n", numProjections, embedTotal)
	fmt.Fprintf(os.Stderr, "Normalize x%d projs: %d bytes\n", numProjections, normalizeTotal)
	fmt.Fprintf(os.Stderr, "Storage ratio:      %.2fx\n\n", storageRatio)
}
