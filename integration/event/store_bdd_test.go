package event_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

var _ = Describe("Event Store", func() {
	var (
		ctx     context.Context
		store   *memory.MemoryStore
		aggID   id.AggregateID
		aggType event.AggregateType
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = memory.NewMemoryStore()
		aggID = id.NewAggregateID()
		aggType = event.AggregateType("TestAggregate")
	})

	Describe("As a developer building an event-sourced system", func() {
		Context("when I save events for a new aggregate", func() {
			It("should persist them with correct version tracking", func() {
				events := []event.Event{
					createTestEvent("TestCreated", aggID, 1, []byte(`{"name":"first"}`)),
				}

				err := store.Save(
					ctx,
					event.NewAggregateRef(aggType, aggID),
					events,
					event.Version(0),
				)
				Expect(err).ToNot(HaveOccurred())

				loaded, err := store.Load(ctx, event.NewAggregateRef(aggType, aggID))
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded).To(HaveLen(1))
				Expect(loaded[0].Type()).To(Equal(event.Type("TestCreated")))
				Expect(loaded[0].AggregateID()).To(Equal(aggID))
				Expect(loaded[0].Version()).To(Equal(event.Version(1)))
			})
		})

		Context("when I append more events to an existing aggregate", func() {
			It("should maintain event order and increment versions", func() {
				first := []event.Event{createTestEvent("TestCreated", aggID, 1, nil)}
				Expect(
					store.Save(ctx, event.NewAggregateRef(aggType, aggID), first, event.Version(0)),
				).To(Succeed())

				second := []event.Event{createTestEvent("TestUpdated", aggID, 2, nil)}
				Expect(
					store.Save(
						ctx,
						event.NewAggregateRef(aggType, aggID),
						second,
						event.Version(1),
					),
				).To(Succeed())

				loaded, err := store.Load(ctx, event.NewAggregateRef(aggType, aggID))
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded).To(HaveLen(2))
				Expect(loaded[0].Type()).To(Equal(event.Type("TestCreated")))
				Expect(loaded[1].Type()).To(Equal(event.Type("TestUpdated")))
				Expect(loaded[1].Version()).To(Equal(event.Version(2)))
			})
		})

		Context("when I save events with the wrong expected version", func() {
			It("should detect the version conflict and reject the save", func() {
				first := []event.Event{createTestEvent("TestCreated", aggID, 1, nil)}
				Expect(
					store.Save(ctx, event.NewAggregateRef(aggType, aggID), first, event.Version(0)),
				).To(Succeed())

				conflicting := []event.Event{createTestEvent("TestConflict", aggID, 2, nil)}
				err := store.Save(
					ctx,
					event.NewAggregateRef(aggType, aggID),
					conflicting,
					event.Version(0),
				)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("version conflict")))
			})
		})

		Context("when I load events for a non-existent aggregate", func() {
			It("should explain that the aggregate was not found", func() {
				_, err := store.Load(ctx, event.NewAggregateRef(aggType, id.NewAggregateID()))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("aggregate not found"))
			})
		})

		Context("when I load events starting from a specific version", func() {
			It("should return only events from that version onward", func() {
				events := []event.Event{
					createTestEvent("E1", aggID, 1, nil),
					createTestEvent("E2", aggID, 2, nil),
					createTestEvent("E3", aggID, 3, nil),
				}
				Expect(
					store.Save(
						ctx,
						event.NewAggregateRef(aggType, aggID),
						events,
						event.Version(0),
					),
				).To(Succeed())

				loaded, err := store.LoadFromVersion(
					ctx,
					event.NewAggregateRef(aggType, aggID),
					event.Version(2),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded).To(HaveLen(1))
				Expect(loaded[0].Type()).To(Equal(event.Type("E3")))
			})
		})

		Context("when I load events from a version beyond the current state", func() {
			It("should return an empty slice without error", func() {
				events := []event.Event{createTestEvent("E1", aggID, 1, nil)}
				Expect(
					store.Save(
						ctx,
						event.NewAggregateRef(aggType, aggID),
						events,
						event.Version(0),
					),
				).To(Succeed())

				loaded, err := store.LoadFromVersion(
					ctx,
					event.NewAggregateRef(aggType, aggID),
					event.Version(99),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded).To(BeEmpty())
			})
		})

		Context("when I use AppendBatch for bulk imports", func() {
			It(
				"should append all events without version checks and preserve versions on load",
				func() {
					events := []event.Event{
						createTestEvent("BatchEvent1", aggID, 1, nil),
						createTestEvent("BatchEvent2", aggID, 2, nil),
						createTestEvent("BatchEvent3", aggID, 3, nil),
					}

					Expect(
						store.AppendBatch(ctx, event.NewAggregateRef(aggType, aggID), events),
					).To(Succeed())

					loaded, err := store.Load(ctx, event.NewAggregateRef(aggType, aggID))
					Expect(err).ToNot(HaveOccurred())
					Expect(loaded).To(HaveLen(3))
					Expect(loaded[0].Version()).To(Equal(event.Version(1)))
					Expect(loaded[1].Version()).To(Equal(event.Version(2)))
					Expect(loaded[2].Version()).To(Equal(event.Version(3)))
					Expect(loaded[0].Type()).To(Equal(event.Type("BatchEvent1")))
					Expect(loaded[2].Type()).To(Equal(event.Type("BatchEvent3")))
				},
			)
		})

		Context("when the store is closed", func() {
			It("should reject all further operations", func() {
				Expect(store.Close()).To(Succeed())

				err := store.Save(
					ctx,
					event.NewAggregateRef(aggType, id.NewAggregateID()),
					nil,
					event.Version(0),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("store is closed"))
			})
		})
	})
})
