package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func TestBuilder_MessageOption_Channels(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddChannel(catalog.Channel{ID: "crm.journal", Name: "CRM Journal"})
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Event[struct{}](
			"contact.created", catalog.Sends,
			catalog.Channels("crm.journal"),
		),
	)

	cat := b.Build()
	evt := cat.Services[0].Events[0]

	if len(evt.Channels) != 1 || evt.Channels[0] != "crm.journal" {
		t.Fatalf("expected Channels [crm.journal] on message, got %v", evt.Channels)
	}
}

func TestBuilder_MessageOption_WithTypedParam(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Query[struct{}](
			"list-things",
			catalog.WithTypedParam("limit", "query", "Page size", false,
				&catalog.Property{Type: catalog.TypeInteger}),
		),
	)

	cat := b.Build()
	q := cat.Services[0].Queries[0]

	if len(q.Schema.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(q.Schema.Parameters))
	}

	param := q.Schema.Parameters[0]
	if param.Schema == nil || param.Schema.Type != catalog.TypeInteger {
		t.Errorf("expected integer parameter schema, got %+v", param.Schema)
	}
}

func TestBuilder_MessageOption_WithResponseHeader(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Command[struct{}](
			"send",
			catalog.WithResponse("429", "Rate limited"),
			catalog.WithResponseHeader("429", "Retry-After", "Seconds until retry"),
		),
	)

	cat := b.Build()
	cmd := cat.Services[0].Commands[0]

	if len(cmd.Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(cmd.Responses))
	}

	headers := cmd.Responses[0].Headers
	if len(headers) != 1 || headers[0].Name != "Retry-After" {
		t.Fatalf("expected Retry-After header on 429, got %v", headers)
	}

	if headers[0].Description != "Seconds until retry" {
		t.Errorf("header description = %q", headers[0].Description)
	}
}

func TestBuilder_MessageOption_WithResponseHeader_CreatesResponse(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Command[struct{}](
			"send",
			catalog.WithResponseHeader("204", "Deprecation", "Sunset header"),
		),
	)

	cat := b.Build()
	cmd := cat.Services[0].Commands[0]

	if len(cmd.Responses) != 1 || cmd.Responses[0].StatusCode != "204" {
		t.Fatalf("expected bare 204 response to be created, got %+v", cmd.Responses)
	}
}

func TestBuilder_MessageOption_WithRequestBodyContent(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Command[struct{}](
			"upload",
			catalog.WithRequestBodyContent("multipart/form-data", &catalog.Schema{
				Type: catalog.TypeObject,
			}),
		),
	)

	cat := b.Build()
	cmd := cat.Services[0].Commands[0]

	if cmd.RequestBodyMediaType != "multipart/form-data" {
		t.Errorf("RequestBodyMediaType = %q", cmd.RequestBodyMediaType)
	}

	if cmd.RequestBodySchema == nil {
		t.Error("RequestBodySchema must carry the declared parts schema")
	}
}
