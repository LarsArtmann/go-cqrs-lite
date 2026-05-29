package event_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

// createTestEvent creates a test event with the given type and version.
func createTestEvent(
	eventType event.Type,
	aggID id.AggregateID,
	version event.Version,
	payload []byte,
) event.Event {
	if payload == nil {
		payload = []byte(`{"test":true}`)
	}

	evt, err := event.NewEvent(eventType, aggID, "TestAggregate", version, payload)
	Expect(err).ToNot(HaveOccurred())

	return evt
}

func expectNewEventValidationFails(
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	expectedMsg string,
) {
	_, err := event.NewEvent(event.Type("BadEvent"), aggID, aggType, version, nil)
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(expectedMsg))
}

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

				err := store.Save(ctx, aggType, aggID, events, event.Version(0))
				Expect(err).ToNot(HaveOccurred())

				loaded, err := store.Load(ctx, aggType, aggID)
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
				Expect(store.Save(ctx, aggType, aggID, first, event.Version(0))).To(Succeed())

				second := []event.Event{createTestEvent("TestUpdated", aggID, 2, nil)}
				Expect(store.Save(ctx, aggType, aggID, second, event.Version(1))).To(Succeed())

				loaded, err := store.Load(ctx, aggType, aggID)
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
				Expect(store.Save(ctx, aggType, aggID, first, event.Version(0))).To(Succeed())

				conflicting := []event.Event{createTestEvent("TestConflict", aggID, 2, nil)}
				err := store.Save(ctx, aggType, aggID, conflicting, event.Version(0))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("version conflict")))
			})
		})

		Context("when I load events for a non-existent aggregate", func() {
			It("should explain that the aggregate was not found", func() {
				_, err := store.Load(ctx, aggType, id.NewAggregateID())
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
				Expect(store.Save(ctx, aggType, aggID, events, event.Version(0))).To(Succeed())

				loaded, err := store.LoadFromVersion(ctx, aggType, aggID, event.Version(2))
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded).To(HaveLen(1))
				Expect(loaded[0].Type()).To(Equal(event.Type("E3")))
			})
		})

		Context("when I load events from a version beyond the current state", func() {
			It("should return an empty slice without error", func() {
				events := []event.Event{createTestEvent("E1", aggID, 1, nil)}
				Expect(store.Save(ctx, aggType, aggID, events, event.Version(0))).To(Succeed())

				loaded, err := store.LoadFromVersion(ctx, aggType, aggID, event.Version(99))
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

					Expect(store.AppendBatch(ctx, aggType, aggID, events)).To(Succeed())

					loaded, err := store.Load(ctx, aggType, aggID)
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

				err := store.Save(ctx, aggType, id.NewAggregateID(), nil, event.Version(0))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("store is closed"))
			})
		})
	})
})

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
				Expect(
					bus.Subscribe(
						event.Type("OrderPlaced"),
						testhelpers.AppendEventsHandler(&received),
					),
				).To(Succeed())

				aggID := id.NewAggregateID()
				evt := createTestEvent("OrderPlaced", aggID, 1, nil)
				Expect(bus.Publish(ctx, evt)).To(Succeed())
				Expect(received).To(HaveLen(1))
				Expect(received[0].Type()).To(Equal(event.Type("OrderPlaced")))
			})
		})

		Context("when I subscribe to all events", func() {
			It("should receive every published event regardless of type", func() {
				Expect(bus.SubscribeAll(testhelpers.AppendEventsHandler(&received))).To(Succeed())

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
					bus.Use(testhelpers.EventMiddleware(&callOrder, "middleware1")),
				).To(Succeed())
				Expect(
					bus.Use(testhelpers.EventMiddleware(&callOrder, "middleware2")),
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
					testhelpers.NoopEventHandler(),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bus is closed"))
			})
		})

		Context("when I subscribe to a specific type AND subscribe to all events", func() {
			It("should deliver the event twice — once per subscription", func() {
				Expect(
					bus.Subscribe(
						event.Type("OrderPlaced"),
						testhelpers.AppendEventsHandler(&received),
					),
				).To(Succeed())

				Expect(bus.SubscribeAll(testhelpers.AppendEventsHandler(&received))).To(Succeed())

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
				Expect(
					bus.Subscribe(
						event.Type("OrderPlaced"),
						testhelpers.AppendEventsHandler(&received),
					),
				).To(Succeed())

				aggID := id.NewAggregateID()
				evt := createTestEvent("OrderCancelled", aggID, 1, nil)
				Expect(bus.Publish(ctx, evt)).To(Succeed())

				Expect(received).To(BeEmpty())
			})
		})
	})
})

var _ = Describe("Event Creation", func() {
	Describe("As a developer creating domain events", func() {
		Context("when I create an event with all metadata", func() {
			It("should preserve every field including tracing IDs", func() {
				aggID := id.NewAggregateID()
				corrID := id.MustParseCorrelationID("01HK154EJG2GP2SR75DK1Q1TBH")
				causeID := id.MustParseCausationID("01HK154FHRS5276AC3V7GRNTYM")
				uID := id.MustParseUserID("01HK1543TRR6BB4AF65NQX5V8S")
				reqID := id.MustParseRequestID("01HK154HG8WXD9A15YBY6FZJYW")
				evt, err := event.NewEvent(
					event.Type("UserRegistered"),
					aggID,
					event.AggregateType("User"),
					1,
					[]byte(`{"email":"alice@example.com"}`),
					event.WithCorrelationID(corrID),
					event.WithCausationID(causeID),
					event.WithUserID(uID),
					event.WithRequestID(reqID),
					event.WithSource("api"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(evt.Type()).To(Equal(event.Type("UserRegistered")))
				Expect(evt.AggregateID()).To(Equal(aggID))
				Expect(evt.AggregateType()).To(Equal(event.AggregateType("User")))
				Expect(evt.Version()).To(Equal(event.Version(1)))
				Expect(evt.Payload()).To(ContainSubstring("alice@example.com"))
				Expect(
					evt.Metadata().CorrelationID,
				).To(Equal(id.MustParseCorrelationID("01HK154EJG2GP2SR75DK1Q1TBH")))
				Expect(evt.Metadata().CausationID).To(
					Equal(id.MustParseCausationID("01HK154FHRS5276AC3V7GRNTYM")),
				)
				Expect(evt.Metadata().UserID).To(
					Equal(id.MustParseUserID("01HK1543TRR6BB4AF65NQX5V8S")),
				)
				Expect(evt.Metadata().RequestID).To(
					Equal(id.MustParseRequestID("01HK154HG8WXD9A15YBY6FZJYW")),
				)
				Expect(evt.Metadata().Source).To(Equal(event.Source("api")))
				Expect(evt.OccurredAt()).To(BeTemporally("<", time.Now().Add(time.Second)))
				Expect(evt.ID().IsZero()).To(BeFalse())
			})
		})

		Context("when I create an event with an empty aggregate ID", func() {
			It("should reject it with a descriptive error", func() {
				var emptyID id.AggregateID

				_, err := event.NewEvent(
					event.Type("BadEvent"),
					emptyID,
					event.AggregateType("User"),
					1,
					nil,
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("aggregate ID is required"))
			})
		})

		Context("when I create an event with an empty aggregate type", func() {
			It("should reject it with a descriptive error", func() {
				expectNewEventValidationFails(
					id.NewAggregateID(),
					event.AggregateType(""),
					1,
					"aggregate type is required",
				)
			})
		})

		Context("when I create an event with a zero version", func() {
			It("should reject it with a descriptive error", func() {
				expectNewEventValidationFails(
					id.NewAggregateID(),
					event.AggregateType("User"),
					0,
					"version",
				)
			})
		})

		Context("when I add custom metadata to an event", func() {
			It("should preserve it through the metadata map", func() {
				aggID := id.NewAggregateID()
				evt, err := event.NewEvent(
					event.Type("TestEvent"),
					aggID,
					event.AggregateType("Test"),
					1,
					nil,
					event.WithCustom(event.MetadataKey("tenant"), "acme-corp"),
					event.WithCustom(event.MetadataKey("region"), "us-east-1"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(
					evt.Metadata().Custom,
				).To(HaveKeyWithValue(event.MetadataKey("tenant"), "acme-corp"))
				Expect(
					evt.Metadata().Custom,
				).To(HaveKeyWithValue(event.MetadataKey("region"), "us-east-1"))
			})
		})
	})
})
