package huma_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/huma"
)

func TestToMessages_GETBecomesQuery(t *testing.T) {
	t.Parallel()

	ops := []huma.HumaOperation{
		{Method: "GET", Path: "/api/users", OperationID: "listUsers", Summary: "List users"},
	}

	msgs := huma.ToMessages(ops)

	builder := catalog.NewBuilder("Test", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "Summary", msgs...)

	cat := builder.Build()
	svc := cat.Services[0]

	if len(svc.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d (commands=%d)", len(svc.Queries), len(svc.Commands))
	}

	qry := svc.Queries[0]
	if qry.ID != "listUsers" {
		t.Errorf("query ID = %q, want %q", qry.ID, "listUsers")
	}

	if qry.Kind != catalog.QueryMessage {
		t.Errorf("kind = %q, want %q", qry.Kind, catalog.QueryMessage)
	}

	if qry.Operation == nil {
		t.Fatal("expected non-nil operation")
	}

	if qry.Operation.Method != "GET" {
		t.Errorf("operation method = %q, want GET", qry.Operation.Method)
	}

	if qry.Operation.Path != "/api/users" {
		t.Errorf("operation path = %q, want /api/users", qry.Operation.Path)
	}
}

func TestToMessages_POSTBecomesCommand(t *testing.T) {
	t.Parallel()

	ops := []huma.HumaOperation{
		{Method: "POST", Path: "/api/users", OperationID: "createUser"},
	}

	msgs := huma.ToMessages(ops)

	builder := catalog.NewBuilder("Test", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "Summary", msgs...)

	cat := builder.Build()
	svc := cat.Services[0]

	if len(svc.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d (queries=%d)", len(svc.Commands), len(svc.Queries))
	}

	cmd := svc.Commands[0]
	if cmd.ID != "createUser" {
		t.Errorf("command ID = %q, want %q", cmd.ID, "createUser")
	}

	if cmd.Kind != catalog.CommandMessage {
		t.Errorf("kind = %q, want %q", cmd.Kind, catalog.CommandMessage)
	}
}

func TestToMessages_PUT_DELETE_PATCH_BecomeCommands(t *testing.T) {
	t.Parallel()

	ops := []huma.HumaOperation{
		{Method: "PUT", Path: "/api/users/{id}", OperationID: "replaceUser"},
		{Method: "DELETE", Path: "/api/users/{id}", OperationID: "deleteUser"},
		{Method: "PATCH", Path: "/api/users/{id}", OperationID: "patchUser"},
	}

	msgs := huma.ToMessages(ops)

	if len(msgs) != 3 {
		t.Fatalf("expected 3 message configs, got %d", len(msgs))
	}

	builder := catalog.NewBuilder("Test", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "Summary", msgs...)

	cat := builder.Build()
	svc := cat.Services[0]

	if len(svc.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(svc.Commands))
	}

	if len(svc.Queries) != 0 {
		t.Fatalf("expected 0 queries, got %d", len(svc.Queries))
	}
}

func TestToMessages_SummaryPropagated(t *testing.T) {
	t.Parallel()

	ops := []huma.HumaOperation{
		{Method: "GET", Path: "/api/health", OperationID: "healthCheck", Summary: "Health check"},
	}

	msgs := huma.ToMessages(ops)

	builder := catalog.NewBuilder("Test", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "Summary", msgs...)

	cat := builder.Build()
	qry := cat.Services[0].Queries[0]

	if string(qry.Summary) != "Health check" {
		t.Errorf("summary = %q, want %q", qry.Summary, "Health check")
	}
}

func TestToMessages_EmptyInput(t *testing.T) {
	t.Parallel()

	msgs := huma.ToMessages(nil)

	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}
}

func TestToMessages_MixedOps(t *testing.T) {
	t.Parallel()

	ops := []huma.HumaOperation{
		{Method: "GET", Path: "/api/items", OperationID: "listItems"},
		{Method: "POST", Path: "/api/items", OperationID: "createItem"},
		{Method: "GET", Path: "/api/items/{id}", OperationID: "getItem"},
		{Method: "DELETE", Path: "/api/items/{id}", OperationID: "deleteItem"},
	}

	msgs := huma.ToMessages(ops)

	builder := catalog.NewBuilder("Test", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "Summary", msgs...)

	cat := builder.Build()
	svc := cat.Services[0]

	if len(svc.Queries) != 2 {
		t.Errorf("expected 2 queries (GETs), got %d", len(svc.Queries))
	}

	if len(svc.Commands) != 2 {
		t.Errorf("expected 2 commands (POST+DELETE), got %d", len(svc.Commands))
	}
}
