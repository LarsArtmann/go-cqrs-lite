package bboltengine_test

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

func TestBboltADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "bbolt",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng, err := bboltengine.NewBboltEngine("")
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

	eng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		t.Skipf("bbolt not available: %v", err)
	}

	adttest.RunCapabilityConformance(t, "bbolt", eng, nil)
}
