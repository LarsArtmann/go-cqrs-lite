package aggregate_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

type product struct {
	*aggregate.Core

	name     string
	price    float64
	quantity int
}

const productType event.AggregateType = "Product"

var _ aggregate.Root = (*product)(nil)

func newProduct(productID id.AggregateID) *product {
	return &product{Core: aggregate.MustNewCore(productID, productType)}
}

func (p *product) Apply(evt event.Event) error {
	switch evt.Type() {
	case "ProductCreated":
		var payload struct {
			Name  string  `json:"name"`
			Price float64 `json:"price"`
		}

		err := json.Unmarshal(evt.Payload(), &payload)
		if err != nil {
			return err
		}

		p.name = payload.Name
		p.price = payload.Price

	case "ProductRestocked":
		var payload struct {
			Quantity int `json:"quantity"`
		}

		err := json.Unmarshal(evt.Payload(), &payload)
		if err != nil {
			return err
		}

		p.quantity += payload.Quantity
	}

	return nil
}

func (p *product) ApplySnapshot(_ []byte) error {
	return nil
}

func (p *product) LoadEvents(events []event.Event) error {
	return p.LoadFromHistory(p, events)
}

// newEvent creates a new event for the product aggregate.
func (p *product) newEvent(eventType string, payload []byte) (*event.Core, error) {
	return event.NewEvent(
		event.Type(eventType),
		p.ID(),
		productType,
		p.Version().Int()+1,
		payload,
	)
}

func (p *product) Create(ctx context.Context, name string, price float64) error {
	payload, err := json.Marshal(struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}{Name: name, Price: price})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	evt, err := p.newEvent("ProductCreated", payload)
	if err != nil {
		return err
	}

	p.name = name
	p.price = price
	p.RecordEvent(ctx, evt)

	return nil
}

func (p *product) Restock(ctx context.Context, quantity int) error {
	payload, err := json.Marshal(struct {
		Quantity int `json:"quantity"`
	}{Quantity: quantity})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	evt, err := p.newEvent("ProductRestocked", payload)
	if err != nil {
		return err
	}

	p.quantity += quantity
	p.RecordEvent(ctx, evt)

	return nil
}

type createProductCmd struct {
	aggregateID id.AggregateID
	name        string
	price       float64
}

func (c *createProductCmd) Type() command.Type          { return "product.create" }
func (c *createProductCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *createProductCmd) IdempotencyKey() string       { return "" }

type restockProductCmd struct {
	aggregateID id.AggregateID
	quantity    int
}

func (c *restockProductCmd) Type() command.Type          { return "product.restock" }
func (c *restockProductCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *restockProductCmd) IdempotencyKey() string       { return "" }

func TestCQRSRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	repo, _ := aggregate.NewRepository(store, bus)
	dispatcher := command.NewDispatcher()

	var busEvents []event.Event

	_ = bus.SubscribeAll(testhelpers.AppendEventsHandler(&busEvents))

	err := dispatcher.Register(
		"product.create",
		func(_ context.Context, cmd command.Command) error {
			c := cmd.(*createProductCmd) //nolint:forcetypeassert

			p := newProduct(c.aggregateID)

			err := p.Create(ctx, c.name, c.price)
			if err != nil {
				return err
			}

			return repo.Save(ctx, p)
		},
	)
	if err != nil {
		t.Fatalf("register create handler: %v", err)
	}

	err = dispatcher.Register(
		"product.restock",
		func(_ context.Context, cmd command.Command) error {
			c := cmd.(*restockProductCmd) //nolint:forcetypeassert

			p := newProduct(c.aggregateID)

			err := repo.Load(ctx, p)
			if err != nil {
				return err
			}

			err = p.Restock(ctx, c.quantity)
			if err != nil {
				return err
			}

			return repo.Save(ctx, p)
		},
	)
	if err != nil {
		t.Fatalf("register restock handler: %v", err)
	}

	productID := id.NewAggregateID()

	err = dispatcher.Dispatch(ctx, &createProductCmd{
		aggregateID: productID,
		name:        "Widget",
		price:       9.99,
	})
	if err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	if len(busEvents) != 1 {
		t.Fatalf("expected 1 bus event after create, got %d", len(busEvents))
	}

	if busEvents[0].Type() != "ProductCreated" {
		t.Errorf("expected ProductCreated event, got %s", busEvents[0].Type())
	}

	err = dispatcher.Dispatch(ctx, &restockProductCmd{
		aggregateID: productID,
		quantity:    100,
	})
	if err != nil {
		t.Fatalf("dispatch restock: %v", err)
	}

	if len(busEvents) != 2 {
		t.Fatalf("expected 2 bus events, got %d", len(busEvents))
	}

	loaded := newProduct(productID)

	err = repo.Load(ctx, loaded)
	if err != nil {
		t.Fatalf("load product: %v", err)
	}

	if loaded.Version() != 2 {
		t.Errorf("expected version 2, got %d", loaded.Version())
	}

	if loaded.name != "Widget" {
		t.Errorf("expected name Widget, got %s", loaded.name)
	}

	if loaded.price != 9.99 {
		t.Errorf("expected price 9.99, got %f", loaded.price)
	}

	if loaded.quantity != 100 {
		t.Errorf("expected quantity 100, got %d", loaded.quantity)
	}
}
