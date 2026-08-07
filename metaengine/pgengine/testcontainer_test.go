package pgengine_test

import (
	"testing"

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
