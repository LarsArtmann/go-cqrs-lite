package sqliteengine_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
)

func TestNewFromDSN_DefaultCacheSizePragma(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng, db, err := sqliteengine.NewFromDSN(":memory:")
	g.Expect(err).NotTo(HaveOccurred())
	defer db.Close()
	defer eng.Close()

	var cacheSize string
	g.Expect(db.QueryRowContext(context.Background(), "PRAGMA cache_size").Scan(&cacheSize)).
		To(Succeed())
	g.Expect(cacheSize).To(Equal("-65536"), "default family must set a 64 MiB page cache")
}

// TestNewFromDSN_PragmasHoldOnEveryPooledConnection pins issue #128's
// per-connection contract: the DSN constructors' production pragmas must hold
// on EVERY pooled connection, not only the first one opened at construction.
func TestNewFromDSN_PragmasHoldOnEveryPooledConnection(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// A file DSN (not :memory:): raised pools open real second connections,
	// and per-connection pragmas must reach them (issue #128 — under
	// WithMaxOpenConns, fresh connections previously carried busy_timeout=0
	// and failed with immediate SQLITE_BUSY under write contention).
	dsn := filepath.Join(t.TempDir(), "pool-pragmas.db")

	eng, db, err := sqliteengine.NewFromDSN(dsn)
	g.Expect(err).NotTo(HaveOccurred())
	defer db.Close()
	defer eng.Close()

	db.SetMaxOpenConns(2)

	ctx := context.Background()

	conn1, err := db.Conn(ctx)
	g.Expect(err).NotTo(HaveOccurred())
	defer conn1.Close()

	conn2, err := db.Conn(ctx) // distinct connection: conn1 is still held
	g.Expect(err).NotTo(HaveOccurred())
	defer conn2.Close()

	for name, conn := range map[string]*sql.Conn{"first": conn1, "second": conn2} {
		var busyTimeout int
		g.Expect(conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout)).
			To(Succeed(), name+" connection: read busy_timeout")
		g.Expect(busyTimeout).To(Equal(5000), name+" pooled connection must honor busy_timeout")

		var cacheSize string
		g.Expect(conn.QueryRowContext(ctx, "PRAGMA cache_size").Scan(&cacheSize)).
			To(Succeed(), name+" connection: read cache_size")
		g.Expect(cacheSize).To(Equal("-65536"), name+" pooled connection must keep the 64 MiB page cache")
	}

	var journalMode string
	g.Expect(conn2.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode)).
		To(Succeed())
	g.Expect(journalMode).To(Equal("wal"), "second pooled connection must see WAL mode")
}

// The caller-override path is structural: NewSQLiteEngineFromDSNWith applies
// caller pragmas AFTER the cache_size default (dsl.go builds the applied list
// as defaults+pragmas), and SQLite gives last-write-wins on repeated pragmas.
// This test pins that a caller cache_size does not break construction.
func TestNewSQLiteEngineFromDSN_CallerCacheSizeAccepted(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng, err := sqliteengine.NewSQLiteEngineFromDSN(":memory:", "cache_size=-1024")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()
}

// TestNewSQLiteEngineFromDSN_InvalidPragmaFailsConstruction pins the eager-ping
// half of the pragmaConnector contract (M37/F228): a pragma whose SQL is
// invalid must fail CONSTRUCTION, not the first query. Note SQLite silently
// ignores unknown pragma NAMES and bad VALUES (probed: busy_timeout=notanumber
// constructs fine), so the failing class is malformed pragma SQL — exactly
// what a typo'd pragma string produces. Both DSL constructors are pinned
// because the delegating convenience form could drift from the With form.
func TestNewSQLiteEngineFromDSN_InvalidPragmaFailsConstruction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng, err := sqliteengine.NewSQLiteEngineFromDSN(":memory:", "bad syntax here")
	g.Expect(err).To(HaveOccurred(), "invalid pragma SQL must fail construction eagerly")
	g.Expect(err.Error()).To(ContainSubstring(`pragma "bad syntax here"`), "error must name the offending pragma")
	g.Expect(eng).To(BeNil(), "no engine may escape a failed construction")
}

func TestNewSQLiteEngineFromDSNWith_InvalidPragmaFailsConstruction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng, err := sqliteengine.NewSQLiteEngineFromDSNWith(":memory:", []string{"foo(bar"})
	g.Expect(err).To(HaveOccurred(), "invalid pragma SQL must fail construction eagerly")
	g.Expect(err.Error()).To(ContainSubstring(`pragma "foo(bar"`), "error must name the offending pragma")
	g.Expect(eng).To(BeNil(), "no engine may escape a failed construction")
}
