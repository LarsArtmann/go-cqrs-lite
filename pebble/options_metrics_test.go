package pebble

import (
	"log/slog"
	"testing"

	"github.com/onsi/gomega"
)

func TestDefaultOptions(t *testing.T) {
	g := gomega.NewWithT(t)

	opts := DefaultOptions()

	g.Expect(opts).NotTo(gomega.BeNil())
	g.Expect(opts.Levels).To(gomega.HaveLen(7))

	for i, level := range opts.Levels {
		g.Expect(level.FilterPolicy).NotTo(gomega.BeNil(), "level %d should have filter policy", i)
	}
}

func TestDefaultOptionsWithLogging(t *testing.T) {
	g := gomega.NewWithT(t)

	opts := DefaultOptionsWithLogging(slog.Default())

	g.Expect(opts).NotTo(gomega.BeNil())
	g.Expect(opts.EventListener).NotTo(gomega.BeNil())
	g.Expect(opts.Levels).To(gomega.HaveLen(7))
}

func TestPebbleMetricsBlockCacheHitRate(t *testing.T) {
	tests := []struct {
		name     string
		m        PebbleMetrics
		expected float64
	}{
		{name: "no accesses", m: PebbleMetrics{}, expected: 0.0},
		{name: "all hits", m: PebbleMetrics{BlockCacheHits: 100}, expected: 1.0},
		{name: "all misses", m: PebbleMetrics{BlockCacheMisses: 100}, expected: 0.0},
		{
			name:     "half hits",
			m:        PebbleMetrics{BlockCacheHits: 50, BlockCacheMisses: 50},
			expected: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.m.BlockCacheHitRate()).To(gomega.BeNumerically("~", tt.expected, 0.001))
		})
	}
}

func TestBackendMetrics(t *testing.T) {
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	backend, err := Open(dir, nil, slog.Default())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer func() { _ = backend.Close() }()

	m := backend.Metrics()
	g.Expect(m.NumFilesTotal).To(gomega.BeNumerically(">=", 0))
	g.Expect(m.BlockCacheHitRate()).To(gomega.BeNumerically(">=", 0.0))
}
