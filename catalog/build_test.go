package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

type TestCreateUser struct {
	Email string `json:"email" doc:"User email address"`
	Name  string `json:"name"  doc:"Display name"`
}

type TestUserCreated struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
}

type TestGetUser struct {
	UserID string `json:"userId"`
}

func TestBuilder_AddService_WithCommand(t *testing.T) {
	t.Parallel()

	builder := catalog.NewBuilder("Test Service", "1.0.0")
	builder.AddService("test-svc", "Test Service", "1.0.0", "A test service",
		catalog.Command[TestCreateUser]("user.create"),
	)

	cat := builder.Build()
	if len(cat.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cat.Services))
	}

	svc := cat.Services[0]
	if len(svc.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(svc.Commands))
	}

	cmd := svc.Commands[0]
	if cmd.ID != "user.create" {
		t.Errorf("command ID = %q, want %q", cmd.ID, "user.create")
	}
	if cmd.Name != "Test Create User" {
		t.Errorf("command name = %q, want %q", cmd.Name, "Test Create User")
	}
	if cmd.Direction != catalog.Receives {
		t.Errorf("command direction = %q, want %q", cmd.Direction, catalog.Receives)
	}
	if cmd.Kind != catalog.CommandMessage {
		t.Errorf("command kind = %q, want %q", cmd.Kind, catalog.CommandMessage)
	}
	if cmd.Schema == nil {
		t.Fatal("expected schema, got nil")
	}
	if cmd.Schema.Type != "object" {
		t.Errorf("schema type = %q, want %q", cmd.Schema.Type, "object")
	}
	if _, ok := cmd.Schema.Properties["email"]; !ok {
		t.Error("expected schema to have 'email' property")
	}
}

func TestBuilder_AddService_WithEvent(t *testing.T) {
	t.Parallel()

	builder := catalog.NewBuilder("Test Service", "1.0.0")
	builder.AddService("test-svc", "Test Service", "1.0.0", "A test service",
		catalog.Event[TestUserCreated]("user.created", catalog.Sends),
	)

	cat := builder.Build()
	svc := cat.Services[0]
	if len(svc.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(svc.Events))
	}

	evt := svc.Events[0]
	if evt.ID != "user.created" {
		t.Errorf("event ID = %q, want %q", evt.ID, "user.created")
	}
	if evt.Name != "Test User Created" {
		t.Errorf("event name = %q, want %q", evt.Name, "Test User Created")
	}
	if evt.Direction != catalog.Sends {
		t.Errorf("event direction = %q, want %q", evt.Direction, catalog.Sends)
	}
	if evt.Kind != catalog.EventMessage {
		t.Errorf("event kind = %q, want %q", evt.Kind, catalog.EventMessage)
	}
}

func TestBuilder_AddService_WithQuery(t *testing.T) {
	t.Parallel()

	builder := catalog.NewBuilder("Test Service", "1.0.0")
	builder.AddService("test-svc", "Test Service", "1.0.0", "A test service",
		catalog.Query[TestGetUser]("user.get"),
	)

	cat := builder.Build()
	svc := cat.Services[0]
	if len(svc.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(svc.Queries))
	}

	qry := svc.Queries[0]
	if qry.ID != "user.get" {
		t.Errorf("query ID = %q, want %q", qry.ID, "user.get")
	}
	if qry.Name != "Test Get User" {
		t.Errorf("query name = %q, want %q", qry.Name, "Test Get User")
	}
	if qry.Direction != catalog.Receives {
		t.Errorf("query direction = %q, want %q", qry.Direction, catalog.Receives)
	}
}

func TestBuilder_AddService_WithOptions(t *testing.T) {
	t.Parallel()

	builder := catalog.NewBuilder("Test Service", "1.0.0")
	builder.AddService("test-svc", "Test Service", "1.0.0", "A test service",
		catalog.Command[TestCreateUser]("user.create",
			catalog.Name("Create User Account"),
			catalog.Summary("Creates a new user with email verification"),
			catalog.Version("2.0.0"),
		),
	)

	cat := builder.Build()
	cmd := cat.Services[0].Commands[0]

	if cmd.Name != "Create User Account" {
		t.Errorf("name = %q, want %q", cmd.Name, "Create User Account")
	}
	if cmd.Summary != "Creates a new user with email verification" {
		t.Errorf("summary = %q, want %q", cmd.Summary, "Creates a new user with email verification")
	}
	if cmd.Version != "2.0.0" {
		t.Errorf("version = %q, want %q", cmd.Version, "2.0.0")
	}
}

func TestBuilder_AddDomain(t *testing.T) {
	t.Parallel()

	builder := catalog.NewBuilder("Test Service", "1.0.0")
	builder.AddService("test-svc", "Test Service", "1.0.0", "A test service")
	builder.AddDomain("identity", "Identity", "1.0.0", "User identity", "test-svc")

	cat := builder.Build()
	if len(cat.Domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(cat.Domains))
	}

	domain := cat.Domains[0]
	if domain.ID != "identity" {
		t.Errorf("domain ID = %q, want %q", domain.ID, "identity")
	}
	if domain.Name != "Identity" {
		t.Errorf("domain name = %q, want %q", domain.Name, "Identity")
	}
	if len(domain.Services) != 1 || domain.Services[0] != "test-svc" {
		t.Errorf("domain services = %v, want [test-svc]", domain.Services)
	}
}

func TestBuilder_MultipleMessages(t *testing.T) {
	t.Parallel()

	builder := catalog.NewBuilder("Test Service", "1.0.0")
	builder.AddService("test-svc", "Test Service", "1.0.0", "A test service",
		catalog.Command[TestCreateUser]("user.create"),
		catalog.Event[TestUserCreated]("user.created", catalog.Sends),
		catalog.Query[TestGetUser]("user.get"),
	)

	cat := builder.Build()
	svc := cat.Services[0]

	if len(svc.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(svc.Commands))
	}
	if len(svc.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(svc.Events))
	}
	if len(svc.Queries) != 1 {
		t.Errorf("expected 1 query, got %d", len(svc.Queries))
	}
}

func TestBuilder_MultipleServices(t *testing.T) {
	t.Parallel()

	builder := catalog.NewBuilder("Platform", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "Manages users",
		catalog.Command[TestCreateUser]("user.create"),
	)
	builder.AddService("order-svc", "Order Service", "1.0.0", "Manages orders",
		catalog.Event[TestUserCreated]("order.placed", catalog.Sends),
	)

	cat := builder.Build()
	if len(cat.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(cat.Services))
	}
}
