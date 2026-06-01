package event_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/event/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

var _ = Describe("Event Bus", func() {
	var (
		ctx      context.Context
		bus      *memory.MemoryBus
		received []event.Event
	)

	BeforeEach(func() {
		ctx = context.Background()
		bus = memory.NewMemoryBus()
		received = nil
	})

	Describe("As a developer reacting to domain events", func() {
		Context("when I subscribe to a specific event type", func() {
			It("should receive only events of that type", func() {
				Expect(subscribeOrderPlaced(bus, &received)).To(Succeed())

				aggID := id.NewAggregateID()
				evt := createTestEvent("OrderPlaced", aggID, 1, nil)
				Expect(bus.Publish(ctx, evt)).To(Succeed())
				Expect(received).To(HaveLen(1))
				Expect(received[0].Type()).To(Equal(event.Type("OrderPlaced")))
			})
		})

		Context("when I subscribe to all events", func() {
			It("should receive every published event regardless of type", func() {
				Expect(bus.SubscribeAll(eventtest.AppendEventsHandler(&received))).To(Succeed())

				aggID := id.NewAggregateID()
				evt1 := createTestEvent("UserCreated", aggID, 1, nil)
				evt2 := createTestEvent("UserUpdated", aggID, 2, nil)
				Expect(bus.Publish(ctx, evt1, evt2)).To(Succeed())
				Expect(received).To(HaveLen(2))
			})
		})

		Context("when I use middleware on the bus", func() {
			It("should wrap handlers in the correct order", func() {
				var callOrder []string

				Expect(
					bus.Use(eventtest.EventMiddleware(&callOrder, "middleware1")),
				).To(Succeed())
				Expect(
					bus.Use(eventtest.EventMiddleware(&callOrder, "middleware2")),
				).To(Succeed())

				Expect(bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
					callOrder = append(callOrder, "handler")
					received = append(received, evt)

					return nil
				})).To(Succeed())

				aggID := id.NewAggregateID()
				Expect(bus.Publish(ctx, createTestEvent("TestEvent", aggID, 1, nil))).To(Succeed())

				Expect(callOrder).To(Equal([]string{"middleware1", "middleware2", "handler"}))
				Expect(received).To(HaveLen(1))
			})
		})

		Context("when a handler fails during publish", func() {
			It("should stop processing and return the wrapped error", func() {
				Expect(
					bus.Subscribe(
						event.Type("BadEvent"),
						func(_ context.Context, _ event.Event) error {
							return context.DeadlineExceeded
						},
					),
				).To(Succeed())

				aggID := id.NewAggregateID()
				evt := createTestEvent("BadEvent", aggID, 1, nil)
				err := bus.Publish(ctx, evt)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("handler 0 failed"))
			})
		})

		Context("when I publish to a closed bus", func() {
			It("should explain that the bus is closed", func() {
				Expect(bus.Close()).To(Succeed())

				aggID := id.NewAggregateID()
				err := bus.Publish(ctx, createTestEvent("Test", aggID, 1, nil))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bus is closed"))
			})
		})

		Context("when I subscribe to a closed bus", func() {
			It("should explain that the bus is closed", func() {
				Expect(bus.Close()).To(Succeed())

				err := bus.Subscribe(
					event.Type("Test"),
					eventtest.NoopEventHandler(),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bus is closed"))
			})
		})

		Context("when I subscribe to a specific type AND subscribe to all events", func() {
			It("should deliver the event twice — once per subscription", func() {
				Expect(subscribeOrderPlaced(bus, &received)).To(Succeed())

				Expect(bus.SubscribeAll(eventtest.AppendEventsHandler(&received))).To(Succeed())

				aggID := id.NewAggregateID()
				evt := createTestEvent("OrderPlaced", aggID, 1, nil)
				Expect(bus.Publish(ctx, evt)).To(Succeed())

				Expect(received).To(HaveLen(2))
				Expect(received[0].ID()).To(Equal(received[1].ID()))
				Expect(received[0].Type()).To(Equal(event.Type("OrderPlaced")))
			})
		})

		Context("when I subscribe to a specific type but publish a different type", func() {
			It("should not receive the event", func() {
				Expect(subscribeOrderPlaced(bus, &received)).To(Succeed())

				aggID := id.NewAggregateID()
				evt := createTestEvent("OrderCancelled", aggID, 1, nil)
				Expect(bus.Publish(ctx, evt)).To(Succeed())

				Expect(received).To(BeEmpty())
			})
		})
	})
})
