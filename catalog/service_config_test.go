package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func TestBuilder_ConfigureService_Badges(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService("svc", "Service", "1.0.0", "test")
	b.ConfigureService(
		"svc",
		catalog.ServiceBadges(
			catalog.Badge{Content: "Production", BackgroundColor: "green"},
		),
	)

	cat := b.Build()
	svc := cat.Services[0]

	if len(svc.Badges) != 1 {
		t.Fatalf("expected 1 badge, got %d", len(svc.Badges))
	}

	if svc.Badges[0].Content != "Production" {
		t.Errorf("expected Production, got %s", svc.Badges[0].Content)
	}
}

func TestBuilder_ConfigureService_Repository(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService("svc", "Service", "1.0.0", "test")
	b.ConfigureService(
		"svc",
		catalog.ServiceRepository("Go", "https://github.com/example/svc"),
	)

	cat := b.Build()
	svc := cat.Services[0]

	if svc.Repository == nil {
		t.Fatal("expected repository to be set")
	}

	if svc.Repository.Language != "Go" {
		t.Errorf("expected Go, got %s", svc.Repository.Language)
	}
}

func TestBuilder_ConfigureService_WritesToReadsFrom(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService("svc", "Service", "1.0.0", "test")
	b.ConfigureService(
		"svc",
		catalog.ServiceWritesTo("orders-db"),
		catalog.ServiceReadsFrom("products-cache"),
	)

	cat := b.Build()
	svc := cat.Services[0]

	if len(svc.WritesTo) != 1 || svc.WritesTo[0] != "orders-db" {
		t.Errorf("expected writesTo [orders-db], got %v", svc.WritesTo)
	}

	if len(svc.ReadsFrom) != 1 || svc.ReadsFrom[0] != "products-cache" {
		t.Errorf("expected readsFrom [products-cache], got %v", svc.ReadsFrom)
	}
}

func TestBuilder_ConfigureService_Entities(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService("svc", "Service", "1.0.0", "test")
	b.ConfigureService(
		"svc",
		catalog.ServiceEntities("Order", "OrderItem"),
	)

	cat := b.Build()
	svc := cat.Services[0]

	if len(svc.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(svc.Entities))
	}
}

func TestBuilder_ConfigureService_Specifications(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService("svc", "Service", "1.0.0", "test")
	b.ConfigureService(
		"svc",
		catalog.ServiceSpecifications(
			catalog.Specification{Type: "asyncapi", Path: "asyncapi.yaml"},
		),
	)

	cat := b.Build()
	svc := cat.Services[0]

	if len(svc.Specifications) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(svc.Specifications))
	}

	if svc.Specifications[0].Type != "asyncapi" {
		t.Errorf("expected asyncapi, got %s", svc.Specifications[0].Type)
	}
}

func TestBuilder_ConfigureService_MultipleOptions(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddService("svc", "Service", "1.0.0", "test")
	b.ConfigureService(
		"svc",
		catalog.ServiceOwners("team-a", "alice"),
		catalog.ServiceBadges(catalog.Badge{Content: "Stable"}),
		catalog.ServiceRepository("Go", "https://github.com/example/svc"),
		catalog.ServiceWritesTo("db"),
		catalog.ServiceAttachments(
			catalog.Attachment{URL: "https://adr.example.com/001", Title: "ADR-001"},
		),
	)

	cat := b.Build()
	svc := cat.Services[0]

	if len(svc.Owners) != 2 {
		t.Errorf("expected 2 owners, got %d", len(svc.Owners))
	}

	if len(svc.Badges) != 1 {
		t.Errorf("expected 1 badge, got %d", len(svc.Badges))
	}

	if svc.Repository == nil {
		t.Error("expected repository")
	}

	if len(svc.WritesTo) != 1 {
		t.Errorf("expected 1 writesTo, got %d", len(svc.WritesTo))
	}

	if len(svc.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(svc.Attachments))
	}
}
