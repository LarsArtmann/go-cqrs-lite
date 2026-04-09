package aggregate_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/aggregate"
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

type product struct {
	*aggregate.Core

	name     string
	price    float64
	quantity int
}

const productType event.AggregateType = "Product"

var (
	_ aggregate.Root          = (*product)(nil)
	_ aggregate.HistoryLoader = (*product)(nil)
)

func newProduct(productID id.AggregateID) *product {
	return &product{Core: aggregate.NewCore(productID, productType)}
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

func (p *product) LoadEvents(events []event.Event) error {
	return p.LoadFromHistory(p, events)
}

func (p *product) Create(ctx context.Context, name string, price float64) error {
	payload, _ := json.Marshal(struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}{Name: name, Price: price})

	evt, err := event.NewEvent(
		"ProductCreated",
		id.MustParseAggregateID(p.ID()),
		productType,
		p.Version()+1,
		payload,
	)
	if err != nil {
		return err
	}

	p.name = name
	p.price = price
	p.RecordEvent(ctx, evt)

	return nil
}

func (p *product) Restock(ctx context.Context, quantity int) error {
	payload, _ := json.Marshal(struct {
		Quantity int `json:"quantity"`
	}{Quantity: quantity})

	evt, err := event.NewEvent(
		"ProductRestocked",
		id.MustParseAggregateID(p.ID()),
		productType,
		p.Version()+1,
		payload,
	)
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

func (c *createProductCmd) Type() command.Type  { return "product.create" }
func (c *createProductCmd) AggregateID() string { return c.aggregateID.String() }

type restockProductCmd struct {
	aggregateID id.AggregateID
	quantity    int
}

func (c *restockProductCmd) Type() command.Type  { return "product.restock" }
func (c *restockProductCmd) AggregateID() string { return c.aggregateID.String() }

func TestCQRSRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store := event.NewMemoryStore()
	bus := event.NewMemoryBus()
	repo := aggregate.NewRepository(store, bus)
	dispatcher := command.NewDispatcher()

	var busEvents []event.Event

	_ = bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		busEvents = append(busEvents, evt)

		return nil
	})

	err := dispatcher.Register(
		"product.create",
		func(_ context.Context, cmd command.Command) error {
			c := cmd.(*createProductCmd)

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
			c := cmd.(*restockProductCmd)

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
