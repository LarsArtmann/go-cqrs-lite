package mysqlengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestMySQL_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on the MySQL engine. Skips without
// MYSQL_TEST_DSN.
func TestMySQL_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)

	enginetest.RunCounterKeyCompositeTest(t, eng)
}
