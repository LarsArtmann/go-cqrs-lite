package tursoengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbeEngineBigTx(t *testing.T) {
	ctx := context.Background()

	eng, err := tursoengine.New("")
	if err != nil {
		t.Skipf("turso unavailable: %v", err)
	}
	defer eng.Close()

	tx := eng.(metaengine.Transactional)

	for _, n := range []int{1000, 5000, 10000} {
		err := tx.RunInTx(ctx, func(ctx context.Context) error {
			mb := eng.(metaengine.MapBackend)

			for i := range n {
				if err := mb.MapSet(ctx, "orders", fmt.Sprintf("k%d", i), map[string]any{"amount": 1.0}); err != nil {
					return fmt.Errorf("set %d: %w", i, err)
				}
			}

			return nil
		})
		t.Logf("n=%d err=%v", n, err)
	}
}
