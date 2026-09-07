package tursoengine_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbeCreationOrder(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	// ORDER MATTERS TEST: create BOTH engines (accel DDL runs at construction)
	// BEFORE seeding either — the exact bench order.
	base, err := tursoengine.New(filepath.Join(dir, "base.db"))
	if err != nil {
		t.Skipf("turso unavailable: %v", err)
	}
	defer base.Close()

	acc, err := tursoengine.New(filepath.Join(dir, "acc.db"), tursoengine.WithMaterializedViews(matviewBenchSpecs()))
	if err != nil {
		t.Fatal(err)
	}
	defer acc.Close()

	seedBase := func() {
		tx := base.(metaengine.Transactional)

		if err := tx.RunInTx(ctx, func(ctx context.Context) error {
			mb := base.(metaengine.MapBackend)

			for i := range 10_000 {
				if err := mb.MapSet(ctx, "orders", fmt.Sprintf("k%d", i), map[string]any{"customer": "c", "amount": 1.0}); err != nil {
					return err
				}
			}

			return nil
		}); err != nil {
			t.Fatalf("base seed: %v", err)
		}
	}

	seedAcc := func(amount float64) {
		tx := acc.(metaengine.Transactional)

		if err := tx.RunInTx(ctx, func(ctx context.Context) error {
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
		}); err != nil {
			t.Fatalf("acc seed: %v", err)
		}
	}

	seedBase()
	t.Log("base seed ok (after DDL of acc exists)")

	seedAcc(1.0)
	t.Log("acc seed 1.0 ok")

	seedAcc(33.13)
	t.Log("acc seed 33.13 ok")
}
