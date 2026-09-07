package tursoengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbeReadPhaseLeak(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	base, err := tursoengine.New(filepath.Join(dir, "base.db"))
	if err != nil {
		t.Skipf("turso unavailable: %v", err)
	}

	seedInTx(ctx, t, base, orderRows(10_000, 99))
	t.Log("base seeded")

	ar := base.(metaengine.AggregateReader)

	// The baseline read phase: many full-scan aggregates.
	for i := 0; i < 150; i++ {
		if _, err := ar.Aggregate(ctx, "orders", metaengine.MatViewSum, "amount", nil); err != nil {
			t.Fatalf("scan %d: %v", i, err)
		}
	}

	t.Log("150 scans done")

	if err := base.Close(); err != nil {
		t.Fatalf("base close: %v", err)
	}

	acc, err := tursoengine.New(filepath.Join(dir, "acc.db"), tursoengine.WithMaterializedViews(matviewBenchSpecs()))
	if err != nil {
		t.Fatal(err)
	}
	defer acc.Close()

	seedInTx(ctx, t, acc, orderRows(10_000, 99))
	t.Log("acc seeded OK after read phase")
}
