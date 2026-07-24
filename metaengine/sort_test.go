package metaengine_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

var _ = Describe("Sorting by non-numeric keys", func() {
	Describe("string sort key", func() {
		type sEvent struct{ ID, Name string }
		type sItem struct{ Name string }
		type sInput struct{ Limit int }
		type sResult struct {
			Items []sItem
			Next  *metaengine.Cursor
		}

		var store *metaengine.Store

		BeforeEach(func() {
			q := metaengine.Query[sInput, sResult](
				"by_name",
				metaengine.On(sEvent{}, func(e sEvent) (string, sItem) {
					return e.ID, sItem{Name: e.Name}
				}),
				metaengine.SortOn(func(r sItem) string { return r.Name }),
			)
			var err error
			store, err = metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, q)
			Expect(err).NotTo(HaveOccurred())

			Expect(
				store.Apply(context.Background(), "sEvent", sEvent{ID: "1", Name: "zebra"}),
			).To(Succeed())
			Expect(
				store.Apply(context.Background(), "sEvent", sEvent{ID: "2", Name: "apple"}),
			).To(Succeed())
			Expect(
				store.Apply(context.Background(), "sEvent", sEvent{ID: "3", Name: "mango"}),
			).To(Succeed())
		})

		AfterEach(func() {
			Expect(store.Close()).To(Succeed())
		})

		It("sorts items alphabetically by the string key", func() {
			result, err := metaengine.ExecuteTyped[sInput, sResult](
				context.Background(), store, sInput{Limit: 10},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Items).To(HaveLen(3))
			Expect(result.Items[0].Name).To(Equal("apple"))
			Expect(result.Items[1].Name).To(Equal("mango"))
			Expect(result.Items[2].Name).To(Equal("zebra"))
		})
	})

	Describe("time.Time sort key", func() {
		type tEvent struct {
			ID string
			At time.Time
		}
		type tItem struct{ At time.Time }
		type tInput struct{ Limit int }
		type tResult struct {
			Items []tItem
			Next  *metaengine.Cursor
		}

		var store *metaengine.Store

		BeforeEach(func() {
			q := metaengine.Query[tInput, tResult](
				"by_time",
				metaengine.On(tEvent{}, func(e tEvent) (string, tItem) {
					return e.ID, tItem{At: e.At}
				}),
				metaengine.SortOn(func(r tItem) time.Time { return r.At }),
			)
			var err error
			store, err = metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, q)
			Expect(err).NotTo(HaveOccurred())

			base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			Expect(
				store.Apply(
					context.Background(),
					"tEvent",
					tEvent{ID: "1", At: base.Add(2 * time.Hour)},
				),
			).To(Succeed())
			Expect(
				store.Apply(context.Background(), "tEvent", tEvent{ID: "2", At: base}),
			).To(Succeed())
			Expect(
				store.Apply(
					context.Background(),
					"tEvent",
					tEvent{ID: "3", At: base.Add(1 * time.Hour)},
				),
			).To(Succeed())
		})

		AfterEach(func() {
			Expect(store.Close()).To(Succeed())
		})

		It("sorts items chronologically by the time key", func() {
			result, err := metaengine.ExecuteTyped[tInput, tResult](
				context.Background(), store, tInput{Limit: 10},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Items).To(HaveLen(3))
			Expect(
				result.Items[0].At.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			).To(BeTrue())
			Expect(
				result.Items[1].At.Equal(time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)),
			).To(BeTrue())
			Expect(
				result.Items[2].At.Equal(time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)),
			).To(BeTrue())
		})
	})
})
