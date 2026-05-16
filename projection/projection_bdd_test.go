package projection_test

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type countingHandler struct {
	name      string
	types     []event.Type
	handler   func(ctx context.Context, evt event.Event) error
	callCount atomic.Int64
}

func (h *countingHandler) Name() string { return h.name }

func (h *countingHandler) EventTypes() []event.Type { return h.types }

func (h *countingHandler) Handle(ctx context.Context, evt event.Event) error {
	h.callCount.Add(1)

	if h.handler != nil {
		return h.handler(ctx, evt)
	}

	return nil
}

func makeBDDEvent(eventType event.Type, version event.Version) event.Event {
	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(eventType, aggID, "TestAggregate", version, []byte(`{}`))
	Expect(err).ToNot(HaveOccurred())

	return evt
}

var _ = Describe("Projection Runner", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		store      *memory.MemoryStore
		bus        *memory.MemoryBus
		checkpoint *memory.MemoryCheckpointStore
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		store = memory.NewMemoryStore()
		bus = memory.NewMemoryBus()
		checkpoint = memory.NewCheckpointStore()
	})

	AfterEach(func() {
		cancel()
	})

	Describe("As a developer building projections", func() {
		Context("when I replay historical events", func() {
			It("should process all past events through my projection", func() {
				// Pre-populate store with events
				aggID := id.NewAggregateID()
				events := []event.Event{
					makeBDDEvent("UserCreated", 1),
					makeBDDEvent("UserUpdated", 2),
					makeBDDEvent("UserDeleted", 3),
				}
				err := store.Save(ctx, event.AggregateType("TestAggregate"), aggID, events, 0)
				Expect(err).ToNot(HaveOccurred())

				runner, err := projection.NewRunner(store, bus, checkpoint)
				Expect(err).ToNot(HaveOccurred())

				handler := &countingHandler{name: "user-projection"}
				Expect(runner.Register(handler)).To(Succeed())

				// Run in background, cancel after replay
				go func() {
					defer GinkgoRecover()
					_ = runner.Run(ctx)
				}()

				Eventually(func() int64 { return handler.callCount.Load() }, time.Second).
					Should(Equal(int64(3)))
			})
		})

		Context("when my projection filters by event type", func() {
			It("should only receive events matching my filter", func() {
				aggID := id.NewAggregateID()
				events := []event.Event{
					makeBDDEvent("UserCreated", 1),
					makeBDDEvent("OrderPlaced", 2),
					makeBDDEvent("UserUpdated", 3),
				}
				err := store.Save(ctx, event.AggregateType("TestAggregate"), aggID, events, 0)
				Expect(err).ToNot(HaveOccurred())

				runner, err := projection.NewRunner(store, bus, checkpoint)
				Expect(err).ToNot(HaveOccurred())

				handler := &countingHandler{
					name:  "user-only",
					types: []event.Type{"UserCreated", "UserUpdated"},
				}
				Expect(runner.Register(handler)).To(Succeed())

				go func() {
					defer GinkgoRecover()
					_ = runner.Run(ctx)
				}()

				Eventually(func() int64 { return handler.callCount.Load() }, time.Second).
					Should(Equal(int64(2)))
			})
		})

		Context("when my projection subscribes to all events", func() {
			It("should receive every event type", func() {
				aggID := id.NewAggregateID()
				events := []event.Event{
					makeBDDEvent("UserCreated", 1),
					makeBDDEvent("OrderPlaced", 2),
				}
				err := store.Save(ctx, event.AggregateType("TestAggregate"), aggID, events, 0)
				Expect(err).ToNot(HaveOccurred())

				runner, err := projection.NewRunner(store, bus, checkpoint)
				Expect(err).ToNot(HaveOccurred())

				handler := &countingHandler{name: "all-events", types: nil}
				Expect(runner.Register(handler)).To(Succeed())

				go func() {
					defer GinkgoRecover()
					_ = runner.Run(ctx)
				}()

				Eventually(func() int64 { return handler.callCount.Load() }, time.Second).
					Should(Equal(int64(2)))
			})
		})

		Context("when I run with no projections registered", func() {
			It("should return ErrNoProjections", func() {
				runner, err := projection.NewRunner(store, bus, checkpoint)
				Expect(err).ToNot(HaveOccurred())

				err = runner.Run(ctx)
				Expect(err).To(MatchError(projection.ErrNoProjections))
			})
		})

		Context("when I register a nil projection", func() {
			It("should return ErrNilHandler", func() {
				runner, err := projection.NewRunner(store, bus, checkpoint)
				Expect(err).ToNot(HaveOccurred())

				err = runner.Register(nil)
				Expect(err).To(MatchError(projection.ErrNilHandler))
			})
		})

		Context("when I receive live events after replay", func() {
			It("should process both replayed and live events", func() {
				// Pre-populate one event
				aggID := id.NewAggregateID()
				events := []event.Event{makeBDDEvent("UserCreated", 1)}
				err := store.Save(ctx, event.AggregateType("TestAggregate"), aggID, events, 0)
				Expect(err).ToNot(HaveOccurred())

				runner, err := projection.NewRunner(store, bus, checkpoint)
				Expect(err).ToNot(HaveOccurred())

				handler := &countingHandler{name: "live-test"}
				Expect(runner.Register(handler)).To(Succeed())

				go func() {
					defer GinkgoRecover()
					_ = runner.Run(ctx)
				}()

				// Wait for replay
				Eventually(func() int64 { return handler.callCount.Load() }, time.Second).
					Should(BeNumerically(">=", 1))

				// Publish a live event
				liveEvt := makeBDDEvent("UserUpdated", 2)
				Expect(bus.Publish(ctx, liveEvt)).To(Succeed())

				Eventually(func() int64 { return handler.callCount.Load() }, time.Second).
					Should(Equal(int64(2)))
			})
		})

		Context("when I check the checkpoint after processing", func() {
			It("should return the last processed event ID", func() {
				aggID := id.NewAggregateID()
				evt := makeBDDEvent("UserCreated", 1)
				err := store.Save(
					ctx,
					event.AggregateType("TestAggregate"),
					aggID,
					[]event.Event{evt},
					0,
				)
				Expect(err).ToNot(HaveOccurred())

				runner, err := projection.NewRunner(store, bus, checkpoint)
				Expect(err).ToNot(HaveOccurred())

				handler := &countingHandler{name: "checkpoint-test"}
				Expect(runner.Register(handler)).To(Succeed())

				go func() {
					defer GinkgoRecover()
					_ = runner.Run(ctx)
				}()

				Eventually(func() int64 { return handler.callCount.Load() }, time.Second).
					Should(BeNumerically(">=", 1))

				cp, err := runner.CurrentCheckpoint(ctx, "checkpoint-test")
				Expect(err).ToNot(HaveOccurred())
				Expect(cp).To(Equal(evt.ID()))
			})
		})
	})

	Describe("As a developer validating my setup", func() {
		Context("when I create a runner without a bus", func() {
			It("should return ErrNilBus", func() {
				_, err := projection.NewRunner(store, nil, checkpoint)
				Expect(err).To(MatchError(projection.ErrNilBus))
			})
		})

		Context("when I create a runner without a checkpoint store", func() {
			It("should return ErrNilCheckpoint", func() {
				_, err := projection.NewRunner(store, bus, nil)
				Expect(err).To(MatchError(projection.ErrNilCheckpoint))
			})
		})

		Context("when I create a runner without a loader", func() {
			It("should work in live-only mode", func() {
				runner, err := projection.NewRunner(nil, bus, checkpoint)
				Expect(err).ToNot(HaveOccurred())

				handler := &countingHandler{name: "live-only"}
				Expect(runner.Register(handler)).To(Succeed())

				// Runner subscribes on Run(); we need to give it time
				go func() {
					defer GinkgoRecover()
					_ = runner.Run(ctx)
				}()

				// Give subscription time to register
				time.Sleep(10 * time.Millisecond)

				// Publish a live event
				liveEvt := makeBDDEvent("UserCreated", 1)
				Expect(bus.Publish(ctx, liveEvt)).To(Succeed())

				Eventually(func() int64 { return handler.callCount.Load() }, time.Second).
					Should(Equal(int64(1)))
			})
		})
	})

	Describe("As a developer using multiple projections", func() {
		Context("when I register multiple projections", func() {
			It("should dispatch events to each independently", func() {
				aggID := id.NewAggregateID()
				events := []event.Event{makeBDDEvent("UserCreated", 1)}
				err := store.Save(ctx, event.AggregateType("TestAggregate"), aggID, events, 0)
				Expect(err).ToNot(HaveOccurred())

				runner, err := projection.NewRunner(store, bus, checkpoint)
				Expect(err).ToNot(HaveOccurred())

				userHandler := &countingHandler{name: "users"}
				orderHandler := &countingHandler{name: "orders", types: []event.Type{"OrderPlaced"}}
				Expect(runner.Register(userHandler)).To(Succeed())
				Expect(runner.Register(orderHandler)).To(Succeed())

				go func() {
					defer GinkgoRecover()
					_ = runner.Run(ctx)
				}()

				Eventually(func() int64 { return userHandler.callCount.Load() }, time.Second).
					Should(Equal(int64(1)))
				Expect(orderHandler.callCount.Load()).To(Equal(int64(0)))
			})
		})
	})
})
