package tursoengine_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func seedAccel(tb testing.TB, withBase bool, baseSeeded bool, amount float64) {
	ctx := context.Background()
	dir := tb.TempDir()

	full7 := matviewBenchSpecs()

	if withBase {
		base, err := tursoengine.New(filepath.Join(dir, "base.db"))
		if err != nil {
			tb.Skipf("turso unavailable: %v", err)
		}
		defer base.Close()

		if baseSeeded {
			tx := base.(metaengine.Transactional)

			_ = tx.RunInTx(ctx, func(ctx context.Context) error {
				mb := base.(metaengine.MapBackend)

				for i := range 10_000 {
					if err := mb.MapSet(ctx, "orders", fmt.Sprintf("k%d", i), map[string]any{"customer": "c", "amount": 1.0}); err != nil {
						return err
					}
				}

				return nil
			})
		}
	}

	acc, err := tursoengine.New(filepath.Join(dir, "acc.db"), tursoengine.WithMaterializedViews(full7))
	if err != nil {
		tb.Fatal(err)
	}
	defer acc.Close()

	tx := acc.(metaengine.Transactional)

	err = tx.RunInTx(ctx, func(ctx context.Context) error {
		mb := acc.(metaengine.MapBackend)

		for i := range 10_000 {
			if err := mb.MapSet(ctx, "orders", fmt.Sprintf("order-%04d", i), map[string]any{
				"customer": fmt.Sprintf("c%d", i%99),
				"amount":   amount,
			}); err != nil {
				return err
			}
		}

		return nil
	})

	tb.Logf("withBase=%v baseSeeded=%v amount=%v: err=%v", withBase, baseSeeded, amount, err)
}

func TestProbeMatrix2(t *testing.T) {
	seedAccel(t, true, false, 1.0)   // base open, unseeded, integer amounts
	seedAccel(t, true, false, 33.13) // base open, unseeded, decimal amounts
	seedAccel(t, true, true, 33.13)  // base open+seeded, decimal amounts
	seedAccel(t, false, false, 33.13)
}
