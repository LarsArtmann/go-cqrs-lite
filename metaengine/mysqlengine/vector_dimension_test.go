package mysqlengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

func TestVectorDimensionGuard(t *testing.T) {
	t.Parallel()

	if mysqlTestDSN() == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	eng := mustNewMySQLEngine(t)

	adttest.AssertVectorDimensionGuard(t, eng)
}
