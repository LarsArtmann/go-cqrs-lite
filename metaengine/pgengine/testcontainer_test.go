package pgengine_test

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

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver for CREATE DATABASE
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	containerDSN string
	adminDB      *sql.DB
	dbCounter    int64
	testDBCache  sync.Map // map[string]string — t.Name() → per-test DSN
)

// TestMain starts a single Postgres container shared across all tests.
// Priority: POSTGRES_TEST_DSN env var (CI) > testcontainers (local) > skip.
func TestMain(m *testing.M) {
	flag.Parse()

	if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		containerDSN = dsn
		os.Exit(m.Run())
	}

	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx := context.Background()

	ctr, err := postgres.Run(
		ctx, "postgres:16-alpine",
		postgres.WithDatabase("metaengine_test"),
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

// pgDSN returns a Postgres DSN for the calling test. When using testcontainers
// (local dev), each test gets its own fresh database within the shared container
// for isolation — critical because tests run in parallel (t.Parallel).
//
// When POSTGRES_TEST_DSN is set (CI service container), the DSN is returned
// directly without per-test isolation.
func pgDSN(t *testing.T) string {
	t.Helper()

	if containerDSN == "" {
		t.Skip("postgres not available: set POSTGRES_TEST_DSN or run with Docker")
	}

	if os.Getenv("POSTGRES_TEST_DSN") != "" {
		return containerDSN
	}

	name := t.Name()
	if dsn, ok := testDBCache.Load(name); ok {
		return dsn.(string)
	}

	dbName := fmt.Sprintf("test_%d", atomic.AddInt64(&dbCounter, 1))
	if _, err := adminDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
		t.Fatalf("create test database %s: %v", dbName, err)
	}

	dsn := replaceDBInDSN(containerDSN, dbName)
	testDBCache.Store(name, dsn)

	t.Cleanup(func() {
		testDBCache.Delete(name)
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
