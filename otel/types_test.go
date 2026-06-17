package otel

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestCQRSHistogramBoundaries(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(CQRSHistogramBoundaries).NotTo(gomega.BeEmpty())
	// Should cover the critical CQRS range (0.05ms to 10s).
	g.Expect(CQRSHistogramBoundaries[0]).To(gomega.BeNumerically("<=", 1.0))
	g.Expect(CQRSHistogramBoundaries[len(CQRSHistogramBoundaries)-1]).
		To(gomega.BeNumerically(">=", 5000.0))

	// Should be monotonically increasing.
	for i := 1; i < len(CQRSHistogramBoundaries); i++ {
		g.Expect(CQRSHistogramBoundaries[i]).
			To(gomega.BeNumerically(">", CQRSHistogramBoundaries[i-1]))
	}
}

func TestServiceResourceAttributes(t *testing.T) {
	g := gomega.NewWithT(t)

	attrs := ServiceResourceAttributes("my-app", "1.0.0", "inst-1")
	g.Expect(attrs).To(gomega.HaveLen(3))

	for _, attr := range attrs {
		g.Expect(attr.Key).NotTo(gomega.BeEmpty())
		g.Expect(attr.Value.AsString()).NotTo(gomega.BeEmpty())
	}
}

func TestAttrHelpers(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(AttrString("k", "v").Value.AsString()).To(gomega.Equal("v"))
	g.Expect(AttrInt("k", 42).Value.AsInt64()).To(gomega.Equal(int64(42)))
	g.Expect(AttrInt64("k", 99).Value.AsInt64()).
		To(gomega.Equal(int64(99)))
}

func TestCounterMetricHelpers(t *testing.T) {
	g := gomega.NewWithT(t)

	// These should not panic — verify they return non-nil options.
	g.Expect(CounterMetricWithDescription("test")).NotTo(gomega.BeNil())
	g.Expect(CounterMetricWithUnit("count")).NotTo(gomega.BeNil())
}
