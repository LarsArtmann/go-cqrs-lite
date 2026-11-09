package httptyped_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/httptyped"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type userResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func TestNewRequestSchema(t *testing.T) {
	t.Parallel()

	reqSchema := httptyped.NewRequestSchema[createUserRequest]()

	sch := reqSchema.Schema()
	if sch == nil {
		t.Fatal("expected non-nil schema")
	}

	if sch.Type != schema.TypeObject {
		t.Errorf("schema type = %q, want %q", sch.Type, schema.TypeObject)
	}

	if len(sch.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(sch.Properties))
	}

	if _, ok := sch.Properties["name"]; !ok {
		t.Error("missing property 'name'")
	}

	if _, ok := sch.Properties["email"]; !ok {
		t.Error("missing property 'email'")
	}
}

func TestNewResponseSchema(t *testing.T) {
	t.Parallel()

	respSchema := httptyped.NewResponseSchema[userResponse]()

	sch := respSchema.Schema()
	if sch == nil {
		t.Fatal("expected non-nil schema")
	}

	if sch.Type != schema.TypeObject {
		t.Errorf("schema type = %q, want %q", sch.Type, schema.TypeObject)
	}

	if len(sch.Properties) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(sch.Properties))
	}
}

func TestResponseSchema_ToResponseSpec(t *testing.T) {
	t.Parallel()

	respSchema := httptyped.NewResponseSchema[userResponse]()

	spec := respSchema.ToResponseSpec("200", "Success")

	if spec.StatusCode != "200" {
		t.Errorf("status code = %q, want %q", spec.StatusCode, "200")
	}

	if spec.Description != "Success" {
		t.Errorf("description = %q, want %q", spec.Description, "Success")
	}

	if spec.Schema == nil {
		t.Fatal("expected non-nil schema in response spec")
	}

	if spec.Schema.Type != schema.TypeObject {
		t.Errorf("schema type = %q, want %q", spec.Schema.Type, schema.TypeObject)
	}
}

func TestCommand_CreatesMessageWithSchema(t *testing.T) {
	t.Parallel()

	builder := cattest.NewTestBuilder(t)
	builder.AddService(
		"svc", "Service", "1.0.0", "Summary",
		httptyped.Command[createUserRequest]("user.create"),
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

	if cmd.Kind != catalog.CommandMessage {
		t.Errorf("kind = %q, want %q", cmd.Kind, catalog.CommandMessage)
	}

	if cmd.Schema == nil {
		t.Fatal("expected non-nil schema")
	}
}

func TestQuery_CreatesMessageWithSchema(t *testing.T) {
	t.Parallel()

	builder := cattest.NewTestBuilder(t)
	builder.AddService(
		"svc", "Service", "1.0.0", "Summary",
		httptyped.Query[createUserRequest]("user.list"),
	)

	cat := builder.Build()
	svc := cat.Services[0]

	if len(svc.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(svc.Queries))
	}

	qry := svc.Queries[0]
	if qry.ID != "user.list" {
		t.Errorf("query ID = %q, want %q", qry.ID, "user.list")
	}

	if qry.Kind != catalog.QueryMessage {
		t.Errorf("kind = %q, want %q", qry.Kind, catalog.QueryMessage)
	}
}

func TestOKResponse_Adds200Response(t *testing.T) {
	t.Parallel()

	builder := cattest.NewTestBuilder(t)
	builder.AddService(
		"svc", "Service", "1.0.0", "Summary",
		httptyped.Command[createUserRequest](
			"user.create",
			httptyped.OKResponse[userResponse]("OK"),
		),
	)

	cat := builder.Build()
	cmd := cat.Services[0].Commands[0]

	if len(cmd.Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(cmd.Responses))
	}

	resp := cmd.Responses[0]
	if resp.StatusCode != "200" {
		t.Errorf("status = %q, want 200", resp.StatusCode)
	}

	if resp.Schema == nil {
		t.Fatal("expected non-nil response schema")
	}
}

func TestCreatedResponse_Adds201Response(t *testing.T) {
	t.Parallel()

	builder := cattest.NewTestBuilder(t)
	builder.AddService(
		"svc", "Service", "1.0.0", "Summary",
		httptyped.Command[createUserRequest](
			"user.create",
			httptyped.CreatedResponse[userResponse]("Created"),
		),
	)

	cat := builder.Build()
	cmd := cat.Services[0].Commands[0]

	if len(cmd.Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(cmd.Responses))
	}

	if cmd.Responses[0].StatusCode != "201" {
		t.Errorf("status = %q, want 201", cmd.Responses[0].StatusCode)
	}
}

func TestErrorResponse_AddsTypedResponse(t *testing.T) {
	t.Parallel()

	type errBody struct {
		Message string `json:"message"`
	}

	builder := cattest.NewTestBuilder(t)
	builder.AddService(
		"svc", "Service", "1.0.0", "Summary",
		httptyped.Command[createUserRequest](
			"user.create",
			httptyped.ErrorResponse[errBody]("400", "Bad Request"),
		),
	)

	cat := builder.Build()
	cmd := cat.Services[0].Commands[0]

	if len(cmd.Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(cmd.Responses))
	}

	if cmd.Responses[0].StatusCode != "400" {
		t.Errorf("status = %q, want 400", cmd.Responses[0].StatusCode)
	}
}
