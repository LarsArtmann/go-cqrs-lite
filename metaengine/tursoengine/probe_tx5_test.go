package tursoengine_test

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func seedN(ctx context.Context, tb testing.TB, eng metaengine.Engine, n int) {
	tb.Helper()

	tx := eng.(metaengine.Transactional)

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		mb := eng.(metaengine.MapBackend)

		for i := range n {
			value := map[string]any{"customer": "c", "amount": 1.0}
			if err := mb.MapSet(ctx, "orders", fmtKey(i), value); err != nil {
				return err
			}
		}

		return nil
	})

	tb.Helper()

	if err != nil {
		tb.Fatalf("seed %d: %v", n, err)
	}
}

func fmtKey(i int) string { return "k" + string(rune('a'+i%26)) + fmtInt(i) }

func fmtInt(i int) string { return strconv.Itoa(i) }

func TestProbeBisect(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	cases := []struct {
		name  string
		dsn   string
		specs []metaengine.MaterializedViewSpec
		n     int
	}{
		{"flag-only-10k", "accel1.db", nil, 10_000},
		{"one-scalar-5k", "accel2.db", []metaengine.MaterializedViewSpec{{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"}}, 5_000},
		{"one-scalar-10k", "accel3.db", []metaengine.MaterializedViewSpec{{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"}}, 10_000},
		{"one-grouped-10k", "accel4.db", []metaengine.MaterializedViewSpec{{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"}}, 10_000},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, err := tursoengine.New(filepath.Join(dir, c.dsn), tursoengine.WithMaterializedViews(c.specs))
			if err != nil {
				t.Skipf("turso unavailable: %v", err)
			}
			defer eng.Close()

			seedN(ctx, t, eng, c.n)
			t.Logf("%d rows OK", c.n)
		})
	}
}
