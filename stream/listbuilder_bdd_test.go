package stream_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/stream"
)

var _ = Describe("ListBuilder", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		store  *memory.MemoryStore
		reader *stream.InMemoryAggregateReader
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		store = memory.NewMemoryStore()
		reader = stream.NewInMemoryAggregateReader(store)
	})

	AfterEach(func() {
		cancel()
	})

	Describe("As a developer building aggregate listing queries", func() {
		DescribeTable(
			"PageSize is clamped to sensible bounds",
			func(pageSize uint) {
				seedStreamEvents(ctx, store)

				page, err := stream.NewListBuilder(reader).
					OfType("User").
					PageSize(pageSize).
					List(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).ToNot(BeEmpty())
			},
			Entry("zero uses default", uint(0)),
			Entry("exceeds max is clamped", uint(200)),
		)

		Context("when I list without filtering by type", func() {
			It("should return aggregates across all types", func() {
				seedStreamEvents(ctx, store)

				page, err := stream.NewListBuilder(reader).
					IncludeDeleted().
					PageSize(20).
					ListWithStatus(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(HaveLen(3))
			})
		})

		Context("when I set After cursor to the last item", func() {
			It("should return an empty page", func() {
				seedStreamEvents(ctx, store)

				allPage, err := stream.NewListBuilder(reader).
					OfType("User").
					IncludeDeleted().
					PageSize(20).
					List(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(allPage.Items).ToNot(BeEmpty())

				lastID := allPage.Items[len(allPage.Items)-1].ID
				page, err := stream.NewListBuilder(reader).
					OfType("User").
					IncludeDeleted().
					After(lastID).
					List(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(BeEmpty())
				Expect(page.HasMore).To(BeFalse())
			})
		})
	})
})

func seedStreamEvents(ctx context.Context, store *memory.MemoryStore) {
	activeID := id.NewAggregateID()
	activeEvt, err := event.NewEvent(
		"user.created", activeID, "User",
		event.Version(1), []byte(`{"name":"Alice"}`),
	)
	Expect(err).ToNot(HaveOccurred())
	Expect(store.Save(ctx, "User", activeID, []event.Event{activeEvt}, event.Version(0))).To(Succeed())

	deletedID := id.NewAggregateID()
	deletedEvt, err := event.NewEvent(
		"user.deleted", deletedID, "User",
		event.Version(1), []byte(`{"reason":"gdpr"}`),
		event.WithCustom(event.MetadataKeyTombstone, "true"),
	)
	Expect(err).ToNot(HaveOccurred())
	Expect(store.Save(ctx, "User", deletedID, []event.Event{deletedEvt}, event.Version(0))).To(Succeed())

	orderID := id.NewAggregateID()
	orderEvt, err := event.NewEvent(
		"order.created", orderID, "Order",
		event.Version(1), []byte(`{"total":99}`),
	)
	Expect(err).ToNot(HaveOccurred())
	Expect(store.Save(ctx, "Order", orderID, []event.Event{orderEvt}, event.Version(0))).To(Succeed())
}
