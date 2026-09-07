package tursoengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbeExactStress(t *testing.T) {
	ctx := context.Background()

	for round := range 5 {
		dir := t.TempDir()

		base, err := tursoengine.New(filepath.Join(dir, "baseline.db"))
		if err != nil {
			t.Skipf("turso unavailable: %v", err)
		}

		acc, err := tursoengine.New(filepath.Join(dir, "accel.db"), tursoengine.WithMaterializedViews(matviewBenchSpecs()))
		if err != nil {
			t.Fatal(err)
		}

		err = func() error {
			defer func() { _ = base.Close(); _ = acc.Close() }()

			tx := base.(metaengine.Transactional)

			if err := tx.RunInTx(ctx, func(ctx context.Context) error {
				return nil
			}); err != nil {
				t.Logf("round %d: warmup tx failed: %v", round, err)
			}

			txAcc := acc.(metaengine.Transactional)

			if err := txAcc.RunInTx(ctx, func(ctx context.Context) error {
				mb := acc.(metaengine.MapBackend)

				for i, row := range orderRows(10_000, 99) {
					_ = i
					if err := mb.MapSet(ctx, "orders", row.Key, map[string]any{"customer": row.Customer, "amount": row.Amount}); err != nil {
						return err
					}
				}

				return nil
			}); err != nil {
				t.Logf("round %d: accel seed FAILED: %v", round, err)

				return err
			}

			t.Logf("round %d: OK", round)

			return nil
		}()

		if err != nil {
			continue
		}
	}
}
