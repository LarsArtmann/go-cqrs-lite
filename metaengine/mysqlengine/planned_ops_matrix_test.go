package mysqlengine_test

import (
	"testing"

	mysqlengine "github.com/larsartmann/go-cqrs-lite/metaengine/mysqlengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestMySQLPlannedOpsMatrix runs the D3 planned-ops parity matrix on live
// MariaDB/MySQL. Skips unless MYSQL_TEST_DSN is set.
func TestMySQLPlannedOpsMatrix(t *testing.T) {
	t.Parallel()

	if mysqlTestDSN() == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	adttest.RunPlannedOpsMatrix(t, []adttest.Factory{
		{
			Name: "mysql",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng, err := mysqlengine.New(mysqlTestDSN())
				if err != nil {
					t.Skipf("MySQL not available: %v", err)
				}

				return eng
			},
			PreClean: func(t *testing.T, collection string) {
				t.Helper()
				cleanupPlannedCollection(t, mysqlTestDSN(), collection)
				t.Cleanup(func() { cleanupPlannedCollection(t, mysqlTestDSN(), collection) })
			},
		},
	})
}
