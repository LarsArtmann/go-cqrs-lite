package pebbleengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestPebble_KeysetPagination pins the compound-cursor keyset contract
// against the Pebble engine (closure path; Pebble has no pushdown scan).
func TestPebble_KeysetPagination(t *testing.T) {
	t.Parallel()

	eng := newPebbleEngineOrSkip(t)

	enginetest.RunKeysetPaginationTest(t, eng)
	enginetest.RunKeysetExactEndTest(t, eng)
}
