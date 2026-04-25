package adapters_test

import (
	"path/filepath"
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

type testCreateUser struct {
	*command.CatalogCore

	Name  string `doc:"Full name of the user" json:"name"`
	Email string `doc:"Email address"         json:"email"`
}

type testChangeEmail struct {
	*command.CatalogCore

	NewEmail string `doc:"New email address" json:"newEmail"`
}

type testGetUser struct {
	*query.CatalogCore

	UserID string `doc:"ID of the user to retrieve" json:"userId"`
}

func newTestCreateUser(tp command.Type, meta command.CatalogMeta) *testCreateUser {
	return &testCreateUser{
		CatalogCore: command.NewCatalogCore(tp, id.NewAggregateID(), meta),
	}
}

func TestBuilder_AddCommand(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "Manages users")

	aggID := id.NewAggregateID()
	cmd := &testCreateUser{
		CatalogCore: command.NewCatalogCore("user.create", aggID, command.CatalogMeta{
			Name:    "CreateUser",
			Version: "1.0.0",
			Summary: "Creates a new user",
		}),
		Name:  "Alice",
		Email: "alice@example.com",
	}
	builder.AddCommand("user-svc", cmd)

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Services", cat.Services, 1)

	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Commands", svc.Commands, 1)

	cmdMsg := svc.Commands[0]
	if cmdMsg.Kind != catalog.CommandMessage {
		t.Errorf("kind = %v, want command", cmdMsg.Kind)
	}

	if cmdMsg.Name != "CreateUser" {
		t.Errorf("name = %q, want CreateUser", cmdMsg.Name)
	}

	if cmdMsg.Summary != "Creates a new user" {
		t.Errorf("summary = %q", cmdMsg.Summary)
	}

	if cmdMsg.Direction != catalog.Receives {
		t.Errorf("direction = %v, want receives", cmdMsg.Direction)
	}

	if cmdMsg.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	if cmdMsg.Schema.Type != "object" {
		t.Errorf("schema type = %q, want object", cmdMsg.Schema.Type)
	}

	cattest.AssertSchemaProperty(t, cmdMsg.Schema, "name")
	cattest.AssertSchemaProperty(t, cmdMsg.Schema, "email")
}

func TestBuilder_AddEvent(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "")

	evtCore, err := event.NewEventCatalogCore(
		"order.created",
		id.NewAggregateID(),
		"Order",
		1,
		nil,
		event.EventCatalogMeta{
			Name:    "OrderCreated",
			Version: "1.0.0",
			Summary: "Order was created",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	type orderCreated struct {
		*event.EventCatalogCore

		OrderID string  `doc:"Unique order ID" json:"orderId"`
		Amount  float64 `doc:"Total amount"    json:"amount"`
	}

	evt := &orderCreated{EventCatalogCore: evtCore}
	builder.AddEvent("order-svc", evt)

	cat := builder.Build()

	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Events", svc.Events, 1)

	evtMsg := svc.Events[0]
	if evtMsg.Kind != catalog.EventMessage {
		t.Errorf("kind = %v, want event", evtMsg.Kind)
	}

	if evtMsg.Name != "OrderCreated" {
		t.Errorf("name = %q, want OrderCreated", evtMsg.Name)
	}

	if evtMsg.Direction != catalog.Sends {
		t.Errorf("direction = %v, want sends", evtMsg.Direction)
	}

	if evtMsg.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	cattest.AssertSchemaProperty(t, evtMsg.Schema, "orderId")
}

func TestBuilder_AddQuery(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "")

	qry := &testGetUser{
		CatalogCore: query.NewCatalogCore("user.get", query.CatalogMeta{
			Name:    "GetUser",
			Version: "1.0.0",
			Summary: "Retrieves a user by ID",
		}),
		UserID: "abc-123",
	}
	builder.AddQuery("user-svc", qry)

	cat := builder.Build()

	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Queries", svc.Queries, 1)

	qryMsg := svc.Queries[0]
	if qryMsg.Kind != catalog.QueryMessage {
		t.Errorf("kind = %v, want query", qryMsg.Kind)
	}

	if qryMsg.Name != "GetUser" {
		t.Errorf("name = %q, want GetUser", qryMsg.Name)
	}

	if qryMsg.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	cattest.AssertSchemaProperty(t, qryMsg.Schema, "userId")
}

func TestBuilder_AddDomain(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "Manages orders")
	builder.AddDomain("ordering", "Ordering", "Order management", []string{"order-svc"})

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Domains", cat.Domains, 1)

	d := cat.Domains[0]
	if d.ID != "ordering" {
		t.Errorf("domain ID = %q, want ordering", d.ID)
	}

	if len(d.Services) != 1 || d.Services[0] != "order-svc" {
		t.Errorf("domain services = %v", d.Services)
	}
}

func TestBuilder_ExportEventCatalog(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	builder := adapters.NewBuilder("E-Commerce", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "Manages orders")

	cmd := newTestCreateUser(
		"order.create",
		command.CatalogMeta{Name: "CreateOrder", Version: "1.0.0", Summary: "Create an order"},
	)
	builder.AddCommand("order-svc", cmd)

	err := builder.ExportEventCatalog(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	svcPath := filepath.Join(tmpDir, "services", "order-svc", "index.mdx")
	cattest.AssertFileExists(t, svcPath)

	cmdPath := filepath.Join(
		tmpDir,
		"services",
		"order-svc",
		"commands",
		"order.create",
		"index.mdx",
	)
	cattest.AssertFileExists(t, cmdPath)

	cfgPath := filepath.Join(tmpDir, "eventcatalog.config.js")
	cattest.AssertFileExists(t, cfgPath)
}

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
		CatalogCore: command.NewCatalogCore("user.create", aggID, command.CatalogMeta{
			Name: "CreateUser", Version: "1.0.0", Summary: "Create user",
		}),
	})
	builder.AddCommand("user-svc", &testChangeEmail{
		CatalogCore: command.NewCatalogCore("user.change_email", aggID, command.CatalogMeta{
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
		CatalogCore: command.NewCatalogCore("user.create", aggID, command.CatalogMeta{
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

	evtCore, err := event.NewEventCatalogCore(
		"order.shipped",
		id.NewAggregateID(),
		"Order",
		1,
		nil,
		event.EventCatalogMeta{
			Name:    "OrderShipped",
			Version: "1.0.0",
			Summary: "Order was shipped",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	type orderShipped struct {
		*event.EventCatalogCore

		TrackingNumber string `doc:"Tracking number" json:"trackingNumber"`
	}

	evt := &orderShipped{EventCatalogCore: evtCore}
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

func TestBuilder_AddEventFromType(t *testing.T) {
	t.Parallel()

	type userDeleted struct {
		*event.EventCatalogCore

		Reason string `doc:"Deletion reason" json:"reason"`
	}

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "")

	adapters.AddEventFromType[userDeleted](
		builder,
		"user-svc",
		"user.deleted",
		event.EventCatalogMeta{
			Name:    "UserDeleted",
			Version: "1.0.0",
			Summary: "User was deleted",
		},
		catalog.Sends,
	)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Events", svc.Events, 1)

	evtMsg := svc.Events[0]
	if evtMsg.Name != "UserDeleted" {
		t.Errorf("name = %q, want UserDeleted", evtMsg.Name)
	}

	if evtMsg.Direction != catalog.Sends {
		t.Errorf("direction = %v, want sends", evtMsg.Direction)
	}

	if evtMsg.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	cattest.AssertSchemaProperty(t, evtMsg.Schema, "reason")

	if _, ok := evtMsg.Schema.Properties["EventCatalogCore"]; ok {
		t.Error("schema should NOT contain embedded EventCatalogCore")
	}
}

func TestBuilder_AddServiceToDomain(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "")
	builder.AddService("payment-svc", "Payment Service", "1.0.0", "")
	builder.AddDomain("ecommerce", "E-Commerce", "Online store", []string{"order-svc"})

	builder.AddServiceToDomain("payment-svc", "ecommerce")

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Domains", cat.Domains, 1)

	d := cat.Domains[0]
	if len(d.Services) != 2 {
		t.Errorf("expected 2 services in domain, got %d", len(d.Services))
	}

	found := map[string]bool{}
	for _, sid := range d.Services {
		found[sid] = true
	}

	if !found["order-svc"] || !found["payment-svc"] {
		t.Errorf("domain services = %v, want both order-svc and payment-svc", d.Services)
	}
}

func TestBuilder_AddServiceToDomain_NonexistentDomain(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "")

	builder.AddServiceToDomain("svc", "nonexistent")

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Domains", cat.Domains, 0)
}

func TestBuilder_AddChannel(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")

	ch := catalog.Channel{
		ID:      "order-events",
		Name:    "Order Events Channel",
		Version: "1.0.0",
		Address: "orders.events",
	}
	builder.AddChannel(ch)

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Channels", cat.Channels, 1)

	got := cat.Channels[0]
	if got.ID != "order-events" {
		t.Errorf("channel ID = %q, want order-events", got.ID)
	}

	if got.Address != "orders.events" {
		t.Errorf("channel address = %q, want orders.events", got.Address)
	}
}

func TestBuilder_FromQueryDispatcher(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	d.RegisterCatalogEntry("user.get", query.CatalogMeta{
		Name:    "GetUser",
		Version: "1.0.0",
		Summary: "Gets a user by ID",
	})

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "")
	adapters.FromQueryDispatcher(builder, "user-svc", d)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Queries", svc.Queries, 1)

	qry := svc.Queries[0]
	if qry.ID != "user.get" {
		t.Errorf("ID = %q, want user.get", qry.ID)
	}

	if qry.Name != "GetUser" {
		t.Errorf("name = %q, want GetUser", qry.Name)
	}
}
