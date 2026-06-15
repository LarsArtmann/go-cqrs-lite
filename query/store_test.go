package query_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func TestNewPersistedQuery_Success(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"q":"alice"}`)

	q, err := query.NewPersistedQuery("user.search", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Type() != "user.search" {
		t.Errorf("Type() = %q, want %q", q.Type(), "user.search")
	}

	if q.ID().IsZero() {
		t.Error("ID() should not be zero")
	}

	if q.ReceivedAt().IsZero() {
		t.Error("ReceivedAt() should not be zero")
	}

	meta := q.Metadata()
	if !meta.CorrelationID.IsZero() || !meta.CausationID.IsZero() || !meta.UserID.IsZero() ||
		!meta.RequestID.IsZero() {
		t.Error("Metadata() should return zero tracing fields")
	}

	gotPayload := q.Payload()
	if string(gotPayload) != `{"q":"alice"}` {
		t.Errorf("Payload() = %q, want %q", gotPayload, `{"q":"alice"}`)
	}
}

func TestNewPersistedQuery_EmptyType(t *testing.T) {
	t.Parallel()

	_, err := query.NewPersistedQuery("", nil)
	if err == nil {
		t.Fatal("expected error for empty query type")
	}

	if !errors.Is(err, query.ErrEmptyQueryType) {
		t.Errorf("errors.Is(err, ErrEmptyQueryType) = false, got: %v", err)
	}
}

func TestNewPersistedQuery_PayloadIsolation(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"q":"alice"}`)

	q, err := query.NewPersistedQuery("user.search", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	returned := q.Payload()
	returned[0] = 'X'

	original := q.Payload()
	if original[0] == 'X' {
		t.Error("mutating Payload() return value should not affect internal state")
	}

	payload[0] = 'Y'
	again := q.Payload()
	if again[0] == 'Y' {
		t.Error("mutating original payload slice should not affect PersistedQuery")
	}
}

func TestNewPersistedQuery_NilPayload(t *testing.T) {
	t.Parallel()

	q, err := query.NewPersistedQuery("user.count", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Payload() != nil {
		t.Errorf("Payload() = %v, want nil", q.Payload())
	}
}

func TestNewPersistedQuery_MetadataIsolation(t *testing.T) {
	t.Parallel()

	meta := query.NewMetadata()
	event.EnsureCustom(&meta)
	meta.Custom["key1"] = "value1"

	q, err := query.NewPersistedQuery(
		"user.search", nil,
		query.WithQueryMetadata(meta),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	returned := q.Metadata()
	returned.Custom["key1"] = "modified"

	original := q.Metadata()
	if original.Custom["key1"] != "value1" {
		t.Error("mutating Metadata() return value should not affect internal state")
	}
}

func TestWithQueryMetadata_IntakeIsolation(t *testing.T) {
	t.Parallel()

	meta := query.NewMetadata()
	event.EnsureCustom(&meta)
	meta.Custom["key"] = "original"

	q, err := query.NewPersistedQuery(
		"user.search", nil,
		query.WithQueryMetadata(meta),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta.Custom["key"] = "mutated_after_passing"

	got := q.Metadata()
	if got.Custom["key"] != "original" {
		t.Error("WithQueryMetadata stored caller's map reference — mutation leaked into query")
	}
}

func TestPersistedQuery_String(t *testing.T) {
	t.Parallel()

	q, err := query.NewPersistedQuery("user.search", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := q.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}
