package event_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2/idtest"
)

func TestNew_StructPayload(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")
	payload := struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}{
		Name:  "Alice",
		Email: "alice@example.com",
	}

	evt, err := event.New("user.created", aggID, "User", event.Version(1), payload)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if evt.Type() != "user.created" {
		t.Errorf("type = %q, want user.created", evt.Type())
	}

	var got struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.Unmarshal(evt.Payload(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != "Alice" || got.Email != "alice@example.com" {
		t.Errorf("got %+v", got)
	}
}

func TestNew_MapPayload(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")
	payload := map[string]any{"key": "value"}

	evt, err := event.New("test.event", aggID, "Test", event.Version(1), payload)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(evt.Payload(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["key"] != "value" {
		t.Errorf("got %v", got)
	}
}

func TestNew_ByteSlicePayload(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")
	data := []byte(`{"raw":true}`)

	evt, err := event.New("test.raw", aggID, "Test", event.Version(1), data)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if string(evt.Payload()) != `{"raw":true}` {
		t.Errorf("payload = %q", evt.Payload())
	}
}

func TestNew_NilPayload(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	_, err := event.New("test.nil", aggID, "Test", event.Version(1), nil)
	if err == nil {
		t.Fatal("expected error for nil payload")
	}

	if !errors.Is(err, event.ErrNilPayload) {
		t.Errorf("error = %v, want ErrNilPayload", err)
	}
}

func TestNew_OptionsPreserved(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")
	corrID := idtest.ParseCorrelationID(t, "01HK1549P84T9XF8R94E960633")
	payload := map[string]any{"x": 1}

	evt, err := event.New(
		"test.opts", aggID, "Test", event.Version(1), payload,
		event.WithCorrelationID(corrID),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if evt.Metadata().CorrelationID != corrID {
		t.Errorf("correlation ID not preserved")
	}
}
