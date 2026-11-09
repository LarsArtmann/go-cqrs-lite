package catalog

import (
	"strings"
	"testing"
)

func coeffectFixture() *Catalog {
	return &Catalog{
		Title:   "Coeffect",
		Version: "1.0.0",
		Services: []Service{
			{
				ID:      "user-svc",
				Name:    "User Service",
				Version: "1.0.0",
				Events: []Message{
					{
						ID:        "user.created",
						Kind:      EventMessage,
						Name:      "User Created",
						Direction: Sends,
					},
				},
			},
			{
				ID:      "audit-svc",
				Name:    "Audit Service",
				Version: "1.0.0",
				Events: []Message{
					{
						ID:        "user.created",
						Kind:      EventMessage,
						Name:      "User Created",
						Direction: Receives,
					},
					{
						ID:        "user.creted",
						Kind:      EventMessage,
						Name:      "User Creted",
						Direction: Receives,
					},
				},
			},
		},
	}
}

func TestValidateCoeffects_FlagsDanglingSubscription(t *testing.T) {
	t.Parallel()

	violations := coeffectFixture().ValidateCoeffects()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}

	if !strings.Contains(violations[0].Message, "user.creted") {
		t.Errorf("message should name the dangling type: %q", violations[0].Message)
	}

	if !strings.Contains(violations[0].Message, "audit-svc") {
		t.Errorf("message should name the consumer: %q", violations[0].Message)
	}
}

func TestValidateCoeffects_NoViolationForProducedEvents(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Coeffect",
		Version: "1.0.0",
		Services: []Service{
			{
				ID:      "user-svc",
				Name:    "User Service",
				Version: "1.0.0",
				Events: []Message{
					{
						ID:        "user.created",
						Kind:      EventMessage,
						Name:      "User Created",
						Direction: Sends,
					},
				},
			},
			{
				ID:      "audit-svc",
				Name:    "Audit Service",
				Version: "1.0.0",
				Events: []Message{
					{
						ID:        "user.created",
						Kind:      EventMessage,
						Name:      "User Created",
						Direction: Receives,
					},
				},
			},
		},
	}

	if violations := cat.ValidateCoeffects(); len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %v", len(violations), violations)
	}
}

// Explicit Producers are honored — the escape hatch for events imported
// from systems outside this catalog.
func TestValidateCoeffects_ExplicitExternalProducerPasses(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Coeffect",
		Version: "1.0.0",
		Services: []Service{
			{
				ID:      "billing-svc",
				Name:    "Billing Service",
				Version: "1.0.0",
				Events: []Message{
					{
						ID:        "payment.settled",
						Kind:      EventMessage,
						Name:      "Payment Settled",
						Direction: Receives,
						Producers: []ServiceID{"stripe"},
					},
				},
			},
		},
	}

	if violations := cat.ValidateCoeffects(); len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %v", len(violations), violations)
	}
}

// Producer-without-consumer is NOT a violation — that is the advisory tier,
// rendered in the export summary instead.
func TestValidateCoeffects_UnconsumedEventIsNotAViolation(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Coeffect",
		Version: "1.0.0",
		Services: []Service{
			{
				ID:      "user-svc",
				Name:    "User Service",
				Version: "1.0.0",
				Events: []Message{
					{
						ID:        "user.archived",
						Kind:      EventMessage,
						Name:      "User Archived",
						Direction: Sends,
					},
				},
			},
		},
	}

	if violations := cat.ValidateCoeffects(); len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %v", len(violations), violations)
	}
}

func TestDeriveProducersConsumers_ExplicitDeclarationsWin(t *testing.T) {
	t.Parallel()

	enriched := coeffectFixture().DeriveProducersConsumers()

	for _, svc := range enriched.Services {
		for _, evt := range svc.Events {
			if evt.ID != "user.created" {
				continue
			}

			if len(evt.Producers) != 1 || evt.Producers[0] != "user-svc" {
				t.Errorf("user.created producers = %v, want [user-svc]", evt.Producers)
			}

			if len(evt.Consumers) != 1 || evt.Consumers[0] != "audit-svc" {
				t.Errorf("user.created consumers = %v, want [audit-svc]", evt.Consumers)
			}
		}
	}
}
