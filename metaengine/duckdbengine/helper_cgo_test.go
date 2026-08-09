//go:build cgo

package duckdbengine_test

import ( //nolint:gci
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// mustNewDuckEngine creates an in-memory DuckDB engine, skipping the test when
// DuckDB/CGo is unavailable. The engine is closed automatically via t.Cleanup.
func mustNewDuckEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}
