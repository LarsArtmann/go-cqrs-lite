package adapters_test

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
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
		CatalogCore: command.MustNewCatalogCore(tp, id.NewAggregateID(), meta),
	}
}

func TestBuilder_AddCommand(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "Manages users")

	aggID := id.NewAggregateID()
	cmd := &testCreateUser{
		CatalogCore: command.MustNewCatalogCore("user.create", aggID, command.CatalogMeta{
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

	evtCore, err := cattest.NewCatalogCore(t,
		"order.created",
		event.CatalogMeta{
			Name:    "OrderCreated",
			Version: "1.0.0",
			Summary: "Order was created",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	type orderCreated struct {
		*event.CatalogCore

		OrderID string  `doc:"Unique order ID" json:"orderId"`
		Amount  float64 `doc:"Total amount"    json:"amount"`
	}

	evt := &orderCreated{CatalogCore: evtCore}
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
		CatalogCore: query.MustNewCatalogCore("user.get", query.CatalogMeta{
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

func TestBuilder_ExportD2(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "Manages orders")

	cmd := newTestCreateUser(
		"order.create",
		command.CatalogMeta{Name: "CreateOrder", Version: "1.0.0", Summary: "Create an order"},
	)
	builder.AddCommand("order-svc", cmd)

	result := builder.ExportD2("Test API", "1.0.0")
	if result == "" {
		t.Error("ExportD2 returned empty string")
	}
}

func TestBuilder_AddMessageToNewService(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")

	cmd := newTestCreateUser(
		"user.create",
		command.CatalogMeta{Name: "CreateUser", Version: "1.0.0", Summary: "Create a user"},
	)
	builder.AddCommand("auto-svc", cmd)

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Services", cat.Services, 1)

	svc := cat.Services[0]
	if svc.ID != "auto-svc" {
		t.Errorf("service ID = %q, want auto-svc", svc.ID)
	}

	cattest.AssertSliceLen(t, "svc.Commands", svc.Commands, 1)
}
