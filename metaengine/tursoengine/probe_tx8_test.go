package tursoengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbeTwoEngines(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	full7 := append(append(append([]metaengine.MaterializedViewSpec{}, specScalars...), specGroupedSum), specGroupedAvg)

	// Both engines open simultaneously, both with specs, both seeding 10k.
	e1, err := tursoengine.New(filepath.Join(dir, "one.db"), tursoengine.WithMaterializedViews(full7))
	if err != nil {
		t.Skipf("turso unavailable: %v", err)
	}
	defer e1.Close()

	e2, err := tursoengine.New(filepath.Join(dir, "two.db"), tursoengine.WithMaterializedViews(full7))
	if err != nil {
		t.Fatal(err)
	}
	defer e2.Close()

	seedNCustomers(ctx, t, e1, 10_000, 99)
	t.Log("e1 seed ok")

	seedNCustomers(ctx, t, e2, 10_000, 99)
	t.Log("e2 seed ok")
}

func TestProbeBaselineThenAccel(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	full7 := append(append(append([]metaengine.MaterializedViewSpec{}, specScalars...), specGroupedSum), specGroupedAvg)

	// The bench pattern: plain engine (no specs) seeded first, accel second.
	base, err := tursoengine.New(filepath.Join(dir, "base.db"))
	if err != nil {
		t.Skipf("turso unavailable: %v", err)
	}
	defer base.Close()

	acc, err := tursoengine.New(filepath.Join(dir, "acc.db"), tursoengine.WithMaterializedViews(full7))
	if err != nil {
		t.Fatal(err)
	}
	defer acc.Close()

	seedNCustomers(ctx, t, base, 10_000, 99)
	t.Log("base seed ok")

	seedNCustomers(ctx, t, acc, 10_000, 99)
	t.Log("acc seed ok")
}
