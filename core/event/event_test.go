package event_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestNewEvent_Valid(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		[]byte(`{"name":"John"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.ID().IsZero() {
		t.Error("event ID should not be empty")
	}

	if evt.Type() != "UserCreated" {
		t.Errorf("expected type UserCreated, got %s", evt.Type())
	}

	if evt.AggregateID() != id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95") {
		t.Errorf("expected aggregate ID user-123, got %s", evt.AggregateID())
	}

	if evt.AggregateType() != "User" {
		t.Errorf("expected aggregate type User, got %s", evt.AggregateType())
	}

	if evt.Version() != 1 {
		t.Errorf("expected version 1, got %d", evt.Version())
	}
}

func TestNewEvent_InvalidInputErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		eventType     event.Type
		aggregateID   id.AggregateID
		aggregateType event.AggregateType
		version       event.Version
		wantErr       error
	}{
		{
			name:          "empty event type",
			eventType:     "",
			aggregateID:   id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
			aggregateType: "User",
			version:       1,
			wantErr:       event.ErrEmptyEventType,
		},
		{
			name:          "missing aggregate ID",
			eventType:     "UserCreated",
			aggregateID:   id.AggregateID{},
			aggregateType: "User",
			version:       1,
			wantErr:       event.ErrNilAggregateID,
		},
		{
			name:          "missing aggregate type",
			eventType:     "UserCreated",
			aggregateID:   id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
			aggregateType: "",
			version:       1,
			wantErr:       event.ErrEmptyAggregateType,
		},
		{
			name:          "zero version",
			eventType:     "UserCreated",
			aggregateID:   id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
			aggregateType: "User",
			version:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := event.NewEvent(
				tt.eventType,
				tt.aggregateID,
				tt.aggregateType,
				tt.version,
				nil,
			)
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("errors.Is(err, %v) = false, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewEvent_ErrorMessagesContainContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		eventType     event.Type
		aggregateID   id.AggregateID
		aggregateType event.AggregateType
		version       event.Version
		wantContains  []string
	}{
		{
			name:          "missing aggregate ID includes event type",
			eventType:     "TestEvent",
			aggregateID:   id.AggregateID{},
			aggregateType: "User",
			version:       1,
			wantContains:  []string{"aggregate ID is required", "TestEvent"},
		},
		{
			name:          "missing aggregate type includes aggregate ID and event type",
			eventType:     "OrderCreated",
			aggregateID:   id.MustParseAggregateID("01HK154BMRQFY6Q98RCCEJDZ74"),
			aggregateType: "",
			version:       1,
			wantContains: []string{
				"aggregate type is required",
				"01HK154BMRQFY6Q98RCCEJDZ74",
				"OrderCreated",
			},
		},
		{
			name:          "zero version includes aggregate ID and event type",
			eventType:     "PaymentProcessed",
			aggregateID:   id.MustParseAggregateID("01HK154CM00YYJAJGC0GE589E2"),
			aggregateType: "Payment",
			version:       0,
			wantContains: []string{
				"version must be positive",
				"01HK154CM00YYJAJGC0GE589E2",
				"PaymentProcessed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := event.NewEvent(
				tt.eventType,
				tt.aggregateID,
				tt.aggregateType,
				tt.version,
				nil,
			)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message %q does not contain %q", errMsg, want)
				}
			}
		})
	}
}

func TestEventOptions(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"TestEvent",
		id.MustParseAggregateID("01HK154DK8FZYV2ANMQ6B0N1JY"),
		"TestAggregate",
		1,
		nil,
		event.WithCorrelationID(id.MustParseCorrelationID("01HK154EJG2GP2SR75DK1Q1TBH")),
		event.WithCausationID(id.MustParseCausationID("01HK154FHRS5276AC3V7GRNTYM")),
		event.WithUserID(id.MustParseUserID("01HK1543TRR6BB4AF65NQX5V8S")),
		event.WithRequestID(id.MustParseRequestID("01HK154GH03H0ZJCWQ2PEYSCZW")),
		event.WithSource(event.Source("test-service")),
		event.WithIPAddress(event.IPAddress("127.0.0.1")),
		event.WithUserAgent(event.UserAgent("test-agent")),
		event.WithCustom("key1", "value1"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := evt.Metadata()
	if m.CorrelationID != id.MustParseCorrelationID("01HK154EJG2GP2SR75DK1Q1TBH") {
		t.Errorf("expected correlation ID corr-123, got %s", m.CorrelationID)
	}

	if m.CausationID != id.MustParseCausationID("01HK154FHRS5276AC3V7GRNTYM") {
		t.Errorf("expected causation ID cause-456, got %s", m.CausationID)
	}

	if m.UserID != id.MustParseUserID("01HK1543TRR6BB4AF65NQX5V8S") {
		t.Errorf("expected user ID user-789, got %s", m.UserID)
	}

	if m.RequestID != id.MustParseRequestID("01HK154GH03H0ZJCWQ2PEYSCZW") {
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
	if m == nil {
		t.Fatal("NewMetadata() should return non-nil")
	}

	if m.Custom == nil {
		t.Error("Custom map should be initialized")
	}

	if !m.CorrelationID.IsZero() {
		t.Errorf("CorrelationID should be zero, got %s", m.CorrelationID)
	}
}

func TestEventAccessors(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		[]byte(`{"name":"John"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Payload() == nil || string(evt.Payload()) != `{"name":"John"}` {
		t.Errorf("expected payload, got %v", evt.Payload())
	}

	if evt.OccurredAt().IsZero() {
		t.Error("OccurredAt should not be zero")
	}

	if evt.Metadata() == nil {
		t.Error("Metadata should not be nil")
	}
}

func TestEvent_PayloadNil(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Payload() != nil {
		t.Errorf("expected nil payload, got %v", evt.Payload())
	}
}

func TestEvent_PayloadDefensiveCopy(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		[]byte("original"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := evt.Payload()
	payload[0] = 'X'

	if string(evt.Payload()) != "original" {
		t.Error("Payload() should return a defensive copy")
	}
}

func TestEvent_MetadataDefensiveCopy(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
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
	custom.CorrelationID = id.MustParseCorrelationID("01HK154EJG2GP2SR75DK1Q1TBH")

	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
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

	correlationID := id.MustParseCorrelationID("01HK154EJG2GP2SR75DK1Q1TBH")
	userID := id.NewUserID()

	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
		event.WithCorrelationID(correlationID),
		event.WithMetadata(&event.Metadata{UserID: userID}),
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

	meta := &event.Metadata{Source: "test"}
	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
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

func TestWithCustom_NilCustomMap(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
		event.WithMetadata(&event.Metadata{}),
		event.WithCustom("key1", "value1"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Metadata().Custom["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %s", evt.Metadata().Custom["key1"])
	}
}

func TestWithCustom_ExistingCustomMap(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		nil,
		event.WithCustom("key1", "value1"),
		event.WithCustom("key2", "value2"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Metadata().Custom["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %s", evt.Metadata().Custom["key1"])
	}

	if evt.Metadata().Custom["key2"] != "value2" {
		t.Errorf("expected key2=value2, got %s", evt.Metadata().Custom["key2"])
	}
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

func TestCore_MetadataNil(t *testing.T) {
	t.Parallel()

	core := &event.Core{}

	if core.Metadata() != nil {
		t.Error("expected nil metadata for zero-value Core")
	}
}

func TestEnsureMetadata_WhenNil(t *testing.T) {
	t.Parallel()

	core := &event.Core{}

	opt := event.WithCorrelationID(id.MustParseCorrelationID("01HK154EJG2GP2SR75DK1Q1TBH"))
	opt(core)

	if core.Metadata() == nil {
		t.Fatal("expected metadata to be initialized by ensureMetadata")
	}

	if core.Metadata().CorrelationID != id.MustParseCorrelationID("01HK154EJG2GP2SR75DK1Q1TBH") {
		t.Errorf("expected correlation ID to be set, got %s", core.Metadata().CorrelationID)
	}
}

func TestWithEventID(t *testing.T) {
	t.Parallel()

	overrideID := id.MustParseEventID("01HK154EJG2GP2SR75DK1Q1TBH")

	evt, err := event.NewEvent(
		"TestEvent",
		id.NewAggregateID(),
		"TestAgg",
		1,
		nil,
		event.WithEventID(overrideID),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.ID() != overrideID {
		t.Errorf("ID = %s, want %s", evt.ID(), overrideID)
	}
}

func TestWithOccurredAt(t *testing.T) {
	t.Parallel()

	ts := time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC)

	evt, err := event.NewEvent(
		"TestEvent",
		id.NewAggregateID(),
		"TestAgg",
		1,
		nil,
		event.WithOccurredAt(ts),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !evt.OccurredAt().Equal(ts) {
		t.Errorf("OccurredAt = %v, want %v", evt.OccurredAt(), ts)
	}
}

func TestParseType(t *testing.T) {
	t.Parallel()

	got, err := event.ParseType("user.created")
	if err != nil {
		t.Fatalf("ParseType: %v", err)
	}

	if got != "user.created" {
		t.Errorf("ParseType = %q, want %q", got, "user.created")
	}

	if got.IsZero() {
		t.Error("IsZero should be false for valid type")
	}
}

func TestParseType_Empty(t *testing.T) {
	t.Parallel()

	_, err := event.ParseType("")
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestMustParseType_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
	}()

	event.MustParseType("")
}

func TestParseAggregateType(t *testing.T) {
	t.Parallel()

	got, err := event.ParseAggregateType("User")
	if err != nil {
		t.Fatalf("ParseAggregateType: %v", err)
	}

	if got != "User" {
		t.Errorf("ParseAggregateType = %q, want %q", got, "User")
	}

	if got.IsZero() {
		t.Error("IsZero should be false for valid type")
	}
}

func TestParseAggregateType_Empty(t *testing.T) {
	t.Parallel()

	_, err := event.ParseAggregateType("")
	if err == nil {
		t.Fatal("expected error for empty aggregate type")
	}
}

func TestMustParseAggregateType_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
	}()

	event.MustParseAggregateType("")
}
