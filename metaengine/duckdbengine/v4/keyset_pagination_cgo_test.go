//go:build cgo

package duckdbengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDuckDB_KeysetPagination pins the compound-cursor keyset contract
// against the DuckDB engine, including its pushdown scan surface.
// Each Run call gets its own engine: the harness's store.Close closes the
// engine it wrapped, so sharing one instance across Runs double-closes.
func TestDuckDB_KeysetPagination(t *testing.T) {
	t.Parallel()

	enginetest.RunKeysetPaginationTest(t, mustNewDuckEngine(t))
	enginetest.RunKeysetExactEndTest(t, mustNewDuckEngine(t))
}
