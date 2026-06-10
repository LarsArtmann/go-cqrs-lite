package event_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)



func parseEventID(s string) id.EventID {
	v, err := id.ParseEventID(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestWithEventID(t *testing.T) {
	t.Parallel()

	overrideID := parseEventID("01HK154EJG2GP2SR75DK1Q1TBH")

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

func TestClone_DeepCopy(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		parseAggID("01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		[]byte("original"),
		event.WithCorrelationID(parseCorrID("01HK154EJG2GP2SR75DK1Q1TBH")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cloned := evt.Clone()

	if cloned.ID() != evt.ID() {
		t.Error("cloned ID should match original")
	}

	if cloned.Type() != evt.Type() {
		t.Error("cloned Type should match original")
	}

	if cloned.AggregateID() != evt.AggregateID() {
		t.Error("cloned AggregateID should match original")
	}

	if cloned.Version() != evt.Version() {
		t.Error("cloned Version should match original")
	}

	if string(cloned.Payload()) != "original" {
		t.Error("cloned payload should match original")
	}

	if cloned.Metadata().CorrelationID != evt.Metadata().CorrelationID {
		t.Error("cloned metadata should match original")
	}
}

func TestClone_IndependentPayload(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.NewAggregateID(),
		"User",
		1,
		[]byte("original"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cloned := evt.Clone()

	clonedPayload := cloned.Payload()
	clonedPayload[0] = 'X'

	if string(evt.Payload()) != "original" {
		t.Error("mutating cloned payload should not affect original")
	}
}

func TestClone_IndependentMetadata(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.NewAggregateID(),
		"User",
		1,
		[]byte("{}"),
		event.WithCorrelationID(parseCorrID("01HK154EJG2GP2SR75DK1Q1TBH")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cloned := evt.Clone()

	clonedMeta := cloned.Metadata()
	clonedMeta.CorrelationID = id.CorrelationID{}

	if evt.Metadata().CorrelationID.String() == "" {
		t.Error("mutating cloned metadata should not affect original")
	}
}

func TestClone_NilPayload(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		id.NewAggregateID(),
		"User",
		1,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cloned := evt.Clone()

	if cloned.Payload() != nil {
		t.Error("cloned nil payload should remain nil")
	}
}
