package memory_test

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func makeMemEvent(eventType event.Type, aggID id.AggregateID, version event.Version) event.Event {
	evt, err := event.NewEvent(eventType, aggID, "TestAggregate", version, []byte(`{}`))
	Expect(err).ToNot(HaveOccurred())

	return evt
}

var _ = Describe("MemoryStore", func() {
	var (
		ctx   context.Context
		store *memory.MemoryStore
		aggID id.AggregateID
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = memory.NewMemoryStore()
		aggID = id.NewAggregateID()
	})

	Describe("As a developer using the in-memory event store", func() {
		Context("when I save events for a new aggregate", func() {
			It("should persist them and allow loading", func() {
				events := []event.Event{makeMemEvent("Created", aggID, 1)}
				err := store.Save(ctx, "TestAggregate", aggID, events, 0)
				Expect(err).ToNot(HaveOccurred())

				loaded, err := store.Load(ctx, "TestAggregate", aggID)
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded).To(HaveLen(1))
				Expect(loaded[0].Type()).To(Equal(event.Type("Created")))
			})
		})

		Context("when I save events with the wrong expected version", func() {
			It("should return an error", func() {
				events := []event.Event{makeMemEvent("Created", aggID, 1)}
				err := store.Save(ctx, "TestAggregate", aggID, events, 0)
				Expect(err).ToNot(HaveOccurred())

				more := []event.Event{makeMemEvent("Updated", aggID, 2)}
				err = store.Save(ctx, "TestAggregate", aggID, more, 0) // wrong expected version
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when I load a non-existent aggregate", func() {
			It("should return ErrAggregateNotFound", func() {
				_, err := store.Load(ctx, "TestAggregate", aggID)
				Expect(err).To(MatchError(event.ErrAggregateNotFound))
			})
		})

		Context("when I append batch events", func() {
			It("should add them after existing events", func() {
				initial := []event.Event{makeMemEvent("Created", aggID, 1)}
				Expect(store.Save(ctx, "TestAggregate", aggID, initial, 0)).To(Succeed())

				batch := []event.Event{
					makeMemEvent("Updated", aggID, 2),
					makeMemEvent("Updated", aggID, 3),
				}
				Expect(store.AppendBatch(ctx, "TestAggregate", aggID, batch)).To(Succeed())

				loaded, err := store.Load(ctx, "TestAggregate", aggID)
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded).To(HaveLen(3))
			})
		})

		Context("when I load events from a specific version", func() {
			It("should return only events from that version onward", func() {
				events := []event.Event{
					makeMemEvent("Created", aggID, 1),
					makeMemEvent("Updated", aggID, 2),
					makeMemEvent("Updated", aggID, 3),
				}
				Expect(store.Save(ctx, "TestAggregate", aggID, events, 0)).To(Succeed())

				// LoadFromVersion(v) returns events from index v onward
				// Version(2) → index 2 → only version 3 event
				fromV2, err := store.LoadFromVersion(ctx, "TestAggregate", aggID, 2)
				Expect(err).ToNot(HaveOccurred())
				Expect(fromV2).To(HaveLen(1))
				Expect(fromV2[0].Version()).To(Equal(event.Version(3)))

				// Version(1) → index 1 → versions 2 and 3
				fromV1, err := store.LoadFromVersion(ctx, "TestAggregate", aggID, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(fromV1).To(HaveLen(2))
			})
		})

		Context("when I delete an aggregate", func() {
			It("should remove all its events", func() {
				events := []event.Event{makeMemEvent("Created", aggID, 1)}
				Expect(store.Save(ctx, "TestAggregate", aggID, events, 0)).To(Succeed())

				Expect(store.Delete(ctx, "TestAggregate", aggID)).To(Succeed())

				_, err := store.Load(ctx, "TestAggregate", aggID)
				Expect(err).To(MatchError(event.ErrAggregateNotFound))
			})
		})

		Context("when I close the store", func() {
			It("should reject further operations", func() {
				Expect(store.Close()).To(Succeed())

				err := store.Save(ctx, "TestAggregate", aggID,
					[]event.Event{makeMemEvent("Created", aggID, 1)}, 0)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

var _ = Describe("MemoryBus", func() {
	var (
		ctx context.Context
		bus *memory.MemoryBus
	)

	BeforeEach(func() {
		ctx = context.Background()
		bus = memory.NewMemoryBus()
	})

	Describe("As a developer using the in-memory event bus", func() {
		Context("when I publish events", func() {
			It("should deliver them to type-specific subscribers", func() {
				var received []event.Event

				err := bus.Subscribe("UserCreated", func(_ context.Context, evt event.Event) error {
					received = append(received, evt)

					return nil
				})
				Expect(err).ToNot(HaveOccurred())

				evt := makeMemEvent("UserCreated", id.NewAggregateID(), 1)
				Expect(bus.Publish(ctx, evt)).To(Succeed())

				Expect(received).To(HaveLen(1))
				Expect(received[0].ID()).To(Equal(evt.ID()))
			})
		})

		Context("when I subscribe to all events", func() {
			It("should receive events of any type", func() {
				var received []event.Event

				err := bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
					received = append(received, evt)

					return nil
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(
					bus.Publish(ctx, makeMemEvent("UserCreated", id.NewAggregateID(), 1)),
				).To(Succeed())
				Expect(
					bus.Publish(ctx, makeMemEvent("OrderPlaced", id.NewAggregateID(), 1)),
				).To(Succeed())

				Expect(received).To(HaveLen(2))
			})
		})

		Context("when I close the bus", func() {
			It("should reject further operations", func() {
				Expect(bus.Close()).To(Succeed())

				err := bus.Publish(ctx, makeMemEvent("UserCreated", id.NewAggregateID(), 1))
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
