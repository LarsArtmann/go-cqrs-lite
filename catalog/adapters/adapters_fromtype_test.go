package adapters_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func TestBuilder_ExportAsyncAPI(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("E-Commerce", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "")

	cmd := newTestCreateUser(
		"order.create",
		command.CatalogMeta{Name: "CreateOrder", Version: "1.0.0", Summary: "Create an order"},
	)
	builder.AddCommand("order-svc", cmd)

	doc, err := builder.ExportAsyncAPI("E-Commerce API", "1.0.0",
		asyncapi.WithServer("production", "kafka:9092", "kafka"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if doc.AsyncAPI != "3.0.0" {
		t.Errorf("asyncapi = %q, want 3.0.0", doc.AsyncAPI)
	}

	if len(doc.Channels) == 0 {
		t.Error("expected at least one channel")
	}

	srv, ok := doc.Servers["production"]
	if !ok {
		t.Fatal("missing production server")
	}

	if srv.Host != "kafka:9092" {
		t.Errorf("server host = %q", srv.Host)
	}
}

func TestBuilder_MultipleMessages(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "Manages users")

	aggID := id.NewAggregateID()
	builder.AddCommand("user-svc", &testCreateUser{
		CatalogCore: command.MustNewCatalogCore("user.create", aggID, command.CatalogMeta{
			Name: "CreateUser", Version: "1.0.0", Summary: "Create user",
		}),
	})
	builder.AddCommand("user-svc", &testChangeEmail{
		CatalogCore: command.MustNewCatalogCore("user.change_email", aggID, command.CatalogMeta{
			Name: "ChangeEmail", Version: "1.0.0", Summary: "Change email",
		}),
	})

	cat := builder.Build()

	svc := cat.Services[0]
	if len(svc.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(svc.Commands))
	}
}

func TestBuilder_AddCommandFromType(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "")
	adapters.AddCommandFromType[testCreateUser](
		builder,
		"user-svc",
		"user.create",
		command.CatalogMeta{
			Name:    "CreateUser",
			Version: "1.0.0",
			Summary: "Creates a new user",
		},
	)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Commands", svc.Commands, 1)

	cmdMsg := svc.Commands[0]
	if cmdMsg.Name != "CreateUser" {
		t.Errorf("name = %q, want CreateUser", cmdMsg.Name)
	}

	if cmdMsg.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	cattest.AssertSchemaProperty(t, cmdMsg.Schema, "name")

	if _, ok := cmdMsg.Schema.Properties["CatalogCore"]; ok {
		t.Error("schema should NOT contain embedded CatalogCore")
	}
}

func TestBuilder_AddQueryFromType(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "")
	adapters.AddQueryFromType[testGetUser](
		builder,
		"user-svc",
		"user.get",
		query.CatalogMeta{
			Name:    "GetUser",
			Version: "1.0.0",
			Summary: "Gets a user",
		},
	)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Queries", svc.Queries, 1)

	if _, ok := svc.Queries[0].Schema.Properties["userId"]; !ok {
		t.Error("schema missing 'userId' property")
	}
}

func TestBuilder_FromCommandDispatcher(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	d.RegisterCatalogEntry("user.create", command.CatalogMeta{
		Name:    "CreateUser",
		Version: "1.0.0",
		Summary: "Creates a new user",
	})
	d.RegisterCatalogEntry("user.change_email", command.CatalogMeta{
		Name:    "ChangeEmail",
		Version: "1.0.0",
		Summary: "Changes user email",
	})

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "Manages users")
	adapters.FromCommandDispatcher(builder, "user-svc", d)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Commands", svc.Commands, 2)

	found := map[string]bool{}
	for _, cmd := range svc.Commands {
		found[cmd.ID] = true
		if cmd.Kind != catalog.CommandMessage {
			t.Errorf("kind = %v, want command", cmd.Kind)
		}

		if cmd.Direction != catalog.Receives {
			t.Errorf("direction = %v, want receives", cmd.Direction)
		}
	}

	if !found["user.create"] {
		t.Error("missing user.create command")
	}

	if !found["user.change_email"] {
		t.Error("missing user.change_email command")
	}
}

func TestBuilder_AddCommandWithSchema(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "")

	aggID := id.NewAggregateID()
	cmd := &testCreateUser{
		CatalogCore: command.MustNewCatalogCore("user.create", aggID, command.CatalogMeta{
			Name: "CreateUser", Version: "1.0.0", Summary: "Creates a user",
		}),
	}
	explicitSchema := &catalog.Schema{
		Type: "object",
		Properties: map[string]catalog.Property{
			"email": {Type: "string"},
		},
	}

	builder.AddCommandWithSchema("user-svc", cmd, explicitSchema)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Commands", svc.Commands, 1)

	cmdMsg := svc.Commands[0]
	if cmdMsg.Name != "CreateUser" {
		t.Errorf("name = %q, want CreateUser", cmdMsg.Name)
	}

	if cmdMsg.Schema != explicitSchema {
		t.Error("schema should be the explicitly provided schema")
	}

	if cmdMsg.Direction != catalog.Receives {
		t.Errorf("direction = %v, want receives", cmdMsg.Direction)
	}
}

func TestBuilder_AddEventWithDirection(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "")

	evtCore, err := cattest.NewCatalogCore(t,
		"order.shipped",
		event.CatalogMeta{
			Name:    "OrderShipped",
			Version: "1.0.0",
			Summary: "Order was shipped",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	type orderShipped struct {
		*event.CatalogCore

		TrackingNumber string `doc:"Tracking number" json:"trackingNumber"`
	}

	evt := &orderShipped{CatalogCore: evtCore}
	builder.AddEventWithDirection("order-svc", evt, catalog.Receives)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Events", svc.Events, 1)

	evtMsg := svc.Events[0]
	if evtMsg.Name != "OrderShipped" {
		t.Errorf("name = %q, want OrderShipped", evtMsg.Name)
	}

	if evtMsg.Direction != catalog.Receives {
		t.Errorf("direction = %v, want receives", evtMsg.Direction)
	}

	cattest.AssertSchemaProperty(t, evtMsg.Schema, "trackingNumber")
}
