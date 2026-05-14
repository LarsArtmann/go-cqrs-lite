package decider_test

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
		Fold:    foldBDDCounter,
	}
}

func makeCreateEvent(aggID id.AggregateID, version event.Version) event.Event {
	evt, err := event.NewEvent("CounterCreated", aggID, "Counter", version.Int(), []byte(`{}`))
	Expect(err).ToNot(HaveOccurred())

	return evt
}

func makeIncrementEvent(aggID id.AggregateID, version event.Version) event.Event {
	evt, err := event.NewEvent("CounterIncremented", aggID, "Counter", version.Int(), []byte(`{}`))
	Expect(err).ToNot(HaveOccurred())

	return evt
}

func createCounter(
	ctx context.Context,
	repo *decider.Repository[bddCounter],
	aggID id.AggregateID,
) {
	err := repo.Execute(
		ctx, aggID, "Counter",
		func(_ bddCounter, v event.Version) ([]event.Event, error) {
			return []event.Event{makeCreateEvent(aggID, v+1)}, nil
		},
	)
	Expect(err).ToNot(HaveOccurred())
}

func incrementCounter(
	ctx context.Context,
	repo *decider.Repository[bddCounter],
	aggID id.AggregateID,
) {
	err := repo.Execute(
		ctx, aggID, "Counter",
		func(_ bddCounter, v event.Version) ([]event.Event, error) {
			return []event.Event{makeIncrementEvent(aggID, v+1)}, nil
		},
	)
	Expect(err).ToNot(HaveOccurred())
}

var _ = Describe("Decider Repository", func() {
	var (
		ctx   context.Context
		store *memory.MemoryStore
		bus   *memory.MemoryBus
		repo  *decider.Repository[bddCounter]
		aggID id.AggregateID
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = memory.NewMemoryStore()
		bus = memory.NewMemoryBus()
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
						return []event.Event{makeCreateEvent(aggID, v+1)}, nil
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
			It("should fold all events into the correct state", func() {
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
				err := repo.Execute(
					ctx, aggID, "Counter",
					func(_ bddCounter, _ event.Version) ([]event.Event, error) {
						return nil, nil
					},
				)
				Expect(err).ToNot(HaveOccurred())

				state, version, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(0))
				Expect(version).To(Equal(event.Version(0)))
			})
		})

		Context("when my decide function returns an error", func() {
			It("should not save any events", func() {
				err := repo.Execute(
					ctx, aggID, "Counter",
					func(_ bddCounter, _ event.Version) ([]event.Event, error) {
						return nil, errBDDRejected
					},
				)
				Expect(err).To(MatchError(errBDDRejected))

				state, version, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(0))
				Expect(version).To(Equal(event.Version(0)))
			})
		})

		Context("when I load an aggregate that does not exist", func() {
			It("should return initial state and version 0", func() {
				state, version, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(0))
				Expect(version).To(Equal(event.Version(0)))
			})
		})

		Context("when I decide based on current state", func() {
			It("should receive the folded state from previous events", func() {
				createCounter(ctx, repo, aggID)

				var receivedState bddCounter
				var receivedVersion event.Version
				err := repo.Execute(
					ctx, aggID, "Counter",
					func(state bddCounter, v event.Version) ([]event.Event, error) {
						receivedState = state
						receivedVersion = v

						return []event.Event{makeIncrementEvent(aggID, v+1)}, nil
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(receivedState.Value).To(Equal(1))
				Expect(receivedVersion).To(Equal(event.Version(1)))
			})
		})

		Context("when I emit multiple events in a single decision", func() {
			It("should save and publish all of them atomically", func() {
				err := repo.Execute(
					ctx, aggID, "Counter",
					func(_ bddCounter, v event.Version) ([]event.Event, error) {
						return []event.Event{
							makeCreateEvent(aggID, v+1),
							makeIncrementEvent(aggID, v+2),
							makeIncrementEvent(aggID, v+3),
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

		Context("when I delete an aggregate", func() {
			BeforeEach(func() {
				createCounter(ctx, repo, aggID)
			})

			It("should remove all events and return initial state on load", func() {
				err := repo.Delete(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())

				state, version, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(0))
				Expect(version).To(Equal(event.Version(0)))
			})
		})
	})

	Describe("As a developer validating my setup", func() {
		Context("when I create a repository without a store", func() {
			It("should return ErrNilStore", func() {
				_, err := decider.NewRepository[bddCounter](nil, bus, bddCounterDecider())
				Expect(err).To(MatchError(decider.ErrNilStore))
			})
		})

		Context("when I create a repository without a bus", func() {
			It("should return ErrNilBus", func() {
				_, err := decider.NewRepository[bddCounter](store, nil, bddCounterDecider())
				Expect(err).To(MatchError(decider.ErrNilBus))
			})
		})

		Context("when I create a repository without a fold function", func() {
			It("should return ErrNilFold", func() {
				_, err := decider.NewRepository[bddCounter](
					store,
					bus,
					decider.Decider[bddCounter]{},
				)
				Expect(err).To(MatchError(decider.ErrNilFold))
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
				repo, err := decider.NewRepository[bddCounter](
					store, bus, bddCounterDecider(),
					decider.WithSnapshotStore[bddCounter](snapStore),
					decider.WithCodec[bddCounter](event.JSONCodec{}),
					decider.WithSnapshotStrategy[bddCounter](event.MustEveryNEvents(2)),
				)
				Expect(err).ToNot(HaveOccurred())

				for i := 0; i < 4; i++ {
					err = repo.Execute(
						ctx, aggID, "Counter",
						func(_ bddCounter, v event.Version) ([]event.Event, error) {
							if v == 0 {
								return []event.Event{makeCreateEvent(aggID, v+1)}, nil
							}

							return []event.Event{makeIncrementEvent(aggID, v+1)}, nil
						},
					)
					Expect(err).ToNot(HaveOccurred())
				}

				state, _, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(4))
			})
		})

		Context("when I load from a snapshot", func() {
			It("should replay only events after the snapshot version", func() {
				repo, err := decider.NewRepository[bddCounter](
					store, bus, bddCounterDecider(),
					decider.WithSnapshotStore[bddCounter](snapStore),
					decider.WithCodec[bddCounter](event.JSONCodec{}),
					decider.WithSnapshotStrategy[bddCounter](event.MustEveryNEvents(2)),
				)
				Expect(err).ToNot(HaveOccurred())

				for i := 0; i < 3; i++ {
					err = repo.Execute(
						ctx, aggID, "Counter",
						func(_ bddCounter, v event.Version) ([]event.Event, error) {
							if v == 0 {
								return []event.Event{makeCreateEvent(aggID, v+1)}, nil
							}

							return []event.Event{makeIncrementEvent(aggID, v+1)}, nil
						},
					)
					Expect(err).ToNot(HaveOccurred())
				}

				state, version, err := repo.Load(ctx, aggID, "Counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Value).To(Equal(3))
				Expect(version).To(Equal(event.Version(3)))
			})
		})
	})
})
