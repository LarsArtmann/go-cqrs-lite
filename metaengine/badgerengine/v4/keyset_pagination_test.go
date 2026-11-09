package badgerengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBadger_KeysetPagination pins the compound-cursor keyset contract
// against the Badger engine (closure path; Badger has no pushdown scan).
// Each Run call gets its own engine: the harness's store.Close closes the
// engine it wrapped, so sharing one instance across Runs double-closes.
func TestBadger_KeysetPagination(t *testing.T) {
	t.Parallel()

	enginetest.RunKeysetPaginationTest(t, newBadgerEngineOrSkip(t))
	enginetest.RunKeysetExactEndTest(t, newBadgerEngineOrSkip(t))
}
