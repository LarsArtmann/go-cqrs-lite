package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBbolt_KeysetPagination pins the compound-cursor keyset contract
// against the bbolt engine (closure path; bbolt has no pushdown scan).
func TestBbolt_KeysetPagination(t *testing.T) {
	t.Parallel()

	eng := newBboltEngineOrSkip(t)

	enginetest.RunKeysetPaginationTest(t, eng)
	enginetest.RunKeysetExactEndTest(t, eng)
}
