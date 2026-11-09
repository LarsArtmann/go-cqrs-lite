package pgengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestPg_KeysetPagination pins the compound-cursor keyset contract against the
// Postgres engine, including its pushdown scan surfaces. Each Run call gets
// its own engine: the harness's store.Close closes the engine it wrapped, so
// sharing one instance across Runs double-closes.
func TestPg_KeysetPagination(t *testing.T) {
	t.Parallel()

	enginetest.RunKeysetPaginationTest(t, mustNewPgEngine(t))
	enginetest.RunKeysetExactEndTest(t, mustNewPgEngine(t))
}
