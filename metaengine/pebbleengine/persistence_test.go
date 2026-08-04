package pebbleengine_test

import (
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
)

func TestPebblePersistence_InMemoryIsVolatile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	g.Expect(eng.Profile().IsVolatile()).To(BeTrue(),
		"in-memory Pebble engine (dir=\"\") should be volatile")
	g.Expect(eng.Profile().IsPersistent()).To(BeFalse())
}

func TestPebblePersistence_OnDiskIsPersistent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := filepath.Join(t.TempDir(), "pebble")
	eng, err := pebbleengine.NewPebbleEngine(dir)
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	g.Expect(eng.Profile().IsPersistent()).To(BeTrue(),
		"on-disk Pebble engine should be persistent")
	g.Expect(eng.Profile().IsVolatile()).To(BeFalse())
}

func TestPebblePersistence_FromDBIsPersistent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := filepath.Join(t.TempDir(), "pebble")
	db, err := pebble.Open(dir, &pebble.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	defer db.Close()

	eng := pebbleengine.NewPebbleEngineFromDB(db)
	defer eng.Close()

	g.Expect(eng.Profile().IsPersistent()).To(BeTrue(),
		"Pebble engine from a caller-owned DB should be persistent")
	g.Expect(eng.Profile().IsVolatile()).To(BeFalse())
}
