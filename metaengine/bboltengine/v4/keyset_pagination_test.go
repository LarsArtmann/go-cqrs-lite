package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBbolt_KeysetPagination pins the compound-cursor keyset contract
// against the bbolt engine (closure path; bbolt has no pushdown scan).
// Each Run call gets its own engine: the harness's store.Close closes the
// engine it wrapped, so sharing one instance across Runs double-closes.
func TestBbolt_KeysetPagination(t *testing.T) {
	t.Parallel()

	enginetest.RunKeysetPaginationTest(t, newBboltEngineOrSkip(t))
	enginetest.RunKeysetExactEndTest(t, newBboltEngineOrSkip(t))
}
