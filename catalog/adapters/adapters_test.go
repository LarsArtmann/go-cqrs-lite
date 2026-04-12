package adapters_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/query"
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
	if len(cat.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cat.Services))
	}

	svc := cat.Services[0]
	if len(svc.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(svc.Commands))
	}

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

	if _, ok := cmdMsg.Schema.Properties["name"]; !ok {
		t.Error("schema missing 'name' property")
	}

	if _, ok := cmdMsg.Schema.Properties["email"]; !ok {
		t.Error("schema missing 'email' property")
	}
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
	if len(svc.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(svc.Events))
	}

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

	if _, ok := evtMsg.Schema.Properties["orderId"]; !ok {
		t.Error("schema missing 'orderId' property")
	}
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
	if len(svc.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(svc.Queries))
	}

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

	if _, ok := qryMsg.Schema.Properties["userId"]; !ok {
		t.Error("schema missing 'userId' property")
	}
}

func TestBuilder_AddDomain(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "Manages orders")
	builder.AddDomain("ordering", "Ordering", "Order management", []string{"order-svc"})

	cat := builder.Build()
	if len(cat.Domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(cat.Domains))
	}

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

	aggID := id.NewAggregateID()
	cmd := &testCreateUser{
		CatalogCore: command.NewCatalogCore("order.create", aggID, command.CatalogMeta{
			Name: "CreateOrder", Version: "1.0.0", Summary: "Create an order",
		}),
	}
	builder.AddCommand("order-svc", cmd)

	err := builder.ExportEventCatalog(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	svcPath := filepath.Join(tmpDir, "services", "order-svc", "index.mdx")
	if _, err := os.Stat(svcPath); os.IsNotExist(err) {
		t.Error("service index.mdx not created")
	}

	cmdPath := filepath.Join(
		tmpDir,
		"services",
		"order-svc",
		"commands",
		"order.create",
		"index.mdx",
	)
	if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
		t.Errorf("command index.mdx not created at %s", cmdPath)
	}

	cfgPath := filepath.Join(tmpDir, "eventcatalog.config.js")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("eventcatalog.config.js not created")
	}
}

func TestBuilder_ExportAsyncAPI(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("E-Commerce", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "")

	aggID := id.NewAggregateID()
	cmd := &testCreateUser{
		CatalogCore: command.NewCatalogCore("order.create", aggID, command.CatalogMeta{
			Name: "CreateOrder", Version: "1.0.0", Summary: "Create an order",
		}),
	}
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
	if len(svc.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(svc.Commands))
	}

	cmdMsg := svc.Commands[0]
	if cmdMsg.Name != "CreateUser" {
		t.Errorf("name = %q, want CreateUser", cmdMsg.Name)
	}

	if cmdMsg.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	if _, ok := cmdMsg.Schema.Properties["name"]; !ok {
		t.Error("schema missing 'name' property")
	}

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
	if len(svc.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(svc.Queries))
	}

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
	if len(svc.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(svc.Commands))
	}

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
	if len(svc.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(svc.Queries))
	}

	qry := svc.Queries[0]
	if qry.ID != "user.get" {
		t.Errorf("ID = %q, want user.get", qry.ID)
	}

	if qry.Name != "GetUser" {
		t.Errorf("name = %q, want GetUser", qry.Name)
	}
}
