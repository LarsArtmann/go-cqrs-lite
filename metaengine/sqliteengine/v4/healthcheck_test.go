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

func newSQLiteEngineForHC(t *testing.T) (metaengine.Engine, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("sqliteengine.NewSQLiteEngine: %v", err)
	}

	return eng, db
}

func TestSQLiteHealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng, db := newSQLiteEngineForHC(t)
	defer func() {
		_ = eng.Close()
		_ = db.Close()
	}()

	hc, ok := eng.(metaengine.HealthChecker)
	g.Expect(ok).To(BeTrue())

	g.Expect(hc.HealthCheck(context.Background())).To(Succeed())
}

func TestSQLiteHealthCheck_ClosedDB(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng, db := newSQLiteEngineForHC(t)

	_ = eng.Close()
	_ = db.Close()

	hc := eng.(metaengine.HealthChecker)

	err := hc.HealthCheck(context.Background())
	g.Expect(err).To(HaveOccurred())
}
