package metaengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestMemory_TransactionalBaseline runs the baseline transactional contract
// against the Memory engine. The Memory engine's RunInTx is a pass-through
// (no real transaction semantics): it verifies commit path and error
// propagation work, and explicitly documents that rollback is NOT supported.
//
// This closes the baseline-parity gap: SQLite, DuckDB, and PG all call
// enginetest.RunTransactionalTest. Memory calls RunTransactionalBaselineTest
// because it deliberately does not implement rollback.
func TestMemory_TransactionalBaseline(t *testing.T) {
	t.Parallel()

	enginetest.RunTransactionalBaselineTest(t, metaengine.NewMemoryEngine())
}
