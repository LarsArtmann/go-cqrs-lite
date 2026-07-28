package postgres_test

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// sharedDSN holds the Postgres connection string used by all tests in this
// package. It is populated by TestMain — either from POSTGRES_TEST_DSN (CI
// service container) or from a testcontainers-managed container (local dev).
var sharedDSN string

// TestMain starts a single Postgres container shared across all tests in the
// package to avoid per-test container startup latency (~2-3s each).
//
// Priority:
//  1. POSTGRES_TEST_DSN env var (CI service container, manual override)
//  2. testcontainers-managed postgres:16-alpine (local development)
//  3. Skip (Docker unavailable) — individual tests call postgresDSN which skips
//
// In -short mode the container is not started; tests skip individually.
func TestMain(m *testing.M) {
	flag.Parse()

	if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		sharedDSN = dsn
		os.Exit(m.Run())
	}

	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("cqrs_test"),
		postgres.WithUsername("cqrs"),
		postgres.WithPassword("cqrs"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		os.Exit(m.Run())
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(ctr)
		os.Exit(m.Run())
	}

	sharedDSN = dsn
	code := m.Run()

	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code)
}

// postgresDSN returns a live Postgres DSN, or skips the test if no container
// or env var is available.
func postgresDSN(t *testing.T) string {
	t.Helper()

	if sharedDSN == "" {
		t.Skip("postgres not available: set POSTGRES_TEST_DSN or run with Docker")
	}

	return sharedDSN
}
