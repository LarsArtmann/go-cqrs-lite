package tursoengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbeNoViewsAfterReads(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	// Phase 1: plain engine, seed + reads (NO matviews anywhere).
	base, err := tursoengine.New(filepath.Join(dir, "base.db"))
	if err != nil {
		t.Skipf("turso unavailable: %v", err)
	}

	seedInTx(ctx, t, base, orderRows(10_000, 99))

	ar := base.(metaengine.AggregateReader)

	for i := 0; i < 150; i++ {
		if _, err := ar.Aggregate(ctx, "orders", metaengine.MatViewSum, "amount", nil); err != nil {
			t.Fatalf("scan %d: %v", i, err)
		}
	}

	_ = base.Close()

	// Phase 2: SECOND plain engine (no views) — big seed tx.
	plain, err := tursoengine.New(filepath.Join(dir, "plain.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Close()

	seedInTx(ctx, t, plain, orderRows(10_000, 99))
	t.Log("plain 10k seed OK after read phase")
}

func TestProbeSingleEngineViews(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	// Phase 1 + 2 in ONE engine: seed, 150 scans, then another 10k tx.
	acc, err := tursoengine.New(filepath.Join(dir, "acc.db"), tursoengine.WithMaterializedViews(matviewBenchSpecs()))
	if err != nil {
		t.Skipf("turso unavailable: %v", err)
	}
	defer acc.Close()

	seedInTx(ctx, t, acc, orderRows(10_000, 99))
	t.Log("acc 10k seed #1 ok")

	ar := acc.(metaengine.AggregateReader)

	for i := 0; i < 150; i++ {
		if _, err := ar.Aggregate(ctx, "orders", metaengine.MatViewSum, "amount", nil); err != nil {
			t.Fatalf("scan %d: %v", i, err)
		}
	}

	seedInTx(ctx, t, acc, orderRows(10_000, 99))
	t.Log("acc 10k seed #2 ok after scans (same engine)")
}
