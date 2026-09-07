package tursoengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbeViewCountStability(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name  string
		specs []metaengine.MaterializedViewSpec
	}{
		{"3views", matviewBenchSpecs()[:3]},
		{"5views", matviewBenchSpecs()[:5]},
		{"7views", matviewBenchSpecs()},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for round := range 3 {
				dir := t.TempDir()

				eng, err := tursoengine.New(filepath.Join(dir, "acc.db"), tursoengine.WithMaterializedViews(c.specs))
				if err != nil {
					t.Skipf("turso unavailable: %v", err)
				}

				seedInTx(ctx, t, eng, orderRows(10_000, 99))
				_ = eng.Close()
				t.Logf("%s round %d: OK", c.name, round)
			}
		})
	}
}
