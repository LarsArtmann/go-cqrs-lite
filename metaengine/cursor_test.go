package metaengine_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
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
			Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
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

var _ = Describe("Cursor.Encode", func() {
	It("returns an empty string and nil error for a nil value", func() {
		s, err := metaengine.Cursor{Value: nil}.Encode()
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(BeEmpty())
	})

	It("returns a non-empty string and nil error for a marshable value", func() {
		s, err := metaengine.Cursor{Value: "task-123"}.Encode()
		Expect(err).NotTo(HaveOccurred())
		Expect(s).NotTo(BeEmpty())
	})

	It("surfaces a marshal error for an unmarshable value", func() {
		// A channel cannot be JSON-marshaled. Encode must surface the error
		// rather than silently returning "" (which ParseCursor treats as
		// "start of stream", silently resetting pagination).
		_, err := metaengine.Cursor{Value: make(chan int)}.Encode()
		Expect(err).To(HaveOccurred())
	})

	It("surfaces a marshal error for a func value", func() {
		// Funcs hit the same json.UnsupportedTypeError as channels.
		_, err := metaengine.Cursor{Value: func() {}}.Encode()
		Expect(err).To(HaveOccurred())
	})

	It("String swallows the same error that Encode surfaces", func() {
		// Documents the divergence: String returns "" (no error), Encode returns
		// the error. Callers crossing a process boundary MUST use Encode.
		Expect(metaengine.Cursor{Value: make(chan int)}.String()).To(BeEmpty())
	})
})

var _ = Describe("Cursor structured-value round-trip", func() {
	// Structured values cross the JSON boundary as their generic decoded forms:
	// structs and maps become map[string]any; slices become []any. These specs
	// lock that contract so callers know exactly what ParseCursor yields.
	It("round-trips a struct as map[string]any", func() {
		original := metaengine.Cursor{Value: FindTaskResult{ID: "t1", Title: "T", Status: "open"}}
		parsed, err := metaengine.ParseCursor(original.String())
		Expect(err).NotTo(HaveOccurred())
		asMap, ok := parsed.Value.(map[string]any)
		Expect(ok).To(BeTrue(), "struct must decode to map[string]any")
		Expect(asMap["ID"]).To(Equal("t1"))
		Expect(asMap["Title"]).To(Equal("T"))
	})

	It("round-trips a slice as []any", func() {
		original := metaengine.Cursor{Value: []string{"a", "b", "c"}}
		parsed, err := metaengine.ParseCursor(original.String())
		Expect(err).NotTo(HaveOccurred())
		asSlice, ok := parsed.Value.([]any)
		Expect(ok).To(BeTrue(), "slice must decode to []any")
		Expect(asSlice).To(HaveLen(3))
		Expect(asSlice[0]).To(Equal("a"))
	})

	It("round-trips a map as map[string]any", func() {
		original := metaengine.Cursor{Value: map[string]int{"x": 1, "y": 2}}
		parsed, err := metaengine.ParseCursor(original.String())
		Expect(err).NotTo(HaveOccurred())
		asMap, ok := parsed.Value.(map[string]any)
		Expect(ok).To(BeTrue(), "map must decode to map[string]any")
		Expect(asMap["x"]).To(Equal(float64(1)))
		Expect(asMap["y"]).To(Equal(float64(2)))
	})
})
