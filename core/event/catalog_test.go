package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestNewCatalogCore_Success(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	meta := event.CatalogMeta{
		Name:          "User Created",
		Version:       "1.0.0",
		Summary:       "A new user was created",
		AggregateType: "User",
	}

	core, err := event.NewCatalogCore(
		"UserCreated",
		aggID,
		"User",
		1,
		[]byte(`{"email":"test@example.com"}`),
		meta,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if core.Type() != "UserCreated" {
		t.Errorf("type = %q, want UserCreated", core.Type())
	}

	if core.AggregateID() != aggID {
		t.Errorf("aggregate ID = %v, want %v", core.AggregateID(), aggID)
	}

	info := core.CatalogInfo()
	if info.Name != "User Created" {
		t.Errorf("catalog name = %q, want %q", info.Name, "User Created")
	}

	if info.Version != "1.0.0" {
		t.Errorf("catalog version = %q, want 1.0.0", info.Version)
	}

	if info.Summary != "A new user was created" {
		t.Errorf("catalog summary = %q", info.Summary)
	}

	if info.AggregateType != "User" {
		t.Errorf("catalog aggregate type = %q, want User", info.AggregateType)
	}
}

func TestNewCatalogCore_InvalidEvent(t *testing.T) {
	t.Parallel()

	meta := event.CatalogMeta{Name: "Test", Version: "1.0.0"}

	_, err := event.NewCatalogCore(
		"",
		id.NewAggregateID(),
		"User",
		1,
		nil,
		meta,
	)
	if err == nil {
		t.Fatal("expected error for empty event type")
	}
}

func TestNewCatalogCore_WithOptions(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	meta := event.CatalogMeta{Name: "Test", Version: "1.0.0"}
	correlationID := id.NewCorrelationID()

	core, err := event.NewCatalogCore(
		"TestEvent",
		aggID,
		"TestAgg",
		1,
		nil,
		meta,
		event.WithCorrelationID(correlationID),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if core.Metadata().CorrelationID != correlationID {
		t.Error("correlation ID not set via option")
	}
}
