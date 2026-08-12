package pebbleengine_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestPebbleHealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	hc, ok := eng.(metaengine.HealthChecker)
	g.Expect(ok).To(BeTrue())

	g.Expect(hc.HealthCheck(context.Background())).To(Succeed())
}

func TestPebbleHealthCheck_ClosedDB(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(eng.Close()).To(Succeed())

	hc := eng.(metaengine.HealthChecker)

	err = hc.HealthCheck(context.Background())
	g.Expect(err).To(HaveOccurred())
}
