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
	containerDSN string           //nolint:gochecknoglobals // test-framework cache
	adminDB      *sql.DB          //nolint:gochecknoglobals // test-framework cache
	dbCounter    atomic.Int64     //nolint:gochecknoglobals // test-framework cache
	dbCache      sync.Map         //nolint:gochecknoglobals // test-framework cache
	afterRunFn   func(*testing.M) //nolint:gochecknoglobals // test-framework hook
)

// AfterRun registers fn to run after m.Run() on every TestMain exit path,
// before process exit. At most one callback is kept (last registration wins).
// Use it for post-run work such as snaps.Clean(m), which otherwise cannot
// hook into TestMain because TestMain calls os.Exit itself.
func AfterRun(fn func(*testing.M)) {
	afterRunFn = fn
}

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
		// Even with an external DSN (CI service container), open an admin
		// connection so DSN() can provision per-test databases for isolation.
		// Without this, all tests share one database and cross-test interference
		// becomes a problem (especially under -race).
		adminDB, _ = sql.Open("pgx", containerDSN)

		finish(m, nil)

		return
	}

	if testing.Short() {
		finish(m, nil)

		return
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
		finish(m, nil)

		return
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(ctr)

		finish(m, nil)

		return
	}

	containerDSN = dsn
	adminDB, _ = sql.Open("pgx", dsn)

	finish(m, func() {
		if adminDB != nil {
			_ = adminDB.Close()
		}

		_ = testcontainers.TerminateContainer(ctr)
	})
}

// finish runs the test binary, invokes the registered AfterRun callback (if
// any), runs the optional cleanup, and exits with the test exit code.
func finish(m *testing.M, cleanup func()) {
	code := m.Run()

	if afterRunFn != nil {
		afterRunFn(m)
	}

	if cleanup != nil {
		cleanup()
	}

	os.Exit(code)
}

// DSN returns a Postgres DSN for the calling test. When using testcontainers,
// each test gets its own fresh database for isolation.
func DSN(tb testing.TB) string {
	tb.Helper()

	if containerDSN == "" {
		tb.Skip("postgres not available: set DATABASE_URL/POSTGRES_TEST_DSN or run with Docker")
	}

	// When no admin connection is available (testcontainer failed to start),
	// fall back to the shared DSN without per-test isolation.
	if adminDB == nil {
		return containerDSN
	}

	name := tb.Name()
	if dsn, ok := dbCache.Load(name); ok {
		dsnStr, ok := dsn.(string)
		if !ok {
			tb.Fatalf("cached DSN has wrong type: %T", dsn)
		}

		return dsnStr
	}

	dbName := fmt.Sprintf("test_%d_%d", os.Getpid(), dbCounter.Add(1))
	if _, err := adminDB.ExecContext(context.Background(),
		fmt.Sprintf(`CREATE DATABASE "%s"`, dbName),
	); err != nil {
		tb.Fatalf("create test database %s: %v", dbName, err)
	}

	dsn := replaceDBInDSN(containerDSN, dbName)
	dbCache.Store(name, dsn)

	tb.Cleanup(func() {
		dbCache.Delete(name)

		_, _ = adminDB.ExecContext(context.Background(),
			fmt.Sprintf(`DROP DATABASE "%s" WITH (FORCE)`, dbName))
	})

	return dsn
}

// replaceDBInDSN swaps the database name in a Postgres DSN. Supports both
// URL format (postgres://user:pass@host:port/db?params) and keyword/value
// format (host=localhost port=5432 dbname=mydb sslmode=disable). If the DSN
// has no parseable database name, the original is returned unchanged.
func replaceDBInDSN(dsn, newDB string) string {
	// URL format: scheme://user:pass@host:port/db?params
	if strings.Contains(dsn, "://") {
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

	// Keyword/value format: key=value pairs separated by spaces.
	const key = "dbname="

	idx := strings.Index(dsn, key)
	if idx < 0 {
		// No dbname keyword — append one so the per-test DB takes effect.
		sep := " "
		if len(dsn) == 0 || dsn[len(dsn)-1] == ' ' {
			sep = ""
		}

		return dsn + sep + key + newDB
	}

	start := idx + len(key)
	rest := dsn[start:]

	// Value ends at the next space or end of string.
	end := strings.IndexByte(rest, ' ')
	if end < 0 {
		return dsn[:start] + newDB
	}

	return dsn[:start] + newDB + rest[end:]
}
