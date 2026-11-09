package postgres_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4"
)

func TestMain(m *testing.M) { pgtestcontainer.TestMain(m) }

// postgresDSN returns a Postgres DSN for the calling test. When using
// testcontainers (local dev), each test gets its own fresh database within the
// shared container for isolation — critical because contracttest.RunSuite runs
// subtests in parallel, each creating a bundle that applies migrations.
//
// When POSTGRES_TEST_DSN is set (CI service container), the DSN is returned
// directly without per-test isolation.
func postgresDSN(t *testing.T) string { return pgtestcontainer.DSN(t) }
