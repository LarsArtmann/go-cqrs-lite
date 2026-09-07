package tursoengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbe100kSingleView(t *testing.T) {
	ctx := context.Background()

	specs := []metaengine.MaterializedViewSpec{
		{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"},
	}

	for round := range 3 {
		dir := t.TempDir()

		eng, err := tursoengine.New(filepath.Join(dir, "acc.db"), tursoengine.WithMaterializedViews(specs))
		if err != nil {
			t.Skipf("turso unavailable: %v", err)
		}

		seedInTx(ctx, t, eng, orderRows(100_000, 316))
		_ = eng.Close()
		t.Logf("round %d: 100k single grouped view OK", round)
	}
}
