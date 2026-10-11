package tursoengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
	"github.com/onsi/gomega"
)

func TestTursoADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "turso",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng := mustNewTursoEngine(t)
				gomega.NewWithT(t).Expect(eng).NotTo(gomega.BeNil())

				return eng
			},
		},
	})
}

// TestCapabilityConformance verifies this engine's Profile() declarations
// against its implemented backend interfaces (declared-vs-implemented table).
func TestCapabilityConformance(t *testing.T) {
	t.Parallel()

	adttest.RunCapabilityConformance(t, "turso", mustNewTursoEngine(t), nil)
}
