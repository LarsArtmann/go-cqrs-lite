package badgerengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
	badgerengine "github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
)

// newEngine builds a fresh in-memory badger engine. Ownership passes to the
// caller: the adttest suites close engines they drive (DeferClose), and a
// factory-registered cleanup would double-close.
func newEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	eng, err := badgerengine.NewBadgerEngine("")
	if err != nil {
		t.Fatalf("NewBadgerEngine: %v", err)
	}

	return eng
}

func TestBadgerDueClaims(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "badger", Create: newEngine}})
	adttest.AssertDedupStore(t, []adttest.Factory{{Name: "badger", Create: newEngine}})
}

func TestBadgerDueClaims_ProfileDeclaresCapabilities(t *testing.T) {
	t.Parallel()

	eng := newEngine(t)
	t.Cleanup(func() { _ = eng.Close() })

	if !metaengine.SupportsDueClaims(eng) || !metaengine.SupportsDedup(eng) {
		t.Fatal("badger engine must satisfy DueClaimer + DedupStore after wiring")
	}

	if _, ok := eng.(metaengine.FactSink); ok {
		t.Fatal("map-shaped engines must NOT claim FactSink (no shared tx across engine calls)")
	}

	if !eng.Profile().IsDegraded(metaengine.ADTDueClaim) {
		t.Fatal("badger claims are the degraded Map runtime — profile must say so")
	}
}
