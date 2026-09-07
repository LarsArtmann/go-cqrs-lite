package tursoengine_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func trySeed(tb testing.TB, label string, amount float64, keyFmt string) {
	ctx := context.Background()
	dir := tb.TempDir()

	acc, err := tursoengine.New(filepath.Join(dir, "acc.db"), tursoengine.WithMaterializedViews(matviewBenchSpecs()))
	if err != nil {
		tb.Skipf("turso unavailable: %v", err)
	}
	defer acc.Close()

	tx := acc.(metaengine.Transactional)

	err = tx.RunInTx(ctx, func(ctx context.Context) error {
		mb := acc.(metaengine.MapBackend)

		for i := range 10_000 {
			v := map[string]any{"customer": fmt.Sprintf("c%d", i%99), "amount": amount}
			if err := mb.MapSet(ctx, "orders", fmt.Sprintf(keyFmt, i), v); err != nil {
				return err
			}
		}

		return nil
	})

	tb.Logf("%s: err=%v", label, err)
}

func TestProbeDataMatrix(t *testing.T) {
	trySeed(t, "amount=1.0 keys=order-%04d", 1.0, "order-%04d")
	trySeed(t, "amount=33.13 keys=k%%d", 33.13, "k%d")
	trySeed(t, "amount=i*1.5 keys=k%%d", 1.5, "k%d")
}
