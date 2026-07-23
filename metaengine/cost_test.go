package metaengine_test

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cost model", func() {
	Describe("CostEstimate WithinBudget", func() {
		DescribeTable("budget enforcement",
			func(latencyMs float64, budgetMs int64, expected bool) {
				ce := metaengine.CostEstimate{EstimatedLatencyMs: latencyMs}
				Expect(ce.WithinBudget(budgetMs)).To(Equal(expected))
			},
			Entry("latency under budget", 0.5, 1, true),
			Entry("latency exactly at budget", 1.0, 1, true),
			Entry("latency over budget", 2.0, 1, false),
			Entry("zero budget means unlimited", 9999.0, 0, true),
			Entry("negative budget means unlimited", 9999.0, -1, true),
		)
	})

	Describe("CostEstimate String", func() {
		It("includes complexity, volume, ops, and latency", func() {
			ce := metaengine.CostEstimate{
				Complexity:         metaengine.ComplexityON,
				Volume:             1000,
				EstimatedOps:       1000,
				EstimatedLatencyMs: 0.1,
			}
			s := ce.String()
			Expect(s).To(ContainSubstring("O(N)"))
			Expect(s).To(ContainSubstring("vol=1000"))
			Expect(s).To(ContainSubstring("latency=0.100ms"))
		})
	})

	Describe("plan cost estimation", func() {
		When("a query declares a volume hint", func() {
			It("populates the cost estimate with the volume", func() {
				q := metaengine.Query[FindTask, FindTaskResult](
					"find_task_vol",
					metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
						return e.ID, FindTaskResult{ID: e.ID, Title: e.Title}
					}),
					metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
					metaengine.Volume(500_000),
				)

				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, q,
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				plan := store.Plan()
				Expect(plan.Queries).To(HaveLen(1))
				Expect(plan.Queries[0].Cost.Volume).To(Equal(int64(500_000)))
				Expect(plan.Queries[0].Cost.EstimatedLatencyMs).To(BeNumerically(">", 0))
			})
		})

		When("a query declares no volume hint", func() {
			It("uses the default volume of 1000", func() {
				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, findTaskQuery(),
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				plan := store.Plan()
				Expect(plan.Queries[0].Cost.Volume).To(Equal(int64(1000)))
			})
		})
	})

	Describe("latency budget enforcement", func() {
		When("the estimated latency exceeds the declared budget", func() {
			It("emits a WARN diagnostic with the estimated latency", func() {
				q := metaengine.Query[FindTask, FindTaskResult](
					"find_task_budget",
					metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
						return e.ID, FindTaskResult{ID: e.ID, Title: e.Title}
					}),
					metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
					// O(1) at 1M items → negligible latency, but set budget to 0.00001ms
					metaengine.Volume(1_000_000),
					metaengine.WithLatencyBudget(0),
				)

				// WithLatencyBudget(0) means unlimited, so no warning.
				// Instead, test with a real budget on a scan query.
				_ = q

				// For a filtered scan at high volume, O(N) is expensive.
				scanQ := metaengine.Query[ListTasksByStatus, ListTasksByStatusResult](
					"list_budget",
					metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
						return e.ID, FindTaskResult{ID: e.ID, Title: e.Title, Status: e.Status}
					}),
					metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
					metaengine.FilterOn(func(r FindTaskResult) string { return r.Status }),
					metaengine.Volume(1_000_000),
					metaengine.WithLatencyBudget(1),
				)

				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, scanQ,
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				assignment := store.Plan().Queries[0]
				Expect(assignment.Diagnostics).To(ContainElement(MatchFields(
					IgnoreExtras,
					Fields{
						"Level":   Equal(metaengine.DiagLevelWarn),
						"Message": MatchRegexp("estimated latency .* exceeds budget 1ms"),
					},
				)))
			})
		})
	})

	Describe("scale threshold warnings", func() {
		When("volume exceeds the optimal range for the ADT structure", func() {
			It("emits a WARN diagnostic suggesting a disk-backed engine", func() {
				q := metaengine.Query[FindTask, FindTaskResult](
					"find_task_huge",
					metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
						return e.ID, FindTaskResult{ID: e.ID, Title: e.Title}
					}),
					metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
					metaengine.Volume(50_000_000),
				)

				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, q,
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				assignment := store.Plan().Queries[0]
				Expect(assignment.Diagnostics).To(ContainElement(MatchFields(
					IgnoreExtras,
					Fields{
						"Level":   Equal(metaengine.DiagLevelWarn),
						"Message": MatchRegexp("exceeds optimal range.*disk-backed"),
					},
				)))
			})
		})

		When("volume is within the optimal range", func() {
			It("does not emit a scale threshold warning", func() {
				q := metaengine.Query[FindTask, FindTaskResult](
					"find_task_normal",
					metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
						return e.ID, FindTaskResult{ID: e.ID, Title: e.Title}
					}),
					metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
					metaengine.Volume(10_000),
				)

				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, q,
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				assignment := store.Plan().Queries[0]
				for _, d := range assignment.Diagnostics {
					Expect(d.Message).NotTo(MatchRegexp("exceeds optimal range"))
				}
			})
		})
	})
})

var _ = Describe("Write amplification budget", func() {
	Describe("default budget", func() {
		It("equals 3", func() {
			Expect(metaengine.DefaultWriteAmplificationBudget).To(Equal(3))
		})
	})

	When("the same event updates more projections than the default budget", func() {
		It("emits a write amplification warning", func() {
			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				allQueries()...,
			)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			// With 3 queries listening to TaskCreated, we need one more to trigger
			// the default budget of 3. Add a 4th query.
			extra := metaengine.Query[FindTask, FindTaskResult](
				"find_task_dup",
				metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
					return e.ID, FindTaskResult{ID: e.ID}
				}),
				metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
			)

			store2, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				findTaskQuery(), listTasksByStatusQuery(), countByStatusQuery(), extra,
			)
			Expect(err).NotTo(HaveOccurred())
			defer store2.Close()

			plan := store2.Plan()
			Expect(plan.Diagnostics.HasWarnings()).To(BeTrue())
		})
	})

	When("the budget is raised via WithWriteAmplificationBudget", func() {
		It("does not warn when amplification is within the raised budget", func() {
			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				findTaskQuery(), listTasksByStatusQuery(), countByStatusQuery(),
				metaengine.WithWriteAmplificationBudget(10),
			)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			plan := store.Plan()
			Expect(plan.Diagnostics.HasWarnings()).To(BeFalse())
		})
	})
})
