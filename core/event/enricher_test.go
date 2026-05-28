package event

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestEnrichEvent(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	enricher := ContextEnricher(func(_ context.Context) []Option {
		return []Option{
			WithCorrelationID(id.MustParseCorrelationID("01JBCORR0LATI0ON0ID0000001")),
			WithUserID(id.MustParseUserID("01JBUSER0ID000000000000001")),
		}
	})

	evt, err := NewEvent("UserCreated", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	enrichEvent(context.Background(), evt, enricher)

	meta := evt.Metadata()
	if meta.CorrelationID.IsZero() {
		t.Error("expected correlation ID to be set")
	}

	if meta.UserID.IsZero() {
		t.Error("expected user ID to be set")
	}
}

func TestCompositeEnricher(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	first := ContextEnricher(func(_ context.Context) []Option {
		return []Option{
			WithCorrelationID(id.MustParseCorrelationID("01JBCORR0LATI0ON0ID0000001")),
		}
	})

	second := ContextEnricher(func(_ context.Context) []Option {
		return []Option{
			WithSource("test-service"),
		}
	})

	composite := CompositeEnricher(first, second)

	evt, err := NewEvent("UserCreated", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	enrichEvent(context.Background(), evt, composite)

	meta := evt.Metadata()
	if meta.CorrelationID.IsZero() {
		t.Error("expected correlation ID from first enricher")
	}

	if meta.Source != "test-service" {
		t.Errorf("expected source from second enricher, got %q", meta.Source)
	}
}

func TestCompositeEnricher_Empty(t *testing.T) {
	t.Parallel()

	composite := CompositeEnricher()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, err := NewEvent("UserCreated", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	enrichEvent(context.Background(), evt, composite)

	meta := evt.Metadata()
	if !meta.CorrelationID.IsZero() {
		t.Error("expected no correlation ID from empty enricher")
	}
}
