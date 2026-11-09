package bboltengine_test

import (
	"testing"

	bboltengine "github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// newEngine builds a fresh volatile bbolt engine. Ownership passes to the
// caller: the adttest suites close engines they drive (DeferClose), and a
// factory-registered cleanup would double-close.
func newEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	eng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		t.Fatalf("NewBboltEngine: %v", err)
	}

	return eng
}

func TestBboltDueClaims(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "bbolt", Create: newEngine}})
	adttest.AssertDedupStore(t, []adttest.Factory{{Name: "bbolt", Create: newEngine}})
}

func TestBboltDueClaims_ProfileDeclaresCapabilities(t *testing.T) {
	t.Parallel()

	eng := newEngine(t)
	t.Cleanup(func() { _ = eng.Close() })

	if !metaengine.SupportsDueClaims(eng) || !metaengine.SupportsDedup(eng) {
		t.Fatal("bbolt engine must satisfy DueClaimer + DedupStore after wiring")
	}

	if _, ok := eng.(metaengine.FactSink); ok {
		t.Fatal("map-shaped engines must NOT claim FactSink (no shared tx across engine calls)")
	}

	if !eng.Profile().IsDegraded(metaengine.ADTDueClaim) {
		t.Fatal("bbolt claims are the degraded Map runtime — profile must say so")
	}
}
