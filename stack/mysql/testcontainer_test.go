package mysql_test

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

	_ "github.com/go-sql-driver/mysql" // register MySQL driver for CREATE DATABASE
	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
)

var (
	containerDSN string
	adminDB      *sql.DB
	dbCounter    int64
	testDBCache  sync.Map // map[string]string — t.Name() → per-test DSN
)

// TestMain starts a single MySQL container shared across all tests.
// Priority: MYSQL_TEST_DSN env var (CI) > testcontainers (local) > skip.
func TestMain(m *testing.M) {
	flag.Parse()

	if dsn := os.Getenv("MYSQL_TEST_DSN"); dsn != "" {
		containerDSN = dsn
		os.Exit(m.Run())
	}

	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx := context.Background()

	ctr, err := tcmysql.Run(
		ctx, "mysql:8.0",
		tcmysql.WithDatabase("cqrs_test"),
		tcmysql.WithUsername("cqrs"),
		tcmysql.WithPassword("cqrs"),
		testcontainers.WithEnv(map[string]string{"MYSQL_ROOT_PASSWORD": "rootpass"}),
	)
	if err != nil {
		os.Exit(m.Run())
	}

	dsn, err := ctr.ConnectionString(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(ctr)
		os.Exit(m.Run())
	}

	containerDSN = ensureParseTime(dsn)
	adminDB, _ = sql.Open("mysql", containerDSN)

	// Grant the cqrs user global CREATE/DROP DATABASE privilege (needed for
	// per-test database isolation in mysqlDSN and multidb tests).
	rootDSN := strings.Replace(containerDSN, "cqrs:cqrs@", "root:rootpass@", 1)
	rootDB, _ := sql.Open("mysql", replaceDBInMySQLDSN(rootDSN, "mysql"))
	if rootDB != nil {
		if _, err := rootDB.Exec("GRANT ALL PRIVILEGES ON *.* TO 'cqrs'@'%'"); err != nil {
			fmt.Fprintf(os.Stderr, "WARN: GRANT failed: %v\n", err)
		}

		_ = rootDB.Close()
	}

	code := m.Run()

	if adminDB != nil {
		_ = adminDB.Close()
	}

	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code)
}

// mysqlDSN returns a MySQL DSN for the calling test. When using testcontainers
// (local dev), each test gets its own fresh database within the shared
// container for isolation — critical because contracttest.RunSuite runs
// subtests in parallel, each creating a bundle that applies migrations.
//
// When MYSQL_TEST_DSN is set (CI service container), the DSN is returned
// directly without per-test isolation.
func mysqlDSN(t *testing.T) string {
	t.Helper()

	if containerDSN == "" {
		t.Skip("mysql not available: set MYSQL_TEST_DSN or run with Docker")
	}

	if os.Getenv("MYSQL_TEST_DSN") != "" {
		return containerDSN
	}

	name := t.Name()
	if dsn, ok := testDBCache.Load(name); ok {
		return dsn.(string)
	}

	dbName := fmt.Sprintf("test_%d", atomic.AddInt64(&dbCounter, 1))
	if _, err := adminDB.Exec("CREATE DATABASE `" + dbName + "`"); err != nil {
		t.Fatalf("create test database %s: %v", dbName, err)
	}

	dsn := replaceDBInMySQLDSN(containerDSN, dbName)
	testDBCache.Store(name, dsn)

	t.Cleanup(func() {
		testDBCache.Delete(name)
		_, _ = adminDB.Exec("DROP DATABASE IF EXISTS `" + dbName + "`")
	})

	return dsn
}

// replaceDBInMySQLDSN swaps the database name in a go-sql-driver/mysql DSN:
// user:pass@tcp(host:port)/olddb?params → user:pass@tcp(host:port)/newdb?params
func replaceDBInMySQLDSN(dsn, newDB string) string {
	slashIdx := strings.Index(dsn, "/")
	if slashIdx < 0 {
		return dsn
	}

	afterSlash := dsn[slashIdx+1:]

	queryIdx := strings.Index(afterSlash, "?")
	query := ""

	if queryIdx >= 0 {
		query = afterSlash[queryIdx:]
	}

	return dsn[:slashIdx+1] + newDB + query
}

// ensureParseTime ensures the DSN has parseTime=true, which is required for
// MySQL to return time.Time values from DATETIME columns.
func ensureParseTime(dsn string) string {
	if strings.Contains(dsn, "parseTime=") {
		return dsn
	}

	if strings.Contains(dsn, "?") {
		return dsn + "&parseTime=true"
	}

	return dsn + "?parseTime=true"
}
