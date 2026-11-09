package metaengine_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

var _ = Describe("Benchmark mode (ADR-0124 §6.3)", func() {
	Describe("BenchmarkPlan", func() {
		It("compares two priority configs and returns results", func() {
			fast := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "fast",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}
			slow := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "slow",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityON,
				},
			}}

			summary, err := metaengine.BenchmarkPlan(
				context.Background(),
				[]metaengine.Engine{slow, fast},
				[]any{findTaskQuery()},
				metaengine.BenchmarkConfig{
					Iterations: 500,
					PriorityConfigs: []*metaengine.PriorityConfig{
						{Global: metaengine.PriorityReadSpeed},
						{Global: metaengine.PriorityWriteSpeed},
						{Global: metaengine.PriorityBalanced},
					},
					Labels: []string{"read-optimized", "write-optimized", "balanced"},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(summary.Results).To(HaveLen(3))

			// All should select the fast engine (O(1) beats O(N) regardless of priority)
			for _, r := range summary.Results {
				Expect(r.EngineName).To(Equal("fast"))
				Expect(r.Throughput).To(BeNumerically(">", 0))
			}

			// ReadSpeed should have lower P50 latency (O(1) factor is lower)
			readResult := summary.Results[0]
			writeResult := summary.Results[1]
			Expect(readResult.LatencyP50).To(BeNumerically("<=", writeResult.LatencyP50))

			// Labels are preserved
			Expect(summary.Results[0].Label).To(Equal("read-optimized"))
			Expect(summary.Results[1].Label).To(Equal("write-optimized"))
			Expect(summary.Results[2].Label).To(Equal("balanced"))
		})

		It("produces a human-readable comparison table", func() {
			fast := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "engine-a",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}

			summary, err := metaengine.BenchmarkPlan(
				context.Background(),
				[]metaengine.Engine{fast},
				[]any{findTaskQuery()},
				metaengine.BenchmarkConfig{
					Iterations: 100,
					PriorityConfigs: []*metaengine.PriorityConfig{
						{Global: metaengine.PriorityReadSpeed},
						{Global: metaengine.PriorityBalanced},
					},
					Labels: []string{"read", "balanced"},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			table := summary.FormatTable()
			Expect(table).To(ContainSubstring("read"))
			Expect(table).To(ContainSubstring("balanced"))
			Expect(table).To(ContainSubstring("THROUGHPUT"))
		})

		It("errors when no priority configs provided", func() {
			_, err := metaengine.BenchmarkPlan(
				context.Background(),
				[]metaengine.Engine{&fakeEngine{profile: metaengine.EngineProfile{
					Name: "test",
					Supports: map[metaengine.ADT]metaengine.Complexity{
						metaengine.ADTMap: metaengine.ComplexityO1,
					},
				}}},
				[]any{findTaskQuery()},
				metaengine.BenchmarkConfig{},
			)
			Expect(err).To(HaveOccurred())
		})
	})
})
