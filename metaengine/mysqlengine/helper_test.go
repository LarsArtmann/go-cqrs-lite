package mysqlengine_test

import (
	"context"
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/mysqlengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/onsi/gomega"
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

	eng, err := mysqlengine.New(dsn)
	if err != nil {
		tb.Skipf("MySQL not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
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

	eng, err := mysqlengine.New(dsn)
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

	eng, err := mysqlengine.New(dsn)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	defer func() { _ = eng.Close() }()

	hc, ok := eng.(metaengine.HealthChecker)
	g.Expect(ok).To(gomega.BeTrue())

	g.Expect(hc.HealthCheck(context.Background())).To(gomega.Succeed())
}
