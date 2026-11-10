package pgengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestPostgres_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on the Postgres engine. Skips without a
// server (testcontainers or POSTGRES_TEST_DSN).
func TestPostgres_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	enginetest.RunCounterKeyCompositeTest(t, eng)
}
