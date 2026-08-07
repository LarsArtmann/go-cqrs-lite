// Package pgtestcontainer provides shared Postgres testcontainer helpers for
// go-cqrs-lite integration tests.
//
// The helper starts a single Postgres container (or accepts an external DSN via
// DATABASE_URL / POSTGRES_TEST_DSN env var) and provisions a per-test database
// for isolation. Each test cleans up its database via t.Cleanup.
//
// Usage:
//
//	//go:build integration
//
//	package mymodule_test
//
//	import "github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4"
//
//	func TestMain(m *testing.M) { pgtestcontainer.TestMain(m) }
//
//	func TestMyFeature(t *testing.T) {
//	    dsn := pgtestcontainer.DSN(t)
//	    // use dsn in test
//	}
package pgtestcontainer

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
	containerDSN string
	adminDB      *sql.DB
	dbCounter    int64
	dbCache      sync.Map
)

// TestMain starts a shared Postgres container for all integration tests in
// the calling package. Each test gets its own fresh database for isolation.
//
// Priority: DATABASE_URL / POSTGRES_TEST_DSN env var (CI service container /
// ephemeral-pg.sh) > testcontainers (local) > skip.
func TestMain(m *testing.M) {
	flag.Parse()

	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		containerDSN = dsn
	} else if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		containerDSN = dsn
	}

	if containerDSN != "" {
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

	containerDSN = dsn
	adminDB, _ = sql.Open("pgx", dsn)

	code := m.Run()

	if adminDB != nil {
		_ = adminDB.Close()
	}

	_ = testcontainers.TerminateContainer(ctr)

	os.Exit(code)
}

// DSN returns a Postgres DSN for the calling test. When using testcontainers,
// each test gets its own fresh database for isolation.
func DSN(tb testing.TB) string {
	tb.Helper()

	if containerDSN == "" {
		tb.Skip("postgres not available: set DATABASE_URL/POSTGRES_TEST_DSN or run with Docker")
	}

	if os.Getenv("DATABASE_URL") != "" || os.Getenv("POSTGRES_TEST_DSN") != "" {
		return containerDSN
	}

	name := tb.Name()
	if dsn, ok := dbCache.Load(name); ok {
		return dsn.(string)
	}

	dbName := fmt.Sprintf("test_%d", atomic.AddInt64(&dbCounter, 1))
	if _, err := adminDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
		tb.Fatalf("create test database %s: %v", dbName, err)
	}

	dsn := replaceDBInDSN(containerDSN, dbName)
	dbCache.Store(name, dsn)

	tb.Cleanup(func() {
		dbCache.Delete(name)
		_, _ = adminDB.Exec(fmt.Sprintf(`DROP DATABASE "%s" WITH (FORCE)`, dbName))
	})

	return dsn
}

// replaceDBInDSN swaps the database name in a URL-format Postgres DSN:
// postgres://user:pass@host:port/olddb?params → .../newdb?params
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
