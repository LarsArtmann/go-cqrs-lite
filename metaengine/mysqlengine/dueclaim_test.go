package mysqlengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestMySQLEngineDueClaims pins the engine's ADR-0142 capabilities to the
// shared conformance contract: claimkit (MySQL dialect) provides the
// implementation, this proves the wiring holds. Live-gated: set
// MYSQL_TEST_DSN (see helper_test.go).
func TestMySQLEngineDueClaims(t *testing.T) {
	eng := mustNewMySQLEngine(t)
	// NOTE: no t.Parallel — the shared live server serializes better.

	adttest.AssertDueClaimer(t, []adttest.Factory{
		{Name: "mysql", Create: func(*testing.T) metaengine.Engine { return eng }},
	})
	adttest.AssertDedupStore(t, []adttest.Factory{
		{Name: "mysql", Create: func(*testing.T) metaengine.Engine { return eng }},
	})
}

func TestMySQLEngineDueClaims_CapabilitySurface(t *testing.T) {
	eng := mustNewMySQLEngine(t)

	if !metaengine.SupportsDueClaims(eng) {
		t.Fatal("mysql engine must satisfy DueClaimer after claimkit wiring")
	}

	if !metaengine.SupportsDedup(eng) {
		t.Fatal("mysql engine must satisfy DedupStore after claimkit wiring")
	}

	if _, ok := eng.(metaengine.FactSink); !ok {
		t.Fatal("mysql engine must satisfy FactSink after claimkit wiring")
	}

	complexity, ok := eng.Profile().SupportsADT(metaengine.ADTDueClaim)
	if !ok || complexity != metaengine.ComplexityOLogN {
		t.Fatalf("profile must declare ADTDueClaim at O(logN), got %s/%v", complexity, ok)
	}

	if complexity, ok := eng.Profile().
		SupportsADT(metaengine.ADTDedup); !ok ||
		complexity != metaengine.ComplexityOLogN {
		t.Fatalf("profile must declare ADTDedup at O(logN), got %s/%v", complexity, ok)
	}

	if eng.Profile().IsDegraded(metaengine.ADTDueClaim) {
		t.Fatal("mysql claims are native (SKIP LOCKED SQL), not degraded")
	}
}
