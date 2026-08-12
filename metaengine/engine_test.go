package metaengine_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// fakeEngine implements metaengine.Engine (Profile + Close) but zero backend
// interfaces. It lets us test multi-engine cost-based selection, DEGRADED
// diagnostics, and error branches where an engine's Profile declares ADT
// support that its runtime type does not back.
type fakeEngine struct {
	profile  metaengine.EngineProfile
	closeErr error
}

func (e *fakeEngine) Profile() metaengine.EngineProfile { return e.profile }
func (e *fakeEngine) Close() error                      { return e.closeErr }

var _ metaengine.Engine = (*fakeEngine)(nil)

// ── Local query types for ADTs not covered by shared fixtures ──

type mmInput struct{ Assignee UserID }

type mmResult struct{ Tasks []TaskID }

func multimapQuery() metaengine.QueryDecl[mmInput, mmResult] {
	return metaengine.Query[mmInput, mmResult](
		"mm_tasks",
		metaengine.On(TaskAssigned{}, func(e TaskAssigned) metaengine.MultiEntry {
			return metaengine.MultiEntry{Key: e.Assignee, Value: e.TaskID}
		}),
	)
}

type logInput struct{}

type logResult struct{ Entries []TaskCreated }

func logQuery() metaengine.QueryDecl[logInput, logResult] {
	return metaengine.Query[logInput, logResult](
		"task_log",
		metaengine.On(TaskCreated{}, func(e TaskCreated) metaengine.Append {
			return metaengine.Append{Value: e}
		}),
	)
}

// ── Multi-engine cost-based selection ──

var _ = Describe("Multi-engine selection", func() {
	When("two engines support the same ADT at different costs", func() {
		It("assigns the query to the cheaper engine", func() {
			fast := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "fast-hash",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}
			slow := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "slow-scan",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityON,
				},
			}}

			store, err := metaengine.Plan(
				[]metaengine.Engine{slow, fast}, // slow first — ensures sort, not first-match
				findTaskQuery(),
			)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			Expect(store.Plan().Queries[0].EngineName).To(Equal("fast-hash"))
		})
	})

	// Both engines downgrade to O(N) for filtered scans (effectiveReadComplexity),
	// so estimated costs are identical. The tiebreaker is complexityRank: the
	// engine with the lower raw complexity (O(1) < O(N)) wins.
	When("two engines have equal cost but different raw complexity", func() {
		It("prefers the lower-complexity engine as tiebreaker", func() {
			o1 := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "o1-engine",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}
			oN := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "on-engine",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityON,
				},
			}}

			store, err := metaengine.Plan(
				[]metaengine.Engine{oN, o1}, // oN first — ensures tiebreaker fires
				listTasksByStatusQuery(),
			)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			Expect(store.Plan().Queries[0].EngineName).To(Equal("o1-engine"))
		})
	})

	When("no engine supports the required ADT", func() {
		It("returns an error naming the query and ADT", func() {
			onlyMap := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "map-only",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}

			_, err := metaengine.Plan(
				[]metaengine.Engine{onlyMap},
				tasksByAssigneeQuery(), // Graph ADT
			)
			Expect(err).To(MatchError(MatchRegexp(`requires ADT graph but no engine supports it`)))
		})
	})
})

// ── DEGRADED diagnostics ──

var _ = Describe("DEGRADED diagnostics", func() {
	When("a graph query is served by an O(N) scan engine", func() {
		It("warns that graph traversal degrades to a full scan", func() {
			scanEngine := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "scan-only",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTGraph: metaengine.ComplexityON,
				},
			}}

			store, err := metaengine.Plan(
				[]metaengine.Engine{scanEngine},
				tasksByAssigneeQuery(),
			)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			Expect(hasDiagnostic(
				store.Plan().Queries[0].Diagnostics,
				metaengine.DiagLevelDegraded,
				"graph traversal via scan",
			)).To(BeTrue())
		})
	})

	When("a paginated query runs on an O(N) engine", func() {
		It("warns that pagination degrades to an in-memory scan", func() {
			scanEngine := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "scan-engine",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityON,
				},
			}}

			store, err := metaengine.Plan(
				[]metaengine.Engine{scanEngine},
				listTasksByStatusQuery(),
			)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			Expect(hasDiagnostic(
				store.Plan().Queries[0].Diagnostics,
				metaengine.DiagLevelDegraded,
				"filtered scan via in-memory",
			)).To(BeTrue())
		})
	})
})

// ── Store.Close error aggregation ──

var _ = Describe("Store.Close error aggregation", func() {
	When("an engine returns an error on close", func() {
		It("returns the first error encountered", func() {
			failing := &fakeEngine{
				profile: metaengine.EngineProfile{
					Name: "failing",
					Supports: map[metaengine.ADT]metaengine.Complexity{
						metaengine.ADTMap: metaengine.ComplexityO1,
					},
				},
				closeErr: errors.New("disk flush failed"),
			}

			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine(), failing},
				findTaskQuery(),
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(store.Close()).To(MatchError("disk flush failed"))
		})
	})
})

// ── Backend capability errors ──
// When an engine's Profile declares ADT support but the engine does not
// implement the corresponding backend interface, Apply and Execute return
// a clear error naming the missing capability.

var _ = Describe("Backend capability errors", func() {
	allADT := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "profile-only",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:      metaengine.ComplexityO1,
			metaengine.ADTSet:      metaengine.ComplexityO1,
			metaengine.ADTCounter:  metaengine.ComplexityO1,
			metaengine.ADTGraph:    metaengine.ComplexityO1,
			metaengine.ADTMultimap: metaengine.ComplexityO1,
			metaengine.ADTLog:      metaengine.ComplexityO1,
		},
	}}

	DescribeTable(
		"Apply returns a clear error when the engine lacks the write backend",
		func(query any, eventType string, payload any, expectedErr string) {
			store, err := metaengine.Plan([]metaengine.Engine{allADT}, query)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			err = store.Apply(context.Background(), eventType, payload)
			Expect(err).To(MatchError(MatchRegexp(expectedErr)))
		},
		Entry("Map insert",
			findTaskQuery(), "TaskCreated",
			TaskCreated{ID: "t1", Title: "T", Status: "open", At: time.Now()},
			"does not support Map operations"),
		Entry("Set add",
			checkAssigneeQuery(), "TaskAssigned",
			TaskAssigned{TaskID: "t1", Assignee: "alice", At: time.Now()},
			"does not support Set operations"),
		Entry("Counter increment",
			countByStatusQuery(), "TaskCreated",
			TaskCreated{ID: "t1", Title: "T", Status: "open", At: time.Now()},
			"does not support Counter operations"),
		Entry("Graph edge",
			tasksByAssigneeQuery(), "TaskAssigned",
			TaskAssigned{TaskID: "t1", Assignee: "alice", At: time.Now()},
			"does not support Graph operations"),
		Entry("Multimap insert",
			multimapQuery(), "TaskAssigned",
			TaskAssigned{TaskID: "t1", Assignee: "alice", At: time.Now()},
			"does not support Multimap operations"),
		Entry("Log append",
			logQuery(), "TaskCreated",
			TaskCreated{ID: "t1", Title: "T", Status: "open", At: time.Now()},
			"does not support Log operations"),
	)

	DescribeTable(
		"Execute returns a clear error when the engine lacks the read backend",
		func(query any, input any, expectedErr string) {
			store, err := metaengine.Plan([]metaengine.Engine{allADT}, query)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			_, err = store.Execute(input)
			Expect(err).To(MatchError(MatchRegexp(expectedErr)))
		},
		Entry("Map point lookup",
			findTaskQuery(), FindTask{ID: "t1"},
			"does not support Map reads"),
		Entry("Set membership",
			checkAssigneeQuery(), CheckAssignee{User: "alice"},
			"does not support Set reads"),
		Entry("Counter aggregate",
			countByStatusQuery(), CountByStatus{},
			"does not support Counter reads"),
		Entry("Graph traversal",
			tasksByAssigneeQuery(), TasksByAssignee{User: "alice", Depth: 1},
			"does not support Graph reads"),
		Entry("Multimap lookup",
			multimapQuery(), mmInput{Assignee: "alice"},
			"does not support Multimap reads"),
		Entry("Log tail",
			logQuery(), logInput{},
			"does not support Log reads"),
	)
})
