package event_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

func TestEventOptions(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"TestEvent",
		idtest.ParseAggregateID(t, "01HK154DK8FZYV2ANMQ6B0N1JY"),
		"TestAggregate",
		1,
		nil,
		event.WithCorrelationID(idtest.ParseCorrelationID(t, "01HK154EJG2GP2SR75DK1Q1TBH")),
		event.WithCausationID(idtest.ParseCausationID(t, "01HK154FHRS5276AC3V7GRNTYM")),
		event.WithUserID(idtest.ParseUserID(t, "01HK1543TRR6BB4AF65NQX5V8S")),
		event.WithRequestID(idtest.ParseRequestID(t, "01HK154GH03H0ZJCWQ2PEYSCZW")),
		event.WithSource(event.Source("test-service")),
		event.WithIPAddress(event.IPAddress("127.0.0.1")),
		event.WithUserAgent(event.UserAgent("test-agent")),
		event.WithCustom("key1", "value1"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := evt.Metadata()
	if m.CorrelationID != idtest.ParseCorrelationID(t, "01HK154EJG2GP2SR75DK1Q1TBH") {
		t.Errorf("expected correlation ID corr-123, got %s", m.CorrelationID)
	}

	if m.CausationID != idtest.ParseCausationID(t, "01HK154FHRS5276AC3V7GRNTYM") {
		t.Errorf("expected causation ID cause-456, got %s", m.CausationID)
	}

	if m.UserID != idtest.ParseUserID(t, "01HK1543TRR6BB4AF65NQX5V8S") {
		t.Errorf("expected user ID user-789, got %s", m.UserID)
	}

	if m.RequestID != idtest.ParseRequestID(t, "01HK154GH03H0ZJCWQ2PEYSCZW") {
		t.Errorf("expected request ID req-001, got %s", m.RequestID)
	}

	if m.Source.String() != "test-service" {
		t.Errorf("expected source test-service, got %s", m.Source)
	}

	if m.IPAddress.String() != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %s", m.IPAddress)
	}

	if m.UserAgent.String() != "test-agent" {
		t.Errorf("expected user agent test-agent, got %s", m.UserAgent)
	}

	if m.Custom["key1"] != "value1" {
		t.Errorf("expected custom key1=value1, got %s", m.Custom["key1"])
	}
}

func TestNewMetadata(t *testing.T) {
	t.Parallel()

	m := event.NewMetadata()

	if m.Custom != nil {
		t.Error("Custom map should be nil (lazy init) for zero-allocation construction")
	}

	if len(m.Custom) != 0 {
		t.Errorf("Custom map should be empty, got %v", m.Custom)
	}

	event.EnsureCustom(&m)
	if m.Custom == nil {
		t.Error("EnsureCustom should initialize the Custom map")
	}

	if !m.CorrelationID.IsZero() {
		t.Errorf("CorrelationID should be zero, got %s", m.CorrelationID)
	}
}

func TestEvent_MetadataDefensiveCopy(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
		event.WithCustom("key1", "value1"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m1 := evt.Metadata()
	m1.Custom["key1"] = "modified"

	m2 := evt.Metadata()
	if m2.Custom["key1"] != "value1" {
		t.Error("Metadata() should return a defensive copy")
	}
}

func TestWithMetadata(t *testing.T) {
	t.Parallel()

	custom := event.NewMetadata()
	custom.CorrelationID = idtest.ParseCorrelationID(t, "01HK154EJG2GP2SR75DK1Q1TBH")

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
		event.WithMetadata(custom),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Metadata().CorrelationID != custom.CorrelationID {
		t.Errorf(
			"expected WithMetadata to set correlation ID, got %s",
			evt.Metadata().CorrelationID,
		)
	}
}

func TestWithMetadata_MergesInsteadOfReplace(t *testing.T) {
	t.Parallel()

	correlationID := idtest.ParseCorrelationID(t, "01HK154EJG2GP2SR75DK1Q1TBH")
	userID := id.NewUserID()

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
		event.WithCorrelationID(correlationID),
		event.WithMetadata(event.Metadata{Tracing: metadata.Tracing{UserID: userID}}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := evt.Metadata()
	if meta.CorrelationID != correlationID {
		t.Errorf(
			"correlation ID should be preserved after WithMetadata, got %s",
			meta.CorrelationID,
		)
	}

	if meta.UserID != userID {
		t.Errorf("user ID should be merged from WithMetadata, got %s", meta.UserID)
	}
}

func TestWithMetadata_NilExisting(t *testing.T) {
	t.Parallel()

	meta := event.Metadata{Source: "test"}
	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
		event.WithMetadata(meta),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Metadata().Source != "test" {
		t.Errorf("expected source=test, got %s", evt.Metadata().Source)
	}
}

func TestMetadataMerge_DoesNotMutateBase(t *testing.T) {
	t.Parallel()

	base := event.Metadata{
		Custom: map[event.MetadataKey]string{"tenant": "acme"},
	}
	overlay := event.Metadata{
		Custom: map[event.MetadataKey]string{"region": "us-east-1"},
	}

	merged := base.Merge(overlay)

	if merged.Custom["tenant"] != "acme" {
		t.Errorf("base Custom lost: tenant = %q", merged.Custom["tenant"])
	}

	if merged.Custom["region"] != "us-east-1" {
		t.Errorf("overlay Custom not copied: region = %q", merged.Custom["region"])
	}

	if _, ok := base.Custom["region"]; ok {
		t.Error("Merge mutated the base Custom map — must return a new map")
	}
}

func TestMetadataKeyConstants(t *testing.T) {
	t.Parallel()

	if event.MetadataKeyClientID != "client.id" {
		t.Errorf("MetadataKeyClientID = %q, want %q", event.MetadataKeyClientID, "client.id")
	}

	if event.MetadataKeyClientOccurredAt != "client.occurred_at" {
		t.Errorf(
			"MetadataKeyClientOccurredAt = %q, want %q",
			event.MetadataKeyClientOccurredAt,
			"client.occurred_at",
		)
	}
}

func assertCustomKV(t *testing.T, m event.Metadata, key, want string) {
	t.Helper()
	if m.Custom[event.MetadataKey(key)] != want {
		t.Errorf("expected %s=%s, got %s", key, want, m.Custom[event.MetadataKey(key)])
	}
}

func TestWithCustom_NilCustomMap(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
		event.WithMetadata(event.Metadata{}),
		event.WithCustom("key1", "value1"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCustomKV(t, evt.Metadata(), "key1", "value1")
}

func TestWithCustom_ExistingCustomMap(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
		event.WithCustom("key1", "value1"),
		event.WithCustom("key2", "value2"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCustomKV(t, evt.Metadata(), "key1", "value1")
	assertCustomKV(t, evt.Metadata(), "key2", "value2")
}

func TestWithClientID(t *testing.T) {
	t.Parallel()

	clientID := id.NewClientID()

	evt, err := event.NewEvent(
		"TestEvent",
		id.NewAggregateID(),
		"Test",
		1,
		nil,
		event.WithClientID(clientID),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := evt.Metadata().Custom["client.id"]
	if got != clientID.String() {
		t.Errorf("client.id = %q, want %q", got, clientID.String())
	}
}

func TestWithClientOccurredAt(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC)

	evt, err := event.NewEvent(
		"TestEvent",
		id.NewAggregateID(),
		"Test",
		1,
		nil,
		event.WithClientOccurredAt(ts),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := evt.Metadata().Custom["client.occurred_at"]
	if got != ts.Format(time.RFC3339Nano) {
		t.Errorf("client.occurred_at = %q, want %q", got, ts.Format(time.RFC3339Nano))
	}
}

func TestCore_MetadataDefaultValue(t *testing.T) {
	t.Parallel()

	core := &event.ImmutableEvent{}

	md := core.Metadata()
	if md.Custom != nil {
		t.Error("expected nil Custom map for zero-value ImmutableEvent")
	}
}

func TestEnsureMetadata_WhenNil(t *testing.T) {
	t.Parallel()

	core := &event.ImmutableEvent{}

	opt := event.WithCorrelationID(idtest.ParseCorrelationID(t, "01HK154EJG2GP2SR75DK1Q1TBH"))
	opt(core)

	if core.Metadata().CorrelationID != idtest.ParseCorrelationID(
		t,
		"01HK154EJG2GP2SR75DK1Q1TBH",
	) {
		t.Errorf("expected correlation ID to be set, got %s", core.Metadata().CorrelationID)
	}
}
