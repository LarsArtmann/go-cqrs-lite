package adapters_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
)

type createUserCmd struct {
	Name  string `json:"name"  doc:"Full name of the user"`
	Email string `json:"email" doc:"Email address"`
}

type changeEmailCmd struct {
	NewEmail string `json:"newEmail" doc:"New email address"`
}

type userCreatedEvt struct {
	UserID string `json:"userId" doc:"User ID"`
	Email  string `json:"email"  doc:"Email address"`
}

type getUserQry struct {
	UserID string `json:"userId" doc:"ID of the user to retrieve"`
}

func TestBuilder_AddService_WithCommand(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService(
		"user-svc", "User Service", "1.0.0", "Manages users",
		catalog.Command[createUserCmd]("user.create"),
	)

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Services", cat.Services, 1)

	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Commands", svc.Commands, 1)

	cmdMsg := svc.Commands[0]
	if cmdMsg.Kind != catalog.CommandMessage {
		t.Errorf("kind = %v, want command", cmdMsg.Kind)
	}

	if cmdMsg.Name != "Create User" {
		t.Errorf("name = %q, want Create User", cmdMsg.Name)
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

func TestBuilder_AddService_WithEvent(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService(
		"order-svc", "Order Service", "1.0.0", "",
		catalog.Event[userCreatedEvt]("user.created", catalog.Sends),
	)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Events", svc.Events, 1)

	evtMsg := svc.Events[0]
	if evtMsg.Kind != catalog.EventMessage {
		t.Errorf("kind = %v, want event", evtMsg.Kind)
	}

	if evtMsg.Name != "User Created" {
		t.Errorf("name = %q, want User Created", evtMsg.Name)
	}

	if evtMsg.Direction != catalog.Sends {
		t.Errorf("direction = %v, want sends", evtMsg.Direction)
	}

	if evtMsg.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	cattest.AssertSchemaProperty(t, evtMsg.Schema, "userId")
}

func TestBuilder_AddService_WithQuery(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService(
		"user-svc", "User Service", "1.0.0", "",
		catalog.Query[getUserQry]("user.get"),
	)

	cat := builder.Build()
	svc := cat.Services[0]
	cattest.AssertSliceLen(t, "svc.Queries", svc.Queries, 1)

	qryMsg := svc.Queries[0]
	if qryMsg.Kind != catalog.QueryMessage {
		t.Errorf("kind = %v, want query", qryMsg.Kind)
	}

	if qryMsg.Name != "Get User" {
		t.Errorf("name = %q, want Get User", qryMsg.Name)
	}

	if qryMsg.Schema == nil {
		t.Fatal("schema should not be nil")
	}

	cattest.AssertSchemaProperty(t, qryMsg.Schema, "userId")
}

func TestBuilder_AddService_WithOptions(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService(
		"user-svc", "User Service", "1.0.0", "",
		catalog.Command[createUserCmd](
			"user.create",
			catalog.Name("Create User Account"),
			catalog.Summary("Creates a new user with email verification"),
			catalog.Version("2.0.0"),
		),
	)

	cat := builder.Build()
	cmdMsg := cat.Services[0].Commands[0]

	if cmdMsg.Name != "Create User Account" {
		t.Errorf("name = %q, want Create User Account", cmdMsg.Name)
	}

	if cmdMsg.Summary != "Creates a new user with email verification" {
		t.Errorf("summary = %q", cmdMsg.Summary)
	}

	if cmdMsg.Version != "2.0.0" {
		t.Errorf("version = %q, want 2.0.0", cmdMsg.Version)
	}
}

func TestBuilder_MultipleMessages(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService(
		"user-svc", "User Service", "1.0.0", "Manages users",
		catalog.Command[createUserCmd]("user.create"),
		catalog.Command[changeEmailCmd]("user.change_email"),
		catalog.Event[userCreatedEvt]("user.created", catalog.Sends),
		catalog.Query[getUserQry]("user.get"),
	)

	cat := builder.Build()
	svc := cat.Services[0]

	if len(svc.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(svc.Commands))
	}
	if len(svc.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(svc.Events))
	}
	if len(svc.Queries) != 1 {
		t.Errorf("expected 1 query, got %d", len(svc.Queries))
	}
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

func TestBuilder_AddServiceToDomain(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("order-svc", "Order Service", "1.0.0", "")
	builder.AddService("payment-svc", "Payment Service", "1.0.0", "")
	builder.AddDomain("ecommerce", "E-Commerce", "Online store", []string{"order-svc"})

	err := builder.AddServiceToDomain("payment-svc", "ecommerce")
	if err != nil {
		t.Fatalf("add service to domain: %v", err)
	}

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Domains", cat.Domains, 1)

	d := cat.Domains[0]
	if len(d.Services) != 2 {
		t.Errorf("expected 2 services in domain, got %d", len(d.Services))
	}

	found := map[string]bool{}
	for _, sid := range d.Services {
		found[string(sid)] = true
	}

	if !found["order-svc"] || !found["payment-svc"] {
		t.Errorf("domain services = %v, want both order-svc and payment-svc", d.Services)
	}
}

func TestBuilder_AddServiceToDomain_NonexistentDomain(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "")

	err := builder.AddServiceToDomain("svc", "nonexistent")
	if err == nil {
		t.Fatal("expected error when adding service to nonexistent domain")
	}
}

func TestBuilder_AddChannel(t *testing.T) {
	t.Parallel()

	builder := adapters.NewBuilder("Test API", "1.0.0")
	builder.AddChannel(catalog.Channel{
		ID:      "user.commands",
		Name:    "User Commands",
		Version: "1.0.0",
		Address: "user.commands",
	})

	cat := builder.Build()
	cattest.AssertSliceLen(t, "cat.Channels", cat.Channels, 1)

	ch := cat.Channels[0]
	if ch.ID != "user.commands" {
		t.Errorf("channel ID = %q, want user.commands", ch.ID)
	}

	if ch.Name != "User Commands" {
		t.Errorf("channel Name = %q, want User Commands", ch.Name)
	}
}
