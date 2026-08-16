package sqliteengine_test

import (
	"context"
	"database/sql"
	"testing"

	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestSQLiteCloseOwnership pins the DB-ownership contract: engines created
// via NewSQLiteEngineFromDSN (driver-factory path) OWN their *sql.DB — Close
// closes it — while engines wrapping a caller-supplied pool leave the
// database open so the caller can keep using it.
func TestSQLiteCloseOwnership(t *testing.T) {
	t.Parallel()

	t.Run("owning engine closes its database", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		eng, err := sqliteengine.NewSQLiteEngineFromDSN(":memory:")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(eng.Close()).To(Succeed())

		hc, ok := eng.(metaengine.HealthChecker)
		g.Expect(ok).To(BeTrue())
		g.Expect(hc.HealthCheck(context.Background())).NotTo(Succeed())
	})

	t.Run("borrowed engine leaves database open", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		eng, db, err := sqliteengine.NewFromDSN(":memory:")
		g.Expect(err).NotTo(HaveOccurred())
		defer func() { _ = db.Close() }()

		g.Expect(eng.Close()).To(Succeed())
		g.Expect(db.PingContext(context.Background())).To(Succeed())
	})

	t.Run("OwnDB flips a wrapped engine to owning", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		db, err := sql.Open("sqlite", ":memory:")
		g.Expect(err).NotTo(HaveOccurred())
		db.SetMaxOpenConns(1)

		eng, err := sqliteengine.NewSQLiteEngine(db)
		g.Expect(err).NotTo(HaveOccurred())

		sqliteengine.OwnDB(eng)
		g.Expect(eng.Close()).To(Succeed())

		g.Expect(db.PingContext(context.Background())).NotTo(Succeed())
	})
}
