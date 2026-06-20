package event_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/schema/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

func initMemoryStoreTest(
	ctx *context.Context,
	store **memory.MemoryStore,
	aggID *id.AggregateID,
	aggType *event.AggregateType,
	typ event.AggregateType,
) {
	*ctx = context.Background()
	*store = memory.NewMemoryStore()
	*aggID = id.NewAggregateID()
	*aggType = typ
}

var _ = Describe("Event Creation", func() {
	Describe("As a developer building domain events", func() {
		Context("when I create a fully populated event", func() {
			It("should preserve every field and auto-generate ID and timestamp", func() {
				aggID := id.NewAggregateID()
				corrID := id.NewCorrelationID()
				causeID := id.NewCausationID()
				userID := id.NewUserID()
				reqID := id.NewRequestID()

				evt, err := event.NewEvent(
					"UserRegistered", aggID, "User", 1,
					[]byte(`{"email":"alice@example.com"}`),
					event.WithCorrelationID(corrID),
					event.WithCausationID(causeID),
					event.WithUserID(userID),
					event.WithRequestID(reqID),
					event.WithSource("api-gateway"),
					event.WithCustom(event.MetadataKey("tenant"), "acme-corp"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(evt.Type()).To(Equal(event.Type("UserRegistered")))
				Expect(evt.AggregateID()).To(Equal(aggID))
				Expect(evt.AggregateType()).To(Equal(event.AggregateType("User")))
				Expect(evt.Version()).To(Equal(event.Version(1)))
				Expect(evt.Payload()).To(Equal([]byte(`{"email":"alice@example.com"}`)))
				Expect(evt.ID().IsZero()).To(BeFalse())
				Expect(evt.OccurredAt()).To(BeTemporally("~", time.Now(), time.Second))

				Expect(evt.Metadata().CorrelationID).To(Equal(corrID))
				Expect(evt.Metadata().CausationID).To(Equal(causeID))
				Expect(evt.Metadata().UserID).To(Equal(userID))
				Expect(evt.Metadata().RequestID).To(Equal(reqID))
				Expect(evt.Metadata().Source).To(Equal(event.Source("api-gateway")))
				Expect(
					evt.Metadata().Custom,
				).To(HaveKeyWithValue(event.MetadataKey("tenant"), "acme-corp"))
			})
		})

		DescribeTable(
			"validation rejects invalid inputs",
			func(typ string, aggID id.AggregateID, aggType string, version event.Version, wantErr string) {
				_, err := event.NewEvent(
					event.Type(typ),
					aggID,
					event.AggregateType(aggType),
					version,
					nil,
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(wantErr))
			},
			Entry(
				"empty type",
				"",
				id.NewAggregateID(),
				"User",
				event.Version(1),
				"event type is required",
			),
			Entry(
				"zero aggregate ID",
				"UserCreated",
				id.AggregateID{},
				"User",
				event.Version(1),
				"aggregate ID is required",
			),
			Entry(
				"empty aggregate type",
				"UserCreated",
				id.NewAggregateID(),
				"",
				event.Version(1),
				"aggregate type is required",
			),
			Entry(
				"version zero",
				"UserCreated",
				id.NewAggregateID(),
				"User",
				event.Version(0),
				"version",
			),
		)

		Context("when I clone an event", func() {
			It(
				"should give me an independent copy so I can mutate it without affecting the original",
				func() {
					evt, err := event.NewEvent(
						"UserCreated", id.NewAggregateID(), "User", 1,
						[]byte(`{"name":"Alice"}`),
						event.WithCustom("key", "value"),
					)
					Expect(err).ToNot(HaveOccurred())

					clone := evt.Clone()

					Expect(clone.ID()).To(Equal(evt.ID()))
					Expect(clone.Type()).To(Equal(evt.Type()))
					Expect(clone.AggregateID()).To(Equal(evt.AggregateID()))
					Expect(clone.Version()).To(Equal(evt.Version()))
					Expect(clone.Payload()).To(Equal(evt.Payload()))

					clone.Payload()[0] = 'X'
					Expect(evt.Payload()[0]).To(Equal(byte('{')))
				},
			)
		})
	})
})

var _ = Describe("Event Store via MemoryStore", func() {
	var (
		ctx     context.Context
		store   *memory.MemoryStore
		aggID   id.AggregateID
		aggType event.AggregateType
	)

	BeforeEach(func() {
		initMemoryStoreTest(&ctx, &store, &aggID, &aggType, event.AggregateType("Order"))
	})

	savePlaced := func(expectedVersion event.Version) {
		Expect(store.Save(ctx, event.NewAggregateRef(aggType, aggID), []event.Event{
			mustNewEvent("OrderPlaced", aggID, aggType, 1),
		}, expectedVersion)).To(Succeed())
	}

	Describe("As a developer persisting aggregate events", func() {
		Context("when I save events for a new aggregate", func() {
			It("should persist them with correct versioning", func() {
				events := []event.Event{
					mustNewEvent("OrderPlaced", aggID, aggType, 1),
				}
				Expect(
					store.Save(ctx, event.NewAggregateRef(aggType, aggID), events, 0),
				).To(Succeed())

				loaded, err := store.Load(ctx, event.NewAggregateRef(aggType, aggID))
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded).To(HaveLen(1))
				Expect(loaded[0].Version()).To(Equal(event.Version(1)))
			})
		})

		Context("when I save with a wrong expected version", func() {
			It("should detect the version conflict", func() {
				savePlaced(0)

				err := store.Save(ctx, event.NewAggregateRef(aggType, aggID), []event.Event{
					mustNewEvent("OrderConfirmed", aggID, aggType, 2),
				}, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("version conflict"))
			})
		})

		Context("when my save conflicts, but I reload and retry with the correct version", func() {
			It(
				"should succeed on the second attempt so I can recover from concurrent writes",
				func() {
					savePlaced(0)

					err := store.Save(ctx, event.NewAggregateRef(aggType, aggID), []event.Event{
						mustNewEvent("OrderConfirmed", aggID, aggType, 2),
					}, 0)
					Expect(err).To(HaveOccurred())

					loaded, loadErr := store.Load(ctx, event.NewAggregateRef(aggType, aggID))
					Expect(loadErr).ToNot(HaveOccurred())
					currentVersion := len(loaded)

					Expect(store.Save(ctx, event.NewAggregateRef(aggType, aggID), []event.Event{
						mustNewEvent("OrderConfirmed", aggID, aggType, 2),
					}, event.Version(currentVersion))).To(Succeed())
				},
			)
		})

		Context("when I close the store", func() {
			BeforeEach(func() {
				savePlaced(0)
			})

			It(
				"should reject any further operations so I don't accidentally use a closed store",
				func() {
					Expect(store.Close()).To(Succeed())

					err := store.Save(ctx, event.NewAggregateRef(aggType, aggID), []event.Event{
						mustNewEvent("OrderConfirmed", aggID, aggType, 2),
					}, 1)
					Expect(err).To(HaveOccurred())
				},
			)
		})
	})
})

var _ = Describe("Schema Evolution", func() {
	var (
		ctx     context.Context
		store   *memory.MemoryStore
		aggID   id.AggregateID
		aggType event.AggregateType
	)

	BeforeEach(func() {
		initMemoryStoreTest(&ctx, &store, &aggID, &aggType, event.AggregateType("User"))
	})

	Describe("As a developer deploying schema v2", func() {
		Context("when old v1 events are loaded", func() {
			It("should automatically upcast them to v2", func() {
				v1Event, err := event.NewEvent(
					"UserCreated", aggID, "User", 1,
					[]byte(`{"name":"Alice"}`),
					event.WithSchemaVersion(1),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(
					store.Save(
						ctx,
						event.NewAggregateRef(aggType, aggID),
						[]event.Event{v1Event},
						0,
					),
				).To(Succeed())

				upcaster := makeUpcaster("UserCreated", 1, []byte(`{"name":"Alice","email":""}`))
				versioned, err := schema.NewVersionedStore(store, upcaster)
				Expect(err).ToNot(HaveOccurred())

				loaded, err := versioned.Load(ctx, event.NewAggregateRef(aggType, aggID))
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded).To(HaveLen(1))
				Expect(loaded[0].SchemaVersion()).To(Equal(event.SchemaVersion(2)))
				Expect(loaded[0].Payload()).To(ContainSubstring("email"))
			})
		})

		Context("when events already at the latest version are loaded", func() {
			It(
				"should skip upcasting and keep my data intact, avoiding unnecessary transformations",
				func() {
					v2Event, err := event.NewEvent(
						"UserCreated", aggID, "User", 1,
						[]byte(`{"name":"Alice","email":"a@b.com"}`),
						event.WithSchemaVersion(2),
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(
						store.Save(
							ctx,
							event.NewAggregateRef(aggType, aggID),
							[]event.Event{v2Event},
							0,
						),
					).To(Succeed())

					upcaster := makeUpcaster("UserCreated", 1, []byte(`{"name":"","email":""}`))
					versioned, err := schema.NewVersionedStore(store, upcaster)
					Expect(err).ToNot(HaveOccurred())

					loaded, err := versioned.Load(ctx, event.NewAggregateRef(aggType, aggID))
					Expect(err).ToNot(HaveOccurred())
					Expect(loaded[0].SchemaVersion()).To(Equal(event.SchemaVersion(2)))
					Expect(loaded[0].Payload()).To(ContainSubstring("a@b.com"))
				},
			)
		})

		Context("when chained upcasters v1→v2→v3 are registered", func() {
			It("should apply them in version order", func() {
				v1Event, err := event.NewEvent(
					"UserCreated", aggID, "User", 1,
					[]byte(`{"name":"Alice"}`),
					event.WithSchemaVersion(1),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(
					store.Save(
						ctx,
						event.NewAggregateRef(aggType, aggID),
						[]event.Event{v1Event},
						0,
					),
				).To(Succeed())

				upcasterV1toV2 := makeUpcaster(
					"UserCreated",
					1,
					[]byte(`{"name":"Alice","email":""}`),
				)
				upcasterV2toV3 := makeUpcaster(
					"UserCreated",
					2,
					[]byte(`{"fullName":"Alice","email":"","verified":false}`),
				)
				versioned, err := schema.NewVersionedStore(store, upcasterV1toV2, upcasterV2toV3)
				Expect(err).ToNot(HaveOccurred())

				loaded, err := versioned.Load(ctx, event.NewAggregateRef(aggType, aggID))
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded[0].SchemaVersion()).To(Equal(event.SchemaVersion(3)))
				Expect(loaded[0].Payload()).To(ContainSubstring("fullName"))
			})
		})
	})
})

var _ = Describe("Gomega Matcher Adoption", func() {
	Describe("As a developer comparing event payloads", func() {
		Context("when JSON payloads have different key ordering", func() {
			It("should match semantically using MatchJSON regardless of field order", func() {
				aggID := id.NewAggregateID()
				evt1, err := event.NewEvent("UserCreated", aggID, "User", 1,
					[]byte(`{"name":"Alice","email":"a@b.com"}`))
				Expect(err).ToNot(HaveOccurred())

				evt2, err := event.NewEvent("UserCreated", aggID, "User", 1,
					[]byte(`{"email":"a@b.com","name":"Alice"}`))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(evt1.Payload())).To(MatchJSON(evt2.Payload()))
			})
		})
	})

	Describe("As a developer checking event types unordered", func() {
		Context("when events arrive in any order", func() {
			It("should verify types with ConsistOf regardless of ordering", func() {
				aggID := id.NewAggregateID()
				events := []event.Event{
					mustNewEvent("UserCreated", aggID, "User", 1),
					mustNewEvent("UserUpdated", aggID, "User", 2),
					mustNewEvent("UserDeleted", aggID, "User", 3),
				}

				types := make([]event.Type, len(events))
				for i, evt := range events {
					types[i] = evt.Type()
				}

				Expect(types).To(ConsistOf(
					event.Type("UserCreated"),
					event.Type("UserUpdated"),
					event.Type("UserDeleted"),
				))
			})
		})
	})
})

var _ = Describe("Error Classification", func() {
	Describe("As a developer building retry logic", func() {
		Context("when I classify errors", func() {
			It(
				"should treat transient errors as safe to retry, so my infrastructure can self-heal",
				func() {
					err := event.NewTransient("test.retry", "connection timeout")
					Expect(event.IsRetryable(err)).To(BeTrue())
				},
			)

			It("should treat rejection errors as permanent, so I stop retrying bad input", func() {
				err := event.NewRejection("test.reject", "invalid input")
				Expect(event.IsRetryable(err)).To(BeFalse())
			})

			It(
				"should treat conflict errors as permanent, so I don't hammer a version mismatch",
				func() {
					err := event.NewConflict("test.conflict", "version mismatch")
					Expect(event.IsRetryable(err)).To(BeFalse())
				},
			)

			It("should treat unknown errors as retryable (safe default)", func() {
				Expect(event.IsRetryable(errors.New("something"))).To(BeTrue())
			})
		})
	})
})

func mustNewEvent(
	eventType event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
) event.Event {
	evt, err := event.NewEvent(eventType, aggID, aggType, version, []byte(`{}`))
	Expect(err).ToNot(HaveOccurred())

	return evt
}

func makeUpcaster(
	targetType event.Type,
	fromVersion event.SchemaVersion,
	newPayload []byte,
) schema.Upcaster {
	return schema.NewUpcaster(
		targetType,
		fromVersion,
		func(evt event.Event) (*event.ImmutableEvent, error) {
			return event.NewEvent(
				evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
				newPayload,
				event.WithSchemaVersion(fromVersion+1),
			)
		},
	)
}
