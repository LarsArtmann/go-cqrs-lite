package sqliteengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestRunInTx_SQLite_ForeignEngineTxNotAdopted pins that an ambient
// transaction belongs to the engine that opened it: a second sqlite engine
// called with a ctx descending from the first engine's RunInTx must write
// through its own pool, not through the foreign transaction.
func TestRunInTx_SQLite_ForeignEngineTxNotAdopted(t *testing.T) {
	t.Parallel()

	for _, planned := range []bool{false, true} {
		owner, other := newTxTestEngine(t), newTxTestEngine(t)
		ctx := context.Background()

		if planned {
			for _, eng := range []metaengine.Engine{owner, other} {
				if err := eng.(interface {
					ApplyLayout(string, []string, []string) error
				}).ApplyLayout("c", []string{"status"}, []string{"n"}); err != nil {
					t.Fatalf("ApplyLayout: %v", err)
				}
			}
		}

		err := owner.(metaengine.Transactional).RunInTx(ctx, func(tctx context.Context) error {
			return other.(metaengine.MapBackend).MapSet(tctx, "c", "k", map[string]any{"n": 1})
		})
		if err != nil {
			t.Fatalf("planned=%v RunInTx: %v", planned, err)
		}

		_, ok, err := other.(metaengine.MapBackend).MapGet(ctx, "c", "k")
		if err != nil || !ok {
			t.Fatalf("planned=%v the second engine must hold the row its MapSet reported as written (ok=%v err=%v)",
				planned, ok, err)
		}

	}
}
