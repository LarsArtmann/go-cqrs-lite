package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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
