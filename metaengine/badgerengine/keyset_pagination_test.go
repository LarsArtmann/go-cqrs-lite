package badgerengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBadger_KeysetPagination pins the compound-cursor keyset contract
// against the Badger engine (closure path; Badger has no pushdown scan).
func TestBadger_KeysetPagination(t *testing.T) {
	t.Parallel()

	eng := newBadgerEngineOrSkip(t)

	enginetest.RunKeysetPaginationTest(t, eng)
	enginetest.RunKeysetExactEndTest(t, eng)
}
