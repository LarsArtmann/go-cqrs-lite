package tursoengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbeChunked100k(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	acc, err := tursoengine.New(filepath.Join(dir, "acc.db"), tursoengine.WithMaterializedViews(matviewBenchSpecs()))
	if err != nil {
		t.Skipf("turso unavailable: %v", err)
	}
	defer acc.Close()

	tx := acc.(metaengine.Transactional)
	mb := acc.(metaengine.MapBackend)

	rows := orderRows(100_000, 316)

	for start := 0; start < len(rows); start += 1000 {
		end := min(start+1000, len(rows))
		chunk := rows[start:end]

		if err := tx.RunInTx(ctx, func(ctx context.Context) error {
			for _, row := range chunk {
				if err := mb.MapSet(ctx, "orders", row.Key, map[string]any{"customer": row.Customer, "amount": row.Amount}); err != nil {
					return err
				}
			}

			return nil
		}); err != nil {
			t.Fatalf("chunk at %d: %v", start, err)
		}
	}

	ar := acc.(metaengine.AggregateReader)

	sum, err := ar.Aggregate(ctx, "orders", metaengine.MatViewSum, "amount", nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("100k rows seeded in 1k chunks: SUM=%v want=%v", sum, sumAmounts(rows))
}
