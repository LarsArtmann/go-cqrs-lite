package event_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/v5/event"
	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

func causeEvent(t *testing.T, opts ...event.Option) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		"user.created", id.NewStreamID(), "User", event.Version(1), nil, opts...)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

// TestCause_Lifecycle_TypedAtBirth pins review R6: causation is typed at
// construction, not reconstructed from precedence prose at read time.
func TestCause_Lifecycle_TypedAtBirth(t *testing.T) {
	t.Parallel()

	cmdID := id.NewCommandID()
	ctx := event.WithCommandCausality(context.Background(), "create_user", cmdID)

	evt := causeEvent(t, event.CommandCausalityEnricher(ctx)...)

	want := record.Cause{Kind: record.CauseCommand, ID: cmdID.String()}
	if evt.Cause() != want {
		t.Errorf("Cause() = %v, want %v (typed at birth via the enricher)", evt.Cause(), want)
	}

	if got := event.AsRecord(evt).MetaData.Cause; got != want {
		t.Errorf("AsRecord Cause = %v, want the field verbatim (field wins)", got)
	}
}

func TestCause_Lifecycle_WithCauseOption(t *testing.T) {
	t.Parallel()

	timerCause := record.Cause{Kind: record.CauseTimer, ID: "order-timeout-1"}
	evt := causeEvent(t, event.WithCause(timerCause))

	if evt.Cause() != timerCause {
		t.Errorf("Cause() = %v, want %v", evt.Cause(), timerCause)
	}
}

func TestCause_Lifecycle_None(t *testing.T) {
	t.Parallel()

	evt := causeEvent(t)
	if !evt.Cause().IsZero() {
		t.Errorf("Cause() = %v, want zero (direct action carries no cause)", evt.Cause())
	}

	if got := event.AsRecord(evt).MetaData.Cause; !got.IsZero() {
		t.Errorf("AsRecord Cause = %v, want zero", got)
	}
}

// TestCause_Lifecycle_TracingFallback: with no field and no twin, a tracing
// CausationID still survives as an honest CauseUnknown.
func TestCause_Lifecycle_TracingFallback(t *testing.T) {
	t.Parallel()

	causationID := id.NewCausationID()
	evt := causeEvent(t, event.WithCausationID(causationID))

	want := record.Cause{Kind: record.CauseUnknown, ID: causationID.String()}
	if got := event.AsRecord(evt).MetaData.Cause; got != want {
		t.Errorf("AsRecord Cause = %v, want the unknown-kind tracing fallback %v", got, want)
	}
}

// TestCause_Lifecycle_ReconstructedTwinFallback: reconstructed events carry
// the metadata Causation twin from storage but no first-class field — the
// bridge consults the twin and derives the typed cause.
func TestCause_Lifecycle_ReconstructedTwinFallback(t *testing.T) {
	t.Parallel()

	cmdID := id.NewCommandID()
	streamID := id.NewStreamID()

	md := event.NewMetadata()
	md.Causation = &event.Causation{
		CommandType: "create_user",
		CommandID:   cmdID,
	}

	evt, err := event.ReconstructEventWithMetadata(
		id.NewEventID(),
		"user.created",
		"User",
		streamID,
		1, 1,
		nil,
		md,
		time.Now(),
		"json",
		"test",
	)
	if err != nil {
		t.Fatalf("ReconstructEventWithMetadata: %v", err)
	}

	if !evt.Cause().IsZero() {
		t.Errorf("Cause() = %v, want zero (the field is not persisted)", evt.Cause())
	}

	want := record.Cause{Kind: record.CauseCommand, ID: cmdID.String()}
	if got := event.AsRecord(evt).MetaData.Cause; got != want {
		t.Errorf("AsRecord Cause = %v, want the twin-derived %v", got, want)
	}
}
