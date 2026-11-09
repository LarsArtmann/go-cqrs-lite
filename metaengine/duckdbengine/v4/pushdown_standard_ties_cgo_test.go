//go:build cgo

package duckdbengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDuckDB_PushdownStandardTies pins compound-cursor pagination on the
// standard (unplanned) pushdown path; the walk lives in the shared harness.
func TestDuckDB_PushdownStandardTies(t *testing.T) {
	t.Parallel()

	enginetest.RunPushdownStandardTiesTest(t, mustNewDuckEngine(t), "pd_ties_duck")
}
