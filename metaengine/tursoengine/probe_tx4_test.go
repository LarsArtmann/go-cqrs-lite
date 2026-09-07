package tursoengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
)

func TestProbeBenchExact(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := tursoengine.New(filepath.Join(dir, "baseline.db"))
	if err != nil {
		t.Skipf("turso unavailable: %v", err)
	}
	defer eng.Close()

	seedInTx(ctx, t, eng, orderRows(10_000, 99))
	t.Log("10k seed ok")

	eng2, err := tursoengine.New(filepath.Join(dir, "accel.db"), tursoengine.WithMaterializedViews(matviewBenchSpecs()))
	if err != nil {
		t.Fatal(err)
	}
	defer eng2.Close()

	seedInTx(ctx, t, eng2, orderRows(10_000, 99))
	t.Log("10k seed with matviews ok")
}
