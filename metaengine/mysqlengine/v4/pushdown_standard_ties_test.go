package mysqlengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestMySQL_PushdownStandardTies pins compound-cursor pagination on the
// standard (unplanned) pushdown path; the walk lives in the shared harness.
func TestMySQL_PushdownStandardTies(t *testing.T) {
	t.Parallel()

	enginetest.RunPushdownStandardTiesTest(t, mustNewMySQLEngine(t), "pd_ties_mysql")
}
