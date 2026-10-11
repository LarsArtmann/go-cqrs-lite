package tursoengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	. "github.com/onsi/gomega"
)

func TestTursoHealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	eng := mustNewTursoEngine(t)

	hc, ok := eng.(metaengine.HealthChecker)
	g.Expect(ok).To(BeTrue())
	g.Expect(hc.HealthCheck(context.Background())).To(Succeed())
}
