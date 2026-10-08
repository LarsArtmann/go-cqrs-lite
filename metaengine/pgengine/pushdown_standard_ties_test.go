package pgengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestPg_PushdownStandardTies pins compound-cursor pagination on the standard
// (unplanned) pushdown path; the walk lives in the shared harness.
func TestPg_PushdownStandardTies(t *testing.T) {
	t.Parallel()

	enginetest.RunPushdownStandardTiesTest(t, mustNewPgEngine(t), "pd_ties_pg")
}
