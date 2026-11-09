package metaengine_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

var _ = Describe("Plan", func() {
	Describe("with a single memory engine", func() {
		var store *metaengine.Store
		var plan *metaengine.PlanResult

		BeforeEach(func() {
			var err error
			store, err = metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				allQueries()...,
			)
			Expect(err).NotTo(HaveOccurred())
			plan = store.Plan()
		})

		AfterEach(func() {
			Expect(store.Close()).To(Succeed())
		})

		It("assigns every declared query to an engine", func() {
			Expect(plan.Queries).To(HaveLen(5))
		})

		It("assigns each query the correct ADT", func() {
			byName := make(map[string]metaengine.QueryAssignment)
			for _, q := range plan.Queries {
				byName[q.QueryName] = q
			}

			Expect(byName["find_task"].ADT).To(Equal(metaengine.ADTMap))
			Expect(byName["check_assignee"].ADT).To(Equal(metaengine.ADTSet))
			Expect(byName["list_tasks_by_status"].ADT).To(Equal(metaengine.ADTMap))
			Expect(byName["count_by_status"].ADT).To(Equal(metaengine.ADTCounter))
			Expect(byName["tasks_by_assignee"].ADT).To(Equal(metaengine.ADTGraph))
		})

		It("assigns each query a complexity from the engine profile", func() {
			byName := make(map[string]metaengine.QueryAssignment)
			for _, q := range plan.Queries {
				byName[q.QueryName] = q
			}

			Expect(byName["find_task"].Complexity).To(Equal(metaengine.ComplexityO1))
			Expect(byName["check_assignee"].Complexity).To(Equal(metaengine.ComplexityO1))
			Expect(
				byName["count_by_status"].Complexity,
			).To(Equal(metaengine.ComplexityO1))
			Expect(
				byName["tasks_by_assignee"].Complexity,
			).To(Equal(metaengine.ComplexityODegree), "graph on memory is O(degree^depth), not degraded")
		})

		It("assigns the memory engine to every query", func() {
			for _, q := range plan.Queries {
				Expect(q.EngineName).To(Equal("memory"))
			}
		})

		It("detects pagination on list_tasks_by_status", func() {
			for _, q := range plan.Queries {
				if q.QueryName == "list_tasks_by_status" {
					Expect(q.IsPaginated).To(BeTrue())
				}
			}
		})
	})

	When("no engines are provided", func() {
		It("returns an error explaining at least one engine is required", func() {
			_, err := metaengine.Plan(nil, findTaskQuery())
			Expect(err).To(MatchError(MatchRegexp("at least one engine required")))
		})
	})

	When("no queries are provided", func() {
		It("returns an error explaining at least one query is required", func() {
			_, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()})
			Expect(err).To(MatchError(MatchRegexp("at least one query required")))
		})
	})

	When("two queries share the same name", func() {
		It("returns an error identifying the duplicate", func() {
			_, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				findTaskQuery(),
				findTaskQuery(),
			)
			Expect(err).To(MatchError(MatchRegexp("duplicate query name")))
		})
	})

	When("an argument is not a valid query declaration", func() {
		It("returns an error about missing queryMeta", func() {
			_, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				"not a query",
			)
			Expect(err).To(MatchError(MatchRegexp("does not implement queryMeta")))
		})
	})
})

var _ = Describe("PlanResult Report", func() {
	It("produces a human-readable report string", func() {
		store, err := metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			findTaskQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
		defer store.Close()

		report := store.Plan().Report()
		Expect(report).To(ContainSubstring("Meta-Engine Plan"))
		Expect(report).To(ContainSubstring("find_task"))
	})
})

var _ = Describe("QueryDecl String", func() {
	It("includes the query name, ADT, and read pattern", func() {
		q := findTaskQuery()
		s := q.String()
		Expect(s).To(ContainSubstring("find_task"))
		Expect(s).To(ContainSubstring("map"))
		Expect(s).To(ContainSubstring("point_lookup"))
	})

	It("marks pagination when present", func() {
		q := listTasksByStatusQuery()
		s := q.String()
		Expect(s).To(ContainSubstring("[paginated]"))
	})
})

var _ = Describe("EngineProfile", func() {
	It("reports supported ADTs and their complexity", func() {
		profile := metaengine.SQLiteEngineProfile()
		c, ok := profile.SupportsADT(metaengine.ADTMap)
		Expect(ok).To(BeTrue())
		Expect(c).To(Equal(metaengine.ComplexityOLogN))
	})

	It("returns false for unsupported ADTs", func() {
		profile := metaengine.NewMemoryEngine().Profile()
		_, ok := profile.SupportsADT("nonexistent")
		Expect(ok).To(BeFalse())
	})

	It("formats a readable string", func() {
		profile := metaengine.NewMemoryEngine().Profile()
		s := profile.String()
		Expect(s).To(ContainSubstring("memory"))
		Expect(s).To(ContainSubstring("map"))
	})
})

var _ = Describe("Diagnostics", func() {
	Describe("HasWarnings", func() {
		It("returns false for an empty diagnostics list", func() {
			Expect(metaengine.Diagnostics{}.HasWarnings()).To(BeFalse())
		})

		It("returns true when a warning is present", func() {
			diags := metaengine.Diagnostics{
				{Level: metaengine.DiagLevelWarn, Query: "q1", Message: "write amp"},
			}
			Expect(diags.HasWarnings()).To(BeTrue())
		})

		It("returns true when a degraded warning is present", func() {
			diags := metaengine.Diagnostics{
				{Level: metaengine.DiagLevelDegraded, Query: "q1", Message: "degraded"},
			}
			Expect(diags.HasWarnings()).To(BeTrue())
		})

		It("returns false for non-warning diagnostics only", func() {
			diags := metaengine.Diagnostics{
				{Level: "INFO", Query: "q1", Message: "all good"},
			}
			Expect(diags.HasWarnings()).To(BeFalse())
		})
	})

	Describe("Diagnostic String", func() {
		It("formats with level, query, and message", func() {
			d := metaengine.Diagnostic{
				Level: metaengine.DiagLevelWarn, Query: "my_query", Message: "something",
			}
			Expect(d.String()).To(ContainSubstring("[WARN]"))
			Expect(d.String()).To(ContainSubstring("my_query"))
			Expect(d.String()).To(ContainSubstring("something"))
		})
	})
})
