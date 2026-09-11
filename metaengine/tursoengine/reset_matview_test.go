package tursoengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ResetEngine must leave materialized views EXACTLY empty: sqliteengine drops
// and recreates them instead of trusting IVM to propagate the DELETE wave
// (grouped views carry known upstream maintenance defects). After a reset a
// replay rebuilds from zero and the views track the replayed rows again.
func TestTursoMatView_ResetEngineRebuildsViews(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := mustEngineWithMatViews(
		t,
		"",
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
		},
		metaengine.MaterializedViewSpec{
			Collection: "orders",
			Fn:         metaengine.MatViewSum,
			Column:     "amount",
			GroupBy:    "customer",
		},
	)

	rows := orderRows(12, 3)
	seedOrders(ctx, t, eng, rows)

	expectScalar(t, eng, "orders", metaengine.MatViewSum, "amount", nil, sumAmounts(rows))

	resetter, ok := eng.(metaengine.EngineResetter)
	if !ok {
		t.Fatal("turso engine (via sqliteengine) must implement EngineResetter")
	}

	if err := resetter.ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	// Exactly empty after the reset — no stale rows, no IVM drift.
	expectScalar(t, eng, "orders", metaengine.MatViewSum, "amount", nil, 0)

	// A replay after the reset rebuilds through the recreated views.
	seedOrders(ctx, t, eng, rows[:6])
	expectScalar(t, eng, "orders", metaengine.MatViewSum, "amount", nil, sumAmounts(rows[:6]))
}
