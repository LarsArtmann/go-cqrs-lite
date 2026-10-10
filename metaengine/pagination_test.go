package metaengine_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

var _ = Describe("Pagination", func() {
	var store *metaengine.Store
	var ctx context.Context

	BeforeEach(func() {
		var err error
		store, err = metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			listTasksByStatusQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()

		base := time.Now()
		for i := range 10 {
			Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID:       TaskID(fmt.Sprintf("t%d", i)),
				Title:    fmt.Sprintf("Task %d", i),
				Status:   "open",
				Priority: i,
				At:       base.Add(time.Duration(i) * time.Hour),
			})).To(Succeed())
		}
	})

	AfterEach(func() {
		Expect(store.Close()).To(Succeed())
	})

	Describe("requesting the first page", func() {
		It("returns the requested number of items", func() {
			result, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
				ctx, store, ListTasksByStatus{Status: "open", Limit: 3},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Tasks).To(HaveLen(3))
		})

		It("sets the Next cursor when more items exist", func() {
			result, _ := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
				ctx, store, ListTasksByStatus{Status: "open", Limit: 3},
			)
			Expect(result.Next).NotTo(BeNil())
		})
	})

	Describe("requesting the last page", func() {
		It("does not set the Next cursor when no more items", func() {
			result, _ := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
				ctx, store, ListTasksByStatus{Status: "open", Limit: 100},
			)
			Expect(result.Tasks).To(HaveLen(10))
			Expect(result.Next).To(BeNil())
		})
	})

	Describe("paginating through all items", func() {
		It("visits every item across multiple pages", func() {
			var seen []TaskID
			input := ListTasksByStatus{Status: "open", Limit: 4}

			for {
				result, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
					ctx,
					store,
					input,
				)
				Expect(err).NotTo(HaveOccurred())
				for _, t := range result.Tasks {
					seen = append(seen, t.ID)
				}

				if result.Next == nil {
					break
				}

				input.After = result.Next
			}

			Expect(seen).To(HaveLen(10))
		})
	})

	Describe("sort stability with equal keys", func() {
		BeforeEach(func() {
			// Add items with the same priority to test tiebreaking.
			for i := range 5 {
				Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
					ID:       TaskID(fmt.Sprintf("eq%d", i)),
					Title:    fmt.Sprintf("Equal %d", i),
					Status:   "open",
					Priority: 42,
					At:       time.Now(),
				})).To(Succeed())
			}
		})

		It("produces identical order on repeated scans", func() {
			var firstIDs []TaskID

			for run := range 5 {
				result, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
					ctx, store, ListTasksByStatus{Status: "open", Limit: 100},
				)
				Expect(err).NotTo(HaveOccurred())

				if run == 0 {
					firstIDs = make([]TaskID, len(result.Tasks))
					for i, t := range result.Tasks {
						firstIDs[i] = t.ID
					}
				} else {
					for i, t := range result.Tasks {
						Expect(t.ID).To(Equal(firstIDs[i]),
							"run %d: item %d changed order — sort is nondeterministic", run, i)
					}
				}
			}
		})
	})
})
