//go:build cgo

package duckdbengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDuckDB_KeysetPagination pins the compound-cursor keyset contract
// against the DuckDB engine, including its pushdown scan surface.
func TestDuckDB_KeysetPagination(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)

	enginetest.RunKeysetPaginationTest(t, eng)
	enginetest.RunKeysetExactEndTest(t, eng)
}
