package tursoengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

var specScalars = []metaengine.MaterializedViewSpec{
	{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"},
	{Collection: "orders", Fn: metaengine.MatViewCount},
	{Collection: "orders", Fn: metaengine.MatViewAvg, Column: "amount"},
	{Collection: "orders", Fn: metaengine.MatViewMin, Column: "amount"},
	{Collection: "orders", Fn: metaengine.MatViewMax, Column: "amount"},
}

var specGroupedSum = metaengine.MaterializedViewSpec{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"}
var specGroupedAvg = metaengine.MaterializedViewSpec{Collection: "orders", Fn: metaengine.MatViewAvg, Column: "amount", GroupBy: "customer"}

func TestProbeCount(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	cases := []struct {
		name  string
		specs []metaengine.MaterializedViewSpec
		n     int
	}{
		{"5scalars-10k", specScalars, 10_000},
		{"5scalars+gsum-10k", append(append([]metaengine.MaterializedViewSpec{}, specScalars...), specGroupedSum), 10_000},
		{"full7-5k", append(append(append([]metaengine.MaterializedViewSpec{}, specScalars...), specGroupedSum), specGroupedAvg), 5_000},
		{"full7-2k", append(append(append([]metaengine.MaterializedViewSpec{}, specScalars...), specGroupedSum), specGroupedAvg), 2_000},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, err := tursoengine.New(filepath.Join(dir, c.name+".db"), tursoengine.WithMaterializedViews(c.specs))
			if err != nil {
				t.Skipf("turso unavailable: %v", err)
			}
			defer eng.Close()

			seedN(ctx, t, eng, c.n)
			t.Logf("%d rows OK with %d views", c.n, len(c.specs))
		})
	}
}
