package tursoengine_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// newFileEngine opens a fresh engine over a LOCAL libSQL file database — the
// real tursogo driver path (not the :memory: default), which is what embedded
// Turso deployments run claimkit over. Each call gets its own file because
// the conformance factories build several isolated engines per package run.
func newFileEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	dsn := filepath.Join(t.TempDir(),
		fmt.Sprintf("turso_claims_%d.db", time.Now().UnixNano()))

	eng, err := tursoengine.New(dsn)
	if err != nil {
		t.Skipf("turso not available: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

// TestTursoDueClaims pins the libSQL path to the shared ADR-0142 conformance
// contract. The engine delegates to sqliteengine, so the wiring is inherited
// — this proves it actually holds over the turso driver on a local file DSN.
func TestTursoDueClaims(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "turso-file", Create: newFileEngine}})
	adttest.AssertDedupStore(t, []adttest.Factory{{Name: "turso-file", Create: newFileEngine}})
	adttest.AssertFactSink(t, []adttest.Factory{{Name: "turso-file", Create: newFileEngine}})
}

func TestTursoDueClaims_CapabilitySurface(t *testing.T) {
	t.Parallel()

	eng := newFileEngine(t)

	if !metaengine.SupportsDueClaims(eng) {
		t.Fatal("turso engine must satisfy DueClaimer via sqliteengine delegation")
	}

	if !metaengine.SupportsDedup(eng) {
		t.Fatal("turso engine must satisfy DedupStore via sqliteengine delegation")
	}

	if complexity, ok := eng.Profile().
		SupportsADT(metaengine.ADTDueClaim); !ok ||
		complexity != metaengine.ComplexityOLogN {
		t.Fatalf("profile must declare ADTDueClaim at O(logN), got %s/%v", complexity, ok)
	}

	if complexity, ok := eng.Profile().
		SupportsADT(metaengine.ADTDedup); !ok ||
		complexity != metaengine.ComplexityOLogN {
		t.Fatalf("profile must declare ADTDedup at O(logN), got %s/%v", complexity, ok)
	}

	if eng.Profile().IsDegraded(metaengine.ADTDueClaim) {
		t.Fatal("turso claims are native (indexed SQL), not degraded")
	}
}
