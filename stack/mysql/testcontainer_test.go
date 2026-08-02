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
	"time"

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

	// Grant the cqrs user global CREATE/DROP DATABASE privilege (needed for
	// per-test database isolation in mysqlDSN and multidb tests). WithDefaultCredentials
	// (called automatically by tcmysql.Run) sets MYSQL_ROOT_PASSWORD to the same
	// value as MYSQL_PASSWORD, so the root password is "cqrs".
	//
	// We connect via Go's database/sql as root (go-sql-driver/mysql v1.10+
	// supports caching_sha2_password) with a retry loop, because MySQL may not
	// accept connections immediately after the wait-strategy log appears.
	rootDSN := replaceUserInMySQLDSN(containerDSN, "root")
	rootDB, err := waitForMySQLReady(rootDSN, 10*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: root connection failed: %v — tests will skip\n", err)
		containerDSN = ""
		_ = testcontainers.TerminateContainer(ctr)
		os.Exit(m.Run())
	}

	if _, err := rootDB.ExecContext(
		ctx,
		"GRANT ALL PRIVILEGES ON *.* TO 'cqrs'@'%' WITH GRANT OPTION",
	); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: GRANT failed: %v — tests will skip\n", err)
		_ = rootDB.Close()
		containerDSN = ""
		_ = testcontainers.TerminateContainer(ctr)
		os.Exit(m.Run())
	}

	_ = rootDB.Close()
	adminDB, _ = sql.Open("mysql", containerDSN)

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

// replaceUserInMySQLDSN swaps the username in a go-sql-driver/mysql DSN:
// user:pass@tcp(host:port)/db?params → newUser:pass@tcp(host:port)/db?params
func replaceUserInMySQLDSN(dsn, newUser string) string {
	atIdx := strings.Index(dsn, "@")
	if atIdx < 0 {
		return dsn
	}

	colonIdx := strings.Index(dsn, ":")
	if colonIdx < 0 || colonIdx >= atIdx {
		return dsn
	}

	return newUser + dsn[colonIdx:]
}

// waitForMySQLReady retries connecting to MySQL until it succeeds or timeout.
func waitForMySQLReady(dsn string, timeout time.Duration) (*sql.DB, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)

			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = db.PingContext(ctx)
		cancel()

		if err != nil {
			_ = db.Close()
			lastErr = err
			time.Sleep(500 * time.Millisecond)

			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("mysql not ready after %s: %w", timeout, lastErr)
}
