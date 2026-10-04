package pebbleengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	. "github.com/onsi/gomega"
)

func TestPebbleHealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng := mustNewPebbleEngine(t)

	hc, ok := eng.(metaengine.HealthChecker)
	g.Expect(ok).To(BeTrue())

	g.Expect(hc.HealthCheck(context.Background())).To(Succeed())
}

func TestPebbleHealthCheck_ClosedDB(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng := newPebbleEngineOrSkip(t)

	g.Expect(eng.Close()).To(Succeed())

	hc := eng.(metaengine.HealthChecker)

	err := hc.HealthCheck(context.Background())
	g.Expect(err).To(HaveOccurred())
}
