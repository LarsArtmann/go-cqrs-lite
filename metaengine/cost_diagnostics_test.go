package metaengine_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

var _ = Describe("Planner diagnostics (volume + selectivity)", func() {
	Describe("volume default", func() {
		When("the query declares no volume hint", func() {
			It("emits an INFO diagnostic that the 1000-item default was assumed", func() {
				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, findTaskQuery(),
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				diags := store.Plan().Queries[0].Diagnostics
				Expect(hasDiagnostic(
					diags, metaengine.DiagLevelInfo, "volume not set",
				)).To(BeTrue(), "expected volume-default INFO among: %v", diags)
			})
		})

		When("the query declares a volume hint", func() {
			It("does not emit the volume-default diagnostic", func() {
				q := metaengine.Query[FindTask, FindTaskResult](
					"find_task_volume_set",
					metaengine.OnRecord(
						TaskCreated{},
						func(_ record.Record, e TaskCreated) (TaskID, FindTaskResult) {
							return e.ID, FindTaskResult{ID: e.ID, Title: e.Title}
						},
					),
					metaengine.OnRecord(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
					metaengine.Volume(10_000),
				)

				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, q,
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				diags := store.Plan().Queries[0].Diagnostics
				Expect(hasDiagnostic(
					diags, metaengine.DiagLevelInfo, "volume not set",
				)).To(BeFalse(), "unexpected volume-default INFO among: %v", diags)
			})
		})
	})

	Describe("filter selectivity", func() {
		When("a filtered query routes to a scan-complexity engine", func() {
			It("emits an INFO diagnostic with the estimated selectivity", func() {
				q := metaengine.Query[ListTasksByStatus, ListTasksByStatusResult](
					"list_selectivity",
					metaengine.OnRecord(
						TaskCreated{},
						func(_ record.Record, e TaskCreated) (TaskID, FindTaskResult) {
							return e.ID, FindTaskResult{ID: e.ID, Title: e.Title, Status: e.Status}
						},
					),
					metaengine.OnRecord(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
					metaengine.FilterOn(func(r FindTaskResult) string { return r.Status }),
				)

				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, q,
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				diags := store.Plan().Queries[0].Diagnostics
				Expect(hasDiagnostic(
					diags, metaengine.DiagLevelInfo, "estimated selectivity",
				)).To(BeTrue(), "expected selectivity INFO among: %v", diags)
			})
		})

		When("the query declares no filters", func() {
			It("does not emit the selectivity diagnostic", func() {
				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, findTaskQuery(),
				)
				Expect(err).NotTo(HaveOccurred())
				defer store.Close()

				diags := store.Plan().Queries[0].Diagnostics
				Expect(hasDiagnostic(
					diags, metaengine.DiagLevelInfo, "estimated selectivity",
				)).To(BeFalse(), "unexpected selectivity INFO among: %v", diags)
			})
		})
	})
})
