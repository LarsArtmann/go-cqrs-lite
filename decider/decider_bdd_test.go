package decider_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func everyN(n int) snapshot.SnapshotStrategy {
	s, err := snapshot.EveryNEvents(n)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return s
}

// bddCounter models a simple counter aggregate for BDD scenarios.
type bddCounter struct {
	Value int
}

var errBDDRejected = errors.New("counter rejected")

func foldBDDCounter(state bddCounter, evt event.Event) (bddCounter, error) {
	switch evt.Type() {
	case "CounterCreated":
		return bddCounter{Value: 1}, nil
	case "CounterIncremented":
		return bddCounter{Value: state.Value + 1}, nil
	}

	return state, nil
}

func bddCounterDecider() decider.Decider[bddCounter] {
	return decider.Decider[bddCounter]{
		Initial: bddCounter{Value: 0},
		Apply:   foldBDDCounter,
	}
}

func makeCounterEvent(
	eventType event.Type,
	aggID id.StreamID,
	version event.Version,
) event.Event {
	evt, err := event.NewEvent(eventType, aggID, "Counter", version, []byte(`{}`))
	Expect(err).ToNot(HaveOccurred())

	return evt
}

func executeCounterNTimes(
	ctx context.Context,
	repo *decider.Repository[bddCounter],
	aggID id.StreamID,
	n int,
) {
	for range n {
		err := repo.Execute(
			ctx, aggID, "Counter",
			func(_ bddCounter, v event.Version) ([]event.Event, error) {
				if v == 0 {
					return []event.Event{makeCounterEvent("CounterCreated", aggID, v+1)}, nil
				}

				return []event.Event{makeCounterEvent("CounterIncremented", aggID, v+1)}, nil
			},
		)
		Expect(err).ToNot(HaveOccurred())
	}
}

func newSnapshotRepo(
	store event.Store,
	bus event.Bus,
	snapStore *memory.MemorySnapshotStore,
	n int,
) (*decider.Repository[bddCounter], error) {
	return decider.NewRepository(
		store, bus, bddCounterDecider(),
		decider.WithSnapshotStore[bddCounter](snapStore),
		decider.WithCodec[bddCounter](codec.JSONCodec{}),
		decider.WithSnapshotStrategy[bddCounter](everyN(n)),
	)
}

func executeCounterCommand(
	ctx context.Context,
	repo *decider.Repository[bddCounter],
	aggID id.StreamID,
	eventName string,
) {
	err := repo.Execute(
		ctx, aggID, "Counter",
		func(_ bddCounter, v event.Version) ([]event.Event, error) {
			return []event.Event{makeCounterEvent(event.Type(eventName), aggID, v+1)}, nil
		},
	)
	Expect(err).ToNot(HaveOccurred())
}

func createCounter(
	ctx context.Context,
	repo *decider.Repository[bddCounter],
	aggID id.StreamID,
) {
	executeCounterCommand(ctx, repo, aggID, "CounterCreated")
}

func incrementCounter(
	ctx context.Context,
	repo *decider.Repository[bddCounter],
	aggID id.StreamID,
) {
	executeCounterCommand(ctx, repo, aggID, "CounterIncremented")
}

func executeAndAssertNoStateChange(
	ctx context.Context,
	repo *decider.Repository[bddCounter],
	aggID id.StreamID,
	decideErr error,
) {
	err := repo.Execute(
		ctx, aggID, "Counter",
		func(_ bddCounter, _ event.Version) ([]event.Event, error) {
			return nil, decideErr
		},
	)
	if decideErr != nil {
		Expect(err).To(MatchError(decideErr))
	} else {
		Expect(err).ToNot(HaveOccurred())
	}

	state, version, err := repo.Load(ctx, aggID, "Counter")
	Expect(err).ToNot(HaveOccurred())
	Expect(state.Value).To(Equal(0))
	Expect(version).To(Equal(event.Version(0)))
}

var _ = Describe("Decider Repository", func() {
	var (
		ctx   context.Context
		store *memory.MemoryStore
		bus   *eventtest.FakeBus
		repo  *decider.Repository[bddCounter]
		aggID id.StreamID
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = memory.NewMemoryStore()
		bus = eventtest.NewFakeBus()
		aggID = id.NewAggregateID()

		var err error
		repo, err = decider.NewRepository(store, bus, bddCounterDecider())
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("As a developer using the Decider pattern", func() {
		Context("when I create a new aggregate", func() {
			It("should save and publish the decision events", func() {
				err := repo.Execute(
					ctx, aggID, "Counter",
					func(_ bddCounter, v event.Version) ([]event.Event, error) {
						return []event.Event{makeCounterEvent("CounterCreated", aggID, v+1)}, nil
					},
				)
				Expect(err).ToNot(HaveOccurred())

				state, version, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(1))
				Expect(version).To(Equal(event.Version(1)))
			})
		})

		Context("when I apply multiple decisions to the same aggregate", func() {
			It("should apply all events into the correct state", func() {
				createCounter(ctx, repo, aggID)
				incrementCounter(ctx, repo, aggID)
				incrementCounter(ctx, repo, aggID)

				state, version, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(3))
				Expect(version).To(Equal(event.Version(3)))
			})
		})

		Context("when my decide function returns no events", func() {
			It("should not save or publish anything", func() {
				executeAndAssertNoStateChange(ctx, repo, aggID, nil)
			})
		})

		Context("when my decide function returns an error", func() {
			It("should not save any events", func() {
				executeAndAssertNoStateChange(ctx, repo, aggID, errBDDRejected)
			})
		})

		Context("when I load an aggregate that does not exist", func() {
			It(
				"should give me the initial state and version 0, so I know this aggregate has no history",
				func() {
					state, version, err := repo.Load(ctx, aggID, "Counter")
					Expect(err).ToNot(HaveOccurred())
					Expect(state.Value).To(Equal(0))
					Expect(version).To(Equal(event.Version(0)))
				},
			)
		})

		Context("when I decide based on current state", func() {
			It(
				"should pass the folded state from previous events so my decide function sees the full picture",
				func() {
					createCounter(ctx, repo, aggID)

					var receivedState bddCounter
					var receivedVersion event.Version
					err := repo.Execute(
						ctx, aggID, "Counter",
						func(state bddCounter, v event.Version) ([]event.Event, error) {
							receivedState = state
							receivedVersion = v

							return []event.Event{
								makeCounterEvent("CounterIncremented", aggID, v+1),
							}, nil
						},
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(receivedState.Value).To(Equal(1))
					Expect(receivedVersion).To(Equal(event.Version(1)))
				},
			)
		})

		Context("when I emit multiple events in a single decision", func() {
			It("should save and publish all of them atomically", func() {
				err := repo.Execute(
					ctx, aggID, "Counter",
					func(_ bddCounter, v event.Version) ([]event.Event, error) {
						return []event.Event{
							makeCounterEvent("CounterCreated", aggID, v+1),
							makeCounterEvent("CounterIncremented", aggID, v+2),
							makeCounterEvent("CounterIncremented", aggID, v+3),
						}, nil
					},
				)
				Expect(err).ToNot(HaveOccurred())

				state, version, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(3))
				Expect(version).To(Equal(event.Version(3)))
			})
		})
	})

	Describe("As a developer validating my setup", func() {
		Context("when I create a repository without a store", func() {
			It("should reject my setup and explain that an event store is required", func() {
				_, err := decider.NewRepository(nil, bus, bddCounterDecider())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("event store is required"))
			})
		})

		Context("when I create a repository without a publisher", func() {
			It("should succeed and operate in pure event-sourcing mode", func() {
				repo, err := decider.NewRepository(store, nil, bddCounterDecider())
				Expect(err).NotTo(HaveOccurred())
				Expect(repo).NotTo(BeNil())
			})
		})

		Context("when I create a repository without a apply function", func() {
			It("should reject my setup and explain that a apply function is required", func() {
				_, err := decider.NewRepository(
					store,
					bus,
					decider.Decider[bddCounter]{},
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("apply function is required"))
			})
		})
	})

	Describe("As a developer building an event-sourced system with snapshots", func() {
		var snapStore *memory.MemorySnapshotStore

		BeforeEach(func() {
			snapStore = memory.NewMemorySnapshotStore()
		})

		Context("when I configure a snapshot strategy", func() {
			It("should save a snapshot every N events", func() {
				repo, err := newSnapshotRepo(store, bus, snapStore, 2)
				Expect(err).ToNot(HaveOccurred())

				executeCounterNTimes(ctx, repo, aggID, 4)

				state, _, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(4))
			})
		})

		Context("when I load from a snapshot", func() {
			It("should replay only events after the snapshot version", func() {
				repo, err := newSnapshotRepo(store, bus, snapStore, 2)
				Expect(err).ToNot(HaveOccurred())

				executeCounterNTimes(ctx, repo, aggID, 3)

				state, version, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(3))
				Expect(version).To(Equal(event.Version(3)))
			})
		})
	})
})
