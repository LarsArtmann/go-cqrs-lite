package event

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/id/idtest"
)

func TestRequestScopeRoundTrip(t *testing.T) {
	t.Parallel()

	scope := RequestScope{
		CorrelationID: idtest.ParseCorrelationID(t, "01JBCORR0LATI0ON0ID0000001"),
		RequestID:     idtest.ParseRequestID(t, "01JBR3QVST1D00000000000000"),
		IPAddress:     "203.0.113.7",
		UserAgent:     "cqrs-test/1.0",
		ClientID:      id.NewClientID(),
	}

	ctx := WithRequestScope(context.Background(), scope)

	got, ok := RequestScopeFromContext(ctx)
	if !ok {
		t.Fatal("scope not found after WithRequestScope")
	}

	if got.CorrelationID != scope.CorrelationID ||
		got.RequestID != scope.RequestID ||
		got.IPAddress != scope.IPAddress ||
		got.UserAgent != scope.UserAgent ||
		got.ClientID != scope.ClientID {
		t.Fatalf("round trip mismatch: got %+v", got)
	}
}

func TestRequestScopeFromContext_AbsentAndZero(t *testing.T) {
	t.Parallel()

	if _, ok := RequestScopeFromContext(context.Background()); ok {
		t.Error("absent scope must report false")
	}

	ctx := WithRequestScope(context.Background(), RequestScope{})
	if _, ok := RequestScopeFromContext(ctx); ok {
		t.Error("all-zero scope must report false")
	}
}

func TestRequestScopeEnricher(t *testing.T) {
	t.Parallel()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")

	scope := RequestScope{
		CorrelationID: idtest.ParseCorrelationID(t, "01JBCORR0LATI0ON0ID0000001"),
		RequestID:     idtest.ParseRequestID(t, "01JBR3QVST1D00000000000000"),
		IPAddress:     "203.0.113.7",
		UserAgent:     "cqrs-test/1.0",
		ClientID:      id.NewClientID(),
	}

	ctx := WithRequestScope(context.Background(), scope)

	evt, err := NewEvent("PageViewed", streamID, "Page", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	enrichEvent(ctx, evt, RequestScopeEnricher)

	meta := evt.Metadata()
	if meta.CorrelationID != scope.CorrelationID {
		t.Errorf("correlation ID = %v, want %v", meta.CorrelationID, scope.CorrelationID)
	}

	if meta.RequestID != scope.RequestID {
		t.Errorf("request ID = %v, want %v", meta.RequestID, scope.RequestID)
	}

	if meta.IPAddress != scope.IPAddress {
		t.Errorf("IP address = %q, want %q", meta.IPAddress, scope.IPAddress)
	}

	if meta.UserAgent != scope.UserAgent {
		t.Errorf("user agent = %q, want %q", meta.UserAgent, scope.UserAgent)
	}

	if got := meta.Custom[MetadataKeyClientID]; got != scope.ClientID.String() {
		t.Errorf("client ID not propagated through metadata, got %q", got)
	}
}

func TestRequestScopeEnricher_PartialScopeEnrichesOnlyKnownFields(t *testing.T) {
	t.Parallel()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")

	ctx := WithRequestScope(context.Background(), RequestScope{
		IPAddress: "198.51.100.4",
	})

	evt, err := NewEvent("Pinged", streamID, "Node", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	enrichEvent(ctx, evt, RequestScopeEnricher)

	meta := evt.Metadata()
	if meta.IPAddress != "198.51.100.4" {
		t.Errorf("IP address = %q, want the partial value", meta.IPAddress)
	}

	if !meta.CorrelationID.IsZero() || !meta.RequestID.IsZero() || meta.UserAgent != "" {
		t.Error("absent scope fields must not be stamped")
	}
}

func TestRequestScopeEnricher_NoScopeReturnsNil(t *testing.T) {
	t.Parallel()

	if opts := RequestScopeEnricher(context.Background()); opts != nil {
		t.Fatalf("no-scope enricher must return nil options, got %d", len(opts))
	}
}

func TestCompositeEnricher_ActorAndRequestScope(t *testing.T) {
	t.Parallel()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	actor := id.NewActorID(id.ActorUser, "user-42")

	ctx := WithActorContext(
		WithRequestScope(context.Background(), RequestScope{
			CorrelationID: idtest.ParseCorrelationID(t, "01JBCORR0LATI0ON0ID0000001"),
		}),
		actor,
	)

	evt, err := NewEvent("OrderPlaced", streamID, "Order", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	enrichEvent(ctx, evt, CompositeEnricher(ActorEnricher, RequestScopeEnricher))

	meta := evt.Metadata()
	if meta.ActorID != actor {
		t.Errorf("actor = %v, want %v", meta.ActorID, actor)
	}

	if meta.CorrelationID.IsZero() {
		t.Error("correlation ID missing from composite enrichment")
	}
}
