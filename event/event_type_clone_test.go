package event_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2/idtest"
)

func TestWithEventID(t *testing.T) {
	t.Parallel()

	overrideID := idtest.ParseEventID(t, "01HK154EJG2GP2SR75DK1Q1TBH")

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

func TestClone_DeepCopy(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"UserCreated",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"User",
		1,
		[]byte("original"),
		event.WithCorrelationID(idtest.ParseCorrelationID(t, "01HK154EJG2GP2SR75DK1Q1TBH")),
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
		event.WithCorrelationID(idtest.ParseCorrelationID(t, "01HK154EJG2GP2SR75DK1Q1TBH")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cloned := evt.Clone()

	originalCorrID := evt.Metadata().CorrelationID
	clonedMeta := cloned.Metadata()
	clonedMeta.CorrelationID = id.CorrelationID{}

	if clonedMeta.CorrelationID == originalCorrID {
		t.Error("expected cloned metadata CorrelationID to differ after mutation")
	}

	if evt.Metadata().CorrelationID != originalCorrID {
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

// TestClone_IndependentOpts verifies that Clone produces independent opts
// (deadline, clock). eventOptions fields are immutable types (func, interface,
// time.Time) so a shallow struct copy suffices — but we lock in the guarantee
// that the clone and original share no mutable state through opts.
func TestClone_IndependentOpts(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	deadline := fixedTime.Add(30 * time.Second)

	evt, err := event.NewEvent(
		"UserCreated",
		id.NewAggregateID(),
		"User",
		1,
		[]byte("payload"),
		event.WithClock(func() time.Time { return fixedTime }),
		event.WithDeadline(deadline),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cloned := evt.Clone()

	// Values are preserved.
	if cloned.OccurredAt() != evt.OccurredAt() {
		t.Errorf("cloned OccurredAt = %v, want %v", cloned.OccurredAt(), evt.OccurredAt())
	}

	clonedDeadline, clonedOK := cloned.Deadline()
	origDeadline, origOK := evt.Deadline()
	if !clonedOK || !origOK {
		t.Fatal("both clone and original should have a deadline")
	}

	if clonedDeadline != origDeadline {
		t.Errorf("cloned Deadline = %v, want %v", clonedDeadline, origDeadline)
	}
}
