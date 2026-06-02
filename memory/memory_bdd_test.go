package memory_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
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
			It(
				"should persist them and let me load them back for my aggregate reconstruction",
				func() {
					events := []event.Event{makeMemEvent("Created", aggID, 1)}
					err := store.Save(
						ctx,
						event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
						events,
						0,
					)
					Expect(err).ToNot(HaveOccurred())

					loaded, err := store.Load(
						ctx,
						event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(loaded).To(HaveLen(1))
					Expect(loaded[0].Type()).To(Equal(event.Type("Created")))
				},
			)
		})

		Context("when I save events with the wrong expected version", func() {
			It(
				"should detect the version mismatch so concurrent writers don't silently overwrite each other",
				func() {
					events := []event.Event{makeMemEvent("Created", aggID, 1)}
					err := store.Save(
						ctx,
						event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
						events,
						0,
					)
					Expect(err).ToNot(HaveOccurred())

					more := []event.Event{makeMemEvent("Updated", aggID, 2)}
					err = store.Save(
						ctx,
						event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
						more,
						0,
					) // wrong expected version
					Expect(err).To(HaveOccurred())
				},
			)
		})

		Context("when I load a non-existent aggregate", func() {
			It("should explain that the aggregate was not found", func() {
				_, err := store.Load(
					ctx,
					event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("aggregate not found"))
			})
		})

		Context("when I append batch events", func() {
			It(
				"should append them after existing events for bulk imports without re-specifying the version",
				func() {
					initial := []event.Event{makeMemEvent("Created", aggID, 1)}
					Expect(
						store.Save(
							ctx,
							event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
							initial,
							0,
						),
					).To(Succeed())

					batch := []event.Event{
						makeMemEvent("Updated", aggID, 2),
						makeMemEvent("Updated", aggID, 3),
					}
					Expect(
						store.AppendBatch(
							ctx,
							event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
							batch,
						),
					).To(Succeed())

					loaded, err := store.Load(
						ctx,
						event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(loaded).To(HaveLen(3))
				},
			)
		})

		Context("when I load events from a specific version", func() {
			It(
				"should return only events from that version onward so I can replay just what I missed",
				func() {
					events := []event.Event{
						makeMemEvent("Created", aggID, 1),
						makeMemEvent("Updated", aggID, 2),
						makeMemEvent("Updated", aggID, 3),
					}
					Expect(
						store.Save(
							ctx,
							event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
							events,
							0,
						),
					).To(Succeed())

					// LoadFromVersion(v) returns events from index v onward
					// Version(2) → index 2 → only version 3 event
					fromV2, err := store.LoadFromVersion(
						ctx,
						event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
						2,
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(fromV2).To(HaveLen(1))
					Expect(fromV2[0].Version()).To(Equal(event.Version(3)))

					// Version(1) → index 1 → versions 2 and 3
					fromV1, err := store.LoadFromVersion(
						ctx,
						event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
						1,
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(fromV1).To(HaveLen(2))
				},
			)
		})

		Context("when I close the store", func() {
			It(
				"should reject all further operations so I don't use a closed store by accident",
				func() {
					Expect(store.Close()).To(Succeed())

					err := store.Save(
						ctx,
						event.NewAggregateRef(event.AggregateType("TestAggregate"), aggID),
						[]event.Event{makeMemEvent("Created", aggID, 1)},
						0,
					)
					Expect(err).To(HaveOccurred())
				},
			)
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
			It(
				"should deliver them only to subscribers who registered for that specific type",
				func() {
					var received []event.Event

					err := bus.Subscribe(
						"UserCreated",
						func(_ context.Context, evt event.Event) error {
							received = append(received, evt)

							return nil
						},
					)
					Expect(err).ToNot(HaveOccurred())

					evt := makeMemEvent("UserCreated", id.NewAggregateID(), 1)
					Expect(bus.Publish(ctx, evt)).To(Succeed())

					Expect(received).To(HaveLen(1))
					Expect(received[0].ID()).To(Equal(evt.ID()))
				},
			)
		})

		Context("when I subscribe to all events", func() {
			It(
				"should receive events of any type for cross-cutting concerns like audit logging",
				func() {
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
				},
			)
		})

		Context("when I close the bus", func() {
			It(
				"should reject all further operations so I don't publish to a shut-down bus",
				func() {
					Expect(bus.Close()).To(Succeed())

					err := bus.Publish(ctx, makeMemEvent("UserCreated", id.NewAggregateID(), 1))
					Expect(err).To(HaveOccurred())
				},
			)
		})
	})
})

var _ = Describe("MemorySnapshotStore", func() {
	var (
		ctx       context.Context
		snapStore *memory.MemorySnapshotStore
		aggID     id.AggregateID
		aggType   event.AggregateType
	)

	BeforeEach(func() {
		ctx = context.Background()
		snapStore = memory.NewMemorySnapshotStore()
		aggID = id.NewAggregateID()
		aggType = event.AggregateType("Order")
	})

	Describe("As a developer speeding up aggregate loading with snapshots", func() {
		Context("when I save a snapshot and load it back", func() {
			It("should roundtrip my aggregate state so I can skip replaying all events", func() {
				snap := snapshot.Snapshot{
					AggregateID:   aggID,
					AggregateType: aggType,
					Version:       event.Version(5),
					State:         []byte(`{"status":"active","items":3}`),
				}
				Expect(snapStore.Save(ctx, snap)).To(Succeed())

				loaded, err := snapStore.Load(ctx, event.NewAggregateRef(aggType, aggID))
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded.Version).To(Equal(event.Version(5)))
				Expect(loaded.State).To(Equal([]byte(`{"status":"active","items":3}`)))
			})
		})

		Context("when I save a newer snapshot for the same aggregate", func() {
			It("should replace the old one so I always get the latest state", func() {
				Expect(snapStore.Save(ctx, eventtest.QuickSnapshot(
					aggID, aggType, event.Version(3), []byte(`{"status":"old"}`),
				))).To(Succeed())

				Expect(snapStore.Save(ctx, eventtest.QuickSnapshot(
					aggID, aggType, event.Version(7), []byte(`{"status":"new"}`),
				))).To(Succeed())

				loaded, err := snapStore.Load(ctx, event.NewAggregateRef(aggType, aggID))
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded.Version).To(Equal(event.Version(7)))
			})
		})

		Context("when I load a snapshot for a non-existent aggregate", func() {
			It("should explain that no snapshot was found so I fall back to full replay", func() {
				_, err := snapStore.Load(ctx, event.NewAggregateRef(aggType, aggID))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("snapshot not found"))
			})
		})
	})
})

var _ = Describe("MemoryCheckpointStore", func() {
	var (
		ctx     context.Context
		cpStore *memory.MemoryCheckpointStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		cpStore = memory.NewMemoryCheckpointStore()
	})

	Describe("As a developer tracking my projection position in the event stream", func() {
		Context("when I save a checkpoint and load it back", func() {
			It("should remember where my projection stopped so I can resume from there", func() {
				evtID := id.NewEventID()
				Expect(
					cpStore.Save(ctx, "user-projection", event.Checkpoint{EventID: evtID}),
				).To(Succeed())

				loaded, err := cpStore.Load(ctx, "user-projection")
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded.EventID).To(Equal(evtID))
			})
		})

		Context("when I load a checkpoint for a projection that never ran", func() {
			It("should return a zero ID so I know to start from the beginning", func() {
				loaded, err := cpStore.Load(ctx, "never-ran-projection")
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded.IsZero()).To(BeTrue())
			})
		})

		Context("when I update the checkpoint after processing more events", func() {
			It("should overwrite the old position so I always resume from the latest", func() {
				first := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}
				Expect(cpStore.Save(ctx, "orders", first)).To(Succeed())

				second := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}
				Expect(cpStore.Save(ctx, "orders", second)).To(Succeed())

				loaded, err := cpStore.Load(ctx, "orders")
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded.EventID).To(Equal(second.EventID))
			})
		})
	})
})
