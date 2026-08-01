package pebbleengine_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// BenchmarkPebbleGetRawValue vs BenchmarkPebbleMapGet shows the JSON tax
// reduction: GetRawValue returns raw bytes (1 JSON op when decoded to V),
// MapGet decodes to any first (2 JSON ops: decode + reify).

type benchUser struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	City string `json:"city"`
}

func setupBenchEngine(b *testing.B) (metaengine.Engine, metaengine.RawValueReader) {
	b.Helper()

	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { eng.Close() })

	mb := eng.(metaengine.MapBackend)
	ctx := context.Background()
	for i := range 100 {
		if err := mb.MapSet(ctx, "bench", i, benchUser{
			Name: "Alice",
			Age:  i,
			City: "NYC",
		}); err != nil {
			b.Fatal(err)
		}
	}

	return eng, eng.(metaengine.RawValueReader)
}

func BenchmarkPebbleMapGet(b *testing.B) {
	eng, _ := setupBenchEngine(b)
	mb := eng.(metaengine.MapBackend)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		for i := range 100 {
			val, found, err := mb.MapGet(ctx, "bench", i)
			if err != nil || !found {
				b.Fatal(err)
			}
			_ = val
		}
	}
}

func BenchmarkPebbleGetRawValue(b *testing.B) {
	_, rvr := setupBenchEngine(b)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		for i := range 100 {
			raw, found, err := rvr.GetRawValue(ctx, "bench", i)
			if err != nil || !found {
				b.Fatal(err)
			}
			_ = raw
		}
	}
}

func BenchmarkPebbleGetRawValueTyped(b *testing.B) {
	_, rvr := setupBenchEngine(b)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		for i := range 100 {
			raw, found, err := rvr.GetRawValue(ctx, "bench", i)
			if err != nil || !found {
				b.Fatal(err)
			}

			var u benchUser
			if err := json.Unmarshal(raw, &u); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPebbleMapGetTyped(b *testing.B) {
	eng, _ := setupBenchEngine(b)
	mb := eng.(metaengine.MapBackend)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		for i := range 100 {
			val, found, err := mb.MapGet(ctx, "bench", i)
			if err != nil || !found {
				b.Fatal(err)
			}

			data, err := json.Marshal(val)
			if err != nil {
				b.Fatal(err)
			}

			var u benchUser
			if err := json.Unmarshal(data, &u); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPebbleScanRawValues(b *testing.B) {
	eng, _ := setupBenchEngine(b)
	rsr := eng.(metaengine.RawScanReader)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		raw, err := rsr.ScanRawValues(ctx, "bench", nil, nil, nil, 0)
		if err != nil {
			b.Fatal(err)
		}
		for _, r := range raw.Items {
			var u benchUser
			if err := json.Unmarshal(r, &u); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPebbleScanRawValuesFiltered(b *testing.B) {
	eng, _ := setupBenchEngine(b)
	rsr := eng.(metaengine.RawScanReader)
	ctx := context.Background()

	filters := []metaengine.FilterSpec{
		{Column: "age", Op: metaengine.FilterGe, Value: float64(50)},
	}
	sortSpec := &metaengine.SortSpec{Column: "age", Desc: true}

	b.ResetTimer()
	for b.Loop() {
		raw, err := rsr.ScanRawValues(ctx, "bench", filters, sortSpec, nil, 10)
		if err != nil {
			b.Fatal(err)
		}
		for _, r := range raw.Items {
			var u benchUser
			if err := json.Unmarshal(r, &u); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPebbleMapScan(b *testing.B) {
	eng, _ := setupBenchEngine(b)
	sb := eng.(metaengine.ScanBackend)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		rows, err := sb.MapScan(ctx, "bench", nil, nil, nil, 0)
		if err != nil {
			b.Fatal(err)
		}
		for _, row := range rows {
			data, err := json.Marshal(row)
			if err != nil {
				b.Fatal(err)
			}

			var u benchUser
			if err := json.Unmarshal(data, &u); err != nil {
				b.Fatal(err)
			}
		}
	}
}
