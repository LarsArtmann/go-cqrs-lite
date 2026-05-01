package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestUpcasterFunc(t *testing.T) {
	t.Parallel()

	upcaster := event.NewUpcaster(
		"UserCreated",
		1,
		func(_ event.Event) (*event.Core, error) {
			aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

			return event.NewEvent("UserCreated", aggID, "User", 2, nil)
		},
	)

	if upcaster.SourceType() != "UserCreated" {
		t.Errorf("SourceType = %q, want UserCreated", upcaster.SourceType())
	}

	if upcaster.SourceVersion() != 1 {
		t.Errorf("SourceVersion = %d, want 1", upcaster.SourceVersion())
	}
}

func TestUpcasterRegistry_NoUpcasters(t *testing.T) {
	t.Parallel()

	registry := event.NewUpcasterRegistry()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{"name":"Alice"}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	result, err := registry.Upcast(evt)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}

	if result.ID() != evt.ID() {
		t.Error("expected same event when no upcasters registered")
	}
}

func TestUpcasterRegistry_SingleUpcaster(t *testing.T) {
	t.Parallel()

	registry := event.NewUpcasterRegistry()

	registry.Register(event.NewUpcaster(
		"UserCreated",
		1,
		func(evt event.Event) (*event.Core, error) {
			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 2,
				[]byte(`{"name":"Alice","email":""}`))
		},
	))

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{"name":"Alice"}`))

	result, err := registry.Upcast(evt)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}

	if string(result.Payload()) != `{"name":"Alice","email":""}` {
		t.Errorf("payload = %q, want enriched payload", string(result.Payload()))
	}
}

func TestUpcasterRegistry_ChainedUpcasters(t *testing.T) {
	t.Parallel()

	registry := event.NewUpcasterRegistry()

	registry.Register(event.NewUpcaster("UserCreated", 1,
		func(evt event.Event) (*event.Core, error) {
			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 2,
				[]byte(`{"name":"Alice","email":""}`))
		},
	))

	registry.Register(event.NewUpcaster("UserCreated", 2,
		func(evt event.Event) (*event.Core, error) {
			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 3,
				[]byte(`{"name":"Alice","email":"","active":true}`))
		},
	))

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{"name":"Alice"}`))

	result, err := registry.Upcast(evt)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}

	want := `{"name":"Alice","email":"","active":true}`
	if string(result.Payload()) != want {
		t.Errorf("payload = %q, want %q", string(result.Payload()), want)
	}
}

func TestUpcasterRegistry_DifferentEventTypes(t *testing.T) {
	t.Parallel()

	registry := event.NewUpcasterRegistry()

	registry.Register(event.NewUpcaster("UserCreated", 1,
		func(evt event.Event) (*event.Core, error) {
			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 2, nil)
		},
	))

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("OrderPlaced", aggID, "Order", 1, []byte(`{}`))

	result, err := registry.Upcast(evt)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}

	if result.ID() != evt.ID() {
		t.Error("expected unchanged event for different type")
	}
}

func TestUpcasterRegistry_VersionSorting(t *testing.T) {
	t.Parallel()

	registry := event.NewUpcasterRegistry()

	var applied []int

	registry.Register(event.NewUpcaster("UserCreated", 2,
		func(evt event.Event) (*event.Core, error) {
			applied = append(applied, 2)

			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 3, nil)
		},
	))

	registry.Register(event.NewUpcaster("UserCreated", 1,
		func(evt event.Event) (*event.Core, error) {
			applied = append(applied, 1)

			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 2, nil)
		},
	))

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, nil)

	_, err := registry.Upcast(evt)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}

	if len(applied) != 2 || applied[0] != 1 || applied[1] != 2 {
		t.Errorf("applied order = %v, want [1 2]", applied)
	}
}

func TestUpcasterRegistry_AlreadyCurrentVersion(t *testing.T) {
	t.Parallel()

	registry := event.NewUpcasterRegistry()

	var applied []int

	registry.Register(event.NewUpcaster("UserCreated", 1,
		func(evt event.Event) (*event.Core, error) {
			applied = append(applied, 1)

			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 2, nil)
		},
	))

	registry.Register(event.NewUpcaster("UserCreated", 2,
		func(evt event.Event) (*event.Core, error) {
			applied = append(applied, 2)

			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 3, nil)
		},
	))

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 3, nil)

	result, err := registry.Upcast(evt)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("upcasters applied = %v, want none (event already at version 3)", applied)
	}

	if result.Version() != 3 {
		t.Errorf("version = %d, want 3", result.Version())
	}
}

func TestUpcasterRegistry_PartialChain(t *testing.T) {
	t.Parallel()

	registry := event.NewUpcasterRegistry()

	var applied []int

	registry.Register(event.NewUpcaster("UserCreated", 1,
		func(evt event.Event) (*event.Core, error) {
			applied = append(applied, 1)

			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 2, nil)
		},
	))

	registry.Register(event.NewUpcaster("UserCreated", 2,
		func(evt event.Event) (*event.Core, error) {
			applied = append(applied, 2)

			return event.NewEvent("UserCreated", evt.AggregateID(), "User", 3, nil)
		},
	))

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 2, nil)

	result, err := registry.Upcast(evt)
	if err != nil {
		t.Fatalf("Upcast: %v", err)
	}

	if len(applied) != 1 || applied[0] != 2 {
		t.Errorf("applied = %v, want [2] (only v2→v3 upcaster)", applied)
	}

	if result.Version() != 3 {
		t.Errorf("version = %d, want 3", result.Version())
	}
}

func TestNewProjection_WithDecode(t *testing.T) {
	t.Parallel()

	codec := event.JSONCodec{}

	type userPayload struct {
		Name string `json:"name"`
	}

	var name string

	proj := event.NewProjection(
		"user-name",
		func(_ context.Context, evt event.Event) error {
			p, err := event.DecodePayload[userPayload](evt, codec)
			if err != nil {
				return err
			}

			name = p.Name

			return nil
		},
		[]event.Type{"UserCreated"},
	)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{"name":"Alice"}`))

	err := proj.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if name != "Alice" {
		t.Errorf("name = %q, want Alice", name)
	}
}
