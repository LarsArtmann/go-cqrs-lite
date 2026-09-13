package metaengine_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

var _ = Describe("Re-layout trigger (ADR-0124 §11)", func() {
	Describe("ReplanLayout", func() {
		var (
			kvEngine *fakeEngine
			store    *metaengine.Store
		)

		BeforeEach(func() {
			kvEngine = &fakeEngine{profile: metaengine.EngineProfile{
				Name: "pebble",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutKV,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}

			var err error
			store, err = metaengine.Plan(
				[]metaengine.Engine{kvEngine},
				findTaskQuery(),
			)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			store.Close()
		})

		It("returns empty diffs when priority matches current layout (Balanced/Embed)", func() {
			diffs, err := store.ReplanLayout(context.Background(), &metaengine.PriorityConfig{
				Global: metaengine.PriorityBalanced,
			})
			Expect(err).NotTo(HaveOccurred())
			// Balanced on KV → Embed (current) → no diff
			Expect(diffs).To(BeEmpty())
		})

		It("returns diff when priority changes layout (WriteSpeed on KV → Normalize)", func() {
			diffs, err := store.ReplanLayout(context.Background(), &metaengine.PriorityConfig{
				Global: metaengine.PriorityWriteSpeed,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(diffs).To(HaveLen(1))
			Expect(diffs[0].QueryName).To(Equal("find_task"))
			Expect(diffs[0].From).To(Equal(metaengine.LayoutEmbed))
			Expect(diffs[0].To).To(Equal(metaengine.LayoutNormalize))
		})

		It("marks small projections as auto-rebuild", func() {
			diffs, err := store.ReplanLayout(context.Background(), &metaengine.PriorityConfig{
				Global: metaengine.PriorityWriteSpeed,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(diffs[0].AutoRebuild).To(BeTrue())
		})

		It("marks large projections as requiring confirmation", func() {
			// Create a query with a large volume
			largeQuery := metaengine.Query[FindTask, FindTaskResult](
				"large_find_task",
				metaengine.Volume(500_000), // > 100K threshold
				metaengine.OnRecord(
					TaskCreated{},
					func(_ record.Record, e TaskCreated) (TaskID, FindTaskResult) {
						return e.ID, FindTaskResult{ID: e.ID, Title: e.Title}
					},
				),
			)

			largeStore, err := metaengine.Plan(
				[]metaengine.Engine{kvEngine},
				largeQuery,
			)
			Expect(err).NotTo(HaveOccurred())
			defer largeStore.Close()

			diffs, err := largeStore.ReplanLayout(context.Background(), &metaengine.PriorityConfig{
				Global: metaengine.PriorityWriteSpeed,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(diffs).To(HaveLen(1))
			Expect(diffs[0].AutoRebuild).To(BeFalse())
		})

		It("returns empty diffs with nil priority config", func() {
			diffs, err := store.ReplanLayout(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			// nil config → Balanced → Embed (current) → no diff
			Expect(diffs).To(BeEmpty())
		})
	})

	Describe("RebuildThreshold", func() {
		It("DefaultRebuildThreshold returns sensible defaults", func() {
			t := metaengine.DefaultRebuildThreshold()
			Expect(t.MaxEventCount).To(Equal(int64(100_000)))
			Expect(t.MaxDataBytes).To(Equal(int64(1 << 30)))
		})
	})

	Describe("ConfirmRebuild", func() {
		It("does not error on empty diffs", func() {
			store, err := metaengine.Plan(
				[]metaengine.Engine{&fakeEngine{profile: metaengine.EngineProfile{
					Name: "test",
					Supports: map[metaengine.ADT]metaengine.Complexity{
						metaengine.ADTMap: metaengine.ComplexityO1,
					},
				}}},
				findTaskQuery(),
			)
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			err = store.ConfirmRebuild(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
