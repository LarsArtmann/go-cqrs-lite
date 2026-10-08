package mysqlengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestMySQL_KeysetPagination pins the compound-cursor keyset contract against
// the MySQL/MariaDB engine, including its pushdown scan surfaces. Each Run
// call gets its own engine: the harness's store.Close closes the engine it
// wrapped, so sharing one instance across Runs double-closes.
func TestMySQL_KeysetPagination(t *testing.T) {
	t.Parallel()

	enginetest.RunKeysetPaginationTest(t, mustNewMySQLEngine(t))
	enginetest.RunKeysetExactEndTest(t, mustNewMySQLEngine(t))
}
