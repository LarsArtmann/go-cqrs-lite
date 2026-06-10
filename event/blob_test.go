package event_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestAttachBinary(t *testing.T) {
	t.Parallel()

	evt := mustTestEvent(t)
	key := event.MetadataKey("test.blob")
	data := []byte("hello world")

	attached, err := event.AttachBinary(evt, key, data)
	if err != nil {
		t.Fatalf("AttachBinary: %v", err)
	}

	extracted, err := event.ExtractBinary(attached, key)
	if err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}

	if string(extracted) != string(data) {
		t.Fatalf("expected %q, got %q", data, extracted)
	}
}

func TestAttachBinary_NilEvent(t *testing.T) {
	t.Parallel()

	_, err := event.AttachBinary(nil, "test.blob", []byte("data"))
	if err == nil {
		t.Fatal("expected error for nil event")
	}

	if event.Classify(err) != event.Rejection {
		t.Fatalf("expected Rejection, got %v", event.Classify(err))
	}
}

func TestExtractBinary_NilEvent(t *testing.T) {
	t.Parallel()

	_, err := event.ExtractBinary(nil, "test.blob")
	if err == nil {
		t.Fatal("expected error for nil event")
	}

	if event.Classify(err) != event.Rejection {
		t.Fatalf("expected Rejection, got %v", event.Classify(err))
	}
}

func TestExtractBinary_NotFound(t *testing.T) {
	t.Parallel()

	evt := mustTestEvent(t)
	_, err := event.ExtractBinary(evt, "test.missing")
	if err == nil {
		t.Fatal("expected error for missing key")
	}

	if !errors.Is(err, event.ErrBinaryNotFound) {
		t.Fatalf("expected ErrBinaryNotFound, got %v", err)
	}
}

func TestHasBinary(t *testing.T) {
	t.Parallel()

	evt := mustTestEvent(t)
	key := event.MetadataKey("test.blob")

	if event.HasBinary(evt, key) {
		t.Fatal("expected HasBinary to be false before attaching")
	}

	attached, err := event.AttachBinary(evt, key, []byte("data"))
	if err != nil {
		t.Fatalf("AttachBinary: %v", err)
	}

	if !event.HasBinary(attached, key) {
		t.Fatal("expected HasBinary to be true after attaching")
	}
}

func TestHasBinary_CorruptBase64(t *testing.T) {
	t.Parallel()

	evt := mustTestEvent(t)
	key := event.MetadataKey("test.blob")

	attached, err := event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		evt.Payload(),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithCustom(key, "!!!not-base64!!!"),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if !event.HasBinary(attached, key) {
		t.Fatal("corrupt base64 should still report HasBinary=true (infrastructure error)")
	}
}

func TestRejectingPublishMiddleware(t *testing.T) {
	t.Parallel()

	mw := event.RejectingPublishMiddleware("test.code", "test message")
	publisher := mw(nil)

	err := publisher.Publish(nil)
	if err == nil {
		t.Fatal("expected error from rejecting middleware")
	}

	if event.Classify(err) != event.Rejection {
		t.Fatalf("expected Rejection, got %v", event.Classify(err))
	}
}

func TestRejectingHandlerMiddleware(t *testing.T) {
	t.Parallel()

	mw := event.RejectingHandlerMiddleware("test.code", "test message")
	handler := mw(nil)

	err := handler(nil, nil)
	if err == nil {
		t.Fatal("expected error from rejecting middleware")
	}

	if event.Classify(err) != event.Rejection {
		t.Fatalf("expected Rejection, got %v", event.Classify(err))
	}
}

func mustTestEvent(t *testing.T) event.Event {
	t.Helper()

		aggregateID := id.NewAggregateID()

	evt, err := event.NewEvent(
		"test.created",
		aggregateID,
		"Test",
		1,
		[]byte(`{"name":"test"}`),
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}
