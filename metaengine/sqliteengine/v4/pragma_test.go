package sqliteengine_test

import (
	"context"
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
