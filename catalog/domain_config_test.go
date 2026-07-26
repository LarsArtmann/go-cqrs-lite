package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func TestBuilder_ConfigureDomain_Sends(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddDomain("orders", "Orders", "1.0.0", "Order management", "order-svc")
	b.ConfigureDomain(
		"orders",
		catalog.DomainSends(catalog.Ref{ID: "order.created"}),
	)

	cat := b.Build()
	d := cat.Domains[0]

	if len(d.Sends) != 1 || d.Sends[0].ID != "order.created" {
		t.Errorf("expected sends [order.created], got %v", d.Sends)
	}
}

func TestBuilder_ConfigureDomain_Receives(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddDomain("orders", "Orders", "1.0.0", "Order management")
	b.ConfigureDomain(
		"orders",
		catalog.DomainReceives(catalog.Ref{ID: "payment.completed"}),
	)

	cat := b.Build()
	d := cat.Domains[0]

	if len(d.Receives) != 1 || d.Receives[0].ID != "payment.completed" {
		t.Errorf("expected receives [payment.completed], got %v", d.Receives)
	}
}

func TestBuilder_ConfigureDomain_Entities(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddDomain("orders", "Orders", "1.0.0", "Order management")
	b.ConfigureDomain(
		"orders",
		catalog.DomainEntities("Order", "OrderItem"),
	)

	cat := b.Build()
	d := cat.Domains[0]

	if len(d.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(d.Entities))
	}

	if d.Entities[0] != "Order" || d.Entities[1] != "OrderItem" {
		t.Errorf("expected [Order, OrderItem], got %v", d.Entities)
	}
}

func TestBuilder_ConfigureDomain_Badges(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddDomain("orders", "Orders", "1.0.0", "Order management")
	b.ConfigureDomain(
		"orders",
		catalog.DomainBadges(catalog.Badge{Content: "Core Domain", BackgroundColor: "blue"}),
	)

	cat := b.Build()
	d := cat.Domains[0]

	if len(d.Badges) != 1 || d.Badges[0].Content != "Core Domain" {
		t.Errorf("expected 1 badge with Core Domain, got %v", d.Badges)
	}
}

func TestBuilder_ConfigureDomain_Owners(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddDomain("orders", "Orders", "1.0.0", "Order management")
	b.ConfigureDomain(
		"orders",
		catalog.DomainOwners("team-commerce", "alice"),
	)

	cat := b.Build()
	d := cat.Domains[0]

	if len(d.Owners) != 2 {
		t.Fatalf("expected 2 owners, got %d", len(d.Owners))
	}
}

func TestBuilder_ConfigureDomain_Attachments(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddDomain("orders", "Orders", "1.0.0", "Order management")
	b.ConfigureDomain(
		"orders",
		catalog.DomainAttachments(
			catalog.Attachment{URL: "https://adr.example.com/001", Title: "ADR-001"},
		),
	)

	cat := b.Build()
	d := cat.Domains[0]

	if len(d.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(d.Attachments))
	}

	if d.Attachments[0].Title != "ADR-001" {
		t.Errorf("expected ADR-001, got %s", d.Attachments[0].Title)
	}
}

func TestBuilder_ConfigureDomain_MultipleOptions(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.AddDomain("orders", "Orders", "1.0.0", "Order management")
	b.ConfigureDomain(
		"orders",
		catalog.DomainSends(catalog.Ref{ID: "order.created"}),
		catalog.DomainReceives(catalog.Ref{ID: "payment.completed"}),
		catalog.DomainEntities("Order"),
		catalog.DomainBadges(catalog.Badge{Content: "Core"}),
		catalog.DomainOwners("team-a"),
		catalog.DomainAttachments(catalog.Attachment{URL: "https://example.com", Title: "Doc"}),
	)

	cat := b.Build()
	d := cat.Domains[0]

	if len(d.Sends) != 1 {
		t.Errorf("expected 1 send, got %d", len(d.Sends))
	}

	if len(d.Receives) != 1 {
		t.Errorf("expected 1 receive, got %d", len(d.Receives))
	}

	if len(d.Entities) != 1 {
		t.Errorf("expected 1 entity, got %d", len(d.Entities))
	}

	if len(d.Badges) != 1 {
		t.Errorf("expected 1 badge, got %d", len(d.Badges))
	}

	if len(d.Owners) != 1 {
		t.Errorf("expected 1 owner, got %d", len(d.Owners))
	}

	if len(d.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(d.Attachments))
	}
}

func TestBuilder_ConfigureDomain_NonexistentIgnored(t *testing.T) {
	t.Parallel()

	b := cattest.NewTestBuilder(t)
	b.ConfigureDomain(
		"nonexistent",
		catalog.DomainBadges(catalog.Badge{Content: "Should not panic"}),
	)

	cat := b.Build()
	if len(cat.Domains) != 0 {
		t.Errorf("expected 0 domains, got %d", len(cat.Domains))
	}
}
