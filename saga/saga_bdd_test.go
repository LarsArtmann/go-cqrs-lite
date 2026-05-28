package saga_test

import (
	"context"
	"errors"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

type bddDispatcher struct {
	mu         sync.Mutex
	dispatched []command.Command
	err        error
	callCount  int
	failOnCall int // fail exactly on this call number (1-based), then succeed
}

func (d *bddDispatcher) Dispatch(_ context.Context, cmd command.Command) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dispatched = append(d.dispatched, cmd)
	d.callCount++
	if d.failOnCall > 0 && d.callCount == d.failOnCall {
		return errors.New("dispatch failed")
	}
	return d.err
}

func (d *bddDispatcher) Dispatched() []command.Command {
	d.mu.Lock()
	defer d.mu.Unlock()
	result := make([]command.Command, len(d.dispatched))
	copy(result, d.dispatched)
	return result
}

func (d *bddDispatcher) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.callCount
}

func (d *bddDispatcher) SetError(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.err = err
}

type orderSaga struct {
	steps []saga.Step
}

func (o *orderSaga) SagaType() string   { return "order" }
func (o *orderSaga) Steps() []saga.Step { return o.steps }

func newBDDCommand(_ context.Context, _ id.AggregateID) command.Command {
	return &testCommand{BasicCommand: *command.MustNew("TestCommand", id.NewAggregateID())}
}

var _ = Describe("Saga Runner", func() {
	var (
		ctx        context.Context
		store      *saga.MemoryStore
		dispatcher *bddDispatcher
		runner     *saga.Runner
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = saga.NewMemoryStore()
		dispatcher = &bddDispatcher{}
		runner = saga.NewRunner(store, dispatcher, saga.WithRetryPolicy(0, 0))
	})

	Describe("As a developer orchestrating multi-step processes", func() {
		Context("when I register a saga definition and start an instance", func() {
			It("should create a running instance and dispatch the initial command", func() {
				def := &orderSaga{steps: []saga.Step{
					{Name: "reserve-stock", Action: newBDDCommand},
				}}
				Expect(runner.Register(def)).To(Succeed())

				initialCmd := command.MustNew("CreateOrder", id.NewAggregateID())
				instance, err := runner.Start(ctx, "order", initialCmd)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance).ToNot(BeNil())
				Expect(instance.Status).To(Equal(saga.StatusRunning))
				Expect(instance.SagaType).To(Equal("order"))
				Expect(instance.ID.IsZero()).To(BeFalse())
				Expect(dispatcher.Count()).To(Equal(1))
			})
		})

		Context("when I execute all steps successfully", func() {
			It("should transition to completed after the last step", func() {
				def := &orderSaga{steps: []saga.Step{
					{Name: "reserve-stock", Action: newBDDCommand},
					{Name: "charge-payment", Action: newBDDCommand},
					{Name: "ship-order", Action: newBDDCommand},
				}}
				Expect(runner.Register(def)).To(Succeed())

				instance, err := runner.Start(ctx, "order", nil)
				Expect(err).ToNot(HaveOccurred())

				Expect(runner.ExecuteStep(ctx, instance.ID)).To(Succeed())
				state, err := store.Load(ctx, instance.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Status).To(Equal(saga.StatusRunning))
				Expect(state.CurrentStep).To(Equal(1))

				Expect(runner.ExecuteStep(ctx, instance.ID)).To(Succeed())
				state, err = store.Load(ctx, instance.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(state.CurrentStep).To(Equal(2))

				Expect(runner.ExecuteStep(ctx, instance.ID)).To(Succeed())
				state, err = store.Load(ctx, instance.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(state.Status).To(Equal(saga.StatusCompleted))
				Expect(state.CurrentStep).To(Equal(3))
			})
		})

		Context("when a step fails on the first step", func() {
			It("should mark the saga as failed without compensation", func() {
				def := &orderSaga{steps: []saga.Step{
					{Name: "reserve-stock", Action: newBDDCommand},
					{Name: "charge-payment", Action: newBDDCommand},
				}}
				Expect(runner.Register(def)).To(Succeed())

				instance, err := runner.Start(ctx, "order", nil)
				Expect(err).ToNot(HaveOccurred())

				dispatcher.SetError(errors.New("stock unavailable"))
				err = runner.ExecuteStep(ctx, instance.ID)
				Expect(err).To(HaveOccurred())

				state, loadErr := store.Load(ctx, instance.ID)
				Expect(loadErr).ToNot(HaveOccurred())
				Expect(state.Status).To(Equal(saga.StatusFailed))
			})
		})

		Context("when a step fails after completing previous steps", func() {
			It("should compensate completed steps in reverse order", func() {
				var compensatedSteps []string
				dispatcher.failOnCall = 3
				def := &orderSaga{steps: []saga.Step{
					{
						Name:   "reserve-stock",
						Action: newBDDCommand,
						Compensate: func(_ context.Context, _ id.AggregateID) command.Command {
							compensatedSteps = append(compensatedSteps, "reserve-stock")
							return newBDDCommand(context.Background(), id.NewAggregateID())
						},
					},
					{
						Name:   "charge-payment",
						Action: newBDDCommand,
						Compensate: func(_ context.Context, _ id.AggregateID) command.Command {
							compensatedSteps = append(compensatedSteps, "charge-payment")
							return newBDDCommand(context.Background(), id.NewAggregateID())
						},
					},
					{Name: "ship-order", Action: newBDDCommand},
				}}
				Expect(runner.Register(def)).To(Succeed())

				instance, err := runner.Start(ctx, "order", nil)
				Expect(err).ToNot(HaveOccurred())

				Expect(runner.ExecuteStep(ctx, instance.ID)).To(Succeed())
				Expect(runner.ExecuteStep(ctx, instance.ID)).To(Succeed())

				err = runner.ExecuteStep(ctx, instance.ID)
				Expect(err).To(HaveOccurred())

				state, loadErr := store.Load(ctx, instance.ID)
				Expect(loadErr).ToNot(HaveOccurred())
				Expect(state.Status).To(Equal(saga.StatusFailed))

				Expect(compensatedSteps).To(Equal([]string{"charge-payment", "reserve-stock"}))
			})
		})

		Context("when a step has no compensation function", func() {
			It("should skip that step during compensation", func() {
				var compensatedSteps []string
				dispatcher.failOnCall = 3
				def := &orderSaga{steps: []saga.Step{
					{
						Name:   "reserve-stock",
						Action: newBDDCommand,
					},
					{
						Name:   "charge-payment",
						Action: newBDDCommand,
						Compensate: func(_ context.Context, _ id.AggregateID) command.Command {
							compensatedSteps = append(compensatedSteps, "charge-payment")
							return newBDDCommand(context.Background(), id.NewAggregateID())
						},
					},
					{Name: "ship-order", Action: newBDDCommand},
				}}
				Expect(runner.Register(def)).To(Succeed())

				instance, err := runner.Start(ctx, "order", nil)
				Expect(err).ToNot(HaveOccurred())

				Expect(runner.ExecuteStep(ctx, instance.ID)).To(Succeed())
				Expect(runner.ExecuteStep(ctx, instance.ID)).To(Succeed())

				_ = runner.ExecuteStep(ctx, instance.ID)

				Expect(compensatedSteps).To(Equal([]string{"charge-payment"}))
			})
		})

		Context("when my step action returns nil", func() {
			It("should return a rejection error without progressing", func() {
				def := &orderSaga{steps: []saga.Step{
					{Name: "reserve-stock", Action: func(_ context.Context, _ id.AggregateID) command.Command {
						return nil
					}},
				}}
				Expect(runner.Register(def)).To(Succeed())

				instance, err := runner.Start(ctx, "order", nil)
				Expect(err).ToNot(HaveOccurred())

				err = runner.ExecuteStep(ctx, instance.ID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("nil command"))

				state, loadErr := store.Load(ctx, instance.ID)
				Expect(loadErr).ToNot(HaveOccurred())
				Expect(state.CurrentStep).To(Equal(0))
			})
		})
	})

	Describe("As a developer managing saga definitions", func() {
		Context("when I register a nil definition", func() {
			It("should return an error", func() {
				err := runner.Register(nil)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when I register a duplicate saga type", func() {
			It("should return ErrSagaAlreadyExists", func() {
				def := &orderSaga{steps: []saga.Step{}}
				Expect(runner.Register(def)).To(Succeed())

				err := runner.Register(def)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, saga.ErrSagaAlreadyExists)).To(BeTrue())
			})
		})

		Context("when I start a saga that is not registered", func() {
			It("should return ErrSagaNotRegistered", func() {
				_, err := runner.Start(ctx, "unknown-saga", nil)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, saga.ErrSagaNotRegistered)).To(BeTrue())
			})
		})
	})

	Describe("As a developer handling initial command failures", func() {
		Context("when the initial command dispatch fails", func() {
			It("should mark the saga instance as failed", func() {
				def := &orderSaga{steps: []saga.Step{
					{Name: "reserve-stock", Action: newBDDCommand},
				}}
				Expect(runner.Register(def)).To(Succeed())

				dispatcher.SetError(errors.New("initial dispatch failed"))
				initialCmd := command.MustNew("CreateOrder", id.NewAggregateID())
				instance, err := runner.Start(ctx, "order", initialCmd)
				Expect(err).To(HaveOccurred())
				Expect(instance.Status).To(Equal(saga.StatusFailed))
			})
		})
	})

	Describe("As a developer using the saga store", func() {
		Context("when I save and load a saga state", func() {
			It("should roundtrip all fields correctly", func() {
				state := &saga.State{
					ID:          id.NewAggregateID(),
					SagaType:    "order",
					Status:      saga.StatusRunning,
					CurrentStep: 2,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				Expect(store.Save(ctx, state)).To(Succeed())

				loaded, err := store.Load(ctx, state.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(loaded.ID).To(Equal(state.ID))
				Expect(loaded.SagaType).To(Equal("order"))
				Expect(loaded.Status).To(Equal(saga.StatusRunning))
				Expect(loaded.CurrentStep).To(Equal(2))
			})
		})

		Context("when I query all running sagas", func() {
			It("should return only running and compensating instances", func() {
				running := &saga.State{
					ID: id.NewAggregateID(), SagaType: "order",
					Status: saga.StatusRunning,
				}
				completed := &saga.State{
					ID: id.NewAggregateID(), SagaType: "order",
					Status: saga.StatusCompleted,
				}
				compensating := &saga.State{
					ID: id.NewAggregateID(), SagaType: "order",
					Status: saga.StatusCompensating,
				}

				Expect(store.Save(ctx, running)).To(Succeed())
				Expect(store.Save(ctx, completed)).To(Succeed())
				Expect(store.Save(ctx, compensating)).To(Succeed())

				runningStates, err := store.LoadAllRunning(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(runningStates).To(HaveLen(2))

				for _, s := range runningStates {
					Expect(s.Status).ToNot(Equal(saga.StatusCompleted))
				}
			})
		})
	})
})
