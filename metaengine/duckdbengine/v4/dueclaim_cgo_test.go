//go:build cgo

package duckdbengine_test

import (
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// newEngine builds a fresh, isolated in-memory DuckDB engine per test.
func newClaimEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Fatalf("duckdbengine.New: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

// TestDuckDBDueClaims pins the engine's ADR-0142 capabilities to the shared
// conformance contract: claimkit (DuckDB dialect) provides the
// implementation, this proves the wiring holds.
func TestDuckDBDueClaims(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "duckdb", Create: newClaimEngine}})
	adttest.AssertDedupStore(t, []adttest.Factory{{Name: "duckdb", Create: newClaimEngine}})
	adttest.AssertFactSink(t, []adttest.Factory{{Name: "duckdb", Create: newClaimEngine}})
}

func TestDuckDBDueClaims_CapabilitySurface(t *testing.T) {
	t.Parallel()

	eng := newClaimEngine(t)

	if !metaengine.SupportsDueClaims(eng) {
		t.Fatal("duckdb engine must satisfy DueClaimer after claimkit wiring")
	}

	if !metaengine.SupportsDedup(eng) {
		t.Fatal("duckdb engine must satisfy DedupStore after claimkit wiring")
	}

	if _, ok := eng.(metaengine.FactSink); !ok {
		t.Fatal("duckdb engine must satisfy FactSink after claimkit wiring")
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
		t.Fatal("duckdb claims are native (indexed SQL), not degraded")
	}
}
