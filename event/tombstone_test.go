package event_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestTombstoneStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status event.TombstoneStatus
		want   string
	}{
		{event.TombstoneActive, "active"},
		{event.TombstoneTombstoned, "tombstoned"},
		{event.TombstoneUndetermined, "undetermined"},
		{event.TombstoneStatus(99), "TombstoneStatus(99)"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("TombstoneStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestTombstoneStatus_Predicates(t *testing.T) {
	t.Parallel()

	if !event.TombstoneActive.IsActive() {
		t.Error("TombstoneActive.IsActive() = false, want true")
	}

	if event.TombstoneActive.IsTombstoned() {
		t.Error("TombstoneActive.IsTombstoned() = true, want false")
	}

	if !event.TombstoneTombstoned.IsTombstoned() {
		t.Error("expected TombstoneTombstoned.IsTombstoned() = true")
	}

	if !event.TombstoneActive.IsKnown() {
		t.Error("TombstoneActive.IsKnown() = false, want true")
	}

	if event.TombstoneUndetermined.IsKnown() {
		t.Error("TombstoneUndetermined.IsKnown() = true, want false")
	}
}

func TestDetectTombstone(t *testing.T) {
	t.Parallel()

	evt := func(metaKey event.MetadataKey, metaVal string) event.Event {
		opts := []event.Option{}
		if metaKey != "" {
			opts = append(opts, event.WithCustom(metaKey, metaVal))
		}

		e, err := event.NewEvent(
			"test.event",
			id.NewAggregateID(),
			"Test",
			event.Version(1),
			[]byte(`{}`),
			opts...,
		)
		if err != nil {
			t.Fatal(err)
		}

		return e
	}

	tests := []struct {
		name   string
		events []event.Event
		want   event.TombstoneStatus
	}{
		{"empty stream", nil, event.TombstoneUndetermined},
		{"no metadata", []event.Event{evt("", "")}, event.TombstoneUndetermined},
		{
			"tombstoned",
			[]event.Event{evt(event.MetadataKeyTombstone, "true")},
			event.TombstoneTombstoned,
		},
		{"rebirth", []event.Event{evt(event.MetadataKeyRebirth, "true")}, event.TombstoneActive},
		{"tombstone then rebirth", []event.Event{
			evt(event.MetadataKeyTombstone, "true"),
			evt(event.MetadataKeyRebirth, "true"),
		}, event.TombstoneActive},
		{"rebirth then tombstone", []event.Event{
			evt(event.MetadataKeyRebirth, "true"),
			evt(event.MetadataKeyTombstone, "true"),
		}, event.TombstoneTombstoned},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := event.DetectTombstone(tt.events)
			if got != tt.want {
				t.Errorf("DetectTombstone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarkTombstone(t *testing.T) {
	t.Parallel()

	t.Run("nil event returns error", func(t *testing.T) {
		t.Parallel()

		_, err := event.MarkTombstone(nil)
		if err == nil {
			t.Error("expected error for nil event")
		}
	})

	t.Run("sets tombstone metadata", func(t *testing.T) {
		t.Parallel()

		orig, err := event.NewEvent(
			"user.deleted", id.NewAggregateID(), "User",
			event.Version(3), []byte(`{"reason":"gdpr"}`),
		)
		if err != nil {
			t.Fatal(err)
		}

		marked, err := event.MarkTombstone(orig)
		if err != nil {
			t.Fatal(err)
		}

		md := marked.Metadata()
		if md.Custom[event.MetadataKeyTombstone] != "true" {
			t.Error("tombstone metadata not set")
		}

		// Original unchanged
		origMd := orig.Metadata()
		if origMd.Custom[event.MetadataKeyTombstone] != "" {
			t.Error("original event was modified")
		}

		// Identity preserved
		if marked.ID() != orig.ID() {
			t.Error("event ID changed")
		}

		if marked.Type() != orig.Type() {
			t.Error("event type changed")
		}
	})
}

func TestMarkRebirth(t *testing.T) {
	t.Parallel()

	t.Run("nil event returns error", func(t *testing.T) {
		t.Parallel()

		_, err := event.MarkRebirth(nil)
		if err == nil {
			t.Error("expected error for nil event")
		}
	})

	t.Run("sets rebirth metadata", func(t *testing.T) {
		t.Parallel()

		orig, err := event.NewEvent(
			"user.reactivated", id.NewAggregateID(), "User",
			event.Version(4), []byte(`{}`),
		)
		if err != nil {
			t.Fatal(err)
		}

		marked, err := event.MarkRebirth(orig)
		if err != nil {
			t.Fatal(err)
		}

		md := marked.Metadata()
		if md.Custom[event.MetadataKeyRebirth] != "true" {
			t.Error("rebirth metadata not set")
		}
	})
}

func TestMarkTombstone_AllFieldsPreserved(t *testing.T) {
	t.Parallel()

	deadline := time.Date(2026, 6, 9, 22, 0, 0, 0, time.UTC)
	schemaV, err := event.ParseSchemaVersion(3)
	if err != nil {
		t.Fatal(err)
	}

	orig, err := event.NewEvent(
		"user.deleted", id.NewAggregateID(), "User",
		event.Version(5), []byte(`{"reason":"gdpr"}`),
		event.WithEncoding(codec.Encoding("custom")),
		event.WithSchemaVersion(schemaV),
		event.WithCorrelationID(id.NewCorrelationID()),
		event.WithCausationID(id.NewCausationID()),
		event.WithUserID(id.NewUserID()),
		event.WithDeadline(deadline),
		event.WithCustom("traceId", "abc-123"),
	)
	if err != nil {
		t.Fatal(err)
	}

	marked, err := event.MarkTombstone(orig)
	if err != nil {
		t.Fatal(err)
	}

	if marked.ID() != orig.ID() {
		t.Error("ID not preserved")
	}

	if marked.Type() != orig.Type() {
		t.Error("Type not preserved")
	}

	if marked.AggregateID() != orig.AggregateID() {
		t.Error("AggregateID not preserved")
	}

	if marked.AggregateType() != orig.AggregateType() {
		t.Error("id.AggregateType not preserved")
	}

	if marked.Version() != orig.Version() {
		t.Error("Version not preserved")
	}

	if marked.SchemaVersion() != orig.SchemaVersion() {
		t.Error("SchemaVersion not preserved")
	}

	if marked.Encoding() != orig.Encoding() {
		t.Errorf("Encoding not preserved: got %q, want %q", marked.Encoding(), orig.Encoding())
	}

	if string(marked.Payload()) != string(orig.Payload()) {
		t.Error("Payload not preserved")
	}

	if marked.OccurredAt() != orig.OccurredAt() {
		t.Error("OccurredAt not preserved")
	}

	markDeadline, markHas := marked.Deadline()
	origDeadline, origHas := orig.Deadline()
	if markHas != origHas || markDeadline != origDeadline {
		t.Error("Deadline not preserved")
	}

	md := marked.Metadata()
	if md.Custom[event.MetadataKeyTombstone] != "true" {
		t.Error("tombstone metadata not set")
	}

	if md.CorrelationID != orig.Metadata().CorrelationID {
		t.Error("CorrelationID not preserved")
	}

	if md.CausationID != orig.Metadata().CausationID {
		t.Error("CausationID not preserved")
	}

	if md.UserID != orig.Metadata().UserID {
		t.Error("UserID not preserved")
	}

	if md.Custom["traceId"] != "abc-123" {
		t.Error("existing custom metadata not preserved")
	}
}

func TestMarkRebirth_AllFieldsPreserved(t *testing.T) {
	t.Parallel()

	orig, err := event.NewEvent(
		"user.reactivated", id.NewAggregateID(), "User",
		event.Version(6), []byte(`{"source":"admin"}`),
		event.WithCorrelationID(id.NewCorrelationID()),
		event.WithUserID(id.NewUserID()),
	)
	if err != nil {
		t.Fatal(err)
	}

	marked, err := event.MarkRebirth(orig)
	if err != nil {
		t.Fatal(err)
	}

	if marked.ID() != orig.ID() {
		t.Error("ID not preserved")
	}

	if marked.Type() != orig.Type() {
		t.Error("Type not preserved")
	}

	if marked.AggregateID() != orig.AggregateID() {
		t.Error("AggregateID not preserved")
	}

	if marked.Version() != orig.Version() {
		t.Error("Version not preserved")
	}

	md := marked.Metadata()
	if md.Custom[event.MetadataKeyRebirth] != "true" {
		t.Error("rebirth metadata not set")
	}

	if md.CorrelationID != orig.Metadata().CorrelationID {
		t.Error("CorrelationID not preserved")
	}

	if md.UserID != orig.Metadata().UserID {
		t.Error("UserID not preserved")
	}

	origMd := orig.Metadata()
	if origMd.Custom[event.MetadataKeyRebirth] != "" {
		t.Error("original event was modified")
	}
}
