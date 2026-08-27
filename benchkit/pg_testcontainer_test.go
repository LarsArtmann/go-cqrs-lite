package benchkit

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4"
)

// TestMain starts a shared Postgres server (external DSN via
// POSTGRES_TEST_DSN, else a testcontainer) for all integration tests in
// this package. Each test gets its own fresh database — including under an
// explicit DSN — and per-process database names avoid collisions when
// several test binaries share one CI service container.
func TestMain(m *testing.M) { pgtestcontainer.TestMain(m) }

func benchPostgresDSN(t *testing.T) string {
	return pgtestcontainer.DSN(t)
}
