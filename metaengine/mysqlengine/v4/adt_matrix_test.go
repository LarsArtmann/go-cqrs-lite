package mysqlengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

func TestMySQLADTMatrix(t *testing.T) {
	t.Parallel()

	dsn := mysqlTestDSN()
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name:   "mysql",
			Create: func(t *testing.T) metaengine.Engine { return mustNewMySQLEngine(t) },
		},
	})
}

// TestCapabilityConformance verifies this engine's Profile() declarations
// against its implemented backend interfaces (declared-vs-implemented table).
func TestCapabilityConformance(t *testing.T) {
	t.Parallel()

	adttest.RunCapabilityConformance(t, "mysql", mustNewMySQLEngine(t), nil)
}
