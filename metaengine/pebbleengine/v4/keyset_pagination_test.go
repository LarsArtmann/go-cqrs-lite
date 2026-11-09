package pebbleengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestPebble_KeysetPagination pins the compound-cursor keyset contract
// against the Pebble engine across its raw, sort-index, and closure paths.
// Each Run call gets its own engine: the harness's store.Close closes the
// engine it wrapped, so sharing one instance across Runs double-closes.
func TestPebble_KeysetPagination(t *testing.T) {
	t.Parallel()

	enginetest.RunKeysetPaginationTest(t, newPebbleEngineOrSkip(t))
	enginetest.RunKeysetExactEndTest(t, newPebbleEngineOrSkip(t))
}
