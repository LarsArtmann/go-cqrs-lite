package mysqlengine_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/mysqlengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// mysqlTestDSN returns the DSN for a test MySQL instance, or empty if not configured.
// Set MYSQL_TEST_DSN to run integration tests (e.g. "root:pass@tcp(localhost:3306)/test?parseTime=true").
func mysqlTestDSN() string { return os.Getenv("MYSQL_TEST_DSN") }

// mustNewMySQLEngine creates a MySQL engine for testing, skipping if no MySQL is available.
func mustNewMySQLEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	dsn := mysqlTestDSN()
	if dsn == "" {
		tb.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	eng, err := newEngineWithRetry(tb, dsn)
	if err != nil {
		if mysqlSkipClass(err) {
			tb.Skipf("MySQL not available: %v", err)
		}

		tb.Fatalf("mysql engine construction failed (not a skip-class error): %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

// newEngineWithRetry constructs the engine, retrying dead-connection
// errors: the QEMU VM's MySQL intermittently resets handshaking/pooled
// connections under runner load ("invalid connection" / "connection reset
// by peer" — 4 consecutive CI failures 2026-11-06 across two test
// surfaces), and mysqlengine.New's schema init rides one connection.
// Bounded so a genuinely broken server still fails loudly (OQ-10 policy:
// retry rides out transients, it never converts a persistent defect into
// a pass or a skip).
func newEngineWithRetry(tb testing.TB, dsn string) (metaengine.Engine, error) {
	tb.Helper()

	const attempts = 4

	var (
		eng metaengine.Engine
		err error
	)

	for range attempts {
		eng, err = mysqlengine.New(dsn)
		if err == nil || !deadConnectionClass(err) {
			return eng, err
		}

		time.Sleep(250 * time.Millisecond)
	}

	return eng, err
}

// deadConnectionClass matches the stale/reset connection class — the
// server answered once but the socket died (go-sql-driver "invalid
// connection", database/sql ErrBadConn, TCP reset). Distinct from
// mysqlSkipClass's never-reachable markers, which stay skips.
func deadConnectionClass(err error) bool {
	msg := strings.ToLower(err.Error())

	return errors.Is(err, driver.ErrBadConn) ||
		strings.Contains(msg, "invalid connection") ||
		strings.Contains(msg, "connection reset by peer")
}

func TestMySQLDriverRegistered(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	drivers := metaengine.RegisteredDrivers()
	g.Expect(drivers).To(gomega.ContainElement("mysql"))
}

func TestMySQLEngineProfile(t *testing.T) {
	t.Parallel()

	dsn := mysqlTestDSN()
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	g := gomega.NewWithT(t)

	eng, err := newEngineWithRetry(t, dsn)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	defer func() { _ = eng.Close() }()

	profile := eng.Profile()
	g.Expect(profile.Name).To(gomega.Equal("mysql"))
	g.Expect(profile.Supports).To(gomega.HaveKey(metaengine.ADTMap))
}

func TestMySQLHealthCheck(t *testing.T) {
	t.Parallel()

	dsn := mysqlTestDSN()
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	g := gomega.NewWithT(t)

	eng, err := newEngineWithRetry(t, dsn)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	defer func() { _ = eng.Close() }()

	hc, ok := eng.(metaengine.HealthChecker)
	g.Expect(ok).To(gomega.BeTrue())

	g.Expect(hc.HealthCheck(context.Background())).To(gomega.Succeed())
}
