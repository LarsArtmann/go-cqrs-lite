package event_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
)

func TestNewEvent_Valid(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
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

	if evt.StreamID() != idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95") {
		t.Errorf("expected aggregate ID user-123, got %s", evt.StreamID())
	}

	if evt.StreamType() != "User" {
		t.Errorf("expected aggregate type User, got %s", evt.StreamType())
	}

	if evt.Version() != 1 {
		t.Errorf("expected version 1, got %d", evt.Version())
	}
}

func TestNewEvent_InvalidInputErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		eventType  event.Type
		streamID   id.StreamID
		streamType id.StreamType
		version    event.Version
		wantErr    error
	}{
		{
			name:       "empty event type",
			eventType:  "",
			streamID:   idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
			streamType: "User",
			version:    1,
			wantErr:    event.ErrEmptyEventType,
		},
		{
			name:       "missing aggregate ID",
			eventType:  "UserCreated",
			streamID:   id.StreamID{},
			streamType: "User",
			version:    1,
			wantErr:    event.ErrNilStreamID,
		},
		{
			name:       "missing aggregate type",
			eventType:  "UserCreated",
			streamID:   idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
			streamType: "",
			version:    1,
			wantErr:    event.ErrEmptyStreamType,
		},
		{
			name:       "zero version",
			eventType:  "UserCreated",
			streamID:   idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
			streamType: "User",
			version:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := event.NewEvent(
				tt.eventType,
				tt.streamID,
				tt.streamType,
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
		name         string
		eventType    event.Type
		streamID     id.StreamID
		streamType   id.StreamType
		version      event.Version
		wantContains []string
	}{
		{
			name:         "missing aggregate ID includes event type",
			eventType:    "TestEvent",
			streamID:     id.StreamID{},
			streamType:   "User",
			version:      1,
			wantContains: []string{"aggregate ID is required", "TestEvent"},
		},
		{
			name:       "missing aggregate type includes aggregate ID and event type",
			eventType:  "OrderCreated",
			streamID:   idtest.ParseAggregateID(t, "01HK154BMRQFY6Q98RCCEJDZ74"),
			streamType: "",
			version:    1,
			wantContains: []string{
				"aggregate type is required",
				"01HK154BMRQFY6Q98RCCEJDZ74",
				"OrderCreated",
			},
		},
		{
			name:       "zero version includes aggregate ID and event type",
			eventType:  "PaymentProcessed",
			streamID:   idtest.ParseAggregateID(t, "01HK154CM00YYJAJGC0GE589E2"),
			streamType: "Payment",
			version:    0,
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
				tt.streamID,
				tt.streamType,
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

func TestEventAccessors(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
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

	if evt.Metadata().Custom != nil {
		t.Error("Metadata Custom map should be nil (lazy init) for events without custom metadata")
	}
}

func TestEvent_PayloadNil(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
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

func TestEvent_PayloadConstructionCopy(t *testing.T) {
	t.Parallel()

	original := []byte("original")
	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		original,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	original[0] = 'X'

	if string(evt.Payload()) != "original" {
		t.Error("Payload should be independent of caller's original slice")
	}
}

func TestEvent_PayloadReturnIsImmutable(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		[]byte("immutable"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := evt.Payload()
	payload[0] = 'X'

	if string(evt.Payload()) != "immutable" {
		t.Error("mutating Payload() return value must not affect the event's internal state")
	}
}
