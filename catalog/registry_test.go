package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func TestRegistry_AddService(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	cattest.AddServiceWithSummary(t, reg, "user-svc", "User Service", "1.0.0", "Manages users")

	cat := reg.Build()
	if len(cat.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cat.Services))
	}

	svc := cat.Services[0]
	if svc.ID != "user-svc" {
		t.Errorf("expected user-svc, got %s", svc.ID)
	}

	if svc.Name != "User Service" {
		t.Errorf("expected User Service, got %s", svc.Name)
	}
}

func TestRegistry_AddCommand(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddCommand("user-svc", catalog.Message{
		ID:        "CreateUser",
		Name:      "Create User",
		Version:   "1.0.0",
		Summary:   "Creates a new user",
		Direction: catalog.Receives,
		Schema: catalog.SchemaFromType[struct {
			Email string `json:"email"`
		}](),
	})

	cat := reg.Build()
	cattest.AssertSliceLen(t, "cat.Services", cat.Services, 1)

	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Commands", svc.Commands, 1)

	cmd := svc.Commands[0]
	if cmd.Kind != catalog.CommandMessage {
		t.Errorf("expected command kind, got %s", cmd.Kind)
	}

	if cmd.ID != "CreateUser" {
		t.Errorf("expected CreateUser, got %s", cmd.ID)
	}

	if cmd.Schema == nil {
		t.Error("expected schema to be set")
	}
}

func TestRegistry_AddEvent(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
	reg.AddEvent("order-svc", catalog.Message{
		ID:        "OrderCreated",
		Name:      "Order Created",
		Version:   "1.0.0",
		Summary:   "Emitted when an order is created",
		Direction: catalog.Sends,
	})

	cat := reg.Build()

	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Events", svc.Events, 1)

	evt := svc.Events[0]
	if evt.Kind != catalog.EventMessage {
		t.Errorf("expected event kind, got %s", evt.Kind)
	}
}

func TestRegistry_AddQuery(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddQuery("user-svc", catalog.Message{
		ID:      "GetUser",
		Name:    "Get User",
		Version: "1.0.0",
		Summary: "Retrieves user details",
	})

	cat := reg.Build()

	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Queries", svc.Queries, 1)

	q := svc.Queries[0]
	if q.Kind != catalog.QueryMessage {
		t.Errorf("expected query kind, got %s", q.Kind)
	}
}

func TestRegistry_AddDomain(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddDomain(catalog.Domain{
		ID:      "orders",
		Name:    "Orders Domain",
		Version: "1.0.0",
		Summary: "Order management",
	})

	cat := reg.Build()
	cattest.AssertSliceLen(t, "cat.Domains", cat.Domains, 1)

	if cat.Domains[0].ID != "orders" {
		t.Errorf("expected orders, got %s", cat.Domains[0].ID)
	}
}

func TestRegistry_AddServiceToDomain(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddDomain(catalog.Domain{ID: "orders", Name: "Orders", Version: "1.0.0"})
	reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})

	err := reg.AddServiceToDomain("order-svc", "orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cat := reg.Build()
	cattest.AssertSliceLen(t, "cat.Domains[0].Services", cat.Domains[0].Services, 1)

	if cat.Domains[0].Services[0] != "order-svc" {
		t.Errorf("expected order-svc, got %s", cat.Domains[0].Services[0])
	}
}

func TestRegistry_AddServiceToDomain_NotFound(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()

	err := reg.AddServiceToDomain("svc", "nonexistent")
	if err == nil {
		t.Error("expected error for missing domain")
	}
}

func TestRegistry_AddChannel(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddChannel(catalog.Channel{
		ID:      "orders.events",
		Name:    "Orders Event Channel",
		Version: "1.0.0",
		Address: "orders.events",
	})

	cat := reg.Build()
	cattest.AssertSliceLen(t, "cat.Channels", cat.Channels, 1)

	if cat.Channels[0].Address != "orders.events" {
		t.Errorf("expected orders.events, got %s", cat.Channels[0].Address)
	}
}

func TestRegistry_Build_MultipleServices(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("E-Commerce", "2.0.0")
	reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
	reg.AddService(catalog.Service{ID: "user-svc", Name: "User Service", Version: "1.0.0"})
	reg.AddCommand(
		"order-svc",
		catalog.Message{ID: "CreateOrder", Name: "Create Order", Version: "1.0.0"},
	)
	reg.AddEvent(
		"order-svc",
		catalog.Message{
			ID:        "OrderCreated",
			Name:      "Order Created",
			Version:   "1.0.0",
			Direction: catalog.Sends,
		},
	)
	reg.AddQuery("user-svc", catalog.Message{ID: "GetUser", Name: "Get User", Version: "1.0.0"})

	cat := reg.Build()
	if cat.Title != "E-Commerce" {
		t.Errorf("expected E-Commerce, got %s", cat.Title)
	}

	if cat.Version != "2.0.0" {
		t.Errorf("expected 2.0.0, got %s", cat.Version)
	}

	if len(cat.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(cat.Services))
	}
}

func TestRegistry_ServiceMerge(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry(catalog.Service{
		ID: "svc", Name: "Service", Version: "1.0.0",
		Commands: []catalog.Message{{ID: "Cmd1", Name: "Cmd1", Version: "1.0.0"}},
	})
	reg.AddService(catalog.Service{
		ID: "svc", Name: "Service", Version: "1.0.0",
		Events: []catalog.Message{{ID: "Evt1", Name: "Evt1", Version: "1.0.0"}},
	})

	cat := reg.Build()
	cattest.AssertSliceLen(t, "cat.Services", cat.Services, 1)

	if len(cat.Services[0].Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(cat.Services[0].Commands))
	}

	if len(cat.Services[0].Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(cat.Services[0].Events))
	}
}

func TestRegistry_ServiceMergeWithQueries(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry(catalog.Service{
		ID:      "svc",
		Name:    "Service",
		Version: "1.0.0",
		Queries: []catalog.Message{
			{ID: "Q1", Name: "Q1", Version: "1.0.0"},
			{ID: "Q2", Name: "Q2", Version: "1.0.0"},
		},
	})

	cat := reg.Build()
	if len(cat.Services[0].Queries) != 2 {
		t.Errorf("expected 2 queries after merge, got %d", len(cat.Services[0].Queries))
	}
}

func TestGetID_FallbackToName(t *testing.T) {
	t.Parallel()

	msg := catalog.Message{Name: "CreateUser"}
	if catalog.Key(msg) != "CreateUser" {
		t.Errorf("expected CreateUser, got %s", catalog.Key(msg))
	}
}

func TestGetID_UsesID(t *testing.T) {
	t.Parallel()

	msg := catalog.Message{ID: "cmd-123", Name: "CreateUser"}
	if catalog.Key(msg) != "cmd-123" {
		t.Errorf("expected cmd-123, got %s", catalog.Key(msg))
	}
}

func TestRegistry_AddServiceMergeNoCommands(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	reg.AddService(catalog.Service{
		ID:       "svc",
		Name:     "Service",
		Version:  "1.0.0",
		Commands: []catalog.Message{{ID: "Cmd1", Name: "Cmd1", Version: "1.0.0"}},
	})

	cat := reg.Build()
	if len(cat.Services[0].Commands) != 1 {
		t.Errorf("expected 1 command after merge, got %d", len(cat.Services[0].Commands))
	}
}

func TestRegistry_BuildWithChannels(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddChannel(catalog.Channel{
		ID:        "ch1",
		Name:      "Channel 1",
		Version:   "1.0.0",
		Address:   "topic1",
		Protocols: []catalog.Protocol{"kafka"},
		Messages:  []catalog.MessageID{"msg1"},
		Summary:   "A test channel",
	})

	cat := reg.Build()
	if len(cat.Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(cat.Channels))
	}

	if cat.Channels[0].Summary != "A test channel" {
		t.Errorf("expected summary, got %s", cat.Channels[0].Summary)
	}
}
