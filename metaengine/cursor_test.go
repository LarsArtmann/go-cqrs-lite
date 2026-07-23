package metaengine_test

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cursor serialization", func() {
	Describe("Cursor.String", func() {
		It("returns an empty string for a nil value", func() {
			Expect(metaengine.Cursor{Value: nil}.String()).To(BeEmpty())
		})

		It("returns a non-empty URL-safe string for a time.Time value", func() {
			c := metaengine.Cursor{Value: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
			s := c.String()
			Expect(s).NotTo(BeEmpty())
			Expect(s).To(MatchRegexp(`^[A-Za-z0-9_-]+$`))
		})

		It("returns a non-empty string for a string value", func() {
			c := metaengine.Cursor{Value: "task-123"}
			Expect(c.String()).NotTo(BeEmpty())
		})
	})

	Describe("ParseCursor", func() {
		It("returns nil cursor and nil error for an empty string", func() {
			c, err := metaengine.ParseCursor("")
			Expect(err).NotTo(HaveOccurred())
			Expect(c).To(BeNil())
		})

		It("returns an error for invalid base64", func() {
			_, err := metaengine.ParseCursor("!!!not-base64!!!")
			Expect(err).To(MatchError(MatchRegexp("invalid base64")))
		})
	})

	Describe("round-trip", func() {
		It("preserves string values", func() {
			original := metaengine.Cursor{Value: "task-abc-123"}
			parsed, err := metaengine.ParseCursor(original.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed.Value).To(Equal("task-abc-123"))
		})

		It("preserves bool values", func() {
			original := metaengine.Cursor{Value: true}
			parsed, err := metaengine.ParseCursor(original.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed.Value).To(Equal(true))
		})

		It("preserves numeric values as float64", func() {
			original := metaengine.Cursor{Value: 42}
			parsed, err := metaengine.ParseCursor(original.String())
			Expect(err).NotTo(HaveOccurred())
			// JSON round-trips numbers through any as float64
			Expect(parsed.Value).To(Equal(float64(42)))
		})

		It("preserves time.Time as ISO 8601 string for correct comparison", func() {
			t := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			original := metaengine.Cursor{Value: t}
			parsed, err := metaengine.ParseCursor(original.String())
			Expect(err).NotTo(HaveOccurred())

			// After round-trip, time.Time becomes string (ISO 8601).
			// Lexicographic comparison of ISO 8601 = chronological comparison.
			Expect(parsed.Value).To(BeAssignableToTypeOf(""))
			Expect(parsed.Value.(string)).To(Equal(t.Format(time.RFC3339Nano)))
		})
	})
})

var _ = Describe("Cursor-based pagination across HTTP boundary", func() {
	It("paginates correctly using serialized cursors", func() {
		store, err := metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()}, listTasksByStatusQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
		defer store.Close()

		base := time.Now()
		for i := range 10 {
			Expect(store.Apply("TaskCreated", TaskCreated{
				ID:       TaskID(string(rune('a' + i))),
				Title:    string(rune('A' + i)),
				Status:   "open",
				Priority: i,
				At:       base.Add(time.Duration(i) * time.Hour),
			})).To(Succeed())
		}

		// Page 1: no cursor
		page1, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
			context.Background(), store, ListTasksByStatus{Status: "open", Limit: 4},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(page1.Tasks).To(HaveLen(4))
		Expect(page1.Next).NotTo(BeNil())

		// Serialize cursor for HTTP transport
		cursorStr := page1.Next.String()
		Expect(cursorStr).NotTo(BeEmpty())

		// Deserialize cursor on the next request
		parsedCursor, err := metaengine.ParseCursor(cursorStr)
		Expect(err).NotTo(HaveOccurred())

		// Page 2: using the deserialized cursor
		page2, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
			context.Background(), store,
			ListTasksByStatus{Status: "open", Limit: 4, After: parsedCursor},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(page2.Tasks).To(HaveLen(4))

		// Verify no overlap between pages
		page1IDs := make(map[TaskID]bool)
		for _, t := range page1.Tasks {
			page1IDs[t.ID] = true
		}
		for _, t := range page2.Tasks {
			Expect(page1IDs[t.ID]).To(BeFalse(), "page 2 should not contain items from page 1")
		}
	})
})
