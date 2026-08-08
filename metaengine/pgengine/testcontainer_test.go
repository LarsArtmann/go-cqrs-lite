package pgengine_test

import ( //nolint:gci
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	"github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4"
)

func TestMain(m *testing.M) { pgtestcontainer.TestMain(m) }

// pgDSN returns a Postgres DSN for the calling test. When using testcontainers
// (local dev), each test gets its own fresh database within the shared container
// for isolation — critical because tests run in parallel (t.Parallel).
//
// When POSTGRES_TEST_DSN is set (CI service container), the DSN is returned
// directly without per-test isolation.
func pgDSN(t testing.TB) string { return pgtestcontainer.DSN(t) }

// mustNewPgEngine creates a Postgres engine, skipping the test when PG is
// unavailable. The engine is closed automatically via t.Cleanup.
func mustNewPgEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}
