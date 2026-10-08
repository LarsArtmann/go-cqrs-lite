package dgraphengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDgraph_KeysetPagination pins the compound-cursor keyset contract
// against the Dgraph engine (closure path via its KV adapter).
// Each Run call gets its own engine: the harness's store.Close closes the
// engine it wrapped, so sharing one instance across Runs double-closes.
func TestDgraph_KeysetPagination(t *testing.T) {
	t.Parallel()

	enginetest.RunKeysetPaginationTest(t, newDgraphEngineOrSkip(t))
	enginetest.RunKeysetExactEndTest(t, newDgraphEngineOrSkip(t))
}
