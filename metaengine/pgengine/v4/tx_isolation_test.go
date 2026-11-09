package pgengine_test

import (
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	adttest "github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestPGEngine_TxIsolationFromForeignContext pins the transaction
// visibility contract: a transaction is visible only to calls whose context
// descends from RunInTx's fn. Postgres MVCC keeps uncommitted rows invisible
// to other sessions, which is what makes the foreign read a discriminator.
// Scenario body lives in adttest (ported from sqliteengine 22ab7b218).
func TestPGEngine_TxIsolationFromForeignContext(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	t.Cleanup(func() { metaengine.DeferClose(eng) })

	adttest.AssertTxIsolationFromForeignContext(t, eng)
}
