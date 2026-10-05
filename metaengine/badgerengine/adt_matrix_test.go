package badgerengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
	"github.com/onsi/gomega"
)

// adt_matrix_test.go runs the full 7-ADT test matrix across the Badger
// engine and the memory engine, asserting cross-engine parity. By
// transitivity, Badger produces identical results to Memory and Pebble.

func TestBadgerADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "badger",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng, err := badgerengine.NewBadgerEngine("")
				gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())

				return eng
			},
		},
	})
}

// TestCapabilityConformance verifies this engine's Profile() declarations
// against its implemented backend interfaces (declared-vs-implemented table).
func TestCapabilityConformance(t *testing.T) {
	t.Parallel()

	eng, err := badgerengine.NewBadgerEngine("")
	if err != nil {
		t.Skipf("badger not available: %v", err)
	}

	adttest.RunCapabilityConformance(t, "badger", eng, nil)
}
