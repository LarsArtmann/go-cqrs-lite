package bench

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// realistic_models_test.go — 3 multi-domain types representing real bounded
// contexts. These exercise different payload sizes, access patterns, and event
// types that real CQRS systems encounter.

// ─── Domain 1: Order (e-commerce, ~512B payloads) ───

type OrderState struct {
	ID       string
	Status   string
	Total    int64
	Items    int
	Customer string
}

type OrderCreatedEvt struct {
	ID       string    `json:"id"`
	Customer string    `json:"customer"`
	Amount   int64     `json:"amount"`
	Product  string    `json:"product"`
	At       time.Time `json:"at"`
}

type OrderItemAddedEvt struct {
	OrderID string `json:"orderId"`
	Product string `json:"product"`
	Price   int64  `json:"price"`
}

type OrderView struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
	Status   string `json:"status"`
	Total    int64  `json:"total"`
	Items    int    `json:"items"`
}

func foldOrder(state OrderState, evt event.Event) (OrderState, error) {
	switch evt.Type() {
	case "order.created":
		p, err := event.DecodePayloadAuto[OrderCreatedEvt](evt)
		if err != nil {
			return state, err
		}
		state.ID = p.ID
		state.Status = "pending"
		state.Total = p.Amount
		state.Items = 1
		state.Customer = p.Customer
	case "order.item_added":
		p, err := event.DecodePayloadAuto[OrderItemAddedEvt](evt)
		if err != nil {
			return state, err
		}
		state.Total += p.Price
		state.Items++
	}
	return state, nil
}

func orderProjection(
	ctx context.Context,
	evt event.Event,
	store *kv.TypedStore[OrderView, id.StreamID],
) error {
	switch evt.Type() {
	case "order.created":
		p, err := event.DecodePayloadAuto[OrderCreatedEvt](evt)
		if err != nil {
			return err
		}
		return store.Set(ctx, evt.StreamID(), &OrderView{
			ID: p.ID, Customer: p.Customer, Status: "pending", Total: p.Amount, Items: 1,
		})
	case "order.item_added":
		p, err := event.DecodePayloadAuto[OrderItemAddedEvt](evt)
		if err != nil {
			return err
		}
		existing, err := store.Get(ctx, evt.StreamID())
		if err != nil || existing == nil {
			return nil
		}
		existing.Total += p.Price
		existing.Items++
		return store.Set(ctx, evt.StreamID(), existing)
	}
	return nil
}

// ─── Domain 2: Task (SaaS, ~256B payloads) ───

type TaskState struct {
	ID       string
	Title    string
	Assignee string
	Status   string
	Done     bool
}

type TaskCreatedEvt struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Owner  string `json:"owner"`
	Status string `json:"status"`
}

type TaskCompletedEvt struct {
	ID string `json:"id"`
}

type TaskView struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Assignee string `json:"assignee"`
	Status   string `json:"status"`
}

func foldTask(state TaskState, evt event.Event) (TaskState, error) {
	switch evt.Type() {
	case "task.created":
		p, err := event.DecodePayloadAuto[TaskCreatedEvt](evt)
		if err != nil {
			return state, err
		}
		state.ID = p.ID
		state.Title = p.Title
		state.Assignee = p.Owner
		state.Status = p.Status
	case "task.completed":
		state.Status = "completed"
		state.Done = true
	}
	return state, nil
}

func taskProjection(
	ctx context.Context,
	evt event.Event,
	store *kv.TypedStore[TaskView, id.StreamID],
) error {
	switch evt.Type() {
	case "task.created":
		p, err := event.DecodePayloadAuto[TaskCreatedEvt](evt)
		if err != nil {
			return err
		}
		return store.Set(ctx, evt.StreamID(), &TaskView{
			ID: p.ID, Title: p.Title, Assignee: p.Owner, Status: p.Status,
		})
	case "task.completed":
		existing, err := store.Get(ctx, evt.StreamID())
		if err != nil || existing == nil {
			return nil
		}
		existing.Status = "completed"
		return store.Set(ctx, evt.StreamID(), existing)
	}
	return nil
}

// ─── Domain 3: User (auth, ~128B payloads) ───

type UserState struct {
	ID     string
	Email  string
	Name   string
	Active bool
}

type UserRegisteredEvt struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type UserActivatedEvt struct {
	ID string `json:"id"`
}

type UserView struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func foldUser(state UserState, evt event.Event) (UserState, error) {
	switch evt.Type() {
	case "user.registered":
		p, err := event.DecodePayloadAuto[UserRegisteredEvt](evt)
		if err != nil {
			return state, err
		}
		state.ID = p.ID
		state.Email = p.Email
		state.Name = p.Name
		state.Active = false
	case "user.activated":
		state.Active = true
	}
	return state, nil
}

func userProjection(
	ctx context.Context,
	evt event.Event,
	store *kv.TypedStore[UserView, id.StreamID],
) error {
	switch evt.Type() {
	case "user.registered":
		p, err := event.DecodePayloadAuto[UserRegisteredEvt](evt)
		if err != nil {
			return err
		}
		return store.Set(ctx, evt.StreamID(), &UserView{
			ID: p.ID, Email: p.Email, Name: p.Name, Active: false,
		})
	case "user.activated":
		existing, err := store.Get(ctx, evt.StreamID())
		if err != nil || existing == nil {
			return nil
		}
		existing.Active = true
		return store.Set(ctx, evt.StreamID(), existing)
	}
	return nil
}

// ─── Mixed-Domain Helpers ───

// allDomainEventTypes returns the event types handled across all 3 domains.
func allDomainEventTypes() []event.Type {
	return []event.Type{
		"order.created", "order.item_added",
		"task.created", "task.completed",
		"user.registered", "user.activated",
	}
}

// multiDomainProjection routes events to the correct domain projection.
func newMultiDomainProjection(
	orderStore *kv.TypedStore[OrderView, id.StreamID],
	taskStore *kv.TypedStore[TaskView, id.StreamID],
	userStore *kv.TypedStore[UserView, id.StreamID],
) func(context.Context, event.Event) error {
	return func(ctx context.Context, evt event.Event) error {
		switch evt.Type() {
		case "order.created", "order.item_added":
			return orderProjection(ctx, evt, orderStore)
		case "task.created", "task.completed":
			return taskProjection(ctx, evt, taskStore)
		case "user.registered", "user.activated":
			return userProjection(ctx, evt, userStore)
		}
		return nil
	}
}
