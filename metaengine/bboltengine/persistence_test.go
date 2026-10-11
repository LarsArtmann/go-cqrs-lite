package bboltengine_test

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	. "github.com/onsi/gomega"
	bolt "go.etcd.io/bbolt"
)

func TestBboltPersistence_InMemoryIsVolatile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng := mustNewBboltEngine(t)

	g.Expect(eng.Profile().IsVolatile()).To(BeTrue(),
		"in-memory bbolt engine (path=\"\") should be volatile")
	g.Expect(eng.Profile().IsPersistent()).To(BeFalse())
}

func TestBboltPersistence_OnDiskIsPersistent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := filepath.Join(t.TempDir(), "bbolt.db")
	eng, err := bboltengine.NewBboltEngine(dir)
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	g.Expect(eng.Profile().IsPersistent()).To(BeTrue(),
		"on-disk bbolt engine should be persistent")
	g.Expect(eng.Profile().IsVolatile()).To(BeFalse())
}

func TestBboltPersistence_FromDBIsPersistent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := filepath.Join(t.TempDir(), "bbolt.db")
	db, err := bolt.Open(dir, 0o600, bolt.DefaultOptions)
	g.Expect(err).NotTo(HaveOccurred())
	defer db.Close()

	eng, err := bboltengine.NewBboltEngineFromDB(db)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(eng.Profile().IsPersistent()).To(BeTrue(),
		"bbolt engine from a caller-owned DB should be persistent")
	g.Expect(eng.Profile().IsVolatile()).To(BeFalse())
}
