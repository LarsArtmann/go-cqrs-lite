// Package mysqltestcontainer provides shared MySQL/MariaDB testcontainer
// helpers for go-cqrs-lite integration tests — the pgtestcontainer pattern
// for the third queue dialect (and any other module that needs live
// MySQL/MariaDB).
//
// The helper resolves a SERVER-level DSN by priority:
//
//  1. MYSQL_TEST_DSN env var (CI service container, vm-mysql-nspawn.sh /
//     vm-mysql.sh legs — never boots a container when set)
//  2. a MariaDB testcontainer via local Docker (mariadb:11.4, the dialect
//     the engine suites are verified against)
//  3. skip (no DSN, no Docker, or -short)
//
// DSN returns the resolved DSN; each test provisions its own throwaway
// database on that server for isolation (see queue/mysql's freshDatabase
// for the canonical pattern — CREATE DATABASE with a random suffix,
// DROP DATABASE in t.Cleanup).
//
// Usage:
//
//	package mymodule
//
//	import "github.com/larsartmann/go-cqrs-lite/testutil/mysqltestcontainer/v4"
//
//	func TestMain(m *testing.M) { mysqltestcontainer.TestMain(m) }
//
//	func TestMyFeature(t *testing.T) {
//	    dsn := mysqltestcontainer.DSN(t)
//	    // dsn names a bootstrap database; create your own throwaway on it
//	}
package mysqltestcontainer

import (
	"context"
	"flag"
	"net"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

var serverDSN string //nolint:gochecknoglobals // test-framework cache

// TestMain resolves the server DSN for the calling package's test binary
// (env DSN > testcontainer > skip) and runs the tests. See the package
// comment for the priority model.
func TestMain(m *testing.M) {
	flag.Parse()

	if dsn := os.Getenv("MYSQL_TEST_DSN"); dsn != "" {
		serverDSN = dsn

		//art-dupl:accept TestMain scaffolding twin of pgtestcontainer; each dep-isolated container module repeats the DSN/short-circuit ladder on purpose
		finish(m, nil)

		return
	}

	if testing.Short() {
		finish(m, nil)

		return
	}

	ctx := context.Background()

	// Fast absence check (2026-09-25, queue M4 tail (b)): with no Docker
	// daemon, testcontainers' own probe burns ~60s per test binary before
	// erroring — the skip was correct but slow. When there is clearly no
	// socket and no DOCKER_HOST override, skip immediately.
	const (
		// dockerProbeTimeout bounds the socket dial: the socket file can
		// exist while the daemon is dead — the exact state that made the
		// probe burn 60s before erroring into the skip.
		dockerProbeTimeout = 500 * time.Millisecond
		// containerStartTimeout bounds the container attempt: a wedged
		// daemon/image pull burns ~60s per test binary otherwise. Covers a
		// warm local image start (mariadb:11.4 ~5-10s); past that, skip.
		containerStartTimeout = 25 * time.Second
	)

	if os.Getenv("DOCKER_HOST") == "" {
		probe := net.Dialer{Timeout: dockerProbeTimeout}
		conn, dialErr := probe.DialContext(ctx, "unix", "/var/run/docker.sock")
		if dialErr != nil {
			finish(m, nil)

			return
		}

		_ = conn.Close()
	}

	runCtx, cancelRun := context.WithTimeout(ctx, containerStartTimeout)
	defer cancelRun()

	ctr, err := mysql.Run(runCtx, "mariadb:11.4",
		mysql.WithDatabase("cqrs_test"),
		mysql.WithUsername("cqrs"),
		mysql.WithPassword("cqrs"),
	)
	if err != nil {
		// No Docker (or the pull failed): tests calling DSN skip, exactly
		// like the pre-testcontainer DSN-gated behavior.
		finish(m, nil)

		return
	}

	dsn, err := ctr.ConnectionString(ctx, "parseTime=true")
	if err != nil {
		_ = testcontainers.TerminateContainer(ctr)

		finish(m, nil)

		return
	}

	serverDSN = dsn

	finish(m, func() { _ = testcontainers.TerminateContainer(ctr) })
}

// finish runs the test binary, then the optional cleanup, and exits with
// the test exit code.
func finish(m *testing.M, cleanup func()) {
	code := m.Run()

	//art-dupl:accept test-infra twin — mysql/pg testcontainer finish helpers are dep-isolated sibling modules
	if cleanup != nil {
		cleanup()
	}

	os.Exit(code)
}

// DSN returns the resolved server-level DSN for the calling test,
// skipping when neither MYSQL_TEST_DSN nor a container is available. The
// DSN names a bootstrap database; point tests at their own throwaway
// database on the same server (the database component of the DSN is
// irrelevant to mysql.ParseDSN-based re-injection).
func DSN(tb testing.TB) string {
	tb.Helper()

	if serverDSN == "" {
		tb.Skip("mysql not available: set MYSQL_TEST_DSN or run with Docker")
	}

	return serverDSN
}
