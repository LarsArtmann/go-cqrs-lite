//go:build integration

package storage_test

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	pgContainerDSN string
	pgAdminDB      *sql.DB
	pgDBCounter    int64
	pgDBCache      sync.Map
)

// TestMain starts a shared Postgres container for all integration tests in
// this package. Each test gets its own fresh database for isolation.
func TestMain(m *testing.M) {
	flag.Parse()

	// Prefer external DSN (CI service container).
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		pgContainerDSN = dsn
	} else if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		pgContainerDSN = dsn
	}

	if pgContainerDSN != "" {
		os.Exit(m.Run())
	}

	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx := context.Background()

	ctr, err := postgres.Run(
		ctx, "postgres:16-alpine",
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

	pgContainerDSN = dsn
	pgAdminDB, _ = sql.Open("pgx", dsn)

	code := m.Run()

	if pgAdminDB != nil {
		_ = pgAdminDB.Close()
	}

	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code)
}

// pgTestDSN returns a Postgres DSN for the calling test. When using
// testcontainers, each test gets its own fresh database for isolation.
func pgTestDSN(t *testing.T) string {
	t.Helper()

	if pgContainerDSN == "" {
		t.Skip("postgres not available: set DATABASE_URL/POSTGRES_TEST_DSN or run with Docker")
	}

	if os.Getenv("DATABASE_URL") != "" || os.Getenv("POSTGRES_TEST_DSN") != "" {
		return pgContainerDSN
	}

	name := t.Name()
	if dsn, ok := pgDBCache.Load(name); ok {
		return dsn.(string)
	}

	dbName := fmt.Sprintf("test_%d", atomic.AddInt64(&pgDBCounter, 1))
	if _, err := pgAdminDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
		t.Fatalf("create test database %s: %v", dbName, err)
	}

	dsn := replaceDBInDSN(pgContainerDSN, dbName)
	pgDBCache.Store(name, dsn)

	t.Cleanup(func() {
		pgDBCache.Delete(name)
		_, _ = pgAdminDB.Exec(fmt.Sprintf(`DROP DATABASE "%s" WITH (FORCE)`, dbName))
	})

	return dsn
}

func replaceDBInDSN(dsn, newDB string) string {
	queryStart := strings.Index(dsn, "?")

	pathPart := dsn
	query := ""

	if queryStart >= 0 {
		pathPart = dsn[:queryStart]
		query = dsn[queryStart:]
	}

	lastSlash := strings.LastIndex(pathPart, "/")
	if lastSlash < 0 {
		return dsn
	}

	return pathPart[:lastSlash+1] + newDB + query
}
