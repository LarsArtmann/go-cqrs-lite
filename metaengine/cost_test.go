package metaengine_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// hasDiagnostic checks if any diagnostic in the list has the given level and
// contains the message substring.
func hasDiagnostic(diags []metaengine.Diagnostic, level string, messageSubstr string) bool {
	for _, d := range diags {
		if d.Level == level && strings.Contains(d.Message, messageSubstr) {
			return true
		}
	}

	return false
}

var _ = Describe("Cost model", func() {
	Describe("CostEstimate WithinBudget", func() {
		It("returns true when latency is under budget", func() {
			ce := metaengine.CostEstimate{EstimatedLatencyMs: 0.5}
			Expect(ce.WithinBudget(1)).To(BeTrue())
		})

		It("returns true when latency equals budget", func() {
			ce := metaengine.CostEstimate{EstimatedLatencyMs: 1.0}
			Expect(ce.WithinBudget(1)).To(BeTrue())
		})

		It("returns false when latency exceeds budget", func() {
			ce := metaengine.CostEstimate{EstimatedLatencyMs: 2.0}
			Expect(ce.WithinBudget(1)).To(BeFalse())
		})

		It("returns true for zero budget (unlimited)", func() {
			ce := metaengine.CostEstimate{EstimatedLatencyMs: 9999.0}
			Expect(ce.WithinBudget(0)).To(BeTrue())
		})

		It("returns true for negative budget (unlimited)", func() {
			ce := metaengine.CostEstimate{EstimatedLatencyMs: 9999.0}
			Expect(ce.WithinBudget(-1)).To(BeTrue())
		})
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
		When("a filtered scan at high volume exceeds the latency budget", func() {
			It("emits a WARN diagnostic naming the estimated latency", func() {
				q := metaengine.Query[ListTasksByStatus, ListTasksByStatusResult](
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
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, q,
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				assignment := store.Plan().Queries[0]
				Expect(hasDiagnostic(
					assignment.Diagnostics, metaengine.DiagLevelWarn, "exceeds budget 1ms",
				)).To(BeTrue(), "expected latency budget warning among: %v", assignment.Diagnostics)
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
				Expect(hasDiagnostic(
					assignment.Diagnostics, metaengine.DiagLevelWarn, "exceeds optimal range",
				)).To(BeTrue(), "expected scale threshold warning among: %v", assignment.Diagnostics)
				Expect(hasDiagnostic(
					assignment.Diagnostics, metaengine.DiagLevelWarn, "disk-backed engine",
				)).To(BeTrue(), "expected disk-backed engine suggestion among: %v", assignment.Diagnostics)
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
				Expect(hasDiagnostic(
					assignment.Diagnostics, metaengine.DiagLevelWarn, "exceeds optimal range",
				)).To(BeFalse())
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

	When("four queries listen to the same event type", func() {
		It("emits a write amplification warning with the default budget of 3", func() {
			extra := metaengine.Query[FindTask, FindTaskResult](
				"find_task_extra",
				metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
					return e.ID, FindTaskResult{ID: e.ID}
				}),
				metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
			)

			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				findTaskQuery(), listTasksByStatusQuery(), countByStatusQuery(), extra,
			)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			plan := store.Plan()
			Expect(plan.Diagnostics.HasWarnings()).To(BeTrue())
			Expect(hasDiagnostic(
				plan.Diagnostics, metaengine.DiagLevelWarn, "write amplification",
			)).To(BeTrue(), "expected write amplification warning among: %v", plan.Diagnostics)
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
