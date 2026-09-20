package mysqlengine_test

import (
	"testing"

	adttest "github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestMySQLEngine_TxIsolationFromForeignContext pins the transaction
// visibility contract: a transaction is visible only to calls whose context
// descends from RunInTx's fn. A concurrent caller with its own context must
// never observe uncommitted writes, because transaction affinity must flow
// through the context — an engine-global "active tx" leaks the transaction
// to unrelated goroutines (dirty reads, and readers dying with
// "sql: Rows are closed" when the foreign tx commits mid-iteration).
// Live-gated on MYSQL_TEST_DSN (MariaDB/MySQL MVCC keeps uncommitted rows
// invisible to other sessions, which is what makes the foreign read a
// discriminator). Ported from sqliteengine 22ab7b218.
func TestMySQLEngine_TxIsolationFromForeignContext(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)

	adttest.AssertTxIsolationFromForeignContext(t, eng)
}
