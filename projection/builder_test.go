package projection_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

func TestBuilder_On_TypedHandler(t *testing.T) {
	t.Parallel()

	type OrderCreated struct {
		OrderID string `json:"order_id"`
		Product string `json:"product"`
	}

	var received OrderCreated

	builder := projection.NewBuilder("order-proj")

	err := projection.On(
		builder, "order.created", codec.JSONCodec{},
		func(_ context.Context, evt OrderCreated) error {
			received = evt
			return nil
		},
	)
	if err != nil {
		t.Fatalf("On: %v", err)
	}

	proj := builder.Build()
	if proj.Name() != "order-proj" {
		t.Errorf("Name = %q, want order-proj", proj.Name())
	}

	types := proj.EventTypes()
	if len(types) != 1 || types[0] != "order.created" {
		t.Errorf("EventTypes = %v, want [order.created]", types)
	}

	payload := `{"order_id":"abc","product":"widget"}`

	evt, err := event.New(
		"order.created",
		id.NewAggregateID(),
		"Order",
		1,
		[]byte(payload),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = proj.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if received.OrderID != "abc" {
		t.Errorf("OrderID = %q, want abc", received.OrderID)
	}

	if received.Product != "widget" {
		t.Errorf("Product = %q, want widget", received.Product)
	}
}

func TestBuilder_On_MultipleEventTypes(t *testing.T) {
	t.Parallel()

	type OrderCreated struct {
		OrderID string `json:"order_id"`
	}

	type OrderShipped struct {
		TrackingCode string `json:"tracking_code"`
	}

	handled := make(chan string, 2)

	builder := projection.NewBuilder("multi-proj")

	err := projection.On(
		builder, "order.created", codec.JSONCodec{},
		func(_ context.Context, _ OrderCreated) error {
			handled <- "order.created"
			return nil
		},
	)
	if err != nil {
		t.Fatalf("On OrderCreated: %v", err)
	}

	err = projection.On(
		builder, "order.shipped", codec.JSONCodec{},
		func(_ context.Context, _ OrderShipped) error {
			handled <- "order.shipped"
			return nil
		},
	)
	if err != nil {
		t.Fatalf("On OrderShipped: %v", err)
	}

	proj := builder.Build()

	types := proj.EventTypes()
	if len(types) != 2 {
		t.Fatalf("EventTypes len = %d, want 2", len(types))
	}

	evt1, err := event.New(
		"order.created",
		id.NewAggregateID(), "Order", 1,
		[]byte(`{"order_id":"o1"}`),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	evt2, err := event.New(
		"order.shipped",
		id.NewAggregateID(), "Order", 2,
		[]byte(`{"tracking_code":"TC123"}`),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = proj.Handle(context.Background(), evt1)
	if err != nil {
		t.Fatalf("Handle evt1: %v", err)
	}

	err = proj.Handle(context.Background(), evt2)
	if err != nil {
		t.Fatalf("Handle evt2: %v", err)
	}

	drainChan(t, handled, 2, "handler")
}

func TestBuilder_On_NilHandler(t *testing.T) {
	t.Parallel()

	type dummy struct{}

	builder := projection.NewBuilder("nil-test")

	err := projection.On[dummy](builder, "test.event", codec.JSONCodec{}, nil)
	if err == nil {
		t.Fatal("expected error for nil handler")
	}
}

func TestBuilder_Build_Empty(t *testing.T) {
	t.Parallel()

	builder := projection.NewBuilder("empty-proj")
	proj := builder.Build()

	if proj.Name() != "empty-proj" {
		t.Errorf("Name = %q, want empty-proj", proj.Name())
	}

	types := proj.EventTypes()
	if len(types) != 0 {
		t.Errorf("EventTypes = %v, want empty", types)
	}
}

func TestBuilder_Handle_UnregisteredType(t *testing.T) {
	t.Parallel()

	type OrderCreated struct{}

	builder := projection.NewBuilder("filter-proj")

	handled := false

	err := projection.On(
		builder, "order.created", codec.JSONCodec{}, func(_ context.Context, _ OrderCreated) error {
			handled = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("On: %v", err)
	}

	proj := builder.Build()

	otherEvt, err := event.New(
		"user.created",
		id.NewAggregateID(), "User", 1,
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = proj.Handle(context.Background(), otherEvt)
	if err != nil {
		t.Fatalf("Handle unregistered type: %v", err)
	}

	if handled {
		t.Error("handler should not be called for unregistered event type")
	}
}

func TestBuilder_Handle_InvalidPayload(t *testing.T) {
	t.Parallel()

	type Strict struct {
		Count int `json:"count"`
	}

	builder := projection.NewBuilder("decode-proj")

	err := projection.On(
		builder, "bad.payload", codec.JSONCodec{},
		func(_ context.Context, _ Strict) error { return nil },
	)
	if err != nil {
		t.Fatalf("On: %v", err)
	}

	proj := builder.Build()

	evt, err := event.New(
		"bad.payload",
		id.NewAggregateID(), "Test", 1,
		[]byte(`not-json`),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = proj.Handle(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error for invalid JSON payload")
	}
}

func TestBuilder_EventTypesIsolation(t *testing.T) {
	t.Parallel()

	type OrderCreated struct{}

	builder := projection.NewBuilder("isolation-proj")

	err := projection.On(
		builder, "order.created", codec.JSONCodec{},
		func(_ context.Context, _ OrderCreated) error { return nil },
	)
	if err != nil {
		t.Fatalf("On: %v", err)
	}

	proj := builder.Build()

	returned := proj.EventTypes()
	returned[0] = "MUTATED"

	again := proj.EventTypes()
	if again[0] == "MUTATED" {
		t.Error("mutating EventTypes() return value should not affect internal state")
	}
}

func TestBuilder_BuildIsolation(t *testing.T) {
	t.Parallel()

	type OrderCreated struct{}
	type OrderShipped struct{}

	builder := projection.NewBuilder("build-isolation-proj")

	err := projection.On(
		builder, "order.created", codec.JSONCodec{},
		func(_ context.Context, _ OrderCreated) error { return nil },
	)
	if err != nil {
		t.Fatalf("On: %v", err)
	}

	proj1 := builder.Build()

	err = projection.On(
		builder, "order.shipped", codec.JSONCodec{},
		func(_ context.Context, _ OrderShipped) error { return nil },
	)
	if err != nil {
		t.Fatalf("On second: %v", err)
	}

	proj1Types := proj1.EventTypes()
	if len(proj1Types) != 1 || proj1Types[0] != "order.created" {
		t.Errorf("proj1 EventTypes = %v, want [order.created] — Build() should isolate from builder reuse", proj1Types)
	}
}
