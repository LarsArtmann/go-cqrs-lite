package middleware

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

// MetadataKeyOTelCorrelationID is the custom metadata key under which
// OTelCorrelationEnricher stores the distributed correlation ID from OTel
// baggage. Use OTelCorrelationIDFromEvent to read it back.
const MetadataKeyOTelCorrelationID event.MetadataKey = "otel.correlation_id"

// OTelCorrelationEnricher bridges OTel baggage correlation IDs into event
// metadata. It reads the correlation ID from OTel baggage (set via
// cqrsotel.WithCorrelationID or propagated through W3C baggage headers) and
// stores it as a custom metadata field on every event produced by the decider.
//
// This enricher complements event.WithCorrelationID (which stores a branded
// ULID for domain-level command→event traceability). The OTel correlation ID
// is stored as a string in custom metadata, preserving distributed trace
// context across services. The two are independent — use both:
//
//	decider.WithEnricher(event.CompositeEnricher(
//	    event.CommandCausalityEnricher,
//	    middleware.OTelCorrelationEnricher,
//	))
//
// Returns nil options when no correlation ID is present in the context,
// making it safe to compose with other enrichers via CompositeEnricher.
func OTelCorrelationEnricher(ctx context.Context) []event.Option {
	raw := cqrsotel.CorrelationIDFromContext(ctx)
	if raw == "" {
		return nil
	}

	return []event.Option{
		event.WithCustom(MetadataKeyOTelCorrelationID, raw),
	}
}

// OTelCorrelationIDFromEvent extracts the OTel distributed correlation ID
// from an event's custom metadata. Returns empty string if not set.
func OTelCorrelationIDFromEvent(evt event.Event) string {
	return evt.Metadata().Custom[MetadataKeyOTelCorrelationID]
}
