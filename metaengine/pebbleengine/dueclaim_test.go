package pebbleengine_test

import (
	"testing"

	pebbleengine "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// newEngine builds a fresh in-memory pebble engine. Ownership passes to the
// caller: the adttest suites close engines they drive (DeferClose), and a
// factory-registered cleanup would double-close — pebble panics on that.
func newEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	eng, err := pebbleengine.NewPebbleEngine("") // in-memory vfs
	if err != nil {
		t.Fatalf("NewPebbleEngine: %v", err)
	}

	return eng
}

// TestPebbleDueClaims pins the engine's ADR-0142 capabilities to the shared
// conformance contract via the Map runtimes.
func TestPebbleDueClaims(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "pebble", Create: newEngine}})
	adttest.AssertDedupStore(t, []adttest.Factory{{Name: "pebble", Create: newEngine}})
}

func TestPebbleDueClaims_ProfileDeclaresCapabilities(t *testing.T) {
	t.Parallel()

	eng := newEngine(t)
	t.Cleanup(func() { _ = eng.Close() })

	if !metaengine.SupportsDueClaims(eng) || !metaengine.SupportsDedup(eng) {
		t.Fatal("pebble engine must satisfy DueClaimer + DedupStore after wiring")
	}

	if _, ok := eng.(metaengine.FactSink); ok {
		t.Fatal("map-shaped engines must NOT claim FactSink (no shared tx across engine calls)")
	}

	if !eng.Profile().IsDegraded(metaengine.ADTDueClaim) {
		t.Fatal("pebble claims are the degraded Map runtime — profile must say so")
	}
}
