package benchkit

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
	benchPgDSN   string
	benchPgAdmin *sql.DB
	benchPgCount int64
	benchPgCache sync.Map
)

func TestMain(m *testing.M) {
	flag.Parse()

	if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		benchPgDSN = dsn
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

	benchPgDSN = dsn
	benchPgAdmin, _ = sql.Open("pgx", dsn)

	code := m.Run()

	if benchPgAdmin != nil {
		_ = benchPgAdmin.Close()
	}

	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code)
}

// benchPostgresDSN returns a Postgres DSN for the calling test, with per-test
// database isolation when using testcontainers.
func benchPostgresDSN(t *testing.T) string {
	t.Helper()

	if benchPgDSN == "" {
		t.Skip("postgres not available: set POSTGRES_TEST_DSN or run with Docker")
	}

	if os.Getenv("POSTGRES_TEST_DSN") != "" {
		return benchPgDSN
	}

	name := t.Name()
	if dsn, ok := benchPgCache.Load(name); ok {
		return dsn.(string)
	}

	dbName := fmt.Sprintf("bench_%d", atomic.AddInt64(&benchPgCount, 1))
	if _, err := benchPgAdmin.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
		t.Fatalf("create test database %s: %v", dbName, err)
	}

	dsn := benchReplaceDB(benchPgDSN, dbName)
	benchPgCache.Store(name, dsn)

	t.Cleanup(func() {
		benchPgCache.Delete(name)
		_, _ = benchPgAdmin.Exec(fmt.Sprintf(`DROP DATABASE "%s" WITH (FORCE)`, dbName))
	})

	return dsn
}

func benchReplaceDB(dsn, newDB string) string {
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
