package metaengine_test

import (
	"context"
	"database/sql"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// pushdown_test.go validates that the SQLite PushdownScan implementation
// correctly pushes WHERE/ORDER BY/LIMIT into SQL via json_extract(),
// and that pushdown results are identical to closure-based MapScan results.

var _ = Describe("PushdownScan", func() {
	var (
		ctx      context.Context
		db       *sql.DB
		eng      metaengine.Engine
		engClose func()
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, err = sql.Open("sqlite", ":memory:")
		Expect(err).NotTo(HaveOccurred())

		eng, err = metaengine.NewSQLiteEngine(db)
		Expect(err).NotTo(HaveOccurred())

		engClose = func() {
			_ = eng.Close()
			_ = db.Close()
		}
	})

	AfterEach(func() {
		engClose()
	})

	Describe("PushdownMapScan basic operations", func() {
		BeforeEach(func() {
			// Populate test data.
			mb := eng.(metaengine.MapBackend)
			items := []struct {
				key   string
				value map[string]any
			}{
				{"t1", map[string]any{"ID": "t1", "Status": "open", "Priority": float64(3)}},
				{"t2", map[string]any{"ID": "t2", "Status": "open", "Priority": float64(1)}},
				{"t3", map[string]any{"ID": "t3", "Status": "done", "Priority": float64(2)}},
				{"t4", map[string]any{"ID": "t4", "Status": "open", "Priority": float64(5)}},
				{"t5", map[string]any{"ID": "t5", "Status": "done", "Priority": float64(4)}},
			}
			for _, item := range items {
				Expect(mb.MapSet(ctx, "tasks", item.key, item.value)).To(Succeed())
			}
		})

		It("filters with WHERE json_extract", func() {
			ps := eng.(metaengine.PushdownScan)
			results, err := ps.PushdownMapScan(
				ctx, "tasks",
				[]metaengine.FilterSpec{
					{Column: "Status", Op: metaengine.FilterEq, Value: "open"},
				},
				nil, nil, 0,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.Items).To(HaveLen(3))
		})

		It("sorts with ORDER BY json_extract", func() {
			ps := eng.(metaengine.PushdownScan)
			desc := false
			results, err := ps.PushdownMapScan(
				ctx, "tasks",
				nil,
				&metaengine.SortSpec{Column: "Priority", Desc: desc},
				nil, 0,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.Items).To(HaveLen(5))

			// Verify ascending order by Priority.
			vals := extractPriorities(results.Items)
			Expect(vals).To(Equal([]float64{1, 2, 3, 4, 5}))
		})

		It("sorts DESC with ORDER BY json_extract DESC", func() {
			ps := eng.(metaengine.PushdownScan)
			results, err := ps.PushdownMapScan(
				ctx, "tasks",
				nil,
				&metaengine.SortSpec{Column: "Priority", Desc: true},
				nil, 0,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.Items).To(HaveLen(5))

			vals := extractPriorities(results.Items)
			Expect(vals).To(Equal([]float64{5, 4, 3, 2, 1}))
		})

		It("limits with LIMIT", func() {
			ps := eng.(metaengine.PushdownScan)
			results, err := ps.PushdownMapScan(
				ctx, "tasks",
				nil,
				&metaengine.SortSpec{Column: "Priority", Desc: true},
				nil, 2,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.Items).To(HaveLen(2))
		})

		It("combines filter + sort + limit", func() {
			ps := eng.(metaengine.PushdownScan)
			results, err := ps.PushdownMapScan(
				ctx, "tasks",
				[]metaengine.FilterSpec{
					{Column: "Status", Op: metaengine.FilterEq, Value: "open"},
				},
				&metaengine.SortSpec{Column: "Priority", Desc: true},
				nil, 2,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.Items).To(HaveLen(2))

			vals := extractPriorities(results.Items)
			Expect(vals).To(Equal([]float64{5, 3}))
		})

		It("applies keyset cursor for ascending sort", func() {
			ps := eng.(metaengine.PushdownScan)
			// First page: top 2 by Priority ascending.
			page1, err := ps.PushdownMapScan(
				ctx, "tasks",
				nil,
				&metaengine.SortSpec{Column: "Priority", Desc: false},
				nil, 2,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(page1.Items).To(HaveLen(2))
			vals1 := extractPriorities(page1.Items)
			Expect(vals1).To(Equal([]float64{1, 2}))

			// Cursor = Priority=2 (last item on the page, excluding the +1).
			cursor := vals1[1] // Priority=2
			page2, err := ps.PushdownMapScan(
				ctx, "tasks",
				nil,
				&metaengine.SortSpec{Column: "Priority", Desc: false},
				cursor, 2,
			)
			Expect(err).NotTo(HaveOccurred())
			vals2 := extractPriorities(page2.Items)
			Expect(vals2).To(Equal([]float64{3, 4}))
		})

		It("applies keyset cursor for descending sort", func() {
			ps := eng.(metaengine.PushdownScan)
			// First page: top 2 by Priority descending.
			page1, err := ps.PushdownMapScan(
				ctx, "tasks",
				nil,
				&metaengine.SortSpec{Column: "Priority", Desc: true},
				nil, 2,
			)
			Expect(err).NotTo(HaveOccurred())
			vals1 := extractPriorities(page1.Items)
			Expect(vals1).To(Equal([]float64{5, 4}))

			// Cursor = Priority=4 (second item).
			cursor := vals1[1] // Priority=4
			page2, err := ps.PushdownMapScan(
				ctx, "tasks",
				nil,
				&metaengine.SortSpec{Column: "Priority", Desc: true},
				cursor, 2,
			)
			Expect(err).NotTo(HaveOccurred())
			vals2 := extractPriorities(page2.Items)
			Expect(vals2).To(Equal([]float64{3, 2}))
		})

		It("returns empty for non-matching filter", func() {
			ps := eng.(metaengine.PushdownScan)
			results, err := ps.PushdownMapScan(
				ctx, "tasks",
				[]metaengine.FilterSpec{
					{Column: "Status", Op: metaengine.FilterEq, Value: "nonexistent"},
				},
				nil, nil, 0,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.Items).To(BeEmpty())
		})

		It("supports inequality operators", func() {
			ps := eng.(metaengine.PushdownScan)
			results, err := ps.PushdownMapScan(
				ctx, "tasks",
				[]metaengine.FilterSpec{
					{Column: "Priority", Op: metaengine.FilterGt, Value: float64(2)},
				},
				&metaengine.SortSpec{Column: "Priority", Desc: false},
				nil, 0,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.Items).To(HaveLen(3))
			vals := extractPriorities(results.Items)
			Expect(vals).To(Equal([]float64{3, 4, 5}))
		})
	})

	Describe("Pushdown via Store API (FilterOnField + SortOnField)", func() {
		It("produces same results as closure-based filtering", func() {
			// Declarative query (pushdown). Auto-layout (LayoutPlanner) is
			// applied by Plan() — no need for manual NewPlannedSQLiteEngine.
			pdQuery := metaengine.Query[ListTasksByStatus, ListTasksByStatusResult](
				"pd_tasks",
				metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
					return e.ID, FindTaskResult{
						ID: e.ID, Title: e.Title, Assignee: e.Assignee,
						Status: e.Status, Priority: e.Priority,
					}
				}),
				metaengine.FilterOnField[FindTaskResult]("Status", metaengine.FilterEq),
				metaengine.SortOnField[FindTaskResult]("Priority", true),
			)

			pdStore, err := metaengine.Plan([]metaengine.Engine{eng}, pdQuery)
			Expect(err).NotTo(HaveOccurred())
			defer pdStore.Close()

			// Populate data through Apply (Plan → Apply → Execute flow).
			// Auto-layout routes writes to the planned table.
			tasks := []TaskCreated{
				{ID: "t1", Title: "A", Status: "open", Priority: 3},
				{ID: "t2", Title: "B", Status: "open", Priority: 1},
				{ID: "t3", Title: "C", Status: "done", Priority: 2},
				{ID: "t4", Title: "D", Status: "open", Priority: 5},
				{ID: "t5", Title: "E", Status: "done", Priority: 4},
			}
			for _, t := range tasks {
				Expect(pdStore.Apply(ctx, "TaskCreated", t)).To(Succeed())
			}

			pdResult, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
				ctx, pdStore, ListTasksByStatus{Status: "open", Limit: 10},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(pdResult.Tasks).To(HaveLen(3))

			// Verify descending priority order: 5, 3, 1.
			Expect(pdResult.Tasks[0].Priority).To(Equal(5))
			Expect(pdResult.Tasks[1].Priority).To(Equal(3))
			Expect(pdResult.Tasks[2].Priority).To(Equal(1))
		})
	})
})

func extractPriorities(results []any) []float64 {
	var vals []float64

	for _, r := range results {
		if m, ok := r.(map[string]any); ok {
			if p, ok := m["Priority"].(float64); ok {
				vals = append(vals, p)
			}
		}
	}

	return vals
}
