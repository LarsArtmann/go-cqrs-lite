package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestBuilder_MessageOption_Producers(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Event[struct{}](
			"evt", catalog.Sends,
			catalog.Producers("svc-a", "svc-b"),
		),
	)

	cat := b.Build()
	evt := cat.Services[0].Events[0]

	if len(evt.Producers) != 2 {
		t.Fatalf("expected 2 producers, got %d", len(evt.Producers))
	}

	if evt.Producers[0] != "svc-a" || evt.Producers[1] != "svc-b" {
		t.Errorf("expected [svc-a, svc-b], got %v", evt.Producers)
	}
}

func TestBuilder_MessageOption_Consumers(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Command[struct{}](
			"cmd",
			catalog.Consumers("handler-svc"),
		),
	)

	cat := b.Build()
	cmd := cat.Services[0].Commands[0]

	if len(cmd.Consumers) != 1 {
		t.Fatalf("expected 1 consumer, got %d", len(cmd.Consumers))
	}

	if cmd.Consumers[0] != "handler-svc" {
		t.Errorf("expected handler-svc, got %s", cmd.Consumers[0])
	}
}

func TestBuilder_MessageOption_Operation(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Command[struct{}](
			"cmd",
			catalog.MsgOperation("POST", "/orders", "201", "400"),
		),
	)

	cat := b.Build()
	cmd := cat.Services[0].Commands[0]

	if cmd.Operation == nil {
		t.Fatal("expected operation to be set")
	}

	if cmd.Operation.Method != "POST" {
		t.Errorf("expected POST, got %s", cmd.Operation.Method)
	}

	if cmd.Operation.Path != "/orders" {
		t.Errorf("expected /orders, got %s", cmd.Operation.Path)
	}

	if len(cmd.Operation.StatusCodes) != 2 {
		t.Errorf("expected 2 status codes, got %d", len(cmd.Operation.StatusCodes))
	}
}

func TestBuilder_MessageOption_Badges(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Event[struct{}](
			"evt", catalog.Sends,
			catalog.MsgBadges(
				catalog.Badge{Content: "Domain Event", BackgroundColor: "orange"},
			),
		),
	)

	cat := b.Build()
	evt := cat.Services[0].Events[0]

	if len(evt.Badges) != 1 {
		t.Fatalf("expected 1 badge, got %d", len(evt.Badges))
	}

	if evt.Badges[0].Content != "Domain Event" {
		t.Errorf("expected Domain Event, got %s", evt.Badges[0].Content)
	}
}

func TestBuilder_MessageOption_Repository(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Command[struct{}](
			"cmd",
			catalog.MsgRepository("Go", "https://github.com/example/orders"),
		),
	)

	cat := b.Build()
	cmd := cat.Services[0].Commands[0]

	if cmd.Repository == nil {
		t.Fatal("expected repository to be set")
	}

	if cmd.Repository.Language != "Go" {
		t.Errorf("expected Go, got %s", cmd.Repository.Language)
	}
}

func TestHttpStatusDescription_KnownCodes(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"200": "OK",
		"201": "Created",
		"202": "Accepted",
		"204": "No Content",
	}

	for code, want := range cases {
		if got := catalog.HttpStatusDescription(code); got != want {
			t.Errorf("HttpStatusDescription(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestHttpStatusDescription_UnknownCodeFallback(t *testing.T) {
	t.Parallel()

	if got := catalog.HttpStatusDescription("210"); got != "Success" {
		t.Errorf("HttpStatusDescription(\"210\") = %q, want \"Success\"", got)
	}
}

// TestWithOperation_DefaultDescription verifies that WithOperation derives a
// spec-compliant non-empty description from the success code, so the generated
// OpenAPI document never violates the 3.0 spec (Response.description is REQUIRED
// and must be non-empty).
func TestWithOperation_DefaultDescription(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddService(
		"svc", "Service", "1.0.0", "test",
		catalog.Command[struct{}](
			"cmd",
			catalog.WithOperation[struct{}]("POST", "/api/items", "201"),
		),
	)

	cat := b.Build()
	cmd := cat.Services[0].Commands[0]

	if len(cmd.Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(cmd.Responses))
	}

	if cmd.Responses[0].Description == "" {
		t.Fatal("expected non-empty description, got empty string (violates OpenAPI 3.0 spec)")
	}

	if cmd.Responses[0].Description != "Created" {
		t.Errorf("expected \"Created\", got %q", cmd.Responses[0].Description)
	}
}
