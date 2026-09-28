//go:build integration

package storage_test

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4"
)

// TestMain starts a shared Postgres server (external DSN via
// DATABASE_URL/POSTGRES_TEST_DSN, else a testcontainer) for all integration
// tests in this package. Each test gets its own fresh database via
// pgTestDSN — including under an explicit DSN, so packages sharing one CI
// service container cannot contaminate each other.
func TestMain(m *testing.M) {
	pgtestcontainer.AfterRun(func(tm *testing.M) { _, _ = snaps.Clean(tm) })
	pgtestcontainer.TestMain(m)
}

func pgTestDSN(t *testing.T) string {
	return pgtestcontainer.DSN(t)
}
