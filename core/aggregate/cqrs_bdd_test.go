package aggregate_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// expense is a test aggregate for BDD scenarios.
type expense struct {
	*aggregate.Core

	description string
	amount      float64
	approved    bool
	paid        bool
}

const expenseType event.AggregateType = "Expense"

var (
	_ aggregate.Root          = (*expense)(nil)
	_ aggregate.HistoryLoader = (*expense)(nil)
)

func newExpense(expenseID id.AggregateID) *expense {
	return &expense{Core: aggregate.NewCore(expenseID, expenseType)}
}

func (e *expense) Apply(evt event.Event) error {
	switch evt.Type() {
	case "ExpenseSubmitted":
		var p struct {
			Description string  `json:"description"`
			Amount      float64 `json:"amount"`
		}

		err := json.Unmarshal(evt.Payload(), &p)
		if err != nil {
			return err
		}

		e.description = p.Description
		e.amount = p.Amount
	case "ExpenseApproved":
		e.approved = true
	case "ExpensePaid":
		e.paid = true
	}

	return nil
}

func (e *expense) LoadEvents(events []event.Event) error {
	return e.LoadFromHistory(e, events)
}

func (e *expense) Submit(ctx context.Context, description string, amount float64) error {
	payload, _ := json.Marshal(struct {
		Description string  `json:"description"`
		Amount      float64 `json:"amount"`
	}{Description: description, Amount: amount})

	evt, err := event.NewEvent(
		"ExpenseSubmitted",
		id.MustParseAggregateID(e.ID()),
		expenseType,
		e.Version()+1,
		payload,
	)
	if err != nil {
		return err
	}

	e.description = description
	e.amount = amount
	e.RecordEvent(ctx, evt)

	return nil
}

func (e *expense) Approve(ctx context.Context) error {
	evt, err := event.NewEvent(
		"ExpenseApproved",
		id.MustParseAggregateID(e.ID()),
		expenseType,
		e.Version()+1,
		nil,
	)
	if err != nil {
		return err
	}

	e.approved = true
	e.RecordEvent(ctx, evt)

	return nil
}

func (e *expense) Pay(ctx context.Context) error {
	evt, err := event.NewEvent(
		"ExpensePaid",
		id.MustParseAggregateID(e.ID()),
		expenseType,
		e.Version()+1,
		nil,
	)
	if err != nil {
		return err
	}

	e.paid = true
	e.RecordEvent(ctx, evt)

	return nil
}

// Command types.
type submitExpenseCmd struct {
	id          id.AggregateID
	description string
	amount      float64
}
type (
	approveExpenseCmd struct{ id id.AggregateID }
	payExpenseCmd     struct{ id id.AggregateID }
)

func (c *submitExpenseCmd) Type() command.Type   { return "expense.submit" }
func (c *submitExpenseCmd) AggregateID() string  { return c.id.String() }
func (c *approveExpenseCmd) Type() command.Type  { return "expense.approve" }
func (c *approveExpenseCmd) AggregateID() string { return c.id.String() }
func (c *payExpenseCmd) Type() command.Type      { return "expense.pay" }
func (c *payExpenseCmd) AggregateID() string     { return c.id.String() }

var _ = Describe("CQRS Flow", func() {
	var (
		ctx        context.Context
		store      *memory.MemoryStore
		bus        *memory.MemoryBus
		repo       *aggregate.EventSourcedRepository
		dispatcher *command.Dispatcher
		busEvents  []event.Event
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = memory.NewMemoryStore()
		bus = memory.NewMemoryBus()
		repo = aggregate.NewRepository(store, bus)
		dispatcher = command.NewDispatcher()
		busEvents = nil

		_ = bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
			busEvents = append(busEvents, evt)

			return nil
		})
	})

	Describe("As a developer building a CQRS application", func() {
		Context("when I submit a new expense", func() {
			It("should persist the event and publish it to the bus", func() {
				expenseID := id.NewAggregateID()

				Expect(
					dispatcher.Register(
						"expense.submit",
						func(_ context.Context, cmd command.Command) error {
							c := cmd.(*submitExpenseCmd)
							e := newExpense(c.id)
							Expect(e.Submit(ctx, c.description, c.amount)).To(Succeed())

							return repo.Save(ctx, e)
						},
					),
				).To(Succeed())

				Expect(dispatcher.Dispatch(ctx, &submitExpenseCmd{
					id: expenseID, description: "Flight to Berlin", amount: 349.50,
				})).To(Succeed())

				Expect(busEvents).To(HaveLen(1))
				Expect(busEvents[0].Type()).To(Equal(event.Type("ExpenseSubmitted")))

				loaded := newExpense(expenseID)
				Expect(repo.Load(ctx, loaded)).To(Succeed())
				Expect(loaded.Version()).To(Equal(1))
				Expect(loaded.description).To(Equal("Flight to Berlin"))
				Expect(loaded.amount).To(Equal(349.50))
			})
		})

		Context("when I approve and pay an expense", func() {
			var expenseID id.AggregateID

			BeforeEach(func() {
				expenseID = id.NewAggregateID()

				Expect(
					dispatcher.Register(
						"expense.submit",
						func(_ context.Context, cmd command.Command) error {
							c := cmd.(*submitExpenseCmd)
							e := newExpense(c.id)
							Expect(e.Submit(ctx, c.description, c.amount)).To(Succeed())

							return repo.Save(ctx, e)
						},
					),
				).To(Succeed())

				Expect(
					dispatcher.Register(
						"expense.approve",
						func(_ context.Context, cmd command.Command) error {
							c := cmd.(*approveExpenseCmd)
							e := newExpense(c.id)
							Expect(repo.Load(ctx, e)).To(Succeed())
							Expect(e.Approve(ctx)).To(Succeed())

							return repo.Save(ctx, e)
						},
					),
				).To(Succeed())

				Expect(
					dispatcher.Register(
						"expense.pay",
						func(_ context.Context, cmd command.Command) error {
							c := cmd.(*payExpenseCmd)
							e := newExpense(c.id)
							Expect(repo.Load(ctx, e)).To(Succeed())
							Expect(e.Pay(ctx)).To(Succeed())

							return repo.Save(ctx, e)
						},
					),
				).To(Succeed())
			})

			It("should maintain a complete audit trail through events", func() {
				Expect(dispatcher.Dispatch(ctx, &submitExpenseCmd{
					id: expenseID, description: "Team dinner", amount: 120.00,
				})).To(Succeed())

				Expect(dispatcher.Dispatch(ctx, &approveExpenseCmd{id: expenseID})).To(Succeed())
				Expect(dispatcher.Dispatch(ctx, &payExpenseCmd{id: expenseID})).To(Succeed())

				Expect(busEvents).To(HaveLen(3))
				Expect(busEvents[0].Type()).To(Equal(event.Type("ExpenseSubmitted")))
				Expect(busEvents[1].Type()).To(Equal(event.Type("ExpenseApproved")))
				Expect(busEvents[2].Type()).To(Equal(event.Type("ExpensePaid")))

				loaded := newExpense(expenseID)
				Expect(repo.Load(ctx, loaded)).To(Succeed())
				Expect(loaded.Version()).To(Equal(3))
				Expect(loaded.description).To(Equal("Team dinner"))
				Expect(loaded.approved).To(BeTrue())
				Expect(loaded.paid).To(BeTrue())
			})

			It("should replay state correctly from the event store alone", func() {
				Expect(dispatcher.Dispatch(ctx, &submitExpenseCmd{
					id: expenseID, description: "Uber ride", amount: 24.99,
				})).To(Succeed())
				Expect(dispatcher.Dispatch(ctx, &approveExpenseCmd{id: expenseID})).To(Succeed())

				fresh := newExpense(expenseID)
				Expect(repo.Load(ctx, fresh)).To(Succeed())
				Expect(fresh.Version()).To(Equal(2))
				Expect(fresh.description).To(Equal("Uber ride"))
				Expect(fresh.approved).To(BeTrue())
				Expect(fresh.paid).To(BeFalse())
			})
		})

		Context("when I dispatch an unregistered command", func() {
			It("should return a handler not found error", func() {
				err := dispatcher.Dispatch(ctx, &submitExpenseCmd{id: id.NewAggregateID()})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("handler not found"))
			})
		})

		Context("when I close the dispatcher", func() {
			It("should reject registration and dispatch", func() {
				Expect(dispatcher.Close()).To(Succeed())

				err := dispatcher.Register(
					"expense.submit",
					func(_ context.Context, cmd command.Command) error {
						return nil
					},
				)
				Expect(err).To(HaveOccurred())

				err = dispatcher.Dispatch(ctx, &submitExpenseCmd{id: id.NewAggregateID()})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when I add command middleware", func() {
			It("should wrap the handler in order", func() {
				var callOrder []string

				dispatcher.Use(func(next command.Handler) command.Handler {
					return func(ctx context.Context, cmd command.Command) error {
						callOrder = append(callOrder, "audit")

						return next(ctx, cmd)
					}
				})

				expenseID := id.NewAggregateID()

				Expect(
					dispatcher.Register(
						"expense.submit",
						func(_ context.Context, cmd command.Command) error {
							c := cmd.(*submitExpenseCmd)
							e := newExpense(c.id)
							Expect(e.Submit(ctx, c.description, c.amount)).To(Succeed())

							callOrder = append(callOrder, "handler")

							return repo.Save(ctx, e)
						},
					),
				).To(Succeed())

				Expect(dispatcher.Dispatch(ctx, &submitExpenseCmd{
					id: expenseID, description: "Audited expense", amount: 42.00,
				})).To(Succeed())

				Expect(callOrder).To(Equal([]string{"audit", "handler"}))

				loaded := newExpense(expenseID)
				Expect(repo.Load(ctx, loaded)).To(Succeed())
				Expect(loaded.description).To(Equal("Audited expense"))
			})
		})
	})
})

var _ = Describe("Aggregate Repository", func() {
	var (
		ctx   context.Context
		store *memory.MemoryStore
		bus   *memory.MemoryBus
		repo  *aggregate.EventSourcedRepository
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = memory.NewMemoryStore()
		bus = memory.NewMemoryBus()
		repo = aggregate.NewRepository(store, bus)
	})

	Describe("As a developer managing aggregate lifecycle", func() {
		Context("when I save an aggregate with no changes", func() {
			It("should be a no-op without error", func() {
				e := newExpense(id.NewAggregateID())
				Expect(repo.Save(ctx, e)).To(Succeed())
			})
		})

		Context("when I load a non-existent aggregate", func() {
			It("should return an error", func() {
				e := newExpense(id.NewAggregateID())
				err := repo.Load(ctx, e)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when I save and then load an aggregate multiple times", func() {
			It("should correctly accumulate events across save/load cycles", func() {
				expenseID := id.NewAggregateID()

				e := newExpense(expenseID)
				Expect(e.Submit(ctx, "Lunch", 15.00)).To(Succeed())
				Expect(repo.Save(ctx, e)).To(Succeed())
				Expect(e.UncommittedChanges()).To(BeEmpty())

				loaded := newExpense(expenseID)
				Expect(repo.Load(ctx, loaded)).To(Succeed())
				Expect(loaded.Approve(ctx)).To(Succeed())
				Expect(repo.Save(ctx, loaded)).To(Succeed())

				final := newExpense(expenseID)
				Expect(repo.Load(ctx, final)).To(Succeed())
				Expect(final.Version()).To(Equal(2))
				Expect(final.description).To(Equal("Lunch"))
				Expect(final.approved).To(BeTrue())
			})
		})

		Context("when I delete an aggregate and then recreate it", func() {
			It("should accept new events starting from version 1", func() {
				expenseID := id.NewAggregateID()

				e := newExpense(expenseID)
				Expect(e.Submit(ctx, "Original", 100.00)).To(Succeed())
				Expect(repo.Save(ctx, e)).To(Succeed())

				Expect(store.Delete(ctx, expenseType, expenseID)).To(Succeed())

				fresh := newExpense(expenseID)
				Expect(fresh.Submit(ctx, "Recreated", 200.00)).To(Succeed())
				Expect(repo.Save(ctx, fresh)).To(Succeed())

				loaded := newExpense(expenseID)
				Expect(repo.Load(ctx, loaded)).To(Succeed())
				Expect(loaded.Version()).To(Equal(1))
				Expect(loaded.description).To(Equal("Recreated"))
				Expect(loaded.amount).To(Equal(200.00))
			})
		})
	})
})

var _ = Describe("CQRS Concurrency and Invariants", func() {
	var (
		ctx   context.Context
		store *memory.MemoryStore
		bus   *memory.MemoryBus
		repo  *aggregate.EventSourcedRepository
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = memory.NewMemoryStore()
		bus = memory.NewMemoryBus()
		repo = aggregate.NewRepository(store, bus)
	})

	Describe("As a developer verifying domain correctness", func() {
		Context("when multiple goroutines save the same aggregate concurrently", func() {
			It("should serialize all writes and produce a consistent event stream", func() {
				expenseID := id.NewAggregateID()

				e := newExpense(expenseID)
				Expect(e.Submit(ctx, "Concurrent test", 50.00)).To(Succeed())
				Expect(repo.Save(ctx, e)).To(Succeed())

				const goroutines = 20

				var (
					wg               sync.WaitGroup
					versionConflicts atomic.Int32
					successes        atomic.Int32
				)

				wg.Add(goroutines)

				for range goroutines {
					go func() {
						defer wg.Done()

						local := newExpense(expenseID)

						err := repo.Load(ctx, local)
						if err != nil {
							return
						}

						_ = local.Approve(ctx)
						if err := repo.Save(ctx, local); err != nil {
							versionConflicts.Add(1)
						} else {
							successes.Add(1)
						}
					}()
				}

				wg.Wait()

				Expect(int(successes.Load()) + int(versionConflicts.Load())).To(Equal(goroutines))

				final := newExpense(expenseID)
				Expect(repo.Load(ctx, final)).To(Succeed())
				Expect(final.Version()).To(BeNumerically(">=", 2))

				events, err := store.Load(ctx, expenseType, expenseID)
				Expect(err).ToNot(HaveOccurred())
				Expect(events).To(HaveLen(final.Version()))
			})
		})

		Context("when I pay an expense without approving it first", func() {
			It("should succeed because the aggregate enforces no approval invariant", func() {
				expenseID := id.NewAggregateID()

				e := newExpense(expenseID)
				Expect(e.Submit(ctx, "Skip approval", 75.00)).To(Succeed())
				Expect(repo.Save(ctx, e)).To(Succeed())

				loaded := newExpense(expenseID)
				Expect(repo.Load(ctx, loaded)).To(Succeed())
				Expect(loaded.Pay(ctx)).To(Succeed())
				Expect(repo.Save(ctx, loaded)).To(Succeed())

				final := newExpense(expenseID)
				Expect(repo.Load(ctx, final)).To(Succeed())
				Expect(final.Version()).To(Equal(2))
				Expect(final.approved).To(BeFalse())
				Expect(final.paid).To(BeTrue())
			})
		})
	})
})
