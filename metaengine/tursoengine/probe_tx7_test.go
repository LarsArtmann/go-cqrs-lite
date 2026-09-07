package tursoengine_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func seedNCustomers(ctx context.Context, tb testing.TB, eng metaengine.Engine, n, customers int) {
	tb.Helper()

	tx := eng.(metaengine.Transactional)

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		mb := eng.(metaengine.MapBackend)

		for i := range n {
			value := map[string]any{"customer": fmt.Sprintf("c%d", i%customers), "amount": 1.0}
			if err := mb.MapSet(ctx, "orders", "k"+strconv.Itoa(i), value); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		tb.Fatalf("seed n=%d cust=%d: %v", n, customers, err)
	}
}

func TestProbeGroups(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	full7 := append(append(append([]metaengine.MaterializedViewSpec{}, specScalars...), specGroupedSum), specGroupedAvg)

	cases := []struct {
		name      string
		n, custom int
	}{
		{"10k-99cust", 10_000, 99},
		{"10k-2cust", 10_000, 2},
		{"4k-99cust", 4_000, 99},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, err := tursoengine.New(filepath.Join(dir, c.name+".db"), tursoengine.WithMaterializedViews(full7))
			if err != nil {
				t.Skipf("turso unavailable: %v", err)
			}
			defer eng.Close()

			seedNCustomers(ctx, t, eng, c.n, c.custom)
			t.Logf("OK %s", c.name)
		})
	}
}
