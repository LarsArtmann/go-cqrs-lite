package adapters_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func TestBuilder_FromCommandDispatcher(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	d.RegisterCatalogEntry("user.create", dispatcher.CatalogEntry{
		Name:    "CreateUser",
		Version: "1.0.0",
		Summary: "Creates a new user",
	})
	d.RegisterCatalogEntry("user.change_email", dispatcher.CatalogEntry{
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
		found[string(cmd.ID)] = true
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
	d.RegisterCatalogEntry("user.get", dispatcher.CatalogEntry{
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
