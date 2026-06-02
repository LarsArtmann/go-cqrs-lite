package schema

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func newTestUpcaster(typ event.Type, version event.SchemaVersion, payload string) Upcaster {
	return NewUpcaster(typ, version, func(evt event.Event) (*event.ImmutableEvent, error) {
		return event.NewEvent(typ, evt.AggregateID(), "User", evt.Version(), []byte(payload))
	})
}

func newTestEvent(version int, payload string) (*upcasterRegistry, event.Event) {
	registry := newUpcasterRegistry()
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("UserCreated", aggID, "User", event.Version(version), []byte(payload))

	return registry, evt
}

func TestUpcasterFunc(t *testing.T) {
	t.Parallel()

	uc := NewUpcaster(
		"UserCreated",
		1,
		func(_ event.Event) (*event.ImmutableEvent, error) {
			aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

			return event.NewEvent("UserCreated", aggID, "User", 2, nil)
		},
	)

	if uc.SourceType() != "UserCreated" {
		t.Errorf("SourceType = %q, want UserCreated", uc.SourceType())
	}

	if uc.SourceVersion() != 1 {
		t.Errorf("SourceVersion = %d, want 1", uc.SourceVersion())
	}
}

func TestUpcasterRegistry_NoUpcasters(t *testing.T) {
	t.Parallel()

	registry := newUpcasterRegistry()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{"name":"Alice"}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	result, err := registry.upcast(evt)
	if err != nil {
		t.Fatalf("upcast: %v", err)
	}

	if result.ID() != evt.ID() {
		t.Error("expected same event when no upcasters registered")
	}
}

func TestUpcasterRegistry_SingleUpcaster(t *testing.T) {
	t.Parallel()

	registry, evt := newTestEvent(5, `{"name":"Alice"}`)
	registry.register(newTestUpcaster("UserCreated", 1, `{"name":"Alice","email":""}`))

	result, err := registry.upcast(evt)
	if err != nil {
		t.Fatalf("upcast: %v", err)
	}

	if string(result.Payload()) != `{"name":"Alice","email":""}` {
		t.Errorf("payload = %q, want enriched payload", string(result.Payload()))
	}

	if result.SchemaVersion() != 2 {
		t.Errorf("SchemaVersion = %d, want 2", result.SchemaVersion())
	}
}

func TestUpcasterRegistry_ChainedUpcasters(t *testing.T) {
	t.Parallel()

	registry, evt := newTestEvent(7, `{"name":"Alice"}`)
	registry.register(newTestUpcaster("UserCreated", 1, `{"name":"Alice","email":""}`))
	registry.register(
		newTestUpcaster("UserCreated", 2, `{"name":"Alice","email":"","active":true}`),
	)

	result, err := registry.upcast(evt)
	if err != nil {
		t.Fatalf("upcast: %v", err)
	}

	want := `{"name":"Alice","email":"","active":true}`
	if string(result.Payload()) != want {
		t.Errorf("payload = %q, want %q", string(result.Payload()), want)
	}

	if result.SchemaVersion() != 3 {
		t.Errorf("SchemaVersion = %d, want 3", result.SchemaVersion())
	}
}

func registerUserCreatedUpcasterV1(registry *upcasterRegistry) {
	registry.register(NewUpcaster(
		"UserCreated", 1,
		func(evt event.Event) (*event.ImmutableEvent, error) {
			return event.NewEvent(
				"UserCreated",
				evt.AggregateID(),
				"User",
				evt.Version(),
				nil,
			)
		},
	))
}

func TestUpcasterRegistry_DifferentEventTypes(t *testing.T) {
	t.Parallel()

	registry := newUpcasterRegistry()
	registerUserCreatedUpcasterV1(registry)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("OrderPlaced", aggID, "Order", 5, []byte(`{}`))

	result, err := registry.upcast(evt)
	if err != nil {
		t.Fatalf("upcast: %v", err)
	}

	if result.ID() != evt.ID() {
		t.Error("expected unchanged event for different type")
	}
}

func registerTrackingUpcaster(
	registry *upcasterRegistry,
	version event.SchemaVersion,
	applied *[]int,
) {
	registry.register(NewUpcaster(
		"UserCreated", version,
		func(evt event.Event) (*event.ImmutableEvent, error) {
			*applied = append(*applied, int(version))

			return event.NewEvent(
				"UserCreated",
				evt.AggregateID(),
				"User",
				evt.Version(),
				nil,
			)
		},
	))
}

func TestUpcasterRegistry_VersionSorting(t *testing.T) {
	t.Parallel()

	registry := newUpcasterRegistry()

	var applied []int

	registerTrackingUpcaster(registry, 2, &applied)
	registerTrackingUpcaster(registry, 1, &applied)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 3, nil)

	_, err := registry.upcast(evt)
	if err != nil {
		t.Fatalf("upcast: %v", err)
	}

	if len(applied) != 2 || applied[0] != 1 || applied[1] != 2 {
		t.Errorf("applied order = %v, want [1 2]", applied)
	}
}

func TestUpcasterRegistry_AlreadyCurrentVersion(t *testing.T) {
	t.Parallel()

	registry := newUpcasterRegistry()

	var applied []int

	registerTrackingUpcaster(registry, 1, &applied)
	registerTrackingUpcaster(registry, 2, &applied)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 10, nil, event.WithSchemaVersion(3))

	result, err := registry.upcast(evt)
	if err != nil {
		t.Fatalf("upcast: %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("upcasters applied = %v, want none (event already at schema version 3)", applied)
	}

	if result.Version() != 10 {
		t.Errorf("stream version = %d, want 10", result.Version())
	}
}

func TestUpcasterRegistry_PartialChain(t *testing.T) {
	t.Parallel()

	registry := newUpcasterRegistry()

	var applied []int

	registerTrackingUpcaster(registry, 1, &applied)
	registerTrackingUpcaster(registry, 2, &applied)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 8, nil, event.WithSchemaVersion(2))

	result, err := registry.upcast(evt)
	if err != nil {
		t.Fatalf("upcast: %v", err)
	}

	if len(applied) != 1 || applied[0] != 2 {
		t.Errorf("applied = %v, want [2] (only v2→v3 upcaster)", applied)
	}

	if result.Version() != 8 {
		t.Errorf("stream version = %d, want 8", result.Version())
	}

	if result.SchemaVersion() != 3 {
		t.Errorf("SchemaVersion = %d, want 3", result.SchemaVersion())
	}
}

func TestUpcasterRegistry_AutoIncrementsSchemaVersion(t *testing.T) {
	t.Parallel()

	registry := newUpcasterRegistry()
	registerUserCreatedUpcasterV1(registry)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 5, nil)

	result, err := registry.upcast(evt)
	if err != nil {
		t.Fatalf("upcast: %v", err)
	}

	if result.SchemaVersion() != 2 {
		t.Errorf("SchemaVersion = %d, want 2 (auto-incremented from 1)", result.SchemaVersion())
	}
}
